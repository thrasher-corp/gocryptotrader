package bsonwire

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshalString(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		in   string
		want []byte
	}{
		"empty":        {"", []byte{0x01, 0x00, 0x00, 0x00, 0x00}},
		"text":         {"BTC", []byte{0x04, 0x00, 0x00, 0x00, 'B', 'T', 'C', 0x00}},
		"embedded NUL": {"a\x00", []byte{0x03, 0x00, 0x00, 0x00, 'a', 0x00, 0x00}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, MarshalString(tc.in), "string should use the BSON string encoding")
		})
	}
}

func TestUnmarshalString(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		typ     byte
		data    []byte
		want    string
		wantErr error
	}{
		"empty": {
			typ:  TypeString,
			data: []byte{0x01, 0x00, 0x00, 0x00, 0x00},
		},
		"text": {
			typ:  TypeString,
			data: []byte{0x04, 0x00, 0x00, 0x00, 'B', 'T', 'C', 0x00},
			want: "BTC",
		},
		"embedded NUL": {
			typ:  TypeString,
			data: []byte{0x03, 0x00, 0x00, 0x00, 'a', 0x00, 0x00},
			want: "a\x00",
		},
		"wrong type": {
			typ:     TypeEmbeddedDocument,
			data:    []byte{0x04, 0x00, 0x00, 0x00, 'B', 'T', 'C', 0x00},
			wantErr: errWrongType,
		},
		"no bytes": {
			typ:     TypeString,
			wantErr: errLength,
		},
		"truncated length": {
			typ:     TypeString,
			data:    []byte{0x05, 0x00},
			wantErr: errLength,
		},
		"zero length": {
			typ:     TypeString,
			data:    []byte{0x00, 0x00, 0x00, 0x00, 0x00},
			wantErr: errLength,
		},
		"negative length": {
			typ:     TypeString,
			data:    []byte{0xFF, 0xFF, 0xFF, 0xFF},
			wantErr: errLength,
		},
		"length beyond data": {
			typ:     TypeString,
			data:    []byte{0xFF, 0x00, 0x00, 0x00, 'G', 0x00},
			wantErr: errLength,
		},
		"length short of data": {
			typ:     TypeString,
			data:    []byte{0x04, 0x00, 0x00, 0x00, 'B', 'T', 'C', 0x00, 0x00},
			wantErr: errLength,
		},
		"missing NUL": {
			typ:     TypeString,
			data:    []byte{0x04, 0x00, 0x00, 0x00, 'B', 'T', 'C', 'X'},
			wantErr: errMissingNUL,
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, err := UnmarshalString(tc.typ, tc.data)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr, "UnmarshalString should return the expected error")
				return
			}
			require.NoError(t, err, "UnmarshalString must not error")
			assert.Equal(t, tc.want, got, "string should decode")
		})
	}
}

func TestMarshalDocument(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		fields []Field
		want   []byte
	}{
		"empty": {
			nil,
			[]byte{0x05, 0x00, 0x00, 0x00, 0x00},
		},
		"one field": {
			[]Field{{Key: "b", Value: "BTC"}},
			[]byte{0x10, 0x00, 0x00, 0x00, 0x02, 'b', 0x00, 0x04, 0x00, 0x00, 0x00, 'B', 'T', 'C', 0x00, 0x00},
		},
		"three fields": {
			[]Field{{Key: "b", Value: "BTC"}, {Key: "d", Value: "-"}, {Key: "q", Value: "USDT"}},
			[]byte{0x25, 0x00, 0x00, 0x00, 0x02, 'b', 0x00, 0x04, 0x00, 0x00, 0x00, 'B', 'T', 'C', 0x00, 0x02, 'd', 0x00, 0x02, 0x00, 0x00, 0x00, '-', 0x00, 0x02, 'q', 0x00, 0x05, 0x00, 0x00, 0x00, 'U', 'S', 'D', 'T', 0x00, 0x00},
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, MarshalDocument(tc.fields...), "fields should use the BSON document encoding")
		})
	}
}

func TestUnmarshalDocument(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		typ     byte
		data    []byte
		want    []Field
		wantErr error
	}{
		"empty": {
			typ:  TypeEmbeddedDocument,
			data: []byte{0x05, 0x00, 0x00, 0x00, 0x00},
			want: []Field{},
		},
		"one field": {
			typ:  TypeEmbeddedDocument,
			data: []byte{0x10, 0x00, 0x00, 0x00, 0x02, 'b', 0x00, 0x04, 0x00, 0x00, 0x00, 'B', 'T', 'C', 0x00, 0x00},
			want: []Field{{Key: "b", Value: "BTC"}},
		},
		"three fields": {
			typ:  TypeEmbeddedDocument,
			data: []byte{0x25, 0x00, 0x00, 0x00, 0x02, 'b', 0x00, 0x04, 0x00, 0x00, 0x00, 'B', 'T', 'C', 0x00, 0x02, 'd', 0x00, 0x02, 0x00, 0x00, 0x00, '-', 0x00, 0x02, 'q', 0x00, 0x05, 0x00, 0x00, 0x00, 'U', 'S', 'D', 'T', 0x00, 0x00},
			want: []Field{{Key: "b", Value: "BTC"}, {Key: "d", Value: "-"}, {Key: "q", Value: "USDT"}},
		},
		"wrong type": {
			typ:     TypeString,
			data:    []byte{0x05, 0x00, 0x00, 0x00, 0x00},
			wantErr: errWrongType,
		},
		"no bytes": {
			typ:     TypeEmbeddedDocument,
			wantErr: errLength,
		},
		"truncated length": {
			typ:     TypeEmbeddedDocument,
			data:    []byte{0x05, 0x00},
			wantErr: errLength,
		},
		"length below minimum": {
			typ:     TypeEmbeddedDocument,
			data:    []byte{0x04, 0x00, 0x00, 0x00},
			wantErr: errLength,
		},
		"length beyond data": {
			typ:     TypeEmbeddedDocument,
			data:    []byte{0x06, 0x00, 0x00, 0x00, 0x00},
			wantErr: errLength,
		},
		"length short of data": {
			typ:     TypeEmbeddedDocument,
			data:    []byte{0x05, 0x00, 0x00, 0x00, 0x00, 0x00},
			wantErr: errLength,
		},
		"missing terminator": {
			typ:     TypeEmbeddedDocument,
			data:    []byte{0x05, 0x00, 0x00, 0x00, 0x01},
			wantErr: errMissingNUL,
		},
		"truncated element": {
			typ:     TypeEmbeddedDocument,
			data:    []byte{0x08, 0x00, 0x00, 0x00, 0x02, 'b', 0x00, 0x00},
			wantErr: errLength,
		},
		"key not terminated": {
			typ:     TypeEmbeddedDocument,
			data:    []byte{0x08, 0x00, 0x00, 0x00, 0x02, 'b', 'c', 0x00},
			wantErr: errUnterminatedKey,
		},
		"non-string field": {
			typ:     TypeEmbeddedDocument,
			data:    []byte{0x0C, 0x00, 0x00, 0x00, 0x10, 'b', 0x00, 0x01, 0x00, 0x00, 0x00, 0x00},
			wantErr: errNonStringField,
		},
		"zero string length": {
			typ:     TypeEmbeddedDocument,
			data:    []byte{0x0C, 0x00, 0x00, 0x00, 0x02, 'b', 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			wantErr: errLength,
		},
		"string beyond document": {
			typ:     TypeEmbeddedDocument,
			data:    []byte{0x0C, 0x00, 0x00, 0x00, 0x02, 'b', 0x00, 0x0A, 0x00, 0x00, 0x00, 0x00},
			wantErr: errLength,
		},
		"string into terminator": {
			typ:     TypeEmbeddedDocument,
			data:    []byte{0x10, 0x00, 0x00, 0x00, 0x02, 'b', 0x00, 0x05, 0x00, 0x00, 0x00, 'B', 'T', 'C', 0x00, 0x00},
			wantErr: errLength,
		},
		"stray byte after last field": {
			typ:     TypeEmbeddedDocument,
			data:    []byte{0x11, 0x00, 0x00, 0x00, 0x02, 'b', 0x00, 0x04, 0x00, 0x00, 0x00, 'B', 'T', 'C', 0x00, 0x02, 0x00},
			wantErr: errUnterminatedKey,
		},
		"empty key": {
			typ:  TypeEmbeddedDocument,
			data: []byte{0x0F, 0x00, 0x00, 0x00, 0x02, 0x00, 0x04, 0x00, 0x00, 0x00, 'B', 'T', 'C', 0x00, 0x00},
			want: []Field{{Key: "", Value: "BTC"}},
		},
		"negative string length": {
			typ:     TypeEmbeddedDocument,
			data:    []byte{0x0C, 0x00, 0x00, 0x00, 0x02, 'b', 0x00, 0xFF, 0xFF, 0xFF, 0xFF, 0x00},
			wantErr: errLength,
		},
		"string length cut short": {
			typ:     TypeEmbeddedDocument,
			data:    []byte{0x0A, 0x00, 0x00, 0x00, 0x02, 'b', 0x00, 0x00, 0x00, 0x00},
			wantErr: errLength,
		},
		"string missing NUL": {
			typ:     TypeEmbeddedDocument,
			data:    []byte{0x10, 0x00, 0x00, 0x00, 0x02, 'b', 0x00, 0x04, 0x00, 0x00, 0x00, 'B', 'T', 'C', 'X', 0x00},
			wantErr: errMissingNUL,
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, err := UnmarshalDocument(tc.typ, tc.data)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr, "UnmarshalDocument should return the expected error")
				return
			}
			require.NoError(t, err, "UnmarshalDocument must not error")
			assert.Equal(t, tc.want, got, "document should decode")
		})
	}
}
