package currency

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestPairsUpper(t *testing.T) {
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
	_, err := NewPairsFromString("")
	if !errors.Is(err, errCannotCreatePair) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errCannotCreatePair)
	}

	pairs, err := NewPairsFromString("ALGO-AUD,BAT-AUD,BCH-AUD,BSV-AUD,BTC-AUD,COMP-AUD,ENJ-AUD,ETC-AUD,ETH-AUD,ETH-BTC,GNT-AUD,LINK-AUD,LTC-AUD,LTC-BTC,MCAU-AUD,OMG-AUD,POWR-AUD,UNI-AUD,USDT-AUD,XLM-AUD,XRP-AUD,XRP-BTC")
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
	pairs := Pairs{
		NewPair(BTC, USD),
		NewPair(LTC, USD),
		NewPair(USD, NZD),
		NewPair(LTC, USDT),
	}
	contains(t, []Code{BTC, LTC, USDT}, pairs.GetCrypto())
}

func TestGetFiat(t *testing.T) {
	pairs := Pairs{
		NewPair(BTC, USD),
		NewPair(LTC, USD),
		NewPair(USD, NZD),
		NewPair(LTC, USDT),
	}
	contains(t, []Code{USD, NZD}, pairs.GetFiat())
}

func TestGetCurrencies(t *testing.T) {
	pairs := Pairs{
		NewPair(BTC, USD),
		NewPair(LTC, USD),
		NewPair(USD, NZD),
		NewPair(LTC, USDT),
	}
	contains(t, []Code{BTC, USD, LTC, NZD, USDT}, pairs.GetCurrencies())
}

func contains(t *testing.T, c1, c2 []Code) {
	t.Helper()
codes:
	for x := range c1 {
		for y := range c2 {
			if c1[x].Match(c2[y]) {
				continue codes
			}
		}
		t.Fatalf("cannot find currency %s in returned currency list %v", c1[x], c2)
	}
}

//  2575473	       474.2 ns/op	     112 B/op	       3 allocs/op
// 4526858	       280.2 ns/op	      48 B/op	       1 allocs/op

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
