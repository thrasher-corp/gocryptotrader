-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE IF NOT EXISTS currency
(
    id bigserial PRIMARY KEY NOT NULL,
    name       varchar(255)  NOT NULL,
    short_name varchar(255)  NOT NULL,
    fiat   boolean NOT NULL,
    crypto boolean NOT NULL,
    CONSTRAINT currency_name_uniq UNIQUE (name)
);
-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE currency;