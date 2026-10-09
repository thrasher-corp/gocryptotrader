package engine

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/dispatch"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"google.golang.org/protobuf/proto"
)

type orderbookResponseStream struct {
	gctrpc.GoCryptoTraderService_GetOrderbookStreamServer
	response *gctrpc.OrderbookResponse
}

func (s *orderbookResponseStream) Send(response *gctrpc.OrderbookResponse) error {
	s.response = response
	return errExpectedTestError
}

func TestRPCOrderbookStringValues(t *testing.T) {
	require.NoError(t, dispatch.Start(dispatch.DefaultMaxWorkers, dispatch.DefaultJobsLimit), "starting the dispatcher must succeed")
	t.Cleanup(func() {
		require.NoError(t, dispatch.Stop(), "stopping the dispatcher must succeed")
	})
	em := NewExchangeManager()
	exch, err := em.NewExchangeByName("binance")
	require.NoError(t, err, "creating the exchange must succeed")
	exch.SetDefaults()
	base := exch.GetBase()
	base.Name = newUniqueFakeExchangeName()
	base.Enabled = true
	pair := currency.NewBTCUSD()
	base.CurrencyPairs.Pairs = map[asset.Item]*currency.PairStore{
		asset.Spot: {
			AssetEnabled:  true,
			ConfigFormat:  &currency.PairFormat{Delimiter: "-"},
			RequestFormat: &currency.PairFormat{Delimiter: "-"},
			Available:     currency.Pairs{pair},
			Enabled:       currency.Pairs{pair},
		},
	}
	require.NoError(t, em.Add(exch), "registering the exchange must succeed")
	s := RPCServer{Engine: &Engine{ExchangeManager: em, Config: &config.Config{}}}
	depth, err := orderbook.DeployDepth(base.Name, pair, asset.Spot)
	require.NoError(t, err, "deploying the depth must succeed")
	require.NoError(t, depth.LoadSnapshot(&orderbook.Book{
		Bids: orderbook.Levels{
			{Price: 1.234567891234567, Amount: 0.00000000123, StrPrice: "1.234567891234567890", StrAmount: "0.0000000012300", ID: 17},
			{Price: 1, Amount: 2, ID: 18},
		},
		Asks: orderbook.Levels{
			{Price: 2.345678912345678, Amount: 3.5, StrPrice: "2.345678912345678901", StrAmount: "3.5000", ID: 29},
			{Price: 3, Amount: 4, ID: 30},
		},
		LastUpdated: time.Now(),
	}), "loading the snapshot must succeed")
	rpcPair := &gctrpc.CurrencyPair{Base: "BTC", Quote: "USD"}
	expectedBids := []*gctrpc.OrderbookItem{
		{Price: 1.234567891234567, Amount: 0.00000000123, StrPrice: "1.234567891234567890", StrAmount: "0.0000000012300"},
		{Price: 1, Amount: 2},
	}
	expectedAsks := []*gctrpc.OrderbookItem{
		{Price: 2.345678912345678, Amount: 3.5, StrPrice: "2.345678912345678901", StrAmount: "3.5000"},
		{Price: 3, Amount: 4},
	}
	checkResponse := func(response *gctrpc.OrderbookResponse) {
		t.Helper()
		require.NotNil(t, response, "the response must be present")
		assert.Empty(t, response.Error, "the book should be valid")
		require.Len(t, response.Bids, 2, "both bids must be returned")
		require.Len(t, response.Asks, 2, "both asks must be returned")
		for i := range expectedBids {
			assert.True(t, proto.Equal(expectedBids[i], response.Bids[i]), "bid fields should retain numeric and optional string values")
			assert.True(t, proto.Equal(expectedAsks[i], response.Asks[i]), "ask fields should retain numeric and optional string values")
		}
	}
	response, err := s.GetOrderbook(t.Context(), &gctrpc.GetOrderbookRequest{Exchange: base.Name, Pair: rpcPair, AssetType: "spot"})
	require.NoError(t, err, "retrieving the snapshot must succeed")
	checkResponse(response)
	responses, err := s.GetOrderbooks(t.Context(), &gctrpc.GetOrderbooksRequest{})
	require.NoError(t, err, "retrieving all books must succeed")
	require.Len(t, responses.Orderbooks, 1, "the exchange must be included")
	require.Len(t, responses.Orderbooks[0].Orderbooks, 1, "the snapshot must be included")
	checkResponse(responses.Orderbooks[0].Orderbooks[0])
	expectedBids[0].Id, expectedBids[1].Id = 17, 18
	expectedAsks[0].Id, expectedAsks[1].Id = 29, 30
	stream := &orderbookResponseStream{}
	require.ErrorIs(t, s.GetOrderbookStream(&gctrpc.GetOrderbookStreamRequest{Exchange: base.Name, Pair: rpcPair, AssetType: "spot"}, stream), errExpectedTestError, "sending the stream response must return the test stop error")
	checkResponse(stream.response)

	stream = &orderbookResponseStream{}
	finished := make(chan error, 1)
	go func() {
		finished <- s.GetExchangeOrderbookStream(&gctrpc.GetExchangeOrderbookStreamRequest{Exchange: base.Name}, stream)
	}()
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()
	for {
		select {
		case err := <-finished:
			require.ErrorIs(t, err, errExpectedTestError, "sending the exchange stream response must return the test stop error")
			checkResponse(stream.response)
			return
		case <-ticker.C:
			depth.Publish()
		case <-timeout.C:
			t.Fatal("the exchange stream must deliver a response")
		}
	}
}
