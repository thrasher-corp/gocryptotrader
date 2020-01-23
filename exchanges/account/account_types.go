package account

import (
	"sync"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

var accounts = make(map[string]*Holdings)
var mtx sync.Mutex

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
