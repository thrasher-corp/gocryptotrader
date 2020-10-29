-- +goose Up
-- +goose StatementBegin
ALTER TABLE trade DROP CONSTRAINT uniquetrade;

CREATE UNIQUE INDEX unique_trade_no_id ON trade (base,quote,asset,price,amount,side, timestamp)
    WHERE tid IS NULL;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP INDEX  unique_trade_no_id;

ALTER TABLE trade ADD CONSTRAINT uniquetrade
    unique(exchange_name_id, base, quote, asset, price, amount, side, timestamp);
-- +goose StatementEnd
