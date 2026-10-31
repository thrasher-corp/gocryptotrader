package bybit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

func TestGenerateInverseDefaultSubscriptions(t *testing.T) {
	t.Parallel()
	ex := websocketTestInstance(t)
	require.NoError(t, ex.CurrencyPairs.StorePairs(asset.CoinMarginedFutures, currency.Pairs{currency.NewBTCUSD()}, false), "StorePairs must not error")
	require.NoError(t, ex.CurrencyPairs.StorePairs(asset.CoinMarginedFutures, currency.Pairs{currency.NewBTCUSD()}, true), "StorePairs must not error")
	subs, err := ex.GenerateInverseDefaultSubscriptions()
	require.NoError(t, err, "GenerateInverseDefaultSubscriptions must not error")
	assert.ElementsMatch(t, []string{"orderbook.50.BTCUSD", "tickers.BTCUSD", "publicTrade.BTCUSD"}, subs.QualifiedChannels(), "inverse subscriptions should cover the orderbook, ticker and trades")
	require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(asset.CoinMarginedFutures, false), "SetAssetEnabled must not error")
	subs, err = ex.GenerateInverseDefaultSubscriptions()
	require.NoError(t, err, "GenerateInverseDefaultSubscriptions must not error for a disabled asset")
	assert.Empty(t, subs, "a disabled asset should generate no subscriptions")
}

func TestInverseSubscribe(t *testing.T) {
	t.Parallel()
	ex := websocketTestInstance(t)
	conn := &FixtureConnection{}
	subs := subscription.List{{Channel: subscription.OrderbookChannel, Asset: asset.CoinMarginedFutures, Pairs: currency.Pairs{currency.NewBTCUSD()}, Levels: 200}}
	require.NoError(t, ex.InverseSubscribe(t.Context(), conn, subs), "InverseSubscribe must not error")
	require.Len(t, conn.sent, 1, "InverseSubscribe must send one request")
	req, ok := conn.sent[0].(*SubscriptionRequest)
	require.True(t, ok, "InverseSubscribe must send a SubscriptionRequest")
	assert.Equal(t, []string{"orderbook.200.BTCUSD"}, req.Arguments, "InverseSubscribe should send the topic")
	require.ErrorIs(t, ex.InverseSubscribe(t.Context(), conn, subscription.List{{Channel: chanPublicTicker, Asset: asset.CoinMarginedFutures}}), subscription.ErrNotSinglePair, "InverseSubscribe must reject a subscription without a pair")
}

func TestInverseUnsubscribe(t *testing.T) {
	t.Parallel()
	ex := websocketTestInstance(t)
	subs := subscription.List{{Channel: chanInsurancePool, Asset: asset.CoinMarginedFutures}}
	require.NoError(t, ex.InverseSubscribe(t.Context(), &FixtureConnection{}, subs), "InverseSubscribe must not error")
	conn := &FixtureConnection{}
	require.NoError(t, ex.InverseUnsubscribe(t.Context(), conn, subs), "InverseUnsubscribe must not error")
	require.Len(t, conn.sent, 1, "InverseUnsubscribe must send one request")
	req, ok := conn.sent[0].(*SubscriptionRequest)
	require.True(t, ok, "InverseUnsubscribe must send a SubscriptionRequest")
	assert.Equal(t, unsubscribeOperation, req.Operation, "request operation should be unsubscribe")
	assert.Equal(t, []string{"insurance.inverse"}, req.Arguments, "InverseUnsubscribe should send the topic")
	assert.Empty(t, ex.Websocket.GetSubscriptions(), "InverseUnsubscribe should remove the subscription")
}
