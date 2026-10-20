package deribit

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const (
	// mockCancelBlockRFQID is the Block RFQ of Deribit's documented private/cancel_block_rfq example
	mockCancelBlockRFQID = 366
	// mockCancelBlockRFQTriggerID is the Block RFQ of Deribit's documented private/cancel_block_rfq_trigger example
	mockCancelBlockRFQTriggerID = 123
	// mockBlockRFQQuoteLabel is the label of Deribit's documented Block RFQ quote examples
	mockBlockRFQQuoteLabel = "example_quote"
	// mockBlockRFQTradesContinuation selects the recorded page of BTC Block RFQ trades, which holds a hedged trade, a
	// three-leg structure without a combo, a leg ratio of 2 and negative prices
	mockBlockRFQTradesContinuation = "1792088246831:57979"
)

// mockBlockRFQHedge returns the hedge leg of Deribit's documented Block RFQ examples
func mockBlockRFQHedge() *BlockRFQHedge {
	return &BlockRFQHedge{InstrumentName: "BTC-PERPETUAL", Direction: DirectionBuy, Price: 70000, Amount: 10}
}

// mockAcceptBlockRFQRequest returns the request of Deribit's documented private/accept_block_rfq example
func mockAcceptBlockRFQRequest() *AcceptBlockRFQRequest {
	return &AcceptBlockRFQRequest{
		BlockRFQID: 1,
		Legs: []BlockRFQLeg{
			{InstrumentName: "BTC-8NOV24-70000-C", Direction: DirectionBuy, Ratio: 1},
			{InstrumentName: "BTC-8NOV24-72000-C", Direction: DirectionSell, Ratio: 1},
		},
		Price:       0.01,
		Amount:      100,
		Direction:   DirectionBuy,
		TimeInForce: BlockRFQTimeInForceFillOrKill,
	}
}

// mockAddBlockRFQQuoteRequest returns the request of Deribit's documented private/add_block_rfq_quote example
func mockAddBlockRFQQuoteRequest() *AddBlockRFQQuoteRequest {
	return &AddBlockRFQQuoteRequest{
		BlockRFQID:           3,
		Amount:               10000,
		Direction:            DirectionBuy,
		Legs:                 []BlockRFQLeg{{InstrumentName: "BTC-15NOV24", Direction: DirectionBuy, Ratio: 1, Price: 69600}},
		Hedge:                mockBlockRFQHedge(),
		Label:                mockBlockRFQQuoteLabel,
		ExecutionInstruction: BlockRFQExecutionAnyPartOf,
		ExpiresAt:            time.UnixMilli(1745312540321),
	}
}

// mockCancelBlockRFQQuoteRequest returns the request of Deribit's documented private/cancel_block_rfq_quote example
func mockCancelBlockRFQQuoteRequest() *CancelBlockRFQQuoteRequest {
	return &CancelBlockRFQQuoteRequest{BlockRFQID: 3, Label: mockBlockRFQQuoteLabel}
}

// mockCreateBlockRFQRequest returns the request of Deribit's documented private/create_block_rfq example
func mockCreateBlockRFQRequest() *CreateBlockRFQRequest {
	return &CreateBlockRFQRequest{
		Legs:   []CreateBlockRFQLeg{{InstrumentName: "BTC-15NOV24", Amount: 20000, Direction: DirectionSell}},
		Hedge:  mockBlockRFQHedge(),
		Label:  "example",
		Makers: []string{"MAKER1"},
	}
}

// mockEditBlockRFQQuoteRequest returns the request of Deribit's documented private/edit_block_rfq_quote example,
// without the direction and execution instruction the example sends, which Deribit does not document as params
func mockEditBlockRFQQuoteRequest() *EditBlockRFQQuoteRequest {
	return &EditBlockRFQQuoteRequest{
		BlockRFQID: 3,
		Label:      mockBlockRFQQuoteLabel,
		Amount:     20000,
		Legs:       []BlockRFQLeg{{InstrumentName: "BTC-15NOV24", Direction: DirectionBuy, Ratio: 1, Price: 74600}},
		Hedge:      mockBlockRFQHedge(),
	}
}

// mockGetBlockRFQsRequest returns the request of Deribit's documented private/get_block_rfqs example, which also works
// live
func mockGetBlockRFQsRequest() *GetBlockRFQsRequest {
	return &GetBlockRFQsRequest{Count: 20, State: BlockRFQStateOpen, Role: BlockRFQRoleMaker}
}

// mockGetBlockRFQTradesRequest returns the request of the recorded page of BTC Block RFQ trades
func mockGetBlockRFQTradesRequest() *GetBlockRFQTradesRequest {
	return &GetBlockRFQTradesRequest{Currency: currency.BTC, Continuation: mockBlockRFQTradesContinuation, Count: 10}
}

// testGetBlockRFQTradesRequest returns the recorded page's request, or live, the latest page's
func testGetBlockRFQTradesRequest() *GetBlockRFQTradesRequest {
	if mockTests {
		return mockGetBlockRFQTradesRequest()
	}
	return &GetBlockRFQTradesRequest{Currency: currency.BTC, Count: 10}
}

// liveCreateBlockRFQRequest returns a Block RFQ of the minimum block trade amount of a perpetual, which live tests
// create and cancel
func liveCreateBlockRFQRequest() *CreateBlockRFQRequest {
	return &CreateBlockRFQRequest{
		Legs:  []CreateBlockRFQLeg{{InstrumentName: liveBlockTradeInstrument, Amount: liveBlockTradeAmount, Direction: DirectionBuy}},
		Label: "gct-test",
	}
}

// testDocumentedAcceptedBlockRFQ returns the block trade of Deribit's documented private/accept_block_rfq example
func testDocumentedAcceptedBlockRFQ() *AcceptBlockRFQResponse {
	trade := UserTrade{
		TradeID:            "771",
		TradeSequence:      113,
		InstrumentName:     "BTC-8NOV24-70000-C",
		Timestamp:          types.Time(time.UnixMilli(1730798381502)),
		OrderType:          "limit",
		OrderID:            "2899",
		Direction:          DirectionBuy,
		TickDirection:      2,
		IndexPrice:         70000,
		Price:              69696.8,
		Amount:             100,
		Contracts:          100,
		Liquidity:          "T",
		Fee:                0.00000015,
		FeeCurrency:        currency.BTC,
		State:              "filled",
		BlockTradeID:       "BLOCK-423",
		BlockTradeLegCount: 2,
		BlockRFQID:         1,
		API:                true,
		MarkPrice:          0.03,
		ComboID:            "BTC-CS-8NOV24-70000_72000",
	}
	secondLeg := trade
	secondLeg.TradeID = "772"
	secondLeg.InstrumentName = "BTC-8NOV24-72000-C"
	secondLeg.OrderID = "2900"
	secondLeg.Direction = DirectionSell
	secondLeg.Price = 69677.4
	secondLeg.MarkPrice = 0.02
	return &AcceptBlockRFQResponse{
		BlockTrades: []BlockTrade{{
			ID:        "BLOCK-423",
			Timestamp: types.Time(time.UnixMilli(1730798381504)),
			Trades:    []UserTrade{trade, secondLeg},
		}},
	}
}

// testDocumentedBlockRFQQuote returns the quote of Deribit's documented private/add_block_rfq_quote example
func testDocumentedBlockRFQQuote() *BlockRFQQuote {
	return &BlockRFQQuote{
		BlockRFQQuoteID:     8,
		BlockRFQID:          3,
		CreationTimestamp:   types.Time(time.UnixMilli(1731076586371)),
		LastUpdateTimestamp: types.Time(time.UnixMilli(1731076586371)),
		QuoteState:          "open",
		Price:               69600,
		Amount:              10000,
		Direction:           DirectionBuy,
		Legs:                []BlockRFQLeg{{InstrumentName: "BTC-15NOV24", Direction: DirectionBuy, Ratio: 1, Price: 69600}},
		Hedge:               mockBlockRFQHedge(),
		Label:               mockBlockRFQQuoteLabel,
	}
}

// testDocumentedCancelledBlockRFQ returns the Block RFQ of Deribit's documented private/cancel_block_rfq example
func testDocumentedCancelledBlockRFQ() *BlockRFQ {
	return &BlockRFQ{
		BlockRFQID:          mockCancelBlockRFQID,
		CreationTimestamp:   types.Time(time.UnixMilli(1729855159611)),
		ExpirationTimestamp: types.Time(time.UnixMilli(1729855459611)),
		Role:                BlockRFQRoleTaker,
		State:               BlockRFQStateCancelled,
		Makers:              []string{},
		Amount:              100000,
		Asks:                []BlockRFQPriceLevel{},
		Bids:                []BlockRFQPriceLevel{},
		Legs: []BlockRFQLeg{
			{InstrumentName: "BTC-1NOV24", Direction: DirectionSell, Ratio: 1},
			{InstrumentName: "BTC-PERPETUAL", Direction: DirectionBuy, Ratio: 1},
		},
		ComboID: "BTC-FS-1NOV24_PERP",
		Label:   "example",
	}
}

// testDocumentedCancelledBlockRFQQuote returns the quote of Deribit's documented private/cancel_block_rfq_quote example
func testDocumentedCancelledBlockRFQQuote() *BlockRFQQuote {
	return &BlockRFQQuote{
		BlockRFQQuoteID:     8,
		BlockRFQID:          3,
		CreationTimestamp:   types.Time(time.UnixMilli(1731076586371)),
		LastUpdateTimestamp: types.Time(time.UnixMilli(1731076655746)),
		QuoteState:          "cancelled",
		Price:               74600,
		Amount:              20000,
		Direction:           DirectionBuy,
		Legs:                []BlockRFQLeg{{InstrumentName: "BTC-15NOV24", Direction: DirectionBuy, Ratio: 1, Price: 74600}},
		Hedge:               mockBlockRFQHedge(),
		Label:               mockBlockRFQQuoteLabel,
	}
}

// testDocumentedBlockRFQWithCancelledTrigger returns the Block RFQ of Deribit's documented
// private/cancel_block_rfq_trigger example
func testDocumentedBlockRFQWithCancelledTrigger() *BlockRFQ {
	return &BlockRFQ{
		BlockRFQID:          mockCancelBlockRFQTriggerID,
		CreationTimestamp:   types.Time(time.UnixMilli(1730798381504)),
		ExpirationTimestamp: types.Time(time.UnixMilli(1730798481504)),
		Role:                BlockRFQRoleTaker,
		State:               BlockRFQStateOpen,
		TakerRating:         "1-2",
		Makers:              []string{"maker1", "maker2"},
		Amount:              100,
		MinTradeAmount:      10,
		Legs: []BlockRFQLeg{
			{InstrumentName: "BTC-8NOV24-70000-C", Ratio: 1},
			{InstrumentName: "BTC-8NOV24-72000-C", Ratio: 1},
		},
		ComboID:      "BTC-CS-8NOV24-70000_72000",
		Label:        "My Block RFQ",
		AppName:      "Example Application",
		MarkPrice:    0.025,
		TradeTrigger: &BlockRFQTradeTrigger{State: "cancelled", Price: 0.01, Direction: DirectionBuy, CancelReason: "User cancelled"},
	}
}

// testDocumentedCreatedBlockRFQ returns the Block RFQ of Deribit's documented private/create_block_rfq example
func testDocumentedCreatedBlockRFQ() *BlockRFQ {
	return &BlockRFQ{
		BlockRFQID:          507,
		CreationTimestamp:   types.Time(time.UnixMilli(1731062187555)),
		ExpirationTimestamp: types.Time(time.UnixMilli(1731062487555)),
		Role:                BlockRFQRoleTaker,
		State:               "created",
		Makers:              []string{"MAKER1"},
		Amount:              20000,
		Asks:                []BlockRFQPriceLevel{},
		Bids:                []BlockRFQPriceLevel{},
		Legs:                []BlockRFQLeg{{InstrumentName: "BTC-15NOV24", Direction: DirectionSell, Ratio: 1}},
		Hedge:               mockBlockRFQHedge(),
		ComboID:             "BTC-15NOV24",
		Label:               "example",
	}
}

// testDocumentedEditedBlockRFQQuote returns the quote of Deribit's documented private/edit_block_rfq_quote example
func testDocumentedEditedBlockRFQQuote() *BlockRFQQuote {
	return &BlockRFQQuote{
		BlockRFQQuoteID:     8,
		BlockRFQID:          3,
		CreationTimestamp:   types.Time(time.UnixMilli(1731076586371)),
		LastUpdateTimestamp: types.Time(time.UnixMilli(1731076638591)),
		QuoteState:          "open",
		Price:               74600,
		Amount:              20000,
		Direction:           DirectionBuy,
		Legs:                []BlockRFQLeg{{InstrumentName: "BTC-15NOV24", Direction: DirectionBuy, Ratio: 1, Price: 74600}},
		Hedge:               mockBlockRFQHedge(),
		Replaced:            true,
		Label:               mockBlockRFQQuoteLabel,
	}
}

// testDocumentedBlockRFQQuotes returns the quotes of Deribit's documented private/get_block_rfq_quotes example
func testDocumentedBlockRFQQuotes() []BlockRFQQuote {
	return []BlockRFQQuote{{
		BlockRFQQuoteID:      8,
		BlockRFQID:           1,
		CreationTimestamp:    types.Time(time.UnixMilli(1731076586371)),
		LastUpdateTimestamp:  types.Time(time.UnixMilli(1731076638591)),
		QuoteState:           "open",
		ExecutionInstruction: BlockRFQExecutionAllOrNone,
		Price:                74600,
		Amount:               20000,
		Direction:            DirectionBuy,
		Legs:                 []BlockRFQLeg{{InstrumentName: "BTC-15NOV24", Direction: DirectionBuy, Ratio: 1, Price: 74600}},
		Hedge:                mockBlockRFQHedge(),
		Label:                mockBlockRFQQuoteLabel,
	}}
}

// testDocumentedBlockRFQUserInfo returns the identities of Deribit's documented private/get_block_rfq_user_info example
func testDocumentedBlockRFQUserInfo() *BlockRFQUserInfoResponse {
	return &BlockRFQUserInfoResponse{
		Parent: BlockRFQParentIdentity{Identity: "MAKER1", IsMaker: true},
		Users: []BlockRFQUserIdentity{
			{UserID: 1, TakerRating: 98.5, Identity: "TAKER1"},
			{UserID: 2, TakerRating: 97},
		},
	}
}

// testDocumentedBlockRFQs returns the Block RFQs of Deribit's documented private/get_block_rfqs example
func testDocumentedBlockRFQs() *BlockRFQsResponse {
	return &BlockRFQsResponse{
		BlockRFQs: []BlockRFQ{{
			BlockRFQID:          508,
			CreationTimestamp:   types.Time(time.UnixMilli(1731062457741)),
			ExpirationTimestamp: types.Time(time.UnixMilli(1731062757741)),
			Role:                BlockRFQRoleMaker,
			State:               BlockRFQStateOpen,
			TakerRating:         "1-2",
			Amount:              40000,
			Legs:                []BlockRFQLeg{{InstrumentName: "BTC-15NOV24", Direction: DirectionSell, Ratio: 1}},
			Hedge:               mockBlockRFQHedge(),
			ComboID:             "BTC-15NOV24",
		}},
	}
}

// testRecordedBlockRFQTrades returns the recorded page of BTC Block RFQ trades, trimmed to five
func testRecordedBlockRFQTrades() *BlockRFQTradesResponse {
	return &BlockRFQTradesResponse{
		BlockRFQs: []BlockRFQTrade{
			{
				ID:        57976,
				Timestamp: types.Time(time.UnixMilli(1792082270445)),
				Direction: DirectionBuy,
				Amount:    50,
				MarkPrice: 0.007482373,
				Legs: []BlockRFQLeg{
					{InstrumentName: "BTC-27NOV26-82000-C", Direction: DirectionBuy, Ratio: 1, Price: 0.054},
					{InstrumentName: "BTC-27NOV26-82000-P", Direction: DirectionSell, Ratio: 1, Price: 0.0472},
				},
				ComboID:     "BTC-REV-27NOV26-82000",
				Hedge:       &BlockRFQHedge{InstrumentName: "BTC-PERPETUAL", Direction: DirectionSell, Price: 82000, Amount: 4100000},
				IndexPrices: map[string]float64{"btc_usd": 82060.01},
				Trades:      []BlockRFQFill{{Direction: DirectionBuy, Price: 0.0068, Amount: 50, HedgeAmount: 4100000}},
			},
			{
				ID:        57975,
				Timestamp: types.Time(time.UnixMilli(1792082170454)),
				Direction: DirectionSell,
				Amount:    25,
				MarkPrice: 0.006155162,
				Legs: []BlockRFQLeg{
					{InstrumentName: "BTC-23OCT26-76000-P", Direction: DirectionBuy, Ratio: 1, Price: 0.003},
					{InstrumentName: "BTC-23OCT26-87000-C", Direction: DirectionBuy, Ratio: 1, Price: 0.0028},
				},
				ComboID:     "BTC-STRG-23OCT26-76000_87000",
				IndexPrices: map[string]float64{"btc_usd": 82007.4},
				Trades:      []BlockRFQFill{{Direction: DirectionSell, Price: 0.0058, Amount: 25}},
			},
			{
				ID:        57973,
				Timestamp: types.Time(time.UnixMilli(1792081973852)),
				Direction: DirectionSell,
				Amount:    25,
				MarkPrice: 0.00698348,
				Legs: []BlockRFQLeg{
					{InstrumentName: "BTC-30OCT26-72000-P", Direction: DirectionBuy, Ratio: 1, Price: 0.0035},
					{InstrumentName: "BTC-30OCT26-90000-C", Direction: DirectionBuy, Ratio: 1, Price: 0.0032},
				},
				ComboID:     "BTC-STRG-30OCT26-72000_90000",
				IndexPrices: map[string]float64{"btc_usd": 81911.48},
				Trades:      []BlockRFQFill{{Direction: DirectionSell, Price: 0.0067, Amount: 25}},
			},
			{
				ID:        57968,
				Timestamp: types.Time(time.UnixMilli(1792078157717)),
				Direction: DirectionBuy,
				Amount:    18,
				MarkPrice: -0.005392112,
				Legs: []BlockRFQLeg{
					{InstrumentName: "BTC-23OCT26-83000-C", Direction: DirectionBuy, Ratio: 1, Price: 0.0161},
					{InstrumentName: "BTC-6NOV26-88000-C", Direction: DirectionSell, Ratio: 1, Price: 0.0128},
					{InstrumentName: "BTC-6NOV26-90000-C", Direction: DirectionSell, Ratio: 1, Price: 0.0085},
				},
				IndexPrices: map[string]float64{"btc_usd": 82258.65},
				Trades:      []BlockRFQFill{{Direction: DirectionBuy, Price: -0.0052, Amount: 18}},
			},
			{
				ID:        57967,
				Timestamp: types.Time(time.UnixMilli(1792078021311)),
				Direction: DirectionBuy,
				Amount:    100,
				MarkPrice: -0.001621056,
				Legs: []BlockRFQLeg{
					{InstrumentName: "BTC-27NOV26-85000-C", Direction: DirectionBuy, Ratio: 1, Price: 0.0395},
					{InstrumentName: "BTC-27NOV26-90000-C", Direction: DirectionSell, Ratio: 2, Price: 0.0203},
				},
				ComboID:     "BTC-CSR12-27NOV26-85000_90000",
				IndexPrices: map[string]float64{"btc_usd": 82113.61},
				Trades:      []BlockRFQFill{{Direction: DirectionBuy, Price: -0.0011, Amount: 100}},
			},
		},
		Continuation: "1792076898602:57958",
	}
}

func TestAcceptBlockRFQ(t *testing.T) {
	t.Parallel()
	_, err := e.AcceptBlockRFQ(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "AcceptBlockRFQ must reject a nil request")
	arg := mockAcceptBlockRFQRequest()
	arg.BlockRFQID = 0
	_, err = e.AcceptBlockRFQ(t.Context(), arg)
	require.ErrorIs(t, err, errBlockRFQIDRequired, "AcceptBlockRFQ must reject an empty Block RFQ ID")
	arg = mockAcceptBlockRFQRequest()
	arg.Legs[0].Ratio = 0
	_, err = e.AcceptBlockRFQ(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidBlockRFQLegRatio, "AcceptBlockRFQ must reject a leg without a ratio")
	arg = mockAcceptBlockRFQRequest()
	arg.Amount = 0
	_, err = e.AcceptBlockRFQ(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidAmount, "AcceptBlockRFQ must reject an empty amount")
	arg = mockAcceptBlockRFQRequest()
	arg.Direction = "long"
	_, err = e.AcceptBlockRFQ(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidDirection, "AcceptBlockRFQ must reject an undocumented direction")
	arg = mockAcceptBlockRFQRequest()
	arg.TimeInForce = "immediate_or_cancel"
	_, err = e.AcceptBlockRFQ(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidBlockRFQTimeInForce, "AcceptBlockRFQ must reject a time in force Deribit does not accept for a Block RFQ")
	arg = mockAcceptBlockRFQRequest()
	arg.Hedge = &BlockRFQHedge{InstrumentName: "BTC-PERPETUAL", Direction: DirectionBuy, Price: 70000}
	_, err = e.AcceptBlockRFQ(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidAmount, "AcceptBlockRFQ must reject a hedge without an amount")

	if !mockTests {
		t.Skip("AcceptBlockRFQ needs makers' quotes on the account's Block RFQ, which live tests cannot arrange")
	}
	result, err := e.AcceptBlockRFQ(t.Context(), mockAcceptBlockRFQRequest())
	require.NoError(t, err, "AcceptBlockRFQ must not error")
	assert.Equal(t, testDocumentedAcceptedBlockRFQ(), result, "AcceptBlockRFQ should decode every field")

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).AcceptBlockRFQ(t.Context(), mockAcceptBlockRFQRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "AcceptBlockRFQ should reject a null result")
	_, err = newUnavailableServerExchange(t).AcceptBlockRFQ(t.Context(), mockAcceptBlockRFQRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "AcceptBlockRFQ should return the request failure")
}

func TestAddBlockRFQQuote(t *testing.T) {
	t.Parallel()
	_, err := e.AddBlockRFQQuote(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "AddBlockRFQQuote must reject a nil request")
	arg := mockAddBlockRFQQuoteRequest()
	arg.BlockRFQID = 0
	_, err = e.AddBlockRFQQuote(t.Context(), arg)
	require.ErrorIs(t, err, errBlockRFQIDRequired, "AddBlockRFQQuote must reject an empty Block RFQ ID")
	arg = mockAddBlockRFQQuoteRequest()
	arg.Amount = -1
	_, err = e.AddBlockRFQQuote(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidAmount, "AddBlockRFQQuote must reject a negative amount")
	arg = mockAddBlockRFQQuoteRequest()
	arg.Direction = ""
	_, err = e.AddBlockRFQQuote(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidDirection, "AddBlockRFQQuote must reject an empty direction")
	arg = mockAddBlockRFQQuoteRequest()
	arg.Legs = nil
	_, err = e.AddBlockRFQQuote(t.Context(), arg)
	require.ErrorIs(t, err, errBlockRFQLegsRequired, "AddBlockRFQQuote must reject a quote without legs")
	arg = mockAddBlockRFQQuoteRequest()
	arg.Hedge.InstrumentName = ""
	_, err = e.AddBlockRFQQuote(t.Context(), arg)
	require.ErrorIs(t, err, errInstrumentNameRequired, "AddBlockRFQQuote must reject a hedge without an instrument")
	arg = mockAddBlockRFQQuoteRequest()
	arg.Label = strings.Repeat("a", maximumBlockRFQLabelLength+1)
	_, err = e.AddBlockRFQQuote(t.Context(), arg)
	require.ErrorIs(t, err, errBlockRFQLabelTooLong, "AddBlockRFQQuote must reject a label over 64 characters")
	arg = mockAddBlockRFQQuoteRequest()
	arg.ExecutionInstruction = "fill_or_kill"
	_, err = e.AddBlockRFQQuote(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidBlockRFQExecutionInstruction, "AddBlockRFQQuote must reject an undocumented execution instruction")

	if !mockTests {
		t.Skip("AddBlockRFQQuote needs another account's open Block RFQ to quote, which live tests cannot arrange")
	}
	result, err := e.AddBlockRFQQuote(t.Context(), mockAddBlockRFQQuoteRequest())
	require.NoError(t, err, "AddBlockRFQQuote must not error")
	assert.Equal(t, testDocumentedBlockRFQQuote(), result, "AddBlockRFQQuote should decode every field")

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).AddBlockRFQQuote(t.Context(), mockAddBlockRFQQuoteRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "AddBlockRFQQuote should reject a null result")
	_, err = newUnavailableServerExchange(t).AddBlockRFQQuote(t.Context(), mockAddBlockRFQQuoteRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "AddBlockRFQQuote should return the request failure")
}

func TestCancelAllBlockRFQQuotes(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllBlockRFQQuotes(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelAllBlockRFQQuotes must reject a nil request")

	skipLiveUnlessCanManipulate(t)
	// Live, the account's open Block RFQ quotes, if any, are cancelled
	arg := &CancelAllBlockRFQQuotesRequest{}
	if mockTests {
		arg.BlockRFQID = 154
	}
	result, err := e.CancelAllBlockRFQQuotes(t.Context(), arg)
	require.NoError(t, err, "CancelAllBlockRFQQuotes must not error")
	if mockTests {
		assert.Equal(t, &CancelAllBlockRFQQuotesResponse{CancelledCount: 1}, result, "CancelAllBlockRFQQuotes should decode the count of quotes cancelled")
	} else {
		assert.Empty(t, result.DetailedReports, "CancelAllBlockRFQQuotes should return a count without a detailed request")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).CancelAllBlockRFQQuotes(t.Context(), arg)
	assert.ErrorIs(t, err, common.ErrNoResponse, "CancelAllBlockRFQQuotes should reject a null result")
	_, err = newUnavailableServerExchange(t).CancelAllBlockRFQQuotes(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "CancelAllBlockRFQQuotes should return the request failure")
}

func TestCancelBlockRFQ(t *testing.T) {
	t.Parallel()
	_, err := e.CancelBlockRFQ(t.Context(), 0)
	require.ErrorIs(t, err, errBlockRFQIDRequired, "CancelBlockRFQ must reject an empty Block RFQ ID")

	skipLiveUnlessCanManipulate(t)
	blockRFQID := uint64(mockCancelBlockRFQID)
	if !mockTests {
		created, err := e.CreateBlockRFQ(t.Context(), liveCreateBlockRFQRequest())
		require.NoError(t, err, "CreateBlockRFQ must create a Block RFQ to cancel")
		blockRFQID = created.BlockRFQID
	}
	result, err := e.CancelBlockRFQ(t.Context(), blockRFQID)
	require.NoError(t, err, "CancelBlockRFQ must not error")
	if mockTests {
		assert.Equal(t, testDocumentedCancelledBlockRFQ(), result, "CancelBlockRFQ should decode every field")
	} else {
		assert.Equal(t, BlockRFQStateCancelled, result.State, "CancelBlockRFQ should cancel the Block RFQ")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).CancelBlockRFQ(t.Context(), mockCancelBlockRFQID)
	assert.ErrorIs(t, err, common.ErrNoResponse, "CancelBlockRFQ should reject a null result")
	_, err = newUnavailableServerExchange(t).CancelBlockRFQ(t.Context(), mockCancelBlockRFQID)
	assert.ErrorIs(t, err, request.ErrBadStatus, "CancelBlockRFQ should return the request failure")
}

func TestCancelBlockRFQQuote(t *testing.T) {
	t.Parallel()
	_, err := e.CancelBlockRFQQuote(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelBlockRFQQuote must reject a nil request")
	_, err = e.CancelBlockRFQQuote(t.Context(), &CancelBlockRFQQuoteRequest{Label: mockBlockRFQQuoteLabel})
	require.ErrorIs(t, err, errBlockRFQQuoteIdentifierRequired, "CancelBlockRFQQuote must reject a label without its Block RFQ")

	if !mockTests {
		t.Skip("CancelBlockRFQQuote needs a quote on another account's open Block RFQ, which live tests cannot arrange")
	}
	result, err := e.CancelBlockRFQQuote(t.Context(), mockCancelBlockRFQQuoteRequest())
	require.NoError(t, err, "CancelBlockRFQQuote must not error")
	assert.Equal(t, testDocumentedCancelledBlockRFQQuote(), result, "CancelBlockRFQQuote should decode every field")

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).CancelBlockRFQQuote(t.Context(), mockCancelBlockRFQQuoteRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "CancelBlockRFQQuote should reject a null result")
	_, err = newUnavailableServerExchange(t).CancelBlockRFQQuote(t.Context(), mockCancelBlockRFQQuoteRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "CancelBlockRFQQuote should return the request failure")
}

func TestCancelBlockRFQTrigger(t *testing.T) {
	t.Parallel()
	_, err := e.CancelBlockRFQTrigger(t.Context(), 0)
	require.ErrorIs(t, err, errBlockRFQIDRequired, "CancelBlockRFQTrigger must reject an empty Block RFQ ID")

	if !mockTests {
		t.Skip("CancelBlockRFQTrigger needs an active trade trigger, which needs makers' quotes live tests cannot arrange")
	}
	result, err := e.CancelBlockRFQTrigger(t.Context(), mockCancelBlockRFQTriggerID)
	require.NoError(t, err, "CancelBlockRFQTrigger must not error")
	assert.Equal(t, testDocumentedBlockRFQWithCancelledTrigger(), result, "CancelBlockRFQTrigger should decode every field")

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).CancelBlockRFQTrigger(t.Context(), mockCancelBlockRFQTriggerID)
	assert.ErrorIs(t, err, common.ErrNoResponse, "CancelBlockRFQTrigger should reject a null result")
	_, err = newUnavailableServerExchange(t).CancelBlockRFQTrigger(t.Context(), mockCancelBlockRFQTriggerID)
	assert.ErrorIs(t, err, request.ErrBadStatus, "CancelBlockRFQTrigger should return the request failure")
}

func TestCreateBlockRFQ(t *testing.T) {
	t.Parallel()
	_, err := e.CreateBlockRFQ(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateBlockRFQ must reject a nil request")
	_, err = e.CreateBlockRFQ(t.Context(), &CreateBlockRFQRequest{})
	require.ErrorIs(t, err, errBlockRFQLegsRequired, "CreateBlockRFQ must reject a Block RFQ without legs")
	arg := mockCreateBlockRFQRequest()
	arg.TradeAllocations = []BlockRFQTradeAllocation{{Amount: 10000}}
	_, err = e.CreateBlockRFQ(t.Context(), arg)
	require.ErrorIs(t, err, errBlockRFQAllocationTargetRequired, "CreateBlockRFQ must reject an allocation without a user or client")
	arg = mockCreateBlockRFQRequest()
	arg.Hedge.Direction = "short"
	_, err = e.CreateBlockRFQ(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidDirection, "CreateBlockRFQ must reject a hedge with an undocumented direction")
	arg = mockCreateBlockRFQRequest()
	arg.Label = strings.Repeat("a", maximumBlockRFQLabelLength+1)
	_, err = e.CreateBlockRFQ(t.Context(), arg)
	require.ErrorIs(t, err, errBlockRFQLabelTooLong, "CreateBlockRFQ must reject a label over 64 characters")

	skipLiveUnlessCanManipulate(t)
	arg = mockCreateBlockRFQRequest()
	if !mockTests {
		arg = liveCreateBlockRFQRequest()
	}
	result, err := e.CreateBlockRFQ(t.Context(), arg)
	require.NoError(t, err, "CreateBlockRFQ must not error")
	if mockTests {
		assert.Equal(t, testDocumentedCreatedBlockRFQ(), result, "CreateBlockRFQ should decode every field")
	} else {
		_, err = e.CancelBlockRFQ(t.Context(), result.BlockRFQID)
		require.NoError(t, err, "CancelBlockRFQ must cancel the Block RFQ created")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).CreateBlockRFQ(t.Context(), mockCreateBlockRFQRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "CreateBlockRFQ should reject a null result")
	_, err = newUnavailableServerExchange(t).CreateBlockRFQ(t.Context(), mockCreateBlockRFQRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "CreateBlockRFQ should return the request failure")
}

func TestEditBlockRFQQuote(t *testing.T) {
	t.Parallel()
	_, err := e.EditBlockRFQQuote(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "EditBlockRFQQuote must reject a nil request")
	arg := mockEditBlockRFQQuoteRequest()
	arg.Label = ""
	_, err = e.EditBlockRFQQuote(t.Context(), arg)
	require.ErrorIs(t, err, errBlockRFQQuoteIdentifierRequired, "EditBlockRFQQuote must reject a Block RFQ without the quote's label")
	arg = mockEditBlockRFQQuoteRequest()
	arg.Amount = 0
	_, err = e.EditBlockRFQQuote(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidAmount, "EditBlockRFQQuote must reject an empty amount")
	arg = mockEditBlockRFQQuoteRequest()
	arg.Legs[0].InstrumentName = ""
	_, err = e.EditBlockRFQQuote(t.Context(), arg)
	require.ErrorIs(t, err, errInstrumentNameRequired, "EditBlockRFQQuote must reject a leg without an instrument")
	arg = mockEditBlockRFQQuoteRequest()
	arg.Hedge.Amount = -10
	_, err = e.EditBlockRFQQuote(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidAmount, "EditBlockRFQQuote must reject a hedge with a negative amount")

	if !mockTests {
		t.Skip("EditBlockRFQQuote needs a quote on another account's open Block RFQ, which live tests cannot arrange")
	}
	result, err := e.EditBlockRFQQuote(t.Context(), mockEditBlockRFQQuoteRequest())
	require.NoError(t, err, "EditBlockRFQQuote must not error")
	assert.Equal(t, testDocumentedEditedBlockRFQQuote(), result, "EditBlockRFQQuote should decode every field")

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).EditBlockRFQQuote(t.Context(), mockEditBlockRFQQuoteRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "EditBlockRFQQuote should reject a null result")
	_, err = newUnavailableServerExchange(t).EditBlockRFQQuote(t.Context(), mockEditBlockRFQQuoteRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "EditBlockRFQQuote should return the request failure")
}

func TestGetBlockRFQMakers(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	result, err := e.GetBlockRFQMakers(t.Context())
	require.NoError(t, err, "GetBlockRFQMakers must not error")
	if mockTests {
		assert.Equal(t, []string{"MAKER1", "MAKER2", "MAKER3"}, result, "GetBlockRFQMakers should decode the makers' aliases")
	} else {
		assert.NotEmpty(t, result, "GetBlockRFQMakers should return the makers' aliases")
	}

	_, err = newUnavailableServerExchange(t).GetBlockRFQMakers(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetBlockRFQMakers should return the request failure")
}

func TestGetBlockRFQQuotes(t *testing.T) {
	t.Parallel()
	_, err := e.GetBlockRFQQuotes(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetBlockRFQQuotes must reject a nil request")

	skipLiveWithoutCredentials(t)
	arg := &GetBlockRFQQuotesRequest{}
	if mockTests {
		arg.BlockRFQID = 1
	}
	result, err := e.GetBlockRFQQuotes(t.Context(), arg)
	require.NoError(t, err, "GetBlockRFQQuotes must not error")
	if mockTests {
		assert.Equal(t, testDocumentedBlockRFQQuotes(), result, "GetBlockRFQQuotes should decode every field")
	} else {
		for i := range result {
			assert.NotZerof(t, result[i].BlockRFQQuoteID, "GetBlockRFQQuotes should return the ID of quote %d", i)
		}
	}

	_, err = newUnavailableServerExchange(t).GetBlockRFQQuotes(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetBlockRFQQuotes should return the request failure")
}

func TestGetBlockRFQUserInfo(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	result, err := e.GetBlockRFQUserInfo(t.Context())
	require.NoError(t, err, "GetBlockRFQUserInfo must not error")
	if mockTests {
		assert.Equal(t, testDocumentedBlockRFQUserInfo(), result, "GetBlockRFQUserInfo should decode every field")
	} else {
		assert.NotEmpty(t, result.Users, "GetBlockRFQUserInfo should return the account's users")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetBlockRFQUserInfo(t.Context())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetBlockRFQUserInfo should reject a null result")
	_, err = newUnavailableServerExchange(t).GetBlockRFQUserInfo(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetBlockRFQUserInfo should return the request failure")
}

func TestGetBlockRFQs(t *testing.T) {
	t.Parallel()
	_, err := e.GetBlockRFQs(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetBlockRFQs must reject a nil request")
	_, err = e.GetBlockRFQs(t.Context(), &GetBlockRFQsRequest{State: "created"})
	require.ErrorIs(t, err, errInvalidBlockRFQState, "GetBlockRFQs must reject a state Deribit does not filter by")
	_, err = e.GetBlockRFQs(t.Context(), &GetBlockRFQsRequest{Role: "Maker"})
	require.ErrorIs(t, err, errInvalidBlockRFQRole, "GetBlockRFQs must reject an undocumented role")

	skipLiveWithoutCredentials(t)
	result, err := e.GetBlockRFQs(t.Context(), mockGetBlockRFQsRequest())
	require.NoError(t, err, "GetBlockRFQs must not error")
	if mockTests {
		assert.Equal(t, testDocumentedBlockRFQs(), result, "GetBlockRFQs should decode every field")
	} else {
		for i := range result.BlockRFQs {
			assert.Equalf(t, BlockRFQRoleMaker, result.BlockRFQs[i].Role, "GetBlockRFQs should return Block RFQ %d in the role requested", i)
		}
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetBlockRFQs(t.Context(), mockGetBlockRFQsRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetBlockRFQs should reject a null result")
	_, err = newUnavailableServerExchange(t).GetBlockRFQs(t.Context(), mockGetBlockRFQsRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetBlockRFQs should return the request failure")
}

func TestGetBlockRFQTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetBlockRFQTrades(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetBlockRFQTrades must reject a nil request")
	_, err = e.GetBlockRFQTrades(t.Context(), &GetBlockRFQTradesRequest{Count: 10})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetBlockRFQTrades must reject an empty currency")

	result, err := e.GetBlockRFQTrades(t.Context(), testGetBlockRFQTradesRequest())
	require.NoError(t, err, "GetBlockRFQTrades must not error")
	if mockTests {
		assert.Equal(t, testRecordedBlockRFQTrades(), result, "GetBlockRFQTrades should decode every field")
	} else {
		assert.NotEmpty(t, result.BlockRFQs, "GetBlockRFQTrades should return Block RFQ trades")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetBlockRFQTrades(t.Context(), testGetBlockRFQTradesRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetBlockRFQTrades should reject a null result")
	_, err = newUnavailableServerExchange(t).GetBlockRFQTrades(t.Context(), testGetBlockRFQTradesRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetBlockRFQTrades should return the request failure")

	ex := newWrapperServerExchange(t, func(_ string, params json.RawMessage) string {
		assert.JSONEq(t, `{"currency":"any","count":10}`, string(params), "GetBlockRFQTrades should send every currency as any, as Deribit rejects ANY")
		return `{"jsonrpc":"2.0","result":{"block_rfqs":[]}}`
	})
	arg := &GetBlockRFQTradesRequest{Currency: currency.ANY, Count: 10}
	_, err = ex.GetBlockRFQTrades(t.Context(), arg)
	require.NoError(t, err, "GetBlockRFQTrades must not error for every currency")
	assert.Equal(t, "ANY", arg.Currency.String(), "GetBlockRFQTrades should leave the request unchanged")
}

func TestValidateBlockRFQLegs(t *testing.T) {
	t.Parallel()
	assert.NoError(t, validateBlockRFQLegs(mockAcceptBlockRFQRequest().Legs), "validateBlockRFQLegs should accept Deribit's documented legs")
	assert.ErrorIs(t, validateBlockRFQLegs(nil), errBlockRFQLegsRequired, "validateBlockRFQLegs should reject a quote without legs")
	for _, tc := range []struct {
		name string
		leg  BlockRFQLeg
		err  error
	}{
		{name: "empty instrument", leg: BlockRFQLeg{Direction: DirectionSell, Ratio: 1}, err: errInstrumentNameRequired},
		{name: "empty direction", leg: BlockRFQLeg{InstrumentName: "BTC-8NOV24-72000-C", Ratio: 1}, err: errInvalidDirection},
		{name: "capitalised direction", leg: BlockRFQLeg{InstrumentName: "BTC-8NOV24-72000-C", Direction: "Sell", Ratio: 1}, err: errInvalidDirection},
		{name: "zero ratio", leg: BlockRFQLeg{InstrumentName: "BTC-8NOV24-72000-C", Direction: DirectionSell}, err: errInvalidBlockRFQLegRatio},
	} {
		legs := []BlockRFQLeg{mockAcceptBlockRFQRequest().Legs[0], tc.leg}
		assert.ErrorIsf(t, validateBlockRFQLegs(legs), tc.err, "validateBlockRFQLegs should return the expected error for a second leg with %s", tc.name)
	}
}

func TestValidateCreateBlockRFQLegs(t *testing.T) {
	t.Parallel()
	assert.NoError(t, validateCreateBlockRFQLegs(mockCreateBlockRFQRequest().Legs), "validateCreateBlockRFQLegs should accept Deribit's documented legs")
	assert.ErrorIs(t, validateCreateBlockRFQLegs(nil), errBlockRFQLegsRequired, "validateCreateBlockRFQLegs should reject a Block RFQ without legs")
	for _, tc := range []struct {
		name string
		leg  CreateBlockRFQLeg
		err  error
	}{
		{name: "empty instrument", leg: CreateBlockRFQLeg{Amount: 10000, Direction: DirectionBuy}, err: errInstrumentNameRequired},
		{name: "empty direction", leg: CreateBlockRFQLeg{InstrumentName: "BTC-PERPETUAL", Amount: 10000}, err: errInvalidDirection},
		{name: "zero amount", leg: CreateBlockRFQLeg{InstrumentName: "BTC-PERPETUAL", Direction: DirectionBuy}, err: errInvalidAmount},
		{name: "negative amount", leg: CreateBlockRFQLeg{InstrumentName: "BTC-PERPETUAL", Amount: -10000, Direction: DirectionBuy}, err: errInvalidAmount},
	} {
		legs := []CreateBlockRFQLeg{mockCreateBlockRFQRequest().Legs[0], tc.leg}
		assert.ErrorIsf(t, validateCreateBlockRFQLegs(legs), tc.err, "validateCreateBlockRFQLegs should return the expected error for a second leg with %s", tc.name)
	}
}

func TestValidateBlockRFQHedge(t *testing.T) {
	t.Parallel()
	assert.NoError(t, validateBlockRFQHedge(nil), "validateBlockRFQHedge should accept a hedge left out")
	assert.NoError(t, validateBlockRFQHedge(mockBlockRFQHedge()), "validateBlockRFQHedge should accept Deribit's documented hedge")
	for _, tc := range []struct {
		name  string
		hedge *BlockRFQHedge
		err   error
	}{
		{name: "empty instrument", hedge: &BlockRFQHedge{Direction: DirectionBuy, Price: 70000, Amount: 10}, err: errInstrumentNameRequired},
		{name: "empty direction", hedge: &BlockRFQHedge{InstrumentName: "BTC-PERPETUAL", Price: 70000, Amount: 10}, err: errInvalidDirection},
		{name: "zero amount", hedge: &BlockRFQHedge{InstrumentName: "BTC-PERPETUAL", Direction: DirectionBuy, Price: 70000}, err: errInvalidAmount},
	} {
		assert.ErrorIsf(t, validateBlockRFQHedge(tc.hedge), tc.err, "validateBlockRFQHedge should return the expected error for a hedge with %s", tc.name)
	}
}

func TestValidateBlockRFQTradeAllocations(t *testing.T) {
	t.Parallel()
	valid := []BlockRFQTradeAllocation{
		{UserID: 57874, Amount: 10000},
		{ClientInfo: &BlockRFQAllocationClient{ClientID: 1, ClientLinkID: 2}, Amount: 10000},
	}
	assert.NoError(t, validateBlockRFQTradeAllocations(nil), "validateBlockRFQTradeAllocations should accept allocations left out")
	assert.NoError(t, validateBlockRFQTradeAllocations(valid), "validateBlockRFQTradeAllocations should accept a user's and a broker client's allocation")
	for _, tc := range []struct {
		name       string
		allocation BlockRFQTradeAllocation
		err        error
	}{
		{name: "neither user nor client", allocation: BlockRFQTradeAllocation{Amount: 10000}, err: errBlockRFQAllocationTargetRequired},
		{name: "zero amount", allocation: BlockRFQTradeAllocation{UserID: 57874}, err: errInvalidAmount},
		{name: "negative amount", allocation: BlockRFQTradeAllocation{ClientInfo: &BlockRFQAllocationClient{ClientID: 1, ClientLinkID: 2}, Amount: -1}, err: errInvalidAmount},
	} {
		allocations := []BlockRFQTradeAllocation{valid[0], tc.allocation}
		assert.ErrorIsf(t, validateBlockRFQTradeAllocations(allocations), tc.err, "validateBlockRFQTradeAllocations should return the expected error for a second allocation with %s", tc.name)
	}
}

func TestValidateBlockRFQLabel(t *testing.T) {
	t.Parallel()
	for _, label := range []string{"", mockBlockRFQQuoteLabel, strings.Repeat("a", maximumBlockRFQLabelLength), strings.Repeat("é", maximumBlockRFQLabelLength)} {
		assert.NoErrorf(t, validateBlockRFQLabel(label), "validateBlockRFQLabel should accept label %q of up to 64 characters", label)
	}
	assert.ErrorIs(t, validateBlockRFQLabel(strings.Repeat("a", maximumBlockRFQLabelLength+1)), errBlockRFQLabelTooLong, "validateBlockRFQLabel should reject a label over 64 characters")
}

func TestValidateBlockRFQQuoteIdentifier(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name            string
		blockRFQQuoteID uint64
		blockRFQID      uint64
		label           string
		err             error
	}{
		{name: "quote ID", blockRFQQuoteID: 8},
		{name: "Block RFQ and label", blockRFQID: 3, label: mockBlockRFQQuoteLabel},
		{name: "nothing", err: errBlockRFQQuoteIdentifierRequired},
		{name: "Block RFQ without label", blockRFQID: 3, err: errBlockRFQQuoteIdentifierRequired},
		{name: "label without Block RFQ", label: mockBlockRFQQuoteLabel, err: errBlockRFQQuoteIdentifierRequired},
	} {
		assert.ErrorIsf(t, validateBlockRFQQuoteIdentifier(tc.blockRFQQuoteID, tc.blockRFQID, tc.label), tc.err, "validateBlockRFQQuoteIdentifier should return the expected error for %s", tc.name)
	}
}

func TestCancelAllBlockRFQQuotesResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		input string
		exp   CancelAllBlockRFQQuotesResponse
	}{
		{input: `7`, exp: CancelAllBlockRFQQuotesResponse{CancelledCount: 7}},
		{input: `[{"block_rfq_quote_id":8},{"block_rfq_quote_id":9}]`, exp: CancelAllBlockRFQQuotesResponse{DetailedReports: []json.RawMessage{json.RawMessage(`{"block_rfq_quote_id":8}`), json.RawMessage(`{"block_rfq_quote_id":9}`)}}},
		{input: `[]`, exp: CancelAllBlockRFQQuotesResponse{DetailedReports: []json.RawMessage{}}},
		{input: `null`},
	} {
		var result CancelAllBlockRFQQuotesResponse
		require.NoErrorf(t, json.Unmarshal([]byte(tc.input), &result), "Unmarshal must not error for %s", tc.input)
		assert.Equalf(t, tc.exp, result, "UnmarshalJSON should decode %s", tc.input)
	}
	for _, input := range []string{`"7"`, `-1`, `1.5`, `{}`, `[1`} {
		var result CancelAllBlockRFQQuotesResponse
		assert.ErrorIsf(t, result.UnmarshalJSON([]byte(input)), errInvalidBlockRFQQuoteCancellation, "UnmarshalJSON should reject %s", input)
	}
}
