-- name: CreateUser :one
INSERT INTO users (
    name,
    email,
    hashedPassword
) VALUES (
    $1, $2, $3
) RETURNING *;

-- name: GetUser :one
SELECT * from users
where email = $1 LIMIT 1;

-- name: GetUserByID :one
SELECT * from users
where id = $1 LIMIT 1;

-- name: ListPresentationsByUser :many
SELECT
  p.id,
  p.title,
  p.currentpollindex,
  p.hasstarted,
  p.hasfinished,
  p.createdat,
  p.prizepool,
  p.prizecurrency,
  (SELECT COUNT(*) FROM polls WHERE polls.presentationid = p.id)::INT AS poll_count,
  (SELECT COUNT(*) FROM players WHERE players.presentationid = p.id)::INT AS player_count
FROM presentations p
WHERE p.userId = sqlc.arg(user_id)
ORDER BY p.createdat DESC, p.id
LIMIT sqlc.arg(page_limit)
OFFSET sqlc.arg(page_offset);
