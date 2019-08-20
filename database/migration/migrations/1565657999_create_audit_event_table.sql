-- up
CREATE TABLE IF NOT EXISTS audit_event
(
    id bigserial  PRIMARY KEY NOT NULL,
    Type       varchar(255)  NOT NULL,
    Identifier varchar(255)  NOT NULL,
    Message    text          NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);
-- down
DROP TABLE audit_event;
