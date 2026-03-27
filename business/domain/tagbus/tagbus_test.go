package tagbus_test

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/casebrophy/planner/business/domain/tagbus"
	"github.com/casebrophy/planner/business/sdk/dbtest"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/sdk/unitest"
)

func Test_Tag(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, "Test_Tag")

	tags, err := tagbus.TestSeedTags(context.Background(), 2, db.BusDomain.Tag)
	if err != nil {
		t.Fatalf("Seeding error: %s", err)
	}

	unitest.Run(t, query(db.BusDomain, tags), "query")
	unitest.Run(t, create(db.BusDomain, tags), "create")
	unitest.Run(t, delete(db.BusDomain, tags), "delete")
}

func query(busDomain dbtest.BusDomain, tags []tagbus.Tag) []unitest.Table {
	return []unitest.Table{
		{
			Name:    "all",
			ExpResp: tags,
			ExcFunc: func(ctx context.Context) any {
				resp, err := busDomain.Tag.Query(ctx, tagbus.QueryFilter{}, tagbus.DefaultOrderBy, page.MustParse("1", "10"))
				if err != nil {
					return err
				}
				return resp
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.([]tagbus.Tag)
				if !exists {
					return "error occurred"
				}
				expResp := exp.([]tagbus.Tag)
				return cmp.Diff(gotResp, expResp,
					cmpopts.SortSlices(func(a, b tagbus.Tag) bool {
						return a.ID.String() < b.ID.String()
					}),
				)
			},
		},
	}
}

func create(busDomain dbtest.BusDomain, _ []tagbus.Tag) []unitest.Table {
	return []unitest.Table{
		{
			Name: "basic",
			ExpResp: tagbus.Tag{
				Name: "newtag",
			},
			ExcFunc: func(ctx context.Context) any {
				nt := tagbus.NewTag{Name: "newtag"}
				resp, err := busDomain.Tag.Create(ctx, nt)
				if err != nil {
					return err
				}
				return resp
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(tagbus.Tag)
				if !exists {
					return "error occurred"
				}
				expResp := exp.(tagbus.Tag)
				expResp.ID = gotResp.ID
				return cmp.Diff(gotResp, expResp)
			},
		},
	}
}

func delete(busDomain dbtest.BusDomain, tags []tagbus.Tag) []unitest.Table {
	return []unitest.Table{
		{
			Name:    "basic",
			ExpResp: nil,
			ExcFunc: func(ctx context.Context) any {
				if err := busDomain.Tag.Delete(ctx, tags[1]); err != nil {
					return err
				}
				return nil
			},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(got, exp)
			},
		},
	}
}
