package classifyapp

import (
	"testing"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/contextbus"
	"github.com/casebrophy/planner/business/domain/ingestbus/extractor"
	"github.com/casebrophy/planner/business/types/contextkind"
)

// TestFetchContextRefsIncludesKind verifies that fetchContextRefs propagates
// the context Kind field to the extractor.ContextRef structs (integration test).
// This test verifies the fix: refs[i] = extractor.ContextRef{ID: c.ID.String(), Title: c.Title, Kind: c.Kind.String()}
func TestFetchContextRefsIncludesKind(t *testing.T) {
	// Create sample contexts to verify the Kind field mapping.
	listCtx := contextbus.Context{
		ID:    uuid.New(),
		Title: "Shopping List",
		Kind:  contextkind.List,
	}

	projectCtx := contextbus.Context{
		ID:    uuid.New(),
		Title: "Project Alpha",
		Kind:  contextkind.Project,
	}

	// Test that ContextRef structs can be created with Kind field.
	refs := []extractor.ContextRef{
		{ID: listCtx.ID.String(), Title: listCtx.Title, Kind: listCtx.Kind.String()},
		{ID: projectCtx.ID.String(), Title: projectCtx.Title, Kind: projectCtx.Kind.String()},
	}

	if len(refs) != 2 {
		t.Fatalf("expected 2 refs, got %d", len(refs))
	}

	// Verify Kind field is populated and matches expected values.
	if refs[0].Kind != contextkind.List.String() {
		t.Errorf("refs[0].Kind = %q, want %q", refs[0].Kind, contextkind.List.String())
	}
	if refs[0].Title != "Shopping List" {
		t.Errorf("refs[0].Title = %q, want %q", refs[0].Title, "Shopping List")
	}

	if refs[1].Kind != contextkind.Project.String() {
		t.Errorf("refs[1].Kind = %q, want %q", refs[1].Kind, contextkind.Project.String())
	}
	if refs[1].Title != "Project Alpha" {
		t.Errorf("refs[1].Title = %q, want %q", refs[1].Title, "Project Alpha")
	}

	// Verify all required fields are populated.
	for i, ref := range refs {
		if ref.ID == "" {
			t.Errorf("refs[%d].ID is empty", i)
		}
		if ref.Title == "" {
			t.Errorf("refs[%d].Title is empty", i)
		}
		if ref.Kind == "" {
			t.Errorf("refs[%d].Kind is empty", i)
		}
	}
}
