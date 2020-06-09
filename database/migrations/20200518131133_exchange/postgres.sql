-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE IF NOT EXISTS exchange
(
    id uuid    PRIMARY KEY DEFAULT gen_random_uuid(),
    name       varchar(255)  NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS "index_name_on_exchange" ON exchange using btree(name);

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE exchange;