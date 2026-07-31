package security

import (
	"context"
	"errors"
	"testing"
)

// ── CheckEgress 测试 ──────────────────────────────────────────────────────

// TestCheckEgress_InternalOnly_Blocked INTERNAL_ONLY 用户请求 external 模型
// 必须被阻断，返回 ErrEgressBlocked（D-CON-02：零外网流量）。
func TestCheckEgress_InternalOnly_Blocked(t *testing.T) {
	user := User{
		ID:           "user-internal",
		EgressPolicy: EgressPolicyInternalOnly,
	}
	model := Model{
		ID:           "gpt-4",
		Name:         "gpt-4",
		NetworkClass: NetworkExternal,
	}

	err := CheckEgress(context.Background(), user, model)
	if err == nil {
		t.Fatal("INTERNAL_ONLY 用户请求 external 模型，预期被阻断但放行")
	}
	if !errors.Is(err, ErrEgressBlocked) {
		t.Errorf("预期 ErrEgressBlocked，实际: %v", err)
	}
}

// TestCheckEgress_HybridAllowed HYBRID_ALLOWED 用户请求 external 模型
// 当前阶段放行（白名单校验尚未实现，留待阶段 D）。
func TestCheckEgress_HybridAllowed(t *testing.T) {
	user := User{
		ID:           "user-hybrid",
		EgressPolicy: EgressPolicyHybridAllowed,
	}
	model := Model{
		ID:           "claude-3",
		Name:         "claude-3",
		NetworkClass: NetworkExternal,
	}

	err := CheckEgress(context.Background(), user, model)
	if err != nil {
		t.Fatalf("HYBRID_ALLOWED 用户请求 external 模型，预期放行但被阻断: %v", err)
	}
}

// TestCheckEgress_OpenAll_Allowed OPEN_ALL 用户请求 external 模型全部放行。
func TestCheckEgress_OpenAll_Allowed(t *testing.T) {
	user := User{
		ID:           "user-open",
		EgressPolicy: EgressPolicyOpenAll,
	}
	model := Model{
		ID:           "gpt-4",
		Name:         "gpt-4",
		NetworkClass: NetworkExternal,
	}

	err := CheckEgress(context.Background(), user, model)
	if err != nil {
		t.Fatalf("OPEN_ALL 用户请求 external 模型，预期放行但被阻断: %v", err)
	}
}

// TestCheckEgress_InternalModel_AlwaysAllowed 内网模型对所有策略用户放行。
func TestCheckEgress_InternalModel_AlwaysAllowed(t *testing.T) {
	tests := []struct {
		name   string
		policy string
	}{
		{"INTERNAL_ONLY", EgressPolicyInternalOnly},
		{"HYBRID_ALLOWED", EgressPolicyHybridAllowed},
		{"OPEN_ALL", EgressPolicyOpenAll},
	}

	model := Model{
		ID:           "internal-llm",
		Name:         "internal-llm",
		NetworkClass: NetworkInternal,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := User{
				ID:           "user-" + tt.name,
				EgressPolicy: tt.policy,
			}
			err := CheckEgress(context.Background(), user, model)
			if err != nil {
				t.Errorf("%s 用户请求 internal 模型，预期放行但被阻断: %v", tt.policy, err)
			}
		})
	}
}

// TestCheckEgress_HybridInternalModel HYBRID_ALLOWED 用户请求 internal 模型放行。
func TestCheckEgress_HybridInternalModel(t *testing.T) {
	user := User{
		ID:           "user-hybrid-int",
		EgressPolicy: EgressPolicyHybridAllowed,
	}
	model := Model{
		ID:           "local-model",
		Name:         "local-model",
		NetworkClass: NetworkInternal,
	}

	err := CheckEgress(context.Background(), user, model)
	if err != nil {
		t.Fatalf("HYBRID_ALLOWED 用户请求 internal 模型，预期放行但被阻断: %v", err)
	}
}

// TestCheckEgress_UnknownPolicy 未知出网策略应保守拒绝。
func TestCheckEgress_UnknownPolicy(t *testing.T) {
	user := User{
		ID:           "user-unknown",
		EgressPolicy: "UNKNOWN_POLICY",
	}
	model := Model{
		ID:           "gpt-4",
		Name:         "gpt-4",
		NetworkClass: NetworkExternal,
	}

	err := CheckEgress(context.Background(), user, model)
	if err == nil {
		t.Fatal("未知出网策略，预期保守拒绝但放行")
	}
	if !errors.Is(err, ErrEgressBlocked) {
		t.Errorf("预期 ErrEgressBlocked，实际: %v", err)
	}
}

// ── HookBlockedError 测试 ─────────────────────────────────────────────────

// TestHookBlockedError_Error 验证 HookBlockedError 的错误信息格式。
func TestHookBlockedError_Error(t *testing.T) {
	err := &HookBlockedError{
		HookIndex: 2,
		Reason:    "内容安全检测到敏感词",
	}
	expected := "security: 钩子 #2 阻断: 内容安全检测到敏感词"
	if err.Error() != expected {
		t.Errorf("HookBlockedError.Error() = %q, want %q", err.Error(), expected)
	}
}
