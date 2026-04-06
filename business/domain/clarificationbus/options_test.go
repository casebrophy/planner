package clarificationbus_test

import (
	"encoding/json"
	"testing"

	"github.com/casebrophy/planner/business/domain/clarificationbus"
)

func TestContextAssignmentOptionsJSONFields(t *testing.T) {
	opts := clarificationbus.ContextAssignmentOptions{
		SuggestedContext: "abc123",
		Confidence:       0.7,
		AvailableContexts: []clarificationbus.ContextRef{
			{ID: "ctx1", Title: "Work"},
		},
	}
	b, err := json.Marshal(opts)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"suggested_context", "confidence", "available_contexts"} {
		if _, ok := got[want]; !ok {
			t.Errorf("missing field %q", want)
		}
	}
	if _, ok := got["suggested_context_id"]; ok {
		t.Error("field suggested_context_id must not exist (old wrong name)")
	}
}

func TestNewContextOptionsJSONFields(t *testing.T) {
	opts := clarificationbus.NewContextOptions{ContextID: "id1", Title: "Work"}
	b, _ := json.Marshal(opts)
	var got map[string]any
	json.Unmarshal(b, &got)
	for _, want := range []string{"context_id", "title"} {
		if _, ok := got[want]; !ok {
			t.Errorf("missing field %q", want)
		}
	}
}

func TestAmbiguousActionOptionsJSONFields(t *testing.T) {
	opts := clarificationbus.AmbiguousActionOptions{Interpretations: []string{"a", "b"}}
	b, _ := json.Marshal(opts)
	var got map[string]any
	json.Unmarshal(b, &got)
	if _, ok := got["interpretations"]; !ok {
		t.Error("missing field interpretations")
	}
}

func TestAmbiguousDeadlineOptionsJSONFields(t *testing.T) {
	opts := clarificationbus.AmbiguousDeadlineOptions{Description: "Friday", RawDate: "friday"}
	b, _ := json.Marshal(opts)
	var got map[string]any
	json.Unmarshal(b, &got)
	for _, want := range []string{"description", "raw_date"} {
		if _, ok := got[want]; !ok {
			t.Errorf("missing field %q", want)
		}
	}
}
