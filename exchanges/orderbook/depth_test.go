package orderbook

import (
	"errors"
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
		maxDepth:         10,
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

	if ob.MaxDepth != 10 {
		t.Errorf("expected max depth %v, but received %v", 10, ob.MaxDepth)
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

func TestHitTheBidsByNominalSlippage(t *testing.T) {
	t.Parallel()
	_, err := getInvalidDepth().HitTheBidsByNominalSlippage(10, 1355.5)
	if !errors.Is(err, ErrOrderbookInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderbookInvalid)
	}

	depth := NewDepth(id)
	_, err = depth.HitTheBidsByNominalSlippage(10, 1355.5)
	if !errors.Is(err, errNoLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoLiquidity)
	}

	depth.LoadSnapshot(bid, ask, 0, time.Time{}, true)

	// First tranche
	amt, err := depth.HitTheBidsByNominalSlippage(0, 1336)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if amt.Sold != 1 {
		t.Fatalf("received: '%+v' but expected: '%v'", amt, 2)
	}

	if amt.NominalPercentage != 0 {
		t.Fatalf("received: '%+v' but expected: '%v'", amt, 0)
	}

	if amt.StartPrice != 1336 {
		t.Fatalf("received: '%+v' but expected: '%v'", amt, 1336)
	}

	if amt.EndPrice != 1336 {
		t.Fatalf("received: '%+v' but expected: '%v'", amt, 1336)
	}

	if amt.FullBookSideConsumed {
		t.Fatalf("received: '%+v' but expected: '%v'", amt.FullBookSideConsumed, false)
	}

	// First and second price
	amt, err = depth.HitTheBidsByNominalSlippage(0.037425149700598806, 1336)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if amt.Sold != 2 {
		t.Fatalf("received: '%+v' but expected: '%v'", amt, 2)
	}

	if amt.NominalPercentage != 0.037425149700598806 {
		t.Fatalf("received: '%+v' but expected: '%v'", amt, 0.037425149700598806)
	}

	if amt.StartPrice != 1336 {
		t.Fatalf("received: '%+v' but expected: '%v'", amt, 1336)
	}

	if amt.EndPrice != 1335 {
		t.Fatalf("received: '%+v' but expected: '%v'", amt, 1335)
	}

	if amt.FullBookSideConsumed {
		t.Fatalf("received: '%+v' but expected: '%v'", amt.FullBookSideConsumed, false)
	}

	// First and half of second tranche
	amt, err = depth.HitTheBidsByNominalSlippage(0.02495009980039353, 1336)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if amt.Sold != 1.4999999999998295 {
		t.Fatalf("received: '%+v' but expected: '%v'", amt, 1.4999999999998295)
	}

	if amt.NominalPercentage != 0.02495009980039353 {
		t.Fatalf("received: '%+v' but expected: '%v'", amt, 0.02495009980039353)
	}

	if amt.StartPrice != 1336 {
		t.Fatalf("received: '%+v' but expected: '%v'", amt, 1336)
	}

	if amt.EndPrice != 1335 {
		t.Fatalf("received: '%+v' but expected: '%v'", amt, 1335)
	}

	if amt.FullBookSideConsumed {
		t.Fatalf("received: '%+v' but expected: '%v'", amt.FullBookSideConsumed, false)
	}

	// All the way up to the last price
	amt, err = depth.HitTheBidsByNominalSlippage(0.7110778443113772, 1336)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	// This exceeds the entire total base available - should be 20.
	if amt.Sold != 20 {
		t.Fatalf("received: '%+v' but expected: '%v'", amt, 20.00721336370539)
	}

	if amt.NominalPercentage != 0.7110778443113772 {
		t.Fatalf("received: '%+v' but expected: '%v'", amt, 0.7110778443113772)
	}

	if amt.StartPrice != 1336 {
		t.Fatalf("received: '%+v' but expected: '%v'", amt, 1336)
	}

	if amt.EndPrice != 1317 {
		t.Fatalf("received: '%+v' but expected: '%v'", amt, 1317)
	}

	if !amt.FullBookSideConsumed {
		t.Fatalf("received: '%+v' but expected: '%v'", amt.FullBookSideConsumed, true)
	}
}

func TestHitTheBidsByNominalSlippageFromMid(t *testing.T) {
	t.Parallel()
	_, err := getInvalidDepth().HitTheBidsByNominalSlippageFromMid(10)
	if !errors.Is(err, ErrOrderbookInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderbookInvalid)
	}

	depth := NewDepth(id)

	_, err = depth.HitTheBidsByNominalSlippageFromMid(10)
	if !errors.Is(err, errNoLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoLiquidity)
	}

	depth.LoadSnapshot(bid, ask, 0, time.Time{}, true)
	// First price from mid point
	amt, err := depth.HitTheBidsByNominalSlippageFromMid(0.03741114852226)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if amt.Sold != 1 {
		t.Fatalf("received: '%v' but expected: '%v'", amt, 1)
	}

	// All the way up to the last price from mid price
	amt, err = depth.HitTheBidsByNominalSlippageFromMid(0.74822297044519)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	// This exceeds the entire total base available
	if amt.Sold != 20 {
		t.Fatalf("received: '%v' but expected: '%v'", amt, 20)
	}
}

func TestHitTheBidsByNominalSlippageFromBest(t *testing.T) {
	t.Parallel()
	_, err := getInvalidDepth().HitTheBidsByNominalSlippageFromBest(10)
	if !errors.Is(err, ErrOrderbookInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderbookInvalid)
	}

	depth := NewDepth(id)

	_, err = depth.HitTheBidsByNominalSlippageFromBest(10)
	if !errors.Is(err, errNoLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoLiquidity)
	}

	depth.LoadSnapshot(bid, ask, 0, time.Time{}, true)
	// First and second price from best bid
	amt, err := depth.HitTheBidsByNominalSlippageFromBest(0.037425149700599)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if amt.Sold != 2 {
		t.Fatalf("received: '%+v' but expected: '%+v'", amt, 2)
	}

	// All the way up to the last price from best bid price
	amt, err = depth.HitTheBidsByNominalSlippageFromBest(0.71107784431138)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	// This exceeds the entire total base available
	if amt.Sold != 20 {
		t.Fatalf("received: '%v' but expected: '%v'", amt, 20)
	}
}

func TestLiftTheAsksByNominalSlippage(t *testing.T) {
	t.Parallel()
	_, err := getInvalidDepth().LiftTheAsksByNominalSlippage(10, 1355.5)
	if !errors.Is(err, ErrOrderbookInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderbookInvalid)
	}

	depth := NewDepth(id)

	_, err = depth.LiftTheAsksByNominalSlippage(10, 1355.5)
	if !errors.Is(err, errNoLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoLiquidity)
	}

	depth.LoadSnapshot(bid, ask, 0, time.Time{}, true)

	// First and second price
	amt, err := depth.LiftTheAsksByNominalSlippage(0.037397157816006, 1337)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if amt.Sold != 2675 {
		t.Fatalf("received: '%v' but expected: '%v'", amt, 2675)
	}

	// All the way up to the last price
	amt, err = depth.LiftTheAsksByNominalSlippage(0.71054599850411, 1337)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if amt.Sold != 26930 {
		t.Fatalf("received: '%v' but expected: '%v'", amt, 26930)
	}
}

func TestLiftTheAsksByNominalSlippageFromMid(t *testing.T) {
	t.Parallel()
	_, err := getInvalidDepth().LiftTheAsksByNominalSlippageFromMid(10)
	if !errors.Is(err, ErrOrderbookInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderbookInvalid)
	}

	depth := NewDepth(id)

	_, err = depth.LiftTheAsksByNominalSlippageFromMid(10)
	if !errors.Is(err, errNoLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoLiquidity)
	}

	depth.LoadSnapshot(bid, ask, 0, time.Time{}, true)
	// First price from mid point
	amt, err := depth.LiftTheAsksByNominalSlippageFromMid(0.074822297044519)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if amt.Sold != 2675 {
		t.Fatalf("received: '%v' but expected: '%v'", amt, 2675)
	}

	// All the way up to the last price from mid price
	amt, err = depth.LiftTheAsksByNominalSlippageFromMid(0.74822297044519)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	// This does not match the entire total quote available
	if amt.Sold != 26930 {
		t.Fatalf("received: '%v' but expected: '%v'", amt, 26930)
	}
}

func TestLiftTheAsksByNominalSlippageFromBest(t *testing.T) {
	t.Parallel()
	_, err := getInvalidDepth().LiftTheAsksByNominalSlippageFromBest(10)
	if !errors.Is(err, ErrOrderbookInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderbookInvalid)
	}

	depth := NewDepth(id)

	_, err = depth.LiftTheAsksByNominalSlippageFromBest(10)
	if !errors.Is(err, errNoLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoLiquidity)
	}

	depth.LoadSnapshot(bid, ask, 0, time.Time{}, true)
	// First and second price from best bid
	amt, err := depth.LiftTheAsksByNominalSlippageFromBest(0.037397157816006)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if amt.Sold != 2675 {
		t.Fatalf("received: '%v' but expected: '%v'", amt, 2675)
	}

	// All the way up to the last price from best bid price
	amt, err = depth.LiftTheAsksByNominalSlippageFromBest(0.71054599850411)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	// This does not match the entire total quote available
	if amt.Sold != 26930 {
		t.Fatalf("received: '%v' but expected: '%v'", amt, 26930)
	}
}

func TestHitTheBidsByImpactSlippage(t *testing.T) {
	t.Parallel()
	_, err := getInvalidDepth().HitTheBidsByImpactSlippage(0.7485029940119761, 1336)
	if !errors.Is(err, ErrOrderbookInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderbookInvalid)
	}

	depth := NewDepth(id)
	depth.LoadSnapshot(bid, ask, 0, time.Time{}, true)

	// First and second price from best bid - price level target 1326 (which should be kept)
	amt, err := depth.HitTheBidsByImpactSlippage(0.7485029940119761, 1336)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if amt.Sold != 10 {
		t.Fatalf("received: '%v' but expected: '%v'", amt, 10)
	}

	// All the way up to the last price from best bid price
	amt, err = depth.HitTheBidsByImpactSlippage(1.4221556886227544, 1336)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	// This does not match the entire total quote available - should be 26930.
	if amt.Sold != 19 {
		t.Fatalf("received: '%v' but expected: '%v'", amt, 19)
	}
}

func TestHitTheBidsByImpactSlippageFromMid(t *testing.T) {
	t.Parallel()
	_, err := getInvalidDepth().HitTheBidsByImpactSlippageFromMid(10)
	if !errors.Is(err, ErrOrderbookInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderbookInvalid)
	}

	depth := NewDepth(id)
	_, err = depth.HitTheBidsByImpactSlippageFromMid(10)
	if !errors.Is(err, errNoLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoLiquidity)
	}

	depth.LoadSnapshot(bid, ask, 0, time.Time{}, true)

	// First and second price from mid - price level target 1326 (which should be kept)
	amt, err := depth.HitTheBidsByImpactSlippageFromMid(0.7485029940119761)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if amt.Sold != 10 {
		t.Fatalf("received: '%v' but expected: '%v'", amt, 10)
	}

	// All the way up to the last price from best bid price
	amt, err = depth.HitTheBidsByImpactSlippageFromMid(1.4221556886227544)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if amt.Sold != 19 {
		t.Fatalf("received: '%v' but expected: '%v'", amt, 19)
	}
}

func TestHitTheBidsByImpactSlippageFromBest(t *testing.T) {
	t.Parallel()
	_, err := getInvalidDepth().HitTheBidsByImpactSlippageFromBest(10)
	if !errors.Is(err, ErrOrderbookInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderbookInvalid)
	}

	depth := NewDepth(id)
	_, err = depth.HitTheBidsByImpactSlippageFromBest(10)
	if !errors.Is(err, errNoLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoLiquidity)
	}
	depth.LoadSnapshot(bid, ask, 0, time.Time{}, true)

	// First and second price from mid - price level target 1326 (which should be kept)
	amt, err := depth.HitTheBidsByImpactSlippageFromBest(0.7485029940119761)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if amt.Sold != 10 {
		t.Fatalf("received: '%v' but expected: '%v'", amt, 10)
	}

	// All the way up to the last price from best bid price
	amt, err = depth.HitTheBidsByImpactSlippageFromBest(1.4221556886227544)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if amt.Sold != 19 {
		t.Fatalf("received: '%v' but expected: '%v'", amt, 19)
	}
}

func TestLiftTheAsksByImpactSlippage(t *testing.T) {
	t.Parallel()
	_, err := getInvalidDepth().LiftTheAsksByImpactSlippage(0.7479431563201197, 1337)
	if !errors.Is(err, ErrOrderbookInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderbookInvalid)
	}

	depth := NewDepth(id)
	depth.LoadSnapshot(bid, ask, 0, time.Time{}, true)

	// First and second price from best bid - price level target 1326 (which should be kept)
	amt, err := depth.LiftTheAsksByImpactSlippage(0.7479431563201197, 1337)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if amt.Sold != 13415 {
		t.Fatalf("received: '%v' but expected: '%v'", amt, 13415)
	}

	// All the way up to the last price from best bid price
	amt, err = depth.LiftTheAsksByImpactSlippage(1.4210919970082274, 1337)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if amt.Sold != 25574 {
		t.Fatalf("received: '%v' but expected: '%v'", amt, 25574)
	}
}

func TestLiftTheAsksByImpactSlippageFromMid(t *testing.T) {
	t.Parallel()
	_, err := getInvalidDepth().LiftTheAsksByImpactSlippageFromMid(10)
	if !errors.Is(err, ErrOrderbookInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderbookInvalid)
	}

	depth := NewDepth(id)
	_, err = depth.LiftTheAsksByImpactSlippageFromMid(10)
	if !errors.Is(err, errNoLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoLiquidity)
	}
	depth.LoadSnapshot(bid, ask, 0, time.Time{}, true)

	// First and second price from mid - price level target 1326 (which should be kept)
	amt, err := depth.LiftTheAsksByImpactSlippageFromMid(0.7485029940119761)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if amt.Sold != 13415 {
		t.Fatalf("received: '%+v' but expected: '%v'", amt, 13415)
	}

	// All the way up to the last price from best bid price
	amt, err = depth.LiftTheAsksByImpactSlippageFromMid(1.4221556886227544)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if amt.Sold != 25574 {
		t.Fatalf("received: '%v' but expected: '%v'", amt, 25574)
	}
}

func TestLiftTheAsksByImpactSlippageFromBest(t *testing.T) {
	t.Parallel()
	_, err := getInvalidDepth().LiftTheAsksByImpactSlippageFromBest(10)
	if !errors.Is(err, ErrOrderbookInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderbookInvalid)
	}

	depth := NewDepth(id)
	_, err = depth.LiftTheAsksByImpactSlippageFromBest(10)
	if !errors.Is(err, errNoLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoLiquidity)
	}
	depth.LoadSnapshot(bid, ask, 0, time.Time{}, true)

	// First and second price from mid - price level target 1326 (which should be kept)
	amt, err := depth.LiftTheAsksByImpactSlippageFromBest(0.7479431563201197)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if amt.Sold != 13415 {
		t.Fatalf("received: '%v' but expected: '%v'", amt, 13415)
	}

	// All the way up to the last price from best bid price
	amt, err = depth.LiftTheAsksByImpactSlippageFromBest(1.4210919970082274)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	// This goes to price 1356, it will not count that tranches' volume as it
	// is needed to sustain the slippage.
	if amt.Sold != 25574 {
		t.Fatalf("received: '%v' but expected: '%v'", amt, 25574)
	}
}

func TestHitTheBids(t *testing.T) {
	t.Parallel()
	depth := NewDepth(id)
	depth.LoadSnapshot(bid, ask, 0, time.Time{}, true)
	mov, err := depth.HitTheBids(20.1, 1336, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if !mov.FullBookSideConsumed {
		t.Fatal("entire side should be consumed by this value")
	}

	mov, err = depth.HitTheBids(1, 1336, false)
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

	mov, err = depth.HitTheBids(19.5, 1336, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if mov.NominalPercentage != 0.692845079072617 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.NominalPercentage, 0.692845079072617)
	}
	if mov.ImpactPercentage != 1.4221556886227544 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.ImpactPercentage, 1.4221556886227544)
	}

	if mov.SlippageCost != 180.5 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.SlippageCost, 180.5)
	}

	// All the way up to the last price from best bid price
	mov, err = depth.HitTheBids(20, 1336, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if mov.NominalPercentage != 0.7110778443113772 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.NominalPercentage, 0.7110778443113772)
	}
	if mov.ImpactPercentage != FullLiquidityExhaustedPercentage {
		t.Fatalf("received: '%v' but expected: '%v'", mov.ImpactPercentage, FullLiquidityExhaustedPercentage)
	}
	if mov.SlippageCost != 190 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.SlippageCost, 190)
	}
}

func TestHitTheBids_QuotationRequired(t *testing.T) {
	t.Parallel()
	_, err := getInvalidDepth().HitTheBids(26531, 1336, true)
	if !errors.Is(err, ErrOrderbookInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderbookInvalid)
	}

	depth := NewDepth(id)
	depth.LoadSnapshot(bid, ask, 0, time.Time{}, true)
	mov, err := depth.HitTheBids(26531, 1336, true)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if !mov.FullBookSideConsumed {
		t.Fatal("entire side should be consumed by this value")
	}

	mov, err = depth.HitTheBids(1336, 1336, true)
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

	mov, err = depth.HitTheBids(25871.5, 1336, true)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if mov.NominalPercentage != 0.692845079072617 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.NominalPercentage, 0.692845079072617)
	}
	if mov.ImpactPercentage != 1.4221556886227544 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.ImpactPercentage, 1.4221556886227544)
	}

	if mov.SlippageCost != 180.5 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.SlippageCost, 180.5)
	}

	// All the way up to the last price from best bid price
	mov, err = depth.HitTheBids(26530, 1336, true)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if mov.NominalPercentage != 0.7110778443113772 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.NominalPercentage, 0.7110778443113772)
	}
	if mov.ImpactPercentage != FullLiquidityExhaustedPercentage {
		t.Fatalf("received: '%v' but expected: '%v'", mov.ImpactPercentage, FullLiquidityExhaustedPercentage)
	}
	if mov.SlippageCost != 190 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.SlippageCost, 190)
	}
}

func TestHitTheBidsFromMid(t *testing.T) {
	t.Parallel()
	_, err := getInvalidDepth().HitTheBidsFromMid(10, false)
	if !errors.Is(err, ErrOrderbookInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderbookInvalid)
	}

	depth := NewDepth(id)
	_, err = depth.HitTheBidsFromMid(10, false)
	if !errors.Is(err, errNoLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoLiquidity)
	}
	depth.LoadSnapshot(bid, ask, 0, time.Time{}, true)
	mov, err := depth.HitTheBidsFromMid(20.1, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if !mov.FullBookSideConsumed {
		t.Fatal("entire side should be consumed by this value")
	}

	mov, err = depth.HitTheBidsFromMid(1, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if mov.NominalPercentage != 0.03741114852225963 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.NominalPercentage, 0.03741114852225963)
	}

	if mov.ImpactPercentage != 0.11223344556677892 { // mid price 1336.5 -> 1335
		t.Fatalf("received: '%v' but expected: '%v'", mov.ImpactPercentage, 0.11223344556677892)
	}
	if mov.SlippageCost != 0 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.SlippageCost, 0)
	}

	mov, err = depth.HitTheBidsFromMid(19.5, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if mov.NominalPercentage != 0.7299970262933156 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.NominalPercentage, 0.7299970262933156)
	}
	if mov.ImpactPercentage != 1.4590347923681257 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.ImpactPercentage, 1.4590347923681257)
	}
	if mov.SlippageCost != 180.5 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.SlippageCost, 180.5)
	}

	// All the way up to the last price from best bid price
	mov, err = depth.HitTheBidsFromMid(20, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if mov.NominalPercentage != 0.7482229704451926 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.NominalPercentage, 0.7482229704451926)
	}
	if mov.ImpactPercentage != FullLiquidityExhaustedPercentage {
		t.Fatalf("received: '%v' but expected: '%v'", mov.ImpactPercentage, FullLiquidityExhaustedPercentage)
	}
	if mov.SlippageCost != 190 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.SlippageCost, 190)
	}
}

func TestHitTheBidsFromMid_QuotationRequired(t *testing.T) {
	t.Parallel()
	depth := NewDepth(id)
	_, err := depth.HitTheBidsFromMid(10, false)
	if !errors.Is(err, errNoLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoLiquidity)
	}
	depth.LoadSnapshot(bid, ask, 0, time.Time{}, true)
	mov, err := depth.HitTheBidsFromMid(26531, true)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if !mov.FullBookSideConsumed {
		t.Fatal("entire side should be consumed by this value")
	}

	mov, err = depth.HitTheBidsFromMid(1336, true)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if mov.NominalPercentage != 0.03741114852225963 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.NominalPercentage, 0.03741114852225963)
	}

	if mov.ImpactPercentage != 0.11223344556677892 { // mid price 1336.5 -> 1335
		t.Fatalf("received: '%v' but expected: '%v'", mov.ImpactPercentage, 0.11223344556677892)
	}
	if mov.SlippageCost != 0 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.SlippageCost, 0)
	}

	mov, err = depth.HitTheBidsFromMid(25871.5, true)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if mov.NominalPercentage != 0.7299970262933156 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.NominalPercentage, 0.7299970262933156)
	}
	if mov.ImpactPercentage != 1.4590347923681257 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.ImpactPercentage, 1.4590347923681257)
	}
	if mov.SlippageCost != 180.5 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.SlippageCost, 180.5)
	}

	// All the way up to the last price from best bid price
	mov, err = depth.HitTheBidsFromMid(26530, true)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if mov.NominalPercentage != 0.7482229704451926 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.NominalPercentage, 0.7482229704451926)
	}
	if mov.ImpactPercentage != FullLiquidityExhaustedPercentage {
		t.Fatalf("received: '%v' but expected: '%v'", mov.ImpactPercentage, FullLiquidityExhaustedPercentage)
	}
	if mov.SlippageCost != 190 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.SlippageCost, 190)
	}
}

func TestHitTheBidsFromBest(t *testing.T) {
	t.Parallel()
	depth := NewDepth(id)
	_, err := depth.HitTheBidsFromBest(10, false)
	if !errors.Is(err, errNoLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoLiquidity)
	}
	depth.LoadSnapshot(bid, ask, 0, time.Time{}, true)
	mov, err := depth.HitTheBidsFromBest(20.1, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if !mov.FullBookSideConsumed {
		t.Fatal("entire side should be consumed by this value")
	}

	mov, err = depth.HitTheBidsFromBest(1, false)
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

	mov, err = depth.HitTheBidsFromBest(19.5, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if mov.NominalPercentage != 0.692845079072617 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.NominalPercentage, 0.692845079072617)
	}
	if mov.ImpactPercentage != 1.4221556886227544 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.ImpactPercentage, 1.4221556886227544)
	}
	if mov.SlippageCost != 180.5 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.SlippageCost, 180.5)
	}

	// All the way up to the last price from best bid price
	mov, err = depth.HitTheBidsFromBest(20, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if mov.NominalPercentage != 0.7110778443113772 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.NominalPercentage, 0.7110778443113772)
	}
	if mov.ImpactPercentage != FullLiquidityExhaustedPercentage {
		t.Fatalf("received: '%v' but expected: '%v'", mov.ImpactPercentage, FullLiquidityExhaustedPercentage)
	}
	if mov.SlippageCost != 190 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.SlippageCost, 190)
	}
}

func TestHitTheBidsFromBest_QuotationRequired(t *testing.T) {
	t.Parallel()
	_, err := getInvalidDepth().HitTheBidsFromBest(10, false)
	if !errors.Is(err, ErrOrderbookInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderbookInvalid)
	}

	depth := NewDepth(id)
	_, err = depth.HitTheBidsFromBest(10, false)
	if !errors.Is(err, errNoLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoLiquidity)
	}
	depth.LoadSnapshot(bid, ask, 0, time.Time{}, true)
	mov, err := depth.HitTheBidsFromBest(26531, true)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if !mov.FullBookSideConsumed {
		t.Fatal("entire side should be consumed by this value")
	}

	mov, err = depth.HitTheBidsFromBest(1336, true)
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

	mov, err = depth.HitTheBidsFromBest(25871.5, true)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if mov.NominalPercentage != 0.692845079072617 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.NominalPercentage, 0.692845079072617)
	}
	if mov.ImpactPercentage != 1.4221556886227544 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.ImpactPercentage, 1.4221556886227544)
	}
	if mov.SlippageCost != 180.5 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.SlippageCost, 180.5)
	}

	// All the way up to the last price from best bid price
	mov, err = depth.HitTheBidsFromBest(26530, true)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if mov.NominalPercentage != 0.7110778443113772 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.NominalPercentage, 0.7110778443113772)
	}
	if mov.ImpactPercentage != FullLiquidityExhaustedPercentage {
		t.Fatalf("received: '%v' but expected: '%v'", mov.ImpactPercentage, FullLiquidityExhaustedPercentage)
	}
	if mov.SlippageCost != 190 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.SlippageCost, 190)
	}
}

func TestLiftTheAsks(t *testing.T) {
	t.Parallel()
	depth := NewDepth(id)
	depth.LoadSnapshot(bid, ask, 0, time.Time{}, true)
	mov, err := depth.LiftTheAsks(26931, 1337, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if !mov.FullBookSideConsumed {
		t.Fatal("entire side should be consumed by this value")
	}

	mov, err = depth.LiftTheAsks(1337, 1337, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if mov.NominalPercentage != 0 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.NominalPercentage, 0)
	}
	if mov.ImpactPercentage != 0.07479431563201197 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.ImpactPercentage, 0.07479431563201197)
	}
	if mov.SlippageCost != 0 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.SlippageCost, 0)
	}

	mov, err = depth.LiftTheAsks(26900, 1337, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if mov.NominalPercentage != 0.7097591258590459 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.NominalPercentage, 0.7097591258590459)
	}
	if mov.ImpactPercentage != 1.4210919970082274 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.ImpactPercentage, 1.4210919970082274)
	}
	if mov.SlippageCost != 189.57964601770072 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.SlippageCost, 189.57964601770072)
	}

	// All the way up to the last price from best bid price
	mov, err = depth.LiftTheAsks(26930, 1336, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if mov.NominalPercentage != 0.7859281437125748 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.NominalPercentage, 0.7859281437125748)
	}
	if mov.ImpactPercentage != FullLiquidityExhaustedPercentage {
		t.Fatalf("received: '%v' but expected: '%v'", mov.ImpactPercentage, FullLiquidityExhaustedPercentage)
	}
	if mov.SlippageCost != 190 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.SlippageCost, 190)
	}
}

func TestLiftTheAsks_BaseRequired(t *testing.T) {
	t.Parallel()
	_, err := getInvalidDepth().LiftTheAsks(21, 1337, true)
	if !errors.Is(err, ErrOrderbookInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderbookInvalid)
	}

	depth := NewDepth(id)
	depth.LoadSnapshot(bid, ask, 0, time.Time{}, true)
	mov, err := depth.LiftTheAsks(21, 1337, true)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if !mov.FullBookSideConsumed {
		t.Fatal("entire side should be consumed by this value")
	}

	mov, err = depth.LiftTheAsks(1, 1337, true)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if mov.NominalPercentage != 0 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.NominalPercentage, 0)
	}
	if mov.ImpactPercentage != 0.07479431563201197 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.ImpactPercentage, 0.07479431563201197)
	}
	if mov.SlippageCost != 0 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.SlippageCost, 0)
	}

	mov, err = depth.LiftTheAsks(19.97787610619469, 1337, true)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if mov.NominalPercentage != 0.7097591258590288 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.NominalPercentage, 0.7097591258590288)
	}
	if mov.ImpactPercentage != 1.4210919970082274 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.ImpactPercentage, 1.4210919970082274)
	}
	if mov.SlippageCost != 189.5796460176971 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.SlippageCost, 189.5796460176971)
	}

	// All the way up to the last price from best bid price
	mov, err = depth.LiftTheAsks(20, 1336, true)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if mov.NominalPercentage != 0.7859281437125748 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.NominalPercentage, 0.7859281437125748)
	}
	if mov.ImpactPercentage != FullLiquidityExhaustedPercentage {
		t.Fatalf("received: '%v' but expected: '%v'", mov.ImpactPercentage, FullLiquidityExhaustedPercentage)
	}
	if mov.SlippageCost != 190 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.SlippageCost, 190)
	}
}

func TestLiftTheAsksFromMid(t *testing.T) {
	t.Parallel()
	_, err := getInvalidDepth().LiftTheAsksFromMid(10, false)
	if !errors.Is(err, ErrOrderbookInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderbookInvalid)
	}

	depth := NewDepth(id)
	_, err = depth.LiftTheAsksFromMid(10, false)
	if !errors.Is(err, errNoLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoLiquidity)
	}
	depth.LoadSnapshot(bid, ask, 0, time.Time{}, true)
	mov, err := depth.LiftTheAsksFromMid(26931, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if !mov.FullBookSideConsumed {
		t.Fatal("entire side should be consumed by this value")
	}

	mov, err = depth.LiftTheAsksFromMid(1337, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if mov.NominalPercentage != 0.03741114852225963 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.NominalPercentage, 0.03741114852225963)
	}
	if mov.ImpactPercentage != 0.11223344556677892 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.ImpactPercentage, 0.11223344556677892)
	}
	if mov.SlippageCost != 0 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.SlippageCost, 0)
	}

	mov, err = depth.LiftTheAsksFromMid(26900, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if mov.NominalPercentage != 0.747435803422031 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.NominalPercentage, 0.747435803422031)
	}
	if mov.ImpactPercentage != 1.4590347923681257 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.ImpactPercentage, 1.4590347923681257)
	}
	if mov.SlippageCost != 189.57964601770072 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.SlippageCost, 189.57964601770072)
	}

	// All the way up to the last price from best bid price
	mov, err = depth.LiftTheAsksFromMid(26930, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if mov.NominalPercentage != 0.7482229704451926 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.NominalPercentage, 0.7482229704451926)
	}
	if mov.ImpactPercentage != FullLiquidityExhaustedPercentage {
		t.Fatalf("received: '%v' but expected: '%v'", mov.ImpactPercentage, FullLiquidityExhaustedPercentage)
	}
	if mov.SlippageCost != 190 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.SlippageCost, 190)
	}
}

func TestLiftTheAsksFromMid_BaseRequired(t *testing.T) {
	t.Parallel()
	_, err := getInvalidDepth().LiftTheAsksFromMid(10, false)
	if !errors.Is(err, ErrOrderbookInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderbookInvalid)
	}

	depth := NewDepth(id)
	_, err = depth.LiftTheAsksFromMid(10, false)
	if !errors.Is(err, errNoLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoLiquidity)
	}
	depth.LoadSnapshot(bid, ask, 0, time.Time{}, true)
	mov, err := depth.LiftTheAsksFromMid(21, true)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if !mov.FullBookSideConsumed {
		t.Fatal("entire side should be consumed by this value")
	}

	mov, err = depth.LiftTheAsksFromMid(1, true)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if mov.NominalPercentage != 0.03741114852225963 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.NominalPercentage, 0.03741114852225963)
	}
	if mov.ImpactPercentage != 0.11223344556677892 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.ImpactPercentage, 0.11223344556677892)
	}
	if mov.SlippageCost != 0 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.SlippageCost, 0)
	}

	mov, err = depth.LiftTheAsksFromMid(19.97787610619469, true)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if mov.NominalPercentage != 0.7474358034220139 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.NominalPercentage, 0.7474358034220139)
	}
	if mov.ImpactPercentage != 1.4590347923681257 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.ImpactPercentage, 1.4590347923681257)
	}
	if mov.SlippageCost != 189.5796460176971 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.SlippageCost, 189.5796460176971)
	}

	// All the way up to the last price from best bid price
	mov, err = depth.LiftTheAsksFromMid(20, true)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if mov.NominalPercentage != 0.7482229704451926 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.NominalPercentage, 0.7482229704451926)
	}
	if mov.ImpactPercentage != FullLiquidityExhaustedPercentage {
		t.Fatalf("received: '%v' but expected: '%v'", mov.ImpactPercentage, FullLiquidityExhaustedPercentage)
	}
	if mov.SlippageCost != 190 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.SlippageCost, 190)
	}
}

func TestLiftTheAsksFromBest(t *testing.T) {
	t.Parallel()
	_, err := getInvalidDepth().LiftTheAsksFromBest(10, false)
	if !errors.Is(err, ErrOrderbookInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderbookInvalid)
	}

	depth := NewDepth(id)
	_, err = depth.LiftTheAsksFromBest(10, false)
	if !errors.Is(err, errNoLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoLiquidity)
	}
	depth.LoadSnapshot(bid, ask, 0, time.Time{}, true)
	mov, err := depth.LiftTheAsksFromBest(26931, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if !mov.FullBookSideConsumed {
		t.Fatal("entire side should be consumed by this value")
	}

	mov, err = depth.LiftTheAsksFromBest(1337, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if mov.NominalPercentage != 0 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.NominalPercentage, 0)
	}
	if mov.ImpactPercentage != 0.07479431563201197 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.ImpactPercentage, 0.07479431563201197)
	}
	if mov.SlippageCost != 0 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.SlippageCost, 0)
	}

	mov, err = depth.LiftTheAsksFromBest(26900, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if mov.NominalPercentage != 0.7097591258590459 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.NominalPercentage, 0.7097591258590459)
	}
	if mov.ImpactPercentage != 1.4210919970082274 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.ImpactPercentage, 1.4210919970082274)
	}
	if mov.SlippageCost != 189.57964601770072 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.SlippageCost, 189.57964601770072)
	}

	// All the way up to the last price from best bid price
	mov, err = depth.LiftTheAsksFromBest(26930, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if mov.NominalPercentage != 0.7105459985041137 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.NominalPercentage, 0.7105459985041137)
	}
	if mov.ImpactPercentage != FullLiquidityExhaustedPercentage {
		t.Fatalf("received: '%v' but expected: '%v'", mov.ImpactPercentage, FullLiquidityExhaustedPercentage)
	}
	if mov.SlippageCost != 190 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.SlippageCost, 190)
	}
}

func TestLiftTheAsksFromBest_BaseRequired(t *testing.T) {
	t.Parallel()
	_, err := getInvalidDepth().LiftTheAsksFromBest(10, false)
	if !errors.Is(err, ErrOrderbookInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderbookInvalid)
	}

	depth := NewDepth(id)
	_, err = depth.LiftTheAsksFromBest(10, false)
	if !errors.Is(err, errNoLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoLiquidity)
	}
	depth.LoadSnapshot(bid, ask, 0, time.Time{}, true)
	mov, err := depth.LiftTheAsksFromBest(21, true)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if !mov.FullBookSideConsumed {
		t.Fatal("entire side should be consumed by this value")
	}

	mov, err = depth.LiftTheAsksFromBest(1, true)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if mov.NominalPercentage != 0 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.NominalPercentage, 0)
	}
	if mov.ImpactPercentage != 0.07479431563201197 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.ImpactPercentage, 0.07479431563201197)
	}
	if mov.SlippageCost != 0 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.SlippageCost, 0)
	}

	mov, err = depth.LiftTheAsksFromBest(19.97787610619469, true)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
	if mov.NominalPercentage != 0.7097591258590288 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.NominalPercentage, 0.7097591258590288)
	}
	if mov.ImpactPercentage != 1.4210919970082274 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.ImpactPercentage, 1.4210919970082274)
	}
	if mov.SlippageCost != 189.5796460176971 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.SlippageCost, 189.5796460176971)
	}

	// All the way up to the last price from best bid price
	mov, err = depth.LiftTheAsksFromBest(20, true)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if mov.NominalPercentage != 0.7105459985041137 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.NominalPercentage, 0.7105459985041137)
	}
	if mov.ImpactPercentage != FullLiquidityExhaustedPercentage {
		t.Fatalf("received: '%v' but expected: '%v'", mov.ImpactPercentage, FullLiquidityExhaustedPercentage)
	}
	if mov.SlippageCost != 190 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.SlippageCost, 190)
	}
}

func TestGetMidPrice_Depth(t *testing.T) {
	t.Parallel()
	_, err := getInvalidDepth().GetMidPrice()
	if !errors.Is(err, ErrOrderbookInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderbookInvalid)
	}

	depth := NewDepth(id)
	_, err = depth.GetMidPrice()
	if !errors.Is(err, errNoLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoLiquidity)
	}
	depth.LoadSnapshot(bid, ask, 0, time.Time{}, true)
	mid, err := depth.GetMidPrice()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if mid != 1336.5 {
		t.Fatalf("received: '%v' but expected: '%v'", mid, 1336.5)
	}
}

func TestGetMidPriceNoLock_Depth(t *testing.T) {
	t.Parallel()
	depth := NewDepth(id)
	_, err := depth.getMidPriceNoLock()
	if !errors.Is(err, errNoLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoLiquidity)
	}
	depth.LoadSnapshot(bid, nil, 0, time.Time{}, true)
	_, err = depth.getMidPriceNoLock()
	if !errors.Is(err, errNoLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoLiquidity)
	}

	depth.LoadSnapshot(bid, ask, 0, time.Time{}, true)
	mid, err := depth.getMidPriceNoLock()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if mid != 1336.5 {
		t.Fatalf("received: '%v' but expected: '%v'", mid, 1336.5)
	}
}

func TestGetBestBidASk_Depth(t *testing.T) {
	t.Parallel()
	_, err := getInvalidDepth().GetBestBid()
	if !errors.Is(err, ErrOrderbookInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderbookInvalid)
	}

	_, err = getInvalidDepth().GetBestAsk()
	if !errors.Is(err, ErrOrderbookInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderbookInvalid)
	}

	depth := NewDepth(id)
	_, err = depth.GetBestBid()
	if !errors.Is(err, errNoLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoLiquidity)
	}
	_, err = depth.GetBestAsk()
	if !errors.Is(err, errNoLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoLiquidity)
	}
	depth.LoadSnapshot(bid, ask, 0, time.Time{}, true)
	mid, err := depth.GetBestBid()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
	if mid != 1336 {
		t.Fatalf("received: '%v' but expected: '%v'", mid, 1336)
	}
	mid, err = depth.GetBestAsk()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
	if mid != 1337 {
		t.Fatalf("received: '%v' but expected: '%v'", mid, 1337)
	}
}

func TestGetSpreadAmount(t *testing.T) {
	t.Parallel()
	_, err := getInvalidDepth().GetSpreadAmount()
	if !errors.Is(err, ErrOrderbookInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderbookInvalid)
	}

	depth := NewDepth(id)

	_, err = depth.GetSpreadAmount()
	if !errors.Is(err, errNoLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoLiquidity)
	}

	depth.LoadSnapshot(nil, ask, 0, time.Time{}, true)

	_, err = depth.GetSpreadAmount()
	if !errors.Is(err, errNoLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoLiquidity)
	}

	depth.LoadSnapshot(bid, ask, 0, time.Time{}, true)

	spread, err := depth.GetSpreadAmount()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if spread != 1 {
		t.Fatalf("received: '%v' but expected: '%v'", spread, 1)
	}
}

func TestGetSpreadPercentage(t *testing.T) {
	t.Parallel()
	_, err := getInvalidDepth().GetSpreadPercentage()
	if !errors.Is(err, ErrOrderbookInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderbookInvalid)
	}

	depth := NewDepth(id)

	_, err = depth.GetSpreadPercentage()
	if !errors.Is(err, errNoLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoLiquidity)
	}

	depth.LoadSnapshot(nil, ask, 0, time.Time{}, true)

	_, err = depth.GetSpreadPercentage()
	if !errors.Is(err, errNoLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoLiquidity)
	}

	depth.LoadSnapshot(bid, ask, 0, time.Time{}, true)

	spread, err := depth.GetSpreadPercentage()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if spread != 0.07479431563201197 {
		t.Fatalf("received: '%v' but expected: '%v'", spread, 0.07479431563201197)
	}
}

func TestGetImbalance_Depth(t *testing.T) {
	t.Parallel()
	_, err := getInvalidDepth().GetImbalance()
	if !errors.Is(err, ErrOrderbookInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderbookInvalid)
	}

	depth := NewDepth(id)

	_, err = depth.GetImbalance()
	if !errors.Is(err, errNoLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoLiquidity)
	}

	depth.LoadSnapshot(nil, ask, 0, time.Time{}, true)

	_, err = depth.GetImbalance()
	if !errors.Is(err, errNoLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoLiquidity)
	}

	depth.LoadSnapshot(bid, ask, 0, time.Time{}, true)

	imbalance, err := depth.GetImbalance()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if imbalance != 0 {
		t.Fatalf("received: '%v' but expected: '%v'", imbalance, 0)
	}
}

func TestGetTranches(t *testing.T) {
	t.Parallel()
	_, _, err := getInvalidDepth().GetTranches(0)
	if !errors.Is(err, ErrOrderbookInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrOrderbookInvalid)
	}

	depth := NewDepth(id)

	_, _, err = depth.GetTranches(-1)
	if !errors.Is(err, errInvalidBookDepth) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidBookDepth)
	}

	askT, bidT, err := depth.GetTranches(0)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if len(askT) != 0 {
		t.Fatalf("received: '%v' but expected: '%v'", len(askT), 0)
	}

	if len(bidT) != 0 {
		t.Fatalf("received: '%v' but expected: '%v'", len(bidT), 0)
	}

	depth.LoadSnapshot(bid, ask, 0, time.Time{}, true)

	askT, bidT, err = depth.GetTranches(0)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if len(askT) != 20 {
		t.Fatalf("received: '%v' but expected: '%v'", len(askT), 20)
	}

	if len(bidT) != 20 {
		t.Fatalf("received: '%v' but expected: '%v'", len(bidT), 20)
	}

	askT, bidT, err = depth.GetTranches(5)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if len(askT) != 5 {
		t.Fatalf("received: '%v' but expected: '%v'", len(askT), 5)
	}

	if len(bidT) != 5 {
		t.Fatalf("received: '%v' but expected: '%v'", len(bidT), 5)
	}
}

func TestGetPair(t *testing.T) {
	t.Parallel()
	depth := NewDepth(id)

	_, err := depth.GetPair()
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Fatalf("received: '%v' but expected: '%v'", err, currency.ErrCurrencyPairEmpty)
	}

	expected := currency.NewPair(currency.BTC, currency.WABI)
	depth.pair = expected

	pair, err := depth.GetPair()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if !pair.Equal(expected) {
		t.Fatalf("received: '%v' but expected: '%v'", pair, expected)
	}
}

func getInvalidDepth() *Depth {
	depth := NewDepth(id)
	_ = depth.Invalidate(errors.New("invalid reasoning"))
	return depth
}
