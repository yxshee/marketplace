-- name: CreateProduct :one
INSERT INTO products (
  vendor_id,
  title,
  description,
  category_id,
  tags,
  price_incl_tax_cents,
  currency,
  stock_qty,
  status
) VALUES (
  $1,
  $2,
  $3,
  $4,
  $5,
  $6,
  $7,
  $8,
  'draft'
)
RETURNING *;

-- name: GetProductByID :one
SELECT *
FROM products
WHERE id = $1
LIMIT 1;

-- name: SubmitProductForModeration :one
UPDATE products
SET status = 'pending_approval',
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: InsertProductModerationEvent :exec
INSERT INTO product_moderation (
  product_id,
  state,
  reviewed_by,
  reason
) VALUES (
  $1,
  $2,
  $3,
  $4
);

-- name: ModerateProduct :one
UPDATE products
SET status = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: ListVisibleProducts :many
SELECT p.*
FROM products p
JOIN vendors v ON p.vendor_id = v.id
WHERE p.status = 'approved'
  AND v.verification_state = 'verified'
ORDER BY p.created_at DESC
LIMIT $1 OFFSET $2;
