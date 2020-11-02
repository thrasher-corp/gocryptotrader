package ticker

import (
	"errors"

	"github.com/shopspring/decimal"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	portfolio "github.com/thrasher-corp/gocryptotrader/backtester/datahandler"
	"github.com/thrasher-corp/gocryptotrader/backtester/event"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
)

type DataFromTick struct {
	ticks []*ticker.Price

	data.Data
}

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

func (d *DataFromTick) Load() error {
	if len(d.ticks) == 0 {
		return errors.New("no tick data provided")
	}

	loadedData := make([]portfolio.DataEventHandler, len(d.ticks))
	for i := range d.ticks {
		loadedData[i] = &Tick{
			Event: event.Event{
				Time:         d.ticks[i].LastUpdated,
				CurrencyPair: d.ticks[i].Pair,
			},
			Ask: d.ticks[i].Ask,
			Bid: d.ticks[i].Bid,
		}
	}
	d.SetStream(loadedData)
	d.SortStream()
	return nil
}
