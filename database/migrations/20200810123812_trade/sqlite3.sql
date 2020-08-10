-- +goose Up
CREATE TABLE IF NOT EXISTS trade
(
    id  TEXT PRIMARY KEY NOT NULL UNIQUE ON CONFLICT REPLACE,
    exchange_id TEXT NOT NULL UNIQUE,
    currency varchar NOT NULL,
    asset varchar NOT NULL,
    event varchar NOT NULL,
    price NUMBER NOT NULL,
    amount NUMBER NOT NULL,
    side varchar NOT NULL
);
-- +goose Down
DROP TABLE trade;
