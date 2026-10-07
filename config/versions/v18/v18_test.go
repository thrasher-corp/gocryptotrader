package v18_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config/versions"
	v18 "github.com/thrasher-corp/gocryptotrader/config/versions/v18"
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
			output, err := new(v18.Version).UpgradeConfig(t.Context(), []byte(tc.input))
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
			name:     "explicit seconds selection",
			input:    `{"remoteControl":{"gRPC":{"timeInNanoSeconds":false}}}`,
			expected: `{"remoteControl":{"gRPC":{"timeInNanoSeconds":false}}}`,
		},
		{
			name:     "explicit nanoseconds selection",
			input:    `{"remoteControl":{"gRPC":{"timeInNanoSeconds":true}}}`,
			expected: `{"remoteControl":{"gRPC":{"timeInNanoSeconds":true}}}`,
		},
		{
			name:     "null selection",
			input:    `{"remoteControl":{"gRPC":{"timeInNanoSeconds":null}}}`,
			expected: `{"remoteControl":{"gRPC":{"timeInNanoSeconds":false}}}`,
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
			output, err := new(v18.Version).DowngradeConfig(t.Context(), []byte(tc.input))
			require.NoError(t, err, "DowngradeConfig must not error")
			assert.JSONEq(t, tc.expected, string(output), "DowngradeConfig should restore the legacy default without overriding explicit settings")
		})
	}
}

func TestRegisteredMigration(t *testing.T) {
	t.Parallel()

	input := []byte(`{"version":17,"remoteControl":{"gRPC":{"enabled":true,"timeInNanoSeconds":true}}}`)
	upgraded, err := versions.Manager.Deploy(t.Context(), input, 18)
	require.NoError(t, err, "Deploy must apply the registered v18 upgrade")
	assert.JSONEq(t, `{"version":18,"remoteControl":{"gRPC":{"enabled":true}}}`, string(upgraded), "Deploy should remove the obsolete setting and set version 18")

	downgraded, err := versions.Manager.Deploy(t.Context(), upgraded, 17)
	require.NoError(t, err, "Deploy must apply the registered v18 downgrade")
	assert.JSONEq(t, `{"version":17,"remoteControl":{"gRPC":{"enabled":true,"timeInNanoSeconds":false}}}`, string(downgraded), "Deploy should restore the legacy default and set version 17")

	previousSeconds := []byte(`{"version":17,"remoteControl":{"gRPC":{"enabled":true,"timeInNanoSeconds":false}}}`)
	upgraded, err = versions.Manager.Deploy(t.Context(), previousSeconds, 18)
	require.NoError(t, err, "Deploy must upgrade a previous seconds selection")
	downgraded, err = versions.Manager.Deploy(t.Context(), upgraded, 17)
	require.NoError(t, err, "Deploy must downgrade the upgraded seconds selection")
	assert.JSONEq(t, `{"version":17,"remoteControl":{"gRPC":{"enabled":true,"timeInNanoSeconds":false}}}`, string(downgraded), "Deploy should return the legacy default unchanged")
}

func TestDowngradeConfigRejectsMalformedConfig(t *testing.T) {
	t.Parallel()

	input := []byte(`{"remoteControl":{"gRPC":`)
	output, err := new(v18.Version).DowngradeConfig(t.Context(), input)
	require.Error(t, err, "DowngradeConfig must reject malformed configuration")
	assert.Equal(t, input, output, "DowngradeConfig should return the original configuration on error")
}
