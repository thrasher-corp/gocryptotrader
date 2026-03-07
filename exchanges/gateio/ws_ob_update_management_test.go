package gateio

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket/buffer"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestFetchWSOrderbookSnapshot(t *testing.T) {
	t.Parallel()

	_, err := e.fetchWSOrderbookSnapshot(t.Context(), currency.NewBTCUSDT(), asset.Spread)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	got, err := e.fetchWSOrderbookSnapshot(t.Context(), currency.NewBTCUSDT(), asset.USDTMarginedFutures)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestExtractOrderbookLimit(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup must not error")
	_, err := e.extractOrderbookLimit(1337)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = e.extractOrderbookLimit(asset.Spot)
	require.ErrorIs(t, err, subscription.ErrNotFound)

	err = e.Websocket.AddSubscriptions(nil, &subscription.Subscription{Channel: subscription.OrderbookChannel, Interval: kline.Interval(time.Millisecond * 420)})
	require.NoError(t, err)

	_, err = e.extractOrderbookLimit(asset.Spot)
	require.ErrorIs(t, err, errInvalidOrderbookUpdateInterval)

	err = e.Websocket.RemoveSubscriptions(nil, &subscription.Subscription{Channel: subscription.OrderbookChannel, Interval: kline.Interval(time.Millisecond * 420)})
	require.NoError(t, err)

	// Add dummy subscription so that it can be matched and a limit/level can be extracted for initial orderbook sync spot.
	err = e.Websocket.AddSubscriptions(nil, &subscription.Subscription{Channel: subscription.OrderbookChannel, Interval: kline.HundredMilliseconds})
	require.NoError(t, err)

	for _, tc := range []struct {
		asset asset.Item
		exp   uint64
	}{
		{asset: asset.Spot, exp: 100},
		{asset: asset.USDTMarginedFutures, exp: futuresOrderbookUpdateLimit},
		{asset: asset.CoinMarginedFutures, exp: futuresOrderbookUpdateLimit},
		{asset: asset.DeliveryFutures, exp: deliveryFuturesUpdateLimit},
		{asset: asset.Options, exp: optionOrderbookUpdateLimit},
	} {
		limit, err := e.extractOrderbookLimit(tc.asset)
		require.NoError(t, err)
		require.Equal(t, tc.exp, limit)
	}
}

func TestCheckPendingUpdate(t *testing.T) {
	t.Parallel()

	skip, err := checkPendingUpdate(100, 100, &orderbook.Update{UpdateID: 100})
	require.NoError(t, err)
	require.True(t, skip)

	_, err = checkPendingUpdate(100, 102, &orderbook.Update{UpdateID: 102})
	require.ErrorIs(t, err, buffer.ErrOrderbookSnapshotOutdated)

	skip, err = checkPendingUpdate(100, 101, &orderbook.Update{UpdateID: 101})
	require.NoError(t, err)
	require.False(t, skip)
}

func TestOBManagerProcessOrderbookUpdateHTTPMocked(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup must not error")
	e.Name = "ManagerHTTPMocked"
	err := testexch.MockHTTPInstance(e, "/api/v4/")
	require.NoError(t, err, "MockHTTPInstance must not error")

	// Add dummy subscription so that it can be matched and a limit/level can be extracted for initial orderbook sync spot.
	err = e.Websocket.AddSubscriptions(nil, &subscription.Subscription{Channel: subscription.OrderbookChannel, Interval: kline.TwentyMilliseconds})
	require.NoError(t, err)

	m := buffer.NewUpdateManager(&buffer.UpdateParams{
		FetchDelay:         0,
		FetchDeadline:      buffer.DefaultWSOrderbookUpdateDeadline,
		FetchOrderbook:     e.fetchWSOrderbookSnapshot,
		CheckPendingUpdate: checkPendingUpdate,
		BufferInstance:     &e.Websocket.Orderbook,
	})
	err = m.ProcessOrderbookUpdate(t.Context(), 27596272446, &orderbook.Update{
		UpdateID:   27596272447,
		Pair:       currency.NewBTCUSDT(),
		Asset:      asset.Spot,
		AllowEmpty: true,
		UpdateTime: time.Now(),
	})
	require.NoError(t, err, "ProcessOrderbookUpdate must not error")

	// Wait for the background sync goroutine to complete and orderbook to be synced
	require.Eventually(t, func() bool {
		_, err := e.Websocket.Orderbook.LastUpdateID(currency.NewBTCUSDT(), asset.Spot)
		return err == nil
	}, time.Second*5, time.Millisecond*50, "orderbook must eventually be synced")

	err = m.ProcessOrderbookUpdate(t.Context(), 27596272448, &orderbook.Update{
		UpdateID:   27596272449,
		Pair:       currency.NewBTCUSDT(),
		Asset:      asset.Spot,
		AllowEmpty: true,
		UpdateTime: time.Now(),
	})
	require.NoError(t, err, "ProcessOrderbookUpdate must not error on synced orderbook")

	id, err := e.Websocket.Orderbook.LastUpdateID(currency.NewBTCUSDT(), asset.Spot)
	require.NoError(t, err, "LastUpdateID must not error")
	assert.Equal(t, int64(27596272449), id, "LastUpdateID should be updated to orderbook.Update.UpdateID")
}
