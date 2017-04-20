package ticker

import (
	"reflect"
	"testing"

	"github.com/thrasher-/gocryptotrader/currency/pair"
)

func TestPriceToString(t *testing.T) {
	t.Parallel()

	newPair := pair.NewCurrencyPair("BTC", "USD")
	priceStruct := TickerPrice{
		Pair:         newPair,
		CurrencyPair: newPair.Pair().String(),
		Last:         1200,
		High:         1298,
		Low:          1148,
		Bid:          1195,
		Ask:          1220,
		Volume:       5,
		PriceATH:     1337,
	}

	newTicker := CreateNewTicker("ANX", newPair, priceStruct)

	if newTicker.PriceToString(newPair, "last") != "1200" {
		t.Error("Test Failed - ticker PriceToString last value is incorrect")
	}
	if newTicker.PriceToString(newPair, "high") != "1298" {
		t.Error("Test Failed - ticker PriceToString high value is incorrect")
	}
	if newTicker.PriceToString(newPair, "low") != "1148" {
		t.Error("Test Failed - ticker PriceToString low value is incorrect")
	}
	if newTicker.PriceToString(newPair, "bid") != "1195" {
		t.Error("Test Failed - ticker PriceToString bid value is incorrect")
	}
	if newTicker.PriceToString(newPair, "ask") != "1220" {
		t.Error("Test Failed - ticker PriceToString ask value is incorrect")
	}
	if newTicker.PriceToString(newPair, "volume") != "5" {
		t.Error("Test Failed - ticker PriceToString volume value is incorrect")
	}
	if newTicker.PriceToString(newPair, "ath") != "1337" {
		t.Error("Test Failed - ticker PriceToString ath value is incorrect")
	}
	if newTicker.PriceToString(newPair, "obtuse") != "" {
		t.Error("Test Failed - ticker PriceToString obtuse value is incorrect")
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()

	newPair := pair.NewCurrencyPair("BTC", "USD")
	priceStruct := TickerPrice{
		Pair:         newPair,
		CurrencyPair: newPair.Pair().String(),
		Last:         1200,
		High:         1298,
		Low:          1148,
		Bid:          1195,
		Ask:          1220,
		Volume:       5,
		PriceATH:     1337,
	}

	bitfinexTicker := CreateNewTicker("bitfinex", newPair, priceStruct)
	Tickers = append(Tickers, bitfinexTicker)

	tickerPrice, err := GetTicker("bitfinex", newPair)
	if err != nil {
		t.Errorf("Test Failed - Ticker GetTicker init error: %s", err)
	}
	if tickerPrice.CurrencyPair != "BTCUSD" {
		t.Error("Test Failed - ticker tickerPrice.CurrencyPair value is incorrect")
	}
}

func TestGetTickerByExchange(t *testing.T) {
	t.Parallel()

	newPair := pair.NewCurrencyPair("BTC", "USD")
	priceStruct := TickerPrice{
		Pair:         newPair,
		CurrencyPair: newPair.Pair().String(),
		Last:         1200,
		High:         1298,
		Low:          1148,
		Bid:          1195,
		Ask:          1220,
		Volume:       5,
		PriceATH:     1337,
	}

	anxTicker := CreateNewTicker("ANX", newPair, priceStruct)
	Tickers = append(Tickers, anxTicker)

	tickerPtr, err := GetTickerByExchange("ANX")
	if err != nil {
		t.Errorf("Test Failed - GetTickerByExchange init error: %s", err)
	}
	if tickerPtr.ExchangeName != "ANX" {
		t.Error("Test Failed - GetTickerByExchange ExchangeName value is incorrect")
	}
}

func TestFirstCurrencyExists(t *testing.T) {
	t.Parallel()

	newPair := pair.NewCurrencyPair("BTC", "USD")
	priceStruct := TickerPrice{
		Pair:         newPair,
		CurrencyPair: newPair.Pair().String(),
		Last:         1200,
		High:         1298,
		Low:          1148,
		Bid:          1195,
		Ask:          1220,
		Volume:       5,
		PriceATH:     1337,
	}

	alphaTicker := CreateNewTicker("alphapoint", newPair, priceStruct)
	Tickers = append(Tickers, alphaTicker)

	if !FirstCurrencyExists("alphapoint", "BTC") {
		t.Error("Test Failed - FirstCurrencyExists1 value return is incorrect")
	}
	if FirstCurrencyExists("alphapoint", "CATS") {
		t.Error("Test Failed - FirstCurrencyExists2 value return is incorrect")
	}
}

func TestSecondCurrencyExists(t *testing.T) {
	t.Parallel()

	newPair := pair.NewCurrencyPair("BTC", "USD")
	priceStruct := TickerPrice{
		Pair:         newPair,
		CurrencyPair: newPair.Pair().String(),
		Last:         1200,
		High:         1298,
		Low:          1148,
		Bid:          1195,
		Ask:          1220,
		Volume:       5,
		PriceATH:     1337,
	}

	bitstampTicker := CreateNewTicker("bitstamp", newPair, priceStruct)
	Tickers = append(Tickers, bitstampTicker)

	if !SecondCurrencyExists("bitstamp", newPair) {
		t.Error("Test Failed - SecondCurrencyExists1 value return is incorrect")
	}

	newPair.SecondCurrency = "DOGS"
	if SecondCurrencyExists("bitstamp", newPair) {
		t.Error("Test Failed - SecondCurrencyExists2 value return is incorrect")
	}
}

func TestCreateNewTicker(t *testing.T) {
	t.Parallel()

	newPair := pair.NewCurrencyPair("BTC", "USD")
	priceStruct := TickerPrice{
		Pair:         newPair,
		CurrencyPair: newPair.Pair().String(),
		Last:         1200,
		High:         1298,
		Low:          1148,
		Bid:          1195,
		Ask:          1220,
		Volume:       5,
		PriceATH:     1337,
	}

	newTicker := CreateNewTicker("ANX", newPair, priceStruct)

	if reflect.ValueOf(newTicker).NumField() != 2 {
		t.Error("Test Failed - ticker CreateNewTicker struct change/or updated")
	}
	if reflect.TypeOf(newTicker.ExchangeName).String() != "string" {
		t.Error("Test Failed - ticker CreateNewTicker.ExchangeName value is not a string")
	}
	if newTicker.ExchangeName != "ANX" {
		t.Error("Test Failed - ticker CreateNewTicker.ExchangeName value is not ANX")
	}

	if newTicker.Price["BTC"]["USD"].Pair.Pair().String() != "BTCUSD" {
		t.Error("Test Failed - ticker newTicker.Price[BTC][USD].Pair.Pair().String() value is not expected 'BTCUSD'")
	}
	if reflect.TypeOf(newTicker.Price["BTC"]["USD"].Ask).String() != "float64" {
		t.Error("Test Failed - ticker newTicker.Price[BTC][USD].Ask value is not a float64")
	}
	if reflect.TypeOf(newTicker.Price["BTC"]["USD"].Bid).String() != "float64" {
		t.Error("Test Failed - ticker newTicker.Price[BTC][USD].Bid value is not a float64")
	}
	if reflect.TypeOf(newTicker.Price["BTC"]["USD"].CurrencyPair).String() != "string" {
		t.Error("Test Failed - ticker newTicker.Price[BTC][USD].CurrencyPair value is not a string")
	}
	if reflect.TypeOf(newTicker.Price["BTC"]["USD"].High).String() != "float64" {
		t.Error("Test Failed - ticker newTicker.Price[BTC][USD].High value is not a float64")
	}
	if reflect.TypeOf(newTicker.Price["BTC"]["USD"].Last).String() != "float64" {
		t.Error("Test Failed - ticker newTicker.Price[BTC][USD].Last value is not a float64")
	}
	if reflect.TypeOf(newTicker.Price["BTC"]["USD"].Low).String() != "float64" {
		t.Error("Test Failed - ticker newTicker.Price[BTC][USD].Low value is not a float64")
	}
	if reflect.TypeOf(newTicker.Price["BTC"]["USD"].PriceATH).String() != "float64" {
		t.Error("Test Failed - ticker newTicker.Price[BTC][USD].PriceATH value is not a float64")
	}
	if reflect.TypeOf(newTicker.Price["BTC"]["USD"].Volume).String() != "float64" {
		t.Error("Test Failed - ticker newTicker.Price[BTC][USD].Volume value is not a float64")
	}
}

func TestProcessTicker(t *testing.T) { //non-appending function to tickers
	t.Parallel()

	newPair := pair.NewCurrencyPair("BTC", "USD")
	priceStruct := TickerPrice{
		Pair:         newPair,
		CurrencyPair: newPair.Pair().String(),
		Last:         1200,
		High:         1298,
		Low:          1148,
		Bid:          1195,
		Ask:          1220,
		Volume:       5,
		PriceATH:     1337,
	}

	ProcessTicker("btcc", newPair, priceStruct)
}
