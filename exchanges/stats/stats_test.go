package stats

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/currency/pair"
)

func TestLenByPrice(t *testing.T) {
	p := pair.NewCurrencyPair("BTC", "USD")
	i := Item{
		Exchange:  "ANX",
		Pair:      p,
		AssetType: "SPOT",
		Price:     1200,
		Volume:    5,
	}

	Items = append(Items, i)
	if ByPrice.Len(Items) < 1 {
		t.Error("Test Failed - stats LenByPrice() length not correct.")
	}
}

func TestLessByPrice(t *testing.T) {
	p := pair.NewCurrencyPair("BTC", "USD")
	i := Item{
		Exchange:  "alphapoint",
		Pair:      p,
		AssetType: "SPOT",
		Price:     1200,
		Volume:    5,
	}

	i2 := Item{
		Exchange:  "bitfinex",
		Pair:      p,
		AssetType: "SPOT",
		Price:     1198,
		Volume:    20,
	}

	Items = append(Items, i)
	Items = append(Items, i2)

	if !ByPrice.Less(Items, 2, 1) {
		t.Error("Test Failed - stats LessByPrice() incorrect return.")
	}
	if ByPrice.Less(Items, 1, 2) {
		t.Error("Test Failed - stats LessByPrice() incorrect return.")
	}
}

func TestSwapByPrice(t *testing.T) {
	p := pair.NewCurrencyPair("BTC", "USD")
	i := Item{
		Exchange:  "bitstamp",
		Pair:      p,
		AssetType: "SPOT",
		Price:     1324,
		Volume:    5,
	}

	i2 := Item{
		Exchange:  "btcc",
		Pair:      p,
		AssetType: "SPOT",
		Price:     7863,
		Volume:    20,
	}

	Items = append(Items, i)
	Items = append(Items, i2)
	ByPrice.Swap(Items, 3, 4)
	if Items[3].Exchange != "btcc" || Items[4].Exchange != "bitstamp" {
		t.Error("Test Failed - stats SwapByPrice did not swap values.")
	}
}

func TestLenByVolume(t *testing.T) {
	if ByVolume.Len(Items) != 5 {
		t.Error("Test Failed - stats lenByVolume did not swap values.")
	}
}

func TestLessByVolume(t *testing.T) {
	if !ByVolume.Less(Items, 1, 2) {
		t.Error("Test Failed - stats LessByVolume() incorrect return.")
	}
	if ByVolume.Less(Items, 2, 1) {
		t.Error("Test Failed - stats LessByVolume() incorrect return.")
	}
}

func TestSwapByVolume(t *testing.T) {
	ByPrice.Swap(Items, 3, 4)

	if Items[4].Exchange != "btcc" || Items[3].Exchange != "bitstamp" {
		t.Error("Test Failed - stats SwapByVolume did not swap values.")
	}
}

func TestAdd(t *testing.T) {
	Items = Items[:0]
	p := pair.NewCurrencyPair("BTC", "USD")
	Add("ANX", p, "SPOT", 1200, 42)

	if len(Items) < 1 {
		t.Error("Test Failed - stats Add did not add exchange info.")
	}

	Add("", p, "", 0, 0)

	if len(Items) != 1 {
		t.Error("Test Failed - stats Add did not add exchange info.")
	}

	p.FirstCurrency = "XBT"
	Add("ANX", p, "SPOT", 1201, 43)

	if Items[1].Pair.Pair() != "XBTUSD" {
		t.Fatal("Test failed. stats Add did not add exchange info.")
	}

	p = pair.NewCurrencyPair("ETH", "USDT")
	Add("ANX", p, "SPOT", 300, 1000)

	if Items[2].Pair.Pair() != "ETHUSD" {
		t.Fatal("Test failed. stats Add did not add exchange info.")
	}
}

func TestAppend(t *testing.T) {
	p := pair.NewCurrencyPair("BTC", "USD")
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
	p := pair.NewCurrencyPair("BTC", "USD")
	if !AlreadyExists("ANX", p, "SPOT", 1200, 42) {
		t.Error("Test Failed - stats AlreadyExists exchange does not exist.")
	}
	p.FirstCurrency = "dii"
	if AlreadyExists("bla", p, "SPOT", 1234, 123) {
		t.Error("Test Failed - stats AlreadyExists found incorrect exchange.")
	}
}

func TestSortExchangesByVolume(t *testing.T) {
	p := pair.NewCurrencyPair("BTC", "USD")
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
	p := pair.NewCurrencyPair("BTC", "USD")
	topPrice := SortExchangesByPrice(p, "SPOT", true)
	if topPrice[0].Exchange != "sillyexchange" {
		t.Error("Test Failed - stats SortExchangesByPrice incorrectly sorted values.")
	}

	topPrice = SortExchangesByPrice(p, "SPOT", false)
	if topPrice[0].Exchange != "ANX" {
		t.Error("Test Failed - stats SortExchangesByPrice incorrectly sorted values.")
	}
}
