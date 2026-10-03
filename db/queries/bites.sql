-- name: ListBites :many
-- A feed, newest first, at most 30: authors that are deleted, banned or hidden from the viewer
-- (blocks either way) are left out.
SELECT sqlc.embed(b), u.name AS author_name, u.area AS author_area,
       (SELECT count(*) FROM bite_likes l WHERE l.bite_id = b.id)::integer AS likes,
       EXISTS (SELECT 1 FROM bite_likes l WHERE l.bite_id = b.id AND l.user_id = sqlc.arg(viewer)::text) AS liked,
       (SELECT count(*) FROM bite_comments c WHERE c.bite_id = b.id)::integer AS comments
FROM bites b JOIN users u ON u.id = b.author_id
WHERE u.deleted_at IS NULL AND NOT u.banned
  AND NOT (b.author_id = ANY (sqlc.arg(hidden)::text[]))
  AND (NOT sqlc.arg(only_following)::boolean OR b.author_id = ANY (sqlc.arg(following)::text[]))
  AND (sqlc.narg(book_id)::text IS NULL OR b.book_id = sqlc.narg(book_id)::text)
  AND (sqlc.narg(author_id)::text IS NULL OR b.author_id = sqlc.narg(author_id)::text)
ORDER BY b.created_at DESC, b.position DESC
LIMIT 30;

-- name: GetBite :one
SELECT sqlc.embed(b), u.name AS author_name, u.area AS author_area,
       (SELECT count(*) FROM bite_likes l WHERE l.bite_id = b.id)::integer AS likes,
       EXISTS (SELECT 1 FROM bite_likes l WHERE l.bite_id = b.id AND l.user_id = sqlc.arg(viewer)::text) AS liked,
       (SELECT count(*) FROM bite_comments c WHERE c.bite_id = b.id)::integer AS comments
FROM bites b JOIN users u ON u.id = b.author_id
WHERE b.id = sqlc.arg(id) AND u.deleted_at IS NULL;

-- name: FindBite :one
SELECT * FROM bites WHERE id = $1;

-- name: InsertBite :exec
INSERT INTO bites (id, author_id, text, book_id, spoiler, created_at) VALUES ($1, $2, $3, $4, $5, $6);

-- name: UpdateBite :exec
UPDATE bites SET text = $2, book_id = $3, spoiler = $4, edited_at = $5 WHERE id = $1;

-- name: DeleteBite :exec
DELETE FROM bites WHERE id = $1;

-- name: LikeBite :exec
INSERT INTO bite_likes (bite_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: UnlikeBite :exec
DELETE FROM bite_likes WHERE bite_id = $1 AND user_id = $2;

-- name: ListBiteComments :many
SELECT c.*, u.name AS author_name
FROM bite_comments c JOIN users u ON u.id = c.author_id
WHERE c.bite_id = $1 ORDER BY c.position;

-- name: FindComment :one
SELECT * FROM bite_comments WHERE id = $1;

-- name: InsertComment :exec
INSERT INTO bite_comments (id, bite_id, parent_id, author_id, text, created_at) VALUES ($1, $2, $3, $4, $5, $6);

-- name: DeleteComment :exec
DELETE FROM bite_comments WHERE id = $1;

-- name: CountBitesBy :one
SELECT count(*)::integer FROM bites WHERE author_id = $1;

-- name: SeedBite :exec
INSERT INTO bites (id, author_id, text, book_id, spoiler, created_at) VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT (id) DO NOTHING;

-- name: SeedComment :exec
INSERT INTO bite_comments (id, bite_id, parent_id, author_id, text, created_at) VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT (id) DO NOTHING;

-- name: SeedLike :exec
INSERT INTO bite_likes (bite_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;
