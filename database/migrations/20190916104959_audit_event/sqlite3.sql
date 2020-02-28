-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE "audit_event" (
    id	        integer not null primary key,
    type    	text not null,
    identifier	text not null,
    message	    text not null,
    created_at  timestamp not null default CURRENT_TIMESTAMP
);
-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE audit_event;