// Package json is an abstraction middleware package to allow switching between json encoder/decoder implementations
// The default implementation is golang.org/encoding/json/v2.
// Build with `sonic_on` tag to switch to using github.com/bytedance/sonic
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
// Behaviour differs by build in both directions. The sonic notes below describe its native
// backend; on the 64-bit platforms it does not cover it falls back to encoding/json, which restores
// v1 encoding and can return v1 error types, while the sonic_on build does not compile for 32-bit
// platforms, illumos, Solaris or Plan 9. Implementation remains "bytedance/sonic" where it builds.
// The default build is the outlier on a bytes.Buffer that grows between Decode calls, or a reader
// serving more after io.EOF: once a Decode has failed because the input ended before or inside a
// value, the next one can read on into later input, where v1 and sonic return their error again.
// v1, the default build and sonic each report something different for a truncated "{": v1
// reports io.ErrUnexpectedEOF, the default build a *SyntaxError and sonic io.EOF. sonic is the
// outlier elsewhere: it ties U+2028 and U+2029 escaping to SetEscapeHTML where v1 escaped them
// either way, re-attempts a failed write rather than latching it, and with indentation disabled
// writes each value's trailing newline separately, so an Encode can fail after writing its value,
// and the next value can run on from it. Its Decoder parts from v1 where a read ends, in two ways.
// It ends a top level number there, where v1 reads on to see whether more of it follows, so "1"
// then "2" decodes as two values and "1.5" then "e3" as 1.5, while "1e" then "2" gets sonic's own
// syntax error, "1e" not being a number by itself, even where the number straddles the end of
// sonic's own buffer, 4096 bytes unless its pool hands over a larger one, from a reader holding the
// whole input. And it reads on past bytes that begin no value, like that "e3" or "xyz", and through
// a value whose end it cannot find, rather than reporting them, so they and any value after them
// surface as the read's failure, or as io.EOF where the stream ends, which is also why it reports
// io.EOF for a truncated "{". A malformed value whose end it does find, such as "[1,]", gets
// sonic's own syntax error instead, unless it is a top level number followed directly by a byte
// that cannot continue it, as in "1.2.3" or "01": that decodes as the number, as in v1, but sonic
// drops the bytes after it up to the next whitespace, comma, "]" or "}", where v1 reports them or
// decodes them as the next value. A stray "]" or "}" decodes as nothing, without error, on every
// call. The Decoder is wrapped so that More reports a failed read rather than the end of the
// stream, sonic reporting the two alike, and Decode returns that failure where the encoding/json
// fallback reports io.EOF in its place; every other method is its own
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
