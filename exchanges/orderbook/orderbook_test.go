package orderbook

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-/gocryptotrader/currency/pair"
)

var Ten = decimal.New(10, 0)
var Hundred = decimal.New(100, 0)
var TwoHundred = decimal.New(200, 0)
var Thousand = decimal.New(1000, 0)

func TestCalculateTotalBids(t *testing.T) {
	t.Parallel()
	currency := pair.NewCurrencyPair("BTC", "USD")
	base := Base{
		Pair:         currency,
		CurrencyPair: currency.Pair().String(),
		Bids:         []Item{{Price: Hundred, Amount: Ten}},
		LastUpdated:  time.Now(),
	}

	a, b := base.CalculateTotalBids()

	if !a.Equal(Ten) && !b.Equal(Thousand) {
		t.Fatal("Test failed. TestCalculateTotalBids expected a = 10 and b = 1000")
	}
}

func TestCalculateTotaAsks(t *testing.T) {
	t.Parallel()
	currency := pair.NewCurrencyPair("BTC", "USD")
	base := Base{
		Pair:         currency,
		CurrencyPair: currency.Pair().String(),
		Asks:         []Item{{Price: Hundred, Amount: Ten}},
		LastUpdated:  time.Now(),
	}

	a, b := base.CalculateTotalAsks()

	if !a.Equal(Ten) && !b.Equal(Thousand) {
		t.Fatal("Test failed. TestCalculateTotalAsks expected a = 10 and b = 1000")
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()
	currency := pair.NewCurrencyPair("BTC", "USD")
	timeNow := time.Now()
	base := Base{
		Pair:         currency,
		CurrencyPair: currency.Pair().String(),
		Asks:         []Item{{Price: Hundred, Amount: Ten}},
		Bids:         []Item{{Price: TwoHundred, Amount: Ten}},
		LastUpdated:  timeNow,
	}

	asks := []Item{{Price: TwoHundred, Amount: decimal.New(101, 0)}}
	bids := []Item{{Price: decimal.New(201, 0), Amount: Hundred}}
	time.Sleep(time.Millisecond * 50)
	base.Update(bids, asks)

	if !base.LastUpdated.After(timeNow) {
		t.Fatal("test failed. TestUpdate expected LastUpdated to be greater then original time")
	}

	a, b := base.CalculateTotalAsks()
	if !a.Equal(Hundred) && !b.Equal(decimal.New(20200, 0)) {
		t.Fatal("Test failed. TestUpdate expected a = 100 and b = 20100")
	}

	a, b = base.CalculateTotalBids()
	if !a.Equal(Hundred) && !b.Equal(decimal.New(20200, 0)) {
		t.Fatal("Test failed. TestUpdate expected a = 100 and b = 20100")
	}
}

func TestGetOrderbook(t *testing.T) {
	currency := pair.NewCurrencyPair("BTC", "USD")
	base := Base{
		Pair:         currency,
		CurrencyPair: currency.Pair().String(),
		Asks:         []Item{{Price: Hundred, Amount: Ten}},
		Bids:         []Item{{Price: TwoHundred, Amount: Ten}},
	}

	CreateNewOrderbook("Exchange", currency, base, Spot)

	result, err := GetOrderbook("Exchange", currency, Spot)
	if err != nil {
		t.Fatalf("Test failed. TestGetOrderbook failed to get orderbook. Error %s",
			err)
	}

	if result.Pair.Pair() != currency.Pair() {
		t.Fatal("Test failed. TestGetOrderbook failed. Mismatched pairs")
	}

	_, err = GetOrderbook("nonexistent", currency, Spot)
	if err == nil {
		t.Fatal("Test failed. TestGetOrderbook retrieved non-existent orderbook")
	}

	currency.FirstCurrency = "blah"
	_, err = GetOrderbook("Exchange", currency, Spot)
	if err == nil {
		t.Fatal("Test failed. TestGetOrderbook retrieved non-existent orderbook using invalid first currency")
	}

	newCurrency := pair.NewCurrencyPair("BTC", "AUD")
	_, err = GetOrderbook("Exchange", newCurrency, Spot)
	if err == nil {
		t.Fatal("Test failed. TestGetOrderbook retrieved non-existent orderbook using invalid second currency")
	}
}

func TestGetOrderbookByExchange(t *testing.T) {
	currency := pair.NewCurrencyPair("BTC", "USD")
	base := Base{
		Pair:         currency,
		CurrencyPair: currency.Pair().String(),
		Asks:         []Item{{Price: Hundred, Amount: Ten}},
		Bids:         []Item{{Price: TwoHundred, Amount: Ten}},
	}

	CreateNewOrderbook("Exchange", currency, base, Spot)

	_, err := GetOrderbookByExchange("Exchange")
	if err != nil {
		t.Fatalf("Test failed. TestGetOrderbookByExchange failed to get orderbook. Error %s",
			err)
	}

	_, err = GetOrderbookByExchange("nonexistent")
	if err == nil {
		t.Fatal("Test failed. TestGetOrderbookByExchange retrieved non-existent orderbook")
	}
}

func TestFirstCurrencyExists(t *testing.T) {
	currency := pair.NewCurrencyPair("BTC", "AUD")
	base := Base{
		Pair:         currency,
		CurrencyPair: currency.Pair().String(),
		Asks:         []Item{{Price: Hundred, Amount: Ten}},
		Bids:         []Item{{Price: TwoHundred, Amount: Ten}},
	}

	CreateNewOrderbook("Exchange", currency, base, Spot)

	if !FirstCurrencyExists("Exchange", currency.FirstCurrency) {
		t.Fatal("Test failed. TestFirstCurrencyExists expected first currency doesn't exist")
	}

	var item pair.CurrencyItem = "blah"
	if FirstCurrencyExists("Exchange", item) {
		t.Fatal("Test failed. TestFirstCurrencyExists unexpected first currency exists")
	}
}

func TestSecondCurrencyExists(t *testing.T) {
	currency := pair.NewCurrencyPair("BTC", "USD")
	base := Base{
		Pair:         currency,
		CurrencyPair: currency.Pair().String(),
		Asks:         []Item{{Price: Hundred, Amount: Ten}},
		Bids:         []Item{{Price: TwoHundred, Amount: Ten}},
	}

	CreateNewOrderbook("Exchange", currency, base, Spot)

	if !SecondCurrencyExists("Exchange", currency) {
		t.Fatal("Test failed. TestSecondCurrencyExists expected first currency doesn't exist")
	}

	currency.SecondCurrency = "blah"
	if SecondCurrencyExists("Exchange", currency) {
		t.Fatal("Test failed. TestSecondCurrencyExists unexpected first currency exists")
	}
}

func TestCreateNewOrderbook(t *testing.T) {
	currency := pair.NewCurrencyPair("BTC", "USD")
	base := Base{
		Pair:         currency,
		CurrencyPair: currency.Pair().String(),
		Asks:         []Item{{Price: Hundred, Amount: Ten}},
		Bids:         []Item{{Price: TwoHundred, Amount: Ten}},
	}

	CreateNewOrderbook("Exchange", currency, base, Spot)

	result, err := GetOrderbook("Exchange", currency, Spot)
	if err != nil {
		t.Fatal("Test failed. TestCreateNewOrderbook failed to create new orderbook")
	}

	if result.Pair.Pair() != currency.Pair() {
		t.Fatal("Test failed. TestCreateNewOrderbook result pair is incorrect")
	}

	a, b := result.CalculateTotalAsks()
	if !a.Equal(Ten) && !b.Equal(Thousand) {
		t.Fatal("Test failed. TestCreateNewOrderbook CalculateTotalAsks value is incorrect")
	}

	a, b = result.CalculateTotalBids()
	if !a.Equal(Ten) && !b.Equal(decimal.New(2000, 0)) {
		t.Fatal("Test failed. TestCreateNewOrderbook CalculateTotalBids value is incorrect")
	}
}

func TestProcessOrderbook(t *testing.T) {
	Orderbooks = []Orderbook{}
	currency := pair.NewCurrencyPair("BTC", "USD")
	base := Base{
		Pair:         currency,
		CurrencyPair: currency.Pair().String(),
		Asks:         []Item{{Price: Hundred, Amount: Ten}},
		Bids:         []Item{{Price: TwoHundred, Amount: Ten}},
	}

	ProcessOrderbook("Exchange", currency, base, Spot)

	result, err := GetOrderbook("Exchange", currency, Spot)
	if err != nil {
		t.Fatal("Test failed. TestProcessOrderbook failed to create new orderbook")
	}

	if result.Pair.Pair() != currency.Pair() {
		t.Fatal("Test failed. TestProcessOrderbook result pair is incorrect")
	}

	currency = pair.NewCurrencyPair("BTC", "GBP")
	base.Pair = currency
	ProcessOrderbook("Exchange", currency, base, Spot)

	result, err = GetOrderbook("Exchange", currency, Spot)
	if err != nil {
		t.Fatal("Test failed. TestProcessOrderbook failed to retrieve new orderbook")
	}

	if result.Pair.Pair() != currency.Pair() {
		t.Fatal("Test failed. TestProcessOrderbook result pair is incorrect")
	}

	base.Asks = []Item{{Price: TwoHundred, Amount: TwoHundred}}
	ProcessOrderbook("Exchange", currency, base, "monthly")

	result, err = GetOrderbook("Exchange", currency, "monthly")
	if err != nil {
		t.Fatal("Test failed. TestProcessOrderbook failed to retrieve new orderbook")
	}

	a, b := result.CalculateTotalAsks()
	if !a.Equal(TwoHundred) && !b.Equal(decimal.New(40000, 0)) {
		t.Fatal("Test failed. TestProcessOrderbook CalculateTotalsAsks incorrect values")
	}

	base.Bids = []Item{{Price: decimal.New(420, 0), Amount: TwoHundred}}
	ProcessOrderbook("Blah", currency, base, "quarterly")
	result, err = GetOrderbook("Blah", currency, "quarterly")
	if err != nil {
		t.Fatal("Test failed. TestProcessOrderbook failed to create new orderbook")
	}

	if !a.Equal(TwoHundred) && !b.Equal(decimal.New(84000, 0)) {
		t.Fatal("Test failed. TestProcessOrderbook CalculateTotalsBids incorrect values")
	}
}
