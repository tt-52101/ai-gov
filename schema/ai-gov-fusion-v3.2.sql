-- ============================================================================
-- ai-gov-fusion-v3.2.sql
-- PostgreSQL DDL for AI Governance Gateway Platform (Token Governance Base)
-- PRD Version: v3.2.0
-- Tables: 40 (29 from fusion-minimal + 11 ABAC/UI/Audit governance)
-- Dialect: PostgreSQL 16
-- ============================================================================
-- 基线: TokenHub(30表) + AxonHub(20表) 语义合并吸收
-- 融合策略:
--   TokenHub 现有表 → 向前兼容扩展（加字段，不改字段）
--   AxonHub 高收益 → itemCode计价/cost_items/渠道探针/配额状态
--   PRD v3.2.0 新增 → ABAC安全治理(9表) / 审计链锚定(1表) / UI权限绑定
-- ============================================================================

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================================================
-- 第1组: 用户与身份 (2表 — 融合 TokenHub admin_users + AxonHub users)
-- ============================================================================

-- 1. users — 统一用户表（融合 TokenHub admin_users + AxonHub users.oidc_identities）
DROP TABLE IF EXISTS users;
CREATE TABLE users (
    id              TEXT PRIMARY KEY,            -- UUID
    username        TEXT NOT NULL UNIQUE,        -- TokenHub
    email           TEXT,                        -- AxonHub + TokenHub
    display_name    TEXT,                        -- 显示名
    password_hash   TEXT,                        -- TokenHub (argon2id)
    role            TEXT NOT NULL DEFAULT 'member', -- admin / member
    status          TEXT NOT NULL DEFAULT 'active', -- active / disabled (PRD 6.2: AUTH_USER_DISABLED)
    -- AxonHub OIDC 字段
    oidc_issuer     TEXT,                        -- AxonHub oidc_identities.issuer
    oidc_subject    TEXT,                        -- AxonHub oidc_identities.subject
    avatar          TEXT,                        -- AxonHub users.avatar
    prefer_language TEXT DEFAULT 'zh-CN',
    last_login_at   TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_email ON users(email);

-- 2. admin_sessions — 管理会话（TokenHub 保留）
DROP TABLE IF EXISTS admin_sessions;
CREATE TABLE admin_sessions (
    token       TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id),
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_admin_sessions_user_id ON admin_sessions(user_id);

-- ============================================================================
-- 第2组: Party 统一主体模型 (3表 — PRD §2 新增)
-- ============================================================================

-- 3. parties — 统一主体（org/project 多态, PRD §2.3）
-- 融合: TokenHub projects + AxonHub projects → 统一 Party
DROP TABLE IF EXISTS parties;
CREATE TABLE parties (
    id              TEXT PRIMARY KEY,            -- UUID
    type            TEXT NOT NULL,               -- org / project (PRD 2.3: 同一层语义)
    name            TEXT NOT NULL,
    description     TEXT DEFAULT '',
    parent_party_id TEXT,                        -- 组织树（可空，项目可不挂靠 PRD 2.3）
    leader_user_id  TEXT,                        -- 负责人 (PRD 2.5)
    cost_center     TEXT,                        -- TokenHub projects.cost_center
    status          TEXT NOT NULL DEFAULT 'active', -- active/archived/liquidating (PRD 8.4)
    metadata        JSONB,                       -- JSON 扩展
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_parties_type ON parties(type);
CREATE INDEX idx_parties_parent ON parties(parent_party_id);
CREATE INDEX idx_parties_status ON parties(status);

-- 4. party_edges — 关系边 (PRD §2.4)
DROP TABLE IF EXISTS party_edges;
CREATE TABLE party_edges (
    id              TEXT PRIMARY KEY,
    src_party_id    TEXT NOT NULL,
    dst_party_id    TEXT NOT NULL,
    edge_type       TEXT NOT NULL,               -- parent/sponsors/owns/participates (PRD 2.4)
    allows_fund     BOOLEAN NOT NULL DEFAULT FALSE,  -- 是否开通划拨 (parent/sponsors=true)
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(src_party_id, dst_party_id, edge_type)
);
CREATE INDEX idx_party_edges_src ON party_edges(src_party_id);
CREATE INDEX idx_party_edges_dst ON party_edges(dst_party_id);

-- 5. party_members — 成员关系 (PRD §2.5)
-- 融合: TokenHub project_teams + AxonHub user_projects
DROP TABLE IF EXISTS party_members;
CREATE TABLE party_members (
    id          TEXT PRIMARY KEY,
    party_id    TEXT NOT NULL,
    user_id     TEXT NOT NULL,
    role        TEXT NOT NULL DEFAULT 'member',  -- leader / member / observer
    is_primary  BOOLEAN DEFAULT FALSE,           -- 主组织标记
    joined_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(party_id, user_id)
);
CREATE INDEX idx_party_members_party ON party_members(party_id);
CREATE INDEX idx_party_members_user ON party_members(user_id);

-- ============================================================================
-- 第3组: 资金治理 (5表 — PRD §5/§8 新增)
-- ============================================================================

-- 6. accounts — 账本 + 预算帽热字段 (PRD §5.3)
DROP TABLE IF EXISTS accounts;
CREATE TABLE accounts (
    id                      TEXT PRIMARY KEY,
    party_id                TEXT NOT NULL UNIQUE,       -- 1:1 主体 (PRD 2.5)
    available_balance       NUMERIC(18,6) NOT NULL DEFAULT 0,    -- 可用余额 (balance - frozen)
    frozen_balance          NUMERIC(18,6) NOT NULL DEFAULT 0,    -- 冻结总额
    status                  TEXT NOT NULL DEFAULT 'active', -- active/frozen/liquidating/closed (PRD 8.4)
    -- 预算帽热字段 (PRD §5.3)
    budget_limit_amount     NUMERIC(18,6),                      -- NULL=未启用
    budget_warn_ratio       NUMERIC(5,4),                       -- 告警比例 如 0.80
    budget_period           TEXT DEFAULT 'none',        -- none/calendar_month/calendar_day/custom
    budget_period_start     TIMESTAMPTZ,
    budget_period_end       TIMESTAMPTZ,
    budget_consumed_amount  NUMERIC(18,6) NOT NULL DEFAULT 0,   -- 本周期已确认 sell 累计
    budget_version          INTEGER NOT NULL DEFAULT 0, -- 配置乐观锁
    -- 清算 (PRD §8.4)
    liquidation_stage       TEXT,                       -- block_new/drain/transfer/completed
    liquidation_target_id   TEXT,                       -- 回流目标账户
    liquidation_started_at  TIMESTAMPTZ,
    -- 并发控制
    version                 INTEGER NOT NULL DEFAULT 0, -- 乐观锁
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_accounts_party ON accounts(party_id);
CREATE INDEX idx_accounts_status ON accounts(status);

-- 7. ledgers — 流水（只追加, PRD §2.5 规则5）
DROP TABLE IF EXISTS ledgers;
CREATE TABLE ledgers (
    id              TEXT PRIMARY KEY,
    account_id      TEXT NOT NULL,                     -- 归属账户
    direction       TEXT NOT NULL,                     -- debit/credit/freeze/unfreeze/settle
    amount          NUMERIC(18,6) NOT NULL,            -- 变动金额
    balance_after   NUMERIC(18,6) NOT NULL,            -- 变动后可用余额快照
    frozen_after    NUMERIC(18,6),                     -- 变动后冻结余额快照
    -- 双轨金额 (PRD §4)
    cost_amount     NUMERIC(18,6),                     -- 上游成本
    sell_amount     NUMERIC(18,6),                     -- 内部结算
    -- 关联
    freeze_id       TEXT,
    request_id      TEXT,
    allocation_id   TEXT,
    user_id         TEXT,                              -- 归属人
    api_key_id      TEXT,
    -- 幂等与审计
    idempotency_key TEXT,
    reason          TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_ledgers_account ON ledgers(account_id, created_at);
CREATE INDEX idx_ledgers_freeze ON ledgers(freeze_id);
CREATE INDEX idx_ledgers_request ON ledgers(request_id);
CREATE INDEX idx_ledgers_idem ON ledgers(account_id, idempotency_key);

-- 8. freezes — 冻结/预占 (PRD §8.3)
DROP TABLE IF EXISTS freezes;
CREATE TABLE freezes (
    id                  TEXT PRIMARY KEY,
    account_id          TEXT NOT NULL,
    request_id          TEXT,                          -- 关联请求
    api_key_id          TEXT,                          -- 关联 Key
    user_id             TEXT NOT NULL,                 -- 归属人
    amount              NUMERIC(18,6) NOT NULL,        -- 冻结金额 (PRD 8.1 冻结=max候选sell)
    estimated_sell      NUMERIC(18,6) NOT NULL,        -- 预估 sell (候选集最大值)
    status              TEXT NOT NULL DEFAULT 'active',-- active/settled/expired/released
    -- TTL 管理 (PRD 8.3: 默认15min, 可配1-60, 流式续期)
    expires_at          TIMESTAMPTZ NOT NULL,          -- 过期时间
    max_lifetime_at     TIMESTAMPTZ,                   -- 流式累计有效期上限(如+2h)
    renewal_count       INTEGER NOT NULL DEFAULT 0,    -- 续期次数
    last_renewed_at     TIMESTAMPTZ,                   -- 最近续期时间
    settled_at          TIMESTAMPTZ,
    settle_amount       NUMERIC(18,6),                 -- 实际结算 sell
    settle_cost         NUMERIC(18,6),                 -- 实际成本 cost
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_freezes_account ON freezes(account_id);
CREATE INDEX idx_freezes_request ON freezes(request_id);
CREATE INDEX idx_freezes_expiry ON freezes(status, expires_at);
CREATE INDEX idx_freezes_user ON freezes(user_id);

-- 9. allocations — 划拨记录 (PRD §8.2/§8.6)
DROP TABLE IF EXISTS allocations;
CREATE TABLE allocations (
    id              TEXT PRIMARY KEY,
    src_account_id  TEXT NOT NULL,
    dst_account_id  TEXT NOT NULL,
    amount          NUMERIC(18,6) NOT NULL,
    channel         TEXT NOT NULL,                     -- parent/sponsors/whitelist (PRD 8.2)
    edge_id         TEXT,                              -- 关联 party_edge
    status          TEXT NOT NULL DEFAULT 'pending',   -- pending/completed/reverted
    idempotency_key TEXT,
    actor_user_id   TEXT,
    reason          TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at    TIMESTAMPTZ
);
CREATE INDEX idx_allocations_src ON allocations(src_account_id);
CREATE INDEX idx_allocations_dst ON allocations(dst_account_id);
CREATE INDEX idx_allocations_idem ON allocations(src_account_id, idempotency_key);

-- 10. liquidations — 清算状态机 (PRD §8.4)
DROP TABLE IF EXISTS liquidations;
CREATE TABLE liquidations (
    id                  TEXT PRIMARY KEY,
    party_id            TEXT NOT NULL,
    account_id          TEXT NOT NULL,
    target_account_id   TEXT,
    status              TEXT NOT NULL DEFAULT 'blocking', -- blocking/draining/refunding/closing/closed
    initiated_by        TEXT NOT NULL,
    initiated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at           TIMESTAMPTZ,
    metadata            JSONB,                          -- JSON
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_liquidations_party ON liquidations(party_id);
CREATE INDEX idx_liquidations_account ON liquidations(account_id);
CREATE INDEX idx_liquidations_status ON liquidations(status);

-- ============================================================================
-- 第4组: API Key (1表 — TokenHub 现有 + AxonHub 扩展 + PRD account_id)
-- ============================================================================

-- 11. api_keys (TokenHub 扩展)
-- 融合: TokenHub api_keys + AxonHub api_keys 字段
-- 新增: account_id(PRD 2.5) + owner_user_id(PRD 2.5)
DROP TABLE IF EXISTS api_keys;
CREATE TABLE api_keys (
    -- === TokenHub 现有字段 ===
    id                      TEXT PRIMARY KEY,
    project_id              TEXT,                       -- → party_id (过渡期保留)
    name                    TEXT NOT NULL,
    key_hash                TEXT NOT NULL UNIQUE,
    key_prefix              TEXT,                       -- 非敏感前缀 (sk-xxxx)
    key_suffix              TEXT,                       -- 非敏感后缀
    -- 权限与限制 (TokenHub)
    allowed                 JSONB,                      -- JSON: 允许的 actions
    ip_allowlist            JSONB,                      -- JSON
    limit_daily_requests    INTEGER,
    limit_monthly_requests  INTEGER,
    limit_daily_tokens      INTEGER,
    limit_monthly_tokens    INTEGER,
    limit_daily_cost_usd    NUMERIC(18,6),
    limit_monthly_cost_usd  NUMERIC(18,6),
    limit_max_concurrency   INTEGER,
    -- === PRD 新增字段 ===
    owner_user_id           TEXT NOT NULL,              -- 归属人 (PRD 2.5: 必须关联实体人)
    account_id              TEXT NOT NULL,              -- 绑定账本 (PRD 2.5: 必须且唯一)
    party_id                TEXT,                       -- 归属主体 (冗余加速)
    -- === AxonHub 扩展字段 ===
    ax_key_type             TEXT DEFAULT 'user',        -- user/service_account/noauth
    scopes                  JSONB,                      -- JSON: AxonHub scopes
    profiles                JSONB,                      -- JSON: AxonHub profiles
    allowed_ips             JSONB,                      -- JSON (与 ip_allowlist 互补)
    -- === 通用 ===
    status                  TEXT NOT NULL DEFAULT 'active', -- active/revoked/expired (PRD KEY-01)
    issued_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at              TIMESTAMPTZ,
    last_used_at            TIMESTAMPTZ,
    rotated_from_id         TEXT,                       -- 轮换来源
    grace_until             TIMESTAMPTZ,                -- 轮换宽限期
    metadata                JSONB,                      -- JSON
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_api_keys_owner ON api_keys(owner_user_id);
CREATE INDEX idx_api_keys_account ON api_keys(account_id);
CREATE INDEX idx_api_keys_party ON api_keys(party_id);
CREATE INDEX idx_api_keys_project ON api_keys(project_id);
CREATE INDEX idx_api_keys_status ON api_keys(status);
CREATE INDEX idx_api_keys_hash ON api_keys(key_hash);

-- ============================================================================
-- 第5组: 模型目录与供应商 (4表 — TokenHub 现有 + AxonHub 扩展)
-- ============================================================================

-- 12. providers — 供应商 (TokenHub 扩展, 吸收 AxonHub Channel 字段)
DROP TABLE IF EXISTS providers;
CREATE TABLE providers (
    -- === TokenHub 现有 ===
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL UNIQUE,
    type            TEXT NOT NULL,                     -- openai/azure/custom (TokenHub)
    base_url        TEXT,
    api_key         TEXT,                              -- 加密存储
    status          TEXT NOT NULL DEFAULT 'active',
    healthy         BOOLEAN DEFAULT TRUE,              -- TokenHub
    priority        INTEGER DEFAULT 0,
    headers         JSONB,                             -- JSON
    options         JSONB,                             -- JSON
    -- === AxonHub Channel 扩展 ===
    ax_channel_type TEXT,                              -- AxonHub 渠道类型 (60+)
    credentials     JSONB,                             -- JSON: ChannelCredentials (加密)
    supported_models JSONB,                            -- JSON: []string
    policies        JSONB,                             -- JSON: ChannelPolicies
    channel_settings JSONB,                            -- JSON
    endpoints       JSONB,                             -- JSON
    ordering_weight INTEGER DEFAULT 0,
    error_message   TEXT,
    remark          TEXT,
    -- === 融合扩展 ===
    account_id      TEXT,                              -- 上游成本记账账户
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_providers_type ON providers(type);
CREATE INDEX idx_providers_status ON providers(status);

-- 13. provider_resources — 供应商资源 (TokenHub 保留)
DROP TABLE IF EXISTS provider_resources;
CREATE TABLE provider_resources (
    id                  TEXT PRIMARY KEY,
    provider_id         TEXT NOT NULL,
    name                TEXT NOT NULL,
    resource_group      TEXT,
    resource_type       TEXT DEFAULT 'llm',            -- llm/image/embedding
    base_url            TEXT,
    api_key             TEXT,                          -- 加密存储
    region              TEXT,
    environment         TEXT,
    status              TEXT NOT NULL DEFAULT 'active',
    healthy             BOOLEAN DEFAULT TRUE,
    priority            INTEGER DEFAULT 0,
    weight              INTEGER DEFAULT 100,
    rate_limit_rpm      INTEGER,
    token_limit_tpm     INTEGER,
    max_concurrency     INTEGER,
    headers             JSONB,                         -- JSON
    options             JSONB,                         -- JSON
    credential_blob     TEXT,                          -- 加密凭证
    -- 熔断
    failure_count       INTEGER DEFAULT 0,
    cooldown_until      TIMESTAMPTZ,
    last_used_at        TIMESTAMPTZ,
    last_checked_at     TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_prov_res_provider ON provider_resources(provider_id);
CREATE INDEX idx_prov_res_group ON provider_resources(resource_group);
CREATE INDEX idx_prov_res_status ON provider_resources(status);

-- 14. models — 逻辑模型目录 (TokenHub 扩展, 吸收 AxonHub Model + 双轨)
-- 融合: TokenHub models + AxonHub models
DROP TABLE IF EXISTS models;
CREATE TABLE models (
    -- === TokenHub 现有 ===
    id                      TEXT PRIMARY KEY,
    name                    TEXT NOT NULL UNIQUE,       -- 逻辑模型名 (PRD 7.2)
    category                TEXT,                       -- chat/embedding/image/audio
    family                  TEXT,
    modality                TEXT,
    context_window          INTEGER,
    -- TokenHub 单轨定价 (保留兼容)
    input_price_per_1m      NUMERIC(18,6),              -- cost 轨道 (上游)
    cache_read_price_per_1m NUMERIC(18,6),
    output_price_per_1m     NUMERIC(18,6),
    embedding_price_per_1m  NUMERIC(18,6),
    input_modalities        JSONB,                      -- JSON
    output_modalities       JSONB,                      -- JSON
    capabilities            JSONB,                      -- JSON: ["chat","streaming","embedding"]
    supported_parameters    JSONB,                      -- JSON
    metadata                JSONB,                      -- JSON
    status                  TEXT NOT NULL DEFAULT 'active',
    -- === AxonHub Model 扩展 ===
    developer               TEXT,                       -- 开发商
    icon                    TEXT,
    model_group             TEXT,                       -- 模型组
    model_card              JSONB,                      -- JSON
    model_settings          JSONB,                      -- JSON
    ax_model_type           TEXT DEFAULT 'chat',        -- chat/embedding/rerank/image/video
    -- === PRD 双轨 sell 价格 (热字段, PRD §4) ===
    sell_input_price_per_1m     NUMERIC(18,6),          -- 内部结算 input
    sell_cache_read_price_per_1m NUMERIC(18,6),
    sell_output_price_per_1m    NUMERIC(18,6),
    sell_embedding_price_per_1m NUMERIC(18,6),
    -- === PRD 扩展 ===
    item_codes              JSONB,                      -- JSON: 支持的 itemCode 列表
    data_classification     TEXT DEFAULT 'internal',    -- public/internal/confidential/restricted
    network_class           TEXT DEFAULT 'external',    -- internal/external
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_models_category ON models(category);
CREATE INDEX idx_models_status ON models(status);
CREATE INDEX idx_models_developer ON models(developer);

-- 15. provider_models — 供应商模型映射 (TokenHub 保留)
DROP TABLE IF EXISTS provider_models;
CREATE TABLE provider_models (
    id                  TEXT PRIMARY KEY,
    provider_id         TEXT NOT NULL,
    upstream_model      TEXT NOT NULL,                 -- 上游实际模型名
    display_name        TEXT,
    canonical_name      TEXT,                           -- → models.name
    category            TEXT,
    family              TEXT,
    modality            TEXT,
    context_window      INTEGER,
    input_price_per_1m  NUMERIC(18,6),
    cache_read_price_per_1m NUMERIC(18,6),
    output_price_per_1m NUMERIC(18,6),
    input_modalities    JSONB,
    output_modalities   JSONB,
    capabilities        JSONB,
    supported_parameters JSONB,
    metadata            JSONB,
    source              TEXT,                           -- auto_sync/manual
    status              TEXT NOT NULL DEFAULT 'active',
    last_seen_at        TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(provider_id, upstream_model)
);
CREATE INDEX idx_prov_models_provider ON provider_models(provider_id);
CREATE INDEX idx_prov_models_canonical ON provider_models(canonical_name);

-- ============================================================================
-- 第6组: 定价与路由 (3表 — PRD §3/§4 新增 + TokenHub 扩展)
-- ============================================================================

-- 16. model_prices — 渠道×模型双轨价目 (PRD §4.4, 对齐 AxonHub channel_model_prices)
DROP TABLE IF EXISTS model_prices;
CREATE TABLE model_prices (
    id              TEXT PRIMARY KEY,
    model_id        TEXT NOT NULL,                     -- → models.name
    channel_id      TEXT,                              -- → provider_resources.id, NULL=默认
    reference_id    TEXT NOT NULL UNIQUE,               -- 版本引用标识
    price_json      JSONB NOT NULL,                    -- JSON: PRD 4.4 结构 {items:[{itemCode,cost:{mode,rate},sell:{mode,rate}}], schedule:{timezone,overrides}}
    status          TEXT NOT NULL DEFAULT 'active',    -- active/archived
    effective_start_at TIMESTAMPTZ,
    effective_end_at   TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- price_json 示例见文档末尾注释
CREATE INDEX idx_model_prices_model ON model_prices(model_id, channel_id);
CREATE INDEX idx_model_prices_ref ON model_prices(reference_id);

-- 17. model_routes — 路由条目 (TokenHub 扩展)
DROP TABLE IF EXISTS model_routes;
CREATE TABLE model_routes (
    -- === TokenHub 现有 ===
    id                  TEXT PRIMARY KEY,
    model_name          TEXT NOT NULL,                 -- → models.name
    provider_id         TEXT,
    provider_resource_id TEXT,
    resource_group      TEXT,
    sticky_session      BOOLEAN DEFAULT FALSE,
    provider_model      TEXT,
    priority            INTEGER DEFAULT 0,
    weight              INTEGER DEFAULT 100,
    quality_score       INTEGER DEFAULT 0,
    cost_score          INTEGER DEFAULT 0,
    status              TEXT NOT NULL DEFAULT 'active',
    strategy            JSONB,                          -- JSON: 策略配置
    last_used_at        TIMESTAMPTZ,
    project_scope       TEXT,                           -- all/selected
    project_ids         JSONB,                          -- JSON
    -- === 融合扩展 (PRD §3.3/§8.1) ===
    route_profile_id    TEXT,                           -- → route_profiles.id
    channel_id          TEXT,                           -- 直接绑定渠道
    model_price_id      TEXT,                           -- → model_prices.id
    price_cap_delta     NUMERIC(18,6) DEFAULT 0.0,       -- δ (PRD 8.1: 默认0, 硬上限0.20)
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_model_routes_model ON model_routes(model_name);
CREATE INDEX idx_model_routes_provider ON model_routes(provider_id);
CREATE INDEX idx_model_routes_resource ON model_routes(provider_resource_id);
CREATE INDEX idx_model_routes_profile ON model_routes(route_profile_id);

-- 18. route_profiles — 策略矩阵档案 (PRD §3.3)
DROP TABLE IF EXISTS route_profiles;
CREATE TABLE route_profiles (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL UNIQUE,
    description     TEXT,
    -- 策略配置 JSON: [{"code":"S-COMPLIANCE","enabled":true,"priority":0}, ...]
    -- 11种策略: S-PRI/S-HEALTH/S-WEIGHT/S-AFFINITY/S-COST/S-LATENCY/S-ERROR/S-RATE/S-TAG/S-COMPLIANCE/S-CACHE
    strategies_json JSONB NOT NULL DEFAULT '[]',
    delta_cap       NUMERIC(18,6) NOT NULL DEFAULT 0.0, -- δ 价格帽 (PRD 8.1)
    max_attempts    INTEGER DEFAULT 3,
    allow_fallback  BOOLEAN DEFAULT TRUE,
    status          TEXT NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================================
-- 第7组: 授权治理 (2表 — PRD §7 新增)
-- ============================================================================

-- 19. grants — 四轴正交授权 (PRD §7.1/§7.3)
DROP TABLE IF EXISTS grants;
CREATE TABLE grants (
    id              TEXT PRIMARY KEY,
    principal_type  TEXT NOT NULL,                     -- user/role/party/key
    principal_id    TEXT NOT NULL,
    axis            TEXT NOT NULL,                     -- data/fund/iam/routing (PRD 7.1)
    action          TEXT NOT NULL,                     -- balance.read/allocate/price.write (PRD 7.3)
    resource_type   TEXT,                              -- party/account/model
    resource_id     TEXT,                              -- 具体资源ID 或 '*'
    effect          TEXT NOT NULL DEFAULT 'allow',     -- allow/deny
    conditions      JSONB,                             -- JSON: 附加条件
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_grants_principal ON grants(principal_type, principal_id);
CREATE INDEX idx_grants_axis ON grants(axis, action);
CREATE INDEX idx_grants_resource ON grants(resource_type, resource_id);

-- 20. model_grants — 模型访问授权 (PRD §7.2)
DROP TABLE IF EXISTS model_grants;
CREATE TABLE model_grants (
    id              TEXT PRIMARY KEY,
    principal_type  TEXT NOT NULL,                     -- party/person/key/role
    principal_id    TEXT NOT NULL,
    model_id        TEXT,                              -- 单个模型
    model_tag       TEXT,                              -- 模型标签组
    effect          TEXT NOT NULL,                     -- allow/deny (deny 优先 PRD 7.2)
    priority        INTEGER DEFAULT 0,                 -- 冲突解析
    quota_limit     INTEGER,                           -- 配额上限 (PRD v3.2.0)
    conditions      JSONB,                             -- JSON
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_model_grants_principal ON model_grants(principal_type, principal_id);
CREATE INDEX idx_model_grants_model ON model_grants(model_id);
CREATE INDEX idx_model_grants_effect ON model_grants(effect);

-- ============================================================================
-- 第8组: 请求与用量 (5表 — TokenHub 扩展 + AxonHub 吸收)
-- ============================================================================

-- 21. request_logs — 请求日志 (TokenHub 扩展, 吸收 AxonHub Request)
DROP TABLE IF EXISTS request_logs;
CREATE TABLE request_logs (
    -- === TokenHub 现有 ===
    id                  TEXT PRIMARY KEY,
    request_id          TEXT NOT NULL UNIQUE,           -- 对外请求标识
    project_id          TEXT,                           -- → party_id
    api_key_id          TEXT,
    model_name          TEXT,
    provider_id         TEXT,
    provider_resource_id TEXT,
    provider_model      TEXT,
    upstream_request_id TEXT,                           -- 上游返回的 request_id
    served_model        TEXT,
    model_e_tag         TEXT,
    transport           TEXT DEFAULT 'https',
    status_code         INTEGER,
    error_code          TEXT,                           -- PRD §6 错误码
    latency_ms          INTEGER,
    client_ip           TEXT,
    user_agent          TEXT,
    -- === AxonHub Request 扩展 ===
    stream              BOOLEAN DEFAULT FALSE,           -- 是否流式
    format              TEXT DEFAULT 'openai/chat_completions',
    request_body        JSONB,                           -- JSON
    response_body       JSONB,                           -- JSON
    ax_status           TEXT,                            -- pending/processing/completed/failed
    -- 延迟细分 (AxonHub)
    first_token_latency_ms   INTEGER,
    reasoning_duration_ms    INTEGER,
    -- === PRD 双轨扩展 ===
    account_id          TEXT,                            -- 鉴权时锁定, 路由不可改 (PRD 7.5)
    freeze_id           TEXT,                            -- 关联冻结
    user_id             TEXT,                            -- 归属人 (归因)
    party_id            TEXT,                            -- 归属主体
    cost_usd            NUMERIC(18,6),                   -- 上游成本
    sell_usd            NUMERIC(18,6),                   -- 内部结算
    cost_items          JSONB,                           -- JSON: itemCode 级明细
    usage_incomplete    BOOLEAN DEFAULT FALSE,           -- 用量不完整标记 (PRD 4.5)
    business_tags       JSONB,                           -- JSON: 业务标签 (仅归因 PRD 1.3)
    route_profile_id    TEXT,                            -- 使用的策略档案
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_request_logs_req ON request_logs(request_id);
CREATE INDEX idx_request_logs_key ON request_logs(api_key_id, created_at);
CREATE INDEX idx_request_logs_project ON request_logs(project_id, created_at);
CREATE INDEX idx_request_logs_account ON request_logs(account_id, created_at);
CREATE INDEX idx_request_logs_user ON request_logs(user_id, created_at);
CREATE INDEX idx_request_logs_model ON request_logs(model_name, created_at);
CREATE INDEX idx_request_logs_error ON request_logs(error_code, created_at);

-- 22. request_payload_logs — 请求响应体 (TokenHub 保留)
DROP TABLE IF EXISTS request_payload_logs;
CREATE TABLE request_payload_logs (
    id                  TEXT PRIMARY KEY,
    request_id          TEXT NOT NULL UNIQUE,
    request_body        JSONB,
    response_body       JSONB,
    request_truncated   BOOLEAN DEFAULT FALSE,
    response_truncated  BOOLEAN DEFAULT FALSE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 23. route_attempt_logs — 路由尝试 (TokenHub 保留)
DROP TABLE IF EXISTS route_attempt_logs;
CREATE TABLE route_attempt_logs (
    id                  TEXT PRIMARY KEY,
    request_id          TEXT NOT NULL,
    attempt_index       INTEGER NOT NULL,
    route_id            TEXT,                           -- → model_routes.id
    provider_id         TEXT,
    provider_resource_id TEXT,
    provider_model      TEXT,
    status_code         INTEGER,
    error_code          TEXT,
    error_message       TEXT,
    invoked             BOOLEAN DEFAULT TRUE,            -- 是否实际调用
    latency_ms          INTEGER,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_route_attempt_req ON route_attempt_logs(request_id);
CREATE INDEX idx_route_attempt_route ON route_attempt_logs(route_id, invoked, created_at);

-- 24. usage_records — 用量记录 (TokenHub 扩展, 吸收 AxonHub UsageLog)
DROP TABLE IF EXISTS usage_records;
CREATE TABLE usage_records (
    -- === TokenHub 现有 ===
    id                  TEXT PRIMARY KEY,
    request_id          TEXT NOT NULL,
    project_id          TEXT,
    api_key_id          TEXT,
    model_name          TEXT,
    provider_id         TEXT,
    provider_resource_id TEXT,
    input_tokens        INTEGER DEFAULT 0,
    cached_input_tokens INTEGER DEFAULT 0,
    cache_write_tokens  INTEGER DEFAULT 0,
    output_tokens       INTEGER DEFAULT 0,
    reasoning_tokens    INTEGER DEFAULT 0,
    total_tokens        INTEGER DEFAULT 0,
    cost_usd            NUMERIC(18,6),
    attributed_user_id  TEXT,
    -- === AxonHub UsageLog 扩展 (itemCode 全覆盖) ===
    channel_id          TEXT,
    prompt_audio_tokens         INTEGER DEFAULT 0,
    prompt_cached_tokens        INTEGER DEFAULT 0,      -- AxonHub: 缓存读
    prompt_write_cached_tokens  INTEGER DEFAULT 0,      -- AxonHub: 缓存写
    prompt_write_cached_5m      INTEGER DEFAULT 0,
    prompt_write_cached_1h      INTEGER DEFAULT 0,
    completion_audio_tokens     INTEGER DEFAULT 0,
    completion_reasoning_tokens INTEGER DEFAULT 0,      -- AxonHub: 推理
    accepted_prediction_tokens  INTEGER DEFAULT 0,
    rejected_prediction_tokens  INTEGER DEFAULT 0,
    source              TEXT DEFAULT 'api',
    format              TEXT DEFAULT 'openai/chat_completions',
    total_cost          NUMERIC(18,6),                   -- AxonHub 原生成本
    cost_price_ref_id   TEXT,                            -- 价目版本引用
    -- === PRD 双轨扩展 ===
    sell_usd            NUMERIC(18,6),                   -- 内部结算
    cost_items          JSONB,                           -- JSON: itemCode→{tokens,cost,sell} 明细
    account_id          TEXT,
    freeze_id           TEXT,
    item_code           TEXT,                            -- 主费用项
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_usage_records_req ON usage_records(request_id);
CREATE INDEX idx_usage_records_key ON usage_records(api_key_id, created_at);
CREATE INDEX idx_usage_records_project ON usage_records(project_id, created_at);
CREATE INDEX idx_usage_records_model ON usage_records(model_name, created_at);
CREATE INDEX idx_usage_records_user ON usage_records(attributed_user_id);
CREATE INDEX idx_usage_records_account ON usage_records(account_id);

-- 25. quota_buckets — Key 配额桶 (TokenHub 保留)
DROP TABLE IF EXISTS quota_buckets;
CREATE TABLE quota_buckets (
    key_id          TEXT NOT NULL,
    scope           TEXT NOT NULL,                     -- daily/monthly
    bucket          TEXT NOT NULL,                     -- 时间桶标识
    requests        INTEGER DEFAULT 0,
    prompt_tokens   INTEGER DEFAULT 0,
    completion_tokens INTEGER DEFAULT 0,
    total_tokens    INTEGER DEFAULT 0,
    cost_usd        NUMERIC(18,6) DEFAULT 0,
    PRIMARY KEY (key_id, scope, bucket)
);

-- ============================================================================
-- 第9组: 可观测与调度 (2表 — AxonHub 吸收)
-- ============================================================================

-- 26. channel_probes — 渠道健康探针 (AxonHub)
DROP TABLE IF EXISTS channel_probes;
CREATE TABLE channel_probes (
    id                      TEXT PRIMARY KEY,
    channel_id              TEXT NOT NULL,             -- → provider_resources.id
    total_request_count     INTEGER NOT NULL DEFAULT 0,
    success_request_count   INTEGER NOT NULL DEFAULT 0,
    avg_tokens_per_second   NUMERIC(18,6),
    avg_first_token_ms      NUMERIC(18,6),
    consecutive_failures    INTEGER DEFAULT 0,
    health_status           TEXT DEFAULT 'unknown',    -- healthy/degraded/unhealthy/circuit_open
    reason_code             TEXT,
    evidence_json           JSONB DEFAULT '{}',
    observed_at             TIMESTAMPTZ NOT NULL,       -- 观测时间窗
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_channel_probes_ch ON channel_probes(channel_id, observed_at);

-- 27. provider_quota_status — 上游配额状态 (AxonHub)
DROP TABLE IF EXISTS provider_quota_status;
CREATE TABLE provider_quota_status (
    id              TEXT PRIMARY KEY,
    channel_id      TEXT NOT NULL UNIQUE,              -- → provider_resources.id
    provider_type   TEXT,
    status          TEXT NOT NULL DEFAULT 'available', -- available/warning/exhausted/unknown
    quota_data      JSONB,                              -- JSON: 配额详情
    next_reset_at   TIMESTAMPTZ,
    ready           BOOLEAN DEFAULT TRUE,
    next_check_at   TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_prov_quota_status ON provider_quota_status(status);
CREATE INDEX idx_prov_quota_check ON provider_quota_status(next_check_at);

-- ============================================================================
-- 第10组: 基础设施 (2表 — TokenHub 扩展)
-- ============================================================================

-- 28. audit_events — 审计事件 (简化自 ai-gov, 对齐 PRD §9.9)
DROP TABLE IF EXISTS audit_events;
CREATE TABLE audit_events (
    id              TEXT PRIMARY KEY,
    actor_user_id   TEXT,
    actor_name      TEXT,
    action          TEXT NOT NULL,                     -- 与 grants.action 对齐
    resource_type   TEXT NOT NULL,
    resource_id     TEXT NOT NULL,
    status          TEXT,                              -- success/failure
    message         TEXT,
    before_snapshot JSONB,                              -- JSON: 变更前
    after_snapshot  JSONB,                              -- JSON: 变更后
    ip              TEXT,
    user_agent      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_audit_events_actor ON audit_events(actor_user_id, created_at);
CREATE INDEX idx_audit_events_resource ON audit_events(resource_type, resource_id, created_at);
CREATE INDEX idx_audit_events_action ON audit_events(action, created_at);

-- 29. idempotency_records — 治理 API 幂等 (PRD §8.7/§11.4)
DROP TABLE IF EXISTS idempotency_records;
CREATE TABLE idempotency_records (
    id              TEXT PRIMARY KEY,
    scope           TEXT NOT NULL,                     -- allocate/liquidate/compensate
    actor_id        TEXT NOT NULL,
    idempotency_key TEXT NOT NULL,
    request_hash    TEXT NOT NULL,                     -- SHA-256 请求体
    status          TEXT NOT NULL DEFAULT 'started',   -- started/succeeded/failed
    response_json   JSONB,                              -- 首次成功响应
    resource_ref    TEXT,                              -- 如 transaction_id
    expires_at      TIMESTAMPTZ NOT NULL,              -- 窗口 ≥24h
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(scope, actor_id, idempotency_key)
);
CREATE INDEX idx_idempotency_expiry ON idempotency_records(expires_at);

-- ============================================================================
-- 第11组: ABAC 安全治理 (9表 — PRD v3.2.0 §10 新增)
-- ============================================================================
-- 设计说明:
--   sys_action_catalogs:     原子操作目录，按四轴 (data/fund/iam/routing) 分类
--   sys_roles:               角色定义，is_system 角色不可删除
--   sys_role_permissions:    角色→操作 N:M 映射
--   sys_subject_role_bindings: 主体→角色绑定，支持作用域 (scope_party_id)
--   sys_access_policies:     ABAC 策略，effect + conditions_json 实现细粒度控制
--   sys_access_policy_bindings: 策略→主体绑定
--   sys_ui_menus:            动态菜单树，self-ref parent_id 支持无限层级
--   sys_ui_routes:           前端路由→菜单→操作 权限映射
--   sys_ui_action_bindings:  页面内按钮→操作 权限映射
-- ============================================================================

-- 30. sys_action_catalogs — 原子操作目录
DROP TABLE IF EXISTS sys_action_catalogs;
CREATE TABLE sys_action_catalogs (
    id              TEXT PRIMARY KEY,
    action_code     TEXT NOT NULL UNIQUE,              -- 例: balance.read / model.invoke / party.create
    action_name     TEXT NOT NULL,                     -- 例: 查看余额
    axis            TEXT NOT NULL,                     -- data / fund / iam / routing
    resource_type   TEXT,                              -- 关联资源类型: party / account / model / key
    description     TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_sys_action_catalogs_axis ON sys_action_catalogs(axis);
CREATE INDEX idx_sys_action_catalogs_resource ON sys_action_catalogs(resource_type);

COMMENT ON TABLE sys_action_catalogs IS '原子操作目录 — 按四轴分类的权限原子操作';
COMMENT ON COLUMN sys_action_catalogs.action_code IS '操作编码，如 balance.read';
COMMENT ON COLUMN sys_action_catalogs.axis IS '治理轴: data(数据) / fund(资金) / iam(身份) / routing(路由)';

-- 31. sys_roles — 角色定义
DROP TABLE IF EXISTS sys_roles;
CREATE TABLE sys_roles (
    id              TEXT PRIMARY KEY,
    role_code       TEXT NOT NULL UNIQUE,              -- 例: super_admin / finance_mgr / model_user
    role_name       TEXT NOT NULL,                     -- 例: 超级管理员
    description     TEXT,
    is_system       BOOLEAN NOT NULL DEFAULT FALSE,     -- 系统角色不可删除
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_sys_roles_code ON sys_roles(role_code);

COMMENT ON TABLE sys_roles IS '角色定义 — 系统角色 (is_system=true) 不可删除';
COMMENT ON COLUMN sys_roles.is_system IS '是否为系统内置角色，系统角色不可删除';

-- 32. sys_role_permissions — 角色权限映射
DROP TABLE IF EXISTS sys_role_permissions;
CREATE TABLE sys_role_permissions (
    id              TEXT PRIMARY KEY,
    role_id         TEXT NOT NULL REFERENCES sys_roles(id) ON DELETE CASCADE,
    action_id       TEXT NOT NULL REFERENCES sys_action_catalogs(id) ON DELETE CASCADE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(role_id, action_id)
);
CREATE INDEX idx_sys_role_perms_role ON sys_role_permissions(role_id);
CREATE INDEX idx_sys_role_perms_action ON sys_role_permissions(action_id);

COMMENT ON TABLE sys_role_permissions IS '角色→操作 多对多映射';

-- 33. sys_subject_role_bindings — 主体角色绑定
DROP TABLE IF EXISTS sys_subject_role_bindings;
CREATE TABLE sys_subject_role_bindings (
    id              TEXT PRIMARY KEY,
    subject_type    TEXT NOT NULL,                     -- user / party
    subject_id      TEXT NOT NULL,
    role_id         TEXT NOT NULL REFERENCES sys_roles(id) ON DELETE CASCADE,
    scope_party_id  TEXT,                              -- NULL=全局作用域 (不限定组织)
    valid_from      TIMESTAMPTZ,
    valid_until     TIMESTAMPTZ,                       -- NULL=永久有效
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_sys_srb_subject ON sys_subject_role_bindings(subject_type, subject_id);
CREATE INDEX idx_sys_srb_role ON sys_subject_role_bindings(role_id);
CREATE INDEX idx_sys_srb_scope ON sys_subject_role_bindings(scope_party_id);
CREATE INDEX idx_sys_srb_validity ON sys_subject_role_bindings(valid_from, valid_until);

COMMENT ON TABLE sys_subject_role_bindings IS '主体→角色绑定 — scope_party_id 限制角色生效的组织范围';
COMMENT ON COLUMN sys_subject_role_bindings.scope_party_id IS 'NULL 表示全局角色，否则仅在指定 party 及其子组织内生效';

-- 34. sys_access_policies — ABAC 策略定义
DROP TABLE IF EXISTS sys_access_policies;
CREATE TABLE sys_access_policies (
    id              TEXT PRIMARY KEY,
    policy_code     TEXT NOT NULL UNIQUE,              -- 例: P-DENY-EXTERNAL-MODEL
    policy_name     TEXT NOT NULL,                     -- 例: 禁止外部网络模型访问
    effect          TEXT NOT NULL DEFAULT 'allow',     -- allow / deny
    conditions_json JSONB NOT NULL DEFAULT '{}',       -- ABAC 条件: {axis, action, resource, env, time}
    priority        INTEGER NOT NULL DEFAULT 0,         -- 优先级 (数值越大越优先)
    is_system       BOOLEAN NOT NULL DEFAULT FALSE,     -- 系统策略不可删除
    description     TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_sys_access_policies_code ON sys_access_policies(policy_code);
CREATE INDEX idx_sys_access_policies_effect ON sys_access_policies(effect);
CREATE INDEX idx_sys_access_policies_priority ON sys_access_policies(priority DESC);

COMMENT ON TABLE sys_access_policies IS 'ABAC 策略 — 基于属性的访问控制策略，deny 优先于 allow';
COMMENT ON COLUMN sys_access_policies.conditions_json IS 'ABAC 条件 JSON，如: {"axis":"data","resource_type":"model","data_classification":["confidential","restricted"]}';
COMMENT ON COLUMN sys_access_policies.priority IS '策略优先级，数值越大越优先，deny 策略应设置较高优先级';

-- 35. sys_access_policy_bindings — 策略主体绑定
DROP TABLE IF EXISTS sys_access_policy_bindings;
CREATE TABLE sys_access_policy_bindings (
    id              TEXT PRIMARY KEY,
    policy_id       TEXT NOT NULL REFERENCES sys_access_policies(id) ON DELETE CASCADE,
    subject_type    TEXT NOT NULL,                     -- user / party / role / key
    subject_id      TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_sys_apb_policy ON sys_access_policy_bindings(policy_id);
CREATE INDEX idx_sys_apb_subject ON sys_access_policy_bindings(subject_type, subject_id);

COMMENT ON TABLE sys_access_policy_bindings IS '策略→主体绑定 — 将 ABAC 策略关联到用户/组织/角色/Key';

-- 36. sys_ui_menus — 动态菜单树
DROP TABLE IF EXISTS sys_ui_menus;
CREATE TABLE sys_ui_menus (
    id              TEXT PRIMARY KEY,
    menu_code       TEXT NOT NULL UNIQUE,              -- 例: dashboard / models / finance / admin
    parent_id       TEXT REFERENCES sys_ui_menus(id) ON DELETE SET NULL,  -- 自引用父菜单
    label           TEXT NOT NULL,                     -- 显示文本
    icon            TEXT,                              -- 图标标识
    sort_order      INTEGER NOT NULL DEFAULT 0,         -- 同级排序
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_sys_ui_menus_parent ON sys_ui_menus(parent_id);
CREATE INDEX idx_sys_ui_menus_sort ON sys_ui_menus(parent_id, sort_order);

COMMENT ON TABLE sys_ui_menus IS '动态菜单树 — self-ref parent_id 支持无限层级嵌套';
COMMENT ON COLUMN sys_ui_menus.parent_id IS '父菜单 ID，NULL 表示顶级菜单';

-- 37. sys_ui_routes — 前端路由权限映射
DROP TABLE IF EXISTS sys_ui_routes;
CREATE TABLE sys_ui_routes (
    id                  TEXT PRIMARY KEY,
    route_path          TEXT NOT NULL UNIQUE,           -- 例: /dashboard/usage
    menu_id             TEXT REFERENCES sys_ui_menus(id) ON DELETE SET NULL,
    required_action_id  TEXT REFERENCES sys_action_catalogs(id) ON DELETE SET NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_sys_ui_routes_menu ON sys_ui_routes(menu_id);
CREATE INDEX idx_sys_ui_routes_action ON sys_ui_routes(required_action_id);

COMMENT ON TABLE sys_ui_routes IS '前端路由→菜单→操作 权限映射 — 访问路由需要对应操作权限';

-- 38. sys_ui_action_bindings — 页面按钮权限映射
DROP TABLE IF EXISTS sys_ui_action_bindings;
CREATE TABLE sys_ui_action_bindings (
    id                  TEXT PRIMARY KEY,
    button_code         TEXT NOT NULL,                  -- 例: btn-create-api-key
    button_label        TEXT NOT NULL,                  -- 例: 创建 API Key
    page_route          TEXT NOT NULL,                  -- 例: /settings/api-keys
    required_action_id  TEXT REFERENCES sys_action_catalogs(id) ON DELETE CASCADE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(button_code, page_route)
);
CREATE INDEX idx_sys_ui_ab_page ON sys_ui_action_bindings(page_route);
CREATE INDEX idx_sys_ui_ab_action ON sys_ui_action_bindings(required_action_id);

COMMENT ON TABLE sys_ui_action_bindings IS '页面按钮→操作权限映射 — 控制按钮可见性与可用性';

-- ============================================================================
-- 第12组: 审计链锚定 (1表 — PRD v3.2.0 §10 新增)
-- ============================================================================

-- 40. audit_chain_anchors — 审计哈希链锚点
DROP TABLE IF EXISTS audit_chain_anchors;
CREATE TABLE audit_chain_anchors (
    id              TEXT PRIMARY KEY,
    anchor_hash     TEXT NOT NULL UNIQUE,               -- SHA-256 链锚哈希
    start_event_id  TEXT NOT NULL REFERENCES audit_events(id),
    end_event_id    TEXT NOT NULL REFERENCES audit_events(id),
    event_count     INTEGER NOT NULL DEFAULT 0,         -- 链内事件数
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_audit_chain_anchors_hash ON audit_chain_anchors(anchor_hash);
CREATE INDEX idx_audit_chain_anchors_start ON audit_chain_anchors(start_event_id);
CREATE INDEX idx_audit_chain_anchors_end ON audit_chain_anchors(end_event_id);
CREATE INDEX idx_audit_chain_anchors_created ON audit_chain_anchors(created_at);

COMMENT ON TABLE audit_chain_anchors IS '审计哈希链锚点 — 将一段连续审计事件锚定为一个不可篡改的哈希链节点';
COMMENT ON COLUMN audit_chain_anchors.anchor_hash IS 'SHA-256 哈希值，锚定 start_event_id 到 end_event_id 之间的所有事件';
COMMENT ON COLUMN audit_chain_anchors.event_count IS '该锚点覆盖的审计事件数量';

-- ============================================================================
-- 第12组补充: 系统配置 (1表 — PRD v3.2.0 治理基础设施)
-- ============================================================================

-- 39. sys_config — 系统级键值配置
DROP TABLE IF EXISTS sys_config;
CREATE TABLE sys_config (
    id              TEXT PRIMARY KEY,
    config_key      TEXT NOT NULL UNIQUE,              -- 配置键，如 system.default_language
    config_value    TEXT NOT NULL,                     -- 配置值 (字符串序列化)
    value_type      TEXT NOT NULL DEFAULT 'string',    -- string / integer / numeric / boolean / json
    category        TEXT DEFAULT 'general',            -- general / security / billing / routing / ui
    description     TEXT,
    is_public       BOOLEAN NOT NULL DEFAULT FALSE,     -- 是否公开可见 (false=仅管理员)
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_sys_config_key ON sys_config(config_key);
CREATE INDEX idx_sys_config_category ON sys_config(category);

COMMENT ON TABLE sys_config IS '系统级键值配置存储 — 支持多种值类型和分类';
COMMENT ON COLUMN sys_config.config_key IS '唯一配置键，命名规范: category.subcategory.name，如 billing.quota.default_daily_limit';
COMMENT ON COLUMN sys_config.value_type IS '值类型标记，用于应用层反序列化: string / integer / numeric / boolean / json';
COMMENT ON COLUMN sys_config.is_public IS '是否公开可见，false 表示仅管理员可读写';

-- ============================================================================
-- 存储过程: ABAC 策略评估
-- ============================================================================
-- 评估给定主体对资源的访问权限。
-- 规则: deny 优先于 allow，ABAC 策略优先于 grants。
-- 返回 TRUE 表示允许访问，FALSE 表示拒绝。

CREATE OR REPLACE FUNCTION evaluate_access(
    p_principal_type    TEXT,
    p_principal_id      TEXT,
    p_axis              TEXT,
    p_action            TEXT,
    p_resource_type     TEXT DEFAULT NULL,
    p_resource_id       TEXT DEFAULT NULL
) RETURNS BOOLEAN
LANGUAGE plpgsql
AS $$
DECLARE
    v_policy_denied     BOOLEAN := FALSE;
    v_policy_allowed    BOOLEAN := FALSE;
    v_grant_denied      BOOLEAN := FALSE;
    v_grant_allowed     BOOLEAN := FALSE;
BEGIN
    -- ── Step 1: 评估 ABAC 策略 (优先级高于 grants) ──

    -- 检查绑定到该主体的 deny 策略 (按优先级降序)
    SELECT TRUE INTO v_policy_denied
    FROM sys_access_policies p
    JOIN sys_access_policy_bindings b ON b.policy_id = p.id
    WHERE b.subject_type = p_principal_type
      AND b.subject_id = p_principal_id
      AND p.effect = 'deny'
      AND (p.conditions_json->>'axis' IS NULL OR p.conditions_json->>'axis' = p_axis)
      AND (p.conditions_json->>'action' IS NULL OR p.conditions_json->>'action' = p_action)
    ORDER BY p.priority DESC
    LIMIT 1;

    IF v_policy_denied THEN
        RETURN FALSE;
    END IF;

    -- 检查绑定到该主体的 allow 策略 (按优先级降序)
    SELECT TRUE INTO v_policy_allowed
    FROM sys_access_policies p
    JOIN sys_access_policy_bindings b ON b.policy_id = p.id
    WHERE b.subject_type = p_principal_type
      AND b.subject_id = p_principal_id
      AND p.effect = 'allow'
      AND (p.conditions_json->>'axis' IS NULL OR p.conditions_json->>'axis' = p_axis)
      AND (p.conditions_json->>'action' IS NULL OR p.conditions_json->>'action' = p_action)
    ORDER BY p.priority DESC
    LIMIT 1;

    -- ── Step 2: 评估 grants (四轴正交授权) ──

    -- 检查显式 deny
    SELECT TRUE INTO v_grant_denied
    FROM grants
    WHERE principal_type = p_principal_type
      AND principal_id = p_principal_id
      AND axis = p_axis
      AND action = p_action
      AND effect = 'deny'
      AND (resource_type IS NULL OR resource_type = p_resource_type OR resource_type = '*')
      AND (resource_id IS NULL OR resource_id = p_resource_id OR resource_id = '*')
    LIMIT 1;

    IF v_grant_denied THEN
        RETURN FALSE;
    END IF;

    -- 检查显式 allow
    SELECT TRUE INTO v_grant_allowed
    FROM grants
    WHERE principal_type = p_principal_type
      AND principal_id = p_principal_id
      AND axis = p_axis
      AND action = p_action
      AND effect = 'allow'
      AND (resource_type IS NULL OR resource_type = p_resource_type OR resource_type = '*')
      AND (resource_id IS NULL OR resource_id = p_resource_id OR resource_id = '*')
    LIMIT 1;

    -- ── Step 3: 合并结果 ──
    -- ABAC allow 或 grant allow 任一通过即可
    RETURN COALESCE(v_policy_allowed, v_grant_allowed, FALSE);
END;
$$;

COMMENT ON FUNCTION evaluate_access(TEXT, TEXT, TEXT, TEXT, TEXT, TEXT) IS
'ABAC 策略评估函数。
参数:
  p_principal_type  - 主体类型 (user/party/key/role)
  p_principal_id    - 主体 ID
  p_axis            - 治理轴 (data/fund/iam/routing)
  p_action          - 操作 (如 balance.read)
  p_resource_type   - 资源类型 (可选)
  p_resource_id     - 资源 ID (可选)
返回: TRUE=允许 FALSE=拒绝
评估顺序: ABAC deny > ABAC allow > grant deny > grant allow';


-- ============================================================================
-- 存储过程: 通过角色评估访问权限
-- ============================================================================
-- 先解析主体绑定的角色及其权限，再结合 ABAC 策略和 grants 综合判断。

CREATE OR REPLACE FUNCTION evaluate_access_via_roles(
    p_subject_type      TEXT,
    p_subject_id        TEXT,
    p_axis              TEXT,
    p_action            TEXT,
    p_resource_type     TEXT DEFAULT NULL,
    p_resource_id       TEXT DEFAULT NULL,
    p_scope_party_id    TEXT DEFAULT NULL
) RETURNS BOOLEAN
LANGUAGE plpgsql
AS $$
DECLARE
    v_role_allowed      BOOLEAN := FALSE;
    v_policy_denied     BOOLEAN := FALSE;
    v_grant_denied      BOOLEAN := FALSE;
    v_grant_allowed     BOOLEAN := FALSE;
BEGIN
    -- ── Step 1: ABAC 策略 deny 检查 ──
    SELECT TRUE INTO v_policy_denied
    FROM sys_access_policies p
    JOIN sys_access_policy_bindings b ON b.policy_id = p.id
    WHERE b.subject_type = p_subject_type
      AND b.subject_id = p_subject_id
      AND p.effect = 'deny'
    ORDER BY p.priority DESC
    LIMIT 1;

    IF v_policy_denied THEN
        RETURN FALSE;
    END IF;

    -- ── Step 2: 检查 grants deny ──
    SELECT TRUE INTO v_grant_denied
    FROM grants
    WHERE principal_type = p_subject_type
      AND principal_id = p_subject_id
      AND axis = p_axis
      AND action = p_action
      AND effect = 'deny'
    LIMIT 1;

    IF v_grant_denied THEN
        RETURN FALSE;
    END IF;

    -- ── Step 3: 检查角色绑定 (通过 sys_subject_role_bindings → sys_role_permissions) ──
    SELECT TRUE INTO v_role_allowed
    FROM sys_subject_role_bindings srb
    JOIN sys_role_permissions rp ON rp.role_id = srb.role_id
    JOIN sys_action_catalogs ac ON ac.id = rp.action_id
    WHERE srb.subject_type = p_subject_type
      AND srb.subject_id = p_subject_id
      AND ac.action_code = p_action
      AND ac.axis = p_axis
      AND (srb.scope_party_id IS NULL OR srb.scope_party_id = p_scope_party_id)
      AND (srb.valid_from IS NULL OR srb.valid_from <= NOW())
      AND (srb.valid_until IS NULL OR srb.valid_until >= NOW())
    LIMIT 1;

    IF v_role_allowed THEN
        RETURN TRUE;
    END IF;

    -- ── Step 4: 检查 grants allow ──
    SELECT TRUE INTO v_grant_allowed
    FROM grants
    WHERE principal_type = p_subject_type
      AND principal_id = p_subject_id
      AND axis = p_axis
      AND action = p_action
      AND effect = 'allow'
    LIMIT 1;

    RETURN COALESCE(v_grant_allowed, FALSE);
END;
$$;

COMMENT ON FUNCTION evaluate_access_via_roles(TEXT, TEXT, TEXT, TEXT, TEXT, TEXT, TEXT) IS
'基于角色的访问权限评估函数。
先解析主体绑定的角色 → 角色的操作权限，再综合 ABAC 策略和 grants 判断。
scope_party_id 用于限定角色生效的组织范围 (NULL=全局)。';


-- ============================================================================
-- 存储过程: 审计链锚定
-- ============================================================================
-- 将一段连续的审计事件锚定为哈希链节点，用于防篡改验证。

CREATE OR REPLACE FUNCTION anchor_audit_chain(
    p_start_event_id    TEXT,
    p_end_event_id      TEXT
) RETURNS TEXT
LANGUAGE plpgsql
AS $$
DECLARE
    v_event_count   INTEGER;
    v_anchor_hash   TEXT;
    v_anchor_id     TEXT;
    v_concat_text   TEXT;
BEGIN
    -- 统计链内事件数
    SELECT COUNT(*) INTO v_event_count
    FROM audit_events
    WHERE id >= p_start_event_id AND id <= p_end_event_id;

    IF v_event_count = 0 THEN
        RAISE EXCEPTION 'No audit events found in range [% - %]', p_start_event_id, p_end_event_id;
    END IF;

    -- 构建锚定内容: 前一锚点哈希 + 事件ID范围 + 事件计数 + 时间戳
    SELECT COALESCE(
        (SELECT anchor_hash FROM audit_chain_anchors ORDER BY created_at DESC LIMIT 1),
        'GENESIS'
    ) || ':' || p_start_event_id || ':' || p_end_event_id || ':' ||
    v_event_count::TEXT || ':' || NOW()::TEXT
    INTO v_concat_text;

    -- 生成 SHA-256 哈希 (需要 pgcrypto 扩展)
    v_anchor_hash := encode(digest(v_concat_text, 'sha256'), 'hex');

    -- 插入锚点记录
    v_anchor_id := gen_random_uuid()::TEXT;
    INSERT INTO audit_chain_anchors (id, anchor_hash, start_event_id, end_event_id, event_count, created_at)
    VALUES (v_anchor_id, v_anchor_hash, p_start_event_id, p_end_event_id, v_event_count, NOW());

    RETURN v_anchor_id;
END;
$$;

COMMENT ON FUNCTION anchor_audit_chain(TEXT, TEXT) IS
'审计链锚定函数。
将 start_event_id 到 end_event_id 之间的连续审计事件锚定为一个不可篡改的哈希链节点。
依赖: pgcrypto 扩展 (用于 digest 函数)
返回: 新创建的 anchor 记录 ID
异常: 如果范围内无审计事件则抛出异常';


-- ============================================================================
-- 附录 A: model_prices.price_json 结构示例（对齐 PRD §4.4 + AxonHub cost_items）
-- ============================================================================
-- {
--   "items": [
--     {"itemCode": "prompt_tokens",
--      "cost": {"mode": "usage_per_unit", "rate": 0.002},
--      "sell": {"mode": "usage_per_unit", "rate": 0.003}},
--     {"itemCode": "completion_tokens",
--      "cost": {"mode": "usage_per_unit", "rate": 0.008},
--      "sell": {"mode": "usage_per_unit", "rate": 0.012}},
--     {"itemCode": "prompt_cached_tokens",
--      "cost": {"mode": "usage_per_unit", "rate": 0.0005},
--      "sell": {"mode": "usage_per_unit", "rate": 0.00075}},
--     {"itemCode": "prompt_write_cached_tokens",
--      "cost": {"mode": "usage_per_unit", "rate": 0.003},
--      "sell": {"mode": "usage_per_unit", "rate": 0.004}},
--     {"itemCode": "completion_reasoning_tokens",
--      "cost": {"mode": "usage_per_unit", "rate": 0.012},
--      "sell": {"mode": "usage_per_unit", "rate": 0.016}}
--   ],
--   "schedule": {"timezone": "Asia/Shanghai", "overrides": []}
-- }
-- 计价模式: flat_fee / usage_per_unit / usage_tiered / usage_volume (PRD §4.2)

-- ============================================================================
-- 附录 B: usage_records.cost_items JSON 结构示例（对齐 AxonHub cost_items + PRD §4.5）
-- ============================================================================
-- [
--   {"itemCode": "prompt_tokens",        "tokens": 5000,  "cost": 0.010, "sell": 0.015},
--   {"itemCode": "completion_tokens",     "tokens": 1500,  "cost": 0.012, "sell": 0.018},
--   {"itemCode": "prompt_cached_tokens",  "tokens": 2000,  "cost": 0.001, "sell": 0.0015}
-- ]

-- ============================================================================
-- 附录 C: sys_access_policies.conditions_json 结构示例
-- ============================================================================
-- {
--   "axis": "data",
--   "actions": ["model.invoke", "model.list"],
--   "resource_type": "model",
--   "conditions": {
--     "data_classification": ["confidential", "restricted"],
--     "network_class": "external",
--     "time_restriction": {"start": "09:00", "end": "18:00", "timezone": "Asia/Shanghai"}
--   }
-- }
-- deny 策略示例: 禁止在外部网络访问机密/受限数据分类的模型
-- allow 策略示例: 允许在工作时间访问内部数据分类的模型

-- ============================================================================
-- 附录 D: 完整数据字典 (40 表一览)
-- ============================================================================
--  1. users                     用户与身份
--  2. admin_sessions            管理会话
--  3. parties                   统一主体 (org/project)
--  4. party_edges               组织关系边
--  5. party_members             主体成员关系
--  6. accounts                  账本 + 预算帽
--  7. ledgers                   流水日志
--  8. freezes                   冻结/预占
--  9. allocations               划拨记录
-- 10. liquidations              清算状态机
-- 11. api_keys                  API 密钥
-- 12. providers                 供应商
-- 13. provider_resources        供应商资源
-- 14. models                    逻辑模型目录
-- 15. provider_models           供应商模型映射
-- 16. model_prices              双轨价目表
-- 17. model_routes              路由条目
-- 18. route_profiles            策略矩阵档案
-- 19. grants                    四轴正交授权
-- 20. model_grants              模型访问授权
-- 21. request_logs              请求日志
-- 22. request_payload_logs      请求响应体
-- 23. route_attempt_logs        路由尝试日志
-- 24. usage_records             用量记录
-- 25. quota_buckets             Key 配额桶
-- 26. channel_probes            渠道健康探针
-- 27. provider_quota_status     上游配额状态
-- 28. audit_events              审计事件
-- 29. idempotency_records       幂等记录
-- 30. sys_action_catalogs       原子操作目录 [NEW v3.2]
-- 31. sys_roles                 角色定义 [NEW v3.2]
-- 32. sys_role_permissions      角色权限映射 [NEW v3.2]
-- 33. sys_subject_role_bindings 主体角色绑定 [NEW v3.2]
-- 34. sys_access_policies       ABAC 策略 [NEW v3.2]
-- 35. sys_access_policy_bindings 策略绑定 [NEW v3.2]
-- 36. sys_ui_menus              动态菜单树 [NEW v3.2]
-- 37. sys_ui_routes             路由权限映射 [NEW v3.2]
-- 38. sys_ui_action_bindings    按钮权限映射 [NEW v3.2]
-- 39. sys_config                 系统配置 [NEW v3.2]
-- 40. audit_chain_anchors       审计哈希链锚点 [NEW v3.2]
-- ============================================================================
-- 存储过程:
--   evaluate_access()             ABAC + grants 综合策略评估
--   evaluate_access_via_roles()   基于角色的权限评估 (含作用域)
--   anchor_audit_chain()          审计链哈希锚定
-- ============================================================================
