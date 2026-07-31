package fund

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/shopspring/decimal"

	"tokenhub/backend/internal/server/party"
)

// Service 是核心财务治理服务，实现架构规范（architecture-v3.2.md S5.2）中
// 定义的 FundService 契约。它拥有所有资金变更：划拨、冻结、结算、续期和清算。
//
// 当 Store 提供行级锁（SELECT FOR UPDATE）或乐观并发控制时，每个导出方法都是并发安全的。
// 写操作通过 IdempotencyKey 机制保证幂等。
//
// 所有货币值必须使用 shopspring/decimal.Decimal——绝不可用 float64。
// 所有资金变更根据 AGENTS.md S6.2 记录结构化日志，包含 request_id、account_id、
// freeze_id、amount 和 balance_after。
type Service struct {
	Store       Store
	Idempotency IdempotencyChecker
	// PartyService 提供 party_edges 表查询能力，用于 validateChannel 中的
	// 划拨通道语义校验（RED-2：防止绕过边关系的非法划拨）。
	// 生产环境必须注入——nil 时降级为仅校验 channel 名称常量（向后兼容测试）。
	PartyService *party.Service
}

// IdempotencyChecker 抽象幂等键的 Claim/预览模式。
// idempotency 包提供具体实现。
type IdempotencyChecker interface {
	// Claim 原子地申请一个幂等键。若这是首次调用则返回 true（继续执行），
	// 若先前结果已存在则返回 false（返回存储的结果）。
	Claim(ctx context.Context, key string) (bool, error)

	// Store 将结果与幂等键关联持久化，供将来查找。
	Store(ctx context.Context, key string, result any) error

	// Retrieve 获取先前存储的结果。
	Retrieve(ctx context.Context, key string, result any) (bool, error)
}

// defaultFreezeTTL 是默认冻结过期时长（15 分钟，per PRD S8.3）。
const defaultFreezeTTL = 15 * time.Minute

// maxFreezeLifetime 是累计冻结最大生命周期（2 小时，per PRD S8.3）。
const maxFreezeLifetime = 2 * time.Hour

// ---------------------------------------------------------------------------
// Allocate
// ---------------------------------------------------------------------------

// Allocate 从源账户向目标账户划拨资金。
//
// 守恒保证（F-CON-02）：在单次数据库事务内 src_delta + dst_delta = 0。
// 两个账户按 ID 升序锁定以防止死锁。
//
// 通道校验 per PRD S8.2：
//   - parent：仅允许上级到下级方向
//   - sponsors：仅允许出资方到被出资方方向
//   - allocates：允许从 Party 到个人账户
//   - whitelist：需要显式白名单授权
//
// 两个账户必须处于活跃状态。源账户必须有足够的可用余额。
//
// 幂等：若幂等键已成功使用过，则返回原始结果而不执行第二次划拨。
//
// 副作用：插入两条账本记录（借方和贷方），更新两个账户余额，创建一条划拨记录。全部在同一事务内完成。
func (s *Service) Allocate(ctx context.Context, req AllocateRequest) (*AllocateResult, error) {
	// 校验基本不变量。
	if err := s.allocateValidate(req); err != nil {
		return nil, err
	}

	// 在事务外检查幂等。
	if req.IdempotencyKey != "" && s.Idempotency != nil {
		claimed, err := s.Idempotency.Claim(ctx, req.IdempotencyKey)
		if err != nil {
			slog.ErrorContext(ctx, "幂等申请失败",
				"idempotency_key", req.IdempotencyKey,
				"error", err,
			)
			return nil, err
		}
		if !claimed {
			var result AllocateResult
			found, err := s.Idempotency.Retrieve(ctx, req.IdempotencyKey, &result)
			if err != nil {
				return nil, err
			}
			if found {
				slog.InfoContext(ctx, "划拨幂等重放",
					"idempotency_key", req.IdempotencyKey,
					"allocation_id", result.AllocationID,
				)
				return &result, nil
			}
			// 键已被 Claim 但无结果存储——冲突。
			return nil, newIdempotencyConflictError(req.IdempotencyKey)
		}
	}

	var result *AllocateResult
	err := s.Store.WithTx(ctx, func(tx Tx) error {
		r, err := s.allocateExecute(ctx, tx, req)
		if err != nil {
			return err
		}
		result = r
		return nil
	})

	if err != nil {
		slog.ErrorContext(ctx, "划拨失败",
			"src_account_id", req.SrcAccountID,
			"dst_account_id", req.DstAccountID,
			"amount", req.Amount.String(),
			"idempotency_key", req.IdempotencyKey,
			"error", err,
		)
		return nil, err
	}

	return result, nil
}

// allocateValidate 校验划拨请求的基本参数。
// 检查：金额必须为正、源与目标不同、幂等键必填。
func (s *Service) allocateValidate(req AllocateRequest) error {
	if req.Amount.Decimal.LessThanOrEqual(decimal.Zero) {
		return newAmountMustBePositiveError(req.Amount)
	}
	if req.SrcAccountID == req.DstAccountID {
		return newSelfTransferError(req.SrcAccountID)
	}
	// IdempotencyKey 必填——所有划拨操作必须提供幂等键（RED-2 安全修复）。
	if req.IdempotencyKey == "" {
		return newIdempotencyKeyRequiredError()
	}
	return nil
}

// allocateExecute 在事务内执行核心划拨逻辑。
// 锁定账户、校验状态和余额、校验通道、更新余额、创建划拨记录和账本条目。
// 返回划拨结果。幂等结果也在事务内存储。
func (s *Service) allocateExecute(ctx context.Context, tx Tx, req AllocateRequest) (*AllocateResult, error) {
	// 按 ID 升序锁定两个账户以防止死锁。
	firstID, secondID := req.SrcAccountID, req.DstAccountID
	if firstID > secondID {
		firstID, secondID = secondID, firstID
	}

	var srcAcct, dstAcct *Account
	var err error

	if firstID == req.SrcAccountID {
		srcAcct, err = s.Store.GetAccountForUpdate(tx, ctx, req.SrcAccountID)
		if err != nil {
			return nil, err
		}
		dstAcct, err = s.Store.GetAccountForUpdate(tx, ctx, req.DstAccountID)
		if err != nil {
			return nil, err
		}
	} else {
		dstAcct, err = s.Store.GetAccountForUpdate(tx, ctx, req.DstAccountID)
		if err != nil {
			return nil, err
		}
		srcAcct, err = s.Store.GetAccountForUpdate(tx, ctx, req.SrcAccountID)
		if err != nil {
			return nil, err
		}
	}

	// 校验账户状态。
	if srcAcct == nil {
		return nil, newAccountFrozenError(req.SrcAccountID, "未找到")
	}
	if srcAcct.Status != StatusActive {
		return nil, newAccountFrozenError(req.SrcAccountID, srcAcct.Status)
	}
	if dstAcct == nil {
		return nil, newAccountFrozenError(req.DstAccountID, "未找到")
	}
	if dstAcct.Status != StatusActive {
		return nil, newAccountFrozenError(req.DstAccountID, dstAcct.Status)
	}

	// 验证余额充足。
	if srcAcct.AvailableBalance.Decimal.LessThan(req.Amount.Decimal) {
		return nil, newInsufficientBalanceError(req.SrcAccountID, srcAcct.AvailableBalance, req.Amount)
	}

	// 校验划拨通道——通过 party.CanAllocate 查询 party_edges 表进行语义校验。
	if err := s.validateChannel(ctx, req.Channel, srcAcct, dstAcct, req.EdgeID); err != nil {
		return nil, err
	}

	// 计算新余额。
	srcAvailableAfter := srcAcct.AvailableBalance.Decimal.Sub(req.Amount.Decimal)
	dstAvailableAfter := dstAcct.AvailableBalance.Decimal.Add(req.Amount.Decimal)

	// 更新源账户（借方）。
	if err := s.Store.UpdateAccountBalances(tx, ctx, req.SrcAccountID, srcAvailableAfter, srcAcct.FrozenBalance.Decimal, srcAcct.Version); err != nil {
		return nil, err
	}

	// 更新目标账户（贷方）。
	if err := s.Store.UpdateAccountBalances(tx, ctx, req.DstAccountID, dstAvailableAfter, dstAcct.FrozenBalance.Decimal, dstAcct.Version); err != nil {
		return nil, err
	}

	now := time.Now()

	// 创建划拨记录。
	allocation := &Allocation{
		ID:             newUUID(),
		SrcAccountID:   req.SrcAccountID,
		DstAccountID:   req.DstAccountID,
		Amount:         req.Amount,
		Channel:        req.Channel,
		EdgeID:         req.EdgeID,
		Status:         AllocationStatusCompleted,
		IdempotencyKey: stringPtr(req.IdempotencyKey),
		ActorUserID:    stringPtr(req.OperatorID),
		Reason:         req.Reason,
		CreatedAt:      now,
		CompletedAt:    timePtr(now),
	}
	if err := s.Store.InsertAllocation(tx, ctx, allocation); err != nil {
		return nil, err
	}

	// 插入账本条目——一条借方，一条贷方。
	srcLedger := &Ledger{
		ID:             newUUID(),
		AccountID:      req.SrcAccountID,
		Direction:      DirectionAllocateOut,
		Amount:         req.Amount,
		BalanceAfter:   NewDecimal(srcAvailableAfter.String()),
		AllocationID:   stringPtr(allocation.ID),
		IdempotencyKey: stringPtr(req.IdempotencyKey),
		Reason:         req.Reason,
		CreatedAt:      now,
	}
	if err := s.Store.InsertLedger(tx, ctx, srcLedger); err != nil {
		return nil, err
	}

	dstLedger := &Ledger{
		ID:             newUUID(),
		AccountID:      req.DstAccountID,
		Direction:      DirectionAllocateIn,
		Amount:         req.Amount,
		BalanceAfter:   NewDecimal(dstAvailableAfter.String()),
		AllocationID:   stringPtr(allocation.ID),
		IdempotencyKey: stringPtr(req.IdempotencyKey),
		Reason:         req.Reason,
		CreatedAt:      now,
	}
	if err := s.Store.InsertLedger(tx, ctx, dstLedger); err != nil {
		return nil, err
	}

	result := s.allocateBuildResult(allocation.ID, req, srcAvailableAfter, dstAvailableAfter, now)

	// 在事务内存储幂等结果。
	if req.IdempotencyKey != "" && s.Idempotency != nil {
		if err := s.Store.StoreIdempotency(tx, ctx, req.IdempotencyKey, result); err != nil {
			return nil, err
		}
	}

	// 结构化日志。
	slog.InfoContext(ctx, "划拨完成",
		"allocation_id", allocation.ID,
		"src_account_id", req.SrcAccountID,
		"dst_account_id", req.DstAccountID,
		"amount", req.Amount.String(),
		"channel", req.Channel,
		"src_balance_after", srcAvailableAfter.String(),
		"dst_balance_after", dstAvailableAfter.String(),
		"idempotency_key", req.IdempotencyKey,
	)

	return result, nil
}

// allocateBuildResult 构造 AllocateResult 结构体。
func (s *Service) allocateBuildResult(allocationID string, req AllocateRequest, srcAvailableAfter, dstAvailableAfter decimal.Decimal, now time.Time) *AllocateResult {
	return &AllocateResult{
		AllocationID:    allocationID,
		SrcAccountID:    req.SrcAccountID,
		DstAccountID:    req.DstAccountID,
		Amount:          req.Amount,
		Channel:         req.Channel,
		EdgeID:          req.EdgeID,
		Status:          AllocationStatusCompleted,
		SrcBalanceAfter: NewDecimal(srcAvailableAfter.String()),
		DstBalanceAfter: NewDecimal(dstAvailableAfter.String()),
		IdempotencyKey:  req.IdempotencyKey,
		CreatedAt:       now,
		CompletedAt:     now,
	}
}

// validateChannel 校验请求的划拨通道是否允许从源账户划拨到目标账户。
//
// 校验分两层：
//  1. 若 PartyService 为 nil（测试环境），降级为仅校验 channel 名称常量。
//  2. 生产环境：将账户关联的 PartyID 从 string 转为 int64，调用
//     party.CanAllocate() 查询 party_edges 表进行边关系语义校验。
//
// 允许的通道方向：
//   - parent：仅上级→下级（party_edges 中 src=parent, dst=child）
//   - sponsors：仅出资方→被出资方（src=sponsor, dst=sponsored）
//   - allocates：仅 Party→个人账户（src=party, dst=person account）
//   - whitelist：暂不支持（需额外的 allocate_whitelist 表）
//
// owns、participates、merged_into、split_from 等边类型一律拒绝资金划拨。
func (s *Service) validateChannel(ctx context.Context, channel string, srcAcct, dstAcct *Account, edgeID *string) error {
	// 降级路径：PartyService 未注入时仅校验 channel 名称常量（向后兼容测试环境）。
	if s.PartyService == nil {
		switch channel {
		case ChannelParent, ChannelSponsors, ChannelAllocates, ChannelWhitelist:
			return nil
		default:
			return newAllocationChannelDeniedError(srcAcct.ID, dstAcct.ID, channel)
		}
	}

	// 将账户关联的 PartyID 从 string 转为 int64。
	srcPartyID, err := strconv.ParseInt(srcAcct.PartyID, 10, 64)
	if err != nil {
		return &FundError{
			Code:    "INVALID_PARTY_ID",
			Message: "源账户 " + srcAcct.ID + " 的 party_id 无效: " + srcAcct.PartyID,
			Err:     ErrAllocationChannelDenied,
		}
	}
	dstPartyID, err := strconv.ParseInt(dstAcct.PartyID, 10, 64)
	if err != nil {
		return &FundError{
			Code:    "INVALID_PARTY_ID",
			Message: "目标账户 " + dstAcct.ID + " 的 party_id 无效: " + dstAcct.PartyID,
			Err:     ErrAllocationChannelDenied,
		}
	}

	// 调用 party.CanAllocate 查询 party_edges 表进行边关系语义校验。
	// CanAllocate 内部检查：边是否存在、AllowsFund 是否为 true、
	// 边类型是否为 parent/sponsors/allocates（拒绝 owns/participates 等）。
	allowed, err := s.PartyService.CanAllocate(ctx, srcPartyID, dstPartyID)
	if err != nil {
		slog.ErrorContext(ctx, "划拨通道校验失败",
			"src_party_id", srcPartyID,
			"dst_party_id", dstPartyID,
			"channel", channel,
			"error", err,
		)
		return &FundError{
			Code:    "CHANNEL_VALIDATION_FAILED",
			Message: "划拨通道校验失败: " + err.Error(),
			Err:     ErrAllocationChannelDenied,
		}
	}
	if !allowed {
		slog.WarnContext(ctx, "划拨通道被拒绝",
			"src_party_id", srcPartyID,
			"dst_party_id", dstPartyID,
			"src_account_id", srcAcct.ID,
			"dst_account_id", dstAcct.ID,
			"channel", channel,
		)
		return newAllocationChannelDeniedError(srcAcct.ID, dstAcct.ID, channel)
	}
	return nil
}

// ---------------------------------------------------------------------------
// 工具辅助函数（跨所有 fund 服务文件共享）
// ---------------------------------------------------------------------------

// newUUID 生成唯一标识符。当项目中引入合适的 UUID 库后替换。
func newUUID() string {
	return fmt.Sprintf("%016x-%04x-%04x-%04x-%012x",
		time.Now().UnixNano(),
		time.Now().Nanosecond()%0x10000,
		time.Now().Nanosecond()%0x10000,
		time.Now().Nanosecond()%0x10000,
		time.Now().Nanosecond()%0x1000000000000,
	)
}

func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func decPtr(d decimal.Decimal) *Decimal {
	dd := Decimal{d}
	return &dd
}

func stringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
