package observationdb

import (
	"bytes"

	"github.com/casebrophy/planner/business/domain/observationbus"
)

func applyFilter(filter observationbus.QueryFilter, data map[string]any, buf *bytes.Buffer) {
	if filter.SubjectType != nil {
		buf.WriteString(" AND subject_type = :filter_subject_type")
		data["filter_subject_type"] = *filter.SubjectType
	}
	if filter.SubjectID != nil {
		buf.WriteString(" AND subject_id = :filter_subject_id")
		data["filter_subject_id"] = *filter.SubjectID
	}
	if filter.Kind != nil {
		buf.WriteString(" AND kind = :filter_kind")
		data["filter_kind"] = filter.Kind.String()
	}
}
