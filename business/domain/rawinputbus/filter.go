package rawinputbus

import (
	"github.com/casebrophy/planner/business/types/rawinputsource"
	"github.com/casebrophy/planner/business/types/rawinputstatus"
)

type QueryFilter struct {
	Status     *rawinputstatus.Status
	SourceType *rawinputsource.Source
}
