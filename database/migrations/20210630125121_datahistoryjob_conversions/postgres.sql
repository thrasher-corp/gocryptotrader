-- +goose Up
ALTER TABLE datahistoryjob
    ADD conversion_interval DOUBLE PRECISION,
    ADD overwrite_data boolean,
    ADD decimal_place_comparison INTEGER;
-- +goose Down
ALTER TABLE datahistoryjob
    DROP conversion_interval,
    DROP overwrite_data,
    DROP decimal_place_comparison;
