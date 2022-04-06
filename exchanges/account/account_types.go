package account

import (
	"errors"
	"sync"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/dispatch"
	"github.com/thrasher-corp/gocryptotrader/exchanges/alert"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// Vars for the ticker package
var (
	service                 Service
	errAccountBalancesIsNil = errors.New("account balances is nil")
)

// Service holds ticker information for each individual exchange
type Service struct {
	exchangeAccounts map[string]*Accounts
	mux              *dispatch.Mux
	m                sync.Mutex
}

// Accounts holds a stream ID and a map to the exchange holdings
type Accounts struct {
	ID          uuid.UUID
	SubAccounts map[string]map[asset.Item]map[*currency.Item]*BalanceInternal
}

// Holdings is a generic type to hold each exchange's holdings for all enabled
// currencies
type Holdings struct {
	Exchange string
	Accounts []SubAccount
}

// SubAccount defines a singular account type with associated currency balances
type SubAccount struct {
	ID         string
	AssetType  asset.Item
	Currencies []Balance
}

// Balance is a sub type to store currency name and individual totals
type Balance struct {
	CurrencyName           currency.Code
	Total                  float64
	Hold                   float64
	Free                   float64
	AvailableWithoutBorrow float64
	Borrowed               float64
}

// Change defines incoming balance change on currency holdings
type Change struct {
	Exchange string
	Currency currency.Code
	Asset    asset.Item
	Amount   float64
	Account  string
}

// BalanceInternal stores the full balance information for that specific asset
type BalanceInternal struct {
	total                  float64
	hold                   float64
	free                   float64
	availableWithoutBorrow float64
	borrowed               float64
	m                      sync.Mutex

	// notice alerts for when the balance changes for strategy inspection and
	// usage.
	notice alert.Notice
}
