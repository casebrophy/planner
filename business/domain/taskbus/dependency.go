package taskbus

import (
	"context"
	"fmt"
	"time"

	"github.com/casebrophy/planner/business/types/taskstatus"
	"github.com/google/uuid"
)

type Dependency struct {
	TaskID      uuid.UUID
	DependsOnID uuid.UUID
	CreatedAt   time.Time
}

type DependencyStorer interface {
	AddDependency(ctx context.Context, dep Dependency) error
	RemoveDependency(ctx context.Context, taskID, dependsOnID uuid.UUID) error
	QueryDependencies(ctx context.Context, taskID uuid.UUID) ([]Task, error)
	QueryDependents(ctx context.Context, taskID uuid.UUID) ([]Task, error)
	HasUnmetDependencies(ctx context.Context, taskID uuid.UUID) (bool, error)
}

func (b *Business) AddDependency(ctx context.Context, taskID, dependsOnID uuid.UUID) error {
	if taskID == dependsOnID {
		return fmt.Errorf("a task cannot depend on itself")
	}

	// Check for direct cycle: dependsOnID already depends on taskID.
	deps, err := b.depStorer.QueryDependencies(ctx, dependsOnID)
	if err != nil {
		return fmt.Errorf("checking for cycles: %w", err)
	}
	for _, d := range deps {
		if d.ID == taskID {
			return fmt.Errorf("adding this dependency would create a cycle")
		}
	}

	dep := Dependency{
		TaskID:      taskID,
		DependsOnID: dependsOnID,
		CreatedAt:   time.Now().UTC(),
	}

	if err := b.depStorer.AddDependency(ctx, dep); err != nil {
		return fmt.Errorf("adding dependency: %w", err)
	}

	// Auto-block the downstream task if upstream isn't done.
	upstream, err := b.storer.QueryByID(ctx, dependsOnID)
	if err != nil {
		return fmt.Errorf("querying upstream task: %w", err)
	}

	if upstream.Status != taskstatus.Done {
		downstream, err := b.storer.QueryByID(ctx, taskID)
		if err != nil {
			return fmt.Errorf("querying downstream task: %w", err)
		}
		if downstream.Status == taskstatus.Open {
			downstream.Status = taskstatus.Blocked
			downstream.UpdatedAt = time.Now().UTC()
			if err := b.storer.Update(ctx, downstream); err != nil {
				return fmt.Errorf("auto-blocking downstream task: %w", err)
			}
		}
	}

	return nil
}

func (b *Business) RemoveDependency(ctx context.Context, taskID, dependsOnID uuid.UUID) error {
	if err := b.depStorer.RemoveDependency(ctx, taskID, dependsOnID); err != nil {
		return fmt.Errorf("removing dependency: %w", err)
	}
	return b.reevaluateBlocked(ctx, taskID)
}

func (b *Business) QueryDependencies(ctx context.Context, taskID uuid.UUID) ([]Task, error) {
	return b.depStorer.QueryDependencies(ctx, taskID)
}

func (b *Business) QueryDependents(ctx context.Context, taskID uuid.UUID) ([]Task, error) {
	return b.depStorer.QueryDependents(ctx, taskID)
}

func (b *Business) reevaluateBlocked(ctx context.Context, taskID uuid.UUID) error {
	task, err := b.storer.QueryByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("querying task: %w", err)
	}

	if task.Status != taskstatus.Blocked {
		return nil
	}
	if task.BlockedReason != "" {
		return nil
	}

	hasUnmet, err := b.depStorer.HasUnmetDependencies(ctx, taskID)
	if err != nil {
		return fmt.Errorf("checking unmet dependencies: %w", err)
	}

	if !hasUnmet {
		task.Status = taskstatus.Open
		task.UpdatedAt = time.Now().UTC()
		if err := b.storer.Update(ctx, task); err != nil {
			return fmt.Errorf("unblocking task: %w", err)
		}
	}

	return nil
}

func (b *Business) UnblockDependents(ctx context.Context, taskID uuid.UUID) error {
	dependents, err := b.depStorer.QueryDependents(ctx, taskID)
	if err != nil {
		return fmt.Errorf("querying dependents: %w", err)
	}

	for _, dep := range dependents {
		if err := b.reevaluateBlocked(ctx, dep.ID); err != nil {
			return fmt.Errorf("re-evaluating dependent %s: %w", dep.ID, err)
		}
	}

	return nil
}
