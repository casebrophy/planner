package mid

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/activitylogbus"
	"github.com/casebrophy/planner/foundation/logger"
	"github.com/casebrophy/planner/foundation/web"
)

// CompletionDetail holds the information needed to create activity log entries
// when a domain object is completed.
type CompletionDetail struct {
	SubjectType string
	SubjectID   string
	// ParentID is set for recurring task children so the habit grid can
	// query completions by parent.
	ParentID string
}

// Completable is implemented by response types that represent a completed
// domain object. The middleware uses this to automatically create activity logs.
type Completable interface {
	CompletionInfo() (CompletionDetail, bool)
}

// ActivityLog returns middleware that creates an activity log entry when the
// handler response implements Completable and reports a completion.
func ActivityLog(log *logger.Logger, alBus *activitylogbus.Business) web.MidFunc {
	return func(handler web.HandlerFunc) web.HandlerFunc {
		return func(ctx context.Context, r *http.Request) web.Encoder {
			resp := handler(ctx, r)

			c, ok := resp.(Completable)
			if !ok {
				return resp
			}

			detail, completed := c.CompletionInfo()
			if !completed {
				return resp
			}

			id, err := uuid.Parse(detail.SubjectID)
			if err != nil {
				log.Warn(ctx, "activity log: invalid subject_id", "error", err)
				return resp
			}

			value := "completed"
			go func() {
				// Log against the completed subject itself.
				if _, err := alBus.Create(context.Background(), activitylogbus.NewLog{
					SubjectType: detail.SubjectType,
					SubjectID:   id,
					Value:       &value,
				}); err != nil {
					log.Warn(context.Background(), "activity log creation failed", "error", err)
				}

				// For recurring task children, also log against the parent so
				// the habit grid can query by recurrence parent ID.
				if detail.ParentID != "" {
					parentID, err := uuid.Parse(detail.ParentID)
					if err != nil {
						log.Warn(context.Background(), "activity log: invalid parent_id", "error", err)
						return
					}
					if _, err := alBus.Create(context.Background(), activitylogbus.NewLog{
						SubjectType: detail.SubjectType,
						SubjectID:   parentID,
						Value:       &value,
					}); err != nil {
						log.Warn(context.Background(), "activity log creation failed (parent)", "error", err)
					}
				}
			}()

			return resp
		}
	}
}
