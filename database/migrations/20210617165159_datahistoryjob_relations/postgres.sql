-- +goose Up
CREATE TABLE datahistoryjobrelations
(
    id SERIAL PRIMARY KEY,
    prerequisite_job_id uuid not null REFERENCES datahistoryjob(id) not null,
    following_job_id uuid not null REFERENCES datahistoryjob(id) not null
);
-- +goose Down
DROP TABLE datahistoryjobrelations;
