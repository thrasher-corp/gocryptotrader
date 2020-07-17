package engine

import (
	"errors"

	"github.com/thrasher-corp/gocryptotrader/database/repository/candle"
	"github.com/thrasher-corp/gocryptotrader/database/repository/exchange"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

// OHLCVDatabaseStore stores kline candles
func OHLCVDatabaseStore(in *kline.Item) error {
	if in.Exchange == "" {
		return errors.New("name cannot be blank")
	}

	exchangeUUID, err := exchange.UUIDByName(in.Exchange)
	if err != nil {
		return err
	}

	databaseCandles := candle.Candle{
		ExchangeID: exchangeUUID.String(),
		Base:       in.Pair.Base.Upper().String(),
		Quote:      in.Pair.Quote.Upper().String(),
		Interval:   in.Interval.Short(),
		Asset:      in.Asset.String(),
	}

	for x := range in.Candles {
		databaseCandles.Tick = append(databaseCandles.Tick, candle.Tick{
			Timestamp: in.Candles[x].Time,
			Open:      in.Candles[x].Open,
			High:      in.Candles[x].High,
			Low:       in.Candles[x].Low,
			Close:     in.Candles[x].Close,
			Volume:    in.Candles[x].Volume,
		})
	}
	return candle.Insert(&databaseCandles)
}
