-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE IF NOT EXISTS withdrawal_crypto
(
    id bigserial PRIMARY KEY NOT NULL,
    withdrawal_crypto_id uuid REFERENCES withdrawal_history(id) ON DELETE RESTRICT,
    address                   text NOT NULL,
    address_tag               text NULL,
    fee                       DOUBLE PRECISION NOT NULL
);
-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE IF EXISTS  withdrawal_crypto;