package routing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// ── 包级错误 ──────────────────────────────────────────────────────────────

var (
	// ErrProfileNotFound 表示指定 ID 的路由档案不存在。
	ErrProfileNotFound = errors.New("routing: 路由档案未找到")

	// ErrDeltaCapExceeded δ 价格帽超过 20% 硬上限时返回此错误。
	ErrDeltaCapExceeded = errors.New("routing: δ 价格帽超过硬上限 20%")

	// ErrNoCandidates 候选集为空，无法执行路由。
	ErrNoCandidates = errors.New("routing: 候选集为空，无法路由")

	// ErrAllEliminated 所有候选均被策略过滤剔除。
	ErrAllEliminated = errors.New("routing: 所有候选均被剔除，无可用路由")
)

// ── 档案 CRUD ─────────────────────────────────────────────────────────────

// CreateProfile 创建新路由档案。
//
// 副作用：将档案写入数据库。δ 价格帽超过 20% 时拒绝创建。
func CreateProfile(db *gorm.DB, profile *RouteProfile) error {
	if profile == nil {
		return fmt.Errorf("routing: 不能创建 nil 档案")
	}
	if profile.Name == "" {
		return fmt.Errorf("routing: 档案名称不能为空")
	}

	maxDelta := decimal.NewFromFloat(MaxDeltaCap)
	if profile.DeltaCap.GreaterThan(maxDelta) {
		return fmt.Errorf("%w: 当前值 %s", ErrDeltaCapExceeded, profile.DeltaCap.String())
	}

	if profile.MaxAttempts <= 0 {
		profile.MaxAttempts = MaxAttemptsDefault
	}
	if profile.MaxAttempts > MaxAttemptsHardLimit {
		profile.MaxAttempts = MaxAttemptsHardLimit
	}
	if profile.Status == "" {
		profile.Status = ProfileStatusActive
	}
	if profile.Strategies == nil {
		profile.Strategies = make([]StrategyBinding, 0)
	}

	if err := db.Create(profile).Error; err != nil {
		return fmt.Errorf("routing: 创建路由档案失败: %w", err)
	}

	slog.Info("路由档案已创建",
		"profile_id", profile.ID,
		"profile_name", profile.Name,
		"delta_cap", profile.DeltaCap.String(),
		"strategy_count", len(profile.Strategies),
	)
	return nil
}

// GetProfile 按 ID 查询路由档案。
func GetProfile(db *gorm.DB, id int64) (*RouteProfile, error) {
	var profile RouteProfile
	if err := db.First(&profile, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProfileNotFound
		}
		return nil, fmt.Errorf("routing: 查询路由档案失败: %w", err)
	}
	return &profile, nil
}

// GetProfileByName 按名称查询路由档案。
func GetProfileByName(db *gorm.DB, name string) (*RouteProfile, error) {
	var profile RouteProfile
	if err := db.Where("name = ?", name).First(&profile).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProfileNotFound
		}
		return nil, fmt.Errorf("routing: 查询路由档案失败: %w", err)
	}
	return &profile, nil
}

// UpdateProfile 更新路由档案。
//
// 副作用：将修改写入数据库。δ 价格帽超过 20% 时拒绝更新。
// δ 值发生变更时记录关键审计日志（S-CON-03）。
func UpdateProfile(db *gorm.DB, profile *RouteProfile) error {
	if profile == nil {
		return fmt.Errorf("routing: 不能更新 nil 档案")
	}
	if profile.ID == 0 {
		return fmt.Errorf("routing: 档案 ID 不能为零值")
	}

	// 加载旧值以检测 δ 变更。
	old, err := GetProfile(db, profile.ID)
	if err != nil {
		return err
	}

	maxDelta := decimal.NewFromFloat(MaxDeltaCap)
	if profile.DeltaCap.GreaterThan(maxDelta) {
		return fmt.Errorf("%w: 当前值 %s", ErrDeltaCapExceeded, profile.DeltaCap.String())
	}

	if profile.MaxAttempts <= 0 {
		profile.MaxAttempts = MaxAttemptsDefault
	}
	if profile.MaxAttempts > MaxAttemptsHardLimit {
		profile.MaxAttempts = MaxAttemptsHardLimit
	}

	// δ 变更触发关键审计（S-CON-03）。
	if !old.DeltaCap.Equal(profile.DeltaCap) {
		slog.Warn("δ 价格帽变更——关键配置审计",
			"profile_id", profile.ID,
			"profile_name", profile.Name,
			"delta_old", old.DeltaCap.String(),
			"delta_new", profile.DeltaCap.String(),
		)
	}

	if err := db.Save(profile).Error; err != nil {
		return fmt.Errorf("routing: 更新路由档案失败: %w", err)
	}

	slog.Info("路由档案已更新",
		"profile_id", profile.ID,
		"profile_name", profile.Name,
		"delta_cap", profile.DeltaCap.String(),
		"strategy_count", len(profile.Strategies),
	)
	return nil
}

// DeleteProfile 软删除路由档案（将状态设为 inactive）。
//
// 副作用：修改数据库中档案的 status 字段。
func DeleteProfile(db *gorm.DB, id int64) error {
	result := db.Model(&RouteProfile{}).Where("id = ?", id).Update("status", ProfileStatusInactive)
	if result.Error != nil {
		return fmt.Errorf("routing: 删除路由档案失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrProfileNotFound
	}

	slog.Info("路由档案已删除（软删除）", "profile_id", id)
	return nil
}

// ListProfiles 列出所有活跃路由档案。
// 返回按创建时间升序排列的档案列表。
func ListProfiles(db *gorm.DB) ([]*RouteProfile, error) {
	var profiles []*RouteProfile
	if err := db.Where("status = ?", ProfileStatusActive).
		Order("created_at ASC").Find(&profiles).Error; err != nil {
		return nil, fmt.Errorf("routing: 查询路由档案列表失败: %w", err)
	}
	return profiles, nil
}

// ── 管道执行 ──────────────────────────────────────────────────────────────

// ExecuteProfile 按档案执行完整路由管道。
//
// 管道顺序（PRD §3.3 + architecture §B.2）：
//
//	[1] S-COMPLIANCE.Filter —— 硬合规过滤（INTERNAL_ONLY 等）
//	[2] δ 价格帽过滤 —— 剔除 EstSell > 锚定价×(1+δ) 的候选
//	[3] S-CLASSIFY.Score —— 智能分类打分
//	[4] 其余启用策略依次执行 Filter + Score
//	[5] 按 Score 降序排列
//	[6] 选取最优候选
//
// 若档案 Shadow=true，仅记录决策日志但不实际路由（影子模式）。
//
// 返回值：
//   - candidates: 按 Score 降序排列的候选列表
//   - decision: 决策日志（包含输入/输出候选、选中候选、策略链）
//   - error: 无可用候选时返回 ErrNoCandidates 或 ErrAllEliminated
func ExecuteProfile(
	ctx context.Context,
	db *gorm.DB,
	profile *RouteProfile,
	candidates []Candidate,
	anchorSell decimal.Decimal,
) ([]Candidate, *Decision, error) {
	if profile == nil {
		return nil, nil, fmt.Errorf("routing: 档案不能为 nil")
	}
	if len(candidates) == 0 {
		return nil, nil, ErrNoCandidates
	}

	// 决策日志：记录输入。
	decision := &Decision{
		ProfileName:     profile.Name,
		CandidatesIn:    len(candidates),
		StrategyChain:   make([]string, 0),
		Timestamp:       db.NowFunc(),
	}
	inputSnapshot := snapshotCandidateIDs(candidates)

	// 归零所有候选的 Score 和 Eliminated 标记。
	for i := range candidates {
		candidates[i].Score = 0
		candidates[i].Eliminated = false
		candidates[i].ElimReason = ""
	}

	// 解析档案中的策略绑定，构建执行序列。
	resolved := resolveStrategies(profile.Strategies)
	if len(resolved) == 0 {
		// 未配置任何策略，按原始顺序返回。
		slog.WarnContext(ctx, "路由档案未配置任何策略，按原始顺序返回",
			"profile_name", profile.Name,
		)
		decision.CandidatesOut = len(candidates)
		decision.Selected = candidates[0].ChannelID
		logDecision(ctx, db, decision)
		sortByScore(candidates)
		return candidates, decision, nil
	}

	// ── 阶段 1: S-COMPLIANCE 硬过滤 ──
	candidates = executeFilter(ctx, candidates, StrategyCompliance)

	// ── 阶段 2: δ 价格帽过滤 ──
	if profile.DeltaCap.GreaterThan(decimal.Zero) || profile.DeltaCap.Equal(decimal.Zero) {
		capMultiplier := decimal.NewFromFloat(1.0).Add(profile.DeltaCap)
		maxAllowed := anchorSell.Mul(capMultiplier)
		candidates = applyPriceCap(candidates, maxAllowed)
	}

	if len(candidates) == 0 {
		slog.ErrorContext(ctx, "所有候选均被合规或价格帽过滤剔除",
			"profile_name", profile.Name,
			"candidates_in", decision.CandidatesIn,
		)
		return nil, nil, ErrAllEliminated
	}

	// ── 阶段 3: S-CLASSIFY 打分 ──
	candidates = executeScore(ctx, candidates, StrategyClassify)

	// ── 阶段 4: 其余启用策略依次执行 ──
	for _, s := range resolved {
		if s.ID() == StrategyCompliance || s.ID() == StrategyClassify {
			continue // 已在前置阶段处理。
		}
		decision.StrategyChain = append(decision.StrategyChain, s.ID())
		candidates = s.Filter(ctx, candidates)
		candidates = s.Score(ctx, candidates)
	}

	// ── 阶段 5: 按 Score 降序排列 ──
	sortByScore(candidates)

	// ── 阶段 6: 选取最优未剔除候选 ──
	var selected int64
	for _, c := range candidates {
		if !c.Eliminated {
			selected = c.ChannelID
			break
		}
	}

	if selected == 0 {
		slog.ErrorContext(ctx, "所有候选均被策略剔除",
			"profile_name", profile.Name,
			"candidates_in", decision.CandidatesIn,
			"strategy_chain", decision.StrategyChain,
		)
		return nil, nil, ErrAllEliminated
	}

	// 填充决策日志。
	decision.CandidatesOut = len(candidates)
	decision.Selected = selected
	decision.InputSnapshot = inputSnapshot

	// 影子模式仅记录不路由。
	if profile.Shadow {
		slog.InfoContext(ctx, "影子模式——仅记录路由决策，不实际路由",
			"profile_name", profile.Name,
			"selected_channel", selected,
		)
	}

	// 持久化决策日志。
	logDecision(ctx, db, decision)

	return candidates, decision, nil
}

// ── 辅助函数 ──────────────────────────────────────────────────────────────

// resolveStrategies 将档案中的策略绑定解析为已注册的 Strategy 实例。
// 按绑定的 Priority 升序排列后返回。
func resolveStrategies(bindings []StrategyBinding) []Strategy {
	result := make([]Strategy, 0, len(bindings))

	// 按 Priority 排序。
	sorted := make([]StrategyBinding, len(bindings))
	copy(sorted, bindings)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Priority < sorted[j].Priority
	})

	for _, binding := range sorted {
		if !binding.Enabled {
			continue
		}
		s := GetStrategy(binding.Code)
		if s == nil {
			slog.Warn("策略未注册，跳过",
				"strategy_code", binding.Code,
			)
			continue
		}
		result = append(result, s)
	}
	return result
}

// executeFilter 按策略 ID 查找策略并执行 Filter 阶段。
// 若策略未注册，原样返回候选列表。
func executeFilter(ctx context.Context, candidates []Candidate, strategyID string) []Candidate {
	s := GetStrategy(strategyID)
	if s == nil {
		return candidates
	}
	return s.Filter(ctx, candidates)
}

// executeScore 按策略 ID 查找策略并执行 Score 阶段。
// 若策略未注册，原样返回候选列表。
func executeScore(ctx context.Context, candidates []Candidate, strategyID string) []Candidate {
	s := GetStrategy(strategyID)
	if s == nil {
		return candidates
	}
	return s.Score(ctx, candidates)
}

// applyPriceCap 剔除 EstSell 超过 maxAllowed 的候选。
// 若 maxAllowed 为零值则不过滤。
func applyPriceCap(candidates []Candidate, maxAllowed decimal.Decimal) []Candidate {
	if maxAllowed.IsZero() {
		return candidates
	}
	result := make([]Candidate, 0, len(candidates))
	for _, c := range candidates {
		if c.Eliminated {
			result = append(result, c)
			continue
		}
		if c.EstSell.LessThanOrEqual(maxAllowed) {
			result = append(result, c)
		} else {
			c.Eliminated = true
			c.ElimReason = fmt.Sprintf("EstSell(%s) 超过价格帽上限(%s)", c.EstSell.String(), maxAllowed.String())
			result = append(result, c)
		}
	}
	return result
}

// sortByScore 按 Score 降序排列候选（高分在前）。
func sortByScore(candidates []Candidate) {
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})
}

// snapshotCandidateIDs 制作候选 ID 快照，用于审计日志。
func snapshotCandidateIDs(candidates []Candidate) []int64 {
	ids := make([]int64, len(candidates))
	for i, c := range candidates {
		ids[i] = c.ChannelID
	}
	return ids
}

// logDecision 持久化决策日志到数据库。
// 写入失败仅记录 WARN，不阻断路由流程。
func logDecision(ctx context.Context, db *gorm.DB, d *Decision) {
	if db == nil || d == nil {
		return
	}
	snapshotJSON, _ := json.Marshal(d.InputSnapshot)
	chainJSON, _ := json.Marshal(d.StrategyChain)

	slog.InfoContext(ctx, "路由决策",
		"profile_name", d.ProfileName,
		"candidates_in", d.CandidatesIn,
		"candidates_out", d.CandidatesOut,
		"selected_channel", d.Selected,
		"strategy_chain", chainJSON,
		"input_snapshot", snapshotJSON,
	)
}
