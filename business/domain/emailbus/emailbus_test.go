package emailbus_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/casebrophy/planner/business/domain/emailbus"
	"github.com/casebrophy/planner/business/domain/rawinputbus"
	"github.com/casebrophy/planner/business/sdk/dbtest"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/sdk/unitest"
)

func Test_Email(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, "Test_Email")
	ctx := context.Background()

	rawInputs, err := rawinputbus.TestSeedRawInputs(ctx, 1, db.BusDomain.RawInput)
	if err != nil {
		t.Fatalf("Seeding raw inputs: %s", err)
	}

	emails, err := emailbus.TestSeedEmails(ctx, 2, rawInputs[0].ID, db.BusDomain.Email)
	if err != nil {
		t.Fatalf("Seeding error: %s", err)
	}

	unitest.Run(t, query(db.BusDomain, emails), "query")
}

func query(busDomain dbtest.BusDomain, emails []emailbus.Email) []unitest.Table {
	return []unitest.Table{
		{
			Name:    "all",
			ExpResp: emails,
			ExcFunc: func(ctx context.Context) any {
				resp, err := busDomain.Email.Query(ctx, emailbus.QueryFilter{}, emailbus.DefaultOrderBy, page.MustParse("1", "10"))
				if err != nil {
					return err
				}
				return resp
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.([]emailbus.Email)
				if !exists {
					return "error occurred"
				}
				expResp := exp.([]emailbus.Email)
				return cmp.Diff(gotResp, expResp,
					cmpopts.EquateApproxTime(time.Second),
					cmpopts.SortSlices(func(a, b emailbus.Email) bool {
						return a.ID.String() < b.ID.String()
					}),
				)
			},
		},
	}
}
