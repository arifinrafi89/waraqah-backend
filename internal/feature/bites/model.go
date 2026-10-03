// Package bites is Book-Bites: short posts that may tag a catalog Book, with likes and comments
// (one level of replies). Port of bite_fake_store.dart, bite_comments.dart and bite_fake_json.dart.
package bites

import (
	"time"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
)

// Refusal is a rule the request broke; the handler answers 200 null with its code.
type Refusal string

func (r Refusal) Error() string { return string(r) }

// Refusal codes (strings live in httpx/errors.go, documented in docs/error-codes.md).
const (
	ErrInvalid         = Refusal(httpx.ErrBiteInvalid)
	ErrUnknown         = Refusal(httpx.ErrBiteUnknown)
	ErrNotYours        = Refusal(httpx.ErrBiteNotYours)
	ErrBookUnknown     = Refusal(httpx.ErrBookUnknown)
	ErrCommentInvalid  = Refusal(httpx.ErrCommentInvalid)
	ErrCommentUnknown  = Refusal(httpx.ErrCommentUnknown)
	ErrCommentNotYours = Refusal(httpx.ErrCommentNotYours)
	ErrBanned          = Refusal(httpx.ErrReaderBanned)
	ErrBlocked         = Refusal(httpx.ErrBlockedReader)
)

// Bite is BiteModel: whose it is (isMine) and whether the viewer liked it come from the token.
type Bite struct {
	ID         string     `json:"id"`
	AuthorID   string     `json:"authorId"`
	AuthorName string     `json:"authorName"`
	AuthorArea string     `json:"authorArea"`
	Text       string     `json:"text"`
	CreatedAt  time.Time  `json:"createdAt"`
	EditedAt   *time.Time `json:"editedAt"`
	BookID     *string    `json:"bookId"`
	BookTitle  *string    `json:"bookTitle"`
	Spoiler    bool       `json:"spoiler"`
	Likes      int        `json:"likes"`
	Liked      bool       `json:"liked"`
	Comments   int        `json:"comments"`
	IsMine     bool       `json:"isMine"`
}

// Comment is BiteCommentModel: top comments carry their replies, a reply has a parentId.
type Comment struct {
	ID         string    `json:"id"`
	AuthorID   string    `json:"authorId"`
	AuthorName string    `json:"authorName"`
	Text       string    `json:"text"`
	CreatedAt  time.Time `json:"createdAt"`
	ParentID   *string   `json:"parentId"`
	IsMine     bool      `json:"isMine"`
	Replies    []Comment `json:"replies"`
}

// Detail is BiteDetailModel: one Bite with its top comments, oldest first.
type Detail struct {
	Bite     Bite      `json:"bite"`
	Comments []Comment `json:"comments"`
}

// Query is BiteQuery: For You or Following, optionally about one Book or by one Reader.
type Query struct {
	Following bool
	BookID    string
	AuthorID  string
}
