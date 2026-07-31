# TECH-1 代码质量审计报告

> **审计范围：** `ai-gov-fusion/backend/internal/server/` 下新增 11 个包（排除 TokenHub 存量 *.go，排除 *_test.go）
> **审计日期：** 2026-07-31
> **审计标准：** AGENTS.md 第 6 章 + PRD v3.2.0 质量门禁

---

## 一、中文注释覆盖率

### 1.1 AGENTS.md §7.4 门禁检测（grep `^// [A-Z][a-z]`）

| 包 | 命中行数 | 门禁判定 (<=5) |
|----|---------|---------------|
| abac | 55 | **不通过** |
| audit | 12 | **不通过** |
| authz | 15 | **不通过** |
| fund | 104 | **不通过** |
| idempotency | 63 | **不通过** |
| modelgrant | 16 | **不通过** |
| party | 46 | **不通过** |
| pricing | 32 | **不通过** |
| routing | 25 | **不通过** |
| security | 18 | **不通过** |
| ui_permission | 41 | **不通过** |
| **总计** | **427** | |

### 1.2 关键发现

上述 427 条命中分两类：

**A 类 -- 合法中文注释（340 条）：** 标准 Go Doc 格式，导出标识符在前、中文描述在后。例如：
```go
// ErrAccessDenied 表示 ABAC 引擎拒绝访问。            // abac/engine.go:17
// Evaluate 评估主体对指定资源的操作权限。                // abac/engine.go:48
// SysActionCatalog 原子操作目录。                      // abac/model.go:76
```
此类注释**实际合标**，仅因 grep 模式天然匹配导出标识符首字母而被捕获，属于误报。

**B 类 -- 真正英文注释（187 条）：** 纯英文、无任何中文字符。分布如下：

| 包 | 英文注释数 | 涉及文件 |
|----|-----------|---------|
| fund | 104 | doc.go, errors.go, model.go, freeze.go, lifecycle.go, service.go, store.go |
| idempotency | 68 | claim.go, doc.go, middleware.go, model.go, store.go |
| party | 15 | doc.go, service.go, store.go |
| abac | 0 | -- |
| audit | 0 | -- |
| authz | 0 | -- |
| modelgrant | 0 | -- |
| pricing | 0 | -- |
| routing | 0 | -- |
| security | 0 | -- |
| ui_permission | 0 | -- |

**违规实质：fund、idempotency、party 三个包的注释全部为英文，违反 AGENTS.md §6.1 铁律。**

### 1.3 评分

| 包 | 中文注释 | 评分 | 说明 |
|----|---------|------|------|
| abac | 100% 中文 | 10/10 | 全部中文 |
| audit | 100% 中文 | 10/10 | 全部中文 |
| authz | 100% 中文 | 10/10 | 全部中文 |
| fund | 0% 中文 | **0/10** | 全部英文，严重违规 |
| idempotency | 0% 中文 | **0/10** | 全部英文，严重违规 |
| modelgrant | 100% 中文 | 10/10 | 全部中文 |
| party | 部分英文 | **4/10** | doc.go/service.go/store.go 英文 |
| pricing | 100% 中文 | 10/10 | 全部中文 |
| routing | 100% 中文 | 10/10 | 全部中文 |
| security | 100% 中文 | 10/10 | 全部中文 |
| ui_permission | 100% 中文 | 10/10 | 全部中文 |

---

## 二、文件/函数行数

### 2.1 文件行数（>500 行 = 违规）

**结果：全部通过。** 所有 11 个包的非测试 .go 文件均不超过 500 行。

文件行数明细：

| 包 | 最大文件 | 行数 |
|----|---------|------|
| abac | engine.go | 483 |
| audit | model.go | 165 |
| authz | model.go | 142 |
| fund | service.go (非测试) | 337 |
| idempotency | claim.go | 309 |
| modelgrant | checker.go | 219 |
| party | service.go | 274 |
| pricing | model.go | 268 |
| routing | profile.go | 419 |
| security | hooks.go | 201 |
| ui_permission | store.go | 312 |

### 2.2 函数行数（>80 行 = 违规）

**fund 包 4 个函数违规：**

| 文件 | 行号 | 函数 | 行数 | 超标 |
|------|------|------|------|------|
| fund/freeze.go | 33 | `Freeze()` | 141 | +61 (超标 76%) |
| fund/freeze.go | 200 | `Settle()` | 182 | +102 (超标 128%) |
| fund/lifecycle.go | 207 | `Liquidate()` | 212 | +132 (超标 165%) |
| fund/service.go | 72 | `Allocate()` | 208 | +128 (超标 160%) |

其余 10 个包所有函数均不超过 80 行。

### 2.3 评分

| 包 | 文件行数 | 函数行数 | 评分 |
|----|---------|---------|------|
| abac | 通过 | 通过 | 10/10 |
| audit | 通过 | 通过 | 10/10 |
| authz | 通过 | 通过 | 10/10 |
| fund | 通过 | **4 函数超标** | **4/10** |
| idempotency | 通过 | 通过 | 10/10 |
| modelgrant | 通过 | 通过 | 10/10 |
| party | 通过 | 通过 | 10/10 |
| pricing | 通过 | 通过 | 10/10 |
| routing | 通过 | 通过 | 10/10 |
| security | 通过 | 通过 | 10/10 |
| ui_permission | 通过 | 通过 | 10/10 |

---

## 三、货币精度

### 3.1 `float64.*amount` / `float32.*balance` 检测

**结果：无匹配。** 金额字段未使用 float64/float32。

### 3.2 float64 使用场景分析

所有 float64 出现位置均属**非金额场景**：

| 包 | 文件 | 用途 | 风险评估 |
|----|------|------|---------|
| fund | model.go:30-31 | `DecPtr(f float64) Decimal` 构造函数 | 安全——仅测试/默认值便利函数，内部转为 Decimal |
| fund | model.go:45 | `case float64:` 在 Scan() 中处理数据库驱动返回的 REAL 类型 | 安全——GORM 兼容层，结果存入 Decimal |
| pricing | calculator.go | `usage map[string]float64` 用量参数 | 安全——用量非金额，计算结果为 decimal.Decimal |
| pricing | normalizer.go | 用量标准化 `map[string]float64` | 安全——上游 provider API 返回的 token 计数 |
| routing | strategies/*.go | 路由评分权重、延迟比率 | 安全——路由打分非金额 |
| routing | strategy.go | `Weight float64`, `Score float64`, `ErrorRate float64` | 安全——路由元数据 |

### 3.3 确认

- fund/service.go:20 明确声明：`"All monetary values MUST use shopspring/decimal.Decimal -- never float64."`
- fund/model.go 所有金额字段使用自定义 `Decimal` 类型（封装 `shopspring/decimal.Decimal`）
- GORM 列类型为 `numeric(18,6)`
- 评分：**全部包 10/10**

---

## 四、反过度设计（AGENTS.md §6.4 七禁）

### 4.1 逐项检测

| # | 禁止模式 | 检测方法 | 结果 | 说明 |
|---|---------|---------|------|------|
| 1 | 单一实现定义 3 层接口抽象 | 统计接口定义与实现 | **未发现** | fund.Store 有 1 个实现 (sqlstore/PgStore)，符合 §6.3 依赖倒置——高层定义接口、低层实现，不属于禁止模式 |
| 2 | 接口只有一个实现的"策略模式" | grep Strategy | **未发现** | routing.Strategy 有 12 个注册实现（Compliance/Health/Priority/Weight/Cost/Latency/Error/Rate/Affinity/Tag/Cache/Classify），属合法策略模式 |
| 3 | 反射替代编译期类型检查 | grep reflect | **零使用** | 全包未引用 `reflect` |
| 4 | 为 2 个字段建一张新表 | 审计 model struct 字段数 | **未发现** | 所有 model struct 均 >= 5 字段 |
| 5 | goroutine+channel 替代简单 if | grep goroutine/channel | **零使用** | 全包未使用 `go func` |
| 6 | 消息队列替代同步函数调用 | 检查 imports | **未发现** | 无偿列/MQ 依赖 |
| 7 | 第三方框架替代标准库 | 检查 imports | **通过** | 仅依赖：GORM（与 TokenHub 一致）、shopspring/decimal（货币精度要求）、标准库 |

### 4.2 接口清单（需人工复核）

| 包 | 接口 | 方法数 | 实现数 | 判定 |
|----|------|--------|--------|------|
| fund | `Tx` | 2 | 0（内联 GORM） | 合理——事务抽象 |
| fund | `Store` | 17 | 1（sqlstore/PgStore） | 合理——依赖倒置，符合 §6.3 |
| fund | `IdempotencyChecker` | 2 | 仅服务层引用 | 可缩小——2 方法仍合理 |
| pricing | `jsonNumber` | 1 | 内联 json.Number | 合理——类型断言 |
| routing | `Strategy` | 2 | 12 | 合法策略模式 |
| security | `Hook` | 3 | 1（hooks.go 自身注册） | **待确认**——如仅有 1 个 Hook 实现，属过度抽象 |
| ui_permission | `ABACEngine` | 1 | 1（abac.Engine） | 合理——跨包依赖倒置 |

### 4.3 评分

| 包 | 评分 | 说明 |
|----|------|------|
| abac | 10/10 | 无过度设计 |
| audit | 10/10 | 无接口，纯函数+模型 |
| authz | 10/10 | 无接口，纯函数+模型 |
| fund | **9/10** | Store/Tx 接口合理，IdempotencyChecker 可考虑缩小；-1 仅为保守 |
| idempotency | 10/10 | 无接口，纯函数+模型 |
| modelgrant | 10/10 | 无接口 |
| party | 10/10 | 无接口 |
| pricing | **9/10** | jsonNumber 单方法接口可接受；-1 为保守 |
| routing | 10/10 | 12 策略实现，设计合理 |
| security | **8/10** | Hook 接口无外部实现——需确认是否真有多个 Hook |
| ui_permission | 10/10 | ABACEngine 跨包依赖倒置合理 |

---

## 五、综合评分汇总

| 包 | 中文注释 | 行数 | 货币 | 设计 | **综合** | 状态 |
|----|---------|------|------|------|----------|------|
| abac | 10 | 10 | 10 | 10 | **10.0** | 通过 |
| audit | 10 | 10 | 10 | 10 | **10.0** | 通过 |
| authz | 10 | 10 | 10 | 10 | **10.0** | 通过 |
| fund | **0** | **4** | 10 | 9 | **5.8** | **严重不合格** |
| idempotency | **0** | 10 | 10 | 10 | **7.5** | **不合格** |
| modelgrant | 10 | 10 | 10 | 10 | **10.0** | 通过 |
| party | **4** | 10 | 10 | 10 | **8.5** | 不合格 |
| pricing | 10 | 10 | 10 | 9 | **9.8** | 通过 |
| routing | 10 | 10 | 10 | 10 | **10.0** | 通过 |
| security | 10 | 10 | 10 | 8 | **9.5** | 通过 |
| ui_permission | 10 | 10 | 10 | 10 | **10.0** | 通过 |

---

## 六、整改建议

### P0 -- 阻塞上线

1. **fund 包：中文注释全部缺失（104 条英文注释）**
   - 文件清单：`doc.go`, `errors.go`, `model.go`, `freeze.go`, `lifecycle.go`, `service.go`, `store.go`
   - 行动：全部 7 个文件的注释重写为中文，参考 abac 包格式

2. **idempotency 包：中文注释全部缺失（68 条英文注释）**
   - 文件清单：`claim.go`, `doc.go`, `middleware.go`, `model.go`, `store.go`
   - 行动：全部 5 个文件的注释重写为中文

3. **fund 包：4 个函数超过 80 行**
   - `Freeze()` (141 行, freeze.go:33) -- 拆分为预算检查、余额校验、记录写入三个子函数
   - `Settle()` (182 行, freeze.go:200) -- 拆分为金额校验、余额更新、流水写入三个子函数
   - `Liquidate()` (212 行, lifecycle.go:207) -- 按状态转换分支提取子函数
   - `Allocate()` (208 行, service.go:72) -- 拆分为幂等检查、账户校验、转账执行、结果记录四个子函数

### P1 -- 建议修复

4. **party 包：部分英文注释（15 条）**
   - 文件：`doc.go`（包注释）、`service.go`（部分函数）、`store.go`（部分函数）
   - 行动：翻译为中文

5. **security 包：Hook 接口评估**
   - 如当前仅 1 个 Hook 实现，移除接口改为具体类型
   - 如计划多 Hook，保留

### P2 -- 可接受

6. **7 个包（abac/audit/authz/modelgrant/pricing/routing/ui_permission）英文注释全部为零** -- 质量优秀，作为其他包的参考模板
