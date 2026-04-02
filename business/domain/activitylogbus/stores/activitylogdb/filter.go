package activitylogdb

import (
	"bytes"

	"github.com/casebrophy/planner/business/domain/activitylogbus"
)

func applyFilter(filter activitylogbus.QueryFilter, data map[string]any, buf *bytes.Buffer) {
	if filter.SubjectType != nil {
		buf.WriteString(" AND subject_type = :filter_subject_type")
		data["filter_subject_type"] = *filter.SubjectType
	}
	if filter.SubjectID != nil {
		buf.WriteString(" AND subject_id = :filter_subject_id")
		data["filter_subject_id"] = *filter.SubjectID
	}
	if filter.StartDate != nil {
		buf.WriteString(" AND logged_at >= :start_date")
		data["start_date"] = *filter.StartDate
	}
	if filter.EndDate != nil {
		buf.WriteString(" AND logged_at <= :end_date")
		data["end_date"] = *filter.EndDate
	}
}
