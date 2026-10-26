-- name: CreateMessage :one
INSERT INTO messages (
    presentationID,
    playerID,
    authorID,
    authorName,
    authorRole,
    body
) VALUES (
    sqlc.arg(presentation_id), sqlc.narg(player_id), sqlc.arg(author_id),
    sqlc.arg(author_name), sqlc.arg(author_role), sqlc.arg(body)
) RETURNING *;

-- name: ListRecentMessages :many
-- The newest messages, returned oldest first.
SELECT recent.* FROM (
    SELECT * FROM messages
    WHERE presentationID = sqlc.arg(presentation_id)
    ORDER BY createdAt DESC
    LIMIT sqlc.arg(page_limit)
) AS recent
ORDER BY recent.createdAt ASC;

-- name: ListMessagesAfter :many
SELECT * FROM messages
WHERE presentationID = sqlc.arg(presentation_id)
    AND createdAt > sqlc.arg(after)
ORDER BY createdAt ASC
LIMIT sqlc.arg(page_limit);
