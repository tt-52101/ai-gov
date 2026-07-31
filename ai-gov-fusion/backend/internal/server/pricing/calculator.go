package pricing

import (
	"fmt"
	"math"

	"github.com/shopspring/decimal"
)

// ── 双轨计价引擎 ──

// CalculateCost 根据原始用量计算上游成本总额与分项明细。
//
// 内部遍历 price_json.items 中的每个 itemCode，按 cost 轨道的定价模式计算:
//   - flat_fee:         每笔固定费用 = cost.Rate
//   - usage_per_unit:   成本 = cost.Rate × 用量
//   - usage_tiered:     按 cost.Tiers 分段累计
//   - usage_volume:     总量落档后整单按对应档位单价
//   - amortization_fixed: 返回固定摊销额 (不依赖 usage)
//
// usage 是 itemCode → 用量的映射 (由 NormalizeUsage 产生)。
// 对于 usage 中存在但 price 中未配置的 itemCode，该分项 cost = 0。
func CalculateCost(price ModelPrice, usage map[string]float64) (decimal.Decimal, []CostItem, error) {
	pj, err := price.ParsePriceJSON()
	if err != nil {
		return decimal.Zero, nil, fmt.Errorf("calculate cost: %w", err)
	}
	return calculateTrack(pj, usage, trackCost)
}

// CalculateSell 根据原始用量计算内部结算总额与分项明细。
//
// 计算逻辑与 CalculateCost 相同，但使用 sell 轨道的定价参数。
// 对缓存类 itemCode，若 sell.CacheDiscountRatio > 0，则 sell = 正常 sell × CacheDiscountRatio。
// 上游成本不受缓存折扣影响。
func CalculateSell(price ModelPrice, usage map[string]float64) (decimal.Decimal, []CostItem, error) {
	pj, err := price.ParsePriceJSON()
	if err != nil {
		return decimal.Zero, nil, fmt.Errorf("calculate sell: %w", err)
	}
	return calculateTrack(pj, usage, trackSell)
}

// CalculateDualTrack 一次遍历同时计算 cost 与 sell，提高性能并保证一致性。
//
// 返回 UsageResult 包含两轨总额和 per-itemCode 双轨明细。
// 这是数据面结算步骤的首选函数。
func CalculateDualTrack(price ModelPrice, usage map[string]float64) (*UsageResult, error) {
	pj, err := price.ParsePriceJSON()
	if err != nil {
		return nil, fmt.Errorf("calculate dual track: %w", err)
	}

	result := &UsageResult{
		CostAmount: decimal.Zero,
		SellAmount: decimal.Zero,
		CostItems:  make([]CostItem, 0, len(usage)),
	}

	for itemCode, qty := range usage {
		item := pj.FindItem(itemCode)
		costAmt := computeTrackAmount(item, itemCode, qty, trackCost)
		sellAmt := computeTrackAmount(item, itemCode, qty, trackSell)

		result.CostAmount = result.CostAmount.Add(costAmt)
		result.SellAmount = result.SellAmount.Add(sellAmt)

		result.CostItems = append(result.CostItems, CostItem{
			ItemCode:   itemCode,
			Usage:      qty,
			CostAmount: costAmt,
			SellAmount: sellAmt,
		})
	}

	return result, nil
}

// tieredPrice 计算阶梯价格的工具函数。
//
// usage 是实际用量，ratePerUnit 是超出所有 tier 范围后的兜底单价。
// tiers 按 UpTo 升序排列，每个 tier 定义 (用量上限, 单价)。
//
// 算法: 从最低档位开始，计算每段用量 × 该段单价，累加。
// 超出最后一档的部分按 ratePerUnit 计费。
func tieredPrice(usage float64, ratePerUnit decimal.Decimal, tiers []TierRange) decimal.Decimal {
	if usage <= 0 {
		return decimal.Zero
	}
	if len(tiers) == 0 {
		return ratePerUnit.Mul(decimal.NewFromFloat(usage))
	}

	total := decimal.Zero
	remaining := usage
	prevUpTo := int64(0)

	for _, tier := range tiers {
		if remaining <= 0 {
			break
		}
		bandSize := tier.UpTo - prevUpTo
		if bandSize <= 0 {
			prevUpTo = tier.UpTo
			continue
		}
		inBand := math.Min(remaining, float64(bandSize))
		bandCost := tier.Rate.Mul(decimal.NewFromFloat(inBand))
		total = total.Add(bandCost)
		remaining -= inBand
		prevUpTo = tier.UpTo
	}

	// 超出最后一档的用量按兜底单价
	if remaining > 0 {
		total = total.Add(ratePerUnit.Mul(decimal.NewFromFloat(remaining)))
	}

	return total
}

// EstimateSell 根据模型价目和预估 Token 数快速估算 sell 金额。
//
// 用于冻结阶段: P_request = max_estimated_sell across candidate set。
// 选择 prompt_tokens 的 sell 单价 (若存在) 乘以 estimatedTokens 作为粗略估算。
// 若未配置 prompt_tokens 项，则遍历所有 item 取最大的 sell 单价。
//
// 注意: 此函数仅用于预冻结估算，不保证与实际结算金额一致。
func EstimateSell(price ModelPrice, estimatedTokens int) decimal.Decimal {
	pj, err := price.ParsePriceJSON()
	if err != nil {
		return decimal.Zero
	}
	if estimatedTokens <= 0 {
		return decimal.Zero
	}

	// 优先使用 prompt_tokens 的 sell 单价
	if item := pj.FindItem(ItemCodePromptTokens); item != nil {
		return item.Sell.Rate.Mul(decimal.NewFromInt(int64(estimatedTokens)))
	}

	// 回退: 取所有 item 中最大的 sell 单价
	bestRate := decimal.Zero
	for i := range pj.Items {
		if pj.Items[i].Sell.Rate.GreaterThan(bestRate) {
			bestRate = pj.Items[i].Sell.Rate
		}
	}
	return bestRate.Mul(decimal.NewFromInt(int64(estimatedTokens)))
}

// ── 内部轨道路由 ──

type trackType int

const (
	trackCost trackType = iota
	trackSell
)

// calculateTrack 沿单条轨道计算所有 itemCode 的合计金额与分项明细。
func calculateTrack(pj *PriceJSON, usage map[string]float64, track trackType) (decimal.Decimal, []CostItem, error) {
	total := decimal.Zero
	items := make([]CostItem, 0, len(usage))

	for itemCode, qty := range usage {
		item := pj.FindItem(itemCode)
		amt := computeTrackAmount(item, itemCode, qty, track)
		total = total.Add(amt)
		items = append(items, CostItem{
			ItemCode:   itemCode,
			Usage:      qty,
			CostAmount: amt, // 单轨场景，CostAmount 兼用
			SellAmount: decimal.Zero,
		})
	}
	return total, items, nil
}

// computeTrackAmount 根据定价模式计算单个 itemCode 的单轨金额。
// 若 item 为 nil (未配置该 itemCode 的价格)，返回 0。
func computeTrackAmount(item *PriceItem, itemCode string, quantity float64, track trackType) decimal.Decimal {
	if item == nil || quantity <= 0 {
		return decimal.Zero
	}

	tier := item.Cost
	if track == trackSell {
		tier = item.Sell
	}

	amt := computeByMode(tier, quantity)

	// 缓存折扣仅 sell 轨道且适用于缓存类 itemCode
	if track == trackSell && IsCachedItemCode(itemCode) && tier.CacheDiscountRatio.GreaterThan(decimal.Zero) {
		amt = amt.Mul(tier.CacheDiscountRatio)
	}

	return amt
}

// computeByMode 根据 PricingTier 的 Mode 计算金额。
// 五种模式: flat_fee / usage_per_unit / usage_tiered / usage_volume / amortization_fixed。
func computeByMode(tier PricingTier, quantity float64) decimal.Decimal {
	switch tier.Mode {
	case ModeFlatFee:
		// 固定费用，与用量无关
		return tier.Rate

	case ModeUsagePerUnit:
		// 按单位用量计费: 金额 = 单价 × 用量
		return tier.Rate.Mul(decimal.NewFromFloat(quantity))

	case ModeUsageTiered:
		// 阶梯计价: 分段累计
		return tieredPrice(quantity, tier.Rate, tier.Tiers)

	case ModeUsageVolume:
		// 总量落档: 找到匹配的档位，整单按该档位单价
		return volumePrice(quantity, tier.Rate, tier.Tiers)

	case ModeAmortizationFixed:
		// 固定摊销: 返回日摊销额 (优先) 或月摊销额 / 30
		if tier.DailyRate.GreaterThan(decimal.Zero) {
			return tier.DailyRate
		}
		if tier.MonthlyRate.GreaterThan(decimal.Zero) {
			return tier.MonthlyRate.Div(decimal.NewFromInt(30))
		}
		return tier.Rate

	default:
		// 未识别的模式 → 返回 0，避免静默出错
		return decimal.Zero
	}
}

// volumePrice 计算总量落档价格: 找到 Usage ≤ UpTo 的最小档位，整单按该档位单价计算。
// 若无匹配档位，回退到 ratePerUnit × usage。
func volumePrice(usage float64, ratePerUnit decimal.Decimal, tiers []TierRange) decimal.Decimal {
	if usage <= 0 {
		return decimal.Zero
	}

	// tiers 应按 UpTo 升序，找到第一个 UpTo >= usage 的档位
	for _, tier := range tiers {
		if float64(tier.UpTo) >= usage {
			return tier.Rate.Mul(decimal.NewFromFloat(usage))
		}
	}

	// 无匹配档位: 用兜底单价
	return ratePerUnit.Mul(decimal.NewFromFloat(usage))
}
