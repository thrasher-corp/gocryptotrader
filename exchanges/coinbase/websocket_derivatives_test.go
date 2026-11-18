package coinbase

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

func TestDerivativesChannelName(t *testing.T) {
	t.Parallel()
	for channel, exp := range map[string]string{
		subscription.TickerChannel:    ChannelDerivativesTicker,
		subscription.OrderbookChannel: ChannelDerivativesBook,
		subscription.AllTradesChannel: ChannelDerivativesTrades,
		subscription.CandlesChannel:   ChannelDerivativesChartTrades,
		subscription.MyOrdersChannel:  ChannelDerivativesUserOrders,
		subscription.MyTradesChannel:  ChannelDerivativesUserTrades,
		ChannelDerivativesQuote:       ChannelDerivativesQuote,
	} {
		t.Run(subtestName(channel), func(t *testing.T) {
			t.Parallel()
			name, err := derivativesChannelName(&subscription.Subscription{Channel: channel})
			require.NoErrorf(t, err, "derivativesChannelName must not error for %s", channel)
			assert.Equalf(t, exp, name, "derivativesChannelName should name %s's gateway channel", channel)
		})
	}
	for _, channel := range []string{subscription.MyAccountChannel, ChannelStatus, "nonsense"} {
		t.Run(subtestName(channel), func(t *testing.T) {
			t.Parallel()
			_, err := derivativesChannelName(&subscription.Subscription{Channel: channel})
			assert.ErrorIsf(t, err, subscription.ErrNotSupported, "derivativesChannelName should error for %s, which the gateway does not serve", channel)
		})
	}
}

func TestQualifyDerivativesInstrumentChannel(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		s    *subscription.Subscription
		exp  string
		err  error
	}{
		{name: "ticker", s: &subscription.Subscription{Channel: subscription.TickerChannel}, exp: "ticker.BTC_USDC-PERPETUAL.100ms"},
		{name: "aggregated ticker", s: &subscription.Subscription{Channel: subscription.TickerChannel, Interval: kline.ThousandMilliseconds}, exp: "ticker.BTC_USDC-PERPETUAL.agg2"},
		{name: "raw book", s: &subscription.Subscription{Channel: subscription.OrderbookChannel, Interval: kline.Raw}, exp: "book.BTC_USDC-PERPETUAL.raw"},
		{name: "grouped book", s: &subscription.Subscription{Channel: subscription.OrderbookChannel, Levels: 10, Params: map[string]any{"group": 5.0}}, exp: "book.BTC_USDC-PERPETUAL.5.10.100ms"},
		{name: "ungrouped book of 20 levels", s: &subscription.Subscription{Channel: subscription.OrderbookChannel, Levels: 20}, exp: "book.BTC_USDC-PERPETUAL.none.20.100ms"},
		{name: "book of 5 levels", s: &subscription.Subscription{Channel: subscription.OrderbookChannel, Levels: 5}, err: subscription.ErrInvalidLevel},
		{name: "book of an unsupported group", s: &subscription.Subscription{Channel: subscription.OrderbookChannel, Levels: 10, Params: map[string]any{"group": "3"}}, err: errDerivativesChannelParameter},
		{name: "five minute candles", s: &subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.FiveMin}, exp: "chart.trades.BTC_USDC-PERPETUAL.5"},
		{name: "candles without an interval", s: &subscription.Subscription{Channel: subscription.CandlesChannel}, err: kline.ErrUnsupportedInterval},
		{name: "two hour candles", s: &subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.TwoHour}, exp: "chart.trades.BTC_USDC-PERPETUAL.120"},
		// The gateway aligns these candles to its 08:00 UTC daily settlement
		{name: "three hour candles", s: &subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.ThreeHour}, err: kline.ErrUnsupportedInterval},
		{name: "six hour candles", s: &subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.SixHour}, err: kline.ErrUnsupportedInterval},
		{name: "twelve hour candles", s: &subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.TwelveHour}, err: kline.ErrUnsupportedInterval},
		{name: "daily candles", s: &subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.OneDay}, err: kline.ErrUnsupportedInterval},
		{name: "quote", s: &subscription.Subscription{Channel: ChannelDerivativesQuote, Interval: kline.OneDay}, exp: "quote.BTC_USDC-PERPETUAL"},
		{name: "one minute ticker", s: &subscription.Subscription{Channel: subscription.TickerChannel, Interval: kline.OneMin}, err: kline.ErrUnsupportedInterval},
		{name: "orders without authentication", s: &subscription.Subscription{Channel: subscription.MyOrdersChannel}, err: errDerivativesChannelNeedsAuth},
		{name: "authenticated raw orders", s: &subscription.Subscription{Channel: subscription.MyOrdersChannel, Authenticated: true, Interval: kline.Raw}, exp: "user.orders.BTC_USDC-PERPETUAL.raw"},
		{name: "unsupported channel", s: &subscription.Subscription{Channel: "nonsense"}, err: subscription.ErrNotSupported},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			qualified, err := qualifyDerivativesInstrumentChannel(tc.s, testUSDCPerpetual)
			if tc.err != nil {
				assert.ErrorIsf(t, err, tc.err, "qualifyDerivativesInstrumentChannel should error for %s", tc.s.Channel)
				return
			}
			require.NoErrorf(t, err, "qualifyDerivativesInstrumentChannel must not error for %s", tc.s.Channel)
			assert.Equalf(t, tc.exp, qualified, "qualifyDerivativesInstrumentChannel should qualify %s", tc.s.Channel)
		})
	}
}

func TestQualifyDerivativesChannel(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		s    *subscription.Subscription
		exp  string
		err  error
	}{
		{name: "platform state", s: &subscription.Subscription{Channel: ChannelDerivativesPlatformState}, exp: "platform_state"},
		{name: "all trades", s: &subscription.Subscription{Channel: subscription.AllTradesChannel}, exp: "trades.any.any.100ms"},
		{name: "trades of a kind and currency", s: &subscription.Subscription{Channel: subscription.AllTradesChannel, Params: map[string]any{"kind": "future", "currency": "usdc"}}, exp: "trades.future.USDC.100ms"},
		{name: "trades of a dotted kind", s: &subscription.Subscription{Channel: subscription.AllTradesChannel, Params: map[string]any{"kind": "future.option"}}, err: errDerivativesChannelParameter},
		{name: "trades of a numeric currency", s: &subscription.Subscription{Channel: subscription.AllTradesChannel, Params: map[string]any{"currency": 1}}, err: errDerivativesChannelParameter},
		{name: "ticker without an instrument", s: &subscription.Subscription{Channel: subscription.TickerChannel}, err: errDerivativesChannelNeedsInstrument},
		{name: "price index", s: &subscription.Subscription{Channel: ChannelDerivativesPriceIndex, Params: map[string]any{"index_name": "btc_usd"}}, exp: "deribit_price_index.btc_usd"},
		{name: "price index without its name", s: &subscription.Subscription{Channel: ChannelDerivativesPriceIndex}, err: errDerivativesChannelParameter},
		{name: "portfolio of a currency code", s: &subscription.Subscription{Channel: ChannelDerivativesUserPortfolio, Authenticated: true, Params: map[string]any{"currency": currency.USDC}}, exp: "user.portfolio.USDC"},
		{name: "portfolio of any currency", s: &subscription.Subscription{Channel: ChannelDerivativesUserPortfolio, Authenticated: true, Params: map[string]any{"currency": "ANY"}}, exp: "user.portfolio.any"},
		{name: "portfolio of a dotted currency", s: &subscription.Subscription{Channel: ChannelDerivativesUserPortfolio, Authenticated: true, Params: map[string]any{"currency": "U.SD"}}, err: errDerivativesChannelParameter},
		{name: "portfolio without authentication", s: &subscription.Subscription{Channel: ChannelDerivativesUserPortfolio}, err: errDerivativesChannelNeedsAuth},
		{name: "authenticated raw trades", s: &subscription.Subscription{Channel: subscription.MyTradesChannel, Authenticated: true, Interval: kline.Raw}, exp: "user.trades.any.any.raw"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			qualified, err := qualifyDerivativesChannel(tc.s)
			if tc.err != nil {
				assert.ErrorIsf(t, err, tc.err, "qualifyDerivativesChannel should error for %s", tc.s.Channel)
				return
			}
			require.NoErrorf(t, err, "qualifyDerivativesChannel must not error for %s", tc.s.Channel)
			assert.Equalf(t, tc.exp, qualified, "qualifyDerivativesChannel should qualify %s", tc.s.Channel)
		})
	}
}

func TestDerivativesBookGroup(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		value any
		exp   string
	}{
		{"unset", nil, "none"},
		{"string", "250", "250"},
		{"float", 10.0, "10"},
		{"integer", 25, "25"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			group, err := derivativesBookGroup(map[string]any{"group": tc.value})
			require.NoErrorf(t, err, "derivativesBookGroup must not error for %v", tc.value)
			assert.Equalf(t, tc.exp, group, "derivativesBookGroup should name group %v", tc.value)
		})
	}
	for name, value := range map[string]any{"unsupported group": "3", "boolean": true} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := derivativesBookGroup(map[string]any{"group": value})
			assert.ErrorIsf(t, err, errDerivativesChannelParameter, "derivativesBookGroup should error for %v", value)
		})
	}
}

func TestDerivativesChannelParameter(t *testing.T) {
	t.Parallel()
	value, err := derivativesChannelParameter(nil, "kind", derivativesChannelAny)
	require.NoError(t, err, "derivativesChannelParameter must not error for an unset parameter with a default")
	assert.Equal(t, derivativesChannelAny, value, "derivativesChannelParameter should return the default for an unset parameter")
	_, err = derivativesChannelParameter(nil, "index_name", "")
	assert.ErrorIs(t, err, errDerivativesChannelParameter, "derivativesChannelParameter should error for an unset required parameter")
	_, err = derivativesChannelParameter(map[string]any{"kind": ""}, "kind", derivativesChannelAny)
	assert.ErrorIs(t, err, errDerivativesChannelParameter, "derivativesChannelParameter should error for an empty parameter")
	_, err = derivativesChannelParameter(map[string]any{"kind": 1}, "kind", derivativesChannelAny)
	assert.ErrorIs(t, err, errDerivativesChannelParameter, "derivativesChannelParameter should error for a parameter that is not a string")
}

func TestDerivativesChannelCurrency(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		value any
		exp   string
	}{
		{"unset", nil, derivativesChannelAny},
		{"mixed case any", "Any", derivativesChannelAny},
		{"lower case", "btc", "BTC"},
		{"currency code", currency.ETH, "ETH"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ccy, err := derivativesChannelCurrency(map[string]any{"currency": tc.value})
			require.NoErrorf(t, err, "derivativesChannelCurrency must not error for %v", tc.value)
			assert.Equalf(t, tc.exp, ccy, "derivativesChannelCurrency should name currency %v", tc.value)
		})
	}
	for name, value := range map[string]any{"empty": "", "dotted": "B.TC", "float": 1.5} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := derivativesChannelCurrency(map[string]any{"currency": value})
			assert.ErrorIsf(t, err, errDerivativesChannelParameter, "derivativesChannelCurrency should error for %v", value)
		})
	}
}

func TestDerivativesResolutionInterval(t *testing.T) {
	t.Parallel()
	for resolution, exp := range map[string]kline.Interval{"1": kline.OneMin, "60": kline.OneHour, "720": kline.TwelveHour, "1D": kline.OneDay} {
		t.Run(subtestName(resolution), func(t *testing.T) {
			t.Parallel()
			interval, err := derivativesResolutionInterval(resolution)
			require.NoErrorf(t, err, "derivativesResolutionInterval must not error for %s", resolution)
			assert.Equalf(t, exp, interval, "derivativesResolutionInterval should return resolution %s's interval", resolution)
		})
	}
	_, err := derivativesResolutionInterval("1W")
	assert.ErrorIs(t, err, kline.ErrInvalidInterval, "derivativesResolutionInterval should error for a resolution that is not minutes")
	_, err = derivativesResolutionInterval("7")
	assert.ErrorIs(t, err, kline.ErrUnsupportedInterval, "derivativesResolutionInterval should error for a resolution chart.trades does not serve")
}

func TestDerivativesInstrumentPair(t *testing.T) {
	t.Parallel()
	for instrument, exp := range map[string]struct {
		a    asset.Item
		pair currency.Pair
	}{
		"BTC_USDC-PERPETUAL": {asset.USDCMarginedFutures, testUSDCPerpetual},
		"BTC-PERPETUAL":      {asset.CoinMarginedFutures, testCoinPerpetual},
	} {
		t.Run(subtestName(instrument), func(t *testing.T) {
			t.Parallel()
			a, pair, err := derivativesInstrumentPair(instrument)
			require.NoErrorf(t, err, "derivativesInstrumentPair must not error for %s", instrument)
			assert.Equalf(t, exp.a, a, "derivativesInstrumentPair should return %s's asset", instrument)
			assert.Equalf(t, exp.pair, pair, "derivativesInstrumentPair should return %s's pair", instrument)
			assert.Equalf(t, instrument, pair.String(), "%s's pair should format as its instrument name", instrument)
		})
	}
	for _, instrument := range []string{"BTC-26DEC26", "BTC_USDC", "BTC-27NOV26-60000-C", "_USDC-PERPETUAL", "-PERPETUAL", "BTC_ETH_USDC-PERPETUAL", ""} {
		t.Run(subtestName(instrument), func(t *testing.T) {
			t.Parallel()
			_, _, err := derivativesInstrumentPair(instrument)
			assert.ErrorIsf(t, err, errDerivativesUnknownInstrument, "derivativesInstrumentPair should error for %q, which no asset trades", instrument)
		})
	}
}

func TestDerivativesNotificationChannel(t *testing.T) {
	t.Parallel()
	for channel, exp := range map[string]string{
		"book.BTC-PERPETUAL.100ms":                    ChannelDerivativesBook,
		"user.isolated.orders.SOL_USDC-PERPETUAL.raw": ChannelDerivativesUserIsolatedOrders,
		"user.orders.BTC-PERPETUAL.raw":               ChannelDerivativesUserOrders,
		"platform_state.public_methods_state":         ChannelDerivativesPublicMethodsState,
		"platform_state":                              ChannelDerivativesPlatformState,
		"user.trailing_orders.trigger_price.ETH":      ChannelDerivativesUserTrailingOrdersTriggerPrice,
		"nonsense.BTC-PERPETUAL":                      "",
	} {
		t.Run(subtestName(channel), func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, exp, derivativesNotificationChannel(channel), "derivativesNotificationChannel should name %s's channel", channel)
		})
	}
}

func TestDerivativesChannelOf(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		s       *subscription.Subscription
		ok      bool
		private bool
	}{
		{"coin margined ticker", &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.CoinMarginedFutures}, true, false},
		{"USDC margined orders", &subscription.Subscription{Channel: subscription.MyOrdersChannel, Asset: asset.USDCMarginedFutures}, true, true},
		{"portfolio", &subscription.Subscription{Channel: ChannelDerivativesUserPortfolio}, true, true},
		{"spot ticker", &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot}, false, false},
		{"account", &subscription.Subscription{Channel: subscription.MyAccountChannel}, false, false},
		{"unsupported channel", &subscription.Subscription{Channel: "nonsense"}, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ch, ok := derivativesChannelOf(tc.s)
			assert.Equalf(t, tc.ok, ok, "derivativesChannelOf should report %s on %s correctly", tc.s.Channel, tc.s.Asset)
			assert.Equalf(t, tc.private, ch.private, "derivativesChannelOf should report whether %s on %s is private", tc.s.Channel, tc.s.Asset)
		})
	}
}

// requestConnection is a connection answering each JSON-RPC request with a result, or an error, and recording the
// requests
type requestConnection struct {
	websocket.Connection
	result   json.RawMessage
	err      error
	requests []JSONRPCRequest
}

// SendMessageReturnResponse records a request and answers it with the connection's result
func (c *requestConnection) SendMessageReturnResponse(_ context.Context, _ request.EndpointLimit, _, payload any) ([]byte, error) {
	req, ok := payload.(*JSONRPCRequest)
	if !ok {
		return nil, errors.New("not a JSON-RPC request")
	}
	c.requests = append(c.requests, *req)
	if c.err != nil {
		return nil, c.err
	}
	return json.Marshal(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": c.result})
}

func TestWebsocketDerivativesManage(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	conn, err := ex.Websocket.CreateTestConnection(derivativesStreamsConnection)
	require.NoError(t, err, "CreateTestConnection must not error")
	require.NoError(t, ex.Websocket.TrackTestConnection(derivativesStreamsConnection, conn), "TrackTestConnection must not error")
	ticker := &subscription.Subscription{Channel: subscription.TickerChannel, QualifiedChannel: "ticker.BTC-PERPETUAL.100ms", Asset: asset.CoinMarginedFutures, Pairs: currency.Pairs{testCoinPerpetual}}
	book := &subscription.Subscription{Channel: subscription.OrderbookChannel, QualifiedChannel: "book.NOPE-PERPETUAL.100ms", Asset: asset.CoinMarginedFutures, Pairs: currency.Pairs{currency.NewPairWithDelimiter("NOPE", "PERPETUAL", "-")}}
	answer := &requestConnection{Connection: conn, result: json.RawMessage(`["ticker.BTC-PERPETUAL.100ms"]`)}

	err = ex.wsDerivativesSubscribe(t.Context(), answer, subscription.List{ticker, book})
	assert.ErrorIs(t, err, errDerivativesSubscriptionUnconfirmed, "a channel the gateway leaves out should be reported")
	require.Len(t, answer.requests, 1, "the channels must be subscribed in one request")
	assert.Equal(t, "public/subscribe", answer.requests[0].Method, "public channels should be subscribed through public/subscribe")
	assert.NotNil(t, ex.Websocket.GetSubscription(ticker), "the acknowledged channel should be recorded")
	assert.Nil(t, ex.Websocket.GetSubscription(book), "a channel left out should not be recorded")

	require.NoError(t, ex.wsDerivativesUnsubscribe(t.Context(), answer, subscription.List{ticker}), "wsDerivativesUnsubscribe must not error")
	assert.Equal(t, "public/unsubscribe", answer.requests[1].Method, "public channels should be unsubscribed through public/unsubscribe")
	assert.Nil(t, ex.Websocket.GetSubscription(ticker), "the unsubscribed channel should be removed")

	orders := &subscription.Subscription{Channel: subscription.MyOrdersChannel, QualifiedChannel: "user.orders.BTC-PERPETUAL.100ms", Asset: asset.CoinMarginedFutures, Pairs: currency.Pairs{testCoinPerpetual}, Authenticated: true}
	answer.result = json.RawMessage(`["user.orders.BTC-PERPETUAL.100ms"]`)
	require.NoError(t, ex.wsDerivativesSubscribe(t.Context(), answer, subscription.List{orders}), "wsDerivativesSubscribe must not error for a private channel")
	assert.Equal(t, "private/subscribe", answer.requests[2].Method, "private channels should be subscribed through private/subscribe")
	require.NoError(t, ex.wsDerivativesUnsubscribe(t.Context(), answer, subscription.List{orders}), "wsDerivativesUnsubscribe must not error for a private channel")
	assert.Equal(t, "private/unsubscribe", answer.requests[3].Method, "private channels should be unsubscribed through private/unsubscribe")

	assert.NoError(t, ex.wsDerivativesSubscribe(t.Context(), answer, nil), "wsDerivativesSubscribe should do nothing without subscriptions")
	failure := errors.New("send failure")
	answer.err = failure
	assert.ErrorIs(t, ex.wsDerivativesSubscribe(t.Context(), answer, subscription.List{ticker}), failure, "wsDerivativesSubscribe should return a failed request's error")
}

func TestWebsocketDerivativesAuthenticateFailure(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	conn := &requestConnection{err: errors.New("refused")}
	require.NoError(t, ex.wsDerivativesAuthenticate(t.Context(), conn), "a failed gateway authentication must not fail the connection")
	assert.True(t, ex.Websocket.CanUseAuthenticatedEndpoints(), "a failed gateway authentication should leave Advanced Trade's authenticated channels on")

	conn = &requestConnection{result: json.RawMessage(`null`)}
	assert.ErrorIs(t, ex.wsDerivativesSendAuth(t.Context(), conn), errDerivativesWebsocketAuthentication, "wsDerivativesSendAuth should error when no session is issued")
	ex.SetCredentials(nil)
	assert.Error(t, ex.wsDerivativesSendAuth(t.Context(), conn), "wsDerivativesSendAuth should error without credentials")
}

// checkedConnection is a closed reauthConnection counting how often it is checked for being connected
type checkedConnection struct {
	reauthConnection
	checks atomic.Int32
}

// IsConnected counts the check and reports the connection closed
func (c *checkedConnection) IsConnected() bool {
	c.checks.Add(1)
	return false
}

func TestWebsocketDerivativesScheduleReauthentication(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	conn := &reauthConnection{result: json.RawMessage(`{"access_token":"t","expires_in":0}`), connected: true}
	ex.wsDerivativesScheduleReauthentication(t.Context(), conn, ex.derivativesSessions.authenticated(conn, testAPIKey), 2*time.Millisecond)
	assert.Eventually(t, func() bool { return conn.sent() == 1 }, 30*time.Second, time.Millisecond, "a session should be authenticated again before it expires")

	closed := &reauthConnection{result: json.RawMessage(`{}`)}
	ex.wsDerivativesScheduleReauthentication(t.Context(), closed, ex.derivativesSessions.authenticated(closed, testAPIKey), time.Millisecond)
	assert.Never(t, func() bool { return closed.sent() != 0 }, 50*time.Millisecond, time.Millisecond, "a closed connection should not be authenticated again")

	t.Run("does not retry a renewal that asks for no retry", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketTestExchange(t)
		closed := new(checkedConnection)
		authentication := ex.derivativesSessions.authenticated(closed, testAPIKey)
		// A later authentication supersedes the renewal, which ends one wrongly retried at once when the test ends
		t.Cleanup(func() { ex.derivativesSessions.authenticated(closed, testAPIKey) })
		ex.wsDerivativesScheduleReauthentication(t.Context(), closed, authentication, time.Millisecond)
		require.Eventually(t, func() bool { return closed.checks.Load() != 0 }, 30*time.Second, time.Millisecond, "the renewal must check whether the connection is connected")
		assert.Never(t, func() bool { return closed.checks.Load() > 1 }, 50*time.Millisecond, time.Millisecond, "a renewal of a closed connection, which asks for no retry, should not be tried again")
		assert.Zero(t, closed.sent(), "a closed connection should not be authenticated again")
	})

	t.Run("retries a failed renewal after the retry interval", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketTestExchange(t)
		synctest.Test(t, func(t *testing.T) {
			refused := &reauthConnection{err: errors.New("refused"), connected: true}
			// A session lasting an hour is renewed 5 minutes before it expires
			ex.wsDerivativesScheduleReauthentication(t.Context(), refused, ex.derivativesSessions.authenticated(refused, testAPIKey), time.Hour)
			time.Sleep(55*time.Minute + time.Second)
			synctest.Wait()
			assert.Equal(t, 1, refused.sent(), "a session should be renewed before it expires")
			time.Sleep(derivativesReauthenticateRetry)
			synctest.Wait()
			assert.Equal(t, 2, refused.sent(), "a failed renewal should be tried again after the retry interval")
		})
	})
}

// switchedConnection is a reauthConnection a test can disconnect while its renewals run
type switchedConnection struct {
	reauthConnection
	up atomic.Bool
}

// IsConnected reports whether the connection is up
func (c *switchedConnection) IsConnected() bool { return c.up.Load() }

func TestWebsocketDerivativesSendAuth(t *testing.T) {
	t.Parallel()
	t.Run("caps a lifetime longer than a duration holds", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketTestExchange(t)
		conn := &switchedConnection{result: json.RawMessage(`{"access_token":"t","expires_in":18446744073709551615}`)}
		conn.up.Store(true)
		// Disconnecting ends any renewal, so a renewal that is wrongly due at once stops with the test
		t.Cleanup(func() { conn.up.Store(false) })
		require.NoError(t, ex.wsDerivativesSendAuth(t.Context(), conn), "the connection's authentication must succeed")
		// A lifetime longer than a duration holds is capped, rather than wrapping round to one already over
		assert.Never(t, func() bool { return conn.sent() > 1 }, 100*time.Millisecond, time.Millisecond, "a session whose lifetime is longer than a duration holds should not be renewed at once")
	})
	t.Run("renews after its caller's context ends", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketTestExchange(t)
		synctest.Test(t, func(t *testing.T) {
			conn := &reauthConnection{result: json.RawMessage(`{"access_token":"t","expires_in":600}`), connected: true}
			require.NoError(t, ex.wsDerivativesSendAuth(t.Context(), conn), "the connection's authentication must succeed")
			// AuthenticateWebsocket authenticates the connection again with its caller's context, which then ends
			ctx, cancel := context.WithCancel(t.Context())
			require.NoError(t, ex.wsDerivativesSendAuth(ctx, conn), "the caller's authentication must succeed")
			cancel()
			time.Sleep(5*time.Minute + time.Second)
			synctest.Wait()
			assert.Equal(t, 3, conn.sent(), "a connection's session should be renewed after the context of its last authentication's caller ends")
		})
	})
	t.Run("schedules one renewal per connection", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketTestExchange(t)
		synctest.Test(t, func(t *testing.T) {
			// Each session lasts 10 minutes, so it is renewed after 5
			conn := &reauthConnection{result: json.RawMessage(`{"access_token":"t","expires_in":600}`), connected: true}
			// AuthenticateWebsocket authenticates a connection already authenticated, which has its renewal pending
			for range 3 {
				require.NoError(t, ex.wsDerivativesSendAuth(t.Context(), conn), "wsDerivativesSendAuth must not error")
			}
			time.Sleep(5*time.Minute + time.Second)
			synctest.Wait()
			assert.Equal(t, 4, conn.sent(), "a connection should renew its session once, however often it was authenticated")
			time.Sleep(5 * time.Minute)
			synctest.Wait()
			assert.Equal(t, 5, conn.sent(), "each renewal should schedule the next, and only it")
		})
	})
}

func TestWebsocketDerivativesSupersededRenewalFailure(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	synctest.Test(t, func(t *testing.T) {
		conn := &firstAuthBlockedConnection{connected: true, blocked: make(chan struct{}), release: make(chan struct{})}
		authentication := ex.derivativesSessions.authenticated(conn, testAPIKey)
		ex.wsDerivativesScheduleReauthentication(t.Context(), conn, authentication, 2*time.Second)
		<-conn.blocked
		// The connection authenticates again while its renewal is under way, which then fails as its session expires
		require.NoError(t, ex.wsDerivativesSendAuth(t.Context(), conn), "the later authentication must succeed")
		close(conn.release)
		synctest.Wait()
		assert.Zero(t, conn.closes(), "a renewal a later authentication superseded should not close the connection")
		assert.Equal(t, testAPIKey, ex.derivativesSessions.authenticatedKey(conn), "a renewal a later authentication superseded should keep its session")
		// The later authentication's own renewal carries on
		time.Sleep(5*time.Minute + time.Second)
		synctest.Wait()
		assert.EqualValues(t, 3, conn.sends.Load(), "the later authentication's session should be renewed")
	})
}

// firstAuthBlockedConnection is a connected request connection whose first request waits for release, then fails, as
// a renewal timing out does, and whose later requests issue a session lasting 10 minutes
type firstAuthBlockedConnection struct {
	reauthConnection
	sends            atomic.Int32
	blocked, release chan struct{}
}

// SendMessageReturnResponse holds the first request until released, then refuses it, and answers the others
func (c *firstAuthBlockedConnection) SendMessageReturnResponse(_ context.Context, _ request.EndpointLimit, _, payload any) ([]byte, error) {
	if c.sends.Add(1) == 1 {
		close(c.blocked)
		<-c.release
		return nil, errors.New("renewal refused")
	}
	req, ok := payload.(*JSONRPCRequest)
	if !ok {
		return nil, errors.New("not a JSON-RPC request")
	}
	return json.Marshal(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": json.RawMessage(`{"access_token":"t","expires_in":600}`)})
}

func TestDerivativesWebsocketSessionsScheduleRenewal(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		var sessions derivativesWebsocketSessions
		var first, second, other, stale atomic.Int32
		conn, otherConn := &reauthConnection{connected: true}, &reauthConnection{connected: true}
		superseded := sessions.authenticated(conn, testAPIKey)
		authentication := sessions.authenticated(conn, testAPIKey)
		assert.True(t, sessions.superseded(conn, superseded), "superseded should report an authentication a later one superseded")
		assert.False(t, sessions.superseded(conn, authentication), "superseded should not report the connection's latest authentication")
		sessions.scheduleRenewal(conn, authentication, time.Minute, func() { first.Add(1) })
		sessions.scheduleRenewal(conn, authentication, time.Hour, func() { second.Add(1) })
		// A renewal of a superseded authentication is not scheduled, and leaves the latest one's renewal
		sessions.scheduleRenewal(conn, superseded, time.Minute, func() { stale.Add(1) })
		sessions.scheduleRenewal(otherConn, sessions.authenticated(otherConn, testAPIKey), time.Minute, func() { other.Add(1) })
		time.Sleep(2 * time.Minute)
		synctest.Wait()
		assert.Zero(t, first.Load(), "scheduleRenewal should stop the renewal it replaces")
		assert.Zero(t, stale.Load(), "scheduleRenewal should not schedule the renewal of a superseded authentication")
		assert.EqualValues(t, 1, other.Load(), "scheduleRenewal should leave another connection's renewal")
		time.Sleep(time.Hour)
		synctest.Wait()
		assert.EqualValues(t, 1, second.Load(), "scheduleRenewal should run the connection's latest renewal")
		assert.False(t, sessions.clearUnlessSuperseded(conn, superseded), "clearUnlessSuperseded should leave a session a later authentication began")
		assert.Equal(t, testAPIKey, sessions.authenticatedKey(conn), "clearUnlessSuperseded should keep the later authentication's key")
		assert.True(t, sessions.clearUnlessSuperseded(conn, authentication), "clearUnlessSuperseded should clear the latest authentication's session")
		assert.Empty(t, sessions.authenticatedKey(conn), "clearUnlessSuperseded should forget the key")
	})
}

func TestWebsocketDerivativesRenewSession(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	refused := &reauthConnection{err: errors.New("refused"), connected: true}
	authentication := ex.derivativesSessions.authenticated(refused, testAPIKey)
	assert.Equal(t, derivativesReauthenticateRetry, ex.wsDerivativesRenewSession(t.Context(), refused, authentication, time.Now().Add(time.Hour)), "a failed renewal should be tried again while the session has time left")
	assert.Empty(t, ex.derivativesSessions.authenticatedKey(refused), "a failed renewal should leave the connection unauthenticated, so the wrapper uses HTTP")
	assert.Zero(t, refused.closes(), "a failed renewal should not close a connection whose session has time left")
	assert.Zero(t, ex.wsDerivativesRenewSession(t.Context(), refused, authentication, time.Now().Add(time.Second)), "a renewal failing as the session expires should not be tried again")
	assert.Equal(t, 1, refused.closes(), "a connection whose session expires unrenewed should be closed, for the manager to reconnect it")

	renewed := &reauthConnection{result: json.RawMessage(`{"access_token":"t","expires_in":0}`), connected: true}
	assert.Zero(t, ex.wsDerivativesRenewSession(t.Context(), renewed, ex.derivativesSessions.authenticated(renewed, ""), time.Now().Add(time.Hour)), "a renewed session should not be tried again")
	assert.Equal(t, testAPIKey, ex.derivativesSessions.authenticatedKey(renewed), "a renewed session should leave the connection authenticated with the exchange's API key")

	// A renewal a later authentication superseded is not sent
	superseded := &reauthConnection{err: errors.New("refused"), connected: true}
	stale := ex.derivativesSessions.authenticated(superseded, testAPIKey)
	ex.derivativesSessions.authenticated(superseded, testAPIKey)
	assert.Zero(t, ex.wsDerivativesRenewSession(t.Context(), superseded, stale, time.Now()), "a superseded renewal should not be tried")
	assert.Zero(t, superseded.sent(), "a superseded renewal should not be sent")
	assert.Equal(t, testAPIKey, ex.derivativesSessions.authenticatedKey(superseded), "a superseded renewal should keep the later authentication's session")

	closed := &reauthConnection{err: errors.New("refused")}
	assert.Zero(t, ex.wsDerivativesRenewSession(t.Context(), closed, ex.derivativesSessions.authenticated(closed, ""), time.Now()), "a closed connection's session should not be renewed")
	assert.Zero(t, closed.sent(), "a closed connection should not be sent a renewal")
	assert.Zero(t, closed.closes(), "a closed connection should not be closed again")
}

func TestDerivativesWebsocketSessions(t *testing.T) {
	t.Parallel()
	var sessions derivativesWebsocketSessions
	first, second := &reauthConnection{connected: true}, &reauthConnection{connected: true}
	assert.Empty(t, sessions.authenticatedKey(first), "authenticatedKey should report no key for a connection never authenticated")
	sessions.set(first, "first-key")
	sessions.set(second, "")
	assert.Equal(t, "first-key", sessions.authenticatedKey(first), "authenticatedKey should report each connection's own session")
	assert.Empty(t, sessions.authenticatedKey(second), "authenticatedKey should report each connection's own session")
	first.connected = false
	sessions.set(second, "second-key")
	assert.Empty(t, sessions.authenticatedKey(first), "set should forget a connection that has closed")
	assert.Equal(t, "second-key", sessions.authenticatedKey(second), "set should record the connection's session")
}

// reauthConnection is a request connection reporting whether it is connected, safe for a timer to send on, and
// counting the close frames it sends
type reauthConnection struct {
	requestConnection
	connected   bool
	mu          sync.Mutex
	closeFrames int
}

// SendRawMessage counts a close frame
func (c *reauthConnection) SendRawMessage(_ context.Context, _ request.EndpointLimit, messageType int, _ []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if messageType == gws.CloseMessage {
		c.closeFrames++
	}
	return nil
}

// closes returns how many close frames the connection has sent
func (c *reauthConnection) closes() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closeFrames
}

// IsConnected reports whether the connection is connected
func (c *reauthConnection) IsConnected() bool { return c.connected }

// SendMessageReturnResponse records a request under the connection's lock
func (c *reauthConnection) SendMessageReturnResponse(ctx context.Context, epl request.EndpointLimit, signature, payload any) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.requestConnection.SendMessageReturnResponse(ctx, epl, signature, payload)
}

// sent returns how many requests the connection has sent
func (c *reauthConnection) sent() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.requests)
}

func TestWebsocketDerivativesAnswerHeartbeat(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	conn := &reauthConnection{result: json.RawMessage(`{"version":"1.2.26"}`), connected: true}
	require.NoError(t, ex.wsHandleDerivatives(t.Context(), conn, []byte(`{"jsonrpc":"2.0","method":"heartbeat","params":{"type":"test_request"}}`)), "a heartbeat test request must be handled")
	assert.Eventually(t, func() bool { return conn.sent() == 1 }, 30*time.Second, time.Millisecond, "a heartbeat test request should be answered with public/test")
	assert.Equal(t, "public/test", conn.requests[0].Method, "a heartbeat test request should be answered with public/test")
	require.NoError(t, ex.wsHandleDerivatives(t.Context(), conn, []byte(`{"jsonrpc":"2.0","method":"heartbeat","params":{"type":"heartbeat"}}`)), "a heartbeat must be handled")
	assert.Never(t, func() bool { return conn.sent() != 1 }, 20*time.Millisecond, time.Millisecond, "a heartbeat should need no answer")
}
