package kucoin

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

func TestCollapseSubscriptionList(t *testing.T) {
	t.Parallel()

	const totalSubscriptions = 101

	marketSubs := make(subscription.List, 0, totalSubscriptions)
	expectedPairs := make(currency.Pairs, 0, totalSubscriptions)
	expectedSuffixes := make([]string, 0, totalSubscriptions)
	for i := range totalSubscriptions {
		base := fmt.Sprintf("COIN%03d", i)
		suffix := base + "-USDT"
		pair := currency.NewPairWithDelimiter(base, "USDT", "-")
		marketSubs = append(marketSubs, &subscription.Subscription{
			Channel:          subscription.AllTradesChannel,
			Asset:            asset.Spot,
			Pairs:            currency.Pairs{pair},
			QualifiedChannel: marketMatchChannel + ":" + suffix,
		})
		expectedPairs = append(expectedPairs, pair)
		expectedSuffixes = append(expectedSuffixes, suffix)
	}

	accountSub := &subscription.Subscription{
		Channel:          accountBalanceChannel,
		Authenticated:    true,
		QualifiedChannel: accountBalanceChannel,
	}

	subs := append(subscription.List{}, marketSubs...)
	subs = append(subs, accountSub)

	collapsed := collapseSubscriptionList(subs)
	require.Len(t, collapsed, 3, "collapseSubscriptionList must create three collapsed batches")

	type batchResult struct {
		assoc *subscription.List
		sub   *subscription.Subscription
	}

	var hundredBatch *batchResult
	var singleBatch *batchResult
	var accountBatch *batchResult

	for assoc, sub := range collapsed {
		switch {
		case sub.QualifiedChannel == accountBalanceChannel:
			accountBatch = &batchResult{assoc: assoc, sub: sub}
		case strings.HasPrefix(sub.QualifiedChannel, marketMatchChannel+":"):
			switch len(*assoc) {
			case 100:
				hundredBatch = &batchResult{assoc: assoc, sub: sub}
			case 1:
				singleBatch = &batchResult{assoc: assoc, sub: sub}
			default:
				t.Fatalf("unexpected market batch size: %d", len(*assoc))
			}
		default:
			t.Fatalf("unexpected collapsed channel: %s", sub.QualifiedChannel)
		}
	}

	require.NotNil(t, hundredBatch, "the 100-item market batch must be present")
	require.NotNil(t, singleBatch, "the single-item market batch must be present")
	require.NotNil(t, accountBatch, "the pairless account batch must be present")

	assertCollapsedBatch(t, marketSubs[:100], expectedPairs[:100], expectedSuffixes[:100], hundredBatch.assoc, hundredBatch.sub)
	assertCollapsedBatch(t, marketSubs[100:], expectedPairs[100:], expectedSuffixes[100:], singleBatch.assoc, singleBatch.sub)

	require.Len(t, *accountBatch.assoc, 1, "the pairless account batch must preserve one original subscription")
	assert.Same(t, accountSub, (*accountBatch.assoc)[0], "the pairless account batch should preserve the original subscription pointer")
	assert.Equal(t, accountBalanceChannel, accountBatch.sub.Channel, "the pairless account batch should preserve the original channel")
	assert.Equal(t, accountBalanceChannel, accountBatch.sub.QualifiedChannel, "the pairless account batch should preserve the qualified channel")
	assert.Empty(t, accountBatch.sub.Pairs, "the pairless account batch should not gain pairs")
	assert.True(t, accountBatch.sub.Authenticated, "the pairless account batch should preserve authentication state")

	assert.Len(t, marketSubs[0].Pairs, 1, "the source subscription should remain unchanged after collapsing")
	assert.Equal(t, marketMatchChannel+":"+expectedSuffixes[0], marketSubs[0].QualifiedChannel, "the source subscription should keep its original qualified channel")
}

func assertCollapsedBatch(t *testing.T, expectedOriginal subscription.List, expectedPairs currency.Pairs, expectedSuffixes []string, assoc *subscription.List, got *subscription.Subscription) {
	t.Helper()

	require.NotNil(t, assoc, "the associated subscription list must not be nil")
	require.NotNil(t, got, "the collapsed subscription must not be nil")
	require.Len(t, *assoc, len(expectedOriginal), "the associated subscription list must preserve the original subscriptions")

	for i := range expectedOriginal {
		assert.Samef(t, expectedOriginal[i], (*assoc)[i], "associated subscription %d should match the original pointer", i)
	}

	assert.Equal(t, subscription.AllTradesChannel, got.Channel, "the collapsed subscription should preserve the channel")
	assert.Equal(t, asset.Spot, got.Asset, "the collapsed subscription should preserve the asset")
	assert.Equal(t, expectedPairs, got.Pairs, "the collapsed subscription should merge pairs in order")
	assert.Equal(t, marketMatchChannel+":"+strings.Join(expectedSuffixes, ","), got.QualifiedChannel, "the collapsed subscription should join the qualified channel suffixes")
	assert.False(t, got.Authenticated, "the collapsed market subscription should remain public")
}
