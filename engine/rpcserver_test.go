package engine

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/drivers"
	"github.com/thrasher-corp/gocryptotrader/database/repository"
	dbexchange "github.com/thrasher-corp/gocryptotrader/database/repository/exchange"
	sqltrade "github.com/thrasher-corp/gocryptotrader/database/repository/trade"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/binance"
	"github.com/thrasher-corp/gocryptotrader/exchanges/collateral"
	"github.com/thrasher-corp/gocryptotrader/exchanges/currencystate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
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

func (f fExchange) GetFuturesPositionSummary(context.Context, *futures.PositionSummaryRequest) (*futures.PositionSummary, error) {
	leet := decimal.NewFromInt(1337)
	return &futures.PositionSummary{
		MaintenanceMarginRequirement: leet,
		InitialMarginRequirement:     leet,
		EstimatedLiquidationPrice:    leet,
		CollateralUsed:               leet,
		MarkPrice:                    leet,
		CurrentSize:                  leet,
		AverageOpenPrice:             leet,
		UnrealisedPNL:                leet,
		MaintenanceMarginFraction:    leet,
		FreeCollateral:               leet,
		TotalCollateral:              leet,
	}, nil
}

func (f fExchange) ChangePositionMargin(_ context.Context, req *margin.PositionChangeRequest) (*margin.PositionChangeResponse, error) {
	return &margin.PositionChangeResponse{
		Exchange:        f.GetName(),
		Pair:            req.Pair,
		Asset:           req.Asset,
		AllocatedMargin: req.NewAllocatedMargin,
		MarginType:      req.MarginType,
	}, nil
}

func (f fExchange) SetLeverage(_ context.Context, _ asset.Item, _ currency.Pair, _ margin.Type, _ float64, _ order.Side) error {
	return nil
}

func (f fExchange) GetLeverage(_ context.Context, _ asset.Item, _ currency.Pair, _ margin.Type, _ order.Side) (float64, error) {
	return 1337, nil
}

func (f fExchange) SetMarginType(_ context.Context, _ asset.Item, _ currency.Pair, _ margin.Type) error {
	return nil
}

func (f fExchange) SetCollateralMode(_ context.Context, _ asset.Item, _ collateral.Mode) error {
	return nil
}

func (f fExchange) GetOpenInterest(_ context.Context, k ...key.PairAsset) ([]futures.OpenInterest, error) {
	if len(k) > 0 {
		return []futures.OpenInterest{
			{
				Key:          key.NewExchangeAssetPair(f.GetName(), k[0].Asset, k[0].Pair()),
				OpenInterest: 1337,
			},
		}, nil
	}
	return nil, nil
}

func (f fExchange) GetCollateralMode(_ context.Context, _ asset.Item) (collateral.Mode, error) {
	return collateral.SingleMode, nil
}

func (f fExchange) GetFuturesPositionOrders(_ context.Context, req *futures.PositionsRequest) ([]futures.PositionResponse, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	resp := make([]futures.PositionResponse, len(req.Pairs))
	tt := time.Now()
	for i := range req.Pairs {
		resp[i] = futures.PositionResponse{
			Asset: req.Asset,
			Pair:  req.Pairs[i],
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

func (f fExchange) GetLatestFundingRates(_ context.Context, request *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	leet := decimal.NewFromInt(1337)
	return []fundingrate.LatestRateResponse{
		{
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
		},
	}, nil
}

func (f fExchange) GetHistoricalFundingRates(_ context.Context, request *fundingrate.HistoricalRatesRequest) (*fundingrate.HistoricalRates, error) {
	leet := decimal.NewFromInt(1337)
	return &fundingrate.HistoricalRates{
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
	for x := range amount {
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

func (f fExchange) GetCurrencyTradeURL(_ context.Context, _ asset.Item, _ currency.Pair) (string, error) {
	return "https://google.com", nil
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

func (f fExchange) GetCachedTicker(p currency.Pair, a asset.Item) (*ticker.Price, error) {
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

// GetCachedSubAccounts overrides testExchange's fetch account info function to do the bare minimum required with no API calls or credentials required
// Only returns balances for creds with a SubAccount populated
func (f fExchange) GetCachedSubAccounts(ctx context.Context, a asset.Item) (accounts.SubAccounts, error) {
	creds, err := f.GetCredentials(ctx)
	if err != nil {
		return nil, err
	}
	if creds.SubAccount == "" {
		return nil, fmt.Errorf("%w for %s credentials %s asset %s", accounts.ErrNoBalances, f.GetName(), creds, a)
	}
	return accounts.SubAccounts{{
		ID: creds.SubAccount,
		Balances: accounts.CurrencyBalances{
			currency.USD: {Currency: currency.USD, Total: 1337},
			currency.BTC: {Currency: currency.BTC, Total: 13337},
		},
	}}, nil
}

// GetCachedCurrencyBalances overrides testExchange's fetch account info function to do the bare minimum required with no API calls or credentials required
// Only returns balances for creds with a SubAccount populated
func (f fExchange) GetCachedCurrencyBalances(ctx context.Context, a asset.Item) (accounts.CurrencyBalances, error) {
	subAccts, err := f.GetCachedSubAccounts(ctx, a)
	if err != nil {
		return nil, err
	}
	return subAccts[0].Balances, nil
}

// CalculateTotalCollateral overrides testExchange's CalculateTotalCollateral function
func (f fExchange) CalculateTotalCollateral(context.Context, *futures.TotalCollateralCalculator) (*futures.TotalCollateralResponse, error) {
	return &futures.TotalCollateralResponse{
		CollateralCurrency:             currency.USD,
		AvailableMaintenanceCollateral: decimal.NewFromInt(1338),
		AvailableCollateral:            decimal.NewFromInt(1337),
		UsedBreakdown: &collateral.UsedBreakdown{
			LockedInStakes:                  decimal.NewFromInt(3),
			LockedInNFTBids:                 decimal.NewFromInt(3),
			LockedInFeeVoucher:              decimal.NewFromInt(3),
			LockedInSpotMarginFundingOffers: decimal.NewFromInt(3),
			LockedInSpotOrders:              decimal.NewFromInt(3),
			LockedAsCollateral:              decimal.NewFromInt(3),
		},
		BreakdownByCurrency: []collateral.ByCurrency{
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
				ScaledUsedBreakdown: &collateral.UsedBreakdown{
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

// UpdateAccountBalances overrides testExchange's update account info function
// to do the bare minimum required with no API calls or credentials required
func (f fExchange) UpdateAccountBalances(_ context.Context, a asset.Item) (accounts.SubAccounts, error) {
	if a == asset.Futures {
		return accounts.SubAccounts{}, asset.ErrNotSupported
	}
	return accounts.SubAccounts{accounts.NewSubAccount(a, "1337")}, nil
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

// RPCTestSetup sets up everything required to run any function inside rpcserver
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
	cp := currency.NewBTCUSD()
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		Available:     currency.Pairs{cp},
		Enabled:       currency.Pairs{cp},
		AssetEnabled:  true,
		ConfigFormat:  &currency.PairFormat{Uppercase: true},
		RequestFormat: &currency.PairFormat{Uppercase: true},
	}
	err = em.Add(exch)
	require.NoError(t, err)

	exch, err = em.NewExchangeByName("Binance")
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	b = exch.GetBase()
	cp = currency.NewBTCUSDT()
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		Available:     currency.Pairs{cp},
		Enabled:       currency.Pairs{cp},
		AssetEnabled:  true,
		ConfigFormat:  &currency.PairFormat{Uppercase: true},
		RequestFormat: &currency.PairFormat{Uppercase: true},
	}
	err = em.Add(exch)
	require.NoError(t, err)

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
	_, err := s.GetSavedTrades(t.Context(), &gctrpc.GetSavedTradesRequest{})
	assert.ErrorIs(t, err, errInvalidArguments)

	_, err = s.GetSavedTrades(t.Context(), &gctrpc.GetSavedTradesRequest{
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
	assert.ErrorIs(t, err, ErrExchangeNotFound)

	_, err = s.GetSavedTrades(t.Context(), &gctrpc.GetSavedTradesRequest{
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
	_, err = s.GetSavedTrades(t.Context(), &gctrpc.GetSavedTradesRequest{
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
	_, err := s.ConvertTradesToCandles(t.Context(), &gctrpc.ConvertTradesToCandlesRequest{})
	assert.ErrorIs(t, err, errInvalidArguments)

	// bad exchange test
	_, err = s.ConvertTradesToCandles(t.Context(), &gctrpc.ConvertTradesToCandlesRequest{
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
	assert.ErrorIs(t, err, ErrExchangeNotFound)

	// no trades test
	_, err = s.ConvertTradesToCandles(t.Context(), &gctrpc.ConvertTradesToCandlesRequest{
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
	assert.ErrorIs(t, err, errNoTrades)

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
	candles, err = s.ConvertTradesToCandles(t.Context(), &gctrpc.ConvertTradesToCandlesRequest{
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
	_, err = s.ConvertTradesToCandles(t.Context(), &gctrpc.ConvertTradesToCandlesRequest{
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
	_, err = s.ConvertTradesToCandles(t.Context(), &gctrpc.ConvertTradesToCandlesRequest{
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
	candles, err = s.GetHistoricCandles(t.Context(), &gctrpc.GetHistoricCandlesRequest{
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
	cp := currency.NewBTCUSD()
	_, err := s.GetHistoricCandles(t.Context(), &gctrpc.GetHistoricCandlesRequest{
		Exchange: "",
		Pair: &gctrpc.CurrencyPair{
			Base:  cp.Base.String(),
			Quote: cp.Quote.String(),
		},
		Start:     defaultStart.Format(common.SimpleTimeFormatWithTimezone),
		End:       defaultEnd.Format(common.SimpleTimeFormatWithTimezone),
		AssetType: asset.Spot.String(),
	})
	assert.ErrorIs(t, err, common.ErrExchangeNameNotSet)

	_, err = s.GetHistoricCandles(t.Context(), &gctrpc.GetHistoricCandlesRequest{
		Exchange: "bruh",
		Pair: &gctrpc.CurrencyPair{
			Base:  cp.Base.String(),
			Quote: cp.Quote.String(),
		},
		Start:     defaultStart.Format(common.SimpleTimeFormatWithTimezone),
		End:       defaultEnd.Format(common.SimpleTimeFormatWithTimezone),
		AssetType: asset.Spot.String(),
	})
	assert.ErrorIs(t, err, ErrExchangeNotFound)

	_, err = s.GetHistoricCandles(t.Context(), &gctrpc.GetHistoricCandlesRequest{
		Exchange:  testExchange,
		Start:     defaultStart.Format(common.SimpleTimeFormatWithTimezone),
		End:       defaultEnd.Format(common.SimpleTimeFormatWithTimezone),
		Pair:      nil,
		AssetType: asset.Spot.String(),
	})
	assert.ErrorIs(t, err, errCurrencyPairUnset)

	_, err = s.GetHistoricCandles(t.Context(), &gctrpc.GetHistoricCandlesRequest{
		Exchange: testExchange,
		Pair: &gctrpc.CurrencyPair{
			Base:  currency.BTC.String(),
			Quote: currency.USD.String(),
		},
		Start: "2020-01-02 15:04:05 UTC",
		End:   "2020-01-02 15:04:05 UTC",
	})
	assert.ErrorIs(t, err, common.ErrStartEqualsEnd)

	var results *gctrpc.GetHistoricCandlesResponse
	// default run
	results, err = s.GetHistoricCandles(t.Context(), &gctrpc.GetHistoricCandlesRequest{
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
	require.NoError(t, err)
	if len(results.Candle) == 0 {
		t.Error("expected results")
	}

	// sync run
	results, err = s.GetHistoricCandles(t.Context(), &gctrpc.GetHistoricCandlesRequest{
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
	require.NoError(t, err)
	if len(results.Candle) == 0 {
		t.Error("expected results")
	}

	// db run
	results, err = s.GetHistoricCandles(t.Context(), &gctrpc.GetHistoricCandlesRequest{
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
	require.NoError(t, err)
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
	results, err = s.GetHistoricCandles(t.Context(), &gctrpc.GetHistoricCandlesRequest{
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
	require.NoError(t, err)
	if results.Candle[len(results.Candle)-1].Close != 1337 {
		t.Error("expected fancy new candle based off fancy new trade data")
	}
}

func TestFindMissingSavedTradeIntervals(t *testing.T) {
	engerino := RPCTestSetup(t)
	defer CleanRPCTest(t, engerino)
	s := RPCServer{Engine: engerino}
	// bad request checks
	_, err := s.FindMissingSavedTradeIntervals(t.Context(), &gctrpc.FindMissingTradePeriodsRequest{})
	if err == nil {
		t.Error("expected error")
		return
	}
	require.ErrorIs(t, err, errInvalidArguments)
	cp := currency.NewBTCUSD()
	// no data found response
	defaultStart := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).UTC()
	defaultEnd := time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC).UTC()
	var resp *gctrpc.FindMissingIntervalsResponse
	resp, err = s.FindMissingSavedTradeIntervals(t.Context(), &gctrpc.FindMissingTradePeriodsRequest{
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

	resp, err = s.FindMissingSavedTradeIntervals(t.Context(), &gctrpc.FindMissingTradePeriodsRequest{
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

	resp, err = s.FindMissingSavedTradeIntervals(t.Context(), &gctrpc.FindMissingTradePeriodsRequest{
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
	_, err := s.FindMissingSavedCandleIntervals(t.Context(), &gctrpc.FindMissingCandlePeriodsRequest{})
	if err == nil {
		t.Error("expected error")
		return
	}
	require.ErrorIs(t, err, errInvalidArguments)
	cp := currency.NewBTCUSD()
	// no data found response
	defaultStart := time.Date(2020, 0, 0, 0, 0, 0, 0, time.UTC)
	defaultEnd := time.Date(2020, 0, 0, 4, 0, 0, 0, time.UTC)
	var resp *gctrpc.FindMissingIntervalsResponse
	_, err = s.FindMissingSavedCandleIntervals(t.Context(), &gctrpc.FindMissingCandlePeriodsRequest{
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

	_, err = s.FindMissingSavedCandleIntervals(t.Context(), &gctrpc.FindMissingCandlePeriodsRequest{
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

	resp, err = s.FindMissingSavedCandleIntervals(t.Context(), &gctrpc.FindMissingCandlePeriodsRequest{
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
	require.NoError(t, err)

	s := RPCServer{Engine: &Engine{ExchangeManager: em}}
	_, err = s.SetExchangeTradeProcessing(t.Context(), &gctrpc.SetExchangeTradeProcessingRequest{Exchange: testExchange, Status: true})
	if err != nil {
		t.Error(err)
		return
	}
	if !b.IsSaveTradeDataEnabled() {
		t.Error("expected true")
	}
	_, err = s.SetExchangeTradeProcessing(t.Context(), &gctrpc.SetExchangeTradeProcessingRequest{Exchange: testExchange, Status: false})
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
	_, err := s.GetRecentTrades(t.Context(), &gctrpc.GetSavedTradesRequest{})
	assert.ErrorIs(t, err, errInvalidArguments)

	_, err = s.GetRecentTrades(t.Context(), &gctrpc.GetSavedTradesRequest{
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
	assert.ErrorIs(t, err, ErrExchangeNotFound)

	_, err = s.GetRecentTrades(t.Context(), &gctrpc.GetSavedTradesRequest{
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
func (d *dummyServer) SendMsg(_ any) error                    { return nil }
func (d *dummyServer) RecvMsg(_ any) error                    { return nil }

func TestGetHistoricTrades(t *testing.T) {
	engerino := RPCTestSetup(t)
	defer CleanRPCTest(t, engerino)
	s := RPCServer{Engine: engerino}
	err := s.GetHistoricTrades(&gctrpc.GetSavedTradesRequest{}, nil)
	assert.ErrorIs(t, err, errInvalidArguments)

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
	assert.ErrorIs(t, err, ErrExchangeNotFound)

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

func TestGetAccountBalances(t *testing.T) {
	t.Parallel()
	em := NewExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	require.NoError(t, err)
	b := exch.GetBase()
	b.Name = fakeExchangeName
	b.Enabled = true
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{AssetEnabled: true}
	fakeExchange := fExchange{IBotExchange: exch}
	err = em.Add(fakeExchange)
	require.NoError(t, err)
	ctx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "fakerino", Secret: "supafake", SubAccount: "42"})
	s := RPCServer{Engine: &Engine{ExchangeManager: em}}
	_, err = s.GetAccountBalances(ctx, &gctrpc.GetAccountBalancesRequest{Exchange: fakeExchangeName, AssetType: asset.Spot.String()})
	assert.NoError(t, err)
}

func TestUpdateAccountBalances(t *testing.T) {
	t.Parallel()
	em := NewExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	require.NoError(t, err)
	b := exch.GetBase()
	b.Name = fakeExchangeName
	b.Enabled = true
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{AssetEnabled: true}
	fakeExchange := fExchange{IBotExchange: exch}
	err = em.Add(fakeExchange)
	require.NoError(t, err)

	s := RPCServer{Engine: &Engine{ExchangeManager: em}}
	ctx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "fakerino", Secret: "supafake", SubAccount: "42"})
	_, err = s.GetAccountBalances(ctx, &gctrpc.GetAccountBalancesRequest{Exchange: fakeExchangeName, AssetType: asset.Spot.String()})
	assert.NoError(t, err)

	_, err = s.UpdateAccountBalances(ctx, &gctrpc.GetAccountBalancesRequest{Exchange: fakeExchangeName, AssetType: asset.Futures.String()})
	assert.ErrorIs(t, err, currency.ErrAssetNotFound)

	_, err = s.UpdateAccountBalances(ctx, &gctrpc.GetAccountBalancesRequest{Exchange: fakeExchangeName, AssetType: asset.Spot.String()})
	assert.NoError(t, err)
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
	cp := currency.NewBTCUSDT()
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		Available:     currency.Pairs{cp},
		Enabled:       currency.Pairs{cp},
		AssetEnabled:  true,
		ConfigFormat:  &currency.PairFormat{Uppercase: true},
		RequestFormat: &currency.PairFormat{Uppercase: true},
	}
	err = em.Add(exch)
	require.NoError(t, err)

	var wg sync.WaitGroup
	om, err := SetupOrderManager(em, engerino.CommunicationsManager, &wg, &config.OrderManager{})
	assert.NoError(t, err)

	om.started = 1
	s := RPCServer{Engine: &Engine{ExchangeManager: em, OrderManager: om}}

	p := &gctrpc.CurrencyPair{
		Delimiter: "-",
		Base:      currency.BTC.String(),
		Quote:     currency.USDT.String(),
	}

	_, err = s.GetOrders(t.Context(), nil)
	assert.ErrorIs(t, err, errInvalidArguments)

	_, err = s.GetOrders(t.Context(), &gctrpc.GetOrdersRequest{
		AssetType: asset.Spot.String(),
		Pair:      p,
	})
	assert.ErrorIs(t, err, common.ErrExchangeNameNotSet)

	_, err = s.GetOrders(t.Context(), &gctrpc.GetOrdersRequest{
		Exchange:  "bruh",
		AssetType: asset.Spot.String(),
		Pair:      p,
	})
	assert.ErrorIs(t, err, ErrExchangeNotFound)

	_, err = s.GetOrders(t.Context(), &gctrpc.GetOrdersRequest{
		Exchange:  exchName,
		AssetType: asset.Spot.String(),
	})
	assert.ErrorIs(t, err, errCurrencyPairUnset)

	_, err = s.GetOrders(t.Context(), &gctrpc.GetOrdersRequest{
		Exchange: exchName,
		Pair:     p,
	})
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = s.GetOrders(t.Context(), &gctrpc.GetOrdersRequest{
		Exchange:  exchName,
		AssetType: asset.Spot.String(),
		Pair:      p,
		StartDate: time.Now().UTC().Add(time.Second).Format(common.SimpleTimeFormatWithTimezone),
		EndDate:   time.Now().UTC().Add(-time.Hour).Format(common.SimpleTimeFormatWithTimezone),
	})
	assert.ErrorIs(t, err, common.ErrStartAfterEnd)

	_, err = s.GetOrders(t.Context(), &gctrpc.GetOrdersRequest{
		Exchange:  exchName,
		AssetType: asset.Spot.String(),
		Pair:      p,
		StartDate: time.Now().UTC().Add(-time.Hour).Format(common.SimpleTimeFormatWithTimezone),
		EndDate:   time.Now().UTC().Add(time.Hour).Format(common.SimpleTimeFormatWithTimezone),
	})
	assert.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty)

	b.SetCredentials("test", "test", "", "", "", "")
	b.API.AuthenticatedSupport = true

	_, err = s.GetOrders(t.Context(), &gctrpc.GetOrdersRequest{
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
	cp := currency.NewBTCUSDT()
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		Available:     currency.Pairs{cp},
		Enabled:       currency.Pairs{cp},
		AssetEnabled:  true,
		ConfigFormat:  &currency.PairFormat{Uppercase: true},
		RequestFormat: &currency.PairFormat{Uppercase: true},
	}
	err = em.Add(exch)
	require.NoError(t, err)

	var wg sync.WaitGroup
	om, err := SetupOrderManager(em, engerino.CommunicationsManager, &wg, &config.OrderManager{})
	assert.NoError(t, err)

	om.started = 1
	assert.NoError(t, err)

	s := RPCServer{Engine: &Engine{ExchangeManager: em, OrderManager: om}}
	p := &gctrpc.CurrencyPair{
		Delimiter: "-",
		Base:      "BTC",
		Quote:     "USDT",
	}

	_, err = s.GetOrder(t.Context(), nil)
	assert.ErrorIs(t, err, errInvalidArguments)

	_, err = s.GetOrder(t.Context(), &gctrpc.GetOrderRequest{
		Exchange: "test123",
		OrderId:  "",
		Pair:     p,
		Asset:    "spot",
	})
	assert.ErrorIs(t, err, ErrExchangeNotFound)

	_, err = s.GetOrder(t.Context(), &gctrpc.GetOrderRequest{
		Exchange: exchName,
		OrderId:  "",
		Pair:     nil,
		Asset:    "",
	})
	assert.ErrorIs(t, err, errCurrencyPairUnset)

	_, err = s.GetOrder(t.Context(), &gctrpc.GetOrderRequest{
		Exchange: exchName,
		OrderId:  "",
		Pair:     p,
		Asset:    "",
	})
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = s.GetOrder(t.Context(), &gctrpc.GetOrderRequest{
		Exchange: exchName,
		OrderId:  "",
		Pair:     p,
		Asset:    asset.Spot.String(),
	})
	assert.ErrorIs(t, err, ErrOrderIDCannotBeEmpty)

	_, err = s.GetOrder(t.Context(), &gctrpc.GetOrderRequest{
		Exchange: exchName,
		OrderId:  "1234",
		Pair:     p,
		Asset:    asset.Spot.String(),
	})
	assert.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty)
}

func TestCheckVars(t *testing.T) {
	t.Parallel()
	var e exchange.IBotExchange
	err := checkParams("Binance", e, asset.Spot, currency.NewBTCUSDT())
	assert.ErrorIs(t, err, errExchangeNotLoaded, "checkParams should error correctly")

	e = &binance.Exchange{}
	err = checkParams("Binance", e, asset.Spot, currency.NewBTCUSDT())
	assert.ErrorIs(t, err, errExchangeNotEnabled, "checkParams should error correctly")

	e.SetEnabled(true)

	err = checkParams("Binance", e, asset.Spot, currency.NewBTCUSDT())
	assert.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "checkParams should error correctly")

	b := e.GetBase()

	for _, a := range []asset.Item{asset.Spot, asset.Margin, asset.CoinMarginedFutures, asset.USDTMarginedFutures} {
		ps := currency.PairStore{
			AssetEnabled:  true,
			RequestFormat: &currency.PairFormat{Uppercase: true},
			ConfigFormat:  &currency.PairFormat{Delimiter: currency.DashDelimiter, Uppercase: true},
		}
		switch a {
		case asset.CoinMarginedFutures:
			ps.RequestFormat = &currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter}
			ps.ConfigFormat = &currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter}
		case asset.USDTMarginedFutures:
			ps.ConfigFormat = &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}
		}
		require.NoError(t, b.SetAssetPairStore(a, ps), "SetAssetPairStore must not error")
	}

	err = checkParams("Binance", e, asset.Spot, currency.NewBTCUSDT())
	assert.ErrorIs(t, err, errCurrencyPairInvalid, "checkParams should error correctly")

	data := []currency.Pair{
		{Delimiter: currency.DashDelimiter, Base: currency.BTC, Quote: currency.USDT},
	}

	err = b.CurrencyPairs.StorePairs(asset.Spot, data, false)
	require.NoError(t, err, "StorePairs must not error")

	err = checkParams("Binance", e, asset.Spot, currency.NewBTCUSDT())
	require.ErrorIs(t, err, errCurrencyNotEnabled, "checkParams must error correctly")

	err = b.CurrencyPairs.EnablePair(asset.Spot, currency.Pair{Delimiter: currency.DashDelimiter, Base: currency.BTC, Quote: currency.USDT})
	require.NoError(t, err, "EnablePair must not error")

	err = checkParams("Binance", e, asset.Spot, currency.NewBTCUSDT())
	require.NoError(t, err, "checkParams must not error")
}

func TestParseEvents(t *testing.T) {
	t.Parallel()
	exchangeName := "Binance"
	testData := make([]*withdraw.Response, 5)
	for x := range 5 {
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
		testData[x] = resp
	}
	v := parseMultipleEvents(testData)
	require.NotNil(t, v, "parseMultipleEvents must not return nil")
	require.Len(t, v.Event, 5, "parseMultipleEvents must return 5 events")

	v = parseSingleEvents(testData[0])
	require.NotNil(t, v, "parseSingleEvents must not return nil")
	require.NotEmpty(t, v.Event, "parseSingleEvents must return an event")
	assert.Equal(t, int64(1), v.Event[0].Request.Type, "parseSingleEvents should return an event with the correct request type")

	v = parseSingleEvents(testData[1])
	require.NotNil(t, v, "parseSingleEvents must not return nil")
	require.NotEmpty(t, v.Event, "parseSingleEvents must return an event")
	assert.Zero(t, v.Event[0].Request.Type, "parseSingleEvents should return an event with the correct request type")
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
	cp := currency.NewBTCUSD()
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		Available:    currency.Pairs{cp},
		Enabled:      currency.Pairs{cp},
		AssetEnabled: true,
	}
	err = em.Add(exch)
	require.NoError(t, err)

	s := RPCServer{Engine: &Engine{dataHistoryManager: m, ExchangeManager: em}}
	_, err = s.UpsertDataHistoryJob(t.Context(), nil)
	assert.ErrorIs(t, err, errNilRequestData)

	_, err = s.UpsertDataHistoryJob(t.Context(), &gctrpc.UpsertDataHistoryJobRequest{})
	assert.ErrorIs(t, err, asset.ErrNotSupported)

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

	_, err = s.UpsertDataHistoryJob(t.Context(), job)
	assert.NoError(t, err)
}

func TestGetDataHistoryJobDetails(t *testing.T) {
	t.Parallel()
	m, _ := createDHM(t)
	s := RPCServer{Engine: &Engine{dataHistoryManager: m}}

	dhj := &DataHistoryJob{
		Nickname:  "TestGetDataHistoryJobDetails",
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewBTCUSD(),
		StartDate: time.Now().UTC().Add(-time.Minute * 2),
		EndDate:   time.Now().UTC(),
		Interval:  kline.OneMin,
	}
	err := m.UpsertJob(dhj, false)
	assert.NoError(t, err)

	_, err = s.GetDataHistoryJobDetails(t.Context(), nil)
	assert.ErrorIs(t, err, errNilRequestData)

	_, err = s.GetDataHistoryJobDetails(t.Context(), &gctrpc.GetDataHistoryJobDetailsRequest{})
	assert.ErrorIs(t, err, errNicknameIDUnset)

	_, err = s.GetDataHistoryJobDetails(t.Context(), &gctrpc.GetDataHistoryJobDetailsRequest{Id: "123", Nickname: "123"})
	assert.ErrorIs(t, err, errOnlyNicknameOrID)

	_, err = s.GetDataHistoryJobDetails(t.Context(), &gctrpc.GetDataHistoryJobDetailsRequest{Nickname: "TestGetDataHistoryJobDetails"})
	assert.NoError(t, err)

	_, err = s.GetDataHistoryJobDetails(t.Context(), &gctrpc.GetDataHistoryJobDetailsRequest{Id: dhj.ID.String()})
	assert.NoError(t, err)

	resp, err := s.GetDataHistoryJobDetails(t.Context(), &gctrpc.GetDataHistoryJobDetailsRequest{Nickname: "TestGetDataHistoryJobDetails", FullDetails: true})
	require.NoError(t, err)

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
		Pair:      currency.NewBTCUSD(),
		StartDate: time.Now().UTC().Add(-time.Minute * 2),
		EndDate:   time.Now().UTC(),
		Interval:  kline.OneMin,
	}
	err := m.UpsertJob(dhj, false)
	require.NoError(t, err)

	_, err = s.SetDataHistoryJobStatus(t.Context(), nil)
	assert.ErrorIs(t, err, errNilRequestData)

	_, err = s.SetDataHistoryJobStatus(t.Context(), &gctrpc.SetDataHistoryJobStatusRequest{})
	assert.ErrorIs(t, err, errNicknameIDUnset)

	_, err = s.SetDataHistoryJobStatus(t.Context(), &gctrpc.SetDataHistoryJobStatusRequest{Id: "123", Nickname: "123"})
	assert.ErrorIs(t, err, errOnlyNicknameOrID)

	id := dhj.ID
	_, err = s.SetDataHistoryJobStatus(t.Context(), &gctrpc.SetDataHistoryJobStatusRequest{Nickname: "TestDeleteDataHistoryJob", Status: int64(dataHistoryStatusRemoved)})
	assert.NoError(t, err)

	dhj.ID = id
	j.Status = int64(dataHistoryStatusActive)
	_, err = s.SetDataHistoryJobStatus(t.Context(), &gctrpc.SetDataHistoryJobStatusRequest{Id: id.String(), Status: int64(dataHistoryStatusRemoved)})
	assert.NoError(t, err)

	_, err = s.SetDataHistoryJobStatus(t.Context(), &gctrpc.SetDataHistoryJobStatusRequest{Id: id.String(), Status: int64(dataHistoryStatusActive)})
	assert.ErrorIs(t, err, errBadStatus)

	j.Status = int64(dataHistoryStatusActive)
	_, err = s.SetDataHistoryJobStatus(t.Context(), &gctrpc.SetDataHistoryJobStatusRequest{Id: id.String(), Status: int64(dataHistoryStatusPaused)})
	assert.NoError(t, err)

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
		Pair:      currency.NewBTCUSD(),
		StartDate: time.Now().UTC().Add(-time.Minute * 2),
		EndDate:   time.Now().UTC(),
		Interval:  kline.OneMin,
	}
	require.NoError(t, m.UpsertJob(dhj, false))

	r, err := s.GetActiveDataHistoryJobs(t.Context(), nil)
	require.NoError(t, err)

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
		Pair:      currency.NewBTCUSD(),
		StartDate: time.Now().UTC().Add(-time.Minute * 2),
		EndDate:   time.Now().UTC(),
		Interval:  kline.OneMin,
	}

	_, err := s.GetDataHistoryJobsBetween(t.Context(), nil)
	require.ErrorIs(t, err, errNilRequestData)

	_, err = s.GetDataHistoryJobsBetween(t.Context(), &gctrpc.GetDataHistoryJobsBetweenRequest{
		StartDate: time.Now().UTC().Add(time.Minute).Format(common.SimpleTimeFormatWithTimezone),
		EndDate:   time.Now().UTC().Format(common.SimpleTimeFormatWithTimezone),
	})
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	err = m.UpsertJob(dhj, false)
	require.NoError(t, err)

	r, err := s.GetDataHistoryJobsBetween(t.Context(), &gctrpc.GetDataHistoryJobsBetweenRequest{
		StartDate: time.Now().Add(-time.Minute).UTC().Format(common.SimpleTimeFormatWithTimezone),
		EndDate:   time.Now().Add(time.Minute).UTC().Format(common.SimpleTimeFormatWithTimezone),
	})
	assert.NoError(t, err)

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
		Pair:      currency.NewBTCUSD(),
		StartDate: time.Now().UTC().Add(-time.Minute * 2),
		EndDate:   time.Now().UTC(),
		Interval:  kline.OneMin,
	}
	assert.NoError(t, m.UpsertJob(dhj, false), "UpsertJob should not error")
	_, err := s.GetDataHistoryJobSummary(t.Context(), nil)
	assert.ErrorIs(t, err, errNilRequestData)

	_, err = s.GetDataHistoryJobSummary(t.Context(), &gctrpc.GetDataHistoryJobDetailsRequest{})
	assert.ErrorIs(t, err, errNicknameUnset)

	resp, err := s.GetDataHistoryJobSummary(t.Context(), &gctrpc.GetDataHistoryJobDetailsRequest{Nickname: "TestGetDataHistoryJobSummary"})
	assert.NoError(t, err, "GetDataHistoryJobSummary should not error")
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.Nickname)
	assert.NotEmpty(t, resp.ResultSummaries)
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
	cp := currency.NewBTCUSDT()
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		Available:     currency.Pairs{cp},
		Enabled:       currency.Pairs{cp},
		AssetEnabled:  true,
		ConfigFormat:  &currency.PairFormat{Uppercase: true},
		RequestFormat: &currency.PairFormat{Uppercase: true},
	}
	err = em.Add(exch)
	require.NoError(t, err)

	var wg sync.WaitGroup
	om, err := SetupOrderManager(em, engerino.CommunicationsManager, &wg, &config.OrderManager{})
	assert.NoError(t, err)

	om.started = 1
	s := RPCServer{Engine: &Engine{ExchangeManager: em, OrderManager: om}}

	p := &gctrpc.CurrencyPair{
		Delimiter: "-",
		Base:      currency.BTC.String(),
		Quote:     currency.USDT.String(),
	}

	_, err = s.GetManagedOrders(t.Context(), nil)
	assert.ErrorIs(t, err, errInvalidArguments)

	_, err = s.GetManagedOrders(t.Context(), &gctrpc.GetOrdersRequest{
		AssetType: asset.Spot.String(),
		Pair:      p,
	})
	assert.ErrorIs(t, err, common.ErrExchangeNameNotSet)

	_, err = s.GetManagedOrders(t.Context(), &gctrpc.GetOrdersRequest{
		Exchange:  "bruh",
		AssetType: asset.Spot.String(),
		Pair:      p,
	})
	assert.ErrorIs(t, err, ErrExchangeNotFound)

	_, err = s.GetManagedOrders(t.Context(), &gctrpc.GetOrdersRequest{
		Exchange:  exchName,
		AssetType: asset.Spot.String(),
	})
	assert.ErrorIs(t, err, errCurrencyPairUnset)

	_, err = s.GetManagedOrders(t.Context(), &gctrpc.GetOrdersRequest{
		Exchange: exchName,
		Pair:     p,
	})
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	o := order.Detail{
		Price:     100000,
		Amount:    0.002,
		Exchange:  "Binance",
		Type:      order.Limit,
		Side:      order.Sell,
		Status:    order.New,
		AssetType: asset.Spot,
		Pair:      currency.NewBTCUSDT(),
	}
	err = om.Add(&o)
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	oo, err := s.GetManagedOrders(t.Context(), &gctrpc.GetOrdersRequest{
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
	one, err := server.GetTicker(t.Context(), request)
	if err != nil {
		t.Error(err)
	}
	if want := now.Unix(); one.LastUpdated != want {
		t.Errorf("have %d, want %d", one.LastUpdated, want)
	}

	// Check if timestamp returned is in nanoseconds if TimeInNanoSeconds.
	server.Config.RemoteControl.GRPC.TimeInNanoSeconds = true
	two, err := server.GetTicker(t.Context(), request)
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
	_, err := s.UpdateDataHistoryJobPrerequisite(t.Context(), nil)
	assert.ErrorIs(t, err, errNilRequestData)

	_, err = s.UpdateDataHistoryJobPrerequisite(t.Context(), &gctrpc.UpdateDataHistoryJobPrerequisiteRequest{})
	assert.ErrorIs(t, err, errNicknameUnset)

	_, err = s.UpdateDataHistoryJobPrerequisite(t.Context(), &gctrpc.UpdateDataHistoryJobPrerequisiteRequest{
		Nickname: "test456",
	})
	assert.NoError(t, err)

	_, err = s.UpdateDataHistoryJobPrerequisite(t.Context(), &gctrpc.UpdateDataHistoryJobPrerequisiteRequest{
		Nickname:                "test456",
		PrerequisiteJobNickname: "test123",
	})
	assert.NoError(t, err)
}

func TestCurrencyStateGetAll(t *testing.T) {
	t.Parallel()
	_, err := (&RPCServer{Engine: &Engine{}}).CurrencyStateGetAll(t.Context(),
		&gctrpc.CurrencyStateGetAllRequest{Exchange: fakeExchangeName})
	assert.ErrorIs(t, err, ErrSubSystemNotStarted)
}

func TestCurrencyStateWithdraw(t *testing.T) {
	t.Parallel()
	_, err := (&RPCServer{
		Engine: &Engine{},
	}).CurrencyStateWithdraw(t.Context(),
		&gctrpc.CurrencyStateWithdrawRequest{
			Exchange: "wow", Asset: "meow",
		})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = (&RPCServer{
		Engine: &Engine{},
	}).CurrencyStateWithdraw(t.Context(),
		&gctrpc.CurrencyStateWithdrawRequest{
			Exchange: "wow", Asset: "spot",
		})
	require.ErrorIs(t, err, ErrSubSystemNotStarted)
}

func TestCurrencyStateDeposit(t *testing.T) {
	t.Parallel()
	_, err := (&RPCServer{
		Engine: &Engine{},
	}).CurrencyStateDeposit(t.Context(),
		&gctrpc.CurrencyStateDepositRequest{Exchange: "wow", Asset: "meow"})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = (&RPCServer{
		Engine: &Engine{},
	}).CurrencyStateDeposit(t.Context(),
		&gctrpc.CurrencyStateDepositRequest{Exchange: "wow", Asset: "spot"})
	require.ErrorIs(t, err, ErrSubSystemNotStarted)
}

func TestCurrencyStateTrading(t *testing.T) {
	t.Parallel()
	_, err := (&RPCServer{
		Engine: &Engine{},
	}).CurrencyStateTrading(t.Context(),
		&gctrpc.CurrencyStateTradingRequest{Exchange: "wow", Asset: "meow"})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = (&RPCServer{
		Engine: &Engine{},
	}).CurrencyStateTrading(t.Context(),
		&gctrpc.CurrencyStateTradingRequest{Exchange: "wow", Asset: "spot"})
	require.ErrorIs(t, err, ErrSubSystemNotStarted)
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
		AssetEnabled: true,
		ConfigFormat: &currency.EMPTYFORMAT,
		Available:    currency.Pairs{cp},
		Enabled:      currency.Pairs{cp},
	}
	fakeExchange := fExchange{
		IBotExchange: exch,
	}
	err = em.Add(fakeExchange)
	require.NoError(t, err)

	s := RPCServer{Engine: &Engine{
		ExchangeManager:      em,
		currencyStateManager: &CurrencyStateManager{started: 1, iExchangeManager: em},
	}}

	_, err = s.CurrencyStateTradingPair(t.Context(),
		&gctrpc.CurrencyStateTradingPairRequest{
			Exchange: fakeExchangeName,
			Pair:     "btc-usd",
			Asset:    "spot",
		})
	require.NoError(t, err)
}

func TestGetFuturesPositionsOrders(t *testing.T) {
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
		AssetEnabled:  true,
		RequestFormat: &currency.PairFormat{Delimiter: "-"},
		ConfigFormat:  &currency.PairFormat{Delimiter: "-"},
		Available:     currency.Pairs{cp},
		Enabled:       currency.Pairs{cp},
	}
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		AssetEnabled:  true,
		ConfigFormat:  &currency.PairFormat{Delimiter: "/"},
		RequestFormat: &currency.PairFormat{Delimiter: "/"},
		Available:     currency.Pairs{cp},
		Enabled:       currency.Pairs{cp},
	}
	fakeExchange := fExchange{
		IBotExchange: exch,
	}
	err = em.Add(fakeExchange)
	require.NoError(t, err)

	var wg sync.WaitGroup
	om, err := SetupOrderManager(em, &CommunicationManager{}, &wg, &config.OrderManager{FuturesTrackingSeekDuration: time.Hour})
	assert.NoError(t, err)

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

	_, err = s.GetFuturesPositionsOrders(t.Context(), &gctrpc.GetFuturesPositionsOrdersRequest{
		Exchange: fakeExchangeName,
		Asset:    asset.Futures.String(),
		Pair: &gctrpc.CurrencyPair{
			Delimiter: currency.DashDelimiter,
			Base:      cp.Base.String(),
			Quote:     cp.Quote.String(),
		},
	})
	require.NoError(t, err)

	_, err = s.GetFuturesPositionsOrders(t.Context(), &gctrpc.GetFuturesPositionsOrdersRequest{
		Exchange: fakeExchangeName,
		Asset:    asset.Spot.String(),
		Pair: &gctrpc.CurrencyPair{
			Delimiter: currency.DashDelimiter,
			Base:      cp.Base.String(),
			Quote:     cp.Quote.String(),
		},
	})
	assert.ErrorIs(t, err, futures.ErrNotFuturesAsset)
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
	b.Accounts, err = accounts.GetStore().GetExchangeAccounts(b)
	require.NoError(t, err, "GetExchangeAccounts must not error")

	cp, err := currency.NewPairFromString("btc-usd")
	require.NoError(t, err)

	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Futures] = &currency.PairStore{
		AssetEnabled: true,
		ConfigFormat: &currency.EMPTYFORMAT,
		Available:    currency.Pairs{cp},
		Enabled:      currency.Pairs{cp},
	}
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		AssetEnabled: true,
		ConfigFormat: &currency.EMPTYFORMAT,
		Available:    currency.Pairs{cp},
		Enabled:      currency.Pairs{cp},
	}
	b.Features.Supports.FuturesCapabilities.Collateral = true
	fakeExchange := fExchange{
		IBotExchange: exch,
	}
	err = em.Add(fakeExchange)
	require.NoError(t, err)

	s := RPCServer{
		Engine: &Engine{
			ExchangeManager: em,
			currencyStateManager: &CurrencyStateManager{
				started: 1, iExchangeManager: em,
			},
		},
	}

	_, err = s.GetCollateral(t.Context(), &gctrpc.GetCollateralRequest{
		Exchange: fakeExchangeName,
		Asset:    asset.Futures.String(),
	})
	require.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty)

	ctx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "fakerino", Secret: "supafake"})

	_, err = s.GetCollateral(ctx, &gctrpc.GetCollateralRequest{
		Exchange: fakeExchangeName,
		Asset:    asset.Futures.String(),
	})
	require.ErrorIs(t, err, accounts.ErrNoBalances)

	ctx = accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "fakerino", Secret: "supafake", SubAccount: "1337"})

	r, err := s.GetCollateral(ctx, &gctrpc.GetCollateralRequest{
		Exchange:         fakeExchangeName,
		Asset:            asset.Futures.String(),
		IncludeBreakdown: true,
	})
	require.NoError(t, err)

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
	assert.ErrorIs(t, err, futures.ErrNotFuturesAsset)

	_, err = s.GetCollateral(ctx, &gctrpc.GetCollateralRequest{
		Exchange:         fakeExchangeName,
		Asset:            asset.Futures.String(),
		IncludeBreakdown: true,
		CalculateOffline: true,
	})
	assert.NoError(t, err)
}

func TestShutdown(t *testing.T) {
	t.Parallel()
	s := RPCServer{Engine: &Engine{}}
	_, err := s.Shutdown(t.Context(), &gctrpc.ShutdownRequest{})
	require.ErrorIs(t, err, errShutdownNotAllowed)

	s.Engine.Settings.EnableGRPCShutdown = true
	_, err = s.Shutdown(t.Context(), &gctrpc.ShutdownRequest{})
	require.ErrorIs(t, err, errGRPCShutdownSignalIsNil)

	s.Engine.GRPCShutdownSignal = make(chan struct{}, 1)
	_, err = s.Shutdown(t.Context(), &gctrpc.ShutdownRequest{})
	require.NoError(t, err)
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
	require.NoError(t, err)

	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Futures] = &currency.PairStore{
		AssetEnabled: true,
		ConfigFormat: &currency.PairFormat{},
		Available:    currency.Pairs{cp},
		Enabled:      currency.Pairs{cp},
	}
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		AssetEnabled: true,
		ConfigFormat: &currency.PairFormat{},
		Available:    currency.Pairs{cp},
		Enabled:      currency.Pairs{cp},
	}

	b.Features.Enabled.Kline.Intervals = kline.DeployExchangeIntervals(kline.IntervalCapacity{Interval: kline.OneDay})
	err = em.Add(fExchange{IBotExchange: exch})
	require.NoError(t, err)

	s := RPCServer{
		Engine: &Engine{
			ExchangeManager: em,
			currencyStateManager: &CurrencyStateManager{
				started:          1,
				iExchangeManager: em,
			},
		},
	}

	_, err = s.GetTechnicalAnalysis(t.Context(), &gctrpc.GetTechnicalAnalysisRequest{})
	require.ErrorIs(t, err, common.ErrExchangeNameNotSet)

	_, err = s.GetTechnicalAnalysis(t.Context(), &gctrpc.GetTechnicalAnalysisRequest{
		Exchange: fakeExchangeName,
	})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = s.GetTechnicalAnalysis(t.Context(), &gctrpc.GetTechnicalAnalysisRequest{
		Exchange:  fakeExchangeName,
		AssetType: "upsideprofitcontract",
		Pair:      &gctrpc.CurrencyPair{},
	})
	require.ErrorIs(t, err, errExpectedTestError)

	_, err = s.GetTechnicalAnalysis(t.Context(), &gctrpc.GetTechnicalAnalysisRequest{
		Exchange:  fakeExchangeName,
		AssetType: "spot",
		Pair:      &gctrpc.CurrencyPair{Base: "btc", Quote: "usd"},
		Interval:  int64(kline.OneDay),
	})
	require.ErrorIs(t, err, errInvalidStrategy)

	resp, err := s.GetTechnicalAnalysis(t.Context(), &gctrpc.GetTechnicalAnalysisRequest{
		Exchange:      fakeExchangeName,
		AssetType:     "spot",
		Pair:          &gctrpc.CurrencyPair{Base: "btc", Quote: "usd"},
		Interval:      int64(kline.OneDay),
		AlgorithmType: "twap",
	})
	require.NoError(t, err)

	if resp.Signals["TWAP"].Signals[0] != 1337 {
		t.Fatalf("received: '%v' but expected: '%v'", resp.Signals["TWAP"].Signals[0], 1337)
	}

	resp, err = s.GetTechnicalAnalysis(t.Context(), &gctrpc.GetTechnicalAnalysisRequest{
		Exchange:      fakeExchangeName,
		AssetType:     "spot",
		Pair:          &gctrpc.CurrencyPair{Base: "btc", Quote: "usd"},
		Interval:      int64(kline.OneDay),
		AlgorithmType: "vwap",
	})
	require.NoError(t, err)

	if len(resp.Signals["VWAP"].Signals) != 33 {
		t.Fatalf("received: '%v' but expected: '%v'", len(resp.Signals["VWAP"].Signals), 33)
	}

	resp, err = s.GetTechnicalAnalysis(t.Context(), &gctrpc.GetTechnicalAnalysisRequest{
		Exchange:      fakeExchangeName,
		AssetType:     "spot",
		Pair:          &gctrpc.CurrencyPair{Base: "btc", Quote: "usd"},
		Interval:      int64(kline.OneDay),
		AlgorithmType: "atr",
		Period:        9,
	})
	require.NoError(t, err)

	if len(resp.Signals["ATR"].Signals) != 33 {
		t.Fatalf("received: '%v' but expected: '%v'", len(resp.Signals["ATR"].Signals), 33)
	}

	resp, err = s.GetTechnicalAnalysis(t.Context(), &gctrpc.GetTechnicalAnalysisRequest{
		Exchange:              fakeExchangeName,
		AssetType:             "spot",
		Pair:                  &gctrpc.CurrencyPair{Base: "btc", Quote: "usd"},
		Interval:              int64(kline.OneDay),
		AlgorithmType:         "bbands",
		Period:                9,
		StandardDeviationUp:   0.5,
		StandardDeviationDown: 0.5,
	})
	require.NoError(t, err)

	if len(resp.Signals["UPPER"].Signals) != 33 {
		t.Fatalf("received: '%v' but expected: '%v'", len(resp.Signals["UPPER"].Signals), 33)
	}

	if len(resp.Signals["MIDDLE"].Signals) != 33 {
		t.Fatalf("received: '%v' but expected: '%v'", len(resp.Signals["MIDDLE"].Signals), 33)
	}

	if len(resp.Signals["LOWER"].Signals) != 33 {
		t.Fatalf("received: '%v' but expected: '%v'", len(resp.Signals["LOWER"].Signals), 33)
	}

	resp, err = s.GetTechnicalAnalysis(t.Context(), &gctrpc.GetTechnicalAnalysisRequest{
		Exchange:      fakeExchangeName,
		AssetType:     "spot",
		Pair:          &gctrpc.CurrencyPair{Base: "btc", Quote: "usd"},
		OtherPair:     &gctrpc.CurrencyPair{Base: "btc", Quote: "usd"},
		Interval:      int64(kline.OneDay),
		AlgorithmType: "COCO",
		Period:        9,
	})
	require.NoError(t, err)

	if len(resp.Signals["COCO"].Signals) != 33 {
		t.Fatalf("received: '%v' but expected: '%v'", len(resp.Signals["COCO"].Signals), 33)
	}

	resp, err = s.GetTechnicalAnalysis(t.Context(), &gctrpc.GetTechnicalAnalysisRequest{
		Exchange:      fakeExchangeName,
		AssetType:     "spot",
		Pair:          &gctrpc.CurrencyPair{Base: "btc", Quote: "usd"},
		Interval:      int64(kline.OneDay),
		AlgorithmType: "sma",
		Period:        9,
	})
	require.NoError(t, err)

	if len(resp.Signals["SMA"].Signals) != 33 {
		t.Fatalf("received: '%v' but expected: '%v'", len(resp.Signals["SMA"].Signals), 33)
	}

	resp, err = s.GetTechnicalAnalysis(t.Context(), &gctrpc.GetTechnicalAnalysisRequest{
		Exchange:      fakeExchangeName,
		AssetType:     "spot",
		Pair:          &gctrpc.CurrencyPair{Base: "btc", Quote: "usd"},
		Interval:      int64(kline.OneDay),
		AlgorithmType: "ema",
		Period:        9,
	})
	require.NoError(t, err)

	if len(resp.Signals["EMA"].Signals) != 33 {
		t.Fatalf("received: '%v' but expected: '%v'", len(resp.Signals["EMA"].Signals), 33)
	}

	resp, err = s.GetTechnicalAnalysis(t.Context(), &gctrpc.GetTechnicalAnalysisRequest{
		Exchange:      fakeExchangeName,
		AssetType:     "spot",
		Pair:          &gctrpc.CurrencyPair{Base: "btc", Quote: "usd"},
		Interval:      int64(kline.OneDay),
		AlgorithmType: "macd",
		Period:        9,
		FastPeriod:    12,
		SlowPeriod:    26,
	})
	require.NoError(t, err)

	if len(resp.Signals["MACD"].Signals) != 33 {
		t.Fatalf("received: '%v' but expected: '%v'", len(resp.Signals["MACD"].Signals), 33)
	}

	if len(resp.Signals["SIGNAL"].Signals) != 33 {
		t.Fatalf("received: '%v' but expected: '%v'", len(resp.Signals["SIGNAL"].Signals), 33)
	}

	if len(resp.Signals["HISTOGRAM"].Signals) != 33 {
		t.Fatalf("received: '%v' but expected: '%v'", len(resp.Signals["HISTOGRAM"].Signals), 33)
	}

	resp, err = s.GetTechnicalAnalysis(t.Context(), &gctrpc.GetTechnicalAnalysisRequest{
		Exchange:      fakeExchangeName,
		AssetType:     "spot",
		Pair:          &gctrpc.CurrencyPair{Base: "btc", Quote: "usd"},
		Interval:      int64(kline.OneDay),
		AlgorithmType: "mfi",
		Period:        9,
	})
	require.NoError(t, err)

	if len(resp.Signals["MFI"].Signals) != 33 {
		t.Fatalf("received: '%v' but expected: '%v'", len(resp.Signals["MFI"].Signals), 33)
	}

	resp, err = s.GetTechnicalAnalysis(t.Context(), &gctrpc.GetTechnicalAnalysisRequest{
		Exchange:      fakeExchangeName,
		AssetType:     "spot",
		Pair:          &gctrpc.CurrencyPair{Base: "btc", Quote: "usd"},
		Interval:      int64(kline.OneDay),
		AlgorithmType: "obv",
		Period:        9,
	})
	require.NoError(t, err)

	if len(resp.Signals["OBV"].Signals) != 33 {
		t.Fatalf("received: '%v' but expected: '%v'", len(resp.Signals["OBV"].Signals), 33)
	}

	resp, err = s.GetTechnicalAnalysis(t.Context(), &gctrpc.GetTechnicalAnalysisRequest{
		Exchange:      fakeExchangeName,
		AssetType:     "spot",
		Pair:          &gctrpc.CurrencyPair{Base: "btc", Quote: "usd"},
		Interval:      int64(kline.OneDay),
		AlgorithmType: "rsi",
		Period:        9,
	})
	require.NoError(t, err)

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
	require.NoError(t, err)

	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		AssetEnabled: true,
		ConfigFormat: &currency.PairFormat{},
		Available:    currency.Pairs{cp},
		Enabled:      currency.Pairs{cp},
	}
	fakeExchange := fExchange{
		IBotExchange: exch,
	}
	err = em.Add(fakeExchange)
	require.NoError(t, err)

	s := RPCServer{
		Engine: &Engine{
			ExchangeManager: em,
			currencyStateManager: &CurrencyStateManager{
				started: 1, iExchangeManager: em,
			},
		},
	}
	_, err = s.GetMarginRatesHistory(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)

	request := &gctrpc.GetMarginRatesHistoryRequest{}
	_, err = s.GetMarginRatesHistory(t.Context(), request)
	assert.ErrorIs(t, err, common.ErrExchangeNameNotSet)

	request.Exchange = fakeExchangeName
	_, err = s.GetMarginRatesHistory(t.Context(), request)
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	request.Asset = asset.Spot.String()
	_, err = s.GetMarginRatesHistory(t.Context(), request)
	assert.ErrorIs(t, err, currency.ErrCurrencyNotFound)

	request.Currency = "usd"
	_, err = s.GetMarginRatesHistory(t.Context(), request)
	assert.NoError(t, err)

	request.GetBorrowRates = true
	request.GetLendingPayments = true
	request.GetBorrowCosts = true
	request.GetPredictedRate = true
	request.IncludeAllRates = true
	resp, err := s.GetMarginRatesHistory(t.Context(), request)
	assert.NoError(t, err)

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
	_, err = s.GetMarginRatesHistory(t.Context(), request)
	assert.ErrorIs(t, err, common.ErrCannotCalculateOffline)

	request.TakerFeeRate = "-1337"
	_, err = s.GetMarginRatesHistory(t.Context(), request)
	assert.ErrorIs(t, err, common.ErrCannotCalculateOffline)

	request.TakerFeeRate = "1337"
	_, err = s.GetMarginRatesHistory(t.Context(), request)
	assert.ErrorIs(t, err, common.ErrCannotCalculateOffline)

	request.Rates = []*gctrpc.MarginRate{
		{
			Time:       time.Now().Format(common.SimpleTimeFormatWithTimezone),
			HourlyRate: "1337",
		},
	}
	_, err = s.GetMarginRatesHistory(t.Context(), request)
	assert.NoError(t, err)

	request.Rates = []*gctrpc.MarginRate{
		{
			Time:           time.Now().Format(common.SimpleTimeFormatWithTimezone),
			HourlyRate:     "1337",
			LendingPayment: &gctrpc.LendingPayment{Size: "1337"},
			BorrowCost:     &gctrpc.BorrowCost{Size: "1337"},
		},
	}
	_, err = s.GetMarginRatesHistory(t.Context(), request)
	assert.NoError(t, err)
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
	err = b.CurrencyPairs.Store(asset.Futures, &currency.PairStore{
		AssetEnabled:  true,
		RequestFormat: &currency.PairFormat{Delimiter: "-"},
		ConfigFormat:  &currency.PairFormat{Delimiter: "-"},
		Available:     currency.Pairs{cp},
		Enabled:       currency.Pairs{cp},
	})
	if err != nil {
		t.Fatal(err)
	}

	err = b.CurrencyPairs.Store(asset.Spot, &currency.PairStore{
		AssetEnabled:  true,
		ConfigFormat:  &currency.PairFormat{Delimiter: "/"},
		RequestFormat: &currency.PairFormat{Delimiter: "/"},
		Available:     currency.Pairs{cp},
		Enabled:       currency.Pairs{cp},
	})
	if err != nil {
		t.Fatal(err)
	}
	b.Features.Supports.FuturesCapabilities.FundingRates = true
	fakeExchange := fExchange{
		IBotExchange: exch,
	}
	err = em.Add(fakeExchange)
	require.NoError(t, err)

	var wg sync.WaitGroup
	om, err := SetupOrderManager(em, &CommunicationManager{}, &wg, &config.OrderManager{FuturesTrackingSeekDuration: time.Hour})
	assert.NoError(t, err)

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

	_, err = s.GetFundingRates(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)

	request := &gctrpc.GetFundingRatesRequest{
		Exchange:         "",
		Asset:            "",
		Pair:             nil,
		StartDate:        "",
		EndDate:          "",
		IncludePredicted: false,
		IncludePayments:  false,
	}
	_, err = s.GetFundingRates(t.Context(), request)
	assert.ErrorIs(t, err, common.ErrExchangeNameNotSet)

	request.Exchange = exch.GetName()
	_, err = s.GetFundingRates(t.Context(), request)
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	request.Asset = asset.Spot.String()
	_, err = s.GetFundingRates(t.Context(), request)
	assert.ErrorIs(t, err, futures.ErrNotFuturesAsset)

	request.Asset = asset.Futures.String()
	request.Pair = &gctrpc.CurrencyPair{
		Delimiter: cp.Delimiter,
		Base:      cp.Base.String(),
		Quote:     cp.Quote.String(),
	}
	request.IncludePredicted = true
	request.IncludePayments = true
	_, err = s.GetFundingRates(t.Context(), request)
	assert.NoError(t, err)
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
	err = b.CurrencyPairs.Store(asset.Futures, &currency.PairStore{
		AssetEnabled:  true,
		RequestFormat: &currency.PairFormat{Delimiter: "-"},
		ConfigFormat:  &currency.PairFormat{Delimiter: "-"},
		Available:     currency.Pairs{cp},
		Enabled:       currency.Pairs{cp},
	})
	if err != nil {
		t.Fatal(err)
	}
	err = b.CurrencyPairs.Store(asset.Spot, &currency.PairStore{
		AssetEnabled:  true,
		ConfigFormat:  &currency.PairFormat{Delimiter: "/"},
		RequestFormat: &currency.PairFormat{Delimiter: "/"},
		Available:     currency.Pairs{cp},
		Enabled:       currency.Pairs{cp},
	})
	if err != nil {
		t.Fatal(err)
	}
	fakeExchange := fExchange{
		IBotExchange: exch,
	}
	err = em.Add(fakeExchange)
	require.NoError(t, err)

	var wg sync.WaitGroup
	om, err := SetupOrderManager(em, &CommunicationManager{}, &wg, &config.OrderManager{FuturesTrackingSeekDuration: time.Hour})
	assert.NoError(t, err)

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

	_, err = s.GetLatestFundingRate(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)

	request := &gctrpc.GetLatestFundingRateRequest{
		Exchange:         "",
		Asset:            "",
		Pair:             nil,
		IncludePredicted: false,
	}
	_, err = s.GetLatestFundingRate(t.Context(), request)
	assert.ErrorIs(t, err, common.ErrExchangeNameNotSet)

	request.Exchange = exch.GetName()
	_, err = s.GetLatestFundingRate(t.Context(), request)
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	request.Asset = asset.Spot.String()
	_, err = s.GetLatestFundingRate(t.Context(), request)
	assert.ErrorIs(t, err, futures.ErrNotFuturesAsset)

	request.Asset = asset.Futures.String()
	request.Pair = &gctrpc.CurrencyPair{
		Delimiter: cp.Delimiter,
		Base:      cp.Base.String(),
		Quote:     cp.Quote.String(),
	}
	request.IncludePredicted = true
	_, err = s.GetLatestFundingRate(t.Context(), request)
	assert.NoError(t, err)
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
		AssetEnabled:  true,
		RequestFormat: &currency.PairFormat{Delimiter: "-"},
		ConfigFormat:  &currency.PairFormat{Delimiter: "-"},
		Available:     currency.Pairs{cp, cp2},
		Enabled:       currency.Pairs{cp, cp2},
	}
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		AssetEnabled:  true,
		ConfigFormat:  &currency.PairFormat{Delimiter: "/"},
		RequestFormat: &currency.PairFormat{Delimiter: "/"},
		Available:     currency.Pairs{cp, cp2},
		Enabled:       currency.Pairs{cp, cp2},
	}
	b.Features.Supports.FuturesCapabilities.OrderManagerPositionTracking = true
	fakeExchange := fExchange{
		IBotExchange: exch,
	}
	err = em.Add(fakeExchange)
	require.NoError(t, err)

	var wg sync.WaitGroup
	om, err := SetupOrderManager(em, &CommunicationManager{}, &wg, &config.OrderManager{FuturesTrackingSeekDuration: time.Hour})
	assert.NoError(t, err)

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
	_, err = s.GetManagedPosition(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)

	request := &gctrpc.GetManagedPositionRequest{}
	_, err = s.GetManagedPosition(t.Context(), request)
	assert.ErrorIs(t, err, common.ErrNilPointer)

	request.Pair = &gctrpc.CurrencyPair{
		Delimiter: "-",
		Base:      "BTC",
		Quote:     "USD",
	}
	_, err = s.GetManagedPosition(t.Context(), request)
	assert.ErrorIs(t, err, common.ErrExchangeNameNotSet)

	request.Exchange = fakeExchangeName
	_, err = s.GetManagedPosition(t.Context(), request)
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	request.Asset = asset.Spot.String()
	_, err = s.GetManagedPosition(t.Context(), request)
	assert.ErrorIs(t, err, futures.ErrNotFuturesAsset)

	request.Asset = asset.Futures.String()
	s.OrderManager, err = SetupOrderManager(em, &CommunicationManager{}, &wg, &config.OrderManager{FuturesTrackingSeekDuration: time.Hour})
	assert.NoError(t, err)

	s.OrderManager.started = 1
	s.OrderManager.activelyTrackFuturesPositions = true
	_, err = s.GetManagedPosition(t.Context(), request)
	assert.ErrorIs(t, err, futures.ErrPositionNotFound)

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
	assert.NoError(t, err)

	_, err = s.GetManagedPosition(t.Context(), request)
	assert.NoError(t, err)
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
		AssetEnabled:  true,
		RequestFormat: &currency.PairFormat{Delimiter: "-"},
		ConfigFormat:  &currency.PairFormat{Delimiter: "-"},
		Available:     currency.Pairs{cp, cp2},
		Enabled:       currency.Pairs{cp, cp2},
	}
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		AssetEnabled:  true,
		ConfigFormat:  &currency.PairFormat{Delimiter: "/"},
		RequestFormat: &currency.PairFormat{Delimiter: "/"},
		Available:     currency.Pairs{cp, cp2},
		Enabled:       currency.Pairs{cp, cp2},
	}
	fakeExchange := fExchange{
		IBotExchange: exch,
	}
	err = em.Add(fakeExchange)
	require.NoError(t, err)

	var wg sync.WaitGroup
	om, err := SetupOrderManager(em, &CommunicationManager{}, &wg, &config.OrderManager{FuturesTrackingSeekDuration: time.Hour})
	assert.NoError(t, err)

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
	_, err = s.GetAllManagedPositions(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)

	request := &gctrpc.GetAllManagedPositionsRequest{}
	s.OrderManager, err = SetupOrderManager(em, &CommunicationManager{}, &wg, &config.OrderManager{FuturesTrackingSeekDuration: time.Hour, ActivelyTrackFuturesPositions: true})
	assert.NoError(t, err)

	s.OrderManager.started = 1
	_, err = s.GetAllManagedPositions(t.Context(), request)
	assert.ErrorIs(t, err, futures.ErrNoPositionsFound)

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
	assert.NoError(t, err)

	request.IncludePredictedRate = true
	request.GetFundingPayments = true
	request.IncludeFullFundingRates = true
	request.IncludeFullOrderData = true
	_, err = s.GetAllManagedPositions(t.Context(), request)
	assert.NoError(t, err)
}

func TestGetOrderbookMovement(t *testing.T) {
	t.Parallel()
	em := NewExchangeManager()
	exch, err := em.NewExchangeByName("binance")
	require.NoError(t, err, "NewExchangeByName must not error")

	exch.SetDefaults()
	b := exch.GetBase()
	b.Name = fakeExchangeName
	b.Enabled = true

	cp := currency.NewPairWithDelimiter("btc", "metal", "-")

	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		AssetEnabled:  true,
		ConfigFormat:  &currency.PairFormat{Delimiter: "/"},
		RequestFormat: &currency.PairFormat{Delimiter: "/"},
		Available:     currency.Pairs{cp},
		Enabled:       currency.Pairs{cp},
	}

	fakeExchange := fExchange{
		IBotExchange: exch,
	}
	err = em.Add(fakeExchange)
	require.NoError(t, err)

	s := RPCServer{Engine: &Engine{ExchangeManager: em}}

	req := &gctrpc.GetOrderbookMovementRequest{}
	_, err = s.GetOrderbookMovement(t.Context(), req)
	require.ErrorIs(t, err, common.ErrExchangeNameNotSet)

	req.Exchange = "fake"
	_, err = s.GetOrderbookMovement(t.Context(), req)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	req.Asset = asset.Spot.String()
	req.Pair = &gctrpc.CurrencyPair{}
	_, err = s.GetOrderbookMovement(t.Context(), req)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	req.Pair = &gctrpc.CurrencyPair{
		Base:  currency.BTC.String(),
		Quote: currency.METAL.String(),
	}
	_, err = s.GetOrderbookMovement(t.Context(), req)
	if !strings.Contains(err.Error(), "cannot find orderbook") {
		t.Fatalf("received: '%+v' but expected: '%v'", err, "cannot find orderbook")
	}

	depth, err := orderbook.DeployDepth(req.Exchange, currency.NewPair(currency.BTC, currency.METAL), asset.Spot)
	require.NoError(t, err, "orderbook.DeployDepth must not error")

	bid := []orderbook.Level{
		{Price: 10, Amount: 1},
		{Price: 9, Amount: 1},
		{Price: 8, Amount: 1},
		{Price: 7, Amount: 1},
	}
	ask := []orderbook.Level{
		{Price: 11, Amount: 1},
		{Price: 12, Amount: 1},
		{Price: 13, Amount: 1},
		{Price: 14, Amount: 1},
	}
	err = depth.LoadSnapshot(&orderbook.Book{Bids: bid, Asks: ask, LastUpdated: time.Now(), LastPushed: time.Now(), RestSnapshot: true})
	require.NoError(t, err, "depth.LoadSnapshot must not error")

	_, err = s.GetOrderbookMovement(t.Context(), req)
	if err.Error() != "quote amount invalid" {
		t.Fatalf("received: '%+v' but expected: '%v'", err, "quote amount invalid")
	}

	req.Amount = 11
	move, err := s.GetOrderbookMovement(t.Context(), req)
	require.NoError(t, err)

	if move.Bought != 1 {
		t.Fatalf("received: '%v' but expected: '%v'", move.Bought, 1)
	}

	req.Sell = true
	req.Amount = 1
	move, err = s.GetOrderbookMovement(t.Context(), req)
	require.NoError(t, err)

	if move.Bought != 10 {
		t.Fatalf("received: '%v' but expected: '%v'", move.Bought, 10)
	}
}

func TestGetOrderbookAmountByNominal(t *testing.T) {
	t.Parallel()
	em := NewExchangeManager()
	exch, err := em.NewExchangeByName("binance")
	require.NoError(t, err, "NewExchangeByName must not error")

	exch.SetDefaults()
	b := exch.GetBase()
	b.Name = fakeExchangeName
	b.Enabled = true

	cp := currency.NewPairWithDelimiter("btc", "meme", "-")

	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		AssetEnabled:  true,
		ConfigFormat:  &currency.PairFormat{Delimiter: "/"},
		RequestFormat: &currency.PairFormat{Delimiter: "/"},
		Available:     currency.Pairs{cp},
		Enabled:       currency.Pairs{cp},
	}

	fakeExchange := fExchange{
		IBotExchange: exch,
	}
	err = em.Add(fakeExchange)
	require.NoError(t, err)

	s := RPCServer{Engine: &Engine{ExchangeManager: em}}

	req := &gctrpc.GetOrderbookAmountByNominalRequest{}
	_, err = s.GetOrderbookAmountByNominal(t.Context(), req)
	require.ErrorIs(t, err, common.ErrExchangeNameNotSet)

	req.Exchange = "fake"
	_, err = s.GetOrderbookAmountByNominal(t.Context(), req)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	req.Asset = asset.Spot.String()
	req.Pair = &gctrpc.CurrencyPair{}
	_, err = s.GetOrderbookAmountByNominal(t.Context(), req)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	req.Pair = &gctrpc.CurrencyPair{
		Base:  currency.BTC.String(),
		Quote: currency.MEME.String(),
	}
	_, err = s.GetOrderbookAmountByNominal(t.Context(), req)
	if !strings.Contains(err.Error(), "cannot find orderbook") {
		t.Fatalf("received: '%+v' but expected: '%v'", err, "cannot find orderbook")
	}

	depth, err := orderbook.DeployDepth(req.Exchange, currency.NewPair(currency.BTC, currency.MEME), asset.Spot)
	require.NoError(t, err, "orderbook.DeployDepth must not error")

	bid := []orderbook.Level{
		{Price: 10, Amount: 1},
		{Price: 9, Amount: 1},
		{Price: 8, Amount: 1},
		{Price: 7, Amount: 1},
	}
	ask := []orderbook.Level{
		{Price: 11, Amount: 1},
		{Price: 12, Amount: 1},
		{Price: 13, Amount: 1},
		{Price: 14, Amount: 1},
	}
	err = depth.LoadSnapshot(&orderbook.Book{Bids: bid, Asks: ask, LastUpdated: time.Now(), LastPushed: time.Now(), RestSnapshot: true})
	require.NoError(t, err, "depth.LoadSnapshot must not error")

	nominal, err := s.GetOrderbookAmountByNominal(t.Context(), req)
	require.NoError(t, err)

	if nominal.AmountRequired != 11 {
		t.Fatalf("received: '%v' but expected: '%v'", nominal.AmountRequired, 11)
	}

	req.Sell = true
	nominal, err = s.GetOrderbookAmountByNominal(t.Context(), req)
	require.NoError(t, err)

	if nominal.AmountRequired != 1 {
		t.Fatalf("received: '%v' but expected: '%v'", nominal.AmountRequired, 1)
	}
}

func TestGetOrderbookAmountByImpact(t *testing.T) {
	t.Parallel()
	em := NewExchangeManager()
	exch, err := em.NewExchangeByName("binance")
	require.NoError(t, err, "NewExchangeByName must not error")

	exch.SetDefaults()
	b := exch.GetBase()
	b.Name = fakeExchangeName
	b.Enabled = true

	cp := currency.NewPairWithDelimiter("btc", "mad", "-")

	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		AssetEnabled:  true,
		ConfigFormat:  &currency.PairFormat{Delimiter: "/"},
		RequestFormat: &currency.PairFormat{Delimiter: "/"},
		Available:     currency.Pairs{cp},
		Enabled:       currency.Pairs{cp},
	}

	fakeExchange := fExchange{
		IBotExchange: exch,
	}
	err = em.Add(fakeExchange)
	require.NoError(t, err)

	s := RPCServer{Engine: &Engine{ExchangeManager: em}}

	req := &gctrpc.GetOrderbookAmountByImpactRequest{}
	_, err = s.GetOrderbookAmountByImpact(t.Context(), req)
	require.ErrorIs(t, err, common.ErrExchangeNameNotSet)

	req.Exchange = "fake"
	_, err = s.GetOrderbookAmountByImpact(t.Context(), req)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	req.Asset = asset.Spot.String()
	req.Pair = &gctrpc.CurrencyPair{}
	_, err = s.GetOrderbookAmountByImpact(t.Context(), req)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	req.Pair = &gctrpc.CurrencyPair{
		Base:  currency.BTC.String(),
		Quote: currency.MAD.String(),
	}
	_, err = s.GetOrderbookAmountByImpact(t.Context(), req)
	if !strings.Contains(err.Error(), "cannot find orderbook") {
		t.Fatalf("received: '%+v' but expected: '%v'", err, "cannot find orderbook")
	}

	depth, err := orderbook.DeployDepth(req.Exchange, currency.NewPair(currency.BTC, currency.MAD), asset.Spot)
	require.NoError(t, err, "orderbook.DeployDepth must not error")

	bid := []orderbook.Level{
		{Price: 10, Amount: 1},
		{Price: 9, Amount: 1},
		{Price: 8, Amount: 1},
		{Price: 7, Amount: 1},
	}
	ask := []orderbook.Level{
		{Price: 11, Amount: 1},
		{Price: 12, Amount: 1},
		{Price: 13, Amount: 1},
		{Price: 14, Amount: 1},
	}
	err = depth.LoadSnapshot(&orderbook.Book{Bids: bid, Asks: ask, LastUpdated: time.Now(), LastPushed: time.Now(), RestSnapshot: true})
	require.NoError(t, err, "depth.LoadSnapshot must not error")

	req.ImpactPercentage = 9.090909090909092
	impact, err := s.GetOrderbookAmountByImpact(t.Context(), req)
	require.NoError(t, err)

	if impact.AmountRequired != 11 {
		t.Fatalf("received: '%v' but expected: '%v'", impact.AmountRequired, 11)
	}

	req.Sell = true
	req.ImpactPercentage = 10
	impact, err = s.GetOrderbookAmountByImpact(t.Context(), req)
	require.NoError(t, err)

	if impact.AmountRequired != 1 {
		t.Fatalf("received: '%v' but expected: '%v'", impact.AmountRequired, 1)
	}
}

func TestChangePositionMargin(t *testing.T) {
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
	b.CurrencyPairs.Pairs[asset.USDTMarginedFutures] = &currency.PairStore{
		AssetEnabled:  true,
		ConfigFormat:  &currency.PairFormat{Delimiter: "/"},
		RequestFormat: &currency.PairFormat{Delimiter: "/"},
		Available:     currency.Pairs{cp},
		Enabled:       currency.Pairs{cp},
	}

	fakeExchange := fExchange{
		IBotExchange: exch,
	}
	err = em.Add(fakeExchange)
	require.NoError(t, err)

	s := RPCServer{Engine: &Engine{ExchangeManager: em}}
	_, err = s.ChangePositionMargin(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)

	req := &gctrpc.ChangePositionMarginRequest{}
	_, err = s.ChangePositionMargin(t.Context(), req)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	req.Exchange = fakeExchangeName
	req.Pair = &gctrpc.CurrencyPair{
		Delimiter: "-",
		Base:      cp.Base.String(),
		Quote:     cp.Quote.String(),
	}
	req.Asset = asset.USDTMarginedFutures.String()
	req.MarginSide = "BOTH"
	req.OriginalAllocatedMargin = 1337
	req.NewAllocatedMargin = 1338
	req.MarginType = "isolated"
	_, err = s.ChangePositionMargin(t.Context(), req)
	assert.NoError(t, err)
}

func TestSetLeverage(t *testing.T) {
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
	b.CurrencyPairs.Pairs[asset.USDTMarginedFutures] = &currency.PairStore{
		AssetEnabled:  true,
		ConfigFormat:  &currency.PairFormat{Delimiter: "/"},
		RequestFormat: &currency.PairFormat{Delimiter: "/"},
		Available:     currency.Pairs{cp},
		Enabled:       currency.Pairs{cp},
	}

	fakeExchange := fExchange{
		IBotExchange: exch,
	}
	err = em.Add(fakeExchange)
	require.NoError(t, err)

	s := RPCServer{Engine: &Engine{ExchangeManager: em}}
	_, err = s.SetLeverage(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)

	req := &gctrpc.SetLeverageRequest{}
	_, err = s.SetLeverage(t.Context(), req)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	req.Exchange = fakeExchangeName
	req.Pair = &gctrpc.CurrencyPair{
		Delimiter: "-",
		Base:      cp.Base.String(),
		Quote:     cp.Quote.String(),
	}
	req.UnderlyingPair = &gctrpc.CurrencyPair{
		Delimiter: "-",
		Base:      cp.Base.String(),
		Quote:     cp.Quote.String(),
	}
	req.Asset = asset.USDTMarginedFutures.String()
	req.MarginType = "isolated"
	req.Leverage = 1337
	_, err = s.SetLeverage(t.Context(), req)
	assert.NoError(t, err)

	req.OrderSide = "lol"
	_, err = s.SetLeverage(t.Context(), req)
	assert.ErrorIs(t, err, order.ErrSideIsInvalid)

	req.OrderSide = order.Long.String()
	_, err = s.SetLeverage(t.Context(), req)
	assert.NoError(t, err)
}

func TestGetLeverage(t *testing.T) {
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
	b.CurrencyPairs.Pairs[asset.USDTMarginedFutures] = &currency.PairStore{
		AssetEnabled:  true,
		ConfigFormat:  &currency.PairFormat{Delimiter: "/"},
		RequestFormat: &currency.PairFormat{Delimiter: "/"},
		Available:     currency.Pairs{cp},
		Enabled:       currency.Pairs{cp},
	}

	fakeExchange := fExchange{
		IBotExchange: exch,
	}
	err = em.Add(fakeExchange)
	require.NoError(t, err)

	s := RPCServer{Engine: &Engine{ExchangeManager: em}}
	_, err = s.GetLeverage(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)

	req := &gctrpc.GetLeverageRequest{}
	_, err = s.GetLeverage(t.Context(), req)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	req.Exchange = fakeExchangeName
	req.Pair = &gctrpc.CurrencyPair{
		Delimiter: "-",
		Base:      cp.Base.String(),
		Quote:     cp.Quote.String(),
	}
	req.UnderlyingPair = &gctrpc.CurrencyPair{
		Delimiter: "-",
		Base:      cp.Base.String(),
		Quote:     cp.Quote.String(),
	}
	req.Asset = asset.USDTMarginedFutures.String()
	req.MarginType = "isolated"
	lev, err := s.GetLeverage(t.Context(), req)
	assert.NoError(t, err)

	if lev.Leverage != 1337 {
		t.Errorf("received '%v' expected '%v'", lev, 1337)
	}

	req.OrderSide = "lol"
	_, err = s.GetLeverage(t.Context(), req)
	assert.ErrorIs(t, err, order.ErrSideIsInvalid)

	req.OrderSide = order.Long.String()
	_, err = s.GetLeverage(t.Context(), req)
	assert.NoError(t, err)
}

func TestSetMarginType(t *testing.T) {
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
	b.CurrencyPairs.Pairs[asset.USDTMarginedFutures] = &currency.PairStore{
		AssetEnabled:  true,
		ConfigFormat:  &currency.PairFormat{Delimiter: "/"},
		RequestFormat: &currency.PairFormat{Delimiter: "/"},
		Available:     currency.Pairs{cp},
		Enabled:       currency.Pairs{cp},
	}

	fakeExchange := fExchange{
		IBotExchange: exch,
	}
	err = em.Add(fakeExchange)
	require.NoError(t, err)

	s := RPCServer{Engine: &Engine{ExchangeManager: em}}
	_, err = s.SetMarginType(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)

	req := &gctrpc.SetMarginTypeRequest{}
	_, err = s.SetMarginType(t.Context(), req)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	req.Exchange = fakeExchangeName
	req.Pair = &gctrpc.CurrencyPair{
		Delimiter: "-",
		Base:      cp.Base.String(),
		Quote:     cp.Quote.String(),
	}
	req.Asset = asset.USDTMarginedFutures.String()
	req.MarginType = "isolated"
	_, err = s.SetMarginType(t.Context(), req)
	assert.NoError(t, err)
}

func TestSetCollateralMode(t *testing.T) {
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
	b.CurrencyPairs.Pairs[asset.USDTMarginedFutures] = &currency.PairStore{
		AssetEnabled:  true,
		ConfigFormat:  &currency.PairFormat{Delimiter: "/"},
		RequestFormat: &currency.PairFormat{Delimiter: "/"},
		Available:     currency.Pairs{cp},
		Enabled:       currency.Pairs{cp},
	}

	fakeExchange := fExchange{
		IBotExchange: exch,
	}
	err = em.Add(fakeExchange)
	require.NoError(t, err)

	s := RPCServer{Engine: &Engine{ExchangeManager: em}}
	_, err = s.SetCollateralMode(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)

	req := &gctrpc.SetCollateralModeRequest{}
	_, err = s.SetCollateralMode(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrExchangeNameNotSet)

	req.Exchange = fakeExchangeName
	req.Asset = asset.USDTMarginedFutures.String()
	req.CollateralMode = "single"
	_, err = s.SetCollateralMode(t.Context(), req)
	assert.NoError(t, err)
}

func TestGetCollateralMode(t *testing.T) {
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
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.USDTMarginedFutures] = &currency.PairStore{
		AssetEnabled: true,
	}

	fakeExchange := fExchange{
		IBotExchange: exch,
	}
	err = em.Add(fakeExchange)
	require.NoError(t, err)

	s := RPCServer{Engine: &Engine{ExchangeManager: em}}
	_, err = s.GetCollateralMode(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)

	req := &gctrpc.GetCollateralModeRequest{}
	_, err = s.GetCollateralMode(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrExchangeNameNotSet)

	req.Exchange = fakeExchangeName
	req.Asset = asset.USDTMarginedFutures.String()
	_, err = s.GetCollateralMode(t.Context(), req)
	assert.NoError(t, err)
}

func TestGetOpenInterest(t *testing.T) {
	t.Parallel()
	em := NewExchangeManager()
	exch, err := em.NewExchangeByName("binance")
	assert.NoError(t, err)

	exch.SetDefaults()
	b := exch.GetBase()
	b.Name = fakeExchangeName
	b.Enabled = true
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.USDTMarginedFutures] = &currency.PairStore{
		AssetEnabled: true,
	}

	fakeExchange := fExchange{
		IBotExchange: exch,
	}
	err = em.Add(fakeExchange)
	assert.NoError(t, err)

	s := RPCServer{Engine: &Engine{ExchangeManager: em}}
	_, err = s.GetOpenInterest(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)

	req := &gctrpc.GetOpenInterestRequest{}
	_, err = s.GetOpenInterest(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrExchangeNameNotSet)

	req.Exchange = fakeExchangeName
	_, err = s.GetOpenInterest(t.Context(), req)
	assert.NoError(t, err)

	req.Data = append(req.Data, &gctrpc.OpenInterestDataRequest{
		Asset: asset.USDTMarginedFutures.String(),
		Pair:  &gctrpc.CurrencyPair{Base: currency.BTC.String(), Quote: currency.USDT.String()},
	})
	_, err = s.GetOpenInterest(t.Context(), req)
	assert.NoError(t, err)
}

func TestStartRPCRESTProxy(t *testing.T) {
	t.Parallel()

	tempDir := filepath.Join(os.TempDir(), "gct-grpc-proxy-test")
	tempDirTLS := filepath.Join(tempDir, "tls")

	t.Cleanup(func() {
		assert.NoErrorf(t, os.RemoveAll(tempDir), "RemoveAll should not error, manual directory deletion required for TempDir: %s", tempDir)
	})

	if !assert.NoError(t, genCert(tempDirTLS), "genCert should not error") {
		t.FailNow()
	}

	gRPCPort := rand.Intn(65535-42069) + 42069 //nolint:gosec // Don't require crypto/rand usage here
	gRPCProxyPort := gRPCPort + 1

	e := &Engine{
		Config: &config.Config{
			RemoteControl: config.RemoteControlConfig{
				Username: "bobmarley",
				Password: "Sup3rdup3rS3cr3t",
				GRPC: config.GRPCConfig{
					Enabled:                true,
					ListenAddress:          "localhost:" + strconv.Itoa(gRPCPort),
					GRPCProxyListenAddress: "localhost:" + strconv.Itoa(gRPCProxyPort),
				},
			},
		},
		Settings: Settings{
			DataDir:      tempDir,
			CoreSettings: CoreSettings{EnableGRPCProxy: true},
		},
	}

	fakeTime := time.Now().Add(-time.Hour)
	e.uptime = fakeTime

	StartRPCServer(e)

	// Give the proxy time to start
	time.Sleep(time.Millisecond * 500)

	certFile := filepath.Join(tempDirTLS, "cert.pem")
	caCert, err := os.ReadFile(certFile)
	require.NoError(t, err, "ReadFile must not error")
	caCertPool := x509.NewCertPool()
	ok := caCertPool.AppendCertsFromPEM(caCert)
	require.True(t, ok, "AppendCertsFromPEM must return true")
	client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: caCertPool, MinVersion: tls.VersionTLS12}}}

	for _, creds := range []struct {
		testDescription string
		username        string
		password        string
	}{
		{"Valid credentials", "bobmarley", "Sup3rdup3rS3cr3t"},
		{"Valid username but invalid password", "bobmarley", "wrongpass"},
		{"Invalid username but valid password", "bonk", "Sup3rdup3rS3cr3t"},
		{"Invalid username and password despite glorious credentials", "bonk", "wif"},
	} {
		t.Run(creds.testDescription, func(t *testing.T) {
			t.Parallel()

			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://localhost:"+strconv.Itoa(gRPCProxyPort)+"/v1/getinfo", http.NoBody)
			require.NoError(t, err, "NewRequestWithContext must not error")
			req.SetBasicAuth(creds.username, creds.password)
			resp, err := client.Do(req)
			require.NoError(t, err, "Do must not error")
			defer resp.Body.Close()

			if creds.username == "bobmarley" && creds.password == "Sup3rdup3rS3cr3t" {
				var info gctrpc.GetInfoResponse
				err = json.NewDecoder(resp.Body).Decode(&info)
				require.NoError(t, err, "Decode must not error")

				uptimeDuration, err := time.ParseDuration(info.Uptime)
				require.NoError(t, err, "ParseDuration must not error")
				assert.InDelta(t, time.Since(fakeTime).Seconds(), uptimeDuration.Seconds(), 1.0, "Uptime should be within 1 second of the expected duration")
			} else {
				respBody, err := io.ReadAll(resp.Body)
				require.NoError(t, err, "ReadAll must not error")
				assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "HTTP status code should be 401")
				assert.Equal(t, "Access denied\n", string(respBody), "Response body should be 'Access denied\n'")
			}
		})
	}
}

func TestRPCProxyAuthClient(t *testing.T) {
	t.Parallel()

	s := new(RPCServer)
	s.Engine = &Engine{
		Config: &config.Config{
			RemoteControl: config.RemoteControlConfig{
				Username: "bobmarley",
				Password: "Sup3rdup3rS3cr3t",
			},
		},
	}

	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("MEOW"))
		assert.NoError(t, err, "Write should not error")
	})

	handler := s.authClient(dummyHandler)

	for _, creds := range []struct {
		testDescription string
		username        string
		password        string
	}{
		{"Valid credentials", "bobmarley", "Sup3rdup3rS3cr3t"},
		{"Valid username but invalid password", "bobmarley", "wrongpass"},
		{"Invalid username but valid password", "bonk", "Sup3rdup3rS3cr3t"},
		{"Invalid username and password despite glorious credentials", "bonk", "wif"},
	} {
		t.Run(creds.testDescription, func(t *testing.T) {
			t.Parallel()

			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody)
			require.NoError(t, err, "NewRequestWithContext must not error")
			req.SetBasicAuth(creds.username, creds.password)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if creds.username == "bobmarley" && creds.password == "Sup3rdup3rS3cr3t" {
				assert.Equal(t, http.StatusOK, rr.Code, "HTTP status code should be 200")
				assert.Equal(t, "MEOW", rr.Body.String(), "Response body should be 'MEOW'")
			} else {
				assert.Equal(t, http.StatusUnauthorized, rr.Code, "HTTP status code should be 401")
				assert.Equal(t, "Access denied\n", rr.Body.String(), "Response body should be 'Access denied\n'")
			}
		})
	}
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	em := NewExchangeManager()
	exch, err := em.NewExchangeByName("binance")
	require.NoError(t, err)

	exch.SetDefaults()
	b := exch.GetBase()
	b.Name = fakeExchangeName
	b.Enabled = true
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	err = b.CurrencyPairs.Store(asset.Spot, &currency.PairStore{
		AssetEnabled:  true,
		Enabled:       []currency.Pair{currency.NewBTCUSDT()},
		Available:     []currency.Pair{currency.NewBTCUSDT()},
		RequestFormat: &currency.PairFormat{Uppercase: true},
		ConfigFormat:  &currency.PairFormat{Uppercase: true},
	})
	require.NoError(t, err)

	fakeExchange := fExchange{
		IBotExchange: exch,
	}
	err = em.Add(fakeExchange)
	require.NoError(t, err)

	s := RPCServer{Engine: &Engine{ExchangeManager: em}}
	_, err = s.GetCurrencyTradeURL(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)

	req := &gctrpc.GetCurrencyTradeURLRequest{}
	_, err = s.GetCurrencyTradeURL(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrExchangeNameNotSet)

	req.Exchange = fakeExchangeName
	_, err = s.GetCurrencyTradeURL(t.Context(), req)
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	req.Asset = "spot"
	_, err = s.GetCurrencyTradeURL(t.Context(), req)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	req.Pair = &gctrpc.CurrencyPair{
		Delimiter: "-",
		Base:      "btc",
		Quote:     "usdt",
	}
	resp, err := s.GetCurrencyTradeURL(t.Context(), req)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Url)
}
