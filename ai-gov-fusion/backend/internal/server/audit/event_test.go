package audit

import (
	"context"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupDB 创建内存 SQLite 数据库并执行审计表迁移。
// 每个测试使用独立数据库，保证测试隔离性。
func setupDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存 SQLite 失败: %v", err)
	}
	if err := Migrate(db); err != nil {
		t.Fatalf("迁移审计表失败: %v", err)
	}
	return db
}

// newTestEvent 创建一条测试用的审计事件，填充默认有效值。
func newTestEvent(id, actorUserID, action, resourceType, resourceID string) *AuditEvent {
	return &AuditEvent{
		ID:           id,
		ActorUserID:  actorUserID,
		ActorName:    "测试用户",
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Status:       StatusSuccess,
		IP:           "127.0.0.1",
		UserAgent:    "go-test/1.0",
	}
}

// ── RecordEvent 测试 ──────────────────────────────────────────────────────

// TestRecordEvent_Success 成功写入审计事件——INSERT 后可通过 GetEvent 查询到
// 完整记录且字段一致。R6-17：配置变更操作必须提供 before/after 快照。
func TestRecordEvent_Success(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	event := newTestEvent(newUUID(), "admin-001", ActionGrantChange, "grant", "g-001")
	event.BeforeSnapshot = `{"grants": []}`
	event.AfterSnapshot = `{"grants": [{"role":"admin"}]}`

	if err := RecordEvent(ctx, db, event); err != nil {
		t.Fatalf("写入审计事件失败: %v", err)
	}

	got, err := GetEvent(ctx, db, event.ID)
	if err != nil {
		t.Fatalf("查询审计事件失败: %v", err)
	}
	if got == nil {
		t.Fatal("查询审计事件返回 nil")
	}
	if got.ActorUserID != "admin-001" {
		t.Errorf("actor_user_id = %q, want %q", got.ActorUserID, "admin-001")
	}
	if got.Action != ActionGrantChange {
		t.Errorf("action = %q, want %q", got.Action, ActionGrantChange)
	}
	if got.ResourceType != "grant" {
		t.Errorf("resource_type = %q, want %q", got.ResourceType, "grant")
	}
	if got.ResourceID != "g-001" {
		t.Errorf("resource_id = %q, want %q", got.ResourceID, "g-001")
	}
	if got.Status != StatusSuccess {
		t.Errorf("status = %q, want %q", got.Status, StatusSuccess)
	}
	if got.CreatedAt.IsZero() {
		t.Error("created_at 不应为零值")
	}
}

// TestRecordEvent_BeforeAfterSnapshot before/after 快照完整——
// 写入带变更前后快照的审计事件后，查询可获取完整快照 JSON。
func TestRecordEvent_BeforeAfterSnapshot(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	beforeJSON := `{"budget_cap": 10000, "currency": "CNY"}`
	afterJSON := `{"budget_cap": 20000, "currency": "CNY"}`

	event := newTestEvent(newUUID(), "admin-002", ActionBudgetCapChange, "account", "acct-001")
	event.BeforeSnapshot = beforeJSON
	event.AfterSnapshot = afterJSON
	event.Message = "预算帽从 10000 上调至 20000"

	if err := RecordEvent(ctx, db, event); err != nil {
		t.Fatalf("写入审计事件失败: %v", err)
	}

	got, err := GetEvent(ctx, db, event.ID)
	if err != nil {
		t.Fatalf("查询审计事件失败: %v", err)
	}
	if got == nil {
		t.Fatal("查询审计事件返回 nil")
	}
	if got.BeforeSnapshot != beforeJSON {
		t.Errorf("before_snapshot = %q, want %q", got.BeforeSnapshot, beforeJSON)
	}
	if got.AfterSnapshot != afterJSON {
		t.Errorf("after_snapshot = %q, want %q", got.AfterSnapshot, afterJSON)
	}
	if got.Message != "预算帽从 10000 上调至 20000" {
		t.Errorf("message = %q, want %q", got.Message, "预算帽从 10000 上调至 20000")
	}
}

// TestRecordEvent_Nil 传入 nil 审计事件应返回错误。
func TestRecordEvent_Nil(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	if err := RecordEvent(ctx, db, nil); err == nil {
		t.Fatal("预期传入 nil 审计事件返回错误，但成功")
	}
}

// TestRecordEvent_MissingRequired 缺少必填字段应返回错误。
func TestRecordEvent_MissingRequired(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	tests := []struct {
		name  string
		event AuditEvent
	}{
		{"缺少 action", AuditEvent{ID: newUUID(), ResourceType: "grant", ResourceID: "g-001"}},
		{"缺少 resource_type", AuditEvent{ID: newUUID(), Action: ActionGrantChange, ResourceID: "g-001"}},
		{"缺少 resource_id", AuditEvent{ID: newUUID(), Action: ActionGrantChange, ResourceType: "grant"}},
		{"缺少 id", AuditEvent{Action: ActionGrantChange, ResourceType: "grant", ResourceID: "g-001"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := RecordEvent(ctx, db, &tt.event); err == nil {
				t.Errorf("预期 %s 返回错误，但成功", tt.name)
			}
		})
	}
}

// TestRecordEvent_FailureStatus 记录一条失败状态的审计事件。
func TestRecordEvent_FailureStatus(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	event := newTestEvent(newUUID(), "admin-003", ActionAllocate, "account", "acct-002")
	event.Status = StatusFailure
	event.Message = "划拨失败：余额不足"

	if err := RecordEvent(ctx, db, event); err != nil {
		t.Fatalf("写入失败状态审计事件失败: %v", err)
	}

	got, err := GetEvent(ctx, db, event.ID)
	if err != nil {
		t.Fatalf("查询审计事件失败: %v", err)
	}
	if got.Status != StatusFailure {
		t.Errorf("status = %q, want %q", got.Status, StatusFailure)
	}
	if got.Message != "划拨失败：余额不足" {
		t.Errorf("message = %q, want %q", got.Message, "划拨失败：余额不足")
	}
}

// TestRecordEvent_MissingSnapshot R6-17 强校验：配置变更类操作缺少 before/after
// 快照时应返回错误。
func TestRecordEvent_MissingSnapshot(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	tests := []struct {
		name           string
		action         string
		beforeSnapshot string
		afterSnapshot  string
	}{
		{"变更操作缺 before", ActionGrantChange, "", `{"grants": [{"role":"admin"}]}`},
		{"变更操作缺 after", ActionGrantChange, `{"grants": []}`, ""},
		{"变更操作全缺", ActionGrantChange, "", ""},
		{"创建操作缺快照", ActionKeyCreate, "", ""},
		{"吊销操作缺快照", ActionKeyRevoke, "", ""},
		{"价目变更缺快照", ActionPriceChange, "", ""},
		{"预算帽变更缺快照", ActionBudgetCapChange, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := newTestEvent(newUUID(), "admin-001", tt.action, "test", "t-001")
			event.BeforeSnapshot = tt.beforeSnapshot
			event.AfterSnapshot = tt.afterSnapshot
			if err := RecordEvent(ctx, db, event); err == nil {
				t.Errorf("预期 %s 返回错误，但成功", tt.name)
			}
		})
	}

	// 资金运行时操作（非配置变更）不需要快照——应成功写入。
	t.Run("划拨操作无需快照", func(t *testing.T) {
		event := newTestEvent(newUUID(), "admin-001", ActionAllocate, "account", "acct-001")
		if err := RecordEvent(ctx, db, event); err != nil {
			t.Errorf("划拨操作不应要求快照: %v", err)
		}
	})
	t.Run("清算操作无需快照", func(t *testing.T) {
		event := newTestEvent(newUUID(), "admin-001", ActionLiquidate, "account", "acct-002")
		if err := RecordEvent(ctx, db, event); err != nil {
			t.Errorf("清算操作不应要求快照: %v", err)
		}
	})
}

// ── SearchEvents 测试 ─────────────────────────────────────────────────────

// TestSearchEvents_Filter 按操作者/类型/时间筛选——写入多条不同事件后，
// 按各维度筛选精确命中。
func TestSearchEvents_Filter(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	// 记录多条审计事件。R6-17：配置变更操作必须提供 before/after 快照。
	snapshotBefore := `{"grants": []}`
	snapshotAfter := `{"grants": [{"role":"admin"}]}`
	priceBefore := `{"price": 0.05}`
	priceAfter := `{"price": 0.10}`
	events := []*AuditEvent{
		newTestEvent(newUUID(), "admin-001", ActionGrantChange, "grant", "g-001"),
		newTestEvent(newUUID(), "admin-001", ActionPriceChange, "price", "p-001"),
		newTestEvent(newUUID(), "admin-002", ActionGrantChange, "grant", "g-002"),
		newTestEvent(newUUID(), "admin-003", ActionAllocate, "account", "acct-001"),
	}
	events[0].BeforeSnapshot = snapshotBefore
	events[0].AfterSnapshot = snapshotAfter
	events[1].BeforeSnapshot = priceBefore
	events[1].AfterSnapshot = priceAfter
	events[2].BeforeSnapshot = snapshotBefore
	events[2].AfterSnapshot = `{"grants": [{"role":"viewer"}]}`
	for _, e := range events {
		if err := RecordEvent(ctx, db, e); err != nil {
			t.Fatalf("写入审计事件失败: %v", err)
		}
	}

	// 按操作用户筛选——admin-001 应有 2 条。
	filter := AuditFilter{ActorUserID: "admin-001", Limit: 50}
	results, total, err := SearchEvents(ctx, db, filter)
	if err != nil {
		t.Fatalf("按操作用户筛选失败: %v", err)
	}
	if total != 2 {
		t.Errorf("admin-001 应有 2 条事件，实际 %d 条", total)
	}
	for _, r := range results {
		if r.ActorUserID != "admin-001" {
			t.Errorf("预期 actor_user_id = admin-001，但出现 %q", r.ActorUserID)
		}
	}

	// 按操作类型筛选——grant.change 应有 2 条（admin-001 和 admin-002）。
	filter = AuditFilter{Action: ActionGrantChange, Limit: 50}
	results, total, err = SearchEvents(ctx, db, filter)
	if err != nil {
		t.Fatalf("按操作类型筛选失败: %v", err)
	}
	if total != 2 {
		t.Errorf("grant.change 应有 2 条事件，实际 %d 条", total)
	}
	for _, r := range results {
		if r.Action != ActionGrantChange {
			t.Errorf("预期 action = grant.change，但出现 %q", r.Action)
		}
	}

	// 按资源类型+ID 筛选——grant/g-001 应有 1 条。
	filter = AuditFilter{ResourceType: "grant", ResourceID: "g-001", Limit: 50}
	results, total, err = SearchEvents(ctx, db, filter)
	if err != nil {
		t.Fatalf("按资源类型筛选失败: %v", err)
	}
	if total != 1 {
		t.Errorf("grant/g-001 应有 1 条事件，实际 %d 条", total)
	}
}

// TestSearchEvents_TimeRange 按时间区间筛选——写入事件后按时间范围过滤。
// SQLite 的时间戳精度为秒级，因此使用 Truncate(time.Second) 对齐比较。
func TestSearchEvents_TimeRange(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	// 先用较大时间缓冲区写入事件，再通过 GetEvent 获取实际的 CreatedAt 做精确断言。
	// R6-17：配置变更操作必须提供 before/after 快照。
	event := newTestEvent(newUUID(), "admin-001", ActionGrantChange, "grant", "g-001")
	event.BeforeSnapshot = `{"grants": []}`
	event.AfterSnapshot = `{"grants": [{"role":"admin"}]}`
	if err := RecordEvent(ctx, db, event); err != nil {
		t.Fatalf("写入审计事件失败: %v", err)
	}

	// 通过 GetEvent 获取实际写入的 created_at（SQLite 精度可能截断纳秒）。
	got, err := GetEvent(ctx, db, event.ID)
	if err != nil {
		t.Fatalf("查询审计事件失败: %v", err)
	}
	actualCreatedAt := got.CreatedAt.Truncate(time.Second)

	// 筛选时间范围包含事件创建时间。
	start := actualCreatedAt.Add(-time.Hour)
	end := actualCreatedAt.Add(time.Hour)
	filter := AuditFilter{StartTime: &start, EndTime: &end, Limit: 50}
	_, total, err := SearchEvents(ctx, db, filter)
	if err != nil {
		t.Fatalf("按时间筛选失败: %v", err)
	}
	if total != 1 {
		t.Errorf("时间范围内应有 1 条事件，实际 %d 条", total)
	}

	// 筛选时间范围不包含事件创建时间（未来 1 小时后）。
	future := actualCreatedAt.Add(time.Hour)
	future2 := actualCreatedAt.Add(2 * time.Hour)
	filter = AuditFilter{StartTime: &future, EndTime: &future2, Limit: 50}
	_, total, err = SearchEvents(ctx, db, filter)
	if err != nil {
		t.Fatalf("按未来时间筛选失败: %v", err)
	}
	if total != 0 {
		t.Errorf("未来时间范围内应有 0 条事件，实际 %d 条", total)
	}
}

// TestSearchEvents_LimitCap 上限截断——limit 超过 200 时自动截断为 200。
func TestSearchEvents_LimitCap(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	// 写入 3 条事件。R6-17：配置变更操作必须提供 before/after 快照。
	for i := 0; i < 3; i++ {
		event := newTestEvent(newUUID(), "admin-001", ActionGrantChange, "grant", "g-test")
		event.BeforeSnapshot = `{"grants": []}`
		event.AfterSnapshot = `{"grants": [{"role":"admin"}]}`
		if err := RecordEvent(ctx, db, event); err != nil {
			t.Fatalf("写入审计事件失败: %v", err)
		}
	}

	// limit 传入 500 应自动截断为 200（实际返回不超过 3 条）。
	filter := AuditFilter{Limit: 500}
	results, total, err := SearchEvents(ctx, db, filter)
	if err != nil {
		t.Fatalf("查询审计事件失败: %v", err)
	}
	if total != 3 {
		t.Errorf("应有 3 条事件，实际 %d 条", total)
	}
	// limit 应被截断为 200——但 3 < 200，所以返回 3 条。
	if len(results) != 3 {
		t.Errorf("预期返回 3 条结果，实际 %d 条", len(results))
	}
}

// TestGetEvent_NotFound 查询不存在的审计事件应返回 nil, nil。
func TestGetEvent_NotFound(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	got, err := GetEvent(ctx, db, "nonexistent-id")
	if err != nil {
		t.Fatalf("查询不存在的事件返回错误: %v", err)
	}
	if got != nil {
		t.Fatal("查询不存在的事件应返回 nil，但返回了非 nil")
	}
}

// TestGetEvent_EmptyID 传入空 ID 应返回错误。
func TestGetEvent_EmptyID(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	_, err := GetEvent(ctx, db, "")
	if err == nil {
		t.Fatal("预期传入空 ID 返回错误，但成功")
	}
}
