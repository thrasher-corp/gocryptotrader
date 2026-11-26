package bitfinex

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

// A single malformed websocket frame must not panic the process.
func TestWsMalformedFrames(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup Instance must not error")

	for _, frame := range []string{`[]`, `[0]`, `[0,"n"]`, `[0,"os"]`, `[0,"n",[]]`} {
		assert.NotPanics(t, func() {
			assert.Error(t, e.wsHandleData(t.Context(), []byte(frame)), frame)
		}, frame)
	}
}
