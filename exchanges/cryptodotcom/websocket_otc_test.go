package cryptodotcom

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestWebsocketAPIRequestOTCQuote(t *testing.T) {
	t.Parallel()
	requireOTCQuoteRequestValidation(t, "RequestOTCQuote", func(req *OTCQuoteRequest) error {
		_, err := e.WebsocketAPI().RequestOTCQuote(t.Context(), req)
		return err
	})
	skipLiveUnlessCanManipulate(t)

	ex, venue := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().RequestOTCQuote(t.Context(), validOTCQuoteRequest())
	require.NoError(t, err, "RequestOTCQuote must not error")
	if mockTests {
		exp := []OTCQuoteRequestResponse{{QuoteRequestID: "4611686018427626536", ClientQuoteRequestID: "abcd13456"}}
		assert.Equal(t, exp, result, "RequestOTCQuote should decode every field")
		requests := venue.received("private/otc/request-quote")
		require.Len(t, requests, 1, "RequestOTCQuote must send one request")
		assert.JSONEq(t, `{"cl_quote_req_id":"abcd13456","settlement_arrangement":"IMMEDIATE","duration":"10000","quote_ttl":"5000","leg_list":[{"instrument_name":"BTC_USD","side":"BUY","quantity":"1"}]}`, string(requests[0].Params), "RequestOTCQuote should send the quote request")

		venue.respond("private/otc/request-quote", `null`)
		_, err = ex.WebsocketAPI().RequestOTCQuote(t.Context(), validOTCQuoteRequest())
		assert.ErrorIs(t, err, common.ErrNoResponse, "RequestOTCQuote should error for a null result")
	} else {
		require.NotEmpty(t, result, "RequestOTCQuote must return the quote request")
		assert.NotEmpty(t, result[0].QuoteRequestID, "RequestOTCQuote should return the quote request's ID")
	}
}

func TestWebsocketAPIRequestOTCDeal(t *testing.T) {
	t.Parallel()
	requireOTCDealValidation(t, "RequestOTCDeal", func(req *OTCDealRequest) error {
		_, err := e.WebsocketAPI().RequestOTCDeal(t.Context(), req)
		return err
	})
	skipLiveUnlessCanManipulate(t)

	ex, venue := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().RequestOTCDeal(t.Context(), validOTCDealRequest())
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
		requests := venue.received("private/otc/request-deal")
		require.Len(t, requests, 1, "RequestOTCDeal must send one request")
		assert.JSONEq(t, `{"deal_type":"QUOTE_REQUEST","cl_deal_id":"client_id_1","quote_id":"4611686018492073851","quote_req_id":"4611686018427623960","leg_list":[{"instrument_name":"BTC_USD","side":"BUY","price":"95734.2","quantity":"1"}]}`, string(requests[0].Params), "RequestOTCDeal should send the deal request")

		venue.respond("private/otc/request-deal", `null`)
		_, err = ex.WebsocketAPI().RequestOTCDeal(t.Context(), validOTCDealRequest())
		assert.ErrorIs(t, err, common.ErrNoResponse, "RequestOTCDeal should error for a null result")
	} else {
		assert.NotEmpty(t, result.DealID, "RequestOTCDeal should return the deal's ID")
	}
}
