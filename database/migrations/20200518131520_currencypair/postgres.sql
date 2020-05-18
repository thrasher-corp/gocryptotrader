-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE IF NOT EXISTS currency
(
    id bigserial PRIMARY KEY NOT NULL,
    base_id      uuid REFERENCES currency(id),
    quote_id     uuid REFERENCES currency(id),
    asset_id     uuid references asset(id)
);
-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE currency;