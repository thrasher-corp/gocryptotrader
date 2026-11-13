package exchange

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsSupported(t *testing.T) {
	assert.True(t, IsSupported("BiTStaMp"), "IsSupported should accept a supported exchange case-insensitively")
	assert.False(t, IsSupported("meowexch"), "IsSupported should reject an unknown exchange")
	assert.False(t, IsSupported("bitmex"), "IsSupported should reject retired BitMEX")
}
