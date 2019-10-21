package modules

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
)

// Wrapper instance of GCT to use for modules
var Wrapper GCT

type GCT interface {
	Exchange
}

type Exchange interface {
	Exchanges(enabledOnly bool) []string

	IsEnabled(exch string) bool
	Orderbook(exch string, pair currency.Pair, item asset.Item) (*orderbook.Base, error)
	Ticker(exch string, pair currency.Pair, item asset.Item) (*ticker.Price, error)
	Pairs(exch string, enabledOnly bool, item asset.Item) (currency.Pairs, error)

	QueryOrder(exch string) error
	SubmitOrder() error
	CancelOrder() error

	AccountInformation(exch string) (AccountInfo, error)
}

// SetModuleWrapper link the wrapper and interface to use for modules
func SetModuleWrapper(wrapper GCT) {
	Wrapper = wrapper
}

// AccountInfo is a Generic type to hold each exchange's holdings in
// all enabled currencies
type AccountInfo struct {
	Exchange string
	Accounts []Account
}

// Account defines a singular account type with asocciated currencies
type Account struct {
	ID         string
	Currencies []AccountCurrencyInfo
}

// AccountCurrencyInfo is a sub type to store currency name and value
type AccountCurrencyInfo struct {
	CurrencyName currency.Code
	TotalValue   float64
	Hold         float64
}

/*
Orderbook - done
Ticker - done
Enabled pairs - done
Enabled exchanges - done
Submit order
Cancel order
Query Order
Account information (balance etc)
deposit address fetching/withdrawal of crypto/fiat

*/
