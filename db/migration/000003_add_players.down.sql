-- Undo the up migration. Player votes cannot be turned back into client
-- votes, so they are removed before clientID is restored.
DELETE FROM votes;

ALTER TABLE votes DROP CONSTRAINT IF EXISTS votes_pollid_playerid_key;
ALTER TABLE votes DROP COLUMN IF EXISTS playerID;

ALTER TABLE votes ADD COLUMN clientID TEXT NOT NULL;
ALTER TABLE votes
    ADD CONSTRAINT votes_pollid_clientid_key UNIQUE (pollID, clientID);

DROP TABLE IF EXISTS players;

ALTER TABLE presentations DROP COLUMN IF EXISTS hasStarted;
