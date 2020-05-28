-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE IF NOT EXISTS candle
(
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    exchange_id uuid REFERENCES exchange(id),
    base varchar(30) NOT NULL,
    quote varchar(30) NOT NULL,
    interval varchar(30) NOT NULL,
    date timestamp with time zone,
    open numeric,
    high numeric,
    low numeric,
    close numeric,
    volume numeric,
    unique(exchange_id, date, base, quote)
);

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE candle;