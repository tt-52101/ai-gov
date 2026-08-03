package abac

import (
	"context"
	"log/slog"

	"gorm.io/gorm"
)

// ── 内置角色编码常量 ──────────────────────────────────────────────────

const (
	// RoleAdminCode 超级管理员角色编码——拥有所有操作的完整权限。
	RoleAdminCode = "super_admin"
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

		p.ID = NewID()
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

		roleID := NewID()
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
			ID:          NewID(),
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

// ── 内置操作目录（原子操作编码） ──────────────────────────────────────────

// builtinActionCatalogs 定义所有系统内置的原子操作编码。
// 按四轴（data/fund/iam/routing）分类，覆盖所有治理 API 端点。
// 对应 PRD §7.2.4 四轴正交设计。
func builtinActionCatalogs() []SysActionCatalog {
	return []SysActionCatalog{
		// ── IAM 身份轴 ──
		{ActionCode: "iam.party.create", ActionName: "创建主体", Axis: AxisIAM, ResourceType: "party"},
		{ActionCode: "iam.party.write", ActionName: "更新主体", Axis: AxisIAM, ResourceType: "party"},
		{ActionCode: "iam.key.read", ActionName: "查看密钥", Axis: AxisIAM, ResourceType: "key"},
		{ActionCode: "iam.key.create", ActionName: "创建密钥", Axis: AxisIAM, ResourceType: "key"},
		{ActionCode: "iam.key.delete", ActionName: "删除密钥", Axis: AxisIAM, ResourceType: "key"},
		{ActionCode: "iam.role.read", ActionName: "查看角色", Axis: AxisIAM, ResourceType: "role"},
		{ActionCode: "iam.role.write", ActionName: "管理角色", Axis: AxisIAM, ResourceType: "role"},
		{ActionCode: "iam.policy.read", ActionName: "查看策略", Axis: AxisIAM, ResourceType: "policy"},
		{ActionCode: "iam.policy.write", ActionName: "管理策略", Axis: AxisIAM, ResourceType: "policy"},
		{ActionCode: "iam.member.write", ActionName: "管理成员", Axis: AxisIAM, ResourceType: "party_member"},
		{ActionCode: "iam.member.delete", ActionName: "删除成员", Axis: AxisIAM, ResourceType: "party_member"},
		{ActionCode: "iam.ui.read", ActionName: "查看UI配置", Axis: AxisIAM, ResourceType: "ui"},
		{ActionCode: "iam.ui.write", ActionName: "管理UI配置", Axis: AxisIAM, ResourceType: "ui"},

		// ── Data 数据轴 ──
		{ActionCode: "data.party.read", ActionName: "查询主体", Axis: AxisData, ResourceType: "party"},
		{ActionCode: "data.member.read", ActionName: "查看成员", Axis: AxisData, ResourceType: "party_member"},
		{ActionCode: "data.ui.read", ActionName: "查看UI权限", Axis: AxisData, ResourceType: "ui"},
		{ActionCode: "data.audit.read", ActionName: "查看审计", Axis: AxisData, ResourceType: "audit"},
		{ActionCode: "data.audit.write", ActionName: "记录审计", Axis: AxisData, ResourceType: "audit"},
		{ActionCode: "data.usage.read", ActionName: "查看用量", Axis: AxisData, ResourceType: "usage"},
		{ActionCode: "data.report.read", ActionName: "查看报表", Axis: AxisData, ResourceType: "report"},

		// ── Fund 资金轴 ──
		{ActionCode: "fund.balance.read", ActionName: "查看余额", Axis: AxisFund, ResourceType: "account"},
		{ActionCode: "fund.balance.write", ActionName: "资金操作", Axis: AxisFund, ResourceType: "account"},
		{ActionCode: "fund.ledger.read", ActionName: "查看流水", Axis: AxisFund, ResourceType: "account"},

		// ── Routing 路由轴 ──
		{ActionCode: "routing.price.read", ActionName: "查看价目", Axis: AxisRouting, ResourceType: "model_price"},
		{ActionCode: "routing.price.write", ActionName: "管理价目", Axis: AxisRouting, ResourceType: "model_price"},
		{ActionCode: "routing.model_grant.read", ActionName: "查看模型授权", Axis: AxisRouting, ResourceType: "model_grant"},
		{ActionCode: "routing.model_grant.write", ActionName: "管理模型授权", Axis: AxisRouting, ResourceType: "model_grant"},
		{ActionCode: "routing.route_profile.read", ActionName: "查看路由档案", Axis: AxisRouting, ResourceType: "route_profile"},
		{ActionCode: "routing.route_profile.write", ActionName: "管理路由档案", Axis: AxisRouting, ResourceType: "route_profile"},
	}
}

// SeedActionCatalogs 将内置操作目录种子写入 sys_action_catalogs 表。
// 使用 UPSERT 语义（按 action_code 去重），确保幂等。
// 返回实际新插入的操作编码数量。
//
// 此函数应在应用启动时由迁移编排层调用，在 SeedBuiltinPolicies 之前执行，
// 因为策略评估引擎（Engine.Evaluate）依赖操作目录查找 action 对应的治理轴。
func SeedActionCatalogs(ctx context.Context, db *gorm.DB) (int, error) {
	created := 0

	for _, a := range builtinActionCatalogs() {
		var existing SysActionCatalog
		err := db.WithContext(ctx).
			Where("action_code = ?", a.ActionCode).
			First(&existing).Error
		if err == nil {
			continue
		}
		if err != nil && err != gorm.ErrRecordNotFound {
			return created, err
		}

		a.ID = NewID()
		if err := db.WithContext(ctx).Create(&a).Error; err != nil {
			slog.ErrorContext(ctx, "种子操作目录失败",
				"action_code", a.ActionCode,
				"error", err,
			)
			return created, err
		}
		created++
	}

	slog.InfoContext(ctx, "操作目录种子完成", "action_created", created)
	return created, nil
}

// SeedAdminRoleAndPermissions 创建超级管理员角色（sys_roles）并绑定所有操作权限。
//
// 执行步骤：
//  1. 查询或创建 role_code = "super_admin" 的系统角色
//  2. 查询所有已注册的操作目录记录
//  3. 将每个操作与超级管理员角色绑定（sys_role_permissions）
//
// 此函数应在 SeedActionCatalogs 之后执行，确保所有操作编码已注册。
// 返回实际创建的角色数 + 权限绑定数。
func SeedAdminRoleAndPermissions(ctx context.Context, db *gorm.DB) (int, error) {
	created := 0

	// 步骤 1：查询或创建超级管理员角色。
	roleID := ""
	var existing SysRole
	err := db.WithContext(ctx).
		Where("role_code = ?", RoleAdminCode).
		First(&existing).Error
	if err == nil {
		roleID = existing.ID
	} else if err != nil && err != gorm.ErrRecordNotFound {
		return created, err
	} else {
		roleID = NewID()
		role := &SysRole{
			ID:          roleID,
			RoleCode:    RoleAdminCode,
			RoleName:    "超级管理员",
			Description: "平台超级管理员——拥有所有治理操作的完整权限。PRD §7.2.5。",
			IsSystem:    true,
		}
		if err := db.WithContext(ctx).Create(role).Error; err != nil {
			slog.ErrorContext(ctx, "种子超级管理员角色失败", "error", err)
			return created, err
		}
		created++
		slog.InfoContext(ctx, "超级管理员角色已创建", "role_id", roleID, "role_code", RoleAdminCode)
	}

	// 步骤 2：查询所有已注册的操作目录。
	var actions []SysActionCatalog
	if err := db.WithContext(ctx).Find(&actions).Error; err != nil {
		return created, err
	}

	if len(actions) == 0 {
		slog.WarnContext(ctx, "无可绑定的操作目录——请先调用 SeedActionCatalogs")
		return created, nil
	}

	// 步骤 3：将每个操作与超级管理员角色绑定（幂等插入）。
	permissionCreated := 0
	for _, a := range actions {
		rp := &SysRolePermission{
			ID:       NewID(),
			RoleID:   roleID,
			ActionID: a.ID,
		}
		if err := db.WithContext(ctx).Create(rp).Error; err != nil {
			if !isUniqueViolation(err) {
				slog.ErrorContext(ctx, "绑定管理员权限失败",
					"role_id", roleID,
					"action_code", a.ActionCode,
					"error", err,
				)
				return created, err
			}
			// 唯一约束冲突——已存在，跳过。
			continue
		}
		permissionCreated++
	}
	created += permissionCreated

	slog.InfoContext(ctx, "超级管理员权限绑定完成",
		"role_code", RoleAdminCode,
		"permission_bound", permissionCreated,
		"total_created", created,
	)
	return created, nil
}
