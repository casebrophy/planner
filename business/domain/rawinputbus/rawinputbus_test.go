package rawinputbus_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/casebrophy/planner/business/domain/rawinputbus"
	"github.com/casebrophy/planner/business/sdk/dbtest"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/sdk/unitest"
	"github.com/casebrophy/planner/business/types/rawinputsource"
	"github.com/casebrophy/planner/business/types/rawinputstatus"
)

func Test_RawInput(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, "Test_RawInput")

	ris, err := rawinputbus.TestSeedRawInputs(context.Background(), 2, db.BusDomain.RawInput)
	if err != nil {
		t.Fatalf("Seeding error: %s", err)
	}

	unitest.Run(t, query(db.BusDomain, ris), "query")
	unitest.Run(t, create(db.BusDomain), "create")
}

func query(busDomain dbtest.BusDomain, ris []rawinputbus.RawInput) []unitest.Table {
	return []unitest.Table{
		{
			Name:    "all",
			ExpResp: ris,
			ExcFunc: func(ctx context.Context) any {
				resp, err := busDomain.RawInput.Query(ctx, rawinputbus.QueryFilter{}, rawinputbus.DefaultOrderBy, page.MustParse("1", "10"))
				if err != nil {
					return err
				}
				return resp
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.([]rawinputbus.RawInput)
				if !exists {
					return "error occurred"
				}
				expResp := exp.([]rawinputbus.RawInput)
				return cmp.Diff(gotResp, expResp,
					cmpopts.EquateApproxTime(time.Second),
					cmpopts.SortSlices(func(a, b rawinputbus.RawInput) bool {
						return a.ID.String() < b.ID.String()
					}),
				)
			},
		},
		{
			Name:    "byid",
			ExpResp: ris[0],
			ExcFunc: func(ctx context.Context) any {
				resp, err := busDomain.RawInput.QueryByID(ctx, ris[0].ID)
				if err != nil {
					return err
				}
				return resp
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(rawinputbus.RawInput)
				if !exists {
					return "error occurred"
				}
				return cmp.Diff(gotResp, exp.(rawinputbus.RawInput), cmpopts.EquateApproxTime(time.Second))
			},
		},
	}
}

func create(busDomain dbtest.BusDomain) []unitest.Table {
	return []unitest.Table{
		{
			Name: "basic",
			ExpResp: rawinputbus.RawInput{
				SourceType: rawinputsource.Email,
				Status:     rawinputstatus.Pending,
				RawContent: "test raw content",
			},
			ExcFunc: func(ctx context.Context) any {
				nri := rawinputbus.NewRawInput{
					SourceType: rawinputsource.Email,
					RawContent: "test raw content",
				}
				resp, err := busDomain.RawInput.Create(ctx, nri)
				if err != nil {
					return err
				}
				return resp
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(rawinputbus.RawInput)
				if !exists {
					return "error occurred"
				}
				expResp := exp.(rawinputbus.RawInput)
				expResp.ID = gotResp.ID
				expResp.CreatedAt = gotResp.CreatedAt
				return cmp.Diff(gotResp, expResp)
			},
		},
	}
}
