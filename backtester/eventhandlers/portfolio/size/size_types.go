package size

import (
	"errors"

	"github.com/thrasher-corp/gocryptotrader/backtester/config"
)

var (
	errNoFunds         = errors.New("no funds available")
	errLessThanMinimum = errors.New("sized amount less than minimum")
	errCannotAllocate  = errors.New("portfolio manager cannot allocate funds for an order")
)

// Size contains buy and sell side rules
type Size struct {
	BuySide  config.MinMax
	SellSide config.MinMax
}
