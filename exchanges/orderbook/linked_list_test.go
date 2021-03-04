package orderbook

import (
	"errors"
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

// Display displays depth content for tests
func (ll *linkedList) display() {
	for tip := ll.head; tip != nil; tip = tip.next {
		fmt.Printf("NODE: %+v %p \n", tip, tip)
	}
	fmt.Println()
}

func TestLoad(t *testing.T) {
	list := asks{}
	Check(list, 0, 0, 0, t)

	stack := &Stack{}
	list.load(Items{
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

	Check(list, 6, 36, 6, t)

	list.load(Items{
		{Price: 1, Amount: 1},
		{Price: 3, Amount: 1},
		{Price: 5, Amount: 1},
	}, stack)

	if stack.count != 3 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 3, stack.count)
	}

	Check(list, 3, 9, 3, t)

	list.load(Items{
		{Price: 1, Amount: 1},
		{Price: 3, Amount: 1},
		{Price: 5, Amount: 1},
		{Price: 7, Amount: 1},
	}, stack)

	if stack.count != 2 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 2, stack.count)
	}

	Check(list, 4, 16, 4, t)

	// purge entire list
	list.load(nil, stack)

	if stack.count != 6 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 6, stack.count)
	}

	Check(list, 0, 0, 0, t)
}

// 22222386	        57.3 ns/op	       0 B/op	       0 allocs/op (old)
// 27906781	        42.4 ns/op	       0 B/op	       0 allocs/op (new)
func BenchmarkLoad(b *testing.B) {
	ll := linkedList{}
	s := Stack{}
	for i := 0; i < b.N; i++ {
		ll.load(ask, &s)
	}
}

func TestUpdateInsertByPrice(t *testing.T) {
	a := asks{}
	stack := Stack{}
	asksSnapshot := Items{
		{Price: 1, Amount: 1},
		{Price: 3, Amount: 1},
		{Price: 5, Amount: 1},
		{Price: 7, Amount: 1},
		{Price: 9, Amount: 1},
		{Price: 11, Amount: 1},
	}
	a.load(asksSnapshot, &stack)

	// Update one instance with matching price
	a.updateInsertByPrice(Items{
		{Price: 1, Amount: 2},
	}, &stack, 0)

	Check(a, 7, 37, 6, t)

	if stack.count != 0 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 0, stack.count)
	}

	// Insert at head
	a.updateInsertByPrice(Items{
		{Price: 0.5, Amount: 2},
	}, &stack, 0)

	Check(a, 9, 38, 7, t)

	if stack.count != 0 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 0, stack.count)
	}

	// Insert at tail
	a.updateInsertByPrice(Items{
		{Price: 12, Amount: 2},
	}, &stack, 0)

	Check(a, 11, 62, 8, t)

	if stack.count != 0 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 0, stack.count)
	}

	// Insert between price and up to and beyond max allowable depth level
	a.updateInsertByPrice(Items{
		{Price: 11.5, Amount: 2},
		{Price: 10.5, Amount: 2},
		{Price: 13, Amount: 2},
	}, &stack, 10)

	Check(a, 15, 106, 10, t)

	if stack.count != 1 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 1, stack.count)
	}

	// delete at tail
	a.updateInsertByPrice(Items{
		{Price: 12, Amount: 0},
	}, &stack, 0)

	Check(a, 13, 82, 9, t)

	if stack.count != 2 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 2, stack.count)
	}

	// delete at mid
	a.updateInsertByPrice(Items{
		{Price: 7, Amount: 0},
	}, &stack, 0)

	Check(a, 12, 75, 8, t)

	if stack.count != 3 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 3, stack.count)
	}

	// delete at head
	a.updateInsertByPrice(Items{
		{Price: 0.5, Amount: 0},
	}, &stack, 0)

	Check(a, 10, 74, 7, t)

	if stack.count != 4 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 4, stack.count)
	}

	b := bids{}
	bidsSnapshot := Items{
		{Price: 11, Amount: 1},
		{Price: 9, Amount: 1},
		{Price: 7, Amount: 1},
		{Price: 5, Amount: 1},
		{Price: 3, Amount: 1},
		{Price: 1, Amount: 1},
	}
	b.load(bidsSnapshot, &stack)

	// Update one instance with matching price
	b.updateInsertByPrice(Items{
		{Price: 11, Amount: 2},
	}, &stack, 0)

	Check(b, 7, 47, 6, t)

	if stack.count != 0 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 0, stack.count)
	}

	// Insert at head
	b.updateInsertByPrice(Items{
		{Price: 12, Amount: 2},
	}, &stack, 0)

	Check(b, 9, 71, 7, t)

	if stack.count != 0 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 0, stack.count)
	}

	// Insert at tail
	b.updateInsertByPrice(Items{
		{Price: 0.5, Amount: 2},
	}, &stack, 0)

	Check(b, 11, 72, 8, t)

	if stack.count != 0 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 0, stack.count)
	}

	// Insert between price and up to and beyond max allowable depth level
	b.updateInsertByPrice(Items{
		{Price: 11.5, Amount: 2},
		{Price: 10.5, Amount: 2},
		{Price: 13, Amount: 2},
	}, &stack, 10)

	Check(b, 15, 141, 10, t)

	if stack.count != 1 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 0, stack.count)
	}

	// Insert between price and up to and beyond max allowable depth level
	b.updateInsertByPrice(Items{
		{Price: 1, Amount: 0},
	}, &stack, 0)

	Check(b, 14, 140, 9, t)

	if stack.count != 2 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 2, stack.count)
	}

	// delete at mid
	b.updateInsertByPrice(Items{
		{Price: 10.5, Amount: 0},
	}, &stack, 0)

	Check(b, 12, 119, 8, t)

	if stack.count != 3 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 3, stack.count)
	}

	// delete at head
	b.updateInsertByPrice(Items{
		{Price: 13, Amount: 0},
	}, &stack, 0)

	Check(b, 10, 93, 7, t)

	if stack.count != 4 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 4, stack.count)
	}
}

// 46154023	        24.0 ns/op	       0 B/op	       0 allocs/op (old)
// 134830672	         9.83 ns/op	       0 B/op	       0 allocs/op (new)
func BenchmarkUpdateInsertByPrice_Amend(b *testing.B) {
	a := asks{}
	stack := Stack{}

	a.load(ask, &stack)

	updates := Items{
		{
			Price:  1337, // Amend
			Amount: 2,
		},
		{
			Price:  1337, // Amend
			Amount: 1,
		},
	}

	for i := 0; i < b.N; i++ {
		a.updateInsertByPrice(updates, &stack, 0)
	}
}

// 49763002	        24.9 ns/op	       0 B/op	       0 allocs/op
func BenchmarkUpdateInsertByPrice_Insert_Delete(b *testing.B) {
	a := asks{}
	stack := Stack{}

	a.load(ask, &stack)

	updates := Items{
		{
			Price:  1337.5, // Insert
			Amount: 2,
		},
		{
			Price:  1337.5, // Delete
			Amount: 0,
		},
	}

	for i := 0; i < b.N; i++ {
		a.updateInsertByPrice(updates, &stack, 0)
	}
}

func TestUpdateByID(t *testing.T) {
	a := asks{}
	s := Stack{}
	asksSnapshot := Items{
		{Price: 1, Amount: 1, ID: 1},
		{Price: 3, Amount: 1, ID: 3},
		{Price: 5, Amount: 1, ID: 5},
		{Price: 7, Amount: 1, ID: 7},
		{Price: 9, Amount: 1, ID: 9},
		{Price: 11, Amount: 1, ID: 11},
	}
	a.load(asksSnapshot, &s)

	err := a.updateByID(Items{
		{Price: 1, Amount: 1, ID: 1},
		{Price: 3, Amount: 1, ID: 3},
		{Price: 5, Amount: 1, ID: 5},
		{Price: 7, Amount: 1, ID: 7},
		{Price: 9, Amount: 1, ID: 9},
		{Price: 11, Amount: 1, ID: 11},
	})
	if err != nil {
		t.Fatal(err)
	}

	Check(a, 6, 36, 6, t)

	err = a.updateByID(Items{
		{Price: 11, Amount: 1, ID: 1337},
	})
	if !errors.Is(err, errIDCannotBeMatched) {
		t.Fatalf("expecting %s but received %v", errIDCannotBeMatched, err)
	}
}

// 46043871	        25.9 ns/op	       0 B/op	       0 allocs/op
func BenchmarkUpdateByID(b *testing.B) {
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
	asks.load(asksSnapshot, &s)

	for i := 0; i < b.N; i++ {
		err := asks.updateByID(Items{
			{Price: 1, Amount: 1, ID: 1},
			{Price: 3, Amount: 1, ID: 3},
			{Price: 5, Amount: 1, ID: 5},
			{Price: 7, Amount: 1, ID: 7},
			{Price: 9, Amount: 1, ID: 9},
			{Price: 11, Amount: 1, ID: 11},
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestDeleteByID(t *testing.T) {
	a := asks{}
	s := Stack{}
	asksSnapshot := Items{
		{Price: 1, Amount: 1, ID: 1},
		{Price: 3, Amount: 1, ID: 3},
		{Price: 5, Amount: 1, ID: 5},
		{Price: 7, Amount: 1, ID: 7},
		{Price: 9, Amount: 1, ID: 9},
		{Price: 11, Amount: 1, ID: 11},
	}
	a.load(asksSnapshot, &s)

	// Delete at head
	err := a.deleteByID(Items{
		{Price: 1, Amount: 1, ID: 1},
	}, &s, false)
	if err != nil {
		t.Fatal(err)
	}

	Check(a, 5, 35, 5, t)

	// Delete at tail
	err = a.deleteByID(Items{
		{Price: 1, Amount: 1, ID: 11},
	}, &s, false)
	if err != nil {
		t.Fatal(err)
	}

	Check(a, 4, 24, 4, t)

	// Delete in middle
	err = a.deleteByID(Items{
		{Price: 1, Amount: 1, ID: 5},
	}, &s, false)
	if err != nil {
		t.Fatal(err)
	}

	Check(a, 3, 19, 3, t)

	// Intentional error
	err = a.deleteByID(Items{
		{Price: 11, Amount: 1, ID: 1337},
	}, &s, false)
	if !errors.Is(err, errIDCannotBeMatched) {
		t.Fatalf("expecting %s but received %v", errIDCannotBeMatched, err)
	}

	// Error bypass
	err = a.deleteByID(Items{
		{Price: 11, Amount: 1, ID: 1337},
	}, &s, true)
	if err != nil {
		t.Fatal(err)
	}
}

func TestUpdateInsertByID(t *testing.T) {
	a := asks{}
	s := Stack{}
	asksSnapshot := Items{
		{Price: 1, Amount: 1, ID: 1},
		{Price: 3, Amount: 1, ID: 3},
		{Price: 5, Amount: 1, ID: 5},
		{Price: 7, Amount: 1, ID: 7},
		{Price: 9, Amount: 1, ID: 9},
		{Price: 11, Amount: 1, ID: 11},
	}
	a.load(asksSnapshot, &s)

	// Update one instance with matching ID
	a.updateInsertByID(Items{
		{Price: 1, Amount: 2, ID: 1},
	}, &s)

	Check(a, 7, 37, 6, t)

	// Reset
	a.load(asksSnapshot, &s)

	// Update all instances with matching ID in order
	a.updateInsertByID(Items{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 5, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	}, &s)

	Check(a, 12, 72, 6, t)

	// Update all instances with matching ID in backwards
	a.updateInsertByID(Items{
		{Price: 11, Amount: 2, ID: 11},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 5, Amount: 2, ID: 5},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 1, Amount: 2, ID: 1},
	}, &s)

	Check(a, 12, 72, 6, t)

	// Update all instances with matching ID all over the ship
	a.updateInsertByID(Items{
		{Price: 11, Amount: 2, ID: 11},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 1, Amount: 2, ID: 1},
		{Price: 5, Amount: 2, ID: 5},
	}, &s)

	Check(a, 12, 72, 6, t)

	// Update all instances move one before ID in middle
	a.updateInsertByID(Items{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 2, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	}, &s)

	Check(a, 12, 66, 6, t)

	// Update all instances move one before ID at head
	a.updateInsertByID(Items{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: .5, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	}, &s)

	Check(a, 12, 63, 6, t)

	// Reset
	a.load(asksSnapshot, &s)

	// Update all instances move one after ID
	a.updateInsertByID(Items{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 8, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	}, &s)

	Check(a, 12, 78, 6, t)

	// Reset
	a.load(asksSnapshot, &s)

	// Update all instances move one after ID to tail
	a.updateInsertByID(Items{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 12, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	}, &s)

	Check(a, 12, 86, 6, t)

	// Update all instances then pop new instance
	a.updateInsertByID(Items{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 12, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
		{Price: 10, Amount: 2, ID: 10},
	}, &s)

	Check(a, 14, 106, 7, t)

	// Reset
	a.load(asksSnapshot, &s)

	// Update all instances pop at head
	a.updateInsertByID(Items{
		{Price: 0.5, Amount: 2, ID: 0},
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 12, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	}, &s)

	Check(a, 14, 87, 7, t)

	// bookmark head and move to a different location
	a.updateInsertByID(Items{
		{Price: 1.5, Amount: 2, ID: 0},
	}, &s)

	Check(a, 14, 89, 7, t)

	// Bids -------------------------------------------------------------------

	b := bids{}
	bidsSnapshot := Items{
		{Price: 11, Amount: 1, ID: 11},
		{Price: 9, Amount: 1, ID: 9},
		{Price: 7, Amount: 1, ID: 7},
		{Price: 5, Amount: 1, ID: 5},
		{Price: 3, Amount: 1, ID: 3},
		{Price: 1, Amount: 1, ID: 1},
	}
	b.load(bidsSnapshot, &s)

	// Update one instance with matching ID
	b.updateInsertByID(Items{
		{Price: 1, Amount: 2, ID: 1},
	}, &s)

	Check(b, 7, 37, 6, t)

	// Reset
	b.load(bidsSnapshot, &s)

	// Update all instances with matching ID in order
	b.updateInsertByID(Items{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 5, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	}, &s)

	Check(b, 12, 72, 6, t)

	// Update all instances with matching ID in backwards
	b.updateInsertByID(Items{
		{Price: 11, Amount: 2, ID: 11},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 5, Amount: 2, ID: 5},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 1, Amount: 2, ID: 1},
	}, &s)

	Check(b, 12, 72, 6, t)

	// Update all instances with matching ID all over the ship
	b.updateInsertByID(Items{
		{Price: 11, Amount: 2, ID: 11},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 1, Amount: 2, ID: 1},
		{Price: 5, Amount: 2, ID: 5},
	}, &s)

	Check(b, 12, 72, 6, t)

	// Update all instances move one before ID in middle
	b.updateInsertByID(Items{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 2, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	}, &s)

	Check(b, 12, 66, 6, t)

	// Update all instances move one before ID at head
	b.updateInsertByID(Items{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: .5, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	}, &s)

	Check(b, 12, 63, 6, t)

	// Reset
	b.load(bidsSnapshot, &s)

	// Update all instances move one after ID
	b.updateInsertByID(Items{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 8, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	}, &s)

	Check(b, 12, 78, 6, t)

	// Reset
	b.load(bidsSnapshot, &s)

	// Update all instances move one after ID to tail
	b.updateInsertByID(Items{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 12, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	}, &s)

	Check(b, 12, 86, 6, t)

	// Update all instances then pop new instance
	b.updateInsertByID(Items{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 12, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
		{Price: 10, Amount: 2, ID: 10},
	}, &s)

	Check(b, 14, 106, 7, t)

	// Reset
	b.load(bidsSnapshot, &s)

	// Update all instances pop at tail
	b.updateInsertByID(Items{
		{Price: 0.5, Amount: 2, ID: 0},
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 12, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	}, &s)

	Check(b, 14, 87, 7, t)

	// bookmark head and move to a different location
	b.updateInsertByID(Items{
		{Price: 9.5, Amount: 2, ID: 0},
	}, &s)

	Check(b, 14, 89, 7, t)
}

func TestInsertUpdatesBid(t *testing.T) {
	b := bids{}
	s := &Stack{}
	bidsSnapshot := Items{
		{Price: 11, Amount: 1, ID: 11},
		{Price: 9, Amount: 1, ID: 9},
		{Price: 7, Amount: 1, ID: 7},
		{Price: 5, Amount: 1, ID: 5},
		{Price: 3, Amount: 1, ID: 3},
		{Price: 1, Amount: 1, ID: 1},
	}
	b.load(bidsSnapshot, s)

	err := b.insertUpdates(Items{
		{Price: 11, Amount: 1, ID: 11},
		{Price: 9, Amount: 1, ID: 9},
		{Price: 7, Amount: 1, ID: 7},
		{Price: 5, Amount: 1, ID: 5},
		{Price: 3, Amount: 1, ID: 3},
		{Price: 1, Amount: 1, ID: 1},
	}, s)
	if !errors.Is(err, errCollisionDetected) {
		t.Fatalf("expected error %s but received %v", errCollisionDetected, err)
	}

	Check(b, 6, 36, 6, t)

	// Insert at head
	err = b.insertUpdates(Items{
		{Price: 12, Amount: 1, ID: 11},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(b, 7, 48, 7, t)

	// Insert at tail
	err = b.insertUpdates(Items{
		{Price: 0.5, Amount: 1, ID: 12},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(b, 8, 48.5, 8, t)

	// Insert at mid
	err = b.insertUpdates(Items{
		{Price: 5.5, Amount: 1, ID: 13},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(b, 9, 54, 9, t)

	// purge
	b.load(nil, s)

	// Add one at head
	err = b.insertUpdates(Items{
		{Price: 5.5, Amount: 1, ID: 13},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(b, 1, 5.5, 1, t)
}

func TestInsertUpdatesAsk(t *testing.T) {
	a := asks{}
	s := &Stack{}
	askSnapshot := Items{
		{Price: 1, Amount: 1, ID: 1},
		{Price: 3, Amount: 1, ID: 3},
		{Price: 5, Amount: 1, ID: 5},
		{Price: 7, Amount: 1, ID: 7},
		{Price: 9, Amount: 1, ID: 9},
		{Price: 11, Amount: 1, ID: 11},
	}
	a.load(askSnapshot, s)

	err := a.insertUpdates(Items{
		{Price: 11, Amount: 1, ID: 11},
		{Price: 9, Amount: 1, ID: 9},
		{Price: 7, Amount: 1, ID: 7},
		{Price: 5, Amount: 1, ID: 5},
		{Price: 3, Amount: 1, ID: 3},
		{Price: 1, Amount: 1, ID: 1},
	}, s)
	if !errors.Is(err, errCollisionDetected) {
		t.Fatalf("expected error %s but received %v", errCollisionDetected, err)
	}

	Check(a, 6, 36, 6, t)

	// Insert at tail
	err = a.insertUpdates(Items{
		{Price: 12, Amount: 1, ID: 11},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(a, 7, 48, 7, t)

	// Insert at head
	err = a.insertUpdates(Items{
		{Price: 0.5, Amount: 1, ID: 12},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(a, 8, 48.5, 8, t)

	// Insert at mid
	err = a.insertUpdates(Items{
		{Price: 5.5, Amount: 1, ID: 13},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(a, 9, 54, 9, t)

	// purge
	a.load(nil, s)

	// Add one at head
	err = a.insertUpdates(Items{
		{Price: 5.5, Amount: 1, ID: 13},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(a, 1, 5.5, 1, t)
}

// check checks depth values after an update has taken place
func Check(depth interface{}, liquidity, value float64, nodeCount int, t *testing.T) {
	t.Helper()
	b, isBid := depth.(bids)
	a, isAsk := depth.(asks)

	var ll linkedList
	if isBid {
		ll = b.linkedList
	} else if isAsk {
		ll = a.linkedList
	} else {
		t.Fatal("value passed in is not of type bids or asks")
	}

	if ll.liquidity() != liquidity {
		ll.display()
		t.Fatalf("mismatched liquidity expecting %v but received %v",
			liquidity,
			ll.liquidity())
	}

	valueTotal := ll.value()
	if valueTotal != value {
		ll.display()
		t.Fatalf("mismatched total value expecting %v but received %v",
			value,
			valueTotal)
	}

	if ll.length != nodeCount {
		ll.display()
		t.Fatalf("mismatched node count expecting %v but received %v",
			nodeCount,
			ll.length)
	}

	if ll.head == nil {
		return
	}

	var tail *Node
	var price float64
	for tip := ll.head; ; tip = tip.next {
		if price == 0 {
			price = tip.value.Price
		} else if isBid && price < tip.value.Price {
			ll.display()
			t.Fatal("Bid pricing out of order should be descending")
		} else if isAsk && price > tip.value.Price {
			ll.display()
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
		ll.display()
		fmt.Println(liquidity, liqReversed)
		t.Fatalf("mismatched liquidity when reversing direction expecting %v but received %v",
			0,
			liquidity-liqReversed)
	}

	if nodeCount-nodeReversed != 0 {
		ll.display()
		t.Fatalf("mismatched node count when reversing direction expecting %v but received %v",
			0,
			nodeCount-nodeReversed)
	}

	if value-valReversed != 0 {
		ll.display()
		fmt.Println(valReversed, value)
		t.Fatalf("mismatched total book value when reversing direction expecting %v but received %v",
			0,
			value-valReversed)
	}
}

func TestAmount(t *testing.T) {
	a := asks{}
	s := &Stack{}
	askSnapshot := Items{
		{Price: 1, Amount: 1, ID: 1},
		{Price: 3, Amount: 1, ID: 3},
		{Price: 5, Amount: 1, ID: 5},
		{Price: 7, Amount: 1, ID: 7},
		{Price: 9, Amount: 1, ID: 9},
		{Price: 11, Amount: 1, ID: 11},
	}
	a.load(askSnapshot, s)

	liquidity, value := a.amount()
	if liquidity != 6 {
		t.Fatalf("incorrect liquidity calculation expected 6 but receieved %f", liquidity)
	}

	if value != 36 {
		t.Fatalf("incorrect value calculation expected 36 but receieved %f", value)
	}
}
