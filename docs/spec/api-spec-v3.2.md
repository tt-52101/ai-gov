# AI-GOV Governance API Specification v3.2.0

> **Baseline:** PRD v3.2.0 | PRD v3.1.0 + v2.0.2 value backfill  
> **Data Model:** schema/ai-gov-fusion-minimal.sql (29 tables) + ABAC/UI governance extension  
> **Governance Principle:** All control plane APIs share a single ABAC engine; the Governance API is capability-equivalent to the Console.

---

## 1. Conventions

### 1.1 General

| Convention | Value |
|---|---|
| Base path | `/gov` |
| Content-Type | `application/json` |
| Accept | `application/json` |
| Auth | `Authorization: Bearer <token>` (admin session) or `X-API-Key: <gateway_key>` |
| Authorization | All requests evaluated by ABAC engine (PRD 7.2). UI permission projection is derived from the same engine. |
| Idempotency | `Idempotency-Key` header (UUID v4, max 255 chars) required on all fund write endpoints. Window >= 24h. |
| Pagination | `?page=1&page_size=20` (default page_size=20, max=200); response includes `total`, `page`, `page_size`, `pages` |
| Date/time | All timestamps in ISO 8601 / RFC 3339 (`2026-07-31T10:30:00Z`) |
| Language | `Accept-Language: zh-CN` or `en`; error messages support both |

### 1.2 Error Format

All error responses follow this structure:

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable message in requested locale",
    "details": {}
  }
}
```

### 1.3 HTTP Status Codes (per PRD §6)

| Code | Meaning | Example Error Codes |
|---|---|---|
| 200 | Success | IDEMPOTENCY_REPLAY |
| 400 | Bad request / validation | |
| 401 | Authentication invalid | AUTH_INVALID_KEY |
| 402 | Payment required | BUDGET_CAP_EXCEEDED, MODEL_BUDGET_EXCEEDED, INSUFFICIENT_BALANCE |
| 403 | Forbidden | ACCOUNT_FROZEN_OR_CLOSED, AUTH_USER_DISABLED, AUTH_KEY_NO_ACCOUNT, AUTHZ_DENIED, MODEL_ACCESS_DENIED, ROUTE_COMPLIANCE_BLOCKED, COMPLIANCE_NETWORK_BLOCKED, CONTENT_BLOCKED |
| 404 | Not found | |
| 409 | Conflict | FREEZE_EXPIRED, IDEMPOTENCY_CONFLICT |
| 422 | Unprocessable (business rule) | NO_ROUTE_WITHIN_PRICE_CAP |
| 429 | Rate limited | RATE_LIMITED |
| 500 | Internal error | INTERNAL_ERROR |
| 502 | Bad gateway | UPSTREAM_ERROR |
| 503 | Service unavailable | NO_ROUTE_AVAILABLE |
| 504 | Gateway timeout | UPSTREAM_TIMEOUT |

### 1.4 Idempotency (PRD §8.6 / §8.7)

| Rule | Value |
|---|---|
| Header | `Idempotency-Key: <UUID v4>` |
| Max length | 255 chars |
| Retention window | >= 24 hours |
| Mechanism | INSERT ON CONFLICT atomic claim on `(scope, actor_id, idempotency_key)` |
| Same key, same hash | Replay stored response (200 IDEMPOTENCY_REPLAY) |
| Same key, different hash | Reject (409 IDEMPOTENCY_CONFLICT) |
| Applies to | All fund write endpoints: allocate, liquidate, compensate, organization merge/split |
| Response header | `Idempotent-Replayed: true` when replaying |

### 1.5 ABAC Action Naming Convention

Actions follow `{axis}.{resource}.{verb}` format:
- axis: `data`, `fund`, `iam`, `routing`
- resource: domain entity (`usage`, `balance`, `ledger`, `key`, `price`, `route_profile`, `model_grant`, `role`, `policy`, etc.)
- verb: `read`, `write`, `create`, `delete`, `allocate`, `liquidate`, `bind`, `evaluate`, etc.

Each endpoint specification below declares the required ABAC action.

---

## 2. Party (主体管理)

> **PRD references:** FUN-01/02, UI-02, §2 (Physical World Party Model)  
> **Tables:** `parties`, `party_edges`, `party_members`

### 2.1 Create Party

```http
POST /gov/parties
```
**ABAC action:** `iam.party.create`

**Request Body:**
```json
{
  "type": "org",
  "name": "AI R&D Department",
  "description": "Central AI research and development",
  "parent_party_id": null,
  "leader_user_id": "uuid-of-leader",
  "cost_center": "CC-AI-001",
  "metadata": {}
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| type | string (enum: `org`, `project`) | yes | Party type. Both share the same API surface (PRD §2.3). |
| name | string (1-128) | yes | Display name, unique within parent scope |
| description | string (0-1024) | no | |
| parent_party_id | string (UUID) | no | Optional organizational parent; projects may omit this (PRD §2.3). |
| leader_user_id | string (UUID) | no | Responsible person. No automatic privileges (A-CON-05). |
| cost_center | string | no | Financial cost center code |
| metadata | object | no | Arbitrary JSON metadata |

**Response 201:**
```json
{
  "id": "uuid",
  "type": "org",
  "name": "AI R&D Department",
  "description": "Central AI research and development",
  "parent_party_id": null,
  "leader_user_id": "uuid-of-leader",
  "cost_center": "CC-AI-001",
  "status": "active",
  "metadata": {},
  "created_at": "2026-07-31T10:00:00Z",
  "updated_at": "2026-07-31T10:00:00Z"
}
```

**Error codes:** 403 (AUTHZ_DENIED), 400 (validation)

---

### 2.2 List Parties

```http
GET /gov/parties
```
**ABAC action:** `data.party.read` (ABAC applies data-scope filter per D-CON-01)

**Query Parameters:**

| Param | Type | Default | Description |
|---|---|---|---|
| type | string (enum: `org`, `project`) | all | Filter by party type |
| parent_party_id | string (UUID) | all | Filter by parent |
| status | string (enum: `active`, `archived`, `liquidating`) | `active` | Filter by status |
| search | string | | Search name/description (LIKE) |
| page | integer | 1 | |
| page_size | integer | 20 | Max 200 |

**Response 200:**
```json
{
  "data": [
    {
      "id": "uuid",
      "type": "org",
      "name": "AI R&D Department",
      "parent_party_id": null,
      "leader_user_id": "uuid",
      "status": "active",
      "created_at": "2026-07-31T10:00:00Z"
    }
  ],
  "total": 42,
  "page": 1,
  "page_size": 20,
  "pages": 3
}
```

**Error codes:** 403 (AUTHZ_DENIED)

---

### 2.3 Get Party

```http
GET /gov/parties/{party_id}
```
**ABAC action:** `data.party.read`

**Path Parameters:**

| Param | Type | Description |
|---|---|---|
| party_id | string (UUID) | |

**Response 200:**
```json
{
  "id": "uuid",
  "type": "org",
  "name": "AI R&D Department",
  "description": "Central AI research and development",
  "parent_party_id": null,
  "leader_user_id": "uuid-of-leader",
  "leader_name": "Zhang San",
  "cost_center": "CC-AI-001",
  "status": "active",
  "metadata": {},
  "account": {
    "id": "account-uuid",
    "available_balance": 500000.00,
    "frozen_balance": 15000.00,
    "status": "active"
  },
  "member_count": 12,
  "created_at": "2026-07-31T10:00:00Z",
  "updated_at": "2026-07-31T10:00:00Z"
}
```

**Error codes:** 404, 403 (AUTHZ_DENIED)

---

### 2.4 Update Party

```http
PATCH /gov/parties/{party_id}
```
**ABAC action:** `iam.party.write`

**Request Body (all fields optional):**
```json
{
  "name": "Updated Name",
  "description": "Updated description",
  "leader_user_id": "new-leader-uuid",
  "cost_center": "CC-AI-002",
  "status": "archived",
  "metadata": {}
}
```

| Field | Type | Description |
|---|---|---|
| name | string (1-128) | |
| description | string (0-1024) | |
| leader_user_id | string (UUID) | |
| cost_center | string | |
| status | string (enum: `active`, `archived`) | Cannot directly set to `liquidating` -- use liquidation flow (§3.3) |
| metadata | object | Replaces existing metadata entirely |

**Response 200:** Full party object as in GET.

**Error codes:** 404, 403 (AUTHZ_DENIED), 400 (validation), 409 (status transition conflict)

---

### 2.5 Create Party Edge

```http
POST /gov/party-edges
```
**ABAC action:** `iam.party.write`

**Request Body:**
```json
{
  "src_party_id": "uuid-src",
  "dst_party_id": "uuid-dst",
  "edge_type": "parent",
  "allows_fund": true
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| src_party_id | string (UUID) | yes | Source party |
| dst_party_id | string (UUID) | yes | Target party |
| edge_type | string (enum) | yes | `parent`, `sponsors`, `owns`, `participates`, `allocates`, `merged_into`, `split_from` (PRD §2.4) |
| allows_fund | boolean | no (default per edge_type) | Whether to auto-open a fund transfer channel. Default `true` for `parent`/`sponsors`/`allocates`; default `false` for `owns`/`participates`. |

**Validation rules:**
- `parent`: src is parent, dst is child. Only one parent edge per pair in that direction.
- `sponsors`: src is sponsor, dst is sponsored project.
- `allocates`: src is Party (org/project), dst must be a Person Account's owner party.
- Duplicate `(src, dst, edge_type)` rejected (409).
- Self-referencing edges rejected (400).

**Response 201:**
```json
{
  "id": "uuid",
  "src_party_id": "uuid-src",
  "dst_party_id": "uuid-dst",
  "edge_type": "parent",
  "allows_fund": true,
  "created_at": "2026-07-31T10:00:00Z"
}
```

**Error codes:** 403 (AUTHZ_DENIED), 400 (validation), 409 (duplicate)

---

### 2.6 Delete Party Edge

```http
DELETE /gov/party-edges/{edge_id}
```
**ABAC action:** `iam.party.write`

**Path Parameters:**

| Param | Type | Description |
|---|---|---|
| edge_id | string (UUID) | |

**Response 200:**
```json
{
  "deleted": true,
  "id": "uuid"
}
```

**Error codes:** 404, 403 (AUTHZ_DENIED), 409 (edge still has active fund channels or pending allocations)

---

### 2.7 List Party Edges

```http
GET /gov/party-edges?party_id={party_id}&edge_type={type}
```
**ABAC action:** `data.party.read`

**Query Parameters:**

| Param | Type | Default | Description |
|---|---|---|---|
| party_id | string (UUID) | (required) | Filter edges where this party is src OR dst |
| edge_type | string | all | Filter by edge type |
| direction | string (enum: `outgoing`, `incoming`) | all | `outgoing` = party is src; `incoming` = party is dst |
| page | integer | 1 | |
| page_size | integer | 20 | |

**Response 200:** Paginated list of edge objects.

---

### 2.8 Create Party Member

```http
POST /gov/party-members
```
**ABAC action:** `iam.member.create`

**Request Body:**
```json
{
  "party_id": "uuid-party",
  "user_id": "uuid-user",
  "role": "member"
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| party_id | string (UUID) | yes | |
| user_id | string (UUID) | yes | |
| role | string (enum: `leader`, `member`, `observer`) | no (default `member`) | `leader` role is descriptive only -- no automatic privileges (A-CON-05) |
| is_primary | boolean | no (default `false`) | Mark as primary membership for this user |

**ABAC validation:** Caller must have `iam.member.create` for the target `party_id` scope.

**Response 201:**
```json
{
  "id": "uuid",
  "party_id": "uuid-party",
  "user_id": "uuid-user",
  "user_name": "Zhang San",
  "role": "member",
  "is_primary": false,
  "joined_at": "2026-07-31T10:00:00Z",
  "created_at": "2026-07-31T10:00:00Z"
}
```

**Error codes:** 403 (AUTHZ_DENIED), 409 (duplicate member), 400 (validation)

---

### 2.9 Remove Party Member

```http
DELETE /gov/party-members/{member_id}
```
**ABAC action:** `iam.member.delete`

**Path Parameters:**

| Param | Type | Description |
|---|---|---|
| member_id | string (UUID) | |

**Response 200:**
```json
{
  "deleted": true,
  "id": "uuid"
}
```

**Error codes:** 404, 403 (AUTHZ_DENIED), 409 (member has active API keys bound to this party's account -- must revoke first)

---

### 2.10 List Party Members

```http
GET /gov/party-members?party_id={party_id}
```
**ABAC action:** `data.member.read`

**Query Parameters:**

| Param | Type | Default | Description |
|---|---|---|---|
| party_id | string (UUID) | (required) | |
| role | string (enum) | all | Filter by role |
| page | integer | 1 | |
| page_size | integer | 20 | |

**Response 200:** Paginated list of party_member objects with `user_name` included.

---

## 3. Fund (资金治理)

> **PRD references:** FUN-03~10, UI-03, §5 (Budget Cap), §8 (Boundary Rules)  
> **Tables:** `accounts`, `ledgers`, `freezes`, `allocations`, `liquidations`, `idempotency_records`  
> **ALL fund write endpoints require `Idempotency-Key` header.**  
> **Fund ABAC axis is independent from data/iam/routing (A-CON-01).**

### 3.1 Allocate (Fund Transfer)

```http
POST /gov/accounts/{src_account_id}/allocate
```
**ABAC action:** `fund.allocate`  
**Idempotency-Key:** REQUIRED  
**Idempotency scope:** `allocate`

**Path Parameters:**

| Param | Type | Description |
|---|---|---|
| src_account_id | string (UUID) | Source account |

**Request Body:**
```json
{
  "dst_account_id": "uuid-dst-account",
  "amount": 80000.00,
  "edge_id": "uuid-party-edge",
  "reason": "Monthly Q3 allocation for AI R&D"
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| dst_account_id | string (UUID) | yes | Destination account |
| amount | decimal (>0) | yes | Transfer amount. Must be > 0. |
| edge_id | string (UUID) | no | Reference to a party_edge that authorizes this channel. If omitted, caller must have a whitelist grant (PRD §8.2). |
| reason | string (0-512) | no | Business justification |

**Validation rules:**
- Source account must be `active` (not `frozen`, `liquidating`, or `closed`).
- Destination account must be `active`.
- Channel must be permitted: `parent` (src parent of dst), `sponsors` (src sponsors dst), `allocates` (src party allocates to dst person), or whitelist (PRD §8.2).
- Source `available_balance >= amount`.
- Conservation (F-CON-02): src debit = dst credit, same amount, same transaction.

**Response 200:**
```json
{
  "allocation_id": "uuid",
  "src_account_id": "uuid-src-account",
  "dst_account_id": "uuid-dst-account",
  "amount": 80000.00,
  "channel": "parent",
  "edge_id": "uuid-party-edge",
  "status": "completed",
  "src_balance_after": 420000.00,
  "dst_balance_after": 80000.00,
  "idempotency_key": "uuid-v4",
  "created_at": "2026-07-31T10:00:00Z",
  "completed_at": "2026-07-31T10:00:00Z"
}
```

**Error codes:**
- 403 (AUTHZ_DENIED) -- no fund.allocate permission for src account scope
- 403 (ACCOUNT_FROZEN_OR_CLOSED) -- src or dst account not active
- 402 (INSUFFICIENT_BALANCE) -- src available_balance < amount
- 400 (validation) -- invalid channel, self-transfer, amount <= 0
- 409 (IDEMPOTENCY_CONFLICT) -- same key, different request body
- 200 (IDEMPOTENCY_REPLAY) -- same key, same request body, returns stored result

---

### 3.2 Start Liquidation

```http
POST /gov/accounts/{account_id}/liquidate
```
**ABAC action:** `fund.liquidate`  
**Idempotency-Key:** REQUIRED  
**Idempotency scope:** `liquidate`

**Path Parameters:**

| Param | Type | Description |
|---|---|---|
| account_id | string (UUID) | Account to liquidate |

**Request Body:**
```json
{
  "target_account_id": "uuid-target-account",
  "reason": "Project closure -- remaining funds return to sponsor"
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| target_account_id | string (UUID) | yes | Account to receive remaining funds after drain |
| reason | string (0-512) | yes | Justification (audited) |

**Effect:** Initiates the liquidation state machine (PRD §8.4):
1. `liquidating_block_new` -- Reject new calls and new freezes immediately.
2. `liquidating_drain` -- Wait for existing freezes to expire/settle.
3. `liquidating_transfer` -- Transfer remaining balance to target.
4. `liquidated` -- Revoke all Keys bound to this account; party becomes read-only.

**Response 200:**
```json
{
  "liquidation_id": "uuid",
  "account_id": "uuid-account",
  "party_id": "uuid-party",
  "target_account_id": "uuid-target-account",
  "status": "liquidating_block_new",
  "initiated_by": "uuid-actor",
  "reason": "Project closure",
  "initiated_at": "2026-07-31T10:00:00Z"
}
```

**Error codes:**
- 403 (AUTHZ_DENIED) -- no fund.liquidate for account scope
- 403 (ACCOUNT_FROZEN_OR_CLOSED) -- account already in liquidation or closed
- 400 (validation) -- target same as source, target not active
- 409 (IDEMPOTENCY_CONFLICT)

---

### 3.3 Get Liquidation Status

```http
GET /gov/accounts/{account_id}/liquidation
```
**ABAC action:** `fund.ledger.read`

**Response 200:**
```json
{
  "liquidation_id": "uuid",
  "account_id": "uuid-account",
  "party_id": "uuid-party",
  "target_account_id": "uuid-target-account",
  "status": "liquidating_drain",
  "stage_descriptions": {
    "liquidating_block_new": "completed",
    "liquidating_drain": "in_progress",
    "liquidating_transfer": "pending",
    "liquidated": "pending"
  },
  "pending_freeze_count": 3,
  "pending_freeze_total": 450.00,
  "available_balance": 78250.00,
  "initiated_at": "2026-07-31T10:00:00Z"
}
```

---

### 3.4 Advance Liquidation Stage

```http
POST /gov/accounts/{account_id}/liquidate/advance
```
**ABAC action:** `fund.liquidate`

Advances the state machine one step (e.g., `drain` -> `transfer` when all freezes cleared, or `transfer` -> `liquidated` after balance moved). The state machine enforces valid transitions (PRD §8.4).

**Request Body:** (empty)

**Response 200:** Updated liquidation object.

**Error codes:** 403, 409 (not ready for transition -- freezes still pending, or balance not zero), 400 (already in terminal state)

---

### 3.5 Set / Update Budget Cap

```http
PATCH /gov/accounts/{account_id}/budget
```
**ABAC action:** `fund.budget.write`

**Path Parameters:**

| Param | Type | Description |
|---|---|---|
| account_id | string (UUID) | |

**Request Body:**
```json
{
  "budget_limit_amount": 80000.00,
  "budget_warn_ratio": 0.80,
  "budget_period": "calendar_month",
  "budget_period_start": null,
  "budget_period_end": null
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| budget_limit_amount | decimal (>= 0) or null | no | Set to null to disable budget cap |
| budget_warn_ratio | decimal (0.0-1.0) | no | Warn ratio (e.g., 0.80 = warn at 80%). Warns only, does not block. |
| budget_period | string (enum: `none`, `calendar_month`, `calendar_day`, `custom`) | no | Budget reset period |
| budget_period_start | string (ISO 8601) | no | Required if period = `custom` |
| budget_period_end | string (ISO 8601) | no | Required if period = `custom` |

**This endpoint triggers a key configuration change audit (AU-CON-03): before/after snapshot with actor, time, and both values recorded in `audit_events`.**

**Response 200:**
```json
{
  "account_id": "uuid",
  "budget_limit_amount": 80000.00,
  "budget_warn_ratio": 0.80,
  "budget_period": "calendar_month",
  "budget_consumed_amount": 0.00,
  "budget_version": 1,
  "updated_at": "2026-07-31T10:00:00Z"
}
```

**Error codes:** 403 (AUTHZ_DENIED), 400 (validation -- warn_ratio > 1.0, custom period without start/end), 409 (budget_version mismatch -- concurrent modification)

---

### 3.6 Get Account

```http
GET /gov/accounts/{account_id}
```
**ABAC action:** `fund.balance.read`

**Response 200:**
```json
{
  "id": "uuid",
  "party_id": "uuid-party",
  "party_name": "AI R&D Department",
  "available_balance": 500000.00,
  "frozen_balance": 15000.00,
  "status": "active",
  "budget": {
    "limit_amount": 80000.00,
    "warn_ratio": 0.80,
    "period": "calendar_month",
    "consumed_amount": 72000.00,
    "consumption_pct": 90.0,
    "warn_active": true
  },
  "liquidation": null,
  "version": 42,
  "created_at": "2026-07-01T00:00:00Z",
  "updated_at": "2026-07-31T10:00:00Z"
}
```

---

### 3.7 List Accounts

```http
GET /gov/accounts
```
**ABAC action:** `fund.balance.read` (ABAC applies fund-axis scope per D-CON-01)

**Query Parameters:**

| Param | Type | Default | Description |
|---|---|---|---|
| party_id | string (UUID) | all | Filter by owning party |
| status | string (enum: `active`, `frozen`, `liquidating`, `closed`) | `active` | |
| budget_exceeded | boolean | | Filter for accounts currently over budget cap |
| budget_warned | boolean | | Filter for accounts currently in warning zone |
| search | string | | Search by party name |
| page | integer | 1 | |
| page_size | integer | 20 | |

**Response 200:** Paginated list of account summaries.

**Error codes:** 403 (AUTHZ_DENIED)

---

### 3.8 Get Ledger (Transaction History)

```http
GET /gov/accounts/{account_id}/ledgers
```
**ABAC action:** `fund.ledger.read`

**Query Parameters:**

| Param | Type | Default | Description |
|---|---|---|---|
| direction | string | all | `debit`, `credit`, `freeze`, `unfreeze`, `settle` |
| from | string (ISO 8601) | | Start of time range |
| to | string (ISO 8601) | | End of time range |
| freeze_id | string (UUID) | | Filter by freeze |
| request_id | string (UUID) | | Filter by call request |
| allocation_id | string (UUID) | | Filter by allocation |
| idempotency_key | string | | Filter by idempotency key |
| sort | string | `-created_at` | Sort order prefix `-` for descending |
| page | integer | 1 | |
| page_size | integer | 20 | Max 200 |

**Response 200:**
```json
{
  "data": [
    {
      "id": "uuid",
      "account_id": "uuid-account",
      "direction": "debit",
      "amount": 100.00,
      "balance_after": 499900.00,
      "frozen_after": 15000.00,
      "sell_amount": null,
      "freeze_id": null,
      "request_id": null,
      "allocation_id": "uuid-allocation",
      "user_id": "uuid-user",
      "idempotency_key": "uuid-v4",
      "reason": "Monthly allocation from corporate pool",
      "created_at": "2026-07-31T10:00:00Z"
    }
  ],
  "total": 1523,
  "page": 1,
  "page_size": 20,
  "pages": 77
}
```

**Error codes:** 403 (AUTHZ_DENIED), 404 (account not found)

---

### 3.9 Get Freezes

```http
GET /gov/accounts/{account_id}/freezes
```
**ABAC action:** `fund.ledger.read`

**Query Parameters:**

| Param | Type | Default | Description |
|---|---|---|---|
| status | string (enum: `active`, `settled`, `expired`, `released`) | all | |
| request_id | string (UUID) | | |
| page | integer | 1 | |
| page_size | integer | 20 | |

**Response 200:** Paginated list of freeze records.

---

### 3.10 List Allocations

```http
GET /gov/allocations
```
**ABAC action:** `fund.ledger.read`

**Query Parameters:**

| Param | Type | Default | Description |
|---|---|---|---|
| src_account_id | string (UUID) | | |
| dst_account_id | string (UUID) | | |
| channel | string | | `parent`, `sponsors`, `whitelist` |
| status | string | | `pending`, `completed`, `reverted` |
| from / to | string (ISO 8601) | | Time range |
| page | integer | 1 | |
| page_size | integer | 20 | |

**Response 200:** Paginated list of allocation records.

---

### 3.11 Get Allocation Detail

```http
GET /gov/allocations/{allocation_id}
```
**ABAC action:** `fund.ledger.read`

**Response 200:** Full allocation object with source and destination ledger entries.

---

## 4. Key & Member (密钥与成员)

> **PRD references:** KEY-01~06, UI-05, §2.5 (Person/Key/Account/Ledger)  
> **Tables:** `api_keys`, `party_members`

### 4.1 Create API Key

```http
POST /gov/keys
```
**ABAC action:** `iam.key.create`

**Request Body:**
```json
{
  "name": "Zhang San's Dev Key",
  "owner_user_id": "uuid-user",
  "account_id": "uuid-account",
  "party_id": "uuid-party",
  "ip_allowlist": ["10.0.0.0/8"],
  "limit_daily_tokens": 1000000,
  "limit_monthly_cost_usd": 2000.00,
  "limit_max_concurrency": 5,
  "expires_at": "2027-07-31T00:00:00Z"
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| name | string (1-128) | yes | Human-readable label |
| owner_user_id | string (UUID) | yes | Must be an active, non-disabled user (KEY-03) |
| account_id | string (UUID) | yes | Target account. Must be within caller's `iam` axis allowed set (KEY-04). |
| party_id | string (UUID) | no | Redundant affiliation for indexing |
| ip_allowlist | array of string | no | CIDR allowlist |
| limit_daily_tokens | integer | no | |
| limit_monthly_cost_usd | decimal | no | |
| limit_max_concurrency | integer | no | |
| expires_at | string (ISO 8601) | no | |

**ABAC validation:**
- Caller must have `iam.key.create` for the target account scope.
- Target account must be within caller's allowed account set.
- Cannot bind to an account the caller has no fund authority over.

**Response 201:**
```json
{
  "id": "uuid",
  "name": "Zhang San's Dev Key",
  "key_prefix": "sk-abc123",
  "key_suffix": "xyz789",
  "api_key": "sk-abc123...xyz789",
  "owner_user_id": "uuid-user",
  "account_id": "uuid-account",
  "party_id": "uuid-party",
  "status": "active",
  "issued_at": "2026-07-31T10:00:00Z",
  "expires_at": "2027-07-31T00:00:00Z"
}
```

**Security (D-CON-03):**
- The full API key is returned ONLY in this response. It is not stored in plaintext (only `key_hash`).
- Subsequent GETs return `key_prefix` and `key_suffix` only. The full key is never re-displayed.
- The full key is never written to application logs.

**Error codes:** 403 (AUTHZ_DENIED), 400 (user disabled, account not active), 409 (account_id not in iam scope)

---

### 4.2 List API Keys

```http
GET /gov/keys
```
**ABAC action:** `iam.key.read`

**Query Parameters:**

| Param | Type | Default | Description |
|---|---|---|---|
| owner_user_id | string (UUID) | | |
| account_id | string (UUID) | | |
| party_id | string (UUID) | | |
| status | string (enum: `active`, `revoked`, `expired`) | `active` | |
| search | string | | Search key name |
| page | integer | 1 | |
| page_size | integer | 20 | |

**Response 200:** Paginated list. Each key shows `key_prefix`, `key_suffix`, but NOT the full key (D-CON-03).

---

### 4.3 Get API Key

```http
GET /gov/keys/{key_id}
```
**ABAC action:** `iam.key.read`

**Response 200:** Full key object (prefix + suffix only, no plaintext key).

---

### 4.4 Revoke API Key

```http
DELETE /gov/keys/{key_id}
```
**ABAC action:** `iam.key.delete`

**Response 200:**
```json
{
  "deleted": true,
  "id": "uuid-key",
  "status": "revoked",
  "revoked_at": "2026-07-31T10:00:00Z"
}
```

**Effect:** Key immediately invalidated for all future requests. Any in-flight requests with existing freezes will still settle.

**Error codes:** 404, 403 (AUTHZ_DENIED), 409 (already revoked)

---

### 4.5 Rotate API Key

```http
POST /gov/keys/{key_id}/rotate
```
**ABAC action:** `iam.key.create`

**Response 201:** New key object. Old key enters grace period (`grace_until`), then auto-revoked.

---

### 4.6 Add Member to Party

```http
POST /gov/members
```
**ABAC action:** `iam.member.create`

(Same as §2.8 -- included here for completeness.)

---

## 5. Pricing (双轨计价)

> **PRD references:** PRI-01~04, UI-04, §4 (Dual-track Pricing)  
> **Tables:** `model_prices`

### 5.1 Create / Update Model Price

```http
PUT /gov/model-prices
```
**ABAC action:** `routing.price.write`

Creates a new price entry or updates an existing one. Uses `reference_id` as the upsert key.

**Request Body:**
```json
{
  "model_id": "gpt-4o",
  "channel_id": "ch-openai-primary",
  "reference_id": "gpt-4o-main-2026-q3",
  "price_json": {
    "items": [
      {
        "itemCode": "prompt_tokens",
        "cost": {"mode": "usage_per_unit", "rate": 0.002},
        "sell": {"mode": "usage_per_unit", "rate": 0.003}
      },
      {
        "itemCode": "completion_tokens",
        "cost": {"mode": "usage_per_unit", "rate": 0.008},
        "sell": {"mode": "usage_per_unit", "rate": 0.012}
      },
      {
        "itemCode": "prompt_cached_tokens",
        "cost": {"mode": "usage_per_unit", "rate": 0.0005},
        "sell": {"mode": "usage_per_unit", "rate": 0.00075, "cache_discount_ratio": 0.5}
      },
      {
        "itemCode": "amortization_fixed",
        "cost": {"mode": "amortization_fixed", "monthly_rate": 5000.00},
        "sell": {"mode": "amortization_fixed", "monthly_rate": 5000.00}
      }
    ],
    "schedule": {"timezone": "Asia/Shanghai", "overrides": []}
  },
  "effective_start_at": "2026-08-01T00:00:00+08:00",
  "effective_end_at": null
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| model_id | string | yes | Logical model name (e.g., `gpt-4o`) |
| channel_id | string | no (null = default price) | Channel/provider_resource id |
| reference_id | string | yes | Unique version identifier; used as upsert key |
| price_json | object | yes | PRD §4.4 structure. Contains `items[]` and `schedule`. |
| price_json.items[].itemCode | string (enum) | yes | One of the 10 baseline itemCodes (PRD §4.1) |
| price_json.items[].cost.mode | string (enum) | yes | `flat_fee`, `usage_per_unit`, `usage_tiered`, `usage_volume`, `amortization_fixed` |
| price_json.items[].cost.rate | decimal | per mode | |
| price_json.items[].sell.mode | string (enum) | yes | Same 5 modes |
| price_json.items[].sell.rate | decimal | per mode | |
| price_json.items[].sell.cache_discount_ratio | decimal (0.0-1.0) | no | Cache discount multiplier (PRD §4.3). Applied as `sell = cost * (1+markup) * cache_discount_ratio`. |
| effective_start_at | string (ISO 8601) | no | |
| effective_end_at | string (ISO 8601) | no | |

**This endpoint triggers a key configuration change audit (AU-CON-03).**

**Response 200 (updated) / 201 (created):**
```json
{
  "id": "uuid",
  "model_id": "gpt-4o",
  "channel_id": "ch-openai-primary",
  "reference_id": "gpt-4o-main-2026-q3",
  "price_json": { "...": "..." },
  "status": "active",
  "effective_start_at": "2026-08-01T00:00:00+08:00",
  "created_at": "2026-07-31T10:00:00Z",
  "updated_at": "2026-07-31T10:00:00Z"
}
```

**Error codes:** 403 (AUTHZ_DENIED), 400 (invalid itemCode, invalid mode, duplicate reference_id on create)

---

### 5.2 List Model Prices

```http
GET /gov/model-prices
```
**ABAC action:** `routing.price.read`

**Query Parameters:**

| Param | Type | Default | Description |
|---|---|---|---|
| model_id | string | all | |
| channel_id | string | all | |
| status | string (enum: `active`, `archived`) | `active` | |
| page | integer | 1 | |
| page_size | integer | 20 | |

**Response 200:** Paginated list.

---

### 5.3 Get Model Price

```http
GET /gov/model-prices/{price_id}
```
**ABAC action:** `routing.price.read`

**Response 200:** Full price object with `price_json`.

---

### 5.4 Archive Model Price

```http
DELETE /gov/model-prices/{price_id}
```
**ABAC action:** `routing.price.write`

Soft-deletes by setting `status = 'archived'`. Triggers audit (AU-CON-03).

**Response 200:**
```json
{
  "archived": true,
  "id": "uuid"
}
```

**Error codes:** 404, 403 (AUTHZ_DENIED), 409 (still referenced by active route profiles)

---

## 6. Model Grant (模型授权)

> **PRD references:** MODEL-01~06 (implicit), §7.3 (Model Access Governance), §5.3 (ModelGrant Level Budget)  
> **Tables:** `model_grants`  
> **Principle:** DENY overrides ALLOW (A-CON-04). Model scope is independent of fund scope (A-CON-04).

### 6.1 Create Model Grant

```http
POST /gov/model-grants
```
**ABAC action:** `routing.model_grant.write`

**Request Body:**
```json
{
  "principal_type": "party",
  "principal_id": "uuid-ai-rd-dept",
  "model_id": "gpt-4o",
  "model_tag": null,
  "effect": "allow",
  "priority": 10,
  "quota_limit": 50000.00,
  "conditions": {}
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| principal_type | string (enum: `party`, `person`, `key`, `role`) | yes | |
| principal_id | string (UUID) | yes | |
| model_id | string | no* | Specific model name. Required if `model_tag` not set. |
| model_tag | string | no* | Tag group. Required if `model_id` not set. |
| effect | string (enum: `allow`, `deny`) | yes | `deny` takes precedence over `allow` at any priority. |
| priority | integer | no (default 0) | For ALLOW-ALLOW conflict: higher priority wins. DENY always wins regardless. |
| quota_limit | decimal (>= 0) or null | no | Model-level budget cap (cumulative). Null = unlimited. Intersects with Account budget cap (PRD §5.2). |
| conditions | object | no | Future extension: time-of-day, IP range, etc. |

**Validation:**
- Cascade evaluation order: Key > Person > Party > global default (PRD §7.3).
- `quota_limit` is the cumulative spend limit under this grant. When `consumed_amount + estimated_spend > quota_limit`, the system returns `MODEL_BUDGET_EXCEEDED`.

**Response 201:**
```json
{
  "id": "uuid",
  "principal_type": "party",
  "principal_id": "uuid-ai-rd-dept",
  "model_id": "gpt-4o",
  "effect": "allow",
  "priority": 10,
  "quota_limit": 50000.00,
  "quota_consumed": 0.00,
  "created_at": "2026-07-31T10:00:00Z"
}
```

**Error codes:** 403 (AUTHZ_DENIED), 400 (neither model_id nor model_tag provided)

---

### 6.2 List Model Grants

```http
GET /gov/model-grants
```
**ABAC action:** `routing.model_grant.read`

**Query Parameters:**

| Param | Type | Default | Description |
|---|---|---|---|
| principal_type | string | all | |
| principal_id | string | all | |
| model_id | string | all | |
| effect | string (enum: `allow`, `deny`) | all | |
| page | integer | 1 | |
| page_size | integer | 20 | |

**Response 200:** Paginated list.

---

### 6.3 Get Model Grant

```http
GET /gov/model-grants/{grant_id}
```
**ABAC action:** `routing.model_grant.read`

**Response 200:** Full grant object.

---

### 6.4 Delete Model Grant

```http
DELETE /gov/model-grants/{grant_id}
```
**ABAC action:** `routing.model_grant.write`

Hard delete. Triggers audit.

**Response 200:**
```json
{
  "deleted": true,
  "id": "uuid"
}
```

---

## 7. Routing (路由调度)

> **PRD references:** RTE-01~06, UI-06, §3.3 (Pluggable Strategy Matrix), §8.1 (Price Constraint)  
> **Tables:** `route_profiles`, `model_routes`, `provider_resources`  
> **Key constraint (S-CON-02):** delta_cap default 0, hard cap 20%. Any delta change triggers key audit.

### 7.1 Create Route Profile

```http
POST /gov/route-profiles
```
**ABAC action:** `routing.route_profile.write`

**Request Body:**
```json
{
  "name": "ha-cost-standard",
  "description": "High-availability with cost optimization",
  "strategies_json": [
    {"code": "S-COMPLIANCE", "enabled": true,  "priority": 0,  "config": {}},
    {"code": "S-PRI",        "enabled": true,  "priority": 10, "config": {}},
    {"code": "S-HEALTH",     "enabled": true,  "priority": 20, "config": {}},
    {"code": "S-WEIGHT",     "enabled": true,  "priority": 30, "config": {}},
    {"code": "S-COST",       "enabled": true,  "priority": 40, "config": {}},
    {"code": "S-AFFINITY",   "enabled": true,  "priority": 50, "config": {}}
  ],
  "delta_cap": 0.10,
  "max_attempts": 3,
  "allow_fallback": true
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| name | string (1-128) | yes | Unique profile name |
| description | string | no | |
| strategies_json | array | yes | Ordered strategy chain. See strategy codes below. |
| strategies_json[].code | string (enum) | yes | Strategy code (see table) |
| strategies_json[].enabled | boolean | yes | |
| strategies_json[].priority | integer | yes | Execution order (ascending). S-COMPLIANCE must be first if enabled. |
| strategies_json[].config | object | no | Strategy-specific parameters |
| delta_cap | decimal (0.0-0.20) | no (default 0.0) | Price cap delta. **Hard max 20%** (S-CON-02). Default 0 = no deviation. |
| max_attempts | integer (1-10) | no (default 3) | |
| allow_fallback | boolean | no (default true) | |

**Available strategy codes (PRD §3.3):**

| Code | Name | Can Disable | Description |
|---|---|---|---|
| `S-COMPLIANCE` | Compliance | **No** (hard) | INTERNAL_ONLY filtering; cannot be disabled for restricted parties |
| `S-PRI` | Priority | Yes | Primary/backup grouping and ordering |
| `S-HEALTH` | Health & Circuit Breaker | Yes | Three-state: Closed/HalfOpen/Open |
| `S-WEIGHT` | Weight & Load | Yes | Weighted distribution based on historical load |
| `S-AFFINITY` | Session Affinity | Yes | Same session prefers same channel |
| `S-COST` | Cost-Aware | Yes | Bias toward lower cost within qualified set |
| `S-LATENCY` | Latency-Aware | Yes | TTFT or end-to-end latency |
| `S-ERROR` | Error Rate Awareness | Yes | Penalize recent failure rate |
| `S-RATE` | Rate Limit Awareness | Yes | Reduce RPM/TPM/429 probability |
| `S-TAG` | Business Tag | Yes | Route by request tag |
| `S-CLASSIFY` | Task Classification | Yes | Lightweight task complexity pre-classification (phase C) |
| `S-CACHE` | Cache Fallback | Yes | Last-resort degradation channel |

**Validation:**
- `delta_cap` must be <= 0.20. Attempting to save > 0.20 returns 400.
- Any delta change triggers key audit (AU-CON-03 / S-CON-03).
- `S-COMPLIANCE`, if enabled, must be priority 0 (first in chain).

**Response 201:**
```json
{
  "id": "uuid",
  "name": "ha-cost-standard",
  "strategies_json": [...],
  "delta_cap": 0.10,
  "max_attempts": 3,
  "status": "active",
  "created_at": "2026-07-31T10:00:00Z"
}
```

**Error codes:** 403 (AUTHZ_DENIED), 400 (delta_cap > 0.20, invalid strategy code, S-COMPLIANCE not first)

---

### 7.2 List Route Profiles

```http
GET /gov/route-profiles
```
**ABAC action:** `routing.route_profile.read`

**Query Parameters:**

| Param | Type | Default | Description |
|---|---|---|---|
| status | string (enum: `active`, `archived`) | `active` | |
| search | string | | Search name |
| page | integer | 1 | |
| page_size | integer | 20 | |

**Response 200:** Paginated list.

---

### 7.3 Get Route Profile

```http
GET /gov/route-profiles/{profile_id}
```
**ABAC action:** `routing.route_profile.read`

**Response 200:** Full profile object.

---

### 7.4 Update Route Profile

```http
PUT /gov/route-profiles/{profile_id}
```
**ABAC action:** `routing.route_profile.write`

Full replacement. Same validation as create. Delta change triggers audit.

**Response 200:** Updated profile object.

---

### 7.5 Delete Route Profile

```http
DELETE /gov/route-profiles/{profile_id}
```
**ABAC action:** `routing.route_profile.write`

Soft-deletes (status = `archived`). Triggers audit.

**Response 200:** `{"deleted": true, "id": "uuid"}`

**Error codes:** 409 (profile still referenced by active routes)

---

### 7.6 List Registered Strategies

```http
GET /gov/route-strategies
```
**ABAC action:** `routing.route_profile.read`

Returns metadata about all registered strategies (code, name, description, whether it can be disabled, config schema).

**Response 200:**
```json
{
  "data": [
    {
      "code": "S-COMPLIANCE",
      "name": "Compliance Network",
      "description": "Hard INTERNAL_ONLY filter; cannot be disabled for restricted parties",
      "can_disable": false,
      "config_schema": {}
    },
    {
      "code": "S-COST",
      "name": "Cost-Aware",
      "description": "Bias toward lower cost within qualified set",
      "can_disable": true,
      "config_schema": {}
    }
  ]
}
```

---

### 7.7 List / Manage Model Routes

```http
GET /gov/model-routes
```
**ABAC action:** `routing.route_profile.read`

Lists individual route entries (model-to-provider mapping with priority, weight, price_cap_delta, etc.).

**Query Parameters:** model_name, provider_id, route_profile_id, status

```http
PUT /gov/model-routes/{route_id}
```
**ABAC action:** `routing.route_profile.write`

Updates a route entry.

```http
DELETE /gov/model-routes/{route_id}
```
**ABAC action:** `routing.route_profile.write`

---

## 8. ABAC (策略引擎)

> **PRD references:** SEC-GOV-01~05, UI-12, §7.2 (ABAC Policy Engine), §7.2.4 (Four-Axis Orthogonal Auth)  
> **Tables:** `sys_action_catalogs`, `sys_roles`, `sys_role_permissions`, `sys_subject_role_bindings`, `sys_access_policies`, `sys_access_policy_bindings`  
> **Principle:** ABAC is the unified authorization engine. RBAC is a subset. DENY takes precedence. Default is deny (A-CON-02).

### 8.1 List Action Catalogs

```http
GET /gov/action-catalogs
```
**ABAC action:** `iam.role.read`

Returns all registered actions in the system, grouped by axis.

**Query Parameters:**

| Param | Type | Default | Description |
|---|---|---|---|
| axis | string (enum: `data`, `fund`, `iam`, `routing`) | all | |

**Response 200:**
```json
{
  "data": [
    {
      "id": "uuid",
      "action_code": "fund.allocate",
      "action_name": "Fund Allocate",
      "axis": "fund",
      "resource_type": "account",
      "description": "Transfer funds between accounts"
    }
  ]
}
```

---

### 8.2 Create Role

```http
POST /gov/roles
```
**ABAC action:** `iam.role.write`

**Request Body:**
```json
{
  "role_code": "finance_admin",
  "role_name": "Financial Administrator",
  "description": "Full fund management access",
  "is_system": false,
  "permissions": [
    "fund.balance.read",
    "fund.ledger.read",
    "fund.allocate",
    "fund.liquidate",
    "fund.budget.write"
  ]
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| role_code | string (1-64) | yes | Unique role identifier |
| role_name | string (1-128) | yes | Display name |
| description | string | no | |
| is_system | boolean | no (default false) | System roles cannot be deleted |
| permissions | array of string | yes | Action codes from sys_action_catalogs |

**Response 201:**
```json
{
  "id": "uuid",
  "role_code": "finance_admin",
  "role_name": "Financial Administrator",
  "description": "Full fund management access",
  "is_system": false,
  "permissions": ["fund.balance.read", "fund.ledger.read", "fund.allocate", "fund.liquidate", "fund.budget.write"],
  "created_at": "2026-07-31T10:00:00Z"
}
```

**Error codes:** 403 (AUTHZ_DENIED), 409 (duplicate role_code)

---

### 8.3 List Roles

```http
GET /gov/roles
```
**ABAC action:** `iam.role.read`

**Response 200:** List of all roles with permission counts.

---

### 8.4 Get Role

```http
GET /gov/roles/{role_id}
```
**ABAC action:** `iam.role.read`

**Response 200:** Full role with expanded permissions.

---

### 8.5 Update Role

```http
PUT /gov/roles/{role_id}
```
**ABAC action:** `iam.role.write`

Full replacement of role data and permissions. Triggers audit.

**Error codes:** 403 (AUTHZ_DENIED), 400 (system role immutable), 404

---

### 8.6 Delete Role

```http
DELETE /gov/roles/{role_id}
```
**ABAC action:** `iam.role.write`

**Error codes:** 403, 400 (system role cannot be deleted), 409 (role still has active bindings)

---

### 8.7 Create Policy

```http
POST /gov/policies
```
**ABAC action:** `iam.policy.write`

**Request Body:**
```json
{
  "policy_code": "sep-route-fund",
  "policy_name": "Route-Fund Separation",
  "description": "Subjects with routing write permission are denied all fund write actions",
  "effect": "deny",
  "priority": 100,
  "is_system": true,
  "conditions_json": {
    "operator": "AND",
    "rules": [
      {"axis": "routing", "action_pattern": "routing.*.write"},
      {"target_axis": "fund", "target_action_pattern": "fund.*"}
    ]
  }
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| policy_code | string (1-64) | yes | Unique policy identifier |
| policy_name | string (1-128) | yes | |
| description | string | no | |
| effect | string (enum: `allow`, `deny`) | yes | `deny` takes precedence regardless of priority |
| priority | integer | no (default 0) | Higher number = evaluated first among same-effect policies |
| is_system | boolean | no (default false) | System policies cannot be deleted or disabled |
| conditions_json | object | yes | ABAC conditions. Supports `operator` (`AND`/`OR`), `rules[]` with `axis`, `action_pattern`, `resource_type`, time range, IP range, etc. |

**Response 201:** Policy object.

**Built-in separation-of-duty policies (PRD §7.2.5):**

| Policy Code | Effect | Rule |
|---|---|---|
| `SEP-ROUTE-FUND` | deny | Any subject with routing axis write permission is denied all fund axis write actions. Cannot be disabled. |
| `AUDITOR-READONLY` | deny | Subjects with auditor role are denied all write actions across all axes. Cannot be disabled. |
| `NO-SELF-APPROVAL` | deny | The initiator of a fund allocation cannot also approve it. |

---

### 8.8 List Policies

```http
GET /gov/policies
```
**ABAC action:** `iam.policy.read`

**Query Parameters:**

| Param | Type | Default | Description |
|---|---|---|---|
| effect | string (enum: `allow`, `deny`) | all | |
| is_system | boolean | all | |
| page | integer | 1 | |
| page_size | integer | 20 | |

---

### 8.9 Get Policy

```http
GET /gov/policies/{policy_id}
```
**ABAC action:** `iam.policy.read`

---

### 8.10 Update Policy

```http
PUT /gov/policies/{policy_id}
```
**ABAC action:** `iam.policy.write`

Triggers audit. System policies cannot be modified (400).

---

### 8.11 Delete Policy

```http
DELETE /gov/policies/{policy_id}
```
**ABAC action:** `iam.policy.write`

System policies cannot be deleted (400).

---

### 8.12 Simulate Policy Evaluation

```http
POST /gov/policies/{policy_id}/evaluate
```
**ABAC action:** `iam.policy.read`

Dry-run: evaluate a policy against a hypothetical subject, resource, and action without making changes.

**Request Body:**
```json
{
  "subject_user_id": "uuid-test-user",
  "resource_type": "account",
  "resource_id": "uuid-test-account",
  "action": "fund.allocate"
}
```

**Response 200:**
```json
{
  "result": "deny",
  "matched_policy_ids": ["uuid-sep-route-fund"],
  "evaluation_details": [
    {
      "policy_id": "uuid-sep-route-fund",
      "policy_code": "SEP-ROUTE-FUND",
      "effect": "deny",
      "matched": true,
      "reason": "Subject has routing.write permission; fund.write is denied by separation-of-duty policy"
    }
  ],
  "evaluated_at": "2026-07-31T10:00:00Z"
}
```

---

### 8.13 Create Subject-Role Binding

```http
POST /gov/subject-role-bindings
```
**ABAC action:** `iam.role.write`

**Request Body:**
```json
{
  "subject_type": "user",
  "subject_id": "uuid-user",
  "role_id": "uuid-role",
  "scope_party_id": "uuid-ai-rd-dept",
  "valid_from": "2026-08-01T00:00:00Z",
  "valid_until": null
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| subject_type | string (enum: `user`, `party`) | yes | |
| subject_id | string (UUID) | yes | |
| role_id | string (UUID) | yes | |
| scope_party_id | string (UUID) | no | NULL = global scope. When set, the role's permissions are scoped to this party. |
| valid_from | string (ISO 8601) | no | |
| valid_until | string (ISO 8601) | no | |

**Principle (A-CON-05):** Leader designation on a party does NOT automatically create a role binding. All permissions must be explicitly bound.

**Response 201:**
```json
{
  "id": "uuid",
  "subject_type": "user",
  "subject_id": "uuid-user",
  "role_id": "uuid-role",
  "role_code": "finance_admin",
  "scope_party_id": "uuid-ai-rd-dept",
  "valid_from": "2026-08-01T00:00:00Z",
  "valid_until": null,
  "created_at": "2026-07-31T10:00:00Z"
}
```

---

### 8.14 List Subject-Role Bindings

```http
GET /gov/subject-role-bindings
```
**ABAC action:** `iam.role.read`

**Query Parameters:**

| Param | Type | Default | Description |
|---|---|---|---|
| subject_type | string | all | |
| subject_id | string | all | |
| role_id | string | all | |
| scope_party_id | string | all | |
| page | integer | 1 | |
| page_size | integer | 20 | |

---

### 8.15 Delete Subject-Role Binding

```http
DELETE /gov/subject-role-bindings/{binding_id}
```
**ABAC action:** `iam.role.write`

---

### 8.16 Create Direct Grant

```http
POST /gov/grants
```
**ABAC action:** `iam.policy.write`

Direct grant (ABAC supplement, PRD §10.1 group 8). Used for one-off "User X can do Y on Resource Z" without creating a full role or policy.

**Request Body:**
```json
{
  "principal_type": "user",
  "principal_id": "uuid-user",
  "axis": "fund",
  "action": "fund.balance.read",
  "resource_type": "account",
  "resource_id": "uuid-account",
  "effect": "allow",
  "conditions": {}
}
```

**ABAC evaluation order:** Policy-based evaluation takes precedence over direct grants.

**Response 201:** Grant object.

---

### 8.17 List / Delete Grants

```http
GET /gov/grants?principal_type={type}&principal_id={id}
```
**ABAC action:** `iam.policy.read`

```http
DELETE /gov/grants/{grant_id}
```
**ABAC action:** `iam.policy.write`

---

## 9. UI Permission (UI权限治理)

> **PRD references:** SEC-GOV-06~08, UI-13, §7.4 (UI Permission Governance)  
> **Tables:** `sys_ui_menus`, `sys_ui_routes`, `sys_ui_action_bindings`  
> **Principle:** UI permission is the projection of ABAC on the presentation layer. The backend ABAC engine is the source of truth.

### 9.1 UI Menus CRUD

```http
GET    /gov/ui-menus
POST   /gov/ui-menus
GET    /gov/ui-menus/{menu_id}
PUT    /gov/ui-menus/{menu_id}
DELETE /gov/ui-menus/{menu_id}
```
**ABAC action (read):** `iam.ui.read`  
**ABAC action (write):** `iam.ui.write`

**Menu Object:**
```json
{
  "id": "uuid",
  "menu_code": "menu-fund",
  "parent_id": "root",
  "label": "Fund Management",
  "label_zh": "资金管理",
  "icon": "wallet",
  "sort_order": 10,
  "required_action_id": null,
  "children": []
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| menu_code | string (1-64) | yes | Unique code |
| parent_id | string | no (null = root) | |
| label | string | yes | Display label (English fallback) |
| label_zh | string | no | Chinese label |
| icon | string | no | Icon identifier |
| sort_order | integer | no (default 0) | Display order |
| required_action_id | string (UUID) | no | Minimal ABAC action to see this menu. If null, menu is always visible. |

**Query parameters (GET list):** `?parent_id={id}&search={term}`

---

### 9.2 UI Routes CRUD

```http
GET    /gov/ui-routes
POST   /gov/ui-routes
GET    /gov/ui-routes/{route_id}
PUT    /gov/ui-routes/{route_id}
DELETE /gov/ui-routes/{route_id}
```
**ABAC action (read):** `iam.ui.read`  
**ABAC action (write):** `iam.ui.write`

**Route Object:**
```json
{
  "id": "uuid",
  "route_path": "/console/fund/allocate",
  "menu_id": "uuid-menu-fund-allocate",
  "required_action_id": "uuid-action-fund-allocate",
  "label": "Fund Allocation",
  "label_zh": "资金划拨"
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| route_path | string (1-256) | yes | Frontend route path |
| menu_id | string (UUID) | yes | Association to menu |
| required_action_id | string (UUID) | yes | ABAC action required to access this route. Route guard enforces this. |
| label | string | yes | |
| label_zh | string | no | |

---

### 9.3 UI Action Bindings CRUD

```http
GET    /gov/ui-action-bindings
POST   /gov/ui-action-bindings
GET    /gov/ui-action-bindings/{binding_id}
PUT    /gov/ui-action-bindings/{binding_id}
DELETE /gov/ui-action-bindings/{binding_id}
```
**ABAC action (read):** `iam.ui.read`  
**ABAC action (write):** `iam.ui.write`

**Action Binding Object:**
```json
{
  "id": "uuid",
  "button_code": "btn-allocate-execute",
  "button_label": "Execute Allocation",
  "button_label_zh": "执行划拨",
  "page_route": "/console/fund/allocate",
  "required_action_id": "uuid-action-fund-allocate",
  "resource_type": "account"
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| button_code | string (1-128) | yes | Unique button identifier |
| button_label | string | yes | |
| button_label_zh | string | no | |
| page_route | string | yes | Which page this button appears on |
| required_action_id | string (UUID) | yes | ABAC action required to see/enable this button. Button hidden/disabled if user lacks this action. |
| resource_type | string | no | Hint for resource context |

**Query parameters (GET list):** `?page_route={path}&required_action_id={action_id}`

---

### 9.4 Generate UI Permission Snapshot

```http
GET /gov/ui-permissions/snapshot
```
**ABAC action:** (none -- derived from current session's ABAC evaluation)

Returns the user's rendered UI permission snapshot: which menus are visible, which routes are accessible, which buttons are enabled.

**Response 200:**
```json
{
  "user_id": "uuid",
  "roles": ["department_leader"],
  "permissions": ["fund.balance.read", "fund.ledger.read", "data.usage.read:self"],
  "menus": ["menu-fund", "menu-dashboard", "menu-keys"],
  "routes": ["/console/dashboard", "/console/fund/overview", "/console/keys"],
  "buttons": ["btn-key-create"],
  "generated_at": "2026-07-31T10:00:00Z"
}
```

This endpoint is typically consumed by the frontend on login and after role changes (PRD §7.4.2).

---

## 10. Audit (审计与对账)

> **PRD references:** AUD-01~04, UI-14, §7.6 (Immutable Audit)  
> **Tables:** `audit_events`, `audit_chain_anchors`, `request_logs`, `route_attempt_logs`, `usage_records`  
> **Principle:** audit_events are append-only; application layer rejects UPDATE/DELETE (AU-CON-02). Retention >= 180 days.

### 10.1 Search Audit Events

```http
GET /gov/audit-events
```
**ABAC action:** `data.audit.read`

**Query Parameters:**

| Param | Type | Default | Description |
|---|---|---|---|
| actor_user_id | string | all | |
| actor_name | string | all | Partial match |
| action | string | all | e.g., `fund.allocate`, `routing.price.write` |
| resource_type | string | all | e.g., `account`, `model_price`, `route_profile` |
| resource_id | string | all | |
| status | string | all | `success`, `failure` |
| from | string (ISO 8601) | | |
| to | string (ISO 8601) | | |
| has_snapshot | boolean | | Filter only events with before/after snapshots |
| page | integer | 1 | |
| page_size | integer | 20 | Max 200 |
| sort | string | `-created_at` | |

**Response 200:**
```json
{
  "data": [
    {
      "id": "uuid",
      "actor_user_id": "uuid-actor",
      "actor_name": "Zhang San",
      "action": "fund.allocate",
      "resource_type": "account",
      "resource_id": "uuid-account",
      "status": "success",
      "message": "Allocated 80000.00 from Corp Pool to AI R&D",
      "has_snapshot": false,
      "ip": "10.0.1.100",
      "created_at": "2026-07-31T10:00:00Z"
    }
  ],
  "total": 4521,
  "page": 1,
  "page_size": 20,
  "pages": 227
}
```

**Error codes:** 403 (AUTHZ_DENIED)

---

### 10.2 Get Audit Event Detail (with Snapshot Diff)

```http
GET /gov/audit-events/{event_id}
```
**ABAC action:** `data.audit.read`

**Response 200:**
```json
{
  "id": "uuid",
  "actor_user_id": "uuid",
  "actor_name": "Zhang San",
  "action": "routing.price.write",
  "resource_type": "model_price",
  "resource_id": "uuid-price",
  "status": "success",
  "message": "Updated gpt-4o pricing",
  "before_snapshot": {
    "sell_input_price_per_1m": 0.0025
  },
  "after_snapshot": {
    "sell_input_price_per_1m": 0.003
  },
  "diff": {
    "sell_input_price_per_1m": {"from": 0.0025, "to": 0.003}
  },
  "ip": "10.0.1.100",
  "user_agent": "Mozilla/5.0 ...",
  "created_at": "2026-07-31T10:00:00Z"
}
```

---

### 10.3 Get Request Log Trace

```http
GET /gov/request-logs/{request_id}/trace
```
**ABAC action:** `data.usage.read`

Full call trace for a single request (AU-CON-01). Includes all audit-relevant fields.

**Response 200:**
```json
{
  "request_id": "uuid",
  "person": {"id": "uuid", "name": "Zhang San"},
  "api_key_id": "key-hash",
  "account": {"id": "uuid", "name": "AI R&D Account"},
  "party": {"id": "uuid", "name": "AI R&D Department"},
  "model": {"logical": "gpt-4o", "actual": "gpt-4o-2024-08-06"},
  "channel": {"id": "uuid", "name": "OpenAI Primary"},
  "usage": {
    "prompt_tokens": 5000,
    "completion_tokens": 1500,
    "cached_tokens": 2000,
    "total_tokens": 8500
  },
  "cost": {"cost_amount": 0.022, "sell_amount": 0.033, "cost_items": "[...]"},
  "freeze": {"id": "uuid", "amount": 0.05, "settled_amount": 0.033},
  "route_result": {
    "profile_id": "uuid",
    "profile_name": "ha-cost-standard",
    "candidates_count": 5,
    "selected_channel": "OpenAI Primary",
    "strategy_chain": ["S-COMPLIANCE", "S-PRI", "S-HEALTH", "S-COST"],
    "attempts": [
      {"index": 1, "channel": "uuid", "invoked": true, "latency_ms": 850, "status": 200}
    ]
  },
  "security_result": {"passed": true, "checks": []},
  "latency_ms": 850,
  "http_status": 200,
  "error_code": null,
  "business_tags": {"user_id": "app-user-123", "task_id": "task-456"},
  "timestamp": "2026-07-31T10:00:00.123Z"
}
```

---

### 10.4 List Request Logs

```http
GET /gov/request-logs
```
**ABAC action:** `data.usage.read` (ABAC applies data-scope filter per D-CON-01)

**Query Parameters:**

| Param | Type | Default | Description |
|---|---|---|---|
| party_id | string | all | |
| account_id | string | all | |
| user_id | string | all | |
| api_key_id | string | all | |
| model_name | string | all | |
| error_code | string | all | |
| from / to | string (ISO 8601) | | |
| page | integer | 1 | |
| page_size | integer | 20 | Max 200 |

---

### 10.5 Audit Chain Anchors

```http
GET /gov/audit-chain-anchors
```
**ABAC action:** `data.audit.read`

List periodic hash anchors that chain audit events for tamper detection.

---

## 11. Dashboard & Reports (仪表盘与报表)

> **PRD references:** UI-07/10/11, §9.8 AUD-04, §9.9

### 11.1 Main Dashboard

```http
GET /gov/dashboard
```
**ABAC action:** `data.report.read`

Returns aggregated consumption, balance, budget, and block rate metrics. Data scope is constrained by the caller's ABAC data-axis scope.

**Query Parameters:**

| Param | Type | Default | Description |
|---|---|---|---|
| party_id | string | caller's scope | Specific party to query |
| period | string | `current_month` | `current_day`, `current_month`, `current_quarter`, `current_year`, `custom` |
| from / to | string (ISO 8601) | | Required if period = `custom` |

**Response 200:**
```json
{
  "period": {"from": "2026-07-01T00:00:00Z", "to": "2026-07-31T23:59:59Z"},
  "consumption": {
    "total_sell": 245000.00,
    "total_cost": 163333.33,
    "markup_pct": 50.0,
    "trend": [
      {"date": "2026-07-01", "sell": 7500.00},
      {"date": "2026-07-02", "sell": 8200.00}
    ]
  },
  "balance": {
    "total_available": 500000.00,
    "total_frozen": 15000.00,
    "total_budget_limit": 350000.00,
    "utilization_pct": 70.0
  },
  "budget_status": {
    "accounts_at_warning": 2,
    "accounts_exceeded": 0,
    "accounts_near_limit": 1
  },
  "block_rates": {
    "BUDGET_CAP_EXCEEDED": 12,
    "INSUFFICIENT_BALANCE": 3,
    "MODEL_ACCESS_DENIED": 45,
    "AUTHZ_DENIED": 0,
    "RATE_LIMITED": 8
  },
  "top_consumers": [
    {"party_id": "uuid", "party_name": "AI R&D", "sell": 82000.00, "pct": 33.5},
    {"party_id": "uuid", "party_name": "Product Team", "sell": 65000.00, "pct": 26.5}
  ],
  "generated_at": "2026-07-31T10:00:00Z"
}
```

---

### 11.2 Security Reports

```http
GET /gov/security-reports
```
**ABAC action:** `data.report.read`

**Query Parameters:** Same time range parameters as dashboard.

**Response 200:**
```json
{
  "egress_stats": {
    "external_calls": 12500,
    "internal_calls": 3400,
    "external_ratio": 0.786
  },
  "block_leaderboard": {
    "by_error_code": [
      {"code": "MODEL_ACCESS_DENIED", "count": 145},
      {"code": "BUDGET_CAP_EXCEEDED", "count": 42},
      {"code": "RATE_LIMITED", "count": 28},
      {"code": "CONTENT_BLOCKED", "count": 7}
    ],
    "by_party": [
      {"party_name": "External Contractor Group", "block_count": 67}
    ]
  },
  "abac_denies": {
    "total": 0,
    "by_axis": {"fund": 0, "data": 0, "iam": 0, "routing": 0},
    "by_policy": []
  },
  "generated_at": "2026-07-31T10:00:00Z"
}
```

---

### 11.3 Call Trace Visualization

```http
GET /gov/trace
```
**ABAC action:** `data.usage.read`

**Query Parameters:**

| Param | Type | Required | Description |
|---|---|---|---|
| request_id | string (UUID) | yes* | |
| task_id | string | yes* | Business tag task_id. At least one of request_id or task_id required. |
| user_id | string | no | Business tag user_id filter |

**Response 200:**
```json
{
  "trace_id": "uuid",
  "segments": [
    {
      "phase": "auth",
      "label": "Key Authentication",
      "latency_ms": 2,
      "result": "success",
      "detail": {"key_id": "hash", "user": "Zhang San", "account": "uuid"}
    },
    {
      "phase": "model_grant",
      "label": "Model Access Check",
      "latency_ms": 1,
      "result": "allow",
      "detail": {"model": "gpt-4o", "grant_id": "uuid"}
    },
    {
      "phase": "budget",
      "label": "Budget Cap Check",
      "latency_ms": 1,
      "result": "pass",
      "detail": {"account_budget_pct": 90.0, "model_quota_remaining": 42000.00}
    },
    {
      "phase": "freeze",
      "label": "Fund Freeze",
      "latency_ms": 3,
      "result": "frozen",
      "detail": {"amount": 0.05, "freeze_id": "uuid"}
    },
    {
      "phase": "routing",
      "label": "Route Selection",
      "latency_ms": 5,
      "result": "selected",
      "detail": {"profile": "ha-cost-standard", "selected_channel": "OpenAI Primary", "attempts": 1}
    },
    {
      "phase": "upstream",
      "label": "Upstream Call",
      "latency_ms": 820,
      "result": "success",
      "detail": {"status": 200, "upstream_request_id": "req-xxx"}
    },
    {
      "phase": "settlement",
      "label": "Settlement",
      "latency_ms": 4,
      "result": "completed",
      "detail": {"cost": 0.022, "sell": 0.033, "settled_amount": 0.033}
    }
  ],
  "total_latency_ms": 850,
  "visualization_url": "/console/trace/viz?request_id=uuid"
}
```

---

## 12. Appendix: Error Code Quick Reference

### 12.1 Fund & Quota

| Code | HTTP | Retryable | Description |
|---|---|---|---|
| `BUDGET_CAP_EXCEEDED` | 402 | No (until next period) | Account-level budget cap hit |
| `MODEL_BUDGET_EXCEEDED` | 402 | No (until grant quota increased) | ModelGrant-level quota_limit exceeded |
| `INSUFFICIENT_BALANCE` | 402 | No (until funds added) | Available balance < freeze amount |
| `ACCOUNT_FROZEN_OR_CLOSED` | 403 | No | Account status is frozen/liquidating/closed |
| `FREEZE_EXPIRED` | 409 | Yes (retry) | Freeze expired before settlement |
| `IDEMPOTENCY_CONFLICT` | 409 | No | Same idempotency key, different request body |
| `IDEMPOTENCY_REPLAY` | 200 | -- | Successful replay of idempotent request |

### 12.2 Auth & Identity

| Code | HTTP | Description |
|---|---|---|
| `AUTH_INVALID_KEY` | 401 | API key invalid or revoked |
| `AUTH_USER_DISABLED` | 403 | Key owner is disabled |
| `AUTH_KEY_NO_ACCOUNT` | 403 | Key not bound to an account |
| `AUTHZ_DENIED` | 403 | ABAC evaluation denied |

### 12.3 Model Access

| Code | HTTP | Description |
|---|---|---|
| `MODEL_ACCESS_DENIED` | 403 | No ALLOW grant or matched DENY grant for this model |

### 12.4 Routing

| Code | HTTP | Description |
|---|---|---|
| `NO_ROUTE_WITHIN_PRICE_CAP` | 422 | No candidate within price cap delta |
| `NO_ROUTE_AVAILABLE` | 503 | No healthy route available |
| `ROUTE_COMPLIANCE_BLOCKED` | 403 | All candidates filtered by compliance policy |

### 12.5 Security

| Code | HTTP | Description |
|---|---|---|
| `COMPLIANCE_NETWORK_BLOCKED` | 403 | Network policy blocked (e.g., INTERNAL_ONLY to external) |
| `CONTENT_BLOCKED` | 403 | Content safety filter triggered |
| `RATE_LIMITED` | 429 | Gateway or policy rate limit hit |

### 12.6 Upstream & System

| Code | HTTP | Description |
|---|---|---|
| `UPSTREAM_ERROR` | 502 | Upstream returned an error |
| `UPSTREAM_TIMEOUT` | 504 | Upstream timed out |
| `INTERNAL_ERROR` | 500 | Internal gateway error |

---

## 13. Appendix: ABAC Action Catalog (sys_action_catalogs)

### 13.1 `data` Axis (Data & Reporting)

| Action Code | Resource Type | Description |
|---|---|---|
| `data.party.read` | party | Read party information |
| `data.usage.read` | request_log | Read usage logs and traces |
| `data.usage.read:self` | request_log | Read only own usage |
| `data.report.read` | dashboard | View dashboards and reports |
| `data.member.read` | party_member | View party membership |
| `data.audit.read` | audit_event | View audit events |

### 13.2 `fund` Axis (Fund Governance)

| Action Code | Resource Type | Description |
|---|---|---|
| `fund.balance.read` | account | Read account balances |
| `fund.ledger.read` | ledger | Read transaction history |
| `fund.allocate` | account | Execute fund transfers |
| `fund.liquidate` | account | Manage liquidation |
| `fund.budget.write` | account | Set budget caps and warn ratios |

### 13.3 `iam` Axis (Identity & Access Management)

| Action Code | Resource Type | Description |
|---|---|---|
| `iam.party.create` | party | Create organizations/projects |
| `iam.party.write` | party | Update party attributes |
| `iam.member.create` | party_member | Add members to party |
| `iam.member.delete` | party_member | Remove members from party |
| `iam.key.create` | api_key | Create API keys |
| `iam.key.read` | api_key | View API keys |
| `iam.key.delete` | api_key | Revoke API keys |
| `iam.role.read` | role | View roles, actions, bindings |
| `iam.role.write` | role | Create/edit/delete roles and bindings |
| `iam.policy.read` | policy | View ABAC policies |
| `iam.policy.write` | policy | Create/edit/delete ABAC policies |
| `iam.ui.read` | ui_element | View UI permission config |
| `iam.ui.write` | ui_element | Edit UI permission config |

### 13.4 `routing` Axis (Routing & Model Governance)

| Action Code | Resource Type | Description |
|---|---|---|
| `routing.price.read` | model_price | View pricing configurations |
| `routing.price.write` | model_price | Create/edit/delete pricing |
| `routing.route_profile.read` | route_profile | View route profiles |
| `routing.route_profile.write` | route_profile | Create/edit/delete route profiles |
| `routing.model_grant.read` | model_grant | View model access grants |
| `routing.model_grant.write` | model_grant | Create/edit/delete model grants |
| `routing.upstream_secret.write` | provider | Manage upstream API keys (RES-03) |
| `routing.channel.write` | provider_resource | Manage provider channels |
| `routing.model_catalog.write` | model | Manage model catalog |

---

## 14. Appendix: Idempotency Summary

| Endpoint | Scope | Idempotency-Key Required |
|---|---|---|
| `POST /gov/accounts/{id}/allocate` | `allocate` | **YES** |
| `POST /gov/accounts/{id}/liquidate` | `liquidate` | **YES** |
| `POST /gov/accounts/{id}/liquidate/advance` | `liquidate` | **YES** |
| All other fund mutation endpoints | per scope | **YES** |
| `PUT /gov/model-prices` | N/A | No (idempotent via reference_id) |
| `POST /gov/parties` | N/A | No |
| `POST /gov/keys` | N/A | No |
| All GET / DELETE endpoints | N/A | No |

---

*Document version: 3.2.0 | Generated: 2026-07-31 | Based on PRD v3.2.0 + schema/ai-gov-fusion-minimal.sql*
