package activitylogapp

import (
	"encoding/json"
	"time"

	"github.com/casebrophy/planner/business/domain/activitylogbus"
)

type ActivityLog struct {
	ID          string  `json:"id"`
	SubjectType string  `json:"subjectType"`
	SubjectID   string  `json:"subjectId"`
	Value       *string `json:"value,omitempty"`
	LoggedAt    string  `json:"loggedAt"`
}

func (a ActivityLog) Encode() ([]byte, string, error) {
	data, err := json.Marshal(a)
	return data, "application/json", err
}

type NewActivityLog struct {
	SubjectType string  `json:"subjectType"`
	SubjectID   string  `json:"subjectId"`
	Value       *string `json:"value"`
}

type Streaks struct {
	Current    int     `json:"current"`
	Longest    int     `json:"longest"`
	TotalCount int     `json:"totalCount"`
	LastLogged *string `json:"lastLogged,omitempty"`
}

func (s Streaks) Encode() ([]byte, string, error) {
	data, err := json.Marshal(s)
	return data, "application/json", err
}

func toAppLog(l activitylogbus.Log) ActivityLog {
	return ActivityLog{
		ID:          l.ID.String(),
		SubjectType: l.SubjectType,
		SubjectID:   l.SubjectID.String(),
		Value:       l.Value,
		LoggedAt:    l.LoggedAt.Format(time.RFC3339),
	}
}

func toAppLogs(ls []activitylogbus.Log) []ActivityLog {
	logs := make([]ActivityLog, len(ls))
	for i, l := range ls {
		logs[i] = toAppLog(l)
	}
	return logs
}

func toAppStreaks(info activitylogbus.StreakInfo) Streaks {
	s := Streaks{
		Current:    info.Current,
		Longest:    info.Longest,
		TotalCount: info.TotalCount,
	}
	if info.LastLogged != nil {
		t := info.LastLogged.Format(time.RFC3339)
		s.LastLogged = &t
	}
	return s
}
