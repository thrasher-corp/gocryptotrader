package bybit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

// optionsReply answers an option subscription request applying every topic it lists
func optionsReply(r any) ([]byte, error) {
	req, ok := r.(*SubscriptionRequest)
	if !ok {
		return nil, errUnexpectedRequest
	}
	return json.Marshal(&SubscriptionResponse{Success: true, ConnectionID: "dat43eb07optjqfk6030-1k2ih", Type: commandResponseType, Data: SubscriptionTopics{FailTopics: []string{}, SuccessTopics: req.Arguments}})
}

func TestGenerateOptionsDefaultSubscriptions(t *testing.T) {
	t.Parallel()
	ex := websocketTestInstance(t)
	require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Options, currency.Pairs{fixtureOptionPair}, false), "StorePairs must not error")
	require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Options, currency.Pairs{fixtureOptionPair}, true), "StorePairs must not error")
	subs, err := ex.GenerateOptionsDefaultSubscriptions()
	require.NoError(t, err, "GenerateOptionsDefaultSubscriptions must not error")
	assert.ElementsMatch(t, []string{"orderbook.25.BTC-24SEP27-170000-P-USDT", "tickers.BTC-24SEP27-170000-P-USDT", "publicTrade.BTC"}, subs.QualifiedChannels(), "option subscriptions should take 25 levels and trades by base coin")
	require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(asset.Options, false), "SetAssetEnabled must not error")
	subs, err = ex.GenerateOptionsDefaultSubscriptions()
	require.NoError(t, err, "GenerateOptionsDefaultSubscriptions must not error for a disabled asset")
	assert.Empty(t, subs, "a disabled asset should generate no subscriptions")
}

func TestOptionsSubscribe(t *testing.T) {
	t.Parallel()
	ex := websocketTestInstance(t)
	conn := &FixtureConnection{reply: optionsReply}
	subs := subscription.List{{Channel: subscription.OrderbookChannel, Asset: asset.Options, Pairs: currency.Pairs{fixtureOptionPair}, Levels: 100}}
	require.NoError(t, ex.OptionsSubscribe(t.Context(), conn, subs), "OptionsSubscribe must not error")
	require.Len(t, conn.sent, 1, "OptionsSubscribe must send one request")
	req, ok := conn.sent[0].(*SubscriptionRequest)
	require.True(t, ok, "OptionsSubscribe must send a SubscriptionRequest")
	assert.Equal(t, []string{"orderbook.100.BTC-24SEP27-170000-P-USDT"}, req.Arguments, "OptionsSubscribe should send the topic")
	assert.Equal(t, []string{"orderbook.100.BTC-24SEP27-170000-P-USDT"}, ex.Websocket.GetSubscriptions().QualifiedChannels(), "OptionsSubscribe should store the subscription")
}

func TestOptionsUnsubscribe(t *testing.T) {
	t.Parallel()
	ex := websocketTestInstance(t)
	subs := subscription.List{{Channel: subscription.AllTradesChannel, Asset: asset.Options, Pairs: currency.Pairs{fixtureOptionPair}}}
	require.NoError(t, ex.OptionsSubscribe(t.Context(), &FixtureConnection{reply: optionsReply}, subs), "OptionsSubscribe must not error")
	conn := &FixtureConnection{reply: optionsReply}
	require.NoError(t, ex.OptionsUnsubscribe(t.Context(), conn, subs), "OptionsUnsubscribe must not error")
	require.Len(t, conn.sent, 1, "OptionsUnsubscribe must send one request")
	req, ok := conn.sent[0].(*SubscriptionRequest)
	require.True(t, ok, "OptionsUnsubscribe must send a SubscriptionRequest")
	assert.Equal(t, unsubscribeOperation, req.Operation, "request operation should be unsubscribe")
	assert.Equal(t, []string{"publicTrade.BTC"}, req.Arguments, "OptionsUnsubscribe should send the base coin trade topic")
	assert.Empty(t, ex.Websocket.GetSubscriptions(), "OptionsUnsubscribe should remove the subscription")
}

func TestOptionsSharedTradeTopic(t *testing.T) {
	t.Parallel()
	trades := func(contract string) subscription.List {
		return subscription.List{{Channel: subscription.AllTradesChannel, Asset: asset.Options, Pairs: currency.Pairs{currency.NewPairWithDelimiter("BTC", contract, currency.DashDelimiter)}}}
	}
	ex := websocketTestInstance(t)
	conn := &FixtureConnection{reply: optionsReply}
	put, call := trades("24SEP27-170000-P-USDT"), trades("24SEP27-170000-C-USDT")
	require.NoError(t, ex.OptionsSubscribe(t.Context(), conn, put), "OptionsSubscribe must not error")
	require.NoError(t, ex.OptionsSubscribe(t.Context(), conn, call), "OptionsSubscribe must not error for another contract of the base coin")
	require.Len(t, conn.sent, 1, "OptionsSubscribe must subscribe a base coin's trade topic once")
	assert.Len(t, ex.Websocket.GetSubscriptions(), 2, "OptionsSubscribe should store both contracts' subscriptions")

	// The topic carries both contracts' trades, so it stays until neither subscription holds it
	require.NoError(t, ex.OptionsUnsubscribe(t.Context(), conn, put), "OptionsUnsubscribe must not error")
	assert.Len(t, conn.sent, 1, "OptionsUnsubscribe should not unsubscribe a topic another subscription holds")
	assert.Nil(t, ex.Websocket.GetSubscription(put[0]), "OptionsUnsubscribe should remove the unsubscribed contract")
	assert.Same(t, call[0], ex.Websocket.GetSubscription(call[0]), "OptionsUnsubscribe should keep the other contract's subscription")

	require.NoError(t, ex.OptionsUnsubscribe(t.Context(), conn, call), "OptionsUnsubscribe must not error for the last subscription")
	require.Len(t, conn.sent, 2, "OptionsUnsubscribe must unsubscribe the topic with its last subscription")
	req, ok := conn.sent[1].(*SubscriptionRequest)
	require.True(t, ok, "OptionsUnsubscribe must send a SubscriptionRequest")
	assert.Equal(t, unsubscribeOperation, req.Operation, "request operation should be unsubscribe")
	assert.Equal(t, []string{"publicTrade.BTC"}, req.Arguments, "OptionsUnsubscribe should send the base coin trade topic")
	assert.Empty(t, ex.Websocket.GetSubscriptions(), "OptionsUnsubscribe should remove the last subscription")
}
