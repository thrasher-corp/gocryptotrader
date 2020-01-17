package modules

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/withdraw"
)

// Wrapper instance of GCT to use for modules
var Wrapper GCT

// GCT interface requirements
type GCT interface {
	Exchange
}

// Exchange interface requirements
type Exchange interface {
	Exchanges(enabledOnly bool) []string
	IsEnabled(exch string) bool
	Orderbook(exch string, pair currency.Pair, item asset.Item) (*orderbook.Base, error)
	Ticker(exch string, pair currency.Pair, item asset.Item) (*ticker.Price, error)
	Pairs(exch string, enabledOnly bool, item asset.Item) (*currency.Pairs, error)
	QueryOrder(exch, orderid string) (*order.Detail, error)
	SubmitOrder(exch string, submit *order.Submit) (*order.SubmitResponse, error)
	CancelOrder(exch, orderid string) (bool, error)
	AccountInformation(exch string) (*AccountInfo, error)
	DepositAddress(exch string, currencyCode currency.Code) (string, error)
	WithdrawalFiatFunds(exch, bankaccountid string, request *withdraw.FiatRequest) (out string, err error)
	WithdrawalCryptoFunds(exch string, request *withdraw.CryptoRequest) (out string, err error)
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

// Account defines a singular account type with associated currencies
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
