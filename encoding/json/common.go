package json

import "encoding/json" //nolint:depguard // This is a wrapper package

type (
	// RawMessage is a raw encoded JSON value.
	// It implements [Marshaler] and [Unmarshaler] and can
	// be used to delay JSON decoding or precompute a JSON encoding.
	RawMessage = json.RawMessage
	// An UnmarshalTypeError describes a JSON value that was
	// not appropriate for a value of a specific Go type.
	UnmarshalTypeError = json.UnmarshalTypeError
)
