package candle

import (
	"os"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/drivers"
	"github.com/thrasher-corp/gocryptotrader/database/testhelpers"
	"github.com/thrasher-corp/sqlboiler/boil"
)

var (
	dbcfg = &database.Config{
		Enabled: true,
		Driver:  "postgres",
		Verbose: true,
		ConnectionDetails: drivers.ConnectionDetails{
			Host:     "localhost",
			Port:     5432,
			Username: "",
			Password: "",
			Database: "gct_dev",
			SSLMode:  "disable",
		},
	}
)

func TestSeries(t *testing.T) {
	_, err := testhelpers.ConnectToDatabase(dbcfg)
	if err != nil {
		t.Fatal(err)
	}

	boil.DebugMode = true
	boil.DebugWriter = os.Stdout

	ret, err := Series("Binance", "BTC", "USDT", "24h", time.Now(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	t.Log(ret)
}
