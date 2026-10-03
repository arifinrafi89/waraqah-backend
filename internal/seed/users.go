package seed

import (
	"context"
	"errors"
	"time"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/auth"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
)

type seedUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

// loadUsers creates the demo accounts: admin@, moderator@, catalog@, support@ and reader@waraqah.test.
func loadUsers(ctx context.Context, r *Run) error {
	var users []seedUser
	if err := r.read("users.json", &users); err != nil {
		return err
	}
	hash, err := r.hash()
	if err != nil {
		return err
	}
	since := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return r.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		for _, u := range users {
			if _, ok := auth.ParseRole(u.Role); !ok {
				return errors.New("unknown role " + u.Role + " for " + u.Email)
			}
			err := q.UpsertSeedUser(ctx, sqlc.UpsertSeedUserParams{
				ID: u.ID, Email: pgText(u.Email), Name: u.Name, Role: u.Role,
				PasswordHash: pgText(hash), MemberSince: since,
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
}
