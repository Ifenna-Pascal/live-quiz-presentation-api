-- Registration is open while hasStarted is false. Once the presenter starts
-- the presentation, no new players can join.
ALTER TABLE presentations
    ADD COLUMN hasStarted BOOLEAN NOT NULL DEFAULT FALSE;

-- Players are pre-registered per presentation. The same name can exist in
-- different presentations, but only once within a presentation.
CREATE TABLE players (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    presentationID UUID NOT NULL REFERENCES presentations(id) ON DELETE CASCADE,
    playerName TEXT NOT NULL,
    createdAt TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(presentationID, playerName)
);

-- Votes now belong to a registered player instead of an arbitrary client id.
-- Existing anonymous votes cannot be attributed to a player, so they are removed.
DELETE FROM votes;

ALTER TABLE votes DROP CONSTRAINT IF EXISTS votes_pollid_clientid_key;
ALTER TABLE votes DROP COLUMN IF EXISTS clientID;

ALTER TABLE votes
    ADD COLUMN playerID UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE;

-- A player can vote only once per poll.
ALTER TABLE votes
    ADD CONSTRAINT votes_pollid_playerid_key UNIQUE (pollID, playerID);

CREATE INDEX ON votes (playerID);
