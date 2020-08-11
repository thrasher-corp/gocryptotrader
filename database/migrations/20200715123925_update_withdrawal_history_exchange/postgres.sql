-- +goose Up
-- SQL in this section is executed when the migration is applied.
INSERT INTO exchange(name) SELECT exchange from withdrawal_history;
ALTER TABLE withdrawal_history ADD COLUMN exchange_name_id UUID REFERENCES exchange(id);
UPDATE withdrawal_history SET exchange_name_id = r.ID FROM (SELECT * from exchange) as r WHERE exchange = r.name;
ALTER TABLE withdrawal_history DROP COLUMN exchange;

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
ALTER TABLE withdrawal_history ADD COLUMN exchange TEXT;
UPDATE withdrawal_history SET exchange = r.name FROM (SELECT * from exchange) as r WHERE exchange_name_id = r.id;
ALTER TABLE withdrawal_history ALTER COLUMN exchange SET NOT NULL;
ALTER TABLE withdrawal_history DROP CONSTRAINT withdrawal_history_exchange_name_id_fkey;
ALTER TABLE withdrawal_history DROP COLUMN exchange_name_id;
