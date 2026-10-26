-- name: CreateVote :one
-- Inserts a vote only if the player is registered in the same presentation
-- as the poll, proves it with their secret, and the presentation has started.
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
  AND pr.secretHash = sqlc.arg(player_secret_hash)::TEXT
  AND p.hasStarted = TRUE
RETURNING id, createdat;

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
ORDER BY votes.createdat;

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
-- One point per poll won (first correct answer). `answered` is the number of
-- polls the player tried. Ties share a rank; the faster total winning time
-- breaks the order within a tie.
SELECT
  pr.id AS "player_id",
  pr.playerName AS "player_name",
  COALESCE(w.wins, 0)::INT AS "score",
  COALESCE(a.answered, 0)::INT AS "answered",
  RANK() OVER (ORDER BY COALESCE(w.wins, 0) DESC)::INT AS "rank"
FROM players pr
  LEFT JOIN (
    SELECT winnerplayerid, COUNT(*) AS wins, SUM(EXTRACT(EPOCH FROM wonat - openedat)) AS seconds
    FROM polls WHERE winnerplayerid IS NOT NULL GROUP BY winnerplayerid
  ) w ON w.winnerplayerid = pr.id
  LEFT JOIN (
    SELECT playerid, COUNT(DISTINCT pollid) AS answered FROM votes GROUP BY playerid
  ) a ON a.playerid = pr.id
WHERE pr.presentationID = sqlc.arg(presentation_id)
ORDER BY "score" DESC, w.seconds ASC NULLS LAST, pr.playerName;

-- name: CountScoredPolls :one
-- The highest score possible so far: polls with a correct option that are
-- closed (won, revealed, timed out, or shown before the presentation finished).
SELECT COUNT(DISTINCT polls.id)::INT
FROM polls
  JOIN presentations ON polls.presentationid = presentations.id
  JOIN options ON options.pollid = polls.id AND options.iscorrect
WHERE presentations.id = sqlc.arg(presentation_id)
  AND (polls.isrevealed
    OR polls.winnerplayerid IS NOT NULL
    OR polls.openedat + make_interval(secs => polls.timelimitseconds) <= clock_timestamp()
    OR (presentations.hasfinished AND polls.pollindex <= presentations.currentpollindex));
