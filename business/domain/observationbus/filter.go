package observationbus

import (
	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/types/observationkind"
)

type QueryFilter struct {
	SubjectType *string
	SubjectID   *uuid.UUID
	Kind        *observationkind.Kind
}
