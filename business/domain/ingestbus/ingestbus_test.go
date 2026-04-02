package ingestbus_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/casebrophy/planner/business/domain/contextbus"
	"github.com/casebrophy/planner/business/domain/ingestbus"
	"github.com/casebrophy/planner/business/domain/ingestbus/extractor"
	"github.com/casebrophy/planner/business/domain/rawinputbus"
	"github.com/casebrophy/planner/business/sdk/dbtest"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/sdk/unitest"
	"github.com/casebrophy/planner/business/types/rawinputsource"
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
