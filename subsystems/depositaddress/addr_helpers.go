package depositaddress

import (
	"errors"
	"strings"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/subsystems/exchangemanager"
)

// Store stores a list of exchange deposit addresses
type Store struct {
	m     sync.Mutex
	Store map[string]map[string]string
}

// Manager manages the exchange deposit address store
type Manager struct {
	Store Store
}

// vars related to the deposit address helpers
var (
	ErrDepositAddressStoreIsNil = errors.New("deposit address store is nil")
	ErrDepositAddressNotFound   = errors.New("deposit address does not exist")
)

// Seed seeds the deposit address store
func (s *Store) Seed(coinData map[string]map[string]string) {
	s.m.Lock()
	defer s.m.Unlock()
	if s.Store == nil {
		s.Store = make(map[string]map[string]string)
	}

	for k, v := range coinData {
		r := make(map[string]string)
		for w, x := range v {
			r[strings.ToUpper(w)] = x
		}
		s.Store[strings.ToUpper(k)] = r
	}
}

// GetDepositAddress returns a deposit address based on the specified item
func (s *Store) GetDepositAddress(exchName string, item currency.Code) (string, error) {
	s.m.Lock()
	defer s.m.Unlock()

	if len(s.Store) == 0 {
		return "", ErrDepositAddressStoreIsNil
	}

	r, ok := s.Store[strings.ToUpper(exchName)]
	if !ok {
		return "", exchangemanager.ErrExchangeNotFound
	}

	addr, ok := r[strings.ToUpper(item.String())]
	if !ok {
		return "", ErrDepositAddressNotFound
	}

	return addr, nil
}

// GetDepositAddresses returns a list of stored deposit addresses
func (s *Store) GetDepositAddresses(exchName string) (map[string]string, error) {
	s.m.Lock()
	defer s.m.Unlock()

	if len(s.Store) == 0 {
		return nil, ErrDepositAddressStoreIsNil
	}

	r, ok := s.Store[strings.ToUpper(exchName)]
	if !ok {
		return nil, ErrDepositAddressNotFound
	}

	return r, nil
}

// GetDepositAddressByExchange returns a deposit address for the specified exchange and cryptocurrency
// if it exists
func (m *Manager) GetDepositAddressByExchange(exchName string, currencyItem currency.Code) (string, error) {
	return m.Store.GetDepositAddress(exchName, currencyItem)
}

// GetDepositAddressesByExchange returns a list of cryptocurrency addresses for the specified
// exchange if they exist
func (m *Manager) GetDepositAddressesByExchange(exchName string) (map[string]string, error) {
	return m.Store.GetDepositAddresses(exchName)
}

// Sync synchronises all deposit addresses
func (m *Manager) Sync(addresses map[string]map[string]string) {
	m.Store.Seed(addresses)
}
