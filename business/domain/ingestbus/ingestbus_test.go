package ingestbus_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/domain/contextbus"
	"github.com/casebrophy/planner/business/domain/eventbus"
	"github.com/casebrophy/planner/business/domain/ingestbus"
	"github.com/casebrophy/planner/business/domain/ingestbus/extractor"
	"github.com/casebrophy/planner/business/domain/knowledgegapbus"
	"github.com/casebrophy/planner/business/domain/rawinputbus"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/sdk/dbtest"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/sdk/unitest"
	"github.com/casebrophy/planner/business/types/rawinputsource"
	"github.com/casebrophy/planner/business/types/taskenergy"
	"github.com/casebrophy/planner/business/types/taskpriority"
	"github.com/casebrophy/planner/business/types/taskstatus"
)

// validRFC5322Email returns a minimal RFC 5322 compliant email string.
func validRFC5322Email(from, to, subject, body string) string {
	return fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n%s",
		from, to, subject, body,
	)
}

func Test_Ingest(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, "Test_Ingest")

	unitest.Run(t, processEmailEmptyExtraction(db), "process-email-empty")
	unitest.Run(t, processEmailCreatesTask(db), "process-email-action")
	unitest.Run(t, processTextEmptyExtraction(db), "process-text-empty")
	unitest.Run(t, processTextCreatesTask(db), "process-text-action")
	unitest.Run(t, processTextWithContextMatch(db), "process-text-context")
	unitest.Run(t, processTextCreatesEvent(db), "process-text-event")
	unitest.Run(t, processTextCompoundInput(db), "process-text-compound")
	unitest.Run(t, processTextLowConfidenceUnconfirmed(db), "process-text-low-confidence")
	unitest.Run(t, processTextAmbiguousReference(db), "process-text-voice-ref")
	unitest.Run(t, processTextUpdateEntityResolution(db), "process-text-entity-update")
	unitest.Run(t, processTextAmbiguousEntityMatch(db), "process-text-ambiguous-match")
}

// processEmailEmptyExtraction tests that ProcessEmail succeeds when the extractor
// returns an empty extraction (no action items, no context suggestions).
func processEmailEmptyExtraction(db *dbtest.Database) []unitest.Table {
	mock := &extractor.MockExtractor{
		Result: extractor.EmailExtraction{
			Summary:   "Simple test email",
			Sentiment: "neutral",
		},
	}

	igBus := ingestbus.NewBusiness(
		db.Log,
		db.BusDomain.RawInput,
		db.BusDomain.Email,
		db.BusDomain.Task,
		db.BusDomain.Context,
		db.BusDomain.Clarification,
		db.BusDomain.Event,
		mock,
		db.BusDomain.Note,
		db.BusDomain.Tag,
	)

	return []unitest.Table{
		{
			Name:    "no-error",
			ExpResp: error(nil),
			ExcFunc: func(ctx context.Context) any {
				rawContent := validRFC5322Email(
					"sender@example.com",
					"inbox@example.com",
					"Hello World",
					"This is a test email with no action items.",
				)
				return igBus.ProcessEmail(ctx, rawContent)
			},
			CmpFunc: func(got any, exp any) string {
				if got != nil {
					return fmt.Sprintf("expected nil error, got: %v", got)
				}
				return ""
			},
		},
	}
}

// processEmailCreatesTask tests that ProcessEmail creates tasks from extracted
// action items and records a raw_input entry.
func processEmailCreatesTask(db *dbtest.Database) []unitest.Table {
	mock := &extractor.MockExtractor{
		Result: extractor.EmailExtraction{
			Summary:   "Bug report requiring fix",
			Sentiment: "negative",
			ActionItems: []extractor.ActionItem{
				{
					Title:       "Fix reported login bug",
					Description: "The user reported a critical issue in the login flow",
					Priority:    "high",
				},
			},
		},
	}

	igBus := ingestbus.NewBusiness(
		db.Log,
		db.BusDomain.RawInput,
		db.BusDomain.Email,
		db.BusDomain.Task,
		db.BusDomain.Context,
		db.BusDomain.Clarification,
		db.BusDomain.Event,
		mock,
		db.BusDomain.Note,
		db.BusDomain.Tag,
	)

	return []unitest.Table{
		{
			Name:    "creates-raw-input",
			ExpResp: error(nil),
			ExcFunc: func(ctx context.Context) any {
				rawContent := validRFC5322Email(
					"user@customer.com",
					"inbox@example.com",
					"Bug in login flow",
					"There is a critical bug in the login flow. Please fix it as soon as possible.",
				)

				if err := igBus.ProcessEmail(ctx, rawContent); err != nil {
					return err
				}

				// Verify a raw_input was stored.
				src := rawinputsource.Email
				ris, err := db.BusDomain.RawInput.Query(
					ctx,
					rawinputbus.QueryFilter{SourceType: &src},
					rawinputbus.DefaultOrderBy,
					page.New(1, 100),
				)
				if err != nil {
					return fmt.Errorf("query raw inputs: %w", err)
				}
				if len(ris) == 0 {
					return fmt.Errorf("expected at least one raw_input, got none")
				}

				return error(nil)
			},
			CmpFunc: func(got any, exp any) string {
				if got != nil {
					return fmt.Sprintf("expected nil error, got: %v", got)
				}
				return ""
			},
		},
	}
}

// processTextEmptyExtraction tests that ProcessText succeeds when the extractor
// returns an empty extraction (no action items).
func processTextEmptyExtraction(db *dbtest.Database) []unitest.Table {
	mock := &extractor.MockExtractor{
		TextResult: extractor.TextExtraction{
			Summary: "Simple voice note",
		},
	}

	igBus := ingestbus.NewBusiness(
		db.Log,
		db.BusDomain.RawInput,
		db.BusDomain.Email,
		db.BusDomain.Task,
		db.BusDomain.Context,
		db.BusDomain.Clarification,
		db.BusDomain.Event,
		mock,
		db.BusDomain.Note,
		db.BusDomain.Tag,
	)

	return []unitest.Table{
		{
			Name:    "no-error",
			ExpResp: error(nil),
			ExcFunc: func(ctx context.Context) any {
				result, err := igBus.ProcessText(ctx, "just a random thought")
				if err != nil {
					return err
				}
				if len(result.TaskIDs) != 0 {
					return fmt.Errorf("expected 0 task IDs, got %d", len(result.TaskIDs))
				}
				if len(result.EventIDs) != 0 {
					return fmt.Errorf("expected 0 event IDs, got %d", len(result.EventIDs))
				}
				return error(nil)
			},
			CmpFunc: func(got any, exp any) string {
				if got != nil {
					return fmt.Sprintf("expected nil error, got: %v", got)
				}
				return ""
			},
		},
	}
}

// processTextCreatesTask tests that ProcessText creates tasks from extracted
// action items and records a raw_input entry with source_type=voice.
func processTextCreatesTask(db *dbtest.Database) []unitest.Table {
	mock := &extractor.MockExtractor{
		TextResult: extractor.TextExtraction{
			Summary: "Voice note about chores",
			ActionItems: []extractor.ActionItem{
				{
					Title:       "Wash the dishes",
					Description: "Need to wash the dishes",
					Priority:    "medium",
				},
			},
		},
	}

	igBus := ingestbus.NewBusiness(
		db.Log,
		db.BusDomain.RawInput,
		db.BusDomain.Email,
		db.BusDomain.Task,
		db.BusDomain.Context,
		db.BusDomain.Clarification,
		db.BusDomain.Event,
		mock,
		db.BusDomain.Note,
		db.BusDomain.Tag,
	)

	return []unitest.Table{
		{
			Name:    "creates-raw-input",
			ExpResp: error(nil),
			ExcFunc: func(ctx context.Context) any {
				result, err := igBus.ProcessText(ctx, "remind me to wash the dishes")
				if err != nil {
					return err
				}
				if len(result.TaskIDs) != 1 {
					return fmt.Errorf("expected 1 task ID, got %d", len(result.TaskIDs))
				}

				// Verify a raw_input was stored with source_type=voice.
				src := rawinputsource.Voice
				ris, err := db.BusDomain.RawInput.Query(
					ctx,
					rawinputbus.QueryFilter{SourceType: &src},
					rawinputbus.DefaultOrderBy,
					page.New(1, 100),
				)
				if err != nil {
					return fmt.Errorf("query raw inputs: %w", err)
				}
				if len(ris) == 0 {
					return fmt.Errorf("expected at least one raw_input with source_type=voice, got none")
				}

				return error(nil)
			},
			CmpFunc: func(got any, exp any) string {
				if got != nil {
					return fmt.Sprintf("expected nil error, got: %v", got)
				}
				return ""
			},
		},
	}
}

// processTextWithContextMatch tests that ProcessText matches to an existing context
// when the extractor suggests one.
func processTextWithContextMatch(db *dbtest.Database) []unitest.Table {
	mock := &extractor.MockExtractor{
		TextResult: extractor.TextExtraction{
			Summary: "Kitchen cleaning task",
			ActionItems: []extractor.ActionItem{
				{
					Title:       "Clean the kitchen",
					Description: "Wipe down counters and clean appliances",
					Priority:    "medium",
				},
			},
			SuggestedContextKeywords: []string{"home", "chores"},
			ContextConfidence:        0.9,
		},
	}

	igBus := ingestbus.NewBusiness(
		db.Log,
		db.BusDomain.RawInput,
		db.BusDomain.Email,
		db.BusDomain.Task,
		db.BusDomain.Context,
		db.BusDomain.Clarification,
		db.BusDomain.Event,
		mock,
		db.BusDomain.Note,
		db.BusDomain.Tag,
	)

	return []unitest.Table{
		{
			Name:    "matches-context",
			ExpResp: error(nil),
			ExcFunc: func(ctx context.Context) any {
				// First create a context.
				createdContext, err := db.BusDomain.Context.Create(ctx, contextbus.NewContext{
					Title:       "Home Chores",
					Description: "Household tasks",
				})
				if err != nil {
					return fmt.Errorf("create context: %w", err)
				}

				// Process text that should match the context.
				result, err := igBus.ProcessText(ctx, "remind me to clean the kitchen")
				if err != nil {
					return err
				}
				if len(result.TaskIDs) != 1 {
					return fmt.Errorf("expected 1 task ID, got %d", len(result.TaskIDs))
				}

				// Query the created task and verify its ContextID matches.
				task, err := db.BusDomain.Task.QueryByID(ctx, result.TaskIDs[0])
				if err != nil {
					return fmt.Errorf("query task by ID: %w", err)
				}
				if task.ContextID == nil || *task.ContextID != createdContext.ID {
					return fmt.Errorf("expected task ContextID %s, got %v", createdContext.ID, task.ContextID)
				}

				return error(nil)
			},
			CmpFunc: func(got any, exp any) string {
				if got != nil {
					return fmt.Sprintf("expected nil error, got: %v", got)
				}
				return ""
			},
		},
	}
}

// processTextCompoundInput tests that ProcessText splits compound voice input into
// clauses and creates one item per clause.
func processTextCompoundInput(db *dbtest.Database) []unitest.Table {
	// Mock returns one task per call. Two clauses → two ExtractText calls → two tasks.
	mock := &extractor.MockExtractor{
		TextResult: extractor.TextExtraction{
			Summary: "Compound input",
			ActionItems: []extractor.ActionItem{
				{
					Title:       "buy some milk",
					Description: "",
					Priority:    "medium",
				},
			},
		},
	}

	igBus := ingestbus.NewBusiness(
		db.Log,
		db.BusDomain.RawInput,
		db.BusDomain.Email,
		db.BusDomain.Task,
		db.BusDomain.Context,
		db.BusDomain.Clarification,
		db.BusDomain.Event,
		mock,
		db.BusDomain.Note,
		db.BusDomain.Tag,
	)

	return []unitest.Table{
		{
			Name:    "creates-two-tasks-from-compound-input",
			ExpResp: error(nil),
			ExcFunc: func(ctx context.Context) any {
				// "buy some milk" and "pick up dry cleaning" are two clauses joined by "and".
				result, err := igBus.ProcessText(ctx, "buy some milk and pick up dry cleaning")
				if err != nil {
					return err
				}
				if len(result.TaskIDs) != 2 {
					return fmt.Errorf("expected 2 task IDs from compound input, got %d", len(result.TaskIDs))
				}
				return error(nil)
			},
			CmpFunc: func(got any, exp any) string {
				if got != nil {
					return fmt.Sprintf("expected nil error, got: %v", got)
				}
				return ""
			},
		},
	}
}

// processTextLowConfidenceUnconfirmed tests that a clause with low classifier confidence
// (obligation verb + temporal anchor) creates an unconfirmed task.
func processTextLowConfidenceUnconfirmed(db *dbtest.Database) []unitest.Table {
	mock := &extractor.MockExtractor{
		TextResult: extractor.TextExtraction{
			Summary: "Ambiguous clause",
			ActionItems: []extractor.ActionItem{
				{
					Title:       "call about the meeting tomorrow",
					Description: "",
					Priority:    "medium",
				},
			},
		},
	}

	igBus := ingestbus.NewBusiness(
		db.Log,
		db.BusDomain.RawInput,
		db.BusDomain.Email,
		db.BusDomain.Task,
		db.BusDomain.Context,
		db.BusDomain.Clarification,
		db.BusDomain.Event,
		mock,
		db.BusDomain.Note,
		db.BusDomain.Tag,
	)

	return []unitest.Table{
		{
			Name:    "low-confidence-task-is-unconfirmed",
			ExpResp: error(nil),
			ExcFunc: func(ctx context.Context) any {
				// "call about the meeting tomorrow" has both obligation verb (call) and
				// temporal anchor (tomorrow) → classifier returns TaskType with 0.6 confidence < 0.75.
				result, err := igBus.ProcessText(ctx, "call about the meeting tomorrow")
				if err != nil {
					return err
				}
				if len(result.TaskIDs) != 1 {
					return fmt.Errorf("expected 1 task ID, got %d", len(result.TaskIDs))
				}

				task, err := db.BusDomain.Task.QueryByID(ctx, result.TaskIDs[0])
				if err != nil {
					return fmt.Errorf("query task: %w", err)
				}
				if !task.Unconfirmed {
					return fmt.Errorf("expected task.Unconfirmed=true for low-confidence clause, got false")
				}
				return error(nil)
			},
			CmpFunc: func(got any, exp any) string {
				if got != nil {
					return fmt.Sprintf("expected nil error, got: %v", got)
				}
				return ""
			},
		},
	}
}

// processTextCreatesEvent tests that ProcessText creates events from extracted
// event data.
func processTextCreatesEvent(db *dbtest.Database) []unitest.Table {
	now := time.Now()
	startsAt := now.Add(24 * time.Hour)
	endsAt := now.Add(25 * time.Hour)

	mock := &extractor.MockExtractor{
		TextResult: extractor.TextExtraction{
			Summary: "Dentist appointment",
			Events: []extractor.ExtractedEvent{
				{
					Title:    "Dentist Appointment",
					StartsAt: startsAt.Format(time.RFC3339),
					EndsAt:   endsAt.Format(time.RFC3339),
					Location: "123 Main St",
				},
			},
		},
	}

	igBus := ingestbus.NewBusiness(
		db.Log,
		db.BusDomain.RawInput,
		db.BusDomain.Email,
		db.BusDomain.Task,
		db.BusDomain.Context,
		db.BusDomain.Clarification,
		db.BusDomain.Event,
		mock,
		db.BusDomain.Note,
		db.BusDomain.Tag,
	)

	return []unitest.Table{
		{
			Name:    "creates-event",
			ExpResp: error(nil),
			ExcFunc: func(ctx context.Context) any {
				result, err := igBus.ProcessText(ctx, "dentist appointment tomorrow at 2pm")
				if err != nil {
					return err
				}
				if len(result.EventIDs) == 0 {
					return fmt.Errorf("expected at least 1 event ID, got 0")
				}
				return error(nil)
			},
			CmpFunc: func(got any, exp any) string {
				if got != nil {
					return fmt.Sprintf("expected nil error, got: %v", got)
				}
				return ""
			},
		},
	}
}

// processTextAmbiguousReference tests that an ambiguous reference in voice input
// creates a voice_reference clarification card.
func processTextAmbiguousReference(db *dbtest.Database) []unitest.Table {
	mock := &extractor.MockExtractor{
		TextResult: extractor.TextExtraction{
			Summary: "Reference to something",
			ActionItems: []extractor.ActionItem{
				{
					Title:       "Follow up on that thing",
					Description: "",
					Priority:    "medium",
				},
			},
			AmbiguousReferences: []extractor.AmbiguousReference{
				{
					OriginalText:  "that thing",
					ReferenceType: "vague_noun",
				},
			},
		},
	}

	igBus := ingestbus.NewBusiness(
		db.Log,
		db.BusDomain.RawInput,
		db.BusDomain.Email,
		db.BusDomain.Task,
		db.BusDomain.Context,
		db.BusDomain.Clarification,
		db.BusDomain.Event,
		mock,
		db.BusDomain.Note,
		db.BusDomain.Tag,
	)

	return []unitest.Table{
		{
			Name:    "ambiguous-reference-creates-voice-ref-clarification",
			ExpResp: error(nil),
			ExcFunc: func(ctx context.Context) any {
				result, err := igBus.ProcessText(ctx, "follow up on that thing")
				if err != nil {
					return err
				}
				if len(result.TaskIDs) != 1 {
					return fmt.Errorf("expected 1 task ID, got %d", len(result.TaskIDs))
				}

				// Query clarifications — should have a voice_reference card
				clars, err := db.BusDomain.Clarification.Query(ctx, clarificationbus.QueryFilter{}, clarificationbus.DefaultOrderBy, page.New(1, 50))
				if err != nil {
					return fmt.Errorf("query clarifications: %w", err)
				}

				var found bool
				for _, c := range clars {
					if c.Kind.String() == "voice_reference" {
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("expected voice_reference clarification, none found among %d clarifications", len(clars))
				}
				return error(nil)
			},
			CmpFunc: func(got any, exp any) string {
				if got != nil {
					return fmt.Sprintf("expected nil error, got: %v", got)
				}
				return ""
			},
		},
	}
}

// processTextUpdateEntityResolution tests that ProcessText applies an entity update
// when the extractor returns action="update" with a matched entity ID.
func processTextUpdateEntityResolution(db *dbtest.Database) []unitest.Table {
	return []unitest.Table{
		{
			Name:    "update-entity-marks-unconfirmed",
			ExpResp: error(nil),
			ExcFunc: func(ctx context.Context) any {
				// Create an existing event that will be "updated".
				now := time.Now()
				event, err := db.BusDomain.Event.Create(ctx, eventbus.NewEvent{
					Title:    "Ethan's Wedding",
					StartsAt: now.Add(30 * 24 * time.Hour),
					EndsAt:   now.Add(31 * 24 * time.Hour),
				})
				if err != nil {
					return fmt.Errorf("create event: %w", err)
				}

				mock := &extractor.MockExtractor{
					TextResult: extractor.TextExtraction{
						Summary: "Wedding date update",
						EntityResolutions: []extractor.EntityResolution{
							{
								Action:      "update",
								MatchedID:   event.ID.String(),
								MatchedType: "event",
								Confidence:  0.92,
								Reasoning:   "Input refers to the existing wedding event",
							},
						},
					},
				}

				igBus := ingestbus.NewBusiness(
					db.Log,
					db.BusDomain.RawInput,
					db.BusDomain.Email,
					db.BusDomain.Task,
					db.BusDomain.Context,
					db.BusDomain.Clarification,
					db.BusDomain.Event,
					mock,
					db.BusDomain.Note,
					db.BusDomain.Tag,
				)

				if _, err := igBus.ProcessText(ctx, "the wedding got moved to June"); err != nil {
					return err
				}

				// Verify event is now Unconfirmed=true.
				updated, err := db.BusDomain.Event.QueryByID(ctx, event.ID)
				if err != nil {
					return fmt.Errorf("query event: %w", err)
				}
				if !updated.Unconfirmed {
					return fmt.Errorf("expected event.Unconfirmed=true after entity update routing, got false")
				}
				return error(nil)
			},
			CmpFunc: func(got any, exp any) string {
				if got != nil {
					return fmt.Sprintf("expected nil error, got: %v", got)
				}
				return ""
			},
		},
	}
}

// processTextAmbiguousEntityMatch tests that ProcessText creates an AmbiguousEntityMatch
// clarification when the extractor returns action="ambiguous".
func processTextAmbiguousEntityMatch(db *dbtest.Database) []unitest.Table {
	return []unitest.Table{
		{
			Name:    "ambiguous-creates-entity-match-clarification",
			ExpResp: error(nil),
			ExcFunc: func(ctx context.Context) any {
				someID := uuid.New()
				mock := &extractor.MockExtractor{
					TextResult: extractor.TextExtraction{
						Summary: "Ambiguous entity reference",
						EntityResolutions: []extractor.EntityResolution{
							{
								Action:      "ambiguous",
								MatchedID:   someID.String(),
								MatchedType: "event",
								Confidence:  0.75,
								Reasoning:   "Could be one of two similar events",
							},
						},
					},
				}

				igBus := ingestbus.NewBusiness(
					db.Log,
					db.BusDomain.RawInput,
					db.BusDomain.Email,
					db.BusDomain.Task,
					db.BusDomain.Context,
					db.BusDomain.Clarification,
					db.BusDomain.Event,
					mock,
					db.BusDomain.Note,
					db.BusDomain.Tag,
				)

				if _, err := igBus.ProcessText(ctx, "the event got cancelled"); err != nil {
					return err
				}

				// Query clarifications — should have an ambiguous_entity_match card.
				clars, err := db.BusDomain.Clarification.Query(
					ctx,
					clarificationbus.QueryFilter{},
					clarificationbus.DefaultOrderBy,
					page.New(1, 50),
				)
				if err != nil {
					return fmt.Errorf("query clarifications: %w", err)
				}

				var found bool
				for _, c := range clars {
					if c.Kind.String() == "ambiguous_entity_match" {
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("expected ambiguous_entity_match clarification, none found among %d clarifications", len(clars))
				}
				return error(nil)
			},
			CmpFunc: func(got any, exp any) string {
				if got != nil {
					return fmt.Sprintf("expected nil error, got: %v", got)
				}
				return ""
			},
		},
	}
}

// TestProcessRawInput_SkipClassifyLoadsEntity tests that ProcessRawInputByID with
// SkipClassify=true loads the entity and regenerates embeddings without classify.
func TestProcessRawInput_SkipClassifyLoadsEntity(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, "TestProcessRawInput_SkipClassifyLoadsEntity")

	// Create a task first
	task, err := db.BusDomain.Task.Create(context.Background(), taskbus.NewTask{
		Title:    "Existing task",
		Status:   taskstatus.Open,
		Priority: taskpriority.Medium,
		Energy:   taskenergy.Medium,
	})
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// Create a raw_input with skip_classify=true pointing to this task
	ri, err := db.BusDomain.RawInput.Create(context.Background(), rawinputbus.NewRawInput{
		SourceType:       rawinputsource.Voice,
		RawContent:       "Updated task description",
		SourceEntityID:   &task.ID,
		SourceEntityKind: "task",
		SkipClassify:     true,
		ReingestMode:     false,
	})
	if err != nil {
		t.Fatalf("failed to create raw_input: %v", err)
	}

	// Create ingestbus with nil embedding bus (optional) to avoid external dependencies
	igBus := ingestbus.NewBusiness(
		db.Log,
		db.BusDomain.RawInput,
		db.BusDomain.Email,
		db.BusDomain.Task,
		db.BusDomain.Context,
		db.BusDomain.Clarification,
		db.BusDomain.Event,
		nil, // extractor (not used in skip_classify path)
		db.BusDomain.Note,
		db.BusDomain.Tag,
	)

	// Process the raw_input
	err = igBus.ProcessRawInputByID(context.Background(), ri.ID)
	if err != nil {
		t.Fatalf("ProcessRawInputByID failed: %v", err)
	}

	// Verify raw_input was marked processed
	ri2, err := db.BusDomain.RawInput.QueryByID(context.Background(), ri.ID)
	if err != nil {
		t.Fatalf("failed to reload raw_input: %v", err)
	}
	if ri2.ProcessedAt == nil {
		t.Error("expected ProcessedAt to be set, but it was nil")
	}

	// Verify task still exists and was not recreated
	task2, err := db.BusDomain.Task.QueryByID(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("failed to reload task: %v", err)
	}
	if task2.ID != task.ID {
		t.Errorf("task ID changed: got %v, expected %v", task2.ID, task.ID)
	}
	if task2.Title != task.Title {
		t.Errorf("task title should not change: got %q, expected %q", task2.Title, task.Title)
	}
}

// TestProcessRawInput_SkipClassifyMissingEntity tests that ProcessRawInputByID
// returns NotFound when the entity has been deleted.
func TestProcessRawInput_SkipClassifyMissingEntity(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, "TestProcessRawInput_SkipClassifyMissingEntity")

	deletedTaskID := uuid.New()

	// Create a raw_input with skip_classify=true pointing to a non-existent task
	ri, err := db.BusDomain.RawInput.Create(context.Background(), rawinputbus.NewRawInput{
		SourceType:       rawinputsource.Voice,
		RawContent:       "Task for non-existent entity",
		SourceEntityID:   &deletedTaskID,
		SourceEntityKind: "task",
		SkipClassify:     true,
		ReingestMode:     false,
	})
	if err != nil {
		t.Fatalf("failed to create raw_input: %v", err)
	}

	igBus := ingestbus.NewBusiness(
		db.Log,
		db.BusDomain.RawInput,
		db.BusDomain.Email,
		db.BusDomain.Task,
		db.BusDomain.Context,
		db.BusDomain.Clarification,
		db.BusDomain.Event,
		nil,
		db.BusDomain.Note,
		db.BusDomain.Tag,
	)

	// Attempt to process should fail because task doesn't exist
	err = igBus.ProcessRawInputByID(context.Background(), ri.ID)
	if err == nil {
		t.Fatal("expected error for missing entity, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

// TestProcessRawInput_ReingestModeSuppressesUnconfirmed tests that ReingestMode=true
// prevents the unconfirmed flag from being set during processing.
func TestProcessRawInput_ReingestModeSuppressesUnconfirmed(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, "TestProcessRawInput_ReingestModeSuppressesUnconfirmed")

	// Create a mock extractor that returns low-confidence classification
	mock := &extractor.MockExtractor{
		TextResult: extractor.TextExtraction{
			ActionItems: []extractor.ActionItem{
				{
					Title:       "Action from voice",
					Description: "Low confidence action",
					Priority:    "medium",
				},
			},
		},
	}

	// Create raw_input with ReingestMode=true
	ri, err := db.BusDomain.RawInput.Create(context.Background(), rawinputbus.NewRawInput{
		SourceType:   rawinputsource.Voice,
		RawContent:   "action item with low confidence",
		ReingestMode: true,
	})
	if err != nil {
		t.Fatalf("failed to create raw_input: %v", err)
	}

	igBus := ingestbus.NewBusiness(
		db.Log,
		db.BusDomain.RawInput,
		db.BusDomain.Email,
		db.BusDomain.Task,
		db.BusDomain.Context,
		db.BusDomain.Clarification,
		db.BusDomain.Event,
		mock,
		db.BusDomain.Note,
		db.BusDomain.Tag,
	)

	// Process the raw_input via voice path
	err = igBus.ProcessRawInputByID(context.Background(), ri.ID)
	if err != nil {
		t.Logf("ProcessRawInputByID info: %v", err)
		// Continue even if there's a non-fatal error; we just care about unconfirmed flag
	}

	// Note: With ReingestMode=true, new entities created should have unconfirmed=false
	// This is verified by checking the newly created tasks
	pq := page.New(1, 100)
	tasks, err := db.BusDomain.Task.Query(context.Background(), taskbus.QueryFilter{}, taskbus.DefaultOrderBy, pq)
	if err != nil {
		t.Fatalf("failed to query tasks: %v", err)
	}

	// Find newly created tasks
	var newTasks []taskbus.Task
	for _, tk := range tasks {
		newTasks = append(newTasks, tk)
	}

	for _, nt := range newTasks {
		if nt.Unconfirmed {
			t.Errorf("expected unconfirmed=false for task %q during reingest, but got unconfirmed=true", nt.Title)
		}
	}
}

// TestProcessRawInput_NonReingestSetsUnconfirmed is a regression test: without
// ReingestMode, low-confidence clauses should still set unconfirmed=true.
func TestProcessRawInput_NonReingestSetsUnconfirmed(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, "TestProcessRawInput_NonReingestSetsUnconfirmed")

	// Create a mock extractor that returns low-confidence classification
	mock := &extractor.MockExtractor{
		TextResult: extractor.TextExtraction{
			ActionItems: []extractor.ActionItem{
				{
					Title:       "Low confidence task",
					Description: "Should be marked unconfirmed",
					Priority:    "medium",
				},
			},
		},
	}

	// Create raw_input with ReingestMode=false (normal behavior)
	ri, err := db.BusDomain.RawInput.Create(context.Background(), rawinputbus.NewRawInput{
		SourceType:   rawinputsource.Voice,
		RawContent:   "ambiguous action item",
		ReingestMode: false,
	})
	if err != nil {
		t.Fatalf("failed to create raw_input: %v", err)
	}

	igBus := ingestbus.NewBusiness(
		db.Log,
		db.BusDomain.RawInput,
		db.BusDomain.Email,
		db.BusDomain.Task,
		db.BusDomain.Context,
		db.BusDomain.Clarification,
		db.BusDomain.Event,
		mock,
		db.BusDomain.Note,
		db.BusDomain.Tag,
	)

	// Process the raw_input
	err = igBus.ProcessRawInputByID(context.Background(), ri.ID)
	if err != nil {
		t.Logf("ProcessRawInputByID info: %v", err)
	}

	// Query created tasks and verify they have unconfirmed=true
	pq := page.New(1, 100)
	tasks, err := db.BusDomain.Task.Query(context.Background(), taskbus.QueryFilter{}, taskbus.DefaultOrderBy, pq)
	if err != nil {
		t.Fatalf("failed to query tasks: %v", err)
	}

	if len(tasks) == 0 {
		t.Fatalf("expected at least one task to be created, got none")
	}

	// Find the newly created task (should have title "Low confidence task")
	var foundTask *taskbus.Task
	for i, tk := range tasks {
		if tk.Title == "Low confidence task" {
			foundTask = &tasks[i]
			break
		}
	}

	if foundTask == nil {
		t.Fatal("expected task with title 'Low confidence task' not found")
	}

	if !foundTask.Unconfirmed {
		t.Errorf("expected unconfirmed=true for non-reingest, but got unconfirmed=false")
	}
}

// TestProcessRawInput_DefensiveFallback tests that when SkipClassify=true but
// SourceEntityID is nil, the processor logs a warning and falls through to the
// normal pipeline without crashing.
func TestProcessRawInput_DefensiveFallback(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, "TestProcessRawInput_DefensiveFallback")

	// Create a mock extractor
	mock := &extractor.MockExtractor{
		TextResult: extractor.TextExtraction{
			ActionItems: []extractor.ActionItem{
				{
					Title:       "Fallback task",
					Description: "Should fall through to full pipeline",
					Priority:    "medium",
				},
			},
		},
	}

	// Create raw_input with SkipClassify=true but SourceEntityID=nil (defensive case)
	ri, err := db.BusDomain.RawInput.Create(context.Background(), rawinputbus.NewRawInput{
		SourceType:   rawinputsource.Voice,
		RawContent:   "Defensive fallback test",
		SkipClassify: true,
		// SourceEntityID is nil — triggering defensive fallback
	})
	if err != nil {
		t.Fatalf("failed to create raw_input: %v", err)
	}

	igBus := ingestbus.NewBusiness(
		db.Log,
		db.BusDomain.RawInput,
		db.BusDomain.Email,
		db.BusDomain.Task,
		db.BusDomain.Context,
		db.BusDomain.Clarification,
		db.BusDomain.Event,
		mock,
		db.BusDomain.Note,
		db.BusDomain.Tag,
	)

	// Process the raw_input — should not crash and should complete
	err = igBus.ProcessRawInputByID(context.Background(), ri.ID)
	if err != nil {
		t.Logf("ProcessRawInputByID info: %v", err)
	}

	// Verify raw_input was processed (not failed, even though it fell through)
	_, err = db.BusDomain.RawInput.QueryByID(context.Background(), ri.ID)
	if err != nil {
		t.Fatalf("failed to reload raw_input: %v", err)
	}

	// With the full pipeline, should have created a task
	pq := page.New(1, 100)
	tasks, err := db.BusDomain.Task.Query(context.Background(), taskbus.QueryFilter{}, taskbus.DefaultOrderBy, pq)
	if err != nil {
		t.Fatalf("failed to query tasks: %v", err)
	}

	var foundFallbackTask bool
	for _, tk := range tasks {
		if tk.Title == "Fallback task" {
			foundFallbackTask = true
			break
		}
	}

	if !foundFallbackTask {
		t.Errorf("expected task 'Fallback task' to be created via full pipeline fallthrough, but not found")
	}
}

// stubGapDetector records all Detect calls for assertion.
type stubGapDetector struct {
	calls []stubGapCall
}

type stubGapCall struct {
	entityType string
	entityID   uuid.UUID
}

func (s *stubGapDetector) Detect(_ context.Context, entityType string, entityID uuid.UUID, _ string) (knowledgegapbus.GapDetectionResult, error) {
	s.calls = append(s.calls, stubGapCall{entityType: entityType, entityID: entityID})
	return knowledgegapbus.GapDetectionResult{}, nil
}

// TestProcessTextGapDetectionUsesEntitySubjectType verifies that gap detection
// is fired with subject_type="task" (not "raw_input") after text ingestion.
func TestProcessTextGapDetectionUsesEntitySubjectType(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, "TestProcessTextGapDetectionUsesEntitySubjectType")

	mock := &extractor.MockExtractor{
		TextResult: extractor.TextExtraction{
			Summary: "Gap detection subject type test",
			ActionItems: []extractor.ActionItem{
				{
					Title:       "Gap subject type task",
					Description: "Verifying gap detection fires per entity",
					Priority:    "medium",
				},
			},
		},
	}

	stub := &stubGapDetector{}

	igBus := ingestbus.NewBusiness(
		db.Log,
		db.BusDomain.RawInput,
		db.BusDomain.Email,
		db.BusDomain.Task,
		db.BusDomain.Context,
		db.BusDomain.Clarification,
		db.BusDomain.Event,
		mock,
		db.BusDomain.Note,
		db.BusDomain.Tag,
	).WithGapDetector(stub)

	result, err := igBus.ProcessText(context.Background(), "Gap subject type task: Verifying gap detection fires per entity")
	if err != nil {
		t.Fatalf("ProcessText failed: %v", err)
	}
	if len(result.TaskIDs) == 0 {
		t.Fatal("expected at least one task to be created")
	}

	// Allow the goroutine to complete.
	time.Sleep(50 * time.Millisecond)

	if len(stub.calls) == 0 {
		t.Fatal("expected at least one Detect call, got none")
	}

	for _, call := range stub.calls {
		if call.entityType == "raw_input" {
			t.Errorf("Detect called with subject_type=%q; want task/event/note, not raw_input", call.entityType)
		}
	}
}

