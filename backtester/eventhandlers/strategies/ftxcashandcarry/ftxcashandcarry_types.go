package ftxcashandcarry

import (
	"errors"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/base"
)

const (
	// Name is the strategy name
	Name                               = "ftx-cash-carry"
	description                        = `A cash and carry trade (or basis trading) consists in taking advantage of the premium of a futures contract over the spot price. For example if Ethereum Futures are trading well above its Spot price (contango) you could perform an arbitrage and take advantage of this opportunity.`
	exchangeName                       = "ftx"
	openShortDistancePercentageString  = "openShortDistancePercentage"
	closeShortDistancePercentageString = "closeShortDistancePercentage"
)

var (
	errFuturesOnly      = errors.New("can only work with futures")
	errOnlyFTXSupported = errors.New("only FTX supported for this strategy")
	errNoSignals        = errors.New("no data signals to process")
)

// Strategy is an implementation of the Handler interface
type Strategy struct {
	base.Strategy
	openShortDistancePercentage  decimal.Decimal
	closeShortDistancePercentage decimal.Decimal
}
