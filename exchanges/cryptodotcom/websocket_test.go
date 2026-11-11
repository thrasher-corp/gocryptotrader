package cryptodotcom

import (
	"bytes"
	"context"
	"errors"
	"math"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/buger/jsonparser"
	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestWebsocketMessageID(t *testing.T) {
	t.Parallel()
	// The cases run in order before the later calls of the same function, so each also checks that a call keeps
	// nothing from the last
	for _, tc := range []struct {
		name string
		msg  string
		exp  int64
	}{
		{"a numeric ID", `{"id":1792891216162,"method":"public/heartbeat","code":0}`, 1792891216162},
		{"a quoted numeric ID", `{"id":"7","method":"private/create-order"}`, 7},
		{"a negative ID", `{"id":-1,"method":"subscribe"}`, -1},
		{"no ID", `{"method":"subscribe"}`, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			id, err := websocketMessageID([]byte(tc.msg))
			require.NoErrorf(t, err, "websocketMessageID must not error for %s", tc.msg)
			assert.Equalf(t, tc.exp, id, "websocketMessageID should read the ID of %s", tc.msg)
		})
	}
	for _, tc := range []struct{ name, msg string }{
		{"a quoted ID that is no number", `{"id":"x"}`},
		{"a boolean ID", `{"id":true}`},
		{"a truncated message", `{"id":`},
		{"a fractional ID", `{"id":1.5}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := websocketMessageID([]byte(tc.msg))
			assert.ErrorIsf(t, err, errMalformedWebsocketMessage, "websocketMessageID should refuse %s", tc.msg)
		})
	}
}

func TestDecodeWebsocketResponse(t *testing.T) {
	t.Parallel()
	t.Run("a result where one is expected", func(t *testing.T) {
		t.Parallel()
		var result OrderIDResponse
		require.NoError(t, decodeWebsocketResponse([]byte(`{"id":1,"method":"private/create-order","code":0,"result":{"order_id":"5"}}`), &result), "decodeWebsocketResponse must not error")
		assert.Equal(t, OrderIDResponse{OrderID: "5"}, result, "decodeWebsocketResponse should decode the result")
	})
	for _, tc := range []struct{ name, response, method string }{
		{name: "no result", response: `{"id":1,"method":"private/cancel-order","code":0}`, method: "private/cancel-order"},
		{name: "a null result", response: `{"id":1,"method":"private/amend-order","code":0,"result":null}`, method: "private/amend-order"},
	} {
		t.Run(tc.name+" where one is expected", func(t *testing.T) {
			t.Parallel()
			result := OrderIDResponse{OrderID: "5"}
			err := decodeWebsocketResponse([]byte(tc.response), &result)
			assert.ErrorIsf(t, err, common.ErrNoResponse, "decodeWebsocketResponse should error for %s where one is expected", tc.name)
			assert.ErrorContainsf(t, err, tc.method, "decodeWebsocketResponse should name the method answered with %s", tc.name)
			assert.Equalf(t, OrderIDResponse{OrderID: "5"}, result, "decodeWebsocketResponse should leave the result alone for %s", tc.name)
		})
	}
	for _, tc := range []struct{ name, response string }{
		{name: "a result", response: `{"id":1,"method":"private/cancel-order","code":0,"result":{}}`},
		{name: "no result", response: `{"id":1,"method":"private/cancel-all-orders","code":0}`},
		{name: "a null result", response: `{"id":1,"method":"private/cancel-all-orders","code":0,"result":null}`},
	} {
		t.Run(tc.name+" where none is expected", func(t *testing.T) {
			t.Parallel()
			assert.NoErrorf(t, decodeWebsocketResponse([]byte(tc.response), nil), "decodeWebsocketResponse should not error for %s where none is expected", tc.name)
		})
	}
	t.Run("an error Crypto.com reports", func(t *testing.T) {
		t.Parallel()
		err := decodeWebsocketResponse([]byte(`{"id":11,"method":"public/auth","code":40101,"message":"Unauthorized","original":"{\"api_key\":\"mock-api-key\"}"}`), nil)
		var apiErr *APIError
		require.ErrorAs(t, err, &apiErr, "decodeWebsocketResponse must return the error Crypto.com reports")
		assert.Equal(t, &APIError{Code: 40101, Message: "Unauthorized"}, apiErr, "decodeWebsocketResponse should leave out the original request, which carries the API key")
	})
	for _, tc := range []struct{ name, response string }{
		{name: "a malformed response", response: `{"id":`},
		{name: "a result that cannot be decoded", response: `{"id":1,"code":0,"result":{"order_id":5}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var result OrderIDResponse
			assert.Errorf(t, decodeWebsocketResponse([]byte(tc.response), &result), "decodeWebsocketResponse should error for %s", tc.name)
		})
	}
}

func TestSendWebsocketRequest(t *testing.T) {
	t.Parallel()
	t.Run("connected", func(t *testing.T) {
		t.Parallel()
		skipLiveWithoutCredentials(t)
		ex, venue := newWebsocketTestExchange(t)
		var result *CancelOnDisconnectResponse
		require.NoError(t, ex.SendWebsocketRequest(t.Context(), "private/get-cancel-on-disconnect", nil, &result), "SendWebsocketRequest must not error")
		require.NotNil(t, result, "SendWebsocketRequest must decode the result")
		if mockTests {
			assert.Equal(t, &CancelOnDisconnectResponse{Scope: CancelOnDisconnectScopeConnection}, result, "SendWebsocketRequest should decode the result")
			requests := venue.received("private/get-cancel-on-disconnect")
			require.Len(t, requests, 1, "SendWebsocketRequest must send one request")
			assert.JSONEq(t, `{}`, string(requests[0].Params), "SendWebsocketRequest should send empty params without any")
		}
	})

	t.Run("connected with params", func(t *testing.T) {
		t.Parallel()
		skipLiveUnlessCanManipulate(t)
		ex, venue := newWebsocketTestExchange(t)
		var result *CancelOnDisconnectResponse
		require.NoError(t, ex.SendWebsocketRequest(t.Context(), "private/set-cancel-on-disconnect", &cancelOnDisconnectRequest{Scope: CancelOnDisconnectScopeConnection}, &result), "SendWebsocketRequest must not error")
		assert.Equal(t, &CancelOnDisconnectResponse{Scope: CancelOnDisconnectScopeConnection}, result, "SendWebsocketRequest should decode the scope set")
		if mockTests {
			requests := venue.received("private/set-cancel-on-disconnect")
			require.Len(t, requests, 1, "SendWebsocketRequest must send one request")
			assert.JSONEq(t, `{"scope":"CONNECTION"}`, string(requests[0].Params), "SendWebsocketRequest should send the params")
		}
	})

	t.Run("answers only the mock gives", func(t *testing.T) {
		t.Parallel()
		venue := newWebsocketVenue(t)
		ex := newTestExchange(t)
		connectWebsocketVenue(t, ex, venue)
		require.Len(t, venue.received(websocketMethodAuth), 1, "the trade connection must authenticate once")
		venue.respond("private/test-method", `{"order_id":"9","client_oid":"c"}`)
		var result OrderIDResponse
		require.NoError(t, ex.SendWebsocketRequest(t.Context(), "private/test-method", map[string]string{"order_id": "9"}, &result), "SendWebsocketRequest must not error")
		assert.Equal(t, OrderIDResponse{OrderID: "9", ClientOrderID: "c"}, result, "SendWebsocketRequest should decode the result")
		requests := venue.received("private/test-method")
		require.Len(t, requests, 1, "SendWebsocketRequest must send one request")
		assert.JSONEq(t, `{"order_id":"9"}`, string(requests[0].Params), "SendWebsocketRequest should send the params")
		assert.Positive(t, requests[0].ID, "SendWebsocketRequest should send a request ID")

		require.NoError(t, ex.SendWebsocketRequest(t.Context(), "private/test-method", nil, &result), "SendWebsocketRequest must not error without params")
		requests = venue.received("private/test-method")
		require.Len(t, requests, 2, "SendWebsocketRequest must send a second request")
		assert.JSONEq(t, `{}`, string(requests[1].Params), "SendWebsocketRequest should send empty params without any")
		assert.NotEqual(t, requests[0].ID, requests[1].ID, "SendWebsocketRequest should send each request with its own ID")

		venue.fail("private/test-method", &APIError{Code: 306, Message: "INSUFFICIENT_AVAILABLE_BALANCE"})
		err := ex.SendWebsocketRequest(t.Context(), "private/test-method", nil, &result)
		var apiErr *APIError
		require.ErrorAs(t, err, &apiErr, "SendWebsocketRequest must return the error Crypto.com reports")
		assert.Equal(t, &APIError{Code: 306, Message: "INSUFFICIENT_AVAILABLE_BALANCE"}, apiErr, "SendWebsocketRequest should decode the reported error")
		assert.Error(t, ex.SendWebsocketRequest(t.Context(), "private/test-method", make(chan int), &result), "SendWebsocketRequest should error for params that cannot be marshalled")

		ex.Websocket.SetCanUseAuthenticatedEndpoints(false)
		err = ex.SendWebsocketRequest(t.Context(), "private/test-method", nil, &result)
		assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "SendWebsocketRequest should refuse to send over a connected websocket without authentication")
	})

	t.Run("connected without a trade connection", func(t *testing.T) {
		t.Parallel()
		ex, err := newMarketDataTestExchange(t, newMarketDataVenue(t), subscription.List{{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: currency.Pairs{spotBTCUSD}}})
		require.NoError(t, err, "Connect must not error")
		require.True(t, ex.Websocket.IsConnected(), "the market data connection must connect the websocket")
		// The trade connection is dialled only while authenticated endpoints can be used, which they now can
		ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
		err = ex.SendWebsocketRequest(t.Context(), "private/get-cancel-on-disconnect", nil, nil)
		assert.ErrorIs(t, err, websocket.ErrNotConnected, "SendWebsocketRequest should report the trade connection not connected")
		assert.ErrorContains(t, err, websocketTradeConnection, "SendWebsocketRequest should report the trade connection, not the websocket, not connected")
	})

	t.Run("not connected", func(t *testing.T) {
		t.Parallel()
		ex := newTestExchange(t)
		// The cases share ex, setting its authentication in turn, so the second also checks the first leaves nothing behind
		for _, authenticated := range []bool{false, true} {
			t.Run(strconv.FormatBool(authenticated), func(t *testing.T) {
				ex.Websocket.SetCanUseAuthenticatedEndpoints(authenticated)
				err := ex.SendWebsocketRequest(t.Context(), "private/test-method", nil, nil)
				assert.ErrorIsf(t, err, websocket.ErrNotConnected, "SendWebsocketRequest should report a websocket not connected as such, whether or not it can authenticate (%t)", authenticated)
			})
		}
	})
}

func TestSendConnectionRequest(t *testing.T) {
	t.Parallel()
	t.Run("connected", func(t *testing.T) {
		t.Parallel()
		skipLiveWithoutCredentials(t)
		ex, venue := newWebsocketTestExchange(t)
		conn, err := ex.Websocket.GetConnection(websocketTradeConnection)
		require.NoError(t, err, "GetConnection must not error")
		var result *CancelOnDisconnectResponse
		require.NoError(t, ex.sendConnectionRequest(t.Context(), conn, "private/get-cancel-on-disconnect", nil, &result), "sendConnectionRequest must not error")
		require.NotNil(t, result, "sendConnectionRequest must decode the result")
		if mockTests {
			assert.Equal(t, &CancelOnDisconnectResponse{Scope: CancelOnDisconnectScopeConnection}, result, "sendConnectionRequest should decode the result")
			requests := venue.received("private/get-cancel-on-disconnect")
			require.Len(t, requests, 1, "sendConnectionRequest must send one request")
			assert.JSONEq(t, `{}`, string(requests[0].Params), "sendConnectionRequest should send empty params without any")
		}
	})

	t.Run("answers only the mock gives", func(t *testing.T) {
		t.Parallel()
		venue := newWebsocketVenue(t)
		ex := newTestExchange(t)
		connectWebsocketVenue(t, ex, venue)
		conn, err := ex.Websocket.GetConnection(websocketTradeConnection)
		require.NoError(t, err, "GetConnection must not error")
		venue.respond("private/test-method", `{"order_id":"9"}`)
		lastID := ex.websocketRequestID.Load()
		var result OrderIDResponse
		require.NoError(t, ex.sendConnectionRequest(t.Context(), conn, "private/test-method", map[string]string{"order_id": "9"}, &result), "sendConnectionRequest must not error")
		assert.Equal(t, OrderIDResponse{OrderID: "9"}, result, "sendConnectionRequest should decode the result")
		requests := venue.received("private/test-method")
		require.Len(t, requests, 1, "sendConnectionRequest must send one request")
		assert.JSONEq(t, `{"order_id":"9"}`, string(requests[0].Params), "sendConnectionRequest should send the params")
		assert.Equal(t, lastID+1, requests[0].ID, "sendConnectionRequest should send the next request ID")
		assert.InDelta(t, time.Now().UnixMilli(), requests[0].Nonce, float64(time.Minute.Milliseconds()), "sendConnectionRequest should send the time in milliseconds as its nonce, which Crypto.com checks is within a minute of its own clock")

		assert.Error(t, ex.sendConnectionRequest(t.Context(), conn, "private/test-method", make(chan int), nil), "sendConnectionRequest should error for params that cannot be marshalled")
		assert.Len(t, venue.received("private/test-method"), 1, "sendConnectionRequest should send nothing for params that cannot be marshalled")

		holdAnswers(t, venue, "private/test-method")
		assert.ErrorIs(t, ex.sendConnectionRequest(endedContext(t), conn, "private/test-method", nil, nil), context.Canceled, "sendConnectionRequest should stop at its context's end")
	})
}

func TestWebsocketAuthenticate(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("authentication failures are tested against the mock websocket")
	}
	venue := newWebsocketVenue(t)
	ex := newTestExchange(t)
	ex.SetCredentials(&accounts.Credentials{Key: testAPIKey, Secret: "wrong-secret"})
	require.NoError(t, ex.Websocket.SetAllConnectionURLs(venue.url), "SetAllConnectionURLs must not error")
	ex.Features.Subscriptions = subscription.List{}
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	// Disabling stops the connection monitor, which retries a refused connection until then
	t.Cleanup(func() { assert.NoError(t, ex.Websocket.Disable(), "Disable should not error") })
	t.Cleanup(venue.closing)
	err := ex.Websocket.Connect(t.Context())
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr, "Connect must fail when authentication is refused")
	assert.Equal(t, &APIError{Code: 40101, Message: "Unauthorized"}, apiErr, "Connect should report the refused authentication")
	auths := venue.received(websocketMethodAuth)
	require.NotEmpty(t, auths, "the trade connection must try to authenticate")
	// The connection monitor retries the refused connection on a new connection each time
	connections := make(common.Seen[int], len(auths))
	var repeated []int
	for i := range auths {
		if connections.CheckAndRecord(auths[i].Connection) {
			repeated = append(repeated, auths[i].Connection)
		}
	}
	assert.Empty(t, repeated, "the trade connection should try to authenticate once a connection")
}

// newSubscriptionLimitsExchange returns an exchange connected to venue, with 220 enabled pairs whose four channels take
// more than a request and a connection hold, and whose book channels alone take more than a request, the longest request
// ID making the longest requests, and the channels it generates and the error connecting
func newSubscriptionLimitsExchange(t *testing.T, venue *websocketVenue) (ex *Exchange, channels []string, err error) {
	t.Helper()
	pairs := make(currency.Pairs, 220)
	for i := range pairs {
		pairs[i] = currency.NewPairWithDelimiter("TESTCOIN"+strconv.Itoa(i), "USD", currency.UnderscoreDelimiter)
	}
	ex, err = newMarketDataTestExchange(t, venue, subscription.List{
		{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel},
		{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel, Levels: defaultBookDepth, Interval: kline.HundredMilliseconds},
		{Enabled: true, Asset: asset.Spot, Channel: subscription.AllTradesChannel},
		{Enabled: true, Asset: asset.Spot, Channel: subscription.CandlesChannel, Interval: kline.OneMin},
	}, func(ex *Exchange) {
		require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Spot, pairs, false), "StorePairs must store the available pairs")
		require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Spot, pairs, true), "StorePairs must store the enabled pairs")
		ex.websocketRequestID.Store(math.MaxInt64 - 1000)
	})
	generated, genErr := ex.generateMarketDataSubscriptions()
	require.NoError(t, genErr, "generateMarketDataSubscriptions must not error")
	require.Len(t, generated, 880, "every enabled pair must generate its channels")
	return ex, qualifiedChannels(generated), err
}

// requestedChannels returns the channels each of a method's requests names, checking the request holds no more than
// Crypto.com answers and, for a subscribe request, names its channels' book parameters
func requestedChannels(t *testing.T, venue *websocketVenue, method string) [][]string {
	t.Helper()
	requests := venue.received(method)
	requested := make([][]string, 0, len(requests))
	for _, r := range requests {
		assert.LessOrEqualf(t, r.Bytes, 4096, "a %s request should be no longer than Crypto.com answers", method)
		var params subscriptionRequest
		require.NoErrorf(t, json.Unmarshal(r.Params, &params), "a %s request's params must decode", method)
		require.NotEmptyf(t, params.Channels, "a %s request must name channels", method)
		for _, channel := range params.Channels {
			if method == websocketMethodSubscribe && strings.HasPrefix(channel, channelBook+".") {
				assert.Equalf(t, BookSubscriptionTypeSnapshotAndUpdate, params.BookSubscriptionType, "%s's request should name its book subscription type", channel)
				assert.Equalf(t, uint64(100), params.BookUpdateFrequency, "%s's request should name its book update frequency", channel)
			} else {
				assert.Emptyf(t, params.BookSubscriptionType, "%s's request should name no book subscription type", channel)
				assert.Zerof(t, params.BookUpdateFrequency, "%s's request should name no book update frequency", channel)
			}
		}
		requested = append(requested, params.Channels)
	}
	return requested
}

func TestWebsocketSubscriptionLimits(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("what a request holds and its refusal are tested against the mock websocket, which records each request and can refuse it")
	}
	t.Run("channels split across requests and connections", func(t *testing.T) {
		t.Parallel()
		venue := newWebsocketVenue(t)
		ex, channels, err := newSubscriptionLimitsExchange(t, venue)
		require.NoError(t, err, "Connect must not error")
		assert.ElementsMatch(t, channels, slices.Concat(requestedChannels(t, venue, websocketMethodSubscribe)...), "each channel should be subscribed once")
		assert.ElementsMatch(t, channels, qualifiedChannels(ex.Websocket.GetSubscriptions()), "every channel should be stored")
		conn, err := ex.Websocket.GetConnection(websocketMarketDataConnection)
		require.NoError(t, err, "GetConnection must not error")
		subs := conn.Subscriptions().List()
		require.Len(t, subs, 400, "the first connection must hold as many channels as Crypto.com allows it")

		require.NoError(t, ex.wsUnsubscribe(t.Context(), conn, nil), "wsUnsubscribe must not error without channels")
		assert.Empty(t, venue.received(websocketMethodUnsubscribe), "wsUnsubscribe should send nothing without channels")
		require.NoError(t, ex.wsUnsubscribe(t.Context(), conn, subs), "wsUnsubscribe must not error")
		assert.ElementsMatch(t, qualifiedChannels(subs), slices.Concat(requestedChannels(t, venue, websocketMethodUnsubscribe)...), "each channel should be unsubscribed once")
		assert.Len(t, ex.Websocket.GetSubscriptions(), len(channels)-len(subs), "only that connection's channels should be removed")
	})

	// Crypto.com refuses the first request of several, whose channels alone are left out or kept, and then one channel of a
	// request
	t.Run("a refused request or channel", func(t *testing.T) {
		t.Parallel()
		refused := &APIError{Code: 42901, Message: "TOO_MANY_REQUESTS"}
		refuseFirst := func(venue *websocketVenue, method string) {
			venue.fail(method, refused)
			venue.beforeAnswering(method, func() { venue.fail(method, nil) })
		}
		var apiErr *APIError

		venue := newWebsocketVenue(t)
		refuseFirst(venue, websocketMethodSubscribe)
		ex, channels, err := newSubscriptionLimitsExchange(t, venue)
		require.ErrorAs(t, err, &apiErr, "Connect must report the refused request")
		assert.Equal(t, refused, apiErr, "Connect should report the refused request's error")
		requested := requestedChannels(t, venue, websocketMethodSubscribe)
		require.Greater(t, len(requested), 1, "the channels must take more than one request")
		assert.ElementsMatch(t, channels, slices.Concat(requested...), "every request should be sent")
		assert.ElementsMatch(t, slices.Concat(requested[1:]...), qualifiedChannels(ex.Websocket.GetSubscriptions()), "only the refused request's channels should be left out")

		venue = newWebsocketVenue(t)
		ex, channels, err = newSubscriptionLimitsExchange(t, venue)
		require.NoError(t, err, "Connect must not error")
		conn, err := ex.Websocket.GetConnection(websocketMarketDataConnection)
		require.NoError(t, err, "GetConnection must not error")
		subs := conn.Subscriptions().List()
		refuseFirst(venue, websocketMethodUnsubscribe)
		require.ErrorAs(t, ex.wsUnsubscribe(t.Context(), conn, subs), &apiErr, "wsUnsubscribe must report the refused request")
		assert.Equal(t, refused, apiErr, "wsUnsubscribe should report the refused request's error")
		requested = requestedChannels(t, venue, websocketMethodUnsubscribe)
		require.Greater(t, len(requested), 1, "the channels must take more than one request")
		assert.ElementsMatch(t, qualifiedChannels(subs), slices.Concat(requested...), "every request should be sent")
		unsubscribed := slices.Concat(requested[1:]...)
		kept := slices.DeleteFunc(channels, func(channel string) bool { return slices.Contains(unsubscribed, channel) })
		assert.ElementsMatch(t, kept, qualifiedChannels(ex.Websocket.GetSubscriptions()), "only the channels of the requests acknowledged should be removed")

		// A channel refused ahead of others that are acknowledged, which Crypto.com answers in the order a request names them
		venue = newWebsocketVenue(t)
		venue.rejectSubscriptions("ticker.TESTCOIN0_USD", refused)
		ex, channels, err = newSubscriptionLimitsExchange(t, venue)
		require.ErrorAs(t, err, &apiErr, "Connect must report the refused channel")
		assert.Equal(t, refused, apiErr, "Connect should report the refused channel's error")
		kept = slices.DeleteFunc(channels, func(channel string) bool { return channel == "ticker.TESTCOIN0_USD" })
		assert.ElementsMatch(t, kept, qualifiedChannels(ex.Websocket.GetSubscriptions()), "only the refused channel should be left out")
	})
}

func TestWebsocketUnsubscribeBatch(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("an unsubscribe request is tested against the mock websocket, which records it")
	}
	venue := newMarketDataVenue(t)
	ex, err := newMarketDataTestExchange(t, venue, subscription.List{{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: currency.Pairs{spotBTCUSD, spotBTCUSDT}}})
	require.NoError(t, err, "Connect must not error")
	conn, err := ex.Websocket.GetConnection(websocketMarketDataConnection)
	require.NoError(t, err, "GetConnection must not error")
	subs := conn.Subscriptions().List()
	require.Len(t, subs, 2, "both channels must be subscribed")

	refused := &APIError{Code: 40001, Message: "BAD_REQUEST"}
	venue.fail(websocketMethodUnsubscribe, refused)
	var apiErr *APIError
	require.ErrorAs(t, ex.wsUnsubscribeBatch(t.Context(), conn, subs[:1]), &apiErr, "wsUnsubscribeBatch must report a refused request")
	assert.Equal(t, refused, apiErr, "wsUnsubscribeBatch should report the refused request's error")
	assert.Len(t, ex.Websocket.GetSubscriptions(), 2, "wsUnsubscribeBatch should keep the channels of a refused request")

	venue.fail(websocketMethodUnsubscribe, nil)
	require.NoError(t, ex.wsUnsubscribeBatch(t.Context(), conn, subs[:1]), "wsUnsubscribeBatch must not error")
	stored := ex.Websocket.GetSubscriptions()
	require.Len(t, stored, 1, "wsUnsubscribeBatch must remove the channel it names")
	assert.Equal(t, subs[1].QualifiedChannel, stored[0].QualifiedChannel, "wsUnsubscribeBatch should keep the channel it does not name")
	requests := venue.received(websocketMethodUnsubscribe)
	require.Len(t, requests, 2, "each call must send one request")
	assert.JSONEq(t, `{"channels":["`+subs[0].QualifiedChannel+`"]}`, string(requests[1].Params), "wsUnsubscribeBatch should name its channels and nothing else")
	assert.ErrorIs(t, ex.wsUnsubscribeBatch(t.Context(), conn, subs[:1]), subscription.ErrNotFound, "wsUnsubscribeBatch should report a channel it cannot remove")

	t.Run("a book channel's books are invalidated once it is unsubscribed", func(t *testing.T) {
		t.Parallel()
		venue := newMarketDataVenue(t)
		answerBookSnapshotFirst(t, venue)
		ex, err := newMarketDataTestExchange(t, venue, recordedBookChannelSubscription())
		require.NoError(t, err, "Connect must not error")
		conn, err := ex.Websocket.GetConnection(websocketMarketDataConnection)
		require.NoError(t, err, "GetConnection must not error")
		requireBooksAt(t, ex, spotBTCUSD, recordedBookSnapshotUpdateID)
		s := conn.Subscriptions().Get(qualifiedChannelKey{&subscription.Subscription{QualifiedChannel: recordedBookChannel}})
		require.NotNil(t, s, "the book channel must be subscribed")
		venue.fail(websocketMethodUnsubscribe, &APIError{Code: 40001, Message: "BAD_REQUEST"})
		require.Error(t, ex.wsUnsubscribeBatch(t.Context(), conn, subscription.List{s}), "wsUnsubscribeBatch must report a refused request")
		requireBooksAt(t, ex, spotBTCUSD, recordedBookSnapshotUpdateID)
		venue.fail(websocketMethodUnsubscribe, nil)
		require.NoError(t, ex.wsUnsubscribeBatch(t.Context(), conn, subscription.List{s}), "wsUnsubscribeBatch must not error")
		for _, a := range []asset.Item{asset.Spot, asset.Margin} {
			_, err := ex.Websocket.Orderbook.LastUpdateID(spotBTCUSD, a)
			assert.ErrorIsf(t, err, errBookUnsubscribed, "wsUnsubscribeBatch should invalidate the %s book the channel loaded", a)
		}
	})
	t.Run("a book channel's books are invalidated when its unsubscription goes unanswered", func(t *testing.T) {
		t.Parallel()
		venue := newMarketDataVenue(t)
		answerBookSnapshotFirst(t, venue)
		ex, err := newMarketDataTestExchange(t, venue, recordedBookChannelSubscription(), withTestResponseMaxLimit(t, venue))
		require.NoError(t, err, "Connect must not error")
		conn, err := ex.Websocket.GetConnection(websocketMarketDataConnection)
		require.NoError(t, err, "GetConnection must not error")
		requireBooksAt(t, ex, spotBTCUSD, recordedBookSnapshotUpdateID)
		s := conn.Subscriptions().Get(qualifiedChannelKey{&subscription.Subscription{QualifiedChannel: recordedBookChannel}})
		require.NotNil(t, s, "the book channel must be subscribed")
		venue.loseAnswers(websocketMethodUnsubscribe, 1)
		require.ErrorIs(t, ex.wsUnsubscribeBatch(t.Context(), conn, subscription.List{s}), websocket.ErrSignatureTimeout, "wsUnsubscribeBatch must return the unanswered request's timeout")
		for _, a := range []asset.Item{asset.Spot, asset.Margin} {
			_, err := ex.Websocket.Orderbook.LastUpdateID(spotBTCUSD, a)
			assert.ErrorIsf(t, err, errBookUnsubscribed, "wsUnsubscribeBatch should invalidate the %s book of a channel Crypto.com may have stopped", a)
		}
	})
	t.Run("a resubscribing book channel's books are invalidated when its unanswered unsubscription is refused once more", func(t *testing.T) {
		t.Parallel()
		venue := newMarketDataVenue(t)
		answerBookSnapshotFirst(t, venue)
		ex, err := newMarketDataTestExchange(t, venue, recordedBookChannelSubscription(), withTestResponseMaxLimit(t, venue))
		require.NoError(t, err, "Connect must not error")
		conn, err := ex.Websocket.GetConnection(websocketMarketDataConnection)
		require.NoError(t, err, "GetConnection must not error")
		requireBooksAt(t, ex, spotBTCUSD, recordedBookSnapshotUpdateID)
		s := conn.Subscriptions().Get(qualifiedChannelKey{&subscription.Subscription{QualifiedChannel: recordedBookChannel}})
		require.NotNil(t, s, "the book channel must be subscribed")
		require.NoError(t, s.SetState(subscription.ResubscribingState), "SetState must not error")
		venue.loseAnswers(websocketMethodUnsubscribe, 1)
		venue.fail(websocketMethodUnsubscribe, &APIError{Code: 40001, Message: "BAD_REQUEST"})
		var apiErr *APIError
		require.ErrorAs(t, ex.wsUnsubscribeBatch(t.Context(), conn, subscription.List{s}), &apiErr, "wsUnsubscribeBatch must return the second request's refusal")
		assert.Len(t, venue.received(websocketMethodUnsubscribe), 2, "wsUnsubscribeBatch should unsubscribe the resubscribing book channel once more")
		for _, a := range []asset.Item{asset.Spot, asset.Margin} {
			_, err := ex.Websocket.Orderbook.LastUpdateID(spotBTCUSD, a)
			assert.ErrorIsf(t, err, errBookUnsubscribed, "wsUnsubscribeBatch should invalidate the %s book, as the unanswered request may have stopped the channel", a)
		}
	})
	t.Run("a book channel's books stay when its unsubscription fails otherwise", func(t *testing.T) {
		t.Parallel()
		venue := newMarketDataVenue(t)
		answerBookSnapshotFirst(t, venue)
		ex, err := newMarketDataTestExchange(t, venue, recordedBookChannelSubscription(), withTestResponseMaxLimit(t, venue))
		require.NoError(t, err, "Connect must not error")
		conn, err := ex.Websocket.GetConnection(websocketMarketDataConnection)
		require.NoError(t, err, "GetConnection must not error")
		requireBooksAt(t, ex, spotBTCUSD, recordedBookSnapshotUpdateID)
		s := conn.Subscriptions().Get(qualifiedChannelKey{&subscription.Subscription{QualifiedChannel: recordedBookChannel}})
		require.NotNil(t, s, "the book channel must be subscribed")
		require.ErrorIs(t, ex.wsUnsubscribeBatch(endedContext(t), conn, subscription.List{s}), context.Canceled, "wsUnsubscribeBatch must return the cancellation")
		requireBooksAt(t, ex, spotBTCUSD, recordedBookSnapshotUpdateID)
	})

	// Only a book channel resubscribing alone is unsubscribed once more, and only when its request times out
	for _, tc := range []struct {
		name     string
		channels []string
		state    subscription.State
		// lost is how many unsubscribe requests the mock leaves unanswered, and refusal what it refuses them with
		lost    int
		refusal *APIError
		// requests is how many unsubscribe requests are sent, and err the error unsubscribing ends with
		requests int
		err      error
	}{
		{name: "a resubscribing book channel answered when sent again", channels: []string{"book.BTC_USD.10"}, state: subscription.ResubscribingState, lost: 1, requests: 2},
		{name: "a resubscribing book channel unanswered twice", channels: []string{"book.BTC_USD.10"}, state: subscription.ResubscribingState, lost: 2, requests: 2, err: websocket.ErrSignatureTimeout},
		{name: "a book channel not resubscribing", channels: []string{"book.BTC_USD.10"}, state: subscription.SubscribedState, lost: 1, requests: 1, err: websocket.ErrSignatureTimeout},
		{name: "a resubscribing ticker channel", channels: []string{"ticker.BTC_USD"}, state: subscription.ResubscribingState, lost: 1, requests: 1, err: websocket.ErrSignatureTimeout},
		{name: "two resubscribing book channels", channels: []string{"book.BTC_USD.10", "book.ETH_USD.10"}, state: subscription.ResubscribingState, lost: 1, requests: 1, err: websocket.ErrSignatureTimeout},
		{name: "a resubscribing book channel refused", channels: []string{"book.BTC_USD.10"}, state: subscription.ResubscribingState, refusal: &APIError{Code: 50001, Message: "ERR_INTERNAL"}, requests: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			venue := newWebsocketVenue(t)
			ex, err := newMarketDataTestExchange(t, venue, subscription.List{
				{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel, Levels: 10, Pairs: currency.Pairs{spotBTCUSD, spotETHUSD}},
				{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: currency.Pairs{spotBTCUSD}},
			}, withTestResponseMaxLimit(t, venue))
			require.NoError(t, err, "Connect must not error")
			conn, err := ex.Websocket.GetConnection(websocketMarketDataConnection)
			require.NoError(t, err, "GetConnection must not error")
			subs := make(subscription.List, len(tc.channels))
			for i, channel := range tc.channels {
				subs[i] = conn.Subscriptions().Get(qualifiedChannelKey{&subscription.Subscription{QualifiedChannel: channel}})
				require.NotNilf(t, subs[i], "%s must be subscribed", channel)
				if tc.state != subscription.SubscribedState {
					require.NoErrorf(t, subs[i].SetState(tc.state), "SetState must not error for %s", channel)
				}
			}
			venue.loseAnswers(websocketMethodUnsubscribe, tc.lost)
			venue.fail(websocketMethodUnsubscribe, tc.refusal)

			err = ex.wsUnsubscribeBatch(t.Context(), conn, subs)
			answered := tc.err == nil && tc.refusal == nil
			switch {
			case tc.refusal != nil:
				var apiErr *APIError
				require.ErrorAs(t, err, &apiErr, "wsUnsubscribeBatch must report the refused request")
				assert.Equal(t, tc.refusal, apiErr, "wsUnsubscribeBatch should report the refused request's error")
			case tc.err != nil:
				require.ErrorIs(t, err, tc.err, "wsUnsubscribeBatch must report the request timing out")
			default:
				require.NoError(t, err, "wsUnsubscribeBatch must not error once its request is answered")
			}
			requests := venue.received(websocketMethodUnsubscribe)
			require.Len(t, requests, tc.requests, "wsUnsubscribeBatch must send the expected number of unsubscribe requests")
			for _, r := range requests[1:] {
				assert.Equal(t, requests[0].Connection, r.Connection, "wsUnsubscribeBatch should send its request again on the same connection")
				assert.NotEqual(t, requests[0].ID, r.ID, "wsUnsubscribeBatch should send its request again with a request ID of its own")
				assert.JSONEq(t, string(requests[0].Params), string(r.Params), "wsUnsubscribeBatch should send the same channels again")
			}
			for _, s := range subs {
				stored := ex.Websocket.GetSubscription(qualifiedChannelKey{&subscription.Subscription{QualifiedChannel: s.QualifiedChannel}})
				if answered {
					assert.Nilf(t, stored, "wsUnsubscribeBatch should remove %s once its request is answered", s.QualifiedChannel)
					assert.Equalf(t, subscription.UnsubscribedState, s.State(), "wsUnsubscribeBatch should leave %s unsubscribed", s.QualifiedChannel)
					continue
				}
				assert.NotNilf(t, stored, "wsUnsubscribeBatch should keep %s when its request fails", s.QualifiedChannel)
				assert.Equalf(t, tc.state, s.State(), "wsUnsubscribeBatch should leave the state of %s alone when its request fails", s.QualifiedChannel)
			}
		})
	}
}

// testResponseMaxLimit is how long the connections of an exchange that withTestResponseMaxLimit prepares wait for each
// answer, so that an answer the mock loses times out soon
const testResponseMaxLimit = 500 * time.Millisecond

// withTestResponseMaxLimit returns a function preparing an exchange whose connections wait testResponseMaxLimit for each
// answer from venue; it sets the websocket up again, as each connection takes its wait from the configuration when the
// websocket is set up
func withTestResponseMaxLimit(t *testing.T, venue *websocketVenue) func(*Exchange) {
	t.Helper()
	return func(ex *Exchange) {
		ex.Config.WebsocketResponseMaxLimit = testResponseMaxLimit
		ex.Websocket = sharedtestvalues.NewTestWebsocket()
		require.NoError(t, ex.setupWebsocket(ex.Config), "setupWebsocket must not error")
		require.NoError(t, ex.Websocket.SetAllConnectionURLs(venue.url), "SetAllConnectionURLs must not error")
	}
}

// holdAnswers has the mock hold back its answers to a method's requests until the test ends, so that an answer cannot end
// a wait for one before its context does, as an ended context does not stop a request being sent when the rate limiter
// has no delay
func holdAnswers(t *testing.T, venue *websocketVenue, method string) {
	t.Helper()
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	venue.beforeAnswering(method, func() { <-release })
}

// endedContext returns a context that has already ended
func endedContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	return ctx
}

func TestWebsocketSubscriptionRequestsStopAtContextEnd(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("a request's end is tested against the mock websocket, which can hold back its answers")
	}
	venue := newMarketDataVenue(t)
	ex, err := newMarketDataTestExchange(t, venue, subscription.List{{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: currency.Pairs{spotBTCUSD}}})
	require.NoError(t, err, "Connect must not error")
	conn, err := ex.Websocket.GetConnection(websocketMarketDataConnection)
	require.NoError(t, err, "GetConnection must not error")
	subs := conn.Subscriptions().List()
	require.Len(t, subs, 1, "the channel must be subscribed")

	holdAnswers(t, venue, websocketMethodSubscribe)
	holdAnswers(t, venue, websocketMethodUnsubscribe)
	assert.ErrorIs(t, ex.wsSubscribe(endedContext(t), conn, subs), context.Canceled, "wsSubscribe should stop at its context's end")
	assert.ErrorIs(t, ex.wsUnsubscribe(endedContext(t), conn, subs), context.Canceled, "wsUnsubscribe should stop at its context's end")
	assert.ErrorIs(t, ex.wsUnsubscribeBatch(endedContext(t), conn, subs), context.Canceled, "wsUnsubscribeBatch should stop at its context's end")
	assert.Len(t, ex.Websocket.GetSubscriptions(), 1, "a channel whose unsubscribe request was not answered should stay stored")
}

func TestChannelLists(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		sizes []int
		exp   []int
	}{
		{"one channel", []int{100}, []int{1}},
		{"channels filling a request", []int{3000, 900}, []int{2}},
		{"a channel past what a request holds", []int{3000, 901}, []int{1, 1}},
		{"channels past what a request holds, more than once", []int{3000, 901, 3000, 900, 3}, []int{1, 1, 2, 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var subs subscription.List
			for _, size := range tc.sizes {
				subs = append(subs, &subscription.Subscription{QualifiedChannel: strings.Repeat("c", size-len(`"",`))})
			}
			lists := channelLists(subs)
			got := make([]int, len(lists))
			for i := range lists {
				got[i] = len(lists[i])
			}
			assert.Equal(t, tc.exp, got, "channelLists should split the channels where they pass what a request holds")
			assert.Equal(t, subs, slices.Concat(lists...), "channelLists should keep every channel, in order")
		})
	}
}

func TestWebsocketHeartbeat(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("heartbeat replies are tested against the mock websocket")
	}
	ex, venue := newWebsocketTestExchange(t)
	conn, err := ex.Websocket.GetConnection(websocketTradeConnection)
	require.NoError(t, err, "GetConnection must not error")
	assert.ErrorIs(t, ex.wsHandleData(t.Context(), conn, []byte(`{"id":1,"method":"public/heartbeat`)), errMalformedWebsocketMessage, "wsHandleData should refuse an unterminated method")
	assert.ErrorIs(t, ex.wsHandleData(t.Context(), conn, []byte(`{"id":true,"method":"public/heartbeat"}`)), errMalformedWebsocketMessage, "wsHandleData should refuse an ID that is neither a number nor a string")
	require.NoError(t, ex.wsHandleData(t.Context(), conn, []byte(`{"id":1792891216162,"method":"public/heartbeat","code":0}`)), "wsHandleData must not error for a heartbeat")
	require.Eventually(t, func() bool { return len(venue.received(websocketMethodRespondHeartbeat)) == 1 }, 5*time.Second, 10*time.Millisecond, "the heartbeat must be answered")
	reply := venue.received(websocketMethodRespondHeartbeat)[0]
	assert.Equal(t, int64(1792891216162), reply.ID, "the heartbeat reply should carry the heartbeat's ID")
	assert.Empty(t, reply.Params, "the heartbeat reply should send no params")
}

func TestWebsocketSubscriptionRequest(t *testing.T) {
	t.Parallel()
	encoded, err := json.Marshal(&subscriptionRequest{Channels: []string{"book.BTC_USD.50"}, BookSubscriptionType: "SNAPSHOT_AND_UPDATE", BookUpdateFrequency: 10})
	require.NoError(t, err, "Marshal must not error")
	assert.JSONEq(t, `{"channels":["book.BTC_USD.50"],"book_subscription_type":"SNAPSHOT_AND_UPDATE","book_update_frequency":10}`, string(encoded), "subscriptionRequest should encode its book options")
	encoded, err = json.Marshal(&subscriptionRequest{Channels: []string{"ticker.BTC_USD"}})
	require.NoError(t, err, "Marshal must not error")
	assert.JSONEq(t, `{"channels":["ticker.BTC_USD"]}`, string(encoded), "subscriptionRequest should leave out unset book options")
}

func TestWebsocketResponseSubscribedChannel(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, msg, exp string }{
		{"an acknowledgement's channel", `{"id":1,"method":"subscribe","code":0,"channel":"ticker.BTC_USD"}`, "ticker.BTC_USD"},
		{"a result's subscription", `{"id":1,"method":"subscribe","code":0,"result":{"subscription":"book.BTC_USD.10","channel":"book","data":[]}}`, "book.BTC_USD.10"},
		{"a rejection's channel", `{"id":1,"method":"subscribe","code":40003,"channel":"ticker.BTC_USDT","message":"Unknown symbol"}`, "ticker.BTC_USDT"},
		{"a refusal naming no channel", `{"id":1,"method":"subscribe","code":40001,"message":"BAD_REQUEST"}`, ""},
		{"a result naming no subscription", `{"id":1,"method":"subscribe","code":0,"result":{"channel":"book","data":[]}}`, ""},
		{"a subscription that is no string", `{"id":1,"method":"subscribe","code":0,"result":{"subscription":1}}`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var resp websocketResponse
			require.NoErrorf(t, json.Unmarshal([]byte(tc.msg), &resp), "Unmarshal must not error for %s", tc.msg)
			assert.Equalf(t, tc.exp, resp.subscribedChannel(), "subscribedChannel should name the channel %s answers", tc.msg)
		})
	}
}

func TestSubscribeInspectorIsFinal(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		msg  string
		exp  bool
	}{
		{"a refusal", `{"id":1,"method":"subscribe","code":40001,"message":"BAD_REQUEST"}`, true},
		{"a refusal with a quoted code", `{"id":1,"method":"subscribe","code":"40001","message":"BAD_REQUEST"}`, true},
		{"a refusal whose result names no subscription", `{"id":1,"method":"subscribe","code":40001,"message":"BAD_REQUEST","result":{}}`, true},
		{"a refusal whose result is no object", `{"id":1,"method":"subscribe","code":40001,"message":"BAD_REQUEST","result":[]}`, true},
		{"a rejection naming its channel", `{"id":1,"method":"subscribe","code":40003,"channel":"ticker.BTC_USDT","message":"Unknown symbol"}`, false},
		{"a rejection naming its subscription", `{"id":1,"method":"subscribe","code":40003,"result":{"subscription":"ticker.BTC_USDT"}}`, false},
		{"an acknowledgement", `{"id":1,"method":"subscribe","code":0,"channel":"ticker.BTC_USD"}`, false},
		{"an acknowledgement naming no channel", `{"id":1,"method":"subscribe","code":0}`, false},
		{"a truncated message", `{"id":`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, tc.exp, subscribeInspector{}.IsFinal([]byte(tc.msg)), "IsFinal should report whether %s refuses its subscribe request as a whole", tc.msg)
		})
	}
}

func TestWebsocketAPISetCancelOnDisconnect(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketAPI().SetCancelOnDisconnect(t.Context(), "")
	require.ErrorIs(t, err, errScopeRequired, "SetCancelOnDisconnect must refuse an empty scope")
	skipLiveUnlessCanManipulate(t)

	ex, venue := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().SetCancelOnDisconnect(t.Context(), CancelOnDisconnectScopeConnection)
	require.NoError(t, err, "SetCancelOnDisconnect must not error")
	exp := &CancelOnDisconnectResponse{Scope: CancelOnDisconnectScopeConnection}
	assert.Equal(t, exp, result, "SetCancelOnDisconnect should decode the scope")
	if mockTests {
		requests := venue.received("private/set-cancel-on-disconnect")
		require.Len(t, requests, 1, "SetCancelOnDisconnect must send one request")
		assert.JSONEq(t, `{"scope":"CONNECTION"}`, string(requests[0].Params), "SetCancelOnDisconnect should send the scope")

		venue.respond("private/set-cancel-on-disconnect", `null`)
		_, err = ex.WebsocketAPI().SetCancelOnDisconnect(t.Context(), CancelOnDisconnectScopeConnection)
		assert.ErrorIs(t, err, common.ErrNoResponse, "SetCancelOnDisconnect should error for a null result")
	}
}

func TestWebsocketAPIGetCancelOnDisconnect(t *testing.T) {
	t.Parallel()
	ex, venue := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetCancelOnDisconnect(t.Context())
	require.NoError(t, err, "GetCancelOnDisconnect must not error")
	require.NotNil(t, result, "GetCancelOnDisconnect must return the scope")
	if mockTests {
		exp := &CancelOnDisconnectResponse{Scope: CancelOnDisconnectScopeConnection}
		assert.Equal(t, exp, result, "GetCancelOnDisconnect should decode the scope")
		requests := venue.received("private/get-cancel-on-disconnect")
		require.Len(t, requests, 1, "GetCancelOnDisconnect must send one request")
		assert.JSONEq(t, `{}`, string(requests[0].Params), "GetCancelOnDisconnect should send empty params")

		venue.respond("private/get-cancel-on-disconnect", `null`)
		_, err = ex.WebsocketAPI().GetCancelOnDisconnect(t.Context())
		assert.ErrorIs(t, err, common.ErrNoResponse, "GetCancelOnDisconnect should error for a null result")
	}
}

// expandedSubscription returns a subscription as its template expands it for one pair of an asset
func expandedSubscription(s *subscription.Subscription, a asset.Item, p currency.Pair, qualifiedChannel string) *subscription.Subscription {
	expanded := s.Clone()
	expanded.Asset = a
	expanded.Pairs = currency.Pairs{p}
	expanded.QualifiedChannel = qualifiedChannel
	return expanded
}

func TestGenerateMarketDataSubscriptions(t *testing.T) {
	t.Parallel()
	// expected is a subscription the template expands configured subscription sub to
	type expected struct {
		sub     int
		asset   asset.Item
		pair    currency.Pair
		channel string
	}
	all := func(channel string, extra ...func(*subscription.Subscription)) *subscription.Subscription {
		s := &subscription.Subscription{Enabled: true, Asset: asset.All, Channel: channel}
		for _, f := range extra {
			f(s)
		}
		return s
	}
	for _, tc := range []struct {
		name string
		subs subscription.List
		exp  []expected
	}{
		{
			name: "tickers of every asset, the margin tickers being the spot tickers",
			subs: subscription.List{all(subscription.TickerChannel)},
			exp: []expected{
				{0, asset.Spot, spotBTCUSD, "ticker.BTC_USD"},
				{0, asset.Spot, spotBTCUSDT, "ticker.BTC_USDT"},
				{0, asset.Spot, spotCROUSD, "ticker.CRO_USD"},
				{0, asset.Spot, spotETHUSD, "ticker.ETH_USD"},
				{0, asset.Futures, futureBTCUSD, "ticker.BTCUSD-261030"},
				{0, asset.PerpetualSwap, perpBTCUSD, "ticker.BTCUSD-PERP"},
				{0, asset.PerpetualSwap, perpETHUSD, "ticker.ETHUSD-PERP"},
			},
		},
		{
			name: "margin tickers",
			subs: subscription.List{{Enabled: true, Asset: asset.Margin, Channel: subscription.TickerChannel}},
			exp:  []expected{{0, asset.Margin, spotBTCUSD, "ticker.BTC_USD"}, {0, asset.Margin, spotETHUSD, "ticker.ETH_USD"}},
		},
		{
			name: "margin tickers before spot tickers, each channel kept from the first",
			subs: subscription.List{{Enabled: true, Asset: asset.Margin, Channel: subscription.TickerChannel}, {Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel}},
			exp: []expected{
				{0, asset.Margin, spotBTCUSD, "ticker.BTC_USD"},
				{0, asset.Margin, spotETHUSD, "ticker.ETH_USD"},
				{1, asset.Spot, spotBTCUSDT, "ticker.BTC_USDT"},
				{1, asset.Spot, spotCROUSD, "ticker.CRO_USD"},
			},
		},
		{
			name: "books of every asset, 50 levels deep by default",
			subs: subscription.List{all(subscription.OrderbookChannel)},
			exp: []expected{
				{0, asset.Spot, spotBTCUSD, "book.BTC_USD.50"},
				{0, asset.Spot, spotBTCUSDT, "book.BTC_USDT.50"},
				{0, asset.Spot, spotCROUSD, "book.CRO_USD.50"},
				{0, asset.Spot, spotETHUSD, "book.ETH_USD.50"},
				{0, asset.Futures, futureBTCUSD, "book.BTCUSD-261030.50"},
				{0, asset.PerpetualSwap, perpBTCUSD, "book.BTCUSD-PERP.50"},
				{0, asset.PerpetualSwap, perpETHUSD, "book.ETHUSD-PERP.50"},
			},
		},
		{
			name: "one book for each instrument, the first subscribed whatever its depth",
			subs: subscription.List{
				{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel, Levels: 10, Interval: kline.TenMilliseconds},
				{Enabled: true, Asset: asset.Margin, Channel: subscription.OrderbookChannel, Levels: 50},
				{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel, Levels: 50, Interval: bookSnapshotInterval},
			},
			exp: []expected{{0, asset.Spot, spotBTCUSD, "book.BTC_USD.10"}, {0, asset.Spot, spotBTCUSDT, "book.BTC_USDT.10"}, {0, asset.Spot, spotCROUSD, "book.CRO_USD.10"}, {0, asset.Spot, spotETHUSD, "book.ETH_USD.10"}},
		},
		{
			name: "trades of every asset",
			subs: subscription.List{all(subscription.AllTradesChannel)},
			exp: []expected{
				{0, asset.Spot, spotBTCUSD, "trade.BTC_USD"},
				{0, asset.Spot, spotBTCUSDT, "trade.BTC_USDT"},
				{0, asset.Spot, spotCROUSD, "trade.CRO_USD"},
				{0, asset.Spot, spotETHUSD, "trade.ETH_USD"},
				{0, asset.Futures, futureBTCUSD, "trade.BTCUSD-261030"},
				{0, asset.PerpetualSwap, perpBTCUSD, "trade.BTCUSD-PERP"},
				{0, asset.PerpetualSwap, perpETHUSD, "trade.ETHUSD-PERP"},
			},
		},
		{
			name: "candles of every asset, and a day, two weeks and a month of spot",
			subs: subscription.List{
				all(subscription.CandlesChannel, func(s *subscription.Subscription) { s.Interval = kline.OneHour }),
				{Enabled: true, Asset: asset.Spot, Channel: subscription.CandlesChannel, Interval: kline.OneDay, Pairs: currency.Pairs{spotBTCUSD}},
				{Enabled: true, Asset: asset.Spot, Channel: subscription.CandlesChannel, Interval: kline.TwoWeek, Pairs: currency.Pairs{spotBTCUSD}},
				{Enabled: true, Asset: asset.Spot, Channel: subscription.CandlesChannel, Interval: kline.OneMonth, Pairs: currency.Pairs{spotBTCUSD}},
			},
			exp: []expected{
				{0, asset.Spot, spotBTCUSD, "candlestick.1h.BTC_USD"},
				{0, asset.Spot, spotBTCUSDT, "candlestick.1h.BTC_USDT"},
				{0, asset.Spot, spotCROUSD, "candlestick.1h.CRO_USD"},
				{0, asset.Spot, spotETHUSD, "candlestick.1h.ETH_USD"},
				{0, asset.Futures, futureBTCUSD, "candlestick.1h.BTCUSD-261030"},
				{0, asset.PerpetualSwap, perpBTCUSD, "candlestick.1h.BTCUSD-PERP"},
				{0, asset.PerpetualSwap, perpETHUSD, "candlestick.1h.ETHUSD-PERP"},
				{1, asset.Spot, spotBTCUSD, "candlestick.1D.BTC_USD"},
				{2, asset.Spot, spotBTCUSD, "candlestick.14D.BTC_USD"},
				{3, asset.Spot, spotBTCUSD, "candlestick.1M.BTC_USD"},
			},
		},
		{
			name: "mark prices of every asset",
			subs: subscription.List{all(MarkPriceChannel)},
			exp: []expected{
				{0, asset.Spot, spotBTCUSD, "mark.BTC_USD"},
				{0, asset.Spot, spotBTCUSDT, "mark.BTC_USDT"},
				{0, asset.Spot, spotCROUSD, "mark.CRO_USD"},
				{0, asset.Spot, spotETHUSD, "mark.ETH_USD"},
				{0, asset.Futures, futureBTCUSD, "mark.BTCUSD-261030"},
				{0, asset.PerpetualSwap, perpBTCUSD, "mark.BTCUSD-PERP"},
				{0, asset.PerpetualSwap, perpETHUSD, "mark.ETHUSD-PERP"},
			},
		},
		{
			name: "the underlying indices of the derivatives, each once",
			subs: subscription.List{all(IndexPriceChannel)},
			exp:  []expected{{0, asset.Futures, futureBTCUSD, "index.BTCUSD-INDEX"}, {0, asset.PerpetualSwap, perpETHUSD, "index.ETHUSD-INDEX"}},
		},
		{
			name: "funding rates of the perpetual swaps",
			subs: subscription.List{all(FundingRateChannel), all(EstimatedFundingRateChannel)},
			exp: []expected{
				{0, asset.PerpetualSwap, perpBTCUSD, "funding.BTCUSD-PERP"},
				{0, asset.PerpetualSwap, perpETHUSD, "funding.ETHUSD-PERP"},
				{1, asset.PerpetualSwap, perpBTCUSD, "estimatedfunding.BTCUSD-PERP"},
				{1, asset.PerpetualSwap, perpETHUSD, "estimatedfunding.ETHUSD-PERP"},
			},
		},
		{
			name: "settlement prices of the futures",
			subs: subscription.List{all(SettlementPriceChannel)},
			exp:  []expected{{0, asset.Futures, futureBTCUSD, "settlement.BTCUSD-261030"}},
		},
		{
			name: "market data alongside a user channel, which the user connection carries",
			subs: subscription.List{{Enabled: true, Asset: asset.Margin, Channel: subscription.TickerChannel}, {Enabled: true, Channel: "user.balance", Authenticated: true}},
			exp:  []expected{{0, asset.Margin, spotBTCUSD, "ticker.BTC_USD"}, {0, asset.Margin, spotETHUSD, "ticker.ETH_USD"}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newTestExchange(t)
			ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
			ex.Features.Subscriptions = tc.subs
			got, err := ex.generateMarketDataSubscriptions()
			require.NoError(t, err, "generateMarketDataSubscriptions must not error")
			exp := make(subscription.List, len(tc.exp))
			for i, x := range tc.exp {
				exp[i] = expandedSubscription(tc.subs[x.sub], x.asset, x.pair, x.channel)
			}
			testsubs.EqualLists(t, exp, got)
		})
	}

	for _, tc := range []struct {
		name string
		sub  *subscription.Subscription
		err  error
	}{
		{"a channel Crypto.com does not serve", all("nope"), subscription.ErrNotSupported},
		{"a market data channel without an asset", &subscription.Subscription{Enabled: true, Channel: subscription.TickerChannel}, subscription.ErrNotSupported},
		{"the funding rates of spot", &subscription.Subscription{Enabled: true, Asset: asset.Spot, Channel: FundingRateChannel}, subscription.ErrNotSupported},
		{"the estimated funding rates of futures", &subscription.Subscription{Enabled: true, Asset: asset.Futures, Channel: EstimatedFundingRateChannel}, subscription.ErrNotSupported},
		{"the underlying indices of margin", &subscription.Subscription{Enabled: true, Asset: asset.Margin, Channel: IndexPriceChannel}, subscription.ErrNotSupported},
		{"the settlement prices of perpetual swaps", &subscription.Subscription{Enabled: true, Asset: asset.PerpetualSwap, Channel: SettlementPriceChannel}, subscription.ErrNotSupported},
		{"books of a depth Crypto.com does not serve", all(subscription.OrderbookChannel, func(s *subscription.Subscription) { s.Levels = 20 }), subscription.ErrInvalidLevel},
		{"books updated at an interval Crypto.com does not serve", all(subscription.OrderbookChannel, func(s *subscription.Subscription) { s.Interval = kline.ThousandMilliseconds }), subscription.ErrInvalidInterval},
		{"candles of an interval Crypto.com does not serve", all(subscription.CandlesChannel, func(s *subscription.Subscription) { s.Interval = kline.ThreeMin }), kline.ErrUnsupportedInterval},
		{"candles without an interval", all(subscription.CandlesChannel), kline.ErrUnsupportedInterval},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newTestExchange(t)
			ex.Features.Subscriptions = subscription.List{tc.sub}
			_, err := ex.generateMarketDataSubscriptions()
			assert.ErrorIs(t, err, tc.err, "generateMarketDataSubscriptions should refuse the subscription")
		})
	}

	t.Run("the default subscriptions", func(t *testing.T) {
		t.Parallel()
		ex := newTestExchange(t)
		got, err := ex.generateMarketDataSubscriptions()
		require.NoError(t, err, "generateMarketDataSubscriptions must not error for the default subscriptions")
		channel := map[string]func(instrument string) string{
			subscription.TickerChannel:    func(instrument string) string { return "ticker." + instrument },
			subscription.OrderbookChannel: func(instrument string) string { return "book." + instrument + ".50" },
			subscription.AllTradesChannel: func(instrument string) string { return "trade." + instrument },
			subscription.CandlesChannel:   func(instrument string) string { return "candlestick.1m." + instrument },
		}
		instruments := []instrumentAsset{
			{Asset: asset.Spot, Pair: spotBTCUSD},
			{Asset: asset.Spot, Pair: spotBTCUSDT},
			{Asset: asset.Spot, Pair: spotCROUSD},
			{Asset: asset.Spot, Pair: spotETHUSD},
			{Asset: asset.Futures, Pair: futureBTCUSD},
			{Asset: asset.PerpetualSwap, Pair: perpBTCUSD},
			{Asset: asset.PerpetualSwap, Pair: perpETHUSD},
		}
		exp := make(subscription.List, 0, len(defaultSubscriptions)*len(instruments))
		for _, s := range defaultSubscriptions {
			if s.Authenticated {
				// The market data connections carry no user channel, whose defaults TestGenerateUserSubscriptions covers
				continue
			}
			require.Containsf(t, channel, s.Channel, "the %s default subscription must be expected", s.Channel)
			for _, a := range instruments {
				exp = append(exp, expandedSubscription(s, a.Asset, a.Pair, channel[s.Channel](a.Pair.String())))
			}
		}
		testsubs.EqualLists(t, exp, got)
	})
}

func TestChannelServesAsset(t *testing.T) { //nolint:tparallel // the cases run in order before the later calls of the same function
	t.Parallel()
	everyAsset := []asset.Item{asset.Spot, asset.Margin, asset.PerpetualSwap, asset.Futures}
	// The cases run in order before the later calls of the same function, so each also checks that a call keeps
	// nothing from the last
	for channel, served := range map[string][]asset.Item{
		subscription.TickerChannel:    everyAsset,
		subscription.OrderbookChannel: everyAsset,
		subscription.AllTradesChannel: everyAsset,
		subscription.CandlesChannel:   everyAsset,
		MarkPriceChannel:              everyAsset,
		IndexPriceChannel:             {asset.PerpetualSwap, asset.Futures},
		FundingRateChannel:            {asset.PerpetualSwap},
		EstimatedFundingRateChannel:   {asset.PerpetualSwap},
		SettlementPriceChannel:        {asset.Futures},
	} {
		t.Run(channel, func(t *testing.T) {
			for _, a := range everyAsset {
				t.Run(a.String(), func(t *testing.T) {
					t.Parallel()
					serves := slices.Contains(served, a)
					got, err := channelServesAsset(&subscription.Subscription{Channel: channel, Asset: asset.All}, a)
					require.NoErrorf(t, err, "channelServesAsset must not error for a %s subscription of every asset", channel)
					assert.Equalf(t, serves, got, "channelServesAsset should report whether the %s channel serves %s", channel, a)
					got, err = channelServesAsset(&subscription.Subscription{Channel: channel, Asset: a}, a)
					if serves {
						require.NoErrorf(t, err, "channelServesAsset must not error for a %s subscription of %s", channel, a)
						assert.Truef(t, got, "channelServesAsset should report the %s channel serves %s", channel, a)
						return
					}
					assert.ErrorIsf(t, err, subscription.ErrNotSupported, "channelServesAsset should refuse a %s subscription of %s", channel, a)
				})
			}
		})
	}
	_, err := channelServesAsset(&subscription.Subscription{Channel: "nope", Asset: asset.All}, asset.Spot)
	assert.ErrorIs(t, err, subscription.ErrNotSupported, "channelServesAsset should refuse a channel Crypto.com does not serve")
}

func TestQualifiedChannel(t *testing.T) { //nolint:tparallel // the cases run in order on e, whose pair formats qualifiedChannel reads
	t.Parallel()

	// The cases run in order on e, whose pair formats qualifiedChannel reads, before the refusals below call it too
	for _, tc := range []struct {
		sub  *subscription.Subscription
		a    asset.Item
		pair currency.Pair
		exp  string
	}{
		{&subscription.Subscription{Channel: subscription.TickerChannel}, asset.Spot, spotBTCUSD, "ticker.BTC_USD"},
		{&subscription.Subscription{Channel: subscription.OrderbookChannel}, asset.Margin, spotETHUSD, "book.ETH_USD.50"},
		{&subscription.Subscription{Channel: subscription.OrderbookChannel, Levels: 10, Interval: kline.TenMilliseconds}, asset.PerpetualSwap, perpBTCUSD, "book.BTCUSD-PERP.10"},
		{&subscription.Subscription{Channel: subscription.AllTradesChannel}, asset.Futures, futureBTCUSD, "trade.BTCUSD-261030"},
		{&subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.SevenDay}, asset.Spot, spotBTCUSD, "candlestick.7D.BTC_USD"},
		{&subscription.Subscription{Channel: IndexPriceChannel}, asset.PerpetualSwap, perpETHUSD, "index.ETHUSD-INDEX"},
		{&subscription.Subscription{Channel: IndexPriceChannel}, asset.Futures, futureBTCUSD, "index.BTCUSD-INDEX"},
		{&subscription.Subscription{Channel: IndexPriceChannel}, asset.PerpetualSwap, currency.NewPairWithDelimiter("SYNTH-USD", "PERP", currency.DashDelimiter), "index.SYNTH-USD-INDEX"},
		{&subscription.Subscription{Channel: MarkPriceChannel}, asset.Spot, spotBTCUSD, "mark.BTC_USD"},
		{&subscription.Subscription{Channel: SettlementPriceChannel}, asset.Futures, futureBTCUSD, "settlement.BTCUSD-261030"},
		{&subscription.Subscription{Channel: FundingRateChannel}, asset.PerpetualSwap, perpBTCUSD, "funding.BTCUSD-PERP"},
		{&subscription.Subscription{Channel: EstimatedFundingRateChannel}, asset.PerpetualSwap, perpBTCUSD, "estimatedfunding.BTCUSD-PERP"},
	} {
		t.Run(tc.exp, func(t *testing.T) {
			got, err := e.qualifiedChannel(tc.sub, tc.a, tc.pair)
			require.NoErrorf(t, err, "qualifiedChannel must not error for the %s channel of %s %s", tc.sub.Channel, tc.a, tc.pair)
			assert.Equalf(t, tc.exp, got, "qualifiedChannel should name the %s channel of %s %s", tc.sub.Channel, tc.a, tc.pair)
		})
	}
	// The cases run in order on e, whose pair formats qualifiedChannel reads
	for _, tc := range []struct {
		name string
		sub  *subscription.Subscription
		a    asset.Item
		err  error
	}{
		{"a channel Crypto.com does not serve", &subscription.Subscription{Channel: "nope"}, asset.Spot, subscription.ErrNotSupported},
		{"the underlying index of a spot pair", &subscription.Subscription{Channel: IndexPriceChannel}, asset.Spot, errNoUnderlyingIndex},
		{"an asset Crypto.com does not trade", &subscription.Subscription{Channel: subscription.TickerChannel}, asset.Options, asset.ErrNotSupported},
		{"a book depth Crypto.com does not serve", &subscription.Subscription{Channel: subscription.OrderbookChannel, Levels: 20}, asset.Spot, subscription.ErrInvalidLevel},
		{"a book interval Crypto.com does not serve", &subscription.Subscription{Channel: subscription.OrderbookChannel, Interval: kline.OneMin}, asset.Spot, subscription.ErrInvalidInterval},
		{"a candle interval Crypto.com does not serve", &subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.ThreeMin}, asset.Spot, kline.ErrUnsupportedInterval},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := e.qualifiedChannel(tc.sub, tc.a, spotBTCUSD)
			assert.ErrorIsf(t, err, tc.err, "qualifiedChannel should refuse the %s channel of %s with levels %d and interval %s", tc.sub.Channel, tc.a, tc.sub.Levels, tc.sub.Interval)
		})
	}
}

func TestBookDepth(t *testing.T) { //nolint:tparallel // the cases run in order before the later calls of the same function
	t.Parallel()
	// The cases run in order before the later calls of the same function, so each also checks that a call keeps
	// nothing from the last
	for levels, exp := range map[int]int{0: 50, 10: 10, 50: 50} {
		t.Run(strconv.Itoa(levels), func(t *testing.T) {
			got, err := bookDepth(&subscription.Subscription{Levels: levels})
			require.NoErrorf(t, err, "bookDepth must not error for %d levels", levels)
			assert.Equalf(t, exp, got, "bookDepth should give the depth of a subscription to %d levels", levels)
		})
	}
	_, err := bookDepth(&subscription.Subscription{Levels: 20})
	assert.ErrorIs(t, err, subscription.ErrInvalidLevel, "bookDepth should refuse a depth Crypto.com does not serve")
}

func TestBookUpdates(t *testing.T) { //nolint:tparallel // the cases run in order before the later calls of the same function
	t.Parallel()
	// The cases run in order before the later calls of the same function, so each also checks that a call keeps
	// nothing from the last
	for _, tc := range []struct {
		interval         kline.Interval
		subscriptionType string
		frequency        uint64
	}{
		{0, BookSubscriptionTypeSnapshotAndUpdate, 100},
		{kline.TenMilliseconds, BookSubscriptionTypeSnapshotAndUpdate, 10},
		{kline.HundredMilliseconds, BookSubscriptionTypeSnapshotAndUpdate, 100},
		{bookSnapshotInterval, BookSubscriptionTypeSnapshot, 500},
	} {
		t.Run(tc.interval.Short(), func(t *testing.T) {
			subscriptionType, frequency, err := bookUpdates(&subscription.Subscription{Interval: tc.interval})
			require.NoErrorf(t, err, "bookUpdates must not error for an interval of %s", tc.interval)
			assert.Equalf(t, tc.subscriptionType, subscriptionType, "bookUpdates should give the subscription type of an interval of %s", tc.interval)
			assert.Equalf(t, tc.frequency, frequency, "bookUpdates should give the update frequency of an interval of %s", tc.interval)
		})
	}
	_, _, err := bookUpdates(&subscription.Subscription{Interval: kline.ThousandMilliseconds})
	assert.ErrorIs(t, err, subscription.ErrInvalidInterval, "bookUpdates should refuse an interval Crypto.com does not update books at")

	// A configured book subscription names its snapshots as 500ms
	var configured kline.Interval
	require.NoError(t, configured.UnmarshalText([]byte("500ms")), "UnmarshalText must not error")
	assert.Equal(t, bookSnapshotInterval, configured, "a configured interval of 500ms should name book snapshots")
}

// errScripted is the error a scriptedConnection fails with where a test has it fail
var errScripted = errors.New("scripted failure")

// scriptedConnection stands in for a websocket connection in a test calling a function that takes one: it answers each
// request with the next of the answers a test scripts, the last answering any after, records what it is dialled with,
// sent and given to route, and holds the subscriptions stored against it
type scriptedConnection struct {
	websocket.Connection
	mu            sync.Mutex
	answers       []scriptedAnswer
	dialErr       error
	sendErr       error
	record        scriptedRecord
	subscriptions *subscription.Store
}

// scriptedAnswer is what a scriptedConnection answers a request with: the responses received, and the error the wait
// for them ends with
type scriptedAnswer struct {
	responses []string
	err       error
}

// scriptedRecord is what a scriptedConnection was dialled with, sent and given to route, and how often its
// subscriptions were looked up
type scriptedRecord struct {
	dials    []scriptedDial
	sent     []any
	requests []scriptedRequest
	routed   []scriptedRouted
	lookups  int
}

// scriptedDial is what a scriptedConnection was dialled with
type scriptedDial struct {
	dialer  *gws.Dialer
	headers http.Header
	values  url.Values
}

// scriptedRequest is a request sent over a scriptedConnection, with the signature its answers are matched by, how many
// answers it waits for and the inspector reading each
type scriptedRequest struct {
	signature any
	request   any
	expected  int
	inspector websocket.Inspector
}

// scriptedRouted is data a scriptedConnection was given to route to the request it answers
type scriptedRouted struct {
	signature any
	data      string
}

// newScriptedConnection returns a scriptedConnection answering requests with answers in turn
func newScriptedConnection(answers ...scriptedAnswer) *scriptedConnection {
	return &scriptedConnection{answers: answers, subscriptions: subscription.NewStore()}
}

// Dial records what the connection is dialled with, and fails with the error scripted for dialling
func (c *scriptedConnection) Dial(_ context.Context, dialer *gws.Dialer, headers http.Header, values url.Values) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.record.dials = append(c.record.dials, scriptedDial{dialer: dialer, headers: headers, values: values})
	return c.dialErr
}

// SendJSONMessage records a message sent, and fails with the error scripted for sending
func (c *scriptedConnection) SendJSONMessage(_ context.Context, _ request.EndpointLimit, payload any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.record.sent = append(c.record.sent, payload)
	return c.sendErr
}

// SendMessageReturnResponse records a request and returns the first response of the next answer
func (c *scriptedConnection) SendMessageReturnResponse(_ context.Context, _ request.EndpointLimit, signature, req any) ([]byte, error) {
	responses, err := c.answer(scriptedRequest{signature: signature, request: req, expected: 1})
	if len(responses) == 0 {
		return nil, err
	}
	return responses[0], err
}

// SendMessageReturnResponsesWithInspector records a request and returns the responses of the next answer
func (c *scriptedConnection) SendMessageReturnResponsesWithInspector(_ context.Context, _ request.EndpointLimit, signature, req any, expected int, inspector websocket.Inspector) ([][]byte, error) {
	return c.answer(scriptedRequest{signature: signature, request: req, expected: expected, inspector: inspector})
}

// answer records a request and returns the responses and the error of the next answer, the last answering any after
func (c *scriptedConnection) answer(r scriptedRequest) ([][]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.record.requests = append(c.record.requests, r)
	if len(c.answers) == 0 {
		return nil, nil
	}
	a := c.answers[0]
	if len(c.answers) > 1 {
		c.answers = c.answers[1:]
	}
	responses := make([][]byte, len(a.responses))
	for i := range a.responses {
		responses[i] = []byte(a.responses[i])
	}
	return responses, a.err
}

// IncomingWithData records data given to route to the request it answers
func (c *scriptedConnection) IncomingWithData(signature any, data []byte) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.record.routed = append(c.record.routed, scriptedRouted{signature: signature, data: string(data)})
	return true
}

// Subscriptions returns the subscriptions stored against the connection, counting the lookups
func (c *scriptedConnection) Subscriptions() *subscription.Store {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.record.lookups++
	return c.subscriptions
}

// GetURL names the connection, as errors about it do
func (*scriptedConnection) GetURL() string {
	return "scripted"
}

// recorded returns what the connection has recorded so far
func (c *scriptedConnection) recorded() scriptedRecord {
	c.mu.Lock()
	defer c.mu.Unlock()
	return scriptedRecord{
		dials:    slices.Clone(c.record.dials),
		sent:     slices.Clone(c.record.sent),
		requests: slices.Clone(c.record.requests),
		routed:   slices.Clone(c.record.routed),
		lookups:  c.record.lookups,
	}
}

// channelPush returns the channel's push a frame's result holds, as wsHandleChannelData hands it to the channel's handler
func channelPush(t *testing.T, frame []byte) *websocketChannelPush {
	t.Helper()
	result, _, _, err := jsonparser.Get(frame, "result")
	require.NoError(t, err, "the frame must hold a result")
	var push websocketChannelPush
	require.NoError(t, json.Unmarshal(result, &push), "the frame's result must decode as a channel's push")
	return &push
}

// fillDataHandler fills an exchange's data handler, so that it relays nothing more, and returns the error it refuses
// more with
func fillDataHandler(t *testing.T, ex *Exchange) error {
	t.Helper()
	for {
		if err := ex.Websocket.DataHandler.Send(t.Context(), nil); err != nil {
			full := errors.Unwrap(err)
			require.Error(t, full, "the data handler must refuse more with an error of its own")
			return full
		}
	}
}

func TestWebsocketAPI(t *testing.T) {
	t.Parallel()
	ex, other := new(Exchange), new(Exchange)
	assert.Same(t, ex, ex.WebsocketAPI().e, "WebsocketAPI should call the methods over the exchange it is taken from")
	assert.Same(t, other, other.WebsocketAPI().e, "WebsocketAPI should call the methods over the exchange it is taken from")
}

func TestWebsocketResponseErr(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		msg  string
		exp  *APIError
	}{
		{"no error", `{"id":1,"method":"subscribe","code":0,"message":"ignored without a code"}`, nil},
		{"no code", `{"id":1,"method":"subscribe"}`, nil},
		{"an error", `{"id":1,"method":"subscribe","code":40001,"message":"BAD_REQUEST"}`, &APIError{Code: 40001, Message: "BAD_REQUEST"}},
		{"an error with a quoted code", `{"id":1,"method":"private/create-order","code":"306","message":"INSUFFICIENT_AVAILABLE_BALANCE"}`, &APIError{Code: 306, Message: "INSUFFICIENT_AVAILABLE_BALANCE"}},
		{"an error without a message", `{"id":1,"method":"subscribe","code":50001}`, &APIError{Code: 50001}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var resp websocketResponse
			require.NoErrorf(t, json.Unmarshal([]byte(tc.msg), &resp), "Unmarshal must not error for %s", tc.msg)
			err := resp.err()
			if tc.exp == nil {
				assert.NoErrorf(t, err, "err should report no error for %s", tc.msg)
				return
			}
			var apiErr *APIError
			require.ErrorAsf(t, err, &apiErr, "err must report the error %s reports", tc.msg)
			assert.Equalf(t, tc.exp, apiErr, "err should report the code and message of %s", tc.msg)
		})
	}
}

func TestSetupWebsocket(t *testing.T) {
	t.Parallel()
	// setUpAgain returns a function setting an exchange's websocket up afresh, its connections pointed at venue in mock
	// mode
	setUpAgain := func(t *testing.T, venue *websocketVenue) func(*Exchange) {
		t.Helper()
		return func(ex *Exchange) {
			ex.Websocket = sharedtestvalues.NewTestWebsocket()
			require.NoError(t, ex.setupWebsocket(ex.Config), "setupWebsocket must not error")
			if mockTests {
				require.NoError(t, ex.Websocket.SetAllConnectionURLs(venue.url), "SetAllConnectionURLs must not error")
			}
		}
	}

	t.Run("market data connections", func(t *testing.T) {
		t.Parallel()
		venue := newMarketDataVenue(t)
		ex := newTestExchange(t)
		ex.Features.Subscriptions = subscription.List{
			{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: currency.Pairs{spotBTCUSD}},
			{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel, Levels: 10, Pairs: currency.Pairs{spotBTCUSD}},
		}
		setUpAgain(t, venue)(ex)
		ex.Websocket.SetCanUseAuthenticatedEndpoints(false)
		t.Cleanup(func() {
			// The test disables and shuts the websocket down itself, unless it stops first
			if ex.Websocket.IsEnabled() {
				assert.NoError(t, ex.Websocket.Disable(), "Disable should not error")
			}
			if err := ex.Websocket.Shutdown(); err != nil {
				assert.ErrorIs(t, err, websocket.ErrNotConnected, "Shutdown should only fail when the websocket has shut down already")
			}
		})
		t.Cleanup(venue.closing)
		require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must connect the market data connection")
		_, err := ex.Websocket.GetConnection(websocketMarketDataConnection)
		require.NoError(t, err, "setupWebsocket must set up the market data connection")
		for _, filter := range []string{websocketUserConnection, websocketTradeConnection} {
			_, err := ex.Websocket.GetConnection(filter)
			assert.ErrorIsf(t, err, websocket.ErrNotConnected, "setupWebsocket should set up the %s connection, left undialled without authentication", filter)
		}
		_, err = ex.Websocket.GetConnection("nope")
		assert.ErrorIs(t, err, websocket.ErrRequestRouteNotFound, "setupWebsocket should set up no other connection")
		for _, a := range []asset.Item{asset.Spot, asset.Margin} {
			require.Eventuallyf(t, func() bool {
				_, err := orderbook.Get(ex.Name, spotBTCUSD, a)
				return err == nil
			}, 30*time.Second, 10*time.Millisecond, "the %s book must be loaded", a)
		}

		require.NoError(t, ex.Websocket.Disable(), "Disable must not error")
		if err := ex.Websocket.Shutdown(); err != nil {
			require.ErrorIs(t, err, websocket.ErrNotConnected, "Shutdown must only fail when the connection monitor has shut the websocket down first")
		}
		for _, a := range []asset.Item{asset.Spot, asset.Margin} {
			_, err := orderbook.Get(ex.Name, spotBTCUSD, a)
			assert.ErrorIsf(t, err, orderbook.ErrOrderbookInvalid, "setupWebsocket should have the %s book invalidated once the websocket shuts down", a)
		}
	})

	t.Run("user channels and trade connections", func(t *testing.T) {
		t.Parallel()
		venue := newWebsocketVenue(t)
		ex, err := newUserDataTestExchange(t, venue, subscription.List{{Enabled: true, Channel: subscription.MyOrdersChannel, Authenticated: true}}, func(ex *Exchange) {
			ex.Config.WebsocketResponseMaxLimit = testResponseMaxLimit
		}, setUpAgain(t, venue))
		require.NoError(t, err, "Connect must connect the user and trade connections")
		for _, filter := range []string{websocketUserConnection, websocketTradeConnection} {
			_, err := ex.Websocket.GetConnection(filter)
			assert.NoErrorf(t, err, "setupWebsocket should set up the %s connection to authenticate", filter)
		}
		_, err = ex.Websocket.GetConnection(websocketMarketDataConnection)
		assert.ErrorIs(t, err, websocket.ErrNotConnected, "setupWebsocket should leave the market data connection undialled without market data")
		assert.Len(t, venue.received(websocketMethodAuth), 2, "setupWebsocket should have the user and trade connections authenticate")

		venue.loseAnswers("private/get-cancel-on-disconnect", 1)
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()
		err = ex.SendWebsocketRequest(ctx, "private/get-cancel-on-disconnect", nil, nil)
		assert.ErrorIs(t, err, websocket.ErrSignatureTimeout, "setupWebsocket should have the trade connection wait for an answer as long as the configuration says")
	})

	t.Run("connection setups", func(t *testing.T) {
		t.Parallel()
		urls := []string{"wss://market.example.invalid/v1/market", "wss://user.example.invalid/v1/user", "wss://trade.example.invalid/v1/user"}
		for _, tc := range []struct {
			name                 string
			tradeFeed, fillsFeed bool
		}{
			{name: "the trade feed enabled", tradeFeed: true},
			{name: "the fills feed enabled", fillsFeed: true},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newTestExchange(t)
				for i, endpoint := range []exchange.URL{exchange.WebsocketSpot, exchange.WebsocketPrivate, exchange.WebsocketTrade} {
					require.NoErrorf(t, ex.API.Endpoints.SetRunningURL(endpoint.String(), urls[i]), "SetRunningURL must not error for %s", endpoint)
				}
				ex.SetTradeFeedStatus(tc.tradeFeed)
				ex.SetFillsFeedStatus(tc.fillsFeed)
				ex.Websocket = sharedtestvalues.NewTestWebsocket()
				require.NoError(t, ex.setupWebsocket(ex.Config), "setupWebsocket must not error")

				got, err := ex.Websocket.GetConfiguredWebsocketURLs()
				require.NoError(t, err, "GetConfiguredWebsocketURLs must not error")
				assert.Equal(t, urls, got, "setupWebsocket should set up a connection to each of the market data, user and trade URLs")
				assert.Equal(t, websocketMaxSubscriptionsPerConnection, ex.Websocket.MaxSubscriptionsPerConnection, "setupWebsocket should have a connection hold as many subscriptions as Crypto.com allows one")

				require.NoError(t, ex.Websocket.Trade.Update(false, trade.Data{TID: "1"}), "Trade.Update must not error")
				fillsErr := ex.Websocket.Fills.Update(fill.Data{TradeID: "1"})
				var exp []any
				if tc.tradeFeed {
					exp = append(exp, []trade.Data{{TID: "1"}})
				}
				if tc.fillsFeed {
					require.NoError(t, fillsErr, "Fills.Update must not error")
					exp = append(exp, []fill.Data{{TradeID: "1"}})
				} else {
					assert.ErrorIs(t, fillsErr, fill.ErrFeedDisabled, "setupWebsocket should leave the fills feed disabled as the exchange's is")
				}
				assert.Equal(t, exp, relayedData(ex), "setupWebsocket should relay trades and fills as the exchange's feeds say")
			})
		}
	})

	t.Run("refusals", func(t *testing.T) {
		t.Parallel()
		// withoutURL returns the configuration of an exchange whose endpoints leave a websocket URL out
		withoutURL := func(missing exchange.URL) func(*testing.T, *Exchange) *config.Exchange {
			return func(t *testing.T, ex *Exchange) *config.Exchange {
				t.Helper()
				endpoints := map[exchange.URL]string{
					exchange.RestSpot:         apiURL,
					exchange.WebsocketSpot:    websocketMarketDataURL,
					exchange.WebsocketPrivate: websocketUserURL,
					exchange.WebsocketTrade:   websocketUserURL,
				}
				delete(endpoints, missing)
				ex.API.Endpoints = ex.NewEndpoints()
				require.NoError(t, ex.API.Endpoints.SetDefaultEndpoints(endpoints), "SetDefaultEndpoints must not error")
				return ex.Config
			}
		}
		for _, tc := range []struct {
			name   string
			config func(*testing.T, *Exchange) *config.Exchange
			err    error
		}{
			{"no configuration", func(*testing.T, *Exchange) *config.Exchange { return nil }, common.ErrNilPointer},
			{"a configuration without features", func(*testing.T, *Exchange) *config.Exchange { return &config.Exchange{Name: "Cryptodotcom"} }, common.ErrNilPointer},
			{"no market data URL", withoutURL(exchange.WebsocketSpot), exchange.ErrEndpointPathNotFound},
			{"no user URL", withoutURL(exchange.WebsocketPrivate), exchange.ErrEndpointPathNotFound},
			{"no trade URL", withoutURL(exchange.WebsocketTrade), exchange.ErrEndpointPathNotFound},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newTestExchange(t)
				cfg := tc.config(t, ex)
				ex.Websocket = sharedtestvalues.NewTestWebsocket()
				assert.ErrorIs(t, ex.setupWebsocket(cfg), tc.err, "setupWebsocket should refuse to set the websocket up")
				urls, err := ex.Websocket.GetConfiguredWebsocketURLs()
				require.NoError(t, err, "GetConfiguredWebsocketURLs must not error")
				assert.Empty(t, urls, "setupWebsocket should set up no connection once it refuses")
			})
		}

		t.Run("a connection the manager refuses", func(t *testing.T) {
			t.Parallel()
			ex := newTestExchange(t)
			ex.Websocket = sharedtestvalues.NewTestWebsocket()
			// The manager sets up no connection without a channel to alert it to traffic on its connections
			ex.Websocket.TrafficAlert = nil
			assert.ErrorContains(t, ex.setupWebsocket(ex.Config), "traffic alert is nil", "setupWebsocket should report a connection the manager refuses")
			urls, err := ex.Websocket.GetConfiguredWebsocketURLs()
			require.NoError(t, err, "GetConfiguredWebsocketURLs must not error")
			assert.Empty(t, urls, "setupWebsocket should stop at the first connection the manager refuses")
		})
	})
}

func TestWebsocketConnect(t *testing.T) {
	t.Parallel()
	t.Run("a connection dialled", func(t *testing.T) {
		t.Parallel()
		venue := newMarketDataVenue(t)
		ex := newTestExchange(t)
		if mockTests {
			require.NoError(t, ex.Websocket.SetAllConnectionURLs(venue.url), "SetAllConnectionURLs must not error")
		}
		t.Cleanup(venue.closing)
		conn, err := ex.Websocket.CreateTestConnection(websocketMarketDataConnection)
		require.NoError(t, err, "CreateTestConnection must not error")
		ex.websocketConnectDelay = 100 * time.Millisecond
		start := time.Now()
		require.NoError(t, ex.wsConnect(t.Context(), conn), "wsConnect must dial the connection")
		t.Cleanup(func() { assert.NoError(t, conn.Shutdown(), "Shutdown should not error") })
		assert.GreaterOrEqual(t, time.Since(start), ex.websocketConnectDelay, "wsConnect should wait the connect delay once it has dialled")
		connected, ok := conn.(interface{ IsConnected() bool })
		require.True(t, ok, "the connection must report whether it is connected")
		assert.True(t, connected.IsConnected(), "wsConnect should leave the connection connected")
	})

	for _, tc := range []struct {
		name    string
		delay   time.Duration
		dialErr error
		// ended ends the context before the call, and endAfter that long into it
		ended    bool
		endAfter time.Duration
		elapsed  time.Duration
		err      error
	}{
		{name: "no connect delay"},
		{name: "a negative connect delay", delay: -time.Second},
		{name: "the connect delay waited", delay: time.Second, elapsed: time.Second},
		{name: "a connection that cannot be dialled", delay: time.Second, dialErr: errScripted, err: errScripted},
		{name: "a context ended while waiting", delay: time.Second, endAfter: 250 * time.Millisecond, elapsed: 250 * time.Millisecond, err: context.Canceled},
		{name: "a context ended before", delay: time.Second, ended: true, err: context.Canceled},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := &Exchange{websocketConnectDelay: tc.delay}
			// The wait runs on a synctest bubble's clock, which a stall of the test cannot move
			synctest.Test(t, func(t *testing.T) {
				conn := newScriptedConnection()
				conn.dialErr = tc.dialErr
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				if tc.ended {
					cancel()
				}
				if tc.endAfter != 0 {
					time.AfterFunc(tc.endAfter, cancel)
				}
				start := time.Now()
				assert.ErrorIs(t, ex.wsConnect(ctx, conn), tc.err, "wsConnect should return the expected error")
				assert.Equal(t, tc.elapsed, time.Since(start), "wsConnect should return after waiting as long as expected")
				exp := scriptedRecord{dials: []scriptedDial{{dialer: &gws.Dialer{}, headers: http.Header{}}}}
				assert.Equal(t, exp, conn.recorded(), "wsConnect should dial the connection once, with no headers or params")
			})
		})
	}
}

func TestNextWebsocketRequestID(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		last int64
		exp  []int64
	}{
		{name: "counting up from 1", exp: []int64{1, 2, 3}},
		{name: "counting up from the last ID", last: 41, exp: []int64{42, 43}},
		{name: "up to the largest ID", last: math.MaxInt64 - 1, exp: []int64{math.MaxInt64}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			ex.websocketRequestID.Store(tc.last)
			got := make([]int64, len(tc.exp))
			for i := range got {
				got[i] = ex.nextWebsocketRequestID()
			}
			assert.Equal(t, tc.exp, got, "nextWebsocketRequestID should count up from the last ID")
		})
	}

	t.Run("an ID of its own for each of concurrent requests", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		const callers, calls = 8, 500
		ids := make([][]int64, callers)
		var wg sync.WaitGroup
		for i := range ids {
			wg.Go(func() {
				for range calls {
					ids[i] = append(ids[i], ex.nextWebsocketRequestID())
				}
			})
		}
		wg.Wait()
		got := slices.Concat(ids...)
		slices.Sort(got)
		exp := make([]int64, callers*calls)
		for i := range exp {
			exp[i] = int64(i) + 1
		}
		assert.Equal(t, exp, got, "nextWebsocketRequestID should give each of concurrent requests an ID of its own")
	})
}

func TestWebsocketHandleData(t *testing.T) {
	t.Parallel()
	t.Run("a heartbeat answered over a connection", func(t *testing.T) {
		t.Parallel()
		venue := newMarketDataVenue(t)
		ex, err := newMarketDataTestExchange(t, venue, subscription.List{{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: currency.Pairs{spotBTCUSD}}})
		require.NoError(t, err, "Connect must not error")
		conn, err := ex.Websocket.GetConnection(websocketMarketDataConnection)
		require.NoError(t, err, "GetConnection must not error")
		require.NoError(t, ex.wsHandleData(t.Context(), conn, []byte(`{"id":1792891216162,"method":"public/heartbeat","code":0}`)), "wsHandleData must answer a heartbeat")
		if mockTests {
			require.Eventually(t, func() bool { return len(venue.received(websocketMethodRespondHeartbeat)) == 1 }, 5*time.Second, 10*time.Millisecond, "the heartbeat must be answered")
			assert.Equal(t, int64(1792891216162), venue.received(websocketMethodRespondHeartbeat)[0].ID, "the heartbeat's answer should carry its ID")
		}
	})

	ex := newTestExchange(t)
	acknowledgement := string(marketDataFrame(t, "ticker.BTC_USD", 0))
	push := marketDataFrame(t, "ticker.BTC_USD", 1)
	unidentified := string(jsonparser.Delete(bytes.Clone(push), "id"))
	unknownInstrument := func(id string) string {
		return `{"id":` + id + `,"method":"subscribe","code":0,"result":{"instrument_name":"NOPE_USD","subscription":"ticker.NOPE_USD","channel":"ticker","data":[]}}`
	}
	// The cases share ex and drain its data handler in order, so each also checks that a call keeps nothing from the last
	for _, tc := range []struct {
		name    string
		frame   string
		sendErr error
		// reply is the ID a heartbeat is answered with, and routed the ID of an answer routed to its request
		reply, routed int64
		relayed       int
		err           error
	}{
		{name: "a heartbeat answered with its ID", frame: `{"id":1792891216162,"method":"public/heartbeat","code":0}`, reply: 1792891216162},
		{name: "a heartbeat with a quoted ID", frame: `{"id":"1792891216163","method":"public/heartbeat","code":0}`, reply: 1792891216163},
		{name: "a heartbeat whose answer cannot be sent", frame: `{"id":1792891216164,"method":"public/heartbeat","code":0}`, sendErr: errScripted, reply: 1792891216164, err: errScripted},
		{name: "an answer routed to its request", frame: `{"id":5,"method":"private/cancel-order","code":0}`, routed: 5},
		{name: "an acknowledgement routed to its request", frame: `{"id":6,"method":"subscribe","code":0,"channel":"ticker.BTC_USD"}`, routed: 6},
		{name: "an acknowledgement carrying its channel's data routed and handled", frame: acknowledgement, routed: 1, relayed: 1},
		{name: "a push handled", frame: string(push), relayed: 1},
		{name: "a push without an ID handled", frame: unidentified, relayed: 1},
		{name: "an answer whose data cannot be handled routed and reported", frame: unknownInstrument("7"), routed: 7, err: currency.ErrPairNotFound},
		{name: "a push whose data cannot be handled reported", frame: unknownInstrument("-1"), err: currency.ErrPairNotFound},
		{name: "a push of another method dropped", frame: `{"id":-1,"method":"public/nope","code":0}`},
		{name: "a message naming neither a method nor an ID dropped", frame: `{"code":0}`},
		{name: "an unterminated method", frame: `{"id":1,"method":"public/heartbeat`, err: errMalformedWebsocketMessage},
		{name: "an ID neither a number nor a string", frame: `{"id":true,"method":"public/heartbeat"}`, err: errMalformedWebsocketMessage},
	} {
		t.Run(tc.name, func(t *testing.T) {
			conn := newScriptedConnection()
			conn.sendErr = tc.sendErr
			assert.ErrorIsf(t, ex.wsHandleData(t.Context(), conn, []byte(tc.frame)), tc.err, "wsHandleData should return the expected error for %s", tc.frame)
			var exp scriptedRecord
			if tc.reply != 0 {
				exp.sent = []any{&websocketRequest{ID: tc.reply, Method: websocketMethodRespondHeartbeat}}
			}
			if tc.routed != 0 {
				exp.routed = []scriptedRouted{{signature: tc.routed, data: tc.frame}}
			}
			assert.Equalf(t, exp, conn.recorded(), "wsHandleData should answer and route %s as expected", tc.frame)
			relayed := relayedData(ex)
			assert.Lenf(t, relayed, tc.relayed, "wsHandleData should hand the data of %s to its channel's handler", tc.frame)
			for _, data := range relayed {
				assert.IsTypef(t, []ticker.Price{}, data, "wsHandleData should relay the tickers %s carries", tc.frame)
			}
		})
	}
}

func TestDecodePushData(t *testing.T) {
	t.Parallel()
	t.Run("data decoded", func(t *testing.T) {
		t.Parallel()
		var prices []WebsocketIndexPrice
		require.NoError(t, decodePushData(&websocketChannelPush{Channel: IndexPriceChannel, Data: json.RawMessage(`[{"v":"80784.28","t":1792892978000},{"v":"80784.29","t":1792892979000}]`)}, &prices), "decodePushData must not error")
		exp := []WebsocketIndexPrice{
			{IndexPrice: 80784.28, Time: types.Time(time.UnixMilli(1792892978000))},
			{IndexPrice: 80784.29, Time: types.Time(time.UnixMilli(1792892979000))},
		}
		assert.Equal(t, exp, prices, "decodePushData should decode the push's data")
	})

	t.Run("null data", func(t *testing.T) {
		t.Parallel()
		var prices []WebsocketIndexPrice
		require.NoError(t, decodePushData(&websocketChannelPush{Channel: IndexPriceChannel, Data: json.RawMessage(`null`)}, &prices), "decodePushData must not error for null data")
		assert.Nil(t, prices, "decodePushData should decode nothing from null data")
	})

	for _, tc := range []struct{ name, data string }{
		{"data of another shape", `{"v":"80784.28"}`},
		{"data that is not JSON", `[{"v":`},
		{"no data", ``},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var prices []WebsocketIndexPrice
			err := decodePushData(&websocketChannelPush{Channel: IndexPriceChannel, Data: json.RawMessage(tc.data)}, &prices)
			assert.ErrorIsf(t, err, errMalformedWebsocketMessage, "decodePushData should refuse %s", tc.name)
			assert.ErrorContainsf(t, err, "index data", "decodePushData should name the channel of %s", tc.name)
		})
	}
}

// subscriptionTemplateContext holds what a subscription template is executed with, as ExpandTemplates fills it in
type subscriptionTemplateContext struct {
	S              *subscription.Subscription
	AssetPairs     map[asset.Item]currency.Pairs
	PairSeparator  string
	AssetSeparator string
}

func TestGetSubscriptionTemplate(t *testing.T) { //nolint:tparallel // the cases run in order on one exchange, whose pair formats the templates read
	t.Parallel()
	ex := newTestExchange(t)
	_, err := ex.GetSubscriptionTemplate(nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSubscriptionTemplate must refuse no subscription")

	user := func(channel string, a asset.Item) *subscription.Subscription {
		return &subscription.Subscription{Channel: channel, Asset: a, Authenticated: true}
	}
	// The cases run in order on ex, whose pair formats the templates read
	for _, tc := range []struct {
		name       string
		sub        *subscription.Subscription
		assetPairs map[asset.Item]currency.Pairs
		template   string
		exp        string
		err        error
	}{
		{
			name:       "a market data channel of each pair of each asset it serves",
			sub:        &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.All},
			assetPairs: map[asset.Item]currency.Pairs{asset.Spot: {spotBTCUSD, spotETHUSD}, asset.Margin: {spotBTCUSD}, asset.PerpetualSwap: {perpBTCUSD}},
			template:   "master.tmpl",
			exp:        "ticker.BTC_USD,ticker.ETH_USD,;ticker.BTC_USD,;ticker.BTCUSD-PERP,;",
		},
		{
			name:       "a market data channel left out of the assets it does not serve",
			sub:        &subscription.Subscription{Channel: FundingRateChannel, Asset: asset.All},
			assetPairs: map[asset.Item]currency.Pairs{asset.Spot: {spotBTCUSD}, asset.PerpetualSwap: {perpBTCUSD, perpETHUSD}},
			template:   "master.tmpl",
			exp:        ";funding.BTCUSD-PERP,funding.ETHUSD-PERP,;",
		},
		{
			name:       "a market data channel of an asset it does not serve",
			sub:        &subscription.Subscription{Channel: FundingRateChannel, Asset: asset.Spot},
			assetPairs: map[asset.Item]currency.Pairs{asset.Spot: {spotBTCUSD}},
			template:   "master.tmpl",
			err:        subscription.ErrNotSupported,
		},
		{
			name:       "a market data channel Crypto.com does not serve",
			sub:        &subscription.Subscription{Channel: "nope", Asset: asset.All},
			assetPairs: map[asset.Item]currency.Pairs{asset.Spot: {spotBTCUSD}},
			template:   "master.tmpl",
			err:        subscription.ErrNotSupported,
		},
		{
			name:       "a book depth Crypto.com does not serve",
			sub:        &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Levels: 20},
			assetPairs: map[asset.Item]currency.Pairs{asset.Spot: {spotBTCUSD}},
			template:   "master.tmpl",
			err:        subscription.ErrInvalidLevel,
		},
		{
			name:       "a user channel of each pair of an asset",
			sub:        user(subscription.MyOrdersChannel, asset.Spot),
			assetPairs: map[asset.Item]currency.Pairs{asset.Spot: {spotBTCUSD, spotETHUSD}},
			template:   "user.tmpl",
			exp:        "user.order.BTC_USD,user.order.ETH_USD,;",
		},
		{
			name:       "a user channel of every instrument",
			sub:        user(subscription.MyTradesChannel, asset.Empty),
			assetPairs: map[asset.Item]currency.Pairs{asset.Empty: nil},
			template:   "user.tmpl",
			exp:        "user.trade;",
		},
		{
			name:       "an account's channel once for all of an asset's pairs",
			sub:        user(subscription.MyAccountChannel, asset.Spot),
			assetPairs: map[asset.Item]currency.Pairs{asset.Spot: {spotBTCUSD, spotETHUSD}},
			template:   "user.tmpl",
			exp:        "user.balance;",
		},
		{
			name:       "Crypto.com's name of a user channel GoCryptoTrader names",
			sub:        user(channelUserOrder, asset.Empty),
			assetPairs: map[asset.Item]currency.Pairs{asset.Empty: nil},
			template:   "user.tmpl",
			err:        subscription.ErrUseConstChannelName,
		},
		{
			name:       "a user channel Crypto.com does not serve",
			sub:        user("nope", asset.Empty),
			assetPairs: map[asset.Item]currency.Pairs{asset.Empty: nil},
			template:   "user.tmpl",
			err:        subscription.ErrNotSupported,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tmpl, err := ex.GetSubscriptionTemplate(tc.sub)
			require.NoError(t, err, "GetSubscriptionTemplate must not error")
			assert.Equal(t, tc.template, tmpl.Name(), "GetSubscriptionTemplate should return the template of the subscription's kind")
			var b strings.Builder
			err = tmpl.Execute(&b, &subscriptionTemplateContext{S: tc.sub, AssetPairs: tc.assetPairs, PairSeparator: ",", AssetSeparator: ";"})
			require.ErrorIs(t, err, tc.err, "the template must execute with the expected error")
			if tc.err == nil {
				assert.Equal(t, tc.exp, b.String(), "the template should name the subscription's channels")
			}
		})
	}
}

func TestDistinctChannels(t *testing.T) { //nolint:tparallel // the cases run in order before the later calls of the same function
	t.Parallel()
	sub := func(channel string, a asset.Item, levels int, qualifiedChannel string) *subscription.Subscription {
		return &subscription.Subscription{Channel: channel, Asset: a, Levels: levels, QualifiedChannel: qualifiedChannel}
	}
	spotTicker, marginTicker := sub(subscription.TickerChannel, asset.Spot, 0, "ticker.BTC_USD"), sub(subscription.TickerChannel, asset.Margin, 0, "ticker.BTC_USD")
	spotTrades, perpTicker := sub(subscription.AllTradesChannel, asset.Spot, 0, "trade.BTC_USD"), sub(subscription.TickerChannel, asset.PerpetualSwap, 0, "ticker.BTCUSD-PERP")
	shallowBook, deepBook := sub(subscription.OrderbookChannel, asset.Spot, 10, "book.BTC_USD.10"), sub(subscription.OrderbookChannel, asset.Margin, 50, "book.BTC_USD.50")
	ethBook := sub(subscription.OrderbookChannel, asset.Spot, 50, "book.ETH_USD.50")
	btcMark, ethMark := sub(MarkPriceChannel, asset.Spot, 0, "mark.BTC_USD"), sub(MarkPriceChannel, asset.Spot, 0, "mark.ETH_USD")
	minuteCandles, fiveMinuteCandles := sub(subscription.CandlesChannel, asset.Spot, 0, "candlestick.1m.BTC_USD"), sub(subscription.CandlesChannel, asset.Spot, 0, "candlestick.5m.BTC_USD")
	// Book channels a book subscription names without a depth after a dot, which Crypto.com does not name
	undottedBook, otherUndottedBook := sub(subscription.OrderbookChannel, asset.Spot, 10, "book"), sub(subscription.OrderbookChannel, asset.Spot, 50, "book")
	leadingDotBook, otherLeadingDotBook := sub(subscription.OrderbookChannel, asset.Spot, 10, ".10"), sub(subscription.OrderbookChannel, asset.Spot, 50, ".50")
	spotOrders, marginOrders := sub(subscription.MyOrdersChannel, asset.Spot, 0, "user.order"), sub(subscription.MyOrdersChannel, asset.Margin, 0, "user.order")
	balances := sub(subscription.MyAccountChannel, asset.Spot, 0, "user.balance")
	// The cases run in order before the later calls of the same function, so each also checks that a call keeps nothing
	// from the last
	for _, tc := range []struct {
		name string
		subs subscription.List
		exp  subscription.List
	}{
		{"no subscriptions", subscription.List{}, subscription.List{}},
		{"channels each subscribed once", subscription.List{spotTicker, spotTrades, perpTicker}, subscription.List{spotTicker, spotTrades, perpTicker}},
		{"spot and margin naming the same channel, the first kept", subscription.List{marginTicker, spotTicker, spotTrades}, subscription.List{marginTicker, spotTrades}},
		{"an instrument's books at different depths, the first kept", subscription.List{shallowBook, deepBook, ethBook}, subscription.List{shallowBook, ethBook}},
		{"channels of other kinds told apart by their whole names", subscription.List{btcMark, ethMark, minuteCandles, fiveMinuteCandles}, subscription.List{btcMark, ethMark, minuteCandles, fiveMinuteCandles}},
		{"book channels without a depth after a dot told apart by their whole names", subscription.List{undottedBook, otherUndottedBook, leadingDotBook, otherLeadingDotBook}, subscription.List{undottedBook, leadingDotBook, otherLeadingDotBook}},
		{"the account's channels, the same for every asset", subscription.List{spotOrders, marginOrders, balances}, subscription.List{spotOrders, balances}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := distinctChannels(tc.subs)
			require.Len(t, got, len(tc.exp), "distinctChannels must keep each channel once")
			for i := range tc.exp {
				assert.Samef(t, tc.exp[i], got[i], "distinctChannels should keep %s from the first subscription naming it, in order", tc.exp[i].QualifiedChannel)
			}
		})
	}
	assert.NotNil(t, distinctChannels(nil), "distinctChannels should return an empty list for no subscriptions")
}

func TestGenerateSubscriptions(t *testing.T) {
	t.Parallel()
	spotTicker := &subscription.Subscription{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: currency.Pairs{spotBTCUSD}}
	marginTicker := &subscription.Subscription{Enabled: true, Asset: asset.Margin, Channel: subscription.TickerChannel, Pairs: currency.Pairs{spotBTCUSD}}
	orders := &subscription.Subscription{Enabled: true, Channel: subscription.MyOrdersChannel, Authenticated: true}
	unservedMarketData := &subscription.Subscription{Enabled: true, Asset: asset.Spot, Channel: "nope"}
	unservedUser := &subscription.Subscription{Enabled: true, Channel: "nope", Authenticated: true}
	// expectedOrders returns the orders subscription as its template expands it
	expectedOrders := func() *subscription.Subscription {
		s := orders.Clone()
		s.QualifiedChannel = "user.order"
		return s
	}
	for _, tc := range []struct {
		name          string
		subs          subscription.List
		authenticated bool
		canAuthorise  bool
		exp           func() subscription.List
		err           error
	}{
		{
			name: "market data channels, spot's and margin's both kept",
			subs: subscription.List{spotTicker, marginTicker, orders}, canAuthorise: true,
			exp: func() subscription.List {
				return subscription.List{expandedSubscription(spotTicker, asset.Spot, spotBTCUSD, "ticker.BTC_USD"), expandedSubscription(marginTicker, asset.Margin, spotBTCUSD, "ticker.BTC_USD")}
			},
		},
		{
			name: "user channels",
			subs: subscription.List{spotTicker, orders}, authenticated: true, canAuthorise: true,
			exp: func() subscription.List { return subscription.List{expectedOrders()} },
		},
		{
			name: "user channels without authentication",
			subs: subscription.List{orders}, authenticated: true,
			exp: func() subscription.List { return subscription.List{} },
		},
		{
			name: "no subscriptions", canAuthorise: true,
			exp: func() subscription.List { return subscription.List{} },
		},
		{
			name: "user channels whatever market data Crypto.com does not serve",
			subs: subscription.List{unservedMarketData, orders}, authenticated: true, canAuthorise: true,
			exp: func() subscription.List { return subscription.List{expectedOrders()} },
		},
		{
			name: "market data Crypto.com does not serve",
			subs: subscription.List{unservedMarketData, orders}, canAuthorise: true,
			err: subscription.ErrNotSupported,
		},
		{
			name: "market data whatever user channels Crypto.com does not serve",
			subs: subscription.List{spotTicker, unservedUser}, canAuthorise: true,
			exp: func() subscription.List {
				return subscription.List{expandedSubscription(spotTicker, asset.Spot, spotBTCUSD, "ticker.BTC_USD")}
			},
		},
		{
			name: "a user channel Crypto.com does not serve",
			subs: subscription.List{spotTicker, unservedUser}, authenticated: true, canAuthorise: true,
			err: subscription.ErrNotSupported,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newTestExchange(t)
			ex.Websocket.SetCanUseAuthenticatedEndpoints(tc.canAuthorise)
			ex.Features.Subscriptions = tc.subs.Clone()
			got, err := ex.generateSubscriptions(tc.authenticated)
			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err, "generateSubscriptions should refuse a subscription of its kind Crypto.com does not serve")
				assert.Nil(t, got, "generateSubscriptions should return no subscriptions with its error")
				return
			}
			require.NoError(t, err, "generateSubscriptions must not error")
			require.NotNil(t, got, "generateSubscriptions must return a list")
			testsubs.EqualLists(t, tc.exp(), got)
		})
	}
}

func TestQualifiedChannels(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		subs subscription.List
		exp  []string
	}{
		{"no subscriptions", nil, []string{}},
		{"one subscription", subscription.List{{QualifiedChannel: "ticker.BTC_USD"}}, []string{"ticker.BTC_USD"}},
		{"subscriptions in order", subscription.List{{QualifiedChannel: "book.BTC_USD.10"}, {QualifiedChannel: "ticker.BTC_USD"}, {QualifiedChannel: "book.ETH_USD.10"}}, []string{"book.BTC_USD.10", "ticker.BTC_USD", "book.ETH_USD.10"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, qualifiedChannels(tc.subs), "qualifiedChannels should name each subscription's channel, in order")
		})
	}
}

// subscribeRequestSent checks a connection was sent one subscribe request, the first request of a new exchange, with
// params and waiting for an answer for each of its channels, which subscribeInspector reads
func subscribeRequestSent(t *testing.T, conn *scriptedConnection, params string, channels int) {
	t.Helper()
	requests := conn.recorded().requests
	require.Len(t, requests, 1, "wsSubscribeBatch must send one request")
	r := requests[0]
	req, ok := r.request.(*websocketRequest)
	require.True(t, ok, "wsSubscribeBatch must send a websocket request")
	assert.InDelta(t, time.Now().UnixMilli(), req.Nonce, float64(time.Minute.Milliseconds()), "wsSubscribeBatch should send the time in milliseconds as its nonce")
	sent := *req
	sent.Nonce = 0
	r.request = &sent
	exp := scriptedRequest{
		signature: int64(1),
		request:   &websocketRequest{ID: 1, Method: websocketMethodSubscribe, Params: json.RawMessage(params)},
		expected:  channels,
		inspector: subscribeInspector{},
	}
	assert.Equal(t, exp, r, "wsSubscribeBatch should send a subscribe request for the channels, waiting for an answer for each")
}

func TestWebsocketSubscribeBatch(t *testing.T) {
	t.Parallel()
	t.Run("channels subscribed over a connection", func(t *testing.T) {
		t.Parallel()
		venue := newMarketDataVenue(t)
		ex, err := newMarketDataTestExchange(t, venue, subscription.List{{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: currency.Pairs{spotBTCUSD}}})
		require.NoError(t, err, "Connect must not error")
		conn, err := ex.Websocket.GetConnection(websocketMarketDataConnection)
		require.NoError(t, err, "GetConnection must not error")
		subs := subscription.List{
			{Channel: subscription.AllTradesChannel, Asset: asset.Spot, Pairs: currency.Pairs{spotBTCUSD}, QualifiedChannel: "trade.BTC_USD"},
			{Channel: subscription.CandlesChannel, Asset: asset.Spot, Pairs: currency.Pairs{spotBTCUSD}, Interval: kline.OneMin, QualifiedChannel: "candlestick.1m.BTC_USD"},
		}
		require.NoError(t, ex.wsSubscribeBatch(t.Context(), conn, &subscriptionRequest{Channels: qualifiedChannels(subs)}, subs), "wsSubscribeBatch must subscribe the channels")
		for _, s := range subs {
			assert.Samef(t, s, ex.Websocket.GetSubscription(s), "wsSubscribeBatch should store %s", s.QualifiedChannel)
			assert.Equalf(t, subscription.SubscribedState, s.State(), "wsSubscribeBatch should leave %s subscribed", s.QualifiedChannel)
		}
		if mockTests {
			requests := venue.received(websocketMethodSubscribe)
			require.Len(t, requests, 2, "the ticker and then the channels must each be subscribed with a request")
			assert.JSONEq(t, `{"channels":["trade.BTC_USD","candlestick.1m.BTC_USD"]}`, string(requests[1].Params), "wsSubscribeBatch should name the channels")
		}
	})

	acknowledgement := func(channel string) string {
		return `{"id":1,"method":"subscribe","code":0,"channel":"` + channel + `"}`
	}
	tradeAcknowledgement := string(marketDataFrame(t, "trade.BTC_USD", 0))
	refusal := `{"id":1,"method":"subscribe","code":40001,"message":"BAD_REQUEST"}`
	newSubscriptions := func() subscription.List {
		return subscription.List{
			{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{spotBTCUSD}, QualifiedChannel: "ticker.BTC_USD"},
			{Channel: subscription.AllTradesChannel, Asset: asset.Spot, Pairs: currency.Pairs{spotBTCUSD}, QualifiedChannel: "trade.BTC_USD"},
		}
	}
	for _, tc := range []struct {
		name   string
		answer scriptedAnswer
		// stored are the channels stored once answered, and errText what the error says of the channels it is about
		stored  []string
		err     error
		apiErr  *APIError
		errText string
	}{
		{
			name:   "each channel acknowledged",
			answer: scriptedAnswer{responses: []string{acknowledgement("ticker.BTC_USD"), tradeAcknowledgement}},
			stored: []string{"ticker.BTC_USD", "trade.BTC_USD"},
		},
		{
			name:    "a channel rejected",
			answer:  scriptedAnswer{responses: []string{`{"id":1,"method":"subscribe","code":40003,"channel":"ticker.BTC_USD","message":"Unknown symbol"}`, tradeAcknowledgement}},
			stored:  []string{"trade.BTC_USD"},
			apiErr:  &APIError{Code: 40003, Message: "Unknown symbol"},
			errText: "ticker.BTC_USD: 40003 Unknown symbol",
		},
		{
			name:    "the request refused as a whole",
			answer:  scriptedAnswer{responses: []string{refusal}},
			apiErr:  &APIError{Code: 40001, Message: "BAD_REQUEST"},
			errText: "subscribing 2 channels: 40001 BAD_REQUEST",
		},
		{
			name:    "the request refused as a whole after a channel is acknowledged",
			answer:  scriptedAnswer{responses: []string{acknowledgement("ticker.BTC_USD"), refusal}},
			stored:  []string{"ticker.BTC_USD"},
			apiErr:  &APIError{Code: 40001, Message: "BAD_REQUEST"},
			errText: "subscribing 2 channels: 40001 BAD_REQUEST",
		},
		{
			name:   "an acknowledgement naming no channel",
			answer: scriptedAnswer{responses: []string{`{"id":1,"method":"subscribe","code":0}`, tradeAcknowledgement}},
			stored: []string{"trade.BTC_USD"},
			err:    errMalformedWebsocketMessage,
		},
		{
			name:   "answers for a channel not requested or answered already passed over",
			answer: scriptedAnswer{responses: []string{acknowledgement("ticker.ETH_USD"), acknowledgement("ticker.BTC_USD"), acknowledgement("ticker.BTC_USD"), tradeAcknowledgement}},
			stored: []string{"ticker.BTC_USD", "trade.BTC_USD"},
		},
		{
			name:   "a wait ended before every channel is answered",
			answer: scriptedAnswer{responses: []string{acknowledgement("ticker.BTC_USD")}, err: websocket.ErrSignatureTimeout},
			stored: []string{"ticker.BTC_USD"},
			err:    websocket.ErrSignatureTimeout,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			ex.Websocket = websocket.NewManager()
			subs := newSubscriptions()
			conn := newScriptedConnection(tc.answer)
			err := ex.wsSubscribeBatch(t.Context(), conn, &subscriptionRequest{Channels: qualifiedChannels(subs)}, subs)
			if tc.apiErr != nil {
				var apiErr *APIError
				require.ErrorAs(t, err, &apiErr, "wsSubscribeBatch must report the error Crypto.com answers with")
				assert.Equal(t, tc.apiErr, apiErr, "wsSubscribeBatch should report the code and message Crypto.com answers with")
				assert.ErrorContains(t, err, tc.errText, "wsSubscribeBatch should say what the error is about")
			} else {
				assert.ErrorIs(t, err, tc.err, "wsSubscribeBatch should return the expected error")
			}
			subscribeRequestSent(t, conn, `{"channels":["ticker.BTC_USD","trade.BTC_USD"]}`, 2)
			for _, s := range subs {
				if slices.Contains(tc.stored, s.QualifiedChannel) {
					assert.Samef(t, s, ex.Websocket.GetSubscription(s), "wsSubscribeBatch should store %s", s.QualifiedChannel)
					assert.Equalf(t, subscription.SubscribedState, s.State(), "wsSubscribeBatch should leave %s subscribed", s.QualifiedChannel)
					continue
				}
				assert.Nilf(t, ex.Websocket.GetSubscription(s), "wsSubscribeBatch should not store %s", s.QualifiedChannel)
				assert.Equalf(t, subscription.InactiveState, s.State(), "wsSubscribeBatch should leave the state of %s alone", s.QualifiedChannel)
			}
		})
	}

	t.Run("an answer that cannot be decoded", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		ex.Websocket = websocket.NewManager()
		subs := newSubscriptions()
		conn := newScriptedConnection(scriptedAnswer{responses: []string{`{"id":1,"method":"subscribe","code":0,"channel":`, tradeAcknowledgement}})
		assert.Error(t, ex.wsSubscribeBatch(t.Context(), conn, &subscriptionRequest{Channels: qualifiedChannels(subs)}, subs), "wsSubscribeBatch should report an answer it cannot decode")
		assert.Nil(t, ex.Websocket.GetSubscription(subs[0]), "wsSubscribeBatch should not store the channel whose answer cannot be decoded")
		assert.Same(t, subs[1], ex.Websocket.GetSubscription(subs[1]), "wsSubscribeBatch should store the channel acknowledged")
	})

	t.Run("a channel that cannot be stored", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		ex.Websocket = websocket.NewManager()
		subs := newSubscriptions()
		conn := newScriptedConnection(scriptedAnswer{responses: []string{acknowledgement("ticker.BTC_USD"), tradeAcknowledgement}})
		require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, subs[0]), "AddSuccessfulSubscriptions must not error")
		err := ex.wsSubscribeBatch(t.Context(), conn, &subscriptionRequest{Channels: qualifiedChannels(subs)}, subs)
		assert.ErrorIs(t, err, subscription.ErrDuplicate, "wsSubscribeBatch should report a channel it cannot store")
		assert.ErrorIs(t, err, subscription.ErrInStateAlready, "wsSubscribeBatch should report a channel it cannot mark subscribed")
		for _, s := range subs {
			assert.Samef(t, s, ex.Websocket.GetSubscription(s), "%s should be stored", s.QualifiedChannel)
		}
	})

	t.Run("book channels with their updates", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		ex.Websocket = websocket.NewManager()
		book := &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{spotBTCUSD}, Levels: 10, Interval: kline.TenMilliseconds, QualifiedChannel: "book.BTC_USD.10"}
		conn := newScriptedConnection(scriptedAnswer{responses: []string{acknowledgement(book.QualifiedChannel)}})
		params := &subscriptionRequest{Channels: []string{book.QualifiedChannel}, BookSubscriptionType: BookSubscriptionTypeSnapshotAndUpdate, BookUpdateFrequency: 10}
		require.NoError(t, ex.wsSubscribeBatch(t.Context(), conn, params, subscription.List{book}), "wsSubscribeBatch must not error")
		subscribeRequestSent(t, conn, `{"channels":["book.BTC_USD.10"],"book_subscription_type":"SNAPSHOT_AND_UPDATE","book_update_frequency":10}`, 1)
		assert.Same(t, book, ex.Websocket.GetSubscription(book), "wsSubscribeBatch should store the book channel")
	})
}

func TestWebsocketUnsubscribe(t *testing.T) {
	t.Parallel()
	t.Run("channels unsubscribed over a connection", func(t *testing.T) {
		t.Parallel()
		venue := newMarketDataVenue(t)
		ex, err := newMarketDataTestExchange(t, venue, subscription.List{
			{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: currency.Pairs{spotBTCUSD}},
			{Enabled: true, Asset: asset.Spot, Channel: subscription.AllTradesChannel, Pairs: currency.Pairs{spotBTCUSD}},
		})
		require.NoError(t, err, "Connect must not error")
		conn, err := ex.Websocket.GetConnection(websocketMarketDataConnection)
		require.NoError(t, err, "GetConnection must not error")
		subs := conn.Subscriptions().List()
		require.Len(t, subs, 2, "both channels must be subscribed")
		require.NoError(t, ex.wsUnsubscribe(t.Context(), conn, subs), "wsUnsubscribe must not error")
		assert.Empty(t, ex.Websocket.GetSubscriptions(), "wsUnsubscribe should remove the channels")
		for _, s := range subs {
			assert.Equalf(t, subscription.UnsubscribedState, s.State(), "wsUnsubscribe should leave %s unsubscribed", s.QualifiedChannel)
		}
		if mockTests {
			requested := requestedChannels(t, venue, websocketMethodUnsubscribe)
			require.Len(t, requested, 1, "wsUnsubscribe must unsubscribe the channels with one request")
			assert.ElementsMatch(t, []string{"ticker.BTC_USD", "trade.BTC_USD"}, requested[0], "wsUnsubscribe should name both channels")
		}
	})

	unsubscribed := `{"id":1,"method":"unsubscribe","code":0}`
	t.Run("no channels", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		ex.Websocket = websocket.NewManager()
		conn := newScriptedConnection(scriptedAnswer{responses: []string{unsubscribed}})
		require.NoError(t, ex.wsUnsubscribe(t.Context(), conn, nil), "wsUnsubscribe must not error without channels")
		require.NoError(t, ex.wsUnsubscribe(t.Context(), conn, subscription.List{}), "wsUnsubscribe must not error for an empty list")
		assert.Empty(t, conn.recorded().requests, "wsUnsubscribe should send nothing without channels")
	})

	// requestChannels returns the channels a request sent over a connection names
	requestChannels := func(t *testing.T, r scriptedRequest) []string {
		t.Helper()
		req, ok := r.request.(*websocketRequest)
		require.True(t, ok, "wsUnsubscribe must send websocket requests")
		require.Equal(t, websocketMethodUnsubscribe, req.Method, "wsUnsubscribe must send unsubscribe requests")
		var params subscriptionRequest
		require.NoError(t, json.Unmarshal(req.Params, &params), "the request's params must decode")
		return params.Channels
	}
	for _, tc := range []struct {
		name string
		// sizes are the lengths of the channels' names, and answers the answers to the requests sent in turn
		sizes   []int
		answers []scriptedAnswer
		// requests holds the indexes of the channels each request names, and kept those of the channels left stored
		requests [][]int
		kept     []int
		apiErr   *APIError
	}{
		{
			name:     "channels unsubscribed with one request",
			sizes:    []int{100, 200},
			answers:  []scriptedAnswer{{responses: []string{unsubscribed}}},
			requests: [][]int{{0, 1}},
		},
		{
			name:     "channels past what a request holds unsubscribed with as many requests as they take",
			sizes:    []int{3000, 1000, 50},
			answers:  []scriptedAnswer{{responses: []string{unsubscribed}}},
			requests: [][]int{{0}, {1, 2}},
		},
		{
			name:     "a request refused keeping its channels, the others unsubscribed",
			sizes:    []int{3000, 1000},
			answers:  []scriptedAnswer{{responses: []string{`{"id":1,"method":"unsubscribe","code":40001,"message":"BAD_REQUEST"}`}}, {responses: []string{unsubscribed}}},
			requests: [][]int{{0}, {1}},
			kept:     []int{0},
			apiErr:   &APIError{Code: 40001, Message: "BAD_REQUEST"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			ex.Websocket = websocket.NewManager()
			conn := newScriptedConnection(tc.answers...)
			subs := make(subscription.List, len(tc.sizes))
			for i, size := range tc.sizes {
				pair := currency.NewPairWithDelimiter("TESTCOIN"+strconv.Itoa(i), "USD", currency.UnderscoreDelimiter)
				subs[i] = &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{pair}, QualifiedChannel: strconv.Itoa(i) + strings.Repeat("c", size-1)}
			}
			require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, subs...), "AddSuccessfulSubscriptions must not error")

			err := ex.wsUnsubscribe(t.Context(), conn, subs)
			if tc.apiErr != nil {
				var apiErr *APIError
				require.ErrorAs(t, err, &apiErr, "wsUnsubscribe must report the request refused")
				assert.Equal(t, tc.apiErr, apiErr, "wsUnsubscribe should report the refusal's code and message")
			} else {
				require.NoError(t, err, "wsUnsubscribe must not error")
			}
			requests := conn.recorded().requests
			require.Len(t, requests, len(tc.requests), "wsUnsubscribe must send as many requests as the channels take")
			for i, indexes := range tc.requests {
				exp := make([]string, len(indexes))
				for j, index := range indexes {
					exp[j] = subs[index].QualifiedChannel
				}
				assert.Equalf(t, exp, requestChannels(t, requests[i]), "request %d should name its channels in order", i)
			}
			for i, s := range subs {
				if slices.Contains(tc.kept, i) {
					assert.Samef(t, s, ex.Websocket.GetSubscription(s), "wsUnsubscribe should keep channel %d, whose request was refused", i)
					assert.Equalf(t, subscription.SubscribedState, s.State(), "wsUnsubscribe should leave channel %d subscribed", i)
					continue
				}
				assert.Nilf(t, ex.Websocket.GetSubscription(s), "wsUnsubscribe should remove channel %d", i)
				assert.Equalf(t, subscription.UnsubscribedState, s.State(), "wsUnsubscribe should leave channel %d unsubscribed", i)
			}
		})
	}
}
