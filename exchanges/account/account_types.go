package account

import (
	"sync"

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
	accounts map[string]*Holdings
	mux      *dispatch.Mux
	sync.Mutex
}

// Change defines incoming balance change on currency holdings
type Change struct {
	Exchange string
	Currency currency.Code
	Asset    asset.Item
	Amount   float64
	Account  string
}

// Balance defines amount levels on an exchange account for a currency holding
type Balance struct {
	// The sum total of balance.
	Total float64
	// The amount currently in use either for lending or locked in a limit order.
	Locked float64
}

// FullSnapshot defines a full snapshot of account asset balances
type FullSnapshot map[string]AssetSnapshot

// AssetSnapshot defines a snapshot for the asset items
type AssetSnapshot map[asset.Item]HoldingsSnapshot

// HoldingsSnapshot defines a currency and its related balance
type HoldingsSnapshot map[currency.Code]Balance

// ident defines identifying variables
type ident struct {
	Exchange, Account string
	Asset             asset.Item
	Currency          currency.Code
}
