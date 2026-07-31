// Package server API Key 模型辅助——网关 API Key 表（api_keys）的
// GORM 钩子、验证及常量定义。APIKey 结构体定义在 types.go 中，
// 本文件仅包含辅助代码。
package server

import (
	"errors"
	"time"
)

// ── API Key 状态常量 ──────────────────────────────────────────────────────

const (
	// APIKeyStatusActive 活跃——密钥可正常使用。
	APIKeyStatusActive = "active"
	// APIKeyStatusRevoked 已吊销——密钥被主动撤销，不可恢复。
	APIKeyStatusRevoked = "revoked"
	// APIKeyStatusExpired 已过期——密钥超过有效期自动失效。
	APIKeyStatusExpired = "expired"
)

// ── TableName 覆盖 ────────────────────────────────────────────────────────

// TableName 显式指定 GORM 表名为 api_keys，确保与已有约定一致。
// GORM 默认会将 APIKey 转为 api_keys，此处显式声明以防未来命名规则变更。
func (APIKey) TableName() string { return "api_keys" }

// ── 验证方法 ──────────────────────────────────────────────────────────────

// Validate 校验 APIKey 结构体的必填字段与约束。
// 若 KeyHash 为空、OwnerUserID 为空或 Status 无效则返回错误。
// 调用方应在 Create 或 Update 操作前调用本方法。
func (k *APIKey) Validate() error {
	if k == nil {
		return errors.New("api_key: 不能为 nil")
	}
	if k.KeyHash == "" {
		return errors.New("api_key: key_hash 为必填项")
	}
	if k.KeyPrefix == "" {
		return errors.New("api_key: key_prefix 为必填项")
	}
	if k.OwnerUserID == "" {
		return errors.New("api_key: owner_user_id 为必填项")
	}
	if k.Status == "" {
		k.Status = APIKeyStatusActive
	}
	if k.Status != APIKeyStatusActive && k.Status != APIKeyStatusRevoked && k.Status != APIKeyStatusExpired {
		return errors.New("api_key: status 必须为 active / revoked / expired 之一")
	}
	return nil
}

// IsExpired 检查密钥是否已过期。若 ExpiresAt 不为空且早于当前时间，返回 true。
func (k *APIKey) IsExpired() bool {
	if k.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*k.ExpiresAt)
}

// IsActive 检查密钥是否处于可用状态。仅当 status 为 active 且未过期时返回 true。
func (k *APIKey) IsActive() bool {
	if k.Status != APIKeyStatusActive {
		return false
	}
	return !k.IsExpired()
}

// TouchLastUsed 更新最近使用时间为当前时间。
// 调用方负责在更新后执行 db.Save。
func (k *APIKey) TouchLastUsed() {
	now := time.Now()
	k.LastUsedAt = &now
}
