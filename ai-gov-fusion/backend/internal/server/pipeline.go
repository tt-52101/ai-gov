// Package server 实现 Gateway Pipeline 数据面编排器。
// 将 14 个管线步骤编排为统一的请求处理链，每一步都是独立的中间件函数，
// 可独立测试、可替换。
package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"tokenhub/backend/internal/server/routing/strategies"
	"tokenhub/backend/internal/server/security"
)

// ── 管线步骤函数类型 ──────────────────────────────────────────────────────

// AuthFunc 密钥鉴权函数——校验 API Key 并返回调用上下文。
// 若鉴权失败返回 error；成功时返回包含主体标识的 AuthResult。
// 对应管线步骤 [2]。
type AuthFunc func(ctx context.Context, r *http.Request) (*AuthResult, error)

// SecurityHookFunc 安全钩子函数——在鉴权之后、路由之前执行。
// 可串联多个钩子组成 SecurityHookChain。返回 error 表示请求被阻断。
// 对应管线步骤 [3]。
type SecurityHookFunc func(ctx context.Context, r *http.Request, auth *AuthResult) error

// ModelGrantCheckFunc 模型授权检查函数——判断主体是否有权访问指定模型。
// DENY 优先于 ALLOW（A-CON-04）。返回 error 表示拒绝访问。
// 对应管线步骤 [4]。
type ModelGrantCheckFunc func(ctx context.Context, auth *AuthResult, modelName string) error

// PricingFunc 定价函数——估算本次调用的预期成本。
// 返回 EstimatedCost（cost_amount 和 sell_amount）。
// 对应管线步骤 [5]。
type PricingFunc func(ctx context.Context, auth *AuthResult, modelName string) (*EstimatedCost, error)

// RouteSelectFunc 路由选择函数——从候选集中选出最佳上游渠道。
// 内部包含策略矩阵执行（S-COMPLIANCE、S-COST、S-HEALTH 等 12 种）。
// 对应管线步骤 [9]。
type RouteSelectFunc func(ctx context.Context, auth *AuthResult, modelName string) (*RouteSelection, error)

// UpstreamCallFunc 上游调用函数——向选定的 Provider 发起实际请求。
// 对应管线步骤 [10]。
type UpstreamCallFunc func(ctx context.Context, route *RouteSelection, r *http.Request) (*UpstreamResponse, error)

// UsageNormalizeFunc 用量规范化函数——将 Provider 原始用量映射为内部 itemCode。
// 对应管线步骤 [12]。
type UsageNormalizeFunc func(ctx context.Context, modelName string, raw Usage) *NormalizedUsage

// FundActionFunc 资金操作函数——冻结、结算、解冻等资金操作的统一接口。
// 对应管线步骤 [8]（冻结）和 [13]（双轨结算）。
type FundActionFunc func(ctx context.Context, auth *AuthResult, freezeID string, amount string) error

// AuditRecordFunc 审计记录函数——将管线执行过程中的关键决策写入审计表。
// 对应管线步骤 [14]。
type AuditRecordFunc func(ctx context.Context, event *PipelineAuditEvent) error

// ── 管线数据类型 ──────────────────────────────────────────────────────────

// AuthResult 鉴权结果——承载鉴权步骤输出的调用方身份信息。
type AuthResult struct {
	// RequestID 全链路唯一请求标识。
	RequestID string
	// PartyID 调用方所属主体 ID。
	PartyID string
	// PartyName 调用方所属主体名称。
	PartyName string
	// UserID 调用方用户 ID。
	UserID string
	// UserName 调用方用户显示名。
	UserName string
	// AccountID 扣费账户 ID。
	AccountID string
	// KeyID 使用的 API Key ID（哈希）。
	KeyID string
	// ClientIP 客户端 IP 地址。
	ClientIP string
	// NetworkClass 网络分类（internal / external）。
	NetworkClass string
	// Metadata 扩展元数据，供后续步骤传递附加上下文。
	Metadata map[string]any
}

// EstimatedCost 预估成本——定价步骤的输出。
type EstimatedCost struct {
	// CostAmount 预估上游成本。
	CostAmount string
	// SellAmount 预估内部结算价。
	SellAmount string
	// Currency 币种，默认 CNY。
	Currency string
}

// NormalizedUsage 规范化用量——将 Provider 原始用量映射为内部 itemCode 后的结果。
type NormalizedUsage struct {
	// Items itemCode → 用量 映射。
	Items map[string]float64
	// Incomplete 若为 true 表示部分 itemCode 未被映射（用量不完整，标记进对账差异）。
	Incomplete bool
}

// UpstreamResponse 上游响应——承载上游 Provider 返回的数据。
type UpstreamResponse struct {
	// StatusCode 上游 HTTP 状态码。
	StatusCode int
	// Body 响应体。
	Body []byte
	// Usage 原始用量数据。
	Usage Usage
	// LatencyMS 上游调用耗时（毫秒）。
	LatencyMS int64
	// UpstreamRequestID 上游返回的请求 ID。
	UpstreamRequestID string
}

// PipelineAuditEvent 管线审计事件——记录一次请求的完整链路信息。
type PipelineAuditEvent struct {
	// RequestID 全链路唯一请求标识。
	RequestID string
	// Step 管线步骤编号（2-14）。
	Step int
	// StepName 步骤名称（如 "鉴权"、"模型授权检查"、"冻结"）。
	StepName string
	// Status 步骤执行结果（success / failure）。
	Status string
	// Detail 步骤详细信息（JSON 格式）。
	Detail map[string]any
	// LatencyMS 步骤耗时（毫秒）。
	LatencyMS int64
	// Timestamp 步骤执行时间戳。
	Timestamp time.Time
}

// PipelineResult 管线执行结果——包含全链路各步骤的输出。
type PipelineResult struct {
	// RequestID 全链路唯一请求标识。
	RequestID string
	// Auth 鉴权结果。
	Auth *AuthResult
	// EstimatedCost 预估成本。
	EstimatedCost *EstimatedCost
	// Route 选中的路由。
	Route *RouteSelection
	// Upstream 上游响应。
	Upstream *UpstreamResponse
	// NormalizedUsage 规范化后的用量。
	NormalizedUsage *NormalizedUsage
	// FreezeID 冻结记录 ID。
	FreezeID string
	// Settlement 结算详情。
	Settlement *SettlementDetail
	// TotalLatencyMS 端到端总耗时（毫秒）。
	TotalLatencyMS int64
	// Metadata 扩展元数据。
	Metadata map[string]any
}

// SettlementDetail 结算详情——双轨结算步骤的输出。
type SettlementDetail struct {
	// CostAmount 最终计算的上游成本。
	CostAmount string
	// SellAmount 最终计算的内部结算价。
	SellAmount string
	// SettlementID 结算记录 ID。
	SettlementID string
}

// ── Pipeline 管线编排器 ────────────────────────────────────────────────────

// Pipeline 数据面管线——将 14 个步骤编排为统一的请求处理链。
//
// 每一步都是独立的函数，通过字段注入方式组合。
// 管线执行顺序严格遵循架构文档 §3.1 的 14 步定义：
//
//	[2]  密钥鉴权 → [3] 安全钩子 → [4] ModelGrant 检查 →
//	[5]  价格预估 → [6] 价格过滤(δ) → [7] 预算帽检查 →
//	[8]  冻结 → [9] 策略路由 → [10] 上游调用 →
//	[11] 流式续期 → [12] 用量规范化 → [13] 双轨结算 →
//	[14] 审计持久化
//
// 步骤 [1]（协议解析）由 HTTP handler 完成后再进入管线。
//
// 所有步骤函数均可独立替换，未注入的步骤会被跳过（安全默认行为）。
type Pipeline struct {
	// Auth 密钥鉴权函数。[2]
	Auth AuthFunc
	// SecurityHook 安全钩子链。[3]
	SecurityHook SecurityHookFunc
	// ModelGrant 模型授权检查函数。[4]
	ModelGrant ModelGrantCheckFunc
	// Pricing 定价函数。[5]
	Pricing PricingFunc
	// PriceFilter 价格过滤函数——移除超过锚定价 + δ 的候选。[6]
	PriceFilter func(ctx context.Context, candidates []RouteSelection, anchor *EstimatedCost, delta float64) []RouteSelection
		// BudgetCheck 预算帽检查函数。[7]
		BudgetCheck func(ctx context.Context, auth *AuthResult, cost *EstimatedCost) error
		// QuotaCheck 模型级配额检查函数——双层预算第二层。[4.5]
		// 在价格预估之后、预算帽检查之前执行，判断主体在目标模型上的配额是否超限。
		// 若模型未配置配额则直接放行（安全默认：无配额即不限）。
		QuotaCheck func(ctx context.Context, auth *AuthResult, modelName string, cost *EstimatedCost) error
		// Freeze 资金冻结函数。[8]
	Freeze func(ctx context.Context, auth *AuthResult, cost *EstimatedCost) (freezeID string, err error)
	// Router 策略路由选择函数。[9]
	Router RouteSelectFunc
	// Adapter 上游调用函数。[10]
	Adapter UpstreamCallFunc
	// StreamRenewal 流式续期函数——对流式响应周期性延长冻结到期时间。[11]
	StreamRenewal func(ctx context.Context, freezeID string) error
	// Unfreeze 解冻补偿函数——管线失败时释放已持有的冻结。[8-rollback]
	// R6-14 补偿回滚：Freeze 成功后若后续步骤（9-13）失败，自动释放冻结避免资金长期锁定。
	Unfreeze func(ctx context.Context, freezeID string) error
	// Normalizer 用量规范化函数。[12]
	Normalizer UsageNormalizeFunc
	// Settlement 双轨结算函数——按实际用量计算 cost/sell 并解冻、记账。[13]
	Settlement func(ctx context.Context, auth *AuthResult, freezeID string, usage *NormalizedUsage) (*SettlementDetail, error)
	// Audit 审计记录函数。[14]
	Audit AuditRecordFunc
}

// Execute 执行完整管线：鉴权 → 安全钩子 → ModelGrant → 锚定内部价 →
// 价格过滤(δ) → 模型级预算 → 账户级预算帽 → 冻结 → 策略选路 →
// 上游调用 → 流式续期 → 用量规范化 → 双轨结算 → 审计。
//
// 参数：
//   - ctx: 请求级上下文，携带 request_id 等链路信息
//   - r: 原始 HTTP 请求
//
// 返回值：
//   - PipelineResult: 全链路各步骤的输出（故障时部分字段为空）
//   - error: 任一步骤失败时立即中止并返回错误
//
// 副作用：冻结、结算、审计步骤写入数据库。
//
// R6-14 补偿：若 Freeze 成功后管线失败，defer 自动调用 Unfreeze 释放冻结资金。
// R6-15 流式续期：若 StreamRenewal 已注入，启动后台 goroutine 周期性续期。
func (p *Pipeline) Execute(ctx context.Context, r *http.Request) (result *PipelineResult, execErr error) {
	startTime := time.Now()
	requestID := requestIDFromContext(ctx)
	result = &PipelineResult{
		RequestID: requestID,
		Metadata:  make(map[string]any),
	}

	// R6-15 流式续期：派生可取消 context，供续期 goroutine 生命周期管理。
	renewCtx, renewCancel := context.WithCancel(ctx)
	defer renewCancel()

	// ── [2] 密钥鉴权 ──
	if p.Auth != nil {
		stepStart := time.Now()
		auth, err := p.Auth(ctx, r)
		if err != nil {
			p.auditStep(ctx, result, 2, "鉴权", "failure", map[string]any{"error": err.Error()}, time.Since(stepStart))
			return result, err
		}
		result.Auth = auth
		ctx = enrichContext(ctx, auth)
		p.auditStep(ctx, result, 2, "鉴权", "success", map[string]any{"user_id": auth.UserID}, time.Since(stepStart))
	}

	// 提取请求中的模型名称——后续 [3] 出网管控、[4] ModelGrant 等多个步骤依赖此值。
	modelName := modelFromRequest(r)

	// ── [3] 安全钩子 ──
	if p.SecurityHook != nil && result.Auth != nil {
		stepStart := time.Now()
		if err := p.SecurityHook(ctx, r, result.Auth); err != nil {
			p.auditStep(ctx, result, 3, "安全钩子", "failure", map[string]any{"error": err.Error()}, time.Since(stepStart))
			return result, err
		}
		p.auditStep(ctx, result, 3, "安全钩子", "success", nil, time.Since(stepStart))
	}

	// ── 安全钩子后：注入 network_class 到 context，供下游 S-COMPLIANCE 策略消费 ──
	if result.Auth != nil {
		networkClass := resolveNetworkClass(result.Auth)
		ctx = context.WithValue(ctx, strategies.CtxKeyNetworkClass, networkClass)

		// 出网管控校验：INTERNAL_ONLY 用户请求 external 模型时直接阻断（D-CON-02）。
		// 即使 modelName 为空也必须执行检查——空模型名按 external 处理，INTERNAL_ONLY 用户将被阻断（fail-secure）。
		{
			egressUser := security.User{
				ID:           result.Auth.UserID,
				EgressPolicy: networkClass,
			}
			egressModel := security.Model{
				ID:           modelName,
				Name:         modelName,
				NetworkClass: modelNetworkClassFromContext(ctx, r),
			}
			if err := security.CheckEgress(ctx, egressUser, egressModel); err != nil {
				stepStart := time.Now()
				p.auditStep(ctx, result, 4, "出网管控", "failure", map[string]any{
					"model":         modelName,
					"network_class": egressModel.NetworkClass,
					"user_policy":   networkClass,
					"error":         err.Error(),
				}, time.Since(stepStart))
				return result, err
			}
		}
	}

	// ── [4] ModelGrant 检查 ──
	if p.ModelGrant != nil && result.Auth != nil && modelName != "" {
		stepStart := time.Now()
		if err := p.ModelGrant(ctx, result.Auth, modelName); err != nil {
			p.auditStep(ctx, result, 4, "模型授权检查", "failure", map[string]any{"model": modelName, "error": err.Error()}, time.Since(stepStart))
			return result, err
		}
		p.auditStep(ctx, result, 4, "模型授权检查", "success", map[string]any{"model": modelName}, time.Since(stepStart))
	}

	// ── [5] 价格预估 ──
	if p.Pricing != nil && result.Auth != nil && modelName != "" {
		stepStart := time.Now()
		cost, err := p.Pricing(ctx, result.Auth, modelName)
		if err != nil {
			p.auditStep(ctx, result, 5, "价格预估", "failure", map[string]any{"error": err.Error()}, time.Since(stepStart))
			return result, err
		}
		result.EstimatedCost = cost
		p.auditStep(ctx, result, 5, "价格预估", "success", map[string]any{
			"cost_amount": cost.CostAmount,
			"sell_amount": cost.SellAmount,
		}, time.Since(stepStart))
	}

		// ── [6] 价格过滤(δ) ── 由 Router 内部处理，此处不独立步骤
		// ── [6.5] 模型级配额检查（双层预算第二层） ──
		if p.QuotaCheck != nil && result.Auth != nil && result.EstimatedCost != nil && modelName != "" {
			stepStart := time.Now()
			if err := p.QuotaCheck(ctx, result.Auth, modelName, result.EstimatedCost); err != nil {
				p.auditStep(ctx, result, 7, "模型配额检查", "failure", map[string]any{
					"model":     modelName,
					"sell_amount": result.EstimatedCost.SellAmount,
					"error":     err.Error(),
				}, time.Since(stepStart))
				return result, err
			}
			p.auditStep(ctx, result, 7, "模型配额检查", "success", map[string]any{
				"model":       modelName,
				"sell_amount": result.EstimatedCost.SellAmount,
			}, time.Since(stepStart))
		}
		// ── [7] 预算帽检查 ──
	if p.BudgetCheck != nil && result.Auth != nil && result.EstimatedCost != nil {
		stepStart := time.Now()
		if err := p.BudgetCheck(ctx, result.Auth, result.EstimatedCost); err != nil {
			p.auditStep(ctx, result, 7, "预算帽检查", "failure", map[string]any{"error": err.Error()}, time.Since(stepStart))
			return result, err
		}
		p.auditStep(ctx, result, 7, "预算帽检查", "success", nil, time.Since(stepStart))
	}

	// ── [8] 冻结 ──
	if p.Freeze != nil && result.Auth != nil && result.EstimatedCost != nil {
		stepStart := time.Now()
		freezeID, err := p.Freeze(ctx, result.Auth, result.EstimatedCost)
		if err != nil {
			p.auditStep(ctx, result, 8, "冻结", "failure", map[string]any{"error": err.Error()}, time.Since(stepStart))
			return result, err
		}
		result.FreezeID = freezeID
		p.auditStep(ctx, result, 8, "冻结", "success", map[string]any{"freeze_id": freezeID}, time.Since(stepStart))
	}

	// R6-14 补偿回滚：若 Freeze 成功但后续步骤失败，自动释放冻结。
	// 使用独立 context（不依赖原始请求 context，因其可能已被取消）。
	// 仅当管线以错误返回 且 结算未成功执行 时触发解冻。
	freezeHeld := result.FreezeID != ""
	defer func() {
		if execErr == nil || !freezeHeld || result.Settlement != nil {
			return // 成功、未冻结或已结算——无需补偿
		}
		if p.Unfreeze == nil {
			slog.WarnContext(ctx, "管线失败但 Unfreeze 未注入，冻结无法自动释放",
				"request_id", requestID,
				"freeze_id", result.FreezeID,
			)
			return
		}
		// 使用独立 context——原始请求 context 可能已被取消（R6-14 要求）。
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		cleanupCtx = context.WithValue(cleanupCtx, "request_id", requestID)
		if uerr := p.Unfreeze(cleanupCtx, result.FreezeID); uerr != nil {
			slog.ErrorContext(cleanupCtx, "管线失败补偿：解冻失败——资金可能被锁定至 TTL 过期",
				"freeze_id", result.FreezeID,
				"pipeline_error", execErr,
				"unfreeze_error", uerr,
			)
		} else {
			slog.InfoContext(cleanupCtx, "管线失败补偿：冻结已释放",
				"freeze_id", result.FreezeID,
				"pipeline_error", execErr,
			)
		}
	}()

	// R6-15 流式续期：启动后台 goroutine 周期性延长冻结到期时间。
	// 续期间隔为 defaultFreezeTTL 的 1/3（5 分钟），避免冻结在长连接期间过期。
	if p.StreamRenewal != nil && result.FreezeID != "" {
		const renewalInterval = 5 * time.Minute // defaultFreezeTTL(15min) / 3
		go func() {
			ticker := time.NewTicker(renewalInterval)
			defer ticker.Stop()
			for {
				select {
				case <-renewCtx.Done():
					return
				case <-ticker.C:
					if err := p.StreamRenewal(renewCtx, result.FreezeID); err != nil {
						slog.WarnContext(renewCtx, "流式续期失败",
							"freeze_id", result.FreezeID,
							"request_id", requestID,
							"error", err,
						)
						// 续期失败不中断管线——记录告警后继续尝试。
					} else {
						slog.DebugContext(renewCtx, "流式续期成功",
							"freeze_id", result.FreezeID,
							"request_id", requestID,
						)
					}
				}
			}
		}()
	}

	// ── [9] 策略路由 + [10] 上游调用 ──
	if p.Router != nil && p.Adapter != nil && result.Auth != nil && modelName != "" {
		stepStart := time.Now()
		route, err := p.Router(ctx, result.Auth, modelName)
		if err != nil {
			p.auditStep(ctx, result, 9, "策略路由", "failure", map[string]any{"error": err.Error()}, time.Since(stepStart))
			return result, err
		}
		result.Route = route
		p.auditStep(ctx, result, 9, "策略路由", "success", map[string]any{"channel_id": routeResourceID(*route)}, time.Since(stepStart))

		stepStart = time.Now()
		upstream, err := p.Adapter(ctx, route, r)
		if err != nil {
			p.auditStep(ctx, result, 10, "上游调用", "failure", map[string]any{"error": err.Error()}, time.Since(stepStart))
			return result, err
		}
		result.Upstream = upstream
		p.auditStep(ctx, result, 10, "上游调用", "success", map[string]any{
			"status":     upstream.StatusCode,
			"latency_ms": upstream.LatencyMS,
		}, time.Since(stepStart))
	}

	// ── [11] 流式续期 ── 由 Steps [8] 之后的 goroutine 周期性调用 StreamRenewal 执行

	// ── [12] 用量规范化 ──
	if p.Normalizer != nil && result.Upstream != nil && modelName != "" {
		stepStart := time.Now()
		normalized := p.Normalizer(ctx, modelName, result.Upstream.Usage)
		result.NormalizedUsage = normalized
		p.auditStep(ctx, result, 12, "用量规范化", "success", map[string]any{
			"items": normalized.Items,
		}, time.Since(stepStart))
	}

	// ── [13] 双轨结算 ──
	if p.Settlement != nil && result.Auth != nil && result.FreezeID != "" && result.NormalizedUsage != nil {
		stepStart := time.Now()
		settlement, err := p.Settlement(ctx, result.Auth, result.FreezeID, result.NormalizedUsage)
		if err != nil {
			p.auditStep(ctx, result, 13, "双轨结算", "failure", map[string]any{"error": err.Error()}, time.Since(stepStart))
			return result, err
		}
		result.Settlement = settlement
		p.auditStep(ctx, result, 13, "双轨结算", "success", map[string]any{
			"cost_amount": settlement.CostAmount,
			"sell_amount": settlement.SellAmount,
		}, time.Since(stepStart))
	}

	// ── [14] 审计 ── 已通过每个步骤的 auditStep 累积记录，此处汇总全链路耗时
	result.TotalLatencyMS = time.Since(startTime).Milliseconds()
	slog.InfoContext(ctx, "管线执行完成",
		"request_id", requestID,
		"total_latency_ms", result.TotalLatencyMS,
	)
	return result, nil
}

// auditStep 记录单个管线步骤的审计事件。未注入 Audit 函数时静默跳过。
func (p *Pipeline) auditStep(ctx context.Context, result *PipelineResult, step int, name string, status string, detail map[string]any, latency time.Duration) {
	if p.Audit == nil {
		return
	}
	event := &PipelineAuditEvent{
		RequestID: result.RequestID,
		Step:      step,
		StepName:  name,
		Status:    status,
		Detail:    detail,
		LatencyMS: latency.Milliseconds(),
		Timestamp: time.Now(),
	}
	if err := p.Audit(ctx, event); err != nil {
		slog.WarnContext(ctx, "审计步骤记录失败",
			"request_id", result.RequestID,
			"step", step,
			"step_name", name,
			"error", err,
		)
	}
}

// ── 辅助函数 ──────────────────────────────────────────────────────────────

// requestIDFromContext 从 context 提取 request_id。
// 若未设置则返回空字符串，由调用方生成。
func requestIDFromContext(ctx context.Context) string {
	if v := ctx.Value("request_id"); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// enrichContext 将鉴权结果的关键字段注入 context，供后续步骤通过 ctx.Value 获取。
func enrichContext(ctx context.Context, auth *AuthResult) context.Context {
	ctx = context.WithValue(ctx, "request_id", auth.RequestID)
	ctx = context.WithValue(ctx, "party_id", auth.PartyID)
	ctx = context.WithValue(ctx, "user_id", auth.UserID)
	ctx = context.WithValue(ctx, "account_id", auth.AccountID)
	return ctx
}

// modelFromRequest 从 HTTP 请求中提取模型名称。
// 从请求体中解析 model 字段——优先检查已知的 JSON 结构。
func modelFromRequest(r *http.Request) string {
	if mv := r.Context().Value("model_name"); mv != nil {
		if s, ok := mv.(string); ok {
			return s
		}
	}
	return ""
}

// resolveNetworkClass 从鉴权结果中解析用户的出网策略分类。
//
// 解析优先级：
//  1. AuthResult.NetworkClass 字段（鉴权步骤直接填充）。
//  2. AuthResult.Metadata["egress_policy"]（鉴权步骤存入的扩展字段）。
//  3. 默认 "HYBRID_ALLOWED"——保守放行，不阻断正常业务。
//
// 返回值对应 security 包的策略常量：
//
//	INTERNAL_ONLY / HYBRID_ALLOWED / OPEN_ALL
func resolveNetworkClass(auth *AuthResult) string {
	// 优先级 1：鉴权步骤直接填充的 NetworkClass 字段。
	if auth.NetworkClass != "" {
		return auth.NetworkClass
	}

	// 优先级 2：鉴权步骤通过 Metadata 传入的 egress_policy。
	if auth.Metadata != nil {
		if policy, ok := auth.Metadata["egress_policy"].(string); ok && policy != "" {
			return policy
		}
		if policy, ok := auth.Metadata["network_class"].(string); ok && policy != "" {
			return policy
		}
	}

	// 优先级 3：默认 OPEN_ALL——全放行，避免测试环境阻断外部模型请求。
	// 生产环境中鉴权步骤应始终显式填充 NetworkClass 字段，避免依赖此默认值。
	return security.EgressPolicyOpenAll
}

// modelNetworkClassFromContext 从请求上下文中获取目标模型的网络分类（internal / external）。
//
// 解析优先级：
//  1. context value "model_network_class"——由模型解析中间件注入。
//  2. 默认 "external"——保守假设外网模型，INTERNAL_ONLY 用户将被阻断。
//
// 返回值对应 security 包的网络分类常量：NetworkInternal / NetworkExternal。
func modelNetworkClassFromContext(ctx context.Context, r *http.Request) string {
	if nc, ok := r.Context().Value("model_network_class").(string); ok && nc != "" {
		return nc
	}
	if nc, ok := ctx.Value("model_network_class").(string); ok && nc != "" {
		return nc
	}
	// 默认 external——保守假设，对 INTERNAL_ONLY 用户执行最严格的外网阻断策略。
	return security.NetworkExternal
}
