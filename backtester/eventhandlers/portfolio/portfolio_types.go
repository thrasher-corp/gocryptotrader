package portfolio

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/risk"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	gctexchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

const notEnoughFundsTo = "not enough funds to"

var (
	errInvalidDirection     = errors.New("invalid direction")
	errRiskManagerUnset     = errors.New("risk manager unset")
	errSizeManagerUnset     = errors.New("size manager unset")
	errAssetUnset           = errors.New("asset unset")
	errCurrencyPairUnset    = errors.New("currency pair unset")
	errExchangeUnset        = errors.New("exchange unset")
	errNegativeRiskFreeRate = errors.New("received negative risk free rate")
	errNoPortfolioSettings  = errors.New("no portfolio settings")
	errNoHoldings           = errors.New("no holdings found")
	errHoldingsNoTimestamp  = errors.New("holding with unset timestamp received")
	errUnsetFuturesTracker  = errors.New("portfolio settings futures tracker unset")
)

// Portfolio stores all holdings and rules to assess orders, allowing the portfolio manager to
// modify, accept or reject strategy signals
type Portfolio struct {
	riskFreeRate                       decimal.Decimal
	sizeManager                        SizeHandler
	riskManager                        risk.Handler
	exchangeAssetPairPortfolioSettings map[key.ExchangeAssetPair]*Settings
}

// Handler contains all functions expected to operate a portfolio manager
type Handler interface {
	OnSignal(signal.Event, *exchange.Settings, funding.IFundReserver) (*order.Order, error)
	OnFill(fill.Event, funding.IFundReleaser) (fill.Event, error)
	GetLatestOrderSnapshotForEvent(common.Event) (compliance.Snapshot, error)
	GetLatestOrderSnapshots() ([]compliance.Snapshot, error)
	ViewHoldingAtTimePeriod(common.Event) (*holdings.Holding, error)
	SetHoldingsForTimestamp(*holdings.Holding) error
	UpdateHoldings(data.Event, funding.IFundReleaser) error
	GetPositions(common.Event) ([]futures.Position, error)
	TrackFuturesOrder(fill.Event, funding.IFundReleaser) (*PNLSummary, error)
	UpdatePNL(common.Event, decimal.Decimal) error
	GetLatestPNLForEvent(common.Event) (*PNLSummary, error)
	CheckLiquidationStatus(data.Event, funding.ICollateralReader, *PNLSummary) error
	CreateLiquidationOrdersForExchange(data.Event, funding.IFundingManager) ([]order.Event, error)
	GetLatestHoldingsForAllCurrencies() []holdings.Holding
	Reset() error
	SetHoldingsForEvent(funding.IFundReader, common.Event) error
	GetLatestComplianceSnapshot(string, asset.Item, currency.Pair) (*compliance.Snapshot, error)
}

// SizeHandler is the interface to help size orders
type SizeHandler interface {
	SizeOrder(order.Event, decimal.Decimal, *exchange.Settings) (*order.Order, decimal.Decimal, error)
}

// Settings holds all important information for the portfolio manager
// to assess purchasing decisions
type Settings struct {
	exchangeName string
	assetType    asset.Item
	pair         currency.Pair

	BuySideSizing     exchange.MinMax
	SellSideSizing    exchange.MinMax
	Leverage          exchange.Leverage
	HoldingsSnapshots map[int64]*holdings.Holding
	ComplianceManager compliance.Manager
	Exchange          gctexchange.IBotExchange
	FuturesTracker    *futures.MultiPositionTracker
}

// PNLSummary holds a PNL result along with
// exchange details
type PNLSummary struct {
	Exchange           string
	Asset              asset.Item
	Pair               currency.Pair
	CollateralCurrency currency.Code
	Offset             int64
	Result             futures.PNLResult
}

// IPNL defines an interface for an implementation
// to retrieve PNL from a position
type IPNL interface {
	GetUnrealisedPNL() BasicPNLResult
	GetRealisedPNL() BasicPNLResult
	GetCollateralCurrency() currency.Code
	GetDirection() gctorder.Side
	GetPositionStatus() gctorder.Status
}

// BasicPNLResult holds the time and the pnl
// of a position
type BasicPNLResult struct {
	Currency currency.Code
	Time     time.Time
	PNL      decimal.Decimal
}
