package engine

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/drivers"
	"github.com/thrasher-corp/gocryptotrader/database/repository"
	dbexchange "github.com/thrasher-corp/gocryptotrader/database/repository/exchange"
	sqltrade "github.com/thrasher-corp/gocryptotrader/database/repository/trade"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/binance"
	"github.com/thrasher-corp/gocryptotrader/exchanges/currencystate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"github.com/thrasher-corp/gocryptotrader/portfolio/banking"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/goose"
	"google.golang.org/grpc/metadata"
)

const (
	unexpectedLackOfError = "unexpected lack of error"
	migrationsFolder      = "migrations"
	databaseFolder        = "database"
	fakeExchangeName      = "fake"
)

// fExchange is a fake exchange with function overrides
// we're not testing an actual exchange's implemented functions
type fExchange struct {
	exchange.IBotExchange
}

func (f fExchange) GetHistoricCandles(ctx context.Context, p currency.Pair, a asset.Item, timeStart, _ time.Time, interval kline.Interval) (kline.Item, error) {
	return kline.Item{
		Exchange: fakeExchangeName,
		Pair:     p,
		Asset:    a,
		Interval: interval,
		Candles: []kline.Candle{
			{
				Time:   timeStart,
				Open:   1337,
				High:   1337,
				Low:    1337,
				Close:  1337,
				Volume: 1337,
			},
		},
	}, nil
}

func (f fExchange) GetHistoricCandlesExtended(ctx context.Context, p currency.Pair, a asset.Item, timeStart, _ time.Time, interval kline.Interval) (kline.Item, error) {
	return kline.Item{
		Exchange: fakeExchangeName,
		Pair:     p,
		Asset:    a,
		Interval: interval,
		Candles: []kline.Candle{
			{
				Time:   timeStart,
				Open:   1337,
				High:   1337,
				Low:    1337,
				Close:  1337,
				Volume: 1337,
			},
		},
	}, nil
}

func (f fExchange) FetchTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	return &ticker.Price{
		Last:         1337,
		High:         1337,
		Low:          1337,
		Bid:          1337,
		Ask:          1337,
		Volume:       1337,
		QuoteVolume:  1337,
		PriceATH:     1337,
		Open:         1337,
		Close:        1337,
		Pair:         p,
		ExchangeName: f.GetName(),
		AssetType:    a,
		LastUpdated:  time.Now(),
	}, nil
}

// FetchAccountInfo overrides testExchange's fetch account info function
// to do the bare minimum required with no API calls or credentials required
func (f fExchange) FetchAccountInfo(_ context.Context, a asset.Item) (account.Holdings, error) {
	return account.Holdings{
		Exchange: f.GetName(),
		Accounts: []account.SubAccount{
			{
				ID:        "1337",
				AssetType: a,
				Currencies: []account.Balance{
					{
						CurrencyName: currency.USD,
						Total:        1337,
					},
					{
						CurrencyName: currency.BTC,
						Total:        13337,
					},
				},
			},
		},
	}, nil
}

// GetFuturesPositions overrides testExchange's GetFuturesPositions function
func (f fExchange) GetFuturesPositions(_ context.Context, a asset.Item, cp currency.Pair, _, _ time.Time) ([]order.Detail, error) {
	return []order.Detail{
		{
			Price:     1337,
			Amount:    1337,
			Fee:       1.337,
			FeeAsset:  currency.Code{},
			Exchange:  f.GetName(),
			ID:        "test",
			Side:      order.Long,
			Status:    order.Open,
			AssetType: a,
			Date:      time.Now(),
			Pair:      cp,
		},
	}, nil
}

// CalculateTotalCollateral overrides testExchange's CalculateTotalCollateral function
func (f fExchange) CalculateTotalCollateral(context.Context, *order.TotalCollateralCalculator) (*order.TotalCollateralResponse, error) {
	return &order.TotalCollateralResponse{
		CollateralCurrency:             currency.USD,
		AvailableMaintenanceCollateral: decimal.NewFromInt(1338),
		AvailableCollateral:            decimal.NewFromInt(1337),
		UsedBreakdown: &order.UsedCollateralBreakdown{
			LockedInStakes:                  decimal.NewFromInt(3),
			LockedInNFTBids:                 decimal.NewFromInt(3),
			LockedInFeeVoucher:              decimal.NewFromInt(3),
			LockedInSpotMarginFundingOffers: decimal.NewFromInt(3),
			LockedInSpotOrders:              decimal.NewFromInt(3),
			LockedAsCollateral:              decimal.NewFromInt(3),
		},
		BreakdownByCurrency: []order.CollateralByCurrency{
			{
				Currency:               currency.USD,
				TotalFunds:             decimal.NewFromInt(1330),
				CollateralContribution: decimal.NewFromInt(1330),
				ScaledCurrency:         currency.USD,
			},
			{
				Currency:   currency.DOGE,
				TotalFunds: decimal.NewFromInt(1000),
				ScaledUsed: decimal.NewFromInt(6),
				ScaledUsedBreakdown: &order.UsedCollateralBreakdown{
					LockedInStakes:                  decimal.NewFromInt(1),
					LockedInNFTBids:                 decimal.NewFromInt(1),
					LockedInFeeVoucher:              decimal.NewFromInt(1),
					LockedInSpotMarginFundingOffers: decimal.NewFromInt(1),
					LockedInSpotOrders:              decimal.NewFromInt(1),
					LockedAsCollateral:              decimal.NewFromInt(1),
				},
				CollateralContribution: decimal.NewFromInt(4),
				ScaledCurrency:         currency.USD,
			},
			{
				Currency:               currency.XRP,
				TotalFunds:             decimal.NewFromInt(1333333333333337),
				CollateralContribution: decimal.NewFromInt(-3),
				ScaledCurrency:         currency.USD,
			},
		},
	}, nil
}

// UpdateAccountInfo overrides testExchange's update account info function
// to do the bare minimum required with no API calls or credentials required
func (f fExchange) UpdateAccountInfo(ctx context.Context, a asset.Item) (account.Holdings, error) {
	if a == asset.Futures {
		return account.Holdings{}, errAssetTypeDisabled
	}
	return account.Holdings{
		Exchange: f.GetName(),
		Accounts: []account.SubAccount{
			{
				ID:         "1337",
				AssetType:  a,
				Currencies: nil,
			},
		},
	}, nil
}

// GetCurrencyStateSnapshot overrides interface function
func (f fExchange) GetCurrencyStateSnapshot() ([]currencystate.Snapshot, error) {
	return []currencystate.Snapshot{
		{
			Code:  currency.BTC,
			Asset: asset.Spot,
		},
	}, nil
}

// CanTradePair overrides interface function
func (f fExchange) CanTradePair(p currency.Pair, a asset.Item) error {
	return nil
}

// CanTrade overrides interface function
func (f fExchange) CanTrade(c currency.Code, a asset.Item) error {
	return nil
}

// CanWithdraw overrides interface function
func (f fExchange) CanWithdraw(c currency.Code, a asset.Item) error {
	return nil
}

// CanDeposit overrides interface function
func (f fExchange) CanDeposit(c currency.Code, a asset.Item) error {
	return nil
}

// Sets up everything required to run any function inside rpcserver
// Only use if you require a database, this makes tests slow
func RPCTestSetup(t *testing.T) *Engine {
	t.Helper()
	var err error
	dbConf := database.Config{
		Enabled: true,
		Driver:  database.DBSQLite3,
		ConnectionDetails: drivers.ConnectionDetails{
			Host:     "localhost",
			Database: "test123.db",
		},
	}
	engerino := new(Engine)
	dbm, err := SetupDatabaseConnectionManager(&dbConf)
	if err != nil {
		t.Fatal(err)
	}
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	dbm.dbConn.DataPath = tempDir
	engerino.DatabaseManager = dbm
	var wg sync.WaitGroup
	err = dbm.Start(&wg)
	if err != nil {
		t.Fatal(err)
	}
	engerino.Config = &config.Config{}
	em := SetupExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	b := exch.GetBase()
	cp := currency.NewPair(currency.BTC, currency.USD)
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		Available:     currency.Pairs{cp},
		Enabled:       currency.Pairs{cp},
		AssetEnabled:  convert.BoolPtr(true),
		ConfigFormat:  &currency.PairFormat{Uppercase: true},
		RequestFormat: &currency.PairFormat{Uppercase: true}}
	em.Add(exch)

	exch, err = em.NewExchangeByName("Binance")
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	b = exch.GetBase()
	cp = currency.NewPair(currency.BTC, currency.USDT)
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		Available:     currency.Pairs{cp},
		Enabled:       currency.Pairs{cp},
		AssetEnabled:  convert.BoolPtr(true),
		ConfigFormat:  &currency.PairFormat{Uppercase: true},
		RequestFormat: &currency.PairFormat{Uppercase: true}}
	em.Add(exch)
	engerino.ExchangeManager = em

	engerino.Config.Database = dbConf
	engerino.DatabaseManager, err = SetupDatabaseConnectionManager(&engerino.Config.Database)
	if err != nil {
		log.Fatal(err)
	}
	err = engerino.DatabaseManager.Start(&engerino.ServicesWG)
	if err != nil {
		log.Fatal(err)
	}
	path := filepath.Join("..", databaseFolder, migrationsFolder)
	err = goose.Run("up", database.DB.SQL, repository.GetSQLDialect(), path, "")
	if err != nil {
		t.Fatalf("failed to run migrations %v", err)
	}
	uuider, err := uuid.NewV4()
	if err != nil {
		t.Fatal(err)
	}
	uuider2, err := uuid.NewV4()
	if err != nil {
		t.Fatal(err)
	}
	err = dbexchange.InsertMany([]dbexchange.Details{{Name: testExchange, UUID: uuider}, {Name: "Binance", UUID: uuider2}})
	if err != nil {
		t.Fatalf("failed to insert exchange %v", err)
	}

	return engerino
}

func CleanRPCTest(t *testing.T, engerino *Engine) {
	t.Helper()
	err := engerino.DatabaseManager.Stop()
	if err != nil {
		t.Error(err)
		return
	}
	err = os.Remove(filepath.Join(engerino.DatabaseManager.dbConn.DataPath, engerino.DatabaseManager.cfg.Database))
	if err != nil {
		t.Error(err)
	}
}

func TestGetSavedTrades(t *testing.T) {
	engerino := RPCTestSetup(t)
	defer CleanRPCTest(t, engerino)
	s := RPCServer{Engine: engerino}
	_, err := s.GetSavedTrades(context.Background(), &gctrpc.GetSavedTradesRequest{})
	if !errors.Is(err, errInvalidArguments) {
		t.Error(err)
	}
	_, err = s.GetSavedTrades(context.Background(), &gctrpc.GetSavedTradesRequest{
		Exchange: fakeExchangeName,
		Pair: &gctrpc.CurrencyPair{
			Delimiter: currency.DashDelimiter,
			Base:      currency.BTC.String(),
			Quote:     currency.USD.String(),
		},
		AssetType: asset.Spot.String(),
		Start:     time.Date(2020, 0, 0, 0, 0, 0, 0, time.UTC).Format(common.SimpleTimeFormat),
		End:       time.Date(2020, 1, 1, 1, 1, 1, 1, time.UTC).Format(common.SimpleTimeFormat),
	})
	if !errors.Is(err, ErrExchangeNotFound) {
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
		End:          time.Date(2020, 0, 0, 1, 0, 0, 0, time.UTC).Format(common.SimpleTimeFormat),
		TimeInterval: int64(kline.OneHour.Duration()),
	})
	if !errors.Is(err, ErrExchangeNotFound) {
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
		Start:        time.Date(2020, 0, 0, 0, 0, 0, 0, time.UTC).Format(common.SimpleTimeFormat),
		End:          time.Date(2020, 0, 0, 1, 0, 0, 0, time.UTC).Format(common.SimpleTimeFormat),
		TimeInterval: int64(kline.OneHour.Duration()),
	})
	if !errors.Is(err, errNoTrades) {
		t.Errorf("received '%v' expected '%v'", err, errNoTrades)
	}

	// add a trade
	err = sqltrade.Insert(sqltrade.Data{
		Timestamp: time.Date(2020, 0, 0, 0, 30, 0, 0, time.UTC),
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
		Start:        time.Date(2020, 0, 0, 0, 0, 0, 0, time.UTC).Format(common.SimpleTimeFormat),
		End:          time.Date(2020, 0, 0, 1, 0, 0, 0, time.UTC).Format(common.SimpleTimeFormat),
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
		Start:        time.Date(2020, 0, 0, 0, 0, 0, 0, time.UTC).Format(common.SimpleTimeFormat),
		End:          time.Date(2020, 0, 0, 1, 0, 0, 0, time.UTC).Format(common.SimpleTimeFormat),
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
		Start:        time.Date(2020, 0, 0, 0, 0, 0, 0, time.UTC).Format(common.SimpleTimeFormat),
		End:          time.Date(2020, 0, 0, 1, 0, 0, 0, time.UTC).Format(common.SimpleTimeFormat),
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
		Start:        time.Date(2020, 0, 0, 0, 0, 0, 0, time.UTC).Format(common.SimpleTimeFormat),
		End:          time.Date(2020, 0, 0, 1, 0, 0, 0, time.UTC).Format(common.SimpleTimeFormat),
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
	defaultStart := time.Date(2020, 0, 0, 0, 0, 0, 0, time.UTC)
	defaultEnd := time.Date(2020, 0, 0, 1, 0, 0, 0, time.UTC)
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
	if !errors.Is(err, ErrExchangeNameIsEmpty) {
		t.Errorf("received '%v', expected '%v'", err, ErrExchangeNameIsEmpty)
	}

	_, err = s.GetHistoricCandles(context.Background(), &gctrpc.GetHistoricCandlesRequest{
		Exchange: "bruh",
		Pair: &gctrpc.CurrencyPair{
			Base:  cp.Base.String(),
			Quote: cp.Quote.String(),
		},
		Start:     defaultStart.Format(common.SimpleTimeFormat),
		End:       defaultEnd.Format(common.SimpleTimeFormat),
		AssetType: asset.Spot.String(),
	})
	if !errors.Is(err, ErrExchangeNotFound) {
		t.Errorf("received '%v', expected '%v'", err, ErrExchangeNotFound)
	}

	_, err = s.GetHistoricCandles(context.Background(), &gctrpc.GetHistoricCandlesRequest{
		Exchange:  testExchange,
		Start:     defaultStart.Format(common.SimpleTimeFormat),
		End:       defaultEnd.Format(common.SimpleTimeFormat),
		Pair:      nil,
		AssetType: asset.Spot.String(),
	})
	if !errors.Is(err, errCurrencyPairUnset) {
		t.Errorf("received '%v', expected '%v'", err, errCurrencyPairUnset)
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
	if !errors.Is(err, common.ErrStartEqualsEnd) {
		t.Errorf("received %v, expected %v", err, common.ErrStartEqualsEnd)
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
		Timestamp:    time.Date(2020, 0, 0, 2, 0, 0, 0, time.UTC),
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
		End:                   time.Date(2020, 0, 0, 3, 0, 0, 0, time.UTC).Format(common.SimpleTimeFormat),
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
	defaultStart := time.Date(2020, 0, 0, 0, 0, 0, 0, time.UTC)
	defaultEnd := time.Date(2020, 0, 0, 4, 0, 0, 0, time.UTC)
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
				Time:   time.Date(2020, 0, 0, 0, 30, 0, 0, time.UTC),
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
				Time:   time.Date(2020, 0, 0, 2, 45, 0, 0, time.UTC),
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
	t.Parallel()
	em := SetupExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	b := exch.GetBase()
	b.Config = &config.Exchange{
		Features: &config.FeaturesConfig{Enabled: config.FeaturesEnabledConfig{SaveTradeData: false}},
	}
	em.Add(exch)
	s := RPCServer{Engine: &Engine{ExchangeManager: em}}
	_, err = s.SetExchangeTradeProcessing(context.Background(), &gctrpc.SetExchangeTradeProcessingRequest{Exchange: testExchange, Status: true})
	if err != nil {
		t.Error(err)
		return
	}
	if !b.IsSaveTradeDataEnabled() {
		t.Error("expected true")
	}
	_, err = s.SetExchangeTradeProcessing(context.Background(), &gctrpc.SetExchangeTradeProcessingRequest{Exchange: testExchange, Status: false})
	if err != nil {
		t.Error(err)
		return
	}
	if b.IsSaveTradeDataEnabled() {
		t.Error("expected false")
	}
}

func TestGetRecentTrades(t *testing.T) {
	engerino := RPCTestSetup(t)
	defer CleanRPCTest(t, engerino)
	s := RPCServer{Engine: engerino}
	_, err := s.GetRecentTrades(context.Background(), &gctrpc.GetSavedTradesRequest{})
	if !errors.Is(err, errInvalidArguments) {
		t.Error(err)
	}
	_, err = s.GetRecentTrades(context.Background(), &gctrpc.GetSavedTradesRequest{
		Exchange: fakeExchangeName,
		Pair: &gctrpc.CurrencyPair{
			Delimiter: currency.DashDelimiter,
			Base:      currency.BTC.String(),
			Quote:     currency.USD.String(),
		},
		AssetType: asset.Spot.String(),
		Start:     time.Date(2020, 0, 0, 0, 0, 0, 0, time.UTC).Format(common.SimpleTimeFormat),
		End:       time.Date(2020, 0, 0, 1, 0, 0, 0, time.UTC).Format(common.SimpleTimeFormat),
	})
	if !errors.Is(err, ErrExchangeNotFound) {
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

// dummyServer implements a basic RPC server interface for deployment in a test
// when streaming occurs, so we can deliver a context value.
type dummyServer struct{}

func (d *dummyServer) Send(*gctrpc.SavedTradesResponse) error { return nil }
func (d *dummyServer) SetHeader(metadata.MD) error            { return nil }
func (d *dummyServer) SendHeader(metadata.MD) error           { return nil }
func (d *dummyServer) SetTrailer(metadata.MD)                 {}
func (d *dummyServer) Context() context.Context               { return context.Background() }
func (d *dummyServer) SendMsg(m interface{}) error            { return nil }
func (d *dummyServer) RecvMsg(m interface{}) error            { return nil }

func TestGetHistoricTrades(t *testing.T) {
	engerino := RPCTestSetup(t)
	defer CleanRPCTest(t, engerino)
	s := RPCServer{Engine: engerino}
	err := s.GetHistoricTrades(&gctrpc.GetSavedTradesRequest{}, nil)
	if !errors.Is(err, errInvalidArguments) {
		t.Error(err)
	}
	err = s.GetHistoricTrades(&gctrpc.GetSavedTradesRequest{
		Exchange: fakeExchangeName,
		Pair: &gctrpc.CurrencyPair{
			Delimiter: currency.DashDelimiter,
			Base:      currency.BTC.String(),
			Quote:     currency.USD.String(),
		},
		AssetType: asset.Spot.String(),
		Start:     time.Date(2020, 0, 0, 0, 0, 0, 0, time.UTC).Format(common.SimpleTimeFormat),
		End:       time.Date(2020, 0, 0, 1, 0, 0, 0, time.UTC).Format(common.SimpleTimeFormat),
	}, nil)
	if !errors.Is(err, ErrExchangeNotFound) {
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
		End:       time.Date(2020, 0, 0, 1, 0, 0, 0, time.UTC).Format(common.SimpleTimeFormat),
	}, &dummyServer{})
	if err != common.ErrFunctionNotSupported {
		t.Error(err)
	}
}

func TestGetAccountInfo(t *testing.T) {
	t.Parallel()
	em := SetupExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	if err != nil {
		t.Fatal(err)
	}
	b := exch.GetBase()
	b.Name = fakeExchangeName
	b.Enabled = true
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		AssetEnabled: convert.BoolPtr(true),
	}
	fakeExchange := fExchange{
		IBotExchange: exch,
	}
	em.Add(fakeExchange)
	s := RPCServer{Engine: &Engine{ExchangeManager: em}}
	_, err = s.GetAccountInfo(context.Background(), &gctrpc.GetAccountInfoRequest{Exchange: fakeExchangeName, AssetType: asset.Spot.String()})
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected '%v'", err, nil)
	}
}

func TestUpdateAccountInfo(t *testing.T) {
	t.Parallel()
	em := SetupExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	if err != nil {
		t.Fatal(err)
	}
	b := exch.GetBase()
	b.Name = fakeExchangeName
	b.Enabled = true
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		AssetEnabled: convert.BoolPtr(true),
	}
	fakeExchange := fExchange{
		IBotExchange: exch,
	}
	em.Add(fakeExchange)
	s := RPCServer{Engine: &Engine{ExchangeManager: em}}

	_, err = s.GetAccountInfo(context.Background(), &gctrpc.GetAccountInfoRequest{Exchange: fakeExchangeName, AssetType: asset.Spot.String()})
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected '%v'", err, nil)
	}

	_, err = s.UpdateAccountInfo(context.Background(), &gctrpc.GetAccountInfoRequest{Exchange: fakeExchangeName, AssetType: asset.Futures.String()})
	if !errors.Is(err, errAssetTypeDisabled) {
		t.Errorf("received '%v', expected '%v'", err, errAssetTypeDisabled)
	}

	_, err = s.UpdateAccountInfo(context.Background(), &gctrpc.GetAccountInfoRequest{
		Exchange:  fakeExchangeName,
		AssetType: asset.Spot.String(),
	})
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected '%v'", err, nil)
	}
}

func TestGetOrders(t *testing.T) {
	t.Parallel()
	exchName := "Binance"
	engerino := &Engine{}
	em := SetupExchangeManager()
	exch, err := em.NewExchangeByName(exchName)
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	b := exch.GetBase()
	cp := currency.NewPair(currency.BTC, currency.USDT)
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		Available:     currency.Pairs{cp},
		Enabled:       currency.Pairs{cp},
		AssetEnabled:  convert.BoolPtr(true),
		ConfigFormat:  &currency.PairFormat{Uppercase: true},
		RequestFormat: &currency.PairFormat{Uppercase: true}}
	em.Add(exch)
	var wg sync.WaitGroup
	om, err := SetupOrderManager(em, engerino.CommunicationsManager, &wg, false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected '%v'", err, nil)
	}
	om.started = 1
	s := RPCServer{Engine: &Engine{ExchangeManager: em, OrderManager: om}}

	p := &gctrpc.CurrencyPair{
		Delimiter: "-",
		Base:      currency.BTC.String(),
		Quote:     currency.USDT.String(),
	}

	_, err = s.GetOrders(context.Background(), nil)
	if !errors.Is(err, errInvalidArguments) {
		t.Errorf("received '%v', expected '%v'", err, errInvalidArguments)
	}

	_, err = s.GetOrders(context.Background(), &gctrpc.GetOrdersRequest{
		AssetType: asset.Spot.String(),
		Pair:      p,
	})
	if !errors.Is(err, ErrExchangeNameIsEmpty) {
		t.Errorf("received '%v', expected '%v'", ErrExchangeNameIsEmpty, err)
	}

	_, err = s.GetOrders(context.Background(), &gctrpc.GetOrdersRequest{
		Exchange:  "bruh",
		AssetType: asset.Spot.String(),
		Pair:      p,
	})
	if !errors.Is(err, ErrExchangeNotFound) {
		t.Errorf("received '%v', expected '%v'", ErrExchangeNotFound, err)
	}

	_, err = s.GetOrders(context.Background(), &gctrpc.GetOrdersRequest{
		Exchange:  exchName,
		AssetType: asset.Spot.String(),
	})
	if !errors.Is(err, errCurrencyPairUnset) {
		t.Errorf("received '%v', expected '%v'", err, errCurrencyPairUnset)
	}

	_, err = s.GetOrders(context.Background(), &gctrpc.GetOrdersRequest{
		Exchange: exchName,
		Pair:     p,
	})
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Errorf("received '%v', expected '%v'", err, asset.ErrNotSupported)
	}

	_, err = s.GetOrders(context.Background(), &gctrpc.GetOrdersRequest{
		Exchange:  exchName,
		AssetType: asset.Spot.String(),
		Pair:      p,
		StartDate: time.Now().UTC().Add(time.Second).Format(common.SimpleTimeFormat),
		EndDate:   time.Now().UTC().Add(-time.Hour).Format(common.SimpleTimeFormat),
	})
	if !errors.Is(err, common.ErrStartAfterTimeNow) {
		t.Errorf("received %v, expected %v", err, common.ErrStartAfterTimeNow)
	}

	_, err = s.GetOrders(context.Background(), &gctrpc.GetOrdersRequest{
		Exchange:  exchName,
		AssetType: asset.Spot.String(),
		Pair:      p,
		StartDate: time.Now().UTC().Add(-time.Hour).Format(common.SimpleTimeFormat),
		EndDate:   time.Now().UTC().Add(time.Hour).Format(common.SimpleTimeFormat),
	})
	if !errors.Is(err, exchange.ErrCredentialsAreEmpty) {
		t.Errorf("received '%v', expected '%v'", err, exchange.ErrCredentialsAreEmpty)
	}

	b.SetCredentials("test", "test", "", "", "", "")
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
	t.Parallel()
	exchName := "Binance"
	engerino := &Engine{}
	em := SetupExchangeManager()
	exch, err := em.NewExchangeByName(exchName)
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	b := exch.GetBase()
	cp := currency.NewPair(currency.BTC, currency.USDT)
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		Available:     currency.Pairs{cp},
		Enabled:       currency.Pairs{cp},
		AssetEnabled:  convert.BoolPtr(true),
		ConfigFormat:  &currency.PairFormat{Uppercase: true},
		RequestFormat: &currency.PairFormat{Uppercase: true}}
	em.Add(exch)
	var wg sync.WaitGroup
	om, err := SetupOrderManager(em, engerino.CommunicationsManager, &wg, false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected '%v'", err, nil)
	}
	om.started = 1
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected '%v'", err, nil)
	}
	s := RPCServer{Engine: &Engine{ExchangeManager: em, OrderManager: om}}
	p := &gctrpc.CurrencyPair{
		Delimiter: "-",
		Base:      "BTC",
		Quote:     "USDT",
	}

	_, err = s.GetOrder(context.Background(), nil)
	if !errors.Is(err, errInvalidArguments) {
		t.Errorf("received '%v', expected '%v'", err, errInvalidArguments)
	}

	_, err = s.GetOrder(context.Background(), &gctrpc.GetOrderRequest{
		Exchange: "test123",
		OrderId:  "",
		Pair:     p,
		Asset:    "spot",
	})
	if !errors.Is(err, ErrExchangeNotFound) {
		t.Errorf("received '%v', expected '%v'", err, ErrExchangeNotFound)
	}

	_, err = s.GetOrder(context.Background(), &gctrpc.GetOrderRequest{
		Exchange: exchName,
		OrderId:  "",
		Pair:     nil,
		Asset:    "",
	})
	if !errors.Is(err, errCurrencyPairUnset) {
		t.Errorf("received '%v', expected '%v'", err, errCurrencyPairUnset)
	}

	_, err = s.GetOrder(context.Background(), &gctrpc.GetOrderRequest{
		Exchange: exchName,
		OrderId:  "",
		Pair:     p,
		Asset:    "",
	})
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Errorf("received '%v', expected '%v'", err, asset.ErrNotSupported)
	}

	_, err = s.GetOrder(context.Background(), &gctrpc.GetOrderRequest{
		Exchange: exchName,
		OrderId:  "",
		Pair:     p,
		Asset:    asset.Spot.String(),
	})
	if !errors.Is(err, ErrOrderIDCannotBeEmpty) {
		t.Errorf("received '%v', expected '%v'", err, ErrOrderIDCannotBeEmpty)
	}
	_, err = s.GetOrder(context.Background(), &gctrpc.GetOrderRequest{
		Exchange: exchName,
		OrderId:  "1234",
		Pair:     p,
		Asset:    asset.Spot.String(),
	})
	if !errors.Is(err, exchange.ErrCredentialsAreEmpty) {
		t.Errorf("received '%v', expected '%v'", err, exchange.ErrCredentialsAreEmpty)
	}
}

func TestCheckVars(t *testing.T) {
	t.Parallel()
	var e exchange.IBotExchange
	err := checkParams("Binance", e, asset.Spot, currency.NewPair(currency.BTC, currency.USDT))
	if !errors.Is(err, errExchangeNotLoaded) {
		t.Errorf("expected %v, got %v", errExchangeNotLoaded, err)
	}

	e = &binance.Binance{}
	err = checkParams("Binance", e, asset.Spot, currency.NewPair(currency.BTC, currency.USDT))
	if !errors.Is(err, errExchangeNotEnabled) {
		t.Errorf("expected %v, got %v", errExchangeNotEnabled, err)
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

	err = e.GetBase().CurrencyPairs.EnablePair(
		asset.Spot,
		currency.Pair{Delimiter: currency.DashDelimiter, Base: currency.BTC, Quote: currency.USDT},
	)
	if err != nil {
		t.Error(err)
	}

	err = checkParams("Binance", e, asset.Spot, currency.NewPair(currency.BTC, currency.USDT))
	if err != nil {
		t.Error(err)
	}
}

func TestParseEvents(t *testing.T) {
	t.Parallel()
	var exchangeName = "Binance"
	var testData []*withdraw.Response
	for x := 0; x < 5; x++ {
		test := fmt.Sprintf("test-%v", x)
		resp := &withdraw.Response{
			ID: withdraw.DryRunID,
			Exchange: withdraw.ExchangeResponse{
				Name:   test,
				ID:     test,
				Status: test,
			},
			RequestDetails: withdraw.Request{
				Exchange:    test,
				Description: test,
				Amount:      1.0,
			},
		}
		if x%2 == 0 {
			resp.RequestDetails.Currency = currency.AUD
			resp.RequestDetails.Type = 1
			resp.RequestDetails.Fiat = withdraw.FiatRequest{
				Bank: banking.Account{
					Enabled:             false,
					ID:                  fmt.Sprintf("test-%v", x),
					BankName:            fmt.Sprintf("test-%v-bank", x),
					AccountName:         "hello",
					AccountNumber:       fmt.Sprintf("test-%v", x),
					BSBNumber:           "123456",
					SupportedCurrencies: "BTC-AUD",
					SupportedExchanges:  exchangeName,
				},
			}
		} else {
			resp.RequestDetails.Currency = currency.BTC
			resp.RequestDetails.Type = 0
			resp.RequestDetails.Crypto.Address = test
			resp.RequestDetails.Crypto.FeeAmount = 0
			resp.RequestDetails.Crypto.AddressTag = test
		}
		testData = append(testData, resp)
	}
	v := parseMultipleEvents(testData)
	if reflect.TypeOf(v).String() != "*gctrpc.WithdrawalEventsByExchangeResponse" {
		t.Fatal("expected type to be *gctrpc.WithdrawalEventsByExchangeResponse")
	}
	if testData == nil || len(testData) < 2 {
		t.Fatal("expected at least 2")
	}

	v = parseSingleEvents(testData[0])
	if reflect.TypeOf(v).String() != "*gctrpc.WithdrawalEventsByExchangeResponse" {
		t.Fatal("expected type to be *gctrpc.WithdrawalEventsByExchangeResponse")
	}

	v = parseSingleEvents(testData[1])
	if v.Event[0].Request.Type != 0 {
		t.Fatal("Expected second entry in slice to return a Request.Type of Crypto")
	}
}

func TestRPCServerUpsertDataHistoryJob(t *testing.T) {
	t.Parallel()
	m, _ := createDHM(t)
	em := SetupExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	b := exch.GetBase()
	cp := currency.NewPair(currency.BTC, currency.USD)
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		Available:    currency.Pairs{cp},
		Enabled:      currency.Pairs{cp},
		AssetEnabled: convert.BoolPtr(true)}
	em.Add(exch)
	s := RPCServer{Engine: &Engine{dataHistoryManager: m, ExchangeManager: em}}
	_, err = s.UpsertDataHistoryJob(context.Background(), nil)
	if !errors.Is(err, errNilRequestData) {
		t.Errorf("received %v, expected %v", err, errNilRequestData)
	}

	_, err = s.UpsertDataHistoryJob(context.Background(), &gctrpc.UpsertDataHistoryJobRequest{})
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Errorf("received %v, expected %v", err, asset.ErrNotSupported)
	}

	job := &gctrpc.UpsertDataHistoryJobRequest{
		Nickname: "hellomoto",
		Exchange: testExchange,
		Asset:    asset.Spot.String(),
		Pair: &gctrpc.CurrencyPair{
			Delimiter: "-",
			Base:      "BTC",
			Quote:     "USD",
		},
		StartDate:        time.Now().Add(-time.Hour * 24).Format(common.SimpleTimeFormat),
		EndDate:          time.Now().Format(common.SimpleTimeFormat),
		Interval:         int64(kline.OneHour.Duration()),
		RequestSizeLimit: 10,
		DataType:         int64(dataHistoryCandleDataType),
		MaxRetryAttempts: 3,
		BatchSize:        500,
	}

	_, err = s.UpsertDataHistoryJob(context.Background(), job)
	if !errors.Is(err, nil) {
		t.Errorf("received %v, expected %v", err, nil)
	}
}

func TestGetDataHistoryJobDetails(t *testing.T) {
	t.Parallel()
	m, _ := createDHM(t)
	s := RPCServer{Engine: &Engine{dataHistoryManager: m}}

	dhj := &DataHistoryJob{
		Nickname:  "TestGetDataHistoryJobDetails",
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USD),
		StartDate: time.Now().UTC().Add(-time.Minute * 2),
		EndDate:   time.Now().UTC(),
		Interval:  kline.OneMin,
	}
	err := m.UpsertJob(dhj, false)
	if !errors.Is(err, nil) {
		t.Errorf("received %v, expected %v", err, nil)
	}

	_, err = s.GetDataHistoryJobDetails(context.Background(), nil)
	if !errors.Is(err, errNilRequestData) {
		t.Errorf("received %v, expected %v", err, errNilRequestData)
	}

	_, err = s.GetDataHistoryJobDetails(context.Background(), &gctrpc.GetDataHistoryJobDetailsRequest{})
	if !errors.Is(err, errNicknameIDUnset) {
		t.Errorf("received %v, expected %v", err, errNicknameIDUnset)
	}

	_, err = s.GetDataHistoryJobDetails(context.Background(), &gctrpc.GetDataHistoryJobDetailsRequest{Id: "123", Nickname: "123"})
	if !errors.Is(err, errOnlyNicknameOrID) {
		t.Errorf("received %v, expected %v", err, errOnlyNicknameOrID)
	}

	_, err = s.GetDataHistoryJobDetails(context.Background(), &gctrpc.GetDataHistoryJobDetailsRequest{Nickname: "TestGetDataHistoryJobDetails"})
	if !errors.Is(err, nil) {
		t.Errorf("received %v, expected %v", err, nil)
	}

	_, err = s.GetDataHistoryJobDetails(context.Background(), &gctrpc.GetDataHistoryJobDetailsRequest{Id: dhj.ID.String()})
	if !errors.Is(err, nil) {
		t.Errorf("received %v, expected %v", err, nil)
	}

	resp, err := s.GetDataHistoryJobDetails(context.Background(), &gctrpc.GetDataHistoryJobDetailsRequest{Nickname: "TestGetDataHistoryJobDetails", FullDetails: true})
	if !errors.Is(err, nil) {
		t.Fatalf("received %v, expected %v", err, nil)
	}
	if resp == nil { //nolint:staticcheck,nolintlint // SA5011 Ignore the nil warnings
		t.Fatal("expected job")
	}
	if !strings.EqualFold(resp.Nickname, "TestGetDataHistoryJobDetails") { //nolint:nolintlint,staticcheck // SA5011 Ignore the nil warnings
		t.Errorf("received %v, expected %v", resp.Nickname, "TestGetDataHistoryJobDetails")
	}
}

func TestSetDataHistoryJobStatus(t *testing.T) {
	t.Parallel()
	m, j := createDHM(t)
	s := RPCServer{Engine: &Engine{dataHistoryManager: m}}

	dhj := &DataHistoryJob{
		Nickname:  "TestDeleteDataHistoryJob",
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USD),
		StartDate: time.Now().UTC().Add(-time.Minute * 2),
		EndDate:   time.Now().UTC(),
		Interval:  kline.OneMin,
	}
	err := m.UpsertJob(dhj, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received %v, expected %v", err, nil)
	}
	_, err = s.SetDataHistoryJobStatus(context.Background(), nil)
	if !errors.Is(err, errNilRequestData) {
		t.Errorf("received %v, expected %v", err, errNilRequestData)
	}

	_, err = s.SetDataHistoryJobStatus(context.Background(), &gctrpc.SetDataHistoryJobStatusRequest{})
	if !errors.Is(err, errNicknameIDUnset) {
		t.Errorf("received %v, expected %v", err, errNicknameIDUnset)
	}

	_, err = s.SetDataHistoryJobStatus(context.Background(), &gctrpc.SetDataHistoryJobStatusRequest{Id: "123", Nickname: "123"})
	if !errors.Is(err, errOnlyNicknameOrID) {
		t.Errorf("received %v, expected %v", err, errOnlyNicknameOrID)
	}

	id := dhj.ID
	_, err = s.SetDataHistoryJobStatus(context.Background(), &gctrpc.SetDataHistoryJobStatusRequest{Nickname: "TestDeleteDataHistoryJob", Status: int64(dataHistoryStatusRemoved)})
	if !errors.Is(err, nil) {
		t.Errorf("received %v, expected %v", err, nil)
	}
	dhj.ID = id
	j.Status = int64(dataHistoryStatusActive)
	_, err = s.SetDataHistoryJobStatus(context.Background(), &gctrpc.SetDataHistoryJobStatusRequest{Id: id.String(), Status: int64(dataHistoryStatusRemoved)})
	if !errors.Is(err, nil) {
		t.Errorf("received %v, expected %v", err, nil)
	}
	_, err = s.SetDataHistoryJobStatus(context.Background(), &gctrpc.SetDataHistoryJobStatusRequest{Id: id.String(), Status: int64(dataHistoryStatusActive)})
	if !errors.Is(err, errBadStatus) {
		t.Errorf("received %v, expected %v", err, errBadStatus)
	}
	j.Status = int64(dataHistoryStatusActive)
	_, err = s.SetDataHistoryJobStatus(context.Background(), &gctrpc.SetDataHistoryJobStatusRequest{Id: id.String(), Status: int64(dataHistoryStatusPaused)})
	if !errors.Is(err, nil) {
		t.Errorf("received %v, expected %v", err, nil)
	}
	if j.Status != int64(dataHistoryStatusPaused) {
		t.Errorf("received %v, expected %v", dataHistoryStatus(j.Status), dataHistoryStatusPaused)
	}
}

func TestGetActiveDataHistoryJobs(t *testing.T) {
	t.Parallel()
	m, _ := createDHM(t)
	s := RPCServer{Engine: &Engine{dataHistoryManager: m}}

	dhj := &DataHistoryJob{
		Nickname:  "TestGetActiveDataHistoryJobs",
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USD),
		StartDate: time.Now().UTC().Add(-time.Minute * 2),
		EndDate:   time.Now().UTC(),
		Interval:  kline.OneMin,
	}

	if err := m.UpsertJob(dhj, false); !errors.Is(err, nil) {
		t.Fatalf("received %v, expected %v", err, nil)
	}

	r, err := s.GetActiveDataHistoryJobs(context.Background(), nil)
	if !errors.Is(err, nil) {
		t.Fatalf("received %v, expected %v", err, nil)
	}
	if len(r.Results) != 1 {
		t.Fatalf("received %v, expected %v", len(r.Results), 1)
	}
}

func TestGetDataHistoryJobsBetween(t *testing.T) {
	t.Parallel()
	m, _ := createDHM(t)
	s := RPCServer{Engine: &Engine{dataHistoryManager: m}}

	dhj := &DataHistoryJob{
		Nickname:  "GetDataHistoryJobsBetween",
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USD),
		StartDate: time.Now().UTC().Add(-time.Minute * 2),
		EndDate:   time.Now().UTC(),
		Interval:  kline.OneMin,
	}

	_, err := s.GetDataHistoryJobsBetween(context.Background(), nil)
	if !errors.Is(err, errNilRequestData) {
		t.Fatalf("received %v, expected %v", err, errNilRequestData)
	}

	_, err = s.GetDataHistoryJobsBetween(context.Background(), &gctrpc.GetDataHistoryJobsBetweenRequest{
		StartDate: time.Now().UTC().Add(time.Minute).Format(common.SimpleTimeFormat),
		EndDate:   time.Now().UTC().Format(common.SimpleTimeFormat),
	})
	if !errors.Is(err, common.ErrStartAfterTimeNow) {
		t.Fatalf("received %v, expected %v", err, common.ErrStartAfterTimeNow)
	}

	err = m.UpsertJob(dhj, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received %v, expected %v", err, nil)
	}

	r, err := s.GetDataHistoryJobsBetween(context.Background(), &gctrpc.GetDataHistoryJobsBetweenRequest{
		StartDate: time.Now().Add(-time.Minute).UTC().Format(common.SimpleTimeFormat),
		EndDate:   time.Now().Add(time.Minute).UTC().Format(common.SimpleTimeFormat),
	})
	if !errors.Is(err, nil) {
		t.Errorf("received %v, expected %v", err, nil)
	}
	if len(r.Results) != 1 {
		t.Errorf("received %v, expected %v", len(r.Results), 1)
	}
}

func TestGetDataHistoryJobSummary(t *testing.T) {
	t.Parallel()
	m, _ := createDHM(t)
	s := RPCServer{Engine: &Engine{dataHistoryManager: m}}

	dhj := &DataHistoryJob{
		Nickname:  "TestGetDataHistoryJobSummary",
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USD),
		StartDate: time.Now().UTC().Add(-time.Minute * 2),
		EndDate:   time.Now().UTC(),
		Interval:  kline.OneMin,
	}

	err := m.UpsertJob(dhj, false)
	if !errors.Is(err, nil) {
		t.Errorf("received %v, expected %v", err, nil)
	}

	_, err = s.GetDataHistoryJobSummary(context.Background(), nil)
	if !errors.Is(err, errNilRequestData) {
		t.Errorf("received %v, expected %v", err, errNilRequestData)
	}

	_, err = s.GetDataHistoryJobSummary(context.Background(), &gctrpc.GetDataHistoryJobDetailsRequest{})
	if !errors.Is(err, errNicknameUnset) {
		t.Errorf("received %v, expected %v", err, errNicknameUnset)
	}

	resp, err := s.GetDataHistoryJobSummary(context.Background(), &gctrpc.GetDataHistoryJobDetailsRequest{Nickname: "TestGetDataHistoryJobSummary"})
	if !errors.Is(err, nil) {
		t.Errorf("received %v, expected %v", err, nil)
	}
	if resp == nil { //nolint:staticcheck,nolintlint // SA5011 Ignore the nil warnings
		t.Fatal("expected job")
	}
	if !strings.EqualFold(resp.Nickname, "TestGetDataHistoryJobSummary") { //nolint:staticcheck,nolintlint // SA5011 Ignore the nil warnings
		t.Fatalf("received %v, expected %v", "TestGetDataHistoryJobSummary", resp.Nickname)
	}
	if resp.ResultSummaries == nil { //nolint:staticcheck,nolintlint // SA5011 Ignore the nil warnings
		t.Fatalf("received %v, expected %v", nil, "result summaries slice")
	}
}

func TestGetManagedOrders(t *testing.T) {
	exchName := "Binance"
	engerino := &Engine{}
	em := SetupExchangeManager()
	exch, err := em.NewExchangeByName(exchName)
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	b := exch.GetBase()
	cp := currency.NewPair(currency.BTC, currency.USDT)
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		Available:     currency.Pairs{cp},
		Enabled:       currency.Pairs{cp},
		AssetEnabled:  convert.BoolPtr(true),
		ConfigFormat:  &currency.PairFormat{Uppercase: true},
		RequestFormat: &currency.PairFormat{Uppercase: true}}
	em.Add(exch)
	var wg sync.WaitGroup
	om, err := SetupOrderManager(em, engerino.CommunicationsManager, &wg, false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected '%v'", err, nil)
	}
	om.started = 1
	s := RPCServer{Engine: &Engine{ExchangeManager: em, OrderManager: om}}

	p := &gctrpc.CurrencyPair{
		Delimiter: "-",
		Base:      currency.BTC.String(),
		Quote:     currency.USDT.String(),
	}

	_, err = s.GetManagedOrders(context.Background(), nil)
	if !errors.Is(err, errInvalidArguments) {
		t.Errorf("received '%v', expected '%v'", err, errInvalidArguments)
	}

	_, err = s.GetManagedOrders(context.Background(), &gctrpc.GetOrdersRequest{
		AssetType: asset.Spot.String(),
		Pair:      p,
	})
	if !errors.Is(err, ErrExchangeNameIsEmpty) {
		t.Errorf("received '%v', expected '%v'", ErrExchangeNameIsEmpty, err)
	}

	_, err = s.GetManagedOrders(context.Background(), &gctrpc.GetOrdersRequest{
		Exchange:  "bruh",
		AssetType: asset.Spot.String(),
		Pair:      p,
	})
	if !errors.Is(err, ErrExchangeNotFound) {
		t.Errorf("received '%v', expected '%v'", ErrExchangeNotFound, err)
	}

	_, err = s.GetManagedOrders(context.Background(), &gctrpc.GetOrdersRequest{
		Exchange:  exchName,
		AssetType: asset.Spot.String(),
	})
	if !errors.Is(err, errCurrencyPairUnset) {
		t.Errorf("received '%v', expected '%v'", err, errCurrencyPairUnset)
	}

	_, err = s.GetManagedOrders(context.Background(), &gctrpc.GetOrdersRequest{
		Exchange: exchName,
		Pair:     p,
	})
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Errorf("received '%v', expected '%v'", err, asset.ErrNotSupported)
	}

	o := order.Detail{
		Price:           100000,
		Amount:          0.002,
		Exchange:        "Binance",
		InternalOrderID: "",
		ID:              "",
		ClientOrderID:   "",
		AccountID:       "",
		ClientID:        "",
		WalletAddress:   "",
		Type:            order.Limit,
		Side:            "SELL",
		Status:          order.New,
		AssetType:       asset.Spot,
		Pair:            currency.NewPair(currency.BTC, currency.USDT),
	}
	err = om.Add(&o)
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	oo, err := s.GetManagedOrders(context.Background(), &gctrpc.GetOrdersRequest{
		Exchange:  exchName,
		AssetType: "spot",
		Pair:      p,
	})
	if err != nil {
		t.Errorf("non expected Error: %v", err)
	} else if oo == nil || len(oo.GetOrders()) != 1 {
		t.Errorf("unexpected order result: %v", oo)
	}
}

func TestRPCServer_unixTimestamp(t *testing.T) {
	t.Parallel()

	s := RPCServer{
		Engine: &Engine{
			Config: &config.Config{
				RemoteControl: config.RemoteControlConfig{
					GRPC: config.GRPCConfig{
						TimeInNanoSeconds: false,
					},
				},
			},
		},
	}
	const sec = 1618888141
	const nsec = 2
	x := time.Unix(sec, nsec)

	timestampSeconds := s.unixTimestamp(x)
	if timestampSeconds != sec {
		t.Errorf("have %d, want %d", timestampSeconds, sec)
	}

	s.Config.RemoteControl.GRPC.TimeInNanoSeconds = true
	timestampNanos := s.unixTimestamp(x)
	if want := int64(sec*1_000_000_000 + nsec); timestampNanos != want {
		t.Errorf("have %d, want %d", timestampSeconds, want)
	}
}

func TestRPCServer_GetTicker_LastUpdatedNanos(t *testing.T) {
	// Make a dummy pair we'll be using for this test.
	pair := currency.NewPairWithDelimiter("XXXXX", "YYYYY", "")

	// Create a mock-up RPCServer and add our newly made pair to its list of
	// available and enabled pairs.
	server := RPCServer{Engine: RPCTestSetup(t)}
	exch, err := server.GetExchangeByName(testExchange)
	if err != nil {
		t.Fatal(err)
	}
	b := exch.GetBase()
	b.CurrencyPairs.Pairs[asset.Spot].Available = append(
		b.CurrencyPairs.Pairs[asset.Spot].Available,
		pair,
	)
	b.CurrencyPairs.Pairs[asset.Spot].Enabled = append(
		b.CurrencyPairs.Pairs[asset.Spot].Enabled,
		pair,
	)

	// Push a mock-up ticker.
	now := time.Now()
	err = ticker.ProcessTicker(&ticker.Price{
		ExchangeName: testExchange,
		Pair:         pair,
		AssetType:    asset.Spot,
		LastUpdated:  now,

		Open:  69,
		High:  96,
		Low:   169,
		Close: 196,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Prepare a ticker request.
	request := &gctrpc.GetTickerRequest{
		Exchange: testExchange,
		Pair: &gctrpc.CurrencyPair{
			Delimiter: pair.Delimiter,
			Base:      pair.Base.String(),
			Quote:     pair.Quote.String(),
		},
		AssetType: asset.Spot.String(),
	}

	// Check if timestamp returned is in seconds if !TimeInNanoSeconds.
	server.Config.RemoteControl.GRPC.TimeInNanoSeconds = false
	one, err := server.GetTicker(context.Background(), request)
	if err != nil {
		t.Error(err)
	}
	if want := now.Unix(); one.LastUpdated != want {
		t.Errorf("have %d, want %d", one.LastUpdated, want)
	}

	// Check if timestamp returned is in nanoseconds if TimeInNanoSeconds.
	server.Config.RemoteControl.GRPC.TimeInNanoSeconds = true
	two, err := server.GetTicker(context.Background(), request)
	if err != nil {
		t.Error(err)
	}
	if want := now.UnixNano(); two.LastUpdated != want {
		t.Errorf("have %d, want %d", two.LastUpdated, want)
	}
}

func TestUpdateDataHistoryJobPrerequisite(t *testing.T) {
	t.Parallel()
	m, _ := createDHM(t)
	s := RPCServer{Engine: &Engine{dataHistoryManager: m}}
	_, err := s.UpdateDataHistoryJobPrerequisite(context.Background(), nil)
	if !errors.Is(err, errNilRequestData) {
		t.Errorf("received %v, expected %v", err, errNilRequestData)
	}

	_, err = s.UpdateDataHistoryJobPrerequisite(context.Background(), &gctrpc.UpdateDataHistoryJobPrerequisiteRequest{})
	if !errors.Is(err, errNicknameUnset) {
		t.Errorf("received %v, expected %v", err, errNicknameUnset)
	}

	_, err = s.UpdateDataHistoryJobPrerequisite(context.Background(), &gctrpc.UpdateDataHistoryJobPrerequisiteRequest{
		Nickname: "test456",
	})
	if !errors.Is(err, nil) {
		t.Errorf("received %v, expected %v", err, nil)
	}

	_, err = s.UpdateDataHistoryJobPrerequisite(context.Background(), &gctrpc.UpdateDataHistoryJobPrerequisiteRequest{
		Nickname:                "test456",
		PrerequisiteJobNickname: "test123",
	})
	if !errors.Is(err, nil) {
		t.Errorf("received %v, expected %v", err, nil)
	}
}

func TestCurrencyStateGetAll(t *testing.T) {
	t.Parallel()
	_, err := (&RPCServer{Engine: &Engine{}}).CurrencyStateGetAll(context.Background(),
		&gctrpc.CurrencyStateGetAllRequest{Exchange: fakeExchangeName})
	if !errors.Is(err, ErrSubSystemNotStarted) {
		t.Errorf("received %v, expected %v", err, ErrSubSystemNotStarted)
	}
}

func TestCurrencyStateWithdraw(t *testing.T) {
	t.Parallel()
	_, err := (&RPCServer{
		Engine: &Engine{},
	}).CurrencyStateWithdraw(context.Background(),
		&gctrpc.CurrencyStateWithdrawRequest{
			Exchange: "wow"})
	if !errors.Is(err, ErrSubSystemNotStarted) {
		t.Fatalf("received: %v, but expected: %v", err, ErrSubSystemNotStarted)
	}
}

func TestCurrencyStateDeposit(t *testing.T) {
	t.Parallel()
	_, err := (&RPCServer{
		Engine: &Engine{},
	}).CurrencyStateDeposit(context.Background(),
		&gctrpc.CurrencyStateDepositRequest{Exchange: "wow"})
	if !errors.Is(err, ErrSubSystemNotStarted) {
		t.Fatalf("received: %v, but expected: %v", err, ErrSubSystemNotStarted)
	}
}

func TestCurrencyStateTrading(t *testing.T) {
	t.Parallel()
	_, err := (&RPCServer{
		Engine: &Engine{},
	}).CurrencyStateTrading(context.Background(),
		&gctrpc.CurrencyStateTradingRequest{Exchange: "wow"})
	if !errors.Is(err, ErrSubSystemNotStarted) {
		t.Fatalf("received: %v, but expected: %v", err, ErrSubSystemNotStarted)
	}
}

func TestCurrencyStateTradingPair(t *testing.T) {
	t.Parallel()
	em := SetupExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	if err != nil {
		t.Fatal(err)
	}
	b := exch.GetBase()
	b.Name = fakeExchangeName
	b.Enabled = true

	cp, err := currency.NewPairFromString("btc-usd")
	if err != nil {
		t.Fatal(err)
	}

	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		AssetEnabled: convert.BoolPtr(true),
		ConfigFormat: &currency.PairFormat{},
		Available:    currency.Pairs{cp},
		Enabled:      currency.Pairs{cp},
	}
	fakeExchange := fExchange{
		IBotExchange: exch,
	}
	em.Add(fakeExchange)
	s := RPCServer{Engine: &Engine{ExchangeManager: em,
		currencyStateManager: &CurrencyStateManager{started: 1, iExchangeManager: em}}}

	_, err = s.CurrencyStateTradingPair(context.Background(),
		&gctrpc.CurrencyStateTradingPairRequest{
			Exchange: fakeExchangeName,
			Pair:     "btc-usd",
			Asset:    "spot",
		})
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v, but expected: %v", err, nil)
	}
}

func TestGetFuturesPositions(t *testing.T) {
	t.Parallel()
	em := SetupExchangeManager()
	exch, err := em.NewExchangeByName("ftx")
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	b := exch.GetBase()
	b.Name = fakeExchangeName
	b.Enabled = true

	cp, err := currency.NewPairFromString("btc-perp")
	if err != nil {
		t.Fatal(err)
	}

	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Futures] = &currency.PairStore{
		AssetEnabled:  convert.BoolPtr(true),
		RequestFormat: &currency.PairFormat{Delimiter: "-"},
		ConfigFormat:  &currency.PairFormat{Delimiter: "-"},
		Available:     currency.Pairs{cp},
		Enabled:       currency.Pairs{cp},
	}
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		AssetEnabled:  convert.BoolPtr(true),
		ConfigFormat:  &currency.PairFormat{Delimiter: "/"},
		RequestFormat: &currency.PairFormat{Delimiter: "/"},
		Available:     currency.Pairs{cp},
		Enabled:       currency.Pairs{cp},
	}
	fakeExchange := fExchange{
		IBotExchange: exch,
	}
	em.Add(fakeExchange)
	var wg sync.WaitGroup
	om, err := SetupOrderManager(em, &CommunicationManager{}, &wg, false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected '%v'", err, nil)
	}
	om.started = 1
	s := RPCServer{
		Engine: &Engine{
			ExchangeManager: em,
			currencyStateManager: &CurrencyStateManager{
				started:          1,
				iExchangeManager: em,
			},
			OrderManager: om,
		},
	}

	_, err = s.GetFuturesPositions(context.Background(), &gctrpc.GetFuturesPositionsRequest{
		Exchange: fakeExchangeName,
		Asset:    asset.Futures.String(),
		Pair: &gctrpc.CurrencyPair{
			Delimiter: currency.DashDelimiter,
			Base:      cp.Base.String(),
			Quote:     cp.Quote.String(),
		},
		Verbose: true,
	})
	if !errors.Is(err, exchange.ErrCredentialsAreEmpty) {
		t.Fatalf("received '%v', expected '%v'", err, exchange.ErrCredentialsAreEmpty)
	}

	ctx := exchange.DeployCredentialsToContext(context.Background(), &exchange.Credentials{
		Key:    "wow",
		Secret: "super wow",
	})

	r, err := s.GetFuturesPositions(ctx, &gctrpc.GetFuturesPositionsRequest{
		Exchange: fakeExchangeName,
		Asset:    asset.Futures.String(),
		Pair: &gctrpc.CurrencyPair{
			Delimiter: currency.DashDelimiter,
			Base:      cp.Base.String(),
			Quote:     cp.Quote.String(),
		},
		Verbose: true,
	})
	if !errors.Is(err, nil) {
		t.Fatalf("received '%v', expected '%v'", err, nil)
	}
	if r == nil { //nolint:staticcheck,nolintlint // SA5011 Ignore the nil warnings
		t.Fatal("expected not nil response")
	}
	if len(r.Positions) != 1 { //nolint:staticcheck,nolintlint // SA5011 Ignore the nil warnings
		t.Fatal("expected 1 position")
	}
	if r.TotalOrders != 1 { //nolint:staticcheck,nolintlint // SA5011 Ignore the nil warnings
		t.Fatal("expected 1 order")
	}

	_, err = s.GetFuturesPositions(ctx, &gctrpc.GetFuturesPositionsRequest{
		Exchange: fakeExchangeName,
		Asset:    asset.Spot.String(),
		Pair: &gctrpc.CurrencyPair{
			Delimiter: currency.DashDelimiter,
			Base:      cp.Base.String(),
			Quote:     cp.Quote.String(),
		},
		Verbose: true,
	})
	if !errors.Is(err, order.ErrNotFuturesAsset) {
		t.Errorf("received '%v', expected '%v'", err, order.ErrNotFuturesAsset)
	}
}

func TestGetCollateral(t *testing.T) {
	t.Parallel()
	em := SetupExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	if err != nil {
		t.Fatal(err)
	}
	b := exch.GetBase()
	b.Name = fakeExchangeName
	b.Enabled = true

	cp, err := currency.NewPairFromString("btc-usd")
	if !errors.Is(err, nil) {
		t.Fatalf("received '%v', expected '%v'", err, nil)
	}

	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Futures] = &currency.PairStore{
		AssetEnabled: convert.BoolPtr(true),
		ConfigFormat: &currency.PairFormat{},
		Available:    currency.Pairs{cp},
		Enabled:      currency.Pairs{cp},
	}
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		AssetEnabled: convert.BoolPtr(true),
		ConfigFormat: &currency.PairFormat{},
		Available:    currency.Pairs{cp},
		Enabled:      currency.Pairs{cp},
	}
	fakeExchange := fExchange{
		IBotExchange: exch,
	}
	em.Add(fakeExchange)
	s := RPCServer{
		Engine: &Engine{
			ExchangeManager: em,
			currencyStateManager: &CurrencyStateManager{
				started: 1, iExchangeManager: em,
			},
		},
	}

	_, err = s.GetCollateral(context.Background(), &gctrpc.GetCollateralRequest{
		Exchange: fakeExchangeName,
		Asset:    asset.Futures.String(),
	})
	if !errors.Is(err, exchange.ErrCredentialsAreEmpty) {
		t.Fatalf("received '%v', expected '%v'", err, exchange.ErrCredentialsAreEmpty)
	}

	ctx := exchange.DeployCredentialsToContext(context.Background(), &exchange.Credentials{Key: "fakerino", Secret: "supafake"})

	_, err = s.GetCollateral(ctx, &gctrpc.GetCollateralRequest{
		Exchange: fakeExchangeName,
		Asset:    asset.Futures.String(),
	})
	if !errors.Is(err, errNoAccountInformation) {
		t.Fatalf("received '%v', expected '%v'", err, errNoAccountInformation)
	}

	ctx = exchange.DeployCredentialsToContext(context.Background(), &exchange.Credentials{Key: "fakerino", Secret: "supafake", SubAccount: "1337"})

	r, err := s.GetCollateral(ctx, &gctrpc.GetCollateralRequest{
		Exchange:         fakeExchangeName,
		Asset:            asset.Futures.String(),
		IncludeBreakdown: true,
	})
	if !errors.Is(err, nil) {
		t.Fatalf("received '%v', expected '%v'", err, nil)
	}
	if len(r.CurrencyBreakdown) != 3 {
		t.Errorf("expected 3 currencies, received '%v'", len(r.CurrencyBreakdown))
	}
	if r.AvailableCollateral != "1337 USD" {
		t.Errorf("received '%v' expected '1337 USD'", r.AvailableCollateral)
	}

	_, err = s.GetCollateral(ctx, &gctrpc.GetCollateralRequest{
		Exchange:         fakeExchangeName,
		Asset:            asset.Spot.String(),
		IncludeBreakdown: true,
	})
	if !errors.Is(err, order.ErrNotFuturesAsset) {
		t.Errorf("received '%v', expected '%v'", err, order.ErrNotFuturesAsset)
	}

	_, err = s.GetCollateral(ctx, &gctrpc.GetCollateralRequest{
		Exchange:         fakeExchangeName,
		Asset:            asset.Futures.String(),
		IncludeBreakdown: true,
		CalculateOffline: true,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected '%v'", err, nil)
	}
}
