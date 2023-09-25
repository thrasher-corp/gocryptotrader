package order

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/validate"
)

var errValidationCheckFailed = errors.New("validation check failed")

func TestSubmit_Validate(t *testing.T) {
	t.Parallel()
	testPair := currency.NewPair(currency.BTC, currency.LTC)
	tester := []struct {
		ExpectedErr error
		Submit      *Submit
		ValidOpts   validate.Checker
	}{
		{
			ExpectedErr: ErrSubmissionIsNil,
			Submit:      nil,
		}, // nil struct
		{
			ExpectedErr: errExchangeNameUnset,
			Submit:      &Submit{},
		}, // empty exchange
		{
			ExpectedErr: ErrPairIsEmpty,
			Submit:      &Submit{Exchange: "test"},
		}, // empty pair
		{

			ExpectedErr: ErrAssetNotSet,
			Submit:      &Submit{Exchange: "test", Pair: testPair},
		}, // valid pair but invalid asset
		{
			ExpectedErr: asset.ErrNotSupported,
			Submit: &Submit{
				Exchange:  "test",
				Pair:      testPair,
				AssetType: 255,
			},
		}, // valid pair but invalid asset
		{
			ExpectedErr: ErrSideIsInvalid,
			Submit: &Submit{
				Exchange:  "test",
				Pair:      testPair,
				AssetType: asset.Spot,
			},
		}, // valid pair but invalid order side
		{
			ExpectedErr: errTimeInForceConflict,
			Submit: &Submit{
				Exchange:          "test",
				Pair:              testPair,
				AssetType:         asset.Spot,
				Side:              Ask,
				Type:              Market,
				ImmediateOrCancel: true,
				FillOrKill:        true,
			},
		},
		{
			ExpectedErr: ErrTypeIsInvalid,
			Submit: &Submit{
				Exchange:  "test",
				Pair:      testPair,
				Side:      Buy,
				AssetType: asset.Spot,
			},
		}, // valid pair and order side but invalid order type
		{
			ExpectedErr: ErrTypeIsInvalid,
			Submit: &Submit{
				Exchange:  "test",
				Pair:      testPair,
				Side:      Sell,
				AssetType: asset.Spot,
			},
		}, // valid pair and order side but invalid order type
		{
			ExpectedErr: ErrTypeIsInvalid,
			Submit: &Submit{
				Exchange:  "test",
				Pair:      testPair,
				Side:      Bid,
				AssetType: asset.Spot,
			},
		}, // valid pair and order side but invalid order type
		{
			ExpectedErr: ErrTypeIsInvalid,
			Submit: &Submit{
				Exchange:  "test",
				Pair:      testPair,
				Side:      Ask,
				AssetType: asset.Spot,
			},
		}, // valid pair and order side but invalid order type
		{
			ExpectedErr: ErrAmountIsInvalid,
			Submit: &Submit{
				Exchange:  "test",
				Pair:      testPair,
				Side:      Ask,
				Type:      Market,
				AssetType: asset.Spot,
			},
		}, // valid pair, order side, type but invalid amount
		{
			ExpectedErr: ErrAmountIsInvalid,
			Submit: &Submit{
				Exchange:  "test",
				Pair:      testPair,
				Side:      Ask,
				Type:      Market,
				AssetType: asset.Spot,
				Amount:    -1,
			},
		}, // valid pair, order side, type but invalid amount
		{
			ExpectedErr: ErrAmountIsInvalid,
			Submit: &Submit{
				Exchange:    "test",
				Pair:        testPair,
				Side:        Ask,
				Type:        Market,
				AssetType:   asset.Spot,
				QuoteAmount: -1,
			},
		}, // valid pair, order side, type but invalid amount
		{
			ExpectedErr: ErrPriceMustBeSetIfLimitOrder,
			Submit: &Submit{
				Exchange:  "test",
				Pair:      testPair,
				Side:      Ask,
				Type:      Limit,
				Amount:    1,
				AssetType: asset.Spot,
			},
		}, // valid pair, order side, type, amount but invalid price
		{
			ExpectedErr: errValidationCheckFailed,
			Submit: &Submit{
				Exchange:  "test",
				Pair:      testPair,
				Side:      Ask,
				Type:      Limit,
				Amount:    1,
				Price:     1000,
				AssetType: asset.Spot,
			},
			ValidOpts: validate.Check(func() error { return errValidationCheckFailed }),
		}, // custom validation error check
		{
			ExpectedErr: nil,
			Submit: &Submit{
				Exchange:  "test",
				Pair:      testPair,
				Side:      Ask,
				Type:      Limit,
				Amount:    1,
				Price:     1000,
				AssetType: asset.Spot,
			},
			ValidOpts: validate.Check(func() error { return nil }),
		}, // valid order!
	}

	for x := range tester {
		err := tester[x].Submit.Validate(tester[x].ValidOpts)
		if !errors.Is(err, tester[x].ExpectedErr) {
			t.Fatalf("Unexpected result. %d Got: %v, want: %v", x+1, err, tester[x].ExpectedErr)
		}
	}
}

func TestSubmit_DeriveSubmitResponse(t *testing.T) {
	t.Parallel()
	var s *Submit
	_, err := s.DeriveSubmitResponse("")
	if !errors.Is(err, errOrderSubmitIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errOrderSubmitIsNil)
	}

	s = &Submit{}
	_, err = s.DeriveSubmitResponse("")
	if !errors.Is(err, ErrOrderIDNotSet) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderIDNotSet)
	}

	resp, err := s.DeriveSubmitResponse("1337")
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if resp.OrderID != "1337" {
		t.Fatal("unexpected value")
	}

	if resp.Status != New {
		t.Fatal("unexpected value")
	}

	if resp.Date.IsZero() {
		t.Fatal("unexpected value")
	}

	if resp.LastUpdated.IsZero() {
		t.Fatal("unexpected value")
	}
}

func TestSubmitResponse_DeriveDetail(t *testing.T) {
	t.Parallel()
	var s *SubmitResponse
	_, err := s.DeriveDetail(uuid.Nil)
	if !errors.Is(err, errOrderSubmitResponseIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errOrderSubmitResponseIsNil)
	}

	id, err := uuid.NewV4()
	if err != nil {
		t.Fatal(err)
	}

	s = &SubmitResponse{}
	deets, err := s.DeriveDetail(id)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if deets.InternalOrderID != id {
		t.Fatal("unexpected value")
	}
}

func TestOrderSides(t *testing.T) {
	t.Parallel()

	var os = Buy
	if os.String() != "BUY" {
		t.Errorf("unexpected string %s", os.String())
	}

	if os.Lower() != "buy" {
		t.Errorf("unexpected string %s", os.Lower())
	}

	if os.Title() != "Buy" {
		t.Errorf("unexpected string %s", os.Title())
	}
}

func TestTitle(t *testing.T) {
	t.Parallel()
	orderType := Limit
	if orderType.Title() != "Limit" {
		t.Errorf("received '%v' expected 'Limit'", orderType.Title())
	}
}

func TestOrderTypes(t *testing.T) {
	t.Parallel()

	var orderType Type
	if orderType.String() != "UNKNOWN" {
		t.Errorf("unexpected string %s", orderType.String())
	}

	if orderType.Lower() != "unknown" {
		t.Errorf("unexpected string %s", orderType.Lower())
	}

	if orderType.Title() != "Unknown" {
		t.Errorf("unexpected string %s", orderType.Title())
	}
}

func TestInferCostsAndTimes(t *testing.T) {
	t.Parallel()

	var detail Detail
	detail.InferCostsAndTimes()
	if detail.Amount != detail.ExecutedAmount+detail.RemainingAmount {
		t.Errorf(
			"Order detail amounts not equals. Expected 0, received %f",
			detail.Amount-(detail.ExecutedAmount+detail.RemainingAmount),
		)
	}

	detail.CloseTime = time.Now()
	detail.InferCostsAndTimes()
	if detail.LastUpdated != detail.CloseTime {
		t.Errorf(
			"Order last updated not equals close time. Expected %s, received %s",
			detail.CloseTime,
			detail.LastUpdated,
		)
	}

	detail.Amount = 1
	detail.ExecutedAmount = 1
	detail.InferCostsAndTimes()
	if detail.AverageExecutedPrice != 0 {
		t.Errorf(
			"Unexpected AverageExecutedPrice. Expected 0, received %f",
			detail.AverageExecutedPrice,
		)
	}

	detail.Amount = 1
	detail.ExecutedAmount = 1
	detail.InferCostsAndTimes()
	if detail.Cost != 0 {
		t.Errorf(
			"Unexpected Cost. Expected 0, received %f",
			detail.Cost,
		)
	}
	detail.ExecutedAmount = 0

	detail.Amount = 1
	detail.RemainingAmount = 1
	detail.InferCostsAndTimes()
	if detail.Amount != detail.ExecutedAmount+detail.RemainingAmount {
		t.Errorf(
			"Order detail amounts not equals. Expected 0, received %f",
			detail.Amount-(detail.ExecutedAmount+detail.RemainingAmount),
		)
	}
	detail.RemainingAmount = 0

	detail.Amount = 1
	detail.ExecutedAmount = 1
	detail.Price = 2
	detail.InferCostsAndTimes()
	if detail.AverageExecutedPrice != 2 {
		t.Errorf(
			"Unexpected AverageExecutedPrice. Expected 2, received %f",
			detail.AverageExecutedPrice,
		)
	}

	detail = Detail{Amount: 1, ExecutedAmount: 2, Cost: 3, Price: 0}
	detail.InferCostsAndTimes()
	if detail.AverageExecutedPrice != 1.5 {
		t.Errorf(
			"Unexpected AverageExecutedPrice. Expected 1.5, received %f",
			detail.AverageExecutedPrice,
		)
	}

	detail = Detail{Amount: 1, ExecutedAmount: 2, AverageExecutedPrice: 3}
	detail.InferCostsAndTimes()
	if detail.Cost != 6 {
		t.Errorf(
			"Unexpected Cost. Expected 6, received %f",
			detail.Cost,
		)
	}
}

func TestFilterOrdersByType(t *testing.T) {
	t.Parallel()

	var orders = []Detail{
		{
			Type: ImmediateOrCancel,
		},
		{
			Type: Limit,
		},
		{}, // Unpopulated fields are preserved for API differences
	}

	FilterOrdersByType(&orders, AnyType)
	if len(orders) != 3 {
		t.Errorf("Orders failed to be filtered. Expected %v, received %v", 2, len(orders))
	}

	FilterOrdersByType(&orders, Limit)
	if len(orders) != 2 {
		t.Errorf("Orders failed to be filtered. Expected %v, received %v", 1, len(orders))
	}

	FilterOrdersByType(&orders, Stop)
	if len(orders) != 1 {
		t.Errorf("Orders failed to be filtered. Expected %v, received %v", 0, len(orders))
	}
}

var filterOrdersByTypeBenchmark = &[]Detail{
	{Type: Limit},
	{Type: Limit},
	{Type: Limit},
	{Type: Limit},
	{Type: Limit},
	{Type: Limit},
	{Type: Limit},
	{Type: Limit},
	{Type: Limit},
	{Type: Limit},
}

// BenchmarkFilterOrdersByType benchmark
//
// 392455	      3226 ns/op	   15840 B/op	       5 allocs/op // PREV
// 9486490	       109.5 ns/op	       0 B/op	       0 allocs/op // CURRENT
func BenchmarkFilterOrdersByType(b *testing.B) {
	for x := 0; x < b.N; x++ {
		FilterOrdersByType(filterOrdersByTypeBenchmark, Limit)
	}
}

func TestFilterOrdersBySide(t *testing.T) {
	t.Parallel()

	var orders = []Detail{
		{
			Side: Buy,
		},
		{
			Side: Sell,
		},
		{}, // Unpopulated fields are preserved for API differences
	}

	FilterOrdersBySide(&orders, AnySide)
	if len(orders) != 3 {
		t.Errorf("Orders failed to be filtered. Expected %v, received %v", 3, len(orders))
	}

	FilterOrdersBySide(&orders, Buy)
	if len(orders) != 2 {
		t.Errorf("Orders failed to be filtered. Expected %v, received %v", 1, len(orders))
	}

	FilterOrdersBySide(&orders, Sell)
	if len(orders) != 1 {
		t.Errorf("Orders failed to be filtered. Expected %v, received %v", 0, len(orders))
	}
}

var filterOrdersBySideBenchmark = &[]Detail{
	{Side: Ask},
	{Side: Ask},
	{Side: Ask},
	{Side: Ask},
	{Side: Ask},
	{Side: Ask},
	{Side: Ask},
	{Side: Ask},
	{Side: Ask},
	{Side: Ask},
}

// BenchmarkFilterOrdersBySide benchmark
//
// 372594	      3049 ns/op	   15840 B/op	       5 allocs/op // PREV
// 7412187	       148.8 ns/op	       0 B/op	       0 allocs/op // CURRENT
func BenchmarkFilterOrdersBySide(b *testing.B) {
	for x := 0; x < b.N; x++ {
		FilterOrdersBySide(filterOrdersBySideBenchmark, Ask)
	}
}

func TestFilterOrdersByTimeRange(t *testing.T) {
	t.Parallel()

	var orders = []Detail{
		{
			Date: time.Unix(100, 0),
		},
		{
			Date: time.Unix(110, 0),
		},
		{
			Date: time.Unix(111, 0),
		},
	}

	err := FilterOrdersByTimeRange(&orders, time.Unix(0, 0), time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	if len(orders) != 3 {
		t.Errorf("Orders failed to be filtered. Expected %v, received %v", 3, len(orders))
	}

	err = FilterOrdersByTimeRange(&orders, time.Unix(100, 0), time.Unix(111, 0))
	if err != nil {
		t.Fatal(err)
	}
	if len(orders) != 3 {
		t.Errorf("Orders failed to be filtered. Expected %v, received %v", 3, len(orders))
	}

	err = FilterOrdersByTimeRange(&orders, time.Unix(101, 0), time.Unix(111, 0))
	if err != nil {
		t.Fatal(err)
	}
	if len(orders) != 2 {
		t.Errorf("Orders failed to be filtered. Expected %v, received %v", 2, len(orders))
	}

	err = FilterOrdersByTimeRange(&orders, time.Unix(200, 0), time.Unix(300, 0))
	if err != nil {
		t.Fatal(err)
	}
	if len(orders) != 0 {
		t.Errorf("Orders failed to be filtered. Expected %v, received %v", 0, len(orders))
	}
	orders = append(orders, Detail{})
	// test for event no timestamp is set on an order, best to include it
	err = FilterOrdersByTimeRange(&orders, time.Unix(200, 0), time.Unix(300, 0))
	if err != nil {
		t.Fatal(err)
	}
	if len(orders) != 1 {
		t.Errorf("Orders failed to be filtered. Expected %v, received %v", 1, len(orders))
	}

	err = FilterOrdersByTimeRange(&orders, time.Unix(300, 0), time.Unix(50, 0))
	if !errors.Is(err, common.ErrStartAfterEnd) {
		t.Fatalf("received: '%v' but expected: '%v'", err, common.ErrStartAfterEnd)
	}
}

var filterOrdersByTimeRangeBenchmark = &[]Detail{
	{Date: time.Unix(100, 0)},
	{Date: time.Unix(100, 0)},
	{Date: time.Unix(100, 0)},
	{Date: time.Unix(100, 0)},
	{Date: time.Unix(100, 0)},
	{Date: time.Unix(100, 0)},
	{Date: time.Unix(100, 0)},
	{Date: time.Unix(100, 0)},
	{Date: time.Unix(100, 0)},
	{Date: time.Unix(100, 0)},
}

// BenchmarkFilterOrdersByTimeRange benchmark
//
// 390822	      3335 ns/op	   15840 B/op	       5 allocs/op // PREV
// 6201034	       172.1 ns/op	       0 B/op	       0 allocs/op // CURRENT
func BenchmarkFilterOrdersByTimeRange(b *testing.B) {
	for x := 0; x < b.N; x++ {
		err := FilterOrdersByTimeRange(filterOrdersByTimeRangeBenchmark, time.Unix(50, 0), time.Unix(150, 0))
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestFilterOrdersByPairs(t *testing.T) {
	t.Parallel()

	var orders = []Detail{
		{
			Pair: currency.NewPair(currency.BTC, currency.USD),
		},
		{
			Pair: currency.NewPair(currency.LTC, currency.EUR),
		},
		{
			Pair: currency.NewPair(currency.DOGE, currency.RUB),
		},
		{}, // Unpopulated fields are preserved for API differences
	}

	currencies := []currency.Pair{currency.NewPair(currency.BTC, currency.USD),
		currency.NewPair(currency.LTC, currency.EUR),
		currency.NewPair(currency.DOGE, currency.RUB)}
	FilterOrdersByPairs(&orders, currencies)
	if len(orders) != 4 {
		t.Errorf("Orders failed to be filtered. Expected %v, received %v", 3, len(orders))
	}

	currencies = []currency.Pair{currency.NewPair(currency.BTC, currency.USD),
		currency.NewPair(currency.LTC, currency.EUR)}
	FilterOrdersByPairs(&orders, currencies)
	if len(orders) != 3 {
		t.Errorf("Orders failed to be filtered. Expected %v, received %v", 2, len(orders))
	}

	currencies = []currency.Pair{currency.NewPair(currency.BTC, currency.USD)}
	FilterOrdersByPairs(&orders, currencies)
	if len(orders) != 2 {
		t.Errorf("Orders failed to be filtered. Expected %v, received %v", 1, len(orders))
	}

	currencies = []currency.Pair{currency.NewPair(currency.USD, currency.BTC)}
	FilterOrdersByPairs(&orders, currencies)
	if len(orders) != 2 {
		t.Errorf("Reverse Orders failed to be filtered. Expected %v, received %v", 1, len(orders))
	}

	currencies = []currency.Pair{}
	FilterOrdersByPairs(&orders, currencies)
	if len(orders) != 2 {
		t.Errorf("Orders failed to be filtered. Expected %v, received %v", 1, len(orders))
	}
	currencies = append(currencies, currency.EMPTYPAIR)
	FilterOrdersByPairs(&orders, currencies)
	if len(orders) != 2 {
		t.Errorf("Orders failed to be filtered. Expected %v, received %v", 1, len(orders))
	}
}

var filterOrdersByPairsBenchmark = &[]Detail{
	{Pair: currency.NewPair(currency.BTC, currency.USD)},
	{Pair: currency.NewPair(currency.BTC, currency.USD)},
	{Pair: currency.NewPair(currency.BTC, currency.USD)},
	{Pair: currency.NewPair(currency.BTC, currency.USD)},
	{Pair: currency.NewPair(currency.BTC, currency.USD)},
	{Pair: currency.NewPair(currency.BTC, currency.USD)},
	{Pair: currency.NewPair(currency.BTC, currency.USD)},
	{Pair: currency.NewPair(currency.BTC, currency.USD)},
	{Pair: currency.NewPair(currency.BTC, currency.USD)},
	{Pair: currency.NewPair(currency.BTC, currency.USD)},
}

// BenchmarkFilterOrdersByPairs benchmark
//
// 400032	      2977 ns/op	   15840 B/op	       5 allocs/op // PREV
// 6977242	       172.8 ns/op	       0 B/op	       0 allocs/op // CURRENT
func BenchmarkFilterOrdersByPairs(b *testing.B) {
	pairs := []currency.Pair{currency.NewPair(currency.BTC, currency.USD)}
	for x := 0; x < b.N; x++ {
		FilterOrdersByPairs(filterOrdersByPairsBenchmark, pairs)
	}
}

func TestSortOrdersByPrice(t *testing.T) {
	t.Parallel()

	orders := []Detail{
		{
			Price: 100,
		}, {
			Price: 0,
		}, {
			Price: 50,
		},
	}

	SortOrdersByPrice(&orders, false)
	if orders[0].Price != 0 {
		t.Errorf("Expected: '%v', received: '%v'", 0, orders[0].Price)
	}

	SortOrdersByPrice(&orders, true)
	if orders[0].Price != 100 {
		t.Errorf("Expected: '%v', received: '%v'", 100, orders[0].Price)
	}
}

func TestSortOrdersByDate(t *testing.T) {
	t.Parallel()

	orders := []Detail{
		{
			Date: time.Unix(0, 0),
		}, {
			Date: time.Unix(1, 0),
		}, {
			Date: time.Unix(2, 0),
		},
	}

	SortOrdersByDate(&orders, false)
	if orders[0].Date.Unix() != time.Unix(0, 0).Unix() {
		t.Errorf("Expected: '%v', received: '%v'",
			time.Unix(0, 0).Unix(),
			orders[0].Date.Unix())
	}

	SortOrdersByDate(&orders, true)
	if orders[0].Date.Unix() != time.Unix(2, 0).Unix() {
		t.Errorf("Expected: '%v', received: '%v'",
			time.Unix(2, 0).Unix(),
			orders[0].Date.Unix())
	}
}

func TestSortOrdersByCurrency(t *testing.T) {
	t.Parallel()

	orders := []Detail{
		{
			Pair: currency.NewPairWithDelimiter(currency.BTC.String(),
				currency.USD.String(),
				"-"),
		}, {
			Pair: currency.NewPairWithDelimiter(currency.DOGE.String(),
				currency.USD.String(),
				"-"),
		}, {
			Pair: currency.NewPairWithDelimiter(currency.BTC.String(),
				currency.RUB.String(),
				"-"),
		}, {
			Pair: currency.NewPairWithDelimiter(currency.LTC.String(),
				currency.EUR.String(),
				"-"),
		}, {
			Pair: currency.NewPairWithDelimiter(currency.LTC.String(),
				currency.AUD.String(),
				"-"),
		},
	}

	SortOrdersByCurrency(&orders, false)
	if orders[0].Pair.String() != currency.BTC.String()+"-"+currency.RUB.String() {
		t.Errorf("Expected: '%v', received: '%v'",
			currency.BTC.String()+"-"+currency.RUB.String(),
			orders[0].Pair.String())
	}

	SortOrdersByCurrency(&orders, true)
	if orders[0].Pair.String() != currency.LTC.String()+"-"+currency.EUR.String() {
		t.Errorf("Expected: '%v', received: '%v'",
			currency.LTC.String()+"-"+currency.EUR.String(),
			orders[0].Pair.String())
	}
}

func TestSortOrdersByOrderSide(t *testing.T) {
	t.Parallel()

	orders := []Detail{
		{
			Side: Buy,
		}, {
			Side: Sell,
		}, {
			Side: Sell,
		}, {
			Side: Buy,
		},
	}

	SortOrdersBySide(&orders, false)
	if !strings.EqualFold(orders[0].Side.String(), Buy.String()) {
		t.Errorf("Expected: '%v', received: '%v'",
			Buy,
			orders[0].Side)
	}

	SortOrdersBySide(&orders, true)
	if !strings.EqualFold(orders[0].Side.String(), Sell.String()) {
		t.Errorf("Expected: '%v', received: '%v'",
			Sell,
			orders[0].Side)
	}
}

func TestSortOrdersByOrderType(t *testing.T) {
	t.Parallel()

	orders := []Detail{
		{
			Type: Market,
		}, {
			Type: Limit,
		}, {
			Type: ImmediateOrCancel,
		}, {
			Type: TrailingStop,
		},
	}

	SortOrdersByType(&orders, false)
	if !strings.EqualFold(orders[0].Type.String(), ImmediateOrCancel.String()) {
		t.Errorf("Expected: '%v', received: '%v'",
			ImmediateOrCancel,
			orders[0].Type)
	}

	SortOrdersByType(&orders, true)
	if !strings.EqualFold(orders[0].Type.String(), TrailingStop.String()) {
		t.Errorf("Expected: '%v', received: '%v'",
			TrailingStop,
			orders[0].Type)
	}
}

func TestStringToOrderSide(t *testing.T) {
	cases := []struct {
		in  string
		out Side
		err error
	}{
		{"buy", Buy, nil},
		{"BUY", Buy, nil},
		{"bUy", Buy, nil},
		{"sell", Sell, nil},
		{"SELL", Sell, nil},
		{"sElL", Sell, nil},
		{"bid", Bid, nil},
		{"BID", Bid, nil},
		{"bId", Bid, nil},
		{"ask", Ask, nil},
		{"ASK", Ask, nil},
		{"aSk", Ask, nil},
		{"lOnG", Long, nil},
		{"ShoRt", Short, nil},
		{"any", AnySide, nil},
		{"ANY", AnySide, nil},
		{"aNy", AnySide, nil},
		{"woahMan", UnknownSide, errUnrecognisedOrderSide},
	}
	for i := range cases {
		testData := &cases[i]
		t.Run(testData.in, func(t *testing.T) {
			out, err := StringToOrderSide(testData.in)
			if !errors.Is(err, testData.err) {
				t.Fatalf("received: '%v' but expected: '%v'", err, testData.err)
			}
			if out != testData.out {
				t.Errorf("Unexpected output %v. Expected %v", out, testData.out)
			}
		})
	}
}

var sideBenchmark Side

// 9756914	       126.7 ns/op	       0 B/op	       0 allocs/op // PREV
// 25200660	        57.63 ns/op	       3 B/op	       1 allocs/op // CURRENT
func BenchmarkStringToOrderSide(b *testing.B) {
	for x := 0; x < b.N; x++ {
		sideBenchmark, _ = StringToOrderSide("any")
	}
}

func TestStringToOrderType(t *testing.T) {
	cases := []struct {
		in  string
		out Type
		err error
	}{
		{"limit", Limit, nil},
		{"LIMIT", Limit, nil},
		{"lImIt", Limit, nil},
		{"market", Market, nil},
		{"MARKET", Market, nil},
		{"mArKeT", Market, nil},
		{"immediate_or_cancel", ImmediateOrCancel, nil},
		{"IMMEDIATE_OR_CANCEL", ImmediateOrCancel, nil},
		{"iMmEdIaTe_Or_CaNcEl", ImmediateOrCancel, nil},
		{"iMmEdIaTe Or CaNcEl", ImmediateOrCancel, nil},
		{"stop", Stop, nil},
		{"STOP", Stop, nil},
		{"sToP", Stop, nil},
		{"sToP LiMit", StopLimit, nil},
		{"ExchangE sToP Limit", StopLimit, nil},
		{"trailing_stop", TrailingStop, nil},
		{"TRAILING_STOP", TrailingStop, nil},
		{"tRaIlInG_sToP", TrailingStop, nil},
		{"tRaIlInG sToP", TrailingStop, nil},
		{"fOk", FillOrKill, nil},
		{"exchange fOk", FillOrKill, nil},
		{"ios", IOS, nil},
		{"post_ONly", PostOnly, nil},
		{"any", AnyType, nil},
		{"ANY", AnyType, nil},
		{"aNy", AnyType, nil},
		{"trigger", Trigger, nil},
		{"TRIGGER", Trigger, nil},
		{"tRiGgEr", Trigger, nil},
		{"conDitiOnal", ConditionalStop, nil},
		{"oCo", OCO, nil},
		{"woahMan", UnknownType, errUnrecognisedOrderType},
	}
	for i := range cases {
		testData := &cases[i]
		t.Run(testData.in, func(t *testing.T) {
			out, err := StringToOrderType(testData.in)
			if !errors.Is(err, testData.err) {
				t.Fatalf("received: '%v' but expected: '%v'", err, testData.err)
			}
			if out != testData.out {
				t.Errorf("Unexpected output %v. Expected %v", out, testData.out)
			}
		})
	}
}

var typeBenchmark Type

// 5703705	       299.9 ns/op	       0 B/op	       0 allocs/op // PREV
// 16353608	        81.23 ns/op	       8 B/op	       1 allocs/op // CURRENT
func BenchmarkStringToOrderType(b *testing.B) {
	for x := 0; x < b.N; x++ {
		typeBenchmark, _ = StringToOrderType("trigger")
	}
}

var stringsToOrderStatus = []struct {
	in  string
	out Status
	err error
}{
	{"any", AnyStatus, nil},
	{"ANY", AnyStatus, nil},
	{"aNy", AnyStatus, nil},
	{"new", New, nil},
	{"NEW", New, nil},
	{"nEw", New, nil},
	{"active", Active, nil},
	{"ACTIVE", Active, nil},
	{"aCtIvE", Active, nil},
	{"partially_filled", PartiallyFilled, nil},
	{"PARTIALLY_FILLED", PartiallyFilled, nil},
	{"pArTiAlLy_FiLlEd", PartiallyFilled, nil},
	{"filled", Filled, nil},
	{"FILLED", Filled, nil},
	{"fIlLeD", Filled, nil},
	{"cancelled", Cancelled, nil},
	{"CANCELlED", Cancelled, nil},
	{"cAnCellEd", Cancelled, nil},
	{"pending_cancel", PendingCancel, nil},
	{"PENDING_CANCEL", PendingCancel, nil},
	{"pENdInG_cAnCeL", PendingCancel, nil},
	{"rejected", Rejected, nil},
	{"REJECTED", Rejected, nil},
	{"rEjEcTeD", Rejected, nil},
	{"expired", Expired, nil},
	{"EXPIRED", Expired, nil},
	{"eXpIrEd", Expired, nil},
	{"hidden", Hidden, nil},
	{"HIDDEN", Hidden, nil},
	{"hIdDeN", Hidden, nil},
	{"market_unavailable", MarketUnavailable, nil},
	{"MARKET_UNAVAILABLE", MarketUnavailable, nil},
	{"mArKeT_uNaVaIlAbLe", MarketUnavailable, nil},
	{"insufficient_balance", InsufficientBalance, nil},
	{"INSUFFICIENT_BALANCE", InsufficientBalance, nil},
	{"iNsUfFiCiEnT_bAlAnCe", InsufficientBalance, nil},
	{"PARTIALLY_CANCELLEd", PartiallyCancelled, nil},
	{"partially canceLLed", PartiallyCancelled, nil},
	{"opeN", Open, nil},
	{"cLosEd", Closed, nil},
	{"cancellinG", Cancelling, nil},
	{"woahMan", UnknownStatus, errUnrecognisedOrderStatus},
	{"PLAcED", New, nil},
	{"ACCePTED", New, nil},
	{"FAILeD", Rejected, nil},
}

func TestStringToOrderStatus(t *testing.T) {
	for i := range stringsToOrderStatus {
		testData := &stringsToOrderStatus[i]
		t.Run(testData.in, func(t *testing.T) {
			out, err := StringToOrderStatus(testData.in)
			if !errors.Is(err, testData.err) {
				t.Fatalf("received: '%v' but expected: '%v'", err, testData.err)
			}
			if out != testData.out {
				t.Errorf("Unexpected output %v. Expected %v", out, testData.out)
			}
		})
	}
}

var statusBenchmark Status

// 3569052	       351.8 ns/op	       0 B/op	       0 allocs/op // PREV
// 11126791	       101.9 ns/op	      24 B/op	       1 allocs/op // CURRENT
func BenchmarkStringToOrderStatus(b *testing.B) {
	for x := 0; x < b.N; x++ {
		statusBenchmark, _ = StringToOrderStatus("market_unavailable")
	}
}

func TestUpdateOrderFromModifyResponse(t *testing.T) {
	od := Detail{OrderID: "1"}
	updated := time.Now()

	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	om := ModifyResponse{
		ImmediateOrCancel: true,
		PostOnly:          true,
		Price:             1,
		Amount:            1,
		TriggerPrice:      1,
		RemainingAmount:   1,
		Exchange:          "1",
		Type:              1,
		Side:              1,
		Status:            1,
		AssetType:         1,
		LastUpdated:       updated,
		Pair:              pair,
	}

	od.UpdateOrderFromModifyResponse(&om)

	if !od.ImmediateOrCancel {
		t.Error("Failed to update")
	}
	if !od.PostOnly {
		t.Error("Failed to update")
	}
	if od.Price != 1 {
		t.Error("Failed to update")
	}
	if od.Amount != 1 {
		t.Error("Failed to update")
	}
	if od.TriggerPrice != 1 {
		t.Error("Failed to update")
	}
	if od.RemainingAmount != 1 {
		t.Error("Failed to update")
	}
	if od.Exchange != "" {
		t.Error("Should not be able to update exchange via modify")
	}
	if od.OrderID != "1" {
		t.Error("Failed to update")
	}
	if od.Type != 1 {
		t.Error("Failed to update")
	}
	if od.Side != 1 {
		t.Error("Failed to update")
	}
	if od.Status != 1 {
		t.Error("Failed to update")
	}
	if od.AssetType != 1 {
		t.Error("Failed to update")
	}
	if od.LastUpdated != updated {
		t.Error("Failed to update")
	}
	if od.Pair.String() != "BTCUSD" {
		t.Error("Failed to update")
	}
	if od.Trades != nil {
		t.Error("Failed to update")
	}
}

func TestUpdateOrderFromDetail(t *testing.T) {
	var leet = "1337"

	updated := time.Now()

	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	id, err := uuid.NewV4()
	if err != nil {
		t.Fatal(err)
	}

	var od *Detail
	err = od.UpdateOrderFromDetail(nil)
	if !errors.Is(err, ErrOrderDetailIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderDetailIsNil)
	}

	om := &Detail{
		ImmediateOrCancel: true,
		HiddenOrder:       true,
		FillOrKill:        true,
		PostOnly:          true,
		Leverage:          1,
		Price:             1,
		Amount:            1,
		LimitPriceUpper:   1,
		LimitPriceLower:   1,
		TriggerPrice:      1,
		QuoteAmount:       1,
		ExecutedAmount:    1,
		RemainingAmount:   1,
		Fee:               1,
		Exchange:          "1",
		InternalOrderID:   id,
		OrderID:           "1",
		AccountID:         "1",
		ClientID:          "1",
		ClientOrderID:     "DukeOfWombleton",
		WalletAddress:     "1",
		Type:              1,
		Side:              1,
		Status:            1,
		AssetType:         1,
		LastUpdated:       updated,
		Pair:              pair,
		Trades:            []TradeHistory{},
	}

	od = &Detail{Exchange: "test"}

	err = od.UpdateOrderFromDetail(nil)
	if !errors.Is(err, ErrOrderDetailIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderDetailIsNil)
	}

	err = od.UpdateOrderFromDetail(om)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
	if od.InternalOrderID != id {
		t.Error("Failed to initialize the internal order ID")
	}
	if !od.ImmediateOrCancel {
		t.Error("Failed to update")
	}
	if !od.HiddenOrder {
		t.Error("Failed to update")
	}
	if !od.FillOrKill {
		t.Error("Failed to update")
	}
	if !od.PostOnly {
		t.Error("Failed to update")
	}
	if od.Leverage != 1 {
		t.Error("Failed to update")
	}
	if od.Price != 1 {
		t.Error("Failed to update")
	}
	if od.Amount != 1 {
		t.Error("Failed to update")
	}
	if od.LimitPriceLower != 1 {
		t.Error("Failed to update")
	}
	if od.LimitPriceUpper != 1 {
		t.Error("Failed to update")
	}
	if od.TriggerPrice != 1 {
		t.Error("Failed to update")
	}
	if od.QuoteAmount != 1 {
		t.Error("Failed to update")
	}
	if od.ExecutedAmount != 1 {
		t.Error("Failed to update")
	}
	if od.RemainingAmount != 1 {
		t.Error("Failed to update")
	}
	if od.Fee != 1 {
		t.Error("Failed to update")
	}
	if od.Exchange != "test" {
		t.Error("Should not be able to update exchange via modify")
	}
	if od.OrderID != "1" {
		t.Error("Failed to update")
	}
	if od.ClientID != "1" {
		t.Error("Failed to update")
	}
	if od.ClientOrderID != "DukeOfWombleton" {
		t.Error("Failed to update")
	}
	if od.WalletAddress != "1" {
		t.Error("Failed to update")
	}
	if od.Type != 1 {
		t.Error("Failed to update")
	}
	if od.Side != 1 {
		t.Error("Failed to update")
	}
	if od.Status != 1 {
		t.Error("Failed to update")
	}
	if od.AssetType != 1 {
		t.Error("Failed to update")
	}
	if od.LastUpdated != updated {
		t.Error("Failed to update")
	}
	if od.Pair.String() != "BTCUSD" {
		t.Error("Failed to update")
	}
	if od.Trades != nil {
		t.Error("Failed to update")
	}

	om.Trades = append(om.Trades, TradeHistory{TID: "1"}, TradeHistory{TID: "2"})
	err = od.UpdateOrderFromDetail(om)
	if err != nil {
		t.Fatal(err)
	}
	if len(od.Trades) != 2 {
		t.Error("Failed to add trades")
	}
	om.Trades[0].Exchange = leet
	om.Trades[0].Price = 1337
	om.Trades[0].Fee = 1337
	om.Trades[0].IsMaker = true
	om.Trades[0].Timestamp = updated
	om.Trades[0].Description = leet
	om.Trades[0].Side = UnknownSide
	om.Trades[0].Type = UnknownType
	om.Trades[0].Amount = 1337
	err = od.UpdateOrderFromDetail(om)
	if err != nil {
		t.Fatal(err)
	}
	if od.Trades[0].Exchange == leet {
		t.Error("Should not be able to update exchange from update")
	}
	if od.Trades[0].Price != 1337 {
		t.Error("Failed to update trades")
	}
	if od.Trades[0].Fee != 1337 {
		t.Error("Failed to update trades")
	}
	if !od.Trades[0].IsMaker {
		t.Error("Failed to update trades")
	}
	if od.Trades[0].Timestamp != updated {
		t.Error("Failed to update trades")
	}
	if od.Trades[0].Description != leet {
		t.Error("Failed to update trades")
	}
	if od.Trades[0].Side != UnknownSide {
		t.Error("Failed to update trades")
	}
	if od.Trades[0].Type != UnknownType {
		t.Error("Failed to update trades")
	}
	if od.Trades[0].Amount != 1337 {
		t.Error("Failed to update trades")
	}

	id, err = uuid.NewV4()
	if err != nil {
		t.Fatal(err)
	}

	om = &Detail{
		InternalOrderID: id,
	}

	err = od.UpdateOrderFromDetail(om)
	if err != nil {
		t.Fatal(err)
	}
	if od.InternalOrderID == id {
		t.Error("Should not be able to update the internal order ID after initialization")
	}
}

func TestClassificationError_Error(t *testing.T) {
	class := ClassificationError{OrderID: "1337", Exchange: "test", Err: errors.New("test error")}
	if class.Error() != "Exchange test: OrderID: 1337 classification error: test error" {
		t.Fatal("unexpected output")
	}
	class.OrderID = ""
	if class.Error() != "Exchange test: classification error: test error" {
		t.Fatal("unexpected output")
	}
}

func TestValidationOnOrderTypes(t *testing.T) {
	var cancelMe *Cancel
	if cancelMe.Validate() != ErrCancelOrderIsNil {
		t.Fatal("unexpected error")
	}

	cancelMe = new(Cancel)
	err := cancelMe.Validate()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	err = cancelMe.Validate(cancelMe.PairAssetRequired())
	if err == nil || err.Error() != ErrPairIsEmpty.Error() {
		t.Errorf("received '%v' expected '%v'", err, ErrPairIsEmpty)
	}

	cancelMe.Pair = currency.NewPair(currency.BTC, currency.USDT)
	err = cancelMe.Validate(cancelMe.PairAssetRequired())
	if err == nil || err.Error() != ErrAssetNotSet.Error() {
		t.Errorf("received '%v' expected '%v'", err, ErrAssetNotSet)
	}

	cancelMe.AssetType = asset.Spot
	err = cancelMe.Validate(cancelMe.PairAssetRequired())
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	if cancelMe.Validate(cancelMe.StandardCancel()) == nil {
		t.Fatal("expected error")
	}

	if cancelMe.Validate(validate.Check(func() error {
		return nil
	})) != nil {
		t.Fatal("should return nil")
	}
	cancelMe.OrderID = "1337"
	if cancelMe.Validate(cancelMe.StandardCancel()) != nil {
		t.Fatal("should return nil")
	}

	var getOrders *MultiOrderRequest
	err = getOrders.Validate()
	if !errors.Is(err, ErrGetOrdersRequestIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrGetOrdersRequestIsNil)
	}

	getOrders = new(MultiOrderRequest)
	err = getOrders.Validate()
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received: '%v' but expected: '%v'", err, asset.ErrNotSupported)
	}

	getOrders.AssetType = asset.Spot
	err = getOrders.Validate()
	if !errors.Is(err, errUnrecognisedOrderSide) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errUnrecognisedOrderSide)
	}

	getOrders.Side = AnySide
	err = getOrders.Validate()
	if !errors.Is(err, errUnrecognisedOrderType) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errUnrecognisedOrderType)
	}

	var errTestError = errors.New("test error")
	getOrders.Type = AnyType
	err = getOrders.Validate(validate.Check(func() error {
		return errTestError
	}))
	if !errors.Is(err, errTestError) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errTestError)
	}

	err = getOrders.Validate(validate.Check(func() error {
		return nil
	}))
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	var modifyOrder *Modify
	if modifyOrder.Validate() != ErrModifyOrderIsNil {
		t.Fatal("unexpected error")
	}

	modifyOrder = new(Modify)
	if modifyOrder.Validate() != ErrPairIsEmpty {
		t.Fatal("unexpected error")
	}

	p, err := currency.NewPairFromString("BTC-USD")
	if err != nil {
		t.Fatal(err)
	}

	modifyOrder.Pair = p
	if modifyOrder.Validate() != ErrAssetNotSet {
		t.Fatal("unexpected error")
	}

	modifyOrder.AssetType = asset.Spot
	if modifyOrder.Validate() != ErrOrderIDNotSet {
		t.Fatal("unexpected error")
	}

	modifyOrder.ClientOrderID = "1337"
	if modifyOrder.Validate() != nil {
		t.Fatal("should not error")
	}

	if modifyOrder.Validate(validate.Check(func() error {
		return errors.New("this should error")
	})) == nil {
		t.Fatal("expected error")
	}

	if modifyOrder.Validate(validate.Check(func() error {
		return nil
	})) != nil {
		t.Fatal("unexpected error")
	}
}

func TestMatchFilter(t *testing.T) {
	t.Parallel()
	id, err := uuid.NewV4()
	if err != nil {
		t.Fatal(err)
	}
	filters := map[int]Filter{
		0:  {},
		1:  {Exchange: "Binance"},
		2:  {InternalOrderID: id},
		3:  {OrderID: "2222"},
		4:  {ClientOrderID: "3333"},
		5:  {ClientID: "4444"},
		6:  {WalletAddress: "5555"},
		7:  {Type: AnyType},
		8:  {Type: Limit},
		9:  {Side: AnySide},
		10: {Side: Sell},
		11: {Status: AnyStatus},
		12: {Status: New},
		13: {AssetType: asset.Spot},
		14: {Pair: currency.NewPair(currency.BTC, currency.USD)},
		15: {Exchange: "Binance", Type: Limit, Status: New},
		16: {Exchange: "Binance", Type: AnyType},
		17: {AccountID: "8888"},
	}

	orders := map[int]Detail{
		0:  {},
		1:  {Exchange: "Binance"},
		2:  {InternalOrderID: id},
		3:  {OrderID: "2222"},
		4:  {ClientOrderID: "3333"},
		5:  {ClientID: "4444"},
		6:  {WalletAddress: "5555"},
		7:  {Type: AnyType},
		8:  {Type: Limit},
		9:  {Side: AnySide},
		10: {Side: Sell},
		11: {Status: AnyStatus},
		12: {Status: New},
		13: {AssetType: asset.Spot},
		14: {Pair: currency.NewPair(currency.BTC, currency.USD)},
		15: {Exchange: "Binance", Type: Limit, Status: New},
		16: {AccountID: "8888"},
	}
	// empty filter tests
	emptyFilter := filters[0]
	for _, o := range orders {
		if !o.MatchFilter(&emptyFilter) {
			t.Error("empty filter should match everything")
		}
	}

	tests := map[int]struct {
		f              Filter
		o              Detail
		expectedResult bool
	}{
		0:  {filters[1], orders[1], true},
		1:  {filters[1], orders[0], false},
		2:  {filters[2], orders[2], true},
		3:  {filters[2], orders[3], false},
		4:  {filters[3], orders[3], true},
		5:  {filters[3], orders[4], false},
		6:  {filters[4], orders[4], true},
		7:  {filters[4], orders[5], false},
		8:  {filters[5], orders[5], true},
		9:  {filters[5], orders[6], false},
		10: {filters[6], orders[6], true},
		11: {filters[6], orders[7], false},
		12: {filters[7], orders[7], true},
		13: {filters[7], orders[8], true},
		14: {filters[7], orders[9], true},
		15: {filters[8], orders[7], false},
		16: {filters[8], orders[8], true},
		17: {filters[8], orders[9], false},
		18: {filters[9], orders[9], true},
		19: {filters[9], orders[10], true},
		20: {filters[9], orders[11], true},
		21: {filters[10], orders[10], true},
		22: {filters[10], orders[11], false},
		23: {filters[10], orders[9], false},
		24: {filters[11], orders[11], true},
		25: {filters[11], orders[12], true},
		26: {filters[11], orders[10], true},
		27: {filters[12], orders[12], true},
		28: {filters[12], orders[13], false},
		29: {filters[12], orders[11], false},
		30: {filters[13], orders[13], true},
		31: {filters[13], orders[12], false},
		32: {filters[14], orders[14], true},
		33: {filters[14], orders[13], false},
		34: {filters[15], orders[15], true},
		35: {filters[16], orders[15], true},
		36: {filters[17], orders[16], true},
		37: {filters[17], orders[15], false},
	}
	// specific tests
	for num, tt := range tests {
		if tt.o.MatchFilter(&tt.f) != tt.expectedResult {
			t.Errorf("tests[%v] failed", num)
		}
	}
}

func TestIsActive(t *testing.T) {
	orders := map[int]Detail{
		0: {Amount: 0.0, Status: Active},
		1: {Amount: 1.0, ExecutedAmount: 0.9, Status: Active},
		2: {Amount: 1.0, ExecutedAmount: 1.0, Status: Active},
		3: {Amount: 1.0, ExecutedAmount: 1.1, Status: Active},
	}

	amountTests := map[int]struct {
		o              Detail
		expectedResult bool
	}{
		0: {orders[0], false},
		1: {orders[1], true},
		2: {orders[2], false},
		3: {orders[3], false},
	}
	// specific tests
	for num, tt := range amountTests {
		if tt.o.IsActive() != tt.expectedResult {
			t.Errorf("amountTests[%v] failed", num)
		}
	}

	statusTests := map[int]struct {
		o              Detail
		expectedResult bool
	}{
		// For now force inactive on any status
		0:  {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: AnyStatus}, false},
		1:  {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: New}, true},
		2:  {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: Active}, true},
		3:  {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: PartiallyCancelled}, false},
		4:  {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: PartiallyFilled}, true},
		5:  {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: Filled}, false},
		6:  {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: Cancelled}, false},
		7:  {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: PendingCancel}, true},
		8:  {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: InsufficientBalance}, false},
		9:  {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: MarketUnavailable}, false},
		10: {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: Rejected}, false},
		11: {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: Expired}, false},
		12: {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: Hidden}, true},
		// For now force inactive on unknown status
		13: {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: UnknownStatus}, false},
		14: {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: Open}, true},
		15: {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: AutoDeleverage}, true},
		16: {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: Closed}, false},
		17: {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: Pending}, true},
	}
	// specific tests
	for num, tt := range statusTests {
		if tt.o.IsActive() != tt.expectedResult {
			t.Fatalf("statusTests[%v] failed", num)
		}
	}
}

var activeBenchmark = Detail{Status: Pending, Amount: 1}

// 610732089	         2.414 ns/op	       0 B/op	       0 allocs/op // PREV
// 1000000000	         1.188 ns/op	       0 B/op	       0 allocs/op // CURRENT
func BenchmarkIsActive(b *testing.B) {
	for x := 0; x < b.N; x++ {
		if !activeBenchmark.IsActive() {
			b.Fatal("expected true")
		}
	}
}

func TestIsInactive(t *testing.T) {
	orders := map[int]Detail{
		0: {Amount: 0.0, Status: Active},
		1: {Amount: 1.0, ExecutedAmount: 0.9, Status: Active},
		2: {Amount: 1.0, ExecutedAmount: 1.0, Status: Active},
		3: {Amount: 1.0, ExecutedAmount: 1.1, Status: Active},
	}

	amountTests := map[int]struct {
		o              Detail
		expectedResult bool
	}{
		0: {orders[0], true},
		1: {orders[1], false},
		2: {orders[2], true},
		3: {orders[3], true},
	}
	// specific tests
	for num, tt := range amountTests {
		if tt.o.IsInactive() != tt.expectedResult {
			t.Errorf("amountTests[%v] failed", num)
		}
	}

	statusTests := map[int]struct {
		o              Detail
		expectedResult bool
	}{
		// For now force inactive on any status
		0:  {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: AnyStatus}, true},
		1:  {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: New}, false},
		2:  {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: Active}, false},
		3:  {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: PartiallyCancelled}, true},
		4:  {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: PartiallyFilled}, false},
		5:  {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: Filled}, true},
		6:  {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: Cancelled}, true},
		7:  {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: PendingCancel}, false},
		8:  {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: InsufficientBalance}, true},
		9:  {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: MarketUnavailable}, true},
		10: {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: Rejected}, true},
		11: {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: Expired}, true},
		12: {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: Hidden}, false},
		// For now force inactive on unknown status
		13: {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: UnknownStatus}, true},
		14: {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: Open}, false},
		15: {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: AutoDeleverage}, false},
		16: {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: Closed}, true},
		17: {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: Pending}, false},
	}
	// specific tests
	for num, tt := range statusTests {
		if tt.o.IsInactive() != tt.expectedResult {
			t.Errorf("statusTests[%v] failed", num)
		}
	}
}

var inactiveBenchmark = Detail{Status: Closed, Amount: 1}

// 1000000000	         1.043 ns/op	       0 B/op	       0 allocs/op // CURRENT
func BenchmarkIsInactive(b *testing.B) {
	for x := 0; x < b.N; x++ {
		if !inactiveBenchmark.IsInactive() {
			b.Fatal("expected true")
		}
	}
}

func TestIsOrderPlaced(t *testing.T) {
	t.Parallel()
	statusTests := map[int]struct {
		o              Detail
		expectedResult bool
	}{
		0:  {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: AnyStatus}, false},
		1:  {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: New}, true},
		2:  {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: Active}, true},
		3:  {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: PartiallyCancelled}, true},
		4:  {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: PartiallyFilled}, true},
		5:  {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: Filled}, true},
		6:  {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: Cancelled}, true},
		7:  {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: PendingCancel}, true},
		8:  {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: InsufficientBalance}, false},
		9:  {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: MarketUnavailable}, false},
		10: {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: Rejected}, false},
		11: {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: Expired}, true},
		12: {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: Hidden}, true},
		13: {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: UnknownStatus}, false},
		14: {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: Open}, true},
		15: {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: AutoDeleverage}, true},
		16: {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: Closed}, true},
		17: {Detail{Amount: 1.0, ExecutedAmount: 0.0, Status: Pending}, true},
	}
	// specific tests
	for num, tt := range statusTests {
		num := num
		tt := tt
		t.Run(fmt.Sprintf("TEST CASE: %d", num), func(t *testing.T) {
			t.Parallel()
			if tt.o.WasOrderPlaced() != tt.expectedResult {
				t.Errorf("statusTests[%v] failed", num)
			}
		})
	}
}

func TestGenerateInternalOrderID(t *testing.T) {
	id, err := uuid.NewV4()
	if err != nil {
		t.Errorf("unable to create uuid: %s", err)
	}
	od := Detail{
		InternalOrderID: id,
	}
	od.GenerateInternalOrderID()
	if od.InternalOrderID != id {
		t.Error("Should not be able to generate a new internal order ID")
	}

	od = Detail{}
	od.GenerateInternalOrderID()
	if od.InternalOrderID.IsNil() {
		t.Error("unable to generate internal order ID")
	}
}

func TestDetail_Copy(t *testing.T) {
	t.Parallel()
	d := []Detail{
		{
			Exchange: "Binance",
		},
		{
			Exchange: "Binance",
			Trades: []TradeHistory{
				{Price: 1},
			},
		},
	}
	for i := range d {
		r := d[i].Copy()
		if !reflect.DeepEqual(d[i], r) {
			t.Errorf("[%d] Copy does not contain same elements, expected: %v\ngot:%v", i, d[i], r)
		}
		if len(d[i].Trades) > 0 {
			if &d[i].Trades[0] == &r.Trades[0] {
				t.Errorf("[%d]Trades point to the same data elements", i)
			}
		}
	}
}

func TestDetail_CopyToPointer(t *testing.T) {
	t.Parallel()
	d := []Detail{
		{
			Exchange: "Binance",
		},
		{
			Exchange: "Binance",
			Trades: []TradeHistory{
				{Price: 1},
			},
		},
	}
	for i := range d {
		r := d[i].CopyToPointer()
		if !reflect.DeepEqual(d[i], *r) {
			t.Errorf("[%d] Copy does not contain same elements, expected: %v\ngot:%v", i, d[i], r)
		}
		if len(d[i].Trades) > 0 {
			if &d[i].Trades[0] == &r.Trades[0] {
				t.Errorf("[%d]Trades point to the same data elements", i)
			}
		}
	}
}

func TestDetail_CopyPointerOrderSlice(t *testing.T) {
	t.Parallel()
	d := []*Detail{
		{
			Exchange: "Binance",
		},
		{
			Exchange: "Binance",
			Trades: []TradeHistory{
				{Price: 1},
			},
		},
	}

	sliceCopy := CopyPointerOrderSlice(d)
	for i := range sliceCopy {
		if !reflect.DeepEqual(*sliceCopy[i], *d[i]) {
			t.Errorf("[%d] Copy does not contain same elements, expected: %v\ngot:%v", i, sliceCopy[i], d[i])
		}
		if len(sliceCopy[i].Trades) > 0 {
			if &sliceCopy[i].Trades[0] == &d[i].Trades[0] {
				t.Errorf("[%d]Trades point to the same data elements", i)
			}
		}
	}
}

func TestDeriveModify(t *testing.T) {
	t.Parallel()
	var o *Detail
	if _, err := o.DeriveModify(); !errors.Is(err, errOrderDetailIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errOrderDetailIsNil)
	}

	pair := currency.NewPair(currency.BTC, currency.AUD)

	o = &Detail{
		Exchange:      "wow",
		OrderID:       "wow2",
		ClientOrderID: "wow3",
		Type:          Market,
		Side:          Long,
		AssetType:     asset.Futures,
		Pair:          pair,
	}

	mod, err := o.DeriveModify()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if mod == nil {
		t.Fatal("should not be nil")
	}

	if mod.Exchange != "wow" ||
		mod.OrderID != "wow2" ||
		mod.ClientOrderID != "wow3" ||
		mod.Type != Market ||
		mod.Side != Long ||
		mod.AssetType != asset.Futures ||
		!mod.Pair.Equal(pair) {
		t.Fatal("unexpected values")
	}
}

func TestDeriveModifyResponse(t *testing.T) {
	t.Parallel()
	var mod *Modify
	if _, err := mod.DeriveModifyResponse(); !errors.Is(err, errOrderDetailIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errOrderDetailIsNil)
	}

	pair := currency.NewPair(currency.BTC, currency.AUD)

	mod = &Modify{
		Exchange:      "wow",
		OrderID:       "wow2",
		ClientOrderID: "wow3",
		Type:          Market,
		Side:          Long,
		AssetType:     asset.Futures,
		Pair:          pair,
	}

	modresp, err := mod.DeriveModifyResponse()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if modresp == nil {
		t.Fatal("should not be nil")
	}

	if modresp.Exchange != "wow" ||
		modresp.OrderID != "wow2" ||
		modresp.ClientOrderID != "wow3" ||
		modresp.Type != Market ||
		modresp.Side != Long ||
		modresp.AssetType != asset.Futures ||
		!modresp.Pair.Equal(pair) {
		t.Fatal("unexpected values")
	}
}

func TestDeriveCancel(t *testing.T) {
	t.Parallel()
	var o *Detail
	if _, err := o.DeriveCancel(); !errors.Is(err, errOrderDetailIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errOrderDetailIsNil)
	}

	pair := currency.NewPair(currency.BTC, currency.AUD)

	o = &Detail{
		Exchange:      "wow",
		OrderID:       "wow1",
		AccountID:     "wow2",
		ClientID:      "wow3",
		ClientOrderID: "wow4",
		WalletAddress: "wow5",
		Type:          Market,
		Side:          Long,
		Pair:          pair,
		AssetType:     asset.Futures,
	}
	cancel, err := o.DeriveCancel()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
	if cancel.Exchange != "wow" ||
		cancel.OrderID != "wow1" ||
		cancel.AccountID != "wow2" ||
		cancel.ClientID != "wow3" ||
		cancel.ClientOrderID != "wow4" ||
		cancel.WalletAddress != "wow5" ||
		cancel.Type != Market ||
		cancel.Side != Long ||
		!cancel.Pair.Equal(pair) ||
		cancel.AssetType != asset.Futures {
		t.Fatalf("unexpected values %+v", cancel)
	}
}

func TestGetOrdersRequest_Filter(t *testing.T) {
	request := new(MultiOrderRequest)
	request.AssetType = asset.Spot
	request.Type = AnyType
	request.Side = AnySide

	var orders = []Detail{
		{OrderID: "0", Pair: btcusd, AssetType: asset.Spot, Type: Limit, Side: Buy},
		{OrderID: "1", Pair: btcusd, AssetType: asset.Spot, Type: Limit, Side: Sell},
		{OrderID: "2", Pair: btcusd, AssetType: asset.Spot, Type: Market, Side: Buy},
		{OrderID: "3", Pair: btcusd, AssetType: asset.Spot, Type: Market, Side: Sell},
		{OrderID: "4", Pair: btcusd, AssetType: asset.Futures, Type: Limit, Side: Buy},
		{OrderID: "5", Pair: btcusd, AssetType: asset.Futures, Type: Limit, Side: Sell},
		{OrderID: "6", Pair: btcusd, AssetType: asset.Futures, Type: Market, Side: Buy},
		{OrderID: "7", Pair: btcusd, AssetType: asset.Futures, Type: Market, Side: Sell},
		{OrderID: "8", Pair: btcltc, AssetType: asset.Spot, Type: Limit, Side: Buy},
		{OrderID: "9", Pair: btcltc, AssetType: asset.Spot, Type: Limit, Side: Sell},
		{OrderID: "10", Pair: btcltc, AssetType: asset.Spot, Type: Market, Side: Buy},
		{OrderID: "11", Pair: btcltc, AssetType: asset.Spot, Type: Market, Side: Sell},
		{OrderID: "12", Pair: btcltc, AssetType: asset.Futures, Type: Limit, Side: Buy},
		{OrderID: "13", Pair: btcltc, AssetType: asset.Futures, Type: Limit, Side: Sell},
		{OrderID: "14", Pair: btcltc, AssetType: asset.Futures, Type: Market, Side: Buy},
		{OrderID: "15", Pair: btcltc, AssetType: asset.Futures, Type: Market, Side: Sell},
	}

	shinyAndClean := request.Filter("test", orders)
	if len(shinyAndClean) != 16 {
		t.Fatalf("received: '%v' but expected: '%v'", len(shinyAndClean), 16)
	}

	for x := range shinyAndClean {
		if strconv.FormatInt(int64(x), 10) != shinyAndClean[x].OrderID {
			t.Fatalf("received: '%v' but expected: '%v'", shinyAndClean[x].OrderID, int64(x))
		}
	}

	request.Pairs = []currency.Pair{btcltc}

	// Kicks off time error
	request.EndTime = time.Unix(1336, 0)
	request.StartTime = time.Unix(1337, 0)

	shinyAndClean = request.Filter("test", orders)

	if len(shinyAndClean) != 8 {
		t.Fatalf("received: '%v' but expected: '%v'", len(shinyAndClean), 8)
	}

	for x := range shinyAndClean {
		if strconv.FormatInt(int64(x)+8, 10) != shinyAndClean[x].OrderID {
			t.Fatalf("received: '%v' but expected: '%v'", shinyAndClean[x].OrderID, int64(x)+8)
		}
	}
}

func TestIsValidOrderSubmissionSide(t *testing.T) {
	t.Parallel()
	if IsValidOrderSubmissionSide(UnknownSide) {
		t.Error("expected false")
	}
	if !IsValidOrderSubmissionSide(Buy) {
		t.Error("expected true")
	}
	if IsValidOrderSubmissionSide(CouldNotBuy) {
		t.Error("expected false")
	}
}

func TestAdjustBaseAmount(t *testing.T) {
	t.Parallel()

	var s *SubmitResponse
	err := s.AdjustBaseAmount(0)
	if !errors.Is(err, errOrderSubmitResponseIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errOrderSubmitResponseIsNil)
	}

	s = &SubmitResponse{}
	err = s.AdjustBaseAmount(0)
	if !errors.Is(err, errAmountIsZero) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errAmountIsZero)
	}

	s.Amount = 1.7777777777
	err = s.AdjustBaseAmount(1.7777777777)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if s.Amount != 1.7777777777 {
		t.Fatalf("received: '%v' but expected: '%v'", s.Amount, 1.7777777777)
	}

	s.Amount = 1.7777777777
	err = s.AdjustBaseAmount(1.777)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if s.Amount != 1.777 {
		t.Fatalf("received: '%v' but expected: '%v'", s.Amount, 1.777)
	}
}

func TestAdjustQuoteAmount(t *testing.T) {
	t.Parallel()

	var s *SubmitResponse
	err := s.AdjustQuoteAmount(0)
	if !errors.Is(err, errOrderSubmitResponseIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errOrderSubmitResponseIsNil)
	}

	s = &SubmitResponse{}
	err = s.AdjustQuoteAmount(0)
	if !errors.Is(err, errAmountIsZero) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errAmountIsZero)
	}

	s.QuoteAmount = 5.222222222222
	err = s.AdjustQuoteAmount(5.222222222222)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if s.QuoteAmount != 5.222222222222 {
		t.Fatalf("received: '%v' but expected: '%v'", s.Amount, 5.222222222222)
	}

	s.QuoteAmount = 5.222222222222
	err = s.AdjustQuoteAmount(5.22222222)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if s.QuoteAmount != 5.22222222 {
		t.Fatalf("received: '%v' but expected: '%v'", s.Amount, 5.22222222)
	}
}
