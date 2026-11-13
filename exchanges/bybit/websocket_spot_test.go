package bybit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

func TestSpotSubscribe(t *testing.T) {
	t.Parallel()
	ex := websocketTestInstance(t)
	subs := subscription.List{{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: currency.Pairs{currency.NewBTCUSDT()}}}
	conn := &FixtureConnection{}
	require.NoError(t, ex.SpotSubscribe(t.Context(), conn, subs), "SpotSubscribe must not error")
	require.Len(t, conn.sent, 1, "SpotSubscribe must send one request")
	req, ok := conn.sent[0].(*SubscriptionRequest)
	require.True(t, ok, "SpotSubscribe must send a SubscriptionRequest")
	assert.Equal(t, subscribeOperation, req.Operation, "request operation should be subscribe")
	assert.Equal(t, []string{"tickers.BTCUSDT"}, req.Arguments, "SpotSubscribe should send the expanded topic")
	assert.Equal(t, []string{"tickers.BTCUSDT"}, ex.Websocket.GetSubscriptions().QualifiedChannels(), "SpotSubscribe should store the subscription")
	require.ErrorIs(t, ex.SpotSubscribe(t.Context(), conn, subscription.List{{Asset: asset.Spot, Channel: subscription.CandlesChannel, Pairs: currency.Pairs{currency.NewBTCUSDT()}}}), kline.ErrUnsupportedInterval, "SpotSubscribe must reject a subscription without a topic")
}

func TestHandleSpotSubscription(t *testing.T) {
	t.Parallel()
	ex := websocketTestInstance(t)
	btc := currency.Pairs{currency.NewBTCUSDT()}
	subs := subscription.List{
		{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: btc},
		{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel, Pairs: btc, Levels: 50},
		{Enabled: true, Asset: asset.Spot, Channel: subscription.CandlesChannel, Pairs: btc, Interval: kline.FiveMin},
	}
	topics := []string{"tickers.BTCUSDT", "orderbook.50.BTCUSDT", "kline.5.BTCUSDT"}
	conn := &FixtureConnection{}
	require.NoError(t, ex.handleSpotSubscription(t.Context(), conn, subscribeOperation, subs), "handleSpotSubscription must not error")
	require.Len(t, conn.sent, 1, "handleSpotSubscription must send one request")
	req, ok := conn.sent[0].(*SubscriptionRequest)
	require.True(t, ok, "handleSpotSubscription must send a SubscriptionRequest")
	assert.Equal(t, subscribeOperation, req.Operation, "handleSpotSubscription should send the operation")
	assert.Equal(t, topics, req.Arguments, "handleSpotSubscription should send each subscription's topic")
	stored := ex.Websocket.GetSubscriptions()
	assert.ElementsMatch(t, topics, stored.QualifiedChannels(), "handleSpotSubscription should store the subscriptions")

	conn = &FixtureConnection{}
	require.NoError(t, ex.handleSpotSubscription(t.Context(), conn, unsubscribeOperation, stored), "handleSpotSubscription must not error when unsubscribing")
	require.Len(t, conn.sent, 1, "handleSpotSubscription must send one unsubscribe request")
	req, ok = conn.sent[0].(*SubscriptionRequest)
	require.True(t, ok, "handleSpotSubscription must send a SubscriptionRequest")
	assert.Equal(t, unsubscribeOperation, req.Operation, "handleSpotSubscription should send the operation")
	assert.ElementsMatch(t, topics, req.Arguments, "handleSpotSubscription should send each stored subscription's topic")
	assert.Empty(t, ex.Websocket.GetSubscriptions(), "handleSpotSubscription should remove the unsubscribed subscriptions")

	conn = &FixtureConnection{}
	err := ex.handleSpotSubscription(t.Context(), conn, subscribeOperation, subscription.List{{Enabled: true, Asset: asset.Spot, Channel: subscription.CandlesChannel, Pairs: btc}})
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval, "handleSpotSubscription must refuse a subscription it cannot name a topic for")
	assert.Empty(t, conn.sent, "handleSpotSubscription should send nothing when a subscription has no topic")
}

func TestSpotUnsubscribe(t *testing.T) {
	t.Parallel()
	ex := websocketTestInstance(t)
	subs, err := subscription.List{{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: currency.Pairs{currency.NewBTCUSDT()}}}.ExpandTemplates(ex)
	require.NoError(t, err, "ExpandTemplates must not error")
	require.NoError(t, ex.SpotSubscribe(t.Context(), &FixtureConnection{}, subs), "SpotSubscribe must not error")
	conn := &FixtureConnection{}
	require.NoError(t, ex.SpotUnsubscribe(t.Context(), conn, subs), "SpotUnsubscribe must not error")
	require.Len(t, conn.sent, 1, "SpotUnsubscribe must send one request")
	req, ok := conn.sent[0].(*SubscriptionRequest)
	require.True(t, ok, "SpotUnsubscribe must send a SubscriptionRequest")
	assert.Equal(t, unsubscribeOperation, req.Operation, "request operation should be unsubscribe")
	assert.Empty(t, ex.Websocket.GetSubscriptions(), "SpotUnsubscribe should remove the subscription")
}
