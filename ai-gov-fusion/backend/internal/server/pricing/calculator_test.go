package pricing

import (
	"testing"

	"github.com/shopspring/decimal"
)

// ── 测试辅助函数 ──

// makeTestPrice 创建一个简单的测试用 ModelPrice，并设置给定的 items。
func makeTestPrice(t *testing.T, items []PriceItem) ModelPrice {
	t.Helper()
	mp := ModelPrice{
		ID:          "test-price-id",
		ModelID:     "gpt-4o",
		ReferenceID: "test-ref-001",
		Status:      StatusActive,
	}
	mp.MustSetPriceJSON(&PriceJSON{Items: items})
	return mp
}

// makeItem 快速构造 PriceItem。
func makeItem(code string, costMode string, costRate float64, sellMode string, sellRate float64) PriceItem {
	return PriceItem{
		ItemCode: code,
		Cost:     PricingTier{Mode: costMode, Rate: decimal.NewFromFloat(costRate)},
		Sell:     PricingTier{Mode: sellMode, Rate: decimal.NewFromFloat(sellRate)},
	}
}

// makeItemTiered 构造带阶梯的 PriceItem。
func makeItemTiered(
	code string, costMode string, costRate float64, costTiers []TierRange,
	sellMode string, sellRate float64, sellTiers []TierRange,
) PriceItem {
	return PriceItem{
		ItemCode: code,
		Cost:     PricingTier{Mode: costMode, Rate: decimal.NewFromFloat(costRate), Tiers: costTiers},
		Sell:     PricingTier{Mode: sellMode, Rate: decimal.NewFromFloat(sellRate), Tiers: sellTiers},
	}
}

// assertDecimal 比较两个 decimal.Decimal，不等时报告。
func assertDecimal(t *testing.T, label string, expected, actual decimal.Decimal) {
	t.Helper()
	if !expected.Equal(actual) {
		t.Errorf("%s: expected %s, got %s", label, expected.String(), actual.String())
	}
}

// ── TestCalculateDualTrack_UsagePerUnit ──

func TestCalculateDualTrack_UsagePerUnit(t *testing.T) {
	items := []PriceItem{
		makeItem(ItemCodePromptTokens, ModeUsagePerUnit, 0.002, ModeUsagePerUnit, 0.003),
		makeItem(ItemCodeCompletionTokens, ModeUsagePerUnit, 0.008, ModeUsagePerUnit, 0.012),
	}
	mp := makeTestPrice(t, items)

	usage := map[string]float64{
		ItemCodePromptTokens:     5000,
		ItemCodeCompletionTokens: 1500,
	}

	result, err := CalculateDualTrack(mp, usage)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// cost: 5000*0.002 + 1500*0.008 = 10 + 12 = 22
	assertDecimal(t, "cost total", decimal.NewFromFloat(22.0), result.CostAmount)
	// sell: 5000*0.003 + 1500*0.012 = 15 + 18 = 33
	assertDecimal(t, "sell total", decimal.NewFromFloat(33.0), result.SellAmount)

	if len(result.CostItems) != 2 {
		t.Fatalf("expected 2 cost items, got %d", len(result.CostItems))
	}
}

// ── TestCalculateDualTrack_Tiered ──

func TestCalculateDualTrack_Tiered(t *testing.T) {
	costTiers := []TierRange{
		{UpTo: 1000000, Rate: decimal.NewFromFloat(2.0)},
	}
	sellTiers := []TierRange{
		{UpTo: 1000000, Rate: decimal.NewFromFloat(3.0)},
	}

	items := []PriceItem{
		makeItemTiered(ItemCodePromptTokens,
			ModeUsageTiered, 1.5, costTiers,
			ModeUsageTiered, 2.0, sellTiers),
	}
	mp := makeTestPrice(t, items)

	// 1.5M tokens: 前 1M @ $2.0 = $2M, 后 0.5M @ $1.5 = $0.75M → $2.75M
	usage := map[string]float64{ItemCodePromptTokens: 1500000}

	result, err := CalculateDualTrack(mp, usage)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// cost: 1M*2 + 0.5M*1.5 = 2M + 0.75M = 2,750,000
	assertDecimal(t, "cost total (tiered)", decimal.NewFromFloat(2750000.0), result.CostAmount)
	// sell: 1M*3 + 0.5M*2 = 3M + 1M = 4,000,000
	assertDecimal(t, "sell total (tiered)", decimal.NewFromFloat(4000000.0), result.SellAmount)
}

// ── TestCalculateDualTrack_FlatFee ──

func TestCalculateDualTrack_FlatFee(t *testing.T) {
	items := []PriceItem{
		makeItem(ItemCodeImageCount, ModeFlatFee, 0.02, ModeFlatFee, 0.03),
	}
	mp := makeTestPrice(t, items)

	usage := map[string]float64{ItemCodeImageCount: 3}

	result, err := CalculateDualTrack(mp, usage)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertDecimal(t, "cost total (flat)", decimal.NewFromFloat(0.02), result.CostAmount)
	assertDecimal(t, "sell total (flat)", decimal.NewFromFloat(0.03), result.SellAmount)
}

// ── TestCalculateDualTrack_CacheDiscount ──

func TestCalculateDualTrack_CacheDiscount(t *testing.T) {
	items := []PriceItem{
		{
			ItemCode: ItemCodePromptCachedTokens,
			Cost:     PricingTier{Mode: ModeUsagePerUnit, Rate: decimal.NewFromFloat(0.0005)},
			Sell: PricingTier{
				Mode:               ModeUsagePerUnit,
				Rate:               decimal.NewFromFloat(0.00075),
				CacheDiscountRatio: decimal.NewFromFloat(0.5),
			},
		},
	}
	mp := makeTestPrice(t, items)

	usage := map[string]float64{ItemCodePromptCachedTokens: 2000}

	result, err := CalculateDualTrack(mp, usage)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// cost: 2000 * 0.0005 = 1.0 (无折扣)
	assertDecimal(t, "cost total (cache)", decimal.NewFromFloat(1.0), result.CostAmount)
	// sell: 2000 * 0.00075 * 0.5 = 0.75
	assertDecimal(t, "sell total (cache 50pct off)", decimal.NewFromFloat(0.75), result.SellAmount)
}

// ── TestCalculateDualTrack_Amortization ──

func TestCalculateDualTrack_Amortization(t *testing.T) {
	items := []PriceItem{
		{
			ItemCode: "amortization_fixed",
			Cost: PricingTier{
				Mode:        ModeAmortizationFixed,
				DailyRate:   decimal.NewFromFloat(166.67),
				MonthlyRate: decimal.NewFromFloat(5000.00),
			},
			Sell: PricingTier{
				Mode:        ModeAmortizationFixed,
				DailyRate:   decimal.NewFromFloat(166.67),
				MonthlyRate: decimal.NewFromFloat(5000.00),
			},
		},
	}
	mp := makeTestPrice(t, items)

	usage := map[string]float64{"amortization_fixed": 999999}

	result, err := CalculateDualTrack(mp, usage)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := decimal.NewFromFloat(166.67)
	assertDecimal(t, "cost total (amortization)", expected, result.CostAmount)
	assertDecimal(t, "sell total (amortization)", expected, result.SellAmount)
}

// TestCalculateDualTrack_AmortizationMonthly 测试仅配 monthly_rate 时自动折算日率。
func TestCalculateDualTrack_AmortizationMonthly(t *testing.T) {
	items := []PriceItem{
		{
			ItemCode: "amortization_fixed",
			Cost:     PricingTier{Mode: ModeAmortizationFixed, MonthlyRate: decimal.NewFromFloat(3000.00)},
			Sell:     PricingTier{Mode: ModeAmortizationFixed, MonthlyRate: decimal.NewFromFloat(3000.00)},
		},
	}
	mp := makeTestPrice(t, items)

	usage := map[string]float64{"amortization_fixed": 1}

	result, err := CalculateDualTrack(mp, usage)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 3000 / 30 = 100
	expected := decimal.NewFromFloat(100.0)
	assertDecimal(t, "cost total (monthly amort)", expected, result.CostAmount)
	assertDecimal(t, "sell total (monthly amort)", expected, result.SellAmount)
}

// ── TestCalculateSell_DeltaCap ──

// TestCalculateSell_DeltaCap 验证 sell 相对 cost 的价差，供上层 price-cap filter 使用。
func TestCalculateSell_DeltaCap(t *testing.T) {
	items := []PriceItem{
		makeItem(ItemCodePromptTokens, ModeUsagePerUnit, 0.002, ModeUsagePerUnit, 0.003),
		makeItem(ItemCodeCompletionTokens, ModeUsagePerUnit, 0.008, ModeUsagePerUnit, 0.012),
	}
	mp := makeTestPrice(t, items)

	usage := map[string]float64{
		ItemCodePromptTokens:     1000,
		ItemCodeCompletionTokens: 500,
	}

	sellTotal, _, err := CalculateSell(mp, usage)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	costTotal, _, err := CalculateCost(mp, usage)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// sell: 1000*0.003 + 500*0.012 = 3 + 6 = 9
	assertDecimal(t, "sell total", decimal.NewFromFloat(9.0), sellTotal)
	// cost: 1000*0.002 + 500*0.008 = 2 + 4 = 6
	assertDecimal(t, "cost total", decimal.NewFromFloat(6.0), costTotal)

	markup := sellTotal.Div(costTotal)
	if !markup.Equal(decimal.NewFromFloat(1.5)) {
		t.Errorf("markup: expected 1.5, got %s", markup.String())
	}

	// 演示: delta=20% 硬上限时 sell 9 > cost*1.2=7.2 → 会被 price-cap filter 拦截
	cap := costTotal.Mul(decimal.NewFromFloat(1.2))
	if sellTotal.GreaterThan(cap) {
		t.Logf("sell %.2f exceeds cap %.2f — would be filtered (expected for 50%% markup)",
			sellTotal.InexactFloat64(), cap.InexactFloat64())
	}
}

// ── TestEstimateSell ──

func TestEstimateSell(t *testing.T) {
	items := []PriceItem{
		makeItem(ItemCodePromptTokens, ModeUsagePerUnit, 0.002, ModeUsagePerUnit, 0.003),
		makeItem(ItemCodeCompletionTokens, ModeUsagePerUnit, 0.008, ModeUsagePerUnit, 0.012),
	}
	mp := makeTestPrice(t, items)

	est := EstimateSell(mp, 1000)
	// prompt_tokens.sell.rate * 1000 = 3.0
	if !est.Equal(decimal.NewFromFloat(3.0)) {
		t.Errorf("EstimateSell: expected 3.0, got %s", est.String())
	}
}

func TestEstimateSell_NoPromptTokens(t *testing.T) {
	items := []PriceItem{
		makeItem(ItemCodeCompletionTokens, ModeUsagePerUnit, 0.008, ModeUsagePerUnit, 0.012),
		makeItem(ItemCodeImageCount, ModeFlatFee, 0.02, ModeFlatFee, 0.03),
	}
	mp := makeTestPrice(t, items)

	est := EstimateSell(mp, 500)
	// max rate = 0.03; 500 * 0.03 = 15.0
	if !est.Equal(decimal.NewFromFloat(15.0)) {
		t.Errorf("EstimateSell (no prompt): expected 15.0, got %s", est.String())
	}
}

func TestEstimateSell_ZeroTokens(t *testing.T) {
	items := []PriceItem{
		makeItem(ItemCodePromptTokens, ModeUsagePerUnit, 0.002, ModeUsagePerUnit, 0.003),
	}
	mp := makeTestPrice(t, items)

	est := EstimateSell(mp, 0)
	if !est.Equal(decimal.Zero) {
		t.Errorf("EstimateSell(0): expected 0, got %s", est.String())
	}
}

// ── TestCalculateDualTrack_MultiItem ──

func TestCalculateDualTrack_MultiItem(t *testing.T) {
	items := []PriceItem{
		makeItem(ItemCodePromptTokens, ModeUsagePerUnit, 0.002, ModeUsagePerUnit, 0.003),
		makeItem(ItemCodeCompletionTokens, ModeUsagePerUnit, 0.008, ModeUsagePerUnit, 0.012),
		{
			ItemCode: ItemCodePromptCachedTokens,
			Cost:     PricingTier{Mode: ModeUsagePerUnit, Rate: decimal.NewFromFloat(0.0005)},
			Sell: PricingTier{
				Mode:               ModeUsagePerUnit,
				Rate:               decimal.NewFromFloat(0.00075),
				CacheDiscountRatio: decimal.NewFromFloat(0.5),
			},
		},
	}
	mp := makeTestPrice(t, items)

	usage := map[string]float64{
		ItemCodePromptTokens:       5000,
		ItemCodeCompletionTokens:   1500,
		ItemCodePromptCachedTokens: 2000,
	}

	result, err := CalculateDualTrack(mp, usage)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// cost: 10 + 12 + 1 = 23
	assertDecimal(t, "cost total (multi)", decimal.NewFromFloat(23.0), result.CostAmount)
	// sell: 15 + 18 + 0.75 = 33.75
	assertDecimal(t, "sell total (multi)", decimal.NewFromFloat(33.75), result.SellAmount)

	if len(result.CostItems) != 3 {
		t.Fatalf("expected 3 cost items, got %d", len(result.CostItems))
	}
}

// ── TestCalculateDualTrack_UsageVolume ──

func TestCalculateDualTrack_UsageVolume(t *testing.T) {
	costTiers := []TierRange{
		{UpTo: 1000, Rate: decimal.NewFromFloat(3.0)},
		{UpTo: 5000, Rate: decimal.NewFromFloat(2.5)},
	}
	sellTiers := []TierRange{
		{UpTo: 1000, Rate: decimal.NewFromFloat(4.5)},
		{UpTo: 5000, Rate: decimal.NewFromFloat(3.75)},
	}

	items := []PriceItem{
		makeItemTiered(ItemCodeImageCount,
			ModeUsageVolume, 2.0, costTiers,
			ModeUsageVolume, 3.0, sellTiers),
	}
	mp := makeTestPrice(t, items)

	// 2000 张图 → 落入第二档: cost = 2000 * 2.5 = 5000
	usage := map[string]float64{ItemCodeImageCount: 2000}

	result, err := CalculateDualTrack(mp, usage)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertDecimal(t, "cost total (volume)", decimal.NewFromFloat(5000.0), result.CostAmount)
	assertDecimal(t, "sell total (volume)", decimal.NewFromFloat(7500.0), result.SellAmount)
}

// ── TestCalculateDualTrack_UnknownItemCode ──

func TestCalculateDualTrack_UnknownItemCode(t *testing.T) {
	items := []PriceItem{
		makeItem(ItemCodePromptTokens, ModeUsagePerUnit, 0.002, ModeUsagePerUnit, 0.003),
	}
	mp := makeTestPrice(t, items)

	usage := map[string]float64{
		ItemCodePromptTokens:     1000,
		ItemCodeCompletionTokens: 500, // 未配价 → 0
	}

	result, err := CalculateDualTrack(mp, usage)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertDecimal(t, "cost total (unknown item)", decimal.NewFromFloat(2.0), result.CostAmount)

	for _, ci := range result.CostItems {
		if ci.ItemCode == ItemCodeCompletionTokens {
			if !ci.CostAmount.Equal(decimal.Zero) || !ci.SellAmount.Equal(decimal.Zero) {
				t.Errorf("unpriced item %s should be zero, got cost=%s sell=%s",
					ci.ItemCode, ci.CostAmount.String(), ci.SellAmount.String())
			}
		}
	}
}
