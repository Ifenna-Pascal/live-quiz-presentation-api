-- name: AddPlayerToPresentation :one
-- Registers a player only while the presentation has not started.
-- Returns no row (sql.ErrNoRows) if the presentation does not exist or has started.
INSERT INTO players (
    presentationID,
    playerName
)
SELECT p.id, sqlc.arg(player_name)
FROM presentations p
WHERE p.id = sqlc.arg(presentation_id)
    AND p.hasStarted = FALSE
RETURNING *;

-- name: GetPlayersInPresentation :many
SELECT
    id,
    playerName AS "name",
    createdAt
FROM players
WHERE presentationID = sqlc.arg(presentation_id)
ORDER BY createdAt;

-- name: GetPlayerInPresentation :one
SELECT *
FROM players
WHERE id = sqlc.arg(player_id)
    AND presentationID = sqlc.arg(presentation_id);

-- name: RemovePlayerFromPresentation :exec
DELETE FROM players
WHERE id = sqlc.arg(player_id)
    AND presentationID = sqlc.arg(presentation_id);
