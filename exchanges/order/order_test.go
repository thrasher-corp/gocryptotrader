package order

import (
	"strings"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

func TestValidate(t *testing.T) {
	testPair := currency.NewPair(currency.BTC, currency.LTC)
	tester := []struct {
		Pair currency.Pair
		Side
		Type
		Amount      float64
		Price       float64
		ExpectedErr error
	}{
		{
			ExpectedErr: ErrPairIsEmpty,
		}, // empty pair
		{
			Pair:        testPair,
			ExpectedErr: ErrSideIsInvalid,
		}, // valid pair but invalid order side
		{
			Pair:        testPair,
			Side:        Buy,
			ExpectedErr: ErrTypeIsInvalid,
		}, // valid pair and order side but invalid order type
		{
			Pair:        testPair,
			Side:        Sell,
			ExpectedErr: ErrTypeIsInvalid,
		}, // valid pair and order side but invalid order type
		{
			Pair:        testPair,
			Side:        Bid,
			ExpectedErr: ErrTypeIsInvalid,
		}, // valid pair and order side but invalid order type
		{
			Pair:        testPair,
			Side:        Ask,
			ExpectedErr: ErrTypeIsInvalid,
		}, // valid pair and order side but invalid order type
		{
			Pair:        testPair,
			Side:        Ask,
			Type:        Market,
			ExpectedErr: ErrAmountIsInvalid,
		}, // valid pair, order side, type but invalid amount
		{
			Pair:        testPair,
			Side:        Ask,
			Type:        Limit,
			Amount:      1,
			ExpectedErr: ErrPriceMustBeSetIfLimitOrder,
		}, // valid pair, order side, type, amount but invalid price
		{
			Pair:        testPair,
			Side:        Ask,
			Type:        Limit,
			Amount:      1,
			Price:       1000,
			ExpectedErr: nil,
		}, // valid order!
	}

	for x := range tester {
		s := Submit{
			Pair:      tester[x].Pair,
			OrderSide: tester[x].Side,
			OrderType: tester[x].Type,
			Amount:    tester[x].Amount,
			Price:     tester[x].Price,
		}
		if err := s.Validate(); err != tester[x].ExpectedErr {
			t.Errorf("Unexpected result. Got: %s, want: %s", err, tester[x].ExpectedErr)
		}
	}
}

func TestOrderSides(t *testing.T) {
	t.Parallel()

	var os = Buy
	if os.String() != "BUY" {
		t.Errorf("unexpected string %s", os.String())
	}

	if os.Lower() != "buy" {
		t.Errorf("unexpected string %s", os.String())
	}
}

func TestOrderTypes(t *testing.T) {
	t.Parallel()

	var ot Type = "Mo'Money"

	if ot.String() != "Mo'Money" {
		t.Errorf("unexpected string %s", ot.String())
	}

	if ot.Lower() != "mo'money" {
		t.Errorf("unexpected string %s", ot.Lower())
	}
}

func TestFilterOrdersByType(t *testing.T) {
	t.Parallel()

	var orders = []Detail{
		{
			OrderType: ImmediateOrCancel,
		},
		{
			OrderType: Limit,
		},
	}

	FilterOrdersByType(&orders, AnyType)
	if len(orders) != 2 {
		t.Errorf("Orders failed to be filtered. Expected %v, received %v", 2, len(orders))
	}

	FilterOrdersByType(&orders, Limit)
	if len(orders) != 1 {
		t.Errorf("Orders failed to be filtered. Expected %v, received %v", 1, len(orders))
	}

	FilterOrdersByType(&orders, Stop)
	if len(orders) != 0 {
		t.Errorf("Orders failed to be filtered. Expected %v, received %v", 0, len(orders))
	}
}

func TestFilterOrdersBySide(t *testing.T) {
	t.Parallel()

	var orders = []Detail{
		{
			OrderSide: Buy,
		},
		{
			OrderSide: Sell,
		},
		{},
	}

	FilterOrdersBySide(&orders, AnySide)
	if len(orders) != 3 {
		t.Errorf("Orders failed to be filtered. Expected %v, received %v", 3, len(orders))
	}

	FilterOrdersBySide(&orders, Buy)
	if len(orders) != 1 {
		t.Errorf("Orders failed to be filtered. Expected %v, received %v", 1, len(orders))
	}

	FilterOrdersBySide(&orders, Sell)
	if len(orders) != 0 {
		t.Errorf("Orders failed to be filtered. Expected %v, received %v", 0, len(orders))
	}
}

func TestFilterOrdersByTickRange(t *testing.T) {
	t.Parallel()

	var orders = []Detail{
		{
			OrderDate: time.Unix(100, 0),
		},
		{
			OrderDate: time.Unix(110, 0),
		},
		{
			OrderDate: time.Unix(111, 0),
		},
	}

	FilterOrdersByTickRange(&orders, time.Unix(0, 0), time.Unix(0, 0))
	if len(orders) != 3 {
		t.Errorf("Orders failed to be filtered. Expected %v, received %v", 3, len(orders))
	}

	FilterOrdersByTickRange(&orders, time.Unix(100, 0), time.Unix(111, 0))
	if len(orders) != 3 {
		t.Errorf("Orders failed to be filtered. Expected %v, received %v", 3, len(orders))
	}

	FilterOrdersByTickRange(&orders, time.Unix(101, 0), time.Unix(111, 0))
	if len(orders) != 2 {
		t.Errorf("Orders failed to be filtered. Expected %v, received %v", 2, len(orders))
	}

	FilterOrdersByTickRange(&orders, time.Unix(200, 0), time.Unix(300, 0))
	if len(orders) != 0 {
		t.Errorf("Orders failed to be filtered. Expected %v, received %v", 0, len(orders))
	}
}

func TestFilterOrdersByCurrencies(t *testing.T) {
	t.Parallel()

	var orders = []Detail{
		{
			CurrencyPair: currency.NewPair(currency.BTC, currency.USD),
		},
		{
			CurrencyPair: currency.NewPair(currency.LTC, currency.EUR),
		},
		{
			CurrencyPair: currency.NewPair(currency.DOGE, currency.RUB),
		},
	}

	currencies := []currency.Pair{currency.NewPair(currency.BTC, currency.USD),
		currency.NewPair(currency.LTC, currency.EUR),
		currency.NewPair(currency.DOGE, currency.RUB)}
	FilterOrdersByCurrencies(&orders, currencies)
	if len(orders) != 3 {
		t.Errorf("Orders failed to be filtered. Expected %v, received %v", 3, len(orders))
	}

	currencies = []currency.Pair{currency.NewPair(currency.BTC, currency.USD),
		currency.NewPair(currency.LTC, currency.EUR)}
	FilterOrdersByCurrencies(&orders, currencies)
	if len(orders) != 2 {
		t.Errorf("Orders failed to be filtered. Expected %v, received %v", 2, len(orders))
	}

	currencies = []currency.Pair{currency.NewPair(currency.BTC, currency.USD)}
	FilterOrdersByCurrencies(&orders, currencies)
	if len(orders) != 1 {
		t.Errorf("Orders failed to be filtered. Expected %v, received %v", 1, len(orders))
	}

	currencies = []currency.Pair{}
	FilterOrdersByCurrencies(&orders, currencies)
	if len(orders) != 1 {
		t.Errorf("Orders failed to be filtered. Expected %v, received %v", 1, len(orders))
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
			OrderDate: time.Unix(0, 0),
		}, {
			OrderDate: time.Unix(1, 0),
		}, {
			OrderDate: time.Unix(2, 0),
		},
	}

	SortOrdersByDate(&orders, false)
	if orders[0].OrderDate.Unix() != time.Unix(0, 0).Unix() {
		t.Errorf("Expected: '%v', received: '%v'",
			time.Unix(0, 0).Unix(),
			orders[0].OrderDate.Unix())
	}

	SortOrdersByDate(&orders, true)
	if orders[0].OrderDate.Unix() != time.Unix(2, 0).Unix() {
		t.Errorf("Expected: '%v', received: '%v'",
			time.Unix(2, 0).Unix(),
			orders[0].OrderDate.Unix())
	}
}

func TestSortOrdersByCurrency(t *testing.T) {
	t.Parallel()

	orders := []Detail{
		{
			CurrencyPair: currency.NewPairWithDelimiter(currency.BTC.String(),
				currency.USD.String(),
				"-"),
		}, {
			CurrencyPair: currency.NewPairWithDelimiter(currency.DOGE.String(),
				currency.USD.String(),
				"-"),
		}, {
			CurrencyPair: currency.NewPairWithDelimiter(currency.BTC.String(),
				currency.RUB.String(),
				"-"),
		}, {
			CurrencyPair: currency.NewPairWithDelimiter(currency.LTC.String(),
				currency.EUR.String(),
				"-"),
		}, {
			CurrencyPair: currency.NewPairWithDelimiter(currency.LTC.String(),
				currency.AUD.String(),
				"-"),
		},
	}

	SortOrdersByCurrency(&orders, false)
	if orders[0].CurrencyPair.String() != currency.BTC.String()+"-"+currency.RUB.String() {
		t.Errorf("Expected: '%v', received: '%v'",
			currency.BTC.String()+"-"+currency.RUB.String(),
			orders[0].CurrencyPair.String())
	}

	SortOrdersByCurrency(&orders, true)
	if orders[0].CurrencyPair.String() != currency.LTC.String()+"-"+currency.EUR.String() {
		t.Errorf("Expected: '%v', received: '%v'",
			currency.LTC.String()+"-"+currency.EUR.String(),
			orders[0].CurrencyPair.String())
	}
}

func TestSortOrdersByOrderSide(t *testing.T) {
	t.Parallel()

	orders := []Detail{
		{
			OrderSide: Buy,
		}, {
			OrderSide: Sell,
		}, {
			OrderSide: Sell,
		}, {
			OrderSide: Buy,
		},
	}

	SortOrdersBySide(&orders, false)
	if !strings.EqualFold(orders[0].OrderSide.String(), Buy.String()) {
		t.Errorf("Expected: '%v', received: '%v'",
			Buy,
			orders[0].OrderSide)
	}

	SortOrdersBySide(&orders, true)
	if !strings.EqualFold(orders[0].OrderSide.String(), Sell.String()) {
		t.Errorf("Expected: '%v', received: '%v'",
			Sell,
			orders[0].OrderSide)
	}
}

func TestSortOrdersByOrderType(t *testing.T) {
	t.Parallel()

	orders := []Detail{
		{
			OrderType: Market,
		}, {
			OrderType: Limit,
		}, {
			OrderType: ImmediateOrCancel,
		}, {
			OrderType: TrailingStop,
		},
	}

	SortOrdersByType(&orders, false)
	if !strings.EqualFold(orders[0].OrderType.String(), ImmediateOrCancel.String()) {
		t.Errorf("Expected: '%v', received: '%v'",
			ImmediateOrCancel,
			orders[0].OrderType)
	}

	SortOrdersByType(&orders, true)
	if !strings.EqualFold(orders[0].OrderType.String(), TrailingStop.String()) {
		t.Errorf("Expected: '%v', received: '%v'",
			TrailingStop,
			orders[0].OrderType)
	}
}
