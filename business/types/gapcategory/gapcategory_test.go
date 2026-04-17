package gapcategory

import (
	"testing"
)

func TestMissingContactConstant(t *testing.T) {
	if MissingContact.value != "missing_contact" {
		t.Errorf("MissingContact.value = %q, want %q", MissingContact.value, "missing_contact")
	}
	if MissingContact.String() != "missing_contact" {
		t.Errorf("MissingContact.String() = %q, want %q", MissingContact.String(), "missing_contact")
	}
}

func TestParseMissingContact(t *testing.T) {
	c, err := Parse("missing_contact")
	if err != nil {
		t.Fatalf("Parse(\"missing_contact\") failed: %v", err)
	}
	if c != MissingContact {
		t.Errorf("Parse(\"missing_contact\") = %v, want %v", c, MissingContact)
	}
}

func TestMustParseMissingContact(t *testing.T) {
	c := MustParse("missing_contact")
	if c != MissingContact {
		t.Errorf("MustParse(\"missing_contact\") = %v, want %v", c, MissingContact)
	}
}

func TestMissingContactMarshalText(t *testing.T) {
	data, err := MissingContact.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText() failed: %v", err)
	}
	if string(data) != "missing_contact" {
		t.Errorf("MarshalText() = %q, want %q", string(data), "missing_contact")
	}
}

func TestMissingContactUnmarshalText(t *testing.T) {
	var c Category
	err := c.UnmarshalText([]byte("missing_contact"))
	if err != nil {
		t.Fatalf("UnmarshalText() failed: %v", err)
	}
	if c != MissingContact {
		t.Errorf("UnmarshalText() = %v, want %v", c, MissingContact)
	}
}

func TestMissingContactEqualString(t *testing.T) {
	if !MissingContact.EqualString("missing_contact") {
		t.Error("MissingContact.EqualString(\"missing_contact\") = false, want true")
	}
	if MissingContact.EqualString("missing_location") {
		t.Error("MissingContact.EqualString(\"missing_location\") = true, want false")
	}
}

func TestAllCategories(t *testing.T) {
	if len(AllCategories) != 8 {
		t.Errorf("AllCategories length = %d, want 8", len(AllCategories))
	}
}

func TestParseAllCategories(t *testing.T) {
	for _, cat := range AllCategories {
		c, err := Parse(cat.String())
		if err != nil {
			t.Fatalf("Parse(%q) failed: %v", cat.String(), err)
		}
		if c != cat {
			t.Errorf("Parse(%q) = %v, want %v", cat.String(), c, cat)
		}
	}
}

func TestMustParseInvalid(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustParse(invalid) did not panic")
		}
	}()
	MustParse("invalid_category")
}

func TestParseInvalid(t *testing.T) {
	c, err := Parse("invalid_category")
	if err == nil {
		t.Errorf("Parse(\"invalid_category\") succeeded, got %v, want error", c)
	}
}

func TestMarshalTextRoundTrip(t *testing.T) {
	for _, cat := range AllCategories {
		data, err := cat.MarshalText()
		if err != nil {
			t.Fatalf("MarshalText(%v) failed: %v", cat, err)
		}
		var c Category
		err = c.UnmarshalText(data)
		if err != nil {
			t.Fatalf("UnmarshalText(%q) failed: %v", string(data), err)
		}
		if c != cat {
			t.Errorf("Round-trip %v != %v", c, cat)
		}
	}
}

func TestMissingLocationConstant(t *testing.T) {
	if MissingLocation.value != "missing_location" {
		t.Errorf("MissingLocation.value = %q, want %q", MissingLocation.value, "missing_location")
	}
}

func TestMissingContextConstant(t *testing.T) {
	if MissingContext.value != "missing_context" {
		t.Errorf("MissingContext.value = %q, want %q", MissingContext.value, "missing_context")
	}
}

func TestMissingDependencyConstant(t *testing.T) {
	if MissingDependency.value != "missing_dependency" {
		t.Errorf("MissingDependency.value = %q, want %q", MissingDependency.value, "missing_dependency")
	}
}

func TestMissingDetailConstant(t *testing.T) {
	if MissingDetail.value != "missing_detail" {
		t.Errorf("MissingDetail.value = %q, want %q", MissingDetail.value, "missing_detail")
	}
}

func TestMissingDeadlineConstant(t *testing.T) {
	if MissingDeadline.value != "missing_deadline" {
		t.Errorf("MissingDeadline.value = %q, want %q", MissingDeadline.value, "missing_deadline")
	}
}

func TestMissingStakeholderConstant(t *testing.T) {
	if MissingStakeholder.value != "missing_stakeholder" {
		t.Errorf("MissingStakeholder.value = %q, want %q", MissingStakeholder.value, "missing_stakeholder")
	}
}

func TestMissingOutcomeConstant(t *testing.T) {
	if MissingOutcome.value != "missing_outcome" {
		t.Errorf("MissingOutcome.value = %q, want %q", MissingOutcome.value, "missing_outcome")
	}
}
