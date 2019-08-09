package ticker

import (
	"math/rand"
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func TestPriceToString(t *testing.T) {
	newPair := currency.NewPairFromStrings("BTC", "USD")
	priceStruct := Price{
		Pair:     newPair,
		Last:     1200,
		High:     1298,
		Low:      1148,
		Bid:      1195,
		Ask:      1220,
		Volume:   5,
		PriceATH: 1337,
	}

	newTicker := CreateNewTicker("ANX", &priceStruct, asset.Spot)

	if newTicker.PriceToString(newPair, "last", asset.Spot) != "1200" {
		t.Error("Test Failed - ticker PriceToString last value is incorrect")
	}
	if newTicker.PriceToString(newPair, "high", asset.Spot) != "1298" {
		t.Error("Test Failed - ticker PriceToString high value is incorrect")
	}
	if newTicker.PriceToString(newPair, "low", asset.Spot) != "1148" {
		t.Error("Test Failed - ticker PriceToString low value is incorrect")
	}
	if newTicker.PriceToString(newPair, "bid", asset.Spot) != "1195" {
		t.Error("Test Failed - ticker PriceToString bid value is incorrect")
	}
	if newTicker.PriceToString(newPair, "ask", asset.Spot) != "1220" {
		t.Error("Test Failed - ticker PriceToString ask value is incorrect")
	}
	if newTicker.PriceToString(newPair, "volume", asset.Spot) != "5" {
		t.Error("Test Failed - ticker PriceToString volume value is incorrect")
	}
	if newTicker.PriceToString(newPair, "ath", asset.Spot) != "1337" {
		t.Error("Test Failed - ticker PriceToString ath value is incorrect")
	}
	if newTicker.PriceToString(newPair, "obtuse", asset.Spot) != "" {
		t.Error("Test Failed - ticker PriceToString obtuse value is incorrect")
	}
}

func TestGetTicker(t *testing.T) {
	newPair := currency.NewPairFromStrings("BTC", "USD")
	priceStruct := Price{
		Pair:     newPair,
		Last:     1200,
		High:     1298,
		Low:      1148,
		Bid:      1195,
		Ask:      1220,
		Volume:   5,
		PriceATH: 1337,
	}

	err := ProcessTicker("bitfinex", &priceStruct, asset.Spot)
	if err != nil {
		t.Fatal("Test failed. ProcessTicker error", err)
	}

	tickerPrice, err := GetTicker("bitfinex", newPair, asset.Spot)
	if err != nil {
		t.Errorf("Test Failed - Ticker GetTicker init error: %s", err)
	}
	if !tickerPrice.Pair.Equal(newPair) {
		t.Error("Test Failed - ticker tickerPrice.CurrencyPair value is incorrect")
	}

	_, err = GetTicker("blah", newPair, asset.Spot)
	if err == nil {
		t.Fatal("Test Failed. TestGetTicker returned nil error on invalid exchange")
	}

	newPair.Base = currency.ETH
	_, err = GetTicker("bitfinex", newPair, asset.Spot)
	if err == nil {
		t.Fatal("Test Failed. TestGetTicker returned ticker for invalid first currency")
	}

	btcltcPair := currency.NewPairFromStrings("BTC", "LTC")
	_, err = GetTicker("bitfinex", btcltcPair, asset.Spot)
	if err == nil {
		t.Fatal("Test Failed. TestGetTicker returned ticker for invalid second currency")
	}

	priceStruct.PriceATH = 9001
	priceStruct.Pair.Base = currency.ETH
	err = ProcessTicker("bitfinex", &priceStruct, "futures_3m")
	if err != nil {
		t.Fatal("Test failed. ProcessTicker error", err)
	}

	tickerPrice, err = GetTicker("bitfinex", newPair, "futures_3m")
	if err != nil {
		t.Errorf("Test Failed - Ticker GetTicker init error: %s", err)
	}

	if tickerPrice.PriceATH != 9001 {
		t.Error("Test Failed - ticker tickerPrice.PriceATH value is incorrect")
	}
}

func TestGetTickerByExchange(t *testing.T) {
	newPair := currency.NewPairFromStrings("BTC", "USD")
	priceStruct := Price{
		Pair:     newPair,
		Last:     1200,
		High:     1298,
		Low:      1148,
		Bid:      1195,
		Ask:      1220,
		Volume:   5,
		PriceATH: 1337,
	}

	anxTicker := CreateNewTicker("ANX", &priceStruct, asset.Spot)
	Tickers = append(Tickers, anxTicker)

	tickerPtr, err := GetTickerByExchange("ANX")
	if err != nil {
		t.Errorf("Test Failed - GetTickerByExchange init error: %s", err)
	}
	if tickerPtr.ExchangeName != "ANX" {
		t.Error("Test Failed - GetTickerByExchange ExchangeName value is incorrect")
	}
}

func TestBaseCurrencyExists(t *testing.T) {
	newPair := currency.NewPairFromStrings("BTC", "USD")
	priceStruct := Price{
		Pair:     newPair,
		Last:     1200,
		High:     1298,
		Low:      1148,
		Bid:      1195,
		Ask:      1220,
		Volume:   5,
		PriceATH: 1337,
	}

	alphaTicker := CreateNewTicker("alphapoint", &priceStruct, asset.Spot)
	Tickers = append(Tickers, alphaTicker)

	if !BaseCurrencyExists("alphapoint", currency.BTC) {
		t.Error("Test Failed - BaseCurrencyExists1 value return is incorrect")
	}
	if BaseCurrencyExists("alphapoint", currency.NewCode("CATS")) {
		t.Error("Test Failed - BaseCurrencyExists2 value return is incorrect")
	}
}

func TestQuoteCurrencyExists(t *testing.T) {
	t.Parallel()

	newPair := currency.NewPairFromStrings("BTC", "USD")
	priceStruct := Price{
		Pair:     newPair,
		Last:     1200,
		High:     1298,
		Low:      1148,
		Bid:      1195,
		Ask:      1220,
		Volume:   5,
		PriceATH: 1337,
	}

	bitstampTicker := CreateNewTicker("bitstamp", &priceStruct, asset.Spot)
	Tickers = append(Tickers, bitstampTicker)

	if !QuoteCurrencyExists("bitstamp", newPair) {
		t.Error("Test Failed - QuoteCurrencyExists1 value return is incorrect")
	}

	newPair.Quote = currency.NewCode("DOGS")
	if QuoteCurrencyExists("bitstamp", newPair) {
		t.Error("Test Failed - QuoteCurrencyExists2 value return is incorrect")
	}
}

func TestCreateNewTicker(t *testing.T) {
	const float64Type = "float64"
	newPair := currency.NewPairFromStrings("BTC", "USD")
	priceStruct := Price{
		Pair:     newPair,
		Last:     1200,
		High:     1298,
		Low:      1148,
		Bid:      1195,
		Ask:      1220,
		Volume:   5,
		PriceATH: 1337,
	}

	newTicker := CreateNewTicker("ANX", &priceStruct, asset.Spot)

	if reflect.ValueOf(newTicker).NumField() != 2 {
		t.Error("Test Failed - ticker CreateNewTicker struct change/or updated")
	}
	if reflect.TypeOf(newTicker.ExchangeName).String() != "string" {
		t.Error("Test Failed - ticker CreateNewTicker.ExchangeName value is not a string")
	}
	if newTicker.ExchangeName != "ANX" {
		t.Error("Test Failed - ticker CreateNewTicker.ExchangeName value is not ANX")
	}

	if !newTicker.Price[currency.BTC.Upper().String()][currency.USD.Upper().String()][asset.Spot.String()].Pair.Equal(newPair) {
		t.Error("Test Failed - ticker newTicker.Price[BTC][USD].Pair.Pair().String() value is not expected 'BTCUSD'")
	}
	if reflect.TypeOf(newTicker.Price["BTC"]["USD"][asset.Spot.String()].Ask).String() != float64Type {
		t.Error("Test Failed - ticker newTicker.Price[BTC][USD].Ask value is not a float64")
	}
	if reflect.TypeOf(newTicker.Price["BTC"]["USD"][asset.Spot.String()].Bid).String() != float64Type {
		t.Error("Test Failed - ticker newTicker.Price[BTC][USD].Bid value is not a float64")
	}
	if reflect.TypeOf(newTicker.Price["BTC"]["USD"][asset.Spot.String()].Pair).String() != "currency.Pair" {
		t.Error("Test Failed - ticker newTicker.Price[BTC][USD].CurrencyPair value is not a currency.Pair")
	}
	if reflect.TypeOf(newTicker.Price["BTC"]["USD"][asset.Spot.String()].High).String() != float64Type {
		t.Error("Test Failed - ticker newTicker.Price[BTC][USD].High value is not a float64")
	}
	if reflect.TypeOf(newTicker.Price["BTC"]["USD"][asset.Spot.String()].Last).String() != float64Type {
		t.Error("Test Failed - ticker newTicker.Price[BTC][USD].Last value is not a float64")
	}
	if reflect.TypeOf(newTicker.Price["BTC"]["USD"][asset.Spot.String()].Low).String() != float64Type {
		t.Error("Test Failed - ticker newTicker.Price[BTC][USD].Low value is not a float64")
	}
	if reflect.TypeOf(newTicker.Price["BTC"]["USD"][asset.Spot.String()].PriceATH).String() != float64Type {
		t.Error("Test Failed - ticker newTicker.Price[BTC][USD].PriceATH value is not a float64")
	}
	if reflect.TypeOf(newTicker.Price["BTC"]["USD"][asset.Spot.String()].Volume).String() != float64Type {
		t.Error("Test Failed - ticker newTicker.Price[BTC][USD].Volume value is not a float64")
	}
}

func TestProcessTicker(t *testing.T) { // non-appending function to tickers
	Tickers = []Ticker{}
	exchName := "bitstamp"
	newPair := currency.NewPairFromStrings("BTC", "USD")
	priceStruct := Price{
		Last:     1200,
		High:     1298,
		Low:      1148,
		Bid:      1195,
		Ask:      1220,
		Volume:   5,
		PriceATH: 1337,
	}

	// test for empty pair
	err := ProcessTicker(exchName, &priceStruct, asset.Spot)
	if err == nil {
		t.Fatal("empty pair should throw an err")
	}

	// test for empty asset type
	priceStruct.Pair = newPair
	err = ProcessTicker(exchName, &priceStruct, "")
	if err == nil {
		t.Fatal("Test failed. ProcessTicker error cannot be nil")
	}

	// now process a valid ticker
	err = ProcessTicker(exchName, &priceStruct, asset.Spot)
	if err != nil {
		t.Fatal("Test failed. ProcessTicker error", err)
	}
	result, err := GetTicker(exchName, newPair, asset.Spot)
	if err != nil {
		t.Fatal("Test failed. TestProcessTicker failed to create and return a new ticker")
	}
	if !result.Pair.Equal(newPair) {
		t.Fatal("Test failed. TestProcessTicker pair mismatch")
	}

	// now test for processing a pair with a different quote currency
	newPair = currency.NewPairFromStrings("BTC", "AUD")
	priceStruct.Pair = newPair
	err = ProcessTicker(exchName, &priceStruct, asset.Spot)
	if err != nil {
		t.Fatal("Test failed. ProcessTicker error", err)
	}
	result, err = GetTicker(exchName, newPair, asset.Spot)
	if err != nil {
		t.Fatal("Test failed. TestProcessTicker failed to create and return a new ticker")
	}
	result, err = GetTicker(exchName, newPair, asset.Spot)
	if err != nil {
		t.Fatal("Test failed. TestProcessTicker failed to return an existing ticker")
	}

	// now test for processing a pair which has a different base currency
	newPair = currency.NewPairFromStrings("LTC", "AUD")
	priceStruct.Pair = newPair
	err = ProcessTicker(exchName, &priceStruct, asset.Spot)
	if err != nil {
		t.Fatal("Test failed. ProcessTicker error", err)
	}
	result, err = GetTicker(exchName, newPair, asset.Spot)
	if err != nil {
		t.Fatal("Test failed. TestProcessTicker failed to create and return a new ticker")
	}
	result, err = GetTicker(exchName, newPair, asset.Spot)
	if err != nil {
		t.Fatal("Test failed. TestProcessTicker failed to return an existing ticker")
	}

	type quick struct {
		Name string
		P    currency.Pair
		TP   Price
	}

	var testArray []quick

	_ = rand.NewSource(time.Now().Unix())

	var wg sync.WaitGroup
	var sm sync.Mutex

	var catastrophicFailure bool
	for i := 0; i < 500; i++ {
		if catastrophicFailure {
			break
		}

		wg.Add(1)
		go func() {
			newName := "Exchange" + strconv.FormatInt(rand.Int63(), 10)
			newPairs := currency.NewPairFromStrings("BTC"+strconv.FormatInt(rand.Int63(), 10),
				"USD"+strconv.FormatInt(rand.Int63(), 10))

			tp := Price{
				Pair: newPairs,
				Last: rand.Float64(),
			}

			sm.Lock()
			err = ProcessTicker(newName, &tp, asset.Spot)
			if err != nil {
				t.Error(err)
				catastrophicFailure = true
				return
			}

			testArray = append(testArray, quick{Name: newName, P: newPairs, TP: tp})
			sm.Unlock()
			wg.Done()
		}()
	}

	if catastrophicFailure {
		t.Fatal("Test failed. ProcessTicker error")
	}

	wg.Wait()

	for _, test := range testArray {
		wg.Add(1)
		fatalErr := false
		go func(test quick) {
			result, err := GetTicker(test.Name, test.P, asset.Spot)
			if err != nil {
				fatalErr = true
				return
			}

			if result.Last != test.TP.Last {
				t.Error("Test failed. TestProcessTicker failed bad values")
			}

			wg.Done()
		}(test)

		if fatalErr {
			t.Fatal("Test failed. TestProcessTicker failed to retrieve new ticker")
		}
	}
	wg.Wait()

}
