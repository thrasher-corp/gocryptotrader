package binanceus

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// apiPartnerTestID is the API partner ID the API partner tests and their recorded responses use
const apiPartnerTestID = "ABCD1234"

func TestCheckUserEligibility(t *testing.T) {
	t.Parallel()
	_, err := e.CheckUserEligibility(t.Context(), "")
	require.ErrorIs(t, err, errPartnerIDRequired, "CheckUserEligibility must require a partner ID")

	t.Run("sends the request's parameters", func(t *testing.T) {
		t.Parallel()
		ex := signedRequestExchange(t, http.MethodGet, "/sapi/v1/apipartner/checkEligibility", url.Values{"partnerId": {apiPartnerTestID}}, `{}`)
		_, err := ex.CheckUserEligibility(t.Context(), apiPartnerTestID)
		require.NoError(t, err, "CheckUserEligibility must not error")
	})

	t.Run("returns the eligibility", func(t *testing.T) {
		t.Parallel()
		skipIfCredentialsUnset(t)
		result, err := e.CheckUserEligibility(t.Context(), apiPartnerTestID)
		require.NoError(t, err, "CheckUserEligibility must not error")
		require.NotNil(t, result, "CheckUserEligibility must return a result")
		if mockTests {
			assert.Equal(t, &UserEligibilityResponse{IsEligible: true, IsNewUser: true, VIPLevel: 1, ReferrerID: 39472261}, result, "CheckUserEligibility should decode the eligibility")
		}
	})
}

func TestGetRebateHistory(t *testing.T) {
	t.Parallel()
	start, end := time.UnixMilli(1676585000000), time.UnixMilli(1676586000000)
	for _, tc := range []struct {
		name string
		req  *RebateHistoryRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "no partner ID", req: &RebateHistoryRequest{}, err: errPartnerIDRequired},
		{name: "a reversed time range", req: &RebateHistoryRequest{PartnerID: apiPartnerTestID, StartTime: end, EndTime: start}, err: common.ErrStartAfterEnd},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetRebateHistory(t.Context(), tc.req)
			require.ErrorIsf(t, err, tc.err, "GetRebateHistory must refuse a request with %s", tc.name)
		})
	}

	t.Run("sends the request's parameters", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			req  *RebateHistoryRequest
			exp  url.Values
		}{
			{name: "partner ID only", req: &RebateHistoryRequest{PartnerID: apiPartnerTestID}, exp: url.Values{"partnerId": {apiPartnerTestID}}},
			{
				name: "every filter",
				req:  &RebateHistoryRequest{PartnerID: apiPartnerTestID, StartTime: start, EndTime: end, Limit: 1000, Page: 2},
				exp:  url.Values{"partnerId": {apiPartnerTestID}, "startTime": {"1676585000000"}, "endTime": {"1676586000000"}, "limit": {"1000"}, "page": {"2"}},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := signedRequestExchange(t, http.MethodGet, "/sapi/v1/apipartner/rebateHistory", tc.exp, `[]`)
				_, err := ex.GetRebateHistory(t.Context(), tc.req)
				require.NoError(t, err, "GetRebateHistory must not error")
			})
		}
	})

	t.Run("returns the rebates", func(t *testing.T) {
		t.Parallel()
		skipIfCredentialsUnset(t)
		result, err := e.GetRebateHistory(t.Context(), &RebateHistoryRequest{PartnerID: apiPartnerTestID, StartTime: start, EndTime: end})
		require.NoError(t, err, "GetRebateHistory must not error")
		if mockTests {
			exp := []RebateRecordResponse{
				{Quantity: 0.02063898, Asset: currency.BTC, Date: types.Time(time.UnixMilli(1676585861493))},
				{Quantity: 1.2063898, Asset: currency.USDT, Date: types.Time(time.UnixMilli(1676585861493))},
			}
			assert.Equal(t, exp, result, "GetRebateHistory should decode the rebates")
		}
	})
}
