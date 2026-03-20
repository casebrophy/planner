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
	"github.com/casebrophy/planner/business/domain/emailbus"
	"github.com/casebrophy/planner/business/domain/observationbus"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/domain/threadbus"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/types/clarificationkind"
	"github.com/casebrophy/planner/business/types/clarificationstatus"
	"github.com/casebrophy/planner/business/types/observationkind"
	"github.com/casebrophy/planner/business/types/taskenergy"
	"github.com/casebrophy/planner/business/types/taskpriority"
	"github.com/casebrophy/planner/business/types/taskstatus"
	"github.com/casebrophy/planner/business/types/threadentrykind"
	"github.com/casebrophy/planner/business/types/threadsource"
	"github.com/casebrophy/planner/foundation/sqldb"
	"github.com/casebrophy/planner/foundation/web"
)

type app struct {
	taskBus          *taskbus.Business
	contextBus       *contextbus.Business
	emailBus         *emailbus.Business
	clarificationBus *clarificationbus.Business
	threadBus        *threadbus.Business
	observationBus   *observationbus.Business
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
		Status:      taskstatus.Todo,
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
		"id":          task.ID.String(),
		"title":       task.Title,
		"description": task.Description,
		"status":      task.Status.String(),
		"priority":    task.Priority.String(),
		"energy":      task.Energy.String(),
		"duration_min": task.DurationMin,
		"created_at":  task.CreatedAt.Format(time.RFC3339),
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
	tasks, _ := a.taskBus.Query(ctx, filter, taskbus.DefaultOrderBy, page.MustParse("1", "50"))

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
