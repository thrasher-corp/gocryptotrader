package kline

import (
	"errors"

	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

var errNoCandleData = errors.New("no candle data provided")

// DataFromKline is a struct which implements the data.Streamer interface
// It holds candle data for a specified range with helper functions
type DataFromKline struct {
	*data.Base
	Item        *gctkline.Item
	RangeHolder *gctkline.IntervalRangeHolder
}
