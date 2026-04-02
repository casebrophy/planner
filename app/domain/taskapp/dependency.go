package taskapp

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/app/sdk/errs"
	"github.com/casebrophy/planner/foundation/web"
)

// taskList wraps a slice of Task so it implements web.Encoder.
type taskList []Task

func (tl taskList) Encode() ([]byte, string, error) {
	data, err := json.Marshal([]Task(tl))
	return data, "application/json", err
}

func (a *app) addDependency(ctx context.Context, r *http.Request) web.Encoder {
	taskID, err := uuid.Parse(web.Param(r, "task_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	dependsOnID, err := uuid.Parse(web.Param(r, "depends_on_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	if err := a.taskBus.AddDependency(ctx, taskID, dependsOnID); err != nil {
		return errs.New(errs.Internal, err)
	}

	return web.NoResponse{}
}

func (a *app) removeDependency(ctx context.Context, r *http.Request) web.Encoder {
	taskID, err := uuid.Parse(web.Param(r, "task_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	dependsOnID, err := uuid.Parse(web.Param(r, "depends_on_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	if err := a.taskBus.RemoveDependency(ctx, taskID, dependsOnID); err != nil {
		return errs.New(errs.Internal, err)
	}

	return web.NoResponse{}
}

func (a *app) queryDependencies(ctx context.Context, r *http.Request) web.Encoder {
	taskID, err := uuid.Parse(web.Param(r, "task_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	tasks, err := a.taskBus.QueryDependencies(ctx, taskID)
	if err != nil {
		return errs.New(errs.Internal, err)
	}

	return taskList(toAppTasks(tasks))
}

func (a *app) queryDependents(ctx context.Context, r *http.Request) web.Encoder {
	taskID, err := uuid.Parse(web.Param(r, "task_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	tasks, err := a.taskBus.QueryDependents(ctx, taskID)
	if err != nil {
		return errs.New(errs.Internal, err)
	}

	return taskList(toAppTasks(tasks))
}
