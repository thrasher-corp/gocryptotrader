package exchange

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsSupported(t *testing.T) {
	assert.True(t, IsSupported("BiTStaMp"), "Supported exchange should be valid")
	assert.False(t, IsSupported("meowexch"), "Unsupported exchange should be invalid")
	assert.False(t, IsSupported("bitmex"), "BitMEX should no longer be supported")
}
