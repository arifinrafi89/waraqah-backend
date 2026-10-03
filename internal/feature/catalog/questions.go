package catalog

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/ids"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/textutil"
)

const maxQuestionText = 500

// Questions lists the questions about a book with their answers, oldest first.
func (s *Service) Questions(ctx context.Context, bookID string) ([]Question, error) {
	qs, err := s.q().ListQuestions(ctx, bookID)
	if err != nil {
		return nil, err
	}
	as, err := s.q().ListAnswersForBook(ctx, bookID)
	if err != nil {
		return nil, err
	}
	answers := map[string][]Answer{}
	for _, a := range as {
		answers[a.QuestionID] = append(answers[a.QuestionID], Answer{ID: a.ID, Text: a.Text, AuthorName: a.AuthorName,
			AnsweredAt: a.AnsweredAt.In(s.Loc).Truncate(time.Second), IsStaff: a.IsStaff})
	}
	out := make([]Question, 0, len(qs))
	for _, q := range qs {
		list := answers[q.ID]
		if list == nil {
			list = []Answer{}
		}
		out = append(out, Question{ID: q.ID, Text: q.Text, AskerName: q.AskerName,
			AskedAt: q.AskedAt.In(s.Loc).Truncate(time.Second), Answers: list})
	}
	return out, nil
}

func validText(text string) bool {
	n := textutil.TrimLen(text)
	return n > 0 && n <= maxQuestionText
}

// Ask adds a question. The asker name comes from the account of the reader, never the request.
func (s *Service) Ask(ctx context.Context, snap *Snapshot, userID, userName, bookID, text string) error {
	if _, ok := snap.Book(bookID); !ok {
		return ErrBookUnknown
	}
	if !validText(text) {
		return ErrQuestionInvalid
	}
	return s.q().InsertQuestion(ctx, sqlc.InsertQuestionParams{ID: "q-" + strings.TrimPrefix(ids.New("x"), "x-"),
		BookID: bookID, UserID: pgtype.Text{String: userID, Valid: true}, AskerName: userName, Text: strings.TrimSpace(text), AskedAt: s.Now()})
}

// Answer adds an answer to a question. The author and the staff flag come from the account: staff
// answer as Waraqah. An unknown question changes nothing, as in the fake API.
func (s *Service) Answer(ctx context.Context, snap *Snapshot, userID, userName string, isStaff bool, bookID, questionID, text string) error {
	if _, ok := snap.Book(bookID); !ok {
		return ErrBookUnknown
	}
	if !validText(text) {
		return ErrQuestionInvalid
	}
	name := userName
	if isStaff {
		name = "Waraqah"
	}
	return s.q().InsertAnswer(ctx, sqlc.InsertAnswerParams{ID: "a-" + strings.TrimPrefix(ids.New("x"), "x-"), ID_2: questionID,
		UserID: pgtype.Text{String: userID, Valid: true}, AuthorName: name, Text: strings.TrimSpace(text), AnsweredAt: s.Now(),
		IsStaff: isStaff, BookID: bookID})
}
