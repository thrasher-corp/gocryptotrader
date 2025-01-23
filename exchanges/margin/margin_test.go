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
	require.True(t, NoMargin.Valid())
	require.True(t, SpotIsolated.Valid())
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
	assert.Equal(t, Isolated.String(), alien.M.String())

	jason = []byte(`{"margin":"cross"}`)
	err = json.Unmarshal(jason, &alien)
	assert.NoError(t, err)
	assert.Equal(t, Multi.String(), alien.M.String())

	jason = []byte(`{"margin":"cash"}`)
	err = json.Unmarshal(jason, &alien)
	assert.NoError(t, err)
	assert.Equal(t, NoMargin.String(), alien.M.String())

	jason = []byte(`{"margin":"spot_isolated"}`)
	err = json.Unmarshal(jason, &alien)
	assert.NoError(t, err)
	assert.Equal(t, SpotIsolated.String(), alien.M.String())

	jason = []byte(`{"margin":"hello moto"}`)
	err = json.Unmarshal(jason, &alien)
	require.ErrorIs(t, err, ErrInvalidMarginType)
	assert.Equal(t, Unknown.String(), alien.M.String())
}

func TestString(t *testing.T) {
	t.Parallel()
	assert.Equal(t, unknownStr, Unknown.String())
	assert.Equal(t, isolatedStr, Isolated.String())
	assert.Equal(t, multiStr, Multi.String())
	assert.Equal(t, unsetStr, Unset.String())
	assert.Equal(t, spotIsolatedStr, SpotIsolated.String())
	assert.Equal(t, cashStr, NoMargin.String())
	assert.Equal(t, "", Type(30).String())
}

func TestUpper(t *testing.T) {
	t.Parallel()
	assert.Equal(t, strings.ToUpper(unknownStr), Unknown.Upper())
	assert.Equal(t, strings.ToUpper(isolatedStr), Isolated.Upper())
	assert.Equal(t, strings.ToUpper(multiStr), Multi.Upper())
	assert.Equal(t, strings.ToUpper(spotIsolatedStr), SpotIsolated.Upper())
	assert.Equal(t, strings.ToUpper(cashStr), NoMargin.Upper())
	assert.Equal(t, strings.ToUpper(unsetStr), Unset.Upper())
}

func TestIsValidString(t *testing.T) {
	t.Parallel()
	assert.False(t, IsValidString("lol"))
	assert.True(t, IsValidString("spot_isolated"))
	assert.True(t, IsValidString("cash"))
	assert.True(t, IsValidString("isolated"))
	assert.True(t, IsValidString("cross"))
	assert.True(t, IsValidString("multi"))
	assert.True(t, IsValidString(""))
	assert.False(t, IsValidString("unknown"))
}

func TestStringToMarginType(t *testing.T) {
	t.Parallel()
	resp, err := StringToMarginType("lol")
	assert.ErrorIs(t, err, ErrInvalidMarginType)
	assert.Equal(t, Unknown, resp)

	resp, err = StringToMarginType("")
	assert.NoError(t, err)
	assert.Equal(t, Unset.String(), resp.String())

	resp, err = StringToMarginType("cross")
	assert.NoError(t, err)
	assert.Equal(t, Multi.String(), resp.String())

	resp, err = StringToMarginType("multi")
	assert.NoError(t, err)
	assert.Equal(t, Multi.String(), resp.String())

	resp, err = StringToMarginType("isolated")
	assert.NoError(t, err)
	assert.Equal(t, Isolated.String(), resp.String())

	resp, err = StringToMarginType("cash")
	assert.NoError(t, err)
	assert.Equal(t, NoMargin.String(), resp.String())

	resp, err = StringToMarginType("spot_isolated")
	assert.NoError(t, err)
	assert.Equal(t, SpotIsolated.String(), resp.String())
}
