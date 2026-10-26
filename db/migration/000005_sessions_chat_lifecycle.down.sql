DROP TABLE IF EXISTS messages;

ALTER TABLE votes
    DROP CONSTRAINT votes_pollid_optionkey_fkey,
    ADD CONSTRAINT votes_pollid_optionkey_fkey
        FOREIGN KEY (pollID, optionKey) REFERENCES options(pollID, optionKey);
ALTER TABLE options
    DROP CONSTRAINT options_pollid_fkey,
    ADD CONSTRAINT options_pollid_fkey FOREIGN KEY (pollID) REFERENCES polls(id);
ALTER TABLE polls
    DROP CONSTRAINT polls_presentationid_fkey,
    ADD CONSTRAINT polls_presentationid_fkey FOREIGN KEY (presentationID) REFERENCES presentations(id);

ALTER TABLE polls DROP COLUMN IF EXISTS isRevealed;
ALTER TABLE presentations DROP COLUMN IF EXISTS hasFinished, DROP COLUMN IF EXISTS title;
ALTER TABLE players DROP COLUMN IF EXISTS secretHash;
DROP TABLE IF EXISTS sessions;
