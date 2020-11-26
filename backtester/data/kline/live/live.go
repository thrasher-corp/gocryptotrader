package live

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

func LoadData(exch exchange.IBotExchange, dataType string, interval time.Duration, fPair currency.Pair, a asset.Item) (*kline.Item, error) {
	var candles kline.Item
	var err error
	switch dataType {
	case common.CandleStr:
		candles, err = exch.GetHistoricCandles(
			fPair,
			a,
			time.Now().Add(-interval), // multiplied by 2 to ensure the latest candle is always included
			time.Now(),
			kline.Interval(interval))
		if err != nil {
			return nil, err
		}
	case common.TradeStr:
		var trades []trade.Data
		trades, err = exch.GetRecentTrades(
			fPair,
			a)
		if err != nil {
			return nil, err
		}

		candles, err = trade.ConvertTradesToCandles(kline.Interval(interval), trades...)
		if err != nil {
			return nil, err
		}
		base := exch.GetBase()
		if len(candles.Candles) <= 1 && base.GetSupportedFeatures().RESTCapabilities.TradeHistory {
			trades, err = exch.GetHistoricTrades(
				fPair,
				a,
				time.Now().Add(-interval), // multiplied by 2 to ensure the latest candle is always included
				time.Now())
			if err != nil {
				return nil, err
			}

			candles, err = trade.ConvertTradesToCandles(kline.Interval(interval), trades...)
			if err != nil {
				return nil, err
			}
		}

	default:
		return nil, fmt.Errorf("unrecognised api datatype received: '%v'", dataType)
	}
	candles.Exchange = strings.ToLower(exch.GetName())
	return &candles, nil
}
