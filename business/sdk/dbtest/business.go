package dbtest

import (
	"github.com/casebrophy/planner/business/domain/activitylogbus"
	"github.com/casebrophy/planner/business/domain/activitylogbus/stores/activitylogdb"
	"github.com/casebrophy/planner/business/domain/classificationcorrectionbus"
	"github.com/casebrophy/planner/business/domain/classificationcorrectionbus/stores/correctiondb"
	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/domain/clarificationbus/stores/clarificationdb"
	"github.com/casebrophy/planner/business/domain/contextbus"
	"github.com/casebrophy/planner/business/domain/contextbus/stores/contextdb"
	"github.com/casebrophy/planner/business/domain/emailbus"
	"github.com/casebrophy/planner/business/domain/emailbus/stores/emaildb"
	"github.com/casebrophy/planner/business/domain/entitylinkbus"
	"github.com/casebrophy/planner/business/domain/entitylinkbus/stores/entitylinkdb"
	"github.com/casebrophy/planner/business/domain/eventbus"
	"github.com/casebrophy/planner/business/domain/eventbus/stores/eventdb"
	"github.com/casebrophy/planner/business/domain/inactivitybus"
	"github.com/casebrophy/planner/business/domain/inactivitybus/stores/inactivitydb"
	"github.com/casebrophy/planner/business/domain/ingestbus"
	"github.com/casebrophy/planner/business/domain/ingestbus/extractor"
	"github.com/casebrophy/planner/business/domain/notebus"
	"github.com/casebrophy/planner/business/domain/notebus/stores/notedb"
	"github.com/casebrophy/planner/business/domain/observationbus"
	"github.com/casebrophy/planner/business/domain/observationbus/stores/observationdb"
	"github.com/casebrophy/planner/business/domain/rawinputbus"
	"github.com/casebrophy/planner/business/domain/rawinputbus/stores/rawinputdb"
	"github.com/casebrophy/planner/business/domain/reclassifybus"
	"github.com/casebrophy/planner/business/domain/tagbus"
	"github.com/casebrophy/planner/business/domain/tagbus/stores/tagdb"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/domain/taskbus/stores/taskdb"
	"github.com/casebrophy/planner/business/domain/threadbus"
	"github.com/casebrophy/planner/business/domain/threadbus/stores/threaddb"
	"github.com/casebrophy/planner/foundation/logger"
	"github.com/jmoiron/sqlx"
)

func newBusDomains(log *logger.Logger, db *sqlx.DB) BusDomain {
	activityLogBus := activitylogbus.NewBusiness(log, activitylogdb.NewStore(log, db))
	taskBus := taskbus.NewBusiness(log, taskdb.NewStore(log, db), taskdb.NewDependencyStore(log, db))
	contextBus := contextbus.NewBusiness(log, contextdb.NewStore(log, db))
	tagBus := tagbus.NewBusiness(log, tagdb.New(log, db))
	clarBus := clarificationbus.NewBusiness(log, clarificationdb.NewStore(log, db))
	emailBus := emailbus.NewBusiness(log, emaildb.NewStore(log, db))
	rawBus := rawinputbus.NewBusiness(log, rawinputdb.NewStore(log, db))
	threadBus := threadbus.NewBusiness(log, threaddb.NewStore(log, db))
	obsBus := observationbus.NewBusiness(log, observationdb.NewStore(log, db))
	eventBus := eventbus.NewBusiness(log, eventdb.NewStore(log, db))
	noteBus := notebus.NewBusiness(log, notedb.NewStore(log, db))
	correctionBus := classificationcorrectionbus.NewBusiness(log, correctiondb.NewStore(log, db))
	ingestBus := ingestbus.NewBusiness(log, rawBus, emailBus, taskBus, contextBus, clarBus, eventBus, &extractor.MockExtractor{}, noteBus, tagBus)
	inactBus := inactivitybus.NewBusiness(log, inactivitydb.NewStore(log, db), clarBus)
	entityLinkBus := entitylinkbus.NewBusiness(log, entitylinkdb.NewStore(log, db))
	reclassifyBus := reclassifybus.NewBusiness(log, taskBus, noteBus, correctionBus, db)

	return BusDomain{
		ActivityLog:              activityLogBus,
		Task:                     taskBus,
		Context:                  contextBus,
		Tag:                      tagBus,
		Clarification:            clarBus,
		Email:                    emailBus,
		RawInput:                 rawBus,
		Thread:                   threadBus,
		Observation:              obsBus,
		Event:                    eventBus,
		Note:                     noteBus,
		Ingest:                   ingestBus,
		Inactivity:               inactBus,
		EntityLink:               entityLinkBus,
		ClassificationCorrection: correctionBus,
		Reclassify:               reclassifyBus,
	}
}
