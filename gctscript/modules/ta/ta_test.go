package ta

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetModuleMap(t *testing.T) {
	require.Len(t, AllModuleNames(), 9, "AllModuleNames must return 9 modules")
}
