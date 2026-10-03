-- name: NextOrderNumber :one
SELECT nextval('order_number_seq')::bigint;

-- name: NextHandledSaleNumber :one
SELECT nextval('handled_sale_seq')::bigint;
