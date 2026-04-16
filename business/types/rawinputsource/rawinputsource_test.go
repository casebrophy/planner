package rawinputsource_test

import (
	"testing"

	"github.com/casebrophy/planner/business/types/rawinputsource"
)

func TestManualEnum(t *testing.T) {
	src, err := rawinputsource.Parse("manual")
	if err != nil {
		t.Fatalf("Parse(manual): %v", err)
	}
	if src != rawinputsource.Manual {
		t.Errorf("Parse(manual) = %v, want Manual", src)
	}
	if src.String() != "manual" {
		t.Errorf("Manual.String() = %q, want %q", src.String(), "manual")
	}
}
