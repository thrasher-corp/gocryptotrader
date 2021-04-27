package account

import (
	"sync"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/dispatch"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// Vars for the ticker package
var (
	service *Service
)

// Service holds ticker information for each individual exchange
type Service struct {
	accounts map[string]*Account
	mux      *dispatch.Mux
	sync.Mutex
}

// Account holds a stream ID and a pointer to the exchange holdings
type Account struct {
	h  *Holdings
	ID uuid.UUID
}

// Holdings is a generic type to hold each exchange's holdings for all enabled
// currencies
type Holdings struct {
	Exchange string
	Accounts []SubAccount
}

// SubAccount defines a singular account type with asocciated currency balances
type SubAccount struct {
	ID         string
	AssetType  asset.Item
	Currencies []Balance
}

// Balance is a sub type to store currency name and individual totals
type Balance struct {
	CurrencyName currency.Code
	TotalValue   float64
	Hold         float64
}

// Change defines incoming balance change on currency holdings
type Change struct {
	Exchange string
	Currency currency.Code
	Asset    asset.Item
	Amount   float64
	Account  string
}
