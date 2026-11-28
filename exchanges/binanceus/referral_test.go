package binanceus

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetReferralRewardHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetReferralRewardHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetReferralRewardHistory must reject a nil request")
	_, err = e.GetReferralRewardHistory(t.Context(), &ReferralRewardHistoryRequest{UserBusinessType: 1, Page: 1})
	require.ErrorIs(t, err, errRowsRequired, "GetReferralRewardHistory must reject a request without rows")

	t.Run("sends the set parameters", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			req  *ReferralRewardHistoryRequest
			exp  url.Values
		}{
			{name: "referrer", req: &ReferralRewardHistoryRequest{Page: 2, Rows: 200}, exp: url.Values{"userBizType": {"0"}, "page": {"2"}, "rows": {"200"}}},
			{name: "referee without a page", req: &ReferralRewardHistoryRequest{UserBusinessType: 1, Rows: 1}, exp: url.Values{"userBizType": {"1"}, "rows": {"1"}}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := signedRequestExchange(t, http.MethodGet, "/sapi/v1/marketing/referral/reward/history", tc.exp, `{}`)
				_, err := ex.GetReferralRewardHistory(t.Context(), tc.req)
				require.NoError(t, err, "GetReferralRewardHistory must not error")
			})
		}
	})

	skipIfCredentialsUnset(t)
	result, err := e.GetReferralRewardHistory(t.Context(), &ReferralRewardHistoryRequest{Page: 1, Rows: 20})
	require.NoError(t, err, "GetReferralRewardHistory must not error")
	require.NotNil(t, result, "GetReferralRewardHistory must return a page of rewards")
	if mockTests {
		exp := &ReferralRewardHistoryResponse{
			Total: 1,
			Rows:  []ReferralReward{{UserID: 350991652, RewardAmount: 8, ReceiveDateTime: types.Time(time.UnixMilli(1651131084091)), RewardType: "USD"}},
		}
		assert.Equal(t, exp, result, "GetReferralRewardHistory should decode every field")
	}
}
