# AI-GOV Fusion Architecture v3.2.0

## 1. Overview

AI-GOV Fusion is a model-governance gateway built on a forked TokenHub base, augmented with AxonHub-inspired pricing, strategy-based routing, and a full financial control plane (accounts, ledgers, freezes, settlements). This document defines the component architecture, dependencies, and integration points for v3.2.0, aligned with PRD v3.2.0.

### 1.1 System Context Diagram

```
                          +---------------------------+
                          |      API Consumers        |
                          |  (SDK / CLI / Web App)    |
                          +------------+--------------+
                                       |
            Compatible API (6 protocols: OpenAI, Anthropic, Gemini, Codex, ...)
                                       |
        +==============================|==============================+
        |                    AI-GOV Fusion Gateway                     |
        |                                                              |
        |   +----------------------------------------------------+     |
        |   |                DATA PLANE (sync)                    |     |
        |   |  Key Auth → Security → ModelGrant → Price Filter   |     |
        |   |  → Budget Cap → Freeze → Strategy Route → Upstream |     |
        |   |  → Stream Renewal → Normalize → Dual-Settle → Audit|     |
        |   +----------------------------------------------------+     |
        |                                                              |
        |   +----------------------------------------------------+     |
        |   |              CONTROL PLANE (API)                    |     |
        |   |  ABAC Engine → UI Projection                        |     |
        |   |  Parties / Accounts / Ledgers / Freezes / Prices    |     |
        |   |  Grants / ModelGrants / Route Profiles / Audit      |     |
        |   +----------------------------------------------------+     |
        |                                                              |
        |   +----------------------------------------------------+     |
        |   |            OPERATIONS PLANE (async)                 |     |
        |   |  Freeze TTL Scanner / Reconciliation / Billing      |     |
        |   |  Provider Health Probe / Alerting / Backups         |     |
        |   +----------------------------------------------------+     |
        +==============================================================+
                                       |
                              +--------+--------+
                              |   Upstream LLM   |
                              |   Providers      |
                              +------------------+
```

### 1.2 Plane Definitions

| Plane | Nature | Key Characteristic | Examples |
|-------|--------|--------------------|----------|
| **Data Plane** | Synchronous, hot-path | Every millisecond counts; must never corrupt funds | Auth, routing, freeze, settle |
| **Control Plane** | Synchronous, cold-path | User/admin-facing CRUD; consistency over latency | Create party, set budget, configure route |
| **Operations Plane** | Asynchronous, background | Scheduled or event-driven; failure-tolerant with retry | TTL scan, reconciliation, billing generation |

---

## 2. Package Architecture

### 2.1 Package Inventory

| Package | Nature | Responsibility | File Location |
|---------|--------|----------------|---------------|
| `fund` | **NEW** | Accounts, ledgers, freezes, allocations, liquidations, budget caps | `internal/server/fund/` |
| `pricing` | **NEW** | ModelPrice JSON (items[] cost/sell + schedule), CalculateCost/Sell, dual-track normalization, price-cap filter (delta) | `internal/server/pricing/` |
| `idempotency` | **NEW** | Idempotency-Key atomic claim; at-most-once for all fund mutations | `internal/server/idempotency/` |
| `party` | **EXTEND** | Parties, party_edges (7 edge types: parent, sponsors, owns, participates, funds, delegates, reports_to), party_members | `internal/server/party/` |
| `abac` | **NEW** | Attribute-based access control engine: policy evaluation over subject/resource/action/environment attributes | `internal/server/abac/` |
| `authz` | **EXTEND** | Grants 4-axis direct authorization (data, fund, iam, routing); serves as supplement to ABAC | `internal/server/authz/` |
| `ui_permission` | **NEW** | sys_ui_menus, sys_ui_routes, sys_ui_action_bindings; projects ABAC decisions onto UI elements | `internal/server/ui_permission/` |
| `routing` | **EXTEND** | Strategy interface + route_profiles (12 strategies); provider selection pipeline | `internal/server/routing/` |
| `modelgrant` | **NEW** | ModelGrant ALLOW/DENY with priority; model-level quota_limit (consumed per-request) | `internal/server/modelgrant/` |
| `security` | **EXTEND** | Security hooks (SEC-05 content safety, egress policy, prompt firewall) | `internal/server/security/` |
| `audit` | **EXTEND** | audit_events (every mutation) + audit_chain_anchors (immutable hash chain) | `internal/server/audit/` |

### 2.2 Full Dependency Graph

```
LAYER 0 — Foundation (zero internal dependencies)
  fund          pricing       idempotency       party
  ├── nil       ├── nil       ├── nil           ├── nil
  └── pure      └── pure      └── pure          └── pure

LAYER 1 — Authorization & Permissions
  authz ──────────→ party
  modelgrant ─────→ party
  abac ───────────→ authz
  ui_permission ──→ abac

LAYER 2 — Middleware & Routing
  routing ────────→ pricing, modelgrant
  security ───────→ authz

LAYER 3 — Cross-cutting
  audit ──────────→ fund, pricing, authz, modelgrant, routing
  gateway pipeline → all (orchestration, not a package dependency)

PACKAGE DEPENDENCY DETAIL:

  fund/
    (none — pure domain logic; GORM models self-contained)

  pricing/
    (none — pure arithmetic; references model_prices table directly)

  idempotency/
    (none — standalone idempotency_records table)

  party/
    (none — pure organizational model)

  authz/
    └── party (resolves party-scoped resource IDs for grant evaluation)

  modelgrant/
    └── party (resolves principal_type=party to actual party records)

  abac/
    └── authz (delegates direct-grant evaluation; ABAC policies are the
        primary path, grants are the supplement)

  ui_permission/
    └── abac (projects ABAC policy outcomes onto UI visibility matrix)

  routing/
    ├── pricing (price-cap filter: removes candidates whose sell price
    │            exceeds internal anchor by more than delta)
    └── modelgrant (ModelGrant checker: removes DENY-blocked models
                     from candidate pool)

  security/
    └── authz (security hooks need to know which actor is calling,
               to enforce egress/content policies)

  audit/
    ├── fund       (records all balance mutations)
    ├── pricing    (records price calculation metadata)
    ├── authz      (records access decisions)
    ├── modelgrant (records model access denials)
    └── routing    (records route decisions and attempts)
```

### 2.3 Layer Rules

| Rule | Description |
|------|-------------|
| **Downward-only** | A package may only import packages in strictly lower-numbered layers. Layer 0 imports nothing from Layers 1-3. |
| **No cycles** | The graph is a DAG; cycle detection enforced by `go vet` and CI. |
| **Interface segregation** | Packages expose narrow interfaces (e.g., `Strategy` has only `Filter` and `Score`); callers depend on the interface, not the implementation. |
| **Pure core** | `fund`, `pricing`, `idempotency` are pure Go with no side effects; all I/O goes through injected repositories. |
| **GORM models live with their package** | Each package owns its GORM model structs; AutoMigrate is called per-package from `store.go`. |

---

## 3. Data Plane Pipeline

### 3.1 14-Step Pipeline Diagram

```
REQUEST
  │
  ▼
[1] COMPATIBLE API PARSE
    HTTP handler deserializes into internal request struct.
    Package: server/http.go (existing TokenHub)
    Input:  HTTP request body + headers
    Output: ChatCompletionRequest / ResponsesRequest / EmbeddingsRequest

  ▼
[2] KEY AUTHENTICATION
    API key lookup, hash verification, status check, expiry check.
    Package: server/store.go → StartCall() (existing TokenHub)
    Input:  Authorization header
    Output: Project + APIKey structs (hydrated)

  ▼
[3] SECURITY HOOKS (no-op → real)
    Content safety, prompt firewall, egress policy hooks.
    Package: security/
    Input:  Request payload + authenticated key
    Output: Pass/fail (HTTPError on block)
    Integration: Inserted between [2] and [4] in StartCall transaction

  ▼
[4] MODELGRANT CHECK (NEW)
    Evaluate ModelGrant ALLOW/DENY for (principal, model).
    Package: modelgrant/
    Input:  Principal (party/person/key/role) + model_key
    Output: Pass (allow) or MODEL_ACCESS_DENIED error
    Rule:   DENY always wins regardless of priority; no ALLOW = implicit DENY
    Integration: Inserted after [3] within StartCall transaction

  ▼
[5] PRICE ESTIMATION (NEW)
    Look up ModelPrice for (channel, model), compute estimated cost.
    Package: pricing/
    Input:  model + channel + request parameters (token estimate)
    Output: EstimatedCost { cost_amount, sell_amount, cost_items[] }

  ▼
[6] PRICE-CAP FILTER (NEW)
    Filter route candidates: remove any whose sell price exceeds
    internal anchor price by more than delta (default 0%, hard cap 20%).
    Package: pricing/
    Input:  []RouteCandidate + anchor_price + delta
    Output: []RouteCandidate (price-eligible subset)
    Error:   PRICE_CAP_EXCEEDED if all candidates filtered out

  ▼
[7] BUDGET CAP CHECK (NEW)
    Check account.budget_consumed_amount + estimate.cost_amount
    against account.budget_limit_amount.
    Package: fund/
    Input:  account_id + estimated_cost
    Output: Pass or BUDGET_CAP_EXCEEDED
    Note:   budget_warn_ratio triggers async alert, does not block

  ▼
[8] FREEZE (NEW)
    Atomically increment account.frozen, insert freezes row.
    Package: fund/
    Input:  account_id + freeze_amount (= max(estimated_cost * safety_margin, 1.2x))
    Output: freeze_id + updated account frozen balance
    Error:   INSUFFICIENT_BALANCE if available < freeze_amount
    TTL:     expires_at = now + configurable_duration (default 30s)

  ▼
[9] STRATEGY MATRIX ROUTING
    Pipeline: S-COMPLIANCE → ModelGrant(deny filter) → PriceCap →
              S-CLASSIFY → remaining strategies → pick top candidate.
    Package: routing/
    Strategies (12 total):
      S-PRIORITY     — static priority from route_profiles
      S-HEALTH       — health score from channel_probes
      S-WEIGHT       — weighted round-robin
      S-AFFINITY     — session affinity (sticky routing)
      S-COST         — lowest sell price
      S-LATENCY      — lowest observed latency
      S-ERROR        — lowest error rate
      S-THROUGHPUT   — highest throughput capacity
      S-COMPLIANCE   — hard filter: INTERNAL_ONLY, REGION_RESTRICTED
      S-CLASSIFY     — buckets candidates into tiers (gold/silver/bronze)
      S-FALLBACK     — ordered fallback chain
      S-RANDOM       — random selection (for A/B or chaos testing)
    Input:  []RouteCandidate (price-eligible) + CallContext
    Output: RouteSelection (single winner)

  ▼
[10] UPSTREAM ADAPTER CALL
     Protocol-specific adapter (OpenAI, Anthropic, Gemini, Codex).
     Package: server/provider_*.go (existing TokenHub)
     Input:  RouteSelection + request payload
     Output: Response body + Usage (prompt_tokens, completion_tokens, total_tokens)

  ▼
[11] STREAM FREEZE RENEWAL
     For streaming responses: extend freezes.expires_at periodically.
     Package: fund/ (renewal logic), server/http.go (integration)
     Trigger: Every N seconds during stream (configurable, default 15s)
     Error:   STREAM_FREEZE_EXPIRED if renewal fails → abort stream

  ▼
[12] USAGE NORMALIZATION
     Map provider-specific token counts to normalized itemCodes.
     Package: pricing/
     Input:  Provider Usage + model + channel
     Output: NormalizedUsage { items: [{itemCode, quantity}] }

  ▼
[13] DUAL-TRACK SETTLEMENT
     Calculate final cost_amount and sell_amount from normalized usage.
     Package: pricing/ (calculation), fund/ (ledger mutation)
     Input:  NormalizedUsage + ModelPrice + freeze_id
     Output: Ledger entries (allocate_out + settle) + updated account balance
     Steps:
       a. CalculateCost: sum over items (quantity * price.cost)
       b. CalculateSell: sum over items (quantity * price.sell)
       c. Apply cache_discount: if cache hit → 0% cost, still charge sell
       d. Apply amortization_fixed: spread one-time fees over estimated lifetime
       e. Unfreeze: decrement account.frozen, update freezes.status = 'settled'
       f. Debit: account.balance -= sell_amount,
                 account.budget_consumed_amount += cost_amount
       g. Write ledger entries (type: settle)
    Error:  SETTLEMENT_MISMATCH if calculated > frozen

  ▼
[14] AUDIT PERSISTENCE
     Write audit_event + update audit_chain_anchor (hash chain).
     Write request_log, usage_record, route_attempt_log.
     Package: audit/ + existing TokenHub logging
     Input:  Full call context + all intermediate decisions
     Output: Persisted immutable records

RESPONSE
```

### 3.2 Integration Point: StartCall Transaction (store.go:3062-3142)

The existing `StartCall` function in `store.go` is a single GORM transaction that performs:

```
Current StartCall:
  1. Lock api_key row (SELECT ... FOR UPDATE)
  2. Load + hydrate APIKey
  3. Load Model, check status
  4. Check AllowedModels whitelist
  5. Merge quota limits (key-level + policy)
  6. Load/init quota buckets (day, month)
  7. Acquire concurrency lease (if MaxConcurrency > 0)
  8. Check request/token/cost quota
  9. Check runtime budget
  10. Increment day + month request counters
  11. Return CallContext
```

**v3.2 Insertion Points** (steps injected between existing steps):

```
Modified StartCall (v3.2):
  1-6. [EXISTING] Same as above
  7.   [EXISTING] Acquire concurrency lease
  8.   [EXISTING] Check request/token/cost quota

  ★ NEW: 8a. SECURITY HOOKS
       Call security.Evaluate(ctx, key, model, requestPayload)
       → Returns error (block) or nil (pass)

  ★ NEW: 8b. MODELGRANT CHECK
       Call modelgrant.CheckAccess(ctx, principal, modelName)
       → Returns MODEL_ACCESS_DENIED if no ALLOW or explicit DENY

  ★ NEW: 8c. PRICE ESTIMATION (pre-flight)
       Call pricing.EstimateCost(ctx, model, channel, estimatedTokens)
       → Returns EstimatedCost

  ★ NEW: 8d. BUDGET CAP CHECK
       Call fund.CheckBudgetCap(ctx, accountID, estimatedCost.CostAmount)
       → Returns BUDGET_CAP_EXCEEDED if budget consumed + estimate > limit

  ★ NEW: 8e. FUND FREEZE
       Call fund.Freeze(ctx, accountID, estimatedCost.SellAmount * safetyMargin)
       → Returns freezeID; atomically updates account.frozen
       → Returns INSUFFICIENT_BALANCE if (balance - frozen) < freeze_amount

  9-11. [EXISTING] Increment counters, return CallContext
```

**Integration approach**: The new checks are called as function calls within the existing GORM transaction, using the same `tx *gorm.DB` for consistency. If any new check fails, the entire transaction rolls back. This ensures:
- No partial state (e.g., key quota consumed but ModelGrant denied)
- Atomicity with existing TokenHub logic
- Zero modification to TokenHub's own code paths (new checks are injected via function composition, not code modification of existing functions)

### 3.3 Error Code Mapping

| Pipeline Step | Error Condition | HTTP Status | Error Code |
|---------------|-----------------|-------------|------------|
| [2] Key Auth | Invalid/missing key | 401 | `invalid_api_key` |
| [2] Key Auth | Key disabled/expired | 403 | `api_key_disabled` / `api_key_expired` |
| [3] Security | Content safety violation | 403 | `content_safety_blocked` |
| [3] Security | Egress policy deny | 403 | `egress_policy_denied` |
| [4] ModelGrant | Model not allowed for principal | 403 | `model_access_denied` |
| [6] Price Cap | All candidates exceed price cap | 403 | `price_cap_exceeded` |
| [7] Budget Cap | Budget consumed > limit | 429 | `budget_cap_exceeded` |
| [8] Freeze | Insufficient available balance | 402 | `insufficient_balance` |
| [8] Freeze | Account not found/disabled | 403 | `account_not_found` |
| [9] Routing | No candidates after all filters | 503 | `no_available_route` |
| [11] Stream | Freeze renewal failed | 503 | `stream_freeze_expired` |
| [13] Settlement | Calculated > frozen | 500 | `settlement_mismatch` |

---

## 4. Control Plane Architecture

### 4.1 ABAC Engine as Unified Auth Backbone

The ABAC engine (`abac` package) provides fine-grained, policy-based access control across the entire control plane. It evaluates policies of the form:

```
Policy: {
  Subject:  { type, id, attributes... }
  Resource: { type, id, attributes... }
  Action:   "read" | "write" | "delete" | "admin"
  Environment: { time_of_day, ip_range, auth_method... }
  Effect:   "allow" | "deny"
}
```

**Data model:**

```
sys_access_policies         sys_access_policy_bindings
┌──────────────────────┐    ┌───────────────────────────┐
│ id                   │    │ id                        │
│ name                 │◄───│ policy_id (FK)            │
│ effect (allow/deny)  │    │ subject_type (role/user)  │
│ resource_type        │    │ subject_id                │
│ action               │    │ priority                  │
│ conditions (JSONB)   │    │ enabled                   │
│ priority             │    └───────────────────────────┘
│ enabled              │
└──────────────────────┘

sys_roles                    sys_subject_role_bindings
┌──────────────────────┐    ┌───────────────────────────┐
│ id                   │    │ id                        │
│ name                 │◄───│ role_id (FK)              │
│ description          │    │ subject_type (user)       │
│ parent_role_id       │    │ subject_id                │
└──────────────────────┘    │ scope_type (party/global) │
                            │ scope_id                  │
                            └───────────────────────────┘
```

**Evaluation flow:**

```
1. Resolve subject attributes:
   - Direct roles: sys_subject_role_bindings WHERE subject_id = actor.id
   - Inherited roles: recursive parent_role_id traversal
   - Session attributes: auth_method, ip_address, auth_time

2. Resolve resource attributes:
   - Resource type from request path
   - Resource ownership from party hierarchy
   - Resource status, sensitivity labels

3. Collect applicable policies:
   - sys_access_policy_bindings WHERE subject_type = 'role'
     AND subject_id IN (resolved_roles)
   - Plus: sys_access_policy_bindings WHERE subject_type = 'user'
     AND subject_id = actor.id
   - Ordered by priority DESC

4. Evaluate each policy:
   - Check resource_type match
   - Check action match
   - Evaluate conditions (JSONB logic)
   - First explicit DENY → deny
   - First explicit ALLOW with no prior DENY → allow

5. Fallback to grants (authz package):
   - If no ABAC policy matched, check grants table
   - grants table = direct assignments ("Zhang San can read account X")
   - ABAC = policy-based ("All department leaders can read their department accounts")
```

### 4.2 ABAC vs Grants Coexistence

| Dimension | ABAC (abac package) | Grants (authz package) |
|-----------|---------------------|------------------------|
| **Granularity** | Policy-based, attribute-driven | Direct assignment |
| **Scalability** | O(1) per new user (inherit via role) | O(n) per new user |
| **Use case** | "All department leaders can..." | "Zhang San specifically can..." |
| **Evaluation order** | First | Second (supplement) |
| **Storage** | `sys_access_policies` + `sys_access_policy_bindings` | `grants` table (4-axis) |
| **UI** | Admin configures policies | Admin assigns grants directly |

The 4-axis grant model (`data`, `fund`, `iam`, `routing`) in the `grants` table provides a simpler mental model for one-off assignments, while ABAC handles the systematic rules.

### 4.3 UI Permission Projection Model

The `ui_permission` package projects ABAC decisions onto frontend UI elements:

```
sys_ui_menus                 sys_ui_routes
┌──────────────────┐        ┌──────────────────────┐
│ id               │        │ id                   │
│ parent_id        │        │ path                 │
│ label            │        │ component            │
│ icon             │        │ required_action      │
│ sort_order       │        │ required_resource    │
│ required_action  │        │ enabled              │
│ enabled          │        └──────────────────────┘
└──────────────────┘

sys_ui_action_bindings
┌──────────────────────────────┐
│ id                           │
│ element_key (menu/button/id) │
│ action (read/write/delete)   │
│ resource_type                │
│ resource_id                  │
│ enabled                      │
└──────────────────────────────┘
```

**Projection flow:**

```
1. Load all ABAC-resolved permissions for actor
   → Set{allowed_actions, allowed_resources}

2. For each sys_ui_menus row:
   - If required_action in allowed_actions → visible
   - If required_resource in allowed_resources → visible
   - Else → hidden

3. For each sys_ui_action_bindings row:
   - If (action, resource) in allowed_set → actionable
   - Else → disabled (greyed out)
```

### 4.4 Audit Immutability Chain

```
audit_events                         audit_chain_anchors
┌────────────────────────┐          ┌──────────────────────┐
│ id                     │          │ id                   │
│ event_type             │   ┌─────►│ last_event_id        │
│ actor_type             │   │      │ last_event_hash      │
│ actor_id               │   │      │ chain_hash           │
│ resource_type          │   │      │ event_count          │
│ resource_id            │   │      │ anchored_at          │
│ action                 │   │      └──────────────────────┘
│ before_snapshot (JSONB)│   │
│ after_snapshot (JSONB) │   │
│ metadata (JSONB)       │   │
│ request_id             │   │
│ created_at             │   │
│ event_hash (SHA-256) ◄─┼───┘ (each event's hash is
└────────────────────────┘        chained into the anchor)
```

**Hash chain construction:**

```
event_hash(N) = SHA-256(
    event_hash(N-1) ||
    actor_id ||
    resource_type || resource_id ||
    action ||
    before_snapshot_hash ||
    after_snapshot_hash ||
    created_at
)

chain_anchor.chain_hash = SHA-256(
    previous_chain_hash ||
    last_event_hash ||
    event_count
)
```

Anchors are written periodically (every N events or every M seconds). The `chain_hash` in each anchor covers all events since the previous anchor, forming an append-only chain. Any tampering with an event changes its hash, which changes all subsequent hashes, which breaks the anchor chain.

---

## 5. Key Interfaces

### 5.1 Strategy (routing package)

```go
// RouteCandidate represents a provider+model combination that can serve
// a request. It carries identity, health, and cost metadata for scoring.
type RouteCandidate struct {
    RouteID       string
    ProviderID    string
    ProviderModel string
    ChannelID     string
    Priority      int
    HealthScore   float64
    CostInfo      CostInfo       // from pricing package
    Metadata      map[string]any // extensible
}

// CostInfo is the pricing snapshot attached to each candidate.
type CostInfo struct {
    SellAmount float64
    CostAmount float64
    Currency   string
}

// Strategy is the interface every routing strategy must implement.
// Strategies are composed in a pipeline: Filter narrows the pool,
// Score ranks the survivors.
type Strategy interface {
    // Name returns a stable identifier for observability and profile config.
    Name() string

    // Filter removes candidates that violate this strategy's constraints.
    // S-COMPLIANCE uses this to strip INTERNAL_ONLY models when the request
    // originates from outside the trusted network.
    Filter(ctx context.Context, candidates []RouteCandidate) []RouteCandidate

    // Score assigns a score to each candidate. Higher is better.
    // Strategies that only filter (not score) return the input unchanged.
    Score(ctx context.Context, candidates []RouteCandidate) []RouteCandidate
}

// StrategyPipeline composes multiple strategies in order.
type StrategyPipeline struct {
    strategies []Strategy
}

func (p *StrategyPipeline) Execute(ctx context.Context, candidates []RouteCandidate) []RouteCandidate {
    for _, s := range p.strategies {
        candidates = s.Filter(ctx, candidates)
    }
    for _, s := range p.strategies {
        candidates = s.Score(ctx, candidates)
    }
    // Sort by cumulative score, return top
    sort.Slice(candidates, func(i, j int) bool {
        return candidates[i].Metadata["score"].(float64) > candidates[j].Metadata["score"].(float64)
    })
    return candidates
}
```

**Strategy Pipeline Configuration (from route_profiles table):**

```sql
CREATE TABLE route_profiles (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    model_pattern   TEXT NOT NULL,  -- glob: "gpt-*", "claude-*", "*"
    strategies      JSONB NOT NULL, -- ["S-COMPLIANCE","S-COST","S-HEALTH"]
    fallback_chain  JSONB,          -- ["provider-a","provider-b"]
    enabled         BOOLEAN DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### 5.2 FundService (fund package)

```go
// FundService is the primary interface for all financial operations.
// Every mutation is idempotent (requires IdempotencyKey) and audit-logged.
type FundService interface {
    // Account management
    CreateAccount(ctx context.Context, partyID string, opts AccountOptions) (*Account, error)
    GetAccount(ctx context.Context, accountID string) (*Account, error)
    UpdateBudget(ctx context.Context, accountID string, budget BudgetConfig) (*Account, error)

    // Balance operations (control plane)
    Allocate(ctx context.Context, req AllocateRequest) (*AllocateResult, error)
    Liquidate(ctx context.Context, req LiquidateRequest) (*LiquidateResult, error)

    // Freeze lifecycle (data plane + operations plane)
    Freeze(ctx context.Context, req FreezeRequest) (*Freeze, error)
    RenewFreeze(ctx context.Context, freezeID string, newExpiresAt time.Time) error
    SettleFreeze(ctx context.Context, freezeID string, actualAmount float64) (*SettleResult, error)
    CancelFreeze(ctx context.Context, freezeID string) error
    ReleaseExpiredFreezes(ctx context.Context) (int, error) // TTL scanner

    // Budget enforcement
    CheckBudgetCap(ctx context.Context, accountID string, estimatedCost float64) error

    // Ledger query
    ListLedgers(ctx context.Context, accountID string, opts ListOptions) ([]LedgerEntry, error)
}

type Account struct {
    ID                   string
    PartyID              string
    Balance              float64 // available = balance - frozen
    Frozen               float64
    Version              int64   // optimistic locking
    BudgetLimitAmount    *float64
    BudgetWarnRatio      *float64
    BudgetPeriod         string
    BudgetConsumedAmount float64
    BudgetVersion        int64
    Status               string
}

type AllocateRequest struct {
    IdempotencyKey string
    FromAccountID  string
    ToAccountID    string
    Amount         float64
    Remark         string
    OperatorID     string
}

type FreezeRequest struct {
    IdempotencyKey string
    AccountID      string
    Amount         float64
    TTL            time.Duration
    RequestID      string
}

type SettleResult struct {
    FreezeID         string
    ActualAmount     float64
    CostAmount       float64
    SellAmount       float64
    BalanceAfter     float64
    FrozenAfter      float64
    LedgerEntryIDs   []string
}
```

### 5.3 ABACEngine (abac package)

```go
// ABACEngine evaluates attribute-based access control policies.
type ABACEngine interface {
    // Evaluate determines whether an action is permitted.
    // It checks ABAC policies first, then falls back to grants (via authz).
    Evaluate(ctx context.Context, req AccessRequest) (*AccessDecision, error)

    // GetPermissions returns the full permission set for a subject.
    // Used by ui_permission to project UI visibility.
    GetPermissions(ctx context.Context, subject Subject) (*PermissionSet, error)
}

type AccessRequest struct {
    Subject     Subject
    Resource    Resource
    Action      string
    Environment Environment
}

type Subject struct {
    Type       string // "user", "role"
    ID         string
    Attributes map[string]any
}

type Resource struct {
    Type       string // "party", "account", "model", "grant"
    ID         string
    Attributes map[string]any // owner_party_id, status, sensitivity...
}

type Environment struct {
    TimeOfDay  string // "business_hours", "after_hours"
    IPAddress  string
    AuthMethod string // "api_key", "oauth", "session"
}

type AccessDecision struct {
    Allowed      bool
    Reason       string
    MatchedRule  string   // which policy/rule granted or denied
    Obligations []string  // additional requirements (e.g., "MFA required")
}

type PermissionSet struct {
    AllowedActions  map[string][]string // resource_type → []action
    AllowedResources map[string][]string // action → []resource_id
}
```

### 5.4 UIProjector (ui_permission package)

```go
// UIProjector maps ABAC permission sets onto UI element visibility.
type UIProjector interface {
    // ProjectMenus returns the visible menu tree for a subject.
    ProjectMenus(ctx context.Context, subject Subject) (*MenuTree, error)

    // ProjectRoutes returns accessible routes with enabled/disabled status.
    ProjectRoutes(ctx context.Context, subject Subject) ([]RouteEntry, error)

    // ProjectActions determines which action buttons/elements are actionable.
    ProjectActions(ctx context.Context, subject Subject, resourceType string, resourceID string) ([]ActionEntry, error)
}

type MenuTree struct {
    Items []MenuItem
}

type MenuItem struct {
    ID       string
    Label    string
    Icon     string
    Path     string
    Visible  bool
    Children []MenuItem
}

type RouteEntry struct {
    Path      string
    Component string
    Accessible bool
}

type ActionEntry struct {
    Key        string
    Label      string
    Actionable bool
    Reason     string // why disabled, if applicable
}
```

### 5.5 ModelGrantChecker (modelgrant package)

```go
// ModelGrantChecker evaluates model access grants for a principal.
type ModelGrantChecker interface {
    // CheckAccess determines if a principal can access a specific model.
    // DENY always wins; no ALLOW means implicit DENY.
    CheckAccess(ctx context.Context, principal Principal, modelKey string) error

    // CheckQuotaLimit evaluates model-level quota_limit.
    // Returns remaining quota or error if exhausted.
    CheckQuotaLimit(ctx context.Context, principal Principal, modelKey string, requestTokens int64) (*QuotaRemaining, error)

    // ConsumeQuota decrements the model-level quota counter for a principal.
    ConsumeQuota(ctx context.Context, principal Principal, modelKey string, tokens int64) error

    // ListAccessibleModels returns all models the principal can access.
    ListAccessibleModels(ctx context.Context, principal Principal) ([]AccessibleModel, error)
}

type Principal struct {
    Type string // "party", "person", "key", "role"
    ID   string
}

type AccessibleModel struct {
    ModelKey      string
    Effect        string // "allow"
    QuotaLimit    *int64 // nil = unlimited
    QuotaConsumed int64
}

type QuotaRemaining struct {
    Limit     int64
    Consumed  int64
    Remaining int64
}
```

---

## 6. Database Schema Map

### 6.1 Table Inventory (40 tables, organized by domain)

| # | Table | Domain | Owning Package | Migration |
|---|-------|--------|---------------|-----------|
| **Identity & Auth (4)** | | | | |
| 1 | `users` | identity | server | ALTER (TokenHub admin_users) |
| 2 | `admin_sessions` | identity | server | EXISTING (TokenHub) |
| 3 | `api_keys` | identity | server | ALTER: +account_id, +owner_user_id |
| 4 | `teams` | identity | server | EXISTING (TokenHub) |
| **Party & Organization (3)** | | | | |
| 5 | `parties` | party | party | NEW |
| 6 | `party_edges` | party | party | NEW |
| 7 | `party_members` | party | party | NEW |
| **Fund & Ledger (5)** | | | | |
| 8 | `accounts` | fund | fund | NEW |
| 9 | `ledgers` | fund | fund | NEW |
| 10 | `freezes` | fund | fund | NEW |
| 11 | `allocations` | fund | fund | NEW |
| 12 | `liquidations` | fund | fund | NEW |
| **Pricing (1)** | | | | |
| 13 | `model_prices` | pricing | pricing | NEW |
| **Providers & Models (5)** | | | | |
| 14 | `providers` | provider | server | EXISTING (TokenHub) |
| 15 | `provider_resources` | provider | server | EXISTING (TokenHub) |
| 16 | `models` | model | server | EXISTING (TokenHub) |
| 17 | `provider_models` | model | server | EXISTING (TokenHub) |
| 18 | `provider_quota_status` | provider | server | NEW (AxonHub absorption) |
| **Routing (3)** | | | | |
| 19 | `model_routes` | routing | routing | EXISTING (TokenHub) |
| 20 | `route_profiles` | routing | routing | NEW |
| 21 | `channel_probes` | routing | server | NEW (AxonHub absorption) |
| **Authorization (4)** | | | | |
| 22 | `grants` | authz | authz | NEW |
| 23 | `model_grants` | modelgrant | modelgrant | NEW |
| 24 | `allocate_whitelist` | fund | fund | NEW |
| 25 | `key_limit_policies` | authz | server | NEW |
| **ABAC & UI Permission (6)** | | | | |
| 26 | `sys_roles` | abac | abac | NEW |
| 27 | `sys_subject_role_bindings` | abac | abac | NEW |
| 28 | `sys_access_policies` | abac | abac | NEW |
| 29 | `sys_access_policy_bindings` | abac | abac | NEW |
| 30 | `sys_ui_menus` | ui_permission | ui_permission | NEW |
| 31 | `sys_ui_routes` | ui_permission | ui_permission | NEW |
| 32 | `sys_ui_action_bindings` | ui_permission | ui_permission | NEW |
| **Request & Usage Logging (5)** | | | | |
| 33 | `request_logs` | logging | server | ALTER: +cost_amount, +sell_amount, +cost_items, +account_id, +freeze_id, +party_id |
| 34 | `request_payload_logs` | logging | server | EXISTING (TokenHub) |
| 35 | `route_attempt_logs` | logging | server | ALTER: +cost_info |
| 36 | `usage_records` | logging | server | ALTER: +cost_amount, +sell_amount, +cost_items |
| 37 | `quota_buckets` | logging | server | EXISTING (TokenHub) |
| **Idempotency (1)** | | | | |
| 38 | `idempotency_records` | idempotency | idempotency | NEW |
| **Audit (2)** | | | | |
| 39 | `audit_events` | audit | audit | NEW |
| 40 | `audit_chain_anchors` | audit | audit | NEW |

### 6.2 Table Ownership Rules

| Rule | Description |
|------|-------------|
| **Single owner** | Each table is owned by exactly one package. Only that package writes to the table. |
| **Read-only access** | Other packages may read through the owning package's exported query methods. |
| **Migration ownership** | The owning package registers its AutoMigrate call. `store.go` orchestrates the order. |
| **ALTER strategy** | Existing TokenHub tables receive ALTER COLUMN additions only; no column removal or rename. |

### 6.3 Migration Strategy

```go
// store.go — Migration orchestration (conceptual)
func (s *GormStore) autoMigrate() error {
    // Phase 0: TokenHub existing tables (no changes to their core schema)
    // Phase 1: Layer 0 packages (no dependencies)
    // Phase 2: Layer 1 packages (depend on Layer 0)
    // Phase 3: Layer 2-3 packages

    migrators := []struct{
        name string
        fn   func(*gorm.DB) error
    }{
        // Phase 0: Existing TokenHub tables
        {"tokenhub_core", s.migrateTokenHubCore},

        // Phase 1: Layer 0
        {"party",    party.Migrate},
        {"fund",     fund.Migrate},
        {"pricing",  pricing.Migrate},
        {"idempotency", idempotency.Migrate},

        // Phase 2: Layer 1
        {"authz",     authz.Migrate},
        {"modelgrant", modelgrant.Migrate},
        {"abac",      abac.Migrate},
        {"ui_permission", ui_permission.Migrate},

        // Phase 3: Layer 2-3
        {"routing",  routing.Migrate},
        {"audit",    audit.Migrate},
    }

    for _, m := range migrators {
        if err := m.fn(s.db); err != nil {
            return fmt.Errorf("migration %s: %w", m.name, err)
        }
    }
    return nil
}
```

GORM AutoMigrate handles:
- CREATE TABLE IF NOT EXISTS (for NEW tables)
- ALTER TABLE ADD COLUMN (for EXTEND tables)
- Index creation

Manual migration SQL files are maintained in `schema/migrations/` for:
- Complex column type changes
- Data backfill scripts
- Rollback scripts

---

## 7. Deployment Topology

### 7.1 MVP: Single-Instance SQLite

```
┌──────────────────────────────────────────┐
│              Single Process               │
│                                           │
│  ┌────────────────────────────────────┐  │
│  │         HTTP Server (:8080)        │  │
│  │  ┌──────────────────────────────┐  │  │
│  │  │   Data Plane Pipeline        │  │  │
│  │  │   Control Plane API          │  │  │
│  │  └──────────────────────────────┘  │  │
│  │  ┌──────────────────────────────┐  │  │
│  │  │   Background Workers         │  │  │
│  │  │   - Freeze TTL Scanner       │  │  │
│  │  │   - Provider Health Probe    │  │  │
│  │  │   - Audit Anchor Flush       │  │  │
│  │  └──────────────────────────────┘  │  │
│  └────────────────────────────────────┘  │
│                    │                      │
│  ┌────────────────────────────────────┐  │
│  │     SQLite (WAL mode)              │  │
│  │     ai-gov-fusion.db               │  │
│  └────────────────────────────────────┘  │
└──────────────────────────────────────────┘

Suitable for:
  - Development and local testing
  - Single-tenant deployments
  - Evaluation/PoC environments
  - < 100 concurrent requests

Configuration:
  database_url: sqlite://data/ai-gov-fusion.db?_journal_mode=WAL&_busy_timeout=5000
```

### 7.2 Production: Multi-Instance PostgreSQL

```
                    ┌──────────────┐
                    │  LB / Nginx  │
                    └──┬──┬──┬──┬──┘
                       │  │  │  │
          ┌────────────┼──┼──┼──┼────────────┐
          │            │  │  │  │            │
          ▼            ▼  ▼  ▼  ▼            ▼
    ┌──────────┐ ┌──────────┐ ┌──────────┐
    │ Instance │ │ Instance │ │ Instance │
    │    1     │ │    2     │ │    N     │
    │  :8080   │ │  :8080   │ │  :8080   │
    └────┬─────┘ └────┬─────┘ └────┬─────┘
         │            │            │
         └────────────┼────────────┘
                      │
         ┌────────────┴────────────┐
         │   PostgreSQL 16 (R/W)   │
         │   ┌──────────────────┐  │
         │   │  ai_governance   │  │
         │   │  schema          │  │
         │   └──────────────────┘  │
         └─────────────────────────┘
                      │
         ┌────────────┴────────────┐
         │   Redis (optional)      │
         │   - Session cache       │
         │   - Rate limit counter  │
         │   - Health probe cache  │
         │   - Pub/sub for config  │
         │     change broadcasts   │
         └─────────────────────────┘

Suitable for:
  - Multi-tenant production
  - > 100 concurrent requests
  - HA required

Configuration:
  database_url: postgres://user:pass@host:5432/ai_governance?sslmode=require
  redis_url:    redis://host:6379/0
```

### 7.3 Instance Responsibilities

| Component | SQLite MVP | PostgreSQL Production |
|-----------|------------|-----------------------|
| Data plane requests | In-process mutex | Row-level locks (SELECT FOR UPDATE) |
| Concurrency lease | In-memory (InFlightLease table) | Same table, PostgreSQL-backed |
| Cluster tasks | Mutex-based singleton | PostgreSQL advisory locks |
| Freeze TTL scanner | In-process goroutine, 10s ticker | Leader-elected via PostgreSQL lease |
| Provider health probe | In-process goroutine, 30s ticker | Leader-elected via PostgreSQL lease |
| Audit anchor flush | In-process, every 100 events or 60s | Same, per-instance anchored independently |
| Session cache | In-memory map | Redis (optional; falls back to PostgreSQL) |

### 7.4 Redis Usage (Optional, Production Only)

| Key Pattern | Purpose | TTL |
|-------------|---------|-----|
| `session:{token}` | Admin session cache | 24h |
| `ratelimit:{key_id}:{bucket}` | Rate limit sliding window | window size |
| `health:{provider_id}` | Provider health probe result | 30s |
| `config:version` | Config change notification pub/sub | persistent |
| `affinity:{hash}` | Session affinity routing cache | 5min |

---

## Appendix A: Refactoring Strategy (from PRD 11.3)

### A.1 New vs Extend

```
NEW packages (written from scratch, zero TokenHub code modification):
  fund/          accounts/ledgers/freezes/allocations/liquidations
  pricing/       model_prices, dual-track calculation, price-cap filter
  idempotency/   Idempotency-Key atomic claim middleware
  abac/          ABAC policy engine (sys_access_policies, sys_roles, etc.)
  ui_permission/ sys_ui_menus, sys_ui_routes, sys_ui_action_bindings
  modelgrant/    ModelGrant ALLOW/DENY + quota_limit

EXTEND packages (incrementally extract from TokenHub monoliths):
  party/         Add party_edges (7 types), party_members
  authz/         Add grants table 4-axis model
  routing/       Add Strategy interface, route_profiles, 12 strategies
  audit/         Add audit_events, audit_chain_anchors
  security/      Add security hooks (SEC-05), content safety, egress
```

### A.2 Extraction Rules

1. **NEW packages** are implemented independently; they do not modify any TokenHub source file.
2. **EXTEND packages** start by extracting existing logic from `store.go` (195KB) and `http.go` (295KB) into dedicated package files. The original code calls into the new package.
3. **Existing code stays functional**: the extracted code path is the same, just located in a dedicated package.
4. **Gate**: after each extraction, run TokenHub's existing test suite AND the new package's tests. Both must pass before the extraction is merged.

### A.3 Target File Size Reduction

| File | Current Size | Target After Extraction |
|------|-------------|------------------------|
| `store.go` | ~5,971 lines (195KB) | ~4,000 lines (core store only) |
| `http.go` | ~8,550 lines (295KB) | ~6,000 lines (HTTP handlers only) |

---

## Appendix B: Strategy Matrix Reference

### B.1 Strategy Categories

| Category | Strategies | Description |
|----------|-----------|-------------|
| **Hard Filter** | S-COMPLIANCE | Removes candidates that violate compliance rules (region, data residency, internal-only) |
| **Classifier** | S-CLASSIFY | Buckets candidates into tiers (gold/silver/bronze) based on SLA |
| **Priority** | S-PRIORITY | Static priority from route_profiles |
| **Health** | S-HEALTH | Health score from channel_probes (latency, error rate, availability) |
| **Load** | S-WEIGHT, S-THROUGHPUT | Weighted distribution, throughput-based selection |
| **Affinity** | S-AFFINITY | Session affinity (sticky routing) |
| **Cost** | S-COST | Lowest sell price wins |
| **Latency** | S-LATENCY | Lowest observed P50/P99 latency |
| **Reliability** | S-ERROR | Lowest error rate |
| **Utility** | S-FALLBACK, S-RANDOM | Ordered fallback chain, random selection |

### B.2 Pipeline Execution Order

```
Input: []RouteCandidate (from SelectRouteCandidates)

[1] S-COMPLIANCE.Filter()     ← hard filter: INTERNAL_ONLY, REGION_RESTRICTED
[2] ModelGrant deny check     ← removes DENY-blocked models
[3] PriceCap filter           ← removes candidates exceeding anchor + delta
    ⚠ If pool empty after [3] → PRICE_CAP_EXCEEDED
[4] S-CLASSIFY.Score()        ← assigns tier labels
[5] Remaining strategies      ← configured via route_profiles.strategies JSON
    (S-COST, S-HEALTH, S-WEIGHT, etc.)
[6] Sort by composite score   ← descending
[7] Pick top candidate        ← RouteSelection

Output: RouteSelection (single winner)
```

---

## Appendix C: Glossary

| Term | Definition |
|------|-----------|
| **ABAC** | Attribute-Based Access Control — policy evaluation based on subject/resource/action/environment attributes |
| **Anchor Price** | The internal reference price for a model; used as baseline for price-cap delta filtering |
| **Dual-Track** | Separate tracking of cost (what the platform pays upstream) and sell (what the customer is charged) |
| **Delta (delta)** | Maximum allowed deviation of sell price above anchor price; 0% = sell must equal anchor, 20% = hard cap |
| **Freeze** | Temporary hold on account balance during an in-flight request; prevents overspend during streaming |
| **Idempotency-Key** | Client-supplied key guaranteeing at-most-once execution of a mutation |
| **ModelGrant** | Per-principal ALLOW/DENY rule for model access, with optional quota_limit |
| **Party** | An organization, department, project, or team that owns accounts and consumes models |
| **Price-Cap Filter** | Pipeline step that removes route candidates whose sell price exceeds anchor + delta |
| **RouteCandidate** | A specific provider+model combination eligible to serve a request |
| **RouteProfile** | Named configuration binding a model pattern to a strategy pipeline |
| **S-{NAME}** | Strategy naming convention: S-COMPLIANCE, S-COST, S-HEALTH, etc. |
| **TTL Scanner** | Background job that releases expired freezes (status = 'timeout_released') |
