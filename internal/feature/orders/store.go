package orders

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/arifinrafi89/waraqah-backend/internal/feature/cart"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/loyalty"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/notifications"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/wallet"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/clock"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
)

// Refusal is a rule the request broke; the handler answers 200 null with its code.
type Refusal string

func (r Refusal) Error() string { return string(r) }

// Refusal codes (strings live in httpx/errors.go, documented in docs/error-codes.md).
const (
	ErrOrderUnknown     = Refusal(httpx.ErrOrderUnknown)
	ErrOrderNotCancel   = Refusal(httpx.ErrOrderNotCancellable)
	ErrReturnNotAllowed = Refusal(httpx.ErrReturnNotAllowed)
	ErrReturnReason     = Refusal(httpx.ErrReturnReasonInvalid)
	ErrStepUnavailable  = Refusal(httpx.ErrOrderStepUnavailable)
	ErrReturnNotWaiting = Refusal(httpx.ErrReturnNotWaiting)
	ErrPhotoInvalid     = Refusal(httpx.ErrPhotoInvalid)
)

// CartAdder is the part of the cart that "buy again" needs.
type CartAdder interface {
	Add(ctx context.Context, userID, kind, itemID string) (cart.Cart, string, error)
}

// Inventory gives stock back when an order is cancelled.
type Inventory interface {
	Restock(ctx context.Context, q *sqlc.Queries, editionID string, quantity int) error
}

// Service holds the order rules.
type Service struct {
	DB            *db.DB
	Wallet        wallet.Ledger
	Points        loyalty.Ledger
	Notify        notifications.Sender
	Cart          CartAdder
	Stock         Inventory
	Clock         clock.Clock
	Loc           *time.Location
	MaxImageBytes int
	Log           *slog.Logger
}

func (s *Service) t(v time.Time) time.Time { return v.In(s.Loc).Truncate(time.Second) }

// build assembles orders (lines, history, return) from stored rows.
func (s *Service) build(ctx context.Context, q *sqlc.Queries, rows []sqlc.Order) ([]Order, error) {
	numbers := make([]string, len(rows))
	for i, r := range rows {
		numbers[i] = r.Number
	}
	lines, err := q.ListLinesOf(ctx, numbers)
	if err != nil {
		return nil, err
	}
	history, err := q.ListHistoryOf(ctx, numbers)
	if err != nil {
		return nil, err
	}
	returns, err := q.ListReturnsOf(ctx, numbers)
	if err != nil {
		return nil, err
	}
	byLines := map[string][]Line{}
	for _, l := range lines {
		byLines[l.OrderNumber] = append(byLines[l.OrderNumber], Line{BookID: l.BookID, Title: l.Title, Author: l.Author,
			Quantity: int(l.Quantity), UnitPriceBdt: int(l.UnitPriceBdt), Format: textPtr(l.Format), Language: textPtr(l.Language),
			CoverSeed: int(l.CoverSeed), EditionID: textPtr(l.EditionID)})
	}
	byHistory := map[string][]Change{}
	for _, h := range history {
		byHistory[h.OrderNumber] = append(byHistory[h.OrderNumber], Change{Status: h.Status, At: s.t(h.At)})
	}
	byReturn := map[string]*Return{}
	for _, r := range returns {
		photos := []string{}
		_ = json.Unmarshal(r.Photos, &photos)
		byReturn[r.OrderNumber] = &Return{Reason: r.Reason, Status: r.Status, RequestedAt: s.t(r.RequestedAt), Note: r.Note, Photos: photos}
	}
	out := make([]Order, 0, len(rows))
	for _, r := range rows {
		o := Order{Number: r.Number, PlacedAt: s.t(r.PlacedAt), Status: r.Status, Lines: byLines[r.Number], History: byHistory[r.Number],
			AddressLabel: r.AddressLabel, AddressLine: r.AddressLine, Payment: r.Payment, SubtotalBdt: int(r.SubtotalBdt),
			DeliveryFeeBdt: int(r.DeliveryFeeBdt), DiscountBdt: int(r.DiscountBdt), TotalBdt: int(r.TotalBdt), NeedsDelivery: r.NeedsDelivery,
			PointsUsed: int(r.PointsUsed), PointsEarned: int(r.PointsEarned), ReturnRequest: byReturn[r.Number], GiftWrapBdt: int(r.GiftWrapBdt),
			IsDonation: r.IsDonation, WalletUsedBdt: int(r.WalletUsedBdt), RefundedBdt: int(r.RefundedBdt), UserID: r.UserID}
		if o.Lines == nil {
			o.Lines = []Line{}
		}
		if o.History == nil {
			o.History = []Change{}
		}
		if len(r.Gift) > 0 {
			var g Gift
			if json.Unmarshal(r.Gift, &g) == nil {
				o.Gift = &g
			}
		}
		out = append(out, o)
	}
	return out, nil
}

func textPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}

func optText(p *string) pgtype.Text {
	if p == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *p, Valid: true}
}

// one loads a single order (locked for the transaction), or false.
func (s *Service) one(ctx context.Context, q *sqlc.Queries, number string) (Order, bool, error) {
	row, err := q.GetOrder(ctx, number)
	if err != nil {
		if isNoRows(err) {
			return Order{}, false, nil
		}
		return Order{}, false, err
	}
	list, err := s.build(ctx, q, []sqlc.Order{row})
	if err != nil || len(list) == 0 {
		return Order{}, false, err
	}
	return list[0], true, nil
}

// ListOf returns the orders of a reader, newest first.
func (s *Service) ListOf(ctx context.Context, userID string) ([]Order, error) {
	rows, err := s.DB.Q().ListOrdersOf(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.build(ctx, s.DB.Q(), rows)
}

// ListAll returns every order, newest first (Staff).
func (s *Service) ListAll(ctx context.Context) ([]Order, error) {
	rows, err := s.DB.Q().ListAllOrders(ctx)
	if err != nil {
		return nil, err
	}
	return s.build(ctx, s.DB.Q(), rows)
}

// Details returns one order of the reader, or nil.
func (s *Service) Details(ctx context.Context, userID, number string) (*Order, error) {
	o, ok, err := s.one(ctx, s.DB.Q(), number)
	if err != nil || !ok || o.UserID != userID {
		return nil, err
	}
	return &o, nil
}

// NewLine is one line of an order being saved.
type NewLine struct {
	BookID       string
	EditionID    *string
	Title        string
	Author       string
	Quantity     int
	UnitPriceBdt int
	Format       *string
	Language     *string
	CoverSeed    int
}

// NewOrder is an order being saved (checkout and donate build these).
type NewOrder struct {
	Number         string
	UserID         string
	PlacedAt       time.Time
	Status         string // default placed
	History        []Change
	AddressLabel   string
	AddressLine    string
	Payment        string
	Lines          []NewLine
	SubtotalBdt    int
	DeliveryFeeBdt int
	DiscountBdt    int
	TotalBdt       int
	NeedsDelivery  bool
	PointsUsed     int
	PointsEarned   int
	Gift           *Gift
	GiftWrapBdt    int
	IsDonation     bool
	DonatePlaceID  string
	WalletUsedBdt  int
}

// Insert saves a new order with its lines and history inside the transaction q.
func Insert(ctx context.Context, q *sqlc.Queries, n NewOrder) error {
	status := n.Status
	if status == "" {
		status = Placed
	}
	history := n.History
	if len(history) == 0 {
		history = []Change{{Status: Placed, At: n.PlacedAt}}
	}
	var gift []byte
	if n.Gift != nil {
		gift, _ = json.Marshal(n.Gift)
	}
	err := q.InsertOrder(ctx, sqlc.InsertOrderParams{Number: n.Number, UserID: n.UserID, Status: status, PlacedAt: n.PlacedAt,
		AddressLabel: n.AddressLabel, AddressLine: n.AddressLine, Payment: n.Payment, SubtotalBdt: int32(n.SubtotalBdt),
		DeliveryFeeBdt: int32(n.DeliveryFeeBdt), DiscountBdt: int32(n.DiscountBdt), TotalBdt: int32(n.TotalBdt), NeedsDelivery: n.NeedsDelivery,
		PointsUsed: int32(n.PointsUsed), PointsEarned: int32(n.PointsEarned), Gift: gift, GiftWrapBdt: int32(n.GiftWrapBdt),
		IsDonation: n.IsDonation, DonatePlaceID: pgtype.Text{String: n.DonatePlaceID, Valid: n.DonatePlaceID != ""},
		WalletUsedBdt: int32(n.WalletUsedBdt), RefundedBdt: 0})
	if err != nil {
		return err
	}
	for i, l := range n.Lines {
		err := q.InsertOrderLine(ctx, sqlc.InsertOrderLineParams{OrderNumber: n.Number, Position: int32(i + 1), BookID: l.BookID,
			EditionID: optText(l.EditionID), Title: l.Title, Author: l.Author, Quantity: int32(l.Quantity), UnitPriceBdt: int32(l.UnitPriceBdt),
			Format: optText(l.Format), Language: optText(l.Language), CoverSeed: int32(l.CoverSeed)})
		if err != nil {
			return err
		}
	}
	for _, h := range history {
		if err := q.InsertOrderHistory(ctx, sqlc.InsertOrderHistoryParams{OrderNumber: n.Number, Status: h.Status, At: h.At}); err != nil {
			return err
		}
	}
	return nil
}

// Numbers are the order figures of the admin dashboard.
type Numbers struct {
	OrdersToday, SalesTodayBdt, ToShip, ReturnsWaiting int
}

// DashboardNumbers counts the orders placed in [from, to) that were not cancelled and their total,
// the orders still to ship, and the returns waiting for a decision.
func (s *Service) DashboardNumbers(ctx context.Context, from, to time.Time) (Numbers, error) {
	r, err := s.DB.Q().DashboardOrderNumbers(ctx, sqlc.DashboardOrderNumbersParams{DayStart: from, DayEnd: to})
	return Numbers{OrdersToday: int(r.OrdersToday), SalesTodayBdt: int(r.SalesTodayBdt), ToShip: int(r.ToShip),
		ReturnsWaiting: int(r.ReturnsWaiting)}, err
}
