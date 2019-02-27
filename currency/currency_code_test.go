package currency

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/common"
)

func TestCodeLower(t *testing.T) {
	expected := "test"
	cc := NewCode("TEST")
	if cc.Lower().String() != expected {
		t.Errorf("Test Failed - Currency Code Lower() error expected %s but recieved %s",
			expected,
			cc.Lower())
	}
}

func TestCodeUpper(t *testing.T) {
	expected := "TEST"
	cc := NewCode("test")
	if cc.Upper().String() != expected {
		t.Errorf("Test Failed - Currency Code Upper() error expected %s but recieved %s",
			expected,
			cc.Upper())
	}
}

func TestCodeUnmarshalJSON(t *testing.T) {
	var unmarshalHere Code
	expected := "BRO"
	encoded, err := common.JSONEncode(expected)
	if err != nil {
		t.Fatal("Test Failed - Currency Code UnmarshalJSON error", err)
	}

	err = common.JSONDecode(encoded, &unmarshalHere)
	if err != nil {
		t.Fatal("Test Failed - Currency Code UnmarshalJSON error", err)
	}

	err = common.JSONDecode(encoded, &unmarshalHere)
	if err != nil {
		t.Fatal("Test Failed - Currency Code UnmarshalJSON error", err)
	}

	if unmarshalHere.String() != expected {
		t.Errorf("Test Failed - Currency Code Upper() error expected %s but recieved %s",
			expected,
			unmarshalHere)
	}
}

func TestCodeMarshalJSON(t *testing.T) {
	quickstruct := struct {
		Codey Code `json:"sweetCodes"`
	}{
		Codey: NewCode("BRO"),
	}

	expectedJSON := `{"sweetCodes":"BRO"}`

	encoded, err := common.JSONEncode(quickstruct)
	if err != nil {
		t.Fatal("Test Failed - Currency Code UnmarshalJSON error", err)
	}

	if string(encoded) != expectedJSON {
		t.Errorf("Test Failed - Currency Code Upper() error expected %s but recieved %s",
			expectedJSON,
			string(encoded))
	}

	quickstruct = struct {
		Codey Code `json:"sweetCodes"`
	}{
		Codey: Code{}, // nil code
	}

	encoded, err = common.JSONEncode(quickstruct)
	if err != nil {
		t.Fatal("Test Failed - Currency Code UnmarshalJSON error", err)
	}

	newExpectedJSON := `{"sweetCodes":""}`
	if string(encoded) != newExpectedJSON {
		t.Errorf("Test Failed - Currency Code Upper() error expected %s but recieved %s",
			newExpectedJSON, string(encoded))
	}
}

func TestIsDefaultCurrency(t *testing.T) {
	if !USD.IsDefaultFiatCurrency() {
		t.Errorf("Test Failed. TestIsDefaultCurrency Cannot match currency %s.",
			USD)
	}
	if !AUD.IsDefaultFiatCurrency() {
		t.Errorf("Test Failed. TestIsDefaultCurrency Cannot match currency, %s.",
			AUD)
	}
	if LTC.IsDefaultFiatCurrency() {
		t.Errorf("Test Failed. TestIsDefaultCurrency Function return is incorrect with, %s.",
			LTC)
	}
}

func TestIsDefaultCryptocurrency(t *testing.T) {
	if !BTC.IsDefaultCryptocurrency() {
		t.Errorf("Test Failed. TestIsDefaultCryptocurrency cannot match currency, %s.",
			BTC)
	}
	if !LTC.IsDefaultCryptocurrency() {
		t.Errorf("Test Failed. TestIsDefaultCryptocurrency cannot match currency, %s.",
			LTC)
	}
	if AUD.IsDefaultCryptocurrency() {
		t.Errorf("Test Failed. TestIsDefaultCryptocurrency function return is incorrect with, %s.",
			AUD)
	}
}

func TestIsFiatCurrency(t *testing.T) {
	if !USD.IsFiatCurrency() {
		t.Errorf(
			"Test Failed. TestIsFiatCurrency cannot match currency, %s.", USD)
	}
	if !CNY.IsFiatCurrency() {
		t.Errorf(
			"Test Failed. TestIsFiatCurrency cannot match currency, %s.", CNY)
	}
	if LINO.IsFiatCurrency() {
		t.Errorf(
			"Test Failed. TestIsFiatCurrency cannot match currency, %s.", LINO,
		)
	}
}

func TestIsCryptocurrency(t *testing.T) {
	if !BTC.IsCryptocurrency() {
		t.Errorf("Test Failed. TestIsFiatCurrency cannot match currency, %s.",
			BTC)
	}
	if !LTC.IsCryptocurrency() {
		t.Errorf("Test Failed. TestIsFiatCurrency cannot match currency, %s.",
			LTC)
	}
	if AUD.IsCryptocurrency() {
		t.Errorf("Test Failed. TestIsFiatCurrency cannot match currency, %s.",
			AUD)
	}
}
