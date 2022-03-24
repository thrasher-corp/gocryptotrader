package currency

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestPairsUpper(t *testing.T) {
	t.Parallel()
	pairs, err := NewPairsFromStrings([]string{"btc_usd", "btc_aud", "btc_ltc"})
	if err != nil {
		t.Fatal(err)
	}
	if expected := "BTC_USD,BTC_AUD,BTC_LTC"; pairs.Upper().Join() != expected {
		t.Errorf("Pairs Join() error expected %s but received %s",
			expected, pairs.Join())
	}
}

func TestPairsLower(t *testing.T) {
	t.Parallel()
	pairs, err := NewPairsFromStrings([]string{"BTC_USD", "BTC_AUD", "BTC_LTC"})
	if err != nil {
		t.Fatal(err)
	}
	if expected := "btc_usd,btc_aud,btc_ltc"; pairs.Lower().Join() != expected {
		t.Errorf("Pairs Join() error expected %s but received %s",
			expected, pairs.Join())
	}
}

func TestPairsString(t *testing.T) {
	t.Parallel()
	pairs, err := NewPairsFromStrings([]string{"btc_usd", "btc_aud", "btc_ltc"})
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"btc_usd", "btc_aud", "btc_ltc"}

	for i, p := range pairs {
		if p.String() != expected[i] {
			t.Errorf("Pairs String() error expected %s but received %s",
				expected, p.String())
		}
	}
}

func TestPairsFromString(t *testing.T) {
	t.Parallel()
	if _, err := NewPairsFromString("", ""); !errors.Is(err, errNoDelimiter) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoDelimiter)
	}

	if _, err := NewPairsFromString("", ","); !errors.Is(err, errCannotCreatePair) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errCannotCreatePair)
	}

	pairs, err := NewPairsFromString("ALGO-AUD,BAT-AUD,BCH-AUD,BSV-AUD,BTC-AUD,COMP-AUD,ENJ-AUD,ETC-AUD,ETH-AUD,ETH-BTC,GNT-AUD,LINK-AUD,LTC-AUD,LTC-BTC,MCAU-AUD,OMG-AUD,POWR-AUD,UNI-AUD,USDT-AUD,XLM-AUD,XRP-AUD,XRP-BTC", ",")
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	expected := []string{"ALGO-AUD", "BAT-AUD", "BCH-AUD", "BSV-AUD", "BTC-AUD",
		"COMP-AUD", "ENJ-AUD", "ETC-AUD", "ETH-AUD", "ETH-BTC", "GNT-AUD",
		"LINK-AUD", "LTC-AUD", "LTC-BTC", "MCAU-AUD", "OMG-AUD", "POWR-AUD",
		"UNI-AUD", "USDT-AUD", "XLM-AUD", "XRP-AUD", "XRP-BTC"}

	returned := pairs.Strings()
	for x := range returned {
		if returned[x] != expected[x] {
			t.Fatalf("received: '%v' but expected: '%v'", returned[x], expected[x])
		}
	}
}

func TestPairsJoin(t *testing.T) {
	t.Parallel()
	pairs, err := NewPairsFromStrings([]string{"btc_usd", "btc_aud", "btc_ltc"})
	if err != nil {
		t.Fatal(err)
	}
	expected := "btc_usd,btc_aud,btc_ltc"

	if pairs.Join() != expected {
		t.Errorf("Pairs Join() error expected %s but received %s",
			expected, pairs.Join())
	}
}

func TestPairsFormat(t *testing.T) {
	t.Parallel()
	pairs, err := NewPairsFromStrings([]string{"btc_usd", "btc_aud", "btc_ltc"})
	if err != nil {
		t.Fatal(err)
	}

	expected := "BTC-USD,BTC-AUD,BTC-LTC"
	if pairs.Format("-", "", true).Join() != expected {
		t.Errorf("Pairs Join() error expected %s but received %s",
			expected, pairs.Format("-", "", true).Join())
	}

	expected = "btc:usd,btc:aud,btc:ltc"
	if pairs.Format(":", "", false).Join() != expected {
		t.Errorf("Pairs Join() error expected %s but received %s",
			expected, pairs.Format(":", "", false).Join())
	}

	if pairs.Format(":", "KRW", false).Join() != "" {
		t.Errorf("Pairs Join() error expected %s but received %s",
			expected, pairs.Format(":", "KRW", true).Join())
	}

	pairs, err = NewPairsFromStrings([]string{"DASHKRW", "BTCKRW"})
	if err != nil {
		t.Fatal(err)
	}
	expected = "dash-krw,btc-krw"
	if pairs.Format("-", "KRW", false).Join() != expected {
		t.Errorf("Pairs Join() error expected %s but received %s",
			expected, pairs.Format("-", "KRW", false).Join())
	}
}

func TestPairsUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var unmarshalHere Pairs
	configPairs := ""
	encoded, err := json.Marshal(configPairs)
	if err != nil {
		t.Fatal("Pairs UnmarshalJSON() error", err)
	}

	err = json.Unmarshal([]byte{1, 3, 3, 7}, &unmarshalHere)
	if err == nil {
		t.Fatal("error cannot be nil")
	}

	err = json.Unmarshal(encoded, &unmarshalHere)
	if err != nil {
		t.Fatal("Pairs UnmarshalJSON() error", err)
	}

	configPairs = "btc_usd,btc_aud,btc_ltc"
	encoded, err = json.Marshal(configPairs)
	if err != nil {
		t.Fatal("Pairs UnmarshalJSON() error", err)
	}

	err = json.Unmarshal(encoded, &unmarshalHere)
	if err != nil {
		t.Fatal("Pairs UnmarshalJSON() error", err)
	}

	err = json.Unmarshal(encoded, &unmarshalHere)
	if err != nil {
		t.Fatal("Pairs UnmarshalJSON() error", err)
	}

	if unmarshalHere.Join() != configPairs {
		t.Errorf("Pairs UnmarshalJSON() error expected %s but received %s",
			configPairs, unmarshalHere.Join())
	}
}

func TestPairsMarshalJSON(t *testing.T) {
	t.Parallel()
	pairs, err := NewPairsFromStrings([]string{"btc_usd", "btc_aud", "btc_ltc"})
	if err != nil {
		t.Fatal(err)
	}

	quickstruct := struct {
		Pairs Pairs `json:"soManyPairs"`
	}{
		Pairs: pairs,
	}

	encoded, err := json.Marshal(quickstruct)
	if err != nil {
		t.Fatal("Pairs MarshalJSON() error", err)
	}

	expected := `{"soManyPairs":"btc_usd,btc_aud,btc_ltc"}`
	if string(encoded) != expected {
		t.Errorf("Pairs MarshalJSON() error expected %s but received %s",
			expected, string(encoded))
	}
}

func TestRemovePairsByFilter(t *testing.T) {
	t.Parallel()
	var pairs = Pairs{
		NewPair(BTC, USD),
		NewPair(LTC, USD),
		NewPair(LTC, USDT),
	}

	pairs = pairs.RemovePairsByFilter(USDT)
	if pairs.Contains(NewPair(LTC, USDT), true) {
		t.Error("TestRemovePairsByFilter unexpected result")
	}
}

func TestGetPairsByFilter(t *testing.T) {
	t.Parallel()
	var pairs = Pairs{
		NewPair(BTC, USD),
		NewPair(LTC, USD),
		NewPair(LTC, USDT),
	}

	filtered := pairs.GetPairsByFilter(LTC)
	if !filtered.Contains(NewPair(LTC, USDT), true) &&
		!filtered.Contains(NewPair(LTC, USD), true) {
		t.Error("TestRemovePairsByFilter unexpected result")
	}
}

func TestRemove(t *testing.T) {
	t.Parallel()
	var pairs = Pairs{
		NewPair(BTC, USD),
		NewPair(LTC, USD),
		NewPair(LTC, USDT),
	}

	p := NewPair(BTC, USD)
	pairs = pairs.Remove(p)
	if pairs.Contains(p, true) || len(pairs) != 2 {
		t.Error("TestRemove unexpected result")
	}
}

func TestAdd(t *testing.T) {
	t.Parallel()
	var pairs = Pairs{
		NewPair(BTC, USD),
		NewPair(LTC, USD),
		NewPair(LTC, USDT),
	}

	// Test adding a new pair to the list of pairs
	p := NewPair(BTC, USDT)
	pairs = pairs.Add(p)
	if !pairs.Contains(p, true) || len(pairs) != 4 {
		t.Error("TestAdd unexpected result")
	}

	// Now test adding a pair which already exists
	pairs = pairs.Add(p)
	if len(pairs) != 4 {
		t.Error("TestAdd unexpected result")
	}
}

func TestContains(t *testing.T) {
	t.Parallel()
	var pairs = Pairs{
		NewPair(BTC, USD),
		NewPair(LTC, USD),
		NewPair(USD, ZRX),
	}

	if !pairs.Contains(NewPair(BTC, USD), true) {
		t.Errorf("TestContains: Expected pair was not found")
	}

	if pairs.Contains(NewPair(USD, BTC), true) {
		t.Errorf("TestContains: Unexpected pair was found")
	}

	if !pairs.Contains(NewPair(USD, BTC), false) {
		t.Errorf("TestContains: Expected pair was not found")
	}

	if pairs.Contains(NewPair(ETH, USD), false) {
		t.Errorf("TestContains: Non-existent pair was found")
	}
}

func TestDeriveFrom(t *testing.T) {
	t.Parallel()
	_, err := Pairs{}.DeriveFrom("")
	if !errors.Is(err, errPairsEmpty) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errPairsEmpty)
	}
	var testCases = Pairs{
		NewPair(BTC, USDT),
		NewPair(USDC, USDT),
		NewPair(USDC, USD),
		NewPair(BTC, LTC),
		NewPair(LTC, SAFEMARS),
	}

	_, err = testCases.DeriveFrom("")
	if !errors.Is(err, errSymbolEmpty) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errSymbolEmpty)
	}

	_, err = testCases.DeriveFrom("btcUSD")
	if !errors.Is(err, ErrPairNotFound) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrPairNotFound)
	}

	got, err := testCases.DeriveFrom("USDCUSD")
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if got.Upper().String() != "USDCUSD" {
		t.Fatalf("received: '%v' but expected: '%v'", got.Upper().String(), "USDCUSD")
	}
}

func TestGetCrypto(t *testing.T) {
	t.Parallel()
	pairs := Pairs{
		NewPair(BTC, USD),
		NewPair(LTC, USD),
		NewPair(USD, NZD),
		NewPair(LTC, USDT),
	}
	contains(t, []Code{BTC, LTC, USDT}, pairs.GetCrypto())
}

func TestGetFiat(t *testing.T) {
	t.Parallel()
	pairs := Pairs{
		NewPair(BTC, USD),
		NewPair(LTC, USD),
		NewPair(USD, NZD),
		NewPair(LTC, USDT),
	}
	contains(t, []Code{USD, NZD}, pairs.GetFiat())
}

func TestGetCurrencies(t *testing.T) {
	t.Parallel()
	pairs := Pairs{
		NewPair(BTC, USD),
		NewPair(LTC, USD),
		NewPair(USD, NZD),
		NewPair(LTC, USDT),
	}
	contains(t, []Code{BTC, USD, LTC, NZD, USDT}, pairs.GetCurrencies())
}

func TestGetStables(t *testing.T) {
	t.Parallel()
	pairs := Pairs{
		NewPair(BTC, USD),
		NewPair(LTC, USD),
		NewPair(USD, NZD),
		NewPair(LTC, USDT),
		NewPair(DAI, USDT),
		NewPair(LTC, USDC),
		NewPair(USDP, USDT),
	}
	contains(t, []Code{USDT, USDP, USDC, DAI}, pairs.GetStables())
}

func contains(t *testing.T, c1, c2 []Code) {
	t.Helper()
codes:
	for x := range c1 {
		for y := range c2 {
			if c1[x].Equal(c2[y]) {
				continue codes
			}
		}
		t.Fatalf("cannot find currency %s in returned currency list %v", c1[x], c2)
	}
}

// Current: 6176922	       260.0 ns/op	      48 B/op	       1 allocs/op
// Prior: 2575473	       474.2 ns/op	     112 B/op	       3 allocs/op
func BenchmarkGetCrypto(b *testing.B) {
	pairs := Pairs{
		NewPair(BTC, USD),
		NewPair(LTC, USD),
		NewPair(USD, NZD),
		NewPair(LTC, USDT),
	}

	for x := 0; x < b.N; x++ {
		_ = pairs.GetCrypto()
	}
}

func TestGetMatch(t *testing.T) {
	t.Parallel()
	pairs := Pairs{
		NewPair(BTC, USD),
		NewPair(LTC, USD),
		NewPair(USD, NZD),
		NewPair(LTC, USDT),
	}

	_, err := pairs.GetMatch(NewPair(BTC, WABI))
	if !errors.Is(err, ErrPairNotFound) {
		t.Fatalf("received: '%v' but expected '%v'", err, ErrPairNotFound)
	}

	expected := NewPair(BTC, USD)
	match, err := pairs.GetMatch(expected)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}

	if !match.Equal(expected) {
		t.Fatalf("received: '%v' but expected '%v'", match, expected)
	}

	match, err = pairs.GetMatch(NewPair(USD, BTC))
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected '%v'", err, nil)
	}
	if !match.Equal(expected) {
		t.Fatalf("received: '%v' but expected '%v'", match, expected)
	}
}

func TestGetStablesMatch(t *testing.T) {
	t.Parallel()
	pairs := Pairs{
		NewPair(BTC, USD),
		NewPair(LTC, USD),
		NewPair(USD, NZD),
		NewPair(LTC, USDT),
		NewPair(LTC, DAI),
		NewPair(USDT, XRP),
		NewPair(DAI, XRP),
	}

	stablePairs := pairs.GetStablesMatch(BTC)
	if len(stablePairs) != 0 {
		t.Fatal("unexpected value")
	}

	stablePairs = pairs.GetStablesMatch(USD)
	if len(stablePairs) != 0 {
		t.Fatal("unexpected value")
	}

	stablePairs = pairs.GetStablesMatch(LTC)
	if len(stablePairs) != 2 {
		t.Fatal("unexpected value")
	}

	if !stablePairs[0].Equal(NewPair(LTC, USDT)) {
		t.Fatal("unexpected value")
	}

	if !stablePairs[1].Equal(NewPair(LTC, DAI)) {
		t.Fatal("unexpected value")
	}

	stablePairs = pairs.GetStablesMatch(XRP)
	if len(stablePairs) != 2 {
		t.Fatal("unexpected value")
	}

	if !stablePairs[0].Equal(NewPair(USDT, XRP)) {
		t.Fatal("unexpected value")
	}

	if !stablePairs[1].Equal(NewPair(DAI, XRP)) {
		t.Fatal("unexpected value")
	}
}

// Current: 5594431	       217.4 ns/op	     168 B/op	       8 allocs/op
// Prev:  3490366	       373.4 ns/op	     296 B/op	      11 allocs/op
func BenchmarkPairsString(b *testing.B) {
	pairs := Pairs{
		NewPair(BTC, USD),
		NewPair(LTC, USD),
		NewPair(USD, NZD),
		NewPair(LTC, USDT),
		NewPair(LTC, DAI),
		NewPair(USDT, XRP),
		NewPair(DAI, XRP),
	}

	for x := 0; x < b.N; x++ {
		_ = pairs.Strings()
	}
}

// Current:  6691011	       184.6 ns/op	     352 B/op	       1 allocs/op
// Prev:  3746151	       317.1 ns/op	     720 B/op	       4 allocs/op
func BenchmarkPairsFormat(b *testing.B) {
	pairs := Pairs{
		NewPair(BTC, USD),
		NewPair(LTC, USD),
		NewPair(USD, NZD),
		NewPair(LTC, USDT),
		NewPair(LTC, DAI),
		NewPair(USDT, XRP),
		NewPair(DAI, XRP),
	}

	for x := 0; x < b.N; x++ {
		_ = pairs.Format("/", "", false)
	}
}

// current: 13075897	       100.4 ns/op	     352 B/op	       1 allocs/o
// prev: 8188616	       148.0 ns/op	     336 B/op	       3 allocs/op
func BenchmarkRemovePairsByFilter(b *testing.B) {
	pairs := Pairs{
		NewPair(BTC, USD),
		NewPair(LTC, USD),
		NewPair(USD, NZD),
		NewPair(LTC, USDT),
		NewPair(LTC, DAI),
		NewPair(USDT, XRP),
		NewPair(DAI, XRP),
	}

	for x := 0; x < b.N; x++ {
		_ = pairs.RemovePairsByFilter(USD)
	}
}

func TestPairsContainsCurrency(t *testing.T) {
	t.Parallel()
	pairs := Pairs{
		NewPair(BTC, USD),
		NewPair(LTC, USD),
		NewPair(USD, NZD),
		NewPair(LTC, USDT),
		NewPair(LTC, DAI),
		NewPair(USDT, XRP),
		NewPair(DAI, XRP),
	}

	if !pairs.ContainsCurrency(BTC) {
		t.Fatalf("expected %s to be %v", BTC, true)
	}
	if !pairs.ContainsCurrency(USD) {
		t.Fatalf("expected %s to be %v", USD, true)
	}
	if !pairs.ContainsCurrency(LTC) {
		t.Fatalf("expected %s to be %v", LTC, true)
	}
	if !pairs.ContainsCurrency(DAI) {
		t.Fatalf("expected %s to be %v", DAI, true)
	}
	if !pairs.ContainsCurrency(XRP) {
		t.Fatalf("expected %s to be %v", XRP, true)
	}
	if pairs.ContainsCurrency(ATOM3L) {
		t.Fatalf("expected %s to be %v", ATOM3L, false)
	}
}

func TestGetPairsByCurrencies(t *testing.T) {
	t.Parallel()
	available := Pairs{
		NewPair(BTC, USD),
		NewPair(LTC, USD),
		NewPair(USD, NZD),
		NewPair(LTC, USDT),
		NewPair(LTC, DAI),
		NewPair(USDT, XRP),
		NewPair(DAI, XRP),
	}

	enabled := available.GetPairsByCurrencies(Currencies{USD})
	if len(enabled) != 0 {
		t.Fatalf("received %v but expected %v", enabled, "no pairs")
	}

	enabled = available.GetPairsByCurrencies(Currencies{USD, BTC})
	if !enabled.Contains(NewPair(BTC, USD), true) {
		t.Fatalf("received %v but expected to contain %v", enabled, NewPair(BTC, USD))
	}

	enabled = available.GetPairsByCurrencies(Currencies{USD, BTC, LTC, NZD, USDT, DAI})
	if len(enabled) != 5 {
		t.Fatalf("received %v but expected  %v", enabled, 5)
	}
}
