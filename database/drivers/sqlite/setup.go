package sqlite

import (
	"github.com/thrasher-/gocryptotrader/database"
)

// Setup is any post connection steps to run such as migration etc
func Setup() (err error) {
	err = createAuditTable()
	return
}

func createAuditTable() error {
	query := `
CREATE TABLE IF NOT EXISTS audit_event
(
    id INTEGER PRIMARY KEY,
    Type       varchar(255),
    Identifier varchar(255),
    Message    text,
    created_at timestamp default CURRENT_TIMESTAMP   
);`
	_, err := database.Conn.SQL.Exec(query)
	if err != nil {
		return err
	}

	return nil
}
