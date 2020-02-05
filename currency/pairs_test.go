package currency

import (
	"encoding/json"
	"testing"
)

func TestPairsUpper(t *testing.T) {
	pairs, err := NewPairsFromStrings([]string{"btc_usd", "btc_aud", "btc_ltc"})
	if err != nil {
		t.Fatal(err)
	}
	expected := "BTC_USD,BTC_AUD,BTC_LTC"

	if pairs.Upper().Join() != expected {
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
