package engine

import (
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/drivers"
	"github.com/thrasher-corp/gocryptotrader/database/repository"
	dbexchange "github.com/thrasher-corp/gocryptotrader/database/repository/exchange"
	sqltrade "github.com/thrasher-corp/gocryptotrader/database/repository/trade"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/binance"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"github.com/thrasher-corp/goose"
)

const (
	unexpectedLackOfError = "unexpected lack of error"
	migrationsFolder      = "migrations"
	databaseFolder        = "database"
	databaseName          = "rpctestdb"
)

// Sets up everything required to run any function inside rpcserver
func RPCTestSetup(t *testing.T) *Engine {
	database.DB.Mu.Lock()
	var err error
	dbConf := database.Config{
		Enabled: true,
		Driver:  database.DBSQLite3,
		ConnectionDetails: drivers.ConnectionDetails{
			Database: databaseName,
		},
	}
	engerino := new(Engine)
	engerino.Config = &config.Config{}
	err = engerino.Config.LoadConfig(config.TestFile, true)
	if err != nil {
		t.Fatalf("SetupTest: Failed to load config: %s", err)
	}

	if engerino.GetExchangeByName(testExchange) == nil {
		err = engerino.LoadExchange(testExchange, false, nil)
		if err != nil {
			t.Fatalf("SetupTest: Failed to load exchange: %s", err)
		}
	}
	engerino.Config.Database = dbConf
	err = engerino.DatabaseManager.Start(engerino)
	if err != nil {
		log.Fatal(err)
	}
	path := filepath.Join("..", databaseFolder, migrationsFolder)
	err = goose.Run("up", dbConn.SQL, repository.GetSQLDialect(), path, "")
	if err != nil {
		t.Fatalf("failed to run migrations %v", err)
	}
	uuider, _ := uuid.NewV4()
	err = dbexchange.Insert(dbexchange.Details{Name: testExchange, UUID: uuider})
	if err != nil {
		t.Fatalf("failed to insert exchange %v", err)
	}
	database.DB.Mu.Unlock()

	return engerino
}

func CleanRPCTest(t *testing.T, engerino *Engine) {
	database.DB.Mu.Lock()
	defer database.DB.Mu.Unlock()
	err := engerino.DatabaseManager.Stop()
	if err != nil {
		t.Error(err)
		return
	}
	err = os.Remove(filepath.Join(common.GetDefaultDataDir(runtime.GOOS), databaseFolder, databaseName))
	if err != nil {
		t.Error(err)
	}
}

func TestGetSavedTrades(t *testing.T) {
	engerino := RPCTestSetup(t)
	defer CleanRPCTest(t, engerino)
	s := RPCServer{Engine: engerino}
	_, err := s.GetSavedTrades(context.Background(), &gctrpc.GetSavedTradesRequest{})
	if err == nil {
		t.Fatal(unexpectedLackOfError)
	}
	if !errors.Is(err, errInvalidArguments) {
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
		Start:     time.Date(2020, 0, 0, 0, 0, 0, 0, time.UTC).Format(common.SimpleTimeFormat),
		End:       time.Date(2020, 1, 1, 1, 1, 1, 1, time.UTC).Format(common.SimpleTimeFormat),
	})
	if err == nil {
		t.Error(unexpectedLackOfError)
		return
	}
	if !errors.Is(err, errExchangeNotLoaded) {
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
		Start:     time.Date(2020, 0, 0, 0, 0, 0, 0, time.UTC).Format(common.SimpleTimeFormat),
		End:       time.Date(2020, 1, 1, 1, 1, 1, 1, time.UTC).Format(common.SimpleTimeFormat),
	})
	if err == nil {
		t.Error(unexpectedLackOfError)
		return
	}
	if err.Error() != "request for Bitstamp spot trade data between 2019-11-30 00:00:00 and 2020-01-01 01:01:01 and returned no results" {
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
		t.Error(err)
		return
	}
	_, err = s.GetSavedTrades(context.Background(), &gctrpc.GetSavedTradesRequest{
		Exchange: testExchange,
		Pair: &gctrpc.CurrencyPair{
			Delimiter: currency.DashDelimiter,
			Base:      currency.BTC.String(),
			Quote:     currency.USD.String(),
		},
		AssetType: asset.Spot.String(),
		Start:     time.Date(2020, 0, 0, 0, 0, 0, 0, time.UTC).Format(common.SimpleTimeFormat),
		End:       time.Date(2020, 1, 1, 1, 1, 1, 1, time.UTC).Format(common.SimpleTimeFormat),
	})
	if err != nil {
		t.Error(err)
	}
}

func TestConvertTradesToCandles(t *testing.T) {
	engerino := RPCTestSetup(t)
	defer CleanRPCTest(t, engerino)
	s := RPCServer{Engine: engerino}
	// bad param test
	_, err := s.ConvertTradesToCandles(context.Background(), &gctrpc.ConvertTradesToCandlesRequest{})
	if err == nil {
		t.Error(unexpectedLackOfError)
		return
	}
	if !errors.Is(err, errInvalidArguments) {
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
		Start:        time.Date(2020, 0, 0, 0, 0, 0, 0, time.UTC).Format(common.SimpleTimeFormat),
		End:          time.Date(2020, 1, 1, 1, 1, 1, 1, time.UTC).Format(common.SimpleTimeFormat),
		TimeInterval: int64(kline.OneHour.Duration()),
	})
	if err == nil {
		t.Error(unexpectedLackOfError)
		return
	}
	if !errors.Is(err, errExchangeNotLoaded) {
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
		Start:        time.Date(2020, 1, 1, 1, 1, 1, 1, time.UTC).Format(common.SimpleTimeFormat),
		End:          time.Date(2020, 2, 2, 2, 2, 2, 2, time.UTC).Format(common.SimpleTimeFormat),
		TimeInterval: int64(kline.OneHour.Duration()),
	})
	if err == nil {
		t.Error(unexpectedLackOfError)
		return
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
		t.Error(err)
		return
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
		Start:        time.Date(2020, 1, 1, 1, 0, 0, 0, time.UTC).Format(common.SimpleTimeFormat),
		End:          time.Date(2020, 2, 2, 2, 2, 2, 2, time.UTC).Format(common.SimpleTimeFormat),
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
		Start:        time.Date(2020, 1, 1, 1, 1, 1, 1, time.UTC).Format(common.SimpleTimeFormat),
		End:          time.Date(2020, 2, 2, 2, 2, 2, 2, time.UTC).Format(common.SimpleTimeFormat),
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
		Start:        time.Date(2020, 1, 1, 1, 1, 1, 1, time.UTC).Format(common.SimpleTimeFormat),
		End:          time.Date(2020, 2, 2, 2, 2, 2, 2, time.UTC).Format(common.SimpleTimeFormat),
		TimeInterval: int64(kline.OneHour.Duration()),
		Sync:         true,
		Force:        true,
	})
	if err != nil {
		t.Error(err)
	}

	// load the saved candle to verify that it was overwritten
	candles, err = s.GetHistoricCandles(context.Background(), &gctrpc.GetHistoricCandlesRequest{
		Exchange: testExchange,
		Pair: &gctrpc.CurrencyPair{
			Delimiter: currency.DashDelimiter,
			Base:      currency.BTC.String(),
			Quote:     currency.USD.String(),
		},
		AssetType:    asset.Spot.String(),
		Start:        time.Date(2020, 1, 1, 1, 0, 0, 0, time.UTC).Format(common.SimpleTimeFormat),
		End:          time.Date(2020, 2, 2, 2, 2, 0, 0, time.UTC).Format(common.SimpleTimeFormat),
		TimeInterval: int64(kline.OneHour.Duration()),
		UseDb:        true,
	})
	if err != nil {
		t.Error(err)
	}

	if len(candles.Candle) != 1 {
		t.Error("expected only one candle")
	}
}

func TestGetHistoricCandles(t *testing.T) {
	engerino := RPCTestSetup(t)
	defer CleanRPCTest(t, engerino)
	s := RPCServer{Engine: engerino}
	// error checks
	defaultStart := time.Date(2020, 1, 1, 1, 1, 1, 1, time.UTC)
	defaultEnd := time.Date(2020, 1, 2, 2, 2, 2, 2, time.UTC)
	cp := currency.NewPair(currency.BTC, currency.USD)
	_, err := s.GetHistoricCandles(context.Background(), &gctrpc.GetHistoricCandlesRequest{
		Exchange: "",
		Pair: &gctrpc.CurrencyPair{
			Base:  cp.Base.String(),
			Quote: cp.Quote.String(),
		},
		Start:     defaultStart.Format(common.SimpleTimeFormat),
		End:       defaultEnd.Format(common.SimpleTimeFormat),
		AssetType: asset.Spot.String(),
	})
	if !errors.Is(err, errExchangeNotLoaded) {
		t.Errorf("expected %v, received %v", errExchangeNotLoaded, err)
	}

	_, err = s.GetHistoricCandles(context.Background(), &gctrpc.GetHistoricCandlesRequest{
		Exchange:  testExchange,
		Start:     defaultStart.Format(common.SimpleTimeFormat),
		End:       defaultEnd.Format(common.SimpleTimeFormat),
		Pair:      nil,
		AssetType: asset.Spot.String(),
	})
	if !errors.Is(err, errCurrencyPairUnset) {
		t.Errorf("expected %v, received %v", errCurrencyPairUnset, err)
	}
	_, err = s.GetHistoricCandles(context.Background(), &gctrpc.GetHistoricCandlesRequest{
		Exchange: testExchange,
		Pair: &gctrpc.CurrencyPair{
			Base:  currency.BTC.String(),
			Quote: currency.USD.String(),
		},
		Start: "2020-01-02 15:04:05",
		End:   "2020-01-02 15:04:05",
	})
	if !errors.Is(err, errInvalidTimes) {
		t.Errorf("expected %v, received %v", errInvalidTimes, err)
	}
	var results *gctrpc.GetHistoricCandlesResponse
	// default run
	results, err = s.GetHistoricCandles(context.Background(), &gctrpc.GetHistoricCandlesRequest{
		Exchange: testExchange,
		Pair: &gctrpc.CurrencyPair{
			Base:  cp.Base.String(),
			Quote: cp.Quote.String(),
		},
		Start:        defaultStart.Format(common.SimpleTimeFormat),
		End:          defaultEnd.Format(common.SimpleTimeFormat),
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
		Start:        defaultStart.Format(common.SimpleTimeFormat),
		End:          defaultEnd.Format(common.SimpleTimeFormat),
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
		Start:        defaultStart.Format(common.SimpleTimeFormat),
		End:          defaultEnd.Format(common.SimpleTimeFormat),
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
		t.Error(err)
		return
	}
	// db run including trades
	results, err = s.GetHistoricCandles(context.Background(), &gctrpc.GetHistoricCandlesRequest{
		Exchange: testExchange,
		Pair: &gctrpc.CurrencyPair{
			Base:  cp.Base.String(),
			Quote: cp.Quote.String(),
		},
		AssetType:             asset.Spot.String(),
		Start:                 defaultStart.Format(common.SimpleTimeFormat),
		End:                   time.Date(2020, 1, 2, 4, 2, 2, 2, time.UTC).Format(common.SimpleTimeFormat),
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
	engerino := RPCTestSetup(t)
	defer CleanRPCTest(t, engerino)
	s := RPCServer{Engine: engerino}
	// bad request checks
	_, err := s.FindMissingSavedTradeIntervals(context.Background(), &gctrpc.FindMissingTradePeriodsRequest{})
	if err == nil {
		t.Error("expected error")
		return
	}
	if !errors.Is(err, errInvalidArguments) {
		t.Error(err)
		return
	}
	cp := currency.NewPair(currency.BTC, currency.USD)
	// no data found response
	defaultStart := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).UTC()
	defaultEnd := time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC).UTC()
	var resp *gctrpc.FindMissingIntervalsResponse
	resp, err = s.FindMissingSavedTradeIntervals(context.Background(), &gctrpc.FindMissingTradePeriodsRequest{
		ExchangeName: testExchange,
		AssetType:    asset.Spot.String(),
		Pair: &gctrpc.CurrencyPair{
			Base:  cp.Base.String(),
			Quote: cp.Quote.String(),
		},
		Start: defaultStart.UTC().Format(common.SimpleTimeFormat),
		End:   defaultEnd.UTC().Format(common.SimpleTimeFormat),
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
		Timestamp:    time.Date(2020, 1, 1, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Error(err)
		return
	}

	resp, err = s.FindMissingSavedTradeIntervals(context.Background(), &gctrpc.FindMissingTradePeriodsRequest{
		ExchangeName: testExchange,
		AssetType:    asset.Spot.String(),
		Pair: &gctrpc.CurrencyPair{
			Base:  cp.Base.String(),
			Quote: cp.Quote.String(),
		},
		Start: defaultStart.In(time.UTC).Format(common.SimpleTimeFormat),
		End:   defaultEnd.In(time.UTC).Format(common.SimpleTimeFormat),
	})
	if err != nil {
		t.Error(err)
	}
	if len(resp.MissingPeriods) != 2 {
		t.Errorf("expected 2 missing period, received: %v", len(resp.MissingPeriods))
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
		Timestamp:    time.Date(2020, 1, 1, 13, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Error(err)
		return
	}

	resp, err = s.FindMissingSavedTradeIntervals(context.Background(), &gctrpc.FindMissingTradePeriodsRequest{
		ExchangeName: testExchange,
		AssetType:    asset.Spot.String(),
		Pair: &gctrpc.CurrencyPair{
			Base:  cp.Base.String(),
			Quote: cp.Quote.String(),
		},
		Start: defaultStart.In(time.UTC).Format(common.SimpleTimeFormat),
		End:   defaultEnd.In(time.UTC).Format(common.SimpleTimeFormat),
	})
	if err != nil {
		t.Error(err)
	}
	if len(resp.MissingPeriods) != 2 {
		t.Errorf("expected 2 missing periods, received: %v", len(resp.MissingPeriods))
	}
}

func TestFindMissingSavedCandleIntervals(t *testing.T) {
	engerino := RPCTestSetup(t)
	defer CleanRPCTest(t, engerino)
	s := RPCServer{Engine: engerino}
	// bad request checks
	_, err := s.FindMissingSavedCandleIntervals(context.Background(), &gctrpc.FindMissingCandlePeriodsRequest{})
	if err == nil {
		t.Error("expected error")
		return
	}
	if !errors.Is(err, errInvalidArguments) {
		t.Error(err)
		return
	}
	cp := currency.NewPair(currency.BTC, currency.USD)
	// no data found response
	defaultStart := time.Date(2020, 1, 1, 1, 1, 1, 1, time.UTC)
	defaultEnd := time.Date(2020, 1, 2, 2, 2, 2, 2, time.UTC)
	var resp *gctrpc.FindMissingIntervalsResponse
	_, err = s.FindMissingSavedCandleIntervals(context.Background(), &gctrpc.FindMissingCandlePeriodsRequest{
		ExchangeName: testExchange,
		AssetType:    asset.Spot.String(),
		Pair: &gctrpc.CurrencyPair{
			Base:  cp.Base.String(),
			Quote: cp.Quote.String(),
		},
		Interval: int64(time.Hour),
		Start:    defaultStart.Format(common.SimpleTimeFormat),
		End:      defaultEnd.Format(common.SimpleTimeFormat),
	})
	if err != nil && err.Error() != "no candle data found: Bitstamp BTC USD 3600 spot" {
		t.Error(err)
		return
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
		t.Error(err)
		return
	}

	_, err = s.FindMissingSavedCandleIntervals(context.Background(), &gctrpc.FindMissingCandlePeriodsRequest{
		ExchangeName: testExchange,
		AssetType:    asset.Spot.String(),
		Pair: &gctrpc.CurrencyPair{
			Base:  cp.Base.String(),
			Quote: cp.Quote.String(),
		},
		Interval: int64(time.Hour),
		Start:    defaultStart.Format(common.SimpleTimeFormat),
		End:      defaultEnd.Format(common.SimpleTimeFormat),
	})
	if err != nil {
		t.Error(err)
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
		t.Error(err)
		return
	}

	resp, err = s.FindMissingSavedCandleIntervals(context.Background(), &gctrpc.FindMissingCandlePeriodsRequest{
		ExchangeName: testExchange,
		AssetType:    asset.Spot.String(),
		Pair: &gctrpc.CurrencyPair{
			Base:  cp.Base.String(),
			Quote: cp.Quote.String(),
		},
		Interval: int64(time.Hour),
		Start:    defaultStart.Format(common.SimpleTimeFormat),
		End:      defaultEnd.Format(common.SimpleTimeFormat),
	})
	if err != nil {
		t.Error(err)
	}
	if len(resp.MissingPeriods) != 2 {
		t.Errorf("expected 2 missing periods, received: %v", len(resp.MissingPeriods))
	}
}

func TestSetExchangeTradeProcessing(t *testing.T) {
	engerino := RPCTestSetup(t)
	defer CleanRPCTest(t, engerino)
	s := RPCServer{Engine: engerino}
	_, err := s.SetExchangeTradeProcessing(context.Background(), &gctrpc.SetExchangeTradeProcessingRequest{Exchange: testExchange, Status: true})
	if err != nil {
		t.Error(err)
		return
	}
	exch := s.GetExchangeByName(testExchange)
	base := exch.GetBase()
	if !base.IsSaveTradeDataEnabled() {
		t.Error("expected true")
	}

	_, err = s.SetExchangeTradeProcessing(context.Background(), &gctrpc.SetExchangeTradeProcessingRequest{Exchange: testExchange, Status: false})
	if err != nil {
		t.Error(err)
		return
	}
	exch = s.GetExchangeByName(testExchange)
	base = exch.GetBase()
	if base.IsSaveTradeDataEnabled() {
		t.Error("expected false")
	}
}

func TestGetRecentTrades(t *testing.T) {
	engerino := RPCTestSetup(t)
	defer CleanRPCTest(t, engerino)
	s := RPCServer{Engine: engerino}
	_, err := s.GetRecentTrades(context.Background(), &gctrpc.GetSavedTradesRequest{})
	if err == nil {
		t.Error(unexpectedLackOfError)
		return
	}
	if !errors.Is(err, errInvalidArguments) {
		t.Error(err)
	}
	_, err = s.GetRecentTrades(context.Background(), &gctrpc.GetSavedTradesRequest{
		Exchange: "fake",
		Pair: &gctrpc.CurrencyPair{
			Delimiter: currency.DashDelimiter,
			Base:      currency.BTC.String(),
			Quote:     currency.USD.String(),
		},
		AssetType: asset.Spot.String(),
		Start:     time.Date(2020, 0, 0, 0, 0, 0, 0, time.UTC).Format(common.SimpleTimeFormat),
		End:       time.Date(2020, 1, 1, 1, 1, 1, 1, time.UTC).Format(common.SimpleTimeFormat),
	})
	if err == nil {
		t.Error(unexpectedLackOfError)
		return
	}
	if !errors.Is(err, errExchangeNotLoaded) {
		t.Error(err)
	}
	_, err = s.GetRecentTrades(context.Background(), &gctrpc.GetSavedTradesRequest{
		Exchange: testExchange,
		Pair: &gctrpc.CurrencyPair{
			Delimiter: currency.DashDelimiter,
			Base:      currency.BTC.String(),
			Quote:     currency.USD.String(),
		},
		AssetType: asset.Spot.String(),
	})
	if err != nil {
		t.Error(err)
	}
}

func TestGetHistoricTrades(t *testing.T) {
	engerino := RPCTestSetup(t)
	defer CleanRPCTest(t, engerino)
	s := RPCServer{Engine: engerino}
	err := s.GetHistoricTrades(&gctrpc.GetSavedTradesRequest{}, nil)
	if err == nil {
		t.Error(unexpectedLackOfError)
		return
	}
	if !errors.Is(err, errInvalidArguments) {
		t.Error(err)
	}
	err = s.GetHistoricTrades(&gctrpc.GetSavedTradesRequest{
		Exchange: "fake",
		Pair: &gctrpc.CurrencyPair{
			Delimiter: currency.DashDelimiter,
			Base:      currency.BTC.String(),
			Quote:     currency.USD.String(),
		},
		AssetType: asset.Spot.String(),
		Start:     time.Date(2020, 0, 0, 0, 0, 0, 0, time.UTC).Format(common.SimpleTimeFormat),
		End:       time.Date(2020, 1, 1, 1, 1, 1, 1, time.UTC).Format(common.SimpleTimeFormat),
	}, nil)
	if err == nil {
		t.Error(unexpectedLackOfError)
		return
	}
	if !errors.Is(err, errExchangeNotLoaded) {
		t.Error(err)
	}
	err = s.GetHistoricTrades(&gctrpc.GetSavedTradesRequest{
		Exchange: testExchange,
		Pair: &gctrpc.CurrencyPair{
			Delimiter: currency.DashDelimiter,
			Base:      currency.BTC.String(),
			Quote:     currency.USD.String(),
		},
		AssetType: asset.Spot.String(),
		Start:     time.Date(2020, 0, 0, 0, 0, 0, 0, time.UTC).Format(common.SimpleTimeFormat),
		End:       time.Date(2020, 1, 1, 1, 1, 1, 1, time.UTC).Format(common.SimpleTimeFormat),
	}, nil)
	if err == nil {
		t.Error(unexpectedLackOfError)
		return
	}
	if err != common.ErrFunctionNotSupported {
		t.Error(err)
	}
}

func TestGetAccountInfo(t *testing.T) {
	bot := CreateTestBot(t)
	s := RPCServer{Engine: bot}

	r, err := s.GetAccountInfo(context.Background(), &gctrpc.GetAccountInfoRequest{Exchange: fakePassExchange, AssetType: asset.Spot.String()})
	if err != nil {
		t.Fatalf("TestGetAccountInfo: Failed to get account info: %s", err)
	}
	if r.Accounts[0].Currencies[0].TotalValue != 10 {
		t.Fatal("TestGetAccountInfo: Unexpected value of the 'TotalValue'")
	}
}

func TestUpdateAccountInfo(t *testing.T) {
	bot := CreateTestBot(t)
	s := RPCServer{Engine: bot}

	getResponse, err := s.GetAccountInfo(context.Background(), &gctrpc.GetAccountInfoRequest{Exchange: fakePassExchange, AssetType: asset.Spot.String()})
	if err != nil {
		t.Fatalf("TestGetAccountInfo: Failed to get account info: %s", err)
	}

	_, err = s.UpdateAccountInfo(context.Background(), &gctrpc.GetAccountInfoRequest{Exchange: fakePassExchange, AssetType: asset.Futures.String()})
	if !errors.Is(err, errAssetTypeDisabled) {
		t.Errorf("expected %v, received %v", errAssetTypeDisabled, err)
	}

	updateResp, err := s.UpdateAccountInfo(context.Background(), &gctrpc.GetAccountInfoRequest{
		Exchange:  fakePassExchange,
		AssetType: asset.Spot.String(),
	})
	if !errors.Is(err, nil) {
		t.Error(err)
	} else if getResponse.Accounts[0].Currencies[0].TotalValue == updateResp.Accounts[0].Currencies[0].TotalValue {
		t.Fatalf("TestGetAccountInfo: Unexpected value of the 'TotalValue'")
	}
}

func TestGetOrders(t *testing.T) {
	exchName := "binance"
	engerino := RPCTestSetup(t)
	defer CleanRPCTest(t, engerino)
	s := RPCServer{Engine: engerino}

	p := &gctrpc.CurrencyPair{
		Delimiter: "-",
		Base:      currency.BTC.String(),
		Quote:     currency.USDT.String(),
	}

	_, err := s.GetOrders(context.Background(), nil)
	if !errors.Is(err, errInvalidArguments) {
		t.Errorf("expected %v, received %v", errInvalidArguments, err)
	}

	_, err = s.GetOrders(context.Background(), &gctrpc.GetOrdersRequest{
		AssetType: asset.Spot.String(),
		Pair:      p,
	})
	if !errors.Is(err, errExchangeNotLoaded) {
		t.Errorf("expected %v, received %v", errExchangeNotLoaded, err)
	}

	err = engerino.LoadExchange(exchName, false, nil)
	if err != nil {
		t.Error(err)
	}

	_, err = s.GetOrders(context.Background(), &gctrpc.GetOrdersRequest{
		Exchange:  exchName,
		AssetType: asset.Spot.String(),
	})
	if !errors.Is(err, errCurrencyPairUnset) {
		t.Errorf("expected %v, received %v", errCurrencyPairUnset, err)
	}

	_, err = s.GetOrders(context.Background(), &gctrpc.GetOrdersRequest{
		Exchange: exchName,
		Pair:     p,
	})
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Errorf("expected %v, received %v", asset.ErrNotSupported, err)
	}

	_, err = s.GetOrders(context.Background(), &gctrpc.GetOrdersRequest{
		Exchange:  exchName,
		AssetType: asset.Spot.String(),
		Pair:      p,
		StartDate: time.Now().Format(common.SimpleTimeFormat),
		EndDate:   time.Now().Add(-time.Hour).Format(common.SimpleTimeFormat),
	})
	if !errors.Is(err, errInvalidTimes) {
		t.Errorf("expected %v, received %v", errInvalidTimes, err)
	}

	_, err = s.GetOrders(context.Background(), &gctrpc.GetOrdersRequest{
		Exchange:  exchName,
		AssetType: asset.Spot.String(),
		Pair:      p,
		StartDate: time.Now().Format(common.SimpleTimeFormat),
		EndDate:   time.Now().Add(time.Hour).Format(common.SimpleTimeFormat),
	})
	if err != nil && !strings.Contains(err.Error(), "not supported due to unset/default API keys") {
		t.Error(err)
	}
	if err == nil {
		t.Error("expected error")
	}

	exch := engerino.GetExchangeByName(exchName)
	if exch == nil {
		t.Fatal("expected an exchange")
	}
	b := exch.GetBase()
	b.API.Credentials.Key = "test"
	b.API.Credentials.Secret = "test"
	b.API.AuthenticatedSupport = true

	_, err = s.GetOrders(context.Background(), &gctrpc.GetOrdersRequest{
		Exchange:  exchName,
		AssetType: asset.Spot.String(),
		Pair:      p,
	})
	if err == nil {
		t.Error("expected error")
	}
}

func TestGetOrder(t *testing.T) {
	exchName := "binance"
	engerino := RPCTestSetup(t)
	defer CleanRPCTest(t, engerino)
	s := RPCServer{Engine: engerino}

	p := &gctrpc.CurrencyPair{
		Delimiter: "-",
		Base:      "BTC",
		Quote:     "USDT",
	}

	_, err := s.GetOrder(context.Background(), nil)
	if !errors.Is(err, errInvalidArguments) {
		t.Errorf("expected %v, received %v", errInvalidArguments, err)
	}

	_, err = s.GetOrder(context.Background(), &gctrpc.GetOrderRequest{
		Exchange: exchName,
		OrderId:  "",
		Pair:     p,
		Asset:    "spot",
	})

	if !errors.Is(err, errExchangeNotLoaded) {
		t.Errorf("expected %v, received %v", errExchangeNotLoaded, err)
	}

	err = engerino.LoadExchange(exchName, false, nil)
	if err != nil {
		t.Error(err)
	}

	_, err = s.GetOrder(context.Background(), &gctrpc.GetOrderRequest{
		Exchange: exchName,
		OrderId:  "",
		Pair:     nil,
		Asset:    "",
	})
	if !errors.Is(err, errCurrencyPairUnset) {
		t.Errorf("expected %v, received %v", errCurrencyPairUnset, err)
	}

	_, err = s.GetOrder(context.Background(), &gctrpc.GetOrderRequest{
		Exchange: exchName,
		OrderId:  "",
		Pair:     p,
		Asset:    "",
	})
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Errorf("expected %v, received %v", asset.ErrNotSupported, err)
	}

	_, err = s.GetOrder(context.Background(), &gctrpc.GetOrderRequest{
		Exchange: exchName,
		OrderId:  "",
		Pair:     p,
		Asset:    asset.Spot.String(),
	})
	if !errors.Is(err, errOrderIDCannotBeEmpty) {
		t.Errorf("expected %v, received %v", errOrderIDCannotBeEmpty, err)
	}
	err = engerino.OrderManager.Start(engerino)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.GetOrder(context.Background(), &gctrpc.GetOrderRequest{
		Exchange: exchName,
		OrderId:  "1234",
		Pair:     p,
		Asset:    asset.Spot.String(),
	})
	if err == nil {
		t.Error("expected error")
	}
}

func TestCheckVars(t *testing.T) {
	var e exchange.IBotExchange

	err := checkParams("Binance", e, asset.Spot, currency.NewPair(currency.BTC, currency.USDT))
	if !errors.Is(err, errExchangeNotLoaded) {
		t.Errorf("expected %v, got %v", errExchangeNotLoaded, err)
	}

	e = &binance.Binance{}
	_, ok := e.(*binance.Binance)
	if !ok {
		t.Fatal("invalid ibotexchange interface")
	}

	err = checkParams("Binance", e, asset.Spot, currency.NewPair(currency.BTC, currency.USDT))
	if !errors.Is(err, errExchangeDisabled) {
		t.Errorf("expected %v, got %v", errExchangeDisabled, err)
	}

	e.SetEnabled(true)

	err = checkParams("Binance", e, asset.Spot, currency.NewPair(currency.BTC, currency.USDT))
	if !errors.Is(err, errAssetTypeDisabled) {
		t.Errorf("expected %v, got %v", errAssetTypeDisabled, err)
	}

	fmt1 := currency.PairStore{
		RequestFormat: &currency.PairFormat{Uppercase: true},
		ConfigFormat: &currency.PairFormat{
			Delimiter: currency.DashDelimiter,
			Uppercase: true,
		},
	}
	coinFutures := currency.PairStore{
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: currency.UnderscoreDelimiter,
		},
		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: currency.UnderscoreDelimiter,
		},
	}
	usdtFutures := currency.PairStore{
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
		},
		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
		},
	}
	err = e.GetBase().StoreAssetPairFormat(asset.Spot, fmt1)
	if err != nil {
		t.Error(err)
	}
	err = e.GetBase().StoreAssetPairFormat(asset.Margin, fmt1)
	if err != nil {
		t.Error(err)
	}
	err = e.GetBase().StoreAssetPairFormat(asset.CoinMarginedFutures, coinFutures)
	if err != nil {
		t.Error(err)
	}
	err = e.GetBase().StoreAssetPairFormat(asset.USDTMarginedFutures, usdtFutures)
	if err != nil {
		t.Error(err)
	}

	err = checkParams("Binance", e, asset.Spot, currency.NewPair(currency.BTC, currency.USDT))
	if !errors.Is(err, errCurrencyPairInvalid) {
		t.Errorf("expected %v, got %v", errCurrencyPairInvalid, err)
	}

	var data = []currency.Pair{
		{Delimiter: currency.DashDelimiter, Base: currency.BTC, Quote: currency.USDT},
	}

	e.GetBase().CurrencyPairs.StorePairs(asset.Spot, data, false)

	err = checkParams("Binance", e, asset.Spot, currency.NewPair(currency.BTC, currency.USDT))
	if !errors.Is(err, errCurrencyNotEnabled) {
		t.Errorf("expected %v, got %v", errCurrencyNotEnabled, err)
	}

	e.GetBase().CurrencyPairs.EnablePair(
		asset.Spot,
		currency.Pair{Delimiter: currency.DashDelimiter, Base: currency.BTC, Quote: currency.USDT},
	)

	err = checkParams("Binance", e, asset.Spot, currency.NewPair(currency.BTC, currency.USDT))
	if err != nil {
		t.Error(err)
	}
}
