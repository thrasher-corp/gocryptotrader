package mexc

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
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
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	b := ex.GetBase()
	b.SkipAuthCheck = true
	require.NoError(t, b.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), srv.URL), "SetRunningURL must not error")
	require.NoError(t, ex.setEnabledPairs(spotTradablePair), "setEnabledPairs must not error")
	return ex
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
	ex := newPrivateTestExchange(t, jsonHandler(t, map[string]string{
		"capital/deposit/hisrec": `[{"amount":"1.5","coin":"USDT","network":"TRC20","status":5,` +
			`"address":"addr","txId":"txhash","confirmTimes":"241","insertTime":1704067200000}]`,
		"capital/withdraw/history": `[]`,
	}))
	result, err := ex.GetAccountFundingHistory(t.Context())
	require.NoError(t, err, "GetAccountFundingHistory must not error on a counter-valued confirmTimes")
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
	ex := newPrivateTestExchange(t, jsonHandler(t, map[string]string{
		"allOrders": `[{"symbol":"BTCUSDT","orderId":"111","price":"50000","origQty":"0.5",` +
			`"executedQty":"0.2","cummulativeQuoteQty":"10000","type":"IMMEDIATE_OR_CANCEL",` +
			`"side":"BUY","status":"PARTIALLY_FILLED","time":1704067200000,"updateTime":1704067260000}]`,
	}))
	orders, err := ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
		AssetType: asset.Spot,
		Pairs:     currency.Pairs{spotTradablePair},
		Side:      order.AnySide,
		Type:      order.AnyType,
	})
	require.NoError(t, err, "GetOrderHistory must not error on an IMMEDIATE_OR_CANCEL order")
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
	ex := newPrivateTestExchange(t, jsonHandler(t, map[string]string{
		"account": `{"accountType":"SPOT","canTrade":true,"balances":[{"asset":"USDT","free":"10","locked":"3"}]}`,
	}))
	subAccounts, err := ex.UpdateAccountBalances(t.Context(), asset.Spot)
	require.NoError(t, err, "UpdateAccountBalances must not error")
	require.Len(t, subAccounts, 1, "one sub-account must be returned")
	bal := subAccounts[0].Balances[currency.USDT]
	assert.Equal(t, 13.0, bal.Total, "Total should be free + locked")
	assert.Equal(t, 3.0, bal.Hold, "Hold should be locked")
	assert.Equal(t, 10.0, bal.Free, "Free should be the available (free) balance")
}

// TestCancelOrderFormatsSymbol sends the delimiter-free symbol the exchange expects. Contract: group
// T defect #4 (symbol format).
func TestCancelOrderFormatsSymbol(t *testing.T) {
	t.Parallel()
	var sentSymbol string
	ex := newPrivateTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		sentSymbol = r.URL.Query().Get("symbol")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"symbol":"BTCUSDT","orderId":"111"}`))
	})
	err := ex.CancelOrder(t.Context(), &order.Cancel{
		OrderID:   "111",
		AssetType: asset.Spot,
		Pair:      currency.NewPairWithDelimiter("BTC", "USDT", "-"),
	})
	require.NoError(t, err, "CancelOrder must not error")
	assert.Equal(t, "BTCUSDT", sentSymbol, "the cancel should send the delimiter-free symbol")
}

// TestCancelAllOrdersNoOrderID is a symbol-wide cancel: it must not require an order id
// (StandardCancel is not used on the symbol-wide path).
func TestCancelAllOrdersNoOrderID(t *testing.T) {
	t.Parallel()
	ex := newPrivateTestExchange(t, jsonHandler(t, map[string]string{
		"openOrders": `[]`,
	}))
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
	ex := newPrivateTestExchange(t, jsonHandler(t, map[string]string{
		"tradeFee": `{"code":0,"data":{"makerCommission":0.001,"takerCommission":0.002}}`,
	}))
	taker, err := ex.GetFeeByType(t.Context(), &exchange.FeeBuilder{
		FeeType:       exchange.CryptocurrencyTradeFee,
		Pair:          spotTradablePair,
		PurchasePrice: 50000,
		Amount:        0.5,
	})
	require.NoError(t, err, "GetFeeByType must not error")
	assert.InDelta(t, 50.0, taker, 1e-9, "taker fee should be rate * price * quantity")
}

// TestGetFeeByTypeFormatsPair pins the symbol the authenticated fee request puts on the wire: MEXC
// rejects the delimited config-format pair a caller gets from GetEnabledPairs.
func TestGetFeeByTypeFormatsPair(t *testing.T) {
	t.Parallel()
	var got url.Values
	ex := newPrivateTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"data":{"makerCommission":0.001,"takerCommission":0.002}}`))
	})
	_, err := ex.GetFeeByType(t.Context(), &exchange.FeeBuilder{
		FeeType:       exchange.CryptocurrencyTradeFee,
		Pair:          currency.NewPairWithDelimiter("BTC", "USDT", currency.DashDelimiter),
		PurchasePrice: 50000,
		Amount:        0.5,
	})
	require.NoError(t, err, "GetFeeByType must not error")
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
	ex := newPrivateTestExchange(t, jsonHandler(t, map[string]string{
		// First order is open: updateTime is null. Second order carries a real updateTime.
		"openOrders": `[{"symbol":"BTCUSDT","orderId":"111","price":"50000","origQty":"0.5",` +
			`"executedQty":"0","cummulativeQuoteQty":"0","type":"LIMIT","side":"BUY",` +
			`"status":"NEW","time":1704067200000,"updateTime":null},` +
			`{"symbol":"BTCUSDT","orderId":"222","price":"50000","origQty":"0.5",` +
			`"executedQty":"0.2","cummulativeQuoteQty":"10000","type":"LIMIT","side":"BUY",` +
			`"status":"PARTIALLY_FILLED","time":1704067200000,"updateTime":1704067260000}]`,
	}))
	orders, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{
		AssetType: asset.Spot,
		Pairs:     currency.Pairs{spotTradablePair},
		Side:      order.AnySide,
		Type:      order.AnyType,
	})
	require.NoError(t, err, "GetActiveOrders must not error")
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
	ex := newPrivateTestExchange(t, jsonHandler(t, map[string]string{
		"order": `{"symbol":"BTCUSDT","orderId":"1","price":"19000","origQty":"1","executedQty":"0",` +
			`"type":"STOP_LIMIT","side":"SELL","status":"NEW","stopPrice":"18000","time":1704067200000}`,
	}))
	detail, err := ex.GetOrderInfo(t.Context(), "1", spotTradablePair, asset.Spot)
	require.NoError(t, err, "GetOrderInfo must not error")
	assert.Equal(t, 18000.0, detail.TriggerPrice, "TriggerPrice should carry the stop price")
	assert.Equal(t, order.StopLimit, detail.Type, "a STOP_LIMIT order should map to StopLimit")
}

// TestGetOrderHistoryMultiPair asserts GetOrderHistory queries every requested pair, not only the
// first, and rejects an empty pair set instead of querying across all symbols.
func TestGetOrderHistoryMultiPair(t *testing.T) {
	t.Parallel()
	e := newSignedTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		symbol := r.URL.Query().Get("symbol")
		// The mock reflects the requested symbol so the order id identifies which pair was queried;
		// it is a test double, not a live response.
		_, _ = w.Write([]byte(`[{"symbol":"` + symbol + `","orderId":"` + symbol + `-1","price":"1","origQty":"1","executedQty":"0","type":"LIMIT","side":"BUY","status":"NEW","time":1704067200000}]`)) //nolint:gosec // test mock reflecting the request
	}))
	btc := currency.NewBTCUSDT()
	eth := currency.NewPair(currency.ETH, currency.USDT)

	_, err := e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType})
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty, "an empty pair set must be rejected")

	orders, err := e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Pairs: currency.Pairs{btc, eth}, Side: order.AnySide, Type: order.AnyType})
	require.NoError(t, err, "GetOrderHistory must not error")
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
			ex := newSignedTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(body))
			}))
			require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{btc}, false), "storing available pairs must not error")
			require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{btc}, true), "storing enabled pairs must not error")

			_, err := tc.call(ex, &order.MultiOrderRequest{AssetType: asset.Spot, Pairs: currency.Pairs{btc}})
			require.ErrorIs(t, err, order.ErrSideIsInvalid, "a request with no side must be rejected")

			orders, err := tc.call(ex, &order.MultiOrderRequest{
				AssetType: asset.Spot, Pairs: currency.Pairs{btc}, Side: order.Buy, Type: order.Limit,
			})
			require.NoError(t, err, "the listing must not error")
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
	ex := newPrivateTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		path, got = r.URL.Path, r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"7213fea8e94b4a5593d507237e5a555b"}`))
	})
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

	_, err = ex.WithdrawCryptocurrencyFunds(t.Context(), &withdraw.Request{Exchange: ex.Name, Type: withdraw.Crypto})
	assert.Error(t, err, "an invalid withdrawal request should be rejected before it is sent")
}
