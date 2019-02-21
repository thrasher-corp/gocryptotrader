package stats

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/currency"
)

func TestLenByPrice(t *testing.T) {
	p := currency.NewPair("BTC", "USD")
	Items = []Item{
		{
			Exchange:  "ANX",
			Pair:      p,
			AssetType: "SPOT",
			Price:     1200,
			Volume:    5,
		},
	}

	if ByPrice.Len(Items) < 1 {
		t.Error("Test Failed - stats LenByPrice() length not correct.")
	}
}

func TestLessByPrice(t *testing.T) {
	p := currency.NewPair("BTC", "USD")

	Items = []Item{
		{
			Exchange:  "alphapoint",
			Pair:      p,
			AssetType: "SPOT",
			Price:     1200,
			Volume:    5,
		},
		{
			Exchange:  "bitfinex",
			Pair:      p,
			AssetType: "SPOT",
			Price:     1198,
			Volume:    20,
		},
	}

	if !ByPrice.Less(Items, 1, 0) {
		t.Error("Test Failed - stats LessByPrice() incorrect return.")
	}
	if ByPrice.Less(Items, 0, 1) {
		t.Error("Test Failed - stats LessByPrice() incorrect return.")
	}
}

func TestSwapByPrice(t *testing.T) {
	p := currency.NewPair("BTC", "USD")

	Items = []Item{
		{
			Exchange:  "bitstamp",
			Pair:      p,
			AssetType: "SPOT",
			Price:     1324,
			Volume:    5,
		},
		{
			Exchange:  "btcc",
			Pair:      p,
			AssetType: "SPOT",
			Price:     7863,
			Volume:    20,
		},
	}

	ByPrice.Swap(Items, 0, 1)
	if Items[0].Exchange != "btcc" || Items[1].Exchange != "bitstamp" {
		t.Error("Test Failed - stats SwapByPrice did not swap values.")
	}
}

func TestLenByVolume(t *testing.T) {
	if ByVolume.Len(Items) != 2 {
		t.Error("Test Failed - stats lenByVolume did not swap values.")
	}
}

func TestLessByVolume(t *testing.T) {
	if !ByVolume.Less(Items, 1, 0) {
		t.Error("Test Failed - stats LessByVolume() incorrect return.")
	}
	if ByVolume.Less(Items, 0, 1) {
		t.Error("Test Failed - stats LessByVolume() incorrect return.")
	}
}

func TestSwapByVolume(t *testing.T) {
	ByPrice.Swap(Items, 0, 1)

	if Items[1].Exchange != "btcc" || Items[0].Exchange != "bitstamp" {
		t.Error("Test Failed - stats SwapByVolume did not swap values.")
	}
}

func TestAdd(t *testing.T) {
	Items = Items[:0]
	p := currency.NewPair("BTC", "USD")
	Add("ANX", p, "SPOT", 1200, 42)

	if len(Items) < 1 {
		t.Error("Test Failed - stats Add did not add exchange info.")
	}

	Add("", p, "", 0, 0)

	if len(Items) != 1 {
		t.Error("Test Failed - stats Add did not add exchange info.")
	}

	p.Base = currency.XBT
	Add("ANX", p, "SPOT", 1201, 43)

	if Items[1].Pair.String() != "XBTUSD" {
		t.Fatal("Test failed. stats Add did not add exchange info.")
	}

	p = currency.NewPair("ETH", "USDT")
	Add("ANX", p, "SPOT", 300, 1000)

	if Items[2].Pair.String() != "ETHUSD" {
		t.Fatal("Test failed. stats Add did not add exchange info.")
	}
}

func TestAppend(t *testing.T) {
	p := currency.NewPair("BTC", "USD")
	Append("sillyexchange", p, "SPOT", 1234, 45)
	if len(Items) < 2 {
		t.Error("Test Failed - stats Append did not add exchange values.")
	}

	Append("sillyexchange", p, "SPOT", 1234, 45)
	if len(Items) == 3 {
		t.Error("Test Failed - stats Append added exchange values")
	}
}

func TestAlreadyExists(t *testing.T) {
	p := currency.NewPair("BTC", "USD")
	if !AlreadyExists("ANX", p, "SPOT", 1200, 42) {
		t.Error("Test Failed - stats AlreadyExists exchange does not exist.")
	}
	p.Base = currency.NewCurrencyCode("dii")
	if AlreadyExists("bla", p, "SPOT", 1234, 123) {
		t.Error("Test Failed - stats AlreadyExists found incorrect exchange.")
	}
}

func TestSortExchangesByVolume(t *testing.T) {
	p := currency.NewPair("BTC", "USD")
	topVolume := SortExchangesByVolume(p, "SPOT", true)
	if topVolume[0].Exchange != "sillyexchange" {
		t.Error("Test Failed - stats SortExchangesByVolume incorrectly sorted values.")
	}

	topVolume = SortExchangesByVolume(p, "SPOT", false)
	if topVolume[0].Exchange != "ANX" {
		t.Error("Test Failed - stats SortExchangesByVolume incorrectly sorted values.")
	}
}

func TestSortExchangesByPrice(t *testing.T) {
	p := currency.NewPair("BTC", "USD")
	topPrice := SortExchangesByPrice(p, "SPOT", true)
	if topPrice[0].Exchange != "sillyexchange" {
		t.Error("Test Failed - stats SortExchangesByPrice incorrectly sorted values.")
	}

	topPrice = SortExchangesByPrice(p, "SPOT", false)
	if topPrice[0].Exchange != "ANX" {
		t.Error("Test Failed - stats SortExchangesByPrice incorrectly sorted values.")
	}
}
