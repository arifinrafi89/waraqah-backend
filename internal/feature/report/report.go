// Package report takes reports of listings, readers, messages, Bites, comments and reviews, and
// keeps the list of readers each reader blocked.
package report

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/arifinrafi89/waraqah-backend/internal/feature/p2p"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/clock"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/ids"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/textutil"
)

// Refusal is a rule the request broke; the handler answers 200 null with its code.
type Refusal string

func (r Refusal) Error() string { return string(r) }

// Refusal codes (strings live in httpx/errors.go, documented in docs/error-codes.md).
const (
	ErrInvalid       = Refusal(httpx.ErrReportInvalid)
	ErrTargetUnknown = Refusal(httpx.ErrReportTargetUnknown)
	ErrBlockInvalid  = Refusal(httpx.ErrBlockInvalid)
)

// Kinds, reasons and statuses (ReportTargetKind, ReportReason, ReportStatus).
var (
	Kinds   = []string{"listing", "user", "message", "bite", "comment", "review"}
	Reasons = []string{"spam", "fake", "photocopy", "harassment", "offensive", "other"}
)

// MaxNote is the longest note of a report.
const MaxNote = 500

// Check is ReportRules.check: "something else" needs a few words to act on, and notes stay short.
func Check(reason, note string) bool {
	n := strings.TrimSpace(note)
	if textutil.Len(n) > MaxNote {
		return false
	}
	return reason != "other" || n != ""
}

// Report is ContentReportModel: what the reader gets back. The reporter is never sent.
type Report struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`
	TargetID  string    `json:"targetId"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"createdAt"`
	Status    string    `json:"status"`
	Note      *string   `json:"note"`
}

// Blocked is BlockedReaderModel.
type Blocked struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	BlockedAt time.Time `json:"blockedAt"`
}

// Marketplace is the part of the marketplace reports and blocks need.
type Marketplace interface {
	SubjectOf(ctx context.Context, listingID string) (*p2p.Subject, error)
	ReaderOf(ctx context.Context, id string) (*p2p.Reader, error)
}

// Service holds the report and block rules. It is also blocks.Checker.
type Service struct {
	DB    *db.DB
	Shop  Marketplace
	Clock clock.Clock
	Loc   *time.Location
	Log   *slog.Logger
}

func isNoRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

func (s *Service) toReport(r sqlc.Report) Report {
	out := Report{ID: r.ID, Kind: r.Kind, TargetID: r.TargetID, Reason: r.Reason, CreatedAt: r.CreatedAt.In(s.Loc).Truncate(time.Second), Status: r.Status}
	if r.Note.Valid {
		out.Note = &r.Note.String
	}
	return out
}

// canReport: a listing that exists and is not the own one, a reader that exists and is not the
// reader themselves; for the other kinds, any id (their features check it when a moderator acts).
func (s *Service) canReport(ctx context.Context, userID, kind, id string) (bool, error) {
	switch kind {
	case "listing":
		l, err := s.Shop.SubjectOf(ctx, id)
		return l != nil && l.SellerID != userID, err
	case "user":
		r, err := s.Shop.ReaderOf(ctx, id)
		return r != nil && id != userID, err
	}
	return id != "", nil
}

// Send takes a report. The earlier open report of the same reader on the same thing is returned
// instead of a second one. Refused when it breaks ReportRules or the thing cannot be reported.
func (s *Service) Send(ctx context.Context, userID, kind, targetID, reason, note string) (*Report, error) {
	if !contains(Kinds, kind) || !contains(Reasons, reason) || !Check(reason, note) {
		return nil, ErrInvalid
	}
	if ok, err := s.canReport(ctx, userID, kind, targetID); err != nil {
		return nil, err
	} else if !ok {
		return nil, ErrTargetUnknown
	}
	note = strings.TrimSpace(note)
	q := s.DB.Q()
	if earlier, err := q.FindOpenReport(ctx, sqlc.FindOpenReportParams{ReporterID: userID, Kind: kind, TargetID: targetID}); err == nil {
		r := s.toReport(earlier)
		return &r, nil
	} else if !isNoRows(err) {
		return nil, err
	}
	id, now := ids.Report(), s.Clock.Now()
	err := q.InsertReport(ctx, sqlc.InsertReportParams{ID: id, Kind: kind, TargetID: targetID, Reason: reason,
		Note: pgtype.Text{String: note, Valid: note != ""}, ReporterID: userID, CreatedAt: now})
	if err != nil {
		return nil, err
	}
	r := Report{ID: id, Kind: kind, TargetID: targetID, Reason: reason, CreatedAt: now.In(s.Loc).Truncate(time.Second), Status: "open"}
	if note != "" {
		r.Note = &note
	}
	return &r, nil
}

// Blocked lists the readers a reader blocked, newest first.
func (s *Service) Blocked(ctx context.Context, userID string) ([]Blocked, error) {
	out := []Blocked{}
	if userID == "" {
		return out, nil
	}
	rows, err := s.DB.Q().ListBlocks(ctx, userID)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		out = append(out, Blocked{ID: r.BlockedID, Name: r.Name, BlockedAt: r.At.In(s.Loc).Truncate(time.Second)})
	}
	return out, nil
}

// Block adds a reader to the blocked list and answers the list. Refused for the reader
// themselves and for someone unknown.
func (s *Service) Block(ctx context.Context, userID, readerID string) ([]Blocked, error) {
	if readerID == userID {
		return nil, ErrBlockInvalid
	}
	if r, err := s.Shop.ReaderOf(ctx, readerID); err != nil {
		return nil, err
	} else if r == nil {
		return nil, ErrBlockInvalid
	}
	if err := s.DB.Q().AddBlock(ctx, sqlc.AddBlockParams{UserID: userID, BlockedID: readerID, At: s.Clock.Now()}); err != nil {
		return nil, err
	}
	return s.Blocked(ctx, userID)
}

// Unblock removes a reader from the blocked list and answers the list.
func (s *Service) Unblock(ctx context.Context, userID, readerID string) ([]Blocked, error) {
	if err := s.DB.Q().RemoveBlock(ctx, sqlc.RemoveBlockParams{UserID: userID, BlockedID: readerID}); err != nil {
		return nil, err
	}
	return s.Blocked(ctx, userID)
}

// IsBlocked is blocks.Checker: whether either of the two readers blocked the other.
func (s *Service) IsBlocked(ctx context.Context, viewer, other string) (bool, error) {
	if viewer == "" || other == "" {
		return false, nil
	}
	return s.DB.Q().IsBlockedEither(ctx, sqlc.IsBlockedEitherParams{UserID: viewer, BlockedID: other})
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
