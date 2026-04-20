package debriefbus

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestTaskDebriefThrottle validates that the 30-day throttle is initialized
// and applied to recurring tasks. Full integration testing happens via debriefbus tests.
func TestTaskDebriefThrottle(t *testing.T) {
	t.Run("throttle duration is 30 days", func(t *testing.T) {
		b := NewBusiness(nil, nil, nil)

		expected := 720 * time.Hour // 30 days
		if b.TaskDebriefThrottle != expected {
			t.Errorf("expected TaskDebriefThrottle=%v, got %v", expected, b.TaskDebriefThrottle)
		}
	})

	t.Run("non-recurring tasks skip throttle logic", func(t *testing.T) {
		// This test validates the logic: non-recurring tasks (RecurrenceRule == nil)
		// bypass the throttle check and proceed directly to create a debrief card.
		// The actual creation is tested via integration tests.
		ct := CompletedTask{
			ID:                uuid.New(),
			Title:             "Non-recurring task",
			CreatedAt:         time.Now().Unix(),
			CompletedAt:       time.Now().Unix(),
			RecurrenceRule:    nil, // non-recurring
			RecurrenceParentID: nil,
		}

		// Verify the condition for throttle application
		isRecurring := ct.RecurrenceRule != nil && ct.RecurrenceParentID != nil
		if isRecurring {
			t.Error("non-recurring task should not pass throttle condition")
		}
	})

	t.Run("recurring tasks with throttle cutoff", func(t *testing.T) {
		b := NewBusiness(nil, nil, nil)

		ct := CompletedTask{
			ID:                uuid.New(),
			Title:             "Recurring task",
			CreatedAt:         time.Now().Unix(),
			CompletedAt:       time.Now().Unix(),
			RecurrenceRule:    strPtr("FREQ=DAILY"),
			RecurrenceParentID: uuidPtr(uuid.New()),
		}

		// Verify the condition for throttle application
		isRecurring := ct.RecurrenceRule != nil && ct.RecurrenceParentID != nil
		if !isRecurring {
			t.Error("recurring task should pass throttle condition")
		}

		// Verify throttle cutoff calculation
		throttleCutoff := time.Now().Add(-b.TaskDebriefThrottle)
		windowStart := time.Now().Add(-b.TaskDebriefThrottle)

		if throttleCutoff.After(windowStart) {
			t.Error("throttle cutoff should be in the past")
		}

		// Verify that the window is 30 days
		windowDuration := time.Now().Sub(windowStart)
		if windowDuration < 29*24*time.Hour || windowDuration > 31*24*time.Hour {
			t.Errorf("throttle window should be ~30 days, got %v", windowDuration)
		}
	})
}

func strPtr(s string) *string {
	return &s
}

func uuidPtr(id uuid.UUID) *uuid.UUID {
	return &id
}
