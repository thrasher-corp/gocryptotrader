package v10_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v10 "github.com/thrasher-corp/gocryptotrader/config/versions/v10"
)

func TestUpgradeConfig(t *testing.T) {
	t.Parallel()
	in := []byte(`{"remoteControl":{"enabled":true,"deprecatedRPC":{"enabled":true,"listenAddress":"localhost:9050"},"websocketRPC":{"enabled":true,"listenAddress":"localhost:9051","connectionLimit":1,"maxAuthFailures":3,"allowInsecureOrigin":true}}}`)
	out, err := new(v10.Version).UpgradeConfig(t.Context(), in)
	require.NoError(t, err)
	const expected = `{"remoteControl":{"enabled":true}}`
	assert.JSONEq(t, expected, string(out))
}

func TestDowngradeConfig(t *testing.T) {
	t.Parallel()
	in := []byte("meow, moocow, woof, quack")
	out, err := new(v10.Version).DowngradeConfig(t.Context(), bytes.Clone(in))
	require.NoError(t, err)
	assert.Equal(t, out, in)
}
