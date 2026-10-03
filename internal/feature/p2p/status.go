package p2p

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
)

// Status is listings.Status (BACKEND_PLAN.md section 8): what the inbox and handled sales do to a
// listing. Every method takes the queries of the caller, so it joins the caller transaction.
type Status interface {
	// Find reads a listing for viewer; nil when unknown.
	Find(ctx context.Context, viewer, id string) (*Listing, error)
	// SetStatus reserves or sells for buyerID, or makes it live again (which clears the buyer).
	SetStatus(ctx context.Context, q *sqlc.Queries, id, status, buyerID string) error
}

// Find reads a listing for viewer.
func (s *Service) Find(ctx context.Context, viewer, id string) (*Listing, error) {
	return s.One(ctx, viewer, id)
}

// SetStatus is P2pFakeStore.setStatus: reserved or sold need the buyer; going back to live clears
// it. An unknown listing changes nothing.
func (s *Service) SetStatus(ctx context.Context, q *sqlc.Queries, id, status, buyerID string) error {
	cur, err := q.LockListing(ctx, id)
	if isNoRows(err) {
		return nil
	}
	if err != nil {
		return err
	}
	buyer := cur.BuyerID
	switch {
	case status == Live:
		buyer = pgtype.Text{}
	case buyerID != "":
		buyer = pgtype.Text{String: buyerID, Valid: true}
	}
	return q.SetListingStatus(ctx, sqlc.SetListingStatusParams{ID: id, Status: status, RejectionReason: cur.RejectionReason, BuyerID: buyer})
}

// Queued is a listing waiting for a moderator, with the strikes of its seller.
type Queued struct {
	Listing       Listing
	SellerStrikes int
}

// Queue lists the listings waiting for review, newest first (moderation).
func (s *Service) Queue(ctx context.Context) ([]Queued, error) {
	q := s.DB.Q()
	rows, err := q.ListingQueue(ctx)
	if err != nil {
		return nil, err
	}
	list, err := s.build(ctx, q, "", rowsOf(rows, func(r sqlc.ListingQueueRow) row { return row{r.Listing, r.SellerName} }))
	if err != nil {
		return nil, err
	}
	out := make([]Queued, len(list))
	for i, l := range list {
		out[i] = Queued{Listing: l, SellerStrikes: int(rows[i].SellerStrikes)}
	}
	return out, nil
}

// Moderate is P2pFakeStore.moderate: a moderator decision (live, changesRequested or rejected)
// with the reason the seller sees; an empty reason clears it. It answers the listing as it was
// before, or nil when unknown or not waiting for review.
func (s *Service) Moderate(ctx context.Context, q *sqlc.Queries, id, status, reason string) (*Listing, error) {
	cur, err := q.LockListing(ctx, id)
	if isNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	before, err := s.oneIn(ctx, q, "", id)
	if err != nil || before == nil {
		return nil, err
	}
	err = q.SetListingStatus(ctx, sqlc.SetListingStatusParams{ID: id, Status: status,
		RejectionReason: pgtype.Text{String: reason, Valid: reason != ""}, BuyerID: cur.BuyerID})
	return before, err
}

// TakeDown removes a reported listing: it is rejected with a reason and its photos are deleted.
// It answers the public ids of the photos to destroy once the transaction commits (Destroy).
func (s *Service) TakeDown(ctx context.Context, q *sqlc.Queries, id, reason string) ([]string, error) {
	if _, err := s.Moderate(ctx, q, id, Rejected, reason); err != nil {
		return nil, err
	}
	return q.DeletePhotosOfListing(ctx, id)
}

// Destroy deletes photo assets, best effort.
func (s *Service) Destroy(ctx context.Context, publicIDs []string) { s.destroy(ctx, publicIDs) }

// Subject is what a report says about a listing: its title and its seller.
type Subject struct {
	Title      string
	SellerID   string
	SellerName string
}

// SubjectOf finds a listing for a report or the moderation log; nil when unknown.
func (s *Service) SubjectOf(ctx context.Context, id string) (*Subject, error) {
	r, err := s.DB.Q().GetListing(ctx, id)
	if isNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &Subject{Title: r.Listing.Title, SellerID: r.Listing.SellerID, SellerName: r.SellerName}, nil
}

// Reader is a person of the marketplace.
type Reader struct {
	ID, Name, Area, District string
}

// ReaderOf finds an active account by id; nil when unknown or deleted.
func (s *Service) ReaderOf(ctx context.Context, id string) (*Reader, error) {
	p, err := s.DB.Q().GetPerson(ctx, id)
	if isNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &Reader{ID: p.ID, Name: p.Name, Area: p.Area, District: p.District}, nil
}

// OnAccountDeleted is the profile.DeleteHook: the listings of the reader leave the marketplace
// (the queries hide deleted sellers) and their photos are deleted.
func (s *Service) OnAccountDeleted(ctx context.Context, userID string) error {
	ids, err := s.DB.Q().DeleteAllListingPhotosOf(ctx, userID)
	if err != nil {
		return err
	}
	s.destroy(ctx, ids)
	return nil
}
