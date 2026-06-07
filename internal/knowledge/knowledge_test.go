package knowledge

import "testing"

func TestNodeValidate(t *testing.T) {
	tests := []struct {
		name    string
		node    Node
		wantErr bool
	}{
		{"valid entity", Node{Kind: KindEntity, EntityType: EntityService, Title: "PostgreSQL", Scope: ScopeGlobal}, false},
		{"valid fact", Node{Kind: KindFact, Title: "uses pnpm", Scope: ScopeProject}, false},
		{"entity without type", Node{Kind: KindEntity, Title: "x", Scope: ScopeGlobal}, true},
		{"fact with entity_type", Node{Kind: KindFact, EntityType: EntityService, Title: "x", Scope: ScopeProject}, true},
		{"bad kind", Node{Kind: "widget", Title: "x", Scope: ScopeGlobal}, true},
		{"missing title", Node{Kind: KindFact, Scope: ScopeProject}, true},
		{"bad scope", Node{Kind: KindFact, Title: "x", Scope: "universe"}, true},
		{"bad maturity", Node{Kind: KindFact, Title: "x", Scope: ScopeProject, Maturity: "wizard"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.node.Validate(); (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestEdgeValidate(t *testing.T) {
	if err := (Edge{SrcID: "a", EdgeType: EdgeUses, DstID: "b"}).Validate(); err != nil {
		t.Fatalf("valid edge rejected: %v", err)
	}
	if err := (Edge{SrcID: "a", EdgeType: "loves", DstID: "b"}).Validate(); err == nil {
		t.Fatal("invalid edge_type accepted")
	}
	if err := (Edge{SrcID: "a", EdgeType: EdgeUses, DstID: "a"}).Validate(); err == nil {
		t.Fatal("self-edge accepted")
	}
	if err := (Edge{SrcID: "", EdgeType: EdgeUses, DstID: "b"}).Validate(); err == nil {
		t.Fatal("empty src accepted")
	}
}

func TestEnumValidity(t *testing.T) {
	if !KindEntity.Valid() || !EntityService.Valid() || !ScopeGlobal.Valid() ||
		!MaturityFact.Valid() || !EdgeRunsOn.Valid() {
		t.Fatal("a known enum value reported invalid")
	}
	if NodeKind("x").Valid() || EntityType("x").Valid() || Scope("x").Valid() ||
		Maturity("x").Valid() || EdgeType("x").Valid() {
		t.Fatal("an unknown enum value reported valid")
	}
}
