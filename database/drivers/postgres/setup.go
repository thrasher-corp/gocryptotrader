package postgres

import "github.com/thrasher-corp/gocryptotrader/database"

// Setup is any post connection steps to run such as migration etc
func Setup() (err error) {
	err = createAuditTable()
	return
}

func createAuditTable() error {
	query := `
CREATE TABLE IF NOT EXISTS audit_event
(
    id bigserial  PRIMARY KEY NOT NULL,
    Type       varchar(255)  NOT NULL,
    Identifier varchar(255)  NOT NULL,
    Message    text          NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);`
	_, err := database.Conn.SQL.Exec(query)
	if err != nil {
		return err
	}

	return nil
}
