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
	groupedData := convertTradeDatasToCandles(kline.FifteenSecond, datas...)
	t.Log(len(groupedData))
	var candles []kline.Candle
	for k, v := range groupedData {
		candles = append(candles, classifyOHLCV(time.Unix(k, 0), v...))
	}
	sort.Sort(kline.ByDate(candles))
	t.Log(candles)
}


