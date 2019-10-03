-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS client
(
    id                  uuid            NOT NULL DEFAULT uuid_generate_v4(),
    user_name           varchar(255)    NOT NULL UNIQUE,
    password            varchar(255)    NOT NULL,
    email               varchar(255)    NOT NULL,
    one_time_password   text,
    enabled             boolean         NOT NULL,
    updated_at          TIMESTAMP       NOT NULL DEFAULT now(),
    created_at          TIMESTAMP       NOT NULL DEFAULT now(),
    PRIMARY KEY (id)
);
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE client;
-- +goose StatementEnd