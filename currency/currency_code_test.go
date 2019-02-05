package currency

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/common"
)

func TestCodeString(t *testing.T) {
	expected := "TEST"
	cc := Code("TEST")
	if cc.String() != expected {
		t.Errorf("Test Failed - Currency Code String() error expected %s but recieved %s",
			expected, cc)
	}

}

func TestCodeLower(t *testing.T) {
	expected := "test"
	cc := Code("TEST")
	if cc.Lower().String() != expected {
		t.Errorf("Test Failed - Currency Code Lower() error expected %s but recieved %s",
			expected, cc.Lower())
	}

}

func TestCodeUpper(t *testing.T) {
	expected := "TEST"
	cc := Code("test")
	if cc.Upper().String() != expected {
		t.Errorf("Test Failed - Currency Code Upper() error expected %s but recieved %s",
			expected, cc.Upper())
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
			expected, unmarshalHere)
	}
}

func TestCodeMarshalJSON(t *testing.T) {
	quickstruct := struct {
		Codey Code `json:"sweetCodes"`
	}{
		Codey: Code("BRO"),
	}

	expectedJSON := `{"sweetCodes":"BRO"}`

	encoded, err := common.JSONEncode(quickstruct)
	if err != nil {
		t.Fatal("Test Failed - Currency Code UnmarshalJSON error", err)
	}

	if string(encoded) != expectedJSON {
		t.Errorf("Test Failed - Currency Code Upper() error expected %s but recieved %s",
			expectedJSON, string(encoded))
	}
}
