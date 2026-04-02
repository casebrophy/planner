package eventbus

import (
	"time"

	"github.com/google/uuid"
)

type QueryFilter struct {
	ContextID *uuid.UUID
	DateFrom  *time.Time
	DateTo    *time.Time
}
