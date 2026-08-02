package modelgrant

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// 模型授权判定错误。
var (
	// ErrModelAccessDenied 模型访问被拒绝（无 ALLOW 规则或命中 DENY 规则）。
	ErrModelAccessDenied = errors.New("MODEL_ACCESS_DENIED")
	// ErrModelBudgetExceeded 模型级预算超限。
	ErrModelBudgetExceeded = errors.New("MODEL_BUDGET_EXCEEDED")
	// ErrNoModelGrantFound 未找到任何模型授权规则。
	ErrNoModelGrantFound = errors.New("未找到模型授权规则")
)

// Checker 模型访问权限检查器。
// 在数据面 Pipeline 第 4 步执行，判断主体是否有权访问指定模型。
type Checker struct {
	// DB 数据库连接，用于查询 model_grants 表与更新配额消耗。
	DB *gorm.DB
}

// NewChecker 创建一个新的 Checker 实例。
func NewChecker(db *gorm.DB) *Checker {
	return &Checker{DB: db}
}

// CheckAccess 检查主体是否有权访问指定模型。
//
// 级联顺序：Key > Person > Party > 全局默认。
// DENY 优先于 ALLOW——任一层级命中 DENY 规则即立即拒绝。
// 禁止仅因 Leader 头衔自动拥有全平台模型权（A-CON-05）。
//
// 返回值：
//   - nil 表示允许访问
//   - ErrModelAccessDenied 表示拒绝访问
//   - 其他错误表示查询异常
func (c *Checker) CheckAccess(ctx context.Context, principal Principal, modelID string) error {
	// 收集与主体相关的所有规则并按级联顺序评估。
	mgs, err := c.loadGrantsForCascade(principal)
	if err != nil {
		return err
	}

	// 第 1 遍：DENY 优先——任意匹配即拒绝。
	for _, mg := range mgs {
		if mg.Effect == EffectDeny && matchGrant(mg, principal, modelID) {
			slog.InfoContext(ctx, "模型访问拒绝——命中 DENY 规则",
				"principal_type", principal.Type,
				"principal_id", principal.ID,
				"model_id", modelID,
				"grant_id", mg.ID,
			)
			return ErrModelAccessDenied
		}
	}

	// 第 2 遍：查找 ALLOW——至少需要一个匹配。
	for _, mg := range mgs {
		if mg.Effect == EffectAllow && matchGrant(mg, principal, modelID) {
			return nil
		}
	}

	// 无匹配规则 → 默认拒绝（A-CON-02：最小权限默认）。
	slog.InfoContext(ctx, "模型访问拒绝——无匹配 ALLOW 规则",
		"principal_type", principal.Type,
		"principal_id", principal.ID,
		"model_id", modelID,
	)
	return ErrModelAccessDenied
}

// CheckQuotaLimit 检查模型级配额（双层预算第二层）。
//
// 若存在匹配的 ModelGrant 且其 quota_limit 已配置，
// 则判断已消耗金额 + 本次预估金额是否超过配额上限。
// 超出则返回 ErrModelBudgetExceeded。
//
// 参数：
//   - principal：请求主体
//   - modelID：目标模型 ID
//   - estimatedSell：本次请求预估内部结算金额
func (c *Checker) CheckQuotaLimit(ctx context.Context, principal Principal, modelID string, estimatedSell decimal.Decimal) error {
	mg, err := c.findQuotaGrant(principal, modelID)
	if err != nil {
		// 未配置配额 → 不限，通过。
		if errors.Is(err, ErrNoModelGrantFound) {
			return nil
		}
		return err
	}

	// 配额未配置 → 不限。
	if mg.QuotaLimit == nil {
		return nil
	}

	// 已消耗 + 预估 > 上限 → 超限。
	predicted := mg.QuotaConsumed.Add(estimatedSell)
	if predicted.GreaterThan(*mg.QuotaLimit) {
		slog.WarnContext(ctx, "模型预算超限",
			"principal_type", principal.Type,
			"principal_id", principal.ID,
			"model_id", modelID,
			"quota_limit", mg.QuotaLimit.String(),
			"quota_consumed", mg.QuotaConsumed.String(),
			"estimated_sell", estimatedSell.String(),
			"predicted", predicted.String(),
		)
		return ErrModelBudgetExceeded
	}

	return nil
}

// ConsumeQuota 调用成功后累加模型级预算消耗。
//
// 该函数在请求完成后调用，原子地将 sellAmount 累加到匹配的 ModelGrant.QuotaConsumed 字段。
// 使用乐观锁确保并发安全——若并发写入导致版本冲突，调用方需重试。
//
// 参数：
//   - principal：请求主体
//   - modelID：已调用的模型 ID
//   - sellAmount：实际结算金额
func (c *Checker) ConsumeQuota(ctx context.Context, principal Principal, modelID string, sellAmount decimal.Decimal) error {
	mg, err := c.findQuotaGrant(principal, modelID)
	if err != nil {
		// 未配置配额 → 无需记录。
		if errors.Is(err, ErrNoModelGrantFound) {
			return nil
		}
		return err
	}

	// 无需配额则跳过。
	if mg.QuotaLimit == nil {
		return nil
	}

	// 乐观锁更新——WHERE 子句包含 version 字段，防止并发竞态。
	// 若并发写入导致版本冲突，RowsAffected 为 0，调用方应重试。
	newConsumed := mg.QuotaConsumed.Add(sellAmount)
	newVersion := mg.Version + 1
	result := c.DB.Model(&ModelGrant{}).
		Where("id = ? AND version = ?", mg.ID, mg.Version).
		Updates(map[string]any{
			"quota_consumed": newConsumed,
			"version":        newVersion,
		})
	if result.Error != nil {
		return fmt.Errorf("modelgrant: 更新配额消耗失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("modelgrant: 配额消耗更新冲突——并发写入，请重试")
	}

	slog.InfoContext(ctx, "模型配额消耗已更新",
		"principal_type", principal.Type,
		"principal_id", principal.ID,
		"model_id", modelID,
		"grant_id", mg.ID,
		"version", newVersion,
		"added", sellAmount.String(),
		"total_consumed", newConsumed.String(),
	)
	return nil
}

// ── 内部辅助 ────────────────────────────────────────────────────────────────

// loadGrantsForCascade 按级联顺序（Key > Person > Party > 全局默认）加载主体相关的所有授权规则。
func (c *Checker) loadGrantsForCascade(principal Principal) ([]*ModelGrant, error) {
	var all []*ModelGrant

	// 级联层级：按优先级从高到低加载。
	cascadeLevels := [][2]string{
		{PrincipalKey, principal.ID},
		{PrincipalPerson, principal.ID},
		{PrincipalParty, principal.ID},
	}

	for _, level := range cascadeLevels {
		typ, id := level[0], level[1]
		// 收集所有级联层级中匹配该 principal 的规则。
		// 原守卫 "if typ == principal.Type" 导致每次只加载单层级规则，
		// 级联评估完全失效——现已移除，所有层级规则均被收集并统一按 DENY-first 评估。
		var grants []*ModelGrant
		if err := c.DB.Where("principal_type = ? AND principal_id = ?", typ, id).
			Order("priority DESC").Find(&grants).Error; err != nil {
			return nil, fmt.Errorf("modelgrant: 查询授权规则失败 (type=%s): %w", typ, err)
		}
		all = append(all, grants...)
	}

	// 全局默认规则：principal_type 与 principal_id 均为空。
	var globalDefaults []*ModelGrant
	if err := c.DB.Where("(principal_type = '' OR principal_type IS NULL) AND (principal_id = '' OR principal_id IS NULL)").
		Order("priority DESC").Find(&globalDefaults).Error; err != nil {
		return nil, fmt.Errorf("modelgrant: 查询全局默认规则失败: %w", err)
	}
	all = append(all, globalDefaults...)

	return all, nil
}

// findQuotaGrant 查找主体在指定模型上配置了配额的授权规则。
// 若未找到则返回 ErrNoModelGrantFound。
func (c *Checker) findQuotaGrant(principal Principal, modelID string) (*ModelGrant, error) {
	var mg ModelGrant
	err := c.DB.Where("principal_type = ? AND principal_id = ? AND model_id = ?",
		principal.Type, principal.ID, modelID).
		First(&mg).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNoModelGrantFound
		}
		return nil, fmt.Errorf("modelgrant: 查询配额授权规则失败: %w", err)
	}
	return &mg, nil
}
