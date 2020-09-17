-- +goose Up
-- SQL in this section is executed when the migration is applied.
ALTER TABLE candle ADD COLUMN asset text;
-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
CREATE TABLE "candle_new" (
    id	        text not null primary key,
    exchange_id uuid REFERENCES exchange(id),
    Base        text NOT NULL,
    Quote       text NOT NULL,
    Interval    text NOT NULL,
    Timestamp   TIMESTAMP NOT NULL,
    Open        REAL NOT NULL,
    High        REAL NOT NULL,
    Low         REAL NOT NULL,
    Close       REAL NOT NULL,
    Volume      REAL NOT NULL,
    unique(Timestamp, exchange_id, Base, Quote, Interval) ON CONFLICT IGNORE
);

INSERT INTO candle_new SELECT id, exchange_id, Base, Quote, Interval, Timestamp, Open, High, Low, Close, Volume FROM candle;
DROP TABLE candle;
ALTER TABLE candle_new RENAME TO candle;