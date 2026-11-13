package mexc

import (
	"bytes"
	"context"
	"encoding/hex"
	"io"
	"maps"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
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
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deposit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
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
	// marketMakerAccount is for a market maker account, the only accounts MEXC serves batch futures orders and
	// self-trade prevention groups to
	marketMakerAccount = false
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

//nolint:tparallel // the cases share e, which stores the futures contracts it reads, so they run in order
func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	// The cases share e, which stores the futures contracts it reads, so each also checks that a call keeps nothing from the last
	for k, v := range assetsAndErrors {
		t.Run(k.String(), func(t *testing.T) {
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
		})
	}
}

//nolint:tparallel // the futures cases share ex in order, so each also checks that a call keeps nothing from the last
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
	// The cases share ex in order, so each also checks that a call keeps nothing from the last
	for _, p := range []currency.Pair{futuresTradablePair, futuresInversePair} {
		t.Run(p.String(), func(t *testing.T) {
			result, err = ex.UpdateTicker(t.Context(), p, asset.Futures)
			require.NoErrorf(t, err, "UpdateTicker must not error for %s", p)
			if mockTests {
				assert.Equalf(t, exp[p], result, "the %s ticker should be mapped", p)
				return
			}
			assert.Truef(t, result.Pair.Equal(p), "the %s ticker should carry its pair", p)
			assert.Positivef(t, result.Last, "the %s ticker should carry the last price", p)
			assert.Positivef(t, result.BaseVolume, "the %s ticker should carry its base volume", p)
			assert.Positivef(t, result.QuoteVolume, "the %s ticker should carry its quote volume", p)
			assert.Positivef(t, result.OpenInterest, "the %s ticker should carry its open interest", p)
			assert.Positivef(t, result.MarkPrice, "the %s ticker should carry its mark price", p)
			assert.Positivef(t, result.IndexPrice, "the %s ticker should carry its index price", p)
		})
	}
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	// The cases share e, which stores the tickers and futures contracts it reads, so they run in order
	for k, v := range assetsAndErrors {
		t.Run(k.String(), func(t *testing.T) {
			err := e.UpdateTickers(t.Context(), k)
			require.ErrorIs(t, err, v)
		})
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
		t.Run(p.String(), func(t *testing.T) {
			t.Parallel()
			result, err := ticker.GetTicker(ex.Name, p, asset.Futures)
			require.NoErrorf(t, err, "GetTicker must not error for %s", p)
			if mockTests {
				assert.Equalf(t, exp[p], result, "the %s ticker should be stored", p)
				return
			}
			assert.Positivef(t, result.Last, "the %s ticker should carry the last price", p)
			assert.Positivef(t, result.OpenInterest, "the %s ticker should carry its open interest", p)
		})
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
	t.Run("failures", func(t *testing.T) {
		t.Parallel()
		for _, tc := range historicCandlesFailures() {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := testExchangeFor(t, tc.handler(t), liveUnavailable)
				if tc.setup != nil {
					tc.setup(t, ex)
				}
				start, end := futuresWindow()
				_, err := ex.GetHistoricCandles(t.Context(), tc.pair, tc.a, kline.OneHour, start, end)
				assert.ErrorIs(t, err, tc.err, "GetHistoricCandles should return why no candles were returned")
			})
		}
	})
	t.Run("source intervals", func(t *testing.T) {
		t.Parallel()
		for _, tc := range candleSources() {
			t.Run(tc.a.String()+" "+tc.interval.Word(), func(t *testing.T) {
				t.Parallel()
				var venue candleIntervals
				ex := testExchangeFor(t, venue.handler(t), liveUnavailable)
				start, end := futuresWindow()
				_, err := ex.GetHistoricCandles(t.Context(), tc.pair, tc.a, tc.interval, start, end)
				require.ErrorIs(t, err, kline.ErrNoTimeSeriesDataToConvert, "GetHistoricCandles must return that the venue sent no candles")
				assert.Equal(t, []string{tc.source}, venue.intervals(), "GetHistoricCandles should request the interval the candles are built from")
			})
		}
	})
	startTime, endTime := recentWindow()
	_, err := e.GetHistoricCandles(t.Context(), currency.EMPTYPAIR, asset.Spot, kline.FiveMin, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.GetHistoricCandles(t.Context(), spotTradablePair, asset.Spot, kline.FifteenSecond, startTime, endTime)
	require.ErrorIs(t, err, kline.ErrCannotConstructInterval, "an interval no spot interval builds must be refused")

	_, err = e.GetHistoricCandles(t.Context(), spotTradablePair, asset.Spot, kline.FiveMin, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	result, err := e.GetHistoricCandles(t.Context(), spotTradablePair, asset.Spot, kline.FiveMin, startTime, endTime)
	require.NoError(t, err)
	require.NotEmpty(t, result.Candles, "candles must be returned")
	for i := range result.Candles {
		assert.NotZerof(t, result.Candles[i].Open, "candle %d should carry real data, not zero padding over a dropped interval", i)
	}
	dashed := currency.NewPairWithDelimiter(spotTradablePair.Base.String(), spotTradablePair.Quote.String(), currency.DashDelimiter)
	fromDashed, err := e.GetHistoricCandles(t.Context(), dashed, asset.Spot, kline.FiveMin, startTime, endTime)
	require.NoError(t, err, "GetHistoricCandles must not error for a dashed spot pair")
	assert.Equal(t, result, fromDashed, "GetHistoricCandles should format a spot pair for the venue")

	// Ten minute and two hour spot candles are built from the venue's five minute and hourly ones over the window widened to
	// whole candles; the two hour window is moved back so that each of its candles has closed when run live.
	// The cases share e and result with the statements around them, so they run in order
	for _, tc := range []struct {
		interval, source kline.Interval
		start, end       time.Time
	}{
		{interval: kline.TenMin, source: kline.FiveMin, start: startTime, end: endTime},
		{interval: kline.TwoHour, source: kline.OneHour, start: startTime.Add(-2 * time.Hour), end: endTime.Add(-2 * time.Hour)},
	} {
		t.Run("spot "+tc.interval.Short(), func(t *testing.T) {
			req, err := e.GetKlineRequest(spotTradablePair, asset.Spot, tc.interval, tc.start, tc.end, false)
			require.NoErrorf(t, err, "GetKlineRequest must not error for %s", tc.interval.Short())
			source, err := e.GetHistoricCandles(t.Context(), spotTradablePair, asset.Spot, tc.source, req.Start, req.End)
			require.NoErrorf(t, err, "GetHistoricCandles must not error for %s spot candles", tc.source.Short())
			built, err := source.ConvertToNewInterval(tc.interval)
			require.NoErrorf(t, err, "ConvertToNewInterval must not error for %s", tc.interval.Short())
			result, err = e.GetHistoricCandles(t.Context(), spotTradablePair, asset.Spot, tc.interval, tc.start, tc.end)
			require.NoErrorf(t, err, "GetHistoricCandles must build %s spot candles", tc.interval.Short())
			assert.Equalf(t, built, result, "GetHistoricCandles should build %s spot candles", tc.interval.Short())
		})
	}

	_, err = e.GetHistoricCandles(t.Context(), spotTradablePair, asset.Options, kline.FiveMin, startTime, endTime)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = e.GetHistoricCandles(t.Context(), spotTradablePair, asset.Spot, kline.EightHour, startTime, endTime)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval, "the futures only eight hour interval must be refused for spot")

	start, end := futuresWindow()
	_, err = e.GetHistoricCandles(t.Context(), futuresTradablePair, asset.Futures, kline.FifteenSecond, start, end)
	require.ErrorIs(t, err, kline.ErrCannotConstructInterval, "an interval no futures interval builds must be refused")
	// Ten minute and two hour candles are built from the venue's five minute and hourly ones, from the start of the candle a
	// window starts in. The cases share e and result with the statements around them, so they run in order
	for _, tc := range []struct{ interval, source kline.Interval }{{kline.TenMin, kline.FiveMin}, {kline.TwoHour, kline.OneHour}} {
		t.Run("futures "+tc.interval.Short(), func(t *testing.T) {
			source, err := e.GetHistoricCandles(t.Context(), futuresTradablePair, asset.Futures, tc.source, start, end)
			require.NoErrorf(t, err, "GetHistoricCandles must not error for %s futures candles", tc.source.Short())
			built, err := source.ConvertToNewInterval(tc.interval)
			require.NoErrorf(t, err, "ConvertToNewInterval must not error for %s", tc.interval.Short())
			for _, window := range [][2]time.Time{{start, end}, {start.Add(time.Minute), end.Add(-time.Minute)}} {
				t.Run(window[0].UTC().Format("15:04")+" to "+window[1].UTC().Format("15:04"), func(t *testing.T) {
					result, err = e.GetHistoricCandles(t.Context(), futuresTradablePair, asset.Futures, tc.interval, window[0], window[1])
					require.NoErrorf(t, err, "GetHistoricCandles must build %s futures candles from %s to %s", tc.interval.Short(), window[0], window[1])
					assert.Equalf(t, built, result, "GetHistoricCandles should build %s futures candles from %s to %s", tc.interval.Short(), window[0], window[1])
				})
			}
		})
	}
	formatted, err := e.GetHistoricCandles(t.Context(), futuresTradablePair, asset.Futures, kline.OneHour, start, end)
	require.NoError(t, err, "GetHistoricCandles must not error for futures candles")
	result, err = e.GetHistoricCandles(t.Context(), currency.NewBTCUSDT(), asset.Futures, kline.OneHour, start, end)
	require.NoError(t, err, "GetHistoricCandles must not error for an unformatted futures pair")
	assert.Equal(t, formatted, result, "GetHistoricCandles should format a futures pair for the venue")
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
	// The cases share e and result with the statements before them, so they run in order
	for _, p := range []currency.Pair{futuresTradablePair, futuresInversePair} {
		t.Run(p.String(), func(t *testing.T) {
			result, err = e.GetHistoricCandles(t.Context(), p, asset.Futures, kline.OneHour, start, end)
			require.NoErrorf(t, err, "GetHistoricCandles must not error for %s", p)
			if mockTests {
				assert.Equalf(t, exp[p], result, "the %s candles should be mapped", p)
				return
			}
			require.NotEmptyf(t, result.Candles, "candles must be returned for %s", p)
			for i := range result.Candles {
				assert.NotZerof(t, result.Candles[i].Open, "%s candle %d should carry real data", p, i)
				assert.Positivef(t, result.Candles[i].Volume, "%s candle %d should carry its base volume", p, i)
				assert.Falsef(t, result.Candles[i].Time.Before(start), "%s candle %d should not open before the window", p, i)
				assert.Truef(t, result.Candles[i].Time.Before(end), "%s candle %d should open before the window's end", p, i)
			}
		})
	}
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	t.Run("failures", func(t *testing.T) {
		t.Parallel()
		for _, tc := range historicCandlesFailures() {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := testExchangeFor(t, tc.handler(t), liveUnavailable)
				if tc.setup != nil {
					tc.setup(t, ex)
				}
				start, end := futuresWindow()
				_, err := ex.GetHistoricCandlesExtended(t.Context(), tc.pair, tc.a, kline.OneHour, start, end)
				assert.ErrorIs(t, err, tc.err, "GetHistoricCandlesExtended should return why no candles were returned")
			})
		}
	})
	t.Run("source intervals", func(t *testing.T) {
		t.Parallel()
		for _, tc := range candleSources() {
			t.Run(tc.a.String()+" "+tc.interval.Word(), func(t *testing.T) {
				t.Parallel()
				var venue candleIntervals
				ex := testExchangeFor(t, venue.handler(t), liveUnavailable)
				start, end := futuresWindow()
				_, err := ex.GetHistoricCandlesExtended(t.Context(), tc.pair, tc.a, tc.interval, start, end)
				require.ErrorIs(t, err, kline.ErrNoTimeSeriesDataToConvert, "GetHistoricCandlesExtended must return that the venue sent no candles")
				assert.Equal(t, []string{tc.source}, venue.intervals(), "GetHistoricCandlesExtended should request the interval the candles are built from")
			})
		}
	})
	_, err := e.GetHistoricCandlesExtended(t.Context(), currency.EMPTYPAIR, asset.Spot, kline.FiveMin, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	startTime, endTime := recentWindow()
	_, err = e.GetHistoricCandlesExtended(t.Context(), spotTradablePair, asset.Spot, kline.FifteenSecond, startTime, endTime)
	require.ErrorIs(t, err, kline.ErrCannotConstructInterval, "an interval no spot interval builds must be refused")

	_, err = e.GetHistoricCandlesExtended(t.Context(), spotTradablePair, asset.Spot, kline.EightHour, startTime, endTime)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval, "the futures only eight hour interval must be refused for spot")

	_, err = e.GetHistoricCandlesExtended(t.Context(), spotTradablePair, asset.Options, kline.TenMin, startTime, endTime)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	result, err := e.GetHistoricCandlesExtended(t.Context(), spotTradablePair, asset.Spot, kline.FiveMin, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
	dashed := currency.NewPairWithDelimiter(spotTradablePair.Base.String(), spotTradablePair.Quote.String(), currency.DashDelimiter)
	fromDashed, err := e.GetHistoricCandlesExtended(t.Context(), dashed, asset.Spot, kline.FiveMin, startTime, endTime)
	require.NoError(t, err, "GetHistoricCandlesExtended must not error for a dashed spot pair")
	assert.Equal(t, result.Candles, fromDashed.Candles, "GetHistoricCandlesExtended should format a spot pair for the venue")

	// The cases share e and result with the statements around them, so they run in order
	for _, tc := range []struct {
		interval   kline.Interval
		start, end time.Time
	}{
		{interval: kline.TenMin, start: startTime, end: endTime},
		{interval: kline.TwoHour, start: startTime.Add(-2 * time.Hour), end: endTime.Add(-2 * time.Hour)},
	} {
		t.Run("spot "+tc.interval.Short(), func(t *testing.T) {
			regular, err := e.GetHistoricCandles(t.Context(), spotTradablePair, asset.Spot, tc.interval, tc.start, tc.end)
			require.NoErrorf(t, err, "GetHistoricCandles must build %s spot candles", tc.interval.Short())
			result, err = e.GetHistoricCandlesExtended(t.Context(), spotTradablePair, asset.Spot, tc.interval, tc.start, tc.end)
			require.NoErrorf(t, err, "GetHistoricCandlesExtended must build %s spot candles", tc.interval.Short())
			assert.Equalf(t, regular, result, "GetHistoricCandlesExtended should build the %s spot candles GetHistoricCandles does", tc.interval.Short())
		})
	}

	_, err = e.GetHistoricCandlesExtended(t.Context(), spotTradablePair, asset.Options, kline.FiveMin, startTime, endTime)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	start, end := futuresWindow()
	// The cases share e and result with the statements around them, so they run in order
	for _, interval := range []kline.Interval{kline.TenMin, kline.TwoHour} {
		t.Run("futures "+interval.Short(), func(t *testing.T) {
			regular, err := e.GetHistoricCandles(t.Context(), futuresTradablePair, asset.Futures, interval, start, end)
			require.NoErrorf(t, err, "GetHistoricCandles must build %s futures candles", interval.Short())
			result, err = e.GetHistoricCandlesExtended(t.Context(), futuresTradablePair, asset.Futures, interval, start, end)
			require.NoErrorf(t, err, "GetHistoricCandlesExtended must build %s futures candles", interval.Short())
			assert.Equalf(t, regular, result, "GetHistoricCandlesExtended should build the %s futures candles GetHistoricCandles does", interval.Short())
		})
	}
	hourly, err := e.GetHistoricCandles(t.Context(), futuresTradablePair, asset.Futures, kline.OneHour, start, end)
	require.NoError(t, err, "GetHistoricCandles must not error for futures candles")
	result, err = e.GetHistoricCandlesExtended(t.Context(), currency.NewBTCUSDT(), asset.Futures, kline.OneHour, start, end)
	require.NoError(t, err, "GetHistoricCandlesExtended must not error for an unformatted futures pair")
	assert.Equal(t, hourly.Candles, result.Candles, "GetHistoricCandlesExtended should format a futures pair for the venue")
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
	t.Run("futures", func(t *testing.T) {
		t.Parallel()
		ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
			method: http.MethodGet,
			path:   "private/account/assets",
			data:   `[{"currency":"USDT","positionMargin":1.5,"frozenBalance":2,"availableBalance":10},{"currency":"","availableBalance":1}]`,
		}), liveReadOnly)
		subAccounts, err := ex.UpdateAccountBalances(t.Context(), asset.Futures)
		require.NoError(t, err, "UpdateAccountBalances must not error for futures")
		require.Len(t, subAccounts, 1, "the futures account must be returned")
		assert.Equal(t, asset.Futures, subAccounts[0].AssetType, "the futures account should be named")
		if !mockTests {
			return
		}
		b := subAccounts[0].Balances[currency.USDT]
		assert.False(t, b.UpdatedAt.IsZero(), "the balance should record when it was updated")
		b.UpdatedAt = time.Time{}
		subAccounts[0].Balances[currency.USDT] = b
		// A balance without a currency is skipped
		exp := accounts.SubAccounts{{
			AssetType: asset.Futures,
			Balances:  accounts.CurrencyBalances{currency.USDT: {Currency: currency.USDT, Total: 13.5, Hold: 3.5, Free: 10}},
		}}
		assert.Equal(t, exp, subAccounts, "the futures balances should be mapped")

		t.Run("debt reported as borrowed", func(t *testing.T) {
			t.Parallel()
			owing := futuresHandler(t, futuresRoute{
				method: http.MethodGet,
				path:   "private/account/assets",
				data: `[{"currency":"USDT","positionMargin":1.5,"frozenBalance":2,"availableBalance":0,"cashBalance":0,"equity":0.75,"unrealized":0.75,` +
					`"bonus":4,"availableCash":0,"availableOpen":0,"debtAmount":7.12345678,"contributeMarginAmount":5},` +
					`{"currency":"USDC","availableBalance":20,"cashBalance":20,"equity":20,"availableCash":20,"debtAmount":0,"contributeMarginAmount":3},` +
					`{"currency":"BTC","availableBalance":0.5,"debtAmount":0.00012345}]`,
			})
			repaying := futuresHandler(t, futuresRoute{
				method: http.MethodGet,
				path:   "private/account/assets",
				data:   `[{"currency":"USDT","positionMargin":1.5,"frozenBalance":2,"debtAmount":3.5},{"currency":"BTC","availableBalance":0.5,"debtAmount":0}]`,
			})
			var refreshes atomic.Int32
			ex := testExchangeFor(t, func(w http.ResponseWriter, r *http.Request) {
				if refreshes.Add(1) == 1 {
					owing(w, r)
					return
				}
				repaying(w, r)
			}, liveUnavailable)
			subAccounts, err := ex.UpdateAccountBalances(t.Context(), asset.Futures)
			require.NoError(t, err, "UpdateAccountBalances must not error for futures")
			require.Len(t, subAccounts, 1, "the futures account must be returned")
			for c, b := range subAccounts[0].Balances {
				b.UpdatedAt = time.Time{}
				subAccounts[0].Balances[c] = b
			}
			exp := accounts.CurrencyBalances{
				currency.USDT: {Currency: currency.USDT, Total: 3.5, Hold: 3.5, Borrowed: 7.12345678},
				currency.USDC: {Currency: currency.USDC, Total: 20, Free: 20},
				currency.BTC:  {Currency: currency.BTC, Total: 0.5, Free: 0.5, Borrowed: 0.00012345},
			}
			assert.Equal(t, exp, subAccounts[0].Balances, "each currency's debt should be reported as borrowed")

			// The USDT debt is partly repaid and the BTC debt in full
			subAccounts, err = ex.UpdateAccountBalances(t.Context(), asset.Futures)
			require.NoError(t, err, "UpdateAccountBalances must not error for futures")
			require.Len(t, subAccounts, 1, "the futures account must be returned")
			assert.InDelta(t, 3.5, subAccounts[0].Balances[currency.USDT].Borrowed, 0, "a partly repaid debt should be reported as what is still owed")
			assert.Zero(t, subAccounts[0].Balances[currency.BTC].Borrowed, "a repaid debt should no longer be reported")
		})
	})
	// Serial: UpdateAccountBalances fills e's accounts store, which the futures case also writes when run live
	t.Run("spot live", func(t *testing.T) {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		result, err := e.UpdateAccountBalances(t.Context(), asset.Spot)
		require.NoError(t, err)
		assert.NotNil(t, result)

		_, err = e.UpdateAccountBalances(t.Context(), asset.Options)
		require.ErrorIs(t, err, asset.ErrNotSupported)
	})
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
	t.Run("futures", func(t *testing.T) {
		t.Parallel()
		ex := testExchangeFor(t, futuresHandler(t, futuresContractsRoute, futuresRoute{
			method: http.MethodGet,
			path:   "private/order/list/open_orders",
			query:  "page_num=1&page_size=100",
			data: `[{"orderId":"750330744886358528","symbol":"ETH_USDT","price":2500,"vol":10,"leverage":1,"side":1,"category":1,` +
				`"orderType":1,"dealVol":0,"openType":2,"state":2,"createTime":1764563189993,"updateTime":1764563190059},` +
				`{"orderId":"750330663772702720","symbol":"BTC_USDT","price":80000,"vol":10,"leverage":1,"side":1,"category":1,` +
				`"orderType":1,"dealVol":0,"openType":2,"state":2,"createTime":1764563170653,"updateTime":1764563170723}]`,
		}), liveReadOnly)
		orders, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Futures, Pairs: currency.Pairs{futuresTradablePair}, Side: order.AnySide, Type: order.AnyType})
		require.NoError(t, err, "GetActiveOrders must not error for futures")
		if !mockTests {
			return
		}
		// Only the pair asked for is returned; an unfilled cross limit order is new, multi margined and rests until
		// cancelled, with every contract remaining
		exp := order.FilteredOrders{{
			TimeInForce:     order.GoodTillCancel,
			Leverage:        1,
			Price:           80000,
			Amount:          10,
			ContractAmount:  10,
			RemainingAmount: 10,
			Exchange:        ex.Name,
			OrderID:         "750330663772702720",
			Type:            order.Limit,
			Side:            order.Buy,
			Status:          order.New,
			AssetType:       asset.Futures,
			Date:            time.UnixMilli(1764563170653),
			LastUpdated:     time.UnixMilli(1764563170723),
			Pair:            futuresTradablePair,
			MarginType:      margin.Multi,
		}}
		assert.Equal(t, exp, orders, "the pair's order should be mapped")
	})
	t.Run("futures pages", func(t *testing.T) {
		t.Parallel()
		for name, tc := range map[string]struct {
			second string
			exp    order.FilteredOrders
		}{
			"next page": {
				second: futuresOrderPage(6000, 20, 80000),
				exp:    append(futuresOrderPageDetails(5000, futuresDefaultPageSizeLimit, 80000), futuresOrderPageDetails(6000, 20, 80000)...),
			},
			// A page number the venue ignores serves the first page again, which must end the paging
			"repeated page": {
				second: futuresOrderPage(5000, futuresDefaultPageSizeLimit, 80000),
				exp:    futuresOrderPageDetails(5000, futuresDefaultPageSizeLimit, 80000),
			},
			// Orders placed while paging push the end of the first page onto the next, which repeats them; the repeats are
			// priced differently to show that each order is returned once, as first listed
			"overlapping page": {
				second: futuresOrderPage(5090, 11, 81000),
				exp:    append(futuresOrderPageDetails(5000, futuresDefaultPageSizeLimit, 80000), futuresOrderPageDetails(5100, 1, 81000)...),
			},
			// A null order decodes to nil and is passed over
			"nil order": {
				second: "[null," + strings.TrimPrefix(futuresOrderPage(6000, 1, 80000), "["),
				exp:    append(futuresOrderPageDetails(5000, futuresDefaultPageSizeLimit, 80000), futuresOrderPageDetails(6000, 1, 80000)...),
			},
		} {
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
					method: http.MethodGet, path: "private/order/list/open_orders", query: "page_num=1&page_size=100", data: futuresOrderPage(5000, futuresDefaultPageSizeLimit, 80000),
				}, futuresRoute{
					method: http.MethodGet, path: "private/order/list/open_orders", query: "page_num=2&page_size=100", data: tc.second,
				}), liveUnavailable)
				// A call knows none of the orders an earlier call read
				for call := 1; call <= 2; call++ {
					orders, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Futures, Side: order.AnySide, Type: order.AnyType})
					require.NoErrorf(t, err, "GetActiveOrders call %d must not error", call)
					assert.Equalf(t, tc.exp, orders, "call %d should return every order once, as first listed", call)
				}
			})
		}
	})
	arg := &order.MultiOrderRequest{AssetType: asset.Options, Side: order.AnySide, Type: order.AnyType}
	_, err := e.GetActiveOrders(t.Context(), arg)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	arg.AssetType = asset.Spot
	arg.Pairs = currency.Pairs{}
	_, err = e.GetActiveOrders(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty)

	// Serial: it continues the checks above on arg and err, against e
	t.Run("spot live", func(t *testing.T) {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		arg.Pairs = currency.Pairs{spotTradablePair}
		_, err = e.GetActiveOrders(t.Context(), arg)
		require.NoError(t, err)
	})
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
	t.Run("futures live", func(t *testing.T) {
		t.Parallel()
		ex := liveExchange(t, liveReadOnly)
		result, err := ex.GetOrderInfo(t.Context(), "739106551624717312", futuresTradablePair, asset.Futures)
		require.NoError(t, answeredOnMerits(err), "GetOrderInfo must be answered on its merits")
		if err == nil && result != nil {
			assert.Equal(t, "739106551624717312", result.OrderID, "the order returned should be the one asked for")
		}
	})
	// The checks below run on e after the futures case's calls, so this one runs on an exchange not yet called
	t.Run("pair refused on an exchange not yet called", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		_, err := ex.GetOrderInfo(t.Context(), "12342", currency.EMPTYPAIR, asset.Spot)
		require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	})
	// Serial, and before the checks below: its calls on e come first, as they could when it was a test of its own, so
	// a call that kept a request for the next fails those checks
	t.Run("futures", func(t *testing.T) {
		_, err := e.GetOrderInfo(t.Context(), "", futuresTradablePair, asset.Futures)
		require.ErrorIs(t, err, order.ErrOrderIDNotSet)

		ex := testExchangeFor(t, futuresHandler(t, futuresContractsRoute, futuresRoute{
			method: http.MethodGet,
			path:   "private/order/get/739106551624717312",
			data:   futuresFilledOrderReply,
		}, futuresRoute{
			method: http.MethodGet,
			path:   "private/order/deal_details/739106551624717312",
			data: `[{"id":"9466823344","symbol":"BTC_USDT","side":1,"vol":5,"price":109777.5,"fee":0.0329,"feeCurrency":"USDT",` +
				`"profit":0,"category":1,"orderId":"739106551624717312","timestamp":1761887134000,"taker":true}]`,
		}), liveUnavailable)
		detail, err := ex.GetOrderInfo(t.Context(), "739106551624717312", futuresTradablePair, asset.Futures)
		require.NoError(t, err, "GetOrderInfo must not error for futures")
		// An opening long is a buy, its amounts are in contracts and its cost is their value in the quote currency
		exp := futuresFilledOrderDetail(ex.Name)
		exp.Trades = []order.TradeHistory{{
			Price:     109777.5,
			Amount:    5,
			Fee:       0.0329,
			Exchange:  ex.Name,
			TID:       "9466823344",
			Side:      order.Buy,
			Timestamp: time.UnixMilli(1761887134000),
			FeeAsset:  "USDT",
		}}
		assert.Equal(t, exp, detail, "the order and its fill should be mapped")
	})
	_, err := e.GetOrderInfo(t.Context(), "12342", currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	// Serial: it reuses err for its checks against e after the one above
	t.Run("spot live", func(t *testing.T) {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		// See TestGetOrderByID: the order does not exist and the tests may not create one.
		if _, err = e.GetOrderInfo(t.Context(), "12342", spotTradablePair, asset.Spot); err != nil {
			assert.ErrorContains(t, err, "Order does not exist", "the only acceptable failure is the order being absent")
		}

		// Not ErrAssetNotSet: the asset is set, it is options, which this exchange does not support.
		_, err = e.GetOrderInfo(t.Context(), "12342", spotTradablePair, asset.Options)
		assert.ErrorIs(t, err, asset.ErrNotSupported)
	})
}

// eachRiskManagementFieldAlone returns risk management settings with one field of one setting set, keyed by the
// setting and the field, so that a check of only some of the fields fails a case
func eachRiskManagementFieldAlone() map[string]order.RiskManagementModes {
	settings := make(map[string]order.RiskManagementModes)
	for setting, set := range map[string]func(*order.RiskManagementModes, order.RiskManagement){
		"take-profit": func(m *order.RiskManagementModes, r order.RiskManagement) { m.TakeProfit = r },
		"stop-loss":   func(m *order.RiskManagementModes, r order.RiskManagement) { m.StopLoss = r },
		"stop entry":  func(m *order.RiskManagementModes, r order.RiskManagement) { m.StopEntry = r },
	} {
		for field, r := range map[string]order.RiskManagement{
			"enabled":            {Enabled: true},
			"trigger price type": {TriggerPriceType: order.MarkPrice},
			"price":              {Price: 44000},
			"limit price":        {LimitPrice: 43900},
			"order type":         {OrderType: order.Limit},
		} {
			var m order.RiskManagementModes
			set(&m, r)
			settings[setting+" "+field+" alone"] = m
		}
	}
	return settings
}

// eachOrderKind returns an order of each side, order type and time in force SubmitOrder takes, keyed by its kind, so
// that a check skipping some of them fails a case
func eachOrderKind() map[string]order.Submit {
	kinds := make(map[string]order.Submit)
	for sideName, side := range map[string]order.Side{"buy": order.Buy, "sell": order.Sell, "bid": order.Bid, "ask": order.Ask, "long": order.Long, "short": order.Short} {
		for tifName, tif := range map[string]order.TimeInForce{
			"":                               order.UnknownTIF,
			"good-till-cancelled ":           order.GoodTillCancel,
			"post-only ":                     order.PostOnly,
			"good-till-cancelled post-only ": order.GoodTillCancel | order.PostOnly,
			"immediate-or-cancel ":           order.ImmediateOrCancel,
			"fill-or-kill ":                  order.FillOrKill,
		} {
			kinds[tifName+"limit "+sideName] = order.Submit{Side: side, Type: order.Limit, TimeInForce: tif, Price: 45000, Amount: 1}
		}
		for tifName, tif := range map[string]order.TimeInForce{"": order.UnknownTIF, "good-till-cancelled ": order.GoodTillCancel, "immediate-or-cancel ": order.ImmediateOrCancel} {
			kinds[tifName+"market "+sideName] = order.Submit{Side: side, Type: order.Market, TimeInForce: tif, Amount: 1}
		}
	}
	return kinds
}

// eachSideAndTypeAlone returns cancellations naming one side or one order type, each but the unknown and any ones, of
// the pair and of every pair, keyed by the side or type and the pairs, so that a check of only some of them fails a case
func eachSideAndTypeAlone(pair currency.Pair) map[string]order.Cancel {
	cancellations := make(map[string]order.Cancel)
	add := func(name string, c order.Cancel) {
		c.Pair = pair
		cancellations[name+" orders"] = c
		c.Pair = currency.EMPTYPAIR
		cancellations[name+" orders of every pair"] = c
	}
	for side := order.Buy; side <= order.MissingData; side <<= 1 {
		if side != order.AnySide {
			add(side.String(), order.Cancel{Side: side})
		}
	}
	for oType := order.Limit; oType <= order.MarketMakerProtection; oType <<= 1 {
		if oType != order.AnyType {
			add(oType.String(), order.Cancel{Type: oType})
		}
	}
	for _, oType := range []order.Type{order.StopLimit, order.StopMarket, order.TakeProfitMarket, order.TrailingStopLimit, order.Bracket} {
		add(oType.String(), order.Cancel{Type: oType})
	}
	return cancellations
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	t.Run("futures live", func(t *testing.T) {
		t.Parallel()
		ex := liveExchange(t, liveManipulatesOrders)
		ctx := context.WithoutCancel(t.Context())
		// A post-only buy far below the market cannot take an offer, so it rests or is refused
		resp, err := ex.SubmitOrder(t.Context(), &order.Submit{
			Exchange:    ex.Name,
			Pair:        futuresTradablePair,
			AssetType:   asset.Futures,
			Side:        order.Buy,
			Type:        order.Limit,
			TimeInForce: order.PostOnly,
			Price:       liveRestingBuyPrice,
			Amount:      1,
			MarginType:  margin.Isolated,
		})
		if err == nil {
			t.Cleanup(func() {
				err := ex.CancelOrder(ctx, &order.Cancel{OrderID: resp.OrderID, Pair: futuresTradablePair, AssetType: asset.Futures})
				assert.NoError(t, answeredOnMerits(err), "CancelOrder should cancel the placed order or be answered on its merits")
				if id, err := parseFuturesOrderID(resp.OrderID); assert.NoError(t, err, "the placed order's ID should parse") {
					assertLiveOrdersGone(ctx, t, ex, id)
				}
			})
		}
		require.NoError(t, answeredOnMerits(err), "SubmitOrder must be answered on its merits")
		if err == nil {
			assert.NotEmpty(t, resp.OrderID, "the placed order should carry the venue's order ID")
		}
	})
	// The checks below run on e after the futures case's calls, so this one runs on an exchange not yet called
	t.Run("nil refused on an exchange not yet called", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		_, err := ex.SubmitOrder(t.Context(), nil)
		require.ErrorIs(t, err, order.ErrSubmissionIsNil)
	})
	t.Run("spot", func(t *testing.T) {
		t.Parallel()
		type refusal struct {
			s   order.Submit
			err error
		}
		stopLoss := order.RiskManagementModes{StopLoss: order.RiskManagement{Enabled: true, Price: 44000, OrderType: order.Market}}
		refusals := map[string]refusal{
			"stop-loss": {s: order.Submit{Side: order.Buy, Type: order.Limit, Price: 45000, Amount: 0.001, RiskManagementModes: stopLoss}, err: errRiskManagementUnsupported},
			"stop-loss on a post-only order": {
				s:   order.Submit{Side: order.Buy, Type: order.Limit, TimeInForce: order.PostOnly, Price: 45000, Amount: 0.001, RiskManagementModes: stopLoss},
				err: errRiskManagementUnsupported,
			},
			"take-profit on a market sell": {s: order.Submit{
				Side: order.Sell, Type: order.Market, Amount: 0.001,
				RiskManagementModes: order.RiskManagementModes{TakeProfit: order.RiskManagement{Enabled: true, Price: 48000, TriggerPriceType: order.MarkPrice}},
			}, err: errRiskManagementUnsupported},
			"stop entry on a market buy by quote amount": {s: order.Submit{
				Side: order.Buy, Type: order.Market, QuoteAmount: 100,
				RiskManagementModes: order.RiskManagementModes{StopEntry: order.RiskManagement{Enabled: true, Price: 46000}},
			}, err: errRiskManagementUnsupported},
			"risk management mode":                               {s: order.Submit{Side: order.Buy, Type: order.Limit, Price: 45000, Amount: 0.001, RiskManagementModes: order.RiskManagementModes{Mode: "full"}}, err: errRiskManagementUnsupported},
			"slippage tolerance on a market buy by quote amount": {s: order.Submit{Side: order.Buy, Type: order.Market, QuoteAmount: 100, SlippageTolerance: 0.001}, err: errSlippageToleranceUnsupported},
			"negative slippage tolerance":                        {s: order.Submit{Side: order.Sell, Type: order.Market, Amount: 0.001, SlippageTolerance: -0.001}, err: errSlippageToleranceUnsupported},
			"slippage tolerance on a limit order":                {s: order.Submit{Side: order.Sell, Type: order.Limit, Price: 45000, Amount: 0.001, SlippageTolerance: 0.001}, err: errSlippageToleranceUnsupported},
			"slippage tolerance on a fill-or-kill order": {
				s:   order.Submit{Side: order.Sell, Type: order.Limit, TimeInForce: order.FillOrKill, Price: 45000, Amount: 0.001, SlippageTolerance: 0.001},
				err: errSlippageToleranceUnsupported,
			},
		}
		for name, modes := range eachRiskManagementFieldAlone() {
			refusals[name] = refusal{s: order.Submit{Side: order.Buy, Type: order.Limit, Price: 45000, Amount: 0.001, RiskManagementModes: modes}, err: errRiskManagementUnsupported}
		}
		endTime := time.Date(2030, time.January, 2, 3, 4, 5, 0, time.UTC)
		for kind, s := range eachOrderKind() {
			withStopLoss, withSlippage, withTriggerPrice, withEndTime := s, s, s, s
			withStopLoss.RiskManagementModes, withSlippage.SlippageTolerance = stopLoss, 0.001
			withTriggerPrice.TriggerPrice, withEndTime.EndTime = 44000, endTime
			refusals[kind+" order with a stop-loss"] = refusal{s: withStopLoss, err: errRiskManagementUnsupported}
			refusals[kind+" order with a slippage tolerance"] = refusal{s: withSlippage, err: errSlippageToleranceUnsupported}
			refusals[kind+" order with a trigger price"] = refusal{s: withTriggerPrice, err: errTriggerPriceUnsupported}
			refusals[kind+" order with an end time"] = refusal{s: withEndTime, err: errEndTimeUnsupported}
		}
		maps.Copy(refusals, map[string]refusal{
			"negative trigger price":                        {s: order.Submit{Side: order.Sell, Type: order.Market, Amount: 0.001, TriggerPrice: -44000}, err: errTriggerPriceUnsupported},
			"trigger price on the index price":              {s: order.Submit{Side: order.Buy, Type: order.Limit, Price: 45000, Amount: 0.001, TriggerPrice: 44000, TriggerPriceType: order.IndexPrice}, err: errTriggerPriceUnsupported},
			"end time at the Unix epoch":                    {s: order.Submit{Side: order.Buy, Type: order.Limit, Price: 45000, Amount: 0.001, EndTime: time.Unix(0, 0)}, err: errEndTimeUnsupported},
			"trigger price on a market buy by quote amount": {s: order.Submit{Side: order.Buy, Type: order.Market, QuoteAmount: 100, TriggerPrice: 44000}, err: errTriggerPriceUnsupported},
			"end time on a market buy by quote amount":      {s: order.Submit{Side: order.Buy, Type: order.Market, QuoteAmount: 100, EndTime: endTime}, err: errEndTimeUnsupported},
			"small trigger price on the mark price, with a client order ID and leverage 1": {
				s:   order.Submit{Side: order.Buy, Type: order.Limit, Price: 45000, Amount: 0.001, Leverage: 1, ClientOrderID: "gct-1", TriggerPrice: 0.00001, TriggerPriceType: order.MarkPrice},
				err: errTriggerPriceUnsupported,
			},
			"end time with a mark price trigger type, a client order ID and leverage 1": {
				s:   order.Submit{Side: order.Buy, Type: order.Limit, Price: 45000, Amount: 0.001, Leverage: 1, ClientOrderID: "gct-1", TriggerPriceType: order.MarkPrice, EndTime: endTime},
				err: errEndTimeUnsupported,
			},
		})
		for name, tc := range refusals {
			t.Run(name+" refused", func(t *testing.T) {
				t.Parallel()
				ex, requests := newRefusingTestExchange(t)
				tc.s.Exchange, tc.s.AssetType, tc.s.Pair = ex.Name, asset.Spot, spotTradablePair
				resp, err := ex.SubmitOrder(t.Context(), &tc.s)
				require.ErrorIs(t, err, tc.err, "SubmitOrder must refuse a setting the spot order would be placed without")
				assert.ErrorContains(t, err, "on a spot order", "the refusal should name the order's asset")
				assert.Nil(t, resp, "no order should be returned")
				assert.Zero(t, requests.Load(), "nothing should be sent to the venue")
			})
		}
	})
	// Serial, and before the checks below: its calls on e come first, as they could when it was a test of its own, so
	// a call that kept a request for the next fails those checks
	t.Run("futures", func(t *testing.T) {
		submit := func(ex *Exchange, s order.Submit) (*order.SubmitResponse, error) {
			s.Exchange, s.AssetType, s.Pair = ex.Name, asset.Futures, futuresTradablePair
			return ex.SubmitOrder(t.Context(), &s)
		}
		_, err := submit(e, order.Submit{Side: order.Buy, Type: order.Market, QuoteAmount: 100})
		require.ErrorIs(t, err, order.ErrAmountIsInvalid, "a futures order sized in quote currency must be refused")
		_, err = submit(e, order.Submit{Side: order.Buy, Type: order.Market, Amount: 1, MarginType: margin.SpotIsolated})
		require.ErrorIs(t, err, margin.ErrMarginTypeUnsupported)
		_, err = submit(e, order.Submit{Side: order.Buy, Type: order.Market, Amount: 1, Leverage: 2.5})
		require.ErrorIs(t, err, errInvalidLeverage, "fractional leverage must be refused")
		_, err = submit(e, order.Submit{Side: order.Buy, Type: order.Limit, Price: 1, Amount: 1, TimeInForce: order.GoodTillTime})
		require.ErrorIs(t, err, order.ErrUnsupportedTimeInForce)

		t.Run("isolated open takes the account's leverage", func(t *testing.T) {
			t.Parallel()
			ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
				method: http.MethodGet,
				path:   "private/position/position_mode",
				data:   "1",
			}, futuresRoute{
				method: http.MethodGet,
				path:   "private/position/leverage",
				query:  "symbol=BTC_USDT",
				data:   `[{"positionType":1,"openType":1,"leverage":20},{"positionType":2,"openType":1,"leverage":10}]`,
			}, futuresRoute{
				method: http.MethodPost,
				path:   "private/order/create",
				body:   `{"symbol":"BTC_USDT","price":50000,"vol":1,"leverage":20,"side":1,"sideType":"buy","type":1,"openType":1,"externalOid":"gct-1","positionMode":1,"stpMode":0}`,
				data:   `{"orderId":"739113577038255616","ts":1761888808839}`,
			}), liveUnavailable)
			resp, err := submit(ex, order.Submit{Side: order.Buy, Type: order.Limit, Price: 50000, Amount: 1, ClientOrderID: "gct-1"})
			require.NoError(t, err, "SubmitOrder must not error for futures")
			// An order without a margin type is isolated, at the account's leverage, and stamped with the venue's time
			exp := &order.SubmitResponse{
				Exchange:      ex.Name,
				Type:          order.Limit,
				Side:          order.Buy,
				Pair:          futuresTradablePair,
				AssetType:     asset.Futures,
				Leverage:      20,
				Price:         50000,
				Amount:        1,
				ClientOrderID: "gct-1",
				LastUpdated:   time.UnixMilli(1761888808839),
				Date:          time.UnixMilli(1761888808839),
				Status:        order.New,
				OrderID:       "739113577038255616",
				MarginType:    margin.Isolated,
			}
			assert.Equal(t, exp, resp, "the placed order should be returned")
		})

		t.Run("isolated open without leverage set", func(t *testing.T) {
			t.Parallel()
			ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
				method: http.MethodGet,
				path:   "private/position/position_mode",
				data:   "1",
			}, futuresRoute{
				method: http.MethodGet,
				path:   "private/position/leverage",
				query:  "symbol=BTC_USDT",
				data:   `[{"positionType":1,"openType":2,"leverage":20}]`,
			}), liveUnavailable)
			_, err := submit(ex, order.Submit{Side: order.Sell, Type: order.Market, Amount: 1})
			assert.ErrorIs(t, err, errInvalidLeverage, "an isolated open without leverage should be refused before it is sent")
		})

		type refusal struct {
			s   order.Submit
			err error
		}
		stopLoss := order.RiskManagementModes{StopLoss: order.RiskManagement{Enabled: true, Price: 44000, OrderType: order.Market}}
		refusals := map[string]refusal{
			"stop-loss": {s: order.Submit{Side: order.Buy, Type: order.Limit, Price: 45000, Amount: 1, Leverage: 20, RiskManagementModes: stopLoss}, err: errRiskManagementUnsupported},
			"take-profit": {s: order.Submit{
				Side: order.Buy, Type: order.Limit, Price: 45000, Amount: 1, Leverage: 20,
				RiskManagementModes: order.RiskManagementModes{TakeProfit: order.RiskManagement{Enabled: true, Price: 48000, TriggerPriceType: order.MarkPrice}},
			}, err: errRiskManagementUnsupported},
			"stop entry": {s: order.Submit{
				Side: order.Buy, Type: order.Limit, Price: 45000, Amount: 1, Leverage: 20,
				RiskManagementModes: order.RiskManagementModes{StopEntry: order.RiskManagement{Enabled: true, Price: 46000}},
			}, err: errRiskManagementUnsupported},
			"stop-loss priced but not enabled": {s: order.Submit{
				Side: order.Buy, Type: order.Limit, Price: 45000, Amount: 1, Leverage: 20,
				RiskManagementModes: order.RiskManagementModes{StopLoss: order.RiskManagement{Price: 44000}},
			}, err: errRiskManagementUnsupported},
			"risk management mode":                  {s: order.Submit{Side: order.Buy, Type: order.Limit, Price: 45000, Amount: 1, Leverage: 20, RiskManagementModes: order.RiskManagementModes{Mode: "full"}}, err: errRiskManagementUnsupported},
			"stop-loss on a market order":           {s: order.Submit{Side: order.Sell, Type: order.Market, Amount: 1, Leverage: 20, RiskManagementModes: stopLoss}, err: errRiskManagementUnsupported},
			"stop-loss on an open without leverage": {s: order.Submit{Side: order.Buy, Type: order.Market, Amount: 1, RiskManagementModes: stopLoss}, err: errRiskManagementUnsupported},
			"stop-loss on a close":                  {s: order.Submit{Side: order.Sell, Type: order.Market, Amount: 1, ReduceOnly: true, RiskManagementModes: stopLoss}, err: errRiskManagementUnsupported},
			"stop-loss on a cross order":            {s: order.Submit{Side: order.Buy, Type: order.Market, Amount: 1, MarginType: margin.Multi, RiskManagementModes: stopLoss}, err: errRiskManagementUnsupported},
			"stop-loss on an isolated order":        {s: order.Submit{Side: order.Buy, Type: order.Market, Amount: 1, Leverage: 20, MarginType: margin.Isolated, RiskManagementModes: stopLoss}, err: errRiskManagementUnsupported},
			"stop-loss on a post-only order": {
				s:   order.Submit{Side: order.Buy, Type: order.Limit, TimeInForce: order.PostOnly, Price: 45000, Amount: 1, Leverage: 20, RiskManagementModes: stopLoss},
				err: errRiskManagementUnsupported,
			},
			"slippage tolerance":                  {s: order.Submit{Side: order.Buy, Type: order.Market, Amount: 1, Leverage: 20, SlippageTolerance: 0.001}, err: errSlippageToleranceUnsupported},
			"negative slippage tolerance":         {s: order.Submit{Side: order.Buy, Type: order.Market, Amount: 1, Leverage: 20, SlippageTolerance: -0.001}, err: errSlippageToleranceUnsupported},
			"slippage tolerance on a limit order": {s: order.Submit{Side: order.Sell, Type: order.Limit, Price: 45000, Amount: 1, Leverage: 20, SlippageTolerance: 0.001}, err: errSlippageToleranceUnsupported},
			"slippage tolerance on a close":       {s: order.Submit{Side: order.Buy, Type: order.Market, Amount: 1, ReduceOnly: true, SlippageTolerance: 0.001}, err: errSlippageToleranceUnsupported},
			"slippage tolerance on a cross order": {s: order.Submit{Side: order.Sell, Type: order.Market, Amount: 1, MarginType: margin.Multi, SlippageTolerance: 0.001}, err: errSlippageToleranceUnsupported},
			"slippage tolerance on a fill-or-kill order": {
				s:   order.Submit{Side: order.Sell, Type: order.Limit, TimeInForce: order.FillOrKill, Price: 45000, Amount: 1, Leverage: 20, SlippageTolerance: 0.001},
				err: errSlippageToleranceUnsupported,
			},
			"slippage tolerance on an isolated order": {
				s:   order.Submit{Side: order.Sell, Type: order.Market, Amount: 1, Leverage: 20, MarginType: margin.Isolated, SlippageTolerance: 0.001},
				err: errSlippageToleranceUnsupported,
			},
		}
		for name, modes := range eachRiskManagementFieldAlone() {
			refusals[name] = refusal{s: order.Submit{Side: order.Buy, Type: order.Limit, Price: 45000, Amount: 1, Leverage: 20, RiskManagementModes: modes}, err: errRiskManagementUnsupported}
		}
		endTime := time.Date(2030, time.January, 2, 3, 4, 5, 0, time.UTC)
		for kind, s := range eachOrderKind() {
			withStopLoss, withSlippage, withTriggerPrice, withEndTime := s, s, s, s
			withStopLoss.RiskManagementModes, withSlippage.SlippageTolerance = stopLoss, 0.001
			withTriggerPrice.TriggerPrice, withEndTime.EndTime = 44000, endTime
			refusals[kind+" order with a stop-loss"] = refusal{s: withStopLoss, err: errRiskManagementUnsupported}
			refusals[kind+" order with a slippage tolerance"] = refusal{s: withSlippage, err: errSlippageToleranceUnsupported}
			refusals[kind+" order with a trigger price"] = refusal{s: withTriggerPrice, err: errTriggerPriceUnsupported}
			refusals[kind+" order with an end time"] = refusal{s: withEndTime, err: errEndTimeUnsupported}
		}
		maps.Copy(refusals, map[string]refusal{
			"negative trigger price":             {s: order.Submit{Side: order.Sell, Type: order.Market, Amount: 1, Leverage: 20, TriggerPrice: -44000}, err: errTriggerPriceUnsupported},
			"trigger price on the index price":   {s: order.Submit{Side: order.Buy, Type: order.Limit, Price: 45000, Amount: 1, Leverage: 20, TriggerPrice: 44000, TriggerPriceType: order.IndexPrice}, err: errTriggerPriceUnsupported},
			"trigger price on a close":           {s: order.Submit{Side: order.Sell, Type: order.Market, Amount: 1, ReduceOnly: true, TriggerPrice: 44000}, err: errTriggerPriceUnsupported},
			"trigger price on a cross order":     {s: order.Submit{Side: order.Buy, Type: order.Market, Amount: 1, MarginType: margin.Multi, TriggerPrice: 44000}, err: errTriggerPriceUnsupported},
			"trigger price on an isolated order": {s: order.Submit{Side: order.Sell, Type: order.Market, Amount: 1, Leverage: 20, MarginType: margin.Isolated, TriggerPrice: 44000}, err: errTriggerPriceUnsupported},
			"end time at the Unix epoch":         {s: order.Submit{Side: order.Buy, Type: order.Limit, Price: 45000, Amount: 1, Leverage: 20, EndTime: time.Unix(0, 0)}, err: errEndTimeUnsupported},
			"end time on a close":                {s: order.Submit{Side: order.Buy, Type: order.Limit, Price: 45000, Amount: 1, ReduceOnly: true, EndTime: endTime}, err: errEndTimeUnsupported},
			"end time on a cross order":          {s: order.Submit{Side: order.Sell, Type: order.Limit, Price: 45000, Amount: 1, MarginType: margin.Multi, EndTime: endTime}, err: errEndTimeUnsupported},
			"end time on an isolated order":      {s: order.Submit{Side: order.Buy, Type: order.Limit, Price: 45000, Amount: 1, Leverage: 20, MarginType: margin.Isolated, EndTime: endTime}, err: errEndTimeUnsupported},
			"small trigger price on the mark price, with a client order ID and leverage 1": {
				s:   order.Submit{Side: order.Buy, Type: order.Market, Amount: 1, Leverage: 1, ClientOrderID: "gct-1", TriggerPrice: 0.00001, TriggerPriceType: order.MarkPrice},
				err: errTriggerPriceUnsupported,
			},
			"end time with a mark price trigger type, a client order ID and leverage 1": {
				s:   order.Submit{Side: order.Buy, Type: order.Limit, Price: 45000, Amount: 1, Leverage: 1, ClientOrderID: "gct-1", TriggerPriceType: order.MarkPrice, EndTime: endTime},
				err: errEndTimeUnsupported,
			},
		})
		for name, tc := range refusals {
			t.Run(name+" refused", func(t *testing.T) {
				t.Parallel()
				ex, requests := newRefusingTestExchange(t)
				resp, err := submit(ex, tc.s)
				require.ErrorIs(t, err, tc.err, "SubmitOrder must refuse a setting the futures order would be placed without")
				assert.ErrorContains(t, err, "on a futures order", "the refusal should name the order's asset")
				assert.Nil(t, resp, "no order should be returned")
				assert.Zero(t, requests.Load(), "nothing should be sent to the venue")
			})
		}

		// A client order ID the venue can't take is refused before anything is sent, the position mode's read included,
		// whether the order opens or closes
		for _, reduceOnly := range []bool{false, true} {
			t.Run("overlong client order ID with reduce-only "+strconv.FormatBool(reduceOnly)+" refused", func(t *testing.T) {
				t.Parallel()
				ex, requests := newRefusingTestExchange(t)
				clientOrderID := strings.Repeat("g", futuresExternalOrderIDLimit+1)
				resp, err := submit(ex, order.Submit{Side: order.Buy, Type: order.Market, Amount: 1, MarginType: margin.Multi, ReduceOnly: reduceOnly, ClientOrderID: clientOrderID})
				require.ErrorIs(t, err, errExternalOrderIDTooLong, "SubmitOrder must refuse a client order ID longer than the venue takes")
				assert.Nil(t, resp, "no order should be returned")
				assert.Zero(t, requests.Load(), "nothing should be sent to the venue")
			})
		}
		t.Run("longest client order ID placed", func(t *testing.T) {
			t.Parallel()
			clientOrderID := strings.Repeat("g", futuresExternalOrderIDLimit)
			ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
				method: http.MethodGet,
				path:   "private/position/position_mode",
				data:   "1",
			}, futuresRoute{
				method: http.MethodPost,
				path:   "private/order/create",
				body:   `{"symbol":"BTC_USDT","vol":1,"side":1,"sideType":"buy","type":5,"openType":2,"externalOid":"` + clientOrderID + `","positionMode":1,"stpMode":0}`,
				data:   `{"orderId":"739113577038255616","ts":1761888808839}`,
			}), liveUnavailable)
			resp, err := submit(ex, order.Submit{Side: order.Buy, Type: order.Market, Amount: 1, MarginType: margin.Multi, ClientOrderID: clientOrderID})
			require.NoError(t, err, "SubmitOrder must place an order whose client order ID is as long as the venue takes")
			assert.Equal(t, clientOrderID, resp.ClientOrderID, "the placed order should carry its client order ID")
		})

		// A close names the position it closes, when one is open, and takes that position's margin mode when it names none.
		// It carries the account's position mode, and on a one-way account it is also sent reduce-only
		isolatedLong := futuresOpenPositionReply
		crossLong := strings.NewReplacer(`"positionId":1109973831`, `"positionId":1109973832`, `"openType":1`, `"openType":2`).Replace(futuresOpenPositionReply)
		for name, tc := range map[string]struct {
			positionMode string
			positions    string
			marginType   margin.Type
			body         string
			exp          margin.Type
			err          error
		}{
			"cross close": {
				positionMode: "1", positions: "[" + isolatedLong + "," + crossLong + "]", marginType: margin.Multi,
				body: `{"symbol":"BTC_USDT","vol":2,"side":4,"sideType":"sell","type":5,"openType":2,"positionId":1109973832,"positionMode":1,"stpMode":0}`, exp: margin.Multi,
			},
			"close of the position's margin mode": {
				positionMode: "1", positions: "[" + crossLong + "]",
				body: `{"symbol":"BTC_USDT","vol":2,"side":4,"sideType":"sell","type":5,"openType":2,"positionId":1109973832,"positionMode":1,"stpMode":0}`, exp: margin.Multi,
			},
			"close without a position": {
				positionMode: "1", positions: "[]",
				body: `{"symbol":"BTC_USDT","vol":2,"side":4,"sideType":"sell","type":5,"openType":1,"positionMode":1,"stpMode":0}`, exp: margin.Isolated,
			},
			"close of either margin mode": {positionMode: "1", positions: "[" + isolatedLong + "," + crossLong + "]", err: errMarginTypeAmbiguous},
			"one-way cross close": {
				positionMode: "2", positions: "[" + crossLong + "]", marginType: margin.Multi,
				body: `{"symbol":"BTC_USDT","vol":2,"side":4,"sideType":"sell","type":5,"openType":2,"positionId":1109973832,"positionMode":2,"reduceOnly":true,"stpMode":0}`,
				exp:  margin.Multi,
			},
			"one-way close without a position": {
				positionMode: "2", positions: "[]",
				body: `{"symbol":"BTC_USDT","vol":2,"side":4,"sideType":"sell","type":5,"openType":1,"positionMode":2,"reduceOnly":true,"stpMode":0}`,
				exp:  margin.Isolated,
			},
		} {
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				routes := []futuresRoute{
					{method: http.MethodGet, path: "private/position/position_mode", data: tc.positionMode},
					{method: http.MethodGet, path: "private/position/open_positions", query: "symbol=BTC_USDT", data: tc.positions},
				}
				if tc.body != "" {
					routes = append(routes, futuresRoute{method: http.MethodPost, path: "private/order/create", body: tc.body, data: `{"orderId":"739113577038255617","ts":1761888808840}`})
				}
				ex := testExchangeFor(t, futuresHandler(t, routes...), liveUnavailable)
				resp, err := submit(ex, order.Submit{Side: order.Sell, Type: order.Market, Amount: 2, ReduceOnly: true, MarginType: tc.marginType})
				require.ErrorIs(t, err, tc.err, "SubmitOrder must return the expected error for a closing order")
				if tc.err != nil {
					return
				}
				// A closing order is returned as reduce-only, and sends no leverage
				exp := &order.SubmitResponse{
					Exchange:    ex.Name,
					Type:        order.Market,
					Side:        order.Sell,
					Pair:        futuresTradablePair,
					AssetType:   asset.Futures,
					ReduceOnly:  true,
					Amount:      2,
					LastUpdated: time.UnixMilli(1761888808840),
					Date:        time.UnixMilli(1761888808840),
					Status:      order.New,
					OrderID:     "739113577038255617",
					MarginType:  tc.exp,
				}
				assert.Equal(t, exp, resp, "the closing order should be returned with the closed position's margin type")
			})
		}
	})
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

	// Serial: it continues the checks above on arg and err, against e
	t.Run("spot live", func(t *testing.T) {
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
	})
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	t.Run("futures live", func(t *testing.T) {
		t.Parallel()
		ex := liveExchange(t, liveManipulatesOrders)
		err := ex.CancelOrder(t.Context(), &order.Cancel{OrderID: "739106551624717312", AssetType: asset.Futures})
		require.NoError(t, answeredOnMerits(err), "CancelOrder must be answered on its merits")
	})
	// The checks below run on e after the futures case's calls, so this one runs on an exchange not yet called
	t.Run("nil refused on an exchange not yet called", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		require.ErrorIs(t, ex.CancelOrder(t.Context(), nil), order.ErrCancelOrderIsNil)
	})
	// Serial, and before the checks below: its calls on e come first, as they could when it was a test of its own, so
	// a call that kept a request for the next fails those checks
	t.Run("futures", func(t *testing.T) {
		cancel := func(ex *Exchange, c order.Cancel) error {
			c.AssetType = asset.Futures
			return ex.CancelOrder(t.Context(), &c)
		}
		require.ErrorIs(t, cancel(e, order.Cancel{}), order.ErrOrderIDNotSet)
		require.ErrorIs(t, cancel(e, order.Cancel{ClientOrderID: "gct-1"}), currency.ErrCurrencyPairEmpty, "a client order ID must be named with its pair")
		require.ErrorIs(t, cancel(e, order.Cancel{OrderID: "abc"}), errInvalidFuturesID)

		const orderID = "739106551624717312"
		for name, tc := range map[string]struct {
			data string
			err  error
		}{
			"cancelled": {data: `[{"orderId":739106551624717312,"errorCode":0,"errorMsg":"success"}]`},
			"refused":   {data: `[{"orderId":739106551624717312,"errorCode":2040,"errorMsg":"order not exist"}]`, err: errFuturesRequestFailed},
			"missing":   {data: `[]`, err: errCancelOutcomeMissing},
		} {
			// The venue's echo of an order ID requested with leading zeros drops them, and still names the order
			for _, requested := range []string{orderID, "000" + orderID} {
				t.Run(name+" "+requested, func(t *testing.T) {
					t.Parallel()
					var sent []byte
					replay := futuresHandler(t, futuresRoute{method: http.MethodPost, path: "private/order/cancel", body: `[739106551624717312]`, data: tc.data})
					ex := testExchangeFor(t, func(w http.ResponseWriter, r *http.Request) {
						var err error
						sent, err = io.ReadAll(r.Body)
						assert.NoError(t, err, "reading the request body should not error")
						r.Body = io.NopCloser(bytes.NewReader(sent))
						replay(w, r)
					}, liveUnavailable)
					err := cancel(ex, order.Cancel{OrderID: requested})
					require.ErrorIsf(t, err, tc.err, "the outcome of cancelling order %s must be returned", requested)
					assert.Equalf(t, "["+orderID+"]", string(sent), "order %s should be sent exactly, as a number", requested)
				})
			}
		}

		for data, exp := range map[string]error{
			`[{"orderId":739106551624717312,"errorCode":0,"errorMsg":"success"}]`:            nil,
			`{"orderId":739106551624717312,"errorCode":0,"errorMsg":"success"}`:              nil,
			`[{"orderId":739106551624717312,"errorCode":2040,"errorMsg":"order not exist"}]`: errFuturesRequestFailed,
			`[]`:   errCancelOutcomeMissing,
			`null`: errCancelOutcomeMissing,
		} {
			t.Run("by client order ID answered "+data, func(t *testing.T) {
				t.Parallel()
				ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
					method: http.MethodPost,
					path:   "private/order/cancel_with_external",
					body:   `{"symbol":"BTC_USDT","externalOid":"gct-1"}`,
					data:   data,
				}), liveUnavailable)
				assert.ErrorIsf(t, cancel(ex, order.Cancel{ClientOrderID: "gct-1", Pair: futuresTradablePair}), exp, "a cancellation by client order ID answered %s should report its outcome", data)
			})
		}
	})
	err := e.CancelOrder(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrCancelOrderIsNil)

	err = e.CancelOrder(t.Context(), &order.Cancel{})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	err = e.CancelOrder(t.Context(), &order.Cancel{OrderID: "987654"})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	err = e.CancelOrder(t.Context(), &order.Cancel{OrderID: "987654", AssetType: asset.Spot})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	// Serial: it reuses err for its checks against e after the ones above
	t.Run("spot live", func(t *testing.T) {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		err = e.CancelOrder(t.Context(), &order.Cancel{OrderID: "987654", Pair: spotTradablePair, AssetType: asset.Spot})
		assert.NoError(t, err)

		err = e.CancelOrder(t.Context(), &order.Cancel{OrderID: "987654", Pair: spotTradablePair, AssetType: asset.Options})
		assert.ErrorIs(t, err, asset.ErrNotSupported)
	})
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	t.Run("futures live", func(t *testing.T) {
		t.Parallel()
		ex := liveExchange(t, liveManipulatesOrders)
		_, err := ex.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Futures, Pair: futuresTradablePair})
		require.NoError(t, answeredOnMerits(err), "CancelAllOrders must be answered on its merits")
	})
	t.Run("futures", func(t *testing.T) {
		t.Parallel()
		for name, tc := range map[string]struct {
			pair currency.Pair
			body string
		}{
			"contract":  {pair: futuresTradablePair, body: `{"symbol":"BTC_USDT"}`},
			"account":   {body: `{"symbol":""}`},
			"formatted": {pair: currency.NewBTCUSDT(), body: `{"symbol":"BTC_USDT"}`},
		} {
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				handler, requests := countRequests(futuresHandler(t, futuresRoute{method: http.MethodPost, path: "private/order/cancel_all", body: tc.body}))
				ex := testExchangeFor(t, handler, liveUnavailable)
				resp, err := ex.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Futures, Pair: tc.pair})
				require.NoError(t, err, "CancelAllOrders must not error for futures")
				assert.Equal(t, order.CancelAllResponse{Status: map[string]string{}}, resp, "the venue names no orders, so none should be reported")
				assert.EqualValues(t, 1, requests.Load(), "the cancel-all should be sent once")
			})
		}
		for name, c := range map[string]order.Cancel{
			"any side and type":  {Side: order.AnySide, Type: order.AnyType},
			"any side":           {Side: order.AnySide},
			"any type":           {Type: order.AnyType},
			"any side, one pair": {Side: order.AnySide, Pair: currency.NewBTCUSDT()},
		} {
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				handler, requests := countRequests(futuresHandler(t, futuresRoute{method: http.MethodPost, path: "private/order/cancel_all", body: `{"symbol":"BTC_USDT"}`}))
				ex := testExchangeFor(t, handler, liveUnavailable)
				if c.Pair.IsEmpty() {
					c.Pair = futuresTradablePair
				}
				c.AssetType = asset.Futures
				resp, err := ex.CancelAllOrders(t.Context(), &c)
				require.NoError(t, err, "CancelAllOrders must cancel the contract's orders of every side and type")
				assert.Equal(t, order.CancelAllResponse{Status: map[string]string{}}, resp, "the venue names no orders, so none should be reported")
				assert.EqualValues(t, 1, requests.Load(), "the cancel-all should be sent once")
			})
		}
		refusals := map[string]order.Cancel{
			"buy limit orders":         {Side: order.Buy, Type: order.Limit, Pair: futuresTradablePair},
			"limit orders of any side": {Side: order.AnySide, Type: order.Limit, Pair: futuresTradablePair},
			"buys of any type":         {Side: order.Buy, Type: order.AnyType, Pair: futuresTradablePair},
			"every contract's sells":   {Side: order.Sell},
			"every contract's limits":  {Type: order.Limit},
		}
		maps.Copy(refusals, eachSideAndTypeAlone(futuresTradablePair))
		for name, c := range refusals {
			t.Run(name+" refused", func(t *testing.T) {
				t.Parallel()
				ex, requests := newRefusingTestExchange(t)
				c.AssetType = asset.Futures
				_, err := ex.CancelAllOrders(t.Context(), &c)
				require.ErrorIs(t, err, common.ErrFunctionNotSupported, "CancelAllOrders must refuse to cancel futures orders of one side or type")
				assert.Zero(t, requests.Load(), "nothing should be sent to the venue")
			})
		}
	})
	t.Run("spot", func(t *testing.T) {
		t.Parallel()
		for name, tc := range map[string]struct {
			cancel order.Cancel
			// sent is the cancellation's method, path and symbol
			sent string
			exp  map[string]string
		}{
			"any side and type":      {cancel: order.Cancel{Side: order.AnySide, Type: order.AnyType, Pair: spotTradablePair}, sent: "DELETE /api/v3/openOrders BTCUSDT", exp: map[string]string{"9": "cancelled"}},
			"any side":               {cancel: order.Cancel{Side: order.AnySide, Pair: spotTradablePair}, sent: "DELETE /api/v3/openOrders BTCUSDT", exp: map[string]string{"9": "cancelled"}},
			"any type":               {cancel: order.Cancel{Type: order.AnyType, Pair: spotTradablePair}, sent: "DELETE /api/v3/openOrders BTCUSDT", exp: map[string]string{"9": "cancelled"}},
			"any side, every symbol": {cancel: order.Cancel{Side: order.AnySide}, sent: "DELETE /api/v3/order/all ", exp: map[string]string{}},
			"any type, every symbol": {cancel: order.Cancel{Type: order.AnyType}, sent: "DELETE /api/v3/order/all ", exp: map[string]string{}},
		} {
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				var sent []string
				ex := testExchangeFor(t, func(w http.ResponseWriter, r *http.Request) {
					sent = append(sent, r.Method+" "+r.URL.Path+" "+r.URL.Query().Get("symbol"))
					if r.URL.Path == "/api/v3/openOrders" {
						_, _ = w.Write([]byte(`[{"symbol":"BTCUSDT","orderId":"9"}]`))
						return
					}
					_, _ = w.Write([]byte(`{"code":200,"msg":"success","timestamp":1778744778528}`))
				}, liveUnavailable)
				tc.cancel.AssetType = asset.Spot
				resp, err := ex.CancelAllOrders(t.Context(), &tc.cancel)
				require.NoError(t, err, "CancelAllOrders must cancel the spot orders of every side and type")
				assert.Equal(t, order.CancelAllResponse{Status: tc.exp}, resp, "the orders the venue names should be reported cancelled")
				assert.Equal(t, []string{tc.sent}, sent, "the venue's cancel-all should be sent once")
			})
		}
		refusals := map[string]order.Cancel{
			"sell market orders":        {Side: order.Sell, Type: order.Market, Pair: spotTradablePair},
			"limit orders of any side":  {Side: order.AnySide, Type: order.Limit, Pair: spotTradablePair},
			"buys of any type":          {Side: order.Buy, Type: order.AnyType, Pair: spotTradablePair},
			"every symbol's sells":      {Side: order.Sell},
			"every symbol's limits":     {Type: order.Limit},
			"every symbol's ask orders": {Side: order.Ask, Type: order.AnyType},
		}
		maps.Copy(refusals, eachSideAndTypeAlone(spotTradablePair))
		for name, c := range refusals {
			t.Run(name+" refused", func(t *testing.T) {
				t.Parallel()
				ex, requests := newRefusingTestExchange(t)
				c.AssetType = asset.Spot
				_, err := ex.CancelAllOrders(t.Context(), &c)
				require.ErrorIs(t, err, common.ErrFunctionNotSupported, "CancelAllOrders must refuse to cancel spot orders of one side or type")
				assert.Zero(t, requests.Load(), "nothing should be sent to the venue")
			})
		}
	})
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
	// Serial: it cancels through e after the checks above, in their order
	t.Run("spot live", func(t *testing.T) {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		result, err := e.CancelAllOrders(t.Context(), &order.Cancel{OrderID: "12345", AssetType: asset.Spot, Pair: spotTradablePair})
		require.NoError(t, err)
		assert.NotNil(t, result)

		_, err = e.CancelAllOrders(t.Context(), &order.Cancel{OrderID: "12345", AssetType: asset.Options, Pair: spotTradablePair})
		require.ErrorIs(t, err, asset.ErrNotSupported)
	})
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

	// The statements after the loop also call SendFuturesHTTPRequest, so the cases run before them, in order
	for reply, exp := range map[string]struct {
		err     error
		message string
	}{
		`{"success":false,"code":1001,"message":"contract not activated"}`: {errFuturesRequestFailed, "code 1001: contract not activated"},
		`{"success":false,"code":600,"msg":"parameter error"}`:             {errFuturesRequestFailed, "code 600: parameter error"},
		// Success false is a refusal whatever its code, and a reply without success is refused only by a code which
		// is neither 0 nor 200
		`{"success":false,"code":0,"message":"refused"}`: {errFuturesRequestFailed, "code 0: refused"},
		`{"success":false,"code":200,"msg":"refused"}`:   {errFuturesRequestFailed, "code 200: refused"},
		`{"success":false}`:                              {errFuturesRequestFailed, "code 0: "},
		`{"code":500,"message":"system error"}`:          {errFuturesRequestFailed, "code 500: system error"},
		`{"success":true,"code":200,"msg":"success"}`:    {},
		`{"success":true,"code":0}`:                      {},
		`{"code":0}`:                                     {},
		`{"code":200,"msg":"success"}`:                   {},
	} {
		t.Run(reply, func(t *testing.T) {
			ex := newPrivateTestExchange(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(reply)) })
			err := ex.SendFuturesHTTPRequest(t.Context(), futuresServerTimeEPL, http.MethodGet, "contract/ping", nil, nil, nil, false)
			assert.ErrorIsf(t, err, exp.err, "the reply %s should be read", reply)
			if exp.err != nil {
				assert.ErrorContainsf(t, err, exp.message, "the refusal %s should carry its code and message", reply)
			}
		})
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
