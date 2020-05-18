-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE IF NOT EXISTS asset
(
    id bigserial PRIMARY KEY NOT NULL,
    name       varchar(255)  NOT NULL,
    short_name varchar(255)  NOT NULL,
    exchange_id uuid REFERENCES exchange(id),
    delimiter varchar(10) NOT NULL
);
-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE asset;