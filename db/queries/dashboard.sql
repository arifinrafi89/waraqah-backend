-- name: BumpSearch :exec
INSERT INTO search_log (term, day, count) VALUES ($1, $2, 1)
ON CONFLICT (term, day) DO UPDATE SET count = search_log.count + 1;

-- name: DropSearch :exec
-- A live search that grew: the shorter term it replaces counts one less (and goes at zero).
WITH lowered AS (
    UPDATE search_log SET count = count - 1 WHERE term = $1 AND day = $2 AND count > 1 RETURNING term
)
DELETE FROM search_log s WHERE s.term = $1 AND s.day = $2 AND s.count <= 1 AND NOT EXISTS (SELECT 1 FROM lowered);

-- name: TopSearches :many
SELECT term, sum(count)::integer AS count FROM search_log GROUP BY term ORDER BY count DESC, term LIMIT $1;

-- name: SeedSearch :exec
INSERT INTO search_log (term, day, count) VALUES ($1, $2, $3) ON CONFLICT (term, day) DO NOTHING;
