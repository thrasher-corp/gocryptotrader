package pair

import (
	"testing"
)

func TestLower(t *testing.T) {
	t.Parallel()
	pair := CurrencyItem("BTCUSD")
	actual := pair.Lower()
	expected := CurrencyItem("btcusd")
	if actual != expected {
		t.Errorf("Test failed. Lower(): %s was not equal to expected value: %s",
			actual, expected)
	}
}

func TestUpper(t *testing.T) {
	t.Parallel()
	pair := CurrencyItem("btcusd")
	actual := pair.Upper()
	expected := CurrencyItem("BTCUSD")
	if actual != expected {
		t.Errorf("Test failed. Upper(): %s was not equal to expected value: %s",
			actual, expected)
	}
}

func TestString(t *testing.T) {
	t.Parallel()
	pair := NewCurrencyPair("BTC", "USD")
	actual := "BTCUSD"
	expected := pair.Pair().String()
	if actual != expected {
		t.Errorf("Test failed. String(): %s was not equal to expected value: %s",
			actual, expected)
	}
}

func TestGetFirstCurrency(t *testing.T) {
	t.Parallel()
	pair := NewCurrencyPair("BTC", "USD")
	actual := pair.GetFirstCurrency()
	expected := CurrencyItem("BTC")
	if actual != expected {
		t.Errorf(
			"Test failed. GetFirstCurrency(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}
}

func TestGetSecondCurrency(t *testing.T) {
	t.Parallel()
	pair := NewCurrencyPair("BTC", "USD")
	actual := pair.GetSecondCurrency()
	expected := CurrencyItem("USD")
	if actual != expected {
		t.Errorf(
			"Test failed. GetSecondCurrency(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}
}

func TestPair(t *testing.T) {
	t.Parallel()
	pair := NewCurrencyPair("BTC", "USD")
	actual := pair.Pair()
	expected := CurrencyItem("BTCUSD")
	if actual != expected {
		t.Errorf(
			"Test failed. Pair(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}
}

func TestDisplay(t *testing.T) {
	t.Parallel()
	pair := NewCurrencyPairDelimiter("BTC-USD", "-")
	actual := pair.Pair()
	expected := CurrencyItem("BTC-USD")
	if actual != expected {
		t.Errorf(
			"Test failed. Pair(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}

	actual = pair.Display("", false)
	expected = CurrencyItem("btcusd")
	if actual != expected {
		t.Errorf(
			"Test failed. Pair(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}

	actual = pair.Display("~", true)
	expected = CurrencyItem("BTC~USD")
	if actual != expected {
		t.Errorf(
			"Test failed. Pair(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}
}

func TestEqual(t *testing.T) {
	t.Parallel()
	pair := NewCurrencyPair("BTC", "USD")
	secondPair := NewCurrencyPair("btc", "uSd")
	actual := pair.Equal(secondPair)
	expected := true
	if actual != expected {
		t.Errorf(
			"Test failed. Equal(): %v was not equal to expected value: %v",
			actual, expected,
		)
	}

	secondPair.SecondCurrency = "ETH"
	actual = pair.Equal(secondPair)
	expected = false
	if actual != expected {
		t.Errorf(
			"Test failed. Equal(): %v was not equal to expected value: %v",
			actual, expected,
		)
	}

	secondPair = NewCurrencyPair("USD", "BTC")
	actual = pair.Equal(secondPair)
	expected = true
	if actual != expected {
		t.Errorf(
			"Test failed. Equal(): %v was not equal to expected value: %v",
			actual, expected,
		)
	}
}

func TestNewCurrencyPair(t *testing.T) {
	t.Parallel()
	pair := NewCurrencyPair("BTC", "USD")
	actual := pair.Pair()
	expected := CurrencyItem("BTCUSD")
	if actual != expected {
		t.Errorf(
			"Test failed. Pair(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}
}

func TestNewCurrencyPairDelimiter(t *testing.T) {
	t.Parallel()
	pair := NewCurrencyPairDelimiter("BTC-USD", "-")
	actual := pair.Pair()
	expected := CurrencyItem("BTC-USD")
	if actual != expected {
		t.Errorf(
			"Test failed. Pair(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}

	actual = CurrencyItem(pair.Delimiter)
	expected = "-"
	if actual != expected {
		t.Errorf(
			"Test failed. Delmiter: %s was not equal to expected value: %s",
			actual, expected,
		)
	}
}

// NewCurrencyPairFromIndex returns a CurrencyPair via a currency string and
// specific index
func TestNewCurrencyPairFromIndex(t *testing.T) {
	t.Parallel()
	currency := "BTCUSD"
	index := "BTC"

	pair := NewCurrencyPairFromIndex(currency, index)
	pair.Delimiter = "-"
	actual := pair.Pair()

	expected := CurrencyItem("BTC-USD")
	if actual != expected {
		t.Errorf(
			"Test failed. Pair(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}

	currency = "DOGEBTC"

	pair = NewCurrencyPairFromIndex(currency, index)
	pair.Delimiter = "-"
	actual = pair.Pair()

	expected = CurrencyItem("DOGE-BTC")
	if actual != expected {
		t.Errorf(
			"Test failed. Pair(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}
}

func TestNewCurrencyPairFromString(t *testing.T) {
	t.Parallel()
	pairStr := "BTC-USD"
	pair := NewCurrencyPairFromString(pairStr)
	actual := pair.Pair()
	expected := CurrencyItem("BTC-USD")
	if actual != expected {
		t.Errorf(
			"Test failed. Pair(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}

	pairStr = "BTCUSD"
	pair = NewCurrencyPairFromString(pairStr)
	actual = pair.Pair()
	expected = CurrencyItem("BTCUSD")
	if actual != expected {
		t.Errorf(
			"Test failed. Pair(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}
}

func TestContains(t *testing.T) {
	pairOne := NewCurrencyPair("BTC", "USD")
	pairTwo := NewCurrencyPair("LTC", "USD")

	var pairs []CurrencyPair
	pairs = append(pairs, pairOne)
	pairs = append(pairs, pairTwo)

	if !Contains(pairs, pairOne) {
		t.Errorf("Test failed. TestContains: Expected pair was not found")
	}

	if Contains(pairs, NewCurrencyPair("ETH", "USD")) {
		t.Errorf("Test failed. TestContains: Non-existant pair was found")
	}
}

func TestFormatPairs(t *testing.T) {
	if len(FormatPairs([]string{""}, "-", "")) > 0 {
		t.Error("Test failed. TestFormatPairs: Empty string returned a valid pair")
	}

	if FormatPairs([]string{"BTC-USD"}, "-", "")[0].Pair().String() != "BTC-USD" {
		t.Error("Test failed. TestFormatPairs: Expected pair was not found")
	}

	if FormatPairs([]string{"BTCUSD"}, "", "BTC")[0].Pair().String() != "BTCUSD" {
		t.Error("Test failed. TestFormatPairs: Expected pair was not found")
	}

	if FormatPairs([]string{"ETHUSD"}, "", "")[0].Pair().String() != "ETHUSD" {
		t.Error("Test failed. TestFormatPairs: Expected pair was not found")
	}
}

func TestCopyPairFormat(t *testing.T) {
	pairOne := NewCurrencyPair("BTC", "USD")
	pairOne.Delimiter = "-"
	pairTwo := NewCurrencyPair("LTC", "USD")

	var pairs []CurrencyPair
	pairs = append(pairs, pairOne)
	pairs = append(pairs, pairTwo)

	testPair := NewCurrencyPair("BTC", "USD")
	testPair.Delimiter = "~"

	result := CopyPairFormat(testPair, pairs)
	if result.Pair().String() != "BTC-USD" {
		t.Error("Test failed. TestCopyPairFormat: Expected pair was not found")
	}

	result = CopyPairFormat(NewCurrencyPair("ETH", "USD"), pairs)
	if result.Pair().String() != "" {
		t.Error("Test failed. TestCopyPairFormat: Unexpected non empty pair returned")
	}
}
