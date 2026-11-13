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

// v1Compat retains the v1 behaviours the codebase relies on. Removing one either silently changes
// exchange parsing or outbound requests, or fails payloads v1 accepted.
var v1Compat = jsonv2.JoinOptions(
	jsonv2.MatchCaseInsensitiveNames(true),       // v2 matches field names exactly; see below
	jsontext.AllowDuplicateNames(true),           // required alongside the above: XRPScan sends "sequence" and "Sequence" in one object
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
	jsonv1.ParseTimeWithLooseRFC3339(true),       // a rejected timestamp fails the whole payload, not the field
)

// The three name-matching options are interdependent. Matching case-insensitively, as v1 did, keeps
// a tag that differs from the wire only in case from decoding to zero without erroring. But it makes
// "sequence" and "Sequence" the same member where only one is tagged, which trips v2's duplicate name
// check on XRPScan's account payload carrying both, and it makes "_" insignificant, which matches
// availableExchanges against a protobuf available_exchanges.

// Marshal returns the JSON encoding of v. See the "encoding/json/v2" documentation for Marshal
func Marshal(v any) ([]byte, error) { return jsonv2.Marshal(v, v1Compat) }

// Unmarshal parses the JSON-encoded data and stores the result in the value pointed to by v. See the "encoding/json/v2" documentation for Unmarshal
func Unmarshal(data []byte, v any) error { return jsonv2.Unmarshal(data, v, v1Compat) }

// MarshalIndent is like Marshal but applies Indent to format the output. See the "encoding/json/v2" documentation for Marshal
func MarshalIndent(v any, prefix, indent string) ([]byte, error) {
	b, err := Marshal(v)
	if err != nil {
		return nil, err
	}
	return indentBytes(b, prefix, indent)
}

// indentBytes formats already-marshalled JSON with v1's Indent. jsontext's indent options are
// deliberately not used: they reach a jsonv2.MarshalerTo through its encoder's Options, so the
// value emitted would depend on the indentation, and jsontext panics outright on an indent holding
// anything other than spaces and tabs, where v1 and sonic both accept any string
func indentBytes(b []byte, prefix, indent string) ([]byte, error) {
	var buf bytes.Buffer
	if err := jsonv1.Indent(&buf, b, prefix, indent); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// encodeOpts is v1Compat with HTML escaping lifted when the caller has turned it off, applied
// last since v1Compat would otherwise override it
func encodeOpts(rawHTML bool) []jsonv2.Options {
	if rawHTML {
		return []jsonv2.Options{v1Compat, jsontext.EscapeForHTML(false)}
	}
	return []jsonv2.Options{v1Compat}
}

// Valid reports whether data is a valid JSON encoding. See the "encoding/json/jsontext" documentation for Value.IsValid
func Valid(data []byte) bool { return jsontext.Value(data).IsValid(v1Compat) }

// Encoder writes JSON values to an output stream
type Encoder struct {
	w              io.Writer
	err            error
	prefix, indent string
	rawHTML        bool
}

// NewEncoder returns a new encoder that writes to w
func NewEncoder(w io.Writer) *Encoder { return &Encoder{w: w} }

// SetIndent instructs the encoder to format each subsequent encoded value as if indented by
// prefix and indent. SetIndent("", "") disables indentation
func (e *Encoder) SetIndent(prefix, indent string) { e.prefix, e.indent = prefix, indent }

// SetEscapeHTML specifies whether <, > and & should be escaped inside JSON quoted strings.
// U+2028 and U+2029 stay escaped either way, matching v1
func (e *Encoder) SetEscapeHTML(on bool) { e.rawHTML = !on }

// Encode writes the JSON encoding of v to the stream, followed by a newline character.
// The value is encoded in full before anything is written, so a value that fails to marshal
// leaves the stream untouched, and the first write error is latched, both as v1 did. json/v2
// would otherwise stream a failing value's prefix out and then re-encode it on the next call
func (e *Encoder) Encode(v any) error {
	if e.err != nil {
		return e.err
	}
	b, err := jsonv2.Marshal(v, encodeOpts(e.rawHTML)...)
	if err != nil {
		return err
	}
	// read after marshalling, as v1 does, so a MarshalJSON that calls SetIndent affects the value
	// it belongs to. SetIndent("", "") disables indentation
	if e.prefix != "" || e.indent != "" {
		if b, err = indentBytes(b, e.prefix, e.indent); err != nil {
			return err
		}
	}
	if _, err := e.w.Write(append(b, '\n')); err != nil {
		e.err = err
		return err
	}
	return nil
}

// Decoder reads and decodes JSON values from an input stream
type Decoder struct {
	dec           *jsontext.Decoder
	r             *readLatch
	rejectUnknown bool
}

// NewDecoder returns a new decoder that reads from r
func NewDecoder(r io.Reader) *Decoder {
	latch := &readLatch{r: r}
	return &Decoder{dec: jsontext.NewDecoder(latch), r: latch}
}

// DisallowUnknownFields causes the Decoder to error when the destination is a struct and the
// input contains object members matching no non-ignored, exported field
func (d *Decoder) DisallowUnknownFields() { d.rejectUnknown = true }

// More reports whether there is another element in the current array or object being parsed
func (d *Decoder) More() bool {
	switch d.dec.PeekKind() {
	case ']', '}':
		return false
	case 0:
		// PeekKind reports 0 for end of input, for a malformed token and for a failed read alike.
		// v1 reports the latter two as more, so `for d.More()` surfaces the error from the next
		// Decode rather than exiting silently. Reading the token is what tells them apart, and it
		// consumes nothing, there being no token to read
		if d.r.err != nil {
			return true
		}
		_, err := d.dec.ReadToken()
		return err != io.EOF
	default:
		return true
	}
}

// Decode reads the next JSON-encoded value from its input and stores it in the value pointed to by v
func (d *Decoder) Decode(v any) error {
	// The latched failure is the reader's own error rather than what jsontext wraps it in, and it is
	// reported by every later call, both as v1 does it
	if d.r.err != nil {
		return d.r.err
	}
	if d.rejectUnknown {
		return jsonv2.UnmarshalDecode(d.dec, v, v1Compat, jsonv2.RejectUnknownMembers(true))
	}
	return jsonv2.UnmarshalDecode(d.dec, v, v1Compat)
}

// Buffered returns a reader of the data remaining in the Decoder's buffer
func (d *Decoder) Buffered() io.Reader { return bytes.NewReader(d.dec.UnreadBuffer()) }
