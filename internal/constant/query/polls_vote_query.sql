-- name: CreateVote :one
-- Inserts a vote only if the player is registered in the same presentation
-- as the poll and the presentation has started.
-- Returns no row (sql.ErrNoRows) if any of those checks fail.
INSERT INTO votes
  (pollid, optionkey, playerid)
SELECT
  pl.id, sqlc.arg(option_key), pr.id
FROM polls pl
  JOIN presentations p ON pl.presentationID = p.id
  JOIN players pr ON pr.presentationID = p.id
WHERE pl.id = sqlc.arg(poll_id)
  AND pr.id = sqlc.arg(player_id)
  AND p.hasStarted = TRUE
RETURNING id;

-- name: GetVotes :many
SELECT
  votes.optionkey AS "key",
  votes.playerid AS "player_id",
  players.playerName AS "player_name"
FROM votes
  JOIN polls ON votes.pollid = polls.id
  JOIN presentations ON polls.presentationID = presentations.id
  JOIN players ON votes.playerid = players.id
WHERE presentations.id = sqlc.arg(presentation_id)
  AND polls.id = sqlc.arg(poll_id)
ORDER BY votes.optionkey;

-- name: ListPresentationVotes :many
-- Every vote of every poll in the presentation, with whether it was correct.
SELECT
  polls.id AS "poll_id",
  polls.pollindex AS "poll_index",
  votes.optionkey AS "key",
  options.iscorrect AS "is_correct",
  votes.playerid AS "player_id",
  players.playerName AS "player_name"
FROM votes
  JOIN polls ON votes.pollid = polls.id
  JOIN options ON options.pollid = votes.pollid AND options.optionkey = votes.optionkey
  JOIN players ON votes.playerid = players.id
WHERE polls.presentationID = sqlc.arg(presentation_id)
ORDER BY polls.pollindex, players.playerName;

-- name: GetLeaderboard :many
-- One row per registered player: one point per vote on a correct option.
-- Players who never voted are included with 0. Ties share the same rank.
SELECT
  pr.id AS "player_id",
  pr.playerName AS "player_name",
  COUNT(o.pollid)::INT AS "score",
  COUNT(v.id)::INT AS "answered",
  RANK() OVER (ORDER BY COUNT(o.pollid) DESC)::INT AS "rank"
FROM players pr
  LEFT JOIN votes v ON v.playerid = pr.id
  LEFT JOIN options o ON o.pollid = v.pollid AND o.optionkey = v.optionkey AND o.iscorrect
WHERE pr.presentationID = sqlc.arg(presentation_id)
GROUP BY pr.id, pr.playerName
ORDER BY "score" DESC, pr.playerName;

-- name: CountScoredPolls :one
-- The highest score possible so far: polls that have a correct option and
-- have already been displayed.
SELECT COUNT(DISTINCT polls.id)::INT
FROM polls
  JOIN presentations ON polls.presentationid = presentations.id
  JOIN options ON options.pollid = polls.id AND options.iscorrect
WHERE presentations.id = sqlc.arg(presentation_id)
  AND polls.pollindex <= presentations.currentpollindex;
