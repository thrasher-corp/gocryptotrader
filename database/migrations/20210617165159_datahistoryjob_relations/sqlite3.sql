-- +goose Up
CREATE TABLE datahistoryjobrelations
(
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    prerequisite_job_id text not null,
    following_job_id text not null,
    FOREIGN KEY(prerequisite_job_id) REFERENCES datahistoryjob(id),
    FOREIGN KEY(following_job_id) REFERENCES datahistoryjob(id)
);
-- +goose Down
DROP TABLE datahistoryjobrelations;
