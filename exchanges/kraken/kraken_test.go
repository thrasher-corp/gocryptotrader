package kraken

import (
	"errors"
	"log"
	"net/http"
	"os"
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
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

var (
	k               *Kraken
	spotTestPair    = currency.NewPair(currency.XBT, currency.USD)
	futuresTestPair = currency.NewPairWithDelimiter("PF", "XBTUSD", "_")
)

// Please add your own APIkeys to do correct due diligence testing.
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

func TestMain(m *testing.M) {
	k = new(Kraken)
	if err := testexch.Setup(k); err != nil {
		log.Fatalf("Kraken Setup error: %s", err)
	}
	if apiKey != "" && apiSecret != "" {
		k.API.AuthenticatedSupport = true
		k.SetCredentials(apiKey, apiSecret, "", "", "", "")
	}
	os.Exit(m.Run())
}

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, k)
}

func TestGetCurrentServerTime(t *testing.T) {
	t.Parallel()
	_, err := k.GetCurrentServerTime(t.Context())
	assert.NoError(t, err, "GetCurrentServerTime should not error")
}

func TestWrapperGetServerTime(t *testing.T) {
	t.Parallel()
	st, err := k.GetServerTime(t.Context(), asset.Spot)
	require.NoError(t, err, "GetServerTime must not error")
	assert.WithinRange(t, st, time.Now().Add(-24*time.Hour), time.Now().Add(24*time.Hour), "ServerTime should be within a day of now")
}

// TestUpdateOrderExecutionLimits exercises UpdateOrderExecutionLimits and GetOrderExecutionLimits
func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()

	err := k.UpdateOrderExecutionLimits(t.Context(), asset.Spot)
	require.NoError(t, err, "UpdateOrderExecutionLimits must not error")
	for _, p := range []currency.Pair{
		currency.NewPair(currency.ETH, currency.USDT),
		currency.NewPair(currency.XBT, currency.USDT),
	} {
		limits, err := k.GetOrderExecutionLimits(asset.Spot, p)
		require.NoErrorf(t, err, "%s GetOrderExecutionLimits must not error", p)
		assert.Positivef(t, limits.PriceStepIncrementSize, "%s PriceStepIncrementSize should be positive", p)
		assert.Positivef(t, limits.MinimumBaseAmount, "%s MinimumBaseAmount should be positive", p)
	}
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := k.FetchTradablePairs(t.Context(), asset.Futures)
	assert.NoError(t, err, "FetchTradablePairs should not error")
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, k)
	_, err := k.UpdateTicker(t.Context(), spotTestPair, asset.Spot)
	assert.NoError(t, err, "UpdateTicker spot asset should not error")

	_, err = k.UpdateTicker(t.Context(), futuresTestPair, asset.Futures)
	assert.NoError(t, err, "UpdateTicker futures asset should not error")
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()

	k := new(Kraken) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes
	require.NoError(t, testexch.Setup(k), "Test instance Setup must not error")

	testexch.UpdatePairsOnce(t, k)

	err := k.UpdateTickers(t.Context(), asset.Spot)
	require.NoError(t, err, "UpdateTickers must not error")

	ap, err := k.GetAvailablePairs(asset.Spot)
	require.NoError(t, err, "GetAvailablePairs must not error")

	for i := range ap {
		_, err = ticker.GetTicker(k.Name, ap[i], asset.Spot)
		assert.NoErrorf(t, err, "GetTicker should not error for %s", ap[i])
	}

	ap, err = k.GetAvailablePairs(asset.Futures)

	require.NoError(t, err, "GetAvailablePairs must not error")
	err = k.UpdateTickers(t.Context(), asset.Futures)
	require.NoError(t, err, "UpdateTickers must not error")

	for i := range ap {
		_, err = ticker.GetTicker(k.Name, ap[i], asset.Futures)
		assert.NoErrorf(t, err, "GetTicker should not error for %s", ap[i])
	}

	err = k.UpdateTickers(t.Context(), asset.Index)
	assert.ErrorIs(t, err, asset.ErrNotSupported, "UpdateTickers should error correctly for asset.Index")
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	_, err := k.UpdateOrderbook(t.Context(), spotTestPair, asset.Spot)
	assert.NoError(t, err, "UpdateOrderbook spot asset should not error")
	_, err = k.UpdateOrderbook(t.Context(), futuresTestPair, asset.Futures)
	assert.NoError(t, err, "UpdateOrderbook futures asset should not error")
}

func TestFuturesBatchOrder(t *testing.T) {
	t.Parallel()
	var data []PlaceBatchOrderData
	var tempData PlaceBatchOrderData
	tempData.PlaceOrderType = "meow"
	tempData.OrderID = "test123"
	tempData.Symbol = futuresTestPair.Lower().String()
	data = append(data, tempData)
	_, err := k.FuturesBatchOrder(t.Context(), data)
	assert.ErrorIs(t, err, errInvalidBatchOrderType, "FuturesBatchOrder should error correctly")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, k, canManipulateRealOrders)

	data[0].PlaceOrderType = "cancel"
	_, err = k.FuturesBatchOrder(t.Context(), data)
	assert.NoError(t, err, "FuturesBatchOrder should not error")
}

func TestFuturesEditOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, k, canManipulateRealOrders)

	_, err := k.FuturesEditOrder(t.Context(), "test123", "", 5.2, 1, 0)
	assert.NoError(t, err, "FuturesEditOrder should not error")
}

func TestFuturesSendOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, k, canManipulateRealOrders)

	_, err := k.FuturesSendOrder(t.Context(), order.Limit, futuresTestPair, "buy", "", "", "", order.ImmediateOrCancel, 1, 1, 0.9)
	assert.NoError(t, err, "FuturesSendOrder should not error")
}

func TestFuturesCancelOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, k, canManipulateRealOrders)

	_, err := k.FuturesCancelOrder(t.Context(), "test123", "")
	assert.NoError(t, err, "FuturesCancelOrder should not error")
}

func TestFuturesGetFills(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, k)

	_, err := k.FuturesGetFills(t.Context(), time.Now().Add(-time.Hour*24))
	assert.NoError(t, err, "FuturesGetFills should not error")
}

func TestFuturesTransfer(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, k)

	_, err := k.FuturesTransfer(t.Context(), "cash", "futures", "btc", 2)
	assert.NoError(t, err, "FuturesTransfer should not error")
}

func TestFuturesGetOpenPositions(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, k)

	_, err := k.FuturesGetOpenPositions(t.Context())
	assert.NoError(t, err, "FuturesGetOpenPositions should not error")
}

func TestFuturesNotifications(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, k)

	_, err := k.FuturesNotifications(t.Context())
	assert.NoError(t, err, "FuturesNotifications should not error")
}

func TestFuturesCancelAllOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, k, canManipulateRealOrders)

	_, err := k.FuturesCancelAllOrders(t.Context(), futuresTestPair)
	assert.NoError(t, err, "FuturesCancelAllOrders should not error")
}

func TestGetFuturesAccountData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, k)

	_, err := k.GetFuturesAccountData(t.Context())
	assert.NoError(t, err, "GetFuturesAccountData should not error")
}

func TestFuturesCancelAllOrdersAfter(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, k, canManipulateRealOrders)

	_, err := k.FuturesCancelAllOrdersAfter(t.Context(), 50)
	assert.NoError(t, err, "FuturesCancelAllOrdersAfter should not error")
}

func TestFuturesOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, k)

	_, err := k.FuturesOpenOrders(t.Context())
	assert.NoError(t, err, "FuturesOpenOrders should not error")
}

func TestFuturesRecentOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, k)

	_, err := k.FuturesRecentOrders(t.Context(), futuresTestPair)
	assert.NoError(t, err, "FuturesRecentOrders should not error")
}

func TestFuturesWithdrawToSpotWallet(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, k, canManipulateRealOrders)

	_, err := k.FuturesWithdrawToSpotWallet(t.Context(), "xbt", 5)
	assert.NoError(t, err, "FuturesWithdrawToSpotWallet should not error")
}

func TestFuturesGetTransfers(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, k, canManipulateRealOrders)

	_, err := k.FuturesGetTransfers(t.Context(), time.Now().Add(-time.Hour*24))
	assert.NoError(t, err, "FuturesGetTransfers should not error")
}

func TestGetFuturesOrderbook(t *testing.T) {
	t.Parallel()
	_, err := k.GetFuturesOrderbook(t.Context(), futuresTestPair)
	assert.NoError(t, err, "GetFuturesOrderbook should not error")
}

func TestGetFuturesMarkets(t *testing.T) {
	t.Parallel()
	_, err := k.GetInstruments(t.Context())
	assert.NoError(t, err, "GetInstruments should not error")
}

func TestGetFuturesTickers(t *testing.T) {
	t.Parallel()
	_, err := k.GetFuturesTickers(t.Context())
	assert.NoError(t, err, "GetFuturesTickers should not error")
}

func TestGetFuturesTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := k.GetFuturesTradeHistory(t.Context(), futuresTestPair, time.Now().Add(-time.Hour*24))
	assert.NoError(t, err, "GetFuturesTradeHistory should not error")
}

// TestGetAssets API endpoint test
func TestGetAssets(t *testing.T) {
	t.Parallel()
	_, err := k.GetAssets(t.Context())
	assert.NoError(t, err, "GetAssets should not error")
}

func TestSeedAssetTranslator(t *testing.T) {
	t.Parallel()

	err := k.SeedAssets(t.Context())
	require.NoError(t, err, "SeedAssets must not error")

	for from, to := range map[string]string{"XBTUSD": "XXBTZUSD", "USD": "ZUSD", "XBT": "XXBT"} {
		assert.Equal(t, from, assetTranslator.LookupAltName(to), "LookupAltName should return the correct value")
		assert.Equal(t, to, assetTranslator.LookupCurrency(from), "LookupCurrency should return the correct value")
	}
}

func TestSeedAssets(t *testing.T) {
	t.Parallel()
	var a assetTranslatorStore
	assert.Empty(t, a.LookupAltName("ZUSD"), "LookupAltName on unseeded store should return empty")
	a.Seed("ZUSD", "USD")
	assert.Equal(t, "USD", a.LookupAltName("ZUSD"), "LookupAltName should return the correct value")
	a.Seed("ZUSD", "BLA")
	assert.Equal(t, "USD", a.LookupAltName("ZUSD"), "Store should ignore second reseed of existing currency")
}

func TestLookupCurrency(t *testing.T) {
	t.Parallel()
	var a assetTranslatorStore
	assert.Empty(t, a.LookupCurrency("USD"), "LookupCurrency on unseeded store should return empty")
	a.Seed("ZUSD", "USD")
	assert.Equal(t, "ZUSD", a.LookupCurrency("USD"), "LookupCurrency should return the correct value")
	assert.Empty(t, a.LookupCurrency("EUR"), "LookupCurrency should still not return an unseeded key")
}

// TestGetAssetPairs API endpoint test
func TestGetAssetPairs(t *testing.T) {
	t.Parallel()
	for _, v := range []string{"fees", "leverage", "margin", ""} {
		_, err := k.GetAssetPairs(t.Context(), []string{}, v)
		require.NoErrorf(t, err, "GetAssetPairs %s must not error", v)
	}
}

// TestGetTicker API endpoint test
func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := k.GetTicker(t.Context(), spotTestPair)
	assert.NoError(t, err, "GetTicker should not error")
}

// TestGetTickers API endpoint test
func TestGetTickers(t *testing.T) {
	t.Parallel()
	_, err := k.GetTickers(t.Context(), "LTCUSD,ETCUSD")
	assert.NoError(t, err, "GetTickers should not error")
}

// TestGetOHLC API endpoint test
func TestGetOHLC(t *testing.T) {
	t.Parallel()
	_, err := k.GetOHLC(t.Context(), currency.NewPairWithDelimiter("XXBT", "ZUSD", ""), "1440")
	assert.NoError(t, err, "GetOHLC should not error")
}

// TestGetDepth API endpoint test
func TestGetDepth(t *testing.T) {
	t.Parallel()
	_, err := k.GetDepth(t.Context(), spotTestPair)
	assert.NoError(t, err, "GetDepth should not error")
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, k)
	r, err := k.GetTrades(t.Context(), spotTestPair, time.Now().Add(-time.Hour*4), 1000)
	require.NoError(t, err, "GetTrades must not error")
	require.NotNil(t, r, "GetTrades must return a valid response")
}

func TestGetSpread(t *testing.T) {
	t.Parallel()
	p := currency.NewPair(currency.BCH, currency.EUR) // XBTUSD not in spread data
	r, err := k.GetSpread(t.Context(), p, time.Now().Add(-time.Hour*4))
	require.NoError(t, err, "GetSpread must not error")
	require.NotNil(t, r, "GetSpread must return a valid response")
	require.NotZero(t, r.Last.Time(), "GetSpread must return a valid last updated time")
	v, ok := r.Spreads[p.String()]
	require.True(t, ok, "GetSpread must return valid spread data for the given pair")
	assert.NotEmpty(t, v, "GetSpread should return some spread data for the given pair")
}

// TestGetBalance API endpoint test
func TestGetBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, k)
	_, err := k.GetBalance(t.Context())
	assert.NoError(t, err, "GetBalance should not error")
}

func TestGetDepositMethods(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, k)
	_, err := k.GetDepositMethods(t.Context(), "USDT")
	assert.NoError(t, err, "GetDepositMethods should not error")
}

// TestGetTradeBalance API endpoint test
func TestGetTradeBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, k)
	args := TradeBalanceOptions{Asset: "ZEUR"}
	_, err := k.GetTradeBalance(t.Context(), args)
	assert.NoError(t, err)
}

// TestGetOpenOrders API endpoint test
func TestGetOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, k)
	args := OrderInfoOptions{Trades: true}
	_, err := k.GetOpenOrders(t.Context(), args)
	assert.NoError(t, err)
}

// TestGetClosedOrders API endpoint test
func TestGetClosedOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, k)
	args := GetClosedOrdersOptions{Trades: true, Start: "OE4KV4-4FVQ5-V7XGPU"}
	_, err := k.GetClosedOrders(t.Context(), args)
	assert.NoError(t, err)
}

// TestQueryOrdersInfo API endpoint test
func TestQueryOrdersInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, k)
	args := OrderInfoOptions{Trades: true}
	_, err := k.QueryOrdersInfo(t.Context(), args, "OR6ZFV-AA6TT-CKFFIW", "OAMUAJ-HLVKG-D3QJ5F")
	assert.NoError(t, err)
}

// TestGetTradesHistory API endpoint test
func TestGetTradesHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, k)
	args := GetTradesHistoryOptions{Trades: true, Start: "TMZEDR-VBJN2-NGY6DX", End: "TVRXG2-R62VE-RWP3UW"}
	_, err := k.GetTradesHistory(t.Context(), args)
	assert.NoError(t, err)
}

// TestQueryTrades API endpoint test
func TestQueryTrades(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, k)
	_, err := k.QueryTrades(t.Context(), true, "TMZEDR-VBJN2-NGY6DX", "TFLWIB-KTT7L-4TWR3L", "TDVRAH-2H6OS-SLSXRX")
	assert.NoError(t, err)
}

// TestOpenPositions API endpoint test
func TestOpenPositions(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, k)
	_, err := k.OpenPositions(t.Context(), false)
	assert.NoError(t, err)
}

// TestGetLedgers API endpoint test
// TODO: Needs a positive test
func TestGetLedgers(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, k)

	args := GetLedgersOptions{Start: "LRUHXI-IWECY-K4JYGO", End: "L5NIY7-JZQJD-3J4M2V", Ofs: 15}
	_, err := k.GetLedgers(t.Context(), args)
	assert.ErrorContains(t, err, "EQuery:Unknown asset pair", "GetLedger should error on imaginary ledgers")
}

// TestQueryLedgers API endpoint test
func TestQueryLedgers(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, k)
	_, err := k.QueryLedgers(t.Context(), "LVTSFS-NHZVM-EXNZ5M")
	assert.NoError(t, err)
}

// TestGetTradeVolume API endpoint test
func TestGetTradeVolume(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, k)
	_, err := k.GetTradeVolume(t.Context(), true, spotTestPair)
	assert.NoError(t, err, "GetTradeVolume should not error")
}

// TestOrders Tests AddOrder and CancelExistingOrder
func TestOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, k, canManipulateRealOrders)

	args := AddOrderOptions{OrderFlags: "fcib"}
	cp, err := currency.NewPairFromString("XXBTZUSD")
	assert.NoError(t, err, "NewPairFromString should not error")
	resp, err := k.AddOrder(t.Context(),
		cp,
		order.Buy.Lower(), order.Limit.Lower(),
		0.0001, 9000, 9000, 0, &args)

	if assert.NoError(t, err, "AddOrder should not error") {
		if assert.Len(t, resp.TransactionIDs, 1, "One TransactionId should be returned") {
			id := resp.TransactionIDs[0]
			_, err = k.CancelExistingOrder(t.Context(), id)
			assert.NoErrorf(t, err, "CancelExistingOrder should not error, Please ensure order %s is cancelled manually", id)
		}
	}
}

// TestCancelExistingOrder API endpoint test
func TestCancelExistingOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, k, canManipulateRealOrders)
	_, err := k.CancelExistingOrder(t.Context(), "OAVY7T-MV5VK-KHDF5X")
	if assert.Error(t, err, "Cancel with imaginary order-id should error") {
		assert.ErrorContains(t, err, "EOrder:Unknown order", "Cancel with imaginary order-id should error Unknown Order")
	}
}

func setFeeBuilder() *exchange.FeeBuilder {
	return &exchange.FeeBuilder{
		Amount:              1,
		FeeType:             exchange.CryptocurrencyTradeFee,
		Pair:                currency.NewPair(currency.XXBT, currency.ZUSD),
		PurchasePrice:       1,
		FiatCurrency:        currency.USD,
		BankTransactionType: exchange.WireTransfer,
	}
}

// TestGetFeeByTypeOfflineTradeFee logic test
func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	t.Parallel()
	feeBuilder := setFeeBuilder()
	f, err := k.GetFeeByType(t.Context(), feeBuilder)
	require.NoError(t, err, "GetFeeByType must not error")
	assert.Positive(t, f, "GetFeeByType should return a positive value")
	if !sharedtestvalues.AreAPICredentialsSet(k) {
		assert.Equal(t, exchange.OfflineTradeFee, feeBuilder.FeeType, "GetFeeByType should set FeeType correctly")
	} else {
		assert.Equal(t, exchange.CryptocurrencyTradeFee, feeBuilder.FeeType, "GetFeeByType should set FeeType correctly")
	}
}

// TestGetFee exercises GetFee
func TestGetFee(t *testing.T) {
	t.Parallel()
	feeBuilder := setFeeBuilder()

	if sharedtestvalues.AreAPICredentialsSet(k) {
		_, err := k.GetFee(t.Context(), feeBuilder)
		assert.NoError(t, err, "CryptocurrencyTradeFee Basic GetFee should not error")

		feeBuilder = setFeeBuilder()
		feeBuilder.Amount = 1000
		feeBuilder.PurchasePrice = 1000
		_, err = k.GetFee(t.Context(), feeBuilder)
		assert.NoError(t, err, "CryptocurrencyTradeFee High quantity GetFee should not error")

		feeBuilder = setFeeBuilder()
		feeBuilder.IsMaker = true
		_, err = k.GetFee(t.Context(), feeBuilder)
		assert.NoError(t, err, "CryptocurrencyTradeFee IsMaker GetFee should not error")

		feeBuilder = setFeeBuilder()
		feeBuilder.PurchasePrice = -1000
		_, err = k.GetFee(t.Context(), feeBuilder)
		assert.NoError(t, err, "CryptocurrencyTradeFee Negative purchase price GetFee should not error")

		feeBuilder = setFeeBuilder()
		feeBuilder.FeeType = exchange.InternationalBankDepositFee
		_, err = k.GetFee(t.Context(), feeBuilder)
		assert.NoError(t, err, "InternationalBankDepositFee Basic GetFee should not error")
	}

	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyDepositFee
	feeBuilder.Pair.Base = currency.XXBT
	_, err := k.GetFee(t.Context(), feeBuilder)
	assert.NoError(t, err, "CryptocurrencyDepositFee Basic GetFee should not error")

	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	_, err = k.GetFee(t.Context(), feeBuilder)
	assert.NoError(t, err, "CryptocurrencyWithdrawalFee Basic GetFee should not error")

	feeBuilder = setFeeBuilder()
	feeBuilder.Pair.Base = currency.NewCode("hello")
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	_, err = k.GetFee(t.Context(), feeBuilder)
	assert.NoError(t, err, "CryptocurrencyWithdrawalFee Invalid currency GetFee should not error")

	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.USD
	_, err = k.GetFee(t.Context(), feeBuilder)
	assert.NoError(t, err, "InternationalBankWithdrawalFee Basic GetFee should not error")
}

// TestFormatWithdrawPermissions logic test
func TestFormatWithdrawPermissions(t *testing.T) {
	t.Parallel()
	exp := exchange.AutoWithdrawCryptoWithSetupText + " & " + exchange.WithdrawCryptoWith2FAText + " & " + exchange.AutoWithdrawFiatWithSetupText + " & " + exchange.WithdrawFiatWith2FAText
	withdrawPermissions := k.FormatWithdrawPermissions()
	assert.Equal(t, exp, withdrawPermissions, "FormatWithdrawPermissions should return correct value")
}

// TestGetActiveOrders wrapper test
func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, k)

	getOrdersRequest := order.MultiOrderRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
		Pairs:     currency.Pairs{spotTestPair},
		Side:      order.AnySide,
	}

	_, err := k.GetActiveOrders(t.Context(), &getOrdersRequest)
	assert.NoError(t, err, "GetActiveOrders should not error")
}

// TestGetOrderHistory wrapper test
func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, k)

	getOrdersRequest := order.MultiOrderRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
		Side:      order.AnySide,
	}

	_, err := k.GetOrderHistory(t.Context(), &getOrdersRequest)
	assert.NoError(t, err)
}

// TestGetOrderInfo exercises GetOrderInfo
func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, k)
	_, err := k.GetOrderInfo(t.Context(), "OZPTPJ-HVYHF-EDIGXS", currency.EMPTYPAIR, asset.Spot)
	assert.ErrorContains(t, err, "order OZPTPJ-HVYHF-EDIGXS not found in response", "Should error that order was not found in response")
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------

// TestSubmitOrder wrapper test
func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, k, canManipulateRealOrders)

	orderSubmission := &order.Submit{
		Exchange:  k.Name,
		Pair:      spotTestPair,
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     1,
		Amount:    1,
		ClientID:  "meowOrder",
		AssetType: asset.Spot,
	}
	response, err := k.SubmitOrder(t.Context(), orderSubmission)
	if sharedtestvalues.AreAPICredentialsSet(k) {
		assert.NoError(t, err, "SubmitOrder should not error")
		assert.Equal(t, order.New, response.Status, "SubmitOrder should return a New order status")
	} else {
		assert.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled, "SubmitOrder should error correctly")
	}
}

// TestCancelExchangeOrder wrapper test
func TestCancelExchangeOrder(t *testing.T) {
	t.Parallel()

	err := k.CancelOrder(t.Context(), &order.Cancel{
		AssetType: asset.Options,
		OrderID:   "1337",
	})
	assert.ErrorIs(t, err, asset.ErrNotSupported, "CancelOrder should error on Options asset")

	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, k, canManipulateRealOrders)

	orderCancellation := &order.Cancel{
		OrderID:   "OGEX6P-B5Q74-IGZ72R",
		AssetType: asset.Spot,
	}

	err = k.CancelOrder(t.Context(), orderCancellation)
	if sharedtestvalues.AreAPICredentialsSet(k) {
		assert.NoError(t, err, "CancelOrder should not error")
	} else {
		assert.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled, "CancelOrder should error correctly")
	}
}

func TestCancelBatchExchangeOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, k, canManipulateRealOrders)

	var ordersCancellation []order.Cancel
	ordersCancellation = append(ordersCancellation, order.Cancel{
		Pair:      currency.NewPairWithDelimiter(currency.BTC.String(), currency.USD.String(), "/"),
		OrderID:   "OGEX6P-B5Q74-IGZ72R,OGEX6P-B5Q74-IGZ722",
		AssetType: asset.Spot,
	})

	_, err := k.CancelBatchOrders(t.Context(), ordersCancellation)
	if sharedtestvalues.AreAPICredentialsSet(k) {
		assert.NoError(t, err, "CancelBatchOrder should not error")
	} else {
		assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "CancelBatchOrders should error correctly")
	}
}

// TestCancelAllExchangeOrders wrapper test
func TestCancelAllExchangeOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, k, canManipulateRealOrders)

	resp, err := k.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Spot})

	if sharedtestvalues.AreAPICredentialsSet(k) {
		assert.NoError(t, err, "CancelAllOrders should not error")
	} else {
		assert.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled, "CancelBatchOrders should error correctly")
	}

	assert.Empty(t, resp.Status, "CancelAllOrders Status should not contain any failed order errors")
}

// TestUpdateAccountInfo exercises UpdateAccountInfo
func TestUpdateAccountInfo(t *testing.T) {
	t.Parallel()

	for _, a := range []asset.Item{asset.Spot, asset.Futures} {
		_, err := k.UpdateAccountInfo(t.Context(), a)

		if sharedtestvalues.AreAPICredentialsSet(k) {
			assert.NoErrorf(t, err, "UpdateAccountInfo should not error for asset %s", a) // Note Well: Spot and Futures have separate api keys
		} else {
			assert.ErrorIsf(t, err, exchange.ErrAuthenticationSupportNotEnabled, "UpdateAccountInfo should error correctly for asset %s", a)
		}
	}
}

// TestModifyOrder wrapper test
func TestModifyOrder(t *testing.T) {
	t.Parallel()

	_, err := k.ModifyOrder(t.Context(), &order.Modify{AssetType: asset.Spot})
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "ModifyOrder should error correctly")
}

// TestWithdraw wrapper test
func TestWithdraw(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, k, canManipulateRealOrders)

	withdrawCryptoRequest := withdraw.Request{
		Exchange: k.Name,
		Crypto: withdraw.CryptoRequest{
			Address: core.BitcoinDonationAddress,
		},
		Amount:        -1,
		Currency:      currency.XXBT,
		Description:   "WITHDRAW IT ALL",
		TradePassword: "Key",
	}

	_, err := k.WithdrawCryptocurrencyFunds(t.Context(),
		&withdrawCryptoRequest)
	if !sharedtestvalues.AreAPICredentialsSet(k) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if sharedtestvalues.AreAPICredentialsSet(k) && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

// TestWithdrawFiat wrapper test
func TestWithdrawFiat(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, k, canManipulateRealOrders)

	withdrawFiatRequest := withdraw.Request{
		Amount:        -1,
		Currency:      currency.EUR,
		Description:   "WITHDRAW IT ALL",
		TradePassword: "someBank",
	}

	_, err := k.WithdrawFiatFunds(t.Context(), &withdrawFiatRequest)
	if !sharedtestvalues.AreAPICredentialsSet(k) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if sharedtestvalues.AreAPICredentialsSet(k) && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

// TestWithdrawInternationalBank wrapper test
func TestWithdrawInternationalBank(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, k, canManipulateRealOrders)

	withdrawFiatRequest := withdraw.Request{
		Amount:        -1,
		Currency:      currency.EUR,
		Description:   "WITHDRAW IT ALL",
		TradePassword: "someBank",
	}

	_, err := k.WithdrawFiatFundsToInternationalBank(t.Context(),
		&withdrawFiatRequest)
	if !sharedtestvalues.AreAPICredentialsSet(k) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if sharedtestvalues.AreAPICredentialsSet(k) && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

func TestGetCryptoDepositAddress(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, k)

	_, err := k.GetCryptoDepositAddress(t.Context(), "Bitcoin", "XBT", false)
	if err != nil {
		t.Error(err)
	}
	if !canManipulateRealOrders {
		t.Skip("canManipulateRealOrders not set, skipping test")
	}
	_, err = k.GetCryptoDepositAddress(t.Context(), "Bitcoin", "XBT", true)
	if err != nil {
		t.Error(err)
	}
}

// TestGetDepositAddress wrapper test
func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	if sharedtestvalues.AreAPICredentialsSet(k) {
		_, err := k.GetDepositAddress(t.Context(), currency.USDT, "", "")
		if err != nil {
			t.Error("GetDepositAddress() error", err)
		}
	} else {
		_, err := k.GetDepositAddress(t.Context(), currency.BTC, "", "")
		if err == nil {
			t.Error("GetDepositAddress() error can not be nil")
		}
	}
}

// TestWithdrawStatus wrapper test
func TestWithdrawStatus(t *testing.T) {
	t.Parallel()
	if sharedtestvalues.AreAPICredentialsSet(k) {
		_, err := k.WithdrawStatus(t.Context(), currency.BTC, "")
		if err != nil {
			t.Error("WithdrawStatus() error", err)
		}
	} else {
		_, err := k.WithdrawStatus(t.Context(), currency.BTC, "")
		if err == nil {
			t.Error("GetDepositAddress() error can not be nil")
		}
	}
}

// TestWithdrawCancel wrapper test
func TestWithdrawCancel(t *testing.T) {
	t.Parallel()
	_, err := k.WithdrawCancel(t.Context(), currency.BTC, "")
	if sharedtestvalues.AreAPICredentialsSet(k) && err == nil {
		t.Error("WithdrawCancel() error cannot be nil")
	} else if !sharedtestvalues.AreAPICredentialsSet(k) && err == nil {
		t.Errorf("WithdrawCancel() error - expecting an error when no keys are set but received nil")
	}
}

// ---------------------------- Websocket tests -----------------------------------------

// TestWsSubscribe tests unauthenticated websocket subscriptions
// Specifically looking to ensure multiple errors are collected and returned and ws.Subscriptions Added/Removed in cases of:
// single pass, single fail, mixed fail, multiple pass, all fail
// No objection to this becoming a fixture test, so long as it integrates through Un/Subscribe roundtrip
func TestWsSubscribe(t *testing.T) {
	k := new(Kraken) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes
	require.NoError(t, testexch.Setup(k), "Setup Instance must not error")
	testexch.SetupWs(t, k)

	for _, enabled := range []bool{false, true} {
		require.NoError(t, k.SetPairs(currency.Pairs{
			spotTestPair,
			currency.NewPairWithDelimiter("ETH", "USD", "/"),
			currency.NewPairWithDelimiter("LTC", "ETH", "/"),
			currency.NewPairWithDelimiter("ETH", "XBT", "/"),
			// Enable pairs that won't error locally, so we get upstream errors to test error combinations
			currency.NewPairWithDelimiter("DWARF", "HOBBIT", "/"),
			currency.NewPairWithDelimiter("DWARF", "GOBLIN", "/"),
			currency.NewPairWithDelimiter("DWARF", "ELF", "/"),
		}, asset.Spot, enabled), "SetPairs must not error")
	}

	err := k.Subscribe(subscription.List{{Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: currency.Pairs{spotTestPair}}})
	require.NoError(t, err, "Simple subscription must not error")
	subs := k.Websocket.GetSubscriptions()
	require.Len(t, subs, 1, "Should add 1 Subscription")
	assert.Equal(t, subscription.SubscribedState, subs[0].State(), "Subscription should be subscribed state")

	err = k.Subscribe(subscription.List{{Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: currency.Pairs{spotTestPair}}})
	assert.ErrorIs(t, err, subscription.ErrDuplicate, "Resubscribing to the same channel should error with SubscribedAlready")
	subs = k.Websocket.GetSubscriptions()
	require.Len(t, subs, 1, "Should not add a subscription on error")
	assert.Equal(t, subscription.SubscribedState, subs[0].State(), "Existing subscription state should not change")

	err = k.Subscribe(subscription.List{{Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: currency.Pairs{currency.NewPairWithDelimiter("DWARF", "HOBBIT", "/")}}})
	assert.ErrorContains(t, err, "Currency pair not supported; Channel: ticker Pairs: DWARF/HOBBIT", "Subscribing to an invalid pair should error correctly")
	require.Len(t, k.Websocket.GetSubscriptions(), 1, "Should not add a subscription on error")

	// Mix success and failure
	err = k.Subscribe(subscription.List{
		{Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: currency.Pairs{currency.NewPairWithDelimiter("ETH", "USD", "/")}},
		{Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: currency.Pairs{currency.NewPairWithDelimiter("DWARF", "HOBBIT", "/")}},
		{Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: currency.Pairs{currency.NewPairWithDelimiter("DWARF", "ELF", "/")}},
	})
	assert.ErrorContains(t, err, "Currency pair not supported; Channel: ticker Pairs:", "Subscribing to an invalid pair should error correctly")
	assert.ErrorContains(t, err, "DWARF/HOBBIT", "Subscribing to an invalid pair should error correctly")
	assert.ErrorContains(t, err, "DWARF/ELF", "Subscribing to an invalid pair should error correctly")
	require.Len(t, k.Websocket.GetSubscriptions(), 2, "Should have 2 subscriptions after mixed success/failures")

	// Just failures
	err = k.Subscribe(subscription.List{
		{Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: currency.Pairs{currency.NewPairWithDelimiter("DWARF", "HOBBIT", "/")}},
		{Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: currency.Pairs{currency.NewPairWithDelimiter("DWARF", "GOBLIN", "/")}},
	})
	assert.ErrorContains(t, err, "Currency pair not supported; Channel: ticker Pairs:", "Subscribing to an invalid pair should error correctly")
	assert.ErrorContains(t, err, "DWARF/HOBBIT", "Subscribing to an invalid pair should error correctly")
	assert.ErrorContains(t, err, "DWARF/GOBLIN", "Subscribing to an invalid pair should error correctly")
	require.Len(t, k.Websocket.GetSubscriptions(), 2, "Should have 2 subscriptions after mixed success/failures")

	// Just success
	err = k.Subscribe(subscription.List{
		{Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: currency.Pairs{currency.NewPairWithDelimiter("ETH", "XBT", "/")}},
		{Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: currency.Pairs{currency.NewPairWithDelimiter("LTC", "ETH", "/")}},
	})
	assert.NoError(t, err, "Multiple successful subscriptions should not error")

	subs = k.Websocket.GetSubscriptions()
	assert.Len(t, subs, 4, "Should have correct number of subscriptions")

	err = k.Unsubscribe(subs[:1])
	assert.NoError(t, err, "Simple Unsubscribe should succeed")
	assert.Len(t, k.Websocket.GetSubscriptions(), 3, "Should have removed 1 channel")

	err = k.Unsubscribe(subscription.List{{Channel: subscription.TickerChannel, Pairs: currency.Pairs{currency.NewPairWithDelimiter("DWARF", "WIZARD", "/")}, Key: 1337}})
	assert.ErrorIs(t, err, subscription.ErrNotFound, "Simple failing Unsubscribe should error NotFound")
	assert.ErrorContains(t, err, "DWARF/WIZARD", "Unsubscribing from an invalid pair should error correctly")
	assert.Len(t, k.Websocket.GetSubscriptions(), 3, "Should not have removed any channels")

	err = k.Unsubscribe(subscription.List{
		subs[1],
		{Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: currency.Pairs{currency.NewPairWithDelimiter("DWARF", "EAGLE", "/")}, Key: 1338},
	})
	assert.ErrorIs(t, err, subscription.ErrNotFound, "Mixed failing Unsubscribe should error NotFound")
	assert.ErrorContains(t, err, "Channel: ticker Pairs: DWARF/EAGLE", "Unsubscribing from an invalid pair should error correctly")

	subs = k.Websocket.GetSubscriptions()
	assert.Len(t, subs, 2, "Should have removed only 1 more channel")

	err = k.Unsubscribe(subs)
	assert.NoError(t, err, "Unsubscribe multiple passing subscriptions should not error")
	assert.Empty(t, k.Websocket.GetSubscriptions(), "Should have successfully removed all channels")

	for _, c := range []string{"ohlc", "ohlc-5"} {
		err = k.Subscribe(subscription.List{{
			Asset:   asset.Spot,
			Channel: c,
			Pairs:   currency.Pairs{spotTestPair},
		}})
		assert.ErrorIs(t, err, subscription.ErrUseConstChannelName, "Must error when trying to use a private channel name")
		assert.ErrorContains(t, err, c+" => subscription.CandlesChannel", "Must error when trying to use a private channel name")
	}
}

// TestWsResubscribe tests websocket resubscription
func TestWsResubscribe(t *testing.T) {
	k := new(Kraken) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes
	require.NoError(t, testexch.Setup(k), "TestInstance must not error")
	testexch.SetupWs(t, k)

	err := k.Subscribe(subscription.List{{Asset: asset.Spot, Channel: subscription.OrderbookChannel, Levels: 1000}})
	require.NoError(t, err, "Subscribe must not error")
	subs := k.Websocket.GetSubscriptions()
	require.Len(t, subs, 1, "Should add 1 Subscription")
	require.Equal(t, subscription.SubscribedState, subs[0].State(), "Subscription must be in a subscribed state")

	require.Eventually(t, func() bool {
		b, e2 := k.Websocket.Orderbook.GetOrderbook(spotTestPair, asset.Spot)
		if e2 == nil {
			return !b.LastUpdated.IsZero()
		}
		return false
	}, time.Second*4, time.Millisecond*10, "orderbook must start streaming")

	// Set the state to Unsub so we definitely know Resub worked
	err = subs[0].SetState(subscription.UnsubscribingState)
	require.NoError(t, err)

	err = k.Websocket.ResubscribeToChannel(k.Websocket.Conn, subs[0])
	require.NoError(t, err, "Resubscribe must not error")
	require.Equal(t, subscription.SubscribedState, subs[0].State(), "subscription must be subscribed again")
}

// TestWsOrderbookSub tests orderbook subscriptions for MaxDepth params
func TestWsOrderbookSub(t *testing.T) {
	t.Parallel()

	k := new(Kraken) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes
	require.NoError(t, testexch.Setup(k), "Setup Instance must not error")
	testexch.SetupWs(t, k)

	err := k.Subscribe(subscription.List{{
		Asset:   asset.Spot,
		Channel: subscription.OrderbookChannel,
		Pairs:   currency.Pairs{spotTestPair},
		Levels:  25,
	}})
	require.NoError(t, err, "Simple subscription must not error")

	subs := k.Websocket.GetSubscriptions()
	require.Equal(t, 1, len(subs), "Must have 1 subscription channel")

	err = k.Unsubscribe(subs)
	assert.NoError(t, err, "Unsubscribe should not error")
	assert.Empty(t, k.Websocket.GetSubscriptions(), "Should have successfully removed all channels")

	err = k.Subscribe(subscription.List{{
		Asset:   asset.Spot,
		Channel: subscription.OrderbookChannel,
		Pairs:   currency.Pairs{spotTestPair},
		Levels:  42,
	}})
	assert.ErrorContains(t, err, "Subscription depth not supported", "Bad subscription should error about depth")
}

// TestWsCandlesSub tests candles subscription for Timeframe params
func TestWsCandlesSub(t *testing.T) {
	t.Parallel()

	k := new(Kraken) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes
	require.NoError(t, testexch.Setup(k), "Setup Instance must not error")
	testexch.SetupWs(t, k)

	err := k.Subscribe(subscription.List{{
		Asset:    asset.Spot,
		Channel:  subscription.CandlesChannel,
		Pairs:    currency.Pairs{spotTestPair},
		Interval: kline.OneHour,
	}})
	require.NoError(t, err, "Simple subscription must not error")

	subs := k.Websocket.GetSubscriptions()
	require.Equal(t, 1, len(subs), "Should add 1 Subscription")

	err = k.Unsubscribe(subs)
	assert.NoError(t, err, "Unsubscribe should not error")
	assert.Empty(t, k.Websocket.GetSubscriptions(), "Should have successfully removed all channels")

	err = k.Subscribe(subscription.List{{
		Asset:    asset.Spot,
		Channel:  subscription.CandlesChannel,
		Pairs:    currency.Pairs{spotTestPair},
		Interval: kline.Interval(time.Minute * time.Duration(127)),
	}})
	assert.ErrorContains(t, err, "Subscription ohlc interval not supported", "Bad subscription should error about interval")
}

// TestWsOwnTradesSub tests the authenticated WS subscription channel for trades
func TestWsOwnTradesSub(t *testing.T) {
	t.Parallel()

	sharedtestvalues.SkipTestIfCredentialsUnset(t, k)

	k := new(Kraken) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes
	require.NoError(t, testexch.Setup(k), "Setup Instance must not error")
	testexch.SetupWs(t, k)

	err := k.Subscribe(subscription.List{{Channel: subscription.MyTradesChannel, Authenticated: true}})
	assert.NoError(t, err, "Subsrcibing to ownTrades should not error")

	subs := k.Websocket.GetSubscriptions()
	assert.Len(t, subs, 1, "Should add 1 Subscription")

	err = k.Unsubscribe(subs)
	assert.NoError(t, err, "Unsubscribing an auth channel should not error")
	assert.Empty(t, k.Websocket.GetSubscriptions(), "Should have successfully removed channel")
}

// TestGenerateSubscriptions tests the subscriptions generated from configuration
func TestGenerateSubscriptions(t *testing.T) {
	t.Parallel()

	pairs, err := k.GetEnabledPairs(asset.Spot)
	require.NoError(t, err, "GetEnabledPairs must not error")
	require.False(t, k.Websocket.CanUseAuthenticatedEndpoints(), "Websocket must not be authenticated by default")
	exp := subscription.List{
		{Channel: subscription.TickerChannel},
		{Channel: subscription.AllTradesChannel},
		{Channel: subscription.CandlesChannel, Interval: kline.OneMin},
		{Channel: subscription.OrderbookChannel, Levels: 1000},
	}
	for _, s := range exp {
		s.QualifiedChannel = channelName(s)
		s.Asset = asset.Spot
		s.Pairs = pairs
	}
	subs, err := k.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	testsubs.EqualLists(t, exp, subs)

	k.Websocket.SetCanUseAuthenticatedEndpoints(true)
	exp = append(exp, subscription.List{
		{Channel: subscription.MyOrdersChannel, QualifiedChannel: krakenWsOpenOrders},
		{Channel: subscription.MyTradesChannel, QualifiedChannel: krakenWsOwnTrades},
	}...)
	subs, err = k.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	testsubs.EqualLists(t, exp, subs)
}

func TestGetWSToken(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, k)

	k := new(Kraken) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes
	require.NoError(t, testexch.Setup(k), "Setup Instance must not error")
	testexch.SetupWs(t, k)

	resp, err := k.GetWebsocketToken(t.Context())
	require.NoError(t, err, "GetWebsocketToken must not error")
	assert.NotEmpty(t, resp, "Token should not be empty")
}

// TestWsAddOrder exercises roundtrip of wsAddOrder; See also: mockWsAddOrder
func TestWsAddOrder(t *testing.T) {
	t.Parallel()

	k := testexch.MockWsInstance[Kraken](t, curryWsMockUpgrader(t, mockWsServer)) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes
	require.True(t, k.IsWebsocketAuthenticationSupported(), "WS must be authenticated")
	id, err := k.wsAddOrder(t.Context(), &WsAddOrderRequest{
		OrderType: order.Limit.Lower(),
		OrderSide: order.Buy.Lower(),
		Pair:      "XBT/USD",
		Price:     80000,
	})
	require.NoError(t, err, "wsAddOrder must not error")
	assert.Equal(t, "ONPNXH-KMKMU-F4MR5V", id, "wsAddOrder should return correct order ID")
}

// TestWsCancelOrders exercises roundtrip of wsCancelOrders; See also: mockWsCancelOrders
func TestWsCancelOrders(t *testing.T) {
	t.Parallel()

	k := testexch.MockWsInstance[Kraken](t, curryWsMockUpgrader(t, mockWsServer)) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes
	require.True(t, k.IsWebsocketAuthenticationSupported(), "WS must be authenticated")

	err := k.wsCancelOrders(t.Context(), []string{"RABBIT", "BATFISH", "SQUIRREL", "CATFISH", "MOUSE"})
	assert.ErrorIs(t, err, errCancellingOrder, "Should error cancelling order")
	assert.ErrorContains(t, err, "BATFISH", "Should error containing txn id")
	assert.ErrorContains(t, err, "CATFISH", "Should error containing txn id")
	assert.ErrorContains(t, err, "[EOrder:Unknown order]", "Should error containing server error")

	err = k.wsCancelOrders(t.Context(), []string{"RABBIT", "SQUIRREL", "MOUSE"})
	assert.NoError(t, err, "Should not error with valid ids")
}

func TestWsCancelAllOrders(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, k, canManipulateRealOrders)
	testexch.SetupWs(t, k)
	_, err := k.wsCancelAllOrders(t.Context())
	require.NoError(t, err, "wsCancelAllOrders must not error")
}

func TestWsHandleData(t *testing.T) {
	t.Parallel()
	k := new(Kraken) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes
	require.NoError(t, testexch.Setup(k), "Setup Instance must not error")
	for _, l := range []int{10, 100} {
		err := k.Websocket.AddSuccessfulSubscriptions(k.Websocket.Conn, &subscription.Subscription{
			Channel: subscription.OrderbookChannel,
			Pairs:   currency.Pairs{spotTestPair},
			Asset:   asset.Spot,
			Levels:  l,
		})
		require.NoError(t, err, "AddSuccessfulSubscriptions must not error")
	}
	testexch.FixtureToDataHandler(t, "testdata/wsHandleData.json", k.wsHandleData)
}

func TestWSProcessTrades(t *testing.T) {
	t.Parallel()

	k := new(Kraken) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes
	require.NoError(t, testexch.Setup(k), "Test instance Setup must not error")
	err := k.Websocket.AddSubscriptions(k.Websocket.Conn, &subscription.Subscription{Asset: asset.Spot, Pairs: currency.Pairs{spotTestPair}, Channel: subscription.AllTradesChannel, Key: 18788})
	require.NoError(t, err, "AddSubscriptions must not error")
	testexch.FixtureToDataHandler(t, "testdata/wsAllTrades.json", k.wsHandleData)
	close(k.Websocket.DataHandler)

	invalid := []any{"trades", []any{[]any{"95873.80000", "0.00051182", "1708731380.3791859"}}}
	rawBytes, err := json.Marshal(invalid)
	require.NoError(t, err, "Marshal must not error marshalling invalid trade data")

	pair := currency.NewPair(currency.XBT, currency.USD)
	err = k.wsProcessTrades(json.RawMessage(rawBytes), pair)
	require.ErrorContains(t, err, "error unmarshalling trade data")

	expJSON := []string{
		`{"AssetType":"spot","CurrencyPair":"XBT/USD","Side":"BUY","Price":95873.80000,"Amount":0.00051182,"Timestamp":"2025-02-23T23:29:40.379186Z"}`,
		`{"AssetType":"spot","CurrencyPair":"XBT/USD","Side":"SELL","Price":95940.90000,"Amount":0.00011069,"Timestamp":"2025-02-24T02:01:12.853682Z"}`,
	}
	require.Len(t, k.Websocket.DataHandler, len(expJSON), "Must see correct number of trades")
	for resp := range k.Websocket.DataHandler {
		switch v := resp.(type) {
		case trade.Data:
			i := 1 - len(k.Websocket.DataHandler)
			exp := trade.Data{Exchange: k.Name, CurrencyPair: spotTestPair}
			require.NoErrorf(t, json.Unmarshal([]byte(expJSON[i]), &exp), "Must not error unmarshalling json %d: %s", i, expJSON[i])
			require.Equalf(t, exp, v, "Trade [%d] must be correct", i)
		case error:
			t.Error(v)
		default:
			t.Errorf("Unexpected type in DataHandler: %T (%s)", v, v)
		}
	}
}

func TestWsOpenOrders(t *testing.T) {
	t.Parallel()
	k := new(Kraken) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes
	require.NoError(t, testexch.Setup(k), "Test instance Setup must not error")
	testexch.UpdatePairsOnce(t, k)
	testexch.FixtureToDataHandler(t, "testdata/wsOpenTrades.json", k.wsHandleData)
	close(k.Websocket.DataHandler)
	assert.Len(t, k.Websocket.DataHandler, 7, "Should see 7 orders")
	for resp := range k.Websocket.DataHandler {
		switch v := resp.(type) {
		case *order.Detail:
			switch len(k.Websocket.DataHandler) {
			case 6:
				assert.Equal(t, "OGTT3Y-C6I3P-XRI6HR", v.OrderID, "OrderID")
				assert.Equal(t, order.Limit, v.Type, "order type")
				assert.Equal(t, order.Sell, v.Side, "order side")
				assert.Equal(t, order.Open, v.Status, "order status")
				assert.Equal(t, 34.5, v.Price, "price")
				assert.Equal(t, 10.00345345, v.Amount, "amount")
			case 5:
				assert.Equal(t, "OKB55A-UEMMN-YUXM2A", v.OrderID, "OrderID")
				assert.Equal(t, order.Market, v.Type, "order type")
				assert.Equal(t, order.Buy, v.Side, "order side")
				assert.Equal(t, order.Pending, v.Status, "order status")
				assert.Equal(t, 0.0, v.Price, "price")
				assert.Equal(t, 0.0001, v.Amount, "amount")
				assert.Equal(t, time.UnixMicro(1692851641361371).UTC(), v.Date.UTC(), "Date")
			case 4:
				assert.Equal(t, "OKB55A-UEMMN-YUXM2A", v.OrderID, "OrderID")
				assert.Equal(t, order.Open, v.Status, "order status")
			case 3:
				assert.Equal(t, "OKB55A-UEMMN-YUXM2A", v.OrderID, "OrderID")
				assert.Equal(t, order.UnknownStatus, v.Status, "order status")
				assert.Equal(t, 26425.2, v.AverageExecutedPrice, "AverageExecutedPrice")
				assert.Equal(t, 0.0001, v.ExecutedAmount, "ExecutedAmount")
				assert.Equal(t, 0.0, v.RemainingAmount, "RemainingAmount") // Not in the message; Testing regression to bad derivation
				assert.Equal(t, 0.00687, v.Fee, "Fee")
			case 2:
				assert.Equal(t, "OKB55A-UEMMN-YUXM2A", v.OrderID, "OrderID")
				assert.Equal(t, order.Closed, v.Status, "order status")
				assert.Equal(t, 0.0001, v.ExecutedAmount, "ExecutedAmount")
				assert.Equal(t, 26425.2, v.AverageExecutedPrice, "AverageExecutedPrice")
				assert.Equal(t, 0.00687, v.Fee, "Fee")
				assert.Equal(t, time.UnixMicro(1692851641361447).UTC(), v.LastUpdated.UTC(), "LastUpdated")
			case 1:
				assert.Equal(t, "OGTT3Y-C6I3P-XRI6HR", v.OrderID, "OrderID")
				assert.Equal(t, order.UnknownStatus, v.Status, "order status")
				assert.Equal(t, 10.00345345, v.ExecutedAmount, "ExecutedAmount")
				assert.Equal(t, 0.001, v.Fee, "Fee")
				assert.Equal(t, 34.5, v.AverageExecutedPrice, "AverageExecutedPrice")
			case 0:
				assert.Equal(t, "OGTT3Y-C6I3P-XRI6HR", v.OrderID, "OrderID")
				assert.Equal(t, order.Closed, v.Status, "order status")
				assert.Equal(t, time.UnixMicro(1692675961789052).UTC(), v.LastUpdated.UTC(), "LastUpdated")
				assert.Equal(t, 10.00345345, v.ExecutedAmount, "ExecutedAmount")
				assert.Equal(t, 0.001, v.Fee, "Fee")
				assert.Equal(t, 34.5, v.AverageExecutedPrice, "AverageExecutedPrice")
			}
		case error:
			t.Error(v)
		default:
			t.Errorf("Unexpected type in DataHandler: %T (%s)", v, v)
		}
	}
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, k)

	_, err := k.GetHistoricCandles(t.Context(), spotTestPair, asset.Spot, kline.OneHour, time.Now().Add(-time.Hour*12), time.Now())
	assert.NoError(t, err, "GetHistoricCandles should not error")

	_, err = k.GetHistoricCandles(t.Context(), futuresTestPair, asset.Futures, kline.OneHour, time.Now().Add(-time.Hour*12), time.Now())
	assert.ErrorIs(t, err, asset.ErrNotSupported, "GetHistoricCandles should error with asset.ErrNotSupported")
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	_, err := k.GetHistoricCandlesExtended(t.Context(), futuresTestPair, asset.Spot, kline.OneMin, time.Now().Add(-time.Minute*3), time.Now())
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "GetHistoricCandlesExtended should error correctly")
}

func TestFormatExchangeKlineInterval(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		interval kline.Interval
		exp      string
	}{
		{kline.OneMin, "1"},
		{kline.OneDay, "1440"},
	} {
		assert.Equalf(t, tt.exp, k.FormatExchangeKlineInterval(tt.interval), "FormatExchangeKlineInterval should return correct output for %s", tt.interval.Short())
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, k)

	_, err := k.GetRecentTrades(t.Context(), spotTestPair, asset.Spot)
	assert.NoError(t, err, "GetRecentTrades should not error")

	_, err = k.GetRecentTrades(t.Context(), futuresTestPair, asset.Futures)
	assert.NoError(t, err, "GetRecentTrades should not error")
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	_, err := k.GetHistoricTrades(t.Context(), spotTestPair, asset.Spot, time.Now().Add(-time.Minute*15), time.Now())
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "GetHistoricTrades should error")
}

var testOb = orderbook.Book{
	Asks: []orderbook.Level{
		// NOTE: 0.00000500 float64 == 0.000005
		{Price: 0.05005, StrPrice: "0.05005", Amount: 0.00000500, StrAmount: "0.00000500"},
		{Price: 0.05010, StrPrice: "0.05010", Amount: 0.00000500, StrAmount: "0.00000500"},
		{Price: 0.05015, StrPrice: "0.05015", Amount: 0.00000500, StrAmount: "0.00000500"},
		{Price: 0.05020, StrPrice: "0.05020", Amount: 0.00000500, StrAmount: "0.00000500"},
		{Price: 0.05025, StrPrice: "0.05025", Amount: 0.00000500, StrAmount: "0.00000500"},
		{Price: 0.05030, StrPrice: "0.05030", Amount: 0.00000500, StrAmount: "0.00000500"},
		{Price: 0.05035, StrPrice: "0.05035", Amount: 0.00000500, StrAmount: "0.00000500"},
		{Price: 0.05040, StrPrice: "0.05040", Amount: 0.00000500, StrAmount: "0.00000500"},
		{Price: 0.05045, StrPrice: "0.05045", Amount: 0.00000500, StrAmount: "0.00000500"},
		{Price: 0.05050, StrPrice: "0.05050", Amount: 0.00000500, StrAmount: "0.00000500"},
	},
	Bids: []orderbook.Level{
		{Price: 0.05000, StrPrice: "0.05000", Amount: 0.00000500, StrAmount: "0.00000500"},
		{Price: 0.04995, StrPrice: "0.04995", Amount: 0.00000500, StrAmount: "0.00000500"},
		{Price: 0.04990, StrPrice: "0.04990", Amount: 0.00000500, StrAmount: "0.00000500"},
		{Price: 0.04980, StrPrice: "0.04980", Amount: 0.00000500, StrAmount: "0.00000500"},
		{Price: 0.04975, StrPrice: "0.04975", Amount: 0.00000500, StrAmount: "0.00000500"},
		{Price: 0.04970, StrPrice: "0.04970", Amount: 0.00000500, StrAmount: "0.00000500"},
		{Price: 0.04965, StrPrice: "0.04965", Amount: 0.00000500, StrAmount: "0.00000500"},
		{Price: 0.04960, StrPrice: "0.04960", Amount: 0.00000500, StrAmount: "0.00000500"},
		{Price: 0.04955, StrPrice: "0.04955", Amount: 0.00000500, StrAmount: "0.00000500"},
		{Price: 0.04950, StrPrice: "0.04950", Amount: 0.00000500, StrAmount: "0.00000500"},
	},
}

const krakenAPIDocChecksum = 974947235

func TestChecksumCalculation(t *testing.T) {
	t.Parallel()
	expected := "5005"
	if v := trim("0.05005"); v != expected {
		t.Errorf("expected %s but received %s", expected, v)
	}

	expected = "500"
	if v := trim("0.00000500"); v != expected {
		t.Errorf("expected %s but received %s", expected, v)
	}

	err := validateCRC32(&testOb, krakenAPIDocChecksum)
	if err != nil {
		t.Error(err)
	}
}

func TestGetCharts(t *testing.T) {
	t.Parallel()
	resp, err := k.GetFuturesCharts(t.Context(), "1d", "spot", futuresTestPair, time.Time{}, time.Time{})
	require.NoError(t, err)
	require.NotEmpty(t, resp.Candles)

	end := resp.Candles[0].Time.Time()
	_, err = k.GetFuturesCharts(t.Context(), "1d", "spot", futuresTestPair, end.Add(-time.Hour*24*7), end)
	require.NoError(t, err)
}

func TestGetFuturesTrades(t *testing.T) {
	t.Parallel()
	_, err := k.GetFuturesTrades(t.Context(), futuresTestPair, time.Time{}, time.Time{})
	assert.NoError(t, err, "GetFuturesTrades should not error")

	_, err = k.GetFuturesTrades(t.Context(), futuresTestPair, time.Now().Add(-time.Hour), time.Now())
	assert.NoError(t, err, "GetFuturesTrades should not error")
}

var websocketXDGUSDOrderbookUpdates = []string{
	`[2304,{"as":[["0.074602700","278.39626342","1690246067.832139"],["0.074611000","555.65134028","1690246086.243668"],["0.074613300","524.87121572","1690245901.574881"],["0.074624600","77.57180740","1690246060.668500"],["0.074632500","620.64648404","1690246010.904883"],["0.074698400","409.57419037","1690246041.269821"],["0.074700000","61067.71115772","1690246089.485595"],["0.074723200","4394.01869240","1690246087.557913"],["0.074725200","4229.57885125","1690246082.911452"],["0.074738400","212.25501214","1690246089.421559"]],"bs":[["0.074597400","53591.43163675","1690246089.451762"],["0.074596700","33594.18269213","1690246089.514152"],["0.074596600","53598.60351469","1690246089.340781"],["0.074594800","5358.57247081","1690246089.347962"],["0.074594200","30168.21074680","1690246089.345112"],["0.074590900","7089.69894583","1690246088.212880"],["0.074586700","46925.20182082","1690246089.074618"],["0.074577200","5500.00000000","1690246087.568856"],["0.074569600","8132.49888631","1690246086.841219"],["0.074562900","8413.11098009","1690246087.024863"]]},"book-10","XDG/USD"]`,
	`[2304,{"a":[["0.074700000","0.00000000","1690246089.516119"],["0.074738500","125000.00000000","1690246063.352141","r"]],"c":"2219685759"},"book-10","XDG/USD"]`,
	`[2304,{"a":[["0.074678800","33476.70673703","1690246089.570183"]],"c":"1897176819"},"book-10","XDG/USD"]`,
	`[2304,{"b":[["0.074562900","0.00000000","1690246089.570206"],["0.074559600","4000.00000000","1690246086.478591","r"]],"c":"2498018751"},"book-10","XDG/USD"]`,
	`[2304,{"b":[["0.074577300","125000.00000000","1690246089.577140"]],"c":"155006629"},"book-10","XDG/USD"]`,
	`[2304,{"a":[["0.074678800","0.00000000","1690246089.584498"],["0.074738500","125000.00000000","1690246063.352141","r"]],"c":"3703147735"},"book-10","XDG/USD"]`,
	`[2304,{"b":[["0.074597500","10000.00000000","1690246089.602477"]],"c":"2989534775"},"book-10","XDG/USD"]`,
	`[2304,{"a":[["0.074738500","0.00000000","1690246089.608769"],["0.074750800","51369.02100000","1690246089.495500","r"]],"c":"1842075082"},"book-10","XDG/USD"]`,
	`[2304,{"b":[["0.074583500","8413.11098009","1690246089.612144"]],"c":"710274752"},"book-10","XDG/USD"]`,
	`[2304,{"b":[["0.074578500","9966.55841398","1690246089.634739"]],"c":"1646135532"},"book-10","XDG/USD"]`,
	`[2304,{"a":[["0.074738400","0.00000000","1690246089.638648"],["0.074751500","80499.09450000","1690246086.679402","r"]],"c":"2509689626"},"book-10","XDG/USD"]`,
	`[2304,{"a":[["0.074750700","290.96851266","1690246089.638754"]],"c":"3981738175"},"book-10","XDG/USD"]`,
	`[2304,{"a":[["0.074720000","61067.71115772","1690246089.662102"]],"c":"1591820326"},"book-10","XDG/USD"]`,
	`[2304,{"a":[["0.074602700","0.00000000","1690246089.670911"],["0.074750800","51369.02100000","1690246089.495500","r"]],"c":"3838272404"},"book-10","XDG/USD"]`,
	`[2304,{"a":[["0.074611000","0.00000000","1690246089.680343"],["0.074758500","159144.39750000","1690246035.158327","r"]],"c":"4241552383"},"book-10","XDG/USD"]	`,
}

var websocketLUNAEUROrderbookUpdates = []string{
	`[9536,{"as":[["0.000074650000","147354.32016076","1690249755.076929"],["0.000074710000","5084881.40000000","1690250711.359411"],["0.000074760000","9700502.70476704","1690250743.279490"],["0.000074990000","2933380.23886300","1690249596.627969"],["0.000075000000","433333.33333333","1690245575.626780"],["0.000075020000","152914.84493416","1690243661.232520"],["0.000075070000","146529.90542161","1690249048.358424"],["0.000075250000","737072.85720004","1690211553.549248"],["0.000075400000","670061.64567140","1690250769.261196"],["0.000075460000","980226.63603417","1690250769.627523"]],"bs":[["0.000074590000","71029.87806720","1690250763.012724"],["0.000074580000","15935576.86404000","1690250763.012710"],["0.000074520000","33758611.79634000","1690250718.290955"],["0.000074350000","3156650.58590277","1690250766.499648"],["0.000074340000","301727260.79999999","1690250766.490238"],["0.000074320000","64611496.53837000","1690250742.680258"],["0.000074310000","104228596.60000000","1690250744.679121"],["0.000074300000","40366046.10582000","1690250762.685914"],["0.000074200000","3690216.57320475","1690250645.311465"],["0.000074060000","1337170.52532521","1690250742.012527"]]},"book-10","LUNA/EUR"]`,
	`[9536,{"b":[["0.000074060000","0.00000000","1690250770.616604"],["0.000074050000","16742421.17790510","1690250710.867730","r"]],"c":"418307145"},"book-10","LUNA/EUR"]`,
}

var websocketGSTEUROrderbookUpdates = []string{
	`[8912,{"as":[["0.01300","850.00000000","1690230914.230506"],["0.01400","323483.99590510","1690256356.615823"],["0.01500","100287.34442717","1690219133.193345"],["0.01600","67995.78441017","1690118389.451216"],["0.01700","41776.38397740","1689676303.381189"],["0.01800","11785.76177777","1688631951.812452"],["0.01900","23700.00000000","1686935422.319042"],["0.02000","3941.17000000","1689415829.176481"],["0.02100","16598.69173066","1689420942.541943"],["0.02200","17572.51572836","1689851425.907427"]],"bs":[["0.01200","14220.66466572","1690256540.842831"],["0.01100","160223.61546438","1690256401.072463"],["0.01000","63083.48958963","1690256604.037673"],["0.00900","6750.00000000","1690252470.633938"],["0.00800","213059.49706376","1690256360.386301"],["0.00700","1000.00000000","1689869458.464975"],["0.00600","4000.00000000","1690221333.528698"],["0.00100","245000.00000000","1690051368.753455"]]},"book-10","GST/EUR"]`,
	`[8912,{"b":[["0.01000","60583.48958963","1690256620.206768"],["0.01000","63083.48958963","1690256620.206783"]],"c":"69619317"},"book-10","GST/EUR"]`,
}

func TestWsOrderbookMax10Depth(t *testing.T) {
	t.Parallel()
	k := new(Kraken) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes
	require.NoError(t, testexch.Setup(k), "Setup Instance must not error")
	pairs := currency.Pairs{
		currency.NewPairWithDelimiter("XDG", "USD", "/"),
		currency.NewPairWithDelimiter("LUNA", "EUR", "/"),
		currency.NewPairWithDelimiter("GST", "EUR", "/"),
	}
	for _, p := range pairs {
		err := k.Websocket.AddSuccessfulSubscriptions(k.Websocket.Conn, &subscription.Subscription{
			Channel: subscription.OrderbookChannel,
			Pairs:   currency.Pairs{p},
			Asset:   asset.Spot,
			Levels:  10,
		})
		require.NoError(t, err, "AddSuccessfulSubscriptions must not error")
	}

	for x := range websocketXDGUSDOrderbookUpdates {
		err := k.wsHandleData(t.Context(), []byte(websocketXDGUSDOrderbookUpdates[x]))
		require.NoError(t, err, "wsHandleData must not error")
	}

	for x := range websocketLUNAEUROrderbookUpdates {
		err := k.wsHandleData(t.Context(), []byte(websocketLUNAEUROrderbookUpdates[x]))
		// TODO: Known issue with LUNA pairs and big number float precision
		// storage and checksum calc. Might need to store raw strings as fields
		// in the orderbook.Level struct.
		// Required checksum: 7465000014735432016076747100005084881400000007476000097005027047670474990000293338023886300750000004333333333333375020000152914844934167507000014652990542161752500007370728572000475400000670061645671407546000098022663603417745900007102987806720745800001593557686404000745200003375861179634000743500003156650585902777434000030172726079999999743200006461149653837000743100001042285966000000074300000403660461058200074200000369021657320475740500001674242117790510
		if x != len(websocketLUNAEUROrderbookUpdates)-1 {
			require.NoError(t, err, "wsHandleData must not error")
		}
	}

	// This has less than 10 bids and still needs a checksum calc.
	for x := range websocketGSTEUROrderbookUpdates {
		err := k.wsHandleData(t.Context(), []byte(websocketGSTEUROrderbookUpdates[x]))
		require.NoError(t, err, "wsHandleData must not error")
	}
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	_, err := k.GetFuturesContractDetails(t.Context(), asset.Spot)
	assert.ErrorIs(t, err, futures.ErrNotFuturesAsset)

	_, err = k.GetFuturesContractDetails(t.Context(), asset.USDTMarginedFutures)
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = k.GetFuturesContractDetails(t.Context(), asset.Futures)
	assert.NoError(t, err, "GetFuturesContractDetails should not error")
}

func TestGetLatestFundingRates(t *testing.T) {
	t.Parallel()
	_, err := k.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset:                asset.USDTMarginedFutures,
		Pair:                 currency.NewBTCUSD(),
		IncludePredictedRate: true,
	})
	assert.ErrorIs(t, err, asset.ErrNotSupported, "GetLatestFundingRates should error")

	_, err = k.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset: asset.Futures,
	})
	assert.NoError(t, err, "GetLatestFundingRates should not error")

	err = k.CurrencyPairs.EnablePair(asset.Futures, futuresTestPair)
	assert.Truef(t, err == nil || errors.Is(err, currency.ErrPairAlreadyEnabled), "EnablePair should not error: %s", err)
	_, err = k.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset:                asset.Futures,
		Pair:                 futuresTestPair,
		IncludePredictedRate: true,
	})
	assert.NoError(t, err, "GetLatestFundingRates should not error")
}

func TestIsPerpetualFutureCurrency(t *testing.T) {
	t.Parallel()
	is, err := k.IsPerpetualFutureCurrency(asset.Binary, currency.NewBTCUSDT())
	assert.NoError(t, err)
	assert.False(t, is, "IsPerpetualFutureCurrency should return false for a binary asset")

	is, err = k.IsPerpetualFutureCurrency(asset.Futures, currency.NewBTCUSDT())
	assert.NoError(t, err)
	assert.False(t, is, "IsPerpetualFutureCurrency should return false for a non-perpetual future")

	is, err = k.IsPerpetualFutureCurrency(asset.Futures, futuresTestPair)
	assert.NoError(t, err)
	assert.True(t, is, "IsPerpetualFutureCurrency should return true for a perpetual future")
}

func TestGetOpenInterest(t *testing.T) {
	t.Parallel()
	k := new(Kraken) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes
	require.NoError(t, testexch.Setup(k), "Test instance Setup must not error")

	_, err := k.GetOpenInterest(t.Context(), key.PairAsset{
		Base:  currency.ETH.Item,
		Quote: currency.USDT.Item,
		Asset: asset.USDTMarginedFutures,
	})
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	cp1 := currency.NewPair(currency.PF, currency.NewCode("XBTUSD"))
	cp2 := currency.NewPair(currency.PF, currency.NewCode("ETHUSD"))
	sharedtestvalues.SetupCurrencyPairsForExchangeAsset(t, k, asset.Futures, cp1, cp2)

	resp, err := k.GetOpenInterest(t.Context(), key.PairAsset{
		Base:  cp1.Base.Item,
		Quote: cp1.Quote.Item,
		Asset: asset.Futures,
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)

	resp, err = k.GetOpenInterest(t.Context(),
		key.PairAsset{
			Base:  cp1.Base.Item,
			Quote: cp1.Quote.Item,
			Asset: asset.Futures,
		},
		key.PairAsset{
			Base:  cp2.Base.Item,
			Quote: cp2.Quote.Item,
			Asset: asset.Futures,
		})
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)

	resp, err = k.GetOpenInterest(t.Context())
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

// curryWsMockUpgrader handles Kraken specific http auth token responses prior to handling off to standard Websocket upgrader
func curryWsMockUpgrader(tb testing.TB, h mockws.WsMockFunc) http.HandlerFunc {
	tb.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "GetWebSocketsToken") {
			_, err := w.Write([]byte(`{"result":{"token":"mockAuth"}}`))
			assert.NoError(tb, err, "Write should not error")
			return
		}
		mockws.WsMockUpgrader(tb, w, r, h)
	}
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, k)
	for _, a := range k.GetAssetTypes(false) {
		pairs, err := k.CurrencyPairs.GetPairs(a, false)
		if len(pairs) == 0 {
			continue
		}
		require.NoErrorf(t, err, "cannot get pairs for %s", a)
		resp, err := k.GetCurrencyTradeURL(t.Context(), a, pairs[0])
		if a != asset.Spot && a != asset.Futures {
			assert.ErrorIs(t, err, asset.ErrNotSupported)
			continue
		}
		require.NoError(t, err)
		assert.NotEmpty(t, resp)
	}
}

func TestErrorResponse(t *testing.T) {
	var g genericRESTResponse

	tests := []struct {
		name          string
		jsonStr       string
		expectError   bool
		errorMsg      string
		warningMsg    string
		requiresReset bool
	}{
		{
			name:    "No errors or warnings",
			jsonStr: `{"error":[],"result":{"unixtime":1721884425,"rfc1123":"Thu, 25 Jul 24 05:13:45 +0000"}}`,
		},
		{
			name:        "Invalid error type int",
			jsonStr:     `{"error":[69420],"result":{}}`,
			expectError: true,
			errorMsg:    "unable to convert 69420 to string",
		},
		{
			name:        "Unhandled error type float64",
			jsonStr:     `{"error":124,"result":{}}`,
			expectError: true,
			errorMsg:    "unhandled error response type float64",
		},
		{
			name:     "Known error string",
			jsonStr:  `{"error":["EQuery:Unknown asset pair"],"result":{}}`,
			errorMsg: "EQuery:Unknown asset pair",
		},
		{
			name:     "Known error string (single)",
			jsonStr:  `{"error":"EService:Unavailable","result":{}}`,
			errorMsg: "EService:Unavailable",
		},
		{
			name:          "Warning string in array",
			jsonStr:       `{"error":["WGeneral:Danger"],"result":{}}`,
			warningMsg:    "WGeneral:Danger",
			requiresReset: true,
		},
		{
			name:          "Warning string",
			jsonStr:       `{"error":"WGeneral:Unknown warning","result":{}}`,
			warningMsg:    "WGeneral:Unknown warning",
			requiresReset: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.requiresReset {
				g = genericRESTResponse{}
			}
			err := json.Unmarshal([]byte(tt.jsonStr), &g)
			if tt.expectError {
				require.ErrorContains(t, err, tt.errorMsg, "Unmarshal must error")
			} else {
				require.NoError(t, err)
				if tt.errorMsg != "" {
					assert.ErrorContainsf(t, g.Error.Errors(), tt.errorMsg, "Errors should contain %s", tt.errorMsg)
				} else {
					assert.NoError(t, g.Error.Errors(), "Errors should not error")
				}
				if tt.warningMsg != "" {
					assert.Containsf(t, g.Error.Warnings(), tt.warningMsg, "Warnings should contain %s", tt.warningMsg)
				} else {
					assert.Empty(t, g.Error.Warnings(), "Warnings should be empty")
				}
			}
		})
	}
}

func TestGetFuturesErr(t *testing.T) {
	t.Parallel()

	assert.ErrorContains(t, getFuturesErr(json.RawMessage(`unparsable rubbish`)), "invalid char", "Bad JSON should error correctly")
	assert.NoError(t, getFuturesErr(json.RawMessage(`{"candles":[]}`)), "JSON with no Result should not error")
	assert.NoError(t, getFuturesErr(json.RawMessage(`{"Result":"4 goats"}`)), "JSON with non-error Result should not error")
	assert.ErrorIs(t, getFuturesErr(json.RawMessage(`{"Result":"error"}`)), common.ErrUnknownError, "JSON with error Result should error correctly")
	assert.ErrorContains(t, getFuturesErr(json.RawMessage(`{"Result":"error", "error": "1 goat"}`)), "1 goat", "JSON with an error should error correctly")
	err := getFuturesErr(json.RawMessage(`{"Result":"error", "errors": ["2 goats", "3 goats"]}`))
	assert.ErrorContains(t, err, "2 goat", "JSON with errors should error correctly")
	assert.ErrorContains(t, err, "3 goat", "JSON with errors should error correctly")
	err = getFuturesErr(json.RawMessage(`{"Result":"error", "error": "too many goats", "errors": ["2 goats", "3 goats"]}`))
	assert.ErrorContains(t, err, "2 goat", "JSON with both error and errors should error correctly")
	assert.ErrorContains(t, err, "3 goat", "JSON with both error and errors should error correctly")
	assert.ErrorContains(t, err, "too many goat", "JSON both error and with errors should error correctly")
}

func TestEnforceStandardChannelNames(t *testing.T) {
	for _, n := range []string{
		krakenWsSpread, krakenWsTicker, subscription.TickerChannel, subscription.OrderbookChannel, subscription.CandlesChannel,
		subscription.AllTradesChannel, subscription.MyTradesChannel, subscription.MyOrdersChannel,
	} {
		assert.NoError(t, enforceStandardChannelNames(&subscription.Subscription{Channel: n}), "Standard channel names and bespoke names should not error")
	}
	for _, n := range []string{krakenWsOrderbook, krakenWsOHLC, krakenWsTrade, krakenWsOwnTrades, krakenWsOpenOrders, krakenWsOrderbook + "-5"} {
		err := enforceStandardChannelNames(&subscription.Subscription{Channel: n})
		assert.ErrorIsf(t, err, subscription.ErrUseConstChannelName, "Private channel names should not be allowed for %s", n)
	}
}

func TestWebsocketAuthToken(t *testing.T) {
	t.Parallel()
	k := new(Kraken)
	k.setWebsocketAuthToken("meep")
	const n = 69
	var wg sync.WaitGroup
	wg.Add(2 * n)

	start := make(chan struct{})
	for range n {
		go func() {
			defer wg.Done()
			<-start
			k.setWebsocketAuthToken("69420")
		}()
	}
	for range n {
		go func() {
			defer wg.Done()
			<-start
			k.websocketAuthToken()
		}()
	}
	close(start)
	wg.Wait()
	assert.Equal(t, "69420", k.websocketAuthToken(), "websocketAuthToken should return correctly after concurrent reads and writes")
}

func TestSetWebsocketAuthToken(t *testing.T) {
	t.Parallel()
	k := new(Kraken)
	k.setWebsocketAuthToken("69420")
	assert.Equal(t, "69420", k.websocketAuthToken())
}
