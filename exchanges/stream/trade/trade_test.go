package trade

import (
	"sort"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var cp = currency.Pair{
	Delimiter: currency.ColonDelimiter,
	Base:      currency.BTC,
	Quote:     currency.USDT,
}

func TestSeparateTradesByUnitOfTime(t *testing.T) {
	var datas []Data
	cap := int64(2000)
	for i :=int64(0); i < cap; i++ {
		datas = append(datas, Data{
			Timestamp:    time.Now().Add(-time.Duration(i) * time.Second),
			CurrencyPair: cp,
			AssetType:    asset.Spot,
			Exchange:     "Binance",
			EventType:    order.Market,
			Price:        float64(i * 7 / 3),
			Amount:       float64(i * 3 / 2),
			Side:         order.Buy,
		})
	}
	sort.Sort(ByDate(datas))
	groupedData := splitTimes(kline.FifteenSecond, datas...)
	t.Log(len(groupedData))
	var candles []kline.Candle
	for k, v := range groupedData {
		candles = append(candles, classifyohlcv(time.Unix(k, 0), v...))
	}
	sort.Sort(kline.ByDate(candles))
	t.Log(candles)
}

func splitTimes(interval kline.Interval, times ...Data) map[int64][]Data {
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

func classifyohlcv (t time.Time, datas ...Data) (c kline.Candle) {
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