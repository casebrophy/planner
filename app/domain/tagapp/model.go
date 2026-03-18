package tagapp

import (
	"encoding/json"

	"github.com/casebrophy/planner/business/domain/tagbus"
)

type Tag struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (t Tag) Encode() ([]byte, string, error) {
	data, err := json.Marshal(t)
	return data, "application/json", err
}

type NewTag struct {
	Name string `json:"name"`
}

func toAppTag(t tagbus.Tag) Tag {
	return Tag{
		ID:   t.ID.String(),
		Name: t.Name,
	}
}

func toAppTags(ts []tagbus.Tag) []Tag {
	tags := make([]Tag, len(ts))
	for i, t := range ts {
		tags[i] = toAppTag(t)
	}
	return tags
}

func toBusNewTag(nt NewTag) tagbus.NewTag {
	return tagbus.NewTag{
		Name: nt.Name,
	}
}
