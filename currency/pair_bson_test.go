package currency

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/internal/bsonwire"
)

var (
	_ bsonwire.ValueMarshaler   = Pair{}
	_ bsonwire.ValueUnmarshaler = (*Pair)(nil)
)

func TestPairBSONValue(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		pair   Pair
		stored []byte
	}{
		"empty": {
			EMPTYPAIR,
			bsonwire.MarshalDocument(),
		},
		"no delimiter": {
			NewBTCUSDT(),
			bsonwire.MarshalDocument(bsonwire.Field{Key: "b", Value: "BTC"}, bsonwire.Field{Key: "q", Value: "USDT"}),
		},
		"delimiter": {
			NewPairWithDelimiter("BTC", "USDT", "-"),
			bsonwire.MarshalDocument(bsonwire.Field{Key: "b", Value: "BTC"}, bsonwire.Field{Key: "d", Value: "-"}, bsonwire.Field{Key: "q", Value: "USDT"}),
		},
		"lower case": {
			NewPairWithDelimiter("eth", "usdc", "/"),
			bsonwire.MarshalDocument(bsonwire.Field{Key: "b", Value: "eth"}, bsonwire.Field{Key: "d", Value: "/"}, bsonwire.Field{Key: "q", Value: "usdc"}),
		},
		"long delimiter": {
			NewPairWithDelimiter("BTC", "USDT", "-_-"),
			bsonwire.MarshalDocument(bsonwire.Field{Key: "b", Value: "BTC"}, bsonwire.Field{Key: "d", Value: "-_-"}, bsonwire.Field{Key: "q", Value: "USDT"}),
		},
		"control in delimiter": {
			NewPairWithDelimiter("BTC", "USDT", "\x1F"),
			bsonwire.MarshalDocument(bsonwire.Field{Key: "b", Value: "BTC"}, bsonwire.Field{Key: "d", Value: "\x1F"}, bsonwire.Field{Key: "q", Value: "USDT"}),
		},
		"base only": {
			NewPair(BTC, EMPTYCODE),
			bsonwire.MarshalDocument(bsonwire.Field{Key: "b", Value: "BTC"}),
		},
		"quote only": {
			NewPairWithDelimiter("", "USDT", "-"),
			bsonwire.MarshalDocument(bsonwire.Field{Key: "d", Value: "-"}, bsonwire.Field{Key: "q", Value: "USDT"}),
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			typ, data, err := tc.pair.MarshalBSONValue()
			require.NoError(t, err, "MarshalBSONValue must not error")
			assert.Equal(t, bsonwire.TypeEmbeddedDocument, typ, "pair should be stored as a BSON document")
			assert.Equal(t, tc.stored, data, "pair should be stored as its non-empty fields")

			out := NewPairWithDelimiter("LTC", "XRP", "_")
			require.NoError(t, out.UnmarshalBSONValue(typ, data), "UnmarshalBSONValue must not error")
			assert.Equal(t, tc.pair, out, "pair should keep its case and delimiter")
			assert.True(t, tc.pair.Equal(out), "pair should read back as the same currency codes")
		})
	}
}

func TestPairUnmarshalBSONValueErrors(t *testing.T) {
	t.Parallel()
	initial := NewPairWithDelimiter("LTC", "XRP", "_")
	for name, tc := range map[string]struct {
		typ  byte
		data []byte
	}{
		"string": {
			bsonwire.TypeString,
			bsonwire.MarshalString("BTC-USDT"),
		},
		"integer": {
			bsonwire.TypeInt32,
			[]byte{1, 0, 0, 0},
		},
		"null": {
			bsonwire.TypeNull,
			nil,
		},
		"array": {
			bsonwire.TypeArray,
			bsonwire.MarshalDocument(bsonwire.Field{Key: "0", Value: "BTC"}, bsonwire.Field{Key: "1", Value: "-"}, bsonwire.Field{Key: "2", Value: "USDT"}),
		},
		"empty array": {
			bsonwire.TypeArray,
			bsonwire.MarshalDocument(),
		},
		"non-string field": {
			bsonwire.TypeEmbeddedDocument,
			[]byte{0x0C, 0x00, 0x00, 0x00, bsonwire.TypeInt32, 'b', 0x00, 0x01, 0x00, 0x00, 0x00, 0x00},
		},
		"null field": {
			bsonwire.TypeEmbeddedDocument,
			[]byte{0x08, 0x00, 0x00, 0x00, bsonwire.TypeNull, 'q', 0x00, 0x00},
		},
		"unexpected field": {
			bsonwire.TypeEmbeddedDocument,
			bsonwire.MarshalDocument(bsonwire.Field{Key: "b", Value: "BTC"}, bsonwire.Field{Key: "base", Value: "BTC"}),
		},
		"unexpected last": {
			bsonwire.TypeEmbeddedDocument,
			bsonwire.MarshalDocument(bsonwire.Field{Key: "b", Value: "BTC"}, bsonwire.Field{Key: "d", Value: "-"}, bsonwire.Field{Key: "q", Value: "USDT"}, bsonwire.Field{Key: "base", Value: "BTC"}),
		},
		"long field names": {
			bsonwire.TypeEmbeddedDocument,
			bsonwire.MarshalDocument(bsonwire.Field{Key: "base", Value: "BTC"}, bsonwire.Field{Key: "quote", Value: "USDT"}),
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			out := initial
			assert.ErrorIs(t, out.UnmarshalBSONValue(tc.typ, tc.data), ErrCreatingPair, "UnmarshalBSONValue should reject the stored value")
			assert.Equal(t, initial, out, "a rejected value should leave the pair unchanged")
		})
	}

	for name, data := range map[string][]byte{
		"truncated length":  {5, 0},
		"length beyond end": {0xFF, 0, 0, 0, 0},
		"truncated element": {8, 0, 0, 0, bsonwire.TypeString, 'b', 0, 0},
		"wrong terminator":  {5, 0, 0, 0, 1},
		"base no NUL":       {16, 0, 0, 0, bsonwire.TypeString, 'b', 0, 4, 0, 0, 0, 'B', 'T', 'C', 'X', 0},
		"delimiter no NUL":  {37, 0, 0, 0, bsonwire.TypeString, 'b', 0, 4, 0, 0, 0, 'B', 'T', 'C', 0, bsonwire.TypeString, 'd', 0, 2, 0, 0, 0, '-', 'X', bsonwire.TypeString, 'q', 0, 5, 0, 0, 0, 'U', 'S', 'D', 'T', 0, 0},
		"quote no NUL":      {28, 0, 0, 0, bsonwire.TypeString, 'b', 0, 4, 0, 0, 0, 'B', 'T', 'C', 0, bsonwire.TypeString, 'q', 0, 5, 0, 0, 0, 'U', 'S', 'D', 'T', 'X', 0},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			out := initial
			assert.ErrorIs(t, out.UnmarshalBSONValue(bsonwire.TypeEmbeddedDocument, data), ErrCreatingPair, "UnmarshalBSONValue should reject malformed bytes")
			assert.Equal(t, initial, out, "a rejected value should leave the pair unchanged")
		})
	}
}
