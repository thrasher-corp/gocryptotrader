package trade

import (
	"sort"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

// Setup creates the trade processor if trading is supported
func (t *Traderino) Setup() {
	go t.Processor()
}

// Shutdown kills the lingering processor
func (t *Traderino) Shutdown() {
	close(t.shutdown)
}

// Process will push trade data onto the buffer
func (t *Traderino) Process(data ...Data) {
	t.mutex.Lock()
	for i := range data {
		buffer = append(buffer, data[i])
	}
	t.mutex.Unlock()
}

// Processor will convert buffered trade data into candles
// then stores the candles and clears the buffer to allow
// more allocations
func (t *Traderino) Processor() {
	timer := time.NewTicker(time.Minute)
	for {
		select {
		case <-t.shutdown:
			return
		case <-timer.C:
			t.mutex.Lock()
			sort.Sort(ByDate(buffer))
			groupedData := splitTradeDataIntoIntervals(kline.FifteenSecond, buffer...)
			var candles []kline.Candle
			for k, v := range groupedData {
				candles = append(candles, classifyOHLCV(time.Unix(k, 0), v...))
			}
			sort.Sort(kline.ByDate(candles))
			// store the candles?
			// xtda.StoreCandles(candles)

			// clear the buffer
			buffer = nil
			t.mutex.Unlock()
		}
	}
}

func splitTradeDataIntoIntervals(interval kline.Interval, times ...Data) map[int64][]Data {
	groupedData := make(map[int64][]Data)
	for i:= range times {
		nearestInterval := getNearestInterval(times[i].Timestamp, interval)
		groupedData[nearestInterval] = append(
			groupedData[nearestInterval],
			times[i],
		)
	}
	return groupedData
}

func getNearestInterval(t time.Time, interval kline.Interval) int64 {
	return t.Truncate(interval.Duration()).Unix()
}

func classifyOHLCV (t time.Time, datas ...Data) (c kline.Candle) {
	sort.Sort(ByDate(datas))
	c.Open = datas[0].Price
	c.Close = datas[len(datas)-1].Price
	for i := range datas {
		// some exchanges will send it as negative for sells
		if datas[i].Price < 0 {
			datas[i].Price = datas[i].Price * -1
		}
		if datas[i].Amount < 0 {
			datas[i].Amount = datas[i].Amount * -1
		}
		if datas[i].Price < c.Low || c.Low == 0 {
			c.Low = datas[i].Price
		}
		if datas[i].Price > c.High {
			c.High = datas[i].Price
		}
		c.Volume += datas[i].Amount
	}
	c.Time = t
	return
}