-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS trade_event
(
    id                  bigserial           PRIMARY KEY NOT NULL,
    trade_id            varchar(255)        NOT NULL,
    order_id            bigserial           NOT NULL,
    exchange            varchar(255)        NOT NULL,
    base_currency       varchar(255)        NOT NULL,
    quote_currency      varchar(255)        NOT NULL,
    side                varchar(255)        NOT NULL,
    volume              DOUBLE PRECISION    NOT NULL,
    price               DOUBLE PRECISION    NOT NULL,
    fee                 DOUBLE PRECISION    NOT NULL,
    tax                 DOUBLE PRECISION    NOT NULL,
    executed_at         TIMESTAMP           NOT NULL,
    updated_at          TIMESTAMP           NOT NULL,
    created_at          TIMESTAMP           NOT NULL DEFAULT now(),
    FOREIGN KEY(order_id) REFERENCES order_event(id)
);
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE trade_event;
-- +goose StatementEnd
