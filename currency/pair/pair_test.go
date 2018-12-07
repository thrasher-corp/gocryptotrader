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

func TestFirstCurrency(t *testing.T) {
	t.Parallel()
	pair := NewCurrencyPair("BTC", "USD")
	actual := pair.FirstCurrency
	expected := CurrencyItem("BTC")
	if actual != expected {
		t.Errorf(
			"Test failed. GetFirstCurrency(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}
}

func TestSecondCurrency(t *testing.T) {
	t.Parallel()
	pair := NewCurrencyPair("BTC", "USD")
	actual := pair.SecondCurrency
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
	actual := pair.Equal(secondPair, false)
	expected := true
	if actual != expected {
		t.Errorf(
			"Test failed. Equal(): %v was not equal to expected value: %v",
			actual, expected,
		)
	}

	secondPair.SecondCurrency = "ETH"
	actual = pair.Equal(secondPair, false)
	expected = false
	if actual != expected {
		t.Errorf(
			"Test failed. Equal(): %v was not equal to expected value: %v",
			actual, expected,
		)
	}

	secondPair = NewCurrencyPair("USD", "BTC")
	actual = pair.Equal(secondPair, false)
	expected = true
	if actual != expected {
		t.Errorf(
			"Test failed. Equal(): %v was not equal to expected value: %v",
			actual, expected,
		)
	}
}

func TestSwap(t *testing.T) {
	t.Parallel()
	pair := NewCurrencyPair("BTC", "USD")
	actual := pair.Swap().Pair()
	expected := CurrencyItem("USDBTC")
	if actual != expected {
		t.Errorf(
			"Test failed. TestSwap: %s was not equal to expected value: %s",
			actual, expected,
		)
	}
}

func TestEmpty(t *testing.T) {
	t.Parallel()
	pair := NewCurrencyPair("BTC", "USD")
	if pair.Empty() {
		t.Error("Test failed. Empty() returned true when the pair was initialised")
	}

	var p CurrencyPair
	if !p.Empty() {
		t.Error("Test failed. Empty() returned true when the pair wasn't initialised")
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

// TestNewCurrencyPairFromIndex returns a CurrencyPair via a currency string and
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

	if !Contains(pairs, pairOne, true) {
		t.Errorf("Test failed. TestContains: Expected pair was not found")
	}

	if Contains(pairs, NewCurrencyPair("ETH", "USD"), false) {
		t.Errorf("Test failed. TestContains: Non-existent pair was found")
	}
}

func TestContainsCurrency(t *testing.T) {
	p := NewCurrencyPair("BTC", "USD")

	if !ContainsCurrency(p, "BTC") {
		t.Error("Test failed. TestContainsCurrency: Expected currency was not found")
	}

	if ContainsCurrency(p, "ETH") {
		t.Error("Test failed. TestContainsCurrency: Non-existent currency was found")
	}
}

func TestRemovePairsByFilter(t *testing.T) {
	var pairs []CurrencyPair
	pairs = append(pairs, NewCurrencyPair("BTC", "USD"))
	pairs = append(pairs, NewCurrencyPair("LTC", "USD"))
	pairs = append(pairs, NewCurrencyPair("LTC", "USDT"))

	pairs = RemovePairsByFilter(pairs, "USDT")
	if Contains(pairs, NewCurrencyPair("LTC", "USDT"), true) {
		t.Error("Test failed. TestRemovePairsByFilter unexpected result")
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

	result := CopyPairFormat(testPair, pairs, false)
	if result.Pair().String() != "BTC-USD" {
		t.Error("Test failed. TestCopyPairFormat: Expected pair was not found")
	}

	result = CopyPairFormat(NewCurrencyPair("ETH", "USD"), pairs, true)
	if result.Pair().String() != "" {
		t.Error("Test failed. TestCopyPairFormat: Unexpected non empty pair returned")
	}
}

func TestFindPairDifferences(t *testing.T) {
	pairList := []string{"BTC-USD", "ETH-USD", "LTC-USD"}

	// Test new pair update
	newPairs, removedPairs := FindPairDifferences(pairList, []string{"DASH-USD"})
	if len(newPairs) != 1 && len(removedPairs) != 3 {
		t.Error("Test failed. TestFindPairDifferences: Unexpected values")
	}

	// Test that we don't allow empty strings for new pairs
	newPairs, removedPairs = FindPairDifferences(pairList, []string{""})
	if len(newPairs) != 0 && len(removedPairs) != 3 {
		t.Error("Test failed. TestFindPairDifferences: Unexpected values")
	}

	// Test that we don't allow empty strings for new pairs
	newPairs, removedPairs = FindPairDifferences([]string{""}, pairList)
	if len(newPairs) != 3 && len(removedPairs) != 0 {
		t.Error("Test failed. TestFindPairDifferences: Unexpected values")
	}

	// Test that the supplied pair lists are the same, so
	// no newPairs or removedPairs
	newPairs, removedPairs = FindPairDifferences(pairList, pairList)
	if len(newPairs) != 0 && len(removedPairs) != 0 {
		t.Error("Test failed. TestFindPairDifferences: Unexpected values")
	}
}

func TestPairsToStringArray(t *testing.T) {
	var pairs []CurrencyPair
	pairs = append(pairs, NewCurrencyPair("BTC", "USD"))

	expected := []string{"BTCUSD"}
	actual := PairsToStringArray(pairs)

	if actual[0] != expected[0] {
		t.Error("Test failed. TestPairsToStringArray: Unexpected values")
	}
}

func TestRandomPairFromPairs(t *testing.T) {
	// Test that an empty pairs array returns an empty currency pair
	result := RandomPairFromPairs([]CurrencyPair{})
	if !result.Empty() {
		t.Error("Test failed. TestRandomPairFromPairs: Unexpected values")
	}

	// Test that a populated pairs array returns a non-empty currency pair
	var pairs []CurrencyPair
	pairs = append(pairs, NewCurrencyPair("BTC", "USD"))
	result = RandomPairFromPairs(pairs)

	if result.Empty() {
		t.Error("Test failed. TestRandomPairFromPairs: Unexpected values")
	}

	// Test that a populated pairs array over a number of attempts returns ALL
	// currency pairs
	pairs = append(pairs, NewCurrencyPair("ETH", "USD"))
	expectedResults := make(map[string]bool)
	for i := 0; i < 50; i++ {
		p := RandomPairFromPairs(pairs).Pair().String()
		_, ok := expectedResults[p]
		if !ok {
			expectedResults[p] = true
		}
	}

	for x := range pairs {
		_, ok := expectedResults[pairs[x].Pair().String()]
		if !ok {
			t.Error("Test failed. TestRandomPairFromPairs: Unexpected values")
		}
	}
}
