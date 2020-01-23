package tests

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/drivers"
	"github.com/thrasher-corp/gocryptotrader/database/repository"
	"github.com/thrasher-corp/gocryptotrader/database/repository/script"
	"github.com/thrasher-corp/goose"
	"github.com/volatiletech/null"
)

func TestScript(t *testing.T) {
	testCases := []struct {
		name   string
		config *database.Config
		runner func()
		closer func(t *testing.T, dbConn *database.Db) error
		output interface{}
	}{
		{
			"SQLite-Write",
			&database.Config{
				Driver:            database.DBSQLite3,
				ConnectionDetails: drivers.ConnectionDetails{Database: "./testdb"},
			},

			writeScript,
			closeDatabase,
			nil,
		},
		{
			"Postgres-Write",
			postgresTestDatabase,
			writeScript,
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

			dbConn, err := connectToDatabase(t, test.config)

			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join("..", "migrations")
			err = goose.Run("up", dbConn.SQL, repository.GetSQLDialect(), path, "")
			if err != nil {
				t.Fatalf("failed to run migrations %v", err)
			}

			if test.runner != nil {
				test.runner()
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

func writeScript() {
	var wg sync.WaitGroup
	for x := 0; x < 20; x++ {
		wg.Add(1)

		go func(x int) {
			defer wg.Done()
			test := fmt.Sprintf("test-%v", x)
			var data null.Bytes
			script.Event(test, test, test, data, test, test, time.Now())
		}(x)
	}
	wg.Wait()
}
