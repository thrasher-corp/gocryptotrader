package orderbook

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newSnapshot(length int) *Base {
	return &Base{
		Bids:           newBids(length),
		Asks:           newAsks(length, length),
		LastUpdated:    time.Now(),
		UpdatePushedAt: time.Now(),
		LastUpdateID:   1,
	}
}

func newBids(length int) Tranches {
	bids := make(Tranches, length)
	for i := range length {
		bids[i] = Tranche{Price: 1337 - float64(i), Amount: 1, ID: int64(i + 1)}
	}
	return bids
}

func newAsks(idOffset, length int) Tranches {
	asks := make(Tranches, length)
	for i := range length {
		asks[i] = Tranche{Price: 1338 + float64(i), Amount: 1, ID: int64(i + 1 + idOffset)}
	}
	return asks
}

func TestProcessUpdate(t *testing.T) {
	t.Parallel()
	d := NewDepth(id)
	require.NoError(t, d.LoadSnapshot(newSnapshot(69)))
	assert.ErrorIs(t, d.ProcessUpdate(&Update{}), ErrEmptyUpdate)
	assert.ErrorIs(t, d.ProcessUpdate(&Update{AllowEmpty: true}), ErrEmptyUpdate, "exercise validation error return from last ProcessUpdate call which invalidates the orderbook")
	require.NoError(t, d.LoadSnapshot(newSnapshot(20)))
	assert.NoError(t, d.ProcessUpdate(&Update{UpdateTime: time.Now(), Asks: Tranches{{Price: 1337.5, Amount: 69420, ID: 69420}}, SkipOutOfOrderLastUpdateID: true}))
	ob, err := d.Retrieve()
	require.NoError(t, err)
	assert.NotEqual(t, int64(69420), ob.Asks[0].ID, "Update above should skip insertion")
	d.options.restSnapshot = true // Simulate the snapshot has been loaded from REST
	err = d.ProcessUpdate(&Update{UpdateTime: time.Now(), Asks: Tranches{{Price: 1337.5, Amount: 69420, ID: 69420}}})
	assert.ErrorIs(t, err, errRESTSnapshot)

	require.NoError(t, d.LoadSnapshot(newSnapshot(20)))
	err = d.ProcessUpdate(&Update{Asks: Tranches{{Price: 1337.5, Amount: 69420, ID: 69420}}})
	assert.ErrorIs(t, err, ErrLastUpdatedNotSet)

	require.NoError(t, d.LoadSnapshot(newSnapshot(20)))
	err = d.ProcessUpdate(&Update{Action: InsertAction, Asks: Tranches{{Price: 1337.5, Amount: 69420, ID: 69420}}})
	assert.ErrorIs(t, err, ErrLastUpdatedNotSet)

	require.NoError(t, d.LoadSnapshot(newSnapshot(20)))
	d.verifyOrderbook = true
	d.askTranches.Tranches[0].Amount = 0
	err = d.ProcessUpdate(&Update{UpdateTime: time.Now(), Asks: Tranches{{Price: 1337.5, Amount: 69420, ID: 69420}}})
	assert.ErrorIs(t, err, errAmountInvalid)

	require.NoError(t, d.LoadSnapshot(newSnapshot(20)))
	err = d.ProcessUpdate(&Update{UpdateTime: time.Now(), Asks: Tranches{{Price: 1337.5, Amount: 69420, ID: 69420}}, ExpectedChecksum: 1337})
	require.ErrorIs(t, err, errChecksumGeneratorUnset)

	require.NoError(t, d.LoadSnapshot(newSnapshot(20)))
	err = d.ProcessUpdate(&Update{UpdateTime: time.Now(), Asks: Tranches{{Price: 1337.5, Amount: 69420, ID: 69420}}, ExpectedChecksum: 1337, GenerateChecksum: func(*Base) uint32 { return 1336 }})
	require.ErrorIs(t, err, errChecksumMismatch)

	require.NoError(t, d.LoadSnapshot(newSnapshot(20)))
	err = d.ProcessUpdate(&Update{UpdateTime: time.Now(), Asks: Tranches{{Price: 1337.5, Amount: 69420, ID: 69420}}, ExpectedChecksum: 1337, GenerateChecksum: func(*Base) uint32 { return 1337 }})
	require.NoError(t, err)
}

func TestUpdateByIDAndAction(t *testing.T) {
	t.Parallel()
	d := NewDepth(id)

	require.NoError(t, d.LoadSnapshot(newSnapshot(20)))
	err := d.updateByIDAndAction(&Update{})
	assert.ErrorIs(t, err, errInvalidAction, "UpdateByIDAndAction should error correctly")

	err = d.updateByIDAndAction(&Update{Action: UpdateAction, UpdateTime: time.Now(), Asks: Tranches{{Price: 1338, Amount: 69420, ID: 69420}}})
	assert.ErrorIs(t, err, errAmendFailure, "UpdateByIDAndAction should error correctly")
	err = d.updateByIDAndAction(&Update{Action: UpdateAction, UpdateTime: time.Now(), Asks: Tranches{{Price: 1338, Amount: 69420, ID: 21}}})
	assert.NoError(t, err, "UpdateByIDAndAction should not error")
	ob, err := d.Retrieve()
	require.NoError(t, err)
	assert.Equal(t, 69420.0, ob.Asks[0].Amount, "First ask amount should be correct")

	require.NoError(t, d.LoadSnapshot(newSnapshot(20)))
	err = d.updateByIDAndAction(&Update{Action: DeleteAction, UpdateTime: time.Now(), Asks: Tranches{{Price: 1338, Amount: 1, ID: 69420}}})
	assert.ErrorIs(t, err, errDeleteFailure, "UpdateByIDAndAction should error correctly")
	err = d.updateByIDAndAction(&Update{Action: DeleteAction, UpdateTime: time.Now(), Asks: Tranches{{ID: 21}}})
	assert.NoError(t, err, "UpdateByIDAndAction should not error")
	ob, err = d.Retrieve()
	require.NoError(t, err)
	assert.NotEqual(t, 21, ob.Asks[0].ID, "Ask element should be deleted")
	assert.Len(t, ob.Asks, 19, "Asks length should be correct")

	require.NoError(t, d.LoadSnapshot(newSnapshot(20)))
	err = d.updateByIDAndAction(&Update{Action: InsertAction, UpdateTime: time.Now(), Asks: Tranches{{Price: 1338, Amount: 1, ID: 21}}})
	assert.ErrorIs(t, err, errInsertFailure, "UpdateByIDAndAction should error correctly")
	err = d.updateByIDAndAction(&Update{Action: InsertAction, UpdateTime: time.Now(), Asks: Tranches{{Price: 1337.5, Amount: 1, ID: 69420}}})
	assert.NoError(t, err, "UpdateByIDAndAction should not error")
	ob, err = d.Retrieve()
	require.NoError(t, err)
	assert.Equal(t, int64(69420), ob.Asks[0].ID, "First ask ID should be correct")

	require.NoError(t, d.LoadSnapshot(newSnapshot(20)))
	err = d.updateByIDAndAction(&Update{Action: UpdateOrInsertAction, UpdateTime: time.Now(), Asks: Tranches{{Price: 1338, Amount: 0, ID: 21}}})
	assert.ErrorIs(t, err, errUpdateInsertFailure, "UpdateByIDAndAction should error correctly")
	err = d.updateByIDAndAction(&Update{Action: UpdateOrInsertAction, UpdateTime: time.Now(), Asks: Tranches{{Price: 1337.5, Amount: 1, ID: 69420}}})
	assert.NoError(t, err, "UpdateByIDAndAction should not error")
	ob, err = d.Retrieve()
	require.NoError(t, err)
	assert.Equal(t, int64(69420), ob.Asks[0].ID, "First ask ID should be correct")
}

func TestUpdateBidAskByID(t *testing.T) {
	t.Parallel()
	d := NewDepth(id)
	err := d.LoadSnapshot(&Base{Bids: Tranches{{Price: 1337, Amount: 1, ID: 1}}, Asks: Tranches{{Price: 1337, Amount: 10, ID: 2}}, LastUpdated: time.Now(), UpdatePushedAt: time.Now()})
	assert.NoError(t, err, "LoadSnapshot should not error")

	updates := &Update{
		Bids: Tranches{{Price: 1337, Amount: 2, ID: 1}},
		Asks: Tranches{{Price: 1337, Amount: 2, ID: 2}},
	}

	err = d.updateBidAskByID(updates)
	assert.ErrorIs(t, err, ErrLastUpdatedNotSet, "UpdateBidAskByID should error correctly")

	updates.UpdateTime = time.Now()
	err = d.updateBidAskByID(updates)
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
	err = d.updateBidAskByID(updates)
	assert.ErrorIs(t, err, errIDCannotBeMatched, "UpdateBidAskByID should error correctly")

	updates = &Update{
		Asks:       Tranches{{Price: 1337, Amount: 2, ID: 69}},
		UpdateTime: time.Now(),
	}
	err = d.updateBidAskByID(updates)
	assert.ErrorIs(t, err, errIDCannotBeMatched, "UpdateBidAskByID should error correctly")
}

func TestDeleteBidAskByID(t *testing.T) {
	t.Parallel()
	d := NewDepth(id)
	err := d.LoadSnapshot(&Base{Bids: Tranches{{Price: 1337, Amount: 1, ID: 1}}, Asks: Tranches{{Price: 1337, Amount: 10, ID: 2}}, LastUpdated: time.Now(), UpdatePushedAt: time.Now()})
	assert.NoError(t, err, "LoadSnapshot should not error")

	updates := &Update{
		Bids: Tranches{{Price: 1337, Amount: 2, ID: 1}},
		Asks: Tranches{{Price: 1337, Amount: 2, ID: 2}},
	}

	err = d.deleteBidAskByID(updates, false)
	assert.ErrorIs(t, err, ErrLastUpdatedNotSet, "DeleteBidAskByID should error correctly")

	updates.UpdateTime = time.Now()
	err = d.deleteBidAskByID(updates, false)
	assert.NoError(t, err, "DeleteBidAskByID should not error")

	ob, err := d.Retrieve()
	assert.NoError(t, err, "Retrieve should not error")
	assert.Empty(t, ob.Asks, "Asks should be empty")
	assert.Empty(t, ob.Bids, "Bids should be empty")

	updates = &Update{
		Bids:       Tranches{{Price: 1337, Amount: 2, ID: 1}},
		UpdateTime: time.Now(),
	}
	err = d.deleteBidAskByID(updates, false)
	assert.ErrorIs(t, err, errIDCannotBeMatched, "DeleteBidAskByID should error correctly")

	updates = &Update{
		Asks:       Tranches{{Price: 1337, Amount: 2, ID: 2}},
		UpdateTime: time.Now(),
	}
	err = d.deleteBidAskByID(updates, false)
	assert.ErrorIs(t, err, errIDCannotBeMatched, "DeleteBidAskByID should error correctly")

	updates = &Update{
		Asks:       Tranches{{Price: 1337, Amount: 2, ID: 2}},
		UpdateTime: time.Now(),
	}
	err = d.deleteBidAskByID(updates, true)
	assert.NoError(t, err, "DeleteBidAskByID should not error")
}

func TestInsertBidAskByID(t *testing.T) {
	t.Parallel()
	d := NewDepth(id)
	err := d.LoadSnapshot(&Base{Bids: Tranches{{Price: 1337, Amount: 1, ID: 1}}, Asks: Tranches{{Price: 1337, Amount: 10, ID: 2}}, LastUpdated: time.Now(), UpdatePushedAt: time.Now()})
	assert.NoError(t, err, "LoadSnapshot should not error")

	updates := &Update{
		Asks: Tranches{{Price: 1337, Amount: 2, ID: 3}},
	}
	err = d.insertBidAskByID(updates)
	assert.ErrorIs(t, err, ErrLastUpdatedNotSet, "InsertBidAskByID should error correctly")

	updates.UpdateTime = time.Now()

	err = d.insertBidAskByID(updates)
	assert.ErrorIs(t, err, errCollisionDetected, "InsertBidAskByID should error correctly on collision")

	err = d.LoadSnapshot(&Base{Bids: Tranches{{Price: 1337, Amount: 1, ID: 1}}, Asks: Tranches{{Price: 1337, Amount: 10, ID: 2}}, LastUpdated: time.Now(), UpdatePushedAt: time.Now()})
	assert.NoError(t, err, "LoadSnapshot should not error")

	updates = &Update{
		Bids:       Tranches{{Price: 1337, Amount: 2, ID: 3}},
		UpdateTime: time.Now(),
	}

	err = d.insertBidAskByID(updates)
	assert.ErrorIs(t, err, errCollisionDetected, "InsertBidAskByID should error correctly on collision")

	err = d.LoadSnapshot(&Base{Bids: Tranches{{Price: 1337, Amount: 1, ID: 1}}, Asks: Tranches{{Price: 1337, Amount: 10, ID: 2}}, LastUpdated: time.Now(), UpdatePushedAt: time.Now()})
	assert.NoError(t, err, "LoadSnapshot should not error")

	updates = &Update{
		Bids:       Tranches{{Price: 1338, Amount: 2, ID: 3}},
		Asks:       Tranches{{Price: 1336, Amount: 2, ID: 4}},
		UpdateTime: time.Now(),
	}
	err = d.insertBidAskByID(updates)
	assert.NoError(t, err, "InsertBidAskByID should not error")

	ob, err := d.Retrieve()
	assert.NoError(t, err, "Retrieve should not error")
	assert.Len(t, ob.Asks, 2, "Should have correct Asks")
	assert.Len(t, ob.Bids, 2, "Should have correct Bids")
}

func TestUpdateInsertByID(t *testing.T) {
	t.Parallel()
	d := NewDepth(id)
	err := d.LoadSnapshot(&Base{Bids: Tranches{{Price: 1337, Amount: 1, ID: 1}}, Asks: Tranches{{Price: 1337, Amount: 10, ID: 2}}, LastUpdated: time.Now(), UpdatePushedAt: time.Now()})
	assert.NoError(t, err, "LoadSnapshot should not error")

	updates := &Update{
		Bids: Tranches{{Price: 1338, Amount: 0, ID: 3}},
		Asks: Tranches{{Price: 1336, Amount: 2, ID: 4}},
	}
	err = d.updateInsertByID(updates)
	assert.ErrorIs(t, err, ErrLastUpdatedNotSet, "UpdateInsertByID should error correctly")

	updates.UpdateTime = time.Now()
	err = d.updateInsertByID(updates)
	assert.ErrorIs(t, err, errAmountCannotBeLessOrEqualToZero, "UpdateInsertByID should error correctly")

	err = d.LoadSnapshot(&Base{Bids: Tranches{{Price: 1337, Amount: 1, ID: 1}}, Asks: Tranches{{Price: 1337, Amount: 10, ID: 2}}, LastUpdated: time.Now(), UpdatePushedAt: time.Now()})
	assert.NoError(t, err, "LoadSnapshot should not error")

	updates = &Update{
		Bids:       Tranches{{Price: 1338, Amount: 2, ID: 3}},
		Asks:       Tranches{{Price: 1336, Amount: 0, ID: 4}},
		UpdateTime: time.Now(),
	}
	err = d.updateInsertByID(updates)
	assert.ErrorIs(t, err, errAmountCannotBeLessOrEqualToZero, "UpdateInsertByID should error correctly")

	err = d.LoadSnapshot(&Base{Bids: Tranches{{Price: 1337, Amount: 1, ID: 1}}, Asks: Tranches{{Price: 1337, Amount: 10, ID: 2}}, LastUpdated: time.Now(), UpdatePushedAt: time.Now()})
	assert.NoError(t, err, "LoadSnapshot should not error")

	updates = &Update{
		Bids:       Tranches{{Price: 1338, Amount: 2, ID: 3}},
		Asks:       Tranches{{Price: 1336, Amount: 2, ID: 4}},
		UpdateTime: time.Now(),
	}
	err = d.updateInsertByID(updates)
	assert.NoError(t, err, "UpdateInsertByID should not error")

	ob, err := d.Retrieve()
	assert.NoError(t, err, "Retrieve should not error")
	assert.Len(t, ob.Asks, 2, "Should have correct Asks")
	assert.Len(t, ob.Bids, 2, "Should have correct Bids")
}

func TestUpdateBidAskByPrice(t *testing.T) {
	t.Parallel()
	d := NewDepth(id)
	err := d.LoadSnapshot(&Base{Bids: Tranches{{Price: 1337, Amount: 1, ID: 1}}, Asks: Tranches{{Price: 1338, Amount: 10, ID: 2}}, LastUpdated: time.Now(), UpdatePushedAt: time.Now()})
	assert.NoError(t, err, "LoadSnapshot should not error")

	err = d.updateBidAskByPrice(&Update{})
	assert.ErrorIs(t, err, ErrLastUpdatedNotSet, "UpdateBidAskByPrice should error correctly")

	err = d.updateBidAskByPrice(&Update{UpdateTime: time.Now()})
	assert.NoError(t, err, "UpdateBidAskByPrice should not error")

	updates := &Update{
		Bids:       Tranches{{Price: 1337, Amount: 2, ID: 1}},
		Asks:       Tranches{{Price: 1338, Amount: 3, ID: 2}},
		UpdateID:   1,
		UpdateTime: time.Now(),
	}
	err = d.updateBidAskByPrice(updates)
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
	err = d.updateBidAskByPrice(updates)
	assert.NoError(t, err, "UpdateBidAskByPrice should not error")

	askLen, err := d.GetAskLength()
	assert.NoError(t, err, "GetAskLength should not error")
	assert.Zero(t, askLen, "Ask Length should be correct")

	bidLen, err := d.GetBidLength()
	assert.NoError(t, err, "GetBidLength should not error")
	assert.Zero(t, bidLen, "Bid Length should be correct")
}
