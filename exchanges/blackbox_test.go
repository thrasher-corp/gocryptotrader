package exchange_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	shared "github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

type mockEx struct {
	shared.CustomEx
	flow chan int
}

func (m *mockEx) UpdateTradablePairs(context.Context) error {
	m.flow <- 42
	return nil
}

func TestBootstrap(t *testing.T) {
	m := &mockEx{
		shared.CustomEx{},
		make(chan int, 1),
	}
	m.Features.Enabled.AutoPairUpdates = true
	err := exchange.Bootstrap(t.Context(), m)
	require.NoError(t, err, "Bootstrap must not error")
	require.Len(t, m.flow, 1)
	assert.Equal(t, 42, <-m.flow, "UpdateTradablePairs should be called on the exchange")
}
