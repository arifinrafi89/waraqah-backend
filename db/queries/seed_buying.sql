-- name: UpsertSeedPerson :exec
INSERT INTO users (id, email, name, role, password_hash, area, district, member_since, sold_before)
VALUES ($1, $2, $3, 'reader', $4, $5, $6, $7, $8)
ON CONFLICT (id) DO UPDATE
SET email = EXCLUDED.email, name = EXCLUDED.name, area = EXCLUDED.area, district = EXCLUDED.district,
    member_since = EXCLUDED.member_since, sold_before = EXCLUDED.sold_before,
    password_hash = COALESCE(users.password_hash, EXCLUDED.password_hash);

-- name: UpsertSeedFlashItem :exec
INSERT INTO flash_sale_items (edition_id, position, price_bdt) VALUES ($1, $2, $3)
ON CONFLICT (edition_id) DO UPDATE SET position = EXCLUDED.position, price_bdt = EXCLUDED.price_bdt;

-- name: UpsertSeedBundle :exec
INSERT INTO bundles (id, position, title, edition_ids, price_bdt) VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (id) DO UPDATE
SET position = EXCLUDED.position, title = EXCLUDED.title, edition_ids = EXCLUDED.edition_ids,
    price_bdt = EXCLUDED.price_bdt;

-- name: UpsertSeedPreorder :exec
INSERT INTO preorders (edition_id, position) VALUES ($1, $2)
ON CONFLICT (edition_id) DO UPDATE SET position = EXCLUDED.position;

-- name: UpsertSeedShare :exec
INSERT INTO wishlist_shares (id, user_id, owner_name) VALUES ($1, $2, $3)
ON CONFLICT (id) DO UPDATE SET owner_name = EXCLUDED.owner_name;

-- name: UpsertSeedWishlistItem :exec
INSERT INTO wishlist_items (user_id, book_id, added_at) VALUES ($1, $2, $3)
ON CONFLICT (user_id, book_id) DO NOTHING;
