package migrations

import (
	"github.com/thrasher-corp/gocryptotrader/database"
)

type Migration struct {
	Sequence int
	Name     string
	UpSQL    string
	DownSQL  string
}

type Migrator struct {
	Conn       *database.Database
	Migrations []Migration
	Log        Logger
}

type Logger interface {
	Printf(format string, v ...interface{})
	Println(v ...interface{})
	Errorf(format string, v ...interface{})
}


var defaultAuditMigration = []byte(`-- up
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
`)

