-- +goose Up
ALTER TABLE candle
    ADD related_job_id TEXT REFERENCES datahistoryjob(id);
ALTER TABLE candle
    ADD validation_issues TEXT;
-- +goose Down
ALTER TABLE candle
    DROP related_job_id;
ALTER TABLE candle
    DROP validation_issues;

