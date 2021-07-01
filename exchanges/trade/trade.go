package trade

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	tradesql "github.com/thrasher-corp/gocryptotrader/database/repository/trade"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// Setup creates the trade processor if trading is supported
func (p *Processor) setup(wg *sync.WaitGroup) {
	p.mutex.Lock()
	p.bufferProcessorInterval = BufferProcessorIntervalTime
	p.mutex.Unlock()
	go p.Run(wg)
}

// AddTradesToBuffer will push trade data onto the buffer
func AddTradesToBuffer(exchangeName string, data ...Data) error {
	cfg := database.DB.GetConfig()
	if database.DB == nil || cfg == nil || !cfg.Enabled {
		return nil
	}
	if len(data) == 0 {
		return nil
	}
	var errs common.Errors
	if atomic.AddInt32(&processor.started, 0) == 0 {
		var wg sync.WaitGroup
		wg.Add(1)
		processor.setup(&wg)
		wg.Wait()
	}
	var validDatas []Data
	for i := range data {
		if data[i].Price == 0 ||
			data[i].Amount == 0 ||
			data[i].CurrencyPair.IsEmpty() ||
			data[i].Exchange == "" ||
			data[i].Timestamp.IsZero() {
			errs = append(errs, fmt.Errorf("%v received invalid trade data: %+v", exchangeName, data[i]))
			continue
		}

		if data[i].Price < 0 {
			data[i].Price *= -1
			data[i].Side = order.Sell
		}
		if data[i].Amount < 0 {
			data[i].Amount *= -1
			data[i].Side = order.Sell
		}
		if data[i].Side == order.Bid {
			data[i].Side = order.Buy
		}
		if data[i].Side == order.Ask {
			data[i].Side = order.Sell
		}
		uu, err := uuid.NewV4()
		if err != nil {
			errs = append(errs, fmt.Errorf("%s uuid failed to generate for trade: %+v", exchangeName, data[i]))
		}
		data[i].ID = uu
		validDatas = append(validDatas, data[i])
	}
	processor.mutex.Lock()
	processor.buffer = append(processor.buffer, validDatas...)
	processor.mutex.Unlock()
	if len(errs) > 0 {
		return errs
	}
	return nil
}

// Run will save trade data to the database in batches
func (p *Processor) Run(wg *sync.WaitGroup) {
	wg.Done()
	if !atomic.CompareAndSwapInt32(&p.started, 0, 1) {
		log.Error(log.Trade, "trade processor already started")
		return
	}
	defer func() {
		atomic.CompareAndSwapInt32(&p.started, 1, 0)
	}()
	p.mutex.Lock()
	ticker := time.NewTicker(p.bufferProcessorInterval)
	p.mutex.Unlock()
	for {
		<-ticker.C
		p.mutex.Lock()
		bufferCopy := append(p.buffer[:0:0], p.buffer...)
		p.buffer = nil
		p.mutex.Unlock()
		if len(bufferCopy) == 0 {
			ticker.Stop()
			return
		}
		err := SaveTradesToDatabase(bufferCopy...)
		if err != nil {
			log.Error(log.Trade, err)
		}
	}
}

// SaveTradesToDatabase converts trades and saves results to database
func SaveTradesToDatabase(trades ...Data) error {
	sqlTrades, err := tradeToSQLData(trades...)
	if err != nil {
		return err
	}
	return tradesql.Insert(sqlTrades...)
}

// GetTradesInRange calls db function to return trades in range
// to minimise tradesql package usage
func GetTradesInRange(exchangeName, assetType, base, quote string, startDate, endDate time.Time) ([]Data, error) {
	if exchangeName == "" || assetType == "" || base == "" || quote == "" || startDate.IsZero() || endDate.IsZero() {
		return nil, errors.New("invalid arguments received")
	}
	results, err := tradesql.GetInRange(exchangeName, assetType, base, quote, startDate, endDate)
	if err != nil {
		return nil, err
	}
	return SQLDataToTrade(results...)
}

// HasTradesInRanges Creates an executes an SQL query to verify if a trade exists within a timeframe
func HasTradesInRanges(exchangeName, assetType, base, quote string, rangeHolder *kline.IntervalRangeHolder) error {
	if exchangeName == "" || assetType == "" || base == "" || quote == "" {
		return errors.New("invalid arguments received")
	}
	return tradesql.VerifyTradeInIntervals(exchangeName, assetType, base, quote, rangeHolder)
}

func tradeToSQLData(trades ...Data) ([]tradesql.Data, error) {
	sort.Sort(ByDate(trades))
	var results []tradesql.Data
	for i := range trades {
		tradeID, err := uuid.NewV4()
		if err != nil {
			return nil, err
		}
		results = append(results, tradesql.Data{
			ID:        tradeID.String(),
			Timestamp: trades[i].Timestamp,
			Exchange:  trades[i].Exchange,
			Base:      trades[i].CurrencyPair.Base.String(),
			Quote:     trades[i].CurrencyPair.Quote.String(),
			AssetType: trades[i].AssetType.String(),
			Price:     trades[i].Price,
			Amount:    trades[i].Amount,
			Side:      trades[i].Side.String(),
			TID:       trades[i].TID,
		})
	}
	return results, nil
}

// SQLDataToTrade converts sql data to glorious trade data
func SQLDataToTrade(dbTrades ...tradesql.Data) (result []Data, err error) {
	for i := range dbTrades {
		var cp currency.Pair
		cp, err = currency.NewPairFromStrings(dbTrades[i].Base, dbTrades[i].Quote)
		if err != nil {
			return nil, err
		}
		cp = cp.Upper()
		var a = asset.Item(dbTrades[i].AssetType)
		if !a.IsValid() {
			return nil, fmt.Errorf("invalid asset type %v", a)
		}
		var s order.Side
		s, err = order.StringToOrderSide(dbTrades[i].Side)
		if err != nil {
			return nil, err
		}
		result = append(result, Data{
			ID:           uuid.FromStringOrNil(dbTrades[i].ID),
			Timestamp:    dbTrades[i].Timestamp.UTC(),
			Exchange:     dbTrades[i].Exchange,
			CurrencyPair: cp.Upper(),
			AssetType:    a,
			Price:        dbTrades[i].Price,
			Amount:       dbTrades[i].Amount,
			Side:         s,
		})
	}
	return result, nil
}

// ConvertTradesToCandles turns trade data into kline.Items
func ConvertTradesToCandles(interval kline.Interval, trades ...Data) (kline.Item, error) {
	if len(trades) == 0 {
		return kline.Item{}, ErrNoTradesSupplied
	}
	groupedData := groupTradesToInterval(interval, trades...)
	candles := kline.Item{
		Exchange: trades[0].Exchange,
		Pair:     trades[0].CurrencyPair,
		Asset:    trades[0].AssetType,
		Interval: interval,
	}
	for k, v := range groupedData {
		candles.Candles = append(candles.Candles, classifyOHLCV(time.Unix(k, 0), v...))
	}

	return candles, nil
}

func groupTradesToInterval(interval kline.Interval, times ...Data) map[int64][]Data {
	groupedData := make(map[int64][]Data)
	for i := range times {
		nearestInterval := getNearestInterval(times[i].Timestamp, interval)
		groupedData[nearestInterval] = append(
			groupedData[nearestInterval],
			times[i],
		)
	}
	return groupedData
}

func getNearestInterval(t time.Time, interval kline.Interval) int64 {
	return t.Truncate(interval.Duration()).UTC().Unix()
}

func classifyOHLCV(t time.Time, datas ...Data) (c kline.Candle) {
	sort.Sort(ByDate(datas))
	c.Open = datas[0].Price
	c.Close = datas[len(datas)-1].Price
	for i := range datas {
		if datas[i].Price < 0 {
			datas[i].Price *= -1
		}
		if datas[i].Amount < 0 {
			datas[i].Amount *= -1
		}
		if datas[i].Price < c.Low || c.Low == 0 {
			c.Low = datas[i].Price
		}
		if datas[i].Price > c.High {
			c.High = datas[i].Price
		}
		c.Volume += datas[i].Amount
	}
	c.Time = t
	return c
}

// FilterTradesByTime removes any trades that are not between the start
// and end times
func FilterTradesByTime(trades []Data, startTime, endTime time.Time) []Data {
	if startTime.IsZero() || endTime.IsZero() {
		// can't filter without boundaries
		return trades
	}
	var filteredTrades []Data
	for i := range trades {
		if trades[i].Timestamp.After(startTime) && trades[i].Timestamp.Before(endTime) {
			filteredTrades = append(filteredTrades, trades[i])
		}
	}

	return filteredTrades
}
