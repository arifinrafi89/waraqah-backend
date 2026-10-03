-- name: ListDonatePlaces :many
SELECT * FROM donate_places WHERE removed_at IS NULL ORDER BY position;

-- name: GetDonatePlace :one
SELECT * FROM donate_places WHERE id = $1 AND removed_at IS NULL;

-- name: LockDonatePlace :one
SELECT * FROM donate_places WHERE id = $1 AND removed_at IS NULL FOR UPDATE;

-- name: ListDonateNeeds :many
SELECT * FROM donate_needs WHERE place_id = ANY($1::text[]) ORDER BY place_id, position;

-- name: InsertDonatePlace :exec
INSERT INTO donate_places (id, name, kind, district, area, story) VALUES ($1, $2, $3, $4, $5, $6);

-- name: UpdateDonatePlace :exec
UPDATE donate_places SET name = $2, kind = $3, district = $4, area = $5, story = $6 WHERE id = $1;

-- name: DeleteDonateNeeds :exec
DELETE FROM donate_needs WHERE place_id = $1;

-- name: InsertDonateNeed :exec
INSERT INTO donate_needs (place_id, position, book_id, wanted, received) VALUES ($1, $2, $3, $4, $5);

-- name: RemoveDonatePlace :execrows
UPDATE donate_places SET removed_at = now() WHERE id = $1 AND removed_at IS NULL;

-- name: AddDonationReceived :exec
UPDATE donate_needs SET received = received + $3 WHERE place_id = $1 AND book_id = $2;

-- name: SeedDonatePlaceExists :one
SELECT EXISTS (SELECT 1 FROM donate_places WHERE id = $1);

-- name: SeedDonatePlace :exec
INSERT INTO donate_places (id, name, kind, district, area, story) VALUES ($1, $2, $3, $4, $5, $6);

-- name: SeedDonateNeed :exec
INSERT INTO donate_needs (place_id, position, book_id, wanted, received) VALUES ($1, $2, $3, $4, $5);
