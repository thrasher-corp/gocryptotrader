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
	assert.Equalf(t, Isolated, alien.M, "received '%v' expected '%v'", alien.M, Isolated)

	jason = []byte(`{"margin":"cross"}`)
	err = json.Unmarshal(jason, &alien)
	assert.NoError(t, err)
	assert.Equalf(t, Multi, alien.M, "received '%v' expected '%v'", alien.M, Multi)

	jason = []byte(`{"margin":"hello moto"}`)
	err = json.Unmarshal(jason, &alien)
	require.ErrorIs(t, err, ErrInvalidMarginType)
	assert.Equalf(t, Unknown, alien.M, "received '%v' expected '%v'", alien.M, Unknown)
}

func TestString(t *testing.T) {
	t.Parallel()
	assert.Equal(t, unknownStr, Unknown.String())
	assert.Equal(t, isolatedStr, Isolated.String())
	assert.Equal(t, multiStr, Multi.String())
	assert.Equal(t, unsetStr, Unset.String())
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
	assert.Equal(t, Unknown, resp)

	resp, err = StringToMarginType("")
	assert.NoError(t, err)
	assert.Equalf(t, Unset, resp, "received '%v' expected '%v'", resp, Unset)

	resp, err = StringToMarginType("cross")
	assert.NoError(t, err)
	assert.Equalf(t, Multi, resp, "received '%v' expected '%v'", resp, Multi)

	resp, err = StringToMarginType("multi")
	assert.NoError(t, err)
	assert.Equalf(t, Multi, resp, "received '%v' expected '%v'", resp, Multi)

	resp, err = StringToMarginType("isolated")
	assert.NoError(t, err)
	assert.Equalf(t, Isolated, resp, "received '%v' expected '%v'", resp, Isolated)

	resp, err = StringToMarginType("cash")
	assert.NoError(t, err)
	assert.Equalf(t, Cash, resp, "received '%v' expected '%v'", resp, Cash)

	resp, err = StringToMarginType("spot_isolated")
	assert.NoError(t, err)
	assert.Equalf(t, SpotIsolated, resp, "received '%v' expected '%v'", resp, SpotIsolated)
}
