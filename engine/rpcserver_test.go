package engine

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/goose"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/drivers"
	"github.com/thrasher-corp/gocryptotrader/database/repository"
	"github.com/thrasher-corp/gocryptotrader/database/repository/exchange"
	sqltrade "github.com/thrasher-corp/gocryptotrader/database/repository/trade"
	"github.com/thrasher-corp/gocryptotrader/database/testhelpers"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
)

// Sets up everything required to run any function inside rpcserver
func RPCTestSetup(t *testing.T) {
	SetupTestHelpers(t)
	dbConf := database.Config{
		Enabled: true,
		Driver:  database.DBSQLite3,
		ConnectionDetails: drivers.ConnectionDetails{
			Database: "rpctestdb",
		},
	}
	Bot.Config.Database = dbConf
	database.DB.Config = &dbConf
	err := Bot.DatabaseManager.Start()
	if err != nil {
		log.Fatal(err)
	}
	path := filepath.Join("..", "database", "migrations")
	err = goose.Run("up", dbConn.SQL, repository.GetSQLDialect(), path, "")
	if err != nil {
		t.Fatalf("failed to run migrations %v", err)
	}
	uuider, _ := uuid.NewV4()
	testhelpers.EnableVerboseTestOutput()
	err = exchange.Insert(exchange.Details{Name: testExchange, UUID: uuider})
	if err != nil {
		t.Fatalf("failed to insert exchange %v", err)
	}
}

func CleanRPCTest(t *testing.T) {
	err := Bot.DatabaseManager.Stop()
	if err != nil {
		t.Fatal(err)
	}
	err = os.Remove(filepath.Join(common.GetDefaultDataDir(runtime.GOOS), "database", "rpctestdb"))
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetSavedTrades(t *testing.T) {
	RPCTestSetup(t)
	defer CleanRPCTest(t)
	var s RPCServer
	_, err := s.GetSavedTrades(context.Background(), &gctrpc.GetSavedTradesRequest{
		Exchange:  "",
		Pair:      nil,
		AssetType: "",
		Start:     0,
		End:       0,
	})
	if err == nil {
		t.Fatal("unexpected lack of error")
	}
	if err.Error() != "invalid arguments received" {
		t.Error(err)
	}
	_, err = s.GetSavedTrades(context.Background(), &gctrpc.GetSavedTradesRequest{
		Exchange: "fake",
		Pair: &gctrpc.CurrencyPair{
			Delimiter: currency.DashDelimiter,
			Base:      currency.BTC.String(),
			Quote:     currency.USD.String(),
		},
		AssetType: asset.Spot.String(),
		Start:     time.Date(2020, 0, 0, 0, 0, 0, 0, time.UTC).Unix(),
		End:       time.Date(2020, 1, 1, 1, 1, 1, 1, time.UTC).Unix(),
	})
	if err == nil {
		t.Fatal("unexpected lack of error")
	}
	if err != errExchangeNotLoaded {
		t.Error(err)
	}
	_, err = s.GetSavedTrades(context.Background(), &gctrpc.GetSavedTradesRequest{
		Exchange: testExchange,
		Pair: &gctrpc.CurrencyPair{
			Delimiter: currency.DashDelimiter,
			Base:      currency.BTC.String(),
			Quote:     currency.USD.String(),
		},
		AssetType: asset.Spot.String(),
		Start:     time.Date(2020, 0, 0, 0, 0, 0, 0, time.UTC).Unix(),
		End:       time.Date(2020, 1, 1, 1, 1, 1, 1, time.UTC).Unix(),
	})
	if err == nil {
		t.Fatal("unexpected lack of error")
	}
	if err.Error() != "request for Bitstamp spot trade data between 1575072000 and 1577840461 and returned no results" {
		t.Error(err)
	}
	err = sqltrade.Insert(sqltrade.Data{
		Timestamp: time.Date(2020, 0, 0, 0, 0, 1, 0, time.UTC).Unix(),
		Exchange:  testExchange,
		Base:      currency.BTC.String(),
		Quote:     currency.USD.String(),
		AssetType: asset.Spot.String(),
		Price:     1337,
		Amount:    1337,
		Side:      order.Buy.String(),
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.GetSavedTrades(context.Background(), &gctrpc.GetSavedTradesRequest{
		Exchange: testExchange,
		Pair: &gctrpc.CurrencyPair{
			Delimiter: currency.DashDelimiter,
			Base:      currency.BTC.String(),
			Quote:     currency.USD.String(),
		},
		AssetType: asset.Spot.String(),
		Start:     time.Date(2020, 0, 0, 0, 0, 0, 0, time.UTC).Unix(),
		End:       time.Date(2020, 1, 1, 1, 1, 1, 1, time.UTC).Unix(),
	})
	if err != nil {
		t.Error(err)
	}
}

func TestConvertTradesToCandles(t *testing.T) {
	RPCTestSetup(t)
	defer CleanRPCTest(t)
	var s RPCServer
	_, err := s.ConvertTradesToCandles(context.Background(), &gctrpc.ConvertTradesToCandlesRequest{
		Exchange:     "",
		Pair:         nil,
		AssetType:    "",
		Start:        0,
		End:          0,
		TimeInterval: 0,
	})
	if err == nil {
		t.Fatal("unexpected lack of error")
	}
	if err.Error() != "invalid arguments received" {
		t.Error(err)
	}

	_, err = s.ConvertTradesToCandles(context.Background(), &gctrpc.ConvertTradesToCandlesRequest{
		Exchange: "fake",
		Pair: &gctrpc.CurrencyPair{
			Delimiter: currency.DashDelimiter,
			Base:      currency.BTC.String(),
			Quote:     currency.USD.String(),
		},
		AssetType:    asset.Spot.String(),
		Start:        time.Date(2020, 0, 0, 0, 0, 0, 0, time.UTC).Unix(),
		End:          time.Date(2020, 1, 1, 1, 1, 1, 1, time.UTC).Unix(),
		TimeInterval: int64(kline.OneHour.Duration()),
	})
	if err == nil {
		t.Fatal("unexpected lack of error")
	}
	if err != errExchangeNotLoaded {
		t.Error(err)
	}

	_, err = s.ConvertTradesToCandles(context.Background(), &gctrpc.ConvertTradesToCandlesRequest{
		Exchange: testExchange,
		Pair: &gctrpc.CurrencyPair{
			Delimiter: currency.DashDelimiter,
			Base:      currency.BTC.String(),
			Quote:     currency.USD.String(),
		},
		AssetType:    asset.Spot.String(),
		Start:        time.Date(2020, 1, 1, 1, 1, 1, 1, time.UTC).Unix(),
		End:          time.Date(2020, 2, 2, 2, 2, 2, 2, time.UTC).Unix(),
		TimeInterval: int64(kline.OneHour.Duration()),
	})
	if err == nil {
		t.Fatal("unexpected lack of error")
	}
	if err.Error() != "no trades returned from supplied params" {
		t.Error(err)
	}

	err = sqltrade.Insert(sqltrade.Data{
		Timestamp: time.Date(2020, 1, 1, 1, 1, 2, 1, time.UTC).Unix(),
		Exchange:  testExchange,
		Base:      currency.BTC.String(),
		Quote:     currency.USD.String(),
		AssetType: asset.Spot.String(),
		Price:     1337,
		Amount:    1337,
		Side:      order.Buy.String(),
	})
	if err != nil {
		t.Fatal(err)
	}
	var candles *gctrpc.GetHistoricCandlesResponse
	candles, err = s.ConvertTradesToCandles(context.Background(), &gctrpc.ConvertTradesToCandlesRequest{
		Exchange: testExchange,
		Pair: &gctrpc.CurrencyPair{
			Delimiter: currency.DashDelimiter,
			Base:      currency.BTC.String(),
			Quote:     currency.USD.String(),
		},
		AssetType:    asset.Spot.String(),
		Start:        time.Date(2020, 1, 1, 1, 1, 1, 1, time.UTC).Unix(),
		End:          time.Date(2020, 2, 2, 2, 2, 2, 2, time.UTC).Unix(),
		TimeInterval: int64(kline.OneHour.Duration()),
	})
	if err != nil {
		t.Error(err)
	}
	if len(candles.Candle) == 0 {
		t.Error("no candles returned")
	}
}
