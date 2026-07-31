package security

import (
	"context"
	"errors"
)

// ── 网络分类常量 ──────────────────────────────────────────────────────────

const (
	// NetworkInternal 内网——请求的目的地为内部私有化部署模型。
	NetworkInternal = "internal"

	// NetworkExternal 外网——请求的目的地为公有云 API（如 OpenAI、Anthropic）。
	NetworkExternal = "external"
)

// ── 用户出网策略常量 ──────────────────────────────────────────────────────

const (
	// EgressPolicyInternalOnly 仅允许内网——外网请求直接阻断（PRD D-CON-02）。
	EgressPolicyInternalOnly = "INTERNAL_ONLY"

	// EgressPolicyHybridAllowed 混合模式——允许外网但受白名单约束。
	EgressPolicyHybridAllowed = "HYBRID_ALLOWED"

	// EgressPolicyOpenAll 全开放——不限制外网访问。
	EgressPolicyOpenAll = "OPEN_ALL"
)

// ── 出网管控类型 ──────────────────────────────────────────────────────────

// User 出网管控视角下的用户摘要——仅包含网络策略判定所需的最小字段集。
//
// 后续阶段可从 users 表或缓存中加载完整 Profile。
type User struct {
	// ID 用户唯一标识。
	ID string

	// EgressPolicy 用户的出网策略（INTERNAL_ONLY / HYBRID_ALLOWED / OPEN_ALL）。
	EgressPolicy string
}

// Model 出网管控视角下的模型摘要——仅包含网络分类判定所需的最小字段集。
type Model struct {
	// ID 模型唯一标识。
	ID string

	// Name 模型名称。
	Name string

	// NetworkClass 模型网络分类（internal / external）。
	NetworkClass string
}

// ── 出网管控 ──────────────────────────────────────────────────────────────

// ErrEgressBlocked 出网阻断错误——当用户策略禁止请求发往外网时返回。
var ErrEgressBlocked = errors.New("security: 出网请求被阻断")

// CheckEgress 检查用户是否有权向目标模型发起外网请求。
//
// 判定规则（PRD SEC-01 / SEC-02）：
//   - INTERNAL_ONLY 用户请求 external 模型：直接阻断，零外网流量（D-CON-02）。
//   - HYBRID_ALLOWED 用户请求 external 模型：当前阶段放行（白名单校验尚未实现）。
//   - 内网模型：所有用户放行。
//
// 返回值：
//   - nil: 请求允许发送。
//   - ErrEgressBlocked: 请求被出网策略阻断。
//
// 注意：本函数当前为 P2 骨架——HYBRID_ALLOWED 的白名单校验留待阶段 D 实现。
func CheckEgress(ctx context.Context, user User, targetModel Model) error {
	// 内网模型——所有用户均可访问。
	if targetModel.NetworkClass == NetworkInternal {
		return nil
	}

	// 外网模型——按用户出网策略判定。
	switch user.EgressPolicy {
	case EgressPolicyInternalOnly:
		// 严禁外网流量（D-CON-02）。
		return ErrEgressBlocked

	case EgressPolicyHybridAllowed:
		// P2 骨架：当前阶段放行所有 HYBRID_ALLOWED 用户。
		// 阶段 D 将接入白名单校验：
		//   1. 查询 sys_config 中的外网白名单。
		//   2. 校验目标模型是否在白名单中。
		//   3. 不在白名单则返回 ErrEgressBlocked。
		return nil

	case EgressPolicyOpenAll:
		return nil

	default:
		// 未知策略——保守拒绝。
		return ErrEgressBlocked
	}
}
