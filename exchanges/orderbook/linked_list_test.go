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

// Display displays depth content for tests
func (ll *linkedList) display() {
	for tip := ll.head; tip != nil; tip = tip.Next {
		fmt.Printf("NODE: %+v %p \n", tip, tip)
	}
	fmt.Println()
}

func TestLoad(t *testing.T) {
	list := asks{}
	Check(t, list, 0, 0, 0)

	stack := newStack()
	list.load(Items{
		{Price: 1, Amount: 1},
		{Price: 3, Amount: 1},
		{Price: 5, Amount: 1},
		{Price: 7, Amount: 1},
		{Price: 9, Amount: 1},
		{Price: 11, Amount: 1},
	}, stack, time.Now())

	if stack.getCount() != 0 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 0, stack.getCount())
	}

	Check(t, list, 6, 36, 6)

	list.load(Items{
		{Price: 1, Amount: 1},
		{Price: 3, Amount: 1},
		{Price: 5, Amount: 1},
	}, stack, time.Now())

	if stack.getCount() != 3 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 3, stack.getCount())
	}

	Check(t, list, 3, 9, 3)

	list.load(Items{
		{Price: 1, Amount: 1},
		{Price: 3, Amount: 1},
		{Price: 5, Amount: 1},
		{Price: 7, Amount: 1},
	}, stack, time.Now())

	if stack.getCount() != 2 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 2, stack.getCount())
	}

	Check(t, list, 4, 16, 4)

	// purge entire list
	list.load(nil, stack, time.Now())

	if stack.getCount() != 6 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 6, stack.getCount())
	}

	Check(t, list, 0, 0, 0)
}

// 22222386	        57.3 ns/op	       0 B/op	       0 allocs/op (old)
// 27906781	        42.4 ns/op	       0 B/op	       0 allocs/op (new)
func BenchmarkLoad(b *testing.B) {
	ll := linkedList{}
	s := newStack()
	for i := 0; i < b.N; i++ {
		ll.load(ask, s, time.Now())
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
	a.load(asksSnapshot, stack, time.Now())

	// Update one instance with matching price
	a.updateInsertByPrice(Items{
		{Price: 1, Amount: 2},
	}, stack, 0, time.Now())

	Check(t, a, 7, 37, 6)

	if stack.getCount() != 0 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 0, stack.getCount())
	}

	// Insert at head
	a.updateInsertByPrice(Items{
		{Price: 0.5, Amount: 2},
	}, stack, 0, time.Now())

	Check(t, a, 9, 38, 7)

	if stack.getCount() != 0 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 0, stack.getCount())
	}

	// Insert at tail
	a.updateInsertByPrice(Items{
		{Price: 12, Amount: 2},
	}, stack, 0, time.Now())

	Check(t, a, 11, 62, 8)

	if stack.getCount() != 0 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 0, stack.getCount())
	}

	// Insert between price and up to and beyond max allowable depth level
	a.updateInsertByPrice(Items{
		{Price: 11.5, Amount: 2},
		{Price: 10.5, Amount: 2},
		{Price: 13, Amount: 2},
	}, stack, 10, time.Now())

	Check(t, a, 15, 106, 10)

	if stack.getCount() != 1 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 1, stack.getCount())
	}

	// delete at tail
	a.updateInsertByPrice(Items{
		{Price: 12, Amount: 0},
	}, stack, 0, time.Now())

	Check(t, a, 13, 82, 9)

	if stack.getCount() != 2 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 2, stack.getCount())
	}

	// delete at mid
	a.updateInsertByPrice(Items{
		{Price: 7, Amount: 0},
	}, stack, 0, time.Now())

	Check(t, a, 12, 75, 8)

	if stack.getCount() != 3 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 3, stack.getCount())
	}

	// delete at head
	a.updateInsertByPrice(Items{
		{Price: 0.5, Amount: 0},
	}, stack, 0, time.Now())

	Check(t, a, 10, 74, 7)

	if stack.getCount() != 4 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 4, stack.getCount())
	}

	// purge if liquidity plunges to zero
	a.load(nil, stack, time.Now())

	// rebuild everything again
	a.updateInsertByPrice(Items{
		{Price: 1, Amount: 1},
		{Price: 3, Amount: 1},
		{Price: 5, Amount: 1},
		{Price: 7, Amount: 1},
		{Price: 9, Amount: 1},
		{Price: 11, Amount: 1},
	}, stack, 0, time.Now())

	Check(t, a, 6, 36, 6)

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
	b.load(bidsSnapshot, stack, time.Now())

	// Update one instance with matching price
	b.updateInsertByPrice(Items{
		{Price: 11, Amount: 2},
	}, stack, 0, time.Now())

	Check(t, b, 7, 47, 6)

	if stack.getCount() != 0 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 0, stack.getCount())
	}

	// Insert at head
	b.updateInsertByPrice(Items{
		{Price: 12, Amount: 2},
	}, stack, 0, time.Now())

	Check(t, b, 9, 71, 7)

	if stack.getCount() != 0 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 0, stack.getCount())
	}

	// Insert at tail
	b.updateInsertByPrice(Items{
		{Price: 0.5, Amount: 2},
	}, stack, 0, time.Now())

	Check(t, b, 11, 72, 8)

	if stack.getCount() != 0 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 0, stack.getCount())
	}

	// Insert between price and up to and beyond max allowable depth level
	b.updateInsertByPrice(Items{
		{Price: 11.5, Amount: 2},
		{Price: 10.5, Amount: 2},
		{Price: 13, Amount: 2},
	}, stack, 10, time.Now())

	Check(t, b, 15, 141, 10)

	if stack.getCount() != 1 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 0, stack.getCount())
	}

	// Insert between price and up to and beyond max allowable depth level
	b.updateInsertByPrice(Items{
		{Price: 1, Amount: 0},
	}, stack, 0, time.Now())

	Check(t, b, 14, 140, 9)

	if stack.getCount() != 2 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 2, stack.getCount())
	}

	// delete at mid
	b.updateInsertByPrice(Items{
		{Price: 10.5, Amount: 0},
	}, stack, 0, time.Now())

	Check(t, b, 12, 119, 8)

	if stack.getCount() != 3 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 3, stack.getCount())
	}

	// delete at head
	b.updateInsertByPrice(Items{
		{Price: 13, Amount: 0},
	}, stack, 0, time.Now())

	Check(t, b, 10, 93, 7)

	if stack.getCount() != 4 {
		t.Fatalf("incorrect stack count expected: %v received: %v", 4, stack.getCount())
	}

	// purge if liquidity plunges to zero
	b.load(nil, stack, time.Now())

	// rebuild everything again
	b.updateInsertByPrice(Items{
		{Price: 1, Amount: 1},
		{Price: 3, Amount: 1},
		{Price: 5, Amount: 1},
		{Price: 7, Amount: 1},
		{Price: 9, Amount: 1},
		{Price: 11, Amount: 1},
	}, stack, 0, time.Now())

	Check(t, b, 6, 36, 6)

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
	a.load(asksSnapshot, stack, time.Now())

	a.cleanup(6, stack, time.Now())
	Check(t, a, 6, 36, 6)
	a.cleanup(5, stack, time.Now())
	Check(t, a, 5, 25, 5)
	a.cleanup(1, stack, time.Now())
	Check(t, a, 1, 1, 1)
	a.cleanup(10, stack, time.Now())
	Check(t, a, 1, 1, 1)
	a.cleanup(0, stack, time.Now()) // will purge, underlying checks are done elseware to prevent this
	Check(t, a, 0, 0, 0)
}

// 46154023	        24.0 ns/op	       0 B/op	       0 allocs/op (old)
// 134830672	         9.83 ns/op	       0 B/op	       0 allocs/op (new)
func BenchmarkUpdateInsertByPrice_Amend(b *testing.B) {
	a := asks{}
	stack := newStack()

	a.load(ask, stack, time.Now())

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
		a.updateInsertByPrice(updates, stack, 0, time.Now())
	}
}

// 49763002	        24.9 ns/op	       0 B/op	       0 allocs/op
func BenchmarkUpdateInsertByPrice_Insert_Delete(b *testing.B) {
	a := asks{}
	stack := newStack()

	a.load(ask, stack, time.Now())

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
		a.updateInsertByPrice(updates, stack, 0, time.Now())
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
	a.load(asksSnapshot, s, time.Now())

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

	Check(t, a, 6, 36, 6)

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

	if got := a.retrieve(2); len(got) != 2 || got[1].Price == 0 {
		t.Fatal("price should not be replaced with zero")
	}

	if got := a.retrieve(3); len(got) != 3 || got[1].Amount != 1337 {
		t.Fatal("unexpected value for update")
	}

	if got := a.retrieve(1000); len(got) != 6 {
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
	asks.load(asksSnapshot, s, time.Now())

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
	a.load(asksSnapshot, s, time.Now())

	// Delete at head
	err := a.deleteByID(Items{
		{Price: 1, Amount: 1, ID: 1},
	}, s, false, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	Check(t, a, 5, 35, 5)

	// Delete at tail
	err = a.deleteByID(Items{
		{Price: 1, Amount: 1, ID: 11},
	}, s, false, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	Check(t, a, 4, 24, 4)

	// Delete in middle
	err = a.deleteByID(Items{
		{Price: 1, Amount: 1, ID: 5},
	}, s, false, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	Check(t, a, 3, 19, 3)

	// Intentional error
	err = a.deleteByID(Items{
		{Price: 11, Amount: 1, ID: 1337},
	}, s, false, time.Now())
	if !errors.Is(err, errIDCannotBeMatched) {
		t.Fatalf("expecting %s but received %v", errIDCannotBeMatched, err)
	}

	// Error bypass
	err = a.deleteByID(Items{
		{Price: 11, Amount: 1, ID: 1337},
	}, s, true, time.Now())
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
	a.load(asksSnapshot, s, time.Now())

	// Update one instance with matching ID
	err := a.updateInsertByID(Items{
		{Price: 1, Amount: 2, ID: 1},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(t, a, 7, 37, 6)

	// Reset
	a.load(asksSnapshot, s, time.Now())

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

	Check(t, a, 12, 72, 6)

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

	Check(t, a, 12, 72, 6)

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

	Check(t, a, 12, 72, 6)

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

	Check(t, a, 12, 66, 6)

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

	Check(t, a, 12, 63, 6)

	// Reset
	a.load(asksSnapshot, s, time.Now())

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

	Check(t, a, 12, 78, 6)

	// Reset
	a.load(asksSnapshot, s, time.Now())

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

	Check(t, a, 12, 86, 6)

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

	Check(t, a, 14, 106, 7)

	// Reset
	a.load(asksSnapshot, s, time.Now())

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

	Check(t, a, 14, 87, 7)

	// bookmark head and move to mid
	err = a.updateInsertByID(Items{
		{Price: 7.5, Amount: 2, ID: 0},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(t, a, 14, 101, 7)

	// bookmark head and move to tail
	err = a.updateInsertByID(Items{
		{Price: 12.5, Amount: 2, ID: 1},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(t, a, 14, 124, 7)

	// move tail location to head
	err = a.updateInsertByID(Items{
		{Price: 2.5, Amount: 2, ID: 1},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(t, a, 14, 104, 7)

	// move tail location to mid
	err = a.updateInsertByID(Items{
		{Price: 8, Amount: 2, ID: 5},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(t, a, 14, 96, 7)

	// insert at tail dont match
	err = a.updateInsertByID(Items{
		{Price: 30, Amount: 2, ID: 1234},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(t, a, 16, 156, 8)

	// insert between last and 2nd last
	err = a.updateInsertByID(Items{
		{Price: 12, Amount: 2, ID: 12345},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(t, a, 18, 180, 9)

	// readjust at end
	err = a.updateInsertByID(Items{
		{Price: 29, Amount: 3, ID: 1234},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(t, a, 19, 207, 9)

	// readjust further and decrease price past tail
	err = a.updateInsertByID(Items{
		{Price: 31, Amount: 3, ID: 1234},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(t, a, 19, 213, 9)

	// purge
	a.load(nil, s, time.Now())

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

	Check(t, a, 14, 87, 7)
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
	b.load(bidsSnapshot, s, time.Now())

	// Update one instance with matching ID
	err := b.updateInsertByID(Items{
		{Price: 1, Amount: 2, ID: 1},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(t, b, 7, 37, 6)

	// Reset
	b.load(bidsSnapshot, s, time.Now())

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

	Check(t, b, 12, 72, 6)

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

	Check(t, b, 12, 72, 6)

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

	Check(t, b, 12, 72, 6)

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

	Check(t, b, 12, 66, 6)

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

	Check(t, b, 12, 63, 6)

	// Reset
	b.load(bidsSnapshot, s, time.Now())

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

	Check(t, b, 12, 78, 6)

	// Reset
	b.load(bidsSnapshot, s, time.Now())

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

	Check(t, b, 12, 86, 6)

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

	Check(t, b, 14, 106, 7)

	// Reset
	b.load(bidsSnapshot, s, time.Now())

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

	Check(t, b, 14, 87, 7)

	// bookmark head and move to mid
	err = b.updateInsertByID(Items{
		{Price: 9.5, Amount: 2, ID: 5},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(t, b, 14, 82, 7)

	// bookmark head and move to tail
	err = b.updateInsertByID(Items{
		{Price: 0.25, Amount: 2, ID: 11},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(t, b, 14, 60.5, 7)

	// move tail location to head
	err = b.updateInsertByID(Items{
		{Price: 10, Amount: 2, ID: 11},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(t, b, 14, 80, 7)

	// move tail location to mid
	err = b.updateInsertByID(Items{
		{Price: 7.5, Amount: 2, ID: 0},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(t, b, 14, 94, 7)

	// insert at head dont match
	err = b.updateInsertByID(Items{
		{Price: 30, Amount: 2, ID: 1234},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(t, b, 16, 154, 8)

	// insert between last and 2nd last
	err = b.updateInsertByID(Items{
		{Price: 1.5, Amount: 2, ID: 12345},
	}, s)
	if err != nil {
		t.Fatal(err)
	}
	Check(t, b, 18, 157, 9)

	// readjust at end
	err = b.updateInsertByID(Items{
		{Price: 1, Amount: 3, ID: 1},
	}, s)
	if err != nil {
		t.Fatal(err)
	}
	Check(t, b, 19, 158, 9)

	// readjust further and decrease price past tail
	err = b.updateInsertByID(Items{
		{Price: .9, Amount: 3, ID: 1},
	}, s)
	if err != nil {
		t.Fatal(err)
	}
	Check(t, b, 19, 157.7, 9)

	// purge
	b.load(nil, s, time.Now())

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

	Check(t, b, 14, 87, 7)
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
	b.load(bidsSnapshot, s, time.Now())

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

	Check(t, b, 6, 36, 6)

	// Insert at head
	err = b.insertUpdates(Items{
		{Price: 12, Amount: 1, ID: 11},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(t, b, 7, 48, 7)

	// Insert at tail
	err = b.insertUpdates(Items{
		{Price: 0.5, Amount: 1, ID: 12},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(t, b, 8, 48.5, 8)

	// Insert at mid
	err = b.insertUpdates(Items{
		{Price: 5.5, Amount: 1, ID: 13},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(t, b, 9, 54, 9)

	// purge
	b.load(nil, s, time.Now())

	// Add one at head
	err = b.insertUpdates(Items{
		{Price: 5.5, Amount: 1, ID: 13},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(t, b, 1, 5.5, 1)
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
	a.load(askSnapshot, s, time.Now())

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

	Check(t, a, 6, 36, 6)

	// Insert at tail
	err = a.insertUpdates(Items{
		{Price: 12, Amount: 1, ID: 11},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(t, a, 7, 48, 7)

	// Insert at head
	err = a.insertUpdates(Items{
		{Price: 0.5, Amount: 1, ID: 12},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(t, a, 8, 48.5, 8)

	// Insert at mid
	err = a.insertUpdates(Items{
		{Price: 5.5, Amount: 1, ID: 13},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(t, a, 9, 54, 9)

	// purge
	a.load(nil, s, time.Now())

	// Add one at head
	err = a.insertUpdates(Items{
		{Price: 5.5, Amount: 1, ID: 13},
	}, s)
	if err != nil {
		t.Fatal(err)
	}

	Check(t, a, 1, 5.5, 1)
}

// check checks depth values after an update has taken place
func Check(t *testing.T, depth interface{}, liquidity, value float64, nodeCount int) {
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
	a.load(askSnapshot, s, time.Now())

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

	if !shiftBookmark(tip, &bookmarkedNode, nil, &Item{Amount: 1336, ID: 1337, Price: 9999}) {
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

	if shiftBookmark(tip, &nilBookmark, nil, &Item{Amount: 1336, ID: 1337, Price: 9999}) {
		t.Fatal("there should not be a bookmarked node")
	}

	if tip != nilBookmark {
		t.Fatal("nilBookmark not reassigned")
	}

	head := bookmarkedNode
	bookmarkedNode.Prev = nil
	bookmarkedNode.Next = originalBookmarkNext
	tip.Next = nil

	if !shiftBookmark(tip, &bookmarkedNode, &head, &Item{Amount: 1336, ID: 1337, Price: 9999}) {
		t.Fatal("There should be liquidity so we don't need to set tip to bookmark")
	}

	if head != originalBookmarkNext {
		t.Fatal("unexpected pointer variable")
	}
}

func TestGetMovementByBaseAmount(t *testing.T) {
	t.Parallel()
	cases := []struct {
		Name            string
		BaseAmount      float64
		ReferencePrice  float64
		BidLiquidity    Items
		ExpectedNominal float64
		ExpectedImpact  float64
		ExpectedCost    float64
		ExpectedError   error
	}{
		{
			Name:          "no amount",
			ExpectedError: errBaseAmountInvalid,
		},
		{
			Name:          "no reference price",
			BaseAmount:    1,
			ExpectedError: errInvalidReferencePrice,
		},
		{
			Name:           "not enough liquidity to service quote amount",
			BaseAmount:     1,
			ReferencePrice: 1000,
			ExpectedError:  errNoLiquidity,
		},
		{
			Name:            "thrasher test",
			BidLiquidity:    Items{{Price: 10000, Amount: 2}, {Price: 9900, Amount: 7}, {Price: 9800, Amount: 3}},
			BaseAmount:      10,
			ReferencePrice:  10000,
			ExpectedNominal: 0.8999999999999999,
			ExpectedImpact:  2,
			ExpectedCost:    900,
		},
		{
			Name:            "consume first tranche",
			BidLiquidity:    Items{{Price: 10000, Amount: 2}, {Price: 9900, Amount: 7}, {Price: 9800, Amount: 3}},
			BaseAmount:      2,
			ReferencePrice:  10000,
			ExpectedNominal: 0,
			ExpectedImpact:  1,
			ExpectedCost:    0,
		},
		{
			Name:            "consume most of first tranche",
			BidLiquidity:    Items{{Price: 10000, Amount: 2}, {Price: 9900, Amount: 7}, {Price: 9800, Amount: 3}},
			BaseAmount:      1.5,
			ReferencePrice:  10000,
			ExpectedNominal: 0,
			ExpectedImpact:  0,
			ExpectedCost:    0,
		},
		{
			Name:            "consume full liquidity",
			BidLiquidity:    Items{{Price: 10000, Amount: 2}, {Price: 9900, Amount: 7}, {Price: 9800, Amount: 3}},
			BaseAmount:      12,
			ReferencePrice:  10000,
			ExpectedNominal: 1.0833333333333395,
			ExpectedImpact:  FullLiquidityExhaustedPercentage,
			ExpectedCost:    1300,
		},
	}

	for x := range cases {
		tt := cases[x]
		t.Run(tt.Name, func(t *testing.T) {
			t.Parallel()
			depth := NewDepth(id)
			err := depth.LoadSnapshot(tt.BidLiquidity, nil, 0, time.Now(), true)
			if err != nil {
				t.Fatal(err)
			}
			movement, err := depth.bids.getMovementByBase(tt.BaseAmount, tt.ReferencePrice, false)
			if !errors.Is(err, tt.ExpectedError) {
				t.Fatalf("received: '%v' but expected: '%v'", err, tt.ExpectedError)
			}

			if movement == nil {
				return
			}
			if movement.NominalPercentage != tt.ExpectedNominal {
				t.Fatalf("nominal received: '%v' but expected: '%v'",
					movement.NominalPercentage, tt.ExpectedNominal)
			}

			if movement.ImpactPercentage != tt.ExpectedImpact {
				t.Fatalf("impact received: '%v' but expected: '%v'",
					movement.ImpactPercentage, tt.ExpectedImpact)
			}

			if movement.SlippageCost != tt.ExpectedCost {
				t.Fatalf("cost received: '%v' but expected: '%v'",
					movement.SlippageCost, tt.ExpectedCost)
			}
		})
	}
}

func TestGetBaseAmountFromNominalSlippage(t *testing.T) {
	t.Parallel()
	cases := []struct {
		Name            string
		NominalSlippage float64
		ReferencePrice  float64
		BidLiquidity    Items
		ExpectedShift   *Movement
		ExpectedError   error
	}{
		{
			Name:            "invalid slippage",
			NominalSlippage: -1,
			ExpectedError:   errInvalidNominalSlippage,
		},
		{
			Name:            "invalid slippage - larger than 100%",
			NominalSlippage: 101,
			ExpectedError:   errInvalidSlippageCannotExceed100,
		},
		{
			Name:            "no reference price",
			NominalSlippage: 1,
			ExpectedError:   errInvalidReferencePrice,
		},
		{
			Name:            "no liquidity to service quote amount",
			NominalSlippage: 1,
			ReferencePrice:  1000,
			ExpectedError:   errNoLiquidity,
		},
		{
			Name:            "thrasher test",
			BidLiquidity:    Items{{Price: 10000, Amount: 2}, {Price: 9900, Amount: 7}, {Price: 9800, Amount: 3}},
			NominalSlippage: 1,
			ReferencePrice:  10000,
			ExpectedShift: &Movement{
				Sold:              11,
				Purchased:         108900,
				AverageOrderCost:  9900,
				NominalPercentage: 1,
				StartPrice:        10000,
				EndPrice:          9800,
			},
		},
		{
			Name:            "consume first tranche - take one amount out of second",
			BidLiquidity:    Items{{Price: 10000, Amount: 2}, {Price: 9900, Amount: 7}, {Price: 9800, Amount: 3}},
			NominalSlippage: 0.33333333333334,
			ReferencePrice:  10000,
			ExpectedShift: &Movement{
				Sold:              3.0000000000000275, // <- expected rounding issue
				Purchased:         29900.00000000027,
				AverageOrderCost:  9966.666666666664,
				NominalPercentage: 0.33333333333334,
				StartPrice:        10000,
				EndPrice:          9900,
			},
		},
		{
			Name:            "consume full liquidity",
			BidLiquidity:    Items{{Price: 10000, Amount: 2}, {Price: 9900, Amount: 7}, {Price: 9800, Amount: 3}},
			NominalSlippage: 10,
			ReferencePrice:  10000,
			ExpectedShift: &Movement{
				Sold:                 12,
				Purchased:            118700,
				AverageOrderCost:     9891.666666666666,
				NominalPercentage:    1.0833333333333395,
				StartPrice:           10000,
				EndPrice:             9800,
				FullBookSideConsumed: true,
			},
		},
		{
			Name:            "scotts lovely slippery slippage requirements",
			BidLiquidity:    Items{{Price: 10000, Amount: 2}, {Price: 9900, Amount: 7}, {Price: 9800, Amount: 3}},
			NominalSlippage: 0.00000000000000000000000000000000000000000000000000000000000000000000000000000000001,
			ReferencePrice:  10000,
			ExpectedShift: &Movement{
				Sold:             2,
				Purchased:        20000,
				AverageOrderCost: 10000,
				StartPrice:       10000,
				EndPrice:         10000,
			},
		},
	}

	for x := range cases {
		tt := cases[x]
		t.Run(tt.Name, func(t *testing.T) {
			t.Parallel()
			depth := NewDepth(id)
			err := depth.LoadSnapshot(tt.BidLiquidity, nil, 0, time.Now(), true)
			if err != nil {
				t.Fatal(err)
			}
			base, err := depth.bids.hitBidsByNominalSlippage(tt.NominalSlippage, tt.ReferencePrice)
			if !errors.Is(err, tt.ExpectedError) {
				t.Fatalf("%s received: '%v' but expected: '%v'",
					tt.Name, err, tt.ExpectedError)
			}
			if !base.IsEqual(tt.ExpectedShift) {
				t.Fatalf("%s quote received: '%+v' but expected: '%+v'",
					tt.Name, base, tt.ExpectedShift)
			}
		})
	}
}

// IsEqual is a tester function for comparison.
func (m *Movement) IsEqual(that *Movement) bool {
	if m == nil || that == nil {
		return m == nil && that == nil
	}
	return m.FullBookSideConsumed == that.FullBookSideConsumed &&
		m.Sold == that.Sold &&
		m.Purchased == that.Purchased &&
		m.NominalPercentage == that.NominalPercentage &&
		m.ImpactPercentage == that.ImpactPercentage &&
		m.EndPrice == that.EndPrice &&
		m.StartPrice == that.StartPrice &&
		m.AverageOrderCost == that.AverageOrderCost
}

func TestGetBaseAmountFromImpact(t *testing.T) {
	t.Parallel()
	cases := []struct {
		Name           string
		ImpactSlippage float64
		ReferencePrice float64
		BidLiquidity   Items
		ExpectedShift  *Movement
		ExpectedError  error
	}{
		{
			Name:          "invalid slippage",
			ExpectedError: errInvalidImpactSlippage,
		},
		{
			Name:           "invalid slippage - exceed 100%",
			ImpactSlippage: 101,
			ExpectedError:  errInvalidSlippageCannotExceed100,
		},
		{
			Name:           "no reference price",
			ImpactSlippage: 1,
			ExpectedError:  errInvalidReferencePrice,
		},
		{
			Name:           "no liquidity",
			ImpactSlippage: 1,
			ReferencePrice: 10000,
			ExpectedError:  errNoLiquidity,
		},
		{
			Name:           "thrasher test",
			BidLiquidity:   Items{{Price: 10000, Amount: 2}, {Price: 9900, Amount: 7}, {Price: 9800, Amount: 3}},
			ImpactSlippage: 1,
			ReferencePrice: 10000,
			ExpectedShift: &Movement{
				Sold:             2,
				Purchased:        20000,
				ImpactPercentage: 1,
				AverageOrderCost: 10000,
				StartPrice:       10000,
				EndPrice:         9900,
			},
		},
		{
			Name:           "consume first tranche and second tranche",
			BidLiquidity:   Items{{Price: 10000, Amount: 2}, {Price: 9900, Amount: 7}, {Price: 9800, Amount: 3}},
			ImpactSlippage: 2,
			ReferencePrice: 10000,
			ExpectedShift: &Movement{
				Sold:             9,
				Purchased:        89300,
				AverageOrderCost: 9922.222222222223,
				ImpactPercentage: 2,
				StartPrice:       10000,
				EndPrice:         9800,
			},
		},
		{
			Name:           "consume full liquidity",
			BidLiquidity:   Items{{Price: 10000, Amount: 2}, {Price: 9900, Amount: 7}, {Price: 9800, Amount: 3}},
			ImpactSlippage: 10,
			ReferencePrice: 10000,
			ExpectedShift: &Movement{
				Sold:                 12,
				Purchased:            118700,
				ImpactPercentage:     FullLiquidityExhaustedPercentage,
				AverageOrderCost:     9891.666666666666,
				StartPrice:           10000,
				EndPrice:             9800,
				FullBookSideConsumed: true,
			},
		},
	}

	for x := range cases {
		tt := cases[x]
		t.Run(tt.Name, func(t *testing.T) {
			t.Parallel()
			depth := NewDepth(id)
			err := depth.LoadSnapshot(tt.BidLiquidity, nil, 0, time.Now(), true)
			if err != nil {
				t.Fatal(err)
			}
			base, err := depth.bids.hitBidsByImpactSlippage(tt.ImpactSlippage, tt.ReferencePrice)
			if !errors.Is(err, tt.ExpectedError) {
				t.Fatalf("%s received: '%v' but expected: '%v'", tt.Name, err, tt.ExpectedError)
			}
			if !base.IsEqual(tt.ExpectedShift) {
				t.Fatalf("%s quote received: '%+v' but expected: '%+v'",
					tt.Name, base, tt.ExpectedShift)
			}
		})
	}
}

func TestGetMovementByQuoteAmount(t *testing.T) {
	t.Parallel()
	cases := []struct {
		Name            string
		QuoteAmount     float64
		ReferencePrice  float64
		AskLiquidity    Items
		ExpectedNominal float64
		ExpectedImpact  float64
		ExpectedCost    float64
		ExpectedError   error
	}{
		{
			Name:          "no amount",
			ExpectedError: errQuoteAmountInvalid,
		},
		{
			Name:          "no reference price",
			QuoteAmount:   1,
			ExpectedError: errInvalidReferencePrice,
		},
		{
			Name:           "not enough liquidity to service quote amount",
			QuoteAmount:    1,
			ReferencePrice: 1000,
			ExpectedError:  errNoLiquidity,
		},
		{
			Name:            "thrasher test",
			AskLiquidity:    Items{{Price: 10000, Amount: 2}, {Price: 10100, Amount: 7}, {Price: 10200, Amount: 3}},
			QuoteAmount:     100900,
			ReferencePrice:  10000,
			ExpectedNominal: 0.8999999999999999,
			ExpectedImpact:  2,
			ExpectedCost:    900,
		},
		{
			Name:            "consume first tranche",
			AskLiquidity:    Items{{Price: 10000, Amount: 2}, {Price: 10100, Amount: 7}, {Price: 10200, Amount: 3}},
			QuoteAmount:     20000,
			ReferencePrice:  10000,
			ExpectedNominal: 0,
			ExpectedImpact:  1,
			ExpectedCost:    0,
		},
		{
			Name:            "consume most of first tranche",
			AskLiquidity:    Items{{Price: 10000, Amount: 2}, {Price: 10100, Amount: 7}, {Price: 10200, Amount: 3}},
			QuoteAmount:     15000,
			ReferencePrice:  10000,
			ExpectedNominal: 0,
			ExpectedImpact:  0,
			ExpectedCost:    0,
		},
		{
			Name:            "consume full liquidity",
			AskLiquidity:    Items{{Price: 10000, Amount: 2}, {Price: 10100, Amount: 7}, {Price: 10200, Amount: 3}},
			QuoteAmount:     121300,
			ReferencePrice:  10000,
			ExpectedNominal: 1.0833333333333395,
			ExpectedImpact:  FullLiquidityExhaustedPercentage,
			ExpectedCost:    1300,
		},
	}

	for x := range cases {
		tt := cases[x]
		t.Run(tt.Name, func(t *testing.T) {
			t.Parallel()
			depth := NewDepth(id)
			err := depth.LoadSnapshot(nil, tt.AskLiquidity, 0, time.Now(), true)
			if err != nil {
				t.Fatal(err)
			}
			movement, err := depth.asks.getMovementByQuotation(tt.QuoteAmount, tt.ReferencePrice, false)
			if !errors.Is(err, tt.ExpectedError) {
				t.Fatalf("received: '%v' but expected: '%v'", err, tt.ExpectedError)
			}

			if movement == nil {
				return
			}
			if movement.NominalPercentage != tt.ExpectedNominal {
				t.Fatalf("nominal received: '%v' but expected: '%v'",
					movement.NominalPercentage, tt.ExpectedNominal)
			}

			if movement.ImpactPercentage != tt.ExpectedImpact {
				t.Fatalf("impact received: '%v' but expected: '%v'",
					movement.ImpactPercentage, tt.ExpectedImpact)
			}

			if movement.SlippageCost != tt.ExpectedCost {
				t.Fatalf("cost received: '%v' but expected: '%v'",
					movement.SlippageCost, tt.ExpectedCost)
			}
		})
	}
}

func TestGetQuoteAmountFromNominalSlippage(t *testing.T) {
	t.Parallel()
	cases := []struct {
		Name            string
		NominalSlippage float64
		ReferencePrice  float64
		AskLiquidity    Items
		ExpectedShift   *Movement
		ExpectedError   error
	}{
		{
			Name:            "invalid slippage",
			NominalSlippage: -1,
			ExpectedError:   errInvalidNominalSlippage,
		},
		{
			Name:            "no reference price",
			NominalSlippage: 1,
			ExpectedError:   errInvalidReferencePrice,
		},
		{
			Name:            "no liquidity",
			NominalSlippage: 1,
			ReferencePrice:  10000,
			ExpectedError:   errNoLiquidity,
		},
		{
			Name:            "consume first tranche - one amount on second tranche",
			AskLiquidity:    Items{{Price: 10000, Amount: 2}, {Price: 10100, Amount: 7}, {Price: 10200, Amount: 3}},
			NominalSlippage: 0.33333333333334,
			ReferencePrice:  10000,
			ExpectedShift: &Movement{
				Sold:              30100.000000000276, // <- expected rounding issue
				Purchased:         3.0000000000000275,
				AverageOrderCost:  10033.333333333333333333333333333,
				NominalPercentage: 0.33333333333334,
				StartPrice:        10000,
				EndPrice:          10100,
			},
		},
		{
			Name:            "last tranche total agg meeting 1 percent nominally",
			AskLiquidity:    Items{{Price: 10000, Amount: 2}, {Price: 10100, Amount: 7}, {Price: 10200, Amount: 3}},
			NominalSlippage: 1,
			ReferencePrice:  10000,
			ExpectedShift: &Movement{
				Sold:              111100,
				Purchased:         11,
				AverageOrderCost:  10100,
				NominalPercentage: 1,
				StartPrice:        10000,
				EndPrice:          10200,
			},
		},
		{
			Name:            "take full second tranche",
			AskLiquidity:    Items{{Price: 10000, Amount: 2}, {Price: 10100, Amount: 7}, {Price: 10200, Amount: 3}},
			NominalSlippage: 0.7777777777777738,
			ReferencePrice:  10000,
			ExpectedShift: &Movement{
				Sold:              90700,
				Purchased:         9,
				AverageOrderCost:  10077.777777777777,
				NominalPercentage: 0.7777777777777738,
				StartPrice:        10000,
				EndPrice:          10100,
			},
		},
		{
			Name:            "consume full liquidity",
			AskLiquidity:    Items{{Price: 10000, Amount: 2}, {Price: 10100, Amount: 7}, {Price: 10200, Amount: 3}},
			NominalSlippage: 10,
			ReferencePrice:  10000,
			ExpectedShift: &Movement{
				Sold:                 121300,
				Purchased:            12,
				AverageOrderCost:     10108.333333333334,
				NominalPercentage:    1.0833333333333395,
				StartPrice:           10000,
				EndPrice:             10200,
				FullBookSideConsumed: true,
			},
		},
		{
			Name:            "scotts lovely slippery slippage requirements",
			AskLiquidity:    Items{{Price: 10000, Amount: 2}, {Price: 10100, Amount: 7}, {Price: 10200, Amount: 3}},
			NominalSlippage: 0.00000000000000000000000000000000000000000000000000000000000000000000000000000000001,
			ReferencePrice:  10000,
			ExpectedShift: &Movement{
				Sold:             20000,
				Purchased:        2,
				AverageOrderCost: 10000,
				StartPrice:       10000,
				EndPrice:         10000,
			},
		},
	}

	for x := range cases {
		tt := cases[x]
		t.Run(tt.Name, func(t *testing.T) {
			t.Parallel()
			depth := NewDepth(id)
			err := depth.LoadSnapshot(nil, tt.AskLiquidity, 0, time.Now(), true)
			if err != nil {
				t.Fatalf("failed to load snapshot: %s", err)
			}
			quote, err := depth.asks.liftAsksByNominalSlippage(tt.NominalSlippage, tt.ReferencePrice)
			if !errors.Is(err, tt.ExpectedError) {
				t.Fatalf("%s received: '%v' but expected: '%v'", tt.Name, err, tt.ExpectedError)
			}
			if !quote.IsEqual(tt.ExpectedShift) {
				t.Fatalf("%s quote received: \n'%+v' \nbut expected: \n'%+v'",
					tt.Name, quote, tt.ExpectedShift)
			}
		})
	}
}

func TestGetQuoteAmountFromImpact(t *testing.T) {
	t.Parallel()
	cases := []struct {
		Name           string
		ImpactSlippage float64
		ReferencePrice float64
		AskLiquidity   Items
		ExpectedShift  *Movement
		ExpectedError  error
	}{
		{
			Name:           "invalid slippage",
			ImpactSlippage: -1,
			ExpectedError:  errInvalidImpactSlippage,
		},
		{
			Name:           "no reference price",
			ImpactSlippage: 1,
			ExpectedError:  errInvalidReferencePrice,
		},
		{
			Name:           "no liquidity",
			ImpactSlippage: 1,
			ReferencePrice: 1000,
			ExpectedError:  errNoLiquidity,
		},
		{
			Name:           "thrasher test",
			AskLiquidity:   Items{{Price: 10000, Amount: 2}, {Price: 10100, Amount: 7}, {Price: 10200, Amount: 3}},
			ImpactSlippage: 1,
			ReferencePrice: 10000,
			ExpectedShift: &Movement{
				Sold:             20000,
				Purchased:        2,
				AverageOrderCost: 10000,
				ImpactPercentage: 1,
				StartPrice:       10000,
				EndPrice:         10100,
			},
		},
		{
			Name:           "consume first tranche and second tranche",
			AskLiquidity:   Items{{Price: 10000, Amount: 2}, {Price: 10100, Amount: 7}, {Price: 10200, Amount: 3}},
			ImpactSlippage: 2,
			ReferencePrice: 10000,
			ExpectedShift: &Movement{
				Sold:             90700,
				Purchased:        9,
				AverageOrderCost: 10077.777777777777777777777777778,
				ImpactPercentage: 2,
				StartPrice:       10000,
				EndPrice:         10200,
			},
		},
		{
			Name:           "consume full liquidity",
			AskLiquidity:   Items{{Price: 10000, Amount: 2}, {Price: 10100, Amount: 7}, {Price: 10200, Amount: 3}},
			ImpactSlippage: 10,
			ReferencePrice: 10000,
			ExpectedShift: &Movement{
				Sold:                 121300,
				Purchased:            12,
				AverageOrderCost:     10108.333333333333333333333333333,
				ImpactPercentage:     FullLiquidityExhaustedPercentage,
				StartPrice:           10000,
				EndPrice:             10200,
				FullBookSideConsumed: true,
			},
		},
	}

	for x := range cases {
		tt := cases[x]
		t.Run(tt.Name, func(t *testing.T) {
			t.Parallel()
			depth := NewDepth(id)
			err := depth.LoadSnapshot(nil, tt.AskLiquidity, 0, time.Now(), true)
			if err != nil {
				t.Fatalf("failed to load snapshot: %s", err)
			}
			quote, err := depth.asks.liftAsksByImpactSlippage(tt.ImpactSlippage, tt.ReferencePrice)
			if !errors.Is(err, tt.ExpectedError) {
				t.Fatalf("received: '%v' but expected: '%v'", err, tt.ExpectedError)
			}
			if !quote.IsEqual(tt.ExpectedShift) {
				t.Fatalf("%s quote received: '%+v' but expected: '%+v'",
					tt.Name, quote, tt.ExpectedShift)
			}
		})
	}
}

func TestGetHeadPrice(t *testing.T) {
	t.Parallel()
	depth := NewDepth(id)
	if _, err := depth.bids.getHeadPriceNoLock(); !errors.Is(err, errNoLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoLiquidity)
	}
	if _, err := depth.asks.getHeadPriceNoLock(); !errors.Is(err, errNoLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoLiquidity)
	}
	err := depth.LoadSnapshot(bid, ask, 0, time.Now(), true)
	if err != nil {
		t.Fatalf("failed to load snapshot: %s", err)
	}

	val, err := depth.bids.getHeadPriceNoLock()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if val != 1336 {
		t.Fatal("unexpected value")
	}

	val, err = depth.asks.getHeadPriceNoLock()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if val != 1337 {
		t.Fatal("unexpected value", val)
	}
}

func TestFinalizeFields(t *testing.T) {
	m := &Movement{}
	_, err := m.finalizeFields(0, 0, 0, 0, false)
	if !errors.Is(err, errInvalidCost) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidCost)
	}
	_, err = m.finalizeFields(1, 0, 0, 0, false)
	if !errors.Is(err, errInvalidAmount) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidAmount)
	}
	_, err = m.finalizeFields(1, 1, 0, 0, false)
	if !errors.Is(err, errInvalidHeadPrice) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidHeadPrice)
	}

	// Test slippage as per https://en.wikipedia.org/wiki/Slippage_(finance)
	mov, err := m.finalizeFields(20000*151.11585, 20000, 151.08, 0, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	// These tests demonstrate the imprecision of relying on floating point numbers
	// That a different OS will return different numbers: macOS: `716.9999999997499` vs '716.9999999995343'
	// speed is important, but having tests look for exact floating point numbers shows that one
	// could have a different impact simply from running it on a different computer
	if mov.SlippageCost != 716.9999999995343 {
		t.Fatalf("received: '%v' but expected: '%v'", mov.SlippageCost, 716.9999999995343)
	}
}
