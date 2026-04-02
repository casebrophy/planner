package scheduleapp

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"time"

	"github.com/casebrophy/planner/app/sdk/errs"
	"github.com/casebrophy/planner/business/domain/eventbus"
	"github.com/casebrophy/planner/business/domain/timeblockbus"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/foundation/web"
)

type app struct {
	eventBus     *eventbus.Business
	timeBlockBus *timeblockbus.Business
}

// ScheduleItem represents either an event or time block in a merged schedule.
type ScheduleItem struct {
	Type      string  `json:"type"` // "event" or "time_block"
	ID        string  `json:"id"`
	Title     string  `json:"title"`
	StartsAt  string  `json:"startsAt"`
	EndsAt    string  `json:"endsAt"`
	AllDay    bool    `json:"allDay,omitempty"`
	Location  *string `json:"location,omitempty"`
	TaskID    *string `json:"taskId,omitempty"`
	Confirmed *bool   `json:"confirmed,omitempty"`
}

type scheduleResponse struct {
	Items []ScheduleItem `json:"items"`
}

func (r scheduleResponse) Encode() ([]byte, string, error) {
	data, err := json.Marshal(r)
	return data, "application/json", err
}

func (a *app) querySchedule(ctx context.Context, r *http.Request) web.Encoder {
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")

	if startStr == "" || endStr == "" {
		return errs.Newf(errs.InvalidArgument, "start and end query parameters are required")
	}

	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		return errs.Newf(errs.InvalidArgument, "invalid start: %s", err)
	}

	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		return errs.Newf(errs.InvalidArgument, "invalid end: %s", err)
	}

	bigPage := page.MustParse("1", "200")

	// Fetch events and time blocks in the date range.
	events, err := a.eventBus.Query(ctx, eventbus.QueryFilter{
		DateFrom: &start,
		DateTo:   &end,
	}, eventbus.DefaultOrderBy, bigPage)
	if err != nil {
		return errs.Newf(errs.Internal, "query events: %s", err)
	}

	blocks, err := a.timeBlockBus.Query(ctx, timeblockbus.QueryFilter{
		DateFrom: &start,
		DateTo:   &end,
	}, timeblockbus.DefaultOrderBy, bigPage)
	if err != nil {
		return errs.Newf(errs.Internal, "query time blocks: %s", err)
	}

	// Merge into unified list.
	items := make([]ScheduleItem, 0, len(events)+len(blocks))

	for _, e := range events {
		item := ScheduleItem{
			Type:     "event",
			ID:       e.ID.String(),
			Title:    e.Title,
			StartsAt: e.StartsAt.Format(time.RFC3339),
			EndsAt:   e.EndsAt.Format(time.RFC3339),
			AllDay:   e.AllDay,
			Location: e.Location,
		}
		items = append(items, item)
	}

	for _, tb := range blocks {
		taskID := tb.TaskID.String()
		confirmed := tb.Confirmed
		item := ScheduleItem{
			Type:      "time_block",
			ID:        tb.ID.String(),
			Title:     "", // Frontend resolves task title from taskId
			StartsAt:  tb.StartsAt.Format(time.RFC3339),
			EndsAt:    tb.EndsAt.Format(time.RFC3339),
			TaskID:    &taskID,
			Confirmed: &confirmed,
		}
		items = append(items, item)
	}

	// Sort by starts_at.
	sort.Slice(items, func(i, j int) bool {
		return items[i].StartsAt < items[j].StartsAt
	})

	return scheduleResponse{Items: items}
}
