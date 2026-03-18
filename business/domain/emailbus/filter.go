package emailbus

import "github.com/google/uuid"

type QueryFilter struct {
	ContextID   *uuid.UUID
	FromAddress *string
	Subject     *string
}
