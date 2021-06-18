-- +goose Up
CREATE TABLE datahistoryjobrelations
(
    id INTEGER primary key,
    prerequisite_job_id text not null,
    following_job_id text not null,
    FOREIGN KEY(prerequisite_job_id) REFERENCES datahistoryjob(id) ON DELETE RESTRICT,
    FOREIGN KEY(following_job_id) REFERENCES datahistoryjob(id) ON DELETE RESTRICT
);
-- +goose Down
DROP TABLE datahistoryjobrelations;
