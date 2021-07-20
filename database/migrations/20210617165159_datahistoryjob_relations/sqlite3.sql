-- +goose Up
CREATE TABLE datahistoryjobrelations
(
    prerequisite_job_id text not null,
    job_id text not null,
    PRIMARY KEY (prerequisite_job_id, job_id),
    FOREIGN KEY (prerequisite_job_id) REFERENCES datahistoryjob(id) ON DELETE RESTRICT,
    FOREIGN KEY (job_id) REFERENCES datahistoryjob(id)  ON DELETE RESTRICT
);
-- +goose Down
DROP TABLE datahistoryjobrelations;
