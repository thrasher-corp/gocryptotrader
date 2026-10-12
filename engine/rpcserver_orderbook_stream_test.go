package engine

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"google.golang.org/grpc"
)

type orderbookTestStream struct {
	grpc.ServerStream
	ctx           context.Context //nolint:containedctx // The fake gRPC stream retains its request context for Context().
	sent          chan *gctrpc.OrderbookResponse
	sendErr       error
	ignoreContext bool
}

func (s *orderbookTestStream) Context() context.Context { return s.ctx }

func (s *orderbookTestStream) Send(response *gctrpc.OrderbookResponse) error {
	if s.ignoreContext {
		s.sent <- response
		return nil
	}
	if err := s.ctx.Err(); err != nil {
		return err
	}
	if s.sendErr != nil {
		return s.sendErr
	}
	select {
	case s.sent <- response:
		return nil
	case <-s.ctx.Done():
		return s.ctx.Err()
	}
}

func TestGetOrderbookStream(t *testing.T) {
	t.Parallel()
	errSend := errors.New("stream send failed")
	for _, tc := range []struct {
		name             string
		deadline         bool
		update           bool
		sendErr          error
		wantErr          error
		alreadyCancelled bool
	}{
		{name: "cancel while book is idle", wantErr: context.Canceled},
		{name: "deadline while book is idle", deadline: true, wantErr: context.DeadlineExceeded},
		{name: "available-only book updates then cancels", update: true, wantErr: context.Canceled},
		{name: "send error", sendErr: errSend, wantErr: errSend},
		{name: "already cancelled sends no cached snapshot", alreadyCancelled: true, wantErr: context.Canceled},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			em := NewExchangeManager()
			exch, err := em.NewExchangeByName(testExchange)
			require.NoError(t, err)
			exch.SetDefaults()
			exch.SetEnabled(true)
			exch.GetBase().Name = t.Name()
			pair := currency.NewBTCUSDT()
			require.NoError(t, exch.GetBase().CurrencyPairs.Store(asset.Spot, &currency.PairStore{Available: currency.Pairs{pair}, AssetEnabled: true, ConfigFormat: &currency.EMPTYFORMAT}))
			require.NoError(t, em.Add(exch))
			depth, err := orderbook.DeployDepth(t.Name(), pair, asset.Spot)
			require.NoError(t, err)
			book := &orderbook.Book{Exchange: t.Name(), Pair: pair, Asset: asset.Spot, LastUpdated: time.Unix(1700000000, 0), Bids: []orderbook.Level{{Price: 100, Amount: 2}}, Asks: []orderbook.Level{{Price: 101, Amount: 3}}}
			depth.AssignOptions(book)
			require.NoError(t, depth.LoadSnapshot(book))
			ctx, cancel := context.WithCancel(t.Context())
			if tc.deadline {
				cancel()
				ctx, cancel = context.WithTimeout(t.Context(), 100*time.Millisecond)
			}
			defer cancel()
			if tc.alreadyCancelled {
				cancel()
			}
			stream := &orderbookTestStream{ctx: ctx, sent: make(chan *gctrpc.OrderbookResponse, 2), sendErr: tc.sendErr, ignoreContext: tc.alreadyCancelled}
			s := RPCServer{Engine: &Engine{ExchangeManager: em}}
			result := make(chan error, 1)
			go func() {
				result <- s.GetOrderbookStream(&gctrpc.GetOrderbookStreamRequest{Exchange: t.Name(), AssetType: "spot", Pair: &gctrpc.CurrencyPair{Base: "BTC", Quote: "USDT"}}, stream)
			}()
			finished := false
			defer func() {
				if finished {
					return
				}
				// Wake the original uncancellable wait if a regression fails the bounded assertion.
				cancel()
				require.NoError(t, depth.LoadSnapshot(book))
				select {
				case <-result:
				case <-time.After(time.Second):
					assert.Fail(t, "test stream routine should finish during cleanup")
				}
			}()
			if tc.sendErr == nil && !tc.alreadyCancelled {
				select {
				case response := <-stream.sent:
					require.Empty(t, response.Error)
					require.Len(t, response.Bids, 1)
					assert.Equal(t, 100.0, response.Bids[0].Price)
				case <-time.After(time.Second):
					require.FailNow(t, "stream must send the cached available-only book")
				}
				if tc.update {
					book.Bids[0].Price = 99
					book.LastUpdated = book.LastUpdated.Add(time.Second)
					poll := time.NewTicker(time.Millisecond)
					defer poll.Stop()
					wait := time.NewTimer(time.Second)
					defer wait.Stop()
				waitUpdate:
					for {
						select {
						case <-poll.C:
							require.NoError(t, depth.LoadSnapshot(book))
						case response := <-stream.sent:
							if response.LastUpdated < book.LastUpdated.UnixMicro() {
								assert.Equal(t, 100.0, response.Bids[0].Price, "an already queued cached snapshot may precede the update")
								continue
							}
							assert.Equal(t, 99.0, response.Bids[0].Price)
							assert.Equal(t, book.LastUpdated.UnixMicro(), response.LastUpdated)
							break waitUpdate
						case <-wait.C:
							require.FailNow(t, "stream must send subsequent book updates")
						}
					}
				}
				if !tc.deadline {
					cancel()
				}
			}
			select {
			case err := <-result:
				finished = true
				require.ErrorIs(t, err, tc.wantErr)
				if tc.alreadyCancelled {
					assert.Empty(t, stream.sent, "the RPC should check cancellation before calling Send")
				}
			case <-time.After(time.Second):
				require.FailNow(t, "stream must return without needing another orderbook push")
			}
		})
	}
}
