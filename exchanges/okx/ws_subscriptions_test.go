package okx

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
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
