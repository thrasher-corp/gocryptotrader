package deribit

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const (
	// mockBlockTradeID is the block trade of Deribit's documented private/get_block_trade example
	mockBlockTradeID = "61"
	// mockBlockTradeBrokerCode is the broker code of Deribit's documented private/get_block_trade_requests example
	mockBlockTradeBrokerCode = "jpqYKgg1"
	// mockBlockTradeSignature is the signature of Deribit's documented private/invalidate_block_trade_signature example
	mockBlockTradeSignature = "1565173369982.1M9tO0Q-.z9n9WyZUU5op9pEz6Jtd2CI71QxQMMsCZAexnIfK9HQRT1pKH3clxeIbY7Bqm-yMcWIoE3IfCDPW5VEdiN-6oS0YkKUyXPD500MUf3ULKhfkmH81EZs"
	// liveBlockTradeInstrument and liveBlockTradeAmount, its minimum block trade amount in USD, are the leg live block
	// trades are verified and simulated with
	liveBlockTradeInstrument = "BTC-PERPETUAL"
	liveBlockTradeAmount     = 200000
)

// mockBlockTradeApprovalRequest returns the pending block trade of Deribit's documented private/approve_block_trade and
// private/reject_block_trade examples
func mockBlockTradeApprovalRequest() *BlockTradeApprovalRequest {
	return &BlockTradeApprovalRequest{Timestamp: time.UnixMilli(1711468813551), Nonce: "bt-468nha", Role: BlockTradeRoleMaker}
}

// mockBlockTradeLegs returns the legs of Deribit's documented private/verify_block_trade and
// private/simulate_block_trade examples
func mockBlockTradeLegs() []BlockTradeLeg {
	return []BlockTradeLeg{
		{InstrumentName: "BTC-PERPETUAL", Direction: DirectionBuy, Price: 11624, Amount: 40},
		{InstrumentName: "BTC-9AUG19-10250-P", Direction: DirectionBuy, Price: 0.0707, Amount: 1.2},
	}
}

// mockExecuteBlockTradeRequest returns the request of Deribit's documented private/execute_block_trade example
func mockExecuteBlockTradeRequest() *ExecuteBlockTradeRequest {
	return &ExecuteBlockTradeRequest{
		Timestamp: time.UnixMilli(1590485535899),
		Nonce:     "bszyprbq",
		Role:      BlockTradeRoleMaker,
		Trades: []BlockTradeLeg{
			{InstrumentName: "BTC-PERPETUAL", Direction: DirectionSell, Price: 8900, Amount: 200000},
			{InstrumentName: "BTC-28MAY20-9000-C", Direction: DirectionSell, Price: 0.0133, Amount: 5},
		},
		CounterpartySignature: "1590485595899.1Mn52L_Q.lNyNBzXXo-_QBT_wDuMgnhA7uS9tBqdQ5TLN6rxbuoAiQhyaJYGJrm5IV_9enp9niY_x8D60AJLm3yEKPUY1Dv3T0TW0n5-ADPpJF7Fpj0eVDZpZ6QCdX8snBWrSJ0TtqevnO64RCBlN1dIm2T70PP9dlhiqPDAUYI4fpB1vLYI",
	}
}

// mockGetBlockTradesRequest returns the request of Deribit's documented private/get_block_trades example, which also
// works live
func mockGetBlockTradesRequest() *GetBlockTradesRequest {
	return &GetBlockTradesRequest{Currency: currency.BTC, Count: 1}
}

// mockGetBrokerTradesRequest returns the request of Deribit's documented private/get_broker_trades example
func mockGetBrokerTradesRequest() *GetBrokerTradesRequest {
	return &GetBrokerTradesRequest{Currency: currency.BTC, Count: 1}
}

// mockSimulateBlockTradeRequest returns the request of Deribit's documented private/simulate_block_trade example
func mockSimulateBlockTradeRequest() *SimulateBlockTradeRequest {
	return &SimulateBlockTradeRequest{Role: BlockTradeRoleMaker, Trades: mockBlockTradeLegs()}
}

// mockVerifyBlockTradeRequest returns the request of Deribit's documented private/verify_block_trade example
func mockVerifyBlockTradeRequest() *VerifyBlockTradeRequest {
	return &VerifyBlockTradeRequest{Timestamp: time.UnixMilli(1565172650935), Nonce: "okpdjkdo", Role: BlockTradeRoleMaker, Trades: mockBlockTradeLegs()}
}

// blockTradeTestTicker is the part of a ticker the live block trade tests price their leg from
type blockTradeTestTicker struct {
	MarkPrice float64 `json:"mark_price"`
}

// liveBlockTradeLegs returns a block trade leg of the minimum block trade amount at the instrument's mark price, which
// a live block trade can be verified and simulated with
func liveBlockTradeLegs(t *testing.T) []BlockTradeLeg {
	t.Helper()
	var ticker blockTradeTestTicker
	err := e.SendHTTPRequest(t.Context(), nonMatchingEPL, "public/ticker", &InstrumentNameRequest{InstrumentName: liveBlockTradeInstrument}, &ticker)
	require.NoError(t, err, "public/ticker must not error")
	require.Positive(t, ticker.MarkPrice, "public/ticker must return a mark price")
	// The perpetual's tick size is 0.5
	price := math.Round(ticker.MarkPrice*2) / 2
	return []BlockTradeLeg{{InstrumentName: liveBlockTradeInstrument, Direction: DirectionBuy, Price: price, Amount: liveBlockTradeAmount}}
}

// liveVerifyBlockTradeRequest returns a block trade of the live leg to verify, with a fresh timestamp and nonce
func liveVerifyBlockTradeRequest(t *testing.T) *VerifyBlockTradeRequest {
	t.Helper()
	now := time.Now()
	return &VerifyBlockTradeRequest{Timestamp: now, Nonce: "gct-" + now.Format("150405.000000"), Role: BlockTradeRoleMaker, Trades: liveBlockTradeLegs(t)}
}

// testBlockTradeID returns the mock block trade, or live, the account's latest block trade, skipping the test when the
// account has none
func testBlockTradeID(t *testing.T) string {
	t.Helper()
	if mockTests {
		return mockBlockTradeID
	}
	trades, err := e.GetBlockTrades(t.Context(), &GetBlockTradesRequest{Count: 1})
	require.NoError(t, err, "GetBlockTrades must not error")
	if len(trades) == 0 {
		t.Skip("the account has no block trades to request")
	}
	return trades[0].ID
}

// testDocumentedBlockTrade returns block trade 61, which Deribit's documented private/get_block_trade and
// private/execute_block_trade examples return
func testDocumentedBlockTrade() *BlockTrade {
	return &BlockTrade{
		ID:        "61",
		Timestamp: types.Time(time.UnixMilli(1565089523720)),
		Trades: []UserTrade{
			{
				TradeID:        "92437",
				TradeSequence:  37,
				InstrumentName: "BTC-9AUG19-10250-C",
				Timestamp:      types.Time(time.UnixMilli(1565089523719)),
				OrderType:      "limit",
				OrderID:        "343062",
				Direction:      DirectionSell,
				TickDirection:  3,
				IndexPrice:     11738,
				Price:          0.0001,
				Amount:         10,
				Liquidity:      "T",
				Fee:            0.00025,
				FeeCurrency:    currency.BTC,
				State:          "filled",
				BlockTradeID:   "61",
			},
			{
				TradeID:        "92435",
				TradeSequence:  25350,
				InstrumentName: "BTC-PERPETUAL",
				Timestamp:      types.Time(time.UnixMilli(1565089523719)),
				OrderType:      "limit",
				OrderID:        "343058",
				Direction:      DirectionBuy,
				TickDirection:  3,
				IndexPrice:     11737.98,
				Price:          11590,
				Amount:         190,
				Liquidity:      "T",
				Fee:            0.00000164,
				FeeCurrency:    currency.BTC,
				State:          "filled",
				BlockTradeID:   "61",
			},
		},
	}
}

// testDocumentedBlockTrades returns the block trades of Deribit's documented private/get_block_trades example
func testDocumentedBlockTrades() []BlockTrade {
	return []BlockTrade{{
		ID:        "66",
		Timestamp: types.Time(time.UnixMilli(1565093070165)),
		Trades: []UserTrade{
			{
				TradeID:           "92462",
				TradeSequence:     4,
				InstrumentName:    "BTC-9AUG19-11500-P",
				Timestamp:         types.Time(time.UnixMilli(1565093070164)),
				OrderType:         "limit",
				OrderID:           "343121",
				Direction:         DirectionSell,
				TickDirection:     2,
				IndexPrice:        11758.65,
				Price:             0.0151,
				Amount:            2.3,
				ImpliedVolatility: 72.38,
				Liquidity:         "M",
				FeeCurrency:       currency.BTC,
				State:             "filled",
				BlockTradeID:      "66",
			},
			{
				TradeID:        "92460",
				TradeSequence:  41,
				InstrumentName: "BTC-9AUG19",
				Timestamp:      types.Time(time.UnixMilli(1565093070164)),
				OrderType:      "limit",
				OrderID:        "343117",
				Direction:      DirectionSell,
				TickDirection:  2,
				IndexPrice:     11758.65,
				Price:          11753,
				Amount:         50,
				Liquidity:      "M",
				FeeCurrency:    currency.BTC,
				State:          "filled",
				BlockTradeID:   "66",
			},
		},
	}}
}

// testDocumentedPendingBlockTrades returns the block trade requests of Deribit's documented
// private/get_block_trade_requests example, whose username the recording scrubs
func testDocumentedPendingBlockTrades() []PendingBlockTrade {
	return []PendingBlockTrade{{
		Timestamp:         types.Time(time.UnixMilli(1742824052538)),
		Nonce:             "29rKkuD3NSBPet4njrpNWEuHBm9s",
		Trades:            []BlockTradeLeg{{InstrumentName: "BTC-PERPETUAL", Direction: DirectionBuy, Price: 87516.83, Amount: 100000}},
		Role:              BlockTradeRoleTaker,
		UserID:            8,
		BrokerCode:        mockBlockTradeBrokerCode,
		BrokerName:        "Test Broker",
		State:             PendingBlockTradeState{Value: "initial", Timestamp: types.Time(time.UnixMilli(1742824052539))},
		CounterpartyState: &PendingBlockTradeState{Value: "approved", Timestamp: types.Time(time.UnixMilli(1742824052538))},
	}}
}

// testDocumentedBrokerClients returns the clients of Deribit's documented private/get_broker_clients example, whose
// link names the recording scrubs
func testDocumentedBrokerClients() []BrokerClient {
	return []BrokerClient{
		{
			ClientID:    1,
			ClientName:  "Acme Trading Ltd",
			CompanyName: "Acme Trading Ltd",
			BrokerCode:  "ACME01",
			MainUserID:  "***456",
			Links: []BrokerClientLink{
				{ClientLinkID: 1, UserID: "***456", Note: "Primary trading account", State: "connected", ConfirmationsRequired: 1},
				{ClientLinkID: 2, UserID: "***789", State: "pending", ConfirmationsRequired: 1},
			},
		},
		{
			ClientID:    2,
			ClientName:  "Beta Capital",
			CompanyName: "Beta Capital GmbH",
			BrokerCode:  "BETA01",
			Links:       []BrokerClientLink{{ClientLinkID: 3, UserID: "***012", State: "connected", ConfirmationsRequired: 2}},
		},
	}
}

// testDocumentedPendingBrokerTrades returns the broker block trade requests of Deribit's documented
// private/get_broker_trade_requests example
func testDocumentedPendingBrokerTrades() []PendingBrokerTrade {
	trades := []BlockTradeLeg{{InstrumentName: "BTC-PERPETUAL", Direction: DirectionBuy, Price: 87516.83, Amount: 100000}}
	taker := BrokerTradeParty{State: "initial", ClientID: 1, UserID: "***008", ClientName: "Test Client", ClientLinkName: "Test Client 1", ClientLinkID: 1}
	return []PendingBrokerTrade{
		{
			Timestamp: types.Time(time.UnixMilli(1742824052547)),
			Nonce:     "3WqPoAsmde9aXCSEBVUmi2XxGkgA",
			State:     "activated",
			Trades:    trades,
			Maker:     BrokerTradeParty{State: "initial", ClientID: 1, UserID: "***009", ClientName: "Test Client", ClientLinkName: "Test Client 2", ClientLinkID: 2},
			Taker:     taker,
		},
		{
			Timestamp: types.Time(time.UnixMilli(1742824052538)),
			Nonce:     "29rKkuD3NSBPet4njrpNWEuHBm9s",
			State:     "activated",
			Trades:    trades,
			Maker:     BrokerTradeParty{State: "approved", ClientID: 1, UserID: "***009", ClientName: "Test Client", ClientLinkName: "Test Client 2", ClientLinkID: 2},
			Taker:     taker,
		},
	}
}

// testDocumentedBrokerTrades returns the broker block trades of Deribit's documented private/get_broker_trades example
func testDocumentedBrokerTrades() *BrokerTradesResponse {
	return &BrokerTradesResponse{
		History: []BrokerTrade{{
			ID:        "BLOCK-7",
			Timestamp: types.Time(time.UnixMilli(1747239767111)),
			Trades: []UserTrade{{
				TradeID:            "7",
				TradeSequence:      7,
				InstrumentName:     "BTC-PERPETUAL",
				Timestamp:          types.Time(time.UnixMilli(1747239767111)),
				Direction:          DirectionBuy,
				TickDirection:      1,
				IndexPrice:         102079.75,
				Price:              102079.75,
				Amount:             100000,
				Contracts:          10000,
				MarkPrice:          102079.75,
				BlockTradeID:       "BLOCK-7",
				BlockTradeLegCount: 1,
			}},
			Maker: BrokerTradeParty{ClientID: 2, UserID: "****009", ClientName: "Test Client 2", ClientLinkName: "Test Client 2 l", ClientLinkID: 2},
			Taker: BrokerTradeParty{ClientID: 1, UserID: "****008", ClientName: "Test Client 1", ClientLinkName: "Test Client 1 l", ClientLinkID: 1},
		}},
		NextStartID: 6,
	}
}

// testDocumentedBlockTradeSignature returns the signature of Deribit's documented private/verify_block_trade example
func testDocumentedBlockTradeSignature() *VerifyBlockTradeResponse {
	return &VerifyBlockTradeResponse{Signature: "1565172710935.1ESE83qh.g6fbgRd4VWagaJz7xdi2WaV-q-d3J0njoz1jZavuRudZJZif9uH8XdUAx1LHsu0E3e0ZG_xe1UPYlwo41xRVrkWU6OMgygDRafUkEmBuk9iLqjc9rh4"}
}

func TestApproveBlockTrade(t *testing.T) {
	t.Parallel()
	err := e.ApproveBlockTrade(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ApproveBlockTrade must reject a nil request")
	err = e.ApproveBlockTrade(t.Context(), &BlockTradeApprovalRequest{Timestamp: time.UnixMilli(1711468813551), Role: BlockTradeRoleMaker})
	require.ErrorIs(t, err, errBlockTradeNonceRequired, "ApproveBlockTrade must reject an empty nonce")

	if !mockTests {
		t.Skip("ApproveBlockTrade needs a block trade pending the account's approval, which only a broker can create")
	}
	require.NoError(t, e.ApproveBlockTrade(t.Context(), mockBlockTradeApprovalRequest()), "ApproveBlockTrade must not error")

	err = newUnavailableServerExchange(t).ApproveBlockTrade(t.Context(), mockBlockTradeApprovalRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "ApproveBlockTrade should return the request failure")
}

func TestExecuteBlockTrade(t *testing.T) {
	t.Parallel()
	_, err := e.ExecuteBlockTrade(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ExecuteBlockTrade must reject a nil request")
	arg := mockExecuteBlockTradeRequest()
	arg.Role = "broker"
	_, err = e.ExecuteBlockTrade(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidBlockTradeRole, "ExecuteBlockTrade must reject an undocumented role")
	arg = mockExecuteBlockTradeRequest()
	arg.Trades = nil
	_, err = e.ExecuteBlockTrade(t.Context(), arg)
	require.ErrorIs(t, err, errBlockTradeLegsRequired, "ExecuteBlockTrade must reject a block trade without legs")
	arg = mockExecuteBlockTradeRequest()
	arg.CounterpartySignature = ""
	_, err = e.ExecuteBlockTrade(t.Context(), arg)
	require.ErrorIs(t, err, errCounterpartySignatureRequired, "ExecuteBlockTrade must reject an empty counterparty signature")

	if !mockTests {
		t.Skip("ExecuteBlockTrade needs a counterparty's signature, which live tests do not have")
	}
	result, err := e.ExecuteBlockTrade(t.Context(), mockExecuteBlockTradeRequest())
	require.NoError(t, err, "ExecuteBlockTrade must not error")
	assert.Equal(t, testDocumentedBlockTrade(), result, "ExecuteBlockTrade should decode every field")

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).ExecuteBlockTrade(t.Context(), mockExecuteBlockTradeRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "ExecuteBlockTrade should reject a null result")
	_, err = newUnavailableServerExchange(t).ExecuteBlockTrade(t.Context(), mockExecuteBlockTradeRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "ExecuteBlockTrade should return the request failure")
}

func TestGetBlockTrade(t *testing.T) {
	t.Parallel()
	_, err := e.GetBlockTrade(t.Context(), "")
	require.ErrorIs(t, err, errBlockTradeIDRequired, "GetBlockTrade must reject an empty block trade ID")

	skipLiveWithoutCredentials(t)
	id := testBlockTradeID(t)
	result, err := e.GetBlockTrade(t.Context(), id)
	require.NoError(t, err, "GetBlockTrade must not error")
	if mockTests {
		assert.Equal(t, testDocumentedBlockTrade(), result, "GetBlockTrade should decode every field")
	} else {
		assert.Equal(t, id, result.ID, "GetBlockTrade should return the block trade requested")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetBlockTrade(t.Context(), mockBlockTradeID)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetBlockTrade should reject a null result")
	_, err = newUnavailableServerExchange(t).GetBlockTrade(t.Context(), mockBlockTradeID)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetBlockTrade should return the request failure")
}

func TestGetBlockTradeRequests(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	// Only broker clients can filter by broker code, so live tests request every block trade request
	var brokerCode string
	if mockTests {
		brokerCode = mockBlockTradeBrokerCode
	}
	result, err := e.GetBlockTradeRequests(t.Context(), brokerCode)
	require.NoError(t, err, "GetBlockTradeRequests must not error")
	if mockTests {
		assert.Equal(t, testDocumentedPendingBlockTrades(), result, "GetBlockTradeRequests should decode every field")
	} else {
		for i := range result {
			assert.NotEmptyf(t, result[i].Nonce, "GetBlockTradeRequests should return the nonce of block trade request %d", i)
		}
	}

	_, err = newUnavailableServerExchange(t).GetBlockTradeRequests(t.Context(), mockBlockTradeBrokerCode)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetBlockTradeRequests should return the request failure")
}

func TestGetBlockTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetBlockTrades(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetBlockTrades must reject a nil request")
	_, err = e.GetBlockTrades(t.Context(), &GetBlockTradesRequest{EndID: "66"})
	require.ErrorIs(t, err, errBlockTradeStartIDRequired, "GetBlockTrades must reject an end ID without a start ID")

	skipLiveWithoutCredentials(t)
	result, err := e.GetBlockTrades(t.Context(), mockGetBlockTradesRequest())
	require.NoError(t, err, "GetBlockTrades must not error")
	if mockTests {
		assert.Equal(t, testDocumentedBlockTrades(), result, "GetBlockTrades should decode every field")
	} else {
		assert.LessOrEqual(t, len(result), 1, "GetBlockTrades should return at most the count requested")
	}

	_, err = newUnavailableServerExchange(t).GetBlockTrades(t.Context(), mockGetBlockTradesRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetBlockTrades should return the request failure")
}

func TestGetBrokerClients(t *testing.T) {
	t.Parallel()
	_, err := e.GetBrokerClients(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetBrokerClients must reject a nil request")

	if !mockTests {
		t.Skip("GetBrokerClients needs a broker account, which live tests do not have")
	}
	result, err := e.GetBrokerClients(t.Context(), &GetBrokerClientsRequest{})
	require.NoError(t, err, "GetBrokerClients must not error")
	assert.Equal(t, testDocumentedBrokerClients(), result, "GetBrokerClients should decode every field")

	_, err = newUnavailableServerExchange(t).GetBrokerClients(t.Context(), &GetBrokerClientsRequest{})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetBrokerClients should return the request failure")
}

func TestGetBrokerTradeRequests(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("GetBrokerTradeRequests needs a broker account, which live tests do not have")
	}
	result, err := e.GetBrokerTradeRequests(t.Context())
	require.NoError(t, err, "GetBrokerTradeRequests must not error")
	assert.Equal(t, testDocumentedPendingBrokerTrades(), result, "GetBrokerTradeRequests should decode every field")

	_, err = newUnavailableServerExchange(t).GetBrokerTradeRequests(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetBrokerTradeRequests should return the request failure")
}

func TestGetBrokerTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetBrokerTrades(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetBrokerTrades must reject a nil request")
	_, err = e.GetBrokerTrades(t.Context(), &GetBrokerTradesRequest{EndID: "BLOCK-7"})
	require.ErrorIs(t, err, errBlockTradeStartIDRequired, "GetBrokerTrades must reject an end ID without a start ID")

	if !mockTests {
		t.Skip("GetBrokerTrades needs a broker account, which live tests do not have")
	}
	result, err := e.GetBrokerTrades(t.Context(), mockGetBrokerTradesRequest())
	require.NoError(t, err, "GetBrokerTrades must not error")
	assert.Equal(t, testDocumentedBrokerTrades(), result, "GetBrokerTrades should decode every field")

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetBrokerTrades(t.Context(), mockGetBrokerTradesRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetBrokerTrades should reject a null result")
	_, err = newUnavailableServerExchange(t).GetBrokerTrades(t.Context(), mockGetBrokerTradesRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetBrokerTrades should return the request failure")
}

func TestInvalidateBlockTradeSignature(t *testing.T) {
	t.Parallel()
	err := e.InvalidateBlockTradeSignature(t.Context(), "")
	require.ErrorIs(t, err, errSignatureRequired, "InvalidateBlockTradeSignature must reject an empty signature")

	skipLiveUnlessCanManipulate(t)
	signature := mockBlockTradeSignature
	if !mockTests {
		verified, err := e.VerifyBlockTrade(t.Context(), liveVerifyBlockTradeRequest(t))
		require.NoError(t, err, "VerifyBlockTrade must sign a block trade to invalidate")
		signature = verified.Signature
	}
	require.NoError(t, e.InvalidateBlockTradeSignature(t.Context(), signature), "InvalidateBlockTradeSignature must not error")

	err = newUnavailableServerExchange(t).InvalidateBlockTradeSignature(t.Context(), mockBlockTradeSignature)
	assert.ErrorIs(t, err, request.ErrBadStatus, "InvalidateBlockTradeSignature should return the request failure")
}

func TestRejectBlockTrade(t *testing.T) {
	t.Parallel()
	err := e.RejectBlockTrade(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "RejectBlockTrade must reject a nil request")
	err = e.RejectBlockTrade(t.Context(), &BlockTradeApprovalRequest{Nonce: "bt-468nha", Role: BlockTradeRoleMaker})
	require.ErrorIs(t, err, errTimestampRequired, "RejectBlockTrade must reject an empty timestamp")

	if !mockTests {
		t.Skip("RejectBlockTrade needs a block trade pending the account's approval, which only a broker can create")
	}
	require.NoError(t, e.RejectBlockTrade(t.Context(), mockBlockTradeApprovalRequest()), "RejectBlockTrade must not error")

	err = newUnavailableServerExchange(t).RejectBlockTrade(t.Context(), mockBlockTradeApprovalRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "RejectBlockTrade should return the request failure")
}

func TestSimulateBlockTrade(t *testing.T) {
	t.Parallel()
	_, err := e.SimulateBlockTrade(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SimulateBlockTrade must reject a nil request")
	_, err = e.SimulateBlockTrade(t.Context(), &SimulateBlockTradeRequest{Role: "broker", Trades: mockBlockTradeLegs()})
	require.ErrorIs(t, err, errInvalidBlockTradeRole, "SimulateBlockTrade must reject an undocumented role")
	_, err = e.SimulateBlockTrade(t.Context(), &SimulateBlockTradeRequest{})
	require.ErrorIs(t, err, errBlockTradeLegsRequired, "SimulateBlockTrade must reject a block trade without legs")
	limited, noDelay := newRateLimitedServerExchange(t, `{"jsonrpc":"2.0","result":true}`)
	_, err = limited.SimulateBlockTrade(noDelay, mockSimulateBlockTradeRequest())
	assert.NoError(t, err, "SimulateBlockTrade should be limited as a non-matching engine request")

	skipLiveWithoutCredentials(t)
	arg := mockSimulateBlockTradeRequest()
	if !mockTests {
		arg = &SimulateBlockTradeRequest{Role: BlockTradeRoleMaker, Trades: liveBlockTradeLegs(t)}
	}
	result, err := e.SimulateBlockTrade(t.Context(), arg)
	require.NoError(t, err, "SimulateBlockTrade must not error")
	if mockTests {
		assert.True(t, result, "SimulateBlockTrade should decode that the block trade can be executed")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).SimulateBlockTrade(t.Context(), mockSimulateBlockTradeRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "SimulateBlockTrade should reject a null result")
	_, err = newUnavailableServerExchange(t).SimulateBlockTrade(t.Context(), mockSimulateBlockTradeRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "SimulateBlockTrade should return the request failure")
}

func TestVerifyBlockTrade(t *testing.T) {
	t.Parallel()
	_, err := e.VerifyBlockTrade(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "VerifyBlockTrade must reject a nil request")
	arg := mockVerifyBlockTradeRequest()
	arg.Timestamp = time.Time{}
	_, err = e.VerifyBlockTrade(t.Context(), arg)
	require.ErrorIs(t, err, errTimestampRequired, "VerifyBlockTrade must reject an empty timestamp")
	arg = mockVerifyBlockTradeRequest()
	arg.Trades[1].Amount = 0
	_, err = e.VerifyBlockTrade(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidAmount, "VerifyBlockTrade must reject a leg without an amount")

	skipLiveUnlessCanManipulate(t)
	arg = mockVerifyBlockTradeRequest()
	if !mockTests {
		arg = liveVerifyBlockTradeRequest(t)
	}
	result, err := e.VerifyBlockTrade(t.Context(), arg)
	require.NoError(t, err, "VerifyBlockTrade must not error")
	if mockTests {
		assert.Equal(t, testDocumentedBlockTradeSignature(), result, "VerifyBlockTrade should decode the signature")
	} else {
		require.NotEmpty(t, result.Signature, "VerifyBlockTrade must return a signature")
		require.NoError(t, e.InvalidateBlockTradeSignature(t.Context(), result.Signature), "InvalidateBlockTradeSignature must cancel the block trade verified")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).VerifyBlockTrade(t.Context(), mockVerifyBlockTradeRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "VerifyBlockTrade should reject a null result")
	_, err = newUnavailableServerExchange(t).VerifyBlockTrade(t.Context(), mockVerifyBlockTradeRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "VerifyBlockTrade should return the request failure")
}

func TestValidateBlockTradeKey(t *testing.T) {
	t.Parallel()
	timestamp := time.UnixMilli(1711468813551)
	for _, tc := range []struct {
		name      string
		timestamp time.Time
		nonce     string
		role      string
		err       error
	}{
		{name: "maker", timestamp: timestamp, nonce: "bt-468nha", role: BlockTradeRoleMaker},
		{name: "taker", timestamp: timestamp, nonce: "bt-468nha", role: BlockTradeRoleTaker},
		{name: "empty timestamp", nonce: "bt-468nha", role: BlockTradeRoleMaker, err: errTimestampRequired},
		{name: "empty nonce", timestamp: timestamp, role: BlockTradeRoleMaker, err: errBlockTradeNonceRequired},
		{name: "empty role", timestamp: timestamp, nonce: "bt-468nha", err: errInvalidBlockTradeRole},
	} {
		assert.ErrorIsf(t, validateBlockTradeKey(tc.timestamp, tc.nonce, tc.role), tc.err, "validateBlockTradeKey should return the expected error for %s", tc.name)
	}
}

func TestValidateBlockTradeRole(t *testing.T) {
	t.Parallel()
	for _, role := range []string{BlockTradeRoleMaker, BlockTradeRoleTaker} {
		assert.NoErrorf(t, validateBlockTradeRole(role), "validateBlockTradeRole should accept role %q", role)
	}
	for _, role := range []string{"", "Maker", "any"} {
		assert.ErrorIsf(t, validateBlockTradeRole(role), errInvalidBlockTradeRole, "validateBlockTradeRole should reject role %q", role)
	}
}

func TestValidateBlockTradeLegs(t *testing.T) {
	t.Parallel()
	assert.NoError(t, validateBlockTradeLegs(mockBlockTradeLegs()), "validateBlockTradeLegs should accept Deribit's documented legs")
	assert.ErrorIs(t, validateBlockTradeLegs(nil), errBlockTradeLegsRequired, "validateBlockTradeLegs should reject a block trade without legs")
	for _, tc := range []struct {
		name string
		leg  BlockTradeLeg
		err  error
	}{
		{name: "empty instrument", leg: BlockTradeLeg{Direction: DirectionBuy, Price: 0.0707, Amount: 1.2}, err: errInstrumentNameRequired},
		{name: "empty direction", leg: BlockTradeLeg{InstrumentName: "BTC-9AUG19-10250-P", Price: 0.0707, Amount: 1.2}, err: errInvalidDirection},
		{name: "capitalised direction", leg: BlockTradeLeg{InstrumentName: "BTC-9AUG19-10250-P", Direction: "Buy", Price: 0.0707, Amount: 1.2}, err: errInvalidDirection},
		{name: "zero amount", leg: BlockTradeLeg{InstrumentName: "BTC-9AUG19-10250-P", Direction: DirectionBuy, Price: 0.0707}, err: errInvalidAmount},
		{name: "negative amount", leg: BlockTradeLeg{InstrumentName: "BTC-9AUG19-10250-P", Direction: DirectionSell, Price: 0.0707, Amount: -1.2}, err: errInvalidAmount},
	} {
		legs := []BlockTradeLeg{mockBlockTradeLegs()[0], tc.leg}
		assert.ErrorIsf(t, validateBlockTradeLegs(legs), tc.err, "validateBlockTradeLegs should return the expected error for a second leg with %s", tc.name)
	}
}
