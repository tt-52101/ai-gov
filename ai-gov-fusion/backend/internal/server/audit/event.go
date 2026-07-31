package audit

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// newUUID 生成 UUID v4 字符串，用作审计事件和锚点记录的主键。
// 使用 crypto/rand 确保密码学强度的随机性。
func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	// UUID v4 格式: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// ── 写入操作 ──────────────────────────────────────────────────────────────

// RecordEvent 记录审计事件——仅执行 INSERT，永远不 UPDATE 或 DELETE。
//
// 铁律（AU-CON-01 / D-CON-04）：
//   - 审计事件一旦写入即不可变更或删除。
//   - before_snapshot 和 after_snapshot 为 JSON 格式的变更前后镜像——
//     对于配置变更类操作（如 δ 修改、价目变更、预算帽调整），必须同时提供两者。
//   - 对于只读查询记录或状态变更日志，快照字段可为空。
//
// 副作用：向 audit_events 表插入一行。
// 并发安全：各调用方独立插入，无锁竞争——数据库级唯一约束处理冲突。
func RecordEvent(ctx context.Context, db *gorm.DB, event *AuditEvent) error {
	if event == nil {
		return errors.New("audit: 审计事件不能为空")
	}
	if event.Action == "" {
		return errors.New("audit: action 为必填字段")
	}
	if event.ResourceType == "" {
		return errors.New("audit: resource_type 为必填字段")
	}
	if event.ResourceID == "" {
		return errors.New("audit: resource_id 为必填字段")
	}
	if event.ID == "" {
		return errors.New("audit: id 为必填字段（由调用方生成 UUID）")
	}

	// 仅 INSERT——不存在 UPDATE 或 DELETE 路径（AU-CON-01）。
	if err := db.WithContext(ctx).Create(event).Error; err != nil {
		return fmt.Errorf("audit: 写入审计事件失败: %w", err)
	}

	return nil
}

// ── 查询操作 ──────────────────────────────────────────────────────────────

// SearchEvents 按操作者、操作类型、时间、资源等多维度检索审计事件。
//
// 参数:
//   - filter: 多维度过滤条件，空字段表示不限制该维度。
//   - filter.Limit 上限 200（若传入超过此值则自动截断）。
//
// 返回值:
//   - 事件列表（按 created_at 降序）。
//   - 符合过滤条件的总记录数（不受分页限制）。
//   - 数据库错误。
//
// 注意：本方法仅执行 SELECT，不修改任何数据。
func SearchEvents(ctx context.Context, db *gorm.DB, filter AuditFilter) ([]*AuditEvent, int64, error) {
	if filter.Limit <= 0 || filter.Limit > 200 {
		filter.Limit = 200
	}

	q := db.WithContext(ctx).Model(&AuditEvent{})

	// 按操作用户筛选。
	if filter.ActorUserID != "" {
		q = q.Where("actor_user_id = ?", filter.ActorUserID)
	}

	// 按操作类型筛选。
	if filter.Action != "" {
		q = q.Where("action = ?", filter.Action)
	}

	// 按资源类型和 ID 筛选。
	if filter.ResourceType != "" {
		q = q.Where("resource_type = ?", filter.ResourceType)
	}
	if filter.ResourceID != "" {
		q = q.Where("resource_id = ?", filter.ResourceID)
	}

	// 按操作状态筛选。
	if filter.Status != "" {
		q = q.Where("status = ?", filter.Status)
	}

	// 按时间区间筛选。
	if filter.StartTime != nil {
		q = q.Where("created_at >= ?", *filter.StartTime)
	}
	if filter.EndTime != nil {
		q = q.Where("created_at <= ?", *filter.EndTime)
	}

	// 先统计总数（不受分页限制）。
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("audit: 统计审计事件失败: %w", err)
	}

	// 分页查询，按时间降序。
	var events []*AuditEvent
	if err := q.Order("created_at DESC").
		Offset(filter.Offset).
		Limit(filter.Limit).
		Find(&events).Error; err != nil {
		return nil, 0, fmt.Errorf("audit: 查询审计事件失败: %w", err)
	}

	return events, total, nil
}

// GetEvent 获取单条审计事件详情，包含 before/after 快照对比数据。
//
// 参数:
//   - id: 审计事件主键（UUID）。
//
// 返回值:
//   - 完整审计事件（含 JSON 快照字段）。
//   - 若未找到记录，返回 nil, nil（不抛错误）。
//   - 数据库错误。
func GetEvent(ctx context.Context, db *gorm.DB, id string) (*AuditEvent, error) {
	if id == "" {
		return nil, errors.New("audit: id 为必填字段")
	}

	var event AuditEvent
	if err := db.WithContext(ctx).First(&event, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("audit: 查询审计事件 %s 失败: %w", id, err)
	}

	return &event, nil
}
