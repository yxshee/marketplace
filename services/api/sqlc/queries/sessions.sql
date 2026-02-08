-- name: CreateSession :one
INSERT INTO sessions (
  user_id,
  refresh_token_hash,
  expires_at,
  ip,
  user_agent
) VALUES (
  $1,
  $2,
  $3,
  $4,
  $5
)
RETURNING *;

-- name: GetSessionByID :one
SELECT *
FROM sessions
WHERE id = $1
LIMIT 1;

-- name: RevokeSession :exec
UPDATE sessions
SET revoked_at = NOW()
WHERE id = $1;
