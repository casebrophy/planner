package clarificationbus_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/types/clarificationkind"
)

func TestNewClarificationItem_Validate(t *testing.T) {
	t.Parallel()

	valid := clarificationbus.NewClarificationItem{
		Kind:               clarificationkind.NewContext,
		SubjectType:        "context",
		SubjectID:          uuid.New(),
		SubjectDescription: "Test context",
		Question:           "Should this be a new context?",
		AnswerOptions:      json.RawMessage(`["yes","no"]`),
	}

	tests := []struct {
		name     string
		mutate   func(nc *clarificationbus.NewClarificationItem)
		wantErr  bool
		wantTerm string
	}{
		{name: "valid", mutate: func(*clarificationbus.NewClarificationItem) {}, wantErr: false},
		{name: "missing kind", mutate: func(nc *clarificationbus.NewClarificationItem) { nc.Kind = clarificationkind.Kind{} }, wantErr: true, wantTerm: "kind"},
		{name: "missing subject_type", mutate: func(nc *clarificationbus.NewClarificationItem) { nc.SubjectType = "  " }, wantErr: true, wantTerm: "subject_type"},
		{name: "missing subject_id", mutate: func(nc *clarificationbus.NewClarificationItem) { nc.SubjectID = uuid.Nil }, wantErr: true, wantTerm: "subject_id"},
		{name: "missing subject_description", mutate: func(nc *clarificationbus.NewClarificationItem) { nc.SubjectDescription = "" }, wantErr: true, wantTerm: "subject_description"},
		{name: "missing question", mutate: func(nc *clarificationbus.NewClarificationItem) { nc.Question = "" }, wantErr: true, wantTerm: "question"},
		{name: "missing answer_options", mutate: func(nc *clarificationbus.NewClarificationItem) { nc.AnswerOptions = nil }, wantErr: true, wantTerm: "answer_options"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			nc := valid
			tc.mutate(&nc)
			err := nc.Validate()
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if !errors.Is(err, clarificationbus.ErrInvalidClarification) {
					t.Fatalf("expected ErrInvalidClarification, got %v", err)
				}
				if !strings.Contains(err.Error(), tc.wantTerm) {
					t.Fatalf("expected error to mention %q, got %v", tc.wantTerm, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected nil, got %v", err)
			}
		})
	}
}
