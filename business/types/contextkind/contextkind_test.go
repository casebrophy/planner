package contextkind

import (
	"testing"
)

func TestListConstant(t *testing.T) {
	if List.value != "list" {
		t.Errorf("List.value = %q, want %q", List.value, "list")
	}
	if List.String() != "list" {
		t.Errorf("List.String() = %q, want %q", List.String(), "list")
	}
}

func TestParseList(t *testing.T) {
	k, err := Parse("list")
	if err != nil {
		t.Fatalf("Parse(\"list\") failed: %v", err)
	}
	if k != List {
		t.Errorf("Parse(\"list\") = %v, want %v", k, List)
	}
}

func TestMustParseList(t *testing.T) {
	k := MustParse("list")
	if k != List {
		t.Errorf("MustParse(\"list\") = %v, want %v", k, List)
	}
}

func TestListMarshalText(t *testing.T) {
	data, err := List.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText() failed: %v", err)
	}
	if string(data) != "list" {
		t.Errorf("MarshalText() = %q, want %q", string(data), "list")
	}
}

func TestListUnmarshalText(t *testing.T) {
	var k Kind
	err := k.UnmarshalText([]byte("list"))
	if err != nil {
		t.Fatalf("UnmarshalText() failed: %v", err)
	}
	if k != List {
		t.Errorf("UnmarshalText() = %v, want %v", k, List)
	}
}

func TestListEqualString(t *testing.T) {
	if !List.EqualString("list") {
		t.Error("List.EqualString(\"list\") = false, want true")
	}
	if List.EqualString("project") {
		t.Error("List.EqualString(\"project\") = true, want false")
	}
}
