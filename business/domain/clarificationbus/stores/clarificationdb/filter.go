package clarificationdb

import (
	"bytes"

	"github.com/casebrophy/planner/business/domain/clarificationbus"
)

func applyFilter(filter clarificationbus.QueryFilter, data map[string]any, buf *bytes.Buffer) {
	if filter.Status != nil {
		buf.WriteString(" AND status = :filter_status")
		data["filter_status"] = filter.Status.String()
	}
	if filter.Kind != nil {
		buf.WriteString(" AND kind = :filter_kind")
		data["filter_kind"] = filter.Kind.String()
	}
	if filter.SubjectType != nil {
		buf.WriteString(" AND subject_type = :filter_subject_type")
		data["filter_subject_type"] = *filter.SubjectType
	}
	if filter.SubjectID != nil {
		buf.WriteString(" AND subject_id = :filter_subject_id")
		data["filter_subject_id"] = *filter.SubjectID
	}
}
