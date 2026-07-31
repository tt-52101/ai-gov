# PRD-1 财务演示 -- FIX-C 复验报告

**复验日期**: 2026-07-31
**复验范围**: batch-002 缺陷收口 -- FIX-C（Key 创建 / validateGovToken / 禁人链路）
**复验结论**: 全部通过（4/4 已修复）

---

## 检查清单

### 1. POST /gov/keys handler -- 是否真实实现？是否返回 raw_key 仅创建时？GET 是否不含明文？

**文件**: `ai-gov-fusion/backend/internal/server/gov_handlers_fund.go`

| 检查项 | 结果 | 证据 |
|--------|------|------|
| POST /gov/keys 有真实实现 | ✅已修复 | `handleKeys` (L66-74) 将 POST 分派到 `handleCreateKey`；`handleCreateKey` (L96-197) 包含 ABAC 鉴权、请求解析、Account 存在校验、密钥生成、SHA-256 哈希、DB 持久化的完整流程 |
| 创建时返回 raw_key | ✅已修复 | `handleCreateKey` 返回 `GovCreatedKeyResponse` (L192-196)，其中内嵌 `GovKeyResponse` + `RawKey` 字段，明文仅在创建响应中出现 |
| GET 不含明文 | ✅已修复 | `handleListKeys` (L199-238) 通过 `fromGovAPIKey` (L231-233) 转换为 `GovKeyResponse`，该结构体不含 `KeyHash`、不含 `RawKey`，仅含 `KeyPrefix` 用于识别 |

**判定**: ✅已修复

---

### 2. validateGovToken -- 是否替换占位实现？是否校验 Key 哈希 + 状态 + 用户禁用？

**文件**: `ai-gov-fusion/backend/internal/server/gov_handlers.go`

| 检查项 | 结果 | 证据 |
|--------|------|------|
| 非占位实现 | ✅已修复 | `validateGovToken` (L412-484) 为完整实现，非 stub / 硬编码返回 |
| 校验 Key SHA-256 哈希 | ✅已修复 | L426-427 对传入 Token 做 `sha256.Sum256`，L436 在 `gov_api_keys` 表中按 `key_hash` 精确匹配 |
| 校验 Key 状态 | ✅已修复 | L444-446 检查 `key.Status` 必须为 active（非空且非 active 即拒绝） |
| 校验 Key 未过期 | ✅已修复 | L449-451 检查 `ExpiresAt` 早于当前时间时拒绝 |
| 禁人即禁Key | ✅已修复 | L453-471 按 `OwnerUserID` 查询 `AdminUser`（表 admin_users），检查其 `Status` 字段 != active 则拒绝，并记录 WARN 日志 |

**判定**: ✅已修复

---

### 3. GovAPIKey 模型字段是否完整？

**文件**: `ai-gov-fusion/backend/internal/server/gov_models.go`

| 字段 | 存在 | 类型 | 用途 |
|------|------|------|------|
| ID | ✅ | string, PK | UUID 主键 (govkey_ 前缀) |
| Name | ✅ | string | 可读名称 |
| KeyHash | ✅ | string, uniqueIndex | SHA-256 哈希，`json:"-"` 不对外暴露 |
| KeyPrefix | ✅ | string | 前缀 8 字符，用于列表识别 |
| OwnerUserID | ✅ | string, index | 所有者用户，禁人即禁Key 的关联键 |
| AccountID | ✅ | string, index | 资金账户绑定，IAM 轴校验 |
| PartyID | ✅ | string, index | 组织/项目归属 |
| Status | ✅ | string | active / disabled / revoked |
| ExpiresAt | ✅ | *time.Time | 可选的过期时间 |
| CreatedAt | ✅ | time.Time | 创建时间 |
| LastUsedAt | ✅ | *time.Time | 最近使用时间 |

配套请求/响应类型（`GovCreateKeyRequest`, `GovKeyResponse`, `GovCreatedKeyResponse`）和转换函数 `fromGovAPIKey` 均完整定义。

**判定**: ✅已修复

---

### 4. APIKey 模型是否有 AccountID / PartyID？

**文件**: `ai-gov-fusion/backend/internal/server/types.go`

| 字段 | 存在 | 行号 | 类型 |
|------|------|------|------|
| AccountID | ✅ | L117 | `int64`, gorm index, `json:"account_id,omitempty"`, 注释: "绑定扣费账户" |
| PartyID | ✅ | L118 | `int64`, gorm index, `json:"party_id,omitempty"`, 注释: "所属 Party" |

**判定**: ✅已修复

---

## 总结

| 检查项 | 结果 |
|--------|------|
| 1. POST /gov/keys 真实实现 + raw_key 仅创建时返回 | ✅已修复 |
| 2. validateGovToken 完整校验链（哈希/状态/过期/禁人） | ✅已修复 |
| 3. GovAPIKey 模型字段完整 | ✅已修复 |
| 4. APIKey 模型含 AccountID / PartyID | ✅已修复 |

**FIX-C 全部 4 条验收项均已关闭，batch-002 缺陷收口确认完成。**
