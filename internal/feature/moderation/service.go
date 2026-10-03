package moderation

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/arifinrafi89/waraqah-backend/internal/feature/notifications"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/p2p"
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
	ErrNotWaiting     = Refusal(httpx.ErrListingNotWaiting)
	ErrDecisionBad    = Refusal(httpx.ErrDecisionInvalid)
	ErrDecisionReason = Refusal(httpx.ErrDecisionReason)
	ErrNotOpen        = Refusal(httpx.ErrReportNotOpen)
	ErrNoOwner        = Refusal(httpx.ErrReportNoOwner)
	ErrActionBad      = Refusal(httpx.ErrActionInvalid)
)

// Marketplace is the part of the marketplace the Moderation Center changes.
type Marketplace interface {
	Queue(ctx context.Context) ([]p2p.Queued, error)
	Moderate(ctx context.Context, q *sqlc.Queries, id, status, reason string) (*p2p.Listing, error)
	TakeDown(ctx context.Context, q *sqlc.Queries, id, reason string) ([]string, error)
	Destroy(ctx context.Context, publicIDs []string)
	SubjectOf(ctx context.Context, id string) (*p2p.Subject, error)
	ReaderOf(ctx context.Context, id string) (*p2p.Reader, error)
}

// Found is what a report says about the thing reported and whose it is.
type Found struct {
	Preview   string
	OwnerID   string
	OwnerName string
}

// Subject is a kind of content a feature lets moderators look up and remove: messages (inbox),
// Bites and comments (bites) and reviews (reviews) register one each (moderation.Remover).
type Subject interface {
	// Find answers nil for an unknown id.
	Find(ctx context.Context, id string) (*Found, error)
	// Remove deletes the content inside the transaction of the moderator action.
	Remove(ctx context.Context, q *sqlc.Queries, id string) error
}

// Service is the Moderation Center. It is also moderation.Bans.
type Service struct {
	DB     *db.DB
	Shop   Marketplace
	Notify notifications.Sender
	Clock  clock.Clock
	Loc    *time.Location
	Log    *slog.Logger

	mu       sync.RWMutex
	subjects map[string]Subject
}

// Register adds the lookup and removal of a kind of content ("message", "bite", "comment", "review").
func (s *Service) Register(kind string, sub Subject) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.subjects == nil {
		s.subjects = map[string]Subject{}
	}
	s.subjects[kind] = sub
}

func (s *Service) subject(kind string) Subject {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.subjects[kind]
}

func isNoRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

// IsBanned is moderation.Bans.
func (s *Service) IsBanned(ctx context.Context, userID string) (bool, error) {
	r, err := s.DB.Q().GetStrikes(ctx, userID)
	if isNoRows(err) {
		return false, nil
	}
	return r.Banned, err
}

func (s *Service) strikesOf(ctx context.Context, q *sqlc.Queries, userID string) (int, bool, error) {
	if userID == "" {
		return 0, false, nil
	}
	r, err := q.GetStrikes(ctx, userID)
	if isNoRows(err) {
		return 0, false, nil
	}
	return int(r.Strikes), r.Banned, err
}

// QueuedListing is QueuedListingModel.
type QueuedListing struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	SellerID   string   `json:"sellerId"`
	SellerName string   `json:"sellerName"`
	PriceBdt   int      `json:"priceBdt"`
	Condition  string   `json:"condition"`
	Flags      []string `json:"flags"`
	Photos     []string `json:"photos"`
	CoverSeed  int      `json:"coverSeed"`
	// PhotoURLs is contract v1.1 (F7), as on listings.
	PhotoURLs     map[string]string `json:"photoUrls"`
	SellerStrikes int               `json:"sellerStrikes"`
	NewPriceBdt   *int              `json:"newPriceBdt"`
	Note          *string           `json:"note"`
}

// Queue lists the listings waiting for approval, newest first.
func (s *Service) Queue(ctx context.Context) ([]QueuedListing, error) {
	list, err := s.Shop.Queue(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]QueuedListing, 0, len(list))
	for _, e := range list {
		l := e.Listing
		out = append(out, QueuedListing{ID: l.ID, Title: l.Title, SellerID: l.SellerID, SellerName: l.SellerName, PriceBdt: l.PriceBdt,
			Condition: l.Condition, Flags: l.Flags, Photos: l.Photos, PhotoURLs: l.PhotoURLs, CoverSeed: l.CoverSeed, SellerStrikes: e.SellerStrikes,
			NewPriceBdt: l.NewPriceBdt, Note: l.Note})
	}
	return out, nil
}

// record writes one line of the audit log.
func (s *Service) record(ctx context.Context, q *sqlc.Queries, staffName, action, subject, why string) error {
	return q.InsertLog(ctx, sqlc.InsertLogParams{At: s.Clock.Now(), ByName: staffName, Action: action, Subject: subject,
		Reason: pgtype.Text{String: why, Valid: why != ""}})
}

// Record writes a line of the audit log in the name of the staff member staffID, inside the
// caller's transaction (handled sales record settled disputes this way).
func (s *Service) Record(ctx context.Context, q *sqlc.Queries, staffID, action, subject, why string) error {
	name, err := q.StaffName(ctx, staffID)
	if err != nil {
		return err
	}
	return s.record(ctx, q, name, action, subject, why)
}

// Decide approves a listing, asks for changes or rejects it (Staff). Refused when it is not
// waiting or the reason is missing or too long. The seller is told, the audit log records it with
// the name of the staff member of the token, and the queue left is answered.
func (s *Service) Decide(ctx context.Context, staffID, listingID, decision, reason string) ([]QueuedListing, error) {
	status, action := "", ""
	switch decision {
	case Approve:
		status, action = p2p.Live, AuditApproved
	case RequestChanges:
		status, action = p2p.ChangesRequested, AuditChangesRequested
	case Reject:
		status, action = p2p.Rejected, AuditRejected
	default:
		return nil, ErrDecisionBad
	}
	if CheckDecision(decision, reason) != ProblemNone {
		return nil, ErrDecisionReason
	}
	reason = strings.TrimSpace(reason)
	if decision == Approve {
		reason = ""
	}
	var title, sellerID string
	err := s.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		cur, err := q.LockListing(ctx, listingID)
		if isNoRows(err) || (err == nil && cur.Status != p2p.InReview) {
			return ErrNotWaiting
		}
		if err != nil {
			return err
		}
		before, err := s.Shop.Moderate(ctx, q, listingID, status, reason)
		if err != nil || before == nil {
			return err
		}
		title, sellerID = before.Title, before.SellerID
		name, err := q.StaffName(ctx, staffID)
		if err != nil {
			return err
		}
		return s.record(ctx, q, name, action, title, reason)
	})
	if err != nil {
		return nil, err
	}
	if err := notifications.ListingDecided(ctx, s.Notify, sellerID, listingID, title, decision, reason); err != nil {
		s.Log.Error("listing decision notification failed", "error", err)
	}
	return s.Queue(ctx)
}
