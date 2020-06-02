package engine

import (
	"errors"

	modelPSQL "github.com/thrasher-corp/gocryptotrader/database/models/postgres"
	"github.com/thrasher-corp/gocryptotrader/database/repository/candle"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/volatiletech/null"
)

func OHLCVDatabaseStore(in *kline.Item) error {
	if in.Exchange == "" {
		return errors.New("name cannot be blank")
	}

	exchangeUUID, err := ExchangeUUIDByName(in.Exchange)
	if err != nil {
		return err
	}

	var databaseCandles []modelPSQL.Candle
	for x := range in.Candles {
		databaseCandles = append(databaseCandles, modelPSQL.Candle{
			ExchangeID: null.NewString(exchangeUUID.String(), true),
			Base:       in.Pair.Base.Upper().String(),
			Quote:      in.Pair.Quote.Upper().String(),
			Date:       in.Candles[x].Time,
			Open:       in.Candles[x].Open,
			High:       in.Candles[x].High,
			Low:        in.Candles[x].Low,
			Close:      in.Candles[x].Close,
			Volume:     in.Candles[x].Volume,
			Interval:   in.Interval.Short(),
		})
	}
	return candle.InsertMany(&databaseCandles)
}
