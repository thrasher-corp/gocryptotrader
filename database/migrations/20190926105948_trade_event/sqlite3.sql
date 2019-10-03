-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS trade_event
(
    id                  bigint          not null primary key,
    trade_id            varchar(255)    not null,
    order_id            bigint          not null,
    exchange            varchar(255)    not null,
    base_currency       varchar(255)    not null,
    quote_currency      varchar(255)    not null,
    side                varchar(255)    not null,
    volume              real            not null,
    price               real            not null,
    fee                 real            not null,
    tax                 real            not null,
    executed_at         timestamp       not null,
    updated_at          timestamp       not null,
    created_at          timestamp       not null DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(order_id) REFERENCES order_event(id)
);
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE trade_event;
-- +goose StatementEnd
