// Package bsonwire encodes and decodes the BSON values used by types that implement the BSON value interfaces: a
// string, and a flat document whose fields are all strings.
package bsonwire

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

// BSON value types used by this package.
const (
	TypeString           byte = 0x02
	TypeEmbeddedDocument byte = 0x03
	TypeArray            byte = 0x04
	TypeNull             byte = 0x0A
	TypeSymbol           byte = 0x0E
	TypeInt32            byte = 0x10
)

var (
	errWrongType       = errors.New("wrong BSON type")
	errLength          = errors.New("invalid BSON length")
	errMissingNUL      = errors.New("missing BSON NUL terminator")
	errUnterminatedKey = errors.New("unterminated BSON document key")
	errNonStringField  = errors.New("non-string BSON document field")
)

// ValueMarshaler is the interface implemented by types that can marshal themselves as a BSON value.
type ValueMarshaler interface {
	MarshalBSONValue() (byte, []byte, error)
}

// ValueUnmarshaler is the interface implemented by types that can unmarshal a BSON value.
type ValueUnmarshaler interface {
	UnmarshalBSONValue(byte, []byte) error
}

// Field is a string field in a BSON document.
type Field struct {
	Key   string
	Value string
}

// MarshalString returns the BSON string value encoding of s.
func MarshalString(s string) []byte {
	data := make([]byte, 4, len(s)+5)
	binary.LittleEndian.PutUint32(data, uint32(len(s)+1)) //nolint:gosec // BSON string lengths fit in the int32 length field
	data = append(data, s...)
	return append(data, 0)
}

// UnmarshalString reads a complete BSON string value. It rejects a type other than string, a length that does not
// match the data, and a value that does not end with a NUL byte.
func UnmarshalString(typ byte, data []byte) (string, error) {
	if typ != TypeString {
		return "", fmt.Errorf("%w: expected a BSON string, got type %#02x", errWrongType, typ)
	}
	if len(data) < 5 {
		return "", fmt.Errorf("%w: BSON string is %d bytes, need at least 5", errLength, len(data))
	}
	length := binary.LittleEndian.Uint32(data[:4])
	if int64(length) != int64(len(data)-4) {
		return "", fmt.Errorf("%w: BSON string length is %d, but %d bytes follow", errLength, length, len(data)-4)
	}
	if data[len(data)-1] != 0 {
		return "", fmt.Errorf("%w: BSON string does not end with a NUL byte", errMissingNUL)
	}
	return string(data[4 : len(data)-1]), nil
}

// MarshalDocument returns the BSON document encoding of fields, in order.
func MarshalDocument(fields ...Field) []byte {
	data := make([]byte, 4, 5)
	for _, f := range fields {
		data = append(data, TypeString)
		data = append(data, f.Key...)
		data = append(data, 0)
		data = append(data, MarshalString(f.Value)...)
	}
	data = append(data, 0)
	binary.LittleEndian.PutUint32(data, uint32(len(data))) //nolint:gosec // BSON document lengths fit in the int32 length field
	return data
}

// UnmarshalDocument reads a complete BSON document whose fields are all strings. It rejects a type other than an
// embedded document, a length that does not match the data, a document that does not end with a NUL byte, a field
// whose key or value is malformed, and a field that is not a string.
func UnmarshalDocument(typ byte, data []byte) ([]Field, error) {
	if typ != TypeEmbeddedDocument {
		return nil, fmt.Errorf("%w: expected a BSON document, got type %#02x", errWrongType, typ)
	}
	if len(data) < 5 {
		return nil, fmt.Errorf("%w: BSON document is %d bytes, need at least 5", errLength, len(data))
	}
	length := binary.LittleEndian.Uint32(data[:4])
	if int64(length) != int64(len(data)) {
		return nil, fmt.Errorf("%w: BSON document length is %d, but it is %d bytes", errLength, length, len(data))
	}
	if data[len(data)-1] != 0 {
		return nil, fmt.Errorf("%w: BSON document does not end with a NUL byte", errMissingNUL)
	}

	fields := make([]Field, 0)
	for i := 4; i < len(data)-1; {
		fieldType := data[i]
		i++

		keyEnd := bytes.IndexByte(data[i:len(data)-1], 0)
		if keyEnd < 0 {
			return nil, fmt.Errorf("%w: BSON document field key is not terminated", errUnterminatedKey)
		}
		key := string(data[i : i+keyEnd])
		i += keyEnd + 1
		if fieldType != TypeString {
			return nil, fmt.Errorf("%w: BSON document field %q has type %#02x, want a string", errNonStringField, key, fieldType)
		}
		if i+4 > len(data)-1 {
			return nil, fmt.Errorf("%w: BSON document field %q has no string length", errLength, key)
		}
		valueLength := binary.LittleEndian.Uint32(data[i : i+4])
		if int64(valueLength) > int64(len(data)-1-i-4) {
			return nil, fmt.Errorf("%w: BSON document field %q string length is %d, beyond the document", errLength, key, valueLength)
		}
		valueEnd := i + 4 + int(valueLength)
		value, err := UnmarshalString(TypeString, data[i:valueEnd])
		if err != nil {
			return nil, fmt.Errorf("BSON document field %q: %w", key, err)
		}
		fields = append(fields, Field{Key: key, Value: value})
		i = valueEnd
	}
	return fields, nil
}
