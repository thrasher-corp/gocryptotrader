-- +goose Up
-- SQL in this section is executed when the migration is applied.
ALTER TABLE withdrawal_history ALTER COLUMN exchange_name_id SET NOT NULL;

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
ALTER TABLE withdrawal_history ALTER COLUMN exchange_name_id DROP NOT NULL;
