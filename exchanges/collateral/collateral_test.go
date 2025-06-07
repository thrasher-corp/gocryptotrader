package collateral

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

func TestValidCollateralType(t *testing.T) {
	t.Parallel()
	if !SingleMode.Valid() {
		t.Fatal("expected 'true', received 'false'")
	}
	if !MultiMode.Valid() {
		t.Fatal("expected 'true', received 'false'")
	}
	if !PortfolioMode.Valid() {
		t.Fatal("expected 'true', received 'false'")
	}
	if !SpotFuturesMode.Valid() {
		t.Fatal("expected 'true', received 'false'")
	}
	if UnsetMode.Valid() {
		t.Fatal("expected 'false', received 'true'")
	}
	if UnknownMode.Valid() {
		t.Fatal("expected 'false', received 'true'")
	}
	if Mode(137).Valid() {
		t.Fatal("expected 'false', received 'true'")
	}
}

func TestUnmarshalJSONCollateralType(t *testing.T) {
	t.Parallel()
	type martian struct {
		M Mode `json:"collateral"`
	}

	var alien martian
	jason := []byte(`{"collateral":"single"}`)
	err := json.Unmarshal(jason, &alien)
	if err != nil {
		t.Error(err)
	}
	if alien.M != SingleMode {
		t.Errorf("received '%v' expected 'single'", alien.M)
	}

	jason = []byte(`{"collateral":"multi"}`)
	err = json.Unmarshal(jason, &alien)
	if err != nil {
		t.Error(err)
	}
	if alien.M != MultiMode {
		t.Errorf("received '%v' expected 'Multi'", alien.M)
	}

	jason = []byte(`{"collateral":"portfolio"}`)
	err = json.Unmarshal(jason, &alien)
	if err != nil {
		t.Error(err)
	}
	if alien.M != PortfolioMode {
		t.Errorf("received '%v' expected 'Portfolio'", alien.M)
	}

	jason = []byte(`{"collateral":"hello moto"}`)
	err = json.Unmarshal(jason, &alien)
	assert.ErrorIs(t, err, ErrInvalidCollateralMode)

	if alien.M != UnknownMode {
		t.Errorf("received '%v' expected 'UnknownMode'", alien.M)
	}
}

func TestStringCollateralType(t *testing.T) {
	t.Parallel()
	if UnknownMode.String() != unknownCollateralStr {
		t.Errorf("received '%v' expected '%v'", UnknownMode.String(), unknownCollateralStr)
	}
	if SingleMode.String() != singleCollateralStr {
		t.Errorf("received '%v' expected '%v'", SingleMode.String(), singleCollateralStr)
	}
	if MultiMode.String() != multiCollateralStr {
		t.Errorf("received '%v' expected '%v'", MultiMode.String(), multiCollateralStr)
	}
	if PortfolioMode.String() != portfolioCollateralStr {
		t.Errorf("received '%v' expected '%v'", PortfolioMode.String(), portfolioCollateralStr)
	}
	if UnsetMode.String() != unsetCollateralStr {
		t.Errorf("received '%v' expected '%v'", UnsetMode.String(), unsetCollateralStr)
	}
}

func TestUpperCollateralType(t *testing.T) {
	t.Parallel()
	if UnknownMode.Upper() != strings.ToUpper(unknownCollateralStr) {
		t.Errorf("received '%v' expected '%v'", UnknownMode.Upper(), strings.ToUpper(unknownCollateralStr))
	}
	if SingleMode.Upper() != strings.ToUpper(singleCollateralStr) {
		t.Errorf("received '%v' expected '%v'", SingleMode.Upper(), strings.ToUpper(singleCollateralStr))
	}
	if MultiMode.Upper() != strings.ToUpper(multiCollateralStr) {
		t.Errorf("received '%v' expected '%v'", MultiMode.Upper(), strings.ToUpper(multiCollateralStr))
	}
	if PortfolioMode.Upper() != strings.ToUpper(portfolioCollateralStr) {
		t.Errorf("received '%v' expected '%v'", PortfolioMode.Upper(), strings.ToUpper(portfolioCollateralStr))
	}
	if UnsetMode.Upper() != strings.ToUpper(unsetCollateralStr) {
		t.Errorf("received '%v' expected '%v'", UnsetMode.Upper(), strings.ToUpper(unsetCollateralStr))
	}
}

func TestIsValidCollateralTypeString(t *testing.T) {
	t.Parallel()
	if IsValidCollateralModeString("lol") {
		t.Fatal("expected 'false', received 'true'")
	}
	if !IsValidCollateralModeString("single") {
		t.Fatal("expected 'true', received 'false'")
	}
	if !IsValidCollateralModeString("multi") {
		t.Fatal("expected 'true', received 'false'")
	}
	if !IsValidCollateralModeString("portfolio") {
		t.Fatal("expected 'true', received 'false'")
	}
	if !IsValidCollateralModeString("unset") {
		t.Fatal("expected 'true', received 'false'")
	}
	if IsValidCollateralModeString("") {
		t.Fatal("expected 'false', received 'true'")
	}
	if IsValidCollateralModeString("unknown") {
		t.Fatal("expected 'false', received 'true'")
	}
}

func TestStringToCollateralType(t *testing.T) {
	t.Parallel()
	resp, err := StringToMode("lol")
	assert.ErrorIs(t, err, ErrInvalidCollateralMode)

	if resp != UnknownMode {
		t.Errorf("received '%v' expected '%v'", resp, UnknownMode)
	}

	resp, err = StringToMode("")
	if err != nil {
		t.Error(err)
	}
	if resp != UnsetMode {
		t.Errorf("received '%v' expected '%v'", resp, UnsetMode)
	}

	resp, err = StringToMode("single")
	if err != nil {
		t.Error(err)
	}
	if resp != SingleMode {
		t.Errorf("received '%v' expected '%v'", resp, SingleMode)
	}

	resp, err = StringToMode("multi")
	if err != nil {
		t.Error(err)
	}
	if resp != MultiMode {
		t.Errorf("received '%v' expected '%v'", resp, MultiMode)
	}

	resp, err = StringToMode("portfolio")
	if err != nil {
		t.Error(err)
	}
	if resp != PortfolioMode {
		t.Errorf("received '%v' expected '%v'", resp, PortfolioMode)
	}
}
