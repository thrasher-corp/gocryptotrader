//go:build !sonic_on

package json

import (
	"bytes"
	jsonv1 "encoding/json"    //nolint:depguard // Acceptable use in gct json wrapper
	"encoding/json/jsontext"  //nolint:depguard // Acceptable use in gct json wrapper
	jsonv2 "encoding/json/v2" //nolint:depguard // Acceptable use in gct json wrapper
	"io"
)

// Implementation is a constant string that represents the current JSON implementation package
const Implementation = "encoding/json/v2"

// v1Compat retains the v1 behaviours the codebase relies on. Removing any one of these
// silently changes exchange parsing or outbound requests rather than erroring.
var v1Compat = jsonv2.JoinOptions(
	jsonv2.MatchCaseInsensitiveNames(true),       // v2 matches field names exactly; see below
	jsontext.AllowDuplicateNames(true),           // required alongside the above: Binance sends "e" and "E" in one object
	jsonv1.MatchCaseSensitiveDelimiter(true),     // and this: without it "_" is insignificant, so available_exchanges matches availableExchanges
	jsonv1.FormatDurationAsNano(true),            // v2 has no default time.Duration representation
	jsonv1.UnmarshalArrayFromAnyLength(true),     // payload length need not match a fixed-size array
	jsonv1.StringifyWithLegacySemantics(true),    // v2 limits the `,string` tag to numeric types
	jsonv1.MergeWithLegacySemantics(true),        // &[N]any{...} positional decoding
	jsonv1.OmitEmptyWithLegacySemantics(true),    // v2 reads omitempty as empty JSON, not zero Go value
	jsonv2.Deterministic(true),                   // v1 sorted map keys; mocks and signed bodies rely on it
	jsonv1.ReportErrorsWithLegacySemantics(true), // v2 returns partial output alongside an error
)

// The three name-matching options are interdependent: matching case-insensitively makes "e" and "E"
// the same member, which trips v2's duplicate name check on Binance payloads carrying both, and
// makes "_" insignificant, which matches availableExchanges against a protobuf available_exchanges.
// Dropping any of them makes a tag that mismatches the wire decode to zero without erroring.

// Marshal returns the JSON encoding of v. See the "encoding/json/v2" documentation for Marshal
func Marshal(v any) ([]byte, error) { return jsonv2.Marshal(v, v1Compat) }

// Unmarshal parses the JSON-encoded data and stores the result in the value pointed to by v. See the "encoding/json/v2" documentation for Unmarshal
func Unmarshal(data []byte, v any) error { return jsonv2.Unmarshal(data, v, v1Compat) }

// MarshalIndent is like Marshal but applies Indent to format the output. See the "encoding/json/v2" documentation for Marshal
func MarshalIndent(v any, prefix, indent string) ([]byte, error) {
	return jsonv2.Marshal(v, v1Compat, jsontext.WithIndentPrefix(prefix), jsontext.WithIndent(indent))
}

// Valid reports whether data is a valid JSON encoding. See the "encoding/json/jsontext" documentation for Value.IsValid
func Valid(data []byte) bool { return jsontext.Value(data).IsValid(v1Compat) }

// Encoder writes JSON values to an output stream
type Encoder struct{ enc *jsontext.Encoder }

// NewEncoder returns a new encoder that writes to w
func NewEncoder(w io.Writer) *Encoder { return &Encoder{enc: jsontext.NewEncoder(w)} }

// Encode writes the JSON encoding of v to the stream, followed by a newline character
func (e *Encoder) Encode(v any) error { return jsonv2.MarshalEncode(e.enc, v, v1Compat) }

// Decoder reads and decodes JSON values from an input stream
type Decoder struct{ dec *jsontext.Decoder }

// NewDecoder returns a new decoder that reads from r
func NewDecoder(r io.Reader) *Decoder { return &Decoder{dec: jsontext.NewDecoder(r)} }

// Decode reads the next JSON-encoded value from its input and stores it in the value pointed to by v
func (d *Decoder) Decode(v any) error { return jsonv2.UnmarshalDecode(d.dec, v, v1Compat) }

// Buffered returns a reader of the data remaining in the Decoder's buffer
func (d *Decoder) Buffered() io.Reader { return bytes.NewReader(d.dec.UnreadBuffer()) }
