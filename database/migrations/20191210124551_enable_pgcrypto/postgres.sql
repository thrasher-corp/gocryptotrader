-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE EXTENSION IF NOT EXISTS pgcrypto;
-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP EXTENSION IF EXISTS pgcrypto;