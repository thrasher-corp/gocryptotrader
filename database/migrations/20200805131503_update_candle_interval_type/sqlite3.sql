-- +goose Up
-- SQL in this section is executed when the migration is applied.
UPDATE candle SET interval = replace (candle.interval, '15s', '15');
UPDATE candle SET interval = replace (candle.interval, '1m', '60');
UPDATE candle SET interval = replace (candle.interval, '3m', '180');
UPDATE candle SET interval = replace (candle.interval, '5m', '300');
UPDATE candle SET interval = replace (candle.interval, '10m', '600');
UPDATE candle SET interval = replace (candle.interval, '15m', '900');
UPDATE candle SET interval = replace (candle.interval, '30m', '1800');
UPDATE candle SET interval = replace (candle.interval, '1h', '3600');
UPDATE candle SET interval = replace (candle.interval, '6h', '21600');
UPDATE candle SET interval = replace (candle.interval, '8h', '28800');
UPDATE candle SET interval = replace (candle.interval, '12h', '43200');
UPDATE candle SET interval = replace (candle.interval, '24h', '86400');
UPDATE candle SET interval = replace (candle.interval, '3d', '259200');
UPDATE candle SET interval = replace (candle.interval, '7w', '604800');
UPDATE candle SET interval = replace (candle.interval, '15d', '1296000');
UPDATE candle SET interval = replace (candle.interval, '1w', '604800');
UPDATE candle SET interval = replace (candle.interval, '2W', '1209600');
UPDATE candle SET interval = replace (candle.interval, '1M', '2678400');
UPDATE candle SET interval = replace (candle.interval, '1Y', '31536000');
UPDATE candle SET interval = replace (candle.interval, '1d', '86400');
UPDATE candle SET interval = replace (candle.interval, '72h', '259200');
UPDATE candle SET interval = replace (candle.interval, '168h', '604800');
UPDATE candle SET interval = replace (candle.interval, '360h', '1296000');
UPDATE candle SET interval = replace (candle.interval, '336h', '1209600');
UPDATE candle SET interval = replace (candle.interval, '744h', '2678400');
UPDATE candle SET interval = replace (candle.interval, '8760h', '31536000');
UPDATE candle SET interval = replace (candle.interval, '4h', '14400');
UPDATE candle SET interval = replace (candle.interval, '2h', '7200');

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
-- Nothing to run for sqlite as we leave the type as text