package entitylinkdb

import (
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/entitylinkbus"
)

type entityLinkDB struct {
	ID         uuid.UUID `db:"link_id"`
	SourceType string    `db:"source_type"`
	SourceID   uuid.UUID `db:"source_id"`
	TargetType string    `db:"target_type"`
	TargetID   uuid.UUID `db:"target_id"`
	Confidence float64   `db:"confidence"`
	Kind       string    `db:"kind"`
	CreatedAt  time.Time `db:"created_at"`
}

func toDBEntityLink(l entitylinkbus.EntityLink) entityLinkDB {
	return entityLinkDB{
		ID:         l.ID,
		SourceType: l.SourceType,
		SourceID:   l.SourceID,
		TargetType: l.TargetType,
		TargetID:   l.TargetID,
		Confidence: l.Confidence,
		Kind:       l.Kind,
		CreatedAt:  l.CreatedAt,
	}
}

func toBusEntityLink(l entityLinkDB) entitylinkbus.EntityLink {
	return entitylinkbus.EntityLink{
		ID:         l.ID,
		SourceType: l.SourceType,
		SourceID:   l.SourceID,
		TargetType: l.TargetType,
		TargetID:   l.TargetID,
		Confidence: l.Confidence,
		Kind:       l.Kind,
		CreatedAt:  l.CreatedAt,
	}
}
