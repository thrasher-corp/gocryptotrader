-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE "script" (
    id	        text not null primary key,
    script_id   text not null,
    script_name text not null,
    script_path text not NULL,
    script_data blob null,
    last_executed_at timestamp not null default CURRENT_TIMESTAMP,
    created_at   timestamp not null default CURRENT_TIMESTAMP
);
-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE "script";
