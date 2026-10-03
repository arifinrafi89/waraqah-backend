package auth

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/db"
)

// ErrNoUser means the user does not exist, was deleted, or is banned.
var ErrNoUser = errors.New("no active user")

// UserSource looks up the current state of a user by id.
type UserSource interface {
	Lookup(ctx context.Context, id string) (User, error)
}

// DBUsers is the UserSource backed by the users table.
type DBUsers struct{ DB *db.DB }

// Lookup returns the active user, or ErrNoUser.
func (s DBUsers) Lookup(ctx context.Context, id string) (User, error) {
	row, err := s.DB.Q().GetUserByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNoUser
	}
	if err != nil {
		return User{}, err
	}
	role, ok := ParseRole(row.Role)
	if !ok || row.Banned {
		return User{}, ErrNoUser
	}
	return User{ID: row.ID, Role: role}, nil
}
