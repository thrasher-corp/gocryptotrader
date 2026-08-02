package mexc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

// newFuturesTestExchange returns an isolated exchange instance with a single enabled futures
// pair, so data handler assertions do not race with the package level instance
func newFuturesTestExchange(t *testing.T) *Exchange {
	t.Helper()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "test exchange Setup must not error")
	spot, err := currency.NewPairFromString("BTCUSDT")
	require.NoError(t, err)
	futures, err := currency.NewPairFromString("BTC_USDT")
	require.NoError(t, err)
	require.NoError(t, ex.setEnabledPairs(spot, futures))
	return ex
}

// drainDataHandler collects everything currently buffered in the data handler
func drainDataHandler(t *testing.T, ex *Exchange) []any {
	t.Helper()
	out := []any{}
	for {
		select {
		case payload := <-ex.Websocket.DataHandler.C:
			out = append(out, payload.Data)
		default:
			return out
		}
	}
}

func TestIsFuturesSymbolChannel(t *testing.T) {
	t.Parallel()
	for _, channel := range []string{channelFTicker, channelFDeal, channelFDepthFull, channelFKline, channelFFundingRate, channelFIndexPrice, channelFFairPrice} {
		assert.Truef(t, isFuturesSymbolChannel(channel), "%s is subscribed per contract", channel)
	}
	for _, channel := range []string{channelFTickers, channelFPersonalOrder, channelFPersonalAssets, channelLogin} {
		assert.Falsef(t, isFuturesSymbolChannel(channel), "%s carries no symbol", channel)
	}
}

func TestFuturesSubscriptionParam(t *testing.T) {
	t.Parallel()
	ex := newFuturesTestExchange(t)
	pair := currency.NewPair(currency.BTC, currency.USDT)

	param, err := ex.futuresSubscriptionParam(&subscription.Subscription{QualifiedChannel: channelFTicker}, pair)
	require.NoError(t, err)
	assert.Equal(t, "BTC_USDT", param.Symbol, "ticker subscription must carry the symbol")

	param, err = ex.futuresSubscriptionParam(&subscription.Subscription{QualifiedChannel: channelFDepthFull}, pair)
	require.NoError(t, err)
	assert.Equal(t, "BTC_USDT", param.Symbol)
	assert.Equal(t, defaultFuturesDepthLevels, param.Limit, "depth subscription without levels must fall back to a default limit")

	param, err = ex.futuresSubscriptionParam(&subscription.Subscription{QualifiedChannel: channelFDepthFull, Levels: 5}, pair)
	require.NoError(t, err)
	assert.Equal(t, 5, param.Limit)

	param, err = ex.futuresSubscriptionParam(&subscription.Subscription{QualifiedChannel: channelFKline, Interval: kline.FifteenMin}, pair)
	require.NoError(t, err)
	assert.Equal(t, "BTC_USDT", param.Symbol)
	assert.Equal(t, "Min15", param.Interval)

	_, err = ex.futuresSubscriptionParam(&subscription.Subscription{QualifiedChannel: channelFKline, Interval: kline.SixMonth}, pair)
	require.Error(t, err, "an unsupported candle interval must error rather than subscribe without one")
}

func TestFuturesSubscriptionPayloads(t *testing.T) {
	t.Parallel()
	ex := newFuturesTestExchange(t)
	pairs := currency.Pairs{
		currency.NewPair(currency.BTC, currency.USDT),
		currency.NewPair(currency.ETH, currency.USDT),
	}

	payloads, err := ex.futuresSubscriptionPayloads(subscription.List{
		{Asset: asset.Futures, Channel: subscription.TickerChannel, QualifiedChannel: channelFTicker, Pairs: pairs},
		{Asset: asset.Futures, Channel: subscription.CandlesChannel, QualifiedChannel: channelFKline, Interval: kline.FifteenMin, Pairs: pairs[:1]},
		{Asset: asset.Futures, Channel: subscription.MyOrdersChannel, QualifiedChannel: channelFPersonalOrder},
	}, futuresSubscribeMethod)
	require.NoError(t, err)
	require.Len(t, payloads, 4, "two ticker pairs, one candle pair and one private channel")

	assert.Equal(t, "sub."+channelFTicker, payloads[0].Method)
	require.NotNil(t, payloads[0].Param)
	assert.Equal(t, "BTC_USDT", payloads[0].Param.Symbol)
	assert.Equal(t, "ETH_USDT", payloads[1].Param.Symbol)
	assert.Equal(t, "sub."+channelFKline, payloads[2].Method)
	assert.Equal(t, "Min15", payloads[2].Param.Interval)
	assert.Equal(t, "sub."+channelFPersonalOrder, payloads[3].Method)
	assert.Nil(t, payloads[3].Param, "a private channel carries no symbol parameters")

	for i := range payloads {
		if payloads[i].Param == nil {
			continue
		}
		assert.NotEmptyf(t, payloads[i].Param.Symbol, "payload %d for a per-contract channel must carry a symbol", i)
	}

	_, err = ex.futuresSubscriptionPayloads(subscription.List{
		{Asset: asset.Futures, Channel: subscription.TickerChannel, QualifiedChannel: channelFTicker},
	}, futuresSubscribeMethod)
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty, "a per-contract channel without pairs must error instead of subscribing without a symbol")

	_, err = ex.futuresSubscriptionPayloads(subscription.List{
		{Asset: asset.Futures, Channel: subscription.CandlesChannel, QualifiedChannel: channelFKline, Interval: kline.SixMonth, Pairs: pairs[:1]},
	}, futuresSubscribeMethod)
	require.Error(t, err, "an unsupported interval must abort payload construction")
}

// captureConn records the payloads a subscription attempt puts on the wire
type captureConn struct {
	websocket.Connection
	sent []any
}

func (c *captureConn) SendJSONMessage(_ context.Context, _ request.EndpointLimit, payload any) error {
	c.sent = append(c.sent, payload)
	return nil
}

func TestSubscribeFuturesRegistersSubscriptions(t *testing.T) {
	t.Parallel()
	ex := newFuturesTestExchange(t)
	conn := &captureConn{}
	subs := subscription.List{
		{Asset: asset.Futures, Channel: subscription.TickerChannel, QualifiedChannel: channelFTicker, Pairs: currency.Pairs{currency.NewPair(currency.BTC, currency.USDT)}},
	}
	require.NoError(t, ex.SubscribeFutures(t.Context(), conn, subs))
	require.Len(t, conn.sent, 1, "one payload per subscribed contract must reach the wire")

	// The manager tears the connection down with ErrSubscriptionsNotAdded unless the
	// subscriber registers what it subscribed to
	registered := ex.Websocket.GetSubscriptions()
	require.Len(t, registered, 1, "the subscription must be registered with the manager")
	assert.Equal(t, channelFTicker, registered[0].QualifiedChannel)

	require.NoError(t, ex.UnsubscribeFutures(t.Context(), conn, subs))
	assert.Empty(t, ex.Websocket.GetSubscriptions(), "unsubscribing must deregister the subscription")
}

func TestGenerateFuturesSubscriptionsUsesVenueChannels(t *testing.T) {
	t.Parallel()
	ex := newFuturesTestExchange(t)
	subs, err := ex.generateFuturesSubscriptions()
	require.NoError(t, err)
	require.NotEmpty(t, subs)
	for _, s := range subs {
		assert.NotEqualf(t, channelFTickers, s.QualifiedChannel, "%s must not resolve to the venue-wide broadcast channel", s.Channel)
		if isFuturesSymbolChannel(s.QualifiedChannel) {
			assert.NotEmptyf(t, s.Pairs, "per-contract channel %s must carry pairs", s.QualifiedChannel)
		}
	}
}

func TestProcessFuturesTickersFiltersUnsubscribedPairs(t *testing.T) {
	t.Parallel()
	ex := newFuturesTestExchange(t)
	// The venue broadcasts every contract on this channel; only the enabled pair may pass
	const payload = `[{"fairPrice":183.01,"lastPrice":183,"symbol":"BSV_USDT","volume24":200},
	  {"fairPrice":220.22,"lastPrice":220.4,"symbol":"BCH_USDT","volume24":200},
	  {"fairPrice":87000,"lastPrice":87001,"symbol":"BTC_USDT","volume24":10}]`
	require.NoError(t, ex.processFuturesTickers(t.Context(), []byte(payload)))

	received := drainDataHandler(t, ex)
	require.Len(t, received, 1, "only the enabled pair may reach the data handler")
	price, ok := received[0].(*ticker.Price)
	require.True(t, ok)
	assert.Equal(t, "BTC_USDT", price.Pair.String())
	assert.Equal(t, 87001.0, price.Last)

	// Negative: a malformed symbol must surface, not be swallowed
	require.Error(t, ex.processFuturesTickers(t.Context(), []byte(`[{"symbol":"","lastPrice":1}]`)))
}

func TestProcessFuturesTickerCarriesBBO(t *testing.T) {
	t.Parallel()
	ex := newFuturesTestExchange(t)
	// maxBidPrice/minAskPrice are the venue price-limit band, bid1/ask1 are the BBO
	const payload = `{"symbol":"LINK_USDT","lastPrice":14.022,"maxBidPrice":16.833,"minAskPrice":11.222,
	  "bid1":14.02,"ask1":14.021,"lower24Price":13.967,"high24Price":14.518,"timestamp":1746351275382}`
	require.NoError(t, ex.processFuturesTicker(t.Context(), []byte(payload)))

	received := drainDataHandler(t, ex)
	require.Len(t, received, 1)
	price, ok := received[0].(*ticker.Price)
	require.True(t, ok)
	assert.Equal(t, 14.021, price.Ask, "ask must be the best offer, not the price-limit band")
	assert.Equal(t, 14.02, price.Bid, "bid must be the best bid, not the price-limit band")
	assert.Positive(t, price.Ask)
	assert.Positive(t, price.Bid)
	assert.LessOrEqual(t, price.Bid, price.Ask, "a BBO cannot be crossed")

	// Negative: an unparsable symbol must error
	require.Error(t, ex.processFuturesTicker(t.Context(), []byte(`{"symbol":"","lastPrice":1}`)))
}

func TestWsHandleFuturesPong(t *testing.T) {
	t.Parallel()
	ex := newFuturesTestExchange(t)
	require.NoError(t, ex.WsHandleFuturesData(t.Context(), nil, []byte(`{"channel":"pong","data":1587442022003,"ts":1587442022003}`)))
	assert.Empty(t, drainDataHandler(t, ex), "a pong must not be relayed as data")
}

func TestWsHandleSpotPong(t *testing.T) {
	t.Parallel()
	ex := newFuturesTestExchange(t)
	require.NoError(t, ex.WsHandleData(t.Context(), nil, []byte(`{"id":0,"code":0,"msg":"PONG"}`)))
	assert.Empty(t, drainDataHandler(t, ex), "a pong must not be reported as an unhandled message")
}

func TestIsSpotPongMessage(t *testing.T) {
	t.Parallel()
	assert.True(t, isSpotPongMessage([]byte(`{"id":0,"code":0,"msg":"PONG"}`)))
	assert.True(t, isSpotPongMessage([]byte(`{"id":0,"code":0,"msg":"pong"}`)))

	assert.False(t, isSpotPongMessage([]byte(`{"id":42,"code":0,"msg":"SOMETHING_ELSE"}`)), "an unrelated message must not be swallowed as a pong")
	assert.False(t, isSpotPongMessage([]byte(`{"id":42,"code":0}`)), "a message without msg must not be swallowed as a pong")
	assert.False(t, isSpotPongMessage([]byte(`{"id":0,"code":0,"msg":"PONGO"}`)), "a partial match must not be swallowed as a pong")
}
