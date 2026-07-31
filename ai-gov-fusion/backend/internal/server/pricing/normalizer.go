package pricing

import (
	"fmt"
	"strconv"
	"strings"
)

// ── 用量规范化 (PRD §4.5) ──

// NormalizeUsage 将上游供应商返回的用量原始响应映射为内部 10 项 itemCode 的标准用量字典。
//
// provider 参数用于选择对应的规范化器 (如 "openai"、"anthropic")。
// rawResponse 是上游返回的 usage 块 JSON 已解析为 map[string]interface{}。
//
// 返回值:
//   - map[string]float64: itemCode → 用量的标准化字典，缺失字段默认 0
//   - bool: 只要有任一可选字段缺失即置为 true (UsageIncomplete)
//   - error: 仅当 provider 未知或 rawResponse 类型错误时返回
//
// 安全约束: NEVER 伪造缓存/隐藏详情字段；缺失记 0 + UsageIncomplete。
func NormalizeUsage(provider string, rawResponse map[string]interface{}) (map[string]float64, bool, error) {
	switch strings.ToLower(provider) {
	case "openai":
		return NormalizeOpenAI(rawResponse)
	case "anthropic":
		return NormalizeAnthropic(rawResponse)
	default:
		return nil, false, fmt.Errorf("pricing: unknown provider %q for usage normalization", provider)
	}
}

// NormalizeOpenAI 将 OpenAI 兼容响应的 usage 块标准化为内部 itemCode 字典。
//
// 处理以下 OpenAI 特征字段:
//   - prompt_tokens          → prompt_tokens
//   - completion_tokens      → completion_tokens
//   - completion_tokens_details.reasoning_tokens → completion_reasoning_tokens
//   - prompt_tokens_details.cached_tokens        → prompt_cached_tokens
//   - prompt_tokens_details.cached_write_tokens  → prompt_write_cached_tokens
//   - prompt_audio_tokens    → prompt_audio_tokens (如有)
//   - completion_audio_tokens → completion_audio_tokens (如有)
//
// 图片和视频用量通常不在 chat/completions usage 中返回，默认取 0。
func NormalizeOpenAI(raw map[string]interface{}) (map[string]float64, bool, error) {
	result := make(map[string]float64)
	for _, code := range AllItemCodes() {
		result[code] = 0
	}
	incomplete := false

	// 直接字段: prompt_tokens, completion_tokens
	if v, ok := getFloat(raw, "prompt_tokens"); ok {
		result[ItemCodePromptTokens] = v
	} else {
		incomplete = true
	}
	if v, ok := getFloat(raw, "completion_tokens"); ok {
		result[ItemCodeCompletionTokens] = v
	} else {
		incomplete = true
	}

	// completion_tokens_details: 嵌套对象，含 reasoning_tokens
	if details, ok := raw["completion_tokens_details"].(map[string]interface{}); ok {
		if v, ok2 := getFloat(details, "reasoning_tokens"); ok2 {
			result[ItemCodeCompletionReasoningTokens] = v
		}
		// 仅当上游明确提供了 reasoning 字段才认为"不缺失"
		// 如果整个 details 对象不存在，incomplete 保持已有值
	}

	// prompt_tokens_details: 嵌套对象，含 cached_tokens 和 cached_write_tokens
	if details, ok := raw["prompt_tokens_details"].(map[string]interface{}); ok {
		// cached_tokens: 缓存读命中 Token
		if v, ok2 := getFloat(details, "cached_tokens"); ok2 {
			result[ItemCodePromptCachedTokens] = v
		}
		// cached_write_tokens: 缓存写 Token (部分供应商支持)
		if v, ok2 := getFloat(details, "cached_write_tokens"); ok2 {
			result[ItemCodePromptWriteCachedTokens] = v
		} else if v, ok2 := getFloat(details, "prompt_write_cached_tokens"); ok2 {
			result[ItemCodePromptWriteCachedTokens] = v
		}
	}

	// 音频 Tokens: 部分 OpenAI 兼容实现会在 usage 中直接返回
	if v, ok := getFloat(raw, "prompt_audio_tokens"); ok {
		result[ItemCodePromptAudioTokens] = v
	}
	if v, ok := getFloat(raw, "completion_audio_tokens"); ok {
		result[ItemCodeCompletionAudioTokens] = v
	}

	// 图片和视频用量不在标准 chat completion usage 中
	// 保持默认 0，不标记 incomplete (这些字段本就不在此协议中出现)

	return result, incomplete, nil
}

// NormalizeAnthropic 将 Anthropic Messages API 响应的 usage 块标准化为内部 itemCode 字典。
//
// Anthropic 的 usage 块结构:
//
//	{
//	  "input_tokens": 1000,
//	  "output_tokens": 500,
//	  "cache_creation_input_tokens": 0,
//	  "cache_read_input_tokens": 200,
//	  "server_tools_use": {...}
//	}
//
// 映射规则:
//   - input_tokens - cache_read_input_tokens → prompt_tokens (排除缓存读)
//   - output_tokens → completion_tokens
//   - cache_read_input_tokens → prompt_cached_tokens
//   - cache_creation_input_tokens → prompt_write_cached_tokens
func NormalizeAnthropic(raw map[string]interface{}) (map[string]float64, bool, error) {
	result := make(map[string]float64)
	for _, code := range AllItemCodes() {
		result[code] = 0
	}
	incomplete := false

	inputTokens, hasInput := getFloat(raw, "input_tokens")
	outputTokens, hasOutput := getFloat(raw, "output_tokens")
	cacheReadTokens, _ := getFloat(raw, "cache_read_input_tokens")
	cacheCreateTokens, _ := getFloat(raw, "cache_creation_input_tokens")

	if hasInput {
		// 缓存的 input token 不计入 prompt_tokens (避免双计)
		result[ItemCodePromptTokens] = inputTokens - cacheReadTokens
		if result[ItemCodePromptTokens] < 0 {
			result[ItemCodePromptTokens] = 0
		}
	} else {
		incomplete = true
	}

	if hasOutput {
		result[ItemCodeCompletionTokens] = outputTokens
	} else {
		incomplete = true
	}

	// 缓存类: 单独分账
	result[ItemCodePromptCachedTokens] = cacheReadTokens
	result[ItemCodePromptWriteCachedTokens] = cacheCreateTokens

	// Anthropic 暂不返回 reasoning / audio / image / video 用量
	return result, incomplete, nil
}

// getFloat 从 map[string]interface{} 中提取 float64 值。
// 支持 JSON 反序列化后的 float64、json.Number (通过 string 转换)、以及 int64。
// 返回提取的值和是否找到。
func getFloat(m map[string]interface{}, key string) (float64, bool) {
	v, ok := m[key]
	if !ok {
		return 0, false
	}
	switch val := v.(type) {
	case float64:
		return val, true
	case jsonNumber:
		f, err := val.Float64()
		if err != nil {
			return 0, false
		}
		return f, true
	case int64:
		return float64(val), true
	case int:
		return float64(val), true
	case string:
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return 0, false
		}
		return f, true
	default:
		return 0, false
	}
}

// jsonNumber 接口用于支持 encoding/json 的 Number 类型，避免强制类型断言。
type jsonNumber interface {
	Float64() (float64, error)
}
