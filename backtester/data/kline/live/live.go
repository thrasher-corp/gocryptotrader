package live

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

// LoadData retrieves data from a GoCryptoTrader exchange wrapper which calls the exchange's API for the latest interval
// note: this is not in a state to utilise with realOrders = true
func LoadData(ctx context.Context, timeToRetrieve time.Time, exch exchange.IBotExchange, dataType int64, interval time.Duration, fPair, underlyingPair currency.Pair, a asset.Item, verbose bool) (*kline.Item, error) {
	if exch == nil {
		return nil, fmt.Errorf("%w IBotExchange", gctcommon.ErrNilPointer)
	}
	var candles kline.Item
	var err error
	var b *exchange.Base
	if verbose {
		b = exch.GetBase()
		b.Verbose = verbose
	}
	switch dataType {
	case common.DataCandle:
		candles, err = exch.GetHistoricCandles(ctx,
			fPair,
			a,
			timeToRetrieve.Truncate(interval).Add(-interval*2),
			timeToRetrieve,
			kline.Interval(interval),
		)
		if err != nil {
			return nil, fmt.Errorf("could not retrieve live candle data for %v %v %v, %v", exch.GetName(), a, fPair, err)
		}
	case common.DataTrade:
		var trades []trade.Data
		trades, err = exch.GetHistoricTrades(ctx,
			fPair,
			a,
			timeToRetrieve.Truncate(interval).Add(-interval*2),
			timeToRetrieve,
		)
		if err != nil {
			return nil, err
		}

		candles, err = trade.ConvertTradesToCandles(kline.Interval(interval), trades...)
		if err != nil {
			return nil, err
		}
		base := exch.GetBase()
		if len(candles.Candles) <= 1 && base.GetSupportedFeatures().RESTCapabilities.TradeHistory {
			trades, err = exch.GetHistoricTrades(ctx,
				fPair,
				a,
				timeToRetrieve.Add(-interval*2),
				timeToRetrieve,
			)
			if err != nil {
				return nil, fmt.Errorf("could not retrieve live trade data for %v %v %v, %v", exch.GetName(), a, fPair, err)
			}

			candles, err = trade.ConvertTradesToCandles(kline.Interval(interval), trades...)
			if err != nil {
				return nil, fmt.Errorf("could not convert live trade data to candles for %v %v %v, %v", exch.GetName(), a, fPair, err)
			}
		}
	default:
		return nil, fmt.Errorf("could not retrieve live data for %v %v %v, %w: '%v'", exch.GetName(), a, fPair, common.ErrInvalidDataType, dataType)
	}
	if verbose && b != nil {
		b.Verbose = false
	}
	candles.Exchange = strings.ToLower(exch.GetName())
	candles.UnderlyingPair = underlyingPair
	return &candles, nil
}
