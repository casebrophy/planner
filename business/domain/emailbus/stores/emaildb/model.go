package emaildb

import (
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/emailbus"
)

type emailDB struct {
	ID          uuid.UUID  `db:"email_id"`
	RawInputID  uuid.UUID  `db:"raw_input_id"`
	MessageID   *string    `db:"message_id"`
	FromAddress string     `db:"from_address"`
	FromName    *string    `db:"from_name"`
	ToAddress   string     `db:"to_address"`
	Subject     string     `db:"subject"`
	BodyText    string     `db:"body_text"`
	BodyHTML    *string    `db:"body_html"`
	ReceivedAt  time.Time  `db:"received_at"`
	ContextID   *uuid.UUID `db:"context_id"`
	CreatedAt   time.Time  `db:"created_at"`
}

func toDBEmail(e emailbus.Email) emailDB {
	return emailDB{
		ID:          e.ID,
		RawInputID:  e.RawInputID,
		MessageID:   e.MessageID,
		FromAddress: e.FromAddress,
		FromName:    e.FromName,
		ToAddress:   e.ToAddress,
		Subject:     e.Subject,
		BodyText:    e.BodyText,
		BodyHTML:    e.BodyHTML,
		ReceivedAt:  e.ReceivedAt,
		ContextID:   e.ContextID,
		CreatedAt:   e.CreatedAt,
	}
}

func toBusEmail(e emailDB) emailbus.Email {
	return emailbus.Email{
		ID:          e.ID,
		RawInputID:  e.RawInputID,
		MessageID:   e.MessageID,
		FromAddress: e.FromAddress,
		FromName:    e.FromName,
		ToAddress:   e.ToAddress,
		Subject:     e.Subject,
		BodyText:    e.BodyText,
		BodyHTML:    e.BodyHTML,
		ReceivedAt:  e.ReceivedAt,
		ContextID:   e.ContextID,
		CreatedAt:   e.CreatedAt,
	}
}

func toBusEmails(es []emailDB) []emailbus.Email {
	items := make([]emailbus.Email, len(es))
	for i, e := range es {
		items[i] = toBusEmail(e)
	}
	return items
}
