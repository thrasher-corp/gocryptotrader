package gateio

import (
	"context"
	"fmt"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/log"
)

type wsOBResubManager struct {
	lookup map[key.PairAsset]bool
	m      sync.RWMutex
}

func newWSOBResubManager() *wsOBResubManager {
	return &wsOBResubManager{lookup: make(map[key.PairAsset]bool)}
}

// IsResubscribing checks if a subscription is currently being resubscribed
func (m *wsOBResubManager) IsResubscribing(pair currency.Pair, a asset.Item) bool {
	m.m.RLock()
	defer m.m.RUnlock()
	return m.lookup[key.PairAsset{Base: pair.Base.Item, Quote: pair.Quote.Item, Asset: a}]
}

// Resubscribe marks a subscription as resubscribing and starts the unsubscribe/resubscribe process
func (m *wsOBResubManager) Resubscribe(ctx context.Context, e *Exchange, conn websocket.Connection, qualifiedChannel string, pair currency.Pair, a asset.Item) error {
	if err := e.Websocket.Orderbook.InvalidateOrderbook(pair, a); err != nil {
		return err
	}

	sub := e.Websocket.GetSubscription(qualifiedChannelKey{&subscription.Subscription{QualifiedChannel: qualifiedChannel}})
	if sub == nil {
		return fmt.Errorf("%w: %q", subscription.ErrNotFound, qualifiedChannel)
	}

	m.m.Lock()
	defer m.m.Unlock()

	m.lookup[key.PairAsset{Base: pair.Base.Item, Quote: pair.Quote.Item, Asset: a}] = true

	go func() { // Has to be called in routine to not impede websocket throughput
		if err := e.Websocket.ResubscribeToChannel(ctx, conn, sub); err != nil {
			m.CompletedResubscribe(pair, a) // Ensure we clear the map entry on failure too
			log.Errorf(log.ExchangeSys, "Failed to resubscribe to channel %q: %v", qualifiedChannel, err)
		}
	}()

	return nil
}

// CompletedResubscribe removes a subscription from the resubscribing map
func (m *wsOBResubManager) CompletedResubscribe(pair currency.Pair, a asset.Item) {
	m.m.Lock()
	defer m.m.Unlock()
	delete(m.lookup, key.PairAsset{Base: pair.Base.Item, Quote: pair.Quote.Item, Asset: a})
}

type qualifiedChannelKey struct {
	*subscription.Subscription
}

func (k qualifiedChannelKey) Match(eachKey subscription.MatchableKey) bool {
	return k.Subscription.QualifiedChannel == eachKey.GetSubscription().QualifiedChannel
}

func (k qualifiedChannelKey) GetSubscription() *subscription.Subscription {
	return k.Subscription
}
