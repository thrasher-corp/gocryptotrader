package gateio

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Please supply your own APIKEYS here for due diligence testing

const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var (
	e                *Exchange
	enabledAssetPair map[asset.Item]currency.Pair
)

func getTime() (startTime, endTime time.Time) {
	if mockTests {
		return time.UnixMilli(1744103854944), time.UnixMilli(1744190254944)
	}
	return time.Now().Add(-time.Hour * 48), time.Now()
}

func getTimeWithInterval(interval kline.Interval) (startTime, endTime time.Time) {
	if mockTests {
		startTime = time.UnixMilli(1744103854944)
		return startTime, startTime.Add(time.Duration(interval))
	}
	startTime = time.Now()
	return startTime.Add(-time.Hour * 48), startTime
}

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, e)
}

func TestCancelAllExchangeOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrCancelOrderIsNil)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	r := &order.Cancel{OrderID: "1", AccountID: "1"}
	for _, a := range e.GetAssetTypes(false) {
		r.AssetType = a
		r.Pair = currency.EMPTYPAIR
		_, err = e.CancelAllOrders(t.Context(), r)
		assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

		r.Pair = getPair(t, a)
		_, err = e.CancelAllOrders(t.Context(), r)
		require.NoError(t, err)
	}
}

func TestGetAccountBalances(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	for _, a := range e.GetAssetTypes(false) {
		_, err := e.UpdateAccountBalances(t.Context(), a)
		assert.NoErrorf(t, err, "UpdateAccountBalances should not error for asset %s", a)
	}
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	cryptocurrencyChains, err := e.GetAvailableTransferChains(t.Context(), currency.BTC)
	require.NoError(t, err, "GetAvailableTransferChains must not error")
	require.NotEmpty(t, cryptocurrencyChains, "GetAvailableTransferChains must return some chains")
	withdrawCryptoRequest := withdraw.Request{
		Exchange:    e.Name,
		Amount:      -0.1,
		Currency:    currency.BTC,
		Description: "WITHDRAW IT ALL",
		Crypto: withdraw.CryptoRequest{
			Address: core.BitcoinDonationAddress,
			Chain:   cryptocurrencyChains[0],
		},
	}
	_, err = e.WithdrawCryptocurrencyFunds(t.Context(), &withdrawCryptoRequest)
	require.NoError(t, err)
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	for _, a := range e.GetAssetTypes(false) {
		_, err := e.GetOrderInfo(t.Context(), "917591554", getPair(t, a), a)
		require.NoErrorf(t, err, "GetOrderInfo must not error for asset %s", a)
	}
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	for _, a := range e.GetAssetTypes(false) {
		_, err := e.UpdateTicker(t.Context(), getPair(t, a), a)
		assert.NoErrorf(t, err, "UpdateTicker should not error for %s", a)
	}
}

func TestListSpotCurrencies(t *testing.T) {
	t.Parallel()
	_, err := e.ListSpotCurrencies(t.Context())
	assert.NoError(t, err)
}

func TestGetCurrencyDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetCurrencyDetail(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	result, err := e.GetCurrencyDetail(t.Context(), currency.BTC)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestListAllCurrencyPairs(t *testing.T) {
	t.Parallel()
	result, err := e.ListSpotCurrencyPairs(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrencyPairDetal(t *testing.T) {
	t.Parallel()
	_, err := e.GetCurrencyPairDetail(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	result, err := e.GetCurrencyPairDetail(t.Context(), getPair(t, asset.Spot))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTickers(t *testing.T) {
	t.Parallel()
	_, err := e.GetTickers(t.Context(), getPair(t, asset.Spot), "")
	assert.NoError(t, err)
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := e.GetTicker(t.Context(), currency.EMPTYPAIR, utc8TimeZone)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	result, err := e.GetTicker(t.Context(), getPair(t, asset.Spot), utc8TimeZone)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderbook(t.Context(), currency.EMPTYPAIR, "0.1", 10, false)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	result, err := e.GetOrderbook(t.Context(), getPair(t, asset.Spot), "0.1", 10, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarketTrades(t *testing.T) {
	t.Parallel()
	result, err := e.GetMarketTrades(t.Context(), getPair(t, asset.Spot), 0, "", true, time.Time{}, time.Time{}, 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCandlestickUnmarshalJSON(t *testing.T) {
	t.Parallel()
	data := []byte(`[["1738108800","229534412.73508700","103734.3","104779.9","101336.6","101343.8","2232.94510000","true"],["1738195200","178316032.62306100","104718.6","106467.1","103286.4","103734.4","1695.00787000","true"],["1738281600","231315376.16747100","102431","106042.7","101555.9","104718.6","2228.03609000","true"]]`)
	var targets []Candlestick
	err := json.Unmarshal(data, &targets)
	require.NoError(t, err)
	require.Len(t, targets, 3)
	assert.Equal(t, Candlestick{
		Timestamp:      types.Time(time.Unix(1738108800, 0)),
		QuoteCcyVolume: 229534412.73508700,
		ClosePrice:     103734.3,
		HighestPrice:   104779.9,
		LowestPrice:    101336.6,
		OpenPrice:      101343.8,
		BaseCcyAmount:  2232.94510000,
		WindowClosed:   true,
	}, targets[0])
}

func TestGetCandlesticks(t *testing.T) {
	t.Parallel()
	result, err := e.GetCandlesticks(t.Context(), getPair(t, asset.Spot), 0, time.Time{}, time.Time{}, kline.OneDay)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTradingFeeRatio(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetTradingFeeRatio(t.Context(), getPair(t, asset.Spot))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountBatchFeeRates(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAccountBatchFeeRates(t.Context(), []string{"BTC_USDT", "ETH_USDT"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSpotAccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSpotAccounts(t.Context(), currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateBatchOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CreateBatchOrders(t.Context(), make([]CreateOrderRequest, 11))
	require.ErrorIs(t, err, errMultipleOrders)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	arg := CreateOrderRequest{
		Side:    order.Sell,
		Amount:  0.001,
		Price:   12349,
		Account: asset.Spot,
		Type:    "limit",
	}
	_, err = e.CreateBatchOrders(t.Context(), []CreateOrderRequest{arg})
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet)

	arg.Text = "1234"
	_, err = e.CreateBatchOrders(t.Context(), []CreateOrderRequest{arg})
	require.ErrorIs(t, err, errInvalidOrderText)

	arg.Text = "t-1234"
	_, err = e.CreateBatchOrders(t.Context(), []CreateOrderRequest{arg})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.CreateBatchOrders(t.Context(), []CreateOrderRequest{
		{
			Text:         "t-123123123",
			CurrencyPair: getPair(t, asset.Spot),
			Side:         order.Sell,
			Amount:       0.001,
			Price:        12349,
			Account:      asset.Spot,
			Type:         "limit",
		},
		{
			Text:         "t-123123124",
			CurrencyPair: getPair(t, asset.Spot),
			Side:         order.Buy,
			Amount:       1,
			Price:        1234567789,
			Account:      asset.Spot,
			Type:         "limit",
		},
	})
	assert.NoError(t, err)
}

func TestGetSpotOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSpotOpenOrders(t.Context(), 0, 0, false)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSpotClosePositionWhenCrossCurrencyDisabled(t *testing.T) {
	t.Parallel()
	_, err := e.SpotClosePositionWhenCrossCurrencyDisabled(t.Context(), &ClosePositionRequestParam{})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.SpotClosePositionWhenCrossCurrencyDisabled(t.Context(), &ClosePositionRequestParam{CurrencyPair: getPair(t, asset.Spot)})
	assert.ErrorIs(t, err, limits.ErrAmountBelowMin)

	_, err = e.SpotClosePositionWhenCrossCurrencyDisabled(t.Context(), &ClosePositionRequestParam{Amount: 0.1, CurrencyPair: getPair(t, asset.Spot)})
	assert.ErrorIs(t, err, limits.ErrPriceBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SpotClosePositionWhenCrossCurrencyDisabled(t.Context(), &ClosePositionRequestParam{
		Amount:       0.1,
		Price:        1234567384,
		CurrencyPair: getPair(t, asset.Spot),
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateSpotOrder(t *testing.T) {
	t.Parallel()
	arg := &CreateOrderRequest{Type: "limit"}
	_, err := e.PlaceSpotOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	arg.CurrencyPair = getPair(t, asset.Spot)
	_, err = e.PlaceSpotOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = order.Buy
	_, err = e.PlaceSpotOrder(t.Context(), arg)
	require.ErrorIs(t, err, asset.ErrInvalidAsset)

	arg.Account = asset.Spot
	_, err = e.PlaceSpotOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	arg.Amount = 1
	arg.Price = -1
	_, err = e.PlaceSpotOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.PlaceSpotOrder(t.Context(), &CreateOrderRequest{
		CurrencyPair: getPair(t, asset.Spot),
		Side:         order.Buy,
		Amount:       1,
		Price:        900000,
		Account:      asset.Spot,
		Type:         "limit",
	})
	assert.NoError(t, err)
}

func TestGetSpotOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotOrders(t.Context(), currency.EMPTYPAIR, statusOpen, 0, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSpotOrders(t.Context(), getPair(t, asset.Spot), statusOpen, 100, 100)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllOpenOrdersSpecifiedCurrencyPair(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllOpenOrdersSpecifiedCurrencyPair(t.Context(), currency.EMPTYPAIR, order.Sell, asset.Empty)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.CancelAllOpenOrdersSpecifiedCurrencyPair(t.Context(), getPair(t, asset.Spot), order.Sell, asset.Spot)
	assert.NoError(t, err)
}

func TestCancelBatchOrdersWithIDList(t *testing.T) {
	t.Parallel()
	_, err := e.CancelBatchOrdersWithIDList(t.Context(), []CancelOrderByIDParam{})
	require.ErrorIs(t, err, errNoValidParameterPassed)

	_, err = e.CancelBatchOrdersWithIDList(t.Context(), []CancelOrderByIDParam{{CurrencyPair: getPair(t, asset.Spot)}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.CancelBatchOrdersWithIDList(t.Context(), []CancelOrderByIDParam{
		{
			CurrencyPair: getPair(t, asset.Spot),
			ID:           "1234567",
		},
		{
			CurrencyPair: getPair(t, asset.Spot),
			ID:           "123453",
		},
	})
	assert.NoError(t, err)
}

func TestGetSpotOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotOrder(t.Context(), "", getPair(t, asset.Spot), asset.Spot)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	_, err = e.GetSpotOrder(t.Context(), "1234", currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetSpotOrder(t.Context(), "1234", getPair(t, asset.Spot), asset.Spot)
	assert.NoError(t, err)
}

func TestAmendSpotOrder(t *testing.T) {
	t.Parallel()
	_, err := e.AmendSpotOrder(t.Context(), "", getPair(t, asset.Spot), false, &PriceAndAmount{Price: 1000})
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet)

	_, err = e.AmendSpotOrder(t.Context(), "123", currency.EMPTYPAIR, false, &PriceAndAmount{Price: 1000})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.AmendSpotOrder(t.Context(), "123", getPair(t, asset.Spot), false, &PriceAndAmount{})
	assert.ErrorContains(t, err, "either price or amount has to be set")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.AmendSpotOrder(t.Context(), "123", getPair(t, asset.Spot), false, &PriceAndAmount{
		Price: 1000,
	})
	assert.NoError(t, err)
}

func TestCancelSingleSpotOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.CancelSingleSpotOrder(t.Context(), "1234", getPair(t, asset.Spot), false)
	assert.NoError(t, err)
}

func TestGetMySpotTradingHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetMySpotTradingHistory(t.Context(), getPair(t, asset.Spot), "", 0, 0, false, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetMySpotTradingHistory(t.Context(), getPair(t, asset.Spot), "987345678234", 1, 100, false, startTime, endTime)
	require.NoError(t, err)
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	_, err := e.GetServerTime(t.Context(), asset.Spot)
	assert.NoError(t, err)
}

func TestCountdownCancelorder(t *testing.T) {
	t.Parallel()
	_, err := e.CountdownCancelorders(t.Context(), CountdownCancelOrderParam{Timeout: 0, CurrencyPair: getPair(t, asset.Spot)})
	require.ErrorIs(t, err, errInvalidCountdown)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.CountdownCancelorders(t.Context(), CountdownCancelOrderParam{Timeout: 10, CurrencyPair: getPair(t, asset.Spot)})
	assert.NoError(t, err)
}

func TestCreatePriceTriggeredOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CreatePriceTriggeredOrder(t.Context(), &PriceTriggeredOrderParam{})
	require.ErrorIs(t, err, order.ErrInvalidTimeInForce)

	_, err = e.CreatePriceTriggeredOrder(t.Context(), &PriceTriggeredOrderParam{
		Put: PutOrderData{
			TimeInForce: "gtc",
		},
	})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.CreatePriceTriggeredOrder(t.Context(), &PriceTriggeredOrderParam{
		Symbol: getPair(t, asset.Spot),
		Put:    PutOrderData{TimeInForce: "GTC"}, Trigger: TriggerPriceInfo{
			Price: -1,
		},
	})
	require.ErrorIs(t, err, limits.ErrPriceBelowMin)

	_, err = e.CreatePriceTriggeredOrder(t.Context(), &PriceTriggeredOrderParam{
		Symbol:  getPair(t, asset.Spot),
		Put:     PutOrderData{TimeInForce: "GTC"},
		Trigger: TriggerPriceInfo{Price: 1},
	})
	require.ErrorContains(t, err, "invalid price trigger condition or rule")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.CreatePriceTriggeredOrder(t.Context(), &PriceTriggeredOrderParam{
		Trigger: TriggerPriceInfo{
			Price:      123,
			Rule:       ">=",
			Expiration: 3600,
		},
		Put: PutOrderData{
			Type:        "limit",
			Side:        "sell",
			Price:       2312312,
			Amount:      30,
			TimeInForce: "gtc",
		},
		Symbol: getPair(t, asset.Spot),
	})
	assert.NoError(t, err)
}

func TestGetPriceTriggeredOrderList(t *testing.T) {
	t.Parallel()
	_, err := e.GetPriceTriggeredOrderList(t.Context(), "", currency.EMPTYPAIR, asset.Empty, 0, 0)
	require.ErrorIs(t, err, errInvalidOrderStatus)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetPriceTriggeredOrderList(t.Context(), statusOpen, currency.EMPTYPAIR, asset.Empty, 0, 0)
	assert.NoError(t, err)
}

func TestCancelAllOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelMultipleSpotOpenOrders(t.Context(), currency.EMPTYPAIR, asset.Empty)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.CancelMultipleSpotOpenOrders(t.Context(), currency.EMPTYPAIR, asset.CrossMargin)
	assert.NoError(t, err)

	_, err = e.CancelMultipleSpotOpenOrders(t.Context(), currency.EMPTYPAIR, asset.Spot)
	assert.NoError(t, err)
}

func TestGetSinglePriceTriggeredOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetSinglePriceTriggeredOrder(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetSinglePriceTriggeredOrder(t.Context(), "1234")
	assert.NoError(t, err)
}

func TestCancelPriceTriggeredOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelPriceTriggeredOrder(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.CancelPriceTriggeredOrder(t.Context(), "1234")
	assert.NoError(t, err)
}

func TestGetMarginAccountList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetIsolatedMarginAccountList(t.Context(), currency.EMPTYPAIR)
	assert.NoError(t, err)

	_, err = e.GetIsolatedMarginAccountList(t.Context(), getPair(t, asset.Margin))
	assert.NoError(t, err)
}

func TestListMarginAccountBalanceChangeHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.ListMarginAccountBalanceChangeHistory(t.Context(), currency.BTC, getPair(t, asset.Margin), endTime, startTime, 0, 0)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.ListMarginAccountBalanceChangeHistory(t.Context(), currency.BTC, getPair(t, asset.Margin), time.Time{}, time.Time{}, 0, 0)
	assert.NoError(t, err)

	_, err = e.ListMarginAccountBalanceChangeHistory(t.Context(), currency.BTC, getPair(t, asset.Margin), startTime, endTime, 1, 100)
	assert.NoError(t, err)
}

func TestGetMarginFundingAccountList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetMarginFundingAccountList(t.Context(), currency.EMPTYCODE)
	assert.NoError(t, err)

	_, err = e.GetMarginFundingAccountList(t.Context(), currency.BTC)
	assert.NoError(t, err)
}

func TestUpdateUsersAutoRepaymentSetting(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.UpdateUsersAutoRepaymentSetting(t.Context(), true)
	assert.NoError(t, err)
}

func TestGetUserAutoRepaymentSetting(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetUserAutoRepaymentSetting(t.Context())
	assert.NoError(t, err)
}

func TestGetMaxTransferableAmountForSpecificMarginCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.GetMaxTransferableAmountForSpecificMarginCurrency(t.Context(), currency.EMPTYCODE, currency.EMPTYPAIR)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetMaxTransferableAmountForSpecificMarginCurrency(t.Context(), currency.BTC, getPair(t, asset.Margin))
	assert.NoError(t, err)
}

func TestCurrencySupportedByCrossMargin(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.CurrencySupportedByCrossMargin(t.Context())
	assert.NoError(t, err)
}

func TestGetCrossMarginSupportedCurrencyDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossMarginSupportedCurrencyDetail(t.Context(), currency.EMPTYCODE)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetCrossMarginSupportedCurrencyDetail(t.Context(), currency.BTC)
	assert.NoError(t, err)
}

func TestGetCrossMarginAccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetCrossMarginAccounts(t.Context())
	assert.NoError(t, err)
}

func TestGetCrossMarginAccountChangeHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetCrossMarginAccountChangeHistory(t.Context(), currency.BTC, endTime, startTime, 0, 6, "in")
	assert.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetCrossMarginAccountChangeHistory(t.Context(), currency.BTC, startTime, endTime, 1, 6, "in")
	assert.NoError(t, err)
}

var createCrossMarginBorrowLoanJSON = `{"id": "17",	"create_time": 1620381696159,	"update_time": 1620381696159,	"currency": "EOS",	"amount": "110.553635",	"text": "web",	"status": 2,	"repaid": "110.506649705159",	"repaid_interest": "0.046985294841",	"unpaid_interest": "0.0000074393366667"}`

func TestCreateCrossMarginBorrowLoan(t *testing.T) {
	t.Parallel()
	var response CrossMarginLoanResponse
	err := json.Unmarshal([]byte(createCrossMarginBorrowLoanJSON), &response)
	assert.NoError(t, err)

	_, err = e.CreateCrossMarginBorrowLoan(t.Context(), CrossMarginBorrowLoanParams{Currency: currency.EMPTYCODE, Amount: 3})
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.CreateCrossMarginBorrowLoan(t.Context(), CrossMarginBorrowLoanParams{Currency: currency.BTC, Amount: 0})
	assert.ErrorIs(t, err, order.ErrAmountIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.CreateCrossMarginBorrowLoan(t.Context(), CrossMarginBorrowLoanParams{
		Currency: currency.BTC,
		Amount:   3,
	})
	assert.NoError(t, err)
}

func TestGetCrossMarginBorrowHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossMarginBorrowHistory(t.Context(), 6, currency.BTC, 0, 0, false)
	require.ErrorIs(t, err, errInvalidOrderStatus)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetCrossMarginBorrowHistory(t.Context(), 1, currency.BTC, 0, 0, false)
	assert.NoError(t, err)
}

func TestGetSingleBorrowLoanDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetSingleBorrowLoanDetail(t.Context(), "")
	require.ErrorIs(t, err, errInvalidLoanID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetSingleBorrowLoanDetail(t.Context(), "1234")
	assert.NoError(t, err)
}

func TestExecuteRepayment(t *testing.T) {
	t.Parallel()
	_, err := e.ExecuteRepayment(t.Context(), CurrencyAndAmount{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.ExecuteRepayment(t.Context(), CurrencyAndAmount{Currency: currency.BTC})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.ExecuteRepayment(t.Context(), CurrencyAndAmount{
		Currency: currency.USD,
		Amount:   1234.55,
	})
	assert.NoError(t, err)
}

func TestGetCrossMarginRepayments(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetCrossMarginRepayments(t.Context(), currency.BTC, "123", 1000, 10, true)
	assert.NoError(t, err)
}

func TestGetMaxTransferableAmountForSpecificCrossMarginCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.GetMaxTransferableAmountForSpecificCrossMarginCurrency(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetMaxTransferableAmountForSpecificCrossMarginCurrency(t.Context(), currency.BTC)
	assert.NoError(t, err)
}

func TestGetMaxBorrowableAmountForSpecificCrossMarginCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.GetMaxBorrowableAmountForSpecificCrossMarginCurrency(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetMaxBorrowableAmountForSpecificCrossMarginCurrency(t.Context(), currency.BTC)
	assert.NoError(t, err)
}

func TestListCurrencyChain(t *testing.T) {
	t.Parallel()
	_, err := e.ListCurrencyChain(t.Context(), currency.EMPTYCODE)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.ListCurrencyChain(t.Context(), currency.BTC)
	assert.NoError(t, err)
}

func TestGenerateCurrencyDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.GenerateCurrencyDepositAddress(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GenerateCurrencyDepositAddress(t.Context(), currency.BTC)
	assert.NoError(t, err)
}

func TestGetWithdrawalRecords(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetWithdrawalRecords(t.Context(), currency.BTC, endTime, startTime, 0, 0)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetWithdrawalRecords(t.Context(), currency.BTC, startTime, endTime, 10, 100)
	assert.NoError(t, err)
}

func TestGetDepositRecords(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetDepositRecords(t.Context(), currency.BTC, endTime, startTime, 0, 0)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetDepositRecords(t.Context(), currency.BTC, startTime, endTime, 10, 100)
	assert.NoError(t, err)
}

func TestTransferCurrency(t *testing.T) {
	t.Parallel()
	arg := &TransferCurrencyParam{}
	_, err := e.TransferCurrency(t.Context(), arg)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	arg.Currency = currency.BTC
	_, err = e.TransferCurrency(t.Context(), arg)
	assert.ErrorIs(t, err, asset.ErrInvalidAsset)

	arg.From = asset.Margin
	_, err = e.TransferCurrency(t.Context(), arg)
	assert.ErrorIs(t, err, asset.ErrInvalidAsset)

	arg.To = asset.Futures
	_, err = e.TransferCurrency(t.Context(), arg)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	arg.CurrencyPair = getPair(t, asset.Margin)
	_, err = e.TransferCurrency(t.Context(), arg)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	arg.Settle = currency.USDT
	_, err = e.TransferCurrency(t.Context(), arg)
	assert.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.TransferCurrency(t.Context(), &TransferCurrencyParam{
		Currency:     currency.BTC,
		From:         asset.Spot,
		To:           asset.Margin,
		Amount:       1202.000,
		CurrencyPair: getPair(t, asset.Spot),
	})
	assert.NoError(t, err)

	_, err = e.TransferCurrency(t.Context(), &TransferCurrencyParam{
		Currency:     currency.BTC,
		Settle:       currency.USDT,
		From:         asset.Futures,
		To:           asset.Margin,
		Amount:       1202.000,
		CurrencyPair: getPair(t, asset.Spot),
	})
	assert.NoError(t, err)
}

func TestSubAccountTransfer(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	req := &SubAccountTransferParam{SubAccountType: "index"}
	require.ErrorIs(t, e.SubAccountTransfer(ctx, req), currency.ErrCurrencyCodeEmpty)
	req.Currency = currency.BTC
	require.ErrorIs(t, e.SubAccountTransfer(ctx, req), errInvalidSubAccount)
	req.SubAccount = "1337"
	require.ErrorIs(t, e.SubAccountTransfer(ctx, req), errInvalidTransferDirection)
	req.Direction = "to"
	require.ErrorIs(t, e.SubAccountTransfer(ctx, req), order.ErrAmountIsInvalid)
	req.Amount = 1.337
	require.ErrorIs(t, e.SubAccountTransfer(ctx, req), asset.ErrNotSupported)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	req.SubAccountType = "spot"
	require.NoError(t, e.SubAccountTransfer(ctx, req))
}

func TestGetSubAccountTransferHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.GetSubAccountTransferHistory(t.Context(), "", time.Time{}, time.Time{}, 0, 0)
	assert.NoError(t, err)
}

func TestSubAccountTransferToSubAccount(t *testing.T) {
	t.Parallel()
	arg := &InterSubAccountTransferParams{}
	_, err := e.SubAccountTransferToSubAccount(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	arg.Currency = currency.BTC
	_, err = e.SubAccountTransferToSubAccount(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidSubAccountUserID)

	arg.SubAccountFromUserID = "123456"
	_, err = e.SubAccountTransferToSubAccount(t.Context(), arg)
	require.ErrorIs(t, err, asset.ErrInvalidAsset)

	arg.SubAccountFromAssetType = asset.Spot
	_, err = e.SubAccountTransferToSubAccount(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidSubAccountUserID)

	arg.SubAccountToUserID = "993883"
	_, err = e.SubAccountTransferToSubAccount(t.Context(), arg)
	require.ErrorIs(t, err, asset.ErrInvalidAsset)

	arg.SubAccountToAssetType = asset.Margin
	_, err = e.SubAccountTransferToSubAccount(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.SubAccountTransferToSubAccount(t.Context(), &InterSubAccountTransferParams{
		Currency:                currency.BTC,
		SubAccountFromUserID:    "1234",
		SubAccountFromAssetType: asset.Spot,
		SubAccountToUserID:      "4567",
		SubAccountToAssetType:   asset.Spot,
		Amount:                  1234,
	})
	assert.NoError(t, err)
}

func TestGetWithdrawalStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetWithdrawalStatus(t.Context(), currency.EMPTYCODE)
	assert.NoError(t, err)
}

func TestGetSubAccountBalances(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSubAccountBalances(t.Context(), "")
	assert.NoError(t, err)
}

func TestGetSubAccountMarginBalances(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSubAccountMarginBalances(t.Context(), "")
	assert.NoError(t, err)
}

func TestGetTransferOrderStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetTransferOrderStatus(t.Context(), "12345678", "abcdefg")
	assert.NoError(t, err)
}

func TestGetSubAccountFuturesBalances(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSubAccountFuturesBalances(t.Context(), "", currency.EMPTYCODE)
	assert.NoError(t, err)
}

func TestGetSubAccountCrossMarginBalances(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSubAccountCrossMarginBalances(t.Context(), "")
	assert.NoError(t, err)
}

func TestGetSavedAddresses(t *testing.T) {
	t.Parallel()
	_, err := e.GetSavedAddresses(t.Context(), currency.EMPTYCODE, "", 0, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetSavedAddresses(t.Context(), currency.BTC, "", 0, 10)
	assert.NoError(t, err)
}

func TestGetPersonalTradingFee(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetPersonalTradingFee(t.Context(), getPair(t, asset.Spot), currency.EMPTYCODE)
	assert.NoError(t, err)
}

func TestGetUsersTotalBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetUsersTotalBalance(t.Context(), currency.BTC)
	assert.NoError(t, err)
}

func TestGetAllFutureContracts(t *testing.T) {
	t.Parallel()

	for _, c := range []currency.Code{currency.BTC, currency.USDT} {
		_, err := e.GetAllFutureContracts(t.Context(), c)
		assert.NoErrorf(t, err, "GetAllFutureContracts %s should not error", c)
	}
}

func TestGetFuturesContract(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesContract(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures).String())
	assert.NoError(t, err)
	_, err = e.GetFuturesContract(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures).String())
	assert.NoError(t, err)
}

func TestGetFuturesOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesOrderbook(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), "", 10, false)
	assert.NoError(t, err)
	_, err = e.GetFuturesOrderbook(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), "", 10, false)
	assert.NoError(t, err)
}

func TestGetFuturesTradingHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesTradingHistory(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), 0, 0, "", time.Time{}, time.Time{})
	assert.NoError(t, err)
	_, err = e.GetFuturesTradingHistory(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), 0, 0, "", time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetFuturesCandlesticks(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesCandlesticks(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), time.Time{}, time.Time{}, 0, kline.OneWeek)
	assert.NoError(t, err)
	_, err = e.GetFuturesCandlesticks(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), time.Time{}, time.Time{}, 0, kline.OneWeek)
	assert.NoError(t, err)
}

func TestPremiumIndexKLine(t *testing.T) {
	t.Parallel()
	_, err := e.PremiumIndexKLine(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), time.Time{}, time.Time{}, 0, kline.OneWeek)
	assert.NoError(t, err)
	_, err = e.PremiumIndexKLine(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), time.Time{}, time.Time{}, 0, kline.OneWeek)
	assert.NoError(t, err)
}

func TestGetFutureTickers(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesTickers(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures))
	assert.NoError(t, err)
	_, err = e.GetFuturesTickers(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures))
	assert.NoError(t, err)
}

func TestGetFutureFundingRates(t *testing.T) {
	t.Parallel()
	_, err := e.GetFutureFundingRates(t.Context(), currency.EMPTYCODE, getPair(t, asset.CoinMarginedFutures), 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.GetFutureFundingRates(t.Context(), currency.BTC, currency.EMPTYPAIR, 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.GetFutureFundingRates(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), 0)
	assert.NoError(t, err)
	_, err = e.GetFutureFundingRates(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), 10)
	assert.NoError(t, err)
}

func TestGetFuturesInsuranceBalanceHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesInsuranceBalanceHistory(t.Context(), currency.EMPTYCODE, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.GetFuturesInsuranceBalanceHistory(t.Context(), currency.USDT, 10)
	assert.NoError(t, err)
}

func TestGetFutureStats(t *testing.T) {
	t.Parallel()
	_, err := e.GetFutureStats(t.Context(), currency.EMPTYCODE, getPair(t, asset.CoinMarginedFutures), time.Time{}, 0, 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.GetFutureStats(t.Context(), currency.BTC, currency.EMPTYPAIR, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.GetFutureStats(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), time.Time{}, 0, 0)
	assert.NoError(t, err)
	_, err = e.GetFutureStats(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), time.Now().Add(-time.Hour), kline.OneMin, 100)
	assert.NoError(t, err)
}

func TestGetIndexConstituent(t *testing.T) {
	t.Parallel()
	_, err := e.GetIndexConstituent(t.Context(), currency.EMPTYCODE, getPair(t, asset.USDTMarginedFutures).String())
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.GetIndexConstituent(t.Context(), currency.USDT, "")
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	result, err := e.GetIndexConstituent(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures).String())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLiquidationHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetLiquidationHistory(t.Context(), currency.EMPTYCODE, getPair(t, asset.USDTMarginedFutures), time.Time{}, time.Time{}, 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.GetLiquidationHistory(t.Context(), currency.USDT, currency.EMPTYPAIR, time.Time{}, time.Time{}, 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	startTime, endTime := getTimeWithInterval(kline.OneHour)
	_, err = e.GetLiquidationHistory(t.Context(), getPair(t, asset.USDTMarginedFutures).Quote, getPair(t, asset.USDTMarginedFutures), endTime, startTime, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterEnd)

	usdtmFuturesTP := getPair(t, asset.USDTMarginedFutures)
	_, err = e.GetLiquidationHistory(t.Context(), usdtmFuturesTP.Quote, usdtmFuturesTP, startTime, endTime, 100)
	assert.NoError(t, err)
}

func TestGetRiskLimitTiers(t *testing.T) {
	t.Parallel()
	_, err := e.GetRiskLimitTiers(t.Context(), currency.EMPTYCODE, getPair(t, asset.USDTMarginedFutures), 10, 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.GetRiskLimitTiers(t.Context(), currency.BTC, currency.EMPTYPAIR, 10, 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	usdtmFuturesTP := getPair(t, asset.USDTMarginedFutures)
	_, err = e.GetRiskLimitTiers(t.Context(), usdtmFuturesTP.Quote, usdtmFuturesTP, 1, 10)
	assert.NoError(t, err)
}

func TestQueryFuturesAccount(t *testing.T) {
	t.Parallel()
	_, err := e.QueryFuturesAccount(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.QueryFuturesAccount(t.Context(), currency.USDT)
	assert.NoError(t, err)
}

func TestGetFuturesAccountBooks(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesAccountBooks(t.Context(), currency.EMPTYCODE, 0, time.Time{}, time.Time{}, "dnw")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	startTime, endTime := getTime()
	_, err = e.GetFuturesAccountBooks(t.Context(), currency.USDT, 0, endTime, startTime, "dnw")
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetFuturesAccountBooks(t.Context(), currency.USDT, 10, startTime, endTime, "dnw")
	assert.NoError(t, err)
}

func TestGetAllFuturesPositionsOfUsers(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllFuturesPositionsOfUsers(t.Context(), currency.EMPTYCODE, true)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetAllFuturesPositionsOfUsers(t.Context(), currency.USDT, true)
	assert.NoError(t, err)
}

func TestGetSinglePosition(t *testing.T) {
	t.Parallel()
	_, err := e.GetSinglePosition(t.Context(), currency.EMPTYCODE, getPair(t, asset.USDTMarginedFutures))
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.GetSinglePosition(t.Context(), currency.USDT, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetSinglePosition(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures))
	assert.NoError(t, err)
}

func TestUpdateFuturesPositionMargin(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateFuturesPositionMargin(t.Context(), currency.EMPTYCODE, 0.01, getPair(t, asset.CoinMarginedFutures))
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.UpdateFuturesPositionMargin(t.Context(), currency.BTC, 0.01, currency.EMPTYPAIR)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.UpdateFuturesPositionMargin(t.Context(), currency.BTC, 0, getPair(t, asset.CoinMarginedFutures))
	assert.ErrorIs(t, err, errChangeHasToBePositive)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.UpdateFuturesPositionMargin(t.Context(), currency.BTC, 0.01, getPair(t, asset.CoinMarginedFutures))
	assert.NoError(t, err)
	_, err = e.UpdateFuturesPositionMargin(t.Context(), currency.USDT, 0.01, getPair(t, asset.USDTMarginedFutures))
	assert.NoError(t, err)
}

func TestUpdateFuturesPositionLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateFuturesPositionLeverage(t.Context(), currency.EMPTYCODE, getPair(t, asset.CoinMarginedFutures), 1, 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.UpdateFuturesPositionLeverage(t.Context(), currency.BTC, currency.EMPTYPAIR, 1, 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.UpdateFuturesPositionLeverage(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), -1, 0)
	assert.ErrorIs(t, err, order.ErrSubmitLeverageNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.UpdateFuturesPositionLeverage(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), 1, 0)
	assert.NoError(t, err)
}

func TestUpdateFuturesPositionRiskLimit(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateFuturesPositionRiskLimit(t.Context(), currency.EMPTYCODE, getPair(t, asset.CoinMarginedFutures), 10)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.UpdateFuturesPositionRiskLimit(t.Context(), currency.BTC, currency.EMPTYPAIR, 10)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.UpdateFuturesPositionRiskLimit(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), 10)
	assert.NoError(t, err)
}

func TestPlaceDeliveryOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.PlaceDeliveryOrder(t.Context(), &ContractOrderCreateParams{
		Contract:    getPair(t, asset.DeliveryFutures),
		Size:        6024,
		Iceberg:     0,
		Price:       3765,
		Text:        "t-my-custom-id",
		Settle:      currency.USDT,
		TimeInForce: gtcTIF,
	})
	assert.NoError(t, err)
}

func TestGetDeliveryOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetDeliveryOrders(t.Context(), currency.EMPTYPAIR, currency.EMPTYCODE, statusOpen, "", 0, 0, true)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.GetDeliveryOrders(t.Context(), getPair(t, asset.DeliveryFutures), currency.USDT, "", "", 0, 0, true)
	require.ErrorIs(t, err, errInvalidOrderStatus)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetDeliveryOrders(t.Context(), getPair(t, asset.DeliveryFutures), currency.USDT, statusOpen, "", 0, 0, true)
	assert.NoError(t, err, "GetDeliveryOrders should not error")
}

func TestCancelMultipleDeliveryOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelMultipleDeliveryOrders(t.Context(), currency.EMPTYPAIR, "ask", currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.CancelMultipleDeliveryOrders(t.Context(), currency.EMPTYPAIR, "ask", currency.USDT)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.CancelMultipleDeliveryOrders(t.Context(), getPair(t, asset.DeliveryFutures), "ask", currency.USDT)
	assert.NoError(t, err)
}

func TestGetSingleDeliveryOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetSingleDeliveryOrder(t.Context(), currency.EMPTYCODE, "123456")
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.GetSingleDeliveryOrder(t.Context(), currency.USDT, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetSingleDeliveryOrder(t.Context(), currency.USDT, "123456")
	assert.NoError(t, err)
}

func TestCancelSingleDeliveryOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelSingleDeliveryOrder(t.Context(), currency.EMPTYCODE, "123456")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.CancelSingleDeliveryOrder(t.Context(), currency.USDT, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.CancelSingleDeliveryOrder(t.Context(), currency.USDT, "123456")
	assert.NoError(t, err)
}

func TestGetMyDeliveryTradingHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetMyDeliveryTradingHistory(t.Context(), currency.EMPTYCODE, "", getPair(t, asset.DeliveryFutures), 0, 0, 1, "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetMyDeliveryTradingHistory(t.Context(), currency.USDT, "", getPair(t, asset.DeliveryFutures), 0, 0, 1, "")
	assert.NoError(t, err)
}

func TestGetDeliveryPositionCloseHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetDeliveryPositionCloseHistory(t.Context(), currency.EMPTYCODE, getPair(t, asset.DeliveryFutures), 0, 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetDeliveryPositionCloseHistory(t.Context(), currency.USDT, getPair(t, asset.DeliveryFutures), 0, 0, time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetDeliveryLiquidationHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetDeliveryLiquidationHistory(t.Context(), currency.EMPTYCODE, getPair(t, asset.DeliveryFutures), 0, time.Now())
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetDeliveryLiquidationHistory(t.Context(), currency.USDT, getPair(t, asset.DeliveryFutures), 0, time.Now())
	assert.NoError(t, err)
}

func TestGetDeliverySettlementHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetDeliverySettlementHistory(t.Context(), currency.EMPTYCODE, getPair(t, asset.DeliveryFutures), 0, time.Now())
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetDeliverySettlementHistory(t.Context(), currency.USDT, getPair(t, asset.DeliveryFutures), 0, time.Now())
	assert.NoError(t, err)
}

func TestGetDeliveryPriceTriggeredOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetDeliveryPriceTriggeredOrder(t.Context(), currency.USDT, &FuturesPriceTriggeredOrderParam{
		Initial: FuturesInitial{
			Price:    1234.,
			Size:     12,
			Contract: getPair(t, asset.DeliveryFutures),
		},
		Trigger: FuturesTrigger{
			Rule:      1,
			OrderType: "close-short-position",
			Price:     123400,
		},
	})
	assert.NoError(t, err)
}

func TestGetDeliveryAllAutoOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetDeliveryAllAutoOrder(t.Context(), "", currency.USDT, getPair(t, asset.DeliveryFutures), 0, 1)
	require.ErrorIs(t, err, errInvalidOrderStatus)

	_, err = e.GetDeliveryAllAutoOrder(t.Context(), statusOpen, currency.EMPTYCODE, getPair(t, asset.DeliveryFutures), 0, 1)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetDeliveryAllAutoOrder(t.Context(), statusOpen, currency.USDT, getPair(t, asset.DeliveryFutures), 0, 1)
	assert.NoError(t, err)
}

func TestCancelAllDeliveryPriceTriggeredOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllDeliveryPriceTriggeredOrder(t.Context(), currency.EMPTYCODE, getPair(t, asset.DeliveryFutures))
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.CancelAllDeliveryPriceTriggeredOrder(t.Context(), currency.USDT, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.CancelAllDeliveryPriceTriggeredOrder(t.Context(), currency.USDT, getPair(t, asset.DeliveryFutures))
	assert.NoError(t, err)
}

func TestGetSingleDeliveryPriceTriggeredOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetSingleDeliveryPriceTriggeredOrder(t.Context(), currency.EMPTYCODE, "12345")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.GetSingleDeliveryPriceTriggeredOrder(t.Context(), currency.USDT, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetSingleDeliveryPriceTriggeredOrder(t.Context(), currency.USDT, "12345")
	assert.NoError(t, err)
}

func TestCancelDeliveryPriceTriggeredOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelDeliveryPriceTriggeredOrder(t.Context(), currency.EMPTYCODE, "12345")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.CancelDeliveryPriceTriggeredOrder(t.Context(), currency.USDT, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.CancelDeliveryPriceTriggeredOrder(t.Context(), currency.USDT, "12345")
	assert.NoError(t, err)
}

func TestEnableOrDisableDualMode(t *testing.T) {
	t.Parallel()
	_, err := e.EnableOrDisableDualMode(t.Context(), currency.EMPTYCODE, true)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.EnableOrDisableDualMode(t.Context(), currency.BTC, true)
	assert.NoError(t, err)
}

func TestRetrivePositionDetailInDualMode(t *testing.T) {
	t.Parallel()
	_, err := e.GetPositionDetailInDualMode(t.Context(), currency.EMPTYCODE, getPair(t, asset.CoinMarginedFutures))
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.GetPositionDetailInDualMode(t.Context(), currency.BTC, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetPositionDetailInDualMode(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures))
	assert.NoError(t, err)

	_, err = e.GetPositionDetailInDualMode(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures))
	assert.NoError(t, err)
}

func TestUpdatePositionMarginInDualMode(t *testing.T) {
	t.Parallel()
	_, err := e.UpdatePositionMarginInDualMode(t.Context(), currency.EMPTYCODE, getPair(t, asset.CoinMarginedFutures), 0.001, "dual_long")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.UpdatePositionMarginInDualMode(t.Context(), currency.BTC, currency.EMPTYPAIR, 0.001, "dual_long")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.UpdatePositionMarginInDualMode(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), 0.001, "")
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.UpdatePositionMarginInDualMode(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), 0.001, "dual_long")
	assert.NoError(t, err)

	_, err = e.UpdatePositionMarginInDualMode(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), 0.001, "dual_long")
	assert.NoError(t, err)
}

func TestUpdatePositionLeverageInDualMode(t *testing.T) {
	t.Parallel()
	_, err := e.UpdatePositionLeverageInDualMode(t.Context(), currency.EMPTYCODE, getPair(t, asset.USDTMarginedFutures), 1, 0.001)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.UpdatePositionLeverageInDualMode(t.Context(), currency.USDT, currency.EMPTYPAIR, 1, 0.001)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.UpdatePositionLeverageInDualMode(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), -1, 0.001)
	require.ErrorIs(t, err, order.ErrSubmitLeverageNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.UpdatePositionLeverageInDualMode(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), 1, 0.001)
	assert.NoError(t, err)

	_, err = e.UpdatePositionLeverageInDualMode(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), 10, 0.001)
	assert.NoError(t, err)
}

func TestUpdatePositionRiskLimitInDualMode(t *testing.T) {
	t.Parallel()
	_, err := e.UpdatePositionRiskLimitInDualMode(t.Context(), currency.EMPTYCODE, getPair(t, asset.CoinMarginedFutures), 10)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.UpdatePositionRiskLimitInDualMode(t.Context(), currency.BTC, currency.EMPTYPAIR, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.UpdatePositionRiskLimitInDualMode(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), 0)
	require.ErrorIs(t, err, errInvalidRiskLimit)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.UpdatePositionRiskLimitInDualMode(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), 20000)
	assert.NoError(t, err)

	_, err = e.UpdatePositionRiskLimitInDualMode(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), 20000)
	assert.NoError(t, err)
}

func TestPlaceFuturesOrder(t *testing.T) {
	t.Parallel()
	arg := &ContractOrderCreateParams{}
	_, err := e.PlaceFuturesOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	arg.Contract = getPair(t, asset.USDTMarginedFutures)
	_, err = e.PlaceFuturesOrder(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidOrderSize)

	arg.Size = 1
	_, err = e.PlaceFuturesOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrUnsupportedTimeInForce)

	arg.TimeInForce = order.GoodTillCancel.Lower()
	_, err = e.PlaceFuturesOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrUnsupportedTimeInForce)

	arg.Price = 123
	arg.Text = "client-order-id"
	_, err = e.PlaceFuturesOrder(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidOrderText)

	arg.Text = "t-client-order-id"
	arg.AutoSize = "long"
	_, err = e.PlaceFuturesOrder(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidAutoSize)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.PlaceFuturesOrder(t.Context(), &ContractOrderCreateParams{
		Contract:    getPair(t, asset.CoinMarginedFutures),
		Size:        6024,
		Iceberg:     0,
		Price:       3765,
		TimeInForce: "gtc",
		Text:        "t-my-custom-id",
		Settle:      currency.BTC,
	})
	assert.NoError(t, err)
}

func TestGetFuturesOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesOrders(t.Context(), currency.NewBTCUSDT(), statusOpen, "", currency.EMPTYCODE, 0, 0, true)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.GetFuturesOrders(t.Context(), currency.NewBTCUSD(), "", "", currency.BTC, 0, 0, true)
	require.ErrorIs(t, err, errInvalidOrderStatus)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetFuturesOrders(t.Context(), currency.NewBTCUSD(), statusOpen, "", currency.BTC, 0, 0, true)
	assert.NoError(t, err, "GetFuturesOrders should not error")
}

func TestCancelMultipleFuturesOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelMultipleFuturesOpenOrders(t.Context(), getPair(t, asset.USDTMarginedFutures), "ask", currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.CancelMultipleFuturesOpenOrders(t.Context(), currency.EMPTYPAIR, "ask", currency.USDT)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.CancelMultipleFuturesOpenOrders(t.Context(), getPair(t, asset.USDTMarginedFutures), "ask", currency.USDT)
	assert.NoError(t, err)
}

func TestGetSingleFuturesPriceTriggeredOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetSingleFuturesPriceTriggeredOrder(t.Context(), currency.EMPTYCODE, "12345")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.GetSingleFuturesPriceTriggeredOrder(t.Context(), currency.BTC, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetSingleFuturesPriceTriggeredOrder(t.Context(), currency.BTC, "12345")
	assert.NoError(t, err)
}

func TestCancelFuturesPriceTriggeredOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelFuturesPriceTriggeredOrder(t.Context(), currency.EMPTYCODE, "12345")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.CancelFuturesPriceTriggeredOrder(t.Context(), currency.BTC, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.CancelFuturesPriceTriggeredOrder(t.Context(), currency.USDT, "12345")
	assert.NoError(t, err)
}

func TestPlaceBatchFuturesOrders(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceBatchFuturesOrders(t.Context(), currency.EMPTYCODE, []ContractOrderCreateParams{{}})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	arg := ContractOrderCreateParams{}
	_, err = e.PlaceBatchFuturesOrders(t.Context(), currency.USDT, []ContractOrderCreateParams{arg})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	arg.Contract = getPair(t, asset.USDTMarginedFutures)
	_, err = e.PlaceBatchFuturesOrders(t.Context(), currency.USDT, []ContractOrderCreateParams{arg})
	require.ErrorIs(t, err, errInvalidOrderSize)

	arg.Size = 1
	_, err = e.PlaceBatchFuturesOrders(t.Context(), currency.USDT, []ContractOrderCreateParams{arg})
	require.ErrorIs(t, err, order.ErrUnsupportedTimeInForce)

	arg.TimeInForce = order.GoodTillCancel.Lower()
	_, err = e.PlaceBatchFuturesOrders(t.Context(), currency.USDT, []ContractOrderCreateParams{arg})
	require.ErrorIs(t, err, order.ErrUnsupportedTimeInForce)

	arg.Price = 123
	arg.Text = "client-order-id"
	_, err = e.PlaceBatchFuturesOrders(t.Context(), currency.USDT, []ContractOrderCreateParams{arg})
	require.ErrorIs(t, err, errInvalidOrderText)

	arg.Text = "t-client-order-id"
	arg.AutoSize = "long"
	_, err = e.PlaceBatchFuturesOrders(t.Context(), currency.USDT, []ContractOrderCreateParams{arg})
	require.ErrorIs(t, err, errInvalidAutoSize)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.PlaceBatchFuturesOrders(t.Context(), currency.BTC, []ContractOrderCreateParams{
		{
			Contract:    getPair(t, asset.CoinMarginedFutures),
			Size:        6024,
			Iceberg:     0,
			Price:       3765,
			TimeInForce: "gtc",
			Text:        "t-my-custom-id",
			Settle:      currency.BTC,
		},
		{
			Contract:    getPair(t, asset.CoinMarginedFutures),
			Size:        232,
			Iceberg:     0,
			Price:       376225,
			TimeInForce: "gtc",
			Text:        "t-my-custom-id",
			Settle:      currency.BTC,
		},
	})
	assert.NoError(t, err)
}

func TestGetSingleFuturesOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetSingleFuturesOrder(t.Context(), currency.EMPTYCODE, "12345")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.GetSingleFuturesOrder(t.Context(), currency.BTC, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetSingleFuturesOrder(t.Context(), currency.BTC, "12345")
	assert.NoError(t, err)
}

func TestCancelSingleFuturesOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelSingleFuturesOrder(t.Context(), currency.EMPTYCODE, "12345")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.CancelSingleFuturesOrder(t.Context(), currency.BTC, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.CancelSingleFuturesOrder(t.Context(), currency.BTC, "12345")
	assert.NoError(t, err)
}

func TestAmendFuturesOrder(t *testing.T) {
	t.Parallel()
	_, err := e.AmendFuturesOrder(t.Context(), currency.EMPTYCODE, "1234", AmendFuturesOrderParam{Price: 12345.990})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.AmendFuturesOrder(t.Context(), currency.BTC, "", AmendFuturesOrderParam{Price: 12345.990})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	_, err = e.AmendFuturesOrder(t.Context(), currency.BTC, "", AmendFuturesOrderParam{Price: 12345.990})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.AmendFuturesOrder(t.Context(), currency.BTC, "1234", AmendFuturesOrderParam{Price: 12345.990})
	assert.NoError(t, err)
}

func TestGetMyFuturesTradingHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetMyFuturesTradingHistory(t.Context(), currency.EMPTYCODE, "", "", getPair(t, asset.CoinMarginedFutures), 0, 0, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetMyFuturesTradingHistory(t.Context(), currency.BTC, "", "", getPair(t, asset.CoinMarginedFutures), 0, 0, 0)
	assert.NoError(t, err)
}

func TestGetFuturesPositionCloseHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesPositionCloseHistory(t.Context(), currency.EMPTYCODE, getPair(t, asset.CoinMarginedFutures), 0, 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetFuturesPositionCloseHistory(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), 0, 0, time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetFuturesLiquidationHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesLiquidationHistory(t.Context(), currency.EMPTYCODE, getPair(t, asset.CoinMarginedFutures), 0, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetFuturesLiquidationHistory(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), 0, time.Time{})
	assert.NoError(t, err)
}

func TestCountdownCancelOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CountdownCancelOrders(t.Context(), currency.EMPTYCODE, CountdownParams{Timeout: 8})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.CountdownCancelOrders(t.Context(), currency.BTC, CountdownParams{Timeout: 8})
	assert.NoError(t, err)
}

func TestCreatePriceTriggeredFuturesOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	for _, tc := range []struct {
		c currency.Code
		a asset.Item
	}{
		{currency.BTC, asset.CoinMarginedFutures},
		{currency.USDT, asset.USDTMarginedFutures},
	} {
		_, err := e.CreatePriceTriggeredFuturesOrder(t.Context(), tc.c, &FuturesPriceTriggeredOrderParam{
			Initial: FuturesInitial{
				Price:    1234.,
				Size:     2,
				Contract: getPair(t, tc.a),
			},
			Trigger: FuturesTrigger{
				Rule:      1,
				OrderType: "close-short-position",
			},
		})
		assert.NoErrorf(t, err, "CreatePriceTriggeredFuturesOrder should not error for settlement currency: %s, asset: %s", tc.c, tc.a)
	}
}

func TestListAllFuturesAutoOrders(t *testing.T) {
	t.Parallel()
	_, err := e.ListAllFuturesAutoOrders(t.Context(), "", currency.BTC, currency.EMPTYPAIR, 0, 0)
	require.ErrorIs(t, err, errInvalidOrderStatus)

	_, err = e.ListAllFuturesAutoOrders(t.Context(), statusOpen, currency.EMPTYCODE, currency.EMPTYPAIR, 0, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.ListAllFuturesAutoOrders(t.Context(), statusOpen, currency.BTC, currency.EMPTYPAIR, 0, 0)
	assert.NoError(t, err)
}

func TestCancelAllFuturesOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllFuturesOpenOrders(t.Context(), currency.EMPTYCODE, getPair(t, asset.CoinMarginedFutures))
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.CancelAllFuturesOpenOrders(t.Context(), currency.BTC, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.CancelAllFuturesOpenOrders(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures))
	assert.NoError(t, err)
	_, err = e.CancelAllFuturesOpenOrders(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures))
	assert.NoError(t, err)
}

func TestGetAllDeliveryContracts(t *testing.T) {
	t.Parallel()
	e.Verbose = true
	r, err := e.GetAllDeliveryContracts(t.Context(), currency.USDT)
	require.NoError(t, err, "GetAllDeliveryContracts must not error")
	assert.NotEmpty(t, r, "GetAllDeliveryContracts should return data")
	r, err = e.GetAllDeliveryContracts(t.Context(), currency.BTC)
	require.NoError(t, err, "GetAllDeliveryContracts must not error")
	// The test below will fail if support for BTC settlement is added. This is intentional, as it ensures we are alerted when it's time to reintroduce support
	if !assert.Empty(t, r, "GetAllDeliveryContracts should not return any data with unsupported settlement currency BTC") {
		t.Error("BTC settlement for delivery futures appears to be supported again by the API. Please raise an issue to reintroduce BTC support for this exchange")
	}
}

func TestGetDeliveryContract(t *testing.T) {
	t.Parallel()
	_, err := e.GetDeliveryContract(t.Context(), currency.EMPTYCODE, getPair(t, asset.DeliveryFutures))
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.GetDeliveryContract(t.Context(), currency.USDT, getPair(t, asset.DeliveryFutures))
	assert.NoError(t, err)
}

func TestGetDeliveryOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.GetDeliveryOrderbook(t.Context(), currency.EMPTYCODE, "0", getPair(t, asset.DeliveryFutures), 0, false)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.GetDeliveryOrderbook(t.Context(), currency.USDT, "0", getPair(t, asset.DeliveryFutures), 0, false)
	assert.NoError(t, err)
}

func TestGetDeliveryTradingHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetDeliveryTradingHistory(t.Context(), currency.EMPTYCODE, "", getPair(t, asset.DeliveryFutures), 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.GetDeliveryTradingHistory(t.Context(), currency.USDT, "", currency.EMPTYPAIR, 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.GetDeliveryTradingHistory(t.Context(), currency.USDT, "", getPair(t, asset.DeliveryFutures), 0, time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetDeliveryFuturesCandlesticks(t *testing.T) {
	t.Parallel()
	_, err := e.GetDeliveryFuturesCandlesticks(t.Context(), currency.EMPTYCODE, getPair(t, asset.DeliveryFutures), time.Time{}, time.Time{}, 0, kline.OneWeek)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.GetDeliveryFuturesCandlesticks(t.Context(), currency.USDT, currency.EMPTYPAIR, time.Time{}, time.Time{}, 0, kline.OneWeek)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	startTime, endTime := getTime()
	_, err = e.GetDeliveryFuturesCandlesticks(t.Context(), currency.USDT, getPair(t, asset.DeliveryFutures), endTime, startTime, 10, kline.OneWeek)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	_, err = e.GetDeliveryFuturesCandlesticks(t.Context(), currency.USDT, getPair(t, asset.DeliveryFutures), startTime, endTime, 10, kline.Interval(5))
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval)

	deliveryFuturesPair := getPair(t, asset.DeliveryFutures)
	_, err = e.GetDeliveryFuturesCandlesticks(t.Context(), currency.USDT, deliveryFuturesPair, startTime, endTime, 0, kline.OneMin)
	assert.NoError(t, err)
}

func TestGetDeliveryFutureTickers(t *testing.T) {
	t.Parallel()
	_, err := e.GetDeliveryFutureTickers(t.Context(), currency.EMPTYCODE, getPair(t, asset.DeliveryFutures))
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	results, err := e.GetDeliveryFutureTickers(t.Context(), currency.USDT, currency.EMPTYPAIR)
	require.NoError(t, err)
	assert.NotNil(t, results)
}

func TestGetDeliveryInsuranceBalanceHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetDeliveryInsuranceBalanceHistory(t.Context(), currency.EMPTYCODE, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.GetDeliveryInsuranceBalanceHistory(t.Context(), currency.BTC, 0)
	assert.NoError(t, err)
}

func TestQueryDeliveryFuturesAccounts(t *testing.T) {
	t.Parallel()
	_, err := e.GetDeliveryFuturesAccounts(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetDeliveryFuturesAccounts(t.Context(), currency.USDT)
	assert.NoError(t, err)
}

func TestGetDeliveryAccountBooks(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetDeliveryAccountBooks(t.Context(), currency.EMPTYCODE, 0, startTime, endTime, "dnw")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.GetDeliveryAccountBooks(t.Context(), currency.USDT, 0, endTime, startTime, "dnw")
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetDeliveryAccountBooks(t.Context(), currency.USDT, 0, startTime, endTime, "dnw")
	assert.NoError(t, err)
}

func TestGetAllDeliveryPositionsOfUser(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllDeliveryPositionsOfUser(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetAllDeliveryPositionsOfUser(t.Context(), currency.USDT)
	assert.NoError(t, err)
}

func TestGetSingleDeliveryPosition(t *testing.T) {
	t.Parallel()
	_, err := e.GetSingleDeliveryPosition(t.Context(), currency.EMPTYCODE, getPair(t, asset.DeliveryFutures))
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.GetSingleDeliveryPosition(t.Context(), currency.USDT, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetSingleDeliveryPosition(t.Context(), currency.USDT, getPair(t, asset.DeliveryFutures))
	assert.NoError(t, err)
}

func TestUpdateDeliveryPositionMargin(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateDeliveryPositionMargin(t.Context(), currency.EMPTYCODE, 0.001, currency.EMPTYPAIR)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.UpdateDeliveryPositionMargin(t.Context(), currency.USDT, 0.001, currency.EMPTYPAIR)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.UpdateDeliveryPositionMargin(t.Context(), currency.USDT, 0, getPair(t, asset.DeliveryFutures))
	assert.ErrorIs(t, err, errChangeHasToBePositive)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.UpdateDeliveryPositionMargin(t.Context(), currency.USDT, 0.001, getPair(t, asset.DeliveryFutures))
	assert.NoError(t, err)
}

func TestUpdateDeliveryPositionLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateDeliveryPositionLeverage(t.Context(), currency.EMPTYCODE, currency.EMPTYPAIR, 0.001)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.UpdateDeliveryPositionLeverage(t.Context(), currency.USDT, currency.EMPTYPAIR, 0.001)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.UpdateDeliveryPositionLeverage(t.Context(), currency.USDT, getPair(t, asset.DeliveryFutures), -1)
	assert.ErrorIs(t, err, order.ErrSubmitLeverageNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.UpdateDeliveryPositionLeverage(t.Context(), currency.USDT, getPair(t, asset.DeliveryFutures), 1)
	assert.NoError(t, err)
}

func TestUpdateDeliveryPositionRiskLimit(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateDeliveryPositionRiskLimit(t.Context(), currency.EMPTYCODE, currency.EMPTYPAIR, 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.UpdateDeliveryPositionRiskLimit(t.Context(), currency.USDT, currency.EMPTYPAIR, 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.UpdateDeliveryPositionRiskLimit(t.Context(), currency.USDT, getPair(t, asset.DeliveryFutures), 30)
	assert.NoError(t, err)
}

func TestGetAllOptionsUnderlyings(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllOptionsUnderlyings(t.Context())
	assert.NoError(t, err)
}

func TestGetExpirationTime(t *testing.T) {
	t.Parallel()
	_, err := e.GetExpirationTime(t.Context(), "")
	assert.ErrorIs(t, err, errInvalidUnderlying)

	_, err = e.GetExpirationTime(t.Context(), "BTC_USDT")
	assert.NoError(t, err)
}

func TestGetAllContractOfUnderlyingWithinExpiryDate(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllContractOfUnderlyingWithinExpiryDate(t.Context(), "", time.Time{})
	assert.ErrorIs(t, err, errInvalidUnderlying)

	_, err = e.GetAllContractOfUnderlyingWithinExpiryDate(t.Context(), "BTC_USDT", time.Time{})
	assert.NoError(t, err)
}

func TestGetOptionsSpecifiedContractDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetOptionsSpecifiedContractDetail(t.Context(), currency.EMPTYPAIR)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	result, err := e.GetOptionsSpecifiedContractDetail(t.Context(), getPair(t, asset.Options))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSettlementHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetSettlementHistory(t.Context(), "", 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errInvalidUnderlying)

	startTime, endTime := getTime()
	_, err = e.GetSettlementHistory(t.Context(), "BTC_USDT", 0, 0, endTime, startTime)
	assert.ErrorIs(t, err, common.ErrStartAfterEnd)

	_, err = e.GetSettlementHistory(t.Context(), "BTC_USDT", 0, 0, time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetOptionsSpecifiedSettlementHistory(t *testing.T) {
	t.Parallel()
	underlying := "BTC_USDT"
	optionsSettlement, err := e.GetSettlementHistory(t.Context(), underlying, 0, 1, time.Time{}, time.Time{})
	require.NoError(t, err)

	cp, err := currency.NewPairFromString(optionsSettlement[0].Contract)
	require.NoError(t, err)

	_, err = e.GetOptionsSpecifiedContractsSettlement(t.Context(), cp, underlying, optionsSettlement[0].Timestamp.Time().Unix())
	assert.NoError(t, err)
}

func TestGetSupportedFlashSwapCurrencies(t *testing.T) {
	t.Parallel()
	result, err := e.GetSupportedFlashSwapCurrencies(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

const flashSwapOrderResponseJSON = `{"id": 54646,  "create_time": 1651116876378,  "update_time": 1651116876378,  "user_id": 11135567,  "sell_currency": "BTC",  "sell_amount": "0.01",  "buy_currency": "USDT",  "buy_amount": "10",  "price": "100",  "status": 1}`

func TestCreateFlashSwapOrder(t *testing.T) {
	t.Parallel()
	var response FlashSwapOrderResponse
	err := json.Unmarshal([]byte(flashSwapOrderResponseJSON), &response)
	assert.NoError(t, err)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.CreateFlashSwapOrder(t.Context(), FlashSwapOrderParams{
		PreviewID:    "1234",
		SellCurrency: currency.USDT,
		BuyCurrency:  currency.BTC,
		BuyAmount:    34234,
		SellAmount:   34234,
	})
	assert.NoError(t, err)
}

func TestGetAllFlashSwapOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetAllFlashSwapOrders(t.Context(), 1, currency.EMPTYCODE, currency.EMPTYCODE, true, 0, 0)
	assert.NoError(t, err)
}

func TestGetSingleFlashSwapOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetSingleFlashSwapOrder(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetSingleFlashSwapOrder(t.Context(), "1234")
	assert.NoError(t, err)
}

func TestInitiateFlashSwapOrderReview(t *testing.T) {
	t.Parallel()
	arg := &FlashSwapOrderParams{
		SellAmount: 100,
	}
	_, err := e.InitiateFlashSwapOrderReview(t.Context(), arg)
	assert.ErrorIs(t, err, errMissingPreviewID)

	arg.PreviewID = "1234"
	_, err = e.InitiateFlashSwapOrderReview(t.Context(), arg)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	arg.SellCurrency = currency.USDT
	_, err = e.InitiateFlashSwapOrderReview(t.Context(), arg)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.InitiateFlashSwapOrderReview(t.Context(), &FlashSwapOrderParams{
		PreviewID:    "1234",
		SellCurrency: currency.USDT,
		BuyCurrency:  currency.BTC,
		SellAmount:   100,
	})
	assert.NoError(t, err)
}

func TestSwapETH2(t *testing.T) {
	t.Parallel()
	err := e.SwapETH2(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	err = e.SwapETH2(t.Context(), &SwapETHParam{
		Side:   "",
		Amount: 231,
	})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	err = e.SwapETH2(t.Context(), &SwapETHParam{
		Side:   "1",
		Amount: 0,
	})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.SwapETH2(t.Context(), &SwapETHParam{
		Side:   "1",
		Amount: 231,
	})
	require.NoError(t, err)
}

func TestGetETH2HistoricalReturnRate(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetETH2HistoricalReturnRate(t.Context())
	require.NoError(t, err)
}

func TestGetDualInvestmentProductList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetDualInvestmentProductList(t.Context(), 123123412341)
	require.NoError(t, err)
}

func TestGetDualInvestmentOrderList(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetDualInvestmentOrderList(t.Context(), endTime, startTime, 0, 100)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetDualInvestmentOrderList(t.Context(), startTime, endTime, 0, 100)
	require.NoError(t, err)
}

func TestPlaceDualInvestmentOrder(t *testing.T) {
	t.Parallel()
	err := e.PlaceDualInvestmentOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	err = e.PlaceDualInvestmentOrder(t.Context(), &DualInvestmentOrderParam{Text: "abc"})
	require.ErrorIs(t, err, errPlanIDRequired)

	err = e.PlaceDualInvestmentOrder(t.Context(), &DualInvestmentOrderParam{PlanID: "12321"})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.PlaceDualInvestmentOrder(t.Context(), &DualInvestmentOrderParam{
		PlanID: "12321",
		Amount: 1223.213,
	})
	require.NoError(t, err)
}

func TestGetStructuredProductList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetStructuredProductList(t.Context(), "", "in_process", 10, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetStructuredProductOrderList(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetStructuredProductOrderList(t.Context(), endTime, startTime, 0, 100)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetStructuredProductOrderList(t.Context(), startTime, endTime, 0, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMyOptionsSettlements(t *testing.T) {
	t.Parallel()
	_, err := e.GetMyOptionsSettlements(t.Context(), "", currency.EMPTYPAIR, 0, 0, time.Time{})
	require.ErrorIs(t, err, errInvalidUnderlying)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetMyOptionsSettlements(t.Context(), "BTC_USDT", currency.EMPTYPAIR, 0, 0, time.Time{})
	assert.NoError(t, err)
}

func TestGetOptionAccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetOptionAccounts(t.Context())
	assert.NoError(t, err)
}

func TestGetAccountChangingHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetAccountChangingHistory(t.Context(), 0, 0, endTime, startTime, "")
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetAccountChangingHistory(t.Context(), 1, 100, startTime, endTime, "prem")
	assert.NoError(t, err)
}

func TestGetUsersPositionSpecifiedUnderlying(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetUsersPositionSpecifiedUnderlying(t.Context(), "BTC_USDT")
	assert.NoError(t, err)
}

func TestGetSpecifiedContractPosition(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpecifiedContractPosition(t.Context(), currency.EMPTYPAIR)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetSpecifiedContractPosition(t.Context(), getPair(t, asset.Options))
	assert.NoError(t, err)
}

func TestGetUsersLiquidationHistoryForSpecifiedUnderlying(t *testing.T) {
	t.Parallel()
	_, err := e.GetUsersLiquidationHistoryForSpecifiedUnderlying(t.Context(), "", currency.EMPTYPAIR)
	require.ErrorIs(t, err, errInvalidUnderlying)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetUsersLiquidationHistoryForSpecifiedUnderlying(t.Context(), "BTC_USDT", currency.EMPTYPAIR)
	assert.NoError(t, err)
}

func TestPlaceOptionOrder(t *testing.T) {
	t.Parallel()
	arg := &OptionOrderParam{
		OrderSize:   -1,
		Iceberg:     0,
		Text:        "-",
		TimeInForce: "gtc",
		Price:       100,
	}
	_, err := e.PlaceOptionOrder(t.Context(), arg)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	arg.Contract = getPair(t, asset.Options)
	_, err = e.PlaceOptionOrder(t.Context(), arg)
	assert.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.PlaceOptionOrder(t.Context(), &OptionOrderParam{
		Contract:    getPair(t, asset.Options),
		OrderSize:   -1,
		Iceberg:     0,
		Text:        "-",
		TimeInForce: "gtc",
		Price:       100,
	})
	assert.NoError(t, err)
}

func TestGetOptionFuturesOrders(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetOptionFuturesOrders(t.Context(), currency.EMPTYPAIR, "", "", 0, 0, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetOptionFuturesOrders(t.Context(), currency.EMPTYPAIR, "", "", 0, 0, time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestCancelOptionOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.CancelMultipleOptionOpenOrders(t.Context(), getPair(t, asset.Options), "", "")
	assert.NoError(t, err)
}

func TestGetSingleOptionOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetSingleOptionOrder(t.Context(), "")
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetSingleOptionOrder(t.Context(), "1234")
	assert.NoError(t, err)
}

func TestCancelSingleOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelOptionSingleOrder(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.CancelOptionSingleOrder(t.Context(), "1234")
	assert.NoError(t, err)
}

func TestGetMyOptionsTradingHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetMyOptionsTradingHistory(t.Context(), "", currency.EMPTYPAIR, 0, 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errInvalidUnderlying)

	startTime, endTime := getTime()
	_, err = e.GetMyOptionsTradingHistory(t.Context(), "BTC_USDT", currency.EMPTYPAIR, 0, 0, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetMyOptionsTradingHistory(t.Context(), "BTC_USDT", currency.EMPTYPAIR, 0, 0, time.Time{}, time.Time{})
	require.NoError(t, err)
}

func TestWithdrawCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawCurrency(t.Context(), &WithdrawalRequestParam{})
	assert.ErrorIs(t, err, order.ErrAmountIsInvalid)

	_, err = e.WithdrawCurrency(t.Context(), &WithdrawalRequestParam{Amount: 0.000000001})
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.WithdrawCurrency(t.Context(), &WithdrawalRequestParam{Amount: 1, Currency: currency.BTC})
	assert.ErrorIs(t, err, errInvalidCurrencyChain)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.WithdrawCurrency(t.Context(), &WithdrawalRequestParam{
		Currency: currency.BTC,
		Amount:   0.00000001,
		Chain:    "ERC20",
		Address:  core.BitcoinDonationAddress,
	})
	assert.NoError(t, err)
}

func TestCancelWithdrawalWithSpecifiedID(t *testing.T) {
	t.Parallel()
	_, err := e.CancelWithdrawalWithSpecifiedID(t.Context(), "")
	require.ErrorIs(t, err, errMissingWithdrawalID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.CancelWithdrawalWithSpecifiedID(t.Context(), "1234567")
	assert.NoError(t, err)
}

func TestGetOptionsOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.GetOptionsOrderbook(t.Context(), currency.EMPTYPAIR, "0.1", 9, true)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	result, err := e.GetOptionsOrderbook(t.Context(), getPair(t, asset.Options), "0.1", 9, true)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOptionsTickers(t *testing.T) {
	t.Parallel()
	_, err := e.GetOptionsTickers(t.Context(), "")
	require.ErrorIs(t, err, errInvalidUnderlying)

	_, err = e.GetOptionsTickers(t.Context(), "BTC_USDT")
	assert.NoError(t, err)
}

func TestGetOptionUnderlyingTickers(t *testing.T) {
	t.Parallel()
	_, err := e.GetOptionUnderlyingTickers(t.Context(), "")
	require.ErrorIs(t, err, errInvalidUnderlying)

	_, err = e.GetOptionUnderlyingTickers(t.Context(), "BTC_USDT")
	assert.NoError(t, err)
}

func TestGetOptionFuturesCandlesticks(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetOptionFuturesCandlesticks(t.Context(), currency.EMPTYPAIR, kline.ThirtyMin, startTime, endTime, 10)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.GetOptionFuturesCandlesticks(t.Context(), getPair(t, asset.Options), kline.ThirtyMin, endTime, startTime, 100)
	assert.ErrorIs(t, err, common.ErrStartAfterEnd)

	_, err = e.GetOptionFuturesCandlesticks(t.Context(), getPair(t, asset.Options), kline.ThirtyMin, startTime, endTime, 1000)
	assert.NoError(t, err)
}

func TestGetOptionFuturesMarkPriceCandlesticks(t *testing.T) {
	t.Parallel()
	_, err := e.GetOptionFuturesMarkPriceCandlesticks(t.Context(), "", 0, time.Time{}, time.Time{}, kline.OneMonth)
	require.ErrorIs(t, err, errInvalidUnderlying)

	startTime, endTime := getTime()
	_, err = e.GetOptionFuturesMarkPriceCandlesticks(t.Context(), "BTC_USDT", 0, endTime, startTime, kline.OneMonth)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	_, err = e.GetOptionFuturesMarkPriceCandlesticks(t.Context(), "BTC_USDT", 0, time.Time{}, time.Time{}, kline.OneMonth)
	assert.NoError(t, err)
}

func TestGetOptionsTradeHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetOptionsTradeHistory(t.Context(), getPair(t, asset.Options), "C", 0, 0, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	_, err = e.GetOptionsTradeHistory(t.Context(), getPair(t, asset.Options), "C", 0, 0, startTime, endTime)
	assert.NoError(t, err)
}

// Sub-account endpoints

func TestCreateNewSubAccount(t *testing.T) {
	t.Parallel()
	_, err := e.CreateNewSubAccount(t.Context(), &SubAccountParams{})
	assert.ErrorIs(t, err, errInvalidSubAccount)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.CreateNewSubAccount(t.Context(), &SubAccountParams{
		SubAccountName: "Sub_Account_for_testing",
	})
	assert.NoError(t, err)
}

func TestGetSubAccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSubAccounts(t.Context())
	assert.NoError(t, err)
}

func TestGetSingleSubAccount(t *testing.T) {
	t.Parallel()
	_, err := e.GetSingleSubAccount(t.Context(), "")
	assert.ErrorIs(t, err, errInvalidSubAccount)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetSingleSubAccount(t.Context(), "123423")
	assert.NoError(t, err)
}

// Wrapper test functions

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	for _, a := range e.GetAssetTypes(false) {
		pairs, err := e.FetchTradablePairs(t.Context(), a)
		require.NoErrorf(t, err, "FetchTradablePairs must not error for %s", a)
		require.NotEmptyf(t, pairs, "FetchTradablePairs must return some pairs for %s", a)

		if a == asset.USDTMarginedFutures || a == asset.CoinMarginedFutures {
			for _, p := range pairs {
				_, err := getSettlementCurrency(p, a)
				require.NoErrorf(t, err, "Fetched pair %s %s must not error on getSettlementCurrency", a, p)
			}
		}
	}
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	for _, a := range e.GetAssetTypes(false) {
		err := e.UpdateTickers(t.Context(), a)
		assert.NoErrorf(t, err, "UpdateTickers should not error for %s", a)
	}
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateOrderbook(t.Context(), currency.EMPTYPAIR, 1336)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	for _, a := range e.GetAssetTypes(false) {
		pair := getPair(t, a)
		t.Run(a.String()+" "+pair.String(), func(t *testing.T) {
			t.Parallel()
			o, err := e.UpdateOrderbook(t.Context(), pair, a)
			require.NoError(t, err)
			if a != asset.Options { // Options orderbooks can be empty
				assert.NotEmpty(t, o.Bids)
				assert.NotEmpty(t, o.Asks)
			}
		})
	}
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Empty)
	assert.NoError(t, err)
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	for _, a := range e.GetAssetTypes(false) {
		if a != asset.CoinMarginedFutures {
			_, err := e.GetRecentTrades(t.Context(), getPair(t, a), a)
			assert.NoErrorf(t, err, "GetRecentTrades should not error for %s", a)
		}
	}
}

func TestSubmitOrder(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	for _, a := range e.GetAssetTypes(false) {
		_, err := e.SubmitOrder(t.Context(), &order.Submit{
			Exchange:    e.Name,
			Pair:        getPair(t, a),
			Side:        order.Buy,
			Type:        order.Limit,
			Price:       1,
			Amount:      1,
			AssetType:   a,
			TimeInForce: order.GoodTillCancel,
		})
		assert.NoErrorf(t, err, "SubmitOrder should not error for %s", a)
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	for _, a := range e.GetAssetTypes(false) {
		orderCancellation := &order.Cancel{
			OrderID:   "1",
			AccountID: "1",
			Pair:      getPair(t, a),
			AssetType: a,
		}
		err := e.CancelOrder(t.Context(), orderCancellation)
		assert.NoErrorf(t, err, "CancelOrder should not error for %s", a)
	}
}

func TestCancelBatchOrders(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	for _, a := range e.GetAssetTypes(false) {
		_, err := e.CancelBatchOrders(t.Context(), []order.Cancel{
			{
				OrderID:   "1",
				AccountID: "1",
				Pair:      getPair(t, a),
				AssetType: a,
			}, {
				OrderID:   "2",
				AccountID: "1",
				Pair:      getPair(t, a),
				AssetType: a,
			},
		})
		assert.NoErrorf(t, err, "CancelBatchOrders should not error for %s", a)
	}
}

func TestGetDepositAddress(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	chains, err := e.GetAvailableTransferChains(t.Context(), currency.BTC)
	require.NoError(t, err)

	for i := range chains {
		_, err = e.GetDepositAddress(t.Context(), currency.BTC, "", chains[i])
		assert.NoErrorf(t, err, "GetDepositAddress error: %v", err)
	}
}

func TestGetActiveOrders(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	for _, a := range e.GetAssetTypes(false) {
		enabledPairs := getPairs(t, a)
		if len(enabledPairs) > 2 {
			enabledPairs = enabledPairs[:2]
		}
		_, err := e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{
			Pairs:     enabledPairs,
			Type:      order.AnyType,
			Side:      order.AnySide,
			AssetType: a,
		})
		assert.NoErrorf(t, err, "GetActiveOrders should not error for %s", a)
	}
}

func TestGetOrderHistory(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	for _, a := range e.GetAssetTypes(false) {
		enabledPairs := getPairs(t, a)
		if len(enabledPairs) > 4 {
			enabledPairs = enabledPairs[:4]
		}
		multiOrderRequest := order.MultiOrderRequest{
			Type:      order.AnyType,
			Side:      order.Buy,
			Pairs:     enabledPairs,
			AssetType: a,
		}
		_, err := e.GetOrderHistory(t.Context(), &multiOrderRequest)
		assert.NoErrorf(t, err, "GetOrderHistory should not error for %s", a)
	}
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	for _, a := range e.GetAssetTypes(false) {
		_, err := e.GetHistoricCandles(t.Context(), getPair(t, a), a, kline.OneDay, startTime, endTime)
		if a == asset.Options {
			assert.ErrorIs(t, err, asset.ErrNotSupported, "GetHistoricCandles should error correctly for options")
		} else {
			assert.NoErrorf(t, err, "GetHistoricCandles should not error for %s", a)
		}
	}
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	for _, a := range e.GetAssetTypes(false) {
		_, err := e.GetHistoricCandlesExtended(t.Context(), getPair(t, a), a, kline.OneMin, startTime, endTime)
		if a == asset.Options {
			assert.ErrorIs(t, err, asset.ErrNotSupported, "GetHistoricCandlesExtended should error correctly for options")
		} else {
			assert.NoErrorf(t, err, "GetHistoricCandlesExtended should not error for %s", a)
		}
	}
}

func TestGetAvailableTransferTrains(t *testing.T) {
	t.Parallel()
	_, err := e.GetAvailableTransferChains(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.GetAvailableTransferChains(t.Context(), currency.USDT)
	assert.NoError(t, err)
}

func TestGetUnderlyingFromCurrencyPair(t *testing.T) {
	t.Parallel()
	uly, err := e.GetUnderlyingFromCurrencyPair(currency.Pair{Delimiter: currency.UnderscoreDelimiter, Base: currency.BTC, Quote: currency.NewCode("USDT_LLK")})
	require.NoError(t, err)
	assert.True(t, uly.Equal(currency.NewBTCUSDT()))
}

var mapSpotPublicPushData = map[string]string{
	"spot.tickers":           `{"time": 1606291803,	"channel": "spot.tickers",	"event": "update",	"result": {	  "currency_pair": "BTC_USDT",	  "last": "19106.55",	  "lowest_ask": "19108.71",	  "highest_bid": "19106.55",	  "change_percentage": "3.66",	  "base_volume": "2811.3042155865",	  "quote_volume": "53441606.52411221454674732293",	  "high_24h": "19417.74",	  "low_24h": "18434.21"	}}`,
	"spot.trades":            `{"time": 1606292218,	"channel": "spot.trades",	"event": "update",	"result": {	  "id": 309143071,	  "create_time": 1606292218,	  "create_time_ms": "1606292218213.4578",	  "side": "sell",	  "currency_pair": "BTC_USDT",	  "amount": "16.4700000000",	  "price": "0.4705000000"}}`,
	"spot.candlesticks":      `{"time": 1606292600,	"channel": "spot.candlesticks",	"event": "update",	"result": {	  "t": "1606292580",	  "v": "2362.32035",	  "c": "19128.1",	  "h": "19128.1",	  "l": "19128.1",	  "o": "19128.1","n": "1m_BTC_USDT"}}`,
	"spot.book_ticker":       `{"time": 1606293275,	"channel": "spot.book_ticker",	"event": "update",	"result": {	  "t": 1606293275123,	  "u": 48733182,	  "s": "BTC_USDT",	  "b": "19177.79",	  "B": "0.0003341504",	  "a": "19179.38",	  "A": "0.09"	}}`,
	"spot.order_book_update": `{"time": 1606294781,	"channel": "spot.order_book_update",	"event": "update",	"result": {	  "t": 1606294781123,	  "e": "depthUpdate",	  "E": 1606294781,"s": "BTC_USDT","U": 48776301,"u": 48776306,"b": [["19137.74","0.0001"],["19088.37","0"]],"a": [["19137.75","0.6135"]]	}}`,
	"spot.order_book":        `{"time":1606295412,"channel": "spot.order_book",	"event": "update",	"result": {	  "t": 1606295412123,	  "lastUpdateId": 48791820,	  "s": "BTC_USDT",	  "bids": [		[		  "19079.55",		  "0.0195"		],		[		  "19079.07",		  "0.7341"],["19076.23",		  "0.00011808"		],		[		  "19073.9",		  "0.105"		],		[		  "19068.83",		  "0.1009"		]	  ],	  "asks": [		[		  "19080.24",		  "0.1638"		],		[		  "19080.91","0.1366"],["19080.92","0.01"],["19081.29","0.01"],["19083.8","0.097"]]}}`,
	"spot.orders":            `{"time": 1605175506,	"channel": "spot.orders",	"event": "update",	"result": [	  {		"id": "30784435",		"user": 123456,		"text": "t-abc",		"create_time": "1605175506",		"create_time_ms": "1605175506123",		"update_time": "1605175506",		"update_time_ms": "1605175506123",		"event": "put",		"currency_pair": "BTC_USDT",		"type": "limit",		"account": "spot",		"side": "sell",		"amount": "1",		"price": "10001",		"time_in_force": "gtc",		"left": "1",		"filled_total": "0",		"fee": "0",		"fee_currency": "USDT",		"point_fee": "0",		"gt_fee": "0",		"gt_discount": true,		"rebated_fee": "0",		"rebated_fee_currency": "USDT"}	]}`,
	"spot.usertrades":        `{"time": 1605176741,	"channel": "spot.usertrades",	"event": "update",	"result": [	  {		"id": 5736713,		"user_id": 1000001,		"order_id": "30784428",		"currency_pair": "BTC_USDT",		"create_time": 1605176741,		"create_time_ms": "1605176741123.456",		"side": "sell",		"amount": "1.00000000",		"role": "taker",		"price": "10000.00000000",		"fee": "0.00200000000000",		"point_fee": "0",		"gt_fee": "0",		"text": "apiv4"	  }	]}`,
	"spot.cross_loan":        `{"time":1658289372,	"channel":"spot.cross_loan",	"event":"update",	"result":{	  "timestamp":1658289372338,	  "user":"1000001",	  "currency":"BTC",	  "change":"0.01",	  "total":"4.992341029566",	  "available":"0.078054772536",	  "borrowed":"0.01",	  "interest":"0.00001375"	}}`,
}

func TestWsHandleSpotData(t *testing.T) {
	t.Parallel()
	for k, v := range mapSpotPublicPushData {
		t.Run(k, func(t *testing.T) {
			t.Parallel()
			err := e.WsHandleSpotData(t.Context(), nil, []byte(v))
			require.NoError(t, err)
		})
	}
}

var mapSpotPrivatePushData = map[string]string{
	"spot.balances":         `{"time": 1605248616,	"channel": "spot.balances",	"event": "update",	"result": [	  {		"timestamp": "1605248616",		"timestamp_ms": "1605248616123",		"user": "1000001",		"currency": "USDT",		"change": "100",		"total": "1032951.325075926","available": "1022943.325075926"}	]}`,
	"spot.funding_balances": `{"time": 1605248616,	"channel": "spot.funding_balances",	"event": "update",	"result": [	  {"timestamp": "1605248616","timestamp_ms": "1605248616123","user": "1000001","currency": "USDT","change": "100","freeze": "100","lent": "0"}	]}`,
	"spot.cross_balances":   `{"time": 1605248616,"channel": "spot.cross_balances","event": "update",	"result": [{"timestamp": "1605248616","timestamp_ms": "1605248616123","user": "1000001","currency": "USDT",	"change": "100","total": "1032951.325075926","available": "1022943.325075926"}]}`,
}

func TestWsHandleSpotDataPrivate(t *testing.T) {
	t.Parallel()
	ctx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "test", Secret: "test"})
	for x := range mapSpotPrivatePushData {
		err := e.WsHandleSpotData(ctx, nil, []byte(mapSpotPrivatePushData[x]))
		require.NoError(t, err)
	}
}

// TestFuturesDataHandler ensures that messages from various futures channels do not error
func TestFuturesDataHandler(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")
	testexch.FixtureToDataHandler(t, "testdata/wsFutures.json", func(ctx context.Context, m []byte) error {
		if strings.Contains(string(m), "futures.balances") {
			ctx = accounts.DeployCredentialsToContext(ctx, &accounts.Credentials{Key: "test", Secret: "test"})
		}
		return e.WsHandleFuturesData(ctx, nil, m, asset.CoinMarginedFutures)
	})
	e.Websocket.DataHandler.Close()
	assert.Len(t, e.Websocket.DataHandler.C, 14, "Should see the correct number of messages")
	for resp := range e.Websocket.DataHandler.C {
		if err, isErr := resp.Data.(error); isErr {
			assert.NoError(t, err, "Should not get any errors down the data handler")
		}
	}
}

// ******************************************** Options web-socket unit test funcs ********************

var wsOptionsPushDataHandler = map[string]string{
	"options.contract_tickers":      `{"time": 1630576352,	"channel": "options.contract_tickers",	"event": "update",	"result": {"name": "BTC_USDT-20211231-59800-P",    "last_price": "11349.5",    "mark_price": "11170.19",    "index_price": "",    "position_size": 993,    "bid1_price": "10611.7",    "bid1_size": 100,    "ask1_price": "11728.7",    "ask1_size": 100,    "vega": "34.8731",    "theta": "-72.80588",    "rho": "-28.53331",    "gamma": "0.00003",    "delta": "-0.78311",    "mark_iv": "0.86695",    "bid_iv": "0.65481",    "ask_iv": "0.88145",    "leverage": "3.5541112718136"	}}`,
	"options.ul_tickers":            `{"time": 1630576352,	"channel": "options.ul_tickers",	"event": "update",	"result": {"trade_put": 800,	   "trade_call": 41700,	   "index_price": "50695.43",	   "name": "BTC_USDT"	}}`,
	"options.trades":                `{"time": 1630576356,	"channel": "options.trades",	"event": "update",	"result": [{"contract": "BTC_USDT-20211231-59800-C",        "create_time": 1639144526,        "id": 12279,        "price": 997.8,        "size": -100,        "create_time_ms": 1639144526597,        "underlying": "BTC_USDT"    }	]}`,
	"options.ul_trades":             `{"time": 1630576356,	"channel": "options.ul_trades",	"event": "update",	"result": [{"contract": "BTC_USDT-20211231-59800-C","create_time": 1639144526,"id": 12279,"price": 997.8,"size": -100,"create_time_ms": 1639144526597,"underlying": "BTC_USDT","is_call": true}	]}`,
	"options.ul_price":              `{"time": 1630576356,	"channel": "options.ul_price",	"event": "update",	"result": {"underlying": "BTC_USDT",	   "price": 49653.24,"time": 1639143988,"time_ms": 1639143988931}}`,
	"options.mark_price":            `{"time": 1630576356,	"channel": "options.mark_price",	"event": "update",	"result": {"contract": "BTC_USDT-20211231-59800-P",    "price": 11021.27,    "time": 1639143401,    "time_ms": 1639143401676}}`,
	"options.settlements":           `{"time": 1630576356,	"channel": "options.settlements",	"event": "update",	"result": {"contract": "BTC_USDT-20211130-55000-P",	   "orderbook_id": 2,	   "position_size": 1,	   "profit": 0.5,"settle_price": 70000,	   "strike_price": 65000,	   "tag": "WEEK",	   "trade_id": 1,	   "trade_size": 1,	   "underlying": "BTC_USDT",	   "time": 1639051907,	   "time_ms": 1639051907000}}`,
	"options.contracts":             `{"time": 1630576356,	"channel": "options.contracts",	"event": "update",	"result": {	   "contract": "BTC_USDT-20211130-50000-P",	   "create_time": 1637917026,	   "expiration_time": 1638230400,"init_margin_high": 0.15,	   "init_margin_low": 0.1,	   "is_call": false,	   "maint_margin_base": 0.075,	   "maker_fee_rate": 0.0004,	   "mark_price_round": 0.1,	   "min_balance_short": 0.5,	   "min_order_margin": 0.1,	   "multiplier": 0.0001,	   "order_price_deviate": 0,	   "order_price_round": 0.1,	   "order_size_max": 1,	   "order_size_min": 10,	   "orders_limit": 100000,	   "ref_discount_rate": 0.1,	   "ref_rebate_rate": 0,	   "strike_price": 50000,	   "tag": "WEEK",	   "taker_fee_rate": 0.0004,	   "underlying": "BTC_USDT",	   "time": 1639051907,	   "time_ms": 1639051907000}}`,
	"options.contract_candlesticks": `{	"time": 1630650451,	"channel": "options.contract_candlesticks",	"event": "update",	"result": [{"t": 1639039260,       "v": 100,       "c": "1041.4",       "h": "1041.4",       "l": "1041.4","o": "1041.4",       "a": "0",       "n": "10s_BTC_USDT-20211231-59800-C"   }	]}`,
	"options.ul_candlesticks":       `{"time": 1630650451,	"channel": "options.ul_candlesticks",	"event": "update",	"result": [{"t": 1639039260,        "v": 100,        "c": "1041.4",        "h": "1041.4",        "l": "1041.4","o": "1041.4",        "a": "0",        "n": "10s_BTC_USDT"    }	]}`,
	"options.book_ticker":           `{"time": 1630650452,	"channel": "options.book_ticker",	"event": "update",	"result": {    "t": 1615366379123,    "u": 2517661076,    "s": "BTC_USDT-20211130-50000-C",    "b": "54696.6",    "B": 37000,    "a": "54696.7",    "A": 47061	}}`,
	"options.order_book_update":     `{"time": 1630650445,	"channel": "options.order_book_update",	"event": "update",	"result": {    "t": 1615366381417,    "s": "BTC_USDT-20211130-50000-C",    "U": 2517661101,    "u": 2517661113,    "b": [{"p": "54672.1","s": 95        },        {            "p": "54664.5",            "s": 58794        }    ],    "a": [        {            "p": "54743.6",            "s": 95        },        {            "p": "54742",            "s": 95        }    ]	}}`,
	"options.order_book":            `{"time": 1630650445,	"channel": "options.order_book",	"event": "all",	"result": {"t": 1541500161123,    "contract": "BTC_USDT-20211130-50000-C",    "id": 93973511,    "asks": [{"p": "97.1","s": 2245},{"p": "97.2",            "s": 2245        }    ],    "bids": [		{            "p": "97.2",            "s": 2245        },        {            "p": "97.1",            "s": 2245        }    ]	}}`,
	"options.order_book-1":          `{"channel": "options.order_book",	"event": "update",	"time": 1630650445,	"result": [{"p": "49525.6",		"s": 7726,		"c": "BTC_USDT-20211130-50000-C",		"id": 93973511	  }	]}`,
	"options.book_ticker-2":         `{"time": 1630650452,	"channel": "options.book_ticker",	"event": "update",	"result": {"t": 1615366379123,    "u": 2517661076,    "s": "BTC_USDT-20211130-50000-C",    "b": "54696.6",    "B": 37000,    "a": "54696.7",    "A": 47061	}}`,
	"options.order_book_update-3":   `{"time": 1630650445,	"channel": "options.order_book_update",	"event": "update",	"result": {"t": 1615366381417,    "s": "BTC_USDT-20211130-50000-C",    "U": 2517661101,"u": 2517661113,    "b": [{"p": "54672.1","s": 95},{"p": "54664.5","s": 58794}],"a": [{"p": "54743.6","s": 95},{"p": "54742","s": 95}]}}`,
	"options.order_book-4":          `{"time": 1630650445,	"channel": "options.order_book",	"event": "all",	"result": {"t": 1541500161123,    "contract": "BTC_USDT-20211130-50000-C",    "id": 93973511,    "asks": [{"p": "97.1","s": 2245},{"p": "97.2","s": 2245}    ],    "bids": [{"p": "97.2",            "s": 2245        },        {            "p": "97.1",            "s": 2245        }    ]	}}`,
	"options.order_book-5":          `{"channel": "options.order_book",	"event": "update",	"time": 1630650445,	"result": [{"p": "49525.6",		"s": 7726,		"c": "BTC_USDT-20211130-50000-C",		"id": 93973511	  }	]}`,
	"options.liquidates":            `{"channel": "options.liquidates",	"event": "update",	"time": 1630654851,	"result": [{"user": "1xxxx",		  "init_margin": 1190,		  "maint_margin": 1042.5,		  "order_margin": 0,		  "time": 1639051907,		  "time_ms": 1639051907000}	]}`,
	"options.positions":             `{"time": 1630654851,	"channel": "options.positions",	"event": "update",	"error": null,	"result": [	   {		  "entry_price": 0,		  "realised_pnl": -13.028,		  "size": 0,		  "contract": "BTC_USDT-20211130-65000-C",		  "user": "9010",		  "time": 1639051907,		  "time_ms": 1639051907000}	]}`,
	"options.pong":                  `{"time":1756700469,"channel":"options.pong","event":"","result":null}`,
}

func TestWsHandleOptionsData(t *testing.T) {
	t.Parallel()
	for k, v := range wsOptionsPushDataHandler {
		t.Run(k, func(t *testing.T) {
			t.Parallel()
			err := e.WsHandleOptionsData(t.Context(), nil, []byte(v))
			require.NoError(t, err)
		})
	}
}

var wsOptionsAuthenticatedDataHandler = map[string]string{
	"options.orders":           `{"time": 1630654851,"channel": "options.orders",	"event": "update",	"result": [{"contract": "BTC_USDT-20211130-65000-C",		  "create_time": 1637897000,		  "fill_price": 0,"finish_as": "cancelled",		  "iceberg": 0,		  "id": 106,		  "is_close": false,		  "is_liq": false,		  "is_reduce_only": false,		  "left": -10,		  "mkfr": 0.0004,		  "price": 15000,		  "refr": 0,		  "refu": 0,		  "size": -10,		  "status": "finished",		  "text": "web",		  "tif": "gtc",		  "tkfr": 0.0004,		  "underlying": "BTC_USDT",		  "user": "9xxx",		  "time": 1639051907,"time_ms": 1639051907000}]}`,
	"options.usertrades":       `{"time": 1639144214,	"channel": "options.usertrades",	"event": "update",	"result": [{"id": "1","underlying": "BTC_USDT","order": "557940","contract": "BTC_USDT-20211216-44800-C","create_time": 1639144214,"create_time_ms": 1639144214583,"price": "4999","role": "taker","size": -1}]}`,
	"options.user_settlements": `{"channel": "options.user_settlements","event": "update",	"time": 1639051907,	"result": [{"contract": "BTC_USDT-20211130-65000-C","realised_pnl": -13.028,"settle_price": 70000,"settle_profit": 5,"size": 10,"strike_price": 65000,"underlying": "BTC_USDT","user": "9xxx","time": 1639051907,"time_ms": 1639051907000}]}`,
	"options.position_closes":  `{"channel": "options.position_closes","event": "update",	"time": 1630654851,	"result": [{"contract": "BTC_USDT-20211130-50000-C","pnl": -0.0056,"settle_size": 0,"side": "long","text": "web","underlying": "BTC_USDT","user": "11xxxxx","time": 1639051907,"time_ms": 1639051907000}]}`,
	"options.balances":         `{"channel": "options.balances",	"event": "update",	"time": 1630654851,	"result": [{"balance": 60.79009,"change": -0.5,"text": "BTC_USDT-20211130-55000-P","type": "set","user": "11xxxx","time": 1639051907,"time_ms": 1639051907000}]}`,
}

func TestPrivateWsHandleOptionsData(t *testing.T) {
	t.Parallel()
	for k, v := range wsOptionsAuthenticatedDataHandler {
		t.Run(k, func(t *testing.T) {
			t.Parallel()
			ctx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "test", Secret: "test"})
			err := e.WsHandleOptionsData(ctx, nil, []byte(v))
			require.NoError(t, err)
		})
	}
}

func TestGenerateSubscriptionsSpot(t *testing.T) {
	t.Parallel()

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

	e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	subs, err := e.generateSubscriptionsSpot()
	require.NoError(t, err, "generateSubscriptions must not error")
	exp := subscription.List{}
	assets := slices.DeleteFunc(e.GetAssetTypes(true), func(a asset.Item) bool { return !e.IsAssetWebsocketSupported(a) })
	for _, s := range e.Features.Subscriptions {
		for _, a := range assets {
			if s.Asset != asset.All && s.Asset != a {
				continue
			}
			pairs, err := e.GetEnabledPairs(a)
			require.NoErrorf(t, err, "GetEnabledPairs %s must not error", a)
			pairs = common.SortStrings(pairs).Format(currency.PairFormat{Uppercase: true, Delimiter: "_"})
			s := s.Clone() //nolint:govet // Intentional lexical scope shadow
			s.Asset = a
			if singleSymbolChannel(channelName(s)) {
				for i := range pairs {
					s := s.Clone() //nolint:govet // Intentional lexical scope shadow
					switch s.Channel {
					case subscription.CandlesChannel:
						s.QualifiedChannel = "5m," + pairs[i].String()
					case subscription.OrderbookChannel:
						s.QualifiedChannel = pairs[i].String() + ",100ms"
					case spotOrderbookChannel:
						s.QualifiedChannel = pairs[i].String() + ",5,1000ms"
					}
					s.Pairs = pairs[i : i+1]
					exp = append(exp, s)
				}
			} else {
				s.Pairs = pairs
				s.QualifiedChannel = pairs.Join()
				exp = append(exp, s)
			}
		}
	}
	testsubs.EqualLists(t, exp, subs)
}

func TestSubscribe(t *testing.T) {
	t.Parallel()
	subs, err := e.Features.Subscriptions.ExpandTemplates(e)
	require.NoError(t, err, "ExpandTemplates must not error")
	e.Features.Subscriptions = subscription.List{}
	err = e.Subscribe(t.Context(), &FixtureConnection{}, subs)
	require.NoError(t, err, "Subscribe must not error")
}

func TestGenerateDeliveryFuturesDefaultSubscriptions(t *testing.T) {
	t.Parallel()
	_, err := e.GenerateDeliveryFuturesDefaultSubscriptions(asset.DeliveryFutures)
	assert.NoError(t, err)
}

func TestGenerateFuturesDefaultSubscriptions(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")
	subs, err := e.GenerateFuturesDefaultSubscriptions(asset.USDTMarginedFutures)
	require.NoError(t, err)
	require.NotEmpty(t, subs)
	subs, err = e.GenerateFuturesDefaultSubscriptions(asset.CoinMarginedFutures)
	require.NoError(t, err)
	require.NotEmpty(t, subs)
	require.NoError(t, e.CurrencyPairs.SetAssetEnabled(asset.USDTMarginedFutures, false), "SetAssetEnabled must not error")
	subs, err = e.GenerateFuturesDefaultSubscriptions(asset.USDTMarginedFutures)
	require.NoError(t, err, "Disabled asset must not error")
	require.Empty(t, subs, "Disabled asset must return no pairs")
}

func TestGenerateOptionsDefaultSubscriptions(t *testing.T) {
	t.Parallel()
	_, err := e.GenerateOptionsDefaultSubscriptions()
	assert.NoError(t, err)
}

func TestCreateAPIKeysOfSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.CreateAPIKeysOfSubAccount(t.Context(), &CreateAPIKeySubAccountParams{
		SubAccountUserID: 12345,
		Body: &SubAccountKey{
			APIKeyName: "12312mnfsndfsfjsdklfjsdlkfj",
			Permissions: []APIV4KeyPerm{
				{
					PermissionName: "wallet",
					ReadOnly:       false,
				},
				{
					PermissionName: "spot",
					ReadOnly:       false,
				},
				{
					PermissionName: "futures",
					ReadOnly:       false,
				},
				{
					PermissionName: "delivery",
					ReadOnly:       false,
				},
				{
					PermissionName: "earn",
					ReadOnly:       false,
				},
				{
					PermissionName: "options",
					ReadOnly:       false,
				},
			},
		},
	})
	assert.NoError(t, err)
}

func TestListAllAPIKeyOfSubAccount(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllAPIKeyOfSubAccount(t.Context(), 0)
	require.ErrorIs(t, err, errUserIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetAllAPIKeyOfSubAccount(t.Context(), 1234)
	assert.NoError(t, err)
}

func TestUpdateAPIKeyOfSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err := e.UpdateAPIKeyOfSubAccount(t.Context(), apiKey, CreateAPIKeySubAccountParams{
		SubAccountUserID: 12345,
		Body: &SubAccountKey{
			APIKeyName: "12312mnfsndfsfjsdklfjsdlkfj",
			Permissions: []APIV4KeyPerm{
				{
					PermissionName: "wallet",
					ReadOnly:       false,
				},
				{
					PermissionName: "spot",
					ReadOnly:       false,
				},
				{
					PermissionName: "futures",
					ReadOnly:       false,
				},
				{
					PermissionName: "delivery",
					ReadOnly:       false,
				},
				{
					PermissionName: "earn",
					ReadOnly:       false,
				},
				{
					PermissionName: "options",
					ReadOnly:       false,
				},
			},
		},
	})
	assert.NoError(t, err)
}

func TestDeleteSubAccountAPIKeyPair(t *testing.T) {
	t.Parallel()
	err := e.DeleteSubAccountAPIKeyPair(t.Context(), 0, "12312mnfsndfsfjsdklfjsdlkfj")
	require.ErrorIs(t, err, errInvalidSubAccountUserID)

	err = e.DeleteSubAccountAPIKeyPair(t.Context(), 12345, "")
	require.ErrorIs(t, err, errMissingAPIKey)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	err = e.DeleteSubAccountAPIKeyPair(t.Context(), 12345, "12312mnfsndfsfjsdklfjsdlkfj")
	require.NoError(t, err)
}

func TestGetAPIKeyOfSubAccount(t *testing.T) {
	t.Parallel()
	_, err := e.GetAPIKeyOfSubAccount(t.Context(), 0, "12312mnfsndfsfjsdklfjsdlkfj")
	require.ErrorIs(t, err, errInvalidSubAccountUserID)

	_, err = e.GetAPIKeyOfSubAccount(t.Context(), 12345, "")
	require.ErrorIs(t, err, errMissingAPIKey)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetAPIKeyOfSubAccount(t.Context(), 1234, "target_api_key")
	assert.NoError(t, err)
}

func TestLockSubAccount(t *testing.T) {
	t.Parallel()
	err := e.LockSubAccount(t.Context(), 0)
	require.ErrorIs(t, err, errInvalidSubAccountUserID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	err = e.LockSubAccount(t.Context(), 1234)
	assert.NoError(t, err)
}

func TestUnlockSubAccount(t *testing.T) {
	t.Parallel()
	err := e.UnlockSubAccount(t.Context(), 0)
	require.ErrorIs(t, err, errInvalidSubAccountUserID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.UnlockSubAccount(t.Context(), 1234)
	assert.NoError(t, err)
}

func TestGetSubAccountMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSubAccountMode(t.Context())
	assert.NoError(t, err)
}

func TestParseGateioMilliSecTimeUnmarshal(t *testing.T) {
	t.Parallel()
	var timeWhenTesting int64 = 1684981731098
	timeWhenTestingString := `"1684981731098"` // Normal string
	integerJSON := `{"number": 1684981731098}`
	float64JSON := `{"number": 1684981731.098}`

	tm := time.UnixMilli(timeWhenTesting)
	var in types.Time
	err := json.Unmarshal([]byte(timeWhenTestingString), &in)
	require.NoError(t, err)
	require.Equalf(t, tm, in.Time(), "found %v, but expected %v", in.Time(), tm)

	var inInteger struct {
		Number types.Time `json:"number"`
	}
	err = json.Unmarshal([]byte(integerJSON), &inInteger)
	require.NoError(t, err)
	require.Equalf(t, tm, inInteger.Number.Time(), "found %v, but expected %v", inInteger.Number.Time(), tm)

	var inFloat64 struct {
		Number types.Time `json:"number"`
	}
	err = json.Unmarshal([]byte(float64JSON), &inFloat64)
	require.NoError(t, err)
	require.Equalf(t, tm, inFloat64.Number.Time(), "found %v, but expected %v", inFloat64.Number.Time(), tm)
}

func TestParseTimeUnmarshal(t *testing.T) {
	t.Parallel()
	var timeWhenTesting int64 = 1684981731
	timeWhenTestingString := `"1684981731"`
	integerJSON := `{"number": 1684981731}`
	float64JSON := `{"number": 1684981731.234}`
	timeWhenTestingStringMicroSecond := `"1691122380942.173000"`

	whenTime := time.Unix(timeWhenTesting, 0)
	var in types.Time
	err := json.Unmarshal([]byte(timeWhenTestingString), &in)
	require.NoError(t, err)
	require.Equalf(t, whenTime, in.Time(), "found %v, but expected %v", in.Time(), whenTime)

	var inInteger struct {
		Number types.Time `json:"number"`
	}
	err = json.Unmarshal([]byte(integerJSON), &inInteger)
	require.NoError(t, err)
	require.Equalf(t, whenTime, inInteger.Number.Time(), "found %v, but expected %v", inInteger.Number.Time(), whenTime)

	var inFloat64 struct {
		Number types.Time `json:"number"`
	}
	err = json.Unmarshal([]byte(float64JSON), &inFloat64)
	require.NoError(t, err)

	msTime := time.UnixMilli(1684981731234)
	require.Equalf(t, time.UnixMilli(1684981731234), inFloat64.Number.Time(), "found %v, but expected %v", inFloat64.Number.Time(), msTime)

	var microSeconds types.Time
	err = json.Unmarshal([]byte(timeWhenTestingStringMicroSecond), &microSeconds)
	require.NoError(t, err)
	require.Equalf(t, time.UnixMicro(1691122380942173), microSeconds.Time(), "found %v, but expected %v", microSeconds.Time(), time.UnixMicro(1691122380942173))
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, e)

	for _, a := range e.GetAssetTypes(false) {
		t.Run(a.String(), func(t *testing.T) {
			t.Parallel()
			switch a {
			case asset.Options:
				return // Options not supported
			case asset.CrossMargin, asset.Margin:
				require.ErrorIs(t, e.UpdateOrderExecutionLimits(t.Context(), a), asset.ErrNotSupported)
			default:
				require.NoError(t, e.UpdateOrderExecutionLimits(t.Context(), a), "UpdateOrderExecutionLimits must not error")
				avail, err := e.GetAvailablePairs(a)
				require.NoError(t, err, "GetAvailablePairs must not error")
				for _, pair := range avail {
					l, err := e.GetOrderExecutionLimits(a, pair)
					require.NoErrorf(t, err, "GetOrderExecutionLimits must not error for %s", pair)
					require.NotNilf(t, l, "GetOrderExecutionLimits %s result cannot be nil", pair)
					assert.Equalf(t, a, l.Key.Asset, "asset should equal for %s", pair)
					assert.Truef(t, pair.Equal(l.Key.Pair()), "pair should equal for %s", pair)
					assert.Positivef(t, l.MinimumBaseAmount, "MinimumBaseAmount should be positive for %s", pair)
					assert.Positivef(t, l.AmountStepIncrementSize, "AmountStepIncrementSize should be positive for %s", pair)

					switch a {
					case asset.USDTMarginedFutures:
						assert.Positivef(t, l.MultiplierDecimal, "MultiplierDecimal should be positive for %s", pair)
						assert.NotZerof(t, l.Listed, "Listed should be populated for %s", pair)
						fallthrough
					case asset.CoinMarginedFutures:
						if !l.Delisted.IsZero() {
							assert.Truef(t, l.Delisted.After(l.Delisting), "Delisted should be after Delisting for %s", pair)
						}
					case asset.Spot:
						assert.Positivef(t, l.MinimumQuoteAmount, "MinimumQuoteAmount should be positive for %s", pair)
						assert.Positivef(t, l.QuoteStepIncrementSize, "QuoteStepIncrementSize should be positive for %s", pair)
					case asset.DeliveryFutures:
						assert.NotZerof(t, l.Expiry, "Expiry should be populated for %s", pair)
					}
				}
			}
		})
	}
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesContractDetails(t.Context(), asset.Spot)
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset)

	_, err = e.GetFuturesContractDetails(t.Context(), asset.PerpetualContract)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	exp, err := e.GetAllDeliveryContracts(t.Context(), currency.USDT)
	require.NoError(t, err, "GetAllDeliveryContracts must not error")
	c, err := e.GetFuturesContractDetails(t.Context(), asset.DeliveryFutures)
	require.NoError(t, err, "GetFuturesContractDetails must not error for DeliveryFutures")
	assert.Equal(t, len(exp), len(c), "GetFuturesContractDetails should return same number of Delivery contracts as exist")

	t.Skip()
	for _, a := range []asset.Item{asset.CoinMarginedFutures, asset.USDTMarginedFutures} {
		c, err = e.GetFuturesContractDetails(t.Context(), a)
		require.NoErrorf(t, err, "GetFuturesContractDetails must not error for %s", a)
		assert.NotEmptyf(t, c, "GetFuturesContractDetails should return some contracts for %s", a)
	}
}

func TestGetLatestFundingRates(t *testing.T) {
	t.Parallel()
	t.Skip()
	_, err := e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset:                asset.USDTMarginedFutures,
		Pair:                 currency.NewBTCUSDT(),
		IncludePredictedRate: true,
	})
	assert.NoError(t, err)

	_, err = e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset: asset.CoinMarginedFutures,
		Pair:  currency.NewBTCUSD(),
	})
	assert.NoError(t, err)

	_, err = e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.CoinMarginedFutures})
	assert.NoError(t, err)
	_, err = e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.USDTMarginedFutures})
	assert.NoError(t, err)
}

func TestGetHistoricalFundingRates(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricalFundingRates(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)

	_, err = e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{})
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.CoinMarginedFutures})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.Futures})
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
		Asset: asset.USDTMarginedFutures,
		Pair:  currency.NewPair(currency.ENJ, currency.USDT),
	})
	assert.ErrorIs(t, err, fundingrate.ErrPaymentCurrencyCannotBeEmpty)

	_, err = e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
		Asset:           asset.USDTMarginedFutures,
		Pair:            currency.NewPair(currency.ENJ, currency.USDT),
		PaymentCurrency: currency.USDT,
		IncludePayments: true,
	})
	assert.ErrorIs(t, err, common.ErrNotYetImplemented)

	_, err = e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
		Asset:                asset.USDTMarginedFutures,
		Pair:                 currency.NewPair(currency.ENJ, currency.USDT),
		PaymentCurrency:      currency.USDT,
		IncludePredictedRate: true,
	})
	assert.ErrorIs(t, err, common.ErrNotYetImplemented)

	startTime, endTime := getTime()
	_, err = e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
		Asset:           asset.USDTMarginedFutures,
		Pair:            currency.NewPair(currency.ENJ, currency.USDT),
		PaymentCurrency: currency.USDT,
		StartDate:       endTime,
		EndDate:         startTime,
	})
	assert.ErrorIs(t, err, common.ErrStartAfterEnd)

	_, err = e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
		Asset:           asset.USDTMarginedFutures,
		Pair:            currency.NewPair(currency.ENJ, currency.USDT),
		PaymentCurrency: currency.USDT,
		StartDate:       time.Now().Add(-time.Hour * 8008),
		EndDate:         time.Now(),
	})
	assert.ErrorIs(t, err, fundingrate.ErrFundingRateOutsideLimits)

	history, err := e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
		Asset:           asset.USDTMarginedFutures,
		Pair:            currency.NewPair(currency.ENJ, currency.USDT),
		PaymentCurrency: currency.USDT,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, history)
}

func TestGetOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := e.GetOpenInterest(t.Context(), key.PairAsset{
		Base:  currency.NewCode("GOLDFISH").Item,
		Quote: currency.USDT.Item,
		Asset: asset.USDTMarginedFutures,
	})
	assert.ErrorIs(t, err, currency.ErrPairNotFound, "GetOpenInterest should error correctly")

	var resp []futures.OpenInterest
	for _, a := range []asset.Item{asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.DeliveryFutures} {
		p := getPair(t, a)
		resp, err = e.GetOpenInterest(t.Context(), key.PairAsset{
			Base:  p.Base.Item,
			Quote: p.Quote.Item,
			Asset: a,
		})
		assert.NoErrorf(t, err, "GetOpenInterest should not error for %s asset", a)
		assert.Lenf(t, resp, 1, "GetOpenInterest should return 1 item for %s asset", a)
	}

	resp, err = e.GetOpenInterest(t.Context())
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, "GetOpenInterest should return some items")
}

func TestGetClientOrderIDFromText(t *testing.T) {
	t.Parallel()
	assert.Empty(t, getClientOrderIDFromText("api"), "should not return anything")
	assert.Equal(t, "t-123", getClientOrderIDFromText("t-123"), "should return t-123")
}

func TestFormatClientOrderID(t *testing.T) {
	t.Parallel()
	assert.Empty(t, formatClientOrderID(""), "should not return anything")
	assert.Equal(t, "t-123", formatClientOrderID("t-123"), "should return t-123")
	assert.Equal(t, "t-456", formatClientOrderID("456"), "should return t-456")
}

func TestGetSideAndAmountFromSize(t *testing.T) {
	t.Parallel()
	side, amount, remaining := getSideAndAmountFromSize(1, 1)
	assert.Equal(t, order.Long, side, "should be a buy order")
	assert.Equal(t, 1.0, amount, "should be 1.0")
	assert.Equal(t, 1.0, remaining, "should be 1.0")

	side, amount, remaining = getSideAndAmountFromSize(-1, -1)
	assert.Equal(t, order.Short, side, "should be a sell order")
	assert.Equal(t, 1.0, amount, "should be 1.0")
	assert.Equal(t, 1.0, remaining, "should be 1.0")
}

func TestGetFutureOrderSize(t *testing.T) {
	t.Parallel()
	_, err := getFutureOrderSize(&order.Submit{Side: order.CouldNotCloseShort, Amount: 1})
	assert.ErrorIs(t, err, order.ErrSideIsInvalid)

	ret, err := getFutureOrderSize(&order.Submit{Side: order.Buy, Amount: 1})
	require.NoError(t, err)
	assert.Equal(t, 1.0, ret)

	ret, err = getFutureOrderSize(&order.Submit{Side: order.Sell, Amount: 1})
	require.NoError(t, err)
	assert.Equal(t, -1.0, ret)
}

func TestProcessFuturesOrdersPushData(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		incoming string
		status   order.Status
	}{
		{`{"channel":"futures.orders","event":"update","time":1541505434,"time_ms":1541505434123,"result":[{"contract":"BTC_USD","create_time":1628736847,"create_time_ms":1628736847325,"fill_price":40000.4,"finish_as":"","finish_time":1628736848,"finish_time_ms":1628736848321,"iceberg":0,"id":4872460,"is_close":false,"is_liq":false,"is_reduce_only":false,"left":0,"mkfr":-0.00025,"price":40000.4,"refr":0,"refu":0,"size":1,"status":"open","text":"-","tif":"gtc","tkfr":0.0005,"user":"110xxxxx"}]}`, order.Open},
		{`{"channel":"futures.orders","event":"update","time":1541505434,"time_ms":1541505434123,"result":[{"contract":"BTC_USD","create_time":1628736847,"create_time_ms":1628736847325,"fill_price":40000.4,"finish_as":"filled","finish_time":1628736848,"finish_time_ms":1628736848321,"iceberg":0,"id":4872460,"is_close":false,"is_liq":false,"is_reduce_only":false,"left":0,"mkfr":-0.00025,"price":40000.4,"refr":0,"refu":0,"size":1,"status":"finished","text":"-","tif":"gtc","tkfr":0.0005,"user":"110xxxxx"}]}`, order.Filled},
		{`{"channel":"futures.orders","event":"update","time":1541505434,"time_ms":1541505434123,"result":[{"contract":"BTC_USD","create_time":1628736847,"create_time_ms":1628736847325,"fill_price":40000.4,"finish_as":"cancelled","finish_time":1628736848,"finish_time_ms":1628736848321,"iceberg":0,"id":4872460,"is_close":false,"is_liq":false,"is_reduce_only":false,"left":0,"mkfr":-0.00025,"price":40000.4,"refr":0,"refu":0,"size":1,"status":"finished","text":"-","tif":"gtc","tkfr":0.0005,"user":"110xxxxx"}]}`, order.Cancelled},
		{`{"channel":"futures.orders","event":"update","time":1541505434,"time_ms":1541505434123,"result":[{"contract":"BTC_USD","create_time":1628736847,"create_time_ms":1628736847325,"fill_price":40000.4,"finish_as":"liquidated","finish_time":1628736848,"finish_time_ms":1628736848321,"iceberg":0,"id":4872460,"is_close":false,"is_liq":false,"is_reduce_only":false,"left":0,"mkfr":-0.00025,"price":40000.4,"refr":0,"refu":0,"size":1,"status":"finished","text":"-","tif":"gtc","tkfr":0.0005,"user":"110xxxxx"}]}`, order.Liquidated},
		{`{"channel":"futures.orders","event":"update","time":1541505434,"time_ms":1541505434123,"result":[{"contract":"BTC_USD","create_time":1628736847,"create_time_ms":1628736847325,"fill_price":40000.4,"finish_as":"ioc","finish_time":1628736848,"finish_time_ms":1628736848321,"iceberg":0,"id":4872460,"is_close":false,"is_liq":false,"is_reduce_only":false,"left":0,"mkfr":-0.00025,"price":40000.4,"refr":0,"refu":0,"size":1,"status":"finished","text":"-","tif":"gtc","tkfr":0.0005,"user":"110xxxxx"}]}`, order.Cancelled},
		{`{"channel":"futures.orders","event":"update","time":1541505434,"time_ms":1541505434123,"result":[{"contract":"BTC_USD","create_time":1628736847,"create_time_ms":1628736847325,"fill_price":40000.4,"finish_as":"auto_deleveraged","finish_time":1628736848,"finish_time_ms":1628736848321,"iceberg":0,"id":4872460,"is_close":false,"is_liq":false,"is_reduce_only":false,"left":0,"mkfr":-0.00025,"price":40000.4,"refr":0,"refu":0,"size":1,"status":"finished","text":"-","tif":"gtc","tkfr":0.0005,"user":"110xxxxx"}]}`, order.AutoDeleverage},
		{`{"channel":"futures.orders","event":"update","time":1541505434,"time_ms":1541505434123,"result":[{"contract":"BTC_USD","create_time":1628736847,"create_time_ms":1628736847325,"fill_price":40000.4,"finish_as":"reduce_only","finish_time":1628736848,"finish_time_ms":1628736848321,"iceberg":0,"id":4872460,"is_close":false,"is_liq":false,"is_reduce_only":false,"left":0,"mkfr":-0.00025,"price":40000.4,"refr":0,"refu":0,"size":1,"status":"finished","text":"-","tif":"gtc","tkfr":0.0005,"user":"110xxxxx"}]}`, order.Cancelled},
		{`{"channel":"futures.orders","event":"update","time":1541505434,"time_ms":1541505434123,"result":[{"contract":"BTC_USD","create_time":1628736847,"create_time_ms":1628736847325,"fill_price":40000.4,"finish_as":"position_closed","finish_time":1628736848,"finish_time_ms":1628736848321,"iceberg":0,"id":4872460,"is_close":false,"is_liq":false,"is_reduce_only":false,"left":0,"mkfr":-0.00025,"price":40000.4,"refr":0,"refu":0,"size":1,"status":"finished","text":"-","tif":"gtc","tkfr":0.0005,"user":"110xxxxx"}]}`, order.Closed},
		{`{"channel":"futures.orders","event":"update","time":1541505434,"time_ms":1541505434123,"result":[{"contract":"BTC_USD","create_time":1628736847,"create_time_ms":1628736847325,"fill_price":40000.4,"finish_as":"stp","finish_time":1628736848,"finish_time_ms":1628736848321,"iceberg":0,"id":4872460,"is_close":false,"is_liq":false,"is_reduce_only":false,"left":0,"mkfr":-0.00025,"price":40000.4,"refr":0,"refu":0,"size":1,"status":"finished","text":"-","tif":"gtc","tkfr":0.0005,"user":"110xxxxx"}]}`, order.STP},
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			t.Parallel()
			processed, err := e.processFuturesOrdersPushData([]byte(tc.incoming), asset.CoinMarginedFutures)
			require.NoError(t, err)
			require.NotNil(t, processed)
			for i := range processed {
				assert.Equal(t, tc.status.String(), processed[i].Status.String())
			}
		})
	}
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, e)
	for _, a := range e.GetAssetTypes(false) {
		pairs, err := e.CurrencyPairs.GetPairs(a, false)
		require.NoErrorf(t, err, "cannot get pairs for %s", a)
		require.NotEmptyf(t, pairs, "no pairs for %s", a)
		resp, err := e.GetCurrencyTradeURL(t.Context(), a, pairs[0])
		if a == asset.Options {
			require.ErrorIs(t, err, asset.ErrNotSupported)
		} else {
			require.NoError(t, err)
			assert.NotEmpty(t, resp)
		}
	}
}

func TestGetUnifiedAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	// Requires unified account to be enabled for this to function.
	payload, err := e.GetUnifiedAccount(t.Context(), currency.EMPTYCODE, "")
	require.NoError(t, err)
	require.NotEmpty(t, payload)
}

func TestGetMaximumBorrowableAmountUnifiedAccount(t *testing.T) {
	t.Parallel()
	_, err := e.GetMaximumBorrowableAmountUnifiedAccount(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetMaximumBorrowableAmountUnifiedAccount(t.Context(), currency.ETH)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUnifiedAccountMaximumTransferableAmount(t *testing.T) {
	t.Parallel()
	_, err := e.GetUnifiedAccountMaximumTransferableAmount(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUnifiedAccountMaximumTransferableAmount(t.Context(), currency.ETH)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMultipleTransferableAmountForUnifiedAccounts(t *testing.T) {
	t.Parallel()
	_, err := e.GetMultipleTransferableAmountForUnifiedAccounts(t.Context())
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.GetMultipleTransferableAmountForUnifiedAccounts(t.Context(), currency.BTC, currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetMultipleTransferableAmountForUnifiedAccounts(t.Context(), currency.BTC, currency.ETH)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBatchUnifiedAccountMaximumBorrowableAmount(t *testing.T) {
	t.Parallel()
	_, err := e.GetBatchUnifiedAccountMaximumBorrowableAmount(t.Context())
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.GetBatchUnifiedAccountMaximumBorrowableAmount(t.Context(), currency.BTC, currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetBatchUnifiedAccountMaximumBorrowableAmount(t.Context(), currency.BTC, currency.ETH)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestBorrowOrRepay(t *testing.T) {
	t.Parallel()
	_, err := e.BorrowOrRepay(t.Context(), &BorrowOrRepayParams{Amount: 1})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.BorrowOrRepay(t.Context(), &BorrowOrRepayParams{Currency: currency.ETH})
	require.ErrorIs(t, err, errLoanTypeIsRequired)
	_, err = e.BorrowOrRepay(t.Context(), &BorrowOrRepayParams{Currency: currency.ETH, Type: "borrow"})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.BorrowOrRepay(t.Context(), &BorrowOrRepayParams{Amount: 2, Currency: currency.ETH, Type: "borrow"})
	assert.NoError(t, err)
}

func TestGetLoans(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetLoans(t.Context(), currency.ETH, "repay", 0, 1000)
	assert.NoError(t, err)
}

func TestGetLoanRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetLoanRecords(t.Context(), currency.ETH, "repay", 0, 1000)
	assert.NoError(t, err)
}

func TestGetInterestDeductionRecords(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetInterestDeductionRecords(t.Context(), currency.BTC, 0, 10, endTime, startTime, "platform")
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetInterestDeductionRecords(t.Context(), currency.BTC, 0, 10, startTime, endTime, "platform")
	assert.NoError(t, err)
}

func TestGetUserRiskUnitDetails(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetUserRiskUnitDetails(t.Context())
	assert.NoError(t, err)
}

func TestSetUnifiedAccountMode(t *testing.T) {
	t.Parallel()
	err := e.SetUnifiedAccountMode(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	err = e.SetUnifiedAccountMode(t.Context(), &UnifiedAccountMode{Settings: &UnifiedAccountModeSettings{}})
	require.ErrorIs(t, err, errMissingUnifiedAccountMode)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.SetUnifiedAccountMode(t.Context(), &UnifiedAccountMode{Mode: "portfolio"})
	assert.NoError(t, err)
}

func TestGetUnifiedAccountMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUnifiedAccountMode(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUnifiedAccountEstimatedInterestRate(t *testing.T) {
	t.Parallel()
	_, err := e.GetUnifiedAccountEstimatedInterestRate(t.Context(), nil)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.GetUnifiedAccountEstimatedInterestRate(t.Context(), []string{"BTC", "USDT", ""})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUnifiedAccountEstimatedInterestRate(t.Context(), []string{"BTC", "USDT", "ETH"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUnifiedAccountTiered(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUnifiedAccountTiered(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUnifiedAccountTieredLoanMargin(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUnifiedAccountTieredLoanMargin(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSettlementCurrency(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		a   asset.Item
		p   currency.Pair
		exp currency.Code
		err error
	}{
		{asset.Futures, currency.EMPTYPAIR, currency.EMPTYCODE, asset.ErrNotSupported},
		{asset.DeliveryFutures, currency.EMPTYPAIR, currency.USDT, nil},
		{asset.DeliveryFutures, getPair(t, asset.DeliveryFutures), currency.USDT, nil},
		{asset.USDTMarginedFutures, currency.EMPTYPAIR, currency.USDT, nil},
		{asset.USDTMarginedFutures, getPair(t, asset.USDTMarginedFutures), currency.USDT, nil},
		{asset.USDTMarginedFutures, getPair(t, asset.DeliveryFutures), currency.EMPTYCODE, errInvalidSettlementQuote},

		{asset.CoinMarginedFutures, currency.EMPTYPAIR, currency.BTC, nil},
		{asset.CoinMarginedFutures, getPair(t, asset.CoinMarginedFutures), currency.BTC, nil},
		{asset.CoinMarginedFutures, currency.Pair{Base: currency.ETH, Quote: currency.USD}, currency.EMPTYCODE, errInvalidSettlementBase},
	} {
		c, err := getSettlementCurrency(tt.p, tt.a)
		if tt.err == nil {
			require.NoErrorf(t, err, "getSettlementCurrency must not error for %s %s", tt.a, tt.p)
		} else {
			assert.ErrorIsf(t, err, tt.err, "getSettlementCurrency should return correct error for %s %s", tt.a, tt.p)
		}
		assert.Equalf(t, tt.exp, c, "getSettlementCurrency should return correct settlement currency for %s %s", tt.a, tt.p)
	}
}

type FixtureConnection struct{ websocket.Connection }

func (d *FixtureConnection) SendMessageReturnResponse(context.Context, request.EndpointLimit, any, any) ([]byte, error) {
	return []byte(`{"time":1726121320,"time_ms":1726121320745,"id":1,"conn_id":"f903779a148987ca","trace_id":"d8ee37cd14347e4ed298d44e69aedaa7","channel":"spot.tickers","event":"subscribe","payload":["BRETT_USDT"],"result":{"status":"success"},"requestId":"d8ee37cd14347e4ed298d44e69aedaa7"}`), nil
}

func (d *FixtureConnection) GetURL() string { return "wss://test" }

func TestHandleSubscriptions(t *testing.T) {
	t.Parallel()
	subs := subscription.List{{Channel: subscription.OrderbookChannel}}

	err := e.handleSubscription(t.Context(), &FixtureConnection{}, subscribeEvent, subs, func(context.Context, string, subscription.List) ([]*WsInput, error) {
		return []*WsInput{{}}, nil
	})
	require.NoError(t, err)

	err = e.handleSubscription(t.Context(), &FixtureConnection{}, unsubscribeEvent, subs, func(context.Context, string, subscription.List) ([]*WsInput, error) {
		return []*WsInput{{}}, nil
	})
	require.NoError(t, err)
}

func TestParseWSHeader(t *testing.T) {
	in := []string{
		`{"time":1726121320,"time_ms":1726121320745,"id":1,"channel":"spot.tickers","event":"subscribe","result":{"status":"success"},"request_id":"a4"}`,
		`{"time_ms":1726121320746,"id":2,"channel":"spot.tickers","event":"subscribe","result":{"status":"success"},"request_id":"a4"}`,
		`{"time":1726121321,"id":3,"channel":"spot.tickers","event":"subscribe","result":{"status":"success"},"request_id":"a4"}`,
	}
	for _, i := range in {
		h, err := parseWSHeader([]byte(i))
		require.NoError(t, err)
		require.NotEmpty(t, h.ID)
		assert.Equal(t, "a4", h.RequestID)
		assert.Equal(t, "spot.tickers", h.Channel)
		assert.Equal(t, "subscribe", h.Event)
		assert.NotEmpty(t, h.Result)
		switch h.ID {
		case 1:
			assert.Equal(t, int64(1726121320745), h.Time.UnixMilli())
		case 2:
			assert.Equal(t, int64(1726121320746), h.Time.UnixMilli())
		case 3:
			assert.Equal(t, int64(1726121321), h.Time.Unix())
		}
	}
}

func TestDeriveSpotWebsocketOrderResponse(t *testing.T) {
	t.Parallel()
	var resp *WebsocketOrderResponse
	require.NoError(t, json.Unmarshal([]byte(`{"left":"0","update_time":"1735720637","amount":"0.0001","create_time":"1735720637","price":"0","finish_as":"filled","time_in_force":"ioc","currency_pair":"BTC_USDT","type":"market","account":"spot","side":"sell","amend_text":"-","text":"t-1735720637181634009","status":"closed","iceberg":"0","avg_deal_price":"93503.3","filled_total":"9.35033","id":"766075454481","fill_price":"9.35033","update_time_ms":1735720637188,"create_time_ms":1735720637188}`), &resp), "unmarshal must not error")

	got, err := e.deriveSpotWebsocketOrderResponse(resp)
	require.NoError(t, err)
	assert.Equal(t, &order.SubmitResponse{
		Exchange:             e.Name,
		OrderID:              "766075454481",
		AssetType:            asset.Spot,
		Pair:                 currency.NewBTCUSDT().Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
		ClientOrderID:        "t-1735720637181634009",
		Date:                 time.UnixMilli(1735720637188),
		LastUpdated:          time.UnixMilli(1735720637188),
		Amount:               0.0001,
		AverageExecutedPrice: 93503.3,
		Type:                 order.Market,
		Side:                 order.Sell,
		Status:               order.Filled,
		TimeInForce:          order.ImmediateOrCancel,
		Cost:                 0.0001,
		Purchased:            9.35033,
	}, got)
}

func TestDeriveSpotWebsocketOrderResponses(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		orders   [][]byte
		error    error
		expected []*order.SubmitResponse
	}{
		{
			name:   "no response",
			orders: [][]byte{},
			error:  common.ErrNoResponse,
		},
		{
			name: "assortment of spot orders",
			orders: [][]byte{
				[]byte(`{"left":"0","update_time":"1735720637","amount":"0.0001","create_time":"1735720637","price":"0","finish_as":"filled","time_in_force":"ioc","currency_pair":"BTC_USDT","type":"market","account":"spot","side":"sell","amend_text":"-","text":"t-1735720637181634009","status":"closed","iceberg":"0","avg_deal_price":"93503.3","filled_total":"9.35033","id":"766075454481","fill_price":"9.35033","update_time_ms":1735720637188,"create_time_ms":1735720637188}`),
				[]byte(`{"left":"0.000008","update_time":"1735720637","amount":"9.99152","create_time":"1735720637","price":"0","finish_as":"filled","time_in_force":"ioc","currency_pair":"HNS_USDT","type":"market","account":"spot","side":"buy","amend_text":"-","text":"t-1735720637126962151","status":"closed","iceberg":"0","avg_deal_price":"0.01224","filled_total":"9.991512","id":"766075454188","fill_price":"9.991512","update_time_ms":1735720637142,"create_time_ms":1735720637142}`),
				[]byte(`{"left":"0","update_time":"1735778597","amount":"200","create_time":"1735778597","price":"0.03673","finish_as":"filled","time_in_force":"fok","currency_pair":"REX_USDT","type":"limit","account":"spot","side":"buy","amend_text":"-","text":"t-1364","status":"closed","iceberg":"0","avg_deal_price":"0.03673","filled_total":"7.346","id":"766488882062","fill_price":"7.346","update_time_ms":1735778597363,"create_time_ms":1735778597363}`),
				[]byte(`{"left":"0.0003","update_time":"1735780321","amount":"0.0003","create_time":"1735780321","price":"20000","finish_as":"open","time_in_force":"poc","currency_pair":"BTC_USDT","type":"limit","account":"spot","side":"buy","amend_text":"-","text":"t-1735780321603944400","status":"open","iceberg":"0","filled_total":"0","id":"766504537761","fill_price":"0","update_time_ms":1735780321729,"create_time_ms":1735780321729}`),
				[]byte(`{"left":"1","update_time":"1735784755","amount":"1","create_time":"1735784755","price":"100","finish_as":"open","time_in_force":"gtc","currency_pair":"GT_USDT","type":"limit","account":"spot","side":"sell","amend_text":"-","text":"t-1735784754905434100","status":"open","iceberg":"0","filled_total":"0","id":"766536556747","fill_price":"0","update_time_ms":1735784755068,"create_time_ms":1735784755068}`),
			},
			expected: []*order.SubmitResponse{
				{
					Exchange:             e.Name,
					OrderID:              "766075454481",
					AssetType:            asset.Spot,
					Pair:                 currency.NewBTCUSDT().Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					ClientOrderID:        "t-1735720637181634009",
					Date:                 time.UnixMilli(1735720637188),
					LastUpdated:          time.UnixMilli(1735720637188),
					Amount:               0.0001,
					AverageExecutedPrice: 93503.3,
					Type:                 order.Market,
					Side:                 order.Sell,
					Status:               order.Filled,
					TimeInForce:          order.ImmediateOrCancel,
					Cost:                 0.0001,
					Purchased:            9.35033,
				},
				{
					Exchange:             e.Name,
					OrderID:              "766075454188",
					AssetType:            asset.Spot,
					Pair:                 currency.NewPair(currency.HNS, currency.USDT).Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					ClientOrderID:        "t-1735720637126962151",
					Date:                 time.UnixMilli(1735720637142),
					LastUpdated:          time.UnixMilli(1735720637142),
					RemainingAmount:      0.000008,
					Amount:               9.99152,
					AverageExecutedPrice: 0.01224,
					Type:                 order.Market,
					Side:                 order.Buy,
					Status:               order.Filled,
					TimeInForce:          order.ImmediateOrCancel,
					Cost:                 9.991512,
					Purchased:            816.3,
				},
				{
					Exchange:             e.Name,
					OrderID:              "766488882062",
					AssetType:            asset.Spot,
					Pair:                 currency.NewPair(currency.NewCode("REX"), currency.USDT).Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					ClientOrderID:        "t-1364",
					Date:                 time.UnixMilli(1735778597363),
					LastUpdated:          time.UnixMilli(1735778597363),
					Amount:               200,
					Price:                0.03673,
					AverageExecutedPrice: 0.03673,
					Type:                 order.Limit,
					Side:                 order.Buy,
					Status:               order.Filled,
					TimeInForce:          order.FillOrKill,
					Cost:                 7.346,
					Purchased:            200,
				},
				{
					Exchange:        e.Name,
					OrderID:         "766504537761",
					AssetType:       asset.Spot,
					Pair:            currency.NewBTCUSDT().Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					ClientOrderID:   "t-1735780321603944400",
					Date:            time.UnixMilli(1735780321729),
					LastUpdated:     time.UnixMilli(1735780321729),
					RemainingAmount: 0.0003,
					Amount:          0.0003,
					Price:           20000,
					Type:            order.Limit,
					Side:            order.Buy,
					Status:          order.Open,
					TimeInForce:     order.PostOnly,
				},
				{
					Exchange:        e.Name,
					OrderID:         "766536556747",
					AssetType:       asset.Spot,
					Pair:            currency.NewPair(currency.NewCode("GT"), currency.USDT).Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					ClientOrderID:   "t-1735784754905434100",
					Date:            time.UnixMilli(1735784755068),
					LastUpdated:     time.UnixMilli(1735784755068),
					RemainingAmount: 1,
					Amount:          1,
					Price:           100,
					Type:            order.Limit,
					Side:            order.Sell,
					Status:          order.Open,
					TimeInForce:     order.GoodTillCancel,
				},
			},
		},
		{
			name: "batch of spot orders with error at end",
			// This is specifically testing the return responses of WebsocketSpotSubmitOrders
			// AverageDealPrice is not returned when using this endpoint so purchased and cost fields cannot be set.
			orders: [][]byte{
				[]byte(`{"account":"spot","status":"closed","side":"buy","amount":"9.98","id":"775453816782","create_time":"1736980695","update_time":"1736980695","text":"t-740","left":"0.047239","currency_pair":"ETH_USDT","type":"market","finish_as":"filled","price":"0","time_in_force":"fok","iceberg":"0","filled_total":"9.932761","fill_price":"9.932761","create_time_ms":1736980695949,"update_time_ms":1736980695949,"succeeded":true}`),
				[]byte(`{"account":"spot","status":"closed","side":"buy","amount":"0.00289718","id":"775453816824","create_time":"1736980695","update_time":"1736980695","text":"t-741","left":"0.00000000962","currency_pair":"LIKE_ETH","type":"market","finish_as":"filled","price":"0","time_in_force":"fok","iceberg":"0","filled_total":"0.00289717038","fill_price":"0.00289717038","create_time_ms":1736980695956,"update_time_ms":1736980695956,"succeeded":true}`),
				[]byte(`{"text":"t-742","label":"BALANCE_NOT_ENOUGH","message":"Not enough balance"}`),
			},
			expected: []*order.SubmitResponse{
				{
					Exchange:        e.Name,
					OrderID:         "775453816782",
					AssetType:       asset.Spot,
					Pair:            currency.NewPair(currency.ETH, currency.USDT).Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					ClientOrderID:   "t-740",
					Date:            time.UnixMilli(1736980695949),
					LastUpdated:     time.UnixMilli(1736980695949),
					Amount:          9.98,
					RemainingAmount: 0.047239,
					Type:            order.Market,
					Side:            order.Buy,
					Status:          order.Filled,
					TimeInForce:     order.FillOrKill,
				},
				{
					Exchange:        e.Name,
					OrderID:         "775453816824",
					AssetType:       asset.Spot,
					Pair:            currency.NewPair(currency.LIKE, currency.ETH).Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					ClientOrderID:   "t-741",
					Date:            time.UnixMilli(1736980695956),
					LastUpdated:     time.UnixMilli(1736980695956),
					RemainingAmount: 0.00000000962,
					Amount:          0.00289718,
					Type:            order.Market,
					Side:            order.Buy,
					Status:          order.Filled,
					TimeInForce:     order.FillOrKill,
				},
				{
					Exchange:        e.Name,
					ClientOrderID:   "t-742",
					SubmissionError: order.ErrUnableToPlaceOrder,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			orders := bytes.Join(tc.orders, []byte(","))
			orders = append([]byte("["), append(orders, []byte("]")...)...)

			var resp []*WebsocketOrderResponse
			require.NoError(t, json.Unmarshal(orders, &resp), "unmarshal must not error")

			got, err := e.deriveSpotWebsocketOrderResponses(resp)
			require.ErrorIs(t, err, tc.error)

			require.Len(t, got, len(tc.expected))
			for i := range got {
				if tc.expected[i].SubmissionError != nil {
					assert.ErrorIs(t, got[i].SubmissionError, tc.expected[i].SubmissionError)
					assert.Equal(t, tc.expected[i].Exchange, got[i].Exchange)
					assert.Equal(t, tc.expected[i].ClientOrderID, got[i].ClientOrderID)
					continue
				}
				assert.Equal(t, tc.expected[i], got[i])
			}
		})
	}
}

func TestDeriveFuturesWebsocketOrderResponse(t *testing.T) {
	t.Parallel()

	var resp *WebsocketFuturesOrderResponse
	require.NoError(t, json.Unmarshal([]byte(`{"text":"t-1337","price":"0","biz_info":"-","tif":"ioc","amend_text":"-","status":"finished","contract":"CWIF_USDT","stp_act":"-","finish_as":"filled","fill_price":"0.0000002625","id":596729318437,"create_time":1735787107.449,"size":2,"finish_time":1735787107.45,"update_time":1735787107.45,"left":0,"user":12870774,"is_reduce_only":true}`), &resp), "unmarshal must not error")

	got, err := e.deriveFuturesWebsocketOrderResponse(resp)
	require.NoError(t, err)
	assert.Equal(t, &order.SubmitResponse{
		Exchange:             e.Name,
		OrderID:              "596729318437",
		AssetType:            asset.Futures,
		Pair:                 currency.NewPair(currency.NewCode("CWIF"), currency.USDT).Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
		ClientOrderID:        "t-1337",
		Date:                 time.UnixMilli(1735787107449),
		LastUpdated:          time.UnixMilli(1735787107450),
		Amount:               2,
		AverageExecutedPrice: 0.0000002625,
		Type:                 order.Market,
		Side:                 order.Long,
		Status:               order.Filled,
		TimeInForce:          order.ImmediateOrCancel,
		ReduceOnly:           true,
	}, got)
}

func TestDeriveFuturesWebsocketOrderResponses(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		orders   [][]byte
		error    error
		expected []*order.SubmitResponse
	}{
		{
			name:   "no response",
			orders: [][]byte{},
			error:  common.ErrNoResponse,
		},
		{
			name: "assortment of futures orders",
			orders: [][]byte{
				[]byte(`{"text":"t-1337","price":"0","biz_info":"-","tif":"ioc","amend_text":"-","status":"finished","contract":"CWIF_USDT","stp_act":"-","finish_as":"filled","fill_price":"0.0000002625","id":596729318437,"create_time":1735787107.449,"size":2,"finish_time":1735787107.45,"update_time":1735787107.45,"left":0,"user":12870774,"is_reduce_only":true}`),
				[]byte(`{"text":"t-1336","price":"0","biz_info":"-","tif":"ioc","amend_text":"-","status":"finished","contract":"REX_USDT","stp_act":"-","finish_as":"filled","fill_price":"0.03654","id":596662040388,"create_time":1735778597.374,"size":-2,"finish_time":1735778597.374,"update_time":1735778597.374,"left":0,"user":12870774}`),
				[]byte(`{"text":"apiv4-ws","price":"40000","biz_info":"-","tif":"gtc","amend_text":"-","status":"open","contract":"BTC_USDT","stp_act":"-","fill_price":"0","id":596746193678,"create_time":1735789790.476,"size":1,"update_time":1735789790.476,"left":1,"user":2365748}`),
				[]byte(`{"text":"apiv4-ws","price":"200000","biz_info":"-","tif":"gtc","amend_text":"-","status":"open","contract":"BTC_USDT","stp_act":"-","fill_price":"0","id":596748780649,"create_time":1735790222.185,"size":-1,"update_time":1735790222.185,"left":-1,"user":2365748}`),
				[]byte(`{"text":"apiv4-ws","price":"0","biz_info":"-","tif":"ioc","amend_text":"-","status":"finished","contract":"BTC_USDT","stp_act":"-","finish_as":"filled","fill_price":"98172.9","id":36028797827161124,"create_time":1740108860.761,"size":1,"finish_time":1740108860.761,"update_time":1740108860.761,"left":0,"user":2365748}`),
				[]byte(`{"text":"apiv4-ws","price":"0","biz_info":"-","tif":"ioc","amend_text":"-","status":"finished","contract":"BTC_USDT","stp_act":"-","finish_as":"filled","fill_price":"98113.1","id":36028797827225781,"create_time":1740109172.06,"size":-1,"finish_time":1740109172.06,"update_time":1740109172.06,"left":0,"user":2365748,"is_reduce_only":true}`),
			},
			expected: []*order.SubmitResponse{
				{
					Exchange:             e.Name,
					OrderID:              "596729318437",
					AssetType:            asset.Futures,
					Pair:                 currency.NewPair(currency.NewCode("CWIF"), currency.USDT).Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					ClientOrderID:        "t-1337",
					Date:                 time.UnixMilli(1735787107449),
					LastUpdated:          time.UnixMilli(1735787107450),
					Amount:               2,
					AverageExecutedPrice: 0.0000002625,
					Type:                 order.Market,
					Side:                 order.Long,
					Status:               order.Filled,
					TimeInForce:          order.ImmediateOrCancel,
					ReduceOnly:           true,
				},
				{
					Exchange:             e.Name,
					OrderID:              "596662040388",
					AssetType:            asset.Futures,
					Pair:                 currency.NewPair(currency.NewCode("REX"), currency.USDT).Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					ClientOrderID:        "t-1336",
					Date:                 time.UnixMilli(1735778597374),
					LastUpdated:          time.UnixMilli(1735778597374),
					Amount:               2,
					AverageExecutedPrice: 0.03654,
					Type:                 order.Market,
					Side:                 order.Short,
					Status:               order.Filled,
					TimeInForce:          order.ImmediateOrCancel,
				},
				{
					Exchange:        e.Name,
					OrderID:         "596746193678",
					AssetType:       asset.Futures,
					Pair:            currency.NewBTCUSDT().Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					Date:            time.UnixMilli(1735789790476),
					LastUpdated:     time.UnixMilli(1735789790476),
					RemainingAmount: 1,
					Amount:          1,
					Price:           40000,
					Type:            order.Limit,
					Side:            order.Long,
					Status:          order.Open,
					TimeInForce:     order.GoodTillCancel,
				},
				{
					Exchange:        e.Name,
					OrderID:         "596748780649",
					AssetType:       asset.Futures,
					Pair:            currency.NewBTCUSDT().Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					Date:            time.UnixMilli(1735790222185),
					LastUpdated:     time.UnixMilli(1735790222185),
					RemainingAmount: 1,
					Amount:          1,
					Price:           200000,
					Type:            order.Limit,
					Side:            order.Short,
					Status:          order.Open,
					TimeInForce:     order.GoodTillCancel,
				},
				{
					Exchange:             e.Name,
					OrderID:              "36028797827161124",
					AssetType:            asset.Futures,
					Pair:                 currency.NewBTCUSDT().Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					Date:                 time.UnixMilli(1740108860761),
					LastUpdated:          time.UnixMilli(1740108860761),
					Amount:               1,
					AverageExecutedPrice: 98172.9,
					Type:                 order.Market,
					Side:                 order.Long,
					Status:               order.Filled,
					TimeInForce:          order.ImmediateOrCancel,
				},
				{
					Exchange:             e.Name,
					OrderID:              "36028797827225781",
					AssetType:            asset.Futures,
					Pair:                 currency.NewBTCUSDT().Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					Date:                 time.UnixMilli(1740109172060),
					LastUpdated:          time.UnixMilli(1740109172060),
					Amount:               1,
					AverageExecutedPrice: 98113.1,
					Type:                 order.Market,
					Side:                 order.Short,
					Status:               order.Filled,
					TimeInForce:          order.ImmediateOrCancel,
					ReduceOnly:           true,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			orders := bytes.Join(tc.orders, []byte(","))
			orders = append([]byte("["), append(orders, []byte("]")...)...)

			var resp []*WebsocketFuturesOrderResponse
			require.NoError(t, json.Unmarshal(orders, &resp), "unmarshal must not error")

			got, err := e.deriveFuturesWebsocketOrderResponses(resp)
			require.ErrorIs(t, err, tc.error)

			require.Len(t, got, len(tc.expected))
			for i := range got {
				assert.Equal(t, tc.expected[i], got[i])
			}
		})
	}
}

func TestConvertSmallBalances(t *testing.T) {
	t.Parallel()
	err := e.ConvertSmallBalances(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.ConvertSmallBalances(t.Context(), currency.F16)
	require.NoError(t, err)
}

func TestGetConvertibleSmallBalanceCurrencyHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetConvertibleSmallBalanceCurrencyHistory(t.Context(), currency.ETH, 1, 10)
	require.NoError(t, err)
}

func TestGetAccountDetails(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	got, err := e.GetAccountDetails(t.Context())
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestGetUserTransactionRateLimitInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	got, err := e.GetUserTransactionRateLimitInfo(t.Context())
	require.NoError(t, err)
	assert.NotEmpty(t, got)
}

func TestCreateSelfTradePreventionUserGroup(t *testing.T) {
	t.Parallel()
	_, err := e.CreateSelfTradePreventionUserGroup(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = e.CreateSelfTradePreventionUserGroup(t.Context(), &STPUserGroup{Name: "", ID: 234235664353})
	require.ErrorIs(t, err, errSTPGroupNameRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CreateSelfTradePreventionUserGroup(t.Context(), &STPUserGroup{
		Name:      "stp_groups",
		CreatorID: 879868760123,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestGetUserSelfTradePreventionGroups(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUserSelfTradePreventionGroups(t.Context(), "thrasher")
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestGetUsersInSTPUserGroup(t *testing.T) {
	t.Parallel()
	_, err := e.GetUsersInSTPUserGroup(t.Context(), "")
	require.ErrorIs(t, err, errSTPGroupIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUsersInSTPUserGroup(t.Context(), "1")
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestAddUsersToSTPUserGroup(t *testing.T) {
	t.Parallel()
	_, err := e.AddUsersToSTPUserGroup(t.Context(), "", nil)
	require.ErrorIs(t, err, errSTPGroupIDRequired)

	_, err = e.AddUsersToSTPUserGroup(t.Context(), "2123", nil)
	require.ErrorIs(t, err, errUserIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.AddUsersToSTPUserGroup(t.Context(), "1", []uint64{12312312, 2132234232})
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestDeleteUserFromSTPUserGroup(t *testing.T) {
	t.Parallel()
	_, err := e.DeleteUserFromSTPUserGroup(t.Context(), "", 0)
	require.ErrorIs(t, err, errSTPGroupIDRequired)

	_, err = e.DeleteUserFromSTPUserGroup(t.Context(), "2123", 0)
	require.ErrorIs(t, err, errUserIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.DeleteUserFromSTPUserGroup(t.Context(), "1", 12312312)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestConfigureGTFeeDeduction(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	err := e.ConfigureGTFeeDeduction(t.Context(), false)
	assert.NoError(t, err)
}

func TestGetGTFeeDeductionConfiguration(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetGTFeeDeductionConfiguration(t.Context())
	assert.NoError(t, err)
}

func TestPlaceMultiCollateralLoanOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceMultiCollateralLoanOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = e.PlaceMultiCollateralLoanOrder(t.Context(), &MultiCollateralLoanOrderParam{BorrowAmount: 123})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.PlaceMultiCollateralLoanOrder(t.Context(), &MultiCollateralLoanOrderParam{BorrowCurrency: currency.ETH})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.PlaceMultiCollateralLoanOrder(t.Context(), &MultiCollateralLoanOrderParam{BorrowCurrency: currency.ETH, BorrowAmount: 123})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

var (
	pairMap    = map[asset.Item]currency.Pairs{}
	pairsGuard sync.RWMutex
)

func getPair(tb testing.TB, a asset.Item) currency.Pair {
	tb.Helper()
	return enabledAssetPair[a]
}

func getPairs(tb testing.TB, a asset.Item) currency.Pairs {
	tb.Helper()
	pairsGuard.RLock()
	p, ok := pairMap[a]
	pairsGuard.RUnlock()
	if ok {
		return p
	}
	pairsGuard.Lock()
	defer pairsGuard.Unlock()
	p, ok = pairMap[a] // Protect Race if we blocked on Lock and another RW populated
	if ok {
		return p
	}

	testexch.UpdatePairsOnce(tb, e)
	enabledPairs, err := e.GetEnabledPairs(a)
	assert.NoErrorf(tb, err, "%s GetEnabledPairs should not error", a)
	if !assert.NotEmptyf(tb, enabledPairs, "%s GetEnabledPairs should not be empty", a) {
		tb.Fatalf("No pair available for asset %s", a)
		return nil
	}
	pairMap[a] = enabledPairs

	return enabledPairs
}

// BenchmarkTimeInForceFromString-8         1528101               767.3 ns/op             0 B/op          0 allocs/op
func BenchmarkTimeInForceFromString(b *testing.B) {
	for b.Loop() {
		for _, tifString := range []string{gtcTIF, iocTIF, pocTIF, fokTIF} {
			_, err := timeInForceFromString(tifString)
			require.NoError(b, err)
		}
	}
}

func TestTimeInForceFromString(t *testing.T) {
	t.Parallel()
	_, err := timeInForceFromString("abcdef")
	assert.ErrorIs(t, err, order.ErrUnsupportedTimeInForce)

	for k, v := range map[string]order.TimeInForce{gtcTIF: order.GoodTillCancel, iocTIF: order.ImmediateOrCancel, pocTIF: order.PostOnly, fokTIF: order.FillOrKill} {
		t.Run(k, func(t *testing.T) {
			t.Parallel()
			tif, err := timeInForceFromString(k)
			require.NoError(t, err)
			assert.Equal(t, v, tif)
		})
	}
}

func TestGetTypeFromTimeInForce(t *testing.T) {
	t.Parallel()
	typeResp := getTypeFromTimeInForce("gtc", 0)
	assert.Equal(t, order.Limit, typeResp)

	typeResp = getTypeFromTimeInForce("ioc", 0)
	assert.Equal(t, order.Market, typeResp, "should be market order")

	typeResp = getTypeFromTimeInForce("poc", 123)
	assert.Equal(t, order.Limit, typeResp, "should be limit order")

	typeResp = getTypeFromTimeInForce("fok", 0)
	assert.Equal(t, order.Market, typeResp, "should be market order")
}

func TestToExchangeTIF(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		tif      order.TimeInForce
		price    float64
		expected string
		err      error
	}{
		{price: 0, expected: iocTIF}, // market orders default to IOC
		{price: 0, tif: order.FillOrKill, expected: fokTIF},
		{price: 420, expected: gtcTIF}, // limit orders default to GTC
		{price: 420, tif: order.GoodTillCancel, expected: gtcTIF},
		{price: 420, tif: order.ImmediateOrCancel, expected: iocTIF},
		{price: 420, tif: order.PostOnly, expected: pocTIF},
		{price: 420, tif: order.FillOrKill, expected: fokTIF},
		{tif: order.GoodTillTime, err: order.ErrUnsupportedTimeInForce},
	} {
		t.Run(fmt.Sprintf("TIF:%q Price:'%v'", tc.tif, tc.price), func(t *testing.T) {
			t.Parallel()
			got, err := toExchangeTIF(tc.tif, tc.price)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tc.expected, got)
		})
	}
}

func TestIsSingleOrderbookChannel(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		channel  string
		expected bool
	}{
		{channel: spotOrderbookUpdateChannel, expected: true},
		{channel: spotOrderbookChannel, expected: true},
		{channel: spotOrderbookTickerChannel, expected: true},
		{channel: futuresOrderbookChannel, expected: true},
		{channel: futuresOrderbookTickerChannel, expected: true},
		{channel: futuresOrderbookUpdateChannel, expected: true},
		{channel: optionsOrderbookChannel, expected: true},
		{channel: optionsOrderbookTickerChannel, expected: true},
		{channel: optionsOrderbookUpdateChannel, expected: true},
		{channel: spotTickerChannel, expected: false},
		{channel: "sad", expected: false},
	} {
		assert.Equal(t, tc.expected, isSingleOrderbookChannel(tc.channel))
	}
}

func TestValidateSubscriptions(t *testing.T) {
	t.Parallel()
	require.NoError(t, e.ValidateSubscriptions(nil))
	require.NoError(t, e.ValidateSubscriptions([]*subscription.Subscription{{Channel: spotTickerChannel, Pairs: []currency.Pair{currency.NewBTCUSDT()}}}))
	require.NoError(t, e.ValidateSubscriptions([]*subscription.Subscription{
		{Channel: spotTickerChannel, Pairs: []currency.Pair{currency.NewBTCUSD()}},
		{Channel: spotOrderbookUpdateChannel, Pairs: []currency.Pair{currency.NewBTCUSD()}},
	}))
	require.NoError(t, e.ValidateSubscriptions([]*subscription.Subscription{
		{Channel: spotTickerChannel, Pairs: []currency.Pair{currency.NewBTCUSD()}},
		{Channel: spotOrderbookUpdateChannel, Pairs: []currency.Pair{currency.NewBTCUSD(), currency.NewBTCUSDT()}},
	}))
	require.NoError(t, e.ValidateSubscriptions([]*subscription.Subscription{
		{Channel: spotTickerChannel, Pairs: []currency.Pair{currency.NewBTCUSD()}},
		{Channel: spotOrderbookUpdateChannel, Pairs: []currency.Pair{currency.NewBTCUSD()}},
		{Channel: spotOrderbookUpdateChannel, Pairs: []currency.Pair{currency.NewBTCUSDT()}},
	}))
	require.ErrorIs(t, e.ValidateSubscriptions([]*subscription.Subscription{
		{Channel: spotTickerChannel, Pairs: []currency.Pair{currency.NewBTCUSD()}},
		{Channel: spotOrderbookUpdateChannel, Pairs: []currency.Pair{currency.NewBTCUSD()}},
		{Channel: spotOrderbookChannel, Pairs: []currency.Pair{currency.NewBTCUSD()}},
	}), subscription.ErrExclusiveSubscription)
}

func TestCandlesChannelIntervals(t *testing.T) {
	t.Parallel()
	s := &subscription.Subscription{Channel: subscription.CandlesChannel, Asset: asset.Spot, Interval: 0}
	_, err := candlesChannelInterval(s)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval, "candlestickChannelInterval must error correctly with a 0 interval")
	s.Interval = kline.ThousandMilliseconds
	i, err := candlesChannelInterval(s)
	require.NoError(t, err)
	assert.Equal(t, "1000ms", i)
}

func TestOrderbookChannelIntervals(t *testing.T) {
	t.Parallel()

	s := &subscription.Subscription{Channel: futuresOrderbookUpdateChannel, Interval: kline.TwentyMilliseconds, Levels: 100}
	_, err := orderbookChannelInterval(s, asset.Futures)
	require.ErrorIs(t, err, subscription.ErrInvalidInterval)
	require.ErrorContains(t, err, "20ms only valid with Levels 20")
	s.Levels = 20
	i, err := orderbookChannelInterval(s, asset.Futures)
	require.NoError(t, err)
	assert.Equal(t, "20ms", i)

	for s, exp := range map[*subscription.Subscription]error{
		{Asset: asset.Binary, Channel: "unknown_channel", Interval: kline.OneYear}:                    nil,
		{Asset: asset.Spot, Channel: spotOrderbookTickerChannel, Interval: kline.OneDay}:              subscription.ErrInvalidInterval,
		{Asset: asset.Spot, Channel: spotOrderbookTickerChannel, Interval: 0}:                         nil,
		{Asset: asset.Spot, Channel: spotOrderbookChannel, Interval: kline.OneDay}:                    subscription.ErrInvalidInterval,
		{Asset: asset.Spot, Channel: spotOrderbookChannel, Interval: kline.HundredMilliseconds}:       nil,
		{Asset: asset.Spot, Channel: spotOrderbookChannel, Interval: kline.ThousandMilliseconds}:      nil,
		{Asset: asset.Spot, Channel: spotOrderbookUpdateChannel, Interval: kline.OneDay}:              subscription.ErrInvalidInterval,
		{Asset: asset.Spot, Channel: spotOrderbookUpdateChannel, Interval: kline.HundredMilliseconds}: nil,

		{Asset: asset.Futures, Channel: futuresOrderbookTickerChannel, Interval: kline.TenMilliseconds}:     subscription.ErrInvalidInterval,
		{Asset: asset.Futures, Channel: futuresOrderbookTickerChannel, Interval: 0}:                         nil,
		{Asset: asset.Futures, Channel: futuresOrderbookChannel, Interval: kline.TenMilliseconds}:           subscription.ErrInvalidInterval,
		{Asset: asset.Futures, Channel: futuresOrderbookChannel, Interval: 0}:                               nil,
		{Asset: asset.Futures, Channel: futuresOrderbookUpdateChannel, Interval: kline.OneDay}:              subscription.ErrInvalidInterval,
		{Asset: asset.Futures, Channel: futuresOrderbookUpdateChannel, Interval: kline.HundredMilliseconds}: nil,

		{Asset: asset.DeliveryFutures, Channel: futuresOrderbookTickerChannel, Interval: kline.TenMilliseconds}:      subscription.ErrInvalidInterval,
		{Asset: asset.DeliveryFutures, Channel: futuresOrderbookTickerChannel, Interval: 0}:                          nil,
		{Asset: asset.DeliveryFutures, Channel: futuresOrderbookChannel, Interval: kline.TenMilliseconds}:            subscription.ErrInvalidInterval,
		{Asset: asset.DeliveryFutures, Channel: futuresOrderbookChannel, Interval: 0}:                                nil,
		{Asset: asset.DeliveryFutures, Channel: futuresOrderbookUpdateChannel, Interval: kline.OneDay}:               subscription.ErrInvalidInterval,
		{Asset: asset.DeliveryFutures, Channel: futuresOrderbookUpdateChannel, Interval: kline.HundredMilliseconds}:  nil,
		{Asset: asset.DeliveryFutures, Channel: futuresOrderbookUpdateChannel, Interval: kline.ThousandMilliseconds}: nil,

		{Asset: asset.Options, Channel: optionsOrderbookTickerChannel, Interval: kline.TenMilliseconds}:          subscription.ErrInvalidInterval,
		{Asset: asset.Options, Channel: optionsOrderbookTickerChannel, Interval: 0}:                              nil,
		{Asset: asset.Options, Channel: optionsOrderbookChannel, Interval: kline.TwoHundredAndFiftyMilliseconds}: subscription.ErrInvalidInterval,
		{Asset: asset.Options, Channel: optionsOrderbookChannel, Interval: 0}:                                    nil,
		{Asset: asset.Options, Channel: optionsOrderbookUpdateChannel, Interval: kline.OneDay}:                   subscription.ErrInvalidInterval,
		{Asset: asset.Options, Channel: optionsOrderbookUpdateChannel, Interval: kline.HundredMilliseconds}:      nil,
		{Asset: asset.Options, Channel: optionsOrderbookUpdateChannel, Interval: kline.ThousandMilliseconds}:     nil,
	} {
		t.Run(s.Asset.String()+"/"+s.Channel+"/"+s.Interval.Short(), func(t *testing.T) {
			t.Parallel()
			i, err := orderbookChannelInterval(s, s.Asset)
			if exp != nil {
				require.ErrorIs(t, err, exp)
			} else {
				switch {
				case s.Channel == "unknown_channel":
					assert.Empty(t, i, "orderbookChannelInterval should return empty for unknown channels")
				case strings.HasSuffix(s.Channel, "_ticker"):
					assert.Empty(t, i)
				case s.Interval == 0:
					assert.Equal(t, "0", i)
				default:
					exp, err2 := getIntervalString(s.Interval)
					require.NoError(t, err2, "getIntervalString must not error for validating expected value")
					require.Equal(t, exp, i)
				}
			}
		})
	}
}

func TestChannelLevels(t *testing.T) {
	t.Parallel()

	for s, exp := range map[*subscription.Subscription]error{
		{Channel: "unknown_channel", Asset: asset.Binary}:                                   nil,
		{Channel: spotOrderbookTickerChannel, Asset: asset.Spot}:                            nil,
		{Channel: spotOrderbookTickerChannel, Asset: asset.Spot, Levels: 1}:                 subscription.ErrInvalidLevel,
		{Channel: spotOrderbookUpdateChannel, Asset: asset.Spot}:                            nil,
		{Channel: spotOrderbookUpdateChannel, Asset: asset.Spot, Levels: 100}:               subscription.ErrInvalidLevel,
		{Channel: spotOrderbookChannel, Asset: asset.Spot}:                                  subscription.ErrInvalidLevel,
		{Channel: spotOrderbookChannel, Asset: asset.Spot, Levels: 5}:                       nil,
		{Channel: spotOrderbookChannel, Asset: asset.Spot, Levels: 10}:                      nil,
		{Channel: spotOrderbookChannel, Asset: asset.Spot, Levels: 20}:                      nil,
		{Channel: spotOrderbookChannel, Asset: asset.Spot, Levels: 50}:                      nil,
		{Channel: spotOrderbookChannel, Asset: asset.Spot, Levels: 100}:                     nil,
		{Channel: futuresOrderbookChannel, Asset: asset.Futures}:                            subscription.ErrInvalidLevel,
		{Channel: futuresOrderbookChannel, Asset: asset.Futures, Levels: 1}:                 nil,
		{Channel: futuresOrderbookChannel, Asset: asset.Futures, Levels: 5}:                 nil,
		{Channel: futuresOrderbookChannel, Asset: asset.Futures, Levels: 10}:                nil,
		{Channel: futuresOrderbookChannel, Asset: asset.Futures, Levels: 20}:                nil,
		{Channel: futuresOrderbookChannel, Asset: asset.Futures, Levels: 50}:                nil,
		{Channel: futuresOrderbookChannel, Asset: asset.Futures, Levels: 100}:               nil,
		{Channel: futuresOrderbookTickerChannel, Asset: asset.Futures}:                      nil,
		{Channel: futuresOrderbookTickerChannel, Asset: asset.Futures, Levels: 1}:           subscription.ErrInvalidLevel,
		{Channel: futuresOrderbookUpdateChannel, Asset: asset.Futures}:                      subscription.ErrInvalidLevel,
		{Channel: futuresOrderbookUpdateChannel, Asset: asset.Futures, Levels: 20}:          nil,
		{Channel: futuresOrderbookUpdateChannel, Asset: asset.Futures, Levels: 50}:          nil,
		{Channel: futuresOrderbookUpdateChannel, Asset: asset.DeliveryFutures}:              subscription.ErrInvalidLevel,
		{Channel: futuresOrderbookUpdateChannel, Asset: asset.DeliveryFutures, Levels: 5}:   nil,
		{Channel: futuresOrderbookUpdateChannel, Asset: asset.DeliveryFutures, Levels: 10}:  nil,
		{Channel: futuresOrderbookUpdateChannel, Asset: asset.DeliveryFutures, Levels: 20}:  nil,
		{Channel: futuresOrderbookUpdateChannel, Asset: asset.DeliveryFutures, Levels: 50}:  nil,
		{Channel: futuresOrderbookUpdateChannel, Asset: asset.DeliveryFutures, Levels: 100}: nil,
		{Channel: optionsOrderbookTickerChannel, Asset: asset.Options}:                      nil,
		{Channel: optionsOrderbookTickerChannel, Asset: asset.Options, Levels: 1}:           subscription.ErrInvalidLevel,
		{Channel: optionsOrderbookUpdateChannel, Asset: asset.Options}:                      subscription.ErrInvalidLevel,
		{Channel: optionsOrderbookUpdateChannel, Asset: asset.Options, Levels: 5}:           nil,
		{Channel: optionsOrderbookUpdateChannel, Asset: asset.Options, Levels: 10}:          nil,
		{Channel: optionsOrderbookUpdateChannel, Asset: asset.Options, Levels: 20}:          nil,
		{Channel: optionsOrderbookUpdateChannel, Asset: asset.Options, Levels: 50}:          nil,
		{Channel: optionsOrderbookChannel, Asset: asset.Options}:                            subscription.ErrInvalidLevel,
		{Channel: optionsOrderbookChannel, Asset: asset.Options, Levels: 5}:                 nil,
		{Channel: optionsOrderbookChannel, Asset: asset.Options, Levels: 10}:                nil,
		{Channel: optionsOrderbookChannel, Asset: asset.Options, Levels: 20}:                nil,
		{Channel: optionsOrderbookChannel, Asset: asset.Options, Levels: 50}:                nil,
	} {
		t.Run(s.Asset.String()+"/"+s.Channel+"/"+strconv.Itoa(s.Levels), func(t *testing.T) {
			t.Parallel()
			l, err := channelLevels(s, s.Asset)
			switch {
			case exp != nil:
				require.ErrorIs(t, err, exp)
			case s.Levels == 0:
				assert.Empty(t, l)
			default:
				require.NoError(t, err)
				require.NotEmpty(t, l)
			}
		})
	}
}

func TestGetIntervalString(t *testing.T) {
	t.Parallel()
	for key, exp := range map[kline.Interval]string{
		kline.TenMilliseconds:                "10ms",
		kline.TwentyMilliseconds:             "20ms",
		kline.HundredMilliseconds:            "100ms",
		kline.TwoHundredAndFiftyMilliseconds: "250ms",
		kline.ThousandMilliseconds:           "1000ms",
		kline.TenSecond:                      "10s",
		kline.ThirtySecond:                   "30s",
		kline.OneMin:                         "1m",
		kline.FiveMin:                        "5m",
		kline.FifteenMin:                     "15m",
		kline.ThirtyMin:                      "30m",
		kline.OneHour:                        "1h",
		kline.TwoHour:                        "2h",
		kline.FourHour:                       "4h",
		kline.EightHour:                      "8h",
		kline.TwelveHour:                     "12h",
		kline.OneDay:                         "1d",
		kline.SevenDay:                       "7d",
		kline.OneMonth:                       "30d",
	} {
		t.Run(exp, func(t *testing.T) {
			t.Parallel()
			s, err := getIntervalString(key)
			require.NoError(t, err)
			assert.Equal(t, exp, s)
		})
	}
	_, err := getIntervalString(0)
	assert.ErrorIs(t, err, kline.ErrUnsupportedInterval, "0 should be an invalid interval")

	_, err = getIntervalString(kline.FiveDay)
	assert.ErrorIs(t, err, kline.ErrUnsupportedInterval, "Any other random interval should also be invalid")
}

func TestTransferBetweenSubAccountsByUID(t *testing.T) {
	t.Parallel()
	_, err := e.TransferBetweenSubAccountsByUID(t.Context(), &SubAccountTransfer{})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid)
	_, err = e.TransferBetweenSubAccountsByUID(t.Context(), &SubAccountTransfer{Amount: 1.})
	require.ErrorIs(t, err, errInvalidSubAccountUserID)

	_, err = e.TransferBetweenSubAccountsByUID(t.Context(), &SubAccountTransfer{Amount: 1., ReceiveUID: 2231234})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.TransferBetweenSubAccountsByUID(t.Context(), &SubAccountTransfer{
		Amount:     1.,
		ReceiveUID: 2231234,
		Currency:   currency.ETH,
	})
	require.NoError(t, err)
}

func TestGetOrderDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderDetails(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOrderDetails(t.Context(), "1234556")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestRepayMultiCollateraLoan(t *testing.T) {
	t.Parallel()
	_, err := e.RepayMultiCollateraLoan(t.Context(), &MultiCollateralLoanRepaymentParams{})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	_, err = e.RepayMultiCollateraLoan(t.Context(), &MultiCollateralLoanRepaymentParams{OrderID: "1234"})
	require.ErrorIs(t, err, currency.ErrCurrencyNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.RepayMultiCollateraLoan(t.Context(), &MultiCollateralLoanRepaymentParams{OrderID: "1234", RepayItems: []LoanRepaymentMultiCollateralAssetItem{{Currency: currency.USDT}}})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMultiCurrencyCollateralRepaymentRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetMultiCurrencyCollateralRepaymentRecords(t.Context(), "", currency.USDT, 0, 10, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errLoanTypeIsRequired)

	startTime, endTime := getTime()
	_, err = e.GetMultiCurrencyCollateralRepaymentRecords(t.Context(), "repay", currency.USDT, 0, 10, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetMultiCurrencyCollateralRepaymentRecords(t.Context(), "repay", currency.USDT, 0, 10, time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAddOrWithdrawCollateral(t *testing.T) {
	t.Parallel()
	_, err := e.AddOrWithdrawCollateral(t.Context(), &AddOrWithdrawCollateralParams{})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	_, err = e.AddOrWithdrawCollateral(t.Context(), &AddOrWithdrawCollateralParams{OrderID: 12344})
	require.ErrorIs(t, err, errLoanTypeIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.AddOrWithdrawCollateral(t.Context(), &AddOrWithdrawCollateralParams{
		OrderID:       12344,
		OperationType: "withdraw",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMultiCollateralAdjustmentRecords(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetMultiCollateralAdjustmentRecords(t.Context(), 1, 10, endTime, startTime, currency.ETH)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetMultiCollateralAdjustmentRecords(t.Context(), 1, 10, startTime, endTime, currency.ETH)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWebsocketSubmitOrders(t *testing.T) {
	t.Parallel()

	_, err := e.WebsocketSubmitOrders(t.Context(), nil)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	sub := &order.Submit{
		Exchange:    e.Name,
		AssetType:   asset.Spot,
		Side:        order.Buy,
		Type:        order.Market,
		QuoteAmount: 10,
	}
	_, err = e.WebsocketSubmitOrders(t.Context(), []*order.Submit{sub})
	require.ErrorIs(t, err, order.ErrPairIsEmpty)

	sub.Pair = currency.NewBTCUSD()
	cpy := *sub
	cpy.AssetType = asset.Futures
	_, err = e.WebsocketSubmitOrders(t.Context(), []*order.Submit{sub, &cpy})
	require.ErrorIs(t, err, errSingleAssetRequired)

	cpy.AssetType = asset.Spread
	sub.AssetType = asset.Spread
	_, err = e.WebsocketSubmitOrders(t.Context(), []*order.Submit{sub, &cpy})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	sub.AssetType = asset.USDTMarginedFutures
	cpy.AssetType = asset.USDTMarginedFutures
	_, err = e.WebsocketSubmitOrders(t.Context(), []*order.Submit{sub, &cpy})
	require.ErrorIs(t, err, common.ErrNotYetImplemented)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	e := newExchangeWithWebsocket(t, asset.Spot)
	sub.AssetType = asset.Spot
	cpy.AssetType = asset.Spot
	_, err = e.WebsocketSubmitOrders(request.WithVerbose(t.Context()), []*order.Submit{sub, &cpy})
	require.NoError(t, err)
}

func TestGetBrokerTransactionHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetBrokerTransactionHistory(t.Context(), currency.EMPTYPAIR, 0, endTime, startTime, 10, 0)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetBrokerTransactionHistory(t.Context(), currency.EMPTYPAIR, 0, startTime, endTime, 10, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBrokerRebateHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetBrokerRebateHistory(t.Context(), currency.USDT, 0, endTime, startTime, 10, 0)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetBrokerRebateHistory(t.Context(), currency.USDT, 0, startTime, endTime, 10, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPartnerRebateRecordsRecommendedUsers(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetPartnerRebateRecordsRecommendedUsers(t.Context(), currency.USDT, endTime, startTime, 10100011213, 10, 0)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetPartnerRebateRecordsRecommendedUsers(t.Context(), currency.USDT, startTime, endTime, 10100011213, 10, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPartnerSubordinateList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetPartnerSubordinateList(t.Context(), 12312312, 10, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestBrokerObtainsUserRebateRecords(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.BrokerObtainsUserRebateRecords(t.Context(), 12312312, endTime, startTime, 0, 0)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.BrokerObtainsUserRebateRecords(t.Context(), 12312312, startTime, endTime, 1, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetRebateBrokerTransactionHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetRebateBrokerTransactionHistory(t.Context(), 12312312, endTime, startTime, 0, 0)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetRebateBrokerTransactionHistory(t.Context(), 12312312, startTime, endTime, 1, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserRebateInformation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUserRebateInformation(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserSubordinateRelationship(t *testing.T) {
	t.Parallel()
	_, err := e.GetUserSubordinateRelationship(t.Context(), nil)
	require.ErrorIs(t, err, errUserIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUserSubordinateRelationship(t.Context(), []string{"12342", "21312312312"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestValidateContractOrderCreateParams(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		params *ContractOrderCreateParams
		isRest bool
		err    error
	}{
		{
			err: common.ErrNilPointer,
		},
		{
			params: &ContractOrderCreateParams{}, err: currency.ErrCurrencyPairEmpty,
		},
		{
			params: &ContractOrderCreateParams{Contract: BTCUSDT},
			err:    errInvalidOrderSize,
		},
		{
			params: &ContractOrderCreateParams{Contract: BTCUSDT, Size: 1, TimeInForce: "bad"},
			err:    order.ErrUnsupportedTimeInForce,
		},
		{
			params: &ContractOrderCreateParams{Contract: BTCUSDT, Size: 1, TimeInForce: pocTIF},
			err:    order.ErrUnsupportedTimeInForce,
		},
		{
			params: &ContractOrderCreateParams{Contract: BTCUSDT, Size: 1, TimeInForce: iocTIF, Text: "test"},
			err:    errInvalidOrderText,
		},
		{
			params: &ContractOrderCreateParams{
				Contract: BTCUSDT, Size: 1, TimeInForce: iocTIF, Text: "t-test", AutoSize: "silly_billy",
			},
			err: errInvalidAutoSize,
		},
		{
			params: &ContractOrderCreateParams{
				Contract: BTCUSDT, Size: 1, TimeInForce: iocTIF, Text: "t-test", AutoSize: "close_long",
			},
		},
		{
			params: &ContractOrderCreateParams{
				Contract: BTCUSDT, TimeInForce: iocTIF, Text: "t-test", AutoSize: "close_long",
			},
			isRest: true,
			err:    currency.ErrCurrencyCodeEmpty,
		},
		{
			params: &ContractOrderCreateParams{
				Contract: BTCUSDT, TimeInForce: iocTIF, Text: "t-test", AutoSize: "close_long", Settle: currency.NewCode("Silly"),
			},
			err: currency.ErrCurrencyCodeEmpty,
		},
		{
			params: &ContractOrderCreateParams{
				Contract: BTCUSDT, TimeInForce: iocTIF, Text: "t-test", AutoSize: "close_long", Settle: currency.USDT,
			},
		},
	} {
		assert.ErrorIs(t, tc.params.validate(tc.isRest), tc.err)
	}
}

func TestMarshalJSONNumber(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		number   number
		expected string
	}{
		{number: 0, expected: `"0"`},
		{number: 1, expected: `"1"`},
		{number: 1.5, expected: `"1.5"`},
	} {
		payload, err := tc.number.MarshalJSON()
		require.NoError(t, err, "MarshalJSON must not error")
		assert.Equal(t, tc.expected, string(payload), "MarshalJSON should return expected value")
	}
}

func TestUnmarshalJSONOrderbookLevels(t *testing.T) {
	t.Parallel()
	var ob OrderbookLevels
	require.NoError(t, ob.UnmarshalJSON([]byte(`[{"p":"123.45","s":"0.001"}]`)))
	assert.Equal(t, 123.45, ob[0].Price, "Price should be correct")
	assert.Equal(t, 0.001, ob[0].Amount, "Amount should be correct")

	require.Error(t, ob.UnmarshalJSON([]byte(`["p":"123.45","s":"0.001"]`)))
}

func TestGetEstimatedInterestRate(t *testing.T) {
	t.Parallel()

	_, err := e.GetEstimatedInterestRate(t.Context(), nil)
	require.ErrorIs(t, err, currency.ErrCurrencyCodesEmpty)

	_, err = e.GetEstimatedInterestRate(t.Context(), currency.Currencies{currency.EMPTYCODE})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.GetEstimatedInterestRate(t.Context(), currency.Currencies{
		currency.USDT,
		currency.BTC,
		currency.ETH,
		currency.XRP,
		currency.LTC,
		currency.DOGE,
		currency.BCH,
		currency.SOL,
		currency.ADA,
		currency.DOT,
		currency.MATIC,
	})
	require.ErrorIs(t, err, errTooManyCurrencyCodes)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	got, err := e.GetEstimatedInterestRate(t.Context(), currency.Currencies{currency.BTC})
	require.NoError(t, err)
	val, ok := got["BTC"]
	require.True(t, ok, "result map must contain BTC key")
	require.Positive(t, val.Float64(), "estimated interest rate must not be 0")
}

// Alpha endpoints unit tests

func TestGetAlphaAccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAlphaAccounts(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAlphaAccountTransactionHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetAlphaAccountTransactionHistory(t.Context(), time.Time{}, time.Time{}, 1, 10)
	require.ErrorIs(t, err, errStartTimeRequired)

	startTime, endTime := getTime()
	_, err = e.GetAlphaAccountTransactionHistory(t.Context(), endTime, startTime, 1, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAlphaAccountTransactionHistory(t.Context(), startTime, endTime, 1, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateAlphaCurrencyQuoteID(t *testing.T) {
	t.Parallel()
	_, err := e.CreateAlphaCurrencyQuoteID(t.Context(), &AlphaCurrencyQuoteInfoRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	arg := &AlphaCurrencyQuoteInfoRequest{Currency: currency.BTC}
	_, err = e.CreateAlphaCurrencyQuoteID(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = order.Sell
	_, err = e.CreateAlphaCurrencyQuoteID(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	arg.Amount = 1
	_, err = e.CreateAlphaCurrencyQuoteID(t.Context(), arg)
	require.ErrorIs(t, err, errGasModeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	arg.GasMode = "custom"
	result, err := e.CreateAlphaCurrencyQuoteID(t.Context(), arg)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPlaceAlphaTradeOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceAlphaTradeOrder(t.Context(), &AlphaCurrencyQuoteInfoRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	arg := &AlphaCurrencyQuoteInfoRequest{Currency: currency.BTC}
	_, err = e.PlaceAlphaTradeOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = order.Sell
	_, err = e.PlaceAlphaTradeOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	arg.Amount = 1
	_, err = e.PlaceAlphaTradeOrder(t.Context(), arg)
	require.ErrorIs(t, err, errGasModeRequired)

	arg.GasMode = "custom"
	_, err = e.PlaceAlphaTradeOrder(t.Context(), arg)
	require.ErrorIs(t, err, errQuoteIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	arg.QuoteID = "123345678"
	result, err := e.PlaceAlphaTradeOrder(t.Context(), arg)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAlphaOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetAlphaOrders(t.Context(), currency.EMPTYCODE, order.Sell, 0, time.Time{}, time.Time{}, 0, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.GetAlphaOrders(t.Context(), currency.ETH, order.Long, 0, time.Time{}, time.Time{}, 0, 10)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	startTime, endTime := getTime()
	_, err = e.GetAlphaOrders(t.Context(), currency.ETH, order.Sell, 0, endTime, startTime, 0, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAlphaOrders(t.Context(), currency.ETH, order.Sell, 1, startTime, endTime, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAlphaOrderByID(t *testing.T) {
	t.Parallel()
	_, err := e.GetAlphaOrderByID(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAlphaOrderByID(t.Context(), "123345678")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAlphaCurrenciesDetail(t *testing.T) {
	t.Parallel()
	result, err := e.GetAlphaCurrenciesDetail(t.Context(), currency.EMPTYCODE, 100, 10)
	require.NoError(t, err)
	require.NotEmpty(t, result)

	result, err = e.GetAlphaCurrenciesDetail(t.Context(), currency.NewCode("memeboxtrump"), 100, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAlphaCurrencyTicker(t *testing.T) {
	t.Parallel()
	result, err := e.GetAlphaCurrencyTicker(t.Context(), currency.EMPTYCODE, 100, 10)
	require.NoError(t, err)
	require.NotEmpty(t, result)

	result, err = e.GetAlphaCurrencyTicker(t.Context(), currency.NewCode("memeboxtrump"), 100, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLendingCurrencyList(t *testing.T) {
	t.Parallel()
	result, err := e.GetLendingCurrencyList(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLendingCurrencyDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetLendingCurrencyDetail(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	result, err := e.GetLendingCurrencyDetail(t.Context(), currency.ETH)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateLendingOrRedemption(t *testing.T) {
	t.Parallel()
	err := e.CreateLendingOrRedemption(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	arg := &LendingOrRedemptionRequest{}
	err = e.CreateLendingOrRedemption(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	arg.Currency = currency.USDT
	err = e.CreateLendingOrRedemption(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	arg.Amount = 1
	err = e.CreateLendingOrRedemption(t.Context(), arg)
	require.ErrorIs(t, err, errLoanTypeIsRequired)

	arg.Type = "lend"
	err = e.CreateLendingOrRedemption(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	arg.MinRate = 0.00000211
	err = e.CreateLendingOrRedemption(t.Context(), arg)
	require.NoError(t, err)
}

func TestGetUserLendingOrderList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUserLendingOrderList(t.Context(), currency.USDT, 10, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAmendUserLendingInformation(t *testing.T) {
	t.Parallel()
	err := e.AmendUserLendingInformation(t.Context(), currency.EMPTYCODE, 123.45)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	err = e.AmendUserLendingInformation(t.Context(), currency.USDT, 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.AmendUserLendingInformation(t.Context(), currency.USDT, 0.0001)
	require.NoError(t, err)
}

func TestGetLendingTransactionRecords(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetLendingTransactionRecords(t.Context(), currency.USDT, 1, 100, endTime, startTime, "lend")
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetLendingTransactionRecords(t.Context(), currency.USDT, 10, 100, startTime, endTime, "redeem")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserTotalInterestIncomePerCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.GetUserTotalInterestIncomePerCurrency(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUserTotalInterestIncomePerCurrency(t.Context(), currency.USDT)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserDividendRecords(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetUserDividendRecords(t.Context(), currency.USDT, 1, 100, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUserDividendRecords(t.Context(), currency.AE, 0, 0, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrencyInterestCompoundingStatus(t *testing.T) {
	t.Parallel()
	_, err := e.GetCurrencyInterestCompoundingStatus(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetCurrencyInterestCompoundingStatus(t.Context(), currency.USDT)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUniLoanCurrencyAnnualizedTrendChart(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetUniLoanCurrencyAnnualizedTrendChart(t.Context(), endTime, startTime, currency.USDT)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUniLoanCurrencyAnnualizedTrendChart(t.Context(), startTime, endTime, currency.AE)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrencyEstimatedAnnualizedInterestRate(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetCurrencyEstimatedAnnualizedInterestRate(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPlaceCollateralLoanOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceCollateralLoanOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	arg := &PlaceColateralLoanRequest{}
	_, err = e.PlaceCollateralLoanOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	arg.CollateralCurrency = currency.BTC
	_, err = e.PlaceCollateralLoanOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	arg.CollateralAmount = .5
	_, err = e.PlaceCollateralLoanOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	arg.BorrowCurrency = currency.USDT
	_, err = e.PlaceCollateralLoanOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	arg.BorrowAmount = 2000
	result, err := e.PlaceCollateralLoanOrder(t.Context(), arg)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCollateralLoanOrderList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetCollateralLoanOrderList(t.Context(), 1, 100, currency.USDT, currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSingleCollateralOrderDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetSingleCollateralOrderDetail(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSingleCollateralOrderDetail(t.Context(), "10000421")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestRepayCollateralLoan(t *testing.T) {
	t.Parallel()
	_, err := e.RepayCollateralLoan(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	arg := &CollateralLoanRepayRequest{}
	_, err = e.RepayCollateralLoan(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	arg.OrderID = 37438962
	_, err = e.RepayCollateralLoan(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	arg.RepayAmount = 100
	result, err := e.RepayCollateralLoan(t.Context(), arg)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCollateralLoanRepaymentRecords(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetCollateralLoanRepaymentRecords(t.Context(), "", 1, 100, currency.BTC, currency.USDT, startTime, endTime)
	require.ErrorIs(t, err, errOperationTypeRequired)

	_, err = e.GetCollateralLoanRepaymentRecords(t.Context(), "repay", 1, 100, currency.BTC, currency.USDT, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetCollateralLoanRepaymentRecords(t.Context(), "repay", 1, 100, currency.BTC, currency.USDT, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestIncreaseOrRedeemCollateral(t *testing.T) {
	t.Parallel()
	err := e.IncreaseOrRedeemCollateral(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	arg := &IncreaseOrRedeemCollateralRequest{}
	err = e.IncreaseOrRedeemCollateral(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	arg.OrderID = 123456
	err = e.IncreaseOrRedeemCollateral(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	arg.CollateralCurrency = currency.BTC
	err = e.IncreaseOrRedeemCollateral(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	arg.CollateralAmount = 2
	err = e.IncreaseOrRedeemCollateral(t.Context(), arg)
	require.ErrorIs(t, err, errOperationTypeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	arg.OperationType = "append"
	err = e.IncreaseOrRedeemCollateral(t.Context(), arg)
	require.NoError(t, err)
}

func TestGetCollateralAdjustmentRecords(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetCollateralAdjustmentRecords(t.Context(), 1, 10, currency.ETH, currency.USDT, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetCollateralAdjustmentRecords(t.Context(), 1, 10, currency.ETH, currency.USDT, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserTotalBorrowingAndCollateralAmount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUserTotalBorrowingAndCollateralAmount(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserCollateralizationRatioAndRemainingBorrowables(t *testing.T) {
	t.Parallel()
	_, err := e.GetUserCollateralizationRatioAndRemainingBorrowables(t.Context(), currency.EMPTYCODE, currency.USDT)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.GetUserCollateralizationRatioAndRemainingBorrowables(t.Context(), currency.BTC, currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUserCollateralizationRatioAndRemainingBorrowables(t.Context(), currency.BTC, currency.USDT)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSupportedBorrowingAndCollateralCurrencies(t *testing.T) {
	t.Parallel()
	_, err := e.GetSupportedBorrowingAndCollateralCurrencies(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSupportedBorrowingAndCollateralCurrencies(t.Context(), currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestListLendingMarkets(t *testing.T) {
	t.Parallel()
	result, err := e.ListLendingMarkets(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLendingMarketDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetLendingMarketDetail(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	result, err := e.GetLendingMarketDetail(t.Context(), getPair(t, asset.Margin))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestLendOrBorrow(t *testing.T) {
	t.Parallel()
	_, err := e.IsolatedMarginLendOrBorrow(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.IsolatedMarginLendOrBorrow(t.Context(), &LendOrBorrowRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.IsolatedMarginLendOrBorrow(t.Context(), &LendOrBorrowRequest{
		Currency: currency.BTC,
	})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.IsolatedMarginLendOrBorrow(t.Context(), &LendOrBorrowRequest{
		Currency:     currency.BTC,
		CurrencyPair: getPair(t, asset.Margin),
	})
	require.ErrorIs(t, err, margin.ErrInvalidMarginType)

	_, err = e.IsolatedMarginLendOrBorrow(t.Context(), &LendOrBorrowRequest{
		Currency:     currency.BTC,
		CurrencyPair: getPair(t, asset.Margin),
		OrderType:    "borrow",
	})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.IsolatedMarginLendOrBorrow(t.Context(), &LendOrBorrowRequest{
		Currency:     currency.BTC,
		CurrencyPair: getPair(t, asset.Margin),
		OrderType:    "borrow",
		Amount:       1,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestListAllIsolatedMarginLoans(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.ListAllIsolatedMarginLoans(t.Context(), getPair(t, asset.Margin), currency.BTC, 1, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIsolatedMarginLoanRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetIsolatedMarginLoanRecords(t.Context(), "borrow", currency.BTC, getPair(t, asset.Margin), 1, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginAccountInterestDeductionRecords(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetIsolatedMarginAccountInterestDeductionRecords(t.Context(), getPair(t, asset.Margin), currency.BTC, 1, 10, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetIsolatedMarginAccountInterestDeductionRecords(t.Context(), getPair(t, asset.Margin), currency.BTC, 1, 10, time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetIsolatedMarginAccountInterestDeductionRecords(t.Context(), getPair(t, asset.Margin), currency.BTC, 1, 10, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIsolatedMarginAccountMaximumBorrowableAmountByCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.GetIsolatedMarginAccountMaximumBorrowableAmountByCurrency(t.Context(), currency.EMPTYCODE, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.GetIsolatedMarginAccountMaximumBorrowableAmountByCurrency(t.Context(), currency.BTC, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetIsolatedMarginAccountMaximumBorrowableAmountByCurrency(t.Context(), currency.BTC, getPair(t, asset.Margin))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUsersOwnLverageLendingTiersInCurrentMarket(t *testing.T) {
	t.Parallel()
	_, err := e.GetUsersOwnLverageLendingTiersInCurrentMarket(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUsersOwnLverageLendingTiersInCurrentMarket(t.Context(), getPair(t, asset.Margin))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIsolatedMarginCurrentMarketLeverageLendingTiers(t *testing.T) {
	t.Parallel()
	_, err := e.GetIsolatedMarginCurrentMarketLeverageLendingTiers(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetIsolatedMarginCurrentMarketLeverageLendingTiers(t.Context(), getPair(t, asset.Margin))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetUserIsolatedMarginAccountMarketLeverageMultiplier(t *testing.T) {
	t.Parallel()
	_, err := e.SetUserIsolatedMarginAccountMarketLeverageMultiplier(t.Context(), getPair(t, asset.Margin), 0)
	require.ErrorIs(t, err, order.ErrSubmitLeverageNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SetUserIsolatedMarginAccountMarketLeverageMultiplier(t.Context(), getPair(t, asset.Margin), 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserIsolatedMarginAccountList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetUserIsolatedMarginAccountList(t.Context(), getPair(t, asset.Margin))
	require.NoError(t, err)

	result, err := e.GetUserIsolatedMarginAccountList(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err)
	assert.NotNil(t, result)
}
