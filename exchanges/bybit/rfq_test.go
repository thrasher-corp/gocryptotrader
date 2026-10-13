package bybit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const (
	rfqTestID          = "1791008100511241492396882271696580"
	rfqTestLinkID      = "rfq-gct-001"
	rfqTestQuoteID     = "1791008120130044334361923221559805"
	rfqTestQuoteLinkID = "quote-gct-001"
	rfqTestCursor      = "page_token%3D14911%26last_time%3D1791009000000000000%26"
	rfqTestOption      = "BTC-25DEC26-100000-C"
)

var (
	rfqTestStart = time.UnixMilli(1790812800000)
	rfqTestEnd   = time.UnixMilli(1791331200000)
)

func TestCreateRFQ(t *testing.T) {
	t.Parallel()
	_, err := e.CreateRFQ(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateRFQ must reject a nil request")
	leg := RFQLegRequest{Category: "option", Symbol: rfqTestOption, Side: "Buy", Quantity: 10}
	_, err = e.CreateRFQ(t.Context(), &CreateRFQRequest{List: []RFQLegRequest{leg}})
	require.ErrorIs(t, err, errCounterpartiesRequired, "CreateRFQ must reject a request without counterparties")
	_, err = e.CreateRFQ(t.Context(), &CreateRFQRequest{Counterparties: []string{"SPMAKERTEST"}})
	require.ErrorIs(t, err, errRFQLegsRequired, "CreateRFQ must reject a request without legs")
	for _, tc := range []struct {
		name string
		leg  RFQLegRequest
		err  error
	}{
		{name: "category", leg: RFQLegRequest{Symbol: rfqTestOption, Side: "Buy", Quantity: 10}, err: errCategoryNotSet},
		{name: "symbol", leg: RFQLegRequest{Category: "option", Side: "Buy", Quantity: 10}, err: errSymbolMissing},
		{name: "side", leg: RFQLegRequest{Category: "option", Symbol: rfqTestOption, Quantity: 10}, err: order.ErrSideIsInvalid},
		{name: "quantity", leg: RFQLegRequest{Category: "option", Symbol: rfqTestOption, Side: "Buy"}, err: order.ErrAmountIsInvalid},
	} {
		_, err = e.CreateRFQ(t.Context(), &CreateRFQRequest{Counterparties: []string{"SPMAKERTEST"}, List: []RFQLegRequest{tc.leg}})
		require.ErrorIsf(t, err, tc.err, "CreateRFQ must reject a leg without a %s", tc.name)
	}
	_, err = e.CreateRFQ(t.Context(), &CreateRFQRequest{
		Counterparties: []string{"SPMAKERTEST"},
		List:           []RFQLegRequest{leg},
		Hedge:          []RFQHedgeRequest{{Category: "linear", Symbol: "BTCUSDT", Side: "Sell"}},
	})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "CreateRFQ must reject a hedge leg without a quantity")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CreateRFQ(t.Context(), &CreateRFQRequest{
		Counterparties: []string{"SPMAKERTEST"},
		RFQLinkID:      rfqTestLinkID,
		Anonymous:      true,
		StrategyType:   "custom",
		List:           []RFQLegRequest{leg},
		Hedge:          []RFQHedgeRequest{{Category: "linear", Symbol: "BTCUSDT", Side: "Sell", Quantity: 4.6, Price: 121500}},
	})
	require.NoError(t, err, "CreateRFQ must not error")
	if mockTests {
		exp := &CreateRFQResponse{
			RFQID:     "1791008100511241492396882271696580",
			RFQLinkID: "rfq-gct-001",
			Status:    "Active",
			ExpiresAt: types.Time(time.UnixMilli(1791008700512)),
			DeskCode:  "1nu9d1",
		}
		assert.Equal(t, exp, result, "CreateRFQ should decode every field")
		return
	}
	assert.NotEmpty(t, result.RFQID, "CreateRFQ should return an RFQ ID")
}

func TestRFQLegBody(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name             string
		category, symbol string
		side             string
		quantity         float64
		err              error
	}{
		{name: "no category", symbol: rfqTestOption, side: "Buy", quantity: 10, err: errCategoryNotSet},
		{name: "no symbol", category: "option", side: "Buy", quantity: 10, err: errSymbolMissing},
		{name: "no side", category: "option", symbol: rfqTestOption, quantity: 10, err: order.ErrSideIsInvalid},
		{name: "no quantity", category: "option", symbol: rfqTestOption, side: "Buy", err: order.ErrAmountIsInvalid},
		{name: "a negative quantity", category: "option", symbol: rfqTestOption, side: "Buy", quantity: -10, err: order.ErrAmountIsInvalid},
	} {
		body, err := rfqLegBody(tc.category, tc.symbol, tc.side, tc.quantity)
		assert.ErrorIsf(t, err, tc.err, "rfqLegBody should reject a leg with %s", tc.name)
		assert.Nilf(t, body, "rfqLegBody should return no body for a leg with %s", tc.name)
	}

	body, err := rfqLegBody("linear", "BTCUSDT", "Sell", 4.6)
	require.NoError(t, err, "rfqLegBody must not error")
	exp := map[string]any{"category": "linear", "symbol": "BTCUSDT", "side": "Sell", "qty": "4.6"}
	assert.Equal(t, exp, body, "rfqLegBody should send every field of the leg, with the quantity as a string")
}

func TestGetRFQConfig(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetRFQConfig(t.Context())
	require.NoError(t, err, "GetRFQConfig must not error")
	if mockTests {
		exp := &RFQConfigResponse{
			DeskCode:                      "1nu9d1",
			MaxLegs:                       25,
			MaxLP:                         50,
			MaxActiveRFQ:                  10,
			RFQExpireTime:                 10,
			MinLimitQuantitySpotOrder:     10,
			MinLimitQuantityContractOrder: 10,
			MinLimitQuantityOptionOrder:   1,
			StrategyTypes: []RFQStrategyType{
				{StrategyName: "custom"},
				{StrategyName: "FundingRate"},
				{StrategyName: "CarryTrade"},
			},
			Counterparties: []RFQCounterparty{
				{TraderName: "Bernie LP", DeskCode: "Bernie", Type: "LP"},
				{TraderName: "SP Maker Test", DeskCode: "SPMAKERTEST"},
			},
		}
		assert.Equal(t, exp, result, "GetRFQConfig should decode every field")
		return
	}
	assert.NotEmpty(t, result.DeskCode, "GetRFQConfig should return the caller's desk code")
}

func TestCancelRFQ(t *testing.T) {
	t.Parallel()
	_, err := e.CancelRFQ(t.Context(), "", "")
	require.ErrorIs(t, err, errRFQIdentifierRequired, "CancelRFQ must reject a request without an RFQ ID or link ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelRFQ(t.Context(), rfqTestID, rfqTestLinkID)
	require.NoError(t, err, "CancelRFQ must not error")
	if mockTests {
		exp := &CancelRFQResponse{RFQID: "1791008100511241492396882271696580", RFQLinkID: "rfq-gct-001"}
		assert.Equal(t, exp, result, "CancelRFQ should decode every field")
	}
}

func TestCancelAllRFQs(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelAllRFQs(t.Context())
	require.NoError(t, err, "CancelAllRFQs must not error")
	if mockTests {
		exp := []RFQCancellation{
			{RFQID: "1791008100511241492396882271696580", RFQLinkID: "rfq-gct-001"},
			{RFQID: "175766967076315412093641573648082", Code: 110001, Message: "RFQ is not active"},
		}
		assert.Equal(t, exp, result, "CancelAllRFQs should decode every field")
	}
}

func TestAcceptNonLPQuote(t *testing.T) {
	t.Parallel()
	_, err := e.AcceptNonLPQuote(t.Context(), "")
	require.ErrorIs(t, err, errRFQIDRequired, "AcceptNonLPQuote must reject an empty RFQ ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.AcceptNonLPQuote(t.Context(), rfqTestID)
	require.NoError(t, err, "AcceptNonLPQuote must not error")
	if mockTests {
		exp := &AcceptNonLPQuoteResponse{RFQID: "1791008100511241492396882271696580"}
		assert.Equal(t, exp, result, "AcceptNonLPQuote should decode every field")
	}
}

func TestCreateRFQQuote(t *testing.T) {
	t.Parallel()
	_, err := e.CreateRFQQuote(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateRFQQuote must reject a nil request")
	buy := []RFQQuoteLegRequest{{Category: "option", Symbol: rfqTestOption, Price: 1250.5}}
	_, err = e.CreateRFQQuote(t.Context(), &CreateRFQQuoteRequest{QuoteBuyList: buy})
	require.ErrorIs(t, err, errRFQIDRequired, "CreateRFQQuote must reject an empty RFQ ID")
	_, err = e.CreateRFQQuote(t.Context(), &CreateRFQQuoteRequest{RFQID: rfqTestID})
	require.ErrorIs(t, err, errQuoteLegsRequired, "CreateRFQQuote must reject a quote without legs")
	_, err = e.CreateRFQQuote(t.Context(), &CreateRFQQuoteRequest{RFQID: rfqTestID, QuoteSellList: []RFQQuoteLegRequest{{Symbol: rfqTestOption, Price: 1260}}})
	require.ErrorIs(t, err, errCategoryNotSet, "CreateRFQQuote must reject a leg without a category")
	_, err = e.CreateRFQQuote(t.Context(), &CreateRFQQuoteRequest{RFQID: rfqTestID, QuoteSellList: []RFQQuoteLegRequest{{Category: "option", Price: 1260}}})
	require.ErrorIs(t, err, errSymbolMissing, "CreateRFQQuote must reject a leg without a symbol")
	_, err = e.CreateRFQQuote(t.Context(), &CreateRFQQuoteRequest{RFQID: rfqTestID, QuoteBuyList: []RFQQuoteLegRequest{{Symbol: rfqTestOption, Price: 1250.5}}})
	require.ErrorIs(t, err, errCategoryNotSet, "CreateRFQQuote must reject a buy leg without a category")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CreateRFQQuote(t.Context(), &CreateRFQQuoteRequest{
		RFQID:         rfqTestID,
		QuoteLinkID:   rfqTestQuoteLinkID,
		Anonymous:     true,
		ExpireIn:      60,
		QuoteBuyList:  append(buy, RFQQuoteLegRequest{Category: "linear", Symbol: "BTCUSDT"}),
		QuoteSellList: []RFQQuoteLegRequest{{Category: "option", Symbol: rfqTestOption, Price: 1260}},
	})
	require.NoError(t, err, "CreateRFQQuote must not error")
	if mockTests {
		exp := &CreateRFQQuoteResponse{
			RFQID:       "1791008100511241492396882271696580",
			QuoteID:     "1791008120130044334361923221559805",
			QuoteLinkID: "quote-gct-001",
			ExpiresAt:   types.Time(time.UnixMilli(1791008180126)),
			DeskCode:    "SPMAKERTEST",
			Status:      "Active",
		}
		assert.Equal(t, exp, result, "CreateRFQQuote should decode every field")
	}
}

func TestRFQQuoteLegsBody(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		legs []RFQQuoteLegRequest
		err  error
	}{
		{name: "a leg without a category", legs: []RFQQuoteLegRequest{{Category: "option", Symbol: rfqTestOption, Price: 1250.5}, {Symbol: "BTCUSDT"}}, err: errCategoryNotSet},
		{name: "a leg without a symbol", legs: []RFQQuoteLegRequest{{Category: "option", Symbol: rfqTestOption, Price: 1250.5}, {Category: "linear"}}, err: errSymbolMissing},
	} {
		body, err := rfqQuoteLegsBody(tc.legs)
		assert.ErrorIsf(t, err, tc.err, "rfqQuoteLegsBody should reject %s", tc.name)
		assert.ErrorContainsf(t, err, "leg 1", "rfqQuoteLegsBody should name the leg it rejects for %s", tc.name)
		assert.Nilf(t, body, "rfqQuoteLegsBody should return no body for %s", tc.name)
	}

	body, err := rfqQuoteLegsBody([]RFQQuoteLegRequest{{Category: "option", Symbol: rfqTestOption, Price: 1250.5}, {Category: "linear", Symbol: "BTCUSDT"}})
	require.NoError(t, err, "rfqQuoteLegsBody must not error")
	exp := []map[string]any{
		{"category": "option", "symbol": rfqTestOption, "price": "1250.5"},
		{"category": "linear", "symbol": "BTCUSDT"},
	}
	assert.Equal(t, exp, body, "rfqQuoteLegsBody should send each leg in order, and a hedge leg without a price so it takes the RFQ's hedge price")
}

func TestExecuteRFQQuote(t *testing.T) {
	t.Parallel()
	_, err := e.ExecuteRFQQuote(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ExecuteRFQQuote must reject a nil request")
	_, err = e.ExecuteRFQQuote(t.Context(), &ExecuteRFQQuoteRequest{QuoteID: rfqTestQuoteID, QuoteSide: "Buy"})
	require.ErrorIs(t, err, errRFQIDRequired, "ExecuteRFQQuote must reject an empty RFQ ID")
	_, err = e.ExecuteRFQQuote(t.Context(), &ExecuteRFQQuoteRequest{RFQID: rfqTestID, QuoteSide: "Buy"})
	require.ErrorIs(t, err, errQuoteIDRequired, "ExecuteRFQQuote must reject an empty quote ID")
	_, err = e.ExecuteRFQQuote(t.Context(), &ExecuteRFQQuoteRequest{RFQID: rfqTestID, QuoteID: rfqTestQuoteID})
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "ExecuteRFQQuote must reject an empty quote side")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.ExecuteRFQQuote(t.Context(), &ExecuteRFQQuoteRequest{RFQID: rfqTestID, QuoteID: rfqTestQuoteID, QuoteSide: "Buy", IsHedge: true})
	require.NoError(t, err, "ExecuteRFQQuote must not error")
	if mockTests {
		exp := &ExecuteRFQQuoteResponse{
			RFQID:     "1791008100511241492396882271696580",
			RFQLinkID: "rfq-gct-001",
			QuoteID:   "1791008120130044334361923221559805",
			Status:    "PendingFill",
		}
		assert.Equal(t, exp, result, "ExecuteRFQQuote should decode every field")
	}
}

func TestCancelRFQQuote(t *testing.T) {
	t.Parallel()
	_, err := e.CancelRFQQuote(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelRFQQuote must reject a nil request")
	_, err = e.CancelRFQQuote(t.Context(), &CancelRFQQuoteRequest{})
	require.ErrorIs(t, err, errQuoteIdentifierRequired, "CancelRFQQuote must reject a request without an identifier")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelRFQQuote(t.Context(), &CancelRFQQuoteRequest{QuoteID: rfqTestQuoteID, RFQID: rfqTestID, QuoteLinkID: rfqTestQuoteLinkID})
	require.NoError(t, err, "CancelRFQQuote must not error")
	if mockTests {
		exp := &CancelRFQQuoteResponse{
			RFQID:       "1791008100511241492396882271696580",
			QuoteID:     "1791008120130044334361923221559805",
			QuoteLinkID: "quote-gct-001",
		}
		assert.Equal(t, exp, result, "CancelRFQQuote should decode every field")
	}
}

func TestCancelAllRFQQuotes(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelAllRFQQuotes(t.Context())
	require.NoError(t, err, "CancelAllRFQQuotes must not error")
	if mockTests {
		exp := []RFQQuoteCancellation{
			{
				RFQID:       "1791008100511241492396882271696580",
				QuoteID:     "1791008120130044334361923221559805",
				QuoteLinkID: "quote-gct-001",
			},
			{
				RFQID:   "175740723913299909861293671607573",
				QuoteID: "1757407497684679708210572531298710",
				Code:    110001,
				Message: "Quote is not active",
			},
		}
		assert.Equal(t, exp, result, "CancelAllRFQQuotes should decode every field")
	}
}

func TestGetRFQsRealtime(t *testing.T) {
	t.Parallel()
	_, err := e.GetRFQsRealtime(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetRFQsRealtime must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetRFQsRealtime(t.Context(), &RFQsRealtimeRequest{TraderType: "request"})
	require.NoError(t, err, "GetRFQsRealtime must not error")
	if mockTests {
		exp := &RFQsRealtimeResponse{
			List: []RFQ{
				{
					RFQID:     "1791008100511241492396882271696580",
					RFQLinkID: "rfq-gct-001",
					Counterparties: []string{
						"SPMAKERTEST",
					},
					ExpiresAt:              types.Time(time.UnixMilli(1791008700512)),
					StrategyType:           "custom",
					Status:                 "Active",
					AcceptOtherQuoteStatus: true,
					DeskCode:               "1nu9d1",
					CreatedAt:              types.Time(time.UnixMilli(1791008100512)),
					UpdatedAt:              types.Time(time.UnixMilli(1791008110131)),
					Legs: []RFQLeg{
						{Category: "option", Symbol: "BTC-25DEC26-100000-C", Side: "Buy", Quantity: 10},
					},
					Hedge: []RFQHedgeLeg{
						{
							Category: "linear",
							Symbol:   "BTCUSDT",
							Side:     "Sell",
							Quantity: 4.6,
							Price:    121500,
						},
					},
				},
				{
					RFQID: "1791008090883736854420161904593980",
					Counterparties: []string{
						"SPMAKERTEST",
						"Bernie",
					},
					ExpiresAt:    types.Time(time.UnixMilli(1791008690885)),
					StrategyType: "custom",
					Status:       "Active",
					Anonymous:    true,
					CreatedAt:    types.Time(time.UnixMilli(1791008090885)),
					UpdatedAt:    types.Time(time.UnixMilli(1791008090885)),
					Legs: []RFQLeg{
						{Category: "linear", Symbol: "BTCUSDT", Side: "Sell", Quantity: 1},
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetRFQsRealtime should decode every field")
	}
	result, err = e.GetRFQsRealtime(t.Context(), &RFQsRealtimeRequest{RFQID: rfqTestID, RFQLinkID: rfqTestLinkID})
	require.NoError(t, err, "GetRFQsRealtime must not error for an RFQ ID and link ID")
	if mockTests {
		exp := &RFQsRealtimeResponse{
			List: []RFQ{
				{
					RFQID:     "1791008100511241492396882271696580",
					RFQLinkID: "rfq-gct-001",
					Counterparties: []string{
						"SPMAKERTEST",
					},
					ExpiresAt:              types.Time(time.UnixMilli(1791008700512)),
					StrategyType:           "custom",
					Status:                 "Active",
					AcceptOtherQuoteStatus: true,
					DeskCode:               "1nu9d1",
					CreatedAt:              types.Time(time.UnixMilli(1791008100512)),
					UpdatedAt:              types.Time(time.UnixMilli(1791008110131)),
					Legs: []RFQLeg{
						{Category: "option", Symbol: "BTC-25DEC26-100000-C", Side: "Buy", Quantity: 10},
					},
					Hedge: []RFQHedgeLeg{
						{
							Category: "linear",
							Symbol:   "BTCUSDT",
							Side:     "Sell",
							Quantity: 4.6,
							Price:    121500,
						},
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetRFQsRealtime should decode every field of case 2")
	}
}

func TestGetRFQs(t *testing.T) {
	t.Parallel()
	_, err := e.GetRFQs(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetRFQs must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetRFQs(t.Context(), &RFQsRequest{TraderType: "request", Limit: 2, Cursor: rfqTestCursor})
	require.NoError(t, err, "GetRFQs must not error")
	if mockTests {
		exp := &RFQsResponse{
			Cursor: "page_token%3D14912%26last_time%3D1791008090885000000%26",
			List: []RFQ{
				{
					RFQID:     "1791008100511241492396882271696580",
					RFQLinkID: "rfq-gct-001",
					Counterparties: []string{
						"SPMAKERTEST",
					},
					ExpiresAt:              types.Time(time.UnixMilli(1791008700512)),
					StrategyType:           "custom",
					Status:                 "Filled",
					AcceptOtherQuoteStatus: true,
					DeskCode:               "1nu9d1",
					CreatedAt:              types.Time(time.UnixMilli(1791008100512)),
					UpdatedAt:              types.Time(time.UnixMilli(1791008130301)),
					Legs: []RFQLeg{
						{Category: "option", Symbol: "BTC-25DEC26-100000-C", Side: "Buy", Quantity: 10},
					},
					Hedge: []RFQHedgeLeg{
						{
							Category: "linear",
							Symbol:   "BTCUSDT",
							Side:     "Sell",
							Quantity: 4.6,
							Price:    121500,
						},
					},
				},
				{
					RFQID: "1791008090883736854420161904593980",
					Counterparties: []string{
						"SPMAKERTEST",
						"Bernie",
					},
					ExpiresAt:    types.Time(time.UnixMilli(1791008690885)),
					StrategyType: "custom",
					Status:       "Expired",
					Anonymous:    true,
					CreatedAt:    types.Time(time.UnixMilli(1791008090885)),
					UpdatedAt:    types.Time(time.UnixMilli(1791008696046)),
					Legs: []RFQLeg{
						{Category: "linear", Symbol: "BTCUSDT", Side: "Sell", Quantity: 1},
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetRFQs should decode every field")
	}
	result, err = e.GetRFQs(t.Context(), &RFQsRequest{RFQID: rfqTestID, RFQLinkID: rfqTestLinkID, Status: "Filled"})
	require.NoError(t, err, "GetRFQs must not error for an RFQ ID, link ID and status")
	if mockTests {
		exp := &RFQsResponse{
			List: []RFQ{
				{
					RFQID:     "1791008100511241492396882271696580",
					RFQLinkID: "rfq-gct-001",
					Counterparties: []string{
						"SPMAKERTEST",
					},
					ExpiresAt:              types.Time(time.UnixMilli(1791008700512)),
					StrategyType:           "custom",
					Status:                 "Filled",
					AcceptOtherQuoteStatus: true,
					DeskCode:               "1nu9d1",
					CreatedAt:              types.Time(time.UnixMilli(1791008100512)),
					UpdatedAt:              types.Time(time.UnixMilli(1791008130301)),
					Legs: []RFQLeg{
						{Category: "option", Symbol: "BTC-25DEC26-100000-C", Side: "Buy", Quantity: 10},
					},
					Hedge: []RFQHedgeLeg{
						{
							Category: "linear",
							Symbol:   "BTCUSDT",
							Side:     "Sell",
							Quantity: 4.6,
							Price:    121500,
						},
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetRFQs should decode every field of case 2")
	}
}

func TestGetRFQDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetRFQDetails(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetRFQDetails must reject a nil request")
	_, err = e.GetRFQDetails(t.Context(), &RFQDetailsRequest{StartTime: rfqTestEnd, EndTime: rfqTestStart})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetRFQDetails must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetRFQDetails(t.Context(), &RFQDetailsRequest{TraderType: "request", StartTime: rfqTestStart, EndTime: rfqTestEnd, Limit: 2, Cursor: rfqTestCursor})
	require.NoError(t, err, "GetRFQDetails must not error")
	if mockTests {
		exp := &RFQDetailsResponse{
			Cursor: "page_token%3D14913%26last_time%3D1791008090885000000%26",
			List: []RFQDetail{
				{
					RFQID:     "1791008100511241492396882271696580",
					RFQLinkID: "rfq-gct-001",
					Counterparties: []string{
						"SPMAKERTEST",
					},
					ExpiresAt:              types.Time(time.UnixMilli(1791008700512)),
					StrategyType:           "custom",
					Status:                 "Filled",
					AcceptOtherQuoteStatus: true,
					ExecutedQuoteID:        "1791008120130044334361923221559805",
					ExecutedQuoteSide:      "Buy",
					DeskCode:               "1nu9d1",
					CreatedAt:              types.Time(time.UnixMilli(1791008100512)),
					UpdatedAt:              types.Time(time.UnixMilli(1791008130301)),
					Legs: []RFQLeg{
						{Category: "option", Symbol: "BTC-25DEC26-100000-C", Side: "Buy", Quantity: 10},
					},
					Hedge: []RFQHedgeLeg{
						{
							Category: "linear",
							Symbol:   "BTCUSDT",
							Side:     "Sell",
							Quantity: 4.6,
							Price:    121500,
						},
					},
					QuoteList: []RFQDetailQuote{
						{
							QuoteID:   "1791008120130044334361923221559805",
							DeskCode:  "SPMAKERTEST",
							Status:    "Filled",
							CreatedAt: types.Time(time.UnixMilli(1791008120126)),
							UpdatedAt: types.Time(time.UnixMilli(1791008130301)),
							ExpiresAt: types.Time(time.UnixMilli(1791008180126)),
							QuoteBuyList: []RFQQuoteLeg{
								{Category: "option", Symbol: "BTC-25DEC26-100000-C", Price: 1250.5, Quantity: 10},
								{
									Category: "linear",
									Symbol:   "BTCUSDT",
									Price:    121500,
									Quantity: 4.6,
									IsHedge:  true,
								},
							},
							QuoteSellList: []RFQQuoteLeg{
								{Category: "option", Symbol: "BTC-25DEC26-100000-C", Price: 1260, Quantity: 10},
								{
									Category: "linear",
									Symbol:   "BTCUSDT",
									Price:    121400,
									Quantity: 4.6,
									IsHedge:  true,
								},
							},
						},
						{
							QuoteID:   "1791008118083427576602342578477746",
							Anonymous: true,
							Status:    "Canceled",
							CreatedAt: types.Time(time.UnixMilli(1791008118080)),
							UpdatedAt: types.Time(time.UnixMilli(1791008121635)),
							ExpiresAt: types.Time(time.UnixMilli(1791008178080)),
							QuoteBuyList: []RFQQuoteLeg{
								{Category: "option", Symbol: "BTC-25DEC26-100000-C", Price: 1249, Quantity: 10},
							},
							QuoteSellList: []RFQQuoteLeg{},
						},
					},
					TradeLegs: []RFQDetailTradeLeg{
						{
							Category:     "option",
							OrderID:      "64fe4108-555e-4361-ae2d-3a8d0c292859",
							Symbol:       "BTC-25DEC26-100000-C",
							Side:         "Buy",
							Price:        1250.5,
							Quantity:     10,
							MarkPrice:    121480.2,
							ExecutionFee: 7.5,
							ExtraFees:    "[{\"feeCoin\":\"USDT\",\"feeType\":\"GST\",\"subFeeType\":\"IND_GST\",\"feeRate\":\"0.0000675\",\"fee\":\"0.000506\"}]",
							ExecutionID:  "42b8be1e-36cf-4aba-bb75-4602cc11df37",
						},
					},
				},
				{
					RFQID: "1791008090883736854420161904593980",
					Counterparties: []string{
						"SPMAKERTEST",
						"Bernie",
					},
					ExpiresAt:         types.Time(time.UnixMilli(1791008690885)),
					StrategyType:      "custom",
					Status:            "Failed",
					ExecutedQuoteID:   "1791008095586174663206671159484665",
					ExecutedQuoteSide: "Sell",
					Anonymous:         true,
					CreatedAt:         types.Time(time.UnixMilli(1791008090885)),
					UpdatedAt:         types.Time(time.UnixMilli(1791008096046)),
					Legs: []RFQLeg{
						{Category: "linear", Symbol: "BTCUSDT", Side: "Sell", Quantity: 1},
					},
					Hedge:     []RFQHedgeLeg{},
					QuoteList: []RFQDetailQuote{},
					TradeLegs: []RFQDetailTradeLeg{
						{
							Category:      "linear",
							OrderID:       "69667acb-7048-48d7-90b9-ccbdfd423130",
							Symbol:        "BTCUSDT",
							Side:          "Sell",
							Price:         121480,
							Quantity:      1,
							ResultCode:    110007,
							ResultMessage: "Insufficient available balance",
							RejectParty:   "Taker",
						},
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetRFQDetails should decode every field")
	}
	result, err = e.GetRFQDetails(t.Context(), &RFQDetailsRequest{RFQID: rfqTestID, RFQLinkID: rfqTestLinkID, Status: "Filled"})
	require.NoError(t, err, "GetRFQDetails must not error for an RFQ ID, link ID and status")
	if mockTests {
		exp := &RFQDetailsResponse{
			List: []RFQDetail{
				{
					RFQID:     "1791008100511241492396882271696580",
					RFQLinkID: "rfq-gct-001",
					Counterparties: []string{
						"SPMAKERTEST",
					},
					ExpiresAt:              types.Time(time.UnixMilli(1791008700512)),
					StrategyType:           "custom",
					Status:                 "Filled",
					AcceptOtherQuoteStatus: true,
					ExecutedQuoteID:        "1791008120130044334361923221559805",
					ExecutedQuoteSide:      "Buy",
					DeskCode:               "1nu9d1",
					CreatedAt:              types.Time(time.UnixMilli(1791008100512)),
					UpdatedAt:              types.Time(time.UnixMilli(1791008130301)),
					Legs: []RFQLeg{
						{Category: "option", Symbol: "BTC-25DEC26-100000-C", Side: "Buy", Quantity: 10},
					},
					Hedge: []RFQHedgeLeg{
						{
							Category: "linear",
							Symbol:   "BTCUSDT",
							Side:     "Sell",
							Quantity: 4.6,
							Price:    121500,
						},
					},
					QuoteList: []RFQDetailQuote{
						{
							QuoteID:   "1791008120130044334361923221559805",
							DeskCode:  "SPMAKERTEST",
							Status:    "Filled",
							CreatedAt: types.Time(time.UnixMilli(1791008120126)),
							UpdatedAt: types.Time(time.UnixMilli(1791008130301)),
							ExpiresAt: types.Time(time.UnixMilli(1791008180126)),
							QuoteBuyList: []RFQQuoteLeg{
								{Category: "option", Symbol: "BTC-25DEC26-100000-C", Price: 1250.5, Quantity: 10},
								{
									Category: "linear",
									Symbol:   "BTCUSDT",
									Price:    121500,
									Quantity: 4.6,
									IsHedge:  true,
								},
							},
							QuoteSellList: []RFQQuoteLeg{
								{Category: "option", Symbol: "BTC-25DEC26-100000-C", Price: 1260, Quantity: 10},
								{
									Category: "linear",
									Symbol:   "BTCUSDT",
									Price:    121400,
									Quantity: 4.6,
									IsHedge:  true,
								},
							},
						},
						{
							QuoteID:   "1791008118083427576602342578477746",
							Anonymous: true,
							Status:    "Canceled",
							CreatedAt: types.Time(time.UnixMilli(1791008118080)),
							UpdatedAt: types.Time(time.UnixMilli(1791008121635)),
							ExpiresAt: types.Time(time.UnixMilli(1791008178080)),
							QuoteBuyList: []RFQQuoteLeg{
								{Category: "option", Symbol: "BTC-25DEC26-100000-C", Price: 1249, Quantity: 10},
							},
							QuoteSellList: []RFQQuoteLeg{},
						},
					},
					TradeLegs: []RFQDetailTradeLeg{
						{
							Category:     "option",
							OrderID:      "64fe4108-555e-4361-ae2d-3a8d0c292859",
							Symbol:       "BTC-25DEC26-100000-C",
							Side:         "Buy",
							Price:        1250.5,
							Quantity:     10,
							MarkPrice:    121480.2,
							ExecutionFee: 7.5,
							ExtraFees:    "[{\"feeCoin\":\"USDT\",\"feeType\":\"GST\",\"subFeeType\":\"IND_GST\",\"feeRate\":\"0.0000675\",\"fee\":\"0.000506\"}]",
							ExecutionID:  "42b8be1e-36cf-4aba-bb75-4602cc11df37",
						},
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetRFQDetails should decode every field of case 2")
	}
}

func TestGetRFQQuotesRealtime(t *testing.T) {
	t.Parallel()
	_, err := e.GetRFQQuotesRealtime(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetRFQQuotesRealtime must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetRFQQuotesRealtime(t.Context(), &RFQQuotesRealtimeRequest{RFQID: rfqTestID, TraderType: "request"})
	require.NoError(t, err, "GetRFQQuotesRealtime must not error for the inquiring party")
	if mockTests {
		exp := &RFQQuotesRealtimeResponse{
			List: []RFQQuote{
				{
					RFQID:             "1791008100511241492396882271696580",
					RFQLinkID:         "rfq-gct-001",
					QuoteID:           "1791008120130044334361923221559805",
					ExpiresAt:         types.Time(time.UnixMilli(1791008180126)),
					DeskCode:          "SPMAKERTEST",
					Anonymous:         true,
					Status:            "PendingFill",
					ExecutedQuoteSide: "Buy",
					CreatedAt:         types.Time(time.UnixMilli(1791008120126)),
					UpdatedAt:         types.Time(time.UnixMilli(1791008130177)),
					QuoteBuyList: []RFQQuoteLeg{
						{Category: "option", Symbol: "BTC-25DEC26-100000-C", Price: 1250.5, Quantity: 10},
						{
							Category: "linear",
							Symbol:   "BTCUSDT",
							Price:    121500,
							Quantity: 4.6,
							IsHedge:  true,
						},
					},
					QuoteSellList: []RFQQuoteLeg{
						{Category: "option", Symbol: "BTC-25DEC26-100000-C", Price: 1260, Quantity: 10},
						{
							Category: "linear",
							Symbol:   "BTCUSDT",
							Price:    121400,
							Quantity: 4.6,
							IsHedge:  true,
						},
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetRFQQuotesRealtime should decode every field")
	}
	result, err = e.GetRFQQuotesRealtime(t.Context(), &RFQQuotesRealtimeRequest{QuoteID: rfqTestQuoteID, QuoteLinkID: rfqTestQuoteLinkID, TraderType: "quote"})
	require.NoError(t, err, "GetRFQQuotesRealtime must not error for the quoting party")
	if mockTests {
		exp := &RFQQuotesRealtimeResponse{
			List: []RFQQuote{
				{
					RFQID:             "1791008100511241492396882271696580",
					QuoteID:           "1791008120130044334361923221559805",
					QuoteLinkID:       "quote-gct-001",
					ExpiresAt:         types.Time(time.UnixMilli(1791008180126)),
					DeskCode:          "SPMAKERTEST",
					Anonymous:         true,
					Status:            "PendingFill",
					ExecutedQuoteSide: "Buy",
					CreatedAt:         types.Time(time.UnixMilli(1791008120126)),
					UpdatedAt:         types.Time(time.UnixMilli(1791008130177)),
					QuoteBuyList: []RFQQuoteLeg{
						{Category: "option", Symbol: "BTC-25DEC26-100000-C", Price: 1250.5, Quantity: 10},
						{
							Category: "linear",
							Symbol:   "BTCUSDT",
							Price:    121500,
							Quantity: 4.6,
							IsHedge:  true,
						},
					},
					QuoteSellList: []RFQQuoteLeg{
						{Category: "option", Symbol: "BTC-25DEC26-100000-C", Price: 1260, Quantity: 10},
						{
							Category: "linear",
							Symbol:   "BTCUSDT",
							Price:    121400,
							Quantity: 4.6,
							IsHedge:  true,
						},
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetRFQQuotesRealtime should decode every field of case 2")
	}
}

func TestGetRFQQuotes(t *testing.T) {
	t.Parallel()
	_, err := e.GetRFQQuotes(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetRFQQuotes must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetRFQQuotes(t.Context(), &RFQQuotesRequest{RFQID: rfqTestID, TraderType: "request", Status: "Filled", Limit: 1, Cursor: rfqTestCursor})
	require.NoError(t, err, "GetRFQQuotes must not error for the inquiring party")
	if mockTests {
		exp := &RFQQuotesResponse{
			Cursor: "page_token%3D2207%26last_time%3D1791008120126000000%26",
			List: []RFQQuote{
				{
					RFQID:             "1791008100511241492396882271696580",
					RFQLinkID:         "rfq-gct-001",
					QuoteID:           "1791008120130044334361923221559805",
					ExpiresAt:         types.Time(time.UnixMilli(1791008180126)),
					DeskCode:          "SPMAKERTEST",
					Anonymous:         true,
					Status:            "Filled",
					ExecutedQuoteSide: "Buy",
					CreatedAt:         types.Time(time.UnixMilli(1791008120126)),
					UpdatedAt:         types.Time(time.UnixMilli(1791008130301)),
					QuoteBuyList: []RFQQuoteLeg{
						{Category: "option", Symbol: "BTC-25DEC26-100000-C", Price: 1250.5, Quantity: 10},
						{
							Category: "linear",
							Symbol:   "BTCUSDT",
							Price:    121500,
							Quantity: 4.6,
							IsHedge:  true,
						},
					},
					QuoteSellList: []RFQQuoteLeg{
						{Category: "option", Symbol: "BTC-25DEC26-100000-C", Price: 1260, Quantity: 10},
						{
							Category: "linear",
							Symbol:   "BTCUSDT",
							Price:    121400,
							Quantity: 4.6,
							IsHedge:  true,
						},
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetRFQQuotes should decode every field")
	}
	result, err = e.GetRFQQuotes(t.Context(), &RFQQuotesRequest{QuoteID: rfqTestQuoteID, QuoteLinkID: rfqTestQuoteLinkID, TraderType: "quote"})
	require.NoError(t, err, "GetRFQQuotes must not error for the quoting party")
	if mockTests {
		exp := &RFQQuotesResponse{
			List: []RFQQuote{
				{
					RFQID:             "1791008100511241492396882271696580",
					QuoteID:           "1791008120130044334361923221559805",
					QuoteLinkID:       "quote-gct-001",
					ExpiresAt:         types.Time(time.UnixMilli(1791008180126)),
					DeskCode:          "SPMAKERTEST",
					Anonymous:         true,
					Status:            "Filled",
					ExecutedQuoteSide: "Buy",
					CreatedAt:         types.Time(time.UnixMilli(1791008120126)),
					UpdatedAt:         types.Time(time.UnixMilli(1791008130301)),
					QuoteBuyList: []RFQQuoteLeg{
						{Category: "option", Symbol: "BTC-25DEC26-100000-C", Price: 1250.5, Quantity: 10},
						{
							Category: "linear",
							Symbol:   "BTCUSDT",
							Price:    121500,
							Quantity: 4.6,
							IsHedge:  true,
						},
					},
					QuoteSellList: []RFQQuoteLeg{
						{Category: "option", Symbol: "BTC-25DEC26-100000-C", Price: 1260, Quantity: 10},
						{
							Category: "linear",
							Symbol:   "BTCUSDT",
							Price:    121400,
							Quantity: 4.6,
							IsHedge:  true,
						},
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetRFQQuotes should decode every field of case 2")
	}
}

func TestGetRFQTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetRFQTradeHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetRFQTradeHistory must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetRFQTradeHistory(t.Context(), &RFQTradeHistoryRequest{TraderType: "request", Limit: 2, Cursor: rfqTestCursor})
	require.NoError(t, err, "GetRFQTradeHistory must not error")
	if mockTests {
		exp := &RFQTradeHistoryResponse{
			Cursor: "page_token%3D981%26last_time%3D1791008090885000000%26",
			List: []RFQTrade{
				{
					RFQID:         "1791008100511241492396882271696580",
					RFQLinkID:     "rfq-gct-001",
					QuoteID:       "1791008120130044334361923221559805",
					QuoteLinkID:   "quote-gct-001",
					QuoteSide:     "Buy",
					StrategyType:  "custom",
					Status:        "Filled",
					RFQDeskCode:   "1nu9d1",
					QuoteDeskCode: "SPMAKERTEST",
					CreatedAt:     types.Time(time.UnixMilli(1791008130177)),
					UpdatedAt:     types.Time(time.UnixMilli(1791008130301)),
					Legs: []RFQTradeLeg{
						{
							Category:     "option",
							OrderID:      "64fe4108-555e-4361-ae2d-3a8d0c292859",
							Symbol:       "BTC-25DEC26-100000-C",
							Side:         "Buy",
							Price:        1250.5,
							Quantity:     10,
							MarkPrice:    121480.2,
							ExecutionFee: 7.5,
							ExecutionID:  "42b8be1e-36cf-4aba-bb75-4602cc11df37",
						},
					},
				},
				{
					RFQID:         "1791008090883736854420161904593980",
					QuoteID:       "1791008095586174663206671159484665",
					QuoteSide:     "Sell",
					StrategyType:  "PerpBasis",
					Status:        "Failed",
					RFQDeskCode:   "1nu9d1",
					QuoteDeskCode: "lines100412673",
					CreatedAt:     types.Time(time.UnixMilli(1791008095588)),
					UpdatedAt:     types.Time(time.UnixMilli(1791008096046)),
					Legs: []RFQTradeLeg{
						{
							Category:      "linear",
							OrderID:       "db852bcd-052e-49b7-ba10-059622e1219b",
							Symbol:        "BTCUSDT-25DEC26",
							Side:          "Sell",
							Price:         121887,
							Quantity:      1,
							ResultCode:    111002,
							ResultMessage: "Rejected caused by another legs",
						},
						{
							Category:      "linear",
							OrderID:       "69667acb-7048-48d7-90b9-ccbdfd423130",
							Symbol:        "BTCUSDT",
							Side:          "Buy",
							Price:         121038,
							Quantity:      1,
							ResultCode:    110007,
							ResultMessage: "Insufficient available balance",
							RejectParty:   "Taker",
						},
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetRFQTradeHistory should decode every field")
	}
	result, err = e.GetRFQTradeHistory(t.Context(), &RFQTradeHistoryRequest{RFQID: rfqTestID, RFQLinkID: rfqTestLinkID, QuoteID: rfqTestQuoteID, QuoteLinkID: rfqTestQuoteLinkID, Status: "Filled"})
	require.NoError(t, err, "GetRFQTradeHistory must not error for RFQ and quote identifiers")
	if mockTests {
		exp := &RFQTradeHistoryResponse{
			List: []RFQTrade{
				{
					RFQID:         "1791008100511241492396882271696580",
					RFQLinkID:     "rfq-gct-001",
					QuoteID:       "1791008120130044334361923221559805",
					QuoteLinkID:   "quote-gct-001",
					QuoteSide:     "Buy",
					StrategyType:  "custom",
					Status:        "Filled",
					RFQDeskCode:   "1nu9d1",
					QuoteDeskCode: "SPMAKERTEST",
					CreatedAt:     types.Time(time.UnixMilli(1791008130177)),
					UpdatedAt:     types.Time(time.UnixMilli(1791008130301)),
					Legs: []RFQTradeLeg{
						{
							Category:     "option",
							OrderID:      "64fe4108-555e-4361-ae2d-3a8d0c292859",
							Symbol:       "BTC-25DEC26-100000-C",
							Side:         "Buy",
							Price:        1250.5,
							Quantity:     10,
							MarkPrice:    121480.2,
							ExecutionFee: 7.5,
							ExecutionID:  "42b8be1e-36cf-4aba-bb75-4602cc11df37",
						},
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetRFQTradeHistory should decode every field of case 2")
	}
}

func TestGetRFQPublicTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetRFQPublicTrades(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetRFQPublicTrades must reject a nil request")
	_, err = e.GetRFQPublicTrades(t.Context(), &RFQPublicTradesRequest{StartTime: rfqTestEnd, EndTime: rfqTestStart})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetRFQPublicTrades must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetRFQPublicTrades(t.Context(), &RFQPublicTradesRequest{StartTime: rfqTestStart, EndTime: rfqTestEnd, Limit: 2, Cursor: rfqTestCursor})
	require.NoError(t, err, "GetRFQPublicTrades must not error")
	if mockTests {
		exp := &RFQPublicTradesResponse{
			Cursor: "page_token%3D14912%26last_time%3D1791008130301000000%26",
			List: []RFQPublicTrade{
				{
					RFQID:        "1791008100511241492396882271696580",
					StrategyType: "custom",
					CreatedAt:    types.Time(time.UnixMilli(1791008130177)),
					UpdatedAt:    types.Time(time.UnixMilli(1791008130301)),
					Legs: []RFQPublicTradeLeg{
						{
							Category:  "option",
							Symbol:    "BTC-25DEC26-100000-C",
							Side:      "Buy",
							Price:     1250.5,
							Quantity:  10,
							MarkPrice: 121480.2,
						},
						{
							Category:  "linear",
							Symbol:    "BTCUSDT",
							Side:      "Sell",
							Price:     121500,
							Quantity:  4.6,
							MarkPrice: 121475.33,
						},
					},
				},
				{
					RFQID:        "1791002080435210075162963643082323",
					StrategyType: "custom",
					CreatedAt:    types.Time(time.UnixMilli(1791002080437)),
					UpdatedAt:    types.Time(time.UnixMilli(1791002081550)),
					Legs: []RFQPublicTradeLeg{
						{
							Category:  "spot",
							Symbol:    "BTCUSDT",
							Side:      "Sell",
							Price:     121490,
							Quantity:  0.5,
							MarkPrice: 121492.17,
						},
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetRFQPublicTrades should decode every field")
	}
}
