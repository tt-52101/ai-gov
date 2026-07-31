// Package abac 实现基于属性的访问控制（ABAC）策略引擎。
//
// ABAC 是本产品的统一授权引擎。所有控制面 API、数据面敏感操作、UI 可见性
// 均由 ABAC 引擎统一判定。RBAC（角色）是 ABAC 的子集——角色是主体属性
// 之一，不作为独立授权体系。
//
// 核心功能：
//   - 策略评估：deny 优先 → allow 策略 → 角色权限 → 默认拒绝
//   - 职责分离：四轴（data/fund/iam/routing）正交，禁止一轴推导另一轴
//   - 权限投影：GetPermissions 返回主体允许的动作列表，供 UI 层消费
//   - 内置策略：SeedBuiltinPolicies 预置 4 条职责分离系统策略（is_system=true）
//
// 数据模型（6 表）：
//   - sys_action_catalogs      原子操作目录（按四轴分类）
//   - sys_roles                角色定义（is_system 角色不可删除）
//   - sys_role_permissions     角色→操作 N:M 映射
//   - sys_subject_role_bindings 主体→角色绑定（含作用域与有效期）
//   - sys_access_policies      ABAC 策略（effect: allow/deny + conditions_json）
//   - sys_access_policy_bindings 策略→主体绑定
//
// 设计原则（PRD §7.2 + A-CON-01~05）：
//   - 最小权限默认：未显式授予即拒绝
//   - 四轴正交：data、fund、iam、routing 独立判定
//   - 职责分离：路由/资金互斥、审计只读、Leader 不万能
//   - 模型权限独立：ModelGrant deny 优先于 allow
package abac
