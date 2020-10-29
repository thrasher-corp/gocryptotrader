-- +goose Up
CREATE TABLE IF NOT EXISTS trade
(
    id text not null primary key,
    exchange_name_id uuid REFERENCES exchange(id) NOT NULL,
    tid TEXT,
    base text NOT NULL,
    quote text NOT NULL,
    asset TEXT NOT NULL,
    price REAL NOT NULL,
    amount REAL NOT NULL,
    side TEXT NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    CONSTRAINT uniquetradeid
        unique(exchange_name_id, tid) ON CONFLICT IGNORE,
    CONSTRAINT uniquetrade
        unique(exchange_name_id, base, quote, asset, price, amount, side, timestamp) ON CONFLICT IGNORE
);
-- +goose Down
DROP TABLE trade;
