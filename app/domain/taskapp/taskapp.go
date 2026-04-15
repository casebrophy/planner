package taskapp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/app/sdk/errs"
	"github.com/casebrophy/planner/app/sdk/query"
	"github.com/casebrophy/planner/business/domain/debriefbus"
	"github.com/casebrophy/planner/business/domain/embeddingbus"
	"github.com/casebrophy/planner/business/domain/knowledgegapbus"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/domain/threadbus"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/sdk/sqldb"
	"github.com/casebrophy/planner/business/types/taskstatus"
	"github.com/casebrophy/planner/business/types/threadentrykind"
	"github.com/casebrophy/planner/business/types/threadsource"
	"github.com/casebrophy/planner/foundation/logger"
	"github.com/casebrophy/planner/foundation/web"
)

type app struct {
	log          *logger.Logger
	taskBus      *taskbus.Business
	threadBus    *threadbus.Business
	debriefBus   *debriefbus.Business
	embeddingBus *embeddingbus.Business
	gapBus       *knowledgegapbus.Business
}

func (a *app) create(ctx context.Context, r *http.Request) web.Encoder {
	var input NewTask
	if err := web.Decode(r, &input); err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	if input.Title == "" {
		return errs.Newf(errs.InvalidArgument, "title is required")
	}

	bt, err := toBusNewTask(input)
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	task, err := a.taskBus.Create(ctx, bt)
	if err != nil {
		return errs.Newf(errs.Internal, "create: %s", err)
	}

	if a.threadBus != nil {
		go func() {
			_, _ = a.threadBus.AddEntry(context.Background(), threadbus.NewThreadEntry{
				SubjectType: "task",
				SubjectID:   task.ID,
				Kind:        threadentrykind.Update,
				Source:      threadsource.System,
				Content:     fmt.Sprintf("Task created: %s", task.Title),
				Extract:     false,
			})
		}()
	}

	if a.embeddingBus != nil {
		go func(id uuid.UUID, title, desc string) {
			content := title
			if desc != "" {
				content = title + "\n" + desc
			}
			a.log.Info(context.Background(), "taskapp.create.embed", "status", "starting", "task_id", id)
			if err := a.embeddingBus.EmbedAndStore(context.Background(), "task", id, content); err != nil {
				a.log.Error(context.Background(), "taskapp.create.embed", "task_id", id, "error", err)
			} else {
				a.log.Info(context.Background(), "taskapp.create.embed", "status", "success", "task_id", id)
			}
		}(task.ID, task.Title, task.Description)
	} else {
		a.log.Info(context.Background(), "taskapp.create.embed", "status", "skipped", "reason", "embeddingBus is nil")
	}

	if a.gapBus != nil {
		go func(id uuid.UUID, title, desc string) {
			content := title
			if desc != "" {
				content = title + "\n" + desc
			}
			_, _ = a.gapBus.Detect(context.Background(), "task", id, content)
		}(task.ID, task.Title, task.Description)
	}

	return toAppTask(task)
}

func (a *app) update(ctx context.Context, r *http.Request) web.Encoder {
	id, err := uuid.Parse(web.Param(r, "task_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	task, err := a.taskBus.QueryByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return errs.New(errs.NotFound, err)
		}
		return errs.Newf(errs.Internal, "query by id: %s", err)
	}

	var input UpdateTask
	if err := web.Decode(r, &input); err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	but, err := toBusUpdateTask(input)
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	updated, err := a.taskBus.Update(ctx, task, but)
	if err != nil {
		return errs.Newf(errs.Internal, "update: %s", err)
	}

	if a.threadBus != nil {
		go func() {
			kind := threadentrykind.Update
			content := fmt.Sprintf("Task updated: %s", updated.Title)
			if but.Status != nil && *but.Status == taskstatus.Done {
				kind = threadentrykind.Milestone
				content = fmt.Sprintf("Task completed: %s", updated.Title)
			}
			_, _ = a.threadBus.AddEntry(context.Background(), threadbus.NewThreadEntry{
				SubjectType: "task",
				SubjectID:   updated.ID,
				Kind:        kind,
				Source:      threadsource.System,
				Content:     content,
				Extract:     false,
			})
		}()
	}

	// Fire debrief on task completion
	if a.debriefBus != nil && but.Status != nil && *but.Status == taskstatus.Done {
		go func() {
			ct := debriefbus.CompletedTask{
				ID:          updated.ID,
				Title:       updated.Title,
				DurationMin: updated.DurationMin,
				CreatedAt:   updated.CreatedAt.Unix(),
				CompletedAt: time.Now().Unix(),
			}
			if err := a.debriefBus.OnTaskCompleted(context.Background(), ct); err != nil {
				_ = err
			}
		}()
	}

	return toAppTask(updated)
}

func (a *app) delete(ctx context.Context, r *http.Request) web.Encoder {
	id, err := uuid.Parse(web.Param(r, "task_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	task, err := a.taskBus.QueryByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return errs.New(errs.NotFound, err)
		}
		return errs.Newf(errs.Internal, "query by id: %s", err)
	}

	if err := a.taskBus.Delete(ctx, task); err != nil {
		return errs.Newf(errs.Internal, "delete: %s", err)
	}

	return web.NoResponse{}
}

func (a *app) deleteBatch(ctx context.Context, r *http.Request) web.Encoder {
	var input DeleteBatchRequest
	if err := web.Decode(r, &input); err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	if len(input.IDs) == 0 {
		return errs.Newf(errs.InvalidArgument, "ids is required and must not be empty")
	}

	ids := make([]uuid.UUID, 0, len(input.IDs))
	for _, raw := range input.IDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			return errs.Newf(errs.InvalidArgument, "invalid id %q: %s", raw, err)
		}
		ids = append(ids, id)
	}

	if err := a.taskBus.DeleteBatch(ctx, ids); err != nil {
		return errs.Newf(errs.Internal, "deletebatch: %s", err)
	}

	return web.NoResponse{}
}

func (a *app) queryAll(ctx context.Context, r *http.Request) web.Encoder {
	pg, err := page.Parse(r.URL.Query().Get("page"), r.URL.Query().Get("rows"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	filter, err := parseFilter(r)
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	orderBy, err := parseOrder(r)
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	tasks, err := a.taskBus.Query(ctx, filter, orderBy, pg)
	if err != nil {
		return errs.Newf(errs.Internal, "query: %s", err)
	}

	total, err := a.taskBus.Count(ctx, filter)
	if err != nil {
		return errs.Newf(errs.Internal, "count: %s", err)
	}

	return query.NewResult(toAppTasks(tasks), total, pg.Number(), pg.RowsPerPage())
}

func (a *app) queryByID(ctx context.Context, r *http.Request) web.Encoder {
	id, err := uuid.Parse(web.Param(r, "task_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	task, err := a.taskBus.QueryByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return errs.New(errs.NotFound, err)
		}
		return errs.Newf(errs.Internal, "query by id: %s", err)
	}

	return toAppTask(task)
}
