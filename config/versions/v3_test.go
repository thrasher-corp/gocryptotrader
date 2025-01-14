package versions

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVersion3UpgradeExchange(t *testing.T) {
	t.Parallel()

	got, err := (&Version3{}).UpgradeExchange(context.Background(), nil)
	require.NoError(t, err)
	require.Nil(t, got)

	payload := []byte(`{"orderbook": {"verificationBypass": false,"websocketBufferLimit": 5,"websocketBufferEnabled": false,"publishPeriod": 10000000000}}`)
	expected := []byte(`{"orderbook": {"verificationBypass": false,"websocketBufferLimit": 5,"websocketBufferEnabled": false}}`)
	got, err = (&Version3{}).UpgradeExchange(context.Background(), payload)
	require.NoError(t, err)
	require.Equal(t, expected, got)
}

func TestVersion3DowngradeExchange(t *testing.T) {
	t.Parallel()

	got, err := (&Version3{}).DowngradeExchange(context.Background(), nil)
	require.NoError(t, err)
	require.Nil(t, got)

	payload := []byte(`{"orderbook": {"verificationBypass": false,"websocketBufferLimit": 5,"websocketBufferEnabled": false}}`)
	expected := []byte(`{"orderbook": {"verificationBypass": false,"websocketBufferLimit": 5,"websocketBufferEnabled": false,"publishPeriod":10000000000}}`)
	got, err = (&Version3{}).DowngradeExchange(context.Background(), payload)
	require.NoError(t, err)
	require.Equal(t, expected, got)
}

func TestVersion3Exchanges(t *testing.T) {
	t.Parallel()
	assert := require.New(t)
	assert.Equal([]string{"*"}, (&Version3{}).Exchanges())
}
