package gateio

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRateLimits(t *testing.T) {
	for epl := range optionsTradingHistoryEPL {
		if epl == 0 {
			continue
		}
		assert.NotEmptyf(t, packageRateLimits[epl], "Empty rate limit not found for const %v", epl)
	}
}
