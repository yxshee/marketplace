-- name: CreateVendor :one
INSERT INTO vendors (
  owner_user_id,
  slug,
  display_name,
  verification_state,
  commission_override_bps
) VALUES (
  $1,
  $2,
  $3,
  'pending',
  NULL
)
RETURNING *;

-- name: GetVendorByID :one
SELECT *
FROM vendors
WHERE id = $1
LIMIT 1;

-- name: GetVendorByOwnerID :one
SELECT *
FROM vendors
WHERE owner_user_id = $1
LIMIT 1;

-- name: UpdateVendorVerification :one
UPDATE vendors
SET verification_state = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateVendorCommissionOverride :one
UPDATE vendors
SET commission_override_bps = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: InsertVendorVerificationReview :exec
INSERT INTO vendor_verification_reviews (
  vendor_id,
  actor_admin_id,
  from_state,
  to_state,
  reason
) VALUES (
  $1,
  $2,
  $3,
  $4,
  $5
);
