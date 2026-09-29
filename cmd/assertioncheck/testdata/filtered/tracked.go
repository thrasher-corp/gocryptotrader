package filtered

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func tracked(t *testing.T) {
	assert.Equalf(t, 1, 1, "should match") // want "has a plain message; use Equal"
}
