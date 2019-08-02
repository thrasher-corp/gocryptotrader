package tests

import (
	"fmt"
	"path"
	"sync"
	"testing"

	"github.com/thrasher-/gocryptotrader/database"
	"github.com/thrasher-/gocryptotrader/database/drivers"
	dbpsql "github.com/thrasher-/gocryptotrader/database/drivers/postgres"
	dbsqlite "github.com/thrasher-/gocryptotrader/database/drivers/sqlite"
	"github.com/thrasher-/gocryptotrader/database/repository/audit"
	auditPSQL "github.com/thrasher-/gocryptotrader/database/repository/audit/postgres"
	auditSQlite "github.com/thrasher-/gocryptotrader/database/repository/audit/sqlite"
)

func TestAudit(t *testing.T) {
	testCases := []struct {
		name   string
		config database.Config
		setup  func() error
		audit  audit.Repository
		runner func(t *testing.T)
		output interface{}
	}{
		{
			"SQLite",
			database.Config{
				Driver:            "sqlite",
				ConnectionDetails: drivers.ConnectionDetails{Database: path.Join(tempDir, "./testdb.db")},
			},
			dbsqlite.Setup,
			auditSQlite.Audit(),
			writeAudit,
			nil,
		},
		{
			"Postgres",
			postgresTestDatabase,
			dbpsql.Setup,
			auditPSQL.Audit(),
			writeAudit,
			nil,
		},
	}

	for _, tests := range testCases {
		test := tests

		t.Run(test.name, func(t *testing.T) {

			if !checkValidConfig(t, &test.config.ConnectionDetails) {
				t.Skip("database not configured skipping test")
			}

			dbConn, err := connectToDatabase(t, &test.config)

			if test.setup != nil {
				err = test.setup()
				if err != nil {
					t.Fatal(err)
				}
			}

			if test.audit != nil {
				audit.Audit = test.audit
			}

			if test.runner != nil {
				test.runner(t)
			}

			switch v := test.output.(type) {

			case error:
				if v.Error() != test.output.(error).Error() {
					t.Fatal(err)
				}
				return
			default:
				break
			}

			err = closeDatabase(t, dbConn)
			if err != nil {
				t.Error("Failed to close database")
			}
		})
	}
}

func writeAudit(t *testing.T) {
	t.Helper()
	var wg sync.WaitGroup

	for x := 0; x < 20; x++ {
		wg.Add(1)

		go func(x int) {
			defer wg.Done()
			test := fmt.Sprintf("test-%v", x)
			audit.Event(test, test, test)
		}(x)
	}

	wg.Wait()
}
