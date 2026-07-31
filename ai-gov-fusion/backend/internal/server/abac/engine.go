package abac

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"gorm.io/gorm"
)

// ── 预定义错误 ──────────────────────────────────────────────────────────

// ErrAccessDenied 表示 ABAC 引擎拒绝访问。
// 调用方应检查 errors.Is(err, ErrAccessDenied) 来判断是否为权限拒绝。
var ErrAccessDenied = errors.New("访问被拒绝")

// ErrActionNotFound 表示指定的操作编码在 sys_action_catalogs 中不存在。
var ErrActionNotFound = errors.New("操作编码未注册")

// ── Engine ──────────────────────────────────────────────────────────────

// Engine 是 ABAC 策略评估引擎。
// 所有控制面 API 和数据面敏感操作的访问判定均由 Engine 统一处理。
//
// 评估顺序（PRD §7.2.3）：
//  1. 提取主体属性（角色、成员关系）
//  2. 查找操作对应的轴线
//  3. 加载适用策略（按优先级降序）
//  4. 评估 deny 策略（任一匹配立即拒绝）
//  5. 评估 allow 策略（匹配则放行）
//  6. 评估角色权限（通过角色绑定获得的权限）
//  7. 默认拒绝（最小权限原则，A-CON-02）
type Engine struct {
	// DB 为 GORM 数据库句柄（可参与事务）。
	DB *gorm.DB
}

// NewEngine 创建新的 ABAC 引擎实例。
// 调用方负责确保数据库连接有效且在调用前已完成迁移。
func NewEngine(db *gorm.DB) *Engine {
	return &Engine{DB: db}
}

// Evaluate 评估主体对指定资源的操作权限。
//
// 评估顺序：deny 策略优先 → allow 策略 → 角色绑定 → 无匹配则拒绝（默认拒绝）。
// 返回 nil 表示允许访问；返回 error 表示拒绝，error 中包含拒绝原因。
func (e *Engine) Evaluate(ctx context.Context, subject Subject, action string, resource Resource) error {
	// 步骤 1：查找操作所属的治理轴。
	actionAxis, err := e.lookupActionAxis(ctx, action)
	if err != nil {
		return err
	}

	// 步骤 2：解析主体绑定的所有角色 ID。
	roleIDs, err := e.resolveSubjectRoles(ctx, subject)
	if err != nil {
		return fmt.Errorf("解析主体角色失败: %w", err)
	}

	// 步骤 3：收集所有适用于当前主体的策略（直接绑定 + 角色绑定）。
	policies, err := e.collectApplicablePolicies(ctx, subject, roleIDs)
	if err != nil {
		return fmt.Errorf("收集适用策略失败: %w", err)
	}

	// 步骤 4：按优先级降序评估 deny 策略。
	for _, p := range policies {
		if p.Effect != EffectDeny {
			continue
		}
		if e.matchPolicyConditions(p.ConditionsJSON, actionAxis, action, resource) {
			slog.WarnContext(ctx, "策略评估拒绝",
				"subject_type", subject.Type,
				"subject_id", subject.ID,
				"action", action,
				"axis", actionAxis,
				"resource_type", resource.Type,
				"policy_code", p.PolicyCode,
				"policy_effect", p.Effect,
			)
			return fmt.Errorf("%w: %s", ErrAccessDenied, p.PolicyCode)
		}
	}

	// 步骤 5：按优先级降序评估 allow 策略。
	for _, p := range policies {
		if p.Effect != EffectAllow {
			continue
		}
		if e.matchPolicyConditions(p.ConditionsJSON, actionAxis, action, resource) {
			slog.InfoContext(ctx, "策略评估通过",
				"subject_type", subject.Type,
				"subject_id", subject.ID,
				"action", action,
				"axis", actionAxis,
				"resource_type", resource.Type,
				"policy_code", p.PolicyCode,
				"policy_effect", p.Effect,
			)
			return nil
		}
	}

	// 步骤 6：评估角色权限。
	if len(roleIDs) > 0 {
		if e.checkRolePermission(ctx, roleIDs, action) {
			slog.InfoContext(ctx, "角色权限评估通过",
				"subject_type", subject.Type,
				"subject_id", subject.ID,
				"action", action,
				"axis", actionAxis,
			)
			return nil
		}
	}

	// 步骤 7：默认拒绝。
	slog.WarnContext(ctx, "默认拒绝访问",
		"subject_type", subject.Type,
		"subject_id", subject.ID,
		"action", action,
		"axis", actionAxis,
		"resource_type", resource.Type,
	)
	return fmt.Errorf("%w: 无匹配策略或权限", ErrAccessDenied)
}

// GetPermissions 返回主体在指定资源类型上的所有允许动作列表。
// 用于 UI 投影——前端根据返回的动作列表决定菜单可见性和按钮显隐。
//
// 返回值是去重后的 action_code 列表（字符串排序）。
func (e *Engine) GetPermissions(ctx context.Context, subject Subject, resourceType string) ([]string, error) {
	// 解析主体角色。
	roleIDs, err := e.resolveSubjectRoles(ctx, subject)
	if err != nil {
		return nil, fmt.Errorf("解析主体角色失败: %w", err)
	}

	actionSet := make(map[string]bool)

	// 来源 1：角色权限中的操作。
	if len(roleIDs) > 0 {
		actions, err := e.loadRoleActions(ctx, roleIDs)
		if err != nil {
			return nil, fmt.Errorf("加载角色操作失败: %w", err)
		}
		for _, a := range actions {
			actionSet[a] = true
		}
	}

	// 来源 2：通过 allow 策略获得的操作（提取策略条件中指定的 actions）。
	policyActions, err := e.loadPolicyAllowedActions(ctx, subject, roleIDs, resourceType)
	if err != nil {
		return nil, fmt.Errorf("加载策略允许操作失败: %w", err)
	}
	for _, a := range policyActions {
		actionSet[a] = true
	}

	result := make([]string, 0, len(actionSet))
	for a := range actionSet {
		result = append(result, a)
	}
	// 使用标准库排序保持输出确定性。
	sortStrings(result)

	slog.InfoContext(ctx, "获取主体权限列表",
		"subject_type", subject.Type,
		"subject_id", subject.ID,
		"resource_type", resourceType,
		"permission_count", len(result),
	)
	return result, nil
}

// ── 内部辅助方法 ───────────────────────────────────────────────────────

// lookupActionAxis 在 sys_action_catalogs 中查找 action 对应的治理轴。
func (e *Engine) lookupActionAxis(ctx context.Context, action string) (string, error) {
	var catalog SysActionCatalog
	err := e.DB.WithContext(ctx).
		Where("action_code = ?", action).
		First(&catalog).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", fmt.Errorf("%w: %s", ErrActionNotFound, action)
		}
		return "", fmt.Errorf("查询操作目录失败: %w", err)
	}
	return catalog.Axis, nil
}

// resolveSubjectRoles 解析主体拥有的所有角色 ID（在有效期内，且考虑 scope）。
func (e *Engine) resolveSubjectRoles(ctx context.Context, subject Subject) ([]string, error) {
	now := time.Now()
	var bindings []SysSubjectRoleBinding

	query := e.DB.WithContext(ctx).
		Where("subject_type = ? AND subject_id = ?", subject.Type, subject.ID).
		Where("(valid_from IS NULL OR valid_from <= ?)", now).
		Where("(valid_until IS NULL OR valid_until >= ?)", now)

	if err := query.Find(&bindings).Error; err != nil {
		return nil, err
	}

	roleIDs := make([]string, 0, len(bindings))
	seen := make(map[string]bool)
	for _, b := range bindings {
		if !seen[b.RoleID] {
			roleIDs = append(roleIDs, b.RoleID)
			seen[b.RoleID] = true
		}
	}
	return roleIDs, nil
}

// collectApplicablePolicies 收集适用于当前主体的所有策略：
//   - 直接绑定到该主体的策略（sys_access_policy_bindings.subject_type = subject.Type, subject_id = subject.ID）
//   - 绑定到主体所拥有角色的策略（subject_type = 'role', subject_id IN roleIDs）
//
// 策略按 priority 降序排列。
func (e *Engine) collectApplicablePolicies(ctx context.Context, subject Subject, roleIDs []string) ([]SysAccessPolicy, error) {
	// 收集所有匹配的 policy_id。
	var bindingRows []SysAccessPolicyBinding

	// 直接绑定。
	directQuery := e.DB.WithContext(ctx).
		Where("subject_type = ? AND subject_id = ?", subject.Type, subject.ID)

	// 通过角色绑定。
	roleQuery := e.DB.WithContext(ctx)
	if len(roleIDs) > 0 {
		roleQuery = roleQuery.Where("subject_type = ? AND subject_id IN ?", SubjectTypeRole, roleIDs)
	} else {
		// 无角色时用永假条件。
		roleQuery = roleQuery.Where("1 = 0")
	}

	// 使用 UNION（GORM 不支持直接 UNION，分两次查询再合并）。
	if err := directQuery.Find(&bindingRows).Error; err != nil {
		return nil, err
	}

	var roleBindingRows []SysAccessPolicyBinding
	if len(roleIDs) > 0 {
		if err := e.DB.WithContext(ctx).
			Where("subject_type = ? AND subject_id IN ?", SubjectTypeRole, roleIDs).
			Find(&roleBindingRows).Error; err != nil {
			return nil, err
		}
	}

	// 合并 policy_id 并去重。
	policyIDSet := make(map[string]bool)
	for _, r := range bindingRows {
		policyIDSet[r.PolicyID] = true
	}
	for _, r := range roleBindingRows {
		policyIDSet[r.PolicyID] = true
	}

	if len(policyIDSet) == 0 {
		return nil, nil
	}

	policyIDs := make([]string, 0, len(policyIDSet))
	for id := range policyIDSet {
		policyIDs = append(policyIDs, id)
	}

	// 加载策略，按 priority 降序排列。
	var policies []SysAccessPolicy
	if err := e.DB.WithContext(ctx).
		Where("id IN ?", policyIDs).
		Order("priority DESC").
		Find(&policies).Error; err != nil {
		return nil, err
	}

	return policies, nil
}

// matchPolicyConditions 判断策略的 conditions_json 是否匹配当前操作与资源。
//
// 条件字段说明：
//   - axis: 匹配操作所属的治理轴（字符串或数组）
//   - actions: 匹配的具体操作编码列表
//   - resource_type: 匹配的资源类型
//
// 条件字段为空表示不做该维度的限制。
func (e *Engine) matchPolicyConditions(conditionsJSON, actionAxis, action string, resource Resource) bool {
	if conditionsJSON == "" || conditionsJSON == "{}" {
		return true
	}

	var conds map[string]any
	if err := json.Unmarshal([]byte(conditionsJSON), &conds); err != nil {
		return false
	}

	// 检查 axis 条件。
	if v, ok := conds["axis"]; ok {
		if !matchAxis(v, actionAxis) {
			return false
		}
	}

	// 检查 actions 条件。
	if v, ok := conds["actions"]; ok {
		if !matchActions(v, action) {
			return false
		}
	}

	// 检查 resource_type 条件。
	if v, ok := conds["resource_type"]; ok {
		if rt, ok2 := v.(string); ok2 {
			if rt != "" && rt != resource.Type {
				return false
			}
		}
	}

	return true
}

// checkRolePermission 检查主体通过角色绑定是否拥有指定操作的权限。
func (e *Engine) checkRolePermission(ctx context.Context, roleIDs []string, action string) bool {
	if len(roleIDs) == 0 {
		return false
	}

	var count int64
	e.DB.WithContext(ctx).
		Table("sys_role_permissions rp").
		Joins("JOIN sys_action_catalogs ac ON ac.id = rp.action_id").
		Where("rp.role_id IN ?", roleIDs).
		Where("ac.action_code = ?", action).
		Count(&count)

	return count > 0
}

// loadRoleActions 加载角色拥有的所有操作编码。
func (e *Engine) loadRoleActions(ctx context.Context, roleIDs []string) ([]string, error) {
	type row struct {
		ActionCode string
	}
	var rows []row
	if err := e.DB.WithContext(ctx).
		Table("sys_role_permissions rp").
		Select("DISTINCT ac.action_code").
		Joins("JOIN sys_action_catalogs ac ON ac.id = rp.action_id").
		Where("rp.role_id IN ?", roleIDs).
		Find(&rows).Error; err != nil {
		return nil, err
	}

	result := make([]string, len(rows))
	for i, r := range rows {
		result[i] = r.ActionCode
	}
	return result, nil
}

// loadPolicyAllowedActions 加载通过 allow 策略获得的操作编码。
// 遍历 allow 策略的 conditions_json，提取其中声明的 actions 列表。
func (e *Engine) loadPolicyAllowedActions(ctx context.Context, subject Subject, roleIDs []string, resourceType string) ([]string, error) {
	policies, err := e.collectApplicablePolicies(ctx, subject, roleIDs)
	if err != nil {
		return nil, err
	}

	actionSet := make(map[string]bool)
	for _, p := range policies {
		if p.Effect != EffectAllow {
			continue
		}
		extracted := extractActionsFromConditions(p.ConditionsJSON, resourceType)
		for _, a := range extracted {
			actionSet[a] = true
		}
	}

	result := make([]string, 0, len(actionSet))
	for a := range actionSet {
		result = append(result, a)
	}
	return result, nil
}

// ── 条件匹配辅助函数 ──────────────────────────────────────────────────

// matchAxis 检查条件中的 axis 字段是否匹配实际轴。
// 支持字符串（精确匹配）和字符串数组（任一匹配）。
func matchAxis(cond any, actualAxis string) bool {
	switch v := cond.(type) {
	case string:
		return v == actualAxis
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok && s == actualAxis {
				return true
			}
		}
		return false
	}
	return false
}

// matchActions 检查条件中的 actions 数组是否包含当前操作编码。
func matchActions(cond any, actualAction string) bool {
	actions, ok := cond.([]any)
	if !ok {
		return false
	}
	for _, item := range actions {
		if s, ok2 := item.(string); ok2 {
			if s == actualAction {
				return true
			}
			// 支持通配符匹配，如 "fund.*" 匹配 "fund.allocate"。
			if strings.HasSuffix(s, ".*") {
				prefix := strings.TrimSuffix(s, ".*")
				if strings.HasPrefix(actualAction, prefix) {
					return true
				}
			}
		}
	}
	return false
}

// extractActionsFromConditions 从 conditions_json 中提取声明的 actions 列表。
// 用于 GetPermissions 时提取策略中明确授权的操作。
func extractActionsFromConditions(conditionsJSON, resourceType string) []string {
	if conditionsJSON == "" || conditionsJSON == "{}" {
		return nil
	}
	var conds map[string]any
	if err := json.Unmarshal([]byte(conditionsJSON), &conds); err != nil {
		return nil
	}

	// 如果指定了 resource_type 且不匹配，跳过。
	if v, ok := conds["resource_type"]; ok {
		if rt, ok2 := v.(string); ok2 && rt != "" && rt != resourceType {
			return nil
		}
	}

	v, ok := conds["actions"]
	if !ok {
		return nil
	}
	actions, ok2 := v.([]any)
	if !ok2 {
		return nil
	}
	var result []string
	for _, item := range actions {
		if s, ok3 := item.(string); ok3 {
			result = append(result, s)
		}
	}
	return result
}

// sortStrings 对字符串切片进行原地排序（简单插入排序，适合小数据集）。
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
