//go:build sonic_on

package json

import (
	jsonv1 "encoding/json" //nolint:depguard // Required to exercise sonic's fallback decoder
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDecoderDecode builds the wrapper over encoding/json's Decoder, as sonic's fallback does
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
