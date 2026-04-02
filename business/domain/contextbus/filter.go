package contextbus

import (
	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/types/contextkind"
)

type QueryFilter struct {
	ID     *uuid.UUID
	Status *Status
	Title  *string
	Kind   *contextkind.Kind
}
