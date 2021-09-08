package engine

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deposit"
)

// vars related to the deposit address helpers
var (
	ErrDepositAddressStoreIsNil = errors.New("deposit address store is nil")
	ErrDepositAddressNotFound   = errors.New("deposit address does not exist")
)

// DepositAddressManager manages the exchange deposit address store
type DepositAddressManager struct {
	m     sync.Mutex
	store map[string]map[string][]DepositAddressExtended
}

type DepositAddressExtended struct {
	deposit.Address
	Chain string
}

// SetupDepositAddressManager returns a DepositAddressManager
func SetupDepositAddressManager() *DepositAddressManager {
	return &DepositAddressManager{
		store: make(map[string]map[string][]DepositAddressExtended),
	}
}

// GetDepositAddressByExchangeAndCurrency returns a deposit address for the specified exchange and cryptocurrency
// if it exists
func (m *DepositAddressManager) GetDepositAddressByExchangeAndCurrency(exchName, chain string, currencyItem currency.Code) (DepositAddressExtended, error) {
	m.m.Lock()
	defer m.m.Unlock()

	if len(m.store) == 0 {
		return DepositAddressExtended{}, ErrDepositAddressStoreIsNil
	}

	r, ok := m.store[strings.ToUpper(exchName)]
	if !ok {
		return DepositAddressExtended{}, ErrExchangeNotFound
	}

	addr, ok := r[strings.ToUpper(currencyItem.String())]
	if !ok {
		return DepositAddressExtended{}, ErrDepositAddressNotFound
	}

	if len(addr) == 0 {
		return DepositAddressExtended{}, errors.New("no addresses retrieved")
	}
	if chain != "" {
		for x := range addr {
			if strings.EqualFold(addr[x].Chain, chain) {
				return addr[x], nil
			}
		}
		return DepositAddressExtended{}, errors.New("deposit address for specified chain not found")
	}
	return addr[0], nil
}

// GetDepositAddressesByExchange returns a list of cryptocurrency addresses for the specified
// exchange if they exist
func (m *DepositAddressManager) GetDepositAddressesByExchange(exchName string) (map[string][]DepositAddressExtended, error) {
	m.m.Lock()
	defer m.m.Unlock()

	if len(m.store) == 0 {
		return nil, ErrDepositAddressStoreIsNil
	}

	r, ok := m.store[strings.ToUpper(exchName)]
	if !ok {
		return nil, ErrDepositAddressNotFound
	}

	return r, nil
}

// Sync synchronises all deposit addresses
func (m *DepositAddressManager) Sync(addresses map[string]map[string][]DepositAddressExtended) error {
	if m == nil {
		return fmt.Errorf("deposit address manager %w", ErrNilSubsystem)
	}
	m.m.Lock()
	defer m.m.Unlock()
	if m.store == nil {
		return ErrDepositAddressStoreIsNil
	}

	for k, v := range addresses {
		r := make(map[string][]DepositAddressExtended)
		for w, x := range v {
			r[strings.ToUpper(w)] = x
		}
		m.store[strings.ToUpper(k)] = r
	}
	return nil
}
