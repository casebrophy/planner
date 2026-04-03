package mcpapp

var tools = []toolDef{
	{
		Name:        "create_task",
		Description: "Create a new task. Use when the user mentions something they need to do.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"title":          map[string]any{"type": "string", "description": "Short title for the task"},
				"description":    map[string]any{"type": "string", "description": "Optional longer description"},
				"priority":       map[string]any{"type": "string", "enum": []string{"low", "medium", "high", "urgent"}, "description": "Task priority, default medium"},
				"energy":         map[string]any{"type": "string", "enum": []string{"low", "medium", "high"}, "description": "Mental effort required, default medium"},
				"due_date":       map[string]any{"type": "string", "description": "ISO 8601 due date if mentioned"},
				"context_id":     map[string]any{"type": "string", "description": "UUID of related context if known"},
				"duration_min":   map[string]any{"type": "integer", "description": "Estimated minutes to complete"},
				"blocked_reason": map[string]any{"type": "string", "description": "Why this task is blocked (manual block reason)"},
				"recurrence_rule": map[string]any{"type": "string", "description": "Recurrence rule, e.g. FREQ=DAILY, FREQ=WEEKLY;BYDAY=MO,WE,FR, FREQ=MONTHLY;BYMONTHDAY=1"},
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
				"status":     map[string]any{"type": "string", "enum": []string{"open", "blocked", "done", "dismissed"}},
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
				"status":       map[string]any{"type": "string", "enum": []string{"open", "blocked", "done", "dismissed"}},
				"priority":     map[string]any{"type": "string", "enum": []string{"low", "medium", "high", "urgent"}},
				"energy":       map[string]any{"type": "string", "enum": []string{"low", "medium", "high"}},
				"due_date":     map[string]any{"type": "string"},
				"duration_min": map[string]any{"type": "integer"},
				"context_id":      map[string]any{"type": "string"},
				"recurrence_rule": map[string]any{"type": "string", "description": "Recurrence rule, e.g. FREQ=DAILY, FREQ=WEEKLY;BYDAY=MO,WE,FR"},
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
				"kind":        map[string]any{"type": "string", "description": "Context kind: 'project' (time-bounded, closeable) or 'area' (ongoing, never closes). Default: project"},
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
	{
		Name:        "list_emails",
		Description: "List ingested emails with optional filters. Use for 'what emails have come in' queries.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"context_id":   map[string]any{"type": "string", "description": "Filter by context UUID"},
				"from_address": map[string]any{"type": "string", "description": "Filter by sender email address"},
				"page":         map[string]any{"type": "integer", "description": "Page number, default 1"},
				"rows":         map[string]any{"type": "integer", "description": "Rows per page, default 20"},
			},
		},
	},
	{
		Name:        "get_email",
		Description: "Get full details of an ingested email by ID.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"email_id": map[string]any{"type": "string", "description": "UUID of the email"},
			},
			"required": []string{"email_id"},
		},
	},
	{
		Name:        "get_clarification_queue",
		Description: "Get pending clarification items that need user review. Returns a prioritized queue of questions the system cannot resolve automatically.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"status": map[string]any{"type": "string", "enum": []string{"pending", "snoozed", "resolved", "dismissed"}, "description": "Filter by status, default pending"},
				"kind":   map[string]any{"type": "string", "enum": []string{"context_assignment", "stale_task", "ambiguous_deadline", "new_context", "overlapping_contexts", "ambiguous_action", "voice_reference", "inactivity_prompt", "context_debrief", "task_debrief"}, "description": "Filter by clarification kind"},
				"page":   map[string]any{"type": "integer", "description": "Page number, default 1"},
				"rows":   map[string]any{"type": "integer", "description": "Rows per page, default 20"},
			},
		},
	},
	{
		Name:        "resolve_clarification",
		Description: "Resolve a clarification item by submitting an answer. Triggers resolution side-effects based on the clarification kind.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"clarification_id": map[string]any{"type": "string", "description": "UUID of the clarification item"},
				"answer":           map[string]any{"description": "The answer to the clarification question (structure depends on the kind)"},
			},
			"required": []string{"clarification_id", "answer"},
		},
	},
	{
		Name:        "snooze_clarification",
		Description: "Snooze a clarification item for a number of hours. It will reappear in the queue after the snooze period.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"clarification_id": map[string]any{"type": "string", "description": "UUID of the clarification item"},
				"hours":            map[string]any{"type": "integer", "description": "Hours to snooze for, default 24"},
			},
			"required": []string{"clarification_id"},
		},
	},
	{
		Name:        "add_thread_entry",
		Description: "Add an update to a task or context thread. Use when the user provides a status update, reports a blocker, makes a decision, or shares any progress note.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"subject_type":    map[string]any{"type": "string", "enum": []string{"task", "context"}, "description": "Type of subject this entry belongs to"},
				"subject_id":      map[string]any{"type": "string", "description": "UUID of the task or context"},
				"kind":            map[string]any{"type": "string", "enum": []string{"update", "blocker", "blocker_resolved", "milestone", "scope_change", "timeline_slip", "external_dep", "decision", "observation", "email", "transaction"}, "description": "Type of thread entry"},
				"content":         map[string]any{"type": "string", "description": "The text content of the entry"},
				"source":          map[string]any{"type": "string", "enum": []string{"user", "voice", "email", "transaction", "system", "claude"}, "description": "Source of the entry, default user"},
				"sentiment":       map[string]any{"type": "string", "enum": []string{"positive", "neutral", "negative", "mixed"}, "description": "Optional sentiment of the entry"},
				"requires_action": map[string]any{"type": "boolean", "description": "Whether this entry requires follow-up action"},
			},
			"required": []string{"subject_type", "subject_id", "kind", "content"},
		},
	},
	{
		Name:        "get_thread",
		Description: "Get the full thread history for a task or context. Use for 'what happened with X' or 'show me the history of X' queries.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"subject_type": map[string]any{"type": "string", "enum": []string{"task", "context"}, "description": "Type of subject"},
				"subject_id":   map[string]any{"type": "string", "description": "UUID of the task or context"},
				"page":         map[string]any{"type": "integer", "description": "Page number, default 1"},
				"rows":         map[string]any{"type": "integer", "description": "Rows per page, default 20"},
			},
			"required": []string{"subject_type", "subject_id"},
		},
	},
	{
		Name:        "record_outcome",
		Description: "Record an outcome observation for a task or context. Used for tracking lessons learned, duration accuracy, blocker profiles, and other retrospective data.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"subject_type": map[string]any{"type": "string", "enum": []string{"task", "context"}, "description": "Type of subject"},
				"subject_id":   map[string]any{"type": "string", "description": "UUID of the task or context"},
				"kind":         map[string]any{"type": "string", "enum": []string{"duration_accuracy", "blocker_profile", "timeline_profile", "lesson", "completion_pattern", "scope_change", "cost_profile"}, "description": "Type of observation"},
				"data":         map[string]any{"description": "Structured observation data (JSON object)"},
				"source":       map[string]any{"type": "string", "enum": []string{"user", "inferred"}, "description": "Source of observation, default user"},
				"confidence":   map[string]any{"type": "number", "description": "Confidence level 0-1, default 1.0"},
			},
			"required": []string{"subject_type", "subject_id", "kind", "data"},
		},
	},
	{
		Name:        "get_outcome_observations",
		Description: "Query outcome observations for a task or context. Returns all recorded observations ordered by most recent first. Use to understand history, patterns, and lessons from past work.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"subject_type": map[string]any{
					"type":        "string",
					"enum":        []string{"task", "context"},
					"description": "Type of subject to query observations for",
				},
				"subject_id": map[string]any{
					"type":        "string",
					"description": "UUID of the task or context",
				},
				"page": map[string]any{
					"type":        "integer",
					"description": "Page number (default 1)",
				},
				"rows_per_page": map[string]any{
					"type":        "integer",
					"description": "Results per page (default 20)",
				},
			},
			"required": []string{"subject_type", "subject_id"},
		},
	},
	{
		Name:        "create_event",
		Description: "Create a new event. Use when the user wants to schedule or record an event (meeting, deadline, milestone, etc.).",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"title":       map[string]any{"type": "string", "description": "Event title"},
				"description": map[string]any{"type": "string", "description": "Optional event description"},
				"location":    map[string]any{"type": "string", "description": "Optional event location"},
				"starts_at":   map[string]any{"type": "string", "description": "ISO 8601 datetime when event starts"},
				"ends_at":     map[string]any{"type": "string", "description": "ISO 8601 datetime when event ends"},
				"all_day":     map[string]any{"type": "boolean", "description": "Whether this is an all-day event, default false"},
				"context_id":  map[string]any{"type": "string", "description": "UUID of related context if known"},
			},
			"required": []string{"title", "starts_at", "ends_at"},
		},
	},
	{
		Name:        "list_events",
		Description: "List events with optional filters. Use for 'what events do I have' or 'show me events in date range' queries.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"date_from":  map[string]any{"type": "string", "description": "Filter events from this ISO 8601 datetime"},
				"date_to":    map[string]any{"type": "string", "description": "Filter events up to this ISO 8601 datetime"},
				"context_id": map[string]any{"type": "string", "description": "Filter by context UUID"},
				"page":       map[string]any{"type": "integer", "description": "Page number, default 1"},
				"rows":       map[string]any{"type": "integer", "description": "Rows per page, default 20"},
			},
		},
	},
	{
		Name:        "get_event",
		Description: "Get a single event by ID with full details.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"event_id": map[string]any{"type": "string", "description": "UUID of the event"},
			},
			"required": []string{"event_id"},
		},
	},
	{
		Name:        "update_event",
		Description: "Update an event's fields. Use when the user changes details about an event.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"event_id":    map[string]any{"type": "string", "description": "UUID of the event to update"},
				"title":       map[string]any{"type": "string"},
				"description": map[string]any{"type": "string"},
				"location":    map[string]any{"type": "string"},
				"starts_at":   map[string]any{"type": "string", "description": "ISO 8601 datetime"},
				"ends_at":     map[string]any{"type": "string", "description": "ISO 8601 datetime"},
				"all_day":     map[string]any{"type": "boolean"},
				"context_id":  map[string]any{"type": "string"},
			},
			"required": []string{"event_id"},
		},
	},
	{
		Name:        "delete_event",
		Description: "Delete an event by ID. Use when the user wants to remove an event.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"event_id": map[string]any{"type": "string", "description": "UUID of the event to delete"},
			},
			"required": []string{"event_id"},
		},
	},
	{
		Name:        "get_daily_plan",
		Description: "Get today's daily plan with grouped tasks and AI suggestions. Returns the most recent plan for the given date.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"date": map[string]any{"type": "string", "description": "Date in YYYY-MM-DD format. Defaults to today."},
			},
		},
	},
	{
		Name:        "generate_daily_plan",
		Description: "Generate or regenerate a daily plan for the given date. Uses AI to group and prioritize open tasks.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"date": map[string]any{"type": "string", "description": "Date in YYYY-MM-DD format. Defaults to today."},
			},
		},
	},
	{
		Name:        "get_schedule",
		Description: "Get a merged schedule of events and time blocks for a date range. Use for 'what's on my calendar' queries.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"date_from": map[string]any{"type": "string", "description": "Start of range, ISO 8601. Defaults to start of today."},
				"date_to":   map[string]any{"type": "string", "description": "End of range, ISO 8601. Defaults to end of today."},
			},
		},
	},
	{
		Name:        "create_time_block",
		Description: "Schedule a task into a specific time slot on the calendar.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"task_id":   map[string]any{"type": "string", "description": "UUID of the task to schedule"},
				"starts_at": map[string]any{"type": "string", "description": "Start time, ISO 8601"},
				"ends_at":   map[string]any{"type": "string", "description": "End time, ISO 8601"},
			},
			"required": []string{"task_id", "starts_at", "ends_at"},
		},
	},
	{
		Name:        "confirm_time_block",
		Description: "Confirm a proposed time block, marking it as committed on the calendar.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"block_id": map[string]any{"type": "string", "description": "UUID of the time block to confirm"},
			},
			"required": []string{"block_id"},
		},
	},
	{
		Name:        "get_inference_context",
		Description: "Get pre-assembled context for a specific inference use case. Returns all relevant data in a single call instead of requiring multiple tool calls.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"use_case": map[string]any{
					"type":        "string",
					"enum":        []string{"daily_plan", "email_extraction", "text_extraction", "thread_classification"},
					"description": "The inference pipeline requesting context",
				},
				"date": map[string]any{
					"type":        "string",
					"description": "ISO 8601 date for date-scoped use cases (daily_plan, text_extraction)",
				},
				"subject_id": map[string]any{
					"type":        "string",
					"description": "UUID of the subject (for thread_classification)",
				},
				"subject_type": map[string]any{
					"type":        "string",
					"enum":        []string{"task", "context"},
					"description": "Type of subject (for thread_classification)",
				},
			},
			"required": []string{"use_case"},
		},
	},
	{
		Name:        "add_task_dependency",
		Description: "Add a dependency between tasks. The first task depends on (is blocked by) the second task.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"task_id":       map[string]any{"type": "string", "description": "ID of the task that depends on another"},
				"depends_on_id": map[string]any{"type": "string", "description": "ID of the task that must be completed first"},
			},
			"required": []string{"task_id", "depends_on_id"},
		},
	},
	{
		Name:        "remove_task_dependency",
		Description: "Remove a dependency between tasks.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"task_id":       map[string]any{"type": "string", "description": "ID of the downstream task"},
				"depends_on_id": map[string]any{"type": "string", "description": "ID of the upstream task to unlink"},
			},
			"required": []string{"task_id", "depends_on_id"},
		},
	},
	{
		Name:        "get_task_dependencies",
		Description: "Get both upstream dependencies (what blocks this task) and downstream dependents (what this task blocks).",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"task_id": map[string]any{"type": "string", "description": "The task ID to query dependencies for"},
			},
			"required": []string{"task_id"},
		},
	},
	{
		Name:        "create_note",
		Description: "Create a new note. Use when the user shares information, knowledge, or reference material that isn't a task or event.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"content":    map[string]any{"type": "string", "description": "The note content"},
				"context_id": map[string]any{"type": "string", "description": "UUID of related context if known"},
				"tags":       map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Tag names to apply to the note"},
			},
			"required": []string{"content"},
		},
	},
	{
		Name:        "search_notes",
		Description: "Search notes by keyword, tag, or context. Use for 'what do I know about X' queries.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"keyword":    map[string]any{"type": "string", "description": "Search text (matches note content)"},
				"tag":        map[string]any{"type": "string", "description": "Filter by tag name"},
				"context_id": map[string]any{"type": "string", "description": "Filter by context UUID"},
				"page":       map[string]any{"type": "integer", "description": "Page number, default 1"},
				"rows":       map[string]any{"type": "integer", "description": "Rows per page, default 20"},
			},
		},
	},
	{
		Name:        "list_notes_by_tag",
		Description: "List all notes with a specific tag. Use when the user asks about a topic that maps to a tag.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"tag_name": map[string]any{"type": "string", "description": "Name of the tag to filter by"},
				"page":     map[string]any{"type": "integer", "description": "Page number, default 1"},
				"rows":     map[string]any{"type": "integer", "description": "Rows per page, default 20"},
			},
			"required": []string{"tag_name"},
		},
	},
	{
		Name:        "log_activity",
		Description: "Log an activity entry for a task or note. Use for habit tracking, check-ins, or recording that something was done.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"subject_type": map[string]any{"type": "string", "enum": []string{"task", "note"}, "description": "Type of item being logged"},
				"subject_id":   map[string]any{"type": "string", "description": "UUID of the task or note"},
				"value":        map[string]any{"type": "string", "description": "Optional value (e.g. '5km', '30min', 'completed')"},
			},
			"required": []string{"subject_type", "subject_id"},
		},
	},
	{
		Name:        "get_streaks",
		Description: "Get streak and frequency info for a tracked item. Use for 'how am I doing with X' or habit tracking queries.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"subject_type": map[string]any{"type": "string", "enum": []string{"task", "note"}, "description": "Type of item"},
				"subject_id":   map[string]any{"type": "string", "description": "UUID of the task or note"},
			},
			"required": []string{"subject_type", "subject_id"},
		},
	},
	{
		Name:        "classify_tasks",
		Description: "Batch classify unlinked tasks into contexts. Assigns high-confidence matches directly and creates clarification cards for low-confidence ones.",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	},
}
