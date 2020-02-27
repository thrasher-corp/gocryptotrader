-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE IF NOT EXISTS "withdrawal_fiat"
(
    id	        integer not null primary key,
    bank_name                 text not null,
    bank_address              text not null,
    bank_account_name         text not null,
    bank_account_number       text not null,
    bsb                       text not null DEFAULT '',
    swift_code                text not null DEFAULT '',
    iban                      text not null DEFAULT '',
    bank_code                 real not null,
    withdrawal_history_id  text NOT NULL,
    FOREIGN KEY(withdrawal_history_id) REFERENCES withdrawal_history(id) ON DELETE RESTRICT
);
-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE IF EXISTS  withdrawal_fiat;
