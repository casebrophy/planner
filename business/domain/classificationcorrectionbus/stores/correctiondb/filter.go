package correctiondb

import (
	"bytes"

	"github.com/casebrophy/planner/business/domain/classificationcorrectionbus"
)

func applyFilter(filter classificationcorrectionbus.QueryFilter, data map[string]any, buf *bytes.Buffer) {
	if filter.Source != nil {
		buf.WriteString(" AND source = :filter_source")
		data["filter_source"] = *filter.Source
	}
	if filter.PredictedType != nil {
		buf.WriteString(" AND predicted_type = :filter_predicted_type")
		data["filter_predicted_type"] = *filter.PredictedType
	}
	if filter.ActualType != nil {
		buf.WriteString(" AND actual_type = :filter_actual_type")
		data["filter_actual_type"] = *filter.ActualType
	}
}
