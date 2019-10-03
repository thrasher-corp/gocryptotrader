-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS order_event
(
    id                  bigint          not null primary key,
    exchange_order_id   varchar(255)    not null,
    client_id           uuid            not null,
    exchange            varchar(255)    not null,
    currency_pair       varchar(255)    not null,
    asset_type          varchar(255)    not null,
    order_type          varchar(255)    not null,
    order_side          varchar(255)    not null,
    order_status        varchar(255)    not null,
    amount              real            not null,
    price               real            not null,
    updated_at          timestamp       not null,
    created_at          timestamp       not null default CURRENT_TIMESTAMP,
    FOREIGN KEY(client_id) REFERENCES client(id)
);
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE order_event;
-- +goose StatementEnd
