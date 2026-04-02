package notebus

import "github.com/google/uuid"

type QueryFilter struct {
	ContextID *uuid.UUID
	Source    *string
	Search   *string
}
