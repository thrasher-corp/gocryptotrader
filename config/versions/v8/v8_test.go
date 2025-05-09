package v8_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	v8 "github.com/thrasher-corp/gocryptotrader/config/versions/v8"
)

func TestVersion10UpgradeExchange(t *testing.T) {
	t.Parallel()

	got, err := (&v8.Version{}).UpgradeExchange(context.Background(), nil)
	require.NoError(t, err)
	require.Nil(t, got)

	payload := []byte(`{"name":"test"}`)
	expected := []byte(`{"name":"test","websocketMetricsLogging":false}`)
	got, err = (&v8.Version{}).UpgradeExchange(context.Background(), payload)
	require.NoError(t, err)
	require.Equal(t, expected, got)

	payload = []byte(`{"name":"test","websocketMetricsLogging":true}`)
	got, err = (&v8.Version{}).UpgradeExchange(context.Background(), payload)
	require.NoError(t, err)
	require.Equal(t, payload, got)
}

func TestVersion10DowngradeExchange(t *testing.T) {
	t.Parallel()

	got, err := (&v8.Version{}).DowngradeExchange(context.Background(), nil)
	require.NoError(t, err)
	require.Nil(t, got)

	payload := []byte(`{"name":"test","websocketMetricsLogging":false}`)
	expected := []byte(`{"name":"test"}`)
	got, err = (&v8.Version{}).DowngradeExchange(context.Background(), payload)
	require.NoError(t, err)
	require.Equal(t, expected, got)
}

func TestVersion10Exchanges(t *testing.T) {
	t.Parallel()
	assert := require.New(t)
	assert.Equal([]string{"*"}, (&v8.Version{}).Exchanges())
}
