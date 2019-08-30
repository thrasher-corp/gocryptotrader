package orderbook

import (
	"math/rand"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func TestVerify(t *testing.T) {
	t.Parallel()
	b := Base{
		ExchangeName: "TestExchange",
		Pair:         currency.NewPair(currency.BTC, currency.USD),
		Bids: []Item{
			{Price: 100}, {Price: 101}, {Price: 99},
		},
		Asks: []Item{
			{Price: 100}, {Price: 99}, {Price: 101},
		},
	}

	b.Verify()
	if r := b.Bids[1].Price; r != 100 {
		t.Error("unexpected result")
	}
	if r := b.Asks[1].Price; r != 100 {
		t.Error("unexpected result")
	}
}

func TestCalculateTotalBids(t *testing.T) {
	t.Parallel()
	currency := currency.NewPairFromStrings("BTC", "USD")
	base := Base{
		Pair:        currency,
		Bids:        []Item{{Price: 100, Amount: 10}},
		LastUpdated: time.Now(),
	}

	a, b := base.TotalBidsAmount()
	if a != 10 && b != 1000 {
		t.Fatal("Test failed. TestCalculateTotalBids expected a = 10 and b = 1000")
	}
}

func TestCalculateTotaAsks(t *testing.T) {
	t.Parallel()
	currency := currency.NewPairFromStrings("BTC", "USD")
	base := Base{
		Pair: currency,
		Asks: []Item{{Price: 100, Amount: 10}},
	}

	a, b := base.TotalAsksAmount()
	if a != 10 && b != 1000 {
		t.Fatal("Test failed. TestCalculateTotalAsks expected a = 10 and b = 1000")
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()
	currency := currency.NewPairFromStrings("BTC", "USD")
	timeNow := time.Now()
	base := Base{
		Pair:        currency,
		Asks:        []Item{{Price: 100, Amount: 10}},
		Bids:        []Item{{Price: 200, Amount: 10}},
		LastUpdated: timeNow,
	}

	asks := []Item{{Price: 200, Amount: 101}}
	bids := []Item{{Price: 201, Amount: 100}}
	time.Sleep(time.Millisecond * 50)
	base.Update(bids, asks)

	if !base.LastUpdated.After(timeNow) {
		t.Fatal("test failed. TestUpdate expected LastUpdated to be greater then original time")
	}

	a, b := base.TotalAsksAmount()
	if a != 100 && b != 20200 {
		t.Fatal("Test failed. TestUpdate expected a = 100 and b = 20100")
	}

	a, b = base.TotalBidsAmount()
	if a != 100 && b != 20100 {
		t.Fatal("Test failed. TestUpdate expected a = 100 and b = 20100")
	}
}

func TestGetOrderbook(t *testing.T) {
	c := currency.NewPairFromStrings("BTC", "USD")
	base := Base{
		Pair:         c,
		Asks:         []Item{{Price: 100, Amount: 10}},
		Bids:         []Item{{Price: 200, Amount: 10}},
		ExchangeName: "Exchange",
		AssetType:    asset.Spot,
	}

	err := service.Update(base)
	if err != nil {
		t.Fatal(err)
	}

	result, err := Get("Exchange", c, asset.Spot)
	if err != nil {
		t.Fatalf("Test failed. TestGetOrderbook failed to get orderbook. Error %s",
			err)
	}
	if !result.Pair.Equal(c) {
		t.Fatal("Test failed. TestGetOrderbook failed. Mismatched pairs")
	}

	_, err = Get("nonexistent", c, asset.Spot)
	if err == nil {
		t.Fatal("Test failed. TestGetOrderbook retrieved non-existent orderbook")
	}

	c.Base = currency.NewCode("blah")
	_, err = Get("Exchange", c, asset.Spot)
	if err == nil {
		t.Fatal("Test failed. TestGetOrderbook retrieved non-existent orderbook using invalid first currency")
	}

	newCurrency := currency.NewPairFromStrings("BTC", "AUD")
	_, err = Get("Exchange", newCurrency, asset.Spot)
	if err == nil {
		t.Fatal("Test failed. TestGetOrderbook retrieved non-existent orderbook using invalid second currency")
	}
}

func TestCreateNewOrderbook(t *testing.T) {
	c := currency.NewPairFromStrings("BTC", "USD")
	base := Base{
		Pair:         c,
		Asks:         []Item{{Price: 100, Amount: 10}},
		Bids:         []Item{{Price: 200, Amount: 10}},
		ExchangeName: "testCreateNewOrderbook",
		AssetType:    asset.Spot,
	}

	err := service.Update(base)
	if err != nil {
		t.Fatal(err)
	}

	result, err := Get("testCreateNewOrderbook", c, asset.Spot)
	if err != nil {
		t.Fatal("Test failed. TestCreateNewOrderbook failed to create new orderbook")
	}

	if !result.Pair.Equal(c) {
		t.Fatal("Test failed. TestCreateNewOrderbook result pair is incorrect")
	}

	a, b := result.TotalAsksAmount()
	if a != 10 && b != 1000 {
		t.Fatal("Test failed. TestCreateNewOrderbook CalculateTotalAsks value is incorrect")
	}

	a, b = result.TotalBidsAmount()
	if a != 10 && b != 2000 {
		t.Fatal("Test failed. TestCreateNewOrderbook CalculateTotalBids value is incorrect")
	}
}

func TestProcessOrderbook(t *testing.T) {
	c := currency.NewPairFromStrings("BTC", "USD")
	base := Base{
		Asks:         []Item{{Price: 100, Amount: 10}},
		Bids:         []Item{{Price: 200, Amount: 10}},
		ExchangeName: "ProcessOrderbook",
	}

	// test for empty pair
	base.Pair = currency.Pair{}
	err := base.Process()
	if err == nil {
		t.Error("empty pair should throw an err")
	}

	// test for empty asset type
	base.Pair = c
	err = base.Process()
	if err == nil {
		t.Error("empty asset type should throw an err")
	}

	// now process a valid orderbook
	base.AssetType = asset.Spot
	err = base.Process()
	if err != nil {
		t.Error("unexpcted result: ", err)
	}
	result, err := Get("ProcessOrderbook", c, asset.Spot)
	if err != nil {
		t.Fatal("Test failed. TestProcessOrderbook failed to create new orderbook")
	}
	if !result.Pair.Equal(c) {
		t.Fatal("Test failed. TestProcessOrderbook result pair is incorrect")
	}

	// now test for processing a pair with a different quote currency
	c = currency.NewPairFromStrings("BTC", "GBP")
	base.Pair = c
	err = base.Process()
	if err != nil {
		t.Error("Test Failed - Process() error", err)
	}
	result, err = Get("ProcessOrderbook", c, asset.Spot)
	if err != nil {
		t.Fatal("Test failed. TestProcessOrderbook failed to retrieve new orderbook")
	}
	if !result.Pair.Equal(c) {
		t.Fatal("Test failed. TestProcessOrderbook result pair is incorrect")
	}

	// now test for processing a pair which has a different base currency
	c = currency.NewPairFromStrings("LTC", "GBP")
	base.Pair = c
	err = base.Process()
	if err != nil {
		t.Error("Test Failed - Process() error", err)
	}
	result, err = Get("ProcessOrderbook", c, asset.Spot)
	if err != nil {
		t.Fatal("Test failed. TestProcessOrderbook failed to retrieve new orderbook")
	}
	if !result.Pair.Equal(c) {
		t.Fatal("Test failed. TestProcessOrderbook result pair is incorrect")
	}

	base.Asks = []Item{{Price: 200, Amount: 200}}
	base.AssetType = "monthly"
	err = base.Process()
	if err != nil {
		t.Error("Test Failed - Process() error", err)
	}

	result, err = Get("ProcessOrderbook", c, "monthly")
	if err != nil {
		t.Fatal("Test failed. TestProcessOrderbook failed to retrieve new orderbook")
	}

	a, b := result.TotalAsksAmount()
	if a != 200 && b != 40000 {
		t.Fatal("Test failed. TestProcessOrderbook CalculateTotalsAsks incorrect values")
	}

	base.Bids = []Item{{Price: 420, Amount: 200}}
	base.ExchangeName = "Blah"
	base.AssetType = "quarterly"
	err = base.Process()
	if err != nil {
		t.Error("Test Failed - Process() error", err)
	}

	result, err = Get("Blah", c, "quarterly")
	if err != nil {
		t.Fatal("Test failed. TestProcessOrderbook failed to create new orderbook")
	}

	if a != 200 && b != 84000 {
		t.Fatal("Test failed. TestProcessOrderbook CalculateTotalsBids incorrect values")
	}

	type quick struct {
		Name string
		P    currency.Pair
		Bids []Item
		Asks []Item
	}

	var testArray []quick

	_ = rand.NewSource(time.Now().Unix())

	var wg sync.WaitGroup
	var m sync.Mutex

	var catastrophicFailure bool

	for i := 0; i < 500; i++ {
		if catastrophicFailure {
			break
		}

		wg.Add(1)
		go func() {
			newName := "Exchange" + strconv.FormatInt(rand.Int63(), 10)
			newPairs := currency.NewPair(currency.NewCode("BTC"+strconv.FormatInt(rand.Int63(), 10)),
				currency.NewCode("USD"+strconv.FormatInt(rand.Int63(), 10)))

			asks := []Item{{Price: rand.Float64(), Amount: rand.Float64()}}
			bids := []Item{{Price: rand.Float64(), Amount: rand.Float64()}}
			base := &Base{
				Pair:         newPairs,
				Asks:         asks,
				Bids:         bids,
				ExchangeName: newName,
				AssetType:    asset.Spot,
			}

			m.Lock()
			err = base.Process()
			if err != nil {
				t.Error(err)
				catastrophicFailure = true
				return
			}

			testArray = append(testArray, quick{Name: newName, P: newPairs, Bids: bids, Asks: asks})
			m.Unlock()
			wg.Done()
		}()
	}

	if catastrophicFailure {
		t.Fatal("Test Failed - Process() error", err)
	}

	wg.Wait()

	for _, test := range testArray {
		wg.Add(1)
		fatalErr := false
		go func(test quick) {
			result, err := Get(test.Name, test.P, asset.Spot)
			if err != nil {
				fatalErr = true
				return
			}

			if result.Asks[0] != test.Asks[0] {
				t.Error("Test failed. TestProcessOrderbook failed bad values")
			}

			if result.Bids[0] != test.Bids[0] {
				t.Error("Test failed. TestProcessOrderbook failed bad values")
			}

			wg.Done()
		}(test)

		if fatalErr {
			t.Fatal("Test failed. TestProcessOrderbook failed to retrieve new orderbook")
		}
	}

	wg.Wait()
}
