-- name: CreateSession :one
INSERT INTO sessions (
    userID,
    refreshTokenHash,
    userAgent,
    clientIP,
    expiresAt
) VALUES (
    sqlc.arg(user_id), sqlc.arg(refresh_token_hash), sqlc.arg(user_agent), sqlc.arg(client_ip), sqlc.arg(expires_at)
) RETURNING *;

-- name: GetSessionByTokenHash :one
-- Locks the row so two refreshes with the same token cannot both succeed.
SELECT * FROM sessions
WHERE refreshTokenHash = sqlc.arg(refresh_token_hash)
FOR UPDATE;

-- name: RevokeSession :exec
UPDATE sessions SET isRevoked = TRUE WHERE id = sqlc.arg(id);

-- name: RevokeAllUserSessions :exec
-- Used when a revoked refresh token is presented again: the token may have
-- been stolen, so every session of that user is ended.
UPDATE sessions SET isRevoked = TRUE
WHERE userID = sqlc.arg(user_id) AND isRevoked = FALSE;
