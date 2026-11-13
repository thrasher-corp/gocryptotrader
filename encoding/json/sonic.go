//go:build sonic_on

package json

import (
	"bytes"
	jsonv1 "encoding/json" //nolint:depguard // Acceptable use in gct json wrapper
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

// Decoder reads and decodes JSON values from an input stream
type Decoder struct {
	sonic.Decoder
	r             *readLatch
	bufferDropped bool
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
// v, returning a read failure More reported where the encoding/json fallback would report io.EOF
func (d *Decoder) Decode(v any) error {
	err := d.Decoder.Decode(v)
	if err == nil {
		return nil
	}
	if _, fallback := d.Decoder.(*jsonv1.Decoder); !fallback {
		d.bufferDropped = true
	}
	if d.r.err != nil && errors.Is(err, io.EOF) {
		return d.r.err
	}
	return err
}

// Buffered returns a reader of the data remaining in the Decoder's buffer. Native sonic drops its
// buffer when a Decode fails, after which its own Buffered can panic
func (d *Decoder) Buffered() io.Reader {
	if d.bufferDropped {
		return bytes.NewReader(nil)
	}
	return d.Decoder.Buffered()
}
