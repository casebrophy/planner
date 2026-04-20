package commands_test

import (
	"testing"

	"github.com/casebrophy/planner/api/tooling/admin/commands"
)

// TestDebriefdedupe_StructExists verifies the command struct is defined.
// Full integration tests require Docker; this verifies the struct compiles.
func TestDebriefdedupe_StructExists(t *testing.T) {
	cmd := &commands.DebriefdedupeCMD{}
	if cmd == nil {
		t.Fatal("DebriefdedupeCMD struct is nil")
	}
}
