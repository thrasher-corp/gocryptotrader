package funding

import (
	"errors"
	"time"

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

// FundManager is the benevolent holder of all funding levels across all
// currencies used in the backtester
type FundManager struct {
	usingExchangeLevelFunding bool
	items                     []*Item
	snapshots                 []snapshotsAtTime
}

type snapshotsAtTime struct {
	time      time.Time
	snapshots []snappy
}

type snappy struct {
	Offset         int64
	Item           Item
	Timestamp      time.Time       `json:"timestamp"`
	PositionsSize  decimal.Decimal `json:"positions-size"`
	PositionsValue decimal.Decimal `json:"positions-value"`
	SoldAmount     decimal.Decimal `json:"sold-amount"`
	SoldValue      decimal.Decimal `json:"sold-value"`
	BoughtAmount   decimal.Decimal `json:"bought-amount"`
	BoughtValue    decimal.Decimal `json:"bought-value"`
	RemainingFunds decimal.Decimal `json:"remaining-funds"`
	CommittedFunds decimal.Decimal `json:"committed-funds"`

	TotalValueDifference      decimal.Decimal
	ChangeInTotalValuePercent decimal.Decimal
	BoughtValueDifference     decimal.Decimal
	SoldValueDifference       decimal.Decimal
	PositionsValueDifference  decimal.Decimal

	TotalValue                   decimal.Decimal `json:"total-value"`
	TotalFees                    decimal.Decimal `json:"total-fees"`
	TotalValueLostToVolumeSizing decimal.Decimal `json:"total-value-lost-to-volume-sizing"`
	TotalValueLostToSlippage     decimal.Decimal `json:"total-value-lost-to-slippage"`
	TotalValueLost               decimal.Decimal `json:"total-value-lost"`

	RiskFreeRate decimal.Decimal `json:"risk-free-rate"`
}

// IFundingManager limits funding usage for portfolio event handling
type IFundingManager interface {
	Reset()
	IsUsingExchangeLevelFunding() bool
	GetFundingForEAC(string, asset.Item, currency.Code) (*Item, error)
	GetFundingForEvent(common.EventHandler) (*Pair, error)
	GetFundingForEAP(string, asset.Item, currency.Pair) (*Pair, error)
	Transfer(decimal.Decimal, *Item, *Item) error
}

// IFundTransferer allows for funding amounts to be transferred
// implementation can be swapped for live transferring
type IFundTransferer interface {
	IsUsingExchangeLevelFunding() bool
	Transfer(decimal.Decimal, *Item, *Item) error
	GetFundingForEAC(string, asset.Item, currency.Code) (*Item, error)
	GetFundingForEvent(common.EventHandler) (*Pair, error)
	GetFundingForEAP(string, asset.Item, currency.Pair) (*Pair, error)
}

// IPairReader is used to limit pair funding functions
// to readonly
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
	IncreaseAvailable(decimal.Decimal, order.Side)
	Release(decimal.Decimal, decimal.Decimal, order.Side) error
}

// Item holds funding data per currency item
type Item struct {
	exchange     string
	asset        asset.Item
	currency     currency.Code
	initialFunds decimal.Decimal
	available    decimal.Decimal
	reserved     decimal.Decimal
	pairedWith   *Item
	transferFee  decimal.Decimal
}

// Pair holds two currencies that are associated with each other
type Pair struct {
	Base  *Item
	Quote *Item
}
