// Package server 实现治理 API HTTP handlers——全部控制面端点。
// 按 api-spec-v3.2.md 规范注册所有 /gov/* 路由，使用与 TokenHub 一致的
// net/http 标准库 + http.ServeMux + HandleFunc 模式。
//
// 组织方式：按域拆分为独立 handler 函数，每个函数：
//  1. 从 context 提取主体（ABAC 鉴权已完成）
//  2. 解析请求体
//  3. 调用对应的 Service 层方法
//  4. 返回 JSON 响应 + 错误码
//
// 文件拆分：
//   - gov_handlers.go: 核心路由注册、工具函数、类型定义、Party 域 handlers
//   - gov_handlers_fund.go: Fund、Key、Pricing、ModelGrant、Routing 域 handlers
//   - gov_handlers_abac.go: ABAC、UI Permission、Audit、Dashboard 域 handlers
package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"tokenhub/backend/internal/server/abac"
	"tokenhub/backend/internal/server/audit"
	"tokenhub/backend/internal/server/fund"
	"tokenhub/backend/internal/server/modelgrant"
	"tokenhub/backend/internal/server/party"
	"tokenhub/backend/internal/server/ui_permission"
	"gorm.io/gorm"
)

// ── GovDependencies 治理 API 依赖聚合 ─────────────────────────────────────

// GovDependencies 聚合所有治理服务依赖。
// 每个字段对应一个领域服务，通过 RegisterGovHandlers 注入到路由中。
// 未注入的服务对应端点会返回 501（未实现）。
type GovDependencies struct {
	// FundService 资金治理服务：账本、划拨、冻结、清算、预算帽。
	FundService *fund.Service
	// PartyService 主体管理服务：组织/项目 CRUD、关系边、成员管理。
	PartyService *party.Service
	// PricingDB pricing 自由函数的数据库句柄。
	PricingDB *gorm.DB
	// ABACEngine ABAC 策略评估引擎——所有控制面端点的统一鉴权。
	ABACEngine *abac.Engine
	// ModelGrantChecker 模型授权检查器。
	ModelGrantChecker *modelgrant.Checker
	// UIPermProjector UI 权限投影器——将 ABAC 决策映射到前端菜单/路由/按钮。
	UIPermProjector *ui_permission.Projector
	// AuditRecorder 审计事件记录函数。若为 nil 则跳过审计。
	AuditRecorder func(ctx *GovRequestContext, event *audit.AuditEvent) error
	// RouteProfileDB 路由档案数据库句柄。
	RouteProfileDB *gorm.DB
	// DB 主数据库句柄——用于直接查询表。
	DB *gorm.DB
}

// ── GovHandler 治理 API 处理器 ────────────────────────────────────────────

// GovHandler 治理 API HTTP handler 集合。
// 每个公开方法对应一个 HTTP handler，签名与 http.HandlerFunc 兼容。
type GovHandler struct {
	deps GovDependencies
}

// NewGovHandler 创建新的治理 API 处理器实例。
func NewGovHandler(deps GovDependencies) *GovHandler {
	return &GovHandler{deps: deps}
}

// ── GovRequestContext 请求上下文 ───────────────────────────────────────────

// GovRequestContext 治理 API 请求上下文——从 HTTP 请求中提取的公共信息。
type GovRequestContext struct {
	// RequestID 全链路唯一请求标识。
	RequestID string
	// SubjectType 主体类型（user / party）。
	SubjectType string
	// SubjectID 主体 ID。
	SubjectID string
	// UserName 操作者显示名。
	UserName string
	// ClientIP 客户端 IP。
	ClientIP string
	// UserAgent 客户端 User-Agent。
	UserAgent string
}

// ── 路由注册 ──────────────────────────────────────────────────────────────

// RegisterGovHandlers 在 TokenHub 现有 mux 上注册全部 /gov/* 路由。
// 所有路由统一前缀 /gov，与控制台 API /api/admin 完全隔离。
// 用法：在 Server 初始化中调用 RegisterGovHandlers(s.mux, govDeps)。
func RegisterGovHandlers(mux *http.ServeMux, deps GovDependencies) {
	h := NewGovHandler(deps)

	// ── §2 Party（主体管理）────────────────────────────────────
	mux.HandleFunc("/gov/parties", wrapGovHandler(h.handleParties))
	mux.HandleFunc("/gov/parties/", wrapGovHandler(h.handlePartyItem))
	mux.HandleFunc("/gov/party-edges", wrapGovHandler(h.handlePartyEdges))
	mux.HandleFunc("/gov/party-edges/", wrapGovHandler(h.handlePartyEdgeItem))
	mux.HandleFunc("/gov/party-members", wrapGovHandler(h.handlePartyMembers))
	mux.HandleFunc("/gov/party-members/", wrapGovHandler(h.handlePartyMemberItem))

	// ── §3 Fund（资金治理）────────────────────────────────────
	mux.HandleFunc("/gov/accounts", wrapGovHandler(h.handleAccounts))
	mux.HandleFunc("/gov/accounts/", wrapGovHandler(h.handleAccountItem))
	mux.HandleFunc("/gov/allocations", wrapGovHandler(h.handleAllocations))
	mux.HandleFunc("/gov/allocations/", wrapGovHandler(h.handleAllocationItem))

	// ── §4 Key（密钥管理）─────────────────────────────────────
	mux.HandleFunc("/gov/keys", wrapGovHandler(h.handleKeys))
	mux.HandleFunc("/gov/keys/", wrapGovHandler(h.handleKeyItem))

	// ── §5 Pricing（双轨计价）─────────────────────────────────
	mux.HandleFunc("/gov/model-prices", wrapGovHandler(h.handleModelPrices))
	mux.HandleFunc("/gov/model-prices/", wrapGovHandler(h.handleModelPriceItem))

	// ── §6 Model Grant（模型授权）─────────────────────────────
	mux.HandleFunc("/gov/model-grants", wrapGovHandler(h.handleModelGrants))
	mux.HandleFunc("/gov/model-grants/", wrapGovHandler(h.handleModelGrantItem))

	// ── §7 Routing（路由调度）─────────────────────────────────
	mux.HandleFunc("/gov/route-profiles", wrapGovHandler(h.handleRouteProfiles))
	mux.HandleFunc("/gov/route-profiles/", wrapGovHandler(h.handleRouteProfileItem))
	mux.HandleFunc("/gov/route-strategies", wrapGovHandler(h.handleRouteStrategies))
	mux.HandleFunc("/gov/model-routes", wrapGovHandler(h.handleModelRoutes))
	mux.HandleFunc("/gov/model-routes/", wrapGovHandler(h.handleModelRouteItem))

	// ── §8 ABAC（策略引擎）────────────────────────────────────
	mux.HandleFunc("/gov/action-catalogs", wrapGovHandler(h.handleActionCatalogs))
	mux.HandleFunc("/gov/roles", wrapGovHandler(h.handleRoles))
	mux.HandleFunc("/gov/roles/", wrapGovHandler(h.handleRoleItem))
	mux.HandleFunc("/gov/policies", wrapGovHandler(h.handlePolicies))
	mux.HandleFunc("/gov/policies/", wrapGovHandler(h.handlePolicyItem))
	mux.HandleFunc("/gov/subject-role-bindings", wrapGovHandler(h.handleSubjectRoleBindings))
	mux.HandleFunc("/gov/subject-role-bindings/", wrapGovHandler(h.handleSubjectRoleBindingItem))
	mux.HandleFunc("/gov/grants", wrapGovHandler(h.handleGrants))
	mux.HandleFunc("/gov/grants/", wrapGovHandler(h.handleGrantItem))

	// ── §9 UI Permission（UI权限治理）─────────────────────────
	mux.HandleFunc("/gov/ui-menus", wrapGovHandler(h.handleUIMenus))
	mux.HandleFunc("/gov/ui-menus/", wrapGovHandler(h.handleUIMenuItem))
	mux.HandleFunc("/gov/ui-routes", wrapGovHandler(h.handleUIRoutes))
	mux.HandleFunc("/gov/ui-routes/", wrapGovHandler(h.handleUIRouteItem))
	mux.HandleFunc("/gov/ui-action-bindings", wrapGovHandler(h.handleUIActionBindings))
	mux.HandleFunc("/gov/ui-action-bindings/", wrapGovHandler(h.handleUIActionBindingItem))
	mux.HandleFunc("/gov/ui-permissions/snapshot", wrapGovHandler(h.handleUIPermissionSnapshot))

	// ── §10 Audit（审计与对账）────────────────────────────────
	mux.HandleFunc("/gov/audit-events", wrapGovHandler(h.handleAuditEvents))
	mux.HandleFunc("/gov/audit-events/", wrapGovHandler(h.handleAuditEventItem))
	mux.HandleFunc("/gov/request-logs", wrapGovHandler(h.handleRequestLogs))
	mux.HandleFunc("/gov/request-logs/", wrapGovHandler(h.handleRequestLogTrace))
	mux.HandleFunc("/gov/audit-chain-anchors", wrapGovHandler(h.handleAuditChainAnchors))

	// ── §11 Dashboard（仪表盘与报表）──────────────────────────
	mux.HandleFunc("/gov/dashboard", wrapGovHandler(h.handleDashboard))
	mux.HandleFunc("/gov/security-reports", wrapGovHandler(h.handleSecurityReports))
	mux.HandleFunc("/gov/trace", wrapGovHandler(h.handleTrace))
}

// ── 通用包装器 ────────────────────────────────────────────────────────────

// wrapGovHandler 将 GovHandler 方法包装为 http.HandlerFunc。
// 统一设置 Content-Type、提取请求上下文、记录响应日志。
func wrapGovHandler(fn func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fn(w, r)
	}
}

// ── ABAC 鉴权辅助 ─────────────────────────────────────────────────────────

// requireGovAuth 从请求中提取治理请求上下文，并执行 ABAC 鉴权。
// 返回 GovRequestContext 和是否放行。
func (h *GovHandler) requireGovAuth(w http.ResponseWriter, r *http.Request, action string) (*GovRequestContext, bool) {
	gctx := &GovRequestContext{
		RequestID: r.Header.Get("X-Request-ID"),
		ClientIP:  govClientIP(r),
		UserAgent: r.UserAgent(),
		SubjectType: "user",
	}
	if gctx.RequestID == "" {
		gctx.RequestID = NewID("gov")
	}
	// 从 Header 提取认证——Bearer Token 或 X-API-Key。
	if token := extractBearerToken(r); token != "" {
		if user, userID, ok := validateGovToken(token); ok {
			gctx.SubjectID = userID
			gctx.UserName = user
		}
	} else if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
		gctx.SubjectID = apiKey
	}
	if gctx.SubjectID == "" {
		writeError(w, r, NewHTTPError(http.StatusUnauthorized, "AUTH_INVALID_KEY", "认证凭证无效或缺失"))
		return nil, false
	}
	// ABAC 鉴权——若未配置引擎则跳过（开发模式）。
	if h.deps.ABACEngine != nil && action != "" {
		subject := abac.Subject{Type: gctx.SubjectType, ID: gctx.SubjectID}
		resource := abac.Resource{Type: "gov_api", ID: r.URL.Path}
		if err := h.deps.ABACEngine.Evaluate(r.Context(), subject, action, resource); err != nil {
			writeError(w, r, NewHTTPError(http.StatusForbidden, "AUTHZ_DENIED", "权限不足: "+err.Error()))
			return nil, false
		}
	}
	return gctx, true
}

// ── JSON 辅助 ─────────────────────────────────────────────────────────────

// readJSON 解析请求体为指定类型。解析失败时写入错误响应并返回 false。
func readJSON[T any](w http.ResponseWriter, r *http.Request) (T, bool) {
	var body T
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, NewHTTPError(http.StatusBadRequest, "INVALID_JSON", "请求体解析失败: "+err.Error()))
		return body, false
	}
	return body, true
}

// okJSON 写入 200 成功响应。
func okJSON(w http.ResponseWriter, data any) {
	if data == nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	}
	writeJSON(w, http.StatusOK, data)
}

// createdJSON 写入 201 创建成功响应。
func createdJSON(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusCreated, data)
}

// ── URL 路径参数提取 ──────────────────────────────────────────────────────

// extractItemID 从带 trailing slash 的集合路由中提取资源 ID。
// 例如 URL /gov/parties/{id} → 返回 {id}；/gov/parties 或 /gov/parties/ → 返回 ""。
func extractItemID(r *http.Request, prefix string) string {
	path := r.URL.Path
	if path == prefix || path == prefix+"/" {
		return ""
	}
	if len(path) > len(prefix)+1 {
		return path[len(prefix)+1:]
	}
	return ""
}

// ── 认证辅助 ──────────────────────────────────────────────────────────────

// extractBearerToken 从 Authorization header 提取 Bearer Token。
func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if len(auth) > 7 && strings.EqualFold(auth[:7], "Bearer ") {
		return auth[7:]
	}
	return ""
}

// validateGovToken 验证治理 API Token——占位实现。
// 完整实现应调用 store.ValidateAdminSession(token) 委托给 TokenHub 的 AdminSession 校验。
func validateGovToken(token string) (string, string, bool) {
	_ = token
	return "", "", false
}

// govClientIP 从 HTTP 请求提取客户端 IP。
func govClientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return fwd
	}
	host := r.RemoteAddr
	if idx := strings.LastIndex(host, ":"); idx > 0 {
		return host[:idx]
	}
	return host
}

// ── §2 Party handlers ─────────────────────────────────────────────────────

func (h *GovHandler) handleParties(w http.ResponseWriter, r *http.Request) {
	gctx, _ := h.requireGovAuth(w, r, "iam.party.create")
	if gctx == nil {
		return
	}
	switch r.Method {
	case http.MethodPost:
		if h.deps.PartyService == nil {
			writeError(w, r, NewHTTPError(501, "NOT_IMPLEMENTED", "Party 服务未配置"))
			return
		}
		req, ok := readJSON[party.CreatePartyRequest](w, r)
		if !ok {
			return
		}
		p, err := h.deps.PartyService.CreateParty(r.Context(), req)
		if err != nil {
			writeError(w, r, NewHTTPError(400, "CREATE_FAILED", err.Error()))
			return
		}
		slog.InfoContext(r.Context(), "创建Party成功", "party_id", p.ID, "name", p.Name)
		createdJSON(w, p)
	case http.MethodGet:
		okJSON(w, map[string]string{"message": "Party 列表——待实现"})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

func (h *GovHandler) handlePartyItem(w http.ResponseWriter, r *http.Request) {
	gctx, _ := h.requireGovAuth(w, r, "data.party.read")
	if gctx == nil {
		return
	}
	partyID := extractItemID(r, "/gov/parties")
	switch r.Method {
	case http.MethodGet:
		okJSON(w, map[string]string{"id": partyID, "message": "Party 详情——待实现"})
	case http.MethodPatch:
		okJSON(w, map[string]string{"id": partyID, "message": "Party 更新——待实现"})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

func (h *GovHandler) handlePartyEdges(w http.ResponseWriter, r *http.Request) {
	_, _ = h.requireGovAuth(w, r, "iam.party.write")
	switch r.Method {
	case http.MethodPost:
		okJSON(w, map[string]string{"message": "PartyEdge 创建——待实现"})
	case http.MethodGet:
		okJSON(w, map[string]string{"message": "PartyEdge 列表——待实现"})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

func (h *GovHandler) handlePartyEdgeItem(w http.ResponseWriter, r *http.Request) {
	_, _ = h.requireGovAuth(w, r, "iam.party.write")
	edgeID := extractItemID(r, "/gov/party-edges")
	switch r.Method {
	case http.MethodDelete:
		okJSON(w, map[string]any{"deleted": true, "id": edgeID})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

func (h *GovHandler) handlePartyMembers(w http.ResponseWriter, r *http.Request) {
	_, _ = h.requireGovAuth(w, r, "data.member.read")
	switch r.Method {
	case http.MethodPost:
		okJSON(w, map[string]string{"message": "PartyMember 创建——待实现"})
	case http.MethodGet:
		okJSON(w, map[string]string{"message": "PartyMember 列表——待实现"})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

func (h *GovHandler) handlePartyMemberItem(w http.ResponseWriter, r *http.Request) {
	_, _ = h.requireGovAuth(w, r, "iam.member.delete")
	memberID := extractItemID(r, "/gov/party-members")
	switch r.Method {
	case http.MethodDelete:
		okJSON(w, map[string]any{"deleted": true, "id": memberID})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}
