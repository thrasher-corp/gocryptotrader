package orderbook

import (
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var (
	id           = uuid.Must(uuid.NewV4())
	accuracy10dp = 1 / math.Pow10(10)
)

func TestGetLength(t *testing.T) {
	t.Parallel()
	d := NewDepth(id)
	err := d.Invalidate(nil)
	assert.ErrorIs(t, err, ErrOrderbookInvalid, "Invalidate should error correctly")

	_, err = d.GetAskLength()
	assert.ErrorIs(t, err, ErrOrderbookInvalid, "GetAskLength should error with invalid depth")

	err = d.LoadSnapshot([]Tranche{{Price: 1337}}, nil, 0, time.Now(), true)
	assert.NoError(t, err, "LoadSnapshot should not error")

	askLen, err := d.GetAskLength()
	assert.NoError(t, err, "GetAskLength should not error")
	assert.Zero(t, askLen, "ask length should be zero")

	d.askTranches.load([]Tranche{{Price: 1337}})

	askLen, err = d.GetAskLength()
	assert.NoError(t, err, "GetAskLength should not error")
	assert.Equal(t, 1, askLen, "Ask Length should be correct")

	d = NewDepth(id)
	err = d.Invalidate(nil)
	assert.ErrorIs(t, err, ErrOrderbookInvalid, "Invalidate should error correctly")

	_, err = d.GetBidLength()
	assert.ErrorIs(t, err, ErrOrderbookInvalid, "GetBidLength should error with invalid depth")

	err = d.LoadSnapshot(nil, []Tranche{{Price: 1337}}, 0, time.Now(), true)
	assert.NoError(t, err, "LoadSnapshot should not error")

	bidLen, err := d.GetBidLength()
	assert.NoError(t, err, "GetBidLength should not error")
	assert.Zero(t, bidLen, "bid length should be zero")

	d.bidTranches.load([]Tranche{{Price: 1337}})

	bidLen, err = d.GetBidLength()
	assert.NoError(t, err, "GetBidLength should not error")
	assert.Equal(t, 1, bidLen, "Bid Length should be correct")
}

func TestRetrieve(t *testing.T) {
	t.Parallel()
	d := NewDepth(id)
	d.askTranches.load([]Tranche{{Price: 1337}})
	d.bidTranches.load([]Tranche{{Price: 1337}})
	d.options = options{
		exchange:               "THE BIG ONE!!!!!!",
		pair:                   currency.NewPair(currency.THETA, currency.USD),
		asset:                  asset.DownsideProfitContract,
		lastUpdated:            time.Now(),
		lastUpdateID:           1337,
		priceDuplication:       true,
		isFundingRate:          true,
		VerifyOrderbook:        true,
		restSnapshot:           true,
		idAligned:              true,
		maxDepth:               10,
		checksumStringRequired: true,
	}

	// If we add anymore options to the options struct later this will complain
	// generally want to return a full carbon copy
	mirrored := reflect.Indirect(reflect.ValueOf(d.options))
	for n := 0; n < mirrored.NumField(); n++ {
		structVal := mirrored.Field(n)
		assert.Falsef(t, structVal.IsZero(), "struct field '%s' not tested", mirrored.Type().Field(n).Name)
	}

	ob, err := d.Retrieve()
	assert.NoError(t, err, "Retrieve should not error")
	assert.Len(t, ob.Asks, 1, "Should have correct Asks")
	assert.Len(t, ob.Bids, 1, "Should have correct Bids")
	assert.Equal(t, 10, ob.MaxDepth, "Should have correct MaxDepth")
}

func TestTotalAmounts(t *testing.T) {
	t.Parallel()
	d := NewDepth(id)

	err := d.Invalidate(nil)
	assert.ErrorIs(t, err, ErrOrderbookInvalid, "Invalidate should error correctly")
	_, _, err = d.TotalBidAmounts()
	assert.ErrorIs(t, err, ErrOrderbookInvalid, "TotalBidAmounts should error correctly")

	d.validationError = nil
	liquidity, value, err := d.TotalBidAmounts()
	assert.NoError(t, err, "TotalBidAmounts should not error")
	assert.Zero(t, liquidity, "total bid liquidity should be zero")
	assert.Zero(t, value, "total bid value should be zero")

	err = d.Invalidate(nil)
	assert.ErrorIs(t, err, ErrOrderbookInvalid, "Invalidate should error correctly")

	_, _, err = d.TotalAskAmounts()
	assert.ErrorIs(t, err, ErrOrderbookInvalid, "TotalAskAmounts should error correctly")

	d.validationError = nil

	liquidity, value, err = d.TotalAskAmounts()
	assert.NoError(t, err, "TotalAskAmounts should not error")
	assert.Zero(t, liquidity, "total ask liquidity should be zero")
	assert.Zero(t, value, "total ask value should be zero")

	d.askTranches.load([]Tranche{{Price: 1337, Amount: 1}})
	d.bidTranches.load([]Tranche{{Price: 1337, Amount: 10}})

	liquidity, value, err = d.TotalBidAmounts()
	assert.NoError(t, err, "TotalBidAmounts should not error")
	assert.Equal(t, 10.0, liquidity, "total bid liquidity should be correct")
	assert.Equal(t, 13370.0, value, "total bid value should be correct")

	liquidity, value, err = d.TotalAskAmounts()
	assert.NoError(t, err, "TotalAskAmounts should not error")
	assert.Equal(t, 1.0, liquidity, "total ask liquidity should be correct")
	assert.Equal(t, 1337.0, value, "total ask value should be correct")
}

func TestLoadSnapshot(t *testing.T) {
	t.Parallel()
	d := NewDepth(id)
	err := d.LoadSnapshot(Tranches{{Price: 1337, Amount: 1}}, Tranches{{Price: 1337, Amount: 10}}, 0, time.Time{}, false)
	assert.ErrorIs(t, err, errLastUpdatedNotSet, "LoadSnapshot should error correctly")

	err = d.LoadSnapshot(Tranches{{Price: 1337, Amount: 2}}, Tranches{{Price: 1338, Amount: 10}}, 0, time.Now(), false)
	assert.NoError(t, err, "LoadSnapshot should not error")

	ob, err := d.Retrieve()
	assert.NoError(t, err, "Retrieve should not error")

	assert.Equal(t, 1338.0, ob.Asks[0].Price, "Top ask price should be correct")
	assert.Equal(t, 10.0, ob.Asks[0].Amount, "Top ask amount should be correct")
	assert.Equal(t, 1337.0, ob.Bids[0].Price, "Top bid price should be correct")
	assert.Equal(t, 2.0, ob.Bids[0].Amount, "Top bid amount should be correct")
}

func TestInvalidate(t *testing.T) {
	t.Parallel()
	d := NewDepth(id)
	d.exchange = "testexchange"
	d.pair = currency.NewPair(currency.BTC, currency.WABI)
	d.asset = asset.Spot

	err := d.LoadSnapshot(Tranches{{Price: 1337, Amount: 1}}, Tranches{{Price: 1337, Amount: 10}}, 0, time.Now(), false)
	assert.NoError(t, err, "LoadSnapshot should not error")

	ob, err := d.Retrieve()
	assert.NoError(t, err, "Retrieve should not error")
	assert.NotNil(t, ob, "ob should not be nil")

	testReason := errors.New("random reason")

	err = d.Invalidate(testReason)
	assert.ErrorIs(t, err, ErrOrderbookInvalid, "Invalidate should error correctly")

	_, err = d.Retrieve()
	assert.ErrorIs(t, err, ErrOrderbookInvalid, "Retrieve should error correctly")
	assert.ErrorIs(t, err, testReason, "Invalidate should error correctly")

	d.validationError = nil

	ob, err = d.Retrieve()
	assert.NoError(t, err, "Retrieve should not error")

	assert.Empty(t, ob.Asks, "Orderbook Asks should be flushed")
	assert.Empty(t, ob.Bids, "Orderbook Bids should be flushed")
}

func TestUpdateBidAskByPrice(t *testing.T) {
	t.Parallel()
	d := NewDepth(id)
	err := d.LoadSnapshot(Tranches{{Price: 1337, Amount: 1, ID: 1}}, Tranches{{Price: 1338, Amount: 10, ID: 2}}, 0, time.Now(), false)
	assert.NoError(t, err, "LoadSnapshot should not error")

	err = d.UpdateBidAskByPrice(&Update{})
	assert.ErrorIs(t, err, errLastUpdatedNotSet, "UpdateBidAskByPrice should error correctly")

	err = d.UpdateBidAskByPrice(&Update{UpdateTime: time.Now()})
	assert.NoError(t, err, "UpdateBidAskByPrice should not error")

	updates := &Update{
		Bids:       Tranches{{Price: 1337, Amount: 2, ID: 1}},
		Asks:       Tranches{{Price: 1338, Amount: 3, ID: 2}},
		UpdateID:   1,
		UpdateTime: time.Now(),
	}
	err = d.UpdateBidAskByPrice(updates)
	assert.NoError(t, err, "UpdateBidAskByPrice should not error")

	ob, err := d.Retrieve()
	assert.NoError(t, err, "Retrieve should not error")
	assert.Equal(t, 3.0, ob.Asks[0].Amount, "Asks amount should be correct")
	assert.Equal(t, 2.0, ob.Bids[0].Amount, "Bids amount should be correct")

	updates = &Update{
		Bids:       Tranches{{Price: 1337, Amount: 0, ID: 1}},
		Asks:       Tranches{{Price: 1338, Amount: 0, ID: 2}},
		UpdateID:   2,
		UpdateTime: time.Now(),
	}
	err = d.UpdateBidAskByPrice(updates)
	assert.NoError(t, err, "UpdateBidAskByPrice should not error")

	askLen, err := d.GetAskLength()
	assert.NoError(t, err, "GetAskLength should not error")
	assert.Zero(t, askLen, "Ask Length should be correct")

	bidLen, err := d.GetBidLength()
	assert.NoError(t, err, "GetBidLength should not error")
	assert.Zero(t, bidLen, "Bid Length should be correct")
}

func TestDeleteBidAskByID(t *testing.T) {
	t.Parallel()
	d := NewDepth(id)
	err := d.LoadSnapshot(Tranches{{Price: 1337, Amount: 1, ID: 1}}, Tranches{{Price: 1337, Amount: 10, ID: 2}}, 0, time.Now(), false)
	assert.NoError(t, err, "LoadSnapshot should not error")

	updates := &Update{
		Bids: Tranches{{Price: 1337, Amount: 2, ID: 1}},
		Asks: Tranches{{Price: 1337, Amount: 2, ID: 2}},
	}

	err = d.DeleteBidAskByID(updates, false)
	assert.ErrorIs(t, err, errLastUpdatedNotSet, "DeleteBidAskByID should error correctly")

	updates.UpdateTime = time.Now()
	err = d.DeleteBidAskByID(updates, false)
	assert.NoError(t, err, "DeleteBidAskByID should not error")

	ob, err := d.Retrieve()
	assert.NoError(t, err, "Retrieve should not error")
	assert.Empty(t, ob.Asks, "Asks should be empty")
	assert.Empty(t, ob.Bids, "Bids should be empty")

	updates = &Update{
		Bids:       Tranches{{Price: 1337, Amount: 2, ID: 1}},
		UpdateTime: time.Now(),
	}
	err = d.DeleteBidAskByID(updates, false)
	assert.ErrorIs(t, err, errIDCannotBeMatched, "DeleteBidAskByID should error correctly")

	updates = &Update{
		Asks:       Tranches{{Price: 1337, Amount: 2, ID: 2}},
		UpdateTime: time.Now(),
	}
	err = d.DeleteBidAskByID(updates, false)
	assert.ErrorIs(t, err, errIDCannotBeMatched, "DeleteBidAskByID should error correctly")

	updates = &Update{
		Asks:       Tranches{{Price: 1337, Amount: 2, ID: 2}},
		UpdateTime: time.Now(),
	}
	err = d.DeleteBidAskByID(updates, true)
	assert.NoError(t, err, "DeleteBidAskByID should not error")
}

func TestUpdateBidAskByID(t *testing.T) {
	t.Parallel()
	d := NewDepth(id)
	err := d.LoadSnapshot(Tranches{{Price: 1337, Amount: 1, ID: 1}}, Tranches{{Price: 1337, Amount: 10, ID: 2}}, 0, time.Now(), false)
	assert.NoError(t, err, "LoadSnapshot should not error")

	updates := &Update{
		Bids: Tranches{{Price: 1337, Amount: 2, ID: 1}},
		Asks: Tranches{{Price: 1337, Amount: 2, ID: 2}},
	}

	err = d.UpdateBidAskByID(updates)
	assert.ErrorIs(t, err, errLastUpdatedNotSet, "UpdateBidAskByID should error correctly")

	updates.UpdateTime = time.Now()
	err = d.UpdateBidAskByID(updates)
	assert.NoError(t, err, "UpdateBidAskByID should not error")

	ob, err := d.Retrieve()
	assert.NoError(t, err, "Retrieve should not error")
	assert.Equal(t, 2.0, ob.Asks[0].Amount, "First ask amount should be correct")
	assert.Equal(t, 2.0, ob.Bids[0].Amount, "First bid amount should be correct")

	updates = &Update{
		Bids:       Tranches{{Price: 1337, Amount: 2, ID: 666}},
		UpdateTime: time.Now(),
	}
	// random unmatching IDs
	err = d.UpdateBidAskByID(updates)
	assert.ErrorIs(t, err, errIDCannotBeMatched, "UpdateBidAskByID should error correctly")

	updates = &Update{
		Asks:       Tranches{{Price: 1337, Amount: 2, ID: 69}},
		UpdateTime: time.Now(),
	}
	err = d.UpdateBidAskByID(updates)
	assert.ErrorIs(t, err, errIDCannotBeMatched, "UpdateBidAskByID should error correctly")
}

func TestInsertBidAskByID(t *testing.T) {
	t.Parallel()
	d := NewDepth(id)
	err := d.LoadSnapshot(Tranches{{Price: 1337, Amount: 1, ID: 1}}, Tranches{{Price: 1337, Amount: 10, ID: 2}}, 0, time.Now(), false)
	assert.NoError(t, err, "LoadSnapshot should not error")

	updates := &Update{
		Asks: Tranches{{Price: 1337, Amount: 2, ID: 3}},
	}
	err = d.InsertBidAskByID(updates)
	assert.ErrorIs(t, err, errLastUpdatedNotSet, "InsertBidAskByID should error correctly")

	updates.UpdateTime = time.Now()

	err = d.InsertBidAskByID(updates)
	assert.ErrorIs(t, err, errCollisionDetected, "InsertBidAskByID should error correctly on collision")

	err = d.LoadSnapshot(Tranches{{Price: 1337, Amount: 1, ID: 1}}, Tranches{{Price: 1337, Amount: 10, ID: 2}}, 0, time.Now(), false)
	assert.NoError(t, err, "LoadSnapshot should not error")

	updates = &Update{
		Bids:       Tranches{{Price: 1337, Amount: 2, ID: 3}},
		UpdateTime: time.Now(),
	}

	err = d.InsertBidAskByID(updates)
	assert.ErrorIs(t, err, errCollisionDetected, "InsertBidAskByID should error correctly on collision")

	err = d.LoadSnapshot(Tranches{{Price: 1337, Amount: 1, ID: 1}}, Tranches{{Price: 1337, Amount: 10, ID: 2}}, 0, time.Now(), false)
	assert.NoError(t, err, "LoadSnapshot should not error")

	updates = &Update{
		Bids:       Tranches{{Price: 1338, Amount: 2, ID: 3}},
		Asks:       Tranches{{Price: 1336, Amount: 2, ID: 4}},
		UpdateTime: time.Now(),
	}
	err = d.InsertBidAskByID(updates)
	assert.NoError(t, err, "InsertBidAskByID should not error")

	ob, err := d.Retrieve()
	assert.NoError(t, err, "Retrieve should not error")
	assert.Len(t, ob.Asks, 2, "Should have correct Asks")
	assert.Len(t, ob.Bids, 2, "Should have correct Bids")
}

func TestUpdateInsertByID(t *testing.T) {
	t.Parallel()
	d := NewDepth(id)
	err := d.LoadSnapshot(Tranches{{Price: 1337, Amount: 1, ID: 1}}, Tranches{{Price: 1337, Amount: 10, ID: 2}}, 0, time.Now(), false)
	assert.NoError(t, err, "LoadSnapshot should not error")

	updates := &Update{
		Bids: Tranches{{Price: 1338, Amount: 0, ID: 3}},
		Asks: Tranches{{Price: 1336, Amount: 2, ID: 4}},
	}
	err = d.UpdateInsertByID(updates)
	assert.ErrorIs(t, err, errLastUpdatedNotSet, "UpdateInsertByID should error correctly")

	updates.UpdateTime = time.Now()
	err = d.UpdateInsertByID(updates)
	assert.ErrorIs(t, err, errAmountCannotBeLessOrEqualToZero, "UpdateInsertByID should error correctly")

	// Above will invalidate the book
	_, err = d.Retrieve()
	assert.ErrorIs(t, err, ErrOrderbookInvalid, "Retrieve should error correctly")

	err = d.LoadSnapshot(Tranches{{Price: 1337, Amount: 1, ID: 1}}, Tranches{{Price: 1337, Amount: 10, ID: 2}}, 0, time.Now(), false)
	assert.NoError(t, err, "LoadSnapshot should not error")

	updates = &Update{
		Bids:       Tranches{{Price: 1338, Amount: 2, ID: 3}},
		Asks:       Tranches{{Price: 1336, Amount: 0, ID: 4}},
		UpdateTime: time.Now(),
	}
	err = d.UpdateInsertByID(updates)
	assert.ErrorIs(t, err, errAmountCannotBeLessOrEqualToZero, "UpdateInsertByID should error correctly")

	// Above will invalidate the book
	_, err = d.Retrieve()
	assert.ErrorIs(t, err, ErrOrderbookInvalid, "Retrieve should error correctly")

	err = d.LoadSnapshot(Tranches{{Price: 1337, Amount: 1, ID: 1}}, Tranches{{Price: 1337, Amount: 10, ID: 2}}, 0, time.Now(), false)
	assert.NoError(t, err, "LoadSnapshot should not error")

	updates = &Update{
		Bids:       Tranches{{Price: 1338, Amount: 2, ID: 3}},
		Asks:       Tranches{{Price: 1336, Amount: 2, ID: 4}},
		UpdateTime: time.Now(),
	}
	err = d.UpdateInsertByID(updates)
	assert.NoError(t, err, "UpdateInsertByID should not error")

	ob, err := d.Retrieve()
	assert.NoError(t, err, "Retrieve should not error")
	assert.Len(t, ob.Asks, 2, "Should have correct Asks")
	assert.Len(t, ob.Bids, 2, "Should have correct Bids")
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

	assert.Equal(t, "test", d.exchange, "exchange should be correct")
	assert.Equal(t, cp, d.pair, "pair should be correct")
	assert.Equal(t, asset.Spot, d.asset, "asset should be correct")
	assert.Equal(t, tn, d.lastUpdated, "lastUpdated should be correct")
	assert.EqualValues(t, 1337, d.lastUpdateID, "lastUpdatedID should be correct")
	assert.True(t, d.priceDuplication, "priceDuplication should be correct")
	assert.True(t, d.isFundingRate, "isFundingRate should be correct")
	assert.True(t, d.VerifyOrderbook, "VerifyOrderbook should be correct")
	assert.True(t, d.restSnapshot, "restSnapshot should be correct")
	assert.True(t, d.idAligned, "idAligned should be correct")
}

func TestGetName(t *testing.T) {
	t.Parallel()
	d := Depth{}
	d.exchange = "test"
	assert.Equal(t, "test", d.GetName(), "GetName should return correct value")
}

func TestIsRestSnapshot(t *testing.T) {
	t.Parallel()
	d := Depth{}
	d.restSnapshot = true
	err := d.Invalidate(nil)
	assert.ErrorIs(t, err, ErrOrderbookInvalid, "Invalidate should error correctly")
	_, err = d.IsRESTSnapshot()
	assert.ErrorIs(t, err, ErrOrderbookInvalid, "IsRESTSnapshot should error correctly")

	d.validationError = nil
	b, err := d.IsRESTSnapshot()
	assert.NoError(t, err, "IsRESTSnapshot should not error")
	assert.True(t, b, "IsRESTSnapshot should return correct value")
}

func TestLastUpdateID(t *testing.T) {
	t.Parallel()
	d := Depth{}
	err := d.Invalidate(nil)
	assert.ErrorIs(t, err, ErrOrderbookInvalid, "Invalidate should error correctly")

	_, err = d.LastUpdateID()
	assert.ErrorIs(t, err, ErrOrderbookInvalid, "LastUpdateID should error correctly")

	d.validationError = nil
	d.lastUpdateID = 1337
	id, err := d.LastUpdateID()
	assert.NoError(t, err, "LastUpdateID should not error")

	assert.EqualValues(t, 1337, id, "LastUpdateID should return correct value")
}

func TestIsFundingRate(t *testing.T) {
	t.Parallel()
	d := Depth{}
	d.isFundingRate = true
	assert.True(t, d.IsFundingRate(), "IsFundingRate should return true")
}

func TestPublish(t *testing.T) {
	t.Parallel()
	d := Depth{}
	err := d.Invalidate(nil)
	assert.ErrorIs(t, err, ErrOrderbookInvalid, "Invalidate should error correctly")
	d.Publish()
	d.validationError = nil
	d.Publish()
}

func TestIsValid(t *testing.T) {
	t.Parallel()
	d := Depth{}
	assert.True(t, d.IsValid(), "IsValid should return correct value")
	err := d.Invalidate(nil)
	assert.ErrorIs(t, err, ErrOrderbookInvalid, "Invalidate should error correctly")
	assert.False(t, d.IsValid(), "IsValid should return correct value after Invalidate")
}

func TestGetMidPrice_Depth(t *testing.T) {
	t.Parallel()
	_, err := getInvalidDepth().GetMidPrice()
	assert.ErrorIs(t, err, ErrOrderbookInvalid, "GetMidPrice should error correctly")

	depth := NewDepth(id)
	_, err = depth.GetMidPrice()
	assert.ErrorIs(t, err, errNoLiquidity, "GetMidPrice should error correctly")

	err = depth.LoadSnapshot(bid, ask, 0, time.Now(), true)
	assert.NoError(t, err, "LoadSnapshot should not error")

	mid, err := depth.GetMidPrice()
	assert.NoError(t, err, "GetMidPrice should not error")
	assert.Equal(t, 1336.5, mid, "Mid price should be correct")
}

func TestGetMidPriceNoLock_Depth(t *testing.T) {
	t.Parallel()
	depth := NewDepth(id)
	_, err := depth.getMidPriceNoLock()
	assert.ErrorIs(t, err, errNoLiquidity, "getMidPriceNoLock should error correctly")

	err = depth.LoadSnapshot(bid, nil, 0, time.Now(), true)
	assert.NoError(t, err, "LoadSnapshot should not error")

	_, err = depth.getMidPriceNoLock()
	assert.ErrorIs(t, err, errNoLiquidity, "getMidPriceNoLock should error correctly")

	err = depth.LoadSnapshot(bid, ask, 0, time.Now(), true)
	assert.NoError(t, err, "LoadSnapshot should not error")

	mid, err := depth.getMidPriceNoLock()
	assert.NoError(t, err, "getMidPriceNoLock should not error")
	assert.Equal(t, 1336.5, mid, "Mid price should be correct")
}

func TestGetBestBidASk_Depth(t *testing.T) {
	t.Parallel()
	_, err := getInvalidDepth().GetBestBid()
	assert.ErrorIs(t, err, ErrOrderbookInvalid, "getInvalidDepth should error correctly")

	_, err = getInvalidDepth().GetBestAsk()
	assert.ErrorIs(t, err, ErrOrderbookInvalid, "getInvalidDepth should error correctly")

	depth := NewDepth(id)
	_, err = depth.GetBestBid()
	assert.ErrorIs(t, err, errNoLiquidity, "GetBestBid should error correctly")

	_, err = depth.GetBestAsk()
	assert.ErrorIs(t, err, errNoLiquidity, "GetBestAsk should error correctly")

	err = depth.LoadSnapshot(bid, ask, 0, time.Now(), true)
	assert.NoError(t, err, "LoadSnapshot should not error")

	mid, err := depth.GetBestBid()
	assert.NoError(t, err, "GetBestBid should not error")
	assert.Equal(t, 1336.0, mid, "Mid price should be correct")

	mid, err = depth.GetBestAsk()
	assert.NoError(t, err, "GetBestAsk should not error")
	assert.Equal(t, 1337.0, mid, "Mid price should be correct")
}

func TestGetSpreadAmount(t *testing.T) {
	t.Parallel()
	_, err := getInvalidDepth().GetSpreadAmount()
	assert.ErrorIs(t, err, ErrOrderbookInvalid, "GetSpreadAmount should error correctly")

	depth := NewDepth(id)

	_, err = depth.GetSpreadAmount()
	assert.ErrorIs(t, err, errNoLiquidity, "GetSpreadAmount should error correctly")

	err = depth.LoadSnapshot(nil, ask, 0, time.Now(), true)
	assert.NoError(t, err, "LoadSnapshot should not error")

	_, err = depth.GetSpreadAmount()
	assert.ErrorIs(t, err, errNoLiquidity, "GetSpreadAmount should error correctly")

	err = depth.LoadSnapshot(bid, ask, 0, time.Now(), true)
	assert.NoError(t, err, "LoadSnapshot should not error")

	spread, err := depth.GetSpreadAmount()
	assert.NoError(t, err, "GetSpreadAmount should not error")
	assert.Equal(t, 1.0, spread, "spread should be correct")
}

func TestGetSpreadPercentage(t *testing.T) {
	t.Parallel()
	_, err := getInvalidDepth().GetSpreadPercentage()
	assert.ErrorIs(t, err, ErrOrderbookInvalid, "GetSpreadPercentage should error correctly")

	depth := NewDepth(id)

	_, err = depth.GetSpreadPercentage()
	assert.ErrorIs(t, err, errNoLiquidity, "GetSpreadPercentage should error correctly")

	err = depth.LoadSnapshot(nil, ask, 0, time.Now(), true)
	assert.NoError(t, err, "LoadSnapshot should not error")

	_, err = depth.GetSpreadPercentage()
	assert.ErrorIs(t, err, errNoLiquidity, "GetSpreadPercentage should error correctly")

	err = depth.LoadSnapshot(bid, ask, 0, time.Now(), true)
	assert.NoError(t, err, "LoadSnapshot should not error")

	spread, err := depth.GetSpreadPercentage()
	assert.NoError(t, err, "GetSpreadPercentage should not error")
	assert.Equal(t, 0.07479431563201197, spread, "spread should be correct")
}

func TestGetImbalance_Depth(t *testing.T) {
	t.Parallel()
	_, err := getInvalidDepth().GetImbalance()
	assert.ErrorIs(t, err, ErrOrderbookInvalid, "GetImbalance should error correctly")

	depth := NewDepth(id)

	_, err = depth.GetImbalance()
	assert.ErrorIs(t, err, errNoLiquidity, "GetImbalance should error correctly")

	err = depth.LoadSnapshot(nil, ask, 0, time.Now(), true)
	assert.NoError(t, err, "LoadSnapshot should not error")

	_, err = depth.GetImbalance()
	assert.ErrorIs(t, err, errNoLiquidity, "GetImbalance should error correctly")

	err = depth.LoadSnapshot(bid, ask, 0, time.Now(), true)
	assert.NoError(t, err, "LoadSnapshot should not error")

	imbalance, err := depth.GetImbalance()
	assert.NoError(t, err, "GetImbalance should not error")
	assert.Zero(t, imbalance, "imbalance should be correct")
}

func TestGetTranches(t *testing.T) {
	t.Parallel()
	_, _, err := getInvalidDepth().GetTranches(0)
	assert.ErrorIs(t, err, ErrOrderbookInvalid, "GetTranches should error correctly")

	depth := NewDepth(id)

	_, _, err = depth.GetTranches(-1)
	assert.ErrorIs(t, err, errInvalidBookDepth, "GetTranches should error correctly")

	askT, bidT, err := depth.GetTranches(0)
	assert.NoError(t, err, "GetTranches should not error")
	assert.Empty(t, askT, "Ask tranche should be empty")
	assert.Empty(t, bidT, "Bid tranche should be empty")

	err = depth.LoadSnapshot(bid, ask, 0, time.Now(), true)
	assert.NoError(t, err, "LoadSnapshot should not error")

	askT, bidT, err = depth.GetTranches(0)
	assert.NoError(t, err, "GetTranches should not error")
	assert.Len(t, askT, 20, "asks should have correct number of tranches")
	assert.Len(t, bidT, 20, "bids should have correct number of tranches")

	askT, bidT, err = depth.GetTranches(5)
	assert.NoError(t, err, "GetTranches should not error")
	assert.Len(t, askT, 5, "asks should have correct number of tranches")
	assert.Len(t, bidT, 5, "bids should have correct number of tranches")
}

func TestGetPair(t *testing.T) {
	t.Parallel()
	depth := NewDepth(id)

	_, err := depth.GetPair()
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetPair should error correctly")

	expected := currency.NewPair(currency.BTC, currency.WABI)
	depth.pair = expected

	pair, err := depth.GetPair()
	assert.NoError(t, err, "GetPair should not error")

	assert.Equal(t, expected, pair, "GetPair should return correct pair")
}

func getInvalidDepth() *Depth {
	depth := NewDepth(id)
	_ = depth.Invalidate(errors.New("invalid reasoning"))
	return depth
}

func TestMovementMethods(t *testing.T) {
	t.Parallel()

	callMethod := func(i any, name string, args []any) (*Movement, error) {
		m := reflect.ValueOf(i).MethodByName(name)
		valueArgs := []reflect.Value{}
		for _, i := range args {
			valueArgs = append(valueArgs, reflect.ValueOf(i))
		}
		r := m.Call(valueArgs)
		movement, ok := r[0].Interface().(*Movement)
		assert.True(t, ok, "Should return an Movement type")
		if err, ok := r[1].Interface().(error); ok {
			return movement, err
		}
		return movement, nil
	}

	for _, tt := range movementTests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			depth := NewDepth(id)
			methodName := strings.Split(tt.name, "_")[0]

			_, err := callMethod(getInvalidDepth(), methodName, tt.tests[0].inputs)
			assert.ErrorIsf(t, err, ErrOrderbookInvalid, "should error correctly with an invalid orderbook")

			_, err = callMethod(depth, methodName, tt.tests[0].inputs)
			assert.ErrorIs(t, err, errNoLiquidity, "should error correctly with no liquidity")

			err = depth.LoadSnapshot(bid, ask, 0, time.Now(), true)
			assert.NoError(t, err, "LoadSnapshot should not error")

			for i, subT := range tt.tests {
				move, err := callMethod(depth, methodName, subT.inputs)
				assert.NoErrorf(t, err, "sub test %d should not error", i)
				meta := reflect.Indirect(reflect.ValueOf(move))
				metaExpect := reflect.Indirect(reflect.ValueOf(subT.expect))
				for j := 0; j < metaExpect.NumField(); j++ {
					field := meta.Field(j)
					expect := metaExpect.Field(j)
					if field.CanFloat() && !expect.IsZero() {
						assert.InDeltaf(t, field.Float(), expect.Float(), accuracy10dp, "sub test %d movement %s should be correct", i, meta.Type().Field(j).Name)
					}
				}
				assert.Equal(t, subT.expect.FullBookSideConsumed, move.FullBookSideConsumed, "sub test %d movement FullBookSideConsumed should be correct", i)
			}
		})
	}
}

type movementTest struct {
	inputs []any
	expect Movement
}

var zero = accuracy10dp // Hack to allow testing of 0 values when we want without testing other fields we haven't specified

var movementTests = []struct {
	name  string
	tests []movementTest
}{
	{"HitTheBidsByImpactSlippage",
		[]movementTest{
			{[]any{0.7485029940119761, 1336.0}, Movement{Sold: 10}}, // First and second price from best bid - price level target 1326 (which should be kept)
			{[]any{1.4221556886227544, 1336.0}, Movement{Sold: 19}}, // All the way up to the last price from best bid price
		}},
	{"HitTheBidsByImpactSlippageFromMid",
		[]movementTest{
			{[]any{0.7485029940119761}, Movement{Sold: 10.0}}, // First and second price from mid - price level target 1326 (which should be kept)
			{[]any{1.4221556886227544}, Movement{Sold: 19.0}}, // All the way up to the last price from best bid price
		}},
	{"HitTheBidsByNominalSlippageFromMid",
		[]movementTest{
			{[]any{0.03741114852226}, Movement{Sold: 1.0}},  // First price from mid point
			{[]any{0.74822297044519}, Movement{Sold: 20.0}}, // All the way up to the last price from mid price
		}},
	{"HitTheBidsByNominalSlippageFromBest",
		[]movementTest{
			{[]any{0.037425149700599}, Movement{Sold: 2.0}},                             // First and second price from best bid
			{[]any{0.71107784431138}, Movement{Sold: 20.0, FullBookSideConsumed: true}}, // All the way up to the last price from best bid price
		}},
	{"LiftTheAsksByNominalSlippage",
		[]movementTest{
			{[]any{0.037397157816006, 1337.0}, Movement{Sold: 2675.0}}, // First and second price
			{[]any{0.71054599850411, 1337.0}, Movement{Sold: 26930.0}}, // All the way up to the last price
		}},
	{"LiftTheAsksByNominalSlippageFromMid",
		[]movementTest{
			{[]any{0.074822297044519}, Movement{Sold: 2675.0}}, // First price from mid point
			{[]any{0.74822297044519}, Movement{Sold: 26930.0}}, // All the way up to the last price from mid price
		}},
	{"LiftTheAsksByNominalSlippageFromBest",
		[]movementTest{
			{[]any{0.037397157816006}, Movement{Sold: 2675.0}}, // First and second price from best bid
			{[]any{0.71054599850411}, Movement{Sold: 26930.0}}, // All the way up to the last price from best bid price
		}},
	{"HitTheBidsByImpactSlippageFromBest",
		[]movementTest{
			{[]any{0.7485029940119761}, Movement{Sold: 10.0}}, // First and second price from mid - price level target 1326 (which should be kept)
			{[]any{1.4221556886227544}, Movement{Sold: 19.0}}, // All the way up to the last price from best bid price
		}},
	{"LiftTheAsksByImpactSlippage",
		[]movementTest{
			{[]any{0.7479431563201197, 1337.0}, Movement{Sold: 13415.0}}, // First and second price from best bid - price level target 1326 (which should be kept)
			{[]any{1.4210919970082274, 1337.0}, Movement{Sold: 25574.0}}, // All the way up to the last price from best bid price
		}},
	{"LiftTheAsksByImpactSlippageFromMid",
		[]movementTest{
			{[]any{0.7485029940119761}, Movement{Sold: 13415.0}}, // First and second price from mid - price level target 1326 (which should be kept)
			{[]any{1.4221556886227544}, Movement{Sold: 25574.0}}, // All the way up to the last price from best bid price
		}},
	{"LiftTheAsksByImpactSlippageFromBest",
		[]movementTest{
			{[]any{0.7479431563201197}, Movement{Sold: 13415.0}}, // First and second price from mid - price level target 1326 (which should be kept)
			// All the way up to the last price from best bid price
			// This goes to price 1356, it will not count that tranches' volume as it is needed to sustain the slippage.
			{[]any{1.4210919970082274}, Movement{Sold: 25574.0}},
		}},
	{"HitTheBidsByNominalSlippage",
		[]movementTest{
			{[]any{0.0, 1336.0}, Movement{Sold: 1.0, NominalPercentage: 0.0, StartPrice: 1336.0, EndPrice: 1336.0}},                                                          // 1st
			{[]any{0.037425149700598806, 1336.0}, Movement{Sold: 2.0, NominalPercentage: 0.037425149700598806, StartPrice: 1336.0, EndPrice: 1335.0}},                        // 2nd
			{[]any{0.02495009980039353, 1336.0}, Movement{Sold: 1.5, NominalPercentage: 0.02495009980039353, StartPrice: 1336.0, EndPrice: 1335.0}},                          // 1.5ish
			{[]any{0.7110778443113772, 1336.0}, Movement{Sold: 20, NominalPercentage: 0.7110778443113772, StartPrice: 1336.0, EndPrice: 1317.0, FullBookSideConsumed: true}}, // All
		}},
	{"HitTheBids",
		[]movementTest{
			{[]any{20.1, 1336.0, false}, Movement{Sold: 20.0, FullBookSideConsumed: true}},
			{[]any{1.0, 1336.0, false}, Movement{ImpactPercentage: 0.07485029940119761, NominalPercentage: zero, SlippageCost: zero}},
			{[]any{19.5, 1336.0, false}, Movement{NominalPercentage: 0.692845079072617, ImpactPercentage: 1.4221556886227544, SlippageCost: 180.5}},
			{[]any{20.0, 1336.0, false}, Movement{NominalPercentage: 0.7110778443113772, ImpactPercentage: FullLiquidityExhaustedPercentage, SlippageCost: 190.0, FullBookSideConsumed: true}},
		}},
	{"HitTheBids_QuotationRequired",
		[]movementTest{
			{[]any{26531.0, 1336.0, true}, Movement{Sold: 20.0, FullBookSideConsumed: true}},
			{[]any{1336.0, 1336.0, true}, Movement{ImpactPercentage: 0.07485029940119761, NominalPercentage: zero, SlippageCost: zero}},
			{[]any{25871.5, 1336.0, true}, Movement{NominalPercentage: 0.692845079072617, ImpactPercentage: 1.4221556886227544, SlippageCost: 180.5}},
			{[]any{26530.0, 1336.0, true}, Movement{NominalPercentage: 0.7110778443113772, ImpactPercentage: FullLiquidityExhaustedPercentage, SlippageCost: 190.0, FullBookSideConsumed: true}},
		}},
	{"HitTheBidsFromMid",
		[]movementTest{
			{[]any{20.1, false}, Movement{Sold: 20.0, FullBookSideConsumed: true}},
			{[]any{1.0, false}, Movement{ImpactPercentage: 0.11223344556677892, NominalPercentage: 0.03741114852225963, SlippageCost: zero}}, // mid price 1336.5 -> 1335
			{[]any{19.5, false}, Movement{NominalPercentage: 0.7299970262933156, ImpactPercentage: 1.4590347923681257, SlippageCost: 180.5}},
			{[]any{20.0, false}, Movement{NominalPercentage: 0.7482229704451926, ImpactPercentage: FullLiquidityExhaustedPercentage, SlippageCost: 190.0, FullBookSideConsumed: true}},
		}},
	{"HitTheBidsFromMid_QuotationRequired",
		[]movementTest{
			{[]any{26531.0, true}, Movement{Sold: 20.0, FullBookSideConsumed: true}},
			{[]any{1336.0, true}, Movement{ImpactPercentage: 0.11223344556677892, NominalPercentage: 0.03741114852225963, SlippageCost: zero}}, // mid price 1336.5 -> 1335
			{[]any{25871.5, true}, Movement{NominalPercentage: 0.7299970262933156, ImpactPercentage: 1.4590347923681257, SlippageCost: 180.5}},
			{[]any{26530.0, true}, Movement{NominalPercentage: 0.7482229704451926, ImpactPercentage: FullLiquidityExhaustedPercentage, SlippageCost: 190.0, FullBookSideConsumed: true}},
		}},
	{"HitTheBidsFromBest",
		[]movementTest{
			{[]any{20.1, false}, Movement{Sold: 20.0, FullBookSideConsumed: true}},
			{[]any{1.0, false}, Movement{ImpactPercentage: 0.07485029940119761, NominalPercentage: zero, SlippageCost: zero}},
			{[]any{19.5, false}, Movement{NominalPercentage: 0.692845079072617, ImpactPercentage: 1.4221556886227544, SlippageCost: 180.5}},
			{[]any{20.0, false}, Movement{NominalPercentage: 0.7110778443113772, ImpactPercentage: FullLiquidityExhaustedPercentage, SlippageCost: 190.0, FullBookSideConsumed: true}},
		}},
	{"HitTheBidsFromBest_QuotationRequired",
		[]movementTest{
			{[]any{26531.0, true}, Movement{Sold: 20.0, FullBookSideConsumed: true}},
			{[]any{1336.0, true}, Movement{ImpactPercentage: 0.07485029940119761, NominalPercentage: zero, SlippageCost: zero}},
			{[]any{25871.5, true}, Movement{NominalPercentage: 0.692845079072617, ImpactPercentage: 1.4221556886227544, SlippageCost: 180.5}},
			{[]any{26530.0, true}, Movement{NominalPercentage: 0.7110778443113772, ImpactPercentage: FullLiquidityExhaustedPercentage, SlippageCost: 190.0, FullBookSideConsumed: true}},
		}},
	{"LiftTheAsks",
		[]movementTest{
			{[]any{26931.0, 1337.0, false}, Movement{Sold: 26930.0, FullBookSideConsumed: true}},
			{[]any{1337.0, 1337.0, false}, Movement{ImpactPercentage: 0.07479431563201197, NominalPercentage: zero, SlippageCost: zero}},
			{[]any{26900.0, 1337.0, false}, Movement{NominalPercentage: 0.7097591258590459, ImpactPercentage: 1.4210919970082274, SlippageCost: 189.57964601770072}},
			{[]any{26930.0, 1336.0, false}, Movement{NominalPercentage: 0.7859281437125748, ImpactPercentage: FullLiquidityExhaustedPercentage, SlippageCost: 190.0, FullBookSideConsumed: true}},
		}},
	{"LiftTheAsks_BaseRequired",
		[]movementTest{
			{[]any{21.0, 1337.0, true}, Movement{Sold: 26930.0, FullBookSideConsumed: true}},
			{[]any{1.0, 1337.0, true}, Movement{ImpactPercentage: 0.07479431563201197, NominalPercentage: zero, SlippageCost: zero}},
			{[]any{19.97787610619469, 1337.0, true}, Movement{NominalPercentage: 0.7097591258590459, ImpactPercentage: 1.4210919970082274, SlippageCost: 189.57964601770072}},
			{[]any{20.0, 1336.0, true}, Movement{NominalPercentage: 0.7859281437125748, ImpactPercentage: FullLiquidityExhaustedPercentage, SlippageCost: 190.0, FullBookSideConsumed: true}},
		}},
	{"LiftTheAsksFromMid",
		[]movementTest{
			{[]any{26931.0, false}, Movement{Sold: 26930.0, FullBookSideConsumed: true}},
			{[]any{1337.0, false}, Movement{NominalPercentage: 0.03741114852225963, ImpactPercentage: 0.11223344556677892, SlippageCost: zero}},
			{[]any{26900.0, false}, Movement{NominalPercentage: 0.747435803422031, ImpactPercentage: 1.4590347923681257, SlippageCost: 189.57964601770072}},
			{[]any{26930.0, false}, Movement{NominalPercentage: 0.7482229704451926, ImpactPercentage: FullLiquidityExhaustedPercentage, SlippageCost: 190.0, FullBookSideConsumed: true}},
		}},
	{"LiftTheAsksFromMid_BaseRequired",
		[]movementTest{
			{[]any{21.0, true}, Movement{Sold: 26930.0, FullBookSideConsumed: true}},
			{[]any{1.0, true}, Movement{NominalPercentage: 0.03741114852225963, ImpactPercentage: 0.11223344556677892, SlippageCost: zero}},
			{[]any{19.97787610619469, true}, Movement{NominalPercentage: 0.7474358034220139, ImpactPercentage: 1.4590347923681257, SlippageCost: 189.5796460176971}},
			{[]any{20.0, true}, Movement{NominalPercentage: 0.7482229704451926, ImpactPercentage: FullLiquidityExhaustedPercentage, SlippageCost: 190.0, FullBookSideConsumed: true}},
		}},
	{"LiftTheAsksFromBest",
		[]movementTest{
			{[]any{26931.0, false}, Movement{Sold: 26930.0, FullBookSideConsumed: true}},
			{[]any{1337.0, false}, Movement{NominalPercentage: zero, ImpactPercentage: 0.07479431563201197, SlippageCost: zero}},
			{[]any{26900.0, false}, Movement{NominalPercentage: 0.7097591258590459, ImpactPercentage: 1.4210919970082274, SlippageCost: 189.579646017701}},
			{[]any{26930.0, false}, Movement{NominalPercentage: 0.7105459985041137, ImpactPercentage: FullLiquidityExhaustedPercentage, SlippageCost: 190.0, FullBookSideConsumed: true}},
		}},
	{"LiftTheAsksFromBest_BaseRequired",
		[]movementTest{
			{[]any{21.0, true}, Movement{Sold: 26930.0, FullBookSideConsumed: true}},
			{[]any{1.0, true}, Movement{NominalPercentage: zero, ImpactPercentage: 0.07479431563201197, SlippageCost: zero}},
			{[]any{19.97787610619469, true}, Movement{NominalPercentage: 0.7097591258590459, ImpactPercentage: 1.4210919970082274, SlippageCost: 189.579646017701}},
			{[]any{20.0, true}, Movement{NominalPercentage: 0.7105459985041137, ImpactPercentage: FullLiquidityExhaustedPercentage, SlippageCost: 190.0, FullBookSideConsumed: true}},
		}},
}
