-- +goose Up
CREATE TABLE IF NOT EXISTS trade
(
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    exchange_id uuid REFERENCES script(id),
    currency varchar NOT NULL,
    asset varchar NOT NULL,
    event varchar NOT NULL,
    price TIMESTAMP NOT NULL DEFAULT (now() at time zone 'utc'),
    amount TIMESTAMP NOT NULL DEFAULT (now() at time zone 'utc'),
    side varchar NOT NULL
);
-- +goose Down
DROP TABLE trade;