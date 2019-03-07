package currency

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/common"
)

func TestPairsUpper(t *testing.T) {
	pairs := NewPairsFromStrings([]string{"btc_usd", "btc_aud", "btc_ltc"})
	expected := "BTC_USD,BTC_AUD,BTC_LTC"

	if pairs.Upper().Join() != expected {
		t.Errorf("Test Failed - Pairs Join() error expected %s but received %s",
			expected, pairs.Join())
	}
}

func TestPairsString(t *testing.T) {
	pairs := NewPairsFromStrings([]string{"btc_usd", "btc_aud", "btc_ltc"})
	expected := []string{"btc_usd", "btc_aud", "btc_ltc"}

	for i, p := range pairs {
		if p.String() != expected[i] {
			t.Errorf("Test Failed - Pairs String() error expected %s but received %s",
				expected, p.String())
		}
	}
}

func TestPairsJoin(t *testing.T) {
	pairs := NewPairsFromStrings([]string{"btc_usd", "btc_aud", "btc_ltc"})
	expected := "btc_usd,btc_aud,btc_ltc"

	if pairs.Join() != expected {
		t.Errorf("Test Failed - Pairs Join() error expected %s but received %s",
			expected, pairs.Join())
	}
}

func TestPairsFormat(t *testing.T) {
	pairs := NewPairsFromStrings([]string{"btc_usd", "btc_aud", "btc_ltc"})

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
		Pairs: NewPairsFromStrings([]string{"btc_usd", "btc_aud", "btc_ltc"}),
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

func TestRemovePairsByFilter(t *testing.T) {
	var pairs = Pairs{
		NewPair(BTC, USD),
		NewPair(LTC, USD),
		NewPair(LTC, USDT),
	}

	pairs = pairs.RemovePairsByFilter(USDT)
	if pairs.Contains(NewPair(LTC, USDT), true) {
		t.Error("Test failed. TestRemovePairsByFilter unexpected result")
	}
}

func TestContains(t *testing.T) {
	var pairs = Pairs{
		NewPair(BTC, USD),
		NewPair(LTC, USD),
	}

	if !pairs.Contains(NewPair(BTC, USD), true) {
		t.Errorf("Test failed. TestContains: Expected pair was not found")
	}

	if pairs.Contains(NewPair(ETH, USD), false) {
		t.Errorf("Test failed. TestContains: Non-existent pair was found")
	}
}
