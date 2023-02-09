package stats

import (
	"errors"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

const testExchange = "OKx"

func TestAdd(t *testing.T) {
	t.Parallel()

	err := Add("", currency.EMPTYPAIR, asset.Empty, 0, 0)
	if !errors.Is(err, errInvalidParams) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidParams)
	}

	err = Add(testExchange, currency.EMPTYPAIR, asset.Empty, 0, 0)
	if !errors.Is(err, errInvalidParams) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidParams)
	}

	stdPair, err := currency.NewPairFromStrings("BTC", "USD")
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	err = Add(testExchange, stdPair, asset.Spot, 0, 0)
	if !errors.Is(err, errInvalidParams) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidParams)
	}

	err = Add(testExchange, stdPair, asset.Spot, 1, 0)
	if !errors.Is(err, errInvalidParams) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidParams)
	}

	err = Add(testExchange, stdPair, asset.Spot, 1200, 42)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if len(getItemsByExchange(testExchange)) != 1 {
		t.Fatal("stats Add did not add exchange info.")
	}

	stdPair.Base = currency.XBT
	err = Add(testExchange, stdPair, asset.Spot, 1201, 43)
	if err != nil {
		t.Fatal(err)
	}

	stored := getItemsByExchange(testExchange)
	if len(stored) != 2 {
		t.Fatalf("received: '%v' but expected: '%v'", len(stored), 2)
	}

	if stored[1].Pair.String() != "XBTUSD" {
		t.Fatal("stats Add did not add exchange info.")
	}

	stdPair, err = currency.NewPairFromStrings("ETH", "USDT")
	if err != nil {
		t.Fatal(err)
	}

	err = Add(testExchange, stdPair, asset.Spot, 300, 1000)
	if err != nil {
		t.Fatal(err)
	}

	stored = getItemsByExchange(testExchange)
	if len(stored) != 4 {
		t.Fatalf("received: '%v' but expected: '%v'", len(stored), 4)
	}

	if stored[2].Pair.String() != "ETHUSD" {
		t.Fatal("stats Add did not add exchange info.")
	}
}

func getItemsByExchange(name string) []Item {
	items.mu.Lock()
	defer items.mu.Unlock()
	result := make([]Item, 0, len(items.bucket))
	for x := range items.bucket {
		if items.bucket[x].Exchange == name {
			result = append(result, items.bucket[x])
		}
	}
	return result
}

func TestUpdate(t *testing.T) {
	t.Parallel()
	p, err := currency.NewPairFromStrings("BTC", "USD")
	if err != nil {
		t.Fatal(err)
	}
	update("sillyexchange", p, asset.Spot, 1234, 45)
	stored := getItemsByExchange("sillyexchange")
	if len(stored) != 1 {
		t.Error("stats AppendResults did not add exchange values.")
	}

	update("sillyexchange", p, asset.Spot, 1234, 45)
	stored = getItemsByExchange("sillyexchange")
	if len(stored) != 1 {
		t.Error("stats AppendResults added exchange values")
	}
}

func TestSortExchangesByVolume(t *testing.T) {
	t.Parallel()
	p, err := currency.NewPairFromStrings("BTC", "USD")
	if err != nil {
		t.Fatal(err)
	}
	err = Add("byVolume1", p, asset.Spot, 1200, 1)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	err = Add("byVolume2", p, asset.Spot, 1200, 500)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	topVolume := SortExchangesByVolume(p, asset.Spot, true)
	if topVolume[0].Exchange != "byVolume2" {
		t.Error("stats SortExchangesByVolume incorrectly sorted values.")
	}

	topVolume = SortExchangesByVolume(p, asset.Spot, false)
	if topVolume[0].Exchange != "byVolume1" {
		t.Error("stats SortExchangesByVolume incorrectly sorted values.")
	}
}

func TestSortExchangesByPrice(t *testing.T) {
	t.Parallel()
	p, err := currency.NewPairFromStrings("BTC", "USD")
	if err != nil {
		t.Fatal(err)
	}
	err = Add("byPrice1", p, asset.Spot, 1, 42)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	err = Add("byPrice2", p, asset.Spot, 5000, 42)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
	topPrice := SortExchangesByPrice(p, asset.Spot, true)
	if topPrice[0].Exchange != "byPrice2" {
		t.Error("stats SortExchangesByPrice incorrectly sorted values.")
	}

	topPrice = SortExchangesByPrice(p, asset.Spot, false)
	if topPrice[0].Exchange != "byPrice1" {
		t.Error("stats SortExchangesByPrice incorrectly sorted values.")
	}
}
