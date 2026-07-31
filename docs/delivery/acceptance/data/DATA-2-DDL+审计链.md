# DATA-2: DDL 对齐 + 审计链 + 数据安全定理审计

> 审计人：DATA-2 (数据安全专家)
> 日期：2026-07-31
> 基线：`schema/ai-gov-fusion-v3.2.sql` vs `docs/prd/AI-GOV-PRD-v3.2.0.md`

---

## 1. DDL 40 表对齐（PRD §10.1 全景表）

### 1.1 逐表核对

| # | 表名 | PRD 组 | SQL 存在 | 状态 |
|---|------|--------|---------|------|
| 1 | `users` | 第1组: 用户与身份 | 行 23 | PASS |
| 2 | `admin_sessions` | 第1组: 用户与身份 | 行 45 | PASS |
| 3 | `parties` | 第2组: Party | 行 60 | PASS |
| 4 | `party_edges` | 第2组: Party | 行 79 | PASS |
| 5 | `party_members` | 第2组: Party | 行 94 | PASS |
| 6 | `accounts` | 第3组: 资金治理 | 行 114 | PASS |
| 7 | `ledgers` | 第3组: 资金治理 | 行 142 | PASS |
| 8 | `freezes` | 第3组: 资金治理 | 行 170 | PASS |
| 9 | `allocations` | 第3组: 资金治理 | 行 197 | PASS |
| 10 | `liquidations` | 第3组: 资金治理 | 行 216 | PASS |
| 11 | `api_keys` | 第4组: API Key | 行 242 | PASS |
| 12 | `providers` | 第5组: 模型目录 | 行 293 | PASS |
| 13 | `provider_resources` | 第5组: 模型目录 | 行 325 | PASS |
| 14 | `models` | 第5组: 模型目录 | 行 359 | PASS |
| 15 | `provider_models` | 第5组: 模型目录 | 行 403 | PASS |
| 16 | `model_prices` | 第6组: 定价与路由 | 行 438 | PASS |
| 17 | `model_routes` | 第6组: 定价与路由 | 行 456 | PASS |
| 18 | `route_profiles` | 第6组: 定价与路由 | 行 489 | PASS |
| 19 | `sys_action_catalogs` | 第7组: 安全治理 | 行 797 | PASS |
| 20 | `sys_roles` | 第7组: 安全治理 | 行 816 | PASS |
| 21 | `sys_role_permissions` | 第7组: 安全治理 | 行 832 | PASS |
| 22 | `sys_subject_role_bindings` | 第7组: 安全治理 | 行 846 | PASS |
| 23 | `sys_access_policies` | 第7组: 安全治理 | 行 867 | PASS |
| 24 | `sys_access_policy_bindings` | 第7组: 安全治理 | 行 889 | PASS |
| 25 | `sys_ui_menus` | 第7组: 安全治理 | 行 903 | PASS |
| 26 | `sys_ui_routes` | 第7组: 安全治理 | 行 921 | PASS |
| 27 | `sys_ui_action_bindings` | 第7组: 安全治理 | 行 936 | PASS |
| 28 | `grants` | 第8组: 授权治理 | 行 510 | PASS |
| 29 | `model_grants` | 第8组: 授权治理 | 行 529 | PASS |
| 30 | `request_logs` | 第9组: 请求与用量 | 行 552 | PASS |
| 31 | `request_payload_logs` | 第9组: 请求与用量 | 行 604 | PASS |
| 32 | `route_attempt_logs` | 第9组: 请求与用量 | 行 616 | PASS |
| 33 | `usage_records` | 第9组: 请求与用量 | 行 636 | PASS |
| 34 | `quota_buckets` | 第9组: 请求与用量 | 行 686 | PASS |
| 35 | `channel_probes` | 第10组: 可观测 | 行 704 | PASS |
| 36 | `provider_quota_status` | 第10组: 可观测 | 行 721 | PASS |
| 37 | `audit_events` | 第11组: 基础设施 | 行 743 | PASS |
| 38 | `audit_chain_anchors` | 第11组: 基础设施 | 行 957 | PASS |
| 39 | `idempotency_records` | 第11组: 基础设施 | 行 764 | PASS |
| -- | `sys_config` | **不在 PRD §10.1 中** | 行 980 | **DISCREPANCY** |

### 1.2 计票

| 指标 | 数量 |
|------|------|
| PRD §10.2 声称 | 40 表 |
| PRD §10.1 实际枚举 | 39 表 |
| SQL DDL 创建 | 40 表 (含 sys_config) |
| 逐表匹配通过 | 39/39 |
| 额外存在 (PRD 未列出) | 1 (`sys_config`) |

**裁定：NUMERIC ANOMALY（非阻塞）**

PRD §10.2 声称"总计 40 表"但 §10.1 只枚举了 39 张表。SQL DDL 中包含 `sys_config`（第 39 个 CREATE TABLE，分组为"第12组补充: 系统配置"），该表在 PRD 全文搜索无任何匹配——PRD 未提及 `sys_config`。

- `sys_config` 本身设计合理（键值配置存储，支持多类型序列化，有公开/私密区分），但若 PRD 是合同基线，则该表缺少规格支撑。

### 1.3 字段级差异

**liquidations 缺少 `liquidation_type` 字段**

- PRD §10.1 (行 915): `liquidation_type(project_close/org_merge/org_split)`
- SQL DDL (行 216-233): 只有 `status(blocking/draining/refunding/closing/closed)` + `metadata(JSONB)`
- 影响：清算类型信息目前只能通过 JSONB metadata 非结构化存储，无约束。

**request_payload_logs 缺少内置脱敏**

- PRD §10.1 (行 968): `request_body(脱敏), response_body(脱敏)`
- SQL DDL (行 603-612): 字段类型均为 JSONB，无脱敏约束
- 实际脱敏在应用层通过 `redactAuditPayload()` / `isSensitiveAuditKey()` 实现（见下文 D-CON-03 分析）

---

## 2. 审计链锚定

### 2.1 audit_chain_anchors 字段完整性

| 字段 | PRD 要求 | SQL 定义 | 状态 |
|------|---------|---------|------|
| `anchor_hash` | SHA-256 链锚哈希 | `TEXT NOT NULL UNIQUE` (行 959) | PASS |
| `start_event_id` | 起始事件 ID | `TEXT NOT NULL REFERENCES audit_events(id)` (行 960) | PASS |
| `end_event_id` | 结束事件 ID | `TEXT NOT NULL REFERENCES audit_events(id)` (行 961) | PASS |
| `event_count` | 链内事件数 | `INTEGER NOT NULL DEFAULT 0` (行 962) | PASS |

4/4 字段齐全，外加 `created_at` 和 4 个索引。

### 2.2 锚定存储过程

`anchor_audit_chain(p_start_event_id, p_end_event_id)` (行 1194-1233):

- 统计范围内事件数，为 0 时抛异常
- 哈希输入 = `前一锚点哈希 : start_event_id : end_event_id : event_count : NOW()`
- 使用 `digest(…, 'sha256')` (依赖 pgcrypto 扩展)
- 返回新锚点 ID
- **链式结构**：每个锚点引用前一锚点的 `anchor_hash`，形成不可篡改链

**裁定：PASS**

### 2.3 审计事件保护

- `audit_events` 表自身无 UPDATE/DELETE 语句的数据库层阻止（无触发器/规则），PRD §10.3 规定"应用层禁止 UPDATE/DELETE"
- 实际代码 `recordAdminAudit()` (行 8192) 只执行 INSERT，未发现 UPDATE/DELETE audit_events 路径
- `before_snapshot` / `after_snapshot` 字段在 DDL 中已定义为 JSONB

---

## 3. 数据安全守恒定理审计

### 3.1 D-CON-01 数据不越权

- PRD 要求：所有列表/详情/导出按 data 轴授权范围过滤
- 代码证据：`sys_action_catalogs` 按四轴 (data/fund/iam/routing) 分类；`evaluate_access_via_roles()` 存储过程支持 scope_party_id 限定
- 安全钩子 (`security/hooks.go`) 提供请求拦截点，当前 noop
- **裁定：架构就绪，运行时需验证**

### 3.2 D-CON-02 数据不出境

- PRD 要求：INTERNAL_ONLY 零外网流量
- `models` 表有 `network_class` 字段 (internal/external)
- `security/doc.go` 明确标注 SEC-01/SEC-02 规划，当前钩子 noop
- **裁定：架构就绪，运行时需验证**

### 3.3 D-CON-03 密钥不透传（重点审计）

**定理原文 (PRD §0.3.2)：**
> 上游 API Key 仅保存在网关侧加密存储；调用方只持有企业下发的网关 Key；完整明文不落日志、不二次回显

**检测手段原文：**
> 日志脱敏中间件；API 响应仅返回 key_prefix + key_suffix

#### 3.3.1 加密存储：PASS

上游供应商 API Key 使用 **AES-256-GCM** 加密存储：

- `encryptSecret()` (store.go:5627): AES-GCM 加密，输出 `enc:v1:<base64(nonce + ciphertext)>`
- `decryptSecret()` (store.go:5647): 对应解密
- `secretKeyBytes()` (store.go:5675): 主密钥通过 SHA-256 派生为 32 字节 AES-256 密钥
- 加密范围：`providers.api_key`、`provider_resources.api_key`、`provider_resources.credential_blob`、OAuth 令牌

#### 3.3.2 网关 Key 哈希存储：PASS

- `APIKey.KeyHash` = `sha256(secret)` (types.go:1122)
- `APIKey.KeyHash` 标记 `json:"-"` (types.go:118)，序列化时排除
- `publicKey()` (store.go:5573) 返回前显式清空 `KeyHash`

#### 3.3.3 日志脱敏：PASS

- `isSensitiveAuditKey()` (http.go:8356-8363): 脱敏 apikey, accesstoken, refreshtoken, clientsecret, secretkey, password, token, secret, authorization 等
- `redactAuditValue()` (http.go:8333): 递归遍历 JSON，敏感键替换为 `[redacted]`
- API Key 审计事件仅记录 `project_id`, `name`, `owner_user_id`，不记录 raw key（http.go:3514）
- API Key 轮换审计仅记录 `new_key_id`（http.go:3563）

#### 3.3.4 API 响应回显：PARTIAL VIOLATION

| 场景 | 位置 | 行为 | 合规 |
|------|------|------|------|
| 创建 API Key | http.go:3515-3522 | 返回 `"api_key": <raw secret>` | **违规** |
| 轮换 API Key | http.go:3564-3572 | 返回 `"api_key": <raw secret>` | **违规** |
| 审批创建 API Key | http.go:6691-6698 | 返回 `"api_key": <raw secret>` | **违规** |
| 查询 API Key 列表 | `publicKey()` | 返回 `key_prefix` + `key_suffix`，不含 raw key | PASS |
| 查询 API Key 详情 | `publicKey()` | 同上 | PASS |

**证据：**
```json
// http.go:3515-3522 — 创建 API Key 响应
{
  "id": "key_xxx",
  "api_key": "<raw_secret>",               // <-- 完整明文
  "name": "...",
  "project_id": "...",
  "owner_user_id": "...",
  "plain_text_visible_once": true          // <-- 前端标记
}
```

**分析：**
这是"一次性明文展示"模式，GitHub、OpenAI、Stripe 等主流 API 均采用此模式——创建/轮换时返回明文，之后不可再获取。但 D-CON-03 明确要求"API 响应仅返回 key_prefix + key_suffix"，严格的合规立场下这是一项偏差。

**缓解因素：**
- `"plain_text_visible_once": true` 标记告知前端仅展示一次
- 前端应通过 UI-08 要求（"密钥仓库无明文二次回显"）约束此行为
- 非创建/轮换路径（列表、详情）均不返回 raw key

**裁定：TECHNICAL DEBT（非阻塞，需要前端遵守 plain_text_visible_once 约定）**

#### 3.3.5 安全钩子运行时保护：NOOP

`security/hooks.go` 中的安全钩子链当前为 `NoopHook`，不执行任何实际脱敏。注释中规划了：
- 内容安全引擎（敏感词检测）
- 提示词注入检测
- **敏感数据脱敏（手机号、身份证号、银行卡号等）**  <-- D-CON-03 相关
- 出网管控（INTERNAL_ONLY 强制拦截）

**裁定：未实施，后续阶段需接入**

### 3.4 D-CON-04 审计不可篡改

- `audit_events` 表：代码仅 INSERT，未发现 UPDATE/DELETE 路径
- `before_snapshot` / `after_snapshot` 记录配置变更前后状态
- `audit_chain_anchors` 提供哈希链锚定防篡改
- **裁定：PASS（应用层遵守）**

### 3.5 A-CON-01~05 权限守恒

- `evaluate_access()` 存储过程四轴独立判定，无隐式继承
- `evaluate_access_via_roles()` 支持 scope_party_id 作用域和有效期
- ModelGrant deny 优先于 allow
- **裁定：架构就绪**

---

## 4. 总结与建议

### 通过项

| 项目 | 状态 |
|------|------|
| 39 张 PRD 表全部存在于 DDL | PASS |
| audit_chain_anchors 4 字段完整 | PASS |
| anchor_audit_chain() 存储过程 | PASS |
| 上游 Key AES-256-GCM 加密存储 | PASS |
| 网关 Key SHA-256 哈希存储 | PASS |
| 审计日志不记录 raw Key | PASS |
| 审计负载脱敏 (redactAuditPayload) | PASS |
| 审计不可篡改（仅 INSERT） | PASS |

### 偏差项

| 项目 | 严重级别 | 详情 |
|------|---------|------|
| `sys_config` 表不在 PRD §10.1 | 文档偏差 | PRD 声称 40 表但仅枚举 39 表；SQL 额外包含 sys_config |
| `liquidations.liquidation_type` 缺失 | 字段偏差 | PRD 定义的列未在 DDL 实现 |
| 创建/轮换 API Key 响应返回明文 | D-CON-03 偏差 | 行业标准做法但与定理严格措辞冲突 |
| 安全钩子 noop | 未实施 | 运行时无密钥脱敏/出网管控等保护 |

### 阻塞判定

- **不阻塞发布**：所有偏差均可通过文档修正或后续迭代解决
- D-CON-03 的明文回显是刻意的设计选择（一次性展示），需验收前端 `plain_text_visible_once` 处理
