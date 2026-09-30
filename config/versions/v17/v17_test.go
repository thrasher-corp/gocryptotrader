package v17_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config/versions"
	v17 "github.com/thrasher-corp/gocryptotrader/config/versions/v17"
)

func TestUpgradeConfig(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "previous default",
			input:    `{"remoteControl":{"gRPC":{"enabled":true,"timeInNanoSeconds":false}}}`,
			expected: `{"remoteControl":{"gRPC":{"enabled":true}}}`,
		},
		{
			name:     "previous nanosecond selection",
			input:    `{"remoteControl":{"gRPC":{"timeInNanoSeconds":true,"custom":"preserved"}}}`,
			expected: `{"remoteControl":{"gRPC":{"custom":"preserved"}}}`,
		},
		{
			name:     "missing setting",
			input:    `{"remoteControl":{"gRPC":{"enabled":true}}}`,
			expected: `{"remoteControl":{"gRPC":{"enabled":true}}}`,
		},
		{
			name:     "missing gRPC configuration",
			input:    `{"name":"gocryptotrader"}`,
			expected: `{"name":"gocryptotrader"}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			output, err := new(v17.Version).UpgradeConfig(t.Context(), []byte(tc.input))
			require.NoError(t, err, "UpgradeConfig must not error")
			assert.JSONEq(t, tc.expected, string(output), "UpgradeConfig should remove only the obsolete timestamp setting")
		})
	}
}

func TestDowngradeConfig(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "gRPC configuration",
			input:    `{"remoteControl":{"gRPC":{"enabled":true,"custom":"preserved"}}}`,
			expected: `{"remoteControl":{"gRPC":{"enabled":true,"custom":"preserved","timeInNanoSeconds":false}}}`,
		},
		{
			name:     "missing gRPC configuration remains absent",
			input:    `{"name":"gocryptotrader"}`,
			expected: `{"name":"gocryptotrader"}`,
		},
		{
			name:     "non-object gRPC configuration",
			input:    `{"remoteControl":{"gRPC":null}}`,
			expected: `{"remoteControl":{"gRPC":null}}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			output, err := new(v17.Version).DowngradeConfig(t.Context(), []byte(tc.input))
			require.NoError(t, err, "DowngradeConfig must not error")
			assert.JSONEq(t, tc.expected, string(output), "DowngradeConfig should restore the legacy default where needed")
		})
	}
}

func TestRegisteredMigration(t *testing.T) {
	t.Parallel()

	input := []byte(`{"version":16,"remoteControl":{"gRPC":{"enabled":true,"timeInNanoSeconds":true}}}`)
	upgraded, err := versions.Manager.Deploy(t.Context(), input, 17)
	require.NoError(t, err, "Deploy must apply the registered v17 upgrade")
	assert.JSONEq(t, `{"version":17,"remoteControl":{"gRPC":{"enabled":true}}}`, string(upgraded), "Deploy should remove the obsolete setting and set version 17")

	downgraded, err := versions.Manager.Deploy(t.Context(), upgraded, 16)
	require.NoError(t, err, "Deploy must apply the registered v17 downgrade")
	assert.JSONEq(t, `{"version":16,"remoteControl":{"gRPC":{"enabled":true,"timeInNanoSeconds":false}}}`, string(downgraded), "Deploy should restore the legacy default and set version 16")
}

func TestDowngradeConfigRejectsMalformedConfig(t *testing.T) {
	t.Parallel()

	input := []byte(`{"remoteControl":{"gRPC":`)
	output, err := new(v17.Version).DowngradeConfig(t.Context(), input)
	require.Error(t, err, "DowngradeConfig must reject malformed configuration")
	assert.Equal(t, input, output, "DowngradeConfig should return the original configuration on error")
}
