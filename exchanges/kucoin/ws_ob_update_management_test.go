package kucoin

import (
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket/orderbookmanager"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestFetchWSOrderbookSnapshot(t *testing.T) {
	t.Parallel()

	_, err := e.fetchWSOrderbookSnapshot(t.Context(), currency.EMPTYPAIR, asset.Futures)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "fetchWSOrderbookSnapshot must reject an empty pair")

	xbtusdtm := currency.NewPair(currency.XBT, currency.USDTM)
	_, err = e.fetchWSOrderbookSnapshot(t.Context(), xbtusdtm, asset.FutureCombo)
	require.ErrorIs(t, err, asset.ErrNotSupported, "fetchWSOrderbookSnapshot must reject an asset without an increment topic")

	t.Run("orderbook_error", func(t *testing.T) {
		t.Parallel()
		noCredentialsExchange := new(Exchange)
		require.NoError(t, testexch.Setup(noCredentialsExchange), "Setup must not error")
		noCredentialsExchange.API.AuthenticatedSupport = true

		got, err := noCredentialsExchange.fetchWSOrderbookSnapshot(t.Context(), currency.NewBTCUSDT(), asset.Spot)
		require.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "must return the authenticated orderbook error")
		assert.Nil(t, got, "snapshot should be nil when the authenticated orderbook fetch fails")
	})

	t.Run("futures_orderbook_error", func(t *testing.T) {
		t.Parallel()
		// KuCoin answers a snapshot request for a contract it does not list with this reply, captured live
		server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/v1/level2/snapshot", r.URL.Path, "the futures snapshot should be requested")
			assert.Equal(t, "NOTACONTRACTUSDTM", r.URL.Query().Get("symbol"), "the snapshot should be requested for the contract")
			_, _ = w.Write([]byte(`{"msg":"Invalid symbol.","code":"200003"}`))
		}))
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
		got, err := ex.fetchWSOrderbookSnapshot(t.Context(), currency.NewPair(currency.NewCode("NOTACONTRACT"), currency.USDTM), asset.Futures)
		require.ErrorContains(t, err, "200003", "fetchWSOrderbookSnapshot must return the futures orderbook error")
		assert.Nil(t, got, "snapshot should be nil when the futures orderbook fetch fails")
	})

	t.Run("asset_without_increment_topic", func(t *testing.T) {
		t.Parallel()
		ex := testInstance(t)
		format := &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}
		require.NoError(t, ex.SetAssetPairStore(asset.Options, currency.PairStore{AssetEnabled: true, RequestFormat: format, ConfigFormat: format}), "SetAssetPairStore must not error")
		_, err := ex.fetchWSOrderbookSnapshot(t.Context(), currency.NewBTCUSDT(), asset.Options)
		require.ErrorIs(t, err, asset.ErrNotSupported, "fetchWSOrderbookSnapshot must reject an asset without an increment topic")
		assert.NotErrorIs(t, err, currency.ErrAssetNotFound, "the asset should be rejected for its missing increment topic, not its pair format")
	})

	got, err := e.fetchWSOrderbookSnapshot(t.Context(), xbtusdtm, asset.Futures)
	require.NoError(t, err, "fetchWSOrderbookSnapshot must not error for futures")
	assert.NotEmpty(t, got.Bids, "the futures snapshot should carry bids")
	assert.Positive(t, got.LastUpdateID, "the futures snapshot should carry its sequence")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	got, err = e.fetchWSOrderbookSnapshot(t.Context(), currency.NewBTCUSDT(), asset.Spot)
	require.NoError(t, err, "fetchWSOrderbookSnapshot must not error for spot")
	assert.NotEmpty(t, got.Bids, "the spot snapshot should carry bids")
	assert.Positive(t, got.LastUpdateID, "the spot snapshot should carry its sequence")
}

func TestCheckPendingUpdate(t *testing.T) {
	t.Parallel()

	_, err := checkPendingUpdate(5, 10, &orderbook.Update{UpdateID: 8})
	require.ErrorIs(t, err, orderbookmanager.ErrOrderbookSnapshotOutdated, "must error when there are missing updates in the sequence")

	skip, err := checkPendingUpdate(5, 2, &orderbook.Update{UpdateID: 4})
	require.NoError(t, err, "checkPendingUpdate must not error for an update before the snapshot")
	require.True(t, skip, "must skip update that is before fetched snapshot")

	bids := []orderbook.Level{{ID: 2}, {ID: 4}, {ID: 6}}
	asks := []orderbook.Level{{ID: 3}, {ID: 5}, {ID: 7}}
	updates := &orderbook.Update{UpdateID: 7, Bids: bids, Asks: asks}

	skip, err = checkPendingUpdate(4, 2, updates)
	require.NoError(t, err, "checkPendingUpdate must not error for an update in sequence")
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

	m := orderbookmanager.NewUpdateManager(&orderbookmanager.UpdateManagerParams{
		FetchDelay:         0,
		FetchDeadline:      orderbookmanager.DefaultWSOrderbookUpdateDeadline,
		FetchOrderbook:     e.fetchWSOrderbookSnapshot,
		CheckPendingUpdate: checkPendingUpdate,
		Orderbook:          &e.Websocket.Orderbook,
	})
	xbtusdtm := currency.NewPair(currency.XBT, currency.USDTM)
	err = m.ProcessOrderbookUpdate(t.Context(), 1748391355097, &orderbook.Update{
		UpdateID:   1748391355097,
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

	err = m.ProcessOrderbookUpdate(t.Context(), 1748391355098, &orderbook.Update{
		UpdateID:   1748391355099,
		Pair:       xbtusdtm,
		Asset:      asset.Futures,
		AllowEmpty: true,
		UpdateTime: time.Now(),
	})
	require.NoError(t, err, "ProcessOrderbookUpdate must not error on synced orderbook")

	id, err := e.Websocket.Orderbook.LastUpdateID(xbtusdtm, asset.Futures)
	require.NoError(t, err, "LastUpdateID must not error")
	assert.Equal(t, int64(1748391355099), id, "LastUpdateID should be updated to orderbook.Update.UpdateID")
}

func TestNewOrderbookUpdateManager(t *testing.T) {
	t.Parallel()
	ex := testInstance(t)
	ex.Name = t.Name()
	m := ex.newOrderbookUpdateManager()
	require.NotNil(t, m, "newOrderbookUpdateManager must return a manager")
	xbtusdtm := currency.NewPair(currency.XBT, currency.USDTM)
	// The update's sequence range covers any snapshot the endpoint returns, so the live book syncs as the recorded one does
	require.NoError(t, m.ProcessOrderbookUpdate(t.Context(), 1, &orderbook.Update{
		UpdateID:   math.MaxInt64,
		Pair:       xbtusdtm,
		Asset:      asset.Futures,
		AllowEmpty: true,
		UpdateTime: time.Now(),
	}), "ProcessOrderbookUpdate must not error")
	require.Eventually(t, func() bool {
		_, err := ex.Websocket.Orderbook.LastUpdateID(xbtusdtm, asset.Futures)
		return err == nil
	}, 10*time.Second, 50*time.Millisecond, "the manager must sync the book into the exchange's websocket orderbook store")
	id, err := ex.Websocket.Orderbook.LastUpdateID(xbtusdtm, asset.Futures)
	require.NoError(t, err, "LastUpdateID must not error")
	assert.Equal(t, int64(math.MaxInt64), id, "the queued update should be applied on top of the snapshot")
}
