-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE IF NOT EXISTS withdrawal_fiat
(
    id bigserial PRIMARY KEY NOT NULL,
    withdrawal_fiat_id        uuid REFERENCES withdrawal_history(id) ON DELETE CASCADE,
    bank_name                 text not null,
    bank_address              text not null,
    bank_account_name         text not null,
    bank_account_number       text not null,
    bsb                       text not null,
    swift_code                text not null,
    iban                      text not null,
    bank_code                 DOUBLE PRECISION not null
);
-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE IF EXISTS  withdrawal_fiat
