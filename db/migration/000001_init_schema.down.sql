-- Undo the up migration: drop the tables in the REVERSE order of creation,
-- because a table that others reference cannot be dropped first.
DROP TABLE IF EXISTS votes;
DROP TABLE IF EXISTS options;
DROP TABLE IF EXISTS polls;
DROP TABLE IF EXISTS presentations;
