-- +goose Up
-- +goose StatementBegin

ALTER TABLE trade ALTER COLUMN side DROP NOT NULL;

DROP INDEX  unique_trade_no_id;

CREATE UNIQUE INDEX unique_trade_no_id ON trade (base,quote,asset,price,amount,timestamp)
    WHERE tid IS NULL;

UPDATE TRADE set side = null where side = 'UNKNOWN';
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
UPDATE TRADE set side = '' where side IS NULL;

ALTER TABLE trade ALTER COLUMN side SET NOT NULL;

DROP INDEX  unique_trade_no_id;

CREATE UNIQUE INDEX unique_trade_no_id ON trade (base,quote,asset,price,amount,side,timestamp)
    WHERE tid IS NULL;
-- +goose StatementEnd
