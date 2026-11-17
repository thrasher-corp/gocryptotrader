package okx

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestSendHTTPRequest(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, payload, errorText string
		slice, text              bool
	}{
		{name: "object", payload: `{"code":"0","data":{"instId":"BTC-USDT-SWAP"}}`},
		{name: "single item", payload: `{"code":"0","data":[{"instId":"BTC-USDT-SWAP"}]}`},
		{name: "full array", payload: `{"code":"0","data":[{"instId":"BTC-USDT-SWAP"},{"instId":"ETH-USDT-SWAP"}]}`, slice: true},
		{name: "escaped string", payload: `{"code":"0","data":"a\"b\\c\n"}`, text: true},
		{name: "single escaped string", payload: `{"code":"0","data":["a\"b\\c\n"]}`, text: true},
		{name: "empty object result", payload: `{"code":"0","data":[]}`, errorText: "got 0 items"},
		{name: "multiple object results", payload: `{"code":"0","data":[{},{}]}`, errorText: "got 2 items"},
		{name: "invalid object", payload: `{"code":"0","data":{"fundingRate":"invalid"}}`, errorText: "cannot unmarshal response data directly"},
		{name: "invalid single item", payload: `{"code":"0","data":[{"fundingRate":"invalid"}]}`, errorText: "as the first array item"},
		{name: "missing data", payload: `{"code":"0"}`, errorText: "cannot unmarshal response data directly"},
		{name: "API error before incompatible data", payload: `{"code":"50001","msg":"service unavailable","data":42}`, errorText: "service unavailable"},
		{name: "unknown API error", payload: `{"code":"999999","data":[]}`, errorText: "999999"},
		{name: "invalid header", payload: `{"code":"invalid","data":[]}`, errorText: "invalid"},
		{name: "malformed JSON", payload: `{"code":"0","data":[}`, errorText: "invalid"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "exchange must initialise")
			server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, err := w.Write([]byte(tc.payload))
				assert.NoError(t, err, "response should write")
			}))
			require.NoError(t, ex.SetHTTPClient(server.Client()), "HTTP client must use the test server")
			require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL+"/"), "endpoint must be local")
			if tc.slice {
				var result []FundingRateResponse
				require.NoError(t, ex.SendHTTPRequest(t.Context(), exchange.RestSpot, getFundingRateHistoryEPL, http.MethodGet, "public/funding-rate", nil, &result, request.UnauthenticatedRequest), "array response must decode")
				require.Len(t, result, 2, "both response items must survive")
				assert.Equal(t, "ETH-USDT-SWAP", result[1].InstrumentID, "second instrument should survive")
				return
			}
			if tc.text {
				var result string
				require.NoError(t, ex.SendHTTPRequest(t.Context(), exchange.RestSpot, getFundingRateHistoryEPL, http.MethodGet, "public/funding-rate", nil, &result, request.UnauthenticatedRequest), "string response must decode")
				assert.Equal(t, "a\"b\\c\n", result, "string escapes should survive envelope extraction")
				return
			}
			var result *FundingRateResponse
			err := ex.SendHTTPRequest(t.Context(), exchange.RestSpot, getFundingRateHistoryEPL, http.MethodGet, "public/funding-rate", nil, &result, request.UnauthenticatedRequest)
			if tc.errorText != "" {
				require.ErrorContains(t, err, tc.errorText, "invalid response must retain useful error context")
				return
			}
			require.NoError(t, err, "typed response must decode")
			require.NotNil(t, result, "response must be populated")
			assert.Equal(t, "BTC-USDT-SWAP", result.InstrumentID, "instrument should decode")
		})
	}
}
