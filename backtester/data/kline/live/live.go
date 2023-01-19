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
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

// LoadData retrieves data from a GoCryptoTrader exchange wrapper which calls the exchange's API for the latest interval
// note: this is not in a state to utilise with realOrders = true
func LoadData(ctx context.Context, timeToRetrieve time.Time, exch exchange.IBotExchange, dataType int64, interval time.Duration, currencyPair, underlyingPair currency.Pair, a asset.Item, verbose bool) (*kline.Item, error) {
	if exch == nil {
		return nil, fmt.Errorf("%w IBotExchange", gctcommon.ErrNilPointer)
	}
	var err error
	if verbose {
		ctx = request.WithVerbose(ctx)
	}
	var startTime, endTime time.Time
	exchBase := exch.GetBase()
	pFmt, err := exchBase.FormatExchangeCurrency(currencyPair, a)
	if err != nil {
		return nil, err
	}
	startTime = timeToRetrieve.Truncate(interval).Add(-interval)
	endTime = timeToRetrieve.Truncate(interval).Add(-1)

	var candles *kline.Item
	switch dataType {
	case common.DataCandle:
		candles, err = exch.GetHistoricCandles(ctx,
			pFmt,
			a,
			kline.Interval(interval),
			startTime,
			endTime)
		if err != nil {
			return nil, fmt.Errorf("could not retrieve live candle data for %v %v %v, %v", exch.GetName(), a, currencyPair, err)
		}
	case common.DataTrade:
		var trades []trade.Data
		trades, err = exch.GetHistoricTrades(ctx,
			pFmt,
			a,
			startTime,
			endTime,
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
				pFmt,
				a,
				startTime,
				endTime,
			)
			if err != nil {
				return nil, fmt.Errorf("could not retrieve live trade data for %v %v %v, %v", exch.GetName(), a, currencyPair, err)
			}

			candles, err = trade.ConvertTradesToCandles(kline.Interval(interval), trades...)
			if err != nil {
				return nil, fmt.Errorf("could not convert live trade data to candles for %v %v %v, %v", exch.GetName(), a, currencyPair, err)
			}
		}
	default:
		return nil, fmt.Errorf("could not retrieve live data for %v %v %v, %w: '%v'", exch.GetName(), a, currencyPair, common.ErrInvalidDataType, dataType)
	}
	candles.Exchange = strings.ToLower(exch.GetName())
	candles.UnderlyingPair = underlyingPair
	return candles, nil
}
