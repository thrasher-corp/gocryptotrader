package ticker

import (
	"log"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common/key"
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
	_, err := SubscribeTicker("", currency.EMPTYPAIR, asset.Empty)
	if err == nil {
		t.Error("error cannot be nil")
	}

	p := currency.NewBTCUSD()

	// force error
	service.mux = nil
	err = ProcessTicker(&Price{
		Pair:         p,
		ExchangeName: "subscribetest",
		AssetType:    asset.Spot,
	})
	if err == nil {
		t.Error("error cannot be nil")
	}

	sillyP := p
	sillyP.Base = currency.GALA_NEO
	err = ProcessTicker(&Price{
		Pair:         sillyP,
		ExchangeName: "subscribetest",
		AssetType:    asset.Spot,
	})
	if err == nil {
		t.Error("error cannot be nil")
	}

	sillyP.Quote = currency.AAA
	err = ProcessTicker(&Price{
		Pair:         sillyP,
		ExchangeName: "subscribetest",
		AssetType:    asset.Spot,
	})
	if err == nil {
		t.Error("error cannot be nil")
	}

	err = ProcessTicker(&Price{
		Pair:         sillyP,
		ExchangeName: "subscribetest",
		AssetType:    asset.DownsideProfitContract,
	})
	if err == nil {
		t.Error("error cannot be nil")
	}
	// reinstate mux
	service.mux = cpyMux

	err = ProcessTicker(&Price{
		Pair:         p,
		ExchangeName: "subscribetest",
		AssetType:    asset.Spot,
	})
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

	p := currency.NewBTCUSD()

	err = ProcessTicker(&Price{
		Pair:         p,
		ExchangeName: "subscribeExchangeTest",
		AssetType:    asset.Spot,
	})
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
	priceStruct.AssetType = asset.DownsideProfitContract
	err = ProcessTicker(&priceStruct)
	if err != nil {
		t.Fatal("ProcessTicker error", err)
	}

	tickerPrice, err = GetTicker("bitfinex", newPair, asset.DownsideProfitContract)
	if err != nil {
		t.Errorf("Ticker GetTicker init error: %s", err)
	}

	if tickerPrice.PriceATH != 9001 {
		t.Error("ticker tickerPrice.PriceATH value is incorrect")
	}

	_, err = GetTicker("bitfinex", newPair, asset.UpsideProfitContract)
	if err == nil {
		t.Error("Ticker GetTicker error cannot be nil")
	}

	priceStruct.AssetType = asset.UpsideProfitContract
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
	assert.ErrorIs(t, err, ErrTickerNotFound)

	err = service.update(&Price{Last: 0, ExchangeName: "testerinos", Pair: cp, AssetType: asset.Spot})
	require.NoError(t, err, "service update must not error")

	_, err = FindLast(cp, asset.Spot)
	assert.ErrorIs(t, err, errInvalidTicker)

	err = service.update(&Price{Last: 1337, ExchangeName: "testerinos", Pair: cp, AssetType: asset.Spot})
	require.NoError(t, err, "service update must not error")

	last, err := FindLast(cp, asset.Spot)
	assert.NoError(t, err)
	assert.Equal(t, 1337.0, last)
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

	err = ProcessTicker(&Price{
		ExchangeName: "Bitfinex",
		Pair:         currency.NewBTCUSD(),
		AssetType:    asset.Margin,
		Bid:          1337,
		Ask:          1337,
	})
	assert.ErrorIs(t, err, ErrBidEqualsAsk, "ProcessTicker should error locked market")

	err = ProcessTicker(&Price{
		ExchangeName: "Bitfinex",
		Pair:         currency.NewBTCUSD(),
		AssetType:    asset.Margin,
		Bid:          1338,
		Ask:          1336,
	})
	assert.ErrorIs(t, err, errBidGreaterThanAsk)

	err = ProcessTicker(&Price{
		ExchangeName: "Bitfinex",
		Pair:         currency.NewBTCUSD(),
		AssetType:    asset.MarginFunding,
		Bid:          1338,
		Ask:          1336,
	})
	assert.NoError(t, err)

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
	for range 500 {
		if catastrophicFailure {
			break
		}

		wg.Add(1)
		go func() {
			//nolint:gosec // no need to import crypo/rand for testing
			newName := "Exchange" + strconv.FormatInt(rand.Int63(), 10)
			newPairs, err := currency.NewPairFromStrings("BTC"+strconv.FormatInt(rand.Int63(), 10), //nolint:gosec // no need to import crypo/rand for testing
				"USD"+strconv.FormatInt(rand.Int63(), 10)) //nolint:gosec // no need to import crypo/rand for testing
			if err != nil {
				log.Fatal(err)
			}

			tp := Price{
				Pair:         newPairs,
				Last:         rand.Float64(), //nolint:gosec // no need to import crypo/rand for testing
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
	assert.ErrorIs(t, err, ErrExchangeNameIsEmpty)

	service.mux = nil

	_, err = service.getAssociations("getassociation")
	if err == nil {
		t.Error("error cannot be nil")
	}

	service.mux = cpyMux
}

func TestGetExchangeTickersPublic(t *testing.T) {
	_, err := GetExchangeTickers("")
	assert.ErrorIs(t, err, ErrExchangeNameIsEmpty)
}

func TestGetExchangeTickers(t *testing.T) {
	t.Parallel()
	s := Service{
		Tickers:  make(map[key.ExchangePairAsset]*Ticker),
		Exchange: make(map[string]uuid.UUID),
	}

	_, err := s.getExchangeTickers("")
	assert.ErrorIs(t, err, ErrExchangeNameIsEmpty)

	_, err = s.getExchangeTickers("test")
	assert.ErrorIs(t, err, errExchangeNotFound)

	s.Tickers[key.ExchangePairAsset{
		Exchange: "test",
		Base:     currency.XBT.Item,
		Quote:    currency.DOGE.Item,
		Asset:    asset.Futures,
	}] = &Ticker{
		Price: Price{
			Pair:         currency.NewPair(currency.XBT, currency.DOGE),
			ExchangeName: "test",
			AssetType:    asset.Futures,
			OpenInterest: 1337,
		},
	}
	s.Exchange["test"] = uuid.Must(uuid.NewV4())

	resp, err := s.getExchangeTickers("test")
	assert.NoError(t, err)
	if len(resp) != 1 {
		t.Fatal("unexpected length")
	}
	assert.Equal(t, 1337.0, resp[0].OpenInterest)
}
