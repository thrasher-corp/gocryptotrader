//go:build sonic_on

package json

import (
	"errors"
	"io"

	"github.com/bytedance/sonic"
)

// Implementation is a constant string that represents the current JSON implementation package
const Implementation = "bytedance/sonic"

var (
	// Marshal returns the JSON encoding of v. See the "github.com/bytedance/sonic" documentation for Marshal
	Marshal = sonic.ConfigStd.Marshal
	// Unmarshal parses the JSON-encoded data and stores the result in the value pointed to by v. See the "github.com/bytedance/sonic" documentation for Unmarshal
	Unmarshal = sonic.ConfigStd.Unmarshal
	// NewEncoder returns a new encoder that writes to w. See the "github.com/bytedance/sonic" documentation for NewEncoder
	NewEncoder = sonic.ConfigStd.NewEncoder
	// MarshalIndent is like Marshal but applies Indent to format the output. See the "github.com/bytedance/sonic" documentation for MarshalIndent
	MarshalIndent = sonic.ConfigStd.MarshalIndent
	// Valid reports whether data is a valid JSON encoding. See the "github.com/bytedance/sonic" documentation for Valid
	Valid = sonic.ConfigStd.Valid
)

// Decoder reads and decodes JSON values from an input stream. sonic's own decoder is embedded, so
// only More and Decode differ from it
type Decoder struct {
	sonic.Decoder
	r *readLatch
}

// NewDecoder returns a new decoder that reads from r
func NewDecoder(r io.Reader) *Decoder {
	latch := &readLatch{r: r}
	return &Decoder{Decoder: sonic.ConfigStd.NewDecoder(latch), r: latch}
}

// More reports whether there is another element in the current array or object being parsed.
// sonic reports a failed read as end of input, so `for d.More()` would exit without ever reaching
// the Decode that surfaces the error, which is what v1 reports it for
func (d *Decoder) More() bool {
	return d.Decoder.More() || d.r.err != nil
}

// Decode reads the next JSON-encoded value from its input and stores it in the value pointed to by
// v. A read failure More reported is returned here rather than the end of input sonic reports, so
// the value the loop goes on to decode carries the error rather than losing it
func (d *Decoder) Decode(v any) error {
	err := d.Decoder.Decode(v)
	if d.r.err != nil && errors.Is(err, io.EOF) {
		return d.r.err
	}
	return err
}
