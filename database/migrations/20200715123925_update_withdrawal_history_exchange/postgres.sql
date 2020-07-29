-- +goose Up
-- SQL in this section is executed when the migration is applied.
ALTER TABLE withdrawal_history RENAME COLUMN exchange TO exchange_name_id;
ALTER TABLE withdrawal_history ALTER COLUMN exchange_name_id SET DATA TYPE UUID USING (gen_random_uuid());
INSERT INTO exchange(name) SELECT INITCAP(exchange) from withdrawal_history;
ALTER TABLE withdrawal_history ADD CONSTRAINT fk_exchange_id_withdrawal_history FOREIGN KEY(exchange_name_id) REFERENCES exchange(id);

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
ALTER TABLE withdrawal_history DROP CONSTRAINT withdrawal_history_exchange_name_id_fkey;
ALTER TABLE withdrawal_history DROP COLUMN exchange_name_id;
