package party

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// ── Party CRUD ──────────────────────────────────────────────────────────

// CreateParty 插入新 Party 记录。
// defaults (Status, timestamps) before calling. Returns the created party
// with its assigned ID populated.
func CreateParty(db *gorm.DB, p *Party) error {
	if p == nil {
		return errors.New("party: cannot create nil party")
	}
	if p.Type != TypeOrg && p.Type != TypeProject {
		return fmt.Errorf("party: invalid type %q, must be %q or %q", p.Type, TypeOrg, TypeProject)
	}
	if p.Name == "" {
		return errors.New("party: name is required")
	}
	if p.Status == "" {
		p.Status = StatusActive
	}
	return db.Create(p).Error
}

// GetParty 按 ID 查询 Party。
// if the party is not found.
func GetParty(db *gorm.DB, id int64) (*Party, error) {
	var p Party
	if err := db.First(&p, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("party: party %d not found", id)
		}
		return nil, err
	}
	return &p, nil
}

// UpdatePartyStatus 更新 Party 状态。
// The status transition is not validated — callers must enforce valid
// state machine transitions (active → inactive → liquidated).
func UpdatePartyStatus(db *gorm.DB, id int64, status string) error {
	if status != StatusActive && status != StatusInactive && status != StatusLiquidated {
		return fmt.Errorf("party: invalid status %q", status)
	}
	result := db.Model(&Party{}).Where("id = ?", id).Update("status", status)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("party: party %d not found", id)
	}
	return nil
}

// ListParties 列出所有 Party，可按类型筛选。
// empty string for partyType to return all parties.
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
// type and set AllowsFund before calling. Returns the created edge with
// its assigned ID populated.
func CreateEdge(db *gorm.DB, e *PartyEdge) error {
	if e == nil {
		return errors.New("party: cannot create nil edge")
	}
	if e.SrcPartyID == e.DstPartyID {
		return errors.New("party: self-referencing edge is not allowed")
	}
	if !ValidEdgeType(e.EdgeType) {
		return fmt.Errorf("party: invalid edge type %q", e.EdgeType)
	}
	return db.Create(e).Error
}

// DeleteEdge 删除一条关系边。调用方负责确认此边上无活跃资金通道或待处理划拨。
// the edge does not exist.
func DeleteEdge(db *gorm.DB, id int64) error {
	result := db.Delete(&PartyEdge{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("party: edge %d not found", id)
	}
	return nil
}

// FindEdge 查找源→目标方向的边。
// if no matching edge exists. When multiple edges of different types exist
// between the same pair, the first one found is returned.
func FindEdge(db *gorm.DB, srcID, dstID int64) (*PartyEdge, error) {
	var e PartyEdge
	if err := db.Where("src_party_id = ? AND dst_party_id = ?", srcID, dstID).First(&e).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // no edge exists, not an error
		}
		return nil, err
	}
	return &e, nil
}

// ListEdges 返回连接到指定 Party 的所有边。
// source or destination.
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
// PartyID, UserID, and Role before calling. Returns the created member
// with its assigned ID populated.
func CreateMember(db *gorm.DB, m *PartyMember) error {
	if m == nil {
		return errors.New("party: cannot create nil member")
	}
	if m.PartyID == 0 {
		return errors.New("party: party_id is required")
	}
	if m.UserID == "" {
		return errors.New("party: user_id is required")
	}
	if m.Role == "" {
		m.Role = RoleMember
	}
	if m.Role != RoleLeader && m.Role != RoleMember && m.Role != RoleObserver {
		return fmt.Errorf("party: invalid role %q, must be %q, %q, or %q",
			m.Role, RoleLeader, RoleMember, RoleObserver)
	}
	return db.Create(m).Error
}

// DeleteMember 按 ID 删除成员。
// the member record does not exist.
func DeleteMember(db *gorm.DB, id int64) error {
	result := db.Delete(&PartyMember{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("party: member %d not found", id)
	}
	return nil
}

// ListMembers 返回指定 Party 的所有成员。
func ListMembers(db *gorm.DB, partyID int64) ([]*PartyMember, error) {
	var members []*PartyMember
	if err := db.Where("party_id = ?", partyID).
		Order("joined_at ASC").Find(&members).Error; err != nil {
		return nil, err
	}
	return members, nil
}

// GetMember retrieves a single party member by primary key.
func GetMember(db *gorm.DB, id int64) (*PartyMember, error) {
	var m PartyMember
	if err := db.First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("party: member %d not found", id)
		}
		return nil, err
	}
	return &m, nil
}

// ── Migration ───────────────────────────────────────────────────────────

// Migrate performs GORM AutoMigrate for the three party tables (parties,
// party_edges, party_members). It creates tables if they do not exist and
// adds missing columns/indexes. Called from the store.go orchestration layer
// during Phase 1 migrations.
func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&Party{}, &PartyEdge{}, &PartyMember{}); err != nil {
		return fmt.Errorf("party migration: %w", err)
	}
	return nil
}
