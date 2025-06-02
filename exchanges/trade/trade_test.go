package trade

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/drivers"
	sqltrade "github.com/thrasher-corp/gocryptotrader/database/repository/trade"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func TestAddTradesToBuffer(t *testing.T) {
	t.Parallel()
	processor.mutex.Lock()
	processor.bufferProcessorInterval = BufferProcessorIntervalTime
	processor.mutex.Unlock()
	dbConf := database.Config{
		Enabled: true,
		Driver:  database.DBSQLite3,
		ConnectionDetails: drivers.ConnectionDetails{
			Host:     "localhost",
			Database: "./rpctestdb",
		},
	}
	var wg sync.WaitGroup
	wg.Add(1)
	processor.setup(&wg)
	wg.Wait()
	err := database.DB.SetConfig(&dbConf)
	if err != nil {
		t.Error(err)
	}
	err = AddTradesToBuffer([]Data{
		{
			Timestamp:    time.Now(),
			Exchange:     "test!",
			CurrencyPair: currency.NewBTCUSD(),
			AssetType:    asset.Spot,
			Price:        1337,
			Amount:       1337,
			Side:         order.Buy,
		},
	}...)
	if err != nil {
		t.Error(err)
	}
	if atomic.AddInt32(&processor.started, 0) == 0 {
		t.Error("expected the processor to have started")
	}

	err = AddTradesToBuffer([]Data{
		{
			Timestamp:    time.Now(),
			Exchange:     "test!",
			CurrencyPair: currency.NewBTCUSD(),
			AssetType:    asset.Spot,
			Price:        0,
			Amount:       0,
			Side:         order.Buy,
		},
	}...)
	if err == nil {
		t.Error("expected error")
	}
	processor.mutex.Lock()
	processor.buffer = nil
	processor.mutex.Unlock()

	err = AddTradesToBuffer([]Data{
		{
			Timestamp:    time.Now(),
			Exchange:     "test!",
			CurrencyPair: currency.NewBTCUSD(),
			AssetType:    asset.Spot,
			Price:        -1,
			Amount:       -1,
		},
	}...)
	if err != nil {
		t.Error(err)
	}
	processor.mutex.Lock()
	if processor.buffer[0].Amount != 1 {
		t.Error("expected positive amount")
	}
	if processor.buffer[0].Side != order.Sell {
		t.Error("expected unknown side")
	}
	processor.mutex.Unlock()
}

func TestSqlDataToTrade(t *testing.T) {
	t.Parallel()
	uuiderino, _ := uuid.NewV4()
	data, err := SQLDataToTrade(sqltrade.Data{
		ID:        uuiderino.String(),
		Timestamp: time.Time{},
		Exchange:  "hello",
		Base:      currency.BTC.String(),
		Quote:     currency.USD.String(),
		AssetType: "spot",
		Price:     1337,
		Amount:    1337,
		Side:      "buy",
	})
	if err != nil {
		t.Error(err)
	}
	if len(data) != 1 {
		t.Fatal("unexpected scenario")
	}
	if data[0].Side != order.Buy {
		t.Error("expected buy side")
	}
	if data[0].CurrencyPair.String() != "BTCUSD" {
		t.Errorf("expected \"BTCUSD\", got %v", data[0].CurrencyPair)
	}
	if data[0].AssetType != asset.Spot {
		t.Error("expected spot")
	}
}

func TestTradeToSQLData(t *testing.T) {
	t.Parallel()
	cp := currency.NewBTCUSD()
	sqlData, err := tradeToSQLData(Data{
		Timestamp:    time.Now(),
		Exchange:     "test!",
		CurrencyPair: cp,
		AssetType:    asset.Spot,
		Price:        1337,
		Amount:       1337,
		Side:         order.Buy,
	})
	if err != nil {
		t.Error(err)
	}
	if len(sqlData) != 1 {
		t.Fatal("unexpected result")
	}
	if sqlData[0].Base != cp.Base.String() {
		t.Errorf("expected \"BTC\", got %v", sqlData[0].Base)
	}
	if sqlData[0].AssetType != asset.Spot.String() {
		t.Error("expected spot")
	}
}

func TestConvertTradesToCandles(t *testing.T) {
	t.Parallel()
	cp := currency.NewBTCUSD()
	startDate := time.Date(2020, 1, 1, 1, 0, 0, 0, time.UTC)
	candles, err := ConvertTradesToCandles(kline.FifteenSecond, []Data{
		{
			Timestamp:    startDate,
			Exchange:     "test!",
			CurrencyPair: cp,
			AssetType:    asset.Spot,
			Price:        1337,
			Amount:       1337,
			Side:         order.Buy,
		},
		{
			Timestamp:    startDate.Add(time.Second),
			Exchange:     "test!",
			CurrencyPair: cp,
			AssetType:    asset.Spot,
			Price:        1337,
			Amount:       1337,
			Side:         order.Buy,
		},
		{
			Timestamp:    startDate.Add(time.Minute),
			Exchange:     "test!",
			CurrencyPair: cp,
			AssetType:    asset.Spot,
			Price:        -1337,
			Amount:       -1337,
			Side:         order.Buy,
		},
	}...)
	if err != nil {
		t.Fatal(err)
	}
	if len(candles.Candles) != 2 {
		t.Fatal("unexpected candle amount")
	}
	if candles.Interval != kline.FifteenSecond {
		t.Error("expected fifteen seconds")
	}
}

func TestShutdown(t *testing.T) {
	t.Parallel()
	var p Processor
	p.mutex.Lock()
	p.bufferProcessorInterval = time.Millisecond
	p.mutex.Unlock()
	var wg sync.WaitGroup
	wg.Add(1)
	go p.Run(&wg)
	wg.Wait()
	if atomic.LoadInt32(&p.started) != 1 {
		t.Error("expected it to start running")
	}
	time.Sleep(time.Millisecond * 20)
	if atomic.LoadInt32(&p.started) != 0 {
		t.Error("expected it to stop running")
	}
}

func TestFilterTradesByTime(t *testing.T) {
	t.Parallel()
	trades := []Data{
		{
			Exchange:  "test",
			Timestamp: time.Now().Add(-time.Second),
		},
	}
	trades = FilterTradesByTime(trades, time.Now().Add(-time.Minute), time.Now())
	if len(trades) != 1 {
		t.Error("failed to filter")
	}
	trades = FilterTradesByTime(trades, time.Now().Add(-time.Millisecond), time.Now())
	if len(trades) != 0 {
		t.Error("failed to filter")
	}
}

func TestSaveTradesToDatabase(t *testing.T) {
	t.Parallel()
	err := SaveTradesToDatabase(Data{})
	if err != nil && err.Error() != "exchange name/uuid not set, cannot insert" {
		t.Error(err)
	}
}

func TestGetTradesInRange(t *testing.T) {
	t.Parallel()
	_, err := GetTradesInRange("", "", "", "", time.Time{}, time.Time{})
	if err != nil && err.Error() != "invalid arguments received" {
		t.Error(err)
	}
}
