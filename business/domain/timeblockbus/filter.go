package timeblockbus

import (
	"time"

	"github.com/google/uuid"
)

type QueryFilter struct {
	TaskID    *uuid.UUID
	DateFrom  *time.Time
	DateTo    *time.Time
	Confirmed *bool
}
