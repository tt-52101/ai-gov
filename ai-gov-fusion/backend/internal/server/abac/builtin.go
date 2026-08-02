package abac

import (
	"context"
	"log/slog"

	"gorm.io/gorm"
)

// ── 内置策略编码常量 ────────────────────────────────────────────────────

const (
	// PolicySODFund 职责分离：资金轴管理员不可操作身份与路由轴。
	PolicySODFund = "P-SOD-FUND"
	// PolicySODRouting 职责分离：路由轴管理员不可操作资金轴。
	PolicySODRouting = "P-SOD-ROUTING"
	// PolicySODIAM 职责分离：身份轴管理员不可操作路由轴。
	PolicySODIAM = "P-SOD-IAM"
	// PolicyAuditReadonly 审计只读：审计角色仅允许读取操作，拒绝所有写入。
	PolicyAuditReadonly = "P-AUDIT-READONLY"
)

// ── 内置 SOD 角色编码常量 ───────────────────────────────────────────────

const (
	// RoleSODFundGuard 职责分离守卫角色：资金轴管理员。
	// 绑定 P-SOD-FUND 策略，禁止访问 iam/routing 轴。
	RoleSODFundGuard = "SOD-FUND-GUARD"
	// RoleSODRoutingGuard 职责分离守卫角色：路由轴管理员。
	// 绑定 P-SOD-ROUTING 策略，禁止访问 fund 轴。
	RoleSODRoutingGuard = "SOD-ROUTING-GUARD"
	// RoleSODIAMGuard 职责分离守卫角色：身份轴管理员。
	// 绑定 P-SOD-IAM 策略，禁止访问 routing 轴。
	RoleSODIAMGuard = "SOD-IAM-GUARD"
	// RoleSODAuditor 职责分离守卫角色：审计员。
	// 绑定 P-AUDIT-READONLY 策略，默认拒绝所有操作。
	RoleSODAuditor = "SOD-AUDITOR"
)

// sodRolePolicyBindings 定义 SOD 角色到策略的绑定映射。
// 每个系统角色绑定一条对应的 deny 策略，实现跨轴互斥。
var sodRolePolicyBindings = map[string]string{
	RoleSODFundGuard:    PolicySODFund,
	RoleSODRoutingGuard: PolicySODRouting,
	RoleSODIAMGuard:     PolicySODIAM,
	RoleSODAuditor:      PolicyAuditReadonly,
}

// sodRoleDescriptions 定义 SOD 系统角色的中文描述。
var sodRoleDescriptions = map[string]string{
	RoleSODFundGuard:    "资金轴管理员守卫角色——绑定 P-SOD-FUND 策略，禁止访问 iam 和 routing 轴。PRD §7.2.5。",
	RoleSODRoutingGuard: "路由轴管理员守卫角色——绑定 P-SOD-ROUTING 策略，禁止访问 fund 轴。PRD §7.2.5。",
	RoleSODIAMGuard:     "身份轴管理员守卫角色——绑定 P-SOD-IAM 策略，禁止访问 routing 轴。PRD §7.2.5。",
	RoleSODAuditor:      "审计员守卫角色——绑定 P-AUDIT-READONLY 策略，默认拒绝所有操作，仅允许显式授予的读取权限。PRD §7.2.5。",
}

// ── 内置策略优先级 ──────────────────────────────────────────────────────

const builtinPriority = 1000

// ── 内置策略定义 ────────────────────────────────────────────────────────

// builtinPolicies 定义所有系统内置的职责分离策略。
// 这些策略的 is_system=true，不可删除。
// 策略需要在部署时绑定到对应的管理员角色才能生效。
//
// 策略 1（PRD §7.2.5 路由-资金分离）：拥有 fund 轴操作权限的角色，
// 被禁止操作 iam 和 routing 轴。
//
// 策略 2（PRD §7.2.5 路由-资金分离）：拥有 routing 轴操作权限的角色，
// 被禁止操作 fund 轴。
//
// 策略 3（PRD §7.2.5）：拥有 iam 轴操作权限的角色，
// 被禁止操作 routing 轴。
//
// 策略 4（PRD §7.2.5 审计只读）：拥有审计角色的主体，
// 被拒绝所有操作（角色权限仅授予读取类操作，此为兜底保障）。
func builtinPolicies() []SysAccessPolicy {
	return []SysAccessPolicy{
		{
			PolicyCode:     PolicySODFund,
			PolicyName:     "职责分离：资金管理员不可操作身份与路由",
			Effect:         EffectDeny,
			ConditionsJSON: `{"axis":["iam","routing"]}`,
			Priority:       builtinPriority,
			IsSystem:       true,
			Description:    "拥有 fund 轴写权限的角色不得执行 iam 或 routing 轴的任何操作。PRD §7.2.5 路由-资金分离。",
		},
		{
			PolicyCode:     PolicySODRouting,
			PolicyName:     "职责分离：路由管理员不可操作资金",
			Effect:         EffectDeny,
			ConditionsJSON: `{"axis":"fund"}`,
			Priority:       builtinPriority,
			IsSystem:       true,
			Description:    "拥有 routing 轴写权限的角色不得执行 fund 轴的任何操作。PRD §7.2.5 路由-资金分离。",
		},
		{
			PolicyCode:     PolicySODIAM,
			PolicyName:     "职责分离：身份管理员不可操作路由",
			Effect:         EffectDeny,
			ConditionsJSON: `{"axis":"routing"}`,
			Priority:       builtinPriority,
			IsSystem:       true,
			Description:    "拥有 iam 轴写权限的角色不得执行 routing 轴的任何操作。PRD §7.2.5。",
		},
		{
			PolicyCode:     PolicyAuditReadonly,
			PolicyName:     "审计只读：审计角色仅允许读取",
			Effect:         EffectDeny,
			ConditionsJSON: `{}`,
			Priority:       builtinPriority,
			IsSystem:       true,
			Description:    "审计角色默认拒绝所有操作——仅通过角色权限显式授予的读取操作允许执行。PRD §7.2.5 审计只读。",
		},
	}
}

// SeedBuiltinPolicies 将内置系统策略写入数据库，并创建对应的 SOD 系统角色
// 及策略绑定，实现 PRD §7.2.5 要求的跨轴职责分离。
//
// 执行步骤：
//  1. 种子写入 4 条内置 deny 策略（UPSERT 语义，按 policy_code 去重）
//  2. 种子写入 4 个 SOD 系统角色（is_system=true，按 role_code 去重）
//  3. 将每条 SOD 策略绑定到对应的系统角色（按 (policy_id, subject_type, subject_id) 去重）
//
// 角色到策略的绑定关系：
//   - SOD-FUND-GUARD    → P-SOD-FUND（禁止 fund 管理员操作 iam/routing 轴）
//   - SOD-ROUTING-GUARD → P-SOD-ROUTING（禁止 routing 管理员操作 fund 轴）
//   - SOD-IAM-GUARD     → P-SOD-IAM（禁止 iam 管理员操作 routing 轴）
//   - SOD-AUDITOR       → P-AUDIT-READONLY（审计角色默认拒绝所有操作）
//
// 管理员需要将具体用户通过 sys_subject_role_bindings 分配到对应的 SOD 角色
// 后，策略才会实际生效。
//
// 此函数应在应用启动时由迁移编排层调用，确保系统策略和角色绑定始终存在。
// 返回实际创建（新插入）的实体数量（策略数 + 角色数 + 绑定数）。
func SeedBuiltinPolicies(ctx context.Context, db *gorm.DB) (int, error) {
	created := 0

	// 步骤 1：种子写入内置策略。
	for _, p := range builtinPolicies() {
		var existing SysAccessPolicy
		err := db.WithContext(ctx).
			Where("policy_code = ?", p.PolicyCode).
			First(&existing).Error
		if err == nil {
			continue
		}
		if err != nil && err != gorm.ErrRecordNotFound {
			return created, err
		}

		p.ID = newID()
		if err := db.WithContext(ctx).Create(&p).Error; err != nil {
			slog.ErrorContext(ctx, "种子内置策略失败",
				"policy_code", p.PolicyCode,
				"error", err,
			)
			return created, err
		}
		created++
	}

	slog.InfoContext(ctx, "内置策略种子完成", "policy_created", created)

	// 步骤 2：种子写入 SOD 系统角色。
	roleCreated := 0
	roleIDs := make(map[string]string) // role_code → role_id
	for roleCode, desc := range sodRoleDescriptions {
		var existing SysRole
		err := db.WithContext(ctx).
			Where("role_code = ?", roleCode).
			First(&existing).Error
		if err == nil {
			roleIDs[roleCode] = existing.ID
			continue
		}
		if err != nil && err != gorm.ErrRecordNotFound {
			return created, err
		}

		roleID := newID()
		r := &SysRole{
			ID:          roleID,
			RoleCode:    roleCode,
			RoleName:    desc,
			Description: desc,
			IsSystem:    true,
		}
		if err := db.WithContext(ctx).Create(r).Error; err != nil {
			slog.ErrorContext(ctx, "种子SOD角色失败",
				"role_code", roleCode,
				"error", err,
			)
			return created, err
		}
		roleIDs[roleCode] = roleID
		roleCreated++
		created++
	}

	slog.InfoContext(ctx, "SOD系统角色种子完成", "role_created", roleCreated)

	// 步骤 3：将每条 SOD 策略绑定到对应的系统角色。
	bindingCreated := 0
	for roleCode, policyCode := range sodRolePolicyBindings {
		roleID, ok := roleIDs[roleCode]
		if !ok {
			// 角色 ID 在上一步可能未获取到（出错的角色），跳过。
			continue
		}

		// 查询策略 ID。
		var policy SysAccessPolicy
		if err := db.WithContext(ctx).
			Where("policy_code = ?", policyCode).
			First(&policy).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				slog.WarnContext(ctx, "SOD策略未找到，跳过绑定",
					"policy_code", policyCode,
					"role_code", roleCode,
				)
				continue
			}
			return created, err
		}

		// 检查绑定是否已存在（幂等）。
		var existingBinding SysAccessPolicyBinding
		err := db.WithContext(ctx).
			Where("policy_id = ? AND subject_type = ? AND subject_id = ?",
				policy.ID, SubjectTypeRole, roleID).
			First(&existingBinding).Error
		if err == nil {
			continue
		}
		if err != nil && err != gorm.ErrRecordNotFound {
			return created, err
		}

		binding := &SysAccessPolicyBinding{
			ID:          newID(),
			PolicyID:    policy.ID,
			SubjectType: SubjectTypeRole,
			SubjectID:   roleID,
		}
		if err := db.WithContext(ctx).Create(binding).Error; err != nil {
			slog.ErrorContext(ctx, "SOD策略绑定失败",
				"policy_code", policyCode,
				"role_code", roleCode,
				"error", err,
			)
			return created, err
		}
		bindingCreated++
		created++
	}

	slog.InfoContext(ctx, "SOD策略绑定完成",
		"binding_created", bindingCreated,
		"total_created", created,
	)
	return created, nil
}
