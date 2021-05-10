-- +goose Up
CREATE TABLE IF NOT EXISTS datahistoryjob
(
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    nickname varchar(255) NOT NULL,
    exchange_name_id uuid REFERENCES exchange(id) NOT NULL,
    asset varchar NOT NULL,
    base varchar(30) NOT NULL,
    quote varchar(30) NOT NULL,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    data_type DOUBLE PRECISION NOT NULL,
    interval DOUBLE PRECISION NOT NULL,
    request_size DOUBLE PRECISION NOT NULL,
    max_retries DOUBLE PRECISION NOT NULL,
    batch_count DOUBLE PRECISION NOT NULL,
    status DOUBLE PRECISION NOT NULL,
    created TIMESTAMPTZ NOT NULL,

    CONSTRAINT uniquenickname
        unique(nickname),
    CONSTRAINT uniquejob
        unique(exchange_name_id, asset, base, quote, start_time, end_time, interval, data_type)
);

CREATE TABLE IF NOT EXISTS datahistoryjobresult
(
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id uuid REFERENCES datahistoryjob(id) NOT NULL,
    result TEXT NULL,
    status DOUBLE PRECISION NOT NULL,
    interval_start_time TIMESTAMPTZ NOT NULL,
    interval_end_time TIMESTAMPTZ NOT NULL,
    run_time TIMESTAMPTZ NOT NULL,
        CONSTRAINT uniquejobtimestamp
            unique(job_id, interval_start_time, interval_end_time)
);
-- +goose Down
DROP TABLE datahistoryjobresult;
DROP TABLE datahistoryjob;


