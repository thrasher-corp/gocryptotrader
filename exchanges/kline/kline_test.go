package kline

import (
	"errors"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
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
	testExchanges = []exchange.Details{
		{
			Name: "one",
		},
	}
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
	if err != nil {
		t.Error(err)
	}

	if trade4[0].TID != "1" || trade4[1].TID != "2" || trade4[2].TID != "3" {
		t.Error("trade history sorted incorrectly")
	}
}

func TestCreateKline(t *testing.T) {
	t.Parallel()
	c, err := CreateKline(nil,
		OneMin,
		currency.NewPair(currency.BTC, currency.USD),
		asset.Spot,
		"Binance")
	if err == nil {
		t.Fatal("error cannot be nil")
	}

	var trades []order.TradeHistory
	rand.Seed(time.Now().Unix())
	for i := 0; i < 24000; i++ {
		trades = append(trades, order.TradeHistory{
			Timestamp: time.Now().Add((time.Duration(rand.Intn(10)) * time.Minute) + // nolint:gosec // no need to import crypo/rand for testing
				(time.Duration(rand.Intn(10)) * time.Second)), // nolint:gosec // no need to import crypo/rand for testing
			TID:    crypto.HexEncodeToString([]byte(string(rune(i)))),
			Amount: float64(rand.Intn(20)) + 1,      // nolint:gosec // no need to import crypo/rand for testing
			Price:  1000 + float64(rand.Intn(1000)), // nolint:gosec // no need to import crypo/rand for testing
		})
	}

	c, err = CreateKline(trades,
		0,
		currency.NewPair(currency.BTC, currency.USD),
		asset.Spot,
		"Binance")
	if err == nil {
		t.Fatal("error cannot be nil")
	}

	c, err = CreateKline(trades,
		OneMin,
		currency.NewPair(currency.BTC, currency.USD),
		asset.Spot,
		"Binance")
	if err != nil {
		t.Fatal(err)
	}

	if len(c.Candles) == 0 {
		t.Fatal("no data returned, expecting a lot.")
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
	if OneDay.Short() != "24h" {
		t.Fatalf("unexpected result: %v", OneDay.Short())
	}
}

func TestDurationToWord(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		interval Interval
	}{
		{
			"FifteenSecond",
			FifteenSecond,
		},
		{
			"OneMin",
			OneMin,
		},
		{
			"ThreeMin",
			ThreeMin,
		},
		{
			"FiveMin",
			FiveMin,
		},
		{
			"TenMin",
			TenMin,
		},
		{
			"FifteenMin",
			FifteenMin,
		},
		{
			"ThirtyMin",
			ThirtyMin,
		},
		{
			"OneHour",
			OneHour,
		},
		{
			"TwoHour",
			TwoHour,
		},
		{
			"FourHour",
			FourHour,
		},
		{
			"SixHour",
			SixHour,
		},
		{
			"EightHour",
			OneHour * 8,
		},
		{
			"TwelveHour",
			TwelveHour,
		},
		{
			"OneDay",
			OneDay,
		},
		{
			"ThreeDay",
			ThreeDay,
		},
		{
			"FifteenDay",
			FifteenDay,
		},
		{
			"OneWeek",
			OneWeek,
		},
		{
			"TwoWeek",
			TwoWeek,
		},
		{
			"OneMonth",
			OneMonth,
		},
		{
			"OneYear",
			OneYear,
		},
		{
			"notfound",
			Interval(time.Hour * 1337),
		},
	}
	for x := range testCases {
		test := testCases[x]
		t.Run(test.name, func(t *testing.T) {
			v := durationToWord(test.interval)
			if !strings.EqualFold(v, test.name) {
				t.Fatalf("%v: received %v expected %v", test.name, v, test.name)
			}
		})
	}
}

func TestKlineErrors(t *testing.T) {
	t.Parallel()
	v := ErrorKline{
		Interval: OneYear,
		Pair:     currency.NewPair(currency.BTC, currency.AUD),
		Err:      errors.New("hello world"),
	}

	if v.Interval != OneYear {
		t.Fatalf("expected OneYear received %v:", v.Interval)
	}

	if v.Pair != currency.NewPair(currency.BTC, currency.AUD) {
		t.Fatalf("expected OneYear received %v:", v.Pair)
	}

	if v.Error() != "hello world" {
		t.Fatal("expected error return received empty value")
	}

	if v.Unwrap().Error() != "hello world" {
		t.Fatal("expected error return received empty value")
	}
}

func TestTotalCandlesPerInterval(t *testing.T) {
	t.Parallel()
	start := time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	testCases := []struct {
		name     string
		interval Interval
		expected float64
	}{
		{
			"FifteenSecond",
			FifteenSecond,
			2102400,
		},
		{
			"OneMin",
			OneMin,
			525600,
		},
		{
			"ThreeMin",
			ThreeMin,
			175200,
		},
		{
			"FiveMin",
			FiveMin,
			105120,
		},
		{
			"TenMin",
			TenMin,
			52560,
		},
		{
			"FifteenMin",
			FifteenMin,
			35040,
		},
		{
			"ThirtyMin",
			ThirtyMin,
			17520,
		},
		{
			"OneHour",
			OneHour,
			8760,
		},
		{
			"TwoHour",
			TwoHour,
			4380,
		},
		{
			"FourHour",
			FourHour,
			2190,
		},
		{
			"SixHour",
			SixHour,
			1460,
		},
		{
			"EightHour",
			OneHour * 8,
			1095,
		},
		{
			"TwelveHour",
			TwelveHour,
			730,
		},
		{
			"OneDay",
			OneDay,
			365,
		},
		{
			"ThreeDay",
			ThreeDay,
			121.66666666666667,
		},
		{
			"FifteenDay",
			FifteenDay,
			24.333333333333332,
		},
		{
			"OneWeek",
			OneWeek,
			52.142857142857146,
		},
		{
			"TwoWeek",
			TwoWeek,
			26.071428571428573,
		},
		{
			"OneMonth",
			OneMonth,
			12.166666666666666,
		},
		{
			"OneYear",
			OneYear,
			1,
		},
	}
	for x := range testCases {
		test := testCases[x]
		t.Run(test.name, func(t *testing.T) {
			v := TotalCandlesPerInterval(start, end, test.interval)
			if v != test.expected {
				t.Fatalf("%v: received %v expected %v", test.name, v, test.expected)
			}
		})
	}
}

func TestCalculateCandleDateRanges(t *testing.T) {
	t.Parallel()
	pt := time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC)
	ft := time.Date(2222, 1, 1, 0, 0, 0, 0, time.UTC)
	et := time.Date(2020, 1, 1, 1, 0, 0, 0, time.UTC)
	nt := time.Time{}

	_, err := CalculateCandleDateRanges(nt, nt, OneMin, 300)
	if !errors.Is(err, common.ErrDateUnset) {
		t.Errorf("received %v expected %v", err, common.ErrDateUnset)
	}

	_, err = CalculateCandleDateRanges(et, pt, OneMin, 300)
	if !errors.Is(err, common.ErrStartAfterEnd) {
		t.Errorf("received %v expected %v", err, common.ErrStartAfterEnd)
	}

	_, err = CalculateCandleDateRanges(et, ft, 0, 300)
	if !errors.Is(err, ErrUnsetInterval) {
		t.Errorf("received %v expected %v", err, ErrUnsetInterval)
	}

	_, err = CalculateCandleDateRanges(et, et, OneMin, 300)
	if !errors.Is(err, common.ErrStartEqualsEnd) {
		t.Errorf("received %v expected %v", err, common.ErrStartEqualsEnd)
	}

	v, err := CalculateCandleDateRanges(pt, et, OneWeek, 300)
	if err != nil {
		t.Error(err)
	}

	if !v.Ranges[0].Start.Time.Equal(time.Unix(1546214400, 0)) {
		t.Errorf("expected %v received %v", 1546214400, v.Ranges[0].Start.Ticks)
	}

	v, err = CalculateCandleDateRanges(pt, et, OneWeek, 100)
	if err != nil {
		t.Error(err)
	}
	if len(v.Ranges) != 1 {
		t.Fatalf("expected %v received %v", 1, len(v.Ranges))
	}
	if len(v.Ranges[0].Intervals) != 52 {
		t.Errorf("expected %v received %v", 52, len(v.Ranges[0].Intervals))
	}
	v, err = CalculateCandleDateRanges(et, ft, OneWeek, 5)
	if err != nil {
		t.Error(err)
	}
	if len(v.Ranges) != 2108 {
		t.Errorf("expected %v received %v", 2108, len(v.Ranges))
	}
	if len(v.Ranges[0].Intervals) != 5 {
		t.Errorf("expected %v received %v", 5, len(v.Ranges[0].Intervals))
	}
	if len(v.Ranges[1].Intervals) != 5 {
		t.Errorf("expected %v received %v", 5, len(v.Ranges[1].Intervals))
	}
	lenRanges := len(v.Ranges) - 1
	lenIntervals := len(v.Ranges[lenRanges].Intervals) - 1
	if !v.Ranges[lenRanges].Intervals[lenIntervals].End.Equal(ft.Round(OneWeek.Duration())) {
		t.Errorf("expected %v received %v", ft.Round(OneDay.Duration()), v.Ranges[lenRanges].Intervals[lenIntervals].End)
	}
}

func TestItem_SortCandlesByTimestamp(t *testing.T) {
	t.Parallel()
	var tempKline = Item{
		Exchange: "testExchange",
		Pair:     currency.NewPair(currency.BTC, currency.USDT),
		Asset:    asset.Spot,
		Interval: OneDay,
	}

	for x := 0; x < 100; x++ {
		y := rand.Float64() // nolint:gosec // used for generating test data, no need to import crypo/rand
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
	if verbose {
		testhelpers.EnableVerboseTestOutput()
	}

	var err error
	testhelpers.MigrationDir = filepath.Join("..", "..", "database", "migrations")
	testhelpers.PostgresTestDatabase = testhelpers.GetConnectionDetails()
	testhelpers.TempDir, err = ioutil.TempDir("", "gct-temp")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
}

func TestStoreInDatabase(t *testing.T) {
	setupTest(t)

	testCases := []struct {
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
	}

	for x := range testCases {
		test := testCases[x]

		t.Run(test.name, func(t *testing.T) {
			if !testhelpers.CheckValidConfig(&test.config.ConnectionDetails) {
				t.Skip("database not configured skipping test")
			}

			dbConn, err := testhelpers.ConnectToDatabase(test.config)
			if err != nil {
				t.Fatal(err)
			}

			if test.seedDB != nil {
				err = test.seedDB(false)
				if err != nil {
					t.Error(err)
				}
			}

			_, ohlcvData, err := genOHCLVData()
			if err != nil {
				t.Fatal(err)
			}
			r, err := StoreInDatabase(&ohlcvData, false)
			if err != nil {
				t.Fatal(err)
			}

			if r != 365 {
				t.Fatalf("unexpected number inserted: %v", r)
			}

			r, err = StoreInDatabase(&ohlcvData, true)
			if err != nil {
				t.Fatal(err)
			}

			if r != 365 {
				t.Fatalf("unexpected number inserted: %v", r)
			}

			err = testhelpers.CloseDatabase(dbConn)
			if err != nil {
				t.Error(err)
			}
		})
	}

	err := os.RemoveAll(testhelpers.TempDir)
	if err != nil {
		t.Fatalf("Failed to remove temp db file: %v", err)
	}
}

func TestLoadFromDatabase(t *testing.T) {
	setupTest(t)

	testCases := []struct {
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
	}

	for x := range testCases {
		test := testCases[x]

		t.Run(test.name, func(t *testing.T) {
			if !testhelpers.CheckValidConfig(&test.config.ConnectionDetails) {
				t.Skip("database not configured skipping test")
			}

			dbConn, err := testhelpers.ConnectToDatabase(test.config)
			if err != nil {
				t.Fatal(err)
			}

			if test.seedDB != nil {
				err = test.seedDB(true)
				if err != nil {
					t.Error(err)
				}
			}

			p, err := currency.NewPairFromString("BTCUSDT")
			if err != nil {
				t.Fatal(err)
			}
			start := time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC)
			end := start.AddDate(1, 0, 0)
			ret, err := LoadFromDatabase(testExchanges[0].Name, p, asset.Spot, OneDay, start, end)
			if err != nil {
				t.Fatal(err)
			}
			if ret.Exchange != testExchanges[0].Name {
				t.Fatalf("uncorrect data returned: %v", ret.Exchange)
			}

			err = testhelpers.CloseDatabase(dbConn)
			if err != nil {
				t.Error(err)
			}
		})
	}

	err := os.RemoveAll(testhelpers.TempDir)
	if err != nil {
		t.Fatalf("Failed to remove temp db file: %v", err)
	}
}

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
	for x := 0; x < 365; x++ {
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
	outItem.Pair = currency.NewPair(currency.BTC, currency.USDT)
	outItem.Exchange = testExchanges[0].Name

	for x := 0; x < 365; x++ {
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
	tt2 := time.Now().Round(OneDay.Duration())
	tt1 := time.Now().Add(-time.Hour * 24).Round(OneDay.Duration())
	dateRanges, err := CalculateCandleDateRanges(tt1, tt2, OneDay, 0)
	if err != nil {
		t.Error(err)
	}
	if dateRanges.HasDataAtDate(tt1) {
		t.Error("unexpected true value")
	}
	dateRanges.SetHasDataFromCandles([]Candle{
		{
			Time: tt1,
		},
	})
	if !dateRanges.HasDataAtDate(tt1) {
		t.Error("expected true")
	}
	dateRanges.SetHasDataFromCandles([]Candle{
		{
			Time: tt2,
		},
	})
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
	if err != nil {
		t.Error(err)
	}
	result := dateRanges.DataSummary(false)
	if len(result) != 1 {
		t.Errorf("expected %v received %v", 1, len(result))
	}
	dateRanges, err = CalculateCandleDateRanges(tt1, tt3, OneDay, 0)
	if err != nil {
		t.Error(err)
	}
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
	tt2 := time.Now().Round(OneDay.Duration())
	tt1 := time.Now().Add(-time.Hour * 24 * 30).Round(OneDay.Duration())
	dateRanges, err := CalculateCandleDateRanges(tt1, tt2, OneDay, 0)
	if err != nil {
		t.Error(err)
	}
	if dateRanges.HasDataAtDate(tt1) {
		t.Error("unexpected true value")
	}

	dateRanges.SetHasDataFromCandles([]Candle{
		{
			Time: tt1,
		},
		{
			Time: tt2,
		},
	})

	if !dateRanges.HasDataAtDate(tt1.Round(OneDay.Duration())) {
		t.Error("unexpected false value")
	}

	if dateRanges.HasDataAtDate(tt2.Add(time.Hour * 24 * 26)) {
		t.Error("should not have data")
	}
}

func TestIntervalsPerYear(t *testing.T) {
	t.Parallel()
	i := OneYear
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

// The purpose of this benchmark is to highlight that requesting
// '.Unix()` frequently is a slow process
func BenchmarkJustifyIntervalTimeStoringUnixValues1(b *testing.B) {
	tt1 := time.Now()
	tt2 := time.Now().Add(-time.Hour)
	tt3 := time.Now().Add(time.Hour)
	for i := 0; i < b.N; i++ {
		if tt1.Unix() == tt2.Unix() || // nolint:staticcheck // it is a benchmark to demonstrate inefficiency in calling
			(tt1.Unix() > tt2.Unix() && tt1.Unix() < tt3.Unix()) {

		}
	}
}

// The purpose of this benchmark is to highlight that storing the unix value
// at time of creation is dramatically faster than frequently requesting `.Unix()`
// at runtime at scale. When dealing with the backtester and comparing
// tens of thousands of candle times
func BenchmarkJustifyIntervalTimeStoringUnixValues2(b *testing.B) {
	tt1 := time.Now().Unix()
	tt2 := time.Now().Add(-time.Hour).Unix()
	tt3 := time.Now().Add(time.Hour).Unix()
	for i := 0; i < b.N; i++ {
		if tt1 >= tt2 && tt1 <= tt3 { // nolint:staticcheck // it is a benchmark to demonstrate inefficiency in calling

		}
	}
}
