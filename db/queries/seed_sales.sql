-- name: SeedBookQuoteBasis :one
-- What Sell Back quotes from: the book and its cheapest printed Edition.
SELECT b.title, b.author, b.cover_seed, min(e.price_bdt)::integer AS new_price_bdt
FROM books b JOIN editions e ON e.book_id = b.id AND e.format <> 'ebook'
WHERE b.id = $1
GROUP BY b.id;
