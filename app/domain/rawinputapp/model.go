package rawinputapp

import (
	"encoding/json"
	"time"

	"github.com/casebrophy/planner/business/domain/rawinputbus"
)

type RawInput struct {
	ID          string  `json:"id"`
	SourceType  string  `json:"sourceType"`
	Status      string  `json:"status"`
	RawContent  string  `json:"rawContent"`
	ProcessedAt *string `json:"processedAt,omitempty"`
	Error       *string `json:"error,omitempty"`
	CreatedAt   string  `json:"createdAt"`
}

func (r RawInput) Encode() ([]byte, string, error) {
	data, err := json.Marshal(r)
	return data, "application/json", err
}

func toAppRawInput(ri rawinputbus.RawInput) RawInput {
	a := RawInput{
		ID:         ri.ID.String(),
		SourceType: ri.SourceType.String(),
		Status:     ri.Status.String(),
		RawContent: ri.RawContent,
		Error:      ri.Error,
		CreatedAt:  ri.CreatedAt.Format(time.RFC3339),
	}

	if ri.ProcessedAt != nil {
		s := ri.ProcessedAt.Format(time.RFC3339)
		a.ProcessedAt = &s
	}

	return a
}

func toAppRawInputs(ris []rawinputbus.RawInput) []RawInput {
	items := make([]RawInput, len(ris))
	for i, ri := range ris {
		items[i] = toAppRawInput(ri)
	}
	return items
}
