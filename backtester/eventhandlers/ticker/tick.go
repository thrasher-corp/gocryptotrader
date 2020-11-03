package ticker

import (
	"github.com/shopspring/decimal"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	portfolio "github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
)

func (t *Tick) LatestPrice() float64 {
	bid := decimal.NewFromFloat(t.Bid)
	ask := decimal.NewFromFloat(t.Ask)
	diff := decimal.New(2, 0)
	latest, _ := bid.Add(ask).Div(diff).Round(common.DecimalPlaces).Float64()
	return latest
}

func (t *Tick) DataType() portfolio.DataType {
	return data.DataTypeTick
}

func (t *Tick) Spread() float64 {
	return t.Bid - t.Ask
}
