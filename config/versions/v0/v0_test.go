package v0_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v0 "github.com/thrasher-corp/gocryptotrader/config/versions/v0"
)

func TestUpgradeConfig(t *testing.T) {
	t.Parallel()
	in := []byte(`{"untouched":true}`)
	out, err := new(v0.Version).UpgradeConfig(t.Context(), in)
	require.NoError(t, err)
	assert.Equal(t, in, out)
}

func TestDowngradeConfig(t *testing.T) {
	t.Parallel()
	in := []byte(`{"untouched":true}`)
	out, err := new(v0.Version).DowngradeConfig(t.Context(), in)
	require.NoError(t, err)
	assert.Equal(t, in, out)
}
