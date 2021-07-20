-- +goose Up
ALTER TABLE datahistoryjob
    ADD conversion_interval real;
ALTER TABLE datahistoryjob
    ADD overwrite_data integer;
ALTER TABLE datahistoryjob
    ADD decimal_place_comparison integer;
ALTER TABLE datahistoryjob
    ADD secondary_exchange_id text REFERENCES exchange(id) ON DELETE RESTRICT;
ALTER TABLE datahistoryjob
    ADD issue_tolerance_percentage real;
ALTER TABLE datahistoryjob
    ADD replace_on_issue integer;

-- +goose Down
ALTER TABLE datahistoryjob
    DROP replace_on_issue;
ALTER TABLE datahistoryjob
    DROP issue_tolerance_percentage;
ALTER TABLE datahistoryjob
    DROP secondary_exchange_id;
ALTER TABLE datahistoryjob
    DROP decimal_place_comparison;
ALTER TABLE datahistoryjob
    DROP overwrite_data;
ALTER TABLE datahistoryjob
    DROP conversion_interval;
