package loyalty

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/auth"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/clock"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
)

// Reasons, named as PointsReason in the app.
const (
	Welcome  = "welcome"
	Earned   = "earned"
	Spent    = "spent"
	Refunded = "refunded"
	Reversed = "reversed"
)

// Ledger is points.Ledger (BACKEND_PLAN.md section 8). Every method takes the queries to use,
// so checkout and orders can change the points inside their own transaction; nil means the plain connection.
type Ledger interface {
	Balance(ctx context.Context, q *sqlc.Queries, userID string) (int, error)
	// Spend uses up to wanted points on an order of booksBdt of books and answers how many were used.
	Spend(ctx context.Context, q *sqlc.Queries, userID, orderNumber string, wanted, booksBdt int) (int, error)
	// Earn gives the points for paidBdt paid for books and answers how many.
	Earn(ctx context.Context, q *sqlc.Queries, userID, orderNumber string, paidBdt int) (int, error)
	// Undo gives back what a cancelled order spent and takes back what it earned.
	Undo(ctx context.Context, q *sqlc.Queries, userID, orderNumber string, spent, earned int) error
}

// Entry is PointsEntryModel.
type Entry struct {
	Points      int       `json:"points"`
	Reason      string    `json:"reason"`
	At          time.Time `json:"at"`
	OrderNumber *string   `json:"orderNumber"`
}

// Account is PointsAccountModel: the balance and its history, newest first.
type Account struct {
	Balance int     `json:"balance"`
	Entries []Entry `json:"entries"`
}

// Service implements Ledger and the points endpoint.
type Service struct {
	DB    *db.DB
	Clock clock.Clock
	Loc   *time.Location
	Log   *slog.Logger
}

func (s *Service) queries(q *sqlc.Queries) *sqlc.Queries {
	if q != nil {
		return q
	}
	return s.DB.Q()
}

func (s *Service) add(ctx context.Context, q *sqlc.Queries, userID string, points int, reason, orderNumber string) error {
	return s.queries(q).AddPointsEntry(ctx, sqlc.AddPointsEntryParams{UserID: userID, Points: int32(points), Reason: reason,
		OrderNumber: pgtype.Text{String: orderNumber, Valid: orderNumber != ""}, At: s.Clock.Now()})
}

// Balance is the sum of the entries.
func (s *Service) Balance(ctx context.Context, q *sqlc.Queries, userID string) (int, error) {
	n, err := s.queries(q).PointsBalance(ctx, userID)
	return int(n), err
}

// Spend implements Ledger.
func (s *Service) Spend(ctx context.Context, q *sqlc.Queries, userID, orderNumber string, wanted, booksBdt int) (int, error) {
	bal, err := s.Balance(ctx, q, userID)
	if err != nil {
		return 0, err
	}
	points := min(max(wanted, 0), Usable(bal, booksBdt))
	if points > 0 {
		err = s.add(ctx, q, userID, -points, Spent, orderNumber)
	}
	return points, err
}

// Earn implements Ledger.
func (s *Service) Earn(ctx context.Context, q *sqlc.Queries, userID, orderNumber string, paidBdt int) (int, error) {
	points := EarnedFor(paidBdt)
	if points > 0 {
		return points, s.add(ctx, q, userID, points, Earned, orderNumber)
	}
	return 0, nil
}

// Undo implements Ledger.
func (s *Service) Undo(ctx context.Context, q *sqlc.Queries, userID, orderNumber string, spent, earned int) error {
	if spent > 0 {
		if err := s.add(ctx, q, userID, spent, Refunded, orderNumber); err != nil {
			return err
		}
	}
	if earned > 0 {
		return s.add(ctx, q, userID, -earned, Reversed, orderNumber)
	}
	return nil
}

// Get returns the points account of a reader.
func (s *Service) Get(ctx context.Context, userID string) (Account, error) {
	rows, err := s.DB.Q().ListPointsEntries(ctx, userID)
	if err != nil {
		return Account{}, err
	}
	out := Account{Entries: []Entry{}}
	for _, r := range rows {
		e := Entry{Points: int(r.Points), Reason: r.Reason, At: r.At.In(s.Loc).Truncate(time.Second)}
		if r.OrderNumber.Valid {
			e.OrderNumber = &r.OrderNumber.String
		}
		out.Balance += e.Points
		out.Entries = append(out.Entries, e)
	}
	return out, nil
}

// Handler is the HTTP side of GET /points.
type Handler struct{ S *Service }

// Points is PointsFakeApi.points. A guest gets the empty account.
func (h Handler) Points(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFrom(r.Context())
	if !ok {
		httpx.JSON(w, Account{Entries: []Entry{}})
		return
	}
	out, err := h.S.Get(r.Context(), u.ID)
	if err != nil {
		h.S.Log.Error("points failed", "error", err)
		httpx.Error(w, r, http.StatusInternalServerError, httpx.CodeInternal, "something went wrong")
		return
	}
	httpx.JSON(w, out)
}

// Routes registers GET /points.
func (h Handler) Routes(r httpx.Router, a *auth.Middleware) {
	r.Handle("GET /points", a.Me(http.HandlerFunc(h.Points))) // PointsFakeApi.points
}
