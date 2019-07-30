package postgresql

import "github.com/thrasher-/gocryptotrader/database"

func Setup() (err error) {
	err = createAuditTable()
	return
}

func createAuditTable() error {
	query := `
CREATE TABLE IF NOT EXISTS audit
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
