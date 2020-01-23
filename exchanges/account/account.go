package account

import (
	"errors"
)

// Process processes new account holdings updates
func Process(h *Holdings) error {
	if h == nil {
		return errors.New("cannot be nil")
	}

	if h.Exchange == "" {
		return errors.New("exchange name unset")
	}

	mtx.Lock()
	defer mtx.Unlock()
	holdings, ok := accounts[h.Exchange]
	if !ok {
		accounts[h.Exchange] = h
		return nil
	}

	holdings.Accounts = h.Accounts
	return nil
}

// GetHoldings returns full holdings for an exchange
func GetHoldings(exch string) (Holdings, error) {
	if exch == "" {
		return Holdings{}, errors.New("exchange name unset")
	}

	mtx.Lock()
	defer mtx.Unlock()
	h, ok := accounts[exch]
	if !ok {
		return Holdings{}, errors.New("exchange account holdings not found")
	}

	return *h, nil
}
