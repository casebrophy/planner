package tagdb

import (
	"github.com/casebrophy/planner/business/domain/tagbus"
	"github.com/google/uuid"
)

type tagDB struct {
	ID   uuid.UUID `db:"tag_id"`
	Name string    `db:"name"`
}

func toDBTag(t tagbus.Tag) tagDB {
	return tagDB{ID: t.ID, Name: t.Name}
}

func toBusTag(t tagDB) tagbus.Tag {
	return tagbus.Tag{ID: t.ID, Name: t.Name}
}

func toBusTags(ts []tagDB) []tagbus.Tag {
	tags := make([]tagbus.Tag, len(ts))
	for i, t := range ts {
		tags[i] = toBusTag(t)
	}
	return tags
}
