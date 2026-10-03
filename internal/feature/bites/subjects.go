package bites

import (
	"context"

	"github.com/arifinrafi89/waraqah-backend/internal/feature/moderation"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
)

// Bites is the moderation subject of kind "bite": a reported Bite (BiteFakeStore.remove).
type Bites struct{ S *Service }

// Comments is the moderation subject of kind "comment": a reported comment, with its replies.
type Comments struct{ S *Service }

var (
	_ moderation.Subject = Bites{}
	_ moderation.Subject = Comments{}
)

func (s *Service) found(ctx context.Context, text, authorID string) *moderation.Found {
	name := "?"
	if people, err := s.DB.Q().ListUserNames(ctx, []string{authorID}); err == nil && len(people) > 0 {
		name = people[0].Name
	}
	return &moderation.Found{Preview: text, OwnerID: authorID, OwnerName: name}
}

// Find looks a Bite up for a report.
func (b Bites) Find(ctx context.Context, id string) (*moderation.Found, error) {
	r, err := b.S.DB.Q().FindBite(ctx, id)
	if isNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return b.S.found(ctx, r.Text, r.AuthorID), nil
}

// Remove deletes the Bite with its likes and comments, in the moderator's transaction.
func (b Bites) Remove(ctx context.Context, q *sqlc.Queries, id string) error {
	return q.DeleteBite(ctx, id)
}

// Find looks a comment up for a report.
func (c Comments) Find(ctx context.Context, id string) (*moderation.Found, error) {
	r, err := c.S.DB.Q().FindComment(ctx, id)
	if isNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return c.S.found(ctx, r.Text, r.AuthorID), nil
}

// Remove deletes the comment and, for a top comment, its replies.
func (c Comments) Remove(ctx context.Context, q *sqlc.Queries, id string) error {
	return q.DeleteComment(ctx, id)
}
