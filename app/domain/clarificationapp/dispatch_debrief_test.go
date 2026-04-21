package clarificationapp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/domain/contextbus"
	"github.com/casebrophy/planner/business/domain/observationbus"
	"github.com/casebrophy/planner/business/sdk/order"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/types/clarificationkind"
	"github.com/casebrophy/planner/business/types/clarificationstatus"
	"github.com/casebrophy/planner/business/types/debriefstatus"
	"github.com/casebrophy/planner/foundation/logger"
)

// fakeClarStorer returns an error from Count to simulate a DB failure on the
// debrief completion check.
type fakeClarStorer struct {
	countErr error
}

func (f *fakeClarStorer) Create(context.Context, clarificationbus.ClarificationItem) error {
	return nil
}
func (f *fakeClarStorer) Upsert(context.Context, clarificationbus.ClarificationItem) (clarificationbus.ClarificationItem, error) {
	return clarificationbus.ClarificationItem{}, nil
}
func (f *fakeClarStorer) Update(context.Context, clarificationbus.ClarificationItem) error {
	return nil
}
func (f *fakeClarStorer) Query(context.Context, clarificationbus.QueryFilter, order.By, page.Page) ([]clarificationbus.ClarificationItem, error) {
	return nil, nil
}
func (f *fakeClarStorer) Count(context.Context, clarificationbus.QueryFilter) (int, error) {
	return 0, f.countErr
}
func (f *fakeClarStorer) QueryByID(context.Context, uuid.UUID) (clarificationbus.ClarificationItem, error) {
	return clarificationbus.ClarificationItem{}, nil
}
func (f *fakeClarStorer) UnsnoozeExpired(context.Context, time.Time) (int, error) {
	return 0, nil
}

type fakeObsStorer struct{}

func (fakeObsStorer) Create(context.Context, observationbus.Observation) error { return nil }
func (fakeObsStorer) Query(context.Context, observationbus.QueryFilter, order.By, page.Page) ([]observationbus.Observation, error) {
	return nil, nil
}
func (fakeObsStorer) Count(context.Context, observationbus.QueryFilter) (int, error) {
	return 0, nil
}

// spyContextStorer records whether Update was called so the test can assert
// debrief completion was (or wasn't) triggered.
type spyContextStorer struct {
	updateCalls []contextbus.Context
}

func (s *spyContextStorer) Create(context.Context, contextbus.Context) error { return nil }
func (s *spyContextStorer) Update(_ context.Context, c contextbus.Context) error {
	s.updateCalls = append(s.updateCalls, c)
	return nil
}
func (s *spyContextStorer) Delete(context.Context, contextbus.Context) error { return nil }
func (s *spyContextStorer) Query(context.Context, contextbus.QueryFilter, order.By, page.Page) ([]contextbus.Context, error) {
	return nil, nil
}
func (s *spyContextStorer) Count(context.Context, contextbus.QueryFilter) (int, error) {
	return 0, nil
}
func (s *spyContextStorer) QueryByID(_ context.Context, id uuid.UUID) (contextbus.Context, error) {
	return contextbus.Context{ID: id, DebriefStatus: debriefstatus.Pending}, nil
}

func newTestLog() *logger.Logger {
	return logger.New(&bytes.Buffer{}, logger.LevelError, "test")
}

// When clarificationBus.Count fails, the dispatcher must log the error and
// skip marking the context's debrief as done — otherwise a transient DB blip
// would falsely complete the debrief.
func TestDispatchResolution_ContextDebrief_CountError_SkipsCompletion(t *testing.T) {
	log := newTestLog()

	clarStore := &fakeClarStorer{countErr: errors.New("db is down")}
	clarBus := clarificationbus.NewBusiness(log, clarStore)

	obsBus := observationbus.NewBusiness(log, fakeObsStorer{})

	ctxSpy := &spyContextStorer{}
	ctxBus := contextbus.NewBusiness(log, ctxSpy)

	a := &app{
		log:              log,
		clarificationBus: clarBus,
		contextBus:       ctxBus,
		observationBus:   obsBus,
	}

	answer := json.RawMessage(`{"response":"reflection text"}`)
	item := clarificationbus.ClarificationItem{
		ID:          uuid.New(),
		Kind:        clarificationkind.ContextDebrief,
		Status:      clarificationstatus.Resolved,
		SubjectType: "context",
		SubjectID:   uuid.New(),
		Question:    "How did it go?",
		Answer:      &answer,
	}

	a.dispatchResolution(context.Background(), item)

	if len(ctxSpy.updateCalls) != 0 {
		t.Fatalf("expected contextBus.Update NOT to be called when Count fails, got %d calls", len(ctxSpy.updateCalls))
	}
}

// Sanity: when Count succeeds and returns zero, the dispatcher marks debrief done.
// Pins the happy path so a regression in the error branch can't silently break it.
func TestDispatchResolution_ContextDebrief_CountZero_MarksDone(t *testing.T) {
	log := newTestLog()

	clarBus := clarificationbus.NewBusiness(log, &fakeClarStorer{countErr: nil})
	obsBus := observationbus.NewBusiness(log, fakeObsStorer{})
	ctxSpy := &spyContextStorer{}
	ctxBus := contextbus.NewBusiness(log, ctxSpy)

	a := &app{
		log:              log,
		clarificationBus: clarBus,
		contextBus:       ctxBus,
		observationBus:   obsBus,
	}

	answer := json.RawMessage(`{"response":"reflection text"}`)
	item := clarificationbus.ClarificationItem{
		ID:          uuid.New(),
		Kind:        clarificationkind.ContextDebrief,
		Status:      clarificationstatus.Resolved,
		SubjectType: "context",
		SubjectID:   uuid.New(),
		Question:    "How did it go?",
		Answer:      &answer,
	}

	a.dispatchResolution(context.Background(), item)

	if len(ctxSpy.updateCalls) != 1 {
		t.Fatalf("expected contextBus.Update to be called exactly once, got %d calls", len(ctxSpy.updateCalls))
	}
	got := ctxSpy.updateCalls[0]
	if got.DebriefStatus != debriefstatus.Done {
		t.Fatalf("expected DebriefStatus=Done, got %v", got.DebriefStatus)
	}
}
