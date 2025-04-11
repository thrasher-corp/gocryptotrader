package v3

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVersionUpgradeExchange(t *testing.T) {
	t.Parallel()

	got, err := (&Version{}).UpgradeExchange(context.Background(), nil)
	require.NoError(t, err)
	require.Nil(t, got)

	payload := []byte(`{"orderbook": {"verificationBypass": false,"websocketBufferLimit": 5,"websocketBufferEnabled": false,"publishPeriod": 10000000000}}`)
	expected := []byte(`{"orderbook": {"verificationBypass": false,"websocketBufferLimit": 5,"websocketBufferEnabled": false}}`)
	got, err = (&Version{}).UpgradeExchange(context.Background(), payload)
	require.NoError(t, err)
	require.Equal(t, expected, got)
}

func TestVersionDowngradeExchange(t *testing.T) {
	t.Parallel()

	got, err := (&Version{}).DowngradeExchange(context.Background(), nil)
	require.NoError(t, err)
	require.Nil(t, got)

	payload := []byte(`{"orderbook": {"verificationBypass": false,"websocketBufferLimit": 5,"websocketBufferEnabled": false}}`)
	expected := []byte(`{"orderbook": {"verificationBypass": false,"websocketBufferLimit": 5,"websocketBufferEnabled": false,"publishPeriod":10000000000}}`)
	got, err = (&Version{}).DowngradeExchange(context.Background(), payload)
	require.NoError(t, err)
	require.Equal(t, expected, got)
}

func TestVersionExchanges(t *testing.T) {
	t.Parallel()
	assert := require.New(t)
	assert.Equal([]string{"*"}, (&Version{}).Exchanges())
}
