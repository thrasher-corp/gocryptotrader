-- +goose Up
CREATE TABLE datahistoryjobqueue
(
    prerequisite_job_id text not null ,
    following_job_id text not null,
    PRIMARY KEY (prerequisite_job_id, following_job_id),
    FOREIGN KEY (prerequisite_job_id) REFERENCES datahistoryjob(id) ON DELETE RESTRICT ,
    FOREIGN KEY (following_job_id) REFERENCES datahistoryjob(id)  ON DELETE RESTRICT
);
-- +goose Down
DROP TABLE datahistoryjobqueue;
