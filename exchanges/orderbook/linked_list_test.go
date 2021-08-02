package orderbook

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

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
	for tip := ll.head; tip != nil; tip = tip.Next {
		fmt.Printf("NODE: %+v %p \n", tip, tip)
	}
	fmt.Println()
}

func TestLoad(t *testing.T) {
	list := asks{}
	Check(list, 0, 0, 0, t)

	stack := newStack()
	list.load(Items{
		{Price: 1, Amount: 1},
		{Price: 3, Amount: 1},
		{Price: 5, Amount: 1},
		{Price: 7, Amount: 1},
		{Price: 9, Amount: 1},
		{Price: 11, Amount: 1},
	}, stack)

	if stack.getCount() != 0 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 0, stack.getCount())
	}

	Check(list, 6, 36, 6, t)

	list.load(Items{
		{Price: 1, Amount: 1},
		{Price: 3, Amount: 1},
		{Price: 5, Amount: 1},
	}, stack)

	if stack.getCount() != 3 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 3, stack.getCount())
	}

	Check(list, 3, 9, 3, t)

	list.load(Items{
		{Price: 1, Amount: 1},
		{Price: 3, Amount: 1},
		{Price: 5, Amount: 1},
		{Price: 7, Amount: 1},
	}, stack)

	if stack.getCount() != 2 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 2, stack.getCount())
	}

	Check(list, 4, 16, 4, t)

	// purge entire list
	list.load(nil, stack)

	if stack.getCount() != 6 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 6, stack.getCount())
	}

	Check(list, 0, 0, 0, t)
}

// 22222386	        57.3 ns/op	       0 B/op	       0 allocs/op (old)
// 27906781	        42.4 ns/op	       0 B/op	       0 allocs/op (new)
func BenchmarkLoad(b *testing.B) {
	ll := linkedList{}
	s := newStack()
	for i := 0; i < b.N; i++ {
		ll.load(ask, s)
	}
}

func TestUpdateInsertByPrice(t *testing.T) {
	a := asks{}
	stack := newStack()
	asksSnapshot := Items{
		{Price: 1, Amount: 1},
		{Price: 3, Amount: 1},
		{Price: 5, Amount: 1},
		{Price: 7, Amount: 1},
		{Price: 9, Amount: 1},
		{Price: 11, Amount: 1},
	}
	a.load(asksSnapshot, stack)

	// Update one instance with matching price
	a.updateInsertByPrice(Items{
		{Price: 1, Amount: 2},
	}, stack, 0, getNow())

	Check(a, 7, 37, 6, t)

	if stack.getCount() != 0 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 0, stack.getCount())
	}

	// Insert at head
	a.updateInsertByPrice(Items{
		{Price: 0.5, Amount: 2},
	}, stack, 0, getNow())

	Check(a, 9, 38, 7, t)

	if stack.getCount() != 0 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 0, stack.getCount())
	}

	// Insert at tail
	a.updateInsertByPrice(Items{
		{Price: 12, Amount: 2},
	}, stack, 0, getNow())

	Check(a, 11, 62, 8, t)

	if stack.getCount() != 0 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 0, stack.getCount())
	}

	// Insert between price and up to and beyond max allowable depth level
	a.updateInsertByPrice(Items{
		{Price: 11.5, Amount: 2},
		{Price: 10.5, Amount: 2},
		{Price: 13, Amount: 2},
	}, stack, 10, getNow())

	Check(a, 15, 106, 10, t)

	if stack.getCount() != 1 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 1, stack.getCount())
	}

	// delete at tail
	a.updateInsertByPrice(Items{
		{Price: 12, Amount: 0},
	}, stack, 0, getNow())

	Check(a, 13, 82, 9, t)

	if stack.getCount() != 2 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 2, stack.getCount())
	}

	// delete at mid
	a.updateInsertByPrice(Items{
		{Price: 7, Amount: 0},
	}, stack, 0, getNow())

	Check(a, 12, 75, 8, t)

	if stack.getCount() != 3 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 3, stack.getCount())
	}

	// delete at head
	a.updateInsertByPrice(Items{
		{Price: 0.5, Amount: 0},
	}, stack, 0, getNow())

	Check(a, 10, 74, 7, t)

	if stack.getCount() != 4 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 4, stack.getCount())
	}

	// purge if liquidity plunges to zero
	a.load(nil, stack)

	// rebuild everything again
	a.updateInsertByPrice(Items{
		{Price: 1, Amount: 1},
		{Price: 3, Amount: 1},
		{Price: 5, Amount: 1},
		{Price: 7, Amount: 1},
		{Price: 9, Amount: 1},
		{Price: 11, Amount: 1},
	}, stack, 0, getNow())

	Check(a, 6, 36, 6, t)

	if stack.getCount() != 5 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 4, stack.getCount())
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
	b.load(bidsSnapshot, stack)

	// Update one instance with matching price
	b.updateInsertByPrice(Items{
		{Price: 11, Amount: 2},
	}, stack, 0, getNow())

	Check(b, 7, 47, 6, t)

	if stack.getCount() != 0 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 0, stack.getCount())
	}

	// Insert at head
	b.updateInsertByPrice(Items{
		{Price: 12, Amount: 2},
	}, stack, 0, getNow())

	Check(b, 9, 71, 7, t)

	if stack.getCount() != 0 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 0, stack.getCount())
	}

	// Insert at tail
	b.updateInsertByPrice(Items{
		{Price: 0.5, Amount: 2},
	}, stack, 0, getNow())

	Check(b, 11, 72, 8, t)

	if stack.getCount() != 0 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 0, stack.getCount())
	}

	// Insert between price and up to and beyond max allowable depth level
	b.updateInsertByPrice(Items{
		{Price: 11.5, Amount: 2},
		{Price: 10.5, Amount: 2},
		{Price: 13, Amount: 2},
	}, stack, 10, getNow())

	Check(b, 15, 141, 10, t)

	if stack.getCount() != 1 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 0, stack.getCount())
	}

	// Insert between price and up to and beyond max allowable depth level
	b.updateInsertByPrice(Items{
		{Price: 1, Amount: 0},
	}, stack, 0, getNow())

	Check(b, 14, 140, 9, t)

	if stack.getCount() != 2 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 2, stack.getCount())
	}

	// delete at mid
	b.updateInsertByPrice(Items{
		{Price: 10.5, Amount: 0},
	}, stack, 0, getNow())

	Check(b, 12, 119, 8, t)

	if stack.getCount() != 3 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 3, stack.getCount())
	}

	// delete at head
	b.updateInsertByPrice(Items{
		{Price: 13, Amount: 0},
	}, stack, 0, getNow())

	Check(b, 10, 93, 7, t)

	if stack.getCount() != 4 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 4, stack.getCount())
	}

	// purge if liquidity plunges to zero
	b.load(nil, stack)

	// rebuild everything again
	b.updateInsertByPrice(Items{
		{Price: 1, Amount: 1},
		{Price: 3, Amount: 1},
		{Price: 5, Amount: 1},
		{Price: 7, Amount: 1},
		{Price: 9, Amount: 1},
		{Price: 11, Amount: 1},
	}, stack, 0, getNow())

	Check(b, 6, 36, 6, t)

	if stack.getCount() != 5 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 4, stack.getCount())
	}
}

func TestCleanup(t *testing.T) {
	a := asks{}
	stack := newStack()
	asksSnapshot := Items{
		{Price: 1, Amount: 1},
		{Price: 3, Amount: 1},
		{Price: 5, Amount: 1},
		{Price: 7, Amount: 1},
		{Price: 9, Amount: 1},
		{Price: 11, Amount: 1},
	}
	a.load(asksSnapshot, stack)

	a.cleanup(6, stack)
	Check(a, 6, 36, 6, t)
	a.cleanup(5, stack)
	Check(a, 5, 25, 5, t)
	a.cleanup(1, stack)
	Check(a, 1, 1, 1, t)
	a.cleanup(10, stack)
	Check(a, 1, 1, 1, t)
	a.cleanup(0, stack) // will purge, underlying checks are done elseware to prevent this
	Check(a, 0, 0, 0, t)
}

// 46154023	        24.0 ns/op	       0 B/op	       0 allocs/op (old)
// 134830672	         9.83 ns/op	       0 B/op	       0 allocs/op (new)
func BenchmarkUpdateInsertByPrice_Amend(b *testing.B) {
	a := asks{}
	stack := newStack()

	a.load(ask, stack)

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
		a.updateInsertByPrice(updates, stack, 0, getNow())
	}
}

// 49763002	        24.9 ns/op	       0 B/op	       0 allocs/op
func BenchmarkUpdateInsertByPrice_Insert_Delete(b *testing.B) {
	a := asks{}
	stack := newStack()

	a.load(ask, stack)

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
		a.updateInsertByPrice(updates, stack, 0, getNow())
	}
}

func TestUpdateByID(t *testing.T) {
	a := asks{}
	s := newStack()
	asksSnapshot := Items{
		{Price: 1, Amount: 1, ID: 1},
		{Price: 3, Amount: 1, ID: 3},
		{Price: 5, Amount: 1, ID: 5},
		{Price: 7, Amount: 1, ID: 7},
		{Price: 9, Amount: 1, ID: 9},
		{Price: 11, Amount: 1, ID: 11},
	}
	a.load(asksSnapshot, s)

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

	err = a.updateByID(Items{ // Simulate Bitmex updating
		{Price: 0, Amount: 1337, ID: 3},
	})
	if !errors.Is(err, nil) {
		t.Fatalf("expecting %v but received %v", nil, err)
	}

	if a.retrieve()[1].Price == 0 {
		t.Fatal("price should not be replaced with zero")
	}

	if a.retrieve()[1].Amount != 1337 {
		t.Fatal("unexpected value for update")
	}
}

// 46043871	        25.9 ns/op	       0 B/op	       0 allocs/op
func BenchmarkUpdateByID(b *testing.B) {
	asks := linkedList{}
	s := newStack()
	asksSnapshot := Items{
		{Price: 1, Amount: 1, ID: 1},
		{Price: 3, Amount: 1, ID: 3},
		{Price: 5, Amount: 1, ID: 5},
		{Price: 7, Amount: 1, ID: 7},
		{Price: 9, Amount: 1, ID: 9},
		{Price: 11, Amount: 1, ID: 11},
	}
	asks.load(asksSnapshot, s)

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
	s := newStack()
	asksSnapshot := Items{
		{Price: 1, Amount: 1, ID: 1},
		{Price: 3, Amount: 1, ID: 3},
		{Price: 5, Amount: 1, ID: 5},
		{Price: 7, Amount: 1, ID: 7},
		{Price: 9, Amount: 1, ID: 9},
		{Price: 11, Amount: 1, ID: 11},
	}
	a.load(asksSnapshot, s)

	// Delete at head
	err := a.deleteByID(Items{
		{Price: 1, Amount: 1, ID: 1},
	}, s, false)
	if err != nil {
		t.Fatal(err)
	}

	Check(a, 5, 35, 5, t)

	// Delete at tail
	err = a.deleteByID(Items{
		{Price: 1, Amount: 1, ID: 11},
	}, s, false)
	if err != nil {
		t.Fatal(err)
	}

	Check(a, 4, 24, 4, t)

	// Delete in middle
	err = a.deleteByID(Items{
		{Price: 1, Amount: 1, ID: 5},
	}, s, false)
	if err != nil {
		t.Fatal(err)
	}

	Check(a, 3, 19, 3, t)

	// Intentional error
	err = a.deleteByID(Items{
		{Price: 11, Amount: 1, ID: 1337},
	}, s, false)
	if !errors.Is(err, errIDCannotBeMatched) {
		t.Fatalf("expecting %s but received %v", errIDCannotBeMatched, err)
	}

	// Error bypass
	err = a.deleteByID(Items{
		{Price: 11, Amount: 1, ID: 1337},
	}, s, true)
	if err != nil {
		t.Fatal(err)
	}
}

func TestUpdateInsertByIDAsk(t *testing.T) {
	a := asks{}
	s := newStack()
	asksSnapshot := Items{
		{Price: 1, Amount: 1, ID: 1},
		{Price: 3, Amount: 1, ID: 3},
		{Price: 5, Amount: 1, ID: 5},
		{Price: 7, Amount: 1, ID: 7},
		{Price: 9, Amount: 1, ID: 9},
		{Price: 11, Amount: 1, ID: 11},
	}
	a.load(asksSnapshot, s)

	// Update one instance with matching ID
	err := a.updateInsertByID(Items{
		{Price: 1, Amount: 2, ID: 1},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(a, 7, 37, 6, t)

	// Reset
	a.load(asksSnapshot, s)

	// Update all instances with matching ID in order
	err = a.updateInsertByID(Items{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 5, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(a, 12, 72, 6, t)

	// Update all instances with matching ID in backwards
	err = a.updateInsertByID(Items{
		{Price: 11, Amount: 2, ID: 11},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 5, Amount: 2, ID: 5},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 1, Amount: 2, ID: 1},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(a, 12, 72, 6, t)

	// Update all instances with matching ID all over the ship
	err = a.updateInsertByID(Items{
		{Price: 11, Amount: 2, ID: 11},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 1, Amount: 2, ID: 1},
		{Price: 5, Amount: 2, ID: 5},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(a, 12, 72, 6, t)

	// Update all instances move one before ID in middle
	err = a.updateInsertByID(Items{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 2, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(a, 12, 66, 6, t)

	// Update all instances move one before ID at head
	err = a.updateInsertByID(Items{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: .5, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(a, 12, 63, 6, t)

	// Reset
	a.load(asksSnapshot, s)

	// Update all instances move one after ID
	err = a.updateInsertByID(Items{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 8, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(a, 12, 78, 6, t)

	// Reset
	a.load(asksSnapshot, s)

	// Update all instances move one after ID to tail
	err = a.updateInsertByID(Items{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 12, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(a, 12, 86, 6, t)

	// Update all instances then pop new instance
	err = a.updateInsertByID(Items{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 12, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
		{Price: 10, Amount: 2, ID: 10},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(a, 14, 106, 7, t)

	// Reset
	a.load(asksSnapshot, s)

	// Update all instances pop at head
	err = a.updateInsertByID(Items{
		{Price: 0.5, Amount: 2, ID: 0},
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 12, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(a, 14, 87, 7, t)

	// bookmark head and move to mid
	err = a.updateInsertByID(Items{
		{Price: 7.5, Amount: 2, ID: 0},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(a, 14, 101, 7, t)

	// bookmark head and move to tail
	err = a.updateInsertByID(Items{
		{Price: 12.5, Amount: 2, ID: 1},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(a, 14, 124, 7, t)

	// move tail location to head
	err = a.updateInsertByID(Items{
		{Price: 2.5, Amount: 2, ID: 1},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(a, 14, 104, 7, t)

	// move tail location to mid
	err = a.updateInsertByID(Items{
		{Price: 8, Amount: 2, ID: 5},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(a, 14, 96, 7, t)

	// insert at tail dont match
	err = a.updateInsertByID(Items{
		{Price: 30, Amount: 2, ID: 1234},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(a, 16, 156, 8, t)

	// insert between last and 2nd last
	err = a.updateInsertByID(Items{
		{Price: 12, Amount: 2, ID: 12345},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(a, 18, 180, 9, t)

	// readjust at end
	err = a.updateInsertByID(Items{
		{Price: 29, Amount: 3, ID: 1234},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(a, 19, 207, 9, t)

	// readjust further and decrease price past tail
	err = a.updateInsertByID(Items{
		{Price: 31, Amount: 3, ID: 1234},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(a, 19, 213, 9, t)

	// purge
	a.load(nil, s)

	// insert with no liquidity and jumbled
	err = a.updateInsertByID(Items{
		{Price: 11, Amount: 2, ID: 11},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 0.5, Amount: 2, ID: 0},
		{Price: 12, Amount: 2, ID: 5},
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(a, 14, 87, 7, t)
}

func TestUpdateInsertByIDBids(t *testing.T) {
	b := bids{}
	s := newStack()
	bidsSnapshot := Items{
		{Price: 11, Amount: 1, ID: 11},
		{Price: 9, Amount: 1, ID: 9},
		{Price: 7, Amount: 1, ID: 7},
		{Price: 5, Amount: 1, ID: 5},
		{Price: 3, Amount: 1, ID: 3},
		{Price: 1, Amount: 1, ID: 1},
	}
	b.load(bidsSnapshot, s)

	// Update one instance with matching ID
	err := b.updateInsertByID(Items{
		{Price: 1, Amount: 2, ID: 1},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(b, 7, 37, 6, t)

	// Reset
	b.load(bidsSnapshot, s)

	// Update all instances with matching ID in order
	err = b.updateInsertByID(Items{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 5, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(b, 12, 72, 6, t)

	// Update all instances with matching ID in backwards
	err = b.updateInsertByID(Items{
		{Price: 11, Amount: 2, ID: 11},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 5, Amount: 2, ID: 5},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 1, Amount: 2, ID: 1},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(b, 12, 72, 6, t)

	// Update all instances with matching ID all over the ship
	err = b.updateInsertByID(Items{
		{Price: 11, Amount: 2, ID: 11},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 1, Amount: 2, ID: 1},
		{Price: 5, Amount: 2, ID: 5},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(b, 12, 72, 6, t)

	// Update all instances move one before ID in middle
	err = b.updateInsertByID(Items{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 2, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(b, 12, 66, 6, t)

	// Update all instances move one before ID at head
	err = b.updateInsertByID(Items{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: .5, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(b, 12, 63, 6, t)

	// Reset
	b.load(bidsSnapshot, s)

	// Update all instances move one after ID
	err = b.updateInsertByID(Items{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 8, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(b, 12, 78, 6, t)

	// Reset
	b.load(bidsSnapshot, s)

	// Update all instances move one after ID to tail
	err = b.updateInsertByID(Items{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 12, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(b, 12, 86, 6, t)

	// Update all instances then pop new instance
	err = b.updateInsertByID(Items{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 12, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
		{Price: 10, Amount: 2, ID: 10},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(b, 14, 106, 7, t)

	// Reset
	b.load(bidsSnapshot, s)

	// Update all instances pop at tail
	err = b.updateInsertByID(Items{
		{Price: 0.5, Amount: 2, ID: 0},
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 12, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(b, 14, 87, 7, t)

	// bookmark head and move to mid
	err = b.updateInsertByID(Items{
		{Price: 9.5, Amount: 2, ID: 5},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(b, 14, 82, 7, t)

	// bookmark head and move to tail
	err = b.updateInsertByID(Items{
		{Price: 0.25, Amount: 2, ID: 11},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(b, 14, 60.5, 7, t)

	// move tail location to head
	err = b.updateInsertByID(Items{
		{Price: 10, Amount: 2, ID: 11},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(b, 14, 80, 7, t)

	// move tail location to mid
	err = b.updateInsertByID(Items{
		{Price: 7.5, Amount: 2, ID: 0},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(b, 14, 94, 7, t)

	// insert at head dont match
	err = b.updateInsertByID(Items{
		{Price: 30, Amount: 2, ID: 1234},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(b, 16, 154, 8, t)

	// insert between last and 2nd last
	err = b.updateInsertByID(Items{
		{Price: 1.5, Amount: 2, ID: 12345},
	}, s)
	if err != nil {
		t.Fatal(err)
	}
	Check(b, 18, 157, 9, t)

	// readjust at end
	err = b.updateInsertByID(Items{
		{Price: 1, Amount: 3, ID: 1},
	}, s)
	if err != nil {
		t.Fatal(err)
	}
	Check(b, 19, 158, 9, t)

	// readjust further and decrease price past tail
	err = b.updateInsertByID(Items{
		{Price: .9, Amount: 3, ID: 1},
	}, s)
	if err != nil {
		t.Fatal(err)
	}
	Check(b, 19, 157.7, 9, t)

	// purge
	b.load(nil, s)

	// insert with no liquidity and jumbled
	err = b.updateInsertByID(Items{
		{Price: 0.5, Amount: 2, ID: 0},
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 12, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(b, 14, 87, 7, t)
}

func TestInsertUpdatesBid(t *testing.T) {
	b := bids{}
	s := newStack()
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
	s := newStack()
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
	switch {
	case isBid:
		ll = b.linkedList
	case isAsk:
		ll = a.linkedList
	default:
		t.Fatal("value passed in is not of type bids or asks")
	}

	liquidityTotal, valueTotal := ll.amount()

	if liquidityTotal != liquidity {
		ll.display()
		t.Fatalf("mismatched liquidity expecting %v but received %v",
			liquidity,
			liquidityTotal)
	}

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
	for tip := ll.head; ; tip = tip.Next {
		switch {
		case price == 0:
			price = tip.Value.Price
		case isBid && price < tip.Value.Price:
			ll.display()
			t.Fatal("Bid pricing out of order should be descending")
		case isAsk && price > tip.Value.Price:
			ll.display()
			t.Fatal("Ask pricing out of order should be ascending")
		default:
			price = tip.Value.Price
		}

		if tip.Next == nil {
			tail = tip
			break
		}
	}

	var liqReversed, valReversed float64
	var nodeReversed int
	for tip := tail; tip != nil; tip = tip.Prev {
		liqReversed += tip.Value.Amount
		valReversed += tip.Value.Amount * tip.Value.Price
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
	s := newStack()
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
		t.Fatalf("incorrect liquidity calculation expected 6 but received %f", liquidity)
	}

	if value != 36 {
		t.Fatalf("incorrect value calculation expected 36 but received %f", value)
	}
}

func TestShiftBookmark(t *testing.T) {
	bookmarkedNode := &Node{
		Value: Item{
			ID:     1337,
			Amount: 1,
			Price:  2,
		},
		Next:    nil,
		Prev:    nil,
		shelved: time.Time{},
	}

	originalBookmarkPrev := &Node{
		Value: Item{
			ID: 1336,
		},
		Next:    bookmarkedNode,
		Prev:    nil, // At head
		shelved: time.Time{},
	}
	originalBookmarkNext := &Node{
		Value: Item{
			ID: 1338,
		},
		Next: nil, // This can be left nil in actuality this will be
		// populated
		Prev:    bookmarkedNode,
		shelved: time.Time{},
	}

	// associate previous and next nodes to bookmarked node
	bookmarkedNode.Prev = originalBookmarkPrev
	bookmarkedNode.Next = originalBookmarkNext

	tip := &Node{
		Value: Item{
			ID: 69420,
		},
		Next:    nil, // In this case tip will be at tail
		Prev:    nil,
		shelved: time.Time{},
	}

	tipprev := &Node{
		Value: Item{
			ID: 69419,
		},
		Next: tip,
		Prev: nil, // This can be left nil in actuality this will be
		// populated
		shelved: time.Time{},
	}

	// associate tips prev field with the correct prev node
	tip.Prev = tipprev

	if !shiftBookmark(tip, &bookmarkedNode, nil, Item{Amount: 1336, ID: 1337, Price: 9999}) {
		t.Fatal("There should be liquidity so we don't need to set tip to bookmark")
	}

	if bookmarkedNode.Value.Price != 9999 ||
		bookmarkedNode.Value.Amount != 1336 ||
		bookmarkedNode.Value.ID != 1337 {
		t.Fatal("bookmarked details are not set correctly with shift")
	}

	if bookmarkedNode.Prev != tip {
		t.Fatal("bookmarked prev memory address does not point to tip")
	}

	if bookmarkedNode.Next != nil {
		t.Fatal("bookmarked next is at tail and should be nil")
	}

	if bookmarkedNode.Next != nil {
		t.Fatal("bookmarked next is at tail and should be nil")
	}

	if originalBookmarkPrev.Next != originalBookmarkNext {
		t.Fatal("original bookmarked prev node should be associated with original bookmarked next node")
	}

	if originalBookmarkNext.Prev != originalBookmarkPrev {
		t.Fatal("original bookmarked next node should be associated with original bookmarked prev node")
	}

	var nilBookmark *Node

	if shiftBookmark(tip, &nilBookmark, nil, Item{Amount: 1336, ID: 1337, Price: 9999}) {
		t.Fatal("there should not be a bookmarked node")
	}

	if tip != nilBookmark {
		t.Fatal("nilBookmark not reassigned")
	}

	head := bookmarkedNode
	bookmarkedNode.Prev = nil
	bookmarkedNode.Next = originalBookmarkNext
	tip.Next = nil

	if !shiftBookmark(tip, &bookmarkedNode, &head, Item{Amount: 1336, ID: 1337, Price: 9999}) {
		t.Fatal("There should be liquidity so we don't need to set tip to bookmark")
	}

	if head != originalBookmarkNext {
		t.Fatal("unexpected pointer variable")
	}
}
