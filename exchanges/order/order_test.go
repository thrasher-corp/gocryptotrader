package order

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/validate"
)

var errValidationCheckFailed = errors.New("validation check failed")

func TestValidate(t *testing.T) {
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
			ExpectedErr: ErrPairIsEmpty,
			Submit:      &Submit{},
		}, // empty pair
		{

			ExpectedErr: ErrAssetNotSet,
			Submit:      &Submit{Pair: testPair},
		}, // valid pair but invalid asset
		{

			ExpectedErr: ErrSideIsInvalid,
			Submit:      &Submit{Pair: testPair, AssetType: asset.Spot},
		}, // valid pair but invalid order side
		{
			ExpectedErr: ErrTypeIsInvalid,
			Submit: &Submit{Pair: testPair,
				Side:      Buy,
				AssetType: asset.Spot},
		}, // valid pair and order side but invalid order type
		{
			ExpectedErr: ErrTypeIsInvalid,
			Submit: &Submit{Pair: testPair,
				Side:      Sell,
				AssetType: asset.Spot},
		}, // valid pair and order side but invalid order type
		{
			ExpectedErr: ErrTypeIsInvalid,
			Submit: &Submit{Pair: testPair,
				Side:      Bid,
				AssetType: asset.Spot},
		}, // valid pair and order side but invalid order type
		{
			ExpectedErr: ErrTypeIsInvalid,
			Submit: &Submit{Pair: testPair,
				Side:      Ask,
				AssetType: asset.Spot},
		}, // valid pair and order side but invalid order type
		{
			ExpectedErr: ErrAmountIsInvalid,
			Submit: &Submit{Pair: testPair,
				Side:      Ask,
				Type:      Market,
				AssetType: asset.Spot},
		}, // valid pair, order side, type but invalid amount
		{
			ExpectedErr: ErrPriceMustBeSetIfLimitOrder,
			Submit: &Submit{Pair: testPair,
				Side:      Ask,
				Type:      Limit,
				Amount:    1,
				AssetType: asset.Spot},
		}, // valid pair, order side, type, amount but invalid price
		{
			ExpectedErr: errValidationCheckFailed,
			Submit: &Submit{Pair: testPair,
				Side:      Ask,
				Type:      Limit,
				Amount:    1,
				Price:     1000,
				AssetType: asset.Spot},
			ValidOpts: validate.Check(func() error { return errValidationCheckFailed }),
		}, // custom validation error check
		{
			ExpectedErr: nil,
			Submit: &Submit{Pair: testPair,
				Side:      Ask,
				Type:      Limit,
				Amount:    1,
				Price:     1000,
				AssetType: asset.Spot},
			ValidOpts: validate.Check(func() error { return nil }),
		}, // valid order!
	}

	for x := range tester {
		err := tester[x].Submit.Validate(tester[x].ValidOpts)
		if !errors.Is(err, tester[x].ExpectedErr) {
			t.Errorf("Unexpected result. Got: %v, want: %v", err, tester[x].ExpectedErr)
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
		t.Errorf("unexpected string %s", os.Lower())
	}

	if os.Title() != "Buy" {
		t.Errorf("unexpected string %s", os.Title())
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

	if ot.Title() != "Mo'Money" {
		t.Errorf("unexpected string %s", ot.Title())
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
			Side: Buy,
		},
		{
			Side: Sell,
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

	FilterOrdersByTimeRange(&orders, time.Unix(0, 0), time.Unix(0, 0))
	if len(orders) != 3 {
		t.Errorf("Orders failed to be filtered. Expected %v, received %v", 3, len(orders))
	}

	FilterOrdersByTimeRange(&orders, time.Unix(100, 0), time.Unix(111, 0))
	if len(orders) != 3 {
		t.Errorf("Orders failed to be filtered. Expected %v, received %v", 3, len(orders))
	}

	FilterOrdersByTimeRange(&orders, time.Unix(101, 0), time.Unix(111, 0))
	if len(orders) != 2 {
		t.Errorf("Orders failed to be filtered. Expected %v, received %v", 2, len(orders))
	}

	FilterOrdersByTimeRange(&orders, time.Unix(200, 0), time.Unix(300, 0))
	if len(orders) != 0 {
		t.Errorf("Orders failed to be filtered. Expected %v, received %v", 0, len(orders))
	}
	orders = append(orders, Detail{})
	// test for event no timestamp is set on an order, best to include it
	FilterOrdersByTimeRange(&orders, time.Unix(200, 0), time.Unix(300, 0))
	if len(orders) != 1 {
		t.Errorf("Orders failed to be filtered. Expected %v, received %v", 1, len(orders))
	}
}

func TestFilterOrdersByCurrencies(t *testing.T) {
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

	currencies = []currency.Pair{currency.NewPair(currency.USD, currency.BTC)}
	FilterOrdersByCurrencies(&orders, currencies)
	if len(orders) != 1 {
		t.Errorf("Reverse Orders failed to be filtered. Expected %v, received %v", 1, len(orders))
	}

	currencies = []currency.Pair{}
	FilterOrdersByCurrencies(&orders, currencies)
	if len(orders) != 1 {
		t.Errorf("Orders failed to be filtered. Expected %v, received %v", 1, len(orders))
	}
	currencies = append(currencies, currency.Pair{})
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

var stringsToOrderSide = []struct {
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
	{"any", AnySide, nil},
	{"ANY", AnySide, nil},
	{"aNy", AnySide, nil},
	{"woahMan", Buy, errors.New("woahMan not recognised as order side")},
}

func TestStringToOrderSide(t *testing.T) {
	for i := range stringsToOrderSide {
		testData := &stringsToOrderSide[i]
		t.Run(testData.in, func(t *testing.T) {
			out, err := StringToOrderSide(testData.in)
			if err != nil {
				if err.Error() != testData.err.Error() {
					t.Error("Unexpected error", err)
				}
			} else if out != testData.out {
				t.Errorf("Unexpected output %v. Expected %v", out, testData.out)
			}
		})
	}
}

var stringsToOrderType = []struct {
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
	{"woahMan", UnknownType, errors.New("woahMan not recognised as order type")},
}

func TestStringToOrderType(t *testing.T) {
	for i := range stringsToOrderType {
		testData := &stringsToOrderType[i]
		t.Run(testData.in, func(t *testing.T) {
			out, err := StringToOrderType(testData.in)
			if err != nil {
				if err.Error() != testData.err.Error() {
					t.Error("Unexpected error", err)
				}
			} else if out != testData.out {
				t.Errorf("Unexpected output %v. Expected %v", out, testData.out)
			}
		})
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
	{"woahMan", UnknownStatus, errors.New("woahMan not recognised as order status")},
}

func TestStringToOrderStatus(t *testing.T) {
	for i := range stringsToOrderStatus {
		testData := &stringsToOrderStatus[i]
		t.Run(testData.in, func(t *testing.T) {
			out, err := StringToOrderStatus(testData.in)
			if err != nil {
				if err.Error() != testData.err.Error() {
					t.Error("Unexpected error", err)
				}
			} else if out != testData.out {
				t.Errorf("Unexpected output %v. Expected %v", out, testData.out)
			}
		})
	}
}

func TestUpdateOrderFromModify(t *testing.T) {
	var leet = "1337"
	od := Detail{
		ImmediateOrCancel: false,
		HiddenOrder:       false,
		FillOrKill:        false,
		PostOnly:          false,
		Leverage:          0,
		Price:             0,
		Amount:            0,
		LimitPriceUpper:   0,
		LimitPriceLower:   0,
		TriggerPrice:      0,
		TargetAmount:      0,
		ExecutedAmount:    0,
		RemainingAmount:   0,
		Fee:               0,
		Exchange:          "",
		ID:                "1",
		AccountID:         "",
		ClientID:          "",
		WalletAddress:     "",
		Type:              "",
		Side:              "",
		Status:            "",
		AssetType:         "",
		Date:              time.Time{},
		LastUpdated:       time.Time{},
		Pair:              currency.Pair{},
		Trades:            nil,
	}
	updated := time.Now()

	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	om := Modify{
		ImmediateOrCancel: true,
		HiddenOrder:       true,
		FillOrKill:        true,
		PostOnly:          true,
		Leverage:          1.0,
		Price:             1,
		Amount:            1,
		LimitPriceUpper:   1,
		LimitPriceLower:   1,
		TriggerPrice:      1,
		TargetAmount:      1,
		ExecutedAmount:    1,
		RemainingAmount:   1,
		Fee:               1,
		Exchange:          "1",
		InternalOrderID:   "1",
		ID:                "1",
		AccountID:         "1",
		ClientID:          "1",
		WalletAddress:     "1",
		Type:              "1",
		Side:              "1",
		Status:            "1",
		AssetType:         "1",
		LastUpdated:       updated,
		Pair:              pair,
		Trades:            []TradeHistory{},
	}

	od.UpdateOrderFromModify(&om)
	if od.InternalOrderID == "1" {
		t.Error("Should not be able to update the internal order ID")
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
	if od.TargetAmount != 1 {
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
	if od.Exchange != "" {
		t.Error("Should not be able to update exchange via modify")
	}
	if od.ID != "1" {
		t.Error("Failed to update")
	}
	if od.ClientID != "1" {
		t.Error("Failed to update")
	}
	if od.WalletAddress != "1" {
		t.Error("Failed to update")
	}
	if od.Type != "1" {
		t.Error("Failed to update")
	}
	if od.Side != "1" {
		t.Error("Failed to update")
	}
	if od.Status != "1" {
		t.Error("Failed to update")
	}
	if od.AssetType != "1" {
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
	od.UpdateOrderFromModify(&om)
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
	od.UpdateOrderFromModify(&om)
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
}

func TestUpdateOrderFromDetail(t *testing.T) {
	var leet = "1337"
	od := Detail{
		ImmediateOrCancel: false,
		HiddenOrder:       false,
		FillOrKill:        false,
		PostOnly:          false,
		Leverage:          0,
		Price:             0,
		Amount:            0,
		LimitPriceUpper:   0,
		LimitPriceLower:   0,
		TriggerPrice:      0,
		TargetAmount:      0,
		ExecutedAmount:    0,
		RemainingAmount:   0,
		Fee:               0,
		Exchange:          "test",
		ID:                "1",
		AccountID:         "",
		ClientID:          "",
		WalletAddress:     "",
		Type:              "",
		Side:              "",
		Status:            "",
		AssetType:         "",
		Date:              time.Time{},
		LastUpdated:       time.Time{},
		Pair:              currency.Pair{},
		Trades:            nil,
	}
	updated := time.Now()

	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	om := Detail{
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
		TargetAmount:      1,
		ExecutedAmount:    1,
		RemainingAmount:   1,
		Fee:               1,
		Exchange:          "1",
		InternalOrderID:   "1",
		ID:                "1",
		AccountID:         "1",
		ClientID:          "1",
		WalletAddress:     "1",
		Type:              "1",
		Side:              "1",
		Status:            "1",
		AssetType:         "1",
		LastUpdated:       updated,
		Pair:              pair,
		Trades:            []TradeHistory{},
	}

	od.UpdateOrderFromDetail(&om)
	if od.InternalOrderID == "1" {
		t.Error("Should not be able to update the internal order ID")
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
	if od.TargetAmount != 1 {
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
	if od.ID != "1" {
		t.Error("Failed to update")
	}
	if od.ClientID != "1" {
		t.Error("Failed to update")
	}
	if od.WalletAddress != "1" {
		t.Error("Failed to update")
	}
	if od.Type != "1" {
		t.Error("Failed to update")
	}
	if od.Side != "1" {
		t.Error("Failed to update")
	}
	if od.Status != "1" {
		t.Error("Failed to update")
	}
	if od.AssetType != "1" {
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
	od.UpdateOrderFromDetail(&om)
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
	od.UpdateOrderFromDetail(&om)
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
}

func TestClassificationError_Error(t *testing.T) {
	class := ClassificationError{OrderID: "1337", Exchange: "test", Err: errors.New("test error")}
	if class.Error() != "test - OrderID: 1337 classification error: test error" {
		t.Fatal("unexpected output")
	}
	class.OrderID = ""
	if class.Error() != "test - classification error: test error" {
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
	cancelMe.ID = "1337"
	if cancelMe.Validate(cancelMe.StandardCancel()) != nil {
		t.Fatal("should return nil")
	}

	var getOrders *GetOrdersRequest
	if getOrders.Validate() != ErrGetOrdersRequestIsNil {
		t.Fatal("unexpected error")
	}

	getOrders = new(GetOrdersRequest)
	if getOrders.Validate() == nil {
		t.Fatal("should error since assetType hasn't been provided")
	}

	if getOrders.Validate(validate.Check(func() error {
		return errors.New("this should error")
	})) == nil {
		t.Fatal("expected error")
	}

	if getOrders.Validate(validate.Check(func() error {
		return nil
	})) == nil {
		t.Fatal("should output an error since assetType isn't provided")
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
	filters := map[int]Filter{
		0:  {},
		1:  {Exchange: "Binance"},
		2:  {InternalOrderID: "1234"},
		3:  {ID: "2222"},
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
		2:  {InternalOrderID: "1234"},
		3:  {ID: "2222"},
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
		f      Filter
		o      Detail
		expRes bool
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
		if tt.o.MatchFilter(&tt.f) != tt.expRes {
			t.Errorf("tests[%v] failed", num)
		}
	}
}

func TestDetail_Copy(t *testing.T) {
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
