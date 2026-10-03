package sellback

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/arifinrafi89/waraqah-backend/internal/feature/notifications"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/wallet"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/textutil"
)

// Draft is SellBackDraft: what the reader sends to get a book picked up.
type Draft struct {
	BookID        string `json:"bookId"`
	Condition     string `json:"condition"`
	Flags         int    `json:"flags"`
	PickupAddress string `json:"pickupAddress"`
}

// Create books a pickup at the instant quote (SellBackFakeStore.create). The book must be one
// Waraqah buys back and the address at least five characters.
func (s *Service) Create(ctx context.Context, userID string, d Draft) (*SellBack, error) {
	book, err := s.FindBook(ctx, d.BookID)
	if err != nil {
		return nil, err
	}
	if book == nil {
		return nil, ErrBookUnknown
	}
	address := strings.TrimSpace(d.PickupAddress)
	if !validCondition(d.Condition) || d.Flags < 0 || textutil.Len(address) < MinAddress {
		return nil, ErrInvalid
	}
	n, err := s.DB.Q().NextSellBackNumber(ctx)
	if err != nil {
		return nil, err
	}
	id := fmt.Sprintf("SB-%d", n)
	err = s.DB.Q().InsertSellBack(ctx, sqlc.InsertSellBackParams{ID: id, UserID: userID, BookID: book.BookID, Title: book.Title,
		Author: book.Author, NewPriceBdt: int32(book.NewPriceBdt), CoverSeed: int32(book.CoverSeed), Condition: d.Condition,
		Flags: int32(d.Flags), QuoteBdt: int32(Quote(book.NewPriceBdt, d.Condition, d.Flags)), PickupAddress: d.PickupAddress,
		CreatedAt: s.Clock.Now()})
	if err != nil {
		return nil, err
	}
	rows, err := s.DB.Q().ListSellBacksOf(ctx, userID)
	if err != nil {
		return nil, err
	}
	var out *SellBack
	for _, r := range rows {
		if r.ID == id {
			v := toSellBack(r, nil, s.Loc)
			out = &v
		}
	}
	s.schedulePickUp()
	return out, nil
}

// Grade is staff grading a picked-up book (SellBackFakeStore.grade). Accepted, Waraqah pays the
// quote for the graded condition into the reader's wallet and puts the copy on sale as Certified
// Used; refused, the book is sent back. The reader is told either way. Answers the queue left.
func (s *Service) Grade(ctx context.Context, id, condition string, accept bool) ([]SellBack, error) {
	if !validCondition(condition) {
		return nil, ErrInvalid
	}
	var notify func()
	err := s.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		r, err := q.LockSellBack(ctx, id)
		if isNoRows(err) || (err == nil && r.Status != PickedUp) {
			return ErrNotWaiting
		}
		if err != nil {
			return err
		}
		if !accept {
			notify = func() {
				s.warn("returned", notifications.SellBackReturnedNotice(ctx, s.Notify, r.UserID, r.ID, r.Title))
			}
			return q.GradeSellBack(ctx, sqlc.GradeSellBackParams{ID: r.ID, Status: Returned})
		}
		paid := Quote(int(r.NewPriceBdt), condition, int(r.Flags))
		err = q.GradeSellBack(ctx, sqlc.GradeSellBackParams{ID: r.ID, Status: PaidOut,
			GradedCondition: pgtype.Text{String: condition, Valid: true}, PaidBdt: pgtype.Int4{Int32: int32(paid), Valid: true}})
		if err != nil {
			return err
		}
		if err := s.Wallet.Credit(ctx, q, r.UserID, paid, wallet.SellBack, "", r.Title); err != nil {
			return err
		}
		if err := s.publish(ctx, q, r.BookID, ResellPrice(int(r.NewPriceBdt), condition), condition); err != nil {
			return err
		}
		notify = func() { s.warn("paid", notifications.SellBackPaidNotice(ctx, s.Notify, r.UserID, r.ID, r.Title, paid)) }
		return nil
	})
	if err != nil {
		return nil, err
	}
	notify()
	return s.Queue(ctx)
}
