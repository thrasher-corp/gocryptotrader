package ticker

import (
	"errors"
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
	err := dispatch.Start(1, dispatch.DefaultJobsLimit)
	if err != nil {
		log.Fatal(err)
	}

	cpyMux = service.mux

	os.Exit(m.Run())
}

var cpyMux *dispatch.Mux

func TestSubscribeTicker(t *testing.T) {
	_, err := SubscribeTicker("", currency.Pair{}, asset.Item(""))
	if err == nil {
		t.Error("error cannot be nil")
	}

	p := currency.NewPair(currency.BTC, currency.USD)

	// force error
	service.mux = nil
	err = ProcessTicker(&Price{
		Pair:         p,
		ExchangeName: "subscribetest",
		AssetType:    asset.Spot})
	if err == nil {
		t.Error("error cannot be nil")
	}

	sillyP := p
	sillyP.Base = currency.GALA_NEO
	err = ProcessTicker(&Price{
		Pair:         sillyP,
		ExchangeName: "subscribetest",
		AssetType:    asset.Spot})
	if err == nil {
		t.Error("error cannot be nil")
	}

	sillyP.Quote = currency.AAA
	err = ProcessTicker(&Price{
		Pair:         sillyP,
		ExchangeName: "subscribetest",
		AssetType:    asset.Spot})
	if err == nil {
		t.Error("error cannot be nil")
	}

	err = ProcessTicker(&Price{
		Pair:         sillyP,
		ExchangeName: "subscribetest",
		AssetType:    "silly",
	})
	if err == nil {
		t.Error("error cannot be nil")
	}
	// reinstate mux
	service.mux = cpyMux

	err = ProcessTicker(&Price{
		Pair:         p,
		ExchangeName: "subscribetest",
		AssetType:    asset.Spot})
	if err != nil {
		t.Fatal(err)
	}

	_, err = SubscribeTicker("subscribetest", p, asset.Spot)
	if err != nil {
		t.Error("cannot subscribe to ticker", err)
	}
}

func TestSubscribeToExchangeTickers(t *testing.T) {
	_, err := SubscribeToExchangeTickers("")
	if err == nil {
		t.Error("error cannot be nil")
	}

	p := currency.NewPair(currency.BTC, currency.USD)

	err = ProcessTicker(&Price{
		Pair:         p,
		ExchangeName: "subscribeExchangeTest",
		AssetType:    asset.Spot})
	if err != nil {
		t.Error(err)
	}

	_, err = SubscribeToExchangeTickers("subscribeExchangeTest")
	if err != nil {
		t.Error("error cannot be nil", err)
	}
}

func TestGetTicker(t *testing.T) {
	newPair, err := currency.NewPairFromStrings("BTC", "USD")
	if err != nil {
		t.Fatal(err)
	}
	priceStruct := Price{
		Pair:         newPair,
		Last:         1200,
		High:         1298,
		Low:          1148,
		Bid:          1195,
		Ask:          1220,
		Volume:       5,
		PriceATH:     1337,
		ExchangeName: "bitfinex",
		AssetType:    asset.Spot,
	}

	err = ProcessTicker(&priceStruct)
	if err != nil {
		t.Fatal("ProcessTicker error", err)
	}

	tickerPrice, err := GetTicker("bitfinex", newPair, asset.Spot)
	if err != nil {
		t.Errorf("Ticker GetTicker init error: %s", err)
	}
	if !tickerPrice.Pair.Equal(newPair) {
		t.Error("ticker tickerPrice.CurrencyPair value is incorrect")
	}

	_, err = GetTicker("blah", newPair, asset.Spot)
	if err == nil {
		t.Fatal("TestGetTicker returned nil error on invalid exchange")
	}

	newPair.Base = currency.ETH
	_, err = GetTicker("bitfinex", newPair, asset.Spot)
	if err == nil {
		t.Fatal("TestGetTicker returned ticker for invalid first currency")
	}

	btcltcPair, err := currency.NewPairFromStrings("BTC", "LTC")
	if err != nil {
		t.Fatal(err)
	}

	_, err = GetTicker("bitfinex", btcltcPair, asset.Spot)
	if err == nil {
		t.Fatal("TestGetTicker returned ticker for invalid second currency")
	}

	priceStruct.PriceATH = 9001
	priceStruct.Pair.Base = currency.ETH
	priceStruct.AssetType = "futures_3m"
	err = ProcessTicker(&priceStruct)
	if err != nil {
		t.Fatal("ProcessTicker error", err)
	}

	tickerPrice, err = GetTicker("bitfinex", newPair, "futures_3m")
	if err != nil {
		t.Errorf("Ticker GetTicker init error: %s", err)
	}

	if tickerPrice.PriceATH != 9001 {
		t.Error("ticker tickerPrice.PriceATH value is incorrect")
	}

	_, err = GetTicker("bitfinex", newPair, "meowCats")
	if err == nil {
		t.Error("Ticker GetTicker error cannot be nil")
	}

	priceStruct.AssetType = "meowCats"
	err = ProcessTicker(&priceStruct)
	if err != nil {
		t.Fatal("ProcessTicker error", err)
	}

	// process update again
	err = ProcessTicker(&priceStruct)
	if err != nil {
		t.Fatal("ProcessTicker error", err)
	}
}

func TestFindLast(t *testing.T) {
	cp := currency.NewPair(currency.BTC, currency.XRP)
	_, err := FindLast(cp, asset.Spot)
	if !errors.Is(err, errTickerNotFound) {
		t.Errorf("received: %v but expected: %v", err, errTickerNotFound)
	}

	err = service.update(&Price{Last: 0, ExchangeName: "testerinos", Pair: cp, AssetType: asset.Spot})
	if err != nil {
		t.Fatal(err)
	}

	_, err = FindLast(cp, asset.Spot)
	if !errors.Is(err, errInvalidTicker) {
		t.Errorf("received: %v but expected: %v", err, errInvalidTicker)
	}

	err = service.update(&Price{Last: 1337, ExchangeName: "testerinos", Pair: cp, AssetType: asset.Spot})
	if err != nil {
		t.Fatal(err)
	}

	last, err := FindLast(cp, asset.Spot)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v but expected: %v", err, nil)
	}

	if last != 1337 {
		t.Fatal("unexpected value")
	}
}

func TestProcessTicker(t *testing.T) { // non-appending function to tickers
	exchName := "bitstamp"
	newPair, err := currency.NewPairFromStrings("BTC", "USD")
	if err != nil {
		t.Fatal(err)
	}

	priceStruct := Price{
		Last:     1200,
		High:     1298,
		Low:      1148,
		Bid:      1195,
		Ask:      1220,
		Volume:   5,
		PriceATH: 1337,
	}

	err = ProcessTicker(&priceStruct)
	if err == nil {
		t.Fatal("empty exchange should throw an err")
	}

	priceStruct.ExchangeName = exchName

	// test for empty pair
	err = ProcessTicker(&priceStruct)
	if err == nil {
		t.Fatal("empty pair should throw an err")
	}

	// test for empty asset type
	priceStruct.Pair = newPair
	err = ProcessTicker(&priceStruct)
	if err == nil {
		t.Fatal("ProcessTicker error cannot be nil")
	}
	priceStruct.AssetType = asset.Spot
	// now process a valid ticker
	err = ProcessTicker(&priceStruct)
	if err != nil {
		t.Fatal("ProcessTicker error", err)
	}
	result, err := GetTicker(exchName, newPair, asset.Spot)
	if err != nil {
		t.Fatal("TestProcessTicker failed to create and return a new ticker")
	}
	if !result.Pair.Equal(newPair) {
		t.Fatal("TestProcessTicker pair mismatch")
	}

	// now test for processing a pair with a different quote currency
	newPair, err = currency.NewPairFromStrings("BTC", "AUD")
	if err != nil {
		t.Fatal(err)
	}

	priceStruct.Pair = newPair
	err = ProcessTicker(&priceStruct)
	if err != nil {
		t.Fatal("ProcessTicker error", err)
	}
	_, err = GetTicker(exchName, newPair, asset.Spot)
	if err != nil {
		t.Fatal("TestProcessTicker failed to create and return a new ticker")
	}
	_, err = GetTicker(exchName, newPair, asset.Spot)
	if err != nil {
		t.Fatal("TestProcessTicker failed to return an existing ticker")
	}

	// now test for processing a pair which has a different base currency
	newPair, err = currency.NewPairFromStrings("LTC", "AUD")
	if err != nil {
		t.Fatal(err)
	}

	priceStruct.Pair = newPair
	err = ProcessTicker(&priceStruct)
	if err != nil {
		t.Fatal("ProcessTicker error", err)
	}
	_, err = GetTicker(exchName, newPair, asset.Spot)
	if err != nil {
		t.Fatal("TestProcessTicker failed to create and return a new ticker")
	}
	_, err = GetTicker(exchName, newPair, asset.Spot)
	if err != nil {
		t.Fatal("TestProcessTicker failed to return an existing ticker")
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
			// nolint:gosec // no need to import crypo/rand for testing
			newName := "Exchange" + strconv.FormatInt(rand.Int63(), 10)
			newPairs, err := currency.NewPairFromStrings("BTC"+strconv.FormatInt(rand.Int63(), 10), // nolint:gosec // no need to import crypo/rand for testing
				"USD"+strconv.FormatInt(rand.Int63(), 10)) // nolint:gosec // no need to import crypo/rand for testing
			if err != nil {
				log.Fatal(err)
			}

			tp := Price{
				Pair:         newPairs,
				Last:         rand.Float64(), // nolint:gosec // no need to import crypo/rand for testing
				ExchangeName: newName,
				AssetType:    asset.Spot,
			}

			sm.Lock()
			err = ProcessTicker(&tp)
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
		t.Fatal("ProcessTicker error")
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
				t.Error("TestProcessTicker failed bad values")
			}

			wg.Done()
		}(test)

		if fatalErr {
			t.Fatal("TestProcessTicker failed to retrieve new ticker")
		}
	}
	wg.Wait()
}

func TestGetAssociation(t *testing.T) {
	_, err := service.getAssociations("")
	if !errors.Is(err, errExchangeNameIsEmpty) {
		t.Errorf("received: %v but expected: %v", err, errExchangeNameIsEmpty)
	}

	service.mux = nil

	_, err = service.getAssociations("getassociation")
	if err == nil {
		t.Error("error cannot be nil")
	}

	service.mux = cpyMux
}
