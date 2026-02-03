package kline

import (
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/drivers"
	"github.com/thrasher-corp/gocryptotrader/database/repository/candle"
	"github.com/thrasher-corp/gocryptotrader/database/repository/exchange"
	"github.com/thrasher-corp/gocryptotrader/database/testhelpers"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var (
	verbose       = false
	testExchanges = []exchange.Details{{Name: "one"}}
)

func TestValidateData(t *testing.T) {
	t.Parallel()
	err := validateData(nil)
	if err == nil {
		t.Error("error cannot be nil")
	}

	var empty []order.TradeHistory
	err = validateData(empty)
	if err == nil {
		t.Error("error cannot be nil")
	}

	tn := time.Now()
	trade1 := []order.TradeHistory{
		{Timestamp: tn.Add(2 * time.Minute), TID: "2"},
		{Timestamp: tn.Add(time.Minute), TID: "1"},
		{Timestamp: tn.Add(3 * time.Minute), TID: "3"},
	}

	err = validateData(trade1)
	if err == nil {
		t.Error("error cannot be nil")
	}

	trade2 := []order.TradeHistory{
		{Timestamp: tn.Add(2 * time.Minute), TID: "2", Amount: 1, Price: 0},
	}

	err = validateData(trade2)
	if err == nil {
		t.Error("error cannot be nil")
	}

	trade3 := []order.TradeHistory{
		{TID: "2", Amount: 1, Price: 0},
	}

	err = validateData(trade3)
	if err == nil {
		t.Error("error cannot be nil")
	}

	trade4 := []order.TradeHistory{
		{Timestamp: tn.Add(2 * time.Minute), TID: "2", Amount: 1, Price: 1000},
		{Timestamp: tn.Add(time.Minute), TID: "1", Amount: 1, Price: 1001},
		{Timestamp: tn.Add(3 * time.Minute), TID: "3", Amount: 1, Price: 1001.5},
	}

	err = validateData(trade4)
	assert.NoError(t, err)

	if trade4[0].TID != "1" || trade4[1].TID != "2" || trade4[2].TID != "3" {
		t.Error("trade history sorted incorrectly")
	}
}

func TestCreateKline(t *testing.T) {
	t.Parallel()

	pair := currency.NewBTCUSD()
	_, err := CreateKline(nil, OneMin, pair, asset.Spot, "Binance")
	require.ErrorIs(t, err, errInsufficientTradeData)

	tradeTotal := 24000
	trades := make([]order.TradeHistory, tradeTotal)
	execution := time.Now()
	for x := range tradeTotal {
		price, rndTime := 1000+float64(rand.Intn(1000)), rand.Intn(10) //nolint:gosec // no need to import crypo/rand for testing
		execution = execution.Add(time.Duration(rndTime) * time.Second)
		trades[x] = order.TradeHistory{
			Timestamp: execution,
			Amount:    1, // Keep as one for counting
			Price:     price,
		}
	}

	_, err = CreateKline(trades, 0, pair, asset.Spot, "Binance")
	require.ErrorIs(t, err, ErrInvalidInterval)

	c, err := CreateKline(trades, OneMin, pair, asset.Spot, "Binance")
	if err != nil {
		t.Fatal(err)
	}

	var amounts float64
	for x := range c.Candles {
		amounts += c.Candles[x].Volume
	}
	if amounts != float64(tradeTotal) {
		t.Fatalf("received: '%v' but expected '%v'", amounts, float64(tradeTotal))
	}
}

func TestKlineWord(t *testing.T) {
	t.Parallel()
	if OneDay.Word() != "oneday" {
		t.Fatalf("unexpected result: %v", OneDay.Word())
	}
}

func TestKlineDuration(t *testing.T) {
	t.Parallel()
	if OneDay.Duration() != time.Hour*24 {
		t.Fatalf("unexpected result: %v", OneDay.Duration())
	}
}

func TestKlineShort(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "24h", OneDay.Short(), "One day should show as 24h")
	assert.Equal(t, "1h", OneHour.Short(), "One hour should truncate 0m0s suffix")
	assert.Equal(t, "raw", Raw.Short(), "Raw should return raw")
}

func TestDurationToWord(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		interval Interval
	}{
		{"raw", Raw},
		{"tenmillisec", TenMilliseconds},
		{"twentymillisec", TwentyMilliseconds},
		{"hundredmillisec", HundredMilliseconds},
		{"twohundredfiftymillisec", TwoHundredAndFiftyMilliseconds},
		{"thousandmillisec", ThousandMilliseconds},
		{"tensec", TenSecond},
		{"fifteensecond", FifteenSecond},
		{"onemin", OneMin},
		{"threemin", ThreeMin},
		{"fivemin", FiveMin},
		{"tenmin", TenMin},
		{"fifteenmin", FifteenMin},
		{"thirtymin", ThirtyMin},
		{"onehour", OneHour},
		{"twohour", TwoHour},
		{"fourhour", FourHour},
		{"sixhour", SixHour},
		{"eighthour", OneHour * 8},
		{"twelvehour", TwelveHour},
		{"oneday", OneDay},
		{"threeday", ThreeDay},
		{"fiveday", FiveDay},
		{"fifteenday", FifteenDay},
		{"oneweek", OneWeek},
		{"twoweek", TwoWeek},
		{"onemonth", OneMonth},
		{"threemonth", ThreeMonth},
		{"sixmonth", SixMonth},
		{"oneyear", OneYear},
		{"notfound", Interval(time.Hour * 1337)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.name, strings.ToLower(tc.interval.Word()))
		})
	}
}

func TestTotalCandlesPerInterval(t *testing.T) {
	t.Parallel()

	tmNow := time.Now()
	assert.Equal(t, uint64(0), TotalCandlesPerInterval(tmNow.AddDate(0, 0, 1), tmNow, OneMin))

	start := time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	for _, tc := range []struct {
		interval Interval
		expected uint64
	}{
		{
			FifteenSecond,
			2102400,
		},
		{
			OneMin,
			525600,
		},
		{
			ThreeMin,
			175200,
		},
		{
			FiveMin,
			105120,
		},
		{
			TenMin,
			52560,
		},
		{
			FifteenMin,
			35040,
		},
		{
			ThirtyMin,
			17520,
		},
		{
			OneHour,
			8760,
		},
		{
			TwoHour,
			4380,
		},
		{
			FourHour,
			2190,
		},
		{
			SixHour,
			1460,
		},
		{
			OneHour * 8,
			1095,
		},
		{
			TwelveHour,
			730,
		},
		{
			OneDay,
			365,
		},
		{
			ThreeDay,
			121,
		},
		{
			FiveDay,
			73,
		},
		{
			FifteenDay,
			24,
		},
		{
			OneWeek,
			52,
		},
		{
			TwoWeek,
			26,
		},
		{
			OneMonth,
			12,
		},
		{
			OneYear,
			1,
		},
	} {
		t.Run(tc.interval.String(), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, TotalCandlesPerInterval(start, end, tc.interval))
		})
	}
}

func TestCalculateCandleDateRanges(t *testing.T) {
	t.Parallel()
	pt := time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC)
	ft := time.Date(2222, 1, 1, 0, 0, 0, 0, time.UTC)
	et := time.Date(2020, 1, 1, 1, 0, 0, 0, time.UTC)

	_, err := CalculateCandleDateRanges(time.Time{}, time.Time{}, OneMin, 300)
	assert.ErrorIs(t, err, common.ErrDateUnset)

	_, err = CalculateCandleDateRanges(et, pt, OneMin, 300)
	assert.ErrorIs(t, err, common.ErrStartAfterEnd)

	_, err = CalculateCandleDateRanges(et, ft, 0, 300)
	assert.ErrorIs(t, err, ErrInvalidInterval)

	_, err = CalculateCandleDateRanges(et, et, OneMin, 300)
	assert.ErrorIs(t, err, common.ErrStartEqualsEnd)

	v, err := CalculateCandleDateRanges(pt, et, OneWeek, 300)
	require.NoError(t, err)
	assert.Equal(t, int64(1546214400), v.Ranges[0].Start.Ticks)

	v, err = CalculateCandleDateRanges(pt, et, OneWeek, 100)
	require.NoError(t, err)
	assert.Equal(t, 1, len(v.Ranges))
	assert.Equal(t, 52, len(v.Ranges[0].Intervals))

	v, err = CalculateCandleDateRanges(et, ft, OneWeek, 5)
	require.NoError(t, err)
	assert.Equal(t, 2108, len(v.Ranges))
	assert.Equal(t, 5, len(v.Ranges[0].Intervals))
	lenRanges := len(v.Ranges) - 1
	lenIntervals := len(v.Ranges[lenRanges].Intervals) - 1
	assert.True(t, v.Ranges[lenRanges].Intervals[lenIntervals].End.Equal(ft.Round(OneWeek.Duration())))

	start := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	v, err = CalculateCandleDateRanges(start, end, OneDay, 0)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), v.Limit)
}

func TestItem_SortCandlesByTimestamp(t *testing.T) {
	t.Parallel()
	tempKline := Item{
		Exchange: "testExchange",
		Pair:     currency.NewBTCUSDT(),
		Asset:    asset.Spot,
		Interval: OneDay,
	}

	for x := range 100 {
		y := rand.Float64() //nolint:gosec // used for generating test data, no need to import crypo/rand
		tempKline.Candles = append(tempKline.Candles,
			Candle{
				Time:   time.Now().AddDate(0, 0, -x),
				Open:   y,
				High:   y + float64(x),
				Low:    y - float64(x),
				Close:  y,
				Volume: y,
			})
	}

	tempKline.SortCandlesByTimestamp(false)
	if tempKline.Candles[0].Time.After(tempKline.Candles[1].Time) {
		t.Fatal("expected kline.Candles to be in descending order")
	}

	tempKline.SortCandlesByTimestamp(true)
	if tempKline.Candles[0].Time.Before(tempKline.Candles[1].Time) {
		t.Fatal("expected kline.Candles to be in ascending order")
	}
}

func setupTest(t *testing.T) {
	t.Helper()
	if verbose {
		err := testhelpers.EnableVerboseTestOutput()
		require.NoError(t, err, "EnableVerboseTestOutput must not error")
	}

	testhelpers.MigrationDir = filepath.Join("..", "..", "database", "migrations")
	testhelpers.PostgresTestDatabase = testhelpers.GetConnectionDetails()
	testhelpers.TempDir = t.TempDir()
}

func TestStoreInDatabase(t *testing.T) {
	setupTest(t)

	for _, tc := range []struct {
		name   string
		config *database.Config
		seedDB func(bool) error
		runner func(t *testing.T)
		closer func(dbConn *database.Instance) error
	}{
		{
			name:   "postgresql",
			config: testhelpers.PostgresTestDatabase,
			seedDB: seedDB,
		},
		{
			name: "SQLite",
			config: &database.Config{
				Driver:            database.DBSQLite3,
				ConnectionDetails: drivers.ConnectionDetails{Database: "./testdb"},
			},
			seedDB: seedDB,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !testhelpers.CheckValidConfig(&tc.config.ConnectionDetails) {
				t.Skip("database not configured skipping test")
			}

			dbConn, err := testhelpers.ConnectToDatabase(tc.config)
			require.NoError(t, err)

			if tc.seedDB != nil {
				require.NoError(t, tc.seedDB(false))
			}

			_, ohlcvData, err := genOHCLVData()
			require.NoError(t, err)
			r, err := StoreInDatabase(&ohlcvData, false)
			require.NoError(t, err)
			assert.Equal(t, uint64(365), r)

			r, err = StoreInDatabase(&ohlcvData, true)
			require.NoError(t, err)
			assert.Equal(t, uint64(365), r)

			err = testhelpers.CloseDatabase(dbConn)
			assert.NoError(t, err)
		})
	}

	err := os.RemoveAll(testhelpers.TempDir)
	if err != nil {
		t.Fatalf("Failed to remove temp db file: %v", err)
	}
}

func TestLoadFromDatabase(t *testing.T) {
	setupTest(t)

	for _, tc := range []struct {
		name   string
		config *database.Config
		seedDB func(bool) error
		runner func(t *testing.T)
		closer func(dbConn *database.Instance) error
	}{
		{
			name:   "postgresql",
			config: testhelpers.PostgresTestDatabase,
			seedDB: seedDB,
		},
		{
			name: "SQLite",
			config: &database.Config{
				Driver:            database.DBSQLite3,
				ConnectionDetails: drivers.ConnectionDetails{Database: "./testdb"},
			},
			seedDB: seedDB,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !testhelpers.CheckValidConfig(&tc.config.ConnectionDetails) {
				t.Skip("database not configured skipping test")
			}

			dbConn, err := testhelpers.ConnectToDatabase(tc.config)
			require.NoError(t, err)

			if tc.seedDB != nil {
				require.NoError(t, tc.seedDB(true))
			}
			start := time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC)
			end := start.AddDate(1, 0, 0)
			ret, err := LoadFromDatabase(testExchanges[0].Name, currency.NewBTCUSDT(), asset.Spot, OneDay, start, end)
			require.NoError(t, err)
			assert.Equal(t, ret.Exchange, testExchanges[0].Name)

			err = testhelpers.CloseDatabase(dbConn)
			assert.NoError(t, err)
		})
	}

	err := os.RemoveAll(testhelpers.TempDir)
	if err != nil {
		t.Fatalf("Failed to remove temp db file: %v", err)
	}
}

// seedDB seeds test data into the database
// TODO: find a better way to handle this to remove duplication between candle test
func seedDB(includeOHLCVData bool) error {
	err := exchange.InsertMany(testExchanges)
	if err != nil {
		return err
	}

	if includeOHLCVData {
		data, _, err := genOHCLVData()
		if err != nil {
			return err
		}
		_, err = candle.Insert(&data)
		return err
	}
	return nil
}

func genOHCLVData() (out candle.Item, outItem Item, err error) {
	exchangeUUID, err := exchange.UUIDByName(testExchanges[0].Name)
	if err != nil {
		return
	}

	out.ExchangeID = exchangeUUID.String()
	out.Base = currency.BTC.String()
	out.Quote = currency.USDT.String()
	out.Interval = 86400
	out.Asset = "spot"

	start := time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC)
	for x := range 365 {
		out.Candles = append(out.Candles, candle.Candle{
			Timestamp: start.Add(time.Hour * 24 * time.Duration(x)),
			Open:      1000,
			High:      1000,
			Low:       1000,
			Close:     1000,
			Volume:    1000,
		})
	}

	outItem.Interval = OneDay
	outItem.Asset = asset.Spot
	outItem.Pair = currency.NewBTCUSDT()
	outItem.Exchange = testExchanges[0].Name

	for x := range 365 {
		outItem.Candles = append(outItem.Candles, Candle{
			Time:   start.Add(time.Hour * 24 * time.Duration(x)),
			Open:   1000,
			High:   1000,
			Low:    1000,
			Close:  1000,
			Volume: 1000,
		})
	}

	return out, outItem, nil
}

func TestLoadCSV(t *testing.T) {
	v, err := LoadFromGCTScriptCSV(filepath.Join("..", "..", "testdata", "binance_BTCUSDT_24h_2019_01_01_2020_01_01.csv"))
	if err != nil {
		t.Fatal(err)
	}

	if v[0].Time.UTC() != time.Unix(1546300800, 0).UTC() {
		t.Fatalf("unexpected value received: %v", v[0].Time)
	}

	if v[269].Close != 8177.91 {
		t.Fatalf("unexpected value received: %v", v[269].Close)
	}

	if v[364].Open != 7246 {
		t.Fatalf("unexpected value received: %v", v[364].Open)
	}
}

func TestVerifyResultsHaveData(t *testing.T) {
	t.Parallel()
	tt1 := time.Now().Round(OneDay.Duration())
	tt2 := tt1.Add(OneDay.Duration())
	tt3 := tt2.Add(OneDay.Duration()) // end date no longer inclusive
	dateRanges, err := CalculateCandleDateRanges(tt1, tt3, OneDay, 0)
	assert.NoError(t, err)

	if dateRanges.HasDataAtDate(tt1) {
		t.Error("unexpected true value")
	}
	err = dateRanges.SetHasDataFromCandles([]Candle{
		{
			Time: tt1,
			Low:  1337,
		},
		{
			Time: tt2,
		},
	})
	assert.NoError(t, err)

	if !dateRanges.HasDataAtDate(tt1) {
		t.Error("expected true")
	}
	err = dateRanges.SetHasDataFromCandles([]Candle{
		{
			Time: tt1,
		},
		{
			Time: tt2,
			Low:  1337,
		},
	})
	assert.NoError(t, err)

	if dateRanges.HasDataAtDate(tt1) {
		t.Error("expected false")
	}
}

func TestDataSummary(t *testing.T) {
	t.Parallel()
	tt1 := time.Now().Add(-time.Hour * 24).Round(OneDay.Duration())
	tt2 := time.Now().Round(OneDay.Duration())
	tt3 := time.Now().Add(time.Hour * 24).Round(OneDay.Duration())
	dateRanges, err := CalculateCandleDateRanges(tt1, tt2, OneDay, 0)
	assert.NoError(t, err)

	result := dateRanges.DataSummary(false)
	if len(result) != 1 {
		t.Errorf("expected %v received %v", 1, len(result))
	}
	dateRanges, err = CalculateCandleDateRanges(tt1, tt3, OneDay, 0)
	assert.NoError(t, err)

	dateRanges.Ranges[0].Intervals[0].HasData = true
	result = dateRanges.DataSummary(true)
	if len(result) != 2 {
		t.Errorf("expected %v received %v", 2, len(result))
	}
	result = dateRanges.DataSummary(false)
	if len(result) != 1 {
		t.Errorf("expected %v received %v", 1, len(result))
	}
}

func TestHasDataAtDate(t *testing.T) {
	t.Parallel()
	tt1 := time.Now().Round(OneDay.Duration())
	tt2 := tt1.Add(OneDay.Duration())
	tt3 := tt2.Add(OneDay.Duration()) // end date no longer inclusive
	dateRanges, err := CalculateCandleDateRanges(tt1, tt3, OneDay, 0)
	assert.NoError(t, err)

	if dateRanges.HasDataAtDate(tt2) {
		t.Error("unexpected true value")
	}

	err = dateRanges.SetHasDataFromCandles([]Candle{
		{
			Time:  tt1,
			Close: 1337,
		},
		{
			Time:  tt2,
			Close: 1337,
		},
	})
	assert.NoError(t, err)

	if !dateRanges.HasDataAtDate(tt2) {
		t.Error("unexpected false value")
	}

	if dateRanges.HasDataAtDate(tt2.Add(time.Hour * 24)) {
		t.Error("should not have data")
	}
}

func TestIntervalsPerYear(t *testing.T) {
	t.Parallel()
	var i Interval
	if i.IntervalsPerYear() != 0 {
		t.Error("expected 0")
	}
	i = OneYear
	if i.IntervalsPerYear() != 1.0 {
		t.Error("expected 1")
	}
	i = OneDay
	if i.IntervalsPerYear() != 365 {
		t.Error("expected 365")
	}
	i = OneHour
	if i.IntervalsPerYear() != 8760 {
		t.Error("expected 8670")
	}
	i = TwoHour
	if i.IntervalsPerYear() != 4380 {
		t.Error("expected 4380")
	}
	i = TwoHour + FifteenSecond
	if i.IntervalsPerYear() != 4370.893970893971 {
		t.Error("expected 4370...")
	}
}

func TestConvertToNewInterval(t *testing.T) {
	_, err := (*Item)(nil).ConvertToNewInterval(OneMin)
	assert.ErrorIs(t, err, errNilKline)

	_, err = (&Item{}).ConvertToNewInterval(OneMin)
	assert.ErrorIs(t, err, ErrInvalidInterval)

	old := &Item{
		Exchange: "lol",
		Pair:     currency.NewBTCUSDT(),
		Asset:    asset.Spot,
		Interval: OneDay,
		Candles: []Candle{
			{
				Time:   time.Now(),
				Open:   1337,
				High:   1339,
				Low:    1336,
				Close:  1338,
				Volume: 1337,
			},
			{
				Time:   time.Now().AddDate(0, 0, 1),
				Open:   1338,
				High:   2000,
				Low:    1332,
				Close:  1696,
				Volume: 6420,
			},
			{
				Time:   time.Now().AddDate(0, 0, 2),
				Open:   1696,
				High:   1998,
				Low:    1337,
				Close:  6969,
				Volume: 2520,
			},
		},
	}

	_, err = old.ConvertToNewInterval(0)
	assert.ErrorIs(t, err, ErrInvalidInterval)

	_, err = old.ConvertToNewInterval(OneMin)
	assert.ErrorIs(t, err, ErrCanOnlyUpscaleCandles)

	old.Interval = ThreeDay
	_, err = old.ConvertToNewInterval(OneWeek)
	assert.ErrorIs(t, err, ErrWholeNumberScaling)

	old.Interval = OneDay
	newInterval := ThreeDay
	newCandle, err := old.ConvertToNewInterval(newInterval)
	require.NoError(t, err)

	if len(newCandle.Candles) != 1 {
		t.Error("expected one candle")
	}
	if newCandle.Candles[0].Open != 1337 &&
		newCandle.Candles[0].High != 2000 &&
		newCandle.Candles[0].Low != 1332 &&
		newCandle.Candles[0].Close != 6969 &&
		newCandle.Candles[0].Volume != (2520+6420+1337) {
		t.Error("unexpected updoot")
	}

	old.Candles = append(old.Candles, Candle{
		Time:   time.Now().AddDate(0, 0, 3),
		Open:   6969,
		High:   1998,
		Low:    2342,
		Close:  7777,
		Volume: 111,
	})
	newCandle, err = old.ConvertToNewInterval(newInterval)
	assert.NoError(t, err)

	if len(newCandle.Candles) != 1 {
		t.Error("expected one candle")
	}

	_, err = old.ConvertToNewInterval(OneMonth)
	assert.ErrorIs(t, err, ErrInsufficientCandleData)

	tn := time.Now().Truncate(time.Duration(OneDay))

	// Test incorrectly padded candles
	old.Candles = []Candle{
		{
			Time:   tn,
			Open:   1337,
			High:   1339,
			Low:    1336,
			Close:  1338,
			Volume: 1337,
		},
		{
			Time:   tn.AddDate(0, 0, 1),
			Open:   1338,
			High:   2000,
			Low:    1332,
			Close:  1696,
			Volume: 6420,
		},
		{
			Time:   tn.AddDate(0, 0, 2),
			Open:   1696,
			High:   1998,
			Low:    1337,
			Close:  6969,
			Volume: 2520,
		},
		// empty candle should be here <---
		// aaaand empty candle should be here <---
		{
			Time:   tn.AddDate(0, 0, 5),
			Open:   6969,
			High:   8888,
			Low:    1111,
			Close:  5555,
			Volume: 2520,
		},
		{
			Time: tn.AddDate(0, 0, 6),
			// Empty end padding
		},
		{
			Time: tn.AddDate(0, 0, 7),
			// Empty end padding
		},
		{
			Time: tn.AddDate(0, 0, 8),
			// Empty end padding
		},
	}

	_, err = old.ConvertToNewInterval(newInterval)
	assert.ErrorIs(t, err, errCandleDataNotPadded)

	err = old.addPadding(tn, tn.AddDate(0, 0, 9), false)
	require.NoError(t, err)

	newCandle, err = old.ConvertToNewInterval(newInterval)
	require.NoError(t, err)

	if len(newCandle.Candles) != 3 {
		t.Errorf("received '%v' expected '%v'", len(newCandle.Candles), 3)
	}
}

func TestAddPadding(t *testing.T) {
	t.Parallel()

	tn := time.Now().Truncate(time.Duration(OneDay))

	var k *Item
	err := k.addPadding(tn, tn.AddDate(0, 0, 5), false)
	require.ErrorIs(t, err, errNilKline)

	k = &Item{}
	k.Candles = []Candle{
		{
			Time:   tn,
			Open:   1337,
			High:   1339,
			Low:    1336,
			Close:  1338,
			Volume: 1337,
		},
	}
	err = k.addPadding(tn, tn.AddDate(0, 0, 5), false)
	require.ErrorIs(t, err, ErrInvalidInterval)

	k.Interval = OneDay
	k.Candles = []Candle{
		{
			Time:   tn.AddDate(0, 0, 1),
			Open:   1338,
			High:   2000,
			Low:    1332,
			Close:  1696,
			Volume: 6420,
		},
		{
			Time:   tn,
			Open:   1337,
			High:   1339,
			Low:    1336,
			Close:  1338,
			Volume: 1337,
		},
	}
	err = k.addPadding(tn.AddDate(0, 0, 5), tn, false)
	require.ErrorIs(t, err, errCannotEstablishTimeWindow)

	k.Candles = []Candle{
		{
			Time:   tn.Add(time.Hour * 8),
			Open:   1337,
			High:   1339,
			Low:    1336,
			Close:  1338,
			Volume: 1337,
		},
		{
			Time:   tn.AddDate(0, 0, 1).Add(time.Hour * 8),
			Open:   1338,
			High:   2000,
			Low:    1332,
			Close:  1696,
			Volume: 6420,
		},
		{
			Time:   tn.AddDate(0, 0, 2).Add(time.Hour * 8),
			Open:   1696,
			High:   1998,
			Low:    1337,
			Close:  6969,
			Volume: 2520,
		},
	}

	err = k.addPadding(tn, tn.AddDate(0, 0, 3), false)
	require.ErrorIs(t, err, errCandleOpenTimeIsNotUTCAligned)

	k.Candles = []Candle{
		{
			Time:   tn,
			Open:   1337,
			High:   1339,
			Low:    1336,
			Close:  1338,
			Volume: 1337,
		},
		{
			Time:   tn.AddDate(0, 0, 1),
			Open:   1338,
			High:   2000,
			Low:    1332,
			Close:  1696,
			Volume: 6420,
		},
		{
			Time:   tn.AddDate(0, 0, 2),
			Open:   1696,
			High:   1998,
			Low:    1337,
			Close:  6969,
			Volume: 2520,
		},
	}

	err = k.addPadding(tn, tn.AddDate(0, 0, 3), false)
	require.NoError(t, err)

	if len(k.Candles) != 3 {
		t.Fatalf("received '%v' expected '%v'", len(k.Candles), 3)
	}

	k.Candles = append(k.Candles, Candle{
		Time:   tn.AddDate(0, 0, 5),
		Open:   6969,
		High:   8888,
		Low:    1111,
		Close:  5555,
		Volume: 2520,
	})

	err = k.addPadding(tn, tn.AddDate(0, 0, 6), false)
	require.NoError(t, err)

	if len(k.Candles) != 6 {
		t.Fatalf("received '%v' expected '%v'", len(k.Candles), 6)
	}

	// No candles test when there is zero activity for that period
	k.Candles = nil

	err = k.addPadding(tn, tn.AddDate(0, 0, 6), false)
	assert.NoError(t, err)

	if len(k.Candles) != 6 {
		t.Errorf("received '%v' expected '%v'", len(k.Candles), 6)
	}
}

func TestGetClosePriceAtTime(t *testing.T) {
	t.Parallel()
	tt := time.Now()
	k := Item{
		Candles: []Candle{
			{
				Time:  tt,
				Close: 1337,
			},
			{
				Time:  tt.Add(time.Hour),
				Close: 1338,
			},
		},
	}
	price, err := k.GetClosePriceAtTime(tt)
	assert.NoError(t, err)

	if price != 1337 {
		t.Errorf("received '%v' expected '%v'", price, 1337)
	}
	_, err = k.GetClosePriceAtTime(tt.Add(time.Minute))
	assert.ErrorIs(t, err, ErrNotFoundAtTime)
}

func TestDeployExchangeIntervals(t *testing.T) {
	t.Parallel()
	exchangeIntervals := DeployExchangeIntervals()
	if exchangeIntervals.ExchangeSupported(OneWeek) {
		t.Errorf("received '%v' expected '%v'", exchangeIntervals.ExchangeSupported(OneWeek), false)
	}

	exchangeIntervals = DeployExchangeIntervals(IntervalCapacity{Interval: OneWeek})
	if !exchangeIntervals.ExchangeSupported(OneWeek) {
		t.Errorf("received '%v' expected '%v'", exchangeIntervals.ExchangeSupported(OneWeek), true)
	}

	_, err := exchangeIntervals.Construct(0)
	assert.ErrorIs(t, err, ErrInvalidInterval)

	_, err = exchangeIntervals.Construct(OneMin)
	assert.ErrorIs(t, err, ErrCannotConstructInterval)

	request, err := exchangeIntervals.Construct(OneWeek)
	assert.NoError(t, err)

	if request != OneWeek {
		t.Errorf("received '%v' expected '%v'", request, OneWeek)
	}

	exchangeIntervals = DeployExchangeIntervals(IntervalCapacity{Interval: OneWeek}, IntervalCapacity{Interval: OneDay})

	request, err = exchangeIntervals.Construct(OneMonth)
	assert.NoError(t, err)

	if request != OneDay {
		t.Errorf("received '%v' expected '%v'", request, OneDay)
	}
}

func TestSetHasDataFromCandles(t *testing.T) {
	t.Parallel()
	ohc := getOneHour()
	localEnd := ohc[len(ohc)-1].Time.Add(OneHour.Duration())
	i, err := CalculateCandleDateRanges(ohc[0].Time, localEnd, OneHour, 100000)
	assert.NoError(t, err)

	err = i.SetHasDataFromCandles(ohc)
	assert.NoError(t, err)

	if !i.Start.Equal(ohc[0].Time) {
		t.Errorf("received '%v' expected '%v'", i.Start.Time, ohc[0].Time)
	}
	if !i.End.Equal(localEnd) {
		t.Errorf("received '%v' expected '%v'", i.End.Time, ohc[len(ohc)-1].Time)
	}

	k := Item{
		Interval: OneHour,
		Candles:  ohc[2:],
	}
	err = k.addPadding(i.Start.Time, i.End.Time, false)
	assert.NoError(t, err)

	err = i.SetHasDataFromCandles(k.Candles)
	assert.NoError(t, err)

	if !i.Start.Equal(k.Candles[0].Time) {
		t.Errorf("received '%v' expected '%v'", i.Start.Time, k.Candles[0].Time)
	}
	if i.HasDataAtDate(k.Candles[0].Time) {
		t.Errorf("received '%v' expected '%v'", false, true)
	}
	if !i.HasDataAtDate(k.Candles[len(k.Candles)-1].Time) {
		t.Errorf("received '%v' expected '%v'", true, false)
	}
}

func TestGetIntervalResultLimit(t *testing.T) {
	t.Parallel()

	var e *ExchangeCapabilitiesEnabled
	_, err := e.GetIntervalResultLimit(OneMin)
	assert.ErrorIs(t, err, errExchangeCapabilitiesEnabledIsNil)

	e = &ExchangeCapabilitiesEnabled{}
	e.Intervals = ExchangeIntervals{}
	_, err = e.GetIntervalResultLimit(OneDay)
	assert.ErrorIs(t, err, errIntervalNotSupported)

	e.Intervals = ExchangeIntervals{
		supported: map[Interval]uint64{
			OneDay: 100000,
			OneMin: 0,
		},
	}

	_, err = e.GetIntervalResultLimit(OneMin)
	assert.ErrorIs(t, err, errCannotFetchIntervalLimit)

	limit, err := e.GetIntervalResultLimit(OneDay)
	assert.NoError(t, err)

	if limit != 100000 {
		t.Errorf("received '%v' expected '%v'", limit, 100000)
	}

	e.GlobalResultLimit = 1337
	limit, err = e.GetIntervalResultLimit(OneMin)
	assert.NoError(t, err)

	if limit != 1337 {
		t.Errorf("received '%v' expected '%v'", limit, 1337)
	}
}

func TestUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var i Interval
	for _, tt := range []struct {
		in  string
		exp Interval
	}{{`"3m"`, ThreeMin}, {`"15s"`, FifteenSecond}, {`720000000000`, OneMin * 12}, {`"-1ns"`, Raw}, {`"raw"`, Raw}} {
		err := i.UnmarshalJSON([]byte(tt.in))
		assert.NoErrorf(t, err, "UnmarshalJSON should not error on %q", tt.in)
	}
	err := i.UnmarshalJSON([]byte(`"6hedgehogs"`))
	assert.ErrorContains(t, err, "unknown unit", "UnmarshalJSON should error")
}
