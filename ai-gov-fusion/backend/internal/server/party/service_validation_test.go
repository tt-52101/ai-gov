package party

import (
	"context"
	"testing"
)

// ── Edge lifecycle tests ────────────────────────────────────────────────

// TestDeleteEdge removes an edge and verifies the edge list is empty.
func TestDeleteEdge(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	a := createTestParty(t, svc, CreatePartyRequest{Type: TypeOrg, Name: "A"})
	b := createTestParty(t, svc, CreatePartyRequest{Type: TypeProject, Name: "B"})

	e, _ := svc.CreateEdge(ctx, CreateEdgeRequest{
		SrcPartyID: a.ID, DstPartyID: b.ID, EdgeType: EdgeParent,
	})

	if err := svc.DeleteEdge(ctx, e.ID); err != nil {
		t.Fatalf("DeleteEdge: %v", err)
	}

	edges, err := svc.GetEdges(ctx, a.ID)
	if err != nil {
		t.Fatalf("GetEdges: %v", err)
	}
	if len(edges) != 0 {
		t.Errorf("expected 0 edges, got %d", len(edges))
	}
}

// TestGetEdges returns all edges connected to a party (both directions).
func TestGetEdges(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	a := createTestParty(t, svc, CreatePartyRequest{Type: TypeOrg, Name: "A"})
	b := createTestParty(t, svc, CreatePartyRequest{Type: TypeProject, Name: "B"})
	c := createTestParty(t, svc, CreatePartyRequest{Type: TypeProject, Name: "C"})

	// a -> b (parent), c -> b (sponsors)
	_, _ = svc.CreateEdge(ctx, CreateEdgeRequest{SrcPartyID: a.ID, DstPartyID: b.ID, EdgeType: EdgeParent})
	_, _ = svc.CreateEdge(ctx, CreateEdgeRequest{SrcPartyID: c.ID, DstPartyID: b.ID, EdgeType: EdgeSponsors})

	edges, err := svc.GetEdges(ctx, b.ID)
	if err != nil {
		t.Fatalf("GetEdges: %v", err)
	}
	if len(edges) != 2 {
		t.Fatalf("expected 2 edges for party b, got %d", len(edges))
	}
}

// ── GetParties tests ────────────────────────────────────────────────────

// TestGetParties_All returns all parties when no type filter is given.
func TestGetParties_All(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	createTestParty(t, svc, CreatePartyRequest{Type: TypeOrg, Name: "Org1"})
	createTestParty(t, svc, CreatePartyRequest{Type: TypeProject, Name: "Proj1"})

	parties, err := svc.GetParties(ctx, "")
	if err != nil {
		t.Fatalf("GetParties: %v", err)
	}
	if len(parties) != 2 {
		t.Errorf("expected 2 parties, got %d", len(parties))
	}
}

// TestGetParties_TypeFilter returns only parties matching the type filter.
func TestGetParties_TypeFilter(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	createTestParty(t, svc, CreatePartyRequest{Type: TypeOrg, Name: "Org1"})
	createTestParty(t, svc, CreatePartyRequest{Type: TypeProject, Name: "Proj1"})

	parties, err := svc.GetParties(ctx, TypeOrg)
	if err != nil {
		t.Fatalf("GetParties: %v", err)
	}
	if len(parties) != 1 {
		t.Errorf("expected 1 org, got %d", len(parties))
	}
	if parties[0].Type != TypeOrg {
		t.Errorf("expected type org, got %s", parties[0].Type)
	}
}

// ── Edge type validation tests ──────────────────────────────────────────

// TestValidEdgeType_AllTypes verifies all seven edge types are recognized.
func TestValidEdgeType_AllTypes(t *testing.T) {
	all := []string{EdgeParent, EdgeSponsors, EdgeOwns, EdgeParticipates,
		EdgeAllocates, EdgeMergedInto, EdgeSplitFrom}
	for _, et := range all {
		if !ValidEdgeType(et) {
			t.Errorf("expected ValidEdgeType(%q) = true", et)
		}
	}
}

// TestValidEdgeType_Unknown rejects unknown edge types.
func TestValidEdgeType_Unknown(t *testing.T) {
	if ValidEdgeType("bogus") {
		t.Error("expected ValidEdgeType(\"bogus\") = false")
	}
}

// ── FundAutoAllowed tests ───────────────────────────────────────────────

// TestFundAutoAllowed verifies the correct edge types default to allows_fund.
func TestFundAutoAllowed(t *testing.T) {
	tests := []struct {
		edgeType string
		want     bool
	}{
		{EdgeParent, true},
		{EdgeSponsors, true},
		{EdgeAllocates, true},
		{EdgeOwns, false},
		{EdgeParticipates, false},
		{EdgeMergedInto, false},
		{EdgeSplitFrom, false},
	}
	for _, tt := range tests {
		got := FundAutoAllowed(tt.edgeType)
		if got != tt.want {
			t.Errorf("FundAutoAllowed(%q) = %v, want %v", tt.edgeType, got, tt.want)
		}
	}
}
