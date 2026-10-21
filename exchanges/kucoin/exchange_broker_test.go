package kucoin

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetExchangeBrokerRebate(t *testing.T) {
	t.Parallel()
	_, err := e.GetExchangeBrokerRebate(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetExchangeBrokerRebate must reject a nil request")
	_, err = e.GetExchangeBrokerRebate(t.Context(), &GetExchangeBrokerRebateRequest{Begin: time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC), End: time.Date(2024, 10, 10, 0, 0, 0, 0, time.UTC)})
	require.ErrorIs(t, err, errTradeTypeMissing, "GetExchangeBrokerRebate must reject an empty trade type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetExchangeBrokerRebate(t.Context(), &GetExchangeBrokerRebateRequest{
		Begin:     time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC),
		End:       time.Date(2024, 10, 10, 0, 0, 0, 0, time.UTC),
		TradeType: "1",
	})
	require.NoError(t, err, "GetExchangeBrokerRebate must not error")
	require.NotNil(t, result, "GetExchangeBrokerRebate must return a result")
	if mockTests {
		exp := &BrokerRebateDownloadResponse{
			URL: "https://kc-v2-promotion.s3.ap-northeast-1.amazonaws.com/broker/671aec522593f600019766d0_file.csv?X-Amz-Security-Token=IQo*********2cd90f14efb",
		}
		assert.Equal(t, exp, result, "GetExchangeBrokerRebate should decode every field")
		return
	}
	assert.NotEmpty(t, result.URL, "GetExchangeBrokerRebate should return a download link")
}

func TestGetExchangeBrokerRebateV3(t *testing.T) {
	t.Parallel()
	_, err := e.GetExchangeBrokerRebateV3(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetExchangeBrokerRebateV3 must reject a nil request")
	_, err = e.GetExchangeBrokerRebateV3(t.Context(), &GetExchangeBrokerRebateV3Request{Begin: time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC), End: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), TradeType: "FUTURE"})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetExchangeBrokerRebateV3 must reject a begin after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetExchangeBrokerRebateV3(t.Context(), &GetExchangeBrokerRebateV3Request{
		Begin:     time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		End:       time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC),
		TradeType: "FUTURE",
	})
	require.NoError(t, err, "GetExchangeBrokerRebateV3 must not error")
	require.NotNil(t, result, "GetExchangeBrokerRebateV3 must return a result")
	if mockTests {
		exp := &BrokerRebateDownloadResponse{URL: "https://example.com/download/rebase_order_20240115.csv"}
		assert.Equal(t, exp, result, "GetExchangeBrokerRebateV3 should decode every field")
		return
	}
	assert.NotEmpty(t, result.URL, "GetExchangeBrokerRebateV3 should return a download link")
}

func TestSubmitExchangeBrokerKYC(t *testing.T) {
	t.Parallel()
	err := e.SubmitExchangeBrokerKYC(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SubmitExchangeBrokerKYC must reject a nil request")
	idCard := SubmitExchangeBrokerKYCRequest{
		ClientUID:      "100000001",
		FirstName:      "Kaylah",
		LastName:       "Padberg",
		IssueCountry:   "JP",
		BirthDate:      time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
		IdentityType:   "idcard",
		IdentityNumber: "55",
		ExpireDate:     time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC),
		FrontPhoto:     "data:image/jpeg;base64,/9j/4AAQSkZJRgABAw==",
		BackendPhoto:   "data:image/jpeg;base64,/9j/4AAQSkZJRgABAg==",
		FacePhoto:      "data:image/jpeg;base64,/9j/4AAQSkZJRgABAQ==",
	}
	for _, tc := range []struct {
		field string
		clear func(*SubmitExchangeBrokerKYCRequest)
		err   error
	}{
		{field: "ClientUID", clear: func(r *SubmitExchangeBrokerKYCRequest) { r.ClientUID = "" }, err: errBrokerClientUIDRequired},
		{field: "FirstName", clear: func(r *SubmitExchangeBrokerKYCRequest) { r.FirstName = "" }, err: errBrokerFirstNameRequired},
		{field: "LastName", clear: func(r *SubmitExchangeBrokerKYCRequest) { r.LastName = "" }, err: errBrokerLastNameRequired},
		{field: "IssueCountry", clear: func(r *SubmitExchangeBrokerKYCRequest) { r.IssueCountry = "" }, err: errBrokerIssueCountryRequired},
		{field: "BirthDate", clear: func(r *SubmitExchangeBrokerKYCRequest) { r.BirthDate = time.Time{} }, err: common.ErrDateUnset},
		{field: "IdentityType", clear: func(r *SubmitExchangeBrokerKYCRequest) { r.IdentityType = "" }, err: errBrokerIdentityTypeRequired},
		{field: "IdentityNumber", clear: func(r *SubmitExchangeBrokerKYCRequest) { r.IdentityNumber = "" }, err: errBrokerIdentityNumberRequired},
		{field: "ExpireDate", clear: func(r *SubmitExchangeBrokerKYCRequest) { r.ExpireDate = time.Time{} }, err: common.ErrDateUnset},
		{field: "FacePhoto", clear: func(r *SubmitExchangeBrokerKYCRequest) { r.FacePhoto = "" }, err: errBrokerFacePhotoRequired},
	} {
		req := idCard
		tc.clear(&req)
		err = e.SubmitExchangeBrokerKYC(t.Context(), &req)
		require.ErrorIsf(t, err, tc.err, "SubmitExchangeBrokerKYC must reject a request without %s", tc.field)
	}

	sent := sentRequest(t, func(ex *Exchange) error {
		return ex.SubmitExchangeBrokerKYC(t.Context(), &idCard)
	})
	assert.JSONEq(t, `{
		"clientUid":"100000001","firstName":"Kaylah","lastName":"Padberg","issueCountry":"JP","birthDate":"2000-01-01",
		"identityType":"idcard","identityNumber":"55","expireDate":"2030-01-01",
		"frontPhoto":"data:image/jpeg;base64,/9j/4AAQSkZJRgABAw==","backendPhoto":"data:image/jpeg;base64,/9j/4AAQSkZJRgABAg==",
		"facePhoto":"data:image/jpeg;base64,/9j/4AAQSkZJRgABAQ=="
	}`, sent.body, "SubmitExchangeBrokerKYC should send the documented strings")

	if !mockTests {
		t.Skip("the client UIDs are examples, and submitting KYC for one of the broker's real sub-accounts is not safe in a test")
	}
	err = e.SubmitExchangeBrokerKYC(t.Context(), &idCard)
	require.NoError(t, err, "SubmitExchangeBrokerKYC must not error for an ID card")
	// A BVN identity may omit both document photos
	err = e.SubmitExchangeBrokerKYC(t.Context(), &SubmitExchangeBrokerKYCRequest{
		ClientUID:      "100000002",
		FirstName:      "Ada",
		LastName:       "Obi",
		IssueCountry:   "NG",
		BirthDate:      time.Date(1995, 6, 15, 0, 0, 0, 0, time.UTC),
		IdentityType:   "bvn",
		IdentityNumber: "22212345678",
		ExpireDate:     time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC),
		FacePhoto:      "data:image/jpeg;base64,/9j/4AAQSkZJRgABBA==",
	})
	require.NoError(t, err, "SubmitExchangeBrokerKYC must not error for a BVN")
}

func TestGetExchangeBrokerKYCStatus(t *testing.T) {
	t.Parallel()
	_, err := e.GetExchangeBrokerKYCStatus(t.Context(), nil)
	require.ErrorIs(t, err, errBrokerClientUIDRequired, "GetExchangeBrokerKYCStatus must reject no client UIDs")
	_, err = e.GetExchangeBrokerKYCStatus(t.Context(), []string{"100000001", ""})
	require.ErrorIs(t, err, errBrokerClientUIDRequired, "GetExchangeBrokerKYCStatus must reject an empty client UID")

	clientUIDs := []string{"100000001", "100000002"}
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		statuses, err := e.GetExchangeBrokerKYCStatusList(t.Context(), 1, 1)
		require.NoError(t, err, "GetExchangeBrokerKYCStatusList must not error")
		require.NotNil(t, statuses, "GetExchangeBrokerKYCStatusList must return a page")
		if len(statuses.Items) == 0 {
			t.Skip("the broker has no KYC submissions to look up")
		}
		clientUIDs = []string{strconv.FormatUint(statuses.Items[0].ClientUID, 10)}
	}
	result, err := e.GetExchangeBrokerKYCStatus(t.Context(), clientUIDs)
	require.NoError(t, err, "GetExchangeBrokerKYCStatus must not error")
	if mockTests {
		exp := []ExchangeBrokerKYCStatusResponse{
			{
				ClientUID: 100000001,
				Status:    "PROCESS",
			},
			{
				ClientUID:    100000002,
				Status:       "REJECT",
				RejectReason: "frontPhoto:Picture is not clear/covered/incomplete",
			},
		}
		assert.Equal(t, exp, result, "GetExchangeBrokerKYCStatus should decode every field")
		return
	}
	assert.Len(t, result, 1, "GetExchangeBrokerKYCStatus should return the status of the client UID asked for")
}

func TestGetExchangeBrokerKYCStatusList(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetExchangeBrokerKYCStatusList(t.Context(), 1, 100)
	require.NoError(t, err, "GetExchangeBrokerKYCStatusList must not error")
	if mockTests {
		exp := &ExchangeBrokerKYCStatusListResponse{
			CurrentPage: 1,
			PageSize:    100,
			TotalNumber: 9,
			TotalPage:   1,
			Items: []ExchangeBrokerKYCStatusResponse{
				{
					ClientUID: 100000001,
					Status:    "PROCESS",
				},
				{
					ClientUID:    100000002,
					Status:       "REJECT",
					RejectReason: "frontPhoto:Picture is not clear/covered/incomplete",
				},
			},
		}
		assert.Equal(t, exp, result, "GetExchangeBrokerKYCStatusList should decode every field")
		return
	}
	assert.NotNil(t, result, "GetExchangeBrokerKYCStatusList should return a result")
}

func TestGetExchangeBrokerInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetExchangeBrokerInfo(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetExchangeBrokerInfo must reject a nil request")
	_, err = e.GetExchangeBrokerInfo(t.Context(), &GetExchangeBrokerInfoRequest{End: time.Date(2024, 10, 10, 0, 0, 0, 0, time.UTC), TradeType: "1"})
	require.ErrorIs(t, err, common.ErrDateUnset, "GetExchangeBrokerInfo must reject an unset begin")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetExchangeBrokerInfo(t.Context(), &GetExchangeBrokerInfoRequest{
		Begin:     time.Date(2024, 5, 10, 0, 0, 0, 0, time.UTC),
		End:       time.Date(2024, 10, 10, 0, 0, 0, 0, time.UTC),
		TradeType: "1",
	})
	require.NoError(t, err, "GetExchangeBrokerInfo must not error")
	if mockTests {
		exp := &ExchangeBrokerInfoResponse{
			AccountSize:    2,
			MaxAccountSize: 100,
			Level:          1,
		}
		assert.Equal(t, exp, result, "GetExchangeBrokerInfo should decode every field")
		return
	}
	assert.NotNil(t, result, "GetExchangeBrokerInfo should return a result")
}

func TestAddExchangeBrokerSubAccount(t *testing.T) {
	t.Parallel()
	_, err := e.AddExchangeBrokerSubAccount(t.Context(), "")
	require.ErrorIs(t, err, errInvalidSubAccountName, "AddExchangeBrokerSubAccount must reject an empty account name")

	sent := sentRequest(t, func(ex *Exchange) error {
		_, err := ex.AddExchangeBrokerSubAccount(t.Context(), "Account15")
		return err
	})
	assert.JSONEq(t, `{"accountName":"Account15"}`, sent.body, "AddExchangeBrokerSubAccount should send the account name")

	if !mockTests {
		t.Skip("sub-account names are unique across KuCoin, so the docs' example name is taken, and a test must not create sub-accounts it cannot remove")
	}
	result, err := e.AddExchangeBrokerSubAccount(t.Context(), "Account15")
	require.NoError(t, err, "AddExchangeBrokerSubAccount must not error")
	exp := &AddExchangeBrokerSubAccountResponse{
		AccountName: "Account15",
		UID:         "100000001",
		CreatedAt:   types.Time(time.UnixMilli(1729819381908)),
		Level:       1,
	}
	assert.Equal(t, exp, result, "AddExchangeBrokerSubAccount should decode every field")
}

func TestGetExchangeBrokerSubAccounts(t *testing.T) {
	t.Parallel()
	_, err := e.GetExchangeBrokerSubAccounts(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetExchangeBrokerSubAccounts must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetExchangeBrokerSubAccounts(t.Context(), &GetExchangeBrokerSubAccountsRequest{UID: "100000001", CurrentPage: 1, PageSize: 20})
	require.NoError(t, err, "GetExchangeBrokerSubAccounts must not error")
	if mockTests {
		exp := &ExchangeBrokerSubAccountsResponse{
			CurrentPage: 1,
			PageSize:    20,
			TotalNumber: 1,
			TotalPage:   1,
			Items: []ExchangeBrokerSubAccountResponse{
				{
					AccountName: "Account15",
					UID:         "100000001",
					CreatedAt:   types.Time(time.UnixMilli(1729819382000)),
					Level:       1,
				},
			},
		}
		assert.Equal(t, exp, result, "GetExchangeBrokerSubAccounts should decode every field")
		return
	}
	assert.NotNil(t, result, "GetExchangeBrokerSubAccounts should return a result")
}

// liveExchangeBrokerSubAccount returns the first of the broker's sub-accounts, so live tests look up one the broker
// owns, and skips the test when there is none
func liveExchangeBrokerSubAccount(t *testing.T) ExchangeBrokerSubAccountResponse {
	t.Helper()
	page, err := e.GetExchangeBrokerSubAccounts(t.Context(), &GetExchangeBrokerSubAccountsRequest{CurrentPage: 1})
	require.NoError(t, err, "GetExchangeBrokerSubAccounts must not error")
	require.NotNil(t, page, "GetExchangeBrokerSubAccounts must return a page")
	if len(page.Items) == 0 {
		t.Skip("the broker has no sub-accounts to look up")
	}
	return page.Items[0]
}

func TestAddExchangeBrokerSubAccountAPI(t *testing.T) {
	t.Parallel()
	_, err := e.AddExchangeBrokerSubAccountAPI(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "AddExchangeBrokerSubAccountAPI must reject a nil request")
	_, err = e.AddExchangeBrokerSubAccountAPI(t.Context(), &AddExchangeBrokerSubAccountAPIRequest{Passphrase: "mock-sub-passphrase", Label: "This is remarks3"})
	require.ErrorIs(t, err, errSubUserIDRequired, "AddExchangeBrokerSubAccountAPI must reject an empty UID")
	_, err = e.AddExchangeBrokerSubAccountAPI(t.Context(), &AddExchangeBrokerSubAccountAPIRequest{UID: "100000001", Label: "This is remarks3"})
	require.ErrorIs(t, err, errInvalidPassPhraseInstance, "AddExchangeBrokerSubAccountAPI must reject an empty passphrase")
	_, err = e.AddExchangeBrokerSubAccountAPI(t.Context(), &AddExchangeBrokerSubAccountAPIRequest{UID: "100000001", Passphrase: "mock-sub-passphrase"})
	require.ErrorIs(t, err, errBrokerAPIKeyLabelRequired, "AddExchangeBrokerSubAccountAPI must reject an empty label")

	req := &AddExchangeBrokerSubAccountAPIRequest{
		UID:         "100000001",
		Passphrase:  "mock-sub-passphrase",
		IPWhitelist: []string{"192.0.2.1", "198.51.100.7"},
		Permissions: []string{"General", "Spot", "Futures", "Margin", "InnerTransfer", "Transfer", "Earn"},
		Label:       "This is remarks3",
	}
	sent := sentRequest(t, func(ex *Exchange) error {
		_, err := ex.AddExchangeBrokerSubAccountAPI(t.Context(), req)
		return err
	})
	assert.JSONEq(t, `{
		"uid":"100000001","passphrase":"mock-sub-passphrase","ipWhitelist":["192.0.2.1","198.51.100.7"],
		"permissions":["General","Spot","Futures","Margin","InnerTransfer","Transfer","Earn"],"label":"This is remarks3"
	}`, sent.body, "AddExchangeBrokerSubAccountAPI should send the UID as a string and the lists as string arrays")

	if !mockTests {
		t.Skip("the sub-account UID is an example, and a test must not leave an API key on one of the broker's real sub-accounts")
	}
	result, err := e.AddExchangeBrokerSubAccountAPI(t.Context(), req)
	require.NoError(t, err, "AddExchangeBrokerSubAccountAPI must not error")
	exp := &AddExchangeBrokerSubAccountAPIResponse{
		UID:         "100000001",
		Label:       "This is remarks3",
		APIKey:      "mock-sub-api-key",
		SecretKey:   "mock-sub-api-secret",
		APIVersion:  3,
		Permissions: []string{"General", "Futures", "Spot", "Earn", "InnerTransfer", "Transfer", "Margin"},
		IPWhitelist: []string{"192.0.2.1", "198.51.100.7"},
		CreatedAt:   types.Time(time.UnixMilli(1771124671000)),
	}
	assert.Equal(t, exp, result, "AddExchangeBrokerSubAccountAPI should decode every field")
}

func TestGetExchangeBrokerSubAccountAPIs(t *testing.T) {
	t.Parallel()
	_, err := e.GetExchangeBrokerSubAccountAPIs(t.Context(), "", "")
	require.ErrorIs(t, err, errSubUserIDRequired, "GetExchangeBrokerSubAccountAPIs must reject an empty UID")

	uid, apiKey := "100000001", "mock-sub-api-key"
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		uid, apiKey = liveExchangeBrokerSubAccount(t).UID, ""
	}
	result, err := e.GetExchangeBrokerSubAccountAPIs(t.Context(), uid, apiKey)
	require.NoError(t, err, "GetExchangeBrokerSubAccountAPIs must not error")
	if mockTests {
		exp := []ExchangeBrokerSubAccountAPIResponse{
			{
				UID:         "100000001",
				Label:       "This is remarks3",
				APIKey:      "mock-sub-api-key",
				APIVersion:  3,
				Permissions: []string{"General", "Spot"},
				IPWhitelist: []string{"192.**.1", "198.**.7"},
				CreatedAt:   types.Time(time.UnixMilli(1771124671000)),
			},
		}
		assert.Equal(t, exp, result, "GetExchangeBrokerSubAccountAPIs should decode every field")
		return
	}
	assert.NotNil(t, result, "GetExchangeBrokerSubAccountAPIs should return a result")
}

func TestModifyExchangeBrokerSubAccountAPI(t *testing.T) {
	t.Parallel()
	_, err := e.ModifyExchangeBrokerSubAccountAPI(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ModifyExchangeBrokerSubAccountAPI must reject a nil request")
	_, err = e.ModifyExchangeBrokerSubAccountAPI(t.Context(), &ModifyExchangeBrokerSubAccountAPIRequest{APIKey: "mock-sub-api-key", Label: "This is remarks"})
	require.ErrorIs(t, err, errSubUserIDRequired, "ModifyExchangeBrokerSubAccountAPI must reject an empty UID")
	_, err = e.ModifyExchangeBrokerSubAccountAPI(t.Context(), &ModifyExchangeBrokerSubAccountAPIRequest{UID: "100000001", Label: "This is remarks"})
	require.ErrorIs(t, err, errAPIKeyRequired, "ModifyExchangeBrokerSubAccountAPI must reject an empty API key")
	_, err = e.ModifyExchangeBrokerSubAccountAPI(t.Context(), &ModifyExchangeBrokerSubAccountAPIRequest{UID: "100000001", APIKey: "mock-sub-api-key"})
	require.ErrorIs(t, err, errBrokerAPIKeyLabelRequired, "ModifyExchangeBrokerSubAccountAPI must reject an empty label")

	req := &ModifyExchangeBrokerSubAccountAPIRequest{
		UID:         "100000001",
		APIKey:      "mock-sub-api-key",
		IPWhitelist: []string{"192.0.2.1", "198.51.100.7"},
		Permissions: []string{"General", "Spot"},
		Label:       "This is remarks",
	}
	sent := sentRequest(t, func(ex *Exchange) error {
		_, err := ex.ModifyExchangeBrokerSubAccountAPI(t.Context(), req)
		return err
	})
	assert.JSONEq(t, `{
		"uid":"100000001","apiKey":"mock-sub-api-key","ipWhitelist":["192.0.2.1","198.51.100.7"],"permissions":["General","Spot"],
		"label":"This is remarks"
	}`, sent.body, "ModifyExchangeBrokerSubAccountAPI should send the UID as a string and the lists as string arrays")

	if !mockTests {
		t.Skip("the sub-account UID and API key are examples, and changing one of the broker's real API keys is not safe in a test")
	}
	result, err := e.ModifyExchangeBrokerSubAccountAPI(t.Context(), req)
	require.NoError(t, err, "ModifyExchangeBrokerSubAccountAPI must not error")
	exp := &ModifyExchangeBrokerSubAccountAPIResponse{
		UID:         "100000001",
		Label:       "This is remarks",
		APIKey:      "mock-sub-api-key",
		APIVersion:  3,
		Permissions: []string{"General", "Spot"},
		IPWhitelist: []string{"192.**.1", "198.**.7"},
		CreatedAt:   types.Time(time.UnixMilli(1771124671000)),
	}
	assert.Equal(t, exp, result, "ModifyExchangeBrokerSubAccountAPI should decode every field")
}

func TestDeleteExchangeBrokerSubAccountAPI(t *testing.T) {
	t.Parallel()
	_, err := e.DeleteExchangeBrokerSubAccountAPI(t.Context(), "", "mock-sub-api-key")
	require.ErrorIs(t, err, errSubUserIDRequired, "DeleteExchangeBrokerSubAccountAPI must reject an empty UID")
	_, err = e.DeleteExchangeBrokerSubAccountAPI(t.Context(), "100000001", "")
	require.ErrorIs(t, err, errAPIKeyRequired, "DeleteExchangeBrokerSubAccountAPI must reject an empty API key")

	if !mockTests {
		t.Skip("the sub-account UID and API key are examples, and deleting one of the broker's real API keys is not safe in a test")
	}
	result, err := e.DeleteExchangeBrokerSubAccountAPI(t.Context(), "100000001", "mock-sub-api-key")
	require.NoError(t, err, "DeleteExchangeBrokerSubAccountAPI must not error")
	assert.True(t, result, "DeleteExchangeBrokerSubAccountAPI should report the API key deleted")
}

func TestExchangeBrokerTransfer(t *testing.T) {
	t.Parallel()
	_, err := e.ExchangeBrokerTransfer(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ExchangeBrokerTransfer must reject a nil request")
	valid := ExchangeBrokerTransferRequest{
		Currency:           currency.USDT,
		Amount:             1,
		Direction:          "OUT",
		AccountType:        "MAIN",
		SpecialUID:         "100000001",
		SpecialAccountType: "MAIN",
		ClientOrderID:      "e6c24d23-6bc2-401b-bf9e-55e2daddfbc1",
	}
	for _, tc := range []struct {
		field string
		clear func(*ExchangeBrokerTransferRequest)
		err   error
	}{
		{field: "Currency", clear: func(r *ExchangeBrokerTransferRequest) { r.Currency = currency.EMPTYCODE }, err: currency.ErrCurrencyCodeEmpty},
		{field: "Amount", clear: func(r *ExchangeBrokerTransferRequest) { r.Amount = 0 }, err: limits.ErrAmountBelowMin},
		{field: "Direction", clear: func(r *ExchangeBrokerTransferRequest) { r.Direction = "" }, err: errTransferDirectionRequired},
		{field: "AccountType", clear: func(r *ExchangeBrokerTransferRequest) { r.AccountType = "" }, err: errAccountTypeMissing},
		{field: "SpecialUID", clear: func(r *ExchangeBrokerTransferRequest) { r.SpecialUID = "" }, err: errSubUserIDRequired},
		{field: "SpecialAccountType", clear: func(r *ExchangeBrokerTransferRequest) { r.SpecialAccountType = "" }, err: errAccountTypeMissing},
		{field: "ClientOrderID", clear: func(r *ExchangeBrokerTransferRequest) { r.ClientOrderID = "" }, err: order.ErrClientOrderIDMustBeSet},
	} {
		req := valid
		tc.clear(&req)
		_, err = e.ExchangeBrokerTransfer(t.Context(), &req)
		require.ErrorIsf(t, err, tc.err, "ExchangeBrokerTransfer must reject a request without %s", tc.field)
	}

	sent := sentRequest(t, func(ex *Exchange) error {
		_, err := ex.ExchangeBrokerTransfer(t.Context(), &valid)
		return err
	})
	assert.JSONEq(t, `{
		"currency":"USDT","amount":"1","direction":"OUT","accountType":"MAIN","specialUid":"100000001","specialAccountType":"MAIN",
		"clientOid":"e6c24d23-6bc2-401b-bf9e-55e2daddfbc1"
	}`, sent.body, "ExchangeBrokerTransfer should send the amount as the string KuCoin documents")

	if !mockTests {
		t.Skip("the sub-account UID is an example, and a test must not move funds to an account it does not own")
	}
	result, err := e.ExchangeBrokerTransfer(t.Context(), &valid)
	require.NoError(t, err, "ExchangeBrokerTransfer must not error")
	exp := &ExchangeBrokerTransferResponse{OrderID: "671b4600c1e3dd000726866d"}
	assert.Equal(t, exp, result, "ExchangeBrokerTransfer should decode every field")
}

func TestGetExchangeBrokerTransferHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetExchangeBrokerTransferHistory(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetExchangeBrokerTransferHistory must reject an empty order ID")

	if !mockTests {
		t.Skip("the order ID is the docs' example, and no endpoint lists the broker's transfers to take a real one from")
	}
	result, err := e.GetExchangeBrokerTransferHistory(t.Context(), "671b4600c1e3dd000726866d")
	require.NoError(t, err, "GetExchangeBrokerTransferHistory must not error")
	exp := &ExchangeBrokerTransferHistoryResponse{
		OrderID:         "671b4600c1e3dd000726866d",
		Currency:        currency.USDT,
		Amount:          1,
		FromUID:         100000000,
		FromAccountType: "MAIN",
		FromAccountTag:  "DEFAULT",
		ToUID:           100000001,
		ToAccountType:   "MAIN",
		ToAccountTag:    "DEFAULT",
		Status:          "FAILURE",
		Reason:          "insufficient balance",
		CreatedAt:       types.Time(time.UnixMilli(1729840640000)),
	}
	assert.Equal(t, exp, result, "GetExchangeBrokerTransferHistory should decode every field")
}

func TestGetExchangeBrokerDepositList(t *testing.T) {
	t.Parallel()
	_, err := e.GetExchangeBrokerDepositList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetExchangeBrokerDepositList must reject a nil request")
	_, err = e.GetExchangeBrokerDepositList(t.Context(), &GetExchangeBrokerDepositListRequest{StartTimestamp: time.UnixMilli(1730505600000), EndTimestamp: time.UnixMilli(1730419200000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetExchangeBrokerDepositList must reject a reversed window")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetExchangeBrokerDepositList(t.Context(), &GetExchangeBrokerDepositListRequest{
		Currency:       currency.USDT,
		Status:         "SUCCESS",
		Hash:           "6724e363a492800007ec602b",
		StartTimestamp: time.UnixMilli(1730419200000),
		EndTimestamp:   time.UnixMilli(1730505600000),
		Limit:          100,
	})
	require.NoError(t, err, "GetExchangeBrokerDepositList must not error")
	if mockTests {
		exp := []ExchangeBrokerDepositResponse{
			{
				UID:             100000001,
				Hash:            "6724e363a492800007ec602b",
				Address:         "test@example.com",
				Memo:            "7025734",
				Amount:          3,
				Fee:             0.1,
				Currency:        currency.USDT,
				IsInner:         true,
				WalletTxID:      "bbbbbbbbb",
				Status:          "SUCCESS",
				Remark:          "Deposit",
				Chain:           "trx",
				CreatedAt:       types.Time(time.UnixMilli(1730470760000)),
				UpdatedAt:       types.Time(time.UnixMilli(1730470761000)),
				StatusRemark:    "credited",
				PreConfirms:     1,
				Confirms:        20,
				CurrentConfirms: 20,
			},
		}
		assert.Equal(t, exp, result, "GetExchangeBrokerDepositList should decode every field")
		return
	}
	assert.NotNil(t, result, "GetExchangeBrokerDepositList should return a result")
}

func TestGetExchangeBrokerDepositDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetExchangeBrokerDepositDetail(t.Context(), currency.EMPTYCODE, "30bb0e0b***4156c5188")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetExchangeBrokerDepositDetail must reject an empty currency")
	_, err = e.GetExchangeBrokerDepositDetail(t.Context(), currency.USDT, "")
	require.ErrorIs(t, err, errBrokerDepositHashRequired, "GetExchangeBrokerDepositDetail must reject an empty hash")

	ccy, hash := currency.USDT, "30bb0e0b***4156c5188"
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		deposits, err := e.GetExchangeBrokerDepositList(t.Context(), &GetExchangeBrokerDepositListRequest{Limit: 1})
		require.NoError(t, err, "GetExchangeBrokerDepositList must not error")
		if len(deposits) == 0 {
			t.Skip("the broker's sub-accounts have no deposits to look up")
		}
		ccy, hash = deposits[0].Currency, deposits[0].Hash
	}
	result, err := e.GetExchangeBrokerDepositDetail(t.Context(), ccy, hash)
	require.NoError(t, err, "GetExchangeBrokerDepositDetail must not error")
	if mockTests {
		exp := &ExchangeBrokerDepositDetailResponse{
			UID:             100000001,
			Hash:            "30bb0e0b***4156c5188",
			Address:         "THLPzUrbd1o***vP7d",
			Memo:            "7025734",
			Amount:          8.5,
			Fee:             0.5,
			Currency:        currency.USDT,
			IsInner:         true,
			WalletTxID:      "30bb0***610d1030f",
			Status:          "SUCCESS",
			Remark:          "Deposit",
			Chain:           "trx",
			CreatedAt:       types.Time(time.UnixMilli(1713429173000)),
			UpdatedAt:       types.Time(time.UnixMilli(1713429174000)),
			StatusRemark:    "credited",
			PreConfirms:     1,
			Confirms:        20,
			CurrentConfirms: 20,
		}
		assert.Equal(t, exp, result, "GetExchangeBrokerDepositDetail should decode every field")
		return
	}
	require.NotNil(t, result, "GetExchangeBrokerDepositDetail must return a result")
	assert.Equal(t, hash, result.Hash, "GetExchangeBrokerDepositDetail should return the deposit asked for")
}

func TestGetExchangeBrokerWithdrawDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetExchangeBrokerWithdrawDetail(t.Context(), "")
	require.ErrorIs(t, err, errBrokerWithdrawalIDRequired, "GetExchangeBrokerWithdrawDetail must reject an empty withdrawal ID")

	if !mockTests {
		t.Skip("the withdrawal ID is the docs' example, and no endpoint lists the broker's withdrawals to take a real one from")
	}
	result, err := e.GetExchangeBrokerWithdrawDetail(t.Context(), "66617a2***3c9a")
	require.NoError(t, err, "GetExchangeBrokerWithdrawDetail must not error")
	exp := &ExchangeBrokerWithdrawDetailResponse{
		ID:                   "66617a2***3c9a",
		Chain:                "ton",
		WalletTxID:           "AJ***eRI=",
		UID:                  100000001,
		UpdatedAt:            types.Time(time.UnixMilli(1717664375000)),
		Amount:               1,
		Memo:                 "7025734",
		Fee:                  0.1,
		Address:              "EQDn***dKbGzr",
		Remark:               "Withdrawal",
		IsInner:              true,
		Currency:             currency.USDT,
		Status:               "FAILURE",
		CreatedAt:            types.Time(time.UnixMilli(1717664288000)),
		FailureReason:        "ADDRESS_INVALID",
		FailureReasonMessage: "The withdrawal address is invalid",
	}
	assert.Equal(t, exp, result, "GetExchangeBrokerWithdrawDetail should decode every field")
}

func TestSetExchangeBrokerMarkupFee(t *testing.T) {
	t.Parallel()
	_, err := e.SetExchangeBrokerMarkupFee(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SetExchangeBrokerMarkupFee must reject a nil request")
	_, err = e.SetExchangeBrokerMarkupFee(t.Context(), &SetExchangeBrokerMarkupFeeRequest{EffectiveType: "DEFAULT", MakerMarkUp: 0.01, TakerMarkUp: 0.01})
	require.ErrorIs(t, err, errTradeTypeMissing, "SetExchangeBrokerMarkupFee must reject an empty trade type")
	_, err = e.SetExchangeBrokerMarkupFee(t.Context(), &SetExchangeBrokerMarkupFeeRequest{TradeType: "SPOT", MakerMarkUp: 0.01, TakerMarkUp: 0.01})
	require.ErrorIs(t, err, errBrokerEffectiveTypeRequired, "SetExchangeBrokerMarkupFee must reject an empty effective type")

	sent := sentRequest(t, func(ex *Exchange) error {
		_, err := ex.SetExchangeBrokerMarkupFee(t.Context(), &SetExchangeBrokerMarkupFeeRequest{
			SubUIDs:       []uint64{100000001, 100000002},
			EffectAt:      time.UnixMilli(1784801085000),
			TradeType:     "SPOT",
			MakerMarkUp:   0.005,
			TakerMarkUp:   0.01,
			EffectiveType: "UID_SPECIFIC",
		})
		return err
	})
	assert.JSONEq(t, `{
		"tradeType":"SPOT","effectiveType":"UID_SPECIFIC","makerMarkUp":0.005,"takerMarkUp":0.01,"subUids":[100000001,100000002],
		"effectAt":1784801085000
	}`, sent.body, "SetExchangeBrokerMarkupFee should send the mark-ups as the numbers KuCoin documents")
	sent = sentRequest(t, func(ex *Exchange) error {
		_, err := ex.SetExchangeBrokerMarkupFee(t.Context(), &SetExchangeBrokerMarkupFeeRequest{TradeType: "FUTURES", EffectiveType: "DEFAULT"})
		return err
	})
	assert.JSONEq(t, `{"tradeType":"FUTURES","effectiveType":"DEFAULT","makerMarkUp":0,"takerMarkUp":0}`, sent.body, "SetExchangeBrokerMarkupFee should send zero mark-ups, which cancel the mark-up")

	if !mockTests {
		t.Skip("setting a mark-up changes the fees the broker's users pay, and the UID-specific case names example sub-accounts")
	}
	for _, tc := range []struct {
		name string
		req  *SetExchangeBrokerMarkupFeeRequest
		exp  *SetExchangeBrokerMarkupFeeResponse
	}{
		{
			name: "DEFAULT",
			req: &SetExchangeBrokerMarkupFeeRequest{
				EffectAt:      time.UnixMilli(1784718285000),
				TradeType:     "SPOT",
				MakerMarkUp:   0.01,
				TakerMarkUp:   0.01,
				EffectiveType: "DEFAULT",
			},
			exp: &SetExchangeBrokerMarkupFeeResponse{
				EffectAt:      types.Time(time.UnixMilli(1784801085000)),
				CreatedAt:     types.Time(time.UnixMilli(1784718950707)),
				TradeType:     "SPOT",
				MakerMarkUp:   0.01,
				TakerMarkUp:   0.01,
				EffectiveType: "DEFAULT",
			},
		},
		{
			name: "UID_SPECIFIC",
			req: &SetExchangeBrokerMarkupFeeRequest{
				SubUIDs:       []uint64{100000001, 100000002},
				EffectAt:      time.UnixMilli(1784801085000),
				TradeType:     "SPOT",
				MakerMarkUp:   0.005,
				TakerMarkUp:   0.01,
				EffectiveType: "UID_SPECIFIC",
			},
			exp: &SetExchangeBrokerMarkupFeeResponse{
				SubUIDs:       []uint64{100000001, 100000002},
				EffectAt:      types.Time(time.UnixMilli(1784801085000)),
				CreatedAt:     types.Time(time.UnixMilli(1784719072007)),
				TradeType:     "SPOT",
				MakerMarkUp:   0.005,
				TakerMarkUp:   0.01,
				EffectiveType: "UID_SPECIFIC",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.SetExchangeBrokerMarkupFee(t.Context(), tc.req)
			require.NoError(t, err, "SetExchangeBrokerMarkupFee must not error")
			assert.Equal(t, tc.exp, result, "SetExchangeBrokerMarkupFee should decode every field")
		})
	}
}

func TestGetExchangeBrokerMarkupFee(t *testing.T) {
	t.Parallel()
	subUIDs := []uint64{100000001, 100000002}
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		uid, err := strconv.ParseUint(liveExchangeBrokerSubAccount(t).UID, 10, 64)
		require.NoError(t, err, "the sub-account UID must be an integer")
		subUIDs = []uint64{uid}
	}
	result, err := e.GetExchangeBrokerMarkupFee(t.Context(), subUIDs)
	require.NoError(t, err, "GetExchangeBrokerMarkupFee must not error")
	if mockTests {
		exp := &ExchangeBrokerMarkupFeeResponse{
			DefaultMarkUp: ExchangeBrokerDefaultMarkUp{
				UpdatedAt:          types.Time(time.UnixMilli(1784718950000)),
				SpotMakerMarkUp:    0.01,
				SpotTakerMarkUp:    0.01,
				FuturesMakerMarkUp: 0.0005,
				FuturesTakerMarkUp: 0.0008,
			},
			Items: []ExchangeBrokerSubAccountMarkUp{
				{
					UID:             100000001,
					UpdatedAt:       types.Time(time.UnixMilli(1784719072000)),
					SpotMakerMarkUp: 0.005,
					SpotTakerMarkUp: 0.01,
				},
				{
					UID:                100000002,
					UpdatedAt:          types.Time(time.UnixMilli(1784719072000)),
					SpotMakerMarkUp:    0.005,
					SpotTakerMarkUp:    0.01,
					FuturesMakerMarkUp: 0.0002,
					FuturesTakerMarkUp: 0.0004,
				},
			},
		}
		assert.Equal(t, exp, result, "GetExchangeBrokerMarkupFee should decode every field")
		return
	}
	assert.NotNil(t, result, "GetExchangeBrokerMarkupFee should return a result")
}
