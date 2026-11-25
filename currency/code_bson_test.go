package currency

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/internal/bsonwire"
	"github.com/thrasher-corp/gocryptotrader/internal/testing/bsonvalue"
)

var (
	_ bsonwire.ValueMarshaler   = Code{}
	_ bsonwire.ValueUnmarshaler = (*Code)(nil)
)

func TestCodeBSONValue(t *testing.T) {
	t.Parallel()
	cases := []bsonvalue.Case[Code]{
		{Name: "empty", In: EMPTYCODE, Want: EMPTYCODE},
		{Name: "upper case", In: NewCode("BTC"), Want: NewCode("BTC")},
		{Name: "lower case", In: NewCode("btc"), Want: NewCode("btc")},
		{Name: "lower case ETH", In: NewCode("eth"), Want: NewCode("eth")},
		{Name: "punctuation", In: NewCode("abc-123"), Want: NewCode("abc-123")},
		{Name: "upper case lowered", In: NewCode("BTC").Lower(), Want: NewCode("BTC").Lower()},
		{Name: "lower case uppered", In: NewCode("btc").Upper(), Want: NewCode("btc").Upper()},
	}
	bsonvalue.AssertRoundTrip(t, NewCode("LTC"), cases)
}

func TestCodeUnmarshalBSONValueErrors(t *testing.T) {
	t.Parallel()
	initial := NewCode("LTC")
	for name, tc := range map[string]struct {
		typ     byte
		data    []byte
		wantErr error
	}{
		"integer": {
			bsonwire.TypeInt32,
			[]byte{1, 0, 0, 0},
			bsonwire.ErrWrongType,
		},
		"null": {
			bsonwire.TypeNull,
			nil,
			bsonwire.ErrWrongType,
		},
		"symbol": {
			bsonwire.TypeSymbol,
			bsonwire.MarshalString("BTC"),
			bsonwire.ErrWrongType,
		},
		"document": {
			bsonwire.TypeEmbeddedDocument,
			bsonwire.MarshalDocument(),
			bsonwire.ErrWrongType,
		},
		"truncated string length": {
			bsonwire.TypeString,
			[]byte{5, 0},
			bsonwire.ErrLength,
		},
		"string length beyond end": {
			bsonwire.TypeString,
			[]byte{0xFF, 0, 0, 0, 'B', 'T', 'C', 0},
			bsonwire.ErrLength,
		},
		"string without NUL": {
			bsonwire.TypeString,
			[]byte{4, 0, 0, 0, 'B', 'T', 'C', 'X'},
			bsonwire.ErrMissingNUL,
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			out := initial
			err := out.UnmarshalBSONValue(tc.typ, tc.data)
			assert.ErrorIs(t, err, tc.wantErr, "UnmarshalBSONValue should reject the stored value")
			assert.ErrorContains(t, err, "currency.Code", "error should name currency.Code")
			assert.Equal(t, initial, out, "a rejected value should leave the code unchanged")
		})
	}
}
