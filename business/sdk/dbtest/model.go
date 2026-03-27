package dbtest

import (
	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/domain/contextbus"
	"github.com/casebrophy/planner/business/domain/emailbus"
	"github.com/casebrophy/planner/business/domain/inactivitybus"
	"github.com/casebrophy/planner/business/domain/ingestbus"
	"github.com/casebrophy/planner/business/domain/observationbus"
	"github.com/casebrophy/planner/business/domain/rawinputbus"
	"github.com/casebrophy/planner/business/domain/tagbus"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/domain/threadbus"
	"github.com/casebrophy/planner/foundation/logger"
	"github.com/jmoiron/sqlx"
)

// Database owns state for running and shutting down tests.
type Database struct {
	DB        *sqlx.DB
	Log       *logger.Logger
	BusDomain BusDomain
}

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
	Ingest        *ingestbus.Business
	Inactivity    *inactivitybus.Business
}
