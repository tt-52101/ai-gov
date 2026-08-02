package party

import (
	"context"
	"crypto/rand"
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
		return nil, fmt.Errorf("party: 无效类型 %q，必须为 %q 或 %q", req.Type, TypeOrg, TypeProject)
	}
	if req.Name == "" {
		return nil, errors.New("party: 名称为必填")
	}

	// 校验父级 Party 是否存在。
	if req.ParentPartyID != nil {
		if _, err := GetParty(s.DB, *req.ParentPartyID); err != nil {
			return nil, fmt.Errorf("party: 父级 party %s 未找到: %w", *req.ParentPartyID, err)
		}
	}

	p := &Party{
		ID:            newPartyID(),
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
		return nil, fmt.Errorf("party: 创建失败: %w", err)
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
func (s *Service) GetParties(ctx context.Context, partyType string) ([]*Party, error) {
	parties, err := ListParties(s.DB, partyType)
	if err != nil {
		return nil, fmt.Errorf("party: 列出失败: %w", err)
	}
	return parties, nil
}

// ── 边管理 ────────────────────────────────────────────────────

// CreateEdge 在两个 Party 之间创建类型化关系边。
// 它根据七种识别类型验证边类型，并对 parent、sponsors 和 allocates 边
// 自动设置 allows_fund=true（per PRD §2.4 资金流规则）。
//
// 自引用边（src == dst）被拒绝。源和目标 Party 必须都存在。
func (s *Service) CreateEdge(ctx context.Context, req CreateEdgeRequest) (*PartyEdge, error) {
	if req.SrcPartyID == req.DstPartyID {
		return nil, errors.New("party: 不允许自引用边")
	}
	if !ValidEdgeType(req.EdgeType) {
		return nil, fmt.Errorf("party: 无效边类型 %q", req.EdgeType)
	}

	// 验证双方都存在。
	if _, err := GetParty(s.DB, req.SrcPartyID); err != nil {
		return nil, fmt.Errorf("party: 源 party %s: %w", req.SrcPartyID, err)
	}
	if _, err := GetParty(s.DB, req.DstPartyID); err != nil {
		return nil, fmt.Errorf("party: 目标 party %s: %w", req.DstPartyID, err)
	}

	e := &PartyEdge{
		ID:         newUUID(),
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
		return nil, fmt.Errorf("party: 创建边失败: %w", err)
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
func (s *Service) DeleteEdge(ctx context.Context, edgeID string) error {
	if err := DeleteEdge(s.DB, edgeID); err != nil {
		slog.ErrorContext(ctx, "删除边失败", "edge_id", edgeID, "error", err)
		return fmt.Errorf("party: 删除边失败: %w", err)
	}
	slog.InfoContext(ctx, "删除边成功", "edge_id", edgeID)
	return nil
}

// GetEdges 返回连接到指定 Party 的所有边（作为源或目标）。
func (s *Service) GetEdges(ctx context.Context, partyID string) ([]*PartyEdge, error) {
	edges, err := ListEdges(s.DB, partyID)
	if err != nil {
		return nil, fmt.Errorf("party: 列出边失败: %w", err)
	}
	return edges, nil
}

// ── 资金划拨资格 ───────────────────────────────────────────

// channelToEdgeType 定义划拨通道常量到边类型的映射。
// 只有 parent、sponsors、allocates 通道允许资金划拨；
// owns、participates、merged_into、split_from 等边类型一律拒绝。
// whitelist 通道无对应边类型，需通过独立的 allocate_whitelist 表校验。
var channelToEdgeType = map[string]string{
	"parent":    EdgeParent,
	"sponsors":  EdgeSponsors,
	"allocates": EdgeAllocates,
}

// CanAllocate 检查是否允许从源 Party 账户向目标 Party 账户进行资金划拨。
//
// 在以下情况下允许资金划拨：
//   - 存在 parent 边，src 为上级、dst 为下级（仅向下方向）。
//   - 存在 sponsors 边，src 为出资方、dst 为被出资方（仅出资方向）。
//   - 存在 allocates 边，src party 向 dst（个人账户）划拨。
//
// 对 owns、participates、merged_into 或 split_from 边，资金划拨不允许。
// 向上级划拨（下级→上级）始终被拒绝。
// 双方之间无任何边的划拨也被拒绝。
func (s *Service) CanAllocate(ctx context.Context, srcPartyID, dstPartyID string) (bool, error) {
	// 检查 src 为来源、dst 为目标的所有边。
	edge, err := FindEdge(s.DB, srcPartyID, dstPartyID)
	if err != nil {
		return false, fmt.Errorf("party: 检查划拨资格失败: %w", err)
	}
	if edge == nil {
		return false, nil
	}

	// 仅 parent、sponsors 和 allocates 边允许资金划拨，
	// 且对 parent/sponsors 仅允许正向（src→dst）方向。
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

// ValidateChannel 校验划拨通道是否有效：源与目标 Party 之间的边必须存在、
// 允许资金划拨、且边类型必须与请求的 channel 参数一致。
//
// channel 参数来自 AllocateRequest.Channel，可选值为 "parent"、"sponsors"、
// "allocates" 或 "whitelist"。whitelist 通道需通过独立的
// allocate_whitelist 表校验，当前通过边校验时会被拒绝。
//
// 返回 (true, nil) 表示通道有效、允许划拨；
// 返回 (false, nil) 表示通道无效（边不存在、不允许资金划拨
// 或边类型与 channel 不匹配）；
// 返回 (false, error) 表示数据库查询异常。
func (s *Service) ValidateChannel(ctx context.Context, srcPartyID, dstPartyID string, channel string) (bool, error) {
	// 查询源→目标方向的边。
	edge, err := FindEdge(s.DB, srcPartyID, dstPartyID)
	if err != nil {
		return false, fmt.Errorf("party: 校验划拨通道失败: %w", err)
	}
	if edge == nil {
		slog.DebugContext(ctx, "划拨通道校验：无边关系",
			"src_party_id", srcPartyID,
			"dst_party_id", dstPartyID,
			"channel", channel,
		)
		return false, nil
	}

	// 检查边是否允许资金划拨。
	if !edge.AllowsFund {
		slog.DebugContext(ctx, "划拨通道校验：边不允许资金划拨",
			"edge_id", edge.ID,
			"edge_type", edge.EdgeType,
			"channel", channel,
		)
		return false, nil
	}

	// 校验 channel 参数与边类型的一致性。
	expectedEdgeType, ok := channelToEdgeType[channel]
	if !ok {
		// 未知 channel（如 whitelist）——不允许通过边校验。
		slog.WarnContext(ctx, "划拨通道校验：未知通道类型",
			"channel", channel,
			"src_party_id", srcPartyID,
			"dst_party_id", dstPartyID,
		)
		return false, nil
	}
	if edge.EdgeType != expectedEdgeType {
		slog.WarnContext(ctx, "划拨通道校验：通道与边类型不匹配",
			"expected_edge_type", expectedEdgeType,
			"actual_edge_type", edge.EdgeType,
			"channel", channel,
			"src_party_id", srcPartyID,
			"dst_party_id", dstPartyID,
			"edge_id", edge.ID,
		)
		return false, nil
	}

	// 确认边类型属于可划拨的三种类型之一（双重保障）。
	if edge.EdgeType != EdgeParent && edge.EdgeType != EdgeSponsors && edge.EdgeType != EdgeAllocates {
		slog.WarnContext(ctx, "划拨通道校验：边类型不可划拨",
			"edge_type", edge.EdgeType,
			"channel", channel,
		)
		return false, nil
	}

	slog.DebugContext(ctx, "划拨通道校验通过",
		"edge_id", edge.ID,
		"edge_type", edge.EdgeType,
		"channel", channel,
		"src_party_id", srcPartyID,
		"dst_party_id", dstPartyID,
	)
	return true, nil
}

// ── 成员管理 ───────────────────────────────────────────────

// AddMember 将用户以指定角色添加到 Party。若未指定角色则默认为 "member"。
// Leader 角色仅为描述性，不授予自动权限（A-CON-05）。
//
// Party 必须存在。重复的 (party_id, user_id) 对被数据库 UNIQUE 约束拒绝。
func (s *Service) AddMember(ctx context.Context, req AddMemberRequest) (*PartyMember, error) {
	if req.PartyID == "" {
		return nil, errors.New("party: party_id 为必填")
	}
	if req.UserID == "" {
		return nil, errors.New("party: user_id 为必填")
	}
	if req.Role == "" {
		req.Role = RoleMember
	}

	// 验证 Party 存在。
	if _, err := GetParty(s.DB, req.PartyID); err != nil {
		return nil, fmt.Errorf("party: party %s 未找到: %w", req.PartyID, err)
	}

	m := &PartyMember{
		ID:        newUUID(),
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
		return nil, fmt.Errorf("party: 添加成员失败: %w", err)
	}

	slog.InfoContext(ctx, "添加成员成功",
		"member_id", m.ID,
		"party_id", m.PartyID,
		"user_id", m.UserID,
		"role", m.Role,
	)
	return m, nil
}

// RemoveMember 按成员 ID 将用户从 Party 中移除。调用方负责确保该用户
// 在移除前没有绑定到此 Party 账户的活跃 API 密钥（per API 规范 §2.9）。
func (s *Service) RemoveMember(ctx context.Context, membershipID string) error {
	if err := DeleteMember(s.DB, membershipID); err != nil {
		slog.ErrorContext(ctx, "移除成员失败", "member_id", membershipID, "error", err)
		return fmt.Errorf("party: 移除成员失败: %w", err)
	}
	slog.InfoContext(ctx, "移除成员成功", "member_id", membershipID)
	return nil
}

// GetMembers 返回 Party 的所有成员，按加入日期升序排列。
func (s *Service) GetMembers(ctx context.Context, partyID string) ([]*PartyMember, error) {
	members, err := ListMembers(s.DB, partyID)
	if err != nil {
		return nil, fmt.Errorf("party: 列出成员失败: %w", err)
	}
	return members, nil
}

// newUUID 生成 UUID v4 格式的 ID。
func newUUID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// fallback 不应到达
		panic("crypto/rand.Read 失败: " + err.Error())
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// newPartyID 生成 UUID v4 格式的 Party ID。
func newPartyID() string { return newUUID() }
