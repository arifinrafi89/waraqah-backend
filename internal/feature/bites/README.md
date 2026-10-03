# bites

Book-Bites: short posts that may tag a catalog Book, with likes and comments (one level of replies).

- **Frontend files:** `lib/features/bites/data/sources/{bite_fake_api,bite_fake_store,bite_fake_json,bite_records,bite_comments,bite_fixtures,bite_fixture_texts}.dart`; rules `bite_rules.dart` (`rules.go`, ported with its test; lengths count graphemes through `textutil.Graphemes`, which adds Unicode 15.1's conjunct rule to `rivo/uniseg` so ক্ষ is one character).
- **Endpoints:** `GET /bites` (`feed=forYou|following`, `bookId`, `authorId`; newest first, at most 30) and `GET /bites/detail` (public, optional token); `POST /bites/post|edit|delete|like|comments/post|comments/delete` (me).
- **Tables:** `bites`, `bite_likes`, `bite_comments` (a reply's parent is a top comment; deleting cascades) (migration `0011`).
- **Per viewer:** `isMine` and `liked` come from the token. Readers blocked either way, banned or deleted are left out of every feed (that is also the account-delete hook: no separate one is needed); likes and comments on a blocked reader's Bite are refused; banned readers cannot post or comment.
- **Notifications:** the Bite's author hears about a comment, the top comment's author about a reply, never the writer.
- **Moderation:** `Bites` and `Comments` are registered as the `bite` and `comment` subjects.
