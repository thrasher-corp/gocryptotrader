package orderbook

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var ask = Tranches{
	{Price: 1337, Amount: 1},
	{Price: 1338, Amount: 1},
	{Price: 1339, Amount: 1},
	{Price: 1340, Amount: 1},
	{Price: 1341, Amount: 1},
	{Price: 1342, Amount: 1},
	{Price: 1343, Amount: 1},
	{Price: 1344, Amount: 1},
	{Price: 1345, Amount: 1},
	{Price: 1346, Amount: 1},
	{Price: 1347, Amount: 1},
	{Price: 1348, Amount: 1},
	{Price: 1349, Amount: 1},
	{Price: 1350, Amount: 1},
	{Price: 1351, Amount: 1},
	{Price: 1352, Amount: 1},
	{Price: 1353, Amount: 1},
	{Price: 1354, Amount: 1},
	{Price: 1355, Amount: 1},
	{Price: 1356, Amount: 1},
}

var bid = Tranches{
	{Price: 1336, Amount: 1},
	{Price: 1335, Amount: 1},
	{Price: 1334, Amount: 1},
	{Price: 1333, Amount: 1},
	{Price: 1332, Amount: 1},
	{Price: 1331, Amount: 1},
	{Price: 1330, Amount: 1},
	{Price: 1329, Amount: 1},
	{Price: 1328, Amount: 1},
	{Price: 1327, Amount: 1},
	{Price: 1326, Amount: 1},
	{Price: 1325, Amount: 1},
	{Price: 1324, Amount: 1},
	{Price: 1323, Amount: 1},
	{Price: 1322, Amount: 1},
	{Price: 1321, Amount: 1},
	{Price: 1320, Amount: 1},
	{Price: 1319, Amount: 1},
	{Price: 1318, Amount: 1},
	{Price: 1317, Amount: 1},
}

// Display displays depth content for tests
func (ts Tranches) display() {
	for x := range ts {
		fmt.Printf("Tranche: %+v %p \n", ts[x], &ts[x])
	}
	fmt.Println()
}

func TestLoad(t *testing.T) {
	list := askTranches{}
	Check(t, list, 0, 0, 0)

	list.load(Tranches{
		{Price: 1, Amount: 1},
		{Price: 3, Amount: 1},
		{Price: 5, Amount: 1},
		{Price: 7, Amount: 1},
		{Price: 9, Amount: 1},
		{Price: 11, Amount: 1},
	})

	Check(t, list, 6, 36, 6)

	list.load(Tranches{
		{Price: 1, Amount: 1},
		{Price: 3, Amount: 1},
		{Price: 5, Amount: 1},
	})

	Check(t, list, 3, 9, 3)

	list.load(Tranches{
		{Price: 1, Amount: 1},
		{Price: 3, Amount: 1},
		{Price: 5, Amount: 1},
		{Price: 7, Amount: 1},
	})

	Check(t, list, 4, 16, 4)

	// purge entire list
	list.load(nil)
	Check(t, list, 0, 0, 0)
}

// 27906781	        42.4 ns/op	       0 B/op	       0 allocs/op (old)
// 84119028	        13.87 ns/op	       0 B/op	       0 allocs/op (new)
func BenchmarkLoad(b *testing.B) {
	ts := Tranches{}
	for i := 0; i < b.N; i++ {
		ts.load(ask)
	}
}

func TestUpdateInsertByPrice(t *testing.T) {
	a := askTranches{}
	asksSnapshot := Tranches{
		{Price: 1, Amount: 1},
		{Price: 3, Amount: 1},
		{Price: 5, Amount: 1},
		{Price: 7, Amount: 1},
		{Price: 9, Amount: 1},
		{Price: 11, Amount: 1},
	}
	a.load(asksSnapshot)

	// Update one instance with matching price
	a.updateInsertByPrice(Tranches{{Price: 1, Amount: 2}}, 0)

	Check(t, a, 7, 37, 6)

	// Insert at head
	a.updateInsertByPrice(Tranches{
		{Price: 0.5, Amount: 2},
	}, 0)

	Check(t, a, 9, 38, 7)

	// Insert at tail
	a.updateInsertByPrice(Tranches{
		{Price: 12, Amount: 2},
	}, 0)

	Check(t, a, 11, 62, 8)

	// Insert between price and up to and beyond max allowable depth level
	a.updateInsertByPrice(Tranches{
		{Price: 11.5, Amount: 2},
		{Price: 10.5, Amount: 2},
		{Price: 13, Amount: 2},
	}, 10)

	Check(t, a, 15, 106, 10)

	// delete at tail
	a.updateInsertByPrice(Tranches{{Price: 12, Amount: 0}}, 0)

	Check(t, a, 13, 82, 9)

	// delete at mid
	a.updateInsertByPrice(Tranches{{Price: 7, Amount: 0}}, 0)

	Check(t, a, 12, 75, 8)

	// delete at head
	a.updateInsertByPrice(Tranches{{Price: 0.5, Amount: 0}}, 0)

	Check(t, a, 10, 74, 7)

	// purge if liquidity plunges to zero
	a.load(nil)

	// rebuild everything again
	a.updateInsertByPrice(Tranches{
		{Price: 1, Amount: 1},
		{Price: 3, Amount: 1},
		{Price: 5, Amount: 1},
		{Price: 7, Amount: 1},
		{Price: 9, Amount: 1},
		{Price: 11, Amount: 1},
	}, 0)

	Check(t, a, 6, 36, 6)

	b := bidTranches{}
	bidsSnapshot := Tranches{
		{Price: 11, Amount: 1},
		{Price: 9, Amount: 1},
		{Price: 7, Amount: 1},
		{Price: 5, Amount: 1},
		{Price: 3, Amount: 1},
		{Price: 1, Amount: 1},
	}
	b.load(bidsSnapshot)

	// Update one instance with matching price
	b.updateInsertByPrice(Tranches{{Price: 11, Amount: 2}}, 0)

	Check(t, b, 7, 47, 6)

	// Insert at head
	b.updateInsertByPrice(Tranches{{Price: 12, Amount: 2}}, 0)

	Check(t, b, 9, 71, 7)

	// Insert at tail
	b.updateInsertByPrice(Tranches{{Price: 0.5, Amount: 2}}, 0)

	Check(t, b, 11, 72, 8)

	// Insert between price and up to and beyond max allowable depth level
	b.updateInsertByPrice(Tranches{
		{Price: 11.5, Amount: 2},
		{Price: 10.5, Amount: 2},
		{Price: 13, Amount: 2},
	}, 10)

	Check(t, b, 15, 141, 10)

	// Insert between price and up to and beyond max allowable depth level
	b.updateInsertByPrice(Tranches{{Price: 1, Amount: 0}}, 0)

	Check(t, b, 14, 140, 9)

	// delete at mid
	b.updateInsertByPrice(Tranches{{Price: 10.5, Amount: 0}}, 0)

	Check(t, b, 12, 119, 8)

	// delete at head
	b.updateInsertByPrice(Tranches{{Price: 13, Amount: 0}}, 0)

	Check(t, b, 10, 93, 7)

	// purge if liquidity plunges to zero
	b.load(nil)

	// rebuild everything again
	b.updateInsertByPrice(Tranches{
		{Price: 1, Amount: 1},
		{Price: 3, Amount: 1},
		{Price: 5, Amount: 1},
		{Price: 7, Amount: 1},
		{Price: 9, Amount: 1},
		{Price: 11, Amount: 1},
	}, 0)

	Check(t, b, 6, 36, 6)
}

// 134830672	         9.83 ns/op	       0 B/op	       0 allocs/op (old)
// 206689897	         5.761 ns/op	   0 B/op	       0 allocs/op (new)
func BenchmarkUpdateInsertByPrice_Amend(b *testing.B) {
	a := askTranches{}
	a.load(ask)

	updates := Tranches{
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
		a.updateInsertByPrice(updates, 0)
	}
}

// 49763002	        24.9 ns/op	       0 B/op	       0 allocs/op (old)
// 25662849	        45.32 ns/op	       0 B/op	       0 allocs/op (new)
func BenchmarkUpdateInsertByPrice_Insert_Delete(b *testing.B) {
	a := askTranches{}

	a.load(ask)

	updates := Tranches{
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
		a.updateInsertByPrice(updates, 0)
	}
}

func TestUpdateByID(t *testing.T) {
	a := askTranches{}
	asksSnapshot := Tranches{
		{Price: 1, Amount: 1, ID: 1},
		{Price: 3, Amount: 1, ID: 3},
		{Price: 5, Amount: 1, ID: 5},
		{Price: 7, Amount: 1, ID: 7},
		{Price: 9, Amount: 1, ID: 9},
		{Price: 11, Amount: 1, ID: 11},
	}
	a.load(asksSnapshot)

	err := a.updateByID(Tranches{
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

	err = a.updateByID(Tranches{
		{Price: 11, Amount: 1, ID: 1337},
	})
	if !errors.Is(err, errIDCannotBeMatched) {
		t.Fatalf("expecting %s but received %v", errIDCannotBeMatched, err)
	}

	err = a.updateByID(Tranches{ // Simulate Bitmex updating
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

// 46043871	        25.9 ns/op	       0 B/op	       0 allocs/op (old)
// 63445401	        18.51 ns/op	       0 B/op	       0 allocs/op (new)
func BenchmarkUpdateByID(b *testing.B) {
	asks := Tranches{}
	asksSnapshot := Tranches{
		{Price: 1, Amount: 1, ID: 1},
		{Price: 3, Amount: 1, ID: 3},
		{Price: 5, Amount: 1, ID: 5},
		{Price: 7, Amount: 1, ID: 7},
		{Price: 9, Amount: 1, ID: 9},
		{Price: 11, Amount: 1, ID: 11},
	}
	asks.load(asksSnapshot)

	for i := 0; i < b.N; i++ {
		err := asks.updateByID(asksSnapshot)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestDeleteByID(t *testing.T) {
	a := askTranches{}
	asksSnapshot := Tranches{
		{Price: 1, Amount: 1, ID: 1},
		{Price: 3, Amount: 1, ID: 3},
		{Price: 5, Amount: 1, ID: 5},
		{Price: 7, Amount: 1, ID: 7},
		{Price: 9, Amount: 1, ID: 9},
		{Price: 11, Amount: 1, ID: 11},
	}
	a.load(asksSnapshot)

	// Delete at head
	err := a.deleteByID(Tranches{{Price: 1, Amount: 1, ID: 1}}, false)
	if err != nil {
		t.Fatal(err)
	}

	Check(t, a, 5, 35, 5)

	// Delete at tail
	err = a.deleteByID(Tranches{{Price: 1, Amount: 1, ID: 11}}, false)
	if err != nil {
		t.Fatal(err)
	}

	Check(t, a, 4, 24, 4)

	// Delete in middle
	err = a.deleteByID(Tranches{{Price: 1, Amount: 1, ID: 5}}, false)
	if err != nil {
		t.Fatal(err)
	}

	Check(t, a, 3, 19, 3)

	// Intentional error
	err = a.deleteByID(Tranches{{Price: 11, Amount: 1, ID: 1337}}, false)
	if !errors.Is(err, errIDCannotBeMatched) {
		t.Fatalf("expecting %s but received %v", errIDCannotBeMatched, err)
	}

	// Error bypass
	err = a.deleteByID(Tranches{{Price: 11, Amount: 1, ID: 1337}}, true)
	if err != nil {
		t.Fatal(err)
	}
}

// 26724331	        44.69 ns/op	       0 B/op	       0 allocs/op
func BenchmarkDeleteByID(b *testing.B) {
	asks := Tranches{}
	asksSnapshot := Tranches{
		{Price: 1, Amount: 1, ID: 1},
		{Price: 3, Amount: 1, ID: 3},
		{Price: 5, Amount: 1, ID: 5},
		{Price: 7, Amount: 1, ID: 7},
		{Price: 9, Amount: 1, ID: 9},
		{Price: 11, Amount: 1, ID: 11},
	}
	asks.load(asksSnapshot)

	for i := 0; i < b.N; i++ {
		err := asks.deleteByID(asksSnapshot, false)
		if err != nil {
			b.Fatal(err)
		}
		asks.load(asksSnapshot) // reset
	}
}

func TestUpdateInsertByIDAsk(t *testing.T) {
	a := askTranches{}
	asksSnapshot := Tranches{
		{Price: 1, Amount: 1, ID: 1},
		{Price: 3, Amount: 1, ID: 3},
		{Price: 5, Amount: 1, ID: 5},
		{Price: 7, Amount: 1, ID: 7},
		{Price: 9, Amount: 1, ID: 9},
		{Price: 11, Amount: 1, ID: 11},
	}
	a.load(asksSnapshot)

	// Update one instance with matching ID
	err := a.updateInsertByID(Tranches{{Price: 1, Amount: 2, ID: 1}})
	if err != nil {
		t.Fatal(err)
	}

	Check(t, a, 7, 37, 6)

	// Reset
	a.load(asksSnapshot)

	// Update all instances with matching ID in order
	err = a.updateInsertByID(Tranches{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 5, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	})
	if err != nil {
		t.Fatal(err)
	}

	Check(t, a, 12, 72, 6)

	// Update all instances with matching ID in backwards
	err = a.updateInsertByID(Tranches{
		{Price: 11, Amount: 2, ID: 11},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 5, Amount: 2, ID: 5},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 1, Amount: 2, ID: 1},
	})
	if err != nil {
		t.Fatal(err)
	}

	Check(t, a, 12, 72, 6)

	// Update all instances with matching ID all over the ship
	err = a.updateInsertByID(Tranches{
		{Price: 11, Amount: 2, ID: 11},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 1, Amount: 2, ID: 1},
		{Price: 5, Amount: 2, ID: 5},
	})
	if err != nil {
		t.Fatal(err)
	}

	Check(t, a, 12, 72, 6)

	// Update all instances move one before ID in middle
	err = a.updateInsertByID(Tranches{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 2, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	})
	if err != nil {
		t.Fatal(err)
	}

	Check(t, a, 12, 66, 6)

	// Update all instances move one before ID at head
	err = a.updateInsertByID(Tranches{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: .5, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	})
	if err != nil {
		t.Fatal(err)
	}

	Check(t, a, 12, 63, 6)

	// Reset
	a.load(asksSnapshot)

	// Update all instances move one after ID
	err = a.updateInsertByID(Tranches{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 8, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	})
	if err != nil {
		t.Fatal(err)
	}

	Check(t, a, 12, 78, 6)

	// Reset
	a.load(asksSnapshot)

	// Update all instances move one after ID to tail
	err = a.updateInsertByID(Tranches{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 12, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	})
	if err != nil {
		t.Fatal(err)
	}

	Check(t, a, 12, 86, 6)

	// Update all instances then pop new instance
	err = a.updateInsertByID(Tranches{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 12, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
		{Price: 10, Amount: 2, ID: 10},
	})
	if err != nil {
		t.Fatal(err)
	}

	Check(t, a, 14, 106, 7)

	// Reset
	a.load(asksSnapshot)

	// Update all instances pop at head
	err = a.updateInsertByID(Tranches{
		{Price: 0.5, Amount: 2, ID: 0},
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 12, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	})
	if err != nil {
		t.Fatal(err)
	}

	Check(t, a, 14, 87, 7)

	// bookmark head and move to mid
	err = a.updateInsertByID(Tranches{{Price: 7.5, Amount: 2, ID: 0}})
	if err != nil {
		t.Fatal(err)
	}

	Check(t, a, 14, 101, 7)

	// bookmark head and move to tail
	err = a.updateInsertByID(Tranches{{Price: 12.5, Amount: 2, ID: 1}})
	if err != nil {
		t.Fatal(err)
	}

	Check(t, a, 14, 124, 7)

	// move tail location to head
	err = a.updateInsertByID(Tranches{{Price: 2.5, Amount: 2, ID: 1}})
	if err != nil {
		t.Fatal(err)
	}

	Check(t, a, 14, 104, 7)

	// move tail location to mid
	err = a.updateInsertByID(Tranches{{Price: 8, Amount: 2, ID: 5}})
	if err != nil {
		t.Fatal(err)
	}

	Check(t, a, 14, 96, 7)

	// insert at tail dont match
	err = a.updateInsertByID(Tranches{{Price: 30, Amount: 2, ID: 1234}})
	if err != nil {
		t.Fatal(err)
	}

	Check(t, a, 16, 156, 8)

	// insert between last and 2nd last
	err = a.updateInsertByID(Tranches{{Price: 12, Amount: 2, ID: 12345}})
	if err != nil {
		t.Fatal(err)
	}

	Check(t, a, 18, 180, 9)

	// readjust at end
	err = a.updateInsertByID(Tranches{{Price: 29, Amount: 3, ID: 1234}})
	if err != nil {
		t.Fatal(err)
	}

	Check(t, a, 19, 207, 9)

	// readjust further and decrease price past tail
	err = a.updateInsertByID(Tranches{{Price: 31, Amount: 3, ID: 1234}})
	if err != nil {
		t.Fatal(err)
	}

	Check(t, a, 19, 213, 9)

	// purge
	a.load(nil)

	// insert with no liquidity and jumbled
	err = a.updateInsertByID(Tranches{
		{Price: 11, Amount: 2, ID: 11},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 0.5, Amount: 2, ID: 0},
		{Price: 12, Amount: 2, ID: 5},
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
	})
	if err != nil {
		t.Fatal(err)
	}

	Check(t, a, 14, 87, 7)
}

// 21614455	        81.74 ns/op	       0 B/op	       0 allocs/op
func BenchmarkUpdateInsertByID_asks(b *testing.B) {
	asks := Tranches{}
	asksSnapshot := Tranches{
		{Price: 1, Amount: 1, ID: 1},
		{Price: 3, Amount: 1, ID: 3},
		{Price: 5, Amount: 1, ID: 5},
		{Price: 7, Amount: 1, ID: 7},
		{Price: 9, Amount: 1, ID: 9},
		{Price: 11, Amount: 1, ID: 11},
	}
	asks.load(asksSnapshot)

	for i := 0; i < b.N; i++ {
		err := asks.updateInsertByID(asksSnapshot, askCompare)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestUpdateInsertByIDBids(t *testing.T) {
	b := bidTranches{}
	bidsSnapshot := Tranches{
		{Price: 11, Amount: 1, ID: 11},
		{Price: 9, Amount: 1, ID: 9},
		{Price: 7, Amount: 1, ID: 7},
		{Price: 5, Amount: 1, ID: 5},
		{Price: 3, Amount: 1, ID: 3},
		{Price: 1, Amount: 1, ID: 1},
	}
	b.load(bidsSnapshot)

	// Update one instance with matching ID
	err := b.updateInsertByID(Tranches{{Price: 1, Amount: 2, ID: 1}})
	if err != nil {
		t.Fatal(err)
	}

	Check(t, b, 7, 37, 6)

	// Reset
	b.load(bidsSnapshot)

	// Update all instances with matching ID in order
	err = b.updateInsertByID(Tranches{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 5, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	})
	if err != nil {
		t.Fatal(err)
	}

	Check(t, b, 12, 72, 6)

	// Update all instances with matching ID in backwards
	err = b.updateInsertByID(Tranches{
		{Price: 11, Amount: 2, ID: 11},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 5, Amount: 2, ID: 5},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 1, Amount: 2, ID: 1},
	})
	if err != nil {
		t.Fatal(err)
	}

	Check(t, b, 12, 72, 6)

	// Update all instances with matching ID all over the ship
	err = b.updateInsertByID(Tranches{
		{Price: 11, Amount: 2, ID: 11},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 1, Amount: 2, ID: 1},
		{Price: 5, Amount: 2, ID: 5},
	})
	if err != nil {
		t.Fatal(err)
	}

	Check(t, b, 12, 72, 6)

	// Update all instances move one before ID in middle
	err = b.updateInsertByID(Tranches{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 2, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	})
	if err != nil {
		t.Fatal(err)
	}

	Check(t, b, 12, 66, 6)

	// Update all instances move one before ID at head
	err = b.updateInsertByID(Tranches{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: .5, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	})
	if err != nil {
		t.Fatal(err)
	}

	Check(t, b, 12, 63, 6)

	// Reset
	b.load(bidsSnapshot)

	// Update all instances move one after ID
	err = b.updateInsertByID(Tranches{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 8, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	})
	if err != nil {
		t.Fatal(err)
	}

	Check(t, b, 12, 78, 6)

	// Reset
	b.load(bidsSnapshot)

	// Update all instances move one after ID to tail
	err = b.updateInsertByID(Tranches{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 12, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	})
	if err != nil {
		t.Fatal(err)
	}

	Check(t, b, 12, 86, 6)

	// Update all instances then pop new instance
	err = b.updateInsertByID(Tranches{
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 12, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
		{Price: 10, Amount: 2, ID: 10},
	})
	if err != nil {
		t.Fatal(err)
	}

	Check(t, b, 14, 106, 7)

	// Reset
	b.load(bidsSnapshot)

	// Update all instances pop at tail
	err = b.updateInsertByID(Tranches{
		{Price: 0.5, Amount: 2, ID: 0},
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 12, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	})
	if err != nil {
		t.Fatal(err)
	}

	Check(t, b, 14, 87, 7)

	// bookmark head and move to mid
	err = b.updateInsertByID(Tranches{{Price: 9.5, Amount: 2, ID: 5}})
	if err != nil {
		t.Fatal(err)
	}

	Check(t, b, 14, 82, 7)

	// bookmark head and move to tail
	err = b.updateInsertByID(Tranches{{Price: 0.25, Amount: 2, ID: 11}})
	if err != nil {
		t.Fatal(err)
	}

	Check(t, b, 14, 60.5, 7)

	// move tail location to head
	err = b.updateInsertByID(Tranches{{Price: 10, Amount: 2, ID: 11}})
	if err != nil {
		t.Fatal(err)
	}

	Check(t, b, 14, 80, 7)

	// move tail location to mid
	err = b.updateInsertByID(Tranches{{Price: 7.5, Amount: 2, ID: 0}})
	if err != nil {
		t.Fatal(err)
	}

	Check(t, b, 14, 94, 7)

	// insert at head dont match
	err = b.updateInsertByID(Tranches{{Price: 30, Amount: 2, ID: 1234}})
	if err != nil {
		t.Fatal(err)
	}

	Check(t, b, 16, 154, 8)

	// insert between last and 2nd last
	err = b.updateInsertByID(Tranches{{Price: 1.5, Amount: 2, ID: 12345}})
	if err != nil {
		t.Fatal(err)
	}
	Check(t, b, 18, 157, 9)

	// readjust at end
	err = b.updateInsertByID(Tranches{{Price: 1, Amount: 3, ID: 1}})
	if err != nil {
		t.Fatal(err)
	}
	Check(t, b, 19, 158, 9)

	// readjust further and decrease price past tail
	err = b.updateInsertByID(Tranches{{Price: .9, Amount: 3, ID: 1}})
	if err != nil {
		t.Fatal(err)
	}
	Check(t, b, 19, 157.7, 9)

	// purge
	b.load(nil)

	// insert with no liquidity and jumbled
	err = b.updateInsertByID(Tranches{
		{Price: 0.5, Amount: 2, ID: 0},
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 12, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	})
	if err != nil {
		t.Fatal(err)
	}

	Check(t, b, 14, 87, 7)
}

// 20328886	        59.94 ns/op	       0 B/op	       0 allocs/op
func BenchmarkUpdateInsertByID_bids(b *testing.B) {
	bids := Tranches{}
	bidsSnapshot := Tranches{
		{Price: 0.5, Amount: 2, ID: 0},
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 12, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	}
	bids.load(bidsSnapshot)

	for i := 0; i < b.N; i++ {
		err := bids.updateInsertByID(bidsSnapshot, bidCompare)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestInsertUpdatesBid(t *testing.T) {
	b := bidTranches{}
	bidsSnapshot := Tranches{
		{Price: 11, Amount: 1, ID: 11},
		{Price: 9, Amount: 1, ID: 9},
		{Price: 7, Amount: 1, ID: 7},
		{Price: 5, Amount: 1, ID: 5},
		{Price: 3, Amount: 1, ID: 3},
		{Price: 1, Amount: 1, ID: 1},
	}
	b.load(bidsSnapshot)

	err := b.insertUpdates(Tranches{
		{Price: 11, Amount: 1, ID: 11},
		{Price: 9, Amount: 1, ID: 9},
		{Price: 7, Amount: 1, ID: 7},
		{Price: 5, Amount: 1, ID: 5},
		{Price: 3, Amount: 1, ID: 3},
		{Price: 1, Amount: 1, ID: 1},
	})
	if !errors.Is(err, errCollisionDetected) {
		t.Fatalf("expected error %s but received %v", errCollisionDetected, err)
	}

	Check(t, b, 6, 36, 6)

	// Insert at head
	err = b.insertUpdates(Tranches{{Price: 12, Amount: 1, ID: 11}})
	if err != nil {
		t.Fatal(err)
	}

	Check(t, b, 7, 48, 7)

	// Insert at tail
	err = b.insertUpdates(Tranches{{Price: 0.5, Amount: 1, ID: 12}})
	if err != nil {
		t.Fatal(err)
	}

	Check(t, b, 8, 48.5, 8)

	// Insert at mid
	err = b.insertUpdates(Tranches{{Price: 5.5, Amount: 1, ID: 13}})
	if err != nil {
		t.Fatal(err)
	}

	Check(t, b, 9, 54, 9)

	// purge
	b.load(nil)

	// Add one at head
	err = b.insertUpdates(Tranches{{Price: 5.5, Amount: 1, ID: 13}})
	if err != nil {
		t.Fatal(err)
	}

	Check(t, b, 1, 5.5, 1)
}

func TestInsertUpdatesAsk(t *testing.T) {
	a := askTranches{}
	askSnapshot := Tranches{
		{Price: 1, Amount: 1, ID: 1},
		{Price: 3, Amount: 1, ID: 3},
		{Price: 5, Amount: 1, ID: 5},
		{Price: 7, Amount: 1, ID: 7},
		{Price: 9, Amount: 1, ID: 9},
		{Price: 11, Amount: 1, ID: 11},
	}
	a.load(askSnapshot)

	err := a.insertUpdates(Tranches{
		{Price: 11, Amount: 1, ID: 11},
		{Price: 9, Amount: 1, ID: 9},
		{Price: 7, Amount: 1, ID: 7},
		{Price: 5, Amount: 1, ID: 5},
		{Price: 3, Amount: 1, ID: 3},
		{Price: 1, Amount: 1, ID: 1},
	})
	if !errors.Is(err, errCollisionDetected) {
		t.Fatalf("expected error %s but received %v", errCollisionDetected, err)
	}

	Check(t, a, 6, 36, 6)

	// Insert at tail
	err = a.insertUpdates(Tranches{{Price: 12, Amount: 1, ID: 11}})
	if err != nil {
		t.Fatal(err)
	}

	Check(t, a, 7, 48, 7)

	// Insert at head
	err = a.insertUpdates(Tranches{{Price: 0.5, Amount: 1, ID: 12}})
	if err != nil {
		t.Fatal(err)
	}

	Check(t, a, 8, 48.5, 8)

	// Insert at mid
	err = a.insertUpdates(Tranches{{Price: 5.5, Amount: 1, ID: 13}})
	if err != nil {
		t.Fatal(err)
	}

	Check(t, a, 9, 54, 9)

	// purge
	a.load(nil)

	// Add one at head
	err = a.insertUpdates(Tranches{{Price: 5.5, Amount: 1, ID: 13}})
	if err != nil {
		t.Fatal(err)
	}

	Check(t, a, 1, 5.5, 1)
}

// check checks depth values after an update has taken place
func Check(t *testing.T, depth interface{}, liquidity, value float64, expectedLen int) {
	t.Helper()
	b, isBid := depth.(bidTranches)
	a, isAsk := depth.(askTranches)

	var ts Tranches
	switch {
	case isBid:
		ts = b.Tranches
	case isAsk:
		ts = a.Tranches
	default:
		t.Fatal("value passed in is not of type bids or asks")
	}

	liquidityTotal, valueTotal := ts.amount()
	if liquidityTotal != liquidity {
		ts.display()
		t.Fatalf("mismatched liquidity expecting %v but received %v",
			liquidity,
			liquidityTotal)
	}

	if valueTotal != value {
		ts.display()
		t.Fatalf("mismatched total value expecting %v but received %v",
			value,
			valueTotal)
	}

	if len(ts) != expectedLen {
		ts.display()
		t.Fatalf("mismatched expected length count expecting %v but received %v",
			expectedLen,
			len(ts))
	}

	if len(ts) == 0 {
		return
	}

	var price float64
	for x := range ts {
		switch {
		case price == 0:
			price = ts[x].Price
		case isBid && price < ts[x].Price:
			ts.display()
			t.Fatal("Bid pricing out of order should be descending")
		case isAsk && price > ts[x].Price:
			ts.display()
			t.Fatal("Ask pricing out of order should be ascending")
		default:
			price = ts[x].Price
		}
	}
}

func TestAmount(t *testing.T) {
	a := askTranches{}
	askSnapshot := Tranches{
		{Price: 1, Amount: 1, ID: 1},
		{Price: 3, Amount: 1, ID: 3},
		{Price: 5, Amount: 1, ID: 5},
		{Price: 7, Amount: 1, ID: 7},
		{Price: 9, Amount: 1, ID: 9},
		{Price: 11, Amount: 1, ID: 11},
	}
	a.load(askSnapshot)

	liquidity, value := a.amount()
	if liquidity != 6 {
		t.Fatalf("incorrect liquidity calculation expected 6 but received %f", liquidity)
	}

	if value != 36 {
		t.Fatalf("incorrect value calculation expected 36 but received %f", value)
	}
}

func TestGetMovementByBaseAmount(t *testing.T) {
	t.Parallel()
	cases := []struct {
		Name            string
		BaseAmount      float64
		ReferencePrice  float64
		BidLiquidity    Tranches
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
			BidLiquidity:    Tranches{{Price: 10000, Amount: 2}, {Price: 9900, Amount: 7}, {Price: 9800, Amount: 3}},
			BaseAmount:      10,
			ReferencePrice:  10000,
			ExpectedNominal: 0.8999999999999999,
			ExpectedImpact:  2,
			ExpectedCost:    900,
		},
		{
			Name:            "consume first tranche",
			BidLiquidity:    Tranches{{Price: 10000, Amount: 2}, {Price: 9900, Amount: 7}, {Price: 9800, Amount: 3}},
			BaseAmount:      2,
			ReferencePrice:  10000,
			ExpectedNominal: 0,
			ExpectedImpact:  1,
			ExpectedCost:    0,
		},
		{
			Name:            "consume most of first tranche",
			BidLiquidity:    Tranches{{Price: 10000, Amount: 2}, {Price: 9900, Amount: 7}, {Price: 9800, Amount: 3}},
			BaseAmount:      1.5,
			ReferencePrice:  10000,
			ExpectedNominal: 0,
			ExpectedImpact:  0,
			ExpectedCost:    0,
		},
		{
			Name:            "consume full liquidity",
			BidLiquidity:    Tranches{{Price: 10000, Amount: 2}, {Price: 9900, Amount: 7}, {Price: 9800, Amount: 3}},
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
			movement, err := depth.bidTranches.getMovementByBase(tt.BaseAmount, tt.ReferencePrice, false)
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

func assertMovement(tb testing.TB, expected, movement *Movement) {
	tb.Helper()
	assert.InDelta(tb, expected.Sold, movement.Sold, accuracy10dp, "Sold should be correct")
	assert.InDelta(tb, expected.Purchased, movement.Purchased, accuracy10dp, "Purchased should be correct")
	assert.InDelta(tb, expected.AverageOrderCost, movement.AverageOrderCost, accuracy10dp, "AverageOrderCost should be correct")
	assert.InDelta(tb, expected.StartPrice, movement.StartPrice, accuracy10dp, "StartPrice should be correct")
	assert.InDelta(tb, expected.EndPrice, movement.EndPrice, accuracy10dp, "EndPrice should be correct")
	assert.InDelta(tb, expected.NominalPercentage, movement.NominalPercentage, accuracy10dp, "NominalPercentage should be correct")
	assert.Equal(tb, expected.FullBookSideConsumed, movement.FullBookSideConsumed, "FullBookSideConsumed should be correct")
}

func TestGetBaseAmountFromNominalSlippage(t *testing.T) {
	t.Parallel()
	cases := []struct {
		Name            string
		NominalSlippage float64
		ReferencePrice  float64
		BidLiquidity    Tranches
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
			BidLiquidity:    Tranches{{Price: 10000, Amount: 2}, {Price: 9900, Amount: 7}, {Price: 9800, Amount: 3}},
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
			BidLiquidity:    Tranches{{Price: 10000, Amount: 2}, {Price: 9900, Amount: 7}, {Price: 9800, Amount: 3}},
			NominalSlippage: 0.33333333333334,
			ReferencePrice:  10000,
			ExpectedShift: &Movement{
				Sold:              3.0000000000000275,
				Purchased:         29900.00000000027,
				AverageOrderCost:  9966.666666666664,
				NominalPercentage: 0.33333333333334,
				StartPrice:        10000,
				EndPrice:          9900,
			},
		},
		{
			Name:            "consume full liquidity",
			BidLiquidity:    Tranches{{Price: 10000, Amount: 2}, {Price: 9900, Amount: 7}, {Price: 9800, Amount: 3}},
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
			BidLiquidity:    Tranches{{Price: 10000, Amount: 2}, {Price: 9900, Amount: 7}, {Price: 9800, Amount: 3}},
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
			assert.NoError(t, err, "LoadSnapshot should not error")

			base, err := depth.bidTranches.hitBidsByNominalSlippage(tt.NominalSlippage, tt.ReferencePrice)
			if tt.ExpectedError != nil {
				assert.ErrorIs(t, err, tt.ExpectedError, "Should error correctly")
			} else {
				assertMovement(t, tt.ExpectedShift, base)
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
		BidLiquidity   Tranches
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
			BidLiquidity:   Tranches{{Price: 10000, Amount: 2}, {Price: 9900, Amount: 7}, {Price: 9800, Amount: 3}},
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
			BidLiquidity:   Tranches{{Price: 10000, Amount: 2}, {Price: 9900, Amount: 7}, {Price: 9800, Amount: 3}},
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
			BidLiquidity:   Tranches{{Price: 10000, Amount: 2}, {Price: 9900, Amount: 7}, {Price: 9800, Amount: 3}},
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
			base, err := depth.bidTranches.hitBidsByImpactSlippage(tt.ImpactSlippage, tt.ReferencePrice)
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
		AskLiquidity    Tranches
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
			AskLiquidity:    Tranches{{Price: 10000, Amount: 2}, {Price: 10100, Amount: 7}, {Price: 10200, Amount: 3}},
			QuoteAmount:     100900,
			ReferencePrice:  10000,
			ExpectedNominal: 0.8999999999999999,
			ExpectedImpact:  2,
			ExpectedCost:    900,
		},
		{
			Name:            "consume first tranche",
			AskLiquidity:    Tranches{{Price: 10000, Amount: 2}, {Price: 10100, Amount: 7}, {Price: 10200, Amount: 3}},
			QuoteAmount:     20000,
			ReferencePrice:  10000,
			ExpectedNominal: 0,
			ExpectedImpact:  1,
			ExpectedCost:    0,
		},
		{
			Name:            "consume most of first tranche",
			AskLiquidity:    Tranches{{Price: 10000, Amount: 2}, {Price: 10100, Amount: 7}, {Price: 10200, Amount: 3}},
			QuoteAmount:     15000,
			ReferencePrice:  10000,
			ExpectedNominal: 0,
			ExpectedImpact:  0,
			ExpectedCost:    0,
		},
		{
			Name:            "consume full liquidity",
			AskLiquidity:    Tranches{{Price: 10000, Amount: 2}, {Price: 10100, Amount: 7}, {Price: 10200, Amount: 3}},
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
			movement, err := depth.askTranches.getMovementByQuotation(tt.QuoteAmount, tt.ReferencePrice, false)
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
		AskLiquidity    Tranches
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
			AskLiquidity:    Tranches{{Price: 10000, Amount: 2}, {Price: 10100, Amount: 7}, {Price: 10200, Amount: 3}},
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
			AskLiquidity:    Tranches{{Price: 10000, Amount: 2}, {Price: 10100, Amount: 7}, {Price: 10200, Amount: 3}},
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
			AskLiquidity:    Tranches{{Price: 10000, Amount: 2}, {Price: 10100, Amount: 7}, {Price: 10200, Amount: 3}},
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
			AskLiquidity:    Tranches{{Price: 10000, Amount: 2}, {Price: 10100, Amount: 7}, {Price: 10200, Amount: 3}},
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
			AskLiquidity:    Tranches{{Price: 10000, Amount: 2}, {Price: 10100, Amount: 7}, {Price: 10200, Amount: 3}},
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
			assert.NoError(t, err, "LoadSnapshot should not error")

			quote, err := depth.askTranches.liftAsksByNominalSlippage(tt.NominalSlippage, tt.ReferencePrice)
			if tt.ExpectedError != nil {
				assert.ErrorIs(t, err, tt.ExpectedError, "Should error correctly")
			} else {
				assertMovement(t, tt.ExpectedShift, quote)
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
		AskLiquidity   Tranches
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
			AskLiquidity:   Tranches{{Price: 10000, Amount: 2}, {Price: 10100, Amount: 7}, {Price: 10200, Amount: 3}},
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
			AskLiquidity:   Tranches{{Price: 10000, Amount: 2}, {Price: 10100, Amount: 7}, {Price: 10200, Amount: 3}},
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
			AskLiquidity:   Tranches{{Price: 10000, Amount: 2}, {Price: 10100, Amount: 7}, {Price: 10200, Amount: 3}},
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
			assert.NoError(t, err, "LoadSnapshot should not error")

			quote, err := depth.askTranches.liftAsksByImpactSlippage(tt.ImpactSlippage, tt.ReferencePrice)
			if tt.ExpectedError != nil {
				assert.ErrorIs(t, err, tt.ExpectedError, "Should error correctly")
			} else {
				assertMovement(t, tt.ExpectedShift, quote)
			}
		})
	}
}

func TestGetHeadPrice(t *testing.T) {
	t.Parallel()
	depth := NewDepth(id)
	if _, err := depth.bidTranches.getHeadPriceNoLock(); !errors.Is(err, errNoLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoLiquidity)
	}
	if _, err := depth.askTranches.getHeadPriceNoLock(); !errors.Is(err, errNoLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoLiquidity)
	}
	err := depth.LoadSnapshot(bid, ask, 0, time.Now(), true)
	if err != nil {
		t.Fatalf("failed to load snapshot: %s", err)
	}

	val, err := depth.bidTranches.getHeadPriceNoLock()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if val != 1336 {
		t.Fatal("unexpected value")
	}

	val, err = depth.askTranches.getHeadPriceNoLock()
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
	assert.ErrorIs(t, err, errInvalidCost, "finalizeFields should error when cost is invalid")

	_, err = m.finalizeFields(1, 0, 0, 0, false)
	assert.ErrorIs(t, err, errInvalidAmount, "finalizeFields should error when amount is invalid")

	_, err = m.finalizeFields(1, 1, 0, 0, false)
	assert.ErrorIs(t, err, errInvalidHeadPrice, "finalizeFields should error correctly with bad head price")

	// Test slippage as per https://en.wikipedia.org/wiki/Slippage_(finance)
	mov, err := m.finalizeFields(20000*151.11585, 20000, 151.08, 0, false)
	assert.NoError(t, err, "finalizeFields should not error")
	assert.InDelta(t, 717.0, mov.SlippageCost, 0.000000001, "SlippageCost should be correct")
}

// 8384302	       150.9 ns/op	     480 B/op	       1 allocs/op
func BenchmarkRetrieve(b *testing.B) {
	asks := Tranches{}
	asksSnapshot := Tranches{
		{Price: 1, Amount: 1, ID: 1},
		{Price: 3, Amount: 1, ID: 3},
		{Price: 5, Amount: 1, ID: 5},
		{Price: 7, Amount: 1, ID: 7},
		{Price: 9, Amount: 1, ID: 9},
		{Price: 11, Amount: 1, ID: 11},
	}
	asks.load(asksSnapshot)

	for i := 0; i < b.N; i++ {
		_ = asks.retrieve(6)
	}
}
