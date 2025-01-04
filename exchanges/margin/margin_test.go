package margin

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValid(t *testing.T) {
	t.Parallel()
	require.True(t, Isolated.Valid())
	require.True(t, Multi.Valid())
	require.False(t, Unset.Valid())
	require.False(t, Unknown.Valid())
	require.False(t, Type(137).Valid())
}

func TestUnmarshalJSON(t *testing.T) {
	t.Parallel()
	type martian struct {
		M Type `json:"margin"`
	}

	var alien martian
	jason := []byte(`{"margin":"isolated"}`)
	err := json.Unmarshal(jason, &alien)
	assert.NoError(t, err)
	assert.Equalf(t, alien.M, Isolated, "received '%v' expected '%v'", alien.M, Isolated)

	jason = []byte(`{"margin":"cross"}`)
	err = json.Unmarshal(jason, &alien)
	assert.NoError(t, err)
	assert.Equalf(t, alien.M, Multi, "received '%v' expected '%v'", alien.M, Multi)

	jason = []byte(`{"margin":"hello moto"}`)
	err = json.Unmarshal(jason, &alien)
	require.ErrorIs(t, err, ErrInvalidMarginType)
	assert.Equalf(t, alien.M, Unknown, "received '%v' expected '%v'", alien.M, Unknown)
}

func TestString(t *testing.T) {
	t.Parallel()
	assert.Equal(t, Unknown.String(), unknownStr)
	assert.Equal(t, Isolated.String(), isolatedStr)
	assert.Equal(t, Multi.String(), multiStr)
	assert.Equal(t, Unset.String(), unsetStr)
}

func TestUpper(t *testing.T) {
	t.Parallel()
	assert.Equal(t, Unknown.Upper(), strings.ToUpper(unknownStr))
	assert.Equal(t, Isolated.Upper(), strings.ToUpper(isolatedStr))
	assert.Equal(t, Multi.Upper(), strings.ToUpper(multiStr))
	assert.Equal(t, Unset.Upper(), strings.ToUpper(unsetStr))
}

func TestIsValidString(t *testing.T) {
	t.Parallel()
	require.False(t, IsValidString("lol"))
	require.True(t, IsValidString("isolated"))
	require.True(t, IsValidString("cross"))
	require.True(t, IsValidString("multi"))
	require.True(t, IsValidString("unset"))
	require.False(t, IsValidString(""))
	require.False(t, IsValidString("unknown"))
}

func TestStringToMarginType(t *testing.T) {
	t.Parallel()
	resp, err := StringToMarginType("lol")
	assert.ErrorIs(t, err, ErrInvalidMarginType)
	assert.Equal(t, resp, Unknown)

	resp, err = StringToMarginType("")
	assert.NoError(t, err)
	assert.Equalf(t, resp, Unset, "received '%v' expected '%v'", resp, Unset)

	resp, err = StringToMarginType("cross")
	assert.NoError(t, err)
	assert.Equalf(t, resp, Multi, "received '%v' expected '%v'", resp, Multi)

	resp, err = StringToMarginType("multi")
	assert.NoError(t, err)
	assert.Equalf(t, resp, Multi, "received '%v' expected '%v'", resp, Multi)

	resp, err = StringToMarginType("isolated")
	assert.NoError(t, err)
	assert.Equalf(t, resp, Isolated, "received '%v' expected '%v'", resp, Isolated)

	resp, err = StringToMarginType("cash")
	assert.NoError(t, err)
	assert.Equalf(t, resp, Cash, "received '%v' expected '%v'", resp, Cash)

	resp, err = StringToMarginType("spot_isolated")
	assert.NoError(t, err)
	assert.Equalf(t, resp, SpotIsolated, "received '%v' expected '%v'", resp, SpotIsolated)
}
