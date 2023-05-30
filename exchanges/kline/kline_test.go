package kline

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	if trade4[0].TID != "1" || trade4[1].TID != "2" || trade4[2].TID != "3" {
		t.Error("trade history sorted incorrectly")
	}
}

func TestCreateKline(t *testing.T) {
	t.Parallel()

	pair := currency.NewPair(currency.BTC, currency.USD)
	_, err := CreateKline(nil, OneMin, pair, asset.Spot, "Binance")
	if !errors.Is(err, errInsufficientTradeData) {
		t.Fatalf("received: '%v' but expected '%v'", err, errInsufficientTradeData)
	}

	tradeTotal := 24000
	var trades []order.TradeHistory
	execution := time.Now()
	for i := 0; i < tradeTotal; i++ {
		price, rndTime := 1000+float64(rand.Intn(1000)), rand.Intn(10) //nolint:gosec // no need to import crypo/rand for testing
		execution = execution.Add(time.Duration(rndTime) * time.Second)
		trades = append(trades, order.TradeHistory{
			Timestamp: execution,
			Amount:    1, // Keep as one for counting
			Price:     price,
		})
	}

	_, err = CreateKline(trades, 0, pair, asset.Spot, "Binance")
	if !errors.Is(err, ErrInvalidInterval) {
		t.Fatalf("received: '%v' but expected '%v'", err, ErrInvalidInterval)
	}

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
			"hundredmillisec",
			HundredMilliseconds,
		},
		{
			"thousandmillisec",
			ThousandMilliseconds,
		},
		{
			"tensec",
			TenSecond,
		},
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
			"FiveDay",
			FiveDay,
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
			"ThreeMonth",
			ThreeMonth,
		},
		{
			"SixMonth",
			SixMonth,
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
			t.Parallel()
			t.Helper()
			v := durationToWord(test.interval)
			if !strings.EqualFold(v, test.name) {
				t.Fatalf("%v: received %v expected %v", test.name, v, test.name)
			}
		})
	}
}

func TestTotalCandlesPerInterval(t *testing.T) {
	t.Parallel()
	start := time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	testCases := []struct {
		name     string
		interval Interval
		expected int64
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
			121,
		},
		{
			"FiveDay",
			FiveDay,
			73,
		},
		{
			"FifteenDay",
			FifteenDay,
			24,
		},
		{
			"OneWeek",
			OneWeek,
			52,
		},
		{
			"TwoWeek",
			TwoWeek,
			26,
		},
		{
			"OneMonth",
			OneMonth,
			12,
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
			t.Parallel()
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
	if !errors.Is(err, ErrInvalidInterval) {
		t.Errorf("received %v expected %v", err, ErrInvalidInterval)
	}

	_, err = CalculateCandleDateRanges(et, et, OneMin, 300)
	if !errors.Is(err, common.ErrStartEqualsEnd) {
		t.Errorf("received %v expected %v", err, common.ErrStartEqualsEnd)
	}

	v, err := CalculateCandleDateRanges(pt, et, OneWeek, 300)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	if !v.Ranges[0].Start.Time.Equal(time.Unix(1546214400, 0)) {
		t.Errorf("expected %v received %v", 1546214400, v.Ranges[0].Start.Ticks)
	}

	v, err = CalculateCandleDateRanges(pt, et, OneWeek, 100)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if len(v.Ranges) != 1 {
		t.Fatalf("expected %v received %v", 1, len(v.Ranges))
	}
	if len(v.Ranges[0].Intervals) != 52 {
		t.Errorf("expected %v received %v", 52, len(v.Ranges[0].Intervals))
	}
	v, err = CalculateCandleDateRanges(et, ft, OneWeek, 5)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
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
		if err != nil {
			fmt.Printf("failed to enable verbose test output: %v", err)
			os.Exit(1)
		}
	}

	var err error
	testhelpers.MigrationDir = filepath.Join("..", "..", "database", "migrations")
	testhelpers.PostgresTestDatabase = testhelpers.GetConnectionDetails()
	testhelpers.TempDir, err = os.MkdirTemp("", "gct-temp")
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
				t.Fatalf("incorrect data returned: %v", ret.Exchange)
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
	tt1 := time.Now().Round(OneDay.Duration())
	tt2 := tt1.Add(OneDay.Duration())
	tt3 := tt2.Add(OneDay.Duration()) // end date no longer inclusive
	dateRanges, err := CalculateCandleDateRanges(tt1, tt3, OneDay, 0)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
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
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

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
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
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
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	result := dateRanges.DataSummary(false)
	if len(result) != 1 {
		t.Errorf("expected %v received %v", 1, len(result))
	}
	dateRanges, err = CalculateCandleDateRanges(tt1, tt3, OneDay, 0)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
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
	tt1 := time.Now().Round(OneDay.Duration())
	tt2 := tt1.Add(OneDay.Duration())
	tt3 := tt2.Add(OneDay.Duration()) // end date no longer inclusive
	dateRanges, err := CalculateCandleDateRanges(tt1, tt3, OneDay, 0)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
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
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

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

// The purpose of this benchmark is to highlight that requesting
// '.Unix()` frequently is a slow process
func BenchmarkJustifyIntervalTimeStoringUnixValues1(b *testing.B) {
	tt1 := time.Now()
	tt2 := time.Now().Add(-time.Hour)
	tt3 := time.Now().Add(time.Hour)
	for i := 0; i < b.N; i++ {
		if tt1.Unix() == tt2.Unix() || (tt1.Unix() > tt2.Unix() && tt1.Unix() < tt3.Unix()) {
			continue
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
		if tt1 >= tt2 && tt1 <= tt3 {
			continue
		}
	}
}

func TestConvertToNewInterval(t *testing.T) {
	_, err := (*Item)(nil).ConvertToNewInterval(OneMin)
	if !errors.Is(err, errNilKline) {
		t.Errorf("received '%v' expected '%v'", err, errNilKline)
	}

	_, err = (&Item{}).ConvertToNewInterval(OneMin)
	if !errors.Is(err, ErrInvalidInterval) {
		t.Errorf("received '%v' expected '%v'", err, ErrInvalidInterval)
	}

	old := &Item{
		Exchange: "lol",
		Pair:     currency.NewPair(currency.BTC, currency.USDT),
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
	if !errors.Is(err, ErrInvalidInterval) {
		t.Errorf("received '%v' expected '%v'", err, ErrInvalidInterval)
	}
	_, err = old.ConvertToNewInterval(OneMin)
	if !errors.Is(err, ErrCanOnlyUpscaleCandles) {
		t.Errorf("received '%v' expected '%v'", err, ErrCanOnlyUpscaleCandles)
	}
	old.Interval = ThreeDay
	_, err = old.ConvertToNewInterval(OneWeek)
	if !errors.Is(err, ErrWholeNumberScaling) {
		t.Errorf("received '%v' expected '%v'", err, ErrWholeNumberScaling)
	}

	old.Interval = OneDay
	newInterval := ThreeDay
	newCandle, err := old.ConvertToNewInterval(newInterval)
	if !errors.Is(err, nil) {
		t.Fatalf("received '%v' expected '%v'", err, nil)
	}
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
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if len(newCandle.Candles) != 1 {
		t.Error("expected one candle")
	}

	_, err = old.ConvertToNewInterval(OneMonth)
	if !errors.Is(err, ErrInsufficientCandleData) {
		t.Errorf("received '%v' expected '%v'", err, ErrInsufficientCandleData)
	}

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
	if !errors.Is(err, errCandleDataNotPadded) {
		t.Errorf("received '%v' expected '%v'", err, errCandleDataNotPadded)
	}

	err = old.addPadding(tn, tn.AddDate(0, 0, 9), false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	newCandle, err = old.ConvertToNewInterval(newInterval)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	if len(newCandle.Candles) != 3 {
		t.Errorf("received '%v' expected '%v'", len(newCandle.Candles), 3)
	}
}

func TestAddPadding(t *testing.T) {
	t.Parallel()

	tn := time.Now().Truncate(time.Duration(OneDay))

	var k *Item
	err := k.addPadding(tn, tn.AddDate(0, 0, 5), false)
	if !errors.Is(err, errNilKline) {
		t.Fatalf("received '%v' expected '%v'", err, errNilKline)
	}

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
	if !errors.Is(err, ErrInvalidInterval) {
		t.Fatalf("received '%v' expected '%v'", err, ErrInvalidInterval)
	}

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
	if !errors.Is(err, errCannotEstablishTimeWindow) {
		t.Fatalf("received '%v' expected '%v'", err, errCannotEstablishTimeWindow)
	}

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
		}}

	err = k.addPadding(tn, tn.AddDate(0, 0, 3), false)
	if !errors.Is(err, errCandleOpenTimeIsNotUTCAligned) {
		t.Fatalf("received '%v' expected '%v'", err, errCandleOpenTimeIsNotUTCAligned)
	}

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
		}}

	err = k.addPadding(tn, tn.AddDate(0, 0, 3), false)
	if !errors.Is(err, nil) {
		t.Fatalf("received '%v' expected '%v'", err, nil)
	}

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
	if !errors.Is(err, nil) {
		t.Fatalf("received '%v' expected '%v'", err, nil)
	}

	if len(k.Candles) != 6 {
		t.Fatalf("received '%v' expected '%v'", len(k.Candles), 6)
	}

	// No candles test when there is zero activity for that period
	k.Candles = nil

	err = k.addPadding(tn, tn.AddDate(0, 0, 6), false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

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
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if price != 1337 {
		t.Errorf("received '%v' expected '%v'", price, 1337)
	}
	_, err = k.GetClosePriceAtTime(tt.Add(time.Minute))
	if !errors.Is(err, ErrNotFoundAtTime) {
		t.Errorf("received '%v' expected '%v'", err, ErrNotFoundAtTime)
	}
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
	if !errors.Is(err, ErrInvalidInterval) {
		t.Errorf("received '%v' expected '%v'", err, ErrInvalidInterval)
	}

	_, err = exchangeIntervals.Construct(OneMin)
	if !errors.Is(err, ErrCannotConstructInterval) {
		t.Errorf("received '%v' expected '%v'", err, ErrCannotConstructInterval)
	}

	request, err := exchangeIntervals.Construct(OneWeek)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	if request != OneWeek {
		t.Errorf("received '%v' expected '%v'", request, OneWeek)
	}

	exchangeIntervals = DeployExchangeIntervals(IntervalCapacity{Interval: OneWeek}, IntervalCapacity{Interval: OneDay})

	request, err = exchangeIntervals.Construct(OneMonth)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	if request != OneDay {
		t.Errorf("received '%v' expected '%v'", request, OneDay)
	}
}

func TestSetHasDataFromCandles(t *testing.T) {
	t.Parallel()
	ohc := getOneHour()
	localEnd := ohc[len(ohc)-1].Time.Add(OneHour.Duration())
	i, err := CalculateCandleDateRanges(ohc[0].Time, localEnd, OneHour, 100000)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	err = i.SetHasDataFromCandles(ohc)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
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
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	err = i.SetHasDataFromCandles(k.Candles)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
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
	if !errors.Is(err, errExchangeCapabilitiesEnabledIsNil) {
		t.Errorf("received '%v' expected '%v'", err, errExchangeCapabilitiesEnabledIsNil)
	}

	e = &ExchangeCapabilitiesEnabled{}
	e.Intervals = ExchangeIntervals{}
	_, err = e.GetIntervalResultLimit(OneDay)
	if !errors.Is(err, errIntervalNotSupported) {
		t.Errorf("received '%v' expected '%v'", err, errIntervalNotSupported)
	}

	e.Intervals = ExchangeIntervals{
		supported: map[Interval]int64{
			OneDay: 100000,
			OneMin: 0,
		},
	}

	_, err = e.GetIntervalResultLimit(OneMin)
	if !errors.Is(err, errCannotFetchIntervalLimit) {
		t.Errorf("received '%v' expected '%v'", err, errCannotFetchIntervalLimit)
	}

	limit, err := e.GetIntervalResultLimit(OneDay)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	if limit != 100000 {
		t.Errorf("received '%v' expected '%v'", limit, 100000)
	}

	e.GlobalResultLimit = 1337
	limit, err = e.GetIntervalResultLimit(OneMin)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	if limit != 1337 {
		t.Errorf("received '%v' expected '%v'", limit, 1337)
	}
}
