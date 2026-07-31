package abac

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"gorm.io/gorm"
)

// ── 预定义错误 ──────────────────────────────────────────────────────────

// ErrSystemPolicy 表示尝试修改或删除系统内置策略。
var ErrSystemPolicy = errors.New("系统内置策略不可删除或修改")

// ── 策略 CRUD ───────────────────────────────────────────────────────────

// CreatePolicy 创建新的 ABAC 策略。
// 策略编码必须唯一；effect 必须是 "allow" 或 "deny"。
// 返回创建后的策略记录（ID 已填充）。
func CreatePolicy(ctx context.Context, db *gorm.DB, p *SysAccessPolicy) error {
	if p == nil {
		return errors.New("abac: 不能创建 nil 策略")
	}
	if p.PolicyCode == "" {
		return errors.New("abac: 策略编码不能为空")
	}
	if p.PolicyName == "" {
		return errors.New("abac: 策略名称不能为空")
	}
	if p.Effect != EffectAllow && p.Effect != EffectDeny {
		return fmt.Errorf("abac: 无效的策略效果 %q，必须是 %q 或 %q", p.Effect, EffectAllow, EffectDeny)
	}
	if p.ConditionsJSON == "" {
		p.ConditionsJSON = "{}"
	}
	if p.ID == "" {
		p.ID = newID()
	}

	if err := db.WithContext(ctx).Create(p).Error; err != nil {
		slog.ErrorContext(ctx, "创建策略失败",
			"policy_code", p.PolicyCode,
			"effect", p.Effect,
			"error", err,
		)
		return fmt.Errorf("abac: 创建策略失败: %w", err)
	}

	slog.InfoContext(ctx, "创建策略成功",
		"policy_id", p.ID,
		"policy_code", p.PolicyCode,
		"effect", p.Effect,
		"priority", p.Priority,
	)
	return nil
}

// GetPolicy 按 ID 查询策略。
// 策略不存在时返回错误。
func GetPolicy(ctx context.Context, db *gorm.DB, id string) (*SysAccessPolicy, error) {
	var p SysAccessPolicy
	if err := db.WithContext(ctx).First(&p, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("abac: 策略 %s 不存在", id)
		}
		return nil, fmt.Errorf("abac: 查询策略失败: %w", err)
	}
	return &p, nil
}

// UpdatePolicy 更新策略的可变字段（名称、效果、条件、优先级、描述）。
// 系统策略（is_system=true）禁止修改。
func UpdatePolicy(ctx context.Context, db *gorm.DB, p *SysAccessPolicy) error {
	if p == nil || p.ID == "" {
		return errors.New("abac: 策略 ID 不能为空")
	}

	// 加载现有记录以检查 is_system。
	existing, err := GetPolicy(ctx, db, p.ID)
	if err != nil {
		return err
	}
	if existing.IsSystem {
		return fmt.Errorf("abac: %w: %s", ErrSystemPolicy, p.PolicyCode)
	}

	updates := map[string]any{
		"policy_name":     p.PolicyName,
		"description":     p.Description,
		"effect":          p.Effect,
		"conditions_json": p.ConditionsJSON,
		"priority":        p.Priority,
	}

	if err := db.WithContext(ctx).Model(&SysAccessPolicy{}).Where("id = ?", p.ID).Updates(updates).Error; err != nil {
		slog.ErrorContext(ctx, "更新策略失败", "policy_id", p.ID, "error", err)
		return fmt.Errorf("abac: 更新策略失败: %w", err)
	}

	slog.InfoContext(ctx, "更新策略成功", "policy_id", p.ID, "policy_code", existing.PolicyCode)
	return nil
}

// DeletePolicy 删除策略。
// 系统策略（is_system=true）禁止删除。
// 删除策略时会级联删除关联的绑定记录（数据库 ON DELETE CASCADE）。
func DeletePolicy(ctx context.Context, db *gorm.DB, id string) error {
	existing, err := GetPolicy(ctx, db, id)
	if err != nil {
		return err
	}
	if existing.IsSystem {
		return fmt.Errorf("abac: %w: %s", ErrSystemPolicy, existing.PolicyCode)
	}

	if err := db.WithContext(ctx).Delete(&SysAccessPolicy{}, "id = ?", id).Error; err != nil {
		slog.ErrorContext(ctx, "删除策略失败", "policy_id", id, "error", err)
		return fmt.Errorf("abac: 删除策略失败: %w", err)
	}

	slog.InfoContext(ctx, "删除策略成功", "policy_id", id, "policy_code", existing.PolicyCode)
	return nil
}

// ListPolicies 列出所有策略，可按效果（allow/deny）筛选。
// 传入空字符串返回所有策略，按 priority 降序排列。
func ListPolicies(ctx context.Context, db *gorm.DB, effect string) ([]SysAccessPolicy, error) {
	var policies []SysAccessPolicy
	q := db.WithContext(ctx).Order("priority DESC")
	if effect != "" {
		q = q.Where("effect = ?", effect)
	}
	if err := q.Find(&policies).Error; err != nil {
		return nil, fmt.Errorf("abac: 查询策略列表失败: %w", err)
	}
	return policies, nil
}

// ── 策略绑定 CRUD ──────────────────────────────────────────────────────

// BindPolicy 将策略绑定到指定主体。
// 一个策略可以绑定多个主体；一个主体可以被多个策略约束。
func BindPolicy(ctx context.Context, db *gorm.DB, policyID, subjectType, subjectID string) error {
	if policyID == "" || subjectType == "" || subjectID == "" {
		return errors.New("abac: policy_id、subject_type、subject_id 均不能为空")
	}

	binding := &SysAccessPolicyBinding{
		ID:          newID(),
		PolicyID:    policyID,
		SubjectType: subjectType,
		SubjectID:   subjectID,
	}

	if err := db.WithContext(ctx).Create(binding).Error; err != nil {
		slog.ErrorContext(ctx, "绑定策略失败",
			"policy_id", policyID,
			"subject_type", subjectType,
			"subject_id", subjectID,
			"error", err,
		)
		return fmt.Errorf("abac: 绑定策略失败: %w", err)
	}

	slog.InfoContext(ctx, "绑定策略成功",
		"binding_id", binding.ID,
		"policy_id", policyID,
		"subject_type", subjectType,
		"subject_id", subjectID,
	)
	return nil
}

// UnbindPolicy 解除策略与主体的绑定。
func UnbindPolicy(ctx context.Context, db *gorm.DB, bindingID string) error {
	result := db.WithContext(ctx).Delete(&SysAccessPolicyBinding{}, "id = ?", bindingID)
	if result.Error != nil {
		return fmt.Errorf("abac: 解除策略绑定失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("abac: 绑定 %s 不存在", bindingID)
	}

	slog.InfoContext(ctx, "解除策略绑定成功", "binding_id", bindingID)
	return nil
}

// ListPolicyBindings 列出指定策略的所有绑定记录。
func ListPolicyBindings(ctx context.Context, db *gorm.DB, policyID string) ([]SysAccessPolicyBinding, error) {
	var bindings []SysAccessPolicyBinding
	if err := db.WithContext(ctx).Where("policy_id = ?", policyID).Find(&bindings).Error; err != nil {
		return nil, fmt.Errorf("abac: 查询策略绑定失败: %w", err)
	}
	return bindings, nil
}

// ── 策略模拟评估 ──────────────────────────────────────────────────────

// EvalResult 策略模拟评估的汇总结果。
type EvalResult struct {
	// Allowed 表示最终是否允许访问。
	Allowed bool `json:"allowed"`
	// Reason 为允许或拒绝的原因。
	Reason string `json:"reason"`
	// MatchedDenyPolicies 为匹配的 deny 策略链。
	MatchedDenyPolicies []PolicyMatch `json:"matched_deny_policies,omitempty"`
	// MatchedAllowPolicies 为匹配的 allow 策略链。
	MatchedAllowPolicies []PolicyMatch `json:"matched_allow_policies,omitempty"`
	// MatchedRole 表示是否通过角色权限匹配。
	MatchedRole bool `json:"matched_role"`
}

// PolicyMatch 记录匹配到的单条策略信息。
type PolicyMatch struct {
	PolicyCode string `json:"policy_code"`
	PolicyName string `json:"policy_name"`
	Effect     string `json:"effect"`
	Priority   int    `json:"priority"`
}

// EvaluatePolicy 模拟策略评估——输入主体/动作/资源，输出评估结果和匹配的策略链。
// 此方法不修改任何数据，仅用于策略调试和管理员可视化验证。
// 评估逻辑与 Engine.Evaluate 完全一致。
func EvaluatePolicy(ctx context.Context, db *gorm.DB, subject Subject, action string, resource Resource) (*EvalResult, error) {
	engine := NewEngine(db)
	result := &EvalResult{}

	// 查找操作轴。
	actionAxis, err := engine.lookupActionAxis(ctx, action)
	if err != nil {
		result.Allowed = false
		result.Reason = err.Error()
		return result, nil
	}

	// 解析角色。
	roleIDs, err := engine.resolveSubjectRoles(ctx, subject)
	if err != nil {
		return nil, fmt.Errorf("abac: 解析角色失败: %w", err)
	}

	// 收集策略。
	policies, err := engine.collectApplicablePolicies(ctx, subject, roleIDs)
	if err != nil {
		return nil, fmt.Errorf("abac: 收集策略失败: %w", err)
	}

	// 评估 deny 策略。
	for _, p := range policies {
		if p.Effect != EffectDeny {
			continue
		}
		if engine.matchPolicyConditions(p.ConditionsJSON, actionAxis, action, resource) {
			result.MatchedDenyPolicies = append(result.MatchedDenyPolicies, PolicyMatch{
				PolicyCode: p.PolicyCode,
				PolicyName: p.PolicyName,
				Effect:     p.Effect,
				Priority:   p.Priority,
			})
		}
	}

	if len(result.MatchedDenyPolicies) > 0 {
		result.Allowed = false
		result.Reason = fmt.Sprintf("命中 %d 条 deny 策略", len(result.MatchedDenyPolicies))
		return result, nil
	}

	// 评估 allow 策略。
	for _, p := range policies {
		if p.Effect != EffectAllow {
			continue
		}
		if engine.matchPolicyConditions(p.ConditionsJSON, actionAxis, action, resource) {
			result.MatchedAllowPolicies = append(result.MatchedAllowPolicies, PolicyMatch{
				PolicyCode: p.PolicyCode,
				PolicyName: p.PolicyName,
				Effect:     p.Effect,
				Priority:   p.Priority,
			})
		}
	}

	if len(result.MatchedAllowPolicies) > 0 {
		result.Allowed = true
		result.Reason = fmt.Sprintf("命中 %d 条 allow 策略", len(result.MatchedAllowPolicies))
		return result, nil
	}

	// 评估角色权限。
	if len(roleIDs) > 0 && engine.checkRolePermission(ctx, roleIDs, action) {
		result.Allowed = true
		result.MatchedRole = true
		result.Reason = "通过角色权限匹配"
		return result, nil
	}

	// 默认拒绝。
	result.Allowed = false
	result.Reason = "默认拒绝：无匹配策略或角色权限"
	return result, nil
}

// ── 策略条件辅助 ──────────────────────────────────────────────────────

// MarshalConditions 将条件映射序列化为 JSON 字符串。
// 用于在创建/更新策略时格式化 conditions_json 字段。
func MarshalConditions(conds map[string]any) (string, error) {
	b, err := json.Marshal(conds)
	if err != nil {
		return "", fmt.Errorf("abac: 序列化策略条件失败: %w", err)
	}
	return string(b), nil
}

// UnmarshalConditions 将 JSON 字符串反序列化为条件映射。
func UnmarshalConditions(jsonStr string) (map[string]any, error) {
	var conds map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &conds); err != nil {
		return nil, fmt.Errorf("abac: 反序列化策略条件失败: %w", err)
	}
	return conds, nil
}
