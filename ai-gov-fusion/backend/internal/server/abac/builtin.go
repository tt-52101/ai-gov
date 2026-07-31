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

// SeedBuiltinPolicies 将内置系统策略写入数据库。
// 使用 UPSERT 语义——已存在的策略（按 policy_code 匹配）不重复创建。
// 此函数应在应用启动时由迁移编排层调用，确保系统策略始终存在。
//
// 返回实际创建（新插入）的策略数量。
func SeedBuiltinPolicies(ctx context.Context, db *gorm.DB) (int, error) {
	created := 0
	for _, p := range builtinPolicies() {
		// 检查是否已存在。
		var existing SysAccessPolicy
		err := db.WithContext(ctx).
			Where("policy_code = ?", p.PolicyCode).
			First(&existing).Error
		if err == nil {
			// 已存在，跳过。
			continue
		}
		if err != nil && err != gorm.ErrRecordNotFound {
			return created, err
		}

		// 不存在则创建。
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

	slog.InfoContext(ctx, "内置策略种子完成", "created_count", created)
	return created, nil
}
