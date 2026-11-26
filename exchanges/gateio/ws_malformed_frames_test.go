package gateio

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

	assert.NotPanics(t, func() {
		_ = e.processSpotBalances(t.Context(), []byte(`[null]`))
	})
}
