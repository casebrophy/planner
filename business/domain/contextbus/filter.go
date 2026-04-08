package contextbus

import (
	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/types/contextkind"
)

type QueryFilter struct {
	ID              *uuid.UUID
	Status          *Status
	Kind            *contextkind.Kind
	Title           *string
	ParentContextID *uuid.UUID
}
