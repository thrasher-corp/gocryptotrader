package orderbook

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var id = uuid.Must(uuid.NewV4())

func TestGetLength(t *testing.T) {
	t.Parallel()
	d := NewDepth(id)
	err := d.Invalidate(nil)
	if !errors.Is(err, ErrOrderbookInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderbookInvalid)
	}
	_, err = d.GetAskLength()
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
	err = d.Invalidate(nil)
	if !errors.Is(err, ErrOrderbookInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderbookInvalid)
	}
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
		asset:            asset.DownsideProfitContract,
		lastUpdated:      time.Now(),
		lastUpdateID:     1337,
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

	err := d.Invalidate(nil)
	if !errors.Is(err, ErrOrderbookInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderbookInvalid)
	}
	_, _, err = d.TotalBidAmounts()
	if !errors.Is(err, ErrOrderbookInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderbookInvalid)
	}

	d.validationError = nil
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

	err = d.Invalidate(nil)
	if !errors.Is(err, ErrOrderbookInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderbookInvalid)
	}

	_, _, err = d.TotalAskAmounts()
	if !errors.Is(err, ErrOrderbookInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderbookInvalid)
	}

	d.validationError = nil

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
	d.exchange = "testexchange"
	d.pair = currency.NewPair(currency.BTC, currency.WABI)
	d.asset = asset.Spot
	d.LoadSnapshot(Items{{Price: 1337, Amount: 1}}, Items{{Price: 1337, Amount: 10}}, 0, time.Time{}, false)

	ob, err := d.Retrieve()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if ob == nil {
		t.Fatalf("unexpected value")
	}

	err = d.Invalidate(errors.New("random reason"))
	if !errors.Is(err, ErrOrderbookInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderbookInvalid)
	}

	_, err = d.Retrieve()
	if !errors.Is(err, ErrOrderbookInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderbookInvalid)
	}

	if err.Error() != "testexchange BTCWABI spot orderbook data integrity compromised Reason: [random reason]" {
		t.Fatal("unexpected string return")
	}

	d.validationError = nil

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
	d.UpdateBidAskByPrice(&Update{})

	updates := &Update{
		Bids:     Items{{Price: 1337, Amount: 2, ID: 1}},
		Asks:     Items{{Price: 1337, Amount: 2, ID: 2}},
		UpdateID: 1,
	}
	d.UpdateBidAskByPrice(updates)

	ob, err := d.Retrieve()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if ob.Asks[0].Amount != 2 || ob.Bids[0].Amount != 2 {
		t.Fatalf("orderbook amounts not updated correctly")
	}

	updates = &Update{
		Bids:     Items{{Price: 1337, Amount: 0, ID: 1}},
		Asks:     Items{{Price: 1337, Amount: 0, ID: 2}},
		UpdateID: 2,
	}
	d.UpdateBidAskByPrice(updates)

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

	updates := &Update{
		Bids: Items{{Price: 1337, Amount: 2, ID: 1}},
		Asks: Items{{Price: 1337, Amount: 2, ID: 2}},
	}
	err := d.DeleteBidAskByID(updates, false)
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

	updates = &Update{
		Bids: Items{{Price: 1337, Amount: 2, ID: 1}},
	}
	err = d.DeleteBidAskByID(updates, false)
	if !strings.Contains(err.Error(), errIDCannotBeMatched.Error()) {
		t.Fatalf("error expected %v received %v", errIDCannotBeMatched, err)
	}

	updates = &Update{
		Asks: Items{{Price: 1337, Amount: 2, ID: 2}},
	}
	err = d.DeleteBidAskByID(updates, false)
	if !strings.Contains(err.Error(), errIDCannotBeMatched.Error()) {
		t.Fatalf("error expected %v received %v", errIDCannotBeMatched, err)
	}

	updates = &Update{
		Asks: Items{{Price: 1337, Amount: 2, ID: 2}},
	}
	err = d.DeleteBidAskByID(updates, true)
	if !errors.Is(err, nil) {
		t.Fatalf("error expected %v received %v", nil, err)
	}
}

func TestUpdateBidAskByID(t *testing.T) {
	t.Parallel()
	d := NewDepth(id)
	d.LoadSnapshot(Items{{Price: 1337, Amount: 1, ID: 1}}, Items{{Price: 1337, Amount: 10, ID: 2}}, 0, time.Time{}, false)

	updates := &Update{
		Bids: Items{{Price: 1337, Amount: 2, ID: 1}},
		Asks: Items{{Price: 1337, Amount: 2, ID: 2}},
	}
	err := d.UpdateBidAskByID(updates)
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

	updates = &Update{
		Bids: Items{{Price: 1337, Amount: 2, ID: 666}},
	}
	// random unmatching IDs
	err = d.UpdateBidAskByID(updates)
	if !strings.Contains(err.Error(), errIDCannotBeMatched.Error()) {
		t.Fatalf("error expected %v received %v", errIDCannotBeMatched, err)
	}

	updates = &Update{
		Asks: Items{{Price: 1337, Amount: 2, ID: 69}},
	}
	err = d.UpdateBidAskByID(updates)
	if !strings.Contains(err.Error(), errIDCannotBeMatched.Error()) {
		t.Fatalf("error expected %v received %v", errIDCannotBeMatched, err)
	}
}

func TestInsertBidAskByID(t *testing.T) {
	t.Parallel()
	d := NewDepth(id)
	d.LoadSnapshot(Items{{Price: 1337, Amount: 1, ID: 1}}, Items{{Price: 1337, Amount: 10, ID: 2}}, 0, time.Time{}, false)

	updates := &Update{
		Asks: Items{{Price: 1337, Amount: 2, ID: 3}},
	}

	err := d.InsertBidAskByID(updates)
	if !strings.Contains(err.Error(), errCollisionDetected.Error()) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errCollisionDetected)
	}

	d.LoadSnapshot(Items{{Price: 1337, Amount: 1, ID: 1}}, Items{{Price: 1337, Amount: 10, ID: 2}}, 0, time.Time{}, false)

	updates = &Update{
		Bids: Items{{Price: 1337, Amount: 2, ID: 3}},
	}

	err = d.InsertBidAskByID(updates)
	if !strings.Contains(err.Error(), errCollisionDetected.Error()) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errCollisionDetected)
	}

	d.LoadSnapshot(Items{{Price: 1337, Amount: 1, ID: 1}}, Items{{Price: 1337, Amount: 10, ID: 2}}, 0, time.Time{}, false)
	updates = &Update{
		Bids: Items{{Price: 1338, Amount: 2, ID: 3}},
		Asks: Items{{Price: 1336, Amount: 2, ID: 4}},
	}
	err = d.InsertBidAskByID(updates)
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

	updates := &Update{
		Bids: Items{{Price: 1338, Amount: 0, ID: 3}},
		Asks: Items{{Price: 1336, Amount: 2, ID: 4}},
	}
	err := d.UpdateInsertByID(updates)
	if !strings.Contains(err.Error(), errAmountCannotBeLessOrEqualToZero.Error()) {
		t.Fatalf("expected: %v but received: %v", errAmountCannotBeLessOrEqualToZero, err)
	}

	// Above will invalidate the book
	_, err = d.Retrieve()
	if !errors.Is(err, ErrOrderbookInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderbookInvalid)
	}

	d.LoadSnapshot(Items{{Price: 1337, Amount: 1, ID: 1}}, Items{{Price: 1337, Amount: 10, ID: 2}}, 0, time.Time{}, false)

	updates = &Update{
		Bids: Items{{Price: 1338, Amount: 2, ID: 3}},
		Asks: Items{{Price: 1336, Amount: 0, ID: 4}},
	}
	err = d.UpdateInsertByID(updates)
	if !strings.Contains(err.Error(), errAmountCannotBeLessOrEqualToZero.Error()) {
		t.Fatalf("expected: %v but received: %v", errAmountCannotBeLessOrEqualToZero, err)
	}

	// Above will invalidate the book
	_, err = d.Retrieve()
	if !errors.Is(err, ErrOrderbookInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderbookInvalid)
	}

	d.LoadSnapshot(Items{{Price: 1337, Amount: 1, ID: 1}}, Items{{Price: 1337, Amount: 10, ID: 2}}, 0, time.Time{}, false)

	updates = &Update{
		Bids: Items{{Price: 1338, Amount: 2, ID: 3}},
		Asks: Items{{Price: 1336, Amount: 2, ID: 4}},
	}
	err = d.UpdateInsertByID(updates)
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
	err := d.Invalidate(nil)
	if !errors.Is(err, ErrOrderbookInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderbookInvalid)
	}
	_, err = d.IsRESTSnapshot()
	if !errors.Is(err, ErrOrderbookInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderbookInvalid)
	}

	d.validationError = nil
	b, err := d.IsRESTSnapshot()
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
	err := d.Invalidate(nil)
	if !errors.Is(err, ErrOrderbookInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderbookInvalid)
	}
	_, err = d.LastUpdateID()
	if !errors.Is(err, ErrOrderbookInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderbookInvalid)
	}

	d.validationError = nil
	d.lastUpdateID = 1337
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
	if err := d.Invalidate(nil); !errors.Is(err, ErrOrderbookInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderbookInvalid)
	}
	d.Publish()
	d.validationError = nil
	d.Publish()
}

func TestIsValid(t *testing.T) {
	t.Parallel()
	d := Depth{}
	if !d.IsValid() {
		t.Fatalf("received: '%v' but expected: '%v'", d.IsValid(), true)
	}
	if err := d.Invalidate(nil); !errors.Is(err, ErrOrderbookInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderbookInvalid)
	}
	if d.IsValid() {
		t.Fatalf("received: '%v' but expected: '%v'", d.IsValid(), false)
	}
}

func TestGetBaseFromNominalSlippage(t *testing.T) {
	t.Parallel()
	depth := NewDepth(id)
	depth.LoadSnapshot(bid, ask, 0, time.Time{}, true)
	_, err := depth.GetBaseFromNominalSlippage(10, 1355.5)
	if !errors.Is(err, errNotEnoughLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNotEnoughLiquidity)
	}

	// First and second price
	amt, err := depth.GetBaseFromNominalSlippage(0.018443378827001, 1336)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if amt != 2.0005644917294916 {
		t.Fatalf("received: '%v' but expected: '%v'", amt, 2.0005644917294916)
	}

	// All the way up to the last price
	amt, err = depth.GetBaseFromNominalSlippage(0.71107784431138, 1336)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	// This exceeds the entire total base available - should be 20.
	if amt != 20.00721336370539 {
		t.Fatalf("received: '%v' but expected: '%v'", amt, 20.00721336370539)
	}
}

func TestGetBaseFromNominalSlippageFromMid(t *testing.T) {
	t.Parallel()
	depth := NewDepth(id)

	_, err := depth.GetBaseFromNominalSlippageFromMid(10)
	if !errors.Is(err, errNoLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoLiquidity)
	}

	depth.LoadSnapshot(bid, ask, 0, time.Time{}, true)
	_, err = depth.GetBaseFromNominalSlippageFromMid(10)
	if !errors.Is(err, errNotEnoughLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNotEnoughLiquidity)
	}

	// First price from mid point
	amt, err := depth.GetBaseFromNominalSlippageFromMid(0.03741114852226)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if amt != 1 {
		t.Fatalf("received: '%v' but expected: '%v'", amt, 1)
	}

	// All the way up to the last price from mid price
	amt, err = depth.GetBaseFromNominalSlippageFromMid(0.74822297044519)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	// This exceeds the entire total base available - should be 20.
	if amt != 20.00721336370539 {
		t.Fatalf("received: '%v' but expected: '%v'", amt, 20.00721336370539)
	}
}

func TestGetBaseFromNominalSlippageFromBest(t *testing.T) {
	t.Parallel()
	depth := NewDepth(id)

	_, err := depth.GetBaseFromNominalSlippageFromBest(10)
	if !errors.Is(err, errNoLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoLiquidity)
	}

	depth.LoadSnapshot(bid, ask, 0, time.Time{}, true)
	_, err = depth.GetBaseFromNominalSlippageFromBest(10)
	if !errors.Is(err, errNotEnoughLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNotEnoughLiquidity)
	}

	// First and second price from best bid
	amt, err := depth.GetBaseFromNominalSlippageFromBest(0.037425149700599)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if amt != 2.000374531835206 {
		t.Fatalf("received: '%v' but expected: '%v'", amt, 2.000374531835206)
	}

	// All the way up to the last price from best bid price
	amt, err = depth.GetBaseFromNominalSlippageFromBest(0.71107784431138)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	// This exceeds the entire total base available - should be 20.
	if amt != 20.00721336370539 {
		t.Fatalf("received: '%v' but expected: '%v'", amt, 20.00721336370539)
	}
}

func TestGetQuoteFromNominalSlippage(t *testing.T) {
	t.Parallel()
	depth := NewDepth(id)
	depth.LoadSnapshot(bid, ask, 0, time.Time{}, true)
	_, err := depth.GetQuoteFromNominalSlippage(10, 1355.5)
	if !errors.Is(err, errNotEnoughLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNotEnoughLiquidity)
	}

	// First and second price
	amt, err := depth.GetQuoteFromNominalSlippage(0.037397157816006, 1337)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if amt != 2674.5 {
		t.Fatalf("received: '%v' but expected: '%v'", amt, 2674.5)
	}

	// All the way up to the last price
	amt, err = depth.GetQuoteFromNominalSlippage(0.71054599850411, 1337)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	// This does not match the entire total quote available - should be 26930.
	if amt != 26920.5 {
		t.Fatalf("received: '%v' but expected: '%v'", amt, 26920.5)
	}
}

func TestGetQuoteFromNominalSlippageFromMid(t *testing.T) {
	t.Parallel()
	depth := NewDepth(id)

	_, err := depth.GetQuoteFromNominalSlippageFromMid(10)
	if !errors.Is(err, errNoLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoLiquidity)
	}

	depth.LoadSnapshot(bid, ask, 0, time.Time{}, true)
	_, err = depth.GetQuoteFromNominalSlippageFromMid(10)
	if !errors.Is(err, errNotEnoughLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNotEnoughLiquidity)
	}

	// First price from mid point
	amt, err := depth.GetQuoteFromNominalSlippageFromMid(0.074822297044519)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	fmt.Println(depth.asks.getSideAmounts())

	if amt != 2674.5 {
		t.Fatalf("received: '%v' but expected: '%v'", amt, 2674.5)
	}

	// All the way up to the last price from mid price
	amt, err = depth.GetQuoteFromNominalSlippageFromMid(0.74822297044519)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	// This does not match the entire total quote available - should be 26930.
	if amt != 26920.5 {
		t.Fatalf("received: '%v' but expected: '%v'", amt, 26920.5)
	}
}

func TestGetQuoteFromNominalSlippageFromBest(t *testing.T) {
	t.Parallel()
	depth := NewDepth(id)

	_, err := depth.GetQuoteFromNominalSlippageFromBest(10)
	if !errors.Is(err, errNoLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoLiquidity)
	}

	depth.LoadSnapshot(bid, ask, 0, time.Time{}, true)
	_, err = depth.GetQuoteFromNominalSlippageFromBest(10)
	if !errors.Is(err, errNotEnoughLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNotEnoughLiquidity)
	}

	// First and second price from best bid
	amt, err := depth.GetQuoteFromNominalSlippageFromBest(0.037397157816006)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if amt != 2674.5 {
		t.Fatalf("received: '%v' but expected: '%v'", amt, 2674.5)
	}

	// All the way up to the last price from best bid price
	amt, err = depth.GetQuoteFromNominalSlippageFromBest(0.71054599850411)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	// This does not match the entire total quote available - should be 26930.
	if amt != 26920.5 {
		t.Fatalf("received: '%v' but expected: '%v'", amt, 26920.5)
	}
}

func TestGetBaseFromImpactSlippage(t *testing.T) {
	t.Parallel()
	depth := NewDepth(id)
	depth.LoadSnapshot(bid, ask, 0, time.Time{}, true)
	_, err := depth.GetBaseFromImpactSlippage(10, 1336)
	if !errors.Is(err, errNotEnoughLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNotEnoughLiquidity)
	}

	// First and second price from best bid - price level target 1326 (which should be kept)
	amt, err := depth.GetBaseFromImpactSlippage(0.7485029940119761, 1336)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if amt != 10 {
		t.Fatalf("received: '%v' but expected: '%v'", amt, 10)
	}

	// All the way up to the last price from best bid price
	amt, err = depth.GetBaseFromImpactSlippage(1.4221556886227544, 1336)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	// This does not match the entire total quote available - should be 26930.
	if amt != 19 {
		t.Fatalf("received: '%v' but expected: '%v'", amt, 19)
	}
}

func TestGetBaseFromImpactSlippageFromMid(t *testing.T) {
	t.Parallel()
	depth := NewDepth(id)
	depth.LoadSnapshot(bid, ask, 0, time.Time{}, true)
	_, err := depth.GetBaseFromImpactSlippageFromMid(10)
	if !errors.Is(err, errNotEnoughLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNotEnoughLiquidity)
	}

	// First and second price from mid - price level target 1326 (which should be kept)
	amt, err := depth.GetBaseFromImpactSlippageFromMid(0.7485029940119761)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if amt != 10 {
		t.Fatalf("received: '%v' but expected: '%v'", amt, 10)
	}

	// All the way up to the last price from best bid price
	amt, err = depth.GetBaseFromImpactSlippageFromMid(1.4221556886227544)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	// This does not match the entire total quote available - should be 26930.
	if amt != 19 {
		t.Fatalf("received: '%v' but expected: '%v'", amt, 19)
	}
}

func TestGetBaseFromImpactSlippageFromBest(t *testing.T) {
	t.Parallel()
	depth := NewDepth(id)
	depth.LoadSnapshot(bid, ask, 0, time.Time{}, true)
	_, err := depth.GetBaseFromImpactSlippageFromBest(10)
	if !errors.Is(err, errNotEnoughLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNotEnoughLiquidity)
	}

	// First and second price from mid - price level target 1326 (which should be kept)
	amt, err := depth.GetBaseFromImpactSlippageFromBest(0.7485029940119761)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if amt != 10 {
		t.Fatalf("received: '%v' but expected: '%v'", amt, 10)
	}

	// All the way up to the last price from best bid price
	amt, err = depth.GetBaseFromImpactSlippageFromBest(1.4221556886227544)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	// This does not match the entire total quote available - should be 26930.
	if amt != 19 {
		t.Fatalf("received: '%v' but expected: '%v'", amt, 19)
	}
}

func TestGetQuoteFromImpactSlippage(t *testing.T) {
	t.Parallel()
	depth := NewDepth(id)
	depth.LoadSnapshot(bid, ask, 0, time.Time{}, true)
	_, err := depth.GetQuoteFromImpactSlippage(10, 1337)
	if !errors.Is(err, errNotEnoughLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNotEnoughLiquidity)
	}

	// First and second price from best bid - price level target 1326 (which should be kept)
	amt, err := depth.GetQuoteFromImpactSlippage(0.7479431563201197, 1337)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if amt != 13415 {
		t.Fatalf("received: '%v' but expected: '%v'", amt, 13415)
	}

	// All the way up to the last price from best bid price
	amt, err = depth.GetQuoteFromImpactSlippage(1.4210919970082274, 1337)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	// This does not match the entire total quote available - should be 26930.
	if amt != 25574 {
		t.Fatalf("received: '%v' but expected: '%v'", amt, 25574)
	}
}

func TestGetQuoteFromImpactSlippageFromMid(t *testing.T) {
	t.Parallel()
	depth := NewDepth(id)
	depth.LoadSnapshot(bid, ask, 0, time.Time{}, true)
	_, err := depth.GetQuoteFromImpactSlippageFromMid(10)
	if !errors.Is(err, errNotEnoughLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNotEnoughLiquidity)
	}

	// First and second price from mid - price level target 1326 (which should be kept)
	amt, err := depth.GetQuoteFromImpactSlippageFromMid(0.7485029940119761)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if amt != 13415 {
		t.Fatalf("received: '%v' but expected: '%v'", amt, 13415)
	}

	// All the way up to the last price from best bid price
	amt, err = depth.GetQuoteFromImpactSlippageFromMid(1.4221556886227544)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	// This does not match the entire total quote available - should be 26930.
	if amt != 25574 {
		t.Fatalf("received: '%v' but expected: '%v'", amt, 25574)
	}
}

func TestGetQuoteFromImpactSlippageFromBest(t *testing.T) {
	t.Parallel()
	depth := NewDepth(id)
	depth.LoadSnapshot(bid, ask, 0, time.Time{}, true)
	_, err := depth.GetQuoteFromImpactSlippageFromBest(10)
	if !errors.Is(err, errNotEnoughLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNotEnoughLiquidity)
	}

	// First and second price from mid - price level target 1326 (which should be kept)
	amt, err := depth.GetQuoteFromImpactSlippageFromBest(0.7485029940119761)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if amt != 13415 {
		t.Fatalf("received: '%v' but expected: '%v'", amt, 13415)
	}

	// All the way up to the last price from best bid price
	amt, err = depth.GetQuoteFromImpactSlippageFromBest(1.4221556886227544)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	// This does not match the entire total quote available - should be 26930.
	if amt != 25574 {
		t.Fatalf("received: '%v' but expected: '%v'", amt, 25574)
	}
}

func TestGetMovementByBase(t *testing.T) {
	t.Parallel()
	depth := NewDepth(id)
	depth.LoadSnapshot(bid, ask, 0, time.Time{}, true)
	_, err := depth.GetMovementByBase(20.1, 1336)
	if !errors.Is(err, errAmountExceedsSideLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errAmountExceedsSideLiquidity)
	}

	mov, err := depth.GetMovementByBase(1, 1336)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if mov.NominalPercentage != 0 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.NominalPercentage, 0)
	}
	if mov.ImpactPercentage != 0.07485029940119761 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.ImpactPercentage, 0.07485029940119761)
	}
	if mov.SlippageCost != 0 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.SlippageCost, 0)
	}

	mov, err = depth.GetMovementByBase(19.5, 1336)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	fmt.Println(depth.bids.getSideAmounts())

	if mov.NominalPercentage != 0.692845079072617 { // Should be 0.6933008982035955
		t.Fatalf("received: '%v' but expected: '%v'", mov.NominalPercentage, 0.692845079072617)
	}
	if mov.ImpactPercentage != 1.4221556886227544 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.ImpactPercentage, 1.4221556886227544)
	}
	if mov.SlippageCost != 180.5 { // Expecting 185.25
		t.Fatalf("received: '%v' but expected: '%v'", mov.SlippageCost, 180.5)
	}

	// // All the way up to the last price from best bid price
	// mov, err = depth.GetMovementByBase(1.4221556886227544, 1336)
	// if !errors.Is(err, nil) {
	// 	t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	// }

	// // This does not match the entire total quote available - should be 26930.
	// if mov != 19 {
	// 	t.Fatalf("received: '%v' but expected: '%v'", mov, 19)
	// }
}

func TestXxx(t *testing.T) {
	for x := 0; x < 20; x++ {
		fmt.Println(1336 - x)
	}

}
