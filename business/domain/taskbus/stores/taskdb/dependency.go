package taskdb

import (
	"github.com/jmoiron/sqlx"

	"github.com/casebrophy/planner/foundation/logger"
)

// DependencyStore handles task dependency persistence.
type DependencyStore struct {
	log *logger.Logger
	db  sqlx.ExtContext
}

// NewDependencyStore constructs a DependencyStore.
func NewDependencyStore(log *logger.Logger, db *sqlx.DB) *DependencyStore {
	return &DependencyStore{
		log: log,
		db:  db,
	}
}
