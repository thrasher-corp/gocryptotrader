package account

import (
	"errors"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/common/key"
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
	mu               sync.Mutex
}

// Accounts holds a stream ID and a map to the exchange holdings
type Accounts struct {
	ID uuid.UUID
	// NOTE: Credentials is a place holder for a future interface type, which
	// will need -
	// TODO: Credential tracker to match to keys that are managed and return
	// pointer.
	// TODO: Have different cred struct for centralized verse DEFI exchanges.
	subAccounts map[Credentials]map[key.SubAccountAsset]currencyBalances
}

type currencyBalances = map[*currency.Item]*ProtectedBalance

// Holdings is a generic type to hold each exchange's holdings for all enabled
// currencies
type Holdings struct {
	Exchange string
	Accounts []SubAccount
}

// SubAccount defines a singular account type with associated currency balances
type SubAccount struct {
	Credentials Protected
	ID          string
	AssetType   asset.Item
	Currencies  []Balance
}

// Balance is a sub-type to store currency name and individual totals
type Balance struct {
	Currency               currency.Code
	Total                  float64
	Hold                   float64
	Free                   float64
	AvailableWithoutBorrow float64
	Borrowed               float64
	UpdatedAt              time.Time
}

// Change defines incoming balance change on currency holdings
type Change struct {
	Account   string
	AssetType asset.Item
	Balance   *Balance
}

// ProtectedBalance stores the full balance information for that specific asset
type ProtectedBalance struct {
	total                  float64
	hold                   float64
	free                   float64
	availableWithoutBorrow float64
	borrowed               float64
	m                      sync.Mutex
	updatedAt              time.Time

	// notice alerts for when the balance changes for strategy inspection and
	// usage.
	notice alert.Notice
}

// Protected limits the access to the underlying credentials outside of this
// package.
type Protected struct {
	creds Credentials
}
