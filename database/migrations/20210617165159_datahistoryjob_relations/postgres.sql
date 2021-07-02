-- +goose Up
CREATE TABLE datahistoryjobrelations
(
    prerequisite_job_id uuid not null REFERENCES datahistoryjob(id),
    job_id uuid not null REFERENCES datahistoryjob(id),
    PRIMARY KEY (prerequisite_job_id, job_id)
);
-- +goose Down
DROP TABLE datahistoryjobrelations;
