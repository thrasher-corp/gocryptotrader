package subscription

import (
	"encoding/json"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

// DefaultKey is the fallback key for AddSuccessfulSubscriptions
type DefaultKey struct {
	Channel string
	Pair    currency.Pair
	Asset   asset.Item
}

// State tracks the status of a subscription channel
type State uint8

const (
	UnknownState       State = iota // UnknownState subscription state is not registered, but doesn't imply Inactive
	SubscribingState                // SubscribingState means channel is in the process of subscribing
	SubscribedState                 // SubscribedState means the channel has finished a successful and acknowledged subscription
	UnsubscribingState              // UnsubscribingState means the channel has started to unsubscribe, but not yet confirmed

	TickerChannel    = "ticker"    // TickerChannel Subscription Type
	OrderbookChannel = "orderbook" // OrderbookChannel Subscription Type
	CandlesChannel   = "candles"   // CandlesChannel Subscription Type
	AllOrdersChannel = "allOrders" // AllOrdersChannel Subscription Type
	AllTradesChannel = "allTrades" // AllTradesChannel Subscription Type
	MyTradesChannel  = "myTrades"  // MyTradesChannel Subscription Type
	MyOrdersChannel  = "myOrders"  // MyOrdersChannel Subscription Type
)

// Subscription container for streaming subscriptions
type Subscription struct {
	Enabled       bool                   `json:"enabled"`
	Key           any                    `json:"-"`
	Channel       string                 `json:"channel,omitempty"`
	Pair          currency.Pair          `json:"pair,omitempty"`
	Asset         asset.Item             `json:"asset,omitempty"`
	Params        map[string]interface{} `json:"params,omitempty"`
	State         State                  `json:"-"`
	Interval      kline.Interval         `json:"interval,omitempty"`
	Levels        int                    `json:"levels,omitempty"`
	Authenticated bool                   `json:"authenticated,omitempty"`
}

// MarshalJSON generates a JSON representation of a Subscription, specifically for config writing
// The only reason it exists is to avoid having to make Pair a pointer, since that would be generally painful
// If Pair becomes a pointer, this method is redundant and should be removed
func (s *Subscription) MarshalJSON() ([]byte, error) {
	// None of the usual type embedding tricks seem to work for not emitting an nil Pair
	// The embedded type's Pair always fills the empty value
	type MaybePair struct {
		Enabled       bool                   `json:"enabled"`
		Channel       string                 `json:"channel,omitempty"`
		Asset         asset.Item             `json:"asset,omitempty"`
		Params        map[string]interface{} `json:"params,omitempty"`
		Interval      kline.Interval         `json:"interval,omitempty"`
		Levels        int                    `json:"levels,omitempty"`
		Authenticated bool                   `json:"authenticated,omitempty"`
		Pair          *currency.Pair         `json:"pair,omitempty"`
	}

	k := MaybePair{s.Enabled, s.Channel, s.Asset, s.Params, s.Interval, s.Levels, s.Authenticated, nil}
	if s.Pair != currency.EMPTYPAIR {
		k.Pair = &s.Pair
	}

	return json.Marshal(k)
}

// String implements the Stringer interface for Subscription, giving a human representation of the subscription
func (s *Subscription) String() string {
	return fmt.Sprintf("%s %s %s", s.Channel, s.Asset, s.Pair)
}

// EnsureKeyed sets the default key on a channel if it doesn't have one
// Returns key for convenience
func (s *Subscription) EnsureKeyed() any {
	if s.Key == nil {
		s.Key = DefaultKey{
			Channel: s.Channel,
			Asset:   s.Asset,
			Pair:    s.Pair,
		}
	}
	return s.Key
}
