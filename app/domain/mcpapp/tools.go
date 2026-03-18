package mcpapp

var tools = []toolDef{
	{
		Name:        "create_task",
		Description: "Create a new task. Use when the user mentions something they need to do.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"title":       map[string]any{"type": "string", "description": "Short title for the task"},
				"description": map[string]any{"type": "string", "description": "Optional longer description"},
				"priority":    map[string]any{"type": "string", "enum": []string{"low", "medium", "high", "urgent"}, "description": "Task priority, default medium"},
				"energy":      map[string]any{"type": "string", "enum": []string{"low", "medium", "high"}, "description": "Mental effort required, default medium"},
				"due_date":    map[string]any{"type": "string", "description": "ISO 8601 due date if mentioned"},
				"context_id":  map[string]any{"type": "string", "description": "UUID of related context if known"},
				"duration_min": map[string]any{"type": "integer", "description": "Estimated minutes to complete"},
			},
			"required": []string{"title"},
		},
	},
	{
		Name:        "list_tasks",
		Description: "List tasks with optional filters. Use for 'what do I need to do' queries.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"status":     map[string]any{"type": "string", "enum": []string{"todo", "in_progress", "done", "cancelled"}},
				"priority":   map[string]any{"type": "string", "enum": []string{"low", "medium", "high", "urgent"}},
				"context_id": map[string]any{"type": "string", "description": "Filter by context UUID"},
				"page":       map[string]any{"type": "integer", "description": "Page number, default 1"},
				"rows":       map[string]any{"type": "integer", "description": "Rows per page, default 20"},
			},
		},
	},
	{
		Name:        "get_task",
		Description: "Get a single task by ID with full details.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"task_id": map[string]any{"type": "string", "description": "UUID of the task"},
			},
			"required": []string{"task_id"},
		},
	},
	{
		Name:        "update_task",
		Description: "Update a task's fields. Use when the user changes details about a task.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"task_id":      map[string]any{"type": "string", "description": "UUID of the task to update"},
				"title":        map[string]any{"type": "string"},
				"description":  map[string]any{"type": "string"},
				"status":       map[string]any{"type": "string", "enum": []string{"todo", "in_progress", "done", "cancelled"}},
				"priority":     map[string]any{"type": "string", "enum": []string{"low", "medium", "high", "urgent"}},
				"energy":       map[string]any{"type": "string", "enum": []string{"low", "medium", "high"}},
				"due_date":     map[string]any{"type": "string"},
				"duration_min": map[string]any{"type": "integer"},
				"context_id":   map[string]any{"type": "string"},
			},
			"required": []string{"task_id"},
		},
	},
	{
		Name:        "complete_task",
		Description: "Mark a task as done. Use when the user says they finished something.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"task_id": map[string]any{"type": "string", "description": "UUID of the task to complete"},
			},
			"required": []string{"task_id"},
		},
	},
	{
		Name:        "create_context",
		Description: "Create a new context (ongoing situation or project). Use when the user starts tracking a new area of their life.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"title":       map[string]any{"type": "string", "description": "Short title for the context"},
				"description": map[string]any{"type": "string", "description": "What this context is about"},
			},
			"required": []string{"title"},
		},
	},
	{
		Name:        "get_context",
		Description: "Get a context with its summary and metadata. Use for 'what's happening with X' queries.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"context_id": map[string]any{"type": "string", "description": "UUID of the context"},
			},
			"required": []string{"context_id"},
		},
	},
	{
		Name:        "list_contexts",
		Description: "List all active contexts. Use for broad 'what do I have going on' queries.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"status": map[string]any{"type": "string", "enum": []string{"active", "paused", "closed"}, "description": "Filter by status, default active"},
				"page":   map[string]any{"type": "integer"},
				"rows":   map[string]any{"type": "integer"},
			},
		},
	},
	{
		Name:        "update_context",
		Description: "Update a context's title, description, status, or summary.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"context_id":  map[string]any{"type": "string", "description": "UUID of the context to update"},
				"title":       map[string]any{"type": "string"},
				"description": map[string]any{"type": "string"},
				"status":      map[string]any{"type": "string", "enum": []string{"active", "paused", "closed"}},
				"summary":     map[string]any{"type": "string"},
			},
			"required": []string{"context_id"},
		},
	},
}
