package database

import (
	"fmt"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

func LoadData(startDate, endDate time.Time, interval time.Duration, exchangeName, dataType string, fPair currency.Pair, a asset.Item) (*kline.DataFromKline, error) {
	resp := &kline.DataFromKline{}
	switch dataType {
	case common.CandleStr:
		datarino, err := getCandleDatabaseData(
			startDate,
			endDate,
			interval,
			exchangeName,
			fPair,
			a)
		if err != nil {
			return nil, err
		}
		resp.Item = datarino
	case common.TradeStr:
		trades, err := trade.GetTradesInRange(
			exchangeName,
			a.String(),
			fPair.Base.String(),
			fPair.Quote.String(),
			startDate,
			endDate)
		if err != nil {
			return nil, err
		}
		datarino, err := trade.ConvertTradesToCandles(
			gctkline.Interval(interval),
			trades...)
		if err != nil {
			return nil, err
		}
		resp.Item = datarino
	default:
		return nil, fmt.Errorf("unexpected database datatype: '%v'", dataType)
	}
	resp.Item.Exchange = strings.ToLower(resp.Item.Exchange)

	return resp, nil
}

func getCandleDatabaseData(startDate, endDate time.Time, interval time.Duration, exchangeName string, fPair currency.Pair, a asset.Item) (gctkline.Item, error) {
	datarino, err := gctkline.LoadFromDatabase(
		exchangeName,
		fPair,
		a,
		gctkline.Interval(interval),
		startDate,
		endDate)
	if err != nil {
		return gctkline.Item{}, err
	}
	return datarino, nil
}
