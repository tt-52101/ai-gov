/*
 Navicat Premium Dump SQL

 Source Server         : 192.168.51.236_5432
 Source Server Type    : PostgreSQL
 Source Server Version : 160013 (160013)
 Source Host           : 192.168.51.236:5432
 Source Catalog        : ai_governance
 Source Schema         : ai_governance

 Target Server Type    : PostgreSQL
 Target Server Version : 160013 (160013)
 File Encoding         : 65001

 Date: 31/07/2026 03:41:00
*/


-- ----------------------------
-- Table structure for approval_decisions
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."approval_decisions";
CREATE TABLE "ai_governance"."approval_decisions" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "approval_request_id" uuid NOT NULL,
  "approver_subject_id" uuid NOT NULL,
  "approver_role_code" varchar(64) COLLATE "pg_catalog"."default" NOT NULL,
  "decision" varchar(16) COLLATE "pg_catalog"."default" NOT NULL,
  "reason" varchar(1024) COLLATE "pg_catalog"."default" NOT NULL,
  "payload_hash" varchar(71) COLLATE "pg_catalog"."default" NOT NULL,
  "decided_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "created_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "operation_trace_id" varchar(128) COLLATE "pg_catalog"."default"
)
;
COMMENT ON COLUMN "ai_governance"."approval_decisions"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."approval_decisions"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."approval_decisions"."approval_request_id" IS '关联 ApprovalRequest；敏感动作须校验同 Tenant、APPROVED、未过期且 payload_hash 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."approval_decisions"."approver_subject_id" IS '逻辑引用 审批人 IdentitySubject 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."approval_decisions"."approver_role_code" IS '审批决定发生时审批人使用的有效角色代码；用于证明审批资格而非动态重算历史。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."approval_decisions"."decision" IS 'Approval Module 受控枚举；允许值：APPROVE（审批通过）、REJECT（审批拒绝）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."approval_decisions"."reason" IS '审批通过或拒绝的人工理由；必须能够解释决定并与 payload_hash、审批角色一同审计。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."approval_decisions"."payload_hash" IS '审批或事件载荷的 SHA-256 摘要；审批后载荷变化必须重新申请。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."approval_decisions"."decided_at" IS 'Approval Module UTC 业务时间 decided_at；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."approval_decisions"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."approval_decisions"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."approval_decisions"."created_by" IS '创建记录的稳定主体标识；保存 ID 或服务主体，不保存仅供展示的姓名。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."approval_decisions"."operation_trace_id" IS '产生该次写入的 Trace 标识；用于跨 Module 联合排障和审计关联。 字段约束：可空。';
COMMENT ON TABLE "ai_governance"."approval_decisions" IS '不可覆盖的审批决定事实。';

-- ----------------------------
-- Table structure for approval_requests
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."approval_requests";
CREATE TABLE "ai_governance"."approval_requests" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "request_code" varchar(64) COLLATE "pg_catalog"."default" NOT NULL,
  "action_code" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "resource_type" varchar(64) COLLATE "pg_catalog"."default" NOT NULL,
  "resource_id" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "scope_type" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "scope_id" uuid NOT NULL,
  "payload_hash" varchar(71) COLLATE "pg_catalog"."default" NOT NULL,
  "requester_subject_id" uuid NOT NULL,
  "required_approver_role" varchar(64) COLLATE "pg_catalog"."default" NOT NULL,
  "required_decision_count" int2 NOT NULL DEFAULT 1,
  "status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "expires_at" timestamptz(6) NOT NULL,
  "decided_at" timestamptz(6),
  "row_version" int8 NOT NULL DEFAULT 1,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "created_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "updated_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "operation_trace_id" varchar(128) COLLATE "pg_catalog"."default"
)
;
COMMENT ON COLUMN "ai_governance"."approval_requests"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."approval_requests"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."approval_requests"."request_code" IS '面向管理和审计的唯一审批单号；不得用数据库主键或展示标题替代。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."approval_requests"."action_code" IS '本次审批授权的精确 resource:verb 动作；必须存在于 sys_action_catalogs 且不允许通配符。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."approval_requests"."resource_type" IS '待审批资源的稳定类型代码；与 action_code、resource_id 和 Scope 共同限定审批对象。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."approval_requests"."resource_id" IS '待审批资源的不透明稳定标识；Approval Module 必须验证资源存在、同 Tenant 且 payload_hash 未变化。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."approval_requests"."scope_type" IS 'Approval Module 受控枚举；允许值：TENANT（租户作用域）、ORGANIZATION（组织作用域）、PROJECT（项目作用域）、USER（企业用户主体）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."approval_requests"."scope_id" IS '逻辑引用 GovernanceScopeRef 指向的领域实体 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."approval_requests"."payload_hash" IS '审批或事件载荷的 SHA-256 摘要；审批后载荷变化必须重新申请。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."approval_requests"."requester_subject_id" IS '逻辑引用 审批申请 IdentitySubject 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."approval_requests"."required_approver_role" IS '满足该审批所需的稳定审批角色代码；审批时按当前有效角色和 Scope 重新校验。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."approval_requests"."required_decision_count" IS 'Approval Module 顺序、计数或重放控制字段 required_decision_count；用于确定性排序、幂等、投影或完整性验证，禁止无审计跳变。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."approval_requests"."status" IS 'Approval Module 受控枚举；允许值：DRAFT（草稿未生效）、PENDING（待处理或待生效）、APPROVED（已批准，尚需发布或生效）、REJECTED（检查结果被业务规则判定拒绝）、CANCELLED（已取消）、EXPIRED（已过有效期）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."approval_requests"."expires_at" IS 'Approval Module UTC 业务时间 expires_at；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."approval_requests"."decided_at" IS 'Approval Module UTC 业务时间 decided_at；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."approval_requests"."row_version" IS '乐观锁版本；每次成功 UPDATE 严格递增，用于防止并发覆盖。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."approval_requests"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."approval_requests"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."approval_requests"."created_by" IS '创建记录的稳定主体标识；保存 ID 或服务主体，不保存仅供展示的姓名。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."approval_requests"."updated_by" IS '最后修改记录的稳定主体标识；必须与 AuditEvent.actor 可关联。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."approval_requests"."operation_trace_id" IS '产生该次写入的 Trace 标识；用于跨 Module 联合排障和审计关联。 字段约束：可空。';
COMMENT ON TABLE "ai_governance"."approval_requests" IS '敏感变更的审批请求及 payload hash 绑定。';

-- ----------------------------
-- Table structure for audit_chain_anchors
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."audit_chain_anchors";
CREATE TABLE "ai_governance"."audit_chain_anchors" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "anchor_id" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "chain_id" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "first_sequence" int8 NOT NULL,
  "last_sequence" int8 NOT NULL,
  "chain_head_hash" varchar(71) COLLATE "pg_catalog"."default" NOT NULL,
  "signature_algorithm" varchar(32) COLLATE "pg_catalog"."default" NOT NULL,
  "signature_value" varchar(4096) COLLATE "pg_catalog"."default" NOT NULL,
  "key_ref" varchar(1024) COLLATE "pg_catalog"."default" NOT NULL,
  "immutable_object_ref" varchar(2048) COLLATE "pg_catalog"."default" NOT NULL,
  "anchored_at" timestamptz(6) NOT NULL,
  "verification_status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp()
)
;
COMMENT ON COLUMN "ai_governance"."audit_chain_anchors"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."audit_chain_anchors"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."audit_chain_anchors"."anchor_id" IS '一次外部审计锚定批次的唯一业务标识；用于定位链范围、签名和不可变对象。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."audit_chain_anchors"."chain_id" IS '被锚定的审计链标识；必须与 first_sequence、last_sequence 和 chain_head_hash 属于同一链。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."audit_chain_anchors"."first_sequence" IS 'Audit Integrity 顺序、计数或重放控制字段 first_sequence；用于确定性排序、幂等、投影或完整性验证，禁止无审计跳变。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."audit_chain_anchors"."last_sequence" IS 'Audit Integrity 顺序、计数或重放控制字段 last_sequence；用于确定性排序、幂等、投影或完整性验证，禁止无审计跳变。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."audit_chain_anchors"."chain_head_hash" IS 'Audit Integrity 完整性摘要 chain_head_hash；保存算法前缀和摘要，用于幂等、篡改检测或重算，禁止敏感原文。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."audit_chain_anchors"."signature_algorithm" IS '链头签名算法和参数版本，例如批准的企业签名套件；验证方按该值选择公钥与算法。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."audit_chain_anchors"."signature_value" IS '对链头与范围元数据生成的签名编码；用于证明完整性，不包含审计事件正文。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."audit_chain_anchors"."key_ref" IS 'Audit Integrity 外部安全引用 key_ref；仅保存 Vault/KMS/对象定位符，不保存 Secret、凭证或对象正文。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."audit_chain_anchors"."immutable_object_ref" IS 'Audit Integrity 外部安全引用 immutable_object_ref；仅保存 Vault/KMS/对象定位符，不保存 Secret、凭证或对象正文。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."audit_chain_anchors"."anchored_at" IS 'Audit Integrity UTC 业务时间 anchored_at；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."audit_chain_anchors"."verification_status" IS 'Audit Integrity 受控枚举；允许值：PENDING（待处理或待生效）、VERIFIED（锚点签名与链头校验通过）、FAILED（执行失败）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."audit_chain_anchors"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."audit_chain_anchors"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON TABLE "ai_governance"."audit_chain_anchors" IS '审计哈希链的外部不可变锚点与签名证明。';

-- ----------------------------
-- Table structure for audit_events
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."audit_events";
CREATE TABLE "ai_governance"."audit_events" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "event_id" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "event_type" varchar(64) COLLATE "pg_catalog"."default" NOT NULL,
  "actor_type" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "actor_id" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "subject_id" uuid,
  "action_code" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "resource_type" varchar(64) COLLATE "pg_catalog"."default" NOT NULL,
  "resource_id" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "scope_type" varchar(24) COLLATE "pg_catalog"."default",
  "scope_id" uuid,
  "authorization_decision" varchar(16) COLLATE "pg_catalog"."default",
  "authorization_reason_codes" varchar(64)[] COLLATE "pg_catalog"."default" NOT NULL DEFAULT ARRAY[]::character varying[],
  "authorization_revision" int8,
  "policy_revision" int8,
  "request_id" uuid,
  "trace_id" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "external_context_json" jsonb NOT NULL DEFAULT '{}'::jsonb,
  "before_hash" varchar(71) COLLATE "pg_catalog"."default",
  "after_hash" varchar(71) COLLATE "pg_catalog"."default",
  "approval_request_id" uuid,
  "source_ip" inet,
  "user_agent_hash" varchar(71) COLLATE "pg_catalog"."default",
  "occurred_at" timestamptz(6) NOT NULL,
  "chain_id" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "chain_sequence" int8 NOT NULL,
  "previous_event_hash" varchar(71) COLLATE "pg_catalog"."default",
  "event_hash" varchar(71) COLLATE "pg_catalog"."default" NOT NULL,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "subject_identity_revision" int8
)
;
COMMENT ON COLUMN "ai_governance"."audit_events"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."audit_events"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."audit_events"."event_id" IS '审计事件的全局幂等标识；重复投递不得产生第二条链节点或改变 chain_sequence。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."audit_events"."event_type" IS '审计事件类型代码；标识身份、授权、调用、安全、资金、配置或管理事件类别。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."audit_events"."actor_type" IS 'Audit Facts 受控枚举；允许值：USER（企业用户主体）、ADMIN（治理管理员发起）、SERVICE（服务主体）、SYSTEM（平台系统任务发起）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."audit_events"."actor_id" IS '执行被审计动作的稳定主体标识；结合 actor_type 区分用户、管理员、服务和系统任务。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."audit_events"."subject_id" IS '逻辑引用 IdentitySubject 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."audit_events"."action_code" IS '被审计的精确 resource:verb 动作；必须与实际后端 PEP 判定使用的动作一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."audit_events"."resource_type" IS '被操作资源的稳定类型代码；与 action_code 和 resource_id 共同支持精确审计检索。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."audit_events"."resource_id" IS '被操作资源的不透明稳定标识；资源删除或归档后仍保持可追溯，不保存展示名称替代。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."audit_events"."scope_type" IS 'Audit Facts 受控枚举；允许值：TENANT（租户作用域）、ORGANIZATION（组织作用域）、PROJECT（项目作用域）、USER（企业用户主体）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."audit_events"."scope_id" IS '逻辑引用 GovernanceScopeRef 指向的领域实体 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."audit_events"."authorization_decision" IS 'Audit Facts 受控枚举；允许值：ALLOW（允许）、DENY（拒绝，优先于允许）、NOT_APPLICABLE（该事件不适用授权决策）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."audit_events"."authorization_reason_codes" IS '本次授权 ALLOW/DENY 的稳定原因码集合；NOT_APPLICABLE 事件可为空数组。 字段约束：必填，默认 ARRAY[]::varchar[]。';
COMMENT ON COLUMN "ai_governance"."audit_events"."authorization_revision" IS '授权 RuntimeSnapshot revision；用于重放当次动作决策。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."audit_events"."policy_revision" IS '策略决策时锁定的不可变 revision；用于历史重放、差异分析和审计证明，旧 revision 不得覆盖新决策。字段约束：可空，仅不适用策略的事件允许为空。';
COMMENT ON COLUMN "ai_governance"."audit_events"."request_id" IS '逻辑引用 UsageRequest 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."audit_events"."trace_id" IS '端到端 Trace 标识；贯通 Request、Attempt、Usage、Ledger、安全和 Audit。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."audit_events"."external_context_json" IS 'Audit Facts JSONB 证据或扩展快照 external_context_json；不承载核心可检索关系，值必须为 object。 字段约束：必填，默认 ''{}''::jsonb。';
COMMENT ON COLUMN "ai_governance"."audit_events"."before_hash" IS 'Audit Facts 完整性摘要 before_hash；保存算法前缀和摘要，用于幂等、篡改检测或重算，禁止敏感原文。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."audit_events"."after_hash" IS 'Audit Facts 完整性摘要 after_hash；保存算法前缀和摘要，用于幂等、篡改检测或重算，禁止敏感原文。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."audit_events"."approval_request_id" IS '关联 ApprovalRequest；敏感动作须校验同 Tenant、APPROVED、未过期且 payload_hash 一致。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."audit_events"."source_ip" IS '管理或调用请求的来源 IP；仅用于安全审计并按隐私保留策略处理，不作为授权唯一依据。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."audit_events"."user_agent_hash" IS 'Audit Facts 完整性摘要 user_agent_hash；保存算法前缀和摘要，用于幂等、篡改检测或重算，禁止敏感原文。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."audit_events"."occurred_at" IS 'Audit Facts UTC 业务时间 occurred_at；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."audit_events"."chain_id" IS '审计哈希链分区标识，通常按 Tenant 和审计域划分；同一链 sequence 必须连续且不可回退。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."audit_events"."chain_sequence" IS 'Audit Facts 顺序、计数或重放控制字段 chain_sequence；用于确定性排序、幂等、投影或完整性验证，禁止无审计跳变。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."audit_events"."previous_event_hash" IS 'Audit Facts 完整性摘要 previous_event_hash；保存算法前缀和摘要，用于幂等、篡改检测或重算，禁止敏感原文。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."audit_events"."event_hash" IS 'Audit Facts 完整性摘要 event_hash；保存算法前缀和摘要，用于幂等、篡改检测或重算，禁止敏感原文。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."audit_events"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."audit_events"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."audit_events"."subject_identity_revision" IS '审计动作涉及 Subject 时的身份 revision；Subject 为空的系统事件允许为空。';
COMMENT ON TABLE "ai_governance"."audit_events" IS '统一追加式审计事实；同时承载授权决策、调用、安全、资金和管理证据。';

-- ----------------------------
-- Table structure for authentication_challenges
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."authentication_challenges";
CREATE TABLE "ai_governance"."authentication_challenges" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "challenge_id" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "challenge_type" varchar(32) COLLATE "pg_catalog"."default" NOT NULL,
  "subject_id" uuid,
  "contact_point_id" uuid,
  "oauth_provider_id" uuid,
  "challenge_digest" varchar(512) COLLATE "pg_catalog"."default" NOT NULL,
  "client_context_hash" varchar(71) COLLATE "pg_catalog"."default" NOT NULL,
  "attempt_count" int4 NOT NULL DEFAULT 0,
  "maximum_attempts" int4 NOT NULL,
  "status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "expires_at" timestamptz(6) NOT NULL,
  "verified_at" timestamptz(6),
  "consumed_at" timestamptz(6),
  "row_version" int8 NOT NULL DEFAULT 1,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "created_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "updated_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "operation_source" varchar(32) COLLATE "pg_catalog"."default" NOT NULL DEFAULT 'AUTH'::character varying,
  "operation_trace_id" varchar(128) COLLATE "pg_catalog"."default",
  "is_deleted" bool NOT NULL DEFAULT false,
  "deleted_at" timestamptz(6),
  "deleted_by" varchar(128) COLLATE "pg_catalog"."default",
  "delete_reason" varchar(512) COLLATE "pg_catalog"."default"
)
;
COMMENT ON COLUMN "ai_governance"."authentication_challenges"."id" IS '应用生成的 UUIDv7 挑战事实主键；不作为发给客户端的挑战秘密。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."authentication_challenges"."tenant_id" IS '挑战所属 Tenant；发送、验证、限速和消费必须保持同 Tenant。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."authentication_challenges"."challenge_id" IS '对外不透明挑战标识；Tenant 内唯一且不能推导验证码、state 或数据库主键。';
COMMENT ON COLUMN "ai_governance"."authentication_challenges"."challenge_type" IS '挑战类型；EMAIL_VERIFY、PHONE_VERIFY、EMAIL_LOGIN、SMS_LOGIN、PASSWORD_RESET、OAUTH_STATE 或 OAUTH_BIND。';
COMMENT ON COLUMN "ai_governance"."authentication_challenges"."subject_id" IS '已知用户的 Subject 逻辑引用；注册前或防账户枚举场景允许为空。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."authentication_challenges"."contact_point_id" IS '邮箱或手机号联系点逻辑引用；OAuth 挑战可为空。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."authentication_challenges"."oauth_provider_id" IS 'OAuth Provider 逻辑引用；非 OAuth 挑战为空。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."authentication_challenges"."challenge_digest" IS '验证码、OAuth state/nonce 或恢复 Token 的带密钥摘要；原文不得落库。';
COMMENT ON COLUMN "ai_governance"."authentication_challenges"."client_context_hash" IS '发起挑战时 Tenant、IP、设备、重定向和客户端上下文的稳定摘要，用于防重放。';
COMMENT ON COLUMN "ai_governance"."authentication_challenges"."attempt_count" IS '当前挑战已验证次数；从零开始且不得超过 maximum_attempts。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."authentication_challenges"."maximum_attempts" IS '挑战最大验证次数，范围 1 到 20；达到上限后进入 FAILED。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."authentication_challenges"."status" IS '状态；PENDING 待验证，VERIFIED 已证明，CONSUMED 已使用，EXPIRED 过期，FAILED 失败，CANCELLED 取消。';
COMMENT ON COLUMN "ai_governance"."authentication_challenges"."expires_at" IS '挑战硬失效时间；必须晚于创建时间，失效后不得恢复或重放。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."authentication_challenges"."verified_at" IS '挑战成功验证时间；仅 VERIFIED 状态必填，消费后保留。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."authentication_challenges"."consumed_at" IS '挑战被注册、登录、重置或绑定命令消费的时间；仅 CONSUMED 状态必填。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."authentication_challenges"."row_version" IS 'GORM 乐观锁版本；保证一次性消费和失败次数并发正确。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."authentication_challenges"."created_at" IS '挑战首次创建时间；数据库生成并作为有效期起点。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."authentication_challenges"."updated_at" IS '挑战状态最近更新时间；由触发器统一刷新。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."authentication_challenges"."created_by" IS '发起挑战的匿名客户端摘要、Subject 或认证服务标识；禁止存敏感原文。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."authentication_challenges"."updated_by" IS '最近更新挑战的认证服务或 Subject 标识；必须可追溯。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."authentication_challenges"."operation_source" IS '挑战操作来源，默认 AUTH；不作为认证决定唯一证据。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."authentication_challenges"."operation_trace_id" IS '挑战 Trace ID；贯通发送 Adapter、验证、会话和审计。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."authentication_challenges"."is_deleted" IS '挑战软删除标志；安全保留期内禁止删除，过期清理仍需审计授权。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."authentication_challenges"."deleted_at" IS '挑战软删除时间；未删除时为空且禁止物理删除。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."authentication_challenges"."deleted_by" IS '执行挑战软删除的服务或管理员标识；未删除时为空。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."authentication_challenges"."delete_reason" IS '挑战软删除原因；仅保留策略清理允许且必须可审计。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON TABLE "ai_governance"."authentication_challenges" IS '邮箱、手机号、密码恢复及 OAuth state 的短时单次认证挑战；摘要化保存并防重放。';

-- ----------------------------
-- Table structure for authentication_events
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."authentication_events";
CREATE TABLE "ai_governance"."authentication_events" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "event_id" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "subject_id" uuid,
  "login_identity_id" uuid,
  "session_id" uuid,
  "event_type" varchar(40) COLLATE "pg_catalog"."default" NOT NULL,
  "auth_method" varchar(32) COLLATE "pg_catalog"."default",
  "result" varchar(16) COLLATE "pg_catalog"."default" NOT NULL,
  "reason_code" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "risk_level" varchar(16) COLLATE "pg_catalog"."default" NOT NULL,
  "source_ip" inet,
  "device_fingerprint_hash" varchar(71) COLLATE "pg_catalog"."default",
  "user_agent_hash" varchar(71) COLLATE "pg_catalog"."default",
  "trace_id" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "occurred_at" timestamptz(6) NOT NULL,
  "evidence_hash" varchar(71) COLLATE "pg_catalog"."default" NOT NULL,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp()
)
;
COMMENT ON COLUMN "ai_governance"."authentication_events"."id" IS '应用生成的 UUIDv7 认证事件主键；事件只追加且不得更新或删除。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."authentication_events"."tenant_id" IS '认证事件所属 Tenant；链路查询和隔离必须先限定该值。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."authentication_events"."event_id" IS '认证域业务事件标识；Tenant 内唯一并用于幂等写入。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."authentication_events"."subject_id" IS '已解析用户的 Subject 逻辑引用；匿名失败或防账户枚举事件允许为空。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."authentication_events"."login_identity_id" IS '涉及的登录标识逻辑引用；注册前未知或泛化风险事件允许为空。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."authentication_events"."session_id" IS '涉及的 user_sessions.id 逻辑引用；未创建会话的失败事件为空。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."authentication_events"."event_type" IS '认证事件类型；REGISTRATION 注册，LOGIN_SUCCEEDED 登录成功，LOGIN_FAILED 登录失败，LOGOUT 登出，TOKEN_REFRESH 刷新，SESSION_REVOKED 会话撤销，ACCOUNT_LOCKED 账户锁定，PASSWORD_CHANGED 修改密码，PASSWORD_RESET 重置密码，CONTACT_VERIFIED 联系方式验证，OAUTH_BOUND 绑定 OAuth，OAUTH_UNBOUND 解绑 OAuth。';
COMMENT ON COLUMN "ai_governance"."authentication_events"."auth_method" IS '事件认证方式；ENTERPRISE_SSO 企业联邦，PASSWORD 密码，EMAIL_OTP 邮箱验证码，SMS_OTP 短信验证码，WECHAT_OAUTH 微信，OIDC_OAUTH 标准 OIDC；未选择方式时为空。';
COMMENT ON COLUMN "ai_governance"."authentication_events"."result" IS '事件结果；SUCCESS 表示操作完成，FAILURE 表示拒绝或失败，不代表业务模型调用结果。';
COMMENT ON COLUMN "ai_governance"."authentication_events"."reason_code" IS '稳定机器原因码；禁止包含手机号、邮箱、密码、验证码、Token 或 Provider 原始声明。';
COMMENT ON COLUMN "ai_governance"."authentication_events"."risk_level" IS '风险级别；LOW 低，MEDIUM 中，HIGH 高，CRITICAL 严重，用于告警和处置。';
COMMENT ON COLUMN "ai_governance"."authentication_events"."source_ip" IS '认证来源 IP；空值仅允许内部离线处置事件。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."authentication_events"."device_fingerprint_hash" IS '可选设备指纹摘要；用于关联风险而不保存设备敏感原文。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."authentication_events"."user_agent_hash" IS 'User-Agent 脱敏哈希；原文不进入认证事件。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."authentication_events"."trace_id" IS '贯通认证入口、Adapter、Subject Resolver、会话和 AuditEvent 的 Trace ID。';
COMMENT ON COLUMN "ai_governance"."authentication_events"."occurred_at" IS '认证事实实际发生的 UTC 时间；不同于数据库落库时间。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."authentication_events"."evidence_hash" IS '认证决定输入与结果的规范化证据 SHA-256；敏感原文不进入事件。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."authentication_events"."created_at" IS '认证事件追加写入时间；数据库生成且事件不可更新。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."authentication_events"."updated_at" IS '标准事实字段；追加后保持创建值，任何 UPDATE 由触发器拒绝。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON TABLE "ai_governance"."authentication_events" IS '认证安全追加事实；覆盖注册、登录、失败、锁定、密码、联系方式、OAuth 和会话变更。';

-- ----------------------------
-- Table structure for balance_projections
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."balance_projections";
CREATE TABLE "ai_governance"."balance_projections" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "account_id" uuid NOT NULL,
  "current_balance_credit" int8 NOT NULL,
  "current_reserved_credit" int8 NOT NULL DEFAULT 0,
  "last_ledger_leg_id" uuid NOT NULL,
  "source_watermark" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "generation" int8 NOT NULL,
  "projection_status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp()
)
;
COMMENT ON COLUMN "ai_governance"."balance_projections"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."balance_projections"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."balance_projections"."account_id" IS '逻辑引用 FundingAccount 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."balance_projections"."current_balance_credit" IS '由 LedgerLeg 重放得到的规范化整数 AI Credit 余额；仅为投影，不是写入资金事实。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."balance_projections"."current_reserved_credit" IS '由活跃 Reservation 对应 Leg 聚合的整数 AI Credit 预占；不得人工直接调整。 字段约束：必填，默认 0。';
COMMENT ON COLUMN "ai_governance"."balance_projections"."last_ledger_leg_id" IS '逻辑引用 投影已消费的最后 LedgerLeg 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."balance_projections"."source_watermark" IS '投影已消费的权威事实水位；用于重建、追赶和新 generation 切换。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."balance_projections"."generation" IS '投影或 RuntimeSnapshot 的重建代次；新 generation 完成全量校验后原子切换，旧 generation 不得覆盖或与新代混读。字段约束：必填且大于 0。';
COMMENT ON COLUMN "ai_governance"."balance_projections"."projection_status" IS 'Ledger Projection 受控枚举；允许值：BUILDING（构建中）、CURRENT（投影与事实水位一致）、STALE（投影已落后）、FAILED（执行失败）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."balance_projections"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."balance_projections"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON TABLE "ai_governance"."balance_projections" IS '由 LedgerLeg 重放得到的账户余额投影；不是可写资金事实。';

-- ----------------------------
-- Table structure for closing_snapshots
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."closing_snapshots";
CREATE TABLE "ai_governance"."closing_snapshots" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "closing_code" varchar(96) COLLATE "pg_catalog"."default" NOT NULL,
  "reconciliation_run_id" uuid NOT NULL,
  "window_start" timestamptz(6) NOT NULL,
  "window_end" timestamptz(6) NOT NULL,
  "fact_watermark" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "company_balance_credit" int8 NOT NULL,
  "company_consumed_credit" int8 NOT NULL,
  "scope_summary_json" jsonb NOT NULL,
  "allocation_summary_json" jsonb NOT NULL,
  "key_summary_json" jsonb NOT NULL,
  "reservation_summary_json" jsonb NOT NULL,
  "emergency_credit_summary_json" jsonb NOT NULL,
  "difference_summary_json" jsonb NOT NULL,
  "unresolved_difference_count" int8 NOT NULL DEFAULT 0,
  "checksum_sha256" varchar(71) COLLATE "pg_catalog"."default" NOT NULL,
  "audit_chain_head_hash" varchar(71) COLLATE "pg_catalog"."default" NOT NULL,
  "closed_at" timestamptz(6) NOT NULL,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp()
)
;
COMMENT ON COLUMN "ai_governance"."closing_snapshots"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."closing_snapshots"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."closing_snapshots"."closing_code" IS '正式日结快照的唯一业务编号；与窗口、watermark 和 checksum 共同标识关账证据。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."closing_snapshots"."reconciliation_run_id" IS '逻辑引用 ReconciliationRun 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."closing_snapshots"."window_start" IS 'Reconciliation UTC 业务时间 window_start；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."closing_snapshots"."window_end" IS 'Reconciliation UTC 业务时间 window_end；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."closing_snapshots"."fact_watermark" IS 'Reconciliation 顺序、计数或重放控制字段 fact_watermark；用于确定性排序、幂等、投影或完整性验证，禁止无审计跳变。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."closing_snapshots"."company_balance_credit" IS 'Reconciliation 整数 AI Credit 字段 company_balance_credit；P0 不表达人民币，变化必须来自 Ledger 或其可重建投影。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."closing_snapshots"."company_consumed_credit" IS 'Reconciliation 整数 AI Credit 字段 company_consumed_credit；P0 不表达人民币，变化必须来自 Ledger 或其可重建投影。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."closing_snapshots"."scope_summary_json" IS 'Reconciliation JSONB 证据或扩展快照 scope_summary_json；不承载核心可检索关系，值必须为 object。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."closing_snapshots"."allocation_summary_json" IS 'Reconciliation JSONB 证据或扩展快照 allocation_summary_json；不承载核心可检索关系，值必须为 object。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."closing_snapshots"."key_summary_json" IS 'Reconciliation JSONB 证据或扩展快照 key_summary_json；不承载核心可检索关系，值必须为 object。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."closing_snapshots"."reservation_summary_json" IS 'Reconciliation JSONB 证据或扩展快照 reservation_summary_json；不承载核心可检索关系，值必须为 object。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."closing_snapshots"."emergency_credit_summary_json" IS 'Reconciliation 整数 AI Credit 字段 emergency_credit_summary_json；P0 不表达人民币，变化必须来自 Ledger 或其可重建投影。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."closing_snapshots"."difference_summary_json" IS 'Reconciliation JSONB 证据或扩展快照 difference_summary_json；不承载核心可检索关系，值必须为 object。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."closing_snapshots"."unresolved_difference_count" IS 'Reconciliation 顺序、计数或重放控制字段 unresolved_difference_count；用于确定性排序、幂等、投影或完整性验证，禁止无审计跳变。 字段约束：必填，默认 0。';
COMMENT ON COLUMN "ai_governance"."closing_snapshots"."checksum_sha256" IS 'Reconciliation 完整性摘要 checksum_sha256；保存算法前缀和摘要，用于幂等、篡改检测或重算，禁止敏感原文。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."closing_snapshots"."audit_chain_head_hash" IS 'Reconciliation 完整性摘要 audit_chain_head_hash；保存算法前缀和摘要，用于幂等、篡改检测或重算，禁止敏感原文。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."closing_snapshots"."closed_at" IS 'Reconciliation UTC 业务时间 closed_at；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."closing_snapshots"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."closing_snapshots"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON TABLE "ai_governance"."closing_snapshots" IS '无未解决差异时生成的不可变日结快照。';

-- ----------------------------
-- Table structure for credit_rate_versions
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."credit_rate_versions";
CREATE TABLE "ai_governance"."credit_rate_versions" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "rate_code" varchar(64) COLLATE "pg_catalog"."default" NOT NULL,
  "model_id" uuid NOT NULL,
  "model_version" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "input_credit_per_million_tokens" int8 NOT NULL,
  "output_credit_per_million_tokens" int8 NOT NULL,
  "rounding_mode" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "minimum_charge_credit" int8 NOT NULL DEFAULT 0,
  "valid_from" timestamptz(6) NOT NULL,
  "valid_until" timestamptz(6),
  "status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "revision" int8 NOT NULL DEFAULT 1,
  "row_version" int8 NOT NULL DEFAULT 1,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "created_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "updated_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "operation_source" varchar(32) COLLATE "pg_catalog"."default" NOT NULL DEFAULT 'ADMIN'::character varying,
  "operation_trace_id" varchar(128) COLLATE "pg_catalog"."default",
  "is_deleted" bool NOT NULL DEFAULT false,
  "deleted_at" timestamptz(6),
  "deleted_by" varchar(128) COLLATE "pg_catalog"."default",
  "delete_reason" varchar(512) COLLATE "pg_catalog"."default"
)
;
COMMENT ON COLUMN "ai_governance"."credit_rate_versions"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."credit_rate_versions"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."credit_rate_versions"."rate_code" IS 'Token 到 AI Credit 换算方案的稳定代码；与 revision 和有效期共同锁定费率。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."credit_rate_versions"."model_id" IS '逻辑引用 ModelCatalogEntry 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."credit_rate_versions"."model_version" IS '费率适用的实际模型版本；Request 预占后锁定，模型升级不得重算历史 Usage。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."credit_rate_versions"."input_credit_per_million_tokens" IS 'Usage Accounting Token 整数计量 input_credit_per_million_tokens；单位为 token，禁止浮点和隐式换算，须关联 Request/Usage 或周期投影。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."credit_rate_versions"."output_credit_per_million_tokens" IS 'Usage Accounting Token 整数计量 output_credit_per_million_tokens；单位为 token，禁止浮点和隐式换算，须关联 Request/Usage 或周期投影。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."credit_rate_versions"."rounding_mode" IS 'Usage Accounting 受控枚举；允许值：CEILING（向上取整）、FLOOR（向下取整）、HALF_UP（四舍五入）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."credit_rate_versions"."minimum_charge_credit" IS 'Usage Accounting 整数 AI Credit 字段 minimum_charge_credit；P0 不表达人民币，变化必须来自 Ledger 或其可重建投影。 字段约束：必填，默认 0。';
COMMENT ON COLUMN "ai_governance"."credit_rate_versions"."valid_from" IS 'Usage Accounting UTC 业务时间 valid_from；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."credit_rate_versions"."valid_until" IS 'Usage Accounting UTC 业务时间 valid_until；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."credit_rate_versions"."status" IS 'Usage Accounting 受控枚举；允许值：DRAFT（草稿未生效）、APPROVED（已批准，尚需发布或生效）、ACTIVE（有效并参与运行决策）、SUPERSEDED（已被更高 revision 替代）、REVOKED（已撤销，不参与新决策）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."credit_rate_versions"."revision" IS '配置业务 revision；每次有效内容变化严格递增，旧 revision 不得覆盖新版本。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."credit_rate_versions"."row_version" IS '乐观锁版本；每次成功 UPDATE 严格递增，用于防止并发覆盖。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."credit_rate_versions"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."credit_rate_versions"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."credit_rate_versions"."created_by" IS '创建记录的稳定主体标识；保存 ID 或服务主体，不保存仅供展示的姓名。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."credit_rate_versions"."updated_by" IS '最后修改记录的稳定主体标识；必须与 AuditEvent.actor 可关联。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."credit_rate_versions"."operation_source" IS '变更来源；ADMIN=治理后台，API=管理接口，SYNC=上游同步，SYSTEM=平台任务，MIGRATION=迁移。 字段约束：必填，默认 ''ADMIN''。';
COMMENT ON COLUMN "ai_governance"."credit_rate_versions"."operation_trace_id" IS '产生该次写入的 Trace 标识；用于跨 Module 联合排障和审计关联。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."credit_rate_versions"."is_deleted" IS '软删除标志；false=有效，true=逻辑删除且必须同时填写 deleted_at、deleted_by、delete_reason。 字段约束：必填，默认 false。';
COMMENT ON COLUMN "ai_governance"."credit_rate_versions"."deleted_at" IS '软删除生效时间，UTC；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."credit_rate_versions"."deleted_by" IS '执行软删除的稳定主体标识；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."credit_rate_versions"."delete_reason" IS '软删除业务原因；必须说明业务依据、影响范围并关联删除 AuditEvent，禁止空原因或无审计删除。字段约束：可空，仅 is_deleted=true 时必填。';
COMMENT ON TABLE "ai_governance"."credit_rate_versions" IS 'Token 到整数 AI Credit 的版本化换算事实；不含人民币。';

-- ----------------------------
-- Table structure for data_integrity_findings
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."data_integrity_findings";
CREATE TABLE "ai_governance"."data_integrity_findings" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "finding_code" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "check_code" varchar(64) COLLATE "pg_catalog"."default" NOT NULL,
  "target_table" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "target_id" uuid NOT NULL,
  "reference_table" varchar(128) COLLATE "pg_catalog"."default",
  "reference_id" uuid,
  "severity" varchar(16) COLLATE "pg_catalog"."default" NOT NULL,
  "evidence_json" jsonb NOT NULL,
  "status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "detected_at" timestamptz(6) NOT NULL,
  "resolved_at" timestamptz(6),
  "resolution_note" varchar(2048) COLLATE "pg_catalog"."default",
  "approval_request_id" uuid,
  "row_version" int8 NOT NULL DEFAULT 1,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "created_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "updated_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "operation_trace_id" varchar(128) COLLATE "pg_catalog"."default"
)
;
COMMENT ON COLUMN "ai_governance"."data_integrity_findings"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."data_integrity_findings"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."data_integrity_findings"."finding_code" IS '完整性发现的唯一业务编号；用于告警、处置、复查和审计，解决后不得复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."data_integrity_findings"."check_code" IS '产生完整性发现的检查规则代码；定位孤儿引用、跨 Tenant、状态冲突或投影差异。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."data_integrity_findings"."target_table" IS '发现完整性问题的 ai_governance 目标表名；必须来自受控表目录，禁止任意 SQL 标识注入。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."data_integrity_findings"."target_id" IS '逻辑引用 target_type 指定的 Menu 或 Route 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."data_integrity_findings"."reference_table" IS '发生孤儿或跨 Tenant 引用时的可选目标表名；非引用类检查允许为空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."data_integrity_findings"."reference_id" IS '逻辑引用 reference 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."data_integrity_findings"."severity" IS 'Data Integrity 受控枚举；允许值：LOW（低等级）、MEDIUM（中等级）、HIGH（高等级）、CRITICAL（关键等级，触发强控制）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."data_integrity_findings"."evidence_json" IS 'Data Integrity JSONB 证据或扩展快照 evidence_json；不承载核心可检索关系，值必须为 object。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."data_integrity_findings"."status" IS 'Data Integrity 受控枚举；允许值：OPEN（已开启，等待处理）、INVESTIGATING（差异调查中）、RESOLVED（差异已按批准方式处置）、ACCEPTED_RISK（风险经审批接受）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."data_integrity_findings"."detected_at" IS 'Data Integrity UTC 业务时间 detected_at；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."data_integrity_findings"."resolved_at" IS 'Data Integrity UTC 业务时间 resolved_at；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."data_integrity_findings"."resolution_note" IS '完整性问题的人工处置说明；RESOLVED 时必填并描述证据、修正事实及防复发措施。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."data_integrity_findings"."approval_request_id" IS '关联 ApprovalRequest；敏感动作须校验同 Tenant、APPROVED、未过期且 payload_hash 一致。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."data_integrity_findings"."row_version" IS '乐观锁版本；每次成功 UPDATE 严格递增，用于防止并发覆盖。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."data_integrity_findings"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."data_integrity_findings"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."data_integrity_findings"."created_by" IS '创建记录的稳定主体标识；保存 ID 或服务主体，不保存仅供展示的姓名。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."data_integrity_findings"."updated_by" IS '最后修改记录的稳定主体标识；必须与 AuditEvent.actor 可关联。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."data_integrity_findings"."operation_trace_id" IS '产生该次写入的 Trace 标识；用于跨 Module 联合排障和审计关联。 字段约束：可空。';
COMMENT ON TABLE "ai_governance"."data_integrity_findings" IS '无外键模型的周期引用完整性检查与处置事实。';

-- ----------------------------
-- Table structure for emergency_credit_grant_keys
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."emergency_credit_grant_keys";
CREATE TABLE "ai_governance"."emergency_credit_grant_keys" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "emergency_credit_grant_id" uuid NOT NULL,
  "key_id" uuid NOT NULL,
  "key_revision" int8 NOT NULL,
  "status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "valid_from" timestamptz(6) NOT NULL,
  "valid_until" timestamptz(6) NOT NULL,
  "row_version" int8 NOT NULL DEFAULT 1,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "created_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "updated_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "operation_trace_id" varchar(128) COLLATE "pg_catalog"."default"
)
;
COMMENT ON COLUMN "ai_governance"."emergency_credit_grant_keys"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."emergency_credit_grant_keys"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."emergency_credit_grant_keys"."emergency_credit_grant_id" IS '逻辑引用 EmergencyCreditGrant 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."emergency_credit_grant_keys"."key_id" IS '逻辑引用 UserApiKey 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."emergency_credit_grant_keys"."key_revision" IS '调用或授权时锁定的 User API Key revision；用于证明历史归属和动作上限。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."emergency_credit_grant_keys"."status" IS 'Funding Module 受控枚举；允许值：ACTIVE（有效并参与运行决策）、REVOKED（已撤销，不参与新决策）、EXPIRED（已过有效期）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."emergency_credit_grant_keys"."valid_from" IS 'Funding Module UTC 业务时间 valid_from；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."emergency_credit_grant_keys"."valid_until" IS 'Funding Module UTC 业务时间 valid_until；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."emergency_credit_grant_keys"."row_version" IS '乐观锁版本；每次成功 UPDATE 严格递增，用于防止并发覆盖。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."emergency_credit_grant_keys"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."emergency_credit_grant_keys"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."emergency_credit_grant_keys"."created_by" IS '创建记录的稳定主体标识；保存 ID 或服务主体，不保存仅供展示的姓名。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."emergency_credit_grant_keys"."updated_by" IS '最后修改记录的稳定主体标识；必须与 AuditEvent.actor 可关联。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."emergency_credit_grant_keys"."operation_trace_id" IS '产生该次写入的 Trace 标识；用于跨 Module 联合排障和审计关联。 字段约束：可空。';
COMMENT ON TABLE "ai_governance"."emergency_credit_grant_keys" IS '紧急信用授权允许使用的 Key 白名单。';

-- ----------------------------
-- Table structure for emergency_credit_grants
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."emergency_credit_grants";
CREATE TABLE "ai_governance"."emergency_credit_grants" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "user_allocation_id" uuid NOT NULL,
  "billing_scope_type" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "billing_scope_id" uuid NOT NULL,
  "emergency_facility_account_id" uuid NOT NULL,
  "max_credit_amount" int8 NOT NULL,
  "reason" varchar(1024) COLLATE "pg_catalog"."default" NOT NULL,
  "approval_request_id" uuid NOT NULL,
  "created_by_subject_id" uuid NOT NULL,
  "approved_by_subject_id" uuid NOT NULL,
  "valid_from" timestamptz(6) NOT NULL,
  "valid_until" timestamptz(6) NOT NULL,
  "status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "row_version" int8 NOT NULL DEFAULT 1,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "operation_trace_id" varchar(128) COLLATE "pg_catalog"."default"
)
;
COMMENT ON COLUMN "ai_governance"."emergency_credit_grants"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."emergency_credit_grants"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."emergency_credit_grants"."user_allocation_id" IS '关联 UserAllocation；多把 Key 共享同一人员额度，禁止创建 Key 钱包。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."emergency_credit_grants"."billing_scope_type" IS '资金归属 Scope 类型；仅 ORGANIZATION 或 PROJECT，请求不得覆盖。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."emergency_credit_grants"."billing_scope_id" IS '资金归属组织或项目 UUID；必须与 Key、UserAllocation 和成员资格同 Tenant 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."emergency_credit_grants"."emergency_facility_account_id" IS '逻辑引用 紧急信用 FundingAccount 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."emergency_credit_grants"."max_credit_amount" IS 'Funding Module 整数 AI Credit 字段 max_credit_amount；P0 不表达人民币，变化必须来自 Ledger 或其可重建投影。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."emergency_credit_grants"."reason" IS '申请紧急信用的业务原因和紧急性说明；必须与审批单、额度、期限和 Key 范围一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."emergency_credit_grants"."approval_request_id" IS '关联 ApprovalRequest；敏感动作须校验同 Tenant、APPROVED、未过期且 payload_hash 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."emergency_credit_grants"."created_by_subject_id" IS '逻辑引用 紧急信用创建 IdentitySubject 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."emergency_credit_grants"."approved_by_subject_id" IS '逻辑引用 紧急信用审批 IdentitySubject 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."emergency_credit_grants"."valid_from" IS 'Funding Module UTC 业务时间 valid_from；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."emergency_credit_grants"."valid_until" IS 'Funding Module UTC 业务时间 valid_until；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."emergency_credit_grants"."status" IS 'Funding Module 受控枚举；允许值：PENDING（待处理或待生效）、ACTIVE（有效并参与运行决策）、EXHAUSTED（授权额度已用尽）、EXPIRED（已过有效期）、COVERED（紧急占用已由后续拨付覆盖）、REVOKED（已撤销，不参与新决策）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."emergency_credit_grants"."row_version" IS '乐观锁版本；每次成功 UPDATE 严格递增，用于防止并发覆盖。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."emergency_credit_grants"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."emergency_credit_grants"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."emergency_credit_grants"."updated_by" IS '最后修改记录的稳定主体标识；必须与 AuditEvent.actor 可关联。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."emergency_credit_grants"."operation_trace_id" IS '产生该次写入的 Trace 标识；用于跨 Module 联合排障和审计关联。 字段约束：可空。';
COMMENT ON TABLE "ai_governance"."emergency_credit_grants" IS '受审批、Scope、Key、额度和期限约束的紧急信用授权。';

-- ----------------------------
-- Table structure for funding_accounts
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."funding_accounts";
CREATE TABLE "ai_governance"."funding_accounts" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "account_code" varchar(96) COLLATE "pg_catalog"."default" NOT NULL,
  "account_type" varchar(40) COLLATE "pg_catalog"."default" NOT NULL,
  "owner_type" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "owner_id" uuid NOT NULL,
  "billing_scope_type" varchar(24) COLLATE "pg_catalog"."default",
  "billing_scope_id" uuid,
  "normal_balance" varchar(8) COLLATE "pg_catalog"."default" NOT NULL,
  "status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "revision" int8 NOT NULL DEFAULT 1,
  "row_version" int8 NOT NULL DEFAULT 1,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "created_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "updated_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "operation_source" varchar(32) COLLATE "pg_catalog"."default" NOT NULL DEFAULT 'ADMIN'::character varying,
  "operation_trace_id" varchar(128) COLLATE "pg_catalog"."default",
  "is_deleted" bool NOT NULL DEFAULT false,
  "deleted_at" timestamptz(6),
  "deleted_by" varchar(128) COLLATE "pg_catalog"."default",
  "delete_reason" varchar(512) COLLATE "pg_catalog"."default"
)
;
COMMENT ON COLUMN "ai_governance"."funding_accounts"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."funding_accounts"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."funding_accounts"."account_code" IS 'FundingAccount 的稳定会计式编号；唯一定位账户，关闭后不得复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."funding_accounts"."account_type" IS 'Funding Module 受控枚举；允许值：COMPANY_AVAILABLE（公司可拨付账户）、SCOPE_AVAILABLE（组织或项目可拨付账户）、USER_AVAILABLE（人员可用账户）、USER_RESERVED（人员预占账户）、PLATFORM_CONSUMED（平台已消耗归集账户）、EMERGENCY_FACILITY_AVAILABLE（紧急信用可用账户）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."funding_accounts"."owner_type" IS 'Funding Module 受控枚举；允许值：TENANT（租户作用域）、ORGANIZATION（组织作用域）、PROJECT（项目作用域）、USER_ALLOCATION（人员额度所有者）、PLATFORM（平台治理域）、EMERGENCY_FACILITY（紧急信用设施所有者）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."funding_accounts"."owner_id" IS '逻辑引用 owner_type 指定的账户所有者 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."funding_accounts"."billing_scope_type" IS '资金归属 Scope 类型；仅 ORGANIZATION 或 PROJECT，请求不得覆盖。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."funding_accounts"."billing_scope_id" IS '资金归属组织或项目 UUID；必须与 Key、UserAllocation 和成员资格同 Tenant 一致。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."funding_accounts"."normal_balance" IS 'Funding Module 受控枚举；允许值：DEBIT（借方，credit_delta 为正）、CREDIT（贷方，credit_delta 为负）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."funding_accounts"."status" IS 'Funding Module 受控枚举；允许值：DRAFT（草稿未生效）、ACTIVE（有效并参与运行决策）、FROZEN（冻结，不接受普通新增交易）、CLOSED（关闭终态，仅保留历史）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."funding_accounts"."revision" IS '配置业务 revision；每次有效内容变化严格递增，旧 revision 不得覆盖新版本。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."funding_accounts"."row_version" IS '乐观锁版本；每次成功 UPDATE 严格递增，用于防止并发覆盖。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."funding_accounts"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."funding_accounts"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."funding_accounts"."created_by" IS '创建记录的稳定主体标识；保存 ID 或服务主体，不保存仅供展示的姓名。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."funding_accounts"."updated_by" IS '最后修改记录的稳定主体标识；必须与 AuditEvent.actor 可关联。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."funding_accounts"."operation_source" IS '变更来源；ADMIN=治理后台，API=管理接口，SYNC=上游同步，SYSTEM=平台任务，MIGRATION=迁移。 字段约束：必填，默认 ''ADMIN''。';
COMMENT ON COLUMN "ai_governance"."funding_accounts"."operation_trace_id" IS '产生该次写入的 Trace 标识；用于跨 Module 联合排障和审计关联。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."funding_accounts"."is_deleted" IS '软删除标志；false=有效，true=逻辑删除且必须同时填写 deleted_at、deleted_by、delete_reason。 字段约束：必填，默认 false。';
COMMENT ON COLUMN "ai_governance"."funding_accounts"."deleted_at" IS '软删除生效时间，UTC；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."funding_accounts"."deleted_by" IS '执行软删除的稳定主体标识；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."funding_accounts"."delete_reason" IS '软删除业务原因；必须说明业务依据、影响范围并关联删除 AuditEvent，禁止空原因或无审计删除。字段约束：可空，仅 is_deleted=true 时必填。';
COMMENT ON TABLE "ai_governance"."funding_accounts" IS 'AI Credit 复式账本账户登记；不保存可写余额。';

-- ----------------------------
-- Table structure for identity_issuer_configs
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."identity_issuer_configs";
CREATE TABLE "ai_governance"."identity_issuer_configs" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "issuer_uri" varchar(512) COLLATE "pg_catalog"."default" NOT NULL,
  "audience" varchar(256) COLLATE "pg_catalog"."default" NOT NULL,
  "jwks_uri" varchar(512) COLLATE "pg_catalog"."default" NOT NULL,
  "allowed_algorithms" varchar(32)[] COLLATE "pg_catalog"."default" NOT NULL DEFAULT ARRAY['RS256'::character varying],
  "clock_skew_seconds" int4 NOT NULL DEFAULT 60,
  "status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "revision" int8 NOT NULL DEFAULT 1,
  "row_version" int8 NOT NULL DEFAULT 1,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "created_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "updated_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "operation_source" varchar(32) COLLATE "pg_catalog"."default" NOT NULL DEFAULT 'ADMIN'::character varying,
  "operation_trace_id" varchar(128) COLLATE "pg_catalog"."default",
  "is_deleted" bool NOT NULL DEFAULT false,
  "deleted_at" timestamptz(6),
  "deleted_by" varchar(128) COLLATE "pg_catalog"."default",
  "delete_reason" varchar(512) COLLATE "pg_catalog"."default",
  "issuer_type" varchar(32) COLLATE "pg_catalog"."default" NOT NULL DEFAULT 'ENTERPRISE_OIDC'::character varying
)
;
COMMENT ON COLUMN "ai_governance"."identity_issuer_configs"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."identity_issuer_configs"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."identity_issuer_configs"."issuer_uri" IS '受信 OIDC/OAuth Issuer 的规范化 URI；必须与主体令牌 iss 完全匹配，禁止模糊或后缀匹配。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."identity_issuer_configs"."audience" IS '业务方治理平台接受的令牌 audience；验签时必须精确匹配，不能由调用请求覆盖。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."identity_issuer_configs"."jwks_uri" IS 'Issuer 公钥集合 JWKS 的 HTTPS 地址；Identity Adapter 获取并缓存验签键，禁止保存私钥。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."identity_issuer_configs"."allowed_algorithms" IS '允许用于主体令牌验签的算法白名单；默认仅 RS256，禁止 none 和未批准算法。 字段约束：必填，默认 ARRAY[''RS256'']::varchar[]。';
COMMENT ON COLUMN "ai_governance"."identity_issuer_configs"."clock_skew_seconds" IS '令牌时间校验允许的时钟偏差，单位秒，范围 0 至 300；不得绕过 exp、nbf 校验。 字段约束：必填，默认 60。';
COMMENT ON COLUMN "ai_governance"."identity_issuer_configs"."status" IS 'Identity Center 受控枚举；允许值：DRAFT（草稿未生效）、ACTIVE（有效并参与运行决策）、SUSPENDED（暂停，不接受新增业务）、RETIRED（退役终态，仅保留历史）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."identity_issuer_configs"."revision" IS '配置业务 revision；每次有效内容变化严格递增，旧 revision 不得覆盖新版本。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."identity_issuer_configs"."row_version" IS '乐观锁版本；每次成功 UPDATE 严格递增，用于防止并发覆盖。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."identity_issuer_configs"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."identity_issuer_configs"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."identity_issuer_configs"."created_by" IS '创建记录的稳定主体标识；保存 ID 或服务主体，不保存仅供展示的姓名。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."identity_issuer_configs"."updated_by" IS '最后修改记录的稳定主体标识；必须与 AuditEvent.actor 可关联。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."identity_issuer_configs"."operation_source" IS '变更来源；ADMIN=治理后台，API=管理接口，SYNC=上游同步，SYSTEM=平台任务，MIGRATION=迁移。 字段约束：必填，默认 ''ADMIN''。';
COMMENT ON COLUMN "ai_governance"."identity_issuer_configs"."operation_trace_id" IS '产生该次写入的 Trace 标识；用于跨 Module 联合排障和审计关联。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."identity_issuer_configs"."is_deleted" IS '软删除标志；false=有效，true=逻辑删除且必须同时填写 deleted_at、deleted_by、delete_reason。 字段约束：必填，默认 false。';
COMMENT ON COLUMN "ai_governance"."identity_issuer_configs"."deleted_at" IS '软删除生效时间，UTC；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."identity_issuer_configs"."deleted_by" IS '执行软删除的稳定主体标识；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."identity_issuer_configs"."delete_reason" IS '软删除业务原因；必须说明业务依据、影响范围并关联删除 AuditEvent，禁止空原因或无审计删除。字段约束：可空，仅 is_deleted=true 时必填。';
COMMENT ON COLUMN "ai_governance"."identity_issuer_configs"."issuer_type" IS 'Issuer 运行类型；ENTERPRISE_OIDC 表示外部企业联邦，PLATFORM_LOCAL 表示平台自治签发，默认 ENTERPRISE_OIDC 兼容旧配置。';
COMMENT ON TABLE "ai_governance"."identity_issuer_configs" IS '企业 SSO/OIDC 发行方信任配置。';

-- ----------------------------
-- Table structure for identity_subjects
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."identity_subjects";
CREATE TABLE "ai_governance"."identity_subjects" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "issuer_config_id" uuid NOT NULL,
  "external_subject" varchar(256) COLLATE "pg_catalog"."default" NOT NULL,
  "subject_type" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "display_name" varchar(160) COLLATE "pg_catalog"."default" NOT NULL,
  "email" varchar(320) COLLATE "pg_catalog"."default",
  "user_type" varchar(32) COLLATE "pg_catalog"."default" NOT NULL DEFAULT 'INTERNAL'::character varying,
  "status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "enterprise_roles" varchar(64)[] COLLATE "pg_catalog"."default" NOT NULL DEFAULT ARRAY[]::character varying[],
  "identity_revision" int8 NOT NULL DEFAULT 1,
  "last_synced_at" timestamptz(6),
  "disabled_at" timestamptz(6),
  "revision" int8 NOT NULL DEFAULT 1,
  "row_version" int8 NOT NULL DEFAULT 1,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "created_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "updated_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "operation_source" varchar(32) COLLATE "pg_catalog"."default" NOT NULL DEFAULT 'SYNC'::character varying,
  "operation_trace_id" varchar(128) COLLATE "pg_catalog"."default",
  "is_deleted" bool NOT NULL DEFAULT false,
  "deleted_at" timestamptz(6),
  "deleted_by" varchar(128) COLLATE "pg_catalog"."default",
  "delete_reason" varchar(512) COLLATE "pg_catalog"."default",
  "identity_origin" varchar(24) COLLATE "pg_catalog"."default" NOT NULL DEFAULT 'FEDERATED'::character varying,
  "primary_login_identity_id" uuid
)
;
COMMENT ON COLUMN "ai_governance"."identity_subjects"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."identity_subjects"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."identity_subjects"."issuer_config_id" IS '主体签发器逻辑引用；FEDERATED/OAUTH 指向受信企业或第三方 Issuer，LOCAL 指向平台本地 Issuer，同 Tenant、ACTIVE 和 revision 由 Identity Center 校验。';
COMMENT ON COLUMN "ai_governance"."identity_subjects"."external_subject" IS '签发器命名空间内的稳定主体标识；企业联邦采用上游 subject，平台本地采用不可变内部 subject 字符串，不得使用手机号、邮箱、openid 或展示名。';
COMMENT ON COLUMN "ai_governance"."identity_subjects"."subject_type" IS 'Identity Center 受控枚举；允许值：USER（企业用户主体）、SERVICE（服务主体）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."identity_subjects"."display_name" IS '企业 IAM 同步的主体展示名；只用于界面和审计可读性，不参与主体唯一性与授权。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."identity_subjects"."email" IS '兼容旧联邦投影的非权威邮箱字段；新写入使用 user_contact_points 加密值与 HMAC，授权、登录和资金不得依赖本列，可空。';
COMMENT ON COLUMN "ai_governance"."identity_subjects"."user_type" IS 'Identity Center 受控枚举；允许值：INTERNAL（内部网络或内部数据）、EXTERNAL（外部网络或供应商）、INTERNAL_ONLY（仅允许内部模型网络）、SERVICE（服务主体）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填，默认 ''INTERNAL''。';
COMMENT ON COLUMN "ai_governance"."identity_subjects"."status" IS 'Identity Center 受控枚举；允许值：PENDING（待处理或待生效）、ACTIVE（有效并参与运行决策）、DISABLED（已停用）、LOCKED（临时锁定）、RETIRED（退役终态，仅保留历史）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."identity_subjects"."enterprise_roles" IS '企业 IAM 同步的企业角色代码集合；只读投影，Authorization 不在本表修改。 字段约束：必填，默认 ARRAY[]::varchar[]。';
COMMENT ON COLUMN "ai_governance"."identity_subjects"."identity_revision" IS '企业身份投影 revision；停用或身份收紧后旧版本不得恢复权限。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."identity_subjects"."last_synced_at" IS 'Identity Center UTC 业务时间 last_synced_at；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."identity_subjects"."disabled_at" IS 'Identity Center UTC 业务时间 disabled_at；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."identity_subjects"."revision" IS '配置业务 revision；每次有效内容变化严格递增，旧 revision 不得覆盖新版本。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."identity_subjects"."row_version" IS '乐观锁版本；每次成功 UPDATE 严格递增，用于防止并发覆盖。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."identity_subjects"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."identity_subjects"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."identity_subjects"."created_by" IS '创建记录的稳定主体标识；保存 ID 或服务主体，不保存仅供展示的姓名。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."identity_subjects"."updated_by" IS '最后修改记录的稳定主体标识；必须与 AuditEvent.actor 可关联。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."identity_subjects"."operation_source" IS '变更来源；ADMIN=治理后台，API=管理接口，SYNC=上游同步，SYSTEM=平台任务，MIGRATION=迁移。 字段约束：必填，默认 ''SYNC''。';
COMMENT ON COLUMN "ai_governance"."identity_subjects"."operation_trace_id" IS '产生该次写入的 Trace 标识；用于跨 Module 联合排障和审计关联。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."identity_subjects"."is_deleted" IS '软删除标志；false=有效，true=逻辑删除且必须同时填写 deleted_at、deleted_by、delete_reason。 字段约束：必填，默认 false。';
COMMENT ON COLUMN "ai_governance"."identity_subjects"."deleted_at" IS '软删除生效时间，UTC；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."identity_subjects"."deleted_by" IS '执行软删除的稳定主体标识；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."identity_subjects"."delete_reason" IS '软删除业务原因；必须说明业务依据、影响范围并关联删除 AuditEvent，禁止空原因或无审计删除。字段约束：可空，仅 is_deleted=true 时必填。';
COMMENT ON COLUMN "ai_governance"."identity_subjects"."identity_origin" IS '主体首次建立来源；FEDERATED 为企业联邦，LOCAL 为平台手机号/邮箱，OAUTH 为 WeChat/OIDC；后续可绑定其他登录方式但不改变 subject_id。';
COMMENT ON COLUMN "ai_governance"."identity_subjects"."primary_login_identity_id" IS 'Identity Center 选定的首选登录标识逻辑引用；空值表示尚未指定，写入时校验同 Tenant、同 subject_id 且状态有效。';
COMMENT ON TABLE "ai_governance"."identity_subjects" IS '企业身份运行投影；企业 IAM 是上游唯一身份事实源。';

-- ----------------------------
-- Table structure for inbox_receipts
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."inbox_receipts";
CREATE TABLE "ai_governance"."inbox_receipts" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "consumer_code" varchar(96) COLLATE "pg_catalog"."default" NOT NULL,
  "event_id" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "schema_version" varchar(16) COLLATE "pg_catalog"."default" NOT NULL,
  "payload_hash" varchar(71) COLLATE "pg_catalog"."default" NOT NULL,
  "status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "received_at" timestamptz(6) NOT NULL,
  "processed_at" timestamptz(6),
  "attempt_count" int4 NOT NULL DEFAULT 0,
  "last_error_code" varchar(128) COLLATE "pg_catalog"."default",
  "row_version" int8 NOT NULL DEFAULT 1,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "trace_id" varchar(128) COLLATE "pg_catalog"."default" NOT NULL
)
;
COMMENT ON COLUMN "ai_governance"."inbox_receipts"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."inbox_receipts"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."inbox_receipts"."consumer_code" IS '事件消费者的稳定代码；与 Tenant、event_id 组成幂等边界。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."inbox_receipts"."event_id" IS '已接收 Outbox 事件的全局幂等标识；与 consumer_code、tenant_id 共同唯一。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."inbox_receipts"."schema_version" IS '消费者实际接收的事件 Schema 版本；处理前必须完成兼容性校验并保留审计。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."inbox_receipts"."payload_hash" IS '审批或事件载荷的 SHA-256 摘要；审批后载荷变化必须重新申请。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."inbox_receipts"."status" IS 'Reliable Eventing 受控枚举；允许值：RECEIVED（事件已接收）、PROCESSING（处理中并持有租约）、PROCESSED（事件已成功处理）、FAILED（执行失败）、DEAD_LETTER（超过重试阈值，进入人工处置）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."inbox_receipts"."received_at" IS 'Reliable Eventing UTC 业务时间 received_at；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."inbox_receipts"."processed_at" IS 'Reliable Eventing UTC 业务时间 processed_at；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."inbox_receipts"."attempt_count" IS 'Reliable Eventing 顺序、计数或重放控制字段 attempt_count；用于确定性排序、幂等、投影或完整性验证，禁止无审计跳变。 字段约束：必填，默认 0。';
COMMENT ON COLUMN "ai_governance"."inbox_receipts"."last_error_code" IS '消费者最近一次事件处理失败的稳定错误码；用于重试、告警和 DEAD_LETTER 分析，成功或尚未失败时为空。字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."inbox_receipts"."row_version" IS '乐观锁版本；每次成功 UPDATE 严格递增，用于防止并发覆盖。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."inbox_receipts"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."inbox_receipts"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."inbox_receipts"."trace_id" IS '端到端 Trace 标识；贯通 Request、Attempt、Usage、Ledger、安全和 Audit。 字段约束：必填。';
COMMENT ON TABLE "ai_governance"."inbox_receipts" IS '消费者事件幂等与处理状态记录。';

-- ----------------------------
-- Table structure for key_limit_policies
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."key_limit_policies";
CREATE TABLE "ai_governance"."key_limit_policies" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "policy_code" varchar(64) COLLATE "pg_catalog"."default" NOT NULL,
  "policy_name" varchar(160) COLLATE "pg_catalog"."default" NOT NULL,
  "per_request_credit_limit" int8,
  "daily_credit_limit" int8,
  "monthly_credit_limit" int8,
  "per_request_token_limit" int8,
  "daily_token_limit" int8,
  "monthly_token_limit" int8,
  "max_concurrency" int4 NOT NULL,
  "rps_limit" int4 NOT NULL,
  "allow_emergency_credit" bool NOT NULL DEFAULT false,
  "valid_from" timestamptz(6) NOT NULL,
  "valid_until" timestamptz(6),
  "status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "revision" int8 NOT NULL DEFAULT 1,
  "row_version" int8 NOT NULL DEFAULT 1,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "created_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "updated_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "operation_source" varchar(32) COLLATE "pg_catalog"."default" NOT NULL DEFAULT 'ADMIN'::character varying,
  "operation_trace_id" varchar(128) COLLATE "pg_catalog"."default",
  "is_deleted" bool NOT NULL DEFAULT false,
  "deleted_at" timestamptz(6),
  "deleted_by" varchar(128) COLLATE "pg_catalog"."default",
  "delete_reason" varchar(512) COLLATE "pg_catalog"."default"
)
;
COMMENT ON COLUMN "ai_governance"."key_limit_policies"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."key_limit_policies"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."key_limit_policies"."policy_code" IS 'Key 消费限制策略的稳定代码；用于签发、revision 锁定和运营检索。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."key_limit_policies"."policy_name" IS 'Key 限额策略显示名称；实际限制由 Credit、Token、并发、RPS 字段决定。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."key_limit_policies"."per_request_credit_limit" IS 'User Key Center 整数 AI Credit 字段 per_request_credit_limit；P0 不表达人民币，变化必须来自 Ledger 或其可重建投影。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."key_limit_policies"."daily_credit_limit" IS 'User Key Center 整数 AI Credit 字段 daily_credit_limit；P0 不表达人民币，变化必须来自 Ledger 或其可重建投影。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."key_limit_policies"."monthly_credit_limit" IS 'User Key Center 整数 AI Credit 字段 monthly_credit_limit；P0 不表达人民币，变化必须来自 Ledger 或其可重建投影。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."key_limit_policies"."per_request_token_limit" IS 'User Key Center Token 整数计量 per_request_token_limit；单位为 token，禁止浮点和隐式换算，须关联 Request/Usage 或周期投影。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."key_limit_policies"."daily_token_limit" IS 'User Key Center Token 整数计量 daily_token_limit；单位为 token，禁止浮点和隐式换算，须关联 Request/Usage 或周期投影。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."key_limit_policies"."monthly_token_limit" IS 'User Key Center Token 整数计量 monthly_token_limit；单位为 token，禁止浮点和隐式换算，须关联 Request/Usage 或周期投影。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."key_limit_policies"."max_concurrency" IS '单 Key 同时处于活跃 Request/Reservation 的最大数量；必须大于 0。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."key_limit_policies"."rps_limit" IS '单 Key 每秒允许的新请求数；必须大于 0，并与并发及额度限制同时生效。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."key_limit_policies"."allow_emergency_credit" IS '是否允许在有效 EmergencyCreditGrant 内使用紧急信用。 字段约束：必填，默认 false。';
COMMENT ON COLUMN "ai_governance"."key_limit_policies"."valid_from" IS 'User Key Center UTC 业务时间 valid_from；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."key_limit_policies"."valid_until" IS 'User Key Center UTC 业务时间 valid_until；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."key_limit_policies"."status" IS 'User Key Center 受控枚举；允许值：DRAFT（草稿未生效）、ACTIVE（有效并参与运行决策）、SUSPENDED（暂停，不接受新增业务）、RETIRED（退役终态，仅保留历史）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."key_limit_policies"."revision" IS '配置业务 revision；每次有效内容变化严格递增，旧 revision 不得覆盖新版本。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."key_limit_policies"."row_version" IS '乐观锁版本；每次成功 UPDATE 严格递增，用于防止并发覆盖。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."key_limit_policies"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."key_limit_policies"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."key_limit_policies"."created_by" IS '创建记录的稳定主体标识；保存 ID 或服务主体，不保存仅供展示的姓名。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."key_limit_policies"."updated_by" IS '最后修改记录的稳定主体标识；必须与 AuditEvent.actor 可关联。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."key_limit_policies"."operation_source" IS '变更来源；ADMIN=治理后台，API=管理接口，SYNC=上游同步，SYSTEM=平台任务，MIGRATION=迁移。 字段约束：必填，默认 ''ADMIN''。';
COMMENT ON COLUMN "ai_governance"."key_limit_policies"."operation_trace_id" IS '产生该次写入的 Trace 标识；用于跨 Module 联合排障和审计关联。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."key_limit_policies"."is_deleted" IS '软删除标志；false=有效，true=逻辑删除且必须同时填写 deleted_at、deleted_by、delete_reason。 字段约束：必填，默认 false。';
COMMENT ON COLUMN "ai_governance"."key_limit_policies"."deleted_at" IS '软删除生效时间，UTC；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."key_limit_policies"."deleted_by" IS '执行软删除的稳定主体标识；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."key_limit_policies"."delete_reason" IS '软删除业务原因；必须说明业务依据、影响范围并关联删除 AuditEvent，禁止空原因或无审计删除。字段约束：可空，仅 is_deleted=true 时必填。';
COMMENT ON TABLE "ai_governance"."key_limit_policies" IS '每把 User API Key 的 Credit、Token、并发和 RPS 消费约束；不是钱包或拨付事实。';

-- ----------------------------
-- Table structure for key_usage_counter_projections
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."key_usage_counter_projections";
CREATE TABLE "ai_governance"."key_usage_counter_projections" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "key_id" uuid NOT NULL,
  "period_type" varchar(16) COLLATE "pg_catalog"."default" NOT NULL,
  "period_start" timestamptz(6) NOT NULL,
  "period_end" timestamptz(6) NOT NULL,
  "settled_credit" int8 NOT NULL DEFAULT 0,
  "reserved_credit" int8 NOT NULL DEFAULT 0,
  "settled_tokens" int8 NOT NULL DEFAULT 0,
  "reserved_token_ceiling" int8 NOT NULL DEFAULT 0,
  "current_concurrency" int4 NOT NULL DEFAULT 0,
  "source_watermark" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "generation" int8 NOT NULL,
  "projection_status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "subject_id" uuid NOT NULL,
  "user_allocation_id" uuid NOT NULL,
  "billing_scope_type" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "billing_scope_id" uuid NOT NULL
)
;
COMMENT ON COLUMN "ai_governance"."key_usage_counter_projections"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."key_usage_counter_projections"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."key_usage_counter_projections"."key_id" IS '逻辑引用 UserApiKey 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."key_usage_counter_projections"."period_type" IS 'Usage Projection 受控枚举；允许值：DAY（自然日统计周期）、MONTH（自然月统计周期）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."key_usage_counter_projections"."period_start" IS 'Usage Projection UTC 业务时间 period_start；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."key_usage_counter_projections"."period_end" IS 'Usage Projection UTC 业务时间 period_end；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."key_usage_counter_projections"."settled_credit" IS '指定周期内已结算的整数 AI Credit；由 SETTLE 分录聚合。 字段约束：必填，默认 0。';
COMMENT ON COLUMN "ai_governance"."key_usage_counter_projections"."reserved_credit" IS '指定周期内未收敛预占的整数 AI Credit；由相关账本分录聚合。 字段约束：必填，默认 0。';
COMMENT ON COLUMN "ai_governance"."key_usage_counter_projections"."settled_tokens" IS 'Usage Projection Token 整数计量 settled_tokens；单位为 token，禁止浮点和隐式换算，须关联 Request/Usage 或周期投影。 字段约束：必填，默认 0。';
COMMENT ON COLUMN "ai_governance"."key_usage_counter_projections"."reserved_token_ceiling" IS 'Usage Projection Token 整数计量 reserved_token_ceiling；单位为 token，禁止浮点和隐式换算，须关联 Request/Usage 或周期投影。 字段约束：必填，默认 0。';
COMMENT ON COLUMN "ai_governance"."key_usage_counter_projections"."current_concurrency" IS '当前 Key 活跃调用数；由 Request/Reservation 生命周期维护，后台不得直接修改。 字段约束：必填，默认 0。';
COMMENT ON COLUMN "ai_governance"."key_usage_counter_projections"."source_watermark" IS '投影已消费的权威事实水位；用于重建、追赶和新 generation 切换。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."key_usage_counter_projections"."generation" IS '投影或 RuntimeSnapshot 的重建代次；新 generation 完成全量校验后原子切换，旧 generation 不得覆盖或与新代混读。字段约束：必填且大于 0。';
COMMENT ON COLUMN "ai_governance"."key_usage_counter_projections"."projection_status" IS 'Usage Projection 受控枚举；允许值：BUILDING（构建中）、CURRENT（投影与事实水位一致）、STALE（投影已落后）、FAILED（执行失败）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."key_usage_counter_projections"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."key_usage_counter_projections"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."key_usage_counter_projections"."subject_id" IS 'Key 计数投影所属 User Subject；由 Key owner 重建且不可作为身份权威。';
COMMENT ON COLUMN "ai_governance"."key_usage_counter_projections"."user_allocation_id" IS 'Key 计数投影对应 UserAllocation；与 subject_id、Key 和 BillingScope 必须一致。';
COMMENT ON COLUMN "ai_governance"."key_usage_counter_projections"."billing_scope_type" IS 'Key 计数资金范围类型；ORGANIZATION 为组织，PROJECT 为项目，来源于 Key 绑定。';
COMMENT ON COLUMN "ai_governance"."key_usage_counter_projections"."billing_scope_id" IS 'Key 计数资金范围 ID；与类型成对存在并用于 User/Scope 周期查询。';
COMMENT ON TABLE "ai_governance"."key_usage_counter_projections" IS '由 Reservation 与已结算分录生成的 Key 周期用量投影；不是 Key 钱包。';

-- ----------------------------
-- Table structure for ledger_legs
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."ledger_legs";
CREATE TABLE "ai_governance"."ledger_legs" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "ledger_transaction_id" uuid NOT NULL,
  "transaction_id" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "leg_no" int2 NOT NULL,
  "entry_type" varchar(40) COLLATE "pg_catalog"."default" NOT NULL,
  "account_id" uuid NOT NULL,
  "account_type" varchar(40) COLLATE "pg_catalog"."default" NOT NULL,
  "debit_credit" varchar(8) COLLATE "pg_catalog"."default" NOT NULL,
  "credit_amount" int8 NOT NULL,
  "credit_delta" int8 NOT NULL,
  "company_account_id" uuid,
  "scope_account_id" uuid,
  "user_allocation_id" uuid,
  "key_id" uuid,
  "key_revision" int8,
  "billing_scope_type" varchar(24) COLLATE "pg_catalog"."default",
  "billing_scope_id" uuid,
  "request_id" uuid,
  "attempt_id" uuid,
  "usage_event_id" uuid,
  "reservation_id" uuid,
  "credit_rate_revision" int8,
  "input_tokens" int8,
  "output_tokens" int8,
  "total_tokens" int8,
  "idempotency_key" varchar(256) COLLATE "pg_catalog"."default" NOT NULL,
  "occurred_at" timestamptz(6) NOT NULL,
  "posted_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "actor_type" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "actor_id" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "approval_request_id" uuid,
  "reversal_of_leg_id" uuid,
  "trace_id" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "audit_event_id" uuid NOT NULL,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "subject_id" uuid
)
;
COMMENT ON COLUMN "ai_governance"."ledger_legs"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."ledger_legs"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."ledger_legs"."ledger_transaction_id" IS '逻辑引用 LedgerTransaction 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."ledger_legs"."transaction_id" IS '冗余保存所属 LedgerTransaction 的稳定业务编号；提交时必须与交易头完全一致，便于独立审计导出。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."ledger_legs"."leg_no" IS 'Ledger Module 顺序、计数或重放控制字段 leg_no；用于确定性排序、幂等、投影或完整性验证，禁止无审计跳变。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."ledger_legs"."entry_type" IS 'Ledger Module 受控枚举；允许值：ISSUE（向公司可用账户发行 AI Credit）、ALLOCATE_TO_SCOPE（公司向组织或项目 Scope 拨付）、ALLOCATE_TO_USER（Scope 向 UserAllocation 拨付）、RECLAIM_FROM_USER（从 UserAllocation 回收到 Scope）、RESERVE（从 User Available 转入 User Reserved）、SUPPLEMENTAL_RESERVE（流式消费接近上限时追加预占）、SETTLE（把实际消费从 User Reserved 转入 Platform Consumed）、RELEASE（把未消耗预占退回 User Available）、ADJUST（经审批的账本调整交易）、REVERSAL（对原交易执行等额反向冲销）、EMERGENCY_FACILITY_GRANT（向紧急信用设施授予可用额度）、EMERGENCY_DRAW（从已批准紧急信用设施支用）、EMERGENCY_COVER（用后续拨付覆盖紧急占用）、EMERGENCY_CLOSE（关闭已清偿的紧急信用设施）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."ledger_legs"."account_id" IS '逻辑引用 FundingAccount 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."ledger_legs"."account_type" IS 'Ledger Module 受控枚举；允许值：COMPANY_AVAILABLE（公司可拨付账户）、SCOPE_AVAILABLE（组织或项目可拨付账户）、USER_AVAILABLE（人员可用账户）、USER_RESERVED（人员预占账户）、PLATFORM_CONSUMED（平台已消耗归集账户）、EMERGENCY_FACILITY_AVAILABLE（紧急信用可用账户）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."ledger_legs"."debit_credit" IS 'Ledger Module 受控枚举；允许值：DEBIT（借方，credit_delta 为正）、CREDIT（贷方，credit_delta 为负）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."ledger_legs"."credit_amount" IS 'Ledger Module 整数 AI Credit 字段 credit_amount；P0 不表达人民币，变化必须来自 Ledger 或其可重建投影。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."ledger_legs"."credit_delta" IS 'Ledger Module 整数 AI Credit 字段 credit_delta；P0 不表达人民币，变化必须来自 Ledger 或其可重建投影。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."ledger_legs"."company_account_id" IS '逻辑引用 COMPANY_AVAILABLE FundingAccount 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."ledger_legs"."scope_account_id" IS '逻辑引用 SCOPE_AVAILABLE FundingAccount 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."ledger_legs"."user_allocation_id" IS '关联 UserAllocation；多把 Key 共享同一人员额度，禁止创建 Key 钱包。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."ledger_legs"."key_id" IS '逻辑引用 UserApiKey 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."ledger_legs"."key_revision" IS '调用或授权时锁定的 User API Key revision；用于证明历史归属和动作上限。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."ledger_legs"."billing_scope_type" IS '资金归属 Scope 类型；仅 ORGANIZATION 或 PROJECT，请求不得覆盖。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."ledger_legs"."billing_scope_id" IS '资金归属组织或项目 UUID；必须与 Key、UserAllocation 和成员资格同 Tenant 一致。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."ledger_legs"."request_id" IS '逻辑引用 UsageRequest 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."ledger_legs"."attempt_id" IS '逻辑引用 UsageAttempt 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."ledger_legs"."usage_event_id" IS '逻辑引用 UsageEvent 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."ledger_legs"."reservation_id" IS '逻辑引用 QuotaReservation 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."ledger_legs"."credit_rate_revision" IS 'Ledger Module 整数 AI Credit 字段 credit_rate_revision；P0 不表达人民币，变化必须来自 Ledger 或其可重建投影。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."ledger_legs"."input_tokens" IS 'Ledger Module Token 整数计量 input_tokens；单位为 token，禁止浮点和隐式换算，须关联 Request/Usage 或周期投影。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."ledger_legs"."output_tokens" IS 'Ledger Module Token 整数计量 output_tokens；单位为 token，禁止浮点和隐式换算，须关联 Request/Usage 或周期投影。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."ledger_legs"."total_tokens" IS 'Ledger Module Token 整数计量 total_tokens；单位为 token，禁止浮点和隐式换算，须关联 Request/Usage 或周期投影。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."ledger_legs"."idempotency_key" IS '命令幂等键；同一作用域内相同键和摘要恢复原结果，不同摘要必须冲突。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."ledger_legs"."occurred_at" IS 'Ledger Module UTC 业务时间 occurred_at；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."ledger_legs"."posted_at" IS 'Ledger Module UTC 业务时间 posted_at；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."ledger_legs"."actor_type" IS 'Ledger Module 受控枚举；允许值：USER（企业用户主体）、ADMIN（治理管理员发起）、SERVICE（服务主体）、SYSTEM（平台系统任务发起）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."ledger_legs"."actor_id" IS '产生该分录的稳定主体标识；每条 Leg 必须与交易头 actor 和审批证据一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."ledger_legs"."approval_request_id" IS '关联 ApprovalRequest；敏感动作须校验同 Tenant、APPROVED、未过期且 payload_hash 一致。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."ledger_legs"."reversal_of_leg_id" IS '逻辑引用 被冲销 LedgerLeg 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."ledger_legs"."trace_id" IS '端到端 Trace 标识；贯通 Request、Attempt、Usage、Ledger、安全和 Audit。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."ledger_legs"."audit_event_id" IS '关联统一 AuditEvent；证明该资金或管理事实已进入审计链。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."ledger_legs"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."ledger_legs"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."ledger_legs"."subject_id" IS '分录归属 User Subject；user_allocation_id 非空时必须同时非空，用于直接追溯人员出入账。';
COMMENT ON TABLE "ai_governance"."ledger_legs" IS '复式交易分录；调用型分录强制携带 Key、Allocation、Scope、Request、Reservation 和 Trace。';

-- ----------------------------
-- Table structure for ledger_transactions
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."ledger_transactions";
CREATE TABLE "ai_governance"."ledger_transactions" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "transaction_id" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "entry_type" varchar(40) COLLATE "pg_catalog"."default" NOT NULL,
  "idempotency_key" varchar(256) COLLATE "pg_catalog"."default" NOT NULL,
  "request_id" uuid,
  "reservation_id" uuid,
  "usage_event_id" uuid,
  "emergency_credit_grant_id" uuid,
  "approval_request_id" uuid,
  "reversal_of_transaction_id" uuid,
  "status" varchar(16) COLLATE "pg_catalog"."default" NOT NULL DEFAULT 'POSTED'::character varying,
  "business_reason" varchar(1024) COLLATE "pg_catalog"."default" NOT NULL,
  "occurred_at" timestamptz(6) NOT NULL,
  "posted_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "actor_type" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "actor_id" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "trace_id" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "audit_event_id" uuid NOT NULL,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "subject_id" uuid
)
;
COMMENT ON COLUMN "ai_governance"."ledger_transactions"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."ledger_transactions"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."ledger_transactions"."transaction_id" IS '复式交易的稳定业务编号；Tenant 内唯一并贯通所有 Leg、幂等、审计和对账。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."ledger_transactions"."entry_type" IS 'Ledger Module 受控枚举；允许值：ISSUE（向公司可用账户发行 AI Credit）、ALLOCATE_TO_SCOPE（公司向组织或项目 Scope 拨付）、ALLOCATE_TO_USER（Scope 向 UserAllocation 拨付）、RECLAIM_FROM_USER（从 UserAllocation 回收到 Scope）、RESERVE（从 User Available 转入 User Reserved）、SUPPLEMENTAL_RESERVE（流式消费接近上限时追加预占）、SETTLE（把实际消费从 User Reserved 转入 Platform Consumed）、RELEASE（把未消耗预占退回 User Available）、ADJUST（经审批的账本调整交易）、REVERSAL（对原交易执行等额反向冲销）、EMERGENCY_FACILITY_GRANT（向紧急信用设施授予可用额度）、EMERGENCY_DRAW（从已批准紧急信用设施支用）、EMERGENCY_COVER（用后续拨付覆盖紧急占用）、EMERGENCY_CLOSE（关闭已清偿的紧急信用设施）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."ledger_transactions"."idempotency_key" IS '命令幂等键；同一作用域内相同键和摘要恢复原结果，不同摘要必须冲突。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."ledger_transactions"."request_id" IS '逻辑引用 UsageRequest 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."ledger_transactions"."reservation_id" IS '逻辑引用 QuotaReservation 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."ledger_transactions"."usage_event_id" IS '逻辑引用 UsageEvent 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."ledger_transactions"."emergency_credit_grant_id" IS '逻辑引用 EmergencyCreditGrant 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."ledger_transactions"."approval_request_id" IS '关联 ApprovalRequest；敏感动作须校验同 Tenant、APPROVED、未过期且 payload_hash 一致。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."ledger_transactions"."reversal_of_transaction_id" IS '逻辑引用 被冲销 LedgerTransaction 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."ledger_transactions"."status" IS '交易入账状态，P0 固定 POSTED；校验失败不创建交易，更正只能新增 ADJUST 或 REVERSAL。 字段约束：必填，默认 ''POSTED''。';
COMMENT ON COLUMN "ai_governance"."ledger_transactions"."business_reason" IS '资金交易的业务依据；拨付、回收、调整、冲销和紧急信用必须可由该说明复核。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."ledger_transactions"."occurred_at" IS 'Ledger Module UTC 业务时间 occurred_at；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."ledger_transactions"."posted_at" IS 'Ledger Module UTC 业务时间 posted_at；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."ledger_transactions"."actor_type" IS 'Ledger Module 受控枚举；允许值：USER（企业用户主体）、ADMIN（治理管理员发起）、SERVICE（服务主体）、SYSTEM（平台系统任务发起）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."ledger_transactions"."actor_id" IS '发起资金交易的用户、管理员、服务或系统主体标识；必须与 actor_type 和 AuditEvent 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."ledger_transactions"."trace_id" IS '端到端 Trace 标识；贯通 Request、Attempt、Usage、Ledger、安全和 Audit。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."ledger_transactions"."audit_event_id" IS '关联统一 AuditEvent；证明该资金或管理事实已进入审计链。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."ledger_transactions"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."ledger_transactions"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."ledger_transactions"."subject_id" IS '交易涉及的消费或受拨付 User Subject；Tenant/Scope 级交易可为空，User 级交易必须由 Posting Interface 写入。';
COMMENT ON TABLE "ai_governance"."ledger_transactions" IS 'AI Credit 复式交易头；写入即 POSTED，只能用新交易更正。';

-- ----------------------------
-- Table structure for model_access_policies
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."model_access_policies";
CREATE TABLE "ai_governance"."model_access_policies" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "policy_code" varchar(64) COLLATE "pg_catalog"."default" NOT NULL,
  "scope_type" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "scope_id" uuid NOT NULL,
  "subject_type" varchar(24) COLLATE "pg_catalog"."default",
  "model_id" uuid NOT NULL,
  "maximum_data_classification" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "required_network_class" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "rps_limit" int4,
  "tps_limit" int8,
  "effect" varchar(8) COLLATE "pg_catalog"."default" NOT NULL,
  "valid_from" timestamptz(6) NOT NULL,
  "valid_until" timestamptz(6),
  "status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "revision" int8 NOT NULL DEFAULT 1,
  "row_version" int8 NOT NULL DEFAULT 1,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "created_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "updated_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "operation_source" varchar(32) COLLATE "pg_catalog"."default" NOT NULL DEFAULT 'ADMIN'::character varying,
  "operation_trace_id" varchar(128) COLLATE "pg_catalog"."default",
  "is_deleted" bool NOT NULL DEFAULT false,
  "deleted_at" timestamptz(6),
  "deleted_by" varchar(128) COLLATE "pg_catalog"."default",
  "delete_reason" varchar(512) COLLATE "pg_catalog"."default"
)
;
COMMENT ON COLUMN "ai_governance"."model_access_policies"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."model_access_policies"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_access_policies"."policy_code" IS '模型授权策略的稳定代码；用于 Scope 授权发布、撤销和审计。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_access_policies"."scope_type" IS 'Model Access 受控枚举；允许值：TENANT（租户作用域）、ORGANIZATION（组织作用域）、PROJECT（项目作用域）、USER（企业用户主体）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_access_policies"."scope_id" IS '逻辑引用 GovernanceScopeRef 指向的领域实体 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_access_policies"."subject_type" IS 'Model Access 受控枚举；允许值：USER（企业用户主体）、SERVICE（服务主体）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."model_access_policies"."model_id" IS '逻辑引用 ModelCatalogEntry 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_access_policies"."maximum_data_classification" IS 'Model Access 受控枚举；允许值：PUBLIC（公开数据）、INTERNAL（内部网络或内部数据）、CONFIDENTIAL（机密数据，仅允许更严格处理）、RESTRICTED（受限数据，仅批准私有模型）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_access_policies"."required_network_class" IS 'Model Access 受控枚举；允许值：INTERNAL（内部网络或内部数据）、EXTERNAL（外部网络或供应商）、ANY（任一条件满足或不限定类别）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_access_policies"."rps_limit" IS '该 ModelGrant 在 Scope 下的可选每秒请求上限；为空表示不附加该层限制。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."model_access_policies"."tps_limit" IS '该 ModelGrant 在 Scope 下的可选每秒 Token 上限；为空表示不附加该层限制。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."model_access_policies"."effect" IS 'Model Access 受控枚举；允许值：ALLOW（允许）、DENY（拒绝，优先于允许）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_access_policies"."valid_from" IS 'Model Access UTC 业务时间 valid_from；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_access_policies"."valid_until" IS 'Model Access UTC 业务时间 valid_until；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."model_access_policies"."status" IS 'Model Access 受控枚举；允许值：DRAFT（草稿未生效）、REVIEWING（待评审）、APPROVED（已批准，尚需发布或生效）、ACTIVE（有效并参与运行决策）、REVOKED（已撤销，不参与新决策）、RETIRED（退役终态，仅保留历史）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_access_policies"."revision" IS '配置业务 revision；每次有效内容变化严格递增，旧 revision 不得覆盖新版本。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."model_access_policies"."row_version" IS '乐观锁版本；每次成功 UPDATE 严格递增，用于防止并发覆盖。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."model_access_policies"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."model_access_policies"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."model_access_policies"."created_by" IS '创建记录的稳定主体标识；保存 ID 或服务主体，不保存仅供展示的姓名。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_access_policies"."updated_by" IS '最后修改记录的稳定主体标识；必须与 AuditEvent.actor 可关联。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_access_policies"."operation_source" IS '变更来源；ADMIN=治理后台，API=管理接口，SYNC=上游同步，SYSTEM=平台任务，MIGRATION=迁移。 字段约束：必填，默认 ''ADMIN''。';
COMMENT ON COLUMN "ai_governance"."model_access_policies"."operation_trace_id" IS '产生该次写入的 Trace 标识；用于跨 Module 联合排障和审计关联。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."model_access_policies"."is_deleted" IS '软删除标志；false=有效，true=逻辑删除且必须同时填写 deleted_at、deleted_by、delete_reason。 字段约束：必填，默认 false。';
COMMENT ON COLUMN "ai_governance"."model_access_policies"."deleted_at" IS '软删除生效时间，UTC；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."model_access_policies"."deleted_by" IS '执行软删除的稳定主体标识；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."model_access_policies"."delete_reason" IS '软删除业务原因；必须说明业务依据、影响范围并关联删除 AuditEvent，禁止空原因或无审计删除。字段约束：可空，仅 is_deleted=true 时必填。';
COMMENT ON TABLE "ai_governance"."model_access_policies" IS 'Scope、主体类型、数据分类和网络条件下的模型授权事实。';

-- ----------------------------
-- Table structure for model_catalog_entries
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."model_catalog_entries";
CREATE TABLE "ai_governance"."model_catalog_entries" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "provider_id" uuid NOT NULL,
  "model_alias" varchar(96) COLLATE "pg_catalog"."default" NOT NULL,
  "actual_model_name" varchar(256) COLLATE "pg_catalog"."default" NOT NULL,
  "model_version" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "capabilities" varchar(32)[] COLLATE "pg_catalog"."default" NOT NULL,
  "network_class" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "max_context_tokens" int8 NOT NULL,
  "max_output_tokens" int8 NOT NULL,
  "data_classification_ceiling" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "revision" int8 NOT NULL DEFAULT 1,
  "row_version" int8 NOT NULL DEFAULT 1,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "created_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "updated_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "operation_source" varchar(32) COLLATE "pg_catalog"."default" NOT NULL DEFAULT 'ADMIN'::character varying,
  "operation_trace_id" varchar(128) COLLATE "pg_catalog"."default",
  "is_deleted" bool NOT NULL DEFAULT false,
  "deleted_at" timestamptz(6),
  "deleted_by" varchar(128) COLLATE "pg_catalog"."default",
  "delete_reason" varchar(512) COLLATE "pg_catalog"."default"
)
;
COMMENT ON COLUMN "ai_governance"."model_catalog_entries"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."model_catalog_entries"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_catalog_entries"."provider_id" IS '逻辑引用 ModelProvider 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_catalog_entries"."model_alias" IS '调用者使用的稳定逻辑模型别名；与实际 Provider 模型解耦，发现和调用均需重新授权。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_catalog_entries"."actual_model_name" IS 'Provider 接口实际接收的模型名称；与稳定 model_alias 分离并锁定 model_version。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_catalog_entries"."model_version" IS 'Provider 实际模型的不可变版本标识；Usage、费率和审计必须锁定调用时版本。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_catalog_entries"."capabilities" IS '模型声明并通过 Adapter 验证的能力集合，例如 CHAT、STREAMING、EMBEDDING；至少一个。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_catalog_entries"."network_class" IS 'Model Catalog 受控枚举；允许值：INTERNAL（内部网络或内部数据）、EXTERNAL（外部网络或供应商）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_catalog_entries"."max_context_tokens" IS 'Model Catalog Token 整数计量 max_context_tokens；单位为 token，禁止浮点和隐式换算，须关联 Request/Usage 或周期投影。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_catalog_entries"."max_output_tokens" IS 'Model Catalog Token 整数计量 max_output_tokens；单位为 token，禁止浮点和隐式换算，须关联 Request/Usage 或周期投影。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_catalog_entries"."data_classification_ceiling" IS 'Model Catalog 受控枚举；允许值：PUBLIC（公开数据）、INTERNAL（内部网络或内部数据）、CONFIDENTIAL（机密数据，仅允许更严格处理）、RESTRICTED（受限数据，仅批准私有模型）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_catalog_entries"."status" IS 'Model Catalog 受控枚举；允许值：DRAFT（草稿未生效）、TESTING（验证中，不进入生产候选）、ACTIVE（有效并参与运行决策）、SUSPENDED（暂停，不接受新增业务）、RETIRED（退役终态，仅保留历史）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_catalog_entries"."revision" IS '配置业务 revision；每次有效内容变化严格递增，旧 revision 不得覆盖新版本。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."model_catalog_entries"."row_version" IS '乐观锁版本；每次成功 UPDATE 严格递增，用于防止并发覆盖。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."model_catalog_entries"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."model_catalog_entries"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."model_catalog_entries"."created_by" IS '创建记录的稳定主体标识；保存 ID 或服务主体，不保存仅供展示的姓名。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_catalog_entries"."updated_by" IS '最后修改记录的稳定主体标识；必须与 AuditEvent.actor 可关联。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_catalog_entries"."operation_source" IS '变更来源；ADMIN=治理后台，API=管理接口，SYNC=上游同步，SYSTEM=平台任务，MIGRATION=迁移。 字段约束：必填，默认 ''ADMIN''。';
COMMENT ON COLUMN "ai_governance"."model_catalog_entries"."operation_trace_id" IS '产生该次写入的 Trace 标识；用于跨 Module 联合排障和审计关联。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."model_catalog_entries"."is_deleted" IS '软删除标志；false=有效，true=逻辑删除且必须同时填写 deleted_at、deleted_by、delete_reason。 字段约束：必填，默认 false。';
COMMENT ON COLUMN "ai_governance"."model_catalog_entries"."deleted_at" IS '软删除生效时间，UTC；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."model_catalog_entries"."deleted_by" IS '执行软删除的稳定主体标识；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."model_catalog_entries"."delete_reason" IS '软删除业务原因；必须说明业务依据、影响范围并关联删除 AuditEvent，禁止空原因或无审计删除。字段约束：可空，仅 is_deleted=true 时必填。';
COMMENT ON TABLE "ai_governance"."model_catalog_entries" IS '模型别名、实际版本、能力、网络和数据级别目录。';

-- ----------------------------
-- Table structure for model_channel_health_events
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."model_channel_health_events";
CREATE TABLE "ai_governance"."model_channel_health_events" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "channel_id" uuid NOT NULL,
  "health_status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "latency_ms" int8,
  "error_rate_basis_points" int4,
  "consecutive_failures" int4 NOT NULL DEFAULT 0,
  "reason_code" varchar(64) COLLATE "pg_catalog"."default" NOT NULL,
  "evidence_json" jsonb NOT NULL DEFAULT '{}'::jsonb,
  "observed_at" timestamptz(6) NOT NULL,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp()
)
;
COMMENT ON COLUMN "ai_governance"."model_channel_health_events"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."model_channel_health_events"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_channel_health_events"."channel_id" IS '逻辑引用 ModelChannel 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_channel_health_events"."health_status" IS 'Model Health 受控枚举；允许值：UNKNOWN（状态未知）、HEALTHY（健康可用）、DEGRADED（降级可用）、UNHEALTHY（不可用）、CIRCUIT_OPEN（熔断开启，排除路由）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_channel_health_events"."latency_ms" IS 'Channel 健康探测或真实观测延迟，单位毫秒且不得为负；用于健康判定和路由排序，采集失败时允许为空。字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."model_channel_health_events"."error_rate_basis_points" IS '错误率，单位基点；0 表示 0%，10000 表示 100%，未知时允许为空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."model_channel_health_events"."consecutive_failures" IS '截至本次观测的连续失败次数；从 0 递增，成功探测后重置。 字段约束：必填，默认 0。';
COMMENT ON COLUMN "ai_governance"."model_channel_health_events"."reason_code" IS '本次 Channel 健康状态变化的稳定原因码；用于触发熔断、告警、半开探测和恢复分析，禁止写自由文本异常。字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_channel_health_events"."evidence_json" IS 'Model Health JSONB 证据或扩展快照 evidence_json；不承载核心可检索关系，值必须为 object。 字段约束：必填，默认 ''{}''::jsonb。';
COMMENT ON COLUMN "ai_governance"."model_channel_health_events"."observed_at" IS 'Model Health UTC 业务时间 observed_at；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_channel_health_events"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."model_channel_health_events"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON TABLE "ai_governance"."model_channel_health_events" IS '模型 Channel 健康观测的追加式事实。';

-- ----------------------------
-- Table structure for model_channels
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."model_channels";
CREATE TABLE "ai_governance"."model_channels" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "model_id" uuid NOT NULL,
  "provider_id" uuid NOT NULL,
  "channel_code" varchar(96) COLLATE "pg_catalog"."default" NOT NULL,
  "endpoint_ref" varchar(1024) COLLATE "pg_catalog"."default" NOT NULL,
  "credential_ref_id" uuid NOT NULL,
  "region" varchar(64) COLLATE "pg_catalog"."default" NOT NULL,
  "network_class" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "capacity_rps" int4 NOT NULL,
  "capacity_tps" int8 NOT NULL,
  "health_status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "revision" int8 NOT NULL DEFAULT 1,
  "row_version" int8 NOT NULL DEFAULT 1,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "created_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "updated_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "operation_source" varchar(32) COLLATE "pg_catalog"."default" NOT NULL DEFAULT 'ADMIN'::character varying,
  "operation_trace_id" varchar(128) COLLATE "pg_catalog"."default",
  "is_deleted" bool NOT NULL DEFAULT false,
  "deleted_at" timestamptz(6),
  "deleted_by" varchar(128) COLLATE "pg_catalog"."default",
  "delete_reason" varchar(512) COLLATE "pg_catalog"."default"
)
;
COMMENT ON COLUMN "ai_governance"."model_channels"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."model_channels"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_channels"."model_id" IS '逻辑引用 ModelCatalogEntry 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_channels"."provider_id" IS '逻辑引用 ModelProvider 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_channels"."channel_code" IS '可路由 Channel 的稳定代码；唯一定位模型、区域、端点引用和凭证组合。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_channels"."endpoint_ref" IS 'Model Catalog 外部安全引用 endpoint_ref；仅保存 Vault/KMS/对象定位符，不保存 Secret、凭证或对象正文。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_channels"."credential_ref_id" IS '逻辑引用 ProviderCredentialRef 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_channels"."region" IS 'Channel 实际部署或数据处理区域代码；路由必须满足 Tenant 数据驻留和出境约束。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_channels"."network_class" IS 'Model Catalog 受控枚举；允许值：INTERNAL（内部网络或内部数据）、EXTERNAL（外部网络或供应商）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_channels"."capacity_rps" IS 'Channel 标称每秒请求容量；单位 request/second，用于路由容量过滤并须大于 0。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_channels"."capacity_tps" IS 'Channel 标称每秒 Token 容量；单位 token/second，用于路由容量过滤并须大于 0。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_channels"."health_status" IS 'Model Catalog 受控枚举；允许值：UNKNOWN（状态未知）、HEALTHY（健康可用）、DEGRADED（降级可用）、UNHEALTHY（不可用）、CIRCUIT_OPEN（熔断开启，排除路由）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_channels"."status" IS 'Model Catalog 受控枚举；允许值：DRAFT（草稿未生效）、ACTIVE（有效并参与运行决策）、SUSPENDED（暂停，不接受新增业务）、RETIRED（退役终态，仅保留历史）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_channels"."revision" IS '配置业务 revision；每次有效内容变化严格递增，旧 revision 不得覆盖新版本。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."model_channels"."row_version" IS '乐观锁版本；每次成功 UPDATE 严格递增，用于防止并发覆盖。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."model_channels"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."model_channels"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."model_channels"."created_by" IS '创建记录的稳定主体标识；保存 ID 或服务主体，不保存仅供展示的姓名。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_channels"."updated_by" IS '最后修改记录的稳定主体标识；必须与 AuditEvent.actor 可关联。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_channels"."operation_source" IS '变更来源；ADMIN=治理后台，API=管理接口，SYNC=上游同步，SYSTEM=平台任务，MIGRATION=迁移。 字段约束：必填，默认 ''ADMIN''。';
COMMENT ON COLUMN "ai_governance"."model_channels"."operation_trace_id" IS '产生该次写入的 Trace 标识；用于跨 Module 联合排障和审计关联。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."model_channels"."is_deleted" IS '软删除标志；false=有效，true=逻辑删除且必须同时填写 deleted_at、deleted_by、delete_reason。 字段约束：必填，默认 false。';
COMMENT ON COLUMN "ai_governance"."model_channels"."deleted_at" IS '软删除生效时间，UTC；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."model_channels"."deleted_by" IS '执行软删除的稳定主体标识；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."model_channels"."delete_reason" IS '软删除业务原因；必须说明业务依据、影响范围并关联删除 AuditEvent，禁止空原因或无审计删除。字段约束：可空，仅 is_deleted=true 时必填。';
COMMENT ON TABLE "ai_governance"."model_channels" IS '路由实际执行粒度；模型、端点引用、凭证引用、区域和健康状态。';

-- ----------------------------
-- Table structure for model_providers
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."model_providers";
CREATE TABLE "ai_governance"."model_providers" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "provider_code" varchar(64) COLLATE "pg_catalog"."default" NOT NULL,
  "provider_name" varchar(160) COLLATE "pg_catalog"."default" NOT NULL,
  "provider_type" varchar(32) COLLATE "pg_catalog"."default" NOT NULL,
  "network_class" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "regions" varchar(64)[] COLLATE "pg_catalog"."default" NOT NULL,
  "protocol_type" varchar(32) COLLATE "pg_catalog"."default" NOT NULL,
  "adapter_key" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "revision" int8 NOT NULL DEFAULT 1,
  "row_version" int8 NOT NULL DEFAULT 1,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "created_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "updated_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "operation_source" varchar(32) COLLATE "pg_catalog"."default" NOT NULL DEFAULT 'ADMIN'::character varying,
  "operation_trace_id" varchar(128) COLLATE "pg_catalog"."default",
  "is_deleted" bool NOT NULL DEFAULT false,
  "deleted_at" timestamptz(6),
  "deleted_by" varchar(128) COLLATE "pg_catalog"."default",
  "delete_reason" varchar(512) COLLATE "pg_catalog"."default"
)
;
COMMENT ON COLUMN "ai_governance"."model_providers"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."model_providers"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_providers"."provider_code" IS 'Provider 的稳定注册代码；Adapter、模型和报表引用该值，供应商改名不影响。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_providers"."provider_name" IS 'Provider 展示名称；路由和 Adapter 绑定使用 provider_id、provider_code。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_providers"."provider_type" IS 'Model Catalog 受控枚举；允许值：INTERNAL（内部网络或内部数据）、EXTERNAL（外部网络或供应商）、HYBRID（同时具有内外部能力）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_providers"."network_class" IS 'Model Catalog 受控枚举；允许值：INTERNAL（内部网络或内部数据）、EXTERNAL（外部网络或供应商）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_providers"."regions" IS 'Provider 可处理数据的区域代码集合；至少一个，路由须与数据驻留政策求交。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_providers"."protocol_type" IS 'Model Catalog 受控枚举；允许值：OPENAI_COMPATIBLE（OpenAI 兼容协议）、AZURE_OPENAI（Azure OpenAI 协议）、CUSTOM_ADAPTER（自定义 Provider Adapter）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_providers"."adapter_key" IS 'Provider Adapter Registry 的受控实现键；必须存在真实 Implementation 和契约测试，禁止任意类名反射。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_providers"."status" IS 'Model Catalog 受控枚举；允许值：DRAFT（草稿未生效）、ACTIVE（有效并参与运行决策）、DEGRADED（降级可用）、SUSPENDED（暂停，不接受新增业务）、RETIRED（退役终态，仅保留历史）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_providers"."revision" IS '配置业务 revision；每次有效内容变化严格递增，旧 revision 不得覆盖新版本。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."model_providers"."row_version" IS '乐观锁版本；每次成功 UPDATE 严格递增，用于防止并发覆盖。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."model_providers"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."model_providers"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."model_providers"."created_by" IS '创建记录的稳定主体标识；保存 ID 或服务主体，不保存仅供展示的姓名。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_providers"."updated_by" IS '最后修改记录的稳定主体标识；必须与 AuditEvent.actor 可关联。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."model_providers"."operation_source" IS '变更来源；ADMIN=治理后台，API=管理接口，SYNC=上游同步，SYSTEM=平台任务，MIGRATION=迁移。 字段约束：必填，默认 ''ADMIN''。';
COMMENT ON COLUMN "ai_governance"."model_providers"."operation_trace_id" IS '产生该次写入的 Trace 标识；用于跨 Module 联合排障和审计关联。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."model_providers"."is_deleted" IS '软删除标志；false=有效，true=逻辑删除且必须同时填写 deleted_at、deleted_by、delete_reason。 字段约束：必填，默认 false。';
COMMENT ON COLUMN "ai_governance"."model_providers"."deleted_at" IS '软删除生效时间，UTC；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."model_providers"."deleted_by" IS '执行软删除的稳定主体标识；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."model_providers"."delete_reason" IS '软删除业务原因；必须说明业务依据、影响范围并关联删除 AuditEvent，禁止空原因或无审计删除。字段约束：可空，仅 is_deleted=true 时必填。';
COMMENT ON TABLE "ai_governance"."model_providers" IS '模型供应商与 Provider Adapter 注册事实。';

-- ----------------------------
-- Table structure for oauth_provider_configs
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."oauth_provider_configs";
CREATE TABLE "ai_governance"."oauth_provider_configs" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "provider_code" varchar(64) COLLATE "pg_catalog"."default" NOT NULL,
  "provider_type" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "client_id" varchar(256) COLLATE "pg_catalog"."default" NOT NULL,
  "client_secret_ref" varchar(512) COLLATE "pg_catalog"."default" NOT NULL,
  "authorization_endpoint" varchar(1024) COLLATE "pg_catalog"."default" NOT NULL,
  "token_endpoint" varchar(1024) COLLATE "pg_catalog"."default" NOT NULL,
  "userinfo_endpoint" varchar(1024) COLLATE "pg_catalog"."default",
  "requested_scopes" varchar(128)[] COLLATE "pg_catalog"."default" NOT NULL DEFAULT ARRAY[]::character varying[],
  "redirect_allowlist" varchar(1024)[] COLLATE "pg_catalog"."default" NOT NULL DEFAULT ARRAY[]::character varying[],
  "status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "revision" int8 NOT NULL DEFAULT 1,
  "row_version" int8 NOT NULL DEFAULT 1,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "created_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "updated_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "operation_source" varchar(32) COLLATE "pg_catalog"."default" NOT NULL DEFAULT 'ADMIN'::character varying,
  "operation_trace_id" varchar(128) COLLATE "pg_catalog"."default",
  "is_deleted" bool NOT NULL DEFAULT false,
  "deleted_at" timestamptz(6),
  "deleted_by" varchar(128) COLLATE "pg_catalog"."default",
  "delete_reason" varchar(512) COLLATE "pg_catalog"."default"
)
;
COMMENT ON COLUMN "ai_governance"."oauth_provider_configs"."id" IS '应用生成的 UUIDv7 OAuth Provider 配置主键；不暴露数据库顺序。';
COMMENT ON COLUMN "ai_governance"."oauth_provider_configs"."tenant_id" IS 'OAuth Provider 所属 Tenant；不同 Tenant 的 Client 与 Secret 必须完全隔离。';
COMMENT ON COLUMN "ai_governance"."oauth_provider_configs"."provider_code" IS 'Tenant 内稳定 Provider 代码，例如 wechat-work；用于 API 路由和配置引用。';
COMMENT ON COLUMN "ai_governance"."oauth_provider_configs"."provider_type" IS 'Provider 类型；WECHAT 表示微信 OAuth，OIDC 表示标准 OpenID Connect。';
COMMENT ON COLUMN "ai_governance"."oauth_provider_configs"."client_id" IS '第三方分配的公开 Client 标识；不包含 Client Secret，变更需发布新 revision。';
COMMENT ON COLUMN "ai_governance"."oauth_provider_configs"."client_secret_ref" IS 'OAuth Client Secret 的 KMS/Vault 引用；数据库、日志和前端禁止保存原文。';
COMMENT ON COLUMN "ai_governance"."oauth_provider_configs"."authorization_endpoint" IS '经过安全审核的授权端点；只允许 HTTPS 和 Provider allowlist 地址。';
COMMENT ON COLUMN "ai_governance"."oauth_provider_configs"."token_endpoint" IS '经过安全审核的 Token 交换端点；只由服务端 Adapter 访问。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."oauth_provider_configs"."userinfo_endpoint" IS '可选用户信息端点；空值表示 Provider 通过 Token 响应返回所需主体声明。';
COMMENT ON COLUMN "ai_governance"."oauth_provider_configs"."requested_scopes" IS '授权时申请的最小 Scope 集合；禁止通过请求参数扩大已发布集合。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."oauth_provider_configs"."redirect_allowlist" IS '允许的精确回调地址集合；禁止通配符、任意 URL 和请求侧覆盖。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."oauth_provider_configs"."status" IS '配置状态；DRAFT 草稿，ACTIVE 可认证，SUSPENDED 暂停，RETIRED 永久退役。';
COMMENT ON COLUMN "ai_governance"."oauth_provider_configs"."revision" IS 'Provider 配置单调版本；端点、Scope、Secret 引用或状态变化后递增。';
COMMENT ON COLUMN "ai_governance"."oauth_provider_configs"."row_version" IS 'GORM 乐观锁版本；避免配置审批和轮换的并发覆盖。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."oauth_provider_configs"."created_at" IS 'Provider 配置首次创建时间；数据库生成且后续不可改写。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."oauth_provider_configs"."updated_at" IS 'Provider 配置最近更新时间；由触发器统一刷新。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."oauth_provider_configs"."created_by" IS '创建 Provider 配置的管理员主体；必须关联审批和 AuditEvent。';
COMMENT ON COLUMN "ai_governance"."oauth_provider_configs"."updated_by" IS '最近修改 Provider 配置的管理员主体；必须关联审批和 AuditEvent。';
COMMENT ON COLUMN "ai_governance"."oauth_provider_configs"."operation_source" IS '配置变更来源，默认 ADMIN；用于运维审计但不替代 actor。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."oauth_provider_configs"."operation_trace_id" IS '配置变更 Trace ID；关联审批、发布和 RuntimeSnapshot。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."oauth_provider_configs"."is_deleted" IS 'Provider 配置软删除标志；true 时不再产生新 OAuth 认证。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."oauth_provider_configs"."deleted_at" IS 'Provider 配置软删除时间；未删除时为空且禁止物理删除。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."oauth_provider_configs"."deleted_by" IS '执行 Provider 配置软删除的管理员标识；未删除时为空。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."oauth_provider_configs"."delete_reason" IS 'Provider 配置软删除原因；删除时必填并进入管理审计。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON TABLE "ai_governance"."oauth_provider_configs" IS 'Tenant 级第三方 OAuth Provider 配置；P0 支持 WeChat 与标准 OIDC，Secret 仅保存 KMS/Vault 引用。';

-- ----------------------------
-- Table structure for organization_memberships
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."organization_memberships";
CREATE TABLE "ai_governance"."organization_memberships" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "subject_id" uuid NOT NULL,
  "org_role_code" varchar(64) COLLATE "pg_catalog"."default" NOT NULL,
  "is_primary" bool NOT NULL DEFAULT false,
  "status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "valid_from" timestamptz(6) NOT NULL,
  "valid_until" timestamptz(6),
  "membership_revision" int8 NOT NULL DEFAULT 1,
  "row_version" int8 NOT NULL DEFAULT 1,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "created_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "updated_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "operation_source" varchar(32) COLLATE "pg_catalog"."default" NOT NULL DEFAULT 'SYNC'::character varying,
  "operation_trace_id" varchar(128) COLLATE "pg_catalog"."default",
  "is_deleted" bool NOT NULL DEFAULT false,
  "deleted_at" timestamptz(6),
  "deleted_by" varchar(128) COLLATE "pg_catalog"."default",
  "delete_reason" varchar(512) COLLATE "pg_catalog"."default"
)
;
COMMENT ON COLUMN "ai_governance"."organization_memberships"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."organization_memberships"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."organization_memberships"."organization_id" IS '逻辑引用 Organization 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."organization_memberships"."subject_id" IS '逻辑引用 IdentitySubject 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."organization_memberships"."org_role_code" IS '成员在该 Organization 内的领域角色代码；由 Org Center 定义，Authorization 只映射动作不复制绑定。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."organization_memberships"."is_primary" IS '是否为用户唯一 ACTIVE 主叶子组织；true 时同 Tenant/Subject 只能有一个。 字段约束：必填，默认 false。';
COMMENT ON COLUMN "ai_governance"."organization_memberships"."status" IS 'Org Center 受控枚举；允许值：PENDING（待处理或待生效）、ACTIVE（有效并参与运行决策）、SUSPENDED（暂停，不接受新增业务）、ENDED（关系结束终态）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."organization_memberships"."valid_from" IS 'Org Center UTC 业务时间 valid_from；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."organization_memberships"."valid_until" IS 'Org Center UTC 业务时间 valid_until；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."organization_memberships"."membership_revision" IS 'Org Center 不可回退版本 membership_revision；发布、决策和历史事实必须锁定当时值。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."organization_memberships"."row_version" IS '乐观锁版本；每次成功 UPDATE 严格递增，用于防止并发覆盖。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."organization_memberships"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."organization_memberships"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."organization_memberships"."created_by" IS '创建记录的稳定主体标识；保存 ID 或服务主体，不保存仅供展示的姓名。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."organization_memberships"."updated_by" IS '最后修改记录的稳定主体标识；必须与 AuditEvent.actor 可关联。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."organization_memberships"."operation_source" IS '变更来源；ADMIN=治理后台，API=管理接口，SYNC=上游同步，SYSTEM=平台任务，MIGRATION=迁移。 字段约束：必填，默认 ''SYNC''。';
COMMENT ON COLUMN "ai_governance"."organization_memberships"."operation_trace_id" IS '产生该次写入的 Trace 标识；用于跨 Module 联合排障和审计关联。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."organization_memberships"."is_deleted" IS '软删除标志；false=有效，true=逻辑删除且必须同时填写 deleted_at、deleted_by、delete_reason。 字段约束：必填，默认 false。';
COMMENT ON COLUMN "ai_governance"."organization_memberships"."deleted_at" IS '软删除生效时间，UTC；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."organization_memberships"."deleted_by" IS '执行软删除的稳定主体标识；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."organization_memberships"."delete_reason" IS '软删除业务原因；必须说明业务依据、影响范围并关联删除 AuditEvent，禁止空原因或无审计删除。字段约束：可空，仅 is_deleted=true 时必填。';
COMMENT ON TABLE "ai_governance"."organization_memberships" IS '组织成员和组织域角色唯一事实。';

-- ----------------------------
-- Table structure for organizations
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."organizations";
CREATE TABLE "ai_governance"."organizations" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "parent_organization_id" uuid,
  "organization_code" varchar(64) COLLATE "pg_catalog"."default" NOT NULL,
  "organization_name" varchar(160) COLLATE "pg_catalog"."default" NOT NULL,
  "organization_path" varchar(1024) COLLATE "pg_catalog"."default" NOT NULL,
  "organization_type" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "depth" int4 NOT NULL,
  "status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "revision" int8 NOT NULL DEFAULT 1,
  "row_version" int8 NOT NULL DEFAULT 1,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "created_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "updated_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "operation_source" varchar(32) COLLATE "pg_catalog"."default" NOT NULL DEFAULT 'SYNC'::character varying,
  "operation_trace_id" varchar(128) COLLATE "pg_catalog"."default",
  "is_deleted" bool NOT NULL DEFAULT false,
  "deleted_at" timestamptz(6),
  "deleted_by" varchar(128) COLLATE "pg_catalog"."default",
  "delete_reason" varchar(512) COLLATE "pg_catalog"."default"
)
;
COMMENT ON COLUMN "ai_governance"."organizations"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."organizations"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."organizations"."parent_organization_id" IS '逻辑引用 父 Organization 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."organizations"."organization_code" IS '企业 IAM 或 Org Center 分配的稳定组织代码；组织改名或移动不改变该值。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."organizations"."organization_name" IS '组织正式名称；允许随 IAM 同步改名，organization_code 和 id 保持稳定。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."organizations"."organization_path" IS '组织树的规范化物化路径；用于子树查询和循环检测，节点移动必须原子更新后代路径与 revision。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."organizations"."organization_type" IS 'Org Center 受控枚举；允许值：COMPANY（公司根组织）、DIVISION（事业部或分支组织节点）、DEPARTMENT（部门组织节点）、TEAM（团队组织节点）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."organizations"."depth" IS '组织节点相对 Tenant 根的层级深度；根为 0，最大 32，并须与 parent 和 organization_path 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."organizations"."status" IS 'Org Center 受控枚举；允许值：DRAFT（草稿未生效）、ACTIVE（有效并参与运行决策）、SUSPENDED（暂停，不接受新增业务）、ARCHIVED（归档终态）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."organizations"."revision" IS '配置业务 revision；每次有效内容变化严格递增，旧 revision 不得覆盖新版本。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."organizations"."row_version" IS '乐观锁版本；每次成功 UPDATE 严格递增，用于防止并发覆盖。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."organizations"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."organizations"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."organizations"."created_by" IS '创建记录的稳定主体标识；保存 ID 或服务主体，不保存仅供展示的姓名。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."organizations"."updated_by" IS '最后修改记录的稳定主体标识；必须与 AuditEvent.actor 可关联。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."organizations"."operation_source" IS '变更来源；ADMIN=治理后台，API=管理接口，SYNC=上游同步，SYSTEM=平台任务，MIGRATION=迁移。 字段约束：必填，默认 ''SYNC''。';
COMMENT ON COLUMN "ai_governance"."organizations"."operation_trace_id" IS '产生该次写入的 Trace 标识；用于跨 Module 联合排障和审计关联。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."organizations"."is_deleted" IS '软删除标志；false=有效，true=逻辑删除且必须同时填写 deleted_at、deleted_by、delete_reason。 字段约束：必填，默认 false。';
COMMENT ON COLUMN "ai_governance"."organizations"."deleted_at" IS '软删除生效时间，UTC；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."organizations"."deleted_by" IS '执行软删除的稳定主体标识；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."organizations"."delete_reason" IS '软删除业务原因；必须说明业务依据、影响范围并关联删除 AuditEvent，禁止空原因或无审计删除。字段约束：可空，仅 is_deleted=true 时必填。';
COMMENT ON TABLE "ai_governance"."organizations" IS '组织树主数据与生命周期事实。';

-- ----------------------------
-- Table structure for outbox_events
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."outbox_events";
CREATE TABLE "ai_governance"."outbox_events" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "event_id" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "aggregate_type" varchar(64) COLLATE "pg_catalog"."default" NOT NULL,
  "aggregate_id" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "aggregate_revision" int8 NOT NULL,
  "event_type" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "schema_version" varchar(16) COLLATE "pg_catalog"."default" NOT NULL,
  "payload_json" jsonb NOT NULL,
  "status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "available_at" timestamptz(6) NOT NULL,
  "lease_owner" varchar(128) COLLATE "pg_catalog"."default",
  "lease_until" timestamptz(6),
  "attempt_count" int4 NOT NULL DEFAULT 0,
  "last_error_code" varchar(128) COLLATE "pg_catalog"."default",
  "published_at" timestamptz(6),
  "row_version" int8 NOT NULL DEFAULT 1,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "trace_id" varchar(128) COLLATE "pg_catalog"."default" NOT NULL
)
;
COMMENT ON COLUMN "ai_governance"."outbox_events"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."outbox_events"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."outbox_events"."event_id" IS '可靠发布事件的全局幂等标识；消费者以 Tenant、consumer、event_id 去重。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."outbox_events"."aggregate_type" IS '产生事件的领域聚合类型；消费者据此选择 Schema 和处理器，不允许任意反射加载。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."outbox_events"."aggregate_id" IS '产生事件的领域聚合稳定标识；与 aggregate_type、revision 共同保证顺序和幂等。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."outbox_events"."aggregate_revision" IS 'Reliable Eventing 不可回退版本 aggregate_revision；发布、决策和历史事实必须锁定当时值。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."outbox_events"."event_type" IS '领域事件类型代码；决定消费者处理语义和 Schema，不允许用聚合名称隐式推断。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."outbox_events"."schema_version" IS 'Outbox payload 的 Schema 版本；破坏性变化升级主版本，消费者对未知控制字段默认拒绝。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."outbox_events"."payload_json" IS 'Reliable Eventing JSONB 证据或扩展快照 payload_json；不承载核心可检索关系，值必须为 object。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."outbox_events"."status" IS 'Reliable Eventing 受控枚举；允许值：PENDING（待处理或待生效）、PROCESSING（处理中并持有租约）、PUBLISHED（已发布但未必激活）、FAILED（执行失败）、DEAD_LETTER（超过重试阈值，进入人工处置）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."outbox_events"."available_at" IS 'Reliable Eventing UTC 业务时间 available_at；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."outbox_events"."lease_owner" IS '当前处理 Outbox 的 Worker 实例标识；仅 PROCESSING 状态非空，租约超时后可被其他实例接管。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."outbox_events"."lease_until" IS 'Reliable Eventing UTC 业务时间 lease_until；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."outbox_events"."attempt_count" IS 'Reliable Eventing 顺序、计数或重放控制字段 attempt_count；用于确定性排序、幂等、投影或完整性验证，禁止无审计跳变。 字段约束：必填，默认 0。';
COMMENT ON COLUMN "ai_governance"."outbox_events"."last_error_code" IS '最近一次发布失败的稳定错误码；成功或尚未失败时为空，禁止保存异常堆栈。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."outbox_events"."published_at" IS 'Reliable Eventing UTC 业务时间 published_at；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."outbox_events"."row_version" IS '乐观锁版本；每次成功 UPDATE 严格递增，用于防止并发覆盖。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."outbox_events"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."outbox_events"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."outbox_events"."trace_id" IS '端到端 Trace 标识；贯通 Request、Attempt、Usage、Ledger、安全和 Audit。 字段约束：必填。';
COMMENT ON TABLE "ai_governance"."outbox_events" IS '与领域事实同事务写入的可靠事件发布记录。';

-- ----------------------------
-- Table structure for project_memberships
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."project_memberships";
CREATE TABLE "ai_governance"."project_memberships" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "project_id" uuid NOT NULL,
  "subject_id" uuid NOT NULL,
  "source_organization_id" uuid NOT NULL,
  "project_role_code" varchar(64) COLLATE "pg_catalog"."default" NOT NULL,
  "status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "valid_from" timestamptz(6) NOT NULL,
  "valid_until" timestamptz(6),
  "membership_revision" int8 NOT NULL DEFAULT 1,
  "row_version" int8 NOT NULL DEFAULT 1,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "created_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "updated_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "operation_source" varchar(32) COLLATE "pg_catalog"."default" NOT NULL DEFAULT 'ADMIN'::character varying,
  "operation_trace_id" varchar(128) COLLATE "pg_catalog"."default",
  "is_deleted" bool NOT NULL DEFAULT false,
  "deleted_at" timestamptz(6),
  "deleted_by" varchar(128) COLLATE "pg_catalog"."default",
  "delete_reason" varchar(512) COLLATE "pg_catalog"."default"
)
;
COMMENT ON COLUMN "ai_governance"."project_memberships"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."project_memberships"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."project_memberships"."project_id" IS '逻辑引用 Project 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."project_memberships"."subject_id" IS '逻辑引用 IdentitySubject 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."project_memberships"."source_organization_id" IS '逻辑引用 成员来源 Organization 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."project_memberships"."project_role_code" IS '成员在该 Project 内的领域角色代码；仅 ACTIVE Membership 贡献项目权限。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."project_memberships"."status" IS 'Project Center 受控枚举；允许值：INVITED（已邀请未激活）、ACTIVE（有效并参与运行决策）、SUSPENDED（暂停，不接受新增业务）、REMOVED（成员移除终态）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."project_memberships"."valid_from" IS 'Project Center UTC 业务时间 valid_from；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."project_memberships"."valid_until" IS 'Project Center UTC 业务时间 valid_until；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."project_memberships"."membership_revision" IS 'Project Center 不可回退版本 membership_revision；发布、决策和历史事实必须锁定当时值。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."project_memberships"."row_version" IS '乐观锁版本；每次成功 UPDATE 严格递增，用于防止并发覆盖。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."project_memberships"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."project_memberships"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."project_memberships"."created_by" IS '创建记录的稳定主体标识；保存 ID 或服务主体，不保存仅供展示的姓名。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."project_memberships"."updated_by" IS '最后修改记录的稳定主体标识；必须与 AuditEvent.actor 可关联。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."project_memberships"."operation_source" IS '变更来源；ADMIN=治理后台，API=管理接口，SYNC=上游同步，SYSTEM=平台任务，MIGRATION=迁移。 字段约束：必填，默认 ''ADMIN''。';
COMMENT ON COLUMN "ai_governance"."project_memberships"."operation_trace_id" IS '产生该次写入的 Trace 标识；用于跨 Module 联合排障和审计关联。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."project_memberships"."is_deleted" IS '软删除标志；false=有效，true=逻辑删除且必须同时填写 deleted_at、deleted_by、delete_reason。 字段约束：必填，默认 false。';
COMMENT ON COLUMN "ai_governance"."project_memberships"."deleted_at" IS '软删除生效时间，UTC；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."project_memberships"."deleted_by" IS '执行软删除的稳定主体标识；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."project_memberships"."delete_reason" IS '软删除业务原因；必须说明业务依据、影响范围并关联删除 AuditEvent，禁止空原因或无审计删除。字段约束：可空，仅 is_deleted=true 时必填。';
COMMENT ON TABLE "ai_governance"."project_memberships" IS '项目成员和项目域角色唯一事实。';

-- ----------------------------
-- Table structure for projection_checkpoints
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."projection_checkpoints";
CREATE TABLE "ai_governance"."projection_checkpoints" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "projection_code" varchar(96) COLLATE "pg_catalog"."default" NOT NULL,
  "partition_key" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "source_watermark" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "generation" int8 NOT NULL,
  "status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "last_event_id" varchar(128) COLLATE "pg_catalog"."default",
  "last_event_at" timestamptz(6),
  "error_code" varchar(128) COLLATE "pg_catalog"."default",
  "row_version" int8 NOT NULL DEFAULT 1,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp()
)
;
COMMENT ON COLUMN "ai_governance"."projection_checkpoints"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."projection_checkpoints"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."projection_checkpoints"."projection_code" IS '可重建投影的稳定代码；与 partition_key 共同维护独立 watermark 和 generation。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."projection_checkpoints"."partition_key" IS '投影分区键，例如 Tenant、Scope 或时间分片；与 projection_code 共同维护独立水位。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."projection_checkpoints"."source_watermark" IS '投影已消费的权威事实水位；用于重建、追赶和新 generation 切换。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."projection_checkpoints"."generation" IS '投影或 RuntimeSnapshot 的重建代次；新 generation 完成全量校验后原子切换，旧 generation 不得覆盖或与新代混读。字段约束：必填且大于 0。';
COMMENT ON COLUMN "ai_governance"."projection_checkpoints"."status" IS 'Projection Governance 受控枚举；允许值：BUILDING（构建中）、CURRENT（投影与事实水位一致）、STALE（投影已落后）、FAILED（执行失败）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."projection_checkpoints"."last_event_id" IS '投影成功应用的最后事件标识；用于重启续跑、重复检测和水位审计。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."projection_checkpoints"."last_event_at" IS 'Projection Governance UTC 业务时间 last_event_at；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."projection_checkpoints"."error_code" IS '投影构建、重放或追赶失败的稳定错误码；用于告警和新 generation 重建，CURRENT 状态必须为空。字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."projection_checkpoints"."row_version" IS '乐观锁版本；每次成功 UPDATE 严格递增，用于防止并发覆盖。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."projection_checkpoints"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."projection_checkpoints"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON TABLE "ai_governance"."projection_checkpoints" IS '可重建投影的 watermark 和 generation。';

-- ----------------------------
-- Table structure for projects
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."projects";
CREATE TABLE "ai_governance"."projects" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "project_code" varchar(64) COLLATE "pg_catalog"."default" NOT NULL,
  "project_name" varchar(160) COLLATE "pg_catalog"."default" NOT NULL,
  "owner_subject_id" uuid NOT NULL,
  "leader_subject_id" uuid NOT NULL,
  "sponsor_organization_id" uuid NOT NULL,
  "purpose" varchar(512) COLLATE "pg_catalog"."default" NOT NULL,
  "data_classification" varchar(24) COLLATE "pg_catalog"."default" NOT NULL DEFAULT 'INTERNAL'::character varying,
  "status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "valid_from" timestamptz(6),
  "valid_until" timestamptz(6),
  "revision" int8 NOT NULL DEFAULT 1,
  "row_version" int8 NOT NULL DEFAULT 1,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "created_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "updated_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "operation_source" varchar(32) COLLATE "pg_catalog"."default" NOT NULL DEFAULT 'ADMIN'::character varying,
  "operation_trace_id" varchar(128) COLLATE "pg_catalog"."default",
  "is_deleted" bool NOT NULL DEFAULT false,
  "deleted_at" timestamptz(6),
  "deleted_by" varchar(128) COLLATE "pg_catalog"."default",
  "delete_reason" varchar(512) COLLATE "pg_catalog"."default"
)
;
COMMENT ON COLUMN "ai_governance"."projects"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."projects"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."projects"."project_code" IS 'Project 的稳定业务代码；用于治理、资金和报表检索，归档后不得被新项目复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."projects"."project_name" IS 'Project 展示名称；允许治理变更，不替代 project_code、id 或资金 Scope。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."projects"."owner_subject_id" IS '逻辑引用 Owner IdentitySubject 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."projects"."leader_subject_id" IS '逻辑引用 Project Leader IdentitySubject 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."projects"."sponsor_organization_id" IS '逻辑引用 Project 发起 Organization 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."projects"."purpose" IS 'Project 的获批业务目的；用于权限、安全、资金审批和审计解释，目的实质变化必须重新审批。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."projects"."data_classification" IS 'Project Center 受控枚举；允许值：PUBLIC（公开数据）、INTERNAL（内部网络或内部数据）、CONFIDENTIAL（机密数据，仅允许更严格处理）、RESTRICTED（受限数据，仅批准私有模型）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填，默认 ''INTERNAL''。';
COMMENT ON COLUMN "ai_governance"."projects"."status" IS 'Project Center 受控枚举；允许值：DRAFT（草稿未生效）、ACTIVE（有效并参与运行决策）、PAUSED（暂停新调用和拨付）、CLOSING（结项收敛，仅允许结算、回收和查询）、ARCHIVED（归档终态）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."projects"."valid_from" IS 'Project Center UTC 业务时间 valid_from；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."projects"."valid_until" IS 'Project Center UTC 业务时间 valid_until；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."projects"."revision" IS '配置业务 revision；每次有效内容变化严格递增，旧 revision 不得覆盖新版本。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."projects"."row_version" IS '乐观锁版本；每次成功 UPDATE 严格递增，用于防止并发覆盖。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."projects"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."projects"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."projects"."created_by" IS '创建记录的稳定主体标识；保存 ID 或服务主体，不保存仅供展示的姓名。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."projects"."updated_by" IS '最后修改记录的稳定主体标识；必须与 AuditEvent.actor 可关联。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."projects"."operation_source" IS '变更来源；ADMIN=治理后台，API=管理接口，SYNC=上游同步，SYSTEM=平台任务，MIGRATION=迁移。 字段约束：必填，默认 ''ADMIN''。';
COMMENT ON COLUMN "ai_governance"."projects"."operation_trace_id" IS '产生该次写入的 Trace 标识；用于跨 Module 联合排障和审计关联。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."projects"."is_deleted" IS '软删除标志；false=有效，true=逻辑删除且必须同时填写 deleted_at、deleted_by、delete_reason。 字段约束：必填，默认 false。';
COMMENT ON COLUMN "ai_governance"."projects"."deleted_at" IS '软删除生效时间，UTC；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."projects"."deleted_by" IS '执行软删除的稳定主体标识；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."projects"."delete_reason" IS '软删除业务原因；必须说明业务依据、影响范围并关联删除 AuditEvent，禁止空原因或无审计删除。字段约束：可空，仅 is_deleted=true 时必填。';
COMMENT ON TABLE "ai_governance"."projects" IS '项目治理、有效期和生命周期唯一事实。';

-- ----------------------------
-- Table structure for prompt_firewall_decisions
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."prompt_firewall_decisions";
CREATE TABLE "ai_governance"."prompt_firewall_decisions" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "request_id" uuid NOT NULL,
  "decision_sequence" int2 NOT NULL,
  "direction" varchar(16) COLLATE "pg_catalog"."default" NOT NULL,
  "policy_id" uuid NOT NULL,
  "policy_revision" int8 NOT NULL,
  "action" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "effective_data_classification" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "reason_codes" varchar(64)[] COLLATE "pg_catalog"."default" NOT NULL DEFAULT ARRAY[]::character varying[],
  "finding_summary_json" jsonb NOT NULL DEFAULT '{}'::jsonb,
  "content_hash" varchar(71) COLLATE "pg_catalog"."default" NOT NULL,
  "evaluated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp()
)
;
COMMENT ON COLUMN "ai_governance"."prompt_firewall_decisions"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_decisions"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_decisions"."request_id" IS '逻辑引用 UsageRequest 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_decisions"."decision_sequence" IS 'Prompt Firewall 顺序、计数或重放控制字段 decision_sequence；用于确定性排序、幂等、投影或完整性验证，禁止无审计跳变。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_decisions"."direction" IS 'Prompt Firewall 受控枚举；允许值：INPUT（输入方向）、OUTPUT（输出方向）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_decisions"."policy_id" IS '逻辑引用 策略主记录 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_decisions"."policy_revision" IS '策略决策时锁定的不可变 revision；用于历史重放、差异分析和审计证明，旧 revision 不得覆盖新决策。字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_decisions"."action" IS 'Prompt Firewall 受控枚举；允许值：ALLOW（允许）、BLOCK（阻断处理）、REDACT（脱敏后继续）、FORCE_INTERNAL（强制仅使用内网模型）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_decisions"."effective_data_classification" IS 'Prompt Firewall 受控枚举；允许值：PUBLIC（公开数据）、INTERNAL（内部网络或内部数据）、CONFIDENTIAL（机密数据，仅允许更严格处理）、RESTRICTED（受限数据，仅批准私有模型）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_decisions"."reason_codes" IS '安全决策命中的稳定原因码集合；不保存敏感正文，用于解释 BLOCK、REDACT 或 FORCE_INTERNAL。 字段约束：必填，默认 ARRAY[]::varchar[]。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_decisions"."finding_summary_json" IS 'Prompt Firewall JSONB 证据或扩展快照 finding_summary_json；不承载核心可检索关系，值必须为 object。 字段约束：必填，默认 ''{}''::jsonb。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_decisions"."content_hash" IS 'Prompt Firewall 完整性摘要 content_hash；保存算法前缀和摘要，用于幂等、篡改检测或重算，禁止敏感原文。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_decisions"."evaluated_at" IS 'Prompt Firewall UTC 业务时间 evaluated_at；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_decisions"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_decisions"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON TABLE "ai_governance"."prompt_firewall_decisions" IS '每次输入/输出安全决策的不可变事实。';

-- ----------------------------
-- Table structure for prompt_firewall_policies
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."prompt_firewall_policies";
CREATE TABLE "ai_governance"."prompt_firewall_policies" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "policy_code" varchar(64) COLLATE "pg_catalog"."default" NOT NULL,
  "policy_name" varchar(160) COLLATE "pg_catalog"."default" NOT NULL,
  "scope_type" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "scope_id" uuid NOT NULL,
  "direction" varchar(16) COLLATE "pg_catalog"."default" NOT NULL,
  "default_action" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "fail_mode" varchar(16) COLLATE "pg_catalog"."default" NOT NULL,
  "status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "valid_from" timestamptz(6) NOT NULL,
  "valid_until" timestamptz(6),
  "revision" int8 NOT NULL DEFAULT 1,
  "row_version" int8 NOT NULL DEFAULT 1,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "created_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "updated_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "operation_source" varchar(32) COLLATE "pg_catalog"."default" NOT NULL DEFAULT 'ADMIN'::character varying,
  "operation_trace_id" varchar(128) COLLATE "pg_catalog"."default",
  "is_deleted" bool NOT NULL DEFAULT false,
  "deleted_at" timestamptz(6),
  "deleted_by" varchar(128) COLLATE "pg_catalog"."default",
  "delete_reason" varchar(512) COLLATE "pg_catalog"."default"
)
;
COMMENT ON COLUMN "ai_governance"."prompt_firewall_policies"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_policies"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_policies"."policy_code" IS 'Prompt Firewall 策略的稳定代码；用于测试、审批、发布和决策审计。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_policies"."policy_name" IS 'Prompt Firewall 策略显示名称；决策锁定 policy_id 和 revision。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_policies"."scope_type" IS 'Prompt Firewall 受控枚举；允许值：TENANT（租户作用域）、ORGANIZATION（组织作用域）、PROJECT（项目作用域）、USER（企业用户主体）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_policies"."scope_id" IS '逻辑引用 GovernanceScopeRef 指向的领域实体 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_policies"."direction" IS 'Prompt Firewall 受控枚举；允许值：INPUT（输入方向）、OUTPUT（输出方向）、BOTH（输入和输出方向）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_policies"."default_action" IS 'Prompt Firewall 受控枚举；允许值：ALLOW（允许）、BLOCK（阻断处理）、REDACT（脱敏后继续）、FORCE_INTERNAL（强制仅使用内网模型）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_policies"."fail_mode" IS 'Prompt Firewall 受控枚举；允许值：CLOSED（关闭终态，仅保留历史）、LKG_INTERNAL_ONLY（依赖失败时仅允许最近已知安全内网配置）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_policies"."status" IS 'Prompt Firewall 受控枚举；允许值：DRAFT（草稿未生效）、REVIEWING（待评审）、APPROVED（已批准，尚需发布或生效）、ACTIVE（有效并参与运行决策）、REVOKED（已撤销，不参与新决策）、RETIRED（退役终态，仅保留历史）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_policies"."valid_from" IS 'Prompt Firewall UTC 业务时间 valid_from；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_policies"."valid_until" IS 'Prompt Firewall UTC 业务时间 valid_until；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_policies"."revision" IS '配置业务 revision；每次有效内容变化严格递增，旧 revision 不得覆盖新版本。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_policies"."row_version" IS '乐观锁版本；每次成功 UPDATE 严格递增，用于防止并发覆盖。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_policies"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_policies"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_policies"."created_by" IS '创建记录的稳定主体标识；保存 ID 或服务主体，不保存仅供展示的姓名。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_policies"."updated_by" IS '最后修改记录的稳定主体标识；必须与 AuditEvent.actor 可关联。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_policies"."operation_source" IS '变更来源；ADMIN=治理后台，API=管理接口，SYNC=上游同步，SYSTEM=平台任务，MIGRATION=迁移。 字段约束：必填，默认 ''ADMIN''。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_policies"."operation_trace_id" IS '产生该次写入的 Trace 标识；用于跨 Module 联合排障和审计关联。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_policies"."is_deleted" IS '软删除标志；false=有效，true=逻辑删除且必须同时填写 deleted_at、deleted_by、delete_reason。 字段约束：必填，默认 false。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_policies"."deleted_at" IS '软删除生效时间，UTC；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_policies"."deleted_by" IS '执行软删除的稳定主体标识；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_policies"."delete_reason" IS '软删除业务原因；必须说明业务依据、影响范围并关联删除 AuditEvent，禁止空原因或无审计删除。字段约束：可空，仅 is_deleted=true 时必填。';
COMMENT ON TABLE "ai_governance"."prompt_firewall_policies" IS '独立 Prompt Firewall 输入输出安全策略。';

-- ----------------------------
-- Table structure for prompt_firewall_rules
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."prompt_firewall_rules";
CREATE TABLE "ai_governance"."prompt_firewall_rules" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "firewall_policy_id" uuid NOT NULL,
  "rule_code" varchar(64) COLLATE "pg_catalog"."default" NOT NULL,
  "rule_name" varchar(160) COLLATE "pg_catalog"."default" NOT NULL,
  "direction" varchar(16) COLLATE "pg_catalog"."default" NOT NULL,
  "rule_type" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "pattern_ref" varchar(1024) COLLATE "pg_catalog"."default" NOT NULL,
  "action" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "severity" varchar(16) COLLATE "pg_catalog"."default" NOT NULL,
  "priority" int4 NOT NULL,
  "status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "revision" int8 NOT NULL DEFAULT 1,
  "row_version" int8 NOT NULL DEFAULT 1,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "created_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "updated_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "operation_source" varchar(32) COLLATE "pg_catalog"."default" NOT NULL DEFAULT 'ADMIN'::character varying,
  "operation_trace_id" varchar(128) COLLATE "pg_catalog"."default",
  "is_deleted" bool NOT NULL DEFAULT false,
  "deleted_at" timestamptz(6),
  "deleted_by" varchar(128) COLLATE "pg_catalog"."default",
  "delete_reason" varchar(512) COLLATE "pg_catalog"."default"
)
;
COMMENT ON COLUMN "ai_governance"."prompt_firewall_rules"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_rules"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_rules"."firewall_policy_id" IS '逻辑引用 PromptFirewallPolicy 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_rules"."rule_code" IS 'Firewall Policy 内唯一规则代码；命中事件和审计通过该值定位规则版本。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_rules"."rule_name" IS '安全规则显示名称；命中和审计使用 rule_code、rule_id 和 revision。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_rules"."direction" IS 'Prompt Firewall 受控枚举；允许值：INPUT（输入方向）、OUTPUT（输出方向）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_rules"."rule_type" IS 'Prompt Firewall 受控枚举；允许值：REGEX（正则规则）、KEYWORD_SET（关键词集合）、PII_DETECTOR（个人信息检测器）、CLASSIFIER（内容分类器）、INJECTION_DETECTOR（提示注入检测器）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_rules"."pattern_ref" IS 'Prompt Firewall 外部安全引用 pattern_ref；仅保存 Vault/KMS/对象定位符，不保存 Secret、凭证或对象正文。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_rules"."action" IS 'Prompt Firewall 受控枚举；允许值：ALLOW（允许）、BLOCK（阻断处理）、REDACT（脱敏后继续）、FORCE_INTERNAL（强制仅使用内网模型）、ALERT（产生安全告警）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_rules"."severity" IS 'Prompt Firewall 受控枚举；允许值：LOW（低等级）、MEDIUM（中等级）、HIGH（高等级）、CRITICAL（关键等级，触发强控制）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_rules"."priority" IS 'Prompt Firewall 顺序、计数或重放控制字段 priority；用于确定性排序、幂等、投影或完整性验证，禁止无审计跳变。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_rules"."status" IS 'Prompt Firewall 受控枚举；允许值：DRAFT（草稿未生效）、ACTIVE（有效并参与运行决策）、SUSPENDED（暂停，不接受新增业务）、RETIRED（退役终态，仅保留历史）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_rules"."revision" IS '配置业务 revision；每次有效内容变化严格递增，旧 revision 不得覆盖新版本。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_rules"."row_version" IS '乐观锁版本；每次成功 UPDATE 严格递增，用于防止并发覆盖。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_rules"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_rules"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_rules"."created_by" IS '创建记录的稳定主体标识；保存 ID 或服务主体，不保存仅供展示的姓名。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_rules"."updated_by" IS '最后修改记录的稳定主体标识；必须与 AuditEvent.actor 可关联。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_rules"."operation_source" IS '变更来源；ADMIN=治理后台，API=管理接口，SYNC=上游同步，SYSTEM=平台任务，MIGRATION=迁移。 字段约束：必填，默认 ''ADMIN''。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_rules"."operation_trace_id" IS '产生该次写入的 Trace 标识；用于跨 Module 联合排障和审计关联。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_rules"."is_deleted" IS '软删除标志；false=有效，true=逻辑删除且必须同时填写 deleted_at、deleted_by、delete_reason。 字段约束：必填，默认 false。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_rules"."deleted_at" IS '软删除生效时间，UTC；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_rules"."deleted_by" IS '执行软删除的稳定主体标识；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."prompt_firewall_rules"."delete_reason" IS '软删除业务原因；必须说明业务依据、影响范围并关联删除 AuditEvent，禁止空原因或无审计删除。字段约束：可空，仅 is_deleted=true 时必填。';
COMMENT ON TABLE "ai_governance"."prompt_firewall_rules" IS 'Prompt Firewall 可测试、可排序的安全规则。';

-- ----------------------------
-- Table structure for provider_credential_refs
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."provider_credential_refs";
CREATE TABLE "ai_governance"."provider_credential_refs" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "provider_id" uuid NOT NULL,
  "credential_name" varchar(160) COLLATE "pg_catalog"."default" NOT NULL,
  "vault_ref" varchar(1024) COLLATE "pg_catalog"."default" NOT NULL,
  "credential_purpose" varchar(32) COLLATE "pg_catalog"."default" NOT NULL,
  "key_pool_code" varchar(64) COLLATE "pg_catalog"."default",
  "valid_from" timestamptz(6) NOT NULL,
  "valid_until" timestamptz(6),
  "rotated_at" timestamptz(6),
  "status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "revision" int8 NOT NULL DEFAULT 1,
  "row_version" int8 NOT NULL DEFAULT 1,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "created_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "updated_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "operation_source" varchar(32) COLLATE "pg_catalog"."default" NOT NULL DEFAULT 'ADMIN'::character varying,
  "operation_trace_id" varchar(128) COLLATE "pg_catalog"."default",
  "is_deleted" bool NOT NULL DEFAULT false,
  "deleted_at" timestamptz(6),
  "deleted_by" varchar(128) COLLATE "pg_catalog"."default",
  "delete_reason" varchar(512) COLLATE "pg_catalog"."default"
)
;
COMMENT ON COLUMN "ai_governance"."provider_credential_refs"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."provider_credential_refs"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."provider_credential_refs"."provider_id" IS '逻辑引用 ModelProvider 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."provider_credential_refs"."credential_name" IS '凭证引用的管理显示名；不得包含 Secret、账号密码或 Vault 内部敏感值。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."provider_credential_refs"."vault_ref" IS 'Model Catalog 外部安全引用 vault_ref；仅保存 Vault/KMS/对象定位符，不保存 Secret、凭证或对象正文。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."provider_credential_refs"."credential_purpose" IS 'Model Catalog 受控枚举；允许值：INFERENCE（模型推理凭证）、DISCOVERY（能力发现凭证）、BILLING（账单查询凭证）、HEALTH_CHECK（健康探测凭证）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."provider_credential_refs"."key_pool_code" IS '凭证所属 Key Pool 的可选稳定代码；用于轮换和容量分组，不包含任何密钥材料。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."provider_credential_refs"."valid_from" IS 'Model Catalog UTC 业务时间 valid_from；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."provider_credential_refs"."valid_until" IS 'Model Catalog UTC 业务时间 valid_until；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."provider_credential_refs"."rotated_at" IS 'Model Catalog UTC 业务时间 rotated_at；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."provider_credential_refs"."status" IS 'Model Catalog 受控枚举；允许值：PENDING（待处理或待生效）、ACTIVE（有效并参与运行决策）、ROTATING（轮换窗口，新旧凭证受控并存）、REVOKED（已撤销，不参与新决策）、EXPIRED（已过有效期）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."provider_credential_refs"."revision" IS '配置业务 revision；每次有效内容变化严格递增，旧 revision 不得覆盖新版本。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."provider_credential_refs"."row_version" IS '乐观锁版本；每次成功 UPDATE 严格递增，用于防止并发覆盖。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."provider_credential_refs"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."provider_credential_refs"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."provider_credential_refs"."created_by" IS '创建记录的稳定主体标识；保存 ID 或服务主体，不保存仅供展示的姓名。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."provider_credential_refs"."updated_by" IS '最后修改记录的稳定主体标识；必须与 AuditEvent.actor 可关联。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."provider_credential_refs"."operation_source" IS '变更来源；ADMIN=治理后台，API=管理接口，SYNC=上游同步，SYSTEM=平台任务，MIGRATION=迁移。 字段约束：必填，默认 ''ADMIN''。';
COMMENT ON COLUMN "ai_governance"."provider_credential_refs"."operation_trace_id" IS '产生该次写入的 Trace 标识；用于跨 Module 联合排障和审计关联。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."provider_credential_refs"."is_deleted" IS '软删除标志；false=有效，true=逻辑删除且必须同时填写 deleted_at、deleted_by、delete_reason。 字段约束：必填，默认 false。';
COMMENT ON COLUMN "ai_governance"."provider_credential_refs"."deleted_at" IS '软删除生效时间，UTC；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."provider_credential_refs"."deleted_by" IS '执行软删除的稳定主体标识；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."provider_credential_refs"."delete_reason" IS '软删除业务原因；必须说明业务依据、影响范围并关联删除 AuditEvent，禁止空原因或无审计删除。字段约束：可空，仅 is_deleted=true 时必填。';
COMMENT ON TABLE "ai_governance"."provider_credential_refs" IS 'Provider 凭证的 Vault/KMS 引用；不保存凭证原文。';

-- ----------------------------
-- Table structure for quota_reservations
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."quota_reservations";
CREATE TABLE "ai_governance"."quota_reservations" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "reservation_id" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "request_id" uuid NOT NULL,
  "user_allocation_id" uuid NOT NULL,
  "key_id" uuid NOT NULL,
  "key_revision" int8 NOT NULL,
  "billing_scope_type" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "billing_scope_id" uuid NOT NULL,
  "initial_reserved_credit" int8 NOT NULL,
  "total_reserved_credit" int8 NOT NULL,
  "estimated_token_ceiling" int8 NOT NULL,
  "credit_rate_revision" int8 NOT NULL,
  "status" varchar(32) COLLATE "pg_catalog"."default" NOT NULL,
  "expires_at" timestamptz(6) NOT NULL,
  "settled_at" timestamptz(6),
  "released_at" timestamptz(6),
  "row_version" int8 NOT NULL DEFAULT 1,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "created_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "updated_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "operation_trace_id" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "subject_id" uuid NOT NULL
)
;
COMMENT ON COLUMN "ai_governance"."quota_reservations"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."quota_reservations"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."quota_reservations"."reservation_id" IS '对外可追踪的唯一预占标识；关联 Request、Ledger 和对账，补预占不创建第二个 Reservation。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."quota_reservations"."request_id" IS '逻辑引用 UsageRequest 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."quota_reservations"."user_allocation_id" IS '关联 UserAllocation；多把 Key 共享同一人员额度，禁止创建 Key 钱包。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."quota_reservations"."key_id" IS '逻辑引用 UserApiKey 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."quota_reservations"."key_revision" IS '调用或授权时锁定的 User API Key revision；用于证明历史归属和动作上限。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."quota_reservations"."billing_scope_type" IS '资金归属 Scope 类型；仅 ORGANIZATION 或 PROJECT，请求不得覆盖。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."quota_reservations"."billing_scope_id" IS '资金归属组织或项目 UUID；必须与 Key、UserAllocation 和成员资格同 Tenant 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."quota_reservations"."initial_reserved_credit" IS 'Usage Accounting 整数 AI Credit 字段 initial_reserved_credit；P0 不表达人民币，变化必须来自 Ledger 或其可重建投影。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."quota_reservations"."total_reserved_credit" IS 'Usage Accounting 整数 AI Credit 字段 total_reserved_credit；P0 不表达人民币，变化必须来自 Ledger 或其可重建投影。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."quota_reservations"."estimated_token_ceiling" IS 'Usage Accounting Token 整数计量 estimated_token_ceiling；单位为 token，禁止浮点和隐式换算，须关联 Request/Usage 或周期投影。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."quota_reservations"."credit_rate_revision" IS 'Usage Accounting 整数 AI Credit 字段 credit_rate_revision；P0 不表达人民币，变化必须来自 Ledger 或其可重建投影。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."quota_reservations"."status" IS 'Usage Accounting 受控枚举；允许值：RESERVED（额度预占成功）、IN_FLIGHT（Provider 可能已接受请求）、SETTLED（已按最终用量结算）、RELEASED（预占已确认释放）、RECONCILING（用量未知，等待核对）、EXPIRED（已过有效期）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."quota_reservations"."expires_at" IS 'Usage Accounting UTC 业务时间 expires_at；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."quota_reservations"."settled_at" IS 'Usage Accounting UTC 业务时间 settled_at；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."quota_reservations"."released_at" IS 'Usage Accounting UTC 业务时间 released_at；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."quota_reservations"."row_version" IS '乐观锁版本；每次成功 UPDATE 严格递增，用于防止并发覆盖。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."quota_reservations"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."quota_reservations"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."quota_reservations"."created_by" IS '创建记录的稳定主体标识；保存 ID 或服务主体，不保存仅供展示的姓名。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."quota_reservations"."updated_by" IS '最后修改记录的稳定主体标识；必须与 AuditEvent.actor 可关联。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."quota_reservations"."operation_trace_id" IS '产生该次写入的 Trace 标识；用于跨 Module 联合排障和审计关联。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."quota_reservations"."subject_id" IS '预占所属 User Subject；与 UserAllocation、Key、BillingScope 必须同 Tenant 且一致。';
COMMENT ON TABLE "ai_governance"."quota_reservations" IS '每个请求唯一的 AI Credit 预占生命周期；补预占累加 total_reserved_credit。';

-- ----------------------------
-- Table structure for reconciliation_differences
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."reconciliation_differences";
CREATE TABLE "ai_governance"."reconciliation_differences" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "reconciliation_run_id" uuid NOT NULL,
  "reconciliation_item_id" uuid NOT NULL,
  "difference_code" varchar(64) COLLATE "pg_catalog"."default" NOT NULL,
  "difference_type" varchar(32) COLLATE "pg_catalog"."default" NOT NULL,
  "severity" varchar(16) COLLATE "pg_catalog"."default" NOT NULL,
  "credit_difference" int8,
  "token_difference" int8,
  "status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "evidence_json" jsonb NOT NULL,
  "detected_at" timestamptz(6) NOT NULL,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "subject_id" uuid
)
;
COMMENT ON COLUMN "ai_governance"."reconciliation_differences"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."reconciliation_differences"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."reconciliation_differences"."reconciliation_run_id" IS '逻辑引用 ReconciliationRun 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."reconciliation_differences"."reconciliation_item_id" IS '逻辑引用 ReconciliationItem 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."reconciliation_differences"."difference_code" IS '对账差异的 Tenant 内唯一业务编号；贯通调查、审批、调整或冲销和日结阻断，差异关闭后仍须保留且不得复用。字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."reconciliation_differences"."difference_type" IS 'Reconciliation 受控枚举；允许值：LEDGER_UNBALANCED（LedgerTransaction 分录代数和不为零）、RESERVATION_OPEN（Reservation 超期或未收敛）、USAGE_MISMATCH（Attempt、Provider 与最终 Usage Token 不一致）、RATE_MISMATCH（Token 按费率重算与结算 Credit 不一致）、KEY_ROLLUP_MISMATCH（Key 分录汇总与 UserAllocation 不一致）、USER_ROLLUP_MISMATCH（用户下所有 Key 汇总与 UserAllocation 不一致）、SCOPE_ROLLUP_MISMATCH（UserAllocation 汇总与 Scope 账户变化不一致）、COMPANY_ROLLUP_MISMATCH（全部 Scope 汇总与公司总盘不一致）、EMERGENCY_GRANT_VIOLATION（紧急支用超出批准额度、期限或 Key 范围）、REPLAY_MISMATCH（从 LedgerLeg 重放的投影与已存投影不一致）、AUDIT_MISSING（资金或关键操作缺少 AuditEvent）、REFERENCE_ORPHAN（逻辑引用目标缺失、跨 Tenant 或状态无效）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."reconciliation_differences"."severity" IS 'Reconciliation 受控枚举；允许值：LOW（低等级）、MEDIUM（中等级）、HIGH（高等级）、CRITICAL（关键等级，触发强控制）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."reconciliation_differences"."credit_difference" IS 'Reconciliation 整数 AI Credit 字段 credit_difference；P0 不表达人民币，变化必须来自 Ledger 或其可重建投影。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."reconciliation_differences"."token_difference" IS 'Reconciliation Token 整数计量 token_difference；单位为 token，禁止浮点和隐式换算，须关联 Request/Usage 或周期投影。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."reconciliation_differences"."status" IS 'Reconciliation 受控枚举；允许值：OPEN（已开启，等待处理）、INVESTIGATING（差异调查中）、RESOLVED（差异已按批准方式处置）、ACCEPTED_RISK（风险经审批接受）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."reconciliation_differences"."evidence_json" IS 'Reconciliation JSONB 证据或扩展快照 evidence_json；不承载核心可检索关系，值必须为 object。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."reconciliation_differences"."detected_at" IS 'Reconciliation UTC 业务时间 detected_at；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."reconciliation_differences"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."reconciliation_differences"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."reconciliation_differences"."subject_id" IS '差异涉及 User 时的直接 Subject 维度；禁止仅把 User 差异藏入 evidence JSON。';
COMMENT ON TABLE "ai_governance"."reconciliation_differences" IS '不可删除的对账差异事实。';

-- ----------------------------
-- Table structure for reconciliation_items
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."reconciliation_items";
CREATE TABLE "ai_governance"."reconciliation_items" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "reconciliation_run_id" uuid NOT NULL,
  "check_code" varchar(64) COLLATE "pg_catalog"."default" NOT NULL,
  "target_type" varchar(64) COLLATE "pg_catalog"."default" NOT NULL,
  "target_id" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "expected_credit" int8,
  "actual_credit" int8,
  "expected_tokens" int8,
  "actual_tokens" int8,
  "expected_hash" varchar(71) COLLATE "pg_catalog"."default",
  "actual_hash" varchar(71) COLLATE "pg_catalog"."default",
  "result" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "evidence_json" jsonb NOT NULL DEFAULT '{}'::jsonb,
  "checked_at" timestamptz(6) NOT NULL,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "subject_id" uuid
)
;
COMMENT ON COLUMN "ai_governance"."reconciliation_items"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."reconciliation_items"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."reconciliation_items"."reconciliation_run_id" IS '逻辑引用 ReconciliationRun 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."reconciliation_items"."check_code" IS '对账检查规则代码；固定标识账本平衡、Reservation、Usage、费率、汇总或审计覆盖检查。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."reconciliation_items"."target_type" IS '本次对账检查目标的稳定类型，例如 TRANSACTION、REQUEST、KEY 或 ACCOUNT；决定 target_id 解析规则。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."reconciliation_items"."target_id" IS '被检查目标的不透明稳定标识；与 check_code 和 target_type 共同唯一定位差异证据。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."reconciliation_items"."expected_credit" IS 'Reconciliation 整数 AI Credit 字段 expected_credit；P0 不表达人民币，变化必须来自 Ledger 或其可重建投影。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."reconciliation_items"."actual_credit" IS 'Reconciliation 整数 AI Credit 字段 actual_credit；P0 不表达人民币，变化必须来自 Ledger 或其可重建投影。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."reconciliation_items"."expected_tokens" IS 'Reconciliation Token 整数计量 expected_tokens；单位为 token，禁止浮点和隐式换算，须关联 Request/Usage 或周期投影。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."reconciliation_items"."actual_tokens" IS 'Reconciliation Token 整数计量 actual_tokens；单位为 token，禁止浮点和隐式换算，须关联 Request/Usage 或周期投影。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."reconciliation_items"."expected_hash" IS 'Reconciliation 完整性摘要 expected_hash；保存算法前缀和摘要，用于幂等、篡改检测或重算，禁止敏感原文。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."reconciliation_items"."actual_hash" IS 'Reconciliation 完整性摘要 actual_hash；保存算法前缀和摘要，用于幂等、篡改检测或重算，禁止敏感原文。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."reconciliation_items"."result" IS 'Reconciliation 受控枚举；允许值：BALANCED（全部检查平衡）、DIFFERENCE（核验发现差异）、SKIPPED（按规则跳过检查并记录原因）、ERROR（检查执行错误，结果不可用于平衡结论）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."reconciliation_items"."evidence_json" IS 'Reconciliation JSONB 证据或扩展快照 evidence_json；不承载核心可检索关系，值必须为 object。 字段约束：必填，默认 ''{}''::jsonb。';
COMMENT ON COLUMN "ai_governance"."reconciliation_items"."checked_at" IS 'Reconciliation UTC 业务时间 checked_at；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."reconciliation_items"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."reconciliation_items"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."reconciliation_items"."subject_id" IS '对账目标涉及 User 时的直接 Subject 维度；Tenant/Scope 全局检查允许为空。';
COMMENT ON TABLE "ai_governance"."reconciliation_items" IS '十二类强制对账检查的目标级结果。';

-- ----------------------------
-- Table structure for reconciliation_resolutions
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."reconciliation_resolutions";
CREATE TABLE "ai_governance"."reconciliation_resolutions" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "reconciliation_difference_id" uuid NOT NULL,
  "resolution_action" varchar(32) COLLATE "pg_catalog"."default" NOT NULL,
  "resolution_reason" varchar(2048) COLLATE "pg_catalog"."default" NOT NULL,
  "correcting_transaction_id" uuid,
  "approval_request_id" uuid NOT NULL,
  "resolved_by_subject_id" uuid NOT NULL,
  "resolved_at" timestamptz(6) NOT NULL,
  "evidence_json" jsonb NOT NULL DEFAULT '{}'::jsonb,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "trace_id" varchar(128) COLLATE "pg_catalog"."default" NOT NULL
)
;
COMMENT ON COLUMN "ai_governance"."reconciliation_resolutions"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."reconciliation_resolutions"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."reconciliation_resolutions"."reconciliation_difference_id" IS '逻辑引用 ReconciliationDifference 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."reconciliation_resolutions"."resolution_action" IS 'Reconciliation 受控枚举；允许值：ADJUST（经审批的账本调整交易）、REVERSAL（对原交易执行等额反向冲销）、REPROCESS（重新执行投影、消息或对账检查）、ACCEPT_RISK（经审批接受差异风险，不修改原事实）、NO_ACTION_VALIDATED（核验后确认无需资金更正）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."reconciliation_resolutions"."resolution_reason" IS '对账差异处置理由；必须说明证据、责任、修正方式及为何能够关闭或接受风险。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."reconciliation_resolutions"."correcting_transaction_id" IS '逻辑引用 差异处置 LedgerTransaction 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."reconciliation_resolutions"."approval_request_id" IS '关联 ApprovalRequest；敏感动作须校验同 Tenant、APPROVED、未过期且 payload_hash 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."reconciliation_resolutions"."resolved_by_subject_id" IS '逻辑引用 差异处置 IdentitySubject 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."reconciliation_resolutions"."resolved_at" IS 'Reconciliation UTC 业务时间 resolved_at；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."reconciliation_resolutions"."evidence_json" IS 'Reconciliation JSONB 证据或扩展快照 evidence_json；不承载核心可检索关系，值必须为 object。 字段约束：必填，默认 ''{}''::jsonb。';
COMMENT ON COLUMN "ai_governance"."reconciliation_resolutions"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."reconciliation_resolutions"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."reconciliation_resolutions"."trace_id" IS '端到端 Trace 标识；贯通 Request、Attempt、Usage、Ledger、安全和 Audit。 字段约束：必填。';
COMMENT ON TABLE "ai_governance"."reconciliation_resolutions" IS '经审批的差异处置；更正只能引用新 ADJUST/REVERSAL 交易。';

-- ----------------------------
-- Table structure for reconciliation_runs
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."reconciliation_runs";
CREATE TABLE "ai_governance"."reconciliation_runs" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "run_code" varchar(96) COLLATE "pg_catalog"."default" NOT NULL,
  "reconciliation_type" varchar(32) COLLATE "pg_catalog"."default" NOT NULL,
  "window_start" timestamptz(6) NOT NULL,
  "window_end" timestamptz(6) NOT NULL,
  "fact_watermark" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "status" varchar(32) COLLATE "pg_catalog"."default" NOT NULL,
  "total_item_count" int8 NOT NULL DEFAULT 0,
  "balanced_item_count" int8 NOT NULL DEFAULT 0,
  "difference_item_count" int8 NOT NULL DEFAULT 0,
  "unresolved_difference_count" int8 NOT NULL DEFAULT 0,
  "started_at" timestamptz(6),
  "finished_at" timestamptz(6),
  "closed_at" timestamptz(6),
  "row_version" int8 NOT NULL DEFAULT 1,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "created_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "updated_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "operation_trace_id" varchar(128) COLLATE "pg_catalog"."default"
)
;
COMMENT ON COLUMN "ai_governance"."reconciliation_runs"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."reconciliation_runs"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."reconciliation_runs"."run_code" IS '一次固定窗口对账运行的唯一业务编号；用于运营检索、日结和证据归档。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."reconciliation_runs"."reconciliation_type" IS 'Reconciliation 受控枚举；允许值：INTERNAL_DAILY（平台每日内部 Token/Credit 对账）、INTERNAL_MANUAL（人工触发的内部对账）、INTEGRITY_REPLAY（从事实重放的完整性对账）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."reconciliation_runs"."window_start" IS 'Reconciliation UTC 业务时间 window_start；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."reconciliation_runs"."window_end" IS 'Reconciliation UTC 业务时间 window_end；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."reconciliation_runs"."fact_watermark" IS 'Reconciliation 顺序、计数或重放控制字段 fact_watermark；用于确定性排序、幂等、投影或完整性验证，禁止无审计跳变。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."reconciliation_runs"."status" IS 'Reconciliation 受控枚举；允许值：OPEN（已开启，等待处理）、RUNNING（正在执行）、DIFFERENCE_FOUND（发现差异）、BALANCED（全部检查平衡）、INVESTIGATING（差异调查中）、RESOLVED（差异已按批准方式处置）、CLOSED（关闭终态，仅保留历史）、FAILED（执行失败）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."reconciliation_runs"."total_item_count" IS 'Reconciliation 顺序、计数或重放控制字段 total_item_count；用于确定性排序、幂等、投影或完整性验证，禁止无审计跳变。 字段约束：必填，默认 0。';
COMMENT ON COLUMN "ai_governance"."reconciliation_runs"."balanced_item_count" IS 'Reconciliation 整数 AI Credit 字段 balanced_item_count；P0 不表达人民币，变化必须来自 Ledger 或其可重建投影。 字段约束：必填，默认 0。';
COMMENT ON COLUMN "ai_governance"."reconciliation_runs"."difference_item_count" IS 'Reconciliation 顺序、计数或重放控制字段 difference_item_count；用于确定性排序、幂等、投影或完整性验证，禁止无审计跳变。 字段约束：必填，默认 0。';
COMMENT ON COLUMN "ai_governance"."reconciliation_runs"."unresolved_difference_count" IS 'Reconciliation 顺序、计数或重放控制字段 unresolved_difference_count；用于确定性排序、幂等、投影或完整性验证，禁止无审计跳变。 字段约束：必填，默认 0。';
COMMENT ON COLUMN "ai_governance"."reconciliation_runs"."started_at" IS 'Reconciliation UTC 业务时间 started_at；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."reconciliation_runs"."finished_at" IS 'Reconciliation UTC 业务时间 finished_at；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."reconciliation_runs"."closed_at" IS 'Reconciliation UTC 业务时间 closed_at；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."reconciliation_runs"."row_version" IS '乐观锁版本；每次成功 UPDATE 严格递增，用于防止并发覆盖。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."reconciliation_runs"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."reconciliation_runs"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."reconciliation_runs"."created_by" IS '创建记录的稳定主体标识；保存 ID 或服务主体，不保存仅供展示的姓名。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."reconciliation_runs"."updated_by" IS '最后修改记录的稳定主体标识；必须与 AuditEvent.actor 可关联。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."reconciliation_runs"."operation_trace_id" IS '产生该次写入的 Trace 标识；用于跨 Module 联合排障和审计关联。 字段约束：可空。';
COMMENT ON TABLE "ai_governance"."reconciliation_runs" IS '固定窗口和事实 watermark 的内部 Credit/Token 对账运行。';

-- ----------------------------
-- Table structure for route_policies
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."route_policies";
CREATE TABLE "ai_governance"."route_policies" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "policy_code" varchar(64) COLLATE "pg_catalog"."default" NOT NULL,
  "policy_name" varchar(160) COLLATE "pg_catalog"."default" NOT NULL,
  "scope_type" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "scope_id" uuid NOT NULL,
  "capability" varchar(32) COLLATE "pg_catalog"."default" NOT NULL,
  "max_attempts" int2 NOT NULL DEFAULT 1,
  "retry_before_first_content_only" bool NOT NULL DEFAULT true,
  "allow_fallback" bool NOT NULL DEFAULT true,
  "status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "revision" int8 NOT NULL DEFAULT 1,
  "row_version" int8 NOT NULL DEFAULT 1,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "created_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "updated_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "operation_source" varchar(32) COLLATE "pg_catalog"."default" NOT NULL DEFAULT 'ADMIN'::character varying,
  "operation_trace_id" varchar(128) COLLATE "pg_catalog"."default",
  "is_deleted" bool NOT NULL DEFAULT false,
  "deleted_at" timestamptz(6),
  "deleted_by" varchar(128) COLLATE "pg_catalog"."default",
  "delete_reason" varchar(512) COLLATE "pg_catalog"."default"
)
;
COMMENT ON COLUMN "ai_governance"."route_policies"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."route_policies"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."route_policies"."policy_code" IS '路由策略的稳定代码；绑定 Scope、Capability 和有序候选，Tenant 内唯一。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."route_policies"."policy_name" IS 'RoutePolicy 显示名称；执行使用 policy_code、revision 和候选序列。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."route_policies"."scope_type" IS 'Routing Decision 受控枚举；允许值：TENANT（租户作用域）、ORGANIZATION（组织作用域）、PROJECT（项目作用域）、USER（企业用户主体）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."route_policies"."scope_id" IS '逻辑引用 GovernanceScopeRef 指向的领域实体 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."route_policies"."capability" IS '该 RoutePolicy 适用的标准模型能力代码，例如 CHAT 或 EMBEDDING；候选必须声明相同能力。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."route_policies"."max_attempts" IS '单 Request 最多真实 Provider Attempt 数，范围 1 至 8；包含首选和全部 Fallback。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."route_policies"."retry_before_first_content_only" IS '是否仅允许在首个安全内容发送前重试；P0 固定为 true。 字段约束：必填，默认 true。';
COMMENT ON COLUMN "ai_governance"."route_policies"."allow_fallback" IS '是否允许在 RoutePlan 授权候选内有限 Fallback。 字段约束：必填，默认 true。';
COMMENT ON COLUMN "ai_governance"."route_policies"."status" IS 'Routing Decision 受控枚举；允许值：DRAFT（草稿未生效）、REVIEWING（待评审）、APPROVED（已批准，尚需发布或生效）、ACTIVE（有效并参与运行决策）、REVOKED（已撤销，不参与新决策）、RETIRED（退役终态，仅保留历史）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."route_policies"."revision" IS '配置业务 revision；每次有效内容变化严格递增，旧 revision 不得覆盖新版本。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."route_policies"."row_version" IS '乐观锁版本；每次成功 UPDATE 严格递增，用于防止并发覆盖。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."route_policies"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."route_policies"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."route_policies"."created_by" IS '创建记录的稳定主体标识；保存 ID 或服务主体，不保存仅供展示的姓名。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."route_policies"."updated_by" IS '最后修改记录的稳定主体标识；必须与 AuditEvent.actor 可关联。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."route_policies"."operation_source" IS '变更来源；ADMIN=治理后台，API=管理接口，SYNC=上游同步，SYSTEM=平台任务，MIGRATION=迁移。 字段约束：必填，默认 ''ADMIN''。';
COMMENT ON COLUMN "ai_governance"."route_policies"."operation_trace_id" IS '产生该次写入的 Trace 标识；用于跨 Module 联合排障和审计关联。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."route_policies"."is_deleted" IS '软删除标志；false=有效，true=逻辑删除且必须同时填写 deleted_at、deleted_by、delete_reason。 字段约束：必填，默认 false。';
COMMENT ON COLUMN "ai_governance"."route_policies"."deleted_at" IS '软删除生效时间，UTC；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."route_policies"."deleted_by" IS '执行软删除的稳定主体标识；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."route_policies"."delete_reason" IS '软删除业务原因；必须说明业务依据、影响范围并关联删除 AuditEvent，禁止空原因或无审计删除。字段约束：可空，仅 is_deleted=true 时必填。';
COMMENT ON TABLE "ai_governance"."route_policies" IS '有边界的顺序路由、重试和 Fallback 策略。';

-- ----------------------------
-- Table structure for route_policy_candidates
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."route_policy_candidates";
CREATE TABLE "ai_governance"."route_policy_candidates" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "route_policy_id" uuid NOT NULL,
  "sequence_no" int2 NOT NULL,
  "model_id" uuid NOT NULL,
  "channel_id" uuid NOT NULL,
  "network_class" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "trigger_error_codes" varchar(64)[] COLLATE "pg_catalog"."default" NOT NULL DEFAULT ARRAY['UPSTREAM_UNAVAILABLE'::character varying],
  "status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "revision" int8 NOT NULL DEFAULT 1,
  "row_version" int8 NOT NULL DEFAULT 1,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "created_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "updated_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "operation_source" varchar(32) COLLATE "pg_catalog"."default" NOT NULL DEFAULT 'ADMIN'::character varying,
  "operation_trace_id" varchar(128) COLLATE "pg_catalog"."default",
  "is_deleted" bool NOT NULL DEFAULT false,
  "deleted_at" timestamptz(6),
  "deleted_by" varchar(128) COLLATE "pg_catalog"."default",
  "delete_reason" varchar(512) COLLATE "pg_catalog"."default"
)
;
COMMENT ON COLUMN "ai_governance"."route_policy_candidates"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."route_policy_candidates"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."route_policy_candidates"."route_policy_id" IS '逻辑引用 RoutePolicy 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."route_policy_candidates"."sequence_no" IS 'Routing Decision 顺序、计数或重放控制字段 sequence_no；用于确定性排序、幂等、投影或完整性验证，禁止无审计跳变。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."route_policy_candidates"."model_id" IS '逻辑引用 ModelCatalogEntry 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."route_policy_candidates"."channel_id" IS '逻辑引用 ModelChannel 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."route_policy_candidates"."network_class" IS 'Routing Decision 受控枚举；允许值：INTERNAL（内部网络或内部数据）、EXTERNAL（外部网络或供应商）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."route_policy_candidates"."trigger_error_codes" IS '允许切换到该后续候选的稳定错误码集合；仅首个安全内容前生效。 字段约束：必填，默认 ARRAY[''UPSTREAM_UNAVAILABLE'']::varchar[]。';
COMMENT ON COLUMN "ai_governance"."route_policy_candidates"."status" IS 'Routing Decision 受控枚举；允许值：DRAFT（草稿未生效）、ACTIVE（有效并参与运行决策）、SUSPENDED（暂停，不接受新增业务）、RETIRED（退役终态，仅保留历史）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."route_policy_candidates"."revision" IS '配置业务 revision；每次有效内容变化严格递增，旧 revision 不得覆盖新版本。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."route_policy_candidates"."row_version" IS '乐观锁版本；每次成功 UPDATE 严格递增，用于防止并发覆盖。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."route_policy_candidates"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."route_policy_candidates"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."route_policy_candidates"."created_by" IS '创建记录的稳定主体标识；保存 ID 或服务主体，不保存仅供展示的姓名。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."route_policy_candidates"."updated_by" IS '最后修改记录的稳定主体标识；必须与 AuditEvent.actor 可关联。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."route_policy_candidates"."operation_source" IS '变更来源；ADMIN=治理后台，API=管理接口，SYNC=上游同步，SYSTEM=平台任务，MIGRATION=迁移。 字段约束：必填，默认 ''ADMIN''。';
COMMENT ON COLUMN "ai_governance"."route_policy_candidates"."operation_trace_id" IS '产生该次写入的 Trace 标识；用于跨 Module 联合排障和审计关联。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."route_policy_candidates"."is_deleted" IS '软删除标志；false=有效，true=逻辑删除且必须同时填写 deleted_at、deleted_by、delete_reason。 字段约束：必填，默认 false。';
COMMENT ON COLUMN "ai_governance"."route_policy_candidates"."deleted_at" IS '软删除生效时间，UTC；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."route_policy_candidates"."deleted_by" IS '执行软删除的稳定主体标识；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."route_policy_candidates"."delete_reason" IS '软删除业务原因；必须说明业务依据、影响范围并关联删除 AuditEvent，禁止空原因或无审计删除。字段约束：可空，仅 is_deleted=true 时必填。';
COMMENT ON TABLE "ai_governance"."route_policy_candidates" IS 'RoutePolicy 的有序授权候选。';

-- ----------------------------
-- Table structure for runtime_snapshots
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."runtime_snapshots";
CREATE TABLE "ai_governance"."runtime_snapshots" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "snapshot_id" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "config_domain" varchar(32) COLLATE "pg_catalog"."default" NOT NULL,
  "config_revision" int8 NOT NULL,
  "generation" int8 NOT NULL,
  "snapshot_hash" varchar(71) COLLATE "pg_catalog"."default" NOT NULL,
  "object_ref" varchar(2048) COLLATE "pg_catalog"."default" NOT NULL,
  "status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "effective_from" timestamptz(6) NOT NULL,
  "expires_at" timestamptz(6),
  "published_at" timestamptz(6),
  "activated_at" timestamptz(6),
  "row_version" int8 NOT NULL DEFAULT 1,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "created_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "updated_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "operation_trace_id" varchar(128) COLLATE "pg_catalog"."default"
)
;
COMMENT ON COLUMN "ai_governance"."runtime_snapshots"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."runtime_snapshots"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."runtime_snapshots"."snapshot_id" IS '已构建配置快照的稳定业务标识；用于数据面获取、校验 Hash、回滚定位和审计。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."runtime_snapshots"."config_domain" IS 'Configuration Publisher 受控枚举；允许值：IDENTITY（身份与撤销配置域）、ORGANIZATION（组织作用域）、PROJECT（项目作用域）、AUTHORIZATION（授权配置域）、NAVIGATION（菜单和路由配置域）、KEY_LIMIT（Key 限额配置域）、MODEL_ACCESS（模型授权配置域）、ROUTING（路由配置域）、PROMPT_FIREWALL（Prompt Firewall 配置域）、CREDIT_RATE（AI Credit 费率配置域）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."runtime_snapshots"."config_revision" IS 'Configuration Publisher 不可回退版本 config_revision；发布、决策和历史事实必须锁定当时值。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."runtime_snapshots"."generation" IS '投影或 RuntimeSnapshot 的重建代次；新 generation 完成全量校验后原子切换，旧 generation 不得覆盖或与新代混读。字段约束：必填且大于 0。';
COMMENT ON COLUMN "ai_governance"."runtime_snapshots"."snapshot_hash" IS 'Configuration Publisher 完整性摘要 snapshot_hash；保存算法前缀和摘要，用于幂等、篡改检测或重算，禁止敏感原文。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."runtime_snapshots"."object_ref" IS 'Configuration Publisher 外部安全引用 object_ref；仅保存 Vault/KMS/对象定位符，不保存 Secret、凭证或对象正文。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."runtime_snapshots"."status" IS 'Configuration Publisher 受控枚举；允许值：BUILDING（构建中）、PUBLISHED（已发布但未必激活）、ACTIVE（有效并参与运行决策）、SUPERSEDED（已被更高 revision 替代）、REVOKED（已撤销，不参与新决策）、FAILED（执行失败）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."runtime_snapshots"."effective_from" IS 'Configuration Publisher UTC 业务时间 effective_from；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."runtime_snapshots"."expires_at" IS 'Configuration Publisher UTC 业务时间 expires_at；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."runtime_snapshots"."published_at" IS 'Configuration Publisher UTC 业务时间 published_at；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."runtime_snapshots"."activated_at" IS 'Configuration Publisher UTC 业务时间 activated_at；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."runtime_snapshots"."row_version" IS '乐观锁版本；每次成功 UPDATE 严格递增，用于防止并发覆盖。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."runtime_snapshots"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."runtime_snapshots"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."runtime_snapshots"."created_by" IS '创建记录的稳定主体标识；保存 ID 或服务主体，不保存仅供展示的姓名。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."runtime_snapshots"."updated_by" IS '最后修改记录的稳定主体标识；必须与 AuditEvent.actor 可关联。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."runtime_snapshots"."operation_trace_id" IS '产生该次写入的 Trace 标识；用于跨 Module 联合排障和审计关联。 字段约束：可空。';
COMMENT ON TABLE "ai_governance"."runtime_snapshots" IS '控制面发布给数据面的完整、不可变、版本化配置快照。';

-- ----------------------------
-- Table structure for safety_events
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."safety_events";
CREATE TABLE "ai_governance"."safety_events" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "request_id" uuid NOT NULL,
  "firewall_decision_id" uuid NOT NULL,
  "rule_id" uuid,
  "direction" varchar(16) COLLATE "pg_catalog"."default" NOT NULL,
  "severity" varchar(16) COLLATE "pg_catalog"."default" NOT NULL,
  "action" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "reason_code" varchar(64) COLLATE "pg_catalog"."default" NOT NULL,
  "content_hash" varchar(71) COLLATE "pg_catalog"."default" NOT NULL,
  "redacted_excerpt" varchar(512) COLLATE "pg_catalog"."default",
  "occurred_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "trace_id" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp()
)
;
COMMENT ON COLUMN "ai_governance"."safety_events"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."safety_events"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."safety_events"."request_id" IS '逻辑引用 UsageRequest 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."safety_events"."firewall_decision_id" IS '逻辑引用 PromptFirewallDecision 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."safety_events"."rule_id" IS '逻辑引用 PromptFirewallRule 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."safety_events"."direction" IS 'Prompt Firewall 受控枚举；允许值：INPUT（输入方向）、OUTPUT（输出方向）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."safety_events"."severity" IS 'Prompt Firewall 受控枚举；允许值：LOW（低等级）、MEDIUM（中等级）、HIGH（高等级）、CRITICAL（关键等级，触发强控制）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."safety_events"."action" IS 'Prompt Firewall 受控枚举；允许值：BLOCK（阻断处理）、REDACT（脱敏后继续）、FORCE_INTERNAL（强制仅使用内网模型）、ALERT（产生安全告警）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."safety_events"."reason_code" IS '安全事件的稳定命中原因码；不包含敏感正文，用于告警、统计和申诉复核。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."safety_events"."content_hash" IS 'Prompt Firewall 完整性摘要 content_hash；保存算法前缀和摘要，用于幂等、篡改检测或重算，禁止敏感原文。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."safety_events"."redacted_excerpt" IS '安全事件的可选脱敏摘录，最长 512 字符；只能保留排障所需最小内容，禁止保存命中原文。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."safety_events"."occurred_at" IS 'Prompt Firewall UTC 业务时间 occurred_at；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."safety_events"."trace_id" IS '端到端 Trace 标识；贯通 Request、Attempt、Usage、Ledger、安全和 Audit。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."safety_events"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."safety_events"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON TABLE "ai_governance"."safety_events" IS '安全命中、阻断、脱敏与强制内部模型的事件事实。';

-- ----------------------------
-- Table structure for schema_migration_contracts
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."schema_migration_contracts";
CREATE TABLE "ai_governance"."schema_migration_contracts" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "migration_version" varchar(16) COLLATE "pg_catalog"."default" NOT NULL,
  "migration_name" varchar(160) COLLATE "pg_catalog"."default" NOT NULL,
  "checksum_sha256" varchar(71) COLLATE "pg_catalog"."default" NOT NULL,
  "applied_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "applied_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "execution_ms" int8 NOT NULL DEFAULT 0,
  "success" bool NOT NULL,
  "tool_version" varchar(64) COLLATE "pg_catalog"."default",
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp()
)
;
COMMENT ON COLUMN "ai_governance"."schema_migration_contracts"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."schema_migration_contracts"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."schema_migration_contracts"."migration_version" IS '全局迁移版本号，格式 V 加三位数字并严格单调递增；同一交付基线不得出现重复版本或子 Module 私有序列。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."schema_migration_contracts"."migration_name" IS '迁移文件的稳定名称，不含路径；与 migration_version 和 SHA-256 共同证明实际执行内容。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."schema_migration_contracts"."checksum_sha256" IS '数据库迁移治理 完整性摘要 checksum_sha256；保存算法前缀和摘要，用于幂等、篡改检测或重算，禁止敏感原文。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."schema_migration_contracts"."applied_by" IS '执行迁移的数据库账号、流水线服务主体或获批操作人标识；必须能够关联发布记录和变更单。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."schema_migration_contracts"."applied_at" IS '数据库迁移治理 UTC 业务时间 applied_at；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."schema_migration_contracts"."execution_ms" IS '迁移执行耗时，单位毫秒且不得为负；用于发布性能审计，不作为成功判定唯一依据。 字段约束：必填，默认 0。';
COMMENT ON COLUMN "ai_governance"."schema_migration_contracts"."success" IS '迁移是否成功；true=成功提交，false=失败并阻止版本推进。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."schema_migration_contracts"."tool_version" IS '执行迁移的 psql 或迁移工具版本；用于复现执行环境，历史环境无法确定时允许为空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."schema_migration_contracts"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."schema_migration_contracts"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON TABLE "ai_governance"."schema_migration_contracts" IS '数据库迁移执行证据；平台部署工具写入，不替代迁移工具自身历史表。';

-- ----------------------------
-- Table structure for sys_access_policies
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."sys_access_policies";
CREATE TABLE "ai_governance"."sys_access_policies" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "policy_code" varchar(64) COLLATE "pg_catalog"."default" NOT NULL,
  "policy_name" varchar(160) COLLATE "pg_catalog"."default" NOT NULL,
  "effect" varchar(8) COLLATE "pg_catalog"."default" NOT NULL,
  "required_subject_status" varchar(24) COLLATE "pg_catalog"."default",
  "required_membership_type" varchar(24) COLLATE "pg_catalog"."default",
  "require_resource_owner" bool NOT NULL DEFAULT false,
  "minimum_auth_strength" int2,
  "allowed_subject_types" varchar(24)[] COLLATE "pg_catalog"."default" NOT NULL DEFAULT ARRAY['USER'::character varying],
  "maximum_data_classification" varchar(24) COLLATE "pg_catalog"."default",
  "require_maker_checker" bool NOT NULL DEFAULT false,
  "field_mask_code" varchar(64) COLLATE "pg_catalog"."default",
  "decision_reason_code" varchar(64) COLLATE "pg_catalog"."default" NOT NULL,
  "status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "valid_from" timestamptz(6) NOT NULL,
  "valid_until" timestamptz(6),
  "revision" int8 NOT NULL DEFAULT 1,
  "row_version" int8 NOT NULL DEFAULT 1,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "created_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "updated_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "operation_source" varchar(32) COLLATE "pg_catalog"."default" NOT NULL DEFAULT 'ADMIN'::character varying,
  "operation_trace_id" varchar(128) COLLATE "pg_catalog"."default",
  "is_deleted" bool NOT NULL DEFAULT false,
  "deleted_at" timestamptz(6),
  "deleted_by" varchar(128) COLLATE "pg_catalog"."default",
  "delete_reason" varchar(512) COLLATE "pg_catalog"."default"
)
;
COMMENT ON COLUMN "ai_governance"."sys_access_policies"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."sys_access_policies"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_access_policies"."policy_code" IS '固定 ABAC 策略的稳定代码；用于审批、发布、回滚和审计检索。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_access_policies"."policy_name" IS '访问策略显示名称；机器判定使用固定字段、绑定和 revision。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_access_policies"."effect" IS 'Authorization Module 受控枚举；允许值：ALLOW（允许）、DENY（拒绝，优先于允许）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_access_policies"."required_subject_status" IS 'Authorization Module 受控枚举；允许值：ACTIVE（有效并参与运行决策）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."sys_access_policies"."required_membership_type" IS 'Authorization Module 受控枚举；允许值：NONE（不要求成员关系）、ORGANIZATION（组织作用域）、PROJECT（项目作用域）、ANY（任一条件满足或不限定类别）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."sys_access_policies"."require_resource_owner" IS '固定 ABAC Owner 条件；true=Subject 必须与领域资源 Owner 匹配，false=不检查 Owner；请求字段不得伪造 Owner。字段约束：必填，默认 false。';
COMMENT ON COLUMN "ai_governance"."sys_access_policies"."minimum_auth_strength" IS '策略要求的最低认证强度等级，范围 0 至 9；为空表示不附加认证强度条件。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."sys_access_policies"."allowed_subject_types" IS '策略允许的主体类型集合；仅 USER 或 SERVICE，默认只允许 USER。 字段约束：必填，默认 ARRAY[''USER'']::varchar[]。';
COMMENT ON COLUMN "ai_governance"."sys_access_policies"."maximum_data_classification" IS 'Authorization Module 受控枚举；允许值：PUBLIC（公开数据）、INTERNAL（内部网络或内部数据）、CONFIDENTIAL（机密数据，仅允许更严格处理）、RESTRICTED（受限数据，仅批准私有模型）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."sys_access_policies"."require_maker_checker" IS 'maker-checker 义务；true=执行前必须存在申请人与审批人不同的有效审批，false=不附加双人复核。字段约束：必填，默认 false。';
COMMENT ON COLUMN "ai_governance"."sys_access_policies"."field_mask_code" IS '授权允许但需字段裁剪时返回的受控遮罩方案代码；为空表示不附加字段遮罩义务。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."sys_access_policies"."decision_reason_code" IS '策略命中后写入 AuthorizationDecision 的稳定原因码；供调用方、安全和审计解释。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_access_policies"."status" IS 'Authorization Module 受控枚举；允许值：DRAFT（草稿未生效）、REVIEWING（待评审）、APPROVED（已批准，尚需发布或生效）、ACTIVE（有效并参与运行决策）、REVOKED（已撤销，不参与新决策）、RETIRED（退役终态，仅保留历史）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_access_policies"."valid_from" IS 'Authorization Module UTC 业务时间 valid_from；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_access_policies"."valid_until" IS 'Authorization Module UTC 业务时间 valid_until；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."sys_access_policies"."revision" IS '配置业务 revision；每次有效内容变化严格递增，旧 revision 不得覆盖新版本。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."sys_access_policies"."row_version" IS '乐观锁版本；每次成功 UPDATE 严格递增，用于防止并发覆盖。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."sys_access_policies"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."sys_access_policies"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."sys_access_policies"."created_by" IS '创建记录的稳定主体标识；保存 ID 或服务主体，不保存仅供展示的姓名。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_access_policies"."updated_by" IS '最后修改记录的稳定主体标识；必须与 AuditEvent.actor 可关联。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_access_policies"."operation_source" IS '变更来源；ADMIN=治理后台，API=管理接口，SYNC=上游同步，SYSTEM=平台任务，MIGRATION=迁移。 字段约束：必填，默认 ''ADMIN''。';
COMMENT ON COLUMN "ai_governance"."sys_access_policies"."operation_trace_id" IS '产生该次写入的 Trace 标识；用于跨 Module 联合排障和审计关联。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."sys_access_policies"."is_deleted" IS '软删除标志；false=有效，true=逻辑删除且必须同时填写 deleted_at、deleted_by、delete_reason。 字段约束：必填，默认 false。';
COMMENT ON COLUMN "ai_governance"."sys_access_policies"."deleted_at" IS '软删除生效时间，UTC；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."sys_access_policies"."deleted_by" IS '执行软删除的稳定主体标识；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."sys_access_policies"."delete_reason" IS '软删除业务原因；必须说明业务依据、影响范围并关联删除 AuditEvent，禁止空原因或无审计删除。字段约束：可空，仅 is_deleted=true 时必填。';
COMMENT ON TABLE "ai_governance"."sys_access_policies" IS 'P0 固定列 ABAC 策略；禁止 AST、SQL、脚本或自定义 DSL。';

-- ----------------------------
-- Table structure for sys_access_policy_bindings
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."sys_access_policy_bindings";
CREATE TABLE "ai_governance"."sys_access_policy_bindings" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "policy_id" uuid NOT NULL,
  "action_id" uuid NOT NULL,
  "role_id" uuid,
  "scope_type" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "scope_id" uuid NOT NULL,
  "status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "valid_from" timestamptz(6) NOT NULL,
  "valid_until" timestamptz(6),
  "priority" int4 NOT NULL DEFAULT 100,
  "revision" int8 NOT NULL DEFAULT 1,
  "row_version" int8 NOT NULL DEFAULT 1,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "created_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "updated_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "operation_source" varchar(32) COLLATE "pg_catalog"."default" NOT NULL DEFAULT 'ADMIN'::character varying,
  "operation_trace_id" varchar(128) COLLATE "pg_catalog"."default",
  "is_deleted" bool NOT NULL DEFAULT false,
  "deleted_at" timestamptz(6),
  "deleted_by" varchar(128) COLLATE "pg_catalog"."default",
  "delete_reason" varchar(512) COLLATE "pg_catalog"."default"
)
;
COMMENT ON COLUMN "ai_governance"."sys_access_policy_bindings"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."sys_access_policy_bindings"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_access_policy_bindings"."policy_id" IS '逻辑引用 策略主记录 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_access_policy_bindings"."action_id" IS '逻辑引用 sys_action_catalogs 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_access_policy_bindings"."role_id" IS '逻辑引用 sys_roles 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."sys_access_policy_bindings"."scope_type" IS 'Authorization Module 受控枚举；允许值：TENANT（租户作用域）、ORGANIZATION（组织作用域）、PROJECT（项目作用域）、USER（企业用户主体）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_access_policy_bindings"."scope_id" IS '逻辑引用 GovernanceScopeRef 指向的领域实体 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_access_policy_bindings"."status" IS 'Authorization Module 受控枚举；允许值：DRAFT（草稿未生效）、ACTIVE（有效并参与运行决策）、REVOKED（已撤销，不参与新决策）、EXPIRED（已过有效期）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_access_policy_bindings"."valid_from" IS 'Authorization Module UTC 业务时间 valid_from；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_access_policy_bindings"."valid_until" IS 'Authorization Module UTC 业务时间 valid_until；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."sys_access_policy_bindings"."priority" IS 'Authorization Module 顺序、计数或重放控制字段 priority；用于确定性排序、幂等、投影或完整性验证，禁止无审计跳变。 字段约束：必填，默认 100。';
COMMENT ON COLUMN "ai_governance"."sys_access_policy_bindings"."revision" IS '配置业务 revision；每次有效内容变化严格递增，旧 revision 不得覆盖新版本。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."sys_access_policy_bindings"."row_version" IS '乐观锁版本；每次成功 UPDATE 严格递增，用于防止并发覆盖。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."sys_access_policy_bindings"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."sys_access_policy_bindings"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."sys_access_policy_bindings"."created_by" IS '创建记录的稳定主体标识；保存 ID 或服务主体，不保存仅供展示的姓名。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_access_policy_bindings"."updated_by" IS '最后修改记录的稳定主体标识；必须与 AuditEvent.actor 可关联。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_access_policy_bindings"."operation_source" IS '变更来源；ADMIN=治理后台，API=管理接口，SYNC=上游同步，SYSTEM=平台任务，MIGRATION=迁移。 字段约束：必填，默认 ''ADMIN''。';
COMMENT ON COLUMN "ai_governance"."sys_access_policy_bindings"."operation_trace_id" IS '产生该次写入的 Trace 标识；用于跨 Module 联合排障和审计关联。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."sys_access_policy_bindings"."is_deleted" IS '软删除标志；false=有效，true=逻辑删除且必须同时填写 deleted_at、deleted_by、delete_reason。 字段约束：必填，默认 false。';
COMMENT ON COLUMN "ai_governance"."sys_access_policy_bindings"."deleted_at" IS '软删除生效时间，UTC；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."sys_access_policy_bindings"."deleted_by" IS '执行软删除的稳定主体标识；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."sys_access_policy_bindings"."delete_reason" IS '软删除业务原因；必须说明业务依据、影响范围并关联删除 AuditEvent，禁止空原因或无审计删除。字段约束：可空，仅 is_deleted=true 时必填。';
COMMENT ON TABLE "ai_governance"."sys_access_policy_bindings" IS '访问策略与动作、角色、Scope 的版本化绑定。';

-- ----------------------------
-- Table structure for sys_action_catalogs
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."sys_action_catalogs";
CREATE TABLE "ai_governance"."sys_action_catalogs" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "action_code" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "resource_type_code" varchar(64) COLLATE "pg_catalog"."default" NOT NULL,
  "verb_code" varchar(64) COLLATE "pg_catalog"."default" NOT NULL,
  "action_name" varchar(160) COLLATE "pg_catalog"."default" NOT NULL,
  "description" varchar(512) COLLATE "pg_catalog"."default" NOT NULL,
  "risk_level" varchar(16) COLLATE "pg_catalog"."default" NOT NULL,
  "requires_mfa" bool NOT NULL DEFAULT false,
  "requires_approval" bool NOT NULL DEFAULT false,
  "status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "revision" int8 NOT NULL DEFAULT 1,
  "row_version" int8 NOT NULL DEFAULT 1,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "created_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "updated_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "operation_source" varchar(32) COLLATE "pg_catalog"."default" NOT NULL DEFAULT 'ADMIN'::character varying,
  "operation_trace_id" varchar(128) COLLATE "pg_catalog"."default",
  "is_deleted" bool NOT NULL DEFAULT false,
  "deleted_at" timestamptz(6),
  "deleted_by" varchar(128) COLLATE "pg_catalog"."default",
  "delete_reason" varchar(512) COLLATE "pg_catalog"."default"
)
;
COMMENT ON COLUMN "ai_governance"."sys_action_catalogs"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."sys_action_catalogs"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_action_catalogs"."action_code" IS '权限动作唯一代码，严格等于 resource_type_code:verb_code；禁止星号、问号和隐式通配。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_action_catalogs"."resource_type_code" IS '动作作用的稳定资源类型代码；与 verb_code 组合成唯一 action_code。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_action_catalogs"."verb_code" IS '对资源执行的原子操作代码，例如 read、create、approve 或 revoke；不得混合多个行为。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_action_catalogs"."action_name" IS '动作的治理后台显示名称；机器授权只使用 action_code。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_action_catalogs"."description" IS '动作允许执行的业务行为、风险和边界说明；供管理员 Review，不参与机器授权计算。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_action_catalogs"."risk_level" IS 'Authorization Module 受控枚举；允许值：LOW（低等级）、MEDIUM（中等级）、HIGH（高等级）、CRITICAL（关键等级，触发强控制）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_action_catalogs"."requires_mfa" IS '动作是否要求 MFA；true=认证强度必须达到 Tenant 配置的 MFA 门槛，否则后端 PEP 拒绝，false=不额外要求 MFA。字段约束：必填，默认 false。';
COMMENT ON COLUMN "ai_governance"."sys_action_catalogs"."requires_approval" IS '动作执行前是否必须取得有效 ApprovalRequest。 字段约束：必填，默认 false。';
COMMENT ON COLUMN "ai_governance"."sys_action_catalogs"."status" IS 'Authorization Module 受控枚举；允许值：DRAFT（草稿未生效）、ACTIVE（有效并参与运行决策）、DEPRECATED（已弃用，兼容期只读）、RETIRED（退役终态，仅保留历史）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_action_catalogs"."revision" IS '配置业务 revision；每次有效内容变化严格递增，旧 revision 不得覆盖新版本。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."sys_action_catalogs"."row_version" IS '乐观锁版本；每次成功 UPDATE 严格递增，用于防止并发覆盖。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."sys_action_catalogs"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."sys_action_catalogs"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."sys_action_catalogs"."created_by" IS '创建记录的稳定主体标识；保存 ID 或服务主体，不保存仅供展示的姓名。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_action_catalogs"."updated_by" IS '最后修改记录的稳定主体标识；必须与 AuditEvent.actor 可关联。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_action_catalogs"."operation_source" IS '变更来源；ADMIN=治理后台，API=管理接口，SYNC=上游同步，SYSTEM=平台任务，MIGRATION=迁移。 字段约束：必填，默认 ''ADMIN''。';
COMMENT ON COLUMN "ai_governance"."sys_action_catalogs"."operation_trace_id" IS '产生该次写入的 Trace 标识；用于跨 Module 联合排障和审计关联。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."sys_action_catalogs"."is_deleted" IS '软删除标志；false=有效，true=逻辑删除且必须同时填写 deleted_at、deleted_by、delete_reason。 字段约束：必填，默认 false。';
COMMENT ON COLUMN "ai_governance"."sys_action_catalogs"."deleted_at" IS '软删除生效时间，UTC；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."sys_action_catalogs"."deleted_by" IS '执行软删除的稳定主体标识；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."sys_action_catalogs"."delete_reason" IS '软删除业务原因；必须说明业务依据、影响范围并关联删除 AuditEvent，禁止空原因或无审计删除。字段约束：可空，仅 is_deleted=true 时必填。';
COMMENT ON TABLE "ai_governance"."sys_action_catalogs" IS '授权动作目录；精确 resource:verb，不支持通配符。';

-- ----------------------------
-- Table structure for sys_role_permissions
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."sys_role_permissions";
CREATE TABLE "ai_governance"."sys_role_permissions" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "role_id" uuid NOT NULL,
  "action_id" uuid NOT NULL,
  "effect" varchar(8) COLLATE "pg_catalog"."default" NOT NULL,
  "status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "valid_from" timestamptz(6) NOT NULL,
  "valid_until" timestamptz(6),
  "revision" int8 NOT NULL DEFAULT 1,
  "row_version" int8 NOT NULL DEFAULT 1,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "created_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "updated_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "operation_source" varchar(32) COLLATE "pg_catalog"."default" NOT NULL DEFAULT 'ADMIN'::character varying,
  "operation_trace_id" varchar(128) COLLATE "pg_catalog"."default",
  "is_deleted" bool NOT NULL DEFAULT false,
  "deleted_at" timestamptz(6),
  "deleted_by" varchar(128) COLLATE "pg_catalog"."default",
  "delete_reason" varchar(512) COLLATE "pg_catalog"."default"
)
;
COMMENT ON COLUMN "ai_governance"."sys_role_permissions"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."sys_role_permissions"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_role_permissions"."role_id" IS '逻辑引用 sys_roles 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_role_permissions"."action_id" IS '逻辑引用 sys_action_catalogs 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_role_permissions"."effect" IS 'Authorization Module 受控枚举；允许值：ALLOW（允许）、DENY（拒绝，优先于允许）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_role_permissions"."status" IS 'Authorization Module 受控枚举；允许值：DRAFT（草稿未生效）、ACTIVE（有效并参与运行决策）、REVOKED（已撤销，不参与新决策）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_role_permissions"."valid_from" IS 'Authorization Module UTC 业务时间 valid_from；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_role_permissions"."valid_until" IS 'Authorization Module UTC 业务时间 valid_until；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."sys_role_permissions"."revision" IS '配置业务 revision；每次有效内容变化严格递增，旧 revision 不得覆盖新版本。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."sys_role_permissions"."row_version" IS '乐观锁版本；每次成功 UPDATE 严格递增，用于防止并发覆盖。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."sys_role_permissions"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."sys_role_permissions"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."sys_role_permissions"."created_by" IS '创建记录的稳定主体标识；保存 ID 或服务主体，不保存仅供展示的姓名。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_role_permissions"."updated_by" IS '最后修改记录的稳定主体标识；必须与 AuditEvent.actor 可关联。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_role_permissions"."operation_source" IS '变更来源；ADMIN=治理后台，API=管理接口，SYNC=上游同步，SYSTEM=平台任务，MIGRATION=迁移。 字段约束：必填，默认 ''ADMIN''。';
COMMENT ON COLUMN "ai_governance"."sys_role_permissions"."operation_trace_id" IS '产生该次写入的 Trace 标识；用于跨 Module 联合排障和审计关联。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."sys_role_permissions"."is_deleted" IS '软删除标志；false=有效，true=逻辑删除且必须同时填写 deleted_at、deleted_by、delete_reason。 字段约束：必填，默认 false。';
COMMENT ON COLUMN "ai_governance"."sys_role_permissions"."deleted_at" IS '软删除生效时间，UTC；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."sys_role_permissions"."deleted_by" IS '执行软删除的稳定主体标识；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."sys_role_permissions"."delete_reason" IS '软删除业务原因；必须说明业务依据、影响范围并关联删除 AuditEvent，禁止空原因或无审计删除。字段约束：可空，仅 is_deleted=true 时必填。';
COMMENT ON TABLE "ai_governance"."sys_role_permissions" IS '角色到动作的 ALLOW/DENY 配置。';

-- ----------------------------
-- Table structure for sys_roles
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."sys_roles";
CREATE TABLE "ai_governance"."sys_roles" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "role_code" varchar(64) COLLATE "pg_catalog"."default" NOT NULL,
  "role_name" varchar(160) COLLATE "pg_catalog"."default" NOT NULL,
  "role_domain" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "role_source_type" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "source_role_code" varchar(64) COLLATE "pg_catalog"."default",
  "description" varchar(512) COLLATE "pg_catalog"."default" NOT NULL,
  "status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "revision" int8 NOT NULL DEFAULT 1,
  "row_version" int8 NOT NULL DEFAULT 1,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "created_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "updated_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "operation_source" varchar(32) COLLATE "pg_catalog"."default" NOT NULL DEFAULT 'ADMIN'::character varying,
  "operation_trace_id" varchar(128) COLLATE "pg_catalog"."default",
  "is_deleted" bool NOT NULL DEFAULT false,
  "deleted_at" timestamptz(6),
  "deleted_by" varchar(128) COLLATE "pg_catalog"."default",
  "delete_reason" varchar(512) COLLATE "pg_catalog"."default"
)
;
COMMENT ON COLUMN "ai_governance"."sys_roles"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."sys_roles"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_roles"."role_code" IS '平台角色模板或领域角色映射的稳定代码；Tenant 内唯一且退役后不复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_roles"."role_name" IS '角色的治理后台显示名称；角色权限计算只使用 role_id、role_code 和 revision。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_roles"."role_domain" IS 'Authorization Module 受控枚举；允许值：PLATFORM（平台治理域）、ENTERPRISE（企业身份域）、ORGANIZATION（组织作用域）、PROJECT（项目作用域）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_roles"."role_source_type" IS 'Authorization Module 受控枚举；允许值：PLATFORM_LOCAL（平台本地角色绑定）、DOMAIN_MAPPING（领域角色到动作集合的映射）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_roles"."source_role_code" IS 'DOMAIN_MAPPING 对应的企业、组织或项目领域角色代码；PLATFORM_LOCAL 角色必须为空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."sys_roles"."description" IS '平台角色或领域角色映射的职责范围说明；禁止用描述文本隐式扩展动作权限。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_roles"."status" IS 'Authorization Module 受控枚举；允许值：DRAFT（草稿未生效）、ACTIVE（有效并参与运行决策）、SUSPENDED（暂停，不接受新增业务）、RETIRED（退役终态，仅保留历史）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_roles"."revision" IS '配置业务 revision；每次有效内容变化严格递增，旧 revision 不得覆盖新版本。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."sys_roles"."row_version" IS '乐观锁版本；每次成功 UPDATE 严格递增，用于防止并发覆盖。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."sys_roles"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."sys_roles"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."sys_roles"."created_by" IS '创建记录的稳定主体标识；保存 ID 或服务主体，不保存仅供展示的姓名。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_roles"."updated_by" IS '最后修改记录的稳定主体标识；必须与 AuditEvent.actor 可关联。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_roles"."operation_source" IS '变更来源；ADMIN=治理后台，API=管理接口，SYNC=上游同步，SYSTEM=平台任务，MIGRATION=迁移。 字段约束：必填，默认 ''ADMIN''。';
COMMENT ON COLUMN "ai_governance"."sys_roles"."operation_trace_id" IS '产生该次写入的 Trace 标识；用于跨 Module 联合排障和审计关联。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."sys_roles"."is_deleted" IS '软删除标志；false=有效，true=逻辑删除且必须同时填写 deleted_at、deleted_by、delete_reason。 字段约束：必填，默认 false。';
COMMENT ON COLUMN "ai_governance"."sys_roles"."deleted_at" IS '软删除生效时间，UTC；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."sys_roles"."deleted_by" IS '执行软删除的稳定主体标识；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."sys_roles"."delete_reason" IS '软删除业务原因；必须说明业务依据、影响范围并关联删除 AuditEvent，禁止空原因或无审计删除。字段约束：可空，仅 is_deleted=true 时必填。';
COMMENT ON TABLE "ai_governance"."sys_roles" IS '平台角色模板和领域角色到动作集合的映射；不拥有组织/项目成员事实。';

-- ----------------------------
-- Table structure for sys_subject_role_bindings
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."sys_subject_role_bindings";
CREATE TABLE "ai_governance"."sys_subject_role_bindings" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "subject_id" uuid NOT NULL,
  "role_id" uuid NOT NULL,
  "scope_type" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "scope_id" uuid NOT NULL,
  "binding_source" varchar(24) COLLATE "pg_catalog"."default" NOT NULL DEFAULT 'PLATFORM_LOCAL'::character varying,
  "status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "valid_from" timestamptz(6) NOT NULL,
  "valid_until" timestamptz(6),
  "approval_request_id" uuid,
  "revision" int8 NOT NULL DEFAULT 1,
  "row_version" int8 NOT NULL DEFAULT 1,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "created_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "updated_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "operation_source" varchar(32) COLLATE "pg_catalog"."default" NOT NULL DEFAULT 'ADMIN'::character varying,
  "operation_trace_id" varchar(128) COLLATE "pg_catalog"."default",
  "is_deleted" bool NOT NULL DEFAULT false,
  "deleted_at" timestamptz(6),
  "deleted_by" varchar(128) COLLATE "pg_catalog"."default",
  "delete_reason" varchar(512) COLLATE "pg_catalog"."default"
)
;
COMMENT ON COLUMN "ai_governance"."sys_subject_role_bindings"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."sys_subject_role_bindings"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_subject_role_bindings"."subject_id" IS '逻辑引用 IdentitySubject 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_subject_role_bindings"."role_id" IS '逻辑引用 sys_roles 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_subject_role_bindings"."scope_type" IS 'Authorization Module 受控枚举；允许值：TENANT（租户作用域）、ORGANIZATION（组织作用域）、PROJECT（项目作用域）、USER（企业用户主体）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_subject_role_bindings"."scope_id" IS '逻辑引用 GovernanceScopeRef 指向的领域实体 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_subject_role_bindings"."binding_source" IS '主体角色绑定来源，P0 固定 PLATFORM_LOCAL；组织和项目角色禁止复制到该表。 字段约束：必填，默认 ''PLATFORM_LOCAL''。';
COMMENT ON COLUMN "ai_governance"."sys_subject_role_bindings"."status" IS 'Authorization Module 受控枚举；允许值：PENDING（待处理或待生效）、ACTIVE（有效并参与运行决策）、SUSPENDED（暂停，不接受新增业务）、REVOKED（已撤销，不参与新决策）、EXPIRED（已过有效期）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_subject_role_bindings"."valid_from" IS 'Authorization Module UTC 业务时间 valid_from；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_subject_role_bindings"."valid_until" IS 'Authorization Module UTC 业务时间 valid_until；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."sys_subject_role_bindings"."approval_request_id" IS '关联 ApprovalRequest；敏感动作须校验同 Tenant、APPROVED、未过期且 payload_hash 一致。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."sys_subject_role_bindings"."revision" IS '配置业务 revision；每次有效内容变化严格递增，旧 revision 不得覆盖新版本。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."sys_subject_role_bindings"."row_version" IS '乐观锁版本；每次成功 UPDATE 严格递增，用于防止并发覆盖。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."sys_subject_role_bindings"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."sys_subject_role_bindings"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."sys_subject_role_bindings"."created_by" IS '创建记录的稳定主体标识；保存 ID 或服务主体，不保存仅供展示的姓名。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_subject_role_bindings"."updated_by" IS '最后修改记录的稳定主体标识；必须与 AuditEvent.actor 可关联。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_subject_role_bindings"."operation_source" IS '变更来源；ADMIN=治理后台，API=管理接口，SYNC=上游同步，SYSTEM=平台任务，MIGRATION=迁移。 字段约束：必填，默认 ''ADMIN''。';
COMMENT ON COLUMN "ai_governance"."sys_subject_role_bindings"."operation_trace_id" IS '产生该次写入的 Trace 标识；用于跨 Module 联合排障和审计关联。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."sys_subject_role_bindings"."is_deleted" IS '软删除标志；false=有效，true=逻辑删除且必须同时填写 deleted_at、deleted_by、delete_reason。 字段约束：必填，默认 false。';
COMMENT ON COLUMN "ai_governance"."sys_subject_role_bindings"."deleted_at" IS '软删除生效时间，UTC；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."sys_subject_role_bindings"."deleted_by" IS '执行软删除的稳定主体标识；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."sys_subject_role_bindings"."delete_reason" IS '软删除业务原因；必须说明业务依据、影响范围并关联删除 AuditEvent，禁止空原因或无审计删除。字段约束：可空，仅 is_deleted=true 时必填。';
COMMENT ON TABLE "ai_governance"."sys_subject_role_bindings" IS '仅保存平台本地角色的主体绑定；禁止复制组织/项目角色绑定。';

-- ----------------------------
-- Table structure for sys_ui_action_bindings
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."sys_ui_action_bindings";
CREATE TABLE "ai_governance"."sys_ui_action_bindings" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "target_type" varchar(16) COLLATE "pg_catalog"."default" NOT NULL,
  "target_id" uuid NOT NULL,
  "action_id" uuid NOT NULL,
  "binding_purpose" varchar(16) COLLATE "pg_catalog"."default" NOT NULL,
  "match_mode" varchar(16) COLLATE "pg_catalog"."default" NOT NULL DEFAULT 'ANY'::character varying,
  "status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "revision" int8 NOT NULL DEFAULT 1,
  "row_version" int8 NOT NULL DEFAULT 1,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "created_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "updated_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "operation_source" varchar(32) COLLATE "pg_catalog"."default" NOT NULL DEFAULT 'ADMIN'::character varying,
  "operation_trace_id" varchar(128) COLLATE "pg_catalog"."default",
  "is_deleted" bool NOT NULL DEFAULT false,
  "deleted_at" timestamptz(6),
  "deleted_by" varchar(128) COLLATE "pg_catalog"."default",
  "delete_reason" varchar(512) COLLATE "pg_catalog"."default"
)
;
COMMENT ON COLUMN "ai_governance"."sys_ui_action_bindings"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."sys_ui_action_bindings"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_ui_action_bindings"."target_type" IS 'UI Navigation Module 受控枚举；允许值：MENU（内部页面菜单）、ROUTE（路由配置域）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_ui_action_bindings"."target_id" IS '逻辑引用 target_type 指定的 Menu 或 Route 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_ui_action_bindings"."action_id" IS '逻辑引用 sys_action_catalogs 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_ui_action_bindings"."binding_purpose" IS 'UI Navigation Module 受控枚举；允许值：VISIBLE（判断导航可见）、ENTER（判断路由可进入）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_ui_action_bindings"."match_mode" IS 'UI Navigation Module 受控枚举；允许值：ANY（任一条件满足或不限定类别）、ALL（全部条件满足）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填，默认 ''ANY''。';
COMMENT ON COLUMN "ai_governance"."sys_ui_action_bindings"."status" IS 'UI Navigation Module 受控枚举；允许值：DRAFT（草稿未生效）、ACTIVE（有效并参与运行决策）、REVOKED（已撤销，不参与新决策）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_ui_action_bindings"."revision" IS '配置业务 revision；每次有效内容变化严格递增，旧 revision 不得覆盖新版本。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."sys_ui_action_bindings"."row_version" IS '乐观锁版本；每次成功 UPDATE 严格递增，用于防止并发覆盖。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."sys_ui_action_bindings"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."sys_ui_action_bindings"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."sys_ui_action_bindings"."created_by" IS '创建记录的稳定主体标识；保存 ID 或服务主体，不保存仅供展示的姓名。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_ui_action_bindings"."updated_by" IS '最后修改记录的稳定主体标识；必须与 AuditEvent.actor 可关联。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_ui_action_bindings"."operation_source" IS '变更来源；ADMIN=治理后台，API=管理接口，SYNC=上游同步，SYSTEM=平台任务，MIGRATION=迁移。 字段约束：必填，默认 ''ADMIN''。';
COMMENT ON COLUMN "ai_governance"."sys_ui_action_bindings"."operation_trace_id" IS '产生该次写入的 Trace 标识；用于跨 Module 联合排障和审计关联。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."sys_ui_action_bindings"."is_deleted" IS '软删除标志；false=有效，true=逻辑删除且必须同时填写 deleted_at、deleted_by、delete_reason。 字段约束：必填，默认 false。';
COMMENT ON COLUMN "ai_governance"."sys_ui_action_bindings"."deleted_at" IS '软删除生效时间，UTC；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."sys_ui_action_bindings"."deleted_by" IS '执行软删除的稳定主体标识；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."sys_ui_action_bindings"."delete_reason" IS '软删除业务原因；必须说明业务依据、影响范围并关联删除 AuditEvent，禁止空原因或无审计删除。字段约束：可空，仅 is_deleted=true 时必填。';
COMMENT ON TABLE "ai_governance"."sys_ui_action_bindings" IS '菜单/路由可见与可进入条件到动作目录的映射。';

-- ----------------------------
-- Table structure for sys_ui_menus
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."sys_ui_menus";
CREATE TABLE "ai_governance"."sys_ui_menus" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "parent_menu_id" uuid,
  "menu_code" varchar(64) COLLATE "pg_catalog"."default" NOT NULL,
  "menu_type" varchar(16) COLLATE "pg_catalog"."default" NOT NULL,
  "route_id" uuid,
  "title_key" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "icon_key" varchar(64) COLLATE "pg_catalog"."default",
  "external_url" varchar(1024) COLLATE "pg_catalog"."default",
  "sort_order" int4 NOT NULL DEFAULT 0,
  "status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "revision" int8 NOT NULL DEFAULT 1,
  "row_version" int8 NOT NULL DEFAULT 1,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "created_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "updated_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "operation_source" varchar(32) COLLATE "pg_catalog"."default" NOT NULL DEFAULT 'ADMIN'::character varying,
  "operation_trace_id" varchar(128) COLLATE "pg_catalog"."default",
  "is_deleted" bool NOT NULL DEFAULT false,
  "deleted_at" timestamptz(6),
  "deleted_by" varchar(128) COLLATE "pg_catalog"."default",
  "delete_reason" varchar(512) COLLATE "pg_catalog"."default"
)
;
COMMENT ON COLUMN "ai_governance"."sys_ui_menus"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."sys_ui_menus"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_ui_menus"."parent_menu_id" IS '逻辑引用 父 sys_ui_menus 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."sys_ui_menus"."menu_code" IS '菜单节点的稳定代码；用于构建树和发布差异，菜单标题变化不改变该值。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_ui_menus"."menu_type" IS 'UI Navigation Module 受控枚举；允许值：DIRECTORY（目录节点）、MENU（内部页面菜单）、LINK（外部链接）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_ui_menus"."route_id" IS '逻辑引用 sys_ui_routes 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."sys_ui_menus"."title_key" IS '菜单标题的国际化资源键；前端按语言解析，禁止直接存多语言文案。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_ui_menus"."icon_key" IS '前端图标 allowlist 键；只影响展示，不参与授权、路由匹配或业务事实判断。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."sys_ui_menus"."external_url" IS 'LINK 类型菜单的外部 HTTPS 地址；必须经过域名 allowlist 和安全审批，其他菜单类型必须为空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."sys_ui_menus"."sort_order" IS 'UI Navigation Module 顺序、计数或重放控制字段 sort_order；用于确定性排序、幂等、投影或完整性验证，禁止无审计跳变。 字段约束：必填，默认 0。';
COMMENT ON COLUMN "ai_governance"."sys_ui_menus"."status" IS 'UI Navigation Module 受控枚举；允许值：DRAFT（草稿未生效）、ACTIVE（有效并参与运行决策）、HIDDEN（对导航隐藏）、RETIRED（退役终态，仅保留历史）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_ui_menus"."revision" IS '配置业务 revision；每次有效内容变化严格递增，旧 revision 不得覆盖新版本。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."sys_ui_menus"."row_version" IS '乐观锁版本；每次成功 UPDATE 严格递增，用于防止并发覆盖。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."sys_ui_menus"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."sys_ui_menus"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."sys_ui_menus"."created_by" IS '创建记录的稳定主体标识；保存 ID 或服务主体，不保存仅供展示的姓名。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_ui_menus"."updated_by" IS '最后修改记录的稳定主体标识；必须与 AuditEvent.actor 可关联。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_ui_menus"."operation_source" IS '变更来源；ADMIN=治理后台，API=管理接口，SYNC=上游同步，SYSTEM=平台任务，MIGRATION=迁移。 字段约束：必填，默认 ''ADMIN''。';
COMMENT ON COLUMN "ai_governance"."sys_ui_menus"."operation_trace_id" IS '产生该次写入的 Trace 标识；用于跨 Module 联合排障和审计关联。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."sys_ui_menus"."is_deleted" IS '软删除标志；false=有效，true=逻辑删除且必须同时填写 deleted_at、deleted_by、delete_reason。 字段约束：必填，默认 false。';
COMMENT ON COLUMN "ai_governance"."sys_ui_menus"."deleted_at" IS '软删除生效时间，UTC；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."sys_ui_menus"."deleted_by" IS '执行软删除的稳定主体标识；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."sys_ui_menus"."delete_reason" IS '软删除业务原因；必须说明业务依据、影响范围并关联删除 AuditEvent，禁止空原因或无审计删除。字段约束：可空，仅 is_deleted=true 时必填。';
COMMENT ON TABLE "ai_governance"."sys_ui_menus" IS '菜单树读模型配置；菜单本身不是权限。';

-- ----------------------------
-- Table structure for sys_ui_routes
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."sys_ui_routes";
CREATE TABLE "ai_governance"."sys_ui_routes" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "route_code" varchar(64) COLLATE "pg_catalog"."default" NOT NULL,
  "route_path" varchar(256) COLLATE "pg_catalog"."default" NOT NULL,
  "route_name" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "component_key" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "layout_key" varchar(64) COLLATE "pg_catalog"."default" NOT NULL,
  "redirect_path" varchar(256) COLLATE "pg_catalog"."default",
  "is_hidden" bool NOT NULL DEFAULT false,
  "keep_alive" bool NOT NULL DEFAULT false,
  "status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "revision" int8 NOT NULL DEFAULT 1,
  "row_version" int8 NOT NULL DEFAULT 1,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "created_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "updated_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "operation_source" varchar(32) COLLATE "pg_catalog"."default" NOT NULL DEFAULT 'ADMIN'::character varying,
  "operation_trace_id" varchar(128) COLLATE "pg_catalog"."default",
  "is_deleted" bool NOT NULL DEFAULT false,
  "deleted_at" timestamptz(6),
  "deleted_by" varchar(128) COLLATE "pg_catalog"."default",
  "delete_reason" varchar(512) COLLATE "pg_catalog"."default"
)
;
COMMENT ON COLUMN "ai_governance"."sys_ui_routes"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."sys_ui_routes"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_ui_routes"."route_code" IS '前端路由的稳定代码；用于 RuntimeSnapshot 发布、版本差异比较和运维检索，不作为后端动作权限或组件加载键。字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_ui_routes"."route_path" IS '前端路由的规范化绝对路径，以斜杠开头；仅用于导航匹配，不作为后端权限标识。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_ui_routes"."route_name" IS '前端路由显示或调试名称；路由匹配使用 route_path，权限使用 action binding。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_ui_routes"."component_key" IS '前端构建产物中的组件 allowlist 键；禁止保存文件路径、任意模块名或可执行表达式。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_ui_routes"."layout_key" IS '前端构建产物中的布局 allowlist 键；用于安全解析页面骨架，不允许动态代码加载。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_ui_routes"."redirect_path" IS '进入路由后的可选内部重定向路径；必须属于当前 Tenant 已发布路由，不允许开放重定向。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."sys_ui_routes"."is_hidden" IS '路由是否从常规导航隐藏；隐藏不等于后端授权。 字段约束：必填，默认 false。';
COMMENT ON COLUMN "ai_governance"."sys_ui_routes"."keep_alive" IS '前端是否缓存路由组件实例；不影响后端 PEP。 字段约束：必填，默认 false。';
COMMENT ON COLUMN "ai_governance"."sys_ui_routes"."status" IS 'UI Navigation Module 受控枚举；允许值：DRAFT（草稿未生效）、ACTIVE（有效并参与运行决策）、HIDDEN（对导航隐藏）、RETIRED（退役终态，仅保留历史）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_ui_routes"."revision" IS '配置业务 revision；每次有效内容变化严格递增，旧 revision 不得覆盖新版本。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."sys_ui_routes"."row_version" IS '乐观锁版本；每次成功 UPDATE 严格递增，用于防止并发覆盖。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."sys_ui_routes"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."sys_ui_routes"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."sys_ui_routes"."created_by" IS '创建记录的稳定主体标识；保存 ID 或服务主体，不保存仅供展示的姓名。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_ui_routes"."updated_by" IS '最后修改记录的稳定主体标识；必须与 AuditEvent.actor 可关联。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."sys_ui_routes"."operation_source" IS '变更来源；ADMIN=治理后台，API=管理接口，SYNC=上游同步，SYSTEM=平台任务，MIGRATION=迁移。 字段约束：必填，默认 ''ADMIN''。';
COMMENT ON COLUMN "ai_governance"."sys_ui_routes"."operation_trace_id" IS '产生该次写入的 Trace 标识；用于跨 Module 联合排障和审计关联。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."sys_ui_routes"."is_deleted" IS '软删除标志；false=有效，true=逻辑删除且必须同时填写 deleted_at、deleted_by、delete_reason。 字段约束：必填，默认 false。';
COMMENT ON COLUMN "ai_governance"."sys_ui_routes"."deleted_at" IS '软删除生效时间，UTC；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."sys_ui_routes"."deleted_by" IS '执行软删除的稳定主体标识；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."sys_ui_routes"."delete_reason" IS '软删除业务原因；必须说明业务依据、影响范围并关联删除 AuditEvent，禁止空原因或无审计删除。字段约束：可空，仅 is_deleted=true 时必填。';
COMMENT ON TABLE "ai_governance"."sys_ui_routes" IS '行业标准前端路由元数据；组件与布局使用前端 allowlist key。';

-- ----------------------------
-- Table structure for tenants
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."tenants";
CREATE TABLE "ai_governance"."tenants" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "tenant_code" varchar(64) COLLATE "pg_catalog"."default" NOT NULL,
  "tenant_name" varchar(160) COLLATE "pg_catalog"."default" NOT NULL,
  "timezone" varchar(64) COLLATE "pg_catalog"."default" NOT NULL DEFAULT 'UTC'::character varying,
  "status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "revision" int8 NOT NULL DEFAULT 1,
  "row_version" int8 NOT NULL DEFAULT 1,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "created_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "updated_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "operation_source" varchar(32) COLLATE "pg_catalog"."default" NOT NULL DEFAULT 'ADMIN'::character varying,
  "operation_trace_id" varchar(128) COLLATE "pg_catalog"."default",
  "is_deleted" bool NOT NULL DEFAULT false,
  "deleted_at" timestamptz(6),
  "deleted_by" varchar(128) COLLATE "pg_catalog"."default",
  "delete_reason" varchar(512) COLLATE "pg_catalog"."default"
)
;
COMMENT ON COLUMN "ai_governance"."tenants"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."tenants"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."tenants"."tenant_code" IS 'Tenant 的稳定短代码；用于配置、日志和运营检索，Tenant 生命周期内不可复用给其他主体。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."tenants"."tenant_name" IS 'Tenant 的正式展示名称；允许经审批改名，不作为 Tenant 关联或隔离键。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."tenants"."timezone" IS 'Tenant 的 IANA 时区标识，仅用于业务周期和页面展示；数据库事实时间始终保存 UTC，禁止存 UTC 偏移缩写。 字段约束：必填，默认 ''UTC''。';
COMMENT ON COLUMN "ai_governance"."tenants"."status" IS 'Tenant 治理 受控枚举；允许值：DRAFT（草稿未生效）、ACTIVE（有效并参与运行决策）、SUSPENDED（暂停，不接受新增业务）、RETIRED（退役终态，仅保留历史）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."tenants"."revision" IS '配置业务 revision；每次有效内容变化严格递增，旧 revision 不得覆盖新版本。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."tenants"."row_version" IS '乐观锁版本；每次成功 UPDATE 严格递增，用于防止并发覆盖。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."tenants"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."tenants"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."tenants"."created_by" IS '创建记录的稳定主体标识；保存 ID 或服务主体，不保存仅供展示的姓名。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."tenants"."updated_by" IS '最后修改记录的稳定主体标识；必须与 AuditEvent.actor 可关联。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."tenants"."operation_source" IS '变更来源；ADMIN=治理后台，API=管理接口，SYNC=上游同步，SYSTEM=平台任务，MIGRATION=迁移。 字段约束：必填，默认 ''ADMIN''。';
COMMENT ON COLUMN "ai_governance"."tenants"."operation_trace_id" IS '产生该次写入的 Trace 标识；用于跨 Module 联合排障和审计关联。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."tenants"."is_deleted" IS '软删除标志；false=有效，true=逻辑删除且必须同时填写 deleted_at、deleted_by、delete_reason。 字段约束：必填，默认 false。';
COMMENT ON COLUMN "ai_governance"."tenants"."deleted_at" IS '软删除生效时间，UTC；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."tenants"."deleted_by" IS '执行软删除的稳定主体标识；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."tenants"."delete_reason" IS '软删除业务原因；必须说明业务依据、影响范围并关联删除 AuditEvent，禁止空原因或无审计删除。字段约束：可空，仅 is_deleted=true 时必填。';
COMMENT ON TABLE "ai_governance"."tenants" IS 'Tenant 主数据；tenant_id 必须与 id 相同。';

-- ----------------------------
-- Table structure for usage_attempts
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."usage_attempts";
CREATE TABLE "ai_governance"."usage_attempts" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "request_id" uuid NOT NULL,
  "attempt_no" int2 NOT NULL,
  "provider_id" uuid NOT NULL,
  "model_id" uuid NOT NULL,
  "channel_id" uuid NOT NULL,
  "provider_request_id" varchar(256) COLLATE "pg_catalog"."default",
  "status" varchar(32) COLLATE "pg_catalog"."default" NOT NULL,
  "started_at" timestamptz(6) NOT NULL,
  "first_content_at" timestamptz(6),
  "finished_at" timestamptz(6),
  "input_tokens" int8,
  "output_tokens" int8,
  "total_tokens" int8,
  "finish_reason" varchar(64) COLLATE "pg_catalog"."default",
  "error_class" varchar(32) COLLATE "pg_catalog"."default",
  "error_code" varchar(128) COLLATE "pg_catalog"."default",
  "retryable" bool,
  "response_digest" varchar(71) COLLATE "pg_catalog"."default",
  "row_version" int8 NOT NULL DEFAULT 1,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "created_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "updated_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "operation_trace_id" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "subject_id" uuid NOT NULL
)
;
COMMENT ON COLUMN "ai_governance"."usage_attempts"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."usage_attempts"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_attempts"."request_id" IS '逻辑引用 UsageRequest 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_attempts"."attempt_no" IS 'Usage Accounting 顺序、计数或重放控制字段 attempt_no；用于确定性排序、幂等、投影或完整性验证，禁止无审计跳变。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_attempts"."provider_id" IS '逻辑引用 ModelProvider 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_attempts"."model_id" IS '逻辑引用 ModelCatalogEntry 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_attempts"."channel_id" IS '逻辑引用 ModelChannel 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_attempts"."provider_request_id" IS 'Provider 返回的请求标识；仅用于供应商排障和 usage 核对，不向普通调用者暴露内部端点。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."usage_attempts"."status" IS 'Usage Accounting 受控枚举；允许值：PLANNED（已生成调用计划，尚未发送）、SENT（已发送到 Provider）、STREAMING（正在接收流式响应）、SUCCEEDED（尝试成功）、FAILED（执行失败）、CANCELLED（已取消）、UNKNOWN（状态未知）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_attempts"."started_at" IS 'Usage Accounting UTC 业务时间 started_at；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_attempts"."first_content_at" IS 'Usage Accounting UTC 业务时间 first_content_at；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."usage_attempts"."finished_at" IS 'Usage Accounting UTC 业务时间 finished_at；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."usage_attempts"."input_tokens" IS 'Usage Accounting Token 整数计量 input_tokens；单位为 token，禁止浮点和隐式换算，须关联 Request/Usage 或周期投影。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."usage_attempts"."output_tokens" IS 'Usage Accounting Token 整数计量 output_tokens；单位为 token，禁止浮点和隐式换算，须关联 Request/Usage 或周期投影。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."usage_attempts"."total_tokens" IS 'Usage Accounting Token 整数计量 total_tokens；单位为 token，禁止浮点和隐式换算，须关联 Request/Usage 或周期投影。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."usage_attempts"."finish_reason" IS 'Provider 或网关归一化的结束原因，例如 stop、length、content_filter；未结束时允许为空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."usage_attempts"."error_class" IS 'Usage Accounting 受控枚举；允许值：AUTH（Provider 或凭证认证错误）、RATE_LIMIT（Provider 限流错误）、TIMEOUT（Provider 调用超时）、NETWORK（网络连接或传输错误）、UPSTREAM_4XX（Provider 返回非认证、非限流的 4xx）、UPSTREAM_5XX（Provider 返回 5xx）、CONTENT（Provider 内容或安全错误）、CANCELLED（已取消）、UNKNOWN（状态未知）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."usage_attempts"."error_code" IS 'Provider 错误归一化后的稳定错误码；与 error_class、retryable 共同决定 Fallback。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."usage_attempts"."retryable" IS '稳定错误的重试判定；true=调用方可按 retry_after 重试，false=不得原请求重试，NULL=尚未完成错误分类且必须保守处理。字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."usage_attempts"."response_digest" IS 'Usage Accounting 完整性摘要 response_digest；保存算法前缀和摘要，用于幂等、篡改检测或重算，禁止敏感原文。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."usage_attempts"."row_version" IS '乐观锁版本；每次成功 UPDATE 严格递增，用于防止并发覆盖。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."usage_attempts"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."usage_attempts"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."usage_attempts"."created_by" IS '创建记录的稳定主体标识；保存 ID 或服务主体，不保存仅供展示的姓名。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_attempts"."updated_by" IS '最后修改记录的稳定主体标识；必须与 AuditEvent.actor 可关联。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_attempts"."operation_trace_id" IS '产生该次写入的 Trace 标识；用于跨 Module 联合排障和审计关联。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_attempts"."subject_id" IS 'Provider Attempt 对应的最小消费主体；从 UsageRequest 固化，禁止通过请求或 Fallback 改写。';
COMMENT ON TABLE "ai_governance"."usage_attempts" IS '每次 Provider 路由尝试；Fallback 独立记录但不独立最终结算。';

-- ----------------------------
-- Table structure for usage_events
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."usage_events";
CREATE TABLE "ai_governance"."usage_events" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "usage_event_id" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "request_id" uuid NOT NULL,
  "reservation_id" uuid NOT NULL,
  "effective_attempt_id" uuid NOT NULL,
  "provider_id" uuid NOT NULL,
  "model_id" uuid NOT NULL,
  "channel_id" uuid NOT NULL,
  "key_id" uuid NOT NULL,
  "key_revision" int8 NOT NULL,
  "user_allocation_id" uuid NOT NULL,
  "billing_scope_type" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "billing_scope_id" uuid NOT NULL,
  "credit_rate_version_id" uuid NOT NULL,
  "credit_rate_revision" int8 NOT NULL,
  "input_tokens" int8 NOT NULL,
  "output_tokens" int8 NOT NULL,
  "total_tokens" int8 NOT NULL,
  "settled_credit_amount" int8 NOT NULL,
  "usage_source" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "measurement_status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "provider_usage_digest" varchar(71) COLLATE "pg_catalog"."default",
  "occurred_at" timestamptz(6) NOT NULL,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "trace_id" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "subject_id" uuid NOT NULL
)
;
COMMENT ON COLUMN "ai_governance"."usage_events"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."usage_events"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_events"."usage_event_id" IS '最终 Usage 事实的稳定唯一标识；同一 Request 至多一个，用于 Ledger 结算和对账。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_events"."request_id" IS '逻辑引用 UsageRequest 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_events"."reservation_id" IS '逻辑引用 QuotaReservation 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_events"."effective_attempt_id" IS '逻辑引用 最终生效 UsageAttempt 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_events"."provider_id" IS '逻辑引用 ModelProvider 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_events"."model_id" IS '逻辑引用 ModelCatalogEntry 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_events"."channel_id" IS '逻辑引用 ModelChannel 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_events"."key_id" IS '逻辑引用 UserApiKey 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_events"."key_revision" IS '调用或授权时锁定的 User API Key revision；用于证明历史归属和动作上限。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_events"."user_allocation_id" IS '关联 UserAllocation；多把 Key 共享同一人员额度，禁止创建 Key 钱包。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_events"."billing_scope_type" IS '资金归属 Scope 类型；仅 ORGANIZATION 或 PROJECT，请求不得覆盖。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_events"."billing_scope_id" IS '资金归属组织或项目 UUID；必须与 Key、UserAllocation 和成员资格同 Tenant 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_events"."credit_rate_version_id" IS '逻辑引用 CreditRateVersion 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_events"."credit_rate_revision" IS 'Usage Accounting 整数 AI Credit 字段 credit_rate_revision；P0 不表达人民币，变化必须来自 Ledger 或其可重建投影。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_events"."input_tokens" IS 'Usage Accounting Token 整数计量 input_tokens；单位为 token，禁止浮点和隐式换算，须关联 Request/Usage 或周期投影。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_events"."output_tokens" IS 'Usage Accounting Token 整数计量 output_tokens；单位为 token，禁止浮点和隐式换算，须关联 Request/Usage 或周期投影。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_events"."total_tokens" IS 'Usage Accounting Token 整数计量 total_tokens；单位为 token，禁止浮点和隐式换算，须关联 Request/Usage 或周期投影。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_events"."settled_credit_amount" IS 'Usage Accounting 整数 AI Credit 字段 settled_credit_amount；P0 不表达人民币，变化必须来自 Ledger 或其可重建投影。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_events"."usage_source" IS 'Usage Accounting 受控枚举；允许值：PROVIDER（Provider 返回用量）、GATEWAY（网关测量用量）、RECONCILIATION（对账确定用量）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_events"."measurement_status" IS 'Usage Accounting 受控枚举；允许值：MEASURED（已按 Provider 或网关证据计量）、RECONCILED（经对账确定用量）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_events"."provider_usage_digest" IS 'Usage Accounting 完整性摘要 provider_usage_digest；保存算法前缀和摘要，用于幂等、篡改检测或重算，禁止敏感原文。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."usage_events"."occurred_at" IS 'Usage Accounting UTC 业务时间 occurred_at；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_events"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."usage_events"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."usage_events"."trace_id" IS '端到端 Trace 标识；贯通 Request、Attempt、Usage、Ledger、安全和 Audit。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_events"."subject_id" IS '最终 Usage 所属 User Subject；从 Request 固化并用于 User 账单、审计、对账和日结。';
COMMENT ON TABLE "ai_governance"."usage_events" IS '一次请求唯一的最终已知 Token 与 AI Credit 计量事实；未知 usage 暂不生成。';

-- ----------------------------
-- Table structure for usage_requests
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."usage_requests";
CREATE TABLE "ai_governance"."usage_requests" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "request_id" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "trace_id" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "subject_id" uuid NOT NULL,
  "subject_type" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "key_id" uuid NOT NULL,
  "key_revision" int8 NOT NULL,
  "user_allocation_id" uuid NOT NULL,
  "billing_scope_type" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "billing_scope_id" uuid NOT NULL,
  "idempotency_key" varchar(256) COLLATE "pg_catalog"."default" NOT NULL,
  "request_digest" varchar(71) COLLATE "pg_catalog"."default" NOT NULL,
  "requested_model_alias" varchar(96) COLLATE "pg_catalog"."default" NOT NULL,
  "capability" varchar(32) COLLATE "pg_catalog"."default" NOT NULL,
  "declared_data_classification" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "effective_data_classification" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "estimated_input_tokens" int8 NOT NULL,
  "maximum_output_tokens" int8 NOT NULL,
  "authorization_revision" int8 NOT NULL,
  "authorization_decision_hash" varchar(71) COLLATE "pg_catalog"."default" NOT NULL,
  "model_policy_revision" int8 NOT NULL,
  "firewall_policy_revision" int8 NOT NULL,
  "route_policy_revision" int8 NOT NULL,
  "external_context_json" jsonb NOT NULL DEFAULT '{}'::jsonb,
  "status" varchar(32) COLLATE "pg_catalog"."default" NOT NULL,
  "accepted_at" timestamptz(6),
  "terminal_at" timestamptz(6),
  "row_version" int8 NOT NULL DEFAULT 1,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "created_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "updated_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "operation_trace_id" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "subject_identity_revision" int8 NOT NULL
)
;
COMMENT ON COLUMN "ai_governance"."usage_requests"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."usage_requests"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_requests"."request_id" IS '对外可见且 Tenant 内唯一的模型请求标识；用于查询、取消、幂等恢复和全链路下钻。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_requests"."trace_id" IS '端到端 Trace 标识；贯通 Request、Attempt、Usage、Ledger、安全和 Audit。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_requests"."subject_id" IS '逻辑引用 IdentitySubject 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_requests"."subject_type" IS 'Usage Accounting 受控枚举；允许值：USER（企业用户主体）、SERVICE（服务主体）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_requests"."key_id" IS '逻辑引用 UserApiKey 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_requests"."key_revision" IS '调用或授权时锁定的 User API Key revision；用于证明历史归属和动作上限。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_requests"."user_allocation_id" IS '关联 UserAllocation；多把 Key 共享同一人员额度，禁止创建 Key 钱包。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_requests"."billing_scope_type" IS '资金归属 Scope 类型；仅 ORGANIZATION 或 PROJECT，请求不得覆盖。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_requests"."billing_scope_id" IS '资金归属组织或项目 UUID；必须与 Key、UserAllocation 和成员资格同 Tenant 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_requests"."idempotency_key" IS '命令幂等键；同一作用域内相同键和摘要恢复原结果，不同摘要必须冲突。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_requests"."request_digest" IS '规范化请求的 SHA-256 摘要；与 Key 作用域内 Idempotency-Key 共同识别重放，摘要冲突必须拒绝，禁止保存请求正文。字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_requests"."requested_model_alias" IS '调用方请求的逻辑模型别名；实际模型由授权、安全和 RoutePlan 决定，不可据此扩权。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_requests"."capability" IS '本次请求需要的标准模型能力；用于模型授权、Channel 过滤和 Provider Adapter 选择。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_requests"."declared_data_classification" IS 'Usage Accounting 受控枚举；允许值：PUBLIC（公开数据）、INTERNAL（内部网络或内部数据）、CONFIDENTIAL（机密数据，仅允许更严格处理）、RESTRICTED（受限数据，仅批准私有模型）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_requests"."effective_data_classification" IS 'Usage Accounting 受控枚举；允许值：PUBLIC（公开数据）、INTERNAL（内部网络或内部数据）、CONFIDENTIAL（机密数据，仅允许更严格处理）、RESTRICTED（受限数据，仅批准私有模型）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_requests"."estimated_input_tokens" IS 'Usage Accounting Token 整数计量 estimated_input_tokens；单位为 token，禁止浮点和隐式换算，须关联 Request/Usage 或周期投影。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_requests"."maximum_output_tokens" IS 'Usage Accounting Token 整数计量 maximum_output_tokens；单位为 token，禁止浮点和隐式换算，须关联 Request/Usage 或周期投影。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_requests"."authorization_revision" IS '授权 RuntimeSnapshot revision；用于重放当次动作决策。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_requests"."authorization_decision_hash" IS 'Usage Accounting 完整性摘要 authorization_decision_hash；保存算法前缀和摘要，用于幂等、篡改检测或重算，禁止敏感原文。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_requests"."model_policy_revision" IS 'Usage Accounting 不可回退版本 model_policy_revision；发布、决策和历史事实必须锁定当时值。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_requests"."firewall_policy_revision" IS 'Usage Accounting 不可回退版本 firewall_policy_revision；发布、决策和历史事实必须锁定当时值。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_requests"."route_policy_revision" IS 'Usage Accounting 不可回退版本 route_policy_revision；发布、决策和历史事实必须锁定当时值。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_requests"."external_context_json" IS 'Usage Accounting JSONB 证据或扩展快照 external_context_json；不承载核心可检索关系，值必须为 object。 字段约束：必填，默认 ''{}''::jsonb。';
COMMENT ON COLUMN "ai_governance"."usage_requests"."status" IS 'Usage Accounting 受控枚举；允许值：CREATED（请求事实已创建）、RESERVED（额度预占成功）、IN_FLIGHT（Provider 可能已接受请求）、CANCELLING（取消传播中）、SETTLED（已按最终用量结算）、RELEASED（预占已确认释放）、RECONCILING（用量未知，等待核对）、CANCELLED（已取消）、FAILED（执行失败）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_requests"."accepted_at" IS 'Usage Accounting UTC 业务时间 accepted_at；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."usage_requests"."terminal_at" IS 'Usage Accounting UTC 业务时间 terminal_at；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."usage_requests"."row_version" IS '乐观锁版本；每次成功 UPDATE 严格递增，用于防止并发覆盖。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."usage_requests"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."usage_requests"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."usage_requests"."created_by" IS '创建记录的稳定主体标识；保存 ID 或服务主体，不保存仅供展示的姓名。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_requests"."updated_by" IS '最后修改记录的稳定主体标识；必须与 AuditEvent.actor 可关联。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_requests"."operation_trace_id" IS '产生该次写入的 Trace 标识；用于跨 Module 联合排障和审计关联。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."usage_requests"."subject_identity_revision" IS '调用开始时锁定的 IdentitySubject revision；用于证明该 User 在请求时的身份状态，必须为正整数。';
COMMENT ON TABLE "ai_governance"."usage_requests" IS '一次受治理模型请求事实；包含 Key、Allocation、Scope、授权和策略 revision。';

-- ----------------------------
-- Table structure for user_allocations
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."user_allocations";
CREATE TABLE "ai_governance"."user_allocations" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "subject_id" uuid NOT NULL,
  "billing_scope_type" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "billing_scope_id" uuid NOT NULL,
  "available_account_id" uuid NOT NULL,
  "reserved_account_id" uuid NOT NULL,
  "allocation_code" varchar(96) COLLATE "pg_catalog"."default" NOT NULL,
  "status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "valid_from" timestamptz(6) NOT NULL,
  "valid_until" timestamptz(6),
  "revision" int8 NOT NULL DEFAULT 1,
  "row_version" int8 NOT NULL DEFAULT 1,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "created_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "updated_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "operation_source" varchar(32) COLLATE "pg_catalog"."default" NOT NULL DEFAULT 'ADMIN'::character varying,
  "operation_trace_id" varchar(128) COLLATE "pg_catalog"."default",
  "is_deleted" bool NOT NULL DEFAULT false,
  "deleted_at" timestamptz(6),
  "deleted_by" varchar(128) COLLATE "pg_catalog"."default",
  "delete_reason" varchar(512) COLLATE "pg_catalog"."default"
)
;
COMMENT ON COLUMN "ai_governance"."user_allocations"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."user_allocations"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."user_allocations"."subject_id" IS '逻辑引用 IdentitySubject 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."user_allocations"."billing_scope_type" IS '资金归属 Scope 类型；仅 ORGANIZATION 或 PROJECT，请求不得覆盖。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."user_allocations"."billing_scope_id" IS '资金归属组织或项目 UUID；必须与 Key、UserAllocation 和成员资格同 Tenant 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."user_allocations"."available_account_id" IS '逻辑引用 USER_AVAILABLE FundingAccount 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."user_allocations"."reserved_account_id" IS '逻辑引用 USER_RESERVED FundingAccount 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."user_allocations"."allocation_code" IS 'User 与 BillingScope 额度关系的稳定编号；贯通拨付、Key、账本和对账。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."user_allocations"."status" IS 'Funding Module 受控枚举；允许值：PENDING（待处理或待生效）、ACTIVE（有效并参与运行决策）、FROZEN（冻结，不接受普通新增交易）、CLOSED（关闭终态，仅保留历史）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."user_allocations"."valid_from" IS 'Funding Module UTC 业务时间 valid_from；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."user_allocations"."valid_until" IS 'Funding Module UTC 业务时间 valid_until；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."user_allocations"."revision" IS '配置业务 revision；每次有效内容变化严格递增，旧 revision 不得覆盖新版本。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."user_allocations"."row_version" IS '乐观锁版本；每次成功 UPDATE 严格递增，用于防止并发覆盖。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."user_allocations"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."user_allocations"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."user_allocations"."created_by" IS '创建记录的稳定主体标识；保存 ID 或服务主体，不保存仅供展示的姓名。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."user_allocations"."updated_by" IS '最后修改记录的稳定主体标识；必须与 AuditEvent.actor 可关联。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."user_allocations"."operation_source" IS '变更来源；ADMIN=治理后台，API=管理接口，SYNC=上游同步，SYSTEM=平台任务，MIGRATION=迁移。 字段约束：必填，默认 ''ADMIN''。';
COMMENT ON COLUMN "ai_governance"."user_allocations"."operation_trace_id" IS '产生该次写入的 Trace 标识；用于跨 Module 联合排障和审计关联。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."user_allocations"."is_deleted" IS '软删除标志；false=有效，true=逻辑删除且必须同时填写 deleted_at、deleted_by、delete_reason。 字段约束：必填，默认 false。';
COMMENT ON COLUMN "ai_governance"."user_allocations"."deleted_at" IS '软删除生效时间，UTC；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."user_allocations"."deleted_by" IS '执行软删除的稳定主体标识；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."user_allocations"."delete_reason" IS '软删除业务原因；必须说明业务依据、影响范围并关联删除 AuditEvent，禁止空原因或无审计删除。字段约束：可空，仅 is_deleted=true 时必填。';
COMMENT ON TABLE "ai_governance"."user_allocations" IS '用户在项目或组织 Scope 下可用额度的唯一归属事实；多 Key 共享。';

-- ----------------------------
-- Table structure for user_api_key_secret_versions
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."user_api_key_secret_versions";
CREATE TABLE "ai_governance"."user_api_key_secret_versions" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "key_id" uuid NOT NULL,
  "secret_version" int4 NOT NULL,
  "key_prefix" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "secret_verifier" varchar(512) COLLATE "pg_catalog"."default" NOT NULL,
  "pepper_ref" varchar(512) COLLATE "pg_catalog"."default" NOT NULL,
  "algorithm" varchar(32) COLLATE "pg_catalog"."default" NOT NULL,
  "status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "valid_from" timestamptz(6) NOT NULL,
  "valid_until" timestamptz(6) NOT NULL,
  "activated_at" timestamptz(6),
  "revoked_at" timestamptz(6),
  "revoke_reason" varchar(512) COLLATE "pg_catalog"."default",
  "row_version" int8 NOT NULL DEFAULT 1,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "created_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "updated_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "operation_trace_id" varchar(128) COLLATE "pg_catalog"."default"
)
;
COMMENT ON COLUMN "ai_governance"."user_api_key_secret_versions"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."user_api_key_secret_versions"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."user_api_key_secret_versions"."key_id" IS '逻辑引用 UserApiKey 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."user_api_key_secret_versions"."secret_version" IS '同一逻辑 Key 的 Secret 版本序号；从 1 递增，轮换创建新版本而不覆盖旧 verifier。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."user_api_key_secret_versions"."key_prefix" IS '用于快速定位 Key Secret 版本的非敏感前缀；不得足以还原 Secret，日志最多记录该前缀。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."user_api_key_secret_versions"."secret_verifier" IS 'Secret 经批准 KDF 与 Pepper 处理后的校验值；不可逆、不可用于调用，原始 Secret 永不落库。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."user_api_key_secret_versions"."pepper_ref" IS 'User Key Center 外部安全引用 pepper_ref；仅保存 Vault/KMS/对象定位符，不保存 Secret、凭证或对象正文。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."user_api_key_secret_versions"."algorithm" IS 'Secret verifier 使用的算法和参数版本代码；轮换算法时创建新 SecretVersion，不覆盖历史。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."user_api_key_secret_versions"."status" IS 'User Key Center 受控枚举；允许值：PENDING（待处理或待生效）、ACTIVE（有效并参与运行决策）、ROTATING（轮换窗口，新旧凭证受控并存）、REVOKED（已撤销，不参与新决策）、EXPIRED（已过有效期）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."user_api_key_secret_versions"."valid_from" IS 'User Key Center UTC 业务时间 valid_from；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."user_api_key_secret_versions"."valid_until" IS 'User Key Center UTC 业务时间 valid_until；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."user_api_key_secret_versions"."activated_at" IS 'User Key Center UTC 业务时间 activated_at；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."user_api_key_secret_versions"."revoked_at" IS 'User Key Center UTC 业务时间 revoked_at；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."user_api_key_secret_versions"."revoke_reason" IS 'SecretVersion 撤销原因；status=REVOKED 时必填，说明泄露、轮换或治理依据。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."user_api_key_secret_versions"."row_version" IS '乐观锁版本；每次成功 UPDATE 严格递增，用于防止并发覆盖。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."user_api_key_secret_versions"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."user_api_key_secret_versions"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."user_api_key_secret_versions"."created_by" IS '创建记录的稳定主体标识；保存 ID 或服务主体，不保存仅供展示的姓名。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."user_api_key_secret_versions"."updated_by" IS '最后修改记录的稳定主体标识；必须与 AuditEvent.actor 可关联。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."user_api_key_secret_versions"."operation_trace_id" IS '产生该次写入的 Trace 标识；用于跨 Module 联合排障和审计关联。 字段约束：可空。';
COMMENT ON TABLE "ai_governance"."user_api_key_secret_versions" IS 'Key Secret 校验版本；原始 Secret 永不落库。';

-- ----------------------------
-- Table structure for user_api_keys
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."user_api_keys";
CREATE TABLE "ai_governance"."user_api_keys" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "key_code" varchar(96) COLLATE "pg_catalog"."default" NOT NULL,
  "key_name" varchar(160) COLLATE "pg_catalog"."default" NOT NULL,
  "owner_subject_id" uuid NOT NULL,
  "billing_scope_type" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "billing_scope_id" uuid NOT NULL,
  "user_allocation_id" uuid NOT NULL,
  "key_limit_policy_id" uuid NOT NULL,
  "allowed_actions" varchar(128)[] COLLATE "pg_catalog"."default" NOT NULL,
  "status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "key_revision" int8 NOT NULL DEFAULT 1,
  "issued_at" timestamptz(6) NOT NULL,
  "expires_at" timestamptz(6) NOT NULL,
  "last_used_at" timestamptz(6),
  "row_version" int8 NOT NULL DEFAULT 1,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "created_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "updated_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "operation_source" varchar(32) COLLATE "pg_catalog"."default" NOT NULL DEFAULT 'ADMIN'::character varying,
  "operation_trace_id" varchar(128) COLLATE "pg_catalog"."default",
  "is_deleted" bool NOT NULL DEFAULT false,
  "deleted_at" timestamptz(6),
  "deleted_by" varchar(128) COLLATE "pg_catalog"."default",
  "delete_reason" varchar(512) COLLATE "pg_catalog"."default"
)
;
COMMENT ON COLUMN "ai_governance"."user_api_keys"."id" IS '应用生成的 UUIDv7 主键；全局不透明，不表达数据库自增或业务顺序。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."user_api_keys"."tenant_id" IS 'Tenant 隔离键；所有读写、唯一性、缓存键和逻辑引用校验必须携带，禁止跨 Tenant 复用。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."user_api_keys"."key_code" IS 'User API Key 的非敏感稳定编号；用于管理和审计，不是 Secret 或 Key 前缀。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."user_api_keys"."key_name" IS '用户为 Key 设置的管理显示名；可改名，不影响 key_code、Secret、归属或历史账单。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."user_api_keys"."owner_subject_id" IS '逻辑引用 Owner IdentitySubject 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."user_api_keys"."billing_scope_type" IS '资金归属 Scope 类型；仅 ORGANIZATION 或 PROJECT，请求不得覆盖。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."user_api_keys"."billing_scope_id" IS '资金归属组织或项目 UUID；必须与 Key、UserAllocation 和成员资格同 Tenant 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."user_api_keys"."user_allocation_id" IS '关联 UserAllocation；多把 Key 共享同一人员额度，禁止创建 Key 钱包。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."user_api_keys"."key_limit_policy_id" IS '逻辑引用 KeyLimitPolicy 的 UUID；无数据库外键，写入时校验目标存在、同 Tenant、状态允许和 revision 一致。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."user_api_keys"."allowed_actions" IS 'Key 可执行动作的硬上限集合；只能收紧主体权限，元素必须来自 sys_action_catalogs。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."user_api_keys"."status" IS 'User Key Center 受控枚举；允许值：PENDING（待处理或待生效）、ACTIVE（有效并参与运行决策）、SUSPENDED（暂停，不接受新增业务）、REVOKED（已撤销，不参与新决策）、EXPIRED（已过有效期）。未知值默认拒绝，迁移遵循状态与生命周期契约。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."user_api_keys"."key_revision" IS '调用或授权时锁定的 User API Key revision；用于证明历史归属和动作上限。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."user_api_keys"."issued_at" IS 'User Key Center UTC 业务时间 issued_at；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."user_api_keys"."expires_at" IS 'User Key Center UTC 业务时间 expires_at；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."user_api_keys"."last_used_at" IS 'User Key Center UTC 业务时间 last_used_at；用于生命周期、有效期、排序或审计，禁止本地时区隐式值。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."user_api_keys"."row_version" IS '乐观锁版本；每次成功 UPDATE 严格递增，用于防止并发覆盖。 字段约束：必填，默认 1。';
COMMENT ON COLUMN "ai_governance"."user_api_keys"."created_at" IS '记录创建时间，UTC timestamptz(6)；用于审计、归档和增量扫描。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."user_api_keys"."updated_at" IS '记录最后更新时间，UTC timestamptz(6)；追加式事实创建后必须保持不变。 字段约束：必填，默认 clock_timestamp()。';
COMMENT ON COLUMN "ai_governance"."user_api_keys"."created_by" IS '创建记录的稳定主体标识；保存 ID 或服务主体，不保存仅供展示的姓名。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."user_api_keys"."updated_by" IS '最后修改记录的稳定主体标识；必须与 AuditEvent.actor 可关联。 字段约束：必填。';
COMMENT ON COLUMN "ai_governance"."user_api_keys"."operation_source" IS '变更来源；ADMIN=治理后台，API=管理接口，SYNC=上游同步，SYSTEM=平台任务，MIGRATION=迁移。 字段约束：必填，默认 ''ADMIN''。';
COMMENT ON COLUMN "ai_governance"."user_api_keys"."operation_trace_id" IS '产生该次写入的 Trace 标识；用于跨 Module 联合排障和审计关联。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."user_api_keys"."is_deleted" IS '软删除标志；false=有效，true=逻辑删除且必须同时填写 deleted_at、deleted_by、delete_reason。 字段约束：必填，默认 false。';
COMMENT ON COLUMN "ai_governance"."user_api_keys"."deleted_at" IS '软删除生效时间，UTC；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."user_api_keys"."deleted_by" IS '执行软删除的稳定主体标识；仅 is_deleted=true 时允许非空。 字段约束：可空。';
COMMENT ON COLUMN "ai_governance"."user_api_keys"."delete_reason" IS '软删除业务原因；必须说明业务依据、影响范围并关联删除 AuditEvent，禁止空原因或无审计删除。字段约束：可空，仅 is_deleted=true 时必填。';
COMMENT ON TABLE "ai_governance"."user_api_keys" IS '用户 API Key 元数据、归属和硬上限；Key 不拥有钱包。';

-- ----------------------------
-- Table structure for user_closing_snapshot_items
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."user_closing_snapshot_items";
CREATE TABLE "ai_governance"."user_closing_snapshot_items" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "closing_snapshot_id" uuid NOT NULL,
  "reconciliation_run_id" uuid NOT NULL,
  "subject_id" uuid NOT NULL,
  "user_allocation_id" uuid NOT NULL,
  "billing_scope_type" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "billing_scope_id" uuid NOT NULL,
  "window_start" timestamptz(6) NOT NULL,
  "window_end" timestamptz(6) NOT NULL,
  "available_credit" int8 NOT NULL,
  "reserved_credit" int8 NOT NULL,
  "consumed_credit" int8 NOT NULL,
  "settled_tokens" int8 NOT NULL,
  "active_key_count" int4 NOT NULL,
  "open_reservation_count" int4 NOT NULL,
  "emergency_drawn_credit" int8 NOT NULL,
  "unresolved_difference_count" int4 NOT NULL,
  "fact_watermark" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "checksum_sha256" varchar(71) COLLATE "pg_catalog"."default" NOT NULL,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp()
)
;
COMMENT ON COLUMN "ai_governance"."user_closing_snapshot_items"."id" IS '应用生成的 UUIDv7 User 日结明细主键；事实只追加且不得更新或删除。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_closing_snapshot_items"."tenant_id" IS '日结明细所属 Tenant；所有 User 日结查询必须先限定该隔离键。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_closing_snapshot_items"."closing_snapshot_id" IS '父 ClosingSnapshot 逻辑引用；同一日结与 Allocation 只允许一条 User 明细。';
COMMENT ON COLUMN "ai_governance"."user_closing_snapshot_items"."reconciliation_run_id" IS '生成该 User 明细的 ReconciliationRun 逻辑引用，用于追溯检查与差异。';
COMMENT ON COLUMN "ai_governance"."user_closing_snapshot_items"."subject_id" IS '日结明细所属 User Subject；不得只存在于 JSON 摘要或通过 Key 反向推导。';
COMMENT ON COLUMN "ai_governance"."user_closing_snapshot_items"."user_allocation_id" IS '日结明细所属 UserAllocation；同一 User 不同 Scope 分别形成明细。';
COMMENT ON COLUMN "ai_governance"."user_closing_snapshot_items"."billing_scope_type" IS '日结资金范围类型；ORGANIZATION 为组织，PROJECT 为项目。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_closing_snapshot_items"."billing_scope_id" IS '日结资金范围 ID；与类型成对保存并锁定历史归属。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_closing_snapshot_items"."window_start" IS '日结固定窗口开始 UTC 时间；与父 ClosingSnapshot 必须一致。';
COMMENT ON COLUMN "ai_governance"."user_closing_snapshot_items"."window_end" IS '日结固定窗口结束 UTC 时间；必须晚于开始时间且与父事实一致。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_closing_snapshot_items"."available_credit" IS '窗口结束时 UserAllocation 可用 AI Credit；整数单位且必须非负。';
COMMENT ON COLUMN "ai_governance"."user_closing_snapshot_items"."reserved_credit" IS '窗口结束时 UserAllocation 已预占 AI Credit；整数单位且必须非负。';
COMMENT ON COLUMN "ai_governance"."user_closing_snapshot_items"."consumed_credit" IS '窗口内该 UserAllocation 已结算消耗 AI Credit；整数单位且必须非负。';
COMMENT ON COLUMN "ai_governance"."user_closing_snapshot_items"."settled_tokens" IS '窗口内该 UserAllocation 已结算 Token；整数单位且必须非负。';
COMMENT ON COLUMN "ai_governance"."user_closing_snapshot_items"."active_key_count" IS '窗口结束时绑定该 Allocation 的有效 Key 数量；单位为把。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_closing_snapshot_items"."open_reservation_count" IS '窗口结束时仍未收敛 Reservation 数量；单位为条并需进入次期跟踪。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_closing_snapshot_items"."emergency_drawn_credit" IS '窗口内该 UserAllocation 使用紧急信用总额；整数 AI Credit 且必须可追溯审批。';
COMMENT ON COLUMN "ai_governance"."user_closing_snapshot_items"."unresolved_difference_count" IS '关闭时该 User/Allocation 未解决差异数量；正式关闭要求为零。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_closing_snapshot_items"."fact_watermark" IS '生成日结明细所使用的权威事实水位；迟到事实进入下一周期并引用本期。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_closing_snapshot_items"."checksum_sha256" IS 'User 日结规范化字段与来源引用的 SHA-256 校验和；用于检测替换或篡改。';
COMMENT ON COLUMN "ai_governance"."user_closing_snapshot_items"."created_at" IS 'User 日结明细追加写入时间；数据库生成且不可更新。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_closing_snapshot_items"."updated_at" IS '标准事实字段；追加后保持创建值，任何 UPDATE 由触发器拒绝。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON TABLE "ai_governance"."user_closing_snapshot_items" IS '日结中的结构化 User/Allocation 明细事实；防止 Tenant 总盘平衡掩盖用户间串账。';

-- ----------------------------
-- Table structure for user_contact_points
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."user_contact_points";
CREATE TABLE "ai_governance"."user_contact_points" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "subject_id" uuid NOT NULL,
  "contact_type" varchar(16) COLLATE "pg_catalog"."default" NOT NULL,
  "normalized_lookup_hmac" varchar(71) COLLATE "pg_catalog"."default" NOT NULL,
  "contact_value_ciphertext" text COLLATE "pg_catalog"."default" NOT NULL,
  "encryption_key_ref" varchar(512) COLLATE "pg_catalog"."default" NOT NULL,
  "masked_value" varchar(160) COLLATE "pg_catalog"."default" NOT NULL,
  "verification_status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "is_primary" bool NOT NULL DEFAULT false,
  "verified_at" timestamptz(6),
  "status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "revision" int8 NOT NULL DEFAULT 1,
  "row_version" int8 NOT NULL DEFAULT 1,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "created_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "updated_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "operation_source" varchar(32) COLLATE "pg_catalog"."default" NOT NULL DEFAULT 'USER'::character varying,
  "operation_trace_id" varchar(128) COLLATE "pg_catalog"."default",
  "is_deleted" bool NOT NULL DEFAULT false,
  "deleted_at" timestamptz(6),
  "deleted_by" varchar(128) COLLATE "pg_catalog"."default",
  "delete_reason" varchar(512) COLLATE "pg_catalog"."default"
)
;
COMMENT ON COLUMN "ai_governance"."user_contact_points"."id" IS '应用生成的 UUIDv7 联系点主键；邮箱与手机号分别形成独立可审计事实。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_contact_points"."tenant_id" IS '联系点所属 Tenant 隔离键；HMAC、唯一索引、查询和授权均在该 Tenant 内执行。';
COMMENT ON COLUMN "ai_governance"."user_contact_points"."subject_id" IS '联系点所属 IdentitySubject 逻辑引用；写入时校验同 Tenant 且主体为 USER。';
COMMENT ON COLUMN "ai_governance"."user_contact_points"."contact_type" IS '联系点类型；EMAIL 为邮箱，PHONE 为手机号；类型决定规范化、掩码、验证和发送 Adapter。';
COMMENT ON COLUMN "ai_governance"."user_contact_points"."normalized_lookup_hmac" IS '规范化邮箱或手机号的 Tenant 级 HMAC；用于唯一查询，禁止由不可加密普通哈希替代。';
COMMENT ON COLUMN "ai_governance"."user_contact_points"."contact_value_ciphertext" IS '手机号或邮箱原值的信封密文；禁止明文、日志、错误响应或普通指标输出。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_contact_points"."encryption_key_ref" IS '解密联系点密文所需的 KMS/Vault Key 引用；数据库不保存解密密钥。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_contact_points"."masked_value" IS '面向用户和管理员展示的脱敏联系方式；不得用于登录匹配或唯一性判断。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_contact_points"."verification_status" IS '验证状态；UNVERIFIED 未验证，PENDING 验证中，VERIFIED 已验证，REVOKED 验证资格撤销。';
COMMENT ON COLUMN "ai_governance"."user_contact_points"."is_primary" IS '是否为该 Subject 与联系点类型的主联系方式；每种类型至多一个有效主联系点。';
COMMENT ON COLUMN "ai_governance"."user_contact_points"."verified_at" IS '最近一次成功验证时间；仅 VERIFICATION_STATUS=VERIFIED 时非空。';
COMMENT ON COLUMN "ai_governance"."user_contact_points"."status" IS '联系点运行状态；ACTIVE 可使用，SUSPENDED 暂停认证，REVOKED 永久停止新认证。';
COMMENT ON COLUMN "ai_governance"."user_contact_points"."revision" IS '联系点单调业务版本；原值、验证、主标记或状态变化后递增。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_contact_points"."row_version" IS 'GORM 乐观锁版本；防止并发验证、换绑和撤销互相覆盖。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_contact_points"."created_at" IS '联系点首次创建时间；数据库生成且后续不可改写。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_contact_points"."updated_at" IS '联系点最近一次更新的数据库时间；由触发器统一刷新。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_contact_points"."created_by" IS '创建联系点的 Subject、管理员或认证服务标识；必须可追溯。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_contact_points"."updated_by" IS '最近修改联系点的 Subject、管理员或认证服务标识；必须可追溯。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_contact_points"."operation_source" IS '最近变更来源；USER 自助、ADMIN 管理或 AUTH 验证流程，不能替代审计 actor。';
COMMENT ON COLUMN "ai_governance"."user_contact_points"."operation_trace_id" IS '最近联系点变更的 Trace ID；用于关联 Challenge、AuthenticationEvent 和 AuditEvent。';
COMMENT ON COLUMN "ai_governance"."user_contact_points"."is_deleted" IS '联系点软删除标志；删除不释放历史认证、Usage 或 Audit 的 subject_id。';
COMMENT ON COLUMN "ai_governance"."user_contact_points"."deleted_at" IS '联系点软删除时间；未删除时为空且数据库拒绝物理删除。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_contact_points"."deleted_by" IS '执行联系点软删除的主体或服务标识；未删除时为空。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_contact_points"."delete_reason" IS '联系点软删除原因；删除时必填并写入审计，未删除时为空。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON TABLE "ai_governance"."user_contact_points" IS '用户手机号与邮箱联系点事实；原值加密、规范化 HMAC 查询并独立维护验证状态。';

-- ----------------------------
-- Table structure for user_governance_projections
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."user_governance_projections";
CREATE TABLE "ai_governance"."user_governance_projections" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "subject_id" uuid NOT NULL,
  "identity_revision" int8 NOT NULL,
  "primary_organization_id" uuid,
  "active_organization_count" int4 NOT NULL DEFAULT 0,
  "active_project_count" int4 NOT NULL DEFAULT 0,
  "active_key_count" int4 NOT NULL DEFAULT 0,
  "active_allocation_count" int4 NOT NULL DEFAULT 0,
  "available_credit" int8 NOT NULL DEFAULT 0,
  "reserved_credit" int8 NOT NULL DEFAULT 0,
  "settled_credit" int8 NOT NULL DEFAULT 0,
  "current_period_tokens" int8 NOT NULL DEFAULT 0,
  "unresolved_difference_count" int8 NOT NULL DEFAULT 0,
  "effective_authorization_revision" int8 NOT NULL,
  "runtime_generation" int8 NOT NULL,
  "source_watermark" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "generation" int8 NOT NULL,
  "projection_status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp()
)
;
COMMENT ON COLUMN "ai_governance"."user_governance_projections"."id" IS '应用生成的 UUIDv7 User 治理投影主键；投影可重建但不得成为写入事实。';
COMMENT ON COLUMN "ai_governance"."user_governance_projections"."tenant_id" IS '投影所属 Tenant；User Workspace 查询必须同时限定 tenant_id 与 subject_id。';
COMMENT ON COLUMN "ai_governance"."user_governance_projections"."subject_id" IS '投影所属最小 User Subject；Tenant 内唯一且不得使用 Key 或邮箱替代。';
COMMENT ON COLUMN "ai_governance"."user_governance_projections"."identity_revision" IS '投影采用的 IdentitySubject revision；落后时标记 STALE 并禁止展示为当前授权。';
COMMENT ON COLUMN "ai_governance"."user_governance_projections"."primary_organization_id" IS '当前有效主叶子组织逻辑引用；无组织用户允许为空。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_governance_projections"."active_organization_count" IS '当前有效组织成员关系数量；单位为条，不包含结束或软删除关系。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_governance_projections"."active_project_count" IS '当前有效 Project 成员关系数量；单位为个，仅统计 ACTIVE 项目资格。';
COMMENT ON COLUMN "ai_governance"."user_governance_projections"."active_key_count" IS '当前有效 User API Key 数量；单位为把，不包含暂停、撤销和过期 Key。';
COMMENT ON COLUMN "ai_governance"."user_governance_projections"."active_allocation_count" IS '当前有效 UserAllocation 数量；单位为个，按 BillingScope 分开统计。';
COMMENT ON COLUMN "ai_governance"."user_governance_projections"."available_credit" IS '所有有效 UserAllocation 可用 AI Credit 汇总；整数单位，仅为可重建展示值。';
COMMENT ON COLUMN "ai_governance"."user_governance_projections"."reserved_credit" IS '所有活跃 Reservation 占用 AI Credit 汇总；整数单位，仅为可重建展示值。';
COMMENT ON COLUMN "ai_governance"."user_governance_projections"."settled_credit" IS '当前统计周期已结算 AI Credit 汇总；整数单位，来源于 Ledger/Usage。';
COMMENT ON COLUMN "ai_governance"."user_governance_projections"."current_period_tokens" IS '当前统计周期已结算 Token 汇总；整数单位，周期由投影构建规则固定。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_governance_projections"."unresolved_difference_count" IS '该 User 关联的未解决对账差异数量；单位为条，不能从 Tenant 汇总均摊。';
COMMENT ON COLUMN "ai_governance"."user_governance_projections"."effective_authorization_revision" IS '构建投影时的有效授权 revision；用于识别用户权限展示是否陈旧。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_governance_projections"."runtime_generation" IS '构建投影时采用的 RuntimeSnapshot generation；必须为正整数。';
COMMENT ON COLUMN "ai_governance"."user_governance_projections"."source_watermark" IS '已消费权威事件的稳定水位；用于重建、切换和判断投影延迟。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_governance_projections"."generation" IS '投影重建世代；新世代完成后原子切换，禁止旧世代覆盖。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_governance_projections"."projection_status" IS '投影状态；BUILDING 构建中，CURRENT 当前，STALE 陈旧，FAILED 构建失败。';
COMMENT ON COLUMN "ai_governance"."user_governance_projections"."created_at" IS '投影行首次创建时间；数据库生成并用于运行审计。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_governance_projections"."updated_at" IS '投影最近成功刷新时间；由触发器统一更新。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON TABLE "ai_governance"."user_governance_projections" IS 'User Workspace 可重建治理概览；按 subject_id 聚合成员、Key、额度、Token、差异和有效 revision。';

-- ----------------------------
-- Table structure for user_login_identities
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."user_login_identities";
CREATE TABLE "ai_governance"."user_login_identities" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "subject_id" uuid NOT NULL,
  "auth_method" varchar(32) COLLATE "pg_catalog"."default" NOT NULL,
  "issuer_config_id" uuid,
  "contact_point_id" uuid,
  "oauth_provider_id" uuid,
  "provider_subject_hmac" varchar(71) COLLATE "pg_catalog"."default",
  "provider_union_subject_hmac" varchar(71) COLLATE "pg_catalog"."default",
  "is_primary" bool NOT NULL DEFAULT false,
  "status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "failed_attempt_count" int4 NOT NULL DEFAULT 0,
  "locked_until" timestamptz(6),
  "last_authenticated_at" timestamptz(6),
  "identity_revision" int8 NOT NULL DEFAULT 1,
  "row_version" int8 NOT NULL DEFAULT 1,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "created_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "updated_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "operation_source" varchar(32) COLLATE "pg_catalog"."default" NOT NULL DEFAULT 'USER'::character varying,
  "operation_trace_id" varchar(128) COLLATE "pg_catalog"."default",
  "is_deleted" bool NOT NULL DEFAULT false,
  "deleted_at" timestamptz(6),
  "deleted_by" varchar(128) COLLATE "pg_catalog"."default",
  "delete_reason" varchar(512) COLLATE "pg_catalog"."default"
)
;
COMMENT ON COLUMN "ai_governance"."user_login_identities"."id" IS '应用生成的 UUIDv7 登录标识主键；每个标识只绑定一个 IdentitySubject。';
COMMENT ON COLUMN "ai_governance"."user_login_identities"."tenant_id" IS '登录标识所属 Tenant；唯一查询、限流和认证不得跨 Tenant。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_login_identities"."subject_id" IS '登录标识绑定的唯一 IdentitySubject 逻辑引用；解绑不删除历史 Subject。';
COMMENT ON COLUMN "ai_governance"."user_login_identities"."auth_method" IS '认证方式；ENTERPRISE_SSO、PASSWORD、EMAIL_OTP、SMS_OTP、WECHAT_OAUTH 或 OIDC_OAUTH。';
COMMENT ON COLUMN "ai_governance"."user_login_identities"."issuer_config_id" IS '企业 SSO Issuer 逻辑引用；仅 ENTERPRISE_SSO 必填，其他方式为空。';
COMMENT ON COLUMN "ai_governance"."user_login_identities"."contact_point_id" IS '邮箱或手机号联系点逻辑引用；密码和 OTP 方式必填且必须已验证。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_login_identities"."oauth_provider_id" IS 'OAuth Provider 配置逻辑引用；WeChat/OIDC OAuth 方式必填。';
COMMENT ON COLUMN "ai_governance"."user_login_identities"."provider_subject_hmac" IS 'Provider Subject 或 WeChat openid 的 Tenant 级 HMAC；用于唯一绑定且原值不落库。';
COMMENT ON COLUMN "ai_governance"."user_login_identities"."provider_union_subject_hmac" IS '可选 WeChat unionid 等跨应用主体 HMAC；空值表示 Provider 未提供，不单独作为用户。';
COMMENT ON COLUMN "ai_governance"."user_login_identities"."is_primary" IS '是否为 Subject 首选登录方式；不影响其他有效登录方式或业务主体 ID。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_login_identities"."status" IS '登录标识状态；PENDING 待验证，ACTIVE 可登录，LOCKED 临时锁定，REVOKED 永久撤销。';
COMMENT ON COLUMN "ai_governance"."user_login_identities"."failed_attempt_count" IS '当前锁定窗口内连续失败次数；成功登录或受控恢复后归零。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_login_identities"."locked_until" IS '登录标识锁定截止时间；仅 LOCKED 状态必填，过期后仍需状态机恢复。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_login_identities"."last_authenticated_at" IS '该登录标识最近成功认证时间；从未成功时为空。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_login_identities"."identity_revision" IS '登录标识单调版本；绑定、验证、锁定、恢复或撤销后递增。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_login_identities"."row_version" IS 'GORM 乐观锁版本；保护并发登录失败计数和状态更新。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_login_identities"."created_at" IS '登录标识首次创建时间；数据库生成且后续不可改写。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_login_identities"."updated_at" IS '登录标识最近更新时间；由触发器统一刷新。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_login_identities"."created_by" IS '创建登录标识的 Subject、管理员或认证服务标识；必须可追溯。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_login_identities"."updated_by" IS '最近修改登录标识的 Subject、管理员或认证服务标识。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_login_identities"."operation_source" IS '最近变更来源；USER、ADMIN、SYNC 或 AUTH，不替代审计主体。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_login_identities"."operation_trace_id" IS '登录标识最近变更 Trace ID；关联认证和管理审计。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_login_identities"."is_deleted" IS '登录标识软删除标志；删除前必须确保仍有可恢复认证方式。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_login_identities"."deleted_at" IS '登录标识软删除时间；未删除时为空且禁止物理删除。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_login_identities"."deleted_by" IS '执行登录标识软删除的主体或服务标识；未删除时为空。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_login_identities"."delete_reason" IS '登录标识软删除原因；删除时必填并关联 AuditEvent。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON TABLE "ai_governance"."user_login_identities" IS '绑定到唯一 IdentitySubject 的登录标识；企业 SSO、密码、验证码和 OAuth 不得各自产生重复用户。';

-- ----------------------------
-- Table structure for user_password_credentials
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."user_password_credentials";
CREATE TABLE "ai_governance"."user_password_credentials" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "subject_id" uuid NOT NULL,
  "login_identity_id" uuid NOT NULL,
  "password_hash" varchar(1024) COLLATE "pg_catalog"."default" NOT NULL,
  "algorithm" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "hash_parameters_json" jsonb NOT NULL,
  "password_revision" int8 NOT NULL DEFAULT 1,
  "must_change_password" bool NOT NULL DEFAULT false,
  "changed_at" timestamptz(6) NOT NULL,
  "expires_at" timestamptz(6),
  "status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "row_version" int8 NOT NULL DEFAULT 1,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "created_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "updated_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "operation_source" varchar(32) COLLATE "pg_catalog"."default" NOT NULL DEFAULT 'USER'::character varying,
  "operation_trace_id" varchar(128) COLLATE "pg_catalog"."default",
  "is_deleted" bool NOT NULL DEFAULT false,
  "deleted_at" timestamptz(6),
  "deleted_by" varchar(128) COLLATE "pg_catalog"."default",
  "delete_reason" varchar(512) COLLATE "pg_catalog"."default"
)
;
COMMENT ON COLUMN "ai_governance"."user_password_credentials"."id" IS '应用生成的 UUIDv7 密码凭证主键；凭证轮换保留审计而不复用旧主键。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_password_credentials"."tenant_id" IS '密码凭证所属 Tenant；密码策略、Pepper 和访问权限均按 Tenant 隔离。';
COMMENT ON COLUMN "ai_governance"."user_password_credentials"."subject_id" IS '密码凭证所属 IdentitySubject 逻辑引用；用于撤销和安全审计。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_password_credentials"."login_identity_id" IS 'PASSWORD 登录标识逻辑引用；同一有效登录标识只允许一条当前凭证。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_password_credentials"."password_hash" IS '批准算法生成的自描述密码强哈希；禁止明文、可逆密文和普通 SHA 摘要。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_password_credentials"."algorithm" IS '密码哈希算法；ARGON2ID 为当前 P0 唯一允许值，参数记录在独立 JSON。';
COMMENT ON COLUMN "ai_governance"."user_password_credentials"."hash_parameters_json" IS '哈希内存、迭代、并行度、Salt 和 Pepper 版本等不可缺失参数；不得包含密码原文。';
COMMENT ON COLUMN "ai_governance"."user_password_credentials"."password_revision" IS '密码凭证单调版本；每次重置或换密递增并使旧会话按策略失效。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_password_credentials"."must_change_password" IS '是否要求下次认证后立即换密；true 不允许直接进入普通业务会话。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_password_credentials"."changed_at" IS '当前密码版本建立时间；用于密码有效期和安全审计。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_password_credentials"."expires_at" IS '密码可选失效时间；空值表示 Tenant 策略未设置固定有效期。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_password_credentials"."status" IS '凭证状态；ACTIVE 可认证，RESET_REQUIRED 要求重置，REVOKED 已撤销，EXPIRED 已过期。';
COMMENT ON COLUMN "ai_governance"."user_password_credentials"."row_version" IS 'GORM 乐观锁版本；防止并发重置和撤销覆盖。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_password_credentials"."created_at" IS '密码凭证首次创建时间；不等同于用户注册时间。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_password_credentials"."updated_at" IS '密码凭证最近更新时间；由触发器统一刷新。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_password_credentials"."created_by" IS '建立凭证的 Subject、管理员或认证服务标识；必须可审计。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_password_credentials"."updated_by" IS '最近修改凭证的 Subject、管理员或认证服务标识。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_password_credentials"."operation_source" IS '凭证变更来源，默认 USER；重置或管理员操作必须使用明确值。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_password_credentials"."operation_trace_id" IS '凭证变更 Trace ID；关联 Challenge、AuthenticationEvent 和 AuditEvent。';
COMMENT ON COLUMN "ai_governance"."user_password_credentials"."is_deleted" IS '密码凭证软删除标志；历史认证事件不随删除消失。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_password_credentials"."deleted_at" IS '密码凭证软删除时间；未删除时为空且禁止物理删除。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_password_credentials"."deleted_by" IS '执行密码凭证软删除的主体或服务标识；未删除时为空。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_password_credentials"."delete_reason" IS '密码凭证软删除原因；删除时必填并进入安全审计。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON TABLE "ai_governance"."user_password_credentials" IS '平台本地密码凭证；仅保存批准算法的自描述强哈希，禁止明文、可逆密文或普通摘要。';

-- ----------------------------
-- Table structure for user_profiles
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."user_profiles";
CREATE TABLE "ai_governance"."user_profiles" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "subject_id" uuid NOT NULL,
  "display_name" varchar(160) COLLATE "pg_catalog"."default" NOT NULL,
  "avatar_object_ref" varchar(512) COLLATE "pg_catalog"."default",
  "locale" varchar(32) COLLATE "pg_catalog"."default" NOT NULL DEFAULT 'zh-CN'::character varying,
  "timezone" varchar(64) COLLATE "pg_catalog"."default" NOT NULL DEFAULT 'Asia/Shanghai'::character varying,
  "profile_revision" int8 NOT NULL DEFAULT 1,
  "row_version" int8 NOT NULL DEFAULT 1,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "created_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "updated_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "operation_source" varchar(32) COLLATE "pg_catalog"."default" NOT NULL DEFAULT 'USER'::character varying,
  "operation_trace_id" varchar(128) COLLATE "pg_catalog"."default",
  "is_deleted" bool NOT NULL DEFAULT false,
  "deleted_at" timestamptz(6),
  "deleted_by" varchar(128) COLLATE "pg_catalog"."default",
  "delete_reason" varchar(512) COLLATE "pg_catalog"."default"
)
;
COMMENT ON COLUMN "ai_governance"."user_profiles"."id" IS '应用生成的 UUIDv7 主键；只标识一条用户资料事实，不暴露顺序或分片语义。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_profiles"."tenant_id" IS '资料所属 Tenant 隔离键；所有读取、唯一性和写入校验必须先限定该值。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_profiles"."subject_id" IS '资料所属 identity_subjects.id 逻辑引用；同 Tenant 活跃 Subject 只允许一条未删除资料。';
COMMENT ON COLUMN "ai_governance"."user_profiles"."display_name" IS '用户自助维护的展示名称；不是实名、登录标识、权限或审计主体权威字段。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_profiles"."avatar_object_ref" IS '头像对象的受控存储引用；空值表示未配置，禁止保存任意外部可执行 URL。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_profiles"."locale" IS '用户界面语言与区域偏好；默认 zh-CN，不参与身份、权限或资金判断。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_profiles"."timezone" IS '用户展示时区 IANA 标识；默认 Asia/Shanghai，账本和审计事实时间仍统一保存 UTC。';
COMMENT ON COLUMN "ai_governance"."user_profiles"."profile_revision" IS '用户资料单调版本；每次有效资料变更递增，用于缓存失效和并发审计。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_profiles"."row_version" IS 'GORM 乐观锁版本；更新成功后递增，冲突时调用方必须重新读取而非覆盖。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_profiles"."created_at" IS '资料事实首次创建的数据库时间，精度为微秒；后续更新不得改写。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_profiles"."updated_at" IS '资料事实最近一次成功更新的数据库时间，由触发器统一刷新。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_profiles"."created_by" IS '创建资料的不可变主体或服务标识；必须能关联管理审计或注册流程。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_profiles"."updated_by" IS '最近更新资料的主体或服务标识；不得使用无法追溯的共享账号。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_profiles"."operation_source" IS '最近变更来源代码，例如 USER、ADMIN、SYNC；用于人工审计但不替代 actor。';
COMMENT ON COLUMN "ai_governance"."user_profiles"."operation_trace_id" IS '最近变更的 Trace 关联值；空值仅允许离线迁移并必须有迁移证据。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_profiles"."is_deleted" IS '资料软删除标志；true 时删除时间、删除人和删除原因必须同时存在。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_profiles"."deleted_at" IS '资料被软删除的 UTC 数据库时间；未删除时必须为空且禁止物理删除。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_profiles"."deleted_by" IS '执行资料软删除的主体或服务标识；未删除时必须为空。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_profiles"."delete_reason" IS '资料软删除业务原因；删除时必填并进入 AuditEvent，未删除时为空。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON TABLE "ai_governance"."user_profiles" IS '平台用户资料事实；与登录标识、权限、资金和凭证分离，由 Identity Center 按 subject_id 管理。';

-- ----------------------------
-- Table structure for user_sessions
-- ----------------------------
DROP TABLE IF EXISTS "ai_governance"."user_sessions";
CREATE TABLE "ai_governance"."user_sessions" (
  "id" uuid NOT NULL,
  "tenant_id" uuid NOT NULL,
  "session_id" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "subject_id" uuid NOT NULL,
  "login_identity_id" uuid NOT NULL,
  "auth_method" varchar(32) COLLATE "pg_catalog"."default" NOT NULL,
  "assurance_level" varchar(16) COLLATE "pg_catalog"."default" NOT NULL,
  "access_token_jti_hash" varchar(71) COLLATE "pg_catalog"."default" NOT NULL,
  "refresh_token_digest" varchar(512) COLLATE "pg_catalog"."default" NOT NULL,
  "device_fingerprint_hash" varchar(71) COLLATE "pg_catalog"."default",
  "source_ip" inet,
  "user_agent_hash" varchar(71) COLLATE "pg_catalog"."default",
  "issued_at" timestamptz(6) NOT NULL,
  "last_seen_at" timestamptz(6) NOT NULL,
  "expires_at" timestamptz(6) NOT NULL,
  "revoked_at" timestamptz(6),
  "revoke_reason" varchar(512) COLLATE "pg_catalog"."default",
  "session_revision" int8 NOT NULL DEFAULT 1,
  "status" varchar(24) COLLATE "pg_catalog"."default" NOT NULL,
  "row_version" int8 NOT NULL DEFAULT 1,
  "created_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "updated_at" timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  "created_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "updated_by" varchar(128) COLLATE "pg_catalog"."default" NOT NULL,
  "operation_source" varchar(32) COLLATE "pg_catalog"."default" NOT NULL DEFAULT 'AUTH'::character varying,
  "operation_trace_id" varchar(128) COLLATE "pg_catalog"."default",
  "is_deleted" bool NOT NULL DEFAULT false,
  "deleted_at" timestamptz(6),
  "deleted_by" varchar(128) COLLATE "pg_catalog"."default",
  "delete_reason" varchar(512) COLLATE "pg_catalog"."default"
)
;
COMMENT ON COLUMN "ai_governance"."user_sessions"."id" IS '应用生成的 UUIDv7 会话事实主键；不等于访问令牌、刷新令牌或外部 session cookie。';
COMMENT ON COLUMN "ai_governance"."user_sessions"."tenant_id" IS '会话所属 Tenant；令牌 audience、密钥和查询均不得跨 Tenant。';
COMMENT ON COLUMN "ai_governance"."user_sessions"."session_id" IS '对外不透明会话标识；Tenant 内唯一，用于撤销和 User Workspace 查询。';
COMMENT ON COLUMN "ai_governance"."user_sessions"."subject_id" IS '会话所属 IdentitySubject 逻辑引用；所有业务请求据此构造 SubjectContext。';
COMMENT ON COLUMN "ai_governance"."user_sessions"."login_identity_id" IS '本次会话使用的登录标识逻辑引用；用于认证方式撤销传播。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_sessions"."auth_method" IS '建立会话的认证方式；ENTERPRISE_SSO、PASSWORD、EMAIL_OTP、SMS_OTP、WECHAT_OAUTH 或 OIDC_OAUTH。';
COMMENT ON COLUMN "ai_governance"."user_sessions"."assurance_level" IS '认证强度；LOW 低，MEDIUM 中，HIGH 高；由认证链决定且请求不能提升。';
COMMENT ON COLUMN "ai_governance"."user_sessions"."access_token_jti_hash" IS '当前访问令牌 JTI 的 HMAC/哈希；用于重放与撤销查询，原 JTI 不作业务 ID。';
COMMENT ON COLUMN "ai_governance"."user_sessions"."refresh_token_digest" IS '刷新令牌带密钥摘要；原刷新令牌只向客户端返回一次且不得记录。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_sessions"."device_fingerprint_hash" IS '可选设备指纹摘要；用于风险检测，空值表示客户端未提供受支持证据。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_sessions"."source_ip" IS '会话创建来源 IP；仅用于安全审计、异常检测和受控合规查询。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_sessions"."user_agent_hash" IS 'User-Agent 脱敏哈希；原文不进入普通日志或指标。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_sessions"."issued_at" IS '会话签发时间；访问与刷新令牌有效期均以此和 Tenant 策略计算。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_sessions"."last_seen_at" IS '会话最近受信访问时间；不得高频无条件写入造成热点。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_sessions"."expires_at" IS '会话硬失效时间；必须晚于签发时间，到期后不能刷新。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_sessions"."revoked_at" IS '会话撤销时间；仅 REVOKED 状态必填，历史记录永久保留。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_sessions"."revoke_reason" IS '会话撤销原因；仅 REVOKED 状态必填并进入 AuthenticationEvent。';
COMMENT ON COLUMN "ai_governance"."user_sessions"."session_revision" IS '会话单调版本；刷新、风险收紧或撤销后递增，旧令牌 revision 失效。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_sessions"."status" IS '会话状态；ACTIVE 可使用，REVOKED 已主动撤销，EXPIRED 已到硬失效时间。';
COMMENT ON COLUMN "ai_governance"."user_sessions"."row_version" IS 'GORM 乐观锁版本；保护并发刷新、使用和撤销。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_sessions"."created_at" IS '会话事实首次创建时间；通常等于但不强制等于 issued_at。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_sessions"."updated_at" IS '会话最近更新时间；由触发器统一刷新。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_sessions"."created_by" IS '创建会话的认证服务标识；不得写匿名不可追溯共享值。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_sessions"."updated_by" IS '最近刷新或撤销会话的主体或服务标识。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_sessions"."operation_source" IS '会话操作来源，默认 AUTH；管理员撤销必须使用明确来源。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_sessions"."operation_trace_id" IS '会话最近操作 Trace ID；关联认证、授权和 AuditEvent。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_sessions"."is_deleted" IS '会话软删除标志；安全保留期内不得删除，撤销应使用状态而不是删除。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_sessions"."deleted_at" IS '会话软删除时间；未删除时为空且禁止物理删除。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_sessions"."deleted_by" IS '执行会话软删除的管理员或保留策略服务标识。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON COLUMN "ai_governance"."user_sessions"."delete_reason" IS '会话软删除原因；仅保留策略允许且必须关联审计。；Review 必须核验 Tenant、Subject、来源、空值、生命周期和审计一致性。';
COMMENT ON TABLE "ai_governance"."user_sessions" IS '用户认证会话事实；访问与刷新令牌只保存摘要，支持过期、撤销、设备和风险审计。';

-- ----------------------------
-- Indexes structure for table approval_decisions
-- ----------------------------
CREATE UNIQUE INDEX "uq_approval_decision_subject" ON "ai_governance"."approval_decisions" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "approval_request_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "approver_subject_id" "pg_catalog"."uuid_ops" ASC NULLS LAST
);

-- ----------------------------
-- Triggers structure for table approval_decisions
-- ----------------------------
CREATE TRIGGER "trg_append_only" BEFORE UPDATE ON "ai_governance"."approval_decisions"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_append_only_mutation"();
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."approval_decisions"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();

-- ----------------------------
-- Checks structure for table approval_decisions
-- ----------------------------
ALTER TABLE "ai_governance"."approval_decisions" ADD CONSTRAINT "approval_decisions_decision_check" CHECK (decision::text = ANY (ARRAY['APPROVE'::character varying, 'REJECT'::character varying]::text[]));

-- ----------------------------
-- Primary Key structure for table approval_decisions
-- ----------------------------
ALTER TABLE "ai_governance"."approval_decisions" ADD CONSTRAINT "approval_decisions_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table approval_requests
-- ----------------------------
CREATE INDEX "ix_approval_pending" ON "ai_governance"."approval_requests" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "status" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "expires_at" "pg_catalog"."timestamptz_ops" ASC NULLS LAST
);
CREATE UNIQUE INDEX "uq_approval_request_code" ON "ai_governance"."approval_requests" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "request_code" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);

-- ----------------------------
-- Triggers structure for table approval_requests
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."approval_requests"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."approval_requests"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();

-- ----------------------------
-- Checks structure for table approval_requests
-- ----------------------------
ALTER TABLE "ai_governance"."approval_requests" ADD CONSTRAINT "approval_requests_check" CHECK (expires_at > created_at);
ALTER TABLE "ai_governance"."approval_requests" ADD CONSTRAINT "approval_requests_required_decision_count_check" CHECK (required_decision_count >= 1 AND required_decision_count <= 9);
ALTER TABLE "ai_governance"."approval_requests" ADD CONSTRAINT "approval_requests_row_version_check" CHECK (row_version > 0);
ALTER TABLE "ai_governance"."approval_requests" ADD CONSTRAINT "approval_requests_scope_type_check" CHECK (scope_type::text = ANY (ARRAY['TENANT'::character varying, 'ORGANIZATION'::character varying, 'PROJECT'::character varying, 'USER'::character varying]::text[]));
ALTER TABLE "ai_governance"."approval_requests" ADD CONSTRAINT "approval_requests_status_check" CHECK (status::text = ANY (ARRAY['DRAFT'::character varying, 'PENDING'::character varying, 'APPROVED'::character varying, 'REJECTED'::character varying, 'CANCELLED'::character varying, 'EXPIRED'::character varying]::text[]));

-- ----------------------------
-- Primary Key structure for table approval_requests
-- ----------------------------
ALTER TABLE "ai_governance"."approval_requests" ADD CONSTRAINT "approval_requests_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table audit_chain_anchors
-- ----------------------------
CREATE UNIQUE INDEX "uq_audit_anchor_business_id" ON "ai_governance"."audit_chain_anchors" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "anchor_id" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);
CREATE UNIQUE INDEX "uq_audit_anchor_range" ON "ai_governance"."audit_chain_anchors" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "chain_id" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "first_sequence" "pg_catalog"."int8_ops" ASC NULLS LAST,
  "last_sequence" "pg_catalog"."int8_ops" ASC NULLS LAST
);

-- ----------------------------
-- Triggers structure for table audit_chain_anchors
-- ----------------------------
CREATE TRIGGER "trg_append_only" BEFORE UPDATE ON "ai_governance"."audit_chain_anchors"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_append_only_mutation"();
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."audit_chain_anchors"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();

-- ----------------------------
-- Checks structure for table audit_chain_anchors
-- ----------------------------
ALTER TABLE "ai_governance"."audit_chain_anchors" ADD CONSTRAINT "audit_chain_anchors_check" CHECK (last_sequence >= first_sequence);
ALTER TABLE "ai_governance"."audit_chain_anchors" ADD CONSTRAINT "audit_chain_anchors_first_sequence_check" CHECK (first_sequence > 0);
ALTER TABLE "ai_governance"."audit_chain_anchors" ADD CONSTRAINT "audit_chain_anchors_verification_status_check" CHECK (verification_status::text = ANY (ARRAY['PENDING'::character varying, 'VERIFIED'::character varying, 'FAILED'::character varying]::text[]));

-- ----------------------------
-- Primary Key structure for table audit_chain_anchors
-- ----------------------------
ALTER TABLE "ai_governance"."audit_chain_anchors" ADD CONSTRAINT "audit_chain_anchors_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table audit_events
-- ----------------------------
CREATE INDEX "ix_audit_actor_time" ON "ai_governance"."audit_events" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "actor_type" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "actor_id" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "occurred_at" "pg_catalog"."timestamptz_ops" DESC NULLS FIRST
);
CREATE INDEX "ix_audit_events_occurred_brin" ON "ai_governance"."audit_events" USING brin (
  "occurred_at" "pg_catalog"."timestamptz_minmax_ops"
);
CREATE INDEX "ix_audit_request" ON "ai_governance"."audit_events" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "request_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "occurred_at" "pg_catalog"."timestamptz_ops" ASC NULLS LAST
);
CREATE INDEX "ix_audit_resource_time" ON "ai_governance"."audit_events" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "resource_type" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "resource_id" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "occurred_at" "pg_catalog"."timestamptz_ops" DESC NULLS FIRST
);
CREATE INDEX "ix_audit_subject_time" ON "ai_governance"."audit_events" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "subject_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "occurred_at" "pg_catalog"."timestamptz_ops" DESC NULLS FIRST
);
CREATE UNIQUE INDEX "uq_audit_chain_sequence" ON "ai_governance"."audit_events" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "chain_id" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "chain_sequence" "pg_catalog"."int8_ops" ASC NULLS LAST
);
CREATE UNIQUE INDEX "uq_audit_event_business_id" ON "ai_governance"."audit_events" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "event_id" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);

-- ----------------------------
-- Triggers structure for table audit_events
-- ----------------------------
CREATE TRIGGER "trg_append_only" BEFORE UPDATE ON "ai_governance"."audit_events"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_append_only_mutation"();
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."audit_events"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();

-- ----------------------------
-- Checks structure for table audit_events
-- ----------------------------
ALTER TABLE "ai_governance"."audit_events" ADD CONSTRAINT "audit_events_authorization_decision_check" CHECK (authorization_decision::text = ANY (ARRAY['ALLOW'::character varying, 'DENY'::character varying, 'NOT_APPLICABLE'::character varying]::text[]));
ALTER TABLE "ai_governance"."audit_events" ADD CONSTRAINT "audit_events_authorization_revision_check" CHECK (authorization_revision IS NULL OR authorization_revision > 0);
ALTER TABLE "ai_governance"."audit_events" ADD CONSTRAINT "audit_events_chain_sequence_check" CHECK (chain_sequence > 0);
ALTER TABLE "ai_governance"."audit_events" ADD CONSTRAINT "audit_events_check" CHECK (scope_type IS NULL AND scope_id IS NULL OR scope_type IS NOT NULL AND scope_id IS NOT NULL);
ALTER TABLE "ai_governance"."audit_events" ADD CONSTRAINT "audit_events_external_context_json_check" CHECK (jsonb_typeof(external_context_json) = 'object'::text);
ALTER TABLE "ai_governance"."audit_events" ADD CONSTRAINT "audit_events_policy_revision_check" CHECK (policy_revision IS NULL OR policy_revision > 0);
ALTER TABLE "ai_governance"."audit_events" ADD CONSTRAINT "audit_events_scope_type_check" CHECK (scope_type::text = ANY (ARRAY['TENANT'::character varying, 'ORGANIZATION'::character varying, 'PROJECT'::character varying, 'USER'::character varying]::text[]));
ALTER TABLE "ai_governance"."audit_events" ADD CONSTRAINT "audit_events_subject_identity_revision_check" CHECK (subject_identity_revision IS NULL OR subject_identity_revision > 0);
ALTER TABLE "ai_governance"."audit_events" ADD CONSTRAINT "audit_events_actor_type_check" CHECK (actor_type::text = ANY (ARRAY['USER'::character varying, 'ADMIN'::character varying, 'SERVICE'::character varying, 'SYSTEM'::character varying]::text[]));
ALTER TABLE "ai_governance"."audit_events" ADD CONSTRAINT "ck_audit_subject_revision" CHECK (subject_id IS NULL AND subject_identity_revision IS NULL OR subject_id IS NOT NULL);

-- ----------------------------
-- Primary Key structure for table audit_events
-- ----------------------------
ALTER TABLE "ai_governance"."audit_events" ADD CONSTRAINT "audit_events_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table authentication_challenges
-- ----------------------------
CREATE INDEX "ix_auth_challenge_expiry" ON "ai_governance"."authentication_challenges" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "status" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "expires_at" "pg_catalog"."timestamptz_ops" ASC NULLS LAST
);
CREATE INDEX "ix_auth_challenge_subject" ON "ai_governance"."authentication_challenges" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "subject_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "created_at" "pg_catalog"."timestamptz_ops" DESC NULLS FIRST
);
CREATE UNIQUE INDEX "uq_auth_challenge_business_id" ON "ai_governance"."authentication_challenges" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "challenge_id" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);

-- ----------------------------
-- Triggers structure for table authentication_challenges
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."authentication_challenges"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."authentication_challenges"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();
CREATE TRIGGER "trg_validate_soft_delete" BEFORE INSERT OR UPDATE ON "ai_governance"."authentication_challenges"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_validate_soft_delete"();

-- ----------------------------
-- Checks structure for table authentication_challenges
-- ----------------------------
ALTER TABLE "ai_governance"."authentication_challenges" ADD CONSTRAINT "authentication_challenges_challenge_type_check" CHECK (challenge_type::text = ANY (ARRAY['EMAIL_VERIFY'::character varying, 'PHONE_VERIFY'::character varying, 'EMAIL_LOGIN'::character varying, 'SMS_LOGIN'::character varying, 'PASSWORD_RESET'::character varying, 'OAUTH_STATE'::character varying, 'OAUTH_BIND'::character varying]::text[]));
ALTER TABLE "ai_governance"."authentication_challenges" ADD CONSTRAINT "authentication_challenges_check" CHECK (expires_at > created_at);
ALTER TABLE "ai_governance"."authentication_challenges" ADD CONSTRAINT "authentication_challenges_check1" CHECK (attempt_count <= maximum_attempts);
ALTER TABLE "ai_governance"."authentication_challenges" ADD CONSTRAINT "authentication_challenges_check2" CHECK (status::text = 'VERIFIED'::text AND verified_at IS NOT NULL OR status::text <> 'VERIFIED'::text);
ALTER TABLE "ai_governance"."authentication_challenges" ADD CONSTRAINT "authentication_challenges_check3" CHECK (status::text = 'CONSUMED'::text AND consumed_at IS NOT NULL OR status::text <> 'CONSUMED'::text);
ALTER TABLE "ai_governance"."authentication_challenges" ADD CONSTRAINT "authentication_challenges_check4" CHECK (is_deleted AND deleted_at IS NOT NULL AND deleted_by IS NOT NULL AND delete_reason IS NOT NULL OR NOT is_deleted AND deleted_at IS NULL AND deleted_by IS NULL AND delete_reason IS NULL);
ALTER TABLE "ai_governance"."authentication_challenges" ADD CONSTRAINT "authentication_challenges_maximum_attempts_check" CHECK (maximum_attempts >= 1 AND maximum_attempts <= 20);
ALTER TABLE "ai_governance"."authentication_challenges" ADD CONSTRAINT "authentication_challenges_row_version_check" CHECK (row_version > 0);
ALTER TABLE "ai_governance"."authentication_challenges" ADD CONSTRAINT "authentication_challenges_attempt_count_check" CHECK (attempt_count >= 0);
ALTER TABLE "ai_governance"."authentication_challenges" ADD CONSTRAINT "authentication_challenges_status_check" CHECK (status::text = ANY (ARRAY['PENDING'::character varying, 'VERIFIED'::character varying, 'CONSUMED'::character varying, 'EXPIRED'::character varying, 'FAILED'::character varying, 'CANCELLED'::character varying]::text[]));

-- ----------------------------
-- Primary Key structure for table authentication_challenges
-- ----------------------------
ALTER TABLE "ai_governance"."authentication_challenges" ADD CONSTRAINT "authentication_challenges_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table authentication_events
-- ----------------------------
CREATE INDEX "ix_authentication_event_risk_time" ON "ai_governance"."authentication_events" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "risk_level" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "occurred_at" "pg_catalog"."timestamptz_ops" DESC NULLS FIRST
);
CREATE INDEX "ix_authentication_event_subject_time" ON "ai_governance"."authentication_events" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "subject_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "occurred_at" "pg_catalog"."timestamptz_ops" DESC NULLS FIRST
);
CREATE UNIQUE INDEX "uq_authentication_event_business_id" ON "ai_governance"."authentication_events" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "event_id" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);

-- ----------------------------
-- Triggers structure for table authentication_events
-- ----------------------------
CREATE TRIGGER "trg_append_only" BEFORE UPDATE ON "ai_governance"."authentication_events"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_append_only_mutation"();
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."authentication_events"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();

-- ----------------------------
-- Checks structure for table authentication_events
-- ----------------------------
ALTER TABLE "ai_governance"."authentication_events" ADD CONSTRAINT "authentication_events_auth_method_check" CHECK (auth_method::text = ANY (ARRAY['ENTERPRISE_SSO'::character varying, 'PASSWORD'::character varying, 'EMAIL_OTP'::character varying, 'SMS_OTP'::character varying, 'WECHAT_OAUTH'::character varying, 'OIDC_OAUTH'::character varying]::text[]));
ALTER TABLE "ai_governance"."authentication_events" ADD CONSTRAINT "authentication_events_event_type_check" CHECK (event_type::text = ANY (ARRAY['REGISTRATION'::character varying, 'LOGIN_SUCCEEDED'::character varying, 'LOGIN_FAILED'::character varying, 'LOGOUT'::character varying, 'TOKEN_REFRESH'::character varying, 'SESSION_REVOKED'::character varying, 'ACCOUNT_LOCKED'::character varying, 'PASSWORD_CHANGED'::character varying, 'PASSWORD_RESET'::character varying, 'CONTACT_VERIFIED'::character varying, 'OAUTH_BOUND'::character varying, 'OAUTH_UNBOUND'::character varying]::text[]));
ALTER TABLE "ai_governance"."authentication_events" ADD CONSTRAINT "authentication_events_result_check" CHECK (result::text = ANY (ARRAY['SUCCESS'::character varying, 'FAILURE'::character varying]::text[]));
ALTER TABLE "ai_governance"."authentication_events" ADD CONSTRAINT "authentication_events_risk_level_check" CHECK (risk_level::text = ANY (ARRAY['LOW'::character varying, 'MEDIUM'::character varying, 'HIGH'::character varying, 'CRITICAL'::character varying]::text[]));

-- ----------------------------
-- Primary Key structure for table authentication_events
-- ----------------------------
ALTER TABLE "ai_governance"."authentication_events" ADD CONSTRAINT "authentication_events_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table balance_projections
-- ----------------------------
CREATE UNIQUE INDEX "uq_balance_projection_account" ON "ai_governance"."balance_projections" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "account_id" "pg_catalog"."uuid_ops" ASC NULLS LAST
);

-- ----------------------------
-- Triggers structure for table balance_projections
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."balance_projections"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."balance_projections"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();

-- ----------------------------
-- Checks structure for table balance_projections
-- ----------------------------
ALTER TABLE "ai_governance"."balance_projections" ADD CONSTRAINT "balance_projections_current_balance_credit_check" CHECK (current_balance_credit >= 0);
ALTER TABLE "ai_governance"."balance_projections" ADD CONSTRAINT "balance_projections_current_reserved_credit_check" CHECK (current_reserved_credit >= 0);
ALTER TABLE "ai_governance"."balance_projections" ADD CONSTRAINT "balance_projections_generation_check" CHECK (generation > 0);
ALTER TABLE "ai_governance"."balance_projections" ADD CONSTRAINT "balance_projections_projection_status_check" CHECK (projection_status::text = ANY (ARRAY['BUILDING'::character varying, 'CURRENT'::character varying, 'STALE'::character varying, 'FAILED'::character varying]::text[]));

-- ----------------------------
-- Primary Key structure for table balance_projections
-- ----------------------------
ALTER TABLE "ai_governance"."balance_projections" ADD CONSTRAINT "balance_projections_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table closing_snapshots
-- ----------------------------
CREATE UNIQUE INDEX "uq_closing_snapshot_code" ON "ai_governance"."closing_snapshots" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "closing_code" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);
CREATE UNIQUE INDEX "uq_closing_snapshot_window" ON "ai_governance"."closing_snapshots" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "window_start" "pg_catalog"."timestamptz_ops" ASC NULLS LAST,
  "window_end" "pg_catalog"."timestamptz_ops" ASC NULLS LAST
);

-- ----------------------------
-- Triggers structure for table closing_snapshots
-- ----------------------------
CREATE TRIGGER "trg_append_only" BEFORE UPDATE ON "ai_governance"."closing_snapshots"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_append_only_mutation"();
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."closing_snapshots"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();

-- ----------------------------
-- Checks structure for table closing_snapshots
-- ----------------------------
ALTER TABLE "ai_governance"."closing_snapshots" ADD CONSTRAINT "closing_snapshots_check" CHECK (window_end > window_start);
ALTER TABLE "ai_governance"."closing_snapshots" ADD CONSTRAINT "closing_snapshots_company_consumed_credit_check" CHECK (company_consumed_credit >= 0);
ALTER TABLE "ai_governance"."closing_snapshots" ADD CONSTRAINT "closing_snapshots_difference_summary_json_check" CHECK (jsonb_typeof(difference_summary_json) = 'object'::text);
ALTER TABLE "ai_governance"."closing_snapshots" ADD CONSTRAINT "closing_snapshots_emergency_credit_summary_json_check" CHECK (jsonb_typeof(emergency_credit_summary_json) = 'object'::text);
ALTER TABLE "ai_governance"."closing_snapshots" ADD CONSTRAINT "closing_snapshots_key_summary_json_check" CHECK (jsonb_typeof(key_summary_json) = 'object'::text);
ALTER TABLE "ai_governance"."closing_snapshots" ADD CONSTRAINT "closing_snapshots_reservation_summary_json_check" CHECK (jsonb_typeof(reservation_summary_json) = 'object'::text);
ALTER TABLE "ai_governance"."closing_snapshots" ADD CONSTRAINT "closing_snapshots_scope_summary_json_check" CHECK (jsonb_typeof(scope_summary_json) = 'object'::text);
ALTER TABLE "ai_governance"."closing_snapshots" ADD CONSTRAINT "closing_snapshots_allocation_summary_json_check" CHECK (jsonb_typeof(allocation_summary_json) = 'object'::text);
ALTER TABLE "ai_governance"."closing_snapshots" ADD CONSTRAINT "closing_snapshots_unresolved_difference_count_check" CHECK (unresolved_difference_count = 0);

-- ----------------------------
-- Primary Key structure for table closing_snapshots
-- ----------------------------
ALTER TABLE "ai_governance"."closing_snapshots" ADD CONSTRAINT "closing_snapshots_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table credit_rate_versions
-- ----------------------------
CREATE INDEX "ix_credit_rate_model_time" ON "ai_governance"."credit_rate_versions" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "model_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "status" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "valid_from" "pg_catalog"."timestamptz_ops" ASC NULLS LAST,
  "valid_until" "pg_catalog"."timestamptz_ops" ASC NULLS LAST
);
CREATE UNIQUE INDEX "uq_credit_rate_version_active" ON "ai_governance"."credit_rate_versions" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "rate_code" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "revision" "pg_catalog"."int8_ops" ASC NULLS LAST
) WHERE is_deleted = false;

-- ----------------------------
-- Triggers structure for table credit_rate_versions
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."credit_rate_versions"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."credit_rate_versions"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();
CREATE TRIGGER "trg_validate_soft_delete" BEFORE INSERT OR UPDATE ON "ai_governance"."credit_rate_versions"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_validate_soft_delete"();

-- ----------------------------
-- Checks structure for table credit_rate_versions
-- ----------------------------
ALTER TABLE "ai_governance"."credit_rate_versions" ADD CONSTRAINT "credit_rate_versions_check1" CHECK (input_credit_per_million_tokens > 0 OR output_credit_per_million_tokens > 0);
ALTER TABLE "ai_governance"."credit_rate_versions" ADD CONSTRAINT "credit_rate_versions_check2" CHECK (is_deleted AND deleted_at IS NOT NULL AND deleted_by IS NOT NULL AND delete_reason IS NOT NULL OR NOT is_deleted AND deleted_at IS NULL AND deleted_by IS NULL AND delete_reason IS NULL);
ALTER TABLE "ai_governance"."credit_rate_versions" ADD CONSTRAINT "credit_rate_versions_input_credit_per_million_tokens_check" CHECK (input_credit_per_million_tokens >= 0);
ALTER TABLE "ai_governance"."credit_rate_versions" ADD CONSTRAINT "credit_rate_versions_minimum_charge_credit_check" CHECK (minimum_charge_credit >= 0);
ALTER TABLE "ai_governance"."credit_rate_versions" ADD CONSTRAINT "credit_rate_versions_output_credit_per_million_tokens_check" CHECK (output_credit_per_million_tokens >= 0);
ALTER TABLE "ai_governance"."credit_rate_versions" ADD CONSTRAINT "credit_rate_versions_revision_check" CHECK (revision > 0);
ALTER TABLE "ai_governance"."credit_rate_versions" ADD CONSTRAINT "credit_rate_versions_rounding_mode_check" CHECK (rounding_mode::text = ANY (ARRAY['CEILING'::character varying, 'FLOOR'::character varying, 'HALF_UP'::character varying]::text[]));
ALTER TABLE "ai_governance"."credit_rate_versions" ADD CONSTRAINT "credit_rate_versions_row_version_check" CHECK (row_version > 0);
ALTER TABLE "ai_governance"."credit_rate_versions" ADD CONSTRAINT "credit_rate_versions_check" CHECK (valid_until IS NULL OR valid_until > valid_from);
ALTER TABLE "ai_governance"."credit_rate_versions" ADD CONSTRAINT "credit_rate_versions_status_check" CHECK (status::text = ANY (ARRAY['DRAFT'::character varying, 'APPROVED'::character varying, 'ACTIVE'::character varying, 'SUPERSEDED'::character varying, 'REVOKED'::character varying]::text[]));

-- ----------------------------
-- Primary Key structure for table credit_rate_versions
-- ----------------------------
ALTER TABLE "ai_governance"."credit_rate_versions" ADD CONSTRAINT "credit_rate_versions_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table data_integrity_findings
-- ----------------------------
CREATE INDEX "ix_integrity_finding_status" ON "ai_governance"."data_integrity_findings" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "status" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "severity" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "detected_at" "pg_catalog"."timestamptz_ops" ASC NULLS LAST
);
CREATE UNIQUE INDEX "uq_integrity_finding_code" ON "ai_governance"."data_integrity_findings" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "finding_code" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);

-- ----------------------------
-- Triggers structure for table data_integrity_findings
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."data_integrity_findings"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."data_integrity_findings"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();

-- ----------------------------
-- Checks structure for table data_integrity_findings
-- ----------------------------
ALTER TABLE "ai_governance"."data_integrity_findings" ADD CONSTRAINT "data_integrity_findings_check" CHECK (status::text = 'RESOLVED'::text AND resolved_at IS NOT NULL AND resolution_note IS NOT NULL OR status::text <> 'RESOLVED'::text);
ALTER TABLE "ai_governance"."data_integrity_findings" ADD CONSTRAINT "data_integrity_findings_evidence_json_check" CHECK (jsonb_typeof(evidence_json) = 'object'::text);
ALTER TABLE "ai_governance"."data_integrity_findings" ADD CONSTRAINT "data_integrity_findings_row_version_check" CHECK (row_version > 0);
ALTER TABLE "ai_governance"."data_integrity_findings" ADD CONSTRAINT "data_integrity_findings_severity_check" CHECK (severity::text = ANY (ARRAY['LOW'::character varying, 'MEDIUM'::character varying, 'HIGH'::character varying, 'CRITICAL'::character varying]::text[]));
ALTER TABLE "ai_governance"."data_integrity_findings" ADD CONSTRAINT "data_integrity_findings_status_check" CHECK (status::text = ANY (ARRAY['OPEN'::character varying, 'INVESTIGATING'::character varying, 'RESOLVED'::character varying, 'ACCEPTED_RISK'::character varying]::text[]));

-- ----------------------------
-- Primary Key structure for table data_integrity_findings
-- ----------------------------
ALTER TABLE "ai_governance"."data_integrity_findings" ADD CONSTRAINT "data_integrity_findings_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table emergency_credit_grant_keys
-- ----------------------------
CREATE INDEX "ix_emergency_key_active" ON "ai_governance"."emergency_credit_grant_keys" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "key_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "status" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "valid_until" "pg_catalog"."timestamptz_ops" ASC NULLS LAST
);
CREATE UNIQUE INDEX "uq_emergency_grant_key" ON "ai_governance"."emergency_credit_grant_keys" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "emergency_credit_grant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "key_id" "pg_catalog"."uuid_ops" ASC NULLS LAST
);

-- ----------------------------
-- Triggers structure for table emergency_credit_grant_keys
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."emergency_credit_grant_keys"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."emergency_credit_grant_keys"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();

-- ----------------------------
-- Checks structure for table emergency_credit_grant_keys
-- ----------------------------
ALTER TABLE "ai_governance"."emergency_credit_grant_keys" ADD CONSTRAINT "emergency_credit_grant_keys_check" CHECK (valid_until > valid_from);
ALTER TABLE "ai_governance"."emergency_credit_grant_keys" ADD CONSTRAINT "emergency_credit_grant_keys_key_revision_check" CHECK (key_revision > 0);
ALTER TABLE "ai_governance"."emergency_credit_grant_keys" ADD CONSTRAINT "emergency_credit_grant_keys_row_version_check" CHECK (row_version > 0);
ALTER TABLE "ai_governance"."emergency_credit_grant_keys" ADD CONSTRAINT "emergency_credit_grant_keys_status_check" CHECK (status::text = ANY (ARRAY['ACTIVE'::character varying, 'REVOKED'::character varying, 'EXPIRED'::character varying]::text[]));

-- ----------------------------
-- Primary Key structure for table emergency_credit_grant_keys
-- ----------------------------
ALTER TABLE "ai_governance"."emergency_credit_grant_keys" ADD CONSTRAINT "emergency_credit_grant_keys_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table emergency_credit_grants
-- ----------------------------
CREATE INDEX "ix_emergency_grant_allocation" ON "ai_governance"."emergency_credit_grants" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "user_allocation_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "status" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "valid_until" "pg_catalog"."timestamptz_ops" ASC NULLS LAST
);

-- ----------------------------
-- Triggers structure for table emergency_credit_grants
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."emergency_credit_grants"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."emergency_credit_grants"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();

-- ----------------------------
-- Checks structure for table emergency_credit_grants
-- ----------------------------
ALTER TABLE "ai_governance"."emergency_credit_grants" ADD CONSTRAINT "emergency_credit_grants_check" CHECK (valid_until > valid_from);
ALTER TABLE "ai_governance"."emergency_credit_grants" ADD CONSTRAINT "emergency_credit_grants_check1" CHECK (created_by_subject_id <> approved_by_subject_id);
ALTER TABLE "ai_governance"."emergency_credit_grants" ADD CONSTRAINT "emergency_credit_grants_billing_scope_type_check" CHECK (billing_scope_type::text = ANY (ARRAY['ORGANIZATION'::character varying, 'PROJECT'::character varying]::text[]));
ALTER TABLE "ai_governance"."emergency_credit_grants" ADD CONSTRAINT "emergency_credit_grants_row_version_check" CHECK (row_version > 0);
ALTER TABLE "ai_governance"."emergency_credit_grants" ADD CONSTRAINT "emergency_credit_grants_status_check" CHECK (status::text = ANY (ARRAY['PENDING'::character varying, 'ACTIVE'::character varying, 'EXHAUSTED'::character varying, 'EXPIRED'::character varying, 'COVERED'::character varying, 'REVOKED'::character varying]::text[]));
ALTER TABLE "ai_governance"."emergency_credit_grants" ADD CONSTRAINT "emergency_credit_grants_max_credit_amount_check" CHECK (max_credit_amount > 0);

-- ----------------------------
-- Primary Key structure for table emergency_credit_grants
-- ----------------------------
ALTER TABLE "ai_governance"."emergency_credit_grants" ADD CONSTRAINT "emergency_credit_grants_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table funding_accounts
-- ----------------------------
CREATE UNIQUE INDEX "uq_funding_account_code_active" ON "ai_governance"."funding_accounts" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "account_code" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
) WHERE is_deleted = false;
CREATE UNIQUE INDEX "uq_funding_account_owner_active" ON "ai_governance"."funding_accounts" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "account_type" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "owner_type" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "owner_id" "pg_catalog"."uuid_ops" ASC NULLS LAST
) WHERE is_deleted = false AND status::text <> 'CLOSED'::text;

-- ----------------------------
-- Triggers structure for table funding_accounts
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."funding_accounts"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."funding_accounts"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();
CREATE TRIGGER "trg_validate_soft_delete" BEFORE INSERT OR UPDATE ON "ai_governance"."funding_accounts"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_validate_soft_delete"();

-- ----------------------------
-- Checks structure for table funding_accounts
-- ----------------------------
ALTER TABLE "ai_governance"."funding_accounts" ADD CONSTRAINT "funding_accounts_billing_scope_type_check" CHECK (billing_scope_type::text = ANY (ARRAY['ORGANIZATION'::character varying, 'PROJECT'::character varying]::text[]));
ALTER TABLE "ai_governance"."funding_accounts" ADD CONSTRAINT "funding_accounts_check" CHECK (billing_scope_type IS NULL AND billing_scope_id IS NULL OR billing_scope_type IS NOT NULL AND billing_scope_id IS NOT NULL);
ALTER TABLE "ai_governance"."funding_accounts" ADD CONSTRAINT "funding_accounts_check1" CHECK (is_deleted AND deleted_at IS NOT NULL AND deleted_by IS NOT NULL AND delete_reason IS NOT NULL OR NOT is_deleted AND deleted_at IS NULL AND deleted_by IS NULL AND delete_reason IS NULL);
ALTER TABLE "ai_governance"."funding_accounts" ADD CONSTRAINT "funding_accounts_normal_balance_check" CHECK (normal_balance::text = ANY (ARRAY['DEBIT'::character varying, 'CREDIT'::character varying]::text[]));
ALTER TABLE "ai_governance"."funding_accounts" ADD CONSTRAINT "funding_accounts_owner_type_check" CHECK (owner_type::text = ANY (ARRAY['TENANT'::character varying, 'ORGANIZATION'::character varying, 'PROJECT'::character varying, 'USER_ALLOCATION'::character varying, 'PLATFORM'::character varying, 'EMERGENCY_FACILITY'::character varying]::text[]));
ALTER TABLE "ai_governance"."funding_accounts" ADD CONSTRAINT "funding_accounts_revision_check" CHECK (revision > 0);
ALTER TABLE "ai_governance"."funding_accounts" ADD CONSTRAINT "funding_accounts_row_version_check" CHECK (row_version > 0);
ALTER TABLE "ai_governance"."funding_accounts" ADD CONSTRAINT "funding_accounts_account_type_check" CHECK (account_type::text = ANY (ARRAY['COMPANY_AVAILABLE'::character varying, 'SCOPE_AVAILABLE'::character varying, 'USER_AVAILABLE'::character varying, 'USER_RESERVED'::character varying, 'PLATFORM_CONSUMED'::character varying, 'EMERGENCY_FACILITY_AVAILABLE'::character varying]::text[]));
ALTER TABLE "ai_governance"."funding_accounts" ADD CONSTRAINT "funding_accounts_status_check" CHECK (status::text = ANY (ARRAY['DRAFT'::character varying, 'ACTIVE'::character varying, 'FROZEN'::character varying, 'CLOSED'::character varying]::text[]));

-- ----------------------------
-- Primary Key structure for table funding_accounts
-- ----------------------------
ALTER TABLE "ai_governance"."funding_accounts" ADD CONSTRAINT "funding_accounts_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table identity_issuer_configs
-- ----------------------------
CREATE UNIQUE INDEX "uq_identity_issuer_active" ON "ai_governance"."identity_issuer_configs" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "issuer_uri" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "audience" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
) WHERE is_deleted = false;

-- ----------------------------
-- Triggers structure for table identity_issuer_configs
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."identity_issuer_configs"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."identity_issuer_configs"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();
CREATE TRIGGER "trg_validate_soft_delete" BEFORE INSERT OR UPDATE ON "ai_governance"."identity_issuer_configs"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_validate_soft_delete"();

-- ----------------------------
-- Checks structure for table identity_issuer_configs
-- ----------------------------
ALTER TABLE "ai_governance"."identity_issuer_configs" ADD CONSTRAINT "identity_issuer_configs_clock_skew_seconds_check" CHECK (clock_skew_seconds >= 0 AND clock_skew_seconds <= 300);
ALTER TABLE "ai_governance"."identity_issuer_configs" ADD CONSTRAINT "identity_issuer_configs_issuer_type_check" CHECK (issuer_type::text = ANY (ARRAY['ENTERPRISE_OIDC'::character varying, 'PLATFORM_LOCAL'::character varying]::text[]));
ALTER TABLE "ai_governance"."identity_issuer_configs" ADD CONSTRAINT "identity_issuer_configs_check" CHECK (is_deleted AND deleted_at IS NOT NULL AND deleted_by IS NOT NULL AND delete_reason IS NOT NULL OR NOT is_deleted AND deleted_at IS NULL AND deleted_by IS NULL AND delete_reason IS NULL);
ALTER TABLE "ai_governance"."identity_issuer_configs" ADD CONSTRAINT "identity_issuer_configs_revision_check" CHECK (revision > 0);
ALTER TABLE "ai_governance"."identity_issuer_configs" ADD CONSTRAINT "identity_issuer_configs_row_version_check" CHECK (row_version > 0);
ALTER TABLE "ai_governance"."identity_issuer_configs" ADD CONSTRAINT "identity_issuer_configs_status_check" CHECK (status::text = ANY (ARRAY['DRAFT'::character varying, 'ACTIVE'::character varying, 'SUSPENDED'::character varying, 'RETIRED'::character varying]::text[]));

-- ----------------------------
-- Primary Key structure for table identity_issuer_configs
-- ----------------------------
ALTER TABLE "ai_governance"."identity_issuer_configs" ADD CONSTRAINT "identity_issuer_configs_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table identity_subjects
-- ----------------------------
CREATE INDEX "ix_identity_subject_status" ON "ai_governance"."identity_subjects" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "status" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "identity_revision" "pg_catalog"."int8_ops" ASC NULLS LAST
);
CREATE UNIQUE INDEX "uq_identity_subject_external_active" ON "ai_governance"."identity_subjects" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "issuer_config_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "external_subject" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
) WHERE is_deleted = false;

-- ----------------------------
-- Triggers structure for table identity_subjects
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."identity_subjects"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."identity_subjects"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();
CREATE TRIGGER "trg_validate_soft_delete" BEFORE INSERT OR UPDATE ON "ai_governance"."identity_subjects"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_validate_soft_delete"();

-- ----------------------------
-- Checks structure for table identity_subjects
-- ----------------------------
ALTER TABLE "ai_governance"."identity_subjects" ADD CONSTRAINT "identity_subjects_identity_origin_check" CHECK (identity_origin::text = ANY (ARRAY['FEDERATED'::character varying, 'LOCAL'::character varying, 'OAUTH'::character varying]::text[]));
ALTER TABLE "ai_governance"."identity_subjects" ADD CONSTRAINT "identity_subjects_identity_revision_check" CHECK (identity_revision > 0);
ALTER TABLE "ai_governance"."identity_subjects" ADD CONSTRAINT "identity_subjects_revision_check" CHECK (revision > 0);
ALTER TABLE "ai_governance"."identity_subjects" ADD CONSTRAINT "identity_subjects_row_version_check" CHECK (row_version > 0);
ALTER TABLE "ai_governance"."identity_subjects" ADD CONSTRAINT "identity_subjects_status_check" CHECK (status::text = ANY (ARRAY['PENDING'::character varying, 'ACTIVE'::character varying, 'DISABLED'::character varying, 'LOCKED'::character varying, 'RETIRED'::character varying]::text[]));
ALTER TABLE "ai_governance"."identity_subjects" ADD CONSTRAINT "identity_subjects_subject_type_check" CHECK (subject_type::text = ANY (ARRAY['USER'::character varying, 'SERVICE'::character varying]::text[]));
ALTER TABLE "ai_governance"."identity_subjects" ADD CONSTRAINT "identity_subjects_check" CHECK (is_deleted AND deleted_at IS NOT NULL AND deleted_by IS NOT NULL AND delete_reason IS NOT NULL OR NOT is_deleted AND deleted_at IS NULL AND deleted_by IS NULL AND delete_reason IS NULL);
ALTER TABLE "ai_governance"."identity_subjects" ADD CONSTRAINT "identity_subjects_user_type_check" CHECK (user_type::text = ANY (ARRAY['INTERNAL'::character varying, 'EXTERNAL'::character varying, 'INTERNAL_ONLY'::character varying, 'SERVICE'::character varying]::text[]));

-- ----------------------------
-- Primary Key structure for table identity_subjects
-- ----------------------------
ALTER TABLE "ai_governance"."identity_subjects" ADD CONSTRAINT "identity_subjects_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table inbox_receipts
-- ----------------------------
CREATE UNIQUE INDEX "uq_inbox_receipt_event" ON "ai_governance"."inbox_receipts" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "consumer_code" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "event_id" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);

-- ----------------------------
-- Triggers structure for table inbox_receipts
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."inbox_receipts"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."inbox_receipts"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();

-- ----------------------------
-- Checks structure for table inbox_receipts
-- ----------------------------
ALTER TABLE "ai_governance"."inbox_receipts" ADD CONSTRAINT "inbox_receipts_attempt_count_check" CHECK (attempt_count >= 0);
ALTER TABLE "ai_governance"."inbox_receipts" ADD CONSTRAINT "inbox_receipts_row_version_check" CHECK (row_version > 0);
ALTER TABLE "ai_governance"."inbox_receipts" ADD CONSTRAINT "inbox_receipts_status_check" CHECK (status::text = ANY (ARRAY['RECEIVED'::character varying, 'PROCESSING'::character varying, 'PROCESSED'::character varying, 'FAILED'::character varying, 'DEAD_LETTER'::character varying]::text[]));

-- ----------------------------
-- Primary Key structure for table inbox_receipts
-- ----------------------------
ALTER TABLE "ai_governance"."inbox_receipts" ADD CONSTRAINT "inbox_receipts_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table key_limit_policies
-- ----------------------------
CREATE UNIQUE INDEX "uq_key_limit_policy_active" ON "ai_governance"."key_limit_policies" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "policy_code" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
) WHERE is_deleted = false;

-- ----------------------------
-- Triggers structure for table key_limit_policies
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."key_limit_policies"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."key_limit_policies"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();
CREATE TRIGGER "trg_validate_soft_delete" BEFORE INSERT OR UPDATE ON "ai_governance"."key_limit_policies"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_validate_soft_delete"();

-- ----------------------------
-- Checks structure for table key_limit_policies
-- ----------------------------
ALTER TABLE "ai_governance"."key_limit_policies" ADD CONSTRAINT "key_limit_policies_check1" CHECK (is_deleted AND deleted_at IS NOT NULL AND deleted_by IS NOT NULL AND delete_reason IS NOT NULL OR NOT is_deleted AND deleted_at IS NULL AND deleted_by IS NULL AND delete_reason IS NULL);
ALTER TABLE "ai_governance"."key_limit_policies" ADD CONSTRAINT "key_limit_policies_daily_credit_limit_check" CHECK (daily_credit_limit IS NULL OR daily_credit_limit > 0);
ALTER TABLE "ai_governance"."key_limit_policies" ADD CONSTRAINT "key_limit_policies_daily_token_limit_check" CHECK (daily_token_limit IS NULL OR daily_token_limit > 0);
ALTER TABLE "ai_governance"."key_limit_policies" ADD CONSTRAINT "key_limit_policies_max_concurrency_check" CHECK (max_concurrency > 0);
ALTER TABLE "ai_governance"."key_limit_policies" ADD CONSTRAINT "key_limit_policies_monthly_credit_limit_check" CHECK (monthly_credit_limit IS NULL OR monthly_credit_limit > 0);
ALTER TABLE "ai_governance"."key_limit_policies" ADD CONSTRAINT "key_limit_policies_monthly_token_limit_check" CHECK (monthly_token_limit IS NULL OR monthly_token_limit > 0);
ALTER TABLE "ai_governance"."key_limit_policies" ADD CONSTRAINT "key_limit_policies_per_request_credit_limit_check" CHECK (per_request_credit_limit IS NULL OR per_request_credit_limit > 0);
ALTER TABLE "ai_governance"."key_limit_policies" ADD CONSTRAINT "key_limit_policies_per_request_token_limit_check" CHECK (per_request_token_limit IS NULL OR per_request_token_limit > 0);
ALTER TABLE "ai_governance"."key_limit_policies" ADD CONSTRAINT "key_limit_policies_revision_check" CHECK (revision > 0);
ALTER TABLE "ai_governance"."key_limit_policies" ADD CONSTRAINT "key_limit_policies_row_version_check" CHECK (row_version > 0);
ALTER TABLE "ai_governance"."key_limit_policies" ADD CONSTRAINT "key_limit_policies_rps_limit_check" CHECK (rps_limit > 0);
ALTER TABLE "ai_governance"."key_limit_policies" ADD CONSTRAINT "key_limit_policies_check" CHECK (valid_until IS NULL OR valid_until > valid_from);
ALTER TABLE "ai_governance"."key_limit_policies" ADD CONSTRAINT "key_limit_policies_status_check" CHECK (status::text = ANY (ARRAY['DRAFT'::character varying, 'ACTIVE'::character varying, 'SUSPENDED'::character varying, 'RETIRED'::character varying]::text[]));

-- ----------------------------
-- Primary Key structure for table key_limit_policies
-- ----------------------------
ALTER TABLE "ai_governance"."key_limit_policies" ADD CONSTRAINT "key_limit_policies_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table key_usage_counter_projections
-- ----------------------------
CREATE INDEX "ix_key_usage_subject_period" ON "ai_governance"."key_usage_counter_projections" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "subject_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "period_type" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "period_start" "pg_catalog"."timestamptz_ops" ASC NULLS LAST
);
CREATE UNIQUE INDEX "uq_key_usage_counter_period" ON "ai_governance"."key_usage_counter_projections" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "key_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "period_type" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "period_start" "pg_catalog"."timestamptz_ops" ASC NULLS LAST
);

-- ----------------------------
-- Triggers structure for table key_usage_counter_projections
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."key_usage_counter_projections"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."key_usage_counter_projections"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();

-- ----------------------------
-- Checks structure for table key_usage_counter_projections
-- ----------------------------
ALTER TABLE "ai_governance"."key_usage_counter_projections" ADD CONSTRAINT "key_usage_counter_projections_billing_scope_type_check" CHECK (billing_scope_type::text = ANY (ARRAY['ORGANIZATION'::character varying, 'PROJECT'::character varying]::text[]));
ALTER TABLE "ai_governance"."key_usage_counter_projections" ADD CONSTRAINT "key_usage_counter_projections_check" CHECK (period_end > period_start);
ALTER TABLE "ai_governance"."key_usage_counter_projections" ADD CONSTRAINT "key_usage_counter_projections_current_concurrency_check" CHECK (current_concurrency >= 0);
ALTER TABLE "ai_governance"."key_usage_counter_projections" ADD CONSTRAINT "key_usage_counter_projections_generation_check" CHECK (generation > 0);
ALTER TABLE "ai_governance"."key_usage_counter_projections" ADD CONSTRAINT "key_usage_counter_projections_period_type_check" CHECK (period_type::text = ANY (ARRAY['DAY'::character varying, 'MONTH'::character varying]::text[]));
ALTER TABLE "ai_governance"."key_usage_counter_projections" ADD CONSTRAINT "key_usage_counter_projections_projection_status_check" CHECK (projection_status::text = ANY (ARRAY['BUILDING'::character varying, 'CURRENT'::character varying, 'STALE'::character varying, 'FAILED'::character varying]::text[]));
ALTER TABLE "ai_governance"."key_usage_counter_projections" ADD CONSTRAINT "key_usage_counter_projections_reserved_credit_check" CHECK (reserved_credit >= 0);
ALTER TABLE "ai_governance"."key_usage_counter_projections" ADD CONSTRAINT "key_usage_counter_projections_reserved_token_ceiling_check" CHECK (reserved_token_ceiling >= 0);
ALTER TABLE "ai_governance"."key_usage_counter_projections" ADD CONSTRAINT "key_usage_counter_projections_settled_credit_check" CHECK (settled_credit >= 0);
ALTER TABLE "ai_governance"."key_usage_counter_projections" ADD CONSTRAINT "ck_key_usage_scope_pair" CHECK (billing_scope_id IS NOT NULL);
ALTER TABLE "ai_governance"."key_usage_counter_projections" ADD CONSTRAINT "key_usage_counter_projections_settled_tokens_check" CHECK (settled_tokens >= 0);

-- ----------------------------
-- Primary Key structure for table key_usage_counter_projections
-- ----------------------------
ALTER TABLE "ai_governance"."key_usage_counter_projections" ADD CONSTRAINT "key_usage_counter_projections_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table ledger_legs
-- ----------------------------
CREATE INDEX "ix_ledger_leg_account_time" ON "ai_governance"."ledger_legs" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "account_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "posted_at" "pg_catalog"."timestamptz_ops" ASC NULLS LAST,
  "id" "pg_catalog"."uuid_ops" ASC NULLS LAST
);
CREATE INDEX "ix_ledger_leg_allocation_time" ON "ai_governance"."ledger_legs" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "user_allocation_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "posted_at" "pg_catalog"."timestamptz_ops" ASC NULLS LAST,
  "id" "pg_catalog"."uuid_ops" ASC NULLS LAST
);
CREATE INDEX "ix_ledger_leg_key_time" ON "ai_governance"."ledger_legs" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "key_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "posted_at" "pg_catalog"."timestamptz_ops" ASC NULLS LAST,
  "id" "pg_catalog"."uuid_ops" ASC NULLS LAST
);
CREATE INDEX "ix_ledger_leg_request" ON "ai_governance"."ledger_legs" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "request_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "posted_at" "pg_catalog"."timestamptz_ops" ASC NULLS LAST
);
CREATE INDEX "ix_ledger_leg_subject_time" ON "ai_governance"."ledger_legs" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "subject_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "posted_at" "pg_catalog"."timestamptz_ops" DESC NULLS FIRST,
  "id" "pg_catalog"."uuid_ops" ASC NULLS LAST
);
CREATE INDEX "ix_ledger_legs_posted_brin" ON "ai_governance"."ledger_legs" USING brin (
  "posted_at" "pg_catalog"."timestamptz_minmax_ops"
);
CREATE UNIQUE INDEX "uq_ledger_leg_no" ON "ai_governance"."ledger_legs" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "ledger_transaction_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "leg_no" "pg_catalog"."int2_ops" ASC NULLS LAST
);

-- ----------------------------
-- Triggers structure for table ledger_legs
-- ----------------------------
CREATE TRIGGER "trg_append_only" BEFORE UPDATE ON "ai_governance"."ledger_legs"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_append_only_mutation"();
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."ledger_legs"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE CONSTRAINT TRIGGER "trg_validate_ledger_leg_parent" AFTER INSERT ON "ai_governance"."ledger_legs"
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_validate_ledger_leg_parent"();

-- ----------------------------
-- Checks structure for table ledger_legs
-- ----------------------------
ALTER TABLE "ai_governance"."ledger_legs" ADD CONSTRAINT "ledger_legs_account_type_check" CHECK (account_type::text = ANY (ARRAY['COMPANY_AVAILABLE'::character varying, 'SCOPE_AVAILABLE'::character varying, 'USER_AVAILABLE'::character varying, 'USER_RESERVED'::character varying, 'PLATFORM_CONSUMED'::character varying, 'EMERGENCY_FACILITY_AVAILABLE'::character varying]::text[]));
ALTER TABLE "ai_governance"."ledger_legs" ADD CONSTRAINT "ledger_legs_actor_type_check" CHECK (actor_type::text = ANY (ARRAY['USER'::character varying, 'ADMIN'::character varying, 'SERVICE'::character varying, 'SYSTEM'::character varying]::text[]));
ALTER TABLE "ai_governance"."ledger_legs" ADD CONSTRAINT "ledger_legs_billing_scope_type_check" CHECK (billing_scope_type::text = ANY (ARRAY['ORGANIZATION'::character varying, 'PROJECT'::character varying]::text[]));
ALTER TABLE "ai_governance"."ledger_legs" ADD CONSTRAINT "ledger_legs_check" CHECK (abs(credit_delta) = credit_amount);
ALTER TABLE "ai_governance"."ledger_legs" ADD CONSTRAINT "ledger_legs_check1" CHECK (debit_credit::text = 'DEBIT'::text AND credit_delta > 0 OR debit_credit::text = 'CREDIT'::text AND credit_delta < 0);
ALTER TABLE "ai_governance"."ledger_legs" ADD CONSTRAINT "ledger_legs_check2" CHECK (billing_scope_type IS NULL AND billing_scope_id IS NULL OR billing_scope_type IS NOT NULL AND billing_scope_id IS NOT NULL);
ALTER TABLE "ai_governance"."ledger_legs" ADD CONSTRAINT "ledger_legs_check3" CHECK (total_tokens IS NULL OR input_tokens IS NULL OR output_tokens IS NULL OR total_tokens = (input_tokens + output_tokens));
ALTER TABLE "ai_governance"."ledger_legs" ADD CONSTRAINT "ledger_legs_check4" CHECK (entry_type::text = 'REVERSAL'::text AND reversal_of_leg_id IS NOT NULL OR entry_type::text <> 'REVERSAL'::text);
ALTER TABLE "ai_governance"."ledger_legs" ADD CONSTRAINT "ledger_legs_check5" CHECK ((entry_type::text <> ALL (ARRAY['RESERVE'::character varying, 'SUPPLEMENTAL_RESERVE'::character varying, 'SETTLE'::character varying, 'RELEASE'::character varying, 'ADJUST'::character varying, 'REVERSAL'::character varying, 'EMERGENCY_DRAW'::character varying]::text[])) OR company_account_id IS NOT NULL AND scope_account_id IS NOT NULL AND user_allocation_id IS NOT NULL AND key_id IS NOT NULL AND key_revision IS NOT NULL AND billing_scope_type IS NOT NULL AND request_id IS NOT NULL AND reservation_id IS NOT NULL AND credit_rate_revision IS NOT NULL AND input_tokens IS NOT NULL AND output_tokens IS NOT NULL AND total_tokens IS NOT NULL);
ALTER TABLE "ai_governance"."ledger_legs" ADD CONSTRAINT "ledger_legs_credit_amount_check" CHECK (credit_amount > 0);
ALTER TABLE "ai_governance"."ledger_legs" ADD CONSTRAINT "ledger_legs_credit_delta_check" CHECK (credit_delta <> 0);
ALTER TABLE "ai_governance"."ledger_legs" ADD CONSTRAINT "ledger_legs_credit_rate_revision_check" CHECK (credit_rate_revision IS NULL OR credit_rate_revision > 0);
ALTER TABLE "ai_governance"."ledger_legs" ADD CONSTRAINT "ledger_legs_debit_credit_check" CHECK (debit_credit::text = ANY (ARRAY['DEBIT'::character varying, 'CREDIT'::character varying]::text[]));
ALTER TABLE "ai_governance"."ledger_legs" ADD CONSTRAINT "ledger_legs_entry_type_check" CHECK (entry_type::text = ANY (ARRAY['ISSUE'::character varying, 'ALLOCATE_TO_SCOPE'::character varying, 'ALLOCATE_TO_USER'::character varying, 'RECLAIM_FROM_USER'::character varying, 'RESERVE'::character varying, 'SUPPLEMENTAL_RESERVE'::character varying, 'SETTLE'::character varying, 'RELEASE'::character varying, 'ADJUST'::character varying, 'REVERSAL'::character varying, 'EMERGENCY_FACILITY_GRANT'::character varying, 'EMERGENCY_DRAW'::character varying, 'EMERGENCY_COVER'::character varying, 'EMERGENCY_CLOSE'::character varying]::text[]));
ALTER TABLE "ai_governance"."ledger_legs" ADD CONSTRAINT "ledger_legs_input_tokens_check" CHECK (input_tokens IS NULL OR input_tokens >= 0);
ALTER TABLE "ai_governance"."ledger_legs" ADD CONSTRAINT "ledger_legs_key_revision_check" CHECK (key_revision IS NULL OR key_revision > 0);
ALTER TABLE "ai_governance"."ledger_legs" ADD CONSTRAINT "ledger_legs_leg_no_check" CHECK (leg_no >= 1 AND leg_no <= 32);
ALTER TABLE "ai_governance"."ledger_legs" ADD CONSTRAINT "ledger_legs_output_tokens_check" CHECK (output_tokens IS NULL OR output_tokens >= 0);
ALTER TABLE "ai_governance"."ledger_legs" ADD CONSTRAINT "ledger_legs_total_tokens_check" CHECK (total_tokens IS NULL OR total_tokens >= 0);
ALTER TABLE "ai_governance"."ledger_legs" ADD CONSTRAINT "ck_ledger_leg_user_dimension" CHECK (user_allocation_id IS NULL AND subject_id IS NULL OR user_allocation_id IS NOT NULL AND subject_id IS NOT NULL);

-- ----------------------------
-- Primary Key structure for table ledger_legs
-- ----------------------------
ALTER TABLE "ai_governance"."ledger_legs" ADD CONSTRAINT "ledger_legs_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table ledger_transactions
-- ----------------------------
CREATE INDEX "ix_ledger_transaction_request" ON "ai_governance"."ledger_transactions" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "request_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "posted_at" "pg_catalog"."timestamptz_ops" ASC NULLS LAST
);
CREATE INDEX "ix_ledger_transaction_subject_time" ON "ai_governance"."ledger_transactions" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "subject_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "posted_at" "pg_catalog"."timestamptz_ops" DESC NULLS FIRST
);
CREATE UNIQUE INDEX "uq_ledger_transaction_business_id" ON "ai_governance"."ledger_transactions" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "transaction_id" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);
CREATE UNIQUE INDEX "uq_ledger_transaction_idempotency" ON "ai_governance"."ledger_transactions" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "idempotency_key" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);

-- ----------------------------
-- Triggers structure for table ledger_transactions
-- ----------------------------
CREATE TRIGGER "trg_append_only" BEFORE UPDATE ON "ai_governance"."ledger_transactions"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_append_only_mutation"();
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."ledger_transactions"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE CONSTRAINT TRIGGER "trg_validate_ledger_transaction" AFTER INSERT ON "ai_governance"."ledger_transactions"
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_validate_ledger_transaction"();

-- ----------------------------
-- Checks structure for table ledger_transactions
-- ----------------------------
ALTER TABLE "ai_governance"."ledger_transactions" ADD CONSTRAINT "ledger_transactions_actor_type_check" CHECK (actor_type::text = ANY (ARRAY['USER'::character varying, 'ADMIN'::character varying, 'SERVICE'::character varying, 'SYSTEM'::character varying]::text[]));
ALTER TABLE "ai_governance"."ledger_transactions" ADD CONSTRAINT "ledger_transactions_check" CHECK (entry_type::text = 'REVERSAL'::text AND reversal_of_transaction_id IS NOT NULL OR entry_type::text <> 'REVERSAL'::text);
ALTER TABLE "ai_governance"."ledger_transactions" ADD CONSTRAINT "ck_ledger_transaction_user_dimension" CHECK ((entry_type::text <> ALL (ARRAY['ALLOCATE_TO_USER'::character varying, 'RECLAIM_FROM_USER'::character varying, 'RESERVE'::character varying, 'SUPPLEMENTAL_RESERVE'::character varying, 'SETTLE'::character varying, 'RELEASE'::character varying, 'EMERGENCY_DRAW'::character varying, 'EMERGENCY_COVER'::character varying]::text[])) OR subject_id IS NOT NULL);
ALTER TABLE "ai_governance"."ledger_transactions" ADD CONSTRAINT "ledger_transactions_status_check" CHECK (status::text = 'POSTED'::text);
ALTER TABLE "ai_governance"."ledger_transactions" ADD CONSTRAINT "ledger_transactions_entry_type_check" CHECK (entry_type::text = ANY (ARRAY['ISSUE'::character varying, 'ALLOCATE_TO_SCOPE'::character varying, 'ALLOCATE_TO_USER'::character varying, 'RECLAIM_FROM_USER'::character varying, 'RESERVE'::character varying, 'SUPPLEMENTAL_RESERVE'::character varying, 'SETTLE'::character varying, 'RELEASE'::character varying, 'ADJUST'::character varying, 'REVERSAL'::character varying, 'EMERGENCY_FACILITY_GRANT'::character varying, 'EMERGENCY_DRAW'::character varying, 'EMERGENCY_COVER'::character varying, 'EMERGENCY_CLOSE'::character varying]::text[]));

-- ----------------------------
-- Primary Key structure for table ledger_transactions
-- ----------------------------
ALTER TABLE "ai_governance"."ledger_transactions" ADD CONSTRAINT "ledger_transactions_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table model_access_policies
-- ----------------------------
CREATE INDEX "ix_model_access_lookup" ON "ai_governance"."model_access_policies" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "scope_type" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "scope_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "model_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "status" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);

-- ----------------------------
-- Triggers structure for table model_access_policies
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."model_access_policies"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."model_access_policies"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();
CREATE TRIGGER "trg_validate_soft_delete" BEFORE INSERT OR UPDATE ON "ai_governance"."model_access_policies"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_validate_soft_delete"();

-- ----------------------------
-- Checks structure for table model_access_policies
-- ----------------------------
ALTER TABLE "ai_governance"."model_access_policies" ADD CONSTRAINT "model_access_policies_check1" CHECK (is_deleted AND deleted_at IS NOT NULL AND deleted_by IS NOT NULL AND delete_reason IS NOT NULL OR NOT is_deleted AND deleted_at IS NULL AND deleted_by IS NULL AND delete_reason IS NULL);
ALTER TABLE "ai_governance"."model_access_policies" ADD CONSTRAINT "model_access_policies_effect_check" CHECK (effect::text = ANY (ARRAY['ALLOW'::character varying, 'DENY'::character varying]::text[]));
ALTER TABLE "ai_governance"."model_access_policies" ADD CONSTRAINT "model_access_policies_maximum_data_classification_check" CHECK (maximum_data_classification::text = ANY (ARRAY['PUBLIC'::character varying, 'INTERNAL'::character varying, 'CONFIDENTIAL'::character varying, 'RESTRICTED'::character varying]::text[]));
ALTER TABLE "ai_governance"."model_access_policies" ADD CONSTRAINT "model_access_policies_required_network_class_check" CHECK (required_network_class::text = ANY (ARRAY['INTERNAL'::character varying, 'EXTERNAL'::character varying, 'ANY'::character varying]::text[]));
ALTER TABLE "ai_governance"."model_access_policies" ADD CONSTRAINT "model_access_policies_revision_check" CHECK (revision > 0);
ALTER TABLE "ai_governance"."model_access_policies" ADD CONSTRAINT "model_access_policies_row_version_check" CHECK (row_version > 0);
ALTER TABLE "ai_governance"."model_access_policies" ADD CONSTRAINT "model_access_policies_rps_limit_check" CHECK (rps_limit IS NULL OR rps_limit > 0);
ALTER TABLE "ai_governance"."model_access_policies" ADD CONSTRAINT "model_access_policies_scope_type_check" CHECK (scope_type::text = ANY (ARRAY['TENANT'::character varying, 'ORGANIZATION'::character varying, 'PROJECT'::character varying, 'USER'::character varying]::text[]));
ALTER TABLE "ai_governance"."model_access_policies" ADD CONSTRAINT "model_access_policies_status_check" CHECK (status::text = ANY (ARRAY['DRAFT'::character varying, 'REVIEWING'::character varying, 'APPROVED'::character varying, 'ACTIVE'::character varying, 'REVOKED'::character varying, 'RETIRED'::character varying]::text[]));
ALTER TABLE "ai_governance"."model_access_policies" ADD CONSTRAINT "model_access_policies_subject_type_check" CHECK (subject_type::text = ANY (ARRAY['USER'::character varying, 'SERVICE'::character varying]::text[]));
ALTER TABLE "ai_governance"."model_access_policies" ADD CONSTRAINT "model_access_policies_check" CHECK (valid_until IS NULL OR valid_until > valid_from);
ALTER TABLE "ai_governance"."model_access_policies" ADD CONSTRAINT "model_access_policies_tps_limit_check" CHECK (tps_limit IS NULL OR tps_limit > 0);

-- ----------------------------
-- Primary Key structure for table model_access_policies
-- ----------------------------
ALTER TABLE "ai_governance"."model_access_policies" ADD CONSTRAINT "model_access_policies_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table model_catalog_entries
-- ----------------------------
CREATE INDEX "ix_model_provider_status" ON "ai_governance"."model_catalog_entries" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "provider_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "status" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);
CREATE UNIQUE INDEX "uq_model_alias_version_active" ON "ai_governance"."model_catalog_entries" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "model_alias" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "model_version" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
) WHERE is_deleted = false;

-- ----------------------------
-- Triggers structure for table model_catalog_entries
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."model_catalog_entries"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."model_catalog_entries"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();
CREATE TRIGGER "trg_validate_soft_delete" BEFORE INSERT OR UPDATE ON "ai_governance"."model_catalog_entries"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_validate_soft_delete"();

-- ----------------------------
-- Checks structure for table model_catalog_entries
-- ----------------------------
ALTER TABLE "ai_governance"."model_catalog_entries" ADD CONSTRAINT "model_catalog_entries_check" CHECK (is_deleted AND deleted_at IS NOT NULL AND deleted_by IS NOT NULL AND delete_reason IS NOT NULL OR NOT is_deleted AND deleted_at IS NULL AND deleted_by IS NULL AND delete_reason IS NULL);
ALTER TABLE "ai_governance"."model_catalog_entries" ADD CONSTRAINT "model_catalog_entries_data_classification_ceiling_check" CHECK (data_classification_ceiling::text = ANY (ARRAY['PUBLIC'::character varying, 'INTERNAL'::character varying, 'CONFIDENTIAL'::character varying, 'RESTRICTED'::character varying]::text[]));
ALTER TABLE "ai_governance"."model_catalog_entries" ADD CONSTRAINT "model_catalog_entries_max_context_tokens_check" CHECK (max_context_tokens > 0);
ALTER TABLE "ai_governance"."model_catalog_entries" ADD CONSTRAINT "model_catalog_entries_max_output_tokens_check" CHECK (max_output_tokens > 0);
ALTER TABLE "ai_governance"."model_catalog_entries" ADD CONSTRAINT "model_catalog_entries_network_class_check" CHECK (network_class::text = ANY (ARRAY['INTERNAL'::character varying, 'EXTERNAL'::character varying]::text[]));
ALTER TABLE "ai_governance"."model_catalog_entries" ADD CONSTRAINT "model_catalog_entries_revision_check" CHECK (revision > 0);
ALTER TABLE "ai_governance"."model_catalog_entries" ADD CONSTRAINT "model_catalog_entries_row_version_check" CHECK (row_version > 0);
ALTER TABLE "ai_governance"."model_catalog_entries" ADD CONSTRAINT "model_catalog_entries_capabilities_check" CHECK (cardinality(capabilities) > 0);
ALTER TABLE "ai_governance"."model_catalog_entries" ADD CONSTRAINT "model_catalog_entries_status_check" CHECK (status::text = ANY (ARRAY['DRAFT'::character varying, 'TESTING'::character varying, 'ACTIVE'::character varying, 'SUSPENDED'::character varying, 'RETIRED'::character varying]::text[]));

-- ----------------------------
-- Primary Key structure for table model_catalog_entries
-- ----------------------------
ALTER TABLE "ai_governance"."model_catalog_entries" ADD CONSTRAINT "model_catalog_entries_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table model_channel_health_events
-- ----------------------------
CREATE INDEX "ix_channel_health_observed" ON "ai_governance"."model_channel_health_events" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "channel_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "observed_at" "pg_catalog"."timestamptz_ops" DESC NULLS FIRST
);

-- ----------------------------
-- Triggers structure for table model_channel_health_events
-- ----------------------------
CREATE TRIGGER "trg_append_only" BEFORE UPDATE ON "ai_governance"."model_channel_health_events"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_append_only_mutation"();
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."model_channel_health_events"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();

-- ----------------------------
-- Checks structure for table model_channel_health_events
-- ----------------------------
ALTER TABLE "ai_governance"."model_channel_health_events" ADD CONSTRAINT "model_channel_health_events_consecutive_failures_check" CHECK (consecutive_failures >= 0);
ALTER TABLE "ai_governance"."model_channel_health_events" ADD CONSTRAINT "model_channel_health_events_error_rate_basis_points_check" CHECK (error_rate_basis_points IS NULL OR error_rate_basis_points >= 0 AND error_rate_basis_points <= 10000);
ALTER TABLE "ai_governance"."model_channel_health_events" ADD CONSTRAINT "model_channel_health_events_evidence_json_check" CHECK (jsonb_typeof(evidence_json) = 'object'::text);
ALTER TABLE "ai_governance"."model_channel_health_events" ADD CONSTRAINT "model_channel_health_events_health_status_check" CHECK (health_status::text = ANY (ARRAY['UNKNOWN'::character varying, 'HEALTHY'::character varying, 'DEGRADED'::character varying, 'UNHEALTHY'::character varying, 'CIRCUIT_OPEN'::character varying]::text[]));
ALTER TABLE "ai_governance"."model_channel_health_events" ADD CONSTRAINT "model_channel_health_events_latency_ms_check" CHECK (latency_ms IS NULL OR latency_ms >= 0);

-- ----------------------------
-- Primary Key structure for table model_channel_health_events
-- ----------------------------
ALTER TABLE "ai_governance"."model_channel_health_events" ADD CONSTRAINT "model_channel_health_events_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table model_channels
-- ----------------------------
CREATE INDEX "ix_model_channel_route" ON "ai_governance"."model_channels" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "model_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "network_class" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "status" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "health_status" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);
CREATE UNIQUE INDEX "uq_model_channel_code_active" ON "ai_governance"."model_channels" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "channel_code" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
) WHERE is_deleted = false;

-- ----------------------------
-- Triggers structure for table model_channels
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."model_channels"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."model_channels"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();
CREATE TRIGGER "trg_validate_soft_delete" BEFORE INSERT OR UPDATE ON "ai_governance"."model_channels"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_validate_soft_delete"();

-- ----------------------------
-- Checks structure for table model_channels
-- ----------------------------
ALTER TABLE "ai_governance"."model_channels" ADD CONSTRAINT "model_channels_capacity_tps_check" CHECK (capacity_tps > 0);
ALTER TABLE "ai_governance"."model_channels" ADD CONSTRAINT "model_channels_check" CHECK (is_deleted AND deleted_at IS NOT NULL AND deleted_by IS NOT NULL AND delete_reason IS NOT NULL OR NOT is_deleted AND deleted_at IS NULL AND deleted_by IS NULL AND delete_reason IS NULL);
ALTER TABLE "ai_governance"."model_channels" ADD CONSTRAINT "model_channels_health_status_check" CHECK (health_status::text = ANY (ARRAY['UNKNOWN'::character varying, 'HEALTHY'::character varying, 'DEGRADED'::character varying, 'UNHEALTHY'::character varying, 'CIRCUIT_OPEN'::character varying]::text[]));
ALTER TABLE "ai_governance"."model_channels" ADD CONSTRAINT "model_channels_network_class_check" CHECK (network_class::text = ANY (ARRAY['INTERNAL'::character varying, 'EXTERNAL'::character varying]::text[]));
ALTER TABLE "ai_governance"."model_channels" ADD CONSTRAINT "model_channels_revision_check" CHECK (revision > 0);
ALTER TABLE "ai_governance"."model_channels" ADD CONSTRAINT "model_channels_row_version_check" CHECK (row_version > 0);
ALTER TABLE "ai_governance"."model_channels" ADD CONSTRAINT "model_channels_capacity_rps_check" CHECK (capacity_rps > 0);
ALTER TABLE "ai_governance"."model_channels" ADD CONSTRAINT "model_channels_status_check" CHECK (status::text = ANY (ARRAY['DRAFT'::character varying, 'ACTIVE'::character varying, 'SUSPENDED'::character varying, 'RETIRED'::character varying]::text[]));

-- ----------------------------
-- Primary Key structure for table model_channels
-- ----------------------------
ALTER TABLE "ai_governance"."model_channels" ADD CONSTRAINT "model_channels_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table model_providers
-- ----------------------------
CREATE UNIQUE INDEX "uq_model_provider_code_active" ON "ai_governance"."model_providers" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "provider_code" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
) WHERE is_deleted = false;

-- ----------------------------
-- Triggers structure for table model_providers
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."model_providers"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."model_providers"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();
CREATE TRIGGER "trg_validate_soft_delete" BEFORE INSERT OR UPDATE ON "ai_governance"."model_providers"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_validate_soft_delete"();

-- ----------------------------
-- Checks structure for table model_providers
-- ----------------------------
ALTER TABLE "ai_governance"."model_providers" ADD CONSTRAINT "model_providers_network_class_check" CHECK (network_class::text = ANY (ARRAY['INTERNAL'::character varying, 'EXTERNAL'::character varying]::text[]));
ALTER TABLE "ai_governance"."model_providers" ADD CONSTRAINT "model_providers_protocol_type_check" CHECK (protocol_type::text = ANY (ARRAY['OPENAI_COMPATIBLE'::character varying, 'AZURE_OPENAI'::character varying, 'CUSTOM_ADAPTER'::character varying]::text[]));
ALTER TABLE "ai_governance"."model_providers" ADD CONSTRAINT "model_providers_provider_type_check" CHECK (provider_type::text = ANY (ARRAY['INTERNAL'::character varying, 'EXTERNAL'::character varying, 'HYBRID'::character varying]::text[]));
ALTER TABLE "ai_governance"."model_providers" ADD CONSTRAINT "model_providers_regions_check" CHECK (cardinality(regions) > 0);
ALTER TABLE "ai_governance"."model_providers" ADD CONSTRAINT "model_providers_revision_check" CHECK (revision > 0);
ALTER TABLE "ai_governance"."model_providers" ADD CONSTRAINT "model_providers_row_version_check" CHECK (row_version > 0);
ALTER TABLE "ai_governance"."model_providers" ADD CONSTRAINT "model_providers_check" CHECK (is_deleted AND deleted_at IS NOT NULL AND deleted_by IS NOT NULL AND delete_reason IS NOT NULL OR NOT is_deleted AND deleted_at IS NULL AND deleted_by IS NULL AND delete_reason IS NULL);
ALTER TABLE "ai_governance"."model_providers" ADD CONSTRAINT "model_providers_status_check" CHECK (status::text = ANY (ARRAY['DRAFT'::character varying, 'ACTIVE'::character varying, 'DEGRADED'::character varying, 'SUSPENDED'::character varying, 'RETIRED'::character varying]::text[]));

-- ----------------------------
-- Primary Key structure for table model_providers
-- ----------------------------
ALTER TABLE "ai_governance"."model_providers" ADD CONSTRAINT "model_providers_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table oauth_provider_configs
-- ----------------------------
CREATE UNIQUE INDEX "uq_oauth_provider_code_active" ON "ai_governance"."oauth_provider_configs" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "provider_code" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
) WHERE is_deleted = false;

-- ----------------------------
-- Triggers structure for table oauth_provider_configs
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."oauth_provider_configs"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."oauth_provider_configs"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();
CREATE TRIGGER "trg_validate_soft_delete" BEFORE INSERT OR UPDATE ON "ai_governance"."oauth_provider_configs"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_validate_soft_delete"();

-- ----------------------------
-- Checks structure for table oauth_provider_configs
-- ----------------------------
ALTER TABLE "ai_governance"."oauth_provider_configs" ADD CONSTRAINT "oauth_provider_configs_check" CHECK (is_deleted AND deleted_at IS NOT NULL AND deleted_by IS NOT NULL AND delete_reason IS NOT NULL OR NOT is_deleted AND deleted_at IS NULL AND deleted_by IS NULL AND delete_reason IS NULL);
ALTER TABLE "ai_governance"."oauth_provider_configs" ADD CONSTRAINT "oauth_provider_configs_provider_type_check" CHECK (provider_type::text = ANY (ARRAY['WECHAT'::character varying, 'OIDC'::character varying]::text[]));
ALTER TABLE "ai_governance"."oauth_provider_configs" ADD CONSTRAINT "oauth_provider_configs_revision_check" CHECK (revision > 0);
ALTER TABLE "ai_governance"."oauth_provider_configs" ADD CONSTRAINT "oauth_provider_configs_row_version_check" CHECK (row_version > 0);
ALTER TABLE "ai_governance"."oauth_provider_configs" ADD CONSTRAINT "oauth_provider_configs_status_check" CHECK (status::text = ANY (ARRAY['DRAFT'::character varying, 'ACTIVE'::character varying, 'SUSPENDED'::character varying, 'RETIRED'::character varying]::text[]));

-- ----------------------------
-- Primary Key structure for table oauth_provider_configs
-- ----------------------------
ALTER TABLE "ai_governance"."oauth_provider_configs" ADD CONSTRAINT "oauth_provider_configs_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table organization_memberships
-- ----------------------------
CREATE INDEX "ix_org_membership_subject" ON "ai_governance"."organization_memberships" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "subject_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "status" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);
CREATE UNIQUE INDEX "uq_org_membership_active" ON "ai_governance"."organization_memberships" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "organization_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "subject_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "org_role_code" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
) WHERE is_deleted = false AND (status::text = ANY (ARRAY['PENDING'::character varying, 'ACTIVE'::character varying, 'SUSPENDED'::character varying]::text[]));

-- ----------------------------
-- Triggers structure for table organization_memberships
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."organization_memberships"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."organization_memberships"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();
CREATE TRIGGER "trg_validate_soft_delete" BEFORE INSERT OR UPDATE ON "ai_governance"."organization_memberships"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_validate_soft_delete"();

-- ----------------------------
-- Checks structure for table organization_memberships
-- ----------------------------
ALTER TABLE "ai_governance"."organization_memberships" ADD CONSTRAINT "organization_memberships_check" CHECK (valid_until IS NULL OR valid_until > valid_from);
ALTER TABLE "ai_governance"."organization_memberships" ADD CONSTRAINT "organization_memberships_check1" CHECK (is_deleted AND deleted_at IS NOT NULL AND deleted_by IS NOT NULL AND delete_reason IS NOT NULL OR NOT is_deleted AND deleted_at IS NULL AND deleted_by IS NULL AND delete_reason IS NULL);
ALTER TABLE "ai_governance"."organization_memberships" ADD CONSTRAINT "organization_memberships_membership_revision_check" CHECK (membership_revision > 0);
ALTER TABLE "ai_governance"."organization_memberships" ADD CONSTRAINT "organization_memberships_row_version_check" CHECK (row_version > 0);
ALTER TABLE "ai_governance"."organization_memberships" ADD CONSTRAINT "organization_memberships_status_check" CHECK (status::text = ANY (ARRAY['PENDING'::character varying, 'ACTIVE'::character varying, 'SUSPENDED'::character varying, 'ENDED'::character varying]::text[]));

-- ----------------------------
-- Primary Key structure for table organization_memberships
-- ----------------------------
ALTER TABLE "ai_governance"."organization_memberships" ADD CONSTRAINT "organization_memberships_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table organizations
-- ----------------------------
CREATE INDEX "ix_org_parent" ON "ai_governance"."organizations" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "parent_organization_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "status" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);
CREATE UNIQUE INDEX "uq_org_code_active" ON "ai_governance"."organizations" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "organization_code" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
) WHERE is_deleted = false;
CREATE UNIQUE INDEX "uq_org_path_active" ON "ai_governance"."organizations" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "organization_path" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
) WHERE is_deleted = false;

-- ----------------------------
-- Triggers structure for table organizations
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."organizations"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."organizations"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();
CREATE TRIGGER "trg_validate_soft_delete" BEFORE INSERT OR UPDATE ON "ai_governance"."organizations"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_validate_soft_delete"();

-- ----------------------------
-- Checks structure for table organizations
-- ----------------------------
ALTER TABLE "ai_governance"."organizations" ADD CONSTRAINT "organizations_check1" CHECK (is_deleted AND deleted_at IS NOT NULL AND deleted_by IS NOT NULL AND delete_reason IS NOT NULL OR NOT is_deleted AND deleted_at IS NULL AND deleted_by IS NULL AND delete_reason IS NULL);
ALTER TABLE "ai_governance"."organizations" ADD CONSTRAINT "organizations_depth_check" CHECK (depth >= 0 AND depth <= 32);
ALTER TABLE "ai_governance"."organizations" ADD CONSTRAINT "organizations_organization_type_check" CHECK (organization_type::text = ANY (ARRAY['COMPANY'::character varying, 'DIVISION'::character varying, 'DEPARTMENT'::character varying, 'TEAM'::character varying]::text[]));
ALTER TABLE "ai_governance"."organizations" ADD CONSTRAINT "organizations_revision_check" CHECK (revision > 0);
ALTER TABLE "ai_governance"."organizations" ADD CONSTRAINT "organizations_row_version_check" CHECK (row_version > 0);
ALTER TABLE "ai_governance"."organizations" ADD CONSTRAINT "organizations_check" CHECK (depth = 0 AND parent_organization_id IS NULL OR depth > 0 AND parent_organization_id IS NOT NULL);
ALTER TABLE "ai_governance"."organizations" ADD CONSTRAINT "organizations_status_check" CHECK (status::text = ANY (ARRAY['DRAFT'::character varying, 'ACTIVE'::character varying, 'SUSPENDED'::character varying, 'ARCHIVED'::character varying]::text[]));

-- ----------------------------
-- Primary Key structure for table organizations
-- ----------------------------
ALTER TABLE "ai_governance"."organizations" ADD CONSTRAINT "organizations_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table outbox_events
-- ----------------------------
CREATE INDEX "ix_outbox_dispatch" ON "ai_governance"."outbox_events" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "status" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "available_at" "pg_catalog"."timestamptz_ops" ASC NULLS LAST,
  "created_at" "pg_catalog"."timestamptz_ops" ASC NULLS LAST
);
CREATE UNIQUE INDEX "uq_outbox_event_business_id" ON "ai_governance"."outbox_events" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "event_id" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);

-- ----------------------------
-- Triggers structure for table outbox_events
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."outbox_events"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."outbox_events"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();

-- ----------------------------
-- Checks structure for table outbox_events
-- ----------------------------
ALTER TABLE "ai_governance"."outbox_events" ADD CONSTRAINT "outbox_events_attempt_count_check" CHECK (attempt_count >= 0);
ALTER TABLE "ai_governance"."outbox_events" ADD CONSTRAINT "outbox_events_check" CHECK (status::text = 'PROCESSING'::text AND lease_owner IS NOT NULL AND lease_until IS NOT NULL OR status::text <> 'PROCESSING'::text);
ALTER TABLE "ai_governance"."outbox_events" ADD CONSTRAINT "outbox_events_aggregate_revision_check" CHECK (aggregate_revision > 0);
ALTER TABLE "ai_governance"."outbox_events" ADD CONSTRAINT "outbox_events_row_version_check" CHECK (row_version > 0);
ALTER TABLE "ai_governance"."outbox_events" ADD CONSTRAINT "outbox_events_status_check" CHECK (status::text = ANY (ARRAY['PENDING'::character varying, 'PROCESSING'::character varying, 'PUBLISHED'::character varying, 'FAILED'::character varying, 'DEAD_LETTER'::character varying]::text[]));
ALTER TABLE "ai_governance"."outbox_events" ADD CONSTRAINT "outbox_events_payload_json_check" CHECK (jsonb_typeof(payload_json) = 'object'::text);

-- ----------------------------
-- Primary Key structure for table outbox_events
-- ----------------------------
ALTER TABLE "ai_governance"."outbox_events" ADD CONSTRAINT "outbox_events_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table project_memberships
-- ----------------------------
CREATE INDEX "ix_project_membership_subject" ON "ai_governance"."project_memberships" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "subject_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "status" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);
CREATE UNIQUE INDEX "uq_project_membership_active" ON "ai_governance"."project_memberships" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "project_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "subject_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "project_role_code" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
) WHERE is_deleted = false AND (status::text = ANY (ARRAY['INVITED'::character varying, 'ACTIVE'::character varying, 'SUSPENDED'::character varying]::text[]));

-- ----------------------------
-- Triggers structure for table project_memberships
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."project_memberships"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."project_memberships"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();
CREATE TRIGGER "trg_validate_soft_delete" BEFORE INSERT OR UPDATE ON "ai_governance"."project_memberships"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_validate_soft_delete"();

-- ----------------------------
-- Checks structure for table project_memberships
-- ----------------------------
ALTER TABLE "ai_governance"."project_memberships" ADD CONSTRAINT "project_memberships_check" CHECK (valid_until IS NULL OR valid_until > valid_from);
ALTER TABLE "ai_governance"."project_memberships" ADD CONSTRAINT "project_memberships_check1" CHECK (is_deleted AND deleted_at IS NOT NULL AND deleted_by IS NOT NULL AND delete_reason IS NOT NULL OR NOT is_deleted AND deleted_at IS NULL AND deleted_by IS NULL AND delete_reason IS NULL);
ALTER TABLE "ai_governance"."project_memberships" ADD CONSTRAINT "project_memberships_membership_revision_check" CHECK (membership_revision > 0);
ALTER TABLE "ai_governance"."project_memberships" ADD CONSTRAINT "project_memberships_row_version_check" CHECK (row_version > 0);
ALTER TABLE "ai_governance"."project_memberships" ADD CONSTRAINT "project_memberships_status_check" CHECK (status::text = ANY (ARRAY['INVITED'::character varying, 'ACTIVE'::character varying, 'SUSPENDED'::character varying, 'REMOVED'::character varying]::text[]));

-- ----------------------------
-- Primary Key structure for table project_memberships
-- ----------------------------
ALTER TABLE "ai_governance"."project_memberships" ADD CONSTRAINT "project_memberships_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table projection_checkpoints
-- ----------------------------
CREATE UNIQUE INDEX "uq_projection_checkpoint" ON "ai_governance"."projection_checkpoints" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "projection_code" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "partition_key" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);

-- ----------------------------
-- Triggers structure for table projection_checkpoints
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."projection_checkpoints"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."projection_checkpoints"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();

-- ----------------------------
-- Checks structure for table projection_checkpoints
-- ----------------------------
ALTER TABLE "ai_governance"."projection_checkpoints" ADD CONSTRAINT "projection_checkpoints_generation_check" CHECK (generation > 0);
ALTER TABLE "ai_governance"."projection_checkpoints" ADD CONSTRAINT "projection_checkpoints_row_version_check" CHECK (row_version > 0);
ALTER TABLE "ai_governance"."projection_checkpoints" ADD CONSTRAINT "projection_checkpoints_status_check" CHECK (status::text = ANY (ARRAY['BUILDING'::character varying, 'CURRENT'::character varying, 'STALE'::character varying, 'FAILED'::character varying]::text[]));

-- ----------------------------
-- Primary Key structure for table projection_checkpoints
-- ----------------------------
ALTER TABLE "ai_governance"."projection_checkpoints" ADD CONSTRAINT "projection_checkpoints_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table projects
-- ----------------------------
CREATE INDEX "ix_project_org_status" ON "ai_governance"."projects" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "sponsor_organization_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "status" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);
CREATE UNIQUE INDEX "uq_project_code_active" ON "ai_governance"."projects" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "project_code" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
) WHERE is_deleted = false;

-- ----------------------------
-- Triggers structure for table projects
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."projects"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."projects"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();
CREATE TRIGGER "trg_validate_soft_delete" BEFORE INSERT OR UPDATE ON "ai_governance"."projects"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_validate_soft_delete"();

-- ----------------------------
-- Checks structure for table projects
-- ----------------------------
ALTER TABLE "ai_governance"."projects" ADD CONSTRAINT "projects_check1" CHECK (is_deleted AND deleted_at IS NOT NULL AND deleted_by IS NOT NULL AND delete_reason IS NOT NULL OR NOT is_deleted AND deleted_at IS NULL AND deleted_by IS NULL AND delete_reason IS NULL);
ALTER TABLE "ai_governance"."projects" ADD CONSTRAINT "projects_data_classification_check" CHECK (data_classification::text = ANY (ARRAY['PUBLIC'::character varying, 'INTERNAL'::character varying, 'CONFIDENTIAL'::character varying, 'RESTRICTED'::character varying]::text[]));
ALTER TABLE "ai_governance"."projects" ADD CONSTRAINT "projects_check" CHECK (valid_until IS NULL OR valid_from IS NULL OR valid_until > valid_from);
ALTER TABLE "ai_governance"."projects" ADD CONSTRAINT "projects_revision_check" CHECK (revision > 0);
ALTER TABLE "ai_governance"."projects" ADD CONSTRAINT "projects_row_version_check" CHECK (row_version > 0);
ALTER TABLE "ai_governance"."projects" ADD CONSTRAINT "projects_status_check" CHECK (status::text = ANY (ARRAY['DRAFT'::character varying, 'ACTIVE'::character varying, 'PAUSED'::character varying, 'CLOSING'::character varying, 'ARCHIVED'::character varying]::text[]));

-- ----------------------------
-- Primary Key structure for table projects
-- ----------------------------
ALTER TABLE "ai_governance"."projects" ADD CONSTRAINT "projects_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table prompt_firewall_decisions
-- ----------------------------
CREATE UNIQUE INDEX "uq_firewall_decision_sequence" ON "ai_governance"."prompt_firewall_decisions" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "request_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "direction" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "decision_sequence" "pg_catalog"."int2_ops" ASC NULLS LAST
);

-- ----------------------------
-- Triggers structure for table prompt_firewall_decisions
-- ----------------------------
CREATE TRIGGER "trg_append_only" BEFORE UPDATE ON "ai_governance"."prompt_firewall_decisions"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_append_only_mutation"();
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."prompt_firewall_decisions"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();

-- ----------------------------
-- Checks structure for table prompt_firewall_decisions
-- ----------------------------
ALTER TABLE "ai_governance"."prompt_firewall_decisions" ADD CONSTRAINT "prompt_firewall_decisions_decision_sequence_check" CHECK (decision_sequence > 0);
ALTER TABLE "ai_governance"."prompt_firewall_decisions" ADD CONSTRAINT "prompt_firewall_decisions_direction_check" CHECK (direction::text = ANY (ARRAY['INPUT'::character varying, 'OUTPUT'::character varying]::text[]));
ALTER TABLE "ai_governance"."prompt_firewall_decisions" ADD CONSTRAINT "prompt_firewall_decisions_action_check" CHECK (action::text = ANY (ARRAY['ALLOW'::character varying, 'BLOCK'::character varying, 'REDACT'::character varying, 'FORCE_INTERNAL'::character varying]::text[]));
ALTER TABLE "ai_governance"."prompt_firewall_decisions" ADD CONSTRAINT "prompt_firewall_decisions_finding_summary_json_check" CHECK (jsonb_typeof(finding_summary_json) = 'object'::text);
ALTER TABLE "ai_governance"."prompt_firewall_decisions" ADD CONSTRAINT "prompt_firewall_decisions_policy_revision_check" CHECK (policy_revision > 0);
ALTER TABLE "ai_governance"."prompt_firewall_decisions" ADD CONSTRAINT "prompt_firewall_decisions_effective_data_classification_check" CHECK (effective_data_classification::text = ANY (ARRAY['PUBLIC'::character varying, 'INTERNAL'::character varying, 'CONFIDENTIAL'::character varying, 'RESTRICTED'::character varying]::text[]));

-- ----------------------------
-- Primary Key structure for table prompt_firewall_decisions
-- ----------------------------
ALTER TABLE "ai_governance"."prompt_firewall_decisions" ADD CONSTRAINT "prompt_firewall_decisions_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table prompt_firewall_policies
-- ----------------------------
CREATE UNIQUE INDEX "uq_prompt_firewall_policy_active" ON "ai_governance"."prompt_firewall_policies" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "policy_code" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
) WHERE is_deleted = false;

-- ----------------------------
-- Triggers structure for table prompt_firewall_policies
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."prompt_firewall_policies"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."prompt_firewall_policies"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();
CREATE TRIGGER "trg_validate_soft_delete" BEFORE INSERT OR UPDATE ON "ai_governance"."prompt_firewall_policies"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_validate_soft_delete"();

-- ----------------------------
-- Checks structure for table prompt_firewall_policies
-- ----------------------------
ALTER TABLE "ai_governance"."prompt_firewall_policies" ADD CONSTRAINT "prompt_firewall_policies_check1" CHECK (is_deleted AND deleted_at IS NOT NULL AND deleted_by IS NOT NULL AND delete_reason IS NOT NULL OR NOT is_deleted AND deleted_at IS NULL AND deleted_by IS NULL AND delete_reason IS NULL);
ALTER TABLE "ai_governance"."prompt_firewall_policies" ADD CONSTRAINT "prompt_firewall_policies_default_action_check" CHECK (default_action::text = ANY (ARRAY['ALLOW'::character varying, 'BLOCK'::character varying, 'REDACT'::character varying, 'FORCE_INTERNAL'::character varying]::text[]));
ALTER TABLE "ai_governance"."prompt_firewall_policies" ADD CONSTRAINT "prompt_firewall_policies_direction_check" CHECK (direction::text = ANY (ARRAY['INPUT'::character varying, 'OUTPUT'::character varying, 'BOTH'::character varying]::text[]));
ALTER TABLE "ai_governance"."prompt_firewall_policies" ADD CONSTRAINT "prompt_firewall_policies_fail_mode_check" CHECK (fail_mode::text = ANY (ARRAY['CLOSED'::character varying, 'LKG_INTERNAL_ONLY'::character varying]::text[]));
ALTER TABLE "ai_governance"."prompt_firewall_policies" ADD CONSTRAINT "prompt_firewall_policies_revision_check" CHECK (revision > 0);
ALTER TABLE "ai_governance"."prompt_firewall_policies" ADD CONSTRAINT "prompt_firewall_policies_row_version_check" CHECK (row_version > 0);
ALTER TABLE "ai_governance"."prompt_firewall_policies" ADD CONSTRAINT "prompt_firewall_policies_scope_type_check" CHECK (scope_type::text = ANY (ARRAY['TENANT'::character varying, 'ORGANIZATION'::character varying, 'PROJECT'::character varying, 'USER'::character varying]::text[]));
ALTER TABLE "ai_governance"."prompt_firewall_policies" ADD CONSTRAINT "prompt_firewall_policies_check" CHECK (valid_until IS NULL OR valid_until > valid_from);
ALTER TABLE "ai_governance"."prompt_firewall_policies" ADD CONSTRAINT "prompt_firewall_policies_status_check" CHECK (status::text = ANY (ARRAY['DRAFT'::character varying, 'REVIEWING'::character varying, 'APPROVED'::character varying, 'ACTIVE'::character varying, 'REVOKED'::character varying, 'RETIRED'::character varying]::text[]));

-- ----------------------------
-- Primary Key structure for table prompt_firewall_policies
-- ----------------------------
ALTER TABLE "ai_governance"."prompt_firewall_policies" ADD CONSTRAINT "prompt_firewall_policies_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table prompt_firewall_rules
-- ----------------------------
CREATE INDEX "ix_prompt_firewall_rule_eval" ON "ai_governance"."prompt_firewall_rules" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "firewall_policy_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "direction" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "status" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "priority" "pg_catalog"."int4_ops" ASC NULLS LAST
);
CREATE UNIQUE INDEX "uq_prompt_firewall_rule_active" ON "ai_governance"."prompt_firewall_rules" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "firewall_policy_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "rule_code" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
) WHERE is_deleted = false;

-- ----------------------------
-- Triggers structure for table prompt_firewall_rules
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."prompt_firewall_rules"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."prompt_firewall_rules"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();
CREATE TRIGGER "trg_validate_soft_delete" BEFORE INSERT OR UPDATE ON "ai_governance"."prompt_firewall_rules"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_validate_soft_delete"();

-- ----------------------------
-- Checks structure for table prompt_firewall_rules
-- ----------------------------
ALTER TABLE "ai_governance"."prompt_firewall_rules" ADD CONSTRAINT "prompt_firewall_rules_check" CHECK (is_deleted AND deleted_at IS NOT NULL AND deleted_by IS NOT NULL AND delete_reason IS NOT NULL OR NOT is_deleted AND deleted_at IS NULL AND deleted_by IS NULL AND delete_reason IS NULL);
ALTER TABLE "ai_governance"."prompt_firewall_rules" ADD CONSTRAINT "prompt_firewall_rules_direction_check" CHECK (direction::text = ANY (ARRAY['INPUT'::character varying, 'OUTPUT'::character varying]::text[]));
ALTER TABLE "ai_governance"."prompt_firewall_rules" ADD CONSTRAINT "prompt_firewall_rules_priority_check" CHECK (priority >= 0 AND priority <= 100000);
ALTER TABLE "ai_governance"."prompt_firewall_rules" ADD CONSTRAINT "prompt_firewall_rules_revision_check" CHECK (revision > 0);
ALTER TABLE "ai_governance"."prompt_firewall_rules" ADD CONSTRAINT "prompt_firewall_rules_row_version_check" CHECK (row_version > 0);
ALTER TABLE "ai_governance"."prompt_firewall_rules" ADD CONSTRAINT "prompt_firewall_rules_rule_type_check" CHECK (rule_type::text = ANY (ARRAY['REGEX'::character varying, 'KEYWORD_SET'::character varying, 'PII_DETECTOR'::character varying, 'CLASSIFIER'::character varying, 'INJECTION_DETECTOR'::character varying]::text[]));
ALTER TABLE "ai_governance"."prompt_firewall_rules" ADD CONSTRAINT "prompt_firewall_rules_severity_check" CHECK (severity::text = ANY (ARRAY['LOW'::character varying, 'MEDIUM'::character varying, 'HIGH'::character varying, 'CRITICAL'::character varying]::text[]));
ALTER TABLE "ai_governance"."prompt_firewall_rules" ADD CONSTRAINT "prompt_firewall_rules_action_check" CHECK (action::text = ANY (ARRAY['ALLOW'::character varying, 'BLOCK'::character varying, 'REDACT'::character varying, 'FORCE_INTERNAL'::character varying, 'ALERT'::character varying]::text[]));
ALTER TABLE "ai_governance"."prompt_firewall_rules" ADD CONSTRAINT "prompt_firewall_rules_status_check" CHECK (status::text = ANY (ARRAY['DRAFT'::character varying, 'ACTIVE'::character varying, 'SUSPENDED'::character varying, 'RETIRED'::character varying]::text[]));

-- ----------------------------
-- Primary Key structure for table prompt_firewall_rules
-- ----------------------------
ALTER TABLE "ai_governance"."prompt_firewall_rules" ADD CONSTRAINT "prompt_firewall_rules_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table provider_credential_refs
-- ----------------------------
CREATE UNIQUE INDEX "uq_provider_credential_name_active" ON "ai_governance"."provider_credential_refs" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "provider_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "credential_name" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
) WHERE is_deleted = false;

-- ----------------------------
-- Triggers structure for table provider_credential_refs
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."provider_credential_refs"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."provider_credential_refs"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();
CREATE TRIGGER "trg_validate_soft_delete" BEFORE INSERT OR UPDATE ON "ai_governance"."provider_credential_refs"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_validate_soft_delete"();

-- ----------------------------
-- Checks structure for table provider_credential_refs
-- ----------------------------
ALTER TABLE "ai_governance"."provider_credential_refs" ADD CONSTRAINT "provider_credential_refs_check1" CHECK (is_deleted AND deleted_at IS NOT NULL AND deleted_by IS NOT NULL AND delete_reason IS NOT NULL OR NOT is_deleted AND deleted_at IS NULL AND deleted_by IS NULL AND delete_reason IS NULL);
ALTER TABLE "ai_governance"."provider_credential_refs" ADD CONSTRAINT "provider_credential_refs_credential_purpose_check" CHECK (credential_purpose::text = ANY (ARRAY['INFERENCE'::character varying, 'DISCOVERY'::character varying, 'BILLING'::character varying, 'HEALTH_CHECK'::character varying]::text[]));
ALTER TABLE "ai_governance"."provider_credential_refs" ADD CONSTRAINT "provider_credential_refs_check" CHECK (valid_until IS NULL OR valid_until > valid_from);
ALTER TABLE "ai_governance"."provider_credential_refs" ADD CONSTRAINT "provider_credential_refs_revision_check" CHECK (revision > 0);
ALTER TABLE "ai_governance"."provider_credential_refs" ADD CONSTRAINT "provider_credential_refs_row_version_check" CHECK (row_version > 0);
ALTER TABLE "ai_governance"."provider_credential_refs" ADD CONSTRAINT "provider_credential_refs_status_check" CHECK (status::text = ANY (ARRAY['PENDING'::character varying, 'ACTIVE'::character varying, 'ROTATING'::character varying, 'REVOKED'::character varying, 'EXPIRED'::character varying]::text[]));

-- ----------------------------
-- Primary Key structure for table provider_credential_refs
-- ----------------------------
ALTER TABLE "ai_governance"."provider_credential_refs" ADD CONSTRAINT "provider_credential_refs_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table quota_reservations
-- ----------------------------
CREATE INDEX "ix_quota_reservation_allocation" ON "ai_governance"."quota_reservations" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "user_allocation_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "status" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "expires_at" "pg_catalog"."timestamptz_ops" ASC NULLS LAST
);
CREATE INDEX "ix_quota_reservation_key" ON "ai_governance"."quota_reservations" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "key_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "status" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "expires_at" "pg_catalog"."timestamptz_ops" ASC NULLS LAST
);
CREATE INDEX "ix_reservation_subject_status" ON "ai_governance"."quota_reservations" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "subject_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "status" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "created_at" "pg_catalog"."timestamptz_ops" DESC NULLS FIRST
);
CREATE UNIQUE INDEX "uq_quota_reservation_business_id" ON "ai_governance"."quota_reservations" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "reservation_id" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);
CREATE UNIQUE INDEX "uq_quota_reservation_request" ON "ai_governance"."quota_reservations" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "request_id" "pg_catalog"."uuid_ops" ASC NULLS LAST
);

-- ----------------------------
-- Triggers structure for table quota_reservations
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."quota_reservations"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."quota_reservations"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();

-- ----------------------------
-- Checks structure for table quota_reservations
-- ----------------------------
ALTER TABLE "ai_governance"."quota_reservations" ADD CONSTRAINT "quota_reservations_check" CHECK (total_reserved_credit >= initial_reserved_credit);
ALTER TABLE "ai_governance"."quota_reservations" ADD CONSTRAINT "quota_reservations_check1" CHECK (expires_at > created_at);
ALTER TABLE "ai_governance"."quota_reservations" ADD CONSTRAINT "quota_reservations_check2" CHECK (status::text = 'SETTLED'::text AND settled_at IS NOT NULL OR status::text <> 'SETTLED'::text);
ALTER TABLE "ai_governance"."quota_reservations" ADD CONSTRAINT "quota_reservations_check3" CHECK (status::text = 'RELEASED'::text AND released_at IS NOT NULL OR status::text <> 'RELEASED'::text);
ALTER TABLE "ai_governance"."quota_reservations" ADD CONSTRAINT "quota_reservations_credit_rate_revision_check" CHECK (credit_rate_revision > 0);
ALTER TABLE "ai_governance"."quota_reservations" ADD CONSTRAINT "quota_reservations_estimated_token_ceiling_check" CHECK (estimated_token_ceiling > 0);
ALTER TABLE "ai_governance"."quota_reservations" ADD CONSTRAINT "quota_reservations_initial_reserved_credit_check" CHECK (initial_reserved_credit > 0);
ALTER TABLE "ai_governance"."quota_reservations" ADD CONSTRAINT "quota_reservations_key_revision_check" CHECK (key_revision > 0);
ALTER TABLE "ai_governance"."quota_reservations" ADD CONSTRAINT "quota_reservations_row_version_check" CHECK (row_version > 0);
ALTER TABLE "ai_governance"."quota_reservations" ADD CONSTRAINT "quota_reservations_status_check" CHECK (status::text = ANY (ARRAY['RESERVED'::character varying, 'IN_FLIGHT'::character varying, 'SETTLED'::character varying, 'RELEASED'::character varying, 'RECONCILING'::character varying, 'EXPIRED'::character varying]::text[]));
ALTER TABLE "ai_governance"."quota_reservations" ADD CONSTRAINT "quota_reservations_billing_scope_type_check" CHECK (billing_scope_type::text = ANY (ARRAY['ORGANIZATION'::character varying, 'PROJECT'::character varying]::text[]));
ALTER TABLE "ai_governance"."quota_reservations" ADD CONSTRAINT "quota_reservations_total_reserved_credit_check" CHECK (total_reserved_credit > 0);

-- ----------------------------
-- Primary Key structure for table quota_reservations
-- ----------------------------
ALTER TABLE "ai_governance"."quota_reservations" ADD CONSTRAINT "quota_reservations_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table reconciliation_differences
-- ----------------------------
CREATE INDEX "ix_reconciliation_difference_status" ON "ai_governance"."reconciliation_differences" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "status" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "severity" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "detected_at" "pg_catalog"."timestamptz_ops" ASC NULLS LAST
);
CREATE INDEX "ix_reconciliation_difference_subject" ON "ai_governance"."reconciliation_differences" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "subject_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "detected_at" "pg_catalog"."timestamptz_ops" DESC NULLS FIRST
);
CREATE UNIQUE INDEX "uq_reconciliation_difference_code" ON "ai_governance"."reconciliation_differences" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "reconciliation_run_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "difference_code" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);

-- ----------------------------
-- Triggers structure for table reconciliation_differences
-- ----------------------------
CREATE TRIGGER "trg_append_only" BEFORE UPDATE ON "ai_governance"."reconciliation_differences"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_append_only_mutation"();
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."reconciliation_differences"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();

-- ----------------------------
-- Checks structure for table reconciliation_differences
-- ----------------------------
ALTER TABLE "ai_governance"."reconciliation_differences" ADD CONSTRAINT "reconciliation_differences_difference_type_check" CHECK (difference_type::text = ANY (ARRAY['LEDGER_UNBALANCED'::character varying, 'RESERVATION_OPEN'::character varying, 'USAGE_MISMATCH'::character varying, 'RATE_MISMATCH'::character varying, 'KEY_ROLLUP_MISMATCH'::character varying, 'USER_ROLLUP_MISMATCH'::character varying, 'SCOPE_ROLLUP_MISMATCH'::character varying, 'COMPANY_ROLLUP_MISMATCH'::character varying, 'EMERGENCY_GRANT_VIOLATION'::character varying, 'REPLAY_MISMATCH'::character varying, 'AUDIT_MISSING'::character varying, 'REFERENCE_ORPHAN'::character varying]::text[]));
ALTER TABLE "ai_governance"."reconciliation_differences" ADD CONSTRAINT "reconciliation_differences_evidence_json_check" CHECK (jsonb_typeof(evidence_json) = 'object'::text);
ALTER TABLE "ai_governance"."reconciliation_differences" ADD CONSTRAINT "reconciliation_differences_severity_check" CHECK (severity::text = ANY (ARRAY['LOW'::character varying, 'MEDIUM'::character varying, 'HIGH'::character varying, 'CRITICAL'::character varying]::text[]));
ALTER TABLE "ai_governance"."reconciliation_differences" ADD CONSTRAINT "reconciliation_differences_status_check" CHECK (status::text = ANY (ARRAY['OPEN'::character varying, 'INVESTIGATING'::character varying, 'RESOLVED'::character varying, 'ACCEPTED_RISK'::character varying]::text[]));

-- ----------------------------
-- Primary Key structure for table reconciliation_differences
-- ----------------------------
ALTER TABLE "ai_governance"."reconciliation_differences" ADD CONSTRAINT "reconciliation_differences_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table reconciliation_items
-- ----------------------------
CREATE INDEX "ix_reconciliation_item_subject" ON "ai_governance"."reconciliation_items" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "subject_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "checked_at" "pg_catalog"."timestamptz_ops" DESC NULLS FIRST
);
CREATE UNIQUE INDEX "uq_reconciliation_item_target" ON "ai_governance"."reconciliation_items" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "reconciliation_run_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "check_code" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "target_type" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "target_id" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);

-- ----------------------------
-- Triggers structure for table reconciliation_items
-- ----------------------------
CREATE TRIGGER "trg_append_only" BEFORE UPDATE ON "ai_governance"."reconciliation_items"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_append_only_mutation"();
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."reconciliation_items"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();

-- ----------------------------
-- Checks structure for table reconciliation_items
-- ----------------------------
ALTER TABLE "ai_governance"."reconciliation_items" ADD CONSTRAINT "reconciliation_items_evidence_json_check" CHECK (jsonb_typeof(evidence_json) = 'object'::text);
ALTER TABLE "ai_governance"."reconciliation_items" ADD CONSTRAINT "reconciliation_items_result_check" CHECK (result::text = ANY (ARRAY['BALANCED'::character varying, 'DIFFERENCE'::character varying, 'SKIPPED'::character varying, 'ERROR'::character varying]::text[]));

-- ----------------------------
-- Primary Key structure for table reconciliation_items
-- ----------------------------
ALTER TABLE "ai_governance"."reconciliation_items" ADD CONSTRAINT "reconciliation_items_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table reconciliation_resolutions
-- ----------------------------
CREATE UNIQUE INDEX "uq_reconciliation_resolution_once" ON "ai_governance"."reconciliation_resolutions" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "reconciliation_difference_id" "pg_catalog"."uuid_ops" ASC NULLS LAST
);

-- ----------------------------
-- Triggers structure for table reconciliation_resolutions
-- ----------------------------
CREATE TRIGGER "trg_append_only" BEFORE UPDATE ON "ai_governance"."reconciliation_resolutions"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_append_only_mutation"();
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."reconciliation_resolutions"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();

-- ----------------------------
-- Checks structure for table reconciliation_resolutions
-- ----------------------------
ALTER TABLE "ai_governance"."reconciliation_resolutions" ADD CONSTRAINT "reconciliation_resolutions_check" CHECK ((resolution_action::text = ANY (ARRAY['ADJUST'::character varying, 'REVERSAL'::character varying]::text[])) AND correcting_transaction_id IS NOT NULL OR (resolution_action::text <> ALL (ARRAY['ADJUST'::character varying, 'REVERSAL'::character varying]::text[])));
ALTER TABLE "ai_governance"."reconciliation_resolutions" ADD CONSTRAINT "reconciliation_resolutions_evidence_json_check" CHECK (jsonb_typeof(evidence_json) = 'object'::text);
ALTER TABLE "ai_governance"."reconciliation_resolutions" ADD CONSTRAINT "reconciliation_resolutions_resolution_action_check" CHECK (resolution_action::text = ANY (ARRAY['ADJUST'::character varying, 'REVERSAL'::character varying, 'REPROCESS'::character varying, 'ACCEPT_RISK'::character varying, 'NO_ACTION_VALIDATED'::character varying]::text[]));

-- ----------------------------
-- Primary Key structure for table reconciliation_resolutions
-- ----------------------------
ALTER TABLE "ai_governance"."reconciliation_resolutions" ADD CONSTRAINT "reconciliation_resolutions_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table reconciliation_runs
-- ----------------------------
CREATE INDEX "ix_reconciliation_run_window" ON "ai_governance"."reconciliation_runs" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "reconciliation_type" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "window_start" "pg_catalog"."timestamptz_ops" ASC NULLS LAST,
  "window_end" "pg_catalog"."timestamptz_ops" ASC NULLS LAST,
  "status" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);
CREATE UNIQUE INDEX "uq_reconciliation_run_code" ON "ai_governance"."reconciliation_runs" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "run_code" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);

-- ----------------------------
-- Triggers structure for table reconciliation_runs
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."reconciliation_runs"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."reconciliation_runs"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();

-- ----------------------------
-- Checks structure for table reconciliation_runs
-- ----------------------------
ALTER TABLE "ai_governance"."reconciliation_runs" ADD CONSTRAINT "reconciliation_runs_check" CHECK (window_end > window_start);
ALTER TABLE "ai_governance"."reconciliation_runs" ADD CONSTRAINT "reconciliation_runs_check1" CHECK ((balanced_item_count + difference_item_count) <= total_item_count);
ALTER TABLE "ai_governance"."reconciliation_runs" ADD CONSTRAINT "reconciliation_runs_check2" CHECK (status::text = 'CLOSED'::text AND closed_at IS NOT NULL AND unresolved_difference_count = 0 OR status::text <> 'CLOSED'::text);
ALTER TABLE "ai_governance"."reconciliation_runs" ADD CONSTRAINT "reconciliation_runs_difference_item_count_check" CHECK (difference_item_count >= 0);
ALTER TABLE "ai_governance"."reconciliation_runs" ADD CONSTRAINT "reconciliation_runs_reconciliation_type_check" CHECK (reconciliation_type::text = ANY (ARRAY['INTERNAL_DAILY'::character varying, 'INTERNAL_MANUAL'::character varying, 'INTEGRITY_REPLAY'::character varying]::text[]));
ALTER TABLE "ai_governance"."reconciliation_runs" ADD CONSTRAINT "reconciliation_runs_row_version_check" CHECK (row_version > 0);
ALTER TABLE "ai_governance"."reconciliation_runs" ADD CONSTRAINT "reconciliation_runs_status_check" CHECK (status::text = ANY (ARRAY['OPEN'::character varying, 'RUNNING'::character varying, 'DIFFERENCE_FOUND'::character varying, 'BALANCED'::character varying, 'INVESTIGATING'::character varying, 'RESOLVED'::character varying, 'CLOSED'::character varying, 'FAILED'::character varying]::text[]));
ALTER TABLE "ai_governance"."reconciliation_runs" ADD CONSTRAINT "reconciliation_runs_total_item_count_check" CHECK (total_item_count >= 0);
ALTER TABLE "ai_governance"."reconciliation_runs" ADD CONSTRAINT "reconciliation_runs_balanced_item_count_check" CHECK (balanced_item_count >= 0);
ALTER TABLE "ai_governance"."reconciliation_runs" ADD CONSTRAINT "reconciliation_runs_unresolved_difference_count_check" CHECK (unresolved_difference_count >= 0);

-- ----------------------------
-- Primary Key structure for table reconciliation_runs
-- ----------------------------
ALTER TABLE "ai_governance"."reconciliation_runs" ADD CONSTRAINT "reconciliation_runs_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table route_policies
-- ----------------------------
CREATE UNIQUE INDEX "uq_route_policy_code_active" ON "ai_governance"."route_policies" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "policy_code" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
) WHERE is_deleted = false;

-- ----------------------------
-- Triggers structure for table route_policies
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."route_policies"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."route_policies"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();
CREATE TRIGGER "trg_validate_soft_delete" BEFORE INSERT OR UPDATE ON "ai_governance"."route_policies"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_validate_soft_delete"();

-- ----------------------------
-- Checks structure for table route_policies
-- ----------------------------
ALTER TABLE "ai_governance"."route_policies" ADD CONSTRAINT "route_policies_max_attempts_check" CHECK (max_attempts >= 1 AND max_attempts <= 8);
ALTER TABLE "ai_governance"."route_policies" ADD CONSTRAINT "route_policies_retry_before_first_content_only_check" CHECK (retry_before_first_content_only);
ALTER TABLE "ai_governance"."route_policies" ADD CONSTRAINT "route_policies_revision_check" CHECK (revision > 0);
ALTER TABLE "ai_governance"."route_policies" ADD CONSTRAINT "route_policies_row_version_check" CHECK (row_version > 0);
ALTER TABLE "ai_governance"."route_policies" ADD CONSTRAINT "route_policies_scope_type_check" CHECK (scope_type::text = ANY (ARRAY['TENANT'::character varying, 'ORGANIZATION'::character varying, 'PROJECT'::character varying, 'USER'::character varying]::text[]));
ALTER TABLE "ai_governance"."route_policies" ADD CONSTRAINT "route_policies_check" CHECK (is_deleted AND deleted_at IS NOT NULL AND deleted_by IS NOT NULL AND delete_reason IS NOT NULL OR NOT is_deleted AND deleted_at IS NULL AND deleted_by IS NULL AND delete_reason IS NULL);
ALTER TABLE "ai_governance"."route_policies" ADD CONSTRAINT "route_policies_status_check" CHECK (status::text = ANY (ARRAY['DRAFT'::character varying, 'REVIEWING'::character varying, 'APPROVED'::character varying, 'ACTIVE'::character varying, 'REVOKED'::character varying, 'RETIRED'::character varying]::text[]));

-- ----------------------------
-- Primary Key structure for table route_policies
-- ----------------------------
ALTER TABLE "ai_governance"."route_policies" ADD CONSTRAINT "route_policies_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table route_policy_candidates
-- ----------------------------
CREATE UNIQUE INDEX "uq_route_candidate_sequence_active" ON "ai_governance"."route_policy_candidates" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "route_policy_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "sequence_no" "pg_catalog"."int2_ops" ASC NULLS LAST
) WHERE is_deleted = false;

-- ----------------------------
-- Triggers structure for table route_policy_candidates
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."route_policy_candidates"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."route_policy_candidates"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();
CREATE TRIGGER "trg_validate_soft_delete" BEFORE INSERT OR UPDATE ON "ai_governance"."route_policy_candidates"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_validate_soft_delete"();

-- ----------------------------
-- Checks structure for table route_policy_candidates
-- ----------------------------
ALTER TABLE "ai_governance"."route_policy_candidates" ADD CONSTRAINT "route_policy_candidates_network_class_check" CHECK (network_class::text = ANY (ARRAY['INTERNAL'::character varying, 'EXTERNAL'::character varying]::text[]));
ALTER TABLE "ai_governance"."route_policy_candidates" ADD CONSTRAINT "route_policy_candidates_check" CHECK (is_deleted AND deleted_at IS NOT NULL AND deleted_by IS NOT NULL AND delete_reason IS NOT NULL OR NOT is_deleted AND deleted_at IS NULL AND deleted_by IS NULL AND delete_reason IS NULL);
ALTER TABLE "ai_governance"."route_policy_candidates" ADD CONSTRAINT "route_policy_candidates_row_version_check" CHECK (row_version > 0);
ALTER TABLE "ai_governance"."route_policy_candidates" ADD CONSTRAINT "route_policy_candidates_sequence_no_check" CHECK (sequence_no >= 1 AND sequence_no <= 64);
ALTER TABLE "ai_governance"."route_policy_candidates" ADD CONSTRAINT "route_policy_candidates_status_check" CHECK (status::text = ANY (ARRAY['DRAFT'::character varying, 'ACTIVE'::character varying, 'SUSPENDED'::character varying, 'RETIRED'::character varying]::text[]));
ALTER TABLE "ai_governance"."route_policy_candidates" ADD CONSTRAINT "route_policy_candidates_revision_check" CHECK (revision > 0);

-- ----------------------------
-- Primary Key structure for table route_policy_candidates
-- ----------------------------
ALTER TABLE "ai_governance"."route_policy_candidates" ADD CONSTRAINT "route_policy_candidates_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table runtime_snapshots
-- ----------------------------
CREATE UNIQUE INDEX "uq_runtime_snapshot_active" ON "ai_governance"."runtime_snapshots" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "config_domain" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
) WHERE status::text = 'ACTIVE'::text;
CREATE UNIQUE INDEX "uq_runtime_snapshot_business_id" ON "ai_governance"."runtime_snapshots" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "snapshot_id" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);
CREATE UNIQUE INDEX "uq_runtime_snapshot_revision" ON "ai_governance"."runtime_snapshots" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "config_domain" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "config_revision" "pg_catalog"."int8_ops" ASC NULLS LAST
);

-- ----------------------------
-- Triggers structure for table runtime_snapshots
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."runtime_snapshots"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."runtime_snapshots"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();

-- ----------------------------
-- Checks structure for table runtime_snapshots
-- ----------------------------
ALTER TABLE "ai_governance"."runtime_snapshots" ADD CONSTRAINT "runtime_snapshots_config_domain_check" CHECK (config_domain::text = ANY (ARRAY['IDENTITY'::character varying, 'ORGANIZATION'::character varying, 'PROJECT'::character varying, 'AUTHORIZATION'::character varying, 'NAVIGATION'::character varying, 'KEY_LIMIT'::character varying, 'MODEL_ACCESS'::character varying, 'ROUTING'::character varying, 'PROMPT_FIREWALL'::character varying, 'CREDIT_RATE'::character varying]::text[]));
ALTER TABLE "ai_governance"."runtime_snapshots" ADD CONSTRAINT "runtime_snapshots_config_revision_check" CHECK (config_revision > 0);
ALTER TABLE "ai_governance"."runtime_snapshots" ADD CONSTRAINT "runtime_snapshots_check" CHECK (expires_at IS NULL OR expires_at > effective_from);
ALTER TABLE "ai_governance"."runtime_snapshots" ADD CONSTRAINT "runtime_snapshots_row_version_check" CHECK (row_version > 0);
ALTER TABLE "ai_governance"."runtime_snapshots" ADD CONSTRAINT "runtime_snapshots_status_check" CHECK (status::text = ANY (ARRAY['BUILDING'::character varying, 'PUBLISHED'::character varying, 'ACTIVE'::character varying, 'SUPERSEDED'::character varying, 'REVOKED'::character varying, 'FAILED'::character varying]::text[]));
ALTER TABLE "ai_governance"."runtime_snapshots" ADD CONSTRAINT "runtime_snapshots_generation_check" CHECK (generation > 0);

-- ----------------------------
-- Primary Key structure for table runtime_snapshots
-- ----------------------------
ALTER TABLE "ai_governance"."runtime_snapshots" ADD CONSTRAINT "runtime_snapshots_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table safety_events
-- ----------------------------
CREATE INDEX "ix_safety_event_request" ON "ai_governance"."safety_events" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "request_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "occurred_at" "pg_catalog"."timestamptz_ops" ASC NULLS LAST
);
CREATE INDEX "ix_safety_events_occurred_brin" ON "ai_governance"."safety_events" USING brin (
  "occurred_at" "pg_catalog"."timestamptz_minmax_ops"
);

-- ----------------------------
-- Triggers structure for table safety_events
-- ----------------------------
CREATE TRIGGER "trg_append_only" BEFORE UPDATE ON "ai_governance"."safety_events"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_append_only_mutation"();
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."safety_events"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();

-- ----------------------------
-- Checks structure for table safety_events
-- ----------------------------
ALTER TABLE "ai_governance"."safety_events" ADD CONSTRAINT "safety_events_action_check" CHECK (action::text = ANY (ARRAY['BLOCK'::character varying, 'REDACT'::character varying, 'FORCE_INTERNAL'::character varying, 'ALERT'::character varying]::text[]));
ALTER TABLE "ai_governance"."safety_events" ADD CONSTRAINT "safety_events_direction_check" CHECK (direction::text = ANY (ARRAY['INPUT'::character varying, 'OUTPUT'::character varying]::text[]));
ALTER TABLE "ai_governance"."safety_events" ADD CONSTRAINT "safety_events_severity_check" CHECK (severity::text = ANY (ARRAY['LOW'::character varying, 'MEDIUM'::character varying, 'HIGH'::character varying, 'CRITICAL'::character varying]::text[]));

-- ----------------------------
-- Primary Key structure for table safety_events
-- ----------------------------
ALTER TABLE "ai_governance"."safety_events" ADD CONSTRAINT "safety_events_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Triggers structure for table schema_migration_contracts
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."schema_migration_contracts"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."schema_migration_contracts"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();

-- ----------------------------
-- Uniques structure for table schema_migration_contracts
-- ----------------------------
ALTER TABLE "ai_governance"."schema_migration_contracts" ADD CONSTRAINT "uq_schema_migration_version" UNIQUE ("tenant_id", "migration_version");

-- ----------------------------
-- Checks structure for table schema_migration_contracts
-- ----------------------------
ALTER TABLE "ai_governance"."schema_migration_contracts" ADD CONSTRAINT "schema_migration_contracts_execution_ms_check" CHECK (execution_ms >= 0);

-- ----------------------------
-- Primary Key structure for table schema_migration_contracts
-- ----------------------------
ALTER TABLE "ai_governance"."schema_migration_contracts" ADD CONSTRAINT "schema_migration_contracts_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table sys_access_policies
-- ----------------------------
CREATE UNIQUE INDEX "uq_sys_access_policy_active" ON "ai_governance"."sys_access_policies" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "policy_code" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
) WHERE is_deleted = false;

-- ----------------------------
-- Triggers structure for table sys_access_policies
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."sys_access_policies"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."sys_access_policies"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();
CREATE TRIGGER "trg_validate_soft_delete" BEFORE INSERT OR UPDATE ON "ai_governance"."sys_access_policies"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_validate_soft_delete"();

-- ----------------------------
-- Checks structure for table sys_access_policies
-- ----------------------------
ALTER TABLE "ai_governance"."sys_access_policies" ADD CONSTRAINT "sys_access_policies_check1" CHECK (is_deleted AND deleted_at IS NOT NULL AND deleted_by IS NOT NULL AND delete_reason IS NOT NULL OR NOT is_deleted AND deleted_at IS NULL AND deleted_by IS NULL AND delete_reason IS NULL);
ALTER TABLE "ai_governance"."sys_access_policies" ADD CONSTRAINT "sys_access_policies_effect_check" CHECK (effect::text = ANY (ARRAY['ALLOW'::character varying, 'DENY'::character varying]::text[]));
ALTER TABLE "ai_governance"."sys_access_policies" ADD CONSTRAINT "sys_access_policies_maximum_data_classification_check" CHECK (maximum_data_classification::text = ANY (ARRAY['PUBLIC'::character varying, 'INTERNAL'::character varying, 'CONFIDENTIAL'::character varying, 'RESTRICTED'::character varying]::text[]));
ALTER TABLE "ai_governance"."sys_access_policies" ADD CONSTRAINT "sys_access_policies_minimum_auth_strength_check" CHECK (minimum_auth_strength >= 0 AND minimum_auth_strength <= 9);
ALTER TABLE "ai_governance"."sys_access_policies" ADD CONSTRAINT "sys_access_policies_required_membership_type_check" CHECK (required_membership_type::text = ANY (ARRAY['NONE'::character varying, 'ORGANIZATION'::character varying, 'PROJECT'::character varying, 'ANY'::character varying]::text[]));
ALTER TABLE "ai_governance"."sys_access_policies" ADD CONSTRAINT "sys_access_policies_required_subject_status_check" CHECK (required_subject_status::text = 'ACTIVE'::text);
ALTER TABLE "ai_governance"."sys_access_policies" ADD CONSTRAINT "sys_access_policies_revision_check" CHECK (revision > 0);
ALTER TABLE "ai_governance"."sys_access_policies" ADD CONSTRAINT "sys_access_policies_row_version_check" CHECK (row_version > 0);
ALTER TABLE "ai_governance"."sys_access_policies" ADD CONSTRAINT "sys_access_policies_check" CHECK (valid_until IS NULL OR valid_until > valid_from);
ALTER TABLE "ai_governance"."sys_access_policies" ADD CONSTRAINT "sys_access_policies_status_check" CHECK (status::text = ANY (ARRAY['DRAFT'::character varying, 'REVIEWING'::character varying, 'APPROVED'::character varying, 'ACTIVE'::character varying, 'REVOKED'::character varying, 'RETIRED'::character varying]::text[]));

-- ----------------------------
-- Primary Key structure for table sys_access_policies
-- ----------------------------
ALTER TABLE "ai_governance"."sys_access_policies" ADD CONSTRAINT "sys_access_policies_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table sys_access_policy_bindings
-- ----------------------------
CREATE INDEX "ix_sys_policy_binding_lookup" ON "ai_governance"."sys_access_policy_bindings" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "action_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "scope_type" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "scope_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "status" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "priority" "pg_catalog"."int4_ops" ASC NULLS LAST
);

-- ----------------------------
-- Triggers structure for table sys_access_policy_bindings
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."sys_access_policy_bindings"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."sys_access_policy_bindings"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();
CREATE TRIGGER "trg_validate_soft_delete" BEFORE INSERT OR UPDATE ON "ai_governance"."sys_access_policy_bindings"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_validate_soft_delete"();

-- ----------------------------
-- Checks structure for table sys_access_policy_bindings
-- ----------------------------
ALTER TABLE "ai_governance"."sys_access_policy_bindings" ADD CONSTRAINT "sys_access_policy_bindings_check1" CHECK (is_deleted AND deleted_at IS NOT NULL AND deleted_by IS NOT NULL AND delete_reason IS NOT NULL OR NOT is_deleted AND deleted_at IS NULL AND deleted_by IS NULL AND delete_reason IS NULL);
ALTER TABLE "ai_governance"."sys_access_policy_bindings" ADD CONSTRAINT "sys_access_policy_bindings_priority_check" CHECK (priority >= 0 AND priority <= 100000);
ALTER TABLE "ai_governance"."sys_access_policy_bindings" ADD CONSTRAINT "sys_access_policy_bindings_revision_check" CHECK (revision > 0);
ALTER TABLE "ai_governance"."sys_access_policy_bindings" ADD CONSTRAINT "sys_access_policy_bindings_row_version_check" CHECK (row_version > 0);
ALTER TABLE "ai_governance"."sys_access_policy_bindings" ADD CONSTRAINT "sys_access_policy_bindings_scope_type_check" CHECK (scope_type::text = ANY (ARRAY['TENANT'::character varying, 'ORGANIZATION'::character varying, 'PROJECT'::character varying, 'USER'::character varying]::text[]));
ALTER TABLE "ai_governance"."sys_access_policy_bindings" ADD CONSTRAINT "sys_access_policy_bindings_check" CHECK (valid_until IS NULL OR valid_until > valid_from);
ALTER TABLE "ai_governance"."sys_access_policy_bindings" ADD CONSTRAINT "sys_access_policy_bindings_status_check" CHECK (status::text = ANY (ARRAY['DRAFT'::character varying, 'ACTIVE'::character varying, 'REVOKED'::character varying, 'EXPIRED'::character varying]::text[]));

-- ----------------------------
-- Primary Key structure for table sys_access_policy_bindings
-- ----------------------------
ALTER TABLE "ai_governance"."sys_access_policy_bindings" ADD CONSTRAINT "sys_access_policy_bindings_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table sys_action_catalogs
-- ----------------------------
CREATE UNIQUE INDEX "uq_sys_action_code_active" ON "ai_governance"."sys_action_catalogs" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "action_code" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
) WHERE is_deleted = false;

-- ----------------------------
-- Triggers structure for table sys_action_catalogs
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."sys_action_catalogs"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."sys_action_catalogs"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();
CREATE TRIGGER "trg_validate_soft_delete" BEFORE INSERT OR UPDATE ON "ai_governance"."sys_action_catalogs"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_validate_soft_delete"();

-- ----------------------------
-- Checks structure for table sys_action_catalogs
-- ----------------------------
ALTER TABLE "ai_governance"."sys_action_catalogs" ADD CONSTRAINT "sys_action_catalogs_check" CHECK (action_code::text = ((resource_type_code::text || ':'::text) || verb_code::text));
ALTER TABLE "ai_governance"."sys_action_catalogs" ADD CONSTRAINT "sys_action_catalogs_check1" CHECK (is_deleted AND deleted_at IS NOT NULL AND deleted_by IS NOT NULL AND delete_reason IS NOT NULL OR NOT is_deleted AND deleted_at IS NULL AND deleted_by IS NULL AND delete_reason IS NULL);
ALTER TABLE "ai_governance"."sys_action_catalogs" ADD CONSTRAINT "sys_action_catalogs_revision_check" CHECK (revision > 0);
ALTER TABLE "ai_governance"."sys_action_catalogs" ADD CONSTRAINT "sys_action_catalogs_risk_level_check" CHECK (risk_level::text = ANY (ARRAY['LOW'::character varying, 'MEDIUM'::character varying, 'HIGH'::character varying, 'CRITICAL'::character varying]::text[]));
ALTER TABLE "ai_governance"."sys_action_catalogs" ADD CONSTRAINT "sys_action_catalogs_row_version_check" CHECK (row_version > 0);
ALTER TABLE "ai_governance"."sys_action_catalogs" ADD CONSTRAINT "sys_action_catalogs_action_code_check" CHECK (action_code::text !~ '[*?]'::text);
ALTER TABLE "ai_governance"."sys_action_catalogs" ADD CONSTRAINT "sys_action_catalogs_status_check" CHECK (status::text = ANY (ARRAY['DRAFT'::character varying, 'ACTIVE'::character varying, 'DEPRECATED'::character varying, 'RETIRED'::character varying]::text[]));

-- ----------------------------
-- Primary Key structure for table sys_action_catalogs
-- ----------------------------
ALTER TABLE "ai_governance"."sys_action_catalogs" ADD CONSTRAINT "sys_action_catalogs_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table sys_role_permissions
-- ----------------------------
CREATE UNIQUE INDEX "uq_sys_role_permission_active" ON "ai_governance"."sys_role_permissions" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "role_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "action_id" "pg_catalog"."uuid_ops" ASC NULLS LAST
) WHERE is_deleted = false AND status::text = 'ACTIVE'::text;

-- ----------------------------
-- Triggers structure for table sys_role_permissions
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."sys_role_permissions"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."sys_role_permissions"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();
CREATE TRIGGER "trg_validate_soft_delete" BEFORE INSERT OR UPDATE ON "ai_governance"."sys_role_permissions"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_validate_soft_delete"();

-- ----------------------------
-- Checks structure for table sys_role_permissions
-- ----------------------------
ALTER TABLE "ai_governance"."sys_role_permissions" ADD CONSTRAINT "sys_role_permissions_check1" CHECK (is_deleted AND deleted_at IS NOT NULL AND deleted_by IS NOT NULL AND delete_reason IS NOT NULL OR NOT is_deleted AND deleted_at IS NULL AND deleted_by IS NULL AND delete_reason IS NULL);
ALTER TABLE "ai_governance"."sys_role_permissions" ADD CONSTRAINT "sys_role_permissions_effect_check" CHECK (effect::text = ANY (ARRAY['ALLOW'::character varying, 'DENY'::character varying]::text[]));
ALTER TABLE "ai_governance"."sys_role_permissions" ADD CONSTRAINT "sys_role_permissions_check" CHECK (valid_until IS NULL OR valid_until > valid_from);
ALTER TABLE "ai_governance"."sys_role_permissions" ADD CONSTRAINT "sys_role_permissions_revision_check" CHECK (revision > 0);
ALTER TABLE "ai_governance"."sys_role_permissions" ADD CONSTRAINT "sys_role_permissions_row_version_check" CHECK (row_version > 0);
ALTER TABLE "ai_governance"."sys_role_permissions" ADD CONSTRAINT "sys_role_permissions_status_check" CHECK (status::text = ANY (ARRAY['DRAFT'::character varying, 'ACTIVE'::character varying, 'REVOKED'::character varying]::text[]));

-- ----------------------------
-- Primary Key structure for table sys_role_permissions
-- ----------------------------
ALTER TABLE "ai_governance"."sys_role_permissions" ADD CONSTRAINT "sys_role_permissions_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table sys_roles
-- ----------------------------
CREATE UNIQUE INDEX "uq_sys_role_code_active" ON "ai_governance"."sys_roles" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "role_code" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
) WHERE is_deleted = false;
CREATE UNIQUE INDEX "uq_sys_role_domain_mapping_active" ON "ai_governance"."sys_roles" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "role_domain" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "source_role_code" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
) WHERE is_deleted = false AND role_source_type::text = 'DOMAIN_MAPPING'::text;

-- ----------------------------
-- Triggers structure for table sys_roles
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."sys_roles"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."sys_roles"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();
CREATE TRIGGER "trg_validate_soft_delete" BEFORE INSERT OR UPDATE ON "ai_governance"."sys_roles"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_validate_soft_delete"();

-- ----------------------------
-- Checks structure for table sys_roles
-- ----------------------------
ALTER TABLE "ai_governance"."sys_roles" ADD CONSTRAINT "sys_roles_check1" CHECK (is_deleted AND deleted_at IS NOT NULL AND deleted_by IS NOT NULL AND delete_reason IS NOT NULL OR NOT is_deleted AND deleted_at IS NULL AND deleted_by IS NULL AND delete_reason IS NULL);
ALTER TABLE "ai_governance"."sys_roles" ADD CONSTRAINT "sys_roles_revision_check" CHECK (revision > 0);
ALTER TABLE "ai_governance"."sys_roles" ADD CONSTRAINT "sys_roles_role_domain_check" CHECK (role_domain::text = ANY (ARRAY['PLATFORM'::character varying, 'ENTERPRISE'::character varying, 'ORGANIZATION'::character varying, 'PROJECT'::character varying]::text[]));
ALTER TABLE "ai_governance"."sys_roles" ADD CONSTRAINT "sys_roles_role_source_type_check" CHECK (role_source_type::text = ANY (ARRAY['PLATFORM_LOCAL'::character varying, 'DOMAIN_MAPPING'::character varying]::text[]));
ALTER TABLE "ai_governance"."sys_roles" ADD CONSTRAINT "sys_roles_row_version_check" CHECK (row_version > 0);
ALTER TABLE "ai_governance"."sys_roles" ADD CONSTRAINT "sys_roles_check" CHECK (role_source_type::text = 'PLATFORM_LOCAL'::text AND role_domain::text = 'PLATFORM'::text AND source_role_code IS NULL OR role_source_type::text = 'DOMAIN_MAPPING'::text AND source_role_code IS NOT NULL);
ALTER TABLE "ai_governance"."sys_roles" ADD CONSTRAINT "sys_roles_status_check" CHECK (status::text = ANY (ARRAY['DRAFT'::character varying, 'ACTIVE'::character varying, 'SUSPENDED'::character varying, 'RETIRED'::character varying]::text[]));

-- ----------------------------
-- Primary Key structure for table sys_roles
-- ----------------------------
ALTER TABLE "ai_governance"."sys_roles" ADD CONSTRAINT "sys_roles_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table sys_subject_role_bindings
-- ----------------------------
CREATE INDEX "ix_sys_subject_roles" ON "ai_governance"."sys_subject_role_bindings" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "subject_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "scope_type" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "scope_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "status" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);
CREATE UNIQUE INDEX "uq_sys_subject_role_binding_active" ON "ai_governance"."sys_subject_role_bindings" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "subject_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "role_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "scope_type" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "scope_id" "pg_catalog"."uuid_ops" ASC NULLS LAST
) WHERE is_deleted = false AND (status::text = ANY (ARRAY['PENDING'::character varying, 'ACTIVE'::character varying, 'SUSPENDED'::character varying]::text[]));

-- ----------------------------
-- Triggers structure for table sys_subject_role_bindings
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."sys_subject_role_bindings"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."sys_subject_role_bindings"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();
CREATE TRIGGER "trg_validate_soft_delete" BEFORE INSERT OR UPDATE ON "ai_governance"."sys_subject_role_bindings"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_validate_soft_delete"();

-- ----------------------------
-- Checks structure for table sys_subject_role_bindings
-- ----------------------------
ALTER TABLE "ai_governance"."sys_subject_role_bindings" ADD CONSTRAINT "sys_subject_role_bindings_check" CHECK (valid_until IS NULL OR valid_until > valid_from);
ALTER TABLE "ai_governance"."sys_subject_role_bindings" ADD CONSTRAINT "sys_subject_role_bindings_check1" CHECK (is_deleted AND deleted_at IS NOT NULL AND deleted_by IS NOT NULL AND delete_reason IS NOT NULL OR NOT is_deleted AND deleted_at IS NULL AND deleted_by IS NULL AND delete_reason IS NULL);
ALTER TABLE "ai_governance"."sys_subject_role_bindings" ADD CONSTRAINT "sys_subject_role_bindings_revision_check" CHECK (revision > 0);
ALTER TABLE "ai_governance"."sys_subject_role_bindings" ADD CONSTRAINT "sys_subject_role_bindings_row_version_check" CHECK (row_version > 0);
ALTER TABLE "ai_governance"."sys_subject_role_bindings" ADD CONSTRAINT "sys_subject_role_bindings_scope_type_check" CHECK (scope_type::text = ANY (ARRAY['TENANT'::character varying, 'ORGANIZATION'::character varying, 'PROJECT'::character varying, 'USER'::character varying]::text[]));
ALTER TABLE "ai_governance"."sys_subject_role_bindings" ADD CONSTRAINT "sys_subject_role_bindings_binding_source_check" CHECK (binding_source::text = 'PLATFORM_LOCAL'::text);
ALTER TABLE "ai_governance"."sys_subject_role_bindings" ADD CONSTRAINT "sys_subject_role_bindings_status_check" CHECK (status::text = ANY (ARRAY['PENDING'::character varying, 'ACTIVE'::character varying, 'SUSPENDED'::character varying, 'REVOKED'::character varying, 'EXPIRED'::character varying]::text[]));

-- ----------------------------
-- Primary Key structure for table sys_subject_role_bindings
-- ----------------------------
ALTER TABLE "ai_governance"."sys_subject_role_bindings" ADD CONSTRAINT "sys_subject_role_bindings_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table sys_ui_action_bindings
-- ----------------------------
CREATE UNIQUE INDEX "uq_sys_ui_action_binding_active" ON "ai_governance"."sys_ui_action_bindings" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "target_type" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "target_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "action_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "binding_purpose" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
) WHERE is_deleted = false;

-- ----------------------------
-- Triggers structure for table sys_ui_action_bindings
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."sys_ui_action_bindings"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."sys_ui_action_bindings"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();
CREATE TRIGGER "trg_validate_soft_delete" BEFORE INSERT OR UPDATE ON "ai_governance"."sys_ui_action_bindings"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_validate_soft_delete"();

-- ----------------------------
-- Checks structure for table sys_ui_action_bindings
-- ----------------------------
ALTER TABLE "ai_governance"."sys_ui_action_bindings" ADD CONSTRAINT "sys_ui_action_bindings_check" CHECK (is_deleted AND deleted_at IS NOT NULL AND deleted_by IS NOT NULL AND delete_reason IS NOT NULL OR NOT is_deleted AND deleted_at IS NULL AND deleted_by IS NULL AND delete_reason IS NULL);
ALTER TABLE "ai_governance"."sys_ui_action_bindings" ADD CONSTRAINT "sys_ui_action_bindings_match_mode_check" CHECK (match_mode::text = ANY (ARRAY['ANY'::character varying, 'ALL'::character varying]::text[]));
ALTER TABLE "ai_governance"."sys_ui_action_bindings" ADD CONSTRAINT "sys_ui_action_bindings_revision_check" CHECK (revision > 0);
ALTER TABLE "ai_governance"."sys_ui_action_bindings" ADD CONSTRAINT "sys_ui_action_bindings_row_version_check" CHECK (row_version > 0);
ALTER TABLE "ai_governance"."sys_ui_action_bindings" ADD CONSTRAINT "sys_ui_action_bindings_status_check" CHECK (status::text = ANY (ARRAY['DRAFT'::character varying, 'ACTIVE'::character varying, 'REVOKED'::character varying]::text[]));
ALTER TABLE "ai_governance"."sys_ui_action_bindings" ADD CONSTRAINT "sys_ui_action_bindings_binding_purpose_check" CHECK (binding_purpose::text = ANY (ARRAY['VISIBLE'::character varying, 'ENTER'::character varying]::text[]));
ALTER TABLE "ai_governance"."sys_ui_action_bindings" ADD CONSTRAINT "sys_ui_action_bindings_target_type_check" CHECK (target_type::text = ANY (ARRAY['MENU'::character varying, 'ROUTE'::character varying]::text[]));

-- ----------------------------
-- Primary Key structure for table sys_ui_action_bindings
-- ----------------------------
ALTER TABLE "ai_governance"."sys_ui_action_bindings" ADD CONSTRAINT "sys_ui_action_bindings_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table sys_ui_menus
-- ----------------------------
CREATE INDEX "ix_sys_ui_menu_tree" ON "ai_governance"."sys_ui_menus" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "parent_menu_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "status" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "sort_order" "pg_catalog"."int4_ops" ASC NULLS LAST
);
CREATE UNIQUE INDEX "uq_sys_ui_menu_code_active" ON "ai_governance"."sys_ui_menus" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "menu_code" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
) WHERE is_deleted = false;

-- ----------------------------
-- Triggers structure for table sys_ui_menus
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."sys_ui_menus"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."sys_ui_menus"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();
CREATE TRIGGER "trg_validate_soft_delete" BEFORE INSERT OR UPDATE ON "ai_governance"."sys_ui_menus"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_validate_soft_delete"();

-- ----------------------------
-- Checks structure for table sys_ui_menus
-- ----------------------------
ALTER TABLE "ai_governance"."sys_ui_menus" ADD CONSTRAINT "sys_ui_menus_check1" CHECK (is_deleted AND deleted_at IS NOT NULL AND deleted_by IS NOT NULL AND delete_reason IS NOT NULL OR NOT is_deleted AND deleted_at IS NULL AND deleted_by IS NULL AND delete_reason IS NULL);
ALTER TABLE "ai_governance"."sys_ui_menus" ADD CONSTRAINT "sys_ui_menus_icon_key_check" CHECK (icon_key IS NULL OR icon_key::text !~ '[./\\]'::text);
ALTER TABLE "ai_governance"."sys_ui_menus" ADD CONSTRAINT "sys_ui_menus_menu_type_check" CHECK (menu_type::text = ANY (ARRAY['DIRECTORY'::character varying, 'MENU'::character varying, 'LINK'::character varying]::text[]));
ALTER TABLE "ai_governance"."sys_ui_menus" ADD CONSTRAINT "sys_ui_menus_revision_check" CHECK (revision > 0);
ALTER TABLE "ai_governance"."sys_ui_menus" ADD CONSTRAINT "sys_ui_menus_row_version_check" CHECK (row_version > 0);
ALTER TABLE "ai_governance"."sys_ui_menus" ADD CONSTRAINT "sys_ui_menus_check" CHECK (menu_type::text = 'DIRECTORY'::text AND route_id IS NULL AND external_url IS NULL OR menu_type::text = 'MENU'::text AND route_id IS NOT NULL AND external_url IS NULL OR menu_type::text = 'LINK'::text AND route_id IS NULL AND external_url IS NOT NULL);
ALTER TABLE "ai_governance"."sys_ui_menus" ADD CONSTRAINT "sys_ui_menus_status_check" CHECK (status::text = ANY (ARRAY['DRAFT'::character varying, 'ACTIVE'::character varying, 'HIDDEN'::character varying, 'RETIRED'::character varying]::text[]));

-- ----------------------------
-- Primary Key structure for table sys_ui_menus
-- ----------------------------
ALTER TABLE "ai_governance"."sys_ui_menus" ADD CONSTRAINT "sys_ui_menus_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table sys_ui_routes
-- ----------------------------
CREATE UNIQUE INDEX "uq_sys_ui_route_code_active" ON "ai_governance"."sys_ui_routes" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "route_code" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
) WHERE is_deleted = false;
CREATE UNIQUE INDEX "uq_sys_ui_route_path_active" ON "ai_governance"."sys_ui_routes" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "route_path" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
) WHERE is_deleted = false;

-- ----------------------------
-- Triggers structure for table sys_ui_routes
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."sys_ui_routes"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."sys_ui_routes"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();
CREATE TRIGGER "trg_validate_soft_delete" BEFORE INSERT OR UPDATE ON "ai_governance"."sys_ui_routes"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_validate_soft_delete"();

-- ----------------------------
-- Checks structure for table sys_ui_routes
-- ----------------------------
ALTER TABLE "ai_governance"."sys_ui_routes" ADD CONSTRAINT "sys_ui_routes_component_key_check" CHECK (component_key::text !~ '[./\\]'::text);
ALTER TABLE "ai_governance"."sys_ui_routes" ADD CONSTRAINT "sys_ui_routes_layout_key_check" CHECK (layout_key::text !~ '[./\\]'::text);
ALTER TABLE "ai_governance"."sys_ui_routes" ADD CONSTRAINT "sys_ui_routes_revision_check" CHECK (revision > 0);
ALTER TABLE "ai_governance"."sys_ui_routes" ADD CONSTRAINT "sys_ui_routes_route_path_check" CHECK (route_path::text ~~ '/%'::text);
ALTER TABLE "ai_governance"."sys_ui_routes" ADD CONSTRAINT "sys_ui_routes_row_version_check" CHECK (row_version > 0);
ALTER TABLE "ai_governance"."sys_ui_routes" ADD CONSTRAINT "sys_ui_routes_check" CHECK (is_deleted AND deleted_at IS NOT NULL AND deleted_by IS NOT NULL AND delete_reason IS NOT NULL OR NOT is_deleted AND deleted_at IS NULL AND deleted_by IS NULL AND delete_reason IS NULL);
ALTER TABLE "ai_governance"."sys_ui_routes" ADD CONSTRAINT "sys_ui_routes_status_check" CHECK (status::text = ANY (ARRAY['DRAFT'::character varying, 'ACTIVE'::character varying, 'HIDDEN'::character varying, 'RETIRED'::character varying]::text[]));

-- ----------------------------
-- Primary Key structure for table sys_ui_routes
-- ----------------------------
ALTER TABLE "ai_governance"."sys_ui_routes" ADD CONSTRAINT "sys_ui_routes_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table tenants
-- ----------------------------
CREATE UNIQUE INDEX "uq_tenants_code_active" ON "ai_governance"."tenants" USING btree (
  "tenant_code" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
) WHERE is_deleted = false;

-- ----------------------------
-- Triggers structure for table tenants
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."tenants"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."tenants"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();
CREATE TRIGGER "trg_validate_soft_delete" BEFORE INSERT OR UPDATE ON "ai_governance"."tenants"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_validate_soft_delete"();

-- ----------------------------
-- Checks structure for table tenants
-- ----------------------------
ALTER TABLE "ai_governance"."tenants" ADD CONSTRAINT "tenants_check1" CHECK (is_deleted AND deleted_at IS NOT NULL AND deleted_by IS NOT NULL AND delete_reason IS NOT NULL OR NOT is_deleted AND deleted_at IS NULL AND deleted_by IS NULL AND delete_reason IS NULL);
ALTER TABLE "ai_governance"."tenants" ADD CONSTRAINT "tenants_operation_source_check" CHECK (operation_source::text = ANY (ARRAY['ADMIN'::character varying, 'API'::character varying, 'SYNC'::character varying, 'SYSTEM'::character varying, 'MIGRATION'::character varying]::text[]));
ALTER TABLE "ai_governance"."tenants" ADD CONSTRAINT "tenants_check" CHECK (tenant_id = id);
ALTER TABLE "ai_governance"."tenants" ADD CONSTRAINT "tenants_revision_check" CHECK (revision > 0);
ALTER TABLE "ai_governance"."tenants" ADD CONSTRAINT "tenants_row_version_check" CHECK (row_version > 0);
ALTER TABLE "ai_governance"."tenants" ADD CONSTRAINT "tenants_status_check" CHECK (status::text = ANY (ARRAY['DRAFT'::character varying, 'ACTIVE'::character varying, 'SUSPENDED'::character varying, 'RETIRED'::character varying]::text[]));

-- ----------------------------
-- Primary Key structure for table tenants
-- ----------------------------
ALTER TABLE "ai_governance"."tenants" ADD CONSTRAINT "tenants_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table usage_attempts
-- ----------------------------
CREATE INDEX "ix_usage_attempt_provider_request" ON "ai_governance"."usage_attempts" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "provider_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "provider_request_id" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);
CREATE INDEX "ix_usage_attempt_subject_time" ON "ai_governance"."usage_attempts" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "subject_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "started_at" "pg_catalog"."timestamptz_ops" DESC NULLS FIRST
);
CREATE INDEX "ix_usage_attempts_created_brin" ON "ai_governance"."usage_attempts" USING brin (
  "created_at" "pg_catalog"."timestamptz_minmax_ops"
);
CREATE UNIQUE INDEX "uq_usage_attempt_no" ON "ai_governance"."usage_attempts" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "request_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "attempt_no" "pg_catalog"."int2_ops" ASC NULLS LAST
);

-- ----------------------------
-- Triggers structure for table usage_attempts
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."usage_attempts"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."usage_attempts"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();

-- ----------------------------
-- Checks structure for table usage_attempts
-- ----------------------------
ALTER TABLE "ai_governance"."usage_attempts" ADD CONSTRAINT "usage_attempts_check" CHECK (total_tokens IS NULL OR input_tokens IS NULL OR output_tokens IS NULL OR total_tokens = (input_tokens + output_tokens));
ALTER TABLE "ai_governance"."usage_attempts" ADD CONSTRAINT "usage_attempts_check1" CHECK (first_content_at IS NULL OR first_content_at >= started_at);
ALTER TABLE "ai_governance"."usage_attempts" ADD CONSTRAINT "usage_attempts_check2" CHECK (finished_at IS NULL OR finished_at >= started_at);
ALTER TABLE "ai_governance"."usage_attempts" ADD CONSTRAINT "usage_attempts_error_class_check" CHECK (error_class IS NULL OR (error_class::text = ANY (ARRAY['AUTH'::character varying, 'RATE_LIMIT'::character varying, 'TIMEOUT'::character varying, 'NETWORK'::character varying, 'UPSTREAM_4XX'::character varying, 'UPSTREAM_5XX'::character varying, 'CONTENT'::character varying, 'CANCELLED'::character varying, 'UNKNOWN'::character varying]::text[])));
ALTER TABLE "ai_governance"."usage_attempts" ADD CONSTRAINT "usage_attempts_input_tokens_check" CHECK (input_tokens IS NULL OR input_tokens >= 0);
ALTER TABLE "ai_governance"."usage_attempts" ADD CONSTRAINT "usage_attempts_output_tokens_check" CHECK (output_tokens IS NULL OR output_tokens >= 0);
ALTER TABLE "ai_governance"."usage_attempts" ADD CONSTRAINT "usage_attempts_row_version_check" CHECK (row_version > 0);
ALTER TABLE "ai_governance"."usage_attempts" ADD CONSTRAINT "usage_attempts_status_check" CHECK (status::text = ANY (ARRAY['PLANNED'::character varying, 'SENT'::character varying, 'STREAMING'::character varying, 'SUCCEEDED'::character varying, 'FAILED'::character varying, 'CANCELLED'::character varying, 'UNKNOWN'::character varying]::text[]));
ALTER TABLE "ai_governance"."usage_attempts" ADD CONSTRAINT "usage_attempts_attempt_no_check" CHECK (attempt_no >= 1 AND attempt_no <= 64);
ALTER TABLE "ai_governance"."usage_attempts" ADD CONSTRAINT "usage_attempts_total_tokens_check" CHECK (total_tokens IS NULL OR total_tokens >= 0);

-- ----------------------------
-- Primary Key structure for table usage_attempts
-- ----------------------------
ALTER TABLE "ai_governance"."usage_attempts" ADD CONSTRAINT "usage_attempts_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table usage_events
-- ----------------------------
CREATE INDEX "ix_usage_event_key_time" ON "ai_governance"."usage_events" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "key_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "occurred_at" "pg_catalog"."timestamptz_ops" DESC NULLS FIRST
);
CREATE INDEX "ix_usage_event_scope_time" ON "ai_governance"."usage_events" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "billing_scope_type" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "billing_scope_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "occurred_at" "pg_catalog"."timestamptz_ops" DESC NULLS FIRST
);
CREATE INDEX "ix_usage_event_subject_time" ON "ai_governance"."usage_events" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "subject_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "occurred_at" "pg_catalog"."timestamptz_ops" DESC NULLS FIRST
);
CREATE INDEX "ix_usage_events_occurred_brin" ON "ai_governance"."usage_events" USING brin (
  "occurred_at" "pg_catalog"."timestamptz_minmax_ops"
);
CREATE UNIQUE INDEX "uq_usage_event_business_id" ON "ai_governance"."usage_events" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "usage_event_id" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);
CREATE UNIQUE INDEX "uq_usage_event_request" ON "ai_governance"."usage_events" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "request_id" "pg_catalog"."uuid_ops" ASC NULLS LAST
);

-- ----------------------------
-- Triggers structure for table usage_events
-- ----------------------------
CREATE TRIGGER "trg_append_only" BEFORE UPDATE ON "ai_governance"."usage_events"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_append_only_mutation"();
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."usage_events"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();

-- ----------------------------
-- Checks structure for table usage_events
-- ----------------------------
ALTER TABLE "ai_governance"."usage_events" ADD CONSTRAINT "usage_events_check" CHECK (total_tokens = (input_tokens + output_tokens));
ALTER TABLE "ai_governance"."usage_events" ADD CONSTRAINT "usage_events_credit_rate_revision_check" CHECK (credit_rate_revision > 0);
ALTER TABLE "ai_governance"."usage_events" ADD CONSTRAINT "usage_events_input_tokens_check" CHECK (input_tokens >= 0);
ALTER TABLE "ai_governance"."usage_events" ADD CONSTRAINT "usage_events_key_revision_check" CHECK (key_revision > 0);
ALTER TABLE "ai_governance"."usage_events" ADD CONSTRAINT "usage_events_measurement_status_check" CHECK (measurement_status::text = ANY (ARRAY['MEASURED'::character varying, 'RECONCILED'::character varying]::text[]));
ALTER TABLE "ai_governance"."usage_events" ADD CONSTRAINT "usage_events_output_tokens_check" CHECK (output_tokens >= 0);
ALTER TABLE "ai_governance"."usage_events" ADD CONSTRAINT "usage_events_settled_credit_amount_check" CHECK (settled_credit_amount >= 0);
ALTER TABLE "ai_governance"."usage_events" ADD CONSTRAINT "usage_events_billing_scope_type_check" CHECK (billing_scope_type::text = ANY (ARRAY['ORGANIZATION'::character varying, 'PROJECT'::character varying]::text[]));
ALTER TABLE "ai_governance"."usage_events" ADD CONSTRAINT "usage_events_usage_source_check" CHECK (usage_source::text = ANY (ARRAY['PROVIDER'::character varying, 'GATEWAY'::character varying, 'RECONCILIATION'::character varying]::text[]));

-- ----------------------------
-- Primary Key structure for table usage_events
-- ----------------------------
ALTER TABLE "ai_governance"."usage_events" ADD CONSTRAINT "usage_events_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table usage_requests
-- ----------------------------
CREATE INDEX "ix_usage_request_key_time" ON "ai_governance"."usage_requests" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "key_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "created_at" "pg_catalog"."timestamptz_ops" DESC NULLS FIRST
);
CREATE INDEX "ix_usage_request_scope_time" ON "ai_governance"."usage_requests" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "billing_scope_type" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "billing_scope_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "created_at" "pg_catalog"."timestamptz_ops" DESC NULLS FIRST
);
CREATE INDEX "ix_usage_request_subject_time" ON "ai_governance"."usage_requests" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "subject_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "created_at" "pg_catalog"."timestamptz_ops" DESC NULLS FIRST
);
CREATE INDEX "ix_usage_requests_created_brin" ON "ai_governance"."usage_requests" USING brin (
  "created_at" "pg_catalog"."timestamptz_minmax_ops"
);
CREATE UNIQUE INDEX "uq_usage_request_business_id" ON "ai_governance"."usage_requests" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "request_id" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);
CREATE UNIQUE INDEX "uq_usage_request_idempotency" ON "ai_governance"."usage_requests" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "key_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "idempotency_key" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);

-- ----------------------------
-- Triggers structure for table usage_requests
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."usage_requests"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."usage_requests"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();

-- ----------------------------
-- Checks structure for table usage_requests
-- ----------------------------
ALTER TABLE "ai_governance"."usage_requests" ADD CONSTRAINT "usage_requests_billing_scope_type_check" CHECK (billing_scope_type::text = ANY (ARRAY['ORGANIZATION'::character varying, 'PROJECT'::character varying]::text[]));
ALTER TABLE "ai_governance"."usage_requests" ADD CONSTRAINT "usage_requests_declared_data_classification_check" CHECK (declared_data_classification::text = ANY (ARRAY['PUBLIC'::character varying, 'INTERNAL'::character varying, 'CONFIDENTIAL'::character varying, 'RESTRICTED'::character varying]::text[]));
ALTER TABLE "ai_governance"."usage_requests" ADD CONSTRAINT "usage_requests_effective_data_classification_check" CHECK (effective_data_classification::text = ANY (ARRAY['PUBLIC'::character varying, 'INTERNAL'::character varying, 'CONFIDENTIAL'::character varying, 'RESTRICTED'::character varying]::text[]));
ALTER TABLE "ai_governance"."usage_requests" ADD CONSTRAINT "usage_requests_estimated_input_tokens_check" CHECK (estimated_input_tokens >= 0);
ALTER TABLE "ai_governance"."usage_requests" ADD CONSTRAINT "usage_requests_external_context_json_check" CHECK (jsonb_typeof(external_context_json) = 'object'::text);
ALTER TABLE "ai_governance"."usage_requests" ADD CONSTRAINT "usage_requests_firewall_policy_revision_check" CHECK (firewall_policy_revision > 0);
ALTER TABLE "ai_governance"."usage_requests" ADD CONSTRAINT "usage_requests_key_revision_check" CHECK (key_revision > 0);
ALTER TABLE "ai_governance"."usage_requests" ADD CONSTRAINT "usage_requests_maximum_output_tokens_check" CHECK (maximum_output_tokens > 0);
ALTER TABLE "ai_governance"."usage_requests" ADD CONSTRAINT "usage_requests_model_policy_revision_check" CHECK (model_policy_revision > 0);
ALTER TABLE "ai_governance"."usage_requests" ADD CONSTRAINT "usage_requests_route_policy_revision_check" CHECK (route_policy_revision > 0);
ALTER TABLE "ai_governance"."usage_requests" ADD CONSTRAINT "usage_requests_row_version_check" CHECK (row_version > 0);
ALTER TABLE "ai_governance"."usage_requests" ADD CONSTRAINT "usage_requests_status_check" CHECK (status::text = ANY (ARRAY['CREATED'::character varying, 'RESERVED'::character varying, 'IN_FLIGHT'::character varying, 'CANCELLING'::character varying, 'SETTLED'::character varying, 'RELEASED'::character varying, 'RECONCILING'::character varying, 'CANCELLED'::character varying, 'FAILED'::character varying]::text[]));
ALTER TABLE "ai_governance"."usage_requests" ADD CONSTRAINT "usage_requests_subject_identity_revision_check" CHECK (subject_identity_revision IS NULL OR subject_identity_revision > 0);
ALTER TABLE "ai_governance"."usage_requests" ADD CONSTRAINT "usage_requests_authorization_revision_check" CHECK (authorization_revision > 0);
ALTER TABLE "ai_governance"."usage_requests" ADD CONSTRAINT "usage_requests_subject_type_check" CHECK (subject_type::text = ANY (ARRAY['USER'::character varying, 'SERVICE'::character varying]::text[]));

-- ----------------------------
-- Primary Key structure for table usage_requests
-- ----------------------------
ALTER TABLE "ai_governance"."usage_requests" ADD CONSTRAINT "usage_requests_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table user_allocations
-- ----------------------------
CREATE UNIQUE INDEX "uq_user_allocation_code_active" ON "ai_governance"."user_allocations" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "allocation_code" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
) WHERE is_deleted = false;
CREATE UNIQUE INDEX "uq_user_allocation_scope_active" ON "ai_governance"."user_allocations" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "subject_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "billing_scope_type" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "billing_scope_id" "pg_catalog"."uuid_ops" ASC NULLS LAST
) WHERE is_deleted = false AND status::text <> 'CLOSED'::text;

-- ----------------------------
-- Triggers structure for table user_allocations
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."user_allocations"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."user_allocations"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();
CREATE TRIGGER "trg_validate_soft_delete" BEFORE INSERT OR UPDATE ON "ai_governance"."user_allocations"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_validate_soft_delete"();

-- ----------------------------
-- Checks structure for table user_allocations
-- ----------------------------
ALTER TABLE "ai_governance"."user_allocations" ADD CONSTRAINT "user_allocations_check" CHECK (available_account_id <> reserved_account_id);
ALTER TABLE "ai_governance"."user_allocations" ADD CONSTRAINT "user_allocations_check1" CHECK (valid_until IS NULL OR valid_until > valid_from);
ALTER TABLE "ai_governance"."user_allocations" ADD CONSTRAINT "user_allocations_check2" CHECK (is_deleted AND deleted_at IS NOT NULL AND deleted_by IS NOT NULL AND delete_reason IS NOT NULL OR NOT is_deleted AND deleted_at IS NULL AND deleted_by IS NULL AND delete_reason IS NULL);
ALTER TABLE "ai_governance"."user_allocations" ADD CONSTRAINT "user_allocations_revision_check" CHECK (revision > 0);
ALTER TABLE "ai_governance"."user_allocations" ADD CONSTRAINT "user_allocations_row_version_check" CHECK (row_version > 0);
ALTER TABLE "ai_governance"."user_allocations" ADD CONSTRAINT "user_allocations_billing_scope_type_check" CHECK (billing_scope_type::text = ANY (ARRAY['ORGANIZATION'::character varying, 'PROJECT'::character varying]::text[]));
ALTER TABLE "ai_governance"."user_allocations" ADD CONSTRAINT "user_allocations_status_check" CHECK (status::text = ANY (ARRAY['PENDING'::character varying, 'ACTIVE'::character varying, 'FROZEN'::character varying, 'CLOSED'::character varying]::text[]));

-- ----------------------------
-- Primary Key structure for table user_allocations
-- ----------------------------
ALTER TABLE "ai_governance"."user_allocations" ADD CONSTRAINT "user_allocations_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table user_api_key_secret_versions
-- ----------------------------
CREATE UNIQUE INDEX "uq_user_api_key_secret_prefix_active" ON "ai_governance"."user_api_key_secret_versions" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "key_prefix" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
) WHERE status::text = ANY (ARRAY['ACTIVE'::character varying, 'ROTATING'::character varying]::text[]);
CREATE UNIQUE INDEX "uq_user_api_key_secret_version" ON "ai_governance"."user_api_key_secret_versions" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "key_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "secret_version" "pg_catalog"."int4_ops" ASC NULLS LAST
);

-- ----------------------------
-- Triggers structure for table user_api_key_secret_versions
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."user_api_key_secret_versions"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."user_api_key_secret_versions"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();

-- ----------------------------
-- Checks structure for table user_api_key_secret_versions
-- ----------------------------
ALTER TABLE "ai_governance"."user_api_key_secret_versions" ADD CONSTRAINT "user_api_key_secret_versions_check" CHECK (valid_until > valid_from);
ALTER TABLE "ai_governance"."user_api_key_secret_versions" ADD CONSTRAINT "user_api_key_secret_versions_check1" CHECK (status::text = 'REVOKED'::text AND revoked_at IS NOT NULL AND revoke_reason IS NOT NULL OR status::text <> 'REVOKED'::text);
ALTER TABLE "ai_governance"."user_api_key_secret_versions" ADD CONSTRAINT "user_api_key_secret_versions_row_version_check" CHECK (row_version > 0);
ALTER TABLE "ai_governance"."user_api_key_secret_versions" ADD CONSTRAINT "user_api_key_secret_versions_secret_version_check" CHECK (secret_version > 0);
ALTER TABLE "ai_governance"."user_api_key_secret_versions" ADD CONSTRAINT "user_api_key_secret_versions_status_check" CHECK (status::text = ANY (ARRAY['PENDING'::character varying, 'ACTIVE'::character varying, 'ROTATING'::character varying, 'REVOKED'::character varying, 'EXPIRED'::character varying]::text[]));

-- ----------------------------
-- Primary Key structure for table user_api_key_secret_versions
-- ----------------------------
ALTER TABLE "ai_governance"."user_api_key_secret_versions" ADD CONSTRAINT "user_api_key_secret_versions_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table user_api_keys
-- ----------------------------
CREATE INDEX "ix_user_api_key_allocation" ON "ai_governance"."user_api_keys" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "user_allocation_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "status" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);
CREATE INDEX "ix_user_api_key_owner_scope" ON "ai_governance"."user_api_keys" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "owner_subject_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "billing_scope_type" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "billing_scope_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "status" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);
CREATE UNIQUE INDEX "uq_user_api_key_code_active" ON "ai_governance"."user_api_keys" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "key_code" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
) WHERE is_deleted = false;

-- ----------------------------
-- Triggers structure for table user_api_keys
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."user_api_keys"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."user_api_keys"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();
CREATE TRIGGER "trg_validate_soft_delete" BEFORE INSERT OR UPDATE ON "ai_governance"."user_api_keys"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_validate_soft_delete"();

-- ----------------------------
-- Checks structure for table user_api_keys
-- ----------------------------
ALTER TABLE "ai_governance"."user_api_keys" ADD CONSTRAINT "user_api_keys_billing_scope_type_check" CHECK (billing_scope_type::text = ANY (ARRAY['ORGANIZATION'::character varying, 'PROJECT'::character varying]::text[]));
ALTER TABLE "ai_governance"."user_api_keys" ADD CONSTRAINT "user_api_keys_check" CHECK (expires_at > issued_at);
ALTER TABLE "ai_governance"."user_api_keys" ADD CONSTRAINT "user_api_keys_check1" CHECK (is_deleted AND deleted_at IS NOT NULL AND deleted_by IS NOT NULL AND delete_reason IS NOT NULL OR NOT is_deleted AND deleted_at IS NULL AND deleted_by IS NULL AND delete_reason IS NULL);
ALTER TABLE "ai_governance"."user_api_keys" ADD CONSTRAINT "user_api_keys_key_revision_check" CHECK (key_revision > 0);
ALTER TABLE "ai_governance"."user_api_keys" ADD CONSTRAINT "user_api_keys_row_version_check" CHECK (row_version > 0);
ALTER TABLE "ai_governance"."user_api_keys" ADD CONSTRAINT "user_api_keys_allowed_actions_check" CHECK (cardinality(allowed_actions) > 0);
ALTER TABLE "ai_governance"."user_api_keys" ADD CONSTRAINT "user_api_keys_status_check" CHECK (status::text = ANY (ARRAY['PENDING'::character varying, 'ACTIVE'::character varying, 'SUSPENDED'::character varying, 'REVOKED'::character varying, 'EXPIRED'::character varying]::text[]));

-- ----------------------------
-- Primary Key structure for table user_api_keys
-- ----------------------------
ALTER TABLE "ai_governance"."user_api_keys" ADD CONSTRAINT "user_api_keys_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table user_closing_snapshot_items
-- ----------------------------
CREATE INDEX "ix_user_closing_subject_window" ON "ai_governance"."user_closing_snapshot_items" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "subject_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "window_end" "pg_catalog"."timestamptz_ops" DESC NULLS FIRST
);
CREATE UNIQUE INDEX "uq_user_closing_allocation" ON "ai_governance"."user_closing_snapshot_items" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "closing_snapshot_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "user_allocation_id" "pg_catalog"."uuid_ops" ASC NULLS LAST
);

-- ----------------------------
-- Triggers structure for table user_closing_snapshot_items
-- ----------------------------
CREATE TRIGGER "trg_append_only" BEFORE UPDATE ON "ai_governance"."user_closing_snapshot_items"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_append_only_mutation"();
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."user_closing_snapshot_items"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();

-- ----------------------------
-- Checks structure for table user_closing_snapshot_items
-- ----------------------------
ALTER TABLE "ai_governance"."user_closing_snapshot_items" ADD CONSTRAINT "user_closing_snapshot_items_available_credit_check" CHECK (available_credit >= 0);
ALTER TABLE "ai_governance"."user_closing_snapshot_items" ADD CONSTRAINT "user_closing_snapshot_items_billing_scope_type_check" CHECK (billing_scope_type::text = ANY (ARRAY['ORGANIZATION'::character varying, 'PROJECT'::character varying]::text[]));
ALTER TABLE "ai_governance"."user_closing_snapshot_items" ADD CONSTRAINT "user_closing_snapshot_items_check" CHECK (window_end > window_start);
ALTER TABLE "ai_governance"."user_closing_snapshot_items" ADD CONSTRAINT "user_closing_snapshot_items_consumed_credit_check" CHECK (consumed_credit >= 0);
ALTER TABLE "ai_governance"."user_closing_snapshot_items" ADD CONSTRAINT "user_closing_snapshot_items_emergency_drawn_credit_check" CHECK (emergency_drawn_credit >= 0);
ALTER TABLE "ai_governance"."user_closing_snapshot_items" ADD CONSTRAINT "user_closing_snapshot_items_open_reservation_count_check" CHECK (open_reservation_count >= 0);
ALTER TABLE "ai_governance"."user_closing_snapshot_items" ADD CONSTRAINT "user_closing_snapshot_items_reserved_credit_check" CHECK (reserved_credit >= 0);
ALTER TABLE "ai_governance"."user_closing_snapshot_items" ADD CONSTRAINT "user_closing_snapshot_items_settled_tokens_check" CHECK (settled_tokens >= 0);
ALTER TABLE "ai_governance"."user_closing_snapshot_items" ADD CONSTRAINT "user_closing_snapshot_items_active_key_count_check" CHECK (active_key_count >= 0);
ALTER TABLE "ai_governance"."user_closing_snapshot_items" ADD CONSTRAINT "user_closing_snapshot_items_unresolved_difference_count_check" CHECK (unresolved_difference_count = 0);

-- ----------------------------
-- Primary Key structure for table user_closing_snapshot_items
-- ----------------------------
ALTER TABLE "ai_governance"."user_closing_snapshot_items" ADD CONSTRAINT "user_closing_snapshot_items_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table user_contact_points
-- ----------------------------
CREATE INDEX "ix_user_contact_subject" ON "ai_governance"."user_contact_points" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "subject_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "status" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);
CREATE UNIQUE INDEX "uq_user_contact_lookup_active" ON "ai_governance"."user_contact_points" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "contact_type" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "normalized_lookup_hmac" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
) WHERE is_deleted = false AND status::text <> 'REVOKED'::text;
CREATE UNIQUE INDEX "uq_user_primary_contact_active" ON "ai_governance"."user_contact_points" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "subject_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "contact_type" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
) WHERE is_deleted = false AND is_primary = true AND status::text = 'ACTIVE'::text;

-- ----------------------------
-- Triggers structure for table user_contact_points
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."user_contact_points"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."user_contact_points"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();
CREATE TRIGGER "trg_validate_soft_delete" BEFORE INSERT OR UPDATE ON "ai_governance"."user_contact_points"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_validate_soft_delete"();

-- ----------------------------
-- Checks structure for table user_contact_points
-- ----------------------------
ALTER TABLE "ai_governance"."user_contact_points" ADD CONSTRAINT "user_contact_points_check1" CHECK (is_deleted AND deleted_at IS NOT NULL AND deleted_by IS NOT NULL AND delete_reason IS NOT NULL OR NOT is_deleted AND deleted_at IS NULL AND deleted_by IS NULL AND delete_reason IS NULL);
ALTER TABLE "ai_governance"."user_contact_points" ADD CONSTRAINT "user_contact_points_contact_type_check" CHECK (contact_type::text = ANY (ARRAY['EMAIL'::character varying, 'PHONE'::character varying]::text[]));
ALTER TABLE "ai_governance"."user_contact_points" ADD CONSTRAINT "user_contact_points_revision_check" CHECK (revision > 0);
ALTER TABLE "ai_governance"."user_contact_points" ADD CONSTRAINT "user_contact_points_row_version_check" CHECK (row_version > 0);
ALTER TABLE "ai_governance"."user_contact_points" ADD CONSTRAINT "user_contact_points_status_check" CHECK (status::text = ANY (ARRAY['ACTIVE'::character varying, 'SUSPENDED'::character varying, 'REVOKED'::character varying]::text[]));
ALTER TABLE "ai_governance"."user_contact_points" ADD CONSTRAINT "user_contact_points_check" CHECK (verification_status::text = 'VERIFIED'::text AND verified_at IS NOT NULL OR verification_status::text <> 'VERIFIED'::text);
ALTER TABLE "ai_governance"."user_contact_points" ADD CONSTRAINT "user_contact_points_verification_status_check" CHECK (verification_status::text = ANY (ARRAY['UNVERIFIED'::character varying, 'PENDING'::character varying, 'VERIFIED'::character varying, 'REVOKED'::character varying]::text[]));

-- ----------------------------
-- Primary Key structure for table user_contact_points
-- ----------------------------
ALTER TABLE "ai_governance"."user_contact_points" ADD CONSTRAINT "user_contact_points_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table user_governance_projections
-- ----------------------------
CREATE INDEX "ix_user_governance_primary_org" ON "ai_governance"."user_governance_projections" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "primary_organization_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "projection_status" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);
CREATE UNIQUE INDEX "uq_user_governance_projection" ON "ai_governance"."user_governance_projections" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "subject_id" "pg_catalog"."uuid_ops" ASC NULLS LAST
);

-- ----------------------------
-- Triggers structure for table user_governance_projections
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."user_governance_projections"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."user_governance_projections"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();

-- ----------------------------
-- Checks structure for table user_governance_projections
-- ----------------------------
ALTER TABLE "ai_governance"."user_governance_projections" ADD CONSTRAINT "user_governance_projections_active_key_count_check" CHECK (active_key_count >= 0);
ALTER TABLE "ai_governance"."user_governance_projections" ADD CONSTRAINT "user_governance_projections_active_organization_count_check" CHECK (active_organization_count >= 0);
ALTER TABLE "ai_governance"."user_governance_projections" ADD CONSTRAINT "user_governance_projections_active_project_count_check" CHECK (active_project_count >= 0);
ALTER TABLE "ai_governance"."user_governance_projections" ADD CONSTRAINT "user_governance_projections_available_credit_check" CHECK (available_credit >= 0);
ALTER TABLE "ai_governance"."user_governance_projections" ADD CONSTRAINT "user_governance_projections_current_period_tokens_check" CHECK (current_period_tokens >= 0);
ALTER TABLE "ai_governance"."user_governance_projections" ADD CONSTRAINT "user_governance_projections_effective_authorization_revis_check" CHECK (effective_authorization_revision > 0);
ALTER TABLE "ai_governance"."user_governance_projections" ADD CONSTRAINT "user_governance_projections_generation_check" CHECK (generation > 0);
ALTER TABLE "ai_governance"."user_governance_projections" ADD CONSTRAINT "user_governance_projections_identity_revision_check" CHECK (identity_revision > 0);
ALTER TABLE "ai_governance"."user_governance_projections" ADD CONSTRAINT "user_governance_projections_projection_status_check" CHECK (projection_status::text = ANY (ARRAY['BUILDING'::character varying, 'CURRENT'::character varying, 'STALE'::character varying, 'FAILED'::character varying]::text[]));
ALTER TABLE "ai_governance"."user_governance_projections" ADD CONSTRAINT "user_governance_projections_reserved_credit_check" CHECK (reserved_credit >= 0);
ALTER TABLE "ai_governance"."user_governance_projections" ADD CONSTRAINT "user_governance_projections_runtime_generation_check" CHECK (runtime_generation > 0);
ALTER TABLE "ai_governance"."user_governance_projections" ADD CONSTRAINT "user_governance_projections_settled_credit_check" CHECK (settled_credit >= 0);
ALTER TABLE "ai_governance"."user_governance_projections" ADD CONSTRAINT "user_governance_projections_active_allocation_count_check" CHECK (active_allocation_count >= 0);
ALTER TABLE "ai_governance"."user_governance_projections" ADD CONSTRAINT "user_governance_projections_unresolved_difference_count_check" CHECK (unresolved_difference_count >= 0);

-- ----------------------------
-- Primary Key structure for table user_governance_projections
-- ----------------------------
ALTER TABLE "ai_governance"."user_governance_projections" ADD CONSTRAINT "user_governance_projections_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table user_login_identities
-- ----------------------------
CREATE INDEX "ix_login_identity_subject" ON "ai_governance"."user_login_identities" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "subject_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "status" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);
CREATE UNIQUE INDEX "uq_login_identity_contact_active" ON "ai_governance"."user_login_identities" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "auth_method" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "contact_point_id" "pg_catalog"."uuid_ops" ASC NULLS LAST
) WHERE is_deleted = false AND contact_point_id IS NOT NULL AND status::text <> 'REVOKED'::text;
CREATE UNIQUE INDEX "uq_login_identity_provider_active" ON "ai_governance"."user_login_identities" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "oauth_provider_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "provider_subject_hmac" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
) WHERE is_deleted = false AND oauth_provider_id IS NOT NULL AND status::text <> 'REVOKED'::text;

-- ----------------------------
-- Triggers structure for table user_login_identities
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."user_login_identities"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."user_login_identities"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();
CREATE TRIGGER "trg_validate_soft_delete" BEFORE INSERT OR UPDATE ON "ai_governance"."user_login_identities"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_validate_soft_delete"();

-- ----------------------------
-- Checks structure for table user_login_identities
-- ----------------------------
ALTER TABLE "ai_governance"."user_login_identities" ADD CONSTRAINT "user_login_identities_check" CHECK (auth_method::text = 'ENTERPRISE_SSO'::text AND issuer_config_id IS NOT NULL OR auth_method::text <> 'ENTERPRISE_SSO'::text);
ALTER TABLE "ai_governance"."user_login_identities" ADD CONSTRAINT "user_login_identities_check1" CHECK ((auth_method::text = ANY (ARRAY['PASSWORD'::character varying, 'EMAIL_OTP'::character varying, 'SMS_OTP'::character varying]::text[])) AND contact_point_id IS NOT NULL OR (auth_method::text <> ALL (ARRAY['PASSWORD'::character varying, 'EMAIL_OTP'::character varying, 'SMS_OTP'::character varying]::text[])));
ALTER TABLE "ai_governance"."user_login_identities" ADD CONSTRAINT "user_login_identities_check2" CHECK ((auth_method::text = ANY (ARRAY['WECHAT_OAUTH'::character varying, 'OIDC_OAUTH'::character varying]::text[])) AND oauth_provider_id IS NOT NULL AND provider_subject_hmac IS NOT NULL OR (auth_method::text <> ALL (ARRAY['WECHAT_OAUTH'::character varying, 'OIDC_OAUTH'::character varying]::text[])));
ALTER TABLE "ai_governance"."user_login_identities" ADD CONSTRAINT "user_login_identities_check3" CHECK (status::text = 'LOCKED'::text AND locked_until IS NOT NULL OR status::text <> 'LOCKED'::text);
ALTER TABLE "ai_governance"."user_login_identities" ADD CONSTRAINT "user_login_identities_check4" CHECK (is_deleted AND deleted_at IS NOT NULL AND deleted_by IS NOT NULL AND delete_reason IS NOT NULL OR NOT is_deleted AND deleted_at IS NULL AND deleted_by IS NULL AND delete_reason IS NULL);
ALTER TABLE "ai_governance"."user_login_identities" ADD CONSTRAINT "user_login_identities_failed_attempt_count_check" CHECK (failed_attempt_count >= 0);
ALTER TABLE "ai_governance"."user_login_identities" ADD CONSTRAINT "user_login_identities_identity_revision_check" CHECK (identity_revision > 0);
ALTER TABLE "ai_governance"."user_login_identities" ADD CONSTRAINT "user_login_identities_row_version_check" CHECK (row_version > 0);
ALTER TABLE "ai_governance"."user_login_identities" ADD CONSTRAINT "user_login_identities_auth_method_check" CHECK (auth_method::text = ANY (ARRAY['ENTERPRISE_SSO'::character varying, 'PASSWORD'::character varying, 'EMAIL_OTP'::character varying, 'SMS_OTP'::character varying, 'WECHAT_OAUTH'::character varying, 'OIDC_OAUTH'::character varying]::text[]));
ALTER TABLE "ai_governance"."user_login_identities" ADD CONSTRAINT "user_login_identities_status_check" CHECK (status::text = ANY (ARRAY['PENDING'::character varying, 'ACTIVE'::character varying, 'LOCKED'::character varying, 'REVOKED'::character varying]::text[]));

-- ----------------------------
-- Primary Key structure for table user_login_identities
-- ----------------------------
ALTER TABLE "ai_governance"."user_login_identities" ADD CONSTRAINT "user_login_identities_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table user_password_credentials
-- ----------------------------
CREATE INDEX "ix_password_credential_subject" ON "ai_governance"."user_password_credentials" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "subject_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "status" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);
CREATE UNIQUE INDEX "uq_password_credential_identity_active" ON "ai_governance"."user_password_credentials" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "login_identity_id" "pg_catalog"."uuid_ops" ASC NULLS LAST
) WHERE is_deleted = false AND (status::text = ANY (ARRAY['ACTIVE'::character varying, 'RESET_REQUIRED'::character varying]::text[]));

-- ----------------------------
-- Triggers structure for table user_password_credentials
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."user_password_credentials"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."user_password_credentials"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();
CREATE TRIGGER "trg_validate_soft_delete" BEFORE INSERT OR UPDATE ON "ai_governance"."user_password_credentials"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_validate_soft_delete"();

-- ----------------------------
-- Checks structure for table user_password_credentials
-- ----------------------------
ALTER TABLE "ai_governance"."user_password_credentials" ADD CONSTRAINT "user_password_credentials_check" CHECK (expires_at IS NULL OR expires_at > changed_at);
ALTER TABLE "ai_governance"."user_password_credentials" ADD CONSTRAINT "user_password_credentials_check1" CHECK (is_deleted AND deleted_at IS NOT NULL AND deleted_by IS NOT NULL AND delete_reason IS NOT NULL OR NOT is_deleted AND deleted_at IS NULL AND deleted_by IS NULL AND delete_reason IS NULL);
ALTER TABLE "ai_governance"."user_password_credentials" ADD CONSTRAINT "user_password_credentials_hash_parameters_json_check" CHECK (jsonb_typeof(hash_parameters_json) = 'object'::text);
ALTER TABLE "ai_governance"."user_password_credentials" ADD CONSTRAINT "user_password_credentials_password_revision_check" CHECK (password_revision > 0);
ALTER TABLE "ai_governance"."user_password_credentials" ADD CONSTRAINT "user_password_credentials_row_version_check" CHECK (row_version > 0);
ALTER TABLE "ai_governance"."user_password_credentials" ADD CONSTRAINT "user_password_credentials_algorithm_check" CHECK (algorithm::text = 'ARGON2ID'::text);
ALTER TABLE "ai_governance"."user_password_credentials" ADD CONSTRAINT "user_password_credentials_status_check" CHECK (status::text = ANY (ARRAY['ACTIVE'::character varying, 'RESET_REQUIRED'::character varying, 'REVOKED'::character varying, 'EXPIRED'::character varying]::text[]));

-- ----------------------------
-- Primary Key structure for table user_password_credentials
-- ----------------------------
ALTER TABLE "ai_governance"."user_password_credentials" ADD CONSTRAINT "user_password_credentials_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table user_profiles
-- ----------------------------
CREATE UNIQUE INDEX "uq_user_profile_subject_active" ON "ai_governance"."user_profiles" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "subject_id" "pg_catalog"."uuid_ops" ASC NULLS LAST
) WHERE is_deleted = false;

-- ----------------------------
-- Triggers structure for table user_profiles
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."user_profiles"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."user_profiles"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();
CREATE TRIGGER "trg_validate_soft_delete" BEFORE INSERT OR UPDATE ON "ai_governance"."user_profiles"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_validate_soft_delete"();

-- ----------------------------
-- Checks structure for table user_profiles
-- ----------------------------
ALTER TABLE "ai_governance"."user_profiles" ADD CONSTRAINT "user_profiles_check" CHECK (is_deleted AND deleted_at IS NOT NULL AND deleted_by IS NOT NULL AND delete_reason IS NOT NULL OR NOT is_deleted AND deleted_at IS NULL AND deleted_by IS NULL AND delete_reason IS NULL);
ALTER TABLE "ai_governance"."user_profiles" ADD CONSTRAINT "user_profiles_profile_revision_check" CHECK (profile_revision > 0);
ALTER TABLE "ai_governance"."user_profiles" ADD CONSTRAINT "user_profiles_row_version_check" CHECK (row_version > 0);

-- ----------------------------
-- Primary Key structure for table user_profiles
-- ----------------------------
ALTER TABLE "ai_governance"."user_profiles" ADD CONSTRAINT "user_profiles_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table user_sessions
-- ----------------------------
CREATE INDEX "ix_user_session_subject_status" ON "ai_governance"."user_sessions" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "subject_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "status" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "expires_at" "pg_catalog"."timestamptz_ops" ASC NULLS LAST
);
CREATE UNIQUE INDEX "uq_user_session_access_jti" ON "ai_governance"."user_sessions" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "access_token_jti_hash" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);
CREATE UNIQUE INDEX "uq_user_session_business_id" ON "ai_governance"."user_sessions" USING btree (
  "tenant_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "session_id" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);

-- ----------------------------
-- Triggers structure for table user_sessions
-- ----------------------------
CREATE TRIGGER "trg_no_physical_delete" BEFORE DELETE ON "ai_governance"."user_sessions"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_reject_physical_delete"();
CREATE TRIGGER "trg_touch_updated_at" BEFORE UPDATE ON "ai_governance"."user_sessions"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_touch_updated_at"();
CREATE TRIGGER "trg_validate_soft_delete" BEFORE INSERT OR UPDATE ON "ai_governance"."user_sessions"
FOR EACH ROW
EXECUTE PROCEDURE "ai_governance"."fn_validate_soft_delete"();

-- ----------------------------
-- Checks structure for table user_sessions
-- ----------------------------
ALTER TABLE "ai_governance"."user_sessions" ADD CONSTRAINT "user_sessions_auth_method_check" CHECK (auth_method::text = ANY (ARRAY['ENTERPRISE_SSO'::character varying, 'PASSWORD'::character varying, 'EMAIL_OTP'::character varying, 'SMS_OTP'::character varying, 'WECHAT_OAUTH'::character varying, 'OIDC_OAUTH'::character varying]::text[]));
ALTER TABLE "ai_governance"."user_sessions" ADD CONSTRAINT "user_sessions_check" CHECK (expires_at > issued_at);
ALTER TABLE "ai_governance"."user_sessions" ADD CONSTRAINT "user_sessions_check1" CHECK (last_seen_at >= issued_at);
ALTER TABLE "ai_governance"."user_sessions" ADD CONSTRAINT "user_sessions_check2" CHECK (status::text = 'REVOKED'::text AND revoked_at IS NOT NULL AND revoke_reason IS NOT NULL OR status::text <> 'REVOKED'::text);
ALTER TABLE "ai_governance"."user_sessions" ADD CONSTRAINT "user_sessions_check3" CHECK (is_deleted AND deleted_at IS NOT NULL AND deleted_by IS NOT NULL AND delete_reason IS NOT NULL OR NOT is_deleted AND deleted_at IS NULL AND deleted_by IS NULL AND delete_reason IS NULL);
ALTER TABLE "ai_governance"."user_sessions" ADD CONSTRAINT "user_sessions_row_version_check" CHECK (row_version > 0);
ALTER TABLE "ai_governance"."user_sessions" ADD CONSTRAINT "user_sessions_session_revision_check" CHECK (session_revision > 0);
ALTER TABLE "ai_governance"."user_sessions" ADD CONSTRAINT "user_sessions_assurance_level_check" CHECK (assurance_level::text = ANY (ARRAY['LOW'::character varying, 'MEDIUM'::character varying, 'HIGH'::character varying]::text[]));
ALTER TABLE "ai_governance"."user_sessions" ADD CONSTRAINT "user_sessions_status_check" CHECK (status::text = ANY (ARRAY['ACTIVE'::character varying, 'REVOKED'::character varying, 'EXPIRED'::character varying]::text[]));

-- ----------------------------
-- Primary Key structure for table user_sessions
-- ----------------------------
ALTER TABLE "ai_governance"."user_sessions" ADD CONSTRAINT "user_sessions_pkey" PRIMARY KEY ("id");
