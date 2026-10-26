-- Refresh-token sessions. Only a SHA-256 hash of each refresh token is
-- stored. A token is single use: refreshing revokes it and issues a new one.
CREATE TABLE sessions (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    userID UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refreshTokenHash TEXT NOT NULL UNIQUE,
    userAgent TEXT NOT NULL DEFAULT '',
    clientIP TEXT NOT NULL DEFAULT '',
    isRevoked BOOLEAN NOT NULL DEFAULT FALSE,
    expiresAt TIMESTAMPTZ NOT NULL,
    createdAt TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX ON sessions (userID);

-- A player proves who they are with a secret token returned once at
-- registration. Only its hash is stored. Players registered before this
-- migration have no secret and can no longer vote.
ALTER TABLE players ADD COLUMN secretHash TEXT;

ALTER TABLE presentations
    ADD COLUMN title TEXT NOT NULL DEFAULT '',
    ADD COLUMN hasFinished BOOLEAN NOT NULL DEFAULT FALSE;

-- Once a poll is revealed its correct answer is public, voting on it closes
-- and it starts counting on the leaderboard.
ALTER TABLE polls ADD COLUMN isRevealed BOOLEAN NOT NULL DEFAULT FALSE;

-- Deleting a presentation removes everything that belongs to it.
ALTER TABLE polls
    DROP CONSTRAINT polls_presentationid_fkey,
    ADD CONSTRAINT polls_presentationid_fkey
        FOREIGN KEY (presentationID) REFERENCES presentations(id) ON DELETE CASCADE;
ALTER TABLE options
    DROP CONSTRAINT options_pollid_fkey,
    ADD CONSTRAINT options_pollid_fkey
        FOREIGN KEY (pollID) REFERENCES polls(id) ON DELETE CASCADE;
ALTER TABLE votes
    DROP CONSTRAINT votes_pollid_optionkey_fkey,
    ADD CONSTRAINT votes_pollid_optionkey_fkey
        FOREIGN KEY (pollID, optionKey) REFERENCES options(pollID, optionKey) ON DELETE CASCADE;

-- Room chat. authorID is the player's id or, for the host, the user's id.
CREATE TABLE messages (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    presentationID UUID NOT NULL REFERENCES presentations(id) ON DELETE CASCADE,
    playerID UUID REFERENCES players(id) ON DELETE CASCADE,
    authorID UUID NOT NULL,
    authorName TEXT NOT NULL,
    authorRole TEXT NOT NULL CHECK (authorRole IN ('host', 'player')),
    body TEXT NOT NULL,
    createdAt TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);
CREATE INDEX ON messages (presentationID, createdAt);
