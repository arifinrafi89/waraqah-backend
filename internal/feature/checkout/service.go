package checkout

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/arifinrafi89/waraqah-backend/internal/feature/cart"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/loyalty"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/orders"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/profile"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/wallet"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/clock"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/ids"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/textutil"
)

func isNoRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

// Refusal is a rule the request broke; the handler answers 200 null with its code.
type Refusal string

func (r Refusal) Error() string { return string(r) }

// Refusal codes (strings live in httpx/errors.go, documented in docs/error-codes.md).
const (
	ErrCartEmpty       = Refusal(httpx.ErrCartEmpty)
	ErrAddressUnknown  = Refusal(httpx.ErrAddressUnknownOrder)
	ErrGiftNameMissing = Refusal(httpx.ErrGiftNameMissing)
	ErrGiftTooLong     = Refusal(httpx.ErrGiftMessageTooLong)
	ErrPaymentInvalid  = Refusal(httpx.ErrPaymentInvalid)
	ErrCouponTaken     = Refusal(httpx.ErrCouponCodeTaken)
	ErrCouponInvalid   = Refusal(httpx.ErrCouponInvalid)
)

var payments = []string{"bkash", "nagad", "cashOnDelivery", "card"}

// Addresses finds one address of a reader inside the transaction (Profile owns them).
type Addresses interface {
	AddressIn(ctx context.Context, q *sqlc.Queries, userID, id string) (profile.Address, bool, error)
}

// Inventory takes the copies of an order out of the stock and counts the sale.
type Inventory interface {
	Take(ctx context.Context, q *sqlc.Queries, editionID string, quantity int, at time.Time) error
	Invalidate()
}

// Receipt is OrderReceiptModel: what the reader sees right after placing an order.
type Receipt struct {
	Number        string  `json:"number"`
	TotalBdt      int     `json:"totalBdt"`
	ItemCount     int     `json:"itemCount"`
	Payment       string  `json:"payment"`
	NeedsDelivery bool    `json:"needsDelivery"`
	InsideDhaka   bool    `json:"insideDhaka"`
	HasPreorders  bool    `json:"hasPreorders"`
	PointsEarned  int     `json:"pointsEarned"`
	GiftFor       *string `json:"giftFor"`
	WalletUsedBdt int     `json:"walletUsedBdt"`
}

// PlaceInput is the body of /orders/place.
type PlaceInput struct {
	AddressID  string       `json:"addressId"`
	Payment    string       `json:"payment"`
	CouponCode string       `json:"couponCode"`
	UsePoints  bool         `json:"usePoints"`
	UseWallet  bool         `json:"useWallet"`
	Gift       *orders.Gift `json:"gift"`
}

// Service holds the checkout rules.
type Service struct {
	DB        *db.DB
	Cart      *cart.Service
	Addresses Addresses
	Wallet    wallet.Ledger
	Points    loyalty.Ledger
	Stock     Inventory
	Clock     clock.Clock
	Loc       *time.Location
	Log       *slog.Logger
}

// Place turns the cart into an order, all or nothing: it prices the cart the way the app does,
// spends points and the wallet, saves the order with a number from the sequence, takes the copies
// out of the stock, counts the sale and empties the cart.
func (s *Service) Place(ctx context.Context, userID string, in PlaceInput) (*Receipt, error) {
	if !contains(payments, in.Payment) {
		return nil, ErrPaymentInvalid
	}
	if in.Gift != nil {
		if strings.TrimSpace(in.Gift.RecipientName) == "" {
			return nil, ErrGiftNameMissing
		}
		if textutil.Len(in.Gift.Message) > GiftMaxMessage {
			return nil, ErrGiftTooLong
		}
	}
	var receipt *Receipt
	err := s.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		// One checkout at a time per reader, so the balances read here are the ones spent below.
		if _, err := q.LockUser(ctx, userID); err != nil {
			return err
		}
		address, ok, err := s.Addresses.AddressIn(ctx, q, userID, in.AddressID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrAddressUnknown
		}
		c, err := s.Cart.CartIn(ctx, q, userID)
		if err != nil {
			return err
		}
		if len(c.Lines) == 0 {
			return ErrCartEmpty
		}
		coupon, err := s.usableCoupon(ctx, q, in.CouponCode)
		if err != nil {
			return err
		}
		pointsBalance, err := s.Points.Balance(ctx, q, userID)
		if err != nil {
			return err
		}
		walletBalance, err := s.Wallet.Balance(ctx, q, userID)
		if err != nil {
			return err
		}
		inside := address.District == "Dhaka"
		totals := Compute(Input{Lines: totalsLines(c), InsideDhaka: inside, Coupon: coupon, PointsBalance: pointsBalance,
			UsePoints: in.UsePoints, GiftWrap: in.Gift != nil && in.Gift.Wrapped, WalletBalance: walletBalance, UseWallet: in.UseWallet})

		number, err := ids.OrderNumber(ctx, s.DB)
		if err != nil {
			return err
		}
		spentPoints, err := s.Points.Spend(ctx, q, userID, number, totals.PointsDiscountBdt, totals.SubtotalBdt-totals.CouponOnBooksBdt)
		if err != nil {
			return err
		}
		earned, err := s.Points.Earn(ctx, q, userID, number, totals.BooksPaidBdt())
		if err != nil {
			return err
		}
		walletUsed, err := s.Wallet.Spend(ctx, q, userID, number, totals.WalletBdt)
		if err != nil {
			return err
		}
		var gift *orders.Gift
		if totals.NeedsDelivery {
			gift = in.Gift
		}
		now := s.Clock.Now()
		err = orders.Insert(ctx, q, orders.NewOrder{Number: number, UserID: userID, PlacedAt: now, AddressLabel: address.Label,
			AddressLine: address.Line + ", " + address.District, Payment: in.Payment, Lines: orderLines(c), SubtotalBdt: totals.SubtotalBdt,
			DeliveryFeeBdt: totals.DeliveryFeeBdt, DiscountBdt: totals.CouponDiscountBdt, TotalBdt: totals.TotalBdt(),
			NeedsDelivery: totals.NeedsDelivery, PointsUsed: spentPoints, PointsEarned: earned, Gift: gift, GiftWrapBdt: totals.GiftWrapBdt,
			WalletUsedBdt: walletUsed})
		if err != nil {
			return err
		}
		for _, l := range c.Lines {
			if l.Kind == "edition" {
				if err := s.Stock.Take(ctx, q, l.ItemID, l.Quantity, now); err != nil {
					return err
				}
			}
		}
		if err := s.Cart.ClearIn(ctx, q, userID); err != nil {
			return err
		}
		receipt = &Receipt{Number: number, TotalBdt: totals.TotalBdt(), ItemCount: itemCount(c), Payment: in.Payment, NeedsDelivery: totals.NeedsDelivery,
			InsideDhaka: inside, HasPreorders: hasPreorders(c), PointsEarned: earned, WalletUsedBdt: walletUsed}
		if gift != nil {
			name := gift.RecipientName
			receipt.GiftFor = &name
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.Stock.Invalidate()
	return receipt, nil
}

// usableCoupon is a coupon an order may use right now: known and not expired. An expired or
// unknown code simply gives no discount.
func (s *Service) usableCoupon(ctx context.Context, q *sqlc.Queries, code string) (*Coupon, error) {
	if strings.TrimSpace(code) == "" {
		return nil, nil
	}
	c, err := s.FindCoupon(ctx, q, code)
	if err != nil || c == nil || c.Expired(s.Clock.Now()) {
		return nil, err
	}
	t := c.Totals()
	return &t, nil
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func totalsLines(c cart.Cart) []CartLine {
	out := make([]CartLine, len(c.Lines))
	for i, l := range c.Lines {
		out[i] = CartLine{UnitPriceBdt: l.UnitPriceBdt, Quantity: l.Quantity, IsEbook: l.Format != nil && *l.Format == "ebook"}
	}
	return out
}

func orderLines(c cart.Cart) []orders.NewLine {
	out := make([]orders.NewLine, len(c.Lines))
	for i, l := range c.Lines {
		nl := orders.NewLine{BookID: l.BookID, Title: l.Title, Author: l.Author, Quantity: l.Quantity, UnitPriceBdt: l.UnitPriceBdt,
			Format: l.Format, Language: l.Language, CoverSeed: l.CoverSeed}
		if l.Kind == "edition" {
			id := l.ItemID
			nl.EditionID = &id
		}
		out[i] = nl
	}
	return out
}

func itemCount(c cart.Cart) int {
	n := 0
	for _, l := range c.Lines {
		n += l.Quantity
	}
	return n
}

func hasPreorders(c cart.Cart) bool {
	for _, l := range c.Lines {
		if l.IsPreorder {
			return true
		}
	}
	return false
}
