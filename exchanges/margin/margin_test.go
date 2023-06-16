package margin

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestValid(t *testing.T) {
	t.Parallel()
	if !Isolated.Valid() {
		t.Fatal("expected 'true', received 'false'")
	}
	if !Multi.Valid() {
		t.Fatal("expected 'true', received 'false'")
	}
	if Unset.Valid() {
		t.Fatal("expected 'false', received 'true'")
	}
	if Unknown.Valid() {
		t.Fatal("expected 'false', received 'true'")
	}
	if Type(137).Valid() {
		t.Fatal("expected 'false', received 'true'")
	}
}

func TestUnmarshalJSON(t *testing.T) {
	t.Parallel()
	type martian struct {
		M Type `json:"margin"`
	}

	var alien martian
	jason := []byte(`{"margin":"isolated"}`)
	err := json.Unmarshal(jason, &alien)
	if err != nil {
		t.Error(err)
	}
	if alien.M != Isolated {
		t.Errorf("received '%v' expected 'isolated'", alien.M)
	}

	jason = []byte(`{"margin":"cross"}`)
	err = json.Unmarshal(jason, &alien)
	if err != nil {
		t.Error(err)
	}
	if alien.M != Multi {
		t.Errorf("received '%v' expected 'Multi'", alien.M)
	}

	jason = []byte(`{"margin":"hello moto"}`)
	err = json.Unmarshal(jason, &alien)
	if !errors.Is(err, ErrInvalidMarginType) {
		t.Error(err)
	}
	if alien.M != Unknown {
		t.Errorf("received '%v' expected 'isolated'", alien.M)
	}
}

func TestString(t *testing.T) {
	t.Parallel()
	if Unknown.String() != unknownStr {
		t.Errorf("received '%v' expected '%v'", Unknown.String(), unknownStr)
	}
	if Isolated.String() != isolatedStr {
		t.Errorf("received '%v' expected '%v'", Isolated.String(), isolatedStr)
	}
	if Multi.String() != multiStr {
		t.Errorf("received '%v' expected '%v'", Multi.String(), multiStr)
	}
	if Unset.String() != unsetStr {
		t.Errorf("received '%v' expected '%v'", Unset.String(), unsetStr)
	}
}

func TestUpper(t *testing.T) {
	t.Parallel()
	if Unknown.Upper() != strings.ToUpper(unknownStr) {
		t.Errorf("received '%v' expected '%v'", Unknown.String(), strings.ToUpper(unknownStr))
	}
	if Isolated.Upper() != strings.ToUpper(isolatedStr) {
		t.Errorf("received '%v' expected '%v'", Isolated.String(), strings.ToUpper(isolatedStr))
	}
	if Multi.Upper() != strings.ToUpper(multiStr) {
		t.Errorf("received '%v' expected '%v'", Multi.String(), strings.ToUpper(multiStr))
	}
	if Unset.Upper() != strings.ToUpper(unsetStr) {
		t.Errorf("received '%v' expected '%v'", Unset.String(), strings.ToUpper(unsetStr))
	}
}

func TestIsValidString(t *testing.T) {
	t.Parallel()
	if IsValidString("lol") {
		t.Fatal("expected 'false', received 'true'")
	}
	if !IsValidString("isolated") {
		t.Fatal("expected 'true', received 'false'")
	}
	if !IsValidString("cross") {
		t.Fatal("expected 'true', received 'false'")
	}
	if !IsValidString("multi") {
		t.Fatal("expected 'true', received 'false'")
	}
	if !IsValidString("unset") {
		t.Fatal("expected 'true', received 'false'")
	}
	if IsValidString("") {
		t.Fatal("expected 'false', received 'true'")
	}
	if IsValidString("unknown") {
		t.Fatal("expected 'false', received 'true'")
	}
}

func TestStringToMarginType(t *testing.T) {
	t.Parallel()
	resp, err := StringToMarginType("lol")
	if !errors.Is(err, ErrInvalidMarginType) {
		t.Error(err)
	}
	if resp != Unknown {
		t.Errorf("received '%v' expected '%v'", resp, Unknown)
	}

	resp, err = StringToMarginType("")
	if err != nil {
		t.Error(err)
	}
	if resp != Unset {
		t.Errorf("received '%v' expected '%v'", resp, Unset)
	}

	resp, err = StringToMarginType("cross")
	if err != nil {
		t.Error(err)
	}
	if resp != Multi {
		t.Errorf("received '%v' expected '%v'", resp, Multi)
	}

	resp, err = StringToMarginType("multi")
	if err != nil {
		t.Error(err)
	}
	if resp != Multi {
		t.Errorf("received '%v' expected '%v'", resp, Multi)
	}

	resp, err = StringToMarginType("isolated")
	if err != nil {
		t.Error(err)
	}
	if resp != Isolated {
		t.Errorf("received '%v' expected '%v'", resp, Isolated)
	}
}
