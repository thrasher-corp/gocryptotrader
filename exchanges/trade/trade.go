package trade

import (
	"errors"
	"fmt"
	"sort"
	"sync/atomic"
	"time"

	"github.com/gofrs/uuid"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	sqltrade "github.com/thrasher-corp/gocryptotrader/database/repository/trade"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// Setup creates the trade processor if trading is supported
func (p *Processor) setup() {
	go p.Run()
}

// AddTradesToBuffer will push trade data onto the buffer
func AddTradesToBuffer(exchangeName string, data ...Data) error {
	if database.DB == nil || database.DB.Config == nil || !database.DB.Config.Enabled {
		return nil
	}
	var errs common.Errors
	if atomic.AddInt32(&processor.started, 0) == 0 {
		processor.setup()
	}
	processor.mutex.Lock()
	for i := range data {
		if data[i].Price == 0 ||
			data[i].Amount == 0 ||
			data[i].CurrencyPair.IsEmpty() ||
			data[i].Exchange == "" ||
			data[i].Timestamp.IsZero() {
			errs = append(errs, fmt.Errorf("%v received invalid trade data: %+v", exchangeName, data[i]))
		}

		if data[i].Price < 0 {
			data[i].Price = data[i].Price * -1
		}
		if data[i].Amount < 0 {
			data[i].Amount = data[i].Amount * -1
		}
		if data[i].Side == "" {
			data[i].Side = order.UnknownSide
		}
		uu, err := uuid.NewV4()
		if err != nil {
			errs = append(errs, fmt.Errorf("%s uuid failed to generate for trade: %+v", exchangeName, data[i]))
		}
		data[i].ID = uu
		buffer = append(buffer, data[i])
	}
	processor.mutex.Unlock()
	if len(errs) > 0 {
		return errs
	}
	return nil
}

// Processor will save trade data to the database in batches
func (p *Processor) Run() {
	if !atomic.CompareAndSwapInt32(&p.started, 0, 1) {
		log.Error(log.Trade, "trade processor already started")
	}
	defer func() {
		atomic.CompareAndSwapInt32(&p.started, 1, 0)
	}()
	log.Info(log.Trade, "trade processor starting...")
	p.mutex.Lock()
	ticker := time.NewTicker(bufferProcessorInterval)
	p.mutex.Unlock()
	for {
		select {
		case <-ticker.C:
			p.mutex.Lock()
			if len(buffer) == 0 {
				p.mutex.Unlock()
				log.Infof(log.Trade, "no trade data received in %v, shutting down", bufferProcessorInterval)
				return
			}
			results := tradeToSQLData()
			err := sqltrade.Insert(results...)
			if err != nil {
				log.Error(log.Trade, err)
			}
			buffer = nil
			p.mutex.Unlock()
		}
	}
}

func tradeToSQLData() []sqltrade.Data {
	sort.Sort(ByDate(buffer))
	var results []sqltrade.Data
	for i := range buffer {
		results = append(results, sqltrade.Data{
			ID:        buffer[i].ID.String(),
			Timestamp: buffer[i].Timestamp.Unix(),
			Exchange:  buffer[i].Exchange,
			Base:      buffer[i].CurrencyPair.Base.String(),
			Quote:     buffer[i].CurrencyPair.Quote.String(),
			AssetType: buffer[i].AssetType.String(),
			Price:     buffer[i].Price,
			Amount:    buffer[i].Amount,
			Side:      buffer[i].Side.String(),
		})
	}
	return results
}

// SqlDataToTrade converts sql data to glorious trade data
func SqlDataToTrade(dbTrades ...sqltrade.Data) (result []Data, err error) {
	for i := range dbTrades {
		var cp currency.Pair
		cp, err = currency.NewPairFromStrings(dbTrades[i].Base, dbTrades[i].Quote)
		if err != nil {
			return nil, err
		}
		cp = cp.Upper()
		var a = asset.Item(dbTrades[i].AssetType)
		if !asset.IsValid(a) {
			return nil, fmt.Errorf("invalid asset type %v", a)
		}
		var s order.Side
		s, err = order.StringToOrderSide(dbTrades[i].Side)
		if err != nil {
			return nil, err
		}
		result = append(result, Data{
			ID:           uuid.FromStringOrNil(dbTrades[i].ID),
			Timestamp:    time.Unix(dbTrades[i].Timestamp, 0),
			Exchange:     dbTrades[i].Exchange,
			CurrencyPair: cp,
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
		return kline.Item{}, errors.New("no trades supplied")
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
	return t.Truncate(interval.Duration()).Unix()
}

func classifyOHLCV(t time.Time, datas ...Data) (c kline.Candle) {
	sort.Sort(ByDate(datas))
	c.Open = datas[0].Price
	c.Close = datas[len(datas)-1].Price
	for i := range datas {
		// some exchanges will send it as negative for sells
		// do they though?
		if datas[i].Price < 0 {
			datas[i].Price = datas[i].Price * -1
		}
		if datas[i].Amount < 0 {
			datas[i].Amount = datas[i].Amount * -1
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
