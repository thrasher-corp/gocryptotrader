package cryptodotcom

import (
	"context"
	"math"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
)

func TestWebsocketMessageID(t *testing.T) {
	t.Parallel()
	for msg, exp := range map[string]int64{
		`{"id":1792891216162,"method":"public/heartbeat","code":0}`: 1792891216162,
		`{"id":"7","method":"private/create-order"}`:                7,
		`{"id":-1,"method":"subscribe"}`:                            -1,
		`{"method":"subscribe"}`:                                    0,
	} {
		id, err := websocketMessageID([]byte(msg))
		require.NoErrorf(t, err, "websocketMessageID must not error for %s", msg)
		assert.Equalf(t, exp, id, "websocketMessageID should read the ID of %s", msg)
	}
	for _, msg := range []string{`{"id":"x"}`, `{"id":true}`, `{"id":`, `{"id":1.5}`} {
		_, err := websocketMessageID([]byte(msg))
		assert.ErrorIsf(t, err, errMalformedWebsocketMessage, "websocketMessageID should refuse %s", msg)
	}
}

func TestDecodeWebsocketResponse(t *testing.T) {
	t.Parallel()
	var result OrderIDResponse
	require.NoError(t, decodeWebsocketResponse([]byte(`{"id":1,"method":"private/create-order","code":0,"result":{"order_id":"5"}}`), &result), "decodeWebsocketResponse must not error")
	assert.Equal(t, OrderIDResponse{OrderID: "5"}, result, "decodeWebsocketResponse should decode the result")
	assert.NoError(t, decodeWebsocketResponse([]byte(`{"id":1,"method":"private/cancel-order","code":0}`), &result), "decodeWebsocketResponse should not error without a result")
	assert.NoError(t, decodeWebsocketResponse([]byte(`{"id":1,"method":"private/cancel-order","code":0,"result":{}}`), nil), "decodeWebsocketResponse should not error without a result to decode into")
	err := decodeWebsocketResponse([]byte(`{"id":11,"method":"public/auth","code":40101,"message":"Unauthorized","original":"{\"api_key\":\"mock-api-key\"}"}`), nil)
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr, "decodeWebsocketResponse must return the error Crypto.com reports")
	assert.Equal(t, &APIError{Code: 40101, Message: "Unauthorized"}, apiErr, "decodeWebsocketResponse should leave out the original request, which carries the API key")
	assert.Error(t, decodeWebsocketResponse([]byte(`{"id":`), &result), "decodeWebsocketResponse should error for a malformed response")
	assert.Error(t, decodeWebsocketResponse([]byte(`{"id":1,"code":0,"result":{"order_id":5}}`), &result), "decodeWebsocketResponse should error for a result that cannot be decoded")
}

func TestSendWebsocketRequest(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("SendWebsocketRequest is tested against the mock websocket, as it sends test methods")
	}
	ex, venue := newWebsocketTestExchange(t)
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
}

func TestSendConnectionRequest(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("sendConnectionRequest is tested against the mock websocket, as it sends test methods")
	}
	ex, venue := newWebsocketTestExchange(t)
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
}

func TestSendWebsocketRequestNotConnected(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	for _, authenticated := range []bool{false, true} {
		ex.Websocket.SetCanUseAuthenticatedEndpoints(authenticated)
		err := ex.SendWebsocketRequest(t.Context(), "private/test-method", nil, nil)
		assert.ErrorIsf(t, err, websocket.ErrNotConnected, "SendWebsocketRequest should report a websocket not connected as such, whether or not it can authenticate (%t)", authenticated)
	}
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
	t.Cleanup(venue.closing)
	err := ex.Websocket.Connect(t.Context())
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr, "Connect must fail when authentication is refused")
	assert.Equal(t, &APIError{Code: 40101, Message: "Unauthorized"}, apiErr, "Connect should report the refused authentication")
	require.Len(t, venue.received(websocketMethodAuth), 1, "the trade connection must try to authenticate once")
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
		t.Skip("what a request holds is tested against the mock websocket, which records it")
	}
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
}

// TestWebsocketSubscriptionLimitsRefusal has Crypto.com refuse the first request of several, whose channels alone are left
// out or kept, and then one channel of a request
func TestWebsocketSubscriptionLimitsRefusal(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("a refusal is tested against the mock websocket")
	}
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
	for msg, exp := range map[string]string{
		`{"id":1,"method":"subscribe","code":0,"channel":"ticker.BTC_USD"}`:                                             "ticker.BTC_USD",
		`{"id":1,"method":"subscribe","code":0,"result":{"subscription":"book.BTC_USD.10","channel":"book","data":[]}}`: "book.BTC_USD.10",
		`{"id":1,"method":"subscribe","code":40003,"channel":"ticker.BTC_USDT","message":"Unknown symbol"}`:             "ticker.BTC_USDT",
		`{"id":1,"method":"subscribe","code":40001,"message":"BAD_REQUEST"}`:                                            "",
		`{"id":1,"method":"subscribe","code":0,"result":{"channel":"book","data":[]}}`:                                  "",
		`{"id":1,"method":"subscribe","code":0,"result":{"subscription":1}}`:                                            "",
	} {
		var resp websocketResponse
		require.NoErrorf(t, json.Unmarshal([]byte(msg), &resp), "Unmarshal must not error for %s", msg)
		assert.Equalf(t, exp, resp.subscribedChannel(), "subscribedChannel should name the channel %s answers", msg)
	}
}

func TestSubscribeInspector(t *testing.T) {
	t.Parallel()
	for msg, exp := range map[string]bool{
		`{"id":1,"method":"subscribe","code":40001,"message":"BAD_REQUEST"}`:                                true,
		`{"id":1,"method":"subscribe","code":"40001","message":"BAD_REQUEST"}`:                              true,
		`{"id":1,"method":"subscribe","code":40003,"channel":"ticker.BTC_USDT","message":"Unknown symbol"}`: false,
		`{"id":1,"method":"subscribe","code":40003,"result":{"subscription":"ticker.BTC_USDT"}}`:            false,
		`{"id":1,"method":"subscribe","code":0,"channel":"ticker.BTC_USD"}`:                                 false,
		`{"id":1,"method":"subscribe","code":0}`:                                                            false,
		`{"id":`:                                                                                            false,
	} {
		assert.Equalf(t, exp, subscribeInspector{}.IsFinal([]byte(msg)), "IsFinal should report whether %s refuses its subscribe request as a whole", msg)
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
}

func TestDefaultSubscriptions(t *testing.T) {
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
}

func TestChannelServesAsset(t *testing.T) {
	t.Parallel()
	everyAsset := []asset.Item{asset.Spot, asset.Margin, asset.PerpetualSwap, asset.Futures}
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
		for _, a := range everyAsset {
			serves := slices.Contains(served, a)
			got, err := channelServesAsset(&subscription.Subscription{Channel: channel, Asset: asset.All}, a)
			require.NoErrorf(t, err, "channelServesAsset must not error for a %s subscription of every asset", channel)
			assert.Equalf(t, serves, got, "channelServesAsset should report whether the %s channel serves %s", channel, a)
			got, err = channelServesAsset(&subscription.Subscription{Channel: channel, Asset: a}, a)
			if serves {
				require.NoErrorf(t, err, "channelServesAsset must not error for a %s subscription of %s", channel, a)
				assert.Truef(t, got, "channelServesAsset should report the %s channel serves %s", channel, a)
				continue
			}
			assert.ErrorIsf(t, err, subscription.ErrNotSupported, "channelServesAsset should refuse a %s subscription of %s", channel, a)
		}
	}
	_, err := channelServesAsset(&subscription.Subscription{Channel: "nope", Asset: asset.All}, asset.Spot)
	assert.ErrorIs(t, err, subscription.ErrNotSupported, "channelServesAsset should refuse a channel Crypto.com does not serve")
}

func TestQualifiedChannel(t *testing.T) {
	t.Parallel()
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
		{&subscription.Subscription{Channel: MarkPriceChannel}, asset.Spot, spotBTCUSD, "mark.BTC_USD"},
		{&subscription.Subscription{Channel: SettlementPriceChannel}, asset.Futures, futureBTCUSD, "settlement.BTCUSD-261030"},
		{&subscription.Subscription{Channel: FundingRateChannel}, asset.PerpetualSwap, perpBTCUSD, "funding.BTCUSD-PERP"},
		{&subscription.Subscription{Channel: EstimatedFundingRateChannel}, asset.PerpetualSwap, perpBTCUSD, "estimatedfunding.BTCUSD-PERP"},
	} {
		got, err := e.qualifiedChannel(tc.sub, tc.a, tc.pair)
		require.NoErrorf(t, err, "qualifiedChannel must not error for the %s channel of %s %s", tc.sub.Channel, tc.a, tc.pair)
		assert.Equalf(t, tc.exp, got, "qualifiedChannel should name the %s channel of %s %s", tc.sub.Channel, tc.a, tc.pair)
	}
	for _, tc := range []struct {
		sub *subscription.Subscription
		a   asset.Item
		err error
	}{
		{&subscription.Subscription{Channel: "nope"}, asset.Spot, subscription.ErrNotSupported},
		{&subscription.Subscription{Channel: IndexPriceChannel}, asset.Spot, errNoUnderlyingIndex},
		{&subscription.Subscription{Channel: subscription.TickerChannel}, asset.Options, asset.ErrNotSupported},
		{&subscription.Subscription{Channel: subscription.OrderbookChannel, Levels: 20}, asset.Spot, subscription.ErrInvalidLevel},
		{&subscription.Subscription{Channel: subscription.OrderbookChannel, Interval: kline.OneMin}, asset.Spot, subscription.ErrInvalidInterval},
		{&subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.ThreeMin}, asset.Spot, kline.ErrUnsupportedInterval},
	} {
		_, err := e.qualifiedChannel(tc.sub, tc.a, spotBTCUSD)
		assert.ErrorIsf(t, err, tc.err, "qualifiedChannel should refuse the %s channel of %s with levels %d and interval %s", tc.sub.Channel, tc.a, tc.sub.Levels, tc.sub.Interval)
	}
}

func TestBookDepth(t *testing.T) {
	t.Parallel()
	for levels, exp := range map[int]int{0: 50, 10: 10, 50: 50} {
		got, err := bookDepth(&subscription.Subscription{Levels: levels})
		require.NoErrorf(t, err, "bookDepth must not error for %d levels", levels)
		assert.Equalf(t, exp, got, "bookDepth should give the depth of a subscription to %d levels", levels)
	}
	_, err := bookDepth(&subscription.Subscription{Levels: 20})
	assert.ErrorIs(t, err, subscription.ErrInvalidLevel, "bookDepth should refuse a depth Crypto.com does not serve")
}

func TestBookUpdates(t *testing.T) {
	t.Parallel()
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
		subscriptionType, frequency, err := bookUpdates(&subscription.Subscription{Interval: tc.interval})
		require.NoErrorf(t, err, "bookUpdates must not error for an interval of %s", tc.interval)
		assert.Equalf(t, tc.subscriptionType, subscriptionType, "bookUpdates should give the subscription type of an interval of %s", tc.interval)
		assert.Equalf(t, tc.frequency, frequency, "bookUpdates should give the update frequency of an interval of %s", tc.interval)
	}
	_, _, err := bookUpdates(&subscription.Subscription{Interval: kline.ThousandMilliseconds})
	assert.ErrorIs(t, err, subscription.ErrInvalidInterval, "bookUpdates should refuse an interval Crypto.com does not update books at")

	// A configured book subscription names its snapshots as 500ms
	var configured kline.Interval
	require.NoError(t, configured.UnmarshalText([]byte("500ms")), "UnmarshalText must not error")
	assert.Equal(t, bookSnapshotInterval, configured, "a configured interval of 500ms should name book snapshots")
}
