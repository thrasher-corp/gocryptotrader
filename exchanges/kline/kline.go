package kline

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// CreateKline creates candles out of trade history data for a set time interval
func CreateKline(trades []order.TradeHistory, interval time.Duration, p currency.Pair, a asset.Item, exchange string) (Item, error) {
	if interval < time.Minute {
		return Item{}, fmt.Errorf("invalid time interval: [%s]", interval)
	}

	err := validateData(trades)
	if err != nil {
		return Item{}, err
	}

	timeIntervalStart := trades[0].Timestamp.Truncate(interval)
	timeIntervalEnd := trades[len(trades)-1].Timestamp

	// Adds time interval buffer zones
	var timeIntervalCache [][]order.TradeHistory
	var OpenClose []HeartBeat

	for t := timeIntervalStart; t.Before(timeIntervalEnd); t = t.Add(interval) {
		timeBufferEnd := t.Add(interval)
		insertionCount := 0

		var zonedTradeHistory []order.TradeHistory
		for i := 0; i < len(trades); i++ {
			if (trades[i].Timestamp.After(t) ||
				trades[i].Timestamp.Equal(t)) &&
				(trades[i].Timestamp.Before(timeBufferEnd) ||
					trades[i].Timestamp.Equal(timeBufferEnd)) {
				zonedTradeHistory = append(zonedTradeHistory, trades[i])
				insertionCount++
				continue
			}
			trades = trades[i:]
			break
		}

		// Insert dummy in time period when there is no price action
		if insertionCount == 0 {
			OpenClose = append(OpenClose, HeartBeat{Open: t, Close: timeBufferEnd})
			timeIntervalCache = append(timeIntervalCache, []order.TradeHistory{})
			continue
		}
		OpenClose = append(OpenClose, HeartBeat{Open: t, Close: timeBufferEnd})
		timeIntervalCache = append(timeIntervalCache, zonedTradeHistory)
	}

	var candles = Item{
		Exchange: exchange,
		Pair:     p,
		Asset:    a,
		Interval: interval,
	}

	var closePriceOfLast float64
	for x := range timeIntervalCache {
		if len(timeIntervalCache[x]) == 0 {
			candles.Candles = append(candles.Candles, Candle{
				Time:  OpenClose[x].Open,
				High:  closePriceOfLast,
				Low:   closePriceOfLast,
				Close: closePriceOfLast,
				Open:  closePriceOfLast})
			continue
		}

		var newCandle Candle
		for y := range timeIntervalCache[x] {
			if y == 0 {
				newCandle.Open = timeIntervalCache[x][y].Price
				newCandle.Time = OpenClose[x].Open
			}
			if y == len(timeIntervalCache[x])-1 {
				newCandle.Close = timeIntervalCache[x][y].Price
				closePriceOfLast = timeIntervalCache[x][y].Price
			}
			if newCandle.High < timeIntervalCache[x][y].Price {
				newCandle.High = timeIntervalCache[x][y].Price
			}
			if newCandle.Low > timeIntervalCache[x][y].Price || newCandle.Low == 0 {
				newCandle.Low = timeIntervalCache[x][y].Price
			}
			newCandle.Volume += timeIntervalCache[x][y].Amount
		}
		candles.Candles = append(candles.Candles, newCandle)
	}
	return candles, nil
}

// validatData checks for zero values on data and sorts before turning
// converting into OHLC
func validateData(trades []order.TradeHistory) error {
	if len(trades) < 2 {
		return errors.New("insufficient data")
	}

	for i := range trades {
		if trades[i].Timestamp.IsZero() ||
			trades[i].Timestamp.Unix() == 0 {
			return fmt.Errorf("timestamp not set for element %d", i)
		}

		if trades[i].Amount == 0 {
			return fmt.Errorf("amount not set for element %d", i)
		}

		if trades[i].Price == 0 {
			return fmt.Errorf("price not set for element %d", i)
		}
	}

	sort.Slice(trades, func(i, j int) bool {
		return trades[i].Timestamp.Before(trades[j].Timestamp)
	})
	return nil
}
