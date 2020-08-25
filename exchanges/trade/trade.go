package trade

import (
	"errors"
	"sort"
	"sync/atomic"
	"time"

	"github.com/gofrs/uuid"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/log"
	sqltrade "github.com/thrasher-corp/gocryptotrader/database/repository/trade"
)

// Setup creates the trade processor if trading is supported
func (p *Processor) Setup(name string) {
	p.Name = name
	go p.Run()
}

// Shutdown kills the lingering processor
func (p *Processor) Shutdown() {
	close(p.shutdown)
}

// AddTradesToBuffer will push trade data onto the buffer
func (p *Processor) AddTradesToBuffer(data ...Data) {
	p.mutex.Lock()
	for i := range data {
		if data[i].Price == 0 ||
			data[i].Amount == 0 ||
			data[i].CurrencyPair.IsEmpty() ||
			data[i].Exchange == "" ||
			data[i].Timestamp.IsZero() {
			log.Errorf(log.WebsocketMgr, "%s received invalid trade data: %+v", p.Name, data[i])
			continue
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
			log.Errorf(log.WebsocketMgr, "%s uuid failed to generate for trade: %+v", p.Name, data[i])
			continue
		}
		data[i].ID = uu
		buffer = append(buffer, data[i])
	}
	p.mutex.Unlock()
}

// Processor will save trade data to the database in batches
func (p *Processor) Run() {
	if atomic.AddInt32(&p.started, 1) != 1 {
		log.Errorf(log.WebsocketMgr, "%s websocket trade processor already started", p.Name)
	}
	defer func() {
		atomic.CompareAndSwapInt32(&p.started, 1, 0)
	}()
	log.Debugf(log.OrderBook, "%s Order manager starting...", p.Name)
	timer := time.NewTicker(time.Minute)
	for {
		select {
		case <-p.shutdown:
			return
		case <-timer.C:
			log.Debugf(log.WebsocketMgr, "%s Processing websocket trade data", p.Name)
			p.mutex.Lock()
			sort.Sort(ByDate(buffer))
			err := sqltrade.Insert(buffer...)
			if err != nil {
				log.Error(log.Trade, err)
			}
			buffer = nil
			p.mutex.Unlock()
		}
	}
}

func TradeSqlToTrade(dbTrades ...sqltrade.Data) (result []Data, err error) {
	for i := range dbTrades {
		var cp currency.Pair
		cp, err = currency.NewPairFromString(dbTrades[i].CurrencyPair)
		if err != nil {
			return nil, err
		}
		var a = asset.Item(dbTrades[i].AssetType)
		if !asset.IsValid(a) {
			return nil, errors.New("invalid asset type lol")
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
			AssetType:     a,
			Price:        dbTrades[i].Price,
			Amount:       dbTrades[i].Amount,
			Side:        s,
		})
	}
	return result, nil
}

func ConvertTradesToCandles(interval kline.Interval, trades ...Data) (kline.Item, error) {
	if len(trades) == 0 {
		return kline.Item{}, errors.New("no trades supplied")
	}
	groupedData := convertTradeDatasToCandles(interval, trades...)
	candles :=  kline.Item{
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

func convertTradeDatasToCandles(interval kline.Interval, times ...Data) map[int64][]Data {
	groupedData := make(map[int64][]Data)
	for i:= range times {
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

func (c *CandleHolder) amendCandle(datas ...Data) {
	log.Debugf(log.WebsocketMgr, "Before: %v", c.candle)

	sort.Sort(ByDate(datas))
	c.trades = append(c.trades, datas...)
	sort.Sort(ByDate(c.trades))
	c.candle.Open = c.trades[0].Price
	c.candle.Close = c.trades[len(c.trades)-1].Price
	for i := range datas {
		c.candle.Volume += datas[i].Amount
	}
	for i := range c.trades {
		// some exchanges will send it as negative for sells
		// do they though?
		if c.trades[i].Price < 0 {
			log.Debug(log.WebsocketMgr, "NEGATIVE TRADE")
			c.trades[i].Price = c.trades[i].Price * -1
		}
		if c.trades[i].Amount < 0 {
			log.Debug(log.WebsocketMgr, "NEGATIVE TRADE")
			c.trades[i].Amount = c.trades[i].Amount * -1
		}
		if c.trades[i].Price < c.candle.Low || c.candle.Low == 0 {
			c.candle.Low = c.trades[i].Price
		}
		if c.trades[i].Price > c.candle.High {
			c.candle.High = c.trades[i].Price
		}
	}
	log.Debugf(log.WebsocketMgr, "After: %v", c.candle)
}

func classifyOHLCV (t time.Time, datas ...Data) (c kline.Candle) {
	sort.Sort(ByDate(datas))
	c.Open = datas[0].Price
	c.Close = datas[len(datas)-1].Price
	for i := range datas {
		// some exchanges will send it as negative for sells
		// do they though?
		if datas[i].Price < 0 {
			log.Debug(log.WebsocketMgr, "NEGATIVE TRADE")
			datas[i].Price = datas[i].Price * -1
		}
		if datas[i].Amount < 0 {
			log.Debug(log.WebsocketMgr, "NEGATIVE TRADE")
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