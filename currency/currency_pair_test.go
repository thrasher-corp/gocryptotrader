package currency

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/common"
)

func TestPairsString(t *testing.T) {
	pairs := NewCurrencyPairListFromString([]string{"btc_usd", "btc_aud", "btc_ltc"})
	expected := []string{"btc_usd", "btc_aud", "btc_ltc"}

	for i, p := range pairs {
		if p.String() != expected[i] {
			t.Errorf("Test Failed - Pairs String() error expected %s but received %s",
				expected, p.String())
		}
	}
}

func TestPairsJoin(t *testing.T) {
	pairs := NewCurrencyPairListFromString([]string{"btc_usd", "btc_aud", "btc_ltc"})
	expected := "btc_usd,btc_aud,btc_ltc"

	if pairs.Join() != expected {
		t.Errorf("Test Failed - Pairs Join() error expected %s but received %s",
			expected, pairs.Join())
	}
}

func TestPairsFormat(t *testing.T) {
	pairs := NewCurrencyPairListFromString([]string{"btc_usd", "btc_aud", "btc_ltc"})

	expected := "BTC-USD,BTC-AUD,BTC-LTC"
	if pairs.Format("-", "", true).Join() != expected {
		t.Errorf("Test Failed - Pairs Join() error expected %s but received %s",
			expected, pairs.Format("-", "", true).Join())
	}

	expected = "btc:usd,btc:aud,btc:ltc"
	if pairs.Format(":", "", false).Join() != expected {
		t.Errorf("Test Failed - Pairs Join() error expected %s but received %s",
			expected, pairs.Format("-", "", true).Join())
	}

	expected = "btc:krw,btc:krw,btc:krw"
	if pairs.Format(":", "krw", false).Join() != expected {
		t.Errorf("Test Failed - Pairs Join() error expected %s but received %s",
			expected, pairs.Format("-", "", true).Join())
	}
}

func TestPairsUnmarshalJSON(t *testing.T) {
	var unmarshalHere Pairs
	configPairs := "btc_usd,btc_aud,btc_ltc"

	encoded, err := common.JSONEncode(configPairs)
	if err != nil {
		t.Fatal("Test Failed - Pairs UnmarshalJSON() error", err)
	}

	err = common.JSONDecode(encoded, &unmarshalHere)
	if err != nil {
		t.Fatal("Test Failed - Pairs UnmarshalJSON() error", err)
	}

	err = common.JSONDecode(encoded, &unmarshalHere)
	if err != nil {
		t.Fatal("Test Failed - Pairs UnmarshalJSON() error", err)
	}

	if unmarshalHere.Join() != configPairs {
		t.Errorf("Test Failed - Pairs UnmarshalJSON() error expected %s but received %s",
			configPairs, unmarshalHere.Join())
	}
}

func TestPairsMarshalJSON(t *testing.T) {
	quickstruct := struct {
		Pairs Pairs `json:"soManyPairs"`
	}{
		Pairs: NewCurrencyPairListFromString([]string{"btc_usd", "btc_aud", "btc_ltc"}),
	}

	encoded, err := common.JSONEncode(quickstruct)
	if err != nil {
		t.Fatal("Test Failed - Pairs MarshalJSON() error", err)
	}

	expected := `{"soManyPairs":"btc_usd,btc_aud,btc_ltc"}`
	if string(encoded) != expected {
		t.Errorf("Test Failed - Pairs MarshalJSON() error expected %s but received %s",
			expected, string(encoded))
	}
}

func TestPairsUpper(t *testing.T) {
	pairs := NewCurrencyPairListFromString([]string{"btc_usd", "btc_aud", "btc_ltc"})
	expected := "BTC_USD,BTC_AUD,BTC_LTC"

	if pairs.Upper().Join() != expected {
		t.Errorf("Test Failed - Pairs Join() error expected %s but received %s",
			expected, pairs.Join())
	}
}

func TestLower(t *testing.T) {
	t.Parallel()
	pair := Code("BTCUSD")
	actual := pair.Lower()
	expected := Code("btcusd")
	if actual != expected {
		t.Errorf("Test failed. Lower(): %s was not equal to expected value: %s",
			actual, expected)
	}
}

func TestUpper(t *testing.T) {
	t.Parallel()
	pair := Code("btcusd")
	actual := pair.Upper()
	expected := Code("BTCUSD")
	if actual != expected {
		t.Errorf("Test failed. Upper(): %s was not equal to expected value: %s",
			actual, expected)
	}
}

func TestPairUnmarshalJSON(t *testing.T) {
	var unmarshalHere Pair
	configPair := "btc_usd"

	encoded, err := common.JSONEncode(configPair)
	if err != nil {
		t.Fatal("Test Failed - Pair UnmarshalJSON() error", err)
	}

	err = common.JSONDecode(encoded, &unmarshalHere)
	if err != nil {
		t.Fatal("Test Failed - Pair UnmarshalJSON() error", err)
	}

	err = common.JSONDecode(encoded, &unmarshalHere)
	if err != nil {
		t.Fatal("Test Failed - Pair UnmarshalJSON() error", err)
	}

	if unmarshalHere.String() != configPair {
		t.Errorf("Test Failed - Pairs UnmarshalJSON() error expected %s but received %s",
			configPair, unmarshalHere)
	}
}

func TestPairMarshalJSON(t *testing.T) {
	quickstruct := struct {
		Pair Pair `json:"superPair"`
	}{
		Pair{Base: BTC, Quote: USD, Delimiter: "-"},
	}

	encoded, err := common.JSONEncode(quickstruct)
	if err != nil {
		t.Fatal("Test Failed - Pair MarshalJSON() error", err)
	}

	expected := `{"superPair":"BTC-USD"}`
	if string(encoded) != expected {
		t.Errorf("Test Failed - Pair MarshalJSON() error expected %s but received %s",
			expected, string(encoded))
	}
}

func TestString(t *testing.T) {
	t.Parallel()
	pair := NewCurrencyPair("BTC", "USD")
	actual := "BTCUSD"
	expected := pair.String()
	if actual != expected {
		t.Errorf("Test failed. String(): %s was not equal to expected value: %s",
			actual, expected)
	}
}

func TestFirstCurrency(t *testing.T) {
	t.Parallel()
	pair := NewCurrencyPair("BTC", "USD")
	actual := pair.Base
	expected := Code("BTC")
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
	actual := pair.Quote
	expected := Code("USD")
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
	actual := pair.String()
	expected := "BTCUSD"
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
	actual := pair.String()
	expected := "BTC-USD"
	if actual != expected {
		t.Errorf(
			"Test failed. Pair(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}

	actual = pair.Display("", false).String()
	expected = "btcusd"
	if actual != expected {
		t.Errorf(
			"Test failed. Pair(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}

	actual = pair.Display("~", true).String()
	expected = "BTC~USD"
	if actual != expected {
		t.Errorf(
			"Test failed. Pair(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}
}

func TestEquall(t *testing.T) {
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

	secondPair.Quote = "ETH"
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
	expected = false
	if actual != expected {
		t.Errorf(
			"Test failed. Equal(): %v was not equal to expected value: %v",
			actual, expected,
		)
	}
}

func TestEqualIncludeReciprocal(t *testing.T) {
	t.Parallel()
	pair := NewCurrencyPair("BTC", "USD")
	secondPair := NewCurrencyPair("btc", "uSd")
	actual := pair.EqualIncludeReciprocal(secondPair)
	expected := true
	if actual != expected {
		t.Errorf(
			"Test failed. Equal(): %v was not equal to expected value: %v",
			actual, expected,
		)
	}

	secondPair.Quote = "ETH"
	actual = pair.EqualIncludeReciprocal(secondPair)
	expected = false
	if actual != expected {
		t.Errorf(
			"Test failed. Equal(): %v was not equal to expected value: %v",
			actual, expected,
		)
	}

	secondPair = NewCurrencyPair("USD", "BTC")
	actual = pair.EqualIncludeReciprocal(secondPair)
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
	actual := pair.Swap().String()
	expected := "USDBTC"
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

	var p Pair
	if !p.Empty() {
		t.Error("Test failed. Empty() returned true when the pair wasn't initialised")
	}
}

func TestNewCurrencyPair(t *testing.T) {
	t.Parallel()
	pair := NewCurrencyPair("BTC", "USD")
	actual := pair.String()
	expected := "BTCUSD"
	if actual != expected {
		t.Errorf(
			"Test failed. Pair(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}
}

func TestNewCurrencyPairWithDelimiter(t *testing.T) {
	t.Parallel()
	pair := NewCurrencyPairWithDelimiter("BTC", "USD", "-test-")
	actual := pair.String()
	expected := "BTC-test-USD"
	if actual != expected {
		t.Errorf(
			"Test failed. Pair(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}

	pair = NewCurrencyPairWithDelimiter("BTC", "USD", "")
	actual = pair.String()
	expected = "BTCUSD"
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
	actual := pair.String()
	expected := "BTC-USD"
	if actual != expected {
		t.Errorf(
			"Test failed. Pair(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}

	actual = pair.Delimiter
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
	actual := pair.String()

	expected := "BTC-USD"
	if actual != expected {
		t.Errorf(
			"Test failed. Pair(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}

	currency = "DOGEBTC"

	pair = NewCurrencyPairFromIndex(currency, index)
	pair.Delimiter = "-"
	actual = pair.String()

	expected = "DOGE-BTC"
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
	actual := pair.String()
	expected := "BTC-USD"
	if actual != expected {
		t.Errorf(
			"Test failed. Pair(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}

	pairStr = "BTCUSD"
	pair = NewCurrencyPairFromString(pairStr)
	actual = pair.String()
	expected = "BTCUSD"
	if actual != expected {
		t.Errorf(
			"Test failed. Pair(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}
}

func TestContains(t *testing.T) {
	pairOne := NewCurrencyPair("BTC", "USD")

	var pairs []Pair
	pairs = append(pairs, pairOne, NewCurrencyPair("LTC", "USD"))

	if !PairsContain(pairs, pairOne, true) {
		t.Errorf("Test failed. TestContains: Expected pair was not found")
	}

	if PairsContain(pairs, NewCurrencyPair("ETH", "USD"), false) {
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
	var pairs []Pair
	pairs = append(pairs, NewCurrencyPair("BTC", "USD"),
		NewCurrencyPair("LTC", "USD"),
		NewCurrencyPair("LTC", "USDT"))

	pairs = RemovePairsByFilter(pairs, "USDT")
	if PairsContain(pairs, NewCurrencyPair("LTC", "USDT"), true) {
		t.Error("Test failed. TestRemovePairsByFilter unexpected result")
	}
}

func TestFormatPairs(t *testing.T) {
	if len(FormatPairs([]string{""}, "-", "")) > 0 {
		t.Error("Test failed. TestFormatPairs: Empty string returned a valid pair")
	}

	if FormatPairs([]string{"BTC-USD"}, "-", "")[0].String() != "BTC-USD" {
		t.Error("Test failed. TestFormatPairs: Expected pair was not found")
	}

	if FormatPairs([]string{"BTCUSD"}, "", "BTC")[0].String() != "BTCUSD" {
		t.Error("Test failed. TestFormatPairs: Expected pair was not found")
	}

	if FormatPairs([]string{"ETHUSD"}, "", "")[0].String() != "ETHUSD" {
		t.Error("Test failed. TestFormatPairs: Expected pair was not found")
	}
}

func TestCopyPairFormat(t *testing.T) {
	pairOne := NewCurrencyPair("BTC", "USD")
	pairOne.Delimiter = "-"

	var pairs []Pair
	pairs = append(pairs, pairOne, NewCurrencyPair("LTC", "USD"))

	testPair := NewCurrencyPair("BTC", "USD")
	testPair.Delimiter = "~"

	result := CopyPairFormat(testPair, pairs, false)
	if result.String() != "BTC-USD" {
		t.Error("Test failed. TestCopyPairFormat: Expected pair was not found")
	}

	result = CopyPairFormat(NewCurrencyPair("ETH", "USD"), pairs, true)
	if result.String() != "" {
		t.Error("Test failed. TestCopyPairFormat: Unexpected non empty pair returned")
	}
}

func TestFindPairDifferences(t *testing.T) {
	pairList := NewCurrencyPairListFromString([]string{"BTC-USD", "ETH-USD", "LTC-USD"})

	// Test new pair update
	newPairs, removedPairs := FindPairDifferences(pairList,
		NewCurrencyPairListFromString([]string{"DASH-USD"}))
	if len(newPairs) != 1 && len(removedPairs) != 3 {
		t.Error("Test failed. TestFindPairDifferences: Unexpected values")
	}

	// Test that we don't allow empty strings for new pairs
	newPairs, removedPairs = FindPairDifferences(pairList,
		NewCurrencyPairListFromString([]string{""}))
	if len(newPairs) != 0 && len(removedPairs) != 3 {
		t.Error("Test failed. TestFindPairDifferences: Unexpected values")
	}

	// Test that we don't allow empty strings for new pairs
	newPairs, removedPairs = FindPairDifferences(NewCurrencyPairListFromString([]string{""}),
		pairList)
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
	var pairs []Pair
	pairs = append(pairs, NewCurrencyPair("BTC", "USD"))

	expected := []string{"BTCUSD"}
	actual := PairsToStringArray(pairs)

	if actual[0] != expected[0] {
		t.Error("Test failed. TestPairsToStringArray: Unexpected values")
	}
}

func TestRandomPairFromPairs(t *testing.T) {
	// Test that an empty pairs array returns an empty currency pair
	result := RandomPairFromPairs([]Pair{})
	if !result.Empty() {
		t.Error("Test failed. TestRandomPairFromPairs: Unexpected values")
	}

	// Test that a populated pairs array returns a non-empty currency pair
	var pairs []Pair
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
		p := RandomPairFromPairs(pairs).String()
		_, ok := expectedResults[p]
		if !ok {
			expectedResults[p] = true
		}
	}

	for x := range pairs {
		_, ok := expectedResults[pairs[x].String()]
		if !ok {
			t.Error("Test failed. TestRandomPairFromPairs: Unexpected values")
		}
	}
}
