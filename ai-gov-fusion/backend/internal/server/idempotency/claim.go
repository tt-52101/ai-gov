package idempotency

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"
)

// ErrIdempotencyConflict 当写请求携带一个已存在但具有不同请求指纹（RequestHash）
// 的幂等键到达时，由 Claim 返回。这是硬拒绝：客户端不得对不同的负载使用相同的键重试。
//
// HTTP 映射：409 Conflict，错误码：IDEMPOTENCY_CONFLICT（PRD §6.1）。
var ErrIdempotencyConflict = errors.New("幂等冲突：相同键但不同请求负载")

// ErrIdempotencyReplay 当写请求携带与先前已完成操作相同的幂等键和请求指纹到达时，
// 由 Claim 返回。调用方应返回现有结果（通过 *Record 参数访问），
// 而非重新执行操作。
//
// HTTP 映射：200 OK，错误码：IDEMPOTENCY_REPLAY（PRD §6.1）。
var ErrIdempotencyReplay = errors.New("幂等重放：重复请求，使用原始结果")

// ErrIdempotencyInProgress 当键已被 Claim 但操作尚未完成或失败时返回。
// 调用方应在短暂退避后重试。
var ErrIdempotencyInProgress = errors.New("幂等键进行中：操作尚未完成")

// errReclaimRace 是在 reclaim 中使用的内部哨兵，表示另一个 goroutine 赢得了回收竞争
//（旧记录已被并发回收删除）。
var errReclaimRace = errors.New("回收竞争：记录已被并发调用方回收")

// DefaultRetentionWindow 是幂等记录的最短生命周期。
// 超过此时间后，记录可由 CleanExpired 清理。
// 设置为 24 小时，per PRD §8.7。
const DefaultRetentionWindow = 24 * time.Hour

// Claim 原子地为资金写操作预约一个幂等键。
//
// 策略（INSERT 优先，铁律 1）：
//  1. 将记录 INSERT 到 idempotency_records。
//  2. 若 INSERT 成功：键已被 Claim。返回 (true, nil, nil)。
//  3. 若 UNIQUE 约束冲突（gorm.ErrDuplicatedKey）：
//     a. 获取现有记录。
//     b. 若现有记录已过期：在事务中尝试原子 DELETE+INSERT 回收。
//     若回收成功，返回 (true, nil, nil)。
//     c. 若现有记录具有相同的 RequestHash：
//     - 若状态为 StatusSucceeded：返回 (false, existing, ErrIdempotencyReplay)。
//     调用方应返回存储的响应。
//     - 若状态为 StatusStarted：返回 (false, nil, ErrIdempotencyInProgress)。
//     - 若状态为 StatusFailed：返回 (false, existing, ErrIdempotencyConflict)。
//     d. 若现有记录具有不同的 RequestHash：
//     返回 (false, nil, ErrIdempotencyConflict)。
//
// 绝不使用 SELECT 后 INSERT 模式：该路径在检查存在性与插入之间存在 TOCTOU 竞态。
// INSERT 优先方法让数据库 UNIQUE 约束保证原子性（PRD §8.7，Stripe 语义）。
//
// 调用方负责在调用 Claim 前设置 rec 上的 ID、Scope、ActorID、IdempotencyKey、
// RequestHash、Status（应为 StatusStarted）和 ExpiresAt。
// ExpiresAt 应为至少 DefaultRetentionWindow 之后的未来时间。
//
// 成功 Claim（claimed=true）后，调用方必须最终调用 Complete 或 Fail 以从
// StatusStarted 状态释放键。
func Claim(ctx context.Context, db *gorm.DB, rec *Record) (claimed bool, existing *Record, err error) {
	// 步骤 1：INSERT——数据库 UNIQUE 约束是原子性机制。
	if err := InsertRecord(db, rec); err == nil {
		// Insert 成功：键已被 Claim，操作可继续。
		slog.InfoContext(ctx, "幂等键已申请",
			"idempotency_key", rec.IdempotencyKey,
			"scope", rec.Scope,
			"actor_id", rec.ActorID,
			"expires_at", rec.ExpiresAt,
		)
		return true, nil, nil
	} else if !isDuplicateKeyError(err) {
		// 非重复键错误（如连接失败）是致命的。
		return false, nil, fmt.Errorf("幂等插入: %w", err)
	}

	// 步骤 2：UNIQUE 约束冲突——记录已存在。
	// 获取现有记录以判断重放 vs 冲突。
	existing, err = GetRecord(db, rec.Scope, rec.ActorID, rec.IdempotencyKey)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 不应发生：UNIQUE 约束保证存在一条记录，
			// 但若它在我们的 INSERT 和 SELECT 之间被删除，重试一次。
			slog.WarnContext(ctx, "幂等键已消失",
				"idempotency_key", rec.IdempotencyKey,
				"scope", rec.Scope,
			)
			return Claim(ctx, db, rec)
		}
		return false, nil, fmt.Errorf("幂等查找: %w", err)
	}

	// 步骤 3：检查现有记录是否已过期。
	// 已过期记录可被原子回收。
	if time.Now().After(existing.ExpiresAt) {
		claimed, err := reclaim(ctx, db, existing, rec)
		if err != nil {
			return false, nil, err
		}
		if claimed {
			return true, nil, nil
		}
		// 回收失败（竞态）：重新获取并进入哈希比较。
		existing, err = GetRecord(db, rec.Scope, rec.ActorID, rec.IdempotencyKey)
		if err != nil {
			return false, nil, fmt.Errorf("回收竞态后重新查找幂等记录: %w", err)
		}
	}

	// 步骤 4：相同指纹 → 重放。
	if existing.RequestHash == rec.RequestHash {
		return handleReplay(ctx, existing)
	}

	// 步骤 5：不同指纹 → 冲突。
	slog.WarnContext(ctx, "幂等键冲突",
		"idempotency_key", rec.IdempotencyKey,
		"scope", rec.Scope,
		"actor_id", rec.ActorID,
		"existing_hash", existing.RequestHash,
		"new_hash", rec.RequestHash,
	)
	return false, nil, ErrIdempotencyConflict
}

// handleReplay 确定当现有幂等键出现相同请求指纹时的结果。
func handleReplay(ctx context.Context, existing *Record) (bool, *Record, error) {
	switch existing.Status {
	case StatusSucceeded:
		slog.InfoContext(ctx, "幂等重放成功",
			"idempotency_key", existing.IdempotencyKey,
			"scope", existing.Scope,
			"resource_ref", ptrVal(existing.ResourceRef),
		)
		return false, existing, ErrIdempotencyReplay

	case StatusFailed:
		slog.WarnContext(ctx, "幂等重放失败",
			"idempotency_key", existing.IdempotencyKey,
			"scope", existing.Scope,
		)
		return false, existing, ErrIdempotencyConflict

	case StatusStarted:
		slog.WarnContext(ctx, "幂等键进行中",
			"idempotency_key", existing.IdempotencyKey,
			"scope", existing.Scope,
		)
		return false, nil, ErrIdempotencyInProgress

	default:
		return false, nil, fmt.Errorf("幂等: 未知状态 %q", existing.Status)
	}
}

// reclaim 在单个数据库事务中原子地删除一条过期记录并插入一条新记录。
// 这防止了两个并发 goroutine 都看到过期记录并尝试回收时的竞态条件：
// 只有一条 INSERT 会成功。
func reclaim(ctx context.Context, db *gorm.DB, old *Record, new *Record) (bool, error) {
	err := db.Transaction(func(tx *gorm.DB) error {
		// 仅当记录仍具有相同的 ID 且仍过期时才删除。
		// 这是防止并发回收已将此记录替换为具有不同 ID 的新记录的防护。
		delRes := tx.Where("id = ? AND expires_at < ?", old.ID, time.Now()).
			Delete(&Record{})
		if delRes.Error != nil {
			return fmt.Errorf("回收删除: %w", delRes.Error)
		}
		if delRes.RowsAffected == 0 {
			// 另一个 goroutine 已回收此键。
			return errReclaimRace
		}

		// 插入新记录。
		if err := tx.Create(new).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		if errors.Is(err, errReclaimRace) || isDuplicateKeyError(err) {
			// 并发回收竞争失败。
			slog.InfoContext(ctx, "幂等回收竞争失败",
				"idempotency_key", new.IdempotencyKey,
				"scope", new.Scope,
			)
			return false, nil
		}
		return false, fmt.Errorf("幂等回收事务: %w", err)
	}

	slog.InfoContext(ctx, "幂等键已回收",
		"idempotency_key", new.IdempotencyKey,
		"scope", new.Scope,
		"actor_id", new.ActorID,
		"old_record_id", old.ID,
	)
	return true, nil
}

// Complete 将幂等记录标记为 StatusSucceeded 并存储响应负载以供将来重放。
//
// 参数：
//   - db：GORM 数据库句柄（可以是调用方的事务）。
//   - id：记录的主键（Record.ID，UUID v4 字符串）。
//   - code：成功响应的 HTTP 状态码。
//   - body：响应体字节。
//   - ref：对已创建资源的引用（如交易 ID）。
//
// Complete 按 ID 加载记录，验证其处于 StatusStarted 状态，
// 然后将其更新为 StatusSucceeded，同时存储响应和资源引用。
//
// 若记录未找到或不在 StatusStarted 状态则返回错误。
func Complete(ctx context.Context, db *gorm.DB, id string, code int, body []byte, ref string) error {
	var rec Record
	if err := db.First(&rec, "id = ?", id).Error; err != nil {
		return fmt.Errorf("幂等完成: 记录 %s: %w", id, err)
	}

	if rec.Status != StatusStarted {
		return fmt.Errorf("幂等完成: 记录 %s 状态为 %q，期望 %q",
			id, rec.Status, StatusStarted)
	}

	encoded := EncodeResponse(code, body)
	rec.Status = StatusSucceeded
	rec.ResponseJSON = &encoded
	rec.ResourceRef = &ref

	if err := UpdateRecord(db, &rec); err != nil {
		return fmt.Errorf("幂等完成: 保存记录 %s: %w", id, err)
	}

	slog.InfoContext(ctx, "幂等操作已完成",
		"idempotency_key", rec.IdempotencyKey,
		"scope", rec.Scope,
		"actor_id", rec.ActorID,
		"response_code", code,
		"resource_ref", ref,
	)
	return nil
}

// Fail 将幂等记录标记为 StatusFailed。这防止相同的键被用于不同负载的重试。
//
// 调用 Fail 后，后续使用相同键+指纹的 Claim 调用将返回
// ErrIdempotencyConflict（而非 ErrIdempotencyReplay），因为失败的操作不应被静默重放。
//
// 参数：
//   - db：GORM 数据库句柄。
//   - id：记录的主键（Record.ID）。
//   - errMsg：描述失败的人类可读错误消息。
//
// 若记录未找到或不在 StatusStarted 状态则返回错误。
func Fail(ctx context.Context, db *gorm.DB, id string, errMsg string) error {
	var rec Record
	if err := db.First(&rec, "id = ?", id).Error; err != nil {
		return fmt.Errorf("幂等失败: 记录 %s: %w", id, err)
	}

	if rec.Status != StatusStarted {
		return fmt.Errorf("幂等失败: 记录 %s 状态为 %q，期望 %q",
			id, rec.Status, StatusStarted)
	}

	rec.Status = StatusFailed
	// 将错误消息存储在 ResponseJSON 中以供诊断。
	encoded := fmt.Sprintf(`{"error":%q}`, errMsg)
	rec.ResponseJSON = &encoded

	if err := UpdateRecord(db, &rec); err != nil {
		return fmt.Errorf("幂等失败: 保存记录 %s: %w", id, err)
	}

	slog.ErrorContext(ctx, "幂等操作已失败",
		"idempotency_key", rec.IdempotencyKey,
		"scope", rec.Scope,
		"actor_id", rec.ActorID,
		"error", errMsg,
	)
	return nil
}

// ptrVal 返回解引用后的字符串，nil 指针返回 "<nil>"。
// 仅用于结构化日志。
func ptrVal(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}
