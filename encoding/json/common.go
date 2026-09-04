// Package json is an abstraction middleware package to allow switching between json encoder/decoder implementations
// The default implementation is golang.org/encoding/json/v2.
// Build with `sonic_on` tag to switch to using github.com/bytedance/sonic
//
// Neither build is a full v1 *encoding/json, and the gaps are not the same on each. Both drop
// Token and InputOffset, which sonic's Decoder interface does not expose. The default build
// additionally drops UseNumber, which json/v2 has no exported equivalent for, while sonic keeps it
// but exposes no Encoder or Decoder type of its own, so only the default build satisfies
// var _ *json.Encoder.
//
// v1Compat pins 16 of the 21 options DefaultOptionsV1 sets. The five left out are latent here:
// [N]byte and named byte slices swap between base64 and arrays (FormatByteArrayAsArray,
// FormatBytesWithLegacySemantics), base64 carrying newlines and a single digit hour no longer parse
// (ParseBytesWithLooseRFC4648, and ParseTimeWithLooseRFC3339, which also covers the sub-second
// separator and the timezone), and MarshalJSON is called for map keys
// and on unaddressable values where v1 skipped it (CallMethodsWithLegacySemantics); currency.Code
// relies on the map key half, so it carries text methods for the sonic build, which does not.
//
// Behaviour differs by build in both directions. The sonic notes below describe its native
// backend; on the platforms it does not cover it falls back to encoding/json and behaves as v1.
// The default build is the outlier on a bytes.Buffer that grows between Decode calls, erroring
// where v1 and sonic both read the next value, and all three disagree on a truncated "{": v1
// reports io.ErrUnexpectedEOF, the default build a *SyntaxError and sonic io.EOF. sonic is the
// outlier elsewhere: it ties U+2028 and U+2029 escaping to SetEscapeHTML where v1 escaped them
// either way, re-attempts a failed write rather than latching it, and with indentation disabled
// writes each value's trailing newline separately while discarding that write's error, so two
// values can run together in the stream.
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
