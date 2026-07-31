package authz

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"gorm.io/gorm"
)

// contextKey 是 context 键的内部类型，避免与其他包的键冲突。
type contextKey string

const (
	// CtxPrincipalType context 中存储的主体类型键。
	CtxPrincipalType contextKey = "authz:principal_type"
	// CtxPrincipalID context 中存储的主体 ID 键。
	CtxPrincipalID contextKey = "authz:principal_id"
)

// MiddlewareConfig 鉴权中间件的配置参数。
type MiddlewareConfig struct {
	// DB 用于查询 grants 表的数据库连接。
	DB *gorm.DB

	// AxisActionMap 定义 URL 路径前缀到 (axis, action) 的映射。
	// 键为请求路径前缀（如 "/api/fund/"），值为对应的治理轴与操作。
	// 若未配置则默认拒绝（最小权限原则）。
	AxisActionMap map[string][2]string // pathPrefix → [axis, action]
}

// NewMiddleware 返回一个 HTTP 中间件，从请求 context 提取主体信息，
// 调用 grants 表评估权限。无权限返回 403 + AUTHZ_DENIED。
//
// 中间件按如下顺序判定：
//  1. 从 context 提取 principal_type 与 principal_id
//  2. 根据请求 URL 路径匹配 AxisActionMap，确定 axis 与 action
//  3. 调用 Evaluate 查询 grants 表
//  4. DENY 优先于 ALLOW；无匹配规则默认拒绝
//
// 使用方式：
//
//	mux.Handle("/api/fund/", authz.NewMiddleware(cfg)(fundHandler))
func NewMiddleware(cfg MiddlewareConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 第 1 步：提取主体信息。
			pt, ok := r.Context().Value(CtxPrincipalType).(string)
			if !ok || pt == "" {
				writeAuthzDenied(w, r, "缺少主体类型")
				return
			}
			pid, ok := r.Context().Value(CtxPrincipalID).(string)
			if !ok || pid == "" {
				writeAuthzDenied(w, r, "缺少主体 ID")
				return
			}

			// 第 2 步：根据请求路径匹配 axis 与 action。
			axis, action := matchAxisAction(r.URL.Path, cfg.AxisActionMap)
			if axis == "" || action == "" {
				writeAuthzDenied(w, r, "未找到匹配的轴与操作映射")
				return
			}

			// 第 3 步：评估权限。
			allowed, err := Evaluate(cfg.DB, pt, pid, axis, action)
			if err != nil {
				slog.ErrorContext(r.Context(), "授权评估失败",
					"error", err,
					"principal_type", pt,
					"principal_id", pid,
					"axis", axis,
					"action", action,
				)
				http.Error(w, `{"error":"INTERNAL_ERROR","message":"授权评估异常"}`, http.StatusInternalServerError)
				return
			}

			// 第 4 步：判定结果。
			if !allowed {
				slog.InfoContext(r.Context(), "授权拒绝",
					"principal_type", pt,
					"principal_id", pid,
					"axis", axis,
					"action", action,
					"path", r.URL.Path,
				)
				writeAuthzDenied(w, r, "AUTHZ_DENIED")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// matchAxisAction 在映射表中查找最长的前缀匹配，返回对应的 (axis, action)。
// 未匹配时返回空字符串。
func matchAxisAction(path string, m map[string][2]string) (string, string) {
	best := ""
	var bestAxis, bestAction string
	for prefix, pair := range m {
		if len(prefix) > len(best) && len(path) >= len(prefix) && path[:len(prefix)] == prefix {
			best = prefix
			bestAxis = pair[0]
			bestAction = pair[1]
		}
	}
	return bestAxis, bestAction
}

// writeAuthzDenied 写入 403 响应，携带 AUTHZ_DENIED 错误码（PRD §6.2）。
func writeAuthzDenied(w http.ResponseWriter, r *http.Request, detail string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusForbidden)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":   "AUTHZ_DENIED",
		"message": detail,
	})
}
