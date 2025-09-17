package engine

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deposit"
)

// vars related to the deposit address helpers
var (
	ErrDepositAddressStoreIsNil    = errors.New("deposit address store is nil")
	ErrDepositAddressNotFound      = errors.New("deposit address does not exist")
	errDepositAddressChainNotFound = errors.New("deposit address for specified chain not found")
	errNoDepositAddressesRetrieved = errors.New("no deposit addresses retrieved")
)

// DepositAddressManager manages the exchange deposit address store
type DepositAddressManager struct {
	m     sync.RWMutex
	store map[string]ExchangeDepositAddresses
}

// ExchangeDepositAddresses is a map of currencies to their deposit addresses
type ExchangeDepositAddresses map[string][]deposit.Address

// IsSynced returns whether or not the deposit address store has synced its data
func (m *DepositAddressManager) IsSynced() bool {
	if m.store == nil {
		return false
	}
	m.m.RLock()
	defer m.m.RUnlock()
	return len(m.store) > 0
}

// SetupDepositAddressManager returns a DepositAddressManager
func SetupDepositAddressManager() *DepositAddressManager {
	return &DepositAddressManager{
		store: make(map[string]ExchangeDepositAddresses),
	}
}

// GetDepositAddressByExchangeAndCurrency returns a deposit address for the specified exchange and cryptocurrency
// if it exists
func (m *DepositAddressManager) GetDepositAddressByExchangeAndCurrency(exchName, chain string, currencyItem currency.Code) (deposit.Address, error) {
	m.m.RLock()
	defer m.m.RUnlock()

	if len(m.store) == 0 {
		return deposit.Address{}, ErrDepositAddressStoreIsNil
	}

	r, ok := m.store[strings.ToUpper(exchName)]
	if !ok {
		return deposit.Address{}, ErrExchangeNotFound
	}

	addr, ok := r[strings.ToUpper(currencyItem.String())]
	if !ok {
		return deposit.Address{}, ErrDepositAddressNotFound
	}

	if len(addr) == 0 {
		return deposit.Address{}, errNoDepositAddressesRetrieved
	}

	if chain != "" {
		for x := range addr {
			if strings.EqualFold(addr[x].Chain, chain) {
				return addr[x], nil
			}
		}
		return deposit.Address{}, errDepositAddressChainNotFound
	}

	for x := range addr {
		if strings.EqualFold(addr[x].Chain, currencyItem.String()) {
			return addr[x], nil
		}
	}
	return addr[0], nil
}

// GetDepositAddressesByExchange returns a list of cryptocurrency addresses for the specified
// exchange if they exist
func (m *DepositAddressManager) GetDepositAddressesByExchange(exchName string) (ExchangeDepositAddresses, error) {
	m.m.RLock()
	defer m.m.RUnlock()

	if len(m.store) == 0 {
		return nil, ErrDepositAddressStoreIsNil
	}

	r, ok := m.store[strings.ToUpper(exchName)]
	if !ok {
		return nil, ErrDepositAddressNotFound
	}

	cpy := make(ExchangeDepositAddresses, len(r))
	for k, v := range r {
		cpy[k] = slices.Clone(v)
	}
	return cpy, nil
}

// Sync synchronises all deposit addresses
func (m *DepositAddressManager) Sync(addresses map[string]ExchangeDepositAddresses) error {
	if m == nil {
		return fmt.Errorf("deposit address manager %w", ErrNilSubsystem)
	}
	m.m.Lock()
	defer m.m.Unlock()
	if m.store == nil {
		return ErrDepositAddressStoreIsNil
	}

	for k, v := range addresses {
		r := make(ExchangeDepositAddresses)
		for w, x := range v {
			r[strings.ToUpper(w)] = x
		}
		m.store[strings.ToUpper(k)] = r
	}
	return nil
}
