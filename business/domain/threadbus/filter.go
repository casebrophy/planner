package threadbus

import (
	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/types/threadentrykind"
)

type QueryFilter struct {
	SubjectType    *string
	SubjectID      *uuid.UUID
	Kind           *threadentrykind.Kind
	RequiresAction *bool
}
