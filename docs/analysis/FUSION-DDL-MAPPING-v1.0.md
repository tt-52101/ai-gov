# AI-GOV 三方融合 DDL 映射与剪裁分析
> 日期: 2026-07-31 | 输出: `schema/ai-gov-fusion-minimal.sql` (29 表)

---

## 一、三方 Schema 全景对比

| 维度 | TokenHub | AxonHub | ai-gov.sql (参考) | **融合输出** |
|------|----------|---------|-------------------|-------------|
| 表数量 | 30 | 20 | 69 | **29** |
| 数据库 | SQLite | SQLite | PostgreSQL | **SQLite (可迁 PG)** |
| 主键风格 | TEXT (UUID) | INTEGER AUTOINCREMENT | UUID v7 | **TEXT** |
| 软删除 | 无 | deleted_at int | is_deleted + 4件套 | **无（简化）** |
| 设计哲学 | 务实轻量 | 务实轻量 | 过度工程化 | **务实轻量** |

---

## 二、逐表融合映射（TokenHub 为主体，AxonHub 吸收，ai-gov 去臃肿）

### 第1组: 用户与身份 (2 表)

| # | 融合表 | TokenHub 来源 | AxonHub 来源 | ai-gov 裁减 |
|---|--------|--------------|-------------|------------|
| 1 | `users` | `admin_users` 字段 | `users` + `oidc_identities` 字段 | 9表→1表（去掉 identity_issuer_configs, identity_subjects, user_login_identities, user_password_credentials, user_contact_points, oauth_provider_configs, authentication_challenges, authentication_events, user_profiles） |
| 2 | `admin_sessions` | `admin_sessions` | — | `user_sessions` 简化 |

### 第2组: Party 统一主体 (3 表 — PRD 新增)

| # | 融合表 | TokenHub 来源 | AxonHub 来源 | ai-gov 裁减 |
|---|--------|--------------|-------------|------------|
| 3 | `parties` | `projects` 字段 | `projects` 字段 | `organizations` + `projects` → 1表。去掉 organization_path/depth/type/leader 拆分，type 枚举 org/project |
| 4 | `party_edges` | — | — | 新增。去掉 organization_memberships/project_memberships 双表 → 统一 `party_members` |
| 5 | `party_members` | `project_teams` | `user_projects` | 2表→1表 |

### 第3组: 资金治理 (5 表 — PRD 新增)

| # | 融合表 | TokenHub 来源 | AxonHub 来源 | ai-gov 裁减 |
|---|--------|--------------|-------------|------------|
| 6 | `accounts` | — | — | `funding_accounts` + `balance_projections` + `budget_*` → 1表。去掉 6 种 account_type、CQRS 投影、emergency_* 2表 |
| 7 | `ledgers` | — | — | `ledger_transactions` + `ledger_legs` → 1表。去掉复式分录 14 种 entry_type，direction 枚举替代 |
| 8 | `freezes` | — | — | `quota_reservations` 简化 |
| 9 | `allocations` | — | — | 新增 |
| 10 | `liquidations` | — | — | 新增 |

### 第4组: API Key (1 表 — 三向融合)

| # | 融合表 | TokenHub 来源 | AxonHub 来源 | ai-gov 裁减 |
|---|--------|--------------|-------------|------------|
| 11 | `api_keys` | 全部保留 + 限额字段 | `type/scopes/profiles/allowed_ips` | 去掉 user_api_key_secret_versions（key_hash 自身足够）、key_limit_policies 进 api_keys 热字段、key_usage_counter_projections 用 quota_buckets 替代 |

### 第5组: 模型与供应商 (4 表)

| # | 融合表 | TokenHub 来源 | AxonHub 来源 | ai-gov 裁减 |
|---|--------|--------------|-------------|------------|
| 12 | `providers` | `providers` 全部保留 | `channels` 字段吸收 | `model_providers` + `model_channels` → 1表。去掉 provider_credential_refs |
| 13 | `provider_resources` | 全部保留 | — | 去掉 model_channel_health_events（用 channel_probes 替代） |
| 14 | `models` | 全部保留 | `developer/icon/model_group/model_card/settings/ax_model_type` | `model_catalog_entries` → 合并。去掉 data_classification_ceiling/network_class（热字段保留） |
| 15 | `provider_models` | 全部保留 | — | 保留 |

### 第6组: 定价与路由 (3 表)

| # | 融合表 | TokenHub 来源 | AxonHub 来源 | ai-gov 裁减 |
|---|--------|--------------|-------------|------------|
| 16 | `model_prices` | — | `channel_model_prices` + `channel_model_price_versions` | `credit_rate_versions` → 替换。2表→1表（version 用 reference_id） |
| 17 | `model_routes` | 全部保留 | — | `route_policies` + `route_policy_candidates` → 合并到此表 + route_profiles |
| 18 | `route_profiles` | — | — | `strategy_profiles` 新增。去掉 sys_access_policies/sys_access_policy_bindings/sys_roles/sys_role_permissions/sys_subject_role_bindings/sys_action_catalogs（6表 ABAC → grants 单表） |

### 第7组: 授权治理 (2 表)

| # | 融合表 | TokenHub 来源 | AxonHub 来源 | ai-gov 裁减 |
|---|--------|--------------|-------------|------------|
| 19 | `grants` | — | — | 替代 6 表 ABAC 引擎 |
| 20 | `model_grants` | — | — | `model_access_policies` 简化 |

### 第8组: 请求与用量 (5 表)

| # | 融合表 | TokenHub 来源 | AxonHub 来源 | ai-gov 裁减 |
|---|--------|--------------|-------------|------------|
| 21 | `request_logs` | 全部保留 | `requests` 字段 | `usage_requests` + `usage_attempts` → 合并。去掉 authorization_*/firewall_*/policy_revision 等过度字段 |
| 22 | `request_payload_logs` | 全部保留 | — | 保留 |
| 23 | `route_attempt_logs` | 全部保留 | `request_executions` → 吸收 | 保留 |
| 24 | `usage_records` | 全部保留 | `usage_logs` 全部 token 类型 | `usage_events` → 合并。去掉 settled_credit_amount（用 sell_usd 替代） |
| 25 | `quota_buckets` | 全部保留 | — | 保留 |

### 第9组: 可观测 (2 表)

| # | 融合表 | TokenHub 来源 | AxonHub 来源 | ai-gov 裁减 |
|---|--------|--------------|-------------|------------|
| 26 | `channel_probes` | — | `channel_probes` | 替代 model_channel_health_events |
| 27 | `provider_quota_status` | — | `provider_quota_status` | 新增 |

### 第10组: 基础设施 (2 表)

| # | 融合表 | TokenHub 来源 | AxonHub 来源 | ai-gov 裁减 |
|---|--------|--------------|-------------|------------|
| 28 | `audit_events` | `audit_events` 扩展 | — | 简化（去哈希链、chain_id、chain_sequence） |
| 29 | `idempotency_records` | — | — | 新增 |

---

## 三、裁减明细：ai-gov 69 表 → 融合 29 表的 40 表剪裁清单

| ai-gov 表 | 裁减原因 |
|-----------|---------|
| `tenants` | 单租户 MVP，无需多租户隔离 |
| `identity_issuer_configs` | 复杂 OIDC 配置 → users.oidc_issuer/subject |
| `identity_subjects` | → users 单表 |
| `user_login_identities` | → users 单表 |
| `user_password_credentials` | → users.password_hash |
| `user_contact_points` | → users.email |
| `oauth_provider_configs` | → users.oidc_* |
| `authentication_challenges` | MVP 不做 |
| `authentication_events` | MVP 不做 |
| `user_profiles` | → users |
| `user_sessions` | → admin_sessions 简化 |
| `organizations` | → parties (type=org) |
| `organization_memberships` | → party_members |
| `projects` | → parties (type=project) |
| `project_memberships` | → party_members |
| `funding_accounts` | → accounts |
| `ledger_transactions` | → ledgers |
| `ledger_legs` | → ledgers (direction 替代 debit/credit) |
| `balance_projections` | → 从 ledger 实时计算 |
| `user_allocations` | → accounts (1 party = 1 account) |
| `quota_reservations` | → freezes |
| `emergency_credit_grants` | MVP 不做 |
| `emergency_credit_grant_keys` | MVP 不做 |
| `credit_rate_versions` | → model_prices |
| `model_providers` | → providers |
| `model_catalog_entries` | → models |
| `model_channels` | → providers + provider_resources |
| `model_channel_health_events` | → channel_probes |
| `model_access_policies` | → model_grants |
| `provider_credential_refs` | → providers.credentials |
| `route_policies` | → route_profiles + model_routes |
| `route_policy_candidates` | → model_routes (sequence_no→priority) |
| `sys_action_catalogs` | → grants.action 枚举 |
| `sys_roles` | → grants.principal_type=role |
| `sys_role_permissions` | → grants |
| `sys_subject_role_bindings` | → grants |
| `sys_access_policies` | → grants |
| `sys_access_policy_bindings` | → grants |
| `approval_requests` | PRD 不在范围 |
| `approval_decisions` | PRD 不在范围 |
| `reconciliation_runs` | P2 阶段补 |
| `reconciliation_items` | P2 阶段补 |
| `reconciliation_differences` | P2 阶段补 |
| `reconciliation_resolutions` | P2 阶段补 |
| `closing_snapshots` | P2 阶段补 |
| `user_closing_snapshot_items` | P2 阶段补 |
| `data_integrity_findings` | MVP 不做 |
| `audit_chain_anchors` | 简化版 audit_events 无需哈希链 |
| `outbox_events` | MVP 不做 |
| `inbox_receipts` | MVP 不做 |
| `runtime_snapshots` | MVP 不做 |
| `projection_checkpoints` | 去掉 CQRS 后无必要 |
| `schema_migration_contracts` | GORM AutoMigrate 替代 |
| `key_limit_policies` | → api_keys 热字段 |
| `key_usage_counter_projections` | → quota_buckets |
| `user_governance_projections` | → 从 usage_records 实时聚合 |
| `user_api_key_secret_versions` | api_keys.key_hash + rotated_from_id 足够 |
| `sys_ui_menus/sys_ui_routes/sys_ui_action_bindings` | 前端自行管理 |

---

## 四、AxonHub 表剪裁清单

| AxonHub 表 | 处理 |
|------------|------|
| `channels` | → providers |
| `channel_model_prices` | → model_prices |
| `channel_model_price_versions` | → model_prices (reference_id 版本化) |
| `channel_probes` | → 保留 |
| `provider_quota_status` | → 保留 |
| `models` | → models 扩展字段 |
| `api_keys` | → api_keys 扩展字段 |
| `requests` | → request_logs 扩展字段 |
| `request_executions` | → route_attempt_logs 吸收 |
| `usage_logs` | → usage_records 扩展字段 (itemCode 全覆盖) |
| `users` | → users 扩展字段 |
| `oidc_identities` | → users.oidc_* |
| `projects` | → parties (type=project) |
| `user_projects` | → party_members |
| `roles` | → grants |
| `user_roles` | → grants |
| `api_key_profile_templates` | 剪裁（MVP 不做） |
| `channel_override_templates` | 剪裁 |
| `data_storages` | 剪裁（视频/图片存储不在 PRD） |
| `invitations` | 剪裁 |
| `prompt_protection_rules` | 剪裁（安全 P2 阶段补） |
| `prompts` | 剪裁（提示词库不在 PRD） |
| `systems` | 剪裁 |
| `threads` | 剪裁（编排不在 PRD） |
| `traces` | 剪裁（编排不在 PRD） |

---

## 五、PRD 验收红线 × DDL 映射

| PRD 验收项 | DDL 支撑 |
|-----------|---------|
| 统一接入 ≥5 类公有 + 1 类私有化 | `providers` + `provider_resources` + `provider_models` |
| 双轨 cost/sell + itemCode | `model_prices.price_json` + `usage_records.cost_items` |
| usage 不完整标记 | `request_logs.usage_incomplete` |
| 独立项目/组织池/出资划拨 | `parties` 多态 + `party_edges` + `allocations` |
| 价格约束 δ 默认0/硬上限20% | `route_profiles.delta_cap` + `model_routes.price_cap_delta` |
| 预算帽 vs 余额不足分码 | `accounts.budget_limit_amount` + 错误码区分 |
| 冻结超时/流式续期 | `freezes.expires_at` + `max_lifetime_at` + `renewal_count` |
| 清算状态机 | `accounts.liquidation_stage` + `liquidations` |
| 幂等 | `idempotency_records` UNIQUE(scope,actor_id,idempotency_key) |
| ModelGrant deny 优先 | `model_grants.effect` ALLOW/DENY + deny 优先逻辑 |
| 四轴越权 | `grants` 四轴分离 data/fund/iam/routing |
| 调度不改账户 | `request_logs.account_id` 鉴权时锁定 |
| 策略矩阵启停 | `route_profiles.strategies_json` |
| 禁人即禁 Key | `users.status=disabled` → 网关层联动 `api_keys.status=revoked` |
| 治理 API 幂等 | `idempotency_records` + `allocations.idempotency_key` |
| 无流水改余额 | `ledgers` 只追加，应用层强制 |

---

## 六、最短交付路径（融合 DDL → 代码 → 上线）

| 天数 | 任务 | 输出 |
|------|------|------|
| **D1** | 执行 29 表 DDL + TokenHub 6 表扩展 | DB schema ready |
| **D2** | GORM 模型 + fund/pricing/idempotency 包骨架 | 核心包编译通过 |
| **D3** | party/authz/modelgrant 包 + 路由策略引擎 | PRD 8 大包骨架就绪 |
| **D4** | 适配层: TokenHub 接口兼容 + AxonHub 数据回填脚本 | 双轨兼容 |
| **D5** | 集成测试 + 财务演示脚本 | PRD §13 验收 |

**总工期: 5 天可交付 MVP。**
