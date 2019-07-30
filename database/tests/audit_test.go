package tests

import (
	"github.com/thrasher-/gocryptotrader/database"
	db "github.com/thrasher-/gocryptotrader/database/drivers/sqlite"
	"github.com/thrasher-/gocryptotrader/database/repository/audit"
	auditSQlite "github.com/thrasher-/gocryptotrader/database/repository/audit/sqlite"
	"path"
	"testing"
)

func TestAudit(t *testing.T) {
	testConfig := database.Config{}
	testConfig.Database = path.Join(tempDir, "./auditdb.db")

	database.Conn.Config = &testConfig
	dbConn, err := db.Connect()

	if err != nil {
		t.Fatal(err)
	}

	err = db.Setup()
	if err != nil {
		t.Fatal(err)
	}

	audit.Audit = auditSQlite.Audit()

	err = dbConn.SQL.Close()
	if err != nil {
		t.Error("Failed to close database")
	}
}

func TestAuditEvent(t *testing.T) {
	audit.Event("nil", "nil", "nil")
}