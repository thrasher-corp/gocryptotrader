package tests

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/drivers"
	"github.com/thrasher-corp/gocryptotrader/database/repository"
	"github.com/thrasher-corp/gocryptotrader/database/repository/audit"
	"github.com/thrasher-corp/goose"
)

func TestAudit(t *testing.T) {
	testCases := []struct {
		name   string
		config database.Config
		runner func(t *testing.T)
		closer func(t *testing.T, dbConn *database.Db) error
		output interface{}
	}{
		{
			"SQLite",
			database.Config{
				Driver:            "sqlite",
				ConnectionDetails: drivers.ConnectionDetails{Database: "./testdb"},
			},

			writeAudit,
			closeDatabase,
			nil,
		},
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
			path := filepath.Join("..", "migrations")
			err = goose.Run("up", dbConn.SQL, repository.GetSQLDialect(), path, "")
			if err != nil {
				t.Fatalf("failed to run migrations %v", err)
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
