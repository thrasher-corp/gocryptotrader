package orderbook

import "testing"

func BenchmarkLoad(b *testing.B) {
	ts := Levels{}
	for b.Loop() {
		ts.load(ask)
	}
}

func BenchmarkUpdateByID(b *testing.B) {
	asks := Levels{}
	asksSnapshot := Levels{
		{Price: 1, Amount: 1, ID: 1},
		{Price: 3, Amount: 1, ID: 3},
		{Price: 5, Amount: 1, ID: 5},
		{Price: 7, Amount: 1, ID: 7},
		{Price: 9, Amount: 1, ID: 9},
		{Price: 11, Amount: 1, ID: 11},
	}
	asks.load(asksSnapshot)

	for b.Loop() {
		err := asks.updateByID(asksSnapshot)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDeleteByID(b *testing.B) {
	asks := Levels{}
	asksSnapshot := Levels{
		{Price: 1, Amount: 1, ID: 1},
		{Price: 3, Amount: 1, ID: 3},
		{Price: 5, Amount: 1, ID: 5},
		{Price: 7, Amount: 1, ID: 7},
		{Price: 9, Amount: 1, ID: 9},
		{Price: 11, Amount: 1, ID: 11},
	}
	asks.load(asksSnapshot)

	for b.Loop() {
		err := asks.deleteByID(asksSnapshot, false)
		if err != nil {
			b.Fatal(err)
		}
		asks.load(asksSnapshot) // reset
	}
}

func BenchmarkRetrieve(b *testing.B) {
	asks := Levels{}
	asksSnapshot := Levels{
		{Price: 1, Amount: 1, ID: 1},
		{Price: 3, Amount: 1, ID: 3},
		{Price: 5, Amount: 1, ID: 5},
		{Price: 7, Amount: 1, ID: 7},
		{Price: 9, Amount: 1, ID: 9},
		{Price: 11, Amount: 1, ID: 11},
	}
	asks.load(asksSnapshot)

	for b.Loop() {
		_ = asks.retrieve(6)
	}
}

func BenchmarkUpdateInsertByPrice_Amend(b *testing.B) {
	a := askLevels{}
	a.load(ask)

	updates := Levels{
		{
			Price:  1337, // Amend
			Amount: 2,
		},
		{
			Price:  1337, // Amend
			Amount: 1,
		},
	}

	for b.Loop() {
		a.updateInsertByPrice(updates, 0)
	}
}

func BenchmarkUpdateInsertByPrice_Insert_Delete(b *testing.B) {
	a := askLevels{}

	a.load(ask)

	updates := Levels{
		{
			Price:  1337.5, // Insert
			Amount: 2,
		},
		{
			Price:  1337.5, // Delete
			Amount: 0,
		},
	}

	for b.Loop() {
		a.updateInsertByPrice(updates, 0)
	}
}

func BenchmarkUpdateInsertByID_asks(b *testing.B) {
	asks := Levels{}
	asksSnapshot := Levels{
		{Price: 1, Amount: 1, ID: 1},
		{Price: 3, Amount: 1, ID: 3},
		{Price: 5, Amount: 1, ID: 5},
		{Price: 7, Amount: 1, ID: 7},
		{Price: 9, Amount: 1, ID: 9},
		{Price: 11, Amount: 1, ID: 11},
	}
	asks.load(asksSnapshot)

	for b.Loop() {
		err := asks.updateInsertByID(asksSnapshot, askCompare)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUpdateInsertByID_bids(b *testing.B) {
	bids := Levels{}
	bidsSnapshot := Levels{
		{Price: 0.5, Amount: 2, ID: 0},
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 12, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	}
	bids.load(bidsSnapshot)

	for b.Loop() {
		err := bids.updateInsertByID(bidsSnapshot, bidCompare)
		if err != nil {
			b.Fatal(err)
		}
	}
}
