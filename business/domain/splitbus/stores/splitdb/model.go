package splitdb

import (
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/splitbus"
)

type splitDB struct {
	ID            uuid.UUID `db:"split_id"`
	TransactionID uuid.UUID `db:"transaction_id"`
	PartyName     string    `db:"party_name"`
	Amount        int       `db:"amount"`
	VenmoHandle   *string   `db:"venmo_handle"`
	Settled       bool      `db:"settled"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
}

func toDBSplit(s splitbus.Split) splitDB {
	return splitDB{
		ID:            s.ID,
		TransactionID: s.TransactionID,
		PartyName:     s.PartyName,
		Amount:        s.Amount,
		VenmoHandle:   s.VenmoHandle,
		Settled:       s.Settled,
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
	}
}

func toBusSplit(s splitDB) splitbus.Split {
	return splitbus.Split{
		ID:            s.ID,
		TransactionID: s.TransactionID,
		PartyName:     s.PartyName,
		Amount:        s.Amount,
		VenmoHandle:   s.VenmoHandle,
		Settled:       s.Settled,
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
	}
}

func toBusSplits(ss []splitDB) []splitbus.Split {
	items := make([]splitbus.Split, len(ss))
	for i, s := range ss {
		items[i] = toBusSplit(s)
	}
	return items
}
