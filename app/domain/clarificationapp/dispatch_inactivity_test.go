package clarificationapp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/domain/contextbus"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/domain/threadbus"
	"github.com/casebrophy/planner/business/sdk/order"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/types/clarificationkind"
	"github.com/casebrophy/planner/business/types/clarificationstatus"
	"github.com/casebrophy/planner/business/types/taskstatus"
)

// spyTaskStorer records writes to Update and returns a predictable task from
// QueryByID so the dispatcher's task-close branch can be exercised.
type spyTaskStorer struct {
	updateCalls []taskbus.Task
}

func (s *spyTaskStorer) Create(context.Context, taskbus.Task) error { return nil }
func (s *spyTaskStorer) CreateWithTx(context.Context, sqlx.ExtContext, taskbus.Task) error { return nil }
func (s *spyTaskStorer) Update(_ context.Context, t taskbus.Task) error {
	s.updateCalls = append(s.updateCalls, t)
	return nil
}
func (s *spyTaskStorer) Delete(context.Context, taskbus.Task) error       { return nil }
func (s *spyTaskStorer) DeleteWithTx(context.Context, sqlx.ExtContext, taskbus.Task) error { return nil }
func (s *spyTaskStorer) DeleteBatch(context.Context, []uuid.UUID) error   { return nil }
func (s *spyTaskStorer) Query(context.Context, taskbus.QueryFilter, order.By, page.Page) ([]taskbus.Task, error) {
	return nil, nil
}
func (s *spyTaskStorer) Count(context.Context, taskbus.QueryFilter) (int, error) {
	return 0, nil
}
func (s *spyTaskStorer) QueryByID(_ context.Context, id uuid.UUID) (taskbus.Task, error) {
	return taskbus.Task{ID: id, Status: taskstatus.Open}, nil
}
func (s *spyTaskStorer) DismissTasksByContext(context.Context, uuid.UUID) (int, error) {
	return 0, nil
}
func (s *spyTaskStorer) DeleteByRawInputUnconfirmed(context.Context, uuid.UUID) error {
	return nil
}
func (s *spyTaskStorer) ResetByContext(context.Context, uuid.UUID) error { return nil }

// fakeTaskDepStorer is a no-op DependencyStorer. QueryDependents must return
// (nil, nil) because taskbus.Update calls UnblockDependents when Status
// transitions to Done.
type fakeTaskDepStorer struct{}

func (fakeTaskDepStorer) AddDependency(context.Context, taskbus.Dependency) error      { return nil }
func (fakeTaskDepStorer) RemoveDependency(context.Context, uuid.UUID, uuid.UUID) error { return nil }
func (fakeTaskDepStorer) QueryDependencies(context.Context, uuid.UUID) ([]taskbus.Task, error) {
	return nil, nil
}
func (fakeTaskDepStorer) QueryDependents(context.Context, uuid.UUID) ([]taskbus.Task, error) {
	return nil, nil
}
func (fakeTaskDepStorer) HasUnmetDependencies(context.Context, uuid.UUID) (bool, error) {
	return false, nil
}

// fakeThreadStorer is a no-op Storer; the dispatcher calls Create indirectly
// via threadBus.AddEntry. Every method stubbed for interface satisfaction.
type fakeThreadStorer struct{}

func (fakeThreadStorer) Create(context.Context, threadbus.ThreadEntry) error { return nil }
func (fakeThreadStorer) Query(context.Context, threadbus.QueryFilter, order.By, page.Page) ([]threadbus.ThreadEntry, error) {
	return nil, nil
}
func (fakeThreadStorer) Count(context.Context, threadbus.QueryFilter) (int, error) {
	return 0, nil
}
func (fakeThreadStorer) QueryByID(context.Context, uuid.UUID) (threadbus.ThreadEntry, error) {
	return threadbus.ThreadEntry{}, nil
}

// When an InactivityPrompt resolves with action="completed" on a task subject,
// the dispatcher must mark the task Done. Regression guard for the FE/BE
// action-value mismatch (previously "close" was silently ignored).
func TestDispatchResolution_InactivityPrompt_Completed_Task_MarksDone(t *testing.T) {
	log := newTestLog()

	taskSpy := &spyTaskStorer{}
	taskBus := taskbus.NewBusiness(log, taskSpy, fakeTaskDepStorer{})
	threadBus := threadbus.NewBusiness(log, fakeThreadStorer{})

	a := &app{
		log:       log,
		taskBus:   taskBus,
		threadBus: threadBus,
	}

	answer := json.RawMessage(`{"action":"completed"}`)
	item := clarificationbus.ClarificationItem{
		ID:          uuid.New(),
		Kind:        clarificationkind.InactivityPrompt,
		Status:      clarificationstatus.Resolved,
		SubjectType: "task",
		SubjectID:   uuid.New(),
		Question:    "Still active?",
		Answer:      &answer,
	}

	a.dispatchResolution(context.Background(), item)

	if len(taskSpy.updateCalls) != 1 {
		t.Fatalf("expected taskBus.Update to be called exactly once, got %d calls", len(taskSpy.updateCalls))
	}
	got := taskSpy.updateCalls[0]
	if got.Status != taskstatus.Done {
		t.Fatalf("expected Status=Done, got %v", got.Status)
	}
}

// Companion case: subject_type="context" with action="completed" must flip the
// context's Status to Closed via contextBus.Update.
func TestDispatchResolution_InactivityPrompt_Completed_Context_MarksClosed(t *testing.T) {
	log := newTestLog()

	ctxSpy := &spyContextStorer{}
	ctxBus := contextbus.NewBusiness(log, ctxSpy)
	threadBus := threadbus.NewBusiness(log, fakeThreadStorer{})

	a := &app{
		log:        log,
		contextBus: ctxBus,
		threadBus:  threadBus,
	}

	answer := json.RawMessage(`{"action":"completed"}`)
	item := clarificationbus.ClarificationItem{
		ID:          uuid.New(),
		Kind:        clarificationkind.InactivityPrompt,
		Status:      clarificationstatus.Resolved,
		SubjectType: "context",
		SubjectID:   uuid.New(),
		Question:    "Still active?",
		Answer:      &answer,
	}

	a.dispatchResolution(context.Background(), item)

	if len(ctxSpy.updateCalls) != 1 {
		t.Fatalf("expected contextBus.Update to be called exactly once, got %d calls", len(ctxSpy.updateCalls))
	}
	got := ctxSpy.updateCalls[0]
	if got.Status != contextbus.Closed {
		t.Fatalf("expected Status=Closed, got %v", got.Status)
	}
}
