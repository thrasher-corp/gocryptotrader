package portfolio

import (
	"errors"

	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/risk"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	"github.com/thrasher-corp/gocryptotrader/backtester/internalordermanager"
	"github.com/thrasher-corp/gocryptotrader/backtester/positions"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var NotEnoughFundsErr = errors.New("not enough funds to buy")
var NoHoldingsToSellErr = errors.New("no holdings to sell")

type Portfolio struct {
	InitialFunds float64
	Funds        float64
	Holdings     map[string]map[asset.Item]map[currency.Pair]positions.Positions
	Transactions []fill.FillEvent
	SizeManager  SizeHandler
	RiskManager  risk.RiskHandler
	Fees         map[string]map[asset.Item]map[currency.Pair]float64
}

type PortfolioHandler interface {
	OnSignal(signal.SignalEvent, interfaces.DataHandler, *exchange.CurrencySettings) (*order.Order, error)
	OnFill(fill.FillEvent, interfaces.DataHandler) (*fill.Fill, error)
	Update(interfaces.DataEventHandler)

	SetInitialFunds(float64)
	GetInitialFunds() float64
	SetFunds(float64)
	GetFunds() float64

	Value() float64
	SetHoldings(string, asset.Item, currency.Pair, positions.Positions)
	ViewHoldings(string, asset.Item, currency.Pair) positions.Positions
	SetFee(string, asset.Item, currency.Pair, float64)
	GetFee(string, asset.Item, currency.Pair) float64
	Reset()
}

type SizeHandler interface {
	SizeOrder(internalordermanager.OrderEvent, interfaces.DataEventHandler, float64, *exchange.CurrencySettings) (*order.Order, error)
}
