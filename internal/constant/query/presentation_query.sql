-- name: GetPresentation :one
SELECT *
FROM presentations
WHERE id = $1;


-- name: CreatePresentationAndPolls :one
WITH presentations_cte AS (
INSERT INTO presentations
  (id, currentpollindex, userId)
VALUES
  (uuid_generate_v4(), 0, sqlc.arg(user_id))
RETURNING id
),
data_cte AS
(
  SELECT
  to_jsonb(sqlc.arg(polls)
::jsonb) AS data
),
polls_cte AS
(
INSERT INTO polls
  (question, presentationid, pollindex)
SELECT
  (arr.elem ->> 'question')
::TEXT AS question,
    pc.id AS presentationid,
    arr.idx AS pollindex
  FROM presentations_cte pc, data_cte dc,
    jsonb_array_elements
(dc.data)
WITH ORDINALITY AS arr
(elem, idx)
  RETURNING id, pollindex
),
options_cte AS
(
INSERT INTO options
  (pollid, optionkey, optionvalue, iscorrect)
SELECT
  pc.id,
  (o ->> 'key')
::TEXT AS optionkey,
(o ->> 'value')::TEXT AS optionvalue,
COALESCE((o ->> 'correct')::BOOLEAN, FALSE) AS iscorrect
  FROM polls_cte pc, data_cte dc,
    LATERAL jsonb_array_elements
(dc.data -> pc.pollindex - 1 -> 'options') AS o
  RETURNING id,pollid
)

SELECT id
FROM presentations_cte;

-- name: GetPresentationAndPoll :one
SELECT 
  pr.id as presentation_id,
  pr.currentpollindex,
  pr.userId,
  pl.id as poll_id,
  pl.question as question,
  pl.pollindex
FROM
  presentations as pr JOIN polls as pl ON pr.id=pl.presentationid
WHERE 
  pr.id=sqlc.arg(presentation_id) AND pl.id=sqlc.arg(poll_id);

-- name: GetPoll :one
SELECT
  p.id AS id,
  p.question AS question,
  (
    SELECT jsonb_agg(jsonb_build_object(
      'key', o.optionkey,
      'value', o.optionvalue
    )) AS options
  FROM options o
  WHERE o.pollid = p.id
  ) AS options
FROM polls p
WHERE p.presentationid = $1 and p.pollindex=(SELECT currentpollindex
  FROM presentations
  WHERE id=$1);


-- name: GetNextPoll :one
WITH updated_polls_cte AS(
UPDATE presentations
	SET currentpollindex = currentpollindex + 1
	WHERE presentations.id = $1 AND currentpollindex + 1 <= (
    SELECT count(*)
  FROM polls
  WHERE presentationid = $1
  )
RETURNING id, currentpollindex
)
SELECT
  p.id AS id,
  p.question AS question,
  (
    SELECT jsonb_agg(jsonb_build_object(
      'key', o.optionkey,
      'value', o.optionvalue
    )) AS options
  FROM options o
    INNER JOIN polls p ON o.pollid = p.id
  WHERE p.presentationid = upc.id AND p.pollindex=upc.currentpollindex
  ) AS options
FROM polls p
  INNER JOIN updated_polls_cte upc ON p.presentationid = upc.id AND p.pollindex=upc.currentpollindex
WHERE upc.id IS NOT NULL;


-- name: GetPreviousPoll :one
WITH updated_polls_cte AS(
UPDATE presentations
	SET currentpollindex = currentpollindex - 1
	WHERE presentations.id = $1 AND currentpollindex - 1 >=0
RETURNING id, currentpollindex
)
SELECT
  p.id AS id,
  p.question AS question,
  (
    SELECT jsonb_agg(jsonb_build_object(
      'key', o.optionkey,
      'value', o.optionvalue
    )) AS options
  FROM options o
    INNER JOIN polls p ON o.pollid = p.id
  WHERE p.presentationid = upc.id AND p.pollindex=upc.currentpollindex
  ) AS options
FROM polls p
  INNER JOIN updated_polls_cte upc ON p.presentationid = upc.id AND p.pollindex=upc.currentpollindex
WHERE upc.id IS NOT NULL;

-- name: ListPolls :many
SELECT
  p.id AS id,
  p.question,
  jsonb_agg(jsonb_build_object(
        'key', o.optionKey,
        'value', o.optionValue
    )) AS options
FROM
  polls p
  JOIN
  options o ON p.id = o.pollID
WHERE
    p.presentationID = $1
GROUP BY
    p.id, p.question, p.pollindex, p.createdAt
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
