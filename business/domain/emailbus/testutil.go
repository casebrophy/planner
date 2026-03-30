package emailbus

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
)

// TestGenerateNewEmails generates n unique NewEmail values for testing.
// rawInputID must reference an existing raw_input row.
func TestGenerateNewEmails(n int, rawInputID uuid.UUID) []NewEmail {
	emails := make([]NewEmail, n)
	idx := rand.Intn(10000)
	for i := range emails {
		idx++
		emails[i] = NewEmail{
			RawInputID:  rawInputID,
			FromAddress: fmt.Sprintf("sender%d@example.com", idx),
			ToAddress:   "inbox@example.com",
			Subject:     fmt.Sprintf("Subject %d", idx),
			BodyText:    fmt.Sprintf("Body text %d", idx),
			ReceivedAt:  time.Now(),
		}
	}
	return emails
}

// TestSeedEmails creates n emails in the database and returns them.
// rawInputID must reference an existing raw_input row.
func TestSeedEmails(ctx context.Context, n int, rawInputID uuid.UUID, api *Business) ([]Email, error) {
	newEmails := TestGenerateNewEmails(n, rawInputID)
	emails := make([]Email, len(newEmails))
	for i, ne := range newEmails {
		e, err := api.Create(ctx, ne)
		if err != nil {
			return nil, fmt.Errorf("seeding email: idx: %d : %w", i, err)
		}
		emails[i] = e
	}
	return emails, nil
}
