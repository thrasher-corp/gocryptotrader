package ticker

import (
	"log"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/dispatch"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func TestMain(m *testing.M) {
	err := dispatch.Start(1)
	if err != nil {
		log.Fatal(err)
	}

	os.Exit(m.Run())
}

func TestSubscribeTicker(t *testing.T) {
	_, err := SubscribeTicker("", currency.Pair{}, asset.Item(""))
	if err == nil {
		t.Error("error cannot be nil")
	}

	p := currency.NewPair(currency.BTC, currency.USD)

	err = ProcessTicker("subscribetest", &Price{Pair: p}, asset.Spot)
	if err != nil {
		t.Error(err)
	}

	_, err = SubscribeTicker("subscribetest", p, asset.Spot)
	if err != nil {
		t.Error("error cannot be nil", err)
	}
}

func TestSubscribeToExchangeTickers(t *testing.T) {
	_, err := SubscribeToExchangeTickers("")
	if err == nil {
		t.Error("error cannot be nil")
	}

	p := currency.NewPair(currency.BTC, currency.USD)

	err = ProcessTicker("subscribeExchangeTest", &Price{Pair: p}, asset.Spot)
	if err != nil {
		t.Error(err)
	}

	_, err = SubscribeToExchangeTickers("subscribeExchangeTest")
	if err != nil {
		t.Error("error cannot be nil", err)
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

	_, err = GetTicker("bitfinex", newPair, "meowCats")
	if err == nil {
		t.Error("Test Failed - Ticker GetTicker error cannot be nil")
	}

	err = ProcessTicker("bitfinex", &priceStruct, "meowCats")
	if err != nil {
		t.Fatal("Test failed. ProcessTicker error", err)
	}

	// process update again
	err = ProcessTicker("bitfinex", &priceStruct, "meowCats")
	if err != nil {
		t.Fatal("Test failed. ProcessTicker error", err)
	}
}

func TestProcessTicker(t *testing.T) { // non-appending function to tickers
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

	err := ProcessTicker("", &priceStruct, asset.Spot)
	if err == nil {
		t.Fatal("empty exchange should throw an err")
	}

	// test for empty pair
	err = ProcessTicker(exchName, &priceStruct, asset.Spot)
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

func TestSetItemID(t *testing.T) {
	err := service.SetItemID(nil)
	if err == nil {
		t.Error("error cannot be nil")
	}

	err = service.SetItemID(&Price{})
	if err == nil {
		t.Error("error cannot be nil")
	}
}

func TestGetAssociation(t *testing.T) {
	_, err := service.GetAssociations(nil)
	if err == nil {
		t.Error("error cannot be nil ")
	}
}
