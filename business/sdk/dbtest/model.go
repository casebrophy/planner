package dbtest

import (
	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/domain/contextbus"
	"github.com/casebrophy/planner/business/domain/emailbus"
	"github.com/casebrophy/planner/business/domain/eventbus"
	"github.com/casebrophy/planner/business/domain/inactivitybus"
	"github.com/casebrophy/planner/business/domain/ingestbus"
	"github.com/casebrophy/planner/business/domain/notebus"
	"github.com/casebrophy/planner/business/domain/observationbus"
	"github.com/casebrophy/planner/business/domain/rawinputbus"
	"github.com/casebrophy/planner/business/domain/tagbus"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/domain/threadbus"
)

// BusDomain represents all the business domain APIs needed for testing.
type BusDomain struct {
	Task          *taskbus.Business
	Context       *contextbus.Business
	Tag           *tagbus.Business
	Clarification *clarificationbus.Business
	Email         *emailbus.Business
	RawInput      *rawinputbus.Business
	Thread        *threadbus.Business
	Observation   *observationbus.Business
	Event         *eventbus.Business
	Note          *notebus.Business
	Ingest        *ingestbus.Business
	Inactivity    *inactivitybus.Business
}
