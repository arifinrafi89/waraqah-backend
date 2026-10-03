package p2p

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
)

// Ratings reads and writes what readers gave each other after a sale (the inbox uses it).

// RatingsFor answers, per listing, the stars each reader gave for its sale.
func (s *Service) RatingsFor(ctx context.Context, q *sqlc.Queries, listingIDs []string) (map[string]map[string]int, error) {
	rows, err := q.ListRatingsForListings(ctx, listingIDs)
	if err != nil {
		return nil, err
	}
	out := map[string]map[string]int{}
	for _, r := range rows {
		if out[r.ListingID.String] == nil {
			out[r.ListingID.String] = map[string]int{}
		}
		out[r.ListingID.String][r.FromID] = int(r.Stars)
	}
	return out, nil
}

// AddRating saves a rating for the sale of a listing.
func (s *Service) AddRating(ctx context.Context, q *sqlc.Queries, fromID, toID, listingID string, stars int, comment string, at time.Time) error {
	return q.InsertRating(ctx, sqlc.InsertRatingParams{FromID: fromID, ToID: toID, ListingID: pgtype.Text{String: listingID, Valid: listingID != ""},
		Stars: int32(stars), Comment: pgtype.Text{String: comment, Valid: comment != ""}, At: at})
}

// Candidate is a copy a book request may match.
type Candidate struct {
	ID         string
	Title      string
	BookID     string
	SellerID   string
	SellerName string
	Status     string
}

func candidate(id, title string, book pgtype.Text, seller, name, status string) Candidate {
	return Candidate{ID: id, Title: title, BookID: book.String, SellerID: seller, SellerName: name, Status: status}
}

// Candidates lists the copies not sold yet of readers other than excludedSeller (only the live ones
// with onlyLive): what a new book request is matched against.
func (s *Service) Candidates(ctx context.Context, excludedSeller string, onlyLive bool) ([]Candidate, error) {
	rows, err := s.DB.Q().ListingCandidates(ctx, sqlc.ListingCandidatesParams{ExcludedSeller: excludedSeller, OnlyLive: onlyLive})
	if err != nil {
		return nil, err
	}
	out := make([]Candidate, len(rows))
	for i, r := range rows {
		out[i] = candidate(r.ID, r.Title, r.BookID, r.SellerID, r.SellerName, r.Status)
	}
	return out, nil
}

// OfSeller lists the copies of a reader that are not sold yet.
func (s *Service) OfSeller(ctx context.Context, sellerID string) ([]Candidate, error) {
	rows, err := s.DB.Q().ListingsOfSeller(ctx, sellerID)
	if err != nil {
		return nil, err
	}
	out := make([]Candidate, len(rows))
	for i, r := range rows {
		out[i] = candidate(r.ID, r.Title, r.BookID, r.SellerID, r.SellerName, r.Status)
	}
	return out, nil
}
