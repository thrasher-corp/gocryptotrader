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
