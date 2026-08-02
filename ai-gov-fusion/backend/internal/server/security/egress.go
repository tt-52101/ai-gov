package security

import (
	"context"
	"errors"
	"log/slog"
	"sync"
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

// ErrEgressNotWhitelisted 白名单阻断错误——HYBRID_ALLOWED 用户请求的目标模型不在白名单中。
var ErrEgressNotWhitelisted = errors.New("security: 出网目标不在白名单中，请求被阻断")

// ── 外网白名单管理 ──────────────────────────────────────────────────────────

// egressWhitelist 存储允许 HYBRID_ALLOWED 用户访问的外网模型名称集合。
// 通过 SetEgressWhitelist 在启动时配置，运行时线程安全（读多写少）。
var egressWhitelist struct {
	mu   sync.RWMutex
	set  map[string]struct{} // 模型名称集合，O(1) 查询
}

// SetEgressWhitelist 设置外网白名单——允许 HYBRID_ALLOWED 用户访问的外网模型名称列表。
// 可在启动时由配置加载，运行期间可通过治理 API 动态更新。
// 传入空列表或 nil 表示不允许任何外网访问（白名单为空时所有 HYBRID_ALLOWED 外网请求均被阻断）。
func SetEgressWhitelist(models []string) {
	egressWhitelist.mu.Lock()
	defer egressWhitelist.mu.Unlock()
	if len(models) == 0 {
		egressWhitelist.set = make(map[string]struct{})
		return
	}
	egressWhitelist.set = make(map[string]struct{}, len(models))
	for _, m := range models {
		egressWhitelist.set[m] = struct{}{}
	}
	slog.Info("外网白名单已更新", "count", len(egressWhitelist.set))
}

// isEgressWhitelisted 检查目标模型是否在外网白名单中。
func isEgressWhitelisted(modelName string) bool {
	egressWhitelist.mu.RLock()
	defer egressWhitelist.mu.RUnlock()
	_, ok := egressWhitelist.set[modelName]
	return ok
}

// CheckEgress 检查用户是否有权向目标模型发起外网请求。
//
// 判定规则（PRD SEC-01 / SEC-02）：
//   - INTERNAL_ONLY 用户请求 external 模型：直接阻断，零外网流量（D-CON-02）。
//   - HYBRID_ALLOWED 用户请求 external 模型：检查白名单，不在白名单中则阻断。
//   - 内网模型：所有用户放行。
//
// 返回值：
//   - nil: 请求允许发送。
//   - ErrEgressBlocked: 请求被出网策略阻断。
//   - ErrEgressNotWhitelisted: HYBRID_ALLOWED 用户请求的目标不在白名单中。
func CheckEgress(ctx context.Context, user User, targetModel Model) error {
	// 内网模型——所有用户均可访问。
	if targetModel.NetworkClass == NetworkInternal {
		return nil
	}

	// 外网模型——按用户出网策略判定。
	switch user.EgressPolicy {
	case EgressPolicyInternalOnly:
		// 严禁外网流量（D-CON-02）。
		slog.WarnContext(ctx, "出网管控阻断",
			"user_id", user.ID,
			"policy", user.EgressPolicy,
			"target_model", targetModel.Name,
			"reason", "INTERNAL_ONLY 禁止外网流量",
		)
		return ErrEgressBlocked

	case EgressPolicyHybridAllowed:
		// 白名单校验：只有白名单中的外网模型才允许访问。
		if !isEgressWhitelisted(targetModel.Name) {
			slog.WarnContext(ctx, "出网管控阻断",
				"user_id", user.ID,
				"policy", user.EgressPolicy,
				"target_model", targetModel.Name,
				"reason", "目标模型不在外网白名单中",
			)
			return ErrEgressNotWhitelisted
		}
		return nil

	case EgressPolicyOpenAll:
		return nil

	default:
		// 未知策略——保守拒绝。
		slog.WarnContext(ctx, "出网管控阻断",
			"user_id", user.ID,
			"policy", user.EgressPolicy,
			"target_model", targetModel.Name,
			"reason", "未知出网策略",
		)
		return ErrEgressBlocked
	}
}
