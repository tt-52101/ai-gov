// Package server 治理 API 数据模型——与 TokenHub 核心模型分表存储，独立管理。
// 全部注释使用中文，符合 AGENTS.md 铁律。
package server

import (
	"time"
)

// ── GovAPIKey 治理 API 密钥模型 ──────────────────────────────────────────

// GovAPIKey 治理 API 密钥——用于 /gov/* 控制面认证。
// 与 TokenHub 的 APIKey（LLM 调用密钥，表 api_keys）分表存储，独立管理。
//
// 表名: gov_api_keys
//
// 创建时返回完整明文（一次性展示），后续 GET 只返回 KeyPrefix。
// 密钥存储使用 SHA-256 哈希（复用 HashSecret），原文不可逆向恢复。
//
// 关联约束：
//   - OwnerUserID 引用 admin_users.id ——禁人时对应 Key 立即失效。
//   - AccountID 引用 admin_resources (kind="accounts")——用于 IAM 轴权限校验。
//   - PartyID 引用 parties——组织/项目归属。
type GovAPIKey struct {
	// ID 为密钥记录唯一标识（UUID，前缀 govkey_）。
	ID string `json:"id" gorm:"primaryKey"`
	// Name 为密钥的可读名称。
	Name string `json:"name"`
	// KeyHash 为原始密钥的 SHA-256 哈希值，唯一索引，不对外返回。
	KeyHash string `json:"-" gorm:"uniqueIndex"`
	// KeyPrefix 为原始密钥的前 8 字符前缀，用于列表展示和识别。
	KeyPrefix string `json:"key_prefix"`
	// OwnerUserID 为密钥所有者用户 ID，关联 admin_users。
	// 用户被禁用后，其名下所有 GovAPIKey 立即失效。
	OwnerUserID string `json:"owner_user_id" gorm:"index"`
	// AccountID 为绑定的资金账户 ID，关联 admin_resources (kind="accounts")。
	// 用于 IAM 轴权限校验——调用方只能对 IAM 允许集合内的账户操作。
	AccountID string `json:"account_id,omitempty" gorm:"index"`
	// PartyID 为绑定的组织/项目 ID，关联 parties。
	PartyID string `json:"party_id,omitempty" gorm:"index"`
	// Status 为密钥状态：active / disabled / revoked。
	Status string `json:"status"`
	// ExpiresAt 为可选的过期时间，过期后 validateGovToken 拒绝该密钥。
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	// CreatedAt 为创建时间。
	CreatedAt time.Time `json:"created_at"`
	// LastUsedAt 为最近一次使用时间（validateGovToken 通过时更新）。
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
}

// TableName v3.2: 合并到 api_keys 表，与 TokenHub APIKey 共享同一个表。
func (GovAPIKey) TableName() string { return "api_keys" }

// ── GovCreateKeyRequest Key 创建请求 ────────────────────────────────────

// GovCreateKeyRequest 治理 API 密钥创建请求体。
type GovCreateKeyRequest struct {
	// Name 为密钥名称，必填。
	Name string `json:"name"`
	// OwnerUserID 为密钥所有者用户 ID，必填。
	OwnerUserID string `json:"owner_user_id"`
	// AccountID 为绑定的资金账户 ID，可选。
	AccountID string `json:"account_id,omitempty"`
	// PartyID 为绑定的组织/项目 ID，可选。
	PartyID string `json:"party_id,omitempty"`
	// KeyPrefix 为自定义密钥前缀，可选（默认 "gov_"）。
	KeyPrefix string `json:"key_prefix,omitempty"`
}

// ── GovKeyResponse Key 查询/列表响应 ────────────────────────────────────

// GovKeyResponse 治理 API 密钥查询响应——不含明文。
// 源码中永远不会包含 KeyHash，公开字段仅 KeyPrefix 用于识别。
type GovKeyResponse struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	KeyPrefix   string     `json:"key_prefix"`
	OwnerUserID string     `json:"owner_user_id"`
	AccountID   string     `json:"account_id,omitempty"`
	PartyID     string     `json:"party_id,omitempty"`
	Status      string     `json:"status"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
}

// GovCreatedKeyResponse 密钥创建成功响应——包含一次性明文。
// 明文仅在创建时返回，之后无法通过任何 API 再次获取。
type GovCreatedKeyResponse struct {
	GovKeyResponse
	// RawKey 为完整的明文密钥，仅在创建时返回一次。
	RawKey string `json:"raw_key"`
}

// fromGovAPIKey 将 GovAPIKey 转换为对外响应（不含明文）。
func fromGovAPIKey(key GovAPIKey) GovKeyResponse {
	return GovKeyResponse{
		ID:          key.ID,
		Name:        key.Name,
		KeyPrefix:   key.KeyPrefix,
		OwnerUserID: key.OwnerUserID,
		AccountID:   key.AccountID,
		PartyID:     key.PartyID,
		Status:      key.Status,
		ExpiresAt:   key.ExpiresAt,
		CreatedAt:   key.CreatedAt,
		LastUsedAt:  key.LastUsedAt,
	}
}
