// Package profile serves the reader's profile, settings, saved addresses and the geography
// list, and ends an account.
package profile

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/cloudinary"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/ids"
)

// Refusal is a rule the request broke; the handler answers 200 null with its code.
type Refusal string

func (r Refusal) Error() string { return string(r) }

// Refusal codes (strings live in httpx/errors.go, documented in docs/error-codes.md).
const (
	ErrProfileInvalid = Refusal(httpx.ErrProfileInvalid)
	ErrPhotoInvalid   = Refusal(httpx.ErrPhotoInvalid)
	ErrAddressInvalid = Refusal(httpx.ErrAddressInvalid)
	ErrAddressUnknown = Refusal(httpx.ErrAddressUnknown)
)

// DeleteHook runs after an account is deleted, so other features can hide the content of the
// deleted reader (listings, Bites). T14 and T17 register theirs.
type DeleteHook func(ctx context.Context, userID string) error

// Service holds the profile rules.
type Service struct {
	DB            *db.DB
	MaxImageBytes int
	RevokeTokens  func(ctx context.Context, userID string) error
	Hooks         []DeleteHook
	Now           func() time.Time
	Log           *slog.Logger
}

func pgText(s string) pgtype.Text { return pgtype.Text{String: s, Valid: s != ""} }

// Profile returns the reader details.
func (s *Service) Profile(ctx context.Context, userID string) (Details, error) {
	row, err := s.DB.Q().GetProfile(ctx, userID)
	if err != nil {
		return Details{}, err
	}
	d := Details{Name: row.Name, Phone: row.Phone}
	if row.PhotoData.Valid {
		d.Photo = &row.PhotoData.String
	}
	return d, nil
}

// SaveProfile checks and saves the details. The phone is stored as 01..., the name trimmed.
func (s *Service) SaveProfile(ctx context.Context, userID string, in Details) (Details, error) {
	if CheckProfile(in.Name, in.Phone) != "" {
		return Details{}, ErrProfileInvalid
	}
	phone := ""
	if m, ok := Mobile(in.Phone); ok {
		phone = m
	}
	photo := pgtype.Text{}
	if in.Photo != nil && *in.Photo != "" {
		if _, err := cloudinary.DecodeImage(*in.Photo, s.MaxImageBytes); err != nil {
			return Details{}, ErrPhotoInvalid
		}
		photo = pgText(*in.Photo)
	}
	name := strings.TrimSpace(in.Name)
	err := s.DB.Q().SaveProfile(ctx, sqlc.SaveProfileParams{ID: userID, Name: name, Phone: phone, PhotoData: photo})
	if err != nil {
		return Details{}, err
	}
	out := Details{Name: name, Phone: phone}
	if photo.Valid {
		out.Photo = &photo.String
	}
	return out, nil
}

// Prefs returns the settings of the reader (the defaults until first saved).
func (s *Service) Prefs(ctx context.Context, userID string) (Prefs, error) {
	row, err := s.DB.Q().GetPrefs(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return DefaultPrefs(), nil
	}
	if err != nil {
		return Prefs{}, err
	}
	return Prefs{Muted: nonNil(row.Muted), ProfileVisible: row.ProfileVisible, ActivityVisible: row.ActivityVisible}, nil
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// SavePrefs saves the settings. Group names the app does not know are dropped.
func (s *Service) SavePrefs(ctx context.Context, userID string, in Prefs) (Prefs, error) {
	muted := []string{}
	for _, g := range MutableGroups { // a fixed order, no duplicates
		if slices.Contains(in.Muted, g) {
			muted = append(muted, g)
		}
	}
	err := s.DB.Q().UpsertPrefs(ctx, sqlc.UpsertPrefsParams{
		UserID: userID, Muted: muted, ProfileVisible: in.ProfileVisible, ActivityVisible: in.ActivityVisible,
	})
	return Prefs{Muted: muted, ProfileVisible: in.ProfileVisible, ActivityVisible: in.ActivityVisible}, err
}

// Muted reports whether the reader muted a notification group (notifications.Sender asks).
func (s *Service) Muted(ctx context.Context, userID, group string) (bool, error) {
	p, err := s.Prefs(ctx, userID)
	return slices.Contains(p.Muted, group), err
}

func toAddress(r sqlc.ListAddressesRow) Address {
	return Address{ID: r.ID, Label: r.Label, Recipient: r.Recipient, Phone: r.Phone, Line: r.Line,
		Upazila: r.Upazila, District: r.District, Division: r.Division, IsDefault: r.IsDefault}
}

// Addresses lists the addresses of the reader, default first, then in the order they were added.
func (s *Service) Addresses(ctx context.Context, userID string) ([]Address, error) {
	return listAddresses(ctx, s.DB.Q(), userID)
}

func listAddresses(ctx context.Context, q *sqlc.Queries, userID string) ([]Address, error) {
	rows, err := q.ListAddresses(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]Address, 0, len(rows))
	for _, r := range rows {
		out = append(out, toAddress(r))
	}
	return out, nil
}

func (s *Service) findAddress(ctx context.Context, q *sqlc.Queries, userID, id string) (sqlc.GetAddressRow, error) {
	row, err := q.GetAddress(ctx, sqlc.GetAddressParams{UserID: userID, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return row, ErrAddressUnknown
	}
	return row, err
}

// SaveAddress adds an address (empty id) or replaces one of the reader. The first address
// becomes the default. It answers the whole list.
func (s *Service) SaveAddress(ctx context.Context, userID string, in Address) ([]Address, error) {
	if CheckAddress(in) != "" {
		return nil, ErrAddressInvalid
	}
	a := TidyAddress(in)
	var out []Address
	err := s.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		if a.ID == "" {
			n, err := q.CountAddresses(ctx, userID)
			if err != nil {
				return err
			}
			err = q.InsertAddress(ctx, sqlc.InsertAddressParams{ID: ids.Address(), UserID: userID, Label: a.Label,
				Recipient: a.Recipient, Phone: a.Phone, Line: a.Line, Upazila: a.Upazila, District: a.District,
				Division: a.Division, IsDefault: n == 0})
			if err != nil {
				return err
			}
		} else {
			if _, err := s.findAddress(ctx, q, userID, a.ID); err != nil {
				return err
			}
			err := q.UpdateAddress(ctx, sqlc.UpdateAddressParams{UserID: userID, ID: a.ID, Label: a.Label,
				Recipient: a.Recipient, Phone: a.Phone, Line: a.Line, Upazila: a.Upazila, District: a.District, Division: a.Division})
			if err != nil {
				return err
			}
		}
		var err error
		out, err = listAddresses(ctx, q, userID)
		return err
	})
	return out, err
}

// MakeDefault makes one address the default.
func (s *Service) MakeDefault(ctx context.Context, userID, id string) ([]Address, error) {
	var out []Address
	err := s.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		if _, err := s.findAddress(ctx, q, userID, id); err != nil {
			return err
		}
		if err := setDefault(ctx, q, userID, id); err != nil {
			return err
		}
		var err error
		out, err = listAddresses(ctx, q, userID)
		return err
	})
	return out, err
}

func setDefault(ctx context.Context, q *sqlc.Queries, userID, id string) error {
	if err := q.ClearDefaultAddress(ctx, userID); err != nil {
		return err
	}
	return q.SetDefaultAddress(ctx, sqlc.SetDefaultAddressParams{UserID: userID, ID: id})
}

// DeleteAddress removes an address. Deleting the default makes the next one the default.
func (s *Service) DeleteAddress(ctx context.Context, userID, id string) ([]Address, error) {
	var out []Address
	err := s.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		gone, err := s.findAddress(ctx, q, userID, id)
		if err != nil {
			return err
		}
		if err := q.DeleteAddress(ctx, sqlc.DeleteAddressParams{UserID: userID, ID: id}); err != nil {
			return err
		}
		if gone.IsDefault {
			next, err := q.FirstAddressID(ctx, userID)
			if err == nil {
				if err := setDefault(ctx, q, userID, next); err != nil {
					return err
				}
			} else if !errors.Is(err, pgx.ErrNoRows) {
				return err
			}
		}
		out, err = listAddresses(ctx, q, userID)
		return err
	})
	return out, err
}

// DeleteAccount soft-deletes and anonymises the account, signs it out everywhere, and lets
// other features hide the content of the reader.
func (s *Service) DeleteAccount(ctx context.Context, userID string) error {
	err := s.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		at := pgtype.Timestamptz{Time: s.Now(), Valid: true}
		if err := q.AnonymiseUser(ctx, sqlc.AnonymiseUserParams{ID: userID, DeletedAt: at}); err != nil {
			return err
		}
		if err := q.DeletePrefs(ctx, userID); err != nil {
			return err
		}
		return q.DeleteAddresses(ctx, userID)
	})
	if err != nil {
		return err
	}
	if err := s.RevokeTokens(ctx, userID); err != nil {
		return err
	}
	for _, h := range s.Hooks {
		if err := h(ctx, userID); err != nil {
			s.Log.Error("account delete hook failed", "user", userID, "error", err) // best effort
		}
	}
	return nil
}

// AddressIn finds one address of a reader inside a transaction (checkout delivers to one of them).
func (s *Service) AddressIn(ctx context.Context, q *sqlc.Queries, userID, id string) (Address, bool, error) {
	row, err := q.GetAddress(ctx, sqlc.GetAddressParams{UserID: userID, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return Address{}, false, nil
	}
	if err != nil {
		return Address{}, false, err
	}
	return toAddress(sqlc.ListAddressesRow(row)), true, nil
}
