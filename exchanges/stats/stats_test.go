package stats

import (
	"testing"
)

func TestLenByPrice(t *testing.T) {
	exchangeInfo := ExchangeInfo{
		Exchange:      "ANX",
		FirstCurrency: "BTC",
		FiatCurrency:  "USD",
		Price:         1200,
		Volume:        5,
	}

	ExchInfo = append(ExchInfo, exchangeInfo)
	if ByPrice.Len(ExchInfo) < 1 {
		t.Error("Test Failed - stats LenByPrice() length not correct.")
	}
}

func TestLessByPrice(t *testing.T) {
	exchangeInfo := ExchangeInfo{
		Exchange:      "alphapoint",
		FirstCurrency: "BTC",
		FiatCurrency:  "USD",
		Price:         1200,
		Volume:        5,
	}

	exchangeInfo2 := ExchangeInfo{
		Exchange:      "bitfinex",
		FirstCurrency: "BTC",
		FiatCurrency:  "USD",
		Price:         1198,
		Volume:        20,
	}

	ExchInfo = append(ExchInfo, exchangeInfo)
	ExchInfo = append(ExchInfo, exchangeInfo2)

	if !ByPrice.Less(ExchInfo, 2, 1) {
		t.Error("Test Failed - stats LessByPrice() incorrect return.")
	}
	if ByPrice.Less(ExchInfo, 1, 2) {
		t.Error("Test Failed - stats LessByPrice() incorrect return.")
	}
}

func TestSwapByPrice(t *testing.T) {
	exchangeInfo := ExchangeInfo{
		Exchange:      "bitstamp",
		FirstCurrency: "BTC",
		FiatCurrency:  "USD",
		Price:         1324,
		Volume:        5,
	}

	exchangeInfo2 := ExchangeInfo{
		Exchange:      "btcc",
		FirstCurrency: "BTC",
		FiatCurrency:  "USD",
		Price:         7863,
		Volume:        20,
	}

	ExchInfo = append(ExchInfo, exchangeInfo)
	ExchInfo = append(ExchInfo, exchangeInfo2)
	ByPrice.Swap(ExchInfo, 3, 4)
	if ExchInfo[3].Exchange != "btcc" || ExchInfo[4].Exchange != "bitstamp" {
		t.Error("Test Failed - stats SwapByPrice did not swap values.")
	}
}

func TestLenByVolume(t *testing.T) {
	if ByVolume.Len(ExchInfo) != 5 {
		t.Error("Test Failed - stats lenByVolume did not swap values.")
	}
}

func TestLessByVolume(t *testing.T) {
	if !ByVolume.Less(ExchInfo, 1, 2) {
		t.Error("Test Failed - stats LessByVolume() incorrect return.")
	}
	if ByVolume.Less(ExchInfo, 2, 1) {
		t.Error("Test Failed - stats LessByVolume() incorrect return.")
	}
}

func TestSwapByVolume(t *testing.T) {
	ByPrice.Swap(ExchInfo, 3, 4)

	if ExchInfo[4].Exchange != "btcc" || ExchInfo[3].Exchange != "bitstamp" {
		t.Error("Test Failed - stats SwapByVolume did not swap values.")
	}
}

func TestAddExchangeInfo(t *testing.T) {
	ExchInfo = ExchInfo[:0]
	AddExchangeInfo("ANX", "BTC", "USD", 1200, 42)

	if len(ExchInfo) < 1 {
		t.Error("Test Failed - stats AddExchangeInfo did not add exchange info.")
	}
}

func TestAppendExchangeInfo(t *testing.T) {
	AppendExchangeInfo("sillyexchange", "BTC", "USD", 1234, 45)
	if len(ExchInfo) < 2 {
		t.Error("Test Failed - stats AppendExchangeInfo did not add exchange values.")
	}
	AppendExchangeInfo("sillyexchange", "BTC", "USD", 1234, 45)
	if len(ExchInfo) == 3 {
		t.Error("Test Failed - stats AppendExchangeInfo added exchange values")
	}
}

func TestExchangeInfoAlreadyExists(t *testing.T) {
	if !ExchangeInfoAlreadyExists("ANX", "BTC", "USD", 1200, 42) {
		t.Error("Test Failed - stats ExchangeInfoAlreadyExists exchange does not exist.")
	}
	if ExchangeInfoAlreadyExists("bla", "dii", "USD", 1234, 123) {
		t.Error("Test Failed - stats ExchangeInfoAlreadyExists found incorrect exchange.")
	}
}

func TestSortExchangesByVolume(t *testing.T) {
	topVolume := SortExchangesByVolume("BTC", "USD", true)
	if topVolume[0].Exchange != "sillyexchange" {
		t.Error("Test Failed - stats SortExchangesByVolume incorrectly sorted values.")
	}
}

func TestSortExchangesByPrice(t *testing.T) {
	topPrice := SortExchangesByPrice("BTC", "USD", true)
	if topPrice[0].Exchange != "sillyexchange" {
		t.Error("Test Failed - stats SortExchangesByPrice incorrectly sorted values.")
	}
}
