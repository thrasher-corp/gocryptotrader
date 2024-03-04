package exchange

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream/buffer"
)

var btcusd = currency.NewPair(currency.BTC, currency.USD)

type websocketProvider struct{ IBotExchange }

func (w *websocketProvider) GetWebsocket() (*stream.Websocket, error) {
	websocketOfJustice := &stream.Websocket{}
	websocketOfJustice.DataHandler = make(chan interface{})
	go func() {
		for {
			<-websocketOfJustice.DataHandler
		}
	}()
	err := websocketOfJustice.Orderbook.Setup(&config.Exchange{}, &buffer.Config{}, websocketOfJustice.DataHandler)
	return websocketOfJustice, err
}

func TestNewOrderbookBuilder(t *testing.T) {
	t.Parallel()

	ch := make(chan struct{})
	fetcher := func(_ context.Context, pair currency.Pair, a asset.Item) (*orderbook.Base, error) {
		<-ch
		return &orderbook.Base{Exchange: "test", Pair: pair, Asset: a, LastUpdated: time.Now()}, nil
	}

	var wg sync.WaitGroup
	wg.Add(10)
	checker := func(*orderbook.Base, *orderbook.Update, bool) (bool, error) {
		wg.Done()
		return false, nil
	}

	_, err := NewOrderbookBuilder(nil, nil, nil)
	assert.ErrorIs(t, err, errExchangeIsNil)

	wsProv := &websocketProvider{}

	_, err = NewOrderbookBuilder(IBotExchange(wsProv), nil, nil)
	assert.ErrorIs(t, err, errOrderbookFetchingFunctionUndefined)

	_, err = NewOrderbookBuilder(IBotExchange(wsProv), fetcher, nil)
	assert.ErrorIs(t, err, errOrderbookCheckingFunctionUndefined)

	builder, err := NewOrderbookBuilder(IBotExchange(wsProv), fetcher, checker)
	require.NoError(t, err)

	err = builder.Process(context.Background(), nil)
	assert.ErrorIs(t, err, errOrderbookUpdateIsNil)

	err = builder.Process(context.Background(), &orderbook.Update{})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	err = builder.Process(context.Background(), &orderbook.Update{Pair: btcusd})
	assert.ErrorIs(t, err, asset.ErrInvalidAsset)

	// Stage a fetch and store pending updates to ensure the builder is working
	for x := 0; x < defaultWindow; x++ {
		err = builder.Process(context.Background(), &orderbook.Update{
			Pair:       btcusd,
			Asset:      asset.Spot,
			Bids:       []orderbook.Item{{Price: 6969, Amount: float64(x)}},
			Asks:       []orderbook.Item{{Price: 69420, Amount: float64(x)}},
			UpdateTime: time.Now(),
		})
		require.NoError(t, err)
	}

	// release the fetcher
	close(ch)
	// Wait until checker has been called
	wg.Wait()
}

func TestTicketMachine(t *testing.T) {
	t.Parallel()

	tm := ticketMachine{}

	// These tickets should instantly be available
	for x := 0; x < defaultWindow; x++ {
		select {
		case <-tm.getTicket():
		default:
			t.Error("ticket should be available", x)
		}
	}

	// These tickets should be pending
	pending := make([]chan struct{}, 0, defaultWindow)
	for x := 0; x < defaultWindow; x++ {
		ticky := tm.getTicket()
		select {
		case <-ticky:
			t.Error("ticket should be pending")
		default:
			pending = append(pending, ticky)
		}
	}

	// Release a ticket
	for x := 0; x < defaultWindow; x++ {
		tm.releaseTicket()
	}

	// These tickets should instantly be available
	for x := range pending {
		select {
		case <-pending[x]:
		default:
			t.Error("ticket should be available")
		}
	}

	// release the rest of the tickets
	for x := 0; x < defaultWindow; x++ {
		tm.releaseTicket()
	}

	tm.releaseTicket() // panic check
}
