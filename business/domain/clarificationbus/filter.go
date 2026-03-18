package clarificationbus

import (
	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/types/clarificationkind"
	"github.com/casebrophy/planner/business/types/clarificationstatus"
)

type QueryFilter struct {
	Status      *clarificationstatus.Status
	Kind        *clarificationkind.Kind
	SubjectType *string
	SubjectID   *uuid.UUID
}
