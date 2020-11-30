package portfolio

import (
	"errors"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/config"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/hodlings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/risk"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var NotEnoughFundsErr = errors.New("not enough funds to buy")
var NoHoldingsToSellErr = errors.New("no holdings to sell")

type Portfolio struct {
	SizeManager               SizeHandler
	RiskManager               risk.RiskHandler
	ExchangeAssetPairSettings map[string]map[asset.Item]map[currency.Pair]*ExchangeAssetPairSettings
}

type ExchangeAssetPairSettings struct {
	InitialFunds      float64
	Fee               float64
	Funds             float64
	BuySideSizing     config.MinMax
	SellSideSizing    config.MinMax
	Leverage          config.Leverage
	PositionSnapshots hodlings.Snapshots
	ComplianceManager compliance.Manager
}

type Handler interface {
	OnSignal(signal.SignalEvent, interfaces.DataHandler, *exchange.CurrencySettings) (*order.Order, error)
	OnFill(fill.FillEvent, interfaces.DataHandler) (*fill.Fill, error)
	Update(interfaces.DataEventHandler)

	SetInitialFunds(string, asset.Item, currency.Pair, float64)
	GetInitialFunds(string, asset.Item, currency.Pair) float64
	SetFunds(string, asset.Item, currency.Pair, float64)
	GetFunds(string, asset.Item, currency.Pair) float64

	GetComplianceManager(string, asset.Item, currency.Pair) (*compliance.Manager, error)

	SetHoldings(string, asset.Item, currency.Pair, time.Time, hodlings.Hodling, bool) error
	ViewHoldingAtTimePeriod(string, asset.Item, currency.Pair, time.Time) (hodlings.Hodling, error)
	SetFee(string, asset.Item, currency.Pair, float64)
	GetFee(string, asset.Item, currency.Pair) float64
	Reset()
}

type SizeHandler interface {
	SizeOrder(exchange.OrderEvent, interfaces.DataEventHandler, float64, *exchange.CurrencySettings) (*order.Order, error)
}
