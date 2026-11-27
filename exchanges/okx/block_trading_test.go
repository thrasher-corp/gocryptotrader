package okx

import (
	"net/http"
	"slices"
	"strconv"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetCounterparties(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCounterparties(contextGenerate())
	require.NoError(t, err, "GetCounterparties must not error")
	if mockTests {
		exp := []CounterpartiesResponse{{TraderName: "Satoshi Nakamoto", TraderCode: "SATOSHI", Type: "LP"}}
		assert.Equal(t, exp, result, "GetCounterparties should decode every field")
	} else {
		assert.NotNil(t, result, "GetCounterparties should return a result")
	}
}

func TestCreateRequestForQuote(t *testing.T) {
	t.Parallel()
	leg := CreateRequestForQuoteLeg{InstrumentID: mainPair.String(), Size: 1, Side: order.Buy.Lower()}
	for _, tc := range []struct {
		name string
		arg  *CreateRequestForQuoteRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "no counterparties", arg: &CreateRequestForQuoteRequest{Counterparties: []string{}}, err: errInvalidCounterParties},
		{name: "no legs", arg: &CreateRequestForQuoteRequest{Counterparties: []string{"Trader1"}}, err: errMissingLegs},
		{name: "more legs than allowed", arg: &CreateRequestForQuoteRequest{Counterparties: []string{"Trader1"}, Legs: slices.Repeat([]CreateRequestForQuoteLeg{leg}, maxRequestForQuoteLegs+1)}, err: errTooManyLegs},
		{name: "leg without an instrument", arg: &CreateRequestForQuoteRequest{Counterparties: []string{"Trader1"}, Legs: []CreateRequestForQuoteLeg{{Size: 1, Side: order.Buy.Lower()}}}, err: errMissingInstrumentID},
		{name: "leg without a size", arg: &CreateRequestForQuoteRequest{Counterparties: []string{"Trader1"}, Legs: []CreateRequestForQuoteLeg{{InstrumentID: mainPair.String(), Side: order.Buy.Lower()}}}, err: errMissingLegSize},
		{name: "leg without a side", arg: &CreateRequestForQuoteRequest{Counterparties: []string{"Trader1"}, Legs: []CreateRequestForQuoteLeg{{InstrumentID: mainPair.String(), Size: 1}}}, err: order.ErrSideIsInvalid},
		{
			name: "limit price on some legs only",
			arg: &CreateRequestForQuoteRequest{Counterparties: []string{"Trader1"}, Legs: []CreateRequestForQuoteLeg{
				{InstrumentID: mainPair.String(), Size: 1, Side: order.Buy.Lower(), LimitPrice: 60000},
				{InstrumentID: perpetualSwapPair.String(), Size: 1, Side: order.Sell.Lower()},
			}},
			err: errLegLimitPriceMismatch,
		},
		{
			name: "allocation without an account",
			arg:  &CreateRequestForQuoteRequest{Counterparties: []string{"Trader1"}, Legs: []CreateRequestForQuoteLeg{leg}, AccountAllocations: []CreateRequestForQuoteAccountAllocation{{}}},
			err:  errMissingAllocationAccount,
		},
		{
			name: "allocation without legs",
			arg:  &CreateRequestForQuoteRequest{Counterparties: []string{"Trader1"}, Legs: []CreateRequestForQuoteLeg{leg}, AccountAllocations: []CreateRequestForQuoteAccountAllocation{{Account: "acct1"}}},
			err:  errMissingLegs,
		},
		{
			name: "allocation leg without an instrument",
			arg: &CreateRequestForQuoteRequest{Counterparties: []string{"Trader1"}, Legs: []CreateRequestForQuoteLeg{leg}, AccountAllocations: []CreateRequestForQuoteAccountAllocation{
				{Account: "acct1", Legs: []CreateRequestForQuoteAllocationLeg{{Size: 1}}},
			}},
			err: errMissingInstrumentID,
		},
		{
			name: "allocation leg without a size",
			arg: &CreateRequestForQuoteRequest{Counterparties: []string{"Trader1"}, Legs: []CreateRequestForQuoteLeg{leg}, AccountAllocations: []CreateRequestForQuoteAccountAllocation{
				{Account: "acct1", Legs: []CreateRequestForQuoteAllocationLeg{{InstrumentID: mainPair.String()}}},
			}},
			err: errMissingLegSize,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.CreateRequestForQuote(t.Context(), tc.arg)
			require.ErrorIs(t, err, tc.err, "CreateRequestForQuote must reject the request")
		})
	}

	t.Run("the maximum number of legs", func(t *testing.T) {
		t.Parallel()
		ex := newEntryCountExchange(t, "legs", maxRequestForQuoteLegs, []map[string]string{{"rfqId": "22534"}})
		_, err := ex.CreateRequestForQuote(t.Context(), &CreateRequestForQuoteRequest{Counterparties: []string{"Trader1"}, Legs: slices.Repeat([]CreateRequestForQuoteLeg{leg}, maxRequestForQuoteLegs)})
		require.NoError(t, err, "CreateRequestForQuote must send the maximum number of legs")
	})

	t.Run("charges every attempt to its own limit", func(t *testing.T) {
		t.Parallel()
		var requests atomic.Int32
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			if requests.Add(1) == 1 {
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
			writeOKXData(t, w, []map[string]string{{"rfqId": "22534"}})
		}))
		pools := chargeToPools(t, ex)
		request.WithBackoff(func(int) time.Duration { return 0 })(ex.Requester)
		_, err := ex.CreateRequestForQuote(chargeContext(t), &CreateRequestForQuoteRequest{Counterparties: []string{"Trader1"}, Legs: []CreateRequestForQuoteLeg{leg}})
		require.NoError(t, err, "CreateRequestForQuote must not error")
		require.Equal(t, int32(2), requests.Load(), "the request must be sent again after the venue refused it")
		assert.Equal(t, map[request.EndpointLimit]float64{createRequestForQuoteEPL: 2}, takenTokens(pools), "CreateRequestForQuote should charge each attempt to its own limit")
	})

	t.Run("is paced by its own limit alone", func(t *testing.T) {
		t.Parallel()
		synctest.Test(t, func(t *testing.T) {
			var sent atomic.Int32
			ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				sent.Add(1)
				writeOKXData(t, w, []map[string]string{{"rfqId": "22534"}})
			}))
			request.WithLimiter(newRateLimits())(ex.Requester)
			require.NoError(t, ex.EnableRateLimiter(), "EnableRateLimiter must not error")
			rfq := &CreateRequestForQuoteRequest{Counterparties: []string{"Trader1"}, Legs: []CreateRequestForQuoteLeg{leg}}
			start := time.Now()
			for range 81 {
				_, err := ex.CreateRequestForQuote(t.Context(), rfq)
				require.NoError(t, err, "CreateRequestForQuote must not error")
			}
			assert.Equal(t, int32(81), sent.Load(), "CreateRequestForQuote should send more than OKX's 80 in 12 hours, which OKX enforces")
			assert.InDelta(t, float64(80*twoSecondsInterval/5), float64(time.Since(start)), float64(time.Millisecond), "CreateRequestForQuote should wait for its own limit of 5 every 2 seconds alone")
		})
	})

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			arg  *CreateRequestForQuoteRequest
			body string
		}{
			{
				name: "every optional parameter",
				arg: &CreateRequestForQuoteRequest{
					Counterparties:          []string{"Trader1"},
					Anonymous:               true,
					ClientRequestForQuoteID: "rfq01",
					Tag:                     "123456",
					AllowPartialExecution:   true,
					Legs: []CreateRequestForQuoteLeg{{
						InstrumentID: perpetualSwapPair.String(), TradeMode: TradeModeCross, Currency: currency.USDT, Size: 2, LimitPrice: 60000.5,
						Side: "BUY", PositionSide: positionSideLong, TargetCurrency: "base_ccy", TradeQuoteCurrency: currency.USDT,
					}},
					AccountAllocations: []CreateRequestForQuoteAccountAllocation{{Account: "acct1", Legs: []CreateRequestForQuoteAllocationLeg{
						{InstrumentID: perpetualSwapPair.String(), Size: 2, TradeMode: TradeModeCross, Currency: currency.USDT, PositionSide: positionSideLong},
					}}},
				},
				body: `{"counterparties":["Trader1"],"anonymous":true,"clRfqId":"rfq01","tag":"123456","allowPartialExecution":true,"legs":[{"instId":"BTC-USDT-SWAP","tdMode":"cross","ccy":"USDT","sz":"2","lmtPx":"60000.5","side":"buy","posSide":"long","tgtCcy":"base_ccy","tradeQuoteCcy":"USDT"}],"acctAlloc":[{"acct":"acct1","legs":[{"instId":"BTC-USDT-SWAP","sz":"2","tdMode":"cross","ccy":"USDT","posSide":"long"}]}]}`,
			},
			{
				name: "required parameters only",
				arg:  &CreateRequestForQuoteRequest{Counterparties: []string{"Trader1"}, Legs: []CreateRequestForQuoteLeg{{InstrumentID: perpetualSwapPair.String(), Size: 2, Side: order.Sell.Lower()}}},
				body: `{"counterparties":["Trader1"],"legs":[{"instId":"BTC-USDT-SWAP","sz":"2","side":"sell"}]}`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newRequestPinExchange(t, "/rfq/create-rfq", tc.body, []map[string]string{{"rfqId": "22534"}})
				_, err := ex.CreateRequestForQuote(t.Context(), tc.arg)
				require.NoError(t, err, "CreateRequestForQuote must not error")
			})
		}
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CreateRequestForQuote(contextGenerate(), &CreateRequestForQuoteRequest{
		Anonymous:               true,
		Counterparties:          []string{"Trader1", "Trader2"},
		ClientRequestForQuoteID: "rfq01",
		Tag:                     "123456",
		AllowPartialExecution:   true,
		Legs: []CreateRequestForQuoteLeg{
			{InstrumentID: "BTC-USD-221208-100000-C", TradeMode: TradeModeCross, Currency: currency.USDT, Size: 25, Side: order.Buy.Lower(), PositionSide: positionSideLong},
			{InstrumentID: mainPair.String(), TradeMode: TradeModeCash, Size: 150, Side: order.Buy.Lower(), TargetCurrency: "base_ccy", TradeQuoteCurrency: currency.USDT},
		},
		AccountAllocations: []CreateRequestForQuoteAccountAllocation{
			{Account: "acct1", Legs: []CreateRequestForQuoteAllocationLeg{
				{InstrumentID: "BTC-USD-221208-100000-C", Size: 10, TradeMode: TradeModeCross, Currency: currency.USDT, PositionSide: positionSideLong},
				{InstrumentID: mainPair.String(), Size: 60, TradeMode: TradeModeCash},
			}},
			{Account: "acct2", Legs: []CreateRequestForQuoteAllocationLeg{
				{InstrumentID: "BTC-USD-221208-100000-C", Size: 15, TradeMode: TradeModeCross, Currency: currency.USDT, PositionSide: positionSideLong},
				{InstrumentID: mainPair.String(), Size: 90, TradeMode: TradeModeCash},
			}},
		},
	})
	require.NoError(t, err, "CreateRequestForQuote must not error")
	if mockTests {
		exp := &RequestForQuoteResponse{
			CreationTime:            types.Time(time.UnixMilli(1611033737572)),
			UpdateTime:              types.Time(time.UnixMilli(1611033737572)),
			ValidUntil:              types.Time(time.UnixMilli(1611033857557)),
			TraderCode:              "SATOSHI",
			RequestForQuoteID:       "22534",
			ClientRequestForQuoteID: "rfq01",
			Tag:                     "123456",
			State:                   "active",
			Counterparties:          []string{"Trader1", "Trader2"},
			AllowPartialExecution:   true,
			GroupID:                 "1611033737572001",
			Legs: []RequestForQuoteOrderLeg{
				{InstrumentID: "BTC-USD-221208-100000-C", TradeMode: TradeModeCross, Currency: currency.USDT, Size: 25, Side: order.Buy.Lower(), PositionSide: positionSideLong},
				{InstrumentID: mainPair.String(), TradeMode: TradeModeCash, Size: 150, Side: order.Buy.Lower(), TargetCurrency: "base_ccy", TradeQuoteCurrency: currency.USDT},
			},
			AccountAllocations: []RequestForQuoteAccountAllocation{
				{Account: "acct1", StatusCode: "0", Legs: []RequestForQuoteAllocationLeg{
					{InstrumentID: "BTC-USD-221208-100000-C", Size: 10, TradeMode: TradeModeCross, Currency: currency.USDT, PositionSide: positionSideLong},
					{InstrumentID: mainPair.String(), Size: 60, TradeMode: TradeModeCash},
				}},
				{Account: "acct2", StatusCode: "0", Legs: []RequestForQuoteAllocationLeg{
					{InstrumentID: "BTC-USD-221208-100000-C", Size: 15, TradeMode: TradeModeCross, Currency: currency.USDT, PositionSide: positionSideLong},
					{InstrumentID: mainPair.String(), Size: 90, TradeMode: TradeModeCash},
				}},
			},
		}
		assert.Equal(t, exp, result, "CreateRequestForQuote should decode every field")
	} else {
		require.NotNil(t, result, "CreateRequestForQuote must return the RFQ")
		assert.NotEmpty(t, result.RequestForQuoteID, "CreateRequestForQuote should return the RFQ ID")
	}
}

func TestCreateRequestForQuoteRequestValidate(t *testing.T) {
	t.Parallel()
	legs := func(n int, side string) []CreateRequestForQuoteLeg {
		return slices.Repeat([]CreateRequestForQuoteLeg{{InstrumentID: mainPair.String(), Size: 1, Side: side}}, n)
	}
	allocated := func(allocationLegs ...CreateRequestForQuoteAllocationLeg) []CreateRequestForQuoteAccountAllocation {
		return []CreateRequestForQuoteAccountAllocation{{Account: "acct1", Legs: allocationLegs}}
	}
	counterparties := []string{"Trader1"}
	for _, tc := range []struct {
		name string
		arg  *CreateRequestForQuoteRequest
		exp  *CreateRequestForQuoteRequest
		err  error
	}{
		{name: "no counterparties", arg: &CreateRequestForQuoteRequest{Legs: legs(1, "buy")}, err: errInvalidCounterParties},
		{name: "no legs", arg: &CreateRequestForQuoteRequest{Counterparties: counterparties}, err: errMissingLegs},
		{name: "legs above the maximum", arg: &CreateRequestForQuoteRequest{Counterparties: counterparties, Legs: legs(maxRequestForQuoteLegs+1, "buy")}, err: errTooManyLegs},
		{name: "leg without an instrument", arg: &CreateRequestForQuoteRequest{Counterparties: counterparties, Legs: []CreateRequestForQuoteLeg{{Size: 1, Side: "buy"}}}, err: errMissingInstrumentID},
		{name: "leg without a size", arg: &CreateRequestForQuoteRequest{Counterparties: counterparties, Legs: []CreateRequestForQuoteLeg{{InstrumentID: mainPair.String(), Side: "buy"}}}, err: errMissingLegSize},
		{name: "leg with a negative size", arg: &CreateRequestForQuoteRequest{Counterparties: counterparties, Legs: []CreateRequestForQuoteLeg{{InstrumentID: mainPair.String(), Size: -1, Side: "buy"}}}, err: errMissingLegSize},
		{name: "leg with a position side as its side", arg: &CreateRequestForQuoteRequest{Counterparties: counterparties, Legs: []CreateRequestForQuoteLeg{{InstrumentID: mainPair.String(), Size: 1, Side: "long"}}}, err: order.ErrSideIsInvalid},
		{name: "later leg without a side", arg: &CreateRequestForQuoteRequest{Counterparties: counterparties, Legs: append(legs(1, "buy"), CreateRequestForQuoteLeg{InstrumentID: mainPair.String(), Size: 1})}, err: order.ErrSideIsInvalid},
		{
			name: "limit price on some legs only",
			arg: &CreateRequestForQuoteRequest{Counterparties: counterparties, Legs: []CreateRequestForQuoteLeg{
				{InstrumentID: mainPair.String(), Size: 1, Side: "buy"},
				{InstrumentID: perpetualSwapPair.String(), Size: 1, Side: "sell", LimitPrice: 60000},
			}},
			err: errLegLimitPriceMismatch,
		},
		{
			name: "allocation without an account",
			arg:  &CreateRequestForQuoteRequest{Counterparties: counterparties, Legs: legs(1, "buy"), AccountAllocations: []CreateRequestForQuoteAccountAllocation{{Legs: []CreateRequestForQuoteAllocationLeg{{InstrumentID: mainPair.String(), Size: 1}}}}},
			err:  errMissingAllocationAccount,
		},
		{name: "allocation without legs", arg: &CreateRequestForQuoteRequest{Counterparties: counterparties, Legs: legs(1, "buy"), AccountAllocations: allocated()}, err: errMissingLegs},
		{name: "allocation leg without an instrument", arg: &CreateRequestForQuoteRequest{Counterparties: counterparties, Legs: legs(1, "buy"), AccountAllocations: allocated(CreateRequestForQuoteAllocationLeg{Size: 1})}, err: errMissingInstrumentID},
		{name: "allocation leg without a size", arg: &CreateRequestForQuoteRequest{Counterparties: counterparties, Legs: legs(1, "buy"), AccountAllocations: allocated(CreateRequestForQuoteAllocationLeg{InstrumentID: mainPair.String()})}, err: errMissingLegSize},
		{name: "allocation leg with a negative size", arg: &CreateRequestForQuoteRequest{Counterparties: counterparties, Legs: legs(1, "buy"), AccountAllocations: allocated(CreateRequestForQuoteAllocationLeg{InstrumentID: mainPair.String(), Size: -1})}, err: errMissingLegSize},
		{
			name: "one leg with an upper case side",
			arg:  &CreateRequestForQuoteRequest{Counterparties: counterparties, Legs: legs(1, "BUY")},
			exp:  &CreateRequestForQuoteRequest{Counterparties: counterparties, Legs: legs(1, "buy")},
		},
		{
			name: "legs at the maximum",
			arg:  &CreateRequestForQuoteRequest{Counterparties: counterparties, Legs: legs(maxRequestForQuoteLegs, "Sell")},
			exp:  &CreateRequestForQuoteRequest{Counterparties: counterparties, Legs: legs(maxRequestForQuoteLegs, "sell")},
		},
		{
			name: "limit price on every leg",
			arg: &CreateRequestForQuoteRequest{Counterparties: counterparties, Legs: []CreateRequestForQuoteLeg{
				{InstrumentID: mainPair.String(), Size: 1, Side: "BUY", LimitPrice: 60000},
				{InstrumentID: perpetualSwapPair.String(), Size: 2, Side: "SELL", LimitPrice: 60001},
			}},
			exp: &CreateRequestForQuoteRequest{Counterparties: counterparties, Legs: []CreateRequestForQuoteLeg{
				{InstrumentID: mainPair.String(), Size: 1, Side: "buy", LimitPrice: 60000},
				{InstrumentID: perpetualSwapPair.String(), Size: 2, Side: "sell", LimitPrice: 60001},
			}},
		},
		{
			name: "allocations",
			arg: &CreateRequestForQuoteRequest{Counterparties: counterparties, Legs: legs(1, "Buy"), AccountAllocations: []CreateRequestForQuoteAccountAllocation{
				{Account: "acct1", Legs: []CreateRequestForQuoteAllocationLeg{{InstrumentID: mainPair.String(), Size: 0.4}}},
				{Account: "acct2", Legs: []CreateRequestForQuoteAllocationLeg{{InstrumentID: mainPair.String(), Size: 0.6, TradeMode: TradeModeCash}}},
			}},
			exp: &CreateRequestForQuoteRequest{Counterparties: counterparties, Legs: legs(1, "buy"), AccountAllocations: []CreateRequestForQuoteAccountAllocation{
				{Account: "acct1", Legs: []CreateRequestForQuoteAllocationLeg{{InstrumentID: mainPair.String(), Size: 0.4}}},
				{Account: "acct2", Legs: []CreateRequestForQuoteAllocationLeg{{InstrumentID: mainPair.String(), Size: 0.6, TradeMode: TradeModeCash}}},
			}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.arg.validate()
			require.ErrorIs(t, err, tc.err, "validate must return the expected error")
			if tc.err == nil {
				assert.Equal(t, tc.exp, tc.arg, "validate should lower case each leg's side and leave the rest of the request unchanged")
			}
		})
	}
}

func TestCancelRequestForQuote(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *CancelRequestForQuoteRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "no RFQ ID", arg: &CancelRequestForQuoteRequest{}, err: order.ErrOrderIDNotSet},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.CancelRequestForQuote(t.Context(), tc.arg)
			require.ErrorIs(t, err, tc.err, "CancelRequestForQuote must reject the request")
		})
	}

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			arg  *CancelRequestForQuoteRequest
			body string
		}{
			{name: "RFQ ID", arg: &CancelRequestForQuoteRequest{RequestForQuoteID: "22535"}, body: `{"rfqId":"22535"}`},
			{name: "client RFQ ID", arg: &CancelRequestForQuoteRequest{ClientRequestForQuoteID: "rfq001"}, body: `{"clRfqId":"rfq001"}`},
			{name: "both IDs", arg: &CancelRequestForQuoteRequest{RequestForQuoteID: "22535", ClientRequestForQuoteID: "rfq001"}, body: `{"rfqId":"22535","clRfqId":"rfq001"}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newRequestPinExchange(t, "/rfq/cancel-rfq", tc.body, []map[string]string{{"rfqId": "22535", "sCode": "0"}})
				_, err := ex.CancelRequestForQuote(t.Context(), tc.arg)
				require.NoError(t, err, "CancelRequestForQuote must not error")
			})
		}
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelRequestForQuote(contextGenerate(), &CancelRequestForQuoteRequest{ClientRequestForQuoteID: "somersdjskfjsdkfjxvxv"})
	require.NoError(t, err, "CancelRequestForQuote must not error")
	if mockTests {
		exp := &CancelRequestForQuoteResponse{RequestForQuoteID: "22535", ClientRequestForQuoteID: "somersdjskfjsdkfjxvxv", StatusCode: "0"}
		assert.Equal(t, exp, result, "CancelRequestForQuote should decode every field")
	} else {
		assert.NotNil(t, result, "CancelRequestForQuote should return a result")
	}
}

func TestCancelMultipleRequestForQuotes(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *CancelMultipleRequestForQuotesRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "no RFQ IDs", arg: &CancelMultipleRequestForQuotesRequest{}, err: order.ErrOrderIDNotSet},
		{name: "one more RFQ ID than allowed", arg: &CancelMultipleRequestForQuotesRequest{RequestForQuoteIDs: make([]string, maxRequestForQuoteCancelIDs+1)}, err: errTooManyCancelIDs},
		{name: "IDs of both kinds over the maximum", arg: &CancelMultipleRequestForQuotesRequest{RequestForQuoteIDs: make([]string, 100), ClientRequestForQuoteIDs: make([]string, 100)}, err: errTooManyCancelIDs},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.CancelMultipleRequestForQuotes(t.Context(), tc.arg)
			require.ErrorIs(t, err, tc.err, "CancelMultipleRequestForQuotes must reject the request")
		})
	}

	t.Run("the maximum number of RFQ IDs", func(t *testing.T) {
		t.Parallel()
		ex := newEntryCountExchange(t, "rfqIds", maxRequestForQuoteCancelIDs, []map[string]string{{"rfqId": "2201", "sCode": "0"}})
		_, err := ex.CancelMultipleRequestForQuotes(t.Context(), &CancelMultipleRequestForQuotesRequest{RequestForQuoteIDs: make([]string, maxRequestForQuoteCancelIDs)})
		require.NoError(t, err, "CancelMultipleRequestForQuotes must send the maximum number of RFQ IDs")
	})

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			arg  *CancelMultipleRequestForQuotesRequest
			body string
		}{
			{name: "RFQ IDs", arg: &CancelMultipleRequestForQuotesRequest{RequestForQuoteIDs: []string{"2201", "2202"}}, body: `{"rfqIds":["2201","2202"]}`},
			{name: "client RFQ IDs", arg: &CancelMultipleRequestForQuotesRequest{ClientRequestForQuoteIDs: []string{"somersdjskfjsdkfjxvxv"}}, body: `{"clRfqIds":["somersdjskfjsdkfjxvxv"]}`},
			{name: "both kinds of ID", arg: &CancelMultipleRequestForQuotesRequest{RequestForQuoteIDs: []string{"2201"}, ClientRequestForQuoteIDs: []string{"r1"}}, body: `{"rfqIds":["2201"],"clRfqIds":["r1"]}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newRequestPinExchange(t, "/rfq/cancel-batch-rfqs", tc.body, []map[string]string{{"rfqId": "2201", "sCode": "0"}})
				_, err := ex.CancelMultipleRequestForQuotes(t.Context(), tc.arg)
				require.NoError(t, err, "CancelMultipleRequestForQuotes must not error")
			})
		}
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelMultipleRequestForQuotes(contextGenerate(), &CancelMultipleRequestForQuotesRequest{RequestForQuoteIDs: []string{"2201", "2202", "2203"}})
	if mockTests {
		require.ErrorIs(t, err, errPartialSuccess, "CancelMultipleRequestForQuotes must report a partial cancellation")
		exp := []CancelRequestForQuoteResponse{
			{RequestForQuoteID: "2201", ClientRequestForQuoteID: "r1", StatusCode: "70000", StatusMessage: "RFQ does not exist."},
			{RequestForQuoteID: "2202", ClientRequestForQuoteID: "r2", StatusCode: "0"},
			{RequestForQuoteID: "2203", ClientRequestForQuoteID: "r3", StatusCode: "0"},
		}
		assert.Equal(t, exp, result, "CancelMultipleRequestForQuotes should decode every row of a partial cancellation")
		return
	}
	require.NoError(t, err, "CancelMultipleRequestForQuotes must not error")
	assert.NotEmpty(t, result, "CancelMultipleRequestForQuotes should return a row per RFQ")
}

func TestCancelAllRequestForQuotes(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelAllRequestForQuotes(contextGenerate())
	require.NoError(t, err, "CancelAllRequestForQuotes must not error")
	if mockTests {
		exp := &BlockTradingTimestampResponse{Timestamp: types.Time(time.UnixMilli(1697026383085))}
		assert.Equal(t, exp, result, "CancelAllRequestForQuotes should decode every field")
	} else {
		require.NotNil(t, result, "CancelAllRequestForQuotes must return a result")
		assert.NotZero(t, result.Timestamp, "CancelAllRequestForQuotes should return the cancellation time")
	}
}

func TestExecuteQuote(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *ExecuteQuoteRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "no RFQ ID", arg: &ExecuteQuoteRequest{}, err: errMissingRequestForQuoteID},
		{name: "no quote ID", arg: &ExecuteQuoteRequest{RequestForQuoteID: "22540"}, err: errMissingQuoteID},
		{name: "leg without an instrument", arg: &ExecuteQuoteRequest{RequestForQuoteID: "22540", QuoteID: "84073", Legs: []ExecuteQuoteLeg{{Size: 1}}}, err: errMissingInstrumentID},
		{name: "leg without a size", arg: &ExecuteQuoteRequest{RequestForQuoteID: "22540", QuoteID: "84073", Legs: []ExecuteQuoteLeg{{InstrumentID: mainPair.String()}}}, err: errMissingLegSize},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.ExecuteQuote(t.Context(), tc.arg)
			require.ErrorIs(t, err, tc.err, "ExecuteQuote must reject the request")
		})
	}

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			arg  *ExecuteQuoteRequest
			body string
		}{
			{name: "partial execution legs", arg: &ExecuteQuoteRequest{RequestForQuoteID: "22540", QuoteID: "84073", Legs: []ExecuteQuoteLeg{{InstrumentID: mainPair.String(), Size: 12.5}}}, body: `{"rfqId":"22540","quoteId":"84073","legs":[{"instId":"BTC-USDT","sz":"12.5"}]}`},
			{name: "required parameters only", arg: &ExecuteQuoteRequest{RequestForQuoteID: "22540", QuoteID: "84073"}, body: `{"rfqId":"22540","quoteId":"84073"}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newRequestPinExchange(t, "/rfq/execute-quote", tc.body, []map[string]string{{"blockTdId": "180184"}})
				_, err := ex.ExecuteQuote(t.Context(), tc.arg)
				require.NoError(t, err, "ExecuteQuote must not error")
			})
		}
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.ExecuteQuote(contextGenerate(), &ExecuteQuoteRequest{RequestForQuoteID: "22540", QuoteID: "84073"})
	require.NoError(t, err, "ExecuteQuote must not error")
	if mockTests {
		exp := &ExecuteQuoteResponse{
			CreationTime:            types.Time(time.UnixMilli(1649670009000)),
			RequestForQuoteID:       "22540",
			ClientRequestForQuoteID: "r0001",
			QuoteID:                 "84073",
			ClientQuoteID:           "q0001",
			BlockTradeID:            "180184",
			Tag:                     "123456",
			TakerTraderCode:         "Trader1",
			MakerTraderCode:         "Trader2",
			Legs: []BlockTradeLeg{
				{InstrumentID: "BTC-USD-220114-13250-C", Price: 0.1, Size: 25, Side: order.Sell.Lower(), Fee: -1.001, FeeCurrency: currency.BTC, TradeID: "10211"},
				{InstrumentID: mainPair.String(), Price: 0.2, Size: 25, Side: order.Buy.Lower(), Fee: -1.001, FeeCurrency: currency.BTC, TradeID: "10212"},
			},
			AccountAllocations: []RequestForQuoteAccountAllocation{
				{Account: "acct1", BlockTradeID: "180184", StatusCode: "0", Legs: []RequestForQuoteAllocationLeg{
					{InstrumentID: "BTC-USD-220114-13250-C", Size: 25, Fee: -1.001, FeeCurrency: currency.BTC, TradeID: "10211"},
					{InstrumentID: mainPair.String(), Size: 25, Fee: -1.001, FeeCurrency: currency.BTC, TradeID: "10212"},
				}},
				{Account: "acct2", StatusCode: "70006", StatusMessage: "Does not meet the minimum asset requirement.", Legs: []RequestForQuoteAllocationLeg{}},
			},
		}
		assert.Equal(t, exp, result, "ExecuteQuote should decode every field")
	} else {
		require.NotNil(t, result, "ExecuteQuote must return the block trade")
		assert.NotEmpty(t, result.BlockTradeID, "ExecuteQuote should return the block trade ID")
	}
}

func TestGetQuoteProducts(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetQuoteProducts(contextGenerate())
	require.NoError(t, err, "GetQuoteProducts must not error")
	if mockTests {
		exp := []QuoteProductResponse{
			{InstrumentType: instTypeOption, IncludeAll: true, Data: []QuoteProductDetail{
				{InstrumentFamily: "BTC-USD", MaxBlockSize: 10000, MakerPriceBand: 5},
				{InstrumentFamily: "SOL-USD", MaxBlockSize: 100000, MakerPriceBand: 15},
			}},
			{InstrumentType: instTypeFutures, Data: []QuoteProductDetail{
				{InstrumentFamily: "BTC-USD", MaxBlockSize: 10000, MakerPriceBand: 5},
				{InstrumentFamily: "ETH-USDT", MaxBlockSize: 100000, MakerPriceBand: 15},
			}},
			{InstrumentType: instTypeSwap, Data: []QuoteProductDetail{
				{InstrumentFamily: "BTC-USD", MaxBlockSize: 10000, MakerPriceBand: 5},
				{InstrumentFamily: "ETH-USDT"},
			}},
			{InstrumentType: instTypeSpot, Data: []QuoteProductDetail{{InstrumentID: mainPair.String()}, {InstrumentID: "TRX-USDT"}}},
		}
		assert.Equal(t, exp, result, "GetQuoteProducts should decode every field")
	} else {
		assert.NotNil(t, result, "GetQuoteProducts should return a result")
	}
}

func TestSetQuoteProducts(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		args []SetQuoteProductRequest
		err  error
	}{
		{name: "no products", args: []SetQuoteProductRequest{}, err: common.ErrEmptyParams},
		{name: "unknown instrument type", args: []SetQuoteProductRequest{{InstrumentType: "ABC"}}, err: errInvalidInstrumentType},
		{name: "no instrument settings", args: []SetQuoteProductRequest{{InstrumentType: "SWAP"}}, err: errMissingMakerInstrumentSettings},
		{name: "swap setting without an instrument family", args: []SetQuoteProductRequest{{InstrumentType: "SWAP", Data: []MakerInstrumentSetting{{MaxBlockSize: 10000, MakerPriceBand: 5}}}}, err: errInstrumentFamilyRequired},
		{name: "spot setting without an instrument ID", args: []SetQuoteProductRequest{{InstrumentType: "SPOT", Data: []MakerInstrumentSetting{{InstrumentFamily: "BTC-USD", MaxBlockSize: 10000, MakerPriceBand: 5}}}}, err: errMissingInstrumentID},
		{
			name: "each group's own settings are checked",
			args: []SetQuoteProductRequest{
				{InstrumentType: "SWAP", Data: []MakerInstrumentSetting{{InstrumentFamily: "BTC-USD"}}},
				{InstrumentType: "SPOT", Data: []MakerInstrumentSetting{{InstrumentFamily: "ETH-USDT"}}},
			},
			err: errMissingInstrumentID,
		},
		{
			name: "a later group missing its family",
			args: []SetQuoteProductRequest{
				{InstrumentType: "SPOT", Data: []MakerInstrumentSetting{{InstrumentID: mainPair.String()}}},
				{InstrumentType: "SWAP", Data: []MakerInstrumentSetting{{}}},
			},
			err: errInstrumentFamilyRequired,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.SetQuoteProducts(t.Context(), tc.args)
			require.ErrorIs(t, err, tc.err, "SetQuoteProducts must reject the settings")
		})
	}

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			args []SetQuoteProductRequest
			body string
		}{
			{
				name: "every optional parameter",
				args: []SetQuoteProductRequest{
					{InstrumentType: "option", IncludeAll: true, Data: []MakerInstrumentSetting{{InstrumentFamily: "BTC-USD", MaxBlockSize: 10000, MakerPriceBand: 5}}},
					{InstrumentType: "SPOT", Data: []MakerInstrumentSetting{{InstrumentID: mainPair.String(), MaxBlockSize: 2.5, MakerPriceBand: 15}}},
				},
				body: `[{"instType":"OPTION","includeAll":true,"data":[{"instFamily":"BTC-USD","maxBlockSz":"10000","makerPxBand":"5"}]},{"instType":"SPOT","data":[{"instId":"BTC-USDT","maxBlockSz":"2.5","makerPxBand":"15"}]}]`,
			},
			{
				name: "required parameters only",
				args: []SetQuoteProductRequest{{InstrumentType: "FUTURES", Data: []MakerInstrumentSetting{{InstrumentFamily: "BTC-USD"}}}},
				body: `[{"instType":"FUTURES","data":[{"instFamily":"BTC-USD"}]}]`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newRequestPinExchange(t, "/rfq/maker-instrument-settings", tc.body, []map[string]bool{{"result": true}})
				_, err := ex.SetQuoteProducts(t.Context(), tc.args)
				require.NoError(t, err, "SetQuoteProducts must not error")
			})
		}
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SetQuoteProducts(contextGenerate(), []SetQuoteProductRequest{
		{
			InstrumentType: "SWAP",
			Data: []MakerInstrumentSetting{
				{InstrumentFamily: "BTC-USD", MaxBlockSize: 10000, MakerPriceBand: 5},
				{InstrumentFamily: mainPair.String()},
			},
		},
	})
	require.NoError(t, err, "SetQuoteProducts must not error")
	if mockTests {
		assert.Equal(t, &SetQuoteProductsResponse{Result: true}, result, "SetQuoteProducts should decode every field")
	} else {
		require.NotNil(t, result, "SetQuoteProducts must return a result")
		assert.True(t, result.Result, "SetQuoteProducts should report success")
	}
}

func TestResetRequestForQuoteMMPStatus(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.ResetRequestForQuoteMMPStatus(contextGenerate())
	require.NoError(t, err, "ResetRequestForQuoteMMPStatus must not error")
	if mockTests {
		exp := &BlockTradingTimestampResponse{Timestamp: types.Time(time.UnixMilli(1597026383085))}
		assert.Equal(t, exp, result, "ResetRequestForQuoteMMPStatus should decode every field")
	} else {
		require.NotNil(t, result, "ResetRequestForQuoteMMPStatus must return a result")
		assert.NotZero(t, result.Timestamp, "ResetRequestForQuoteMMPStatus should return the reset time")
	}
}

func TestSetRequestForQuoteMMP(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *SetRequestForQuoteMMPRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "time interval over the maximum", arg: &SetRequestForQuoteMMPRequest{TimeInterval: maxRequestForQuoteMMPTimeInterval + 1, FrozenInterval: 2000, CountLimit: 100}, err: errInvalidMMPTimeInterval},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.SetRequestForQuoteMMP(t.Context(), tc.arg)
			require.ErrorIs(t, err, tc.err, "SetRequestForQuoteMMP must reject the request")
		})
	}

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			arg  *SetRequestForQuoteMMPRequest
			body string
		}{
			{name: "maximum time interval", arg: &SetRequestForQuoteMMPRequest{TimeInterval: maxRequestForQuoteMMPTimeInterval, FrozenInterval: 2000, CountLimit: 100}, body: `{"timeInterval":"600000","frozenInterval":"2000","countLimit":"100"}`},
			{name: "disabled", arg: &SetRequestForQuoteMMPRequest{}, body: `{"timeInterval":"0","frozenInterval":"0","countLimit":"0"}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newRequestPinExchange(t, "/rfq/mmp-config", tc.body, []map[string]string{{"timeInterval": "0"}})
				_, err := ex.SetRequestForQuoteMMP(t.Context(), tc.arg)
				require.NoError(t, err, "SetRequestForQuoteMMP must not error")
			})
		}
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SetRequestForQuoteMMP(contextGenerate(), &SetRequestForQuoteMMPRequest{TimeInterval: 5000, FrozenInterval: 2000, CountLimit: 100})
	require.NoError(t, err, "SetRequestForQuoteMMP must not error")
	if mockTests {
		exp := &SetRequestForQuoteMMPResponse{TimeInterval: 5000, FrozenInterval: 2000, CountLimit: 100}
		assert.Equal(t, exp, result, "SetRequestForQuoteMMP should decode every field")
	} else {
		require.NotNil(t, result, "SetRequestForQuoteMMP must return the configuration")
		assert.Equal(t, 5000.0, result.TimeInterval.Float64(), "SetRequestForQuoteMMP should return the time interval set")
	}
}

func TestGetRequestForQuoteMMPConfig(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetRequestForQuoteMMPConfig(contextGenerate())
	require.NoError(t, err, "GetRequestForQuoteMMPConfig must not error")
	if mockTests {
		exp := []RequestForQuoteMMPConfigResponse{{TimeInterval: 5000, FrozenInterval: 2000, CountLimit: 10, MMPFrozen: true, MMPFrozenUntil: 1000}}
		assert.Equal(t, exp, result, "GetRequestForQuoteMMPConfig should decode every field")
	} else {
		assert.NotNil(t, result, "GetRequestForQuoteMMPConfig should return a result")
	}
}

func TestCreateQuote(t *testing.T) {
	t.Parallel()
	validLeg := CreateQuoteLeg{InstrumentID: "SOL-USD-220909", Size: 2, Price: 1234, Side: order.Sell.Lower()}
	for _, tc := range []struct {
		name string
		arg  *CreateQuoteRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "no RFQ ID", arg: &CreateQuoteRequest{}, err: errMissingRequestForQuoteID},
		{name: "no quote side", arg: &CreateQuoteRequest{RequestForQuoteID: "123456789"}, err: order.ErrSideIsInvalid},
		{name: "expiry under the minimum", arg: &CreateQuoteRequest{RequestForQuoteID: "123456789", QuoteSide: "sell", ExpiresIn: minQuoteExpirySeconds - 1, Legs: []CreateQuoteLeg{validLeg}}, err: errInvalidQuoteExpiry},
		{name: "expiry over the maximum", arg: &CreateQuoteRequest{RequestForQuoteID: "123456789", QuoteSide: "sell", ExpiresIn: maxQuoteExpirySeconds + 1, Legs: []CreateQuoteLeg{validLeg}}, err: errInvalidQuoteExpiry},
		{name: "no legs", arg: &CreateQuoteRequest{RequestForQuoteID: "123456789", QuoteSide: "sell"}, err: errMissingLegs},
		{name: "leg without an instrument", arg: &CreateQuoteRequest{RequestForQuoteID: "123456789", QuoteSide: "sell", Legs: []CreateQuoteLeg{{}}}, err: errMissingInstrumentID},
		{name: "leg without a size", arg: &CreateQuoteRequest{RequestForQuoteID: "123456789", QuoteSide: "sell", Legs: []CreateQuoteLeg{{InstrumentID: "SOL-USD-220909"}}}, err: errMissingLegSize},
		{name: "leg without a price", arg: &CreateQuoteRequest{RequestForQuoteID: "123456789", QuoteSide: "sell", Legs: []CreateQuoteLeg{{InstrumentID: "SOL-USD-220909", Size: 2}}}, err: errMissingLegPrice},
		{name: "leg without a side", arg: &CreateQuoteRequest{RequestForQuoteID: "123456789", QuoteSide: "sell", Legs: []CreateQuoteLeg{{InstrumentID: "SOL-USD-220909", Size: 2, Price: 1234}}}, err: order.ErrSideIsInvalid},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.CreateQuote(t.Context(), tc.arg)
			require.ErrorIs(t, err, tc.err, "CreateQuote must reject the request")
		})
	}

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			arg  *CreateQuoteRequest
			body string
		}{
			{
				name: "every optional parameter",
				arg: &CreateQuoteRequest{
					RequestForQuoteID: "22540", ClientQuoteID: "q0001", Tag: "123456", Anonymous: true, QuoteSide: "BUY", ExpiresIn: maxQuoteExpirySeconds,
					Legs: []CreateQuoteLeg{{
						InstrumentID: perpetualSwapPair.String(), TradeMode: TradeModeCross, Currency: currency.USDT, Size: 2, Price: 60000.5,
						Side: "Sell", PositionSide: positionSideShort, TargetCurrency: "base_ccy", TradeQuoteCurrency: currency.USDT,
					}},
				},
				body: `{"rfqId":"22540","clQuoteId":"q0001","tag":"123456","anonymous":true,"quoteSide":"buy","expiresIn":"120","legs":[{"instId":"BTC-USDT-SWAP","tdMode":"cross","ccy":"USDT","sz":"2","px":"60000.5","side":"sell","posSide":"short","tgtCcy":"base_ccy","tradeQuoteCcy":"USDT"}]}`,
			},
			{
				name: "minimum expiry",
				arg:  &CreateQuoteRequest{RequestForQuoteID: "22540", QuoteSide: "sell", ExpiresIn: minQuoteExpirySeconds, Legs: []CreateQuoteLeg{validLeg}},
				body: `{"rfqId":"22540","quoteSide":"sell","expiresIn":"10","legs":[{"instId":"SOL-USD-220909","sz":"2","px":"1234","side":"sell"}]}`,
			},
			{
				name: "required parameters only",
				arg:  &CreateQuoteRequest{RequestForQuoteID: "22540", QuoteSide: "sell", Legs: []CreateQuoteLeg{validLeg}},
				body: `{"rfqId":"22540","quoteSide":"sell","legs":[{"instId":"SOL-USD-220909","sz":"2","px":"1234","side":"sell"}]}`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newRequestPinExchange(t, "/rfq/create-quote", tc.body, []map[string]string{{"quoteId": "25092"}})
				_, err := ex.CreateQuote(t.Context(), tc.arg)
				require.NoError(t, err, "CreateQuote must not error")
			})
		}
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CreateQuote(contextGenerate(), &CreateQuoteRequest{
		RequestForQuoteID: "12345",
		QuoteSide:         order.Buy.Lower(),
		Legs: []CreateQuoteLeg{
			{Price: 1234, Size: 2, InstrumentID: "SOL-USD-220909", Side: order.Sell.Lower()},
			{Price: 1234, Size: 1, InstrumentID: "SOL-USD-220909", Side: order.Buy.Lower()},
		},
	})
	require.NoError(t, err, "CreateQuote must not error")
	if mockTests {
		exp := &QuoteResponse{
			CreationTime:      types.Time(time.UnixMilli(1608267227834)),
			UpdateTime:        types.Time(time.UnixMilli(1608267227834)),
			ValidUntil:        types.Time(time.UnixMilli(1608267287834)),
			QuoteID:           "25092",
			RequestForQuoteID: "12345",
			QuoteSide:         order.Buy.Lower(),
			TraderCode:        "Aksha",
			State:             "active",
			Legs: []QuoteLeg{
				{InstrumentID: "SOL-USD-220909", TradeMode: TradeModeCross, Currency: currency.USDT, Size: 2, Price: 1234, Side: order.Sell.Lower(), PositionSide: positionSideLong},
				{InstrumentID: "SOL-USD-220909", TradeMode: TradeModeCross, Currency: currency.USDT, Size: 1, Price: 1234, Side: order.Buy.Lower(), PositionSide: positionSideLong},
			},
		}
		assert.Equal(t, exp, result, "CreateQuote should decode every field")
	} else {
		require.NotNil(t, result, "CreateQuote must return the quote")
		assert.NotEmpty(t, result.QuoteID, "CreateQuote should return the quote ID")
	}
}

func TestCancelQuote(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *CancelQuoteRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "no quote ID", arg: &CancelQuoteRequest{RequestForQuoteID: "22540"}, err: order.ErrOrderIDNotSet},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.CancelQuote(t.Context(), tc.arg)
			require.ErrorIs(t, err, tc.err, "CancelQuote must reject the request")
		})
	}

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			arg  *CancelQuoteRequest
			body string
		}{
			{name: "quote ID", arg: &CancelQuoteRequest{QuoteID: "1234"}, body: `{"quoteId":"1234"}`},
			{name: "client quote ID", arg: &CancelQuoteRequest{ClientQuoteID: "1234"}, body: `{"clQuoteId":"1234"}`},
			{name: "every parameter", arg: &CancelQuoteRequest{QuoteID: "1234", ClientQuoteID: "Bond007", RequestForQuoteID: "22540"}, body: `{"quoteId":"1234","clQuoteId":"Bond007","rfqId":"22540"}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newRequestPinExchange(t, "/rfq/cancel-quote", tc.body, []map[string]string{{"quoteId": "1234", "sCode": "0"}})
				_, err := ex.CancelQuote(t.Context(), tc.arg)
				require.NoError(t, err, "CancelQuote must not error")
			})
		}
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelQuote(contextGenerate(), &CancelQuoteRequest{QuoteID: "1234"})
	require.NoError(t, err, "CancelQuote must not error")
	if mockTests {
		exp := &CancelQuoteResponse{QuoteID: "1234", ClientQuoteID: "Bond007", StatusCode: "0"}
		assert.Equal(t, exp, result, "CancelQuote should decode every field")
	} else {
		assert.NotNil(t, result, "CancelQuote should return a result")
	}
}

func TestCancelMultipleQuotes(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *CancelMultipleQuotesRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "no quote IDs", arg: &CancelMultipleQuotesRequest{}, err: order.ErrOrderIDNotSet},
		{name: "one more quote ID than allowed", arg: &CancelMultipleQuotesRequest{QuoteIDs: make([]string, maxRequestForQuoteCancelIDs-1), ClientQuoteIDs: make([]string, 2)}, err: errTooManyCancelIDs},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.CancelMultipleQuotes(t.Context(), tc.arg)
			require.ErrorIs(t, err, tc.err, "CancelMultipleQuotes must reject the request")
		})
	}

	t.Run("the maximum number of quote IDs", func(t *testing.T) {
		t.Parallel()
		ex := newEntryCountExchange(t, "clQuoteIds", maxRequestForQuoteCancelIDs, []map[string]string{{"quoteId": "1150", "sCode": "0"}})
		_, err := ex.CancelMultipleQuotes(t.Context(), &CancelMultipleQuotesRequest{ClientQuoteIDs: make([]string, maxRequestForQuoteCancelIDs)})
		require.NoError(t, err, "CancelMultipleQuotes must send the maximum number of quote IDs")
	})

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			arg  *CancelMultipleQuotesRequest
			body string
		}{
			{name: "quote IDs", arg: &CancelMultipleQuotesRequest{QuoteIDs: []string{"1150", "1151"}}, body: `{"quoteIds":["1150","1151"]}`},
			{name: "client quote IDs", arg: &CancelMultipleQuotesRequest{ClientQuoteIDs: []string{"q1"}}, body: `{"clQuoteIds":["q1"]}`},
			{name: "both kinds of ID", arg: &CancelMultipleQuotesRequest{QuoteIDs: []string{"1150"}, ClientQuoteIDs: []string{"q1"}}, body: `{"quoteIds":["1150"],"clQuoteIds":["q1"]}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newRequestPinExchange(t, "/rfq/cancel-batch-quotes", tc.body, []map[string]string{{"quoteId": "1150", "sCode": "0"}})
				_, err := ex.CancelMultipleQuotes(t.Context(), tc.arg)
				require.NoError(t, err, "CancelMultipleQuotes must not error")
			})
		}
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelMultipleQuotes(contextGenerate(), &CancelMultipleQuotesRequest{QuoteIDs: []string{"1150", "1151", "1152"}})
	if mockTests {
		require.ErrorIs(t, err, errPartialSuccess, "CancelMultipleQuotes must report a partial cancellation")
		exp := []CancelQuoteResponse{
			{QuoteID: "1150", ClientQuoteID: "q1", StatusCode: "0"},
			{QuoteID: "1151", ClientQuoteID: "q2", StatusCode: "70001", StatusMessage: "Quote does not exist."},
			{QuoteID: "1152", ClientQuoteID: "q3", StatusCode: "70001", StatusMessage: "Quote does not exist."},
		}
		assert.Equal(t, exp, result, "CancelMultipleQuotes should decode every row of a partial cancellation")
		return
	}
	require.NoError(t, err, "CancelMultipleQuotes must not error")
	assert.NotEmpty(t, result, "CancelMultipleQuotes should return a row per quote")
}

func TestCancelAllQuotes(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelAllQuotes(contextGenerate())
	require.NoError(t, err, "CancelAllQuotes must not error")
	if mockTests {
		exp := &BlockTradingTimestampResponse{Timestamp: types.Time(time.UnixMilli(1697026383085))}
		assert.Equal(t, exp, result, "CancelAllQuotes should decode every field")
	} else {
		require.NotNil(t, result, "CancelAllQuotes must return a result")
		assert.NotZero(t, result.Timestamp, "CancelAllQuotes should return the cancellation time")
	}
}

func TestCancelAllQuotesAfterCountdown(t *testing.T) {
	t.Parallel()
	for _, timeout := range []time.Duration{-10 * time.Second, 2 * time.Second, 10*time.Second - time.Millisecond, 120*time.Second + time.Millisecond} {
		t.Run(timeout.String(), func(t *testing.T) {
			t.Parallel()
			_, err := e.CancelAllQuotesAfterCountdown(t.Context(), timeout)
			require.ErrorIs(t, err, errCountdownTimeoutRequired, "CancelAllQuotesAfterCountdown must reject the timeout")
		})
	}

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			timeout time.Duration
			body    string
		}{
			{0, `{"timeOut":"0"}`},
			{10 * time.Second, `{"timeOut":"10"}`},
			{30*time.Second + 500*time.Millisecond, `{"timeOut":"30"}`},
			{120 * time.Second, `{"timeOut":"120"}`},
		} {
			t.Run(tc.timeout.String(), func(t *testing.T) {
				t.Parallel()
				ex := newRequestPinExchange(t, "/rfq/cancel-all-after", tc.body, []map[string]string{{"triggerTime": "0", "ts": "1587971400000"}})
				_, err := ex.CancelAllQuotesAfterCountdown(t.Context(), tc.timeout)
				require.NoError(t, err, "CancelAllQuotesAfterCountdown must not error")
			})
		}
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelAllQuotesAfterCountdown(contextGenerate(), time.Minute)
	require.NoError(t, err, "CancelAllQuotesAfterCountdown must not error")
	if mockTests {
		exp := &CancelAllQuotesAfterCountdownResponse{TriggerTime: types.Time(time.Unix(1587971460, 0)), Timestamp: types.Time(time.Unix(1587971400, 0))}
		assert.Equal(t, exp, result, "CancelAllQuotesAfterCountdown should decode every field")
	} else {
		require.NotNil(t, result, "CancelAllQuotesAfterCountdown must return a result")
		assert.True(t, result.TriggerTime.Time().After(result.Timestamp.Time()), "CancelAllQuotesAfterCountdown should trigger after the request")
	}
}

func TestGetRequestForQuotes(t *testing.T) {
	t.Parallel()
	_, err := e.GetRequestForQuotes(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetRequestForQuotes must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetRequestForQuotes(contextGenerate(), &RequestForQuotesRequest{
		RequestForQuoteID:       "123456",
		ClientRequestForQuoteID: "r123456",
		State:                   "FILLED",
		BeginID:                 "123455",
		EndID:                   "123457",
		Limit:                   1,
	})
	require.NoError(t, err, "GetRequestForQuotes must not error")
	if !mockTests {
		assert.NotNil(t, result, "GetRequestForQuotes should return a result")
		return
	}
	exp := []RequestForQuoteResponse{{
		CreationTime:            types.Time(time.UnixMilli(1650968131817)),
		UpdateTime:              types.Time(time.UnixMilli(1650968164944)),
		ValidUntil:              types.Time(time.UnixMilli(1650969031817)),
		TraderCode:              "VITALIK",
		RequestForQuoteID:       "123456",
		ClientRequestForQuoteID: "r123456",
		Tag:                     "123456",
		State:                   "filled",
		Counterparties:          []string{"SATOSHI"},
		Legs: []RequestForQuoteOrderLeg{
			{InstrumentID: mainPair.String(), TradeMode: TradeModeCross, Currency: currency.USDT, Size: 25, Side: order.Buy.Lower(), PositionSide: positionSideLong, TargetCurrency: "base_ccy", TradeQuoteCurrency: currency.USDT},
		},
	}}
	assert.Equal(t, exp, result, "GetRequestForQuotes should decode every field")
}

func TestGetQuotes(t *testing.T) {
	t.Parallel()
	_, err := e.GetQuotes(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetQuotes must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetQuotes(contextGenerate(), &QuotesRequest{
		RequestForQuoteID:       "18753",
		ClientRequestForQuoteID: "cr001",
		QuoteID:                 "25092",
		ClientQuoteID:           "cq001",
		State:                   "Canceled",
		BeginID:                 "25091",
		EndID:                   "25093",
		Limit:                   3,
	})
	require.NoError(t, err, "GetQuotes must not error")
	if !mockTests {
		assert.NotNil(t, result, "GetQuotes should return a result")
		return
	}
	exp := []QuoteResponse{{
		CreationTime:            types.Time(time.UnixMilli(1608267227834)),
		UpdateTime:              types.Time(time.UnixMilli(1608267227834)),
		ValidUntil:              types.Time(time.UnixMilli(1608997227834)),
		QuoteID:                 "25092",
		ClientQuoteID:           "cq001",
		RequestForQuoteID:       "18753",
		ClientRequestForQuoteID: "cr001",
		QuoteSide:               order.Sell.Lower(),
		TraderCode:              "Trader1",
		State:                   "canceled",
		Reason:                  "mmp_canceled",
		Tag:                     "123456",
		Legs: []QuoteLeg{
			{InstrumentID: "BTC-USD-220114-25000-C", TradeMode: TradeModeCross, Currency: currency.USDT, Size: 25, Price: 46000, Side: order.Sell.Lower(), PositionSide: positionSideLong},
			{InstrumentID: mainPair.String(), TradeMode: TradeModeCross, Currency: currency.USDT, Size: 25, Price: 45000, Side: order.Buy.Lower(), PositionSide: positionSideLong, TargetCurrency: "base_ccy", TradeQuoteCurrency: currency.USDT},
		},
	}}
	assert.Equal(t, exp, result, "GetQuotes should decode every field")
}

func TestGetRequestForQuoteTrades(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *RequestForQuoteTradesRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "begin after end", arg: &RequestForQuoteTradesRequest{BeginTime: time.UnixMilli(1650970000000), EndTime: time.UnixMilli(1650960000000)}, err: common.ErrStartAfterEnd},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetRequestForQuoteTrades(t.Context(), tc.arg)
			require.ErrorIs(t, err, tc.err, "GetRequestForQuoteTrades must reject the request")
		})
	}

	unsuccessful := false
	for _, tc := range []struct {
		name string
		arg  *RequestForQuoteTradesRequest
		exp  []RequestForQuoteTradeResponse
	}{
		{
			name: "successful trades",
			arg:  &RequestForQuoteTradesRequest{Limit: 1},
			exp: []RequestForQuoteTradeResponse{{
				CreationTime:            types.Time(time.UnixMilli(1650968164900)),
				RequestForQuoteID:       "123456",
				ClientRequestForQuoteID: "r0001",
				QuoteID:                 "0T5342O",
				BlockTradeID:            "439127542058958848",
				Tag:                     "123456",
				TakerTraderCode:         "SATS",
				MakerTraderCode:         "MIKE",
				IsSuccessful:            true,
				Legs: []BlockTradeLeg{
					{InstrumentID: mainPair.String(), Price: 100, Size: 0.666, Side: order.Sell.Lower(), Fee: -0.0333, FeeCurrency: currency.USDT, TradeID: "439127542058958850", TradeQuoteCurrency: currency.USDT},
				},
				AccountAllocations: []RequestForQuoteAccountAllocation{{
					Account: "acct1", BlockTradeID: "439127542058958848", ErrorCode: "0", Legs: []RequestForQuoteAllocationLeg{
						{InstrumentID: mainPair.String(), Size: 0.666, Fee: -0.0333, FeeCurrency: currency.USDT, TradeID: "439127542058958850"},
					},
				}},
			}},
		},
		{
			name: "unsuccessful trades with every filter",
			arg: &RequestForQuoteTradesRequest{
				RequestForQuoteID:       "1234567",
				ClientRequestForQuoteID: "cr002",
				QuoteID:                 "0T533T0",
				ClientQuoteID:           "q0002",
				BlockTradeID:            "439121886014849024",
				BeginID:                 "439121886014849023",
				EndID:                   "439121886014849025",
				BeginTime:               time.UnixMilli(1650960000000),
				EndTime:                 time.UnixMilli(1650970000000),
				Limit:                   1,
				IsSuccessful:            &unsuccessful,
			},
			exp: []RequestForQuoteTradeResponse{{
				CreationTime:       types.Time(time.UnixMilli(1650966816550)),
				RequestForQuoteID:  "1234567",
				QuoteID:            "0T533T0",
				ClientQuoteID:      "q0002",
				BlockTradeID:       "439121886014849024",
				Tag:                "123456",
				TakerTraderCode:    "SATS",
				MakerTraderCode:    "MIKE",
				ErrorCode:          "70504",
				Legs:               []BlockTradeLeg{{InstrumentID: mainPair.String(), Price: 100, Size: 0.532, Side: order.Sell.Lower(), FeeCurrency: currency.USDT, TradeID: "439121886014849026", TradeQuoteCurrency: currency.USDT}},
				AccountAllocations: []RequestForQuoteAccountAllocation{},
			}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if !mockTests {
				sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
			}
			result, err := e.GetRequestForQuoteTrades(contextGenerate(), tc.arg)
			require.NoError(t, err, "GetRequestForQuoteTrades must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetRequestForQuoteTrades should decode every field")
			} else {
				assert.NotNil(t, result, "GetRequestForQuoteTrades should return a result")
			}
		})
	}
}

func TestGetBlockTickers(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *GetBlockTickersRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "no instrument type", arg: &GetBlockTickersRequest{}, err: errInvalidInstrumentType},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetBlockTickers(t.Context(), tc.arg)
			require.ErrorIs(t, err, tc.err, "GetBlockTickers must reject the request")
		})
	}

	for _, tc := range []struct {
		name           string
		arg            *GetBlockTickersRequest
		instrumentType string
		exp            []BlockTickerResponse
	}{
		{
			name:           "swaps",
			arg:            &GetBlockTickersRequest{InstrumentType: instTypeSwap},
			instrumentType: instTypeSwap,
			exp: []BlockTickerResponse{
				{InstrumentType: instTypeSwap, InstrumentID: "MP-USDT-SWAP", Timestamp: types.Time(time.UnixMilli(1795406235071))},
				{InstrumentType: instTypeSwap, InstrumentID: "EDGE-USDT-SWAP", Timestamp: types.Time(time.UnixMilli(1795406235071))},
				{InstrumentType: instTypeSwap, InstrumentID: "PI-USDT-SWAP", Timestamp: types.Time(time.UnixMilli(1795406235071))},
			},
		},
		{
			name:           "futures of an instrument family",
			arg:            &GetBlockTickersRequest{InstrumentType: "futures", InstrumentFamily: "BTC-USD_UM_XPERP"},
			instrumentType: instTypeFutures,
			exp: []BlockTickerResponse{{
				InstrumentType:                instTypeFutures,
				InstrumentID:                  "BTC-USD_UM_XPERP-310404",
				TradingVolume24HourInCurrency: 18.053,
				TradingVolume24HourInContract: 180530,
				Timestamp:                     types.Time(time.UnixMilli(1795406289586)),
			}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetBlockTickers(contextGenerate(), tc.arg)
			require.NoError(t, err, "GetBlockTickers must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetBlockTickers should decode every field")
				return
			}
			require.NotEmpty(t, result, "GetBlockTickers must return tickers")
			for i := range result {
				assert.Equal(t, tc.instrumentType, result[i].InstrumentType, "GetBlockTickers should return tickers of the requested instrument type")
			}
		})
	}
}

func TestGetBlockTicker(t *testing.T) {
	t.Parallel()
	_, err := e.GetBlockTicker(t.Context(), "")
	require.ErrorIs(t, err, errMissingInstrumentID, "GetBlockTicker must reject an empty instrument ID")

	for _, tc := range []struct {
		name         string
		instrumentID string
		exp          *BlockTickerResponse
	}{
		{
			name:         "spot",
			instrumentID: mainPair.String(),
			exp:          &BlockTickerResponse{InstrumentType: instTypeSpot, InstrumentID: mainPair.String(), Timestamp: types.Time(time.UnixMilli(1795406485345))},
		},
		{
			name:         "future with block volume",
			instrumentID: "BTC-USD_UM_XPERP-310404",
			exp: &BlockTickerResponse{
				InstrumentType:                instTypeFutures,
				InstrumentID:                  "BTC-USD_UM_XPERP-310404",
				TradingVolume24HourInCurrency: 18.053,
				TradingVolume24HourInContract: 180530,
				Timestamp:                     types.Time(time.UnixMilli(1795406269808)),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetBlockTicker(contextGenerate(), tc.instrumentID)
			require.NoError(t, err, "GetBlockTicker must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetBlockTicker should decode every field")
				return
			}
			require.NotNil(t, result, "GetBlockTicker must return a ticker")
			assert.Equal(t, tc.instrumentID, result.InstrumentID, "GetBlockTicker should return the requested instrument")
			assert.NotZero(t, result.Timestamp, "GetBlockTicker should return the ticker time")
		})
	}
}

func TestGetPublicRequestForQuoteTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetPublicRequestForQuoteTrades(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPublicRequestForQuoteTrades must reject a nil request")

	const endID = "4122799370179948544"
	result, err := e.GetPublicRequestForQuoteTrades(contextGenerate(), &GetPublicRequestForQuoteTradesRequest{EndID: endID, Limit: 3})
	require.NoError(t, err, "GetPublicRequestForQuoteTrades must not error")
	if mockTests {
		exp := []PublicTradesResponse{
			{
				BlockTradeID: "4122695986793918464",
				Strategy:     "CALL",
				CreationTime: types.Time(time.UnixMilli(1795368314915)),
				Legs: []PublicBlockTradeLeg{
					{InstrumentID: "ETH-USD-261127-1800-C", Side: order.Sell.Lower(), Size: 200, Price: 0.0115, TradeID: "4122695986768752642"},
				},
			},
			{
				BlockTradeID: "4122660996936871936",
				Strategy:     "PUT_SPREAD",
				Inverse:      true,
				CreationTime: types.Time(time.UnixMilli(1795367272135)),
				Legs: []PublicBlockTradeLeg{
					{InstrumentID: "BTC-USD-261123-59500-P", Side: order.Sell.Lower(), Size: 200, Price: 0.0039, TradeID: "4122660996911706114"},
					{InstrumentID: "BTC-USD-261123-58000-P", Side: order.Buy.Lower(), Size: 200, Price: 0.0015, TradeID: "4122660996911706115"},
				},
			},
			{
				BlockTradeID: "4122644987983147008",
				Strategy:     "CALL_SPREAD",
				Inverse:      true,
				CreationTime: types.Time(time.UnixMilli(1795366795032)),
				Legs: []PublicBlockTradeLeg{
					{InstrumentID: "BTC-USD-261123-62000-C", Side: order.Sell.Lower(), Size: 200, Price: 0.0046, TradeID: "4122644987957981186"},
					{InstrumentID: "BTC-USD-261123-63000-C", Side: order.Buy.Lower(), Size: 200, Price: 0.0012, TradeID: "4122644987957981187"},
				},
			},
		}
		assert.Equal(t, exp, result, "GetPublicRequestForQuoteTrades should decode every field")
		return
	}
	assert.LessOrEqual(t, len(result), 3, "GetPublicRequestForQuoteTrades should return no more trades than the limit")
	for i := range result {
		assert.Less(t, result[i].BlockTradeID, endID, "GetPublicRequestForQuoteTrades should return trades older than the end ID")
		assert.NotEmpty(t, result[i].Legs, "GetPublicRequestForQuoteTrades should return each trade's legs")
	}
}

func TestGetPublicBlockTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetPublicBlockTrades(t.Context(), "")
	require.ErrorIs(t, err, errMissingInstrumentID, "GetPublicBlockTrades must reject an empty instrument ID")

	t.Run("option fields", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the options case crawls live listings instead of a fixed option, which expires")
		}
		result, err := e.GetPublicBlockTrades(contextGenerate(), "BTC-USD-261204-50000-P")
		require.NoError(t, err, "GetPublicBlockTrades must not error")
		exp := []PublicBlockTradeResponse{{
			InstrumentID:   "BTC-USD-261204-50000-P",
			TradeID:        "4123805415744450562",
			Price:          0.0039,
			Size:           494,
			Side:           order.Sell,
			FillVolatility: 0.7629479272460937,
			ForwardPrice:   60775.356288500974,
			IndexPrice:     60716.5,
			MarkPrice:      0.0038424697235885,
			Timestamp:      types.Time(time.UnixMilli(1795401378475)),
		}}
		assert.Equal(t, exp, result, "GetPublicBlockTrades should decode every option field")
	})

	t.Run("options", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the crawl needs live option listings and their block trades")
		}
		testexch.UpdatePairsOnce(t, e)

		pairs, err := e.GetAvailablePairs(asset.Options)
		require.NoError(t, err, "GetAvailablePairs must not error")
		require.NotEmpty(t, pairs, "GetAvailablePairs must return option pairs")

		publicTrades, err := e.GetPublicRequestForQuoteTrades(contextGenerate(), &GetPublicRequestForQuoteTradesRequest{Limit: 100})
		require.NoError(t, err, "GetPublicRequestForQuoteTrades must not error")

		tested := false
	LOOP:
		for _, pt := range publicTrades {
			for _, leg := range pt.Legs {
				p, err := e.MatchSymbolWithAvailablePairs(leg.InstrumentID, asset.Options, true)
				if err != nil {
					continue
				}

				publicBlockTrades, err := e.GetPublicBlockTrades(contextGenerate(), p.String())
				require.NoError(t, err, "GetPublicBlockTrades must not error on Options")
				for _, pbt := range publicBlockTrades {
					assert.Equal(t, p.String(), pbt.InstrumentID, "InstrumentID should have correct value")
					assert.NotEmpty(t, pbt.TradeID, "TradeID should not be empty")
					assert.Positive(t, pbt.Price.Float64(), "Price should have a positive value")
					assert.Positive(t, pbt.Size.Float64(), "Size should have a positive value")
					assert.Contains(t, []order.Side{order.Buy, order.Sell}, pbt.Side, "Side should be a side")
					assert.GreaterOrEqual(t, pbt.FillVolatility.Float64(), float64(0), "FillVolatility should not be negative")
					assert.Positive(t, pbt.ForwardPrice.Float64(), "ForwardPrice should have a positive value")
					assert.Positive(t, pbt.IndexPrice.Float64(), "IndexPrice should have a positive value")
					assert.Positive(t, pbt.MarkPrice.Float64(), "MarkPrice should have a positive value")
					assert.NotEmpty(t, pbt.Timestamp, "Timestamp should not be empty")
					tested = true
					break LOOP
				}
			}
		}
		assert.True(t, tested, "the crawl should find at least one option block trade")
	})

	result, err := e.GetPublicBlockTrades(contextGenerate(), mainPair.String())
	require.NoError(t, err, "GetPublicBlockTrades must not error")
	if mockTests {
		exp := []PublicBlockTradeResponse{{
			InstrumentID: mainPair.String(),
			TradeID:      "4087392299584151554",
			Price:        74446.2,
			Size:         604.46335743,
			Side:         order.Buy,
			Timestamp:    types.Time(time.UnixMilli(1794316183047)),
		}}
		assert.Equal(t, exp, result, "GetPublicBlockTrades should decode every field")
		return
	}
	if len(result) == 0 {
		return // there aren't always block trades returned on mainPair
	}
	blockTrade := result[0]
	assert.Equal(t, mainPair.String(), blockTrade.InstrumentID, "InstrumentID should have correct value")
	assert.NotEmpty(t, blockTrade.TradeID, "TradeID should not be empty")
	assert.Positive(t, blockTrade.Price.Float64(), "Price should have a positive value")
	assert.Positive(t, blockTrade.Size.Float64(), "Size should have a positive value")
	assert.Contains(t, []order.Side{order.Buy, order.Sell}, blockTrade.Side, "Side should be a side")
	assert.WithinRange(t, blockTrade.Timestamp.Time(), time.Now().Add(time.Hour*-24*90), time.Now(), "Timestamp should be within last 90 days")
}

func TestNormaliseLegSide(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		side, exp string
		err       error
	}{
		{side: "buy", exp: "buy"},
		{side: "SELL", exp: "sell"},
		{side: "", exp: "", err: order.ErrSideIsInvalid},
		{side: "short", exp: "short", err: order.ErrSideIsInvalid},
	} {
		t.Run(strconv.Quote(tc.side), func(t *testing.T) {
			t.Parallel()
			side := tc.side
			err := normaliseLegSide(&side)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err, "normaliseLegSide must reject the side")
			} else {
				require.NoError(t, err, "normaliseLegSide must accept the side")
			}
			assert.Equal(t, tc.exp, side, "normaliseLegSide should lower case the side")
		})
	}
}
