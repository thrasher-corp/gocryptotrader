package orderbook

import (
	"fmt"
	"testing"
)

var ASC = func(priceTip, priceUpdate float64) bool { return priceTip > priceUpdate }
var DSC = func(priceTip, priceUpdate float64) bool { return priceTip < priceUpdate }

func TestUpdateInsertByID(t *testing.T) {
	// asks := linkedList{}
	s := Stack{}
	// asksSnapshot := Items{
	// 	{Price: 1, Amount: 1, ID: 1},
	// 	{Price: 3, Amount: 1, ID: 3},
	// 	{Price: 5, Amount: 1, ID: 5},
	// 	{Price: 7, Amount: 1, ID: 7},
	// 	{Price: 9, Amount: 1, ID: 9},
	// 	{Price: 11, Amount: 1, ID: 11},
	// }
	// asks.Load(asksSnapshot, &s)

	// // Update one instance with matching ID
	// asks.updateInsertByID(Items{
	// 	{Price: 1, Amount: 2, ID: 1},
	// }, ASC, &s)

	// Check(asks, 7, 37, 6, t)

	// // Reset
	// asks.Load(asksSnapshot, &s)

	// // Update all instances with matching ID in order
	// asks.updateInsertByID(Items{
	// 	{Price: 1, Amount: 2, ID: 1},
	// 	{Price: 3, Amount: 2, ID: 3},
	// 	{Price: 5, Amount: 2, ID: 5},
	// 	{Price: 7, Amount: 2, ID: 7},
	// 	{Price: 9, Amount: 2, ID: 9},
	// 	{Price: 11, Amount: 2, ID: 11},
	// }, ASC, &s)

	// Check(asks, 12, 72, 6, t)

	// // Update all instances with matching ID in backwards
	// asks.updateInsertByID(Items{
	// 	{Price: 11, Amount: 2, ID: 11},
	// 	{Price: 9, Amount: 2, ID: 9},
	// 	{Price: 7, Amount: 2, ID: 7},
	// 	{Price: 5, Amount: 2, ID: 5},
	// 	{Price: 3, Amount: 2, ID: 3},
	// 	{Price: 1, Amount: 2, ID: 1},
	// }, ASC, &s)

	// Check(asks, 12, 72, 6, t)

	// // Update all instances with matching ID all over the ship
	// asks.updateInsertByID(Items{
	// 	{Price: 11, Amount: 2, ID: 11},
	// 	{Price: 3, Amount: 2, ID: 3},
	// 	{Price: 7, Amount: 2, ID: 7},
	// 	{Price: 9, Amount: 2, ID: 9},
	// 	{Price: 1, Amount: 2, ID: 1},
	// 	{Price: 5, Amount: 2, ID: 5},
	// }, ASC, &s)

	// Check(asks, 12, 72, 6, t)

	// // Update all instances move one before ID in middle
	// asks.updateInsertByID(Items{
	// 	{Price: 1, Amount: 2, ID: 1},
	// 	{Price: 3, Amount: 2, ID: 3},
	// 	{Price: 2, Amount: 2, ID: 5},
	// 	{Price: 7, Amount: 2, ID: 7},
	// 	{Price: 9, Amount: 2, ID: 9},
	// 	{Price: 11, Amount: 2, ID: 11},
	// }, ASC, &s)

	// Check(asks, 12, 66, 6, t)

	// // Update all instances move one before ID at head
	// asks.updateInsertByID(Items{
	// 	{Price: 1, Amount: 2, ID: 1},
	// 	{Price: 3, Amount: 2, ID: 3},
	// 	{Price: .5, Amount: 2, ID: 5},
	// 	{Price: 7, Amount: 2, ID: 7},
	// 	{Price: 9, Amount: 2, ID: 9},
	// 	{Price: 11, Amount: 2, ID: 11},
	// }, ASC, &s)

	// Check(asks, 12, 63, 6, t)

	// // Reset
	// asks.Load(asksSnapshot, &s)

	// // Update all instances move one after ID
	// asks.updateInsertByID(Items{
	// 	{Price: 1, Amount: 2, ID: 1},
	// 	{Price: 3, Amount: 2, ID: 3},
	// 	{Price: 8, Amount: 2, ID: 5},
	// 	{Price: 7, Amount: 2, ID: 7},
	// 	{Price: 9, Amount: 2, ID: 9},
	// 	{Price: 11, Amount: 2, ID: 11},
	// }, ASC, &s)

	// Check(asks, 12, 78, 6, t)

	// // Reset
	// asks.Load(asksSnapshot, &s)

	// // Update all instances move one after ID to tail
	// asks.updateInsertByID(Items{
	// 	{Price: 1, Amount: 2, ID: 1},
	// 	{Price: 3, Amount: 2, ID: 3},
	// 	{Price: 12, Amount: 2, ID: 5},
	// 	{Price: 7, Amount: 2, ID: 7},
	// 	{Price: 9, Amount: 2, ID: 9},
	// 	{Price: 11, Amount: 2, ID: 11},
	// }, ASC, &s)

	// Check(asks, 12, 86, 6, t)

	// // Update all instances then pop new instance
	// asks.updateInsertByID(Items{
	// 	{Price: 1, Amount: 2, ID: 1},
	// 	{Price: 3, Amount: 2, ID: 3},
	// 	{Price: 12, Amount: 2, ID: 5},
	// 	{Price: 7, Amount: 2, ID: 7},
	// 	{Price: 9, Amount: 2, ID: 9},
	// 	{Price: 11, Amount: 2, ID: 11},
	// 	{Price: 10, Amount: 2, ID: 10},
	// }, ASC, &s)

	// Check(asks, 14, 106, 7, t)

	// // Reset
	// asks.Load(asksSnapshot, &s)

	// // Update all instances pop at head
	// asks.updateInsertByID(Items{
	// 	{Price: 0.5, Amount: 2, ID: 0},
	// 	{Price: 1, Amount: 2, ID: 1},
	// 	{Price: 3, Amount: 2, ID: 3},
	// 	{Price: 12, Amount: 2, ID: 5},
	// 	{Price: 7, Amount: 2, ID: 7},
	// 	{Price: 9, Amount: 2, ID: 9},
	// 	{Price: 11, Amount: 2, ID: 11},
	// }, ASC, &s)

	// Check(asks, 14, 87, 7, t)

	// Bids -------------------------------------------------------------------

	bids := linkedList{}
	bidsSnapshot := Items{
		{Price: 11, Amount: 1, ID: 11},
		{Price: 9, Amount: 1, ID: 9},
		{Price: 7, Amount: 1, ID: 7},
		{Price: 5, Amount: 1, ID: 5},
		{Price: 3, Amount: 1, ID: 3},
		{Price: 1, Amount: 1, ID: 1},
	}
	bids.Load(bidsSnapshot, &s)

	// Update one instance with matching ID
	bids.updateInsertByID(Items{
		{Price: 1, Amount: 2, ID: 1},
	}, DSC, &s)

	Check(bids, 7, 37, 6, true, t)

	// Reset
	bids.Load(bidsSnapshot, &s)

	// Update all instances with matching ID in order
	bids.updateInsertByID(Items{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 5, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	}, DSC, &s)

	Check(bids, 12, 72, 6, true, t)

	// Update all instances with matching ID in backwards
	bids.updateInsertByID(Items{
		{Price: 11, Amount: 2, ID: 11},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 5, Amount: 2, ID: 5},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 1, Amount: 2, ID: 1},
	}, DSC, &s)

	Check(bids, 12, 72, 6, true, t)

	// Update all instances with matching ID all over the ship
	bids.updateInsertByID(Items{
		{Price: 11, Amount: 2, ID: 11},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 1, Amount: 2, ID: 1},
		{Price: 5, Amount: 2, ID: 5},
	}, DSC, &s)

	Check(bids, 12, 72, 6, true, t)

	// Update all instances move one before ID in middle
	bids.updateInsertByID(Items{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 2, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	}, DSC, &s)

	Check(bids, 12, 66, 6, true, t)

	// Update all instances move one before ID at head
	bids.updateInsertByID(Items{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: .5, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	}, DSC, &s)

	Check(bids, 12, 63, 6, true, t)

	// Reset
	bids.Load(bidsSnapshot, &s)

	// Update all instances move one after ID
	bids.updateInsertByID(Items{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 8, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	}, DSC, &s)

	Check(bids, 12, 78, 6, true, t)

	// Reset
	bids.Load(bidsSnapshot, &s)

	// Update all instances move one after ID to tail
	bids.updateInsertByID(Items{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 12, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	}, DSC, &s)

	Check(bids, 12, 86, 6, true, t)

	// Update all instances then pop new instance
	bids.updateInsertByID(Items{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 12, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
		{Price: 10, Amount: 2, ID: 10},
	}, DSC, &s)

	Check(bids, 14, 106, 7, true, t)

	// Reset
	bids.Load(bidsSnapshot, &s)

	bids.Display()
	// Update all instances pop at tail
	bids.updateInsertByID(Items{
		{Price: 0.5, Amount: 2, ID: 0},
		// {Price: 1, Amount: 2, ID: 1},
		// {Price: 3, Amount: 2, ID: 3},
		// {Price: 12, Amount: 2, ID: 5},
		// {Price: 7, Amount: 2, ID: 7},
		// {Price: 9, Amount: 2, ID: 9},
		// {Price: 11, Amount: 2, ID: 11},
	}, DSC, &s)

	Check(bids, 8, 87, 7, true, t)

	bids.Display()
}

// check checks depth values after an update has taken place
func Check(ll linkedList, liquidity, value float64, nodeCount int, bid bool, t *testing.T) {
	t.Helper()
	if ll.Liquidity() != liquidity {
		ll.Display()
		t.Fatalf("mismatched liquidity expecting %v but received %v",
			liquidity,
			ll.Liquidity())
	}

	if ll.length != nodeCount {
		ll.Display()
		t.Fatalf("mismatched node count expecting %v but received %v",
			nodeCount,
			ll.length)
	}

	valueTotal := ll.Value()
	if value != value {
		ll.Display()
		t.Fatalf("mismatched total value expecting %v but received %v",
			value,
			valueTotal)
	}

	var tail *Node
	var price float64
	for tip := ll.head; ; tip = tip.next {
		if price == 0 {
			price = tip.value.Price
		} else if bid && price < tip.value.Price {
			ll.Display()
			t.Fatal("pricing out of order should be descending")
		} else if !bid && price > tip.value.Price {
			ll.Display()
			t.Fatal("pricing out of order should be ascending")
		} else {
			price = tip.value.Price
		}

		if tip.next == nil {

			tail = tip
			break
		}
	}

	var liqReversed, valReversed float64
	var nodeReversed int
	for tip := tail; tip != nil; tip = tip.prev {
		liqReversed += tip.value.Amount
		valReversed += tip.value.Amount * tip.value.Price
		nodeReversed++
	}

	if liquidity-liqReversed != 0 {
		ll.Display()
		fmt.Println(liquidity, liqReversed)
		t.Fatalf("mismatched liquidity when reversing direction expecting %v but received %v",
			0,
			liquidity-liqReversed)
	}

	if nodeCount-nodeReversed != 0 {
		ll.Display()
		t.Fatalf("mismatched node count when reversing direction expecting %v but received %v",
			0,
			nodeCount-nodeReversed)
	}

	if value-valReversed != 0 {
		ll.Display()
		fmt.Println(valReversed, value)
		t.Fatalf("mismatched total book value when reversing direction expecting %v but received %v",
			0,
			value-valReversed)
	}
}
