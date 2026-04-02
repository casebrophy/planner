package activitylogdb

import (
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/activitylogbus"
)

type logDB struct {
	ID          uuid.UUID  `db:"log_id"`
	SubjectType string     `db:"subject_type"`
	SubjectID   uuid.UUID  `db:"subject_id"`
	Value       *string    `db:"value"`
	LoggedAt    time.Time  `db:"logged_at"`
}

func toDBLog(l activitylogbus.Log) logDB {
	return logDB{
		ID:          l.ID,
		SubjectType: l.SubjectType,
		SubjectID:   l.SubjectID,
		Value:       l.Value,
		LoggedAt:    l.LoggedAt,
	}
}

func toBusLog(l logDB) activitylogbus.Log {
	return activitylogbus.Log{
		ID:          l.ID,
		SubjectType: l.SubjectType,
		SubjectID:   l.SubjectID,
		Value:       l.Value,
		LoggedAt:    l.LoggedAt,
	}
}

func toBusLogs(ls []logDB) []activitylogbus.Log {
	logs := make([]activitylogbus.Log, len(ls))
	for i, l := range ls {
		logs[i] = toBusLog(l)
	}
	return logs
}
