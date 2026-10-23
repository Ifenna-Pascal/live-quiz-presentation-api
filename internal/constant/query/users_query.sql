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

-- name: ListPresentationsByUser :many
SELECT
  p.id,
  p.currentpollindex,
  p.hasstarted,
  p.createdat,
  (SELECT COUNT(*) FROM polls WHERE polls.presentationid = p.id)::INT AS poll_count
FROM presentations p
WHERE p.userId = sqlc.arg(user_id)
ORDER BY p.createdat DESC, p.id
LIMIT sqlc.arg(page_limit)
OFFSET sqlc.arg(page_offset);
