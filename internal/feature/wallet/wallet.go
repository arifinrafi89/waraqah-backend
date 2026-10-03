// Package wallet keeps the taka a reader holds with Waraqah. The balance is always the sum of the
// entries, never a stored number. Checkout spends from it; cancelled orders, approved returns,
// refunded sales and Sell Back pay into it.
package wallet

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

// Reasons, named as WalletReason in the app.
const (
	CancelRefund = "cancelRefund"
	ReturnRefund = "returnRefund"
	SaleRefund   = "saleRefund"
	SellBack     = "sellBack"
	Spent        = "spent"
)

// Ledger is wallet.Ledger (BACKEND_PLAN.md section 8). Every method takes the queries to use, so
// an order can change the wallet inside its own transaction; nil means the plain connection.
type Ledger interface {
	Balance(ctx context.Context, q *sqlc.Queries, userID string) (int, error)
	// Credit puts money in (a refund, a Sell Back). Zero or less changes nothing.
	Credit(ctx context.Context, q *sqlc.Queries, userID string, amountBdt int, reason, orderNumber, note string) error
	// Spend takes up to wanted taka for an order and answers how much it really took.
	Spend(ctx context.Context, q *sqlc.Queries, userID, orderNumber string, wanted int) (int, error)
}

// Entry is WalletEntryModel.
type Entry struct {
	AmountBdt   int       `json:"amountBdt"`
	Reason      string    `json:"reason"`
	At          time.Time `json:"at"`
	OrderNumber *string   `json:"orderNumber"`
	Note        *string   `json:"note"`
}

// Wallet is WalletModel: the balance and its history, newest first.
type Wallet struct {
	BalanceBdt int     `json:"balanceBdt"`
	Entries    []Entry `json:"entries"`
}

// Service implements Ledger and the wallet endpoint.
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

func text(s string) pgtype.Text { return pgtype.Text{String: s, Valid: s != ""} }

// Balance is the sum of the entries.
func (s *Service) Balance(ctx context.Context, q *sqlc.Queries, userID string) (int, error) {
	n, err := s.queries(q).WalletBalance(ctx, userID)
	return int(n), err
}

// Credit puts money in.
func (s *Service) Credit(ctx context.Context, q *sqlc.Queries, userID string, amountBdt int, reason, orderNumber, note string) error {
	if amountBdt <= 0 {
		return nil
	}
	return s.queries(q).AddWalletEntry(ctx, sqlc.AddWalletEntryParams{UserID: userID, AmountBdt: int32(amountBdt), Reason: reason,
		OrderNumber: text(orderNumber), Note: text(note), At: s.Clock.Now()})
}

// Spend takes up to wanted taka, never more than the balance.
func (s *Service) Spend(ctx context.Context, q *sqlc.Queries, userID, orderNumber string, wanted int) (int, error) {
	bal, err := s.Balance(ctx, q, userID)
	if err != nil {
		return 0, err
	}
	amount := min(max(wanted, 0), bal)
	if amount > 0 {
		err = s.queries(q).AddWalletEntry(ctx, sqlc.AddWalletEntryParams{UserID: userID, AmountBdt: int32(-amount), Reason: Spent,
			OrderNumber: text(orderNumber), At: s.Clock.Now()})
	}
	return amount, err
}

// Get returns the wallet of a reader.
func (s *Service) Get(ctx context.Context, userID string) (Wallet, error) {
	rows, err := s.DB.Q().ListWalletEntries(ctx, userID)
	if err != nil {
		return Wallet{}, err
	}
	out := Wallet{Entries: []Entry{}}
	for _, r := range rows {
		e := Entry{AmountBdt: int(r.AmountBdt), Reason: r.Reason, At: r.At.In(s.Loc).Truncate(time.Second)}
		if r.OrderNumber.Valid {
			e.OrderNumber = &r.OrderNumber.String
		}
		if r.Note.Valid {
			e.Note = &r.Note.String
		}
		out.BalanceBdt += e.AmountBdt
		out.Entries = append(out.Entries, e)
	}
	return out, nil
}

// Handler is the HTTP side of GET /wallet.
type Handler struct{ S *Service }

// Wallet is WalletFakeApi.wallet. A guest gets the empty wallet.
func (h Handler) Wallet(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFrom(r.Context())
	if !ok {
		httpx.JSON(w, Wallet{Entries: []Entry{}})
		return
	}
	out, err := h.S.Get(r.Context(), u.ID)
	if err != nil {
		h.S.Log.Error("wallet failed", "error", err)
		httpx.Error(w, r, http.StatusInternalServerError, httpx.CodeInternal, "something went wrong")
		return
	}
	httpx.JSON(w, out)
}

// Routes registers GET /wallet.
func (h Handler) Routes(r httpx.Router, a *auth.Middleware) {
	r.Handle("GET /wallet", a.Me(http.HandlerFunc(h.Wallet))) // WalletFakeApi.wallet
}
