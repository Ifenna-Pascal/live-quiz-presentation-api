-- Marks the option(s) that answer a poll correctly. Polls with no correct
-- option (opinion polls) are simply not scored.
ALTER TABLE options
    ADD COLUMN isCorrect BOOLEAN NOT NULL DEFAULT FALSE;

-- Lets "my presentations" be listed newest first. Existing rows get the
-- time this migration runs.
ALTER TABLE presentations
    ADD COLUMN createdAt TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;
