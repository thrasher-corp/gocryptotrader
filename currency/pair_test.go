package currency

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/common"
)

const (
	defaultPair           = "BTCUSD"
	defaultPairWDelimiter = "BTC-USD"
)

func TestLower(t *testing.T) {
	t.Parallel()
	pair := NewPairFromString(defaultPair)
	actual := pair.Lower()
	expected := NewPairFromString(defaultPair).Lower()
	if actual != expected {
		t.Errorf("Test failed. Lower(): %s was not equal to expected value: %s",
			actual, expected)
	}
}

func TestUpper(t *testing.T) {
	t.Parallel()
	pair := NewPairFromString(defaultPair)
	actual := pair.Upper()
	expected := NewPairFromString(defaultPair)
	if actual != expected {
		t.Errorf("Test failed. Upper(): %s was not equal to expected value: %s",
			actual, expected)
	}
}

func TestPairUnmarshalJSON(t *testing.T) {
	var unmarshalHere Pair
	configPair := NewPairDelimiter("btc_usd", "_")

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

	if !unmarshalHere.Equal(configPair) {
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

func TestIsCryptoPair(t *testing.T) {
	if !NewPair(BTC, LTC).IsCryptoPair() {
		t.Error("Test Failed. TestIsCryptoPair. Expected true result")
	}

	if NewPair(BTC, USD).IsCryptoPair() {
		t.Error("Test Failed. TestIsCryptoPair. Expected false result")
	}
}

func TestIsCryptoFiatPair(t *testing.T) {
	if !NewPair(BTC, USD).IsCryptoFiatPair() {
		t.Error("Test Failed. TestIsCryptoPair. Expected true result")
	}

	if NewPair(BTC, LTC).IsCryptoFiatPair() {
		t.Error("Test Failed. TestIsCryptoPair. Expected false result")
	}
}

func TestIsFiatPair(t *testing.T) {
	if !NewPair(AUD, USD).IsFiatPair() {
		t.Error("Test Failed. TestIsFiatPair. Expected true result")
	}

	if NewPair(BTC, AUD).IsFiatPair() {
		t.Error("Test Failed. TestIsFiatPair. Expected false result")
	}
}

func TestString(t *testing.T) {
	t.Parallel()
	pair := NewPair(BTC, USD)
	actual := defaultPair
	expected := pair.String()
	if actual != expected {
		t.Errorf("Test failed. String(): %s was not equal to expected value: %s",
			actual, expected)
	}
}

func TestFirstCurrency(t *testing.T) {
	t.Parallel()
	pair := NewPair(BTC, USD)
	actual := pair.Base
	expected := BTC
	if actual != expected {
		t.Errorf(
			"Test failed. GetFirstCurrency(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}
}

func TestSecondCurrency(t *testing.T) {
	t.Parallel()
	pair := NewPair(BTC, USD)
	actual := pair.Quote
	expected := USD
	if actual != expected {
		t.Errorf(
			"Test failed. GetSecondCurrency(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}
}

func TestPair(t *testing.T) {
	t.Parallel()
	pair := NewPair(BTC, USD)
	actual := pair.String()
	expected := defaultPair
	if actual != expected {
		t.Errorf(
			"Test failed. Pair(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}
}

func TestDisplay(t *testing.T) {
	t.Parallel()
	pair := NewPairDelimiter(defaultPairWDelimiter, "-")
	actual := pair.String()
	expected := defaultPairWDelimiter
	if actual != expected {
		t.Errorf(
			"Test failed. Pair(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}

	actual = pair.Format("", false).String()
	expected = "btcusd"
	if actual != expected {
		t.Errorf(
			"Test failed. Pair(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}

	actual = pair.Format("~", true).String()
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
	pair := NewPair(BTC, USD)
	secondPair := NewPair(BTC, USD)
	actual := pair.Equal(secondPair)
	expected := true
	if actual != expected {
		t.Errorf(
			"Test failed. Equal(): %v was not equal to expected value: %v",
			actual, expected,
		)
	}

	secondPair.Quote = ETH
	actual = pair.Equal(secondPair)
	expected = false
	if actual != expected {
		t.Errorf(
			"Test failed. Equal(): %v was not equal to expected value: %v",
			actual, expected,
		)
	}

	secondPair = NewPair(USD, BTC)
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
	pair := NewPair(BTC, USD)
	secondPair := NewPair(BTC, USD)
	actual := pair.EqualIncludeReciprocal(secondPair)
	expected := true
	if actual != expected {
		t.Errorf(
			"Test failed. Equal(): %v was not equal to expected value: %v",
			actual, expected,
		)
	}

	secondPair.Quote = ETH
	actual = pair.EqualIncludeReciprocal(secondPair)
	expected = false
	if actual != expected {
		t.Errorf(
			"Test failed. Equal(): %v was not equal to expected value: %v",
			actual, expected,
		)
	}

	secondPair = NewPair(USD, BTC)
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
	pair := NewPair(BTC, USD)
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
	pair := NewPair(BTC, USD)
	if pair.IsEmpty() {
		t.Error("Test failed. Empty() returned true when the pair was initialised")
	}

	p := NewPair(NewCode(""), NewCode(""))
	if !p.IsEmpty() {
		t.Error("Test failed. Empty() returned true when the pair wasn't initialised")
	}
}

func TestNewPair(t *testing.T) {
	t.Parallel()
	pair := NewPair(BTC, USD)
	actual := pair.String()
	expected := defaultPair
	if actual != expected {
		t.Errorf(
			"Test failed. Pair(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}
}

func TestNewPairWithDelimiter(t *testing.T) {
	t.Parallel()
	pair := NewPairWithDelimiter("BTC", "USD", "-test-")
	actual := pair.String()
	expected := "BTC-test-USD"
	if actual != expected {
		t.Errorf(
			"Test failed. Pair(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}

	pair = NewPairWithDelimiter("BTC", "USD", "")
	actual = pair.String()
	expected = defaultPair
	if actual != expected {
		t.Errorf(
			"Test failed. Pair(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}
}

func TestNewPairDelimiter(t *testing.T) {
	t.Parallel()
	pair := NewPairDelimiter(defaultPairWDelimiter, "-")
	actual := pair.String()
	expected := defaultPairWDelimiter
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

// TestNewPairFromIndex returns a CurrencyPair via a currency string and
// specific index
func TestNewPairFromIndex(t *testing.T) {
	t.Parallel()
	currency := defaultPair
	index := "BTC"

	pair, err := NewPairFromIndex(currency, index)
	if err != nil {
		t.Error("test failed - NewPairFromIndex() error", err)
	}

	pair.Delimiter = "-"
	actual := pair.String()

	expected := defaultPairWDelimiter
	if actual != expected {
		t.Errorf(
			"Test failed. Pair(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}

	currency = "DOGEBTC"

	pair, err = NewPairFromIndex(currency, index)
	if err != nil {
		t.Error("test failed - NewPairFromIndex() error", err)
	}

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

func TestNewPairFromString(t *testing.T) {
	t.Parallel()
	pairStr := defaultPairWDelimiter
	pair := NewPairFromString(pairStr)
	actual := pair.String()
	expected := defaultPairWDelimiter
	if actual != expected {
		t.Errorf(
			"Test failed. Pair(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}

	pairStr = defaultPair
	pair = NewPairFromString(pairStr)
	actual = pair.String()
	expected = defaultPair
	if actual != expected {
		t.Errorf(
			"Test failed. Pair(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}
}

func TestNewPairFromFormattedPairs(t *testing.T) {
	t.Parallel()
	pairs := Pairs{
		NewPairDelimiter("BTC-USDT", "-"),
		NewPairDelimiter("LTC-USD", "-"),
	}

	p := NewPairFromFormattedPairs("BTCUSDT", pairs, PairFormat{Uppercase: true})
	if p.String() != "BTC-USDT" {
		t.Error("Test failed. TestNewPairFromFormattedPairs: Expected currency was not found")
	}

	p = NewPairFromFormattedPairs("btcusdt", pairs, PairFormat{Uppercase: false})
	if p.String() != "BTC-USDT" {
		t.Error("Test failed. TestNewPairFromFormattedPairs: Expected currency was not found")
	}

	// Now a wrong one, will default to NewPairFromString
	p = NewPairFromFormattedPairs("ethusdt", pairs, PairFormat{})
	if p.String() != "ethusdt" && p.Base.String() != "eth" {
		t.Error("Test failed. TestNewPairFromFormattedPairs: Expected currency was not found")
	}
}

func TestContainsCurrency(t *testing.T) {
	p := NewPair(BTC, USD)

	if !p.ContainsCurrency(BTC) {
		t.Error("Test failed. TestContainsCurrency: Expected currency was not found")
	}

	if p.ContainsCurrency(ETH) {
		t.Error("Test failed. TestContainsCurrency: Non-existent currency was found")
	}
}

func TestFormatPairs(t *testing.T) {
	newP, err := FormatPairs([]string{""}, "-", "")
	if err != nil {
		t.Error("Test Failed - FormatPairs() error", err)
	}

	if len(newP) > 0 {
		t.Error("Test failed. TestFormatPairs: Empty string returned a valid pair")
	}

	newP, err = FormatPairs([]string{defaultPairWDelimiter}, "-", "")
	if err != nil {
		t.Error("Test Failed - FormatPairs() error", err)
	}

	if newP[0].String() != defaultPairWDelimiter {
		t.Error("Test failed. TestFormatPairs: Expected pair was not found")
	}

	newP, err = FormatPairs([]string{defaultPair}, "", "BTC")
	if err != nil {
		t.Error("Test Failed - FormatPairs() error", err)
	}

	if newP[0].String() != defaultPair {
		t.Error("Test failed. TestFormatPairs: Expected pair was not found")
	}
	newP, err = FormatPairs([]string{"ETHUSD"}, "", "")
	if err != nil {
		t.Error("Test Failed - FormatPairs() error", err)
	}

	if newP[0].String() != "ETHUSD" {
		t.Error("Test failed. TestFormatPairs: Expected pair was not found")
	}
}

func TestCopyPairFormat(t *testing.T) {
	pairOne := NewPair(BTC, USD)
	pairOne.Delimiter = "-"

	var pairs []Pair
	pairs = append(pairs, pairOne, NewPair(LTC, USD))

	testPair := NewPair(BTC, USD)
	testPair.Delimiter = "~"

	result := CopyPairFormat(testPair, pairs, false)
	if result.String() != defaultPairWDelimiter {
		t.Error("Test failed. TestCopyPairFormat: Expected pair was not found")
	}

	result = CopyPairFormat(NewPair(ETH, USD), pairs, true)
	if result.String() != "" {
		t.Error("Test failed. TestCopyPairFormat: Unexpected non empty pair returned")
	}
}

func TestFindPairDifferences(t *testing.T) {
	pairList := NewPairsFromStrings([]string{defaultPairWDelimiter, "ETH-USD", "LTC-USD"})

	// Test new pair update
	newPairs, removedPairs := pairList.FindDifferences(NewPairsFromStrings([]string{"DASH-USD"}))
	if len(newPairs) != 1 && len(removedPairs) != 3 {
		t.Error("Test failed. TestFindPairDifferences: Unexpected values")
	}

	// Test that we don't allow empty strings for new pairs
	newPairs, removedPairs = pairList.FindDifferences(NewPairsFromStrings([]string{""}))
	if len(newPairs) != 0 && len(removedPairs) != 3 {
		t.Error("Test failed. TestFindPairDifferences: Unexpected values")
	}

	// Test that we don't allow empty strings for new pairs
	newPairs, removedPairs = NewPairsFromStrings([]string{""}).FindDifferences(pairList)
	if len(newPairs) != 3 && len(removedPairs) != 0 {
		t.Error("Test failed. TestFindPairDifferences: Unexpected values")
	}

	// Test that the supplied pair lists are the same, so
	// no newPairs or removedPairs
	newPairs, removedPairs = pairList.FindDifferences(pairList)
	if len(newPairs) != 0 && len(removedPairs) != 0 {
		t.Error("Test failed. TestFindPairDifferences: Unexpected values")
	}
}

func TestPairsToStringArray(t *testing.T) {
	var pairs Pairs
	pairs = append(pairs, NewPair(BTC, USD))

	expected := []string{defaultPair}
	actual := pairs.Strings()

	if actual[0] != expected[0] {
		t.Error("Test failed. TestPairsToStringArray: Unexpected values")
	}
}

func TestRandomPairFromPairs(t *testing.T) {
	// Test that an empty pairs array returns an empty currency pair
	var emptyPairs Pairs
	result := emptyPairs.GetRandomPair()
	if !result.IsEmpty() {
		t.Error("Test failed. TestRandomPairFromPairs: Unexpected values")
	}

	// Test that a populated pairs array returns a non-empty currency pair
	var pairs Pairs
	pairs = append(pairs, NewPair(BTC, USD))
	result = pairs.GetRandomPair()

	if result.IsEmpty() {
		t.Error("Test failed. TestRandomPairFromPairs: Unexpected values")
	}

	// Test that a populated pairs array over a number of attempts returns ALL
	// currency pairs
	pairs = append(pairs, NewPair(ETH, USD))
	expectedResults := make(map[string]bool)
	for i := 0; i < 50; i++ {
		p := pairs.GetRandomPair().String()
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

func TestIsInvalid(t *testing.T) {
	p := NewPair(LTC, LTC)
	if !p.IsInvalid() {
		t.Error("Test Failed - IsInvalid() error expect true but received false")
	}
}
