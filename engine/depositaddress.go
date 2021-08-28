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
	store map[string]map[string]deposit.Address
}

// SetupDepositAddressManager returns a DepositAddressManager
func SetupDepositAddressManager() *DepositAddressManager {
	return &DepositAddressManager{
		store: make(map[string]map[string]deposit.Address),
	}
}

// GetDepositAddressByExchangeAndCurrency returns a deposit address for the specified exchange and cryptocurrency
// if it exists
func (m *DepositAddressManager) GetDepositAddressByExchangeAndCurrency(exchName string, currencyItem currency.Code) (deposit.Address, error) {
	m.m.Lock()
	defer m.m.Unlock()

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

	return addr, nil
}

// GetDepositAddressesByExchange returns a list of cryptocurrency addresses for the specified
// exchange if they exist
func (m *DepositAddressManager) GetDepositAddressesByExchange(exchName string) (map[string]deposit.Address, error) {
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
func (m *DepositAddressManager) Sync(addresses map[string]map[string]deposit.Address) error {
	if m == nil {
		return fmt.Errorf("deposit address manager %w", ErrNilSubsystem)
	}
	m.m.Lock()
	defer m.m.Unlock()
	if m.store == nil {
		return ErrDepositAddressStoreIsNil
	}

	for k, v := range addresses {
		r := make(map[string]deposit.Address)
		for w, x := range v {
			r[strings.ToUpper(w)] = x
		}
		m.store[strings.ToUpper(k)] = r
	}
	return nil
}
