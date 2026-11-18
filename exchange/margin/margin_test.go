package margin

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
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
	for name, tc := range map[string]struct {
		in   string
		want Type
		err  error
	}{
		"isolated":     {`{"margin":"isolated"}`, Isolated, nil},
		"cross":        {`{"margin":"cross"}`, Multi, nil},
		"cash":         {`{"margin":"cash"}`, NoMargin, nil},
		"spotIsolated": {`{"margin":"spot_isolated"}`, SpotIsolated, nil},
		"invalid":      {`{"margin":"hello moto"}`, Unknown, ErrInvalidMarginType},
		"unset":        {`{"margin":""}`, Unset, nil},
		"":             {`{"margin":""}`, Unset, nil},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var alien struct {
				M Type `json:"margin"`
			}
			err := json.Unmarshal([]byte(tc.in), &alien)
			assert.ErrorIs(t, err, tc.err)
			assert.Equal(t, tc.want, alien.M)
		})
	}
}

func TestMarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		in   Type
		want string
	}{
		{Isolated, fmt.Sprintf(`%q`, isolatedStr)},
		{Multi, fmt.Sprintf(`%q`, multiStr)},
		{NoMargin, fmt.Sprintf(`%q`, cashStr)},
		{SpotIsolated, fmt.Sprintf(`%q`, spotIsolatedStr)},
		{Type(uint8(123)), fmt.Sprintf(`%q`, unknownStr)},
		{Unset, fmt.Sprintf(`%q`, unsetStr)},
	} {
		t.Run(tc.want, func(t *testing.T) {
			t.Parallel()
			resp, err := json.Marshal(tc.in)
			require.NoError(t, err)
			assert.Equal(t, tc.want, string(resp))
		})
	}
}

func TestString(t *testing.T) {
	t.Parallel()
	assert.Equal(t, unknownStr, Unknown.String())
	assert.Equal(t, isolatedStr, Isolated.String())
	assert.Equal(t, multiStr, Multi.String())
	assert.Equal(t, unsetStr, Unset.String())
	assert.Equal(t, spotIsolatedStr, SpotIsolated.String())
	assert.Equal(t, cashStr, NoMargin.String())
	assert.Equal(t, unknownStr, Type(30).String())
	assert.Empty(t, Unset.String())
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
	for label, v := range map[string]struct {
		MarginType Type
		Error      error
	}{
		"lol":           {Unknown, ErrInvalidMarginType},
		"":              {Unset, nil},
		"cross":         {Multi, nil},
		"multi":         {Multi, nil},
		"isolated":      {Isolated, nil},
		"cash":          {NoMargin, nil},
		"spot_isolated": {SpotIsolated, nil},
	} {
		resp, err := StringToMarginType(label)
		assert.ErrorIs(t, err, v.Error)
		assert.Equal(t, v.MarginType.String(), resp.String())
	}
}
