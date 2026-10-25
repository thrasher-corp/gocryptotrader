package mexc

import (
	"context"
	"encoding/hex"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deposit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

// Please supply your own keys here to do authenticated endpoint testing
const (
	apiKey                    = ""
	apiSecret                 = ""
	canManipulateRealOrders   = false
	canManipulateAPIEndpoints = false
)

var (
	e *Exchange

	assetsAndErrors = map[asset.Item]error{
		asset.Spot:    nil,
		asset.Futures: nil,
		asset.Options: asset.ErrNotSupported,
	}

	spotTradablePair currency.Pair

	futuresTradablePair = currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter)
	futuresInversePair  = currency.NewPairWithDelimiter("BTC", "USD", currency.UnderscoreDelimiter)
)

// recentWindow returns a query window that stays inside the history endpoints' limits. Several of
// them accept only recent data - "Only 7 day's data can be queried", "The start time must be within
// 90 days from now", "Exceeded the queryable time range" - so a window pinned to a fixed date works
// when it is written and fails once that date has aged out.
// It stays short as well as recent, because the same window feeds the candle and trade endpoints,
// where a multi-day span would ask for more bars than a single response carries.
//
// Under the mock the window is the recorded one instead: mock responses are matched on the full
// request URL, so a moving window matches nothing.
func recentWindow() (start, end time.Time) {
	if mockTests {
		return time.UnixMilli(1767204283384), time.UnixMilli(1767204403384)
	}
	now := time.Now()
	return now.Add(-2 * time.Hour), now.Add(-time.Hour)
}

// skipIfAccountCannot skips when the exchange refused because of what the account is rather than
// what the code sent. Broker endpoints need a broker account, sub-account endpoints need
// sub-accounts to exist, and a lookup by a made-up id finds no record - none of which a plain spot
// key can satisfy. Skipping names the reason; passing would claim a coverage that never happened.
func skipIfAccountCannot(tb testing.TB, err error) {
	tb.Helper()
	if err == nil {
		return
	}
	for _, refusal := range []string{
		"No permission to access the endpoint",
		"subAccount not exist",
		"Record does not exist",
	} {
		if strings.Contains(err.Error(), refusal) {
			tb.Skipf("this account cannot exercise the endpoint: %s", refusal)
		}
	}
}

func (e *Exchange) setEnabledPairs(spotTradablePair currency.Pair) error {
	if err := e.CurrencyPairs.StorePairs(asset.Spot, []currency.Pair{spotTradablePair}, false); err != nil {
		return err
	}
	if err := e.CurrencyPairs.StorePairs(asset.Spot, []currency.Pair{spotTradablePair}, true); err != nil {
		return err
	}
	futuresPairs := currency.Pairs{futuresTradablePair, futuresInversePair}
	if err := e.CurrencyPairs.StorePairs(asset.Futures, futuresPairs, false); err != nil {
		return err
	}
	return e.CurrencyPairs.StorePairs(asset.Futures, futuresPairs, true)
}

// newIsolatedTestExchange returns an exchange named for the test, serving the recorded responses under the mock. The
// tickers and books it stores are keyed by its name, so a test reading them back cannot pass on what another test
// stored
func newIsolatedTestExchange(t *testing.T) *Exchange {
	t.Helper()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	if mockTests {
		require.NoError(t, testexch.MockHTTPInstance(ex), "MockHTTPInstance must not error")
	}
	require.NoError(t, ex.setEnabledPairs(spotTradablePair), "setEnabledPairs must not error")
	return ex
}

func TestGetSymbols(t *testing.T) {
	t.Parallel()
	_, err := e.GetSymbols(t.Context(), nil)
	assert.NoError(t, err)

	result, err := e.GetSymbols(t.Context(), []string{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSystemTime(t *testing.T) {
	t.Parallel()
	result, err := e.GetSystemTime(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDefaultSymbols(t *testing.T) {
	t.Parallel()
	result, err := e.GetDefaultSymbols(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderbook(t.Context(), currency.EMPTYPAIR, 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetOrderbook(t.Context(), spotTradablePair, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetRecentTradesList(t *testing.T) {
	t.Parallel()
	_, err := e.GetRecentTradesList(t.Context(), currency.EMPTYPAIR, 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetRecentTradesList(t.Context(), spotTradablePair, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAggregatedTrades(t *testing.T) {
	t.Parallel()
	startTime, endTime := recentWindow()
	_, err := e.GetAggregatedTrades(t.Context(), currency.EMPTYPAIR, endTime, startTime, 0)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = e.GetAggregatedTrades(t.Context(), spotTradablePair, endTime, startTime, 0)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)
	_, err = e.GetAggregatedTrades(t.Context(), spotTradablePair, startTime, time.Time{}, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset, "a start time without an end time should be rejected")
	_, err = e.GetAggregatedTrades(t.Context(), spotTradablePair, time.Time{}, endTime, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset, "an end time without a start time should be rejected")

	result, err := e.GetAggregatedTrades(t.Context(), spotTradablePair, startTime, endTime, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestIntervalToString(t *testing.T) {
	t.Parallel()
	_, err := intervalToString(kline.TenMin)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval)

	intervalString, err := intervalToString(kline.FiveMin)
	require.NoError(t, err)
	assert.NotEmpty(t, intervalString)
}

func TestIntervalFromString(t *testing.T) {
	t.Parallel()
	_, err := IntervalFromString("invalid")
	require.ErrorIs(t, err, kline.ErrInvalidInterval)

	interval, err := IntervalFromString("Min5")
	require.NoError(t, err)
	assert.Equal(t, kline.FiveMin, interval)

	interval, err = IntervalFromString("1d")
	require.NoError(t, err)
	assert.Equal(t, kline.OneDay, interval)
}

func TestGetCandlestick(t *testing.T) {
	t.Parallel()
	intervalString, err := intervalToString(kline.FiveMin)
	require.NoError(t, err)

	startTime, endTime := recentWindow()
	_, err = e.GetCandlestick(t.Context(), currency.EMPTYPAIR, intervalString, startTime, endTime, 0)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = e.GetCandlestick(t.Context(), spotTradablePair, "", startTime, endTime, 0)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval)

	result, err := e.GetCandlestick(t.Context(), spotTradablePair, "5m", startTime, endTime, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

// TestCandlesFromCandlestick asserts a candle is stamped with its open time, not its close time. The
// exchange sends [openTime, o, h, l, c, volume, closeTime, quoteVolume]; stamping the candle with the
// close time shifted every candle forward by one interval.
func TestCandlesFromCandlestick(t *testing.T) {
	t.Parallel()
	var c CandlestickData
	require.NoError(t, json.Unmarshal([]byte(`[1767056100000,"87231.15","87269.54","87231.12","87250.54","17.48934453",1767056400000,"1525897.53"]`), &c))
	got := candlesFromCandlestick([]*CandlestickData{&c})
	require.Len(t, got, 1, "one candle must be produced")
	assert.Equal(t, time.UnixMilli(1767056100000), got[0].Time, "candle time should be the open time")
	assert.NotEqual(t, time.UnixMilli(1767056400000), got[0].Time, "candle time should not be the close time")
	assert.Equal(t, 87231.15, got[0].Open, "Open should be correct")
	assert.Equal(t, 87250.54, got[0].Close, "Close should be correct")
	assert.Equal(t, 87269.54, got[0].High, "High should be correct")
	assert.Equal(t, 87231.12, got[0].Low, "Low should be correct")
	assert.Equal(t, 17.48934453, got[0].Volume, "Volume should be the base volume")
}

// TestUpdateTickersResolvesConcatenatedSymbol asserts the 24h ticker symbol is resolved against the
// known pairs rather than split naively. METALUSDT is present in the recorded 24h response and splits
// to MET/ALUSDT under a naive parser; with the pair catalogued it must resolve to METAL/USDT.
func TestUpdateTickersResolvesConcatenatedSymbol(t *testing.T) {
	metal := currency.NewPair(currency.NewCode("METAL"), currency.USDT)
	origAvailable, err := e.GetAvailablePairs(asset.Spot)
	require.NoError(t, err, "GetAvailablePairs must not error")
	origEnabled, err := e.GetEnabledPairs(asset.Spot)
	require.NoError(t, err, "GetEnabledPairs must not error")
	t.Cleanup(func() {
		require.NoError(t, e.CurrencyPairs.StorePairs(asset.Spot, origAvailable, false), "restoring available pairs must not error")
		require.NoError(t, e.CurrencyPairs.StorePairs(asset.Spot, origEnabled, true), "restoring enabled pairs must not error")
	})
	available := origAvailable
	if !available.Contains(metal, true) {
		available = append(available, metal)
	}
	require.NoError(t, e.CurrencyPairs.StorePairs(asset.Spot, available, false), "StorePairs must not error")

	require.NoError(t, e.UpdateTickers(t.Context(), asset.Spot), "UpdateTickers must not error")

	tick, err := ticker.GetTicker(e.Name, metal, asset.Spot)
	require.NoError(t, err, "METALUSDT must resolve to METAL/USDT, not a mis-split")
	assert.Equal(t, currency.NewCode("METAL"), tick.Pair.Base, "base should be METAL")
	assert.Equal(t, currency.USDT, tick.Pair.Quote, "quote should be USDT")
}

func TestGetCurrentAveragePrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetCurrentAveragePrice(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetCurrentAveragePrice(t.Context(), spotTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGet24HourTickerPriceChangeStatistics(t *testing.T) {
	t.Parallel()
	result, err := e.Get24HourTickerPriceChangeStatistics(t.Context(), []string{})
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.Get24HourTickerPriceChangeStatistics(t.Context(), []string{spotTradablePair.String()})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSymbolPriceTicker(t *testing.T) {
	t.Parallel()
	_, err := e.GetSymbolPriceTicker(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	result, err := e.GetSymbolPriceTicker(t.Context(), spotTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSymbolsPriceTicker(t *testing.T) {
	result, err := e.GetSymbolsPriceTicker(t.Context(), []string{})
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetSymbolsPriceTicker(t.Context(), []string{spotTradablePair.String()})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSymbolOrderbookTicker(t *testing.T) {
	t.Parallel()
	_, err := e.GetSymbolOrderbookTicker(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetSymbolOrderbookTicker(t.Context(), spotTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderbookTickers(t *testing.T) {
	t.Parallel()
	result, err := e.GetOrderbookTickers(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateSubAccount(t *testing.T) {
	t.Parallel()
	_, err := e.CreateSubAccount(t.Context(), "", "sub-account notes")
	require.ErrorIs(t, err, errInvalidSubAccountName)

	_, err = e.CreateSubAccount(t.Context(), "Test1", "")
	require.ErrorIs(t, err, errInvalidSubAccountNote)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CreateSubAccount(t.Context(), "Test1", "sub-account notes")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSubAccountList(t.Context(), "sam", true, 1, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateAPIKeyForSubAccount(t *testing.T) {
	t.Parallel()
	_, err := e.CreateAPIKeyForSubAccount(t.Context(), "", "123", "SPOT_DEAL_WRITE", "")
	require.ErrorIs(t, err, errInvalidSubAccountName)
	_, err = e.CreateAPIKeyForSubAccount(t.Context(), "SubAcc2", "", "SPOT_DEAL_WRITE", "")
	require.ErrorIs(t, err, errInvalidSubAccountNote)
	_, err = e.CreateAPIKeyForSubAccount(t.Context(), "SubAcc2", "123", "", "")
	require.ErrorIs(t, err, errUnsupportedPermissionValue)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CreateAPIKeyForSubAccount(t.Context(), "SubAcc2", "123", "SPOT_DEAL_WRITE", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAccountAPIKey(t.Context(), "")
	require.ErrorIs(t, err, errInvalidSubAccountName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSubAccountAPIKey(t.Context(), "SubAcc1")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestDeleteAPIKeySubAccount(t *testing.T) {
	t.Parallel()
	_, err := e.DeleteAPIKeySubAccount(t.Context(), "", "key")
	require.ErrorIs(t, err, errInvalidSubAccountName)
	_, err = e.DeleteAPIKeySubAccount(t.Context(), "SubAcc1", "")
	require.ErrorIs(t, err, errAPIKeyMissing)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.DeleteAPIKeySubAccount(t.Context(), "SubAcc1", "apikey")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUniversalTransfer(t *testing.T) {
	t.Parallel()
	_, err := e.SubAccountUniversalTransfer(t.Context(), "master@test.com", "subaccount@test.com", asset.Empty, asset.Spot, currency.USDT, 1234.)
	require.ErrorIs(t, err, asset.ErrNotSupported)
	_, err = e.SubAccountUniversalTransfer(t.Context(), "master@test.com", "subaccount@test.com", asset.Spot, asset.Empty, currency.USDT, 1234.)
	require.ErrorIs(t, err, asset.ErrNotSupported)
	_, err = e.SubAccountUniversalTransfer(t.Context(), "master@test.com", "subaccount@test.com", asset.Spot, asset.Options, currency.USDT, 1234.)
	require.ErrorIs(t, err, asset.ErrNotSupported)
	_, err = e.SubAccountUniversalTransfer(t.Context(), "master@test.com", "subaccount@test.com", asset.Spot, asset.Spot, currency.EMPTYCODE, 1234.)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.SubAccountUniversalTransfer(t.Context(), "master@test.com", "subaccount@test.com", asset.Spot, asset.Spot, currency.USDT, 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SubAccountUniversalTransfer(t.Context(), "master@test.com", "subaccount@test.com", asset.Spot, asset.Spot, currency.USDT, 1234.)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountUniversalTransferHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAccountUniversalTransferHistory(t.Context(), "master@test.com", "subaccount@test.com", asset.Empty, asset.Spot, time.Now().Add(-time.Hour*50), time.Now().Add(-time.Hour*20), 10, 20)
	require.ErrorIs(t, err, asset.ErrNotSupported)
	_, err = e.GetSubAccountUniversalTransferHistory(t.Context(), "master@test.com", "subaccount@test.com", asset.Spot, asset.Empty, time.Now().Add(-time.Hour*50), time.Now().Add(-time.Hour*20), 10, 20)
	require.ErrorIs(t, err, asset.ErrNotSupported)
	_, err = e.GetSubAccountUniversalTransferHistory(t.Context(), "master@test.com", "subaccount@test.com", asset.Spot, asset.Options, time.Now().Add(-time.Hour*50), time.Now().Add(-time.Hour*20), 10, 20)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSubAccountUniversalTransferHistory(t.Context(), "master@test.com", "subaccount@test.com", asset.Spot, asset.Spot, time.Now().Add(-time.Hour*50), time.Now().Add(-time.Hour*20), 10, 20)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountAsset(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAccountAsset(t.Context(), "", asset.Spot)
	require.ErrorIs(t, err, errInvalidSubAccountName)
	_, err = e.GetSubAccountAsset(t.Context(), "thesubaccount@test.com", asset.Empty)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSubAccountAsset(t.Context(), "thesubaccount@test.com", asset.Spot)
	skipIfAccountCannot(t, err)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetKYCStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetKYCStatus(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUseAPIDefaultSymbols(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.UseAPIDefaultSymbols(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestNewTestOrder(t *testing.T) {
	t.Parallel()
	_, err := e.NewTestOrder(t.Context(), currency.EMPTYPAIR, "123123", "SELL", typeLimit, 1, 0, 123456.78)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = e.NewTestOrder(t.Context(), spotTradablePair, "123123", "", typeLimit, 1, 0, 123456.78)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	_, err = e.NewTestOrder(t.Context(), spotTradablePair, "123123", "SELL", "", 1, 0, 123456.78)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)
	_, err = e.NewTestOrder(t.Context(), spotTradablePair, "123123", "SELL", typeLimit, 0, 0, 123456.78)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)
	_, err = e.NewTestOrder(t.Context(), spotTradablePair, "123123", "SELL", typeLimit, 1, 0, 0)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin)
	_, err = e.NewTestOrder(t.Context(), spotTradablePair, "123123", "SELL", typeMarket, 0, 0, 123456.78)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.NewTestOrder(t.Context(), spotTradablePair, "123123", "SELL", typeLimit, 1, 0, 123456.78)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

// TestOrderTypeStringUnsupportedMessage asserts the unsupported-mapping error leads with the order
// type, which is the field that has no MEXC mapping, rather than the usually-empty time-in-force,
// while still wrapping both sentinels so errors.Is resolves either.
func TestOrderTypeStringUnsupportedMessage(t *testing.T) {
	t.Parallel()
	_, err := e.OrderTypeStringFromOrderTypeAndTimeInForce(order.StopMarket, order.UnknownTIF)
	require.Error(t, err, "an unmapped order type must return an error")
	require.ErrorIs(t, err, order.ErrUnsupportedOrderType, "the error must wrap the unsupported order type sentinel")
	require.ErrorIs(t, err, order.ErrUnsupportedTimeInForce, "the error must still wrap the time-in-force sentinel")
	assert.True(t, strings.HasPrefix(err.Error(), order.ErrUnsupportedOrderType.Error()), "the message should lead with the unsupported order type, not the time-in-force")
}

func TestStringToOrderTypeAndTimeInForce(t *testing.T) {
	t.Parallel()
	orderTypeAndTimeInForceFromOrderTypeString := []struct {
		String      string
		Type        order.Type
		TimeInForce order.TimeInForce
		Error       error
	}{
		{typeLimit, order.Limit, order.GoodTillCancel, nil},
		{typeLimitMaker, order.Limit, order.PostOnly, nil},
		{typePostOnly, order.Limit, order.PostOnly, nil},
		{typeMarket, order.Market, order.UnknownTIF, nil},
		{typeImmediateOrCancel, order.Limit, order.ImmediateOrCancel, nil},
		{typeFillOrKill, order.Limit, order.FillOrKill, nil},
		{typeStopLimit, order.StopLimit, order.UnknownTIF, nil},
		{typeStopMarketOrder, order.StopMarket, order.UnknownTIF, nil},
		{"", order.UnknownType, order.UnknownTIF, order.ErrUnsupportedOrderType},
	}
	for x := range orderTypeAndTimeInForceFromOrderTypeString {
		t.Run(orderTypeAndTimeInForceFromOrderTypeString[x].String, func(t *testing.T) {
			t.Parallel()
			oType, tif, err := e.StringToOrderTypeAndTimeInForce(orderTypeAndTimeInForceFromOrderTypeString[x].String)
			require.ErrorIs(t, err, orderTypeAndTimeInForceFromOrderTypeString[x].Error)
			assert.Equal(t, orderTypeAndTimeInForceFromOrderTypeString[x].Type, oType)
			assert.Equal(t, orderTypeAndTimeInForceFromOrderTypeString[x].TimeInForce, tif)
		})
	}
}

func TestNewOrder(t *testing.T) {
	t.Parallel()
	_, err := e.NewOrder(t.Context(), currency.EMPTYPAIR, "123123", "SELL", typeLimit, 1, 0, 123456.78)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = e.NewOrder(t.Context(), spotTradablePair, "123123", "", typeLimit, 1, 0, 123456.78)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	_, err = e.NewOrder(t.Context(), spotTradablePair, "123123", "SELL", "", 1, 0, 123456.78)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)
	_, err = e.NewOrder(t.Context(), spotTradablePair, "123123", "SELL", typeLimit, 0, 0, 123456.78)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)
	_, err = e.NewOrder(t.Context(), spotTradablePair, "123123", "SELL", typeLimit, 1, 0, 0)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin)
	_, err = e.NewOrder(t.Context(), spotTradablePair, "123123", "SELL", typeMarket, 0, 0, 123456.78)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)
	// IMMEDIATE_OR_CANCEL and FILL_OR_KILL are limit order types on MEXC: both require a price.
	_, err = e.NewOrder(t.Context(), spotTradablePair, "123123", "SELL", typeImmediateOrCancel, 1, 0, 0)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin)
	_, err = e.NewOrder(t.Context(), spotTradablePair, "123123", "SELL", typeFillOrKill, 1, 0, 0)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin)
	_, err = e.NewOrder(t.Context(), spotTradablePair, "123123", "SELL", typeFillOrKill, 0, 0, 123456.78)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.NewOrder(t.Context(), spotTradablePair, "123123", "BUY", typeLimit, 1, 0, 123456.78)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateBatchOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CreateBatchOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = e.CreateBatchOrder(t.Context(), make([]BatchOrderCreationParam, 0, 1))
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = e.CreateBatchOrder(t.Context(), []BatchOrderCreationParam{{}})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := BatchOrderCreationParam{
		NewClientOrderID: "1234",
	}
	_, err = e.CreateBatchOrder(t.Context(), []BatchOrderCreationParam{arg})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	arg.Symbol = spotTradablePair
	_, err = e.CreateBatchOrder(t.Context(), []BatchOrderCreationParam{arg})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = order.Sell.String()
	_, err = e.CreateBatchOrder(t.Context(), []BatchOrderCreationParam{arg})
	require.ErrorIs(t, err, order.ErrUnsupportedOrderType)

	arg.OrderType, err = e.OrderTypeStringFromOrderTypeAndTimeInForce(order.Limit, order.UnknownTIF)
	require.NoError(t, err)
	_, err = e.CreateBatchOrder(t.Context(), []BatchOrderCreationParam{arg})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	arg.Quantity = 123478.5
	_, err = e.CreateBatchOrder(t.Context(), []BatchOrderCreationParam{arg})
	require.ErrorIs(t, err, limits.ErrPriceBelowMin)

	arg.OrderType, err = e.OrderTypeStringFromOrderTypeAndTimeInForce(order.Market, order.UnknownTIF)
	require.NoError(t, err)

	arg.Quantity = 0
	_, err = e.CreateBatchOrder(t.Context(), []BatchOrderCreationParam{arg})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	arg.Quantity = 1231231
	result, err := e.CreateBatchOrder(t.Context(), []BatchOrderCreationParam{arg})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestOrderTypeString(t *testing.T) {
	t.Parallel()
	typesMap := map[struct {
		Type        order.Type
		TimeInForce order.TimeInForce
	}]struct {
		String string
		Error  error
	}{
		{Type: order.Limit}:                    {String: typeLimit},
		{TimeInForce: order.PostOnly}:          {String: typeLimitMaker},
		{Type: order.Market}:                   {String: typeMarket},
		{TimeInForce: order.ImmediateOrCancel}: {String: typeImmediateOrCancel},
		{TimeInForce: order.FillOrKill}:        {String: typeFillOrKill},
		{Type: order.StopLimit}:                {String: typeStopLimit},
		{Type: order.MarketMakerProtection}:    {String: "", Error: order.ErrUnsupportedOrderType},
	}
	for a := range typesMap {
		t.Run(typesMap[a].String, func(t *testing.T) {
			t.Parallel()
			value, err := e.OrderTypeStringFromOrderTypeAndTimeInForce(a.Type, a.TimeInForce)
			require.Equal(t, typesMap[a].String, value)
			require.ErrorIs(t, err, typesMap[a].Error)
		})
	}
}

func TestCancelTradeOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelTradeOrder(t.Context(), currency.EMPTYPAIR, "", "", "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = e.CancelTradeOrder(t.Context(), spotTradablePair, "", "", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelTradeOrder(t.Context(), spotTradablePair, "1234", "", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllOpenOrdersBySymbol(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllOpenOrdersBySymbol(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelAllOpenOrdersBySymbol(t.Context(), spotTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderByID(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderByID(t.Context(), currency.EMPTYPAIR, "123455", "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = e.GetOrderByID(t.Context(), spotTradablePair, "", "")
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	// Order "1234" belongs to nobody, and the tests may not create one (canManipulateRealOrders),
	// so "Order does not exist" is the expected answer from any account. What is asserted is that
	// the request reaches the exchange and comes back with that answer rather than anything else.
	result, err := e.GetOrderByID(t.Context(), spotTradablePair, "1234", "")
	if err != nil {
		assert.ErrorContains(t, err, "Order does not exist", "the only acceptable failure is the order being absent")
		return
	}
	assert.NotNil(t, result)
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetOpenOrders(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOpenOrders(t.Context(), spotTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllOrders(t.Context(), currency.EMPTYPAIR, time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	// Relative to now: the endpoint answers "Only 7 day's data can be queried", so a window pinned
	// to a fixed date falls outside the queryable range as soon as that date is a week old.
	startTime, endTime := recentWindow()
	_, err = e.GetAllOrders(t.Context(), spotTradablePair, endTime, startTime, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAllOrders(t.Context(), spotTradablePair, startTime, endTime, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountInformation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAccountInformation(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountTradeList(t *testing.T) {
	t.Parallel()
	_, err := e.GetAccountTradeList(t.Context(), currency.EMPTYPAIR, "", time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	// Relative to now: the endpoint answers "Exceeded the queryable time range" for a window that
	// has aged out, which a fixed date does within a week of being written.
	startTime, endTime := recentWindow()
	_, err = e.GetAccountTradeList(t.Context(), spotTradablePair, "", endTime, startTime, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAccountTradeList(t.Context(), spotTradablePair, "", startTime, endTime, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestEnableMXDeduct(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.EnableMXDeduct(t.Context(), true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMXDeductStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetMXDeductStatus(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSymbolTradingFee(t *testing.T) {
	t.Parallel()
	_, err := e.GetSymbolTradingFee(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSymbolTradingFee(t.Context(), spotTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrencyInformation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetCurrencyInformation(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdrawCapital(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawCapital(t.Context(), 1.2, currency.EMPTYCODE, "", "BNB", "1234", core.BitcoinDonationAddress, "abcd", "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.WithdrawCapital(t.Context(), 1.2, currency.BTC, "", "BNB", "", "", "abcd", "")
	require.ErrorIs(t, err, errAddressRequired)
	_, err = e.WithdrawCapital(t.Context(), 0, currency.BTC, "", "BNB", "1234", core.BitcoinDonationAddress, "abcd", "")
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.WithdrawCapital(t.Context(), 1.2, currency.BTC, "", "BNB", "1234", core.BitcoinDonationAddress, "abcd", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelWithdrawal(t *testing.T) {
	t.Parallel()
	_, err := e.CancelWithdrawal(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelWithdrawal(t.Context(), "1231212")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFundDepositHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := recentWindow()
	_, err := e.GetFundDepositHistory(t.Context(), currency.BTC, "", endTime, startTime, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFundDepositHistory(t.Context(), currency.BTC, "", startTime, endTime, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWithdrawalHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := recentWindow()
	_, err := e.GetWithdrawalHistory(t.Context(), currency.USDT, endTime, startTime, 0, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetWithdrawalHistory(t.Context(), currency.USDT, startTime, endTime, 1, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGenerateDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.GenerateDepositAddress(t.Context(), currency.EMPTYCODE, "TRC20")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.GenerateDepositAddress(t.Context(), currency.USDT, "")
	require.ErrorIs(t, err, errNetworkNameRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.GenerateDepositAddress(t.Context(), currency.USDT, "TRC20")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDepositAddressOfCoin(t *testing.T) {
	t.Parallel()
	_, err := e.GetDepositAddressOfCoin(t.Context(), currency.EMPTYCODE, "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetDepositAddressOfCoin(t.Context(), currency.USDT, "ERC20")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWithdrawalAddress(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetWithdrawalAddress(t.Context(), currency.USDT, 1, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUserUniversalTransfer(t *testing.T) {
	t.Parallel()
	_, err := e.UserUniversalTransfer(t.Context(), asset.Empty, asset.Spot, currency.USDT, 1000)
	require.ErrorIs(t, err, asset.ErrInvalidAsset)
	_, err = e.UserUniversalTransfer(t.Context(), asset.Futures, asset.Empty, currency.USDT, 1000)
	require.ErrorIs(t, err, asset.ErrInvalidAsset)
	_, err = e.UserUniversalTransfer(t.Context(), asset.Futures, asset.Spot, currency.EMPTYCODE, 1000)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.UserUniversalTransfer(t.Context(), asset.Futures, asset.Spot, currency.USDT, 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.UserUniversalTransfer(t.Context(), asset.Futures, asset.Spot, currency.USDT, 1000)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUniversalTransferHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := recentWindow()
	_, err := e.GetUniversalTransferHistory(t.Context(), asset.Empty, asset.Futures, startTime, endTime, 0, 10)
	require.ErrorIs(t, err, asset.ErrInvalidAsset)
	_, err = e.GetUniversalTransferHistory(t.Context(), asset.Spot, asset.Empty, startTime, endTime, 0, 10)
	require.ErrorIs(t, err, asset.ErrInvalidAsset)
	_, err = e.GetUniversalTransferHistory(t.Context(), asset.Spot, asset.Futures, endTime, startTime, 0, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUniversalTransferHistory(t.Context(), asset.Spot, asset.Futures, startTime, endTime, 1, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUniversalTransferDetailByID(t *testing.T) {
	t.Parallel()
	_, err := e.GetUniversalTransferDetailByID(t.Context(), "")
	require.ErrorIs(t, err, errTransactionIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUniversalTransferDetailByID(t.Context(), "12345678")
	skipIfAccountCannot(t, err)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAssetThatCanBeConvertedintoMX(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAssetThatCanBeConvertedintoMX(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestDustConvert(t *testing.T) {
	t.Parallel()
	_, err := e.DustConvert(t.Context(), []currency.Code{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.DustConvert(t.Context(), []currency.Code{currency.EMPTYCODE, currency.ETH})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.DustConvert(t.Context(), make([]currency.Code, dustConvertMaxAssets+1))
	require.ErrorIs(t, err, errTooManyDustAssets)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.DustConvert(t.Context(), []currency.Code{currency.BTC, currency.ETH})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestDustLog(t *testing.T) {
	t.Parallel()
	startTime, endTime := recentWindow()
	_, err := e.DustLog(t.Context(), endTime, startTime, 0, 0)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)
	_, err = e.DustLog(t.Context(), startTime, endTime, 0, 0)
	require.ErrorIs(t, err, errPaginationLimitIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.DustLog(t.Context(), startTime, endTime, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestInternalTransfer(t *testing.T) {
	t.Parallel()
	_, err := e.InternalTransfer(t.Context(), "", "someone@example.com", "+251", currency.USDT, 1.2)
	require.ErrorIs(t, err, errAccountTypeRequired)
	_, err = e.InternalTransfer(t.Context(), "EMAIL", "", "+251", currency.USDT, 1.2)
	require.ErrorIs(t, err, errAddressRequired)
	_, err = e.InternalTransfer(t.Context(), "EMAIL", "someone@example.com", "+251", currency.EMPTYCODE, 1.2)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.InternalTransfer(t.Context(), "EMAIL", "someone@example.com", "+251", currency.USDT, 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.InternalTransfer(t.Context(), "EMAIL", "someone@example.com", "+251", currency.USDT, 1.2)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetInternalTransferHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := recentWindow()
	_, err := e.GetInternalTransferHistory(t.Context(), "11945860693", endTime, startTime, 0, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetInternalTransferHistory(t.Context(), "11945860693", startTime, endTime, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetRebateHistoryRecords(t *testing.T) {
	t.Parallel()
	startTime, endTime := recentWindow()
	_, err := e.GetRebateHistoryRecords(t.Context(), endTime, startTime, 100)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetRebateHistoryRecords(t.Context(), startTime, endTime, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetRebateRecordsDetail(t *testing.T) {
	t.Parallel()
	startTime, endTime := recentWindow()
	_, err := e.GetRebateRecordsDetail(t.Context(), endTime, startTime, 1000)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetRebateRecordsDetail(t.Context(), startTime, endTime, 1000)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSelfRebateRecordsDetail(t *testing.T) {
	t.Parallel()
	startTime, endTime := recentWindow()
	_, err := e.GetSelfRebateRecordsDetail(t.Context(), endTime, startTime, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSelfRebateRecordsDetail(t.Context(), startTime, endTime, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetReferCode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetReferCode(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAffiliateCommissionRecord(t *testing.T) {
	t.Parallel()
	startTime, endTime := recentWindow()
	_, err := e.GetAffiliateCommissionRecord(t.Context(), endTime, startTime, "abcdef", 1, 100)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAffiliateCommissionRecord(t.Context(), startTime, endTime, "abcdef", 1, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAffiliateWithdrawRecord(t *testing.T) {
	t.Parallel()
	startTime, endTime := recentWindow()
	_, err := e.GetAffiliateWithdrawRecord(t.Context(), endTime, startTime, 0, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAffiliateWithdrawRecord(t.Context(), startTime, endTime, 1, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAffiliateCommissionDetailRecord(t *testing.T) {
	t.Parallel()
	startTime, endTime := recentWindow()
	_, err := e.GetAffiliateCommissionDetailRecord(t.Context(), endTime, startTime, "", "1", 0, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAffiliateCommissionDetailRecord(t.Context(), startTime, endTime, "", "1", 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAffiliateCampaignData(t *testing.T) {
	t.Parallel()
	startTime, endTime := recentWindow()
	_, err := e.GetAffiliateCampaignData(t.Context(), endTime, startTime, 0, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAffiliateCampaignData(t.Context(), startTime, endTime, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAffiliateReferralData(t *testing.T) {
	t.Parallel()
	startTime, endTime := recentWindow()
	_, err := e.GetAffiliateReferralData(t.Context(), endTime, startTime, "", "", 1, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAffiliateReferralData(t.Context(), startTime, endTime, "350882", "abc", 1, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAffiliateData(t *testing.T) {
	t.Parallel()
	startTime, endTime := recentWindow()
	_, err := e.GetSubAffiliateData(t.Context(), endTime, startTime, "", 1, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSubAffiliateData(t.Context(), startTime, endTime, "", 1, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBrokerUniversalTransferHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetBrokerUniversalTransferHistory(t.Context(), asset.Empty, asset.Empty, "test1@thrasher.io", "test2@thrasher.io", time.Time{}, time.Time{}, 0, 10)
	require.ErrorIs(t, err, asset.ErrInvalidAsset)
	_, err = e.GetBrokerUniversalTransferHistory(t.Context(), asset.Empty, asset.Empty, "test1@thrasher.io", "test2@thrasher.io", time.Time{}, time.Time{}, 0, 10)
	require.ErrorIs(t, err, asset.ErrInvalidAsset)

	startTime, endTime := recentWindow()
	_, err = e.GetBrokerUniversalTransferHistory(t.Context(), asset.Futures, asset.Empty, "test1@thrasher.io", "test2@thrasher.io", startTime, endTime, 0, 10)
	require.ErrorIs(t, err, asset.ErrInvalidAsset)
	_, err = e.GetBrokerUniversalTransferHistory(t.Context(), asset.Futures, asset.Spot, "test1@thrasher.io", "test2@thrasher.io", endTime, startTime, 1, 100)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetBrokerUniversalTransferHistory(t.Context(), asset.Spot, asset.Futures, "test1@thrasher.io", "test2@thrasher.io", startTime, endTime, 1, 10)
	skipIfAccountCannot(t, err)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateBrokerSubAccount(t *testing.T) {
	t.Parallel()
	_, err := e.CreateBrokerSubAccount(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "a nil request must be rejected")
	_, err = e.CreateBrokerSubAccount(t.Context(), &BrokerSubAccountCreationParams{Note: "note"})
	require.ErrorIs(t, err, errInvalidSubAccountName, "a missing sub-account name must be rejected")
	_, err = e.CreateBrokerSubAccount(t.Context(), &BrokerSubAccountCreationParams{SubAccount: "sub1"})
	require.ErrorIs(t, err, errInvalidSubAccountNote, "a missing note must be rejected")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CreateBrokerSubAccount(t.Context(), &BrokerSubAccountCreationParams{SubAccount: "sub1", Note: "created by gct integration test"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBrokerAccountSubAccountList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetBrokerAccountSubAccountList(t.Context(), "my-subaccount-name", 10, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountStatus(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAccountStatus(t.Context(), "")
	require.ErrorIs(t, err, errInvalidSubAccountName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSubAccountStatus(t.Context(), "my-subaccount-name")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateBrokerSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	_, err := e.CreateBrokerSubAccountAPIKey(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	arg := &BrokerSubAccountAPIKeyParams{
		IP: []string{"127.0.0.1"},
	}
	_, err = e.CreateBrokerSubAccountAPIKey(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidSubAccountName)

	arg.SubAccount = "my-subaccount-name"
	_, err = e.CreateBrokerSubAccountAPIKey(t.Context(), arg)
	require.ErrorIs(t, err, errUnsupportedPermissionValue)

	arg.Permissions = []string{"SPOT_ACCOUNT_READ", "SPOT_ACCOUNT_WRITE"}
	_, err = e.CreateBrokerSubAccountAPIKey(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidSubAccountNote)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateAPIEndpoints)
	result, err := e.CreateBrokerSubAccountAPIKey(t.Context(), &BrokerSubAccountAPIKeyParams{
		SubAccount:  "my-subaccount-name",
		Permissions: []string{"SPOT_ACCOUNT_READ", "SPOT_ACCOUNT_WRITE"},
		Note:        "note-here",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBrokerSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	_, err := e.GetBrokerSubAccountAPIKey(t.Context(), "")
	require.ErrorIs(t, err, errInvalidSubAccountName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetBrokerSubAccountAPIKey(t.Context(), "my-subaccount-name")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMarshalStringList(t *testing.T) {
	t.Parallel()
	data := &struct {
		Data StringList `json:"data"`
	}{
		Data: []string{"SPOT_ACCOUNT_READ", "SPOT_ACCOUNT_WRITE"},
	}
	result, err := json.Marshal(data)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.JSONEq(t, `{"data":"SPOT_ACCOUNT_READ,SPOT_ACCOUNT_WRITE"}`, string(result))
}

func TestDeleteBrokerAPIKeySubAccount(t *testing.T) {
	t.Parallel()
	_, err := e.DeleteBrokerAPIKeySubAccount(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.DeleteBrokerAPIKeySubAccount(t.Context(), &BrokerSubAccountAPIKeyDeletionParams{APIKey: "api-key-here"})
	require.ErrorIs(t, err, errInvalidSubAccountName)
	_, err = e.DeleteBrokerAPIKeySubAccount(t.Context(), &BrokerSubAccountAPIKeyDeletionParams{SubAccount: "sub-account-detail-here"})
	require.ErrorIs(t, err, errAPIKeyMissing)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateAPIEndpoints)
	result, err := e.DeleteBrokerAPIKeySubAccount(t.Context(), &BrokerSubAccountAPIKeyDeletionParams{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGenerateBrokerSubAccountDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.GenerateBrokerSubAccountDepositAddress(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GenerateBrokerSubAccountDepositAddress(t.Context(), &BrokerSubAccountDepositAddressCreationParams{Network: "ERC20"})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.GenerateBrokerSubAccountDepositAddress(t.Context(), &BrokerSubAccountDepositAddressCreationParams{Coin: currency.ETH})
	require.ErrorIs(t, err, errNetworkNameRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.GenerateBrokerSubAccountDepositAddress(t.Context(), &BrokerSubAccountDepositAddressCreationParams{Coin: currency.ETH, Network: "ERC20"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBrokerSubAccountDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.GetBrokerSubAccountDepositAddress(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetBrokerSubAccountDepositAddress(t.Context(), currency.BTC)
	skipIfAccountCannot(t, err)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountDepositHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := recentWindow()
	_, err := e.GetSubAccountDepositHistory(t.Context(), currency.ETH, "1", endTime, startTime, 0, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSubAccountDepositHistory(t.Context(), currency.ETH, "1", startTime, endTime, 1, 10)
	skipIfAccountCannot(t, err)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllRecentSubAccountDepositHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := recentWindow()
	_, err := e.GetAllRecentSubAccountDepositHistory(t.Context(), currency.ETH, "1", endTime, startTime, 1, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAllRecentSubAccountDepositHistory(t.Context(), currency.ETH, "1", startTime, endTime, 1, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	for k, v := range assetsAndErrors {
		result, err := e.FetchTradablePairs(t.Context(), k)
		require.ErrorIs(t, err, v)
		if v == nil {
			assert.NotNil(t, result)
		}
		if k == asset.Spot && mockTests {
			require.NotEmpty(t, result, "the permitted spot symbols must remain")
			// BTCUSDT carries isSpotTradingAllowed:false in the fixture, so it must be filtered out
			// even though its status is 1 like every other symbol.
			assert.False(t, result.Contains(currency.NewBTCUSDT(), false),
				"a symbol the venue does not allow for spot trading should be excluded from the tradable pairs")
		}
		if k == asset.Futures {
			if mockTests {
				// HOOKED_USDT is listed with apiAllowed:false in the fixture: it trades on the website only
				exp := currency.Pairs{futuresTradablePair, currency.NewPairWithDelimiter("ETH", "USDT", currency.UnderscoreDelimiter), futuresInversePair}
				assert.Equal(t, exp, result, "the contracts open to API trading should be tradable")
			} else {
				assert.True(t, result.Contains(futuresTradablePair, true), "a linear contract should be tradable")
				assert.True(t, result.Contains(futuresInversePair, true), "an inverse contract should be tradable")
			}
		}
	}
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateTicker(t.Context(), spotTradablePair, asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = e.UpdateTicker(t.Context(), currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	result, err := e.UpdateTicker(t.Context(), spotTradablePair, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)

	_, err = e.UpdateTicker(t.Context(), spotTradablePair, asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	ex := newIsolatedTestExchange(t)
	// A linear contract's base volume and open interest are its contracts times their size; an inverse contract's base
	// volume is its turnover, and its quote volume and open interest are its contracts times their size
	exp := map[currency.Pair]*ticker.Price{
		futuresTradablePair: {
			Last:         80563,
			High:         82032.7,
			Low:          80409.4,
			Bid:          80563,
			Ask:          80563.1,
			BaseVolume:   36384.836500000005,
			QuoteVolume:  2955827156.60217,
			OpenInterest: 48989.418000000005,
			MarkPrice:    80564.6,
			IndexPrice:   80602.6,
			Pair:         futuresTradablePair,
			ExchangeName: ex.Name,
			AssetType:    asset.Futures,
			LastUpdated:  time.UnixMilli(1792843033884),
		},
		futuresInversePair: {
			Last:         80467.9,
			High:         81923.7,
			Low:          80331.9,
			Bid:          80467.8,
			Ask:          80467.9,
			BaseVolume:   134.1655007433945,
			QuoteVolume:  10871000,
			OpenInterest: 172493300,
			MarkPrice:    80468,
			IndexPrice:   80499.4,
			Pair:         futuresInversePair,
			ExchangeName: ex.Name,
			AssetType:    asset.Futures,
			LastUpdated:  time.UnixMilli(1792843034088),
		},
	}
	for _, p := range []currency.Pair{futuresTradablePair, futuresInversePair} {
		result, err = ex.UpdateTicker(t.Context(), p, asset.Futures)
		require.NoErrorf(t, err, "UpdateTicker must not error for %s", p)
		if mockTests {
			assert.Equalf(t, exp[p], result, "the %s ticker should be mapped", p)
			continue
		}
		assert.Truef(t, result.Pair.Equal(p), "the %s ticker should carry its pair", p)
		assert.Positivef(t, result.Last, "the %s ticker should carry the last price", p)
		assert.Positivef(t, result.BaseVolume, "the %s ticker should carry its base volume", p)
		assert.Positivef(t, result.QuoteVolume, "the %s ticker should carry its quote volume", p)
		assert.Positivef(t, result.OpenInterest, "the %s ticker should carry its open interest", p)
		assert.Positivef(t, result.MarkPrice, "the %s ticker should carry its mark price", p)
		assert.Positivef(t, result.IndexPrice, "the %s ticker should carry its index price", p)
	}
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	for k, v := range assetsAndErrors {
		err := e.UpdateTickers(t.Context(), k)
		require.ErrorIs(t, err, v)
	}
	ex := newIsolatedTestExchange(t)
	require.NoError(t, ex.UpdateTickers(t.Context(), asset.Futures), "UpdateTickers must not error for futures")
	exp := map[currency.Pair]*ticker.Price{
		futuresTradablePair: {
			Last:         80563,
			High:         82032.7,
			Low:          80409.4,
			Bid:          80563,
			Ask:          80563.1,
			BaseVolume:   36395.3187,
			QuoteVolume:  2956671651.56431,
			OpenInterest: 49021.712400000004,
			MarkPrice:    80564.1,
			IndexPrice:   80602.7,
			Pair:         futuresTradablePair,
			ExchangeName: ex.Name,
			AssetType:    asset.Futures,
			LastUpdated:  time.UnixMilli(1792843113905),
		},
		futuresInversePair: {
			Last:         80467.8,
			High:         81923.7,
			Low:          80331.9,
			Bid:          80467.8,
			Ask:          80467.9,
			BaseVolume:   134.1779280668535,
			QuoteVolume:  10872000,
			OpenInterest: 172493300,
			MarkPrice:    80467.8,
			IndexPrice:   80494,
			Pair:         futuresInversePair,
			ExchangeName: ex.Name,
			AssetType:    asset.Futures,
			LastUpdated:  time.UnixMilli(1792843114110),
		},
	}
	for _, p := range []currency.Pair{futuresTradablePair, futuresInversePair} {
		result, err := ticker.GetTicker(ex.Name, p, asset.Futures)
		require.NoErrorf(t, err, "GetTicker must not error for %s", p)
		if mockTests {
			assert.Equalf(t, exp[p], result, "the %s ticker should be stored", p)
			continue
		}
		assert.Positivef(t, result.Last, "the %s ticker should carry the last price", p)
		assert.Positivef(t, result.OpenInterest, "the %s ticker should carry its open interest", p)
	}
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateOrderbook(t.Context(), spotTradablePair, asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = e.UpdateOrderbook(t.Context(), currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.UpdateOrderbook(t.Context(), spotTradablePair, asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	result, err := e.UpdateOrderbook(t.Context(), spotTradablePair, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)

	_, err = e.UpdateOrderbook(t.Context(), spotTradablePair, asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	ex := newIsolatedTestExchange(t)
	result, err = ex.UpdateOrderbook(t.Context(), futuresTradablePair, asset.Futures)
	require.NoError(t, err, "UpdateOrderbook must not error for futures")
	if !mockTests {
		require.NotEmpty(t, result.Bids, "the futures book must have bids")
		require.NotEmpty(t, result.Asks, "the futures book must have asks")
		assert.Positive(t, result.LastUpdateID, "the futures book should carry the venue's version")
		assert.False(t, result.LastUpdated.IsZero(), "the futures book should carry the venue's time")
		return
	}
	assert.False(t, result.InsertedAt.IsZero(), "the futures book should record when it was loaded")
	result.InsertedAt = time.Time{}
	// The book's amounts are in contracts, and it carries the venue's version and time
	exp := &orderbook.Book{
		Bids: orderbook.Levels{
			{Price: 80563.2, Amount: 10563, OrderCount: 5},
			{Price: 80563.1, Amount: 306, OrderCount: 1},
			{Price: 80563, Amount: 213, OrderCount: 2},
			{Price: 80562.9, Amount: 184, OrderCount: 1},
			{Price: 80562.8, Amount: 184, OrderCount: 1},
		},
		Asks: orderbook.Levels{
			{Price: 80563.3, Amount: 13502, OrderCount: 3},
			{Price: 80563.4, Amount: 184, OrderCount: 1},
			{Price: 80563.7, Amount: 184, OrderCount: 1},
			{Price: 80564.3, Amount: 306, OrderCount: 1},
			{Price: 80564.7, Amount: 306, OrderCount: 1},
		},
		Exchange:          ex.Name,
		Pair:              futuresTradablePair,
		Asset:             asset.Futures,
		LastUpdated:       time.UnixMilli(1792843031070),
		LastUpdateID:      42904904540,
		ValidateOrderbook: true,
		RestSnapshot:      true,
	}
	assert.Equal(t, exp, result, "the futures book should be loaded")
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	startTime, endTime := recentWindow()
	_, err := e.GetHistoricCandles(t.Context(), currency.EMPTYPAIR, asset.Spot, kline.FiveMin, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.GetHistoricCandles(t.Context(), spotTradablePair, asset.Spot, kline.TenMin, startTime, endTime)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval)

	_, err = e.GetHistoricCandles(t.Context(), spotTradablePair, asset.Spot, kline.FiveMin, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	result, err := e.GetHistoricCandles(t.Context(), spotTradablePair, asset.Spot, kline.FiveMin, startTime, endTime)
	require.NoError(t, err)
	require.NotEmpty(t, result.Candles, "candles must be returned")
	for i := range result.Candles {
		assert.NotZerof(t, result.Candles[i].Open, "candle %d should carry real data, not zero padding over a dropped interval", i)
	}

	_, err = e.GetHistoricCandles(t.Context(), spotTradablePair, asset.Options, kline.FiveMin, startTime, endTime)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = e.GetHistoricCandles(t.Context(), spotTradablePair, asset.Spot, kline.EightHour, startTime, endTime)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval, "the futures only eight hour interval must be refused for spot")

	start, end := futuresWindow()
	_, err = e.GetHistoricCandles(t.Context(), futuresTradablePair, asset.Futures, kline.TenMin, start, end)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval)
	// A linear contract's volume is its contracts times their size, and an inverse contract's is its turnover
	exp := map[currency.Pair]*kline.Item{
		futuresTradablePair: {
			Exchange: "MEXC",
			Pair:     futuresTradablePair,
			Asset:    asset.Futures,
			Interval: kline.OneHour,
			Candles: []kline.Candle{
				{Time: time.Unix(1792800000, 0), Open: 81159.9, High: 81188.2, Low: 80574.2, Close: 80588.3, Volume: 1649.9022},
				{Time: time.Unix(1792803600, 0), Open: 80588.3, High: 80935, Low: 80586.4, Close: 80898.9, Volume: 1000.6081},
				{Time: time.Unix(1792807200, 0), Open: 80898.9, High: 80965, Low: 80801.5, Close: 80963.4, Volume: 403.0439},
				{Time: time.Unix(1792810800, 0), Open: 80963.4, High: 81068.6, Low: 80811, Close: 80943.9, Volume: 567.5471},
				{Time: time.Unix(1792814400, 0), Open: 80943.9, High: 81069.4, Low: 80812, Close: 80857.2, Volume: 534.5322},
				{Time: time.Unix(1792818000, 0), Open: 80857.2, High: 80859.8, Low: 80681.7, Close: 80732.4, Volume: 736.3386},
			},
		},
		futuresInversePair: futuresInverseCandles(),
	}
	for _, p := range []currency.Pair{futuresTradablePair, futuresInversePair} {
		result, err = e.GetHistoricCandles(t.Context(), p, asset.Futures, kline.OneHour, start, end)
		require.NoErrorf(t, err, "GetHistoricCandles must not error for %s", p)
		if mockTests {
			assert.Equalf(t, exp[p], result, "the %s candles should be mapped", p)
			continue
		}
		require.NotEmptyf(t, result.Candles, "candles must be returned for %s", p)
		for i := range result.Candles {
			assert.NotZerof(t, result.Candles[i].Open, "%s candle %d should carry real data", p, i)
			assert.Positivef(t, result.Candles[i].Volume, "%s candle %d should carry its base volume", p, i)
			assert.Falsef(t, result.Candles[i].Time.Before(start), "%s candle %d should not open before the window", p, i)
			assert.Truef(t, result.Candles[i].Time.Before(end), "%s candle %d should open before the window's end", p, i)
		}
	}
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricCandlesExtended(t.Context(), currency.EMPTYPAIR, asset.Spot, kline.FiveMin, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	startTime, endTime := recentWindow()
	_, err = e.GetHistoricCandlesExtended(t.Context(), spotTradablePair, asset.Spot, kline.TenMin, startTime, endTime)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval)

	_, err = e.GetHistoricCandlesExtended(t.Context(), spotTradablePair, asset.Options, kline.TenMin, startTime, endTime)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	result, err := e.GetHistoricCandlesExtended(t.Context(), spotTradablePair, asset.Spot, kline.FiveMin, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)

	_, err = e.GetHistoricCandlesExtended(t.Context(), spotTradablePair, asset.Options, kline.FiveMin, startTime, endTime)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	start, end := futuresWindow()
	result, err = e.GetHistoricCandlesExtended(t.Context(), futuresInversePair, asset.Futures, kline.OneHour, start, end)
	require.NoError(t, err, "GetHistoricCandlesExtended must not error for futures")
	if mockTests {
		assert.Equal(t, futuresInverseCandles(), result, "the futures candles should be mapped")
		return
	}
	require.NotEmpty(t, result.Candles, "futures candles must be returned")
	assert.False(t, result.Candles[0].Time.Before(start), "no futures candle should open before the window")
}

// futuresInverseCandles returns the recorded hourly BTC_USD candles of futuresWindow as the wrapper maps them
func futuresInverseCandles() *kline.Item {
	return &kline.Item{
		Exchange: "MEXC",
		Pair:     futuresInversePair,
		Asset:    asset.Futures,
		Interval: kline.OneHour,
		Candles: []kline.Candle{
			{Time: time.Unix(1792800000, 0), Open: 81056.1, High: 81079.9, Low: 80477.2, Close: 80493, Volume: 6.476927305570522},
			{Time: time.Unix(1792803600, 0), Open: 80493, High: 80840.1, Low: 80488.6, Close: 80803.1, Volume: 2.661336822224324},
			{Time: time.Unix(1792807200, 0), Open: 80803.1, High: 80866.3, Low: 80720, Close: 80866.3, Volume: 2.377179236252418},
			{Time: time.Unix(1792810800, 0), Open: 80866.3, High: 80960.1, Low: 80722.5, Close: 80845.5, Volume: 4.291749117604468},
			{Time: time.Unix(1792814400, 0), Open: 80845.5, High: 80963.5, Low: 80719.5, Close: 80755.5, Volume: 2.006685669693846},
			{Time: time.Unix(1792818000, 0), Open: 80755.5, High: 80761.5, Low: 80598, Close: 80633.4, Volume: 2.8458177805402807},
		},
	}
}

// TestKlineGlobalResultLimit pins the kline page size. MEXC's /api/v3/klines returns at most 500 rows
// however large a limit is requested, so a GlobalResultLimit of 1000 makes GetHistoricCandlesExtended
// size each range at 1000 candles, fill it half way, and zero-pad the remainder silently.
func TestKlineGlobalResultLimit(t *testing.T) {
	t.Parallel()
	assert.Equal(t, uint64(500), e.Features.Enabled.Kline.GlobalResultLimit, "MEXC klines return at most 500 rows, so the global result limit should be 500")
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	sTime, err := e.GetServerTime(t.Context(), asset.Empty)
	require.NoError(t, err)
	assert.NotEmpty(t, sTime)

	sTime, err = e.GetServerTime(t.Context(), asset.Futures)
	require.NoError(t, err, "GetServerTime must not error for futures")
	if mockTests {
		assert.Equal(t, time.UnixMilli(1792843031176), sTime, "GetServerTime should return the futures API's time")
	} else {
		assert.False(t, sTime.IsZero(), "GetServerTime should return the futures API's time")
	}
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	err := e.UpdateOrderExecutionLimits(t.Context(), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	err = e.UpdateOrderExecutionLimits(t.Context(), asset.Spot)
	require.NoErrorf(t, err, "Error fetching %s pairs for test: %v", asset.Spot, err)

	instrumentInfo, err := e.GetSymbols(t.Context(), []string{spotTradablePair.String()})
	require.NoError(t, err)
	require.NotEmpty(t, instrumentInfo.Symbols[0])

	lms, err := e.GetOrderExecutionLimits(asset.Spot, spotTradablePair)
	require.NoError(t, err)
	require.NotNil(t, lms)

	symbolDetail := instrumentInfo.Symbols[0]
	require.NotNil(t, symbolDetail, "instrument required to be found")
	// The price tick is 10^-quotePrecision, not the quoteAmountPrecision value: quoteAmountPrecision
	// is the minimum quote order amount (min notional). The previous mapping put that amount into the
	// price step, quantizing prices to whole quote units.
	require.Equal(t, math.Pow(10, -symbolDetail.QuotePrecision), lms.PriceStepIncrementSize, "price tick must be 10^-quotePrecision")
	assert.Equal(t, symbolDetail.QuoteAmountPrecision.Float64(), lms.MinimumQuoteAmount, "quoteAmountPrecision should map to the minimum quote amount")
	assert.Equal(t, math.Pow(10, -symbolDetail.BaseAssetPrecision), lms.AmountStepIncrementSize, "base amount step should be 10^-baseAssetPrecision")
	assert.Equal(t, symbolDetail.BaseSizePrecision.Float64(), lms.MinimumBaseAmount)
	assert.Equal(t, symbolDetail.MaxQuoteAmount.Float64(), lms.MaximumQuoteAmount)

	require.NoError(t, e.UpdateOrderExecutionLimits(t.Context(), asset.Futures), "UpdateOrderExecutionLimits must not error for futures")
	contracts, err := e.GetFuturesContracts(t.Context(), futuresTradablePair)
	require.NoError(t, err, "GetFuturesContracts must not error")
	require.Len(t, contracts, 1, "the contract must be returned")
	lms, err = e.GetOrderExecutionLimits(asset.Futures, futuresTradablePair)
	require.NoError(t, err, "GetOrderExecutionLimits must not error for futures")
	if mockTests {
		assert.False(t, lms.UpdatedAt.IsZero(), "the futures limits should record when they were loaded")
		lms.UpdatedAt = time.Time{}
		// The contract's price unit is the price tick and its volume unit the amount step; limit orders are capped at the
		// larger of its order caps and market orders at its maximum volume, and its size is the multiplier
		exp := limits.MinMaxLevel{
			Key:                     key.NewExchangeAssetPair("MEXC", asset.Futures, futuresTradablePair),
			PriceStepIncrementSize:  0.1,
			MultiplierDecimal:       0.0001,
			MinimumBaseAmount:       1,
			MaximumBaseAmount:       2500000,
			AmountStepIncrementSize: 1,
			MarketMinQty:            1,
			MarketMaxQty:            400000,
			MarketStepIncrementSize: 1,
		}
		assert.Equal(t, exp, lms, "the futures limits should be the contract's")
		return
	}
	c := contracts[0]
	assert.Equal(t, c.PriceUnit, lms.PriceStepIncrementSize, "the price tick should be the contract's price unit")
	assert.Equal(t, c.VolumeUnit, lms.AmountStepIncrementSize, "the amount step should be the contract's volume unit")
	assert.Equal(t, c.MinVolume, lms.MinimumBaseAmount, "the minimum amount should be the contract's minimum volume")
	assert.Equal(t, max(c.LimitMaxVolume, c.MaxVolume), lms.MaximumBaseAmount, "the maximum amount should be the larger limit order cap")
	assert.Equal(t, c.MaxVolume, lms.MarketMaxQty, "the market order cap should be the contract's maximum volume")
	assert.Equal(t, c.ContractSize, lms.MultiplierDecimal, "the multiplier should be the contract size")
}

func TestUpdateAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.UpdateAccountBalances(t.Context(), asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)

	_, err = e.UpdateAccountBalances(t.Context(), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported)
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Empty)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFundingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAccountFundingHistory(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetRecentTrades(t.Context(), currency.EMPTYPAIR, asset.Options)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.GetRecentTrades(t.Context(), spotTradablePair, asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	result, err := e.GetRecentTrades(t.Context(), spotTradablePair, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)

	_, err = e.GetRecentTrades(t.Context(), spotTradablePair, asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	result, err = e.GetRecentTrades(t.Context(), futuresTradablePair, asset.Futures)
	require.NoError(t, err, "GetRecentTrades must not error for futures")
	if mockTests {
		// The taker's side is the trade's, and the amounts are in contracts
		recorded := func(id string, side order.Side, price, amount float64, ms int64) trade.Data {
			return trade.Data{TID: id, Exchange: "MEXC", CurrencyPair: futuresTradablePair, AssetType: asset.Futures, Side: side, Price: price, Amount: amount, Timestamp: time.UnixMilli(ms)}
		}
		exp := []trade.Data{
			recorded("16745174629", order.Sell, 80565, 50, 1792843027529),
			recorded("16745174619", order.Buy, 80565.1, 6, 1792843027337),
			recorded("16745174618", order.Buy, 80565.1, 4, 1792843027337),
			recorded("16745174489", order.Sell, 80565, 6, 1792843026337),
			recorded("16745174482", order.Sell, 80565, 4, 1792843026336),
		}
		assert.Equal(t, exp, result, "the futures trades should be mapped")
		return
	}
	require.NotEmpty(t, result, "futures trades must be returned")
	for i := range result {
		assert.NotEmptyf(t, result[i].TID, "futures trade %d should carry its ID", i)
		assert.NotEqualf(t, order.UnknownSide, result[i].Side, "futures trade %d should carry the taker's side", i)
		assert.Positivef(t, result[i].Amount, "futures trade %d should carry its amount", i)
	}
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	startTime, endTime := recentWindow()
	_, err := e.GetHistoricTrades(t.Context(), currency.EMPTYPAIR, asset.Options, startTime, endTime)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.GetHistoricTrades(t.Context(), spotTradablePair, asset.Options, startTime, endTime)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	result, err := e.GetHistoricTrades(ctx, spotTradablePair, asset.Spot, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)

	_, err = e.GetHistoricTrades(t.Context(), spotTradablePair, asset.Options, startTime, endTime)
	require.ErrorIs(t, err, asset.ErrNotSupported)
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.GetDepositAddress(t.Context(), currency.EMPTYCODE, "", "TON")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetDepositAddress(t.Context(), currency.BTC, "", "TON")
	require.True(t, err != nil || err == deposit.ErrAddressNotFound)
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	arg := &order.MultiOrderRequest{AssetType: asset.Options, Side: order.AnySide, Type: order.AnyType}
	_, err := e.GetActiveOrders(t.Context(), arg)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	arg.AssetType = asset.Spot
	arg.Pairs = currency.Pairs{}
	_, err = e.GetActiveOrders(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	arg.Pairs = currency.Pairs{spotTradablePair}
	_, err = e.GetActiveOrders(t.Context(), arg)
	require.NoError(t, err)
}

func TestGenerateListenKey(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	listenKey, err := e.GenerateListenKey(t.Context())
	require.NoError(t, err)
	assert.NotEmpty(t, listenKey)
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderInfo(t.Context(), "12342", currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	// See TestGetOrderByID: the order does not exist and the tests may not create one.
	if _, err = e.GetOrderInfo(t.Context(), "12342", spotTradablePair, asset.Spot); err != nil {
		assert.ErrorContains(t, err, "Order does not exist", "the only acceptable failure is the order being absent")
	}

	// Not ErrAssetNotSet: the asset is set, it is options, which this exchange does not support.
	_, err = e.GetOrderInfo(t.Context(), "12342", spotTradablePair, asset.Options)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	_, err := e.SubmitOrder(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrSubmissionIsNil)

	_, err = e.SubmitOrder(t.Context(), &order.Submit{Exchange: e.Name})
	require.ErrorIs(t, err, order.ErrPairIsEmpty)

	arg := &order.Submit{
		Exchange:  e.Name,
		Pair:      spotTradablePair,
		AssetType: asset.Options,
		Type:      order.Liquidation,
		Side:      order.Long,
		Amount:    1,
	}
	_, err = e.SubmitOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrAssetNotFound)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	// SubmitOrder assigns the formatted pair back onto the submission, so a failed format leaves it
	// empty for the next call.
	arg.Pair = spotTradablePair
	arg.AssetType = asset.Options
	_, err = e.SubmitOrder(t.Context(), arg)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	arg.Pair = spotTradablePair
	arg.AssetType = asset.Spot
	_, err = e.SubmitOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrUnsupportedTimeInForce)
	require.ErrorIs(t, err, order.ErrUnsupportedOrderType)

	// A submission the shared validator refuses must not reach the venue.
	arg.Type = order.Limit
	arg.Side = order.Sell
	arg.Amount = -1
	arg.QuoteAmount = 100
	_, err = e.SubmitOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	// Spot orders test: a zero amount and a missing limit price now surface through the shared
	// validator, ahead of the venue's own limits errors.
	arg.Amount = 0
	arg.QuoteAmount = 0
	_, err = e.SubmitOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid)

	arg.Amount = .1
	_, err = e.SubmitOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrPriceMustBeSetIfLimitOrder)

	arg.Price = 1234567
	result, err := e.SubmitOrder(t.Context(), arg)
	require.NoError(t, err)
	assert.NotNil(t, result)

	arg.AssetType = asset.Options
	_, err = e.SubmitOrder(t.Context(), arg)
	require.ErrorIs(t, err, asset.ErrNotSupported)
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	err := e.CancelOrder(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrCancelOrderIsNil)

	err = e.CancelOrder(t.Context(), &order.Cancel{})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	err = e.CancelOrder(t.Context(), &order.Cancel{OrderID: "987654"})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	err = e.CancelOrder(t.Context(), &order.Cancel{OrderID: "987654", AssetType: asset.Spot})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.CancelOrder(t.Context(), &order.Cancel{OrderID: "987654", Pair: spotTradablePair, AssetType: asset.Spot})
	assert.NoError(t, err)

	err = e.CancelOrder(t.Context(), &order.Cancel{OrderID: "987654", Pair: spotTradablePair, AssetType: asset.Options})
	assert.ErrorIs(t, err, asset.ErrNotSupported)
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrCancelOrderIsNil)

	// A symbol-wide cancel does not require an order id: an empty request now
	// fails on the unset asset, and supplying an order id changes nothing.
	_, err = e.CancelAllOrders(t.Context(), &order.Cancel{})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = e.CancelAllOrders(t.Context(), &order.Cancel{OrderID: "12345"})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	// A spot request without a pair cancels account-wide; TestCancelAllOrdersWithoutPairCancelsAccountWide
	// covers it against a local server rather than cancelling every order of a live account here.

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelAllOrders(t.Context(), &order.Cancel{OrderID: "12345", AssetType: asset.Spot, Pair: spotTradablePair})
	require.NoError(t, err)
	assert.NotNil(t, result)

	_, err = e.CancelAllOrders(t.Context(), &order.Cancel{OrderID: "12345", AssetType: asset.Options, Pair: spotTradablePair})
	require.ErrorIs(t, err, asset.ErrNotSupported)
}

func TestAccountStatusToString(t *testing.T) {
	t.Parallel()
	for status, expected := range map[uint8]string{
		1: "SMALL", 2: "TIME_DELAY", 3: "LARGE_DELAY", 4: "PENDING",
		5: "SUCCESS", 6: "AUDITING", 7: "REJECTED", 8: "REFUND", 9: "PRE_SUCCESS",
		10: "INVALID", 11: "RESTRICTED", 12: "COMPLETED", 0: "", 99: "",
	} {
		assert.Equalf(t, expected, accountStatusToString(status), "accountStatusToString should return correct value for %d", status)
	}
}

func TestWithdrawalStatusToString(t *testing.T) {
	t.Parallel()
	for status, expected := range map[uint8]string{
		1: "APPLY", 2: "AUDITING", 3: "WAIT", 4: "PROCESSING", 5: "WAIT_PACKAGING",
		6: "WAIT_CONFIRM", 7: "SUCCESS", 8: "FAILED", 9: "CANCEL", 10: "MANUAL", 0: "", 99: "",
	} {
		assert.Equalf(t, expected, withdrawalStatusToString(status), "withdrawalStatusToString should return correct value for %d", status)
	}
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	resp, err := e.GetCurrencyTradeURL(t.Context(), asset.Spot, spotTradablePair)
	require.NoError(t, err, "GetCurrencyTradeURL must not error")
	assert.Equal(t, "https://www.mexc.com/exchange/"+spotTradablePair.Base.Upper().String()+"_"+spotTradablePair.Quote.Upper().String(), resp, "the spot trade URL should name the pair as BASE_QUOTE")

	_, err = e.GetCurrencyTradeURL(t.Context(), asset.Options, spotTradablePair)
	assert.ErrorIs(t, err, asset.ErrNotSupported, "an unsupported asset should be reported as unsupported")

	resp, err = e.GetCurrencyTradeURL(t.Context(), asset.Futures, futuresTradablePair)
	require.NoError(t, err, "GetCurrencyTradeURL must not error for futures")
	assert.Equal(t, "https://www.mexc.com/futures/BTC_USDT", resp, "the futures trade URL should name the contract")
}

func TestSendFuturesHTTPRequest(t *testing.T) {
	t.Parallel()
	// signature returns what the venue verifies a request's Signature header against
	signature := func(t *testing.T, r *http.Request, signed string) string {
		t.Helper()
		mac, err := crypto.GetHMAC(crypto.HashSHA256, []byte("test-key"+r.Header.Get("Request-Time")+signed), []byte("test-secret"))
		assert.NoError(t, err, "GetHMAC should not error")
		return hex.EncodeToString(mac)
	}
	for _, method := range []string{http.MethodGet, http.MethodDelete, http.MethodPost} {
		t.Run("signs a "+method, func(t *testing.T) {
			t.Parallel()
			values, body := url.Values{"z": {"1,2"}, "a": {"x y"}}, any(nil)
			if method == http.MethodPost {
				values, body = nil, map[string]any{"b": 2, "a": "x y"}
			}
			ex := newPrivateTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
				sent, err := io.ReadAll(r.Body)
				assert.NoError(t, err, "reading the request body should not error")
				signed := r.URL.RawQuery
				if method == http.MethodPost {
					signed = string(sent)
					assert.JSONEq(t, `{"a":"x y","b":2}`, signed, "a POST should carry its parameters as a JSON body")
				} else {
					const sorted = "a=x%20y&z=1%2C2"
					assert.Equal(t, sorted, signed, "the query should be sorted, with a space encoded as the venue signs it")
					assert.Empty(t, sent, "a request without a body should send none")
				}
				assert.Equal(t, "test-key", r.Header.Get("ApiKey"), "the API key should be sent")
				assert.Equal(t, signature(t, r, signed), r.Header.Get("Signature"), "the request should be signed over what it sends")
				_, _ = w.Write([]byte(`{"success":true,"code":0}`))
			})
			require.NoError(t, ex.SendFuturesHTTPRequest(t.Context(), futuresAccountAssetsEPL, method, "private/probe", values, body, nil, true), "SendFuturesHTTPRequest must not error")
		})
	}

	for reply, exp := range map[string]struct {
		err     error
		message string
	}{
		`{"success":false,"code":1001,"message":"contract not activated"}`: {errFuturesRequestFailed, "code 1001: contract not activated"},
		`{"success":false,"code":600,"msg":"parameter error"}`:             {errFuturesRequestFailed, "code 600: parameter error"},
		// As ccxt reads the envelope, a reply is refused only when it carries neither success nor a success code
		`{"success":true,"code":200,"msg":"success"}`: {},
		`{"success":false,"code":0}`:                  {},
	} {
		ex := newPrivateTestExchange(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(reply)) })
		err := ex.SendFuturesHTTPRequest(t.Context(), futuresServerTimeEPL, http.MethodGet, "contract/ping", nil, nil, nil, false)
		assert.ErrorIsf(t, err, exp.err, "the reply %s should be read", reply)
		if exp.err != nil {
			assert.ErrorContainsf(t, err, exp.message, "the refusal %s should carry its code and message", reply)
		}
	}

	ex := newPrivateTestExchange(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"success":false,"code":401,"message":"Not logged in"}`))
	})
	err := ex.SendFuturesHTTPRequest(t.Context(), futuresAccountAssetsEPL, http.MethodGet, "private/account/assets", nil, nil, nil, true)
	assert.ErrorIs(t, err, request.ErrAuthRequestFailed, "a refused key should fail the authenticated request")

	// A retry is signed afresh, since the venue refuses a request time outside its window
	var mu sync.Mutex
	var times, signatures []string
	ex = newPrivateTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		times, signatures = append(times, r.Header.Get("Request-Time")), append(signatures, r.Header.Get("Signature"))
		attempt := len(times)
		mu.Unlock()
		if attempt == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(`{"success":true,"code":0}`))
	})
	require.NoError(t, ex.SendFuturesHTTPRequest(t.Context(), futuresAccountAssetsEPL, http.MethodGet, "private/account/assets", nil, nil, nil, true), "a retried request must succeed")
	require.Len(t, times, 2, "the request must be retried once")
	assert.NotEqual(t, times[0], times[1], "the retry should carry a new request time")
	assert.NotEqual(t, signatures[0], signatures[1], "the retry should be signed again")
}
