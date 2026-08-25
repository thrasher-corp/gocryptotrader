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
	jsonv2.FormatNilSliceAsNull(true),            // v1 marshalled a nil slice as null, not []
	jsonv2.FormatNilMapAsNull(true),              // and a nil map as null, not {}
	jsontext.EscapeForHTML(true),                 // v1 escaped <, > and &
	jsontext.EscapeForJS(true),                   // and U+2028/U+2029
	jsontext.PreserveRawStrings(true),            // a RawMessage keeps the escaping it arrived with
	jsontext.AllowInvalidUTF8(true),              // v1 substituted U+FFFD rather than failing the decode
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
type Encoder struct {
	enc     *jsontext.Encoder
	w       io.Writer
	rawHTML bool
}

// NewEncoder returns a new encoder that writes to w
func NewEncoder(w io.Writer) *Encoder { return &Encoder{enc: jsontext.NewEncoder(w), w: w} }

// SetIndent instructs the encoder to format each subsequent encoded value as if indented by
// prefix and indent. SetIndent("", "") disables indentation.
// json/v2 fixes formatting for an encoder's lifetime, so this rebuilds it
func (e *Encoder) SetIndent(prefix, indent string) {
	if prefix == "" && indent == "" {
		e.enc.Reset(e.w)
		return
	}
	e.enc.Reset(e.w, jsontext.WithIndentPrefix(prefix), jsontext.WithIndent(indent))
}

// SetEscapeHTML specifies whether <, > and & should be escaped inside JSON quoted strings.
// U+2028 and U+2029 stay escaped either way, matching v1
func (e *Encoder) SetEscapeHTML(on bool) { e.rawHTML = !on }

// Encode writes the JSON encoding of v to the stream, followed by a newline character
func (e *Encoder) Encode(v any) error {
	if e.rawHTML {
		// applied here rather than on the encoder, since v1Compat would otherwise override it
		return jsonv2.MarshalEncode(e.enc, v, v1Compat, jsontext.EscapeForHTML(false))
	}
	return jsonv2.MarshalEncode(e.enc, v, v1Compat)
}

// Decoder reads and decodes JSON values from an input stream
type Decoder struct {
	dec           *jsontext.Decoder
	rejectUnknown bool
}

// NewDecoder returns a new decoder that reads from r
func NewDecoder(r io.Reader) *Decoder { return &Decoder{dec: jsontext.NewDecoder(r)} }

// DisallowUnknownFields causes the Decoder to error when the destination is a struct and the
// input contains object members matching no non-ignored, exported field
func (d *Decoder) DisallowUnknownFields() { d.rejectUnknown = true }

// More reports whether there is another element in the current array or object being parsed
func (d *Decoder) More() bool {
	k := d.dec.PeekKind()
	return k != 0 && k != ']' && k != '}'
}

// Decode reads the next JSON-encoded value from its input and stores it in the value pointed to by v
func (d *Decoder) Decode(v any) error {
	if d.rejectUnknown {
		return jsonv2.UnmarshalDecode(d.dec, v, v1Compat, jsonv2.RejectUnknownMembers(true))
	}
	return jsonv2.UnmarshalDecode(d.dec, v, v1Compat)
}

// Buffered returns a reader of the data remaining in the Decoder's buffer
func (d *Decoder) Buffered() io.Reader { return bytes.NewReader(d.dec.UnreadBuffer()) }
