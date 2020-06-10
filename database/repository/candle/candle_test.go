package candle

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/database/repository"
	"github.com/thrasher-corp/gocryptotrader/database/seed"
	"github.com/thrasher-corp/gocryptotrader/database/testhelpers"
	"github.com/thrasher-corp/goose"
	"github.com/thrasher-corp/sqlboiler/boil"
)

func TestMain(m *testing.M) {
	var err error
	testhelpers.PostgresTestDatabase = testhelpers.GetConnectionDetails()
	testhelpers.TempDir, err = ioutil.TempDir("", "gct-temp")
	if err != nil {
		fmt.Printf("failed to create temp file: %v", err)
		os.Exit(1)
	}

	dbConn, err := testhelpers.ConnectToDatabase(testhelpers.PostgresTestDatabase)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}

	path := filepath.Join("..", "..", "migrations")
	err = goose.Run("reset", dbConn.SQL, repository.GetSQLDialect(), path, "")
	if err != nil {
		fmt.Printf("failed to reset database %v", err)
		os.Exit(2)
	}

	err = goose.Run("up", dbConn.SQL, repository.GetSQLDialect(), path, "")
	if err != nil {
		fmt.Printf("failed to run migrations %v", err)
		os.Exit(2)
	}

	err = seed.Run()
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
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
