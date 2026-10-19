package coinbase

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDerivativesOrderPriceUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for input, exp := range map[string]DerivativesOrderPrice{
		`65000.5`:        {Value: 65000.5},
		`"65000.5"`:      {Value: 65000.5},
		`"market_price"`: {Market: true},
		`null`:           {},
	} {
		var price DerivativesOrderPrice
		require.NoErrorf(t, price.UnmarshalJSON([]byte(input)), "UnmarshalJSON must not error for %s", input)
		assert.Equalf(t, exp, price, "UnmarshalJSON should decode %s", input)
	}
	var price DerivativesOrderPrice
	for _, input := range []string{`"cheap"`, `true`, `"unterminated`} {
		assert.ErrorIsf(t, price.UnmarshalJSON([]byte(input)), errInvalidDerivativesOrderPrice, "UnmarshalJSON should refuse %s", input)
	}
}

func TestUnmarshalTuple(t *testing.T) {
	t.Parallel()
	var action string
	var price, amount float64
	require.NoError(t, unmarshalTuple([]byte(`["new",65000.5,10]`), &action, &price, &amount), "unmarshalTuple must not error")
	assert.Equal(t, "new", action, "unmarshalTuple should decode the first element")
	assert.Equal(t, 65000.5, price, "unmarshalTuple should decode the second element")
	assert.Equal(t, 10.0, amount, "unmarshalTuple should decode the third element")
	require.NoError(t, unmarshalTuple([]byte(` null `), &action), "unmarshalTuple must not error for null")
	assert.Equal(t, "new", action, "unmarshalTuple should leave the targets unchanged for null")
	assert.ErrorIs(t, unmarshalTuple([]byte(`["new",1]`), &action, &price, &amount), errInvalidTuple, "unmarshalTuple should refuse too few elements")
	assert.ErrorIs(t, unmarshalTuple([]byte(`{"a":1}`), &action), errInvalidTuple, "unmarshalTuple should refuse an object")
	assert.ErrorIs(t, unmarshalTuple([]byte(`[1]`), &action), errInvalidTuple, "unmarshalTuple should refuse an element of the wrong type")
}
