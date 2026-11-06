package currency

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

var (
	_ bson.ValueMarshaler   = Pair{}
	_ bson.ValueUnmarshaler = (*Pair)(nil)
)

// bsonPairDocument holds a pair as a document field, since BSON stores values within documents
type bsonPairDocument struct {
	Pair Pair `bson:"pair"`
}

// bsonPairPointerDocument holds a pair through a pointer, which the driver sets to nil for BSON null without calling
// UnmarshalBSONValue
type bsonPairPointerDocument struct {
	Pair *Pair `bson:"pair"`
}

func TestPairBSONValue(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		pair   Pair
		stored bson.D
	}{
		"empty":                {EMPTYPAIR, bson.D{}},
		"no delimiter":         {NewBTCUSDT(), bson.D{{Key: "b", Value: "BTC"}, {Key: "q", Value: "USDT"}}},
		"delimiter":            {NewPairWithDelimiter("BTC", "USDT", "-"), bson.D{{Key: "b", Value: "BTC"}, {Key: "d", Value: "-"}, {Key: "q", Value: "USDT"}}},
		"lower case":           {NewPairWithDelimiter("eth", "usdc", "/"), bson.D{{Key: "b", Value: "eth"}, {Key: "d", Value: "/"}, {Key: "q", Value: "usdc"}}},
		"long delimiter":       {NewPairWithDelimiter("BTC", "USDT", "-_-"), bson.D{{Key: "b", Value: "BTC"}, {Key: "d", Value: "-_-"}, {Key: "q", Value: "USDT"}}},
		"control in delimiter": {NewPairWithDelimiter("BTC", "USDT", "\x1F"), bson.D{{Key: "b", Value: "BTC"}, {Key: "d", Value: "\x1F"}, {Key: "q", Value: "USDT"}}},
		"base only":            {NewPair(BTC, EMPTYCODE), bson.D{{Key: "b", Value: "BTC"}}},
		"quote only":           {NewPairWithDelimiter("", "USDT", "-"), bson.D{{Key: "d", Value: "-"}, {Key: "q", Value: "USDT"}}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			data, err := bson.Marshal(bsonPairDocument{Pair: tc.pair})
			require.NoError(t, err, "Marshal must not error")
			want, err := bson.Marshal(tc.stored)
			require.NoError(t, err, "Marshal of the expected document must not error")
			stored, ok := bson.Raw(data).Lookup("pair").DocumentOK()
			require.True(t, ok, "pair must be stored as a BSON document")
			assert.Equal(t, bson.Raw(want), stored, "pair should be stored as its non-empty fields")

			out := bsonPairDocument{Pair: NewPair(LTC, XRP)}
			require.NoError(t, bson.Unmarshal(data, &out), "Unmarshal must not error")
			assert.Equal(t, tc.pair, out.Pair, "pair should keep its case and delimiter")
			assert.True(t, tc.pair.Equal(out.Pair), "pair should read back as the same currency codes")

			data, err = bson.Marshal(bsonPairPointerDocument{Pair: &tc.pair})
			require.NoError(t, err, "Marshal of a pointer field must not error")
			var ptrOut bsonPairPointerDocument
			require.NoError(t, bson.Unmarshal(data, &ptrOut), "Unmarshal of a pointer field must not error")
			require.NotNil(t, ptrOut.Pair, "pointer field must be set")
			assert.Equal(t, tc.pair, *ptrOut.Pair, "pair should round-trip through a pointer field")
		})
	}
}

func TestPairBSONValueNilPointer(t *testing.T) {
	t.Parallel()
	data, err := bson.Marshal(bsonPairPointerDocument{})
	require.NoError(t, err, "Marshal must not error")
	assert.Equal(t, bson.TypeNull, bson.Raw(data).Lookup("pair").Type, "a nil pair should be stored as null")
	out := bsonPairPointerDocument{Pair: &Pair{Base: BTC, Quote: USDT}}
	require.NoError(t, bson.Unmarshal(data, &out), "Unmarshal must not error")
	assert.Nil(t, out.Pair, "null should read back as a nil pair")
}

func TestPairUnmarshalBSONValueErrors(t *testing.T) {
	t.Parallel()
	for name, value := range map[string]any{
		"string":           "BTC-USDT",
		"integer":          int32(1),
		"null":             nil,
		"array":            bson.A{"BTC", "-", "USDT"},
		"non-string field": bson.D{{Key: "b", Value: int32(1)}},
		"null field":       bson.D{{Key: "q", Value: nil}},
		"unexpected field": bson.D{{Key: "b", Value: "BTC"}, {Key: "base", Value: "BTC"}},
		"long field names": bson.D{{Key: "base", Value: "BTC"}, {Key: "quote", Value: "USDT"}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			data, err := bson.Marshal(bson.D{{Key: "pair", Value: value}})
			require.NoError(t, err, "Marshal must not error")
			assert.ErrorIs(t, bson.Unmarshal(data, &bsonPairDocument{}), ErrCreatingPair, "Unmarshal should reject the stored value")
		})
	}
}

// TestPairUnmarshalBSONValueMalformed calls UnmarshalBSONValue directly, since the driver rejects a malformed document
// before calling it
func TestPairUnmarshalBSONValueMalformed(t *testing.T) {
	t.Parallel()
	for name, data := range map[string][]byte{
		"truncated length":  {5, 0},
		"length beyond end": {0xFF, 0, 0, 0, 0},
		"truncated element": {8, 0, 0, 0, byte(bson.TypeString), 'b', 0, 0},
		"wrong terminator":  {5, 0, 0, 0, 1},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			p := NewBTCUSDT()
			assert.ErrorIs(t, p.UnmarshalBSONValue(byte(bson.TypeEmbeddedDocument), data), ErrCreatingPair, "UnmarshalBSONValue should reject malformed bytes")
			assert.Equal(t, NewBTCUSDT(), p, "a rejected value should leave the pair unchanged")
		})
	}
}
