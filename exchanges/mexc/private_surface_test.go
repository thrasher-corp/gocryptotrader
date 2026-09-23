package mexc

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// newPrivateTestExchange builds an isolated Exchange whose spot REST endpoint points at a local
// httptest server returning recorded (fixture) MEXC responses. It never reaches a live private
// endpoint: it exercises the private response mapping without live keys. The credentials are dummy
// and the server ignores the signature. It exists because the shipped auth tests skip under mock
// without keys, so the private response-mapping was never exercised - which is how these defects
// reached review.
func newPrivateTestExchange(t *testing.T, handler http.HandlerFunc) *Exchange {
	t.Helper()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.SetCredentials(&accounts.Credentials{Key: "test-key", Secret: "test-secret"})
	// The limiters are shared by every instance and the server is local, so a test exchange does not
	// draw on the budget the other tests wait on.
	require.NoError(t, ex.Requester.DisableRateLimiter(), "DisableRateLimiter must not error")
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	b := ex.GetBase()
	b.SkipAuthCheck = true
	require.NoError(t, b.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), srv.URL), "SetRunningURL must not error")
	require.NoError(t, ex.setEnabledPairs(spotTradablePair), "setEnabledPairs must not error")
	return ex
}

// liveAccess states what a test needs to run against the venue when built with -tags mock_test_off
type liveAccess uint8

const (
	// livePublic tests call public endpoints only and run live without credentials
	livePublic liveAccess = iota
	// liveReadOnly tests read account data and run live once credentials are set
	liveReadOnly
	// liveManipulatesOrders tests place, cancel or withdraw and also need canManipulateRealOrders
	liveManipulatesOrders
)

// testExchangeFor returns the exchange a private test runs against. In mock mode it is an isolated
// instance serving the handler's recorded responses, so the test can assert the exact payload and
// values. Built with -tags mock_test_off it is the live instance, skipped unless the access the test
// needs is configured; a live test asserts the shape of the venue's response rather than the recorded
// values.
func testExchangeFor(t *testing.T, handler http.HandlerFunc, access liveAccess) *Exchange {
	t.Helper()
	if mockTests {
		return newPrivateTestExchange(t, handler)
	}
	switch access {
	case liveReadOnly:
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	case liveManipulatesOrders:
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	return e
}

// jsonHandler routes on the request path suffix and writes the matching recorded body.
func jsonHandler(t *testing.T, bySuffix map[string]string) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		for suffix, body := range bySuffix {
			if strings.HasSuffix(r.URL.Path, suffix) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(body))
				return
			}
		}
		t.Errorf("unexpected request path %q", r.URL.Path)
		http.Error(w, "unexpected path", http.StatusNotFound)
	}
}

// TestGetAccountFundingHistoryDepositTimestamp stamps the deposit at insertTime, not confirmTimes.
// confirmTimes is a confirmation counter ("241"): decoding it as a timestamp failed to parse or
// stamped the record at the zero time.
func TestGetAccountFundingHistoryDepositTimestamp(t *testing.T) {
	t.Parallel()
	const insertTime = 1704067200000 // 2024-01-01T00:00:00Z
	ex := testExchangeFor(t, jsonHandler(t, map[string]string{
		"capital/deposit/hisrec": `[{"amount":"1.5","coin":"USDT","network":"TRC20","status":5,` +
			`"address":"addr","txId":"txhash","confirmTimes":"241","insertTime":1704067200000}]`,
		"capital/withdraw/history": `[]`,
	}), liveReadOnly)
	result, err := ex.GetAccountFundingHistory(t.Context())
	require.NoError(t, err, "GetAccountFundingHistory must not error on a counter-valued confirmTimes")
	if !mockTests {
		for i := range result {
			assert.Falsef(t, result[i].Timestamp.IsZero(), "record %d should carry a timestamp", i)
			assert.NotEmptyf(t, result[i].Currency, "record %d should carry a currency", i)
		}
		return
	}
	require.Len(t, result, 1, "the single deposit must be relayed")
	assert.Equal(t, int64(insertTime), result[0].Timestamp.UnixMilli(), "Timestamp should come from insertTime")
	assert.Equal(t, "txhash", result[0].TransferID, "TransferID should be the txId")
}

// TestGetOrderHistoryPairAndTimestamps fills the pair and the order's real timestamps, and parses the
// MEXC-specific IMMEDIATE_OR_CANCEL type instead of failing the whole query.
func TestGetOrderHistoryPairAndTimestamps(t *testing.T) {
	t.Parallel()
	const (
		created = 1704067200000
		updated = 1704067260000
	)
	ex := testExchangeFor(t, jsonHandler(t, map[string]string{
		"allOrders": `[{"symbol":"BTCUSDT","orderId":"111","price":"50000","origQty":"0.5",` +
			`"executedQty":"0.2","cummulativeQuoteQty":"10000","type":"IMMEDIATE_OR_CANCEL",` +
			`"side":"BUY","status":"PARTIALLY_FILLED","time":1704067200000,"updateTime":1704067260000}]`,
	}), liveReadOnly)
	orders, err := ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
		AssetType: asset.Spot,
		Pairs:     currency.Pairs{spotTradablePair},
		Side:      order.AnySide,
		Type:      order.AnyType,
	})
	require.NoError(t, err, "GetOrderHistory must not error on an IMMEDIATE_OR_CANCEL order")
	if !mockTests {
		for i := range orders {
			assert.Truef(t, orders[i].Pair.Equal(spotTradablePair), "order %s should carry the requested pair", orders[i].OrderID)
			assert.Falsef(t, orders[i].Date.IsZero(), "order %s should carry its creation time", orders[i].OrderID)
		}
		return
	}
	require.Len(t, orders, 1, "the single order must be relayed")
	assert.Equal(t, spotTradablePair, orders[0].Pair, "the pair should be filled in")
	assert.False(t, orders[0].Date.IsZero(), "the creation time should be set")
	assert.Equal(t, int64(created), orders[0].Date.UnixMilli(), "Date should come from time")
	assert.Equal(t, int64(updated), orders[0].LastUpdated.UnixMilli(), "LastUpdated should come from updateTime")
	assert.Equal(t, order.ImmediateOrCancel, orders[0].TimeInForce, "the IOC time-in-force should be preserved")
}

// TestUpdateAccountBalancesArithmetic reports free/locked as Total=free+locked, Hold=locked,
// Free=free. Free was left unset, so available balance read as zero.
func TestUpdateAccountBalancesArithmetic(t *testing.T) {
	t.Parallel()
	ex := testExchangeFor(t, jsonHandler(t, map[string]string{
		"account": `{"accountType":"SPOT","canTrade":true,"balances":[{"asset":"USDT","free":"10","locked":"3"}]}`,
	}), liveReadOnly)
	subAccounts, err := ex.UpdateAccountBalances(t.Context(), asset.Spot)
	require.NoError(t, err, "UpdateAccountBalances must not error")
	require.Len(t, subAccounts, 1, "one sub-account must be returned")
	if !mockTests {
		for code, bal := range subAccounts[0].Balances {
			assert.InDeltaf(t, bal.Free+bal.Hold, bal.Total, 1e-9, "%s Total should be free + locked", code)
		}
		return
	}
	bal := subAccounts[0].Balances[currency.USDT]
	assert.Equal(t, 13.0, bal.Total, "Total should be free + locked")
	assert.Equal(t, 3.0, bal.Hold, "Hold should be locked")
	assert.Equal(t, 10.0, bal.Free, "Free should be the available (free) balance")
}

// TestCancelOrderFormatsSymbol sends the delimiter-free symbol the exchange expects: a cancel for a pair
// carrying a delimiter must not put that delimiter on the wire.
func TestCancelOrderFormatsSymbol(t *testing.T) {
	t.Parallel()
	var sentSymbol string
	ex := testExchangeFor(t, func(w http.ResponseWriter, r *http.Request) {
		sentSymbol = r.URL.Query().Get("symbol")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"symbol":"BTCUSDT","orderId":"111"}`))
	}, liveManipulatesOrders)
	err := ex.CancelOrder(t.Context(), &order.Cancel{
		OrderID:   "111",
		AssetType: asset.Spot,
		Pair:      currency.NewPairWithDelimiter("BTC", "USDT", "-"),
	})
	if !mockTests {
		// No such order exists, so the venue refuses the cancel; it must refuse the order, not the symbol.
		if err != nil {
			assert.NotContains(t, err.Error(), "Invalid symbol", "the venue should accept the symbol the cancel sends")
		}
		return
	}
	require.NoError(t, err, "CancelOrder must not error")
	assert.Equal(t, "BTCUSDT", sentSymbol, "the cancel should send the delimiter-free symbol")
}

// TestCancelAllOrdersNoOrderID is a symbol-wide cancel: it must not require an order id
// (StandardCancel is not used on the symbol-wide path).
func TestCancelAllOrdersNoOrderID(t *testing.T) {
	t.Parallel()
	ex := testExchangeFor(t, jsonHandler(t, map[string]string{
		"openOrders": `[]`,
	}), liveManipulatesOrders)
	_, err := ex.CancelAllOrders(t.Context(), &order.Cancel{
		AssetType: asset.Spot,
		Pair:      spotTradablePair,
	})
	require.NotErrorIs(t, err, order.ErrOrderIDNotSet, "a symbol-wide cancel must not demand an order id")
	require.NoError(t, err, "the symbol-wide cancel must proceed")
}

// TestGetFeeByTypeReturnsAmount returns the absolute fee (rate * price * quantity), not the bare
// rate.
func TestGetFeeByTypeReturnsAmount(t *testing.T) {
	t.Parallel()
	ex := testExchangeFor(t, jsonHandler(t, map[string]string{
		"tradeFee": `{"code":0,"data":{"makerCommission":0.001,"takerCommission":0.002}}`,
	}), liveReadOnly)
	taker, err := ex.GetFeeByType(t.Context(), &exchange.FeeBuilder{
		FeeType:       exchange.CryptocurrencyTradeFee,
		Pair:          spotTradablePair,
		PurchasePrice: 50000,
		Amount:        0.5,
	})
	require.NoError(t, err, "GetFeeByType must not error")
	if !mockTests {
		// The account's rate is unknown here, but the fee on a 25000 quote order is at most a few percent of it.
		assert.GreaterOrEqual(t, taker, 0.0, "the fee should not be negative")
		assert.Less(t, taker, 25000*0.05, "the fee should be an amount on the order, not a rate scaled wrongly")
		return
	}
	assert.InDelta(t, 50.0, taker, 1e-9, "taker fee should be rate * price * quantity")
}

// TestGetFeeByTypeFormatsPair pins the symbol the authenticated fee request puts on the wire: MEXC
// rejects the delimited config-format pair a caller gets from GetEnabledPairs.
func TestGetFeeByTypeFormatsPair(t *testing.T) {
	t.Parallel()
	var got url.Values
	ex := testExchangeFor(t, func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"data":{"makerCommission":0.001,"takerCommission":0.002}}`))
	}, liveReadOnly)
	_, err := ex.GetFeeByType(t.Context(), &exchange.FeeBuilder{
		FeeType:       exchange.CryptocurrencyTradeFee,
		Pair:          currency.NewPairWithDelimiter("BTC", "USDT", currency.DashDelimiter),
		PurchasePrice: 50000,
		Amount:        0.5,
	})
	// Live, the venue answers a config-format symbol with an error, so no error is the assertion there.
	require.NoError(t, err, "GetFeeByType must not error")
	if !mockTests {
		return
	}
	assert.Equal(t, "BTCUSDT", got.Get("symbol"), "the fee request should send the exchange-format symbol, not the config-format pair")
}

// TestOrderTypeStringPostOnlyAndTIF maps a limit order's time-in-force into MEXC's order type field:
// post-only is LIMIT_MAKER and a limit IOC/FOK must not degrade to a plain LIMIT.
func TestOrderTypeStringPostOnlyAndTIF(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		typ  order.Type
		tif  order.TimeInForce
		want string
	}{
		{"limit post-only is LIMIT_MAKER", order.Limit, order.PostOnly, typeLimitMaker},
		{"limit IOC is preserved", order.Limit, order.ImmediateOrCancel, typeImmediateOrCancel},
		{"limit FOK is preserved", order.Limit, order.FillOrKill, typeFillOrKill},
		{"plain limit stays LIMIT", order.Limit, order.GoodTillCancel, typeLimit},
		{"market IOC is a plain MARKET", order.Market, order.ImmediateOrCancel, typeMarket},
		{"plain market stays MARKET", order.Market, order.UnknownTIF, typeMarket},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := e.OrderTypeStringFromOrderTypeAndTimeInForce(tc.typ, tc.tif)
			require.NoError(t, err, "mapping must not error")
			assert.Equal(t, tc.want, got, "order type string mismatch")
		})
	}
}

// TestStringToOrderTypeAndTimeInForceIOC asserts the reverse mapping recognises MEXC's own order
// types (the generic parser rejects them).
func TestStringToOrderTypeAndTimeInForceIOC(t *testing.T) {
	t.Parallel()
	oType, tif, err := e.StringToOrderTypeAndTimeInForce(typeImmediateOrCancel)
	require.NoError(t, err, "IMMEDIATE_OR_CANCEL must be recognised")
	assert.Equal(t, order.ImmediateOrCancel, tif, "time-in-force should be IOC")
	assert.Equal(t, order.Limit, oType, "IMMEDIATE_OR_CANCEL is a limit order type on MEXC")
}

// TestActiveOrdersLastUpdatedFallback covers the LastUpdated timestamp for open orders. MEXC returns
// updateTime:null on an open (still-working) order; without a fallback LastUpdated was stamped at the
// zero time. The fallback uses the creation time (time) when updateTime is empty, and keeps
// updateTime when it is present.
func TestActiveOrdersLastUpdatedFallback(t *testing.T) {
	t.Parallel()
	const (
		created = 1704067200000 // 2024-01-01T00:00:00Z
		updated = 1704067260000 // 2024-01-01T00:01:00Z
	)
	ex := testExchangeFor(t, jsonHandler(t, map[string]string{
		// First order is open: updateTime is null. Second order carries a real updateTime.
		"openOrders": `[{"symbol":"BTCUSDT","orderId":"111","price":"50000","origQty":"0.5",` +
			`"executedQty":"0","cummulativeQuoteQty":"0","type":"LIMIT","side":"BUY",` +
			`"status":"NEW","time":1704067200000,"updateTime":null},` +
			`{"symbol":"BTCUSDT","orderId":"222","price":"50000","origQty":"0.5",` +
			`"executedQty":"0.2","cummulativeQuoteQty":"10000","type":"LIMIT","side":"BUY",` +
			`"status":"PARTIALLY_FILLED","time":1704067200000,"updateTime":1704067260000}]`,
	}), liveReadOnly)
	orders, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{
		AssetType: asset.Spot,
		Pairs:     currency.Pairs{spotTradablePair},
		Side:      order.AnySide,
		Type:      order.AnyType,
	})
	require.NoError(t, err, "GetActiveOrders must not error")
	if !mockTests {
		for i := range orders {
			assert.Falsef(t, orders[i].LastUpdated.IsZero(), "order %s should carry a last updated time", orders[i].OrderID)
		}
		return
	}
	require.Len(t, orders, 2, "both open orders must be relayed")

	// Fallback case: updateTime was null, so LastUpdated must fall back to the creation time.
	assert.False(t, orders[0].LastUpdated.IsZero(), "LastUpdated should not be the zero time when updateTime is null")
	assert.Equal(t, int64(created), orders[0].LastUpdated.UnixMilli(), "LastUpdated should fall back to time (creation) when updateTime is empty")

	// Positive case: a real updateTime must still be used verbatim (existing behaviour preserved).
	assert.Equal(t, int64(updated), orders[1].LastUpdated.UnixMilli(), "LastUpdated should come from updateTime when it is present")
}

// TestOrderTypeStringMarketTimeInForce covers the forward mapping of a market order's time-in-force. A
// market order never rests, so IOC (and no time-in-force) is a plain MARKET; all-or-nothing has no
// market equivalent on MEXC, so a market FOK is rejected rather than silently sent.
func TestOrderTypeStringMarketTimeInForce(t *testing.T) {
	t.Parallel()
	got, err := e.OrderTypeStringFromOrderTypeAndTimeInForce(order.Market, order.ImmediateOrCancel)
	require.NoError(t, err, "a market IOC order must map without error")
	assert.Equal(t, typeMarket, got, "a market IOC order should be sent as a plain MARKET")

	got, err = e.OrderTypeStringFromOrderTypeAndTimeInForce(order.Market, order.UnknownTIF)
	require.NoError(t, err, "a plain market order must map without error")
	assert.Equal(t, typeMarket, got, "a market order without a time-in-force should be a plain MARKET")

	_, err = e.OrderTypeStringFromOrderTypeAndTimeInForce(order.Market, order.FillOrKill)
	require.ErrorIs(t, err, order.ErrUnsupportedTimeInForce, "a market FOK order has no MEXC equivalent and must be rejected")
}

// TestOrderTypeStringHonoursCombinedTimeInForce keeps a post-only flag carried with GTC, and rejects a
// constraint MEXC cannot express instead of sending a plain LIMIT, LIMIT_MAKER or MARKET without it.
func TestOrderTypeStringHonoursCombinedTimeInForce(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		typ  order.Type
		tif  order.TimeInForce
		want string
		err  error
	}{
		{order.Limit, order.GoodTillCancel | order.PostOnly, typeLimitMaker, nil},
		{order.Limit, order.GoodTillDay, "", order.ErrUnsupportedTimeInForce},
		{order.Limit, order.GoodTillCrossing, "", order.ErrUnsupportedTimeInForce},
		{order.Limit, order.GoodTillDay | order.PostOnly, "", order.ErrUnsupportedTimeInForce},
		{order.Market, order.PostOnly, "", order.ErrUnsupportedTimeInForce},
	} {
		got, err := e.OrderTypeStringFromOrderTypeAndTimeInForce(tc.typ, tc.tif)
		require.ErrorIsf(t, err, tc.err, "%s with %s must map as expected", tc.typ, tc.tif)
		assert.Equalf(t, tc.want, got, "%s with %s should map to %q", tc.typ, tc.tif, tc.want)
	}
}

// TestGetOrderInfoTriggerPrice asserts a stop order's trigger price (stopPrice) is reported on the
// domain order.
func TestGetOrderInfoTriggerPrice(t *testing.T) {
	t.Parallel()
	ex := testExchangeFor(t, jsonHandler(t, map[string]string{
		"order": `{"symbol":"BTCUSDT","orderId":"1","price":"19000","origQty":"1","executedQty":"0",` +
			`"type":"STOP_LIMIT","side":"SELL","status":"NEW","stopPrice":"18000","time":1704067200000}`,
	}), liveReadOnly)
	orderID := "1"
	if !mockTests {
		history, err := ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
			AssetType: asset.Spot, Pairs: currency.Pairs{spotTradablePair}, Side: order.AnySide, Type: order.AnyType,
		})
		require.NoError(t, err, "GetOrderHistory must not error")
		if len(history) == 0 {
			t.Skipf("the account has no %s order to look up", spotTradablePair)
		}
		orderID = history[0].OrderID
	}
	detail, err := ex.GetOrderInfo(t.Context(), orderID, spotTradablePair, asset.Spot)
	require.NoError(t, err, "GetOrderInfo must not error")
	if !mockTests {
		assert.Equal(t, orderID, detail.OrderID, "the order looked up should be returned")
		assert.True(t, detail.Pair.Equal(spotTradablePair), "the order should carry its pair")
		return
	}
	assert.Equal(t, 18000.0, detail.TriggerPrice, "TriggerPrice should carry the stop price")
	assert.Equal(t, order.StopLimit, detail.Type, "a STOP_LIMIT order should map to StopLimit")
}

// TestGetOrderHistoryMultiPair asserts GetOrderHistory queries every requested pair, not only the
// first, and rejects an empty pair set instead of querying across all symbols.
func TestGetOrderHistoryMultiPair(t *testing.T) {
	t.Parallel()
	ex := testExchangeFor(t, func(w http.ResponseWriter, r *http.Request) {
		symbol := r.URL.Query().Get("symbol")
		// The mock reflects the requested symbol so the order id identifies which pair was queried;
		// it is a test double, not a live response.
		_, _ = w.Write([]byte(`[{"symbol":"` + symbol + `","orderId":"` + symbol + `-1","price":"1","origQty":"1","executedQty":"0","type":"LIMIT","side":"BUY","status":"NEW","time":1704067200000}]`)) //nolint:gosec // test mock reflecting the request
	}, liveReadOnly)
	btc := currency.NewBTCUSDT()
	eth := currency.NewPair(currency.ETH, currency.USDT)

	_, err := ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType})
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty, "an empty pair set must be rejected")

	orders, err := ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Pairs: currency.Pairs{btc, eth}, Side: order.AnySide, Type: order.AnyType})
	require.NoError(t, err, "GetOrderHistory must not error")
	if !mockTests {
		for i := range orders {
			assert.Truef(t, orders[i].Pair.Equal(btc) || orders[i].Pair.Equal(eth), "order %s should belong to a requested pair", orders[i].OrderID)
		}
		return
	}
	require.Len(t, orders, 2, "one order per requested pair must be returned")
	ids := []string{orders[0].OrderID, orders[1].OrderID}
	assert.Contains(t, ids, "BTCUSDT-1", "the BTCUSDT pair should be queried")
	assert.Contains(t, ids, "ETHUSDT-1", "the ETHUSDT pair should be queried, not only the first pair")
}

// TestOrderListingsValidateAndFilter asserts both spot order listings honour the request contract:
// an unvalidatable request is rejected, and the returned orders are the filtered set the caller
// asked for rather than everything the venue returned.
func TestOrderListingsValidateAndFilter(t *testing.T) {
	t.Parallel()
	btc := currency.NewBTCUSDT()
	body := `[` +
		`{"symbol":"BTCUSDT","orderId":"buy-limit","price":"1","origQty":"1","executedQty":"0","type":"LIMIT","side":"BUY","status":"NEW","time":1704067200000},` +
		`{"symbol":"BTCUSDT","orderId":"sell-limit","price":"1","origQty":"1","executedQty":"0","type":"LIMIT","side":"SELL","status":"NEW","time":1704067200000},` +
		`{"symbol":"BTCUSDT","orderId":"buy-market","price":"1","origQty":"1","executedQty":"0","type":"MARKET","side":"BUY","status":"NEW","time":1704067200000}` +
		`]`
	for _, tc := range []struct {
		name string
		call func(*Exchange, *order.MultiOrderRequest) (order.FilteredOrders, error)
	}{
		{"GetActiveOrders", func(e *Exchange, r *order.MultiOrderRequest) (order.FilteredOrders, error) {
			return e.GetActiveOrders(t.Context(), r)
		}},
		{"GetOrderHistory", func(e *Exchange, r *order.MultiOrderRequest) (order.FilteredOrders, error) {
			return e.GetOrderHistory(t.Context(), r)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := testExchangeFor(t, func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(body))
			}, liveReadOnly)
			if mockTests {
				require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{btc}, false), "storing available pairs must not error")
				require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{btc}, true), "storing enabled pairs must not error")
			}

			_, err := tc.call(ex, &order.MultiOrderRequest{AssetType: asset.Spot, Pairs: currency.Pairs{btc}})
			require.ErrorIs(t, err, order.ErrSideIsInvalid, "a request with no side must be rejected")

			orders, err := tc.call(ex, &order.MultiOrderRequest{
				AssetType: asset.Spot, Pairs: currency.Pairs{btc}, Side: order.Buy, Type: order.Limit,
			})
			require.NoError(t, err, "the listing must not error")
			if !mockTests {
				for i := range orders {
					assert.Equalf(t, order.Buy, orders[i].Side, "order %s should be a buy", orders[i].OrderID)
					assert.Equalf(t, order.Limit, orders[i].Type, "order %s should be a limit order", orders[i].OrderID)
				}
				return
			}
			require.Len(t, orders, 1, "only the buy limit order matches the request")
			assert.Equal(t, "buy-limit", orders[0].OrderID, "the returned order should be the one the request asked for")
		})
	}
}

// TestWithdrawCryptocurrencyFunds sends a crypto withdrawal through the current withdraw endpoint,
// which names the network netWork and carries the destination tag as memo.
func TestWithdrawCryptocurrencyFunds(t *testing.T) {
	t.Parallel()
	var got url.Values
	var path string
	ex := testExchangeFor(t, func(w http.ResponseWriter, r *http.Request) {
		path, got = r.URL.Path, r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"7213fea8e94b4a5593d507237e5a555b"}`))
	}, liveManipulatesOrders)
	_, err := ex.WithdrawCryptocurrencyFunds(t.Context(), &withdraw.Request{Exchange: ex.Name, Type: withdraw.Crypto})
	assert.Error(t, err, "an invalid withdrawal request should be rejected before it is sent")
	if !mockTests {
		// A negative amount keeps a live run from moving funds: the request is refused on its amount alone.
		_, err := ex.WithdrawCryptocurrencyFunds(t.Context(), &withdraw.Request{
			Exchange: ex.Name,
			Currency: currency.BTC,
			Amount:   -0.1,
			Type:     withdraw.Crypto,
			Crypto:   withdraw.CryptoRequest{Address: core.BitcoinDonationAddress, Chain: "BTC"},
		})
		assert.ErrorContains(t, err, withdraw.ErrStrAmountMustBeGreaterThanZero, "a negative withdrawal should be refused on its amount")
		return
	}
	resp, err := ex.WithdrawCryptocurrencyFunds(t.Context(), &withdraw.Request{
		Exchange:      ex.Name,
		Currency:      currency.USDT,
		Amount:        12.5,
		Type:          withdraw.Crypto,
		ClientOrderID: "w1",
		Description:   "rent",
		Crypto:        withdraw.CryptoRequest{Address: "TXYZ", AddressTag: "42", Chain: "TRX"},
	})
	require.NoError(t, err, "WithdrawCryptocurrencyFunds must not error")
	assert.Equal(t, "7213fea8e94b4a5593d507237e5a555b", resp.ID, "ID should be the withdrawal id the venue returned")
	assert.True(t, strings.HasSuffix(path, "/capital/withdraw"), "the withdrawal should go to the current withdraw endpoint")
	for field, want := range map[string]string{
		"coin":            "USDT",
		"address":         "TXYZ",
		"amount":          "12.5",
		"netWork":         "TRX",
		"memo":            "42",
		"withdrawOrderId": "w1",
		"remark":          "rent",
	} {
		assert.Equalf(t, want, got.Get(field), "%s should carry the request field", field)
	}
	assert.Empty(t, got.Get("network"), "the legacy network parameter should not be sent")
}

// TestGetAvailableTransferChains lists a currency's withdraw networks by their netWork value, the one
// the withdraw endpoint takes, rather than the display name in network.
func TestGetAvailableTransferChains(t *testing.T) {
	t.Parallel()
	ex := testExchangeFor(t, jsonHandler(t, map[string]string{
		"capital/config/getall": `[{"coin":"USDT","name":"TetherUS","networkList":[` +
			`{"coin":"USDT","network":"Tron(TRC20)","netWork":"TRX"},{"coin":"USDT","network":"Ethereum(ERC20)","netWork":"ETH"}]},` +
			`{"coin":"USDTX","name":"Other","networkList":[{"coin":"USDTX","network":"BNB Smart Chain(BEP20)","netWork":"BSC"}]}]`,
	}), liveReadOnly)
	chains, err := ex.GetAvailableTransferChains(t.Context(), currency.NewCode("usdt"))
	require.NoError(t, err, "GetAvailableTransferChains must not error")
	if mockTests {
		assert.Equal(t, []string{"TRX", "ETH"}, chains, "the chains should be the netWork values of the matching coin only")
	} else {
		assert.NotEmpty(t, chains, "USDT should have withdraw networks")
		for _, chain := range chains {
			assert.NotContains(t, chain, "(", "a chain should be the netWork identifier, not the display name")
		}
	}

	chains, err = ex.GetAvailableTransferChains(t.Context(), currency.NewCode("NOPE"))
	require.NoError(t, err, "GetAvailableTransferChains must not error for an unlisted coin")
	assert.Empty(t, chains, "an unlisted coin should have no chains")

	_, err = ex.GetAvailableTransferChains(t.Context(), currency.EMPTYCODE)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "an empty currency should be rejected")
}

// TestUpdateOrderExecutionLimitsPercentPriceBySide loads the PERCENT_PRICE_BY_SIDE band as ratios to
// the last price: a buy may be priced up to lastPrice*(1+bidMultiplierUp) and a sell down to
// lastPrice*(1-askMultiplierDown).
func TestUpdateOrderExecutionLimitsPercentPriceBySide(t *testing.T) {
	t.Parallel()
	ex := testExchangeFor(t, jsonHandler(t, map[string]string{
		"exchangeInfo": `{"symbols":[{"symbol":"LIMBANDUSDT","status":"1","baseAsset":"LIMBAND","baseAssetPrecision":2,` +
			`"quoteAsset":"USDT","quotePrecision":4,"quoteAssetPrecision":4,"orderTypes":["LIMIT","MARKET"],"isSpotTradingAllowed":true,` +
			`"quoteAmountPrecision":"1","baseSizePrecision":"0","maxQuoteAmount":"2000000","quoteAmountPrecisionMarket":"1","maxQuoteAmountMarket":"100000",` +
			`"filters":[{"filterType":"PERCENT_PRICE_BY_SIDE","bidMultiplierUp":"0.2","askMultiplierDown":"0.1"}]},` +
			`{"symbol":"LIMNONEUSDT","status":"1","baseAsset":"LIMNONE","baseAssetPrecision":2,"quoteAsset":"USDT","quotePrecision":4,` +
			`"quoteAssetPrecision":4,"orderTypes":["LIMIT"],"isSpotTradingAllowed":true,"quoteAmountPrecision":"1","filters":[]}]}`,
	}), livePublic)
	require.NoError(t, ex.UpdateOrderExecutionLimits(t.Context(), asset.Spot), "UpdateOrderExecutionLimits must not error")
	if !mockTests {
		// Every spot symbol carries the band live, so the traded pair must have one inside (0, 1) and above 1.
		l, err := ex.GetOrderExecutionLimits(asset.Spot, spotTradablePair)
		require.NoError(t, err, "GetOrderExecutionLimits must not error")
		assert.Greater(t, l.MultiplierUp, 1.0, "MultiplierUp should be 1+bidMultiplierUp")
		assert.Greater(t, l.MultiplierDown, 0.0, "MultiplierDown should be 1-askMultiplierDown")
		assert.Less(t, l.MultiplierDown, 1.0, "MultiplierDown should be 1-askMultiplierDown")
		return
	}

	banded, err := ex.GetOrderExecutionLimits(asset.Spot, currency.NewPair(currency.NewCode("LIMBAND"), currency.USDT))
	require.NoError(t, err, "GetOrderExecutionLimits must not error")
	assert.InDelta(t, 1.2, banded.MultiplierUp, 1e-12, "MultiplierUp should be 1+bidMultiplierUp")
	assert.InDelta(t, 0.9, banded.MultiplierDown, 1e-12, "MultiplierDown should be 1-askMultiplierDown")

	unbanded, err := ex.GetOrderExecutionLimits(asset.Spot, currency.NewPair(currency.NewCode("LIMNONE"), currency.USDT))
	require.NoError(t, err, "GetOrderExecutionLimits must not error")
	assert.Zero(t, unbanded.MultiplierUp, "a symbol without the filter should carry no upper band")
	assert.Zero(t, unbanded.MultiplierDown, "a symbol without the filter should carry no lower band")
}
