package okx

import (
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

type channelPairKey struct {
	key.PairAsset
	Channel string
}

type spotMarginEvaluator map[channelPairKey]bool

// NeedsOutboundSubscription checks if a subscription is needed based on existing spot/margin subscriptions. If asset
// is not spot or margin, it always returns true.
func (s spotMarginEvaluator) NeedsOutboundSubscription(pair currency.Pair, channel string, assetType asset.Item) (bool, error) {
	if assetType != asset.Spot && assetType != asset.Margin {
		return true, nil
	}
	need, ok := s[getChannelKey(pair, channel, assetType)]
	if !ok {
		return false, fmt.Errorf("%w: pair %s, channel %s, asset %s not found in evaluator", subscription.ErrNotFound, pair, channel, assetType)
	}
	return need, nil
}

func (s *spotMarginEvaluator) add(pair currency.Pair, channel string, assetType asset.Item, need bool) {
	(*s)[getChannelKey(pair, channel, assetType)] = need
}

func (s *spotMarginEvaluator) exists(pair currency.Pair, channel string, assetType asset.Item) bool {
	_, ok := (*s)[getChannelKey(pair, channel, assetType)]
	return ok
}

func getChannelKey(pair currency.Pair, channel string, assetType asset.Item) channelPairKey {
	return channelPairKey{
		PairAsset: key.PairAsset{Base: pair.Base.Item, Quote: pair.Quote.Item, Asset: assetType},
		Channel:   channel,
	}
}

// getSpotMarginEvaluator evaluates a list of subscriptions and determines which spot/margin subscriptions are needed to
// be sent outbound and returns a lookup table for evaluation. If the lists contain a spot and margin subscription for the same
// pair and channel, only one subscription is needed. If only one of the two asset types exist in the list,
// it checks existing subscriptions to determine if the subscription is needed based on operation (subscribe/unsubscribe).
func (e *Exchange) getSpotMarginEvaluator(subs []*subscription.Subscription) spotMarginEvaluator {
	eval := make(spotMarginEvaluator)
incoming:
	for i, s := range subs {
		if s.Asset != asset.Spot && s.Asset != asset.Margin {
			continue
		}

		if eval.exists(s.Pairs[0], s.Channel, s.Asset) {
			continue
		}

		// most straight forwards search path, when both subs are in the same subscription.List
		for _, s2 := range subs[i+1:] {
			if s2.Asset != asset.Spot && s2.Asset != asset.Margin {
				continue
			}
			if s.Pairs[0] == s2.Pairs[0] && s.Channel == s2.Channel {
				eval.add(s.Pairs[0], s.Channel, s.Asset, true)
				eval.add(s2.Pairs[0], s2.Channel, s2.Asset, false) // other asset type not needed
				continue incoming
			}
		}

		// invert asset type so that we can check for existing *potential* subscription
		inverse := s.Clone()
		switch s.Asset {
		case asset.Spot:
			inverse.Asset = asset.Margin
		case asset.Margin:
			inverse.Asset = asset.Spot
		}
		eval.add(s.Pairs[0], s.Channel, s.Asset, e.Websocket.GetSubscription(inverse) == nil)
	}
	return eval
}
