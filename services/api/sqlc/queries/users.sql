-- name: CreateUser :one
INSERT INTO users (
  email,
  password_hash,
  user_type,
  status
) VALUES (
  $1,
  $2,
  $3,
  'active'
)
RETURNING *;

-- name: GetUserByEmail :one
SELECT *
FROM users
WHERE lower(email) = lower($1)
LIMIT 1;

-- name: GetUserByID :one
SELECT *
FROM users
WHERE id = $1
LIMIT 1;

-- name: UpdateUserRole :exec
UPDATE users
SET user_type = $2,
    updated_at = NOW()
WHERE id = $1;
