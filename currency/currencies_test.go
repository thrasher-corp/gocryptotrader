package currency

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

func TestCurrenciesUnmarshalJSON(t *testing.T) {
	var unmarshalHere Currencies
	expected := "btc,usd,ltc,bro,things"
	encoded, err := json.Marshal(expected)
	if err != nil {
		t.Fatal("Currencies UnmarshalJSON() error", err)
	}

	err = json.Unmarshal(encoded, &unmarshalHere)
	if err != nil {
		t.Fatal("Currencies UnmarshalJSON() error", err)
	}

	err = json.Unmarshal(encoded, &unmarshalHere)
	if err != nil {
		t.Fatal("Currencies UnmarshalJSON() error", err)
	}

	if unmarshalHere.Join() != expected {
		t.Errorf("Currencies UnmarshalJSON() error expected %s but received %s",
			expected, unmarshalHere.Join())
	}
}

func TestCurrenciesMarshalJSON(t *testing.T) {
	quickStruct := struct {
		C Currencies `json:"amazingCurrencies"`
	}{
		C: NewCurrenciesFromStringArray([]string{"btc", "usd", "ltc", "bro", "things"}),
	}

	encoded, err := json.Marshal(quickStruct)
	if err != nil {
		t.Fatal("Currencies MarshalJSON() error", err)
	}

	expected := `{"amazingCurrencies":"btc,usd,ltc,bro,things"}`
	if string(encoded) != expected {
		t.Errorf("Currencies MarshalJSON() error expected %s but received %s",
			expected, string(encoded))
	}
}

func TestMatch(t *testing.T) {
	matchString := []string{"btc", "usd", "ltc", "bro", "things"}
	c := NewCurrenciesFromStringArray(matchString)

	if !c.Match(NewCurrenciesFromStringArray(matchString)) {
		t.Fatal("should match")
	}
	if c.Match(NewCurrenciesFromStringArray([]string{"btc", "usd", "ltc", "bro"})) {
		t.Fatal("should not match")
	}
	if c.Match(NewCurrenciesFromStringArray([]string{"btc", "usd", "ltc", "bro", "garbo"})) {
		t.Fatal("should not match")
	}
}

func TestCurrenciesAdd(t *testing.T) {
	c := Currencies{}
	c = c.Add(BTC)
	assert.Len(t, c, 1, "Should have one currency")
	c = c.Add(ETH)
	assert.Len(t, c, 2, "Should have two currencies")
	c = c.Add(BTC)
	assert.Len(t, c, 2, "Adding a duplicate should not change anything")
}
