package currency

import (
	"encoding/json"
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
	actual := pair.Lower()
	expected, err := NewPairFromString(defaultPair)
	if err != nil {
		t.Fatal(err)
	}

	if actual.String() != expected.Lower().String() {
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
	actual := pair.Upper()
	expected, err := NewPairFromString(defaultPair)
	if err != nil {
		t.Fatal(err)
	}
	if actual.String() != expected.String() {
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
		t.Errorf("Pairs UnmarshalJSON() error expected %s but received %s",
			configPair, unmarshalHere)
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

func TestString(t *testing.T) {
	t.Parallel()
	pair := NewPair(BTC, USD)
	actual := defaultPair
	expected := pair.String()
	if actual != expected {
		t.Errorf("String(): %s was not equal to expected value: %s",
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
			"GetFirstCurrency(): %s was not equal to expected value: %s",
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
			"GetSecondCurrency(): %s was not equal to expected value: %s",
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

	actual = pair.Format("", false).String()
	expected = "btcusd"
	if actual != expected {
		t.Errorf(
			"Pair(): %s was not equal to expected value: %s",
			actual, expected,
		)
	}

	actual = pair.Format("~", true).String()
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
	actual := pair.Swap().String()
	expected := "USDBTC"
	if actual != expected {
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
	actual := pair.String()
	expected := defaultPair
	if actual != expected {
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
	p, err = NewPairFromFormattedPairs("ethusdt", pairs, PairFormat{})
	if err != nil {
		t.Fatal(err)
	}

	if p.String() != "ethusdt" && p.Base.String() != "eth" {
		t.Error("TestNewPairFromFormattedPairs: Expected currency was not found")
	}
}

func TestContainsCurrency(t *testing.T) {
	p := NewPair(BTC, USD)

	if !p.ContainsCurrency(BTC) {
		t.Error("TestContainsCurrency: Expected currency was not found")
	}

	if p.ContainsCurrency(ETH) {
		t.Error("TestContainsCurrency: Non-existent currency was found")
	}
}

func TestFormatPairs(t *testing.T) {
	newP, err := FormatPairs([]string{""}, "-", "")
	if err != nil {
		t.Error("FormatPairs() error", err)
	}

	if len(newP) > 0 {
		t.Error("TestFormatPairs: Empty string returned a valid pair")
	}

	newP, err = FormatPairs([]string{defaultPairWDelimiter}, "-", "")
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
	newPairs, removedPairs := pairList.FindDifferences(dash)
	if len(newPairs) != 1 && len(removedPairs) != 3 {
		t.Error("TestFindPairDifferences: Unexpected values")
	}

	emptyPairsList, err := NewPairsFromStrings([]string{""})
	if err != nil {
		t.Fatal(err)
	}

	// Test that we don't allow empty strings for new pairs
	newPairs, removedPairs = pairList.FindDifferences(emptyPairsList)
	if len(newPairs) != 0 && len(removedPairs) != 3 {
		t.Error("TestFindPairDifferences: Unexpected values")
	}

	// Test that we don't allow empty strings for new pairs
	newPairs, removedPairs = emptyPairsList.FindDifferences(pairList)
	if len(newPairs) != 3 && len(removedPairs) != 0 {
		t.Error("TestFindPairDifferences: Unexpected values")
	}

	// Test that the supplied pair lists are the same, so
	// no newPairs or removedPairs
	newPairs, removedPairs = pairList.FindDifferences(pairList)
	if len(newPairs) != 0 && len(removedPairs) != 0 {
		t.Error("TestFindPairDifferences: Unexpected values")
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
	result := emptyPairs.GetRandomPair()
	if !result.IsEmpty() {
		t.Error("TestRandomPairFromPairs: Unexpected values")
	}

	// Test that a populated pairs array returns a non-empty currency pair
	var pairs Pairs
	pairs = append(pairs, NewPair(BTC, USD))
	result = pairs.GetRandomPair()

	if result.IsEmpty() {
		t.Error("TestRandomPairFromPairs: Unexpected values")
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
			arg:    Pair{},
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
