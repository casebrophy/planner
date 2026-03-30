package transactionbus

import "github.com/google/uuid"

type QueryFilter struct {
	ContextID *uuid.UUID
	Source    *string
	Reviewed  *bool
	Category  *string
}
