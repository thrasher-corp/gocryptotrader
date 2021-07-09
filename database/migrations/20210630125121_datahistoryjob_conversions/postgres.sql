-- +goose Up
ALTER TABLE datahistoryjob
    ADD conversion_interval DOUBLE PRECISION,
    ADD overwrite_data boolean,
    ADD decimal_place_comparison INTEGER,
    ADD secondary_exchange_id uuid REFERENCES exchange(id),
    ADD issue_tolerance_percentage DOUBLE PRECISION,
    ADD replace_on_issue boolean;

-- +goose Down
ALTER TABLE datahistoryjob
    DROP conversion_interval,
    DROP overwrite_data,
    DROP decimal_place_comparison,
    DROP CONSTRAINT datahistoryjob_exchange_name_id_fkey,
    DROP secondary_exchange_id ,
    DROP issue_tolerance_percentage,
    DROP replace_on_issue;

