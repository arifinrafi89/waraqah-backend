package moderation

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
)

// Report is ModerationReportModel: an open report, one per reported thing.
type Report struct {
	ID           string    `json:"id"`
	Kind         string    `json:"kind"`
	TargetID     string    `json:"targetId"`
	Reason       string    `json:"reason"`
	CreatedAt    time.Time `json:"createdAt"`
	Preview      string    `json:"preview"`
	OwnerID      string    `json:"ownerId"`
	OwnerName    string    `json:"ownerName"`
	ReportCount  int       `json:"reportCount"`
	OwnerStrikes int       `json:"ownerStrikes"`
	OwnerBanned  bool      `json:"ownerBanned"`
	Note         *string   `json:"note"`
}

// find looks a reported thing up in the records of its feature.
func (s *Service) find(ctx context.Context, kind, id string) (Found, error) {
	unknown := Found{Preview: id, OwnerName: "?"}
	switch kind {
	case "listing":
		l, err := s.Shop.SubjectOf(ctx, id)
		if err != nil || l == nil {
			return unknown, err
		}
		return Found{Preview: l.Title, OwnerID: l.SellerID, OwnerName: l.SellerName}, nil
	case "user":
		r, err := s.Shop.ReaderOf(ctx, id)
		if err != nil || r == nil {
			return unknown, err
		}
		return Found{Preview: fmt.Sprintf("%s · %s, %s", r.Name, r.Area, r.District), OwnerID: r.ID, OwnerName: r.Name}, nil
	}
	sub := s.subject(kind)
	if sub == nil {
		return unknown, nil
	}
	f, err := sub.Find(ctx, id)
	if err != nil || f == nil {
		return unknown, err
	}
	return *f, nil
}

// OpenReports lists the open reports, oldest first, one per reported thing.
func (s *Service) OpenReports(ctx context.Context) ([]Report, error) {
	q := s.DB.Q()
	rows, err := q.ListOpenReports(ctx)
	if err != nil {
		return nil, err
	}
	var order []string
	first := map[string]sqlc.Report{}
	count := map[string]int{}
	for _, r := range rows {
		key := r.Kind + ":" + r.TargetID
		if _, ok := first[key]; !ok {
			first[key] = r
			order = append(order, key)
		}
		count[key]++
	}
	out := make([]Report, 0, len(order))
	for _, key := range order {
		r := first[key]
		f, err := s.find(ctx, r.Kind, r.TargetID)
		if err != nil {
			return nil, err
		}
		strikes, banned, err := s.strikesOf(ctx, q, f.OwnerID)
		if err != nil {
			return nil, err
		}
		item := Report{ID: r.ID, Kind: r.Kind, TargetID: r.TargetID, Reason: r.Reason, CreatedAt: r.CreatedAt.In(s.Loc).Truncate(time.Second),
			Preview: f.Preview, OwnerID: f.OwnerID, OwnerName: f.OwnerName, ReportCount: count[key], OwnerStrikes: strikes, OwnerBanned: banned}
		if r.Note.Valid {
			item.Note = &r.Note.String
		}
		out = append(out, item)
	}
	return out, nil
}

// Entry is AuditEntryModel.
type Entry struct {
	ID      string    `json:"id"`
	At      time.Time `json:"at"`
	By      string    `json:"by"`
	Action  string    `json:"action"`
	Subject string    `json:"subject"`
	Reason  *string   `json:"reason"`
}

// AuditLog is the audit log, newest first.
func (s *Service) AuditLog(ctx context.Context) ([]Entry, error) {
	rows, err := s.DB.Q().ListLog(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Entry, 0, len(rows))
	for _, r := range rows {
		e := Entry{ID: "au-" + strconv.FormatInt(r.ID, 10), At: r.At.In(s.Loc).Truncate(time.Second), By: r.ByName, Action: r.Action, Subject: r.Subject}
		if r.Reason.Valid {
			e.Reason = &r.Reason.String
		}
		out = append(out, e)
	}
	return out, nil
}
