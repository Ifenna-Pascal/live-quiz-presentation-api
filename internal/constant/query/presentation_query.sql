-- name: GetPresentation :one
SELECT *
FROM presentations
WHERE id = $1;

-- name: CreatePresentation :one
-- Poll 0 is "nothing on screen yet". Starting moves to poll 1.
INSERT INTO presentations
  (currentpollindex, userId, title, prizePool, prizeCurrency)
VALUES
  (0, sqlc.arg(user_id), sqlc.arg(title), sqlc.narg(prize_pool), sqlc.arg(prize_currency))
RETURNING id;

-- name: InsertPolls :exec
-- Inserts every poll of `polls` (a JSON array of {question, time_limit_seconds,
-- options:[{key, value, correct}]}) with poll indexes starting at 1.
WITH data_cte AS (
  SELECT sqlc.arg(polls)::jsonb AS data
),
polls_cte AS (
  INSERT INTO polls (question, presentationid, pollindex, timelimitseconds)
  SELECT
    (arr.elem ->> 'question')::TEXT,
    sqlc.arg(presentation_id)::uuid,
    arr.idx,
    COALESCE(NULLIF((arr.elem ->> 'time_limit_seconds')::INT, 0), 20)
  FROM data_cte dc, jsonb_array_elements(dc.data) WITH ORDINALITY AS arr(elem, idx)
  RETURNING id, pollindex
)
INSERT INTO options (pollid, optionkey, optionvalue, iscorrect)
SELECT
  pc.id,
  (o ->> 'key')::TEXT,
  (o ->> 'value')::TEXT,
  COALESCE((o ->> 'correct')::BOOLEAN, FALSE)
FROM polls_cte pc, data_cte dc,
  LATERAL jsonb_array_elements(dc.data -> (pc.pollindex - 1)::INT -> 'options') AS o;

-- name: DeletePolls :exec
DELETE FROM polls WHERE presentationid = sqlc.arg(presentation_id);

-- name: UpdatePresentationDetails :exec
UPDATE presentations
SET title = sqlc.arg(title),
  prizePool = sqlc.narg(prize_pool),
  prizeCurrency = sqlc.arg(prize_currency)
WHERE id = sqlc.arg(presentation_id);

-- name: DeletePresentation :execrows
DELETE FROM presentations WHERE id = sqlc.arg(presentation_id);

-- name: GetPresentationAndPoll :one
SELECT
  pr.id as presentation_id,
  pr.currentpollindex,
  pr.userId,
  pr.hasfinished,
  pl.id as poll_id,
  pl.question as question,
  pl.pollindex,
  pl.isrevealed
FROM
  presentations as pr JOIN polls as pl ON pr.id=pl.presentationid
WHERE
  pr.id=sqlc.arg(presentation_id) AND pl.id=sqlc.arg(poll_id);

-- name: LockPollForVote :one
-- Locks the poll row until the vote's transaction ends, so answers to one
-- poll are handled strictly one after another: exactly one player can win.
SELECT
  pr.currentpollindex,
  pr.hasfinished,
  pl.pollindex,
  pl.isrevealed,
  pl.winnerplayerid,
  pl.openedat,
  pl.timelimitseconds,
  clock_timestamp()::timestamptz AS now
FROM polls pl
  JOIN presentations pr ON pr.id = pl.presentationid
WHERE pr.id = sqlc.arg(presentation_id) AND pl.id = sqlc.arg(poll_id)
FOR UPDATE OF pl;

-- name: IsOptionCorrect :one
SELECT iscorrect FROM options
WHERE pollid = sqlc.arg(poll_id) AND optionkey = sqlc.arg(option_key);

-- name: PollHasCorrectOption :one
SELECT EXISTS (SELECT 1 FROM options WHERE pollid = sqlc.arg(poll_id) AND iscorrect);

-- name: PlayerHasVoted :one
SELECT EXISTS (SELECT 1 FROM votes WHERE pollid = sqlc.arg(poll_id) AND playerid = sqlc.arg(player_id));

-- name: SetPollWinner :one
-- Records the winner and closes the poll. Matches nothing if someone already won.
UPDATE polls
SET winnerPlayerID = sqlc.arg(player_id), wonAt = clock_timestamp(), isRevealed = TRUE
WHERE id = sqlc.arg(poll_id) AND winnerPlayerID IS NULL
RETURNING wonat, openedat;

-- name: OpenCurrentPoll :exec
-- Starts the countdown of the poll on screen, the first time it is shown.
UPDATE polls
SET openedAt = clock_timestamp()
WHERE presentationid = sqlc.arg(presentation_id)
  AND openedAt IS NULL
  AND pollindex = (SELECT pr.currentpollindex FROM presentations pr WHERE pr.id = sqlc.arg(presentation_id));

-- name: GetPoll :one
-- The poll currently on screen, with its timer and winner.
SELECT
  p.id AS id,
  p.question AS question,
  p.pollindex,
  p.isrevealed,
  p.timelimitseconds,
  p.openedat,
  p.wonat,
  p.winnerplayerid,
  w.playername AS winner_name,
  COALESCE((
    SELECT v.optionkey FROM votes v
    WHERE v.pollid = p.id AND v.playerid = p.winnerplayerid
    ORDER BY v.createdat DESC LIMIT 1
  ), '')::TEXT AS winner_key,
  pr.hasfinished,
  clock_timestamp()::timestamptz AS server_time,
  (
    SELECT jsonb_agg(jsonb_build_object(
      'key', o.optionkey,
      'value', o.optionvalue,
      'correct', o.iscorrect
    ) ORDER BY o.optionkey)
  FROM options o
  WHERE o.pollid = p.id
  )::jsonb AS options
FROM polls p
  JOIN presentations pr ON pr.id = p.presentationid
  LEFT JOIN players w ON w.id = p.winnerplayerid
WHERE p.presentationid = $1 and p.pollindex = pr.currentpollindex;

-- name: MoveToNextPoll :one
UPDATE presentations
SET currentpollindex = currentpollindex + 1
WHERE presentations.id = $1
  AND currentpollindex + 1 <= (SELECT count(*) FROM polls WHERE polls.presentationid = $1)
RETURNING currentpollindex;

-- name: MoveToPreviousPoll :one
-- Never goes below poll 1, so a poll always stays on screen.
UPDATE presentations
SET currentpollindex = currentpollindex - 1
WHERE presentations.id = $1 AND currentpollindex - 1 >= 1
RETURNING currentpollindex;

-- name: RevealCurrentPoll :one
UPDATE polls
SET isRevealed = TRUE
WHERE presentationid = sqlc.arg(presentation_id)
  AND pollindex = (SELECT pr.currentpollindex FROM presentations pr WHERE pr.id = sqlc.arg(presentation_id))
RETURNING id;

-- name: ListPolls :many
SELECT
  p.id AS id,
  p.question,
  p.isrevealed,
  p.timelimitseconds,
  p.winnerplayerid,
  jsonb_agg(jsonb_build_object(
        'key', o.optionKey,
        'value', o.optionValue,
        'correct', o.isCorrect
    ) ORDER BY o.optionKey)::jsonb AS options
FROM
  polls p
  JOIN
  options o ON p.id = o.pollID
WHERE
    p.presentationID = $1
GROUP BY
    p.id, p.question, p.isrevealed, p.timelimitseconds, p.winnerplayerid, p.pollindex, p.createdAt
ORDER BY
    p.pollindex;

-- name: StartPresentation :one
-- Closes player registration and displays the first poll, so a started
-- presentation always has a poll on screen. GREATEST keeps the current poll
-- if the owner already moved forward before starting.
-- Returns no row if it was already started.
UPDATE presentations
SET hasStarted = TRUE,
  currentpollindex = GREATEST(currentpollindex, 1)
WHERE id = sqlc.arg(presentation_id)
  AND hasStarted = FALSE
RETURNING *;

-- name: FinishPresentation :one
-- Returns no row unless the presentation is running.
UPDATE presentations
SET hasFinished = TRUE
WHERE id = sqlc.arg(presentation_id)
  AND hasStarted = TRUE
  AND hasFinished = FALSE
RETURNING *;
