// ============================================================================
// E2E-1: 数据库建表 + 种子数据测试
// 铁律：零 Mock、真实 SQLite :memory: 数据库、基于事实
// ============================================================================
import Database from 'better-sqlite3';
import { randomUUID } from 'crypto';
import { writeFileSync, mkdirSync, existsSync } from 'fs';
import { join, dirname } from 'path';

// ── Helpers ──────────────────────────────────────────────────────────────────
const uid = () => randomUUID();
const nowISO = () => new Date().toISOString();
const laterISO = (hours = 1) => new Date(Date.now() + hours * 3600_000).toISOString();

// ── Initialize SQLite ────────────────────────────────────────────────────────
const db = new Database(':memory:');
db.pragma('journal_mode = WAL');
db.pragma('foreign_keys = ON');

const results = { tables: [], seeds: {}, constraints: { passed: [], failed: [] }, indexes: { passed: [], failed: [] }, balances: { passed: [], failed: [] } };

function log(level, msg) {
  const ts = new Date().toISOString().slice(11, 23);
  console.log(`[${ts}] [${level}] ${msg}`);
}

// ============================================================================
// STEP 1: 创建全部 40 表 (SQLite 方言适配)
// ============================================================================
log('INFO', '=== STEP 1: 创建 40 张表 ===');

const DDL_STATEMENTS = [
  // ── 第1组: 用户与身份 (2表) ──
  `CREATE TABLE users (
    id              TEXT PRIMARY KEY,
    username        TEXT NOT NULL UNIQUE,
    email           TEXT,
    display_name    TEXT,
    password_hash   TEXT,
    role            TEXT NOT NULL DEFAULT 'member',
    status          TEXT NOT NULL DEFAULT 'active',
    oidc_issuer     TEXT,
    oidc_subject    TEXT,
    avatar          TEXT,
    prefer_language TEXT DEFAULT 'zh-CN',
    last_login_at   TEXT,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
  )`,
  `CREATE INDEX idx_users_status ON users(status)`,
  `CREATE INDEX idx_users_email ON users(email)`,

  `CREATE TABLE admin_sessions (
    token       TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id),
    expires_at  TEXT NOT NULL,
    created_at  TEXT NOT NULL DEFAULT (datetime('now'))
  )`,
  `CREATE INDEX idx_admin_sessions_user_id ON admin_sessions(user_id)`,

  // ── 第2组: Party 统一主体模型 (3表) ──
  `CREATE TABLE parties (
    id              TEXT PRIMARY KEY,
    type            TEXT NOT NULL,
    name            TEXT NOT NULL,
    description     TEXT DEFAULT '',
    parent_party_id TEXT,
    leader_user_id  TEXT,
    cost_center     TEXT,
    status          TEXT NOT NULL DEFAULT 'active',
    metadata        TEXT,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
  )`,
  `CREATE INDEX idx_parties_type ON parties(type)`,
  `CREATE INDEX idx_parties_parent ON parties(parent_party_id)`,
  `CREATE INDEX idx_parties_status ON parties(status)`,

  `CREATE TABLE party_edges (
    id              TEXT PRIMARY KEY,
    src_party_id    TEXT NOT NULL,
    dst_party_id    TEXT NOT NULL,
    edge_type       TEXT NOT NULL,
    allows_fund     INTEGER NOT NULL DEFAULT 0,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(src_party_id, dst_party_id, edge_type)
  )`,
  `CREATE INDEX idx_party_edges_src ON party_edges(src_party_id)`,
  `CREATE INDEX idx_party_edges_dst ON party_edges(dst_party_id)`,

  `CREATE TABLE party_members (
    id          TEXT PRIMARY KEY,
    party_id    TEXT NOT NULL,
    user_id     TEXT NOT NULL,
    role        TEXT NOT NULL DEFAULT 'member',
    is_primary  INTEGER DEFAULT 0,
    joined_at   TEXT NOT NULL DEFAULT (datetime('now')),
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at  TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(party_id, user_id)
  )`,
  `CREATE INDEX idx_party_members_party ON party_members(party_id)`,
  `CREATE INDEX idx_party_members_user ON party_members(user_id)`,

  // ── 第3组: 资金治理 (5表) ──
  `CREATE TABLE accounts (
    id                      TEXT PRIMARY KEY,
    party_id                TEXT NOT NULL UNIQUE,
    available_balance       REAL NOT NULL DEFAULT 0,
    frozen_balance          REAL NOT NULL DEFAULT 0,
    status                  TEXT NOT NULL DEFAULT 'active',
    budget_limit_amount     REAL,
    budget_warn_ratio       REAL,
    budget_period           TEXT DEFAULT 'none',
    budget_period_start     TEXT,
    budget_period_end       TEXT,
    budget_consumed_amount  REAL NOT NULL DEFAULT 0,
    budget_version          INTEGER NOT NULL DEFAULT 0,
    liquidation_stage       TEXT,
    liquidation_target_id   TEXT,
    liquidation_started_at  TEXT,
    version                 INTEGER NOT NULL DEFAULT 0,
    created_at              TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at              TEXT NOT NULL DEFAULT (datetime('now'))
  )`,
  `CREATE INDEX idx_accounts_party ON accounts(party_id)`,
  `CREATE INDEX idx_accounts_status ON accounts(status)`,

  `CREATE TABLE ledgers (
    id              TEXT PRIMARY KEY,
    account_id      TEXT NOT NULL,
    direction       TEXT NOT NULL,
    amount          REAL NOT NULL,
    balance_after   REAL NOT NULL,
    frozen_after    REAL,
    cost_amount     REAL,
    sell_amount     REAL,
    freeze_id       TEXT,
    request_id      TEXT,
    allocation_id   TEXT,
    user_id         TEXT,
    api_key_id      TEXT,
    idempotency_key TEXT,
    reason          TEXT,
    created_at      TEXT NOT NULL DEFAULT (datetime('now'))
  )`,
  `CREATE INDEX idx_ledgers_account ON ledgers(account_id, created_at)`,
  `CREATE INDEX idx_ledgers_freeze ON ledgers(freeze_id)`,
  `CREATE INDEX idx_ledgers_request ON ledgers(request_id)`,
  `CREATE INDEX idx_ledgers_idem ON ledgers(account_id, idempotency_key)`,

  `CREATE TABLE freezes (
    id                  TEXT PRIMARY KEY,
    account_id          TEXT NOT NULL,
    request_id          TEXT,
    api_key_id          TEXT,
    user_id             TEXT NOT NULL,
    amount              REAL NOT NULL,
    estimated_sell      REAL NOT NULL,
    status              TEXT NOT NULL DEFAULT 'active',
    expires_at          TEXT NOT NULL,
    max_lifetime_at     TEXT,
    renewal_count       INTEGER NOT NULL DEFAULT 0,
    last_renewed_at     TEXT,
    settled_at          TEXT,
    settle_amount       REAL,
    settle_cost         REAL,
    created_at          TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at          TEXT NOT NULL DEFAULT (datetime('now'))
  )`,
  `CREATE INDEX idx_freezes_account ON freezes(account_id)`,
  `CREATE INDEX idx_freezes_request ON freezes(request_id)`,
  `CREATE INDEX idx_freezes_expiry ON freezes(status, expires_at)`,
  `CREATE INDEX idx_freezes_user ON freezes(user_id)`,

  `CREATE TABLE allocations (
    id              TEXT PRIMARY KEY,
    src_account_id  TEXT NOT NULL,
    dst_account_id  TEXT NOT NULL,
    amount          REAL NOT NULL,
    channel         TEXT NOT NULL,
    edge_id         TEXT,
    status          TEXT NOT NULL DEFAULT 'pending',
    idempotency_key TEXT,
    actor_user_id   TEXT,
    reason          TEXT,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    completed_at    TEXT
  )`,
  `CREATE INDEX idx_allocations_src ON allocations(src_account_id)`,
  `CREATE INDEX idx_allocations_dst ON allocations(dst_account_id)`,
  `CREATE INDEX idx_allocations_idem ON allocations(src_account_id, idempotency_key)`,

  `CREATE TABLE liquidations (
    id                  TEXT PRIMARY KEY,
    party_id            TEXT NOT NULL,
    account_id          TEXT NOT NULL,
    target_account_id   TEXT,
    status              TEXT NOT NULL DEFAULT 'blocking',
    initiated_by        TEXT NOT NULL,
    initiated_at        TEXT NOT NULL DEFAULT (datetime('now')),
    closed_at           TEXT,
    metadata            TEXT,
    created_at          TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at          TEXT NOT NULL DEFAULT (datetime('now'))
  )`,
  `CREATE INDEX idx_liquidations_party ON liquidations(party_id)`,
  `CREATE INDEX idx_liquidations_account ON liquidations(account_id)`,
  `CREATE INDEX idx_liquidations_status ON liquidations(status)`,

  // ── 第4组: API Key ──
  `CREATE TABLE api_keys (
    id                      TEXT PRIMARY KEY,
    project_id              TEXT,
    name                    TEXT NOT NULL,
    key_hash                TEXT NOT NULL UNIQUE,
    key_prefix              TEXT,
    key_suffix              TEXT,
    allowed                 TEXT,
    ip_allowlist            TEXT,
    limit_daily_requests    INTEGER,
    limit_monthly_requests  INTEGER,
    limit_daily_tokens      INTEGER,
    limit_monthly_tokens    INTEGER,
    limit_daily_cost_usd    REAL,
    limit_monthly_cost_usd  REAL,
    limit_max_concurrency   INTEGER,
    owner_user_id           TEXT NOT NULL,
    account_id              TEXT NOT NULL,
    party_id                TEXT,
    ax_key_type             TEXT DEFAULT 'user',
    scopes                  TEXT,
    profiles                TEXT,
    allowed_ips             TEXT,
    status                  TEXT NOT NULL DEFAULT 'active',
    issued_at               TEXT NOT NULL DEFAULT (datetime('now')),
    expires_at              TEXT,
    last_used_at            TEXT,
    rotated_from_id         TEXT,
    grace_until             TEXT,
    metadata                TEXT,
    created_at              TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at              TEXT NOT NULL DEFAULT (datetime('now'))
  )`,
  `CREATE INDEX idx_api_keys_owner ON api_keys(owner_user_id)`,
  `CREATE INDEX idx_api_keys_account ON api_keys(account_id)`,
  `CREATE INDEX idx_api_keys_party ON api_keys(party_id)`,
  `CREATE INDEX idx_api_keys_project ON api_keys(project_id)`,
  `CREATE INDEX idx_api_keys_status ON api_keys(status)`,
  `CREATE INDEX idx_api_keys_hash ON api_keys(key_hash)`,

  // ── 第5组: 模型目录与供应商 (4表) ──
  `CREATE TABLE providers (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL UNIQUE,
    type            TEXT NOT NULL,
    base_url        TEXT,
    api_key         TEXT,
    status          TEXT NOT NULL DEFAULT 'active',
    healthy         INTEGER DEFAULT 1,
    priority        INTEGER DEFAULT 0,
    headers         TEXT,
    options         TEXT,
    ax_channel_type TEXT,
    credentials     TEXT,
    supported_models TEXT,
    policies        TEXT,
    channel_settings TEXT,
    endpoints       TEXT,
    ordering_weight INTEGER DEFAULT 0,
    error_message   TEXT,
    remark          TEXT,
    account_id      TEXT,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
  )`,
  `CREATE INDEX idx_providers_type ON providers(type)`,
  `CREATE INDEX idx_providers_status ON providers(status)`,

  `CREATE TABLE provider_resources (
    id                  TEXT PRIMARY KEY,
    provider_id         TEXT NOT NULL,
    name                TEXT NOT NULL,
    resource_group      TEXT,
    resource_type       TEXT DEFAULT 'llm',
    base_url            TEXT,
    api_key             TEXT,
    region              TEXT,
    environment         TEXT,
    status              TEXT NOT NULL DEFAULT 'active',
    healthy             INTEGER DEFAULT 1,
    priority            INTEGER DEFAULT 0,
    weight              INTEGER DEFAULT 100,
    rate_limit_rpm      INTEGER,
    token_limit_tpm     INTEGER,
    max_concurrency     INTEGER,
    headers             TEXT,
    options             TEXT,
    credential_blob     TEXT,
    failure_count       INTEGER DEFAULT 0,
    cooldown_until      TEXT,
    last_used_at        TEXT,
    last_checked_at     TEXT,
    created_at          TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at          TEXT NOT NULL DEFAULT (datetime('now'))
  )`,
  `CREATE INDEX idx_prov_res_provider ON provider_resources(provider_id)`,
  `CREATE INDEX idx_prov_res_group ON provider_resources(resource_group)`,
  `CREATE INDEX idx_prov_res_status ON provider_resources(status)`,

  `CREATE TABLE models (
    id                      TEXT PRIMARY KEY,
    name                    TEXT NOT NULL UNIQUE,
    category                TEXT,
    family                  TEXT,
    modality                TEXT,
    context_window          INTEGER,
    input_price_per_1m      REAL,
    cache_read_price_per_1m REAL,
    output_price_per_1m     REAL,
    embedding_price_per_1m  REAL,
    input_modalities        TEXT,
    output_modalities       TEXT,
    capabilities            TEXT,
    supported_parameters    TEXT,
    metadata                TEXT,
    status                  TEXT NOT NULL DEFAULT 'active',
    developer               TEXT,
    icon                    TEXT,
    model_group             TEXT,
    model_card              TEXT,
    model_settings          TEXT,
    ax_model_type           TEXT DEFAULT 'chat',
    sell_input_price_per_1m     REAL,
    sell_cache_read_price_per_1m REAL,
    sell_output_price_per_1m    REAL,
    sell_embedding_price_per_1m REAL,
    item_codes              TEXT,
    data_classification     TEXT DEFAULT 'internal',
    network_class           TEXT DEFAULT 'external',
    created_at              TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at              TEXT NOT NULL DEFAULT (datetime('now'))
  )`,
  `CREATE INDEX idx_models_category ON models(category)`,
  `CREATE INDEX idx_models_status ON models(status)`,
  `CREATE INDEX idx_models_developer ON models(developer)`,

  `CREATE TABLE provider_models (
    id                  TEXT PRIMARY KEY,
    provider_id         TEXT NOT NULL,
    upstream_model      TEXT NOT NULL,
    display_name        TEXT,
    canonical_name      TEXT,
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
    source              TEXT,
    status              TEXT NOT NULL DEFAULT 'active',
    last_seen_at        TEXT,
    created_at          TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at          TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(provider_id, upstream_model)
  )`,
  `CREATE INDEX idx_prov_models_provider ON provider_models(provider_id)`,
  `CREATE INDEX idx_prov_models_canonical ON provider_models(canonical_name)`,

  // ── 第6组: 定价与路由 (3表) ──
  `CREATE TABLE model_prices (
    id              TEXT PRIMARY KEY,
    model_id        TEXT NOT NULL,
    channel_id      TEXT,
    reference_id    TEXT NOT NULL UNIQUE,
    price_json      TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'active',
    effective_start_at TEXT,
    effective_end_at   TEXT,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
  )`,
  `CREATE INDEX idx_model_prices_model ON model_prices(model_id, channel_id)`,
  `CREATE INDEX idx_model_prices_ref ON model_prices(reference_id)`,

  `CREATE TABLE model_routes (
    id                  TEXT PRIMARY KEY,
    model_name          TEXT NOT NULL,
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
    strategy            TEXT,
    last_used_at        TEXT,
    project_scope       TEXT,
    project_ids         TEXT,
    route_profile_id    TEXT,
    channel_id          TEXT,
    model_price_id      TEXT,
    price_cap_delta     REAL DEFAULT 0.0,
    created_at          TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at          TEXT NOT NULL DEFAULT (datetime('now'))
  )`,
  `CREATE INDEX idx_model_routes_model ON model_routes(model_name)`,
  `CREATE INDEX idx_model_routes_provider ON model_routes(provider_id)`,
  `CREATE INDEX idx_model_routes_resource ON model_routes(provider_resource_id)`,
  `CREATE INDEX idx_model_routes_profile ON model_routes(route_profile_id)`,

  `CREATE TABLE route_profiles (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL UNIQUE,
    description     TEXT,
    strategies_json TEXT NOT NULL DEFAULT '[]',
    delta_cap       REAL NOT NULL DEFAULT 0.0,
    max_attempts    INTEGER DEFAULT 3,
    allow_fallback  INTEGER DEFAULT 1,
    status          TEXT NOT NULL DEFAULT 'active',
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
  )`,

  // ── 第7组: 授权治理 (2表) ──
  `CREATE TABLE grants (
    id              TEXT PRIMARY KEY,
    principal_type  TEXT NOT NULL,
    principal_id    TEXT NOT NULL,
    axis            TEXT NOT NULL,
    action          TEXT NOT NULL,
    resource_type   TEXT,
    resource_id     TEXT,
    effect          TEXT NOT NULL DEFAULT 'allow',
    conditions      TEXT,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
  )`,
  `CREATE INDEX idx_grants_principal ON grants(principal_type, principal_id)`,
  `CREATE INDEX idx_grants_axis ON grants(axis, action)`,
  `CREATE INDEX idx_grants_resource ON grants(resource_type, resource_id)`,

  `CREATE TABLE model_grants (
    id              TEXT PRIMARY KEY,
    principal_type  TEXT NOT NULL,
    principal_id    TEXT NOT NULL,
    model_id        TEXT,
    model_tag       TEXT,
    effect          TEXT NOT NULL,
    priority        INTEGER DEFAULT 0,
    quota_limit     INTEGER,
    conditions      TEXT,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
  )`,
  `CREATE INDEX idx_model_grants_principal ON model_grants(principal_type, principal_id)`,
  `CREATE INDEX idx_model_grants_model ON model_grants(model_id)`,
  `CREATE INDEX idx_model_grants_effect ON model_grants(effect)`,

  // ── 第8组: 请求与用量 (5表) ──
  `CREATE TABLE request_logs (
    id                  TEXT PRIMARY KEY,
    request_id          TEXT NOT NULL UNIQUE,
    project_id          TEXT,
    api_key_id          TEXT,
    model_name          TEXT,
    provider_id         TEXT,
    provider_resource_id TEXT,
    provider_model      TEXT,
    upstream_request_id TEXT,
    served_model        TEXT,
    model_e_tag         TEXT,
    transport           TEXT DEFAULT 'https',
    status_code         INTEGER,
    error_code          TEXT,
    latency_ms          INTEGER,
    client_ip           TEXT,
    user_agent          TEXT,
    stream              INTEGER DEFAULT 0,
    format              TEXT DEFAULT 'openai/chat_completions',
    request_body        TEXT,
    response_body       TEXT,
    ax_status           TEXT,
    first_token_latency_ms   INTEGER,
    reasoning_duration_ms    INTEGER,
    account_id          TEXT,
    freeze_id           TEXT,
    user_id             TEXT,
    party_id            TEXT,
    cost_usd            REAL,
    sell_usd            REAL,
    cost_items          TEXT,
    usage_incomplete    INTEGER DEFAULT 0,
    business_tags       TEXT,
    route_profile_id    TEXT,
    created_at          TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at          TEXT NOT NULL DEFAULT (datetime('now'))
  )`,
  `CREATE INDEX idx_request_logs_req ON request_logs(request_id)`,
  `CREATE INDEX idx_request_logs_key ON request_logs(api_key_id, created_at)`,
  `CREATE INDEX idx_request_logs_project ON request_logs(project_id, created_at)`,
  `CREATE INDEX idx_request_logs_account ON request_logs(account_id, created_at)`,
  `CREATE INDEX idx_request_logs_user ON request_logs(user_id, created_at)`,
  `CREATE INDEX idx_request_logs_model ON request_logs(model_name, created_at)`,
  `CREATE INDEX idx_request_logs_error ON request_logs(error_code, created_at)`,

  `CREATE TABLE request_payload_logs (
    id                  TEXT PRIMARY KEY,
    request_id          TEXT NOT NULL UNIQUE,
    request_body        TEXT,
    response_body       TEXT,
    request_truncated   INTEGER DEFAULT 0,
    response_truncated  INTEGER DEFAULT 0,
    created_at          TEXT NOT NULL DEFAULT (datetime('now'))
  )`,

  `CREATE TABLE route_attempt_logs (
    id                  TEXT PRIMARY KEY,
    request_id          TEXT NOT NULL,
    attempt_index       INTEGER NOT NULL,
    route_id            TEXT,
    provider_id         TEXT,
    provider_resource_id TEXT,
    provider_model      TEXT,
    status_code         INTEGER,
    error_code          TEXT,
    error_message       TEXT,
    invoked             INTEGER DEFAULT 1,
    latency_ms          INTEGER,
    created_at          TEXT NOT NULL DEFAULT (datetime('now'))
  )`,
  `CREATE INDEX idx_route_attempt_req ON route_attempt_logs(request_id)`,
  `CREATE INDEX idx_route_attempt_route ON route_attempt_logs(route_id, invoked, created_at)`,

  `CREATE TABLE usage_records (
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
    channel_id          TEXT,
    prompt_audio_tokens         INTEGER DEFAULT 0,
    prompt_cached_tokens        INTEGER DEFAULT 0,
    prompt_write_cached_tokens  INTEGER DEFAULT 0,
    prompt_write_cached_5m      INTEGER DEFAULT 0,
    prompt_write_cached_1h      INTEGER DEFAULT 0,
    completion_audio_tokens     INTEGER DEFAULT 0,
    completion_reasoning_tokens INTEGER DEFAULT 0,
    accepted_prediction_tokens  INTEGER DEFAULT 0,
    rejected_prediction_tokens  INTEGER DEFAULT 0,
    source              TEXT DEFAULT 'api',
    format              TEXT DEFAULT 'openai/chat_completions',
    total_cost          REAL,
    cost_price_ref_id   TEXT,
    sell_usd            REAL,
    cost_items          TEXT,
    account_id          TEXT,
    freeze_id           TEXT,
    item_code           TEXT,
    created_at          TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at          TEXT NOT NULL DEFAULT (datetime('now'))
  )`,
  `CREATE INDEX idx_usage_records_req ON usage_records(request_id)`,
  `CREATE INDEX idx_usage_records_key ON usage_records(api_key_id, created_at)`,
  `CREATE INDEX idx_usage_records_project ON usage_records(project_id, created_at)`,
  `CREATE INDEX idx_usage_records_model ON usage_records(model_name, created_at)`,
  `CREATE INDEX idx_usage_records_user ON usage_records(attributed_user_id)`,
  `CREATE INDEX idx_usage_records_account ON usage_records(account_id)`,

  `CREATE TABLE quota_buckets (
    key_id          TEXT NOT NULL,
    scope           TEXT NOT NULL,
    bucket          TEXT NOT NULL,
    requests        INTEGER DEFAULT 0,
    prompt_tokens   INTEGER DEFAULT 0,
    completion_tokens INTEGER DEFAULT 0,
    total_tokens    INTEGER DEFAULT 0,
    cost_usd        REAL DEFAULT 0,
    PRIMARY KEY (key_id, scope, bucket)
  )`,

  // ── 第9组: 可观测与调度 (2表) ──
  `CREATE TABLE channel_probes (
    id                      TEXT PRIMARY KEY,
    channel_id              TEXT NOT NULL,
    total_request_count     INTEGER NOT NULL DEFAULT 0,
    success_request_count   INTEGER NOT NULL DEFAULT 0,
    avg_tokens_per_second   REAL,
    avg_first_token_ms      REAL,
    consecutive_failures    INTEGER DEFAULT 0,
    health_status           TEXT DEFAULT 'unknown',
    reason_code             TEXT,
    evidence_json           TEXT DEFAULT '{}',
    observed_at             TEXT NOT NULL,
    created_at              TEXT NOT NULL DEFAULT (datetime('now'))
  )`,
  `CREATE INDEX idx_channel_probes_ch ON channel_probes(channel_id, observed_at)`,

  `CREATE TABLE provider_quota_status (
    id              TEXT PRIMARY KEY,
    channel_id      TEXT NOT NULL UNIQUE,
    provider_type   TEXT,
    status          TEXT NOT NULL DEFAULT 'available',
    quota_data      TEXT,
    next_reset_at   TEXT,
    ready           INTEGER DEFAULT 1,
    next_check_at   TEXT NOT NULL,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
  )`,
  `CREATE INDEX idx_prov_quota_status ON provider_quota_status(status)`,
  `CREATE INDEX idx_prov_quota_check ON provider_quota_status(next_check_at)`,

  // ── 第10组: 基础设施 (2表) ──
  `CREATE TABLE audit_events (
    id              TEXT PRIMARY KEY,
    actor_user_id   TEXT,
    actor_name      TEXT,
    action          TEXT NOT NULL,
    resource_type   TEXT NOT NULL,
    resource_id     TEXT NOT NULL,
    status          TEXT,
    message         TEXT,
    before_snapshot TEXT,
    after_snapshot  TEXT,
    ip              TEXT,
    user_agent      TEXT,
    created_at      TEXT NOT NULL DEFAULT (datetime('now'))
  )`,
  `CREATE INDEX idx_audit_events_actor ON audit_events(actor_user_id, created_at)`,
  `CREATE INDEX idx_audit_events_resource ON audit_events(resource_type, resource_id, created_at)`,
  `CREATE INDEX idx_audit_events_action ON audit_events(action, created_at)`,

  `CREATE TABLE idempotency_records (
    id              TEXT PRIMARY KEY,
    scope           TEXT NOT NULL,
    actor_id        TEXT NOT NULL,
    idempotency_key TEXT NOT NULL,
    request_hash    TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'started',
    response_json   TEXT,
    resource_ref    TEXT,
    expires_at      TEXT NOT NULL,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(scope, actor_id, idempotency_key)
  )`,
  `CREATE INDEX idx_idempotency_expiry ON idempotency_records(expires_at)`,

  // ── 第11组: ABAC 安全治理 (9表) ──
  `CREATE TABLE sys_action_catalogs (
    id              TEXT PRIMARY KEY,
    action_code     TEXT NOT NULL UNIQUE,
    action_name     TEXT NOT NULL,
    axis            TEXT NOT NULL,
    resource_type   TEXT,
    description     TEXT,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
  )`,
  `CREATE INDEX idx_sys_action_catalogs_axis ON sys_action_catalogs(axis)`,
  `CREATE INDEX idx_sys_action_catalogs_resource ON sys_action_catalogs(resource_type)`,

  `CREATE TABLE sys_roles (
    id              TEXT PRIMARY KEY,
    role_code       TEXT NOT NULL UNIQUE,
    role_name       TEXT NOT NULL,
    description     TEXT,
    is_system       INTEGER NOT NULL DEFAULT 0,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
  )`,
  `CREATE INDEX idx_sys_roles_code ON sys_roles(role_code)`,

  `CREATE TABLE sys_role_permissions (
    id              TEXT PRIMARY KEY,
    role_id         TEXT NOT NULL REFERENCES sys_roles(id) ON DELETE CASCADE,
    action_id       TEXT NOT NULL REFERENCES sys_action_catalogs(id) ON DELETE CASCADE,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(role_id, action_id)
  )`,
  `CREATE INDEX idx_sys_role_perms_role ON sys_role_permissions(role_id)`,
  `CREATE INDEX idx_sys_role_perms_action ON sys_role_permissions(action_id)`,

  `CREATE TABLE sys_subject_role_bindings (
    id              TEXT PRIMARY KEY,
    subject_type    TEXT NOT NULL,
    subject_id      TEXT NOT NULL,
    role_id         TEXT NOT NULL REFERENCES sys_roles(id) ON DELETE CASCADE,
    scope_party_id  TEXT,
    valid_from      TEXT,
    valid_until     TEXT,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
  )`,
  `CREATE INDEX idx_sys_srb_subject ON sys_subject_role_bindings(subject_type, subject_id)`,
  `CREATE INDEX idx_sys_srb_role ON sys_subject_role_bindings(role_id)`,
  `CREATE INDEX idx_sys_srb_scope ON sys_subject_role_bindings(scope_party_id)`,
  `CREATE INDEX idx_sys_srb_validity ON sys_subject_role_bindings(valid_from, valid_until)`,

  `CREATE TABLE sys_access_policies (
    id              TEXT PRIMARY KEY,
    policy_code     TEXT NOT NULL UNIQUE,
    policy_name     TEXT NOT NULL,
    effect          TEXT NOT NULL DEFAULT 'allow',
    conditions_json TEXT NOT NULL DEFAULT '{}',
    priority        INTEGER NOT NULL DEFAULT 0,
    is_system       INTEGER NOT NULL DEFAULT 0,
    description     TEXT,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
  )`,
  `CREATE INDEX idx_sys_access_policies_code ON sys_access_policies(policy_code)`,
  `CREATE INDEX idx_sys_access_policies_effect ON sys_access_policies(effect)`,
  `CREATE INDEX idx_sys_access_policies_priority ON sys_access_policies(priority DESC)`,

  `CREATE TABLE sys_access_policy_bindings (
    id              TEXT PRIMARY KEY,
    policy_id       TEXT NOT NULL REFERENCES sys_access_policies(id) ON DELETE CASCADE,
    subject_type    TEXT NOT NULL,
    subject_id      TEXT NOT NULL,
    created_at      TEXT NOT NULL DEFAULT (datetime('now'))
  )`,
  `CREATE INDEX idx_sys_apb_policy ON sys_access_policy_bindings(policy_id)`,
  `CREATE INDEX idx_sys_apb_subject ON sys_access_policy_bindings(subject_type, subject_id)`,

  `CREATE TABLE sys_ui_menus (
    id              TEXT PRIMARY KEY,
    menu_code       TEXT NOT NULL UNIQUE,
    parent_id       TEXT REFERENCES sys_ui_menus(id) ON DELETE SET NULL,
    label           TEXT NOT NULL,
    icon            TEXT,
    sort_order      INTEGER NOT NULL DEFAULT 0,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
  )`,
  `CREATE INDEX idx_sys_ui_menus_parent ON sys_ui_menus(parent_id)`,
  `CREATE INDEX idx_sys_ui_menus_sort ON sys_ui_menus(parent_id, sort_order)`,

  `CREATE TABLE sys_ui_routes (
    id                  TEXT PRIMARY KEY,
    route_path          TEXT NOT NULL UNIQUE,
    menu_id             TEXT REFERENCES sys_ui_menus(id) ON DELETE SET NULL,
    required_action_id  TEXT REFERENCES sys_action_catalogs(id) ON DELETE SET NULL,
    created_at          TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at          TEXT NOT NULL DEFAULT (datetime('now'))
  )`,
  `CREATE INDEX idx_sys_ui_routes_menu ON sys_ui_routes(menu_id)`,
  `CREATE INDEX idx_sys_ui_routes_action ON sys_ui_routes(required_action_id)`,

  `CREATE TABLE sys_ui_action_bindings (
    id                  TEXT PRIMARY KEY,
    button_code         TEXT NOT NULL,
    button_label        TEXT NOT NULL,
    page_route          TEXT NOT NULL,
    required_action_id  TEXT REFERENCES sys_action_catalogs(id) ON DELETE CASCADE,
    created_at          TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at          TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(button_code, page_route)
  )`,
  `CREATE INDEX idx_sys_ui_ab_page ON sys_ui_action_bindings(page_route)`,
  `CREATE INDEX idx_sys_ui_ab_action ON sys_ui_action_bindings(required_action_id)`,

  // ── 第12组: 审计链锚定 + 系统配置 (2表) ──
  `CREATE TABLE audit_chain_anchors (
    id              TEXT PRIMARY KEY,
    anchor_hash     TEXT NOT NULL UNIQUE,
    start_event_id  TEXT NOT NULL REFERENCES audit_events(id),
    end_event_id    TEXT NOT NULL REFERENCES audit_events(id),
    event_count     INTEGER NOT NULL DEFAULT 0,
    created_at      TEXT NOT NULL DEFAULT (datetime('now'))
  )`,
  `CREATE INDEX idx_audit_chain_anchors_hash ON audit_chain_anchors(anchor_hash)`,
  `CREATE INDEX idx_audit_chain_anchors_start ON audit_chain_anchors(start_event_id)`,
  `CREATE INDEX idx_audit_chain_anchors_end ON audit_chain_anchors(end_event_id)`,
  `CREATE INDEX idx_audit_chain_anchors_created ON audit_chain_anchors(created_at)`,

  `CREATE TABLE sys_config (
    id              TEXT PRIMARY KEY,
    config_key      TEXT NOT NULL UNIQUE,
    config_value    TEXT NOT NULL,
    value_type      TEXT NOT NULL DEFAULT 'string',
    category        TEXT DEFAULT 'general',
    description     TEXT,
    is_public       INTEGER NOT NULL DEFAULT 0,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
  )`,
  `CREATE INDEX idx_sys_config_key ON sys_config(config_key)`,
  `CREATE INDEX idx_sys_config_category ON sys_config(category)`,
];

let tableCount = 0;
for (const stmt of DDL_STATEMENTS) {
  try {
    db.exec(stmt);
    if (stmt.trim().toUpperCase().startsWith('CREATE TABLE')) {
      tableCount++;
      const name = stmt.match(/CREATE TABLE (\w+)/i)?.[1] || '?';
      results.tables.push({ name, status: 'OK' });
    }
  } catch (e) {
    results.tables.push({ name: stmt.slice(0, 60), status: 'FAIL', error: e.message });
  }
}

log('INFO', `DDL 执行完成: ${tableCount} 张表创建成功`);

// Verify table count
const tableList = db.prepare("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name").all();
log('INFO', `sqlite_master 中实际表数: ${tableList.length}`);
tableList.forEach(t => log('DEBUG', `  ${t.name}`));

// ============================================================================
// STEP 2: 插入种子数据
// ============================================================================
log('INFO', '=== STEP 2: 插入种子数据 ===');

// ── 2.1 users (4人) ──
const userIds = {
  admin: uid(), finance: uid(), deptLeader: uid(), employee: uid()
};
const insertUser = db.prepare(`INSERT INTO users (id, username, email, display_name, password_hash, role, status, prefer_language) VALUES (?,?,?,?,?,?,?,?)`);
for (const [key, id] of Object.entries(userIds)) {
  const profiles = {
    admin:      ['admin',      'admin@agov.local',    '超级管理员',   '$argon2id$hash_admin',      'admin',  'zh-CN'],
    finance:    ['finance_mgr','finance@agov.local',   '财务经理',     '$argon2id$hash_finance',    'member', 'zh-CN'],
    deptLeader: ['dept_lead',  'lead-rd@agov.local',   '研发部Leader', '$argon2id$hash_dept_lead',  'member', 'zh-CN'],
    employee:   ['employee1',  'emp1@agov.local',      '普通员工',     '$argon2id$hash_employee',   'member', 'zh-CN'],
  };
  insertUser.run(id, ...profiles[key]);
}
const userCount = db.prepare('SELECT COUNT(*) AS cnt FROM users').get();
log('INFO', `users 插入: ${userCount.cnt} 行`);
results.seeds.users = userCount.cnt;

// ── 2.2 parties (3个) ──
const partyIds = {
  company: uid(), rdDept: uid(), aiProject: uid()
};
const insertParty = db.prepare(`INSERT INTO parties (id, type, name, description, parent_party_id, leader_user_id, cost_center, status) VALUES (?,?,?,?,?,?,?,?)`);
insertParty.run(partyIds.company,  'org',     'AGov总公司', '顶层组织',       null,               userIds.admin,      'CC-ROOT-001', 'active');
insertParty.run(partyIds.rdDept,   'org',     '研发部',     '研发部门',       partyIds.company,   userIds.deptLeader,  'CC-RD-001',   'active');
insertParty.run(partyIds.aiProject,'project', 'AI智能问答项目', 'AI项目',    partyIds.rdDept,    userIds.deptLeader,  'CC-AI-001',   'active');
const partyCount = db.prepare('SELECT COUNT(*) AS cnt FROM parties').get();
log('INFO', `parties 插入: ${partyCount.cnt} 行`);
results.seeds.parties = partyCount.cnt;

// ── 2.3 party_edges (parent边 + sponsors边) ──
const insertEdge = db.prepare(`INSERT INTO party_edges (id, src_party_id, dst_party_id, edge_type, allows_fund) VALUES (?,?,?,?,?)`);
// parent 边: 研发部 → 总公司 (child→parent, 可划拨)
insertEdge.run(uid(), partyIds.rdDept, partyIds.company, 'parent', 1);
// parent 边: AI项目 → 研发部
insertEdge.run(uid(), partyIds.aiProject, partyIds.rdDept, 'parent', 1);
// sponsors 边: 总公司 → 研发部 (可划拨)
insertEdge.run(uid(), partyIds.company, partyIds.rdDept, 'sponsors', 1);
// sponsors 边: 总公司 → AI项目
insertEdge.run(uid(), partyIds.company, partyIds.aiProject, 'sponsors', 1);
const edgeCount = db.prepare('SELECT COUNT(*) AS cnt FROM party_edges').get();
log('INFO', `party_edges 插入: ${edgeCount.cnt} 行`);
results.seeds.party_edges = edgeCount.cnt;

// ── 2.4 accounts (每个party一个账户，含预算帽) ──
const accountIds = {
  company: uid(), rdDept: uid(), aiProject: uid()
};
const insertAccount = db.prepare(`INSERT INTO accounts (id, party_id, available_balance, frozen_balance, status, budget_limit_amount, budget_warn_ratio, budget_period, budget_consumed_amount, version) VALUES (?,?,?,?,?,?,?,?,?,?)`);
// 总公司: 大额余额 + 月预算帽 $50000
insertAccount.run(accountIds.company,  partyIds.company,  100000.0, 0.0, 'active', 50000.0, 0.80, 'calendar_month', 0.0, 0);
// 研发部: 初始 $30000 + 月预算帽 $15000
insertAccount.run(accountIds.rdDept,   partyIds.rdDept,   30000.0,  0.0, 'active', 15000.0, 0.75, 'calendar_month', 0.0, 0);
// AI项目: 初始 $8000 + 月预算帽 $5000
insertAccount.run(accountIds.aiProject, partyIds.aiProject, 8000.0,  0.0, 'active', 5000.0,  0.90, 'calendar_month', 0.0, 0);
const acctCount = db.prepare('SELECT COUNT(*) AS cnt FROM accounts').get();
log('INFO', `accounts 插入: ${acctCount.cnt} 行`);
results.seeds.accounts = acctCount.cnt;

// ── 2.5 api_keys (admin/admin_key + user/user_key) ──
const apiKeyIds = { adminKey: uid(), userKey: uid() };
const insertApiKey = db.prepare(`INSERT INTO api_keys (id, name, key_hash, key_prefix, key_suffix, owner_user_id, account_id, party_id, status, ax_key_type) VALUES (?,?,?,?,?,?,?,?,?,?)`);
insertApiKey.run(apiKeyIds.adminKey, 'admin_key', 'hash_admin_key_abc123', 'sk-admin', 'xxxx', userIds.admin, accountIds.company, partyIds.company, 'active', 'service_account');
insertApiKey.run(apiKeyIds.userKey,  'user_key',  'hash_user_key_def456',  'sk-user',  'yyyy', userIds.employee, accountIds.aiProject, partyIds.aiProject, 'active', 'user');
const keyCount = db.prepare('SELECT COUNT(*) AS cnt FROM api_keys').get();
log('INFO', `api_keys 插入: ${keyCount.cnt} 行`);
results.seeds.api_keys = keyCount.cnt;

// ── 2.6 sys_action_catalogs (fund/iam/routing/data 四轴动作) ──
const actionIds = {};
const actions = [
  ['fund',    'balance.read',       '查看余额',     'account'],
  ['fund',    'balance.allocate',   '资金划拨',     'account'],
  ['fund',    'ledger.read',        '查看流水',     'account'],
  ['fund',    'freeze.create',      '创建冻结',     'account'],
  ['fund',    'freeze.settle',      '结算冻结',     'account'],
  ['fund',    'budget.set',         '设置预算帽',   'account'],
  ['iam',     'user.create',        '创建用户',     'user'],
  ['iam',     'user.disable',       '禁用用户',     'user'],
  ['iam',     'role.assign',        '分配角色',     'role'],
  ['iam',     'key.create',         '创建API Key',  'key'],
  ['iam',     'key.revoke',         '吊销API Key',  'key'],
  ['routing', 'model.invoke',       '调用模型',     'model'],
  ['routing', 'model.list',         '查看模型列表', 'model'],
  ['routing', 'price.write',        '修改价目表',   'model'],
  ['routing', 'route.create',       '创建路由',     'route'],
  ['data',    'usage.read',         '查看用量',     'usage'],
  ['data',    'audit.read',         '查看审计',     'audit'],
  ['data',    'report.export',      '导出报表',     'report'],
];
const insertAction = db.prepare(`INSERT INTO sys_action_catalogs (id, action_code, action_name, axis, resource_type) VALUES (?,?,?,?,?)`);
for (const [axis, code, name, resType] of actions) {
  const id = uid();
  actionIds[code] = id;
  insertAction.run(id, code, name, axis, resType);
}
const actionCount = db.prepare('SELECT COUNT(*) AS cnt FROM sys_action_catalogs').get();
log('INFO', `sys_action_catalogs 插入: ${actionCount.cnt} 行`);
results.seeds.sys_action_catalogs = actionCount.cnt;

// ── 2.7 sys_roles (4角色) ──
const roleIds = {};
const roles = [
  ['super_admin',   '超级管理员', '全部权限', 1],
  ['fund_manager',  '资金管理员', '资金治理权限', 0],
  ['dept_leader',   '部门负责人', '部门级管理权限', 0],
  ['employee',      '普通员工',   '基础使用权限', 1],
];
const insertRole = db.prepare(`INSERT INTO sys_roles (id, role_code, role_name, description, is_system) VALUES (?,?,?,?,?)`);
for (const [code, name, desc, isSys] of roles) {
  const id = uid();
  roleIds[code] = id;
  insertRole.run(id, code, name, desc, isSys);
}
const roleCount = db.prepare('SELECT COUNT(*) AS cnt FROM sys_roles').get();
log('INFO', `sys_roles 插入: ${roleCount.cnt} 行`);
results.seeds.sys_roles = roleCount.cnt;

// ── 2.8 sys_role_permissions (角色→动作绑定) ──
const rolePerms = [
  // super_admin: 全部18个动作
  ...Object.values(actionIds).map(aid => [roleIds.super_admin, aid]),
  // fund_manager: fund 轴全部6个动作
  ['fund_manager', 'balance.read'],
  ['fund_manager', 'balance.allocate'],
  ['fund_manager', 'ledger.read'],
  ['fund_manager', 'freeze.create'],
  ['fund_manager', 'freeze.settle'],
  ['fund_manager', 'budget.set'],
  // dept_leader: 部分fund + routing + data
  ['dept_leader', 'balance.read'],
  ['dept_leader', 'ledger.read'],
  ['dept_leader', 'model.invoke'],
  ['dept_leader', 'model.list'],
  ['dept_leader', 'key.create'],
  ['dept_leader', 'usage.read'],
  ['dept_leader', 'report.export'],
  // employee: 基础权限
  ['employee', 'model.invoke'],
  ['employee', 'model.list'],
  ['employee', 'usage.read'],
  ['employee', 'key.create'],
];
const insertRolePerm = db.prepare(`INSERT INTO sys_role_permissions (id, role_id, action_id) VALUES (?,?,?)`);
for (const [roleCode, actionCode] of rolePerms) {
  insertRolePerm.run(uid(), roleIds[roleCode], actionIds[actionCode]);
}
const rpCount = db.prepare('SELECT COUNT(*) AS cnt FROM sys_role_permissions').get();
log('INFO', `sys_role_permissions 插入: ${rpCount.cnt} 行`);
results.seeds.sys_role_permissions = rpCount.cnt;

// ── 2.9 sys_access_policies (4条内置职责分离策略) ──
const policyIds = {};
const policies = [
  ['P-DENY-EXTERNAL-MODEL',    '禁止外部网络访问受限模型', 'deny',  JSON.stringify({axis:"data",actions:["model.invoke"],resource_type:"model",conditions:{data_classification:["confidential","restricted"],network_class:"external"}}), 100, 1, 'Deny access to confidential/restricted models from external network'],
  ['P-DENY-SELF-APPROVE',      '禁止自审批资金划拨',       'deny',  JSON.stringify({axis:"fund",actions:["balance.allocate"],conditions:{self_approve:true}}), 90, 1, 'Prevent same person from initiating and approving fund transfers'],
  ['P-ALLOW-WORK-HOURS',       '仅工作时间允许调用模型',   'allow', JSON.stringify({axis:"routing",actions:["model.invoke"],conditions:{time_restriction:{start:"09:00",end:"18:00",timezone:"Asia/Shanghai"}}}), 50, 1, 'Allow model invocation only during business hours'],
  ['P-ALLOW-INTERNAL-DATA',    '允许内部数据分类访问',     'allow', JSON.stringify({axis:"data",actions:["usage.read","report.export"],resource_type:"model",conditions:{data_classification:["public","internal"]}}), 50, 1, 'Allow reading usage data for public/internal classification models'],
];
const insertPolicy = db.prepare(`INSERT INTO sys_access_policies (id, policy_code, policy_name, effect, conditions_json, priority, is_system, description) VALUES (?,?,?,?,?,?,?,?)`);
for (const [code, name, effect, cond, pri, isSys, desc] of policies) {
  const id = uid();
  policyIds[code] = id;
  insertPolicy.run(id, code, name, effect, cond, pri, isSys, desc);
}
const polCount = db.prepare('SELECT COUNT(*) AS cnt FROM sys_access_policies').get();
log('INFO', `sys_access_policies 插入: ${polCount.cnt} 行`);
results.seeds.sys_access_policies = polCount.cnt;

// ── 2.10 model_grants (研发部 ALLOW GPT-4 + DENY GPT-4.5) ──
// First insert models
const modelIdGpt4 = uid();
const modelIdGpt45 = uid();
const insertModel = db.prepare(`INSERT INTO models (id, name, category, developer, context_window, input_price_per_1m, output_price_per_1m, sell_input_price_per_1m, sell_output_price_per_1m, status, data_classification, network_class) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`);
insertModel.run(modelIdGpt4,  'gpt-4',   'chat', 'OpenAI', 8192,  30.0, 60.0,  45.0, 90.0,  'active', 'internal',     'external');
insertModel.run(modelIdGpt45, 'gpt-4.5', 'chat', 'OpenAI', 16384, 75.0, 150.0, 112.5, 225.0, 'active', 'confidential', 'external');
log('INFO', `models 插入: gpt-4, gpt-4.5`);

const insertModelGrant = db.prepare(`INSERT INTO model_grants (id, principal_type, principal_id, model_id, effect, priority, quota_limit) VALUES (?,?,?,?,?,?,?)`);
// 研发部 ALLOW GPT-4
insertModelGrant.run(uid(), 'party', partyIds.rdDept, modelIdGpt4, 'allow', 10, 1000000);
// 研发部 DENY GPT-4.5
insertModelGrant.run(uid(), 'party', partyIds.rdDept, modelIdGpt45, 'deny', 20, null);
const mgCount = db.prepare('SELECT COUNT(*) AS cnt FROM model_grants').get();
log('INFO', `model_grants 插入: ${mgCount.cnt} 行`);
results.seeds.model_grants = mgCount.cnt;

// ── 2.11 model_prices (GPT-4 双轨价目) ──
const mpId = uid();
const insertModelPrice = db.prepare(`INSERT INTO model_prices (id, model_id, channel_id, reference_id, price_json, status) VALUES (?,?,?,?,?,?)`);
const priceJsonGpt4 = JSON.stringify({
  items: [
    {itemCode:"prompt_tokens",           cost:{mode:"usage_per_unit",rate:0.030}, sell:{mode:"usage_per_unit",rate:0.045}},
    {itemCode:"completion_tokens",       cost:{mode:"usage_per_unit",rate:0.060}, sell:{mode:"usage_per_unit",rate:0.090}},
    {itemCode:"prompt_cached_tokens",    cost:{mode:"usage_per_unit",rate:0.015}, sell:{mode:"usage_per_unit",rate:0.0225}},
    {itemCode:"prompt_write_cached_tokens",cost:{mode:"usage_per_unit",rate:0.030},sell:{mode:"usage_per_unit",rate:0.045}},
    {itemCode:"completion_reasoning_tokens",cost:{mode:"usage_per_unit",rate:0.120},sell:{mode:"usage_per_unit",rate:0.160}}
  ],
  schedule: {timezone:"Asia/Shanghai", overrides:[]}
});
insertModelPrice.run(mpId, modelIdGpt4, null, 'REF-GPT4-STANDARD-v1', priceJsonGpt4, 'active');
const mpCount = db.prepare('SELECT COUNT(*) AS cnt FROM model_prices').get();
log('INFO', `model_prices 插入: ${mpCount.cnt} 行`);
results.seeds.model_prices = mpCount.cnt;

// Also insert a provider for completeness
const providerId = uid();
db.prepare(`INSERT INTO providers (id, name, type, status) VALUES (?,?,?,?)`).run(providerId, 'OpenAI', 'openai', 'active');

// ── Seed data summary ──
log('INFO', '=== 种子数据插入完成 ===');
const seedSummary = {
  users: userCount.cnt,
  parties: partyCount.cnt,
  party_edges: edgeCount.cnt,
  accounts: acctCount.cnt,
  api_keys: keyCount.cnt,
  sys_action_catalogs: actionCount.cnt,
  sys_roles: roleCount.cnt,
  sys_role_permissions: rpCount.cnt,
  sys_access_policies: polCount.cnt,
  model_grants: mgCount.cnt,
  model_prices: mpCount.cnt,
};
console.log(JSON.stringify(seedSummary, null, 2));

// ============================================================================
// STEP 3: 约束校验
// ============================================================================
log('INFO', '=== STEP 3: 约束校验 ===');

// ── 3.1 UNIQUE 约束校验 ──
const uniqueTests = [
  { desc: 'users.username UNIQUE', test: () => {
    try { db.prepare('INSERT INTO users (id, username) VALUES (?,?)').run(uid(), 'admin'); return 'FAIL (should reject)'; }
    catch(e) { return e.message.includes('UNIQUE') ? 'PASS' : 'FAIL: ' + e.message; }
  }},
  { desc: 'party_edges UNIQUE(src,dst,type)', test: () => {
    try { db.prepare('INSERT INTO party_edges (id, src_party_id, dst_party_id, edge_type) VALUES (?,?,?,?)').run(uid(), partyIds.rdDept, partyIds.company, 'parent'); return 'FAIL'; }
    catch(e) { return e.message.includes('UNIQUE') ? 'PASS' : 'FAIL: ' + e.message; }
  }},
  { desc: 'party_members UNIQUE(party,user)', test: () => {
    try { db.prepare('INSERT INTO party_members (id, party_id, user_id) VALUES (?,?,?)').run(uid(), partyIds.company, userIds.admin); return 'FAIL'; }
    catch(e) { return e.message.includes('UNIQUE') ? 'PASS' : 'FAIL: ' + e.message; }
  }},
  { desc: 'accounts.party_id UNIQUE', test: () => {
    try { db.prepare('INSERT INTO accounts (id, party_id) VALUES (?,?)').run(uid(), partyIds.company); return 'FAIL'; }
    catch(e) { return e.message.includes('UNIQUE') ? 'PASS' : 'FAIL: ' + e.message; }
  }},
  { desc: 'api_keys.key_hash UNIQUE', test: () => {
    try { db.prepare('INSERT INTO api_keys (id, name, key_hash, owner_user_id, account_id) VALUES (?,?,?,?,?)').run(uid(), 'dup', 'hash_admin_key_abc123', userIds.admin, accountIds.company); return 'FAIL'; }
    catch(e) { return e.message.includes('UNIQUE') ? 'PASS' : 'FAIL: ' + e.message; }
  }},
  { desc: 'sys_action_catalogs.action_code UNIQUE', test: () => {
    try { db.prepare('INSERT INTO sys_action_catalogs (id, action_code, action_name, axis) VALUES (?,?,?,?)').run(uid(), 'balance.read', 'dup', 'fund'); return 'FAIL'; }
    catch(e) { return e.message.includes('UNIQUE') ? 'PASS' : 'FAIL: ' + e.message; }
  }},
  { desc: 'sys_roles.role_code UNIQUE', test: () => {
    try { db.prepare('INSERT INTO sys_roles (id, role_code, role_name) VALUES (?,?,?)').run(uid(), 'super_admin', 'dup'); return 'FAIL'; }
    catch(e) { return e.message.includes('UNIQUE') ? 'PASS' : 'FAIL: ' + e.message; }
  }},
  { desc: 'sys_role_permissions UNIQUE(role,action)', test: () => {
    try { db.prepare('INSERT INTO sys_role_permissions (id, role_id, action_id) VALUES (?,?,?)').run(uid(), roleIds.super_admin, actionIds['balance.read']); return 'FAIL'; }
    catch(e) { return e.message.includes('UNIQUE') ? 'PASS' : 'FAIL: ' + e.message; }
  }},
  { desc: 'sys_access_policies.policy_code UNIQUE', test: () => {
    try { db.prepare('INSERT INTO sys_access_policies (id, policy_code, policy_name, effect, conditions_json) VALUES (?,?,?,?,?)').run(uid(), 'P-DENY-EXTERNAL-MODEL', 'dup', 'allow', '{}'); return 'FAIL'; }
    catch(e) { return e.message.includes('UNIQUE') ? 'PASS' : 'FAIL: ' + e.message; }
  }},
  { desc: 'model_prices.reference_id UNIQUE', test: () => {
    try { db.prepare('INSERT INTO model_prices (id, model_id, reference_id, price_json) VALUES (?,?,?,?)').run(uid(), modelIdGpt4, 'REF-GPT4-STANDARD-v1', '{}'); return 'FAIL'; }
    catch(e) { return e.message.includes('UNIQUE') ? 'PASS' : 'FAIL: ' + e.message; }
  }},
  { desc: 'idempotency_records UNIQUE(scope,actor,key)', test: () => {
    db.prepare('INSERT INTO idempotency_records (id, scope, actor_id, idempotency_key, request_hash, expires_at) VALUES (?,?,?,?,?,?)').run(uid(), 'allocate', 'actor1', 'idem-key-001', 'abc', laterISO(24));
    try { db.prepare('INSERT INTO idempotency_records (id, scope, actor_id, idempotency_key, request_hash, expires_at) VALUES (?,?,?,?,?,?)').run(uid(), 'allocate', 'actor1', 'idem-key-001', 'def', laterISO(24)); return 'FAIL'; }
    catch(e) { return e.message.includes('UNIQUE') ? 'PASS' : 'FAIL: ' + e.message; }
  }},
];

for (const t of uniqueTests) {
  const result = t.test();
  const passed = result === 'PASS';
  results.constraints[passed ? 'passed' : 'failed'].push({ desc: t.desc, result });
  log(passed ? 'PASS' : 'FAIL', `${t.desc}: ${result}`);
}

// ── 3.2 外键约束校验 ──
const fkTests = [
  { desc: 'admin_sessions → users FK (无效 user_id 应被拒绝)', test: () => {
    try { db.prepare('INSERT INTO admin_sessions (token, user_id, expires_at) VALUES (?,?,?)').run(uid(), 'non-existent-user', laterISO(1)); return 'FAIL'; }
    catch(e) { return e.message.includes('FOREIGN KEY') ? 'PASS' : 'FAIL: ' + e.message; }
  }},
  { desc: 'api_keys → accounts FK (无效 account_id 应被拒绝)', test: () => {
    try { db.prepare('INSERT INTO api_keys (id, name, key_hash, owner_user_id, account_id) VALUES (?,?,?,?,?)').run(uid(), 'fk-test', 'hash_fk', userIds.admin, 'non-existent-acct'); return 'FAIL'; }
    catch(e) { return e.message.includes('FOREIGN KEY') ? 'PASS' : 'FAIL: ' + e.message; }
  }},
  { desc: 'sys_role_permissions → sys_roles FK CASCADE', test: () => {
    // This tests the FK directly: referencing a non-existent role should fail
    try { db.prepare('INSERT INTO sys_role_permissions (id, role_id, action_id) VALUES (?,?,?)').run(uid(), 'non-existent-role', actionIds['balance.read']); return 'FAIL'; }
    catch(e) { return e.message.includes('FOREIGN KEY') ? 'PASS' : 'FAIL: ' + e.message; }
  }},
  { desc: 'sys_subject_role_bindings → sys_roles FK CASCADE', test: () => {
    try { db.prepare('INSERT INTO sys_subject_role_bindings (id, subject_type, subject_id, role_id) VALUES (?,?,?,?)').run(uid(), 'user', userIds.admin, 'non-existent-role'); return 'FAIL'; }
    catch(e) { return e.message.includes('FOREIGN KEY') ? 'PASS' : 'FAIL: ' + e.message; }
  }},
  { desc: 'sys_access_policy_bindings → sys_access_policies FK CASCADE', test: () => {
    try { db.prepare('INSERT INTO sys_access_policy_bindings (id, policy_id, subject_type, subject_id) VALUES (?,?,?,?)').run(uid(), 'non-existent-policy', 'user', userIds.admin); return 'FAIL'; }
    catch(e) { return e.message.includes('FOREIGN KEY') ? 'PASS' : 'FAIL: ' + e.message; }
  }},
  { desc: 'sys_ui_menus self-ref FK', test: () => {
    try { db.prepare('INSERT INTO sys_ui_menus (id, menu_code, label, parent_id) VALUES (?,?,?,?)').run(uid(), 'child-menu', 'Child', 'non-existent-menu'); return 'FAIL'; }
    catch(e) { return e.message.includes('FOREIGN KEY') ? 'PASS' : 'FAIL: ' + e.message; }
  }},
  { desc: 'sys_ui_routes → sys_ui_menus FK (SET NULL)', test: () => {
    try { db.prepare('INSERT INTO sys_ui_routes (id, route_path, menu_id) VALUES (?,?,?)').run(uid(), '/test-route-fk', 'non-existent-menu'); return 'FAIL'; }
    catch(e) { return e.message.includes('FOREIGN KEY') ? 'PASS' : 'FAIL: ' + e.message; }
  }},
  { desc: 'audit_chain_anchors → audit_events FK', test: () => {
    try { db.prepare('INSERT INTO audit_chain_anchors (id, anchor_hash, start_event_id, end_event_id) VALUES (?,?,?,?)').run(uid(), 'hash_fk_test', 'non-existent-event', 'non-existent-event-2'); return 'FAIL'; }
    catch(e) { return e.message.includes('FOREIGN KEY') ? 'PASS' : 'FAIL: ' + e.message; }
  }},
];

for (const t of fkTests) {
  const result = t.test();
  const passed = result === 'PASS';
  results.constraints[passed ? 'passed' : 'failed'].push({ desc: t.desc, result });
  log(passed ? 'PASS' : 'FAIL', `${t.desc}: ${result}`);
}

// ── 3.3 CASCADE DELETE 测试 ──
log('INFO', '--- CASCADE DELETE 测试 ---');
// Create temp role for cascade test
const tempRoleId = uid();
const tempActionId = uid();
db.prepare('INSERT INTO sys_action_catalogs (id, action_code, action_name, axis) VALUES (?,?,?,?)').run(tempActionId, 'temp.action.cascade', 'Temp Action', 'fund');
db.prepare('INSERT INTO sys_roles (id, role_code, role_name, is_system) VALUES (?,?,?,?)').run(tempRoleId, 'temp_role_cascade', 'Temp Role', 0);
const tempPermId = uid();
db.prepare('INSERT INTO sys_role_permissions (id, role_id, action_id) VALUES (?,?,?)').run(tempPermId, tempRoleId, tempActionId);

// Delete role → permissions should cascade
db.prepare('DELETE FROM sys_roles WHERE id = ?').run(tempRoleId);
const cascadeCheck = db.prepare('SELECT COUNT(*) AS cnt FROM sys_role_permissions WHERE id = ?').get(tempPermId);
const cascadePassed = cascadeCheck.cnt === 0;
results.constraints[cascadePassed ? 'passed' : 'failed'].push({ desc: 'CASCADE DELETE: sys_roles → sys_role_permissions', result: cascadePassed ? 'PASS' : `FAIL: ${cascadeCheck.cnt} rows remain` });
log(cascadePassed ? 'PASS' : 'FAIL', `CASCADE DELETE: sys_roles → sys_role_permissions: ${cascadePassed ? 'PASS' : 'FAIL'}`);

// ── 3.4 索引校验 ──
log('INFO', '--- 索引校验 ---');
// Check all expected indexes exist
const expectedIndexes = [
  'idx_users_status', 'idx_users_email',
  'idx_admin_sessions_user_id',
  'idx_parties_type', 'idx_parties_parent', 'idx_parties_status',
  'idx_party_edges_src', 'idx_party_edges_dst',
  'idx_party_members_party', 'idx_party_members_user',
  'idx_accounts_party', 'idx_accounts_status',
  'idx_ledgers_account', 'idx_ledgers_freeze', 'idx_ledgers_request', 'idx_ledgers_idem',
  'idx_freezes_account', 'idx_freezes_request', 'idx_freezes_expiry', 'idx_freezes_user',
  'idx_allocations_src', 'idx_allocations_dst', 'idx_allocations_idem',
  'idx_liquidations_party', 'idx_liquidations_account', 'idx_liquidations_status',
  'idx_api_keys_owner', 'idx_api_keys_account', 'idx_api_keys_party', 'idx_api_keys_project', 'idx_api_keys_status', 'idx_api_keys_hash',
  'idx_providers_type', 'idx_providers_status',
  'idx_prov_res_provider', 'idx_prov_res_group', 'idx_prov_res_status',
  'idx_models_category', 'idx_models_status', 'idx_models_developer',
  'idx_prov_models_provider', 'idx_prov_models_canonical',
  'idx_model_prices_model', 'idx_model_prices_ref',
  'idx_model_routes_model', 'idx_model_routes_provider', 'idx_model_routes_resource', 'idx_model_routes_profile',
  'idx_grants_principal', 'idx_grants_axis', 'idx_grants_resource',
  'idx_model_grants_principal', 'idx_model_grants_model', 'idx_model_grants_effect',
  'idx_request_logs_req', 'idx_request_logs_key', 'idx_request_logs_project', 'idx_request_logs_account', 'idx_request_logs_user', 'idx_request_logs_model', 'idx_request_logs_error',
  'idx_route_attempt_req', 'idx_route_attempt_route',
  'idx_usage_records_req', 'idx_usage_records_key', 'idx_usage_records_project', 'idx_usage_records_model', 'idx_usage_records_user', 'idx_usage_records_account',
  'idx_channel_probes_ch',
  'idx_prov_quota_status', 'idx_prov_quota_check',
  'idx_audit_events_actor', 'idx_audit_events_resource', 'idx_audit_events_action',
  'idx_idempotency_expiry',
  'idx_sys_action_catalogs_axis', 'idx_sys_action_catalogs_resource',
  'idx_sys_roles_code',
  'idx_sys_role_perms_role', 'idx_sys_role_perms_action',
  'idx_sys_srb_subject', 'idx_sys_srb_role', 'idx_sys_srb_scope', 'idx_sys_srb_validity',
  'idx_sys_access_policies_code', 'idx_sys_access_policies_effect', 'idx_sys_access_policies_priority',
  'idx_sys_apb_policy', 'idx_sys_apb_subject',
  'idx_sys_ui_menus_parent', 'idx_sys_ui_menus_sort',
  'idx_sys_ui_routes_menu', 'idx_sys_ui_routes_action',
  'idx_sys_ui_ab_page', 'idx_sys_ui_ab_action',
  'idx_audit_chain_anchors_hash', 'idx_audit_chain_anchors_start', 'idx_audit_chain_anchors_end', 'idx_audit_chain_anchors_created',
  'idx_sys_config_key', 'idx_sys_config_category',
];

const actualIndexes = new Set(
  db.prepare("SELECT name FROM sqlite_master WHERE type='index' AND name NOT LIKE 'sqlite_autoindex_%'").all().map(r => r.name)
);

let indexPass = 0, indexFail = 0;
for (const idx of expectedIndexes) {
  if (actualIndexes.has(idx)) {
    indexPass++;
  } else {
    indexFail++;
    results.indexes.failed.push({ name: idx, result: 'MISSING' });
    log('FAIL', `Index missing: ${idx}`);
  }
}
results.indexes.passed.push({ count: indexPass });
log('INFO', `索引检查: ${indexPass} 通过, ${indexFail} 缺失`);

// ============================================================================
// STEP 4: 账户余额与预算帽校验
// ============================================================================
log('INFO', '=== STEP 4: 账户余额与预算帽校验 ===');

const balanceTests = [
  { desc: '总公司账户余额 = 100000', test: () => {
    const r = db.prepare('SELECT available_balance FROM accounts WHERE party_id = ?').get(partyIds.company);
    return r.available_balance === 100000 ? 'PASS' : `FAIL: expected 100000, got ${r.available_balance}`;
  }},
  { desc: '研发部账户余额 = 30000', test: () => {
    const r = db.prepare('SELECT available_balance FROM accounts WHERE party_id = ?').get(partyIds.rdDept);
    return r.available_balance === 30000 ? 'PASS' : `FAIL: expected 30000, got ${r.available_balance}`;
  }},
  { desc: 'AI项目账户余额 = 8000', test: () => {
    const r = db.prepare('SELECT available_balance FROM accounts WHERE party_id = ?').get(partyIds.aiProject);
    return r.available_balance === 8000 ? 'PASS' : `FAIL: expected 8000, got ${r.available_balance}`;
  }},
  { desc: '总公司预算帽 = 50000', test: () => {
    const r = db.prepare('SELECT budget_limit_amount FROM accounts WHERE party_id = ?').get(partyIds.company);
    return r.budget_limit_amount === 50000 ? 'PASS' : `FAIL: expected 50000, got ${r.budget_limit_amount}`;
  }},
  { desc: '研发部预算帽 = 15000', test: () => {
    const r = db.prepare('SELECT budget_limit_amount FROM accounts WHERE party_id = ?').get(partyIds.rdDept);
    return r.budget_limit_amount === 15000 ? 'PASS' : `FAIL: expected 15000, got ${r.budget_limit_amount}`;
  }},
  { desc: 'AI项目预算帽 = 5000', test: () => {
    const r = db.prepare('SELECT budget_limit_amount FROM accounts WHERE party_id = ?').get(partyIds.aiProject);
    return r.budget_limit_amount === 5000 ? 'PASS' : `FAIL: expected 5000, got ${r.budget_limit_amount}`;
  }},
  { desc: '总公司预算告警比例 = 0.80', test: () => {
    const r = db.prepare('SELECT budget_warn_ratio FROM accounts WHERE party_id = ?').get(partyIds.company);
    return Math.abs(r.budget_warn_ratio - 0.80) < 0.0001 ? 'PASS' : `FAIL: expected 0.80, got ${r.budget_warn_ratio}`;
  }},
  { desc: '研发部预算告警比例 = 0.75', test: () => {
    const r = db.prepare('SELECT budget_warn_ratio FROM accounts WHERE party_id = ?').get(partyIds.rdDept);
    return Math.abs(r.budget_warn_ratio - 0.75) < 0.0001 ? 'PASS' : `FAIL: expected 0.75, got ${r.budget_warn_ratio}`;
  }},
  { desc: 'AI项目预算告警比例 = 0.90', test: () => {
    const r = db.prepare('SELECT budget_warn_ratio FROM accounts WHERE party_id = ?').get(partyIds.aiProject);
    return Math.abs(r.budget_warn_ratio - 0.90) < 0.0001 ? 'PASS' : `FAIL: expected 0.90, got ${r.budget_warn_ratio}`;
  }},
  { desc: '所有账户 frozen_balance = 0', test: () => {
    const r = db.prepare('SELECT COUNT(*) AS cnt FROM accounts WHERE frozen_balance != 0').get();
    return r.cnt === 0 ? 'PASS' : `FAIL: ${r.cnt} accounts have non-zero frozen_balance`;
  }},
  { desc: '所有账户 budget_consumed_amount = 0', test: () => {
    const r = db.prepare('SELECT COUNT(*) AS cnt FROM accounts WHERE budget_consumed_amount != 0').get();
    return r.cnt === 0 ? 'PASS' : `FAIL: ${r.cnt} accounts have non-zero budget_consumed_amount`;
  }},
  { desc: '所有账户 budget_period = calendar_month', test: () => {
    const r = db.prepare("SELECT COUNT(*) AS cnt FROM accounts WHERE budget_period != 'calendar_month'").get();
    return r.cnt === 0 ? 'PASS' : `FAIL: ${r.cnt} accounts don't have calendar_month period`;
  }},
  { desc: '所有账户 status = active', test: () => {
    const r = db.prepare("SELECT COUNT(*) AS cnt FROM accounts WHERE status != 'active'").get();
    return r.cnt === 0 ? 'PASS' : `FAIL: ${r.cnt} accounts are not active`;
  }},
];

for (const t of balanceTests) {
  const result = t.test();
  const passed = result === 'PASS';
  results.balances[passed ? 'passed' : 'failed'].push({ desc: t.desc, result });
  log(passed ? 'PASS' : 'FAIL', `${t.desc}: ${result}`);
}

// ============================================================================
// STEP 5: 补充数据完整性查询
// ============================================================================
log('INFO', '=== STEP 5: 补充验证 ===');

// 账户与 party 1:1 关系
const acctPartyCheck = db.prepare(`
  SELECT p.id AS party_id, p.name, a.id AS account_id, a.available_balance, a.budget_limit_amount
  FROM parties p JOIN accounts a ON a.party_id = p.id
`).all();
log('INFO', '账户-Party 1:1 映射:');
acctPartyCheck.forEach(r => log('DEBUG', `  ${r.name} → balance=${r.available_balance} budget=${r.budget_limit_amount}`));

// api_keys 归属
const keyCheck = db.prepare(`
  SELECT ak.name, ak.ax_key_type, u.username AS owner, p.name AS party, a.available_balance
  FROM api_keys ak
  JOIN users u ON u.id = ak.owner_user_id
  JOIN accounts a ON a.id = ak.account_id
  LEFT JOIN parties p ON p.id = ak.party_id
`).all();
log('INFO', 'API Keys 归属:');
keyCheck.forEach(r => log('DEBUG', `  ${r.name} (${r.ax_key_type}) → owner=${r.owner} party=${r.party} balance=${r.available_balance}`));

// model_grants 详情
const mgCheck = db.prepare(`
  SELECT mg.effect, mg.principal_type, p.name AS principal_name, m.name AS model_name, mg.priority
  FROM model_grants mg
  LEFT JOIN parties p ON mg.principal_type='party' AND p.id=mg.principal_id
  LEFT JOIN models m ON m.id=mg.model_id
`).all();
log('INFO', 'Model Grants:');
mgCheck.forEach(r => log('DEBUG', `  ${r.effect.toUpperCase()} ${r.principal_name} → ${r.model_name} (priority=${r.priority})`));

// 角色-权限-动作 完整链路
const rolePermChain = db.prepare(`
  SELECT sr.role_code, sr.role_name, COUNT(rp.id) AS perm_count
  FROM sys_roles sr
  LEFT JOIN sys_role_permissions rp ON rp.role_id = sr.id
  GROUP BY sr.id
  ORDER BY perm_count DESC
`).all();
log('INFO', '角色权限统计:');
rolePermChain.forEach(r => log('DEBUG', `  ${r.role_code} (${r.role_name}): ${r.perm_count} 权限`));

// ── Generate final report ──
const totalConstraintPassed = results.constraints.passed.length;
const totalConstraintFailed = results.constraints.failed.length;
const totalBalancePassed = results.balances.passed.length;
const totalBalanceFailed = results.balances.failed.length;

log('INFO', '═══════════════════════════════════════');
log('INFO', '最终汇总');
log('INFO', `  建表: ${tableCount}/40 成功`);
log('INFO', `  索引: ${indexPass}/${expectedIndexes.length} 存在`);
log('INFO', `  约束: ${totalConstraintPassed} 通过 / ${totalConstraintFailed} 失败`);
log('INFO', `  余额: ${totalBalancePassed} 通过 / ${totalBalanceFailed} 失败`);
log('INFO', '═══════════════════════════════════════');

// Store for report generation
const finalResults = {
  tableCount,
  tableList: tableList.map(t => t.name),
  seedSummary,
  constraints: results.constraints,
  indexes: { expected: expectedIndexes.length, actual: indexPass, passed: indexPass, failed: indexFail, missing: results.indexes.failed.map(f => f.name) },
  balances: results.balances,
  acctPartyMapping: acctPartyCheck,
  apiKeyMapping: keyCheck,
  modelGrants: mgCheck,
  rolePermStats: rolePermChain,
  totalConstraintPassed,
  totalConstraintFailed,
  totalBalancePassed,
  totalBalanceFailed,
};

// Write results JSON for report generation
const outDir = 'D:/ai-work/grok/a-gov/docs/delivery/acceptance/e2e';
if (!existsSync(outDir)) mkdirSync(outDir, { recursive: true });
writeFileSync(join(outDir, 'E2E-1-results.json'), JSON.stringify(finalResults, null, 2));
log('INFO', `Results written to ${outDir}/E2E-1-results.json`);

db.close();
console.log('DONE');
