package noteapp

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/notebus"
)

type Note struct {
	ID         string  `json:"id"`
	ContextID  *string `json:"contextId,omitempty"`
	Content    string  `json:"content"`
	Source     string  `json:"source"`
	RawInputID *string `json:"rawInputId,omitempty"`
	CreatedAt  string  `json:"createdAt"`
	UpdatedAt  string  `json:"updatedAt"`
}

func (n Note) Encode() ([]byte, string, error) {
	data, err := json.Marshal(n)
	return data, "application/json", err
}

type NewNote struct {
	ContextID *string `json:"contextId"`
	Content   string  `json:"content"`
	Source    string  `json:"source"`
}

type UpdateNote struct {
	ContextID *string `json:"contextId"`
	Content   *string `json:"content"`
	Source    *string `json:"source"`
}

func toAppNote(n notebus.Note) Note {
	an := Note{
		ID:        n.ID.String(),
		Content:   n.Content,
		Source:    n.Source,
		CreatedAt: n.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: n.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if n.ContextID != nil {
		s := n.ContextID.String()
		an.ContextID = &s
	}
	if n.RawInputID != nil {
		s := n.RawInputID.String()
		an.RawInputID = &s
	}

	return an
}

func toAppNotes(ns []notebus.Note) []Note {
	notes := make([]Note, len(ns))
	for i, n := range ns {
		notes[i] = toAppNote(n)
	}
	return notes
}

func toBusNewNote(nn NewNote) (notebus.NewNote, error) {
	bnn := notebus.NewNote{
		Content: nn.Content,
		Source:  nn.Source,
	}

	if nn.ContextID != nil {
		id, err := uuid.Parse(*nn.ContextID)
		if err != nil {
			return notebus.NewNote{}, fmt.Errorf("contextId: %w", err)
		}
		bnn.ContextID = &id
	}

	return bnn, nil
}

func toBusUpdateNote(un UpdateNote) (notebus.UpdateNote, error) {
	var bun notebus.UpdateNote

	if un.ContextID != nil {
		id, err := uuid.Parse(*un.ContextID)
		if err != nil {
			return notebus.UpdateNote{}, fmt.Errorf("contextId: %w", err)
		}
		bun.ContextID = &id
	}

	bun.Content = un.Content
	bun.Source = un.Source

	return bun, nil
}
