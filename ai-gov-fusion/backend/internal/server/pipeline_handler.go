// Package server 实现流水线聊天补全 HTTP handler。
// 将 14 步数据面 Pipeline 编排器接入 TokenHub /v1/chat/completions 路由。
//
// 降级策略：
//   - PipelineEnabled=false → 回退到原有 handler
//   - Pipeline 未注入（nil）→ 回退到原有 handler
//   - 管线任一步骤失败 → 回退到原有 handler
//   - 流式请求 → 回退到原有 handler（流式续期尚未集成）
//
// 安全原则：降级路径保证请求不丢不拦，宁可走老路也不拒绝服务。
package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"tokenhub/backend/internal/server/modelgrant"
)

// pipelineChatHandler /v1/chat/completions 的 Pipeline 入口 handler。
// 签名与 http.HandlerFunc 兼容，通过 gatewayInFlight 包装调用。
//
// 执行流程：
//  1. 校验 HTTP 方法（仅 POST）
//  2. 认证 API Key → 获取 Project + APIKey
//  3. 解析 ChatCompletionRequest → 提取 model 名称
//  4. 判断 Pipeline 是否启用——未启用则回退到原有 handler
//  5. 设置 context（request_id、model_name）供管线步骤消费
//  6. 懒初始化 Pipeline（首次请求触发 buildPipeline）
//  7. 调用 pipeline.Execute(ctx, r)
//  8. 返回 OpenAI 兼容 JSON 响应
//  9. 任一步骤失败 → 回退到原有 handler
func (s *Server) pipelineChatHandler(w http.ResponseWriter, r *http.Request) {
	// 步骤 1：校验 HTTP 方法
	if r.Method != http.MethodPost {
		writeError(w, r, NewHTTPError(405, "method_not_allowed", "请求方法不允许"))
		return
	}

	// 步骤 2：密钥鉴权——复用 existing authenticate 逻辑
	project, key, err := s.authenticate(r)
	if err != nil {
		writeError(w, r, err)
		return
	}

	// 步骤 3：解析请求体
	var req ChatCompletionRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, r, NewHTTPError(400, "invalid_request", err.Error()))
		return
	}
	if req.Model == "" {
		writeError(w, r, NewHTTPError(400, "missing_model", "model 参数为必填"))
		return
	}

	// 步骤 4：判断 Pipeline 是否启用——未启用或 nil 时降级
	if !s.config.PipelineEnabled || s.pipeline == nil {
		s.fallbackChatCompletions(w, r, project, key, req)
		return
	}

	// 步骤 4b：流式请求暂不支持 Pipeline 路径，降级到原有 handler
	if req.Stream {
		slog.DebugContext(r.Context(), "Pipeline: 流式请求降级到原有路径",
			"model", req.Model,
		)
		s.fallbackChatCompletions(w, r, project, key, req)
		return
	}

	// 步骤 5：设置 context——注入 request_id 和 model_name 供管线各步骤消费
	requestID := NewID("req")
	ctx := context.WithValue(r.Context(), "request_id", requestID)
	ctx = context.WithValue(ctx, "model_name", req.Model)
	r = r.WithContext(ctx)

	// 步骤 6：懒初始化 Pipeline（首次请求时 buildPipeline 尚未被调用）
	if s.govDeps.Pipeline == nil && s.pipeline == nil {
		s.pipeline = s.buildPipeline()
	}

	// 步骤 7：执行管线
	result, pipeErr := s.pipeline.Execute(r.Context(), r)
	if pipeErr != nil {
		// V-4.1 修复：若为 ModelGrant 拒绝（访问拒绝或预算超限），直接返回 403，
		// 不降级到无 ModelGrant 检查的旧路径。
		if errors.Is(pipeErr, modelgrant.ErrModelAccessDenied) || errors.Is(pipeErr, modelgrant.ErrModelBudgetExceeded) {
			slog.WarnContext(r.Context(), "Pipeline: ModelGrant 拒绝，不降级",
				"request_id", requestID,
				"model", req.Model,
				"error", pipeErr.Error(),
			)
			writeError(w, r, NewHTTPError(403, "model_access_denied", pipeErr.Error()))
			return
		}
		// 管线步骤失败——降级到原有路径（fallback 中有 ModelGrant 检查）
		slog.WarnContext(r.Context(), "Pipeline: 执行失败，降级到原有路径",
			"request_id", requestID,
			"model", req.Model,
			"error", pipeErr.Error(),
		)
		s.fallbackChatCompletions(w, r, project, key, req)
		return
	}

	// 步骤 8：返回 OpenAI 兼容 JSON 响应
	if result.Upstream == nil || result.Upstream.Body == nil {
		// 上游响应为空——降级到原有路径
		slog.WarnContext(r.Context(), "Pipeline: 上游响应为空，降级到原有路径",
			"request_id", requestID,
			"model", req.Model,
		)
		s.fallbackChatCompletions(w, r, project, key, req)
		return
	}

	w.Header().Set("x-request-id", requestID)
	w.Header().Set("content-type", "application/json")
	w.Header().Set("x-pipeline-enabled", "true")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(result.Upstream.Body); err != nil {
		slog.ErrorContext(r.Context(), "Pipeline: 写入响应体失败",
			"request_id", requestID,
			"error", err,
		)
	}
}

// fallbackChatCompletions 降级到原有 handleChatCompletions 完整逻辑。
// 复制自 handleChatCompletions 的全部代码（流式 + 非流式），
// 保证降级路径与原有行为完全一致。
//
// 参数 project/key/req 已在上层完成认证和解析，无需重复。
//
// 安全增强（V-4.1）：在进入旧路径之前执行 ModelGrant 检查，
// 防止流式请求和管线失败降级绕过模型授权。
func (s *Server) fallbackChatCompletions(w http.ResponseWriter, r *http.Request, project Project, key APIKey, req ChatCompletionRequest) {
	// ── ModelGrant 检查（V-4.1 修复：防止降级路径绕过） ──
	if s.govDeps.ModelGrantChecker != nil {
		principal := modelgrant.Principal{
			Type: "party",
			ID:   key.PartyID,
		}
		if err := s.govDeps.ModelGrantChecker.CheckAccess(r.Context(), principal, req.Model); err != nil {
			slog.WarnContext(r.Context(), "降级路径 ModelGrant 检查拒绝",
				"model", req.Model,
				"party_id", key.PartyID,
				"error", err.Error(),
			)
			writeError(w, r, NewHTTPError(403, "model_access_denied", "模型访问被拒绝"))
			return
		}
	}

	routed, ok := s.startRoutedCall(w, r, project, key, req.Model, req.Stream, req)
	if !ok {
		return
	}

	affinity, err := s.chatCacheLocalityAffinity(key.ID, r.Header, req)
	if err != nil {
		s.finishFailedRoutedCall(r, routed, nil, err)
		s.recordRequestPayload(routed.Call.RequestID, req, auditErrorPayload(err, routed.Call.RequestID))
		writeError(w, r, err)
		return
	}
	if affinity != nil {
		routed.Affinity = affinity
		routed.Call.Affinity = affinity
		routed.Routes = s.planRouteOrder(routed.Call, routed.Routes)
	}

	// ── 流式分支 —— 复制自 handleChatCompletions 的 stream 分支
	if req.Stream {
		tracker := &streamWriteTracker{writer: w}
		allowEffortFallback := normalizedReasoningEffort(req.ReasoningEffort) != nil
		_, route, usage, attempts, streamErr := executeRoutedWithStore(r.Context(), s.store, routed, allowEffortFallback,
			func(ctx context.Context, candidate RouteSelection, omitReasoningEffort bool, attempt int) (struct{}, Usage, error) {
				prepared, prepareErr := s.prepareRouteForUpstream(ctx, candidate)
				if prepareErr != nil {
					return struct{}{}, Usage{}, prepareErr
				}
				adapter, adapterErr := s.adapterForRoute(prepared)
				if adapterErr != nil {
					return struct{}{}, Usage{}, adapterErr
				}
				upstreamReq := req
				if omitReasoningEffort {
					upstreamReq.ReasoningEffort = nil
				}
				// 推迟响应头到首字节写入时设置。
				tracker.onFirstWrite = func() {
					w.Header().Set("content-type", "text/event-stream")
					w.Header().Set("cache-control", "no-cache")
					w.Header().Set("x-request-id", routed.Call.RequestID)
					s.writeRouteHeaders(w, routed.Call, prepared, attempt)
				}
				streamUsage, err := adapter.ChatStream(ctx, prepared.Provider, prepared.ProviderModel, upstreamReq, tracker)
				return struct{}{}, streamUsage, classifyStreamError(ctx, err, tracker.Wrote())
			})

		status, code := statusAndCode(streamErr)
		if streamErr == nil {
			// 上游可能返回 200 + 空 body——此时 onFirstWrite 未触发，
			// 客户端可能收不到任何流式 header。
			tracker.ensureStarted()
			s.store.MarkRouteUsed(route.Route.ID)
			s.store.MarkProviderResourceUsed(routeResourceID(route))
		}
		s.store.RecordRouteAttempts(routed.Call.RequestID, attempts)
		s.store.FinishCall(routed.Call, route, usage, status, code, s.clientIP(r), r.UserAgent())
		s.recordRequestPayload(routed.Call.RequestID, req, auditStreamPayload(status, code, streamErr))
		if streamErr != nil && !tracker.Wrote() {
			// 客户端未收到任何数据——降级为 JSON 错误响应。
			w.Header().Del("cache-control")
			s.writeRouteHeaders(w, routed.Call, lastAttemptRoute(attempts), len(attempts))
			writeError(w, r, streamErr)
		}
		return
	}

	// ── 非流式分支 —— 复制自 handleChatCompletions 的非 stream 分支
	resp, route, usage, attempts, err := s.executeRoutedChat(r, routed, req)
	if err != nil {
		s.finishFailedRoutedCall(r, routed, attempts, err)
		s.recordRequestPayload(routed.Call.RequestID, req, auditErrorPayload(err, routed.Call.RequestID))
		writeError(w, r, err)
		return
	}
	s.store.MarkRouteUsed(route.Route.ID)
	s.store.MarkProviderResourceUsed(routeResourceID(route))
	s.store.RecordRouteAttempts(routed.Call.RequestID, attempts)
	s.store.FinishCall(routed.Call, route, usage, http.StatusOK, "", s.clientIP(r), r.UserAgent())
	s.recordRequestPayload(routed.Call.RequestID, req, resp)
	w.Header().Set("x-request-id", routed.Call.RequestID)
	s.writeRouteHeaders(w, routed.Call, route, len(attempts))
	w.Header().Set("x-pipeline-enabled", "false")
	writeJSON(w, http.StatusOK, resp)
}
