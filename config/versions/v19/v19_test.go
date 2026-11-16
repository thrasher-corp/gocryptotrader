package v19_test

import (
	"testing"

	"github.com/buger/jsonparser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config/versions"
	v19 "github.com/thrasher-corp/gocryptotrader/config/versions/v19"
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
			output, err := new(v19.Version).UpgradeConfig(t.Context(), []byte(tc.input))
			require.NoError(t, err, "UpgradeConfig must not error")
			assert.JSONEq(t, tc.expected, string(output), "UpgradeConfig should remove, and only remove, the obsolete timestamp setting")
		})
	}
}

func TestDowngradeConfig(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		input    string
		expected string
		errs     []error
		errField string
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
		{
			name:     "malformed gRPC configuration",
			input:    `{"remoteControl":{"gRPC":`,
			errs:     []error{common.ErrGettingField, jsonparser.MalformedJsonError},
			errField: "gRPC configuration",
		},
		{
			name:     "malformed timestamp precision",
			input:    `{"remoteControl":{"gRPC":{"timeInNanoSeconds":true!}}}`,
			errs:     []error{common.ErrGettingField, jsonparser.UnknownValueTypeError},
			errField: "gRPC timestamp precision",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			output, err := new(v19.Version).DowngradeConfig(t.Context(), []byte(tc.input))
			if len(tc.errs) != 0 {
				for _, want := range tc.errs {
					require.ErrorIs(t, err, want, "DowngradeConfig must preserve the field-access sentinel and parser error")
				}
				require.ErrorContains(t, err, tc.errField, "DowngradeConfig must name the field it failed to read")
				assert.Equal(t, tc.input, string(output), "DowngradeConfig should return the original configuration on error")
				return
			}
			require.NoError(t, err, "DowngradeConfig must not error")
			assert.JSONEq(t, tc.expected, string(output), "DowngradeConfig should restore the legacy default without overriding explicit settings")
		})
	}
}

func TestRegisteredMigration(t *testing.T) {
	t.Parallel()

	input := []byte(`{"version":18,"remoteControl":{"gRPC":{"enabled":true,"timeInNanoSeconds":true}},"exchanges":[{"name":"Okx","api":{"urlEndpoints":{"WebsocketSpotURL":"wss://ws.okx.com/ws/v5/public"}}}]}`)
	upgraded, err := versions.Manager.Deploy(t.Context(), input, 19)
	require.NoError(t, err, "Deploy must apply the registered v19 upgrade")
	assert.JSONEq(t, `{"version":19,"remoteControl":{"gRPC":{"enabled":true}},"exchanges":[{"name":"Okx","api":{"urlEndpoints":{"WebsocketSpotURL":"wss://ws.okx.com/ws/v5/public"}}}]}`, string(upgraded), "Deploy should remove the obsolete setting, preserve v18 endpoints and set version 19")

	downgraded, err := versions.Manager.Deploy(t.Context(), upgraded, 18)
	require.NoError(t, err, "Deploy must apply the registered v19 downgrade")
	assert.JSONEq(t, `{"version":18,"remoteControl":{"gRPC":{"enabled":true,"timeInNanoSeconds":false}},"exchanges":[{"name":"Okx","api":{"urlEndpoints":{"WebsocketSpotURL":"wss://ws.okx.com/ws/v5/public"}}}]}`, string(downgraded), "Deploy should restore the legacy default, preserve v18 endpoints and set version 18")
}
