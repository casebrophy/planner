package clarificationbus_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/sdk/dbtest"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/sdk/unitest"
	"github.com/casebrophy/planner/business/types/clarificationkind"
	"github.com/casebrophy/planner/business/types/clarificationstatus"
)

func Test_Clarification(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, "Test_Clarification")

	items, err := clarificationbus.TestSeedClarifications(context.Background(), 2, db.BusDomain.Clarification)
	if err != nil {
		t.Fatalf("Seeding error: %s", err)
	}

	unitest.Run(t, query(db.BusDomain, items), "query")
	unitest.Run(t, create(db.BusDomain, items), "create")
	unitest.Run(t, resolve(db.BusDomain, items), "resolve")
	unitest.Run(t, snooze(db.BusDomain, items), "snooze")
}

func query(busDomain dbtest.BusDomain, items []clarificationbus.ClarificationItem) []unitest.Table {
	pendingStatus := clarificationstatus.Pending

	return []unitest.Table{
		{
			Name:    "pending",
			ExpResp: items,
			ExcFunc: func(ctx context.Context) any {
				filter := clarificationbus.QueryFilter{
					Status: &pendingStatus,
				}
				resp, err := busDomain.Clarification.Query(ctx, filter, clarificationbus.DefaultOrderBy, page.New(1, 10))
				if err != nil {
					return err
				}
				return resp
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.([]clarificationbus.ClarificationItem)
				if !exists {
					return "error occurred"
				}
				expResp := exp.([]clarificationbus.ClarificationItem)
				return cmp.Diff(gotResp, expResp,
					cmpopts.EquateApproxTime(time.Second),
					cmpopts.EquateComparable(clarificationkind.Kind{}, clarificationstatus.Status{}),
					cmpopts.SortSlices(func(a, b clarificationbus.ClarificationItem) bool {
						return a.ID.String() < b.ID.String()
					}),
				)
			},
		},
	}
}

func create(busDomain dbtest.BusDomain, _ []clarificationbus.ClarificationItem) []unitest.Table {
	return []unitest.Table{
		{
			Name: "basic",
			ExpResp: clarificationbus.ClarificationItem{
				Kind:               clarificationkind.NewContext,
				Status:             clarificationstatus.Pending,
				SubjectType:        "context",
				SubjectDescription: "Test context",
				Question:           "Should this be a new context?",
				AnswerOptions:      json.RawMessage(`["yes","no"]`),
				PriorityScore:      0.9,
			},
			ExcFunc: func(ctx context.Context) any {
				ni := clarificationbus.NewClarificationItem{
					Kind:               clarificationkind.NewContext,
					SubjectType:        "context",
					SubjectID:          uuid.New(),
					SubjectDescription: "Test context",
					Question:           "Should this be a new context?",
					AnswerOptions:      json.RawMessage(`["yes","no"]`),
					PriorityScore:      0.9,
				}
				resp, err := busDomain.Clarification.Create(ctx, ni)
				if err != nil {
					return err
				}
				return resp
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(clarificationbus.ClarificationItem)
				if !exists {
					return "error occurred"
				}
				expResp := exp.(clarificationbus.ClarificationItem)
				expResp.ID = gotResp.ID
				expResp.SubjectID = gotResp.SubjectID
				expResp.CreatedAt = gotResp.CreatedAt
				expResp.PriorityScore = gotResp.PriorityScore
				return cmp.Diff(gotResp, expResp, cmpopts.EquateComparable(clarificationkind.Kind{}, clarificationstatus.Status{}))
			},
		},
	}
}

func resolve(busDomain dbtest.BusDomain, items []clarificationbus.ClarificationItem) []unitest.Table {
	return []unitest.Table{
		{
			Name: "basic",
			ExpResp: clarificationbus.ClarificationItem{
				Status: clarificationstatus.Resolved,
			},
			ExcFunc: func(ctx context.Context) any {
				rc := clarificationbus.ResolveClarificationItem{
					Answer: json.RawMessage(`"yes"`),
				}
				resp, err := busDomain.Clarification.Resolve(ctx, items[0], rc)
				if err != nil {
					return err
				}
				return resp
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(clarificationbus.ClarificationItem)
				if !exists {
					return "error occurred"
				}
				// Only verify the status transition occurred correctly.
				if gotResp.Status != clarificationstatus.Resolved {
					return cmp.Diff(gotResp.Status.String(), clarificationstatus.Resolved.String())
				}
				return ""
			},
		},
	}
}

func snooze(busDomain dbtest.BusDomain, items []clarificationbus.ClarificationItem) []unitest.Table {
	return []unitest.Table{
		{
			Name: "basic",
			ExpResp: clarificationbus.ClarificationItem{
				Status: clarificationstatus.Snoozed,
			},
			ExcFunc: func(ctx context.Context) any {
				until := time.Now().Add(24 * time.Hour)
				resp, err := busDomain.Clarification.Snooze(ctx, items[1], until)
				if err != nil {
					return err
				}
				return resp
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(clarificationbus.ClarificationItem)
				if !exists {
					return "error occurred"
				}
				// Only verify the status transition occurred correctly.
				if gotResp.Status != clarificationstatus.Snoozed {
					return cmp.Diff(gotResp.Status.String(), clarificationstatus.Snoozed.String())
				}
				return ""
			},
		},
	}
}

func Test_ClarificationUpsert_Idempotency(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, "Test_ClarificationUpsert_Idempotency")

	unitest.Run(t, upsertIdempotency(db.BusDomain), "upsert_idempotency")
}

func Test_ClarificationUpsert_PreservesAnswer(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, "Test_ClarificationUpsert_PreservesAnswer")

	unitest.Run(t, upsertPreservesAnswer(db.BusDomain), "upsert_preserves_answer")
}

func Test_ClarificationUpsert_RespectsDedupKey(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, "Test_ClarificationUpsert_RespectsDedupKey")

	unitest.Run(t, upsertRespectsDedupKey(db.BusDomain), "upsert_respects_dedup_key")
}

func Test_ClarificationUpsert_SourceHashDedup(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, "Test_ClarificationUpsert_SourceHashDedup")

	unitest.Run(t, upsertSourceHashDedup(db.BusDomain), "upsert_source_hash_dedup")
}

func upsertIdempotency(busDomain dbtest.BusDomain) []unitest.Table {
	return []unitest.Table{
		{
			Name:    "same_input_twice",
			ExpResp: 1,
			ExcFunc: func(ctx context.Context) any {
				subjectID := uuid.New()
				nc := clarificationbus.NewClarificationItem{
					Kind:               clarificationkind.NewContext,
					SubjectType:        "context",
					SubjectID:          subjectID,
					SubjectDescription: "Test context",
					Question:           "Should this be a new context?",
					AnswerOptions:      json.RawMessage(`["yes","no"]`),
				}
				if _, err := busDomain.Clarification.Upsert(ctx, nc); err != nil {
					return err
				}
				if _, err := busDomain.Clarification.Upsert(ctx, nc); err != nil {
					return err
				}
				count, err := busDomain.Clarification.Count(ctx, clarificationbus.QueryFilter{
					SubjectID: &subjectID,
				})
				if err != nil {
					return err
				}
				return count
			},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(got, exp)
			},
		},
	}
}

func upsertPreservesAnswer(busDomain dbtest.BusDomain) []unitest.Table {
	return []unitest.Table{
		{
			Name:    "answer_preserved_on_reupsert",
			ExpResp: true,
			ExcFunc: func(ctx context.Context) any {
				subjectID := uuid.New()
				nc := clarificationbus.NewClarificationItem{
					Kind:               clarificationkind.NewContext,
					SubjectType:        "context",
					SubjectID:          subjectID,
					SubjectDescription: "Test context",
					Question:           "Original question?",
					AnswerOptions:      json.RawMessage(`["yes","no"]`),
				}
				item, err := busDomain.Clarification.Upsert(ctx, nc)
				if err != nil {
					return err
				}

				// Simulate user answering
				answer := json.RawMessage(`"yes"`)
				rc := clarificationbus.ResolveClarificationItem{Answer: answer}
				_, err = busDomain.Clarification.Resolve(ctx, item, rc)
				if err != nil {
					return err
				}

				// Upsert again with different AI fields
				nc.Question = "Updated question?"
				if _, err := busDomain.Clarification.Upsert(ctx, nc); err != nil {
					return err
				}

				// Reload and check answer preserved
				updated, err := busDomain.Clarification.QueryByID(ctx, item.ID)
				if err != nil {
					return err
				}

				return updated.Status == clarificationstatus.Resolved && updated.Answer != nil
			},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(got, exp)
			},
		},
	}
}

func upsertRespectsDedupKey(busDomain dbtest.BusDomain) []unitest.Table {
	return []unitest.Table{
		{
			Name:    "different_kinds_create_distinct_rows",
			ExpResp: 2,
			ExcFunc: func(ctx context.Context) any {
				subjectID := uuid.New()
				nc1 := clarificationbus.NewClarificationItem{
					Kind:               clarificationkind.NewContext,
					SubjectType:        "context",
					SubjectID:          subjectID,
					SubjectDescription: "Test context",
					Question:           "New context question?",
					AnswerOptions:      json.RawMessage(`["yes","no"]`),
				}
				nc2 := clarificationbus.NewClarificationItem{
					Kind:               clarificationkind.StaleTask,
					SubjectType:        "context",
					SubjectID:          subjectID,
					SubjectDescription: "Test context",
					Question:           "Stale task question?",
					AnswerOptions:      json.RawMessage(`["yes","no"]`),
				}
				if _, err := busDomain.Clarification.Upsert(ctx, nc1); err != nil {
					return err
				}
				if _, err := busDomain.Clarification.Upsert(ctx, nc2); err != nil {
					return err
				}
				count, err := busDomain.Clarification.Count(ctx, clarificationbus.QueryFilter{
					SubjectID: &subjectID,
				})
				if err != nil {
					return err
				}
				return count
			},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(got, exp)
			},
		},
	}
}

func upsertSourceHashDedup(busDomain dbtest.BusDomain) []unitest.Table {
	return []unitest.Table{
		{
			Name:    "source_hash_dedup_on_re_ingest",
			ExpResp: 1,
			ExcFunc: func(ctx context.Context) any {
				// Create an ingestion-derived clarification with source_hash
				sourceHash := clarificationbus.ComputeSourceHash("test-email-id-123")
				initialSubjectID := uuid.New()

				nc := clarificationbus.NewClarificationItem{
					Kind:               clarificationkind.ContextAssignment,
					SubjectType:        "email",
					SubjectID:          initialSubjectID,
					SubjectDescription: "Email: test subject",
					Question:           "Which context?",
					AnswerOptions:      json.RawMessage(`{"suggested":null}`),
					SourceHash:         sourceHash,
				}

				// First Upsert creates the clarification
				first, err := busDomain.Clarification.Upsert(ctx, nc)
				if err != nil {
					return err
				}

				// Second Upsert with different subject_id but same source_hash (re-ingest scenario)
				// should update the existing clarification, not create a new one
				newSubjectID := uuid.New()
				nc.SubjectID = newSubjectID
				nc.SubjectDescription = "Email: updated subject (re-ingested)"
				nc.Question = "Which context? (updated)"
				second, err := busDomain.Clarification.Upsert(ctx, nc)
				if err != nil {
					return err
				}

				// Should return the same ID (dedup worked)
				if second.ID != first.ID {
					return "expected same ID after re-ingest, got different IDs"
				}

				// Verify only one clarification exists
				count, err := busDomain.Clarification.Count(ctx, clarificationbus.QueryFilter{})
				if err != nil {
					return err
				}

				return count
			},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(got, exp)
			},
		},
	}
}
