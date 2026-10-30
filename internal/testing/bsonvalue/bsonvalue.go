// Package bsonvalue tests types that store themselves as BSON strings through bson.ValueMarshaler and
// bson.ValueUnmarshaler
package bsonvalue

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
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

// document holds a value as a document field, since BSON stores values within documents
type document[T any] struct {
	Value T `bson:"value"`
}

// pointerDocument holds a value through a pointer, which the driver sets to nil for BSON null without calling
// UnmarshalBSONValue
type pointerDocument[T any] struct {
	Value *T `bson:"value"`
}

// AssertRoundTrip stores each case's value as a document field, checks that it is stored as its string, and reads it
// back into a document holding initial, both as a value and through a pointer. Choose an initial value that no case
// reads back as, so that a value left unchanged by UnmarshalBSONValue is caught. A nil pointer is stored as null and
// reads back as nil.
func AssertRoundTrip[T fmt.Stringer](t *testing.T, initial T, cases []Case[T]) {
	t.Helper()
	t.Run("nil pointer", func(t *testing.T) {
		t.Parallel()
		data, err := bson.Marshal(pointerDocument[T]{})
		require.NoError(t, err, "Marshal must not error")
		assert.Equal(t, bson.TypeNull, bson.Raw(data).Lookup("value").Type, "a nil pointer should be stored as null")
		ptrInitial := initial
		out := pointerDocument[T]{Value: &ptrInitial}
		require.NoError(t, bson.Unmarshal(data, &out), "Unmarshal must not error")
		assert.Nil(t, out.Value, "null should read back as a nil pointer")
	})
	for _, tc := range cases {
		t.Run(tc.In.String(), func(t *testing.T) {
			t.Parallel()
			data, err := bson.Marshal(document[T]{Value: tc.In})
			require.NoError(t, err, "Marshal must not error")
			stored, ok := bson.Raw(data).Lookup("value").StringValueOK()
			require.True(t, ok, "value must be stored as a BSON string")
			assert.Equal(t, tc.In.String(), stored, "value should be stored as its string")
			out := document[T]{Value: initial}
			require.NoError(t, bson.Unmarshal(data, &out), "Unmarshal must not error")
			assert.Equal(t, tc.Want, out.Value, "value should read back as the expected value")

			data, err = bson.Marshal(pointerDocument[T]{Value: &tc.In})
			require.NoError(t, err, "Marshal of a pointer field must not error")
			ptrInitial := initial
			ptrOut := pointerDocument[T]{Value: &ptrInitial}
			require.NoError(t, bson.Unmarshal(data, &ptrOut), "Unmarshal of a pointer field must not error")
			require.NotNil(t, ptrOut.Value, "pointer field must be set")
			assert.Equal(t, tc.Want, *ptrOut.Value, "value should read back through a pointer field")
		})
	}
}

// AssertUnmarshalErrors checks that reading a T from an unrecognised string, a BSON integer, a BSON null, a BSON
// symbol holding valid's string, or malformed string bytes returns errInvalid. A symbol is encoded the same way as a string,
// so only a check of the BSON type rejects it. The driver rejects malformed bytes before calling UnmarshalBSONValue, so
// they are passed to it directly.
func AssertUnmarshalErrors[T fmt.Stringer, PT interface {
	*T
	bson.ValueUnmarshaler
}](t *testing.T, valid T, unrecognised string, errInvalid error) {
	t.Helper()
	for name, data := range map[string][]byte{
		"truncated string length":  {5, 0},
		"string length beyond end": {0xFF, 0, 0, 0, 'G', 0},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var v T
			assert.ErrorIs(t, PT(&v).UnmarshalBSONValue(byte(bson.TypeString), data), errInvalid, "UnmarshalBSONValue should reject malformed bytes")
		})
	}
	for name, value := range map[string]any{
		"unrecognised string": unrecognised,
		"integer":             int32(1),
		"null":                nil,
		"symbol":              bson.Symbol(valid.String()),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			data, err := bson.Marshal(bson.D{{Key: "value", Value: value}})
			require.NoError(t, err, "Marshal must not error")
			assert.ErrorIs(t, bson.Unmarshal(data, &document[T]{}), errInvalid, "Unmarshal should reject the stored value")
		})
	}
}
