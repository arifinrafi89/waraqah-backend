package seed

import (
	"context"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
)

type seedAddress struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Recipient string `json:"recipient"`
	Phone     string `json:"phone"`
	Line      string `json:"line"`
	Upazila   string `json:"upazila"`
	District  string `json:"district"`
	Division  string `json:"division"`
	IsDefault bool   `json:"isDefault"`
}

// loadAddresses gives the demo reader the two addresses the fake API starts with.
func loadAddresses(ctx context.Context, r *Run) error {
	var list []seedAddress
	if err := r.read("addresses.json", &list); err != nil {
		return err
	}
	return r.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		for _, a := range list {
			err := q.UpsertSeedAddress(ctx, sqlc.UpsertSeedAddressParams{ID: a.ID, UserID: r.Me, Label: a.Label,
				Recipient: a.Recipient, Phone: a.Phone, Line: a.Line, Upazila: a.Upazila, District: a.District,
				Division: a.Division, IsDefault: a.IsDefault})
			if err != nil {
				return err
			}
		}
		return nil
	})
}
