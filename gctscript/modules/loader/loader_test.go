package loader

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetModuleMap(t *testing.T) {
	x := GetModuleMap()
	require.NotNil(t, x, "GetModuleMap must not return nil")
	assert.NotZero(t, x.Len(), "GetModuleMap should return a map with entries")
}
