package generator

import (
	"encoding/json"
	"fmt"
	"strings"
)

type implicationHint struct {
	EventTitle    string   `json:"event_title"`
	EventStartsAt string   `json:"event_starts_at"`
	PrepTaskIDs   []string `json:"prep_task_ids"`
	PrepTaskTitles []string `json:"prep_task_titles"`
}

func buildPlanPrompt(tasks []TaskRef, events []EventRef, carryover []CarryoverItem, implications []ImplicationResult, tzName string) string {
	tasksJSON, _ := json.Marshal(tasks)
	eventsJSON, _ := json.Marshal(events)
	carryoverJSON, _ := json.Marshal(carryover)

	implSection := buildImplicationSection(implications)

	return fmt.Sprintf(`You are a personal daily planner. Create a prioritized, grouped plan for today.

Open tasks:
%s

Today's events (fixed commitments — these block time, don't schedule tasks during them):
%s

All event times are in the user's local timezone (%s). Schedule tasks around these events in the same timezone.

Tasks carried forward from yesterday (not completed):
%s
%s
Create a daily plan by grouping tasks into time-of-day slots based primarily on the task's energy field:

TIME-OF-DAY SLOT RULES (energy is the PRIMARY driver for slot assignment):
- Morning (before ~noon): high-energy tasks — deep work, complex problems, creative tasks
- Mid-day (~noon to ~3pm): medium-energy tasks — meetings, reviews, coordination, lighter focus work
- Afternoon/Evening (after ~3pm): low-energy tasks — admin, errands, email, routine follow-ups

Group naming: combine time-of-day with context/type (e.g., "Morning - Deep Work", "Morning - Dev", "Mid-day - Admin", "Afternoon - Errands"). Groups must reflect the energy-based time-of-day split.

Within each time-of-day slot:
1. Order by urgency (due date approaching) → priority → context
2. Estimate duration for each task in minutes (ai_duration_min). Be realistic — short tasks ~15min, medium ~30min, long ~60min+
3. Consider events as time constraints — if user has a meeting at 2pm, avoid scheduling high-energy tasks that would be interrupted; front-load important morning tasks
4. Surface prerequisite relationships in priority_reason (e.g., "Needs to happen before road trip Saturday")
5. Carry forward yesterday's incomplete tasks in their appropriate energy-based slot
6. Don't include more tasks than can reasonably fit in a day (aim for 4-8 hours of task time)

Rules:
- Each task_id must be a UUID from the open tasks list above
- Every task should appear in exactly one group (no duplicates, no omissions of important tasks)
- Group names should be short and descriptive (2-3 words max)
- priority_reason should explain WHY this task is in this position (1 sentence)
- If a task has a due_date soon, mention it in the priority_reason
- high-energy tasks MUST go in morning groups; low-energy tasks MUST go in afternoon/evening groups`, string(tasksJSON), string(eventsJSON), tzName, string(carryoverJSON), implSection)
}

// buildImplicationSection formats detected event-task implications as a prompt section.
// Groups implications by event so the prompt stays concise.
func buildImplicationSection(implications []ImplicationResult) string {
	if len(implications) == 0 {
		return ""
	}

	// Group by event ID.
	type eventGroup struct {
		hint implicationHint
	}
	order := []string{}
	byEvent := map[string]*implicationHint{}

	for _, imp := range implications {
		if _, ok := byEvent[imp.EventID]; !ok {
			order = append(order, imp.EventID)
			byEvent[imp.EventID] = &implicationHint{
				EventTitle:    imp.EventTitle,
				EventStartsAt: imp.EventStartsAt.Format("15:04"),
			}
		}
		h := byEvent[imp.EventID]
		h.PrepTaskIDs = append(h.PrepTaskIDs, imp.TaskID)
		h.PrepTaskTitles = append(h.PrepTaskTitles, imp.TaskTitle)
	}

	var sb strings.Builder
	sb.WriteString("\nDetected event-task prerequisites (schedule these prep tasks BEFORE their events and mention the event in priority_reason):\n")
	for _, evID := range order {
		h := byEvent[evID]
		hJSON, _ := json.Marshal(h)
		sb.WriteString(string(hJSON))
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
	return sb.String()
}
