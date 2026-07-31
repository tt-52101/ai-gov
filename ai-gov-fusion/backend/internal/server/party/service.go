package party

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"gorm.io/gorm"
)

// Service 提供 Party 管理的业务逻辑层。
// 封装 GORM 数据库句柄，暴露组织/项目创建、关系边管理、
// 资金划拨资格校验、成员管理等方法。
//
// 所有方法接受 context.Context 用于结构化日志和取消传播。
// 写数据库的方法不保证幂等——需要幂等的调用方应使用 fund 包的
// Idempotency-Key 机制处理资金级操作。
type Service struct {
	DB *gorm.DB
}

// NewService 使用给定数据库句柄创建新的 party Service。
// 调用方负责确保数据库连接有效且在调用服务方法前已完成迁移。
func NewService(db *gorm.DB) *Service {
	return &Service{DB: db}
}

// ── Party 管理 ────────────────────────────────────────────────────────────

// CreateParty 创建新的组织或项目主体。
// 项目无需指定父级（PRD §2.3：组织与项目平级）。
//
// 如果请求指定了 ParentPartyID，父级 Party 必须存在。
// 成功插入后，返回的 Party 的 ID 已被填充。
func (s *Service) CreateParty(ctx context.Context, req CreatePartyRequest) (*Party, error) {
	if req.Type != TypeOrg && req.Type != TypeProject {
		return nil, fmt.Errorf("party: invalid type %q, must be %q or %q", req.Type, TypeOrg, TypeProject)
	}
	if req.Name == "" {
		return nil, errors.New("party: name is required")
	}

		// 校验父级 Party 是否存在。
		if req.ParentPartyID != nil {
		if _, err := GetParty(s.DB, *req.ParentPartyID); err != nil {
			return nil, fmt.Errorf("party: parent party %d not found: %w", *req.ParentPartyID, err)
		}
	}

	p := &Party{
		Type:          req.Type,
		Name:          req.Name,
		Description:   req.Description,
		ParentPartyID: req.ParentPartyID,
		LeaderUserID:  req.LeaderUserID,
		CostCenter:    req.CostCenter,
		Status:        StatusActive,
		Metadata:      req.Metadata,
	}

	if err := CreateParty(s.DB, p); err != nil {
		slog.ErrorContext(ctx, "创建Party失败",
			"type", req.Type,
			"name", req.Name,
			"error", err,
		)
		return nil, fmt.Errorf("party: create failed: %w", err)
	}

	slog.InfoContext(ctx, "创建Party成功",
		"party_id", p.ID,
		"type", p.Type,
		"name", p.Name,
		"status", p.Status,
	)
	return p, nil
}

// GetParties 返回所有 Party，可按类型筛选。传入空字符串返回全部类型。
// string for partyType to return all parties regardless of type.
func (s *Service) GetParties(ctx context.Context, partyType string) ([]*Party, error) {
	parties, err := ListParties(s.DB, partyType)
	if err != nil {
		return nil, fmt.Errorf("party: list failed: %w", err)
	}
	return parties, nil
}

// ── 边管理 ────────────────────────────────────────────────────

// CreateEdge creates a typed relationship edge between two parties. It
// validates the edge type against the seven recognized types and
// automatically sets allows_fund=true for parent, sponsors, and allocates
// edges (per PRD §2.4 fund flow rules).
//
// Self-referencing edges (src == dst) are rejected. The source and
// destination parties must both exist.
func (s *Service) CreateEdge(ctx context.Context, req CreateEdgeRequest) (*PartyEdge, error) {
	if req.SrcPartyID == req.DstPartyID {
		return nil, errors.New("party: self-referencing edge is not allowed")
	}
	if !ValidEdgeType(req.EdgeType) {
		return nil, fmt.Errorf("party: invalid edge type %q", req.EdgeType)
	}

	// Verify both parties exist.
	if _, err := GetParty(s.DB, req.SrcPartyID); err != nil {
		return nil, fmt.Errorf("party: src party %d: %w", req.SrcPartyID, err)
	}
	if _, err := GetParty(s.DB, req.DstPartyID); err != nil {
		return nil, fmt.Errorf("party: dst party %d: %w", req.DstPartyID, err)
	}

	e := &PartyEdge{
		SrcPartyID: req.SrcPartyID,
		DstPartyID: req.DstPartyID,
		EdgeType:   req.EdgeType,
		AllowsFund: FundAutoAllowed(req.EdgeType),
	}

	if err := CreateEdge(s.DB, e); err != nil {
		slog.ErrorContext(ctx, "创建边失败",
			"src_party_id", req.SrcPartyID,
			"dst_party_id", req.DstPartyID,
			"edge_type", req.EdgeType,
			"error", err,
		)
		return nil, fmt.Errorf("party: create edge failed: %w", err)
	}

	slog.InfoContext(ctx, "创建边成功",
		"edge_id", e.ID,
		"src_party_id", e.SrcPartyID,
		"dst_party_id", e.DstPartyID,
		"edge_type", e.EdgeType,
		"allows_fund", e.AllowsFund,
	)
	return e, nil
}

// DeleteEdge 删除一条关系边。调用方负责确认此边上无活跃资金通道或待处理划拨。
// verifying that no active fund channels or pending allocations depend
// on this edge.
func (s *Service) DeleteEdge(ctx context.Context, edgeID int64) error {
	if err := DeleteEdge(s.DB, edgeID); err != nil {
		slog.ErrorContext(ctx, "删除边失败", "edge_id", edgeID, "error", err)
		return fmt.Errorf("party: delete edge failed: %w", err)
	}
	slog.InfoContext(ctx, "删除边成功", "edge_id", edgeID)
	return nil
}

// GetEdges 返回连接到指定 Party 的所有边（作为源或目标）。
// destination).
func (s *Service) GetEdges(ctx context.Context, partyID int64) ([]*PartyEdge, error) {
	edges, err := ListEdges(s.DB, partyID)
	if err != nil {
		return nil, fmt.Errorf("party: list edges failed: %w", err)
	}
	return edges, nil
}

// ── 资金划拨资格 ───────────────────────────────────────────

// CanAllocate checks whether a fund transfer is permitted from the source
// party's account to the destination party's account.
//
// Fund transfer is allowed when:
//   - A parent edge exists with src as parent and dst as child (downward only).
//   - A sponsors edge exists with src as sponsor and dst as sponsored
//     (sponsor→sponsored direction only).
//   - An allocates edge exists with src party allocating to dst (person account).
//
// Fund transfer is NOT allowed for owns, participates, merged_into, or
// split_from edges. Upward parent transfers (child→parent) are always denied.
// Transfers without any edge between the parties are denied.
func (s *Service) CanAllocate(ctx context.Context, srcPartyID, dstPartyID int64) (bool, error) {
	// Check all edges where src is the source and dst is the destination.
	edge, err := FindEdge(s.DB, srcPartyID, dstPartyID)
	if err != nil {
		return false, fmt.Errorf("party: check allocation failed: %w", err)
	}
	if edge == nil {
		return false, nil
	}

	// Only parent, sponsors, and allocates edges permit fund transfer,
	// and only in the forward (src→dst) direction for parent/sponsors.
	allowed := edge.AllowsFund && (edge.EdgeType == EdgeParent ||
		edge.EdgeType == EdgeSponsors ||
		edge.EdgeType == EdgeAllocates)

	slog.DebugContext(ctx, "资金划拨资格校验",
		"src_party_id", srcPartyID,
		"dst_party_id", dstPartyID,
		"edge_id", edge.ID,
		"edge_type", edge.EdgeType,
		"allows_fund", edge.AllowsFund,
		"result", allowed,
	)
	return allowed, nil
}

// ── 成员管理 ───────────────────────────────────────────────

// AddMember adds a user to a party with the specified role. The role
// defaults to "member" if not specified. Leader role is descriptive only
// and does not grant automatic privileges (A-CON-05).
//
// The party must exist. Duplicate (party_id, user_id) pairs are rejected
// by the database UNIQUE constraint.
func (s *Service) AddMember(ctx context.Context, req AddMemberRequest) (*PartyMember, error) {
	if req.PartyID == 0 {
		return nil, errors.New("party: party_id is required")
	}
	if req.UserID == "" {
		return nil, errors.New("party: user_id is required")
	}
	if req.Role == "" {
		req.Role = RoleMember
	}

	// Verify party exists.
	if _, err := GetParty(s.DB, req.PartyID); err != nil {
		return nil, fmt.Errorf("party: party %d not found: %w", req.PartyID, err)
	}

	m := &PartyMember{
		PartyID:   req.PartyID,
		UserID:    req.UserID,
		Role:      req.Role,
		IsPrimary: req.IsPrimary,
	}

	if err := CreateMember(s.DB, m); err != nil {
		slog.ErrorContext(ctx, "添加成员失败",
			"party_id", req.PartyID,
			"user_id", req.UserID,
			"role", req.Role,
			"error", err,
		)
		return nil, fmt.Errorf("party: add member failed: %w", err)
	}

	slog.InfoContext(ctx, "添加成员成功",
		"member_id", m.ID,
		"party_id", m.PartyID,
		"user_id", m.UserID,
		"role", m.Role,
	)
	return m, nil
}

// RemoveMember removes a user from a party by membership ID. The caller
// is responsible for ensuring the user has no active API keys bound to
// this party's account before removal (per API spec §2.9).
func (s *Service) RemoveMember(ctx context.Context, membershipID int64) error {
	if err := DeleteMember(s.DB, membershipID); err != nil {
		slog.ErrorContext(ctx, "移除成员失败", "member_id", membershipID, "error", err)
		return fmt.Errorf("party: remove member failed: %w", err)
	}
	slog.InfoContext(ctx, "移除成员成功", "member_id", membershipID)
	return nil
}

// GetMembers returns all members of a party, ordered by join date ascending.
func (s *Service) GetMembers(ctx context.Context, partyID int64) ([]*PartyMember, error) {
	members, err := ListMembers(s.DB, partyID)
	if err != nil {
		return nil, fmt.Errorf("party: list members failed: %w", err)
	}
	return members, nil
}
