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
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
)

var databaseFolder = "database"
var migrationsFolder = "migrations"
var databaseName = "rpctestdb"

// Sets up everything required to run any function inside rpcserver
func RPCTestSetup(t *testing.T) {
	SetupTestHelpers(t)
	dbConf := database.Config{
		Enabled: true,
		Driver:  database.DBSQLite3,
		ConnectionDetails: drivers.ConnectionDetails{
			Database: databaseName,
		},
		Verbose: true,
	}
	Bot.Config.Database = dbConf
	database.DB.Config = &dbConf
	err := Bot.DatabaseManager.Start()
	if err != nil {
		log.Fatal(err)
	}
	path := filepath.Join("..", databaseFolder, migrationsFolder)
	err = goose.Run("up", dbConn.SQL, repository.GetSQLDialect(), path, "")
	if err != nil {
		t.Fatalf("failed to run migrations %v", err)
	}
	uuider, _ := uuid.NewV4()
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
	err = os.Remove(filepath.Join(common.GetDefaultDataDir(runtime.GOOS), databaseFolder, databaseName))
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetSavedTrades(t *testing.T) {
	RPCTestSetup(t)
	defer CleanRPCTest(t)
	var s RPCServer
	_, err := s.GetSavedTrades(context.Background(), &gctrpc.GetSavedTradesRequest{})
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
		Timestamp: time.Date(2020, 0, 0, 0, 0, 1, 0, time.UTC),
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
	// bad param test
	_, err := s.ConvertTradesToCandles(context.Background(), &gctrpc.ConvertTradesToCandlesRequest{})
	if err == nil {
		t.Fatal("unexpected lack of error")
	}
	if err.Error() != "invalid arguments received" {
		t.Error(err)
	}

	// bad exchange test
	_, err = s.ConvertTradesToCandles(context.Background(), &gctrpc.ConvertTradesToCandlesRequest{
		Exchange: "faker",
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

	// no trades test
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

	// add a trade
	err = sqltrade.Insert(sqltrade.Data{
		Timestamp: time.Date(2020, 1, 1, 1, 1, 2, 1, time.UTC),
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

	// get candle from one trade
	var candles *gctrpc.GetHistoricCandlesResponse
	candles, err = s.ConvertTradesToCandles(context.Background(), &gctrpc.ConvertTradesToCandlesRequest{
		Exchange: testExchange,
		Pair: &gctrpc.CurrencyPair{
			Delimiter: currency.DashDelimiter,
			Base:      currency.BTC.String(),
			Quote:     currency.USD.String(),
		},
		AssetType:    asset.Spot.String(),
		Start:        time.Date(2020, 1, 1, 1, 0, 0, 0, time.UTC).Unix(),
		End:          time.Date(2020, 2, 2, 2, 2, 2, 2, time.UTC).Unix(),
		TimeInterval: int64(kline.OneHour.Duration()),
	})
	if err != nil {
		t.Error(err)
	}
	if len(candles.Candle) == 0 {
		t.Error("no candles returned")
	}

	// save generated candle to database
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
		Sync:         true,
	})
	if err != nil {
		t.Error(err)
	}

	// forcefully remove previous candle and insert a new one
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
		Sync:         true,
		Force:        true,
	})
	if err != nil {
		t.Error(err)
	}

	// load the saved candle to verify that it was overwritten
	getSavedCandles, err := s.GetHistoricCandles(context.Background(), &gctrpc.GetHistoricCandlesRequest{
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
		UseDb:        true,
	})
	if err != nil {
		t.Error(err)
	}

	if len(getSavedCandles.Candle) != 1 {
		t.Error("expected only one candle")
	}
}

func TestGetHistoricCandles(t *testing.T) {
	RPCTestSetup(t)
	defer CleanRPCTest(t)
	var s RPCServer
	// error checks
	_, err := s.GetHistoricCandles(context.Background(), &gctrpc.GetHistoricCandlesRequest{
		Exchange: "",
	})
	if err != nil && err.Error() != errExchangeNameUnset {
		t.Error(err)
	}

	_, err = s.GetHistoricCandles(context.Background(), &gctrpc.GetHistoricCandlesRequest{
		Exchange: testExchange,
		Pair:     &gctrpc.CurrencyPair{},
	})
	if err != nil && err.Error() != errCurrencyPairUnset {
		t.Error(err)
	}
	_, err = s.GetHistoricCandles(context.Background(), &gctrpc.GetHistoricCandlesRequest{
		Exchange: testExchange,
		Pair: &gctrpc.CurrencyPair{
			Base:  currency.BTC.String(),
			Quote: currency.USD.String(),
		},
	})
	if err != nil && err.Error() != errStartEndTimesUnset {
		t.Error(err)
	}
	var results *gctrpc.GetHistoricCandlesResponse
	defaultStart := time.Date(2020, 1, 1, 1, 1, 1, 1, time.UTC).Unix()
	defaultEnd := time.Date(2020, 1, 2, 2, 2, 2, 2, time.UTC).Unix()
	cp := currency.NewPair(currency.BTC, currency.USD)
	// default run
	results, err = s.GetHistoricCandles(context.Background(), &gctrpc.GetHistoricCandlesRequest{
		Exchange: testExchange,
		Pair: &gctrpc.CurrencyPair{
			Base:  cp.Base.String(),
			Quote: cp.Quote.String(),
		},
		Start:        defaultStart,
		End:          defaultEnd,
		AssetType:    asset.Spot.String(),
		TimeInterval: int64(kline.OneHour.Duration()),
	})
	if err != nil {
		t.Error(err)
	}
	if len(results.Candle) == 0 {
		t.Error("expected results")
	}

	// sync run
	results, err = s.GetHistoricCandles(context.Background(), &gctrpc.GetHistoricCandlesRequest{
		Exchange: testExchange,
		Pair: &gctrpc.CurrencyPair{
			Base:  cp.Base.String(),
			Quote: cp.Quote.String(),
		},
		AssetType:    asset.Spot.String(),
		Start:        defaultStart,
		End:          defaultEnd,
		TimeInterval: int64(kline.OneHour.Duration()),
		Sync:         true,
		ExRequest:    true,
	})
	if err != nil {
		t.Error(err)
	}
	if len(results.Candle) == 0 {
		t.Error("expected results")
	}

	// db run
	results, err = s.GetHistoricCandles(context.Background(), &gctrpc.GetHistoricCandlesRequest{
		Exchange: testExchange,
		Pair: &gctrpc.CurrencyPair{
			Base:  cp.Base.String(),
			Quote: cp.Quote.String(),
		},
		AssetType:    asset.Spot.String(),
		Start:        defaultStart,
		End:          defaultEnd,
		TimeInterval: int64(kline.OneHour.Duration()),
		UseDb:        true,
	})
	if err != nil {
		t.Error(err)
	}
	if len(results.Candle) == 0 {
		t.Error("expected results")
	}
	err = trade.SaveTradesToDatabase(trade.Data{
		TID:          "test123",
		Exchange:     testExchange,
		CurrencyPair: cp,
		AssetType:    asset.Spot,
		Price:        1337,
		Amount:       1337,
		Side:         order.Buy,
		Timestamp:    time.Date(2020, 1, 2, 3, 1, 1, 7, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	// db run including trades
	results, err = s.GetHistoricCandles(context.Background(), &gctrpc.GetHistoricCandlesRequest{
		Exchange: testExchange,
		Pair: &gctrpc.CurrencyPair{
			Base:  cp.Base.String(),
			Quote: cp.Quote.String(),
		},
		AssetType:             asset.Spot.String(),
		Start:                 defaultStart,
		End:                   time.Date(2020, 1, 2, 4, 2, 2, 2, time.UTC).Unix(),
		TimeInterval:          int64(kline.OneHour.Duration()),
		UseDb:                 true,
		FillMissingWithTrades: true,
	})
	if err != nil {
		t.Error(err)
	}
	if results.Candle[len(results.Candle)-1].Close != 1337 {
		t.Error("expected fancy new candle based off fancy new trade data")
	}
}

func TestFindMissingSavedTradeIntervals(t *testing.T) {
	RPCTestSetup(t)
	defer CleanRPCTest(t)
	var s RPCServer
	// bad request checks
	_, err := s.FindMissingSavedTradeIntervals(context.Background(), &gctrpc.FindMissingTradePeriodsRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "invalid arguments received" {
		t.Fatal(err)
	}
	cp := currency.NewPair(currency.BTC, currency.USDT)
	// no data found response
	defaultStart := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
	defaultEnd := time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC).Unix()
	var resp *gctrpc.FindMissingIntervalsResponse
	resp, err = s.FindMissingSavedTradeIntervals(context.Background(), &gctrpc.FindMissingTradePeriodsRequest{
		ExchangeName: testExchange,
		AssetType:    asset.Spot.String(),
		Pair: &gctrpc.CurrencyPair{
			Base:  cp.Base.String(),
			Quote: cp.Quote.String(),
		},
		Start: defaultStart,
		End:   defaultEnd,
	})
	if err != nil {
		t.Error(err)
	}
	if resp.Status == "" {
		t.Errorf("expected a status message")
	}

	// one trade response
	err = trade.SaveTradesToDatabase(trade.Data{
		TID:          "test1234",
		Exchange:     testExchange,
		CurrencyPair: cp,
		AssetType:    asset.Spot,
		Price:        1337,
		Amount:       1337,
		Side:         order.Buy,
		Timestamp:    time.Date(2020, 1, 1, 2, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}

	resp, err = s.FindMissingSavedTradeIntervals(context.Background(), &gctrpc.FindMissingTradePeriodsRequest{
		ExchangeName: testExchange,
		AssetType:    asset.Spot.String(),
		Pair: &gctrpc.CurrencyPair{
			Base:  cp.Base.String(),
			Quote: cp.Quote.String(),
		},
		Start: defaultStart,
		End:   defaultEnd,
	})
	if err != nil {
		t.Error(err)
	}
	if len(resp.MissingPeriods) != 23 {
		t.Errorf("expected 23 missing periods, received: %v", len(resp.MissingPeriods))
	}
	if len(resp.FoundPeriods) != 1 {
		t.Errorf("expected 1 found periods, received: %v", len(resp.FoundPeriods))
	}

	// two trades response
	err = trade.SaveTradesToDatabase(trade.Data{
		TID:          "test123",
		Exchange:     testExchange,
		CurrencyPair: cp,
		AssetType:    asset.Spot,
		Price:        1337,
		Amount:       1337,
		Side:         order.Buy,
		Timestamp:    time.Date(2020, 1, 1, 4, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}

	resp, err = s.FindMissingSavedTradeIntervals(context.Background(), &gctrpc.FindMissingTradePeriodsRequest{
		ExchangeName: testExchange,
		AssetType:    asset.Spot.String(),
		Pair: &gctrpc.CurrencyPair{
			Base:  cp.Base.String(),
			Quote: cp.Quote.String(),
		},
		Start: defaultStart,
		End:   defaultEnd,
	})
	if err != nil {
		t.Error(err)
	}
	if len(resp.MissingPeriods) != 22 {
		t.Errorf("expected 22 missing periods, received: %v", len(resp.MissingPeriods))
	}
	if len(resp.FoundPeriods) != 2 {
		t.Errorf("expected 2 found periods, received: %v", len(resp.FoundPeriods))
	}
}

func TestFindMissingSavedCandleIntervals(t *testing.T) {
	RPCTestSetup(t)
	defer CleanRPCTest(t)
	var s RPCServer
	// bad request checks
	_, err := s.FindMissingSavedCandleIntervals(context.Background(), &gctrpc.FindMissingCandlePeriodsRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "invalid arguments received" {
		t.Fatal(err)
	}
	cp := currency.NewPair(currency.BTC, currency.USDT)
	// no data found response
	defaultStart := time.Date(2020, 1, 1, 1, 1, 1, 1, time.UTC).Unix()
	defaultEnd := time.Date(2020, 1, 2, 2, 2, 2, 2, time.UTC).Unix()
	var resp *gctrpc.FindMissingIntervalsResponse
	_, err = s.FindMissingSavedCandleIntervals(context.Background(), &gctrpc.FindMissingCandlePeriodsRequest{
		ExchangeName: testExchange,
		AssetType:    asset.Spot.String(),
		Pair: &gctrpc.CurrencyPair{
			Base:  cp.Base.String(),
			Quote: cp.Quote.String(),
		},
		Interval: int64(time.Hour),
		Start:    defaultStart,
		End:      defaultEnd,
	})
	if err != nil && err.Error() != "no candle data found: Bitstamp BTC USDT 3600 spot" {
		t.Fatal(err)
	}

	// one candle missing periods response
	_, err = kline.StoreInDatabase(&kline.Item{
		Exchange: testExchange,
		Pair:     cp,
		Asset:    asset.Spot,
		Interval: kline.OneHour,
		Candles: []kline.Candle{
			{
				Time:   time.Date(2020, 1, 1, 2, 1, 1, 1, time.UTC),
				Open:   1337,
				High:   1337,
				Low:    1337,
				Close:  1337,
				Volume: 1337,
			},
		},
	}, false)
	if err != nil {
		t.Fatal(err)
	}

	resp, err = s.FindMissingSavedCandleIntervals(context.Background(), &gctrpc.FindMissingCandlePeriodsRequest{
		ExchangeName: testExchange,
		AssetType:    asset.Spot.String(),
		Pair: &gctrpc.CurrencyPair{
			Base:  cp.Base.String(),
			Quote: cp.Quote.String(),
		},
		Interval: int64(time.Hour),
		Start:    defaultStart,
		End:      defaultEnd,
	})
	if err != nil {
		t.Error(err)
	}
	if len(resp.FoundPeriods) != 1 {
		t.Errorf("expected a candle")
	}

	// two candle missing periods response
	_, err = kline.StoreInDatabase(&kline.Item{
		Exchange: testExchange,
		Pair:     cp,
		Asset:    asset.Spot,
		Interval: kline.OneHour,
		Candles: []kline.Candle{
			{
				Time:   time.Date(2020, 1, 1, 3, 1, 1, 1, time.UTC),
				Open:   1337,
				High:   1337,
				Low:    1337,
				Close:  1337,
				Volume: 1337,
			},
		},
	}, false)
	if err != nil {
		t.Fatal(err)
	}

	resp, err = s.FindMissingSavedCandleIntervals(context.Background(), &gctrpc.FindMissingCandlePeriodsRequest{
		ExchangeName: testExchange,
		AssetType:    asset.Spot.String(),
		Pair: &gctrpc.CurrencyPair{
			Base:  cp.Base.String(),
			Quote: cp.Quote.String(),
		},
		Interval: int64(time.Hour),
		Start:    defaultStart,
		End:      defaultEnd,
	})
	if err != nil {
		t.Error(err)
	}
	if len(resp.MissingPeriods) != 23 {
		t.Errorf("expected 23 missing periods, received: %v", len(resp.MissingPeriods))
	}
}
