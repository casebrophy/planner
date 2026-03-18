package emailapp

import (
	"encoding/json"
	"time"

	"github.com/casebrophy/planner/business/domain/emailbus"
)

type Email struct {
	ID          string  `json:"id"`
	RawInputID  string  `json:"rawInputId"`
	MessageID   *string `json:"messageId,omitempty"`
	FromAddress string  `json:"fromAddress"`
	FromName    *string `json:"fromName,omitempty"`
	ToAddress   string  `json:"toAddress"`
	Subject     string  `json:"subject"`
	BodyText    string  `json:"bodyText"`
	BodyHTML    *string `json:"bodyHtml,omitempty"`
	ReceivedAt  string  `json:"receivedAt"`
	ContextID   *string `json:"contextId,omitempty"`
	CreatedAt   string  `json:"createdAt"`
}

func (e Email) Encode() ([]byte, string, error) {
	data, err := json.Marshal(e)
	return data, "application/json", err
}

func toAppEmail(e emailbus.Email) Email {
	ae := Email{
		ID:          e.ID.String(),
		RawInputID:  e.RawInputID.String(),
		MessageID:   e.MessageID,
		FromAddress: e.FromAddress,
		FromName:    e.FromName,
		ToAddress:   e.ToAddress,
		Subject:     e.Subject,
		BodyText:    e.BodyText,
		BodyHTML:    e.BodyHTML,
		ReceivedAt:  e.ReceivedAt.Format(time.RFC3339),
		CreatedAt:   e.CreatedAt.Format(time.RFC3339),
	}

	if e.ContextID != nil {
		s := e.ContextID.String()
		ae.ContextID = &s
	}

	return ae
}

func toAppEmails(es []emailbus.Email) []Email {
	items := make([]Email, len(es))
	for i, e := range es {
		items[i] = toAppEmail(e)
	}
	return items
}
