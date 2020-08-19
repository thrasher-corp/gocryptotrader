-- +goose Up
ALTER TABLE candle DROP CONSTRAINT candle_timestamp_exchange_id_base_quote_interval_asset_key;
ALTER TABLE candle RENAME COLUMN exchange_id TO exchange_name_id;
ALTER TABLE candle ALTER COLUMN exchange_name_id SET NOT NULL;
ALTER TABLE candle ADD CONSTRAINT candle_timestamp_exchange_id_base_quote_interval_asset_key UNIQUE(Timestamp, exchange_name_id, Base, Quote, Interval, asset);
-- +goose Down
ALTER TABLE candle DROP CONSTRAINT candle_timestamp_exchange_id_base_quote_interval_asset_key;
ALTER TABLE candle RENAME COLUMN exchange_name_id TO exchange_id;
ALTER TABLE candle ALTER COLUMN exchange_id SET NOT NULL;
ALTER TABLE candle ADD CONSTRAINT candle_timestamp_exchange_id_base_quote_interval_asset_key UNIQUE(Timestamp, exchange_id, Base, Quote, Interval);