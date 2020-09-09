-- +goose Up
-- SQL in this section is executed when the migration is applied.
ALTER TABLE candle ALTER COLUMN asset SET NOT NULL;

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
ALTER TABLE candle ALTER COLUMN asset DROP NOT NULL;
