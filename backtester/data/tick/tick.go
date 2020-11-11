package tick

import (
	"errors"

	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/ticker"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	ticker2 "github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
)

type DataFromTick struct {
	ticks []*ticker2.Price
	data.Data
}

func (d *DataFromTick) Load() error {
	if len(d.ticks) == 0 {
		return errors.New("no tick data provided")
	}

	loadedData := make([]interfaces.DataEventHandler, len(d.ticks))
	for i := range d.ticks {
		loadedData[i] = &ticker.Tick{
			Event: event.Event{
				Exchange:     d.ticks[i].ExchangeName,
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
