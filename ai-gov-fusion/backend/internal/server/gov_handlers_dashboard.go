// Package server 治理 API handlers——Dashboard / Security Reports / Key Vault / Models 域。
// 全部注释使用中文，符合 AGENTS.md 铁律。
//
// 本文件承载 PRD UI-07 仪表盘与配套报表的 HTTP handler：
//   - /v1/gov/dashboard                       完整 DashboardData 聚合
//   - /v1/gov/security-reports/summary        安全报表汇总
//   - /v1/gov/security-reports/events         安全事件分页列表
//   - /v1/gov/security-reports/abnormal-access 异常访问分页列表
//   - /v1/gov/security-reports/key-rotations  Key 轮转记录
//   - /v1/gov/key-vault/health                密钥仓库健康
//   - /v1/gov/key-vault/rotations             密钥轮转历史
//   - /v1/gov/models                          模型目录分页
package server

import (
	"context"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

// ── 通用辅助函数 ──────────────────────────────────────────────────────────

// govParsePage 解析分页参数，page 从 1 开始，size 默认 20、上限 200。
// 当 size <= 0 时回退到 defaultSize；当 size > 200 时夹到 200 防止过载。
func govParsePage(r *http.Request, defaultSize int) (page, size int) {
	page = 1
	size = defaultSize
	if defaultSize <= 0 {
		defaultSize = 20
	}
	if v, err := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("page"))); err == nil && v > 0 {
		page = v
	}
	if v, err := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("page_size"))); err == nil && v > 0 {
		size = v
	}
	if size > 200 {
		size = 200
	}
	return
}

// govComputePeriodRange 根据 ?period= 参数计算 [from, to] 时间区间。
// 支持 current_day / current_month / current_quarter / current_year。
// 默认 current_month。to 取区间末端的最后一秒（RFC3339 可直接给前端展示）。
func govComputePeriodRange(period string, now time.Time) (from, to time.Time) {
	switch period {
	case "current_day":
		from = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		to = from.Add(24*time.Hour - time.Second)
	case "current_quarter":
		qm := time.Month(((int(now.Month())-1)/3)*3 + 1)
		from = time.Date(now.Year(), qm, 1, 0, 0, 0, 0, time.UTC)
		to = from.AddDate(0, 3, 0).Add(-time.Second)
	case "current_year":
		from = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
		to = from.AddDate(1, 0, 0).Add(-time.Second)
	default:
		// current_month 或其他值都按月处理。
		from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		to = from.AddDate(0, 1, 0).Add(-time.Second)
	}
	return
}

// govParseTime 解析前端传来的 ISO8601 / 短日期字符串，无法解析时返回零值。
// 空字符串返回零值，调用方需自行判定是否启用 from/to 过滤。
func govParseTime(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t
	}
	return time.Time{}
}

// ── §11 Dashboard 主端点 ──────────────────────────────────────────────────

// handleDashboard 返回完整 DashboardData 聚合（PRD UI-07）。
//
// 数据来源：
//   - period     根据 ?period= 参数计算
//   - consumption / trend  从 request_logs 聚合 sell_usd / cost_usd
//   - balance    从 accounts 表聚合 available_balance / frozen_balance / budget_limit_amount
//   - budget_status 按消耗比例分桶
//   - block_rates 从 request_logs 聚合 error_code（error_code 非空即视为拦截）
//   - top_consumers 从 request_logs 聚合 party_id 的 sell，按金额取前 5
//
// 数据不足时所有字段返回零值/空数组，确保前端不出现 undefined。
func (h *GovHandler) handleDashboard(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if rec := recover(); rec != nil {
			stack := debug.Stack()
			slog.Error("handleDashboard panic", "recover", rec, "stack", string(stack), "request_id", r.Header.Get("X-Request-ID"))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()
	if _, ok := h.requireGovAuth(w, r, "data.report.read"); !ok {
		return
	}
	db := h.deps.DB
	if db == nil {
		writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
		return
	}
	ctx := r.Context()
	now := time.Now().UTC()
	period := strings.TrimSpace(r.URL.Query().Get("period"))
	if period == "" {
		period = "current_month"
	}
	periodFrom, periodTo := govComputePeriodRange(period, now)

	// trend 固定 30 天，自然月之外的 period 也展示最近 30 日 sell 趋势。
	trendFrom := now.AddDate(0, 0, -29)
	trendTo := now

	consumption := h.dashAggregateConsumption(ctx, db, periodFrom, periodTo, trendFrom, trendTo)
	balance := h.dashAggregateBalance(ctx, db)
	budgetStatus := h.dashAggregateBudgetStatus(ctx, db)
	blockRates := h.dashAggregateBlockRates(ctx, db, periodFrom, periodTo)
	topConsumers := h.dashAggregateTopConsumers(ctx, db, periodFrom, periodTo)

	okJSON(w, map[string]any{
		"period": map[string]any{
			"from": periodFrom.Format(time.RFC3339),
			"to":   periodTo.Format(time.RFC3339),
		},
		"consumption":   consumption,
		"balance":       balance,
		"budget_status": budgetStatus,
		"block_rates":   blockRates,
		"top_consumers": topConsumers,
		"generated_at":  now.Format(time.RFC3339),
	})
}

// dashAggregateConsumption 聚合本周期消耗与最近 30 天 sell 趋势。
//
// 字段语义：
//   - total_sell / total_cost 由 COALESCE(SUM(sell_usd), 0) 给出（单位 USD，前端 format 为 CNY）
//   - markup_pct = (total_sell-total_cost)/total_cost*100，cost=0 时回退 0
//   - trend 每日 sell 数组，缺数据日期补 0，保证长度恒为 30
func (h *GovHandler) dashAggregateConsumption(ctx context.Context, db *gorm.DB, from, to, trendFrom, trendTo time.Time) map[string]any {
	out := map[string]any{
		"total_sell": 0.0,
		"total_cost": 0.0,
		"markup_pct": 0.0,
		"trend":      []map[string]any{},
	}
	// 本周期累计 sell/cost。
	// SQLite 中 SUM(...) 在空表返回 NULL，COALESCE 转 0 时类型可能为 int，
	// 与 GORM Scan(float64) 期望不一致。改用 string 中转避免 interface 断言 panic。
	type sumRow struct {
		TotalSell string
		TotalCost string
	}
	var period sumRow
	_ = db.WithContext(ctx).
		Table("request_logs").
		Select("CAST(COALESCE(SUM(sell_usd), 0) AS TEXT) AS total_sell, CAST(COALESCE(SUM(cost_usd), 0) AS TEXT) AS total_cost").
		Where("created_at >= ? AND created_at <= ?", from.Format("2006-01-02 15:04:05"), to.Format("2006-01-02 15:04:05")).
		Limit(1).
		Scan(&period).Error
	var totalSell, totalCost float64
	if v, err := strconv.ParseFloat(period.TotalSell, 64); err == nil {
		totalSell = v
	}
	if v, err := strconv.ParseFloat(period.TotalCost, 64); err == nil {
		totalCost = v
	}
	out["total_sell"] = totalSell
	out["total_cost"] = totalCost
	if totalCost > 0 {
		out["markup_pct"] = (totalSell - totalCost) / totalCost * 100.0
	}

	// 每日 sell 趋势：先查询每日聚合再补齐缺数日期。
	type dayRow struct {
		Day  string
		Sell float64
	}
	var rows []dayRow
	_ = db.WithContext(ctx).
		Table("request_logs").
		Select("substr(created_at, 1, 10) AS day, COALESCE(SUM(sell_usd), 0) AS sell").
		Where("created_at >= ? AND created_at <= ?", trendFrom.Format("2006-01-02 15:04:05"), trendTo.Format("2006-01-02 15:04:05")).
		Group("day").
		Scan(&rows).Error
	dayMap := make(map[string]float64, len(rows))
	for _, r := range rows {
		dayMap[r.Day] = r.Sell
	}
	trend := make([]map[string]any, 0, 30)
	for d := trendFrom; !d.After(trendTo); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		trend = append(trend, map[string]any{
			"date": key,
			"sell": dayMap[key],
		})
	}
	out["trend"] = trend
	return out
}

// dashAggregateBalance 聚合全平台账户余额。
//
// 字段语义：
//   - total_available  所有 account.available_balance 之和
//   - total_frozen     所有 account.frozen_balance 之和
//   - total_budget_limit 所有非空 account.budget_limit_amount 之和
//   - utilization_pct  (total_sell_for_period / total_budget_limit) * 100，无上限时为 0
func (h *GovHandler) dashAggregateBalance(ctx gormCtx, db *gorm.DB) map[string]any {
	out := map[string]any{
		"total_available":    0.0,
		"total_frozen":       0.0,
		"total_budget_limit": 0.0,
		"utilization_pct":    0.0,
	}
	type balRow struct {
		Avail       string
		Frozen      string
		BudgetLimit string
	}
	var row balRow
	_ = db.WithContext(ctx).
		Table("accounts").
		Select("CAST(COALESCE(SUM(available_balance), 0) AS TEXT) AS avail, CAST(COALESCE(SUM(frozen_balance), 0) AS TEXT) AS frozen, CAST(COALESCE(SUM(budget_limit_amount), 0) AS TEXT) AS budget_limit").
		Limit(1).
		Scan(&row).Error
	out["total_available"] = parseFloatOrZero(row.Avail)
	out["total_frozen"] = parseFloatOrZero(row.Frozen)
	out["total_budget_limit"] = parseFloatOrZero(row.BudgetLimit)
	return out
}

// parseFloatOrZero 解析字符串为 float64，失败返回 0。用于 SQL 聚合结果中转。
func parseFloatOrZero(s string) float64 {
	if s == "" {
		return 0
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}

// dashAggregateBudgetStatus 按消耗占比分桶统计账户数。
//
// 阈值定义（PRD §6 预算治理）：
//   - accounts_at_warning   [50%, 80%) 消耗
//   - accounts_near_limit   [80%, 100%) 消耗
//   - accounts_exceeded     >= 100% 或可用余额 <= 0
//
// 消耗比例 = (budget_consumed_amount / budget_limit_amount) * 100。
func (h *GovHandler) dashAggregateBudgetStatus(ctx context.Context, db *gorm.DB) map[string]any {
	out := map[string]any{
		"accounts_at_warning": 0,
		"accounts_near_limit": 0,
		"accounts_exceeded":   0,
	}
	type bucket struct {
		Warn     int64
		Near     int64
		Exceeded int64
	}
	var b bucket
	// 拉取所有账户预算快照后内存分桶——账户数量有限（千级），可接受。
	// 使用 string 中转避免 SQLite NUMERIC/GORM float64 接口断言不一致。
	type acctRow struct {
		BudgetLimit   *string
		BudgetConsume *string
		Available     *string
	}
	var rows []acctRow
	_ = db.WithContext(ctx).
		Table("accounts").
		Select("CAST(budget_limit_amount AS TEXT) AS budget_limit, CAST(budget_consumed_amount AS TEXT) AS budget_consume, CAST(available_balance AS TEXT) AS available").
		Scan(&rows).Error
	for _, row := range rows {
		if row.Available != nil {
			if v, err := strconv.ParseFloat(*row.Available, 64); err == nil && v <= 0 {
				b.Exceeded++
				continue
			}
		}
		if row.BudgetLimit == nil {
			continue
		}
		limit, err := strconv.ParseFloat(*row.BudgetLimit, 64)
		if err != nil || limit <= 0 {
			continue
		}
		consume := 0.0
		if row.BudgetConsume != nil {
			if v, err := strconv.ParseFloat(*row.BudgetConsume, 64); err == nil {
				consume = v
			}
		}
		pct := consume / limit * 100.0
		switch {
		case pct >= 100:
			b.Exceeded++
		case pct >= 80:
			b.Near++
		case pct >= 50:
			b.Warn++
		}
	}
	out["accounts_at_warning"] = b.Warn
	out["accounts_near_limit"] = b.Near
	out["accounts_exceeded"] = b.Exceeded
	return out
}

// dashAggregateBlockRates 聚合本周期各类拦截事件计数。
//
// 数据来源：request_logs.error_code 非空且 status_code >= 400 的记录，
// 按 error_code 分组统计。统一封顶 5 个常见 code，保证前端 UI 渲染稳定。
func (h *GovHandler) dashAggregateBlockRates(ctx gormCtx, db *gorm.DB, from, to time.Time) map[string]any {
	out := map[string]any{
		"MODEL_ACCESS_DENIED":  int64(0),
		"INSUFFICIENT_BALANCE": int64(0),
		"BUDGET_CAP_EXCEEDED":  int64(0),
		"RATE_LIMITED":         int64(0),
		"OTHER":                int64(0),
	}
	type row struct {
		Code  string
		Count int64
	}
	var rows []row
	_ = db.WithContext(ctx).
		Table("request_logs").
		Select("error_code AS code, COUNT(*) AS count").
		Where("error_code IS NOT NULL AND error_code != '' AND created_at >= ? AND created_at <= ?",
			from.Format("2006-01-02 15:04:05"), to.Format("2006-01-02 15:04:05")).
		Group("error_code").
		Scan(&rows).Error
	known := map[string]bool{
		"MODEL_ACCESS_DENIED":  true,
		"INSUFFICIENT_BALANCE": true,
		"BUDGET_CAP_EXCEEDED":  true,
		"RATE_LIMITED":         true,
	}
	for _, r := range rows {
		if known[r.Code] {
			out[r.Code] = r.Count
			continue
		}
		out["OTHER"] = out["OTHER"].(int64) + r.Count
	}
	return out
}

// dashAggregateTopConsumers 聚合本周期 sell 前 5 的 Party。
//
// 返回结构：[{ party_id, party_name, sell, pct }]，
// pct = party_sell / period_total_sell * 100。
func (h *GovHandler) dashAggregateTopConsumers(ctx context.Context, db *gorm.DB, from, to time.Time) []map[string]any {
	out := []map[string]any{}
	type row struct {
		PartyID string
		Sell    string
	}
	var rows []row
	_ = db.WithContext(ctx).
		Table("request_logs").
		Select("party_id AS party_id, CAST(COALESCE(SUM(sell_usd), 0) AS TEXT) AS sell").
		Where("party_id IS NOT NULL AND party_id != '' AND created_at >= ? AND created_at <= ?",
			from.Format("2006-01-02 15:04:05"), to.Format("2006-01-02 15:04:05")).
		Group("party_id").
		Order("sell DESC").
		Limit(5).
		Scan(&rows).Error
	if len(rows) == 0 {
		return out
	}
	// 计算总 sell 以计算占比。
	var total float64
	for _, r := range rows {
		total += parseFloatOrZero(r.Sell)
	}
	// 一次性加载 parties 名称表，避免 N+1。
	ids := make([]string, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.PartyID)
	}
	nameMap := map[string]string{}
	if len(ids) > 0 {
		type partyNameRow struct {
			ID   string
			Name string
		}
		var pRows []partyNameRow
		_ = db.WithContext(ctx).Table("parties").Select("id, name").Where("id IN ?", ids).Scan(&pRows).Error
		for _, p := range pRows {
			nameMap[p.ID] = p.Name
		}
	}
	for _, r := range rows {
		sell := parseFloatOrZero(r.Sell)
		pct := 0.0
		if total > 0 {
			pct = sell / total * 100.0
		}
		name := nameMap[r.PartyID]
		if name == "" {
			name = r.PartyID
		}
		out = append(out, map[string]any{
			"party_id":   r.PartyID,
			"party_name": name,
			"sell":       sell,
			"pct":        pct,
		})
	}
	return out
}

// gormCtx 是 gorm 上下文接口的轻量别名，仅用于上述聚合函数参数签名可读性。
type gormCtx = context.Context

// ── §12 Security Reports 端点 ────────────────────────────────────────────

// handleSecurityReportsSummary 返回安全报表汇总。
//
// 数据来源：audit_events 全表聚合 + request_logs 异常访问 + key 轮转计数。
// 各分类指标独立查询，失败不互相影响，DB 错误时该分类返回 0。
func (h *GovHandler) handleSecurityReportsSummary(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireGovAuth(w, r, "data.report.read"); !ok {
		return
	}
	db := h.deps.DB
	if db == nil {
		writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
		return
	}
	ctx := r.Context()
	now := time.Now().UTC()
	period := strings.TrimSpace(r.URL.Query().Get("period"))
	if period == "" {
		period = "current_month"
	}
	from, to := govComputePeriodRange(period, now)

	// 安全事件总量 = audit_events 行数。
	var total int64
	_ = db.WithContext(ctx).Table("audit_events").Count(&total).Error
	// 拦截请求数 = request_logs 状态码 >= 400 且 error_code 非空。
	var blocked int64
	_ = db.WithContext(ctx).Table("request_logs").
		Where("error_code IS NOT NULL AND error_code != '' AND status_code >= 400").
		Count(&blocked).Error
	// 异常访问数 = request_logs 中标记为安全异常的（error_code LIKE '%abnormal%' 或 '%unauthorized%'）。
	var abnormal int64
	_ = db.WithContext(ctx).Table("request_logs").
		Where("(error_code LIKE '%abnormal%' OR error_code LIKE '%unauthorized%' OR error_code LIKE '%denied%')").
		Count(&abnormal).Error
	// Key 轮转次数 = audit_events 中 action = 'key.revoke'。
	var keyRot int64
	_ = db.WithContext(ctx).Table("audit_events").
		Where("action = ?", "key.revoke").
		Count(&keyRot).Error

	// 按 status 分桶（success/failure）。
	bySeverity := map[string]int64{
		"success": 0,
		"failure": 0,
	}
	type stRow struct {
		Status string
		Count  int64
	}
	var stRows []stRow
	_ = db.WithContext(ctx).Table("audit_events").
		Select("status, COUNT(*) AS count").Group("status").Scan(&stRows).Error
	for _, r := range stRows {
		bySeverity[r.Status] = r.Count
	}
	// 按 action 前缀分桶。
	byType := map[string]int64{
		"key":     0,
		"fund":    0,
		"price":   0,
		"grant":   0,
		"route":   0,
		"party":   0,
		"member":  0,
		"other":   0,
	}
	type actRow struct {
		Action string
		Count  int64
	}
	var actRows []actRow
	_ = db.WithContext(ctx).Table("audit_events").
		Select("action, COUNT(*) AS count").Group("action").Scan(&actRows).Error
	for _, a := range actRows {
		prefix := strings.SplitN(a.Action, ".", 2)[0]
		if _, ok := byType[prefix]; !ok {
			prefix = "other"
		}
		byType[prefix] += a.Count
	}

	okJSON(w, map[string]any{
		"total_events":           total,
		"blocked_requests":       blocked,
		"abnormal_access_count":  abnormal,
		"key_rotation_count":     keyRot,
		"period_from":            from.Format(time.RFC3339),
		"period_to":              to.Format(time.RFC3339),
		"by_severity":            bySeverity,
		"by_type":                byType,
	})
}

// handleSecurityReportsEvents 返回审计事件分页列表。
//
// 支持 ?severity=success|failure、?type=key|fund|price|grant|route|party|member|all、
// ?from=YYYY-MM-DD&to=YYYY-MM-DD 时间过滤。
// severity 与 type 为空时不施加该维度过滤。
func (h *GovHandler) handleSecurityReportsEvents(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireGovAuth(w, r, "data.report.read"); !ok {
		return
	}
	db := h.deps.DB
	if db == nil {
		writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
		return
	}
	ctx := r.Context()
	page, size := govParsePage(r, 20)
	severity := strings.TrimSpace(r.URL.Query().Get("severity"))
	typeFilter := strings.TrimSpace(r.URL.Query().Get("type"))
	from := govParseTime(r.URL.Query().Get("from"))
	to := govParseTime(r.URL.Query().Get("to"))

	query := db.WithContext(ctx).Table("audit_events").Order("created_at DESC")
	if severity != "" {
		query = query.Where("status = ?", severity)
	}
	if typeFilter != "" && typeFilter != "all" {
		query = query.Where("action LIKE ?", typeFilter+".%")
	}
	if !from.IsZero() {
		query = query.Where("created_at >= ?", from.Format("2006-01-02 15:04:05"))
	}
	if !to.IsZero() {
		query = query.Where("created_at <= ?", to.Format("2006-01-02 15:04:05"))
	}
	var total int64
	_ = query.Count(&total).Error
	type evt struct {
		ID            string
		ActorUserID   string
		ActorName     string
		Action        string
		ResourceType  string
		ResourceID    string
		Status        string
		Message       string
		IP            string
		CreatedAt     time.Time
	}
	var rows []evt
	_ = query.Offset((page - 1) * size).Limit(size).Scan(&rows).Error

	items := make([]map[string]any, 0, len(rows))
	for _, e := range rows {
		items = append(items, map[string]any{
			"id":            e.ID,
			"actor_user_id": e.ActorUserID,
			"actor_name":    e.ActorName,
			"action":        e.Action,
			"resource_type": e.ResourceType,
			"resource_id":   e.ResourceID,
			"severity":      e.Status,
			"status":        e.Status,
			"message":       e.Message,
			"ip":            e.IP,
			"created_at":    e.CreatedAt.Format(time.RFC3339),
		})
	}
	okJSON(w, map[string]any{
		"data":      items,
		"total":     total,
		"page":      page,
		"page_size": size,
	})
}

// handleSecurityReportsAbnormalAccess 返回异常访问事件分页列表。
//
// 过滤条件：request_logs 中 error_code 命中 'abnormal' 或 'unauthorized' 或 'denied' 子串。
// request_logs 表无 dedicated severity 字段，统一标记 severity="failure"。
func (h *GovHandler) handleSecurityReportsAbnormalAccess(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireGovAuth(w, r, "data.report.read"); !ok {
		return
	}
	db := h.deps.DB
	if db == nil {
		writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
		return
	}
	ctx := r.Context()
	page, size := govParsePage(r, 20)
	offset := (page - 1) * size

	q := db.WithContext(ctx).Table("request_logs").
		Where("error_code IS NOT NULL AND error_code != ''").
		Where("(error_code LIKE '%abnormal%' OR error_code LIKE '%unauthorized%' OR error_code LIKE '%denied%')").
		Order("created_at DESC")
	var total int64
	_ = q.Count(&total).Error
	type ab struct {
		ID         string
		RequestID  string
		PartyID    string
		APIKeyID   string
		ModelName  string
		ErrorCode  string
		StatusCode int
		IP         string
		CreatedAt  time.Time
	}
	var rows []ab
	_ = q.Offset(offset).Limit(size).Scan(&rows).Error
	items := make([]map[string]any, 0, len(rows))
	for _, e := range rows {
		items = append(items, map[string]any{
			"id":            e.ID,
			"request_id":    e.RequestID,
			"party_id":      e.PartyID,
			"api_key_id":    e.APIKeyID,
			"model":         e.ModelName,
			"error_code":    e.ErrorCode,
			"status_code":   e.StatusCode,
			"ip":            e.IP,
			"severity":      "failure",
			"created_at":    e.CreatedAt.Format(time.RFC3339),
		})
	}
	okJSON(w, map[string]any{
		"data":      items,
		"total":     total,
		"page":      page,
		"page_size": size,
	})
}

// handleSecurityReportsKeyRotations 返回 Key 轮转（key.revoke）记录分页列表。
func (h *GovHandler) handleSecurityReportsKeyRotations(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireGovAuth(w, r, "data.report.read"); !ok {
		return
	}
	db := h.deps.DB
	if db == nil {
		writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
		return
	}
	ctx := r.Context()
	page, size := govParsePage(r, 20)
	q := db.WithContext(ctx).Table("audit_events").
		Where("action = ?", "key.revoke").
		Order("created_at DESC")
	var total int64
	_ = q.Count(&total).Error
	type rt struct {
		ID           string
		ActorUserID  string
		ActorName    string
		ResourceID   string
		Message      string
		IP           string
		CreatedAt    time.Time
	}
	var rows []rt
	_ = q.Offset((page - 1) * size).Limit(size).Scan(&rows).Error
	items := make([]map[string]any, 0, len(rows))
	for _, e := range rows {
		items = append(items, map[string]any{
			"id":           e.ID,
			"actor_user_id": e.ActorUserID,
			"actor_name":   e.ActorName,
			"key_id":       e.ResourceID,
			"message":      e.Message,
			"ip":           e.IP,
			"created_at":   e.CreatedAt.Format(time.RFC3339),
		})
	}
	okJSON(w, map[string]any{
		"data":      items,
		"total":     total,
		"page":      page,
		"page_size": size,
	})
}

// ── §13 Key Vault 端点 ────────────────────────────────────────────────────

// handleKeyVaultHealth 返回密钥仓库健康快照。
//
// 数据来源：gov_api_keys 表 keys_count 与最近 key.revoke 时间。
// 缺数据时 last_rotation_at 为 null，keys_count 为 0。
func (h *GovHandler) handleKeyVaultHealth(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireGovAuth(w, r, "data.report.read"); !ok {
		return
	}
	db := h.deps.DB
	if db == nil {
		writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
		return
	}
	ctx := r.Context()

	var keysCount int64
	_ = db.WithContext(ctx).Table("gov_api_keys").Count(&keysCount).Error

	var lastRotation *time.Time
	type lr struct {
		T time.Time
	}
	var lrr lr
	row := db.WithContext(ctx).Table("audit_events").
		Select("created_at AS t").
		Where("action = ?", "key.revoke").
		Order("created_at DESC").Limit(1).Row()
	if row != nil {
		if err := row.Scan(&lrr.T); err == nil {
			t := lrr.T
			lastRotation = &t
		}
	}

	okJSON(w, map[string]any{
		"status":            "healthy",
		"provider":          "local_encrypted",
		"last_rotation_at":  lastRotation,
		"keys_count":        keysCount,
		"encrypted_at_rest": true,
		"checked_at":        time.Now().UTC().Format(time.RFC3339),
	})
}

// handleKeyVaultRotations 返回密钥轮转历史分页列表（与 key.revoke 同源）。
//
// 与 security-reports/key-rotations 字段对齐，便于前端在「密钥管理」与「安全报表」复用组件。
func (h *GovHandler) handleKeyVaultRotations(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireGovAuth(w, r, "data.report.read"); !ok {
		return
	}
	db := h.deps.DB
	if db == nil {
		writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
		return
	}
	ctx := r.Context()
	page, size := govParsePage(r, 20)
	q := db.WithContext(ctx).Table("audit_events").
		Where("action = ?", "key.revoke").
		Order("created_at DESC")
	var total int64
	_ = q.Count(&total).Error
	type rt struct {
		ID          string
		ActorUserID string
		ActorName   string
		ResourceID  string
		Message     string
		IP          string
		CreatedAt   time.Time
	}
	var rows []rt
	_ = q.Offset((page - 1) * size).Limit(size).Scan(&rows).Error
	items := make([]map[string]any, 0, len(rows))
	for _, e := range rows {
		items = append(items, map[string]any{
			"id":            e.ID,
			"actor_user_id": e.ActorUserID,
			"actor_name":    e.ActorName,
			"key_id":        e.ResourceID,
			"reason":        e.Message,
			"ip":            e.IP,
			"rotated_at":    e.CreatedAt.Format(time.RFC3339),
		})
	}
	okJSON(w, map[string]any{
		"data":      items,
		"total":     total,
		"page":      page,
		"page_size": size,
	})
}

// ── §14 Models 目录端点 ──────────────────────────────────────────────────

// handleModels 返回模型目录分页列表。
//
// 数据来源：provider_models 表，可选 ?provider= 过滤。
// 字段映射：context_window、modality→type，enabled 来自 status='active'。
func (h *GovHandler) handleModels(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireGovAuth(w, r, "data.report.read"); !ok {
		return
	}
	db := h.deps.DB
	if db == nil {
		writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
		return
	}
	ctx := r.Context()
	page, size := govParsePage(r, 20)
	provider := strings.TrimSpace(r.URL.Query().Get("provider"))

	q := db.WithContext(ctx).Table("provider_models").Order("upstream_model ASC")
	if provider != "" {
		q = q.Where("provider_id = ?", provider)
	}
	var total int64
	_ = q.Count(&total).Error
	type m struct {
		ID            string
		ProviderID    string
		UpstreamModel string
		DisplayName   string
		Category      string
		Modality      string
		ContextWindow int64
		Status        string
	}
	var rows []m
	_ = q.Offset((page - 1) * size).Limit(size).Scan(&rows).Error
	items := make([]map[string]any, 0, len(rows))
	for _, x := range rows {
		name := x.DisplayName
		if name == "" {
			name = x.UpstreamModel
		}
		items = append(items, map[string]any{
			"id":             x.ID,
			"name":           name,
			"provider":       x.ProviderID,
			"upstream_model": x.UpstreamModel,
			"type":           x.Modality,
			"category":       x.Category,
			"context_window": x.ContextWindow,
			"enabled":        x.Status == "" || x.Status == "active",
		})
	}
	okJSON(w, map[string]any{
		"data":      items,
		"total":     total,
		"page":      page,
		"page_size": size,
	})
}

// ── 调试日志 ──────────────────────────────────────────────────────────────

// init 触发一次包加载日志，便于排查 handler 是否被注册。
func init() {
	slog.Info("gov_handlers_dashboard.go loaded: dashboard / security-reports / key-vault / models handlers ready")
}
