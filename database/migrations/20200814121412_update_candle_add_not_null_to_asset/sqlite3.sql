-- +goose Up
CREATE TABLE "candle_new" (
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
                              Asset text NOT NULL,
                              unique(Timestamp, exchange_id, Base, Quote, Interval, Asset) ON CONFLICT IGNORE
);
INSERT INTO candle_new SELECT id, exchange_id, Base, Quote, Interval, Timestamp, Open, High, Low, Close, Volume, Asset FROM candle;
DROP TABLE candle;
ALTER TABLE candle_new RENAME TO candle;
-- +goose Down
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
                              Asset       TEXT,
                              unique(Timestamp, exchange_id, Base, Quote, Interval, Asset) ON CONFLICT IGNORE
);

INSERT INTO candle_new SELECT id, exchange_id, Base, Quote, Interval, Timestamp, Open, High, Low, Close, Volume, Asset FROM candle;
DROP TABLE candle;
ALTER TABLE candle_new RENAME TO candle;
