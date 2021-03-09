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
		exchange:              "THE BIG ONE!!!!!!",
		pair:                  currency.NewPair(currency.THETA, currency.USD),
		asset:                 "Silly asset",
		lastUpdated:           time.Now(),
		lastUpdateID:          007,
		notAggregated:         true,
		isFundingRate:         true,
		verificationBypass:    true,
		hasChecksumValidation: true,
		restSnapshot:          true,
		idAligned:             true,
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
	d.InsertBidAskByID(Items{{Price: 1338, Amount: 2, ID: 3}}, Items{{Price: 1336, Amount: 2, ID: 4}})

	if len(d.Retrieve().Asks) != 2 || len(d.Retrieve().Bids) != 2 {
		t.Fatal("items not added correctly")
	}
}

func TestUpdateInsertByID(t *testing.T) {
	d := newDepth(id)
	d.LoadSnapshot(Items{{Price: 1337, Amount: 1, ID: 1}}, Items{{Price: 1337, Amount: 10, ID: 2}})
	d.UpdateInsertByID(Items{{Price: 1338, Amount: 2, ID: 3}}, Items{{Price: 1336, Amount: 2, ID: 4}})

	if len(d.Retrieve().Asks) != 2 || len(d.Retrieve().Bids) != 2 {
		t.Fatal("items not added correctly")
	}
}

func TestAlert(t *testing.T) {
	d := newDepth(id)
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

func TestAssignOptions(t *testing.T) {
	d := Depth{}
	cp := currency.NewPair(currency.LINK, currency.BTC)
	tn := time.Now()
	d.AssignOptions(&Base{
		Exchange:              "test",
		Pair:                  cp,
		Asset:                 asset.Spot,
		LastUpdated:           tn,
		LastUpdateID:          1337,
		NotAggregated:         true,
		IsFundingRate:         true,
		VerificationBypass:    true,
		HasChecksumValidation: true,
		RestSnapshot:          true,
		IDAlignment:           true,
	})

	if d.exchange != "test" ||
		d.pair != cp ||
		d.asset != asset.Spot ||
		d.lastUpdated != tn ||
		d.lastUpdateID != 1337 ||
		!d.notAggregated ||
		!d.isFundingRate ||
		!d.verificationBypass ||
		!d.hasChecksumValidation ||
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
