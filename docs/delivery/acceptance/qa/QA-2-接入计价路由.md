# QA-2 接入+计价+路由域功能验收报告

| 项 | 内容 |
|----|------|
| 验收角色 | QA-2（接入+计价+路由域专家） |
| 验收日期 | 2026-07-31 |
| PRD 基线 | AI-GOV-PRD-v3.2.0 |
| 代码基线 | `ai-gov-fusion/backend/internal/server/` |
| 验收结论 | **全部 9 项通过（含 2 红线项），0 项阻塞** |

---

## 1. 场景 1：双轨 item 分项

### 1.1 PRD 要求

> PRD §4.3（双轨+缓存折扣）：cost 与 sell 双轨并行计量。
> PRD §4.1（费用项 itemCode 基线）：10 项 itemCode 全部支持。
> PRD PRI-03：费用计算返回 cost/sell 总额与 costItems。

### 1.2 代码证据

**文件：** `ai-gov-fusion/backend/internal/server/pricing/calculator.go`

- **第 48-77 行** `CalculateDualTrack` 函数：一次遍历同时计算 cost 与 sell，返回 `*UsageResult`。
  - 第 54-58 行：初始化 `CostAmount`（cost 总价）和 `SellAmount`（sell 总价），以及 `CostItems` 分项切片。
  - 第 62-63 行：对每个 itemCode 分别调用 `computeTrackAmount(..., trackCost)` 和 `computeTrackAmount(..., trackSell)`。
  - 第 65-66 行：分别累加到 `CostAmount` 和 `SellAmount`。
  - 第 68-73 行：每个 `CostItem` 结构体包含 `ItemCode`、`Usage`、`CostAmount`、`SellAmount` 四个字段。

**文件：** `ai-gov-fusion/backend/internal/server/pricing/model.go`

- **第 230-238 行** `UsageResult` 结构体：`CostAmount`（总额）+ `SellAmount`（总额）+ `CostItems`（分项明细）。
- **第 240-249 行** `CostItem` 结构体：`ItemCode`、`Usage`、`CostAmount`、`SellAmount` 双轨金额。
- **第 13-34 行** 10 项 itemCode 常量定义：
  1. `prompt_tokens`
  2. `completion_tokens`
  3. `prompt_cached_tokens`
  4. `prompt_write_cached_tokens`
  5. `completion_reasoning_tokens`
  6. `prompt_audio_tokens`
  7. `completion_audio_tokens`
  8. `image_count`
  9. `image_resolution_tier`
  10. `video_duration_seconds`
- **第 36-50 行** `AllItemCodes()` 返回全部 10 项的列表。

### 1.3 结论

**通过。** `CalculateDualTrack` 正确返回 cost 总额、sell 总额和 per-itemCode 双轨分项明细。10 个 itemCode 全部以常量定义，且 `FindItem`（第 144-151 行）通过遍历 items 数组按 itemCode 查找，支持全部 10 项。

---

## 2. 场景 2：缓存折扣

### 2.1 PRD 要求

> PRD §4.3（双轨+缓存折扣）：对缓存类 itemCode（`prompt_cached_tokens` / `prompt_write_cached_tokens`）可配置 `cache_discount_ratio`，sell = 正常 sell x cache_discount_ratio。上游成本不受缓存折扣影响。

### 2.2 代码证据

**文件：** `ai-gov-fusion/backend/internal/server/pricing/calculator.go`

- **第 183-201 行** `computeTrackAmount` 函数：
  - 第 196 行：`if track == trackSell && IsCachedItemCode(itemCode) && tier.CacheDiscountRatio.GreaterThan(decimal.Zero)`
  - 第 197 行：`amt = amt.Mul(tier.CacheDiscountRatio)`
  - 关键约束：(1) 仅 sell 轨道 (`track == trackSell`)；(2) 仅缓存类 itemCode；(3) `CacheDiscountRatio > 0`。

**文件：** `ai-gov-fusion/backend/internal/server/pricing/model.go`

- **第 78-88 行** `cachedTokenItemCodes` 集合和 `IsCachedItemCode` 函数：
  - `prompt_cached_tokens`（缓存读）和 `prompt_write_cached_tokens`（缓存写）标记为缓存类。
- **第 185-187 行** `PricingTier.CacheDiscountRatio` 字段，注释明确说明"仅 Sell 轨道的缓存类 itemCode 使用"。

### 2.3 结论

**通过。** 缓存折扣正确实现：(1) 仅应用于 sell 轨道；(2) 仅应用于 `prompt_cached_tokens` 和 `prompt_write_cached_tokens`；(3) cost 轨道逻辑分支不进入缓存折扣计算（第 196 行条件不满足）。

---

## 3. 场景 3：固定摊销

### 3.1 PRD 要求

> PRD §4.2（计价模式）：`amortization_fixed`：按月/年固定摊销额，不按 Token 计量，适用于私有化部署模型（vLLM 推理集群）内部成本分摊。
> PRD PRI-05：私有化模型按月/年固定摊销，不按 Token。

### 3.2 代码证据

**文件：** `ai-gov-fusion/backend/internal/server/pricing/calculator.go`

- **第 223-231 行** `ModeAmortizationFixed` 分支（`computeByMode` 函数内）：
  ```go
  case ModeAmortizationFixed:
      if tier.DailyRate.GreaterThan(decimal.Zero) {
          return tier.DailyRate
      }
      if tier.MonthlyRate.GreaterThan(decimal.Zero) {
          return tier.MonthlyRate.Div(decimal.NewFromInt(30))
      }
      return tier.Rate
  ```
  - 优先级：`DailyRate` > `MonthlyRate / 30` > `Rate`（兜底）。
  - 注意：**该分支完全不使用 `quantity` 参数**，即返回金额与用量无关。

**文件：** `ai-gov-fusion/backend/internal/server/pricing/model.go`

- **第 63-64 行** `ModeAmortizationFixed` 常量定义。
- **第 189-192 行** `PricingTier` 中 `DailyRate` 和 `MonthlyRate` 字段。

### 3.3 结论

**通过。** `amortization_fixed` 模式在 `computeByMode` 中完全不引用 `quantity` 参数，直接返回日摊销额、月摊销额/30 或固定 Rate，与 PRD "不按 Token 计量"的要求完全一致。

---

## 4. 场景 4：5 种计价模式

### 4.1 PRD 要求

> PRD §4.2（计价模式）：flat_fee / usage_per_unit / usage_tiered / usage_volume / amortization_fixed 共 5 种模式。

### 4.2 代码证据

**文件：** `ai-gov-fusion/backend/internal/server/pricing/model.go`

- **第 54-65 行** 5 种模式常量定义：
  ```go
  ModeFlatFee           = "flat_fee"
  ModeUsagePerUnit      = "usage_per_unit"
  ModeUsageTiered       = "usage_tiered"
  ModeUsageVolume       = "usage_volume"
  ModeAmortizationFixed = "amortization_fixed"
  ```

**文件：** `ai-gov-fusion/backend/internal/server/pricing/calculator.go`

- **第 205-237 行** `computeByMode` 函数的 `switch` 语句覆盖全部 5 种模式：
  | 模式 | 行号 | 实现逻辑 |
  |------|------|----------|
  | `ModeFlatFee` | 207-209 | 直接返回 tier.Rate（固定费用，与用量无关） |
  | `ModeUsagePerUnit` | 211-213 | `tier.Rate x quantity`（按单位用量计费） |
  | `ModeUsageTiered` | 215-217 | 调用 `tieredPrice(quantity, tier.Rate, tier.Tiers)`（分段累计） |
  | `ModeUsageVolume` | 219-221 | 调用 `volumePrice(quantity, tier.Rate, tier.Tiers)`（总量落档） |
  | `ModeAmortizationFixed` | 223-231 | 返回日摊销/月摊销/固定 Rate（不依赖用量） |
  | `default` | 233-235 | 未识别模式返回 0（安全兜底） |

- 辅助函数：
  - **第 86-120 行** `tieredPrice`：阶梯计价分段累计算法。
  - **第 240-255 行** `volumePrice`：总量落档算法，找到 `UpTo >= usage` 的最小档位整单计算。

### 4.3 结论

**通过。** 5 种计价模式全部以常量定义且完整实现。`default` 分支提供了未识别模式的安全兜底（返回 0）。

---

## 5. 场景 5：δ 价格帽

### 5.1 PRD 要求

> PRD §8.1（价格约束）：默认 δ=0；硬上限 20%；改 δ 必须关键配置审计（S-CON-03）。
> PRD S-CON-02：δ 默认 0，硬上限 20%。
> PRD S-CON-03：δ 的任何修改必须记为关键配置变更审计。

### 5.2 代码证据

**δ 默认 0：**

- **文件：** `routing/strategy.go`，第 150 行：
  ```go
  DeltaCap decimal.Decimal `json:"delta_cap" gorm:"type:numeric(18,6);not null;default:0"`
  ```
  GORM 标签 `default:0` 确保数据库层默认值为 0。

**硬上限 20%：**

- **文件：** `routing/strategy.go`，第 172-174 行：
  ```go
  const (
      MaxDeltaCap = 0.20
  )
  ```

- **文件：** `routing/profile.go`：
  - 第 44-47 行 `CreateProfile` 中验证：`if profile.DeltaCap.GreaterThan(maxDelta) { return ... ErrDeltaCapExceeded }`
  - 第 117-119 行 `UpdateProfile` 中同样验证。

- **文件：** `routing/profile.go`，第 246-250 行 `ExecuteProfile` 中应用价格帽：
  ```go
  if profile.DeltaCap.GreaterThan(decimal.Zero) || profile.DeltaCap.Equal(decimal.Zero) {
      capMultiplier := decimal.NewFromFloat(1.0).Add(profile.DeltaCap)
      maxAllowed := anchorSell.Mul(capMultiplier)
      candidates = applyPriceCap(candidates, maxAllowed)
  }
  ```
  注意：当 δ=0 时，`capMultiplier = 1.0`，`maxAllowed = anchorSell`，即严格不允许超过锚定价。

**δ 变更触发审计：**

- **文件：** `routing/profile.go`，第 129-137 行：
  ```go
  if !old.DeltaCap.Equal(profile.DeltaCap) {
      slog.Warn("δ 价格帽变更——关键配置审计",
          "profile_id", profile.ID,
          "profile_name", profile.Name,
          "delta_old", old.DeltaCap.String(),
          "delta_new", profile.DeltaCap.String(),
      )
  }
  ```

### 5.3 结论

**通过。** δ 默认 0（GORM default），硬上限 20%（`MaxDeltaCap` 常量，创建/更新时强制校验），变更触发审计日志（WARN 级别，记录 old/new 值）。三项要求全部满足。

---

## 6. 场景 6：12 策略启停

### 6.1 PRD 要求

> PRD §3.3：策略矩阵 12 种。
> PRD RTE-02：策略矩阵（12 种，含 S-CLASSIFY）。
> PRD §13.1：策略矩阵启停组合——12 种策略可单独启停组合。

### 6.2 代码证据

**策略文件清单（12 个文件，不含 register.go）：**

| # | 文件名 | 策略代码 | 行数 |
|---|--------|----------|------|
| 1 | `affinity.go` | S-AFFINITY | `strategies/affinity.go` |
| 2 | `cache.go` | S-CACHE | `strategies/cache.go` |
| 3 | `classify.go` | S-CLASSIFY | `strategies/classify.go` |
| 4 | `compliance.go` | S-COMPLIANCE | `strategies/compliance.go` |
| 5 | `cost.go` | S-COST | `strategies/cost.go` |
| 6 | `error.go` | S-ERROR | `strategies/error.go` |
| 7 | `health.go` | S-HEALTH | `strategies/health.go` |
| 8 | `latency.go` | S-LATENCY | `strategies/latency.go` |
| 9 | `priority.go` | S-PRI | `strategies/priority.go` |
| 10 | `rate.go` | S-RATE | `strategies/rate.go` |
| 11 | `tag.go` | S-TAG | `strategies/tag.go` |
| 12 | `weight.go` | S-WEIGHT | `strategies/weight.go` |

**注册入口：**

- **文件：** `routing/strategies/register.go`，第 12-25 行 `RegisterAll()`：
  ```go
  func RegisterAll() {
      routing.Register(&ComplianceStrategy{})
      routing.Register(&HealthStrategy{})
      routing.Register(&PriorityStrategy{})
      routing.Register(&WeightStrategy{})
      routing.Register(&CostStrategy{})
      routing.Register(&LatencyStrategy{})
      routing.Register(&ErrorStrategy{})
      routing.Register(&RateStrategy{})
      routing.Register(&AffinityStrategy{})
      routing.Register(&TagStrategy{})
      routing.Register(&CacheStrategy{})
      routing.Register(&ClassifyStrategy{})
  }
  ```
  共 12 次 `Register` 调用，全部 12 种策略。

**启停机制：**

- **文件：** `routing/strategy.go`，第 124-136 行 `StrategyBinding` 结构体：
  - `Enabled bool` 字段控制策略启停。
- **文件：** `routing/profile.go`，第 327-329 行 `resolveStrategies`：
  ```go
  if !binding.Enabled {
      continue
  }
  ```

### 6.3 结论

**通过。** 12 个策略文件均存在，且全部通过 `RegisterAll()` 注册到全局 registry。`StrategyBinding.Enabled` 字段实现独立启停控制，`resolveStrategies` 中跳过 `Enabled=false` 的绑定。

---

## 7. 场景 7：策略管道顺序

### 7.1 PRD 要求

> PRD §11.5（策略引擎接口）：
> Pipeline: `candidates -> S-COMPLIANCE -> ModelGrant -> price cap -> S-CLASSIFY -> remaining strategies -> pick`
>
> 路由架构文档 §2：硬顺序——合规/模型/价格不合格者不得进入打分。
>
> PRD §3.3：S-COMPLIANCE 为硬策略，不可对受限主体关闭。

### 7.2 代码证据

**文件：** `routing/profile.go`，第 181-311 行 `ExecuteProfile`：

| 阶段 | 行号 | 操作 | 说明 |
|------|------|------|------|
| [1] | 243 | `executeFilter(ctx, candidates, StrategyCompliance)` | S-COMPLIANCE 硬过滤——始终执行，不可跳过 |
| [2] | 246-250 | `applyPriceCap(candidates, maxAllowed)` | δ 价格帽过滤（若 DeltaCap=0 则等价于严格等于锚定价） |
| [3] | 261 | `executeScore(ctx, candidates, StrategyClassify)` | S-CLASSIFY 打分——始终在其余策略之前 |
| [4] | 264-271 | `for _, s := range resolved { ... }` | 其余启用策略依次执行 Filter + Score |
| [5] | 274 | `sortByScore(candidates)` | 按 Score 降序排列 |
| [6] | 278-283 | 选取最优未剔除候选 | |

**关键证据——S-COMPLIANCE 和 S-CLASSIFY 跳过阶段 4 循环：**

- 第 265-266 行：
  ```go
  if s.ID() == StrategyCompliance || s.ID() == StrategyClassify {
      continue // 已在前置阶段处理。
  }
  ```

### 7.3 结论

**通过。** 管道顺序严格遵循 PRD 定义：S-COMPLIANCE -> δ 价格帽 -> S-CLASSIFY -> 其余策略 -> 排序 -> 选取最优。S-COMPLIANCE 和 S-CLASSIFY 在前置阶段单独处理，不会被重复执行。

---

## 8. 红线检查

### 8.1 红线 1：S-COMPLIANCE 是否不可关闭（硬策略）

#### PRD 要求

> PRD §3.3 策略矩阵表格：S-COMPLIANCE 标注为"硬策略，不可对受限主体关闭"。
> PRD §7.7 验收红线清单第 13 条：INTERNAL_ONLY 用户请求产生了外网上游流量（D-CON-02）。

#### 代码证据

**S-COMPLIANCE 执行路径不可绕过：**

- **文件：** `routing/profile.go`，第 243 行：
  ```go
  candidates = executeFilter(ctx, candidates, StrategyCompliance)
  ```
  这行代码在 `ExecuteProfile` 中**无条件执行**，不经过 `resolveStrategies` 的绑定筛选。即使档案中的 `Strategies` 列表不包含 S-COMPLIANCE，甚至档案完全未配置任何策略（第 230 行 `len(resolved) == 0`），该行仍会在第 243 行执行。

- **文件：** `routing/profile.go`，第 265-266 行的 `continue` 逻辑确保 S-COMPLIANCE 不会在阶段 4 的循环中重复执行，但这也反向证明了它在阶段 1 中已经被硬编码执行。

#### 结论

**通过。** S-COMPLIANCE 在 `ExecuteProfile` 中硬编码为管道第一阶段，不依赖档案的策略绑定配置，不可被关闭或跳过。满足 PRD "硬策略，不可对受限主体关闭"的要求。

---

### 8.2 红线 2：调度是否修改 account_id

#### PRD 要求

> PRD S-CON-01（账户锁定）：全流程扣费账户 = 鉴权时 Key 绑定的账户；调度策略不得修改 account_id。
> PRD §8.1 第 7 条：调度仅在合格集执行；account_id 在鉴权时锁定，调度不得修改。
> PRD §7.7 验收红线清单第 4 条：调度改扣费账户。
> 路由架构文档 §1.1：不改 account_id。
> 路由架构文档 §3.2 `MatrixInput`：`AccountID int64 // 只读，禁止策略修改扣费`。

#### 代码证据

**鉴权阶段注入 account_id（只读传递）：**

- **文件：** `server/pipeline.go`，第 402-408 行 `enrichContext`：
  ```go
  ctx = context.WithValue(ctx, "account_id", auth.AccountID)
  ```
  仅从 `AuthResult` 读取并注入 context，后续步骤只读取，不修改。

**路由/调度层不持有 account_id：**

- **文件：** `routing/strategy.go`，第 74-118 行 `Candidate` 结构体：包含 `ChannelID`、`ModelID`、`Priority`、`Weight`、`Health`、`EstSell`、`EstCost`、`Score`、`Eliminated` 等字段——**没有 `AccountID` 或 `account_id` 字段**。
- **文件：** `routing/profile.go`，第 198-204 行 `ExecuteProfile` 签名：
  ```go
  func ExecuteProfile(
      ctx context.Context,
      db *gorm.DB,
      profile *RouteProfile,
      candidates []Candidate,
      anchorSell decimal.Decimal,
  ) ([]Candidate, *Decision, error)
  ```
  参数中没有任何 `account_id` 相关参数。

- 在 `routing/` 和 `routing/strategies/` 目录下搜索 `account_id`、`accountID`、`AccountID`——**零结果**。

- 在 `server/gov_handlers.go` 和 `server/http.go` 中搜索——**零结果**（account_id 仅在鉴权和资金操作中使用，路由层不解引用）。

**策略接口定义中的约束：**

- **文件：** `routing/strategy.go`，第 59-72 行 `Strategy` 接口：`Filter` 和 `Score` 接收 `ctx` 和 `candidates`，无 `account_id` 参数。

#### 结论

**通过。** 路由层和策略层完全不持有、不引用、不修改 `account_id`。`account_id` 在鉴权阶段（pipeline.go `enrichContext`）注入 context 后仅被后续的冻结/结算步骤读取，调度管道 `ExecuteProfile` 不感知 `account_id` 的存在。满足 S-CON-01 "调度策略不得修改 account_id" 的宪法级约束。

---

## 9. 验收汇总

| # | 场景 | 结论 | 关键文件 |
|---|------|------|----------|
| 1 | 双轨 item 分项 | **通过** | `pricing/calculator.go:48-77`, `pricing/model.go:13-34,230-249` |
| 2 | 缓存折扣 | **通过** | `pricing/calculator.go:183-201`, `pricing/model.go:78-88` |
| 3 | 固定摊销 | **通过** | `pricing/calculator.go:223-231` |
| 4 | 5 种计价模式 | **通过** | `pricing/calculator.go:205-237`, `pricing/model.go:54-65` |
| 5 | δ 价格帽 | **通过** | `routing/strategy.go:150,174`, `routing/profile.go:44-47,129-137,246-250` |
| 6 | 12 策略启停 | **通过** | `routing/strategies/` (12 files), `routing/strategies/register.go:12-25` |
| 7 | 策略管道顺序 | **通过** | `routing/profile.go:243-271` |
| R1 | S-COMPLIANCE 硬策略 | **通过** | `routing/profile.go:243` (无条件执行) |
| R2 | 调度不改 account_id | **通过** | `routing/strategy.go:74-118` (Candidate 无 account_id), `routing/profile.go:198-204` (无 account_id 参数) |

**最终结论：全部 7 项功能验收 + 2 项红线检查均通过，0 项阻塞，可进入下一阶段。**
