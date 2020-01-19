package account

import (
	"errors"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

var accounts = make(map[string]*Holdings)

// Holdings is a generic type to hold each exchange's holdings for all enabled
// currencies
type Holdings struct {
	Exchange string
	Accounts []SubAccount
}

// SubAccount defines a singular account type with asocciated currency balances
type SubAccount struct {
	ID         string
	Currencies []Balance
}

// Balance is a sub type to store currency name and individual totals
type Balance struct {
	CurrencyName currency.Code
	TotalValue   float64
	Hold         float64
}

// Process processes new account holdings updates
func Process(h *Holdings) error {
	if h == nil {
		return errors.New("cannot be nil")
	}

	if h.Exchange == "" {
		return errors.New("exchange name unset")
	}

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

	h, ok := accounts[exch]
	if !ok {
		return Holdings{}, errors.New("exchange account holdings not found")
	}

	return *h, nil
}
