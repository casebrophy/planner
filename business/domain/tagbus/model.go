package tagbus

import "github.com/google/uuid"

type Tag struct {
	ID   uuid.UUID
	Name string
}

type NewTag struct {
	Name string
}

type QueryFilter struct {
	ID   *uuid.UUID
	Name *string
}
