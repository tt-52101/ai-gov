package party

import (
	"context"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// newTestDB opens an in-memory SQLite database, runs AutoMigrate for the
// party tables, and returns the *gorm.DB handle.
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

// newTestService creates a Service backed by the test database.
func newTestService(t *testing.T) *Service {
	t.Helper()
	return NewService(newTestDB(t))
}

// createTestParty creates a party and fails the test on error.
func createTestParty(t *testing.T, svc *Service, req CreatePartyRequest) *Party {
	t.Helper()
	p, err := svc.CreateParty(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateParty: %v", err)
	}
	return p
}

// ── Party creation tests ────────────────────────────────────────────────

// TestCreateParty_Org verifies that an org-type party can be created with
// all fields populated correctly.
func TestCreateParty_Org(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	p, err := svc.CreateParty(ctx, CreatePartyRequest{
		Type:        TypeOrg,
		Name:        "AI R&D Department",
		Description: "Central AI research and development",
		CostCenter:  "CC-AI-001",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if p.Type != TypeOrg {
		t.Errorf("expected type %q, got %q", TypeOrg, p.Type)
	}
	if p.Name != "AI R&D Department" {
		t.Errorf("expected name %q, got %q", "AI R&D Department", p.Name)
	}
	if p.Status != StatusActive {
		t.Errorf("expected status %q, got %q", StatusActive, p.Status)
	}
	if p.Description != "Central AI research and development" {
		t.Errorf("expected description %q, got %q", "Central AI research and development", p.Description)
	}
	if p.CostCenter != "CC-AI-001" {
		t.Errorf("expected cost_center %q, got %q", "CC-AI-001", p.CostCenter)
	}
}

// TestCreateParty_Project_NoParent verifies that a project can be created
// without a parent party (PRD §2.3: projects and orgs are peer-level).
func TestCreateParty_Project_NoParent(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	p, err := svc.CreateParty(ctx, CreatePartyRequest{
		Type: TypeProject,
		Name: "Q3 Strategic Initiative",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if p.Type != TypeProject {
		t.Errorf("expected type %q, got %q", TypeProject, p.Type)
	}
	if p.ParentPartyID != nil {
		t.Errorf("expected nil parent_party_id, got %d", *p.ParentPartyID)
	}
}

// TestCreateParty_Project_WithParent verifies that a project can be created
// under an existing org parent.
func TestCreateParty_Project_WithParent(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	parent := createTestParty(t, svc, CreatePartyRequest{
		Type: TypeOrg,
		Name: "Parent Org",
	})

	child, err := svc.CreateParty(ctx, CreatePartyRequest{
		Type:          TypeProject,
		Name:          "Child Project",
		ParentPartyID: &parent.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if child.ParentPartyID == nil || *child.ParentPartyID != parent.ID {
		t.Errorf("expected parent_party_id %d, got %v", parent.ID, child.ParentPartyID)
	}
}

// TestCreateParty_InvalidType rejects unknown party types.
func TestCreateParty_InvalidType(t *testing.T) {
	svc := newTestService(t)
	_, err := svc.CreateParty(context.Background(), CreatePartyRequest{
		Type: "invalid",
		Name: "Test",
	})
	if err == nil {
		t.Fatal("expected error for invalid type")
	}
}

// TestCreateParty_MissingName rejects empty names.
func TestCreateParty_MissingName(t *testing.T) {
	svc := newTestService(t)
	_, err := svc.CreateParty(context.Background(), CreatePartyRequest{
		Type: TypeOrg,
		Name: "",
	})
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

// ── Edge creation tests ─────────────────────────────────────────────────

// TestCreateEdge_Parent verifies that a parent edge is created with
// allows_fund=true set automatically.
func TestCreateEdge_Parent(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	org := createTestParty(t, svc, CreatePartyRequest{Type: TypeOrg, Name: "Parent"})
	proj := createTestParty(t, svc, CreatePartyRequest{Type: TypeProject, Name: "Child"})

	e, err := svc.CreateEdge(ctx, CreateEdgeRequest{
		SrcPartyID: org.ID,
		DstPartyID: proj.ID,
		EdgeType:   EdgeParent,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.ID == 0 {
		t.Error("expected non-zero edge ID")
	}
	if e.EdgeType != EdgeParent {
		t.Errorf("expected edge_type %q, got %q", EdgeParent, e.EdgeType)
	}
	if !e.AllowsFund {
		t.Error("parent edge should have allows_fund=true")
	}
}

// TestCreateEdge_Sponsors verifies that a sponsors edge is created with
// allows_fund=true set automatically.
func TestCreateEdge_Sponsors(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	sponsor := createTestParty(t, svc, CreatePartyRequest{Type: TypeOrg, Name: "Sponsor"})
	recipient := createTestParty(t, svc, CreatePartyRequest{Type: TypeProject, Name: "Recipient"})

	e, err := svc.CreateEdge(ctx, CreateEdgeRequest{
		SrcPartyID: sponsor.ID,
		DstPartyID: recipient.ID,
		EdgeType:   EdgeSponsors,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !e.AllowsFund {
		t.Error("sponsors edge should have allows_fund=true")
	}
}

// TestCreateEdge_Owns verifies that an owns edge is created with
// allows_fund=false (fund transfer not auto-allowed per PRD §2.4).
func TestCreateEdge_Owns(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	a := createTestParty(t, svc, CreatePartyRequest{Type: TypeOrg, Name: "Owner"})
	b := createTestParty(t, svc, CreatePartyRequest{Type: TypeProject, Name: "Owned"})

	e, err := svc.CreateEdge(ctx, CreateEdgeRequest{
		SrcPartyID: a.ID,
		DstPartyID: b.ID,
		EdgeType:   EdgeOwns,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.AllowsFund {
		t.Error("owns edge should have allows_fund=false")
	}
}

// TestCreateEdge_SelfRef rejects self-referencing edges.
func TestCreateEdge_SelfRef(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	p := createTestParty(t, svc, CreatePartyRequest{Type: TypeOrg, Name: "Test"})

	_, err := svc.CreateEdge(ctx, CreateEdgeRequest{
		SrcPartyID: p.ID,
		DstPartyID: p.ID,
		EdgeType:   EdgeParent,
	})
	if err == nil {
		t.Fatal("expected error for self-referencing edge")
	}
}

// ── Fund allocation eligibility tests ────────────────────────────────────

// TestCanAllocate_ParentDownward verifies that fund transfer is allowed
// from parent to child via a parent edge.
func TestCanAllocate_ParentDownward(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	parent := createTestParty(t, svc, CreatePartyRequest{Type: TypeOrg, Name: "Parent"})
	child := createTestParty(t, svc, CreatePartyRequest{Type: TypeProject, Name: "Child"})

	_, err := svc.CreateEdge(ctx, CreateEdgeRequest{
		SrcPartyID: parent.ID,
		DstPartyID: child.ID,
		EdgeType:   EdgeParent,
	})
	if err != nil {
		t.Fatalf("create edge: %v", err)
	}

	allowed, err := svc.CanAllocate(ctx, parent.ID, child.ID)
	if err != nil {
		t.Fatalf("CanAllocate: %v", err)
	}
	if !allowed {
		t.Error("parent→child should allow fund transfer")
	}
}

// TestCanAllocate_ParentUpward verifies that fund transfer is blocked
// from child to parent (only downward parent edges permit fund flow).
func TestCanAllocate_ParentUpward(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	parent := createTestParty(t, svc, CreatePartyRequest{Type: TypeOrg, Name: "Parent"})
	child := createTestParty(t, svc, CreatePartyRequest{Type: TypeProject, Name: "Child"})

	// Create parent edge: parent → child (regular direction).
	_, err := svc.CreateEdge(ctx, CreateEdgeRequest{
		SrcPartyID: parent.ID,
		DstPartyID: child.ID,
		EdgeType:   EdgeParent,
	})
	if err != nil {
		t.Fatalf("create edge: %v", err)
	}

	// Check reverse direction: child → parent (should be blocked).
	allowed, err := svc.CanAllocate(ctx, child.ID, parent.ID)
	if err != nil {
		t.Fatalf("CanAllocate: %v", err)
	}
	if allowed {
		t.Error("child→parent should NOT allow fund transfer")
	}
}

// TestCanAllocate_SponsorsDirection verifies that fund transfer is allowed
// from sponsor to sponsored party.
func TestCanAllocate_SponsorsDirection(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	sponsor := createTestParty(t, svc, CreatePartyRequest{Type: TypeOrg, Name: "Sponsor"})
	sponsored := createTestParty(t, svc, CreatePartyRequest{Type: TypeProject, Name: "Sponsored"})

	_, err := svc.CreateEdge(ctx, CreateEdgeRequest{
		SrcPartyID: sponsor.ID,
		DstPartyID: sponsored.ID,
		EdgeType:   EdgeSponsors,
	})
	if err != nil {
		t.Fatalf("create edge: %v", err)
	}

	allowed, err := svc.CanAllocate(ctx, sponsor.ID, sponsored.ID)
	if err != nil {
		t.Fatalf("CanAllocate: %v", err)
	}
	if !allowed {
		t.Error("sponsor→sponsored should allow fund transfer")
	}
}

// TestCanAllocate_OwnsDenied verifies that fund transfer is NOT allowed
// via an owns edge (PRD §2.4: owns does not auto-allow fund transfer).
func TestCanAllocate_OwnsDenied(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	owner := createTestParty(t, svc, CreatePartyRequest{Type: TypeOrg, Name: "Owner"})
	owned := createTestParty(t, svc, CreatePartyRequest{Type: TypeProject, Name: "Owned"})

	_, err := svc.CreateEdge(ctx, CreateEdgeRequest{
		SrcPartyID: owner.ID,
		DstPartyID: owned.ID,
		EdgeType:   EdgeOwns,
	})
	if err != nil {
		t.Fatalf("create edge: %v", err)
	}

	allowed, err := svc.CanAllocate(ctx, owner.ID, owned.ID)
	if err != nil {
		t.Fatalf("CanAllocate: %v", err)
	}
	if allowed {
		t.Error("owns edge should NOT allow fund transfer")
	}
}

// TestCanAllocate_NoEdge denies fund transfer when no edge exists between
// the two parties.
func TestCanAllocate_NoEdge(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	a := createTestParty(t, svc, CreatePartyRequest{Type: TypeOrg, Name: "A"})
	b := createTestParty(t, svc, CreatePartyRequest{Type: TypeProject, Name: "B"})

	allowed, err := svc.CanAllocate(ctx, a.ID, b.ID)
	if err != nil {
		t.Fatalf("CanAllocate: %v", err)
	}
	if allowed {
		t.Error("no edge should deny fund transfer")
	}
}

// ── Member management tests ─────────────────────────────────────────────

// TestAddMember_Leader adds a member with the leader role and verifies
// the returned membership fields.
func TestAddMember_Leader(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	org := createTestParty(t, svc, CreatePartyRequest{Type: TypeOrg, Name: "Org"})

	m, err := svc.AddMember(ctx, AddMemberRequest{
		PartyID: org.ID,
		UserID:  "user-leader-1",
		Role:    RoleLeader,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.ID == 0 {
		t.Error("expected non-zero member ID")
	}
	if m.Role != RoleLeader {
		t.Errorf("expected role %q, got %q", RoleLeader, m.Role)
	}
	if m.UserID != "user-leader-1" {
		t.Errorf("expected user_id %q, got %q", "user-leader-1", m.UserID)
	}
	if m.PartyID != org.ID {
		t.Errorf("expected party_id %d, got %d", org.ID, m.PartyID)
	}
}

// TestAddMember_Observer adds a member with the observer role.
func TestAddMember_Observer(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	org := createTestParty(t, svc, CreatePartyRequest{Type: TypeOrg, Name: "Org"})

	m, err := svc.AddMember(ctx, AddMemberRequest{
		PartyID: org.ID,
		UserID:  "user-observer-1",
		Role:    RoleObserver,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Role != RoleObserver {
		t.Errorf("expected role %q, got %q", RoleObserver, m.Role)
	}
}

// TestAddMember_DefaultRole defaults to "member" when no role is specified.
func TestAddMember_DefaultRole(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	org := createTestParty(t, svc, CreatePartyRequest{Type: TypeOrg, Name: "Org"})

	m, err := svc.AddMember(ctx, AddMemberRequest{
		PartyID: org.ID,
		UserID:  "user-no-role",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Role != RoleMember {
		t.Errorf("expected default role %q, got %q", RoleMember, m.Role)
	}
}

// TestGetMembers returns all members of a party in join order.
func TestGetMembers(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	org := createTestParty(t, svc, CreatePartyRequest{Type: TypeOrg, Name: "Org"})

	// Add two members.
	_, _ = svc.AddMember(ctx, AddMemberRequest{PartyID: org.ID, UserID: "user-1", Role: RoleMember})
	_, _ = svc.AddMember(ctx, AddMemberRequest{PartyID: org.ID, UserID: "user-2", Role: RoleLeader})

	members, err := svc.GetMembers(ctx, org.ID)
	if err != nil {
		t.Fatalf("GetMembers: %v", err)
	}
	if len(members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(members))
	}
	if members[0].UserID != "user-1" {
		t.Errorf("expected first member user-1, got %s", members[0].UserID)
	}
	if members[1].UserID != "user-2" {
		t.Errorf("expected second member user-2, got %s", members[1].UserID)
	}
}

// TestRemoveMember removes a member and verifies they no longer appear.
func TestRemoveMember(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	org := createTestParty(t, svc, CreatePartyRequest{Type: TypeOrg, Name: "Org"})
	m, _ := svc.AddMember(ctx, AddMemberRequest{PartyID: org.ID, UserID: "user-rm", Role: RoleMember})

	if err := svc.RemoveMember(ctx, m.ID); err != nil {
		t.Fatalf("RemoveMember: %v", err)
	}

	members, err := svc.GetMembers(ctx, org.ID)
	if err != nil {
		t.Fatalf("GetMembers: %v", err)
	}
	if len(members) != 0 {
		t.Errorf("expected 0 members after removal, got %d", len(members))
	}
}


