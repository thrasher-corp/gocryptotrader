package engine

import (
	"context"
	"errors"
	"fmt"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
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

var errExpectedTestError = errors.New("expected test error")

// fExchange is a fake exchange with function overrides
// we're not testing an actual exchange's implemented functions
type fExchange struct {
	exchange.IBotExchange
}

func (f fExchange) GetPositionSummary(context.Context, *order.PositionSummaryRequest) (*order.PositionSummary, error) {
	leet := decimal.NewFromInt(1337)
	return &order.PositionSummary{
		MaintenanceMarginRequirement: leet,
		InitialMarginRequirement:     leet,
		EstimatedLiquidationPrice:    leet,
		CollateralUsed:               leet,
		MarkPrice:                    leet,
		CurrentSize:                  leet,
		BreakEvenPrice:               leet,
		AverageOpenPrice:             leet,
		RecentPNL:                    leet,
		MarginFraction:               leet,
		FreeCollateral:               leet,
		TotalCollateral:              leet,
	}, nil
}

func (f fExchange) GetFuturesPositions(_ context.Context, req *order.PositionsRequest) ([]order.PositionDetails, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	resp := make([]order.PositionDetails, len(req.Pairs))
	tt := time.Now()
	for i := range req.Pairs {
		resp[i] = order.PositionDetails{
			Exchange: f.GetName(),
			Asset:    req.Asset,
			Pair:     req.Pairs[i],
			Orders: []order.Detail{
				{
					Exchange:        f.GetName(),
					Price:           1337,
					Amount:          1337,
					InternalOrderID: id,
					OrderID:         "1337",
					ClientOrderID:   "1337",
					Type:            order.Market,
					Side:            order.Short,
					Status:          order.Open,
					AssetType:       req.Asset,
					Date:            tt,
					CloseTime:       tt,
					LastUpdated:     tt,
					Pair:            req.Pairs[i],
				},
			},
		}
	}
	return resp, nil
}

func (f fExchange) GetLatestFundingRate(_ context.Context, request *fundingrate.LatestRateRequest) (*fundingrate.LatestRateResponse, error) {
	leet := decimal.NewFromInt(1337)
	return &fundingrate.LatestRateResponse{
		Exchange: f.GetName(),
		Asset:    request.Asset,
		Pair:     request.Pair,
		LatestRate: fundingrate.Rate{
			Time:    time.Now(),
			Rate:    leet,
			Payment: leet,
		},
		PredictedUpcomingRate: fundingrate.Rate{
			Time:    time.Now(),
			Rate:    leet,
			Payment: leet,
		},
		TimeOfNextRate: time.Now(),
	}, nil
}

func (f fExchange) GetFundingRates(_ context.Context, request *fundingrate.RatesRequest) (*fundingrate.Rates, error) {
	leet := decimal.NewFromInt(1337)
	return &fundingrate.Rates{
		Exchange:  f.GetName(),
		Asset:     request.Asset,
		Pair:      request.Pair,
		StartDate: request.StartDate,
		EndDate:   request.EndDate,
		LatestRate: fundingrate.Rate{
			Time:    request.EndDate,
			Rate:    leet,
			Payment: leet,
		},
		PredictedUpcomingRate: fundingrate.Rate{
			Time:    request.EndDate,
			Rate:    leet,
			Payment: leet,
		},
		FundingRates: []fundingrate.Rate{
			{
				Time:    request.EndDate,
				Rate:    leet,
				Payment: leet,
			},
		},
		PaymentSum: leet,
	}, nil
}

func (f fExchange) GetHistoricCandles(_ context.Context, p currency.Pair, a asset.Item, interval kline.Interval, timeStart, _ time.Time) (*kline.Item, error) {
	return &kline.Item{
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

func generateCandles(amount int, timeStart time.Time, interval kline.Interval) []kline.Candle {
	candy := make([]kline.Candle, amount)
	for x := 0; x < amount; x++ {
		candy[x] = kline.Candle{
			Time:   timeStart,
			Open:   1337,
			High:   1337,
			Low:    1337,
			Close:  1337,
			Volume: 1337,
		}
		timeStart = timeStart.Add(interval.Duration())
	}
	return candy
}

func (f fExchange) GetHistoricCandlesExtended(_ context.Context, p currency.Pair, a asset.Item, interval kline.Interval, timeStart, _ time.Time) (*kline.Item, error) {
	if interval == 0 {
		return nil, errExpectedTestError
	}
	return &kline.Item{
		Exchange: fakeExchangeName,
		Pair:     p,
		Asset:    a,
		Interval: interval,
		Candles:  generateCandles(33, timeStart, interval),
	}, nil
}

func (f fExchange) GetMarginRatesHistory(context.Context, *margin.RateHistoryRequest) (*margin.RateHistoryResponse, error) {
	leet := decimal.NewFromInt(1337)
	rates := []margin.Rate{
		{
			Time:             time.Now(),
			MarketBorrowSize: leet,
			HourlyRate:       leet,
			HourlyBorrowRate: leet,
			LendingPayment: margin.LendingPayment{
				Payment: leet,
				Size:    leet,
			},
			BorrowCost: margin.BorrowCost{
				Cost: leet,
				Size: leet,
			},
		},
	}
	resp := &margin.RateHistoryResponse{
		Rates:              rates,
		SumBorrowCosts:     leet,
		AverageBorrowSize:  leet,
		SumLendingPayments: leet,
		AverageLendingSize: leet,
		PredictedRate:      rates[0],
		TakerFeeRate:       leet,
	}

	return resp, nil
}

func (f fExchange) FetchTicker(_ context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
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
						Currency: currency.USD,
						Total:    1337,
					},
					{
						Currency: currency.BTC,
						Total:    13337,
					},
				},
			},
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
func (f fExchange) UpdateAccountInfo(_ context.Context, a asset.Item) (account.Holdings, error) {
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
func (f fExchange) CanTradePair(_ currency.Pair, _ asset.Item) error {
	return nil
}

// CanTrade overrides interface function
func (f fExchange) CanTrade(_ currency.Code, _ asset.Item) error {
	return nil
}

// CanWithdraw overrides interface function
func (f fExchange) CanWithdraw(_ currency.Code, _ asset.Item) error {
	return nil
}

// CanDeposit overrides interface function
func (f fExchange) CanDeposit(_ currency.Code, _ asset.Item) error {
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
	dbm.dbConn.DataPath = t.TempDir()
	engerino.DatabaseManager = dbm
	var wg sync.WaitGroup
	err = dbm.Start(&wg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		err = dbm.Stop()
		if err != nil {
			t.Fatal(err)
		}
	})

	engerino.Config = &config.Config{}
	em := NewExchangeManager()
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
	err = em.Add(exch)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

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
	err = em.Add(exch)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
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
		Start:     time.Date(2020, 0, 0, 0, 0, 0, 0, time.UTC).Format(common.SimpleTimeFormatWithTimezone),
		End:       time.Date(2020, 1, 1, 1, 1, 1, 1, time.UTC).Format(common.SimpleTimeFormatWithTimezone),
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
		Start:     time.Date(2020, 0, 0, 0, 0, 0, 0, time.UTC).Format(common.SimpleTimeFormatWithTimezone),
		End:       time.Date(2020, 1, 1, 1, 1, 1, 1, time.UTC).Format(common.SimpleTimeFormatWithTimezone),
	})
	if err == nil {
		t.Error(unexpectedLackOfError)
		return
	}
	if err.Error() != "request for Bitstamp spot trade data between 2019-11-30 00:00:00 UTC and 2020-01-01 01:01:01 UTC and returned no results" {
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
		Start:     time.Date(2020, 0, 0, 0, 0, 0, 0, time.UTC).Format(common.SimpleTimeFormatWithTimezone),
		End:       time.Date(2020, 1, 1, 1, 1, 1, 1, time.UTC).Format(common.SimpleTimeFormatWithTimezone),
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
		Start:        time.Date(2020, 0, 0, 0, 0, 0, 0, time.UTC).Format(common.SimpleTimeFormatWithTimezone),
		End:          time.Date(2020, 0, 0, 1, 0, 0, 0, time.UTC).Format(common.SimpleTimeFormatWithTimezone),
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
		Start:        time.Date(2020, 0, 0, 0, 0, 0, 0, time.UTC).Format(common.SimpleTimeFormatWithTimezone),
		End:          time.Date(2020, 0, 0, 1, 0, 0, 0, time.UTC).Format(common.SimpleTimeFormatWithTimezone),
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
		Start:        time.Date(2020, 0, 0, 0, 0, 0, 0, time.UTC).Format(common.SimpleTimeFormatWithTimezone),
		End:          time.Date(2020, 0, 0, 1, 0, 0, 0, time.UTC).Format(common.SimpleTimeFormatWithTimezone),
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
		Start:        time.Date(2020, 0, 0, 0, 0, 0, 0, time.UTC).Format(common.SimpleTimeFormatWithTimezone),
		End:          time.Date(2020, 0, 0, 1, 0, 0, 0, time.UTC).Format(common.SimpleTimeFormatWithTimezone),
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
		Start:        time.Date(2020, 0, 0, 0, 0, 0, 0, time.UTC).Format(common.SimpleTimeFormatWithTimezone),
		End:          time.Date(2020, 0, 0, 1, 0, 0, 0, time.UTC).Format(common.SimpleTimeFormatWithTimezone),
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
		Start:        time.Date(2020, 0, 0, 0, 0, 0, 0, time.UTC).Format(common.SimpleTimeFormatWithTimezone),
		End:          time.Date(2020, 0, 0, 1, 0, 0, 0, time.UTC).Format(common.SimpleTimeFormatWithTimezone),
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
		Start:     defaultStart.Format(common.SimpleTimeFormatWithTimezone),
		End:       defaultEnd.Format(common.SimpleTimeFormatWithTimezone),
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
		Start:     defaultStart.Format(common.SimpleTimeFormatWithTimezone),
		End:       defaultEnd.Format(common.SimpleTimeFormatWithTimezone),
		AssetType: asset.Spot.String(),
	})
	if !errors.Is(err, ErrExchangeNotFound) {
		t.Errorf("received '%v', expected '%v'", err, ErrExchangeNotFound)
	}

	_, err = s.GetHistoricCandles(context.Background(), &gctrpc.GetHistoricCandlesRequest{
		Exchange:  testExchange,
		Start:     defaultStart.Format(common.SimpleTimeFormatWithTimezone),
		End:       defaultEnd.Format(common.SimpleTimeFormatWithTimezone),
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
		Start: "2020-01-02 15:04:05 UTC",
		End:   "2020-01-02 15:04:05 UTC",
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
		Start:        defaultStart.Format(common.SimpleTimeFormatWithTimezone),
		End:          defaultEnd.Format(common.SimpleTimeFormatWithTimezone),
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
		Start:        defaultStart.Format(common.SimpleTimeFormatWithTimezone),
		End:          defaultEnd.Format(common.SimpleTimeFormatWithTimezone),
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
		Start:        defaultStart.Format(common.SimpleTimeFormatWithTimezone),
		End:          defaultEnd.Format(common.SimpleTimeFormatWithTimezone),
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
		Start:                 defaultStart.Format(common.SimpleTimeFormatWithTimezone),
		End:                   time.Date(2020, 0, 0, 3, 0, 0, 0, time.UTC).Format(common.SimpleTimeFormatWithTimezone),
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
		Start: defaultStart.UTC().Format(common.SimpleTimeFormatWithTimezone),
		End:   defaultEnd.UTC().Format(common.SimpleTimeFormatWithTimezone),
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
		Start: defaultStart.In(time.UTC).Format(common.SimpleTimeFormatWithTimezone),
		End:   defaultEnd.In(time.UTC).Format(common.SimpleTimeFormatWithTimezone),
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
		Start: defaultStart.In(time.UTC).Format(common.SimpleTimeFormatWithTimezone),
		End:   defaultEnd.In(time.UTC).Format(common.SimpleTimeFormatWithTimezone),
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
		Start:    defaultStart.Format(common.SimpleTimeFormatWithTimezone),
		End:      defaultEnd.Format(common.SimpleTimeFormatWithTimezone),
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
		Start:    defaultStart.Format(common.SimpleTimeFormatWithTimezone),
		End:      defaultEnd.Format(common.SimpleTimeFormatWithTimezone),
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
		Start:    defaultStart.Format(common.SimpleTimeFormatWithTimezone),
		End:      defaultEnd.Format(common.SimpleTimeFormatWithTimezone),
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
	em := NewExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	b := exch.GetBase()
	b.Config = &config.Exchange{
		Features: &config.FeaturesConfig{Enabled: config.FeaturesEnabledConfig{SaveTradeData: false}},
	}
	err = em.Add(exch)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
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
		Start:     time.Date(2020, 0, 0, 0, 0, 0, 0, time.UTC).Format(common.SimpleTimeFormatWithTimezone),
		End:       time.Date(2020, 0, 0, 1, 0, 0, 0, time.UTC).Format(common.SimpleTimeFormatWithTimezone),
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
func (d *dummyServer) SendMsg(_ interface{}) error            { return nil }
func (d *dummyServer) RecvMsg(_ interface{}) error            { return nil }

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
		Start:     time.Date(2020, 0, 0, 0, 0, 0, 0, time.UTC).Format(common.SimpleTimeFormatWithTimezone),
		End:       time.Date(2020, 0, 0, 1, 0, 0, 0, time.UTC).Format(common.SimpleTimeFormatWithTimezone),
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
		Start:     time.Date(2020, 0, 0, 0, 0, 0, 0, time.UTC).Format(common.SimpleTimeFormatWithTimezone),
		End:       time.Date(2020, 0, 0, 1, 0, 0, 0, time.UTC).Format(common.SimpleTimeFormatWithTimezone),
	}, &dummyServer{})
	if err != common.ErrFunctionNotSupported {
		t.Error(err)
	}
}

func TestGetAccountInfo(t *testing.T) {
	t.Parallel()
	em := NewExchangeManager()
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
	err = em.Add(fakeExchange)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
	s := RPCServer{Engine: &Engine{ExchangeManager: em}}
	_, err = s.GetAccountInfo(context.Background(), &gctrpc.GetAccountInfoRequest{Exchange: fakeExchangeName, AssetType: asset.Spot.String()})
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected '%v'", err, nil)
	}
}

func TestUpdateAccountInfo(t *testing.T) {
	t.Parallel()
	em := NewExchangeManager()
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
	err = em.Add(fakeExchange)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
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
	em := NewExchangeManager()
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
	err = em.Add(exch)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
	var wg sync.WaitGroup
	om, err := SetupOrderManager(em, engerino.CommunicationsManager, &wg, false, false, 0)
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
		StartDate: time.Now().UTC().Add(time.Second).Format(common.SimpleTimeFormatWithTimezone),
		EndDate:   time.Now().UTC().Add(-time.Hour).Format(common.SimpleTimeFormatWithTimezone),
	})
	if !errors.Is(err, common.ErrStartAfterEnd) {
		t.Errorf("received %v, expected %v", err, common.ErrStartAfterEnd)
	}

	_, err = s.GetOrders(context.Background(), &gctrpc.GetOrdersRequest{
		Exchange:  exchName,
		AssetType: asset.Spot.String(),
		Pair:      p,
		StartDate: time.Now().UTC().Add(-time.Hour).Format(common.SimpleTimeFormatWithTimezone),
		EndDate:   time.Now().UTC().Add(time.Hour).Format(common.SimpleTimeFormatWithTimezone),
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
	em := NewExchangeManager()
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
	err = em.Add(exch)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
	var wg sync.WaitGroup
	om, err := SetupOrderManager(em, engerino.CommunicationsManager, &wg, false, false, 0)
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
			Delimiter: currency.DashDelimiter,
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

	err = e.GetBase().CurrencyPairs.StorePairs(asset.Spot, data, false)
	if err != nil {
		t.Fatal(err)
	}

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
	em := NewExchangeManager()
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
	err = em.Add(exch)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
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
		StartDate:        time.Now().Add(-time.Hour * 24).Format(common.SimpleTimeFormatWithTimezone),
		EndDate:          time.Now().Format(common.SimpleTimeFormatWithTimezone),
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
		StartDate: time.Now().UTC().Add(time.Minute).Format(common.SimpleTimeFormatWithTimezone),
		EndDate:   time.Now().UTC().Format(common.SimpleTimeFormatWithTimezone),
	})
	if !errors.Is(err, common.ErrStartAfterEnd) {
		t.Fatalf("received %v, expected %v", err, common.ErrStartAfterEnd)
	}

	err = m.UpsertJob(dhj, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received %v, expected %v", err, nil)
	}

	r, err := s.GetDataHistoryJobsBetween(context.Background(), &gctrpc.GetDataHistoryJobsBetweenRequest{
		StartDate: time.Now().Add(-time.Minute).UTC().Format(common.SimpleTimeFormatWithTimezone),
		EndDate:   time.Now().Add(time.Minute).UTC().Format(common.SimpleTimeFormatWithTimezone),
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
	if resp.Nickname == "" {
		t.Fatalf("received %v, expected %v", "", dhj.Nickname)
	}
	if resp.ResultSummaries == nil { //nolint:staticcheck,nolintlint // SA5011 Ignore the nil warnings
		t.Fatalf("received %v, expected %v", nil, "result summaries slice")
	}
}

func TestGetManagedOrders(t *testing.T) {
	exchName := "Binance"
	engerino := &Engine{}
	em := NewExchangeManager()
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
	err = em.Add(exch)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
	var wg sync.WaitGroup
	om, err := SetupOrderManager(em, engerino.CommunicationsManager, &wg, false, false, 0)
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
		Price:     100000,
		Amount:    0.002,
		Exchange:  "Binance",
		Type:      order.Limit,
		Side:      order.Sell,
		Status:    order.New,
		AssetType: asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USDT),
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
	t.Parallel()
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
			Exchange: "wow", Asset: "meow"})
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received: %v, but expected: %v", err, asset.ErrNotSupported)
	}

	_, err = (&RPCServer{
		Engine: &Engine{},
	}).CurrencyStateWithdraw(context.Background(),
		&gctrpc.CurrencyStateWithdrawRequest{
			Exchange: "wow", Asset: "spot"})
	if !errors.Is(err, ErrSubSystemNotStarted) {
		t.Fatalf("received: %v, but expected: %v", err, ErrSubSystemNotStarted)
	}
}

func TestCurrencyStateDeposit(t *testing.T) {
	t.Parallel()
	_, err := (&RPCServer{
		Engine: &Engine{},
	}).CurrencyStateDeposit(context.Background(),
		&gctrpc.CurrencyStateDepositRequest{Exchange: "wow", Asset: "meow"})
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received: %v, but expected: %v", err, asset.ErrNotSupported)
	}

	_, err = (&RPCServer{
		Engine: &Engine{},
	}).CurrencyStateDeposit(context.Background(),
		&gctrpc.CurrencyStateDepositRequest{Exchange: "wow", Asset: "spot"})
	if !errors.Is(err, ErrSubSystemNotStarted) {
		t.Fatalf("received: %v, but expected: %v", err, ErrSubSystemNotStarted)
	}
}

func TestCurrencyStateTrading(t *testing.T) {
	t.Parallel()
	_, err := (&RPCServer{
		Engine: &Engine{},
	}).CurrencyStateTrading(context.Background(),
		&gctrpc.CurrencyStateTradingRequest{Exchange: "wow", Asset: "meow"})
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received: %v, but expected: %v", err, asset.ErrNotSupported)
	}

	_, err = (&RPCServer{
		Engine: &Engine{},
	}).CurrencyStateTrading(context.Background(),
		&gctrpc.CurrencyStateTradingRequest{Exchange: "wow", Asset: "spot"})
	if !errors.Is(err, ErrSubSystemNotStarted) {
		t.Fatalf("received: %v, but expected: %v", err, ErrSubSystemNotStarted)
	}
}

func TestCurrencyStateTradingPair(t *testing.T) {
	t.Parallel()
	em := NewExchangeManager()
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
		ConfigFormat: &currency.EMPTYFORMAT,
		Available:    currency.Pairs{cp},
		Enabled:      currency.Pairs{cp},
	}
	fakeExchange := fExchange{
		IBotExchange: exch,
	}
	err = em.Add(fakeExchange)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
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
	em := NewExchangeManager()
	exch, err := em.NewExchangeByName("binance")
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
	cp.Delimiter = ""

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
	err = em.Add(fakeExchange)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
	var wg sync.WaitGroup
	om, err := SetupOrderManager(em, &CommunicationManager{}, &wg, false, false, time.Hour)
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
	})
	if !errors.Is(err, exchange.ErrCredentialsAreEmpty) {
		t.Fatalf("received '%v', expected '%v'", err, exchange.ErrCredentialsAreEmpty)
	}

	ctx := account.DeployCredentialsToContext(context.Background(),
		&account.Credentials{
			Key:    "wow",
			Secret: "super wow",
		},
	)

	_, err = s.GetFuturesPositions(ctx, &gctrpc.GetFuturesPositionsRequest{
		Exchange: "test",
		Asset:    asset.Futures.String(),
		Pair: &gctrpc.CurrencyPair{
			Delimiter: currency.DashDelimiter,
			Base:      cp.Base.String(),
			Quote:     cp.Quote.String(),
		},
		IncludeFullOrderData:    true,
		IncludeFullFundingRates: true,
		IncludePredictedRate:    true,
		GetPositionStats:        true,
		GetFundingPayments:      true,
	})
	if !errors.Is(err, ErrExchangeNotFound) {
		t.Errorf("received '%v', expected '%v'", err, ErrExchangeNotFound)
	}

	od := &order.Detail{
		Price:     1337,
		Amount:    1337,
		Fee:       1.337,
		Exchange:  fakeExchangeName,
		OrderID:   "test",
		Side:      order.Long,
		Status:    order.Open,
		AssetType: asset.Futures,
		Date:      time.Now(),
		Pair:      cp,
	}
	err = s.OrderManager.orderStore.futuresPositionController.TrackNewOrder(od)
	if !errors.Is(err, nil) {
		t.Fatalf("received '%v', expected '%v'", err, nil)
	}
	_, err = s.GetFuturesPositions(ctx, &gctrpc.GetFuturesPositionsRequest{
		Exchange: fakeExchangeName,
		Asset:    asset.Futures.String(),
		Pair: &gctrpc.CurrencyPair{
			Delimiter: currency.DashDelimiter,
			Base:      cp.Base.String(),
			Quote:     cp.Quote.String(),
		},
		IncludeFullOrderData: true,
	})
	if !errors.Is(err, nil) {
		t.Fatalf("received '%v', expected '%v'", err, nil)
	}

	_, err = s.GetFuturesPositions(ctx, &gctrpc.GetFuturesPositionsRequest{
		Exchange: fakeExchangeName,
		Asset:    asset.Spot.String(),
		Pair: &gctrpc.CurrencyPair{
			Delimiter: currency.DashDelimiter,
			Base:      cp.Base.String(),
			Quote:     cp.Quote.String(),
		},
	})
	if !errors.Is(err, order.ErrNotFuturesAsset) {
		t.Errorf("received '%v', expected '%v'", err, order.ErrNotFuturesAsset)
	}
}

func TestGetCollateral(t *testing.T) {
	t.Parallel()
	em := NewExchangeManager()
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
		ConfigFormat: &currency.EMPTYFORMAT,
		Available:    currency.Pairs{cp},
		Enabled:      currency.Pairs{cp},
	}
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		AssetEnabled: convert.BoolPtr(true),
		ConfigFormat: &currency.EMPTYFORMAT,
		Available:    currency.Pairs{cp},
		Enabled:      currency.Pairs{cp},
	}
	fakeExchange := fExchange{
		IBotExchange: exch,
	}
	err = em.Add(fakeExchange)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
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

	ctx := account.DeployCredentialsToContext(context.Background(),
		&account.Credentials{Key: "fakerino", Secret: "supafake"})

	_, err = s.GetCollateral(ctx, &gctrpc.GetCollateralRequest{
		Exchange: fakeExchangeName,
		Asset:    asset.Futures.String(),
	})
	if !errors.Is(err, errNoAccountInformation) {
		t.Fatalf("received '%v', expected '%v'", err, errNoAccountInformation)
	}

	ctx = account.DeployCredentialsToContext(context.Background(),
		&account.Credentials{Key: "fakerino", Secret: "supafake", SubAccount: "1337"})

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

func TestShutdown(t *testing.T) {
	t.Parallel()
	s := RPCServer{Engine: &Engine{}}
	_, err := s.Shutdown(context.Background(), &gctrpc.ShutdownRequest{})
	if !errors.Is(err, errShutdownNotAllowed) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errShutdownNotAllowed)
	}

	s.Engine.Settings.EnableGRPCShutdown = true
	_, err = s.Shutdown(context.Background(), &gctrpc.ShutdownRequest{})
	if !errors.Is(err, errGRPCShutdownSignalIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errGRPCShutdownSignalIsNil)
	}

	s.Engine.GRPCShutdownSignal = make(chan struct{}, 1)
	_, err = s.Shutdown(context.Background(), &gctrpc.ShutdownRequest{})
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
}

func TestGetTechnicalAnalysis(t *testing.T) {
	t.Parallel()

	em := NewExchangeManager()
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

	b.Features.Enabled.Kline.Intervals = kline.DeployExchangeIntervals(kline.IntervalCapacity{Interval: kline.OneDay})
	err = em.Add(fExchange{IBotExchange: exch})
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
	s := RPCServer{
		Engine: &Engine{
			ExchangeManager: em,
			currencyStateManager: &CurrencyStateManager{
				started:          1,
				iExchangeManager: em,
			},
		},
	}

	_, err = s.GetTechnicalAnalysis(context.Background(), &gctrpc.GetTechnicalAnalysisRequest{})
	if !errors.Is(err, ErrExchangeNameIsEmpty) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrExchangeNameIsEmpty)
	}

	_, err = s.GetTechnicalAnalysis(context.Background(), &gctrpc.GetTechnicalAnalysisRequest{
		Exchange: fakeExchangeName,
	})
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received: '%v' but expected: '%v'", err, asset.ErrNotSupported)
	}

	_, err = s.GetTechnicalAnalysis(context.Background(), &gctrpc.GetTechnicalAnalysisRequest{
		Exchange:  fakeExchangeName,
		AssetType: "upsideprofitcontract",
		Pair:      &gctrpc.CurrencyPair{},
	})
	if !errors.Is(err, errExpectedTestError) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errExpectedTestError)
	}

	_, err = s.GetTechnicalAnalysis(context.Background(), &gctrpc.GetTechnicalAnalysisRequest{
		Exchange:  fakeExchangeName,
		AssetType: "spot",
		Pair:      &gctrpc.CurrencyPair{Base: "btc", Quote: "usd"},
		Interval:  int64(kline.OneDay),
	})
	if !errors.Is(err, errInvalidStrategy) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidStrategy)
	}

	resp, err := s.GetTechnicalAnalysis(context.Background(), &gctrpc.GetTechnicalAnalysisRequest{
		Exchange:      fakeExchangeName,
		AssetType:     "spot",
		Pair:          &gctrpc.CurrencyPair{Base: "btc", Quote: "usd"},
		Interval:      int64(kline.OneDay),
		AlgorithmType: "twap",
	})
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if resp.Signals["TWAP"].Signals[0] != 1337 {
		t.Fatalf("received: '%v' but expected: '%v'", resp.Signals["TWAP"].Signals[0], 1337)
	}

	resp, err = s.GetTechnicalAnalysis(context.Background(), &gctrpc.GetTechnicalAnalysisRequest{
		Exchange:      fakeExchangeName,
		AssetType:     "spot",
		Pair:          &gctrpc.CurrencyPair{Base: "btc", Quote: "usd"},
		Interval:      int64(kline.OneDay),
		AlgorithmType: "vwap",
	})
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if len(resp.Signals["VWAP"].Signals) != 33 {
		t.Fatalf("received: '%v' but expected: '%v'", len(resp.Signals["VWAP"].Signals), 33)
	}

	resp, err = s.GetTechnicalAnalysis(context.Background(), &gctrpc.GetTechnicalAnalysisRequest{
		Exchange:      fakeExchangeName,
		AssetType:     "spot",
		Pair:          &gctrpc.CurrencyPair{Base: "btc", Quote: "usd"},
		Interval:      int64(kline.OneDay),
		AlgorithmType: "atr",
		Period:        9,
	})
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if len(resp.Signals["ATR"].Signals) != 33 {
		t.Fatalf("received: '%v' but expected: '%v'", len(resp.Signals["ATR"].Signals), 33)
	}

	resp, err = s.GetTechnicalAnalysis(context.Background(), &gctrpc.GetTechnicalAnalysisRequest{
		Exchange:              fakeExchangeName,
		AssetType:             "spot",
		Pair:                  &gctrpc.CurrencyPair{Base: "btc", Quote: "usd"},
		Interval:              int64(kline.OneDay),
		AlgorithmType:         "bbands",
		Period:                9,
		StandardDeviationUp:   0.5,
		StandardDeviationDown: 0.5,
	})
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if len(resp.Signals["UPPER"].Signals) != 33 {
		t.Fatalf("received: '%v' but expected: '%v'", len(resp.Signals["UPPER"].Signals), 33)
	}

	if len(resp.Signals["MIDDLE"].Signals) != 33 {
		t.Fatalf("received: '%v' but expected: '%v'", len(resp.Signals["MIDDLE"].Signals), 33)
	}

	if len(resp.Signals["LOWER"].Signals) != 33 {
		t.Fatalf("received: '%v' but expected: '%v'", len(resp.Signals["LOWER"].Signals), 33)
	}

	resp, err = s.GetTechnicalAnalysis(context.Background(), &gctrpc.GetTechnicalAnalysisRequest{
		Exchange:      fakeExchangeName,
		AssetType:     "spot",
		Pair:          &gctrpc.CurrencyPair{Base: "btc", Quote: "usd"},
		OtherPair:     &gctrpc.CurrencyPair{Base: "btc", Quote: "usd"},
		Interval:      int64(kline.OneDay),
		AlgorithmType: "COCO",
		Period:        9,
	})
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if len(resp.Signals["COCO"].Signals) != 33 {
		t.Fatalf("received: '%v' but expected: '%v'", len(resp.Signals["COCO"].Signals), 33)
	}

	resp, err = s.GetTechnicalAnalysis(context.Background(), &gctrpc.GetTechnicalAnalysisRequest{
		Exchange:      fakeExchangeName,
		AssetType:     "spot",
		Pair:          &gctrpc.CurrencyPair{Base: "btc", Quote: "usd"},
		Interval:      int64(kline.OneDay),
		AlgorithmType: "sma",
		Period:        9,
	})
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if len(resp.Signals["SMA"].Signals) != 33 {
		t.Fatalf("received: '%v' but expected: '%v'", len(resp.Signals["SMA"].Signals), 33)
	}

	resp, err = s.GetTechnicalAnalysis(context.Background(), &gctrpc.GetTechnicalAnalysisRequest{
		Exchange:      fakeExchangeName,
		AssetType:     "spot",
		Pair:          &gctrpc.CurrencyPair{Base: "btc", Quote: "usd"},
		Interval:      int64(kline.OneDay),
		AlgorithmType: "ema",
		Period:        9,
	})
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if len(resp.Signals["EMA"].Signals) != 33 {
		t.Fatalf("received: '%v' but expected: '%v'", len(resp.Signals["EMA"].Signals), 33)
	}

	resp, err = s.GetTechnicalAnalysis(context.Background(), &gctrpc.GetTechnicalAnalysisRequest{
		Exchange:      fakeExchangeName,
		AssetType:     "spot",
		Pair:          &gctrpc.CurrencyPair{Base: "btc", Quote: "usd"},
		Interval:      int64(kline.OneDay),
		AlgorithmType: "macd",
		Period:        9,
		FastPeriod:    12,
		SlowPeriod:    26,
	})
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if len(resp.Signals["MACD"].Signals) != 33 {
		t.Fatalf("received: '%v' but expected: '%v'", len(resp.Signals["MACD"].Signals), 33)
	}

	if len(resp.Signals["SIGNAL"].Signals) != 33 {
		t.Fatalf("received: '%v' but expected: '%v'", len(resp.Signals["SIGNAL"].Signals), 33)
	}

	if len(resp.Signals["HISTOGRAM"].Signals) != 33 {
		t.Fatalf("received: '%v' but expected: '%v'", len(resp.Signals["HISTOGRAM"].Signals), 33)
	}

	resp, err = s.GetTechnicalAnalysis(context.Background(), &gctrpc.GetTechnicalAnalysisRequest{
		Exchange:      fakeExchangeName,
		AssetType:     "spot",
		Pair:          &gctrpc.CurrencyPair{Base: "btc", Quote: "usd"},
		Interval:      int64(kline.OneDay),
		AlgorithmType: "mfi",
		Period:        9,
	})
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if len(resp.Signals["MFI"].Signals) != 33 {
		t.Fatalf("received: '%v' but expected: '%v'", len(resp.Signals["MFI"].Signals), 33)
	}

	resp, err = s.GetTechnicalAnalysis(context.Background(), &gctrpc.GetTechnicalAnalysisRequest{
		Exchange:      fakeExchangeName,
		AssetType:     "spot",
		Pair:          &gctrpc.CurrencyPair{Base: "btc", Quote: "usd"},
		Interval:      int64(kline.OneDay),
		AlgorithmType: "obv",
		Period:        9,
	})
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if len(resp.Signals["OBV"].Signals) != 33 {
		t.Fatalf("received: '%v' but expected: '%v'", len(resp.Signals["OBV"].Signals), 33)
	}

	resp, err = s.GetTechnicalAnalysis(context.Background(), &gctrpc.GetTechnicalAnalysisRequest{
		Exchange:      fakeExchangeName,
		AssetType:     "spot",
		Pair:          &gctrpc.CurrencyPair{Base: "btc", Quote: "usd"},
		Interval:      int64(kline.OneDay),
		AlgorithmType: "rsi",
		Period:        9,
	})
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if len(resp.Signals["RSI"].Signals) != 33 {
		t.Fatalf("received: '%v' but expected: '%v'", len(resp.Signals["RSI"].Signals), 33)
	}
}

func TestGetMarginRatesHistory(t *testing.T) {
	t.Parallel()
	em := NewExchangeManager()
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
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		AssetEnabled: convert.BoolPtr(true),
		ConfigFormat: &currency.PairFormat{},
		Available:    currency.Pairs{cp},
		Enabled:      currency.Pairs{cp},
	}
	fakeExchange := fExchange{
		IBotExchange: exch,
	}
	err = em.Add(fakeExchange)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
	s := RPCServer{
		Engine: &Engine{
			ExchangeManager: em,
			currencyStateManager: &CurrencyStateManager{
				started: 1, iExchangeManager: em,
			},
		},
	}
	_, err = s.GetMarginRatesHistory(context.Background(), nil)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, common.ErrNilPointer)
	}

	request := &gctrpc.GetMarginRatesHistoryRequest{}
	_, err = s.GetMarginRatesHistory(context.Background(), request)
	if !errors.Is(err, ErrExchangeNameIsEmpty) {
		t.Errorf("received '%v' expected '%v'", err, ErrExchangeNameIsEmpty)
	}

	request.Exchange = fakeExchangeName
	_, err = s.GetMarginRatesHistory(context.Background(), request)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Errorf("received '%v' expected '%v'", err, asset.ErrNotSupported)
	}

	request.Asset = asset.Spot.String()
	_, err = s.GetMarginRatesHistory(context.Background(), request)
	if !errors.Is(err, currency.ErrCurrencyNotFound) {
		t.Errorf("received '%v' expected '%v'", err, currency.ErrCurrencyNotFound)
	}

	request.Currency = "usd"
	_, err = s.GetMarginRatesHistory(context.Background(), request)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	request.GetBorrowRates = true
	request.GetLendingPayments = true
	request.GetBorrowCosts = true
	request.GetPredictedRate = true
	request.IncludeAllRates = true
	resp, err := s.GetMarginRatesHistory(context.Background(), request)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if len(resp.Rates) == 0 {
		t.Errorf("received '%v' expected '%v'", len(resp.Rates), 1)
	}
	if resp.PredictedRate == nil {
		t.Errorf("received '%v' expected '%v'", nil, "not nil")
	}
	if resp.TakerFeeRate != "1337" {
		t.Errorf("received '%v' expected '%v'", resp.TakerFeeRate, "1337")
	}
	if resp.SumLendingPayments != "1337" {
		t.Errorf("received '%v' expected '%v'", resp.SumLendingPayments, "1337")
	}
	if resp.AvgBorrowSize != "1337" {
		t.Errorf("received '%v' expected '%v'", resp.AvgBorrowSize, "1337")
	}
	if resp.AvgLendingSize != "1337" {
		t.Errorf("received '%v' expected '%v'", resp.AvgLendingSize, "1337")
	}
	if resp.SumBorrowCosts != "1337" {
		t.Errorf("received '%v' expected '%v'", resp.SumBorrowCosts, "1337")
	}

	request.CalculateOffline = true
	_, err = s.GetMarginRatesHistory(context.Background(), request)
	if !errors.Is(err, common.ErrCannotCalculateOffline) {
		t.Errorf("received '%v' expected '%v'", err, common.ErrCannotCalculateOffline)
	}

	request.TakerFeeRate = "-1337"
	_, err = s.GetMarginRatesHistory(context.Background(), request)
	if !errors.Is(err, common.ErrCannotCalculateOffline) {
		t.Errorf("received '%v' expected '%v'", err, common.ErrCannotCalculateOffline)
	}

	request.TakerFeeRate = "1337"
	_, err = s.GetMarginRatesHistory(context.Background(), request)
	if !errors.Is(err, common.ErrCannotCalculateOffline) {
		t.Errorf("received '%v' expected '%v'", err, common.ErrCannotCalculateOffline)
	}

	request.Rates = []*gctrpc.MarginRate{
		{
			Time:       time.Now().Format(common.SimpleTimeFormatWithTimezone),
			HourlyRate: "1337",
		},
	}
	_, err = s.GetMarginRatesHistory(context.Background(), request)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	request.Rates = []*gctrpc.MarginRate{
		{
			Time:           time.Now().Format(common.SimpleTimeFormatWithTimezone),
			HourlyRate:     "1337",
			LendingPayment: &gctrpc.LendingPayment{Size: "1337"},
			BorrowCost:     &gctrpc.BorrowCost{Size: "1337"},
		},
	}
	_, err = s.GetMarginRatesHistory(context.Background(), request)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
}

func TestGetFundingRates(t *testing.T) {
	t.Parallel()

	em := NewExchangeManager()
	exch, err := em.NewExchangeByName("binance")
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
	err = em.Add(fakeExchange)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
	var wg sync.WaitGroup
	om, err := SetupOrderManager(em, &CommunicationManager{}, &wg, false, false, time.Hour)
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

	_, err = s.GetFundingRates(context.Background(), nil)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Errorf("received: '%v' but expected: '%v'", err, common.ErrNilPointer)
	}
	request := &gctrpc.GetFundingRatesRequest{
		Exchange:         "",
		Asset:            "",
		Pair:             nil,
		StartDate:        "",
		EndDate:          "",
		IncludePredicted: false,
		IncludePayments:  false,
	}
	_, err = s.GetFundingRates(context.Background(), request)
	if !errors.Is(err, ErrExchangeNameIsEmpty) {
		t.Errorf("received: '%v' but expected: '%v'", err, ErrExchangeNameIsEmpty)
	}
	request.Exchange = exch.GetName()
	_, err = s.GetFundingRates(context.Background(), request)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Errorf("received: '%v' but expected: '%v'", err, asset.ErrNotSupported)
	}

	request.Asset = asset.Spot.String()
	_, err = s.GetFundingRates(context.Background(), request)
	if !errors.Is(err, order.ErrNotFuturesAsset) {
		t.Errorf("received: '%v' but expected: '%v'", err, order.ErrNotFuturesAsset)
	}

	request.Asset = asset.Futures.String()
	request.Pair = &gctrpc.CurrencyPair{
		Delimiter: cp.Delimiter,
		Base:      cp.Base.String(),
		Quote:     cp.Quote.String(),
	}
	request.IncludePredicted = true
	request.IncludePayments = true
	_, err = s.GetFundingRates(context.Background(), request)
	if !errors.Is(err, nil) {
		t.Errorf("received: '%v' but expected: '%v'", err, nil)
	}
}

func TestGetLatestFundingRate(t *testing.T) {
	t.Parallel()

	em := NewExchangeManager()
	exch, err := em.NewExchangeByName("binance")
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
	err = em.Add(fakeExchange)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
	var wg sync.WaitGroup
	om, err := SetupOrderManager(em, &CommunicationManager{}, &wg, false, false, time.Hour)
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

	_, err = s.GetLatestFundingRate(context.Background(), nil)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Errorf("received: '%v' but expected: '%v'", err, common.ErrNilPointer)
	}
	request := &gctrpc.GetLatestFundingRateRequest{
		Exchange:         "",
		Asset:            "",
		Pair:             nil,
		IncludePredicted: false,
	}
	_, err = s.GetLatestFundingRate(context.Background(), request)
	if !errors.Is(err, ErrExchangeNameIsEmpty) {
		t.Errorf("received: '%v' but expected: '%v'", err, ErrExchangeNameIsEmpty)
	}
	request.Exchange = exch.GetName()
	_, err = s.GetLatestFundingRate(context.Background(), request)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Errorf("received: '%v' but expected: '%v'", err, asset.ErrNotSupported)
	}

	request.Asset = asset.Spot.String()
	_, err = s.GetLatestFundingRate(context.Background(), request)
	if !errors.Is(err, order.ErrNotFuturesAsset) {
		t.Errorf("received: '%v' but expected: '%v'", err, order.ErrNotFuturesAsset)
	}

	request.Asset = asset.Futures.String()
	request.Pair = &gctrpc.CurrencyPair{
		Delimiter: cp.Delimiter,
		Base:      cp.Base.String(),
		Quote:     cp.Quote.String(),
	}
	request.IncludePredicted = true
	_, err = s.GetLatestFundingRate(context.Background(), request)
	if !errors.Is(err, nil) {
		t.Errorf("received: '%v' but expected: '%v'", err, nil)
	}
}

func TestGetManagedPosition(t *testing.T) {
	t.Parallel()
	em := NewExchangeManager()
	exch, err := em.NewExchangeByName("binance")
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
	cp2, err := currency.NewPairFromString("btc-usd")
	if err != nil {
		t.Fatal(err)
	}

	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Futures] = &currency.PairStore{
		AssetEnabled:  convert.BoolPtr(true),
		RequestFormat: &currency.PairFormat{Delimiter: "-"},
		ConfigFormat:  &currency.PairFormat{Delimiter: "-"},
		Available:     currency.Pairs{cp, cp2},
		Enabled:       currency.Pairs{cp, cp2},
	}
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		AssetEnabled:  convert.BoolPtr(true),
		ConfigFormat:  &currency.PairFormat{Delimiter: "/"},
		RequestFormat: &currency.PairFormat{Delimiter: "/"},
		Available:     currency.Pairs{cp, cp2},
		Enabled:       currency.Pairs{cp, cp2},
	}
	fakeExchange := fExchange{
		IBotExchange: exch,
	}
	err = em.Add(fakeExchange)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
	var wg sync.WaitGroup
	om, err := SetupOrderManager(em, &CommunicationManager{}, &wg, false, false, time.Hour)
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
	_, err = s.GetManagedPosition(context.Background(), nil)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Errorf("received '%v', expected '%v'", err, common.ErrNilPointer)
	}

	request := &gctrpc.GetManagedPositionRequest{}
	_, err = s.GetManagedPosition(context.Background(), request)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Errorf("received '%v', expected '%v'", err, common.ErrNilPointer)
	}

	request.Pair = &gctrpc.CurrencyPair{
		Delimiter: "-",
		Base:      "BTC",
		Quote:     "USD",
	}
	_, err = s.GetManagedPosition(context.Background(), request)
	if !errors.Is(err, ErrExchangeNameIsEmpty) {
		t.Errorf("received '%v', expected '%v'", err, ErrExchangeNameIsEmpty)
	}

	request.Exchange = fakeExchangeName
	_, err = s.GetManagedPosition(context.Background(), request)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Errorf("received '%v', expected '%v'", err, asset.ErrNotSupported)
	}

	request.Asset = asset.Spot.String()
	_, err = s.GetManagedPosition(context.Background(), request)
	if !errors.Is(err, order.ErrNotFuturesAsset) {
		t.Errorf("received '%v', expected '%v'", err, order.ErrNotFuturesAsset)
	}

	request.Asset = asset.Futures.String()
	s.OrderManager, err = SetupOrderManager(em, &CommunicationManager{}, &wg, false, false, time.Hour)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected '%v'", err, nil)
	}
	s.OrderManager.started = 1
	s.OrderManager.activelyTrackFuturesPositions = true
	_, err = s.GetManagedPosition(context.Background(), request)
	if !errors.Is(err, order.ErrPositionNotFound) {
		t.Errorf("received '%v', expected '%v'", err, order.ErrPositionNotFound)
	}

	err = s.OrderManager.orderStore.futuresPositionController.TrackNewOrder(&order.Detail{
		Leverage:             1337,
		Price:                1337,
		Amount:               1337,
		LimitPriceUpper:      1337,
		LimitPriceLower:      1337,
		TriggerPrice:         1337,
		AverageExecutedPrice: 1337,
		QuoteAmount:          1337,
		ExecutedAmount:       1337,
		RemainingAmount:      1337,
		Cost:                 1337,
		Exchange:             fakeExchangeName,
		OrderID:              "1337",
		Type:                 order.Market,
		Side:                 order.Buy,
		Status:               order.Filled,
		AssetType:            asset.Futures,
		Date:                 time.Now(),
		LastUpdated:          time.Now(),
		Pair:                 cp2,
		Trades: []order.TradeHistory{
			{
				Timestamp: time.Now(),
				Side:      order.Buy,
			},
		},
	})
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected '%v'", err, nil)
	}
	_, err = s.GetManagedPosition(context.Background(), request)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected '%v'", err, nil)
	}
}

func TestGetAllManagedPositions(t *testing.T) {
	t.Parallel()

	em := NewExchangeManager()
	exch, err := em.NewExchangeByName("binance")
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
	cp2, err := currency.NewPairFromString("btc-usd")
	if err != nil {
		t.Fatal(err)
	}

	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Futures] = &currency.PairStore{
		AssetEnabled:  convert.BoolPtr(true),
		RequestFormat: &currency.PairFormat{Delimiter: "-"},
		ConfigFormat:  &currency.PairFormat{Delimiter: "-"},
		Available:     currency.Pairs{cp, cp2},
		Enabled:       currency.Pairs{cp, cp2},
	}
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		AssetEnabled:  convert.BoolPtr(true),
		ConfigFormat:  &currency.PairFormat{Delimiter: "/"},
		RequestFormat: &currency.PairFormat{Delimiter: "/"},
		Available:     currency.Pairs{cp, cp2},
		Enabled:       currency.Pairs{cp, cp2},
	}
	fakeExchange := fExchange{
		IBotExchange: exch,
	}
	err = em.Add(fakeExchange)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
	var wg sync.WaitGroup
	om, err := SetupOrderManager(em, &CommunicationManager{}, &wg, false, false, time.Hour)
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
	_, err = s.GetAllManagedPositions(context.Background(), nil)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Errorf("received '%v', expected '%v'", err, common.ErrNilPointer)
	}

	request := &gctrpc.GetAllManagedPositionsRequest{}
	s.OrderManager, err = SetupOrderManager(em, &CommunicationManager{}, &wg, false, true, time.Hour)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected '%v'", err, nil)
	}
	s.OrderManager.started = 1
	_, err = s.GetAllManagedPositions(context.Background(), request)
	if !errors.Is(err, order.ErrNoPositionsFound) {
		t.Errorf("received '%v', expected '%v'", err, order.ErrNoPositionsFound)
	}

	err = s.OrderManager.orderStore.futuresPositionController.TrackNewOrder(&order.Detail{
		Leverage:             1337,
		Price:                1337,
		Amount:               1337,
		LimitPriceUpper:      1337,
		LimitPriceLower:      1337,
		TriggerPrice:         1337,
		AverageExecutedPrice: 1337,
		QuoteAmount:          1337,
		ExecutedAmount:       1337,
		RemainingAmount:      1337,
		Cost:                 1337,
		Exchange:             fakeExchangeName,
		OrderID:              "1337",
		Type:                 order.Market,
		Side:                 order.Buy,
		Status:               order.Filled,
		AssetType:            asset.Futures,
		Date:                 time.Now(),
		LastUpdated:          time.Now(),
		Pair:                 cp2,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected '%v'", err, nil)
	}
	request.IncludePredictedRate = true
	request.GetFundingPayments = true
	request.IncludeFullFundingRates = true
	request.IncludeFullOrderData = true
	_, err = s.GetAllManagedPositions(context.Background(), request)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected '%v'", err, nil)
	}
}

func TestGetOrderbookMovement(t *testing.T) {
	t.Parallel()
	em := NewExchangeManager()
	exch, err := em.NewExchangeByName("binance")
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	b := exch.GetBase()
	b.Name = fakeExchangeName
	b.Enabled = true

	cp, err := currency.NewPairFromString("btc-metal")
	if err != nil {
		t.Fatal(err)
	}

	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
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
	err = em.Add(fakeExchange)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	s := RPCServer{Engine: &Engine{ExchangeManager: em}}

	req := &gctrpc.GetOrderbookMovementRequest{}
	_, err = s.GetOrderbookMovement(context.Background(), req)
	if !errors.Is(err, ErrExchangeNameIsEmpty) {
		t.Fatalf("received: '%+v' but expected: '%v'", err, ErrExchangeNameIsEmpty)
	}

	req.Exchange = "fake"
	_, err = s.GetOrderbookMovement(context.Background(), req)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received: '%+v' but expected: '%v'", err, asset.ErrNotSupported)
	}

	req.Asset = asset.Spot.String()
	req.Pair = &gctrpc.CurrencyPair{}
	_, err = s.GetOrderbookMovement(context.Background(), req)
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Fatalf("received: '%+v' but expected: '%v'", err, currency.ErrCurrencyPairEmpty)
	}

	req.Pair = &gctrpc.CurrencyPair{
		Base:  currency.BTC.String(),
		Quote: currency.METAL.String(),
	}
	_, err = s.GetOrderbookMovement(context.Background(), req)
	if !strings.Contains(err.Error(), "cannot find orderbook") {
		t.Fatalf("received: '%+v' but expected: '%v'", err, "cannot find orderbook")
	}

	depth, err := orderbook.DeployDepth(req.Exchange, currency.NewPair(currency.BTC, currency.METAL), asset.Spot)
	if err != nil {
		t.Fatal(err)
	}

	bid := []orderbook.Item{
		{Price: 10, Amount: 1},
		{Price: 9, Amount: 1},
		{Price: 8, Amount: 1},
		{Price: 7, Amount: 1},
	}
	ask := []orderbook.Item{
		{Price: 11, Amount: 1},
		{Price: 12, Amount: 1},
		{Price: 13, Amount: 1},
		{Price: 14, Amount: 1},
	}
	depth.LoadSnapshot(bid, ask, 0, time.Time{}, true)

	_, err = s.GetOrderbookMovement(context.Background(), req)
	if err.Error() != "quote amount invalid" {
		t.Fatalf("received: '%+v' but expected: '%v'", err, "quote amount invalid")
	}

	req.Amount = 11
	move, err := s.GetOrderbookMovement(context.Background(), req)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%+v' but expected: '%v'", err, nil)
	}

	if move.Bought != 1 {
		t.Fatalf("received: '%v' but expected: '%v'", move.Bought, 1)
	}

	req.Sell = true
	req.Amount = 1
	move, err = s.GetOrderbookMovement(context.Background(), req)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%+v' but expected: '%v'", err, nil)
	}

	if move.Bought != 10 {
		t.Fatalf("received: '%v' but expected: '%v'", move.Bought, 10)
	}
}

func TestGetOrderbookAmountByNominal(t *testing.T) {
	t.Parallel()
	em := NewExchangeManager()
	exch, err := em.NewExchangeByName("binance")
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	b := exch.GetBase()
	b.Name = fakeExchangeName
	b.Enabled = true

	cp, err := currency.NewPairFromString("btc-meme")
	if err != nil {
		t.Fatal(err)
	}

	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
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
	err = em.Add(fakeExchange)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	s := RPCServer{Engine: &Engine{ExchangeManager: em}}

	req := &gctrpc.GetOrderbookAmountByNominalRequest{}
	_, err = s.GetOrderbookAmountByNominal(context.Background(), req)
	if !errors.Is(err, ErrExchangeNameIsEmpty) {
		t.Fatalf("received: '%+v' but expected: '%v'", err, ErrExchangeNameIsEmpty)
	}

	req.Exchange = "fake"
	_, err = s.GetOrderbookAmountByNominal(context.Background(), req)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received: '%+v' but expected: '%v'", err, asset.ErrNotSupported)
	}

	req.Asset = asset.Spot.String()
	req.Pair = &gctrpc.CurrencyPair{}
	_, err = s.GetOrderbookAmountByNominal(context.Background(), req)
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Fatalf("received: '%+v' but expected: '%v'", err, currency.ErrCurrencyPairEmpty)
	}

	req.Pair = &gctrpc.CurrencyPair{
		Base:  currency.BTC.String(),
		Quote: currency.MEME.String(),
	}
	_, err = s.GetOrderbookAmountByNominal(context.Background(), req)
	if !strings.Contains(err.Error(), "cannot find orderbook") {
		t.Fatalf("received: '%+v' but expected: '%v'", err, "cannot find orderbook")
	}

	depth, err := orderbook.DeployDepth(req.Exchange, currency.NewPair(currency.BTC, currency.MEME), asset.Spot)
	if err != nil {
		t.Fatal(err)
	}

	bid := []orderbook.Item{
		{Price: 10, Amount: 1},
		{Price: 9, Amount: 1},
		{Price: 8, Amount: 1},
		{Price: 7, Amount: 1},
	}
	ask := []orderbook.Item{
		{Price: 11, Amount: 1},
		{Price: 12, Amount: 1},
		{Price: 13, Amount: 1},
		{Price: 14, Amount: 1},
	}
	depth.LoadSnapshot(bid, ask, 0, time.Time{}, true)

	nominal, err := s.GetOrderbookAmountByNominal(context.Background(), req)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%+v' but expected: '%v'", err, nil)
	}

	if nominal.AmountRequired != 11 {
		t.Fatalf("received: '%v' but expected: '%v'", nominal.AmountRequired, 11)
	}

	req.Sell = true
	nominal, err = s.GetOrderbookAmountByNominal(context.Background(), req)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%+v' but expected: '%v'", err, nil)
	}

	if nominal.AmountRequired != 1 {
		t.Fatalf("received: '%v' but expected: '%v'", nominal.AmountRequired, 1)
	}
}

func TestGetOrderbookAmountByImpact(t *testing.T) {
	t.Parallel()
	em := NewExchangeManager()
	exch, err := em.NewExchangeByName("binance")
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	b := exch.GetBase()
	b.Name = fakeExchangeName
	b.Enabled = true

	cp, err := currency.NewPairFromString("btc-mad")
	if err != nil {
		t.Fatal(err)
	}

	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
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
	err = em.Add(fakeExchange)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	s := RPCServer{Engine: &Engine{ExchangeManager: em}}

	req := &gctrpc.GetOrderbookAmountByImpactRequest{}
	_, err = s.GetOrderbookAmountByImpact(context.Background(), req)
	if !errors.Is(err, ErrExchangeNameIsEmpty) {
		t.Fatalf("received: '%+v' but expected: '%v'", err, ErrExchangeNameIsEmpty)
	}

	req.Exchange = "fake"
	_, err = s.GetOrderbookAmountByImpact(context.Background(), req)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received: '%+v' but expected: '%v'", err, asset.ErrNotSupported)
	}

	req.Asset = asset.Spot.String()
	req.Pair = &gctrpc.CurrencyPair{}
	_, err = s.GetOrderbookAmountByImpact(context.Background(), req)
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Fatalf("received: '%+v' but expected: '%v'", err, currency.ErrCurrencyPairEmpty)
	}

	req.Pair = &gctrpc.CurrencyPair{
		Base:  currency.BTC.String(),
		Quote: currency.MAD.String(),
	}
	_, err = s.GetOrderbookAmountByImpact(context.Background(), req)
	if !strings.Contains(err.Error(), "cannot find orderbook") {
		t.Fatalf("received: '%+v' but expected: '%v'", err, "cannot find orderbook")
	}

	depth, err := orderbook.DeployDepth(req.Exchange, currency.NewPair(currency.BTC, currency.MAD), asset.Spot)
	if err != nil {
		t.Fatal(err)
	}

	bid := []orderbook.Item{
		{Price: 10, Amount: 1},
		{Price: 9, Amount: 1},
		{Price: 8, Amount: 1},
		{Price: 7, Amount: 1},
	}
	ask := []orderbook.Item{
		{Price: 11, Amount: 1},
		{Price: 12, Amount: 1},
		{Price: 13, Amount: 1},
		{Price: 14, Amount: 1},
	}
	depth.LoadSnapshot(bid, ask, 0, time.Time{}, true)

	req.ImpactPercentage = 9.090909090909092
	impact, err := s.GetOrderbookAmountByImpact(context.Background(), req)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%+v' but expected: '%v'", err, nil)
	}

	if impact.AmountRequired != 11 {
		t.Fatalf("received: '%v' but expected: '%v'", impact.AmountRequired, 11)
	}

	req.Sell = true
	req.ImpactPercentage = 10
	impact, err = s.GetOrderbookAmountByImpact(context.Background(), req)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%+v' but expected: '%v'", err, nil)
	}

	if impact.AmountRequired != 1 {
		t.Fatalf("received: '%v' but expected: '%v'", impact.AmountRequired, 1)
	}
}
