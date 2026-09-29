package filtered

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func untracked(t *testing.T) {
	assert.Equalf(t, 1, 1, "should match")
}
