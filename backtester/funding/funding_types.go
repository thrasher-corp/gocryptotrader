package funding

import (
	"errors"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var (
	ErrCannotAllocate = errors.New("cannot allocate funds")
	ErrFundsNotFound  = errors.New("funding not found")
)

// IFundingManager limits funding usage for portfolio event handling
type IFundingManager interface {
	Reset()
	IsUsingExchangeLevelFunding() bool
	GetFundingForEAC(string, asset.Item, currency.Code) (*Item, error)
	GetFundingForEvent(common.EventHandler) (*Pair, error)
	GetFundingForEAP(string, asset.Item, currency.Pair) (*Pair, error)
}

// AllFunds is the benevolent holder of all funding levels across all
// currencies used in the backtester
type AllFunds struct {
	usingExchangeLevelFunding bool
	items                     []*Item
}

type IPairReader interface {
	BaseInitialFunds() decimal.Decimal
	QuoteInitialFunds() decimal.Decimal
	BaseAvailable() decimal.Decimal
	QuoteAvailable() decimal.Decimal
}

// IPairReserver limits funding usage for portfolio event handling
type IPairReserver interface {
	IPairReader
	CanPlaceOrder(order.Side) bool
	Reserve(decimal.Decimal, order.Side) error
}

// IPairReleaser limits funding usage for exchange event handling
type IPairReleaser interface {
	IPairReader
	Increase(decimal.Decimal, order.Side)
	Release(decimal.Decimal, decimal.Decimal, order.Side) error
}
