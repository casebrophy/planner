package entitylinkapp

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/entitylinkbus"
)

// EntityLink is the app-layer representation of an entity link.
type EntityLink struct {
	ID         string  `json:"id"`
	SourceType string  `json:"sourceType"`
	SourceID   string  `json:"sourceId"`
	TargetType string  `json:"targetType"`
	TargetID   string  `json:"targetId"`
	Confidence float64 `json:"confidence"`
	Kind       string  `json:"kind"`
	CreatedAt  string  `json:"createdAt"`
}

func (e EntityLink) Encode() ([]byte, string, error) {
	data, err := json.Marshal(e)
	return data, "application/json", err
}

// EntityLinks is the list response.
type EntityLinks struct {
	Items []EntityLink `json:"items"`
	Total int          `json:"total"`
}

func (e EntityLinks) Encode() ([]byte, string, error) {
	data, err := json.Marshal(e)
	return data, "application/json", err
}

// NewEntityLink is the request body for creating a link.
type NewEntityLink struct {
	SourceType string `json:"sourceType"`
	SourceID   string `json:"sourceId"`
	TargetType string `json:"targetType"`
	TargetID   string `json:"targetId"`
}

func toAppEntityLink(l entitylinkbus.EntityLink) EntityLink {
	return EntityLink{
		ID:         l.ID.String(),
		SourceType: l.SourceType,
		SourceID:   l.SourceID.String(),
		TargetType: l.TargetType,
		TargetID:   l.TargetID.String(),
		Confidence: l.Confidence,
		Kind:       l.Kind,
		CreatedAt:  l.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func toBusNewEntityLink(n NewEntityLink) (entitylinkbus.NewEntityLink, error) {
	sourceID, err := uuid.Parse(n.SourceID)
	if err != nil {
		return entitylinkbus.NewEntityLink{}, fmt.Errorf("sourceId: %w", err)
	}

	targetID, err := uuid.Parse(n.TargetID)
	if err != nil {
		return entitylinkbus.NewEntityLink{}, fmt.Errorf("targetId: %w", err)
	}

	return entitylinkbus.NewEntityLink{
		SourceType: n.SourceType,
		SourceID:   sourceID,
		TargetType: n.TargetType,
		TargetID:   targetID,
		Kind:       "manual",
		Confidence: 1.0,
	}, nil
}
