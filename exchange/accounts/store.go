package accounts

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/thrasher-corp/gocryptotrader/dispatch"
)

// Store contains accounts for exchanges.
type Store struct {
	exchangeAccounts exchangeMap
	mu               sync.Mutex
	mux              *dispatch.Mux
}

type exchangeMap map[exchange]*Accounts

type exchange interface {
	GetName() string
	GetCredentials(context.Context) (*Credentials, error)
}

type exchangeWrapper interface {
	GetBase() exchange
}

var global atomic.Pointer[Store]

// NewStore returns a new store with the default global dispatcher mux.
func NewStore() *Store {
	return &Store{
		exchangeAccounts: make(exchangeMap),
		mux:              dispatch.GetNewMux(nil),
	}
}

// GetStore returns the singleton accounts store for global use; Initialising if necessary.
func GetStore() *Store {
	if s := global.Load(); s != nil {
		return s
	}
	_ = global.CompareAndSwap(nil, NewStore())
	return global.Load()
}

// GetExchangeAccounts returns accounts for a specific exchange.
func (s *Store) GetExchangeAccounts(e exchange) (a *Accounts, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if w, ok := e.(exchangeWrapper); ok {
		// Because SetupDefaults is called on Base, it's easiest to just use the Base pointer as the key
		e = w.GetBase()
	}
	a, ok := s.exchangeAccounts[e]
	if !ok {
		a, err = NewAccounts(e, s.mux)
		if err != nil {
			return nil, err
		}
		s.exchangeAccounts[e] = a
	}
	return a, nil
}
