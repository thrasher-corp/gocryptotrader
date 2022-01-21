package ftxcashandcarry

import (
	"errors"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/base"
)

const (
	// Name is the strategy name
	Name         = "ftx-cash-carry"
	description  = `The relative strength index is a technical indicator used in the analysis of financial markets. It is intended to chart the current and historical strength or weakness of a stock or market based on the closing prices of a recent trading period`
	exchangeName = "ftx"
)

var (
	errFuturesOnly      = errors.New("can only work with futures")
	errOnlyFTXSupported = errors.New("only FTX supported for this strategy")
)

// Strategy is an implementation of the Handler interface
type Strategy struct {
	base.Strategy
	rsiPeriod decimal.Decimal
	rsiLow    decimal.Decimal
	rsiHigh   decimal.Decimal
}
