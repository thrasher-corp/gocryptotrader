package subscription

import (
	"errors"
	"fmt"
	"maps"
	"slices"
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

// Channel constants
const (
	TickerChannel    = "ticker"
	OrderbookChannel = "orderbook"
	CandlesChannel   = "candles"
	AllOrdersChannel = "allOrders"
	AllTradesChannel = "allTrades"
	MyTradesChannel  = "myTrades"
	MyOrdersChannel  = "myOrders"
	MyWalletChannel  = "myWallet"
	MyAccountChannel = "myAccount"
	HeartbeatChannel = "heartbeat"
)

// Public errors
var (
	ErrNotFound              = errors.New("subscription not found")
	ErrNotSinglePair         = errors.New("only single pair subscriptions expected")
	ErrBatchingNotSupported  = errors.New("subscription batching not supported")
	ErrInStateAlready        = errors.New("subscription already in state")
	ErrInvalidState          = errors.New("invalid subscription state")
	ErrDuplicate             = errors.New("duplicate subscription")
	ErrUseConstChannelName   = errors.New("must use standard channel name constants")
	ErrNotSupported          = errors.New("subscription channel not supported")
	ErrExclusiveSubscription = errors.New("exclusive subscription detected")
	ErrInvalidInterval       = errors.New("invalid interval")
	ErrInvalidLevel          = errors.New("invalid level")
)

// State tracks the status of a subscription channel
type State uint8

// ListValidator validates a list of subscriptions, this is optionally handled through expand templates method
type ListValidator interface {
	ValidateSubscriptions(List) error
}

// Subscription container for streaming subscriptions
type Subscription struct {
	Enabled          bool           `json:"enabled"`
	Key              any            `json:"-"`
	Channel          string         `json:"channel,omitempty"`
	Pairs            currency.Pairs `json:"pairs,omitempty"`
	Asset            asset.Item     `json:"asset,omitempty"`
	Params           map[string]any `json:"params,omitempty"`
	Interval         kline.Interval `json:"interval,omitempty"`
	Levels           int            `json:"levels,omitempty"`
	Authenticated    bool           `json:"authenticated,omitempty"`
	QualifiedChannel string         `json:"-"`
	state            State
	m                sync.RWMutex
}

// String implements Stringer, and aims to informatively and uniquely identify a subscription for errors and information
// returns a string of the subscription key by delegating to MatchableKey.String() when possible
// If the key is not a MatchableKey then both the key and an ExactKey.String() will be returned; e.g. 1137: spot MyTrades
func (s *Subscription) String() string {
	key := s.EnsureKeyed()
	s.m.RLock()
	defer s.m.RUnlock()
	if k, ok := key.(MatchableKey); ok {
		return k.String()
	}
	return fmt.Sprintf("%v: %s", key, ExactKey{s}.String())
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
	// Juggle RLock/WLock to minimize concurrent bottleneck for hottest path
	s.m.RLock()
	if s.Key != nil {
		defer s.m.RUnlock()
		return s.Key
	}
	s.m.RUnlock()
	s.m.Lock()
	defer s.m.Unlock()
	if s.Key == nil { // Ensure race hasn't updated Key whilst we swapped locks
		s.Key = &ExactKey{s}
	}
	return s.Key
}

// Clone returns a copy of a subscription
// Key is set to nil, because most Key types contain a pointer to the subscription, and because the clone isn't added to the store yet
// QualifiedChannel is not copied because it's expected that the contributing fields will be changed
// Users should allow a default key to be assigned on AddSubscription or can SetKey as necessary
func (s *Subscription) Clone() *Subscription {
	s.m.RLock()
	c := &Subscription{
		Key:              nil,
		Enabled:          s.Enabled,
		Channel:          s.Channel,
		Asset:            s.Asset,
		Params:           maps.Clone(s.Params),
		Interval:         s.Interval,
		Levels:           s.Levels,
		Authenticated:    s.Authenticated,
		state:            s.state,
		Pairs:            slices.Clone(s.Pairs),
		QualifiedChannel: s.QualifiedChannel,
	}
	s.m.RUnlock()
	return c
}

// SetPairs does what it says on the tin safely for concurrency
func (s *Subscription) SetPairs(pairs currency.Pairs) {
	s.m.Lock()
	s.Pairs = pairs
	s.m.Unlock()
}

// AddPairs does what it says on the tin safely for concurrency
func (s *Subscription) AddPairs(pairs ...currency.Pair) {
	if len(pairs) == 0 {
		return
	}
	s.m.Lock()
	s.Pairs = s.Pairs.Add(pairs...)
	s.m.Unlock()
}
