package deribit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebsocketAPIApproveBlockTrade(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("WebsocketAPI.ApproveBlockTrade needs a block trade pending the account's approval, which only a broker can create")
	}
	ex, venue := newWebsocketTestExchange(t)
	require.NoError(t, ex.WebsocketAPI().ApproveBlockTrade(t.Context(), mockBlockTradeApprovalRequest()), "WebsocketAPI.ApproveBlockTrade must not error")
	requests := venue.received("private/approve_block_trade")
	require.Len(t, requests, 1, "WebsocketAPI.ApproveBlockTrade must send one request")
	assert.JSONEq(t, `{"timestamp":1711468813551,"nonce":"bt-468nha","role":"maker"}`, string(requests[0].Params), "WebsocketAPI.ApproveBlockTrade should send the timestamp in milliseconds, the nonce and the role")

	err := newTestExchange(t).WebsocketAPI().ApproveBlockTrade(t.Context(), mockBlockTradeApprovalRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.ApproveBlockTrade should error without an authenticated connection")
}

func TestWebsocketAPIExecuteBlockTrade(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("WebsocketAPI.ExecuteBlockTrade needs a counterparty's signature, which live tests do not have")
	}
	ex, venue := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().ExecuteBlockTrade(t.Context(), mockExecuteBlockTradeRequest())
	require.NoError(t, err, "WebsocketAPI.ExecuteBlockTrade must not error")
	assert.Equal(t, testDocumentedBlockTrade(), result, "WebsocketAPI.ExecuteBlockTrade should decode every field")
	requests := venue.received("private/execute_block_trade")
	require.Len(t, requests, 1, "WebsocketAPI.ExecuteBlockTrade must send one request")
	exp := `{"timestamp":1590485535899,"nonce":"bszyprbq","role":"maker","trades":[{"instrument_name":"BTC-PERPETUAL","direction":"sell","price":8900,"amount":200000},{"instrument_name":"BTC-28MAY20-9000-C","direction":"sell","price":0.0133,"amount":5}],"counterparty_signature":"1590485595899.1Mn52L_Q.lNyNBzXXo-_QBT_wDuMgnhA7uS9tBqdQ5TLN6rxbuoAiQhyaJYGJrm5IV_9enp9niY_x8D60AJLm3yEKPUY1Dv3T0TW0n5-ADPpJF7Fpj0eVDZpZ6QCdX8snBWrSJ0TtqevnO64RCBlN1dIm2T70PP9dlhiqPDAUYI4fpB1vLYI"}`
	assert.JSONEq(t, exp, string(requests[0].Params), "WebsocketAPI.ExecuteBlockTrade should send Deribit's documented params, the timestamp in milliseconds")

	_, err = newTestExchange(t).WebsocketAPI().ExecuteBlockTrade(t.Context(), mockExecuteBlockTradeRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.ExecuteBlockTrade should error without an authenticated connection")
}

func TestWebsocketAPIGetBlockTrade(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	id := testBlockTradeID(t)
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetBlockTrade(t.Context(), id)
	require.NoError(t, err, "WebsocketAPI.GetBlockTrade must not error")
	if mockTests {
		assert.Equal(t, testDocumentedBlockTrade(), result, "WebsocketAPI.GetBlockTrade should decode every field")
	} else {
		assert.Equal(t, id, result.ID, "WebsocketAPI.GetBlockTrade should return the block trade requested")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetBlockTrade(t.Context(), id)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetBlockTrade should error without an authenticated connection")
}

func TestWebsocketAPIGetBlockTradeRequests(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	var brokerCode string
	if mockTests {
		brokerCode = mockBlockTradeBrokerCode
	}
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetBlockTradeRequests(t.Context(), brokerCode)
	require.NoError(t, err, "WebsocketAPI.GetBlockTradeRequests must not error")
	if mockTests {
		assert.Equal(t, testDocumentedPendingBlockTrades(), result, "WebsocketAPI.GetBlockTradeRequests should decode every field")
	} else {
		for i := range result {
			assert.NotEmptyf(t, result[i].Nonce, "WebsocketAPI.GetBlockTradeRequests should return the nonce of block trade request %d", i)
		}
	}

	_, err = newTestExchange(t).WebsocketAPI().GetBlockTradeRequests(t.Context(), brokerCode)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetBlockTradeRequests should error without an authenticated connection")
}

func TestWebsocketAPIGetBlockTrades(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetBlockTrades(t.Context(), mockGetBlockTradesRequest())
	require.NoError(t, err, "WebsocketAPI.GetBlockTrades must not error")
	if mockTests {
		assert.Equal(t, testDocumentedBlockTrades(), result, "WebsocketAPI.GetBlockTrades should decode every field")
	} else {
		assert.LessOrEqual(t, len(result), 1, "WebsocketAPI.GetBlockTrades should return at most the count requested")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetBlockTrades(t.Context(), mockGetBlockTradesRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetBlockTrades should error without an authenticated connection")
}

func TestWebsocketAPIGetBrokerClients(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("WebsocketAPI.GetBrokerClients needs a broker account, which live tests do not have")
	}
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetBrokerClients(t.Context(), &GetBrokerClientsRequest{})
	require.NoError(t, err, "WebsocketAPI.GetBrokerClients must not error")
	assert.Equal(t, testDocumentedBrokerClients(), result, "WebsocketAPI.GetBrokerClients should decode every field")

	_, err = newTestExchange(t).WebsocketAPI().GetBrokerClients(t.Context(), &GetBrokerClientsRequest{})
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetBrokerClients should error without an authenticated connection")
}

func TestWebsocketAPIGetBrokerTradeRequests(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("WebsocketAPI.GetBrokerTradeRequests needs a broker account, which live tests do not have")
	}
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetBrokerTradeRequests(t.Context())
	require.NoError(t, err, "WebsocketAPI.GetBrokerTradeRequests must not error")
	assert.Equal(t, testDocumentedPendingBrokerTrades(), result, "WebsocketAPI.GetBrokerTradeRequests should decode every field")

	_, err = newTestExchange(t).WebsocketAPI().GetBrokerTradeRequests(t.Context())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetBrokerTradeRequests should error without an authenticated connection")
}

func TestWebsocketAPIGetBrokerTrades(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("WebsocketAPI.GetBrokerTrades needs a broker account, which live tests do not have")
	}
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetBrokerTrades(t.Context(), mockGetBrokerTradesRequest())
	require.NoError(t, err, "WebsocketAPI.GetBrokerTrades must not error")
	assert.Equal(t, testDocumentedBrokerTrades(), result, "WebsocketAPI.GetBrokerTrades should decode every field")

	_, err = newTestExchange(t).WebsocketAPI().GetBrokerTrades(t.Context(), mockGetBrokerTradesRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetBrokerTrades should error without an authenticated connection")
}

func TestWebsocketAPIInvalidateBlockTradeSignature(t *testing.T) {
	t.Parallel()
	skipLiveUnlessCanManipulate(t)
	ex, _ := newWebsocketTestExchange(t)
	signature := mockBlockTradeSignature
	if !mockTests {
		verified, err := ex.WebsocketAPI().VerifyBlockTrade(t.Context(), liveVerifyBlockTradeRequest(t))
		require.NoError(t, err, "WebsocketAPI.VerifyBlockTrade must sign a block trade to invalidate")
		signature = verified.Signature
	}
	require.NoError(t, ex.WebsocketAPI().InvalidateBlockTradeSignature(t.Context(), signature), "WebsocketAPI.InvalidateBlockTradeSignature must not error")

	err := newTestExchange(t).WebsocketAPI().InvalidateBlockTradeSignature(t.Context(), signature)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.InvalidateBlockTradeSignature should error without an authenticated connection")
}

func TestWebsocketAPIRejectBlockTrade(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("WebsocketAPI.RejectBlockTrade needs a block trade pending the account's approval, which only a broker can create")
	}
	ex, venue := newWebsocketTestExchange(t)
	require.NoError(t, ex.WebsocketAPI().RejectBlockTrade(t.Context(), mockBlockTradeApprovalRequest()), "WebsocketAPI.RejectBlockTrade must not error")
	requests := venue.received("private/reject_block_trade")
	require.Len(t, requests, 1, "WebsocketAPI.RejectBlockTrade must send one request")
	assert.JSONEq(t, `{"timestamp":1711468813551,"nonce":"bt-468nha","role":"maker"}`, string(requests[0].Params), "WebsocketAPI.RejectBlockTrade should send the timestamp in milliseconds, the nonce and the role")

	err := newTestExchange(t).WebsocketAPI().RejectBlockTrade(t.Context(), mockBlockTradeApprovalRequest())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.RejectBlockTrade should error without an authenticated connection")
}

func TestWebsocketAPISimulateBlockTrade(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	arg := mockSimulateBlockTradeRequest()
	if !mockTests {
		arg = &SimulateBlockTradeRequest{Role: BlockTradeRoleMaker, Trades: liveBlockTradeLegs(t)}
	}
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().SimulateBlockTrade(t.Context(), arg)
	require.NoError(t, err, "WebsocketAPI.SimulateBlockTrade must not error")
	if mockTests {
		assert.True(t, result, "WebsocketAPI.SimulateBlockTrade should decode that the block trade can be executed")
	}

	_, err = newTestExchange(t).WebsocketAPI().SimulateBlockTrade(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.SimulateBlockTrade should error without an authenticated connection")
}

func TestWebsocketAPIVerifyBlockTrade(t *testing.T) {
	t.Parallel()
	skipLiveUnlessCanManipulate(t)
	arg := mockVerifyBlockTradeRequest()
	if !mockTests {
		arg = liveVerifyBlockTradeRequest(t)
	}
	ex, venue := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().VerifyBlockTrade(t.Context(), arg)
	require.NoError(t, err, "WebsocketAPI.VerifyBlockTrade must not error")
	if mockTests {
		assert.Equal(t, testDocumentedBlockTradeSignature(), result, "WebsocketAPI.VerifyBlockTrade should decode the signature")
		requests := venue.received("private/verify_block_trade")
		require.Len(t, requests, 1, "WebsocketAPI.VerifyBlockTrade must send one request")
		exp := `{"timestamp":1565172650935,"nonce":"okpdjkdo","role":"maker","trades":[{"instrument_name":"BTC-PERPETUAL","direction":"buy","price":11624,"amount":40},{"instrument_name":"BTC-9AUG19-10250-P","direction":"buy","price":0.0707,"amount":1.2}]}`
		assert.JSONEq(t, exp, string(requests[0].Params), "WebsocketAPI.VerifyBlockTrade should send Deribit's documented params, the timestamp in milliseconds")
	} else {
		require.NotEmpty(t, result.Signature, "WebsocketAPI.VerifyBlockTrade must return a signature")
		require.NoError(t, ex.WebsocketAPI().InvalidateBlockTradeSignature(t.Context(), result.Signature), "WebsocketAPI.InvalidateBlockTradeSignature must cancel the block trade verified")
	}

	_, err = newTestExchange(t).WebsocketAPI().VerifyBlockTrade(t.Context(), arg)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.VerifyBlockTrade should error without an authenticated connection")
}
