package commands

import (
	"bytes"
	"context"
	"flag"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/casebrophy/planner/foundation/logger"
)

// DebriefdedupeCMD implements the debrief-dedupe admin command.
type DebriefdedupeCMD struct {
	dryRun bool
	limit  int
}

// Run executes the debrief-dedupe command.
func (cmd *DebriefdedupeCMD) Run(
	ctx context.Context,
	log *logger.Logger,
	db *sqlx.DB,
	args []string,
) error {
	fs := flag.NewFlagSet("debrief-dedupe", flag.ContinueOnError)
	fs.BoolVar(&cmd.dryRun, "dry-run", false, "count duplicates but do not modify DB")
	fs.IntVar(&cmd.limit, "limit", 1000, "max duplicate groups to process (max 5000)")

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}

	if cmd.limit > 5000 {
		cmd.limit = 5000
	}
	if cmd.limit < 1 {
		cmd.limit = 1
	}

	log.Info(ctx, "debrief-dedupe", "status", "starting deduplication",
		"dry_run", cmd.dryRun, "limit", cmd.limit)

	// Query groups of duplicate task_debrief clarifications for recurring tasks
	groups, err := cmd.findDuplicateGroups(ctx, log, db)
	if err != nil {
		return fmt.Errorf("find duplicate groups: %w", err)
	}

	if len(groups) == 0 {
		log.Info(ctx, "debrief-dedupe", "status", "no duplicates found")
		return nil
	}

	totalDismissed := 0
	groupsProcessed := 0

	// Process each group
	for i, group := range groups {
		if groupsProcessed >= cmd.limit {
			break
		}

		// Keep the most recent (index 0), dismiss all others
		dismissed, err := cmd.processGroup(ctx, log, db, group, cmd.dryRun)
		if err != nil {
			return fmt.Errorf("process group %d: %w", i, err)
		}

		totalDismissed += dismissed
		groupsProcessed++
	}

	log.Info(ctx, "debrief-dedupe", "status", "complete",
		"groups_processed", groupsProcessed, "total_dismissed", totalDismissed, "dry_run", cmd.dryRun)

	return nil
}

// duplicateGroup represents a group of duplicate debriefs for the same recurring task parent.
type duplicateGroup struct {
	recurrenceParentID string
	clarificationIDs   []string // most recent first (created_at DESC)
}

// findDuplicateGroups queries clarifications table for task_debrief cards
// grouped by recurrence_parent_id (from tasks), keeping order by created_at DESC.
func (cmd *DebriefdedupeCMD) findDuplicateGroups(
	ctx context.Context,
	log *logger.Logger,
	db *sqlx.DB,
) ([]duplicateGroup, error) {
	const q = `
	WITH task_debriefs AS (
		SELECT
			c.clarification_id,
			c.subject_id,
			c.created_at,
			t.recurrence_parent_id,
			ROW_NUMBER() OVER (PARTITION BY t.recurrence_parent_id ORDER BY c.created_at DESC) as rn
		FROM clarification_items c
		INNER JOIN tasks t ON c.subject_id = t.task_id
		WHERE c.kind = 'task_debrief'
		  AND c.status IN ('pending', 'snoozed')
		  AND t.recurrence_rule IS NOT NULL
		  AND t.recurrence_parent_id IS NOT NULL
		  AND c.subject_type = 'task'
	)
	SELECT
		recurrence_parent_id,
		STRING_AGG(clarification_id::TEXT, ',' ORDER BY rn ASC) as clarification_ids
	FROM task_debriefs
	GROUP BY recurrence_parent_id
	HAVING COUNT(*) > 1
	ORDER BY COUNT(*) DESC
	LIMIT $1
	`

	rows, err := db.QueryxContext(ctx, q, cmd.limit)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var groups []duplicateGroup
	for rows.Next() {
		var (
			parentID           string
			clarificationIDStr string
		)

		if err := rows.Scan(&parentID, &clarificationIDStr); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}

		// Parse comma-separated IDs
		var ids []string
		if clarificationIDStr != "" {
			// Re-parse the aggregated string
			var buf bytes.Buffer
			for _, ch := range clarificationIDStr {
				if ch == ',' {
					if buf.Len() > 0 {
						ids = append(ids, buf.String())
						buf.Reset()
					}
				} else {
					buf.WriteRune(ch)
				}
			}
			if buf.Len() > 0 {
				ids = append(ids, buf.String())
			}
		}

		if len(ids) < 2 {
			continue
		}

		group := duplicateGroup{
			recurrenceParentID: parentID,
			clarificationIDs:   ids,
		}
		groups = append(groups, group)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return groups, nil
}

// processGroup dismisses all but the most recent clarification in a group.
func (cmd *DebriefdedupeCMD) processGroup(
	ctx context.Context,
	log *logger.Logger,
	db *sqlx.DB,
	group duplicateGroup,
	dryRun bool,
) (int, error) {
	if len(group.clarificationIDs) < 2 {
		return 0, nil
	}

	// Keep the first (most recent), dismiss the rest
	toDismiss := group.clarificationIDs[1:]
	dismissCount := len(toDismiss)

	if dryRun {
		log.Info(ctx, "debrief-dedupe", "group", group.recurrenceParentID,
			"total", len(group.clarificationIDs), "would_dismiss", dismissCount)
		return dismissCount, nil
	}

	// Build UPDATE query with IN clause
	var buf bytes.Buffer
	buf.WriteString("UPDATE clarification_items SET status = 'dismissed' WHERE clarification_id IN (")
	for i, id := range toDismiss {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString("'")
		buf.WriteString(id)
		buf.WriteString("'")
	}
	buf.WriteString(")")

	if _, err := db.ExecContext(ctx, buf.String()); err != nil {
		return 0, fmt.Errorf("exec: %w", err)
	}

	log.Info(ctx, "debrief-dedupe", "group", group.recurrenceParentID,
		"dismissed", dismissCount)

	return dismissCount, nil
}
