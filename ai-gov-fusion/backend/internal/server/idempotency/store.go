package idempotency

import (
	"errors"
	"log/slog"
	"strings"
	"time"

	"gorm.io/gorm"
)

// isDuplicateKeyError 检查 err 是否在 SQLite 和 PostgreSQL 驱动中都是重复键/唯一约束冲突。
//
// PostgreSQL：gorm.ErrDuplicatedKey（包装 pq 错误码 23505）。
// SQLite：sqlite3 驱动返回包含 "UNIQUE constraint failed" 字符串的错误。
//
// 此辅助函数避免核心 Claim 逻辑耦合到单一驱动的错误类型。
// 铁律 1：仅此函数和 InsertRecord 与表示唯一性冲突的数据库错误交互。
func isDuplicateKeyError(err error) bool {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	// SQLite 驱动（mattn/go-sqlite3）直接返回 SQLite 错误消息；
	// GORM 并非在所有版本中都将其转换为 ErrDuplicatedKey。
	if strings.Contains(err.Error(), "UNIQUE constraint failed") {
		return true
	}
	return false
}

// InsertRecord 执行向 idempotency_records 的 INSERT。
// 它依赖 UNIQUE(scope, actor_id, idempotency_key) 约束在数据库层保证原子性。
// 若约束冲突，返回 gorm.ErrDuplicatedKey——调用方必须将其处理为重放或冲突。
//
// 唯一的插入路径——禁止 SELECT 后 INSERT 模式，这会引入竞态条件（铁律 1）。
//
// 调用方负责在调用 InsertRecord 前设置以下字段：
// ID、Scope、ActorID、IdempotencyKey、RequestHash、Status（应为 StatusStarted）和 ExpiresAt。
func InsertRecord(db *gorm.DB, rec *Record) error {
	result := db.Create(rec)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// GetRecord 按复合业务键（scope, actor_id, idempotency_key）检索幂等记录。
// 若记录不存在则返回 gorm.ErrRecordNotFound。
//
// 在 InsertRecord 因重复键错误失败后调用此函数是安全的，
// 因为 UNIQUE 约束保证此时恰好存在一条记录。
func GetRecord(db *gorm.DB, scope, actorID, key string) (*Record, error) {
	var rec Record
	result := db.Where("scope = ? AND actor_id = ? AND idempotency_key = ?",
		scope, actorID, key).First(&rec)
	if result.Error != nil {
		return nil, result.Error
	}
	return &rec, nil
}

// UpdateRecord 原地更新一条现有的幂等记录。
// 通常由 Complete 或 Fail 调用以将记录从 StatusStarted 转换为
// StatusSucceeded 或 StatusFailed。
//
// 调用方必须确保记录的 ID 已设置且要更新的字段已在结构体上填充。
// GORM 将为具有非零值的特定列生成 UPDATE 语句。
func UpdateRecord(db *gorm.DB, rec *Record) error {
	result := db.Save(rec)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// CleanExpired 删除所有 expires_at 时间戳早于当前时间的幂等记录。
// 设计为被后台 goroutine 定期调用（如每 60 秒），以强制执行
// PRD §8.7（最短 24 小时）中定义的保留窗口。
//
// 返回删除的行数和任何发生的错误。
//
// 使用原始 DELETE 语句以避免将过期记录加载到内存中。
// expires_at 列具有 B-tree 索引（参见 DDL），因此 WHERE 子句是高效的。
func CleanExpired(db *gorm.DB) (int64, error) {
	result := db.Exec("DELETE FROM idempotency_records WHERE expires_at < ?", time.Now())
	if result.Error != nil {
		slog.Error("幂等清理失败",
			"error", result.Error.Error(),
		)
		return 0, result.Error
	}
	deleted := result.RowsAffected
	if deleted > 0 {
		slog.Info("幂等清理完成",
			"deleted_count", deleted,
		)
	}
	return deleted, nil
}
