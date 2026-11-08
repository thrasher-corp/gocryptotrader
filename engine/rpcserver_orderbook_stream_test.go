package engine

import (
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/dispatch"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
)

var errStopOrderbookStream = errors.New("stop orderbook stream")

// orderbookStreamRecorder records the first frame and makes Send fail so a handler must return.
type orderbookStreamRecorder struct {
	gctrpc.GoCryptoTraderService_GetOrderbookStreamServer
	sent *gctrpc.OrderbookResponse
}

func (r *orderbookStreamRecorder) Send(resp *gctrpc.OrderbookResponse) error {
	r.sent = resp
	return errStopOrderbookStream
}

// setupOrderbookStream creates an isolated exchange and book with distinct source timestamps.
func setupOrderbookStream(t *testing.T) (*RPCServer, *orderbook.Depth, *gctrpc.GetOrderbookStreamRequest, time.Time) {
	t.Helper()
	em := NewExchangeManager()
	exch, err := em.NewExchangeByName("binance")
	require.NoError(t, err, "NewExchangeByName must not error")
	exch.SetDefaults()
	b := exch.GetBase()
	b.Name = newUniqueFakeExchangeName()
	b.Enabled = true
	pair := currency.NewBTCUSDT()
	b.CurrencyPairs.Pairs = map[asset.Item]*currency.PairStore{
		asset.Spot: {
			Available: currency.Pairs{pair}, Enabled: currency.Pairs{pair}, AssetEnabled: true,
			ConfigFormat: &currency.PairFormat{Uppercase: true}, RequestFormat: &currency.PairFormat{Uppercase: true},
		},
	}
	require.NoError(t, em.Add(fExchange{IBotExchange: exch}), "Add must not error")
	depth, err := orderbook.DeployDepth(b.Name, pair, asset.Spot)
	require.NoError(t, err, "DeployDepth must not error")
	lastUpdated := time.Unix(2_243_640_188, 123_456_789).UTC()
	require.NoError(t, depth.LoadSnapshot(&orderbook.Book{
		Exchange: b.Name, Pair: pair, Asset: asset.Spot,
		LastUpdated: lastUpdated, LastPushed: lastUpdated.Add(time.Second),
		Bids: orderbook.Levels{{Price: 100, Amount: 1}}, Asks: orderbook.Levels{{Price: 101, Amount: 1}}, RestSnapshot: true,
	}), "LoadSnapshot must not error")
	return &RPCServer{Engine: &Engine{ExchangeManager: em}}, depth, &gctrpc.GetOrderbookStreamRequest{
		Exchange: b.Name, Pair: &gctrpc.CurrencyPair{Base: pair.Base.String(), Quote: pair.Quote.String()}, AssetType: asset.Spot.String(),
	}, lastUpdated
}

// requireOrderbookStreamStops bounds failures of the Send error propagation contract.
func requireOrderbookStreamStops(t *testing.T, errs <-chan error) {
	t.Helper()
	select {
	case err := <-errs:
		require.ErrorIs(t, err, errStopOrderbookStream, "orderbook stream must return its Send error")
	case <-time.After(5 * time.Second):
		require.FailNow(t, "orderbook stream must stop once its stream fails")
	}
}

func TestGetOrderbookStream(t *testing.T) {
	t.Parallel()
	for name, invalid := range map[string]bool{"valid book": false, "invalid book": true} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) {
				s, depth, request, lastUpdated := setupOrderbookStream(t)
				if invalid {
					require.ErrorIs(t, depth.Invalidate(errors.New("test invalidation")), orderbook.ErrOrderbookInvalid, "Invalidate must report the invalid book")
				}
				before := time.Now()
				stream := new(orderbookStreamRecorder)
				errs := make(chan error, 1)
				go func() { errs <- s.GetOrderbookStream(request, stream) }()
				requireOrderbookStreamStops(t, errs)
				require.NotNil(t, stream.sent, "GetOrderbookStream must send a frame")
				require.NotNil(t, stream.sent.LastUpdated, "every frame must carry a timestamp")
				require.NoError(t, stream.sent.LastUpdated.CheckValid(), "every timestamp must be valid")
				if invalid {
					assert.NotEmpty(t, stream.sent.Error, "an invalid book should carry an error")
					assert.WithinRange(t, stream.sent.LastUpdated.AsTime(), before, time.Now(), "an error frame should be stamped when sent")
				} else {
					assert.Empty(t, stream.sent.Error, "a valid book should carry no error")
					assert.Equal(t, lastUpdated, stream.sent.LastUpdated.AsTime(), "the book timestamp should preserve nanoseconds and its source field")
				}
			})
		})
	}
}

func TestGetExchangeOrderbookStream(t *testing.T) {
	// The orderbook store uses the global dispatcher, so finish before parallel engine tests change its lifecycle.
	for name, invalid := range map[string]bool{"valid book": false, "invalid book": true} {
		t.Run(name, func(t *testing.T) {
			require.NoError(t, dispatch.Start(1, 16), "dispatcher must start")
			defer func() { assert.NoError(t, dispatch.Stop(), "dispatcher should stop") }()
			s, depth, request, lastUpdated := setupOrderbookStream(t)
			if invalid {
				require.ErrorIs(t, depth.Invalidate(errors.New("test invalidation")), orderbook.ErrOrderbookInvalid, "Invalidate must report the invalid book")
			}
			before := time.Now()
			stream := new(orderbookStreamRecorder)
			errs := make(chan error, 1)
			go func() {
				errs <- s.GetExchangeOrderbookStream(&gctrpc.GetExchangeOrderbookStreamRequest{Exchange: request.Exchange}, stream)
			}()
			// The global dispatcher pools channels across tests, so they cannot belong to a synctest bubble.
			// Publish until the subscription receives a frame, with a deadline if the handler ignores Send errors.
			ticker := time.NewTicker(time.Millisecond)
			defer ticker.Stop()
			timeout := time.NewTimer(5 * time.Second)
			defer timeout.Stop()
		waitForStream:
			for {
				select {
				case err := <-errs:
					require.ErrorIs(t, err, errStopOrderbookStream, "GetExchangeOrderbookStream must return its Send error")
					break waitForStream
				case <-ticker.C:
					depth.Publish()
				case <-timeout.C:
					require.FailNow(t, "GetExchangeOrderbookStream must stop after a failed Send")
				}
			}
			require.NotNil(t, stream.sent, "GetExchangeOrderbookStream must send a frame")
			require.NotNil(t, stream.sent.LastUpdated, "every frame must carry a timestamp")
			require.NoError(t, stream.sent.LastUpdated.CheckValid(), "every timestamp must be valid")
			if invalid {
				assert.NotEmpty(t, stream.sent.Error, "an invalid book should carry an error")
				assert.WithinRange(t, stream.sent.LastUpdated.AsTime(), before, time.Now(), "an error frame should be stamped when sent")
			} else {
				assert.Empty(t, stream.sent.Error, "a valid book should carry no error")
				assert.Equal(t, lastUpdated, stream.sent.LastUpdated.AsTime(), "the book timestamp should preserve nanoseconds and its source field")
			}
		})
	}
}
