package mcpapp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/domain/contextbus"
	"github.com/casebrophy/planner/business/domain/dailyplanbus"
	"github.com/casebrophy/planner/business/domain/debriefbus"
	"github.com/casebrophy/planner/business/domain/emailbus"
	"github.com/casebrophy/planner/business/domain/eventbus"
	"github.com/casebrophy/planner/business/domain/timeblockbus"
	"github.com/casebrophy/planner/business/domain/observationbus"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/domain/threadbus"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/sdk/sqldb"
	"github.com/casebrophy/planner/business/types/clarificationkind"
	"github.com/casebrophy/planner/business/types/clarificationstatus"
	"github.com/casebrophy/planner/business/types/observationkind"
	"github.com/casebrophy/planner/business/types/taskenergy"
	"github.com/casebrophy/planner/business/types/taskpriority"
	"github.com/casebrophy/planner/business/types/taskstatus"
	"github.com/casebrophy/planner/business/types/threadentrykind"
	"github.com/casebrophy/planner/business/types/threadsource"
	"github.com/casebrophy/planner/foundation/web"
)

type app struct {
	taskBus          *taskbus.Business
	contextBus       *contextbus.Business
	emailBus         *emailbus.Business
	eventBus         *eventbus.Business
	timeBlockBus     *timeblockbus.Business
	clarificationBus *clarificationbus.Business
	threadBus        *threadbus.Business
	observationBus   *observationbus.Business
	debriefBus       *debriefbus.Business
	dailyPlanBus     *dailyplanbus.Business
}

func (a *app) handle(ctx context.Context, r *http.Request) web.Encoder {
	var req rpcRequest
	if err := web.Decode(r, &req); err != nil {
		return rpcResponse{
			JSONRPC: "2.0",
			ID:      nil,
			Error:   &rpcError{Code: -32700, Message: "parse error"},
		}
	}

	switch req.Method {
	case "initialize":
		return rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: initializeResult{
				ProtocolVersion: "2025-03-26",
				ServerInfo:      serverInfo{Name: "planner", Version: "0.1.0"},
				Capabilities:    map[string]any{"tools": map[string]any{}},
			},
		}

	case "notifications/initialized":
		return rpcResponse{JSONRPC: "2.0", ID: req.ID}

	case "tools/list":
		return rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  map[string]any{"tools": tools},
		}

	case "tools/call":
		var params toolCallParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return rpcResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &rpcError{Code: -32602, Message: "invalid params"},
			}
		}

		result, err := a.callTool(ctx, params)
		if err != nil {
			return rpcResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result:  toolResult{Content: []toolContent{{Type: "text", Text: err.Error()}}, IsError: true},
			}
		}

		return rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  result,
		}

	default:
		return rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &rpcError{Code: -32601, Message: fmt.Sprintf("method not found: %s", req.Method)},
		}
	}
}

func (a *app) callTool(ctx context.Context, params toolCallParams) (toolResult, error) {
	switch params.Name {
	case "create_task":
		return a.toolCreateTask(ctx, params.Arguments)
	case "list_tasks":
		return a.toolListTasks(ctx, params.Arguments)
	case "get_task":
		return a.toolGetTask(ctx, params.Arguments)
	case "update_task":
		return a.toolUpdateTask(ctx, params.Arguments)
	case "complete_task":
		return a.toolCompleteTask(ctx, params.Arguments)
	case "create_context":
		return a.toolCreateContext(ctx, params.Arguments)
	case "get_context":
		return a.toolGetContext(ctx, params.Arguments)
	case "list_contexts":
		return a.toolListContexts(ctx, params.Arguments)
	case "update_context":
		return a.toolUpdateContext(ctx, params.Arguments)
	case "list_emails":
		return a.toolListEmails(ctx, params.Arguments)
	case "get_email":
		return a.toolGetEmail(ctx, params.Arguments)
	case "get_clarification_queue":
		return a.toolGetClarificationQueue(ctx, params.Arguments)
	case "resolve_clarification":
		return a.toolResolveClarification(ctx, params.Arguments)
	case "snooze_clarification":
		return a.toolSnoozeClarification(ctx, params.Arguments)
	case "add_thread_entry":
		return a.toolAddThreadEntry(ctx, params.Arguments)
	case "get_thread":
		return a.toolGetThread(ctx, params.Arguments)
	case "record_outcome":
		return a.toolRecordOutcome(ctx, params.Arguments)
	case "get_outcome_observations":
		return a.toolGetOutcomeObservations(ctx, params.Arguments)
	case "create_event":
		return a.toolCreateEvent(ctx, params.Arguments)
	case "list_events":
		return a.toolListEvents(ctx, params.Arguments)
	case "get_event":
		return a.toolGetEvent(ctx, params.Arguments)
	case "update_event":
		return a.toolUpdateEvent(ctx, params.Arguments)
	case "delete_event":
		return a.toolDeleteEvent(ctx, params.Arguments)
	case "get_daily_plan":
		return a.toolGetDailyPlan(ctx, params.Arguments)
	case "generate_daily_plan":
		return a.toolGenerateDailyPlan(ctx, params.Arguments)
	case "get_schedule":
		return a.toolGetSchedule(ctx, params.Arguments)
	case "create_time_block":
		return a.toolCreateTimeBlock(ctx, params.Arguments)
	case "confirm_time_block":
		return a.toolConfirmTimeBlock(ctx, params.Arguments)
	case "get_inference_context":
		return a.toolGetInferenceContext(ctx, params.Arguments)
	default:
		return toolResult{}, fmt.Errorf("unknown tool: %s", params.Name)
	}
}

// Helper to return JSON text result
func textResult(v any) (toolResult, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return toolResult{}, err
	}
	return toolResult{
		Content: []toolContent{{Type: "text", Text: string(data)}},
	}, nil
}

func (a *app) toolCreateTask(ctx context.Context, args json.RawMessage) (toolResult, error) {
	var input struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Priority    string `json:"priority"`
		Energy      string `json:"energy"`
		DueDate     string `json:"due_date"`
		ContextID   string `json:"context_id"`
		DurationMin *int   `json:"duration_min"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return toolResult{}, fmt.Errorf("invalid arguments: %w", err)
	}

	nt := taskbus.NewTask{
		Title:       input.Title,
		Description: input.Description,
		Status:      taskstatus.Open,
		Priority:    taskpriority.Medium,
		Energy:      taskenergy.Medium,
		DurationMin: input.DurationMin,
	}

	if input.Priority != "" {
		p, err := taskpriority.Parse(input.Priority)
		if err != nil {
			return toolResult{}, err
		}
		nt.Priority = p
	}

	if input.Energy != "" {
		e, err := taskenergy.Parse(input.Energy)
		if err != nil {
			return toolResult{}, err
		}
		nt.Energy = e
	}

	if input.DueDate != "" {
		t, err := time.Parse(time.RFC3339, input.DueDate)
		if err != nil {
			// Try date-only format
			t, err = time.Parse("2006-01-02", input.DueDate)
			if err != nil {
				return toolResult{}, fmt.Errorf("invalid due_date: %w", err)
			}
		}
		nt.DueDate = &t
	}

	if input.ContextID != "" {
		id, err := uuid.Parse(input.ContextID)
		if err != nil {
			return toolResult{}, fmt.Errorf("invalid context_id: %w", err)
		}
		nt.ContextID = &id
	}

	task, err := a.taskBus.Create(ctx, nt)
	if err != nil {
		return toolResult{}, err
	}

	return textResult(map[string]any{
		"id":       task.ID.String(),
		"title":    task.Title,
		"status":   task.Status.String(),
		"priority": task.Priority.String(),
		"message":  fmt.Sprintf("Created task: %s", task.Title),
	})
}

func (a *app) toolListTasks(ctx context.Context, args json.RawMessage) (toolResult, error) {
	var input struct {
		Status    string `json:"status"`
		Priority  string `json:"priority"`
		ContextID string `json:"context_id"`
		Page      int    `json:"page"`
		Rows      int    `json:"rows"`
	}
	if args != nil {
		json.Unmarshal(args, &input)
	}

	var filter taskbus.QueryFilter
	if input.Status != "" {
		s, err := taskstatus.Parse(input.Status)
		if err != nil {
			return toolResult{}, err
		}
		filter.Status = &s
	}
	if input.Priority != "" {
		p, err := taskpriority.Parse(input.Priority)
		if err != nil {
			return toolResult{}, err
		}
		filter.Priority = &p
	}
	if input.ContextID != "" {
		id, err := uuid.Parse(input.ContextID)
		if err != nil {
			return toolResult{}, err
		}
		filter.ContextID = &id
	}

	pageStr := "1"
	rowsStr := "20"
	if input.Page > 0 {
		pageStr = strconv.Itoa(input.Page)
	}
	if input.Rows > 0 {
		rowsStr = strconv.Itoa(input.Rows)
	}

	pg, err := page.Parse(pageStr, rowsStr)
	if err != nil {
		return toolResult{}, err
	}

	tasks, err := a.taskBus.Query(ctx, filter, taskbus.DefaultOrderBy, pg)
	if err != nil {
		return toolResult{}, err
	}

	total, err := a.taskBus.Count(ctx, filter)
	if err != nil {
		return toolResult{}, err
	}

	type taskSummary struct {
		ID       string  `json:"id"`
		Title    string  `json:"title"`
		Status   string  `json:"status"`
		Priority string  `json:"priority"`
		DueDate  *string `json:"due_date,omitempty"`
	}

	summaries := make([]taskSummary, len(tasks))
	for i, t := range tasks {
		ts := taskSummary{
			ID:       t.ID.String(),
			Title:    t.Title,
			Status:   t.Status.String(),
			Priority: t.Priority.String(),
		}
		if t.DueDate != nil {
			s := t.DueDate.Format("2006-01-02")
			ts.DueDate = &s
		}
		summaries[i] = ts
	}

	return textResult(map[string]any{
		"tasks": summaries,
		"total": total,
		"page":  pg.Number(),
	})
}

func (a *app) toolGetTask(ctx context.Context, args json.RawMessage) (toolResult, error) {
	var input struct {
		TaskID string `json:"task_id"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return toolResult{}, err
	}

	id, err := uuid.Parse(input.TaskID)
	if err != nil {
		return toolResult{}, fmt.Errorf("invalid task_id: %w", err)
	}

	task, err := a.taskBus.QueryByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return toolResult{}, fmt.Errorf("task not found: %s", input.TaskID)
		}
		return toolResult{}, err
	}

	return textResult(map[string]any{
		"id":           task.ID.String(),
		"title":        task.Title,
		"description":  task.Description,
		"status":       task.Status.String(),
		"priority":     task.Priority.String(),
		"energy":       task.Energy.String(),
		"duration_min": task.DurationMin,
		"created_at":   task.CreatedAt.Format(time.RFC3339),
	})
}

func (a *app) toolUpdateTask(ctx context.Context, args json.RawMessage) (toolResult, error) {
	var input struct {
		TaskID      string `json:"task_id"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Status      string `json:"status"`
		Priority    string `json:"priority"`
		Energy      string `json:"energy"`
		DueDate     string `json:"due_date"`
		DurationMin *int   `json:"duration_min"`
		ContextID   string `json:"context_id"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return toolResult{}, err
	}

	id, err := uuid.Parse(input.TaskID)
	if err != nil {
		return toolResult{}, fmt.Errorf("invalid task_id: %w", err)
	}

	task, err := a.taskBus.QueryByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return toolResult{}, fmt.Errorf("task not found: %s", input.TaskID)
		}
		return toolResult{}, err
	}

	var ut taskbus.UpdateTask
	if input.Title != "" {
		ut.Title = &input.Title
	}
	if input.Description != "" {
		ut.Description = &input.Description
	}
	if input.Status != "" {
		s, err := taskstatus.Parse(input.Status)
		if err != nil {
			return toolResult{}, err
		}
		ut.Status = &s
	}
	if input.Priority != "" {
		p, err := taskpriority.Parse(input.Priority)
		if err != nil {
			return toolResult{}, err
		}
		ut.Priority = &p
	}
	if input.Energy != "" {
		e, err := taskenergy.Parse(input.Energy)
		if err != nil {
			return toolResult{}, err
		}
		ut.Energy = &e
	}
	if input.DueDate != "" {
		t, err := time.Parse(time.RFC3339, input.DueDate)
		if err != nil {
			t, err = time.Parse("2006-01-02", input.DueDate)
			if err != nil {
				return toolResult{}, fmt.Errorf("invalid due_date: %w", err)
			}
		}
		ut.DueDate = &t
	}
	ut.DurationMin = input.DurationMin
	if input.ContextID != "" {
		cid, err := uuid.Parse(input.ContextID)
		if err != nil {
			return toolResult{}, fmt.Errorf("invalid context_id: %w", err)
		}
		ut.ContextID = &cid
	}

	updated, err := a.taskBus.Update(ctx, task, ut)
	if err != nil {
		return toolResult{}, err
	}

	// Fire debrief trigger if task was just completed
	if updated.CompletedAt != nil {
		go func() {
			ct := debriefbus.CompletedTask{
				ID:          updated.ID,
				Title:       updated.Title,
				DurationMin: updated.DurationMin,
				CreatedAt:   updated.CreatedAt.Unix(),
				CompletedAt: updated.CompletedAt.Unix(),
			}
			if err := a.debriefBus.OnTaskCompleted(context.Background(), ct); err != nil {
				_ = err // best-effort, logged inside debriefbus
			}
		}()
	}

	return textResult(map[string]any{
		"id":      updated.ID.String(),
		"title":   updated.Title,
		"status":  updated.Status.String(),
		"message": fmt.Sprintf("Updated task: %s", updated.Title),
	})
}

func (a *app) toolCompleteTask(ctx context.Context, args json.RawMessage) (toolResult, error) {
	var input struct {
		TaskID string `json:"task_id"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return toolResult{}, err
	}

	id, err := uuid.Parse(input.TaskID)
	if err != nil {
		return toolResult{}, fmt.Errorf("invalid task_id: %w", err)
	}

	task, err := a.taskBus.QueryByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return toolResult{}, fmt.Errorf("task not found: %s", input.TaskID)
		}
		return toolResult{}, err
	}

	done := taskstatus.Done
	updated, err := a.taskBus.Update(ctx, task, taskbus.UpdateTask{Status: &done})
	if err != nil {
		return toolResult{}, err
	}

	// Fire debrief trigger (best-effort)
	if updated.CompletedAt != nil {
		go func() {
			ct := debriefbus.CompletedTask{
				ID:          updated.ID,
				Title:       updated.Title,
				DurationMin: updated.DurationMin,
				CreatedAt:   updated.CreatedAt.Unix(),
				CompletedAt: updated.CompletedAt.Unix(),
			}
			if err := a.debriefBus.OnTaskCompleted(context.Background(), ct); err != nil {
				_ = err // best-effort, logged inside debriefbus
			}
		}()
	}

	return textResult(map[string]any{
		"id":      updated.ID.String(),
		"title":   updated.Title,
		"status":  "done",
		"message": fmt.Sprintf("Completed: %s", updated.Title),
	})
}

func (a *app) toolCreateContext(ctx context.Context, args json.RawMessage) (toolResult, error) {
	var input struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return toolResult{}, err
	}

	c, err := a.contextBus.Create(ctx, contextbus.NewContext{
		Title:       input.Title,
		Description: input.Description,
	})
	if err != nil {
		return toolResult{}, err
	}

	return textResult(map[string]any{
		"id":      c.ID.String(),
		"title":   c.Title,
		"status":  c.Status.String(),
		"message": fmt.Sprintf("Created context: %s", c.Title),
	})
}

func (a *app) toolGetContext(ctx context.Context, args json.RawMessage) (toolResult, error) {
	var input struct {
		ContextID string `json:"context_id"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return toolResult{}, err
	}

	id, err := uuid.Parse(input.ContextID)
	if err != nil {
		return toolResult{}, fmt.Errorf("invalid context_id: %w", err)
	}

	c, err := a.contextBus.QueryByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return toolResult{}, fmt.Errorf("context not found: %s", input.ContextID)
		}
		return toolResult{}, err
	}

	// Also get open tasks for this context
	filter := taskbus.QueryFilter{ContextID: &id}
	tasks, _ := a.taskBus.Query(ctx, filter, taskbus.DefaultOrderBy, page.New(1, 50))

	type taskSummary struct {
		ID       string `json:"id"`
		Title    string `json:"title"`
		Status   string `json:"status"`
		Priority string `json:"priority"`
	}

	taskSums := make([]taskSummary, len(tasks))
	for i, t := range tasks {
		taskSums[i] = taskSummary{
			ID: t.ID.String(), Title: t.Title,
			Status: t.Status.String(), Priority: t.Priority.String(),
		}
	}

	return textResult(map[string]any{
		"id":          c.ID.String(),
		"title":       c.Title,
		"description": c.Description,
		"status":      c.Status.String(),
		"summary":     c.Summary,
		"tasks":       taskSums,
	})
}

func (a *app) toolListContexts(ctx context.Context, args json.RawMessage) (toolResult, error) {
	var input struct {
		Status string `json:"status"`
		Page   int    `json:"page"`
		Rows   int    `json:"rows"`
	}
	if args != nil {
		json.Unmarshal(args, &input)
	}

	filter := contextbus.QueryFilter{}
	if input.Status != "" {
		switch input.Status {
		case "active":
			s := contextbus.Active
			filter.Status = &s
		case "paused":
			s := contextbus.Paused
			filter.Status = &s
		case "closed":
			s := contextbus.Closed
			filter.Status = &s
		default:
			return toolResult{}, fmt.Errorf("invalid status: %s", input.Status)
		}
	} else {
		active := contextbus.Active
		filter.Status = &active
	}

	pageStr := "1"
	rowsStr := "20"
	if input.Page > 0 {
		pageStr = strconv.Itoa(input.Page)
	}
	if input.Rows > 0 {
		rowsStr = strconv.Itoa(input.Rows)
	}

	pg, err := page.Parse(pageStr, rowsStr)
	if err != nil {
		return toolResult{}, err
	}

	contexts, err := a.contextBus.Query(ctx, filter, contextbus.DefaultOrderBy, pg)
	if err != nil {
		return toolResult{}, err
	}

	type ctxSummary struct {
		ID      string `json:"id"`
		Title   string `json:"title"`
		Status  string `json:"status"`
		Summary string `json:"summary"`
	}

	summaries := make([]ctxSummary, len(contexts))
	for i, c := range contexts {
		summaries[i] = ctxSummary{
			ID: c.ID.String(), Title: c.Title,
			Status: c.Status.String(), Summary: c.Summary,
		}
	}

	return textResult(map[string]any{
		"contexts": summaries,
	})
}

func (a *app) toolUpdateContext(ctx context.Context, args json.RawMessage) (toolResult, error) {
	var input struct {
		ContextID   string `json:"context_id"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Status      string `json:"status"`
		Summary     string `json:"summary"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return toolResult{}, err
	}

	id, err := uuid.Parse(input.ContextID)
	if err != nil {
		return toolResult{}, fmt.Errorf("invalid context_id: %w", err)
	}

	c, err := a.contextBus.QueryByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return toolResult{}, fmt.Errorf("context not found: %s", input.ContextID)
		}
		return toolResult{}, err
	}

	var uc contextbus.UpdateContext
	if input.Title != "" {
		uc.Title = &input.Title
	}
	if input.Description != "" {
		uc.Description = &input.Description
	}
	if input.Summary != "" {
		uc.Summary = &input.Summary
	}
	if input.Status != "" {
		var s contextbus.Status
		switch input.Status {
		case "active":
			s = contextbus.Active
		case "paused":
			s = contextbus.Paused
		case "closed":
			s = contextbus.Closed
		default:
			return toolResult{}, fmt.Errorf("invalid status: %s", input.Status)
		}
		uc.Status = &s
	}

	updated, err := a.contextBus.Update(ctx, c, uc)
	if err != nil {
		return toolResult{}, err
	}

	// Fire debrief trigger if context was just closed
	if input.Status == "closed" {
		go func() {
			cc := debriefbus.ClosedContext{
				ID:    updated.ID,
				Title: updated.Title,
			}
			if err := a.debriefBus.OnContextClosed(context.Background(), cc); err != nil {
				_ = err // best-effort, logged inside debriefbus
			}
		}()
	}

	return textResult(map[string]any{
		"id":      updated.ID.String(),
		"title":   updated.Title,
		"status":  updated.Status.String(),
		"message": fmt.Sprintf("Updated context: %s", updated.Title),
	})
}

func (a *app) toolListEmails(ctx context.Context, args json.RawMessage) (toolResult, error) {
	var input struct {
		ContextID   string `json:"context_id"`
		FromAddress string `json:"from_address"`
		Page        int    `json:"page"`
		Rows        int    `json:"rows"`
	}
	if args != nil {
		json.Unmarshal(args, &input)
	}

	var filter emailbus.QueryFilter
	if input.ContextID != "" {
		id, err := uuid.Parse(input.ContextID)
		if err != nil {
			return toolResult{}, fmt.Errorf("invalid context_id: %w", err)
		}
		filter.ContextID = &id
	}
	if input.FromAddress != "" {
		filter.FromAddress = &input.FromAddress
	}

	pageStr := "1"
	rowsStr := "20"
	if input.Page > 0 {
		pageStr = strconv.Itoa(input.Page)
	}
	if input.Rows > 0 {
		rowsStr = strconv.Itoa(input.Rows)
	}

	pg, err := page.Parse(pageStr, rowsStr)
	if err != nil {
		return toolResult{}, err
	}

	emails, err := a.emailBus.Query(ctx, filter, emailbus.DefaultOrderBy, pg)
	if err != nil {
		return toolResult{}, err
	}

	total, err := a.emailBus.Count(ctx, filter)
	if err != nil {
		return toolResult{}, err
	}

	type emailSummary struct {
		ID          string  `json:"id"`
		FromAddress string  `json:"from_address"`
		Subject     string  `json:"subject"`
		ReceivedAt  string  `json:"received_at"`
		ContextID   *string `json:"context_id,omitempty"`
	}

	summaries := make([]emailSummary, len(emails))
	for i, e := range emails {
		es := emailSummary{
			ID:          e.ID.String(),
			FromAddress: e.FromAddress,
			Subject:     e.Subject,
			ReceivedAt:  e.ReceivedAt.Format(time.RFC3339),
		}
		if e.ContextID != nil {
			s := e.ContextID.String()
			es.ContextID = &s
		}
		summaries[i] = es
	}

	return textResult(map[string]any{
		"emails": summaries,
		"total":  total,
		"page":   pg.Number(),
	})
}

func (a *app) toolGetClarificationQueue(ctx context.Context, args json.RawMessage) (toolResult, error) {
	var input struct {
		Status string `json:"status"`
		Kind   string `json:"kind"`
		Page   int    `json:"page"`
		Rows   int    `json:"rows"`
	}
	if args != nil {
		json.Unmarshal(args, &input)
	}

	filter := clarificationbus.QueryFilter{}

	if input.Status != "" {
		s, err := clarificationstatus.Parse(input.Status)
		if err != nil {
			return toolResult{}, err
		}
		filter.Status = &s
	} else {
		pending := clarificationstatus.Pending
		filter.Status = &pending
	}

	if input.Kind != "" {
		k, err := clarificationkind.Parse(input.Kind)
		if err != nil {
			return toolResult{}, fmt.Errorf("invalid kind: %w", err)
		}
		filter.Kind = &k
	}

	pageStr := "1"
	rowsStr := "20"
	if input.Page > 0 {
		pageStr = strconv.Itoa(input.Page)
	}
	if input.Rows > 0 {
		rowsStr = strconv.Itoa(input.Rows)
	}

	pg, err := page.Parse(pageStr, rowsStr)
	if err != nil {
		return toolResult{}, err
	}

	items, err := a.clarificationBus.Query(ctx, filter, clarificationbus.DefaultOrderBy, pg)
	if err != nil {
		return toolResult{}, err
	}

	total, err := a.clarificationBus.Count(ctx, filter)
	if err != nil {
		return toolResult{}, err
	}

	type itemSummary struct {
		ID            string          `json:"id"`
		Kind          string          `json:"kind"`
		Status        string          `json:"status"`
		SubjectType   string          `json:"subject_type"`
		SubjectID     string          `json:"subject_id"`
		Question      string          `json:"question"`
		ClaudeGuess   json.RawMessage `json:"claude_guess,omitempty"`
		AnswerOptions json.RawMessage `json:"answer_options"`
		PriorityScore float32         `json:"priority_score"`
		CreatedAt     string          `json:"created_at"`
	}

	summaries := make([]itemSummary, len(items))
	for i, item := range items {
		s := itemSummary{
			ID:            item.ID.String(),
			Kind:          item.Kind.String(),
			Status:        item.Status.String(),
			SubjectType:   item.SubjectType,
			SubjectID:     item.SubjectID.String(),
			Question:      item.Question,
			AnswerOptions: item.AnswerOptions,
			PriorityScore: item.PriorityScore,
			CreatedAt:     item.CreatedAt.Format(time.RFC3339),
		}
		if item.ClaudeGuess != nil {
			s.ClaudeGuess = *item.ClaudeGuess
		}
		summaries[i] = s
	}

	return textResult(map[string]any{
		"items": summaries,
		"total": total,
		"page":  pg.Number(),
	})
}

func (a *app) toolResolveClarification(ctx context.Context, args json.RawMessage) (toolResult, error) {
	var input struct {
		ClarificationID string `json:"clarification_id"`
		Answer          any    `json:"answer"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return toolResult{}, err
	}

	id, err := uuid.Parse(input.ClarificationID)
	if err != nil {
		return toolResult{}, fmt.Errorf("invalid clarification_id: %w", err)
	}

	item, err := a.clarificationBus.QueryByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return toolResult{}, fmt.Errorf("clarification not found: %s", input.ClarificationID)
		}
		return toolResult{}, err
	}

	answerJSON, err := json.Marshal(input.Answer)
	if err != nil {
		return toolResult{}, fmt.Errorf("invalid answer: %w", err)
	}

	rc := clarificationbus.ResolveClarificationItem{
		Answer: answerJSON,
	}

	resolved, err := a.clarificationBus.Resolve(ctx, item, rc)
	if err != nil {
		return toolResult{}, err
	}

	return textResult(map[string]any{
		"id":      resolved.ID.String(),
		"status":  resolved.Status.String(),
		"message": "Clarification resolved",
	})
}

func (a *app) toolSnoozeClarification(ctx context.Context, args json.RawMessage) (toolResult, error) {
	var input struct {
		ClarificationID string `json:"clarification_id"`
		Hours           int    `json:"hours"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return toolResult{}, err
	}

	id, err := uuid.Parse(input.ClarificationID)
	if err != nil {
		return toolResult{}, fmt.Errorf("invalid clarification_id: %w", err)
	}

	item, err := a.clarificationBus.QueryByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return toolResult{}, fmt.Errorf("clarification not found: %s", input.ClarificationID)
		}
		return toolResult{}, err
	}

	hours := 24
	if input.Hours > 0 {
		hours = input.Hours
	}

	until := time.Now().Add(time.Duration(hours) * time.Hour)

	snoozed, err := a.clarificationBus.Snooze(ctx, item, until)
	if err != nil {
		return toolResult{}, err
	}

	return textResult(map[string]any{
		"id":            snoozed.ID.String(),
		"status":        snoozed.Status.String(),
		"snoozed_until": snoozed.SnoozedUntil.Format(time.RFC3339),
		"message":       fmt.Sprintf("Snoozed for %d hours", hours),
	})
}

func (a *app) toolAddThreadEntry(ctx context.Context, args json.RawMessage) (toolResult, error) {
	var input struct {
		SubjectType    string `json:"subject_type"`
		SubjectID      string `json:"subject_id"`
		Kind           string `json:"kind"`
		Content        string `json:"content"`
		Source         string `json:"source"`
		Sentiment      string `json:"sentiment"`
		RequiresAction bool   `json:"requires_action"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return toolResult{}, err
	}

	subjectID, err := uuid.Parse(input.SubjectID)
	if err != nil {
		return toolResult{}, fmt.Errorf("invalid subject_id: %w", err)
	}

	kind, err := threadentrykind.Parse(input.Kind)
	if err != nil {
		return toolResult{}, err
	}

	source := threadsource.User
	if input.Source != "" {
		source, err = threadsource.Parse(input.Source)
		if err != nil {
			return toolResult{}, err
		}
	}

	ne := threadbus.NewThreadEntry{
		SubjectType:    input.SubjectType,
		SubjectID:      subjectID,
		Kind:           kind,
		Content:        input.Content,
		Source:         source,
		RequiresAction: input.RequiresAction,
	}

	if input.Sentiment != "" {
		ne.Sentiment = &input.Sentiment
	}

	entry, err := a.threadBus.AddEntry(ctx, ne)
	if err != nil {
		return toolResult{}, err
	}

	return textResult(map[string]any{
		"id":           entry.ID.String(),
		"subject_type": entry.SubjectType,
		"subject_id":   entry.SubjectID.String(),
		"kind":         entry.Kind.String(),
		"message":      fmt.Sprintf("Added %s entry to %s thread", entry.Kind.String(), entry.SubjectType),
	})
}

func (a *app) toolGetThread(ctx context.Context, args json.RawMessage) (toolResult, error) {
	var input struct {
		SubjectType string `json:"subject_type"`
		SubjectID   string `json:"subject_id"`
		Page        int    `json:"page"`
		Rows        int    `json:"rows"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return toolResult{}, err
	}

	subjectID, err := uuid.Parse(input.SubjectID)
	if err != nil {
		return toolResult{}, fmt.Errorf("invalid subject_id: %w", err)
	}

	pageStr := "1"
	rowsStr := "20"
	if input.Page > 0 {
		pageStr = strconv.Itoa(input.Page)
	}
	if input.Rows > 0 {
		rowsStr = strconv.Itoa(input.Rows)
	}

	pg, err := page.Parse(pageStr, rowsStr)
	if err != nil {
		return toolResult{}, err
	}

	entries, err := a.threadBus.QueryBySubject(ctx, input.SubjectType, subjectID, pg)
	if err != nil {
		return toolResult{}, err
	}

	total, err := a.threadBus.CountBySubject(ctx, input.SubjectType, subjectID)
	if err != nil {
		return toolResult{}, err
	}

	type entrySummary struct {
		ID             string  `json:"id"`
		Kind           string  `json:"kind"`
		Content        string  `json:"content"`
		Source         string  `json:"source"`
		Sentiment      *string `json:"sentiment,omitempty"`
		RequiresAction bool    `json:"requires_action"`
		CreatedAt      string  `json:"created_at"`
	}

	summaries := make([]entrySummary, len(entries))
	for i, e := range entries {
		summaries[i] = entrySummary{
			ID:             e.ID.String(),
			Kind:           e.Kind.String(),
			Content:        e.Content,
			Source:         e.Source.String(),
			Sentiment:      e.Sentiment,
			RequiresAction: e.RequiresAction,
			CreatedAt:      e.CreatedAt.Format(time.RFC3339),
		}
	}

	return textResult(map[string]any{
		"entries":      summaries,
		"total":        total,
		"subject_type": input.SubjectType,
		"subject_id":   input.SubjectID,
		"page":         pg.Number(),
	})
}

func (a *app) toolRecordOutcome(ctx context.Context, args json.RawMessage) (toolResult, error) {
	var input struct {
		SubjectType string  `json:"subject_type"`
		SubjectID   string  `json:"subject_id"`
		Kind        string  `json:"kind"`
		Data        any     `json:"data"`
		Source      string  `json:"source"`
		Confidence  float32 `json:"confidence"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return toolResult{}, err
	}

	subjectID, err := uuid.Parse(input.SubjectID)
	if err != nil {
		return toolResult{}, fmt.Errorf("invalid subject_id: %w", err)
	}

	kind, err := observationkind.Parse(input.Kind)
	if err != nil {
		return toolResult{}, err
	}

	dataJSON, err := json.Marshal(input.Data)
	if err != nil {
		return toolResult{}, fmt.Errorf("invalid data: %w", err)
	}

	source := "user"
	if input.Source != "" {
		source = input.Source
	}

	confidence := float32(1.0)
	if input.Confidence > 0 {
		confidence = input.Confidence
	}

	no := observationbus.NewObservation{
		SubjectType: input.SubjectType,
		SubjectID:   subjectID,
		Kind:        kind,
		Data:        dataJSON,
		Source:      source,
		Confidence:  confidence,
		Weight:      1.0,
	}

	obs, err := a.observationBus.Record(ctx, no)
	if err != nil {
		return toolResult{}, err
	}

	return textResult(map[string]any{
		"id":           obs.ID.String(),
		"subject_type": obs.SubjectType,
		"subject_id":   obs.SubjectID.String(),
		"kind":         obs.Kind.String(),
		"message":      fmt.Sprintf("Recorded %s observation", obs.Kind.String()),
	})
}

func (a *app) toolGetOutcomeObservations(ctx context.Context, args json.RawMessage) (toolResult, error) {
	var input struct {
		SubjectType string `json:"subject_type"`
		SubjectID   string `json:"subject_id"`
		Page        int    `json:"page"`
		RowsPerPage int    `json:"rows_per_page"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return toolResult{}, fmt.Errorf("invalid arguments: %w", err)
	}

	if input.SubjectType == "" {
		return toolResult{}, fmt.Errorf("subject_type is required")
	}

	subjectID, err := uuid.Parse(input.SubjectID)
	if err != nil {
		return toolResult{}, fmt.Errorf("invalid subject_id: %w", err)
	}

	pageNum := input.Page
	if pageNum < 1 {
		pageNum = 1
	}
	rowsPerPage := input.RowsPerPage
	if rowsPerPage < 1 {
		rowsPerPage = 20
	}

	pg, err := page.Parse(strconv.Itoa(pageNum), strconv.Itoa(rowsPerPage))
	if err != nil {
		return toolResult{}, fmt.Errorf("invalid pagination: %w", err)
	}

	observations, err := a.observationBus.QueryBySubject(ctx, input.SubjectType, subjectID, pg)
	if err != nil {
		return toolResult{}, fmt.Errorf("query observations: %w", err)
	}

	total, err := a.observationBus.Count(ctx, observationbus.QueryFilter{
		SubjectType: &input.SubjectType,
		SubjectID:   &subjectID,
	})
	if err != nil {
		return toolResult{}, fmt.Errorf("count observations: %w", err)
	}

	return textResult(map[string]any{
		"observations":  observations,
		"total":         total,
		"page":          pageNum,
		"rows_per_page": rowsPerPage,
	})
}

func (a *app) toolGetEmail(ctx context.Context, args json.RawMessage) (toolResult, error) {
	var input struct {
		EmailID string `json:"email_id"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return toolResult{}, err
	}

	id, err := uuid.Parse(input.EmailID)
	if err != nil {
		return toolResult{}, fmt.Errorf("invalid email_id: %w", err)
	}

	email, err := a.emailBus.QueryByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return toolResult{}, fmt.Errorf("email not found: %s", input.EmailID)
		}
		return toolResult{}, err
	}

	result := map[string]any{
		"id":           email.ID.String(),
		"raw_input_id": email.RawInputID.String(),
		"from_address": email.FromAddress,
		"to_address":   email.ToAddress,
		"subject":      email.Subject,
		"body_text":    email.BodyText,
		"received_at":  email.ReceivedAt.Format(time.RFC3339),
		"created_at":   email.CreatedAt.Format(time.RFC3339),
	}

	if email.MessageID != nil {
		result["message_id"] = *email.MessageID
	}
	if email.FromName != nil {
		result["from_name"] = *email.FromName
	}
	if email.ContextID != nil {
		result["context_id"] = email.ContextID.String()
	}

	return textResult(result)
}

func (a *app) toolCreateEvent(ctx context.Context, args json.RawMessage) (toolResult, error) {
	var input struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Location    string `json:"location"`
		StartsAt    string `json:"starts_at"`
		EndsAt      string `json:"ends_at"`
		AllDay      bool   `json:"all_day"`
		ContextID   string `json:"context_id"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return toolResult{}, fmt.Errorf("invalid arguments: %w", err)
	}

	// Parse timestamps
	startsAt, err := time.Parse(time.RFC3339, input.StartsAt)
	if err != nil {
		return toolResult{}, fmt.Errorf("invalid starts_at: %w", err)
	}

	endsAt, err := time.Parse(time.RFC3339, input.EndsAt)
	if err != nil {
		return toolResult{}, fmt.Errorf("invalid ends_at: %w", err)
	}

	ne := eventbus.NewEvent{
		Title:       input.Title,
		Description: input.Description,
		StartsAt:    startsAt,
		EndsAt:      endsAt,
		AllDay:      input.AllDay,
	}

	if input.Location != "" {
		ne.Location = &input.Location
	}

	if input.ContextID != "" {
		id, err := uuid.Parse(input.ContextID)
		if err != nil {
			return toolResult{}, fmt.Errorf("invalid context_id: %w", err)
		}
		ne.ContextID = &id
	}

	event, err := a.eventBus.Create(ctx, ne)
	if err != nil {
		return toolResult{}, err
	}

	return textResult(map[string]any{
		"id":       event.ID.String(),
		"title":    event.Title,
		"starts_at": event.StartsAt.Format(time.RFC3339),
		"ends_at":   event.EndsAt.Format(time.RFC3339),
		"message":  fmt.Sprintf("Created event: %s (%s)", event.Title, event.ID.String()),
	})
}

func (a *app) toolListEvents(ctx context.Context, args json.RawMessage) (toolResult, error) {
	var input struct {
		DateFrom  string `json:"date_from"`
		DateTo    string `json:"date_to"`
		ContextID string `json:"context_id"`
		Page      int    `json:"page"`
		Rows      int    `json:"rows"`
	}
	if args != nil {
		json.Unmarshal(args, &input)
	}

	var filter eventbus.QueryFilter

	if input.DateFrom != "" {
		t, err := time.Parse(time.RFC3339, input.DateFrom)
		if err != nil {
			return toolResult{}, fmt.Errorf("invalid date_from: %w", err)
		}
		filter.DateFrom = &t
	}

	if input.DateTo != "" {
		t, err := time.Parse(time.RFC3339, input.DateTo)
		if err != nil {
			return toolResult{}, fmt.Errorf("invalid date_to: %w", err)
		}
		filter.DateTo = &t
	}

	if input.ContextID != "" {
		id, err := uuid.Parse(input.ContextID)
		if err != nil {
			return toolResult{}, fmt.Errorf("invalid context_id: %w", err)
		}
		filter.ContextID = &id
	}

	pageStr := "1"
	rowsStr := "20"
	if input.Page > 0 {
		pageStr = strconv.Itoa(input.Page)
	}
	if input.Rows > 0 {
		rowsStr = strconv.Itoa(input.Rows)
	}

	pg, err := page.Parse(pageStr, rowsStr)
	if err != nil {
		return toolResult{}, err
	}

	events, err := a.eventBus.Query(ctx, filter, eventbus.DefaultOrderBy, pg)
	if err != nil {
		return toolResult{}, err
	}

	total, err := a.eventBus.Count(ctx, filter)
	if err != nil {
		return toolResult{}, err
	}

	type eventSummary struct {
		ID        string `json:"id"`
		Title     string `json:"title"`
		StartsAt  string `json:"starts_at"`
		EndsAt    string `json:"ends_at"`
		AllDay    bool   `json:"all_day"`
		ContextID string `json:"context_id,omitempty"`
	}

	summaries := make([]eventSummary, len(events))
	for i, e := range events {
		es := eventSummary{
			ID:       e.ID.String(),
			Title:    e.Title,
			StartsAt: e.StartsAt.Format(time.RFC3339),
			EndsAt:   e.EndsAt.Format(time.RFC3339),
			AllDay:   e.AllDay,
		}
		if e.ContextID != nil {
			es.ContextID = e.ContextID.String()
		}
		summaries[i] = es
	}

	return textResult(map[string]any{
		"events": summaries,
		"total":  total,
		"page":   pg.Number(),
	})
}

func (a *app) toolGetEvent(ctx context.Context, args json.RawMessage) (toolResult, error) {
	var input struct {
		EventID string `json:"event_id"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return toolResult{}, err
	}

	id, err := uuid.Parse(input.EventID)
	if err != nil {
		return toolResult{}, fmt.Errorf("invalid event_id: %w", err)
	}

	event, err := a.eventBus.QueryByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return toolResult{}, fmt.Errorf("event not found: %s", input.EventID)
		}
		return toolResult{}, err
	}

	result := map[string]any{
		"id":           event.ID.String(),
		"title":        event.Title,
		"description":  event.Description,
		"starts_at":    event.StartsAt.Format(time.RFC3339),
		"ends_at":      event.EndsAt.Format(time.RFC3339),
		"all_day":      event.AllDay,
		"created_at":   event.CreatedAt.Format(time.RFC3339),
		"updated_at":   event.UpdatedAt.Format(time.RFC3339),
	}

	if event.Location != nil {
		result["location"] = *event.Location
	}
	if event.ContextID != nil {
		result["context_id"] = event.ContextID.String()
	}
	if event.RawInputID != nil {
		result["raw_input_id"] = event.RawInputID.String()
	}

	return textResult(result)
}

func (a *app) toolUpdateEvent(ctx context.Context, args json.RawMessage) (toolResult, error) {
	var input struct {
		EventID     string `json:"event_id"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Location    string `json:"location"`
		StartsAt    string `json:"starts_at"`
		EndsAt      string `json:"ends_at"`
		AllDay      *bool  `json:"all_day"`
		ContextID   string `json:"context_id"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return toolResult{}, err
	}

	id, err := uuid.Parse(input.EventID)
	if err != nil {
		return toolResult{}, fmt.Errorf("invalid event_id: %w", err)
	}

	event, err := a.eventBus.QueryByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return toolResult{}, fmt.Errorf("event not found: %s", input.EventID)
		}
		return toolResult{}, err
	}

	var ue eventbus.UpdateEvent

	if input.Title != "" {
		ue.Title = &input.Title
	}
	if input.Description != "" {
		ue.Description = &input.Description
	}
	if input.Location != "" {
		ue.Location = &input.Location
	}
	if input.StartsAt != "" {
		t, err := time.Parse(time.RFC3339, input.StartsAt)
		if err != nil {
			return toolResult{}, fmt.Errorf("invalid starts_at: %w", err)
		}
		ue.StartsAt = &t
	}
	if input.EndsAt != "" {
		t, err := time.Parse(time.RFC3339, input.EndsAt)
		if err != nil {
			return toolResult{}, fmt.Errorf("invalid ends_at: %w", err)
		}
		ue.EndsAt = &t
	}
	if input.AllDay != nil {
		ue.AllDay = input.AllDay
	}
	if input.ContextID != "" {
		cid, err := uuid.Parse(input.ContextID)
		if err != nil {
			return toolResult{}, fmt.Errorf("invalid context_id: %w", err)
		}
		ue.ContextID = &cid
	}

	updated, err := a.eventBus.Update(ctx, event, ue)
	if err != nil {
		return toolResult{}, err
	}

	return textResult(map[string]any{
		"id":         updated.ID.String(),
		"title":      updated.Title,
		"starts_at":  updated.StartsAt.Format(time.RFC3339),
		"ends_at":    updated.EndsAt.Format(time.RFC3339),
		"message":    fmt.Sprintf("Updated event: %s", updated.Title),
	})
}

func (a *app) toolDeleteEvent(ctx context.Context, args json.RawMessage) (toolResult, error) {
	var input struct {
		EventID string `json:"event_id"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return toolResult{}, err
	}

	id, err := uuid.Parse(input.EventID)
	if err != nil {
		return toolResult{}, fmt.Errorf("invalid event_id: %w", err)
	}

	event, err := a.eventBus.QueryByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return toolResult{}, fmt.Errorf("event not found: %s", input.EventID)
		}
		return toolResult{}, err
	}

	if err := a.eventBus.Delete(ctx, event); err != nil {
		return toolResult{}, err
	}

	return textResult(map[string]any{
		"message": fmt.Sprintf("Deleted event: %s (%s)", event.Title, event.ID.String()),
	})
}

func (a *app) toolGetDailyPlan(ctx context.Context, args json.RawMessage) (toolResult, error) {
	var input struct {
		Date string `json:"date"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return toolResult{}, fmt.Errorf("invalid arguments: %w", err)
	}

	dateStr := input.Date
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return toolResult{}, fmt.Errorf("invalid date format: %w", err)
	}

	plan, items, err := a.dailyPlanBus.GetByDate(ctx, date)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			// Return empty plan
			return textResult(map[string]any{
				"planDate":   dateStr,
				"generation": 0,
				"items":      []map[string]any{},
			})
		}
		return toolResult{}, fmt.Errorf("query plan: %w", err)
	}

	// Convert items to JSON-friendly format
	itemsJSON := make([]map[string]any, len(items))
	for i, item := range items {
		itemsJSON[i] = map[string]any{
			"id":               item.ID.String(),
			"taskId":           item.TaskID.String(),
			"groupName":        item.GroupName,
			"groupPosition":    item.GroupPosition,
			"position":         item.Position,
			"status":           item.Status,
			"aiDurationMin":    item.AIDurationMin,
			"aiPriorityReason": item.AIPriorityReason,
			"userPosition":     item.UserPosition,
			"userDurationMin":  item.UserDurationMin,
			"createdAt":        item.CreatedAt.Format(time.RFC3339),
		}
		if item.CompletedAt != nil {
			itemsJSON[i]["completedAt"] = item.CompletedAt.Format(time.RFC3339)
		}
		if item.DismissReason != nil {
			itemsJSON[i]["dismissReason"] = *item.DismissReason
		}
		if item.DismissNote != nil {
			itemsJSON[i]["dismissNote"] = *item.DismissNote
		}
	}

	return textResult(map[string]any{
		"id":         plan.ID.String(),
		"planDate":   plan.PlanDate.Format("2006-01-02"),
		"generation": plan.Generation,
		"modelUsed":  plan.ModelUsed,
		"createdAt":  plan.CreatedAt.Format(time.RFC3339),
		"items":      itemsJSON,
	})
}

func (a *app) toolGenerateDailyPlan(ctx context.Context, args json.RawMessage) (toolResult, error) {
	var input struct {
		Date string `json:"date"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return toolResult{}, fmt.Errorf("invalid arguments: %w", err)
	}

	dateStr := input.Date
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}

	if _, err := time.Parse("2006-01-02", dateStr); err != nil {
		return toolResult{}, fmt.Errorf("invalid date format: %w", err)
	}

	// TODO: Implement full plan generation via MCP
	// For now, return a message indicating generation was triggered
	return textResult(map[string]any{
		"message": fmt.Sprintf("Plan generation for %s triggered. Use get_daily_plan to retrieve the generated plan.", dateStr),
		"status":  "plan_generation_not_yet_implemented_in_mcp",
	})
}

func (a *app) toolGetSchedule(ctx context.Context, args json.RawMessage) (toolResult, error) {
	var input struct {
		DateFrom string `json:"date_from"`
		DateTo   string `json:"date_to"`
	}
	if args != nil {
		json.Unmarshal(args, &input)
	}

	now := time.Now()
	var dateFrom, dateTo time.Time

	if input.DateFrom != "" {
		t, err := time.Parse(time.RFC3339, input.DateFrom)
		if err != nil {
			return toolResult{}, fmt.Errorf("invalid date_from: %w", err)
		}
		dateFrom = t
	} else {
		dateFrom = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	}

	if input.DateTo != "" {
		t, err := time.Parse(time.RFC3339, input.DateTo)
		if err != nil {
			return toolResult{}, fmt.Errorf("invalid date_to: %w", err)
		}
		dateTo = t
	} else {
		dateTo = dateFrom.Add(24 * time.Hour)
	}

	bigPage := page.New(1, 200)

	events, err := a.eventBus.Query(ctx, eventbus.QueryFilter{
		DateFrom: &dateFrom,
		DateTo:   &dateTo,
	}, eventbus.DefaultOrderBy, bigPage)
	if err != nil {
		return toolResult{}, fmt.Errorf("query events: %w", err)
	}

	blocks, err := a.timeBlockBus.Query(ctx, timeblockbus.QueryFilter{
		DateFrom: &dateFrom,
		DateTo:   &dateTo,
	}, timeblockbus.DefaultOrderBy, bigPage)
	if err != nil {
		return toolResult{}, fmt.Errorf("query time blocks: %w", err)
	}

	type scheduleItem struct {
		Type      string  `json:"type"`
		ID        string  `json:"id"`
		Title     string  `json:"title,omitempty"`
		StartsAt  string  `json:"starts_at"`
		EndsAt    string  `json:"ends_at"`
		AllDay    bool    `json:"all_day,omitempty"`
		Location  *string `json:"location,omitempty"`
		TaskID    string  `json:"task_id,omitempty"`
		Confirmed bool    `json:"confirmed,omitempty"`
	}

	items := make([]scheduleItem, 0, len(events)+len(blocks))

	for _, e := range events {
		items = append(items, scheduleItem{
			Type:     "event",
			ID:       e.ID.String(),
			Title:    e.Title,
			StartsAt: e.StartsAt.Format(time.RFC3339),
			EndsAt:   e.EndsAt.Format(time.RFC3339),
			AllDay:   e.AllDay,
			Location: e.Location,
		})
	}

	for _, tb := range blocks {
		items = append(items, scheduleItem{
			Type:      "time_block",
			ID:        tb.ID.String(),
			TaskID:    tb.TaskID.String(),
			StartsAt:  tb.StartsAt.Format(time.RFC3339),
			EndsAt:    tb.EndsAt.Format(time.RFC3339),
			Confirmed: tb.Confirmed,
		})
	}

	return textResult(map[string]any{
		"items": items,
		"count": len(items),
	})
}

func (a *app) toolCreateTimeBlock(ctx context.Context, args json.RawMessage) (toolResult, error) {
	var input struct {
		TaskID   string `json:"task_id"`
		StartsAt string `json:"starts_at"`
		EndsAt   string `json:"ends_at"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return toolResult{}, fmt.Errorf("invalid arguments: %w", err)
	}

	taskID, err := uuid.Parse(input.TaskID)
	if err != nil {
		return toolResult{}, fmt.Errorf("invalid task_id: %w", err)
	}

	startsAt, err := time.Parse(time.RFC3339, input.StartsAt)
	if err != nil {
		return toolResult{}, fmt.Errorf("invalid starts_at: %w", err)
	}

	endsAt, err := time.Parse(time.RFC3339, input.EndsAt)
	if err != nil {
		return toolResult{}, fmt.Errorf("invalid ends_at: %w", err)
	}

	block, err := a.timeBlockBus.Create(ctx, timeblockbus.NewTimeBlock{
		TaskID:   taskID,
		StartsAt: startsAt,
		EndsAt:   endsAt,
	})
	if err != nil {
		return toolResult{}, err
	}

	return textResult(map[string]any{
		"id":        block.ID.String(),
		"task_id":   block.TaskID.String(),
		"starts_at": block.StartsAt.Format(time.RFC3339),
		"ends_at":   block.EndsAt.Format(time.RFC3339),
		"confirmed": block.Confirmed,
		"message":   fmt.Sprintf("Scheduled task %s from %s to %s", block.TaskID.String(), input.StartsAt, input.EndsAt),
	})
}

func (a *app) toolConfirmTimeBlock(ctx context.Context, args json.RawMessage) (toolResult, error) {
	var input struct {
		BlockID string `json:"block_id"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return toolResult{}, fmt.Errorf("invalid arguments: %w", err)
	}

	blockID, err := uuid.Parse(input.BlockID)
	if err != nil {
		return toolResult{}, fmt.Errorf("invalid block_id: %w", err)
	}

	block, err := a.timeBlockBus.QueryByID(ctx, blockID)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return toolResult{}, fmt.Errorf("time block not found: %s", blockID)
		}
		return toolResult{}, err
	}

	confirmed := true
	updated, err := a.timeBlockBus.Update(ctx, block, timeblockbus.UpdateTimeBlock{
		Confirmed: &confirmed,
	})
	if err != nil {
		return toolResult{}, err
	}

	return textResult(map[string]any{
		"id":        updated.ID.String(),
		"confirmed": updated.Confirmed,
		"message":   fmt.Sprintf("Time block %s confirmed", updated.ID.String()),
	})
}

func (a *app) toolGetInferenceContext(ctx context.Context, args json.RawMessage) (toolResult, error) {
	var input struct {
		UseCase     string `json:"use_case"`
		Date        string `json:"date"`
		SubjectID   string `json:"subject_id"`
		SubjectType string `json:"subject_type"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return toolResult{}, fmt.Errorf("invalid arguments: %w", err)
	}

	switch input.UseCase {
	case "daily_plan":
		return a.inferenceContextDailyPlan(ctx, input.Date)
	case "email_extraction":
		return a.inferenceContextEmailExtraction(ctx)
	case "text_extraction":
		return a.inferenceContextTextExtraction(ctx, input.Date)
	case "thread_classification":
		return a.inferenceContextThreadClassification(ctx, input.SubjectID, input.SubjectType)
	default:
		return toolResult{}, fmt.Errorf("unknown use_case: %s", input.UseCase)
	}
}

func (a *app) inferenceContextDailyPlan(ctx context.Context, dateStr string) (toolResult, error) {
	// Open tasks.
	taskFilter := taskbus.QueryFilter{}
	openStatus := taskstatus.MustParse("open")
	taskFilter.Status = &openStatus
	tasks, err := a.taskBus.Query(ctx, taskFilter, taskbus.DefaultOrderBy, page.New(1, 100))
	if err != nil {
		return toolResult{}, fmt.Errorf("query tasks: %w", err)
	}

	// Also get blocked tasks.
	blockedStatus := taskstatus.MustParse("blocked")
	taskFilter.Status = &blockedStatus
	blockedTasks, err := a.taskBus.Query(ctx, taskFilter, taskbus.DefaultOrderBy, page.New(1, 100))
	if err != nil {
		return toolResult{}, fmt.Errorf("query blocked tasks: %w", err)
	}
	tasks = append(tasks, blockedTasks...)

	// Today's events.
	now := time.Now()
	if dateStr != "" {
		if t, err := time.Parse("2006-01-02", dateStr); err == nil {
			now = t
		}
	}
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	dayEnd := dayStart.Add(24 * time.Hour)
	eventFilter := eventbus.QueryFilter{
		DateFrom: &dayStart,
		DateTo:   &dayEnd,
	}
	events, err := a.eventBus.Query(ctx, eventFilter, eventbus.DefaultOrderBy, page.New(1, 50))
	if err != nil {
		return toolResult{}, fmt.Errorf("query events: %w", err)
	}

	// Active contexts for enrichment.
	activeStatus := contextbus.Active
	ctxFilter := contextbus.QueryFilter{Status: &activeStatus}
	contexts, err := a.contextBus.Query(ctx, ctxFilter, contextbus.DefaultOrderBy, page.New(1, 50))
	if err != nil {
		return toolResult{}, fmt.Errorf("query contexts: %w", err)
	}

	result := map[string]any{
		"tasks":    tasks,
		"events":   events,
		"contexts": contexts,
	}

	return textResult(result)
}

func (a *app) inferenceContextEmailExtraction(ctx context.Context) (toolResult, error) {
	activeStatus := contextbus.Active
	ctxFilter := contextbus.QueryFilter{Status: &activeStatus}
	contexts, err := a.contextBus.Query(ctx, ctxFilter, contextbus.DefaultOrderBy, page.New(1, 50))
	if err != nil {
		return toolResult{}, fmt.Errorf("query contexts: %w", err)
	}

	result := map[string]any{
		"active_contexts": contexts,
	}

	return textResult(result)
}

func (a *app) inferenceContextTextExtraction(ctx context.Context, dateStr string) (toolResult, error) {
	// Active contexts.
	activeStatus := contextbus.Active
	ctxFilter := contextbus.QueryFilter{Status: &activeStatus}
	contexts, err := a.contextBus.Query(ctx, ctxFilter, contextbus.DefaultOrderBy, page.New(1, 50))
	if err != nil {
		return toolResult{}, fmt.Errorf("query contexts: %w", err)
	}

	// Today's events for temporal grounding.
	now := time.Now()
	if dateStr != "" {
		if t, err := time.Parse("2006-01-02", dateStr); err == nil {
			now = t
		}
	}
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	dayEnd := dayStart.Add(24 * time.Hour)
	eventFilter := eventbus.QueryFilter{
		DateFrom: &dayStart,
		DateTo:   &dayEnd,
	}
	events, err := a.eventBus.Query(ctx, eventFilter, eventbus.DefaultOrderBy, page.New(1, 50))
	if err != nil {
		return toolResult{}, fmt.Errorf("query events: %w", err)
	}

	result := map[string]any{
		"active_contexts": contexts,
		"todays_events":   events,
	}

	return textResult(result)
}

func (a *app) inferenceContextThreadClassification(ctx context.Context, subjectID, subjectType string) (toolResult, error) {
	if subjectID == "" || subjectType == "" {
		return toolResult{}, fmt.Errorf("subject_id and subject_type are required for thread_classification")
	}

	sid, err := uuid.Parse(subjectID)
	if err != nil {
		return toolResult{}, fmt.Errorf("invalid subject_id: %w", err)
	}

	// Thread history (last 10 entries).
	entries, err := a.threadBus.QueryBySubject(ctx, subjectType, sid, page.New(1, 10))
	if err != nil {
		return toolResult{}, fmt.Errorf("query thread: %w", err)
	}

	// Subject details.
	var subject any
	switch subjectType {
	case "task":
		task, err := a.taskBus.QueryByID(ctx, sid)
		if err != nil {
			return toolResult{}, fmt.Errorf("query task: %w", err)
		}
		subject = task
	case "context":
		ctxObj, err := a.contextBus.QueryByID(ctx, sid)
		if err != nil {
			return toolResult{}, fmt.Errorf("query context: %w", err)
		}
		subject = ctxObj
	default:
		return toolResult{}, fmt.Errorf("invalid subject_type: %s", subjectType)
	}

	result := map[string]any{
		"thread_entries": entries,
		"subject":        subject,
	}

	return textResult(result)
}
