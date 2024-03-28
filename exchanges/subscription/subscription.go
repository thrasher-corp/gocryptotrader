package subscription

import (
	"errors"
	"fmt"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

// State constants
const (
	InactiveState State = iota
	SubscribingState
	SubscribedState
	UnsubscribingState
)

// Ticker constants
const (
	TickerChannel    = "ticker"
	OrderbookChannel = "orderbook"
	CandlesChannel   = "candles"
	AllOrdersChannel = "allOrders"
	AllTradesChannel = "allTrades"
	MyTradesChannel  = "myTrades"
	MyOrdersChannel  = "myOrders"
)

// Public errors
var (
	ErrNotFound       = errors.New("subscription not found")
	ErrNotSinglePair  = errors.New("only single pair subscriptions expected")
	ErrInStateAlready = errors.New("subscription already in state")
	ErrInvalidState   = errors.New("invalid subscription state")
	ErrDuplicate      = errors.New("duplicate subscription")
)

// State tracks the status of a subscription channel
type State uint8

// Subscription container for streaming subscriptions
type Subscription struct {
	Enabled       bool                   `json:"enabled"`
	Key           any                    `json:"-"`
	Channel       string                 `json:"channel,omitempty"`
	Pairs         currency.Pairs         `json:"pairs,omitempty"`
	Asset         asset.Item             `json:"asset,omitempty"`
	Params        map[string]interface{} `json:"params,omitempty"`
	Interval      kline.Interval         `json:"interval,omitempty"`
	Levels        int                    `json:"levels,omitempty"`
	Authenticated bool                   `json:"authenticated,omitempty"`
	state         State
	m             sync.RWMutex
}

// MatchableKey interface should be implemented by Key types which want a more complex matching than a simple key equality check
type MatchableKey interface {
	Match(any) bool
}

// String implements the Stringer interface for Subscription, giving a human representation of the subscription
func (s *Subscription) String() string {
	p := s.Pairs.Format(currency.PairFormat{Uppercase: true, Delimiter: "/"})
	return fmt.Sprintf("%s %s %s", s.Channel, s.Asset, p.Join())
}

// State returns the subscription state
func (s *Subscription) State() State {
	s.m.RLock()
	defer s.m.RUnlock()
	return s.state
}

// SetState sets the subscription state
// Errors if already in that state or the new state is not valid
func (s *Subscription) SetState(state State) error {
	s.m.Lock()
	defer s.m.Unlock()
	if state == s.state {
		return ErrInStateAlready
	}
	if state > UnsubscribingState {
		return ErrInvalidState
	}
	s.state = state
	return nil
}

// EnsureKeyed returns the subscription key
// If no key exists then a pointer to the subscription itself will be used, since Subscriptions implement MatchableKey
func (s *Subscription) EnsureKeyed() any {
	if s.Key == nil {
		s.Key = s
	}
	return s.Key
}

// Match returns if the two keys match Channels, Assets, Pairs, Interval and Levels:
// s is the key being searched for, and eachSubKey is the key of every sub in the store
// Key Pairs comparison:
// 1) If s has Empty pairs then only a key without pairs match
// 2) If len(s.Pairs) >= 1 then a key which contain all the pairs match
// Such that a subscription for all enabled pairs will be matched when searching for any one pair
func (s *Subscription) Match(eachSubKey any) bool {
	var eachSub *Subscription
	switch v := eachSubKey.(type) {
	case *Subscription:
		eachSub = v
	case Subscription:
		eachSub = &v
	default:
		return false
	}

	switch {
	case eachSub.Channel != s.Channel,
		eachSub.Asset != s.Asset,
		// len(eachSub.Pairs) == 0 && len(s.Pairs) == 0: Okay; continue to next non-pairs check
		len(eachSub.Pairs) == 0 && len(s.Pairs) != 0,
		len(eachSub.Pairs) != 0 && len(s.Pairs) == 0,
		len(s.Pairs) != 0 && eachSub.Pairs.ContainsAll(s.Pairs, true) != nil,
		eachSub.Levels != s.Levels,
		eachSub.Interval != s.Interval:
		return false
	}

	return true
}

// Clone returns a copy of a subscription
func (s *Subscription) Clone() *Subscription {
	s.m.RLock()
	n := *s //nolint:govet // Replacing lock immediately below
	s.m.RUnlock()
	n.m = sync.RWMutex{}
	if s.Key == s {
		n.Key = &n
	}
	return &n
}
