-- +goose Up
ALTER TABLE candle
    ADD source_job_id uuid references datahistoryjob(id),
    ADD validation_job_id uuid REFERENCES datahistoryjob(id),
    ADD validation_issues TEXT;
-- +goose Down
ALTER TABLE candle
    DROP validation_issues,
    DROP CONSTRAINT candle_validation_job_id_fkey,
    DROP validation_job_id,
    DROP CONSTRAINT candle_source_job_id_fkey,
    DROP source_job_id;
