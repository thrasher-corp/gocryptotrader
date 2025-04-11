package versions

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVersion10UpgradeExchange(t *testing.T) {
	t.Parallel()

	got, err := (&Version10{}).UpgradeExchange(context.Background(), nil)
	require.NoError(t, err)
	require.Nil(t, got)

	payload := []byte(`{"name":"test"}`)
	expected := []byte(`{"name":"test","websocketMetricsLogging":false}`)
	got, err = (&Version10{}).UpgradeExchange(context.Background(), payload)
	require.NoError(t, err)
	require.Equal(t, expected, got)

	payload = []byte(`{"name":"test","websocketMetricsLogging":true}`)
	got, err = (&Version10{}).UpgradeExchange(context.Background(), payload)
	require.NoError(t, err)
	require.Equal(t, payload, got)
}

func TestVersion10DowngradeExchange(t *testing.T) {
	t.Parallel()

	got, err := (&Version10{}).DowngradeExchange(context.Background(), nil)
	require.NoError(t, err)
	require.Nil(t, got)

	payload := []byte(`{"name":"test","websocketMetricsLogging":false}`)
	expected := []byte(`{"name":"test"}`)
	got, err = (&Version10{}).DowngradeExchange(context.Background(), payload)
	require.NoError(t, err)
	require.Equal(t, expected, got)
}

func TestVersion10Exchanges(t *testing.T) {
	t.Parallel()
	assert := require.New(t)
	assert.Equal([]string{"*"}, (&Version10{}).Exchanges())
}
