package candle

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/repository"
	"github.com/thrasher-corp/gocryptotrader/database/seed"
	"github.com/thrasher-corp/gocryptotrader/database/testhelpers"
	"github.com/thrasher-corp/goose"
	"github.com/thrasher-corp/sqlboiler/boil"
)

var (
	dbConn *database.Instance
	dbIsSeeded bool
)

func TestMain(m *testing.M) {
	var err error
	testhelpers.PostgresTestDatabase = testhelpers.GetConnectionDetails()
	testhelpers.TempDir, err = ioutil.TempDir("", "gct-temp")
	if err != nil {
		fmt.Printf("failed tand o create temp file: %v", err)
		os.Exit(1)
	}

	t := m.Run()

	err = os.RemoveAll(testhelpers.TempDir)
	if err != nil {
		fmt.Printf("Failed to remove temp db file: %v", err)
	}

	err = testhelpers.CloseDatabase(dbConn)
	if err != nil {
		fmt.Println(err)
	}

	os.Exit(t)
}

func TestSeries(t *testing.T) {
	boil.DebugMode = true
	boil.DebugWriter = os.Stdout

	start := time.Date(2019, 0, 0, 0, 0, 0, 0, time.UTC)
	end := time.Date(2019, 12, 31, 23, 59, 59, 59, time.UTC)
	ret, err := Series("Binance", "BTC", "USDT", "1h", start, end)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(ret)
}

func seedDB(db *sql.DB, migrationDir string) error {
	// path := filepath.Join("..", "..", "migrations")
	err := goose.Run("reset", db, repository.GetSQLDialect(), migrationDir, "")
	if err != nil {
		return err
	}

	err = goose.Run("up", db, repository.GetSQLDialect(), migrationDir, "")
	if err != nil {
		return err
	}

	return seed.Run()
}