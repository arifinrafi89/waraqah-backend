package moderation

import (
	"context"

	"github.com/arifinrafi89/waraqah-backend/internal/feature/notifications"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
)

// Act follows ModerationFakeReports.act: a moderator dismisses a report, removes the thing
// reported, warns its owner or bans them. Every open report about the same thing ends with it.
// A warning adds a strike and the third bans the account. It answers the open reports left.
// Refused when the report is not open, or when there is nobody to warn or ban.
func (s *Service) Act(ctx context.Context, staffID, reportID, action string) ([]Report, error) {
	switch action {
	case ActDismiss, ActRemove, ActWarn, ActBan:
	default:
		return nil, ErrActionBad
	}
	var (
		destroy []string
		notify  func(context.Context) error
	)
	err := s.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		report, err := q.GetOpenReport(ctx, reportID)
		if isNoRows(err) {
			return ErrNotOpen
		}
		if err != nil {
			return err
		}
		subject, err := s.find(ctx, report.Kind, report.TargetID)
		if err != nil {
			return err
		}
		owner := subject.OwnerID
		if (action == ActWarn || action == ActBan) && owner == "" {
			return ErrNoOwner
		}
		staff, err := q.StaffName(ctx, staffID)
		if err != nil {
			return err
		}
		why := report.Reason
		label := subject.Preview
		if report.Kind != "listing" && report.Kind != "user" {
			label = subject.OwnerName + ": “" + subject.Preview + "”"
		}
		status := "dismissed"
		switch action {
		case ActDismiss:
			err = s.record(ctx, q, staff, AuditDismissed, label, why)
		case ActRemove:
			status = "removed"
			if report.Kind == "listing" {
				destroy, err = s.Shop.TakeDown(ctx, q, report.TargetID, "Removed after a report: "+why)
			} else if sub := s.subject(report.Kind); sub != nil {
				err = sub.Remove(ctx, q, report.TargetID)
			}
			if err == nil {
				err = s.record(ctx, q, staff, AuditRemoved, label, why)
			}
		case ActWarn:
			status = "warned"
			var strikes int
			var banned bool
			strikes, banned, err = s.warn(ctx, q, owner)
			if err != nil {
				return err
			}
			if err = s.record(ctx, q, staff, AuditWarned, subject.OwnerName, why); err == nil && banned {
				status = "banned"
				err = s.record(ctx, q, staff, AuditBanned, subject.OwnerName, "strikes")
			}
			notify = func(ctx context.Context) error {
				return notifications.Warned(ctx, s.Notify, owner, strikes, MaxStrikes)
			}
		case ActBan:
			status = "banned"
			err = q.SetStrikes(ctx, sqlc.SetStrikesParams{ID: owner, Strikes: MaxStrikes, Banned: true})
			if err == nil {
				err = s.record(ctx, q, staff, AuditBanned, subject.OwnerName, why)
			}
			notify = func(ctx context.Context) error { return notifications.BannedNotice(ctx, s.Notify, owner) }
		}
		if err != nil {
			return err
		}
		return q.CloseReportsAbout(ctx, sqlc.CloseReportsAboutParams{Kind: report.Kind, TargetID: report.TargetID, Status: status})
	})
	if err != nil {
		return nil, err
	}
	s.Shop.Destroy(ctx, destroy)
	if notify != nil {
		if err := notify(ctx); err != nil {
			s.Log.Error("moderation notification failed", "error", err)
		}
	}
	return s.OpenReports(ctx)
}

// warn adds a strike; the last one bans. It answers the new count.
func (s *Service) warn(ctx context.Context, q *sqlc.Queries, ownerID string) (int, bool, error) {
	cur, _, err := s.strikesOf(ctx, q, ownerID)
	if err != nil {
		return 0, false, err
	}
	next, banned := AfterWarning(cur)
	return next, banned, q.SetStrikes(ctx, sqlc.SetStrikesParams{ID: ownerID, Strikes: int32(next), Banned: banned})
}
