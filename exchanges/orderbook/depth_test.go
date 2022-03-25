package orderbook

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var id, _ = uuid.NewV4()

func TestGetLength(t *testing.T) {
	t.Parallel()
	d := NewDepth(id)
	_, err := d.GetAskLength()
	if !errors.Is(err, ErrOrderbookInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderbookInvalid)
	}

	d.LoadSnapshot([]Item{{Price: 1337}}, nil, 0, time.Time{}, true)

	askLen, err := d.GetAskLength()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if askLen != 0 {
		t.Errorf("expected len %v, but received %v", 0, askLen)
	}

	d.asks.load([]Item{{Price: 1337}}, d.stack)

	askLen, err = d.GetAskLength()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if askLen != 1 {
		t.Errorf("expected len %v, but received %v", 1, askLen)
	}

	d = NewDepth(id)
	_, err = d.GetBidLength()
	if !errors.Is(err, ErrOrderbookInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderbookInvalid)
	}

	d.LoadSnapshot(nil, []Item{{Price: 1337}}, 0, time.Time{}, true)

	bidLen, err := d.GetBidLength()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if bidLen != 0 {
		t.Errorf("expected len %v, but received %v", 0, bidLen)
	}

	d.bids.load([]Item{{Price: 1337}}, d.stack)

	bidLen, err = d.GetBidLength()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if bidLen != 1 {
		t.Errorf("expected len %v, but received %v", 1, bidLen)
	}
}

func TestRetrieve(t *testing.T) {
	t.Parallel()
	d := NewDepth(id)
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
		isValid:          true, // not needed but here to pass test
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

	ob, err := d.Retrieve()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if len(ob.Asks) != 1 {
		t.Errorf("expected len %v, but received %v", 1, len(ob.Bids))
	}

	if len(ob.Bids) != 1 {
		t.Errorf("expected len %v, but received %v", 1, len(ob.Bids))
	}
}

func TestTotalAmounts(t *testing.T) {
	t.Parallel()
	d := NewDepth(id)

	_, _, err := d.TotalBidAmounts()
	if !errors.Is(err, ErrOrderbookInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderbookInvalid)
	}

	d.isValid = true

	liquidity, value, err := d.TotalBidAmounts()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if liquidity != 0 || value != 0 {
		t.Fatalf("liquidity expected %f received %f value expected %f received %f",
			0.,
			liquidity,
			0.,
			value)
	}

	liquidity, value, err = d.TotalAskAmounts()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if liquidity != 0 || value != 0 {
		t.Fatalf("liquidity expected %f received %f value expected %f received %f",
			0.,
			liquidity,
			0.,
			value)
	}

	d.asks.load([]Item{{Price: 1337, Amount: 1}}, d.stack)
	d.bids.load([]Item{{Price: 1337, Amount: 10}}, d.stack)

	liquidity, value, err = d.TotalBidAmounts()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if liquidity != 10 || value != 13370 {
		t.Fatalf("liquidity expected %f received %f value expected %f received %f",
			10.,
			liquidity,
			13370.,
			value)
	}

	liquidity, value, err = d.TotalAskAmounts()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if liquidity != 1 || value != 1337 {
		t.Fatalf("liquidity expected %f received %f value expected %f received %f",
			1.,
			liquidity,
			1337.,
			value)
	}
}

func TestLoadSnapshot(t *testing.T) {
	t.Parallel()
	d := NewDepth(id)
	d.LoadSnapshot(Items{{Price: 1337, Amount: 1}}, Items{{Price: 1337, Amount: 10}}, 0, time.Time{}, false)

	ob, err := d.Retrieve()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if ob.Asks[0].Price != 1337 || ob.Bids[0].Price != 1337 {
		t.Fatalf("not set")
	}
}

func TestInvalidate(t *testing.T) {
	t.Parallel()
	d := NewDepth(id)
	d.LoadSnapshot(Items{{Price: 1337, Amount: 1}}, Items{{Price: 1337, Amount: 10}}, 0, time.Time{}, false)

	ob, err := d.Retrieve()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if ob == nil {
		t.Fatalf("unexpected value")
	}

	d.Invalidate()

	_, err = d.Retrieve()
	if !errors.Is(err, ErrOrderbookInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderbookInvalid)
	}

	d.isValid = true

	ob, err = d.Retrieve()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if len(ob.Asks) != 0 || len(ob.Bids) != 0 {
		t.Fatalf("not flushed")
	}
}

func TestUpdateBidAskByPrice(t *testing.T) {
	t.Parallel()
	d := NewDepth(id)
	d.LoadSnapshot(Items{{Price: 1337, Amount: 1, ID: 1}}, Items{{Price: 1337, Amount: 10, ID: 2}}, 0, time.Time{}, false)

	// empty
	d.UpdateBidAskByPrice(nil, nil, 0, 1, time.Time{})

	d.UpdateBidAskByPrice(Items{{Price: 1337, Amount: 2, ID: 1}}, Items{{Price: 1337, Amount: 2, ID: 2}}, 0, 1, time.Time{})

	ob, err := d.Retrieve()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if ob.Asks[0].Amount != 2 || ob.Bids[0].Amount != 2 {
		t.Fatalf("orderbook amounts not updated correctly")
	}

	d.UpdateBidAskByPrice(Items{{Price: 1337, Amount: 0, ID: 1}}, Items{{Price: 1337, Amount: 0, ID: 2}}, 0, 2, time.Time{})

	askLen, err := d.GetAskLength()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	bidLen, err := d.GetBidLength()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if askLen != 0 || bidLen != 0 {
		t.Fatalf("orderbook amounts not updated correctly")
	}
}

func TestDeleteBidAskByID(t *testing.T) {
	t.Parallel()
	d := NewDepth(id)
	d.LoadSnapshot(Items{{Price: 1337, Amount: 1, ID: 1}}, Items{{Price: 1337, Amount: 10, ID: 2}}, 0, time.Time{}, false)
	err := d.DeleteBidAskByID(Items{{Price: 1337, Amount: 2, ID: 1}}, Items{{Price: 1337, Amount: 2, ID: 2}}, false, 0, time.Time{})
	if err != nil {
		t.Fatal(err)
	}

	ob, err := d.Retrieve()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if len(ob.Asks) != 0 || len(ob.Bids) != 0 {
		t.Fatalf("items not deleted")
	}

	err = d.DeleteBidAskByID(Items{{Price: 1337, Amount: 2, ID: 1}}, nil, false, 0, time.Time{})
	if !errors.Is(err, errIDCannotBeMatched) {
		t.Fatalf("error expected %v received %v", errIDCannotBeMatched, err)
	}

	err = d.DeleteBidAskByID(nil, Items{{Price: 1337, Amount: 2, ID: 2}}, false, 0, time.Time{})
	if !errors.Is(err, errIDCannotBeMatched) {
		t.Fatalf("error expected %v received %v", errIDCannotBeMatched, err)
	}

	err = d.DeleteBidAskByID(nil, Items{{Price: 1337, Amount: 2, ID: 2}}, true, 0, time.Time{})
	if !errors.Is(err, nil) {
		t.Fatalf("error expected %v received %v", nil, err)
	}
}

func TestUpdateBidAskByID(t *testing.T) {
	t.Parallel()
	d := NewDepth(id)
	d.LoadSnapshot(Items{{Price: 1337, Amount: 1, ID: 1}}, Items{{Price: 1337, Amount: 10, ID: 2}}, 0, time.Time{}, false)
	err := d.UpdateBidAskByID(Items{{Price: 1337, Amount: 2, ID: 1}}, Items{{Price: 1337, Amount: 2, ID: 2}}, 0, time.Time{})
	if err != nil {
		t.Fatal(err)
	}

	ob, err := d.Retrieve()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if ob.Asks[0].Amount != 2 || ob.Bids[0].Amount != 2 {
		t.Fatalf("orderbook amounts not updated correctly")
	}

	// random unmatching IDs
	err = d.UpdateBidAskByID(Items{{Price: 1337, Amount: 2, ID: 666}}, nil, 0, time.Time{})
	if !errors.Is(err, errIDCannotBeMatched) {
		t.Fatalf("error expected %v received %v", errIDCannotBeMatched, err)
	}

	err = d.UpdateBidAskByID(nil, Items{{Price: 1337, Amount: 2, ID: 69}}, 0, time.Time{})
	if !errors.Is(err, errIDCannotBeMatched) {
		t.Fatalf("error expected %v received %v", errIDCannotBeMatched, err)
	}
}

func TestInsertBidAskByID(t *testing.T) {
	t.Parallel()
	d := NewDepth(id)
	d.LoadSnapshot(Items{{Price: 1337, Amount: 1, ID: 1}}, Items{{Price: 1337, Amount: 10, ID: 2}}, 0, time.Time{}, false)
	err := d.InsertBidAskByID(Items{{Price: 1338, Amount: 2, ID: 3}}, Items{{Price: 1336, Amount: 2, ID: 4}}, 0, time.Time{})
	if err != nil {
		t.Fatal(err)
	}

	ob, err := d.Retrieve()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if len(ob.Asks) != 2 || len(ob.Bids) != 2 {
		t.Fatalf("items not added correctly")
	}
}

func TestUpdateInsertByID(t *testing.T) {
	t.Parallel()
	d := NewDepth(id)
	d.LoadSnapshot(Items{{Price: 1337, Amount: 1, ID: 1}}, Items{{Price: 1337, Amount: 10, ID: 2}}, 0, time.Time{}, false)
	err := d.UpdateInsertByID(Items{{Price: 1338, Amount: 0, ID: 3}}, Items{{Price: 1336, Amount: 2, ID: 4}}, 0, time.Time{})
	if !errors.Is(err, errAmountCannotBeLessOrEqualToZero) {
		t.Fatalf("expected: %v but received: %v", errAmountCannotBeLessOrEqualToZero, err)
	}

	// Above will invalidate the book
	_, err = d.Retrieve()
	if !errors.Is(err, ErrOrderbookInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderbookInvalid)
	}

	d.LoadSnapshot(Items{{Price: 1337, Amount: 1, ID: 1}}, Items{{Price: 1337, Amount: 10, ID: 2}}, 0, time.Time{}, false)
	err = d.UpdateInsertByID(Items{{Price: 1338, Amount: 2, ID: 3}}, Items{{Price: 1336, Amount: 0, ID: 4}}, 0, time.Time{})
	if !errors.Is(err, errAmountCannotBeLessOrEqualToZero) {
		t.Fatalf("expected: %v but received: %v", errAmountCannotBeLessOrEqualToZero, err)
	}

	// Above will invalidate the book
	_, err = d.Retrieve()
	if !errors.Is(err, ErrOrderbookInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderbookInvalid)
	}

	d.LoadSnapshot(Items{{Price: 1337, Amount: 1, ID: 1}}, Items{{Price: 1337, Amount: 10, ID: 2}}, 0, time.Time{}, false)
	err = d.UpdateInsertByID(Items{{Price: 1338, Amount: 2, ID: 3}}, Items{{Price: 1336, Amount: 2, ID: 4}}, 0, time.Time{})
	if err != nil {
		t.Fatal(err)
	}

	ob, err := d.Retrieve()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if len(ob.Asks) != 2 || len(ob.Bids) != 2 {
		t.Fatalf("items not added correctly")
	}
}

func TestAssignOptions(t *testing.T) {
	t.Parallel()
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
		t.Fatalf("failed to set correctly")
	}
}

func TestGetName(t *testing.T) {
	t.Parallel()
	d := Depth{}
	d.exchange = "test"
	if d.GetName() != "test" {
		t.Fatalf("failed to get correct value")
	}
}

func TestIsRestSnapshot(t *testing.T) {
	t.Parallel()
	d := Depth{}
	d.restSnapshot = true
	if _, err := d.IsRestSnapshot(); !errors.Is(err, ErrOrderbookInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderbookInvalid)
	}

	d.isValid = true
	b, err := d.IsRestSnapshot()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if !b {
		t.Fatalf("received: '%v' but expected: '%v'", b, true)
	}
}

func TestLastUpdateID(t *testing.T) {
	t.Parallel()
	d := Depth{}
	d.lastUpdateID = 1337
	if _, err := d.LastUpdateID(); !errors.Is(err, ErrOrderbookInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderbookInvalid)
	}

	d.isValid = true
	id, err := d.LastUpdateID()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if id != 1337 {
		t.Fatalf("received: '%v' but expected: '%v'", id, 1337)
	}
}

func TestIsFundingRate(t *testing.T) {
	t.Parallel()
	d := Depth{}
	d.isFundingRate = true
	if !d.IsFundingRate() {
		t.Fatalf("failed to get correct value")
	}
}

func TestPublish(t *testing.T) {
	t.Parallel()
	d := Depth{}
	d.Publish()
	d.isValid = true
	d.Publish()
}
