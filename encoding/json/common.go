// Package json provides JSON encoding and decoding, using encoding/json/v2 by default and
// github.com/bytedance/sonic when built with -tags sonic_on.
//
// Neither build is a full v1 *encoding/json, and the gaps are not the same on each. Both drop
// Token and InputOffset, which sonic's Decoder interface does not expose. The default build
// additionally drops UseNumber, which json/v2 has no exported equivalent for, while sonic keeps it
// but exposes no Encoder type of its own, so only the default build satisfies var _ *json.Encoder.
//
// v1Compat pins 17 of the 21 options DefaultOptionsV1 sets. ParseTimeWithLooseRFC3339 is among
// them because a timestamp v1 accepted and v2 rejects fails the whole payload rather than the one
// field, and a bare time.Time carries a json tag on over two hundred fields here. The four left
// out are latent: [N]byte and named byte slices swap between base64 and arrays
// (FormatByteArrayAsArray, FormatBytesWithLegacySemantics), base64 carrying newlines no longer
// parses (ParseBytesWithLooseRFC4648), and MarshalJSON is called for map keys and on unaddressable
// values where v1 skipped it (CallMethodsWithLegacySemantics); currency.Code relies on the map key
// half, so it carries text methods for the sonic build, which does not.
//
// Behaviour differs by build:
//
//   - sonic_on selects encoding/json (v1) outside amd64 and arm64, or on Go 1.28 or later. It does
//     not compile for 32-bit platforms, nor natively for illumos, Solaris or Plan 9.
//     Implementation remains "bytedance/sonic".
//   - Only the default build can resume decoding input supplied after a Decode fails because the
//     input ended, or after More returns false at its end.
//   - Native sonic ties U+2028/U+2029 escaping to SetEscapeHTML, where v1 always escapes them, and
//     does not latch write errors. Without indentation it writes newlines separately, so Encode can
//     fail after writing a value, and the next value can run into it.
//   - Native sonic's decoder can split, truncate or reject top level numbers at read boundaries,
//     including its own buffer's, and can discard valid values after a top level number. It can
//     read past unrecognised or unterminated input, and the values after it, to EOF or a read
//     error. A stray "]" or "}" succeeds on every call without consuming input. A truncated "{"
//     returns io.EOF, where the default build returns a *SyntaxError and v1 io.ErrUnexpectedEOF.
//     Once a Decode fails, every later Decode returns the same error, where v1 and the default
//     build decode the next value after one that fails to unmarshal, such as a type mismatch.
//   - The sonic wrapper makes More report read failures, Decode return the failure the fallback
//     hides behind io.EOF, and Buffered return an empty reader after a native Decode error.
package json

import (
	jsonv1 "encoding/json"   //nolint:depguard // Acceptable use in gct json wrapper
	"encoding/json/jsontext" //nolint:depguard // Acceptable use in gct json wrapper
	"io"
)

// readLatch remembers the first read failure so that every later call reports it, as v1 latches it,
// rather than carrying on with whatever the reader served next. A failure served alongside data is
// not latched: v1 reports the value that data completed and only fails once a read it needed does
type readLatch struct {
	r   io.Reader
	err error
}

func (l *readLatch) Read(p []byte) (int, error) {
	n, err := l.r.Read(p)
	if n == 0 && err != nil && err != io.EOF && l.err == nil {
		l.err = err
	}
	return n, err
}

type (
	// RawMessage is a raw encoded JSON value.
	// It implements [Marshaler] and [Unmarshaler] and can
	// be used to delay JSON decoding or precompute a JSON encoding.
	RawMessage = jsontext.Value
	// An UnmarshalTypeError describes a JSON value that was not appropriate for a value of a specific Go type.
	// The default build surfaces it through ReportErrorsWithLegacySemantics. Sonic's standard-library
	// fallback also returns it, while native Sonic returns its own MismatchTypeError, which does not
	// match this alias. Named after the type actually aliased, since json/v2 has its own distinct
	// SemanticError and SyntacticError types
	UnmarshalTypeError = jsonv1.UnmarshalTypeError
	// A SyntaxError describes improper JSON. The default build and Sonic's standard-library fallback
	// can return this type; native Sonic returns its own SyntaxError, which does not match this alias.
	// The fallback's Unmarshal returns io.ErrUnexpectedEOF for truncated input and a SyntaxError with
	// an empty message for trailing data
	SyntaxError = jsonv1.SyntaxError
)
