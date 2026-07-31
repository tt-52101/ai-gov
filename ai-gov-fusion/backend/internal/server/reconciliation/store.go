// Package reconciliation 持久层——GORM AutoMigrate + CRUD 骨架。
// 当前阶段（P0）仅提供建表和基础查询，具体业务逻辑延至阶段 D 实现。
package reconciliation

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// Migrate 执行 GORM AutoMigrate，确保 reconciliation_runs 表存在且
// 列定义与模型一致。若表不存在则创建；若模型新增列或索引则自动添加。
//
// 本函数由外部编排层（store.go）在阶段 B 迁移中调用。
//
// 注意：AutoMigrate 不删除已有列，也不修改列类型——需要变更列类型时应在
// 独立迁移脚本中处理。
func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&ReconciliationRun{}); err != nil {
		return fmt.Errorf("reconciliation: 迁移失败: %w", err)
	}
	return nil
}

// ── CRUD 操作 ─────────────────────────────────────────────────────────────────

// CreateRun 创建一次新的对账运行记录。
//
// 参数:
//   - run: 对账运行记录（RunID 由调用方生成，PeriodStart/PeriodEnd 为必填）。
//
// 阶段 D 注意事项：应校验同一账期不重复创建。
func CreateRun(ctx context.Context, db *gorm.DB, run *ReconciliationRun) error {
	if run == nil {
		return fmt.Errorf("reconciliation: 对账运行记录不能为空")
	}
	if run.RunID == "" {
		return fmt.Errorf("reconciliation: run_id 为必填字段")
	}
	if err := db.WithContext(ctx).Create(run).Error; err != nil {
		return fmt.Errorf("reconciliation: 创建对账运行失败: %w", err)
	}
	return nil
}

// GetRun 按主键查询对账运行记录。
//
// 返回值:
//   - 若未找到记录，返回 nil, nil（不抛错误）。
//   - 数据库错误。
func GetRun(ctx context.Context, db *gorm.DB, runID string) (*ReconciliationRun, error) {
	if runID == "" {
		return nil, fmt.Errorf("reconciliation: run_id 为必填字段")
	}

	var run ReconciliationRun
	if err := db.WithContext(ctx).First(&run, "run_id = ?", runID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("reconciliation: 查询对账运行 %s 失败: %w", runID, err)
	}
	return &run, nil
}

// ListRuns 按时间区间查询对账运行列表，按创建时间降序排列。
//
// 参数:
//   - limit: 每页最大条数，上限 200。
//   - offset: 分页偏移量。
//
// 阶段 D 扩展：可按 status、period_start/end 区间筛选。
func ListRuns(ctx context.Context, db *gorm.DB, limit, offset int) ([]*ReconciliationRun, int64, error) {
	if limit <= 0 || limit > 200 {
		limit = 200
	}

	q := db.WithContext(ctx).Model(&ReconciliationRun{})

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("reconciliation: 统计对账运行失败: %w", err)
	}

	var runs []*ReconciliationRun
	if err := q.Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&runs).Error; err != nil {
		return nil, 0, fmt.Errorf("reconciliation: 查询对账运行列表失败: %w", err)
	}

	return runs, total, nil
}

// UpdateRunStatus 更新对账运行的状态。阶段 D 扩展为支持更多字段更新。
func UpdateRunStatus(ctx context.Context, db *gorm.DB, runID, status string) error {
	if runID == "" {
		return fmt.Errorf("reconciliation: run_id 为必填字段")
	}
	if status == "" {
		return fmt.Errorf("reconciliation: status 为必填字段")
	}

	result := db.WithContext(ctx).
		Model(&ReconciliationRun{}).
		Where("run_id = ?", runID).
		Update("status", status)
	if result.Error != nil {
		return fmt.Errorf("reconciliation: 更新对账运行状态失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("reconciliation: 对账运行 %s 不存在", runID)
	}
	return nil
}
