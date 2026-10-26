-- Timed questions: the host sets a time limit per poll. The countdown starts
-- when the poll is first shown (openedAt). The first player to pick a correct
-- option wins the poll, which closes it on the spot.
ALTER TABLE polls
    ADD COLUMN timeLimitSeconds INT NOT NULL DEFAULT 20 CHECK (timeLimitSeconds BETWEEN 5 AND 600),
    ADD COLUMN openedAt TIMESTAMPTZ,
    ADD COLUMN winnerPlayerID UUID REFERENCES players(id) ON DELETE SET NULL,
    ADD COLUMN wonAt TIMESTAMPTZ;

-- Polls already shown in running presentations count as opened (and so are
-- already timed out).
UPDATE polls pl
SET openedAt = pl.createdAt
FROM presentations pr
WHERE pl.presentationID = pr.id
    AND pr.hasStarted
    AND pl.pollIndex <= pr.currentPollIndex;

-- A player may retry after a wrong answer, but never pick the same option
-- twice. Votes keep the time they were cast, to the microsecond.
ALTER TABLE votes ADD COLUMN createdAt TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp();
ALTER TABLE votes DROP CONSTRAINT votes_pollid_playerid_key;
ALTER TABLE votes ADD CONSTRAINT votes_pollid_playerid_optionkey_key UNIQUE (pollID, playerID, optionKey);
