package deribit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
)

func TestWebsocketAPIAcceptBlockRFQ(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("WebsocketAPI.AcceptBlockRFQ needs makers' quotes on the account's Block RFQ, which live tests cannot arrange")
	}
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().AcceptBlockRFQ(t.Context(), mockAcceptBlockRFQRequest())
	require.NoError(t, err, "WebsocketAPI.AcceptBlockRFQ must not error")
	assert.Equal(t, testDocumentedAcceptedBlockRFQ(), result, "WebsocketAPI.AcceptBlockRFQ should decode every field")

	_, err = newTestExchange(t).WebsocketAPI().AcceptBlockRFQ(t.Context(), mockAcceptBlockRFQRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.AcceptBlockRFQ should error without an authenticated connection")
}

func TestWebsocketAPIAddBlockRFQQuote(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("WebsocketAPI.AddBlockRFQQuote needs another account's open Block RFQ to quote, which live tests cannot arrange")
	}
	ex, venue := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().AddBlockRFQQuote(t.Context(), mockAddBlockRFQQuoteRequest())
	require.NoError(t, err, "WebsocketAPI.AddBlockRFQQuote must not error")
	assert.Equal(t, testDocumentedBlockRFQQuote(), result, "WebsocketAPI.AddBlockRFQQuote should decode every field")
	requests := venue.received("private/add_block_rfq_quote")
	require.Len(t, requests, 1, "WebsocketAPI.AddBlockRFQQuote must send one request")
	exp := `{"label":"example_quote","block_rfq_id":3,"amount":10000,"direction":"buy","legs":[{"instrument_name":"BTC-15NOV24","price":69600,"ratio":1,"direction":"buy"}],"hedge":{"amount":10,"direction":"buy","price":70000,"instrument_name":"BTC-PERPETUAL"},"execution_instruction":"any_part_of","expires_at":1745312540321}`
	assert.JSONEq(t, exp, string(requests[0].Params), "WebsocketAPI.AddBlockRFQQuote should send Deribit's documented params, the expiry in milliseconds")

	_, err = newTestExchange(t).WebsocketAPI().AddBlockRFQQuote(t.Context(), mockAddBlockRFQQuoteRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.AddBlockRFQQuote should error without an authenticated connection")
}

func TestWebsocketAPICancelAllBlockRFQQuotes(t *testing.T) {
	t.Parallel()
	skipLiveUnlessCanManipulate(t)
	arg := &CancelAllBlockRFQQuotesRequest{}
	if mockTests {
		arg.BlockRFQID = 154
	}
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().CancelAllBlockRFQQuotes(t.Context(), arg)
	require.NoError(t, err, "WebsocketAPI.CancelAllBlockRFQQuotes must not error")
	if mockTests {
		assert.Equal(t, &CancelAllBlockRFQQuotesResponse{CancelledCount: 1}, result, "WebsocketAPI.CancelAllBlockRFQQuotes should decode the count of quotes cancelled")
	} else {
		assert.Empty(t, result.DetailedReports, "WebsocketAPI.CancelAllBlockRFQQuotes should return a count without a detailed request")
	}

	_, err = newTestExchange(t).WebsocketAPI().CancelAllBlockRFQQuotes(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.CancelAllBlockRFQQuotes should error without an authenticated connection")
}

func TestWebsocketAPICancelBlockRFQ(t *testing.T) {
	t.Parallel()
	skipLiveUnlessCanManipulate(t)
	ex, _ := newWebsocketTestExchange(t)
	blockRFQID := uint64(mockCancelBlockRFQID)
	if !mockTests {
		created, err := ex.WebsocketAPI().CreateBlockRFQ(t.Context(), liveCreateBlockRFQRequest())
		require.NoError(t, err, "WebsocketAPI.CreateBlockRFQ must create a Block RFQ to cancel")
		blockRFQID = created.BlockRFQID
	}
	result, err := ex.WebsocketAPI().CancelBlockRFQ(t.Context(), blockRFQID)
	require.NoError(t, err, "WebsocketAPI.CancelBlockRFQ must not error")
	if mockTests {
		assert.Equal(t, testDocumentedCancelledBlockRFQ(), result, "WebsocketAPI.CancelBlockRFQ should decode every field")
	} else {
		assert.Equal(t, BlockRFQStateCancelled, result.State, "WebsocketAPI.CancelBlockRFQ should cancel the Block RFQ")
	}

	_, err = newTestExchange(t).WebsocketAPI().CancelBlockRFQ(t.Context(), blockRFQID)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.CancelBlockRFQ should error without an authenticated connection")
}

func TestWebsocketAPICancelBlockRFQQuote(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("WebsocketAPI.CancelBlockRFQQuote needs a quote on another account's open Block RFQ, which live tests cannot arrange")
	}
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().CancelBlockRFQQuote(t.Context(), mockCancelBlockRFQQuoteRequest())
	require.NoError(t, err, "WebsocketAPI.CancelBlockRFQQuote must not error")
	assert.Equal(t, testDocumentedCancelledBlockRFQQuote(), result, "WebsocketAPI.CancelBlockRFQQuote should decode every field")

	_, err = newTestExchange(t).WebsocketAPI().CancelBlockRFQQuote(t.Context(), mockCancelBlockRFQQuoteRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.CancelBlockRFQQuote should error without an authenticated connection")
}

func TestWebsocketAPICancelBlockRFQTrigger(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("WebsocketAPI.CancelBlockRFQTrigger needs an active trade trigger, which needs makers' quotes live tests cannot arrange")
	}
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().CancelBlockRFQTrigger(t.Context(), mockCancelBlockRFQTriggerID)
	require.NoError(t, err, "WebsocketAPI.CancelBlockRFQTrigger must not error")
	assert.Equal(t, testDocumentedBlockRFQWithCancelledTrigger(), result, "WebsocketAPI.CancelBlockRFQTrigger should decode every field")

	_, err = newTestExchange(t).WebsocketAPI().CancelBlockRFQTrigger(t.Context(), mockCancelBlockRFQTriggerID)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.CancelBlockRFQTrigger should error without an authenticated connection")
}

func TestWebsocketAPICreateBlockRFQ(t *testing.T) {
	t.Parallel()
	skipLiveUnlessCanManipulate(t)
	arg := mockCreateBlockRFQRequest()
	if !mockTests {
		arg = liveCreateBlockRFQRequest()
	}
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().CreateBlockRFQ(t.Context(), arg)
	require.NoError(t, err, "WebsocketAPI.CreateBlockRFQ must not error")
	if mockTests {
		assert.Equal(t, testDocumentedCreatedBlockRFQ(), result, "WebsocketAPI.CreateBlockRFQ should decode every field")
	} else {
		_, err = ex.WebsocketAPI().CancelBlockRFQ(t.Context(), result.BlockRFQID)
		require.NoError(t, err, "WebsocketAPI.CancelBlockRFQ must cancel the Block RFQ created")
	}

	_, err = newTestExchange(t).WebsocketAPI().CreateBlockRFQ(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.CreateBlockRFQ should error without an authenticated connection")
}

func TestWebsocketAPIEditBlockRFQQuote(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("WebsocketAPI.EditBlockRFQQuote needs a quote on another account's open Block RFQ, which live tests cannot arrange")
	}
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().EditBlockRFQQuote(t.Context(), mockEditBlockRFQQuoteRequest())
	require.NoError(t, err, "WebsocketAPI.EditBlockRFQQuote must not error")
	assert.Equal(t, testDocumentedEditedBlockRFQQuote(), result, "WebsocketAPI.EditBlockRFQQuote should decode every field")

	_, err = newTestExchange(t).WebsocketAPI().EditBlockRFQQuote(t.Context(), mockEditBlockRFQQuoteRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.EditBlockRFQQuote should error without an authenticated connection")
}

func TestWebsocketAPIGetBlockRFQMakers(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetBlockRFQMakers(t.Context())
	require.NoError(t, err, "WebsocketAPI.GetBlockRFQMakers must not error")
	if mockTests {
		assert.Equal(t, []string{"MAKER1", "MAKER2", "MAKER3"}, result, "WebsocketAPI.GetBlockRFQMakers should decode the makers' aliases")
	} else {
		assert.NotEmpty(t, result, "WebsocketAPI.GetBlockRFQMakers should return the makers' aliases")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetBlockRFQMakers(t.Context())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetBlockRFQMakers should error without an authenticated connection")
}

func TestWebsocketAPIGetBlockRFQQuotes(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	arg := &GetBlockRFQQuotesRequest{}
	if mockTests {
		arg.BlockRFQID = 1
	}
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetBlockRFQQuotes(t.Context(), arg)
	require.NoError(t, err, "WebsocketAPI.GetBlockRFQQuotes must not error")
	if mockTests {
		assert.Equal(t, testDocumentedBlockRFQQuotes(), result, "WebsocketAPI.GetBlockRFQQuotes should decode every field")
	} else {
		for i := range result {
			assert.NotZerof(t, result[i].BlockRFQQuoteID, "WebsocketAPI.GetBlockRFQQuotes should return the ID of quote %d", i)
		}
	}

	_, err = newTestExchange(t).WebsocketAPI().GetBlockRFQQuotes(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetBlockRFQQuotes should error without an authenticated connection")
}

func TestWebsocketAPIGetBlockRFQUserInfo(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetBlockRFQUserInfo(t.Context())
	require.NoError(t, err, "WebsocketAPI.GetBlockRFQUserInfo must not error")
	if mockTests {
		assert.Equal(t, testDocumentedBlockRFQUserInfo(), result, "WebsocketAPI.GetBlockRFQUserInfo should decode every field")
	} else {
		assert.NotEmpty(t, result.Users, "WebsocketAPI.GetBlockRFQUserInfo should return the account's users")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetBlockRFQUserInfo(t.Context())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetBlockRFQUserInfo should error without an authenticated connection")
}

func TestWebsocketAPIGetBlockRFQs(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetBlockRFQs(t.Context(), mockGetBlockRFQsRequest())
	require.NoError(t, err, "WebsocketAPI.GetBlockRFQs must not error")
	if mockTests {
		assert.Equal(t, testDocumentedBlockRFQs(), result, "WebsocketAPI.GetBlockRFQs should decode every field")
	} else {
		for i := range result.BlockRFQs {
			assert.Equalf(t, BlockRFQRoleMaker, result.BlockRFQs[i].Role, "WebsocketAPI.GetBlockRFQs should return Block RFQ %d in the role requested", i)
		}
	}

	_, err = newTestExchange(t).WebsocketAPI().GetBlockRFQs(t.Context(), mockGetBlockRFQsRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetBlockRFQs should error without an authenticated connection")
}

func TestWebsocketAPIGetBlockRFQTrades(t *testing.T) {
	t.Parallel()
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetBlockRFQTrades(t.Context(), testGetBlockRFQTradesRequest())
	require.NoError(t, err, "WebsocketAPI.GetBlockRFQTrades must not error")
	if mockTests {
		assert.Equal(t, testRecordedBlockRFQTrades(), result, "WebsocketAPI.GetBlockRFQTrades should decode every field")
	} else {
		assert.NotEmpty(t, result.BlockRFQs, "WebsocketAPI.GetBlockRFQTrades should return Block RFQ trades")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetBlockRFQTrades(t.Context(), testGetBlockRFQTradesRequest())
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetBlockRFQTrades should error without a websocket connection")

	if !mockTests {
		return
	}
	anyCurrency, venue := newWebsocketTestExchange(t)
	anyTrades, err := anyCurrency.WebsocketAPI().GetBlockRFQTrades(t.Context(), &GetBlockRFQTradesRequest{Currency: currency.ANY, Count: 10})
	require.NoError(t, err, "WebsocketAPI.GetBlockRFQTrades must not error for every currency")
	require.Len(t, anyTrades.BlockRFQs, 1, "WebsocketAPI.GetBlockRFQTrades must return the recorded trade")
	assert.Equal(t, uint64(58310), anyTrades.BlockRFQs[0].ID, "WebsocketAPI.GetBlockRFQTrades should return every currency's trades")
	requests := venue.received("public/get_block_rfq_trades")
	require.Len(t, requests, 1, "WebsocketAPI.GetBlockRFQTrades must send one request")
	assert.JSONEq(t, `{"currency":"any","count":10}`, string(requests[0].Params), "WebsocketAPI.GetBlockRFQTrades should send every currency as any, as Deribit rejects ANY")
}
