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
	ResubscribingState
	UnsubscribingState
	UnsubscribedState
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
	if state > UnsubscribedState {
		return ErrInvalidState
	}
	s.state = state
	return nil
}

// SetKey does what it says on the tin safely for concurrency
func (s *Subscription) SetKey(key any) {
	s.m.Lock()
	defer s.m.Unlock()
	s.Key = key
}

// EnsureKeyed returns the subscription key
// If no key exists then ExactKey will be used
func (s *Subscription) EnsureKeyed() any {
	if s.Key == nil {
		s.Key = &ExactKey{s}
	}
	return s.Key
}

// Clone returns a copy of a subscription
// Key is set to nil, because any original key is meaningless on a clone
func (s *Subscription) Clone() *Subscription {
	s.m.RLock()
	n := *s //nolint:govet // Replacing lock immediately below
	s.m.RUnlock()
	n.m = sync.RWMutex{}
	n.Key = nil
	return &n
}

// SetPairs does what it says on the tin safely for currency
func (s *Subscription) SetPairs(pairs currency.Pairs) {
	s.m.Lock()
	s.Pairs = pairs
	s.m.Unlock()
}

// AddPairs does what it says on the tin safely for concurrency
func (s *Subscription) AddPairs(pairs ...currency.Pair) {
	s.m.Lock()
	for _, p := range pairs {
		s.Pairs = s.Pairs.Add(p)
	}
	s.m.Unlock()
}
