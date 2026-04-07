package noteapi_test

import (
	"fmt"
	"net/http"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/casebrophy/planner/app/domain/noteapp"
	"github.com/casebrophy/planner/app/sdk/apitest"
	"github.com/casebrophy/planner/app/sdk/query"
	"github.com/casebrophy/planner/business/domain/notebus"
)

func queryByTaskID200(sd seedData) []apitest.Table {
	// notes[0] and notes[1] are linked to tasks[0]; notes[2] is unlinked.
	taskID := sd.tasks[0].ID.String()
	return []apitest.Table{
		{
			Name:       "by-task-id",
			URL:        fmt.Sprintf("/api/v1/notes?task_id=%s&page=1&rows=10", taskID),
			Token:      apitest.TestAPIKey,
			Method:     http.MethodGet,
			StatusCode: http.StatusOK,
			GotResp:    &query.Result[noteapp.Note]{},
			ExpResp: &query.Result[noteapp.Note]{
				Total:       2,
				Page:        1,
				RowsPerPage: 10,
				Items:       toAppNotes(sd.notes[:2]),
			},
			CmpFunc: func(got any, exp any) string {
				gotResp := got.(*query.Result[noteapp.Note])
				expResp := exp.(*query.Result[noteapp.Note])

				if gotResp.Total != expResp.Total {
					return fmt.Sprintf("total: got %d, exp %d", gotResp.Total, expResp.Total)
				}

				clearTimes := func(items []noteapp.Note) {
					for i := range items {
						items[i].CreatedAt = ""
						items[i].UpdatedAt = ""
					}
				}
				clearTimes(gotResp.Items)
				clearTimes(expResp.Items)

				return cmp.Diff(gotResp, expResp,
					cmpopts.SortSlices(func(a, b noteapp.Note) bool {
						return a.ID < b.ID
					}),
				)
			},
		},
	}
}

func queryAll200(sd seedData) []apitest.Table {
	return []apitest.Table{
		{
			Name:       "all",
			URL:        "/api/v1/notes?page=1&rows=10",
			Token:      apitest.TestAPIKey,
			Method:     http.MethodGet,
			StatusCode: http.StatusOK,
			GotResp:    &query.Result[noteapp.Note]{},
			ExpResp: &query.Result[noteapp.Note]{
				Total:       3,
				Page:        1,
				RowsPerPage: 10,
				Items:       toAppNotes(sd.notes),
			},
			CmpFunc: func(got any, exp any) string {
				gotResp := got.(*query.Result[noteapp.Note])
				expResp := exp.(*query.Result[noteapp.Note])

				if gotResp.Total != expResp.Total {
					return fmt.Sprintf("total: got %d, exp %d", gotResp.Total, expResp.Total)
				}

				clearTimes := func(items []noteapp.Note) {
					for i := range items {
						items[i].CreatedAt = ""
						items[i].UpdatedAt = ""
					}
				}
				clearTimes(gotResp.Items)
				clearTimes(expResp.Items)

				return cmp.Diff(gotResp, expResp,
					cmpopts.SortSlices(func(a, b noteapp.Note) bool {
						return a.ID < b.ID
					}),
				)
			},
		},
	}
}

func toAppNotes(notes []notebus.Note) []noteapp.Note {
	appNotes := make([]noteapp.Note, len(notes))
	for i, n := range notes {
		an := noteapp.Note{
			ID:      n.ID.String(),
			Content: n.Content,
			Source:  n.Source,
		}
		if n.TaskID != nil {
			s := n.TaskID.String()
			an.TaskID = &s
		}
		if n.ContextID != nil {
			s := n.ContextID.String()
			an.ContextID = &s
		}
		if n.RawInputID != nil {
			s := n.RawInputID.String()
			an.RawInputID = &s
		}
		appNotes[i] = an
	}
	return appNotes
}
