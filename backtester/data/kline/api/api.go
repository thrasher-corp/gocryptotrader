package api

import (
	"fmt"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

func LoadData(dataType string, startDate, endDate time.Time, interval time.Duration, exch exchange.IBotExchange, fPair currency.Pair, a asset.Item) (*kline.Item, error) {
	var candles kline.Item
	var err error
	switch dataType {
	case common.CandleStr:
		candles, err = exch.GetHistoricCandlesExtended(
			fPair,
			a,
			startDate,
			endDate,
			kline.Interval(interval))
		if err != nil {
			return nil, err
		}
	case common.TradeStr:
		var trades []trade.Data
		trades, err = exch.GetHistoricTrades(
			fPair,
			a,
			startDate,
			endDate)
		if err != nil {
			return nil, err
		}

		candles, err = trade.ConvertTradesToCandles(kline.Interval(interval), trades...)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unrecognised api datatype received: '%v'", dataType)
	}
	candles.Exchange = strings.ToLower(candles.Exchange)
	return &candles, nil
}
