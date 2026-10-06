package stats

import (
	"cmp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

const (
	testExchange = "Okx"
)

func TestAdd(t *testing.T) {
	items = items[:0]
	p, err := currency.NewPairFromStrings("BTC", "USD")
	if err != nil {
		t.Fatal(err)
	}
	err = Add(testExchange, p, asset.Spot, 1200, 42)
	if err != nil {
		t.Fatal(err)
	}

	if len(items) < 1 {
		t.Error("stats Add did not add exchange info.")
	}

	err = Add("", p, asset.Empty, 0, 0)
	if err == nil {
		t.Fatal("error cannot be nil")
	}

	if len(items) != 1 {
		t.Error("stats Add did not add exchange info.")
	}

	p.Base = currency.XBT
	err = Add(testExchange, p, asset.Spot, 1201, 43)
	if err != nil {
		t.Fatal(err)
	}

	if items[1].Pair.String() != "XBTUSD" {
		t.Fatal("stats Add did not add exchange info.")
	}

	p, err = currency.NewPairFromStrings("ETH", "USDT")
	if err != nil {
		t.Fatal(err)
	}

	err = Add(testExchange, p, asset.Spot, 300, 1000)
	if err != nil {
		t.Fatal(err)
	}

	if items[2].Pair.String() != "ETHUSD" {
		t.Fatal("stats Add did not add exchange info.")
	}
}

func TestAppend(t *testing.T) {
	p, err := currency.NewPairFromStrings("BTC", "USD")
	if err != nil {
		t.Fatal(err)
	}
	Append("sillyexchange", p, asset.Spot, 1234, 45)
	if len(items) < 2 {
		t.Error("stats AppendResults did not add exchange values.")
	}

	Append("sillyexchange", p, asset.Spot, 1234, 45)
	if len(items) == 3 {
		t.Error("stats AppendResults added exchange values")
	}
}

func TestAlreadyExists(t *testing.T) {
	p, err := currency.NewPairFromStrings("BTC", "USD")
	if err != nil {
		t.Fatal(err)
	}
	if !AlreadyExists(testExchange, p, asset.Spot, 1200, 42) {
		t.Error("stats AlreadyExists exchange does not exist.")
	}
	p.Base = currency.NewCode("dii")
	if AlreadyExists("bla", p, asset.Spot, 1234, 123) {
		t.Error("stats AlreadyExists found incorrect exchange.")
	}
}

func TestSortExchangesByVolume(t *testing.T) {
	p, err := currency.NewPairFromStrings("BTC", "USD")
	if err != nil {
		t.Fatal(err)
	}
	topVolume := SortExchangesByVolume(p, asset.Spot, true)
	if topVolume[0].Exchange != "sillyexchange" {
		t.Error("stats SortExchangesByVolume incorrectly sorted values.")
	}

	topVolume = SortExchangesByVolume(p, asset.Spot, false)
	if topVolume[0].Exchange != testExchange {
		t.Error("stats SortExchangesByVolume incorrectly sorted values.")
	}
}

func TestSortExchangesByPrice(t *testing.T) {
	p, err := currency.NewPairFromStrings("BTC", "USD")
	if err != nil {
		t.Fatal(err)
	}
	topPrice := SortExchangesByPrice(p, asset.Spot, true)
	if topPrice[0].Exchange != "sillyexchange" {
		t.Error("stats SortExchangesByPrice incorrectly sorted values.")
	}

	topPrice = SortExchangesByPrice(p, asset.Spot, false)
	if topPrice[0].Exchange != testExchange {
		t.Error("stats SortExchangesByPrice incorrectly sorted values.")
	}
}

func TestSortItems(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		reverse bool
		want    []float64
	}{
		{
			name: "ascending",
			want: []float64{1, 2, 2, 3},
		},
		{
			name:    "descending",
			reverse: true,
			want:    []float64{3, 2, 2, 1},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			items := []Item{
				{Price: 2},
				{Price: 3},
				{Price: 1},
				{Price: 2},
			}
			sortItems(items, tc.reverse, func(a, b Item) int { return cmp.Compare(a.Price, b.Price) })

			got := make([]float64, len(items))
			for i := range items {
				got[i] = items[i].Price
			}
			assert.Equal(t, tc.want, got, "sortItems should order the prices")
		})
	}

	assert.NotPanics(t, func() { sortItems(nil, true, func(a, b Item) int { return cmp.Compare(a.Price, b.Price) }) },
		"sortItems should accept a nil slice")
}
