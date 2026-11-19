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
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetSubAccountInformation(t *testing.T) {
	t.Parallel()
	t.Run("sends the set filters", func(t *testing.T) {
		t.Parallel()
		ex := signedRequestExchange(t, http.MethodGet, "/sapi/v3/sub-account/list", url.Values{"email": {"123@test.com"}, "status": {"enabled"}, "page": {"2"}, "limit": {"100"}}, `{}`)
		_, err := ex.GetSubAccountInformation(t.Context(), &SubAccountInformationRequest{Email: "123@test.com", Status: "enabled", Page: 2, Limit: 100})
		require.NoError(t, err, "GetSubAccountInformation must not error")
	})

	skipIfCredentialsUnset(t)
	result, err := e.GetSubAccountInformation(t.Context(), nil)
	require.NoError(t, err, "GetSubAccountInformation must not error")
	require.NotNil(t, result, "GetSubAccountInformation must return a result")
	if mockTests {
		exp := &SubAccountInformationResponse{
			Success:     true,
			SubAccounts: []SubAccount{{Email: "123@test.com", Status: "enabled", Activated: true, Mobile: "91605290", GAuth: true, CreateTime: types.Time(time.UnixMilli(1544433328000))}},
		}
		assert.Equal(t, exp, result, "GetSubAccountInformation should decode every field")
	}
}

func TestGetSubAccountTransferHistory(t *testing.T) {
	t.Parallel()
	start := time.UnixMilli(1794464000000)
	_, err := e.GetSubAccountTransferHistory(t.Context(), &SubAccountTransferHistoryRequest{StartTime: start, EndTime: start.Add(-time.Second)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSubAccountTransferHistory must reject a start after the end")

	t.Run("sends only the set filters", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			req  *SubAccountTransferHistoryRequest
			exp  url.Values
		}{
			{name: "no range", req: &SubAccountTransferHistoryRequest{Email: "aa.email.com"}, exp: url.Values{"email": {"aa.email.com"}}},
			{name: "every filter", req: &SubAccountTransferHistoryRequest{Email: "aa.email.com", StartTime: start, EndTime: start.Add(time.Hour), Page: 2, Limit: 100}, exp: url.Values{"email": {"aa.email.com"}, "startTime": {"1794464000000"}, "endTime": {"1794467600000"}, "page": {"2"}, "limit": {"100"}}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := signedRequestExchange(t, http.MethodGet, "/sapi/v3/sub-account/transfer/history", tc.exp, `{}`)
				_, err := ex.GetSubAccountTransferHistory(t.Context(), tc.req)
				require.NoError(t, err, "GetSubAccountTransferHistory must not error")
			})
		}
	})

	skipIfCredentialsUnset(t)
	result, err := e.GetSubAccountTransferHistory(t.Context(), nil)
	require.NoError(t, err, "GetSubAccountTransferHistory must not error")
	require.NotNil(t, result, "GetSubAccountTransferHistory must return a result")
	if mockTests {
		exp := &SubAccountTransferHistoryResponse{
			Message:   "string",
			Success:   true,
			Transfers: []SubAccountTransfer{{Asset: currency.BNB, From: "aa.email.com", Quantity: 10, Time: types.Time(time.Unix(1637789299, 0)), To: "bb.email.com"}},
		}
		assert.Equal(t, exp, result, "GetSubAccountTransferHistory should decode every field, a time in seconds included")
	}
}

func TestExecuteSubAccountTransfer(t *testing.T) {
	t.Parallel()
	valid := SubAccountTransferRequest{FromEmail: "master@test.com", ToEmail: "sub@test.com", Asset: currency.BTC, Amount: 0.25}
	for _, tc := range []struct {
		name   string
		modify func(*SubAccountTransferRequest)
		err    error
	}{
		{name: "no sender", modify: func(r *SubAccountTransferRequest) { r.FromEmail = "" }, err: errSenderEmailRequired},
		{name: "no recipient", modify: func(r *SubAccountTransferRequest) { r.ToEmail = "" }, err: errRecipientEmailRequired},
		{name: "no asset", modify: func(r *SubAccountTransferRequest) { r.Asset = currency.EMPTYCODE }, err: currency.ErrCurrencyCodeEmpty},
		{name: "no amount", modify: func(r *SubAccountTransferRequest) { r.Amount = 0 }, err: order.ErrAmountIsInvalid},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := valid
			tc.modify(&req)
			_, err := e.ExecuteSubAccountTransfer(t.Context(), &req)
			require.ErrorIsf(t, err, tc.err, "ExecuteSubAccountTransfer must reject a request with %s", tc.name)
		})
	}
	_, err := e.ExecuteSubAccountTransfer(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ExecuteSubAccountTransfer must reject a nil request")

	t.Run("sends a fractional amount in full", func(t *testing.T) {
		t.Parallel()
		ex := signedRequestExchange(t, http.MethodPost, "/sapi/v3/sub-account/transfer", url.Values{"fromEmail": {"master@test.com"}, "toEmail": {"sub@test.com"}, "asset": {"BTC"}, "amount": {"0.00012345"}}, `{}`)
		req := valid
		req.Amount = 0.00012345
		req.Asset = currency.NewCode("btc")
		_, err := ex.ExecuteSubAccountTransfer(t.Context(), &req)
		require.NoError(t, err, "ExecuteSubAccountTransfer must not error")
	})

	if !mockTests && !canManipulateRealOrders {
		t.Skip("canManipulateRealOrders is false; skipping a live transfer")
	}
	skipIfCredentialsUnset(t)
	result, err := e.ExecuteSubAccountTransfer(t.Context(), &valid)
	require.NoError(t, err, "ExecuteSubAccountTransfer must not error")
	if mockTests {
		assert.Equal(t, &SubAccountTransferResponse{Message: "Success", Success: true, TransactionID: "2966662589"}, result, "ExecuteSubAccountTransfer should decode every field")
	}
}

func TestGetSubAccountAssets(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAccountAssets(t.Context(), "")
	require.ErrorIs(t, err, errEmailRequired, "GetSubAccountAssets must reject an empty email")

	skipIfCredentialsUnset(t)
	result, err := e.GetSubAccountAssets(t.Context(), "sub@test.com")
	require.NoError(t, err, "GetSubAccountAssets must not error")
	require.NotNil(t, result, "GetSubAccountAssets must return a result")
	if mockTests {
		assert.Equal(t, &SubAccountAssetsResponse{Balances: []SubAccountBalance{{Asset: currency.BNB, Free: 98, Locked: 0.5}}, Success: true}, result, "GetSubAccountAssets should decode every field")
	}
}

func TestGetMasterAccountTotalUSDValue(t *testing.T) {
	t.Parallel()
	t.Run("sends the set filters", func(t *testing.T) {
		t.Parallel()
		ex := signedRequestExchange(t, http.MethodGet, "/sapi/v1/sub-account/spotSummary", url.Values{"email": {"test001@123.com"}, "page": {"1"}, "size": {"10"}}, `{}`)
		_, err := ex.GetMasterAccountTotalUSDValue(t.Context(), &MasterAccountTotalUSDValueRequest{Email: "test001@123.com", Page: 1, Size: 10})
		require.NoError(t, err, "GetMasterAccountTotalUSDValue must not error")
	})

	skipIfCredentialsUnset(t)
	result, err := e.GetMasterAccountTotalUSDValue(t.Context(), nil)
	require.NoError(t, err, "GetMasterAccountTotalUSDValue must not error")
	require.NotNil(t, result, "GetMasterAccountTotalUSDValue must return a result")
	if mockTests {
		exp := &MasterAccountTotalUSDValueResponse{
			TotalCount:                2,
			MasterAccountTotalAsset:   401.5,
			SpotSubUserAssetBTCVOList: []SubAccountTotalAsset{{Email: "test001@123.com", TotalAsset: 201.5}, {Email: "test002@123.com", TotalAsset: 200}},
		}
		assert.Equal(t, exp, result, "GetMasterAccountTotalUSDValue should decode every field")
	}
}

func TestGetSubAccountStatusList(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAccountStatusList(t.Context(), "")
	require.ErrorIs(t, err, errEmailRequired, "GetSubAccountStatusList must reject an empty email")

	skipIfCredentialsUnset(t)
	result, err := e.GetSubAccountStatusList(t.Context(), "test001@123.com")
	require.NoError(t, err, "GetSubAccountStatusList must not error")
	if mockTests {
		exp := []SubAccountStatusResponse{{
			Email:            "test001@123.com",
			InsertTime:       types.Time(time.UnixMilli(1652944721676)),
			Mobile:           "XXXXXXXXXX",
			IsUserActive:     true,
			IsMarginEnabled:  true,
			IsSubUserEnabled: true,
			IsFutureEnabled:  true,
		}}
		assert.Equal(t, exp, result, "GetSubAccountStatusList should decode every field")
	}
}
