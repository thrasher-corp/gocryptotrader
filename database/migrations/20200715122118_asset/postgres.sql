-- +goose Up
-- SQL in this section is executed when the migration is applied.
ALTER TABLE candle ADD COLUMN asset VARCHAR(255) NOT NULL;
-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
ALTER TABLE candle DROP COLUMN asset;
