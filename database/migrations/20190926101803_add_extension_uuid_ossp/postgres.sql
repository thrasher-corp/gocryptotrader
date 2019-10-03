-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP EXTENSION "uuid-ossp";
-- +goose StatementEnd
