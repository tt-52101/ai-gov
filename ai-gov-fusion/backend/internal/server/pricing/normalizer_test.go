package pricing

import (
	"testing"
)

// ── TestNormalizeOpenAI_Standard ──

func TestNormalizeOpenAI_Standard(t *testing.T) {
	raw := map[string]interface{}{
		"prompt_tokens":     float64(5000),
		"completion_tokens": float64(1500),
		"completion_tokens_details": map[string]interface{}{
			"reasoning_tokens": float64(200),
		},
		"prompt_tokens_details": map[string]interface{}{
			"cached_tokens": float64(1000),
		},
	}

	usage, incomplete, err := NormalizeOpenAI(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if incomplete {
		t.Error("expected complete usage, got incomplete")
	}

	if usage[ItemCodePromptTokens] != 5000 {
		t.Errorf("prompt_tokens: expected 5000, got %f", usage[ItemCodePromptTokens])
	}
	if usage[ItemCodeCompletionTokens] != 1500 {
		t.Errorf("completion_tokens: expected 1500, got %f", usage[ItemCodeCompletionTokens])
	}
	if usage[ItemCodeCompletionReasoningTokens] != 200 {
		t.Errorf("reasoning_tokens: expected 200, got %f", usage[ItemCodeCompletionReasoningTokens])
	}
	if usage[ItemCodePromptCachedTokens] != 1000 {
		t.Errorf("cached_tokens: expected 1000, got %f", usage[ItemCodePromptCachedTokens])
	}
}

// ── TestNormalizeOpenAI_Incomplete ──

func TestNormalizeOpenAI_Incomplete(t *testing.T) {
	raw := map[string]interface{}{
		"completion_tokens": float64(800),
	}

	usage, incomplete, err := NormalizeOpenAI(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !incomplete {
		t.Error("expected incomplete=true for missing prompt_tokens")
	}
	if usage[ItemCodePromptTokens] != 0 {
		t.Errorf("prompt_tokens: expected 0, got %f", usage[ItemCodePromptTokens])
	}
	if usage[ItemCodeCompletionTokens] != 800 {
		t.Errorf("completion_tokens: expected 800, got %f", usage[ItemCodeCompletionTokens])
	}
}

// ── TestNormalizeAnthropic ──

func TestNormalizeAnthropic_Standard(t *testing.T) {
	raw := map[string]interface{}{
		"input_tokens":                float64(3000),
		"output_tokens":               float64(1200),
		"cache_read_input_tokens":     float64(500),
		"cache_creation_input_tokens": float64(100),
	}

	usage, incomplete, err := NormalizeAnthropic(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if incomplete {
		t.Error("expected complete usage, got incomplete")
	}

	// prompt_tokens = input_tokens - cache_read = 3000 - 500 = 2500
	if usage[ItemCodePromptTokens] != 2500 {
		t.Errorf("prompt_tokens: expected 2500, got %f", usage[ItemCodePromptTokens])
	}
	if usage[ItemCodeCompletionTokens] != 1200 {
		t.Errorf("completion_tokens: expected 1200, got %f", usage[ItemCodeCompletionTokens])
	}
	if usage[ItemCodePromptCachedTokens] != 500 {
		t.Errorf("cached_tokens: expected 500, got %f", usage[ItemCodePromptCachedTokens])
	}
	if usage[ItemCodePromptWriteCachedTokens] != 100 {
		t.Errorf("write_cached_tokens: expected 100, got %f", usage[ItemCodePromptWriteCachedTokens])
	}
}

// ── TestNormalizeUsage_UnknownProvider ──

func TestNormalizeUsage_UnknownProvider(t *testing.T) {
	_, _, err := NormalizeUsage("unknown-provider", nil)
	if err == nil {
		t.Error("expected error for unknown provider")
	}
}

// ── TestIsCachedItemCode ──

func TestIsCachedItemCode(t *testing.T) {
	if !IsCachedItemCode(ItemCodePromptCachedTokens) {
		t.Error("prompt_cached_tokens should be cached")
	}
	if !IsCachedItemCode(ItemCodePromptWriteCachedTokens) {
		t.Error("prompt_write_cached_tokens should be cached")
	}
	if IsCachedItemCode(ItemCodePromptTokens) {
		t.Error("prompt_tokens should NOT be cached")
	}
	if IsCachedItemCode(ItemCodeCompletionTokens) {
		t.Error("completion_tokens should NOT be cached")
	}
}

// ── TestParsePriceJSON ──

func TestParsePriceJSON(t *testing.T) {
	mp := ModelPrice{
		ID:          "test",
		PriceJSONStr: `{"items":[{"itemCode":"prompt_tokens","cost":{"mode":"usage_per_unit","rate":"0.002"},"sell":{"mode":"usage_per_unit","rate":"0.003"}}]}`,
	}

	pj, err := mp.ParsePriceJSON()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(pj.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(pj.Items))
	}
	if pj.Items[0].ItemCode != ItemCodePromptTokens {
		t.Errorf("expected prompt_tokens, got %q", pj.Items[0].ItemCode)
	}
}

// ── TestFindItem ──

func TestFindItem(t *testing.T) {
	pj := &PriceJSON{
		Items: []PriceItem{
			{ItemCode: ItemCodePromptTokens},
			{ItemCode: ItemCodeCompletionTokens},
		},
	}

	if item := pj.FindItem(ItemCodePromptTokens); item == nil {
		t.Error("expected to find prompt_tokens")
	}
	if item := pj.FindItem("non_existent"); item != nil {
		t.Error("expected nil for non-existent item")
	}
}
