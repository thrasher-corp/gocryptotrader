-- +goose Up
ALTER TABLE datahistoryjob
    ADD conversion_interval real;
ALTER TABLE datahistoryjob
    ADD overwrite_data integer;
ALTER TABLE datahistoryjob
    ADD decimal_place_comparison integer;

-- +goose Down
ALTER TABLE datahistoryjob
    DROP conversion_interval;
ALTER TABLE datahistoryjob
    DROP overwrite_data;
ALTER TABLE datahistoryjob
    DROP decimal_place_comparison;
