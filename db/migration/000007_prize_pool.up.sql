-- Optional prize money. When prizePool is set, the top 3 ranks share it in
-- proportion to their scores (players tied on a rank share that rank).
-- Amounts are whole units of prizeCurrency.
ALTER TABLE presentations
    ADD COLUMN prizePool BIGINT CHECK (prizePool IS NULL OR prizePool > 0),
    ADD COLUMN prizeCurrency TEXT NOT NULL DEFAULT 'NGN';
