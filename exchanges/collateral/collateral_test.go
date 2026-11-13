package collateral

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

func TestValidCollateralType(t *testing.T) {
	t.Parallel()
	require.True(t, SingleMode.Valid(), "Mode.Valid must return true for SingleMode")
	require.True(t, MultiMode.Valid(), "Mode.Valid must return true for MultiMode")
	require.True(t, PortfolioMode.Valid(), "Mode.Valid must return true for PortfolioMode")
	require.True(t, SpotFuturesMode.Valid(), "Mode.Valid must return true for SpotFuturesMode")
	require.False(t, UnsetMode.Valid(), "Mode.Valid must return false for UnsetMode")
	require.False(t, UnknownMode.Valid(), "Mode.Valid must return false for UnknownMode")
	require.False(t, Mode(137).Valid(), "Mode.Valid must return false for an unsupported value")
}

func TestUnmarshalJSONCollateralType(t *testing.T) {
	t.Parallel()
	type martian struct {
		M Mode `json:"collateral"`
	}

	var alien martian
	jason := []byte(`{"collateral":"single"}`)
	err := json.Unmarshal(jason, &alien)
	require.NoError(t, err, "json.Unmarshal must not error for SingleMode")
	assert.Equal(t, SingleMode, alien.M, "json.Unmarshal should set Mode to SingleMode")

	jason = []byte(`{"collateral":"multi"}`)
	err = json.Unmarshal(jason, &alien)
	require.NoError(t, err, "json.Unmarshal must not error for MultiMode")
	assert.Equal(t, MultiMode, alien.M, "json.Unmarshal should set Mode to MultiMode")

	jason = []byte(`{"collateral":"portfolio"}`)
	err = json.Unmarshal(jason, &alien)
	require.NoError(t, err, "json.Unmarshal must not error for PortfolioMode")
	assert.Equal(t, PortfolioMode, alien.M, "json.Unmarshal should set Mode to PortfolioMode")

	jason = []byte(`{"collateral":"hello moto"}`)
	err = json.Unmarshal(jason, &alien)
	assert.ErrorIs(t, err, ErrInvalidCollateralMode, "json.Unmarshal should return ErrInvalidCollateralMode for an unknown mode")
	assert.Equal(t, UnknownMode, alien.M, "json.Unmarshal should set Mode to UnknownMode")

	mode := SingleMode
	err = mode.UnmarshalJSON([]byte(`1`))
	assert.Error(t, err, "Mode.UnmarshalJSON should error for a non-string value")
	assert.Equal(t, SingleMode, mode, "Mode.UnmarshalJSON should leave Mode unchanged after a decoding error")
}

func TestStringCollateralType(t *testing.T) {
	t.Parallel()
	assert.Equal(t, unknownCollateralStr, UnknownMode.String(), "Mode.String should return the correct value for UnknownMode")
	assert.Equal(t, singleCollateralStr, SingleMode.String(), "Mode.String should return the correct value for SingleMode")
	assert.Equal(t, multiCollateralStr, MultiMode.String(), "Mode.String should return the correct value for MultiMode")
	assert.Equal(t, portfolioCollateralStr, PortfolioMode.String(), "Mode.String should return the correct value for PortfolioMode")
	assert.Equal(t, unsetCollateralStr, UnsetMode.String(), "Mode.String should return the correct value for UnsetMode")
	assert.Equal(t, spotFuturesCollateralStr, SpotFuturesMode.String(), "Mode.String should return the correct value for SpotFuturesMode")
	assert.Empty(t, Mode(137).String(), "Mode.String should return an empty value for an unsupported mode")
}

func TestUpperCollateralType(t *testing.T) {
	t.Parallel()
	assert.Equal(t, strings.ToUpper(unknownCollateralStr), UnknownMode.Upper(), "Mode.Upper should return the correct value for UnknownMode")
	assert.Equal(t, strings.ToUpper(singleCollateralStr), SingleMode.Upper(), "Mode.Upper should return the correct value for SingleMode")
	assert.Equal(t, strings.ToUpper(multiCollateralStr), MultiMode.Upper(), "Mode.Upper should return the correct value for MultiMode")
	assert.Equal(t, strings.ToUpper(portfolioCollateralStr), PortfolioMode.Upper(), "Mode.Upper should return the correct value for PortfolioMode")
	assert.Equal(t, strings.ToUpper(unsetCollateralStr), UnsetMode.Upper(), "Mode.Upper should return the correct value for UnsetMode")
}

func TestIsValidCollateralTypeString(t *testing.T) {
	t.Parallel()
	require.False(t, IsValidCollateralModeString("lol"), "IsValidCollateralModeString must return false for an invalid value")
	require.True(t, IsValidCollateralModeString("single"), "IsValidCollateralModeString must return true for single")
	require.True(t, IsValidCollateralModeString("multi"), "IsValidCollateralModeString must return true for multi")
	require.True(t, IsValidCollateralModeString("portfolio"), "IsValidCollateralModeString must return true for portfolio")
	require.True(t, IsValidCollateralModeString("unset"), "IsValidCollateralModeString must return true for unset")
	require.False(t, IsValidCollateralModeString(""), "IsValidCollateralModeString must return false for an empty value")
	require.False(t, IsValidCollateralModeString("unknown"), "IsValidCollateralModeString must return false for unknown")
}

func TestStringToCollateralType(t *testing.T) {
	t.Parallel()
	resp, err := StringToMode("lol")
	assert.ErrorIs(t, err, ErrInvalidCollateralMode, "StringToMode should return ErrInvalidCollateralMode for an invalid value")
	assert.Equal(t, UnknownMode, resp, "StringToMode should return UnknownMode for an invalid value")

	resp, err = StringToMode("")
	require.NoError(t, err, "StringToMode must not error for an empty value")
	assert.Equal(t, UnsetMode, resp, "StringToMode should return UnsetMode for an empty value")

	resp, err = StringToMode("single")
	require.NoError(t, err, "StringToMode must not error for single")
	assert.Equal(t, SingleMode, resp, "StringToMode should return SingleMode for single")

	resp, err = StringToMode("multi")
	require.NoError(t, err, "StringToMode must not error for multi")
	assert.Equal(t, MultiMode, resp, "StringToMode should return MultiMode for multi")

	resp, err = StringToMode("portfolio")
	require.NoError(t, err, "StringToMode must not error for portfolio")
	assert.Equal(t, PortfolioMode, resp, "StringToMode should return PortfolioMode for portfolio")
}
