package contextbus

import (
	"github.com/google/uuid"
)

type QueryFilter struct {
	ID     *uuid.UUID
	Status *Status
	Title  *string
}
