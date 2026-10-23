-- Undo the up migration: drop the column that references users first,
-- then the users table itself.
ALTER TABLE presentations DROP COLUMN IF EXISTS userID;
DROP TABLE IF EXISTS users;
