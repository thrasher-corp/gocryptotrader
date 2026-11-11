// Package bsonvalue tests types that store themselves as BSON strings through MarshalBSONValue and
// UnmarshalBSONValue.
package bsonvalue

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/internal/bsonwire"
)

// Case is a value to store, and the value it should read back as
type Case[T any] struct {
	In   T
	Want T
}

// Named returns a Case for each value, reading it back as itself
func Named[T any](values ...T) []Case[T] {
	cases := make([]Case[T], 0, len(values))
	for _, v := range values {
		cases = append(cases, Case[T]{In: v, Want: v})
	}
	return cases
}

// AssertRoundTrip checks that each case's value is stored as its string, and reads it back into a value holding initial.
// Choose an initial value that no case reads back as, so that a value left unchanged by UnmarshalBSONValue is caught.
func AssertRoundTrip[T interface {
	fmt.Stringer
	bsonwire.ValueMarshaler
}, PT interface {
	*T
	bsonwire.ValueUnmarshaler
}](t *testing.T, initial T, cases []Case[T]) {
	t.Helper()
	for _, tc := range cases {
		t.Run(fmt.Sprintf("%v %#v", tc.In, tc.In), func(t *testing.T) {
			t.Parallel()
			typ, data, err := tc.In.MarshalBSONValue()
			require.NoError(t, err, "MarshalBSONValue must not error")
			assert.Equal(t, bsonwire.TypeString, typ, "value should be stored as a BSON string")
			assert.Equal(t, bsonwire.MarshalString(tc.In.String()), data, "value should be stored as its string")
			out := initial
			require.NoError(t, PT(&out).UnmarshalBSONValue(typ, data), "UnmarshalBSONValue must not error")
			assert.Equal(t, tc.Want, out, "value should read back as the expected value")
		})
	}
}

// AssertUnmarshalErrors checks that reading a T from an unrecognised string, a BSON integer, a BSON null, a BSON
// symbol holding valid's string, or malformed string bytes returns errInvalid and leaves a value holding initial
// unchanged. A symbol is encoded the same way as a string, so only a check of the BSON type rejects it. Pass the
// initial value given to AssertRoundTrip, which no case reads back as.
func AssertUnmarshalErrors[T fmt.Stringer, PT interface {
	*T
	bsonwire.ValueUnmarshaler
}](t *testing.T, initial, valid T, unrecognised string, errInvalid error) {
	t.Helper()
	for name, data := range map[string][]byte{
		"truncated string length":  {5, 0},
		"string length beyond end": {0xFF, 0, 0, 0, 'G', 0},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			v := initial
			assert.ErrorIs(t, PT(&v).UnmarshalBSONValue(bsonwire.TypeString, data), errInvalid, "UnmarshalBSONValue should reject malformed bytes")
			assert.Equal(t, initial, v, "a rejected value should leave the value unchanged")
		})
	}
	for name, tc := range map[string]struct {
		typ  byte
		data []byte
	}{
		"unrecognised string": {bsonwire.TypeString, bsonwire.MarshalString(unrecognised)},
		"integer":             {bsonwire.TypeInt32, []byte{1, 0, 0, 0}},
		"null":                {bsonwire.TypeNull, nil},
		"symbol":              {bsonwire.TypeSymbol, bsonwire.MarshalString(valid.String())},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			v := initial
			assert.ErrorIs(t, PT(&v).UnmarshalBSONValue(tc.typ, tc.data), errInvalid, "UnmarshalBSONValue should reject the stored value")
			assert.Equal(t, initial, v, "a rejected value should leave the value unchanged")
		})
	}
}
