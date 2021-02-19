package orderbook

import (
	"fmt"
	"testing"
)

var bid = Items{
	Item{Price: 1336, Amount: 1},
	Item{Price: 1335, Amount: 1},
	Item{Price: 1334, Amount: 1},
	Item{Price: 1333, Amount: 1},
	Item{Price: 1332, Amount: 1},
	Item{Price: 1331, Amount: 1},
	Item{Price: 1330, Amount: 1},
	Item{Price: 1329, Amount: 1},
	Item{Price: 1328, Amount: 1},
	Item{Price: 1327, Amount: 1},
	Item{Price: 1326, Amount: 1},
	Item{Price: 1325, Amount: 1},
	Item{Price: 1324, Amount: 1},
	Item{Price: 1323, Amount: 1},
	Item{Price: 1322, Amount: 1},
	Item{Price: 1321, Amount: 1},
	Item{Price: 1320, Amount: 1},
	Item{Price: 1319, Amount: 1},
	Item{Price: 1318, Amount: 1},
	Item{Price: 1317, Amount: 1},
}

var ask = Items{
	Item{Price: 1337, Amount: 1},
	Item{Price: 1338, Amount: 1},
	Item{Price: 1339, Amount: 1},
	Item{Price: 1340, Amount: 1},
	Item{Price: 1341, Amount: 1},
	Item{Price: 1342, Amount: 1},
	Item{Price: 1343, Amount: 1},
	Item{Price: 1344, Amount: 1},
	Item{Price: 1345, Amount: 1},
	Item{Price: 1346, Amount: 1},
	Item{Price: 1347, Amount: 1},
	Item{Price: 1348, Amount: 1},
	Item{Price: 1349, Amount: 1},
	Item{Price: 1350, Amount: 1},
	Item{Price: 1351, Amount: 1},
	Item{Price: 1352, Amount: 1},
	Item{Price: 1353, Amount: 1},
	Item{Price: 1354, Amount: 1},
	Item{Price: 1355, Amount: 1},
	Item{Price: 1356, Amount: 1},
}

func TestLoad(t *testing.T) {
	list := linkedList{}
	Check(list, 0, 0, 0, false, t)

	stack := &Stack{}
	list.Load(Items{
		{Price: 1, Amount: 1},
		{Price: 3, Amount: 1},
		{Price: 5, Amount: 1},
		{Price: 7, Amount: 1},
		{Price: 9, Amount: 1},
		{Price: 11, Amount: 1},
	}, stack)

	if stack.count != 0 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 0, stack.count)
	}

	Check(list, 6, 36, 6, false, t)

	list.Load(Items{
		{Price: 1, Amount: 1},
		{Price: 3, Amount: 1},
		{Price: 5, Amount: 1},
	}, stack)

	if stack.count != 3 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 3, stack.count)
	}

	Check(list, 3, 9, 3, false, t)

	list.Load(Items{
		{Price: 1, Amount: 1},
		{Price: 3, Amount: 1},
		{Price: 5, Amount: 1},
		{Price: 7, Amount: 1},
	}, stack)

	if stack.count != 2 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 3, stack.count)
	}

	Check(list, 4, 16, 4, false, t)
}

func TestUpdateInsertByPrice(t *testing.T) {
	asks := linkedList{}
	stack := Stack{}
	asksSnapshot := Items{
		{Price: 1, Amount: 1},
		{Price: 3, Amount: 1},
		{Price: 5, Amount: 1},
		{Price: 7, Amount: 1},
		{Price: 9, Amount: 1},
		{Price: 11, Amount: 1},
	}
	asks.Load(asksSnapshot, &stack)

	// Update one instance with matching price
	asks.updateInsertAsksByPrice(Items{
		{Price: 1, Amount: 2},
	}, &stack, 0)

	Check(asks, 7, 37, 6, false, t)

	if stack.count != 0 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 0, stack.count)
	}

	// Insert at head
	asks.updateInsertAsksByPrice(Items{
		{Price: 0.5, Amount: 2},
	}, &stack, 0)

	Check(asks, 9, 38, 7, false, t)

	if stack.count != 0 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 0, stack.count)
	}

	// Insert at tail
	asks.updateInsertAsksByPrice(Items{
		{Price: 12, Amount: 2},
	}, &stack, 0)

	Check(asks, 11, 62, 8, false, t)

	if stack.count != 0 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 0, stack.count)
	}

	// Insert between price and up to and beyond max allowable depth level
	asks.updateInsertAsksByPrice(Items{
		{Price: 11.5, Amount: 2},
		{Price: 10.5, Amount: 2},
		{Price: 13, Amount: 2},
	}, &stack, 10)

	Check(asks, 15, 106, 10, false, t)

	if stack.count != 1 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 0, stack.count)
	}

	// delete at tail
	asks.updateInsertAsksByPrice(Items{
		{Price: 12, Amount: 0},
	}, &stack, 0)

	Check(asks, 13, 82, 9, false, t)

	if stack.count != 2 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 0, stack.count)
	}

	bids := linkedList{}
	bidsSnapshot := Items{
		{Price: 11, Amount: 1},
		{Price: 9, Amount: 1},
		{Price: 7, Amount: 1},
		{Price: 5, Amount: 1},
		{Price: 3, Amount: 1},
		{Price: 1, Amount: 1},
	}
	bids.Load(bidsSnapshot, &stack)

	// Update one instance with matching price
	bids.updateInsertBidsByPrice(Items{
		{Price: 11, Amount: 2},
	}, &stack, 0)

	Check(bids, 7, 47, 6, true, t)

	if stack.count != 0 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 0, stack.count)
	}

	// Insert at head
	bids.updateInsertBidsByPrice(Items{
		{Price: 12, Amount: 2},
	}, &stack, 0)

	Check(bids, 9, 71, 7, true, t)

	if stack.count != 0 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 0, stack.count)
	}

	// Insert at tail
	bids.updateInsertBidsByPrice(Items{
		{Price: 0.5, Amount: 2},
	}, &stack, 0)

	Check(bids, 11, 72, 8, true, t)

	if stack.count != 0 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 0, stack.count)
	}

	// Insert between price and up to and beyond max allowable depth level
	bids.updateInsertBidsByPrice(Items{
		{Price: 11.5, Amount: 2},
		{Price: 10.5, Amount: 2},
		{Price: 13, Amount: 2},
	}, &stack, 10)

	Check(bids, 15, 141, 10, true, t)

	if stack.count != 1 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 0, stack.count)
	}

	// Insert between price and up to and beyond max allowable depth level
	bids.updateInsertBidsByPrice(Items{
		{Price: 1, Amount: 0},
	}, &stack, 0)

	Check(bids, 14, 140, 9, true, t)

	if stack.count != 2 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 0, stack.count)
	}
}

func TestUpdateInsertByID(t *testing.T) {
	asks := linkedList{}
	s := Stack{}
	asksSnapshot := Items{
		{Price: 1, Amount: 1, ID: 1},
		{Price: 3, Amount: 1, ID: 3},
		{Price: 5, Amount: 1, ID: 5},
		{Price: 7, Amount: 1, ID: 7},
		{Price: 9, Amount: 1, ID: 9},
		{Price: 11, Amount: 1, ID: 11},
	}
	asks.Load(asksSnapshot, &s)

	// Update one instance with matching ID
	asks.updateInsertByIDAsk(Items{
		{Price: 1, Amount: 2, ID: 1},
	}, &s)

	Check(asks, 7, 37, 6, false, t)

	// Reset
	asks.Load(asksSnapshot, &s)

	// Update all instances with matching ID in order
	asks.updateInsertByIDAsk(Items{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 5, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	}, &s)

	Check(asks, 12, 72, 6, false, t)

	// Update all instances with matching ID in backwards
	asks.updateInsertByIDAsk(Items{
		{Price: 11, Amount: 2, ID: 11},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 5, Amount: 2, ID: 5},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 1, Amount: 2, ID: 1},
	}, &s)

	Check(asks, 12, 72, 6, false, t)

	// Update all instances with matching ID all over the ship
	asks.updateInsertByIDAsk(Items{
		{Price: 11, Amount: 2, ID: 11},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 1, Amount: 2, ID: 1},
		{Price: 5, Amount: 2, ID: 5},
	}, &s)

	Check(asks, 12, 72, 6, false, t)

	// Update all instances move one before ID in middle
	asks.updateInsertByIDAsk(Items{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 2, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	}, &s)

	Check(asks, 12, 66, 6, false, t)

	// Update all instances move one before ID at head
	asks.updateInsertByIDAsk(Items{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: .5, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	}, &s)

	Check(asks, 12, 63, 6, false, t)

	// Reset
	asks.Load(asksSnapshot, &s)

	// Update all instances move one after ID
	asks.updateInsertByIDAsk(Items{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 8, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	}, &s)

	Check(asks, 12, 78, 6, false, t)

	// Reset
	asks.Load(asksSnapshot, &s)

	// Update all instances move one after ID to tail
	asks.updateInsertByIDAsk(Items{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 12, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	}, &s)

	Check(asks, 12, 86, 6, false, t)

	// Update all instances then pop new instance
	asks.updateInsertByIDAsk(Items{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 12, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
		{Price: 10, Amount: 2, ID: 10},
	}, &s)

	Check(asks, 14, 106, 7, false, t)

	// Reset
	asks.Load(asksSnapshot, &s)

	// Update all instances pop at head
	asks.updateInsertByIDAsk(Items{
		{Price: 0.5, Amount: 2, ID: 0},
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 12, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	}, &s)

	Check(asks, 14, 87, 7, false, t)

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
	bids.updateInsertByIDBid(Items{
		{Price: 1, Amount: 2, ID: 1},
	}, &s)

	Check(bids, 7, 37, 6, true, t)

	// Reset
	bids.Load(bidsSnapshot, &s)

	// Update all instances with matching ID in order
	bids.updateInsertByIDBid(Items{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 5, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	}, &s)

	Check(bids, 12, 72, 6, true, t)

	// Update all instances with matching ID in backwards
	bids.updateInsertByIDBid(Items{
		{Price: 11, Amount: 2, ID: 11},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 5, Amount: 2, ID: 5},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 1, Amount: 2, ID: 1},
	}, &s)

	Check(bids, 12, 72, 6, true, t)

	// Update all instances with matching ID all over the ship
	bids.updateInsertByIDBid(Items{
		{Price: 11, Amount: 2, ID: 11},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 1, Amount: 2, ID: 1},
		{Price: 5, Amount: 2, ID: 5},
	}, &s)

	Check(bids, 12, 72, 6, true, t)

	// Update all instances move one before ID in middle
	bids.updateInsertByIDBid(Items{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 2, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	}, &s)

	Check(bids, 12, 66, 6, true, t)

	// Update all instances move one before ID at head
	bids.updateInsertByIDBid(Items{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: .5, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	}, &s)

	Check(bids, 12, 63, 6, true, t)

	// Reset
	bids.Load(bidsSnapshot, &s)

	// Update all instances move one after ID
	bids.updateInsertByIDBid(Items{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 8, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	}, &s)

	Check(bids, 12, 78, 6, true, t)

	// Reset
	bids.Load(bidsSnapshot, &s)

	// Update all instances move one after ID to tail
	bids.updateInsertByIDBid(Items{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 12, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	}, &s)

	Check(bids, 12, 86, 6, true, t)

	// Update all instances then pop new instance
	bids.updateInsertByIDBid(Items{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 12, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
		{Price: 10, Amount: 2, ID: 10},
	}, &s)

	Check(bids, 14, 106, 7, true, t)

	// Reset
	bids.Load(bidsSnapshot, &s)

	bids.Display()
	// Update all instances pop at tail
	bids.updateInsertByIDBid(Items{
		{Price: 0.5, Amount: 2, ID: 0},
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 12, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	}, &s)

	Check(bids, 14, 87, 7, true, t)

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
	if valueTotal != value {
		ll.Display()
		t.Fatalf("mismatched total value expecting %v but received %v",
			value,
			valueTotal)
	}

	if ll.head == nil {
		return
	}

	var tail *Node
	var price float64
	for tip := ll.head; ; tip = tip.next {
		if price == 0 {
			price = tip.value.Price
		} else if bid && price < tip.value.Price {
			ll.Display()
			t.Fatal("Bid pricing out of order should be descending")
		} else if !bid && price > tip.value.Price {
			ll.Display()
			t.Fatal("Ask pricing out of order should be ascending")
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

//  158	   9,521,717 ns/op	 9600104 B/op	  100001 allocs/op
func BenchmarkWithoutStack(b *testing.B) {
	var n *Node
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 100000; j++ {
			n = new(Node)
			n.value.Price = 1337
		}
	}
}

//  949	   1,427,820 ns/op	       0 B/op	       0 allocs/op
func BenchmarkWithStack(b *testing.B) {
	var n *Node
	stack := NewStack()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 100000; j++ {
			n = stack.Pop()
			n.value.Price = 1337
			stack.Push(n)
		}
	}
}
