package orderbook

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
	err := dispatch.Start(1, dispatch.DefaultJobsLimit)
	if err != nil {
		log.Fatal(err)
	}

	cpyMux = service.mux

	os.Exit(m.Run())
}

var cpyMux *dispatch.Mux

func TestSubscribeOrderbook(t *testing.T) {
	_, err := SubscribeOrderbook("", currency.Pair{}, asset.Item(""))
	if err == nil {
		t.Error("error cannot be nil")
	}

	p := currency.NewPair(currency.BTC, currency.USD)

	b := Base{
		Pair:      p,
		AssetType: asset.Spot,
	}

	err = b.Process()
	if err == nil {
		t.Error("error cannot be nil")
	}

	b.ExchangeName = "SubscribeOBTest"

	err = b.Process()
	if err == nil {
		t.Error("error cannot be nil")
	}

	b.Bids = []Item{{}}

	err = b.Process()
	if err != nil {
		t.Error("process error", err)
	}

	_, err = SubscribeOrderbook("SubscribeOBTest", p, asset.Spot)
	if err != nil {
		t.Error("error cannot be nil")
	}

	// process redundant update
	err = b.Process()
	if err != nil {
		t.Error("process error", err)
	}
}

func TestUpdateBooks(t *testing.T) {
	p := currency.NewPair(currency.BTC, currency.USD)

	b := Base{
		Pair:         p,
		AssetType:    asset.Spot,
		ExchangeName: "UpdateTest",
	}

	service.mux = nil

	err := service.Update(&b)
	if err == nil {
		t.Error("error cannot be nil")
	}

	b.Pair.Base = currency.CYC
	err = service.Update(&b)
	if err == nil {
		t.Error("error cannot be nil")
	}

	b.Pair.Quote = currency.ENAU
	err = service.Update(&b)
	if err == nil {
		t.Error("error cannot be nil")
	}

	b.AssetType = "unicorns"
	err = service.Update(&b)
	if err == nil {
		t.Error("error cannot be nil")
	}

	service.mux = cpyMux
}

func TestSubscribeToExchangeOrderbooks(t *testing.T) {
	_, err := SubscribeToExchangeOrderbooks("")
	if err == nil {
		t.Error("error cannot be nil")
	}

	p := currency.NewPair(currency.BTC, currency.USD)

	b := Base{
		Pair:         p,
		AssetType:    asset.Spot,
		ExchangeName: "SubscribeToExchangeOrderbooks",
		Bids:         []Item{{}},
	}

	err = b.Process()
	if err != nil {
		t.Error(err)
	}

	_, err = SubscribeToExchangeOrderbooks("SubscribeToExchangeOrderbooks")
	if err != nil {
		t.Error(err)
	}
}

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
	curr, err := currency.NewPairFromStrings("BTC", "USD")
	if err != nil {
		t.Fatal(err)
	}
	base := Base{
		Pair:        curr,
		Bids:        []Item{{Price: 100, Amount: 10}},
		LastUpdated: time.Now(),
	}

	a, b := base.TotalBidsAmount()
	if a != 10 && b != 1000 {
		t.Fatal("TestCalculateTotalBids expected a = 10 and b = 1000")
	}
}

func TestCalculateTotaAsks(t *testing.T) {
	t.Parallel()
	curr, err := currency.NewPairFromStrings("BTC", "USD")
	if err != nil {
		t.Fatal(err)
	}
	base := Base{
		Pair: curr,
		Asks: []Item{{Price: 100, Amount: 10}},
	}

	a, b := base.TotalAsksAmount()
	if a != 10 && b != 1000 {
		t.Fatal("TestCalculateTotalAsks expected a = 10 and b = 1000")
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()
	curr, err := currency.NewPairFromStrings("BTC", "USD")
	if err != nil {
		t.Fatal(err)
	}
	timeNow := time.Now()
	base := Base{
		Pair:        curr,
		Asks:        []Item{{Price: 100, Amount: 10}},
		Bids:        []Item{{Price: 200, Amount: 10}},
		LastUpdated: timeNow,
	}

	asks := []Item{{Price: 200, Amount: 101}}
	bids := []Item{{Price: 201, Amount: 100}}
	time.Sleep(time.Millisecond * 50)
	base.Update(bids, asks)

	if !base.LastUpdated.After(timeNow) {
		t.Fatal("TestUpdate expected LastUpdated to be greater then original time")
	}

	a, b := base.TotalAsksAmount()
	if a != 100 && b != 20200 {
		t.Fatal("TestUpdate expected a = 100 and b = 20100")
	}

	a, b = base.TotalBidsAmount()
	if a != 100 && b != 20100 {
		t.Fatal("TestUpdate expected a = 100 and b = 20100")
	}
}

func TestGetOrderbook(t *testing.T) {
	c, err := currency.NewPairFromStrings("BTC", "USD")
	if err != nil {
		t.Fatal(err)
	}
	base := &Base{
		Pair:         c,
		Asks:         []Item{{Price: 100, Amount: 10}},
		Bids:         []Item{{Price: 200, Amount: 10}},
		ExchangeName: "Exchange",
		AssetType:    asset.Spot,
	}

	err = base.Process()
	if err != nil {
		t.Fatal(err)
	}

	result, err := Get("Exchange", c, asset.Spot)
	if err != nil {
		t.Fatalf("TestGetOrderbook failed to get orderbook. Error %s",
			err)
	}
	if !result.Pair.Equal(c) {
		t.Fatal("TestGetOrderbook failed. Mismatched pairs")
	}

	_, err = Get("nonexistent", c, asset.Spot)
	if err == nil {
		t.Fatal("TestGetOrderbook retrieved non-existent orderbook")
	}

	c.Base = currency.NewCode("blah")
	_, err = Get("Exchange", c, asset.Spot)
	if err == nil {
		t.Fatal("TestGetOrderbook retrieved non-existent orderbook using invalid first currency")
	}

	newCurrency, err := currency.NewPairFromStrings("BTC", "AUD")
	if err != nil {
		t.Fatal(err)
	}
	_, err = Get("Exchange", newCurrency, asset.Spot)
	if err == nil {
		t.Fatal("TestGetOrderbook retrieved non-existent orderbook using invalid second currency")
	}

	base.Pair = newCurrency
	err = base.Process()
	if err != nil {
		t.Error(err)
	}

	_, err = Get("Exchange", newCurrency, "meowCats")
	if err == nil {
		t.Error("error cannot be nil")
	}
}

func TestCreateNewOrderbook(t *testing.T) {
	c, err := currency.NewPairFromStrings("BTC", "USD")
	if err != nil {
		t.Fatal(err)
	}
	base := &Base{
		Pair:         c,
		Asks:         []Item{{Price: 100, Amount: 10}},
		Bids:         []Item{{Price: 200, Amount: 10}},
		ExchangeName: "testCreateNewOrderbook",
		AssetType:    asset.Spot,
	}

	err = base.Process()
	if err != nil {
		t.Fatal(err)
	}

	result, err := Get("testCreateNewOrderbook", c, asset.Spot)
	if err != nil {
		t.Fatal("TestCreateNewOrderbook failed to create new orderbook", err)
	}

	if !result.Pair.Equal(c) {
		t.Fatal("TestCreateNewOrderbook result pair is incorrect")
	}

	a, b := result.TotalAsksAmount()
	if a != 10 && b != 1000 {
		t.Fatal("TestCreateNewOrderbook CalculateTotalAsks value is incorrect")
	}

	a, b = result.TotalBidsAmount()
	if a != 10 && b != 2000 {
		t.Fatal("TestCreateNewOrderbook CalculateTotalBids value is incorrect")
	}
}

func TestProcessOrderbook(t *testing.T) {
	c, err := currency.NewPairFromStrings("BTC", "USD")
	if err != nil {
		t.Fatal(err)
	}
	base := Base{
		Asks:         []Item{{Price: 100, Amount: 10}},
		Bids:         []Item{{Price: 200, Amount: 10}},
		ExchangeName: "ProcessOrderbook",
	}

	// test for empty pair
	base.Pair = currency.Pair{}
	err = base.Process()
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
		t.Fatal("TestProcessOrderbook failed to create new orderbook")
	}
	if !result.Pair.Equal(c) {
		t.Fatal("TestProcessOrderbook result pair is incorrect")
	}

	// now test for processing a pair with a different quote currency
	c, err = currency.NewPairFromStrings("BTC", "GBP")
	if err != nil {
		t.Fatal(err)
	}
	base.Pair = c
	err = base.Process()
	if err != nil {
		t.Error("Process() error", err)
	}
	result, err = Get("ProcessOrderbook", c, asset.Spot)
	if err != nil {
		t.Fatal("TestProcessOrderbook failed to retrieve new orderbook")
	}
	if !result.Pair.Equal(c) {
		t.Fatal("TestProcessOrderbook result pair is incorrect")
	}

	// now test for processing a pair which has a different base currency
	c, err = currency.NewPairFromStrings("LTC", "GBP")
	if err != nil {
		t.Fatal(err)
	}
	base.Pair = c
	err = base.Process()
	if err != nil {
		t.Error("Process() error", err)
	}
	result, err = Get("ProcessOrderbook", c, asset.Spot)
	if err != nil {
		t.Fatal("TestProcessOrderbook failed to retrieve new orderbook")
	}
	if !result.Pair.Equal(c) {
		t.Fatal("TestProcessOrderbook result pair is incorrect")
	}

	base.Asks = []Item{{Price: 200, Amount: 200}}
	base.AssetType = "monthly"
	err = base.Process()
	if err != nil {
		t.Error("Process() error", err)
	}

	result, err = Get("ProcessOrderbook", c, "monthly")
	if err != nil {
		t.Fatal("TestProcessOrderbook failed to retrieve new orderbook")
	}

	a, b := result.TotalAsksAmount()
	if a != 200 && b != 40000 {
		t.Fatal("TestProcessOrderbook CalculateTotalsAsks incorrect values")
	}

	base.Bids = []Item{{Price: 420, Amount: 200}}
	base.ExchangeName = "Blah"
	base.AssetType = "quarterly"
	err = base.Process()
	if err != nil {
		t.Error("Process() error", err)
	}

	_, err = Get("Blah", c, "quarterly")
	if err != nil {
		t.Fatal("TestProcessOrderbook failed to create new orderbook")
	}

	if a != 200 && b != 84000 {
		t.Fatal("TestProcessOrderbook CalculateTotalsBids incorrect values")
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
			newName := "Exchange" + strconv.FormatInt(rand.Int63(), 10) // nolint:gosec // no need to import crypo/rand for testing
			newPairs := currency.NewPair(currency.NewCode("BTC"+strconv.FormatInt(rand.Int63(), 10)),
				currency.NewCode("USD"+strconv.FormatInt(rand.Int63(), 10))) // nolint:gosec // no need to import crypo/rand for testing

			asks := []Item{{Price: rand.Float64(), Amount: rand.Float64()}} // nolint:gosec // no need to import crypo/rand for testing
			bids := []Item{{Price: rand.Float64(), Amount: rand.Float64()}} // nolint:gosec // no need to import crypo/rand for testing
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
		t.Fatal("Process() error", err)
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
				t.Error("TestProcessOrderbook failed bad values")
			}

			if result.Bids[0] != test.Bids[0] {
				t.Error("TestProcessOrderbook failed bad values")
			}

			wg.Done()
		}(test)

		if fatalErr {
			t.Fatal("TestProcessOrderbook failed to retrieve new orderbook")
		}
	}

	wg.Wait()
}

func TestSetNewData(t *testing.T) {
	err := service.SetNewData(nil, "")
	if err == nil {
		t.Error("error cannot be nil")
	}
}

func TestGetAssociations(t *testing.T) {
	_, err := service.GetAssociations(nil, "")
	if err == nil {
		t.Error("error cannot be nil")
	}
}
