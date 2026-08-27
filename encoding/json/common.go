// Package json is an abstraction middleware package to allow switching between json encoder/decoder implementations
// The default implementation is golang.org/encoding/json/v2.
// Build with `sonic_on` tag to switch to using github.com/bytedance/sonic
//
// Encoder and Decoder expose the intersection of what both implementations provide. Relative to a
// v1 *encoding/json.Decoder this drops UseNumber, which json/v2 has no exported equivalent for,
// and Token and InputOffset, which sonic never provided.
//
// Three behaviours still differ by build, all on sonic's side: it ties U+2028 and U+2029 escaping
// to SetEscapeHTML where v1 escaped them either way; it re-attempts a failed write rather than
// latching it; and it writes each value's trailing newline separately and discards that write's
// error, so two values can run together in the stream.
package json

import (
	jsonv1 "encoding/json"   //nolint:depguard // Acceptable use in gct json wrapper
	"encoding/json/jsontext" //nolint:depguard // Acceptable use in gct json wrapper
)

type (
	// RawMessage is a raw encoded JSON value.
	// It implements [Marshaler] and [Unmarshaler] and can
	// be used to delay JSON decoding or precompute a JSON encoding.
	RawMessage = jsontext.Value
	// An UnmarshalTypeError describes a JSON value that was not appropriate for a value of a specific Go type.
	// Both implementations surface the v1 type, json/v2 via ReportErrorsWithLegacySemantics. Named after
	// the type actually aliased, since json/v2 has its own distinct SemanticError and SyntacticError types.
	UnmarshalTypeError = jsonv1.UnmarshalTypeError
	// A SyntaxError describes improper JSON
	SyntaxError = jsonv1.SyntaxError
)
