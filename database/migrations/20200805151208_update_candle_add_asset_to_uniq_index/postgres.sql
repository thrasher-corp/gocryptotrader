-- +goose Up
ALTER TABLE candle DROP CONSTRAINT candle_timestamp_exchange_id_base_quote_interval_key;
ALTER TABLE candle ADD CONSTRAINT candle_timestamp_exchange_id_base_quote_interval_asset_key UNIQUE(Timestamp, exchange_id, Base, Quote, Interval, asset);
-- +goose Down
ALTER TABLE candle DROP CONSTRAINT candle_timestamp_exchange_id_base_quote_interval_asset_key;
ALTER TABLE candle ADD CONSTRAINT candle_timestamp_exchange_id_base_quote_interval_key UNIQUE(Timestamp, exchange_id, Base, Quote, Interval);