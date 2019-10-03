-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS client
(
    id                  uuid            not null primary key,
    user_name           varchar(255)    not null UNIQUE,
    password            varchar(255)    not null,
    email               varchar(255)    not null,
    one_time_password   text,
    enabled             boolean         not null,
    updated_at          timestamp       not null default CURRENT_TIMESTAMP,
    created_at          timestamp       not null default CURRENT_TIMESTAMP
);
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE client;
-- +goose StatementEnd

