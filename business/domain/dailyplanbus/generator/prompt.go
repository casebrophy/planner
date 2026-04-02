package generator

import (
	"encoding/json"
	"fmt"
)

func buildPlanPrompt(tasks []TaskRef, events []EventRef, carryover []CarryoverItem) string {
	tasksJSON, _ := json.Marshal(tasks)
	eventsJSON, _ := json.Marshal(events)
	carryoverJSON, _ := json.Marshal(carryover)

	return fmt.Sprintf(`You are a personal daily planner. Create a prioritized, grouped plan for today.

Open tasks:
%s

Today's events (fixed commitments — these block time, don't schedule tasks during them):
%s

Tasks carried forward from yesterday (not completed):
%s

Create a daily plan by:
1. Group tasks by context, errand type, or energy level (e.g., "Errands", "Deep Work", "Admin", "Home", or use context titles)
2. Within each group, order by: urgency (due date approaching) → priority → energy (high-energy first for morning)
3. Estimate duration for each task in minutes (ai_duration_min). Be realistic — short tasks ~15min, medium ~30min, long ~60min+
4. Consider events as time constraints — if user has a meeting at 2pm, front-load important tasks
5. Surface prerequisite relationships in priority_reason (e.g., "Needs to happen before road trip Saturday")
6. Carry forward yesterday's incomplete tasks with appropriate priority
7. Don't include more tasks than can reasonably fit in a day (aim for 4-8 hours of task time)

Rules:
- Each task_id must be a UUID from the open tasks list above
- Every task should appear in exactly one group (no duplicates, no omissions of important tasks)
- Group names should be short and descriptive (2-3 words max)
- priority_reason should explain WHY this task is in this position (1 sentence)
- If a task has a due_date soon, mention it in the priority_reason
- Put urgent/high-priority tasks in earlier groups`, string(tasksJSON), string(eventsJSON), string(carryoverJSON))
}
