package trade

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

var buffer []*Data
var candles []kline.Candle

func (t *Traderino) Process(data ...*Data) {
	t.mutex.Lock()
	for i := range data {
		buffer = append(buffer, data[i])
	}
	t.mutex.Unlock()
}
/*
func (t *Traderino) Processor() {
	timer := time.NewTicker(time.Minute)
	for {
		select {
		case <-timer.C:
			t.mutex.RLock()
			for i := range buffer {
				if
			}
			t.mutex.RUnlock()
		}
	}
}
*/
func (t *Traderino) CandleProcessor() {
	timer := time.NewTicker(time.Minute)
	for {
		select {
		case <-timer.C:

		}
	}
}

