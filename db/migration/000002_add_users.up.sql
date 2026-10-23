-- Create the Users table with UUID primary key
CREATE TABLE users (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    hashedPassword TEXT NOT NULL,
    createdAt TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- A user can own many presentations. The column is nullable so that
-- presentations created before this migration remain valid.
ALTER TABLE presentations
    ADD COLUMN userID UUID REFERENCES users(id) ON DELETE CASCADE;

CREATE INDEX ON presentations (userID);
