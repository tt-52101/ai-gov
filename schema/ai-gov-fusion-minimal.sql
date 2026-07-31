-- ============================================================================
-- AI-GOV 融合治理平台 - 最小化融合 DDL (SQLite)
-- ============================================================================
-- 基线: TokenHub(30表) + AxonHub(20表) 语义合并吸收
-- 参考: ai-gov.sql(69表) 取其设计思想，去其过度工程化
-- 目标: 27 表覆盖 PRD v2.0.1 全部需求，最短时间交付
-- 日期: 2026-07-31
-- ============================================================================
-- 融合策略:
--   TokenHub 现有表 → 向前兼容扩展（加字段，不改字段）
--   AxonHub 高收益 → itemCode计价/cost_items/渠道探针/配额状态
--   ai-gov 裁减 → 去掉 CQRS/ABAC引擎/复式账本14entry/对账5表/哈希链审计
--   PRD 新增 → parties/party_edges/accounts/ledgers/freezes/model_prices/
--               idempotency_records/model_grants/grants/route_profiles
-- ============================================================================

PRAGMA foreign_keys = ON;

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
    last_login_at   TEXT,                        -- ISO8601
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_email ON users(email);

-- 2. admin_sessions — 管理会话（TokenHub 保留）
DROP TABLE IF EXISTS admin_sessions;
CREATE TABLE admin_sessions (
    token       TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id),
    expires_at  TEXT NOT NULL,
    created_at  TEXT NOT NULL DEFAULT (datetime('now'))
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
    metadata        TEXT,                        -- JSON 扩展
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
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
    allows_fund     INTEGER NOT NULL DEFAULT 0,  -- 是否开通划拨 (parent/sponsors=1)
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
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
    is_primary  INTEGER DEFAULT 0,               -- 主组织标记
    joined_at   TEXT NOT NULL DEFAULT (datetime('now')),
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at  TEXT NOT NULL DEFAULT (datetime('now')),
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
    available_balance       REAL NOT NULL DEFAULT 0,    -- 可用余额 (balance - frozen)
    frozen_balance          REAL NOT NULL DEFAULT 0,    -- 冻结总额
    status                  TEXT NOT NULL DEFAULT 'active', -- active/frozen/liquidating/closed (PRD 8.4)
    -- 预算帽热字段 (PRD §5.3)
    budget_limit_amount     REAL,                       -- NULL=未启用
    budget_warn_ratio       REAL,                       -- 告警比例 如 0.80
    budget_period           TEXT DEFAULT 'none',        -- none/calendar_month/calendar_day/custom
    budget_period_start     TEXT,
    budget_period_end       TEXT,
    budget_consumed_amount  REAL NOT NULL DEFAULT 0,    -- 本周期已确认 sell 累计
    budget_version          INTEGER NOT NULL DEFAULT 0, -- 配置乐观锁
    -- 清算 (PRD §8.4)
    liquidation_stage       TEXT,                       -- block_new/drain/transfer/completed
    liquidation_target_id   TEXT,                       -- 回流目标账户
    liquidation_started_at  TEXT,
    -- 并发控制
    version                 INTEGER NOT NULL DEFAULT 0, -- 乐观锁
    created_at              TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at              TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_accounts_party ON accounts(party_id);
CREATE INDEX idx_accounts_status ON accounts(status);

-- 7. ledgers — 流水（只追加, PRD §2.5 规则5）
DROP TABLE IF EXISTS ledgers;
CREATE TABLE ledgers (
    id              TEXT PRIMARY KEY,
    account_id      TEXT NOT NULL,                     -- 归属账户
    direction       TEXT NOT NULL,                     -- debit/credit/freeze/unfreeze/settle
    amount          REAL NOT NULL,                     -- 变动金额
    balance_after   REAL NOT NULL,                     -- 变动后可用余额快照
    frozen_after    REAL,                              -- 变动后冻结余额快照
    -- 双轨金额 (PRD §4)
    cost_amount     REAL,                              -- 上游成本
    sell_amount     REAL,                              -- 内部结算
    -- 关联
    freeze_id       TEXT,
    request_id      TEXT,
    allocation_id   TEXT,
    user_id         TEXT,                              -- 归属人
    api_key_id      TEXT,
    -- 幂等与审计
    idempotency_key TEXT,
    reason          TEXT,
    created_at      TEXT NOT NULL DEFAULT (datetime('now'))
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
    amount              REAL NOT NULL,                 -- 冻结金额 (PRD 8.1 冻结=max候选sell)
    estimated_sell      REAL NOT NULL,                 -- 预估 sell (候选集最大值)
    status              TEXT NOT NULL DEFAULT 'active',-- active/settled/expired/released
    -- TTL 管理 (PRD 8.3: 默认15min, 可配1-60, 流式续期)
    expires_at          TEXT NOT NULL,                 -- 过期时间
    max_lifetime_at     TEXT,                          -- 流式累计有效期上限(如+2h)
    renewal_count       INTEGER NOT NULL DEFAULT 0,    -- 续期次数
    last_renewed_at     TEXT,                          -- 最近续期时间
    settled_at          TEXT,
    settle_amount       REAL,                          -- 实际结算 sell
    settle_cost         REAL,                          -- 实际成本 cost
    created_at          TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at          TEXT NOT NULL DEFAULT (datetime('now'))
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
    amount          REAL NOT NULL,
    channel         TEXT NOT NULL,                     -- parent/sponsors/whitelist (PRD 8.2)
    edge_id         TEXT,                              -- 关联 party_edge
    status          TEXT NOT NULL DEFAULT 'pending',   -- pending/completed/reverted
    idempotency_key TEXT,
    actor_user_id   TEXT,
    reason          TEXT,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    completed_at    TEXT
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
    initiated_at        TEXT NOT NULL DEFAULT (datetime('now')),
    closed_at           TEXT,
    metadata            TEXT,                           -- JSON
    created_at          TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at          TEXT NOT NULL DEFAULT (datetime('now'))
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
    allowed                 TEXT,                       -- JSON: 允许的 actions
    ip_allowlist            TEXT,                       -- JSON
    limit_daily_requests    INTEGER,
    limit_monthly_requests  INTEGER,
    limit_daily_tokens      INTEGER,
    limit_monthly_tokens    INTEGER,
    limit_daily_cost_usd    REAL,
    limit_monthly_cost_usd  REAL,
    limit_max_concurrency   INTEGER,
    -- === PRD 新增字段 ===
    owner_user_id           TEXT NOT NULL,              -- 归属人 (PRD 2.5: 必须关联实体人)
    account_id              TEXT NOT NULL,              -- 绑定账本 (PRD 2.5: 必须且唯一)
    party_id                TEXT,                       -- 归属主体 (冗余加速)
    -- === AxonHub 扩展字段 ===
    ax_key_type             TEXT DEFAULT 'user',        -- user/service_account/noauth
    scopes                  TEXT,                       -- JSON: AxonHub scopes
    profiles                TEXT,                       -- JSON: AxonHub profiles
    allowed_ips             TEXT,                       -- JSON (与 ip_allowlist 互补)
    -- === 通用 ===
    status                  TEXT NOT NULL DEFAULT 'active', -- active/revoked/expired (PRD KEY-01)
    issued_at               TEXT NOT NULL DEFAULT (datetime('now')),
    expires_at              TEXT,
    last_used_at            TEXT,
    rotated_from_id         TEXT,                       -- 轮换来源
    grace_until             TEXT,                       -- 轮换宽限期
    metadata                TEXT,                       -- JSON
    created_at              TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at              TEXT NOT NULL DEFAULT (datetime('now'))
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
    healthy         INTEGER DEFAULT 1,                 -- TokenHub
    priority        INTEGER DEFAULT 0,
    headers         TEXT,                              -- JSON
    options         TEXT,                              -- JSON
    -- === AxonHub Channel 扩展 ===
    ax_channel_type TEXT,                              -- AxonHub 渠道类型 (60+)
    credentials     TEXT,                              -- JSON: ChannelCredentials (加密)
    supported_models TEXT,                             -- JSON: []string
    policies        TEXT,                              -- JSON: ChannelPolicies
    channel_settings TEXT,                             -- JSON
    endpoints       TEXT,                              -- JSON
    ordering_weight INTEGER DEFAULT 0,
    error_message   TEXT,
    remark          TEXT,
    -- === 融合扩展 ===
    account_id      TEXT,                              -- 上游成本记账账户
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
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
    healthy             INTEGER DEFAULT 1,
    priority            INTEGER DEFAULT 0,
    weight              INTEGER DEFAULT 100,
    rate_limit_rpm      INTEGER,
    token_limit_tpm     INTEGER,
    max_concurrency     INTEGER,
    headers             TEXT,                          -- JSON
    options             TEXT,                          -- JSON
    credential_blob     TEXT,                          -- 加密凭证
    -- 熔断
    failure_count       INTEGER DEFAULT 0,
    cooldown_until      TEXT,
    last_used_at        TEXT,
    last_checked_at     TEXT,
    created_at          TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at          TEXT NOT NULL DEFAULT (datetime('now'))
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
    input_price_per_1m      REAL,                       -- cost 轨道 (上游)
    cache_read_price_per_1m REAL,
    output_price_per_1m     REAL,
    embedding_price_per_1m  REAL,
    input_modalities        TEXT,                       -- JSON
    output_modalities       TEXT,                       -- JSON
    capabilities            TEXT,                       -- JSON: ["chat","streaming","embedding"]
    supported_parameters    TEXT,                       -- JSON
    metadata                TEXT,                       -- JSON
    status                  TEXT NOT NULL DEFAULT 'active',
    -- === AxonHub Model 扩展 ===
    developer               TEXT,                       -- 开发商
    icon                    TEXT,
    model_group             TEXT,                       -- 模型组
    model_card              TEXT,                       -- JSON
    model_settings          TEXT,                       -- JSON
    ax_model_type           TEXT DEFAULT 'chat',        -- chat/embedding/rerank/image/video
    -- === PRD 双轨 sell 价格 (热字段, PRD §4) ===
    sell_input_price_per_1m     REAL,                   -- 内部结算 input
    sell_cache_read_price_per_1m REAL,
    sell_output_price_per_1m    REAL,
    sell_embedding_price_per_1m REAL,
    -- === PRD 扩展 ===
    item_codes              TEXT,                       -- JSON: 支持的 itemCode 列表
    data_classification     TEXT DEFAULT 'internal',    -- public/internal/confidential/restricted
    network_class           TEXT DEFAULT 'external',    -- internal/external
    created_at              TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at              TEXT NOT NULL DEFAULT (datetime('now'))
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
    input_price_per_1m  REAL,
    cache_read_price_per_1m REAL,
    output_price_per_1m REAL,
    input_modalities    TEXT,
    output_modalities   TEXT,
    capabilities        TEXT,
    supported_parameters TEXT,
    metadata            TEXT,
    source              TEXT,                           -- auto_sync/manual
    status              TEXT NOT NULL DEFAULT 'active',
    last_seen_at        TEXT,
    created_at          TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at          TEXT NOT NULL DEFAULT (datetime('now')),
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
    price_json      TEXT NOT NULL,                     -- JSON: PRD 4.4 结构 {items:[{itemCode,cost:{mode,rate},sell:{mode,rate}}], schedule:{timezone,overrides}}
    status          TEXT NOT NULL DEFAULT 'active',    -- active/archived
    effective_start_at TEXT,
    effective_end_at   TEXT,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
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
    sticky_session      INTEGER DEFAULT 0,
    provider_model      TEXT,
    priority            INTEGER DEFAULT 0,
    weight              INTEGER DEFAULT 100,
    quality_score       INTEGER DEFAULT 0,
    cost_score          INTEGER DEFAULT 0,
    status              TEXT NOT NULL DEFAULT 'active',
    strategy            TEXT,                          -- JSON: 策略配置
    last_used_at        TEXT,
    project_scope       TEXT,                          -- all/selected
    project_ids         TEXT,                          -- JSON
    -- === 融合扩展 (PRD §3.3/§8.1) ===
    route_profile_id    TEXT,                          -- → route_profiles.id
    channel_id          TEXT,                          -- 直接绑定渠道
    model_price_id      TEXT,                          -- → model_prices.id
    price_cap_delta     REAL DEFAULT 0.0,              -- δ (PRD 8.1: 默认0, 硬上限0.20)
    created_at          TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at          TEXT NOT NULL DEFAULT (datetime('now'))
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
    strategies_json TEXT NOT NULL DEFAULT '[]',
    delta_cap       REAL NOT NULL DEFAULT 0.0,         -- δ 价格帽 (PRD 8.1)
    max_attempts    INTEGER DEFAULT 3,
    allow_fallback  INTEGER DEFAULT 1,
    status          TEXT NOT NULL DEFAULT 'active',
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
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
    conditions      TEXT,                              -- JSON: 附加条件
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
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
    conditions      TEXT,                              -- JSON
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
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
    stream              INTEGER DEFAULT 0,              -- 是否流式
    format              TEXT DEFAULT 'openai/chat_completions',
    request_body        TEXT,                           -- JSON
    response_body       TEXT,                           -- JSON
    ax_status           TEXT,                           -- pending/processing/completed/failed
    -- 延迟细分 (AxonHub)
    first_token_latency_ms   INTEGER,
    reasoning_duration_ms    INTEGER,
    -- === PRD 双轨扩展 ===
    account_id          TEXT,                           -- 鉴权时锁定, 路由不可改 (PRD 7.5)
    freeze_id           TEXT,                           -- 关联冻结
    user_id             TEXT,                           -- 归属人 (归因)
    party_id            TEXT,                           -- 归属主体
    cost_usd            REAL,                           -- 上游成本
    sell_usd            REAL,                           -- 内部结算
    cost_items          TEXT,                           -- JSON: itemCode 级明细
    usage_incomplete    INTEGER DEFAULT 0,              -- 用量不完整标记 (PRD 4.5)
    business_tags       TEXT,                           -- JSON: 业务标签 (仅归因 PRD 1.3)
    route_profile_id    TEXT,                           -- 使用的策略档案
    created_at          TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at          TEXT NOT NULL DEFAULT (datetime('now'))
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
    request_body        TEXT,
    response_body       TEXT,
    request_truncated   INTEGER DEFAULT 0,
    response_truncated  INTEGER DEFAULT 0,
    created_at          TEXT NOT NULL DEFAULT (datetime('now'))
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
    invoked             INTEGER DEFAULT 1,              -- 是否实际调用
    latency_ms          INTEGER,
    created_at          TEXT NOT NULL DEFAULT (datetime('now'))
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
    cost_usd            REAL,
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
    total_cost          REAL,                           -- AxonHub 原生成本
    cost_price_ref_id   TEXT,                           -- 价目版本引用
    -- === PRD 双轨扩展 ===
    sell_usd            REAL,                           -- 内部结算
    cost_items          TEXT,                           -- JSON: itemCode→{tokens,cost,sell} 明细
    account_id          TEXT,
    freeze_id           TEXT,
    item_code           TEXT,                           -- 主费用项
    created_at          TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at          TEXT NOT NULL DEFAULT (datetime('now'))
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
    cost_usd        REAL DEFAULT 0,
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
    avg_tokens_per_second   REAL,
    avg_first_token_ms      REAL,
    consecutive_failures    INTEGER DEFAULT 0,
    health_status           TEXT DEFAULT 'unknown',    -- healthy/degraded/unhealthy/circuit_open
    reason_code             TEXT,
    evidence_json           TEXT DEFAULT '{}',
    observed_at             TEXT NOT NULL,             -- 观测时间窗
    created_at              TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_channel_probes_ch ON channel_probes(channel_id, observed_at);

-- 27. provider_quota_status — 上游配额状态 (AxonHub)
DROP TABLE IF EXISTS provider_quota_status;
CREATE TABLE provider_quota_status (
    id              TEXT PRIMARY KEY,
    channel_id      TEXT NOT NULL UNIQUE,              -- → provider_resources.id
    provider_type   TEXT,
    status          TEXT NOT NULL DEFAULT 'available', -- available/warning/exhausted/unknown
    quota_data      TEXT,                              -- JSON: 配额详情
    next_reset_at   TEXT,
    ready           INTEGER DEFAULT 1,
    next_check_at   TEXT NOT NULL,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
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
    before_snapshot TEXT,                              -- JSON: 变更前
    after_snapshot  TEXT,                              -- JSON: 变更后
    ip              TEXT,
    user_agent      TEXT,
    created_at      TEXT NOT NULL DEFAULT (datetime('now'))
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
    response_json   TEXT,                              -- 首次成功响应
    resource_ref    TEXT,                              -- 如 transaction_id
    expires_at      TEXT NOT NULL,                     -- 窗口 ≥24h
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(scope, actor_id, idempotency_key)
);
CREATE INDEX idx_idempotency_expiry ON idempotency_records(expires_at);

-- ============================================================================
-- 附录: model_prices.price_json 结构示例（对齐 PRD §4.4 + AxonHub cost_items）
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
-- usage_records.cost_items JSON 结构示例（对齐 AxonHub cost_items + PRD §4.5）
-- ============================================================================
-- [
--   {"itemCode": "prompt_tokens",        "tokens": 5000,  "cost": 0.010, "sell": 0.015},
--   {"itemCode": "completion_tokens",     "tokens": 1500,  "cost": 0.012, "sell": 0.018},
--   {"itemCode": "prompt_cached_tokens",  "tokens": 2000,  "cost": 0.001, "sell": 0.0015}
-- ]

-- ============================================================================
-- 设计决策: 为什么只有 29 表（对比 ai-gov 69 表 / 融合方案 56 表）
-- ============================================================================
-- 剪裁项                          | ai-gov 表数 | 理由
-- ---------------------------------|------------|----------------------------------
-- 复式账本 6 种 account_type       | 6→1        | accounts 单表多态
-- ledgers 14 种 entry_type          | 遗弃        | direction 枚举替代
-- CQRS balance_projections          | 1→0        | 直接从 ledger 计算余额
-- ABAC 引擎 (6表)                   | 6→1        | grants 表覆盖
-- 对账系统 (5表)                    | 5→0        | MVP 不做, P2 阶段补
-- 哈希链审计 (2表)                  | 2→0        | audit_events 简化版
-- 审批工作流 (2表)                  | 2→0        | PRD 不在范围
-- 复杂身份 (9表)                    | 9→1        | users 单表
-- UI 导航 (3表)                     | 3→0        | 前端自行管理
-- 紧急信用 (2表)                    | 2→0        | MVP 不做
-- 日结快照 (2表)                    | 2→0        | P2 阶段补
-- 事件总线 outbox/inbox (2表)       | 2→0        | MVP 不做
-- 基础设施 (5表)                    | 5→0        | 去掉过度工程化
-- AxonHub 非核心 (invitations/      | 6→0        | 提示词库/邀请等不在 PRD
--   prompts/prompt_rules/systems/   |            |
--   templates/data_storages/        |            |
--   threads/traces)                 |            |
-- ---------------------------------|------------|----------------------------------
-- 总计                             | 69→29      | 剪裁 40 表
-- ============================================================================

PRAGMA foreign_keys = ON;
