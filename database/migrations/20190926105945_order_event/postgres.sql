-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS order_event
(
    id                  bigserial           PRIMARY KEY NOT NULL,
    exchange_order_id   varchar(255)        NOT NULL,
    client_id           UUID                NOT NULL,
    exchange            varchar(255)        NOT NULL,
    currency_pair       varchar(255)        NOT NULL,
    asset_type          varchar(255)        NOT NULL,
    order_type          varchar(255)        NOT NULL,
    order_side          varchar(255)        NOT NULL,
    order_status        varchar(255)        NOT NULL,
    amount              DOUBLE PRECISION    NOT NULL,
    price               DOUBLE PRECISION    NOT NULL,
    updated_at          TIMESTAMP           NOT NULL,
    created_at          TIMESTAMP           NOT NULL DEFAULT now(),
    FOREIGN KEY(client_id) REFERENCES client(id)
);
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE order_event;
-- +goose StatementEnd
