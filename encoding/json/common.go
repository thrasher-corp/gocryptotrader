// json is an abstraction middleware package to allow switching between json encoder/decoder implementations
// The default implementation is sonic.
// Build with `sonic_off`, '386' or 'arm64' tags to switch to golang.org/encoding/json.
package json

import "encoding/json" //nolint:depguard // Acceptable use in gct json wrapper

type (
	// RawMessage is a raw encoded JSON value.
	// It implements [Marshaler] and [Unmarshaler] and can
	// be used to delay JSON decoding or precompute a JSON encoding.
	RawMessage = json.RawMessage
	// An UnmarshalTypeError describes a JSON value that was
	// not appropriate for a value of a specific Go type.
	UnmarshalTypeError = json.UnmarshalTypeError
)
