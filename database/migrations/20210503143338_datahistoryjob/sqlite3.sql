-- +goose Up
CREATE TABLE datahistoryjob
(
    id text NOT NULL primary key,
    nickname text NOT NULL,
    exchange_name_id text NOT NULL,
    asset text NOT NULL,
    base text NOT NULL,
    quote text NOT NULL,
    start_time timestamp NOT NULL,
    end_time timestamp NOT NULL,
    interval real NOT NULL,
    data_type real NOT NULL,
    request_size real NOT NULL,
    max_retries real NOT NULL,
    batch_count real NOT NULL,
    status real NOT NULL,
    created timestamp NOT NULL default CURRENT_TIMESTAMP,
    FOREIGN KEY(exchange_name_id) REFERENCES exchange(id) ON DELETE RESTRICT,
    UNIQUE(id) ON CONFLICT REPLACE,
    UNIQUE(nickname) ON CONFLICT REPLACE
);

CREATE TABLE datahistoryjobresult
(
    id text not null primary key,
    job_id text NOT NULL,
    result text NULL,
    status real NOT NULL,
    interval_start_time timestamp NOT NULL,
    interval_end_time timestamp NOT NULL,
    run_time timestamp NOT NULL default CURRENT_TIMESTAMP,
    UNIQUE(id) ON CONFLICT REPLACE,
    FOREIGN KEY(job_id) REFERENCES datahistoryjob(id) ON DELETE RESTRICT
);

-- +goose Down
DROP TABLE datahistoryjob;
DROP TABLE datahistoryjobresult;

