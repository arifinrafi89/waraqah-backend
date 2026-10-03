// Package alerts keeps the price and stock alerts of a reader and tells the reader once, when an
// alert first fires. The Sweeper runs after Staff change prices or stock.
package alerts

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/arifinrafi89/waraqah-backend/internal/feature/catalog"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/notifications"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/auth"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/clock"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/ids"
)

// Alert is BookAlertModel, checked against the catalog today.
type Alert struct {
	ID              string `json:"id"`
	Kind            string `json:"kind"`
	BookID          string `json:"bookId"`
	EditionID       string `json:"editionId"`
	BookTitle       string `json:"bookTitle"`
	CurrentPriceBdt int    `json:"currentPriceBdt"`
	IsTriggered     bool   `json:"isTriggered"`
	TargetPriceBdt  *int   `json:"targetPriceBdt"`
}

// SetInput is the body of /alerts/set.
type SetInput struct {
	Kind           string `json:"kind"`
	BookID         string `json:"bookId"`
	EditionID      string `json:"editionId"`
	TargetPriceBdt *int   `json:"targetPriceBdt"`
}

// Service holds the alert rules.
type Service struct {
	DB     *db.DB
	Books  catalog.Books
	Notify notifications.Sender
	Clock  clock.Clock
	Log    *slog.Logger
}

// triggered: back in stock when the Edition can be ordered; a price drop when the price is at or
// under the target.
func triggered(kind string, e catalog.Edition, target *int) bool {
	if kind == "backInStock" {
		return e.IsOrderable()
	}
	return target != nil && e.PriceBdt <= *target
}

func (s *Service) check(ctx context.Context, a sqlc.Alert) (Alert, bool, error) {
	b, e, ok, err := s.Books.FindEdition(ctx, a.EditionID)
	if err != nil || !ok {
		return Alert{}, false, err
	}
	var target *int
	if a.TargetPriceBdt.Valid {
		v := int(a.TargetPriceBdt.Int32)
		target = &v
	}
	return Alert{ID: a.ID, Kind: a.Kind, BookID: b.ID, EditionID: e.ID, BookTitle: b.Title, CurrentPriceBdt: e.PriceBdt,
		IsTriggered: triggered(a.Kind, e, target), TargetPriceBdt: target}, true, nil
}

// List returns the alerts of a reader, each checked against the catalog today. An alert whose
// book is gone is left out.
func (s *Service) List(ctx context.Context, userID string) ([]Alert, error) {
	rows, err := s.DB.Q().ListAlerts(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := []Alert{}
	for _, r := range rows {
		if a, ok, err := s.check(ctx, r); err != nil {
			return nil, err
		} else if ok {
			out = append(out, a)
		}
	}
	return out, nil
}

// Set saves an alert, replacing one of the same kind on the same Edition. One that already fires
// when set needs no notification: the page shows it. refused is set when the alert is not valid.
func (s *Service) Set(ctx context.Context, userID string, in SetInput) (list []Alert, refused string, err error) {
	if in.Kind != "backInStock" && in.Kind != "priceDrop" {
		refused = httpx.ErrAlertInvalid
	}
	b, e, ok, err := s.Books.FindEdition(ctx, in.EditionID)
	if err != nil {
		return nil, "", err
	}
	if refused == "" && (!ok || b.ID != in.BookID) {
		refused = httpx.ErrAlertInvalid
	}
	if refused == "" {
		err = s.DB.WithTx(ctx, func(q *sqlc.Queries) error {
			if err := q.DeleteAlertsOfKind(ctx, sqlc.DeleteAlertsOfKindParams{UserID: userID, Kind: in.Kind, EditionID: in.EditionID}); err != nil {
				return err
			}
			target := pgtype.Int4{}
			if in.TargetPriceBdt != nil {
				target = pgtype.Int4{Int32: int32(*in.TargetPriceBdt), Valid: true}
			}
			fired := pgtype.Timestamptz{}
			if triggered(in.Kind, e, in.TargetPriceBdt) {
				fired = pgtype.Timestamptz{Time: s.Clock.Now(), Valid: true}
			}
			return q.InsertAlert(ctx, sqlc.InsertAlertParams{ID: ids.Alert(), UserID: userID, Kind: in.Kind, BookID: b.ID,
				EditionID: e.ID, TargetPriceBdt: target, FiredAt: fired})
		})
		if err != nil {
			return nil, "", err
		}
	}
	list, err = s.List(ctx, userID)
	return list, refused, err
}

// Remove drops an alert of the reader.
func (s *Service) Remove(ctx context.Context, userID, id string) ([]Alert, error) {
	if err := s.DB.Q().DeleteAlert(ctx, sqlc.DeleteAlertParams{UserID: userID, ID: id}); err != nil {
		return nil, err
	}
	return s.List(ctx, userID)
}

// Sweep checks every alert that has not fired yet and, for each one that fires now, marks it and
// notifies its reader once (AlertFakeStore.sweep). It implements the sweeper that
// catalog admin calls after a price or stock change.
func (s *Service) Sweep(ctx context.Context) error {
	rows, err := s.DB.Q().ListUnfiredAlerts(ctx)
	if err != nil {
		return err
	}
	var failed error
	for _, r := range rows {
		a, ok, err := s.check(ctx, r)
		if err != nil {
			return err
		}
		if !ok || !a.IsTriggered {
			continue
		}
		n, err := s.DB.Q().MarkAlertFired(ctx, sqlc.MarkAlertFiredParams{ID: r.ID, FiredAt: pgtype.Timestamptz{Time: s.Clock.Now(), Valid: true}})
		if err != nil {
			return err
		}
		if n == 0 {
			continue // another sweep got there first
		}
		if err := notifications.AlertTriggered(ctx, s.Notify, r.UserID, a.BookID, a.BookTitle, a.Kind == "backInStock"); err != nil {
			failed = errors.Join(failed, err)
		}
	}
	return failed
}

// Handler is the HTTP side of the alert endpoints.
type Handler struct{ S *Service }

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	h.S.Log.Error("alerts endpoint failed", "error", err)
	httpx.Error(w, r, http.StatusInternalServerError, httpx.CodeInternal, "something went wrong")
}

// Alerts is AlertFakeApi.alerts. A guest gets an empty list.
func (h *Handler) Alerts(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFrom(r.Context())
	if !ok {
		httpx.JSON(w, []Alert{})
		return
	}
	list, err := h.S.List(r.Context(), u.ID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, list)
}

// Set is AlertFakeApi.set. An invalid alert changes nothing; the list is answered with the code in the header.
func (h *Handler) Set(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	var in SetInput
	if !httpx.Decode(w, r, &in) {
		return
	}
	list, refused, err := h.S.Set(r.Context(), u.ID, in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	if refused != "" {
		w.Header().Set(httpx.ErrorHeader, refused)
	}
	httpx.JSON(w, list)
}

// Remove is AlertFakeApi.remove.
func (h *Handler) Remove(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	var in struct {
		ID string `json:"id"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	list, err := h.S.Remove(r.Context(), u.ID, in.ID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, list)
}

// Routes registers the alert endpoints.
func (h *Handler) Routes(r httpx.Router, a *auth.Middleware) {
	r.Handle("GET /alerts", a.Me(http.HandlerFunc(h.Alerts)))         // AlertFakeApi.alerts
	r.Handle("POST /alerts/set", a.Me(http.HandlerFunc(h.Set)))       // AlertFakeApi.set
	r.Handle("POST /alerts/remove", a.Me(http.HandlerFunc(h.Remove))) // AlertFakeApi.remove
}
