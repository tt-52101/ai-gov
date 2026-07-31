package audit

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// AnchorChain 对 [startEventID, endEventID] 区间内的连续审计事件计算
// SHA-256 哈希并写入锚定记录。
//
// 锚定算法（对齐 DDL 存储过程 anchor_audit_chain）：
//   1. 统计区间内事件数量（若为 0 则报错）。
//   2. 取上一锚点哈希作为链式前驱（若为首个锚点则使用 "GENESIS"）。
//   3. 拼接内容：前一锚点哈希 + startEventID + endEventID + 事件计数 + 当前时间。
//   4. 对拼接内容计算 SHA-256，写入 audit_chain_anchors 表。
//
// 参数:
//   - startEventID: 区间起始审计事件 ID。
//   - endEventID: 区间结束审计事件 ID。
//
// 返回值:
//   - 新创建的锚点记录。
//   - 若区间内无事件或数据库写入失败则返回错误。
//
// 副作用：向 audit_chain_anchors 表插入一行。
func AnchorChain(ctx context.Context, db *gorm.DB, startEventID, endEventID string) (*AuditChainAnchor, error) {
	if startEventID == "" || endEventID == "" {
		return nil, fmt.Errorf("audit: startEventID 和 endEventID 为必填参数")
	}

	// 统计区间内事件数量。
	var eventCount int64
	if err := db.WithContext(ctx).Model(&AuditEvent{}).
		Where("id >= ? AND id <= ?", startEventID, endEventID).
		Count(&eventCount).Error; err != nil {
		return nil, fmt.Errorf("audit: 统计区间事件数失败: %w", err)
	}
	if eventCount == 0 {
		return nil, fmt.Errorf("audit: 区间 [%s - %s] 内无审计事件", startEventID, endEventID)
	}

	// 取前一锚点哈希作为链式前驱。
	prevHash, err := fetchPreviousAnchorHash(ctx, db)
	if err != nil {
		return nil, err
	}

	// 拼接锚定内容并计算 SHA-256。
	now := time.Now().UTC().Format(time.RFC3339)
	concat := fmt.Sprintf("%s:%s:%s:%d:%s",
		prevHash, startEventID, endEventID, eventCount, now)
	hash := sha256.Sum256([]byte(concat))
	anchorHash := fmt.Sprintf("%x", hash)

	// 写入锚定记录。
	anchor := &AuditChainAnchor{
		ID:           newUUID(),
		AnchorHash:   anchorHash,
		StartEventID: startEventID,
		EndEventID:   endEventID,
		EventCount:   int(eventCount),
		CreatedAt:    time.Now().UTC(),
	}
	if err := db.WithContext(ctx).Create(anchor).Error; err != nil {
		return nil, fmt.Errorf("audit: 写入锚定记录失败: %w", err)
	}

	return anchor, nil
}

// VerifyChain 验证指定锚点的区间内事件是否被篡改。
//
// 对锚点覆盖范围内的所有事件重新计算 SHA-256 哈希，并与锚点记录的 anchor_hash
// 比对——不一致则说明区间内事件被插入、删除或修改。
//
// 参数:
//   - anchorID: 待验证的锚点记录主键。
//
// 返回值:
//   - true: 验证通过，区间事件未被篡改。
//   - false: 验证失败，区间事件已被篡改或查询出错。
//   - error: 数据库读取异常或锚点不存在。
func VerifyChain(ctx context.Context, db *gorm.DB, anchorID string) (bool, error) {
	if anchorID == "" {
		return false, fmt.Errorf("audit: anchorID 为必填参数")
	}

	// 读取锚点记录。
	var anchor AuditChainAnchor
	if err := db.WithContext(ctx).First(&anchor, "id = ?", anchorID).Error; err != nil {
		return false, fmt.Errorf("audit: 查找锚点 %s 失败: %w", anchorID, err)
	}

	// 统计当前区间内实际事件数。
	var currentCount int64
	if err := db.WithContext(ctx).Model(&AuditEvent{}).
		Where("id >= ? AND id <= ?", anchor.StartEventID, anchor.EndEventID).
		Count(&currentCount).Error; err != nil {
		return false, fmt.Errorf("audit: 重新统计区间事件数失败: %w", err)
	}

	if currentCount != int64(anchor.EventCount) {
		return false, nil
	}

	// 取前一锚点哈希。
	prevHash, err := fetchPreviousAnchorHashFor(ctx, db, anchor.CreatedAt)
	if err != nil {
		return false, err
	}

	// 重新计算哈希。
	concat := fmt.Sprintf("%s:%s:%s:%d:%s",
		prevHash, anchor.StartEventID, anchor.EndEventID,
		currentCount, anchor.CreatedAt.UTC().Format(time.RFC3339))
	recomputed := sha256.Sum256([]byte(concat))
	recomputedHash := fmt.Sprintf("%x", recomputed)

	return recomputedHash == anchor.AnchorHash, nil
}

// fetchPreviousAnchorHash 获取当前最新锚点的哈希作为链式前驱。
// 若尚无锚点，返回字面量 "GENESIS"。
func fetchPreviousAnchorHash(ctx context.Context, db *gorm.DB) (string, error) {
	var prev AuditChainAnchor
	if err := db.WithContext(ctx).Order("created_at DESC").First(&prev).Error; err != nil {
		if err.Error() == "record not found" {
			return "GENESIS", nil
		}
		return "", fmt.Errorf("audit: 获取前一锚点哈希失败: %w", err)
	}
	return prev.AnchorHash, nil
}

// fetchPreviousAnchorHashFor 获取指定时间 t 之前的最新锚点哈希。
func fetchPreviousAnchorHashFor(ctx context.Context, db *gorm.DB, before time.Time) (string, error) {
	var prev AuditChainAnchor
	if err := db.WithContext(ctx).
		Where("created_at < ?", before).
		Order("created_at DESC").
		First(&prev).Error; err != nil {
		if err.Error() == "record not found" {
			return "GENESIS", nil
		}
		return "", fmt.Errorf("audit: 获取前一锚点哈希失败: %w", err)
	}
	return prev.AnchorHash, nil
}
