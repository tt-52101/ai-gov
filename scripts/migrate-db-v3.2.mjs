#!/usr/bin/env node
// ============================================================================
// migrate-db-v3.2.mjs
// Migrate tokenhub.db from legacy TokenHub schema to ai-gov-fusion-v3.2 (40 tables)
//
// Usage:
//   node scripts/migrate-db-v3.2.mjs
//
// The script is idempotent — safe to re-run on an already-migrated database.
// It backs up the DB on first run and skips columns/tables that already exist.
// ============================================================================

import Database from "better-sqlite3";
import { randomUUID } from "crypto";
import { readFileSync, copyFileSync, existsSync, mkdirSync } from "fs";
import { join, dirname } from "path";
import { fileURLToPath } from "url";
import { createHash } from "crypto";

const __dirname = dirname(fileURLToPath(import.meta.url));
const ROOT = join(__dirname, "..");
const DB_PATH = join(ROOT, "ai-gov-fusion", "backend", "data", "tokenhub.db");
const BACKUP_PATH = DB_PATH + ".backup-2026-07-31";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------
function uuid() {
  return randomUUID();
}

function now() {
  return new Date().toISOString().replace("T", " ").slice(0, 19);
}

function sha256(s) {
  return createHash("sha256").update(s).digest("hex");
}

function tableExists(db, name) {
  const row = db
    .prepare("SELECT name FROM sqlite_master WHERE type='table' AND name=?")
    .get(name);
  return !!row;
}

function columnExists(db, table, column) {
  const cols = db.prepare(`PRAGMA table_info('${table}')`).all();
  return cols.some((c) => c.name === column);
}

function addColumn(db, table, column, type, defaultValue) {
  if (columnExists(db, table, column)) {
    console.log(`  [SKIP] ${table}.${column} already exists`);
    return false;
  }
  // SQLite ALTER TABLE only supports constant defaults (no expressions like (datetime('now')))
  const isExprDefault = defaultValue !== undefined && defaultValue.includes("(");
  const dfltClause = (!isExprDefault && defaultValue !== undefined) ? ` DEFAULT ${defaultValue}` : "";
  db.exec(`ALTER TABLE ${table} ADD COLUMN ${column} ${type}${dfltClause}`);
  if (isExprDefault) {
    // For expression defaults, update existing rows after adding the column
    db.exec(`UPDATE "${table}" SET "${column}" = ${defaultValue} WHERE "${column}" IS NULL`);
  }
  console.log(`  [ADD] ${table}.${column} ${type}${dfltClause || " (expr default applied)"}`);
  return true;
}

function createTableIfNotExists(db, name, ddl) {
  if (tableExists(db, name)) {
    console.log(`  [SKIP] Table ${name} already exists`);
    return false;
  }
  db.exec(ddl);
  console.log(`  [CREATE] Table ${name}`);
  return true;
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------
async function main() {
  // ---------- Step 0: Backup ----------
  if (!existsSync(BACKUP_PATH)) {
    copyFileSync(DB_PATH, BACKUP_PATH);
    console.log(`[BACKUP] Created ${BACKUP_PATH}`);
  } else {
    console.log(`[BACKUP] Already exists: ${BACKUP_PATH}`);
  }

  const db = new Database(DB_PATH);
  db.pragma("journal_mode = WAL");
  db.pragma("foreign_keys = ON");

  try {
    // ===================================================================
    // SECTION 1: New tables (not in old schema)
    // ===================================================================
    console.log("\n=== SECTION 1: Creating new tables ===");

    // 3. parties (maps from old projects, but with extended schema)
    createTableIfNotExists(db, "parties", `
      CREATE TABLE parties (
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
      )
    `);

    // 4. party_edges
    createTableIfNotExists(db, "party_edges", `
      CREATE TABLE party_edges (
        id              TEXT PRIMARY KEY,
        src_party_id    TEXT NOT NULL,
        dst_party_id    TEXT NOT NULL,
        edge_type       TEXT NOT NULL,
        allows_fund     INTEGER NOT NULL DEFAULT 0,
        created_at      TEXT NOT NULL DEFAULT (datetime('now')),
        UNIQUE(src_party_id, dst_party_id, edge_type)
      )
    `);

    // 5. party_members (maps from old project_teams)
    createTableIfNotExists(db, "party_members", `
      CREATE TABLE party_members (
        id          TEXT PRIMARY KEY,
        party_id    TEXT NOT NULL,
        user_id     TEXT NOT NULL,
        role        TEXT NOT NULL DEFAULT 'member',
        is_primary  INTEGER DEFAULT 0,
        joined_at   TEXT NOT NULL DEFAULT (datetime('now')),
        created_at  TEXT NOT NULL DEFAULT (datetime('now')),
        updated_at  TEXT NOT NULL DEFAULT (datetime('now')),
        UNIQUE(party_id, user_id)
      )
    `);

    // 6. accounts
    createTableIfNotExists(db, "accounts", `
      CREATE TABLE accounts (
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
      )
    `);

    // 7. ledgers
    createTableIfNotExists(db, "ledgers", `
      CREATE TABLE ledgers (
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
      )
    `);

    // 8. freezes
    createTableIfNotExists(db, "freezes", `
      CREATE TABLE freezes (
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
      )
    `);

    // 9. allocations
    createTableIfNotExists(db, "allocations", `
      CREATE TABLE allocations (
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
      )
    `);

    // 10. liquidations
    createTableIfNotExists(db, "liquidations", `
      CREATE TABLE liquidations (
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
      )
    `);

    // 16. model_prices
    createTableIfNotExists(db, "model_prices", `
      CREATE TABLE model_prices (
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
      )
    `);

    // 18. route_profiles
    createTableIfNotExists(db, "route_profiles", `
      CREATE TABLE route_profiles (
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
      )
    `);

    // 19. grants
    createTableIfNotExists(db, "grants", `
      CREATE TABLE grants (
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
      )
    `);

    // 20. model_grants
    createTableIfNotExists(db, "model_grants", `
      CREATE TABLE model_grants (
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
      )
    `);

    // 26. channel_probes
    createTableIfNotExists(db, "channel_probes", `
      CREATE TABLE channel_probes (
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
      )
    `);

    // 27. provider_quota_status
    createTableIfNotExists(db, "provider_quota_status", `
      CREATE TABLE provider_quota_status (
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
      )
    `);

    // 29. idempotency_records
    createTableIfNotExists(db, "idempotency_records", `
      CREATE TABLE idempotency_records (
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
      )
    `);

    // 30-38: ABAC + UI governance tables
    createTableIfNotExists(db, "sys_action_catalogs", `
      CREATE TABLE sys_action_catalogs (
        id              TEXT PRIMARY KEY,
        action_code     TEXT NOT NULL UNIQUE,
        action_name     TEXT NOT NULL,
        axis            TEXT NOT NULL,
        resource_type   TEXT,
        description     TEXT,
        created_at      TEXT NOT NULL DEFAULT (datetime('now')),
        updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
      )
    `);

    createTableIfNotExists(db, "sys_roles", `
      CREATE TABLE sys_roles (
        id              TEXT PRIMARY KEY,
        role_code       TEXT NOT NULL UNIQUE,
        role_name       TEXT NOT NULL,
        description     TEXT,
        is_system       INTEGER NOT NULL DEFAULT 0,
        created_at      TEXT NOT NULL DEFAULT (datetime('now')),
        updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
      )
    `);

    createTableIfNotExists(db, "sys_role_permissions", `
      CREATE TABLE sys_role_permissions (
        id              TEXT PRIMARY KEY,
        role_id         TEXT NOT NULL REFERENCES sys_roles(id) ON DELETE CASCADE,
        action_id       TEXT NOT NULL REFERENCES sys_action_catalogs(id) ON DELETE CASCADE,
        created_at      TEXT NOT NULL DEFAULT (datetime('now')),
        UNIQUE(role_id, action_id)
      )
    `);

    createTableIfNotExists(db, "sys_subject_role_bindings", `
      CREATE TABLE sys_subject_role_bindings (
        id              TEXT PRIMARY KEY,
        subject_type    TEXT NOT NULL,
        subject_id      TEXT NOT NULL,
        role_id         TEXT NOT NULL REFERENCES sys_roles(id) ON DELETE CASCADE,
        scope_party_id  TEXT,
        valid_from      TEXT,
        valid_until     TEXT,
        created_at      TEXT NOT NULL DEFAULT (datetime('now')),
        updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
      )
    `);

    createTableIfNotExists(db, "sys_access_policies", `
      CREATE TABLE sys_access_policies (
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
      )
    `);

    createTableIfNotExists(db, "sys_access_policy_bindings", `
      CREATE TABLE sys_access_policy_bindings (
        id              TEXT PRIMARY KEY,
        policy_id       TEXT NOT NULL REFERENCES sys_access_policies(id) ON DELETE CASCADE,
        subject_type    TEXT NOT NULL,
        subject_id      TEXT NOT NULL,
        created_at      TEXT NOT NULL DEFAULT (datetime('now'))
      )
    `);

    createTableIfNotExists(db, "sys_ui_menus", `
      CREATE TABLE sys_ui_menus (
        id              TEXT PRIMARY KEY,
        menu_code       TEXT NOT NULL UNIQUE,
        parent_id       TEXT REFERENCES sys_ui_menus(id) ON DELETE SET NULL,
        label           TEXT NOT NULL,
        icon            TEXT,
        sort_order      INTEGER NOT NULL DEFAULT 0,
        created_at      TEXT NOT NULL DEFAULT (datetime('now')),
        updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
      )
    `);

    createTableIfNotExists(db, "sys_ui_routes", `
      CREATE TABLE sys_ui_routes (
        id                  TEXT PRIMARY KEY,
        route_path          TEXT NOT NULL UNIQUE,
        menu_id             TEXT REFERENCES sys_ui_menus(id) ON DELETE SET NULL,
        required_action_id  TEXT REFERENCES sys_action_catalogs(id) ON DELETE SET NULL,
        created_at          TEXT NOT NULL DEFAULT (datetime('now')),
        updated_at          TEXT NOT NULL DEFAULT (datetime('now'))
      )
    `);

    createTableIfNotExists(db, "sys_ui_action_bindings", `
      CREATE TABLE sys_ui_action_bindings (
        id                  TEXT PRIMARY KEY,
        button_code         TEXT NOT NULL,
        button_label        TEXT NOT NULL,
        page_route          TEXT NOT NULL,
        required_action_id  TEXT REFERENCES sys_action_catalogs(id) ON DELETE CASCADE,
        created_at          TEXT NOT NULL DEFAULT (datetime('now')),
        updated_at          TEXT NOT NULL DEFAULT (datetime('now')),
        UNIQUE(button_code, page_route)
      )
    `);

    // 39. sys_config
    createTableIfNotExists(db, "sys_config", `
      CREATE TABLE sys_config (
        id              TEXT PRIMARY KEY,
        config_key      TEXT NOT NULL UNIQUE,
        config_value    TEXT NOT NULL,
        value_type      TEXT NOT NULL DEFAULT 'string',
        category        TEXT DEFAULT 'general',
        description     TEXT,
        is_public       INTEGER NOT NULL DEFAULT 0,
        created_at      TEXT NOT NULL DEFAULT (datetime('now')),
        updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
      )
    `);

    // 40. audit_chain_anchors
    createTableIfNotExists(db, "audit_chain_anchors", `
      CREATE TABLE audit_chain_anchors (
        id              TEXT PRIMARY KEY,
        anchor_hash     TEXT NOT NULL UNIQUE,
        start_event_id  TEXT NOT NULL,
        end_event_id    TEXT NOT NULL,
        event_count     INTEGER NOT NULL DEFAULT 0,
        created_at      TEXT NOT NULL DEFAULT (datetime('now'))
      )
    `);

    // ===================================================================
    // SECTION 2: users table (maps from admin_users)
    // ===================================================================
    console.log("\n=== SECTION 2: Creating users table (from admin_users) ===");

    if (!tableExists(db, "users")) {
      db.exec(`
        CREATE TABLE users (
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
        )
      `);
      console.log("  [CREATE] Table users");

      // Migrate data from admin_users
      const admins = db.prepare("SELECT * FROM admin_users").all();
      if (admins.length > 0) {
        const insert = db.prepare(`
          INSERT INTO users (id, username, email, display_name, password_hash, role, status, last_login_at, created_at, updated_at)
          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        `);
        const tx = db.transaction(() => {
          for (const a of admins) {
            insert.run(
              a.id || uuid(),
              a.username || "",
              a.email || null,
              a.name || a.username || "",
              a.password_hash || null,
              a.role || "member",
              a.status || "active",
              a.last_login_at || null,
              a.created_at || now(),
              a.updated_at || now()
            );
          }
        });
        tx();
        console.log(`  [DATA] Migrated ${admins.length} rows from admin_users → users`);
      }
    } else {
      console.log("  [SKIP] Table users already exists");
    }

    // ===================================================================
    // SECTION 3: Alter existing tables — add new columns
    // ===================================================================
    console.log("\n=== SECTION 3: Adding columns to existing tables ===");

    // --- admin_sessions ---
    if (!columnExists(db, "admin_sessions", "created_at")) {
      addColumn(db, "admin_sessions", "created_at", "TEXT", "(datetime('now'))");
    }

    // --- api_keys ---
    addColumn(db, "api_keys", "owner_user_id", "TEXT");
    addColumn(db, "api_keys", "account_id", "TEXT");
    addColumn(db, "api_keys", "party_id", "TEXT");
    addColumn(db, "api_keys", "ax_key_type", "TEXT", "'user'");
    addColumn(db, "api_keys", "scopes", "TEXT");
    addColumn(db, "api_keys", "profiles", "TEXT");
    addColumn(db, "api_keys", "allowed_ips", "TEXT");
    addColumn(db, "api_keys", "issued_at", "TEXT", "(datetime('now'))");
    addColumn(db, "api_keys", "updated_at", "TEXT", "(datetime('now'))");

    // --- models ---
    addColumn(db, "models", "developer", "TEXT");
    addColumn(db, "models", "icon", "TEXT");
    addColumn(db, "models", "model_group", "TEXT");
    addColumn(db, "models", "model_card", "TEXT");
    addColumn(db, "models", "model_settings", "TEXT");
    addColumn(db, "models", "ax_model_type", "TEXT", "'chat'");
    addColumn(db, "models", "sell_input_price_per_1m", "REAL");
    addColumn(db, "models", "sell_cache_read_price_per_1m", "REAL");
    addColumn(db, "models", "sell_output_price_per_1m", "REAL");
    addColumn(db, "models", "sell_embedding_price_per_1m", "REAL");
    addColumn(db, "models", "item_codes", "TEXT");
    addColumn(db, "models", "data_classification", "TEXT", "'internal'");
    addColumn(db, "models", "network_class", "TEXT", "'external'");
    addColumn(db, "models", "updated_at", "TEXT", "(datetime('now'))");

    // --- providers ---
    addColumn(db, "providers", "ax_channel_type", "TEXT");
    addColumn(db, "providers", "credentials", "TEXT");
    addColumn(db, "providers", "supported_models", "TEXT");
    addColumn(db, "providers", "policies", "TEXT");
    addColumn(db, "providers", "channel_settings", "TEXT");
    addColumn(db, "providers", "endpoints", "TEXT");
    addColumn(db, "providers", "ordering_weight", "INTEGER", "0");
    addColumn(db, "providers", "error_message", "TEXT");
    addColumn(db, "providers", "remark", "TEXT");
    addColumn(db, "providers", "account_id", "TEXT");
    addColumn(db, "providers", "updated_at", "TEXT", "(datetime('now'))");

    // --- provider_resources ---
    addColumn(db, "provider_resources", "updated_at", "TEXT", "(datetime('now'))");

    // --- provider_models ---
    addColumn(db, "provider_models", "updated_at", "TEXT", "(datetime('now'))");

    // --- model_routes ---
    addColumn(db, "model_routes", "route_profile_id", "TEXT");
    addColumn(db, "model_routes", "channel_id", "TEXT");
    addColumn(db, "model_routes", "model_price_id", "TEXT");
    addColumn(db, "model_routes", "price_cap_delta", "REAL", "0.0");
    addColumn(db, "model_routes", "updated_at", "TEXT", "(datetime('now'))");

    // --- request_logs ---
    addColumn(db, "request_logs", "stream", "INTEGER", "0");
    addColumn(db, "request_logs", "format", "TEXT", "'openai/chat_completions'");
    addColumn(db, "request_logs", "request_body", "TEXT");
    addColumn(db, "request_logs", "response_body", "TEXT");
    addColumn(db, "request_logs", "ax_status", "TEXT");
    addColumn(db, "request_logs", "first_token_latency_ms", "INTEGER");
    addColumn(db, "request_logs", "reasoning_duration_ms", "INTEGER");
    addColumn(db, "request_logs", "account_id", "TEXT");
    addColumn(db, "request_logs", "freeze_id", "TEXT");
    addColumn(db, "request_logs", "user_id", "TEXT");
    addColumn(db, "request_logs", "party_id", "TEXT");
    addColumn(db, "request_logs", "cost_usd", "REAL");
    addColumn(db, "request_logs", "sell_usd", "REAL");
    addColumn(db, "request_logs", "cost_items", "TEXT");
    addColumn(db, "request_logs", "usage_incomplete", "INTEGER", "0");
    addColumn(db, "request_logs", "business_tags", "TEXT");
    addColumn(db, "request_logs", "route_profile_id", "TEXT");
    addColumn(db, "request_logs", "updated_at", "TEXT", "(datetime('now'))");

    // --- request_payload_logs --- (no structural changes needed beyond what exists)
    // Already has request_body, response_body as TEXT (JSON compatible)

    // --- route_attempt_logs --- (no new columns from v3.2 DDL)

    // --- usage_records ---
    addColumn(db, "usage_records", "channel_id", "TEXT");
    addColumn(db, "usage_records", "prompt_audio_tokens", "INTEGER", "0");
    addColumn(db, "usage_records", "prompt_cached_tokens", "INTEGER", "0");
    addColumn(db, "usage_records", "prompt_write_cached_tokens", "INTEGER", "0");
    addColumn(db, "usage_records", "prompt_write_cached_5m", "INTEGER", "0");
    addColumn(db, "usage_records", "prompt_write_cached_1h", "INTEGER", "0");
    addColumn(db, "usage_records", "completion_audio_tokens", "INTEGER", "0");
    addColumn(db, "usage_records", "completion_reasoning_tokens", "INTEGER", "0");
    addColumn(db, "usage_records", "accepted_prediction_tokens", "INTEGER", "0");
    addColumn(db, "usage_records", "rejected_prediction_tokens", "INTEGER", "0");
    addColumn(db, "usage_records", "source", "TEXT", "'api'");
    addColumn(db, "usage_records", "format", "TEXT", "'openai/chat_completions'");
    addColumn(db, "usage_records", "total_cost", "REAL");
    addColumn(db, "usage_records", "cost_price_ref_id", "TEXT");
    addColumn(db, "usage_records", "sell_usd", "REAL");
    addColumn(db, "usage_records", "cost_items", "TEXT");
    addColumn(db, "usage_records", "account_id", "TEXT");
    addColumn(db, "usage_records", "freeze_id", "TEXT");
    addColumn(db, "usage_records", "item_code", "TEXT");
    addColumn(db, "usage_records", "updated_at", "TEXT", "(datetime('now'))");

    // --- audit_events ---
    addColumn(db, "audit_events", "before_snapshot", "TEXT");
    addColumn(db, "audit_events", "after_snapshot", "TEXT");

    // ===================================================================
    // SECTION 4: Migrate projects → parties
    // ===================================================================
    console.log("\n=== SECTION 4: Migrating projects → parties ===");
    const partyCount = db.prepare("SELECT COUNT(*) as c FROM parties").get().c;
    if (partyCount === 0) {
      const projects = db.prepare("SELECT * FROM projects").all();
      if (projects.length > 0) {
        const insertParty = db.prepare(`
          INSERT INTO parties (id, type, name, cost_center, leader_user_id, status, created_at, updated_at)
          VALUES (?, 'project', ?, ?, ?, ?, ?, ?)
        `);
        const tx = db.transaction(() => {
          for (const p of projects) {
            insertParty.run(
              p.id,
              p.name,
              p.cost_center || null,
              p.owner_user_id || null,
              p.status || "active",
              p.created_at || now(),
              p.updated_at || now()
            );
          }
        });
        tx();
        console.log(`  [DATA] Migrated ${projects.length} projects → parties`);
      }

      // Also create a default "admin" org party
      const adminPartyId = uuid();
      db.prepare(`
        INSERT INTO parties (id, type, name, description, status, created_at, updated_at)
        VALUES (?, 'org', '默认组织', '系统默认组织', 'active', datetime('now'), datetime('now'))
      `).run(adminPartyId);
      console.log(`  [DATA] Created default org party: ${adminPartyId}`);
    } else {
      console.log(`  [SKIP] parties already has ${partyCount} rows`);
    }

    // ===================================================================
    // SECTION 5: Migrate project_teams → party_members
    // ===================================================================
    console.log("\n=== SECTION 5: Migrating project_teams → party_members ===");
    const pmCount = db.prepare("SELECT COUNT(*) as c FROM party_members").get().c;
    if (pmCount === 0) {
      const teams = db.prepare("SELECT * FROM project_teams").all();
      if (teams.length > 0) {
        const insertPM = db.prepare(`
          INSERT INTO party_members (id, party_id, user_id, role, joined_at, created_at, updated_at)
          VALUES (?, ?, ?, ?, datetime('now'), datetime('now'), datetime('now'))
        `);
        const tx = db.transaction(() => {
          for (const t of teams) {
            insertPM.run(uuid(), t.project_id, t.team_id, t.role || "member");
          }
        });
        tx();
        console.log(`  [DATA] Migrated ${teams.length} project_teams → party_members`);
      }
    } else {
      console.log(`  [SKIP] party_members already has ${pmCount} rows`);
    }

    // ===================================================================
    // SECTION 6: Create admin users if none exist
    // ===================================================================
    console.log("\n=== SECTION 6: Seed data ===");

    const userCount = db.prepare("SELECT COUNT(*) as c FROM users").get().c;
    if (userCount === 0) {
      const adminId = uuid();
      db.prepare(`
        INSERT INTO users (id, username, email, display_name, role, status, created_at, updated_at)
        VALUES (?, 'admin', 'admin@ai-gov.local', '超级管理员', 'admin', 'active', datetime('now'), datetime('now'))
      `).run(adminId);
      console.log(`  [SEED] Created admin user: ${adminId}`);
    } else {
      console.log(`  [SKIP] users already has ${userCount} rows`);
    }

    // Get the admin user ID for subsequent seeds
    const adminUser = db.prepare("SELECT id FROM users WHERE username = 'admin' LIMIT 1").get();
    const adminUserId = adminUser ? adminUser.id : null;

    // Get or create default org party
    let orgParty = db.prepare("SELECT id FROM parties WHERE type = 'org' AND name = '默认组织' LIMIT 1").get();
    const orgPartyId = orgParty ? orgParty.id : null;

    // Create account for org party (if not exists)
    if (orgPartyId) {
      const acctCount = db.prepare("SELECT COUNT(*) as c FROM accounts WHERE party_id = ?").get(orgPartyId).c;
      if (acctCount === 0) {
        db.prepare(`
          INSERT INTO accounts (id, party_id, available_balance, status, created_at, updated_at)
          VALUES (?, ?, 100000.0, 'active', datetime('now'), datetime('now'))
        `).run(uuid(), orgPartyId);
        console.log(`  [SEED] Created account for org party ${orgPartyId}`);
      }
    }

    // Create a seed API key for admin (if none)
    const apiKeyCount = db.prepare("SELECT COUNT(*) as c FROM api_keys WHERE owner_user_id IS NOT NULL").get().c;
    if (apiKeyCount === 0 && adminUserId && orgPartyId) {
      const account = db.prepare("SELECT id FROM accounts WHERE party_id = ? LIMIT 1").get(orgPartyId);
      if (account) {
        const keyId = uuid();
        const keyStr = "sk-admin-" + randomUUID().slice(0, 16);
        db.prepare(`
          INSERT INTO api_keys (id, name, key_hash, key_prefix, status, owner_user_id, account_id, party_id, issued_at, created_at, updated_at)
          VALUES (?, '默认管理员Key', ?, 'sk-admin', 'active', ?, ?, ?, datetime('now'), datetime('now'), datetime('now'))
        `).run(keyId, sha256(keyStr), adminUserId, account.id, orgPartyId);
        console.log(`  [SEED] Created admin API key (prefix: sk-admin)`);
      }
    }

    // ===================================================================
    // SECTION 7: Seed ABAC governance data
    // ===================================================================
    console.log("\n=== SECTION 7: Seeding ABAC governance data ===");

    // --- sys_action_catalogs ---
    const actionCount = db.prepare("SELECT COUNT(*) as c FROM sys_action_catalogs").get().c;
    if (actionCount === 0) {
      const actions = [
        // fund axis
        ["fund", "balance.read", "查看余额", "account"],
        ["fund", "balance.allocate", "划拨资金", "account"],
        ["fund", "balance.freeze", "冻结资金", "account"],
        ["fund", "balance.liquidate", "清算账户", "account"],
        ["fund", "ledger.read", "查看流水", "ledger"],
        // data axis
        ["data", "model.invoke", "调用模型", "model"],
        ["data", "model.list", "查看模型列表", "model"],
        ["data", "model.create", "创建模型", "model"],
        ["data", "model.update", "更新模型", "model"],
        ["data", "model.delete", "删除模型", "model"],
        ["data", "usage.read", "查看用量", "usage"],
        // iam axis
        ["iam", "user.create", "创建用户", "user"],
        ["iam", "user.update", "更新用户", "user"],
        ["iam", "user.delete", "删除用户", "user"],
        ["iam", "key.create", "创建API Key", "key"],
        ["iam", "key.revoke", "吊销API Key", "key"],
        ["iam", "role.assign", "分配角色", "role"],
        ["iam", "policy.manage", "管理策略", "policy"],
        // routing axis
        ["routing", "route.read", "查看路由", "route"],
        ["routing", "route.update", "更新路由", "route"],
        ["routing", "channel.manage", "管理渠道", "channel"],
      ];

      const insertAction = db.prepare(`
        INSERT INTO sys_action_catalogs (id, axis, action_code, action_name, resource_type, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, datetime('now'), datetime('now'))
      `);
      const tx = db.transaction(() => {
        for (const [axis, code, name, rtype] of actions) {
          insertAction.run(uuid(), axis, code, name, rtype || null);
        }
      });
      tx();
      console.log(`  [SEED] Created ${actions.length} action catalogs`);
    }

    // --- sys_roles ---
    const roleCount = db.prepare("SELECT COUNT(*) as c FROM sys_roles").get().c;
    if (roleCount === 0) {
      const roles = [
        ["super_admin", "超级管理员", "系统最高权限，拥有所有操作权限", 1],
        ["finance_mgr", "财务管理", "管理资金、账本、划拨、清算", 1],
        ["model_admin", "模型管理员", "管理模型目录、定价、路由", 1],
        ["model_user", "模型用户", "调用模型、查看用量", 0],
        ["auditor", "审计员", "查看审计日志和流水", 0],
      ];

      const roleIds = {};
      const insertRole = db.prepare(`
        INSERT INTO sys_roles (id, role_code, role_name, description, is_system, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, datetime('now'), datetime('now'))
      `);
      const tx = db.transaction(() => {
        for (const [code, name, desc, isSys] of roles) {
          const id = uuid();
          roleIds[code] = id;
          insertRole.run(id, code, name, desc, isSys);
        }
      });
      tx();
      console.log(`  [SEED] Created ${roles.length} roles`);

      // --- sys_role_permissions: super_admin gets everything ---
      const allActions = db.prepare("SELECT id, action_code FROM sys_action_catalogs").all();
      const insertPerm = db.prepare(`
        INSERT INTO sys_role_permissions (id, role_id, action_id, created_at)
        VALUES (?, ?, ?, datetime('now'))
      `);
      const tx2 = db.transaction(() => {
        for (const a of allActions) {
          insertPerm.run(uuid(), roleIds["super_admin"], a.id);
        }
        // finance_mgr gets fund actions
        for (const a of allActions) {
          if (a.action_code.startsWith("balance.") || a.action_code.startsWith("ledger.")) {
            insertPerm.run(uuid(), roleIds["finance_mgr"], a.id);
          }
        }
        // model_admin gets model + routing + usage actions
        for (const a of allActions) {
          if (a.action_code.startsWith("model.") || a.action_code.startsWith("route.") || a.action_code.startsWith("channel.") || a.action_code === "usage.read") {
            insertPerm.run(uuid(), roleIds["model_admin"], a.id);
          }
        }
        // model_user gets model.invoke, model.list, usage.read
        for (const a of allActions) {
          if (["model.invoke", "model.list", "usage.read"].includes(a.action_code)) {
            insertPerm.run(uuid(), roleIds["model_user"], a.id);
          }
        }
        // auditor gets balance.read, ledger.read, usage.read
        for (const a of allActions) {
          if (["balance.read", "ledger.read", "usage.read"].includes(a.action_code)) {
            insertPerm.run(uuid(), roleIds["auditor"], a.id);
          }
        }
      });
      tx2();
      console.log("  [SEED] Assigned role permissions");

      // --- sys_subject_role_bindings: bind admin user to super_admin ---
      const bindingCount = db.prepare("SELECT COUNT(*) as c FROM sys_subject_role_bindings").get().c;
      if (bindingCount === 0 && adminUserId) {
        db.prepare(`
          INSERT INTO sys_subject_role_bindings (id, subject_type, subject_id, role_id, scope_party_id, created_at, updated_at)
          VALUES (?, 'user', ?, ?, NULL, datetime('now'), datetime('now'))
        `).run(uuid(), adminUserId, roleIds["super_admin"]);
        console.log("  [SEED] Bound admin user to super_admin role");
      }
    }

    // --- sys_access_policies: 4 built-in ABAC policies ---
    const policyCount = db.prepare("SELECT COUNT(*) as c FROM sys_access_policies").get().c;
    if (policyCount === 0) {
      const policies = [
        ["P-DENY-EXTERNAL-MODEL", "禁止外部网络访问受限模型", "deny",
         JSON.stringify({ axis: "data", actions: ["model.invoke"], resource_type: "model", conditions: { data_classification: ["confidential", "restricted"], network_class: "external" } }),
         100, 1, "阻止从外部网络调用机密/受限级别的模型"],
        ["P-ALLOW-WORKHOURS", "仅工作时间允许调用", "allow",
         JSON.stringify({ axis: "data", actions: ["model.invoke"], resource_type: "model", conditions: { time_restriction: { start: "09:00", end: "18:00", timezone: "Asia/Shanghai" } } }),
         50, 1, "默认仅在工作时间 (09:00-18:00) 允许模型调用"],
        ["P-DENY-OVER-BUDGET", "超预算禁止调用", "deny",
         JSON.stringify({ axis: "fund", actions: ["model.invoke"], resource_type: "account", conditions: { budget_exceeded: true } }),
         90, 1, "当账户预算消耗超过限额时阻止模型调用"],
        ["P-ALLOW-INTERNAL-ALL", "内部网络全通", "allow",
         JSON.stringify({ axis: "data", actions: ["model.invoke", "model.list"], resource_type: "model", conditions: { network_class: "internal" } }),
         10, 1, "内部网络允许所有模型数据操作"],
      ];

      const insertPolicy = db.prepare(`
        INSERT INTO sys_access_policies (id, policy_code, policy_name, effect, conditions_json, priority, is_system, description, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, 1, ?, datetime('now'), datetime('now'))
      `);
      const tx = db.transaction(() => {
        for (const [code, name, effect, cond, pri, _, desc] of policies) {
          insertPolicy.run(uuid(), code, name, effect, cond, pri, desc);
        }
      });
      tx();
      console.log(`  [SEED] Created ${policies.length} ABAC policies`);
    }

    // --- sys_access_policy_bindings ---
    // Bind P-ALLOW-WORKHOURS to model_user role
    const apbCount = db.prepare("SELECT COUNT(*) as c FROM sys_access_policy_bindings").get().c;
    if (apbCount === 0) {
      const modelUserRole = db.prepare("SELECT id FROM sys_roles WHERE role_code = 'model_user' LIMIT 1").get();
      const allowWorkPolicy = db.prepare("SELECT id FROM sys_access_policies WHERE policy_code = 'P-ALLOW-WORKHOURS' LIMIT 1").get();
      const allowInternalPolicy = db.prepare("SELECT id FROM sys_access_policies WHERE policy_code = 'P-ALLOW-INTERNAL-ALL' LIMIT 1").get();

      if (modelUserRole && allowWorkPolicy) {
        db.prepare(`
          INSERT INTO sys_access_policy_bindings (id, policy_id, subject_type, subject_id, created_at)
          VALUES (?, ?, 'role', ?, datetime('now'))
        `).run(uuid(), allowWorkPolicy.id, modelUserRole.id);
        console.log("  [SEED] Bound P-ALLOW-WORKHOURS to model_user role");
      }
      if (modelUserRole && allowInternalPolicy) {
        db.prepare(`
          INSERT INTO sys_access_policy_bindings (id, policy_id, subject_type, subject_id, created_at)
          VALUES (?, ?, 'role', ?, datetime('now'))
        `).run(uuid(), allowInternalPolicy.id, modelUserRole.id);
        console.log("  [SEED] Bound P-ALLOW-INTERNAL-ALL to model_user role");
      }
    }

    // --- sys_ui_menus ---
    const menuCount = db.prepare("SELECT COUNT(*) as c FROM sys_ui_menus").get().c;
    if (menuCount === 0) {
      const menus = [
        ["dashboard", null, "仪表盘", "dashboard", 0],
        ["models", null, "模型管理", "model", 10],
        ["finance", null, "财务管理", "finance", 20],
        ["admin", null, "系统管理", "admin", 30],
        ["model-catalog", "models", "模型目录", null, 0],
        ["model-routes", "models", "路由管理", null, 1],
        ["model-pricing", "models", "定价管理", null, 2],
        ["accounts", "finance", "账本管理", null, 0],
        ["ledgers", "finance", "流水查询", null, 1],
        ["allocations", "finance", "划拨记录", null, 2],
        ["users", "admin", "用户管理", null, 0],
        ["roles", "admin", "角色管理", null, 1],
        ["policies", "admin", "策略管理", null, 2],
        ["audit", "admin", "审计日志", null, 3],
      ];

      const menuIds = {};
      const insertMenu = db.prepare(`
        INSERT INTO sys_ui_menus (id, menu_code, parent_id, label, icon, sort_order, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))
      `);
      const tx = db.transaction(() => {
        for (const [code, parentCode, label, icon, sort] of menus) {
          const id = uuid();
          menuIds[code] = id;
          const parentId = parentCode ? menuIds[parentCode] || null : null;
          insertMenu.run(id, code, parentId, label, icon || null, sort);
        }
      });
      tx();
      console.log(`  [SEED] Created ${menus.length} UI menus`);
    }

    // --- sys_config ---
    const configCount = db.prepare("SELECT COUNT(*) as c FROM sys_config").get().c;
    if (configCount === 0) {
      const configs = [
        ["system.default_language", "zh-CN", "string", "general", "系统默认语言", 1],
        ["system.site_name", "AI Governance Gateway", "string", "general", "站点名称", 1],
        ["security.session_timeout_min", "480", "integer", "security", "会话超时时间(分钟)", 0],
        ["billing.default_daily_limit", "10000", "integer", "billing", "默认每日用量限额", 0],
        ["routing.default_max_attempts", "3", "integer", "routing", "默认最大路由尝试次数", 0],
        ["routing.default_timeout_ms", "30000", "integer", "routing", "默认请求超时(毫秒)", 0],
      ];

      db.prepare(`
        INSERT INTO sys_config (id, config_key, config_value, value_type, category, description, is_public, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))
      `);
      const tx = db.transaction(() => {
        for (const [key, val, vtype, cat, desc, pub] of configs) {
          db.prepare(`
            INSERT INTO sys_config (id, config_key, config_value, value_type, category, description, is_public, created_at, updated_at)
            VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))
          `).run(uuid(), key, val, vtype, cat, desc, pub);
        }
      });
      tx();
      console.log(`  [SEED] Created ${configs.length} system configs`);
    }

    // ===================================================================
    // SECTION 8: Create indexes for new tables
    // ===================================================================
    console.log("\n=== SECTION 8: Creating indexes ===");
    const indexDefs = [
      ["idx_parties_type", "parties", "type"],
      ["idx_parties_parent", "parties", "parent_party_id"],
      ["idx_parties_status", "parties", "status"],
      ["idx_party_edges_src", "party_edges", "src_party_id"],
      ["idx_party_edges_dst", "party_edges", "dst_party_id"],
      ["idx_party_members_party", "party_members", "party_id"],
      ["idx_party_members_user", "party_members", "user_id"],
      ["idx_accounts_party", "accounts", "party_id"],
      ["idx_accounts_status", "accounts", "status"],
      ["idx_ledgers_account", "ledgers", "account_id, created_at"],
      ["idx_ledgers_freeze", "ledgers", "freeze_id"],
      ["idx_ledgers_request", "ledgers", "request_id"],
      ["idx_ledgers_idem", "ledgers", "account_id, idempotency_key"],
      ["idx_freezes_account", "freezes", "account_id"],
      ["idx_freezes_request", "freezes", "request_id"],
      ["idx_freezes_expiry", "freezes", "status, expires_at"],
      ["idx_freezes_user", "freezes", "user_id"],
      ["idx_allocations_src", "allocations", "src_account_id"],
      ["idx_allocations_dst", "allocations", "dst_account_id"],
      ["idx_allocations_idem", "allocations", "src_account_id, idempotency_key"],
      ["idx_liquidations_party", "liquidations", "party_id"],
      ["idx_liquidations_account", "liquidations", "account_id"],
      ["idx_liquidations_status", "liquidations", "status"],
      ["idx_model_prices_model", "model_prices", "model_id, channel_id"],
      ["idx_model_prices_ref", "model_prices", "reference_id"],
      ["idx_model_routes_profile", "model_routes", "route_profile_id"],
      ["idx_grants_principal", "grants", "principal_type, principal_id"],
      ["idx_grants_axis", "grants", "axis, action"],
      ["idx_grants_resource", "grants", "resource_type, resource_id"],
      ["idx_model_grants_principal", "model_grants", "principal_type, principal_id"],
      ["idx_model_grants_model", "model_grants", "model_id"],
      ["idx_model_grants_effect", "model_grants", "effect"],
      ["idx_users_status", "users", "status"],
      ["idx_users_email", "users", "email"],
      ["idx_api_keys_owner", "api_keys", "owner_user_id"],
      ["idx_api_keys_account", "api_keys", "account_id"],
      ["idx_api_keys_party", "api_keys", "party_id"],
      ["idx_api_keys_status", "api_keys", "status"],
      ["idx_request_logs_account", "request_logs", "account_id, created_at"],
      ["idx_request_logs_user", "request_logs", "user_id, created_at"],
      ["idx_request_logs_error", "request_logs", "error_code, created_at"],
      ["idx_usage_records_user", "usage_records", "attributed_user_id"],
      ["idx_usage_records_account", "usage_records", "account_id"],
      ["idx_channel_probes_ch", "channel_probes", "channel_id, observed_at"],
      ["idx_prov_quota_status", "provider_quota_status", "status"],
      ["idx_prov_quota_check", "provider_quota_status", "next_check_at"],
      ["idx_audit_events_resource", "audit_events", "resource_type, resource_id, created_at"],
      ["idx_audit_events_action", "audit_events", "action, created_at"],
      ["idx_idempotency_expiry", "idempotency_records", "expires_at"],
      ["idx_sys_action_catalogs_axis", "sys_action_catalogs", "axis"],
      ["idx_sys_action_catalogs_resource", "sys_action_catalogs", "resource_type"],
      ["idx_sys_roles_code", "sys_roles", "role_code"],
      ["idx_sys_role_perms_role", "sys_role_permissions", "role_id"],
      ["idx_sys_role_perms_action", "sys_role_permissions", "action_id"],
      ["idx_sys_srb_subject", "sys_subject_role_bindings", "subject_type, subject_id"],
      ["idx_sys_srb_role", "sys_subject_role_bindings", "role_id"],
      ["idx_sys_srb_scope", "sys_subject_role_bindings", "scope_party_id"],
      ["idx_sys_srb_validity", "sys_subject_role_bindings", "valid_from, valid_until"],
      ["idx_sys_access_policies_code", "sys_access_policies", "policy_code"],
      ["idx_sys_access_policies_effect", "sys_access_policies", "effect"],
      ["idx_sys_access_policies_priority", "sys_access_policies", "priority DESC"],
      ["idx_sys_apb_policy", "sys_access_policy_bindings", "policy_id"],
      ["idx_sys_apb_subject", "sys_access_policy_bindings", "subject_type, subject_id"],
      ["idx_sys_ui_menus_parent", "sys_ui_menus", "parent_id"],
      ["idx_sys_ui_menus_sort", "sys_ui_menus", "parent_id, sort_order"],
      ["idx_sys_ui_routes_menu", "sys_ui_routes", "menu_id"],
      ["idx_sys_ui_routes_action", "sys_ui_routes", "required_action_id"],
      ["idx_sys_ui_ab_page", "sys_ui_action_bindings", "page_route"],
      ["idx_sys_ui_ab_action", "sys_ui_action_bindings", "required_action_id"],
      ["idx_audit_chain_anchors_hash", "audit_chain_anchors", "anchor_hash"],
      ["idx_audit_chain_anchors_start", "audit_chain_anchors", "start_event_id"],
      ["idx_audit_chain_anchors_end", "audit_chain_anchors", "end_event_id"],
      ["idx_audit_chain_anchors_created", "audit_chain_anchors", "created_at"],
      ["idx_sys_config_key", "sys_config", "config_key"],
      ["idx_sys_config_category", "sys_config", "category"],
    ];

    let idxCreated = 0;
    let idxSkipped = 0;
    for (const [idxName, table, cols] of indexDefs) {
      const existing = db.prepare("SELECT name FROM sqlite_master WHERE type='index' AND name=?").get(idxName);
      if (!existing) {
        try {
          db.exec(`CREATE INDEX ${idxName} ON ${table}(${cols})`);
          idxCreated++;
        } catch (e) {
          console.log(`  [WARN] Could not create ${idxName}: ${e.message}`);
        }
      } else {
        idxSkipped++;
      }
    }
    console.log(`  Indexes: ${idxCreated} created, ${idxSkipped} already existed`);

    // ===================================================================
    // SECTION 9: Final verification
    // ===================================================================
    console.log("\n========================================");
    console.log("=== MIGRATION COMPLETE ===");
    console.log("========================================");

    const allTables = db
      .prepare("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
      .all();
    console.log(`\nTotal tables: ${allTables.length}`);
    for (const t of allTables) {
      const count = db.prepare(`SELECT COUNT(*) as c FROM "${t.name}"`).get().c;
      console.log(`  ${t.name} (${count} rows)`);
    }

    // Verify key table counts
    console.log("\n--- Key seed data counts ---");
    const keyTables = [
      "users", "parties", "party_members", "accounts", "api_keys",
      "sys_action_catalogs", "sys_roles", "sys_role_permissions",
      "sys_subject_role_bindings", "sys_access_policies", "sys_access_policy_bindings",
      "sys_ui_menus", "sys_config",
    ];
    for (const t of keyTables) {
      if (tableExists(db, t)) {
        const c = db.prepare(`SELECT COUNT(*) as c FROM "${t}"`).get().c;
        console.log(`  ${t}: ${c} rows`);
      }
    }
  } finally {
    db.close();
  }
}

main().catch((err) => {
  console.error("Migration failed:", err);
  process.exit(1);
});
