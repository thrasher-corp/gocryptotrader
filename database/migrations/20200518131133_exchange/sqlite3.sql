-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE "exchange" (
    id	        text not null primary key,
    name    	text not null,
    unique(name) ON CONFLICT IGNORE
);

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE "exchange";