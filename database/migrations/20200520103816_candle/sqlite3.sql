-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE "candle" (
    id	        text not null primary key,
    exchange_id uuid REFERENCES exchange(id),
    Base text NOT NULL,
    Quote text NOT NULL,
    Interval text NOT NULL,
    Timestamp TIMESTAMP NOT NULL,
    Open REAL NOT NULL,
    High REAL NOT NULL,
    Low REAL NOT NULL,
    Close REAL NOT NULL,
    Volume REAL NOT NULL,
    unique(Timestamp, exchange_id, Base, Quote, Interval) ON CONFLICT IGNORE
);
-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE "candle";