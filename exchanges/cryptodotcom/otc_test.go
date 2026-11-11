package cryptodotcom

import (
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

// otcHistoryStart and otcHistoryEnd bound an OTC query's window in the request shape checks
var (
	otcHistoryStart = time.Unix(0, 1749081600123456789)
	otcHistoryEnd   = time.Unix(0, 1749168000987654321)
)

// validOTCQuoteRequest returns the quote request the recorded responses answer, which every check passes
func validOTCQuoteRequest() *OTCQuoteRequest {
	return &OTCQuoteRequest{
		ClientQuoteRequestID:  "abcd13456",
		SettlementArrangement: OTCSettlementArrangementImmediate,
		Duration:              10 * time.Second,
		QuoteTimeToLive:       5 * time.Second,
		Legs:                  []OTCQuoteRequestLeg{{InstrumentName: "BTC_USD", Side: SideBuy, Quantity: 1}},
	}
}

// requireOTCQuoteRequestValidation checks a quote request method refuses each request its validation rejects
func requireOTCQuoteRequestValidation(t *testing.T, method string, call func(*OTCQuoteRequest) error) {
	t.Helper()
	require.ErrorIsf(t, call(nil), common.ErrNilPointer, "%s must error for a nil request", method)
	// The cases run in order through call, on the caller's exchange and before its later calls of the method, so each
	// also checks that a refusal leaves nothing behind
	for _, tc := range []struct {
		name   string
		modify func(*OTCQuoteRequest)
		err    error
	}{
		{name: "without a client quote request ID", modify: func(r *OTCQuoteRequest) { r.ClientQuoteRequestID = "" }, err: errOTCClientQuoteRequestIDRequired},
		{name: "without legs", modify: func(r *OTCQuoteRequest) { r.Legs = nil }, err: errOTCLegsRequired},
		{name: "for a leg without an instrument", modify: func(r *OTCQuoteRequest) { r.Legs[0].InstrumentName = "" }, err: errInstrumentNameRequired},
		{name: "for a leg without a side", modify: func(r *OTCQuoteRequest) { r.Legs[0].Side = "" }, err: errInvalidSide},
		{name: "for a leg with a negative quantity", modify: func(r *OTCQuoteRequest) { r.Legs[0].Quantity = -1 }, err: errInvalidQuantity},
		{name: "for a leg with a negative notional", modify: func(r *OTCQuoteRequest) { r.Legs[0].Notional = -1 }, err: errInvalidAmount},
		{name: "for a leg with neither a quantity nor a notional", modify: func(r *OTCQuoteRequest) { r.Legs[0].Quantity = 0 }, err: errOTCQuantityOrNotionalRequired},
		{name: "for a leg with both a quantity and a notional", modify: func(r *OTCQuoteRequest) { r.Legs[0].Notional = 100000 }, err: errOTCQuantityOrNotionalRequired},
		{name: "for an unknown settlement arrangement", modify: func(r *OTCQuoteRequest) { r.SettlementArrangement = "T2" }, err: errInvalidOTCSettlementArrangement},
		{name: "for a duration not offered", modify: func(r *OTCQuoteRequest) { r.Duration = 7 * time.Second }, err: errInvalidOTCQuoteRequestDuration},
		{name: "for a negative duration", modify: func(r *OTCQuoteRequest) { r.Duration = -10 * time.Second }, err: errInvalidOTCQuoteRequestDuration},
		{name: "for a time to live not offered", modify: func(r *OTCQuoteRequest) { r.QuoteTimeToLive = 6 * time.Second }, err: errInvalidOTCQuoteTimeToLive},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := validOTCQuoteRequest()
			tc.modify(req)
			require.ErrorIsf(t, call(req), tc.err, "%s must error %s", method, tc.name)
		})
	}
}

// validOTCDealRequest returns the deal request the recorded responses answer, which every check passes
func validOTCDealRequest() *OTCDealRequest {
	return &OTCDealRequest{
		ClientDealID:   "client_id_1",
		QuoteID:        "4611686018492073851",
		QuoteRequestID: "4611686018427623960",
		Legs:           []OTCDealLegRequest{{InstrumentName: "BTC_USD", Side: SideBuy, Price: 95734.2, Quantity: 1}},
	}
}

// requireOTCDealValidation checks a deal request method refuses each request its validation rejects
func requireOTCDealValidation(t *testing.T, method string, call func(*OTCDealRequest) error) {
	t.Helper()
	require.ErrorIsf(t, call(nil), common.ErrNilPointer, "%s must error for a nil request", method)
	// The cases run in order through call, on the caller's exchange and before its later calls of the method, so each
	// also checks that a refusal leaves nothing behind
	for _, tc := range []struct {
		name   string
		modify func(*OTCDealRequest)
		err    error
	}{
		{name: "without a client deal ID", modify: func(r *OTCDealRequest) { r.ClientDealID = "" }, err: errOTCClientDealIDRequired},
		{name: "without a quote ID", modify: func(r *OTCDealRequest) { r.QuoteID = "" }, err: errOTCQuoteIDRequired},
		{name: "without a quote request ID", modify: func(r *OTCDealRequest) { r.QuoteRequestID = "" }, err: errOTCQuoteRequestIDRequired},
		{name: "without legs", modify: func(r *OTCDealRequest) { r.Legs = nil }, err: errOTCLegsRequired},
		{name: "for a leg without an instrument", modify: func(r *OTCDealRequest) { r.Legs[0].InstrumentName = "" }, err: errInstrumentNameRequired},
		{name: "for a leg with an unknown side", modify: func(r *OTCDealRequest) { r.Legs[0].Side = "HOLD" }, err: errInvalidSide},
		{name: "for a leg with a negative quantity", modify: func(r *OTCDealRequest) { r.Legs[0].Quantity = -1 }, err: errInvalidQuantity},
		{name: "for a leg with a negative notional", modify: func(r *OTCDealRequest) { r.Legs[0].Notional = -1 }, err: errInvalidAmount},
		{name: "for a leg with neither a quantity nor a notional", modify: func(r *OTCDealRequest) { r.Legs[0].Quantity = 0 }, err: errOTCQuantityOrNotionalRequired},
		{name: "for a leg with both a quantity and a notional", modify: func(r *OTCDealRequest) { r.Legs[0].Notional = 95734.2 }, err: errOTCQuantityOrNotionalRequired},
		{name: "for a leg without a price", modify: func(r *OTCDealRequest) { r.Legs[0].Price = 0 }, err: errInvalidPrice},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := validOTCDealRequest()
			tc.modify(req)
			require.ErrorIsf(t, call(req), tc.err, "%s must error %s", method, tc.name)
		})
	}
}

// requireOTCWindowValidation checks an OTC query method refuses each window its validation rejects, and a page size
// above the maximum when the method takes one
func requireOTCWindowValidation(t *testing.T, method string, takesLimit bool, call func(start, end time.Time, limit uint64) error) {
	t.Helper()
	require.ErrorIsf(t, call(otcHistoryEnd, otcHistoryStart, 0), common.ErrStartAfterEnd, "%s must error for a start after the end", method)
	require.ErrorIsf(t, call(time.UnixMilli(-1), time.Time{}, 0), errTimeBeforeUnixEpoch, "%s must error for a start before 1970", method)
	require.ErrorIsf(t, call(time.Time{}, time.UnixMilli(-1), 0), errTimeBeforeUnixEpoch, "%s must error for an end before 1970", method)
	if takesLimit {
		require.ErrorIsf(t, call(time.Time{}, time.Time{}, otcHistoryMaximumLimit+1), errInvalidCount, "%s must error for a page size above the maximum", method)
	}
}

func TestRequestOTCQuote(t *testing.T) {
	t.Parallel()
	requireOTCQuoteRequestValidation(t, "RequestOTCQuote", func(req *OTCQuoteRequest) error {
		_, err := e.RequestOTCQuote(t.Context(), req)
		return err
	})
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		skipLiveUnlessCanManipulate(t)
		result, err := e.RequestOTCQuote(t.Context(), validOTCQuoteRequest())
		require.NoError(t, err, "RequestOTCQuote must not error")
		if mockTests {
			exp := []OTCQuoteRequestResponse{{QuoteRequestID: "4611686018427626536", ClientQuoteRequestID: "abcd13456"}}
			assert.Equal(t, exp, result, "RequestOTCQuote should decode every field")
		} else {
			require.NotEmpty(t, result, "RequestOTCQuote must return the quote request")
			assert.NotEmpty(t, result[0].QuoteRequestID, "RequestOTCQuote should return the quote request's ID")
		}
	})

	// The cases run in order, before the calls of RequestOTCQuote below, which parallel cases would follow
	for _, tc := range []struct {
		name string
		req  *OTCQuoteRequest
		exp  string
	}{
		{
			name: "a firm quote of two legs with every param",
			req: &OTCQuoteRequest{
				ClientQuoteRequestID:  "abcd13456",
				FirmQuote:             true,
				SettlementArrangement: OTCSettlementArrangementT1,
				Duration:              5 * time.Minute,
				QuoteTimeToLive:       4 * time.Second,
				Legs: []OTCQuoteRequestLeg{
					{InstrumentName: "BTC_USD", Side: SideBuy, Quantity: 1},
					{InstrumentName: "ETH_USD", Side: SideSell, Notional: 1000, Type: OTCLegTypePrice},
				},
			},
			exp: `{"cl_quote_req_id":"abcd13456","firm_quote":true,"settlement_arrangement":"T1","duration":"300000","quote_ttl":"4000","leg_list":[{"instrument_name":"BTC_USD","side":"BUY","quantity":"1"},{"instrument_name":"ETH_USD","side":"SELL","notional":"1000"}]}`,
		},
		{
			name: "only the required params",
			req:  &OTCQuoteRequest{ClientQuoteRequestID: "q-1", Legs: []OTCQuoteRequestLeg{{InstrumentName: "BTC_USD", Side: SideBuy, Quantity: 0.5}}},
			exp:  `{"cl_quote_req_id":"q-1","leg_list":[{"instrument_name":"BTC_USD","side":"BUY","quantity":"0.5"}]}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ex := newSignedParamsServer(t, "private/otc/request-quote", tc.exp, `{"id":1,"method":"private/otc/request-quote","code":0,"result":{"data":[]}}`)
			_, err := ex.RequestOTCQuote(t.Context(), tc.req)
			require.NoErrorf(t, err, "RequestOTCQuote must not error against the test server for %+v", tc.req)
		})
	}
	_, err := newResponseServerExchange(t, `{"id":1,"method":"private/otc/request-quote","code":0,"result":null}`).RequestOTCQuote(t.Context(), validOTCQuoteRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "RequestOTCQuote should error for a null result")
	_, err = newUnavailableServerExchange(t).RequestOTCQuote(t.Context(), validOTCQuoteRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "RequestOTCQuote should return a request failure")
}

func TestGetOTCInstruments(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		skipLiveWithoutCredentials(t)
		result, err := e.GetOTCInstruments(t.Context())
		require.NoError(t, err, "GetOTCInstruments must not error")
		if mockTests {
			exp := []OTCInstrumentResponse{
				{
					InstrumentName:   "BTC_USD",
					BaseCurrency:     currency.BTC,
					QuoteCurrency:    currency.USD,
					Type:             "SPOT",
					PriceTickSize:    0.01,
					QuoteDecimals:    2,
					QuantityTickSize: 0.00001,
					QuantityDecimals: 5,
				},
			}
			assert.Equal(t, exp, result, "GetOTCInstruments should decode every field")
		} else {
			assert.NotEmpty(t, result, "GetOTCInstruments should return instruments")
		}
	})

	ex := newSignedParamsServer(t, "private/otc/get-otc-instruments", `{}`, `{"id":1,"method":"private/otc/get-otc-instruments","code":0,"result":{"instrument_list":[]}}`)
	_, err := ex.GetOTCInstruments(t.Context())
	require.NoError(t, err, "GetOTCInstruments must not error against the test server")
	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/otc/get-otc-instruments","code":0,"result":null}`).GetOTCInstruments(t.Context())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetOTCInstruments should error for a null result")
	_, err = newUnavailableServerExchange(t).GetOTCInstruments(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetOTCInstruments should return a request failure")
}

// otcQuoteRequestHistoryParamCases are a quote request query with every param set and with none, and the params each
// sends
var otcQuoteRequestHistoryParamCases = []struct {
	name string
	req  *OTCQuoteRequestHistoryRequest
	exp  string
}{
	{
		name: "every param",
		req:  &OTCQuoteRequestHistoryRequest{ClientQuoteRequestID: "abcd13456", QuoteRequestID: "4611686018427626536", StartTime: otcHistoryStart, EndTime: otcHistoryEnd, Limit: 200},
		exp:  `{"cl_quote_req_id":"abcd13456","quote_req_id":"4611686018427626536","start_time":"1749081600123456789","end_time":"1749168000987654321","limit":200}`,
	},
	{name: "no params", req: &OTCQuoteRequestHistoryRequest{}, exp: `{}`},
}

func TestGetOTCOpenQuoteRequests(t *testing.T) {
	t.Parallel()
	_, err := e.GetOTCOpenQuoteRequests(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOTCOpenQuoteRequests must error for a nil request")
	requireOTCWindowValidation(t, "GetOTCOpenQuoteRequests", true, func(start, end time.Time, limit uint64) error {
		_, err := e.GetOTCOpenQuoteRequests(t.Context(), &OTCQuoteRequestHistoryRequest{StartTime: start, EndTime: end, Limit: limit})
		return err
	})
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		skipLiveWithoutCredentials(t)
		result, err := e.GetOTCOpenQuoteRequests(t.Context(), &OTCQuoteRequestHistoryRequest{})
		require.NoError(t, err, "GetOTCOpenQuoteRequests must not error")
		if mockTests {
			exp := []OTCQuoteRequestResponse{
				{
					QuoteRequestID:        "4611686018427626536",
					ClientQuoteRequestID:  "abcd13456",
					SettlementArrangement: OTCSettlementArrangementImmediate,
					Status:                OTCQuoteRequestStatusActive,
					Reason:                "NO_ERROR",
					RequestTime:           types.Time(time.Unix(0, 1749104833703281311)),
					Legs:                  []OTCQuoteRequestLeg{{InstrumentName: "BTC_USD", Side: SideBuy, Quantity: 1, Type: OTCLegTypePrice}},
				},
			}
			assert.Equal(t, exp, result, "GetOTCOpenQuoteRequests should decode every field")
		} else {
			for i := range result {
				assert.NotEmptyf(t, result[i].QuoteRequestID, "GetOTCOpenQuoteRequests should return quote request %d's ID", i)
			}
		}
	})

	// The cases run in order, before the calls of GetOTCOpenQuoteRequests below, which parallel cases would follow
	for _, tc := range otcQuoteRequestHistoryParamCases {
		t.Run(tc.name, func(t *testing.T) {
			ex := newSignedParamsServer(t, "private/otc/get-open-quote-requests", tc.exp, `{"id":1,"method":"private/otc/get-open-quote-requests","code":0,"result":{"count":0,"quote_request_list":[]}}`)
			_, err := ex.GetOTCOpenQuoteRequests(t.Context(), tc.req)
			require.NoErrorf(t, err, "GetOTCOpenQuoteRequests must not error against the test server for %+v", tc.req)
		})
	}
	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/otc/get-open-quote-requests","code":0,"result":null}`).GetOTCOpenQuoteRequests(t.Context(), &OTCQuoteRequestHistoryRequest{})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetOTCOpenQuoteRequests should error for a null result")
	_, err = newUnavailableServerExchange(t).GetOTCOpenQuoteRequests(t.Context(), &OTCQuoteRequestHistoryRequest{})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetOTCOpenQuoteRequests should return a request failure")
}

// otcQuoteHistoryParamCases are a quote query with every param set and with none, and the params each sends
var otcQuoteHistoryParamCases = []struct {
	name string
	req  *OTCQuoteHistoryRequest
	exp  string
}{
	{
		name: "every param",
		req:  &OTCQuoteHistoryRequest{QuoteID: "4611686018492075123", LiquidityProviderQuoteID: "98765", StartTime: otcHistoryStart, EndTime: otcHistoryEnd, Limit: 50},
		exp:  `{"quote_id":"4611686018492075123","lp_quote_id":"98765","start_time":"1749081600123456789","end_time":"1749168000987654321","limit":50}`,
	},
	{name: "no params", req: &OTCQuoteHistoryRequest{}, exp: `{}`},
}

func TestGetOTCOpenQuotes(t *testing.T) {
	t.Parallel()
	_, err := e.GetOTCOpenQuotes(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOTCOpenQuotes must error for a nil request")
	requireOTCWindowValidation(t, "GetOTCOpenQuotes", true, func(start, end time.Time, limit uint64) error {
		_, err := e.GetOTCOpenQuotes(t.Context(), &OTCQuoteHistoryRequest{StartTime: start, EndTime: end, Limit: limit})
		return err
	})
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		skipLiveWithoutCredentials(t)
		result, err := e.GetOTCOpenQuotes(t.Context(), &OTCQuoteHistoryRequest{LiquidityProviderQuoteID: "98765"})
		require.NoError(t, err, "GetOTCOpenQuotes must not error")
		if mockTests {
			exp := []OTCQuoteResponse{
				{
					QuoteID:                  "4611686018492075123",
					LiquidityProviderQuoteID: "98765",
					QuoteRequestID:           "4611686018427626536",
					Status:                   OTCQuoteStatusActive,
					Reason:                   "NO_ERROR",
					ResponseTime:             types.Time(time.Unix(0, 1749104834012345678)),
					ExpiryTime:               types.Time(time.Unix(0, 1749104839012345678)),
					Legs:                     []OTCQuoteLeg{{InstrumentName: "BTC_USD", Quantity: 1, Bid: 95085.99, Ask: 96086.01, Type: OTCLegTypePrice}},
				},
			}
			assert.Equal(t, exp, result, "GetOTCOpenQuotes should decode every field, the expiry time from its alias")
		} else {
			for i := range result {
				assert.NotEmptyf(t, result[i].QuoteID, "GetOTCOpenQuotes should return quote %d's ID", i)
			}
		}
	})

	// The cases run in order, before the calls of GetOTCOpenQuotes below, which parallel cases would follow
	for _, tc := range otcQuoteHistoryParamCases {
		t.Run(tc.name, func(t *testing.T) {
			ex := newSignedParamsServer(t, "private/otc/get-open-quotes", tc.exp, `{"id":1,"method":"private/otc/get-open-quotes","code":0,"result":{"count":0,"quote_list":[]}}`)
			_, err := ex.GetOTCOpenQuotes(t.Context(), tc.req)
			require.NoErrorf(t, err, "GetOTCOpenQuotes must not error against the test server for %+v", tc.req)
		})
	}
	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/otc/get-open-quotes","code":0,"result":null}`).GetOTCOpenQuotes(t.Context(), &OTCQuoteHistoryRequest{})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetOTCOpenQuotes should error for a null result")
	_, err = newUnavailableServerExchange(t).GetOTCOpenQuotes(t.Context(), &OTCQuoteHistoryRequest{})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetOTCOpenQuotes should return a request failure")
}

func TestGetOTCQuoteHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetOTCQuoteHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOTCQuoteHistory must error for a nil request")
	requireOTCWindowValidation(t, "GetOTCQuoteHistory", true, func(start, end time.Time, limit uint64) error {
		_, err := e.GetOTCQuoteHistory(t.Context(), &OTCQuoteHistoryRequest{StartTime: start, EndTime: end, Limit: limit})
		return err
	})
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		skipLiveWithoutCredentials(t)
		result, err := e.GetOTCQuoteHistory(t.Context(), &OTCQuoteHistoryRequest{LiquidityProviderQuoteID: "98765"})
		require.NoError(t, err, "GetOTCQuoteHistory must not error")
		if mockTests {
			exp := []OTCQuoteResponse{
				{
					QuoteID:                  "4611686018492068402",
					LiquidityProviderQuoteID: "98765",
					QuoteRequestID:           "4611686018427625581",
					Status:                   OTCQuoteStatusExpired,
					Reason:                   "NO_ERROR",
					ResponseTime:             types.Time(time.Unix(0, 1747622246812345678)),
					ExpiryTime:               types.Time(time.Unix(0, 1747622251812345678)),
					Legs:                     []OTCQuoteLeg{{InstrumentName: "BTC_USD", Quantity: 1, Bid: 100000, Ask: 100010, Type: OTCLegTypePrice}},
				},
			}
			assert.Equal(t, exp, result, "GetOTCQuoteHistory should decode every field")
		} else {
			for i := range result {
				assert.NotEmptyf(t, result[i].QuoteID, "GetOTCQuoteHistory should return quote %d's ID", i)
			}
		}
	})

	// The cases run in order, before the calls of GetOTCQuoteHistory below, which parallel cases would follow
	for _, tc := range otcQuoteHistoryParamCases {
		t.Run(tc.name, func(t *testing.T) {
			ex := newSignedParamsServer(t, "private/otc/get-quote-history", tc.exp, `{"id":1,"method":"private/otc/get-quote-history","code":0,"result":{"count":0,"quote_list":[]}}`)
			_, err := ex.GetOTCQuoteHistory(t.Context(), tc.req)
			require.NoErrorf(t, err, "GetOTCQuoteHistory must not error against the test server for %+v", tc.req)
		})
	}
	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/otc/get-quote-history","code":0,"result":null}`).GetOTCQuoteHistory(t.Context(), &OTCQuoteHistoryRequest{})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetOTCQuoteHistory should error for a null result")
	_, err = newUnavailableServerExchange(t).GetOTCQuoteHistory(t.Context(), &OTCQuoteHistoryRequest{})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetOTCQuoteHistory should return a request failure")
}

func TestGetOTCQuoteRequestHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetOTCQuoteRequestHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOTCQuoteRequestHistory must error for a nil request")
	requireOTCWindowValidation(t, "GetOTCQuoteRequestHistory", true, func(start, end time.Time, limit uint64) error {
		_, err := e.GetOTCQuoteRequestHistory(t.Context(), &OTCQuoteRequestHistoryRequest{StartTime: start, EndTime: end, Limit: limit})
		return err
	})
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		skipLiveWithoutCredentials(t)
		result, err := e.GetOTCQuoteRequestHistory(t.Context(), &OTCQuoteRequestHistoryRequest{})
		require.NoError(t, err, "GetOTCQuoteRequestHistory must not error")
		if mockTests {
			exp := []OTCQuoteRequestResponse{
				{
					QuoteRequestID:        "4611686018427625581",
					ClientQuoteRequestID:  "abcd13456",
					SettlementArrangement: OTCSettlementArrangementImmediate,
					Status:                OTCQuoteRequestStatusCompleted,
					RequestTime:           types.Time(time.Unix(0, 1747622246551416954)),
					Legs:                  []OTCQuoteRequestLeg{{InstrumentName: "BTC_USD", Side: SideBuy, Quantity: 1, Type: OTCLegTypePrice}},
				},
				{
					QuoteRequestID:        "4611686018427625594",
					ClientQuoteRequestID:  "1739762096888",
					SettlementArrangement: OTCSettlementArrangementImmediate,
					Status:                OTCQuoteRequestStatusRejected,
					Reason:                "INVALID_OTC_INSTRUMENT",
					RequestTime:           types.Time(time.Unix(0, 1747622312004501226)),
					Legs:                  []OTCQuoteRequestLeg{{InstrumentName: "ABC", Side: SideBuy, Quantity: 0.6, Type: OTCLegTypePrice}},
				},
			}
			assert.Equal(t, exp, result, "GetOTCQuoteRequestHistory should decode every field")
		} else {
			for i := range result {
				assert.NotEmptyf(t, result[i].QuoteRequestID, "GetOTCQuoteRequestHistory should return quote request %d's ID", i)
			}
		}
	})

	// The cases run in order, before the calls of GetOTCQuoteRequestHistory below, which parallel cases would follow
	for _, tc := range otcQuoteRequestHistoryParamCases {
		t.Run(tc.name, func(t *testing.T) {
			ex := newSignedParamsServer(t, "private/otc/get-quote-request-history", tc.exp, `{"id":1,"method":"private/otc/get-quote-request-history","code":0,"result":{"count":0,"quote_request_list":[]}}`)
			_, err := ex.GetOTCQuoteRequestHistory(t.Context(), tc.req)
			require.NoErrorf(t, err, "GetOTCQuoteRequestHistory must not error against the test server for %+v", tc.req)
		})
	}
	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/otc/get-quote-request-history","code":0,"result":null}`).GetOTCQuoteRequestHistory(t.Context(), &OTCQuoteRequestHistoryRequest{})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetOTCQuoteRequestHistory should error for a null result")
	_, err = newUnavailableServerExchange(t).GetOTCQuoteRequestHistory(t.Context(), &OTCQuoteRequestHistoryRequest{})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetOTCQuoteRequestHistory should return a request failure")
}

func TestRequestOTCDeal(t *testing.T) {
	t.Parallel()
	requireOTCDealValidation(t, "RequestOTCDeal", func(req *OTCDealRequest) error {
		_, err := e.RequestOTCDeal(t.Context(), req)
		return err
	})
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		skipLiveUnlessCanManipulate(t)
		result, err := e.RequestOTCDeal(t.Context(), validOTCDealRequest())
		require.NoError(t, err, "RequestOTCDeal must not error")
		if mockTests {
			exp := &OTCDealResponse{
				DealID:         "4611686018429016536",
				ClientDealID:   "client_id_1",
				QuoteID:        "4611686018492073851",
				QuoteRequestID: "4611686018427623960",
				DealType:       OTCDealTypeQuoteRequest,
				DealStatus:     OTCDealStatusAccepted,
				Reason:         "NO_ERROR",
				UpdateTime:     types.Time(time.Unix(0, 1739329285207188882)),
				Legs:           []OTCDealLeg{{InstrumentName: "BTC_USD", Side: SideBuy, Price: preciseNumber("95734.2"), Quantity: preciseNumber("1"), Type: OTCLegTypePrice}},
			}
			assert.Equal(t, exp, result, "RequestOTCDeal should decode every field")
		} else {
			assert.NotEmpty(t, result.DealID, "RequestOTCDeal should return the deal's ID")
		}
	})

	everyLeg := validOTCDealRequest()
	everyLeg.Legs = []OTCDealLegRequest{
		{InstrumentName: "BTC_USD", Side: SideBuy, Price: 95734.2, Quantity: 1},
		{InstrumentName: "ETH_USD", Side: SideSell, Price: 3500.5, Notional: 7001},
	}

	// The cases run in order, one using everyLeg, before the calls of RequestOTCDeal below, which parallel cases
	// would follow
	for _, tc := range []struct {
		name string
		req  *OTCDealRequest
		exp  string
	}{
		{
			name: "two legs, one by notional",
			req:  everyLeg,
			exp:  `{"deal_type":"QUOTE_REQUEST","cl_deal_id":"client_id_1","quote_id":"4611686018492073851","quote_req_id":"4611686018427623960","leg_list":[{"instrument_name":"BTC_USD","side":"BUY","price":"95734.2","quantity":"1"},{"instrument_name":"ETH_USD","side":"SELL","price":"3500.5","notional":"7001"}]}`,
		},
		{
			name: "one leg by quantity",
			req:  validOTCDealRequest(),
			exp:  `{"deal_type":"QUOTE_REQUEST","cl_deal_id":"client_id_1","quote_id":"4611686018492073851","quote_req_id":"4611686018427623960","leg_list":[{"instrument_name":"BTC_USD","side":"BUY","price":"95734.2","quantity":"1"}]}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ex := newSignedParamsServer(t, "private/otc/request-deal", tc.exp, `{"id":1,"method":"private/otc/request-deal","code":0,"result":{"deal_id":"1"}}`)
			_, err := ex.RequestOTCDeal(t.Context(), tc.req)
			require.NoErrorf(t, err, "RequestOTCDeal must not error against the test server for %+v", tc.req)
		})
	}
	_, err := newResponseServerExchange(t, `{"id":1,"method":"private/otc/request-deal","code":0,"result":null}`).RequestOTCDeal(t.Context(), validOTCDealRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "RequestOTCDeal should error for a null result")
	_, err = newUnavailableServerExchange(t).RequestOTCDeal(t.Context(), validOTCDealRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "RequestOTCDeal should return a request failure")
}

// otcDealHistoryParamCases are a deal query with every param set and with none, and the params each sends, which always
// name the deal type
var otcDealHistoryParamCases = []struct {
	name string
	req  *OTCDealHistoryRequest
	exp  string
}{
	{
		name: "every param",
		req:  &OTCDealHistoryRequest{DealID: "4611686018429016536", StartTime: otcHistoryStart, EndTime: otcHistoryEnd},
		exp:  `{"deal_type":"QUOTE_REQUEST","deal_id":"4611686018429016536","start_time":"1749081600123456789","end_time":"1749168000987654321"}`,
	},
	{name: "no params", req: &OTCDealHistoryRequest{}, exp: `{"deal_type":"QUOTE_REQUEST"}`},
}

func TestGetOTCOpenDeals(t *testing.T) {
	t.Parallel()
	_, err := e.GetOTCOpenDeals(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOTCOpenDeals must error for a nil request")
	requireOTCWindowValidation(t, "GetOTCOpenDeals", false, func(start, end time.Time, _ uint64) error {
		_, err := e.GetOTCOpenDeals(t.Context(), &OTCDealHistoryRequest{StartTime: start, EndTime: end})
		return err
	})
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		skipLiveWithoutCredentials(t)
		result, err := e.GetOTCOpenDeals(t.Context(), &OTCDealHistoryRequest{})
		require.NoError(t, err, "GetOTCOpenDeals must not error")
		if mockTests {
			exp := []OTCDealResponse{
				{
					DealID:                "4611686018429016536",
					ClientDealID:          "client_id_1",
					QuoteID:               "4611686018492073851",
					QuoteRequestID:        "4611686018427623960",
					DealType:              OTCDealTypeQuoteRequest,
					DealStatus:            OTCDealStatusConfirmed,
					SettlementArrangement: OTCSettlementArrangementImmediate,
					CreateTime:            types.Time(time.Unix(0, 1739329285207188882)),
					UpdateTime:            types.Time(time.Unix(0, 1739329285874211375)),
					Legs: []OTCDealLeg{
						{InstrumentName: "BTC_USD", Side: SideBuy, Price: preciseNumber("95734.2"), Quantity: preciseNumber("1"), ExecutedQuantity: preciseNumber("1"), ExecutedNotional: preciseNumber("95734.2"), Type: OTCLegTypePrice},
					},
				},
			}
			assert.Equal(t, exp, result, "GetOTCOpenDeals should decode every field")
		} else {
			for i := range result {
				assert.NotEmptyf(t, result[i].DealID, "GetOTCOpenDeals should return deal %d's ID", i)
			}
		}
	})

	// The cases run in order, before the calls of GetOTCOpenDeals below, which parallel cases would follow
	for _, tc := range otcDealHistoryParamCases {
		t.Run(tc.name, func(t *testing.T) {
			ex := newSignedParamsServer(t, "private/otc/get-open-deals", tc.exp, `{"id":1,"method":"private/otc/get-open-deals","code":0,"result":{"deal_list":[]}}`)
			_, err := ex.GetOTCOpenDeals(t.Context(), tc.req)
			require.NoErrorf(t, err, "GetOTCOpenDeals must not error against the test server for %+v", tc.req)
		})
	}
	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/otc/get-open-deals","code":0,"result":null}`).GetOTCOpenDeals(t.Context(), &OTCDealHistoryRequest{})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetOTCOpenDeals should error for a null result")
	_, err = newUnavailableServerExchange(t).GetOTCOpenDeals(t.Context(), &OTCDealHistoryRequest{})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetOTCOpenDeals should return a request failure")
}

func TestGetOTCDealHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetOTCDealHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOTCDealHistory must error for a nil request")
	requireOTCWindowValidation(t, "GetOTCDealHistory", false, func(start, end time.Time, _ uint64) error {
		_, err := e.GetOTCDealHistory(t.Context(), &OTCDealHistoryRequest{StartTime: start, EndTime: end})
		return err
	})
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		skipLiveWithoutCredentials(t)
		result, err := e.GetOTCDealHistory(t.Context(), &OTCDealHistoryRequest{StartTime: time.UnixMilli(1696154400000), EndTime: time.UnixMilli(1697364000000)})
		require.NoError(t, err, "GetOTCDealHistory must not error")
		if mockTests {
			exp := []OTCDealResponse{
				{
					DealID:                "4611686018428874931",
					ClientDealID:          "external_deal_id_123",
					QuoteID:               "4611686018491964487",
					QuoteRequestID:        "4611686018427544106",
					DealType:              OTCDealTypeQuoteRequest,
					DealStatus:            OTCDealStatusSettled,
					SettlementArrangement: OTCSettlementArrangementT1,
					CreateTime:            types.Time(time.Unix(0, 1696160390000000000)),
					UpdateTime:            types.Time(time.Unix(0, 1696262400000000000)),
					SettleTime:            types.Time(time.Unix(0, 1696262400000000000)),
					Legs: []OTCDealLeg{
						{InstrumentName: "BTC_USD", Side: SideBuy, Price: preciseNumber("26000.5"), Quantity: preciseNumber("0.1"), ExecutedQuantity: preciseNumber("0.1"), ExecutedNotional: preciseNumber("2600.05"), Type: OTCLegTypePrice},
					},
				},
			}
			assert.Equal(t, exp, result, "GetOTCDealHistory should decode every field")
		} else {
			for i := range result {
				assert.NotEmptyf(t, result[i].DealID, "GetOTCDealHistory should return deal %d's ID", i)
			}
		}
	})

	// The cases run in order, before the calls of GetOTCDealHistory below, which parallel cases would follow
	for _, tc := range otcDealHistoryParamCases {
		t.Run(tc.name, func(t *testing.T) {
			ex := newSignedParamsServer(t, "private/otc/get-deal-history", tc.exp, `{"id":1,"method":"private/otc/get-deal-history","code":0,"result":{"deal_list":[]}}`)
			_, err := ex.GetOTCDealHistory(t.Context(), tc.req)
			require.NoErrorf(t, err, "GetOTCDealHistory must not error against the test server for %+v", tc.req)
		})
	}
	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/otc/get-deal-history","code":0,"result":null}`).GetOTCDealHistory(t.Context(), &OTCDealHistoryRequest{})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetOTCDealHistory should error for a null result")
	_, err = newUnavailableServerExchange(t).GetOTCDealHistory(t.Context(), &OTCDealHistoryRequest{})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetOTCDealHistory should return a request failure")
}

func TestGetOTCSettleLaterLimit(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		skipLiveWithoutCredentials(t)
		result, err := e.GetOTCSettleLaterLimit(t.Context())
		require.NoError(t, err, "GetOTCSettleLaterLimit must not error")
		if mockTests {
			exp := &OTCSettleLaterLimitResponse{AccountID: "52e7c00f-1324-5a6e-bf4d-de445bde21c1", ConfiguredLimit: 1000000, UsedLimit: 20000}
			assert.Equal(t, exp, result, "GetOTCSettleLaterLimit should decode every field")
		} else {
			assert.NotEmpty(t, result.AccountID, "GetOTCSettleLaterLimit should return the account's ID")
		}
	})

	ex := newSignedParamsServer(t, "private/otc/get-settle-later-limit", `{}`, `{"id":1,"method":"private/otc/get-settle-later-limit","code":0,"result":{"account_id":"1"}}`)
	_, err := ex.GetOTCSettleLaterLimit(t.Context())
	require.NoError(t, err, "GetOTCSettleLaterLimit must not error against the test server")
	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/otc/get-settle-later-limit","code":0,"result":null}`).GetOTCSettleLaterLimit(t.Context())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetOTCSettleLaterLimit should error for a null result")
	_, err = newUnavailableServerExchange(t).GetOTCSettleLaterLimit(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetOTCSettleLaterLimit should return a request failure")
}

func TestGetOTCUnsettledAmounts(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		skipLiveWithoutCredentials(t)
		result, err := e.GetOTCUnsettledAmounts(t.Context())
		require.NoError(t, err, "GetOTCUnsettledAmounts must not error")
		if mockTests {
			exp := []OTCUnsettledAmountResponse{
				{SettlementDate: time.Unix(0, 1762848000000000000), Currency: currency.ADA, Amount: preciseNumber("-1000")},
				{SettlementDate: time.Unix(0, 1762848000000000000), Currency: currency.USD, Amount: preciseNumber("583.080000")},
				{SettlementDate: time.Unix(0, 1764057600000000000), Currency: currency.BTC, Amount: preciseNumber("-0.01")},
				{SettlementDate: time.Unix(0, 1764057600000000000), Currency: currency.USD, Amount: preciseNumber("1022.0646")},
			}
			assert.Equal(t, exp, result, "GetOTCUnsettledAmounts should decode every amount, ordered by date and currency")
		} else {
			for i := range result {
				assert.NotEmptyf(t, result[i].Currency, "GetOTCUnsettledAmounts should return amount %d's currency", i)
			}
		}
	})

	ex := newSignedParamsServer(t, "private/otc/get-unsettled-amounts", `{}`, `{"id":1,"method":"private/otc/get-unsettled-amounts","code":0,"result":{"unsettled_ccys":{}}}`)
	_, err := ex.GetOTCUnsettledAmounts(t.Context())
	require.NoError(t, err, "GetOTCUnsettledAmounts must not error against the test server")
	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/otc/get-unsettled-amounts","code":0,"result":{"unsettled_ccys":{"tomorrow":{"BTC":"1"}}}}`).GetOTCUnsettledAmounts(t.Context())
	assert.ErrorIs(t, err, errInvalidOTCSettlementDate, "GetOTCUnsettledAmounts should error for a settlement date that is not a Unix time")
	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/otc/get-unsettled-amounts","code":0,"result":null}`).GetOTCUnsettledAmounts(t.Context())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetOTCUnsettledAmounts should error for a null result")
	_, err = newUnavailableServerExchange(t).GetOTCUnsettledAmounts(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetOTCUnsettledAmounts should return a request failure")
}

func TestOTCUnsettledAmountsResponseAmounts(t *testing.T) {
	t.Parallel()
	// The cases run in order before the later calls of the same function, so each also checks that a call keeps
	// nothing from the last
	for _, tc := range []struct {
		name    string
		payload string
		exp     []OTCUnsettledAmountResponse
	}{
		{name: "no unsettled currencies", payload: `{}`, exp: []OTCUnsettledAmountResponse{}},
		{name: "empty unsettled currencies", payload: `{"unsettled_ccys":{}}`, exp: []OTCUnsettledAmountResponse{}},
		{
			name:    "dates and currencies out of order",
			payload: `{"unsettled_ccys":{"1764057600000000000":{"USD":"1022.0646","BTC":"-0.01"},"1762848000000000000":{"USD":"583.08"}}}`,
			exp: []OTCUnsettledAmountResponse{
				{SettlementDate: time.Unix(0, 1762848000000000000), Currency: currency.USD, Amount: preciseNumber("583.08")},
				{SettlementDate: time.Unix(0, 1764057600000000000), Currency: currency.BTC, Amount: preciseNumber("-0.01")},
				{SettlementDate: time.Unix(0, 1764057600000000000), Currency: currency.USD, Amount: preciseNumber("1022.0646")},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var resp otcUnsettledAmountsResponse
			require.NoErrorf(t, json.Unmarshal([]byte(tc.payload), &resp), "Unmarshal must not error for %s", tc.payload)
			amounts, err := resp.amounts()
			require.NoErrorf(t, err, "amounts must not error for %s", tc.payload)
			assert.Equalf(t, tc.exp, amounts, "amounts should order the amounts of %s by date and currency", tc.payload)
		})
	}
	for _, date := range []string{"tomorrow", "12345", "1764057600x"} {
		t.Run(date, func(t *testing.T) {
			t.Parallel()
			resp := otcUnsettledAmountsResponse{UnsettledCurrencies: map[string]map[currency.Code]types.PreciseNumber{date: {currency.BTC: preciseNumber("1")}}}
			_, err := resp.amounts()
			assert.ErrorIsf(t, err, errInvalidOTCSettlementDate, "amounts should error for the settlement date %q", date)
		})
	}
}

func TestOTCQuoteResponseUnmarshalJSON(t *testing.T) { //nolint:tparallel // the cases run in order before the later calls of the same function
	t.Parallel()
	expiry := types.Time(time.Unix(0, 1749104839012345678))
	// The cases run in order before the later calls of the same function, so each also checks that a call keeps
	// nothing from the last
	for _, tc := range []struct {
		name    string
		payload string
		exp     types.Time
	}{
		{name: "expiry_time_ns", payload: `{"quote_id":"1","expiry_time_ns":"1749104839012345678"}`, exp: expiry},
		{name: "its alias expire_time_ns", payload: `{"quote_id":"1","expire_time_ns":"1749104839012345678"}`, exp: expiry},
		{name: "expiry_time_ns over its alias", payload: `{"quote_id":"1","expiry_time_ns":"1749104839012345678","expire_time_ns":"1749104840000000000"}`, exp: expiry},
		{name: "neither", payload: `{"quote_id":"1"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var quote OTCQuoteResponse
			require.NoErrorf(t, json.Unmarshal([]byte(tc.payload), &quote), "Unmarshal must not error for %s", tc.name)
			assert.Equalf(t, OTCQuoteResponse{QuoteID: "1", ExpiryTime: tc.exp}, quote, "Unmarshal should take the expiry time from %s", tc.name)
		})
	}
	// Serial too, so the cases run before the later calls of the same function
	t.Run("reuse with expiry alias", func(t *testing.T) {
		var quote OTCQuoteResponse
		require.NoError(t, json.Unmarshal([]byte(`{"quote_id":"1","expiry_time_ns":"1749104839012345678"}`), &quote), "Unmarshal must not error for the first quote")
		require.NoError(t, json.Unmarshal([]byte(`{"quote_id":"2","expire_time_ns":"1749104840000000000"}`), &quote), "Unmarshal must not error for the second quote")
		assert.Equal(t, OTCQuoteResponse{QuoteID: "2", ExpiryTime: types.Time(time.Unix(0, 1749104840000000000))}, quote, "Unmarshal should take the expiry time from the quote it decodes")
	})
	t.Run("null leaves a quote unchanged", func(t *testing.T) {
		quote := OTCQuoteResponse{QuoteID: "1", ExpiryTime: expiry}
		require.NoError(t, json.Unmarshal([]byte(`null`), &quote), "Unmarshal must not error for null")
		assert.Equal(t, OTCQuoteResponse{QuoteID: "1", ExpiryTime: expiry}, quote, "Unmarshal should leave a quote unchanged for null")
	})
	var quote OTCQuoteResponse
	assert.Error(t, json.Unmarshal([]byte(`{"quote_id":1}`), &quote), "Unmarshal should error for a malformed quote")
	assert.Error(t, json.Unmarshal([]byte(`{"expire_time_ns":"soon"}`), &quote), "Unmarshal should error for a malformed alias")
}

func TestNewOTCQuoteRequestParams(t *testing.T) {
	t.Parallel()
	quoteRequest := func(modify func(*OTCQuoteRequest)) *OTCQuoteRequest {
		req := validOTCQuoteRequest()
		modify(req)
		return req
	}
	for _, tc := range []struct {
		name string
		req  *OTCQuoteRequest
		exp  *otcQuoteRequestParams
		err  error
		leg  string
	}{
		{
			name: "the recorded quote request",
			req:  validOTCQuoteRequest(),
			exp: &otcQuoteRequestParams{
				ClientQuoteRequestID:  "abcd13456",
				SettlementArrangement: OTCSettlementArrangementImmediate,
				Duration:              10000,
				QuoteTimeToLive:       5000,
				Legs:                  []OTCQuoteRequestLeg{{InstrumentName: "BTC_USD", Side: SideBuy, Quantity: 1}},
			},
		},
		{
			name: "a firm quote of two legs with every param",
			req: &OTCQuoteRequest{
				ClientQuoteRequestID:  "abcd13456",
				FirmQuote:             true,
				SettlementArrangement: OTCSettlementArrangementT1,
				Duration:              5 * time.Minute,
				QuoteTimeToLive:       4 * time.Second,
				Legs: []OTCQuoteRequestLeg{
					{InstrumentName: "BTC_USD", Side: SideBuy, Quantity: 1},
					{InstrumentName: "ETH_USD", Side: SideSell, Notional: 1000, Type: OTCLegTypePrice},
				},
			},
			exp: &otcQuoteRequestParams{
				ClientQuoteRequestID:  "abcd13456",
				FirmQuote:             true,
				SettlementArrangement: OTCSettlementArrangementT1,
				Duration:              300000,
				QuoteTimeToLive:       4000,
				Legs: []OTCQuoteRequestLeg{
					{InstrumentName: "BTC_USD", Side: SideBuy, Quantity: 1},
					{InstrumentName: "ETH_USD", Side: SideSell, Notional: 1000},
				},
			},
		},
		{
			name: "only the required params",
			req:  &OTCQuoteRequest{ClientQuoteRequestID: "q-1", Legs: []OTCQuoteRequestLeg{{InstrumentName: "BTC_USD", Side: SideBuy, Quantity: 0.5}}},
			exp:  &otcQuoteRequestParams{ClientQuoteRequestID: "q-1", Legs: []OTCQuoteRequestLeg{{InstrumentName: "BTC_USD", Side: SideBuy, Quantity: 0.5}}},
		},
		{name: "a nil request", err: common.ErrNilPointer},
		{name: "no client quote request ID", req: quoteRequest(func(r *OTCQuoteRequest) { r.ClientQuoteRequestID = "" }), err: errOTCClientQuoteRequestIDRequired},
		{name: "no legs", req: quoteRequest(func(r *OTCQuoteRequest) { r.Legs = nil }), err: errOTCLegsRequired},
		{name: "an empty list of legs", req: quoteRequest(func(r *OTCQuoteRequest) { r.Legs = []OTCQuoteRequestLeg{} }), err: errOTCLegsRequired},
		{name: "a leg without an instrument", req: quoteRequest(func(r *OTCQuoteRequest) { r.Legs[0].InstrumentName = "" }), err: errInstrumentNameRequired, leg: "leg 0"},
		{
			name: "a second leg with both a quantity and a notional",
			req: quoteRequest(func(r *OTCQuoteRequest) {
				r.Legs = append(r.Legs, OTCQuoteRequestLeg{InstrumentName: "ETH_USD", Side: SideSell, Quantity: 1, Notional: 1000})
			}),
			err: errOTCQuantityOrNotionalRequired,
			leg: "leg 1",
		},
		{name: "an unknown settlement arrangement", req: quoteRequest(func(r *OTCQuoteRequest) { r.SettlementArrangement = "T2" }), err: errInvalidOTCSettlementArrangement},
		{name: "a lower case settlement arrangement", req: quoteRequest(func(r *OTCQuoteRequest) { r.SettlementArrangement = "t1" }), err: errInvalidOTCSettlementArrangement},
		{name: "a duration not offered", req: quoteRequest(func(r *OTCQuoteRequest) { r.Duration = 7 * time.Second }), err: errInvalidOTCQuoteRequestDuration},
		{name: "a duration only a time to live takes", req: quoteRequest(func(r *OTCQuoteRequest) { r.Duration = 4 * time.Second }), err: errInvalidOTCQuoteRequestDuration},
		{name: "a negative duration", req: quoteRequest(func(r *OTCQuoteRequest) { r.Duration = -10 * time.Second }), err: errInvalidOTCQuoteRequestDuration},
		{name: "a time to live not offered", req: quoteRequest(func(r *OTCQuoteRequest) { r.QuoteTimeToLive = 6 * time.Second }), err: errInvalidOTCQuoteTimeToLive},
		{name: "a time to live only a duration takes", req: quoteRequest(func(r *OTCQuoteRequest) { r.QuoteTimeToLive = 10 * time.Second }), err: errInvalidOTCQuoteTimeToLive},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := newOTCQuoteRequestParams(tc.req)
			require.ErrorIsf(t, err, tc.err, "newOTCQuoteRequestParams must return the expected error for %s", tc.name)
			if tc.leg != "" {
				assert.ErrorContainsf(t, err, tc.leg+": ", "newOTCQuoteRequestParams should name %s", tc.leg)
			}
			assert.Equalf(t, tc.exp, got, "newOTCQuoteRequestParams should return the params of %s", tc.name)
		})
	}
	t.Run("leaves the request's legs as they were", func(t *testing.T) {
		t.Parallel()
		req := validOTCQuoteRequest()
		req.Legs[0].Type = OTCLegTypePrice
		_, err := newOTCQuoteRequestParams(req)
		require.NoError(t, err, "newOTCQuoteRequestParams must not error for the recorded quote request")
		assert.Equal(t, []OTCQuoteRequestLeg{{InstrumentName: "BTC_USD", Side: SideBuy, Quantity: 1, Type: OTCLegTypePrice}}, req.Legs, "newOTCQuoteRequestParams should leave the request's legs, types and all, as they were")
	})
}

func TestNewOTCDealParams(t *testing.T) {
	t.Parallel()
	dealRequest := func(modify func(*OTCDealRequest)) *OTCDealRequest {
		req := validOTCDealRequest()
		modify(req)
		return req
	}
	for _, tc := range []struct {
		name string
		req  *OTCDealRequest
		exp  *otcDealParams
		err  error
		leg  string
	}{
		{
			name: "the recorded deal request",
			req:  validOTCDealRequest(),
			exp: &otcDealParams{
				DealType:       OTCDealTypeQuoteRequest,
				ClientDealID:   "client_id_1",
				QuoteID:        "4611686018492073851",
				QuoteRequestID: "4611686018427623960",
				Legs:           []OTCDealLegRequest{{InstrumentName: "BTC_USD", Side: SideBuy, Price: 95734.2, Quantity: 1}},
			},
		},
		{
			name: "two legs, one by notional",
			req: dealRequest(func(r *OTCDealRequest) {
				r.Legs = append(r.Legs, OTCDealLegRequest{InstrumentName: "ETH_USD", Side: SideSell, Price: 3500.5, Notional: 7001})
			}),
			exp: &otcDealParams{
				DealType:       OTCDealTypeQuoteRequest,
				ClientDealID:   "client_id_1",
				QuoteID:        "4611686018492073851",
				QuoteRequestID: "4611686018427623960",
				Legs: []OTCDealLegRequest{
					{InstrumentName: "BTC_USD", Side: SideBuy, Price: 95734.2, Quantity: 1},
					{InstrumentName: "ETH_USD", Side: SideSell, Price: 3500.5, Notional: 7001},
				},
			},
		},
		{name: "a nil request", err: common.ErrNilPointer},
		{name: "no client deal ID", req: dealRequest(func(r *OTCDealRequest) { r.ClientDealID = "" }), err: errOTCClientDealIDRequired},
		{name: "no quote ID", req: dealRequest(func(r *OTCDealRequest) { r.QuoteID = "" }), err: errOTCQuoteIDRequired},
		{name: "no quote request ID", req: dealRequest(func(r *OTCDealRequest) { r.QuoteRequestID = "" }), err: errOTCQuoteRequestIDRequired},
		{name: "no legs", req: dealRequest(func(r *OTCDealRequest) { r.Legs = nil }), err: errOTCLegsRequired},
		{name: "an empty list of legs", req: dealRequest(func(r *OTCDealRequest) { r.Legs = []OTCDealLegRequest{} }), err: errOTCLegsRequired},
		{name: "a leg with an unknown side", req: dealRequest(func(r *OTCDealRequest) { r.Legs[0].Side = "HOLD" }), err: errInvalidSide, leg: "leg 0"},
		{
			name: "a second leg with neither a quantity nor a notional",
			req: dealRequest(func(r *OTCDealRequest) {
				r.Legs = append(r.Legs, OTCDealLegRequest{InstrumentName: "ETH_USD", Side: SideSell, Price: 3500.5})
			}),
			err: errOTCQuantityOrNotionalRequired,
			leg: "leg 1",
		},
		{name: "a leg without a price", req: dealRequest(func(r *OTCDealRequest) { r.Legs[0].Price = 0 }), err: errInvalidPrice, leg: "leg 0"},
		{
			name: "a second leg with a negative price",
			req: dealRequest(func(r *OTCDealRequest) {
				r.Legs = append(r.Legs, OTCDealLegRequest{InstrumentName: "ETH_USD", Side: SideSell, Price: -1, Notional: 7001})
			}),
			err: errInvalidPrice,
			leg: "leg 1",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := newOTCDealParams(tc.req)
			require.ErrorIsf(t, err, tc.err, "newOTCDealParams must return the expected error for %s", tc.name)
			if tc.leg != "" {
				assert.ErrorContainsf(t, err, tc.leg+": ", "newOTCDealParams should name %s", tc.leg)
			}
			assert.Equalf(t, tc.exp, got, "newOTCDealParams should return the params of %s", tc.name)
		})
	}
}

func TestValidateOTCLeg(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name           string
		instrumentName string
		side           string
		quantity       types.Number
		notional       types.Number
		err            error
	}{
		{name: "a buy by quantity", instrumentName: "BTC_USD", side: SideBuy, quantity: 1},
		{name: "a sell by notional", instrumentName: "ETH_USD", side: SideSell, notional: 1000},
		{name: "no instrument", side: SideBuy, quantity: 1, err: errInstrumentNameRequired},
		{name: "no side", instrumentName: "BTC_USD", quantity: 1, err: errInvalidSide},
		{name: "a lower case side", instrumentName: "BTC_USD", side: "buy", quantity: 1, err: errInvalidSide},
		{name: "an unknown side", instrumentName: "BTC_USD", side: "HOLD", quantity: 1, err: errInvalidSide},
		{name: "a negative quantity", instrumentName: "BTC_USD", side: SideBuy, quantity: -1, err: errInvalidQuantity},
		{name: "a negative quantity beside a notional", instrumentName: "BTC_USD", side: SideBuy, quantity: -1, notional: 1000, err: errInvalidQuantity},
		{name: "a negative notional", instrumentName: "BTC_USD", side: SideBuy, notional: -1, err: errInvalidAmount},
		{name: "a negative notional beside a quantity", instrumentName: "BTC_USD", side: SideBuy, quantity: 1, notional: -1, err: errInvalidAmount},
		{name: "neither a quantity nor a notional", instrumentName: "BTC_USD", side: SideBuy, err: errOTCQuantityOrNotionalRequired},
		{name: "both a quantity and a notional", instrumentName: "BTC_USD", side: SideBuy, quantity: 1, notional: 1000, err: errOTCQuantityOrNotionalRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIsf(t, validateOTCLeg(tc.instrumentName, tc.side, tc.quantity, tc.notional), tc.err, "validateOTCLeg should return the expected error for %s", tc.name)
		})
	}
}

func TestOTCDurationParam(t *testing.T) { //nolint:tparallel // the cases share otcQuoteRequestDurations in order
	t.Parallel()
	// The cases share otcQuoteRequestDurations in order, so each also checks that a call keeps nothing from the last
	for _, tc := range []struct {
		duration time.Duration
		exp      uint64
		err      error
	}{
		{},
		{duration: 10 * time.Second, exp: 10000},
		{duration: 10 * time.Minute, exp: 600000},
		{duration: 7 * time.Second, err: errInvalidOTCQuoteRequestDuration},
		{duration: -10 * time.Second, err: errInvalidOTCQuoteRequestDuration},
		{duration: time.Nanosecond, err: errInvalidOTCQuoteRequestDuration},
	} {
		t.Run(tc.duration.String(), func(t *testing.T) {
			got, err := otcDurationParam(tc.duration, otcQuoteRequestDurations, errInvalidOTCQuoteRequestDuration)
			require.ErrorIsf(t, err, tc.err, "otcDurationParam must return the expected error for %s", tc.duration)
			assert.Equalf(t, tc.exp, got, "otcDurationParam should give %s in milliseconds", tc.duration)
		})
	}
}

func TestNewOTCQuoteRequestHistoryParams(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *OTCQuoteRequestHistoryRequest
		exp  *otcQuoteRequestHistoryParams
		err  error
	}{
		{
			name: "every field",
			req:  &OTCQuoteRequestHistoryRequest{ClientQuoteRequestID: "abcd13456", QuoteRequestID: "4611686018427626536", StartTime: otcHistoryStart, EndTime: otcHistoryEnd, Limit: otcHistoryMaximumLimit},
			exp:  &otcQuoteRequestHistoryParams{ClientQuoteRequestID: "abcd13456", QuoteRequestID: "4611686018427626536", StartTime: 1749081600123456789, EndTime: 1749168000987654321, Limit: otcHistoryMaximumLimit},
		},
		{name: "no fields", req: &OTCQuoteRequestHistoryRequest{}, exp: &otcQuoteRequestHistoryParams{}},
		{name: "a nil request", err: common.ErrNilPointer},
		{name: "a limit above the largest", req: &OTCQuoteRequestHistoryRequest{Limit: otcHistoryMaximumLimit + 1}, err: errInvalidCount},
		{name: "a window starting after it ends", req: &OTCQuoteRequestHistoryRequest{StartTime: otcHistoryEnd, EndTime: otcHistoryStart}, err: common.ErrStartAfterEnd},
		{name: "an end before the Unix epoch", req: &OTCQuoteRequestHistoryRequest{EndTime: time.Unix(0, -1)}, err: errTimeBeforeUnixEpoch},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := newOTCQuoteRequestHistoryParams(tc.req)
			require.ErrorIsf(t, err, tc.err, "newOTCQuoteRequestHistoryParams must return the expected error for %s", tc.name)
			assert.Equalf(t, tc.exp, got, "newOTCQuoteRequestHistoryParams should return the params of %s", tc.name)
		})
	}
}

func TestNewOTCQuoteHistoryParams(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *OTCQuoteHistoryRequest
		exp  *otcQuoteHistoryParams
		err  error
	}{
		{
			name: "every field",
			req:  &OTCQuoteHistoryRequest{QuoteID: "4611686018492075123", LiquidityProviderQuoteID: "98765", StartTime: otcHistoryStart, EndTime: otcHistoryEnd, Limit: 50},
			exp:  &otcQuoteHistoryParams{QuoteID: "4611686018492075123", LiquidityProviderQuoteID: "98765", StartTime: 1749081600123456789, EndTime: 1749168000987654321, Limit: 50},
		},
		{name: "no fields", req: &OTCQuoteHistoryRequest{}, exp: &otcQuoteHistoryParams{}},
		{name: "a nil request", err: common.ErrNilPointer},
		{name: "a limit above the largest", req: &OTCQuoteHistoryRequest{Limit: otcHistoryMaximumLimit + 1}, err: errInvalidCount},
		{name: "a window starting after it ends", req: &OTCQuoteHistoryRequest{StartTime: otcHistoryEnd, EndTime: otcHistoryStart}, err: common.ErrStartAfterEnd},
		{name: "a start before the Unix epoch", req: &OTCQuoteHistoryRequest{StartTime: time.Unix(0, -1)}, err: errTimeBeforeUnixEpoch},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := newOTCQuoteHistoryParams(tc.req)
			require.ErrorIsf(t, err, tc.err, "newOTCQuoteHistoryParams must return the expected error for %s", tc.name)
			assert.Equalf(t, tc.exp, got, "newOTCQuoteHistoryParams should return the params of %s", tc.name)
		})
	}
}

func TestNewOTCDealHistoryParams(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *OTCDealHistoryRequest
		exp  *otcDealHistoryParams
		err  error
	}{
		{
			name: "every field",
			req:  &OTCDealHistoryRequest{DealID: "4611686018429016536", StartTime: otcHistoryStart, EndTime: otcHistoryEnd},
			exp:  &otcDealHistoryParams{DealType: OTCDealTypeQuoteRequest, DealID: "4611686018429016536", StartTime: 1749081600123456789, EndTime: 1749168000987654321},
		},
		{name: "no fields", req: &OTCDealHistoryRequest{}, exp: &otcDealHistoryParams{DealType: OTCDealTypeQuoteRequest}},
		{name: "a nil request", err: common.ErrNilPointer},
		{name: "a window starting after it ends", req: &OTCDealHistoryRequest{StartTime: otcHistoryEnd, EndTime: otcHistoryStart}, err: common.ErrStartAfterEnd},
		{name: "a start before the Unix epoch", req: &OTCDealHistoryRequest{StartTime: time.Unix(0, -1)}, err: errTimeBeforeUnixEpoch},
		{name: "an end before the Unix epoch", req: &OTCDealHistoryRequest{EndTime: time.Unix(0, -1)}, err: errTimeBeforeUnixEpoch},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := newOTCDealHistoryParams(tc.req)
			require.ErrorIsf(t, err, tc.err, "newOTCDealHistoryParams must return the expected error for %s", tc.name)
			assert.Equalf(t, tc.exp, got, "newOTCDealHistoryParams should return the params of %s", tc.name)
		})
	}
}

func TestNewOTCWindow(t *testing.T) {
	t.Parallel()
	now := time.Now()
	for _, tc := range []struct {
		name       string
		start, end time.Time
		limit      uint64
		exp        otcWindow
		err        error
	}{
		{name: "a window and a limit", start: otcHistoryStart, end: otcHistoryEnd, limit: 50, exp: otcWindow{start: 1749081600123456789, end: 1749168000987654321}},
		{name: "no window or limit"},
		{name: "a start without an end", start: otcHistoryStart, exp: otcWindow{start: 1749081600123456789}},
		{name: "an end without a start", end: otcHistoryEnd, exp: otcWindow{end: 1749168000987654321}},
		{name: "a start at the Unix epoch", start: time.Unix(0, 0)},
		{name: "a start a nanosecond after the Unix epoch", start: time.Unix(0, 1), exp: otcWindow{start: 1}},
		{name: "the largest limit", limit: otcHistoryMaximumLimit},
		{name: "a limit above the largest", limit: otcHistoryMaximumLimit + 1, err: errInvalidCount},
		{name: "a window starting after it ends", start: otcHistoryEnd, end: otcHistoryStart, err: common.ErrStartAfterEnd},
		{name: "a window starting as it ends", start: otcHistoryStart, end: otcHistoryStart, err: common.ErrStartEqualsEnd},
		{name: "a window starting in the future", start: now.Add(time.Hour), end: now.Add(2 * time.Hour), err: common.ErrStartAfterTimeNow},
		{name: "a start a nanosecond before the Unix epoch", start: time.Unix(0, -1), err: errTimeBeforeUnixEpoch},
		{name: "an end a nanosecond before the Unix epoch", end: time.Unix(0, -1), err: errTimeBeforeUnixEpoch},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := newOTCWindow(tc.start, tc.end, tc.limit)
			require.ErrorIsf(t, err, tc.err, "newOTCWindow must return the expected error for %s", tc.name)
			assert.Equalf(t, tc.exp, got, "newOTCWindow should return the window of %s", tc.name)
		})
	}
}
