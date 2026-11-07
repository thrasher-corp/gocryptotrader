package bybit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

func TestGenerateLinearDefaultSubscriptions(t *testing.T) {
	t.Parallel()
	_, err := e.GenerateLinearDefaultSubscriptions(asset.OptionCombo)
	require.ErrorIs(t, err, asset.ErrInvalidAsset, "GenerateLinearDefaultSubscriptions must reject a non-linear asset")

	ex := websocketTestInstance(t)
	for a, p := range map[asset.Item]currency.Pair{asset.USDTMarginedFutures: currency.NewBTCUSDT(), asset.USDCMarginedFutures: currency.NewPair(currency.BTC, currency.PERP)} {
		require.NoErrorf(t, ex.CurrencyPairs.StorePairs(a, currency.Pairs{p}, false), "StorePairs must not error for %s", a)
		require.NoErrorf(t, ex.CurrencyPairs.StorePairs(a, currency.Pairs{p}, true), "StorePairs must not error for %s", a)
	}
	subs, err := ex.GenerateLinearDefaultSubscriptions(asset.USDTMarginedFutures)
	require.NoError(t, err, "GenerateLinearDefaultSubscriptions must not error")
	assert.ElementsMatch(t, []string{"orderbook.50.BTCUSDT", "tickers.BTCUSDT", "publicTrade.BTCUSDT"}, subs.QualifiedChannels(), "USDT margined subscriptions should cover the orderbook, ticker and trades")
	for _, s := range subs {
		assert.Equal(t, asset.USDTMarginedFutures, s.Asset, "subscriptions should be for the USDT margined asset")
	}
	subs, err = ex.GenerateLinearDefaultSubscriptions(asset.USDCMarginedFutures)
	require.NoError(t, err, "GenerateLinearDefaultSubscriptions must not error")
	assert.ElementsMatch(t, []string{"orderbook.50.BTCPERP", "tickers.BTCPERP", "publicTrade.BTCPERP"}, subs.QualifiedChannels(), "USDC margined subscriptions should cover the orderbook, ticker and trades")

	require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(asset.USDTMarginedFutures, false), "SetAssetEnabled must not error")
	subs, err = ex.GenerateLinearDefaultSubscriptions(asset.USDTMarginedFutures)
	require.NoError(t, err, "GenerateLinearDefaultSubscriptions must not error for a disabled asset")
	assert.Empty(t, subs, "a disabled asset should generate no subscriptions")
}

func TestCheckLinearAsset(t *testing.T) {
	t.Parallel()
	for _, a := range []asset.Item{asset.USDTMarginedFutures, asset.USDCMarginedFutures} {
		assert.NoErrorf(t, checkLinearAsset(a), "checkLinearAsset should accept %s, which the linear stream serves", a)
	}
	for _, a := range []asset.Item{asset.Spot, asset.CoinMarginedFutures, asset.Options, asset.OptionCombo} {
		assert.ErrorIsf(t, checkLinearAsset(a), asset.ErrInvalidAsset, "checkLinearAsset should refuse %s, which the linear stream does not serve", a)
	}
}

func TestLinearSubscribe(t *testing.T) {
	t.Parallel()
	ex := websocketTestInstance(t)
	subs := subscription.List{
		{Channel: subscription.CandlesChannel, Asset: asset.USDTMarginedFutures, Pairs: currency.Pairs{currency.NewBTCUSDT()}, Interval: 0},
		{Channel: chanInsurancePool, Asset: asset.USDTMarginedFutures},
		{Channel: chanAllLiquidation, Asset: asset.USDTMarginedFutures, Pairs: currency.Pairs{currency.NewBTCUSDT()}},
	}
	require.ErrorIs(t, ex.LinearSubscribe(t.Context(), &FixtureConnection{}, asset.OptionCombo, subs), asset.ErrInvalidAsset, "LinearSubscribe must reject a non-linear asset")
	conn := &FixtureConnection{}
	err := ex.LinearSubscribe(t.Context(), conn, asset.USDTMarginedFutures, subs)
	require.ErrorIs(t, err, subscription.ErrInvalidInterval, "LinearSubscribe must report a subscription without a topic")
	require.Len(t, conn.sent, 1, "LinearSubscribe must still send the valid subscriptions")
	req, ok := conn.sent[0].(*SubscriptionRequest)
	require.True(t, ok, "LinearSubscribe must send a SubscriptionRequest")
	assert.Equal(t, []string{"insurance.USDT", "allLiquidation.BTCUSDT"}, req.Arguments, "LinearSubscribe should send the topics of the valid subscriptions")
	assert.ElementsMatch(t, []string{"insurance.USDT", "allLiquidation.BTCUSDT"}, ex.Websocket.GetSubscriptions().QualifiedChannels(), "LinearSubscribe should store the valid subscriptions")
}

func TestLinearUnsubscribe(t *testing.T) {
	t.Parallel()
	ex := websocketTestInstance(t)
	subs := subscription.List{{Channel: chanADLAlert, Asset: asset.USDCMarginedFutures}}
	require.NoError(t, ex.LinearSubscribe(t.Context(), &FixtureConnection{}, asset.USDCMarginedFutures, subs), "LinearSubscribe must not error")
	require.ErrorIs(t, ex.LinearUnsubscribe(t.Context(), &FixtureConnection{}, asset.OptionCombo, subs), asset.ErrInvalidAsset, "LinearUnsubscribe must reject a non-linear asset")
	conn := &FixtureConnection{}
	require.NoError(t, ex.LinearUnsubscribe(t.Context(), conn, asset.USDCMarginedFutures, subs), "LinearUnsubscribe must not error")
	require.Len(t, conn.sent, 1, "LinearUnsubscribe must send one request")
	req, ok := conn.sent[0].(*SubscriptionRequest)
	require.True(t, ok, "LinearUnsubscribe must send a SubscriptionRequest")
	assert.Equal(t, unsubscribeOperation, req.Operation, "request operation should be unsubscribe")
	assert.Equal(t, []string{"adlAlert.USDC"}, req.Arguments, "LinearUnsubscribe should send the topic")
	assert.Empty(t, ex.Websocket.GetSubscriptions(), "LinearUnsubscribe should remove the subscription")
}
