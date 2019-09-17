package tests

import (
	"fmt"
	"sync"
	"testing"

	"github.com/xtda/goose"

	"github.com/thrasher-corp/gocryptotrader/database"

	"github.com/thrasher-corp/gocryptotrader/database/repository/audit"
)

func TestAudit(t *testing.T) {
	testCases := []struct {
		name   string
		config database.Config
		runner func(t *testing.T)
		closer func(t *testing.T, dbConn *database.Db) error
		output interface{}
	}{
		//{
		//	"SQLite",
		//	database.Config{
		//		Driver:            "sqlite",
		//		ConnectionDetails: drivers.ConnectionDetails{Database: path.Join(tempDir, "./testdb.db")},
		//	},
		//
		//	writeAudit,
		//	closeDatabase,
		//	nil,
		//},
		{
			"Postgres",
			postgresTestDatabase,
			writeAudit,
			nil,
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

			if err != nil {
				t.Fatal(err)
			}

			err = goose.Run("up", dbConn.SQL, "../migrations", "")
			if err != nil {
				t.Fatal("failed to run migrations")
			}

			if test.runner != nil {
				test.runner(t)
			}

			if test.closer != nil {
				err = test.closer(t, dbConn)
				if err != nil {
					t.Log(err)
				}
			}
		})
	}
}

func writeAudit(t *testing.T) {
	t.Helper()
	var wg sync.WaitGroup

	for x := 0; x < 200; x++ {
		wg.Add(1)

		go func(x int) {
			defer wg.Done()
			test := fmt.Sprintf("test-%v", x)
			audit.Event(test, test, test)
		}(x)
	}

	wg.Wait()
}
