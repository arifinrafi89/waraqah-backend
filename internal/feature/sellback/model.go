package sellback

import (
	"time"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
)

// Book is SellBackBookModel: a catalog Book Waraqah will buy back, with its new price.
type Book struct {
	BookID      string `json:"bookId"`
	Title       string `json:"title"`
	Author      string `json:"author"`
	NewPriceBdt int    `json:"newPriceBdt"`
	CoverSeed   int    `json:"coverSeed"`
}

// SellBack is SellBackModel.
type SellBack struct {
	ID              string    `json:"id"`
	Book            Book      `json:"book"`
	Condition       string    `json:"condition"`
	QuoteBdt        int       `json:"quoteBdt"`
	Status          string    `json:"status"`
	PickupAddress   string    `json:"pickupAddress"`
	CreatedAt       time.Time `json:"createdAt"`
	Flags           int       `json:"flags"`
	GradedCondition *string   `json:"gradedCondition"`
	PaidBdt         *int      `json:"paidBdt"`
	// ReaderName is who is selling, for staff only.
	ReaderName *string `json:"readerName"`
}

func toSellBack(r sqlc.SellBack, readerName *string, loc *time.Location) SellBack {
	out := SellBack{ID: r.ID, Book: Book{BookID: r.BookID, Title: r.Title, Author: r.Author, NewPriceBdt: int(r.NewPriceBdt), CoverSeed: int(r.CoverSeed)},
		Condition: r.Condition, QuoteBdt: int(r.QuoteBdt), Status: r.Status, PickupAddress: r.PickupAddress,
		CreatedAt: r.CreatedAt.In(loc).Truncate(time.Second), Flags: int(r.Flags), ReaderName: readerName}
	if r.GradedCondition.Valid {
		out.GradedCondition = &r.GradedCondition.String
	}
	if r.PaidBdt.Valid {
		v := int(r.PaidBdt.Int32)
		out.PaidBdt = &v
	}
	return out
}
