package gateio

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
)

func TestApplyUpdate(t *testing.T) {
	t.Parallel()

	m := wsOBUpdateManager{m: make(map[key.PairAsset]*updateCache)}
	err := m.applyUpdate(t.Context(), g, 20, 1337, &orderbook.Update{})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	pair := currency.NewPair(currency.BABY, currency.BABYDOGE)
	err = g.Websocket.Orderbook.LoadSnapshot(&orderbook.Base{
		Exchange:       g.Name,
		Pair:           pair,
		Asset:          asset.Futures,
		Bids:           []orderbook.Tranche{{Price: 1, Amount: 1}},
		Asks:           []orderbook.Tranche{{Price: 1, Amount: 1}},
		LastUpdated:    time.Now(),
		UpdatePushedAt: time.Now(),
		LastUpdateID:   1336,
	})
	require.NoError(t, err)

	err = m.applyUpdate(t.Context(), g, 20, 1335, &orderbook.Update{
		UpdateID: 1336,
		Pair:     pair,
		Asset:    asset.Futures,
	})
	require.ErrorIs(t, err, errOutOfOrder)

	err = m.applyUpdate(t.Context(), g, 20, 1337, &orderbook.Update{
		UpdateID:   1338,
		Pair:       pair,
		Asset:      asset.Futures,
		AllowEmpty: true,
		UpdateTime: time.Now(),
	})
	require.NoError(t, err)

	// Test orderbook snapshot is behind update
	err = m.applyUpdate(t.Context(), g, 20, 1340, &orderbook.Update{
		UpdateID:   1341,
		Pair:       pair,
		Asset:      asset.Futures,
		AllowEmpty: true,
		UpdateTime: time.Now(),
	})
	require.NoError(t, err)

	cache := m.getCache(pair, asset.Futures)

	cache.mtx.Lock()
	assert.Len(t, cache.buffer, 1)
	assert.True(t, cache.updating)
	cache.mtx.Unlock()

	// Test orderbook snapshot is behind update
	err = m.applyUpdate(t.Context(), g, 20, 1342, &orderbook.Update{
		UpdateID:   1343,
		Pair:       pair,
		Asset:      asset.Futures,
		AllowEmpty: true,
		UpdateTime: time.Now(),
	})
	require.NoError(t, err)

	cache.mtx.Lock()
	assert.Len(t, cache.buffer, 2)
	assert.True(t, cache.updating)
	cache.mtx.Unlock()
}
