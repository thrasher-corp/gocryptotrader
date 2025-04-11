package v3_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	v3 "github.com/thrasher-corp/gocryptotrader/config/versions/v3"
)

func TestUpgradeExchange(t *testing.T) {
	t.Parallel()

	got, err := (&v3.Version{}).UpgradeExchange(context.Background(), nil)
	require.NoError(t, err)
	require.Nil(t, got)

	payload := []byte(`{"orderbook": {"verificationBypass": false,"websocketBufferLimit": 5,"websocketBufferEnabled": false,"publishPeriod": 10000000000}}`)
	expected := []byte(`{"orderbook": {"verificationBypass": false,"websocketBufferLimit": 5,"websocketBufferEnabled": false}}`)
	got, err = (&v3.Version{}).UpgradeExchange(context.Background(), payload)
	require.NoError(t, err)
	require.Equal(t, expected, got)
}

func TestDowngradeExchange(t *testing.T) {
	t.Parallel()

	got, err := (&v3.Version{}).DowngradeExchange(context.Background(), nil)
	require.NoError(t, err)
	require.Nil(t, got)

	payload := []byte(`{"orderbook": {"verificationBypass": false,"websocketBufferLimit": 5,"websocketBufferEnabled": false}}`)
	expected := []byte(`{"orderbook": {"verificationBypass": false,"websocketBufferLimit": 5,"websocketBufferEnabled": false,"publishPeriod":10000000000}}`)
	got, err = (&v3.Version{}).DowngradeExchange(context.Background(), payload)
	require.NoError(t, err)
	require.Equal(t, expected, got)
}

func TestExchanges(t *testing.T) {
	t.Parallel()
	assert := require.New(t)
	assert.Equal([]string{"*"}, (&v3.Version{}).Exchanges())
}
