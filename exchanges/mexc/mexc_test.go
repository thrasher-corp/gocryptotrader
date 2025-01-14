package mexc

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

// Please supply your own keys here to do authenticated endpoint testing
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var me = &MEXC{}

func TestMain(m *testing.M) {
	me.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal(err)
	}

	exchCfg, err := cfg.GetExchangeConfig("Mexc")
	if err != nil {
		log.Fatal(err)
	}

	exchCfg.API.AuthenticatedSupport = true
	exchCfg.API.AuthenticatedWebsocketSupport = true
	exchCfg.API.Credentials.Key = apiKey
	exchCfg.API.Credentials.Secret = apiSecret

	err = me.Setup(exchCfg)
	if err != nil {
		log.Fatal(err)
	}

	os.Exit(m.Run())
}

// Ensures that this exchange package is compatible with IBotExchange
// func TestInterface(t *testing.T) {
// 	var e exchange.IBotExchange
// 	if e = new(MEXC); e == nil {
// 		t.Fatal("unable to allocate exchange")
// 	}
// }

// Implement tests for API endpoints below

func TestGetSymbols(t *testing.T) {
	t.Parallel()
	result, err := me.GetSymbols(context.Background(), []string{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSystemTime(t *testing.T) {
	t.Parallel()
	result, err := me.GetSystemTime(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDefaultSumbols(t *testing.T) {
	t.Parallel()
	result, err := me.GetDefaultSumbols(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := me.GetOrderbook(context.Background(), "", 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := me.GetOrderbook(context.Background(), "BTCUSDT", 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetRecentTradesList(t *testing.T) {
	t.Parallel()
	_, err := me.GetRecentTradesList(context.Background(), "", 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := me.GetRecentTradesList(context.Background(), "BTCUSDT", 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAggregatedTrades(t *testing.T) {
	t.Parallel()
	_, err := me.GetAggregatedTrades(context.Background(), "", time.Now().Add(-time.Hour*1), time.Now(), 0)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := me.GetAggregatedTrades(context.Background(), "BTCUSDT", time.Now().Add(-time.Hour*1), time.Now(), 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCandlestick(t *testing.T) {
	t.Parallel()
	_, err := me.GetCandlestick(context.Background(), "", kline.FiveMin, time.Time{}, time.Time{}, 0)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = me.GetCandlestick(context.Background(), "BTCUSDT", kline.TwelveHour, time.Time{}, time.Time{}, 0)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval)

	result, err := me.GetCandlestick(context.Background(), "BTCUSDT", kline.FiveMin, time.Time{}, time.Time{}, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrentAveragePrice(t *testing.T) {
	t.Parallel()
	_, err := me.GetCurrentAveragePrice(context.Background(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	me.Verbose = true
	result, err := me.GetCurrentAveragePrice(context.Background(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGet24HourTickerPriceChangeStatistics(t *testing.T) {
	t.Parallel()
	result, err := me.Get24HourTickerPriceChangeStatistics(context.Background(), []string{})
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = me.Get24HourTickerPriceChangeStatistics(context.Background(), []string{"BTCUSDT"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSymbolPriceTicker(t *testing.T) {
	t.Parallel()
	me.Verbose = true
	result, err := me.GetSymbolPriceTicker(context.Background(), []string{})
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = me.GetSymbolPriceTicker(context.Background(), []string{"BTCUSDT"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSymbolOrderbookTicker(t *testing.T) {
	t.Parallel()
	result, err := me.GetSymbolOrderbookTicker(context.Background(), "")
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = me.GetSymbolOrderbookTicker(context.Background(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateSubAccount(t *testing.T) {
	t.Parallel()
	_, err := me.CreateSubAccount(context.Background(), "", "sub-account notes")
	require.ErrorIs(t, err, errInvalidSubAccountName)
	_, err = me.CreateSubAccount(context.Background(), "Test1", "")
	require.ErrorIs(t, err, errInvalidSubAccountNote)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me, canManipulateRealOrders)
	result, err := me.CreateSubAccount(context.Background(), "Test1", "sub-account notes")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetSubAccountList(context.Background(), "SubAcc1", false, 1, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateAPIKeyForSubAccount(t *testing.T) {
	t.Parallel()
	_, err := me.CreateAPIKeyForSubAccount(context.Background(), "", "123", "SPOT_DEAL_WRITE", "")
	require.ErrorIs(t, err, errInvalidSubAccountName)
	_, err = me.CreateAPIKeyForSubAccount(context.Background(), "SubAcc2", "", "SPOT_DEAL_WRITE", "")
	require.ErrorIs(t, err, errInvalidSubAccountNote)
	_, err = me.CreateAPIKeyForSubAccount(context.Background(), "SubAcc2", "123", "", "")
	require.ErrorIs(t, err, errUnsupportedPermissionValue)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me, canManipulateRealOrders)
	result, err := me.CreateAPIKeyForSubAccount(context.Background(), "SubAcc2", "123", "SPOT_DEAL_WRITE", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	_, err := me.GetSubAccountAPIKey(context.Background(), "")
	require.ErrorIs(t, err, errInvalidSubAccountName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetSubAccountAPIKey(context.Background(), "SubAcc1")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestDeleteAPIKeySubAccount(t *testing.T) {
	t.Parallel()
	_, err := me.DeleteAPIKeySubAccount(context.Background(), "")
	require.ErrorIs(t, err, errInvalidSubAccountName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me, canManipulateRealOrders)
	result, err := me.DeleteAPIKeySubAccount(context.Background(), "SubAcc1")
	require.NoError(t, err)
	assert.NotNil(t, result)
}
