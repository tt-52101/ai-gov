package audit

import (
	"fmt"

	"gorm.io/gorm"
)

// Migrate 执行 GORM AutoMigrate，确保 audit_events 和 audit_chain_anchors
// 两张表存在且列定义与模型一致。若表不存在则创建；若模型新增列或索引则自动添加。
//
// 本函数由外部编排层（store.go）在阶段 B 迁移中调用。
//
// 注意：AutoMigrate 不删除已有列，也不修改列类型——需要变更列类型时应在
// 独立迁移脚本中处理。
func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&AuditEvent{}, &AuditChainAnchor{}); err != nil {
		return fmt.Errorf("audit: 迁移失败: %w", err)
	}
	return nil
}
