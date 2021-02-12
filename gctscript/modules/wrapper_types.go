package modules

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

const (
	// ErrParameterConvertFailed error to return when type conversion fails
	ErrParameterConvertFailed = "%v failed conversion"
	// ErrParameterWithPositionConvertFailed error to return when a positional conversion fails
	ErrParameterWithPositionConvertFailed = "%v at position %v failed conversion"
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
	QueryOrder(exch, orderid string, pair currency.Pair, assetType asset.Item) (*order.Detail, error)
	SubmitOrder(submit *order.Submit) (*order.SubmitResponse, error)
	CancelOrder(exch, orderid string, pair currency.Pair, item asset.Item) (bool, error)
	AccountInformation(exch string, assetType asset.Item) (account.Holdings, error)
	DepositAddress(exch string, currencyCode currency.Code) (string, error)
	WithdrawalFiatFunds(bankAccountID string, request *withdraw.Request) (out string, err error)
	WithdrawalCryptoFunds(request *withdraw.Request) (out string, err error)
	OHLCV(exch string, pair currency.Pair, item asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error)
}

// SetModuleWrapper link the wrapper and interface to use for modules
func SetModuleWrapper(wrapper GCT) {
	Wrapper = wrapper
}
