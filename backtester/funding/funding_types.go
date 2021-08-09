package funding

import (
	"errors"

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
	BaseInitialFunds() float64
	QuoteInitialFunds() float64
	BaseAvailable() float64
	QuoteAvailable() float64
}

// IPairReserver limits funding usage for portfolio event handling
type IPairReserver interface {
	IPairReader
	CanPlaceOrder(order.Side) bool
	Reserve(float64, order.Side) error
}

// IPairReleaser limits funding usage for exchange event handling
type IPairReleaser interface {
	IPairReader
	Increase(float64, order.Side)
	Release(float64, float64, order.Side) error
}
