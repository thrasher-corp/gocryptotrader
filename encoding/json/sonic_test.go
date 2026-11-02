//go:build sonic_on

// Tests for the Decoder this build wraps around the encoding/json fallback, which CI does not run;
// json_test.go is untagged and compiles against the default build too, so it cannot construct one

package json

import (
	jsonv1 "encoding/json" //nolint:depguard // the fallback's Decoder is the one encoding/json returns
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDecoderDecode covers the Decode override over the encoding/json fallback, with a reader that
// reports the end of its input and then fails. Where native sonic does not run, sonic's NewDecoder
// returns encoding/json's Decoder, built here as it builds it. That Decoder latches the end of
// input, so once More has reported the failure that followed, its own Decode reports the end of
// input again, and the wrapper's must return the failure in its place on every later call, whatever
// the reader serves next
func TestDecoderDecode(t *testing.T) {
	t.Parallel()
	for _, failure := range []error{errStreamFailed, io.ErrUnexpectedEOF} {
		for name, next := range map[string][]readStep{
			"the end after it": nil,
			"a value after it": {{data: "1 "}},
		} {
			t.Run(fmt.Sprintf("fallback/%v/%s", failure, name), func(t *testing.T) {
				t.Parallel()
				steps := append([]readStep{{err: io.EOF}, {err: failure}}, next...)
				latch := &readLatch{r: &scriptedReader{steps: steps}}
				dec := &Decoder{Decoder: jsonv1.NewDecoder(latch), r: latch}
				var v any
				require.ErrorIs(t, dec.Decode(&v), io.EOF, "Decode must report the end of input")
				for range 2 {
					require.True(t, dec.More(), "More must report the failed read")
					err := dec.Decode(&v)
					assert.ErrorIs(t, err, failure, "Decode should return the failure More reported")
					assert.NotErrorIs(t, err, io.EOF, "Decode should not also report the end of input")
				}
			})
		}
	}
}
