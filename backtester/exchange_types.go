package backtest

import "github.com/thrasher-corp/gocryptotrader/currency"

type Exchange struct {
	CurrencyPair   currency.Pair
	ExchangeFee    float64
	CommissionRate float64
	MakerFee       float64
	TakerFee       float64
}
