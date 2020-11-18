package database

import (
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	kline2 "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

func LoadData(configOverride *database.Config, startDate, endDate time.Time, interval time.Duration, exchangeName, dataType string, fPair currency.Pair, a asset.Item) (*kline.DataFromKline, error) {
	var resp *kline.DataFromKline
	var err error
	if configOverride != nil {
		engine.Bot.Config.Database = *configOverride
		err = engine.Bot.DatabaseManager.Start(engine.Bot)
		if err != nil {
			return nil, err
		}
	}
	defer func() {
		err = engine.Bot.DatabaseManager.Stop()
		if err != nil {
			log.Error(log.BackTester, err)
		}
	}()
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
			kline2.Interval(interval),
			trades...)
		if err != nil {
			return nil, err
		}
		resp.Item = datarino
	default:
		return nil, fmt.Errorf("unexpected database datatype: '%v'", dataType)
	}
	return resp, nil
}

func getCandleDatabaseData(startDate, endDate time.Time, interval time.Duration, exchangeName string, fPair currency.Pair, a asset.Item) (kline2.Item, error) {
	datarino, err := kline2.LoadFromDatabase(
		exchangeName,
		fPair,
		a,
		kline2.Interval(interval),
		startDate,
		endDate)
	if err != nil {
		return kline2.Item{}, err
	}
	return datarino, nil
}
