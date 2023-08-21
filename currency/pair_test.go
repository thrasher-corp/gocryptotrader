package currency

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
)

const (
	defaultPair           = "BTCUSD"
	defaultPairWDelimiter = "BTC-USD"
)

func TestLower(t *testing.T) {
	t.Parallel()
	pair, err := NewPairFromString(defaultPair)
	if err != nil {
		t.Fatal(err)
	}
	expected, err := NewPairFromString(defaultPair)
	if err != nil {
		t.Fatal(err)
	}
	if actual := pair.Lower(); actual.String() != expected.Lower().String() {
		t.Errorf("Lower(): %s was not equal to expected value: %s",
			actual,
			expected.Lower())
	}
}

func TestUpper(t *testing.T) {
	t.Parallel()
	pair, err := NewPairFromString(defaultPair)
	if err != nil {
		t.Fatal(err)
	}
	expected, err := NewPairFromString(defaultPair)
	if err != nil {
		t.Fatal(err)
	}
	if actual := pair.Upper(); actual.String() != expected.String() {
		t.Errorf("Upper(): %s was not equal to expected value: %s",
			actual, expected)
	}
}

func TestPairUnmarshalJSON(t *testing.T) {
	var unmarshalHere Pair
	configPair, err := NewPairDelimiter("btc_usd", "_")
	if err != nil {
		t.Fatal(err)
	}

	encoded, err := json.Marshal(configPair)
	if err != nil {
		t.Fatal("Pair UnmarshalJSON() error", err)
	}

	err = json.Unmarshal(encoded, &unmarshalHere)
	if err != nil {
		t.Fatal("Pair UnmarshalJSON() error", err)
	}

	err = json.Unmarshal(encoded, &unmarshalHere)
	if err != nil {
		t.Fatal("Pair UnmarshalJSON() error", err)
	}

	if !unmarshalHere.Equal(configPair) {
		t.Errorf("Pair UnmarshalJSON() error expected %s but received %s",
			configPair, unmarshalHere)
	}

	encoded, err = json.Marshal(EMPTYPAIR)
	if err != nil {
		t.Fatal(err)
	}
	err = json.Unmarshal(encoded, &unmarshalHere)
	if err != nil {
		t.Fatal("Pair UnmarshalJSON() error", err)
	}
	err = json.Unmarshal([]byte("null"), &unmarshalHere)
	if err != nil {
		t.Fatal("Pair UnmarshalJSON() error", err)
	}
	if unmarshalHere != EMPTYPAIR {
		t.Fatalf("Expected EMPTYPAIR got: %s", unmarshalHere)
	}
}

func TestPairMarshalJSON(t *testing.T) {
	quickstruct := struct {
		Pair *Pair `json:"superPair"`
	}{
		&Pair{Base: BTC, Quote: USD, Delimiter: "-"},
	}

	encoded, err := json.Marshal(quickstruct)
	if err != nil {
		t.Fatal("Pair MarshalJSON() error", err)
	}

	expected := `{"superPair":"BTC-USD"}`
	if string(encoded) != expected {
		t.Errorf("Pair MarshalJSON() error expected %s but received %s",
			expected, string(encoded))
	}
}

func TestIsCryptoPair(t *testing.T) {
	if !NewPair(BTC, LTC).IsCryptoPair() {
		t.Error("TestIsCryptoPair. Expected true result")
	}

	if NewPair(BTC, USD).IsCryptoPair() {
		t.Error("TestIsCryptoPair. Expected false result")
	}
}

func TestIsCryptoFiatPair(t *testing.T) {
	if !NewPair(BTC, USD).IsCryptoFiatPair() {
		t.Error("TestIsCryptoPair. Expected true result")
	}

	if NewPair(BTC, LTC).IsCryptoFiatPair() {
		t.Error("TestIsCryptoPair. Expected false result")
	}
}

func TestIsFiatPair(t *testing.T) {
	if !NewPair(AUD, USD).IsFiatPair() {
		t.Error("TestIsFiatPair. Expected true result")
	}

	if NewPair(BTC, AUD).IsFiatPair() {
		t.Error("TestIsFiatPair. Expected false result")
	}
}

func TestIsCryptoStablePair(t *testing.T) {
	if !NewPair(BTC, USDT).IsCryptoStablePair() {
		t.Error("TestIsCryptoStablePair. Expected true result")
	}

	if !NewPair(DAI, USDT).IsCryptoStablePair() {
		t.Error("TestIsCryptoStablePair. Expected true result")
	}

	if NewPair(AUD, USDT).IsCryptoStablePair() {
		t.Error("TestIsCryptoStablePair. Expected false result")
	}
}

func TestIsStablePair(t *testing.T) {
	if !NewPair(USDT, DAI).IsStablePair() {
		t.Error("TestIsStablePair. Expected true result")
	}

	if NewPair(USDT, AUD).IsStablePair() {
		t.Error("TestIsStablePair. Expected false result")
	}

	if NewPair(USDT, LTC).IsStablePair() {
		t.Error("TestIsStablePair. Expected false result")
	}
}

func TestString(t *testing.T) {
	t.Parallel()
	pair := NewPair(BTC, USD)
	if actual, expected := defaultPair, pair.String(); actual != expected {
		t.Errorf("String(): %s was not equal to expected value: %s",
			actual, expected)
	}
}

func TestFirstCurrency(t *testing.T) {
	t.Parallel()
	pair := NewPair(BTC, USD)
	if actual, expected := pair.Base, BTC; !actual.Equal(expected) {
		t.Errorf(
			"GetFirstCurrency(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}
}

func TestSecondCurrency(t *testing.T) {
	t.Parallel()
	pair := NewPair(BTC, USD)
	if actual, expected := pair.Quote, USD; !actual.Equal(expected) {
		t.Errorf(
			"GetSecondCurrency(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}
}

func TestPair(t *testing.T) {
	t.Parallel()
	pair := NewPair(BTC, USD)
	if actual, expected := pair.String(), defaultPair; actual != expected {
		t.Errorf(
			"Pair(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}
}

func TestDisplay(t *testing.T) {
	t.Parallel()
	_, err := NewPairDelimiter(defaultPairWDelimiter, "wow")
	if err == nil {
		t.Fatal("error cannot be nil")
	}
	pair, err := NewPairDelimiter(defaultPairWDelimiter, "-")
	if err != nil {
		t.Fatal(err)
	}
	actual := pair.String()
	expected := defaultPairWDelimiter
	if actual != expected {
		t.Errorf(
			"Pair(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}

	actual = EMPTYFORMAT.Format(pair)
	expected = "btcusd"
	if actual != expected {
		t.Errorf(
			"Pair(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}

	actual = pair.Format(PairFormat{Delimiter: "~", Uppercase: true}).String()
	expected = "BTC~USD"
	if actual != expected {
		t.Errorf(
			"Pair(): %s was not equal to expected value: %s",
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
			"Equal(): %v was not equal to expected value: %v",
			actual, expected,
		)
	}

	secondPair.Quote = ETH
	actual = pair.Equal(secondPair)
	expected = false
	if actual != expected {
		t.Errorf(
			"Equal(): %v was not equal to expected value: %v",
			actual, expected,
		)
	}

	secondPair = NewPair(USD, BTC)
	actual = pair.Equal(secondPair)
	expected = false
	if actual != expected {
		t.Errorf(
			"Equal(): %v was not equal to expected value: %v",
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
			"Equal(): %v was not equal to expected value: %v",
			actual, expected,
		)
	}

	secondPair.Quote = ETH
	actual = pair.EqualIncludeReciprocal(secondPair)
	expected = false
	if actual != expected {
		t.Errorf(
			"Equal(): %v was not equal to expected value: %v",
			actual, expected,
		)
	}

	secondPair = NewPair(USD, BTC)
	actual = pair.EqualIncludeReciprocal(secondPair)
	expected = true
	if actual != expected {
		t.Errorf(
			"Equal(): %v was not equal to expected value: %v",
			actual, expected,
		)
	}
}

func TestSwap(t *testing.T) {
	t.Parallel()
	pair := NewPair(BTC, USD)
	if actual, expected := pair.Swap().String(), "USDBTC"; actual != expected {
		t.Errorf(
			"TestSwap: %s was not equal to expected value: %s",
			actual, expected,
		)
	}
}

func TestEmpty(t *testing.T) {
	t.Parallel()
	pair := NewPair(BTC, USD)
	if pair.IsEmpty() {
		t.Error("Empty() returned true when the pair was initialised")
	}

	p := NewPair(NewCode(""), NewCode(""))
	if !p.IsEmpty() {
		t.Error("Empty() returned true when the pair wasn't initialised")
	}
}

func TestNewPair(t *testing.T) {
	t.Parallel()
	pair := NewPair(BTC, USD)
	if expected, actual := defaultPair, pair.String(); actual != expected {
		t.Errorf(
			"Pair(): %s was not equal to expected value: %s",
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
			"Pair(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}

	pair = NewPairWithDelimiter("BTC", "USD", "")
	actual = pair.String()
	expected = defaultPair
	if actual != expected {
		t.Errorf(
			"Pair(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}
}

func TestNewPairDelimiter(t *testing.T) {
	t.Parallel()
	_, err := NewPairDelimiter("", "")
	if err == nil {
		t.Fatal("error cannot be nil")
	}
	_, err = NewPairDelimiter("BTC_USD", "wow")
	if err == nil {
		t.Fatal("error cannot be nil")
	}

	_, err = NewPairDelimiter("BTC_USD", " ")
	if err == nil {
		t.Fatal("error cannot be nil")
	}

	pair, err := NewPairDelimiter(defaultPairWDelimiter, "-")
	if err != nil {
		t.Fatal(err)
	}
	actual := pair.String()
	expected := defaultPairWDelimiter
	if actual != expected {
		t.Errorf(
			"Pair(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}

	actual = pair.Delimiter
	expected = "-"
	if actual != expected {
		t.Errorf(
			"Delmiter: %s was not equal to expected value: %s",
			actual, expected,
		)
	}

	pair, err = NewPairDelimiter("BTC-MOVE-0626", "-")
	if err != nil {
		t.Fatal(err)
	}
	actual = pair.String()
	expected = "BTC-MOVE-0626"
	if actual != expected {
		t.Errorf(
			"Pair(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}

	pair, err = NewPairDelimiter("fBTC-USDT", "-")
	if err != nil {
		t.Fatal(err)
	}
	actual = pair.String()
	expected = "fbtc-USDT"
	if actual != expected {
		t.Errorf(
			"Pair(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}
}

// TestNewPairFromIndex returns a CurrencyPair via a currency string and
// specific index
func TestNewPairFromIndex(t *testing.T) {
	t.Parallel()
	curr := defaultPair
	index := "BTC"

	pair, err := NewPairFromIndex(curr, index)
	if err != nil {
		t.Error("NewPairFromIndex() error", err)
	}

	pair.Delimiter = "-"
	actual := pair.String()

	expected := defaultPairWDelimiter
	if actual != expected {
		t.Errorf(
			"Pair(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}

	curr = "DOGEBTC"

	pair, err = NewPairFromIndex(curr, index)
	if err != nil {
		t.Error("NewPairFromIndex() error", err)
	}

	pair.Delimiter = "-"
	actual = pair.String()

	expected = "DOGE-BTC"
	if actual != expected {
		t.Errorf(
			"Pair(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}
}

func TestNewPairFromString(t *testing.T) {
	t.Parallel()
	pairStr := defaultPairWDelimiter
	pair, err := NewPairFromString(pairStr)
	if err != nil {
		t.Fatal(err)
	}
	actual := pair.String()
	expected := defaultPairWDelimiter
	if actual != expected {
		t.Errorf(
			"Pair(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}

	pairStr = defaultPair
	pair, err = NewPairFromString(pairStr)
	if err != nil {
		t.Fatal(err)
	}
	actual = pair.String()
	expected = defaultPair
	if actual != expected {
		t.Errorf(
			"Pair(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}
	pairMap := map[string]Pair{
		"BTC_USDT-20230630-45000-C": {Base: NewCode("BTC"), Delimiter: UnderscoreDelimiter, Quote: NewCode("USDT-20230630-45000-C")},
		"BTC-USD-221007":            {Base: NewCode("BTC"), Delimiter: DashDelimiter, Quote: NewCode("USD-221007")},
		"IHT_ETH":                   {Base: NewCode("IHT"), Delimiter: UnderscoreDelimiter, Quote: NewCode("ETH")},
		"BTC-USD-220930-30000-P":    {Base: NewCode("BTC"), Delimiter: DashDelimiter, Quote: NewCode("USD-220930-30000-P")},
		"XBTUSDTM":                  {Base: NewCode("XBT"), Delimiter: "", Quote: NewCode("USDTM")},
		"BTC-PERPETUAL":             {Base: NewCode("BTC"), Delimiter: DashDelimiter, Quote: NewCode("PERPETUAL")},
		"SOL-21OCT22-20-C":          {Base: NewCode("SOL"), Delimiter: DashDelimiter, Quote: NewCode("21OCT22-20-C")},
		"SOL-FS-30DEC22_28OCT22":    {Base: NewCode("SOL"), Delimiter: DashDelimiter, Quote: NewCode("FS-30DEC22_28OCT22")},
	}
	for key, expectedPair := range pairMap {
		pair, err = NewPairFromString(key)
		if err != nil {
			t.Fatal(err)
		}
		if !pair.Equal(expectedPair) || pair.Delimiter != expectedPair.Delimiter {
			t.Errorf("Pair(): %s was not equal to expected value: %s", pair.String(), expectedPair.String())
		}
	}
}

func TestNewPairFromFormattedPairs(t *testing.T) {
	t.Parallel()
	p1, err := NewPairDelimiter("BTC-USDT", "-")
	if err != nil {
		t.Fatal(err)
	}
	p2, err := NewPairDelimiter("LTC-USD", "-")
	if err != nil {
		t.Fatal(err)
	}
	pairs := Pairs{
		p1,
		p2,
	}

	p, err := NewPairFromFormattedPairs("BTCUSDT", pairs, PairFormat{
		Uppercase: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if p.String() != "BTC-USDT" {
		t.Error("TestNewPairFromFormattedPairs: Expected currency was not found")
	}

	p, err = NewPairFromFormattedPairs("btcusdt", pairs, PairFormat{Uppercase: false})
	if err != nil {
		t.Fatal(err)
	}

	if p.String() != "BTC-USDT" {
		t.Error("TestNewPairFromFormattedPairs: Expected currency was not found")
	}

	// Now a wrong one, will default to NewPairFromString
	p, err = NewPairFromFormattedPairs("ethusdt", pairs, EMPTYFORMAT)
	if err != nil {
		t.Fatal(err)
	}

	if p.String() != "ethusdt" && p.Base.String() != "eth" {
		t.Error("TestNewPairFromFormattedPairs: Expected currency was not found")
	}
}

func TestContainsCurrency(t *testing.T) {
	p := NewPair(BTC, USD)

	if !p.Contains(BTC) {
		t.Error("TestContains: Expected currency was not found")
	}

	if p.Contains(ETH) {
		t.Error("TestContains: Non-existent currency was found")
	}
}

func TestFormatPairs(t *testing.T) {
	_, err := FormatPairs([]string{""}, "-", "")
	if !errors.Is(err, errEmptyPairString) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errEmptyPairString)
	}

	newP, err := FormatPairs([]string{defaultPairWDelimiter}, "-", "")
	if err != nil {
		t.Error("FormatPairs() error", err)
	}

	if newP[0].String() != defaultPairWDelimiter {
		t.Error("TestFormatPairs: Expected pair was not found")
	}

	newP, err = FormatPairs([]string{defaultPair}, "", "BTC")
	if err != nil {
		t.Error("FormatPairs() error", err)
	}

	if newP[0].String() != defaultPair {
		t.Error("TestFormatPairs: Expected pair was not found")
	}
	newP, err = FormatPairs([]string{"ETHUSD"}, "", "")
	if err != nil {
		t.Error("FormatPairs() error", err)
	}

	if newP[0].String() != "ETHUSD" {
		t.Error("TestFormatPairs: Expected pair was not found")
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
		t.Error("TestCopyPairFormat: Expected pair was not found")
	}

	np := NewPair(ETH, USD)
	result = CopyPairFormat(np, pairs, true)
	if result.String() != "" {
		t.Error("TestCopyPairFormat: Unexpected non empty pair returned")
	}
}

func TestFindPairDifferences(t *testing.T) {
	pairList, err := NewPairsFromStrings([]string{defaultPairWDelimiter, "ETH-USD", "LTC-USD"})
	if err != nil {
		t.Fatal(err)
	}

	dash, err := NewPairsFromStrings([]string{"DASH-USD"})
	if err != nil {
		t.Fatal(err)
	}

	// Test new pair update
	diff, err := pairList.FindDifferences(dash, PairFormat{Delimiter: DashDelimiter, Uppercase: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.New) != 1 && len(diff.Remove) != 3 && diff.FormatDifference {
		t.Error("TestFindPairDifferences: Unexpected values")
	}

	diff, err = pairList.FindDifferences(Pairs{}, EMPTYFORMAT)
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.New) != 0 && len(diff.Remove) != 3 && !diff.FormatDifference {
		t.Error("TestFindPairDifferences: Unexpected values")
	}

	diff, err = Pairs{}.FindDifferences(pairList, EMPTYFORMAT)
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.New) != 3 && len(diff.Remove) != 0 && diff.FormatDifference {
		t.Error("TestFindPairDifferences: Unexpected values")
	}

	// Test that the supplied pair lists are the same, so
	// no newPairs or removedPairs
	diff, err = pairList.FindDifferences(pairList, PairFormat{Delimiter: DashDelimiter, Uppercase: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.New) != 0 && len(diff.Remove) != 0 && !diff.FormatDifference {
		t.Error("TestFindPairDifferences: Unexpected values")
	}

	_, err = pairList.FindDifferences(Pairs{EMPTYPAIR}, EMPTYFORMAT)
	if !errors.Is(err, ErrCurrencyPairEmpty) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrCurrencyPairEmpty)
	}

	_, err = Pairs{EMPTYPAIR}.FindDifferences(pairList, EMPTYFORMAT)
	if !errors.Is(err, ErrCurrencyPairEmpty) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrCurrencyPairEmpty)
	}

	// Test duplication
	duplication, err := NewPairsFromStrings([]string{defaultPairWDelimiter, "ETH-USD", "LTC-USD", "ETH-USD"})
	if err != nil {
		t.Fatal(err)
	}

	_, err = pairList.FindDifferences(duplication, EMPTYFORMAT)
	if !errors.Is(err, ErrPairDuplication) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrPairDuplication)
	}

	// This will allow for the removal of the duplicated item to be returned if
	// contained in the original list.
	diff, err = duplication.FindDifferences(pairList, EMPTYFORMAT)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if len(diff.Remove) != 1 {
		t.Fatal("expected removal value in pair difference struct")
	}

	if !diff.Remove[0].Equal(pairList[1]) {
		t.Fatal("unexpected value returned", diff.Remove[0], pairList[1])
	}
}

func TestPairsToStringArray(t *testing.T) {
	var pairs Pairs
	pairs = append(pairs, NewPair(BTC, USD))

	expected := []string{defaultPair}
	actual := pairs.Strings()

	if actual[0] != expected[0] {
		t.Error("TestPairsToStringArray: Unexpected values")
	}
}

func TestRandomPairFromPairs(t *testing.T) {
	// Test that an empty pairs array returns an empty currency pair
	var emptyPairs Pairs
	result, err := emptyPairs.GetRandomPair()
	if !errors.Is(err, ErrCurrencyPairsEmpty) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrCurrencyPairsEmpty)
	}
	if !result.IsEmpty() {
		t.Error("TestRandomPairFromPairs: Unexpected values")
	}

	// Test that a populated pairs array returns a non-empty currency pair
	var pairs Pairs
	pairs = append(pairs, NewPair(BTC, USD))
	result, err = pairs.GetRandomPair()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if result.IsEmpty() {
		t.Error("TestRandomPairFromPairs: Unexpected values")
	}

	// Test that a populated pairs array over a number of attempts returns ALL
	// currency pairs
	pairs = append(pairs, NewPair(ETH, USD))
	expectedResults := make(map[string]bool)
	for i := 0; i < 50; i++ {
		result, err = pairs.GetRandomPair()
		if !errors.Is(err, nil) {
			t.Fatalf("received: '%v' but expected: '%v'", err, nil)
		}
		expectedResults[result.String()] = true
	}

	for x := range pairs {
		if !expectedResults[pairs[x].String()] {
			t.Error("TestRandomPairFromPairs: Unexpected values")
		}
	}
}

func TestIsInvalid(t *testing.T) {
	p := NewPair(LTC, LTC)
	if !p.IsInvalid() {
		t.Error("IsInvalid() error expect true but received false")
	}
}

func TestMatchPairsWithNoDelimiter(t *testing.T) {
	p1, err := NewPairDelimiter("BTC-USDT", "-")
	if err != nil {
		t.Fatal(err)
	}
	p2, err := NewPairDelimiter("LTC-USD", "-")
	if err != nil {
		t.Fatal(err)
	}
	p3, err := NewPairFromStrings("EQUAD", "BTC")
	if err != nil {
		t.Fatal(err)
	}
	p4, err := NewPairFromStrings("HTDF", "USDT")
	if err != nil {
		t.Fatal(err)
	}
	p5, err := NewPairFromStrings("BETHER", "ETH")
	if err != nil {
		t.Fatal(err)
	}
	pairs := Pairs{
		p1,
		p2,
		p3,
		p4,
		p5,
	}

	p, err := MatchPairsWithNoDelimiter("BTCUSDT", pairs, PairFormat{
		Uppercase: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if p.Quote.String() != "USDT" && p.Base.String() != "BTC" {
		t.Error("unexpected response")
	}

	p, err = MatchPairsWithNoDelimiter("EQUADBTC", pairs, PairFormat{
		Uppercase: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if p.Base.String() != "EQUAD" && p.Quote.String() != "BTC" {
		t.Errorf("unexpected response base: %v quote: %v", p.Base.String(), p.Quote.String())
	}

	p, err = MatchPairsWithNoDelimiter("EQUADBTC", pairs, PairFormat{
		Uppercase: true,
		Delimiter: "/",
	})
	if err != nil {
		t.Fatal(err)
	}
	if p.Base.String() != "EQUAD" && p.Quote.String() != "BTC" {
		t.Errorf("unexpected response base: %v quote: %v", p.Base.String(), p.Quote.String())
	}

	p, err = MatchPairsWithNoDelimiter("HTDFUSDT", pairs, PairFormat{
		Uppercase: true,
		Delimiter: "/",
	})
	if err != nil {
		t.Fatal(err)
	}
	if p.Base.String() != "HTDF" && p.Quote.String() != "USDT" {
		t.Errorf("unexpected response base: %v quote: %v", p.Base.String(), p.Quote.String())
	}

	p, err = MatchPairsWithNoDelimiter("BETHERETH", pairs, PairFormat{
		Uppercase: true,
		Delimiter: "/",
	})
	if err != nil {
		t.Fatal(err)
	}
	if p.Base.String() != "BETHER" && p.Quote.String() != "ETH" {
		t.Errorf("unexpected response base: %v quote: %v", p.Base.String(), p.Quote.String())
	}
}

func TestPairFormat_Format(t *testing.T) {
	type fields struct {
		Uppercase bool
		Delimiter string
		Separator string
		Index     string
	}
	tests := []struct {
		name   string
		fields fields
		arg    Pair
		want   string
	}{
		{
			name:   "empty",
			fields: fields{},
			arg:    EMPTYPAIR,
			want:   "",
		},
		{
			name:   "empty format",
			fields: fields{},
			arg: Pair{
				Delimiter: "<>",
				Base:      AAA,
				Quote:     BTC,
			},
			want: "aaabtc",
		},
		{
			name: "format",
			fields: fields{
				Uppercase: true,
				Delimiter: "!!!",
			},
			arg: Pair{
				Delimiter: "<>",
				Base:      AAA,
				Quote:     BTC,
			},
			want: "AAA!!!BTC",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			f := &PairFormat{
				Uppercase: tt.fields.Uppercase,
				Delimiter: tt.fields.Delimiter,
				Separator: tt.fields.Separator,
				Index:     tt.fields.Index,
			}
			if got := f.Format(tt.arg); got != tt.want {
				t.Errorf("PairFormat.Format() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOther(t *testing.T) {
	received, err := NewPair(DAI, XRP).Other(DAI)
	if err != nil {
		t.Fatal(err)
	}
	if !received.Equal(XRP) {
		t.Fatal("unexpected value")
	}
	received, err = NewPair(DAI, XRP).Other(XRP)
	if err != nil {
		t.Fatal(err)
	}
	if !received.Equal(DAI) {
		t.Fatal("unexpected value")
	}
	if _, err := NewPair(DAI, XRP).Other(BTC); !errors.Is(err, ErrCurrencyCodeEmpty) {
		t.Fatal("unexpected value")
	}
}

func TestIsPopulated(t *testing.T) {
	if receiver := NewPair(BTC, USDT).IsPopulated(); !receiver {
		t.Fatal("unexpected value")
	}
	if receiver := NewPair(BTC, NewCode("USD-1245")).IsPopulated(); !receiver {
		t.Fatal("unexpected value")
	}
	if receiver := NewPair(BTC, EMPTYCODE).IsPopulated(); receiver {
		t.Fatal("unexpected value")
	}
	if receiver := NewPair(EMPTYCODE, EMPTYCODE).IsPopulated(); receiver {
		t.Fatal("unexpected value")
	}
}

func TestGetOrderParameters(t *testing.T) {
	t.Parallel()

	p := NewPair(BTC, USDT)
	testCases := []struct {
		Pair           Pair
		currency       Code
		market         bool
		selling        bool
		expectedParams *OrderParameters
		expectedError  error
	}{
		{expectedError: ErrCurrencyPairEmpty},
		{Pair: p, expectedError: ErrCurrencyCodeEmpty},
		{Pair: p, currency: XRP, selling: true, market: true, expectedError: ErrCurrencyNotAssociatedWithPair},

		{Pair: p, currency: BTC, selling: true, market: true, expectedParams: &OrderParameters{SellingCurrency: BTC, PurchasingCurrency: USDT, IsBuySide: false, IsAskLiquidity: false, Pair: p}},
		{Pair: p, currency: BTC, selling: false, market: true, expectedParams: &OrderParameters{SellingCurrency: USDT, PurchasingCurrency: BTC, IsBuySide: true, IsAskLiquidity: true, Pair: p}},
		{Pair: p, currency: BTC, selling: true, market: false, expectedParams: &OrderParameters{SellingCurrency: BTC, PurchasingCurrency: USDT, IsBuySide: false, IsAskLiquidity: true, Pair: p}},
		{Pair: p, currency: BTC, selling: false, market: false, expectedParams: &OrderParameters{SellingCurrency: USDT, PurchasingCurrency: BTC, IsBuySide: true, IsAskLiquidity: false, Pair: p}},

		{Pair: p, currency: USDT, selling: true, market: true, expectedParams: &OrderParameters{SellingCurrency: USDT, PurchasingCurrency: BTC, IsBuySide: true, IsAskLiquidity: true, Pair: p}},
		{Pair: p, currency: USDT, selling: false, market: true, expectedParams: &OrderParameters{SellingCurrency: BTC, PurchasingCurrency: USDT, IsBuySide: false, IsAskLiquidity: false, Pair: p}},
		{Pair: p, currency: USDT, selling: true, market: false, expectedParams: &OrderParameters{SellingCurrency: USDT, PurchasingCurrency: BTC, IsBuySide: true, IsAskLiquidity: false, Pair: p}},
		{Pair: p, currency: USDT, selling: false, market: false, expectedParams: &OrderParameters{SellingCurrency: BTC, PurchasingCurrency: USDT, IsBuySide: false, IsAskLiquidity: true, Pair: p}},
	}

	for i, tc := range testCases {
		tc := tc
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Parallel()
			var resp *OrderParameters
			var err error
			switch {
			case tc.market && tc.selling:
				resp, err = tc.Pair.MarketSellOrderParameters(tc.currency)
			case tc.market && !tc.selling:
				resp, err = tc.Pair.MarketBuyOrderParameters(tc.currency)
			case !tc.market && tc.selling:
				resp, err = tc.Pair.LimitSellOrderParameters(tc.currency)
			case !tc.market && !tc.selling:
				resp, err = tc.Pair.LimitBuyOrderParameters(tc.currency)
			}

			if !errors.Is(err, tc.expectedError) {
				t.Fatalf("received %v, expected %v", err, tc.expectedError)
			}

			if tc.expectedParams == nil {
				if resp != nil {
					t.Fatalf("received %v, expected nil", resp)
				}
				return
			}

			if resp.SellingCurrency != tc.expectedParams.SellingCurrency {
				t.Fatalf("SellingCurrency received %v, expected %v", resp.SellingCurrency, tc.expectedParams.SellingCurrency)
			}

			if resp.PurchasingCurrency != tc.expectedParams.PurchasingCurrency {
				t.Fatalf("PurchasingCurrency received %v, expected %v", resp.PurchasingCurrency, tc.expectedParams.PurchasingCurrency)
			}

			if resp.IsBuySide != tc.expectedParams.IsBuySide {
				t.Fatalf("BuySide received %v, expected %v", resp.IsBuySide, tc.expectedParams.IsBuySide)
			}

			if resp.IsAskLiquidity != tc.expectedParams.IsAskLiquidity {
				t.Fatalf("AskLiquidity received %v, expected %v", resp.IsAskLiquidity, tc.expectedParams.IsAskLiquidity)
			}

			if resp.Pair != tc.expectedParams.Pair {
				t.Fatalf("Pair received %v, expected %v", resp.Pair, tc.expectedParams.Pair)
			}
		})
	}
}

func TestIsAssociated(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		Pair           Pair
		associate      Pair
		expectedResult bool
	}{
		{Pair: NewPair(BTC, USDT), associate: NewPair(BTC, USDT), expectedResult: true},
		{Pair: NewPair(USDT, BTC), associate: NewPair(BTC, USDT), expectedResult: true},
		{Pair: NewPair(BTC, USDT), associate: NewPair(USDT, BTC), expectedResult: true},
		{Pair: NewPair(BTC, USDT), associate: NewPair(XRP, USDT), expectedResult: true},
		{Pair: NewPair(BTC, LTC), associate: NewPair(XRP, USDT), expectedResult: false},
		{Pair: NewPair(MA, LTC), associate: NewPair(LTC, USDT), expectedResult: true},
	}

	for x := range testCases {
		x := x
		t.Run(fmt.Sprintf("%d", x), func(t *testing.T) {
			t.Parallel()
			if testCases[x].Pair.IsAssociated(testCases[x].associate) != testCases[x].expectedResult {
				t.Fatalf("Test %d failed. Expected %v, received %v", x, testCases[x].expectedResult, testCases[x].Pair.IsAssociated(testCases[x].associate))
			}
		})
	}
}

func TestPair_GetFormatting(t *testing.T) {
	t.Parallel()
	p := NewPair(BTC, USDT)
	pFmt, err := p.GetFormatting()
	if err != nil {
		t.Error(err)
	}
	if !pFmt.Uppercase || pFmt.Delimiter != "" {
		t.Error("incorrect formatting")
	}

	p = NewPairWithDelimiter("eth", "usdt", "/")
	pFmt, err = p.GetFormatting()
	if err != nil {
		t.Error(err)
	}
	if pFmt.Uppercase || pFmt.Delimiter != "/" {
		t.Error("incorrect formatting")
	}

	p = NewPairWithDelimiter("eth", "USDT", "/")
	_, err = p.GetFormatting()
	if !errors.Is(err, errPairFormattingInconsistent) {
		t.Error(err)
	}
}

func TestNewBTCUSD(t *testing.T) {
	t.Parallel()
	p := NewBTCUSD()
	if !p.Base.Equal(BTC) {
		t.Fatal("expected base BTC from function NewBTCUSD")
	}
	if !p.Quote.Equal(USD) {
		t.Fatal("expected quote USD from function NewBTCUSD")
	}
}

func TestNewBTCUSDT(t *testing.T) {
	t.Parallel()
	p := NewBTCUSDT()
	if !p.Base.Equal(BTC) {
		t.Fatal("expected base BTC from function NewBTCUSDT")
	}
	if !p.Quote.Equal(USDT) {
		t.Fatal("expected quote USDT from function NewBTCUSDT")
	}
}
