package common

import "github.com/thrasher-corp/gocryptotrader/exchanges/order"

const (
	// DecimalPlaces is a lovely little holder
	// for the amount of decimal places we want to allow
	DecimalPlaces = 8
	// DoNothing is an explicit signal for the backtester to not perform an action
	// based upon indicator results
	DoNothing order.Side = "DO NOTHING"
	// used to identify the type of data in a config
	CandleStr = "candle"
	TradeStr  = "trade"
)
