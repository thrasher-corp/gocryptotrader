-- +goose Up
-- SQL in this section is executed when the migration is applied.
ALTER TABLE candle ADD COLUMN asset text;
-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
-- SQLite does not support dropping columns