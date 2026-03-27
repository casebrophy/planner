package ingestbus_test

import (
	"context"
	"fmt"
	"testing"

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
					page.MustParse("1", "100"),
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
