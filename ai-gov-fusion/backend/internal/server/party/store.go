package party

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// ── Party CRUD ──────────────────────────────────────────────────────────

// CreateParty 插入新 Party 记录。
// 调用方应在调用前设置默认值（Status、时间戳）。返回已填充 ID 的已创建 Party。
func CreateParty(db *gorm.DB, p *Party) error {
	if p == nil {
		return errors.New("party: 不能创建 nil party")
	}
	if p.Type != TypeOrg && p.Type != TypeProject {
		return fmt.Errorf("party: 无效类型 %q，必须为 %q 或 %q", p.Type, TypeOrg, TypeProject)
	}
	if p.Name == "" {
		return errors.New("party: 名称为必填")
	}
	if p.Status == "" {
		p.Status = StatusActive
	}
	return db.Create(p).Error
}

// GetParty 按 ID 查询 Party。若 party 不存在则返回错误。
func GetParty(db *gorm.DB, id int64) (*Party, error) {
	var p Party
	if err := db.First(&p, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("party: party %d 未找到", id)
		}
		return nil, err
	}
	return &p, nil
}

// UpdatePartyStatus 更新 Party 状态。
// 状态转换不做验证——调用方必须强制执行合法的状态机转换
//（active → inactive → liquidated）。
func UpdatePartyStatus(db *gorm.DB, id int64, status string) error {
	if status != StatusActive && status != StatusInactive && status != StatusLiquidated {
		return fmt.Errorf("party: 无效状态 %q", status)
	}
	result := db.Model(&Party{}).Where("id = ?", id).Update("status", status)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("party: party %d 未找到", id)
	}
	return nil
}

// ListParties 列出所有 Party，可按类型筛选。传入空字符串返回所有 Party。
func ListParties(db *gorm.DB, partyType string) ([]*Party, error) {
	var parties []*Party
	q := db.Order("created_at ASC")
	if partyType != "" {
		q = q.Where("type = ?", partyType)
	}
	if err := q.Find(&parties).Error; err != nil {
		return nil, err
	}
	return parties, nil
}

// ── PartyEdge CRUD ──────────────────────────────────────────────────────

// CreateEdge 插入新关系边。
// 调用方应在调用前设置边类型并设置 AllowsFund。返回已填充 ID 的已创建边。
func CreateEdge(db *gorm.DB, e *PartyEdge) error {
	if e == nil {
		return errors.New("party: 不能创建 nil 边")
	}
	if e.SrcPartyID == e.DstPartyID {
		return errors.New("party: 不允许自引用边")
	}
	if !ValidEdgeType(e.EdgeType) {
		return fmt.Errorf("party: 无效边类型 %q", e.EdgeType)
	}
	return db.Create(e).Error
}

// DeleteEdge 删除一条关系边。调用方负责确认此边上无活跃资金通道或待处理划拨。
// 若边不存在则返回错误。
func DeleteEdge(db *gorm.DB, id int64) error {
	result := db.Delete(&PartyEdge{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("party: 边 %d 未找到", id)
	}
	return nil
}

// FindEdge 查找源→目标方向的边。
// 若无匹配边则返回 nil。当同对之间有多个不同类型的边时，返回找到的第一条。
func FindEdge(db *gorm.DB, srcID, dstID int64) (*PartyEdge, error) {
	var e PartyEdge
	if err := db.Where("src_party_id = ? AND dst_party_id = ?", srcID, dstID).First(&e).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // 无匹配边，非错误
		}
		return nil, err
	}
	return &e, nil
}

// ListEdges 返回连接到指定 Party 的所有边（作为源或目标）。
func ListEdges(db *gorm.DB, partyID int64) ([]*PartyEdge, error) {
	var edges []*PartyEdge
	if err := db.Where("src_party_id = ? OR dst_party_id = ?", partyID, partyID).
		Order("created_at ASC").Find(&edges).Error; err != nil {
		return nil, err
	}
	return edges, nil
}

// ── PartyMember CRUD ────────────────────────────────────────────────────

// CreateMember 插入新成员记录。
// 调用方应在调用前设置 PartyID、UserID 和 Role。
// 返回已填充 ID 的已创建成员。
func CreateMember(db *gorm.DB, m *PartyMember) error {
	if m == nil {
		return errors.New("party: 不能创建 nil 成员")
	}
	if m.PartyID == 0 {
		return errors.New("party: party_id 为必填")
	}
	if m.UserID == "" {
		return errors.New("party: user_id 为必填")
	}
	if m.Role == "" {
		m.Role = RoleMember
	}
	if m.Role != RoleLeader && m.Role != RoleMember && m.Role != RoleObserver {
		return fmt.Errorf("party: 无效角色 %q，必须为 %q、%q 或 %q",
			m.Role, RoleLeader, RoleMember, RoleObserver)
	}
	return db.Create(m).Error
}

// DeleteMember 按 ID 删除成员。若成员记录不存在则返回错误。
func DeleteMember(db *gorm.DB, id int64) error {
	result := db.Delete(&PartyMember{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("party: 成员 %d 未找到", id)
	}
	return nil
}

// ListMembers 返回指定 Party 的所有成员，按加入日期升序排列。
func ListMembers(db *gorm.DB, partyID int64) ([]*PartyMember, error) {
	var members []*PartyMember
	if err := db.Where("party_id = ?", partyID).
		Order("joined_at ASC").Find(&members).Error; err != nil {
		return nil, err
	}
	return members, nil
}

// GetMember 按主键检索单个 Party 成员。
func GetMember(db *gorm.DB, id int64) (*PartyMember, error) {
	var m PartyMember
	if err := db.First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("party: 成员 %d 未找到", id)
		}
		return nil, err
	}
	return &m, nil
}

// ── 迁移 ───────────────────────────────────────────────────────────

// Migrate 对三张 party 表（parties、party_edges、party_members）执行 GORM AutoMigrate。
// 若表不存在则创建，并添加缺失的列/索引。
// 由 store.go 编排层在阶段 1 迁移中调用。
func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&Party{}, &PartyEdge{}, &PartyMember{}); err != nil {
		return fmt.Errorf("party 迁移: %w", err)
	}
	return nil
}
