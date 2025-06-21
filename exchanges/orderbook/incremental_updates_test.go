package orderbook

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newSnapshot(length int) *Book {
	return &Book{
		Bids:         newBids(length),
		Asks:         newAsks(length, length),
		LastUpdated:  time.Now(),
		LastPushed:   time.Now(),
		LastUpdateID: 1,
	}
}

func newBids(length int) Levels {
	bids := make(Levels, length)
	for i := range length {
		bids[i] = Level{Price: 1337 - float64(i), Amount: 1, ID: int64(i + 1)}
	}
	return bids
}

func newAsks(idOffset, length int) Levels {
	asks := make(Levels, length)
	for i := range length {
		asks[i] = Level{Price: 1338 + float64(i), Amount: 1, ID: int64(i + 1 + idOffset)}
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
	assert.NoError(t, d.ProcessUpdate(&Update{UpdateTime: time.Now(), Asks: Levels{{Price: 1337.5, Amount: 69420, ID: 69420}}, SkipOutOfOrderLastUpdateID: true}))
	ob, err := d.Retrieve()
	require.NoError(t, err)
	assert.NotEqual(t, int64(69420), ob.Asks[0].ID, "Update above should skip insertion")
	d.options.restSnapshot = true // Simulate the snapshot has been loaded from REST
	err = d.ProcessUpdate(&Update{UpdateTime: time.Now(), Asks: Levels{{Price: 1337.5, Amount: 69420, ID: 69420}}})
	assert.ErrorIs(t, err, errRESTSnapshot)

	require.NoError(t, d.LoadSnapshot(newSnapshot(20)))
	err = d.ProcessUpdate(&Update{Asks: Levels{{Price: 1337.5, Amount: 69420, ID: 69420}}})
	assert.ErrorIs(t, err, ErrLastUpdatedNotSet)

	require.NoError(t, d.LoadSnapshot(newSnapshot(20)))
	err = d.ProcessUpdate(&Update{Action: InsertAction, Asks: Levels{{Price: 1337.5, Amount: 69420, ID: 69420}}})
	assert.ErrorIs(t, err, ErrLastUpdatedNotSet)

	require.NoError(t, d.LoadSnapshot(newSnapshot(20)))
	d.validateOrderbook = true
	d.askLevels.Levels[0].Amount = 0
	err = d.ProcessUpdate(&Update{UpdateTime: time.Now(), Asks: Levels{{Price: 1337.5, Amount: 69420, ID: 69420}}})
	assert.ErrorIs(t, err, errAmountInvalid)

	require.NoError(t, d.LoadSnapshot(newSnapshot(20)))
	err = d.ProcessUpdate(&Update{UpdateTime: time.Now(), Asks: Levels{{Price: 1337.5, Amount: 69420, ID: 69420}}})
	require.NoError(t, err)

	require.NoError(t, d.LoadSnapshot(newSnapshot(20)))
	err = d.ProcessUpdate(&Update{UpdateTime: time.Now(), Asks: Levels{{Price: 1337.5, Amount: 69420, ID: 69420}}, ExpectedChecksum: 1337})
	require.ErrorIs(t, err, errChecksumGeneratorUnset)

	require.NoError(t, d.LoadSnapshot(newSnapshot(20)))
	err = d.ProcessUpdate(&Update{UpdateTime: time.Now(), Asks: Levels{{Price: 1337.5, Amount: 69420, ID: 69420}}, ExpectedChecksum: 1337, GenerateChecksum: func(*Book) uint32 { return 1336 }})
	require.ErrorIs(t, err, errChecksumMismatch)

	require.NoError(t, d.LoadSnapshot(newSnapshot(20)))
	err = d.ProcessUpdate(&Update{UpdateTime: time.Now(), Asks: Levels{{Price: 1337.5, Amount: 69420, ID: 69420}}, ExpectedChecksum: 1337, GenerateChecksum: func(*Book) uint32 { return 1337 }})
	require.NoError(t, err)

	require.NoError(t, d.LoadSnapshot(newSnapshot(20)))
	d.askLevels.Levels[0].Amount = 0
	d.validateOrderbook = false // Disable verification
	err = d.ProcessUpdate(&Update{UpdateTime: time.Now(), Asks: Levels{{Price: 1337.5, Amount: 69420, ID: 69420}}, ExpectedChecksum: 1337, GenerateChecksum: func(*Book) uint32 { return 1337 }})
	require.NoError(t, err, "must not error when ValidateOrderbook is false")
}

func TestUpdate(t *testing.T) {
	t.Parallel()
	d := NewDepth(id)

	require.NoError(t, d.LoadSnapshot(newSnapshot(20)))
	err := d.update(&Update{})
	assert.ErrorIs(t, err, errInvalidAction, "update should error correctly")

	err = d.update(&Update{Action: UpdateAction, UpdateTime: time.Now(), Asks: Levels{{Price: 1338, Amount: 69420, ID: 69420}}})
	assert.ErrorIs(t, err, errUpdateFailed, "update should error correctly")
	assert.ErrorContains(t, err, "Update")
	err = d.update(&Update{Action: UpdateAction, UpdateTime: time.Now(), Asks: Levels{{Price: 1338, Amount: 69420, ID: 21}}})
	assert.NoError(t, err, "update should not error")
	ob, err := d.Retrieve()
	require.NoError(t, err)
	assert.Equal(t, 69420.0, ob.Asks[0].Amount, "First ask amount should be correct")

	require.NoError(t, d.LoadSnapshot(newSnapshot(20)))
	err = d.update(&Update{Action: DeleteAction, UpdateTime: time.Now(), Asks: Levels{{Price: 1338, Amount: 1, ID: 69420}}})
	assert.ErrorIs(t, err, errDeleteFailed, "update should error correctly")
	assert.ErrorContains(t, err, "Delete")
	err = d.update(&Update{Action: DeleteAction, UpdateTime: time.Now(), Asks: Levels{{ID: 21}}})
	assert.NoError(t, err, "update should not error")
	ob, err = d.Retrieve()
	require.NoError(t, err)
	assert.NotEqual(t, 21, ob.Asks[0].ID, "Ask element should be deleted")
	assert.Len(t, ob.Asks, 19, "Asks length should be correct")

	require.NoError(t, d.LoadSnapshot(newSnapshot(20)))
	err = d.update(&Update{Action: InsertAction, UpdateTime: time.Now(), Asks: Levels{{Price: 1338, Amount: 1, ID: 21}}})
	assert.ErrorIs(t, err, errUpdateFailed, "update should error correctly")
	assert.ErrorContains(t, err, "Insert")
	err = d.update(&Update{Action: InsertAction, UpdateTime: time.Now(), Asks: Levels{{Price: 1337.5, Amount: 1, ID: 69420}}})
	assert.NoError(t, err, "update should not error")
	ob, err = d.Retrieve()
	require.NoError(t, err)
	assert.Equal(t, int64(69420), ob.Asks[0].ID, "First ask ID should be correct")

	require.NoError(t, d.LoadSnapshot(newSnapshot(20)))
	err = d.update(&Update{Action: UpdateOrInsertAction, UpdateTime: time.Now(), Asks: Levels{{Price: 1338, Amount: 0, ID: 21}}})
	assert.ErrorIs(t, err, errUpdateFailed, "update should error correctly")
	assert.ErrorContains(t, err, "UpdateOrInsert")
	err = d.update(&Update{Action: UpdateOrInsertAction, UpdateTime: time.Now(), Asks: Levels{{Price: 1337.5, Amount: 1, ID: 69420}}})
	assert.NoError(t, err, "update should not error")
	ob, err = d.Retrieve()
	require.NoError(t, err)
	assert.Equal(t, int64(69420), ob.Asks[0].ID, "First ask ID should be correct")
}

func TestUpdateBidAskByID(t *testing.T) {
	t.Parallel()
	d := NewDepth(id)
	err := d.LoadSnapshot(&Book{Bids: Levels{{Price: 1337, Amount: 1, ID: 1}}, Asks: Levels{{Price: 1337, Amount: 10, ID: 2}}, LastUpdated: time.Now(), LastPushed: time.Now()})
	assert.NoError(t, err, "LoadSnapshot should not error")

	updates := &Update{
		Bids: Levels{{Price: 1337, Amount: 2, ID: 1}},
		Asks: Levels{{Price: 1337, Amount: 2, ID: 2}},
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
		Bids:       Levels{{Price: 1337, Amount: 2, ID: 666}},
		UpdateTime: time.Now(),
	}
	// random unmatching IDs
	err = d.updateBidAskByID(updates)
	assert.ErrorIs(t, err, errIDCannotBeMatched, "UpdateBidAskByID should error correctly")

	updates = &Update{
		Asks:       Levels{{Price: 1337, Amount: 2, ID: 69}},
		UpdateTime: time.Now(),
	}
	err = d.updateBidAskByID(updates)
	assert.ErrorIs(t, err, errIDCannotBeMatched, "UpdateBidAskByID should error correctly")
}

func TestDelete(t *testing.T) {
	t.Parallel()
	d := NewDepth(id)
	err := d.LoadSnapshot(&Book{Bids: Levels{{Price: 1337, Amount: 1, ID: 1}}, Asks: Levels{{Price: 1337, Amount: 10, ID: 2}}, LastUpdated: time.Now(), LastPushed: time.Now()})
	assert.NoError(t, err, "LoadSnapshot should not error")

	updates := &Update{
		Bids: Levels{{Price: 1337, Amount: 2, ID: 1}},
		Asks: Levels{{Price: 1337, Amount: 2, ID: 2}},
	}

	err = d.delete(updates, false)
	assert.ErrorIs(t, err, ErrLastUpdatedNotSet, "delete should error correctly")

	updates.UpdateTime = time.Now()
	err = d.delete(updates, false)
	assert.NoError(t, err, "delete should not error")

	ob, err := d.Retrieve()
	assert.NoError(t, err, "Retrieve should not error")
	assert.Empty(t, ob.Asks, "Asks should be empty")
	assert.Empty(t, ob.Bids, "Bids should be empty")

	updates = &Update{
		Bids:       Levels{{Price: 1337, Amount: 2, ID: 1}},
		UpdateTime: time.Now(),
	}
	err = d.delete(updates, false)
	assert.ErrorIs(t, err, errIDCannotBeMatched, "delete should error correctly")

	updates = &Update{
		Asks:       Levels{{Price: 1337, Amount: 2, ID: 2}},
		UpdateTime: time.Now(),
	}
	err = d.delete(updates, false)
	assert.ErrorIs(t, err, errIDCannotBeMatched, "delete should error correctly")

	updates = &Update{
		Asks:       Levels{{Price: 1337, Amount: 2, ID: 2}},
		UpdateTime: time.Now(),
	}
	err = d.delete(updates, true)
	assert.NoError(t, err, "delete should not error")
}

func TestInsert(t *testing.T) {
	t.Parallel()
	d := NewDepth(id)
	err := d.LoadSnapshot(&Book{Bids: Levels{{Price: 1337, Amount: 1, ID: 1}}, Asks: Levels{{Price: 1337, Amount: 10, ID: 2}}, LastUpdated: time.Now(), LastPushed: time.Now()})
	assert.NoError(t, err, "LoadSnapshot should not error")

	updates := &Update{
		Asks: Levels{{Price: 1337, Amount: 2, ID: 3}},
	}
	err = d.insert(updates)
	assert.ErrorIs(t, err, ErrLastUpdatedNotSet, "insert should error correctly")

	updates.UpdateTime = time.Now()

	err = d.insert(updates)
	assert.ErrorIs(t, err, errCollisionDetected, "insert should error correctly on collision")

	err = d.LoadSnapshot(&Book{Bids: Levels{{Price: 1337, Amount: 1, ID: 1}}, Asks: Levels{{Price: 1337, Amount: 10, ID: 2}}, LastUpdated: time.Now(), LastPushed: time.Now()})
	assert.NoError(t, err, "LoadSnapshot should not error")

	updates = &Update{
		Bids:       Levels{{Price: 1337, Amount: 2, ID: 3}},
		UpdateTime: time.Now(),
	}

	err = d.insert(updates)
	assert.ErrorIs(t, err, errCollisionDetected, "insert should error correctly on collision")

	err = d.LoadSnapshot(&Book{Bids: Levels{{Price: 1337, Amount: 1, ID: 1}}, Asks: Levels{{Price: 1337, Amount: 10, ID: 2}}, LastUpdated: time.Now(), LastPushed: time.Now()})
	assert.NoError(t, err, "LoadSnapshot should not error")

	updates = &Update{
		Bids:       Levels{{Price: 1338, Amount: 2, ID: 3}},
		Asks:       Levels{{Price: 1336, Amount: 2, ID: 4}},
		UpdateTime: time.Now(),
	}
	err = d.insert(updates)
	assert.NoError(t, err, "InsertBidAskByID should not error")

	ob, err := d.Retrieve()
	assert.NoError(t, err, "Retrieve should not error")
	assert.Len(t, ob.Asks, 2, "Should have correct Asks")
	assert.Len(t, ob.Bids, 2, "Should have correct Bids")
}

func TestUpdateOrInsert(t *testing.T) {
	t.Parallel()
	d := NewDepth(id)
	err := d.LoadSnapshot(&Book{Bids: Levels{{Price: 1337, Amount: 1, ID: 1}}, Asks: Levels{{Price: 1337, Amount: 10, ID: 2}}, LastUpdated: time.Now(), LastPushed: time.Now()})
	assert.NoError(t, err, "LoadSnapshot should not error")

	updates := &Update{
		Bids: Levels{{Price: 1338, Amount: 0, ID: 3}},
		Asks: Levels{{Price: 1336, Amount: 2, ID: 4}},
	}
	err = d.updateOrInsert(updates)
	assert.ErrorIs(t, err, ErrLastUpdatedNotSet, "updateOrInsert should error correctly")

	updates.UpdateTime = time.Now()
	err = d.updateOrInsert(updates)
	assert.ErrorIs(t, err, errAmountCannotBeLessOrEqualToZero, "updateOrInsert should error correctly")

	err = d.LoadSnapshot(&Book{Bids: Levels{{Price: 1337, Amount: 1, ID: 1}}, Asks: Levels{{Price: 1337, Amount: 10, ID: 2}}, LastUpdated: time.Now(), LastPushed: time.Now()})
	assert.NoError(t, err, "LoadSnapshot should not error")

	updates = &Update{
		Bids:       Levels{{Price: 1338, Amount: 2, ID: 3}},
		Asks:       Levels{{Price: 1336, Amount: 0, ID: 4}},
		UpdateTime: time.Now(),
	}
	err = d.updateOrInsert(updates)
	assert.ErrorIs(t, err, errAmountCannotBeLessOrEqualToZero, "updateOrInsert should error correctly")

	err = d.LoadSnapshot(&Book{Bids: Levels{{Price: 1337, Amount: 1, ID: 1}}, Asks: Levels{{Price: 1337, Amount: 10, ID: 2}}, LastUpdated: time.Now(), LastPushed: time.Now()})
	assert.NoError(t, err, "LoadSnapshot should not error")

	updates = &Update{
		Bids:       Levels{{Price: 1338, Amount: 2, ID: 3}},
		Asks:       Levels{{Price: 1336, Amount: 2, ID: 4}},
		UpdateTime: time.Now(),
	}
	err = d.updateOrInsert(updates)
	assert.NoError(t, err, "updateOrInsert should not error")

	ob, err := d.Retrieve()
	assert.NoError(t, err, "Retrieve should not error")
	assert.Len(t, ob.Asks, 2, "Should have correct Asks")
	assert.Len(t, ob.Bids, 2, "Should have correct Bids")
}

func TestUpdateBidAskByPrice(t *testing.T) {
	t.Parallel()
	d := NewDepth(id)
	err := d.LoadSnapshot(&Book{Bids: Levels{{Price: 1337, Amount: 1, ID: 1}}, Asks: Levels{{Price: 1338, Amount: 10, ID: 2}}, LastUpdated: time.Now(), LastPushed: time.Now()})
	assert.NoError(t, err, "LoadSnapshot should not error")

	err = d.updateBidAskByPrice(&Update{})
	assert.ErrorIs(t, err, ErrLastUpdatedNotSet, "UpdateBidAskByPrice should error correctly")

	err = d.updateBidAskByPrice(&Update{UpdateTime: time.Now()})
	assert.NoError(t, err, "UpdateBidAskByPrice should not error")

	updates := &Update{
		Bids:       Levels{{Price: 1337, Amount: 2, ID: 1}},
		Asks:       Levels{{Price: 1338, Amount: 3, ID: 2}},
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
		Bids:       Levels{{Price: 1337, Amount: 0, ID: 1}},
		Asks:       Levels{{Price: 1338, Amount: 0, ID: 2}},
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

func TestString(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		action   ActionType
		expected string
	}{
		{action: UpdateAction, expected: "Update"},
		{action: InsertAction, expected: "Insert"},
		{action: UpdateOrInsertAction, expected: "UpdateOrInsert"},
		{action: DeleteAction, expected: "Delete"},
		{action: UnknownAction, expected: "Unknown"},
		{action: ActionType(69), expected: "Unknown(69)"},
	} {
		t.Run(tc.expected, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, tc.action.String(), "String representation should match")
		})
	}
}
