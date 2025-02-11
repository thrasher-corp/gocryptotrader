package versions

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVersion4UpgradeExchange(t *testing.T) {
	t.Parallel()

	got, err := (&Version4{}).UpgradeExchange(context.Background(), nil)
	require.NoError(t, err)
	require.Nil(t, got)

	payload := []byte(`{"name":"test"}`)
	expected := []byte(`{"name":"test","websocketMetricsLogging":false}`)
	got, err = (&Version4{}).UpgradeExchange(context.Background(), payload)
	require.NoError(t, err)
	require.Equal(t, expected, got)

	payload = []byte(`{"name":"test","websocketMetricsLogging":true}`)
	got, err = (&Version4{}).UpgradeExchange(context.Background(), payload)
	require.NoError(t, err)
	require.Equal(t, payload, got)
}

func TestVersion4DowngradeExchange(t *testing.T) {
	t.Parallel()

	got, err := (&Version4{}).DowngradeExchange(context.Background(), nil)
	require.NoError(t, err)
	require.Nil(t, got)

	payload := []byte(`{"name":"test","websocketMetricsLogging":false}`)
	expected := []byte(`{"name":"test"}`)
	got, err = (&Version4{}).DowngradeExchange(context.Background(), payload)
	require.NoError(t, err)
	require.Equal(t, expected, got)
}

func TestVersion4Exchanges(t *testing.T) {
	t.Parallel()
	assert := require.New(t)
	assert.Equal([]string{"*"}, (&Version4{}).Exchanges())
}
