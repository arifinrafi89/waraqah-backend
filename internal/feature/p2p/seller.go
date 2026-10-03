package p2p

import (
	"context"
	"time"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
)

// SellerReview is what a buyer wrote about a seller.
type SellerReview struct {
	FromName string    `json:"fromName"`
	Stars    int       `json:"stars"`
	At       time.Time `json:"at"`
	Comment  *string   `json:"comment"`
}

// SellerProfile is SellerProfileModel: a reader seller page.
type SellerProfile struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Area          string         `json:"area"`
	District      string         `json:"district"`
	MemberSince   time.Time      `json:"memberSince"`
	BooksSold     int            `json:"booksSold"`
	RatingCount   int            `json:"ratingCount"`
	RatingAverage *float64       `json:"ratingAverage"`
	Reviews       []SellerReview `json:"reviews"`
	Listings      []Listing      `json:"listings"`
}

// Seller builds the seller page of a reader, or nil for someone the marketplace does not know.
// Books sold are those before the app records plus the listings sold since.
func (s *Service) Seller(ctx context.Context, viewer, id string) (*SellerProfile, error) {
	q := s.DB.Q()
	who, err := q.GetReader(ctx, id)
	if isNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	sold, err := q.CountSoldBy(ctx, id)
	if err != nil {
		return nil, err
	}
	ratings, err := q.ListRatingsTo(ctx, id)
	if err != nil {
		return nil, err
	}
	live, err := q.LiveListingsOf(ctx, id)
	if err != nil {
		return nil, err
	}
	listings, err := s.build(ctx, q, viewer, rowsOf(live, func(r sqlc.LiveListingsOfRow) row { return row{r.Listing, r.SellerName} }))
	if err != nil {
		return nil, err
	}
	p := &SellerProfile{ID: who.ID, Name: who.Name, Area: who.Area, District: who.District, MemberSince: who.MemberSince.In(s.Loc).Truncate(time.Second),
		BooksSold: int(who.SoldBefore) + int(sold), RatingCount: len(ratings), Reviews: make([]SellerReview, 0, len(ratings)), Listings: listings}
	stars := 0
	for _, r := range ratings {
		stars += int(r.Stars)
		p.Reviews = append(p.Reviews, SellerReview{FromName: r.FromName, Stars: int(r.Stars), At: r.At.In(s.Loc).Truncate(time.Second), Comment: text(r.Comment)})
	}
	if len(ratings) > 0 {
		avg := float64(stars) / float64(len(ratings))
		p.RatingAverage = &avg
	}
	return p, nil
}
