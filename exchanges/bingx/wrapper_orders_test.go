package bingx

import (
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"math"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const (
	wrapperHedgeModeReply  = `{"code":0,"data":{"dualSidePosition":"true"}}`
	wrapperOneWayModeReply = `{"code":0,"data":{"dualSidePosition":"false"}}`
)

func TestPositionModeCacheGet(t *testing.T) {
	t.Parallel()
	var c positionModeCache
	_, ok, generation := c.get("a")
	assert.False(t, ok, "get should report an account without a cached mode")
	assert.Zero(t, generation, "get should start an account at the first generation")
	c.set("a", generation, true)
	hedgeMode, ok, _ := c.get("a")
	require.True(t, ok, "get must return the mode set")
	assert.True(t, hedgeMode, "get should return the hedge mode set")
	c.forget("a")
	_, _, generation = c.get("a")
	assert.Equal(t, uint64(1), generation, "get should return the generation a lookup started now keeps its mode at")
}

func TestPositionModeCacheSet(t *testing.T) {
	t.Parallel()
	var c positionModeCache
	_, _, generation := c.get("a")
	c.forget("a")
	c.set("a", generation, false)
	_, ok, _ := c.get("a")
	assert.False(t, ok, "set should drop a mode looked up before the account's mode was forgotten")
	_, _, generation = c.get("a")
	c.set("a", generation, false)
	hedgeMode, ok, _ := c.get("a")
	assert.True(t, ok, "set should keep a mode looked up since the account's mode was forgotten")
	assert.False(t, hedgeMode, "set should keep the one-way mode")
}

func TestPositionModeCacheForget(t *testing.T) {
	t.Parallel()
	var c positionModeCache
	c.forget("a")
	_, _, generation := c.get("a")
	c.set("a", generation, true)
	c.set("b", 0, true)
	c.forget("a")
	_, ok, generation := c.get("a")
	assert.False(t, ok, "forget should drop the account's mode")
	assert.Equal(t, uint64(2), generation, "forget should advance the account's generation")
	_, ok, _ = c.get("b")
	assert.True(t, ok, "forget should keep other accounts' modes")
}

// testOrderModificationHolders returns the count of modifications holding or waiting for the lock of each order
func testOrderModificationHolders(l *orderModificationLocks) map[orderModificationKey]int {
	l.mu.Lock()
	defer l.mu.Unlock()
	holders := make(map[orderModificationKey]int, len(l.locks))
	for k, entry := range l.locks {
		holders[k] = entry.holders
	}
	return holders
}

func TestOrderModificationLocksLock(t *testing.T) {
	t.Parallel()
	held := orderModificationKey{account: "a", asset: asset.Spot, orderID: 11}
	t.Run("same order", func(t *testing.T) {
		t.Parallel()
		var l orderModificationLocks
		unlock, err := l.lock(t.Context(), held)
		require.NoError(t, err, "lock must not error")
		type result struct {
			unlock func()
			err    error
		}
		acquired := make(chan result, 1)
		go func() {
			second, err := l.lock(t.Context(), held)
			acquired <- result{second, err}
		}()
		require.Eventually(t, func() bool { return testOrderModificationHolders(&l)[held] == 2 }, 5*time.Second, time.Millisecond, "lock must count a modification waiting for the order's lock")
		select {
		case <-acquired:
			require.FailNow(t, "lock must wait while another modification holds the order's lock")
		default:
		}
		unlock()
		var second result
		select {
		case second = <-acquired:
		case <-time.After(10 * time.Second):
			require.FailNow(t, "lock must return once the order's lock is released")
		}
		require.NoError(t, second.err, "lock must not error once the order's lock is released")
		assert.Equal(t, map[orderModificationKey]int{held: 1}, testOrderModificationHolders(&l), "lock should count only the modification left holding the order's lock")
		second.unlock()
		assert.Empty(t, testOrderModificationHolders(&l), "lock should drop an order's entry once its last holder releases it")
	})

	for _, tc := range []struct {
		name  string
		other orderModificationKey
	}{
		{name: "another account", other: orderModificationKey{account: "b", asset: asset.Spot, orderID: 11}},
		{name: "another asset", other: orderModificationKey{account: "a", asset: asset.USDTMarginedFutures, orderID: 11}},
		{name: "another order ID", other: orderModificationKey{account: "a", asset: asset.Spot, orderID: 12}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var l orderModificationLocks
			unlock, err := l.lock(t.Context(), held)
			require.NoError(t, err, "lock must not error")
			defer unlock()
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			otherUnlock, err := l.lock(ctx, tc.other)
			require.NoError(t, err, "lock must not wait for another order's lock")
			assert.Equal(t, map[orderModificationKey]int{held: 1, tc.other: 1}, testOrderModificationHolders(&l), "lock should keep each order's lock apart")
			otherUnlock()
			assert.Equal(t, map[orderModificationKey]int{held: 1}, testOrderModificationHolders(&l), "lock should drop the other order's entry once it is released")
		})
	}

	t.Run("context ends while waiting", func(t *testing.T) {
		t.Parallel()
		var l orderModificationLocks
		unlock, err := l.lock(t.Context(), held)
		require.NoError(t, err, "lock must not error")
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		_, err = l.lock(ctx, held)
		assert.ErrorIs(t, err, context.Canceled, "lock should return the context's error when it ends while waiting")
		assert.Equal(t, map[orderModificationKey]int{held: 1}, testOrderModificationHolders(&l), "lock should stop counting a modification that gave up waiting")
		unlock()
		assert.Empty(t, testOrderModificationHolders(&l), "lock should drop an order's entry once its last holder releases it")
	})
}

func TestCredentialsContext(t *testing.T) {
	t.Parallel()
	ex, _ := wrapperTestServer(t, map[string][]string{})
	ctx, err := ex.credentialsContext(t.Context())
	require.NoError(t, err, "credentialsContext must not error")
	ex.SetCredentials(&accounts.Credentials{Key: "second-key", Secret: "second-secret"})
	creds, err := ex.GetCredentials(ctx)
	require.NoError(t, err, "GetCredentials must not error")
	assert.Equal(t, testAPIKey, creds.Key, "credentialsContext should keep the credentials ctx named although the default credentials change")
	contextCreds := &accounts.Credentials{Key: "context-key", Secret: "context-secret"}
	ctx, err = ex.credentialsContext(accounts.DeployCredentialsToContext(t.Context(), contextCreds))
	require.NoError(t, err, "credentialsContext must not error for context credentials")
	creds, err = ex.GetCredentials(ctx)
	require.NoError(t, err, "GetCredentials must not error")
	assert.Equal(t, contextCreds.Key, creds.Key, "credentialsContext should keep the context's own credentials")
	ex.SetCredentials(&accounts.Credentials{})
	_, err = ex.credentialsContext(t.Context())
	assert.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "credentialsContext should return the credentials error")
}

func TestProductAccount(t *testing.T) {
	t.Parallel()
	ex, _ := wrapperTestServer(t, map[string][]string{})
	for _, ePath := range []exchange.URL{exchange.RestUSDTMargined, exchange.RestCoinMargined} {
		endpoint, err := ex.API.Endpoints.GetURL(ePath)
		require.NoErrorf(t, err, "GetURL must not error for %s", ePath)
		account, err := ex.productAccount(t.Context(), ePath)
		require.NoErrorf(t, err, "productAccount must not error for %s", ePath)
		assert.Equalf(t, endpoint+" "+testAPIKey, account, "productAccount should key the %s account by endpoint and API key", ePath)
	}
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestCoinMargined.String(), "https://coin-margined.example"), "SetRunningURL must not error")
	account, err := ex.productAccount(t.Context(), exchange.RestCoinMargined)
	require.NoError(t, err, "productAccount must not error")
	assert.Equal(t, "https://coin-margined.example "+testAPIKey, account, "productAccount should key the account by the product's own endpoint")
	ctx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "other-key", Secret: "other-secret"})
	account, err = ex.productAccount(ctx, exchange.RestCoinMargined)
	require.NoError(t, err, "productAccount must not error")
	assert.Equal(t, "https://coin-margined.example other-key", account, "productAccount should key the context's account by its own API key")
	withoutEndpoints, _ := wrapperTestServer(t, map[string][]string{})
	withoutEndpoints.API.Endpoints = withoutEndpoints.NewEndpoints()
	_, err = withoutEndpoints.productAccount(t.Context(), exchange.RestUSDTMargined)
	assert.ErrorIs(t, err, exchange.ErrEndpointPathNotFound, "productAccount should return the endpoint lookup's error")
	ex.SetCredentials(&accounts.Credentials{})
	_, err = ex.productAccount(t.Context(), exchange.RestUSDTMargined)
	assert.Error(t, err, "productAccount should return the credentials error")
}

func TestSwapHedgeMode(t *testing.T) {
	t.Parallel()
	ex, requests := wrapperTestServer(t, map[string][]string{
		"GET /openApi/swap/v1/positionSide/dual": {wrapperHedgeModeReply, wrapperOneWayModeReply, `{"code":100500,"msg":"busy"}`},
	})
	for range 2 {
		hedgeMode, err := ex.swapHedgeMode(t.Context())
		require.NoError(t, err, "swapHedgeMode must not error")
		assert.True(t, hedgeMode, "swapHedgeMode should return hedge mode")
	}
	assert.Len(t, requests(), 1, "swapHedgeMode should look the mode up once and then keep it")
	ex.forgetSwapPositionMode(t.Context())
	hedgeMode, err := ex.swapHedgeMode(t.Context())
	require.NoError(t, err, "swapHedgeMode must not error")
	assert.False(t, hedgeMode, "swapHedgeMode should look the mode up again once forgotten")
	ctx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "other-key", Secret: testAPISecret})
	_, err = ex.swapHedgeMode(ctx)
	assert.ErrorIs(t, err, errAPIResponse, "swapHedgeMode should return another account's lookup error")
	ex.SetCredentials(&accounts.Credentials{})
	_, err = ex.swapHedgeMode(t.Context())
	assert.Error(t, err, "swapHedgeMode should return the credentials error")
	assert.Len(t, requests(), 3, "swapHedgeMode should look up each account's mode")
}

func TestForgetSwapPositionMode(t *testing.T) {
	t.Parallel()
	ex, requests := wrapperTestServer(t, map[string][]string{
		"GET /openApi/swap/v1/positionSide/dual": {wrapperHedgeModeReply, wrapperHedgeModeReply},
	})
	_, err := ex.swapHedgeMode(t.Context())
	require.NoError(t, err, "swapHedgeMode must not error")
	ex.forgetSwapPositionMode(t.Context())
	_, err = ex.swapHedgeMode(t.Context())
	require.NoError(t, err, "swapHedgeMode must not error")
	assert.Len(t, requests(), 2, "forgetSwapPositionMode should make the next order look the mode up")
	ex.SetCredentials(&accounts.Credentials{})
	assert.NotPanics(t, func() { ex.forgetSwapPositionMode(t.Context()) }, "forgetSwapPositionMode should ignore a context without credentials")
}

func TestCoinMarginedHedgeMode(t *testing.T) {
	t.Parallel()
	ex, requests := wrapperTestServer(t, map[string][]string{})
	hedgeMode, err := ex.coinMarginedHedgeMode(t.Context())
	require.NoError(t, err, "coinMarginedHedgeMode must not error")
	assert.True(t, hedgeMode, "coinMarginedHedgeMode should take an account to be in hedge mode until BingX refuses an order")
	ex.recordCoinMarginedPositionMode(t.Context(), false)
	hedgeMode, err = ex.coinMarginedHedgeMode(t.Context())
	require.NoError(t, err, "coinMarginedHedgeMode must not error")
	assert.False(t, hedgeMode, "coinMarginedHedgeMode should return the mode recorded for the account")
	hedgeMode, err = ex.coinMarginedHedgeMode(accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "other-key", Secret: "other-secret"}))
	require.NoError(t, err, "coinMarginedHedgeMode must not error")
	assert.True(t, hedgeMode, "coinMarginedHedgeMode should keep each account's mode apart")
	assert.Empty(t, requests(), "coinMarginedHedgeMode should not query BingX, which offers no coin-M position mode query")
	ex.SetCredentials(&accounts.Credentials{})
	_, err = ex.coinMarginedHedgeMode(t.Context())
	assert.Error(t, err, "coinMarginedHedgeMode should return the credentials error")
}

func TestRecordCoinMarginedPositionMode(t *testing.T) {
	t.Parallel()
	ex, _ := wrapperTestServer(t, map[string][]string{})
	for _, hedgeMode := range []bool{false, true, false} {
		ex.recordCoinMarginedPositionMode(t.Context(), hedgeMode)
		recorded, err := ex.coinMarginedHedgeMode(t.Context())
		require.NoError(t, err, "coinMarginedHedgeMode must not error")
		assert.Equalf(t, hedgeMode, recorded, "recordCoinMarginedPositionMode should record hedge mode %t", hedgeMode)
	}
	account, err := ex.productAccount(t.Context(), exchange.RestUSDTMargined)
	require.NoError(t, err, "productAccount must not error")
	_, cached, _ := ex.positionModes.get(account)
	assert.False(t, cached, "recordCoinMarginedPositionMode should leave the USDT-M mode alone")
	ex.SetCredentials(&accounts.Credentials{})
	assert.NotPanics(t, func() { ex.recordCoinMarginedPositionMode(t.Context(), false) }, "recordCoinMarginedPositionMode should ignore a context without credentials")
}

func TestPerpetualPositionSide(t *testing.T) {
	t.Parallel()
	hedge, _ := wrapperTestServer(t, map[string][]string{"GET /openApi/swap/v1/positionSide/dual": {wrapperHedgeModeReply}})
	oneWay, _ := wrapperTestServer(t, map[string][]string{"GET /openApi/swap/v1/positionSide/dual": {wrapperOneWayModeReply}})
	coinMarginedOneWay, _ := wrapperTestServer(t, map[string][]string{})
	coinMarginedOneWay.recordCoinMarginedPositionMode(t.Context(), false)
	for _, tc := range []struct {
		ex            *Exchange
		a             asset.Item
		side          order.Side
		reduceOnly    bool
		expSide       PositionSide
		expReduceOnly bool
	}{
		{hedge, asset.USDTMarginedFutures, order.Buy, false, PositionSideLong, false},
		{hedge, asset.USDTMarginedFutures, order.Short, false, PositionSideShort, false},
		{hedge, asset.USDTMarginedFutures, order.Sell, true, PositionSideLong, false},
		{hedge, asset.USDTMarginedFutures, order.Long, true, PositionSideShort, false},
		{oneWay, asset.USDTMarginedFutures, order.Buy, false, PositionSideBoth, false},
		{oneWay, asset.USDTMarginedFutures, order.Sell, true, PositionSideBoth, true},
		{oneWay, asset.CoinMarginedFutures, order.Buy, false, PositionSideLong, false},
		{oneWay, asset.CoinMarginedFutures, order.Buy, true, PositionSideShort, false},
		{coinMarginedOneWay, asset.CoinMarginedFutures, order.Buy, false, PositionSideBoth, false},
		{coinMarginedOneWay, asset.CoinMarginedFutures, order.Sell, true, PositionSideBoth, true},
	} {
		side, reduceOnly, err := tc.ex.perpetualPositionSide(t.Context(), tc.a, tc.side, tc.reduceOnly)
		require.NoErrorf(t, err, "perpetualPositionSide must not error for %s %s", tc.a, tc.side)
		assert.Equalf(t, tc.expSide, side, "perpetualPositionSide should place a %s %s order with reduce only %t on %s", tc.a, tc.side, tc.reduceOnly, tc.expSide)
		assert.Equalf(t, tc.expReduceOnly, reduceOnly, "perpetualPositionSide should send reduce only %t for a %s %s order", tc.expReduceOnly, tc.a, tc.side)
	}
	failing, _ := wrapperTestServer(t, map[string][]string{"GET /openApi/swap/v1/positionSide/dual": {`{"code":100500,"msg":"busy"}`}})
	_, _, err := failing.perpetualPositionSide(t.Context(), asset.USDTMarginedFutures, order.Buy, false)
	assert.ErrorIs(t, err, errAPIResponse, "perpetualPositionSide should return the position mode lookup's error")
	failing.SetCredentials(&accounts.Credentials{})
	_, _, err = failing.perpetualPositionSide(t.Context(), asset.CoinMarginedFutures, order.Buy, false)
	assert.Error(t, err, "perpetualPositionSide should return the coin-M credentials error")
}

func TestHedgePositionSide(t *testing.T) {
	t.Parallel()
	assert.Equal(t, PositionSideLong, hedgePositionSide(order.Buy, false), "hedgePositionSide should open a long position with a buy")
	assert.Equal(t, PositionSideShort, hedgePositionSide(order.Sell, false), "hedgePositionSide should open a short position with a sell")
	assert.Equal(t, PositionSideShort, hedgePositionSide(order.Buy, true), "hedgePositionSide should close a short position with a reduce only buy")
	assert.Equal(t, PositionSideLong, hedgePositionSide(order.Sell, true), "hedgePositionSide should close a long position with a reduce only sell")
}

func TestIsPositionModeRejection(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		err error
		exp bool
	}{
		{&APIError{Code: 109400, Message: "positionSide: LONG is invalid"}, true},
		{&APIError{Code: 109400, Message: "Reduce Only order is not allowed in hedge mode"}, true},
		{&APIError{Code: 109400, Message: "the position side does not match the position mode"}, true},
		{fmt.Errorf("error placing order: %w", &APIError{Code: 109400, Message: "reduceOnly is invalid"}), true},
		{&APIError{Code: 109400, Message: "Invalid parameters"}, false},
		{&APIError{Code: 101204, Message: "positionSide insufficient margin"}, false},
		{errors.New("positionSide"), false},
		{nil, false},
	} {
		assert.Equalf(t, tc.exp, isPositionModeRejection(tc.err), "isPositionModeRejection should report %v as %t", tc.err, tc.exp)
	}
}

func TestParseOrderID(t *testing.T) {
	t.Parallel()
	id, err := parseOrderID("")
	require.NoError(t, err, "parseOrderID must not error for an empty ID")
	assert.Zero(t, id, "parseOrderID should read an empty ID as none")
	id, err = parseOrderID("1736012449498123456")
	require.NoError(t, err, "parseOrderID must not error")
	assert.Equal(t, uint64(1736012449498123456), id, "parseOrderID should read the whole ID")
	for _, orderID := range []string{"abc", "-1", "1.5"} {
		_, err = parseOrderID(orderID)
		assert.ErrorIsf(t, err, errInvalidOrderID, "parseOrderID should reject %q", orderID)
	}
}

func TestRemainingAmount(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 0.2, remainingAmount(0.3, 0.1), "remainingAmount should subtract in decimal arithmetic")
	assert.Equal(t, 4.8, remainingAmount(5.9, 1.1), "remainingAmount should subtract in decimal arithmetic")
	assert.Zero(t, remainingAmount(1, 2), "remainingAmount should not go below zero")
}

func TestParseLeverage(t *testing.T) {
	t.Parallel()
	for leverage, exp := range map[string]float64{"20X": 20, "15x": 15, "3": 3, "": 0} {
		got, err := parseLeverage(leverage)
		require.NoErrorf(t, err, "parseLeverage must not error for %q", leverage)
		assert.Equalf(t, exp, got, "parseLeverage should read %q", leverage)
	}
	_, err := parseLeverage("highX")
	assert.ErrorIs(t, err, errInvalidOrderLeverage, "parseLeverage should reject text that is not a number")
}

func TestLogOrderIssue(t *testing.T) {
	t.Parallel()
	assert.NotPanics(t, func() { e.logOrderIssue(asset.Spot, "1", errInvalidOrderID) }, "logOrderIssue should log the issue")
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	_, err := e.SubmitOrder(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrSubmissionIsNil, "SubmitOrder must reject a nil submission")
	submission := &order.Submit{Exchange: e.Name, Pair: currency.NewBTCUSDT(), AssetType: asset.Spot, Side: order.Buy, Type: order.Market, Amount: 1}
	_, err = e.SubmitOrder(t.Context(), submission)
	require.ErrorIs(t, err, order.ErrAmountMustBeSet, "SubmitOrder must take a spot market buy in the quote currency")
	submission = &order.Submit{Exchange: e.Name, Pair: currency.NewBTCUSDT(), AssetType: asset.USDTMarginedFutures, Side: order.Buy, Type: order.Market, Amount: 1, Leverage: 5}
	_, err = e.SubmitOrder(t.Context(), submission)
	require.ErrorIs(t, err, order.ErrSubmitLeverageNotSupported, "SubmitOrder must refuse a leverage")
	submission = &order.Submit{Exchange: e.Name, Pair: currency.NewBTCUSDT(), AssetType: asset.Futures, Side: order.Buy, Type: order.Market, Amount: 1}
	_, err = e.SubmitOrder(t.Context(), submission)
	require.ErrorIs(t, err, asset.ErrNotSupported, "SubmitOrder must reject an unsupported asset")

	t.Run("recorded", func(t *testing.T) {
		t.Parallel()
		ex := newWrapperTestExchange(t)
		wrapperSkipIfCannotManipulateOrders(t, ex)
		if !mockTests {
			t.Skip("the recorded orders are documented examples, which are not placed live")
		}
		for _, tc := range []struct {
			submission *order.Submit
			orderID    string
			status     order.Status
			remaining  float64
		}{
			{&order.Submit{Pair: currency.NewPairWithDelimiter("NEMS", "USDT", currency.DashDelimiter), AssetType: asset.Spot, Side: order.Sell, Type: order.Limit, Price: 0.1557, Amount: 115, TimeInForce: order.GoodTillCancel}, "1735963671248581000", order.Pending, 115},
			{&order.Submit{Pair: currency.NewBTCUSDT(), AssetType: asset.USDTMarginedFutures, Side: order.Buy, Type: order.Limit, Price: 50000, Amount: 0.001, TimeInForce: order.GoodTillCancel}, "1735950529123455000", order.New, 0.001},
			// Coin-M placement replies report no execution
			{&order.Submit{Pair: currency.NewPairWithDelimiter("BTC", "USD", currency.DashDelimiter), AssetType: asset.CoinMarginedFutures, Side: order.Buy, Type: order.Limit, Price: 10000, Amount: 1, TimeInForce: order.PostOnly}, "1802706634360750000", order.New, 0},
		} {
			tc.submission.Exchange = ex.Name
			resp, err := ex.SubmitOrder(t.Context(), tc.submission)
			require.NoErrorf(t, err, "SubmitOrder must not error for %s", tc.submission.AssetType)
			assert.Equalf(t, tc.orderID, resp.OrderID, "SubmitOrder should return the %s order's ID", tc.submission.AssetType)
			assert.Equalf(t, tc.status, resp.Status, "SubmitOrder should return the %s order's status", tc.submission.AssetType)
			assert.Equalf(t, tc.remaining, resp.RemainingAmount, "SubmitOrder should report the %s order's unfilled amount", tc.submission.AssetType)
			if tc.submission.AssetType == asset.Spot {
				assert.Equal(t, time.UnixMilli(1702720966528), resp.Date, "SubmitOrder should date the spot order by its placement")
			}
		}
	})

	t.Run("resolved type", func(t *testing.T) {
		t.Parallel()
		ex, requests := wrapperTestServer(t, map[string][]string{
			"POST /openApi/spot/v1/trade/order": {`{"code":0,"data":{"symbol":"BTC-USDT","orderId":1,"transactTime":1700000000000,"origQty":"1","executedQty":"0","status":"NEW","type":"TAKE_STOP_LIMIT","side":"BUY"}}`},
		})
		resp, err := ex.SubmitOrder(t.Context(), &order.Submit{Exchange: ex.Name, Pair: currency.NewBTCUSDT(), AssetType: asset.Spot, Side: order.Buy, Type: order.Stop, Price: 10, TriggerPrice: 9, Amount: 1})
		require.NoError(t, err, "SubmitOrder must not error")
		assert.Equal(t, order.StopLimit, resp.Type, "SubmitOrder should report a priced stop order as a stop limit order")
		assert.Equal(t, url.Values{"symbol": {"BTC-USDT"}, "side": {"BUY"}, "type": {"TAKE_STOP_LIMIT"}, "price": {"10"}, "stopPrice": {"9"}, "quantity": {"1"}}, requests()[0].params, "SubmitOrder should place a priced stop order as a TAKE_STOP_LIMIT order")
	})

	t.Run("credentials changing during the submission", func(t *testing.T) {
		t.Parallel()
		const positionModeRefusal = `{"code":109400,"msg":"In the one-way mode, the 'PositionSide' field can only be set to BOTH."}`
		for _, tc := range []struct {
			name       string
			submission order.Submit
			routes     map[string][]string
			requests   int
		}{
			{
				name:       "USDT-M",
				submission: order.Submit{Pair: currency.NewBTCUSDT(), AssetType: asset.USDTMarginedFutures, Side: order.Buy, Type: order.Market, Amount: 1},
				routes: map[string][]string{
					"GET /openApi/swap/v1/positionSide/dual": {wrapperCredentialsChange + wrapperHedgeModeReply, wrapperOneWayModeReply},
					"POST /openApi/swap/v2/trade/order":      {wrapperCredentialsChange + positionModeRefusal, `{"code":0,"data":{"order":{"symbol":"BTC-USDT","orderId":7,"side":"BUY","positionSide":"BOTH","type":"MARKET"}}}`},
				},
				requests: 4,
			},
			{
				name:       "coin-M",
				submission: order.Submit{Pair: currency.NewBTCUSD(), AssetType: asset.CoinMarginedFutures, Side: order.Buy, Type: order.Market, Amount: 1},
				routes: map[string][]string{
					"POST /openApi/cswap/v2/trade/order": {wrapperCredentialsChange + positionModeRefusal, `{"code":0,"data":{"order":{"orderId":"8","symbol":"BTC-USD","side":"BUY","type":"MARKET","quantity":1}}}`},
				},
				requests: 2,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex, requests := wrapperTestServer(t, tc.routes)
				s := tc.submission
				s.Exchange = ex.Name
				_, err := ex.SubmitOrder(t.Context(), &s)
				require.NoError(t, err, "SubmitOrder must place the order once more after a refusal for its position mode")
				got := requests()
				require.Lenf(t, got, tc.requests, "SubmitOrder must send each %s request", tc.name)
				for _, r := range got {
					assert.Equalf(t, testAPIKey, r.apiKey, "SubmitOrder should send %s %s for the account it started with", r.method, r.path)
				}
			})
		}
	})

	t.Run("margin types and reduce only", func(t *testing.T) {
		t.Parallel()
		replies := map[asset.Item]map[string][]string{
			asset.Spot: {"POST /openApi/spot/v1/trade/order": {`{"code":0,"data":{"symbol":"BTC-USDT","orderId":1,"transactTime":1700000000000,"origQty":"1","executedQty":"0","status":"NEW","type":"LIMIT","side":"BUY"}}`}},
			asset.USDTMarginedFutures: {
				"GET /openApi/swap/v1/positionSide/dual": {wrapperHedgeModeReply},
				"POST /openApi/swap/v2/trade/order":      {`{"code":0,"data":{"order":{"symbol":"BTC-USDT","orderId":2,"side":"BUY","positionSide":"LONG","type":"LIMIT","status":"NEW"}}}`},
			},
			asset.CoinMarginedFutures: {"POST /openApi/cswap/v2/trade/order": {`{"code":0,"data":{"order":{"orderId":"3","symbol":"BTC-USD","side":"BUY","type":"LIMIT","quantity":1}}}`}},
		}
		pairs := map[asset.Item]currency.Pair{asset.Spot: currency.NewBTCUSDT(), asset.USDTMarginedFutures: currency.NewBTCUSDT(), asset.CoinMarginedFutures: currency.NewBTCUSD()}
		for _, a := range []asset.Item{asset.Spot, asset.USDTMarginedFutures, asset.CoinMarginedFutures} {
			for _, mt := range []margin.Type{margin.Unset, margin.Isolated, margin.Multi, margin.NoMargin, margin.SpotIsolated, margin.Unknown} {
				for _, reduceOnly := range []bool{false, true} {
					for _, orderType := range []order.Type{order.Limit, order.Market, order.StopLimit} {
						for _, side := range []order.Side{order.Buy, order.Sell} {
							t.Run(fmt.Sprintf("%s %s %s %s reduce only %t", a, side, orderType, mt, reduceOnly), func(t *testing.T) {
								t.Parallel()
								var exp error
								switch {
								case mt != margin.Unset && (a != asset.Spot || mt != margin.NoMargin):
									exp = margin.ErrMarginTypeUnsupported
								case reduceOnly && a == asset.Spot:
									exp = errSpotReduceOnly
								}
								routes := make(map[string][]string)
								if exp == nil {
									maps.Copy(routes, replies[a])
								}
								ex, requests := wrapperTestServer(t, routes)
								s := &order.Submit{Exchange: ex.Name, Pair: pairs[a], AssetType: a, Side: side, Type: orderType, Price: 10, TriggerPrice: 9, Amount: 1, MarginType: mt, ReduceOnly: reduceOnly}
								if orderType == order.Market {
									s.Price, s.TriggerPrice = 0, 0
									if a == asset.Spot && side == order.Buy {
										s.Amount, s.QuoteAmount = 0, 10
									}
								}
								_, err := ex.SubmitOrder(t.Context(), s)
								if exp != nil {
									assert.ErrorIs(t, err, exp, "SubmitOrder should refuse what the order cannot carry")
									assert.Empty(t, requests(), "SubmitOrder should send nothing for an order it refuses")
									return
								}
								require.NoError(t, err, "SubmitOrder must place the order")
								assert.NotEmpty(t, requests(), "SubmitOrder should place the order")
							})
						}
					}
				}
			}
		}
	})

	t.Run("perpetual order without credentials", func(t *testing.T) {
		t.Parallel()
		ex, requests := wrapperTestServer(t, map[string][]string{})
		ex.SetCredentials(&accounts.Credentials{})
		_, err := ex.SubmitOrder(t.Context(), &order.Submit{Exchange: ex.Name, Pair: currency.NewBTCUSDT(), AssetType: asset.USDTMarginedFutures, Side: order.Buy, Type: order.Market, Amount: 1})
		assert.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "SubmitOrder should return the credentials error")
		assert.Empty(t, requests(), "SubmitOrder should send nothing without credentials")
	})
}

func TestSpotOrderRequest(t *testing.T) {
	t.Parallel()
	pair := currency.NewBTCUSDT()
	for _, tc := range []struct {
		name string
		s    *order.Submit
		exp  *PlaceSpotOrderRequest
	}{
		{"market buy", &order.Submit{Pair: pair, Side: order.Buy, Type: order.Market, Amount: 1, QuoteAmount: 100}, &PlaceSpotOrderRequest{Symbol: pair, Side: SideBuy, Type: OrderTypeMarket, QuoteOrderQuantity: 100}},
		{"market sell", &order.Submit{Pair: pair, Side: order.Sell, Type: order.Market, Amount: 2}, &PlaceSpotOrderRequest{Symbol: pair, Side: SideSell, Type: OrderTypeMarket, Quantity: 2}},
		{"limit", &order.Submit{Pair: pair, Side: order.Buy, Type: order.Limit, Price: 10, Amount: 2, ClientOrderID: "c1", TimeInForce: order.ImmediateOrCancel}, &PlaceSpotOrderRequest{Symbol: pair, Side: SideBuy, Type: OrderTypeLimit, Price: 10, Quantity: 2, NewClientOrderID: "c1", TimeInForce: TimeInForceImmediateOrCancel}},
		{"limit in quote", &order.Submit{Pair: pair, Side: order.Buy, Type: order.Limit, Price: 10, QuoteAmount: 50}, &PlaceSpotOrderRequest{Symbol: pair, Side: SideBuy, Type: OrderTypeLimit, Price: 10, QuoteOrderQuantity: 50}},
		{"stop limit", &order.Submit{Pair: pair, Side: order.Sell, Type: order.StopLimit, Price: 8, TriggerPrice: 9, Amount: 1}, &PlaceSpotOrderRequest{Symbol: pair, Side: SideSell, Type: OrderTypeTakeStopLimit, Price: 8, StopPrice: 9, Quantity: 1}},
		{"stop market", &order.Submit{Pair: pair, Side: order.Sell, Type: order.StopMarket, Price: 8, TriggerPrice: 9, Amount: 1}, &PlaceSpotOrderRequest{Symbol: pair, Side: SideSell, Type: OrderTypeTakeStopMarket, StopPrice: 9, Quantity: 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req, err := spotOrderRequest(tc.s)
			require.NoError(t, err, "spotOrderRequest must not error")
			assert.Equal(t, tc.exp, req, "spotOrderRequest should build the order BingX takes")
		})
	}
	for _, tc := range []struct {
		name string
		s    *order.Submit
		err  error
	}{
		{"side", &order.Submit{Pair: pair, Side: order.AnySide, Type: order.Limit, Price: 1, Amount: 1}, order.ErrSideIsInvalid},
		{"type", &order.Submit{Pair: pair, Side: order.Buy, Type: order.TakeProfit, Price: 1, Amount: 1}, order.ErrUnsupportedOrderType},
		{"time in force", &order.Submit{Pair: pair, Side: order.Buy, Type: order.Limit, Price: 1, Amount: 1, TimeInForce: order.GoodTillDay}, order.ErrUnsupportedTimeInForce},
		{"price", &order.Submit{Pair: pair, Side: order.Buy, Type: order.StopLimit, TriggerPrice: 1, Amount: 1}, order.ErrPriceMustBeSetIfLimitOrder},
		{"trigger", &order.Submit{Pair: pair, Side: order.Buy, Type: order.StopMarket, Amount: 1}, errTriggerPriceRequired},
		{"amount", &order.Submit{Pair: pair, Side: order.Buy, Type: order.StopMarket, TriggerPrice: 1, QuoteAmount: 1}, order.ErrAmountIsInvalid},
	} {
		_, err := spotOrderRequest(tc.s)
		assert.ErrorIsf(t, err, tc.err, "spotOrderRequest should reject an order with an invalid %s", tc.name)
	}
}

func TestSubmitSpotOrder(t *testing.T) {
	t.Parallel()
	ex, requests := wrapperTestServer(t, map[string][]string{
		"POST /openApi/spot/v1/trade/order": {
			`{"code":0,"data":{"symbol":"BTC-USDT","orderId":11,"transactTime":1700000000000,"origQty":"2","executedQty":"2","cummulativeQuoteQty":"5","status":"FILLED","type":"MARKET","side":"SELL","clientOrderID":"c11"}}`,
			`{"code":0,"data":{"symbol":"BTC-USDT","orderId":12,"transactTime":1700000000000,"origQty":"2","executedQty":"0","status":"ODD","type":"LIMIT","side":"BUY"}}`,
			`{"code":100500,"msg":"busy"}`,
		},
	})
	resp, err := ex.submitSpotOrder(t.Context(), &order.Submit{Exchange: ex.Name, Pair: currency.NewBTCUSDT(), AssetType: asset.Spot, Side: order.Sell, Type: order.Market, Amount: 2, ClientOrderID: "c11"})
	require.NoError(t, err, "submitSpotOrder must not error")
	assert.Equal(t, "11", resp.OrderID, "submitSpotOrder should return the order ID")
	assert.Equal(t, "c11", resp.ClientOrderID, "submitSpotOrder should return the client order ID")
	assert.Equal(t, order.New, resp.Status, "submitSpotOrder should report an order filled at placement as new")
	assert.Equal(t, time.UnixMilli(1700000000000), resp.Date, "submitSpotOrder should date the order by its placement")
	assert.Equal(t, resp.Date, resp.LastUpdated, "submitSpotOrder should report the order updated at its placement")
	assert.Equal(t, 5.0, resp.Cost, "submitSpotOrder should report the quote spent")
	assert.Equal(t, 2.5, resp.AverageExecutedPrice, "submitSpotOrder should report the average price")
	assert.Zero(t, resp.RemainingAmount, "submitSpotOrder should report nothing left of a filled order")
	assert.Equal(t, url.Values{"symbol": {"BTC-USDT"}, "side": {"SELL"}, "type": {"MARKET"}, "quantity": {"2"}, "newClientOrderId": {"c11"}}, requests()[0].params, "submitSpotOrder should place the order")

	resp, err = ex.submitSpotOrder(t.Context(), &order.Submit{Exchange: ex.Name, Pair: currency.NewBTCUSDT(), AssetType: asset.Spot, Side: order.Buy, Type: order.Limit, Price: 1, Amount: 2})
	require.NoError(t, err, "submitSpotOrder must not error for a status it cannot convert")
	assert.Equal(t, order.New, resp.Status, "submitSpotOrder should keep the submission's status for one it cannot convert")
	assert.Equal(t, 2.0, resp.RemainingAmount, "submitSpotOrder should report the unfilled amount")

	_, err = ex.submitSpotOrder(t.Context(), &order.Submit{Exchange: ex.Name, Pair: currency.NewBTCUSDT(), AssetType: asset.Spot, Side: order.Buy, Type: order.Limit, Price: 1, Amount: 2})
	assert.ErrorIs(t, err, errAPIResponse, "submitSpotOrder should return the request's error")
	_, err = ex.submitSpotOrder(t.Context(), &order.Submit{Pair: currency.NewBTCUSDT(), Side: order.AnySide, Type: order.Limit, Price: 1, Amount: 2})
	assert.ErrorIs(t, err, order.ErrSideIsInvalid, "submitSpotOrder should return the order's error")

	t.Run("market order statuses", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name, fields string
			exp          order.Status
		}{
			{"no status", `,"executedQty":"0"`, order.New},
			{"an unrecognised status", `,"executedQty":"0","status":"WAITING"`, order.New},
			{"a partial fill", `,"executedQty":"1","status":"PARTIALLY_FILLED"`, order.PartiallyFilled},
			{"a cancellation", `,"executedQty":"0","status":"CANCELED"`, order.Cancelled},
		} {
			ex, _ := wrapperTestServer(t, map[string][]string{
				"POST /openApi/spot/v1/trade/order": {`{"code":0,"data":{"symbol":"BTC-USDT","orderId":13,"transactTime":1700000000000,"origQty":"2","type":"MARKET","side":"SELL"` + tc.fields + `}}`},
			})
			resp, err := ex.submitSpotOrder(t.Context(), &order.Submit{Exchange: ex.Name, Pair: currency.NewBTCUSDT(), AssetType: asset.Spot, Side: order.Sell, Type: order.Market, Amount: 2})
			require.NoErrorf(t, err, "submitSpotOrder must not error for a reply with %s", tc.name)
			assert.Equalf(t, tc.exp, resp.Status, "submitSpotOrder should report a market order's status for a reply with %s", tc.name)
		}
	})

	t.Run("sized in the quote currency", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name      string
			s         *order.Submit
			reply     string
			amount    float64
			remaining float64
			average   float64
			status    order.Status
		}{
			{
				name:   "filled market buy",
				s:      &order.Submit{Side: order.Buy, Type: order.Market, QuoteAmount: 100},
				reply:  `"origQty":"0.002","executedQty":"0.002","cummulativeQuoteQty":"100","status":"FILLED","type":"MARKET"`,
				amount: 0.002, average: 50000, status: order.New,
			},
			{
				name:   "market buy with an amount it does not trade",
				s:      &order.Submit{Side: order.Buy, Type: order.Market, Amount: 1, QuoteAmount: 100},
				reply:  `"origQty":"0.002","executedQty":"0.002","cummulativeQuoteQty":"100","status":"FILLED","type":"MARKET"`,
				amount: 0.002, average: 50000, status: order.New,
			},
			{
				name:   "reply without an original quantity",
				s:      &order.Submit{Side: order.Buy, Type: order.Market, QuoteAmount: 100},
				reply:  `"origQty":"0","executedQty":"0.002","cummulativeQuoteQty":"100","status":"FILLED","type":"MARKET"`,
				amount: 0.002, average: 50000, status: order.New,
			},
			{
				name:   "resting limit buy",
				s:      &order.Submit{Side: order.Buy, Type: order.Limit, Price: 10, QuoteAmount: 50},
				reply:  `"origQty":"5","executedQty":"0","status":"PENDING","type":"LIMIT"`,
				amount: 5, remaining: 5, status: order.Pending,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex, requests := wrapperTestServer(t, map[string][]string{
					"POST /openApi/spot/v1/trade/order": {`{"code":0,"data":{"symbol":"BTC-USDT","orderId":13,"transactTime":1700000000000,` + tc.reply + `,"side":"BUY"}}`},
				})
				tc.s.Exchange, tc.s.Pair, tc.s.AssetType = ex.Name, currency.NewBTCUSDT(), asset.Spot
				resp, err := ex.submitSpotOrder(t.Context(), tc.s)
				require.NoError(t, err, "submitSpotOrder must not error")
				assert.Empty(t, requests()[0].params.Get("quantity"), "submitSpotOrder should size the order in the quote currency")
				assert.Equal(t, tc.amount, resp.Amount, "submitSpotOrder should report the base amount the reply gives")
				assert.Equal(t, tc.remaining, resp.RemainingAmount, "submitSpotOrder should report the unfilled amount")
				assert.Equal(t, tc.average, resp.AverageExecutedPrice, "submitSpotOrder should report the average price")
				assert.Equal(t, tc.status, resp.Status, "submitSpotOrder should report an order filled at placement as new, for the order store to settle")
			})
		}
	})
}

func TestSetPlacedOrderStatus(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, reported string
		exp            order.Status
	}{
		{name: "no status kept", exp: order.New},
		{name: "reported status set", reported: "FILLED", exp: order.Filled},
		{name: "unconvertible status kept", reported: "ODD", exp: order.New},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			status := order.New
			e.setPlacedOrderStatus(&status, asset.Spot, "1", tc.reported)
			assert.Equalf(t, tc.exp, status, "setPlacedOrderStatus should set status %q as %s", tc.reported, tc.exp)
		})
	}
}

func TestReportExecutedPlacementNew(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		status   order.Status
		amount   float64
		executed types.Number
		exp      order.Status
	}{
		{"filled at placement", order.Filled, 1, 1, order.New},
		{"cancelled after executing", order.Cancelled, 2, 1, order.New},
		{"finished without executing", order.Filled, 1, 0, order.Filled},
		{"still active", order.PartiallyFilled, 2, 1, order.PartiallyFilled},
		{"no amount", order.Filled, 0, 1, order.Filled},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			status := tc.status
			reportExecutedPlacementNew(&status, tc.amount, tc.executed)
			assert.Equalf(t, tc.exp, status, "reportExecutedPlacementNew should report a %s order of %v executing %v as %s", tc.status, tc.amount, tc.executed, tc.exp)
		})
	}
}

func TestPerpetualOrderValues(t *testing.T) {
	t.Parallel()
	pair := currency.NewBTCUSDT()
	side, orderType, timeInForce, err := perpetualOrderValues(&order.Submit{Pair: pair, AssetType: asset.USDTMarginedFutures, Side: order.Long, Type: order.StopLimit, Price: 10, TriggerPrice: 9, Amount: 1, TimeInForce: order.PostOnly})
	require.NoError(t, err, "perpetualOrderValues must not error")
	assert.Equal(t, SideBuy, side, "perpetualOrderValues should format the side")
	assert.Equal(t, OrderTypeStop, orderType, "perpetualOrderValues should format the type")
	assert.Equal(t, TimeInForcePostOnly, timeInForce, "perpetualOrderValues should format the time in force")
	for _, tc := range []struct {
		name string
		s    *order.Submit
		err  error
	}{
		{"amount", &order.Submit{Pair: pair, AssetType: asset.USDTMarginedFutures, Side: order.Buy, Type: order.Market, QuoteAmount: 1}, order.ErrAmountIsInvalid},
		{"side", &order.Submit{Pair: pair, AssetType: asset.USDTMarginedFutures, Side: order.AnySide, Type: order.Market, Amount: 1}, order.ErrSideIsInvalid},
		{"type", &order.Submit{Pair: pair, AssetType: asset.CoinMarginedFutures, Side: order.Buy, Type: order.TrailingStop, Amount: 1}, order.ErrUnsupportedOrderType},
		{"time in force", &order.Submit{Pair: pair, AssetType: asset.USDTMarginedFutures, Side: order.Buy, Type: order.Market, Amount: 1, TimeInForce: order.GoodTillTime}, order.ErrUnsupportedTimeInForce},
		{"price", &order.Submit{Pair: pair, AssetType: asset.USDTMarginedFutures, Side: order.Buy, Type: order.TakeProfit, TriggerPrice: 1, Amount: 1}, order.ErrPriceMustBeSetIfLimitOrder},
		{"trigger", &order.Submit{Pair: pair, AssetType: asset.CoinMarginedFutures, Side: order.Buy, Type: order.TakeProfitMarket, Amount: 1}, errTriggerPriceRequired},
	} {
		_, _, _, err := perpetualOrderValues(tc.s)
		assert.ErrorIsf(t, err, tc.err, "perpetualOrderValues should reject an order with an invalid %s", tc.name)
	}
}

func TestSwapOrderRequest(t *testing.T) {
	t.Parallel()
	pair := currency.NewBTCUSDT()
	ex, requests := wrapperTestServer(t, map[string][]string{"GET /openApi/swap/v1/positionSide/dual": {wrapperOneWayModeReply}})
	for _, tc := range []struct {
		name string
		s    *order.Submit
		exp  *SwapOrderRequest
	}{
		{"limit", &order.Submit{Pair: pair, AssetType: asset.USDTMarginedFutures, Side: order.Buy, Type: order.Limit, Price: 10, Amount: 2, ClientOrderID: "c1", TimeInForce: order.FillOrKill}, &SwapOrderRequest{Symbol: pair, Type: OrderTypeLimit, Side: SideBuy, PositionSide: PositionSideBoth, Price: 10, Quantity: 2, ClientOrderID: "c1", TimeInForce: TimeInForceFillOrKill}},
		{"reduce only", &order.Submit{Pair: pair, AssetType: asset.USDTMarginedFutures, Side: order.Sell, Type: order.Market, Amount: 2, ReduceOnly: true}, &SwapOrderRequest{Symbol: pair, Type: OrderTypeMarket, Side: SideSell, PositionSide: PositionSideBoth, ReduceOnly: true, Quantity: 2}},
		{"stop market", &order.Submit{Pair: pair, AssetType: asset.USDTMarginedFutures, Side: order.Sell, Type: order.StopMarket, Price: 8, TriggerPrice: 9, TriggerPriceType: order.MarkPrice, Amount: 2}, &SwapOrderRequest{Symbol: pair, Type: OrderTypeStopMarket, Side: SideSell, PositionSide: PositionSideBoth, StopPrice: 9, WorkingType: WorkingTypeMarkPrice, Quantity: 2}},
		{"trailing distance", &order.Submit{Pair: pair, AssetType: asset.USDTMarginedFutures, Side: order.Sell, Type: order.TrailingStop, TrackingMode: order.Distance, TrackingValue: 5, TriggerPrice: 100, Amount: 2}, &SwapOrderRequest{Symbol: pair, Type: OrderTypeTrailingStopMarket, Side: SideSell, PositionSide: PositionSideBoth, Price: 5, ActivationPrice: 100, WorkingType: WorkingTypeContractPrice, Quantity: 2}},
		{"trailing percentage", &order.Submit{Pair: pair, AssetType: asset.USDTMarginedFutures, Side: order.Buy, Type: order.TrailingStop, TrackingMode: order.Percentage, TrackingValue: 1.5, TriggerPriceType: order.IndexPrice, Amount: 2}, &SwapOrderRequest{Symbol: pair, Type: OrderTypeTrailingStopMarket, Side: SideBuy, PositionSide: PositionSideBoth, PriceRate: 0.015, WorkingType: WorkingTypeIndexPrice, Quantity: 2}},
	} {
		req, err := ex.swapOrderRequest(t.Context(), tc.s)
		require.NoErrorf(t, err, "swapOrderRequest must not error for a %s order", tc.name)
		assert.Equalf(t, tc.exp, req, "swapOrderRequest should build the %s order BingX takes", tc.name)
	}
	assert.Len(t, requests(), 1, "swapOrderRequest should look the position mode up once")
	for _, tc := range []struct {
		name string
		s    *order.Submit
		err  error
	}{
		{"order", &order.Submit{Pair: pair, AssetType: asset.USDTMarginedFutures, Side: order.Buy, Type: order.Market}, order.ErrAmountIsInvalid},
		{"trailing value", &order.Submit{Pair: pair, AssetType: asset.USDTMarginedFutures, Side: order.Buy, Type: order.TrailingStop, TrackingMode: order.Distance, Amount: 1}, errTrailingValueRequired},
		{"tracking mode", &order.Submit{Pair: pair, AssetType: asset.USDTMarginedFutures, Side: order.Buy, Type: order.TrailingStop, TrackingValue: 1, Amount: 1}, order.ErrUnknownTrackingMode},
		{"price type", &order.Submit{Pair: pair, AssetType: asset.USDTMarginedFutures, Side: order.Buy, Type: order.StopMarket, TriggerPrice: 1, TriggerPriceType: order.PriceType(99), Amount: 1}, order.ErrUnknownPriceType},
	} {
		_, err := ex.swapOrderRequest(t.Context(), tc.s)
		assert.ErrorIsf(t, err, tc.err, "swapOrderRequest should reject an order with an invalid %s", tc.name)
	}
	failing, _ := wrapperTestServer(t, map[string][]string{"GET /openApi/swap/v1/positionSide/dual": {`{"code":100500,"msg":"busy"}`}})
	_, err := failing.swapOrderRequest(t.Context(), &order.Submit{Pair: pair, AssetType: asset.USDTMarginedFutures, Side: order.Buy, Type: order.Market, Amount: 1})
	assert.ErrorIs(t, err, errAPIResponse, "swapOrderRequest should return the position mode lookup's error")

	t.Run("client order ID on a type other than LIMIT or MARKET", func(t *testing.T) {
		t.Parallel()
		ex, requests := wrapperTestServer(t, nil)
		_, err := ex.swapOrderRequest(t.Context(), &order.Submit{Pair: pair, AssetType: asset.USDTMarginedFutures, Side: order.Sell, Type: order.StopMarket, TriggerPrice: 9, Amount: 1, ClientOrderID: "c1"})
		assert.ErrorIs(t, err, errSwapClientOrderIDUnsupportedType, "swapOrderRequest should refuse a client order ID on a STOP_MARKET order")
		assert.Empty(t, requests(), "swapOrderRequest should refuse the order before looking the position mode up")
	})
}

func TestSubmitSwapOrder(t *testing.T) {
	t.Parallel()
	submission := &order.Submit{Pair: currency.NewBTCUSDT(), AssetType: asset.USDTMarginedFutures, Side: order.Buy, Type: order.Limit, Price: 10, Amount: 2}
	ex, requests := wrapperTestServer(t, map[string][]string{
		"GET /openApi/swap/v1/positionSide/dual": {wrapperHedgeModeReply, wrapperOneWayModeReply},
		"POST /openApi/swap/v2/trade/order": {
			`{"code":109400,"msg":"In the one-way mode, the 'PositionSide' field can only be set to BOTH."}`,
			`{"code":0,"data":{"order":{"symbol":"BTC-USDT","orderId":7,"side":"BUY","positionSide":"BOTH","type":"LIMIT","clientOrderId":"c7","status":"FILLED","avgPrice":"10.5","executedQty":"2"}}}`,
		},
	})
	submission.Exchange = ex.Name
	resp, err := ex.submitSwapOrder(t.Context(), submission)
	require.NoError(t, err, "submitSwapOrder must not error")
	got := requests()
	require.Len(t, got, 4, "submitSwapOrder must look the position mode up again and place the order once more")
	assert.Equal(t, "LONG", got[1].params.Get("positionSide"), "submitSwapOrder should first place the order for the cached hedge mode")
	assert.Equal(t, "BOTH", got[3].params.Get("positionSide"), "submitSwapOrder should place the order again for the mode looked up")
	assert.Equal(t, "7", resp.OrderID, "submitSwapOrder should return the order ID")
	assert.Equal(t, "c7", resp.ClientOrderID, "submitSwapOrder should return the client order ID")
	assert.Equal(t, order.New, resp.Status, "submitSwapOrder should report an order filled at placement as new")
	assert.Equal(t, 10.5, resp.AverageExecutedPrice, "submitSwapOrder should report the average price")
	assert.Equal(t, 21.0, resp.Cost, "submitSwapOrder should report the cost")
	assert.Zero(t, resp.RemainingAmount, "submitSwapOrder should report nothing left of a filled order")

	t.Run("reply statuses", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name, field string
			exp         order.Status
		}{
			{"no status", "", order.New},
			{"an unrecognised status", `,"status":"WAITING"`, order.New},
			{"a partial fill", `,"status":"PARTIALLY_FILLED","executedQty":"1"`, order.PartiallyFilled},
			{"a cancellation", `,"status":"CANCELED"`, order.Cancelled},
		} {
			ex, _ := wrapperTestServer(t, map[string][]string{
				"GET /openApi/swap/v1/positionSide/dual": {wrapperHedgeModeReply},
				"POST /openApi/swap/v2/trade/order":      {`{"code":0,"data":{"order":{"symbol":"BTC-USDT","orderId":8,"side":"BUY","positionSide":"LONG","type":"MARKET","clientOrderId":"","workingType":"MARK_PRICE"` + tc.field + `}}}`},
			})
			resp, err := ex.submitSwapOrder(t.Context(), &order.Submit{Exchange: ex.Name, Pair: currency.NewBTCUSDT(), AssetType: asset.USDTMarginedFutures, Side: order.Buy, Type: order.Market, Amount: 2})
			require.NoErrorf(t, err, "submitSwapOrder must not error for a reply with %s", tc.name)
			assert.Equalf(t, tc.exp, resp.Status, "submitSwapOrder should report a market order's status for a reply with %s", tc.name)
		}
	})

	t.Run("errors", func(t *testing.T) {
		t.Parallel()
		ex, requests := wrapperTestServer(t, map[string][]string{
			"GET /openApi/swap/v1/positionSide/dual": {wrapperHedgeModeReply, wrapperHedgeModeReply, `{"code":100500,"msg":"busy"}`},
			"POST /openApi/swap/v2/trade/order": {
				`{"code":109400,"msg":"positionSide is invalid"}`,
				`{"code":109400,"msg":"positionSide is invalid"}`,
				`{"code":101204,"msg":"Insufficient margin"}`,
				`{"code":109400,"msg":"positionSide is invalid"}`,
			},
		})
		s := *submission
		s.Exchange = ex.Name
		_, err := ex.submitSwapOrder(t.Context(), &s)
		assert.ErrorIs(t, err, errAPIResponse, "submitSwapOrder should return a second refusal")
		assert.Len(t, requests(), 4, "submitSwapOrder should place the order twice at most")
		_, err = ex.submitSwapOrder(t.Context(), &s)
		assert.ErrorIs(t, err, errAPIResponse, "submitSwapOrder should return another refusal without placing the order again")
		_, err = ex.submitSwapOrder(t.Context(), &s)
		assert.ErrorIs(t, err, errAPIResponse, "submitSwapOrder should return the position mode lookup's error")
		_, err = ex.submitSwapOrder(t.Context(), &order.Submit{Pair: currency.NewBTCUSDT(), AssetType: asset.USDTMarginedFutures, Side: order.Buy, Type: order.Market})
		assert.ErrorIs(t, err, order.ErrAmountIsInvalid, "submitSwapOrder should return the order's error")
		sent := len(requests())
		_, err = ex.submitSwapOrder(t.Context(), &order.Submit{Pair: currency.NewBTCUSDT(), AssetType: asset.USDTMarginedFutures, Side: order.Sell, Type: order.StopMarket, TriggerPrice: 9, Amount: 1, ClientOrderID: "c1"})
		assert.ErrorIs(t, err, errSwapClientOrderIDUnsupportedType, "submitSwapOrder should refuse a client order ID on a STOP_MARKET order")
		assert.Len(t, requests(), sent, "submitSwapOrder should send nothing for an order it refuses")
	})
}

func TestCoinMarginedOrderRequest(t *testing.T) {
	t.Parallel()
	pair := currency.NewPairWithDelimiter("BTC", "USD", currency.DashDelimiter)
	// The position mode cache is keyed by the account, so the exchange needs credentials, which live tests may lack
	hedged, _ := wrapperTestServer(t, map[string][]string{})
	req, err := hedged.coinMarginedOrderRequest(t.Context(), &order.Submit{Pair: pair, AssetType: asset.CoinMarginedFutures, Side: order.Sell, Type: order.StopMarket, Price: 8, TriggerPrice: 9, TriggerPriceType: order.IndexPrice, Amount: 3, ClientOrderID: "c1"})
	require.NoError(t, err, "coinMarginedOrderRequest must not error")
	assert.Equal(t, &PlaceCoinMarginedOrderRequest{Symbol: pair, Type: OrderTypeStopMarket, Side: SideSell, PositionSide: PositionSideShort, Quantity: 3, StopPrice: 9, WorkingType: WorkingTypeIndexPrice, ClientOrderID: "c1"}, req, "coinMarginedOrderRequest should build the order BingX takes")
	req, err = hedged.coinMarginedOrderRequest(t.Context(), &order.Submit{Pair: pair, AssetType: asset.CoinMarginedFutures, Side: order.Buy, Type: order.Limit, Price: 10, Amount: 1, ReduceOnly: true, TimeInForce: order.ImmediateOrCancel})
	require.NoError(t, err, "coinMarginedOrderRequest must not error")
	assert.Equal(t, &PlaceCoinMarginedOrderRequest{Symbol: pair, Type: OrderTypeLimit, Side: SideBuy, PositionSide: PositionSideShort, Price: 10, Quantity: 1, TimeInForce: TimeInForceImmediateOrCancel}, req, "coinMarginedOrderRequest should close the short position with a reduce only buy")
	_, err = hedged.coinMarginedOrderRequest(t.Context(), &order.Submit{Pair: pair, AssetType: asset.CoinMarginedFutures, Side: order.Buy, Type: order.StopMarket, TriggerPrice: 1, TriggerPriceType: order.PriceType(99), Amount: 1})
	assert.ErrorIs(t, err, order.ErrUnknownPriceType, "coinMarginedOrderRequest should reject an unknown trigger price type")
	_, err = hedged.coinMarginedOrderRequest(t.Context(), &order.Submit{Pair: pair, AssetType: asset.CoinMarginedFutures, Side: order.Buy, Type: order.Market})
	assert.ErrorIs(t, err, order.ErrAmountIsInvalid, "coinMarginedOrderRequest should return the order's error")

	t.Run("one-way mode", func(t *testing.T) {
		t.Parallel()
		ex, _ := wrapperTestServer(t, map[string][]string{})
		ex.recordCoinMarginedPositionMode(t.Context(), false)
		req, err := ex.coinMarginedOrderRequest(t.Context(), &order.Submit{Pair: pair, AssetType: asset.CoinMarginedFutures, Side: order.Sell, Type: order.Market, Amount: 2})
		require.NoError(t, err, "coinMarginedOrderRequest must not error")
		assert.Equal(t, &PlaceCoinMarginedOrderRequest{Symbol: pair, Type: OrderTypeMarket, Side: SideSell, PositionSide: PositionSideBoth, Quantity: 2}, req, "coinMarginedOrderRequest should place a one-way order on BOTH")
		req, err = ex.coinMarginedOrderRequest(t.Context(), &order.Submit{Pair: pair, AssetType: asset.CoinMarginedFutures, Side: order.Sell, Type: order.Market, Amount: 2, ReduceOnly: true})
		require.NoError(t, err, "coinMarginedOrderRequest must not error for a reduce only order")
		assert.Equal(t, &PlaceCoinMarginedOrderRequest{Symbol: pair, Type: OrderTypeMarket, Side: SideSell, PositionSide: PositionSideLong, Quantity: 2}, req, "coinMarginedOrderRequest should place a reduce only order on the hedge mode side it closes when one-way mode is cached")
		ex.SetCredentials(&accounts.Credentials{})
		_, err = ex.coinMarginedOrderRequest(t.Context(), &order.Submit{Pair: pair, AssetType: asset.CoinMarginedFutures, Side: order.Sell, Type: order.Market, Amount: 2})
		assert.Error(t, err, "coinMarginedOrderRequest should return the position side's error")
	})
}

func TestSubmitCoinMarginedOrder(t *testing.T) {
	t.Parallel()
	pair := currency.NewPairWithDelimiter("BTC", "USD", currency.DashDelimiter)
	ex, requests := wrapperTestServer(t, map[string][]string{
		"POST /openApi/cswap/v2/trade/order": {`{"code":0,"data":{"order":{"orderId":"5","symbol":"BTC-USD","positionSide":"LONG","side":"BUY","type":"MARKET","quantity":2,"clientOrderId":"c5"}}}`, `{"code":100500,"msg":"busy"}`},
	})
	resp, err := ex.submitCoinMarginedOrder(t.Context(), &order.Submit{Exchange: ex.Name, Pair: pair, AssetType: asset.CoinMarginedFutures, Side: order.Buy, Type: order.Market, Amount: 2, ClientOrderID: "c5"})
	require.NoError(t, err, "submitCoinMarginedOrder must not error")
	assert.Equal(t, "5", resp.OrderID, "submitCoinMarginedOrder should return the order ID")
	assert.Equal(t, "c5", resp.ClientOrderID, "submitCoinMarginedOrder should return the client order ID")
	assert.Equal(t, order.New, resp.Status, "submitCoinMarginedOrder should report a market order as new, as the reply has no status")
	assert.Equal(t, url.Values{"symbol": {"BTC-USD"}, "type": {"MARKET"}, "side": {"BUY"}, "positionSide": {"LONG"}, "quantity": {"2"}, "clientOrderId": {"c5"}}, requests()[0].params, "submitCoinMarginedOrder should place the order on the hedge mode side")
	_, err = ex.submitCoinMarginedOrder(t.Context(), &order.Submit{Exchange: ex.Name, Pair: pair, AssetType: asset.CoinMarginedFutures, Side: order.Buy, Type: order.Market, Amount: 2})
	assert.ErrorIs(t, err, errAPIResponse, "submitCoinMarginedOrder should return the request's error")
	_, err = ex.submitCoinMarginedOrder(t.Context(), &order.Submit{Pair: pair, AssetType: asset.CoinMarginedFutures, Side: order.Buy, Type: order.Market})
	assert.ErrorIs(t, err, order.ErrAmountIsInvalid, "submitCoinMarginedOrder should return the order's error")

	t.Run("position mode rejection", func(t *testing.T) {
		t.Parallel()
		hedgeRefusal := `{"code":109400,"msg":"In the one-way mode, the 'PositionSide' field can only be set to BOTH."}`
		oneWayRefusal := `{"code":109400,"msg":"In the hedge mode, the positionSide field can only be set to LONG or SHORT."}`
		placed := `{"code":0,"data":{"order":{"orderId":"6","symbol":"BTC-USD","side":"BUY","type":"MARKET","quantity":2}}}`
		ex, requests := wrapperTestServer(t, map[string][]string{
			"POST /openApi/cswap/v2/trade/order": {hedgeRefusal, placed, placed, oneWayRefusal, placed, hedgeRefusal, hedgeRefusal, `{"code":101204,"msg":"Insufficient margin"}`},
		})
		submission := order.Submit{Exchange: ex.Name, Pair: pair, AssetType: asset.CoinMarginedFutures, Side: order.Buy, Type: order.Market, Amount: 2}
		positionSides := func() []string {
			got := requests()
			sides := make([]string, len(got))
			for i := range got {
				sides[i] = got[i].params.Get("positionSide")
			}
			return sides
		}
		resp, err := ex.submitCoinMarginedOrder(t.Context(), &submission)
		require.NoError(t, err, "submitCoinMarginedOrder must not error")
		assert.Equal(t, "6", resp.OrderID, "submitCoinMarginedOrder should return the order placed once more")
		assert.Equal(t, []string{"LONG", "BOTH"}, positionSides(), "submitCoinMarginedOrder should place an order refused for its hedge mode side once more on BOTH")
		_, err = ex.submitCoinMarginedOrder(t.Context(), &submission)
		require.NoError(t, err, "submitCoinMarginedOrder must not error")
		assert.Equal(t, "BOTH", positionSides()[2], "submitCoinMarginedOrder should place later orders in one-way mode")
		_, err = ex.submitCoinMarginedOrder(t.Context(), &submission)
		require.NoError(t, err, "submitCoinMarginedOrder must not error")
		assert.Equal(t, []string{"BOTH", "LONG"}, positionSides()[3:], "submitCoinMarginedOrder should place an order refused on BOTH once more on its hedge mode side")
		_, err = ex.submitCoinMarginedOrder(t.Context(), &submission)
		assert.ErrorIs(t, err, errAPIResponse, "submitCoinMarginedOrder should return a second refusal")
		assert.Equal(t, []string{"LONG", "BOTH"}, positionSides()[5:], "submitCoinMarginedOrder should place an order twice at most")
		_, err = ex.submitCoinMarginedOrder(t.Context(), &submission)
		assert.ErrorIs(t, err, errAPIResponse, "submitCoinMarginedOrder should return a refusal for another reason")
		assert.Len(t, requests(), 8, "submitCoinMarginedOrder should not place an order refused for another reason again")
	})

	t.Run("reduce only", func(t *testing.T) {
		t.Parallel()
		hedgeRefusal := `{"code":109400,"msg":"In the one-way mode, the 'PositionSide' field can only be set to BOTH."}`
		placed := `{"code":0,"data":{"order":{"orderId":"7","symbol":"BTC-USD","side":"BUY","type":"MARKET","quantity":2}}}`
		ex, requests := wrapperTestServer(t, map[string][]string{
			"POST /openApi/cswap/v2/trade/order": {hedgeRefusal, hedgeRefusal, placed, placed, `{"code":101204,"msg":"Insufficient margin"}`},
		})
		hedgeMode := func() bool {
			t.Helper()
			hedgeMode, err := ex.coinMarginedHedgeMode(t.Context())
			require.NoError(t, err, "coinMarginedHedgeMode must not error")
			return hedgeMode
		}
		reduceOnly := order.Submit{Exchange: ex.Name, Pair: pair, AssetType: asset.CoinMarginedFutures, Side: order.Buy, Type: order.Market, Amount: 2, ReduceOnly: true}
		_, err := ex.submitCoinMarginedOrder(t.Context(), &reduceOnly)
		assert.ErrorIs(t, err, errCoinMarginedReduceOnlyOneWay, "submitCoinMarginedOrder should report a reduce only order refused for its position side")
		assert.ErrorIs(t, err, errAPIResponse, "submitCoinMarginedOrder should return the refusal")
		assert.False(t, hedgeMode(), "submitCoinMarginedOrder should record one-way mode once BingX refuses a reduce only order's position side")

		_, err = ex.submitCoinMarginedOrder(t.Context(), &reduceOnly)
		assert.ErrorIs(t, err, errCoinMarginedReduceOnlyOneWay, "submitCoinMarginedOrder should report a reduce only order refused for its position side")
		got := requests()
		require.Len(t, got, 2, "submitCoinMarginedOrder must place each refused reduce only order once")
		assert.Equal(t, "SHORT", got[1].params.Get("positionSide"), "submitCoinMarginedOrder should place a reduce only order on the hedge mode side it closes when one-way mode is cached")
		assert.False(t, hedgeMode(), "submitCoinMarginedOrder should keep one-way mode after BingX refuses a reduce only order's position side")

		resp, err := ex.submitCoinMarginedOrder(t.Context(), &reduceOnly)
		require.NoError(t, err, "submitCoinMarginedOrder must not error for a reduce only order BingX accepts")
		assert.Equal(t, "7", resp.OrderID, "submitCoinMarginedOrder should return the order ID")
		assert.Equal(t, "SHORT", requests()[2].params.Get("positionSide"), "submitCoinMarginedOrder should place a reduce only order on the hedge mode side it closes")
		assert.True(t, hedgeMode(), "submitCoinMarginedOrder should record hedge mode once BingX accepts a reduce only order")
		_, err = ex.submitCoinMarginedOrder(t.Context(), &order.Submit{Exchange: ex.Name, Pair: pair, AssetType: asset.CoinMarginedFutures, Side: order.Buy, Type: order.Market, Amount: 2})
		require.NoError(t, err, "submitCoinMarginedOrder must not error")
		assert.Equal(t, "LONG", requests()[3].params.Get("positionSide"), "submitCoinMarginedOrder should place later orders in hedge mode")

		ex.recordCoinMarginedPositionMode(t.Context(), false)
		_, err = ex.submitCoinMarginedOrder(t.Context(), &reduceOnly)
		assert.ErrorIs(t, err, errAPIResponse, "submitCoinMarginedOrder should return a refusal for another reason")
		assert.NotErrorIs(t, err, errCoinMarginedReduceOnlyOneWay, "submitCoinMarginedOrder should not report a refusal for another reason as one for the position mode")
		assert.False(t, hedgeMode(), "submitCoinMarginedOrder should record no mode for a reduce only order refused for another reason")
		assert.Len(t, requests(), 5, "submitCoinMarginedOrder should place a reduce only order once")
	})
}

// wrapperSpotOrderReply is a spot order details reply for a partly filled LIMIT buy of 3 at 10, order 11
const wrapperSpotOrderReply = `{"code":0,"data":{"symbol":"BTC-USDT","orderId":11,"price":"10","stopPrice":"0","origQty":"3","executedQty":"1","status":"PARTIALLY_FILLED","type":"LIMIT","side":"BUY","time":1700000000000,"updateTime":1700000000001}}`

// wrapperSpotCancelledReply returns a spot cancellation reply for an order cancelled after executing the quantity given
func wrapperSpotCancelledReply(orderID int, executed string) string {
	return `{"code":0,"data":{"symbol":"BTC-USDT","orderId":` + strconv.Itoa(orderID) + `,"executedQty":"` + executed + `","status":"CANCELED"}}`
}

// wrapperSpotPlacedReply is a spot order placement reply for order 12, a LIMIT buy of 2 at 11
const wrapperSpotPlacedReply = `{"code":0,"data":{"symbol":"BTC-USDT","orderId":12,"transactTime":1700000000002,"price":"11","stopPrice":"0","origQty":"2","executedQty":"0","cummulativeQuoteQty":"0","status":"PENDING","type":"LIMIT","side":"BUY","clientOrderID":"n12"}}`

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	_, err := e.ModifyOrder(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrModifyOrderIsNil, "ModifyOrder must reject a nil modification")
	_, err = e.ModifyOrder(t.Context(), &order.Modify{Pair: currency.NewPairWithDelimiter("BTC", "USD", currency.DashDelimiter), AssetType: asset.CoinMarginedFutures, OrderID: "1", Price: 1})
	require.ErrorIs(t, err, common.ErrFunctionNotSupported, "ModifyOrder must report that BingX modifies no coin-M orders")
	_, err = e.ModifyOrder(t.Context(), &order.Modify{Pair: currency.NewBTCUSDT(), AssetType: asset.Futures, OrderID: "1", Price: 1})
	require.ErrorIs(t, err, asset.ErrNotSupported, "ModifyOrder must reject an unsupported asset")

	ex, requests := wrapperTestServer(t, map[string][]string{
		"GET /openApi/spot/v1/trade/query":   {wrapperSpotOrderReply},
		"POST /openApi/spot/v1/trade/cancel": {wrapperSpotCancelledReply(11, "1")},
		"POST /openApi/spot/v1/trade/order":  {wrapperSpotPlacedReply},
		"GET /openApi/swap/v2/trade/order":   {`{"code":0,"data":{"order":{"symbol":"BTC-USDT","orderId":21,"side":"BUY","positionSide":"LONG","type":"LIMIT","origQty":"2","price":"10","executedQty":"0","status":"PENDING"}}}`},
		"POST /openApi/swap/v1/trade/amend":  {`{"code":0,"data":{"orderId":21,"clientOrderId":"","symbol":"BTC-USDT","quantity":3}}`},
	})
	resp, err := ex.ModifyOrder(t.Context(), &order.Modify{Pair: currency.NewBTCUSDT(), AssetType: asset.Spot, OrderID: "11", Price: 11})
	require.NoError(t, err, "ModifyOrder must not error")
	assert.Equal(t, "12", resp.OrderID, "ModifyOrder should return a spot replacement's order ID")
	resp, err = ex.ModifyOrder(t.Context(), &order.Modify{Pair: currency.NewBTCUSDT(), AssetType: asset.USDTMarginedFutures, OrderID: "21", Amount: 3})
	require.NoError(t, err, "ModifyOrder must not error")
	assert.Equal(t, 3.0, resp.Amount, "ModifyOrder should amend a USDT-M order's amount")
	assert.Len(t, requests(), 5, "ModifyOrder should fetch and then change each order")

	t.Run("invalid prices", func(t *testing.T) {
		t.Parallel()
		for _, a := range []asset.Item{asset.Spot, asset.USDTMarginedFutures} {
			for _, tc := range []struct {
				name                string
				price, triggerPrice float64
			}{
				{"negative price", -1, 0},
				{"NaN price", math.NaN(), 0},
				{"infinite price", math.Inf(1), 0},
				{"negative infinite price", math.Inf(-1), 0},
				{"negative trigger price", 10, -1},
				{"NaN trigger price", 10, math.NaN()},
				{"infinite trigger price", 10, math.Inf(1)},
				{"negative infinite trigger price", 10, math.Inf(-1)},
			} {
				t.Run(a.String()+" "+tc.name, func(t *testing.T) {
					t.Parallel()
					ex, requests := wrapperTestServer(t, map[string][]string{})
					_, err := ex.ModifyOrder(t.Context(), &order.Modify{Pair: currency.NewBTCUSDT(), AssetType: a, OrderID: "11", Amount: 2, Price: tc.price, TriggerPrice: tc.triggerPrice})
					assert.ErrorIs(t, err, errInvalidPrice, "ModifyOrder should refuse an invalid price")
					assert.Empty(t, requests(), "ModifyOrder should send nothing for a price it refuses")
				})
			}
		}
	})

	t.Run("errors before the lookup", func(t *testing.T) {
		t.Parallel()
		ex, requests := wrapperTestServer(t, map[string][]string{})
		account, err := ex.productAccount(t.Context(), exchange.RestSpot)
		require.NoError(t, err, "productAccount must not error")
		unlock, err := ex.orderModifications.lock(t.Context(), orderModificationKey{account: account, asset: asset.Spot, orderID: 11})
		require.NoError(t, err, "lock must not error")
		defer unlock()
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		_, err = ex.ModifyOrder(ctx, &order.Modify{Pair: currency.NewBTCUSDT(), AssetType: asset.Spot, OrderID: "11", Price: 11})
		assert.ErrorIs(t, err, context.Canceled, "ModifyOrder should return the context's error when it ends while another modification of the order runs")
		_, err = ex.ModifyOrder(t.Context(), &order.Modify{Pair: currency.NewBTCUSDT(), AssetType: asset.Spot, OrderID: "x", Price: 11})
		assert.ErrorIs(t, err, errInvalidOrderID, "ModifyOrder should reject an invalid order ID")
		ex.SetCredentials(&accounts.Credentials{})
		_, err = ex.ModifyOrder(t.Context(), &order.Modify{Pair: currency.NewBTCUSDT(), AssetType: asset.USDTMarginedFutures, OrderID: "21", Price: 11})
		assert.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "ModifyOrder should return the credentials error")
		assert.Empty(t, requests(), "ModifyOrder should send nothing when it cannot name the order's account or wait for its lock")
		withoutEndpoints, requests := wrapperTestServer(t, map[string][]string{})
		withoutEndpoints.API.Endpoints = withoutEndpoints.NewEndpoints()
		_, err = withoutEndpoints.ModifyOrder(t.Context(), &order.Modify{Pair: currency.NewBTCUSDT(), AssetType: asset.Spot, OrderID: "11", Price: 11})
		assert.ErrorIs(t, err, exchange.ErrEndpointPathNotFound, "ModifyOrder should return the endpoint lookup's error")
		assert.Empty(t, requests(), "ModifyOrder should send nothing without the product's endpoint")
	})

	t.Run("concurrent modifications of one order", func(t *testing.T) {
		t.Parallel()
		type product struct {
			a                                                   asset.Item
			lookupRoute, cancelRoute, placeRoute                string
			openReply, closedReply, cancelledReply, placedReply string
		}
		spot := product{
			a:           asset.Spot,
			lookupRoute: "GET /openApi/spot/v1/trade/query", cancelRoute: "POST /openApi/spot/v1/trade/cancel", placeRoute: "POST /openApi/spot/v1/trade/order",
			openReply:      wrapperSpotOrderReply,
			closedReply:    `{"code":0,"data":{"symbol":"BTC-USDT","orderId":11,"price":"10","stopPrice":"0","origQty":"3","executedQty":"1","status":"CANCELED","type":"LIMIT","side":"BUY"}}`,
			cancelledReply: wrapperSpotCancelledReply(11, "1"),
			placedReply:    wrapperSpotPlacedReply,
		}
		usdtMargined := product{
			a:           asset.USDTMarginedFutures,
			lookupRoute: "GET /openApi/swap/v2/trade/order", cancelRoute: "DELETE /openApi/swap/v2/trade/order", placeRoute: "POST /openApi/swap/v2/trade/order",
			openReply:      wrapperSwapOrderReply(""),
			closedReply:    `{"code":0,"data":{"order":{"symbol":"BTC-USDT","orderId":21,"side":"BUY","positionSide":"LONG","type":"LIMIT","origQty":"2","price":"10","executedQty":"0.5","status":"CANCELLED","clientOrderId":"c21"}}}`,
			cancelledReply: wrapperSwapCancelledReply(21, "0.5"),
			placedReply:    wrapperSwapPlacedReply("LIMIT", ""),
		}
		for _, tc := range []struct {
			name    string
			product product
			// names holds the ways the modifications name the order, taken in turn
			names []order.Modify
		}{
			{name: "spot order ID", product: spot, names: []order.Modify{{OrderID: "11"}}},
			{name: "spot client order ID", product: spot, names: []order.Modify{{ClientOrderID: "c11"}}},
			{name: "spot order ID with and without a client order ID", product: spot, names: []order.Modify{{OrderID: "11"}, {OrderID: "11", ClientOrderID: "c11"}}},
			{name: "spot order ID and client order ID", product: spot, names: []order.Modify{{OrderID: "11"}, {ClientOrderID: "c11"}}},
			{name: "USDT-M order ID", product: usdtMargined, names: []order.Modify{{OrderID: "21"}}},
			{name: "USDT-M order ID and client order ID", product: usdtMargined, names: []order.Modify{{OrderID: "21"}, {ClientOrderID: "c21"}}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				const modifications = 4
				var ex *Exchange
				var mu sync.Mutex
				var closed bool
				var lookups, cancellations, placements int
				// linedUp reports whether each modification waits for a lock another holds or has looked the order up
				linedUp := func() bool {
					var waiting int
					for _, holders := range testOrderModificationHolders(&ex.orderModifications) {
						waiting += holders - 1
					}
					mu.Lock()
					defer mu.Unlock()
					return waiting+lookups >= modifications
				}
				// The first lookup and cancellation are answered once the modifications line up, so any modification
				// not made to wait for the first looks the order up while it is open
				waitLinedUp := func() {
					deadline := time.Now().Add(5 * time.Second)
					for !linedUp() {
						if time.Now().After(deadline) {
							assert.Fail(t, "each modification should wait for the order's lock or look the order up")
							return
						}
						time.Sleep(time.Millisecond)
					}
				}
				ex = newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
					switch route := r.Method + " " + r.URL.Path; route {
					case tc.product.lookupRoute:
						mu.Lock()
						lookups++
						first, reply := lookups == 1, tc.product.openReply
						if closed {
							reply = tc.product.closedReply
						}
						mu.Unlock()
						if first {
							waitLinedUp()
						}
						_, err := io.WriteString(w, reply)
						assert.NoError(t, err, "WriteString should not error")
					case tc.product.cancelRoute:
						mu.Lock()
						cancellations++
						first := cancellations == 1
						mu.Unlock()
						if first {
							waitLinedUp()
						}
						mu.Lock()
						wasClosed := closed
						closed = true
						mu.Unlock()
						if wasClosed {
							// The reply to a cancellation of the order another cancelled is lost, which leaves the outcome
							// to the lookup that follows
							panic(http.ErrAbortHandler)
						}
						_, err := io.WriteString(w, tc.product.cancelledReply)
						assert.NoError(t, err, "WriteString should not error")
					case tc.product.placeRoute:
						mu.Lock()
						placements++
						mu.Unlock()
						_, err := io.WriteString(w, tc.product.placedReply)
						assert.NoError(t, err, "WriteString should not error")
					default:
						assert.Failf(t, "unexpected request", "%s should not be requested", route)
						w.WriteHeader(http.StatusNotFound)
					}
				})
				errs := make(chan error, modifications)
				for i := range modifications {
					action := tc.names[i%len(tc.names)]
					action.Exchange, action.Pair, action.AssetType, action.Price = ex.Name, currency.NewBTCUSDT(), tc.product.a, 11
					go func() {
						_, err := ex.ModifyOrder(t.Context(), &action)
						errs <- err
					}()
				}
				var replaced, refused int
				for range modifications {
					select {
					case err := <-errs:
						switch {
						case err == nil:
							replaced++
						case errors.Is(err, errOrderNotOpen):
							refused++
						default:
							assert.Failf(t, "unexpected ModifyOrder error", "ModifyOrder should replace the order or refuse it as no longer open, not return %v", err)
						}
					case <-time.After(10 * time.Second):
						require.FailNow(t, "every ModifyOrder must return")
					}
				}
				assert.Equal(t, 1, replaced, "ModifyOrder should replace the order once")
				assert.Equal(t, modifications-1, refused, "ModifyOrder should refuse each other modification, which finds the order closed")
				mu.Lock()
				defer mu.Unlock()
				assert.Equal(t, 1, cancellations, "ModifyOrder should cancel the order once")
				assert.Equal(t, 1, placements, "ModifyOrder should place one replacement for the order")
			})
		}
	})

	t.Run("modifications of different orders", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name          string
			first, second order.Modify
			// secondCredentials, when set, are the second modification's context credentials
			secondCredentials *accounts.Credentials
		}{
			{name: "order IDs", first: order.Modify{OrderID: "11"}, second: order.Modify{OrderID: "13"}},
			{name: "client order IDs", first: order.Modify{ClientOrderID: "c11"}, second: order.Modify{ClientOrderID: "c13"}},
			{name: "one order ID on two accounts", first: order.Modify{OrderID: "11"}, second: order.Modify{OrderID: "11"}, secondCredentials: &accounts.Credentials{Key: "other-key", Secret: "other-secret"}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				firstArrived, secondArrived := make(chan struct{}), make(chan struct{})
				var lookups atomic.Int32
				var overlapped atomic.Bool
				ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
					var reply string
					switch route := r.Method + " " + r.URL.Path; route {
					case "GET /openApi/spot/v1/trade/query":
						query := r.URL.Query()
						orderID := query.Get("orderId")
						switch {
						case orderID == "":
							// A lookup by client order ID names the order whose lock the modification then waits for
							orderID = strings.TrimPrefix(query.Get("clientOrderID"), "c")
						case lookups.Add(1) == 1:
							close(firstArrived)
							// Answered once the second order's lookup arrives, which a modification waiting for the first's
							// lock cannot send
							select {
							case <-secondArrived:
								overlapped.Store(true)
							case <-time.After(5 * time.Second):
							}
						default:
							close(secondArrived)
						}
						reply = wrapperSpotOrderReply
						if orderID == "13" {
							reply = strings.Replace(wrapperSpotOrderReply, `"orderId":11`, `"orderId":13`, 1)
						}
					case "POST /openApi/spot/v1/trade/cancel":
						reply = wrapperSpotCancelledReply(11, "1")
					case "POST /openApi/spot/v1/trade/order":
						reply = wrapperSpotPlacedReply
					default:
						assert.Failf(t, "unexpected request", "%s should not be requested", route)
						w.WriteHeader(http.StatusNotFound)
						return
					}
					_, err := io.WriteString(w, reply)
					assert.NoError(t, err, "WriteString should not error")
				})
				modify := func(ctx context.Context, action order.Modify) <-chan error {
					done := make(chan error, 1)
					action.Exchange, action.Pair, action.AssetType, action.Price = ex.Name, currency.NewBTCUSDT(), asset.Spot, 11
					go func() {
						_, err := ex.ModifyOrder(ctx, &action)
						done <- err
					}()
					return done
				}
				first := modify(t.Context(), tc.first)
				select {
				case <-firstArrived:
				case <-time.After(5 * time.Second):
					require.FailNow(t, "the first order's lookup must arrive")
				}
				secondCtx := t.Context()
				if tc.secondCredentials != nil {
					secondCtx = accounts.DeployCredentialsToContext(secondCtx, tc.secondCredentials)
				}
				second := modify(secondCtx, tc.second)
				for _, done := range []<-chan error{first, second} {
					select {
					case err := <-done:
						assert.NoError(t, err, "ModifyOrder should not error")
					case <-time.After(10 * time.Second):
						require.FailNow(t, "ModifyOrder must return")
					}
				}
				assert.True(t, overlapped.Load(), "ModifyOrder should look the second order up while a modification of the first holds its lock")
			})
		}
	})

	t.Run("client order ID", func(t *testing.T) {
		t.Parallel()
		ex, requests := wrapperTestServer(t, map[string][]string{
			"GET /openApi/spot/v1/trade/query":    {wrapperSpotOrderReply, wrapperSpotOrderReply, `{"code":100404,"msg":"order not exist"}`},
			"POST /openApi/spot/v1/trade/cancel":  {wrapperSpotCancelledReply(11, "1")},
			"POST /openApi/spot/v1/trade/order":   {wrapperSpotPlacedReply},
			"GET /openApi/swap/v2/trade/order":    {wrapperSwapOrderReply(""), wrapperSwapOrderReply("")},
			"DELETE /openApi/swap/v2/trade/order": {wrapperSwapCancelledReply(21, "0.5")},
			"POST /openApi/swap/v2/trade/order":   {wrapperSwapPlacedReply("LIMIT", "")},
		})
		resp, err := ex.ModifyOrder(t.Context(), &order.Modify{Pair: currency.NewBTCUSDT(), AssetType: asset.Spot, ClientOrderID: "c11", Price: 11})
		require.NoError(t, err, "ModifyOrder must not error for a spot order named by its client order ID")
		assert.Equal(t, "12", resp.OrderID, "ModifyOrder should return the spot replacement's order ID")
		resp, err = ex.ModifyOrder(t.Context(), &order.Modify{Pair: currency.NewBTCUSDT(), AssetType: asset.USDTMarginedFutures, ClientOrderID: "c21", Price: 11})
		require.NoError(t, err, "ModifyOrder must not error for a USDT-M order named by its client order ID")
		assert.Equal(t, "22", resp.OrderID, "ModifyOrder should return the USDT-M replacement's order ID")
		got := requests()
		require.Len(t, got, 8, "ModifyOrder must look each order up by its client order ID and again by its order ID, cancel it and place its replacement")
		for i, exp := range []url.Values{
			{"symbol": {"BTC-USDT"}, "clientOrderID": {"c11"}},
			{"symbol": {"BTC-USDT"}, "orderId": {"11"}},
			{"symbol": {"BTC-USDT"}, "orderId": {"11"}},
		} {
			assert.Equalf(t, exp, got[i].params, "ModifyOrder should send spot request %d by the order ID the client order ID's lookup returned", i)
		}
		for i, exp := range []url.Values{
			{"symbol": {"BTC-USDT"}, "clientOrderId": {"c21"}},
			{"symbol": {"BTC-USDT"}, "orderId": {"21"}},
			{"symbol": {"BTC-USDT"}, "orderId": {"21"}},
		} {
			assert.Equalf(t, exp, got[4+i].params, "ModifyOrder should send USDT-M request %d by the order ID the client order ID's lookup returned", i)
		}
		_, err = ex.ModifyOrder(t.Context(), &order.Modify{Pair: currency.NewBTCUSDT(), AssetType: asset.Spot, ClientOrderID: "c19", Price: 11})
		assert.ErrorIs(t, err, order.ErrOrderNotFound, "ModifyOrder should return the error of a lookup by client order ID")
		assert.Len(t, requests(), 9, "ModifyOrder should send nothing more once the lookup by client order ID fails")
		assert.Empty(t, testOrderModificationHolders(&ex.orderModifications), "ModifyOrder should take no lock for an order it could not look up")
	})

	t.Run("credentials changing during the modification", func(t *testing.T) {
		t.Parallel()
		const (
			spotClosed = `{"code":0,"data":{"symbol":"BTC-USDT","orderId":11,"price":"10","stopPrice":"0","origQty":"3","executedQty":"1","status":"CANCELED","type":"LIMIT","side":"BUY"}}`
			swapClosed = `{"code":0,"data":{"order":{"symbol":"BTC-USDT","orderId":21,"side":"BUY","positionSide":"LONG","type":"LIMIT","origQty":"2","price":"10","executedQty":"0.5","status":"CANCELLED"}}}`
		)
		for _, tc := range []struct {
			name   string
			action order.Modify
			routes map[string][]string
		}{
			{
				name: "spot cancellation answered", action: order.Modify{AssetType: asset.Spot, OrderID: "11"},
				routes: map[string][]string{
					"GET /openApi/spot/v1/trade/query":   {wrapperSpotOrderReply},
					"POST /openApi/spot/v1/trade/cancel": {wrapperCredentialsChange + wrapperSpotCancelledReply(11, "1")},
					"POST /openApi/spot/v1/trade/order":  {wrapperSpotPlacedReply},
				},
			},
			{
				name: "spot cancellation unanswered", action: order.Modify{AssetType: asset.Spot, OrderID: "11"},
				routes: map[string][]string{
					"GET /openApi/spot/v1/trade/query":   {wrapperSpotOrderReply, spotClosed},
					"POST /openApi/spot/v1/trade/cancel": {wrapperCredentialsChange + wrapperNoReply},
					"POST /openApi/spot/v1/trade/order":  {wrapperSpotPlacedReply},
				},
			},
			{
				name: "spot order named by its client order ID", action: order.Modify{AssetType: asset.Spot, ClientOrderID: "c11"},
				routes: map[string][]string{
					"GET /openApi/spot/v1/trade/query":   {wrapperCredentialsChange + wrapperSpotOrderReply, wrapperSpotOrderReply},
					"POST /openApi/spot/v1/trade/cancel": {wrapperSpotCancelledReply(11, "1")},
					"POST /openApi/spot/v1/trade/order":  {wrapperSpotPlacedReply},
				},
			},
			{
				name: "USDT-M cancellation answered", action: order.Modify{AssetType: asset.USDTMarginedFutures, OrderID: "21"},
				routes: map[string][]string{
					"GET /openApi/swap/v2/trade/order":    {wrapperSwapOrderReply("")},
					"DELETE /openApi/swap/v2/trade/order": {wrapperCredentialsChange + wrapperSwapCancelledReply(21, "0.5")},
					"POST /openApi/swap/v2/trade/order":   {wrapperSwapPlacedReply("LIMIT", "")},
				},
			},
			{
				name: "USDT-M cancellation unanswered", action: order.Modify{AssetType: asset.USDTMarginedFutures, OrderID: "21"},
				routes: map[string][]string{
					"GET /openApi/swap/v2/trade/order":    {wrapperSwapOrderReply(""), swapClosed},
					"DELETE /openApi/swap/v2/trade/order": {wrapperCredentialsChange + wrapperNoReply},
					"POST /openApi/swap/v2/trade/order":   {wrapperSwapPlacedReply("LIMIT", "")},
				},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex, requests := wrapperTestServer(t, tc.routes)
				action := tc.action
				action.Exchange, action.Pair, action.Price = ex.Name, currency.NewBTCUSDT(), 11
				_, err := ex.ModifyOrder(t.Context(), &action)
				require.NoError(t, err, "ModifyOrder must replace the order")
				got := requests()
				require.NotEmpty(t, got, "ModifyOrder must send its requests")
				last := got[len(got)-1]
				assert.Equal(t, http.MethodPost, last.method, "ModifyOrder should place the replacement last")
				assert.True(t, strings.HasSuffix(last.path, "/trade/order"), "ModifyOrder should place the replacement last")
				for _, r := range got {
					assert.Equalf(t, testAPIKey, r.apiKey, "ModifyOrder should send %s %s for the account it started with", r.method, r.path)
				}
			})
		}
	})
}

func TestModifiedOrderID(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		action order.Modify
		routes map[string][]string
		exp    uint64
		// lookup holds the parameters of the lookup by client order ID, or nil when nothing is looked up
		lookup url.Values
		err    error
	}{
		{name: "order ID", action: order.Modify{AssetType: asset.Spot, OrderID: "16", ClientOrderID: "c11"}, exp: 16},
		{name: "invalid order ID", action: order.Modify{AssetType: asset.Spot, OrderID: "x"}, err: errInvalidOrderID},
		{
			name: "spot client order ID", action: order.Modify{AssetType: asset.Spot, ClientOrderID: "c11"},
			routes: map[string][]string{"GET /openApi/spot/v1/trade/query": {wrapperSpotOrderReply}}, exp: 11,
			lookup: url.Values{"symbol": {"BTC-USDT"}, "clientOrderID": {"c11"}},
		},
		{
			name: "zero order ID with a client order ID", action: order.Modify{AssetType: asset.Spot, OrderID: "0", ClientOrderID: "c11"},
			routes: map[string][]string{"GET /openApi/spot/v1/trade/query": {wrapperSpotOrderReply}}, exp: 11,
			lookup: url.Values{"symbol": {"BTC-USDT"}, "clientOrderID": {"c11"}},
		},
		{
			name: "USDT-M client order ID", action: order.Modify{AssetType: asset.USDTMarginedFutures, ClientOrderID: "c21"},
			routes: map[string][]string{"GET /openApi/swap/v2/trade/order": {wrapperSwapOrderReply("")}}, exp: 21,
			lookup: url.Values{"symbol": {"BTC-USDT"}, "clientOrderId": {"c21"}},
		},
		{
			name: "spot lookup error", action: order.Modify{AssetType: asset.Spot, ClientOrderID: "c11"},
			routes: map[string][]string{"GET /openApi/spot/v1/trade/query": {`{"code":100404,"msg":"order not exist"}`}},
			lookup: url.Values{"symbol": {"BTC-USDT"}, "clientOrderID": {"c11"}}, err: order.ErrOrderNotFound,
		},
		{
			name: "USDT-M lookup error", action: order.Modify{AssetType: asset.USDTMarginedFutures, ClientOrderID: "c21"},
			routes: map[string][]string{"GET /openApi/swap/v2/trade/order": {`{"code":109421,"msg":"order not exist"}`}},
			lookup: url.Values{"symbol": {"BTC-USDT"}, "clientOrderId": {"c21"}}, err: order.ErrOrderNotFound,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex, requests := wrapperTestServer(t, tc.routes)
			action := tc.action
			action.Pair = currency.NewBTCUSDT()
			orderID, err := ex.modifiedOrderID(t.Context(), &action)
			require.ErrorIs(t, err, tc.err, "modifiedOrderID must return the expected error")
			assert.Equal(t, tc.exp, orderID, "modifiedOrderID should return the order ID of the order the modification names")
			got := requests()
			if tc.lookup == nil {
				assert.Empty(t, got, "modifiedOrderID should look nothing up for a modification naming an order ID")
				return
			}
			require.Len(t, got, 1, "modifiedOrderID must look the order up once")
			assert.Equal(t, tc.lookup, got[0].params, "modifiedOrderID should look the order up by its client order ID")
			if tc.err != nil {
				assert.ErrorContains(t, err, "error fetching order to modify", "modifiedOrderID should say which lookup failed")
			}
		})
	}
}

func TestReplacementQuantity(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name               string
		amount             float64
		original, executed types.Number
		exp                float64
		err                error
	}{
		{name: "existing total less the executed quantity", original: 3, executed: 1, exp: 2},
		{name: "desired total less the executed quantity", amount: 10, original: 10, executed: 4, exp: 6},
		{name: "larger desired total", amount: 12, original: 10, executed: 4, exp: 8},
		{name: "decimal arithmetic", amount: 0.3, original: 0.5, executed: 0.1, exp: 0.2},
		{name: "nothing executed", amount: 2, original: 1, exp: 2},
		{name: "desired total executed", amount: 4, original: 10, executed: 4, err: errModifyAmountNotAboveExecuted},
		{name: "desired total below the executed quantity", amount: 3, original: 10, executed: 4, err: errModifyAmountNotAboveExecuted},
		{name: "existing total executed", original: 1, executed: 1, err: errModifyAmountNotAboveExecuted},
		{name: "negative amount", amount: -1, original: 1, err: order.ErrAmountIsInvalid},
		{name: "NaN amount", amount: math.NaN(), original: 1, err: order.ErrAmountIsInvalid},
		{name: "infinite amount", amount: math.Inf(1), original: 1, err: order.ErrAmountIsInvalid},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			quantity, err := replacementQuantity(tc.amount, tc.original, tc.executed)
			require.ErrorIs(t, err, tc.err, "replacementQuantity must return the expected error")
			assert.Equal(t, tc.exp, quantity, "replacementQuantity should return the desired total less the executed quantity")
		})
	}
}

func TestCancelledReplacementQuantity(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                          string
		amount                        float64
		original, lookedUp, cancelled types.Number
		exp                           float64
		err                           error
	}{
		{name: "cancellation reports the lookup's execution", amount: 10, original: 10, lookedUp: 4, cancelled: 4, exp: 6},
		{name: "cancellation reports more execution", amount: 10, original: 10, lookedUp: 4, cancelled: 5, exp: 5},
		{name: "cancellation reports less execution", amount: 10, original: 10, lookedUp: 4, exp: 6},
		{name: "existing total", original: 3, lookedUp: 1, cancelled: 2, exp: 1},
		{name: "desired total executed by the cancellation", amount: 10, original: 10, lookedUp: 4, cancelled: 10, err: errCancelledOrderExecuted},
		{name: "more than the desired total executed by the cancellation", amount: 8, original: 10, lookedUp: 4, cancelled: 9, err: errCancelledOrderExecuted},
		{name: "invalid amount", amount: math.NaN(), original: 10, err: order.ErrAmountIsInvalid},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			quantity, err := cancelledReplacementQuantity("16", tc.amount, tc.original, tc.lookedUp, tc.cancelled)
			require.ErrorIs(t, err, tc.err, "cancelledReplacementQuantity must return the expected error")
			assert.Equal(t, tc.exp, quantity, "cancelledReplacementQuantity should return the desired total less the larger executed quantity")
		})
	}
	_, err := cancelledReplacementQuantity("16", 10, 10, 4, 10)
	assert.ErrorContains(t, err, "order 16 had executed 10 when cancelled, and no replacement was placed", "cancelledReplacementQuantity should name the order and its final executed quantity")
	_, err = cancelledReplacementQuantity("16", math.NaN(), 10, 0, 0)
	assert.NotErrorIs(t, err, errCancelledOrderExecuted, "cancelledReplacementQuantity should not report an invalid amount as an order cancelled without a replacement")
}

func TestCancellationRefused(t *testing.T) {
	t.Parallel()
	// badStatus returns the error the requester returns for a reply with an HTTP status other than 2xx
	badStatus := func(status int) error {
		return fmt.Errorf("BingX %w: %d raw response: <html></html>", request.ErrBadStatus, status)
	}
	for _, tc := range []struct {
		name    string
		err     error
		refused bool
	}{
		{name: "rate limit wait past the deadline", err: fmt.Errorf("%w: %w", errRequestNotSent, fmt.Errorf("rate limit delay of 1h0m0s will exceed deadline: %w", context.DeadlineExceeded)), refused: true},
		{name: "400 page", err: badStatus(http.StatusBadRequest), refused: true},
		{name: "403 page", err: badStatus(http.StatusForbidden), refused: true},
		{name: "418 page", err: badStatus(http.StatusTeapot), refused: true},
		{name: "429 page", err: badStatus(http.StatusTooManyRequests), refused: true},
		{name: "500 page", err: badStatus(http.StatusInternalServerError)},
		{name: "502 page", err: badStatus(http.StatusBadGateway)},
		{name: "504 page", err: badStatus(http.StatusGatewayTimeout)},
		{name: "spot order not found", err: fmt.Errorf("%w: %w", order.ErrOrderNotFound, &APIError{Code: 100404, Message: "order not exist"}), refused: true},
		{name: "USDT-M order not found", err: &APIError{Code: 109421, Message: "order not exist"}, refused: true},
		{name: "parameter error on an error status", err: fmt.Errorf("%w: %w", &APIError{Code: 109400, Message: "Invalid parameters"}, request.ErrBadStatus), refused: true},
		{name: "system busy", err: &APIError{Code: 100500, Message: "System busy"}},
		{name: "internal server error", err: &APIError{Code: 109500, Message: "Internal server error"}},
		{name: "network request exception", err: &APIError{Code: 109701, Message: "Network request exception"}},
		{name: "order system busy", err: &APIError{Code: 110500, Message: "Order system busy"}},
		{name: "internal server error on an error status", err: fmt.Errorf("%w: %w", &APIError{Code: 109500, Message: "Internal server error"}, request.ErrBadStatus)},
		{name: "connection reset", err: errors.New("connection reset")},
		{name: "timeout", err: fmt.Errorf("request timed out: %w", context.DeadlineExceeded)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.refused, cancellationRefused(tc.err), "cancellationRefused should report only an answer showing the cancellation did nothing")
		})
	}
}

func TestFailedCancellationExecuted(t *testing.T) {
	t.Parallel()
	// Spot wraps the API error BingX refuses a cancellation with, as for an order not found
	refusal := fmt.Errorf("%w: %w", order.ErrOrderNotFound, &APIError{Code: 100404, Message: "order not exist"})
	noAnswer, lookupErr := errors.New("connection reset"), errors.New("lookup refused")
	type testCase struct {
		name, status  string
		cancelErr     error
		lookupErr     error
		exp           types.Number
		errs, notErrs []error
	}
	cases := []testCase{
		{name: "no answer, spot order cancelled", status: "CANCELED", cancelErr: noAnswer, exp: 2},
		{name: "no answer, USDT-M order cancelled", status: "CANCELLED", cancelErr: noAnswer, exp: 2},
		{name: "no answer, order expired", status: "EXPIRED", cancelErr: noAnswer, exp: 2},
		{name: "no answer, order filled", status: "FILLED", cancelErr: noAnswer, errs: []error{errCancelledOrderExecuted, noAnswer}, notErrs: []error{errCancelOutcomeUnknown}},
		{name: "no answer, new order", status: "NEW", cancelErr: noAnswer, errs: []error{errCancelReplaceFailed, errCancelOutcomeUnknown, noAnswer}},
		{name: "no answer, pending order", status: "PENDING", cancelErr: noAnswer, errs: []error{errCancelReplaceFailed, errCancelOutcomeUnknown, noAnswer}},
		{name: "no answer, partially filled order", status: "PARTIALLY_FILLED", cancelErr: noAnswer, errs: []error{errCancelReplaceFailed, errCancelOutcomeUnknown, noAnswer}},
		{name: "no answer, unknown status", status: "Unknown", cancelErr: noAnswer, errs: []error{errCancelReplaceFailed, errCancelOutcomeUnknown, noAnswer}},
		{name: "no answer, status that cannot be converted", status: "ODD", cancelErr: noAnswer, errs: []error{errCancelReplaceFailed, errCancelOutcomeUnknown, noAnswer}},
		{name: "no answer, failed lookup", cancelErr: noAnswer, lookupErr: lookupErr, errs: []error{errCancelReplaceFailed, errCancelOutcomeUnknown, noAnswer, lookupErr}},
		{name: "refusal, spot order cancelled", status: "CANCELED", cancelErr: refusal, errs: []error{errOrderNotOpen, errAPIResponse, order.ErrOrderNotFound}, notErrs: []error{errCancelReplaceFailed, errCancelOutcomeUnknown}},
		{name: "refusal, USDT-M order cancelled", status: "CANCELLED", cancelErr: refusal, errs: []error{errOrderNotOpen, errAPIResponse}, notErrs: []error{errCancelReplaceFailed, errCancelOutcomeUnknown}},
		{name: "refusal, order expired", status: "EXPIRED", cancelErr: refusal, errs: []error{errOrderNotOpen, errAPIResponse}, notErrs: []error{errCancelReplaceFailed, errCancelOutcomeUnknown}},
		{name: "refusal, order filled", status: "FILLED", cancelErr: refusal, errs: []error{errCancelledOrderExecuted, errAPIResponse}, notErrs: []error{errOrderNotOpen, errCancelOutcomeUnknown}},
		{name: "refusal, new order", status: "NEW", cancelErr: refusal, errs: []error{errCancelReplaceFailed, errAPIResponse}, notErrs: []error{errCancelOutcomeUnknown}},
		{name: "refusal, pending order", status: "PENDING", cancelErr: refusal, errs: []error{errCancelReplaceFailed, errAPIResponse}, notErrs: []error{errCancelOutcomeUnknown}},
		{name: "refusal, partially filled order", status: "PARTIALLY_FILLED", cancelErr: refusal, errs: []error{errCancelReplaceFailed, errAPIResponse}, notErrs: []error{errCancelOutcomeUnknown}},
		{name: "refusal, unknown status", status: "Unknown", cancelErr: refusal, errs: []error{errCancelReplaceFailed, errAPIResponse}, notErrs: []error{errCancelOutcomeUnknown}},
		{name: "refusal, status that cannot be converted", status: "ODD", cancelErr: refusal, errs: []error{errCancelReplaceFailed, errAPIResponse}, notErrs: []error{errCancelOutcomeUnknown}},
		{name: "refusal, failed lookup", cancelErr: refusal, lookupErr: lookupErr, errs: []error{errCancelReplaceFailed, errCancelOutcomeUnknown, errAPIResponse, lookupErr}},
	}
	codes := []int64{100500, 109500, 109701, 110500}
	busy := make([]testCase, 0, 4*len(codes))
	for _, code := range codes {
		failure := &APIError{Code: code, Message: "busy"}
		name := "code " + strconv.FormatInt(code, 10)
		busy = append(busy,
			testCase{name: name + ", order cancelled", status: "CANCELLED", cancelErr: failure, exp: 2},
			testCase{name: name + ", order filled", status: "FILLED", cancelErr: failure, errs: []error{errCancelledOrderExecuted, failure}, notErrs: []error{errOrderNotOpen, errCancelOutcomeUnknown}},
			testCase{name: name + ", order still open", status: "NEW", cancelErr: failure, errs: []error{errCancelReplaceFailed, errCancelOutcomeUnknown, failure}},
			testCase{name: name + ", status that cannot be converted", status: "ODD", cancelErr: failure, errs: []error{errCancelReplaceFailed, errCancelOutcomeUnknown, failure}},
		)
	}
	for _, tc := range append(cases, busy...) {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			executed, err := failedCancellationExecuted("16", tc.status, 2, tc.cancelErr, tc.lookupErr)
			assert.Equal(t, tc.exp, executed, "failedCancellationExecuted should return the executed quantity of an order a cancellation without an answer may have cancelled")
			if len(tc.errs) == 0 {
				assert.NoError(t, err, "failedCancellationExecuted should not error for an order a cancellation without an answer may have cancelled")
				return
			}
			for _, target := range tc.errs {
				assert.ErrorIs(t, err, target, "failedCancellationExecuted should return the outcome the cancellation's answer and the order's status show")
			}
			for _, target := range tc.notErrs {
				assert.NotErrorIs(t, err, target, "failedCancellationExecuted should not report another outcome")
			}
		})
	}

	for _, tc := range []struct {
		name, status string
		cancelErr    error
		lookupErr    error
		exp          string
	}{
		{name: "no answer, order still open", status: "NEW", cancelErr: noAnswer, exp: "cancelling order 16 got no answer showing its outcome while the order is still open, so no replacement was placed, although the cancellation may still land"},
		{name: "no answer, unreadable status", status: "ODD", cancelErr: noAnswer, exp: `cancelling order 16 got no answer showing its outcome, and its status "ODD" does not show whether it was cancelled`},
		{name: "busy code, order still open", status: "NEW", cancelErr: &APIError{Code: 109500, Message: "Internal server error"}, exp: "cancelling order 16 got no answer showing its outcome while the order is still open, so no replacement was placed, although the cancellation may still land: BingX API error: code 109500 msg Internal server error"},
		{name: "refusal, order closed", status: "CANCELED", cancelErr: refusal, exp: "cancelling order 16 was refused, and the order is CANCELLED, so no replacement was placed"},
		{name: "refusal, order still open", status: "NEW", cancelErr: refusal, exp: "cancelling order 16 was refused, so no replacement was placed"},
		{name: "refusal, unreadable status", status: "ODD", cancelErr: refusal, exp: `cancelling order 16 was refused, and its status "ODD" does not show whether it is still open`},
		{name: "failed lookup", cancelErr: refusal, lookupErr: lookupErr, exp: "cancelling order 16 failed, and so did looking it up again"},
	} {
		t.Run(tc.name+" message", func(t *testing.T) {
			t.Parallel()
			_, err := failedCancellationExecuted("16", tc.status, 2, tc.cancelErr, tc.lookupErr)
			assert.ErrorContains(t, err, tc.exp, "failedCancellationExecuted should name the order and its outcome")
		})
	}
}

func TestOrderOpenToModify(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, status string
		closed       bool
	}{
		{name: "new order", status: "NEW"},
		{name: "pending order", status: "PENDING"},
		{name: "partially filled spot order", status: "PARTIALLY_FILLED"},
		{name: "partially filled USDT-M order", status: "PARTIALLYFILLED"},
		{name: "unknown status", status: "Unknown"},
		{name: "status that cannot be converted", status: "ODD"},
		{name: "spot order cancelled", status: "CANCELED", closed: true},
		{name: "USDT-M order cancelled", status: "CANCELLED", closed: true},
		{name: "order filled", status: "FILLED", closed: true},
		{name: "order expired", status: "EXPIRED", closed: true},
		{name: "order failed", status: "FAILED", closed: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := orderOpenToModify("16", tc.status)
			if !tc.closed {
				assert.NoError(t, err, "orderOpenToModify should not error for an order its lookup does not show closed")
				return
			}
			assert.ErrorIs(t, err, errOrderNotOpen, "orderOpenToModify should refuse an order its lookup shows closed")
			assert.ErrorContains(t, err, "order 16", "orderOpenToModify should name the order")
		})
	}
}

func TestCancelSpotOrderForReplacement(t *testing.T) {
	t.Parallel()
	cancelledLookup := `{"code":0,"data":{"symbol":"BTC-USDT","orderId":11,"price":"10","origQty":"3","executedQty":"2","status":"CANCELED","type":"LIMIT","side":"BUY"}}`
	notFound := `{"code":100404,"msg":"order not exist"}`
	ex, requests := wrapperTestServer(t, map[string][]string{
		"POST /openApi/spot/v1/trade/cancel": {wrapperSpotCancelledReply(11, "1"), wrapperNoReply, notFound, notFound},
		"GET /openApi/spot/v1/trade/query":   {cancelledLookup, cancelledLookup, `{"code":100001,"msg":"signature"}`},
	})
	executed, err := ex.cancelSpotOrderForReplacement(t.Context(), currency.NewBTCUSDT(), 11)
	require.NoError(t, err, "cancelSpotOrderForReplacement must not error")
	assert.Equal(t, types.Number(1), executed, "cancelSpotOrderForReplacement should return the quantity the cancellation reports executed")
	assert.Len(t, requests(), 1, "cancelSpotOrderForReplacement should not look up an order its cancellation cancelled")

	executed, err = ex.cancelSpotOrderForReplacement(t.Context(), currency.NewBTCUSDT(), 11)
	require.NoError(t, err, "cancelSpotOrderForReplacement must not error for an order its lookup shows cancelled after a cancellation BingX did not answer")
	assert.Equal(t, types.Number(2), executed, "cancelSpotOrderForReplacement should return the quantity the lookup after a cancellation without an answer reports executed")
	got := requests()
	require.Len(t, got, 3, "cancelSpotOrderForReplacement must look the order up again after a cancellation without an answer")
	assert.Equal(t, url.Values{"symbol": {"BTC-USDT"}, "orderId": {"11"}}, got[2].params, "cancelSpotOrderForReplacement should look the order up by its order ID")

	_, err = ex.cancelSpotOrderForReplacement(t.Context(), currency.NewBTCUSDT(), 11)
	assert.ErrorIs(t, err, errOrderNotOpen, "cancelSpotOrderForReplacement should report an order its lookup shows cancelled after BingX refused the cancellation as no longer open")
	assert.ErrorIs(t, err, order.ErrOrderNotFound, "cancelSpotOrderForReplacement should return the cancellation's error")

	_, err = ex.cancelSpotOrderForReplacement(t.Context(), currency.NewBTCUSDT(), 11)
	assert.ErrorIs(t, err, errCancelOutcomeUnknown, "cancelSpotOrderForReplacement should report the outcome as unknown when the lookup fails")
	assert.ErrorIs(t, err, order.ErrOrderNotFound, "cancelSpotOrderForReplacement should return the cancellation's error")
	assert.ErrorIs(t, err, request.ErrAuthRequestFailed, "cancelSpotOrderForReplacement should return the lookup's error")

	t.Run("cancellation that times out", func(t *testing.T) {
		t.Parallel()
		var cancellations atomic.Int32
		ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			// The server notices the client giving up, which ends the request's context, only once the body is read
			_, err := io.Copy(io.Discard, r.Body)
			assert.NoError(t, err, "reading the request body should not error")
			var reply string
			switch route := r.Method + " " + r.URL.Path; route {
			case "POST /openApi/spot/v1/trade/cancel":
				if cancellations.Add(1) == 1 {
					select {
					case <-r.Context().Done():
					case <-time.After(5 * time.Second):
					}
					return
				}
				// A cancellation sent again finds the order its first cancelled gone
				reply = notFound
			case "GET /openApi/spot/v1/trade/query":
				reply = cancelledLookup
			default:
				assert.Failf(t, "unexpected request", "%s should not be requested", route)
				w.WriteHeader(http.StatusNotFound)
				return
			}
			_, err = io.WriteString(w, reply)
			assert.NoError(t, err, "WriteString should not error")
		})
		// Long enough for a request to arrive while the package's parallel tests load the machine
		require.NoError(t, ex.SetHTTPClientTimeout(time.Second), "SetHTTPClientTimeout must not error")
		executed, err := ex.cancelSpotOrderForReplacement(t.Context(), currency.NewBTCUSDT(), 11)
		require.NoError(t, err, "cancelSpotOrderForReplacement must not error for an order its lookup shows cancelled after the cancellation timed out")
		assert.Equal(t, types.Number(2), executed, "cancelSpotOrderForReplacement should return the quantity the lookup reports executed")
		assert.Equal(t, int32(1), cancellations.Load(), "cancelSpotOrderForReplacement should send a cancellation that timed out once")
	})
}

// wrapperFailedCancellationCase is a cancel-replace whose cancellation fails: the cancellation's reply, the reply to the
// lookup that follows it, and the quantity the replacement places, or else the errors the cancel-replace returns
type wrapperFailedCancellationCase struct {
	name, cancel, lookup, quantity string
	errs, notErrs                  []error
}

// wrapperRateLimited stands for a cancellation the rate limiter refuses before sending it, which
// wrapperRefuseByRateLimit sets up
const wrapperRateLimited = "rate limited"

// wrapperRefuseByRateLimit turns the exchange's rate limiter on with the budget of endpoint f, one request an hour,
// spent, and returns a context whose deadline the endpoint's next request cannot meet, so the limiter refuses that
// request without sending it while other requests go through
func wrapperRefuseByRateLimit(t *testing.T, ex *Exchange, f request.EndpointLimit) context.Context {
	t.Helper()
	ex.Requester.GetRateLimiterDefinitions()[f] = request.NewRateLimitWithWeight(time.Hour, 1, 1)
	require.NoError(t, ex.EnableRateLimiter(), "EnableRateLimiter must not error")
	require.NoError(t, ex.Requester.InitiateRateLimit(t.Context(), f), "InitiateRateLimit must spend the endpoint's budget")
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	t.Cleanup(cancel)
	return ctx
}

// wrapperSendFailureCases returns cases of cancellations that fail before reaching BingX's order system, made from the
// refusal and no answer cases given with each lookup outcome. A rate limit wait past the deadline, which sends nothing,
// and a 403 page, which turns the cancellation away, end as a refusal does, while a 504 page leaves the outcome
// unknown, as no answer does
func wrapperSendFailureCases(cases []wrapperFailedCancellationCase) []wrapperFailedCancellationCase {
	var derived []wrapperFailedCancellationCase
	for _, failure := range []struct {
		name, cancel, like string
		err                error
	}{
		{name: "rate limit wait past the deadline", cancel: wrapperRateLimited, like: "refusal", err: errRequestNotSent},
		{name: "403 page", cancel: wrapperStatusReply(http.StatusForbidden, "<html><title>403 Forbidden</title></html>"), like: "refusal", err: request.ErrBadStatus},
		{name: "504 page", cancel: wrapperStatusReply(http.StatusGatewayTimeout, "<html><title>504 Gateway Time-out</title></html>"), like: "no answer", err: request.ErrBadStatus},
	} {
		for _, tc := range cases {
			outcome, ok := strings.CutPrefix(tc.name, failure.like+", ")
			if !ok {
				continue
			}
			tc.name, tc.cancel = failure.name+", "+outcome, failure.cancel
			errs := make([]error, len(tc.errs))
			for i, err := range tc.errs {
				// The refusal cases' order not found error is the code BingX refused them with, in whose place these
				// failures return their own
				if errors.Is(err, order.ErrOrderNotFound) {
					err = failure.err
				}
				errs[i] = err
			}
			tc.errs = errs
			derived = append(derived, tc)
		}
	}
	return derived
}

func TestReplaceSpotOrder(t *testing.T) {
	t.Parallel()
	pair := currency.NewBTCUSDT()
	ex, requests := wrapperTestServer(t, map[string][]string{
		"GET /openApi/spot/v1/trade/query": {
			wrapperSpotOrderReply,
			`{"code":0,"data":{"symbol":"BTC-USDT","orderId":13,"price":"0","stopPrice":"9","origQty":"1","executedQty":"0","status":"NEW","type":"TAKE_STOP_MARKET","side":"SELL"}}`,
		},
		"POST /openApi/spot/v1/trade/cancel": {wrapperSpotCancelledReply(11, "1"), wrapperSpotCancelledReply(13, "0")},
		"POST /openApi/spot/v1/trade/order": {
			wrapperSpotPlacedReply,
			`{"code":0,"data":{"symbol":"BTC-USDT","orderId":14,"transactTime":1700000000003,"stopPrice":"8","origQty":"1","executedQty":"0","status":"NEW","type":"TAKE_STOP_MARKET","side":"SELL"}}`,
		},
	})
	resp, err := ex.replaceSpotOrder(t.Context(), &order.Modify{Exchange: ex.Name, Pair: pair, AssetType: asset.Spot, OrderID: "11", Price: 11, NewClientOrderID: "n12"}, 11)
	require.NoError(t, err, "replaceSpotOrder must not error")
	got := requests()
	require.Len(t, got, 3, "replaceSpotOrder must fetch the order, cancel it and then place its replacement")
	assert.Equal(t, url.Values{"symbol": {"BTC-USDT"}, "orderId": {"11"}}, got[0].params, "replaceSpotOrder should fetch the order")
	assert.Equal(t, "/openApi/spot/v1/trade/cancel", got[1].path, "replaceSpotOrder should cancel the order before placing its replacement")
	assert.Equal(t, url.Values{"symbol": {"BTC-USDT"}, "orderId": {"11"}}, got[1].params, "replaceSpotOrder should cancel the order by its order ID")
	assert.Equal(t, url.Values{
		"symbol":           {"BTC-USDT"},
		"side":             {"BUY"},
		"type":             {"LIMIT"},
		"price":            {"11"},
		"quantity":         {"2"},
		"newClientOrderId": {"n12"},
	}, got[2].params, "replaceSpotOrder should place the order's unfilled amount, keeping what the modification leaves unchanged")
	assert.Equal(t, "12", resp.OrderID, "replaceSpotOrder should return the replacement's order ID")
	assert.Equal(t, "n12", resp.ClientOrderID, "replaceSpotOrder should return the replacement's client order ID")
	assert.Equal(t, 11.0, resp.Price, "replaceSpotOrder should return the replacement's price")
	assert.Equal(t, 2.0, resp.Amount, "replaceSpotOrder should return the replacement's amount")
	assert.Equal(t, 2.0, resp.RemainingAmount, "replaceSpotOrder should return the replacement's unfilled amount")
	assert.Equal(t, order.Pending, resp.Status, "replaceSpotOrder should return the replacement's status")
	assert.Equal(t, order.Limit, resp.Type, "replaceSpotOrder should return the replacement's type")
	assert.Equal(t, order.Buy, resp.Side, "replaceSpotOrder should return the replacement's side")
	assert.Equal(t, order.UnknownTIF, resp.TimeInForce, "replaceSpotOrder should report no time in force for a modification without one, as spot orders report none")
	assert.Equal(t, time.UnixMilli(1700000000002), resp.LastUpdated, "replaceSpotOrder should date the replacement by its placement")

	resp, err = ex.replaceSpotOrder(t.Context(), &order.Modify{Exchange: ex.Name, Pair: pair, AssetType: asset.Spot, OrderID: "13", TriggerPrice: 8}, 13)
	require.NoError(t, err, "replaceSpotOrder must not error")
	assert.Equal(t, url.Values{"symbol": {"BTC-USDT"}, "side": {"SELL"}, "type": {"TAKE_STOP_MARKET"}, "stopPrice": {"8"}, "quantity": {"1"}}, requests()[5].params, "replaceSpotOrder should replace a stop market order without a price")
	assert.Equal(t, 8.0, resp.TriggerPrice, "replaceSpotOrder should return the replacement's trigger price")

	t.Run("carried values", func(t *testing.T) {
		t.Parallel()
		existing := `{"code":0,"data":{"symbol":"BTC-USDT","orderId":15,"price":"12","stopPrice":"11","origQty":"3","executedQty":"1","status":"PARTIALLY_FILLED","type":"TRIGGER_LIMIT","side":"SELL"}}`
		placed := `{"code":0,"data":{"symbol":"BTC-USDT","orderId":16,"price":"13","stopPrice":"12","origQty":"2","executedQty":"0","status":"NEW","type":"TRIGGER_LIMIT","side":"SELL"}}`
		ex, requests := wrapperTestServer(t, map[string][]string{
			"GET /openApi/spot/v1/trade/query":   {existing, existing},
			"POST /openApi/spot/v1/trade/cancel": {wrapperSpotCancelledReply(15, "1"), wrapperSpotCancelledReply(15, "1")},
			"POST /openApi/spot/v1/trade/order":  {placed, placed},
		})
		for _, tc := range []struct {
			name   string
			action order.Modify
			exp    url.Values
		}{
			{
				name:   "changed values",
				action: order.Modify{Price: 13, TriggerPrice: 12, NewClientOrderID: "n16", TimeInForce: order.ImmediateOrCancel},
				exp:    url.Values{"symbol": {"BTC-USDT"}, "side": {"SELL"}, "type": {"TRIGGER_LIMIT"}, "price": {"13"}, "stopPrice": {"12"}, "quantity": {"2"}, "newClientOrderId": {"n16"}, "timeInForce": {"IOC"}},
			},
			{
				name:   "values the modification leaves unchanged",
				action: order.Modify{Amount: 4, TimeInForce: order.PostOnly},
				exp:    url.Values{"symbol": {"BTC-USDT"}, "side": {"SELL"}, "type": {"TRIGGER_LIMIT"}, "price": {"12"}, "stopPrice": {"11"}, "quantity": {"3"}, "timeInForce": {"PostOnly"}},
			},
		} {
			// The rows share the server's queued replies, so they run one at a time in order
			t.Run(tc.name, func(t *testing.T) {
				action := tc.action
				action.Exchange, action.Pair, action.AssetType, action.OrderID = ex.Name, pair, asset.Spot, "15"
				resp, err := ex.replaceSpotOrder(t.Context(), &action, 15)
				require.NoErrorf(t, err, "replaceSpotOrder must not error for %s", tc.name)
				got := requests()
				assert.Equalf(t, tc.exp, got[len(got)-1].params, "replaceSpotOrder should place a replacement carrying the %s", tc.name)
				assert.Equalf(t, tc.action.TimeInForce, resp.TimeInForce, "replaceSpotOrder should return the time in force it sent for %s", tc.name)
			})
		}
	})

	t.Run("desired total", func(t *testing.T) {
		t.Parallel()
		existing := `{"code":0,"data":{"symbol":"BTC-USDT","orderId":16,"price":"10","stopPrice":"0","origQty":"10","executedQty":"4","status":"PARTIALLY_FILLED","type":"LIMIT","side":"BUY"}}`
		placed := func(quantity string) string {
			return `{"code":0,"data":{"symbol":"BTC-USDT","orderId":17,"price":"11","origQty":"` + quantity + `","executedQty":"0","status":"PENDING","type":"LIMIT","side":"BUY"}}`
		}
		ex, requests := wrapperTestServer(t, map[string][]string{
			"GET /openApi/spot/v1/trade/query":   {existing, existing, existing, existing},
			"POST /openApi/spot/v1/trade/cancel": {wrapperSpotCancelledReply(16, "4"), wrapperSpotCancelledReply(16, "5"), wrapperSpotCancelledReply(16, "10")},
			"POST /openApi/spot/v1/trade/order":  {placed("6"), placed("5")},
		})
		modify := order.Modify{Exchange: ex.Name, Pair: pair, AssetType: asset.Spot, OrderID: "16", Price: 11, Amount: 10}
		resp, err := ex.replaceSpotOrder(t.Context(), &modify, 16)
		require.NoError(t, err, "replaceSpotOrder must not error")
		got := requests()
		require.Len(t, got, 3, "replaceSpotOrder must fetch the order, cancel it and then place its replacement")
		assert.Equal(t, "6", got[2].params.Get("quantity"), "replaceSpotOrder should place the desired total less the executed quantity")
		assert.Equal(t, 6.0, resp.Amount, "replaceSpotOrder should return the replacement's amount")

		resp, err = ex.replaceSpotOrder(t.Context(), &modify, 16)
		require.NoError(t, err, "replaceSpotOrder must not error when the order executed more before its cancellation")
		got = requests()
		require.Len(t, got, 6, "replaceSpotOrder must fetch the order, cancel it and then place its replacement")
		assert.Equal(t, "5", got[5].params.Get("quantity"), "replaceSpotOrder should size the replacement by the quantity the cancellation reports executed")
		assert.Equal(t, 5.0, resp.Amount, "replaceSpotOrder should return the replacement's amount")

		_, err = ex.replaceSpotOrder(t.Context(), &modify, 16)
		assert.ErrorIs(t, err, errCancelledOrderExecuted, "replaceSpotOrder should report an order whose cancellation executed its desired total")
		got = requests()
		require.Len(t, got, 8, "replaceSpotOrder must fetch and cancel the order")
		assert.Equal(t, "/openApi/spot/v1/trade/cancel", got[7].path, "replaceSpotOrder should place nothing for an order that executed its desired total")

		modify.Amount = 4
		_, err = ex.replaceSpotOrder(t.Context(), &modify, 16)
		assert.ErrorIs(t, err, errModifyAmountNotAboveExecuted, "replaceSpotOrder should refuse a desired total the order has executed")
		assert.Len(t, requests(), 9, "replaceSpotOrder should refuse the modification before cancelling the order")
	})

	t.Run("errors", func(t *testing.T) {
		t.Parallel()
		ex, requests := wrapperTestServer(t, map[string][]string{
			// The third lookup follows the failed cancellation, and shows the order still open
			"GET /openApi/spot/v1/trade/query":   {`{"code":100500,"msg":"busy"}`, wrapperSpotOrderReply, wrapperSpotOrderReply, wrapperSpotOrderReply},
			"POST /openApi/spot/v1/trade/cancel": {`{"code":100500,"msg":"busy"}`, wrapperSpotCancelledReply(11, "1")},
			"POST /openApi/spot/v1/trade/order":  {`{"code":100500,"msg":"busy"}`},
		})
		action := &order.Modify{Exchange: ex.Name, Pair: pair, AssetType: asset.Spot, OrderID: "11", Price: 11}
		_, err := ex.replaceSpotOrder(t.Context(), &order.Modify{Pair: pair, AssetType: asset.Spot, OrderID: "11", TimeInForce: order.GoodTillDay}, 11)
		assert.ErrorIs(t, err, order.ErrUnsupportedTimeInForce, "replaceSpotOrder should reject a time in force BingX does not take")
		assert.Empty(t, requests(), "replaceSpotOrder should send nothing for a modification it rejects")
		_, err = ex.replaceSpotOrder(t.Context(), action, 11)
		assert.ErrorIs(t, err, errAPIResponse, "replaceSpotOrder should return the order request's error")
		_, err = ex.replaceSpotOrder(t.Context(), action, 11)
		assert.ErrorIs(t, err, errAPIResponse, "replaceSpotOrder should return the cancellation's error")
		assert.ErrorIs(t, err, errCancelReplaceFailed, "replaceSpotOrder should report a failed cancellation")
		assert.ErrorIs(t, err, errCancelOutcomeUnknown, "replaceSpotOrder should report a cancellation BingX found busy as one whose outcome is unknown")
		assert.ErrorContains(t, err, "cancelling order 11 got no answer showing its outcome while the order is still open, so no replacement was placed", "replaceSpotOrder should report that nothing was placed")
		_, err = ex.replaceSpotOrder(t.Context(), action, 11)
		assert.ErrorIs(t, err, errAPIResponse, "replaceSpotOrder should return the placement's error")
		assert.ErrorIs(t, err, errCancelReplaceFailed, "replaceSpotOrder should report a failed replacement")
		assert.ErrorContains(t, err, "order 11 was cancelled, but placing its replacement failed", "replaceSpotOrder should report the order cancelled without its replacement")
		var placements int
		for _, r := range requests() {
			if r.path == "/openApi/spot/v1/trade/order" {
				placements++
			}
		}
		assert.Equal(t, 1, placements, "replaceSpotOrder should place a replacement only after cancelling the order")
	})

	t.Run("order no longer open", func(t *testing.T) {
		t.Parallel()
		ex, requests := wrapperTestServer(t, map[string][]string{
			"GET /openApi/spot/v1/trade/query": {`{"code":0,"data":{"symbol":"BTC-USDT","orderId":11,"price":"10","origQty":"3","executedQty":"1","status":"CANCELED","type":"LIMIT","side":"BUY"}}`},
		})
		_, err := ex.replaceSpotOrder(t.Context(), &order.Modify{Exchange: ex.Name, Pair: pair, AssetType: asset.Spot, OrderID: "11", Price: 11}, 11)
		assert.ErrorIs(t, err, errOrderNotOpen, "replaceSpotOrder should refuse an order its lookup shows cancelled")
		assert.Len(t, requests(), 1, "replaceSpotOrder should neither cancel nor replace an order no longer open")
	})

	t.Run("failed cancellation", func(t *testing.T) {
		t.Parallel()
		lookedUp := func(status, executed string) string {
			return `{"code":0,"data":{"symbol":"BTC-USDT","orderId":11,"price":"10","stopPrice":"0","origQty":"3","executedQty":"` + executed + `","status":"` + status + `","type":"LIMIT","side":"BUY"}}`
		}
		refusal, failedLookup := `{"code":100404,"msg":"order not exist"}`, `{"code":100001,"msg":"signature"}`
		cases := []wrapperFailedCancellationCase{
			{name: "refusal, order cancelled", cancel: refusal, lookup: lookedUp("CANCELED", "2"), errs: []error{errOrderNotOpen, order.ErrOrderNotFound}},
			{name: "refusal, order cancelled after executing the desired total", cancel: refusal, lookup: lookedUp("CANCELED", "3"), errs: []error{errOrderNotOpen, order.ErrOrderNotFound}},
			{name: "refusal, order filled", cancel: refusal, lookup: lookedUp("FILLED", "3"), errs: []error{errCancelledOrderExecuted, order.ErrOrderNotFound}},
			{name: "refusal, order still open", cancel: refusal, lookup: lookedUp("PARTIALLY_FILLED", "1"), errs: []error{errCancelReplaceFailed, order.ErrOrderNotFound}, notErrs: []error{errCancelOutcomeUnknown}},
			{name: "refusal, status that does not show whether the order is open", cancel: refusal, lookup: lookedUp("ODD", "1"), errs: []error{errCancelReplaceFailed, order.ErrOrderNotFound}, notErrs: []error{errCancelOutcomeUnknown}},
			{name: "refusal, failed lookup", cancel: refusal, lookup: failedLookup, errs: []error{errCancelOutcomeUnknown, order.ErrOrderNotFound, request.ErrAuthRequestFailed}},
			{name: "no answer, order cancelled", cancel: wrapperNoReply, lookup: lookedUp("CANCELED", "2"), quantity: "1"},
			{name: "no answer, order cancelled after executing the desired total", cancel: wrapperNoReply, lookup: lookedUp("CANCELED", "3"), errs: []error{errCancelledOrderExecuted}},
			{name: "no answer, order filled", cancel: wrapperNoReply, lookup: lookedUp("FILLED", "3"), errs: []error{errCancelledOrderExecuted}},
			{name: "no answer, order still open", cancel: wrapperNoReply, lookup: lookedUp("PARTIALLY_FILLED", "1"), errs: []error{errCancelReplaceFailed, errCancelOutcomeUnknown}},
			{name: "no answer, status that does not show whether the order is open", cancel: wrapperNoReply, lookup: lookedUp("ODD", "1"), errs: []error{errCancelReplaceFailed, errCancelOutcomeUnknown}},
			{name: "no answer, failed lookup", cancel: wrapperNoReply, lookup: failedLookup, errs: []error{errCancelOutcomeUnknown, request.ErrAuthRequestFailed}},
		}
		codes := []string{"100500", "109500", "109701", "110500"}
		busy := make([]wrapperFailedCancellationCase, 0, 2*len(codes))
		for _, code := range codes {
			failure := `{"code":` + code + `,"msg":"busy"}`
			busy = append(busy,
				wrapperFailedCancellationCase{name: "code " + code + ", order cancelled", cancel: failure, lookup: lookedUp("CANCELED", "2"), quantity: "1"},
				wrapperFailedCancellationCase{name: "code " + code + ", order still open", cancel: failure, lookup: lookedUp("PARTIALLY_FILLED", "1"), errs: []error{errCancelReplaceFailed, errCancelOutcomeUnknown, errAPIResponse}},
			)
		}
		for _, tc := range append(append(cases, busy...), wrapperSendFailureCases(cases)...) {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				routes := map[string][]string{"GET /openApi/spot/v1/trade/query": {wrapperSpotOrderReply, tc.lookup}}
				// The order is looked up, cancelled and looked up again, unless the rate limiter refuses the cancellation
				sent := 3
				if tc.cancel == wrapperRateLimited {
					sent = 2
				} else {
					routes["POST /openApi/spot/v1/trade/cancel"] = []string{tc.cancel}
				}
				if tc.quantity != "" {
					routes["POST /openApi/spot/v1/trade/order"] = []string{wrapperSpotPlacedReply}
				}
				ex, requests := wrapperTestServer(t, routes)
				ctx := t.Context()
				if tc.cancel == wrapperRateLimited {
					ctx = wrapperRefuseByRateLimit(t, ex, spotCancelOrderRate)
				}
				resp, err := ex.replaceSpotOrder(ctx, &order.Modify{Exchange: ex.Name, Pair: pair, AssetType: asset.Spot, OrderID: "11", Price: 11}, 11)
				got := requests()
				if tc.quantity == "" {
					require.Error(t, err, "replaceSpotOrder must place nothing for this outcome")
					for _, target := range tc.errs {
						assert.ErrorIs(t, err, target, "replaceSpotOrder should return the outcome the cancellation's answer and the lookup after it show")
					}
					for _, target := range tc.notErrs {
						assert.NotErrorIs(t, err, target, "replaceSpotOrder should not report another outcome")
					}
					assert.Len(t, got, sent, "replaceSpotOrder should look the order up again after a failed cancellation and place nothing")
					return
				}
				require.NoError(t, err, "replaceSpotOrder must replace an order its lookup shows cancelled after a cancellation without an answer showing its outcome")
				require.Len(t, got, sent+1, "replaceSpotOrder must fetch the order, cancel it, look it up again and then place its replacement")
				assert.Equal(t, "/openApi/spot/v1/trade/query", got[sent-1].path, "replaceSpotOrder should look the order up again after a failed cancellation")
				assert.Equal(t, tc.quantity, got[sent].params.Get("quantity"), "replaceSpotOrder should size the replacement by the quantity the lookup reports executed")
				assert.Equal(t, "12", resp.OrderID, "replaceSpotOrder should return the replacement's order ID")
			})
		}
	})

	t.Run("values it cannot convert", func(t *testing.T) {
		t.Parallel()
		ex, _ := wrapperTestServer(t, map[string][]string{
			"GET /openApi/spot/v1/trade/query":   {wrapperSpotOrderReply},
			"POST /openApi/spot/v1/trade/cancel": {wrapperSpotCancelledReply(11, "1")},
			"POST /openApi/spot/v1/trade/order":  {`{"code":0,"data":{"symbol":"BTC-USDT","orderId":15,"origQty":"2","status":"ODD","type":"ODD","side":"ODD"}}`},
		})
		resp, err := ex.replaceSpotOrder(t.Context(), &order.Modify{Exchange: ex.Name, Pair: pair, AssetType: asset.Spot, OrderID: "11", Price: 11}, 11)
		require.NoError(t, err, "replaceSpotOrder must not error for a replacement BingX placed")
		assert.Equal(t, "15", resp.OrderID, "replaceSpotOrder should return the replacement's order ID")
		assert.Equal(t, order.UnknownType, resp.Type, "replaceSpotOrder should leave a type it cannot convert unset")
		assert.Equal(t, order.UnknownSide, resp.Side, "replaceSpotOrder should leave a side it cannot convert unset")
		assert.Equal(t, order.New, resp.Status, "replaceSpotOrder should report a replacement whose status it cannot convert as new, as a submission is")
	})

	t.Run("executed at placement", func(t *testing.T) {
		t.Parallel()
		placed := func(executed, cost, status string) string {
			return `{"code":0,"data":{"symbol":"BTC-USDT","orderId":12,"transactTime":1700000000002,"price":"11","origQty":"2","executedQty":"` + executed + `","cummulativeQuoteQty":"` + cost + `","status":"` + status + `","type":"LIMIT","side":"BUY"}}`
		}
		for _, tc := range []struct {
			name, placed string
			exp          order.Status
			remaining    float64
		}{
			{name: "filled replacement", placed: placed("2", "22", "FILLED"), exp: order.New},
			{name: "replacement cancelled after executing", placed: placed("1", "11", "CANCELED"), exp: order.New, remaining: 1},
			{name: "replacement cancelled without executing", placed: placed("0", "0", "CANCELED"), exp: order.Cancelled, remaining: 2},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex, _ := wrapperTestServer(t, map[string][]string{
					"GET /openApi/spot/v1/trade/query":   {wrapperSpotOrderReply},
					"POST /openApi/spot/v1/trade/cancel": {wrapperSpotCancelledReply(11, "1")},
					"POST /openApi/spot/v1/trade/order":  {tc.placed},
				})
				resp, err := ex.replaceSpotOrder(t.Context(), &order.Modify{Exchange: ex.Name, Pair: pair, AssetType: asset.Spot, OrderID: "11", Price: 11}, 11)
				require.NoError(t, err, "replaceSpotOrder must not error")
				assert.Equal(t, tc.exp, resp.Status, "replaceSpotOrder should report a replacement finished after executing as new, for the order manager to settle it from a later query")
				assert.Equal(t, tc.remaining, resp.RemainingAmount, "replaceSpotOrder should return the replacement's unfilled amount")
			})
		}
	})
}

func TestSwapAmountOnlyChange(t *testing.T) {
	t.Parallel()
	existing := &SwapOrderResponse{OriginalQuantity: 2, Price: 10}
	for _, tc := range []struct {
		action *order.Modify
		exp    bool
	}{
		{&order.Modify{Amount: 3}, true},
		{&order.Modify{Amount: 3, Price: 10, TimeInForce: order.GoodTillCancel}, true},
		{&order.Modify{Amount: 2}, false},
		{&order.Modify{Price: 11}, false},
		{&order.Modify{Amount: 3, Price: 11}, false},
		{&order.Modify{Amount: 3, TriggerPrice: 9}, false},
		{&order.Modify{Amount: 3, TimeInForce: order.PostOnly}, false},
		{&order.Modify{Amount: 3, NewClientOrderID: "n"}, false},
	} {
		assert.Equalf(t, tc.exp, swapAmountOnlyChange(existing, tc.action), "swapAmountOnlyChange should report %+v as changing the amount alone: %t", tc.action, tc.exp)
	}
}

func TestSwapOrderTimeInForce(t *testing.T) {
	t.Parallel()
	assert.Equal(t, order.PostOnly, swapOrderTimeInForce(&SwapOrderResponse{PostOnly: true}), "swapOrderTimeInForce should read a post only order")
	assert.Equal(t, order.GoodTillCancel, swapOrderTimeInForce(&SwapOrderResponse{}), "swapOrderTimeInForce should read another resting order as GTC")
}

func TestSwapAttachedOrder(t *testing.T) {
	t.Parallel()
	assert.Nil(t, swapAttachedOrder(&SwapTakeProfitStopLossResponse{}), "swapAttachedOrder should return nothing without an attached order")
	assert.Equal(t, &SwapTakeProfitStopLossRequest{Type: OrderTypeStopMarket, StopPrice: 9, Price: 8, WorkingType: WorkingTypeMarkPrice, Quantity: 1, StopGuaranteed: "true"},
		swapAttachedOrder(&SwapTakeProfitStopLossResponse{Type: OrderTypeStopMarket, Quantity: 1, StopPrice: 9, Price: 8, WorkingType: WorkingTypeMarkPrice, StopGuaranteed: "true"}),
		"swapAttachedOrder should copy every value of the attached order")
}

func TestSwapOrderCloses(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		o    *SwapOrderResponse
		exp  bool
	}{
		{"one-way reduce only", &SwapOrderResponse{Side: SideSell, PositionSide: PositionSideBoth, ReduceOnly: true}, true},
		{"close position", &SwapOrderResponse{Side: SideSell, PositionSide: PositionSideLong, ClosePosition: true}, true},
		{"hedge mode sell on LONG", &SwapOrderResponse{Side: SideSell, PositionSide: PositionSideLong}, true},
		{"hedge mode buy on SHORT", &SwapOrderResponse{Side: SideBuy, PositionSide: PositionSideShort}, true},
		{"hedge mode buy on LONG", &SwapOrderResponse{Side: SideBuy, PositionSide: PositionSideLong}, false},
		{"hedge mode sell on SHORT", &SwapOrderResponse{Side: SideSell, PositionSide: PositionSideShort}, false},
		{"one-way order", &SwapOrderResponse{Side: SideSell, PositionSide: PositionSideBoth}, false},
	} {
		assert.Equalf(t, tc.exp, swapOrderCloses(tc.o), "swapOrderCloses should report whether a %s closes a position", tc.name)
	}
}

// wrapperSwapOrderReply returns a USDT-M order details reply for order 21, a BTC-USDT LIMIT buy of 2 at 10 with 0.5
// filled, with extra JSON members
func wrapperSwapOrderReply(extra string) string {
	return `{"code":0,"data":{"order":{"symbol":"BTC-USDT","orderId":21,"side":"BUY","positionSide":"LONG","type":"LIMIT","origQty":"2","price":"10","executedQty":"0.5","status":"PARTIALLY_FILLED","clientOrderId":"c21"` + extra + `}}}`
}

// wrapperSwapCancelledReply returns a USDT-M cancellation reply for an order cancelled after executing the quantity given
func wrapperSwapCancelledReply(orderID int, executed string) string {
	return `{"code":0,"data":{"order":{"symbol":"BTC-USDT","orderId":` + strconv.Itoa(orderID) + `,"executedQty":"` + executed + `","status":"CANCELLED"}}}`
}

// wrapperSwapPlacedReply returns a USDT-M order placement reply for order 22 of type orderType, with extra JSON members.
// As BingX's documented example, it reports no price, quantity, time in force or status
func wrapperSwapPlacedReply(orderType, extra string) string {
	return `{"code":0,"data":{"order":{"symbol":"BTC-USDT","orderId":22,"side":"BUY","positionSide":"LONG","type":"` + orderType + `","clientOrderId":"n22","workingType":"MARK_PRICE"` + extra + `}}}`
}

func TestCancelSwapOrderForReplacement(t *testing.T) {
	t.Parallel()
	cancelledLookup := `{"code":0,"data":{"order":{"symbol":"BTC-USDT","orderId":21,"side":"BUY","positionSide":"LONG","type":"LIMIT","origQty":"2","price":"10","executedQty":"1","status":"CANCELLED"}}}`
	notFound := `{"code":109421,"msg":"order not exist"}`
	ex, requests := wrapperTestServer(t, map[string][]string{
		"DELETE /openApi/swap/v2/trade/order": {wrapperSwapCancelledReply(21, "0.5"), wrapperNoReply, notFound, notFound},
		"GET /openApi/swap/v2/trade/order":    {cancelledLookup, cancelledLookup, `{"code":100001,"msg":"signature"}`},
	})
	executed, err := ex.cancelSwapOrderForReplacement(t.Context(), currency.NewBTCUSDT(), 21)
	require.NoError(t, err, "cancelSwapOrderForReplacement must not error")
	assert.Equal(t, types.Number(0.5), executed, "cancelSwapOrderForReplacement should return the quantity the cancellation reports executed")
	assert.Len(t, requests(), 1, "cancelSwapOrderForReplacement should not look up an order its cancellation cancelled")

	executed, err = ex.cancelSwapOrderForReplacement(t.Context(), currency.NewBTCUSDT(), 21)
	require.NoError(t, err, "cancelSwapOrderForReplacement must not error for an order its lookup shows cancelled after a cancellation BingX did not answer")
	assert.Equal(t, types.Number(1), executed, "cancelSwapOrderForReplacement should return the quantity the lookup after a cancellation without an answer reports executed")
	got := requests()
	require.Len(t, got, 3, "cancelSwapOrderForReplacement must look the order up again after a cancellation without an answer")
	assert.Equal(t, http.MethodGet, got[2].method, "cancelSwapOrderForReplacement should look the order up again")
	assert.Equal(t, url.Values{"symbol": {"BTC-USDT"}, "orderId": {"21"}}, got[2].params, "cancelSwapOrderForReplacement should look the order up by its order ID")

	_, err = ex.cancelSwapOrderForReplacement(t.Context(), currency.NewBTCUSDT(), 21)
	assert.ErrorIs(t, err, errOrderNotOpen, "cancelSwapOrderForReplacement should report an order its lookup shows cancelled after BingX refused the cancellation as no longer open")
	assert.ErrorIs(t, err, order.ErrOrderNotFound, "cancelSwapOrderForReplacement should return the cancellation's error")

	_, err = ex.cancelSwapOrderForReplacement(t.Context(), currency.NewBTCUSDT(), 21)
	assert.ErrorIs(t, err, errCancelOutcomeUnknown, "cancelSwapOrderForReplacement should report the outcome as unknown when the lookup fails")
	assert.ErrorIs(t, err, order.ErrOrderNotFound, "cancelSwapOrderForReplacement should return the cancellation's error")
	assert.ErrorIs(t, err, request.ErrAuthRequestFailed, "cancelSwapOrderForReplacement should return the lookup's error")
}

func TestAmendOrReplaceSwapOrder(t *testing.T) {
	t.Parallel()
	pair := currency.NewBTCUSDT()
	t.Run("amend", func(t *testing.T) {
		t.Parallel()
		ex, requests := wrapperTestServer(t, map[string][]string{
			"GET /openApi/swap/v2/trade/order":  {wrapperSwapOrderReply("")},
			"POST /openApi/swap/v1/trade/amend": {`{"code":0,"data":{"orderId":21,"clientOrderId":"c21","symbol":"BTC-USDT","quantity":3}}`},
		})
		resp, err := ex.amendOrReplaceSwapOrder(t.Context(), &order.Modify{Exchange: ex.Name, Pair: pair, AssetType: asset.USDTMarginedFutures, OrderID: "21", Amount: 3}, 21)
		require.NoError(t, err, "amendOrReplaceSwapOrder must not error")
		got := requests()
		require.Len(t, got, 2, "amendOrReplaceSwapOrder must fetch the order and then amend it")
		assert.Equal(t, url.Values{"symbol": {"BTC-USDT"}, "orderId": {"21"}}, got[0].params, "amendOrReplaceSwapOrder should fetch the order")
		assert.Equal(t, url.Values{"symbol": {"BTC-USDT"}, "orderId": {"21"}, "quantity": {"3"}}, got[1].params, "amendOrReplaceSwapOrder should amend the order's amount")
		assert.Equal(t, "21", resp.OrderID, "amendOrReplaceSwapOrder should keep the order ID")
		assert.Equal(t, "c21", resp.ClientOrderID, "amendOrReplaceSwapOrder should keep the client order ID")
		assert.Equal(t, 3.0, resp.Amount, "amendOrReplaceSwapOrder should return the new amount")
		assert.Equal(t, 10.0, resp.Price, "amendOrReplaceSwapOrder should keep the price")
		assert.Equal(t, 2.5, resp.RemainingAmount, "amendOrReplaceSwapOrder should return the unfilled amount")
		assert.Equal(t, order.GoodTillCancel, resp.TimeInForce, "amendOrReplaceSwapOrder should keep the time in force")
		assert.Equal(t, order.Limit, resp.Type, "amendOrReplaceSwapOrder should keep the type")
		assert.Equal(t, order.Buy, resp.Side, "amendOrReplaceSwapOrder should keep the side")
		assert.Equal(t, order.PartiallyFilled, resp.Status, "amendOrReplaceSwapOrder should keep the status")
	})

	t.Run("trigger price type", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name                             string
			orderType, existing, workingType string
			priceType                        order.PriceType
			triggerPrice                     float64
			err                              error
		}{
			{"unset keeps the order's", "STOP", "INDEX_PRICE", "INDEX_PRICE", order.LastPrice, 8, nil},
			{"mark price", "STOP", "INDEX_PRICE", "MARK_PRICE", order.MarkPrice, 8, nil},
			{"index price", "STOP", "MARK_PRICE", "INDEX_PRICE", order.IndexPrice, 8, nil},
			{"price type alone", "STOP", "INDEX_PRICE", "MARK_PRICE", order.MarkPrice, 0, nil},
			{"price type with the same trigger price", "STOP", "INDEX_PRICE", "MARK_PRICE", order.MarkPrice, 9, nil},
			{"unknown", "STOP", "MARK_PRICE", "", order.UnknownPriceType, 8, order.ErrUnknownPriceType},
			{"trailing stop", "TRAILING_STOP_MARKET", "INDEX_PRICE", "MARK_PRICE", order.MarkPrice, 8, nil},
			{"stop market", "STOP_MARKET", "INDEX_PRICE", "MARK_PRICE", order.MarkPrice, 8, nil},
			{"take profit", "TAKE_PROFIT", "INDEX_PRICE", "MARK_PRICE", order.MarkPrice, 8, nil},
			{"take profit market", "TAKE_PROFIT_MARKET", "INDEX_PRICE", "MARK_PRICE", order.MarkPrice, 8, nil},
			{"trigger limit", "TRIGGER_LIMIT", "INDEX_PRICE", "MARK_PRICE", order.MarkPrice, 8, nil},
			{"trigger market", "TRIGGER_MARKET", "INDEX_PRICE", "MARK_PRICE", order.MarkPrice, 8, nil},
			{"trailing take profit or stop loss", "TRAILING_TP_SL", "INDEX_PRICE", "MARK_PRICE", order.MarkPrice, 8, nil},
			{"limit order keeps the order's", "LIMIT", "INDEX_PRICE", "INDEX_PRICE", order.MarkPrice, 8, nil},
			{"limit order ignores an unknown type", "LIMIT", "INDEX_PRICE", "INDEX_PRICE", order.UnknownPriceType, 8, nil},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				existing := `{"code":0,"data":{"order":{"symbol":"BTC-USDT","orderId":21,"side":"BUY","positionSide":"LONG","type":"` + tc.orderType + `","origQty":"2","price":"10","stopPrice":"9","executedQty":"0","status":"NEW","workingType":"` + tc.existing + `"}}}`
				routes := map[string][]string{"GET /openApi/swap/v2/trade/order": {existing}}
				if tc.err == nil {
					routes["DELETE /openApi/swap/v2/trade/order"] = []string{wrapperSwapCancelledReply(21, "0")}
					routes["POST /openApi/swap/v2/trade/order"] = []string{wrapperSwapPlacedReply(tc.orderType, "")}
				}
				ex, requests := wrapperTestServer(t, routes)
				_, err := ex.amendOrReplaceSwapOrder(t.Context(), &order.Modify{Exchange: ex.Name, Pair: pair, AssetType: asset.USDTMarginedFutures, OrderID: "21", TriggerPrice: tc.triggerPrice, TriggerPriceType: tc.priceType}, 21)
				got := requests()
				if tc.err != nil {
					assert.ErrorIs(t, err, tc.err, "amendOrReplaceSwapOrder should refuse a price type BingX does not take")
					assert.Len(t, got, 1, "amendOrReplaceSwapOrder should cancel nothing for a price type it refuses")
					return
				}
				require.NoError(t, err, "amendOrReplaceSwapOrder must not error")
				require.Len(t, got, 3, "amendOrReplaceSwapOrder must fetch the order, cancel it and place its replacement")
				assert.Equal(t, tc.workingType, got[2].params.Get("workingType"), "amendOrReplaceSwapOrder should trigger the replacement on the price type asked for")
				if tc.triggerPrice == 0 {
					assert.Equal(t, "9", got[2].params.Get("stopPrice"), "amendOrReplaceSwapOrder should keep the order's trigger price when the modification sets none")
				}
			})
		}
	})

	t.Run("replace", func(t *testing.T) {
		t.Parallel()
		ex, requests := wrapperTestServer(t, map[string][]string{
			"GET /openApi/swap/v2/trade/order": {
				wrapperSwapOrderReply(`,"workingType":"MARK_PRICE","stopLoss":{"type":"STOP_MARKET","stopPrice":8,"price":0,"quantity":0,"workingType":"MARK_PRICE"}`),
				wrapperSwapOrderReply(""),
			},
			"DELETE /openApi/swap/v2/trade/order": {wrapperSwapCancelledReply(21, "0.5"), wrapperSwapCancelledReply(21, "0.5")},
			"POST /openApi/swap/v2/trade/order":   {wrapperSwapPlacedReply("LIMIT", ""), wrapperSwapPlacedReply("LIMIT", `,"status":"PARTIALLY_FILLED","executedQty":"0.5"`)},
		})
		resp, err := ex.amendOrReplaceSwapOrder(t.Context(), &order.Modify{Exchange: ex.Name, Pair: pair, AssetType: asset.USDTMarginedFutures, OrderID: "21", Price: 11, NewClientOrderID: "n22"}, 21)
		require.NoError(t, err, "amendOrReplaceSwapOrder must not error")
		got := requests()
		require.Len(t, got, 3, "amendOrReplaceSwapOrder must fetch the order, cancel it and then place its replacement")
		assert.Equal(t, http.MethodDelete, got[1].method, "amendOrReplaceSwapOrder should cancel the order before placing its replacement")
		assert.Equal(t, url.Values{"symbol": {"BTC-USDT"}, "orderId": {"21"}}, got[1].params, "amendOrReplaceSwapOrder should cancel the order by its order ID")
		assert.Equal(t, url.Values{
			"symbol":        {"BTC-USDT"},
			"type":          {"LIMIT"},
			"side":          {"BUY"},
			"positionSide":  {"LONG"},
			"price":         {"11"},
			"quantity":      {"1.5"},
			"workingType":   {"MARK_PRICE"},
			"stopLoss":      {`{"type":"STOP_MARKET","stopPrice":8,"workingType":"MARK_PRICE"}`},
			"clientOrderId": {"n22"},
			"timeInForce":   {"GTC"},
		}, got[2].params, "amendOrReplaceSwapOrder should place the order's unfilled amount, keeping what the modification leaves unchanged")
		assert.Equal(t, "22", resp.OrderID, "amendOrReplaceSwapOrder should return the replacement's order ID")
		assert.Equal(t, "n22", resp.ClientOrderID, "amendOrReplaceSwapOrder should return the replacement's client order ID")
		assert.Equal(t, 11.0, resp.Price, "amendOrReplaceSwapOrder should return the replacement's price")
		assert.Equal(t, 1.5, resp.Amount, "amendOrReplaceSwapOrder should return the replacement's amount")
		assert.Equal(t, 1.5, resp.RemainingAmount, "amendOrReplaceSwapOrder should return the replacement's unfilled amount")
		assert.Equal(t, order.New, resp.Status, "amendOrReplaceSwapOrder should report a replacement without a status new")
		assert.Equal(t, order.GoodTillCancel, resp.TimeInForce, "amendOrReplaceSwapOrder should return the replacement's time in force")
		assert.Equal(t, order.Limit, resp.Type, "amendOrReplaceSwapOrder should return the replacement's type")
		assert.Equal(t, order.Buy, resp.Side, "amendOrReplaceSwapOrder should return the replacement's side")

		resp, err = ex.amendOrReplaceSwapOrder(t.Context(), &order.Modify{Exchange: ex.Name, Pair: pair, AssetType: asset.USDTMarginedFutures, OrderID: "21", Price: 11}, 21)
		require.NoError(t, err, "amendOrReplaceSwapOrder must not error")
		assert.Equal(t, order.PartiallyFilled, resp.Status, "amendOrReplaceSwapOrder should return the status the placement reports")
		assert.Equal(t, 1.0, resp.RemainingAmount, "amendOrReplaceSwapOrder should return the replacement's unfilled amount less what its placement executed")
	})

	t.Run("carried values", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name     string
			existing string
			action   order.Modify
			exp      url.Values
		}{
			{
				name:     "one-way reduce only stop limit order",
				existing: `{"symbol":"BTC-USDT","orderId":23,"side":"SELL","positionSide":"BOTH","type":"STOP","origQty":"1","price":"9","stopPrice":"9.5","executedQty":"0","status":"NEW","reduceOnly":true,"workingType":"MARK_PRICE","stopGuaranteed":"true","positionID":77,"takeProfit":{"type":"TAKE_PROFIT_MARKET","stopPrice":12,"price":0,"quantity":0,"workingType":"MARK_PRICE"},"stopLoss":{"type":"STOP_MARKET","stopPrice":8,"price":0,"quantity":0,"workingType":"CONTRACT_PRICE"}}`,
				action:   order.Modify{TimeInForce: order.ImmediateOrCancel},
				exp: url.Values{
					"symbol": {"BTC-USDT"}, "type": {"STOP"}, "side": {"SELL"}, "positionSide": {"BOTH"}, "reduceOnly": {"true"}, "price": {"9"}, "quantity": {"1"},
					"stopPrice": {"9.5"}, "workingType": {"MARK_PRICE"}, "takeProfit": {`{"type":"TAKE_PROFIT_MARKET","stopPrice":12,"workingType":"MARK_PRICE"}`},
					"stopLoss": {`{"type":"STOP_MARKET","stopPrice":8,"workingType":"CONTRACT_PRICE"}`}, "timeInForce": {"IOC"}, "stopGuaranteed": {"true"}, "positionId": {"77"},
				},
			},
			{
				name:     "close position order",
				existing: `{"symbol":"BTC-USDT","orderId":23,"side":"SELL","positionSide":"LONG","type":"STOP_MARKET","origQty":"1","stopPrice":"9","executedQty":"0","status":"NEW","closePosition":"true","positionID":78}`,
				action:   order.Modify{TriggerPrice: 8},
				exp:      url.Values{"symbol": {"BTC-USDT"}, "type": {"STOP_MARKET"}, "side": {"SELL"}, "positionSide": {"LONG"}, "quantity": {"1"}, "stopPrice": {"8"}, "timeInForce": {"GTC"}, "closePosition": {"true"}, "positionId": {"78"}},
			},
			{
				name:     "trailing order with a price rate",
				existing: `{"symbol":"BTC-USDT","orderId":23,"side":"SELL","positionSide":"LONG","type":"TRAILING_STOP_MARKET","origQty":"1","executedQty":"0","status":"NEW","workingType":"MARK_PRICE","priceRate":"0.01","actPrice":"100","positionID":79}`,
				action:   order.Modify{TriggerPrice: 110},
				exp:      url.Values{"symbol": {"BTC-USDT"}, "type": {"TRAILING_STOP_MARKET"}, "side": {"SELL"}, "positionSide": {"LONG"}, "quantity": {"1"}, "priceRate": {"0.01"}, "workingType": {"MARK_PRICE"}, "timeInForce": {"GTC"}, "activationPrice": {"110"}, "positionId": {"79"}},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex, requests := wrapperTestServer(t, map[string][]string{
					"GET /openApi/swap/v2/trade/order":    {`{"code":0,"data":{"order":` + tc.existing + `}}`},
					"DELETE /openApi/swap/v2/trade/order": {wrapperSwapCancelledReply(23, "0")},
					"POST /openApi/swap/v2/trade/order":   {wrapperSwapPlacedReply("LIMIT", "")},
				})
				action := tc.action
				action.Exchange, action.Pair, action.AssetType, action.OrderID = ex.Name, pair, asset.USDTMarginedFutures, "23"
				_, err := ex.amendOrReplaceSwapOrder(t.Context(), &action, 23)
				require.NoErrorf(t, err, "amendOrReplaceSwapOrder must not error for a %s", tc.name)
				got := requests()
				require.Lenf(t, got, 3, "amendOrReplaceSwapOrder must fetch, cancel and then replace a %s", tc.name)
				assert.Equalf(t, tc.exp, got[2].params, "amendOrReplaceSwapOrder should place a replacement carrying every value of a %s", tc.name)
			})
		}
	})

	t.Run("desired total", func(t *testing.T) {
		t.Parallel()
		existing := `{"code":0,"data":{"order":{"symbol":"BTC-USDT","orderId":27,"side":"BUY","positionSide":"LONG","type":"LIMIT","origQty":"10","price":"10","executedQty":"4","status":"PARTIALLY_FILLED"}}}`
		ex, requests := wrapperTestServer(t, map[string][]string{
			"GET /openApi/swap/v2/trade/order":    {existing, existing, existing, existing},
			"DELETE /openApi/swap/v2/trade/order": {wrapperSwapCancelledReply(27, "4"), wrapperSwapCancelledReply(27, "5"), wrapperSwapCancelledReply(27, "10")},
			"POST /openApi/swap/v2/trade/order":   {wrapperSwapPlacedReply("LIMIT", ""), wrapperSwapPlacedReply("LIMIT", "")},
		})
		modify := order.Modify{Exchange: ex.Name, Pair: pair, AssetType: asset.USDTMarginedFutures, OrderID: "27", Price: 11, Amount: 10}
		resp, err := ex.amendOrReplaceSwapOrder(t.Context(), &modify, 27)
		require.NoError(t, err, "amendOrReplaceSwapOrder must not error")
		got := requests()
		require.Len(t, got, 3, "amendOrReplaceSwapOrder must fetch the order, cancel it and then place its replacement")
		assert.Equal(t, "6", got[2].params.Get("quantity"), "amendOrReplaceSwapOrder should place the desired total less the executed quantity")
		assert.Equal(t, 6.0, resp.Amount, "amendOrReplaceSwapOrder should return the replacement's amount")

		resp, err = ex.amendOrReplaceSwapOrder(t.Context(), &modify, 27)
		require.NoError(t, err, "amendOrReplaceSwapOrder must not error when the order executed more before its cancellation")
		got = requests()
		require.Len(t, got, 6, "amendOrReplaceSwapOrder must fetch the order, cancel it and then place its replacement")
		assert.Equal(t, "5", got[5].params.Get("quantity"), "amendOrReplaceSwapOrder should size the replacement by the quantity the cancellation reports executed")
		assert.Equal(t, 5.0, resp.Amount, "amendOrReplaceSwapOrder should return the replacement's amount")

		_, err = ex.amendOrReplaceSwapOrder(t.Context(), &modify, 27)
		assert.ErrorIs(t, err, errCancelledOrderExecuted, "amendOrReplaceSwapOrder should report an order whose cancellation executed its desired total")
		got = requests()
		require.Len(t, got, 8, "amendOrReplaceSwapOrder must fetch and cancel the order")
		assert.Equal(t, http.MethodDelete, got[7].method, "amendOrReplaceSwapOrder should place nothing for an order that executed its desired total")

		modify.Amount = 4
		_, err = ex.amendOrReplaceSwapOrder(t.Context(), &modify, 27)
		assert.ErrorIs(t, err, errModifyAmountNotAboveExecuted, "amendOrReplaceSwapOrder should refuse a desired total the order has executed")
		assert.Len(t, requests(), 9, "amendOrReplaceSwapOrder should refuse the modification before cancelling the order")
	})

	t.Run("position ID", func(t *testing.T) {
		t.Parallel()
		lookup := func(side, positionID string) string {
			return `{"code":0,"data":{"order":{"symbol":"BTC-USDT","orderId":26,"side":"` + side + `","positionSide":"LONG","type":"LIMIT","origQty":"1","price":"12","executedQty":"0","status":"NEW","positionID":` + positionID + `}}}`
		}
		ex, requests := wrapperTestServer(t, map[string][]string{
			"GET /openApi/swap/v2/trade/order":      {lookup("SELL", "2085388523545022500"), lookup("BUY", "2085388523545022500"), lookup("SELL", "0"), lookup("SELL", "0")},
			"GET /openApi/swap/v2/trade/marginType": {`{"code":0,"data":{"marginType":"SEPARATE_ISOLATED"}}`, `{"code":100500,"msg":"busy"}`},
			"DELETE /openApi/swap/v2/trade/order":   {wrapperSwapCancelledReply(26, "0"), wrapperSwapCancelledReply(26, "0")},
			"POST /openApi/swap/v2/trade/order":     {wrapperSwapPlacedReply("LIMIT", ""), wrapperSwapPlacedReply("LIMIT", "")},
		})
		modify := &order.Modify{Exchange: ex.Name, Pair: pair, AssetType: asset.USDTMarginedFutures, OrderID: "26", Price: 13}
		_, err := ex.amendOrReplaceSwapOrder(t.Context(), modify, 26)
		require.NoError(t, err, "amendOrReplaceSwapOrder must not error")
		got := requests()
		require.Len(t, got, 3, "amendOrReplaceSwapOrder must replace a closing order whose position ID its lookup returned without checking the margin type")
		assert.Equal(t, "2085388523545022500", got[2].params.Get("positionId"), "amendOrReplaceSwapOrder should keep a closing order's position ID")

		_, err = ex.amendOrReplaceSwapOrder(t.Context(), modify, 26)
		require.NoError(t, err, "amendOrReplaceSwapOrder must not error")
		got = requests()
		require.Len(t, got, 6, "amendOrReplaceSwapOrder must fetch, cancel and then replace an opening order")
		assert.False(t, got[5].params.Has("positionId"), "amendOrReplaceSwapOrder should not send the position ID of an order that closes no position")

		_, err = ex.amendOrReplaceSwapOrder(t.Context(), modify, 26)
		assert.ErrorIs(t, err, errSwapPositionIDRequired, "amendOrReplaceSwapOrder should refuse to replace a separate isolated closing order without its position ID")
		_, err = ex.amendOrReplaceSwapOrder(t.Context(), modify, 26)
		assert.ErrorIs(t, err, errAPIResponse, "amendOrReplaceSwapOrder should return the margin type lookup's error")
		for _, r := range requests()[6:] {
			assert.NotEqual(t, http.MethodDelete, r.method, "amendOrReplaceSwapOrder should not cancel an order it refuses to replace")
		}
	})

	t.Run("client order ID", func(t *testing.T) {
		t.Parallel()
		stop := `{"code":0,"data":{"order":{"symbol":"BTC-USDT","orderId":28,"side":"SELL","positionSide":"BOTH","type":"STOP","origQty":"1","price":"9","stopPrice":"9.5","executedQty":"0","status":"NEW","reduceOnly":true}}}`
		ex, requests := wrapperTestServer(t, map[string][]string{"GET /openApi/swap/v2/trade/order": {stop}})
		_, err := ex.amendOrReplaceSwapOrder(t.Context(), &order.Modify{Exchange: ex.Name, Pair: pair, AssetType: asset.USDTMarginedFutures, OrderID: "28", Price: 8, NewClientOrderID: "n28"}, 28)
		assert.ErrorIs(t, err, errSwapClientOrderIDUnsupportedType, "amendOrReplaceSwapOrder should refuse a new client order ID for a STOP order")
		assert.Len(t, requests(), 1, "amendOrReplaceSwapOrder should refuse the modification before cancelling the order or looking its margin type up")
	})

	t.Run("trailing stop", func(t *testing.T) {
		t.Parallel()
		trailing := `{"code":0,"data":{"order":{"symbol":"BTC-USDT","orderId":24,"side":"SELL","positionSide":"LONG","type":"TRAILING_STOP_MARKET","origQty":"1","price":"5","executedQty":"0","status":"NEW","workingType":"MARK_PRICE","actPrice":"100"}}}`
		ex, requests := wrapperTestServer(t, map[string][]string{
			"GET /openApi/swap/v2/trade/order":      {trailing, trailing},
			"GET /openApi/swap/v2/trade/marginType": {`{"code":0,"data":{"marginType":"ISOLATED"}}`, `{"code":0,"data":{"marginType":"ISOLATED"}}`},
			"DELETE /openApi/swap/v2/trade/order":   {wrapperSwapCancelledReply(24, "0"), wrapperSwapCancelledReply(24, "0")},
			"POST /openApi/swap/v2/trade/order":     {wrapperSwapPlacedReply("TRAILING_STOP_MARKET", ""), wrapperSwapPlacedReply("TRAILING_STOP_MARKET", "")},
		})
		for _, tc := range []struct {
			name       string
			trigger    float64
			activation float64
		}{
			{"activation price changes", 110, 110},
			{"activation price is kept", 0, 100},
		} {
			resp, err := ex.amendOrReplaceSwapOrder(t.Context(), &order.Modify{Exchange: ex.Name, Pair: pair, AssetType: asset.USDTMarginedFutures, OrderID: "24", TriggerPrice: tc.trigger}, 24)
			require.NoErrorf(t, err, "amendOrReplaceSwapOrder must not error when the %s", tc.name)
			got := requests()
			assert.Equalf(t, url.Values{
				"symbol":          {"BTC-USDT"},
				"type":            {"TRAILING_STOP_MARKET"},
				"side":            {"SELL"},
				"positionSide":    {"LONG"},
				"price":           {"5"},
				"quantity":        {"1"},
				"workingType":     {"MARK_PRICE"},
				"timeInForce":     {"GTC"},
				"activationPrice": {strconv.FormatFloat(tc.activation, 'f', -1, 64)},
			}, got[len(got)-1].params, "amendOrReplaceSwapOrder should take a trailing stop's trigger price as its activation price when the %s", tc.name)
			assert.Zerof(t, resp.Price, "amendOrReplaceSwapOrder should not report a trailing distance as the price when the %s", tc.name)
			assert.Equalf(t, tc.activation, resp.TriggerPrice, "amendOrReplaceSwapOrder should report the activation price as the trigger price when the %s", tc.name)
		}
	})

	t.Run("errors", func(t *testing.T) {
		t.Parallel()
		ex, requests := wrapperTestServer(t, map[string][]string{
			"GET /openApi/swap/v2/trade/order":    {`{"code":100500,"msg":"busy"}`, wrapperSwapOrderReply(""), wrapperSwapOrderReply(""), wrapperSwapOrderReply(""), wrapperSwapOrderReply(""), wrapperSwapOrderReply("")},
			"POST /openApi/swap/v1/trade/amend":   {`{"code":100500,"msg":"busy"}`},
			"DELETE /openApi/swap/v2/trade/order": {`{"code":100500,"msg":"busy"}`, wrapperSwapCancelledReply(21, "0.5")},
			"POST /openApi/swap/v2/trade/order":   {`{"code":101204,"msg":"insufficient margin"}`},
		})
		modify := func(action order.Modify) error {
			action.Exchange, action.Pair, action.AssetType, action.OrderID = ex.Name, pair, asset.USDTMarginedFutures, "21"
			_, err := ex.amendOrReplaceSwapOrder(t.Context(), &action, 21)
			return err
		}
		assert.ErrorIs(t, modify(order.Modify{Amount: 3}), errAPIResponse, "amendOrReplaceSwapOrder should return the order request's error")
		assert.ErrorIs(t, modify(order.Modify{Amount: 3}), errAPIResponse, "amendOrReplaceSwapOrder should return the amendment's error")
		assert.ErrorIs(t, modify(order.Modify{Price: 11, TimeInForce: order.GoodTillDay}), order.ErrUnsupportedTimeInForce, "amendOrReplaceSwapOrder should reject a time in force BingX does not take")
		err := modify(order.Modify{Price: 11})
		assert.ErrorIs(t, err, errAPIResponse, "amendOrReplaceSwapOrder should return the cancellation's error")
		assert.ErrorIs(t, err, errCancelReplaceFailed, "amendOrReplaceSwapOrder should report a failed cancellation")
		assert.ErrorIs(t, err, errCancelOutcomeUnknown, "amendOrReplaceSwapOrder should report a cancellation BingX found busy as one whose outcome is unknown")
		assert.ErrorContains(t, err, "cancelling order 21 got no answer showing its outcome while the order is still open, so no replacement was placed", "amendOrReplaceSwapOrder should report that nothing was placed")
		err = modify(order.Modify{Price: 11})
		assert.ErrorIs(t, err, errAPIResponse, "amendOrReplaceSwapOrder should return the placement's error")
		assert.ErrorIs(t, err, errCancelReplaceFailed, "amendOrReplaceSwapOrder should report a failed replacement")
		assert.ErrorContains(t, err, "order 21 was cancelled, but placing its replacement failed", "amendOrReplaceSwapOrder should report the order cancelled without its replacement")
		var placements int
		for _, r := range requests() {
			if r.method == http.MethodPost && r.path == "/openApi/swap/v2/trade/order" {
				placements++
			}
		}
		assert.Equal(t, 1, placements, "amendOrReplaceSwapOrder should place a replacement only after cancelling the order")
	})

	t.Run("order no longer open", func(t *testing.T) {
		t.Parallel()
		ex, requests := wrapperTestServer(t, map[string][]string{
			"GET /openApi/swap/v2/trade/order": {`{"code":0,"data":{"order":{"symbol":"BTC-USDT","orderId":21,"side":"BUY","positionSide":"LONG","type":"LIMIT","origQty":"2","price":"10","executedQty":"0","status":"CANCELLED"}}}`},
		})
		_, err := ex.amendOrReplaceSwapOrder(t.Context(), &order.Modify{Exchange: ex.Name, Pair: pair, AssetType: asset.USDTMarginedFutures, OrderID: "21", Price: 11}, 21)
		assert.ErrorIs(t, err, errOrderNotOpen, "amendOrReplaceSwapOrder should refuse an order its lookup shows cancelled")
		assert.Len(t, requests(), 1, "amendOrReplaceSwapOrder should neither cancel nor replace an order no longer open")
	})

	t.Run("failed cancellation", func(t *testing.T) {
		t.Parallel()
		lookedUp := func(status, executed string) string {
			return `{"code":0,"data":{"order":{"symbol":"BTC-USDT","orderId":21,"side":"BUY","positionSide":"LONG","type":"LIMIT","origQty":"2","price":"10","executedQty":"` + executed + `","status":"` + status + `"}}}`
		}
		refusal, failedLookup := `{"code":109421,"msg":"order not exist"}`, `{"code":100001,"msg":"signature"}`
		cases := []wrapperFailedCancellationCase{
			{name: "refusal, order cancelled", cancel: refusal, lookup: lookedUp("CANCELLED", "1"), errs: []error{errOrderNotOpen, order.ErrOrderNotFound}},
			{name: "refusal, order cancelled after executing the desired total", cancel: refusal, lookup: lookedUp("CANCELLED", "2"), errs: []error{errOrderNotOpen, order.ErrOrderNotFound}},
			{name: "refusal, order filled", cancel: refusal, lookup: lookedUp("FILLED", "2"), errs: []error{errCancelledOrderExecuted, order.ErrOrderNotFound}},
			{name: "refusal, order still open", cancel: refusal, lookup: lookedUp("PARTIALLY_FILLED", "0.5"), errs: []error{errCancelReplaceFailed, order.ErrOrderNotFound}, notErrs: []error{errCancelOutcomeUnknown}},
			{name: "refusal, status that does not show whether the order is open", cancel: refusal, lookup: lookedUp("ODD", "0.5"), errs: []error{errCancelReplaceFailed, order.ErrOrderNotFound}, notErrs: []error{errCancelOutcomeUnknown}},
			{name: "refusal, failed lookup", cancel: refusal, lookup: failedLookup, errs: []error{errCancelOutcomeUnknown, order.ErrOrderNotFound, request.ErrAuthRequestFailed}},
			{name: "no answer, order cancelled", cancel: wrapperNoReply, lookup: lookedUp("CANCELLED", "1"), quantity: "1"},
			{name: "no answer, order cancelled after executing the desired total", cancel: wrapperNoReply, lookup: lookedUp("CANCELLED", "2"), errs: []error{errCancelledOrderExecuted}},
			{name: "no answer, order filled", cancel: wrapperNoReply, lookup: lookedUp("FILLED", "2"), errs: []error{errCancelledOrderExecuted}},
			{name: "no answer, order still open", cancel: wrapperNoReply, lookup: lookedUp("PARTIALLY_FILLED", "0.5"), errs: []error{errCancelReplaceFailed, errCancelOutcomeUnknown}},
			{name: "no answer, status that does not show whether the order is open", cancel: wrapperNoReply, lookup: lookedUp("ODD", "0.5"), errs: []error{errCancelReplaceFailed, errCancelOutcomeUnknown}},
			{name: "no answer, failed lookup", cancel: wrapperNoReply, lookup: failedLookup, errs: []error{errCancelOutcomeUnknown, request.ErrAuthRequestFailed}},
		}
		codes := []string{"100500", "109500", "109701", "110500"}
		busy := make([]wrapperFailedCancellationCase, 0, 2*len(codes))
		for _, code := range codes {
			failure := `{"code":` + code + `,"msg":"busy"}`
			busy = append(busy,
				wrapperFailedCancellationCase{name: "code " + code + ", order cancelled", cancel: failure, lookup: lookedUp("CANCELLED", "1"), quantity: "1"},
				wrapperFailedCancellationCase{name: "code " + code + ", order still open", cancel: failure, lookup: lookedUp("PARTIALLY_FILLED", "0.5"), errs: []error{errCancelReplaceFailed, errCancelOutcomeUnknown, errAPIResponse}},
			)
		}
		for _, tc := range append(append(cases, busy...), wrapperSendFailureCases(cases)...) {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				routes := map[string][]string{"GET /openApi/swap/v2/trade/order": {wrapperSwapOrderReply(""), tc.lookup}}
				// The order is looked up, cancelled and looked up again, unless the rate limiter refuses the cancellation
				sent := 3
				if tc.cancel == wrapperRateLimited {
					sent = 2
				} else {
					routes["DELETE /openApi/swap/v2/trade/order"] = []string{tc.cancel}
				}
				if tc.quantity != "" {
					routes["POST /openApi/swap/v2/trade/order"] = []string{wrapperSwapPlacedReply("LIMIT", "")}
				}
				ex, requests := wrapperTestServer(t, routes)
				ctx := t.Context()
				if tc.cancel == wrapperRateLimited {
					ctx = wrapperRefuseByRateLimit(t, ex, swapCancelOrderRate)
				}
				resp, err := ex.amendOrReplaceSwapOrder(ctx, &order.Modify{Exchange: ex.Name, Pair: pair, AssetType: asset.USDTMarginedFutures, OrderID: "21", Price: 11}, 21)
				got := requests()
				if tc.quantity == "" {
					require.Error(t, err, "amendOrReplaceSwapOrder must place nothing for this outcome")
					for _, target := range tc.errs {
						assert.ErrorIs(t, err, target, "amendOrReplaceSwapOrder should return the outcome the cancellation's answer and the lookup after it show")
					}
					for _, target := range tc.notErrs {
						assert.NotErrorIs(t, err, target, "amendOrReplaceSwapOrder should not report another outcome")
					}
					assert.Len(t, got, sent, "amendOrReplaceSwapOrder should look the order up again after a failed cancellation and place nothing")
					return
				}
				require.NoError(t, err, "amendOrReplaceSwapOrder must replace an order its lookup shows cancelled after a cancellation without an answer showing its outcome")
				require.Len(t, got, sent+1, "amendOrReplaceSwapOrder must fetch the order, cancel it, look it up again and then place its replacement")
				assert.Equal(t, http.MethodGet, got[sent-1].method, "amendOrReplaceSwapOrder should look the order up again after a failed cancellation")
				assert.Equal(t, tc.quantity, got[sent].params.Get("quantity"), "amendOrReplaceSwapOrder should size the replacement by the quantity the lookup reports executed")
				assert.Equal(t, "22", resp.OrderID, "amendOrReplaceSwapOrder should return the replacement's order ID")
			})
		}
	})

	t.Run("values it cannot convert", func(t *testing.T) {
		t.Parallel()
		ex, _ := wrapperTestServer(t, map[string][]string{
			"GET /openApi/swap/v2/trade/order": {
				`{"code":0,"data":{"order":{"symbol":"BTC-USDT","orderId":21,"side":"ODD","positionSide":"LONG","type":"LIMIT","origQty":"2","price":"10","executedQty":"0","status":"ODD"}}}`,
				wrapperSwapOrderReply(""),
			},
			"POST /openApi/swap/v1/trade/amend":   {`{"code":0,"data":{"orderId":21,"symbol":"BTC-USDT","quantity":3}}`},
			"DELETE /openApi/swap/v2/trade/order": {wrapperSwapCancelledReply(21, "0.5")},
			"POST /openApi/swap/v2/trade/order":   {`{"code":0,"data":{"order":{"symbol":"BTC-USDT","orderId":22,"side":"ODD","positionSide":"LONG","type":"ODD","status":"ODD"}}}`},
		})
		resp, err := ex.amendOrReplaceSwapOrder(t.Context(), &order.Modify{Exchange: ex.Name, Pair: pair, AssetType: asset.USDTMarginedFutures, OrderID: "21", Amount: 3}, 21)
		require.NoError(t, err, "amendOrReplaceSwapOrder must not error for an amendment BingX made")
		assert.Equal(t, order.UnknownSide, resp.Side, "amendOrReplaceSwapOrder should leave a side it cannot convert unset")
		assert.Equal(t, order.UnknownStatus, resp.Status, "amendOrReplaceSwapOrder should leave a status it cannot convert unset")
		resp, err = ex.amendOrReplaceSwapOrder(t.Context(), &order.Modify{Exchange: ex.Name, Pair: pair, AssetType: asset.USDTMarginedFutures, OrderID: "21", Price: 11}, 21)
		require.NoError(t, err, "amendOrReplaceSwapOrder must not error for a replacement BingX placed")
		assert.Equal(t, "22", resp.OrderID, "amendOrReplaceSwapOrder should return the replacement's order ID")
		assert.Equal(t, order.New, resp.Status, "amendOrReplaceSwapOrder should report a replacement whose status it cannot convert as new, as a submission is")
		assert.Equal(t, order.UnknownType, resp.Type, "amendOrReplaceSwapOrder should leave a type it cannot convert unset")
		assert.Equal(t, order.UnknownSide, resp.Side, "amendOrReplaceSwapOrder should leave a side it cannot convert unset")
	})

	t.Run("executed at placement", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name, placed string
			exp          order.Status
			remaining    float64
		}{
			{name: "filled replacement", placed: `,"status":"FILLED","executedQty":"1.5","avgPrice":"11"`, exp: order.New},
			{name: "replacement cancelled after executing", placed: `,"status":"CANCELLED","executedQty":"0.5","avgPrice":"11"`, exp: order.New, remaining: 1},
			{name: "replacement cancelled without executing", placed: `,"status":"CANCELLED","executedQty":"0"`, exp: order.Cancelled, remaining: 1.5},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex, _ := wrapperTestServer(t, map[string][]string{
					"GET /openApi/swap/v2/trade/order":    {wrapperSwapOrderReply("")},
					"DELETE /openApi/swap/v2/trade/order": {wrapperSwapCancelledReply(21, "0.5")},
					"POST /openApi/swap/v2/trade/order":   {wrapperSwapPlacedReply("LIMIT", tc.placed)},
				})
				resp, err := ex.amendOrReplaceSwapOrder(t.Context(), &order.Modify{Exchange: ex.Name, Pair: pair, AssetType: asset.USDTMarginedFutures, OrderID: "21", Price: 11}, 21)
				require.NoError(t, err, "amendOrReplaceSwapOrder must not error")
				assert.Equal(t, tc.exp, resp.Status, "amendOrReplaceSwapOrder should report a replacement finished after executing as new, for the order manager to settle it from a later query")
				assert.Equal(t, tc.remaining, resp.RemainingAmount, "amendOrReplaceSwapOrder should return the replacement's unfilled amount")
			})
		}
	})
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.CancelOrder(t.Context(), nil), order.ErrCancelOrderIsNil, "CancelOrder must reject a nil cancellation")
	require.ErrorIs(t, e.CancelOrder(t.Context(), &order.Cancel{AssetType: asset.Spot, OrderID: "1"}), order.ErrPairIsEmpty, "CancelOrder must require a pair")
	require.ErrorIs(t, e.CancelOrder(t.Context(), &order.Cancel{Pair: currency.NewBTCUSDT(), AssetType: asset.Spot, OrderID: "x"}), errInvalidOrderID, "CancelOrder must reject an invalid order ID")
	require.ErrorIs(t, e.CancelOrder(t.Context(), &order.Cancel{Pair: currency.NewBTCUSDT(), AssetType: asset.Spot}), order.ErrOrderIDNotSet, "CancelOrder must require an order ID or client order ID")
	require.ErrorIs(t, e.CancelOrder(t.Context(), &order.Cancel{Pair: currency.NewBTCUSDT(), AssetType: asset.Futures, OrderID: "1"}), asset.ErrNotSupported, "CancelOrder must reject an unsupported asset")

	t.Run("recorded", func(t *testing.T) {
		t.Parallel()
		ex := newWrapperTestExchange(t)
		wrapperSkipIfCannotManipulateOrders(t, ex)
		if !mockTests {
			t.Skip("the recorded cancellations are documented examples, which do not name live orders")
		}
		for _, c := range []*order.Cancel{
			{Pair: currency.NewPairWithDelimiter("NEMS", "USDT", currency.DashDelimiter), AssetType: asset.Spot, OrderID: "1735964079647111280"},
			{Pair: currency.NewPairWithDelimiter("RNDR", "USDT", currency.DashDelimiter), AssetType: asset.USDTMarginedFutures, OrderID: "1736011869418901234", ClientOrderID: "ignored"},
			{Pair: currency.NewPairWithDelimiter("BTC", "USD", currency.DashDelimiter), AssetType: asset.CoinMarginedFutures, OrderID: "1736011869418901234"},
		} {
			assert.NoErrorf(t, ex.CancelOrder(t.Context(), c), "CancelOrder should not error for %s", c.AssetType)
		}
	})

	t.Run("client order ID", func(t *testing.T) {
		t.Parallel()
		ex, requests := wrapperTestServer(t, map[string][]string{
			"POST /openApi/spot/v1/trade/cancel":         {`{"code":0,"data":{"symbol":"BTC-USDT","orderId":1,"status":"CANCELED"}}`},
			"DELETE /openApi/swap/v2/trade/order":        {`{"code":0,"data":{"order":{"symbol":"BTC-USDT","orderId":2,"status":"CANCELLED"}}}`},
			"DELETE /openApi/cswap/v1/trade/cancelOrder": {`{"code":100500,"msg":"busy"}`},
		})
		require.NoError(t, ex.CancelOrder(t.Context(), &order.Cancel{Pair: currency.NewBTCUSDT(), AssetType: asset.Spot, ClientOrderID: "c1"}), "CancelOrder must not error")
		require.NoError(t, ex.CancelOrder(t.Context(), &order.Cancel{Pair: currency.NewBTCUSDT(), AssetType: asset.USDTMarginedFutures, ClientOrderID: "c2"}), "CancelOrder must not error")
		assert.ErrorIs(t, ex.CancelOrder(t.Context(), &order.Cancel{Pair: currency.NewPairWithDelimiter("BTC", "USD", currency.DashDelimiter), AssetType: asset.CoinMarginedFutures, OrderID: "3"}), errAPIResponse, "CancelOrder should return the request's error")
		got := requests()
		require.Len(t, got, 3, "CancelOrder must make a request for each cancellation")
		assert.Equal(t, url.Values{"symbol": {"BTC-USDT"}, "clientOrderID": {"c1"}}, got[0].params, "CancelOrder should cancel a spot order by its client order ID")
		assert.Equal(t, url.Values{"symbol": {"BTC-USDT"}, "clientOrderId": {"c2"}}, got[1].params, "CancelOrder should cancel a USDT-M order by its client order ID")
		assert.Equal(t, url.Values{"symbol": {"BTC-USD"}, "orderId": {"3"}}, got[2].params, "CancelOrder should cancel a coin-M order by its order ID")
	})
}

func TestCancelBatchOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelBatchOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrCancelOrderIsNil, "CancelBatchOrders must reject an empty batch")
	spot := currency.NewPairWithDelimiter("GM", "USDT", currency.DashDelimiter)
	for _, tc := range []struct {
		name    string
		cancels []order.Cancel
		err     error
	}{
		{"unsupported asset", []order.Cancel{{Pair: spot, AssetType: asset.Futures, OrderID: "1"}}, asset.ErrNotSupported},
		{"mixed assets", []order.Cancel{{Pair: spot, AssetType: asset.Spot, OrderID: "1"}, {Pair: spot, AssetType: asset.USDTMarginedFutures, OrderID: "2"}}, errBatchCancelMixedAssets},
		{"empty pair", []order.Cancel{{AssetType: asset.Spot, OrderID: "1"}}, currency.ErrCurrencyPairEmpty},
		{"invalid order ID", []order.Cancel{{Pair: spot, AssetType: asset.Spot, OrderID: "x"}}, errInvalidOrderID},
		{"no order ID", []order.Cancel{{Pair: spot, AssetType: asset.Spot}}, order.ErrOrderIDNotSet},
	} {
		_, err := e.CancelBatchOrders(t.Context(), tc.cancels)
		require.ErrorIsf(t, err, tc.err, "CancelBatchOrders must reject a batch with an %s", tc.name)
	}

	t.Run("recorded", func(t *testing.T) {
		t.Parallel()
		ex := newWrapperTestExchange(t)
		wrapperSkipIfCannotManipulateOrders(t, ex)
		if !mockTests {
			t.Skip("the recorded cancellations are documented examples, which do not name live orders")
		}
		missing := errBatchCancelResultMissing.Error()
		for _, tc := range []struct {
			cancels []order.Cancel
			exp     map[string]string
		}{
			{
				[]order.Cancel{{Pair: spot, AssetType: asset.Spot, OrderID: "1735964997957275648"}, {Pair: spot, AssetType: asset.Spot, OrderID: "1735965127519326208"}},
				// The documented reply names other orders than its request
				map[string]string{
					"1735964997957275600": "CANCELED",
					"1735965127519326200": "CANCELED",
					"1874730962584076288": "the  orderId:1874730962584076288 order status :CANCELED,can not delete, please verify and retry later",
					"1735964997957275648": missing,
					"1735965127519326208": missing,
				},
			},
			{
				[]order.Cancel{{Pair: currency.NewBTCUSDT(), AssetType: asset.USDTMarginedFutures, OrderID: "1735924831603391122"}, {Pair: currency.NewBTCUSDT(), AssetType: asset.USDTMarginedFutures, OrderID: "1735924833239172233"}},
				map[string]string{
					"1735924831603391200": "CANCELLED",
					"1735924833239172400": "CANCELLED",
					"1735924831603391122": missing,
					"1735924833239172233": missing,
				},
			},
			{
				[]order.Cancel{{Pair: currency.NewPairWithDelimiter("BTC", "USD", currency.DashDelimiter), AssetType: asset.CoinMarginedFutures, OrderID: "1736011869418901234"}},
				map[string]string{"1736011869418901234": "CANCELLED"},
			},
		} {
			resp, err := ex.CancelBatchOrders(t.Context(), tc.cancels)
			require.NoErrorf(t, err, "CancelBatchOrders must not error for %s", tc.cancels[0].AssetType)
			assert.Equalf(t, tc.exp, resp.Status, "CancelBatchOrders should report each %s order's outcome", tc.cancels[0].AssetType)
		}
	})

	t.Run("batches and failures", func(t *testing.T) {
		t.Parallel()
		success := func(ids ...int) string {
			orders := make([]string, len(ids))
			for i, id := range ids {
				orders[i] = `{"orderId":` + strconv.Itoa(id) + `,"status":"CANCELLED"}`
			}
			return `{"code":0,"data":{"success":[` + strings.Join(orders, ",") + `],"failed":null}}`
		}
		ex, requests := wrapperTestServer(t, map[string][]string{
			"DELETE /openApi/swap/v2/trade/batchOrders": {
				// BingX reports client order IDs in lower case
				strings.Replace(success(1, 2, 3, 4, 5, 6, 7, 8, 9, 10), `],"failed":null`, `,{"orderId":13,"clientOrderId":"abc","status":"CANCELLED"}],"failed":[{"orderId":0,"clientOrderId":"def","errorCode":80012,"errorMessage":"cancel order failed"}]`, 1),
				success(11),
				`{"code":100500,"msg":"busy"}`,
			},
			"DELETE /openApi/cswap/v1/trade/cancelOrder": {`{"code":0,"data":{"order":{"orderId":1,"status":"CANCELLED"}}}`, `{"code":100500,"msg":"busy"}`},
		})
		cancels := make([]order.Cancel, 0, 13)
		for i := range 11 {
			cancels = append(cancels, order.Cancel{Pair: currency.NewBTCUSDT(), AssetType: asset.USDTMarginedFutures, OrderID: strconv.Itoa(i + 1)})
		}
		cancels = append(cancels,
			order.Cancel{Pair: currency.NewBTCUSDT(), AssetType: asset.USDTMarginedFutures, ClientOrderID: "ABC"},
			order.Cancel{Pair: currency.NewBTCUSDT(), AssetType: asset.USDTMarginedFutures, ClientOrderID: "DEF"},
			order.Cancel{Pair: currency.NewPairWithDelimiter("ETH", "USDT", currency.DashDelimiter), AssetType: asset.USDTMarginedFutures, OrderID: "12"},
		)
		resp, err := ex.CancelBatchOrders(t.Context(), cancels)
		require.ErrorIs(t, err, errAPIResponse, "CancelBatchOrders must return a failed request's error")
		assert.ErrorContains(t, err, "[12]", "CancelBatchOrders should name the orders a failed request covered")
		got := requests()
		require.Len(t, got, 3, "CancelBatchOrders must cancel ten order IDs and ten client order IDs of a pair a request")
		assert.Equal(t, url.Values{"symbol": {"BTC-USDT"}, "orderIdList": {"[1,2,3,4,5,6,7,8,9,10]"}, "clientOrderIdList": {`["ABC","DEF"]`}}, got[0].params, "CancelBatchOrders should send the first ten of each list")
		assert.Equal(t, url.Values{"symbol": {"BTC-USDT"}, "orderIdList": {"[11]"}}, got[1].params, "CancelBatchOrders should send the rest in the next request")
		assert.Equal(t, url.Values{"symbol": {"ETH-USDT"}, "orderIdList": {"[12]"}}, got[2].params, "CancelBatchOrders should cancel each pair's orders apart")
		require.Len(t, resp.Status, 14, "CancelBatchOrders must report every order")
		assert.Equal(t, "CANCELLED", resp.Status["10"], "CancelBatchOrders should report a cancelled order's status")
		assert.Equal(t, "CANCELLED", resp.Status["ABC"], "CancelBatchOrders should match a client order ID BingX lowered to the ID requested")
		assert.Equal(t, (&APIError{Code: 80012, Message: "cancel order failed"}).Error(), resp.Status["DEF"], "CancelBatchOrders should report why an order was not cancelled")
		assert.Contains(t, resp.Status["12"], "code 100500", "CancelBatchOrders should report a failed request against each of its orders")

		resp, err = ex.CancelBatchOrders(t.Context(), []order.Cancel{
			{Pair: currency.NewPairWithDelimiter("BTC", "USD", currency.DashDelimiter), AssetType: asset.CoinMarginedFutures, OrderID: "1"},
			{Pair: currency.NewPairWithDelimiter("BTC", "USD", currency.DashDelimiter), AssetType: asset.CoinMarginedFutures, ClientOrderID: "c2"},
		})
		require.ErrorIs(t, err, errAPIResponse, "CancelBatchOrders must return a failed coin-M cancellation's error")
		assert.Equal(t, "CANCELLED", resp.Status["1"], "CancelBatchOrders should report a cancelled coin-M order")
		assert.Contains(t, resp.Status["c2"], "code 100500", "CancelBatchOrders should report why a coin-M order was not cancelled")
	})
}

func TestRecordBatchRequestError(t *testing.T) {
	t.Parallel()
	resp := &order.CancelBatchResponse{Status: map[string]string{}}
	err := recordBatchRequestError(resp, []string{"1", "c2"}, errAPIResponse)
	assert.ErrorIs(t, err, errAPIResponse, "recordBatchRequestError should return the request's error")
	assert.ErrorContains(t, err, "[1 c2]", "recordBatchRequestError should name the orders")
	assert.Equal(t, map[string]string{"1": errAPIResponse.Error(), "c2": errAPIResponse.Error()}, resp.Status, "recordBatchRequestError should record the error against each order")
}

func TestRecordMissingBatchResults(t *testing.T) {
	t.Parallel()
	resp := &order.CancelBatchResponse{Status: map[string]string{"1": "CANCELLED"}}
	recordMissingBatchResults(resp, []string{"1", "2"})
	assert.Equal(t, map[string]string{"1": "CANCELLED", "2": errBatchCancelResultMissing.Error()}, resp.Status, "recordMissingBatchResults should record the orders a reply left out")
}

func TestBatchKeys(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []string{"1", "2", "a"}, batchKeys([]uint64{1, 2}, []string{"a"}), "batchKeys should list the order IDs then the client order IDs")
	assert.Empty(t, batchKeys(nil, nil), "batchKeys should list nothing for an empty batch")
}

func TestCancelSpotBatch(t *testing.T) {
	t.Parallel()
	ex, requests := wrapperTestServer(t, map[string][]string{
		"POST /openApi/spot/v1/trade/cancelOrders": {
			`{"code":0,"data":{"fails":[{"orderId":"2","error":"order status :CANCELED"},{"orderId":0,"error":"unknown order"}],"orders":[{"orderId":1,"status":"CANCELED"},{"orderId":3,"status":"CANCELED","clientOrderID":"c3"}]}}`,
			`{"code":100500,"msg":"busy"}`,
		},
	})
	group := &batchCancelGroup{pair: currency.NewBTCUSDT(), orderIDs: []uint64{1, 2}, clientOrderIDs: []string{"c3", "c4"}}
	resp := &order.CancelBatchResponse{Status: map[string]string{}}
	require.NoError(t, ex.cancelSpotBatch(t.Context(), group, resp), "cancelSpotBatch must not error")
	assert.Equal(t, url.Values{"symbol": {"BTC-USDT"}, "process": {"1"}, "orderIds": {"1,2"}, "clientOrderIDs": {"c3,c4"}}, requests()[0].params, "cancelSpotBatch should cancel what can be cancelled and report the rest")
	assert.Equal(t, map[string]string{
		"1":  "CANCELED",
		"2":  "order status :CANCELED",
		"c3": "CANCELED",
		"c4": errBatchCancelResultMissing.Error(),
	}, resp.Status, "cancelSpotBatch should report each order's outcome by the ID it was requested with")
	resp = &order.CancelBatchResponse{Status: map[string]string{}}
	assert.ErrorIs(t, ex.cancelSpotBatch(t.Context(), group, resp), errAPIResponse, "cancelSpotBatch should return the request's error")
	assert.Len(t, resp.Status, 4, "cancelSpotBatch should report the request's error against each order")
}

func TestCancelSwapBatch(t *testing.T) {
	t.Parallel()
	ex, _ := wrapperTestServer(t, map[string][]string{
		"DELETE /openApi/swap/v2/trade/batchOrders": {
			`{"code":0,"data":{"success":[{"orderId":1,"status":"CANCELLED"},{"orderId":5,"clientOrderId":"mixedcase","status":"CANCELLED"}],"failed":[{"orderId":2,"errorCode":80012,"errorMessage":"cancel order failed"}]}}`,
			`{"code":100500,"msg":"busy"}`,
		},
	})
	resp := &order.CancelBatchResponse{Status: map[string]string{}}
	require.NoError(t, ex.cancelSwapBatch(t.Context(), currency.NewBTCUSDT(), []uint64{1, 2, 3}, []string{"MixedCase"}, resp), "cancelSwapBatch must not error")
	assert.Equal(t, map[string]string{
		"1":         "CANCELLED",
		"2":         (&APIError{Code: 80012, Message: "cancel order failed"}).Error(),
		"3":         errBatchCancelResultMissing.Error(),
		"MixedCase": "CANCELLED",
	}, resp.Status, "cancelSwapBatch should report each order's outcome by the ID it was requested with")
	resp = &order.CancelBatchResponse{Status: map[string]string{}}
	assert.ErrorIs(t, ex.cancelSwapBatch(t.Context(), currency.NewBTCUSDT(), []uint64{1}, nil, resp), errAPIResponse, "cancelSwapBatch should return the request's error")
	assert.Contains(t, resp.Status["1"], "code 100500", "cancelSwapBatch should report the request's error against each order")
}

func TestCancelCoinMarginedBatchOrder(t *testing.T) {
	t.Parallel()
	ex, requests := wrapperTestServer(t, map[string][]string{
		"DELETE /openApi/cswap/v1/trade/cancelOrder": {`{"code":0,"data":{"order":{"orderId":1,"status":"CANCELLED"}}}`, `{"code":100500,"msg":"busy"}`},
	})
	pair := currency.NewPairWithDelimiter("BTC", "USD", currency.DashDelimiter)
	status, err := ex.cancelCoinMarginedBatchOrder(t.Context(), pair, 0, "c1")
	require.NoError(t, err, "cancelCoinMarginedBatchOrder must not error")
	assert.Equal(t, "CANCELLED", status, "cancelCoinMarginedBatchOrder should return the cancelled status")
	assert.Equal(t, url.Values{"symbol": {"BTC-USD"}, "clientOrderId": {"c1"}}, requests()[0].params, "cancelCoinMarginedBatchOrder should cancel the order by the ID given")
	_, err = ex.cancelCoinMarginedBatchOrder(t.Context(), pair, 1, "")
	assert.ErrorIs(t, err, errAPIResponse, "cancelCoinMarginedBatchOrder should return the request's error")
}

func TestRecordCoinMarginedCancel(t *testing.T) {
	t.Parallel()
	resp := &order.CancelBatchResponse{Status: map[string]string{}}
	require.NoError(t, recordCoinMarginedCancel(resp, "1", "CANCELLED", nil), "recordCoinMarginedCancel must not error for a cancelled order")
	err := recordCoinMarginedCancel(resp, "2", "", errAPIResponse)
	assert.ErrorIs(t, err, errAPIResponse, "recordCoinMarginedCancel should return the cancellation's error")
	assert.ErrorContains(t, err, "order 2", "recordCoinMarginedCancel should name the order")
	assert.Equal(t, map[string]string{"1": "CANCELLED", "2": errAPIResponse.Error()}, resp.Status, "recordCoinMarginedCancel should record each order's outcome")
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrCancelOrderIsNil, "CancelAllOrders must reject a nil cancellation")
	_, err = e.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Futures})
	require.ErrorIs(t, err, asset.ErrNotSupported, "CancelAllOrders must reject an unsupported asset")
	_, err = e.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Futures, Side: order.Buy})
	require.ErrorIs(t, err, asset.ErrNotSupported, "CancelAllOrders must reject an unsupported asset for a side")

	t.Run("recorded", func(t *testing.T) {
		t.Parallel()
		ex := newWrapperTestExchange(t)
		wrapperSkipIfCannotManipulateOrders(t, ex)
		if !mockTests {
			t.Skip("the recorded cancellations are documented examples, which do not name live orders")
		}
		for _, tc := range []struct {
			c   *order.Cancel
			exp map[string]string
		}{
			{&order.Cancel{Pair: currency.NewPairWithDelimiter("GM", "USDT", currency.DashDelimiter), AssetType: asset.Spot}, map[string]string{"1735964997957275600": "CANCELED", "1735965127519326200": "CANCELED"}},
			{&order.Cancel{Pair: currency.NewPairWithDelimiter("ATOM", "USDT", currency.DashDelimiter), AssetType: asset.USDTMarginedFutures}, map[string]string{
				"1736013373487123500": "CANCELLED",
				"111111":              (&APIError{Code: 80012, Message: "cancel order failed"}).Error(),
				"222222":              (&APIError{Code: 80012, Message: "cancel order failed"}).Error(),
			}},
			// The documented reply lists its one order as both cancelled and failed
			{&order.Cancel{Pair: currency.NewPairWithDelimiter("BTC", "USD", currency.DashDelimiter), AssetType: asset.CoinMarginedFutures}, map[string]string{"1801610628516806656": (&APIError{Code: 123}).Error()}},
		} {
			resp, err := ex.CancelAllOrders(t.Context(), tc.c)
			require.NoErrorf(t, err, "CancelAllOrders must not error for %s", tc.c.AssetType)
			assert.Equalf(t, tc.exp, resp.Status, "CancelAllOrders should report each %s order's outcome", tc.c.AssetType)
		}
	})

	t.Run("side and type", func(t *testing.T) {
		t.Parallel()
		for _, a := range []asset.Item{asset.Spot, asset.USDTMarginedFutures, asset.CoinMarginedFutures} {
			for _, tc := range []struct {
				name       string
				side       order.Side
				orderType  order.Type
				everyPair  bool
				listFails  bool
				cancelFail string
				exp        []string
			}{
				{name: "buy limit", side: order.Buy, orderType: order.Limit, exp: []string{"1"}},
				{name: "sell", side: order.Sell, exp: []string{"2"}},
				{name: "sell of any type", side: order.Sell, orderType: order.AnyType, exp: []string{"2"}},
				{name: "limit", orderType: order.Limit, exp: []string{"1", "2", "5"}},
				{name: "limit of any side", side: order.AnySide, orderType: order.Limit, exp: []string{"1", "2", "5"}},
				{name: "buy", side: order.Buy, exp: []string{"1", "3", "4"}},
				{name: "stop limit", orderType: order.StopLimit, exp: []string{"3"}},
				{name: "buy limit on every symbol", side: order.Buy, orderType: order.Limit, everyPair: true, exp: []string{"1", "6"}},
				{name: "nothing matching", orderType: order.Market},
				{name: "listing error", side: order.Buy, listFails: true},
				{name: "cancellation failure", side: order.Sell, cancelFail: "2", exp: []string{"2"}},
			} {
				t.Run(a.String()+" "+tc.name, func(t *testing.T) {
					t.Parallel()
					symbols := [2]string{"BTC-USDT", "ETH-USDT"}
					listRoute, cancelRoute := "/openApi/spot/v1/trade/openOrders", "/openApi/spot/v1/trade/cancelOrders"
					switch a {
					case asset.USDTMarginedFutures:
						listRoute, cancelRoute = "/openApi/swap/v2/trade/openOrders", "/openApi/swap/v2/trade/batchOrders"
					case asset.CoinMarginedFutures:
						symbols = [2]string{"BTC-USD", "ETH-USD"}
						listRoute, cancelRoute = "/openApi/cswap/v1/trade/openOrders", "/openApi/cswap/v1/trade/cancelOrder"
					}
					var mu sync.Mutex
					var listed []url.Values
					cancelled := make(map[string]string)
					ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
						params := swapTradeBusinessParams(r)
						mu.Lock()
						defer mu.Unlock()
						switch r.URL.Path {
						case listRoute:
							listed = append(listed, params)
							if tc.listFails {
								_, _ = io.WriteString(w, `{"code":100500,"msg":"busy"}`)
								return
							}
							orders := []string{
								`{"symbol":"` + symbols[0] + `","orderId":1,"side":"BUY","type":"LIMIT","status":"NEW"}`,
								`{"symbol":"` + symbols[0] + `","orderId":2,"side":"SELL","type":"LIMIT","status":"NEW"}`,
								`{"symbol":"` + symbols[0] + `","orderId":3,"side":"BUY","type":"TRIGGER_LIMIT","status":"NEW"}`,
								`{"symbol":"` + symbols[0] + `","orderId":4,"side":"BUY","type":"NEW_TYPE","status":"NEW"}`,
								`{"symbol":"` + symbols[0] + `","orderId":5,"side":"NEW_SIDE","type":"LIMIT","status":"NEW"}`,
							}
							if !params.Has("symbol") {
								orders = append(orders, `{"symbol":"`+symbols[1]+`","orderId":6,"side":"BUY","type":"LIMIT","status":"NEW"}`)
							}
							_, _ = io.WriteString(w, `{"code":0,"data":{"orders":[`+strings.Join(orders, ",")+`]}}`)
						case cancelRoute:
							var ids []string
							switch a {
							case asset.Spot:
								ids = strings.Split(params.Get("orderIds"), ",")
							case asset.USDTMarginedFutures:
								ids = strings.Split(strings.Trim(params.Get("orderIdList"), "[]"), ",")
							default:
								ids = []string{params.Get("orderId")}
							}
							for _, id := range ids {
								cancelled[id] = params.Get("symbol")
							}
							var replies []string
							for _, id := range []string{"1", "2", "3", "4", "5", "6"} {
								if slices.Contains(ids, id) {
									replies = append(replies, `{"orderId":`+id+`,"status":"CANCELED"}`)
								}
							}
							switch {
							case len(ids) == 1 && ids[0] == tc.cancelFail:
								_, _ = io.WriteString(w, `{"code":100500,"msg":"busy"}`)
							case a == asset.Spot:
								_, _ = io.WriteString(w, `{"code":0,"data":{"fails":[],"orders":[`+strings.Join(replies, ",")+`]}}`)
							case a == asset.USDTMarginedFutures:
								_, _ = io.WriteString(w, `{"code":0,"data":{"success":[`+strings.Join(replies, ",")+`],"failed":null}}`)
							default:
								_, _ = io.WriteString(w, `{"code":0,"data":{"order":`+replies[0]+`}}`)
							}
						default:
							assert.Failf(t, "unexpected request", "%s %s should not be requested", r.Method, r.URL.Path)
							w.WriteHeader(http.StatusNotFound)
						}
					})
					req := &order.Cancel{AssetType: a, Side: tc.side, Type: tc.orderType}
					if !tc.everyPair {
						req.Pair = currency.NewPairWithDelimiter(strings.Split(symbols[0], "-")[0], strings.Split(symbols[0], "-")[1], currency.DashDelimiter)
					}
					resp, err := ex.CancelAllOrders(t.Context(), req)
					mu.Lock()
					defer mu.Unlock()
					require.Len(t, listed, 1, "CancelAllOrders must list the open orders once")
					if tc.everyPair {
						assert.Empty(t, listed[0], "CancelAllOrders should list every symbol's open orders without a pair")
					} else {
						assert.Equal(t, url.Values{"symbol": {symbols[0]}}, listed[0], "CancelAllOrders should list the pair's open orders")
					}
					if tc.listFails {
						assert.ErrorIs(t, err, errAPIResponse, "CancelAllOrders should return the listing's error")
						assert.Empty(t, cancelled, "CancelAllOrders should cancel nothing when the listing fails")
						return
					}
					expCancelled := make(map[string]string, len(tc.exp))
					expStatus := make(map[string]string, len(tc.exp))
					for _, id := range tc.exp {
						expCancelled[id] = symbols[0]
						expStatus[id] = "CANCELED"
					}
					if tc.everyPair {
						expCancelled["6"] = symbols[1]
					}
					assert.Equal(t, expCancelled, cancelled, "CancelAllOrders should cancel the matching orders alone, each on its own symbol")
					if tc.cancelFail != "" {
						require.ErrorIs(t, err, errAPIResponse, "CancelAllOrders must return a failed cancellation's error")
						assert.Contains(t, resp.Status[tc.cancelFail], "code 100500", "CancelAllOrders should report why an order was not cancelled")
						return
					}
					require.NoError(t, err, "CancelAllOrders must not error")
					assert.Equal(t, expStatus, resp.Status, "CancelAllOrders should report each matching order's outcome")
				})
			}
		}
	})

	t.Run("credentials changing during a narrowed cancellation", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			a      asset.Item
			pair   currency.Pair
			routes map[string][]string
		}{
			{asset.Spot, currency.NewBTCUSDT(), map[string][]string{
				"GET /openApi/spot/v1/trade/openOrders":    {wrapperCredentialsChange + `{"code":0,"data":{"orders":[{"symbol":"BTC-USDT","orderId":1,"side":"BUY","type":"LIMIT","status":"NEW"}]}}`},
				"POST /openApi/spot/v1/trade/cancelOrders": {`{"code":0,"data":{"fails":[],"orders":[{"orderId":1,"status":"CANCELED"}]}}`},
			}},
			{asset.USDTMarginedFutures, currency.NewBTCUSDT(), map[string][]string{
				"GET /openApi/swap/v2/trade/openOrders":     {wrapperCredentialsChange + `{"code":0,"data":{"orders":[{"symbol":"BTC-USDT","orderId":1,"side":"BUY","type":"LIMIT","status":"NEW"}]}}`},
				"DELETE /openApi/swap/v2/trade/batchOrders": {`{"code":0,"data":{"success":[{"orderId":1,"status":"CANCELLED"}],"failed":null}}`},
			}},
			{asset.CoinMarginedFutures, currency.NewBTCUSD(), map[string][]string{
				"GET /openApi/cswap/v1/trade/openOrders":     {wrapperCredentialsChange + `{"code":0,"data":{"orders":[{"symbol":"BTC-USD","orderId":1,"side":"BUY","type":"LIMIT","status":"NEW"}]}}`},
				"DELETE /openApi/cswap/v1/trade/cancelOrder": {`{"code":0,"data":{"order":{"orderId":1,"status":"CANCELLED"}}}`},
			}},
		} {
			for _, c := range []order.Cancel{{Side: order.Buy}, {Type: order.Limit}} {
				t.Run(tc.a.String()+" side "+c.Side.String()+" type "+c.Type.String(), func(t *testing.T) {
					t.Parallel()
					ex, requests := wrapperTestServer(t, maps.Clone(tc.routes))
					c.Pair, c.AssetType = tc.pair, tc.a
					_, err := ex.CancelAllOrders(t.Context(), &c)
					require.NoError(t, err, "CancelAllOrders must not error")
					got := requests()
					require.Len(t, got, 2, "CancelAllOrders must list the open orders and cancel the matching one")
					for _, r := range got {
						assert.Equalf(t, testAPIKey, r.apiKey, "CancelAllOrders should send %s %s for the account it started with", r.method, r.path)
					}
				})
			}
		}
		_, err := new(Exchange).CancelAllOrders(t.Context(), &order.Cancel{Pair: currency.NewBTCUSDT(), AssetType: asset.Spot, Side: order.Buy})
		assert.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "CancelAllOrders should return the credentials error")
	})

	t.Run("context credentials replaced during a narrowed cancellation", func(t *testing.T) {
		t.Parallel()
		var mu sync.Mutex
		var keys []string
		var store *accounts.ContextCredentialsStore
		ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			defer mu.Unlock()
			keys = append(keys, r.Header.Get(keyHeader))
			if r.URL.Path == "/openApi/spot/v1/trade/openOrders" {
				store.Load(&accounts.Credentials{Key: wrapperOtherAPIKey, Secret: "other-secret"})
				_, _ = io.WriteString(w, `{"code":0,"data":{"orders":[{"symbol":"BTC-USDT","orderId":1,"side":"BUY","type":"LIMIT","status":"NEW"}]}}`)
				return
			}
			_, _ = io.WriteString(w, `{"code":0,"data":{"fails":[],"orders":[{"orderId":1,"status":"CANCELED"}]}}`)
		})
		ctx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "context-key", Secret: "context-secret"})
		mu.Lock()
		store, _ = ctx.Value(accounts.ContextCredentialsFlag).(*accounts.ContextCredentialsStore)
		mu.Unlock()
		require.NotNil(t, store, "the context must hold a credentials store")
		_, err := ex.CancelAllOrders(ctx, &order.Cancel{Pair: currency.NewBTCUSDT(), AssetType: asset.Spot, Side: order.Buy})
		require.NoError(t, err, "CancelAllOrders must not error")
		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, []string{"context-key", "context-key"}, keys, "CancelAllOrders should list and cancel for the account the context named when it started")
	})

	t.Run("any side and type", func(t *testing.T) {
		t.Parallel()
		busy := []string{`{"code":100500,"msg":"busy"}`, `{"code":100500,"msg":"busy"}`, `{"code":100500,"msg":"busy"}`}
		ex, requests := wrapperTestServer(t, map[string][]string{
			"POST /openApi/spot/v1/trade/cancelOpenOrders": slices.Clone(busy),
			"DELETE /openApi/swap/v2/trade/allOpenOrders":  slices.Clone(busy),
			"DELETE /openApi/cswap/v1/trade/allOpenOrders": slices.Clone(busy),
		})
		for _, a := range []asset.Item{asset.Spot, asset.USDTMarginedFutures, asset.CoinMarginedFutures} {
			for _, c := range []order.Cancel{{Side: order.AnySide, Type: order.AnyType}, {Side: order.AnySide}, {Type: order.AnyType}} {
				c.AssetType = a
				_, err := ex.CancelAllOrders(t.Context(), &c)
				assert.ErrorIsf(t, err, errAPIResponse, "CancelAllOrders should cancel every %s order with side %s and type %s through the cancel-all endpoint", a, c.Side, c.Type)
			}
		}
		assert.Len(t, requests(), 9, "CancelAllOrders should send one cancel-all request for each and list nothing")
	})

	t.Run("request errors", func(t *testing.T) {
		t.Parallel()
		ex, requests := wrapperTestServer(t, map[string][]string{
			"POST /openApi/spot/v1/trade/cancelOpenOrders": {`{"code":100500,"msg":"busy"}`},
			"DELETE /openApi/swap/v2/trade/allOpenOrders":  {`{"code":100500,"msg":"busy"}`},
			"DELETE /openApi/cswap/v1/trade/allOpenOrders": {`{"code":100500,"msg":"busy"}`},
		})
		for _, a := range ex.GetAssetTypes(false) {
			_, err := ex.CancelAllOrders(t.Context(), &order.Cancel{AssetType: a})
			assert.ErrorIsf(t, err, errAPIResponse, "CancelAllOrders should return the %s request's error", a)
		}
		for _, r := range requests() {
			assert.Emptyf(t, r.params, "CancelAllOrders should cancel every symbol's orders without a pair on %s", r.path)
		}
	})
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderInfo(t.Context(), "1", currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetOrderInfo must reject an empty pair")
	_, err = e.GetOrderInfo(t.Context(), "", currency.NewBTCUSDT(), asset.Spot)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetOrderInfo must reject an empty order ID")
	_, err = e.GetOrderInfo(t.Context(), "x", currency.NewBTCUSDT(), asset.Spot)
	require.ErrorIs(t, err, errInvalidOrderID, "GetOrderInfo must reject an invalid order ID")
	_, err = e.GetOrderInfo(t.Context(), "1", currency.NewBTCUSDT(), asset.Futures)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetOrderInfo must reject an unsupported asset")

	t.Run("recorded", func(t *testing.T) {
		t.Parallel()
		ex := newWrapperTestExchange(t)
		sharedtestvalues.SkipTestIfCredentialsUnset(t, ex)
		if !mockTests {
			t.Skip("the recorded orders are documented examples, which do not name live orders")
		}
		brg := currency.NewPairWithDelimiter("BRG", "USDT", currency.DashDelimiter)
		opUSDT := currency.NewPairWithDelimiter("OP", "USDT", currency.DashDelimiter)
		opUSD := currency.NewPairWithDelimiter("OP", "USD", currency.DashDelimiter)
		for _, tc := range []struct {
			a       asset.Item
			pair    currency.Pair
			orderID string
			exp     order.Detail
		}{
			{asset.Spot, brg, "1735965009395131234", order.Detail{
				Price: 0.0005027, Amount: 4038, AverageExecutedPrice: 0.0005028, RemainingAmount: 4038, CostAsset: currency.USDT,
				FeeAsset: currency.NewCode("BRG"), OrderID: "1735965009395131100", Type: order.Limit, Side: order.Buy, Status: order.Pending,
				Date: time.UnixMilli(1702721285567), LastUpdated: time.UnixMilli(1702721285567),
			}},
			{asset.USDTMarginedFutures, opUSDT, "1736012449498123456", order.Detail{
				TimeInForce: order.GoodTillCancel, Price: 2.171, Amount: 1, RemainingAmount: 1, CostAsset: currency.USDT, FeeAsset: currency.USDT,
				OrderID: "1736012449498123500", ClientOrderID: "l*****e", Type: order.Limit, Side: order.Sell, Status: order.Pending,
				Date: time.UnixMilli(1702732596168), LastUpdated: time.UnixMilli(1702732596188), SettlementCurrency: currency.USDT,
			}},
			{asset.CoinMarginedFutures, opUSD, "1736012449498123456", order.Detail{
				Price: 2.171, Amount: 1, RemainingAmount: 1, FeeAsset: currency.NewCode("OP"), OrderID: "1736012449498123500", Type: order.Limit,
				Side: order.Sell, Status: order.Pending, Date: time.UnixMilli(1702732596168), LastUpdated: time.UnixMilli(1702732596188),
				SettlementCurrency: currency.NewCode("OP"),
			}},
		} {
			result, err := ex.GetOrderInfo(t.Context(), tc.orderID, tc.pair, tc.a)
			require.NoErrorf(t, err, "GetOrderInfo must not error for %s", tc.a)
			tc.exp.Exchange, tc.exp.AssetType, tc.exp.Pair = ex.Name, tc.a, tc.pair
			assert.Equalf(t, &tc.exp, result, "GetOrderInfo should convert the %s order", tc.a)
		}
	})

	t.Run("errors", func(t *testing.T) {
		t.Parallel()
		ex, _ := wrapperTestServer(t, map[string][]string{
			"GET /openApi/spot/v1/trade/query":        {`{"code":100404,"msg":"order not exist"}`, `{"code":0,"data":{"symbol":"BTC-USDT","orderId":1,"status":"ODD","type":"LIMIT","side":"BUY"}}`},
			"GET /openApi/swap/v2/trade/order":        {`{"code":100500,"msg":"busy"}`, `{"code":0,"data":{"order":{"symbol":"BTC-USDT","orderId":1,"status":"NEW","type":"ODD","side":"BUY"}}}`},
			"GET /openApi/cswap/v1/trade/orderDetail": {`{"code":100500,"msg":"busy"}`, `{"code":0,"data":{"order":{"symbol":"BTC-USD","orderId":1,"status":"Pending","type":"LIMIT","side":"ODD"}}}`},
		})
		_, err := ex.GetOrderInfo(t.Context(), "1", currency.NewBTCUSDT(), asset.Spot)
		assert.ErrorIs(t, err, order.ErrOrderNotFound, "GetOrderInfo should report a spot order BingX does not hold")
		for _, a := range ex.GetAssetTypes(false) {
			if a != asset.Spot {
				_, err = ex.GetOrderInfo(t.Context(), "1", currency.NewBTCUSDT(), a)
				assert.ErrorIsf(t, err, errAPIResponse, "GetOrderInfo should return the %s request's error", a)
			}
			_, err = ex.GetOrderInfo(t.Context(), "1", currency.NewBTCUSDT(), a)
			assert.ErrorContainsf(t, err, "error converting", "GetOrderInfo should report a %s order it cannot convert", a)
		}
	})
}

func TestSpotOrderDetail(t *testing.T) {
	t.Parallel()
	pair := currency.NewBTCUSDT()
	o := &SpotOrderDetailResponse{
		Symbol: "BTC-USDT", OrderID: 5, Price: 10, StopPrice: 9, OriginalQuantity: 3, ExecutedQuantity: 1, CummulativeQuoteQuantity: 10.5,
		Status: "PARTIALLY_FILLED", Type: OrderTypeTakeStopLimit, Side: SideSell, Time: types.Time(time.UnixMilli(1)), UpdateTime: types.Time(time.UnixMilli(2)),
		OriginalQuoteOrderQuantity: 30, Fee: -0.01, FeeAsset: currency.USDT, ClientOrderID: "c5", AveragePrice: 10.5,
	}
	d, err := e.spotOrderDetail(o, pair)
	require.NoError(t, err, "spotOrderDetail must not error")
	assert.Equal(t, order.Detail{
		Price: 10, Amount: 3, TriggerPrice: 9, AverageExecutedPrice: 10.5, QuoteAmount: 30, ExecutedAmount: 1, RemainingAmount: 2, Cost: 10.5,
		CostAsset: currency.USDT, Fee: 0.01, FeeAsset: currency.USDT, Exchange: e.Name, OrderID: "5", ClientOrderID: "c5", Type: order.StopLimit,
		Side: order.Sell, Status: order.PartiallyFilled, AssetType: asset.Spot, Date: time.UnixMilli(1), LastUpdated: time.UnixMilli(2), Pair: pair,
	}, d, "spotOrderDetail should convert every value, the fee as paid")
	o.Status, o.Type, o.Side = "ODD", "ODD", "ODD"
	d, err = e.spotOrderDetail(o, pair)
	assert.ErrorIs(t, err, order.ErrUnsupportedStatusType, "spotOrderDetail should report a status it cannot convert")
	assert.ErrorIs(t, err, order.ErrUnrecognisedOrderType, "spotOrderDetail should report a type it cannot convert")
	assert.ErrorIs(t, err, order.ErrSideIsInvalid, "spotOrderDetail should report a side it cannot convert")
	assert.Equal(t, "5", d.OrderID, "spotOrderDetail should convert the other values")
}

func TestSwapOrderDetail(t *testing.T) {
	t.Parallel()
	pair := currency.NewPairWithDelimiter("BTC", "USDC", currency.DashDelimiter)
	o := &SwapOrderResponse{
		Symbol: "BTC-USDC", OrderID: 6, Side: SideBuy, PositionSide: PositionSideLong, Type: OrderTypeLimit, OriginalQuantity: 2, Price: 10,
		ExecutedQuantity: 0.5, AveragePrice: 10, CumulativeQuote: 5, StopPrice: 0, Commission: -0.003779, Status: "PARTIALLY_FILLED",
		Time: types.Time(time.UnixMilli(1)), UpdateTime: types.Time(time.UnixMilli(2)), ClientOrderID: "c6", Leverage: "20X", PostOnly: true,
		ClosePosition: true,
	}
	d, err := e.swapOrderDetail(o, pair)
	require.NoError(t, err, "swapOrderDetail must not error")
	assert.Equal(t, order.Detail{
		TimeInForce: order.PostOnly, ReduceOnly: true, Leverage: 20, Price: 10, Amount: 2, AverageExecutedPrice: 10, ExecutedAmount: 0.5,
		RemainingAmount: 1.5, Cost: 5, CostAsset: currency.USDC, Fee: 0.003779, FeeAsset: currency.USDC, Exchange: e.Name, OrderID: "6",
		ClientOrderID: "c6", Type: order.Limit, Side: order.Buy, Status: order.PartiallyFilled, AssetType: asset.USDTMarginedFutures,
		Date: time.UnixMilli(1), LastUpdated: time.UnixMilli(2), Pair: pair, SettlementCurrency: currency.USDC,
	}, d, "swapOrderDetail should convert every value, in the settlement currency")
	trailing := &SwapOrderResponse{OrderID: 7, Type: OrderTypeTrailingStopMarket, Side: SideSell, Status: "NEW", Price: 5, ActivationPrice: 100}
	d, err = e.swapOrderDetail(trailing, pair)
	require.NoError(t, err, "swapOrderDetail must not error")
	assert.Zero(t, d.Price, "swapOrderDetail should leave a trailing stop's distance out of its price")
	assert.Equal(t, 100.0, d.TriggerPrice, "swapOrderDetail should trigger a trailing stop at its activation price")
	assert.Equal(t, order.TrailingStop, d.Type, "swapOrderDetail should convert a trailing stop")
	o.Status, o.Type, o.Side, o.Leverage = "ODD", "ODD", "ODD", "highX"
	_, err = e.swapOrderDetail(o, pair)
	assert.ErrorIs(t, err, order.ErrUnsupportedStatusType, "swapOrderDetail should report a status it cannot convert")
	assert.ErrorIs(t, err, order.ErrUnrecognisedOrderType, "swapOrderDetail should report a type it cannot convert")
	assert.ErrorIs(t, err, order.ErrSideIsInvalid, "swapOrderDetail should report a side it cannot convert")
	assert.ErrorIs(t, err, errInvalidOrderLeverage, "swapOrderDetail should report a leverage it cannot read")
}

func TestCoinMarginedOrderDetail(t *testing.T) {
	t.Parallel()
	pair := currency.NewPairWithDelimiter("BTC", "USD", currency.DashDelimiter)
	o := &CoinMarginedOrderResponse{
		Symbol: "BTC-USD", OrderID: 8, Side: SideSell, PositionSide: "Short", Type: OrderTypeLimit, Quantity: 4, Price: 70000, ExecutedQuantity: 1,
		AveragePrice: 70000, Commission: -0.000002, Status: "Pending", Time: types.Time(time.UnixMilli(1)), UpdateTime: types.Time(time.UnixMilli(2)),
		ClientOrderID: "c8", Leverage: "15X", ReduceOnly: true, TimeInForce: TimeInForceImmediateOrCancel,
	}
	d, err := e.coinMarginedOrderDetail(o, pair)
	require.NoError(t, err, "coinMarginedOrderDetail must not error")
	assert.Equal(t, order.Detail{
		TimeInForce: order.ImmediateOrCancel, ReduceOnly: true, Leverage: 15, Price: 70000, Amount: 4, AverageExecutedPrice: 70000,
		ExecutedAmount: 1, RemainingAmount: 3, Fee: 0.000002, FeeAsset: currency.BTC, Exchange: e.Name, OrderID: "8", ClientOrderID: "c8",
		Type: order.Limit, Side: order.Sell, Status: order.Pending, AssetType: asset.CoinMarginedFutures, Date: time.UnixMilli(1),
		LastUpdated: time.UnixMilli(2), Pair: pair, SettlementCurrency: currency.BTC,
	}, d, "coinMarginedOrderDetail should convert every value, taking the contracts from quantity without origQty")
	o.OriginalQuantity, o.PostOnly = 5, true
	d, err = e.coinMarginedOrderDetail(o, pair)
	require.NoError(t, err, "coinMarginedOrderDetail must not error")
	assert.Equal(t, 5.0, d.Amount, "coinMarginedOrderDetail should take the contracts from origQty")
	assert.Equal(t, order.PostOnly, d.TimeInForce, "coinMarginedOrderDetail should read a post only order")
	o.Status, o.Type, o.Side, o.TimeInForce, o.Leverage = "ODD", "ODD", "ODD", "GTX", "highX"
	_, err = e.coinMarginedOrderDetail(o, pair)
	assert.ErrorIs(t, err, order.ErrUnsupportedStatusType, "coinMarginedOrderDetail should report a status it cannot convert")
	assert.ErrorIs(t, err, order.ErrUnrecognisedOrderType, "coinMarginedOrderDetail should report a type it cannot convert")
	assert.ErrorIs(t, err, order.ErrSideIsInvalid, "coinMarginedOrderDetail should report a side it cannot convert")
	assert.ErrorIs(t, err, order.ErrInvalidTimeInForce, "coinMarginedOrderDetail should report a time in force it cannot convert")
	assert.ErrorIs(t, err, errInvalidOrderLeverage, "coinMarginedOrderDetail should report a leverage it cannot read")
}

func TestAppendListedOrder(t *testing.T) {
	t.Parallel()
	ex, _ := wrapperTestServer(t, map[string][]string{})
	convert := func(pair currency.Pair) (order.Detail, error) {
		return order.Detail{OrderID: "1", Pair: pair}, nil
	}
	requested := currency.NewPairWithDelimiter("XRP", "USDT", currency.DashDelimiter)
	orders := ex.appendListedOrder(nil, asset.Spot, "1", requested, "ETH-USDT", convert)
	require.Len(t, orders, 1, "appendListedOrder must add the order")
	assert.Equal(t, requested, orders[0].Pair, "appendListedOrder should keep the pair requested")
	orders = ex.appendListedOrder(orders, asset.Spot, "2", currency.EMPTYPAIR, "GPT-OLD-USDT", convert)
	require.Len(t, orders, 2, "appendListedOrder must add the order")
	assert.Equal(t, "GPT-OLD-USDT", orders[1].Pair.String(), "appendListedOrder should take the pair the symbol names")
	orders = ex.appendListedOrder(orders, asset.Spot, "3", currency.EMPTYPAIR, "NODASH", convert)
	assert.Len(t, orders, 2, "appendListedOrder should leave out an order whose symbol cannot be read")
	orders = ex.appendListedOrder(orders, asset.Spot, "4", requested, "XRP-USDT", func(pair currency.Pair) (order.Detail, error) {
		return order.Detail{OrderID: "4", Pair: pair}, order.ErrUnsupportedStatusType
	})
	require.Len(t, orders, 3, "appendListedOrder must keep an order with a value it cannot convert")
	assert.Equal(t, "4", orders[2].OrderID, "appendListedOrder should keep the order's other values")
}

func TestListingPairs(t *testing.T) {
	t.Parallel()
	pair := currency.NewBTCUSDT()
	assert.Equal(t, currency.Pairs{pair}, listingPairs(currency.Pairs{pair}), "listingPairs should request the one pair asked for")
	assert.Equal(t, currency.Pairs{currency.EMPTYPAIR}, listingPairs(nil), "listingPairs should request every symbol when no pair is asked for")
	assert.Equal(t, currency.Pairs{currency.EMPTYPAIR}, listingPairs(currency.Pairs{pair, currency.NewPair(currency.ETH, currency.USDT)}), "listingPairs should request every symbol in one request for several pairs")
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetActiveOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrGetOrdersRequestIsNil, "GetActiveOrders must reject a nil request")
	_, err = e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Futures, Side: order.AnySide, Type: order.AnyType})
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetActiveOrders must reject an unsupported asset")

	t.Run("recorded", func(t *testing.T) {
		t.Parallel()
		ex := newWrapperTestExchange(t)
		sharedtestvalues.SkipTestIfCredentialsUnset(t, ex)
		if !mockTests {
			t.Skip("the recorded orders are documented examples on other pairs")
		}
		for _, tc := range []struct {
			a      asset.Item
			pair   currency.Pair
			prices []float64
		}{
			{asset.Spot, currency.NewPairWithDelimiter("BNB", "USDC", currency.DashDelimiter), []float64{255.27}},
			{asset.USDTMarginedFutures, currency.NewBTCUSDT(), []float64{44459.6, 44454.6}},
			{asset.CoinMarginedFutures, currency.NewPairWithDelimiter("BTC", "USD", currency.DashDelimiter), []float64{44459.6, 44454.6}},
		} {
			orders, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: tc.a, Pairs: currency.Pairs{tc.pair}, Side: order.AnySide, Type: order.AnyType})
			require.NoErrorf(t, err, "GetActiveOrders must not error for %s", tc.a)
			require.Lenf(t, orders, len(tc.prices), "GetActiveOrders must return every open %s order", tc.a)
			for i, price := range tc.prices {
				assert.Equalf(t, price, orders[i].Price, "GetActiveOrders should convert the %s order", tc.a)
				assert.Equalf(t, tc.pair, orders[i].Pair, "GetActiveOrders should list the %s order under the pair requested", tc.a)
				assert.Equalf(t, order.Pending, orders[i].Status, "GetActiveOrders should convert the %s order's status", tc.a)
			}
		}
	})

	t.Run("every symbol", func(t *testing.T) {
		t.Parallel()
		ex, requests := wrapperTestServer(t, map[string][]string{
			"GET /openApi/spot/v1/trade/openOrders":  {`{"code":0,"data":{"orders":[{"symbol":"ETH-USDT","orderId":1,"price":"2","origQty":"1","status":"NEW","type":"LIMIT","side":"BUY"},{"symbol":"NODASH","orderId":2,"status":"NEW","type":"LIMIT","side":"BUY"}]}}`, `{"code":100500,"msg":"busy"}`},
			"GET /openApi/swap/v2/trade/openOrders":  {`{"code":100500,"msg":"busy"}`},
			"GET /openApi/cswap/v1/trade/openOrders": {`{"code":100500,"msg":"busy"}`},
		})
		orders, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSDT(), currency.NewPairWithDelimiter("ETH", "USDT", currency.DashDelimiter)}, Side: order.AnySide, Type: order.AnyType})
		require.NoError(t, err, "GetActiveOrders must not error")
		require.Len(t, orders, 1, "GetActiveOrders must return the requested pairs' orders")
		assert.Equal(t, "ETH-USDT", orders[0].Pair.String(), "GetActiveOrders should take each order's pair from its symbol")
		assert.Empty(t, requests()[0].params, "GetActiveOrders should list several pairs' orders in one request for every symbol")
		for _, a := range ex.GetAssetTypes(false) {
			_, err = ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: a, Side: order.AnySide, Type: order.AnyType})
			assert.ErrorIsf(t, err, errAPIResponse, "GetActiveOrders should return the %s request's error", a)
		}
	})
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderHistory(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrGetOrdersRequestIsNil, "GetOrderHistory must reject a nil request")
	now := time.Now()
	_, err = e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType, StartTime: now, EndTime: now.Add(-time.Hour)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetOrderHistory must reject a start after its end")
	_, err = e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType, FromOrderID: "x"})
	require.ErrorIs(t, err, errInvalidOrderID, "GetOrderHistory must reject an invalid order ID")
	_, err = e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Futures, Side: order.AnySide, Type: order.AnyType})
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetOrderHistory must reject an unsupported asset")

	t.Run("recorded", func(t *testing.T) {
		t.Parallel()
		ex := newWrapperTestExchange(t)
		sharedtestvalues.SkipTestIfCredentialsUnset(t, ex)
		if !mockTests {
			t.Skip("the recorded orders are documented examples in past windows")
		}
		swch := currency.NewPairWithDelimiter("SWCH", "USDT", currency.DashDelimiter)
		orders, err := ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Pairs: currency.Pairs{swch}, Side: order.AnySide, Type: order.AnyType, StartTime: time.UnixMilli(1702720925417), EndTime: time.UnixMilli(1702721825418)})
		require.NoError(t, err, "GetOrderHistory must not error")
		require.Len(t, orders, 1, "GetOrderHistory must return the spot order")
		assert.Equal(t, order.Detail{
			Price: 0.8548, Amount: 5.9, AverageExecutedPrice: 0.8881, ExecutedAmount: 1.1, RemainingAmount: 4.8, Cost: 0.9402800000000001,
			CostAsset: currency.USDT, Exchange: ex.Name, OrderID: "1735966927102231300", Type: order.Limit, Side: order.Sell, Status: order.Cancelled,
			AssetType: asset.Spot, Date: time.UnixMilli(1702721743000), LastUpdated: time.UnixMilli(1702721743000), Pair: swch,
		}, orders[0], "GetOrderHistory should convert the spot order")
		// The recorded perpetual orders fall outside the documented windows, so they are filtered out
		for _, tc := range []struct {
			a    asset.Item
			pair currency.Pair
		}{
			{asset.USDTMarginedFutures, currency.NewPairWithDelimiter("BTC", "USDC", currency.DashDelimiter)},
			{asset.CoinMarginedFutures, currency.NewPairWithDelimiter("BTC", "USD", currency.DashDelimiter)},
		} {
			orders, err := ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: tc.a, Pairs: currency.Pairs{tc.pair}, Side: order.AnySide, Type: order.AnyType, StartTime: time.UnixMilli(1785503625603), EndTime: time.UnixMilli(1786021625603)})
			require.NoErrorf(t, err, "GetOrderHistory must not error for %s", tc.a)
			assert.Emptyf(t, orders, "GetOrderHistory should filter out %s orders outside the window", tc.a)
		}
	})

	t.Run("windows", func(t *testing.T) {
		t.Parallel()
		ex, requests := wrapperTestServer(t, map[string][]string{
			"GET /openApi/spot/v1/trade/historyOrders": {`{"code":0,"data":{"orders":[{"symbol":"BTC-USDT","orderId":3,"status":"FILLED","type":"LIMIT","side":"BUY","time":1700000000000}]}}`, `{"code":0,"data":{"orders":[]}}`},
			"GET /openApi/swap/v2/trade/allOrders":     {`{"code":0,"data":{"orders":[]}}`, `{"code":100500,"msg":"busy"}`},
		})
		start := time.Now().Add(-time.Hour).Truncate(time.Millisecond)
		orders, err := ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSDT()}, Side: order.AnySide, Type: order.AnyType, StartTime: start})
		require.NoError(t, err, "GetOrderHistory must not error")
		assert.Empty(t, orders, "GetOrderHistory should filter out orders before the window")
		got := requests()
		require.Len(t, got, 1, "GetOrderHistory must request the window")
		assert.Equal(t, wrapperMilliseconds(start), got[0].params.Get("startTime"), "GetOrderHistory should start the window at the request's start")
		assert.NotEmpty(t, got[0].params.Get("endTime"), "GetOrderHistory should end a window without an end now")
		_, err = ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType, FromOrderID: "3"})
		require.NoError(t, err, "GetOrderHistory must not error")
		assert.Equal(t, url.Values{"orderId": {"3"}, "pageIndex": {"1"}, "pageSize": {"100"}}, requests()[1].params, "GetOrderHistory should page every symbol's orders from the order ID")
		_, err = ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.USDTMarginedFutures, Side: order.AnySide, Type: order.AnyType})
		assert.ErrorIs(t, err, errAPIResponse, "GetOrderHistory should return a request's error")
	})
}

func TestWindowedOrders(t *testing.T) {
	t.Parallel()
	start := time.UnixMilli(1700000000000)
	// fetched holds the windows fetched, as milliseconds from start
	var fetched [][2]int64
	offset := func(t time.Time) int64 { return t.Sub(start).Milliseconds() }
	// Each window holds the orders opened in it, one a millisecond up to the sixth, and is cut short beyond three
	fetch := func(from, to time.Time) ([]int, bool, error) {
		fetched = append(fetched, [2]int64{offset(from), offset(to)})
		var orders []int
		for ms := offset(from); ms <= offset(to) && ms < 6; ms++ {
			orders = append(orders, int(ms))
		}
		return orders, len(orders) > 3, nil
	}
	idOf := func(o *int) string { return strconv.Itoa(*o) }
	orders, err := windowedOrders(start, start.Add(5*time.Millisecond), 0, fetch, idOf)
	require.NoError(t, err, "windowedOrders must not error")
	assert.ElementsMatch(t, []int{0, 1, 2, 3, 4, 5}, orders, "windowedOrders should collect each order once from halved windows")
	assert.Equal(t, [][2]int64{{0, 5}, {0, 2}, {2, 5}, {2, 3}, {3, 5}}, fetched, "windowedOrders should halve each window that was cut short, sharing the boundary millisecond")

	fetched = nil
	_, err = windowedOrders(start, start.Add(25*time.Hour), 10*time.Hour, func(from, to time.Time) ([]int, bool, error) {
		fetched = append(fetched, [2]int64{offset(from), offset(to)})
		return nil, false, nil
	}, idOf)
	require.NoError(t, err, "windowedOrders must not error")
	hour := time.Hour.Milliseconds()
	assert.Equal(t, [][2]int64{{0, 10 * hour}, {10 * hour, 20 * hour}, {20 * hour, 25 * hour}}, fetched, "windowedOrders should split a range into the spans the endpoint takes")

	_, err = windowedOrders(start, start.Add(time.Millisecond), 0, func(time.Time, time.Time) ([]int, bool, error) { return nil, true, nil }, idOf)
	assert.ErrorIs(t, err, errOrderHistoryLimit, "windowedOrders should report a millisecond holding more orders than the endpoint returns")
	_, err = windowedOrders(start, start.Add(time.Hour), 0, func(time.Time, time.Time) ([]int, bool, error) { return nil, false, errAPIResponse }, idOf)
	assert.ErrorIs(t, err, errAPIResponse, "windowedOrders should return a window's error")
	_, err = windowedOrders(start, start.Add(25*time.Hour), 10*time.Hour, func(time.Time, time.Time) ([]int, bool, error) { return nil, false, errAPIResponse }, idOf)
	assert.ErrorIs(t, err, errAPIResponse, "windowedOrders should return a span's error")
	var calls int
	_, err = windowedOrders(start, start.Add(time.Hour), 0, func(time.Time, time.Time) ([]int, bool, error) {
		calls++
		if calls == 1 {
			return nil, true, nil
		}
		return nil, false, errAPIResponse
	}, idOf)
	assert.ErrorIs(t, err, errAPIResponse, "windowedOrders should return a halved window's error")
	calls = 0
	_, err = windowedOrders(start, start.Add(time.Hour), 0, func(time.Time, time.Time) ([]int, bool, error) {
		calls++
		if calls < 3 {
			return nil, calls == 1, nil
		}
		return nil, false, errAPIResponse
	}, idOf)
	assert.ErrorIs(t, err, errAPIResponse, "windowedOrders should return the second half's error")
}

func TestCursorOrders(t *testing.T) {
	t.Parallel()
	idOf := func(o *uint64) uint64 { return *o }
	ids := func(first, last uint64) []uint64 {
		var held []uint64
		for id := first; id <= last; id++ {
			held = append(held, id)
		}
		return held
	}
	for _, tc := range []struct {
		name      string
		inclusive bool
		limit     int
		held      []uint64
		from      uint64
		exp       []uint64
		cursors   []uint64
	}{
		{name: "inclusive cursor", inclusive: true, limit: 3, held: ids(10, 17), from: 10, exp: ids(10, 17), cursors: []uint64{10, 12, 14, 16}},
		{name: "exclusive cursor", limit: 3, held: ids(10, 17), from: 10, exp: ids(11, 17), cursors: []uint64{10, 13, 16}},
		{name: "single remaining order", inclusive: true, limit: 3, held: ids(10, 15), from: 10, exp: ids(10, 15), cursors: []uint64{10, 12, 14}},
		{name: "whole pages", limit: 2, held: ids(1, 5), from: 1, exp: ids(2, 5), cursors: []uint64{1, 3, 5}},
		{name: "no orders", inclusive: true, limit: 3, held: ids(10, 17), from: 20, cursors: []uint64{20}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var cursors []uint64
			orders, err := cursorOrders(tc.from, tc.limit, func(cursor uint64) ([]uint64, error) {
				cursors = append(cursors, cursor)
				require.LessOrEqual(t, len(cursors), 10, "cursorOrders must stop paging")
				var page []uint64
				for _, id := range tc.held {
					if (id > cursor || tc.inclusive && id == cursor) && len(page) < tc.limit {
						page = append(page, id)
					}
				}
				return page, nil
			}, idOf)
			require.NoError(t, err, "cursorOrders must not error")
			assert.Equal(t, tc.exp, orders, "cursorOrders should collect every order from the cursor once, in the order fetched")
			assert.Equal(t, tc.cursors, cursors, "cursorOrders should continue from the highest order ID until a page comes back short")
		})
	}

	t.Run("page leaving the cursor in place", func(t *testing.T) {
		t.Parallel()
		var requests int
		orders, err := cursorOrders(12, 3, func(cursor uint64) ([]uint64, error) {
			requests++
			require.LessOrEqual(t, requests, 10, "cursorOrders must stop paging")
			return []uint64{cursor, cursor - 1, cursor - 2}, nil
		}, idOf)
		require.NoError(t, err, "cursorOrders must not error")
		assert.Equal(t, []uint64{12, 11, 10}, orders, "cursorOrders should keep a page's orders before stopping on its lack of progress")
		assert.Equal(t, 1, requests, "cursorOrders should stop at a page that leaves the cursor where it was")
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		var requests int
		orders, err := cursorOrders(1, 2, func(cursor uint64) ([]uint64, error) {
			requests++
			if requests == 2 {
				return nil, errAPIResponse
			}
			return []uint64{cursor + 1, cursor + 2}, nil
		}, idOf)
		assert.ErrorIs(t, err, errAPIResponse, "cursorOrders should return a page's error")
		assert.Nil(t, orders, "cursorOrders should return no partial history with an error")
	})
}

func TestEarlierTime(t *testing.T) {
	t.Parallel()
	a, b := time.UnixMilli(1), time.UnixMilli(2)
	assert.Equal(t, a, earlierTime(a, b), "earlierTime should return the first time when it is earlier")
	assert.Equal(t, a, earlierTime(b, a), "earlierTime should return the second time when it is earlier")
}

// wrapperSpotHistoryPage returns a spot order history reply of n orders, numbered from first
func wrapperSpotHistoryPage(first, n int) string {
	return `{"code":0,"data":{"orders":` + wrapperRecordsPage(first, n, func(i int) string {
		return `{"symbol":"BTC-USDT","orderId":` + strconv.Itoa(i+1) + `,"status":"FILLED","type":"LIMIT","side":"BUY"}`
	}) + `}}`
}

func TestSpotOrderPages(t *testing.T) {
	t.Parallel()
	ex, requests := wrapperTestServer(t, map[string][]string{
		"GET /openApi/spot/v1/trade/historyOrders": {wrapperSpotHistoryPage(0, spotOrderHistoryPageSize), wrapperSpotHistoryPage(spotOrderHistoryPageSize, 5), `{"code":100500,"msg":"busy"}`},
	})
	orders, cutShort, err := ex.spotOrderPages(t.Context(), &SpotOrderHistoryRequest{Symbol: currency.NewBTCUSDT()})
	require.NoError(t, err, "spotOrderPages must not error")
	assert.False(t, cutShort, "spotOrderPages should report a query whose last page was not full as whole")
	assert.Len(t, orders, spotOrderHistoryPageSize+5, "spotOrderPages should return every page's orders")
	got := requests()
	require.Len(t, got, 2, "spotOrderPages must request pages until one is not full")
	assert.Equal(t, url.Values{"symbol": {"BTC-USDT"}, "pageIndex": {"2"}, "pageSize": {"100"}}, got[1].params, "spotOrderPages should request the next page")
	_, _, err = ex.spotOrderPages(t.Context(), &SpotOrderHistoryRequest{})
	assert.ErrorIs(t, err, errAPIResponse, "spotOrderPages should return a page's error")

	t.Run("furthest page", func(t *testing.T) {
		t.Parallel()
		pages := make([]string, spotOrderHistoryPagesPerSpan)
		for i := range pages {
			pages[i] = wrapperSpotHistoryPage(i*spotOrderHistoryPageSize, spotOrderHistoryPageSize)
		}
		ex, requests := wrapperTestServer(t, map[string][]string{"GET /openApi/spot/v1/trade/historyOrders": pages})
		orders, cutShort, err := ex.spotOrderPages(t.Context(), &SpotOrderHistoryRequest{})
		require.NoError(t, err, "spotOrderPages must not error")
		assert.True(t, cutShort, "spotOrderPages should report a full furthest page as cut short")
		assert.Len(t, orders, spotOrderHistoryMaxPageWindow, "spotOrderPages should return every page's orders")
		assert.Len(t, requests(), spotOrderHistoryPagesPerSpan, "spotOrderPages should stop at the furthest page BingX serves")
	})
}

func TestSpotOrderHistory(t *testing.T) {
	t.Parallel()
	ex, requests := wrapperTestServer(t, map[string][]string{
		"GET /openApi/spot/v1/trade/historyOrders": {wrapperSpotHistoryPage(0, 1), wrapperSpotHistoryPage(1, 1), `{"code":100500,"msg":"busy"}`},
	})
	start := time.UnixMilli(1700000000000)
	orders, err := ex.spotOrderHistory(t.Context(), &order.MultiOrderRequest{StartTime: start, EndTime: start.Add(time.Hour)}, currency.EMPTYPAIR, 0)
	require.NoError(t, err, "spotOrderHistory must not error")
	require.Len(t, orders, 1, "spotOrderHistory must return the order")
	assert.Equal(t, "BTC-USDT", orders[0].Pair.String(), "spotOrderHistory should take the pair from the symbol")
	assert.Equal(t, url.Values{"startTime": {"1700000000000"}, "endTime": {"1700003600000"}, "pageIndex": {"1"}, "pageSize": {"100"}}, requests()[0].params, "spotOrderHistory should request the window")
	_, err = ex.spotOrderHistory(t.Context(), &order.MultiOrderRequest{StartTime: start, EndTime: start.Add(time.Hour)}, currency.NewBTCUSDT(), 7)
	require.NoError(t, err, "spotOrderHistory must not error")
	assert.Equal(t, url.Values{"symbol": {"BTC-USDT"}, "orderId": {"7"}, "pageIndex": {"1"}, "pageSize": {"100"}}, requests()[1].params, "spotOrderHistory should page from the order ID without the window, which the filter applies")
	_, err = ex.spotOrderHistory(t.Context(), &order.MultiOrderRequest{}, currency.NewBTCUSDT(), 0)
	assert.ErrorIs(t, err, errAPIResponse, "spotOrderHistory should return the request's error")

	t.Run("furthest page", func(t *testing.T) {
		t.Parallel()
		pages := make([]string, 0, 2*spotOrderHistoryPagesPerSpan)
		for range 2 {
			for i := range spotOrderHistoryPagesPerSpan {
				pages = append(pages, wrapperSpotHistoryPage(i*spotOrderHistoryPageSize, spotOrderHistoryPageSize))
			}
		}
		ex, requests := wrapperTestServer(t, map[string][]string{"GET /openApi/spot/v1/trade/historyOrders": pages})
		orders, err := ex.spotOrderHistory(t.Context(), &order.MultiOrderRequest{}, currency.NewBTCUSDT(), 0)
		require.NoError(t, err, "spotOrderHistory must not error")
		assert.Len(t, orders, spotOrderHistoryMaxPageWindow, "spotOrderHistory should return BingX's most recent orders without an order ID")
		_, err = ex.spotOrderHistory(t.Context(), &order.MultiOrderRequest{}, currency.NewBTCUSDT(), 7)
		assert.ErrorIs(t, err, errOrderHistoryTruncated, "spotOrderHistory should report orders from an order ID beyond the furthest page as truncated")
		assert.Len(t, requests(), 2*spotOrderHistoryPagesPerSpan, "spotOrderHistory should stop at the furthest page BingX serves")
	})
}

func TestSwapSettlementCurrencies(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []currency.Code{currency.USDC}, swapSettlementCurrencies(currency.NewPair(currency.BTC, currency.USDC)), "swapSettlementCurrencies should return a pair's quote")
	assert.Equal(t, []currency.Code{currency.USDT, currency.USDC}, swapSettlementCurrencies(currency.EMPTYPAIR), "swapSettlementCurrencies should cover both settlement currencies for every symbol")
}

// wrapperPerpetualHistoryPage returns a perpetual order history reply of n orders, numbered from first
func wrapperPerpetualHistoryPage(first, n int, symbol string) string {
	return `{"code":0,"data":{"orders":` + wrapperRecordsPage(first, n, func(i int) string {
		return `{"symbol":"` + symbol + `","orderId":` + strconv.Itoa(i+1) + `,"status":"FILLED","type":"LIMIT","side":"BUY"}`
	}) + `}}`
}

func TestSwapOrderHistory(t *testing.T) {
	t.Parallel()
	ex, requests := wrapperTestServer(t, map[string][]string{
		"GET /openApi/swap/v2/trade/allOrders": {
			wrapperPerpetualHistoryPage(0, 1, "BTC-USDT"),
			wrapperPerpetualHistoryPage(1, 1, "BTC-USDC"),
			wrapperPerpetualHistoryPage(0, perpetualOrderHistoryLimit, "BTC-USDT"),
			wrapperPerpetualHistoryPage(0, 2, "BTC-USDT"),
			wrapperPerpetualHistoryPage(2, 2, "BTC-USDT"),
			wrapperPerpetualHistoryPage(4, 1, "BTC-USDT"),
			`{"code":100500,"msg":"busy"}`,
		},
	})
	orders, err := ex.swapOrderHistory(t.Context(), &order.MultiOrderRequest{}, currency.EMPTYPAIR, 9)
	require.NoError(t, err, "swapOrderHistory must not error")
	require.Len(t, orders, 2, "swapOrderHistory must return both settlement currencies' orders")
	assert.Equal(t, "BTC-USDC", orders[1].Pair.String(), "swapOrderHistory should take the pair from the symbol")
	got := requests()
	assert.Equal(t, url.Values{"currency": {"USDT"}, "orderId": {"9"}, "limit": {"1000"}}, got[0].params, "swapOrderHistory should list USDT-settled orders from the order ID")
	assert.Equal(t, url.Values{"currency": {"USDC"}, "orderId": {"9"}, "limit": {"1000"}}, got[1].params, "swapOrderHistory should list USDC-settled orders from the order ID")

	start := time.UnixMilli(1700000000000)
	end := start.Add(perpetualOrderHistorySpan + time.Hour)
	orders, err = ex.swapOrderHistory(t.Context(), &order.MultiOrderRequest{StartTime: start, EndTime: end}, currency.NewBTCUSDT(), 0)
	require.NoError(t, err, "swapOrderHistory must not error")
	assert.Len(t, orders, 5, "swapOrderHistory should collect each order once from every window")
	got = requests()[2:]
	require.Len(t, got, 4, "swapOrderHistory must request each span and halve a full one")
	middle := start.Add(perpetualOrderHistorySpan / 2).Truncate(time.Millisecond)
	assert.Equal(t, url.Values{"symbol": {"BTC-USDT"}, "currency": {"USDT"}, "startTime": {wrapperMilliseconds(start)}, "endTime": {wrapperMilliseconds(start.Add(perpetualOrderHistorySpan))}, "limit": {"1000"}}, got[0].params, "swapOrderHistory should request the first span")
	assert.Equal(t, wrapperMilliseconds(middle), got[1].params.Get("endTime"), "swapOrderHistory should halve a span holding the most orders a request returns")
	assert.Equal(t, wrapperMilliseconds(end), got[3].params.Get("endTime"), "swapOrderHistory should end the last span at the range's end")
	_, err = ex.swapOrderHistory(t.Context(), &order.MultiOrderRequest{}, currency.NewBTCUSDT(), 0)
	assert.ErrorIs(t, err, errAPIResponse, "swapOrderHistory should return the request's error")

	t.Run("from an order ID", func(t *testing.T) {
		t.Parallel()
		// Orders 9 to 1008 fill the first page, and the next, from 1008, repeats it before the last order, 1009
		ex, requests := wrapperTestServer(t, map[string][]string{
			"GET /openApi/swap/v2/trade/allOrders": {
				wrapperPerpetualHistoryPage(8, perpetualOrderHistoryLimit, "BTC-USDT"),
				wrapperPerpetualHistoryPage(1007, 2, "BTC-USDT"),
				wrapperPerpetualHistoryPage(2000, 1, "BTC-USDC"),
				`{"code":100500,"msg":"busy"}`,
			},
		})
		orders, err := ex.swapOrderHistory(t.Context(), &order.MultiOrderRequest{}, currency.EMPTYPAIR, 9)
		require.NoError(t, err, "swapOrderHistory must not error")
		orderIDs := make([]string, len(orders))
		for i := range orders {
			orderIDs[i] = orders[i].OrderID
		}
		exp := make([]string, 0, perpetualOrderHistoryLimit+2)
		for id := 9; id <= 1009; id++ {
			exp = append(exp, strconv.Itoa(id))
		}
		assert.Equal(t, append(exp, "2001"), orderIDs, "swapOrderHistory should collect every order from the order ID once")
		got := requests()
		require.Len(t, got, 3, "swapOrderHistory must page each settlement currency until a page comes back short")
		assert.Equal(t, url.Values{"currency": {"USDT"}, "orderId": {"1008"}, "limit": {"1000"}}, got[1].params, "swapOrderHistory should continue from the highest order ID returned")
		assert.Equal(t, url.Values{"currency": {"USDC"}, "orderId": {"9"}, "limit": {"1000"}}, got[2].params, "swapOrderHistory should page USDC-settled orders from the order ID")
		_, err = ex.swapOrderHistory(t.Context(), &order.MultiOrderRequest{}, currency.NewBTCUSDT(), 9)
		assert.ErrorIs(t, err, errAPIResponse, "swapOrderHistory should return a page's error")
	})
}

func TestCoinMarginedOrderHistory(t *testing.T) {
	t.Parallel()
	ex, requests := wrapperTestServer(t, map[string][]string{
		"GET /openApi/cswap/v1/trade/orderHistory": {wrapperPerpetualHistoryPage(0, 1, "ETH-USD"), wrapperPerpetualHistoryPage(1, 2, "BTC-USD"), `{"code":100500,"msg":"busy"}`},
	})
	orders, err := ex.coinMarginedOrderHistory(t.Context(), &order.MultiOrderRequest{}, currency.EMPTYPAIR, 9)
	require.NoError(t, err, "coinMarginedOrderHistory must not error")
	require.Len(t, orders, 1, "coinMarginedOrderHistory must return the order")
	assert.Equal(t, "ETH-USD", orders[0].Pair.String(), "coinMarginedOrderHistory should take the pair from the symbol")
	assert.Equal(t, url.Values{"orderId": {"9"}, "limit": {"1000"}}, requests()[0].params, "coinMarginedOrderHistory should list every symbol's orders from the order ID")
	start := time.UnixMilli(1700000000000)
	pair := currency.NewPairWithDelimiter("BTC", "USD", currency.DashDelimiter)
	orders, err = ex.coinMarginedOrderHistory(t.Context(), &order.MultiOrderRequest{StartTime: start, EndTime: start.Add(time.Hour)}, pair, 0)
	require.NoError(t, err, "coinMarginedOrderHistory must not error")
	assert.Len(t, orders, 2, "coinMarginedOrderHistory should return the window's orders")
	assert.Equal(t, url.Values{"symbol": {"BTC-USD"}, "startTime": {"1700000000000"}, "endTime": {"1700003600000"}, "limit": {"1000"}}, requests()[1].params, "coinMarginedOrderHistory should request the window")
	_, err = ex.coinMarginedOrderHistory(t.Context(), &order.MultiOrderRequest{}, pair, 0)
	assert.ErrorIs(t, err, errAPIResponse, "coinMarginedOrderHistory should return the request's error")

	t.Run("from an order ID", func(t *testing.T) {
		t.Parallel()
		ex, requests := wrapperTestServer(t, map[string][]string{
			"GET /openApi/cswap/v1/trade/orderHistory": {
				wrapperPerpetualHistoryPage(8, perpetualOrderHistoryLimit, "BTC-USD"),
				wrapperPerpetualHistoryPage(1007, 2, "BTC-USD"),
				`{"code":100500,"msg":"busy"}`,
			},
		})
		orders, err := ex.coinMarginedOrderHistory(t.Context(), &order.MultiOrderRequest{}, pair, 9)
		require.NoError(t, err, "coinMarginedOrderHistory must not error")
		orderIDs := make([]string, len(orders))
		for i := range orders {
			orderIDs[i] = orders[i].OrderID
		}
		exp := make([]string, 0, perpetualOrderHistoryLimit+1)
		for id := 9; id <= 1009; id++ {
			exp = append(exp, strconv.Itoa(id))
		}
		assert.Equal(t, exp, orderIDs, "coinMarginedOrderHistory should collect every order from the order ID once")
		got := requests()
		require.Len(t, got, 2, "coinMarginedOrderHistory must page until a page comes back short")
		assert.Equal(t, url.Values{"symbol": {"BTC-USD"}, "orderId": {"1008"}, "limit": {"1000"}}, got[1].params, "coinMarginedOrderHistory should continue from the highest order ID returned")
		_, err = ex.coinMarginedOrderHistory(t.Context(), &order.MultiOrderRequest{}, pair, 9)
		assert.ErrorIs(t, err, errAPIResponse, "coinMarginedOrderHistory should return a page's error")
	})
}
