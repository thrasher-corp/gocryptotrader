-- +goose Up
CREATE TABLE IF NOT EXISTS trade
(
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    exchange_name_id uuid REFERENCES exchange(id) NOT NULL,
    tid varchar,
    base varchar(30) NOT NULL,
    quote varchar(30) NOT NULL,
    asset varchar NOT NULL,
    price DOUBLE PRECISION NOT NULL,
    amount DOUBLE PRECISION NOT NULL,
    side varchar NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL,
    CONSTRAINT uniquetradeid
        unique(exchange_name_id, tid),
    CONSTRAINT uniquetrade
        unique(exchange_name_id, base, quote, asset, price, amount, side, timestamp)
);
-- +goose Down
DROP TABLE trade;