package bites

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/arifinrafi89/waraqah-backend/internal/feature/notifications"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/ids"
)

// Draft is BiteDraft: a new Bite, or an edit when ID is set.
type Draft struct {
	ID      string  `json:"id"`
	Text    string  `json:"text"`
	BookID  *string `json:"bookId"`
	Spoiler bool    `json:"spoiler"`
}

func (d Draft) bookID() string {
	if d.BookID == nil {
		return ""
	}
	return strings.TrimSpace(*d.BookID)
}

// check is BiteFakeStore._valid: the rules, and a tagged Book the catalog has.
func (s *Service) check(ctx context.Context, d Draft) error {
	if Check(d.Text, d.bookID(), d.Spoiler) != ProblemNone {
		return ErrInvalid
	}
	if id := d.bookID(); id != "" {
		if _, ok, err := s.Books.Find(ctx, id); err != nil || !ok {
			if err == nil {
				err = ErrBookUnknown
			}
			return err
		}
	}
	return nil
}

func (s *Service) notBanned(ctx context.Context, userID string) error {
	banned, err := s.Bans.IsBanned(ctx, userID)
	if err == nil && banned {
		return ErrBanned
	}
	return err
}

// Post is BiteFakeStore.post: a banned reader cannot post.
func (s *Service) Post(ctx context.Context, viewer string, d Draft) (*Bite, error) {
	if err := s.notBanned(ctx, viewer); err != nil {
		return nil, err
	}
	if err := s.check(ctx, d); err != nil {
		return nil, err
	}
	id := ids.Bite()
	err := s.DB.Q().InsertBite(ctx, sqlc.InsertBiteParams{ID: id, AuthorID: viewer, Text: strings.TrimSpace(d.Text),
		BookID: pgtype.Text{String: d.bookID(), Valid: d.bookID() != ""}, Spoiler: d.Spoiler, CreatedAt: s.Clock.Now()})
	if err != nil {
		return nil, err
	}
	return s.One(ctx, viewer, id)
}

// mine finds a Bite of the viewer; others are refused.
func (s *Service) mine(ctx context.Context, viewer, id string) (sqlc.Bite, error) {
	b, err := s.DB.Q().FindBite(ctx, id)
	if isNoRows(err) {
		return b, ErrUnknown
	}
	if err == nil && b.AuthorID != viewer {
		return b, ErrNotYours
	}
	return b, err
}

// Edit is BiteFakeStore.edit: own Bites only. The tag is replaced by the one sent (none clears it).
func (s *Service) Edit(ctx context.Context, viewer string, d Draft) (*Bite, error) {
	if _, err := s.mine(ctx, viewer, d.ID); err != nil {
		return nil, err
	}
	if err := s.check(ctx, d); err != nil {
		return nil, err
	}
	err := s.DB.Q().UpdateBite(ctx, sqlc.UpdateBiteParams{ID: d.ID, Text: strings.TrimSpace(d.Text),
		BookID: pgtype.Text{String: d.bookID(), Valid: d.bookID() != ""}, Spoiler: d.Spoiler,
		EditedAt: pgtype.Timestamptz{Time: s.Clock.Now(), Valid: true}})
	if err != nil {
		return nil, err
	}
	return s.One(ctx, viewer, d.ID)
}

// Delete is BiteFakeStore.delete: own Bites only, with their likes and comments.
func (s *Service) Delete(ctx context.Context, viewer, id string) error {
	if _, err := s.mine(ctx, viewer, id); err != nil {
		return err
	}
	return s.DB.Q().DeleteBite(ctx, id)
}

// visible finds a Bite the viewer may act on: known, and not by a reader blocked either way.
func (s *Service) visible(ctx context.Context, viewer, id string) (sqlc.Bite, error) {
	b, err := s.DB.Q().FindBite(ctx, id)
	if isNoRows(err) {
		return b, ErrUnknown
	}
	if err != nil {
		return b, err
	}
	blocked, err := s.Blocks.IsBlocked(ctx, viewer, b.AuthorID)
	if err == nil && blocked {
		err = ErrBlocked
	}
	return b, err
}

// Like is BiteFakeStore.like: liked true adds the viewer's like, false takes it back.
func (s *Service) Like(ctx context.Context, viewer, id string, liked bool) (*Bite, error) {
	if _, err := s.visible(ctx, viewer, id); err != nil {
		return nil, err
	}
	var err error
	if liked {
		err = s.DB.Q().LikeBite(ctx, sqlc.LikeBiteParams{BiteID: id, UserID: viewer})
	} else {
		err = s.DB.Q().UnlikeBite(ctx, sqlc.UnlikeBiteParams{BiteID: id, UserID: viewer})
	}
	if err != nil {
		return nil, err
	}
	return s.One(ctx, viewer, id)
}

// Comment is BiteComments.comment: a reply to a reply moves up to its top comment. The Bite's
// author hears about a comment and the top comment's author about a reply, never the writer.
func (s *Service) Comment(ctx context.Context, viewer, biteID, text string, parentID *string) (*Detail, error) {
	b, err := s.visible(ctx, viewer, biteID)
	if err != nil {
		return nil, err
	}
	if err := s.notBanned(ctx, viewer); err != nil {
		return nil, err
	}
	if CheckComment(text) != ProblemNone {
		return nil, ErrCommentInvalid
	}
	q := s.DB.Q()
	var top pgtype.Text
	tell := b.AuthorID
	if parentID != nil && *parentID != "" {
		parent, err := q.FindComment(ctx, *parentID)
		if isNoRows(err) || (err == nil && parent.BiteID != biteID) {
			return nil, ErrCommentUnknown
		}
		if err != nil {
			return nil, err
		}
		top = pgtype.Text{String: parent.ID, Valid: true}
		if parent.ParentID.Valid {
			top = parent.ParentID
			if grand, err := q.FindComment(ctx, top.String); err == nil {
				parent = grand
			}
		}
		tell = parent.AuthorID
	}
	err = q.InsertComment(ctx, sqlc.InsertCommentParams{ID: ids.Comment(), BiteID: biteID, ParentID: top, AuthorID: viewer,
		Text: strings.TrimSpace(text), CreatedAt: s.Clock.Now()})
	if err != nil {
		return nil, err
	}
	if tell != viewer {
		s.tell(ctx, viewer, tell, b, top.Valid)
	}
	return s.Detail(ctx, viewer, biteID)
}

func (s *Service) tell(ctx context.Context, writer, to string, b sqlc.Bite, reply bool) {
	name := "?"
	if people, err := s.DB.Q().ListUserNames(ctx, []string{writer}); err == nil && len(people) > 0 {
		name = people[0].Name
	}
	var err error
	if reply {
		err = notifications.CommentReplied(ctx, s.Notify, to, b.ID, name)
	} else {
		err = notifications.BiteCommented(ctx, s.Notify, to, b.ID, name, excerpt(b.Text, 60))
	}
	if err != nil {
		s.Log.Error("bite notification failed", "error", err)
	}
}

// excerpt is the first n characters, as `text.characters.take(n)` does.
func excerpt(text string, n int) string {
	r := []rune(text)
	if len(r) <= n {
		return text
	}
	return string(r[:n])
}

// DeleteComment is BiteComments.deleteComment: own comments only; a top comment takes its
// replies with it. Answers the Bite's detail.
func (s *Service) DeleteComment(ctx context.Context, viewer, id string) (*Detail, error) {
	q := s.DB.Q()
	c, err := q.FindComment(ctx, id)
	if isNoRows(err) {
		return nil, ErrCommentUnknown
	}
	if err != nil {
		return nil, err
	}
	if c.AuthorID != viewer {
		return nil, ErrCommentNotYours
	}
	if err := q.DeleteComment(ctx, id); err != nil {
		return nil, err
	}
	return s.Detail(ctx, viewer, c.BiteID)
}
