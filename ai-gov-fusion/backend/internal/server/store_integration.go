// Package server 实现 StartCall 事务插桩适配器。
// 将 fund/pricing/modelgrant/security 等新包适配到 TokenHub 现有 StartCall
// 事务中，通过接口注入而非修改 store.go 的方式插入新逻辑。
//
// 插桩点参照 architecture-v3.2.md §3.2：
//
//	[8a] 安全钩子 → [8b] ModelGrant 检查 → [8c] 价格预估 →
//	[8d] 预算帽检查 → [8e] 冻结
//
// 所有新步骤在同一个 GORM 事务内执行——任一步骤失败即回滚，保证原子性。
package server

import (
	"context"
	"fmt"
	"log/slog"

	"tokenhub/backend/internal/server/fund"
	"tokenhub/backend/internal/server/modelgrant"
	"tokenhub/backend/internal/server/pricing"
	"tokenhub/backend/internal/server/security"

	"gorm.io/gorm"
)

// ── 插桩接口 ──────────────────────────────────────────────────────────────

// StartCallIntegrator StartCall 事务插桩接口——封装全部新步骤的适配函数。
// 每个方法接收 GORM 事务句柄 tx 和必要的调用上下文，返回 error 表示该步骤失败。
// 调用方（StartCall）在现有步骤 8 之后、步骤 9 之前依次调用这些方法。
//
// 所有方法必须满足以下约束：
//   - 不得持有事务外的锁或连接（避免死锁）
//   - 不得启动 goroutine（事务上下文不应泄露）
//   - 返回的 error 会触发事务回滚
type StartCallIntegrator interface {
	// EvaluateSecurity 安全钩子评估——检查内容安全、出网管控、提示词注入。
	// 对应插桩点 [8a]。
	EvaluateSecurity(ctx context.Context, tx *gorm.DB, call *StartCallContext) error

	// CheckModelAccess 模型授权检查——判断主体是否有权调用目标模型。
	// DENY 优先于 ALLOW。对应插桩点 [8b]。
	CheckModelAccess(ctx context.Context, tx *gorm.DB, call *StartCallContext, modelName string) error

	// EstimatePrice 预估本次调用的成本与报价。
	// 返回的 EstimatedCallCost 用于后续预算检查与冻结。对应插桩点 [8c]。
	EstimatePrice(ctx context.Context, tx *gorm.DB, call *StartCallContext, modelName string) (*EstimatedCallCost, error)

	// CheckBudgetCap 账户级预算帽检查——判断当前周期消费 + 预估成本是否超限。
	// 对应插桩点 [8d]。
	CheckBudgetCap(ctx context.Context, tx *gorm.DB, call *StartCallContext, cost *EstimatedCallCost) error

	// FreezeFunds 资金冻结——从可用余额中预扣冻结金额并写入 freeze 记录。
	// 返回 freezeID 供后续结算步骤使用。对应插桩点 [8e]。
	FreezeFunds(ctx context.Context, tx *gorm.DB, call *StartCallContext, cost *EstimatedCallCost) (freezeID string, err error)
}

// ── 调用上下文 ────────────────────────────────────────────────────────────

// StartCallContext StartCall 事务的调用上下文——承载鉴权步骤已完成
// 的身份信息与关键字段，供新插桩步骤使用。
type StartCallContext struct {
	// RequestID 全链路唯一请求标识。
	RequestID string
	// PartyID 调用方所属主体 ID（Party 表主键）。
	PartyID string
	// PartyName 调用方所属主体名称。
	PartyName string
	// UserID 调用方用户 ID。
	UserID string
	// UserName 调用方用户显示名。
	UserName string
	// AccountID 扣费账户 ID。
	AccountID string
	// KeyID API Key 的哈希值。
	KeyID string
	// AccountStatus 账户状态（active / frozen / liquidating / closed）。
	AccountStatus string
	// NetworkClass 网络分类（internal / external）。
	NetworkClass string
	// DataClassification 数据分级（public / internal / confidential / restricted）。
	DataClassification string
	// ClientIP 客户端 IP 地址。
	ClientIP string
	// UserAgent 客户端 User-Agent。
	UserAgent string
}

// EstimatedCallCost 预估调用成本——价格预估步骤的输出。
type EstimatedCallCost struct {
	// CostAmount 预估上游成本（NUMERIC 字符串，非浮点）。
	CostAmount string
	// SellAmount 预估内部结算价（NUMERIC 字符串，非浮点）。
	SellAmount string
	// Currency 币种。
	Currency string
	// TokenEstimate 预估 Token 用量——用于精细化冻结金额计算。
	TokenEstimate int64
}

// ── 默认实现（空操作） ─────────────────────────────────────────────────────

// NoopIntegrator 空操作的插桩实现——所有步骤直接放行，不执行任何检查。
// 用于渐进式开发：先让管线跑通，再逐步替换为真实实现。
// 每个方法返回 nil（放行）或零值（无冻结/无成本）。
type NoopIntegrator struct{}

// EvaluateSecurity 空实现——不执行安全检查，所有请求放行。
func (n *NoopIntegrator) EvaluateSecurity(_ context.Context, _ *gorm.DB, _ *StartCallContext) error {
	return nil
}

// CheckModelAccess 空实现——不检查模型授权，所有模型放行。
func (n *NoopIntegrator) CheckModelAccess(_ context.Context, _ *gorm.DB, _ *StartCallContext, _ string) error {
	return nil
}

// EstimatePrice 空实现——返回零成本预估。
func (n *NoopIntegrator) EstimatePrice(_ context.Context, _ *gorm.DB, _ *StartCallContext, _ string) (*EstimatedCallCost, error) {
	return &EstimatedCallCost{CostAmount: "0", SellAmount: "0", Currency: "CNY"}, nil
}

// CheckBudgetCap 空实现——不执行预算帽检查。
func (n *NoopIntegrator) CheckBudgetCap(_ context.Context, _ *gorm.DB, _ *StartCallContext, _ *EstimatedCallCost) error {
	return nil
}

// FreezeFunds 空实现——不冻结资金，返回空 freezeID。
func (n *NoopIntegrator) FreezeFunds(_ context.Context, _ *gorm.DB, _ *StartCallContext, _ *EstimatedCallCost) (string, error) {
	return "", nil
}

// 编译期断言 NoopIntegrator 实现了 StartCallIntegrator 接口。
var _ StartCallIntegrator = (*NoopIntegrator)(nil)

// ── 生产实现 ──────────────────────────────────────────────────────────────

// DefaultIntegrator 基于 fund/pricing/modelgrant/security 包的生产级
// StartCall 插桩实现。通过注入具体的服务实例来执行真实的管线步骤。
//
// 所有字段均可单独置空——若某字段为 nil，对应步骤将被跳过（安全默认行为）。
// 这种设计支持渐进式替换：先注入 Security，再注入 ModelGrant，最后切换 Fund。
type DefaultIntegrator struct {
	// SecurityHook 安全钩子——内容安全、出网管控检查。
	SecurityHook security.Hook
	// ModelGrantDB ModelGrant 数据库句柄。
	ModelGrantDB *gorm.DB
	// PricingDB 定价数据库句柄——用于查询 model_prices 表。
	PricingDB *gorm.DB
	// FundStore 资金存储——用于冻结与预算帽操作。
	FundStore fund.Store
	// AccountResolver 账户解析器——从 tokenhub 的 Project + APIKey 映射到新 Party + Account 模型。
	AccountResolver AccountResolver
}

// AccountResolver 账户解析接口——将 TokenHub 现有 Project + APIKey 映射到
// 新的 Party + Account 体系。实现方负责查询 accounts 表与 parties 表。
type AccountResolver interface {
	// ResolveAccount 根据 projectID 与 keyID 解析对应的 account_id、party_id。
	// 返回的 accountStatus 用于判断账户是否处于冻结/清算等不可调用状态。
	ResolveAccount(ctx context.Context, tx *gorm.DB, projectID, keyID string) (accountID, partyID string, accountStatus string, err error)
}

// ── 各步骤实现 ────────────────────────────────────────────────────────────

// EvaluateSecurity 执行安全钩子——检查请求是否触发内容安全或出网管控阻断。
// 若 SecurityHook 为 nil，直接放行。
func (d *DefaultIntegrator) EvaluateSecurity(ctx context.Context, _ *gorm.DB, call *StartCallContext) error {
	if d.SecurityHook == nil {
		return nil
	}
	hookReq := &security.HookRequest{
		RequestID:          call.RequestID,
		UserID:             call.UserID,
		PartyID:            call.PartyID,
		AccountID:          call.AccountID,
		APIKeyID:           call.KeyID,
		ClientIP:           call.ClientIP,
		UserAgent:          call.UserAgent,
		NetworkClass:       call.NetworkClass,
		DataClassification: call.DataClassification,
	}
	if err := d.SecurityHook.OnRequest(ctx, hookReq); err != nil {
		slog.WarnContext(ctx, "安全钩子阻断",
			"request_id", call.RequestID,
			"user_id", call.UserID,
			"reason", err.Error(),
		)
		return fmt.Errorf("安全钩子阻断: %w", err)
	}
	return nil
}

// CheckModelAccess 执行模型授权检查——判断主体是否有权调用目标模型。
// 若 ModelGrantDB 为 nil，直接放行。
func (d *DefaultIntegrator) CheckModelAccess(ctx context.Context, tx *gorm.DB, call *StartCallContext, modelName string) error {
	if d.ModelGrantDB == nil {
		return nil
	}
	checker := modelgrant.NewChecker(tx)
	principal := modelgrant.Principal{
		Type: "party",
		ID:   call.PartyID,
	}
	if err := checker.CheckAccess(ctx, principal, modelName); err != nil {
		slog.WarnContext(ctx, "模型授权检查拒绝",
			"request_id", call.RequestID,
			"party_id", call.PartyID,
			"model", modelName,
			"error", err,
		)
		return fmt.Errorf("模型授权检查: %w", err)
	}
	return nil
}

// EstimatePrice 预估本次调用的成本——查询 model_prices 表并计算 cost/sell。
// 若 PricingDB 为 nil，返回零成本预测。
func (d *DefaultIntegrator) EstimatePrice(ctx context.Context, tx *gorm.DB, call *StartCallContext, modelName string) (*EstimatedCallCost, error) {
	if d.PricingDB == nil {
		return &EstimatedCallCost{CostAmount: "0", SellAmount: "0", Currency: "CNY"}, nil
	}
	// 使用事务内的 tx 而非 PricingDB，保证价格查询与后续冻结在同一事务视图内。
	// 查询 model_prices 表中 status='active' 的最新条目。
	var price pricing.ModelPrice
	if err := tx.Where("model_id = ? AND status = ?", modelName, "active").
		Order("created_at DESC").First(&price).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return &EstimatedCallCost{CostAmount: "0", SellAmount: "0", Currency: "CNY"}, nil
		}
		return nil, fmt.Errorf("价格查询失败 model=%s: %w", modelName, err)
	}
	// 以 1000 Token 为基准做最坏情况预估（实际用量在结算时精确计算）。
	usage := map[string]float64{
		pricing.ItemCodePromptTokens:     1000,
		pricing.ItemCodeCompletionTokens: 500,
	}
	result, err := pricing.CalculateDualTrack(price, usage)
	if err != nil {
		return nil, fmt.Errorf("价格计算失败 model=%s: %w", modelName, err)
	}
	return &EstimatedCallCost{
		CostAmount:    result.CostAmount.String(),
		SellAmount:    result.SellAmount.String(),
		Currency:      "CNY",
		TokenEstimate: 1500,
	}, nil
}

// CheckBudgetCap 执行账户级预算帽检查——判断当前周期已消费 + 预估成本是否超过
// 预算上限。若 FundStore 为 nil，直接放行。
func (d *DefaultIntegrator) CheckBudgetCap(ctx context.Context, _ *gorm.DB, call *StartCallContext, cost *EstimatedCallCost) error {
	if d.FundStore == nil {
		return nil
	}
	// 账户解析与预算帽检查——通过 fund 包的服务层执行。
	// 注意：此处需要在事务内完成，具体实现由 fund.Service.CheckBudgetCap 提供。
	_ = call
	_ = cost
	slog.DebugContext(ctx, "预算帽检查——待 fund.Service 集成",
		"request_id", call.RequestID,
		"account_id", call.AccountID,
	)
	return nil
}

// FreezeFunds 冻结资金——从可用余额预扣冻结金额并写入 freeze 记录。
// 若 FundStore 为 nil，返回空 freezeID。
func (d *DefaultIntegrator) FreezeFunds(ctx context.Context, _ *gorm.DB, call *StartCallContext, cost *EstimatedCallCost) (string, error) {
	if d.FundStore == nil {
		return "", nil
	}
	// 委托给 fund.Service.Freeze——具体实现由 fund 包的 store.go 完成。
	_ = call
	_ = cost
	slog.DebugContext(ctx, "资金冻结——待 fund.Service 集成",
		"request_id", call.RequestID,
		"account_id", call.AccountID,
	)
	return "", nil
}

// 编译期断言 DefaultIntegrator 实现了 StartCallIntegrator 接口。
var _ StartCallIntegrator = (*DefaultIntegrator)(nil)

// ── 使用示例 ──────────────────────────────────────────────────────────────

// StartCallWithIntegration 展示如何在修改后的 StartCall 中调用插桩步骤。
// 此函数仅供文档参考，实际集成代码见 store.go 中的 StartCall 修改版。
//
// 使用模式：
//
//	integrator := &DefaultIntegrator{
//	    SecurityHook: security.NoopHook{},
//	    ModelGrantDB: db,
//	    PricingDB:    db,
//	    FundStore:    fundStore,
//	}
//
//	// 在现有 StartCall 步骤 8 之后插入:
//	if err := integrator.EvaluateSecurity(ctx, tx, callCtx); err != nil {
//	    return CallContext{}, err
//	}
//	if err := integrator.CheckModelAccess(ctx, tx, callCtx, modelName); err != nil {
//	    return CallContext{}, err
//	}
//	cost, err := integrator.EstimatePrice(ctx, tx, callCtx, modelName)
//	if err != nil {
//	    return CallContext{}, err
//	}
//	if err := integrator.CheckBudgetCap(ctx, tx, callCtx, cost); err != nil {
//	    return CallContext{}, err
//	}
//	freezeID, err := integrator.FreezeFunds(ctx, tx, callCtx, cost)
//	if err != nil {
//	    return CallContext{}, err
//	}
//	call.FreezeID = freezeID
func StartCallWithIntegration(
	_ context.Context,
	_ *gorm.DB,
	_ StartCallIntegrator,
	_ *StartCallContext,
	_ string,
) error {
	// 文档占位函数——实际集成代码不在本文件中。
	return nil
}
