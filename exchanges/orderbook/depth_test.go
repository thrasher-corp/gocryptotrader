package orderbook

import (
	"errors"
	"log"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var id, _ = uuid.NewV4()

func TestGetLength(t *testing.T) {
	d := newDepth(id)
	if d.GetAskLength() != 0 {
		t.Errorf("expected len %v, but received %v", 0, d.GetAskLength())
	}

	d.asks.load([]Item{{Price: 1337}}, d.stack)

	if d.GetAskLength() != 1 {
		t.Errorf("expected len %v, but received %v", 1, d.GetAskLength())
	}

	d = newDepth(id)
	if d.GetBidLength() != 0 {
		t.Errorf("expected len %v, but received %v", 0, d.GetBidLength())
	}

	d.bids.load([]Item{{Price: 1337}}, d.stack)

	if d.GetBidLength() != 1 {
		t.Errorf("expected len %v, but received %v", 1, d.GetBidLength())
	}
}

func TestRetrieve(t *testing.T) {
	d := newDepth(id)
	d.asks.load([]Item{{Price: 1337}}, d.stack)
	d.bids.load([]Item{{Price: 1337}}, d.stack)
	d.options = options{
		exchange:         "THE BIG ONE!!!!!!",
		pair:             currency.NewPair(currency.THETA, currency.USD),
		asset:            "Silly asset",
		lastUpdated:      time.Now(),
		lastUpdateID:     007,
		priceDuplication: true,
		isFundingRate:    true,
		VerifyOrderbook:  true,
		restSnapshot:     true,
		idAligned:        true,
	}

	// If we add anymore options to the options struct later this will complain
	// generally want to return a full carbon copy
	mirrored := reflect.Indirect(reflect.ValueOf(d.options))
	for n := 0; n < mirrored.NumField(); n++ {
		structVal := mirrored.Field(n)
		if structVal.IsZero() {
			t.Fatalf("struct value options not set for field %v",
				mirrored.Type().Field(n).Name)
		}
	}
	theBigD := d.Retrieve()
	if len(theBigD.Asks) != 1 {
		t.Errorf("expected len %v, but received %v", 1, len(theBigD.Bids))
	}

	if len(theBigD.Bids) != 1 {
		t.Errorf("expected len %v, but received %v", 1, len(theBigD.Bids))
	}
}

func TestTotalAmounts(t *testing.T) {
	d := newDepth(id)

	liquidity, value := d.TotalBidAmounts()
	if liquidity != 0 || value != 0 {
		t.Fatalf("liquidity expected %f received %f value expected %f received %f",
			0.,
			liquidity,
			0.,
			value)
	}

	liquidity, value = d.TotalAskAmounts()
	if liquidity != 0 || value != 0 {
		t.Fatalf("liquidity expected %f received %f value expected %f received %f",
			0.,
			liquidity,
			0.,
			value)
	}

	d.asks.load([]Item{{Price: 1337, Amount: 1}}, d.stack)
	d.bids.load([]Item{{Price: 1337, Amount: 10}}, d.stack)

	liquidity, value = d.TotalBidAmounts()
	if liquidity != 10 || value != 13370 {
		t.Fatalf("liquidity expected %f received %f value expected %f received %f",
			10.,
			liquidity,
			13370.,
			value)
	}

	liquidity, value = d.TotalAskAmounts()
	if liquidity != 1 || value != 1337 {
		t.Fatalf("liquidity expected %f received %f value expected %f received %f",
			1.,
			liquidity,
			1337.,
			value)
	}
}

func TestLoadSnapshot(t *testing.T) {
	d := newDepth(id)
	d.LoadSnapshot(Items{{Price: 1337, Amount: 1}}, Items{{Price: 1337, Amount: 10}})
	if d.Retrieve().Asks[0].Price != 1337 || d.Retrieve().Bids[0].Price != 1337 {
		t.Fatal("not set")
	}
}

func TestFlush(t *testing.T) {
	d := newDepth(id)
	d.LoadSnapshot(Items{{Price: 1337, Amount: 1}}, Items{{Price: 1337, Amount: 10}})
	d.Flush()
	if len(d.Retrieve().Asks) != 0 || len(d.Retrieve().Bids) != 0 {
		t.Fatal("not flushed")
	}
	d.LoadSnapshot(Items{{Price: 1337, Amount: 1}}, Items{{Price: 1337, Amount: 10}})
	d.Flush()
	if len(d.Retrieve().Asks) != 0 || len(d.Retrieve().Bids) != 0 {
		t.Fatal("not flushed")
	}
}

func TestUpdateBidAskByPrice(t *testing.T) {
	d := newDepth(id)
	d.LoadSnapshot(Items{{Price: 1337, Amount: 1, ID: 1}}, Items{{Price: 1337, Amount: 10, ID: 2}})
	d.UpdateBidAskByPrice(Items{{Price: 1337, Amount: 2, ID: 1}}, Items{{Price: 1337, Amount: 2, ID: 2}}, 0)
	if d.Retrieve().Asks[0].Amount != 2 || d.Retrieve().Bids[0].Amount != 2 {
		t.Fatal("orderbook amounts not updated correctly")
	}
	d.UpdateBidAskByPrice(Items{{Price: 1337, Amount: 0, ID: 1}}, Items{{Price: 1337, Amount: 0, ID: 2}}, 0)
	if d.GetAskLength() != 0 || d.GetBidLength() != 0 {
		t.Fatal("orderbook amounts not updated correctly")
	}
}

func TestDeleteBidAskByID(t *testing.T) {
	d := newDepth(id)
	d.LoadSnapshot(Items{{Price: 1337, Amount: 1, ID: 1}}, Items{{Price: 1337, Amount: 10, ID: 2}})
	err := d.DeleteBidAskByID(Items{{Price: 1337, Amount: 2, ID: 1}}, Items{{Price: 1337, Amount: 2, ID: 2}}, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Retrieve().Asks) != 0 || len(d.Retrieve().Bids) != 0 {
		t.Fatal("items not deleted")
	}

	err = d.DeleteBidAskByID(Items{{Price: 1337, Amount: 2, ID: 1}}, nil, false)
	if !errors.Is(err, errIDCannotBeMatched) {
		t.Fatalf("error expected %v received %v", errIDCannotBeMatched, err)
	}

	err = d.DeleteBidAskByID(nil, Items{{Price: 1337, Amount: 2, ID: 2}}, false)
	if !errors.Is(err, errIDCannotBeMatched) {
		t.Fatalf("error expected %v received %v", errIDCannotBeMatched, err)
	}

	err = d.DeleteBidAskByID(nil, Items{{Price: 1337, Amount: 2, ID: 2}}, true)
	if !errors.Is(err, nil) {
		t.Fatalf("error expected %v received %v", nil, err)
	}
}

func TestUpdateBidAskByID(t *testing.T) {
	d := newDepth(id)
	d.LoadSnapshot(Items{{Price: 1337, Amount: 1, ID: 1}}, Items{{Price: 1337, Amount: 10, ID: 2}})
	err := d.UpdateBidAskByID(Items{{Price: 1337, Amount: 2, ID: 1}}, Items{{Price: 1337, Amount: 2, ID: 2}})
	if err != nil {
		t.Fatal(err)
	}
	if d.Retrieve().Asks[0].Amount != 2 || d.Retrieve().Bids[0].Amount != 2 {
		t.Fatal("orderbook amounts not updated correctly")
	}

	// random unmatching IDs
	err = d.UpdateBidAskByID(Items{{Price: 1337, Amount: 2, ID: 666}}, nil)
	if !errors.Is(err, errIDCannotBeMatched) {
		t.Fatalf("error expected %v received %v", errIDCannotBeMatched, err)
	}

	err = d.UpdateBidAskByID(nil, Items{{Price: 1337, Amount: 2, ID: 69}})
	if !errors.Is(err, errIDCannotBeMatched) {
		t.Fatalf("error expected %v received %v", errIDCannotBeMatched, err)
	}
}

func TestInsertBidAskByID(t *testing.T) {
	d := newDepth(id)
	d.LoadSnapshot(Items{{Price: 1337, Amount: 1, ID: 1}}, Items{{Price: 1337, Amount: 10, ID: 2}})
	err := d.InsertBidAskByID(Items{{Price: 1338, Amount: 2, ID: 3}}, Items{{Price: 1336, Amount: 2, ID: 4}})
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Retrieve().Asks) != 2 || len(d.Retrieve().Bids) != 2 {
		t.Fatal("items not added correctly")
	}
}

func TestUpdateInsertByID(t *testing.T) {
	d := newDepth(id)
	d.LoadSnapshot(Items{{Price: 1337, Amount: 1, ID: 1}}, Items{{Price: 1337, Amount: 10, ID: 2}})

	err := d.UpdateInsertByID(Items{{Price: 1338, Amount: 0, ID: 3}}, Items{{Price: 1336, Amount: 2, ID: 4}})
	if !errors.Is(err, errAmountCannotBeLessOrEqualToZero) {
		t.Fatalf("expected: %v but received: %v", errAmountCannotBeLessOrEqualToZero, err)
	}

	err = d.UpdateInsertByID(Items{{Price: 1338, Amount: 2, ID: 3}}, Items{{Price: 1336, Amount: 0, ID: 4}})
	if !errors.Is(err, errAmountCannotBeLessOrEqualToZero) {
		t.Fatalf("expected: %v but received: %v", errAmountCannotBeLessOrEqualToZero, err)
	}

	err = d.UpdateInsertByID(Items{{Price: 1338, Amount: 2, ID: 3}}, Items{{Price: 1336, Amount: 2, ID: 4}})
	if err != nil {
		t.Fatal(err)
	}

	if len(d.Retrieve().Asks) != 2 || len(d.Retrieve().Bids) != 2 {
		t.Fatal("items not added correctly")
	}
}

func TestAssignOptions(t *testing.T) {
	d := Depth{}
	cp := currency.NewPair(currency.LINK, currency.BTC)
	tn := time.Now()
	d.AssignOptions(&Base{
		Exchange:         "test",
		Pair:             cp,
		Asset:            asset.Spot,
		LastUpdated:      tn,
		LastUpdateID:     1337,
		PriceDuplication: true,
		IsFundingRate:    true,
		VerifyOrderbook:  true,
		RestSnapshot:     true,
		IDAlignment:      true,
	})

	if d.exchange != "test" ||
		d.pair != cp ||
		d.asset != asset.Spot ||
		d.lastUpdated != tn ||
		d.lastUpdateID != 1337 ||
		!d.priceDuplication ||
		!d.isFundingRate ||
		!d.VerifyOrderbook ||
		!d.restSnapshot ||
		!d.idAligned {
		t.Fatal("failed to set correctly")
	}
}

func TestSetLastUpdate(t *testing.T) {
	d := Depth{}
	tn := time.Now()
	d.SetLastUpdate(tn, 1337, true)
	if d.lastUpdated != tn ||
		d.lastUpdateID != 1337 ||
		!d.restSnapshot {
		t.Fatal("failed to set correctly")
	}
}

func TestGetName(t *testing.T) {
	d := Depth{}
	d.exchange = "test"
	if d.GetName() != "test" {
		t.Fatal("failed to get correct value")
	}
}

func TestIsRestSnapshot(t *testing.T) {
	d := Depth{}
	d.restSnapshot = true
	if !d.IsRestSnapshot() {
		t.Fatal("failed to set correctly")
	}
}

func TestLastUpdateID(t *testing.T) {
	d := Depth{}
	d.lastUpdateID = 1337
	if d.LastUpdateID() != 1337 {
		t.Fatal("failed to get correct value")
	}
}

func TestIsFundingRate(t *testing.T) {
	d := Depth{}
	d.isFundingRate = true
	if !d.IsFundingRate() {
		t.Fatal("failed to get correct value")
	}
}

func TestPublish(t *testing.T) {
	d := Depth{}
	d.Publish()
}

func TestWait(t *testing.T) {
	wait := Alert{}
	var wg sync.WaitGroup

	// standard alert
	wg.Add(100)
	for x := 0; x < 100; x++ {
		go func() {
			w := wait.Wait(nil)
			wg.Done()
			if <-w {
				log.Fatal("incorrect routine wait response for alert expecting false")
			}
			wg.Done()
		}()
	}

	wg.Wait()
	wg.Add(100)
	isLeaky(&wait, nil, t)
	wait.alert()
	wg.Wait()
	isLeaky(&wait, nil, t)

	// use kick
	ch := make(chan struct{})
	wg.Add(100)
	for x := 0; x < 100; x++ {
		go func() {
			w := wait.Wait(ch)
			wg.Done()
			if !<-w {
				log.Fatal("incorrect routine wait response for kick expecting true")
			}
			wg.Done()
		}()
	}
	wg.Wait()
	wg.Add(100)
	isLeaky(&wait, ch, t)
	close(ch)
	wg.Wait()
	ch = make(chan struct{})
	isLeaky(&wait, ch, t)

	// late receivers
	wg.Add(100)
	for x := 0; x < 100; x++ {
		go func(x int) {
			bb := wait.Wait(ch)
			wg.Done()
			if x%2 == 0 {
				time.Sleep(time.Millisecond * 5)
			}
			b := <-bb
			if b {
				log.Fatal("incorrect routine wait response since we call alert below; expecting false")
			}
			wg.Done()
		}(x)
	}
	wg.Wait()
	wg.Add(100)
	isLeaky(&wait, ch, t)
	wait.alert()
	wg.Wait()
	isLeaky(&wait, ch, t)
}

// isLeaky tests to see if the wait functionality is returning an abnormal
// channel that is operational when it shouldn't be.
func isLeaky(a *Alert, ch chan struct{}, t *testing.T) {
	t.Helper()
	check := a.Wait(ch)
	time.Sleep(time.Millisecond * 5) // When we call wait a routine for hold is
	// spawned, so for a test we need to add in a time for goschedular to allow
	// routine to actually wait on the forAlert and kick channels
	select {
	case <-check:
		t.Fatal("leaky waiter")
	default:
	}
}
