package trade

import (
	"sync"
	"testing"
	"testing/synctest"
	"time"
	"uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
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
		Enabled:  true,
		Driver:   database.DBSQLite3,
		Host:     "localhost",
		Database: "./rpctestdb",
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
	if !processor.started.Load() {
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

func TestSQLDataToTrade(t *testing.T) {
	t.Parallel()
	uuiderino := uuid.NewV4()
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
	require.NoError(t, err, "SQLDataToTrade must not error")
	require.Len(t, data, 1, "SQLDataToTrade must return one trade")
	assert.Equal(t, uuiderino, data[0].ID, "ID should decode")
	assert.Equal(t, order.Buy, data[0].Side, "Side should decode")
	assert.Equal(t, "BTCUSD", data[0].CurrencyPair.String(), "CurrencyPair should decode")
	assert.Equal(t, asset.Spot, data[0].AssetType, "AssetType should decode")

	_, err = SQLDataToTrade(sqltrade.Data{
		ID:        "not-a-uuid",
		Exchange:  "hello",
		Base:      currency.BTC.String(),
		Quote:     currency.USD.String(),
		AssetType: "spot",
		Side:      "buy",
	})
	assert.ErrorIs(t, err, errInvalidTradeID, "SQLDataToTrade should reject a malformed stored ID")
	assert.ErrorContains(t, err, `"not-a-uuid"`, "error should name the offending ID")
}

func TestTradeToSQLData(t *testing.T) {
	t.Parallel()
	cp := currency.NewBTCUSD()
	sqlData := tradeToSQLData(Data{
		Timestamp:    time.Now(),
		Exchange:     "test!",
		CurrencyPair: cp,
		AssetType:    asset.Spot,
		Price:        1337,
		Amount:       1337,
		Side:         order.Buy,
	})
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
	synctest.Test(t, func(t *testing.T) { //nolint:thelper,nolintlint // false positive
		var p Processor
		p.mutex.Lock()
		p.bufferProcessorInterval = time.Millisecond
		p.mutex.Unlock()
		var wg sync.WaitGroup
		wg.Add(1)
		go p.Run(&wg)
		wg.Wait()
		assert.True(t, p.started.Load(), "Run should report the processor started")
		// returns once the processor has drained an empty buffer and stopped its ticker
		synctest.Sleep(20 * time.Millisecond)
		assert.False(t, p.started.Load(), "Run should report the processor stopped")
	})
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
