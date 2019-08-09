package currency

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/common"
)

func TestCurrenciesUnmarshalJSON(t *testing.T) {
	var unmarshalHere Currencies
	expected := "btc,usd,ltc,bro,things"
	encoded, err := common.JSONEncode(expected)
	if err != nil {
		t.Fatal("Test Failed - Currencies UnmarshalJSON() error", err)
	}

	err = common.JSONDecode(encoded, &unmarshalHere)
	if err != nil {
		t.Fatal("Test Failed - Currencies UnmarshalJSON() error", err)
	}

	err = common.JSONDecode(encoded, &unmarshalHere)
	if err != nil {
		t.Fatal("Test Failed - Currencies UnmarshalJSON() error", err)
	}

	if unmarshalHere.Join() != expected {
		t.Errorf("Test Failed - Currencies UnmarshalJSON() error expected %s but received %s",
			expected, unmarshalHere.Join())
	}
}

func TestCurrenciesMarshalJSON(t *testing.T) {
	quickStruct := struct {
		C Currencies `json:"amazingCurrencies"`
	}{
		C: NewCurrenciesFromStringArray([]string{"btc", "usd", "ltc", "bro", "things"}),
	}

	encoded, err := common.JSONEncode(quickStruct)
	if err != nil {
		t.Fatal("Test Failed - Currencies MarshalJSON() error", err)
	}

	expected := `{"amazingCurrencies":"btc,usd,ltc,bro,things"}`
	if string(encoded) != expected {
		t.Errorf("Test Failed - Currencies MarshalJSON() error expected %s but received %s",
			expected, string(encoded))
	}
}
