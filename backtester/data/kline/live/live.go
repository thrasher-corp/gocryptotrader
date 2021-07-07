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

// LoadData retrieves data from a GoCryptoTrader exchange wrapper which calls the exchange's API for the latest interval
func LoadData(exch exchange.IBotExchange, dataType int64, interval time.Duration, fPair currency.Pair, a asset.Item) (*kline.Item, error) {
	var candles kline.Item
	var err error
	switch dataType {
	case common.DataCandle:
		candles, err = exch.GetHistoricCandles(
			fPair,
			a,
			time.Now().Add(-interval*2), // multiplied by 2 to ensure the latest candle is always included
			time.Now(),
			kline.Interval(interval))
		if err != nil {
			return nil, fmt.Errorf("could not retrieve live candle data for %v %v %v, %v", exch.GetName(), a, fPair, err)
		}
	case common.DataTrade:
		var trades []trade.Data
		trades, err = exch.GetHistoricTrades(fPair, a, time.Now().Add(-interval*2), time.Now()) // multiplied by 2 to ensure the latest candle is always included
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
				return nil, fmt.Errorf("could not retrieve live trade data for %v %v %v, %v", exch.GetName(), a, fPair, err)
			}

			candles, err = trade.ConvertTradesToCandles(kline.Interval(interval), trades...)
			if err != nil {
				return nil, fmt.Errorf("could not convert live trade data to candles for %v %v %v, %v", exch.GetName(), a, fPair, err)
			}
		}
	default:
		return nil, fmt.Errorf("could not retrieve live data for %v %v %v, %w", exch.GetName(), a, fPair, common.ErrInvalidDataType)
	}
	candles.Exchange = strings.ToLower(exch.GetName())
	return &candles, nil
}
