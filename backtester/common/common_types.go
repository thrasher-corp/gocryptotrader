package common

import "github.com/thrasher-corp/gocryptotrader/exchanges/order"

// DecimalPlaces is a lovely little holder
// for the amount of decimal places we want to allow
const DecimalPlaces = 8

// DoNothing is an explicit signal for the backtester to not perform an action
// based upon indicator results
const DoNothing order.Side = "DO NOTHING"
