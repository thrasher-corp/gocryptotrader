package okx

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestGetSpotMarginEvaluator(t *testing.T) {
	t.Parallel()
	eval := e.getSpotMarginEvaluator(nil)
	require.Empty(t, eval)

	spotBTCUSDTSub := &subscription.Subscription{Asset: asset.Spot, Pairs: []currency.Pair{currency.NewBTCUSDT()}, Channel: "trades"}
	futuresBTCUSDTSub := &subscription.Subscription{Asset: asset.USDTMarginedFutures, Pairs: []currency.Pair{currency.NewBTCUSDT()}, Channel: "trades"}
	marginBTCUSDTSub := &subscription.Subscription{Asset: asset.Margin, Pairs: []currency.Pair{currency.NewBTCUSDT()}, Channel: "trades"}

	subs := []*subscription.Subscription{spotBTCUSDTSub, futuresBTCUSDTSub, marginBTCUSDTSub}
	eval = e.getSpotMarginEvaluator(subs)
	require.True(t, eval.exists(currency.NewBTCUSDT(), "trades", asset.Spot))
	require.False(t, eval.exists(currency.NewBTCUSDT(), "trades", asset.USDTMarginedFutures))
	require.True(t, eval.exists(currency.NewBTCUSDT(), "trades", asset.Margin))

	needed, err := eval.NeedsOutboundSubscription(currency.NewBTCUSDT(), "trades", asset.Spot)
	require.NoError(t, err)
	require.True(t, needed, "must be needed as no spot or margin subscription exists")
	needed, err = eval.NeedsOutboundSubscription(currency.NewBTCUSDT(), "trades", asset.USDTMarginedFutures)
	require.NoError(t, err)
	require.True(t, needed, "must be needed due to being a futures subscription")
	needed, err = eval.NeedsOutboundSubscription(currency.NewBTCUSDT(), "trades", asset.Margin)
	require.NoError(t, err)
	require.False(t, needed, "must not be needed as spot subscription will be used")

	subs = []*subscription.Subscription{spotBTCUSDTSub, futuresBTCUSDTSub}
	eval = e.getSpotMarginEvaluator(subs)
	needed, err = eval.NeedsOutboundSubscription(currency.NewBTCUSDT(), "trades", asset.Spot)
	require.NoError(t, err)
	require.True(t, needed, "must be needed as no margin subscription exists")
	needed, err = eval.NeedsOutboundSubscription(currency.NewBTCUSDT(), "trades", asset.USDTMarginedFutures)
	require.NoError(t, err)
	require.True(t, needed, "must be needed due to being a futures subscription")

	subs = []*subscription.Subscription{spotBTCUSDTSub, futuresBTCUSDTSub}
	err = e.Websocket.AddSuccessfulSubscriptions(nil, marginBTCUSDTSub)
	require.NoError(t, err)
	eval = e.getSpotMarginEvaluator(subs)
	needed, err = eval.NeedsOutboundSubscription(currency.NewBTCUSDT(), "trades", asset.Spot)
	require.NoError(t, err)
	require.False(t, needed, "must not be needed as margin subscription exists")
	needed, err = eval.NeedsOutboundSubscription(currency.NewBTCUSDT(), "trades", asset.USDTMarginedFutures)
	require.NoError(t, err)
	require.True(t, needed, "must be needed due to being a futures subscription")

	subs = []*subscription.Subscription{spotBTCUSDTSub, futuresBTCUSDTSub}
	eval = e.getSpotMarginEvaluator(subs)
	needed, err = eval.NeedsOutboundSubscription(currency.NewBTCUSDT(), "trades", asset.Spot)
	require.NoError(t, err)
	require.False(t, needed, "must not be needed as margin subscription exists and only the spot sub is being removed")
	needed, err = eval.NeedsOutboundSubscription(currency.NewBTCUSDT(), "trades", asset.USDTMarginedFutures)
	require.NoError(t, err)
	require.True(t, needed, "must be needed due to being a futures subscription")

	subs = []*subscription.Subscription{spotBTCUSDTSub, futuresBTCUSDTSub}
	err = e.Websocket.RemoveSubscriptions(nil, marginBTCUSDTSub)
	require.NoError(t, err)
	eval = e.getSpotMarginEvaluator(subs)
	needed, err = eval.NeedsOutboundSubscription(currency.NewBTCUSDT(), "trades", asset.Spot)
	require.NoError(t, err)
	require.True(t, needed, "must be needed as margin subscription does not exist and the subscription is no longer required")
	needed, err = eval.NeedsOutboundSubscription(currency.NewBTCUSDT(), "trades", asset.USDTMarginedFutures)
	require.NoError(t, err)
	require.True(t, needed, "must be needed due to being a futures subscription")

	subs = []*subscription.Subscription{marginBTCUSDTSub, futuresBTCUSDTSub}
	eval = e.getSpotMarginEvaluator(subs)
	needed, err = eval.NeedsOutboundSubscription(currency.NewBTCUSDT(), "trades", asset.Margin)
	require.NoError(t, err)
	require.True(t, needed, "must be needed as spot subscription does not exist and the subscription is no longer required")
	needed, err = eval.NeedsOutboundSubscription(currency.NewBTCUSDT(), "trades", asset.USDTMarginedFutures)
	require.NoError(t, err)
	require.True(t, needed, "must be needed due to being a futures subscription")
}

func TestNeedsOutboundSubscription(t *testing.T) {
	t.Parallel()
	eval := make(spotMarginEvaluator)
	eval.add(currency.NewBTCUSDT(), "trades", asset.Spot, true)
	require.True(t, eval.exists(currency.NewBTCUSDT(), "trades", asset.Spot), "subscription must exist")
	needed, err := eval.NeedsOutboundSubscription(currency.NewBTCUSDT(), "trades", asset.Spot)
	require.NoError(t, err)
	require.True(t, needed, "subscription must be needed")

	needed, err = eval.NeedsOutboundSubscription(currency.NewBTCUSDT(), "trades", asset.USDTMarginedFutures)
	require.NoError(t, err)
	require.True(t, needed, "subscription must be needed")

	needed, err = eval.NeedsOutboundSubscription(currency.NewBTCUSDT(), "trades", asset.Margin)
	require.ErrorIs(t, err, subscription.ErrNotFound)
	require.False(t, needed, "subscription must not be needed")
}

func TestOptionFamilyChannels(t *testing.T) {
	t.Parallel()

	require.Equal(t, channelOptionTrades, channelName(&subscription.Subscription{
		Channel: subscription.AllTradesChannel,
		Asset:   asset.Options,
	}), "all trades for options should map to option-trades")

	require.True(t, isInstFamilyChannel(&subscription.Subscription{
		Channel: subscription.AllTradesChannel,
		Asset:   asset.Options,
	}), "options all trades should be an instrument family channel")

	require.True(t, isInstFamilyChannel(&subscription.Subscription{
		Channel: channelOptSummary,
		Asset:   asset.Options,
	}), "option summary should be an instrument family channel")
}

func TestOptionInstrumentFamilyFromPair(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		pair     currency.Pair
		expected string
	}{
		{
			name:     "standard option pair",
			pair:     currency.NewPairWithDelimiter("BTC", "USD-230224-18000-C", "-"),
			expected: "BTC-USD",
		},
		{
			name:     "spot-like pair",
			pair:     currency.NewPairWithDelimiter("ETH", "USDT", "/"),
			expected: "ETH-USDT",
		},
		{
			name:     "empty pair",
			pair:     currency.EMPTYPAIR,
			expected: "",
		},
		{
			name:     "missing quote family",
			pair:     currency.NewPairWithDelimiter("BTC", "-230224-18000-C", "-"),
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.expected, optionInstrumentFamilyFromPair(tc.pair))
		})
	}
}

func TestGenerateSubscriptionsOptionTradesUseInstrumentFamily(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	require.NoError(t,
		ex.GetBase().SetPairs(currency.Pairs{currency.NewPairWithDelimiter("BTC", "USD", "-")}, asset.Options, false),
		"SetPairs available must not error")
	require.NoError(t,
		ex.GetBase().SetPairs(currency.Pairs{currency.NewPairWithDelimiter("BTC", "USD", "-")}, asset.Options, true),
		"SetPairs must not error")
	ex.Features.Subscriptions = subscription.List{
		{
			Channel: subscription.AllTradesChannel,
			Asset:   asset.Options,
		},
	}

	subs, err := ex.generateSubscriptions(true)
	require.NoError(t, err, "generateSubscriptions must not error")
	require.Len(t, subs, 1, "should generate one options all-trades subscription")
	require.Contains(t, subs[0].QualifiedChannel, `"channel":"option-trades"`)
	require.Contains(t, subs[0].QualifiedChannel, `"instFamily":"BTC-USD"`)
	require.Contains(t, subs[0].QualifiedChannel, `"instType":"OPTION"`)
	require.NotContains(t, subs[0].QualifiedChannel, `"instID"`, "option-trades should use instFamily instead of instID")
}

func TestGenerateSubscriptionsOptionSummaryUseInstrumentFamily(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	require.NoError(t,
		ex.GetBase().SetPairs(currency.Pairs{currency.NewPairWithDelimiter("BTC", "USD", "-")}, asset.Options, false),
		"SetPairs available must not error")
	require.NoError(t,
		ex.GetBase().SetPairs(currency.Pairs{currency.NewPairWithDelimiter("BTC", "USD", "-")}, asset.Options, true),
		"SetPairs must not error")
	ex.Features.Subscriptions = subscription.List{
		{
			Channel: subscription.TickerChannel,
			Asset:   asset.Options,
		},
	}

	subs, err := ex.generateSubscriptions(true)
	require.NoError(t, err, "generateSubscriptions must not error")
	require.Len(t, subs, 1, "should generate one options ticker subscription")
	require.Contains(t, subs[0].QualifiedChannel, `"channel":"opt-summary"`)
	require.Contains(t, subs[0].QualifiedChannel, `"instFamily":"BTC-USD"`)
	require.Contains(t, subs[0].QualifiedChannel, `"instType":"OPTION"`)
	require.NotContains(t, subs[0].QualifiedChannel, `"uly"`, "opt-summary should use instFamily instead of uly")
}

func TestChunkRequestsDeduplicatesOptionFamilyArguments(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	require.NoError(t, ex.GetBase().SetPairs(currency.Pairs{
		currency.NewPairWithDelimiter("BTC", "USD-230224-18000-C", "-"),
		currency.NewPairWithDelimiter("BTC", "USD-230224-19000-C", "-"),
	}, asset.Options, false), "SetPairs available must not error")
	require.NoError(t, ex.GetBase().SetPairs(currency.Pairs{
		currency.NewPairWithDelimiter("BTC", "USD-230224-18000-C", "-"),
		currency.NewPairWithDelimiter("BTC", "USD-230224-19000-C", "-"),
	}, asset.Options, true), "SetPairs must not error")

	ex.Features.Subscriptions = subscription.List{
		{
			Channel: subscription.AllTradesChannel,
			Asset:   asset.Options,
		},
	}
	subs, err := ex.generateSubscriptions(true)
	require.NoError(t, err, "generateSubscriptions must not error")
	require.Len(t, subs, 2, "template expansion should still track each input options pair")

	requests, err := ex.chunkRequests(subs, operationSubscribe)
	require.NoError(t, err, "chunkRequests must not error")
	require.NotEmpty(t, requests, "chunkRequests must return at least one request")
	require.Equal(t, 1, len(requests[0].Arguments), "only one outbound instFamily argument should be sent")
	require.Equal(t, 2, len(requests[0].subs), "all pair subscriptions should remain tracked")
}
