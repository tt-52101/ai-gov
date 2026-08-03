// Package party 实现统一 Party 模型——组织和项目均视为平级"主体"。
// Party 持有账本、包含成员、通过关系边连接，边的类型决定资金划拨权限。
//
// 本包扩展 TokenHub 现有 projects 模型，引入 party_edges（7 种关系类型）
// 和 party_members（含角色分配），支持图状组织结构，超越简单项目-团队层级。
package party

import "time"

// ── Party type constants ────────────────────────────────────────────────

const (
	// TypeOrg 组织单位（部门、事业部等）。
	TypeOrg = "org"
	// TypeProject 项目实体，与组织平级。
	TypeProject = "project"
)

// ── Party status constants ──────────────────────────────────────────────

const (
	// StatusActive 正常运行。
	StatusActive = "active"
	// StatusInactive 软禁用。
	StatusInactive = "inactive"
	// StatusLiquidated 已清算，资产已回流。
	// have been transferred and it is now read-only.
	StatusLiquidated = "liquidated"
)

// ── Edge type constants (PRD §2.4) ──────────────────────────────────────

const (
	// EdgeParent defines an organizational tree edge (src is parent, dst is child).
	// Fund transfer is allowed downward (parent→child) by default.
	EdgeParent = "parent"

	// EdgeSponsors defines a sponsorship edge (src sponsors dst).
	// Fund transfer is allowed downward (sponsor→sponsored) by default.
	EdgeSponsors = "sponsors"

	// EdgeOwns defines a primary-responsibility edge.
	// Fund transfer is NOT auto-allowed.
	EdgeOwns = "owns"

	// EdgeParticipates defines a collaboration edge.
	// Fund transfer is NOT auto-allowed.
	EdgeParticipates = "participates"

	// EdgeAllocates defines a personal-fund injection edge from a Party
	// to a Person Account's owner party.
	// Fund transfer IS allowed (party→person account).
	EdgeAllocates = "allocates"

	// EdgeMergedInto records an organizational merge: src was merged into dst.
	// Fund transfer follows liquidation flow, not normal allocation.
	EdgeMergedInto = "merged_into"

	// EdgeSplitFrom records an organizational split: dst was split from src.
	// Fund transfer follows liquidation flow, not normal allocation.
	EdgeSplitFrom = "split_from"
)

// ── Member role constants ───────────────────────────────────────────────

const (
	// RoleLeader designates a party leader. This is descriptive only; no
	// automatic privileges are granted (A-CON-05).
	RoleLeader = "leader"

	// RoleMember designates a regular party member.
	RoleMember = "member"

	// RoleObserver designates a read-only observer of the party.
	RoleObserver = "observer"
)

// validEdgeTypes is the set of all recognized edge type values.
var validEdgeTypes = map[string]bool{
	EdgeParent:       true,
	EdgeSponsors:     true,
	EdgeOwns:         true,
	EdgeParticipates: true,
	EdgeAllocates:    true,
	EdgeMergedInto:   true,
	EdgeSplitFrom:    true,
}

// fundAutoEdges is the set of edge types for which allows_fund defaults to true.
var fundAutoEdges = map[string]bool{
	EdgeParent:   true,
	EdgeSponsors: true,
	EdgeAllocates: true,
}

// ── Domain models ───────────────────────────────────────────────────────

// Party 统一主体——组织与项目平级语义。
// table concept by introducing a type discriminator (org vs project) and
// graph-based relationships via party_edges. Every party can own an account,
// contain members, and participate in fund transfers governed by edge rules.
//
// GORM 表: parties
// Party v3.2: parties 表使用 TEXT 主键，不再使用自增整数 ID。
type Party struct {
	ID            string    `json:"id" gorm:"type:text;primaryKey"`              // v3.2: TEXT UUID
	Type          string    `json:"type" gorm:"type:varchar(32);not null;index"`
	Name          string    `json:"name" gorm:"type:varchar(128);not null"`
	Description   string    `json:"description,omitempty" gorm:"type:text"`
	ParentPartyID *string   `json:"parent_party_id,omitempty" gorm:"type:text;index"` // v3.2: TEXT
	TeamID          string    `json:"team_id,omitempty" gorm:"type:varchar(64);index"`   // 兼容旧版 Project 模型的团队归属
	DefaultQuotaRef string    `json:"default_quota_ref,omitempty" gorm:"type:varchar(64);index"` // 兼容旧版 Project 模型的默认配额引用
	LeaderUserID    string    `json:"leader_user_id,omitempty" gorm:"type:varchar(64)"`
	CostCenter      string    `json:"cost_center,omitempty" gorm:"type:varchar(64);index"`
	Status          string    `json:"status" gorm:"type:varchar(32);not null;default:active;index"`
	Metadata      string    `json:"metadata,omitempty" gorm:"type:text"`
	CreatedAt     time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt     time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 覆盖 GORM 默认表名。
func (Party) TableName() string { return "parties" }

// PartyEdge 两个 Party 之间的有向关系边。
// 7 种边类型：parent/sponsors/owns/participates/allocates/merged_into/split_from
// merged_into, split_from) determine whether fund transfers are permitted
// between the connected parties.
//
// GORM 表: party_edges
// PartyEdge v3.2: party_edges 表使用 TEXT 主键。
type PartyEdge struct {
	ID         string    `json:"id" gorm:"type:text;primaryKey"`
	SrcPartyID string    `json:"src_party_id" gorm:"type:text;not null;index"`
	DstPartyID string    `json:"dst_party_id" gorm:"type:text;not null;index"`
	EdgeType   string    `json:"edge_type" gorm:"type:varchar(32);not null"`
	AllowsFund bool      `json:"allows_fund" gorm:"not null;default:false"`
	CreatedAt  time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// TableName 覆盖 GORM 默认表名。
func (PartyEdge) TableName() string { return "party_edges" }

// PartyMember v3.2: party_members 表使用 TEXT 主键。
type PartyMember struct {
	ID        string    `json:"id" gorm:"type:text;primaryKey"`
	PartyID   string    `json:"party_id" gorm:"type:text;not null;index"`
	UserID    string    `json:"user_id" gorm:"type:varchar(64);not null;index"`
	Role      string    `json:"role" gorm:"type:varchar(32);not null;default:member"`
	IsPrimary bool      `json:"is_primary" gorm:"default:false"`
	JoinedAt  time.Time `json:"joined_at" gorm:"autoCreateTime"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 覆盖 GORM 默认表名。
func (PartyMember) TableName() string { return "party_members" }

// ── Request types ───────────────────────────────────────────────────────

// CreatePartyRequest 创建 Party 的请求参数。
// 项目不必须指定父级组织（PRD §2.3：组织与项目平级）。
// are peer-level entities).
type CreatePartyRequest struct {
	// Type must be one of "org" or "project".
	Type string `json:"type"`

	// Name is the display name (1-128 characters).
	Name string `json:"name"`

	// Description is optional free-form text.
	Description string `json:"description,omitempty"`

	// ParentPartyID is the optional organizational parent.
	// Projects may omit this entirely; orgs may use it to form a tree.
	ParentPartyID *string `json:"parent_party_id,omitempty"`

	// LeaderUserID is the responsible person for this party.
	// No automatic privileges are granted (A-CON-05).
	LeaderUserID string `json:"leader_user_id,omitempty"`

	// CostCenter is an optional financial cost center code.
	CostCenter string `json:"cost_center,omitempty"`

	// Metadata carries arbitrary JSON metadata.
	Metadata string `json:"metadata,omitempty"`
}

// CreateEdgeRequest v3.2: 使用 TEXT party ID。
type CreateEdgeRequest struct {
	SrcPartyID string `json:"src_party_id"`
	DstPartyID string `json:"dst_party_id"`
	EdgeType   string `json:"edge_type"`
}

// AddMemberRequest v3.2: 使用 TEXT party ID。
type AddMemberRequest struct {
	PartyID string `json:"party_id"`
	UserID  string `json:"user_id"`

	// Role must be one of "leader", "member", or "observer".
	// Defaults to "member" if empty.
	Role string `json:"role,omitempty"`

	// IsPrimary marks this as the user's primary membership.
	IsPrimary bool `json:"is_primary,omitempty"`
}

// ── Validation helpers ──────────────────────────────────────────────────

// ValidEdgeType 校验 t 是否为 7 种合法边类型之一。
func ValidEdgeType(t string) bool { return validEdgeTypes[t] }

// FundAutoAllowed 返回边类型 t 是否默认开通资金划拨。
func FundAutoAllowed(t string) bool { return fundAutoEdges[t] }
