package pricing

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

// ── ItemCode 计费项编码常量 (PRD §4.1, 10 项基线) ──

const (
	// ItemCodePromptTokens 输入文本 Token 用量。
	ItemCodePromptTokens = "prompt_tokens"
	// ItemCodeCompletionTokens 输出文本 Token 用量。
	ItemCodeCompletionTokens = "completion_tokens"
	// ItemCodePromptCachedTokens 缓存读 Token 用量，可享受 sell 端缓存折扣。
	ItemCodePromptCachedTokens = "prompt_cached_tokens"
	// ItemCodePromptWriteCachedTokens 缓存写 Token 用量。
	ItemCodePromptWriteCachedTokens = "prompt_write_cached_tokens"
	// ItemCodeCompletionReasoningTokens 推理输出 Token 用量。
	ItemCodeCompletionReasoningTokens = "completion_reasoning_tokens"
	// ItemCodePromptAudioTokens 音频输入 Token 用量。
	ItemCodePromptAudioTokens = "prompt_audio_tokens"
	// ItemCodeCompletionAudioTokens 音频输出 Token 用量。
	ItemCodeCompletionAudioTokens = "completion_audio_tokens"
	// ItemCodeImageCount 图片张数。
	ItemCodeImageCount = "image_count"
	// ItemCodeImageResolutionTier 图片分辨率档位 (standard/hd/4k)。
	ItemCodeImageResolutionTier = "image_resolution_tier"
	// ItemCodeVideoDurationSeconds 视频时长 (秒)。
	ItemCodeVideoDurationSeconds = "video_duration_seconds"
)

// AllItemCodes 返回所有 10 项基线 itemCode 的完整列表。
func AllItemCodes() []string {
	return []string{
		ItemCodePromptTokens,
		ItemCodeCompletionTokens,
		ItemCodePromptCachedTokens,
		ItemCodePromptWriteCachedTokens,
		ItemCodeCompletionReasoningTokens,
		ItemCodePromptAudioTokens,
		ItemCodeCompletionAudioTokens,
		ItemCodeImageCount,
		ItemCodeImageResolutionTier,
		ItemCodeVideoDurationSeconds,
	}
}

// ── 定价模式常量 ──

const (
	// ModeFlatFee 按次固定费用，与用量无关。
	ModeFlatFee = "flat_fee"
	// ModeUsagePerUnit 按单位用量计费 (通常每 1 Token、每张图片等)。
	ModeUsagePerUnit = "usage_per_unit"
	// ModeUsageTiered 阶梯价格，分段累计。
	ModeUsageTiered = "usage_tiered"
	// ModeUsageVolume 总量落档后整单同一单价。
	ModeUsageVolume = "usage_volume"
	// ModeAmortizationFixed 按月/年固定摊销额，不按 Token 计量，适用于私有化部署模型。
	ModeAmortizationFixed = "amortization_fixed"
)

// ── 状态常量 ──

const (
	// StatusActive 表示价目有效，可被路由和结算引用。
	StatusActive = "active"
	// StatusArchived 表示价目已归档，不再参与新调用。
	StatusArchived = "archived"
)

// ── 缓存折扣相关的 itemCode 集合 ──

// cachedTokenItemCodes 定义可享受 sell 端缓存折扣的 itemCode 集合。
// 只有缓存读/写 Token 才会配置 cache_discount_ratio，其余 itemCode 忽略此字段。
var cachedTokenItemCodes = map[string]bool{
	ItemCodePromptCachedTokens:      true,
	ItemCodePromptWriteCachedTokens: true,
}

// IsCachedItemCode 判断指定 itemCode 是否为缓存类型，可享受 sell 端折扣。
func IsCachedItemCode(code string) bool {
	return cachedTokenItemCodes[code]
}

// ── 数据表模型 ──

// ModelPrice 对应 model_prices 表，存储渠道 x 模型的双轨价目 JSON。
// 每条记录由 reference_id 唯一标识，支持按模型 + 渠道查询。
// GORM 自动建表时使用 model_prices 作为表名。
type ModelPrice struct {
	ID               string     `json:"id" gorm:"primaryKey"`
	ModelID          string     `json:"model_id" gorm:"index:idx_model_prices_lookup,priority:1;not null"`
	ChannelID        *string    `json:"channel_id,omitempty" gorm:"index:idx_model_prices_lookup,priority:2"`
	ReferenceID      string     `json:"reference_id" gorm:"uniqueIndex;not null"`
	PriceJSONStr     string     `json:"-" gorm:"column:price_json;type:jsonb;not null"`
	Status           string     `json:"status" gorm:"default:active;not null"`
	EffectiveStartAt *time.Time `json:"effective_start_at,omitempty"`
	EffectiveEndAt   *time.Time `json:"effective_end_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// TableName 返回 model_prices 表名 (GORM 约定)。
func (ModelPrice) TableName() string { return "model_prices" }

// ParsePriceJSON 将 PriceJSONStr (JSON 字符串) 解析为 PriceJSON 结构体。
// 解析失败时返回错误，调用方应将其视为数据损坏并记录告警。
func (mp *ModelPrice) ParsePriceJSON() (*PriceJSON, error) {
	if mp.PriceJSONStr == "" {
		return nil, fmt.Errorf("model_price %s: price_json is empty", mp.ID)
	}
	var pj PriceJSON
	if err := json.Unmarshal([]byte(mp.PriceJSONStr), &pj); err != nil {
		return nil, fmt.Errorf("model_price %s: parse price_json: %w", mp.ID, err)
	}
	return &pj, nil
}

// SetPriceJSON 将 PriceJSON 结构体序列化并写入 PriceJSONStr 字段。
func (mp *ModelPrice) SetPriceJSON(pj *PriceJSON) error {
	b, err := json.Marshal(pj)
	if err != nil {
		return fmt.Errorf("marshal price_json: %w", err)
	}
	mp.PriceJSONStr = string(b)
	return nil
}

// MustSetPriceJSON 同 SetPriceJSON，但在序列化失败时 panic。
// 仅用于测试与初始化阶段，生产代码应使用 SetPriceJSON。
func (mp *ModelPrice) MustSetPriceJSON(pj *PriceJSON) {
	if err := mp.SetPriceJSON(pj); err != nil {
		panic(err)
	}
}

// FindItem 在价目 JSON 的 items 数组中按 itemCode 查找价格项。
// 未找到时返回 nil；调用方需自行处理缺项逻辑。
func (pj *PriceJSON) FindItem(itemCode string) *PriceItem {
	for i := range pj.Items {
		if pj.Items[i].ItemCode == itemCode {
			return &pj.Items[i]
		}
	}
	return nil
}

// ── 价格 JSON 结构体 (PRD §4.4) ──

// PriceJSON 定义 model_prices.price_json 列的完整 Go 结构。
// 包含计费项数组和可选的时段覆盖配置。
type PriceJSON struct {
	// Items 是计费项列表，每个元素对应一个 itemCode 的双轨定价。
	Items []PriceItem `json:"items"`
	// Schedule 是可选的时段覆盖配置，nil 表示单一费率。
	Schedule *PriceSchedule `json:"schedule,omitempty"`
}

// PriceItem 定义单个计费项的 cost 与 sell 双轨定价。
type PriceItem struct {
	// ItemCode 计费项编码，必须是 10 项基线之一。
	ItemCode string `json:"itemCode"`
	// Cost 上游成本定价档位。
	Cost PricingTier `json:"cost"`
	// Sell 内部结算定价档位。
	Sell PricingTier `json:"sell"`
}

// PricingTier 定义一种定价模式的参数。
// 5 种模式: flat_fee / usage_per_unit / usage_tiered / usage_volume / amortization_fixed。
type PricingTier struct {
	// Mode 定价模式标识，必须为五种常量之一。
	Mode string `json:"mode"`
	// Rate 单价 — 对 flat_fee 为每笔固定费用；对 usage_per_unit 为每单位用量价格。
	// 使用 decimal.Decimal 以避免浮点舍入误差。
	Rate decimal.Decimal `json:"rate"`
	// Tiers 阶梯范围数组，仅 usage_tiered / usage_volume 模式使用。
	// usage_tiered: 按 UpTo 分段累计；usage_volume: 总量落档后整单使用对应 Rate。
	Tiers []TierRange `json:"tiers,omitempty"`
	// CacheDiscountRatio 缓存折扣比例，仅 Sell 轨道的缓存类 itemCode 使用 (PRD §4.3)。
	// 例如 0.5 表示 sell = 正常 sell * 0.5。
	// cost 轨道的此字段被忽略。
	CacheDiscountRatio decimal.Decimal `json:"cache_discount_ratio,omitempty"`
	// DailyRate 每日摊销金额，仅 amortization_fixed 模式使用。
	DailyRate decimal.Decimal `json:"daily_rate,omitempty"`
	// MonthlyRate 每月摊销金额，仅 amortization_fixed 模式使用。
	MonthlyRate decimal.Decimal `json:"monthly_rate,omitempty"`
}

// TierRange 定义阶梯计价的一个档位范围。
type TierRange struct {
	// UpTo 该档位的用量上限 (含)，对 usage_tiered 和 usage_volume 的含义不同:
	//   usage_tiered: 前 UpTo 单位按此 Rate 计费
	//   usage_volume:   总量 ≤ UpTo 时整单按此 Rate 计费
	UpTo int64 `json:"up_to"`
	// Rate 该档位的单价。
	Rate decimal.Decimal `json:"rate"`
}

// PriceSchedule 定义分时段定价配置。
// 当前存储时区与覆盖规则；具体的分时计价逻辑在 calculator 中实现。
type PriceSchedule struct {
	// Timezone 时区标识，如 "Asia/Shanghai"、"UTC"。
	Timezone string `json:"timezone"`
	// Overrides 时段覆盖列表，按优先级排序。空列表表示无覆盖。
	Overrides []ScheduleOverride `json:"overrides,omitempty"`
}

// ScheduleOverride 定义单个时段的价格覆盖规则。
type ScheduleOverride struct {
	// Name 覆盖规则名称，用于审计与展示。
	Name string `json:"name"`
	// StartHour 起始小时 (0-23)，含。
	StartHour int `json:"start_hour"`
	// EndHour 结束小时 (0-24)，不含。
	EndHour int `json:"end_hour"`
	// DaysOfWeek 生效的星期 (0=周日, 6=周六)，nil 表示每天。
	DaysOfWeek []int `json:"days_of_week,omitempty"`
	// ItemOverrides 该时段内对特定 itemCode 的覆盖价格。
	ItemOverrides []PriceItem `json:"item_overrides,omitempty"`
}

// ── 计算结果结构体 ──

// UsageResult 包含双轨计价完整结果：cost 总额、sell 总额、分项明细。
type UsageResult struct {
	// CostAmount 上游成本总额。
	CostAmount decimal.Decimal `json:"cost_amount"`
	// SellAmount 内部结算总额 (已应用缓存折扣)。
	SellAmount decimal.Decimal `json:"sell_amount"`
	// CostItems 分项明细，每个 itemCode 的用量与金额。
	CostItems []CostItem `json:"cost_items"`
}

// CostItem 定义单个计费项的双轨金额明细。
type CostItem struct {
	// ItemCode 计费项编码。
	ItemCode string `json:"item_code"`
	// Usage 该计费项的原始用量 (Token 数、图片张数、视频秒数等)。
	Usage float64 `json:"usage"`
	// CostAmount 上游成本金额。
	CostAmount decimal.Decimal `json:"cost_amount"`
	// SellAmount 内部结算金额 (已应用缓存折扣)。
	SellAmount decimal.Decimal `json:"sell_amount"`
}

// TotalCost 对分项明细按 cost 求和，用于快速校验。
func (r *UsageResult) TotalCost() decimal.Decimal {
	sum := decimal.Zero
	for _, it := range r.CostItems {
		sum = sum.Add(it.CostAmount)
	}
	return sum
}

// TotalSell 对分项明细按 sell 求和，用于快速校验。
func (r *UsageResult) TotalSell() decimal.Decimal {
	sum := decimal.Zero
	for _, it := range r.CostItems {
		sum = sum.Add(it.SellAmount)
	}
	return sum
}
