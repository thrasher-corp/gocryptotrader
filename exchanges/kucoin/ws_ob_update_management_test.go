package kucoin

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket/buffer"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestFetchWSOrderbookSnapshot(t *testing.T) {
	t.Parallel()

	_, err := e.fetchWSOrderbookSnapshot(t.Context(), currency.EMPTYPAIR, asset.Futures)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	xbtusdtm := currency.NewPair(currency.XBT, currency.USDTM)
	_, err = e.fetchWSOrderbookSnapshot(t.Context(), xbtusdtm, asset.FutureCombo)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	got, err := e.fetchWSOrderbookSnapshot(t.Context(), currency.NewBTCUSDT(), asset.Spot)
	require.NoError(t, err)
	require.NotEmpty(t, got)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	got, err = e.fetchWSOrderbookSnapshot(t.Context(), xbtusdtm, asset.Futures)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestCheckPendingUpdate(t *testing.T) {
	t.Parallel()

	_, err := checkPendingUpdate(5, 10, &orderbook.Update{UpdateID: 8})
	require.ErrorIs(t, err, buffer.ErrOrderbookSnapshotOutdated, "must error when there are missing updates in the sequence")

	skip, err := checkPendingUpdate(5, 2, &orderbook.Update{UpdateID: 4})
	require.NoError(t, err)
	require.True(t, skip, "must skip update that is before fetched snapshot")

	bids := []orderbook.Level{{ID: 2}, {ID: 4}, {ID: 6}}
	asks := []orderbook.Level{{ID: 3}, {ID: 5}, {ID: 7}}
	updates := &orderbook.Update{UpdateID: 7, Bids: bids, Asks: asks}

	skip, err = checkPendingUpdate(4, 2, updates)
	require.NoError(t, err)
	require.False(t, skip, "must not skip update that is in sequence")
	require.Len(t, updates.Bids, 1, "must retain only relevant bid updates")
	require.Len(t, updates.Asks, 2, "must retain only relevant ask updates")
}

func TestOBManagerProcessOrderbookUpdateHTTPMocked(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup must not error")
	e.Name = "ManagerHTTPMocked"
	err := testexch.MockHTTPInstance(e, "/api")
	require.NoError(t, err, "MockHTTPInstance must not error")

	m := buffer.NewUpdateManager(&buffer.UpdateParams{
		FetchDelay:         0,
		FetchDeadline:      buffer.DefaultWSOrderbookUpdateDeadline,
		FetchOrderbook:     e.fetchWSOrderbookSnapshot,
		CheckPendingUpdate: checkPendingUpdate,
		BufferInstance:     &e.Websocket.Orderbook,
	})
	xbtusdtm := currency.NewPair(currency.XBT, currency.USDTM)
	err = m.ProcessOrderbookUpdate(t.Context(), 1732973817560, &orderbook.Update{
		UpdateID:   1732973817560,
		Pair:       xbtusdtm,
		Asset:      asset.Futures,
		AllowEmpty: true,
		UpdateTime: time.Now(),
	})
	require.NoError(t, err, "ProcessOrderbookUpdate must not error")

	// Wait for the background sync goroutine to complete and orderbook to be synced
	require.Eventually(t, func() bool {
		_, err := e.Websocket.Orderbook.LastUpdateID(xbtusdtm, asset.Futures)
		return err == nil
	}, time.Second*5, time.Millisecond*50, "orderbook must eventually be synced")

	err = m.ProcessOrderbookUpdate(t.Context(), 1732973817561, &orderbook.Update{
		UpdateID:   1732973817562,
		Pair:       xbtusdtm,
		Asset:      asset.Futures,
		AllowEmpty: true,
		UpdateTime: time.Now(),
	})
	require.NoError(t, err, "ProcessOrderbookUpdate must not error on synced orderbook")

	id, err := e.Websocket.Orderbook.LastUpdateID(xbtusdtm, asset.Futures)
	require.NoError(t, err, "LastUpdateID must not error")
	assert.Equal(t, int64(1732973817562), id, "LastUpdateID should be updated to orderbook.Update.UpdateID")
}
