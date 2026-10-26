-- Keep only each player's first attempt per poll before restoring one vote per poll.
DELETE FROM votes v
USING votes older
WHERE v.pollID = older.pollID
    AND v.playerID = older.playerID
    AND (older.createdAt, older.id) < (v.createdAt, v.id);

ALTER TABLE votes DROP CONSTRAINT votes_pollid_playerid_optionkey_key;
ALTER TABLE votes ADD CONSTRAINT votes_pollid_playerid_key UNIQUE (pollID, playerID);
ALTER TABLE votes DROP COLUMN createdAt;

ALTER TABLE polls
    DROP COLUMN wonAt,
    DROP COLUMN winnerPlayerID,
    DROP COLUMN openedAt,
    DROP COLUMN timeLimitSeconds;
