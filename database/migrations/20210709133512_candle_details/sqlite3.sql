-- +goose Up
ALTER TABLE candle
    ADD source_job_id TEXT REFERENCES datahistoryjob(id);
ALTER TABLE candle
    ADD validation_job_id TEXT REFERENCES datahistoryjob(id);
ALTER TABLE candle
    ADD validation_issues TEXT;
-- +goose Down
ALTER TABLE candle
    DROP validation_issues;
ALTER TABLE candle
    DROP validation_job_id;
ALTER TABLE candle
    DROP source_job_id;

