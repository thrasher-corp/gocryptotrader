package order

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/validate"
)

var errValidationCheckFailed = errors.New("validation check failed")

func TestSubmitValidate(t *testing.T) {
	t.Parallel()
	testPair := currency.NewPair(currency.BTC, currency.LTC)
	tester := []struct {
		ExpectedErr                     error
		Submit                          *Submit
		ValidOpts                       validate.Checker
		HasToPurchaseWithQuoteAmountSet bool
		HasToSellWithBaseAmountSet      bool
		RequiresID                      bool
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
				AssetType: asset.All,
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
			ExpectedErr: ErrInvalidTimeInForce,
			Submit: &Submit{
				Exchange:    "test",
				Pair:        testPair,
				AssetType:   asset.Spot,
				Side:        Ask,
				Type:        Market,
				TimeInForce: TimeInForce(89),
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
		{
			ExpectedErr: ErrAmountMustBeSet,
			Submit: &Submit{
				Exchange:  "test",
				Pair:      testPair,
				Side:      Buy,
				Type:      Market,
				Amount:    1,
				AssetType: asset.Spot,
			},
			HasToPurchaseWithQuoteAmountSet: true,
			ValidOpts:                       validate.Check(func() error { return nil }),
		},
		{
			ExpectedErr: ErrAmountMustBeSet,
			Submit: &Submit{
				Exchange:    "test",
				Pair:        testPair,
				Side:        Sell,
				Type:        Market,
				QuoteAmount: 1,
				AssetType:   asset.Spot,
			},
			HasToSellWithBaseAmountSet: true,
			ValidOpts:                  validate.Check(func() error { return nil }),
		},
		{
			ExpectedErr: ErrClientOrderIDMustBeSet,
			Submit: &Submit{
				Exchange:  "test",
				Pair:      testPair,
				Side:      Buy,
				Type:      Market,
				Amount:    1,
				AssetType: asset.Spot,
			},
			RequiresID: true,
			ValidOpts:  validate.Check(func() error { return nil }),
		},
		{
			ExpectedErr: nil,
			Submit: &Submit{
				Exchange:      "test",
				Pair:          testPair,
				Side:          Buy,
				Type:          Market,
				Amount:        1,
				AssetType:     asset.Spot,
				ClientOrderID: "69420",
			},
			RequiresID: true,
			ValidOpts:  validate.Check(func() error { return nil }),
		},
	}

	for x, tc := range tester {
		t.Run(strconv.Itoa(x), func(t *testing.T) {
			t.Parallel()
			requirements := protocol.TradingRequirements{
				SpotMarketOrderAmountPurchaseQuotationOnly: tc.HasToPurchaseWithQuoteAmountSet,
				SpotMarketOrderAmountSellBaseOnly:          tc.HasToSellWithBaseAmountSet,
				ClientOrderID:                              tc.RequiresID,
			}
			err := tc.Submit.Validate(requirements, tc.ValidOpts)
			assert.ErrorIs(t, err, tc.ExpectedErr)
		})
	}
}

func TestSubmit_DeriveSubmitResponse(t *testing.T) {
	t.Parallel()
	var s *Submit
	_, err := s.DeriveSubmitResponse("")
	require.ErrorIs(t, err, errOrderSubmitIsNil)

	s = &Submit{}
	_, err = s.DeriveSubmitResponse("")
	require.ErrorIs(t, err, ErrOrderIDNotSet)

	resp, err := s.DeriveSubmitResponse("1337")
	require.NoError(t, err)
	require.Equal(t, "1337", resp.OrderID)
	require.Equal(t, New, resp.Status)
	require.False(t, resp.Date.IsZero())
	assert.False(t, resp.LastUpdated.IsZero())
}

func TestSubmitResponse_DeriveDetail(t *testing.T) {
	t.Parallel()
	var s *SubmitResponse
	_, err := s.DeriveDetail(uuid.Nil)
	require.ErrorIs(t, err, errOrderSubmitResponseIsNil)

	id, err := uuid.NewV4()
	require.NoError(t, err)

	s = &SubmitResponse{}
	deets, err := s.DeriveDetail(id)
	require.NoError(t, err)
	assert.Equal(t, id, deets.InternalOrderID)
}

func TestOrderSides(t *testing.T) {
	t.Parallel()
	os := Buy
	assert.Equal(t, "BUY", os.String())
	assert.Equal(t, "buy", os.Lower())
	assert.Equal(t, "Buy", os.Title())
}

func TestTitle(t *testing.T) {
	t.Parallel()
	orderType := Limit
	require.Equal(t, "Limit", orderType.Title())
}

func TestOrderIs(t *testing.T) {
	t.Parallel()
	orderComparisonList := []struct {
		Type    Type
		Targets []Type
	}{
		{Type: Limit | TakeProfit, Targets: []Type{TakeProfit, Limit}},
		{Type: IOS, Targets: []Type{IOS}},
		{Type: Stop, Targets: []Type{Stop}},
		{Type: AnyType, Targets: []Type{AnyType}},
		{Type: StopLimit, Targets: []Type{Stop, Limit}},
		{Type: TakeProfit, Targets: []Type{TakeProfit}},
		{Type: StopMarket, Targets: []Type{Stop, Market}},
		{Type: TrailingStop, Targets: []Type{TrailingStop}},
		{Type: UnknownType | Limit, Targets: []Type{Limit}},
		{Type: TakeProfitMarket, Targets: []Type{TakeProfit, Market}},
	}
	for _, oType := range orderComparisonList {
		t.Run(oType.Type.String(), func(t *testing.T) {
			t.Parallel()
			for _, target := range oType.Targets {
				assert.Truef(t, oType.Type.Is(target), "expected %v, got %q", target, oType.Type.String())
			}
		})
	}
}

func TestOrderTypes(t *testing.T) {
	t.Parallel()
	var orderType Type
	assert.Equal(t, "UNKNOWN", orderType.String())
	assert.Equal(t, "unknown", orderType.Lower())
	assert.Equal(t, "Unknown", orderType.Title())
}

func TestOrderTypeToString(t *testing.T) {
	t.Parallel()
	orderToToStringsList := []struct {
		OrderType Type
		String    string
	}{
		{StopMarket, "STOP MARKET"},
		{StopLimit, "STOP LIMIT"},
		{Limit, "LIMIT"},
		{Market, "MARKET"},
		{Stop, "STOP"},
		{ConditionalStop, "CONDITIONAL"},
		{TWAP, "TWAP"},
		{Chase, "CHASE"},
		{TakeProfit, "TAKE PROFIT"},
		{TakeProfitMarket, "TAKE PROFIT MARKET"},
		{TrailingStop, "TRAILING_STOP"},
		{IOS, "IOS"},
		{Liquidation, "LIQUIDATION"},
		{Trigger, "TRIGGER"},
		{OCO, "OCO"},
		{OptimalLimit, "OPTIMAL_LIMIT"},
		{MarketMakerProtection, "MMP"},
		{AnyType, "ANY"},
		{UnknownType | Limit, "LIMIT"},
		{StopMarket | ConditionalStop, "UNKNOWN"},
	}
	for _, tt := range orderToToStringsList {
		t.Run(tt.String, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.String, tt.OrderType.String())
		})
	}
}

func TestInferCostsAndTimes(t *testing.T) {
	t.Parallel()
	var detail Detail
	detail.InferCostsAndTimes()
	assert.Zero(t, detail.Amount, "InferCostsAndTimes on empty details should set correct Amount")

	detail.CloseTime = time.Now()
	detail.InferCostsAndTimes()
	assert.Equal(t, detail.CloseTime, detail.LastUpdated, "Order last updated not equals close time")

	detail.Amount = 1
	detail.ExecutedAmount = 1
	detail.InferCostsAndTimes()
	assert.Zero(t, detail.AverageExecutedPrice, "InferCostsAndTimes should set AverageExecutedPrice correctly")

	detail.Amount = 1
	detail.ExecutedAmount = 1
	detail.InferCostsAndTimes()
	assert.Zero(t, detail.Cost, "InferCostsAndTimes should set Cost correctly")

	detail.ExecutedAmount = 0

	detail.Amount = 1
	detail.RemainingAmount = 1
	detail.InferCostsAndTimes()
	assert.Equal(t, detail.ExecutedAmount+detail.RemainingAmount, detail.Amount)
	detail.RemainingAmount = 0

	detail.Amount = 1
	detail.ExecutedAmount = 1
	detail.Price = 2
	detail.InferCostsAndTimes()
	assert.Equal(t, 2.0, detail.AverageExecutedPrice)

	detail = Detail{Amount: 1, ExecutedAmount: 2, Cost: 3, Price: 0}
	detail.InferCostsAndTimes()
	assert.Equal(t, 1.5, detail.AverageExecutedPrice)

	detail = Detail{Amount: 1, ExecutedAmount: 2, AverageExecutedPrice: 3}
	detail.InferCostsAndTimes()
	assert.Equal(t, 6.0, detail.Cost)
}

func TestFilterOrdersByType(t *testing.T) {
	t.Parallel()

	orders := []Detail{
		{
			Type: Limit,
		},
		{}, // Unpopulated fields are preserved for API differences
	}

	FilterOrdersByType(&orders, AnyType)
	assert.Len(t, orders, 2, "Orders should be filtered correctly")

	FilterOrdersByType(&orders, Limit)
	assert.Len(t, orders, 2, "Orders should be filtered correctly")

	FilterOrdersByType(&orders, Stop)
	assert.Len(t, orders, 1, "Orders should be filtered correctly")
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
	for b.Loop() {
		FilterOrdersByType(filterOrdersByTypeBenchmark, Limit)
	}
}

func TestFilterOrdersBySide(t *testing.T) {
	t.Parallel()

	orders := []Detail{
		{
			Side: Buy,
		},
		{
			Side: Sell,
		},
		{}, // Unpopulated fields are preserved for API differences
	}

	FilterOrdersBySide(&orders, AnySide)
	assert.Len(t, orders, 3, "Orders should be filtered correctly")

	FilterOrdersBySide(&orders, Buy)
	assert.Len(t, orders, 2, "Orders should be filtered correctly")

	FilterOrdersBySide(&orders, Sell)
	assert.Len(t, orders, 1, "Orders should be filtered correctly")
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
	for b.Loop() {
		FilterOrdersBySide(filterOrdersBySideBenchmark, Ask)
	}
}

func TestFilterOrdersByTimeRange(t *testing.T) {
	t.Parallel()

	orders := []Detail{
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
	require.NoError(t, err)
	assert.Len(t, orders, 3, "Orders should be filtered correctly")

	err = FilterOrdersByTimeRange(&orders, time.Unix(100, 0), time.Unix(111, 0))
	require.NoError(t, err)
	assert.Len(t, orders, 3, "Orders should be filtered correctly")

	err = FilterOrdersByTimeRange(&orders, time.Unix(101, 0), time.Unix(111, 0))
	require.NoError(t, err)
	assert.Len(t, orders, 2, "Orders should be filtered correctly")

	err = FilterOrdersByTimeRange(&orders, time.Unix(200, 0), time.Unix(300, 0))
	require.NoError(t, err)
	assert.Empty(t, orders, "Orders should be filtered correctly")

	orders = append(orders, Detail{})
	// test for event no timestamp is set on an order, best to include it
	err = FilterOrdersByTimeRange(&orders, time.Unix(200, 0), time.Unix(300, 0))
	require.NoError(t, err)
	assert.Len(t, orders, 1, "Orders should be filtered correctly")

	err = FilterOrdersByTimeRange(&orders, time.Unix(300, 0), time.Unix(50, 0))
	require.ErrorIs(t, err, common.ErrStartAfterEnd)
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
	for b.Loop() {
		err := FilterOrdersByTimeRange(filterOrdersByTimeRangeBenchmark, time.Unix(50, 0), time.Unix(150, 0))
		require.NoError(b, err)
	}
}

func TestFilterOrdersByPairs(t *testing.T) {
	t.Parallel()

	orders := []Detail{
		{
			Pair: currency.NewBTCUSD(),
		},
		{
			Pair: currency.NewPair(currency.LTC, currency.EUR),
		},
		{
			Pair: currency.NewPair(currency.DOGE, currency.RUB),
		},
		{}, // Unpopulated fields are preserved for API differences
	}

	currencies := []currency.Pair{
		currency.NewBTCUSD(),
		currency.NewPair(currency.LTC, currency.EUR),
		currency.NewPair(currency.DOGE, currency.RUB),
	}
	FilterOrdersByPairs(&orders, currencies)
	assert.Len(t, orders, 4, "Orders should be filtered correctly")

	currencies = []currency.Pair{
		currency.NewBTCUSD(),
		currency.NewPair(currency.LTC, currency.EUR),
	}
	FilterOrdersByPairs(&orders, currencies)
	assert.Len(t, orders, 3, "Orders should be filtered correctly")

	currencies = []currency.Pair{currency.NewBTCUSD()}
	FilterOrdersByPairs(&orders, currencies)
	assert.Len(t, orders, 2, "Orders should be filtered correctly")

	currencies = []currency.Pair{currency.NewPair(currency.USD, currency.BTC)}
	FilterOrdersByPairs(&orders, currencies)
	assert.Len(t, orders, 2, "Reverse Orders should be filtered correctly")

	currencies = []currency.Pair{}
	FilterOrdersByPairs(&orders, currencies)
	assert.Len(t, orders, 2, "Orders should be filtered correctly")

	currencies = append(currencies, currency.EMPTYPAIR)
	FilterOrdersByPairs(&orders, currencies)
	assert.Len(t, orders, 2, "Orders should be filtered correctly")
}

var filterOrdersByPairsBenchmark = &[]Detail{
	{Pair: currency.NewBTCUSD()},
	{Pair: currency.NewBTCUSD()},
	{Pair: currency.NewBTCUSD()},
	{Pair: currency.NewBTCUSD()},
	{Pair: currency.NewBTCUSD()},
	{Pair: currency.NewBTCUSD()},
	{Pair: currency.NewBTCUSD()},
	{Pair: currency.NewBTCUSD()},
	{Pair: currency.NewBTCUSD()},
	{Pair: currency.NewBTCUSD()},
}

// BenchmarkFilterOrdersByPairs benchmark
//
// 400032	      2977 ns/op	   15840 B/op	       5 allocs/op // PREV
// 6977242	       172.8 ns/op	       0 B/op	       0 allocs/op // CURRENT
func BenchmarkFilterOrdersByPairs(b *testing.B) {
	pairs := []currency.Pair{currency.NewBTCUSD()}
	for b.Loop() {
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
	assert.Zero(t, orders[0].Price, "Price should be correct")

	SortOrdersByPrice(&orders, true)
	assert.Equal(t, 100.0, orders[0].Price, "Price should be correct")
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
	assert.Equal(t, orders[0].Date.Unix(), time.Unix(0, 0).Unix())

	SortOrdersByDate(&orders, true)
	assert.Equal(t, orders[0].Date, time.Unix(2, 0))
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
	assert.Equal(t, currency.BTC.String()+"-"+currency.RUB.String(), orders[0].Pair.String())

	SortOrdersByCurrency(&orders, true)
	assert.Equal(t, currency.LTC.String()+"-"+currency.EUR.String(), orders[0].Pair.String())
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
	assert.Equal(t, Buy.String(), orders[0].Side.String())

	SortOrdersBySide(&orders, true)
	assert.Equal(t, Sell.String(), orders[0].Side.String())
}

func TestSortOrdersByOrderType(t *testing.T) {
	t.Parallel()

	orders := []Detail{
		{
			Type: Market,
		}, {
			Type: Limit,
		}, {
			Type: TrailingStop,
		},
	}

	SortOrdersByType(&orders, false)
	assert.Equal(t, Limit.String(), orders[0].Type.String())

	SortOrdersByType(&orders, true)
	assert.Equal(t, TrailingStop.String(), orders[0].Type.String())
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
		{"woahMan", UnknownSide, ErrSideIsInvalid},
	}
	for i := range cases {
		testData := &cases[i]
		t.Run(testData.in, func(t *testing.T) {
			out, err := StringToOrderSide(testData.in)
			require.ErrorIs(t, err, testData.err)
			require.Equal(t, out, testData.out)
		})
	}
}

var sideBenchmark Side

// 9756914	       126.7 ns/op	       0 B/op	       0 allocs/op // PREV
// 25200660	        57.63 ns/op	       3 B/op	       1 allocs/op // CURRENT
func BenchmarkStringToOrderSide(b *testing.B) {
	for b.Loop() {
		sideBenchmark, _ = StringToOrderSide("any")
	}
}

func TestStringToOrderType(t *testing.T) {
	t.Parallel()
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
		{"stop", Stop, nil},
		{"STOP", Stop, nil},
		{"sToP", Stop, nil},
		{"sToP LiMit", StopLimit, nil},
		{"ExchangE sToP Limit", StopLimit, nil},
		{"trailing_stop", TrailingStop, nil},
		{"TRAILING_STOP", TrailingStop, nil},
		{"tRaIlInG_sToP", TrailingStop, nil},
		{"tRaIlInG sToP", TrailingStop, nil},
		{"ios", IOS, nil},
		{"any", AnyType, nil},
		{"ANY", AnyType, nil},
		{"aNy", AnyType, nil},
		{"trigger", Trigger, nil},
		{"TRIGGER", Trigger, nil},
		{"tRiGgEr", Trigger, nil},
		{"conDitiOnal", ConditionalStop, nil},
		{"oCo", OCO, nil},
		{"mMp", MarketMakerProtection, nil},
		{"tWaP", TWAP, nil},
		{"TWAP", TWAP, nil},
		{"woahMan", UnknownType, errUnrecognisedOrderType},
		{"chase", Chase, nil},
		{"MOVE_ORDER_STOP", TrailingStop, nil},
		{"mOVe_OrdeR_StoP", TrailingStop, nil},
		{"Stop_market", StopMarket, nil},
		{"liquidation", Liquidation, nil},
		{"LiQuidation", Liquidation, nil},
		{"take_profit", TakeProfit, nil},
		{"Take ProfIt", TakeProfit, nil},
		{"TAKE PROFIT MARkEt", TakeProfitMarket, nil},
		{"TAKE_PROFIT_MARkEt", TakeProfitMarket, nil},
		{"optimal_limit", OptimalLimit, nil},
		{"OPTIMAL_LIMIT", OptimalLimit, nil},
	}
	for i := range cases {
		testData := &cases[i]
		t.Run(testData.in, func(t *testing.T) {
			t.Parallel()
			out, err := StringToOrderType(testData.in)
			require.ErrorIs(t, err, testData.err)
			assert.Equal(t, testData.out, out)
		})
	}
}

var typeBenchmark Type

// 5703705	       299.9 ns/op	       0 B/op	       0 allocs/op // PREV
// 16353608	        81.23 ns/op	       8 B/op	       1 allocs/op // CURRENT
func BenchmarkStringToOrderType(b *testing.B) {
	for b.Loop() {
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
			require.ErrorIs(t, err, testData.err)
			assert.Equal(t, testData.out, out)
		})
	}
}

var statusBenchmark Status

// 3569052	       351.8 ns/op	       0 B/op	       0 allocs/op // PREV
// 11126791	       101.9 ns/op	      24 B/op	       1 allocs/op // CURRENT
func BenchmarkStringToOrderStatus(b *testing.B) {
	for b.Loop() {
		statusBenchmark, _ = StringToOrderStatus("market_unavailable")
	}
}

func TestUpdateOrderFromModifyResponse(t *testing.T) {
	od := Detail{OrderID: "1"}
	updated := time.Now()

	om := ModifyResponse{
		TimeInForce:     PostOnly | GoodTillTime,
		Price:           1,
		Amount:          1,
		TriggerPrice:    1,
		RemainingAmount: 1,
		Exchange:        "1",
		Type:            1,
		Side:            1,
		Status:          1,
		AssetType:       1,
		LastUpdated:     updated,
		Pair:            currency.NewBTCUSD(),
	}

	od.UpdateOrderFromModifyResponse(&om)
	require.NotEqual(t, UnknownTIF, od.TimeInForce)
	assert.True(t, od.TimeInForce.Is(GoodTillTime))
	assert.True(t, od.TimeInForce.Is(PostOnly))
	assert.Equal(t, 1.0, od.Price)
	assert.Equal(t, 1.0, od.Amount)
	assert.Equal(t, 1.0, od.TriggerPrice)
	assert.Equal(t, 1.0, od.RemainingAmount)
	assert.Empty(t, od.Exchange, "Should not be able to update exchange via modify")
	assert.Equal(t, "1", od.OrderID)
	assert.Equal(t, Type(1), od.Type)
	assert.Equal(t, Side(1), od.Side)
	assert.Equal(t, Status(1), od.Status)
	assert.Equal(t, asset.Item(1), od.AssetType)
	assert.Equal(t, od.LastUpdated, updated)
	assert.Equal(t, "BTCUSD", od.Pair.String())
	assert.Nil(t, od.Trades)
}

func TestUpdateOrderFromDetail(t *testing.T) {
	t.Parallel()

	var od *Detail
	err := od.UpdateOrderFromDetail(nil)
	require.ErrorIs(t, err, ErrOrderDetailIsNil)

	id, err := uuid.NewV4()
	require.NoError(t, err)
	const leet = "1337"
	updated := time.Now()

	om := &Detail{
		TimeInForce:     GoodTillCancel | PostOnly,
		HiddenOrder:     true,
		Leverage:        1,
		Price:           1,
		Amount:          1,
		LimitPriceUpper: 1,
		LimitPriceLower: 1,
		TriggerPrice:    1,
		QuoteAmount:     1,
		ExecutedAmount:  1,
		RemainingAmount: 1,
		Fee:             1,
		Exchange:        "1",
		InternalOrderID: id,
		OrderID:         "1",
		AccountID:       "1",
		ClientID:        "1",
		ClientOrderID:   "DukeOfWombleton",
		Type:            1,
		Side:            1,
		Status:          1,
		AssetType:       1,
		LastUpdated:     updated,
		Pair:            currency.NewBTCUSD(),
		Trades:          []TradeHistory{},
	}

	od = &Detail{Exchange: "test"}

	err = od.UpdateOrderFromDetail(nil)
	require.ErrorIs(t, err, ErrOrderDetailIsNil)

	err = od.UpdateOrderFromDetail(om)
	require.NoError(t, err)

	assert.Equal(t, od.InternalOrderID, id)
	assert.True(t, od.TimeInForce.Is(GoodTillCancel))
	assert.True(t, od.TimeInForce.Is(PostOnly))
	require.True(t, od.HiddenOrder)
	assert.Equal(t, 1.0, od.Leverage)
	assert.Equal(t, 1.0, od.Price)
	assert.Equal(t, 1.0, od.Amount)
	assert.Equal(t, 1.0, od.LimitPriceLower)
	assert.Equal(t, 1.0, od.LimitPriceUpper)
	assert.Equal(t, 1.0, od.TriggerPrice)
	assert.Equal(t, 1.0, od.QuoteAmount)
	assert.Equal(t, 1.0, od.ExecutedAmount)
	assert.Equal(t, 1.0, od.RemainingAmount)
	assert.Equal(t, 1.0, od.Fee)
	assert.Equal(t, "test", od.Exchange, "Should not be able to update exchange via modify")
	assert.Equal(t, "1", od.OrderID)
	assert.Equal(t, "1", od.ClientID)
	assert.Equal(t, "DukeOfWombleton", od.ClientOrderID)
	assert.Equal(t, Type(1), od.Type)
	assert.Equal(t, Side(1), od.Side)
	assert.Equal(t, Status(1), od.Status)
	assert.Equal(t, asset.Item(1), od.AssetType)
	assert.Equal(t, updated, od.LastUpdated)
	assert.Equal(t, "BTCUSD", od.Pair.String())
	assert.Nil(t, od.Trades)

	om.Trades = append(om.Trades, TradeHistory{TID: "1"}, TradeHistory{TID: "2"})
	err = od.UpdateOrderFromDetail(om)
	require.NoError(t, err)
	assert.Len(t, od.Trades, 2)
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
	require.NoError(t, err)
	assert.NotEqual(t, leet, od.Trades[0].Exchange, "Should not be able to update exchange from update")
	assert.Equal(t, 1337.0, od.Trades[0].Price)
	assert.Equal(t, 1337.0, od.Trades[0].Fee)
	assert.True(t, od.Trades[0].IsMaker)
	assert.Equal(t, updated, od.Trades[0].Timestamp)
	assert.Equal(t, leet, od.Trades[0].Description)
	assert.Equal(t, UnknownSide, od.Trades[0].Side)
	assert.Equal(t, UnknownType, od.Trades[0].Type)
	assert.Equal(t, 1337.0, od.Trades[0].Amount)

	id, err = uuid.NewV4()
	require.NoError(t, err)

	om = &Detail{
		InternalOrderID: id,
	}

	err = od.UpdateOrderFromDetail(om)
	require.NoError(t, err)
	assert.NotEqual(t, id, od.InternalOrderID, "Should not be able to update the internal order ID after initialization")
}

func TestClassificationError_Error(t *testing.T) {
	class := ClassificationError{OrderID: "1337", Exchange: "test", Err: errors.New("test error")}
	require.Equal(t, "Exchange test: OrderID: 1337 classification error: test error", class.Error())
	class.OrderID = ""
	assert.Equal(t, "Exchange test: classification error: test error", class.Error())
}

func TestValidationOnOrderTypes(t *testing.T) {
	var cancelMe *Cancel
	require.ErrorIs(t, cancelMe.Validate(), ErrCancelOrderIsNil)

	cancelMe = new(Cancel)
	err := cancelMe.Validate()
	assert.NoError(t, err)

	err = cancelMe.Validate(cancelMe.PairAssetRequired())
	assert.ErrorIs(t, err, ErrPairIsEmpty)

	cancelMe.Pair = currency.NewBTCUSDT()
	err = cancelMe.Validate(cancelMe.PairAssetRequired())
	assert.ErrorIs(t, err, ErrAssetNotSet)

	cancelMe.AssetType = asset.Spot
	err = cancelMe.Validate(cancelMe.PairAssetRequired())
	assert.NoError(t, err)
	require.Error(t, cancelMe.Validate(cancelMe.StandardCancel()))

	require.NoError(t, cancelMe.Validate(validate.Check(func() error {
		return nil
	})))
	cancelMe.OrderID = "1337"
	require.NoError(t, cancelMe.Validate(cancelMe.StandardCancel()))

	var getOrders *MultiOrderRequest
	err = getOrders.Validate()
	require.ErrorIs(t, err, ErrGetOrdersRequestIsNil)

	getOrders = new(MultiOrderRequest)
	err = getOrders.Validate()
	require.ErrorIs(t, err, asset.ErrNotSupported)

	getOrders.AssetType = asset.Spot
	err = getOrders.Validate()
	require.ErrorIs(t, err, ErrSideIsInvalid)

	getOrders.Side = AnySide
	err = getOrders.Validate()
	require.ErrorIs(t, err, errUnrecognisedOrderType)

	errTestError := errors.New("test error")
	getOrders.Type = AnyType
	err = getOrders.Validate(validate.Check(func() error {
		return errTestError
	}))
	require.ErrorIs(t, err, errTestError)

	err = getOrders.Validate(validate.Check(func() error {
		return nil
	}))
	require.NoError(t, err)

	var modifyOrder *Modify
	require.ErrorIs(t, modifyOrder.Validate(), ErrModifyOrderIsNil)

	modifyOrder = new(Modify)
	require.ErrorIs(t, modifyOrder.Validate(), ErrPairIsEmpty)

	modifyOrder.Pair = currency.NewBTCUSD()
	require.ErrorIs(t, modifyOrder.Validate(), ErrAssetNotSet)

	modifyOrder.AssetType = asset.Spot
	require.ErrorIs(t, modifyOrder.Validate(), ErrOrderIDNotSet)

	modifyOrder.ClientOrderID = "1337"
	require.NoError(t, modifyOrder.Validate())
	require.Error(t, modifyOrder.Validate(validate.Check(func() error { return errors.New("this must error") })))
	require.NoError(t, modifyOrder.Validate(validate.Check(func() error { return nil })))
}

func TestMatchFilter(t *testing.T) {
	t.Parallel()
	id := uuid.Must(uuid.NewV4())

	assert.True(t, new(Detail).MatchFilter(&Filter{}), "an empty filter should match an empty order")
	assert.True(t, (&Detail{Exchange: "E", OrderID: "A", Side: Sell, Pair: currency.NewBTCUSD()}).MatchFilter(&Filter{}), "an empty filter should match any order")

	tests := []struct {
		description string
		filter      Filter
		order       Detail
		result      bool
	}{
		{"Exchange ✓", Filter{Exchange: "A"}, Detail{Exchange: "A"}, true},
		{"Exchange 𐄂", Filter{Exchange: "A"}, Detail{Exchange: "B"}, false},
		{"Exchange Empty", Filter{Exchange: "A"}, Detail{}, false},
		{"InternalOrderID ✓", Filter{InternalOrderID: id}, Detail{InternalOrderID: id}, true},
		{"InternalOrderID 𐄂", Filter{InternalOrderID: id}, Detail{InternalOrderID: uuid.Must(uuid.NewV4())}, false},
		{"InternalOrderID Empty", Filter{InternalOrderID: id}, Detail{}, false},
		{"OrderID ✓", Filter{OrderID: "A"}, Detail{OrderID: "A"}, true},
		{"OrderID 𐄂", Filter{OrderID: "A"}, Detail{OrderID: "B"}, false},
		{"OrderID Empty", Filter{OrderID: "A"}, Detail{}, false},
		{"ClientOrderID ✓", Filter{ClientOrderID: "A"}, Detail{ClientOrderID: "A"}, true},
		{"ClientOrderID 𐄂", Filter{ClientOrderID: "A"}, Detail{ClientOrderID: "B"}, false},
		{"ClientOrderID Empty", Filter{ClientOrderID: "A"}, Detail{}, false},
		{"ClientID ✓", Filter{ClientID: "A"}, Detail{ClientID: "A"}, true},
		{"ClientID 𐄂", Filter{ClientID: "A"}, Detail{ClientID: "B"}, false},
		{"ClientID Empty", Filter{ClientID: "A"}, Detail{}, false},
		{"AnySide Buy", Filter{Side: AnySide}, Detail{Side: Buy}, true},
		{"AnySide Sell", Filter{Side: AnySide}, Detail{Side: Sell}, true},
		{"AnySide Empty", Filter{Side: AnySide}, Detail{}, true},
		{"Side ✓", Filter{Side: Buy}, Detail{Side: Buy}, true},
		{"Side 𐄂", Filter{Side: Buy}, Detail{Side: Sell}, false},
		{"Side Empty", Filter{Side: Buy}, Detail{}, false},
		{"Status ✓", Filter{Status: Open}, Detail{Status: Open}, true},
		{"Status 𐄂", Filter{Status: Open}, Detail{Status: New}, false},
		{"Status Empty", Filter{Status: Open}, Detail{}, false},
		{"AssetType ✓", Filter{AssetType: asset.Spot}, Detail{AssetType: asset.Spot}, true},
		{"AssetType 𐄂", Filter{AssetType: asset.Spot}, Detail{AssetType: asset.Index}, false},
		{"AssetType Empty", Filter{AssetType: asset.Spot}, Detail{}, false},
		{"Pair ✓", Filter{Pair: currency.NewBTCUSDT()}, Detail{Pair: currency.NewBTCUSDT()}, true},
		{"Pair 𐄂", Filter{Pair: currency.NewBTCUSDT()}, Detail{Pair: currency.NewBTCUSD()}, false},
		{"Pair Empty", Filter{Pair: currency.NewBTCUSDT()}, Detail{}, false},
		{"AccountID ✓", Filter{AccountID: "A"}, Detail{AccountID: "A"}, true},
		{"AccountID 𐄂", Filter{AccountID: "A"}, Detail{AccountID: "B"}, false},
		{"AccountID Empty", Filter{AccountID: "A"}, Detail{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.result, tt.order.MatchFilter(&tt.filter), "MatchFilter must return correctly")
		})
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
		assert.Equalf(t, tt.expectedResult, tt.o.IsActive(), "amountTests[%v] failed", num)
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
		require.Equalf(t, tt.expectedResult, tt.o.IsActive(), "statusTests[%v] failed", num)
	}
}

var activeBenchmark = Detail{Status: Pending, Amount: 1}

// 610732089	         2.414 ns/op	       0 B/op	       0 allocs/op // PREV
// 1000000000	         1.188 ns/op	       0 B/op	       0 allocs/op // CURRENT
func BenchmarkIsActive(b *testing.B) {
	for b.Loop() {
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
		assert.Equalf(t, tt.expectedResult, tt.o.IsInactive(), "amountTests[%v] failed", num)
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
		assert.Equalf(t, tt.expectedResult, tt.o.IsInactive(), "statusTests[%v] failed", num)
	}
}

var inactiveBenchmark = Detail{Status: Closed, Amount: 1}

// 1000000000	         1.043 ns/op	       0 B/op	       0 allocs/op // CURRENT
func BenchmarkIsInactive(b *testing.B) {
	for b.Loop() {
		require.True(b, inactiveBenchmark.IsInactive())
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
		t.Run(fmt.Sprintf("TEST CASE: %d", num), func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, tt.expectedResult, tt.o.WasOrderPlaced(), "statusTests[%v] failed", num)
		})
	}
}

func TestGenerateInternalOrderID(t *testing.T) {
	id, err := uuid.NewV4()
	assert.NoError(t, err)
	od := Detail{
		InternalOrderID: id,
	}
	od.GenerateInternalOrderID()
	assert.Equal(t, id, od.InternalOrderID, "Should not be able to generate a new internal order ID")

	od = Detail{}
	od.GenerateInternalOrderID()
	assert.False(t, od.InternalOrderID.IsNil(), "unable to generate internal order ID")
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
		assert.Truef(t, reflect.DeepEqual(d[i], r), "[%d] Copy does not contain same elements, expected: %v\ngot:%v", i, d[i], r)
		if len(d[i].Trades) > 0 {
			assert.NotSamef(t, &d[i].Trades[0], &r.Trades[0], "[%d]Trades point to the same data elements", i)
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
		assert.Truef(t, reflect.DeepEqual(d[i], *r), "[%d] Copy does not contain same elements, expected: %v\ngot:%v", i, d[i], r)
		if len(d[i].Trades) > 0 {
			assert.NotSamef(t, &d[i].Trades[0], &r.Trades[0], "[%d]Trades point to the same data elements", i)
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
		assert.Truef(t, reflect.DeepEqual(*sliceCopy[i], *d[i]), "[%d] Copy does not contain same elements, expected: %v\ngot:%v", i, sliceCopy[i], d[i])
		if len(sliceCopy[i].Trades) > 0 {
			assert.NotSamef(t, &sliceCopy[i].Trades[0], &d[i].Trades[0], "[%d]Trades point to the same data elements", i)
		}
	}
}

func TestDeriveModify(t *testing.T) {
	t.Parallel()
	var o *Detail
	_, err := o.DeriveModify()
	require.ErrorIs(t, err, errOrderDetailIsNil)

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
	require.NoError(t, err)
	require.NotNil(t, mod)

	exp := &Modify{
		Exchange:      "wow",
		OrderID:       "wow2",
		ClientOrderID: "wow3",
		Type:          Market,
		Side:          Long,
		AssetType:     asset.Futures,
		Pair:          pair,
	}
	assert.Equal(t, exp, mod)
}

func TestDeriveModifyResponse(t *testing.T) {
	t.Parallel()
	var mod *Modify
	_, err := mod.DeriveModifyResponse()
	require.ErrorIs(t, err, errOrderDetailIsNil)

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
	require.NoError(t, err, "DeriveModifyResponse must not error")
	require.NotNil(t, modresp)

	exp := &ModifyResponse{
		Exchange:      "wow",
		OrderID:       "wow2",
		ClientOrderID: "wow3",
		Type:          Market,
		Side:          Long,
		AssetType:     asset.Futures,
		Pair:          pair,
	}
	assert.Equal(t, exp, modresp)
}

func TestDeriveCancel(t *testing.T) {
	t.Parallel()
	var o *Detail
	_, err := o.DeriveCancel()
	require.ErrorIs(t, err, errOrderDetailIsNil)

	pair := currency.NewPair(currency.BTC, currency.AUD)

	o = &Detail{
		Exchange:      "wow",
		OrderID:       "wow1",
		AccountID:     "wow2",
		ClientID:      "wow3",
		ClientOrderID: "wow4",
		Type:          Market,
		Side:          Long,
		Pair:          pair,
		AssetType:     asset.Futures,
	}
	cancel, err := o.DeriveCancel()
	require.NoError(t, err)
	assert.Equal(t, "wow", cancel.Exchange, "DeriveCancel should set Exchange correctly")
	assert.Equal(t, "wow1", cancel.OrderID, "DeriveCancel should set OrderID correctly")
	assert.Equal(t, "wow2", cancel.AccountID, "DeriveCancel should set AccountID correctly")
	assert.Equal(t, "wow3", cancel.ClientID, "DeriveCancel should set ClientID correctly")
	assert.Equal(t, "wow4", cancel.ClientOrderID, "DeriveCancel should set ClientOrderID correctly")
	assert.Equal(t, Market, cancel.Type, "DeriveCancel should set Type correctly")
	assert.Equal(t, Long, cancel.Side, "DeriveCancel should set Side correctly")
	assert.True(t, pair.Equal(cancel.Pair), "DeriveCancel should set Pair correctly")
	assert.Equal(t, asset.Futures, cancel.AssetType, "DeriveCancel should set AssetType correctly")
}

func TestGetOrdersRequest_Filter(t *testing.T) {
	request := new(MultiOrderRequest)
	request.AssetType = asset.Spot
	request.Type = AnyType
	request.Side = AnySide

	orders := []Detail{
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
	require.Len(t, shinyAndClean, 16)

	for x := range shinyAndClean {
		require.Equal(t, strconv.FormatInt(int64(x), 10), shinyAndClean[x].OrderID)
	}

	request.Pairs = []currency.Pair{btcltc}

	// Kicks off time error
	request.EndTime = time.Unix(1336, 0)
	request.StartTime = time.Unix(1337, 0)

	shinyAndClean = request.Filter("test", orders)
	require.Len(t, shinyAndClean, 8)

	for x := range shinyAndClean {
		require.Equal(t, strconv.FormatInt(int64(x)+8, 10), shinyAndClean[x].OrderID)
	}
}

func TestIsValidOrderSubmissionSide(t *testing.T) {
	t.Parallel()
	assert.False(t, IsValidOrderSubmissionSide(UnknownSide))
	assert.True(t, IsValidOrderSubmissionSide(Buy))
	assert.False(t, IsValidOrderSubmissionSide(CouldNotBuy))
}

func TestAdjustBaseAmount(t *testing.T) {
	t.Parallel()

	var s *SubmitResponse
	err := s.AdjustBaseAmount(0)
	require.ErrorIs(t, err, errOrderSubmitResponseIsNil)

	s = &SubmitResponse{}
	err = s.AdjustBaseAmount(0)
	require.ErrorIs(t, err, errAmountIsZero)

	s.Amount = 1.7777777777
	err = s.AdjustBaseAmount(1.7777777777)
	require.NoError(t, err)
	require.Equal(t, 1.7777777777, s.Amount)

	s.Amount = 1.7777777777
	err = s.AdjustBaseAmount(1.777)
	require.NoError(t, err)
	assert.Equal(t, 1.777, s.Amount)
}

func TestAdjustQuoteAmount(t *testing.T) {
	t.Parallel()

	var s *SubmitResponse
	err := s.AdjustQuoteAmount(0)
	require.ErrorIs(t, err, errOrderSubmitResponseIsNil)

	s = &SubmitResponse{}
	err = s.AdjustQuoteAmount(0)
	require.ErrorIs(t, err, errAmountIsZero)

	s.QuoteAmount = 5.222222222222
	err = s.AdjustQuoteAmount(5.222222222222)
	require.NoError(t, err)
	require.Equal(t, 5.222222222222, s.QuoteAmount)

	s.QuoteAmount = 5.222222222222
	err = s.AdjustQuoteAmount(5.22222222)
	require.NoError(t, err)
	assert.Equal(t, 5.22222222, s.QuoteAmount)
}

func TestSideUnmarshal(t *testing.T) {
	t.Parallel()
	var s Side
	assert.NoError(t, s.UnmarshalJSON([]byte(`"SELL"`)), "Quoted valid side okay")
	assert.Equal(t, Sell, s, "Correctly set order Side")
	assert.ErrorIs(t, s.UnmarshalJSON([]byte(`"STEAL"`)), ErrSideIsInvalid, "Quoted invalid side errors")
	var jErr *json.UnmarshalTypeError
	assert.ErrorAs(t, s.UnmarshalJSON([]byte(`14`)), &jErr, "non-string valid json is rejected")
}

func TestSideMarshalJSON(t *testing.T) {
	t.Parallel()
	b, err := Buy.MarshalJSON()
	assert.NoError(t, err)
	assert.Equal(t, `"BUY"`, string(b))
	b, err = UnknownSide.MarshalJSON()
	assert.NoError(t, err)
	assert.Equal(t, `"UNKNOWN"`, string(b))
}

func TestGetTradeAmount(t *testing.T) {
	t.Parallel()
	var s *Submit
	require.Zero(t, s.GetTradeAmount(protocol.TradingRequirements{}))
	baseAmount := 420.0
	quoteAmount := 69.0
	s = &Submit{Amount: baseAmount, QuoteAmount: quoteAmount}
	// below will default to base amount with nothing set
	require.Equal(t, baseAmount, s.GetTradeAmount(protocol.TradingRequirements{}))
	require.Equal(t, baseAmount, s.GetTradeAmount(protocol.TradingRequirements{SpotMarketOrderAmountPurchaseQuotationOnly: true}))
	s.AssetType = asset.Spot
	s.Type = Market
	s.Side = Buy
	require.Equal(t, quoteAmount, s.GetTradeAmount(protocol.TradingRequirements{SpotMarketOrderAmountPurchaseQuotationOnly: true}))
	require.Equal(t, baseAmount, s.GetTradeAmount(protocol.TradingRequirements{SpotMarketOrderAmountSellBaseOnly: true}))
	s.Side = Sell
	require.Equal(t, baseAmount, s.GetTradeAmount(protocol.TradingRequirements{SpotMarketOrderAmountSellBaseOnly: true}))
}

func TestStringToTrackingMode(t *testing.T) {
	t.Parallel()
	inputs := map[string]TrackingMode{
		"diStance":   Distance,
		"distance":   Distance,
		"Percentage": Percentage,
		"percentage": Percentage,
		"":           UnknownTrackingMode,
	}
	for k, v := range inputs {
		assert.Equal(t, v, StringToTrackingMode(k))
	}
}

func TestTrackingModeString(t *testing.T) {
	t.Parallel()
	inputs := map[TrackingMode]string{
		Distance:            "distance",
		Percentage:          "percentage",
		UnknownTrackingMode: "",
	}
	for k, v := range inputs {
		require.Equal(t, v, k.String())
	}
}

func TestMarshalOrder(t *testing.T) {
	t.Parallel()
	btx := currency.NewBTCUSDT()
	btx.Delimiter = "-"
	orderSubmit := Submit{
		Exchange:   "test",
		Pair:       btx,
		AssetType:  asset.Spot,
		MarginType: margin.Multi,
		Side:       Buy,
		Type:       Market,
		Amount:     1,
		Price:      1000,
	}
	j, err := json.Marshal(orderSubmit)
	require.NoError(t, err, "json.Marshal must not error")
	exp := []byte(`{"Exchange":"test","Type":4,"Side":"BUY","Pair":"BTC-USDT","AssetType":"spot","TimeInForce":"","ReduceOnly":false,"Leverage":0,"Price":1000,"Amount":1,"QuoteAmount":0,"TriggerPrice":0,"TriggerPriceType":0,"ClientID":"","ClientOrderID":"","AutoBorrow":false,"MarginType":"multi","RetrieveFees":false,"RetrieveFeeDelay":0,"RiskManagementModes":{"Mode":"","TakeProfit":{"Enabled":false,"TriggerPriceType":0,"Price":0,"LimitPrice":0,"OrderType":0},"StopLoss":{"Enabled":false,"TriggerPriceType":0,"Price":0,"LimitPrice":0,"OrderType":0},"StopEntry":{"Enabled":false,"TriggerPriceType":0,"Price":0,"LimitPrice":0,"OrderType":0}},"Hidden":false,"Iceberg":false,"TrackingMode":0,"TrackingValue":0}`)
	assert.Equal(t, exp, j)
}
