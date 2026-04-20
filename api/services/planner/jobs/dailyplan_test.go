package jobs_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/api/services/planner/jobs"
	"github.com/casebrophy/planner/business/domain/contextbus"
	"github.com/casebrophy/planner/business/domain/dailyplanbus"
	"github.com/casebrophy/planner/business/domain/dailyplanbus/generator"
	"github.com/casebrophy/planner/business/domain/eventbus"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/sdk/order"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/types/taskstatus"
)

// -- mocks ------------------------------------------------------------

type mockDPTaskBus struct {
	mu    sync.Mutex
	calls int32
	tasks []taskbus.Task
}

func (m *mockDPTaskBus) Query(_ context.Context, _ taskbus.QueryFilter, _ order.By, _ page.Page) ([]taskbus.Task, error) {
	atomic.AddInt32(&m.calls, 1)
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.tasks, nil
}
func (m *mockDPTaskBus) Calls() int32 { return atomic.LoadInt32(&m.calls) }

type mockCtxBus struct {
	calls int32
}

func (m *mockCtxBus) Query(_ context.Context, _ contextbus.QueryFilter, _ order.By, _ page.Page) ([]contextbus.Context, error) {
	atomic.AddInt32(&m.calls, 1)
	return nil, nil
}

type mockEvtBus struct {
	calls int32
}

func (m *mockEvtBus) Query(_ context.Context, _ eventbus.QueryFilter, _ order.By, _ page.Page) ([]eventbus.Event, error) {
	atomic.AddInt32(&m.calls, 1)
	return nil, nil
}
func (m *mockEvtBus) Calls() int32 { return atomic.LoadInt32(&m.calls) }

type mockDpBus struct {
	mu          sync.Mutex
	createCalls int32
	addItems    int32
}

func (m *mockDpBus) Create(_ context.Context, ne dailyplanbus.NewDailyPlan) (dailyplanbus.DailyPlan, error) {
	atomic.AddInt32(&m.createCalls, 1)
	return dailyplanbus.DailyPlan{
		ID:         uuid.New(),
		PlanDate:   ne.PlanDate,
		Generation: ne.Generation,
		ModelUsed:  ne.ModelUsed,
		CreatedAt:  time.Now(),
	}, nil
}
func (m *mockDpBus) AddItem(_ context.Context, _ dailyplanbus.NewDailyPlanItem) (dailyplanbus.DailyPlanItem, error) {
	atomic.AddInt32(&m.addItems, 1)
	return dailyplanbus.DailyPlanItem{ID: uuid.New()}, nil
}
func (m *mockDpBus) CreateCalls() int32 { return atomic.LoadInt32(&m.createCalls) }
func (m *mockDpBus) AddItemCalls() int32 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return atomic.LoadInt32(&m.addItems)
}

type mockGenerator struct {
	calls     int32
	lastTasks []generator.TaskRef
	output    generator.PlanOutput
}

func (m *mockGenerator) Generate(_ context.Context, tasks []generator.TaskRef, _ []generator.EventRef, _ []generator.CarryoverItem, _ string) (generator.PlanOutput, []generator.ImplicationResult, string, error) {
	atomic.AddInt32(&m.calls, 1)
	m.lastTasks = tasks
	return m.output, nil, "test-model", nil
}
func (m *mockGenerator) Calls() int32 { return atomic.LoadInt32(&m.calls) }

// -- tests ------------------------------------------------------------

func TestDailyPlanJob_TimeMismatchIsNoop(t *testing.T) {
	tb := &mockDPTaskBus{}
	cb := &mockCtxBus{}
	eb := &mockEvtBus{}
	dp := &mockDpBus{}
	gen := &mockGenerator{}

	// Fixed "now" = 09:30, FireAt = 08:00 — mismatch
	fixed := time.Date(2026, 4, 20, 9, 30, 0, 0, time.UTC)

	job := jobs.DailyPlanJob{
		Log:       newTestLogger(),
		Generator: gen,
		TaskBus:   tb,
		CtxBus:    cb,
		EvtBus:    eb,
		DpBus:     dp,
		UserTZ:    time.UTC,
		FireAt:    "08:00",
		Now:       func() time.Time { return fixed },
		Interval:  5 * time.Millisecond,
	}

	runJobFor(t, job.Run, 40*time.Millisecond)

	if tb.Calls() != 0 {
		t.Fatalf("expected no TaskBus.Query calls on time mismatch, got %d", tb.Calls())
	}
	if gen.Calls() != 0 {
		t.Fatalf("expected no Generator.Generate calls on time mismatch, got %d", gen.Calls())
	}
	if dp.CreateCalls() != 0 {
		t.Fatalf("expected no plan Create calls on time mismatch, got %d", dp.CreateCalls())
	}
}

func TestDailyPlanJob_FiresAndAddsItems(t *testing.T) {
	// Build tasks: 2 open, 1 done (filtered), 1 with unrecognized status (filtered)
	openID1 := uuid.New()
	openID2 := uuid.New()
	tb := &mockDPTaskBus{
		tasks: []taskbus.Task{
			{ID: openID1, Title: "a", Status: taskstatus.Open},
			{ID: openID2, Title: "b", Status: taskstatus.Blocked},
			{ID: uuid.New(), Title: "c", Status: taskstatus.Done},
		},
	}
	cb := &mockCtxBus{}
	eb := &mockEvtBus{}
	dp := &mockDpBus{}

	gen := &mockGenerator{
		output: generator.PlanOutput{
			Groups: []generator.PlanGroup{
				{
					Name:   "morning",
					Reason: "start",
					Items: []generator.PlanItem{
						{TaskID: openID1.String(), AIDurationMin: 30, PriorityReason: "urgent"},
						{TaskID: openID2.String(), AIDurationMin: 15, PriorityReason: "next"},
					},
				},
			},
		},
	}

	// Fixed "now" = 08:00, FireAt = 08:00 — match
	fixed := time.Date(2026, 4, 20, 8, 0, 0, 0, time.UTC)

	job := jobs.DailyPlanJob{
		Log:       newTestLogger(),
		Generator: gen,
		TaskBus:   tb,
		CtxBus:    cb,
		EvtBus:    eb,
		DpBus:     dp,
		UserTZ:    time.UTC,
		FireAt:    "08:00",
		Now:       func() time.Time { return fixed },
		Interval:  5 * time.Millisecond,
	}

	runJobFor(t, job.Run, 60*time.Millisecond)

	// 1) Flow fires
	if tb.Calls() < 1 {
		t.Fatalf("expected TaskBus.Query to fire, got %d", tb.Calls())
	}
	if gen.Calls() != 1 {
		t.Fatalf("expected Generator.Generate to fire exactly once (lastGenDate guard), got %d", gen.Calls())
	}
	if dp.CreateCalls() != 1 {
		t.Fatalf("expected plan Create exactly once, got %d", dp.CreateCalls())
	}

	// 2) Filter: done/unknown tasks dropped; only the 2 open/blocked reach generator
	if got := len(gen.lastTasks); got != 2 {
		t.Fatalf("expected 2 taskRefs after status filter, got %d", got)
	}

	// 3) AddItem called once per plan item
	if got := dp.AddItemCalls(); got != 2 {
		t.Fatalf("expected 2 AddItem calls, got %d", got)
	}
}

func TestDailyPlanJob_ContextCancelExits(t *testing.T) {
	job := jobs.DailyPlanJob{
		Log:       newTestLogger(),
		Generator: &mockGenerator{},
		TaskBus:   &mockDPTaskBus{},
		CtxBus:    &mockCtxBus{},
		EvtBus:    &mockEvtBus{},
		DpBus:     &mockDpBus{},
		UserTZ:    time.UTC,
		FireAt:    "08:00",
		Interval:  10 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		job.Run(ctx)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("Run did not return after ctx timeout")
	}
}
