package taskapp

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/app/sdk/mid"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/types/debriefstatus"
	"github.com/casebrophy/planner/business/types/taskenergy"
	"github.com/casebrophy/planner/business/types/taskpriority"
	"github.com/casebrophy/planner/business/types/taskstatus"
)

type Task struct {
	ID                 string   `json:"id"`
	ContextID          *string  `json:"contextId,omitempty"`
	Title              string   `json:"title"`
	Description        string   `json:"description"`
	Status             string   `json:"status"`
	Priority           string   `json:"priority"`
	Energy             string   `json:"energy"`
	DurationMin        *int     `json:"durationMin,omitempty"`
	DueDate            *string  `json:"dueDate,omitempty"`
	ScheduledAt        *string  `json:"scheduledAt,omitempty"`
	ExpectedUpdateDays *float64 `json:"expectedUpdateDays,omitempty"`
	LastThreadAt       *string  `json:"lastThreadAt,omitempty"`
	BlockedReason      string   `json:"blockedReason,omitempty"`
	DebriefStatus      string   `json:"debriefStatus"`
	CreatedAt          string   `json:"createdAt"`
	UpdatedAt          string   `json:"updatedAt"`
	CompletedAt        *string  `json:"completedAt,omitempty"`
	RecurrenceRule     *string  `json:"recurrenceRule,omitempty"`
	RecurrenceParentID *string  `json:"recurrenceParentId,omitempty"`
	TrackOutcome       bool     `json:"trackOutcome"`
	Unconfirmed        bool     `json:"unconfirmed"`
}

func (t Task) Encode() ([]byte, string, error) {
	data, err := json.Marshal(t)
	return data, "application/json", err
}

// CompletionInfo implements mid.Completable so the activity log middleware
// can automatically record task completions.
func (t Task) CompletionInfo() (mid.CompletionDetail, bool) {
	if t.Status != "done" {
		return mid.CompletionDetail{}, false
	}

	d := mid.CompletionDetail{
		SubjectType: "task",
		SubjectID:   t.ID,
	}
	if t.RecurrenceParentID != nil {
		d.ParentID = *t.RecurrenceParentID
	}

	return d, true
}

type NewTask struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	ContextID   *string `json:"contextId"`
	Priority    string  `json:"priority"`
	Energy      string  `json:"energy"`
	DurationMin *int    `json:"durationMin"`
	DueDate        *string `json:"dueDate"`
	RecurrenceRule *string `json:"recurrenceRule"`
}

type UpdateTask struct {
	Title              *string  `json:"title"`
	Description        *string  `json:"description"`
	ContextID          *string  `json:"contextId"`
	Status             *string  `json:"status"`
	Priority           *string  `json:"priority"`
	Energy             *string  `json:"energy"`
	DurationMin        *int     `json:"durationMin"`
	DueDate            *string  `json:"dueDate"`
	ScheduledAt        *string  `json:"scheduledAt"`
	ExpectedUpdateDays *float64 `json:"expectedUpdateDays"`
	BlockedReason      *string  `json:"blockedReason"`
	DebriefStatus      *string  `json:"debriefStatus"`
	RecurrenceRule     *string  `json:"recurrenceRule"`
	TrackOutcome       *bool    `json:"trackOutcome,omitempty"`
}

func toAppTask(t taskbus.Task) Task {
	at := Task{
		ID:                 t.ID.String(),
		Title:              t.Title,
		Description:        t.Description,
		Status:             t.Status.String(),
		Priority:           t.Priority.String(),
		Energy:             t.Energy.String(),
		DurationMin:        t.DurationMin,
		ExpectedUpdateDays: t.ExpectedUpdateDays,
		BlockedReason:      t.BlockedReason,
		DebriefStatus:      t.DebriefStatus.String(),
		CreatedAt:          t.CreatedAt.Format(time.RFC3339),
		UpdatedAt:          t.UpdatedAt.Format(time.RFC3339),
	}

	if t.ContextID != nil {
		s := t.ContextID.String()
		at.ContextID = &s
	}
	if t.DueDate != nil {
		s := t.DueDate.Format(time.RFC3339)
		at.DueDate = &s
	}
	if t.ScheduledAt != nil {
		s := t.ScheduledAt.Format(time.RFC3339)
		at.ScheduledAt = &s
	}
	if t.LastThreadAt != nil {
		s := t.LastThreadAt.Format(time.RFC3339)
		at.LastThreadAt = &s
	}
	if t.CompletedAt != nil {
		s := t.CompletedAt.Format(time.RFC3339)
		at.CompletedAt = &s
	}
	at.RecurrenceRule = t.RecurrenceRule
	if t.RecurrenceParentID != nil {
		s := t.RecurrenceParentID.String()
		at.RecurrenceParentID = &s
	}
	at.TrackOutcome = t.TrackOutcome
	at.Unconfirmed = t.Unconfirmed

	return at
}

func toAppTasks(ts []taskbus.Task) []Task {
	tasks := make([]Task, len(ts))
	for i, t := range ts {
		tasks[i] = toAppTask(t)
	}
	return tasks
}

func toBusNewTask(nt NewTask) (taskbus.NewTask, error) {
	priority := taskpriority.Medium
	if nt.Priority != "" {
		var err error
		priority, err = taskpriority.Parse(nt.Priority)
		if err != nil {
			return taskbus.NewTask{}, fmt.Errorf("priority: %w", err)
		}
	}

	energy := taskenergy.Medium
	if nt.Energy != "" {
		var err error
		energy, err = taskenergy.Parse(nt.Energy)
		if err != nil {
			return taskbus.NewTask{}, fmt.Errorf("energy: %w", err)
		}
	}

	bt := taskbus.NewTask{
		Title:          nt.Title,
		Description:    nt.Description,
		Status:         taskstatus.Open,
		Priority:       priority,
		Energy:         energy,
		DurationMin:    nt.DurationMin,
		RecurrenceRule: nt.RecurrenceRule,
	}

	if nt.ContextID != nil {
		id, err := uuid.Parse(*nt.ContextID)
		if err != nil {
			return taskbus.NewTask{}, fmt.Errorf("contextId: %w", err)
		}
		bt.ContextID = &id
	}

	if nt.DueDate != nil {
		t, err := time.Parse(time.RFC3339, *nt.DueDate)
		if err != nil {
			return taskbus.NewTask{}, fmt.Errorf("dueDate: %w", err)
		}
		bt.DueDate = &t
	}

	return bt, nil
}

func toBusUpdateTask(ut UpdateTask) (taskbus.UpdateTask, error) {
	var but taskbus.UpdateTask

	but.Title = ut.Title
	but.Description = ut.Description
	but.DurationMin = ut.DurationMin

	if ut.Status != nil {
		s, err := taskstatus.Parse(*ut.Status)
		if err != nil {
			return taskbus.UpdateTask{}, fmt.Errorf("status: %w", err)
		}
		but.Status = &s
	}

	if ut.Priority != nil {
		p, err := taskpriority.Parse(*ut.Priority)
		if err != nil {
			return taskbus.UpdateTask{}, fmt.Errorf("priority: %w", err)
		}
		but.Priority = &p
	}

	if ut.Energy != nil {
		e, err := taskenergy.Parse(*ut.Energy)
		if err != nil {
			return taskbus.UpdateTask{}, fmt.Errorf("energy: %w", err)
		}
		but.Energy = &e
	}

	if ut.ContextID != nil {
		id, err := uuid.Parse(*ut.ContextID)
		if err != nil {
			return taskbus.UpdateTask{}, fmt.Errorf("contextId: %w", err)
		}
		but.ContextID = &id
	}

	if ut.DueDate != nil {
		t, err := time.Parse(time.RFC3339, *ut.DueDate)
		if err != nil {
			return taskbus.UpdateTask{}, fmt.Errorf("dueDate: %w", err)
		}
		but.DueDate = &t
	}

	if ut.ScheduledAt != nil {
		t, err := time.Parse(time.RFC3339, *ut.ScheduledAt)
		if err != nil {
			return taskbus.UpdateTask{}, fmt.Errorf("scheduledAt: %w", err)
		}
		but.ScheduledAt = &t
	}

	but.ExpectedUpdateDays = ut.ExpectedUpdateDays
	but.RecurrenceRule = ut.RecurrenceRule
	but.TrackOutcome = ut.TrackOutcome

	if ut.BlockedReason != nil {
		but.BlockedReason = ut.BlockedReason
	}

	if ut.DebriefStatus != nil {
		ds, err := debriefstatus.Parse(*ut.DebriefStatus)
		if err != nil {
			return taskbus.UpdateTask{}, fmt.Errorf("debriefStatus: %w", err)
		}
		but.DebriefStatus = &ds
	}

	return but, nil
}
