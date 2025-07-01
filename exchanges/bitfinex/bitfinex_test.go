package bitfinex

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/buger/jsonparser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Please supply API keys here or in config/testdata.json to test authenticated endpoints
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var (
	b          *Bitfinex
	btcusdPair = currency.NewBTCUSD()
)

func TestMain(m *testing.M) {
	b = new(Bitfinex)
	if err := testexch.Setup(b); err != nil {
		log.Fatalf("Bitfinex Setup error: %s", err)
	}

	if apiKey != "" && apiSecret != "" {
		b.Websocket.SetCanUseAuthenticatedEndpoints(true)
		b.API.AuthenticatedSupport = true
		b.API.AuthenticatedWebsocketSupport = true
		b.SetCredentials(apiKey, apiSecret, "", "", "", "")
	}

	os.Exit(m.Run())
}

func TestGetV2MarginFunding(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetV2MarginFunding(t.Context(), "fUSD", "2", 2)
	if err != nil {
		t.Error(err)
	}
}

func TestGetV2MarginInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetV2MarginInfo(t.Context(), "base")
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetV2MarginInfo(t.Context(), "tBTCUSD")
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetV2MarginInfo(t.Context(), "sym_all")
	if err != nil {
		t.Error(err)
	}
}

func TestGetAccountInfoV2(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetAccountInfoV2(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestGetV2FundingInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetV2FundingInfo(t.Context(), "fUST")
	if err != nil {
		t.Error(err)
	}
}

func TestGetV2Balances(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetV2Balances(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestGetDerivativeStatusInfo(t *testing.T) {
	t.Parallel()
	_, err := b.GetDerivativeStatusInfo(t.Context(), "ALL", "", "", 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetPairs(t *testing.T) {
	t.Parallel()

	_, err := b.GetPairs(t.Context(), asset.Binary)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	assets := b.GetAssetTypes(false)
	for x := range assets {
		_, err := b.GetPairs(t.Context(), assets[x])
		if err != nil {
			t.Error(err)
		}
	}
}

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, b)
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	tests := map[asset.Item][]currency.Pair{
		asset.Spot: {
			currency.NewPair(currency.ETH, currency.UST),
			currency.NewPair(currency.BTC, currency.UST),
		},
	}
	for assetItem, pairs := range tests {
		if err := b.UpdateOrderExecutionLimits(t.Context(), assetItem); err != nil {
			t.Errorf("Error fetching %s pairs for test: %v", assetItem, err)
			continue
		}
		for _, pair := range pairs {
			limits, err := b.GetOrderExecutionLimits(assetItem, pair)
			if err != nil {
				t.Errorf("GetOrderExecutionLimits() error during TestExecutionLimits; Asset: %s Pair: %s Err: %v", assetItem, pair, err)
				continue
			}
			if limits.MinimumBaseAmount == 0 {
				t.Errorf("UpdateOrderExecutionLimits empty minimum base amount; Pair: %s Expected Limit: %v", pair, limits.MinimumBaseAmount)
			}
		}
	}
}

func TestAppendOptionalDelimiter(t *testing.T) {
	t.Parallel()
	curr1, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	b.appendOptionalDelimiter(&curr1)
	if curr1.Delimiter != "" {
		t.Errorf("Expected no delimiter, received %v", curr1.Delimiter)
	}
	curr2, err := currency.NewPairFromString("DUSK:USD")
	if err != nil {
		t.Fatal(err)
	}

	curr2.Delimiter = ""
	b.appendOptionalDelimiter(&curr2)
	if curr2.Delimiter != ":" {
		t.Errorf("Expected \":\" as a delimiter, received %v", curr2.Delimiter)
	}
}

func TestGetPlatformStatus(t *testing.T) {
	t.Parallel()
	result, err := b.GetPlatformStatus(t.Context())
	if err != nil {
		t.Errorf("TestGetPlatformStatus error: %s", err)
	}

	if result != bitfinexOperativeMode && result != bitfinexMaintenanceMode {
		t.Errorf("TestGetPlatformStatus unexpected response code")
	}
}

func TestGetTickerBatch(t *testing.T) {
	t.Parallel()
	ticks, err := b.GetTickerBatch(t.Context())
	require.NoError(t, err, "GetTickerBatch must not error")
	require.NotEmpty(t, ticks, "GetTickerBatch must return some ticks")
	require.Contains(t, ticks, "tBTCUSD", "Ticker batch must contain tBTCUSD")
	checkTradeTick(t, ticks["tBTCUSD"])
	require.Contains(t, ticks, "fUSD", "Ticker batch must contain fUSD")
	checkTradeTick(t, ticks["fUSD"])
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	tick, err := b.GetTicker(t.Context(), "tBTCUSD")
	require.NoError(t, err, "GetTicker must not error")
	checkTradeTick(t, tick)
}

func TestTickerFromResp(t *testing.T) {
	t.Parallel()
	_, err := tickerFromResp("tBTCUSD", []any{100.0, nil, 100.0, nil, nil, nil, nil, nil, nil, nil})
	assert.ErrorIs(t, err, errTickerInvalidResp, "tickerFromResp should error correctly")
	assert.ErrorContains(t, err, "BidSize", "tickerFromResp should error correctly")
	assert.ErrorContains(t, err, "tBTCUSD", "tickerFromResp should error correctly")

	_, err = tickerFromResp("tBTCUSD", []any{100.0, nil, 100.0, nil, nil, nil, nil, nil, nil})
	assert.ErrorIs(t, err, errTickerInvalidFieldCount, "tickerFromResp should error correctly")
	assert.ErrorContains(t, err, "tBTCUSD", "tickerFromResp should error correctly")

	tick, err := tickerFromResp("tBTCUSD", []any{1.1, 2.2, 3.3, 4.4, 5.5, 6.6, 7.7, 8.8, 9.9, 10.10})
	require.NoError(t, err, "tickerFromResp must error correctly")
	assert.Equal(t, 1.1, tick.Bid, "Tick Bid should be correct")
	assert.Equal(t, 2.2, tick.BidSize, "Tick BidSize should be correct")
	assert.Equal(t, 3.3, tick.Ask, "Tick Ask should be correct")
	assert.Equal(t, 4.4, tick.AskSize, "Tick AskSize should be correct")
	assert.Equal(t, 5.5, tick.DailyChange, "Tick DailyChange should be correct")
	assert.Equal(t, 6.6, tick.DailyChangePerc, "Tick DailyChangePerc should be correct")
	assert.Equal(t, 7.7, tick.Last, "Tick Last should be correct")
	assert.Equal(t, 8.8, tick.Volume, "Tick Volume should be correct")
	assert.Equal(t, 9.9, tick.High, "Tick High should be correct")
	assert.Equal(t, 10.10, tick.Low, "Tick Low should be correct")

	_, err = tickerFromResp("fBTC", []any{100.0, nil, 100.0, nil, nil, nil, nil, nil, nil, nil})
	assert.ErrorIs(t, err, errTickerInvalidFieldCount, "tickerFromResp should delegate to tickerFromFundingResp and error correctly")
	assert.ErrorContains(t, err, "fBTC", "tickerFromResp should delegate to tickerFromFundingResp and error correctly")
}

func TestTickerFromFundingResp(t *testing.T) {
	t.Parallel()
	_, err := tickerFromFundingResp("fBTC", []any{nil, 100.0, nil, 100.0, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil})
	assert.ErrorIs(t, err, errTickerInvalidResp, "tickerFromFundingResp should error correctly")
	assert.ErrorContains(t, err, "FlashReturnRate", "tickerFromFundingResp should error correctly")
	assert.ErrorContains(t, err, "fBTC", "tickerFromFundingResp should error correctly")

	_, err = tickerFromFundingResp("fBTC", []any{100.0, nil, 100.0, nil, nil, nil, nil, nil, nil})
	assert.ErrorIs(t, err, errTickerInvalidFieldCount, "tickerFromFundingResp should error correctly")
	assert.ErrorContains(t, err, "fBTC", "tickerFromFundingResp should error correctly")

	tick, err := tickerFromFundingResp("fBTC", []any{1.1, 2.2, 3.0, 4.4, 5.5, 6.0, 7.7, 8.8, 9.9, 10.10, 11.11, 12.12, 13.13, nil, nil, 15.15})
	require.NoError(t, err, "tickerFromFundingResp must error correctly")
	assert.Equal(t, 1.1, tick.FlashReturnRate, "Tick FlashReturnRate should be correct")
	assert.Equal(t, 2.2, tick.Bid, "Tick Bid should be correct")
	assert.Equal(t, int64(3), tick.BidPeriod, "Tick BidPeriod should be correct")
	assert.Equal(t, 4.4, tick.BidSize, "Tick BidSize should be correct")
	assert.Equal(t, 5.5, tick.Ask, "Tick Ask should be correct")
	assert.Equal(t, int64(6), tick.AskPeriod, "Tick AskPeriod should be correct")
	assert.Equal(t, 7.7, tick.AskSize, "Tick AskSize should be correct")
	assert.Equal(t, 8.8, tick.DailyChange, "Tick DailyChange should be correct")
	assert.Equal(t, 9.9, tick.DailyChangePerc, "Tick DailyChangePerc should be correct")
	assert.Equal(t, 10.10, tick.Last, "Tick Last should be correct")
	assert.Equal(t, 11.11, tick.Volume, "Tick Volume should be correct")
	assert.Equal(t, 12.12, tick.High, "Tick High should be correct")
	assert.Equal(t, 13.13, tick.Low, "Tick Low should be correct")
	assert.Equal(t, 15.15, tick.FRRAmountAvailable, "Tick FRRAmountAvailable should be correct")
}

func TestGetTickerFunding(t *testing.T) {
	t.Parallel()
	tick, err := b.GetTicker(t.Context(), "fUSD")
	require.NoError(t, err, "GetTicker must not error")
	checkFundingTick(t, tick)
}

func checkTradeTick(tb testing.TB, tick *Ticker) {
	tb.Helper()
	assert.Positive(tb, tick.Bid, "Tick Bid should be positive")
	assert.Positive(tb, tick.BidSize, "Tick BidSize should be positive")
	assert.Positive(tb, tick.Ask, "Tick Ask should be positive")
	assert.Positive(tb, tick.AskSize, "Tick AskSize should be positive")
	assert.Positive(tb, tick.Last, "Tick Last should be positive")
	// Can't test DailyChange*, Volume, High or Low without false positives when they're occasionally 0
}

func checkFundingTick(tb testing.TB, tick *Ticker) {
	tb.Helper()
	assert.NotZero(tb, tick.FlashReturnRate, "Tick FlashReturnRate should not be zero")
	assert.Positive(tb, tick.Bid, "Tick Bid should be positive")
	assert.Positive(tb, tick.BidPeriod, "Tick BidPeriod should be positive")
	assert.Positive(tb, tick.BidSize, "Tick BidSize should be positive")
	assert.Positive(tb, tick.Ask, "Tick Ask should be positive")
	assert.Positive(tb, tick.AskPeriod, "Tick AskPeriod should be positive")
	assert.Positive(tb, tick.AskSize, "Tick AskSize should be positive")
	assert.Positive(tb, tick.Last, "Tick Last should be positive")
	// Can't test FRRAmountAvailable as it's occasionally 0
}

func TestGetTrades(t *testing.T) {
	t.Parallel()

	r, err := b.GetTrades(t.Context(), "tBTCUSD", 5, time.Time{}, time.Time{}, false)
	require.NoError(t, err, "GetTrades must not error")
	assert.NotEmpty(t, r, "GetTrades should return some trades")
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := b.GetOrderbook(t.Context(), "tBTCUSD", "R0", 1)
	if err != nil {
		t.Error(err)
	}

	_, err = b.GetOrderbook(t.Context(), "fUSD", "R0", 1)
	if err != nil {
		t.Error(err)
	}

	_, err = b.GetOrderbook(t.Context(), "tBTCUSD", "P0", 1)
	if err != nil {
		t.Error(err)
	}

	_, err = b.GetOrderbook(t.Context(), "fUSD", "P0", 1)
	if err != nil {
		t.Error(err)
	}

	_, err = b.GetOrderbook(t.Context(), "tLINK:UST", "P0", 1)
	if err != nil {
		t.Error(err)
	}
}

func TestGetStats(t *testing.T) {
	t.Parallel()
	_, err := b.GetStats(t.Context(), "btcusd")
	if err != nil {
		t.Error(err)
	}
}

func TestGetFundingBook(t *testing.T) {
	t.Parallel()
	_, err := b.GetFundingBook(t.Context(), "usd")
	if err != nil {
		t.Error(err)
	}
}

func TestGetLends(t *testing.T) {
	t.Parallel()
	_, err := b.GetLends(t.Context(), "usd", nil)
	if err != nil {
		t.Error(err)
	}
}

func TestGetCandles(t *testing.T) {
	t.Parallel()
	c, err := b.GetCandles(t.Context(), "fUST", "1D", time.Now().AddDate(0, 0, -1), time.Now(), 10000, true)
	require.NoError(t, err, "GetCandles must not error")
	assert.NotEmpty(t, c, "GetCandles should return some candles")
}

func TestGetLeaderboard(t *testing.T) {
	t.Parallel()
	// Test invalid key
	_, err := b.GetLeaderboard(t.Context(), "", "", "", 0, 0, "", "")
	if err == nil {
		t.Error("an error should have been thrown for an invalid key")
	}
	// Test default
	_, err = b.GetLeaderboard(t.Context(),
		LeaderboardUnrealisedProfitInception,
		"1M",
		"tGLOBAL:USD",
		0,
		0,
		"",
		"")
	if err != nil {
		t.Fatal(err)
	}
	// Test params
	var result []LeaderboardEntry
	result, err = b.GetLeaderboard(t.Context(),
		LeaderboardUnrealisedProfitInception,
		"1M",
		"tGLOBAL:USD",
		-1,
		1000,
		"1582695181661",
		"1583299981661")
	if err != nil {
		t.Fatal(err)
	}
	if len(result) == 0 {
		t.Error("should have retrieved leaderboard data")
	}
}

func TestGetAccountFees(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	_, err := b.UpdateAccountInfo(t.Context(), asset.Spot)
	if err != nil {
		t.Error("GetAccountInfo error", err)
	}
}

func TestGetWithdrawalFee(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetWithdrawalFees(t.Context())
	if err != nil {
		t.Error("GetAccountInfo error", err)
	}
}

func TestGetAccountSummary(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetAccountSummary(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestNewDeposit(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.NewDeposit(t.Context(), "blabla", "testwallet", 0)
	if err != nil {
		t.Error(err)
	}

	_, err = b.NewDeposit(t.Context(), "bitcoin", "testwallet", 0)
	if err != nil {
		t.Error(err)
	}

	_, err = b.NewDeposit(t.Context(), "ripple", "", 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetKeyPermissions(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetKeyPermissions(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarginInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetMarginInfo(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestGetAccountBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetAccountBalance(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestWalletTransfer(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.WalletTransfer(t.Context(), 0.01, "btc", "bla", "bla")
	if err != nil {
		t.Error(err)
	}
}

func TestNewOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.NewOrder(t.Context(),
		"BTCUSD",
		order.Limit.Lower(),
		-1,
		2,
		false,
		true)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()

	_, err := b.UpdateTicker(t.Context(), btcusdPair, asset.Spot)
	assert.NoError(t, common.ExcludeError(err, ticker.ErrBidEqualsAsk), "UpdateTicker may only error about locked markets")
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()

	b := new(Bitfinex) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes
	require.NoError(t, testexch.Setup(b), "Test instance Setup must not error")
	testexch.UpdatePairsOnce(t, b)

	for _, a := range b.GetAssetTypes(true) {
		avail, err := b.GetAvailablePairs(a)
		require.NoError(t, err, "GetAvailablePairs must not error")

		err = b.CurrencyPairs.StorePairs(a, avail, true)
		require.NoError(t, err, "StorePairs must not error")

		err = b.UpdateTickers(t.Context(), a)
		require.NoError(t, common.ExcludeError(err, ticker.ErrBidEqualsAsk), "UpdateTickers must only error about locked markets")

		// Bitfinex leaves delisted pairs in Available info/conf endpoints
		// We want to assert that most pairs are valid, so we'll check that no more than 5% are erroring
		acceptableThreshold := 95.0
		okay := 0.0
		var errs error
		for _, p := range avail {
			if _, err = ticker.GetTicker(b.Name, p, a); err != nil {
				errs = common.AppendError(errs, err)
			} else {
				okay++
			}
		}
		if !assert.Greaterf(t, okay/float64(len(avail))*100.0, acceptableThreshold, "At least %.f%% of %s tickers should not error", acceptableThreshold, a) {
			assert.NoError(t, errs, "Collection of all the ticker errors")
		}
	}
}

func TestNewOrderMulti(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	newOrder := []PlaceOrder{
		{
			Symbol:   "BTCUSD",
			Amount:   1,
			Price:    1,
			Exchange: "bitfinex",
			Side:     order.Buy.Lower(),
			Type:     order.Limit.Lower(),
		},
	}

	_, err := b.NewOrderMulti(t.Context(), newOrder)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.CancelExistingOrder(t.Context(), 1337)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelMultipleOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.CancelMultipleOrders(t.Context(), []int64{1337, 1336})
	if err != nil {
		t.Error(err)
	}
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.CancelAllExistingOrders(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestReplaceOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.ReplaceOrder(t.Context(), 1337, "BTCUSD",
		1, 1, true, order.Limit.Lower(), false)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetOrderStatus(t.Context(), 1337)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetOpenOrders(t.Context())
	if err != nil {
		t.Error(err)
	}

	_, err = b.GetOpenOrders(t.Context(), 1, 2, 3, 4)
	if err != nil {
		t.Error(err)
	}
}

func TestGetActivePositions(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetActivePositions(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestClaimPosition(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.ClaimPosition(t.Context(), 1337)
	if err != nil {
		t.Error(err)
	}
}

func TestGetBalanceHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetBalanceHistory(t.Context(),
		"USD", time.Time{}, time.Time{}, 1, "deposit")
	if err != nil {
		t.Error(err)
	}
}

func TestGetMovementHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetMovementHistory(t.Context(), "USD", "bitcoin", time.Time{}, time.Time{}, 1)
	if err != nil {
		t.Error(err)
	}
}

func TestMovementHistoryUnmarshalJSON(t *testing.T) {
	t.Parallel()
	deposit := []byte(`[13105603,"ETH","ETHEREUM",null,null,1569348774000,1569348774000,null,null,"COMPLETED",null,null,0.26300954,-0.00135,null,null,"DESTINATION_ADDRESS",null,null,null,"TRANSACTION_ID",null]`)
	var result MovementHistory
	require.NoError(t, json.Unmarshal(deposit, &result))
	stringPtr := func(s string) *string {
		return &s
	}
	exp := MovementHistory{
		ID:                 13105603,
		Currency:           "ETH",
		CurrencyName:       "ETHEREUM",
		MTSStarted:         types.Time(time.Unix(1569348774, 0)),
		MTSUpdated:         types.Time(time.Unix(1569348774, 0)),
		Status:             "COMPLETED",
		Amount:             0.26300954,
		Fees:               -0.00135,
		DestinationAddress: "DESTINATION_ADDRESS",
		TransactionID:      stringPtr("TRANSACTION_ID"),
		TransactionType:    "deposit",
	}
	assert.Equal(t, exp, result, "MovementHistory should unmarshal correctly")
	withdrawal := []byte(`[13293039,"ETH","ETHEREUM",null,null,1574175052000,1574181326000,null,null,"CANCELED",null,null,-0.24,-0.00135,null,null,"DESTINATION_ADDRESS",null,null,null,"TRANSACTION_ID","Purchase of 100 pizzas"]`)
	require.NoError(t, json.Unmarshal(withdrawal, &result))
	exp = MovementHistory{
		ID:                 13293039,
		Currency:           "ETH",
		CurrencyName:       "ETHEREUM",
		MTSStarted:         types.Time(time.Unix(1574175052, 0)),
		MTSUpdated:         types.Time(time.Unix(1574181326, 0)),
		Status:             "CANCELED",
		Amount:             -0.24,
		Fees:               -0.00135,
		DestinationAddress: "DESTINATION_ADDRESS",
		TransactionID:      stringPtr("TRANSACTION_ID"),
		TransactionNote:    stringPtr("Purchase of 100 pizzas"),
		TransactionType:    "withdrawal",
	}
	assert.Equal(t, exp, result, "MovementHistory should unmarshal correctly")
}

func TestNewOffer(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.NewOffer(t.Context(), "BTC", 1, 1, 1, "loan")
	if err != nil {
		t.Error(err)
	}
}

func TestCancelOffer(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.CancelOffer(t.Context(), 1337)
	if err != nil {
		t.Error(err)
	}
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOfferStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetOfferStatus(t.Context(), 1337)
	if err != nil {
		t.Error(err)
	}
}

func TestGetActiveCredits(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetActiveCredits(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestGetActiveOffers(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetActiveOffers(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestGetActiveMarginFunding(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetActiveMarginFunding(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestGetUnusedMarginFunds(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetUnusedMarginFunds(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarginTotalTakenFunds(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetMarginTotalTakenFunds(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestCloseMarginFunding(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.CloseMarginFunding(t.Context(), 1337)
	if err != nil {
		t.Error(err)
	}
}

func setFeeBuilder() *exchange.FeeBuilder {
	return &exchange.FeeBuilder{
		Amount:        1,
		FeeType:       exchange.CryptocurrencyTradeFee,
		Pair:          currency.NewPair(currency.BTC, currency.LTC),
		PurchasePrice: 1,
	}
}

// TestGetFeeByTypeOfflineTradeFee logic test
func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	feeBuilder := setFeeBuilder()
	_, err := b.GetFeeByType(t.Context(), feeBuilder)
	if err != nil {
		t.Fatal(err)
	}
	if !sharedtestvalues.AreAPICredentialsSet(b) {
		if feeBuilder.FeeType != exchange.OfflineTradeFee {
			t.Errorf("Expected %v, received %v", exchange.OfflineTradeFee, feeBuilder.FeeType)
		}
	} else {
		if feeBuilder.FeeType != exchange.CryptocurrencyTradeFee {
			t.Errorf("Expected %v, received %v", exchange.CryptocurrencyTradeFee, feeBuilder.FeeType)
		}
	}
}

func TestGetFee(t *testing.T) {
	feeBuilder := setFeeBuilder()
	t.Parallel()

	if sharedtestvalues.AreAPICredentialsSet(b) {
		// CryptocurrencyTradeFee Basic
		if _, err := b.GetFee(t.Context(), feeBuilder); err != nil {
			t.Error(err)
		}

		// CryptocurrencyTradeFee High quantity
		feeBuilder = setFeeBuilder()
		feeBuilder.Amount = 1000
		feeBuilder.PurchasePrice = 1000
		if _, err := b.GetFee(t.Context(), feeBuilder); err != nil {
			t.Error(err)
		}

		// CryptocurrencyTradeFee IsMaker
		feeBuilder = setFeeBuilder()
		feeBuilder.IsMaker = true
		if _, err := b.GetFee(t.Context(), feeBuilder); err != nil {
			t.Error(err)
		}

		// CryptocurrencyTradeFee Negative purchase price
		feeBuilder = setFeeBuilder()
		feeBuilder.PurchasePrice = -1000
		if _, err := b.GetFee(t.Context(), feeBuilder); err != nil {
			t.Error(err)
		}

		// CryptocurrencyWithdrawalFee Basic
		feeBuilder = setFeeBuilder()
		feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
		if _, err := b.GetFee(t.Context(), feeBuilder); err != nil {
			t.Error(err)
		}
	}

	// CryptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyDepositFee
	if _, err := b.GetFee(t.Context(), feeBuilder); err != nil {
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	feeBuilder.FiatCurrency = currency.HKD
	if _, err := b.GetFee(t.Context(), feeBuilder); err != nil {
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.HKD
	if _, err := b.GetFee(t.Context(), feeBuilder); err != nil {
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	t.Parallel()
	expectedResult := exchange.AutoWithdrawCryptoWithAPIPermissionText + " & " + exchange.AutoWithdrawFiatWithAPIPermissionText
	withdrawPermissions := b.FormatWithdrawPermissions()
	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s", expectedResult, withdrawPermissions)
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	getOrdersRequest := order.MultiOrderRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
		Side:      order.AnySide,
	}

	_, err := b.GetActiveOrders(t.Context(), &getOrdersRequest)
	if sharedtestvalues.AreAPICredentialsSet(b) && err != nil {
		t.Errorf("Could not get open orders: %s", err)
	} else if !sharedtestvalues.AreAPICredentialsSet(b) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	getOrdersRequest := order.MultiOrderRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
		Side:      order.AnySide,
	}
	_, err := b.GetOrderHistory(t.Context(), &getOrdersRequest)
	if err != nil {
		t.Error(err)
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------
func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, b, canManipulateRealOrders)
	orderSubmission := &order.Submit{
		Exchange: b.Name,
		Pair: currency.Pair{
			Delimiter: "_",
			Base:      currency.XRP,
			Quote:     currency.USD,
		},
		AssetType: asset.Spot,
		Side:      order.Sell,
		Type:      order.Limit,
		Price:     1000,
		Amount:    20,
		ClientID:  "meowOrder",
	}
	response, err := b.SubmitOrder(t.Context(), orderSubmission)

	if sharedtestvalues.AreAPICredentialsSet(b) && err != nil {
		t.Fatalf("Could not place order: %v", err)
	}
	if sharedtestvalues.AreAPICredentialsSet(b) && response.Status != order.New {
		t.Error("Order not placed")
	}
	if !sharedtestvalues.AreAPICredentialsSet(b) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, b, canManipulateRealOrders)

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)
	orderCancellation := &order.Cancel{
		OrderID:   "1",
		AccountID: "1",
		Pair:      currencyPair,
		AssetType: asset.Spot,
	}

	err := b.CancelOrder(t.Context(), orderCancellation)
	if !sharedtestvalues.AreAPICredentialsSet(b) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if sharedtestvalues.AreAPICredentialsSet(b) && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, b, canManipulateRealOrders)

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)
	orderCancellation := &order.Cancel{
		OrderID:   "1",
		AccountID: "1",
		Pair:      currencyPair,
		AssetType: asset.Spot,
	}

	resp, err := b.CancelAllOrders(t.Context(), orderCancellation)

	if !sharedtestvalues.AreAPICredentialsSet(b) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if sharedtestvalues.AreAPICredentialsSet(b) && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}

	if len(resp.Status) > 0 {
		t.Errorf("%v orders failed to cancel", len(resp.Status))
	}
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.ModifyOrder(
		t.Context(),
		&order.Modify{
			OrderID:   "1337",
			AssetType: asset.Spot,
			Pair:      currency.NewBTCUSD(),
		})
	if err != nil {
		t.Error(err)
	}
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, b, canManipulateRealOrders)

	withdrawCryptoRequest := withdraw.Request{
		Exchange:    b.Name,
		Amount:      -1,
		Currency:    currency.USDT,
		Description: "WITHDRAW IT ALL",
		Crypto: withdraw.CryptoRequest{
			Address: "0x1nv4l1d",
			Chain:   "tetheruse",
		},
	}

	_, err := b.WithdrawCryptocurrencyFunds(t.Context(),
		&withdrawCryptoRequest)
	if !sharedtestvalues.AreAPICredentialsSet(b) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if sharedtestvalues.AreAPICredentialsSet(b) && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

func TestWithdrawFiat(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, b, canManipulateRealOrders)

	withdrawFiatRequest := withdraw.Request{
		Amount:      -1,
		Currency:    currency.USD,
		Description: "WITHDRAW IT ALL",
		Fiat: withdraw.FiatRequest{
			WireCurrency: currency.USD.String(),
		},
	}

	_, err := b.WithdrawFiatFunds(t.Context(), &withdrawFiatRequest)
	if !sharedtestvalues.AreAPICredentialsSet(b) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if sharedtestvalues.AreAPICredentialsSet(b) && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, b, canManipulateRealOrders)

	withdrawFiatRequest := withdraw.Request{
		Amount:      -1,
		Currency:    currency.BTC,
		Description: "WITHDRAW IT ALL",
		Fiat: withdraw.FiatRequest{
			WireCurrency:                  currency.USD.String(),
			RequiresIntermediaryBank:      true,
			IsExpressWire:                 false,
			IntermediaryBankAccountNumber: 12345,
			IntermediaryBankAddress:       "123 Fake St",
			IntermediaryBankCity:          "Tarry Town",
			IntermediaryBankCountry:       "Hyrule",
			IntermediaryBankName:          "Federal Reserve Bank",
			IntermediarySwiftCode:         "Taylor",
		},
	}

	_, err := b.WithdrawFiatFundsToInternationalBank(t.Context(),
		&withdrawFiatRequest)
	if !sharedtestvalues.AreAPICredentialsSet(b) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if sharedtestvalues.AreAPICredentialsSet(b) && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	if sharedtestvalues.AreAPICredentialsSet(b) {
		_, err := b.GetDepositAddress(t.Context(), currency.USDT, "", "TETHERUSE")
		if err != nil {
			t.Error("GetDepositAddress() error", err)
		}
	} else {
		_, err := b.GetDepositAddress(t.Context(), currency.BTC, "deposit", "")
		if err == nil {
			t.Error("GetDepositAddress() error cannot be nil")
		}
	}
}

// TestWSAuth dials websocket, sends login request.
func TestWSAuth(t *testing.T) {
	if !b.Websocket.IsEnabled() {
		t.Skip(websocket.ErrWebsocketNotEnabled.Error())
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	if !b.API.AuthenticatedWebsocketSupport {
		t.Skip("Authentecated API support not enabled")
	}
	testexch.SetupWs(t, b)
	require.True(t, b.Websocket.CanUseAuthenticatedEndpoints(), "CanUseAuthenticatedEndpoints must be turned on")

	var resp map[string]any
	catcher := func() (ok bool) {
		select {
		case v := <-b.Websocket.ToRoutine:
			resp, ok = v.(map[string]any)
		default:
		}
		return
	}

	if assert.Eventually(t, catcher, sharedtestvalues.WebsocketResponseDefaultTimeout, time.Millisecond*10, "Auth response should arrive") {
		assert.Equal(t, "auth", resp["event"], "event should be correct")
		assert.Equal(t, "OK", resp["status"], "status should be correct")
		assert.NotEmpty(t, resp["auth_id"], "status should be correct")
	}
}

func TestGenerateSubscriptions(t *testing.T) {
	t.Parallel()

	b := new(Bitfinex) //nolint:govet // Intentional shadow of b to avoid future copy/paste mistakes
	require.NoError(t, testexch.Setup(b), "Setup must not error")
	b.Websocket.SetCanUseAuthenticatedEndpoints(true)
	require.True(t, b.Websocket.CanUseAuthenticatedEndpoints(), "CanUseAuthenticatedEndpoints must return true")
	subs, err := b.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	exp := subscription.List{}
	for _, baseSub := range b.Features.Subscriptions {
		for _, a := range b.GetAssetTypes(true) {
			if baseSub.Asset != asset.All && baseSub.Asset != a {
				continue
			}
			pairs, err := b.GetEnabledPairs(a)
			require.NoErrorf(t, err, "GetEnabledPairs %s must not error", a)
			for _, p := range pairs.Format(currency.PairFormat{Uppercase: true}) {
				s := baseSub.Clone()
				s.Asset = a
				s.Pairs = currency.Pairs{p}
				prefix := "t"
				if a == asset.MarginFunding {
					prefix = "f"
				}
				switch s.Channel {
				case subscription.TickerChannel:
					s.QualifiedChannel = `{"channel":"ticker","symbol":"` + prefix + p.String() + `"}`
				case subscription.CandlesChannel:
					if a == asset.MarginFunding {
						s.QualifiedChannel = `{"channel":"candles","key":"trade:1m:` + prefix + p.String() + `:p30"}`
					} else {
						s.QualifiedChannel = `{"channel":"candles","key":"trade:1m:` + prefix + p.String() + `"}`
					}
				case subscription.OrderbookChannel:
					s.QualifiedChannel = `{"channel":"book","len":100,"prec":"R0","symbol":"` + prefix + p.String() + `"}`
				case subscription.AllTradesChannel:
					s.QualifiedChannel = `{"channel":"trades","symbol":"` + prefix + p.String() + `"}`
				}
				exp = append(exp, s)
			}
		}
	}
	testsubs.EqualLists(t, exp, subs)
}

// TestWSSubscribe tests Subscribe and Unsubscribe functionality
// See also TestSubscribeReq which covers key and symbol conversion
func TestWSSubscribe(t *testing.T) {
	b := new(Bitfinex) //nolint:govet // Intentional shadow of b to avoid future copy/paste mistakes
	require.NoError(t, testexch.Setup(b), "TestInstance must not error")
	testexch.SetupWs(t, b)
	err := b.Subscribe(subscription.List{{Channel: subscription.TickerChannel, Pairs: currency.Pairs{currency.NewBTCUSD()}, Asset: asset.Spot}})
	require.NoError(t, err, "Subrcribe must not error")
	catcher := func() (ok bool) {
		i := <-b.Websocket.ToRoutine
		_, ok = i.(*ticker.Price)
		return
	}
	assert.Eventually(t, catcher, sharedtestvalues.WebsocketResponseDefaultTimeout, time.Millisecond*10, "Ticker response should arrive")

	subs, err := b.GetSubscriptions()
	require.NoError(t, err, "GetSubscriptions must not error")
	require.Len(t, subs, 1, "We must only have 1 subscription; subID subscription must have been Removed by subscribeToChan")

	err = b.Subscribe(subscription.List{{Channel: subscription.TickerChannel, Pairs: currency.Pairs{currency.NewBTCUSD()}, Asset: asset.Spot}})
	require.ErrorContains(t, err, "subscribe: dup (code: 10301)", "Duplicate subscription must error correctly")

	assert.EventuallyWithT(t, func(t *assert.CollectT) {
		i := <-b.Websocket.ToRoutine
		e, ok := i.(error)
		require.True(t, ok, "must find an error")
		assert.ErrorContains(t, e, "subscribe: dup (code: 10301)", "error should be correct")
	}, sharedtestvalues.WebsocketResponseDefaultTimeout, time.Millisecond*10, "error response should go to ToRoutine")

	subs, err = b.GetSubscriptions()
	require.NoError(t, err, "GetSubscriptions must not error")
	require.Len(t, subs, 1, "We must only have one subscription after an error attempt")

	err = b.Unsubscribe(subs)
	assert.NoError(t, err, "Unsubscribing should not error")

	chanID, ok := subs[0].Key.(int)
	assert.True(t, ok, "sub.Key should be an int")

	err = b.Unsubscribe(subs)
	assert.ErrorContains(t, err, strconv.Itoa(chanID), "Unsubscribe should contain correct chanId")
	assert.ErrorContains(t, err, "unsubscribe: invalid (code: 10400)", "Unsubscribe should contain correct upstream error")

	err = b.Subscribe(subscription.List{{
		Channel: subscription.TickerChannel,
		Pairs:   currency.Pairs{currency.NewBTCUSD()},
		Asset:   asset.Spot,
		Params:  map[string]any{"key": "tBTCUSD"},
	}})
	assert.ErrorIs(t, err, errParamNotAllowed, "Trying to use a 'key' param should error errParamNotAllowed")
}

// TestSubToMap tests the channel to request map marshalling
func TestSubToMap(t *testing.T) {
	s := &subscription.Subscription{
		Channel:  subscription.CandlesChannel,
		Asset:    asset.Spot,
		Pairs:    currency.Pairs{currency.NewBTCUSD()},
		Interval: kline.OneMin,
	}

	r := subToMap(s, s.Asset, s.Pairs[0])
	assert.Equal(t, "trade:1m:tBTCUSD", r["key"], "key should contain a specific timeframe and no period")

	s.Interval = kline.FifteenMin
	s.Asset = asset.MarginFunding
	s.Params = map[string]any{CandlesPeriodKey: "p30"}

	r = subToMap(s, s.Asset, s.Pairs[0])
	assert.Equal(t, "trade:15m:fBTCUSD:p30", r["key"], "key should contain a period and specific timeframe")

	s.Interval = kline.FifteenMin

	s = &subscription.Subscription{
		Channel: subscription.OrderbookChannel,
		Pairs:   currency.Pairs{currency.NewPair(currency.BTC, currency.DOGE)},
		Asset:   asset.Spot,
	}
	r = subToMap(s, s.Asset, s.Pairs[0])
	assert.Equal(t, "tBTC:DOGE", r["symbol"], "symbol should use colon delimiter if a currency is > 3 chars")

	s.Pairs = currency.Pairs{currency.NewPair(currency.BTC, currency.LTC)}
	r = subToMap(s, s.Asset, s.Pairs[0])
	assert.Equal(t, "tBTCLTC", r["symbol"], "symbol should not use colon delimiter if both currencies < 3 chars")
}

func TestWSNewOrder(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	testexch.SetupWs(t, b)
	_, err := b.WsNewOrder(t.Context(), &WsNewOrderRequest{
		GroupID: 1,
		Type:    "EXCHANGE LIMIT",
		Symbol:  "tXRPUSD",
		Amount:  -20,
		Price:   1000,
	})
	assert.NoError(t, err)
}

func TestWSCancelOrder(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	testexch.SetupWs(t, b)
	err := b.WsCancelOrder(t.Context(), 1234)
	assert.NoError(t, err)
}

func TestWSModifyOrder(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	testexch.SetupWs(t, b)
	err := b.WsModifyOrder(t.Context(), &WsUpdateOrderRequest{
		OrderID: 1234,
		Price:   -111,
		Amount:  111,
	})
	assert.NoError(t, err)
}

func TestWSCancelAllOrders(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	testexch.SetupWs(t, b)
	err := b.WsCancelAllOrders(t.Context())
	assert.NoError(t, err)
}

func TestWSCancelMultiOrders(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	testexch.SetupWs(t, b)
	err := b.WsCancelMultiOrders(t.Context(), []int64{1, 2, 3, 4})
	assert.NoError(t, err)
}

func TestWSNewOffer(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	testexch.SetupWs(t, b)
	err := b.WsNewOffer(t.Context(), &WsNewOfferRequest{
		Type:   order.Limit.String(),
		Symbol: "fBTC",
		Amount: -10,
		Rate:   10,
		Period: 30,
	})
	assert.NoError(t, err)
}

func TestWSCancelOffer(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	testexch.SetupWs(t, b)
	err := b.WsCancelOffer(t.Context(), 1234)
	assert.NoError(t, err)
}

func TestWSSubscribedResponse(t *testing.T) {
	ch, err := b.Websocket.Match.Set("subscribe:waiter1", 1)
	assert.NoError(t, err, "Setting a matcher should not error")
	err = b.wsHandleData(t.Context(), []byte(`{"event":"subscribed","channel":"ticker","chanId":224555,"subId":"waiter1","symbol":"tBTCUSD","pair":"BTCUSD"}`))
	if assert.Error(t, err, "Should error if sub is not registered yet") {
		assert.ErrorIs(t, err, websocket.ErrSubscriptionFailure, "Should error SubFailure if sub isn't registered yet")
		assert.ErrorIs(t, err, subscription.ErrNotFound, "Should error SubNotFound if sub isn't registered yet")
		assert.ErrorContains(t, err, "waiter1", "Should error containing subID if")
	}

	err = b.Websocket.AddSubscriptions(b.Websocket.Conn, &subscription.Subscription{Key: "waiter1"})
	require.NoError(t, err, "AddSubscriptions must not error")
	err = b.wsHandleData(t.Context(), []byte(`{"event":"subscribed","channel":"ticker","chanId":224555,"subId":"waiter1","symbol":"tBTCUSD","pair":"BTCUSD"}`))
	assert.NoError(t, err, "wsHandleData should not error")
	if assert.NotEmpty(t, ch, "Matcher should have received a sub notification") {
		msg := <-ch
		cID, err := jsonparser.GetInt(msg, "chanId")
		assert.NoError(t, err, "Should get chanId from sub notification without error")
		assert.EqualValues(t, 224555, cID, "Should get the correct chanId through the matcher notification")
	}
}

func TestWSOrderBook(t *testing.T) {
	err := b.Websocket.AddSubscriptions(b.Websocket.Conn, &subscription.Subscription{Key: 23405, Asset: asset.Spot, Pairs: currency.Pairs{btcusdPair}, Channel: subscription.OrderbookChannel})
	require.NoError(t, err, "AddSubscriptions must not error")
	pressXToJSON := `[23405,[[38334303613,9348.8,0.53],[38334308111,9348.8,5.98979404],[38331335157,9344.1,1.28965787],[38334302803,9343.8,0.08230094],[38334279092,9343,0.8],[38334307036,9342.938663676,0.8],[38332749107,9342.9,0.2],[38332277330,9342.8,0.85],[38329406786,9342,0.1432012],[38332841570,9341.947288638,0.3],[38332163238,9341.7,0.3],[38334303384,9341.6,0.324],[38332464840,9341.4,0.5],[38331935870,9341.2,0.5],[38334312082,9340.9,0.02126899],[38334261292,9340.8,0.26763],[38334138680,9340.625455254,0.12],[38333896802,9339.8,0.85],[38331627527,9338.9,1.57863959],[38334186713,9338.9,0.26769],[38334305819,9338.8,2.999],[38334211180,9338.75285796,3.999],[38334310699,9337.8,0.10679883],[38334307414,9337.5,1],[38334179822,9337.1,0.26773],[38334306600,9336.659955102,1.79],[38334299667,9336.6,1.1],[38334306452,9336.6,0.13979771],[38325672859,9336.3,1.25],[38334311646,9336.2,1],[38334258509,9336.1,0.37],[38334310592,9336,1.79],[38334310378,9335.6,1.43],[38334132444,9335.2,0.26777],[38331367325,9335,0.07],[38334310703,9335,0.10680562],[38334298209,9334.7,0.08757301],[38334304857,9334.456899462,0.291],[38334309940,9334.088390727,0.0725],[38334310377,9333.7,1.2868],[38334297615,9333.607784,0.1108],[38334095188,9333.3,0.26785],[38334228913,9332.7,0.40861186],[38334300526,9332.363996604,0.3884],[38334310701,9332.2,0.10680562],[38334303548,9332.005382871,0.07],[38334311798,9331.8,0.41285228],[38334301012,9331.7,1.7952],[38334089877,9331.4,0.2679],[38321942150,9331.2,0.2],[38334310670,9330,1.069],[38334063096,9329.6,0.26796],[38334310700,9329.4,0.10680562],[38334310404,9329.3,1],[38334281630,9329.1,6.57150597],[38334036864,9327.7,0.26801],[38334310702,9326.6,0.10680562],[38334311799,9326.1,0.50220625],[38334164163,9326,0.219638],[38334309722,9326,1.5],[38333051682,9325.8,0.26807],[38334302027,9325.7,0.75],[38334203435,9325.366592,0.32397696],[38321967613,9325,0.05],[38334298787,9324.9,0.3],[38334301719,9324.8,3.6227592],[38331316716,9324.763454646,0.71442],[38334310698,9323.8,0.10680562],[38334035499,9323.7,0.23431017],[38334223472,9322.670551788,0.42150603],[38334163459,9322.560399006,0.143967],[38321825171,9320.8,2],[38334075805,9320.467496148,0.30772633],[38334075800,9319.916732238,0.61457592],[38333682302,9319.7,0.0011],[38331323088,9319.116771762,0.12913],[38333677480,9319,0.0199],[38334277797,9318.6,0.89],[38325235155,9318.041088,1.20249],[38334310910,9317.82382938,1.79],[38334311811,9317.2,0.61079138],[38334311812,9317.2,0.71937652],[38333298214,9317.1,50],[38334306359,9317,1.79],[38325531545,9316.382823951,0.21263],[38333727253,9316.3,0.02316372],[38333298213,9316.1,45],[38333836479,9316,2.135],[38324520465,9315.9,2.7681],[38334307411,9315.5,1],[38330313617,9315.3,0.84455],[38334077770,9315.294024,0.01248397],[38334286663,9315.294024,1],[38325533762,9315.290315394,2.40498],[38334310018,9315.2,3],[38333682617,9314.6,0.0011],[38334304794,9314.6,0.76364676],[38334304798,9314.3,0.69242113],[38332915733,9313.8,0.0199],[38334084411,9312.8,1],[38334311893,9350.1,-1.015],[38334302734,9350.3,-0.26737],[38334300732,9350.8,-5.2],[38333957619,9351,-0.90677089],[38334300521,9351,-1.6457],[38334301600,9351.012829557,-0.0523],[38334308878,9351.7,-2.5],[38334299570,9351.921544,-0.1015],[38334279367,9352.1,-0.26732],[38334299569,9352.411802928,-0.4036],[38334202773,9353.4,-0.02139404],[38333918472,9353.7,-1.96412776],[38334278782,9354,-0.26731],[38334278606,9355,-1.2785],[38334302105,9355.439221251,-0.79191542],[38313897370,9355.569409242,-0.43363],[38334292995,9355.584296,-0.0979],[38334216989,9355.8,-0.03686414],[38333894025,9355.9,-0.26721],[38334293798,9355.936691952,-0.4311],[38331159479,9356,-0.4204022],[38333918888,9356.1,-1.10885563],[38334298205,9356.4,-0.20124428],[38328427481,9356.5,-0.1],[38333343289,9356.6,-0.41034213],[38334297205,9356.6,-0.08835018],[38334277927,9356.741101161,-0.0737],[38334311645,9356.8,-0.5],[38334309002,9356.9,-5],[38334309736,9357,-0.10680107],[38334306448,9357.4,-0.18645275],[38333693302,9357.7,-0.2672],[38332815159,9357.8,-0.0011],[38331239824,9358.2,-0.02],[38334271608,9358.3,-2.999],[38334311971,9358.4,-0.55],[38333919260,9358.5,-1.9972841],[38334265365,9358.5,-1.7841],[38334277960,9359,-3],[38334274601,9359.020969848,-3],[38326848839,9359.1,-0.84],[38334291080,9359.247048,-0.16199869],[38326848844,9359.4,-1.84],[38333680200,9359.6,-0.26713],[38331326606,9359.8,-0.84454],[38334309738,9359.8,-0.10680107],[38331314707,9359.9,-0.2],[38333919803,9360.9,-1.41177599],[38323651149,9361.33417827,-0.71442],[38333656906,9361.5,-0.26705],[38334035500,9361.5,-0.40861586],[38334091886,9362.4,-6.85940815],[38334269617,9362.5,-4],[38323629409,9362.545858872,-2.40497],[38334309737,9362.7,-0.10680107],[38334312380,9362.7,-3],[38325280830,9362.8,-1.75123],[38326622800,9362.8,-1.05145],[38333175230,9363,-0.0011],[38326848745,9363.2,-0.79],[38334308960,9363.206775564,-0.12],[38333920234,9363.3,-1.25318113],[38326848843,9363.4,-1.29],[38331239823,9363.4,-0.02],[38333209613,9363.4,-0.26719],[38334299964,9364,-0.05583123],[38323470224,9364.161816648,-0.12912],[38334284711,9365,-0.21346019],[38334299594,9365,-2.6757062],[38323211816,9365.073132585,-0.21262],[38334312456,9365.1,-0.11167861],[38333209612,9365.2,-0.26719],[38327770474,9365.3,-0.0073],[38334298788,9365.3,-0.3],[38334075803,9365.409831204,-0.30772637],[38334309740,9365.5,-0.10680107],[38326608767,9365.7,-2.76809],[38333920657,9365.7,-1.25848083],[38329594226,9366.6,-0.02587],[38334311813,9366.7,-4.72290945],[38316386301,9367.39258128,-2.37581],[38334302026,9367.4,-4.5],[38334228915,9367.9,-0.81725458],[38333921381,9368.1,-1.72213641],[38333175678,9368.2,-0.0011],[38334301150,9368.2,-2.654604],[38334297208,9368.3,-0.78036466],[38334309739,9368.3,-0.10680107],[38331227515,9368.7,-0.02],[38331184470,9369,-0.003975],[38334203436,9369.319616,-0.32397695],[38334269964,9369.7,-0.5],[38328386732,9370,-4.11759935],[38332719555,9370,-0.025],[38333921935,9370.5,-1.2224398],[38334258511,9370.5,-0.35],[38326848842,9370.8,-0.34],[38333985038,9370.9,-0.8551502],[38334283018,9370.9,-1],[38326848744,9371,-1.34]],5]`
	err = b.wsHandleData(t.Context(), []byte(pressXToJSON))
	if err != nil {
		t.Error(err)
	}
	pressXToJSON = `[23405,[7617,52.98726298,7617.1,53.601795929999994,-550.9,-0.0674,7617,8318.92961981,8257.8,7500],6]`
	err = b.wsHandleData(t.Context(), []byte(pressXToJSON))
	if err != nil {
		t.Error(err)
	}
	pressXToJSON = `[23405,[7617,52.98726298,7617.1,53.601795929999994,-550.9,-0.0674,7617,8318.92961981,8257.8,7500]]`
	assert.NotPanics(t, func() { err = b.wsHandleData(t.Context(), []byte(pressXToJSON)) }, "handleWSBookUpdate should not panic when seqNo is not configured to be sent")
	assert.ErrorIs(t, err, errNoSeqNo, "handleWSBookUpdate should send correct error")
}

func TestWSAllTrades(t *testing.T) {
	t.Parallel()

	b := new(Bitfinex) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes
	require.NoError(t, testexch.Setup(b), "Test instance Setup must not error")
	err := b.Websocket.AddSubscriptions(b.Websocket.Conn, &subscription.Subscription{Asset: asset.Spot, Pairs: currency.Pairs{btcusdPair}, Channel: subscription.AllTradesChannel, Key: 18788})
	require.NoError(t, err, "AddSubscriptions must not error")
	testexch.FixtureToDataHandler(t, "testdata/wsAllTrades.json", b.wsHandleData)
	close(b.Websocket.DataHandler)
	expJSON := []string{
		`{"TID":"412685577","AssetType":"spot","Side":"BUY","Price":176.3,"Amount":11.1998,"Timestamp":"2020-01-29T03:27:24.802Z"}`,
		`{"TID":"412685578","AssetType":"spot","Side":"SELL","Price":176.29952759,"Amount":5,"Timestamp":"2020-01-29T03:28:04.802Z"}`,
		`{"TID":"412685579","AssetType":"marginFunding","Side":"BUY","Price":0.1244,"Amount":4.2,"Timestamp":"2020-01-29T03:36:45.757Z"}`,
		`{"TID":"5690221201","AssetType":"spot","Side":"BUY","Price":102570,"Amount":0.00991467,"Timestamp":"2024-12-15T04:30:17.719Z"}`,
		`{"TID":"5690221202","AssetType":"spot","Side":"SELL","Price":102560,"Amount":0.01925285,"Timestamp":"2024-12-15T04:30:17.704Z"}`,
		`{"TID":"5690221203","AssetType":"marginFunding","Side":"BUY","Price":102550,"Amount":0.00991467,"Timestamp":"2024-12-15T04:30:18.019Z"}`,
		`{"TID":"5690221204","AssetType":"marginFunding","Side":"SELL","Price":102540,"Amount":0.01925285,"Timestamp":"2024-12-15T04:30:18.094Z"}`,
	}
	require.Len(t, b.Websocket.DataHandler, len(expJSON), "Must see correct number of trades")
	for resp := range b.Websocket.DataHandler {
		switch v := resp.(type) {
		case trade.Data:
			i := 6 - len(b.Websocket.DataHandler)
			exp := trade.Data{
				Exchange:     b.Name,
				CurrencyPair: btcusdPair,
			}
			require.NoErrorf(t, json.Unmarshal([]byte(expJSON[i]), &exp), "Must not error unmarshalling json %d: %s", i, expJSON[i])
			require.Equalf(t, exp, v, "Trade [%d] must be correct", i)
		case error:
			t.Error(v)
		default:
			t.Errorf("Unexpected type in DataHandler: %T (%s)", v, v)
		}
	}
}

func TestWSTickerResponse(t *testing.T) {
	err := b.Websocket.AddSubscriptions(b.Websocket.Conn, &subscription.Subscription{Asset: asset.Spot, Pairs: currency.Pairs{btcusdPair}, Channel: subscription.TickerChannel, Key: 11534})
	require.NoError(t, err, "AddSubscriptions must not error")
	pressXToJSON := `[11534,[61.304,2228.36155358,61.305,1323.2442970500003,0.395,0.0065,61.371,50973.3020771,62.5,57.421]]`
	err = b.wsHandleData(t.Context(), []byte(pressXToJSON))
	if err != nil {
		t.Error(err)
	}
	pair, err := currency.NewPairFromString("XAUTF0:USTF0")
	if err != nil {
		t.Error(err)
	}
	err = b.Websocket.AddSubscriptions(b.Websocket.Conn, &subscription.Subscription{Asset: asset.Spot, Pairs: currency.Pairs{pair}, Channel: subscription.TickerChannel, Key: 123412})
	require.NoError(t, err, "AddSubscriptions must not error")
	pressXToJSON = `[123412,[61.304,2228.36155358,61.305,1323.2442970500003,0.395,0.0065,61.371,50973.3020771,62.5,57.421]]`
	err = b.wsHandleData(t.Context(), []byte(pressXToJSON))
	if err != nil {
		t.Error(err)
	}
	pair, err = currency.NewPairFromString("trade:1m:tXRPUSD")
	if err != nil {
		t.Error(err)
	}
	err = b.Websocket.AddSubscriptions(b.Websocket.Conn, &subscription.Subscription{Asset: asset.Spot, Pairs: currency.Pairs{pair}, Channel: subscription.TickerChannel, Key: 123413})
	require.NoError(t, err, "AddSubscriptions must not error")
	pressXToJSON = `[123413,[61.304,2228.36155358,61.305,1323.2442970500003,0.395,0.0065,61.371,50973.3020771,62.5,57.421]]`
	err = b.wsHandleData(t.Context(), []byte(pressXToJSON))
	if err != nil {
		t.Error(err)
	}
	pair, err = currency.NewPairFromString("trade:1m:fZRX:p30")
	if err != nil {
		t.Error(err)
	}
	err = b.Websocket.AddSubscriptions(b.Websocket.Conn, &subscription.Subscription{Asset: asset.Spot, Pairs: currency.Pairs{pair}, Channel: subscription.TickerChannel, Key: 123414})
	require.NoError(t, err, "AddSubscriptions must not error")
	pressXToJSON = `[123414,[61.304,2228.36155358,61.305,1323.2442970500003,0.395,0.0065,61.371,50973.3020771,62.5,57.421]]`
	err = b.wsHandleData(t.Context(), []byte(pressXToJSON))
	if err != nil {
		t.Error(err)
	}
}

func TestWSCandleResponse(t *testing.T) {
	err := b.Websocket.AddSubscriptions(b.Websocket.Conn, &subscription.Subscription{Asset: asset.Spot, Pairs: currency.Pairs{btcusdPair}, Channel: subscription.CandlesChannel, Key: 343351})
	require.NoError(t, err, "AddSubscriptions must not error")
	pressXToJSON := `[343351,[[1574698260000,7379.785503,7383.8,7388.3,7379.785503,1.68829482]]]`
	err = b.wsHandleData(t.Context(), []byte(pressXToJSON))
	if err != nil {
		t.Error(err)
	}
	pressXToJSON = `[343351,[1574698200000,7399.9,7379.7,7399.9,7371.8,41.63633658]]`
	err = b.wsHandleData(t.Context(), []byte(pressXToJSON))
	if err != nil {
		t.Error(err)
	}
}

func TestWSOrderSnapshot(t *testing.T) {
	pressXToJSON := `[0,"os",[[34930659963,null,1574955083558,"tETHUSD",1574955083558,1574955083573,0.201104,0.201104,"EXCHANGE LIMIT",null,null,null,0,"ACTIVE",null,null,120,0,0,0,null,null,null,0,0,null,null,null,"BFX",null,null,null]]]`
	err := b.wsHandleData(t.Context(), []byte(pressXToJSON))
	if err != nil {
		t.Error(err)
	}
	pressXToJSON = `[0,"oc",[34930659963,null,1574955083558,"tETHUSD",1574955083558,1574955354487,0.201104,0.201104,"EXCHANGE LIMIT",null,null,null,0,"CANCELED",null,null,120,0,0,0,null,null,null,0,0,null,null,null,"BFX",null,null,null]]`
	err = b.wsHandleData(t.Context(), []byte(pressXToJSON))
	if err != nil {
		t.Error(err)
	}
}

func TestWSNotifications(t *testing.T) {
	pressXToJSON := `[0,"n",[1575282446099,"fon-req",null,null,[41238905,null,null,null,-1000,null,null,null,null,null,null,null,null,null,0.002,2,null,null,null,null,null],null,"SUCCESS","Submitting funding bid of 1000.0 USD at 0.2000 for 2 days."]]`
	err := b.wsHandleData(t.Context(), []byte(pressXToJSON))
	if err != nil {
		t.Error(err)
	}

	pressXToJSON = `[0,"n",[1575287438.515,"on-req",null,null,[1185815098,null,1575287436979,"tETHUSD",1575287438515,1575287438515,-2.5,-2.5,"LIMIT",null,null,null,0,"ACTIVE",null,null,230,0,0,0,null,null,null,0,null,null,null,null,"API>BFX",null,null,null],null,"SUCCESS","Submitting limit sell order for -2.5 ETH."]]`
	err = b.wsHandleData(t.Context(), []byte(pressXToJSON))
	if err != nil {
		t.Error(err)
	}
}

func TestWSFundingOfferSnapshotAndUpdate(t *testing.T) {
	pressXToJSON := `[0,"fos",[[41237920,"fETH",1573912039000,1573912039000,0.5,0.5,"LIMIT",null,null,0,"ACTIVE",null,null,null,0.0024,2,0,0,null,0,null]]]`
	if err := b.wsHandleData(t.Context(), []byte(pressXToJSON)); err != nil {
		t.Error(err)
	}

	pressXToJSON = `[0,"fon",[41238747,"fUST",1575026670000,1575026670000,5000,5000,"LIMIT",null,null,0,"ACTIVE",null,null,null,0.006000000000000001,30,0,0,null,0,null]]`
	if err := b.wsHandleData(t.Context(), []byte(pressXToJSON)); err != nil {
		t.Error(err)
	}
}

func TestWSFundingCreditSnapshotAndUpdate(t *testing.T) {
	pressXToJSON := `[0,"fcs",[[26223578,"fUST",1,1575052261000,1575296187000,350,0,"ACTIVE",null,null,null,0,30,1575052261000,1575293487000,0,0,null,0,null,0,"tBTCUST"],[26223711,"fUSD",-1,1575291961000,1575296187000,180,0,"ACTIVE",null,null,null,0.002,7,1575282446000,1575295587000,0,0,null,0,null,0,"tETHUSD"]]]`
	if err := b.wsHandleData(t.Context(), []byte(pressXToJSON)); err != nil {
		t.Error(err)
	}

	pressXToJSON = `[0,"fcu",[26223578,"fUST",1,1575052261000,1575296787000,350,0,"ACTIVE",null,null,null,0,30,1575052261000,1575293487000,0,0,null,0,null,0,"tBTCUST"]]`
	if err := b.wsHandleData(t.Context(), []byte(pressXToJSON)); err != nil {
		t.Error(err)
	}
}

func TestWSFundingLoanSnapshotAndUpdate(t *testing.T) {
	pressXToJSON := `[0,"fls",[[2995442,"fUSD",-1,1575291961000,1575295850000,820,0,"ACTIVE",null,null,null,0.002,7,1575282446000,1575295850000,0,0,null,0,null,0]]]`
	if err := b.wsHandleData(t.Context(), []byte(pressXToJSON)); err != nil {
		t.Error(err)
	}

	pressXToJSON = `[0,"fln",[2995444,"fUSD",-1,1575298742000,1575298742000,1000,0,"ACTIVE",null,null,null,0.002,7,1575298742000,1575298742000,0,0,null,0,null,0]]`
	if err := b.wsHandleData(t.Context(), []byte(pressXToJSON)); err != nil {
		t.Error(err)
	}
}

func TestWSWalletSnapshot(t *testing.T) {
	pressXToJSON := `[0,"ws",[["exchange","SAN",19.76,0,null,null,null]]]`
	if err := b.wsHandleData(t.Context(), []byte(pressXToJSON)); err != nil {
		t.Error(err)
	}
}

func TestWSBalanceUpdate(t *testing.T) {
	const pressXToJSON = `[0,"bu",[4131.85,4131.85]]`
	if err := b.wsHandleData(t.Context(), []byte(pressXToJSON)); err != nil {
		t.Error(err)
	}
}

func TestWSMarginInfoUpdate(t *testing.T) {
	const pressXToJSON = `[0,"miu",["base",[-13.014640000000007,0,49331.70267297,49318.68803297,27]]]`
	if err := b.wsHandleData(t.Context(), []byte(pressXToJSON)); err != nil {
		t.Error(err)
	}
}

func TestWSFundingInfoUpdate(t *testing.T) {
	const pressXToJSON = `[0,"fiu",["sym","tETHUSD",[149361.09689202666,149639.26293509,830.0182168075556,895.0658432466332]]]`
	if err := b.wsHandleData(t.Context(), []byte(pressXToJSON)); err != nil {
		t.Error(err)
	}
}

func TestWSFundingTrade(t *testing.T) {
	pressXToJSON := `[0,"fte",[636854,"fUSD",1575282446000,41238905,-1000,0.002,7,null]]`
	if err := b.wsHandleData(t.Context(), []byte(pressXToJSON)); err != nil {
		t.Error(err)
	}

	pressXToJSON = `[0,"ftu",[636854,"fUSD",1575282446000,41238905,-1000,0.002,7,null]]`
	if err := b.wsHandleData(t.Context(), []byte(pressXToJSON)); err != nil {
		t.Error(err)
	}
}

func TestGetHistoricCandles(t *testing.T) {
	startTime := time.Now().Add(-time.Hour * 24)
	endTime := time.Now().Add(-time.Hour * 20)

	if _, err := b.GetHistoricCandles(t.Context(), btcusdPair, asset.Spot, kline.OneHour, startTime, endTime); err != nil {
		t.Fatal(err)
	}
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	startTime := time.Now().Add(-time.Hour * 24)
	endTime := time.Now().Add(-time.Hour * 20)

	if _, err := b.GetHistoricCandlesExtended(t.Context(), btcusdPair, asset.Spot, kline.OneHour, startTime, endTime); err != nil {
		t.Fatal(err)
	}
}

func TestFixCasing(t *testing.T) {
	ret, err := b.fixCasing(btcusdPair, asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	if ret != "tBTCUSD" {
		t.Errorf("unexpected result: %v", ret)
	}
	pair, err := currency.NewPairFromString("TBTCUSD")
	if err != nil {
		t.Fatal(err)
	}
	ret, err = b.fixCasing(pair, asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	if ret != "tBTCUSD" {
		t.Errorf("unexpected result: %v", ret)
	}
	pair, err = currency.NewPairFromString("tBTCUSD")
	if err != nil {
		t.Fatal(err)
	}
	ret, err = b.fixCasing(pair, asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	if err != nil {
		t.Fatal(err)
	}
	if ret != "tBTCUSD" {
		t.Errorf("unexpected result: %v", ret)
	}
	ret, err = b.fixCasing(btcusdPair, asset.Margin)
	if err != nil {
		t.Fatal(err)
	}
	if ret != "tBTCUSD" {
		t.Errorf("unexpected result: %v", ret)
	}
	ret, err = b.fixCasing(btcusdPair, asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	if ret != "tBTCUSD" {
		t.Errorf("unexpected result: %v", ret)
	}
	pair, err = currency.NewPairFromString("FUNETH")
	if err != nil {
		t.Fatal(err)
	}
	ret, err = b.fixCasing(pair, asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	if ret != "tFUNETH" {
		t.Errorf("unexpected result: %v", ret)
	}
	pair, err = currency.NewPairFromString("TNBUSD")
	if err != nil {
		t.Fatal(err)
	}
	ret, err = b.fixCasing(pair, asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	if ret != "tTNBUSD" {
		t.Errorf("unexpected result: %v", ret)
	}

	pair, err = currency.NewPairFromString("tTNBUSD")
	if err != nil {
		t.Fatal(err)
	}
	ret, err = b.fixCasing(pair, asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	if ret != "tTNBUSD" {
		t.Errorf("unexpected result: %v", ret)
	}
	pair, err = currency.NewPairFromStrings("fUSD", "")
	if err != nil {
		t.Fatal(err)
	}
	ret, err = b.fixCasing(pair, asset.MarginFunding)
	if err != nil {
		t.Fatal(err)
	}
	if ret != "fUSD" {
		t.Errorf("unexpected result: %v", ret)
	}
	pair, err = currency.NewPairFromStrings("USD", "")
	if err != nil {
		t.Fatal(err)
	}
	ret, err = b.fixCasing(pair, asset.MarginFunding)
	if err != nil {
		t.Fatal(err)
	}
	if ret != "fUSD" {
		t.Errorf("unexpected result: %v", ret)
	}

	pair, err = currency.NewPairFromStrings("FUSD", "")
	if err != nil {
		t.Fatal(err)
	}
	ret, err = b.fixCasing(pair, asset.MarginFunding)
	if err != nil {
		t.Fatal(err)
	}
	if ret != "fUSD" {
		t.Errorf("unexpected result: %v", ret)
	}

	_, err = b.fixCasing(currency.NewPair(currency.EMPTYCODE, currency.BTC), asset.MarginFunding)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = b.fixCasing(currency.NewPair(currency.BTC, currency.EMPTYCODE), asset.MarginFunding)
	require.NoError(t, err)

	_, err = b.fixCasing(currency.EMPTYPAIR, asset.MarginFunding)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
}

func TestFormatExchangeKlineInterval(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		interval kline.Interval
		output   string
	}{
		{
			kline.OneMin,
			"1m",
		},
		{
			kline.OneDay,
			"1D",
		},
		{
			kline.OneWeek,
			"7D",
		},
		{
			kline.OneWeek * 2,
			"14D",
		},
	} {
		t.Run(tc.interval.String(), func(t *testing.T) {
			t.Parallel()
			ret, err := b.FormatExchangeKlineInterval(tc.interval)
			require.NoError(t, err, "FormatExchangeKlineInterval must not error")
			assert.Equal(t, tc.output, ret)
		})
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	if _, err := b.GetRecentTrades(t.Context(), btcusdPair, asset.Spot); err != nil {
		t.Error(err)
	}

	currencyPair, err := currency.NewPairFromString("USD")
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.GetRecentTrades(t.Context(), currencyPair, asset.Margin)
	if err != nil {
		t.Error(err)
	}
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	if _, err := b.GetHistoricTrades(t.Context(),
		btcusdPair, asset.Spot, time.Now().Add(-time.Minute*15), time.Now()); err != nil {
		t.Error(err)
	}

	// longer term test
	if _, err := b.GetHistoricTrades(t.Context(),
		btcusdPair, asset.Spot,
		time.Now().Add(-time.Hour*100),
		time.Now().Add(-time.Hour*99)); err != nil {
		t.Error(err)
	}
}

var testOb = orderbook.Book{
	Asks: []orderbook.Level{
		{Price: 0.05005, Amount: 0.00000500},
		{Price: 0.05010, Amount: 0.00000500},
		{Price: 0.05015, Amount: 0.00000500},
		{Price: 0.05020, Amount: 0.00000500},
		{Price: 0.05025, Amount: 0.00000500},
		{Price: 0.05030, Amount: 0.00000500},
		{Price: 0.05035, Amount: 0.00000500},
		{Price: 0.05040, Amount: 0.00000500},
		{Price: 0.05045, Amount: 0.00000500},
		{Price: 0.05050, Amount: 0.00000500},
	},
	Bids: []orderbook.Level{
		{Price: 0.05000, Amount: 0.00000500},
		{Price: 0.04995, Amount: 0.00000500},
		{Price: 0.04990, Amount: 0.00000500},
		{Price: 0.04980, Amount: 0.00000500},
		{Price: 0.04975, Amount: 0.00000500},
		{Price: 0.04970, Amount: 0.00000500},
		{Price: 0.04965, Amount: 0.00000500},
		{Price: 0.04960, Amount: 0.00000500},
		{Price: 0.04955, Amount: 0.00000500},
		{Price: 0.04950, Amount: 0.00000500},
	},
}

func TestChecksum(t *testing.T) {
	err := validateCRC32(&testOb, 190468240)
	if err != nil {
		t.Fatal(err)
	}
}

func TestReOrderbyID(t *testing.T) {
	asks := []orderbook.Level{
		{ID: 4, Price: 100, Amount: 0.00000500},
		{ID: 3, Price: 100, Amount: 0.00000500},
		{ID: 2, Price: 100, Amount: 0.00000500},
		{ID: 1, Price: 100, Amount: 0.00000500},
		{ID: 5, Price: 101, Amount: 0.00000500},
		{ID: 6, Price: 102, Amount: 0.00000500},
		{ID: 8, Price: 103, Amount: 0.00000500},
		{ID: 7, Price: 103, Amount: 0.00000500},
		{ID: 9, Price: 104, Amount: 0.00000500},
		{ID: 10, Price: 105, Amount: 0.00000500},
	}
	reOrderByID(asks)

	for i := range asks {
		if asks[i].ID != int64(i+1) {
			t.Fatal("order by ID failure")
		}
	}

	bids := []orderbook.Level{
		{ID: 4, Price: 100, Amount: 0.00000500},
		{ID: 3, Price: 100, Amount: 0.00000500},
		{ID: 2, Price: 100, Amount: 0.00000500},
		{ID: 1, Price: 100, Amount: 0.00000500},
		{ID: 5, Price: 99, Amount: 0.00000500},
		{ID: 6, Price: 98, Amount: 0.00000500},
		{ID: 8, Price: 97, Amount: 0.00000500},
		{ID: 7, Price: 97, Amount: 0.00000500},
		{ID: 9, Price: 96, Amount: 0.00000500},
		{ID: 10, Price: 95, Amount: 0.00000500},
	}
	reOrderByID(bids)

	for i := range bids {
		if bids[i].ID != int64(i+1) {
			t.Fatal("order by ID failure")
		}
	}
}

func TestPopulateAcceptableMethods(t *testing.T) {
	t.Parallel()
	if acceptableMethods.loaded() {
		// we may have been loaded from another test, so reset
		acceptableMethods.m.Lock()
		acceptableMethods.a = make(map[string][]string)
		acceptableMethods.m.Unlock()
		if acceptableMethods.loaded() {
			t.Error("expected false")
		}
	}
	if err := b.PopulateAcceptableMethods(t.Context()); err != nil {
		t.Fatal(err)
	}
	if !acceptableMethods.loaded() {
		t.Error("acceptable method store should be loaded")
	}
	if methods := acceptableMethods.lookup(currency.NewCode("UST")); len(methods) == 0 {
		t.Error("USDT should have many available methods")
	}
	if methods := acceptableMethods.lookup(currency.NewCode("ASdasdasdasd")); len(methods) != 0 {
		t.Error("non-existent code should return no methods")
	}
	// since we're already loaded, this will return nil
	if err := b.PopulateAcceptableMethods(t.Context()); err != nil {
		t.Fatal(err)
	}
}

func TestGetAvailableTransferChains(t *testing.T) {
	t.Parallel()
	r, err := b.GetAvailableTransferChains(t.Context(), currency.USDT)
	if err != nil {
		t.Fatal(err)
	}
	if len(r) < 2 {
		t.Error("there should be many available USDT transfer chains")
	}
}

func TestAccetableMethodStore(t *testing.T) {
	t.Parallel()
	var a acceptableMethodStore
	if a.loaded() {
		t.Error("should be empty")
	}
	data := map[string][]string{
		"BITCOIN": {"BTC"},
		"TETHER1": {"UST"},
		"TETHER2": {"UST"},
	}
	a.load(data)
	if !a.loaded() {
		t.Error("data should be loaded")
	}
	if name := a.lookup(currency.NewCode("BTC")); len(name) != 1 && name[1] != "BITCOIN" {
		t.Error("incorrect values")
	}
	if name := a.lookup(currency.NewCode("UST")); (name[0] != "TETHER1" && name[1] != "TETHER2") &&
		(name[0] != "TETHER2" && name[1] != "TETHER1") {
		t.Errorf("incorrect values")
	}
	if name := a.lookup(currency.NewCode("PANDA_HORSE")); len(name) != 0 {
		t.Error("incorrect values")
	}
}

func TestGetSiteListConfigData(t *testing.T) {
	t.Parallel()

	_, err := b.GetSiteListConfigData(t.Context(), "")
	require.ErrorIs(t, err, errSetCannotBeEmpty)

	pairs, err := b.GetSiteListConfigData(t.Context(), bitfinexSecuritiesPairs)
	require.NoError(t, err)

	if len(pairs) == 0 {
		t.Fatal("expected pairs")
	}
}

func TestGetSiteInfoConfigData(t *testing.T) {
	t.Parallel()
	for _, assetType := range []asset.Item{asset.Spot, asset.Futures} {
		pairs, err := b.GetSiteInfoConfigData(t.Context(), assetType)
		assert.NoError(t, err)

		if len(pairs) == 0 {
			t.Errorf("GetSiteInfoConfigData returned no pairs for %s", assetType)
		}
	}
}

func TestOrderUpdate(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.OrderUpdate(t.Context(), "1234", "", "", 1, 1, 1)
	if err != nil {
		t.Error(err)
	}
}

func TestGetInactiveOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetInactiveOrders(t.Context(), "tBTCUSD")
	if err != nil {
		t.Error(err)
	}

	_, err = b.GetInactiveOrders(t.Context(), "tBTCUSD", 1, 2, 3, 4)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelMultipleOrdersV2(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.CancelMultipleOrdersV2(t.Context(), 1337, 0, 0, time.Time{}, false)
	if err != nil {
		t.Error(err)
	}
}

// TestGetErrResp unit tests the helper func getErrResp
func TestGetErrResp(t *testing.T) {
	t.Parallel()
	fixture, err := os.Open("testdata/getErrResp.json")
	if !assert.NoError(t, err, "Opening fixture should not error") {
		t.FailNow()
	}
	s := bufio.NewScanner(fixture)
	seen := 0
	for s.Scan() {
		testErr := b.getErrResp(s.Bytes())
		seen++
		switch seen {
		case 1: // no event
			assert.ErrorIs(t, testErr, common.ErrParsingWSField, "Message with no event should get correct error type")
			assert.ErrorContains(t, testErr, "'event'", "Message with no event error should contain missing field name")
			assert.ErrorContains(t, testErr, "nightjar", "Message with no event error should contain the message")
		case 2: // with {} for event
			assert.NoError(t, testErr, "Message with '{}' for event field should not error")
		case 3: // event != 'error'
			assert.NoError(t, testErr, "Message with non-'error' event field should not error")
		case 4: // event="error"
			assert.ErrorIs(t, testErr, common.ErrUnknownError, "error without a message should throw unknown error")
			assert.ErrorContains(t, testErr, "code: 0", "error without a code should throw code 0")
		case 5: // Fully formatted
			assert.ErrorContains(t, testErr, "redcoats", "message field should be in the error")
			assert.ErrorContains(t, testErr, "code: 42", "code field should be in the error")
		}
	}
	assert.NoError(t, s.Err(), "Fixture Scanner should not error")
	assert.NoError(t, fixture.Close(), "Closing the fixture file should not error")
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, b)
	for _, a := range b.GetAssetTypes(false) {
		pairs, err := b.CurrencyPairs.GetPairs(a, false)
		require.NoErrorf(t, err, "cannot get pairs for %s", a)
		require.NotEmptyf(t, pairs, "no pairs for %s", a)
		resp, err := b.GetCurrencyTradeURL(t.Context(), a, pairs[0])
		require.NoError(t, err)
		assert.NotEmpty(t, resp)
	}
}
