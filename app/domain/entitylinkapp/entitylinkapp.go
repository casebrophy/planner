package entitylinkapp

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/app/sdk/errs"
	"github.com/casebrophy/planner/business/domain/entitylinkbus"
	"github.com/casebrophy/planner/business/sdk/sqldb"
	"github.com/casebrophy/planner/foundation/web"
)

type app struct {
	entityLinkBus *entitylinkbus.Business
}

// queryByEntity returns all links (source or target) for a given entity.
// Query params: entity_type (task|note|event), entity_id (UUID)
func (a *app) queryByEntity(ctx context.Context, r *http.Request) web.Encoder {
	entityType := r.URL.Query().Get("entity_type")
	if entityType == "" {
		return errs.Newf(errs.InvalidArgument, "entity_type is required")
	}

	entityIDStr := r.URL.Query().Get("entity_id")
	entityID, err := uuid.Parse(entityIDStr)
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	links, err := a.entityLinkBus.QueryByEntity(ctx, entityType, entityID)
	if err != nil {
		return errs.Newf(errs.Internal, "query: %s", err)
	}

	items := make([]EntityLink, len(links))
	for i, l := range links {
		items[i] = toAppEntityLink(l)
	}

	return EntityLinks{Items: items, Total: len(items)}
}

// create creates a new manual entity link.
func (a *app) create(ctx context.Context, r *http.Request) web.Encoder {
	var input NewEntityLink
	if err := web.Decode(r, &input); err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	busNew, err := toBusNewEntityLink(input)
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	link, err := a.entityLinkBus.Create(ctx, busNew)
	if err != nil {
		return errs.Newf(errs.Internal, "create: %s", err)
	}

	return toAppEntityLink(link)
}

// delete removes an entity link by ID.
func (a *app) delete(ctx context.Context, r *http.Request) web.Encoder {
	id, err := uuid.Parse(web.Param(r, "link_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	// Pre-flight check: verify the link exists before attempting deletion
	_, err = a.entityLinkBus.QueryByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return errs.New(errs.NotFound, err)
		}
		return errs.Newf(errs.Internal, "query by id: %s", err)
	}

	if err := a.entityLinkBus.Delete(ctx, id); err != nil {
		return errs.Newf(errs.Internal, "delete: %s", err)
	}

	return web.NoResponse{}
}
