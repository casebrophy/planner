package taskbus_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/sdk/dbtest"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/sdk/unitest"
	"github.com/casebrophy/planner/business/types/debriefstatus"
	"github.com/casebrophy/planner/business/types/taskenergy"
	"github.com/casebrophy/planner/business/types/taskpriority"
	"github.com/casebrophy/planner/business/types/taskstatus"
)

func Test_Task(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, "Test_Task")

	tasks, err := taskbus.TestSeedTasks(context.Background(), 2, db.BusDomain.Task)
	if err != nil {
		t.Fatalf("Seeding error: %s", err)
	}

	unitest.Run(t, query(db.BusDomain, tasks), "query")
	unitest.Run(t, create(db.BusDomain, tasks), "create")
	unitest.Run(t, update(db.BusDomain, tasks), "update")
	unitest.Run(t, delete(db.BusDomain, tasks), "delete")
}

func Test_Task_HasRecurrence(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, "Test_Task_HasRecurrence")
	ctx := context.Background()

	// Create a task WITHOUT recurrence.
	noRecur, err := db.BusDomain.Task.Create(ctx, taskbus.NewTask{
		Title:       "No Recurrence",
		Description: "Task without recurrence",
		Status:      taskstatus.Open,
		Priority:    taskpriority.Medium,
		Energy:      taskenergy.Medium,
	})
	if err != nil {
		t.Fatalf("creating non-recurring task: %s", err)
	}

	// Create a task WITH recurrence.
	rule := "FREQ=WEEKLY;BYDAY=MO"
	recurTask, err := db.BusDomain.Task.Create(ctx, taskbus.NewTask{
		Title:          "Has Recurrence",
		Description:    "Task with recurrence",
		Status:         taskstatus.Open,
		Priority:       taskpriority.High,
		Energy:         taskenergy.Low,
		RecurrenceRule: &rule,
	})
	if err != nil {
		t.Fatalf("creating recurring task: %s", err)
	}

	cmpOpts := cmp.Options{
		cmpopts.EquateApproxTime(time.Second),
		cmpopts.EquateComparable(taskstatus.Status{}, taskpriority.Priority{}, taskenergy.Energy{}, debriefstatus.Status{}),
		cmpopts.SortSlices(func(a, b taskbus.Task) bool {
			return a.ID.String() < b.ID.String()
		}),
	}

	t.Run("has_recurrence=true", func(t *testing.T) {
		hasRecur := true
		filter := taskbus.QueryFilter{HasRecurrence: &hasRecur}

		got, err := db.BusDomain.Task.Query(ctx, filter, taskbus.DefaultOrderBy, page.New(1, 10))
		if err != nil {
			t.Fatalf("querying has_recurrence=true: %s", err)
		}

		if len(got) != 1 {
			t.Fatalf("expected 1 recurring task, got %d", len(got))
		}

		if diff := cmp.Diff(got[0], recurTask, cmpOpts...); diff != "" {
			t.Fatalf("recurring task mismatch:\n%s", diff)
		}
	})

	t.Run("has_recurrence=false", func(t *testing.T) {
		hasRecur := false
		filter := taskbus.QueryFilter{HasRecurrence: &hasRecur}

		got, err := db.BusDomain.Task.Query(ctx, filter, taskbus.DefaultOrderBy, page.New(1, 10))
		if err != nil {
			t.Fatalf("querying has_recurrence=false: %s", err)
		}

		if len(got) != 1 {
			t.Fatalf("expected 1 non-recurring task, got %d", len(got))
		}

		if diff := cmp.Diff(got[0], noRecur, cmpOpts...); diff != "" {
			t.Fatalf("non-recurring task mismatch:\n%s", diff)
		}
	})

	t.Run("no_filter_returns_both", func(t *testing.T) {
		got, err := db.BusDomain.Task.Query(ctx, taskbus.QueryFilter{}, taskbus.DefaultOrderBy, page.New(1, 10))
		if err != nil {
			t.Fatalf("querying without filter: %s", err)
		}

		if len(got) != 2 {
			t.Fatalf("expected 2 tasks, got %d", len(got))
		}
	})
}

func Test_Task_CreateWithRawInputID(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, "Test_Task_CreateWithRawInputID")
	ctx := context.Background()

	rawInputID := uuid.New()

	task, err := db.BusDomain.Task.Create(ctx, taskbus.NewTask{
		Title:       "Task with RawInputID",
		Description: "Created from raw input",
		Status:      taskstatus.Open,
		Priority:    taskpriority.Medium,
		Energy:      taskenergy.Medium,
		RawInputID:  &rawInputID,
	})
	if err != nil {
		t.Fatalf("creating task with RawInputID: %s", err)
	}

	if task.RawInputID == nil {
		t.Fatalf("expected RawInputID to be set")
	}
	if *task.RawInputID != rawInputID {
		t.Fatalf("expected RawInputID %s, got %s", rawInputID, *task.RawInputID)
	}

	retrieved, err := db.BusDomain.Task.QueryByID(ctx, task.ID)
	if err != nil {
		t.Fatalf("querying task by ID: %s", err)
	}

	if retrieved.RawInputID == nil {
		t.Fatalf("expected RawInputID to persist on query")
	}
	if *retrieved.RawInputID != rawInputID {
		t.Fatalf("expected RawInputID %s, got %s", rawInputID, *retrieved.RawInputID)
	}
}

func query(busDomain dbtest.BusDomain, tasks []taskbus.Task) []unitest.Table {
	return []unitest.Table{
		{
			Name:    "all",
			ExpResp: tasks,
			ExcFunc: func(ctx context.Context) any {
				resp, err := busDomain.Task.Query(ctx, taskbus.QueryFilter{}, taskbus.DefaultOrderBy, page.New(1, 10))
				if err != nil {
					return err
				}
				return resp
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.([]taskbus.Task)
				if !exists {
					return "error occurred"
				}
				expResp := exp.([]taskbus.Task)
				return cmp.Diff(gotResp, expResp,
					cmpopts.EquateApproxTime(time.Second),
					cmpopts.EquateComparable(taskstatus.Status{}, taskpriority.Priority{}, taskenergy.Energy{}, debriefstatus.Status{}),
					cmpopts.SortSlices(func(a, b taskbus.Task) bool {
						return a.ID.String() < b.ID.String()
					}),
				)
			},
		},
		{
			Name:    "byid",
			ExpResp: tasks[0],
			ExcFunc: func(ctx context.Context) any {
				resp, err := busDomain.Task.QueryByID(ctx, tasks[0].ID)
				if err != nil {
					return err
				}
				return resp
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(taskbus.Task)
				if !exists {
					return "error occurred"
				}
				return cmp.Diff(gotResp, exp.(taskbus.Task),
					cmpopts.EquateApproxTime(time.Second),
					cmpopts.EquateComparable(taskstatus.Status{}, taskpriority.Priority{}, taskenergy.Energy{}, debriefstatus.Status{}),
				)
			},
		},
	}
}

func create(busDomain dbtest.BusDomain, _ []taskbus.Task) []unitest.Table {
	return []unitest.Table{
		{
			Name: "basic",
			ExpResp: taskbus.Task{
				Title:         "New Task",
				Description:   "New Description",
				Status:        taskstatus.Open,
				Priority:      taskpriority.High,
				Energy:        taskenergy.Low,
				DebriefStatus: debriefstatus.Pending,
			},
			ExcFunc: func(ctx context.Context) any {
				nt := taskbus.NewTask{
					Title:       "New Task",
					Description: "New Description",
					Status:      taskstatus.Open,
					Priority:    taskpriority.High,
					Energy:      taskenergy.Low,
				}
				resp, err := busDomain.Task.Create(ctx, nt)
				if err != nil {
					return err
				}
				return resp
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(taskbus.Task)
				if !exists {
					return "error occurred"
				}
				expResp := exp.(taskbus.Task)
				expResp.ID = gotResp.ID
				expResp.CreatedAt = gotResp.CreatedAt
				expResp.UpdatedAt = gotResp.UpdatedAt
				return cmp.Diff(gotResp, expResp,
					cmpopts.EquateComparable(taskstatus.Status{}, taskpriority.Priority{}, taskenergy.Energy{}, debriefstatus.Status{}),
				)
			},
		},
	}
}

func update(busDomain dbtest.BusDomain, tasks []taskbus.Task) []unitest.Table {
	newTitle := "Updated Title"
	newStatus := taskstatus.Blocked

	return []unitest.Table{
		{
			Name: "basic",
			ExpResp: taskbus.Task{
				ID:            tasks[0].ID,
				Title:         "Updated Title",
				Description:   tasks[0].Description,
				Status:        taskstatus.Blocked,
				Priority:      tasks[0].Priority,
				Energy:        tasks[0].Energy,
				DebriefStatus: tasks[0].DebriefStatus,
				CreatedAt:     tasks[0].CreatedAt,
			},
			ExcFunc: func(ctx context.Context) any {
				ut := taskbus.UpdateTask{
					Title:  &newTitle,
					Status: &newStatus,
				}
				resp, err := busDomain.Task.Update(ctx, tasks[0], ut)
				if err != nil {
					return err
				}
				return resp
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(taskbus.Task)
				if !exists {
					return "error occurred"
				}
				expResp := exp.(taskbus.Task)
				expResp.UpdatedAt = gotResp.UpdatedAt
				return cmp.Diff(gotResp, expResp,
					cmpopts.EquateApproxTime(time.Second),
					cmpopts.EquateComparable(taskstatus.Status{}, taskpriority.Priority{}, taskenergy.Energy{}, debriefstatus.Status{}),
				)
			},
		},
	}
}

func delete(busDomain dbtest.BusDomain, tasks []taskbus.Task) []unitest.Table {
	return []unitest.Table{
		{
			Name:    "basic",
			ExpResp: nil,
			ExcFunc: func(ctx context.Context) any {
				if err := busDomain.Task.Delete(ctx, tasks[1]); err != nil {
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
