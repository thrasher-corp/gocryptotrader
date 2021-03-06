package orderbook

import (
	"errors"
	"log"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

func TestGetLength(t *testing.T) {
	d := newDepth()
	if d.GetAskLength() != 0 {
		t.Errorf("expected len %v, bu received %v", 0, d.GetAskLength())
	}

	d.asks.load([]Item{{Price: 1337}}, d.stack)

	if d.GetAskLength() != 1 {
		t.Errorf("expected len %v, bu received %v", 1, d.GetAskLength())
	}

	d = newDepth()
	if d.GetBidLength() != 0 {
		t.Errorf("expected len %v, bu received %v", 0, d.GetBidLength())
	}

	d.bids.load([]Item{{Price: 1337}}, d.stack)

	if d.GetBidLength() != 1 {
		t.Errorf("expected len %v, bu received %v", 1, d.GetBidLength())
	}
}

func TestRetrieve(t *testing.T) {
	d := newDepth()
	d.asks.load([]Item{{Price: 1337}}, d.stack)
	d.bids.load([]Item{{Price: 1337}}, d.stack)
	d.options = options{
		Exchange:              "THE BIG ONE!!!!!!",
		Pair:                  currency.NewPair(currency.THETA, currency.USD),
		Asset:                 "Silly asset",
		LastUpdated:           time.Now(),
		LastUpdateID:          007,
		NotAggregated:         true,
		IsFundingRate:         true,
		VerificationBypass:    true,
		HasChecksumValidation: true,
		RestSnapshot:          true,
		IDAligned:             true,
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
	// retreieve the d lol
	theBigD := d.Retrieve()
	if len(theBigD.Asks) != 1 {
		t.Errorf("expected len %v, bu received %v", 1, len(theBigD.Bids))
	}

	if len(theBigD.Bids) != 1 {
		t.Errorf("expected len %v, bu received %v", 1, len(theBigD.Bids))
	}
}

func TestTotalAmounts(t *testing.T) {
	d := newDepth()

	liquidity, value := d.TotalBidAmounts()
	if liquidity != 0 || value != 0 {
		t.Fatalf("liquidity expected %f receieved %f value expected %f receieved %f",
			0.,
			liquidity,
			0.,
			value)
	}

	liquidity, value = d.TotalAskAmounts()
	if liquidity != 0 || value != 0 {
		t.Fatalf("liquidity expected %f receieved %f value expected %f receieved %f",
			0.,
			liquidity,
			0.,
			value)
	}

	d.asks.load([]Item{{Price: 1337, Amount: 1}}, d.stack)
	d.bids.load([]Item{{Price: 1337, Amount: 10}}, d.stack)

	liquidity, value = d.TotalBidAmounts()
	if liquidity != 10 || value != 13370 {
		t.Fatalf("liquidity expected %f receieved %f value expected %f receieved %f",
			10.,
			liquidity,
			13370.,
			value)
	}

	liquidity, value = d.TotalAskAmounts()
	if liquidity != 1 || value != 1337 {
		t.Fatalf("liquidity expected %f receieved %f value expected %f receieved %f",
			1.,
			liquidity,
			1337.,
			value)
	}
}

func TestLoadSnapshot(t *testing.T) {
	d := newDepth()
	d.LoadSnapshot(Items{{Price: 1337, Amount: 1}}, Items{{Price: 1337, Amount: 10}})
	if d.Retrieve().Asks[0].Price != 1337 || d.Retrieve().Bids[0].Price != 1337 {
		t.Fatal("not set")
	}
}

func TestFlush(t *testing.T) {
	d := newDepth()
	d.LoadSnapshot(Items{{Price: 1337, Amount: 1}}, Items{{Price: 1337, Amount: 10}})
	d.flush()
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
	d := newDepth()
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
	d := newDepth()
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
		t.Fatalf("error expected %v receieved %v", errIDCannotBeMatched, err)
	}

	err = d.DeleteBidAskByID(nil, Items{{Price: 1337, Amount: 2, ID: 2}}, false)
	if !errors.Is(err, errIDCannotBeMatched) {
		t.Fatalf("error expected %v receieved %v", errIDCannotBeMatched, err)
	}

	err = d.DeleteBidAskByID(nil, Items{{Price: 1337, Amount: 2, ID: 2}}, true)
	if !errors.Is(err, nil) {
		t.Fatalf("error expected %v receieved %v", nil, err)
	}
}

func TestUpdateBidAskByID(t *testing.T) {
	d := newDepth()
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
		t.Fatalf("error expected %v receieved %v", errIDCannotBeMatched, err)
	}

	err = d.UpdateBidAskByID(nil, Items{{Price: 1337, Amount: 2, ID: 69}})
	if !errors.Is(err, errIDCannotBeMatched) {
		t.Fatalf("error expected %v receieved %v", errIDCannotBeMatched, err)
	}
}

func TestInsertBidAskByID(t *testing.T) {
	d := newDepth()
	d.LoadSnapshot(Items{{Price: 1337, Amount: 1, ID: 1}}, Items{{Price: 1337, Amount: 10, ID: 2}})
	d.InsertBidAskByID(Items{{Price: 1338, Amount: 2, ID: 3}}, Items{{Price: 1336, Amount: 2, ID: 4}})

	if len(d.Retrieve().Asks) != 2 || len(d.Retrieve().Bids) != 2 {
		t.Fatal("items not added correctly")
	}
}

func TestUpdateInsertByID(t *testing.T) {
	d := newDepth()
	d.LoadSnapshot(Items{{Price: 1337, Amount: 1, ID: 1}}, Items{{Price: 1337, Amount: 10, ID: 2}})
	d.UpdateInsertByID(Items{{Price: 1338, Amount: 2, ID: 3}}, Items{{Price: 1336, Amount: 2, ID: 4}})

	if len(d.Retrieve().Asks) != 2 || len(d.Retrieve().Bids) != 2 {
		t.Fatal("items not added correctly")
	}
}

func TestAlert(t *testing.T) {
	d := newDepth()
	d.alert()

	var wg sync.WaitGroup
	wg.Add(5)
	var kick = timeInForce(0)
	for i := 0; i < 5; i++ {
		go func() {
			if d.Wait(kick) {
				log.Fatal("expected routine to be kicked by channel")
			}
		}()
	}
	var wait sync.WaitGroup
	wait.Add(5)
	for i := 0; i < 5; i++ {
		go func() {
			wait.Done()
			if d.Wait(nil) {
				wg.Done()
			}
		}()
	}
	wait.Wait()
	d.alert()
	wg.Wait()
}
