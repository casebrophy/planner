package clarificationapp_test

// Resolution dispatcher tests require a running database.
// Run with:
//
//	make db-up && make migrate && make seed
//	go test ./app/domain/clarificationapp/... -run TestDispatch -count=1 -v
//
// Test cases to verify:
// - context_assignment: resolving updates email.context_id or task.context_id
// - stale_task: resolving with status="done" marks task done
// - ambiguous_action: resolving with is_task=true creates a new task
// - ambiguous_deadline: resolving with due_date sets the parsed date
// - new_context: resolving with action="confirm" updates context title/description
// - new_context: resolving with action="merge" deletes the auto-created context
// - context_debrief: resolving records a debrief observation
// - context_debrief: resolving last card updates context.debrief_status to "done"
// - inactivity_prompt: resolving creates a thread entry
// - inactivity_prompt: resolving with action="completed" marks task done
