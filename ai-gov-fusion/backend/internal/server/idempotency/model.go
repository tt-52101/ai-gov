// Package idempotency 提供基于幂等键的变更操作去重机制。
// 保证余额变更、划拨请求和管理操作的最多执行一次语义。
package idempotency

import (
	"encoding/json"
	"time"
)

// 状态常量定义幂等记录的生命周期。状态机：started → succeeded | failed。一旦到达终态，状态永不改变。
const (
	// StatusStarted 标记一个新申请的幂等键，其操作尚未完成。
	// 调用方必须最终调用 Complete 或 Fail 以脱离此状态。
	StatusStarted = "started"

	// StatusSucceeded 标记一条已完成的幂等记录，其响应存储在 ResponseJSON 中。
	// 相同键+哈希的重放将返回存储的响应而不重新执行操作。
	StatusSucceeded = "succeeded"

	// StatusFailed 标记一条操作失败的幂等记录。
	// 失败记录阻止调用方使用相同键重试。
	StatusFailed = "failed"
)

// Record 表示 idempotency_records 表中的一行。
// 它存储幂等键保护的写操作的状态。
//
// 字段映射自 DDL（schema/ai-gov-fusion-v3.2.sql 表 29）：
//
//	id              TEXT PRIMARY KEY          — UUID v4
//	scope           TEXT NOT NULL             — allocate / liquidate / compensate
//	actor_id        TEXT NOT NULL             — 用户或系统主体
//	idempotency_key TEXT NOT NULL             — 客户端提供的 UUID v4，≤255 字符
//	request_hash    TEXT NOT NULL             — 请求体的 SHA-256 哈希
//	status          TEXT NOT NULL DEFAULT 'started'
//	response_json   JSONB                     — 首次成功响应快照
//	resource_ref    TEXT                      — 如 transaction_id
//	expires_at      TIMESTAMPTZ NOT NULL      — ≥24h 窗口（PRD §8.7）
//
// UNIQUE(scope, actor_id, idempotency_key) — 由数据库层通过
// Scope、ActorID 和 IdempotencyKey 字段上的 uniqueIndex:idx_idempotency_scope_actor_key 强制执行。
// GORM 的 AutoMigrate 创建此复合索引。
type Record struct {
	// ID 是主键，由调用方在调用 Claim 前生成为 UUID v4。
	// 按 DDL 存储为 TEXT。
	ID string `json:"id" gorm:"primaryKey"`

	// Scope 标识操作域（如 "allocate"、"liquidate"）。
	// 属于唯一约束：(scope, actor_id, idempotency_key)。
	Scope string `json:"scope" gorm:"uniqueIndex:idx_idempotency_scope_actor_key,priority:1;not null"`

	// ActorID 标识执行操作的用户或系统主体。
	// 属于唯一约束：(scope, actor_id, idempotency_key)。
	ActorID string `json:"actor_id" gorm:"uniqueIndex:idx_idempotency_scope_actor_key,priority:2;not null;column:actor_id"`

	// IdempotencyKey 是客户端提供的 UUID v4 键（最大 255 字符）。
	// 属于唯一约束：(scope, actor_id, idempotency_key)。
	IdempotencyKey string `json:"idempotency_key" gorm:"uniqueIndex:idx_idempotency_scope_actor_key,priority:3;not null;column:idempotency_key"`

	// RequestHash 是请求负载的 SHA-256 十六进制摘要。
	// 用于检测重放（相同哈希）vs 冲突（不同哈希）。
	RequestHash string `json:"request_hash" gorm:"not null;column:request_hash"`

	// Status 追踪操作生命周期：started → succeeded | failed。
	// 参见 StatusStarted、StatusSucceeded、StatusFailed 常量。
	Status string `json:"status" gorm:"not null;default:started"`

	// ResponseJSON 将首次成功响应存储为 JSONB 快照。
	// 重放时，此值直接返回而不重新执行操作。
	// 操作尚未完成时为 nil。
	ResponseJSON *string `json:"response_json,omitempty" gorm:"type:jsonb;column:response_json"`

	// ResourceRef 存储对已创建资源的引用，如交易 ID 或划拨 ID。
	// 操作尚未产生资源时为 nil。
	ResourceRef *string `json:"resource_ref,omitempty" gorm:"column:resource_ref"`

	// ExpiresAt 是该记录可被后台 CleanExpired 任务清理的时间。
	// 保留窗口至少为 24 小时 per PRD §8.7。
	ExpiresAt time.Time `json:"expires_at" gorm:"not null;index;column:expires_at"`

	// CreatedAt 在插入时自动设置。实现 GORM 的自动创建时间戳约定。
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at"`

	// UpdatedAt 在插入和更新时自动设置。实现 GORM 的自动更新时间戳约定。
	UpdatedAt time.Time `json:"updated_at" gorm:"column:updated_at"`
}

// TableName 覆盖 GORM 默认表名以匹配 DDL。
func (Record) TableName() string {
	return "idempotency_records"
}

// ResponsePayload 是存储在 ResponseJSON 中的结构化内容。
// 它捕获已完成幂等操作的 HTTP 状态码和响应体，实现精确重放。
type ResponsePayload struct {
	// Code 是原始操作返回的 HTTP 状态码。
	Code int `json:"code"`

	// Body 是原始操作返回的响应体。
	Body json.RawMessage `json:"body"`
}

// EncodeResponse 将响应码和体序列化为适合存储在 Record.ResponseJSON 中的 JSON 字符串。
func EncodeResponse(code int, body []byte) string {
	payload, _ := json.Marshal(ResponsePayload{Code: code, Body: body})
	return string(payload)
}

// DecodeResponse 从存储的 ResponseJSON 值中提取 HTTP 状态码和体。
// 若存储的 JSON 无效或为空则返回零值。
func DecodeResponse(raw *string) (code int, body []byte) {
	if raw == nil || *raw == "" {
		return 0, nil
	}
	var p ResponsePayload
	if err := json.Unmarshal([]byte(*raw), &p); err != nil {
		return 0, nil
	}
	return p.Code, p.Body
}
