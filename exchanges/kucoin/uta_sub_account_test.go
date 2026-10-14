package kucoin

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetUTASubAccountCurrencyAssets(t *testing.T) {
	t.Parallel()
	_, err := e.GetUTASubAccountCurrencyAssets(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUTASubAccountCurrencyAssets must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	exp := &UTASubAccountCurrencyAssetsResponse{
		Timestamp: types.Time(time.UnixMilli(1785234451817)),
		UserList: []UTASubAccountUserAssets{
			{
				UID: 100000003,
				Accounts: []UTASubAccountAssets{
					{
						AccountType: "SPOT",
						Currencies: []UTASubAccountCurrencyAsset{
							{Currency: currency.USDT, Locked: 1, Available: 200, Balance: 201},
						},
					},
					{
						AccountType:    "ISOLATED",
						SubAccountType: "AAVE-USDT",
						Currencies: []UTASubAccountCurrencyAsset{
							{
								Currency:           currency.USDT,
								Available:          8392,
								Balance:            8392,
								Liability:          100.5,
								LiabilityInterest:  0.5,
								LiabilityPrincipal: 100,
							},
						},
					},
					{
						AccountType: "FUTURES",
						Currencies: []UTASubAccountCurrencyAsset{
							{Currency: currency.USDT, Available: 783, Balance: 783, Equity: 790.5, UnrealisedPNL: 7.5},
						},
					},
				},
			},
		},
	}
	for name, req := range map[string]*UTASubAccountCurrencyAssetsRequest{
		"all":      {},
		"filtered": {UID: "100000003", PageSize: 50, LastID: 1000},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if !mockTests && req.UID != "" {
				req = &UTASubAccountCurrencyAssetsRequest{UID: strconv.FormatUint(utaLiveSubAccount(t).UID, 10), PageSize: 50}
			}
			result, err := e.GetUTASubAccountCurrencyAssets(t.Context(), req)
			require.NoError(t, err, "GetUTASubAccountCurrencyAssets must not error")
			if mockTests {
				assert.Equal(t, exp, result, "GetUTASubAccountCurrencyAssets should decode every field")
				return
			}
			assert.NotNil(t, result, "GetUTASubAccountCurrencyAssets should return a result")
		})
	}
}

func TestGetUTASubAccountList(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	exp := &UTASubAccountListResponse{
		CurrentPage: 1,
		PageSize:    10,
		TotalNumber: 2,
		TotalPage:   1,
		Items: []UTASubAccount{
			{
				UID:              100000003,
				SubName:          "robot8795000",
				Status:           2,
				Type:             1,
				Access:           "All",
				CreatedAt:        types.Time(time.UnixMilli(1775111561000)),
				Remarks:          "robot sub user",
				TradeTypes:       []string{"Spot"},
				OpenedTradeTypes: []string{"Spot", "Futures"},
			},
			{
				UserID:           "69e1f2a3b4c5d6e7f80119a4",
				UID:              100000004,
				SubName:          "oncsubaccount",
				Status:           3,
				Type:             5,
				Access:           "All",
				CreatedAt:        types.Time(time.UnixMilli(1776409601000)),
				Remarks:          "test",
				TradeTypes:       []string{"Spot", "Futures", "Margin"},
				OpenedTradeTypes: []string{"Spot", "Futures", "Margin"},
				HostedStatus:     2,
			},
		},
	}
	for _, tc := range []struct {
		name                  string
		currentPage, pageSize uint64
	}{
		{name: "defaults"},
		{name: "paged", currentPage: 1, pageSize: 10},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetUTASubAccountList(t.Context(), tc.currentPage, tc.pageSize)
			require.NoError(t, err, "GetUTASubAccountList must not error")
			if mockTests {
				assert.Equal(t, exp, result, "GetUTASubAccountList should decode every field")
				return
			}
			assert.NotNil(t, result, "GetUTASubAccountList should return a result")
		})
	}
}

// utaLiveSubAccount returns one of the account's sub-accounts for a live test, skipping the test when it has none
func utaLiveSubAccount(t *testing.T) *UTASubAccount {
	t.Helper()
	subs, err := e.GetUTASubAccountList(t.Context(), 0, 0)
	require.NoError(t, err, "GetUTASubAccountList must not error")
	require.NotNil(t, subs, "GetUTASubAccountList must return a page")
	if len(subs.Items) == 0 {
		t.Skip("The account has no sub-accounts")
	}
	return &subs.Items[0]
}

func TestGetUTASubAccountAPIList(t *testing.T) {
	t.Parallel()
	_, err := e.GetUTASubAccountAPIList(t.Context(), "670621e3a25958000159c82f", "")
	require.ErrorIs(t, err, errInvalidSubAccountName, "GetUTASubAccountAPIList must reject an empty sub-account name")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		sub := utaLiveSubAccount(t)
		result, err := e.GetUTASubAccountAPIList(t.Context(), "", sub.SubName)
		require.NoError(t, err, "GetUTASubAccountAPIList must not error")
		for i := range result {
			assert.Equalf(t, sub.SubName, result[i].SubName, "GetUTASubAccountAPIList should return key %d of the sub-account asked for", i)
		}
		return
	}
	keys := []UTASubAccountAPI{
		{
			SubName:     "test6662134",
			Remark:      "TheRemark3",
			APIKey:      "670621e3a25958000159c82f",
			APIVersion:  3,
			Permission:  "General,Futures,Unified,Spot",
			IPWhitelist: "192.0.2.1",
			CreatedAt:   types.Time(time.UnixMilli(1773045560000)),
			UID:         100000003,
			IsMaster:    true,
		},
		{
			SubName:     "test6662134",
			Remark:      "TheRemark2",
			APIKey:      "670621e3a25958000159c830",
			APIVersion:  3,
			Permission:  "General,Spot",
			IPWhitelist: "192.0.2.2",
			CreatedAt:   types.Time(time.UnixMilli(1773045518000)),
			UID:         100000003,
		},
		{
			SubName:     "test6662134",
			Remark:      "TheRemark",
			APIKey:      "670621e3a25958000159c831",
			APIVersion:  3,
			Permission:  "General,Futures,Unified,Spot",
			IPWhitelist: "192.0.2.3",
			CreatedAt:   types.Time(time.UnixMilli(1773045502000)),
			UID:         100000003,
		},
	}
	for _, tc := range []struct {
		name   string
		apiKey string
		exp    []UTASubAccountAPI
	}{
		{name: "every key", exp: keys},
		{name: "one key", apiKey: "670621e3a25958000159c82f", exp: keys[:1]},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetUTASubAccountAPIList(t.Context(), tc.apiKey, "test6662134")
			require.NoError(t, err, "GetUTASubAccountAPIList must not error")
			assert.Equal(t, tc.exp, result, "GetUTASubAccountAPIList should decode every field")
		})
	}
}

func TestSetUTASubAccountTransferPermission(t *testing.T) {
	t.Parallel()
	_, err := e.SetUTASubAccountTransferPermission(t.Context(), nil, true)
	require.ErrorIs(t, err, errSubUserIDRequired, "SetUTASubAccountTransferPermission must reject no sub-account UIDs")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SetUTASubAccountTransferPermission(t.Context(), []string{"100000003", "100000004"}, true)
	require.NoError(t, err, "SetUTASubAccountTransferPermission must not error")
	if mockTests {
		exp := []UTASubAccountTransferPermission{
			{SubUID: "100000003", SubToSub: true},
			{SubUID: "100000004", SubToSub: true},
		}
		assert.Equal(t, exp, result, "SetUTASubAccountTransferPermission should decode every field")
		return
	}
	assert.Len(t, result, 2, "SetUTASubAccountTransferPermission should return each sub-account's permission")
}

func TestAddUTASubAccount(t *testing.T) {
	t.Parallel()
	_, err := e.AddUTASubAccount(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "AddUTASubAccount must reject a nil request")
	_, err = e.AddUTASubAccount(t.Context(), &AddUTASubAccountRequest{SubName: "modeprobe9002", Access: "Spot"})
	require.ErrorIs(t, err, errPasswordRequired, "AddUTASubAccount must reject an empty password")
	_, err = e.AddUTASubAccount(t.Context(), &AddUTASubAccountRequest{Password: "Mock1234pw", Access: "Spot"})
	require.ErrorIs(t, err, errInvalidSubAccountName, "AddUTASubAccount must reject an empty sub-account name")
	_, err = e.AddUTASubAccount(t.Context(), &AddUTASubAccountRequest{Password: "Mock1234pw", SubName: "modeprobe9002"})
	require.ErrorIs(t, err, errAccessRequired, "AddUTASubAccount must reject empty access")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name string
		req  *AddUTASubAccountRequest
		exp  *AddUTASubAccountResponse
	}{
		{
			name: "required",
			req:  &AddUTASubAccountRequest{Password: "Mock1234pw", SubName: "modeprobe9002", Access: "Spot"},
			exp:  &AddUTASubAccountResponse{UID: 100000007, SubName: "modeprobe9002", Access: "Spot"},
		},
		{
			name: "every",
			req:  &AddUTASubAccountRequest{Password: "Mock1234pw", Remarks: "modeprobe", SubName: "modeprobe9001", Access: "Spot,Futures", Mode: "UNIFIED"},
			exp:  &AddUTASubAccountResponse{UID: 100000006, SubName: "modeprobe9001", Remarks: "modeprobe", Access: "Spot,Futures"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.AddUTASubAccount(t.Context(), tc.req)
			require.NoError(t, err, "AddUTASubAccount must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "AddUTASubAccount should decode every field")
				return
			}
			assert.Equal(t, tc.req.SubName, result.SubName, "AddUTASubAccount should return the sub-account name")
		})
	}
}

func TestAddUTASubAccountAPI(t *testing.T) {
	t.Parallel()
	_, err := e.AddUTASubAccountAPI(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "AddUTASubAccountAPI must reject a nil request")
	_, err = e.AddUTASubAccountAPI(t.Context(), &AddUTASubAccountAPIRequest{Passphrase: "mock-passphrase", Remark: "TheRemark2115"})
	require.ErrorIs(t, err, errInvalidSubAccountName, "AddUTASubAccountAPI must reject an empty sub-account name")
	_, err = e.AddUTASubAccountAPI(t.Context(), &AddUTASubAccountAPIRequest{SubName: "testapi2", Remark: "TheRemark2115"})
	require.ErrorIs(t, err, errInvalidPassPhraseInstance, "AddUTASubAccountAPI must reject an empty passphrase")
	_, err = e.AddUTASubAccountAPI(t.Context(), &AddUTASubAccountAPIRequest{SubName: "testapi2", Passphrase: "mock-passphrase"})
	require.ErrorIs(t, err, errRemarkIsRequired, "AddUTASubAccountAPI must reject an empty remark")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name string
		req  *AddUTASubAccountAPIRequest
		exp  *AddUTASubAccountAPIResponse
	}{
		{
			name: "required",
			req:  &AddUTASubAccountAPIRequest{SubName: "testapi2", Passphrase: "mock-passphrase", Remark: "TheRemark2115"},
			exp: &AddUTASubAccountAPIResponse{
				SubName:    "testapi2",
				Remark:     "TheRemark2115",
				APIKey:     "670621e3a25958000159c831",
				APISecret:  "********",
				APIVersion: 3,
				Passphrase: "********",
				Permission: "General",
				CreatedAt:  types.Time(time.UnixMilli(1776845099000)),
			},
		},
		{
			name: "every",
			req: &AddUTASubAccountAPIRequest{
				SubName:     "testapi2",
				Passphrase:  "mock-passphrase",
				Remark:      "TheRemark2114",
				Permission:  "General,Spot,Futures,Unified",
				IPWhitelist: "192.0.2.1",
				Expire:      "90",
			},
			exp: &AddUTASubAccountAPIResponse{
				SubName:     "testapi2",
				Remark:      "TheRemark2114",
				APIKey:      "670621e3a25958000159c82f",
				APISecret:   "********",
				APIVersion:  3,
				Passphrase:  "********",
				Permission:  "General,Futures,Unified,Spot",
				IPWhitelist: "192.0.2.1",
				CreatedAt:   types.Time(time.UnixMilli(1776845047000)),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.AddUTASubAccountAPI(t.Context(), tc.req)
			require.NoError(t, err, "AddUTASubAccountAPI must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "AddUTASubAccountAPI should decode every field")
				return
			}
			assert.NotEmpty(t, result.APIKey, "AddUTASubAccountAPI should return the API key")
		})
	}
}

func TestModifyUTASubAccountAPI(t *testing.T) {
	t.Parallel()
	_, err := e.ModifyUTASubAccountAPI(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ModifyUTASubAccountAPI must reject a nil request")
	_, err = e.ModifyUTASubAccountAPI(t.Context(), &ModifyUTASubAccountAPIRequest{APIKey: "670621e3a25958000159c82f", Passphrase: "mock-passphrase"})
	require.ErrorIs(t, err, errInvalidSubAccountName, "ModifyUTASubAccountAPI must reject an empty sub-account name")
	_, err = e.ModifyUTASubAccountAPI(t.Context(), &ModifyUTASubAccountAPIRequest{SubName: "testapi2", Passphrase: "mock-passphrase"})
	require.ErrorIs(t, err, errAPIKeyRequired, "ModifyUTASubAccountAPI must reject an empty API key")
	_, err = e.ModifyUTASubAccountAPI(t.Context(), &ModifyUTASubAccountAPIRequest{SubName: "testapi2", APIKey: "670621e3a25958000159c82f"})
	require.ErrorIs(t, err, errInvalidPassPhraseInstance, "ModifyUTASubAccountAPI must reject an empty passphrase")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name string
		req  *ModifyUTASubAccountAPIRequest
		exp  *ModifyUTASubAccountAPIResponse
	}{
		{
			name: "required",
			req:  &ModifyUTASubAccountAPIRequest{SubName: "testapi2", APIKey: "670621e3a25958000159c82f", Passphrase: "mock-passphrase"},
			exp:  &ModifyUTASubAccountAPIResponse{SubName: "testapi2", APIKey: "670621e3a25958000159c82f", Permission: "General,Futures,Unified,Spot"},
		},
		{
			name: "every",
			req: &ModifyUTASubAccountAPIRequest{
				SubName:     "testapi2",
				APIKey:      "670621e3a25958000159c82f",
				Passphrase:  "mock-passphrase",
				Permission:  "General,Spot,Unified,InnerTransfer",
				IPWhitelist: "192.0.2.1",
				Expire:      "-1",
			},
			exp: &ModifyUTASubAccountAPIResponse{
				SubName:     "testapi2",
				APIKey:      "670621e3a25958000159c82f",
				Permission:  "General,Unified,Spot,InnerTransfer",
				IPWhitelist: "192.0.2.1",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.ModifyUTASubAccountAPI(t.Context(), tc.req)
			require.NoError(t, err, "ModifyUTASubAccountAPI must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "ModifyUTASubAccountAPI should decode every field")
				return
			}
			assert.Equal(t, tc.req.APIKey, result.APIKey, "ModifyUTASubAccountAPI should return the API key")
		})
	}
}

func TestSetUTASubAccountsAPIRateLimit(t *testing.T) {
	t.Parallel()
	_, err := e.SetUTASubAccountsAPIRateLimit(t.Context(), nil)
	require.ErrorIs(t, err, errNoRateLimits, "SetUTASubAccountsAPIRateLimit must reject no limits")
	_, err = e.SetUTASubAccountsAPIRateLimit(t.Context(), []SetUTASubAccountAPIRateLimitRequest{{Rate: 50}})
	require.ErrorIs(t, err, errUIDRequired, "SetUTASubAccountsAPIRateLimit must reject an empty UID")
	_, err = e.SetUTASubAccountsAPIRateLimit(t.Context(), []SetUTASubAccountAPIRateLimitRequest{{UID: "100000003"}})
	require.ErrorIs(t, err, errRateRequired, "SetUTASubAccountsAPIRateLimit must reject a zero rate")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SetUTASubAccountsAPIRateLimit(t.Context(), []SetUTASubAccountAPIRateLimitRequest{
		{UID: "100000003", Rate: 50},
		{UID: "100000004", Rate: 300},
	})
	require.NoError(t, err, "SetUTASubAccountsAPIRateLimit must not error")
	if mockTests {
		exp := &SetUTASubAccountsAPIRateLimitResponse{
			Items: []UTASubAccountAPIRateLimitResult{
				{UID: "100000003", Rate: 50, Code: "200000", Message: "success"},
				{UID: "100000004", Rate: 300, Code: "501001", Message: "Requested limit exceeds maximum allowed per user"},
			},
		}
		assert.Equal(t, exp, result, "SetUTASubAccountsAPIRateLimit should decode every field")
		return
	}
	assert.Len(t, result.Items, 2, "SetUTASubAccountsAPIRateLimit should return each limit's outcome")
}

func TestDeleteUTASubAccountAPI(t *testing.T) {
	t.Parallel()
	_, err := e.DeleteUTASubAccountAPI(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "DeleteUTASubAccountAPI must reject a nil request")
	_, err = e.DeleteUTASubAccountAPI(t.Context(), &DeleteUTASubAccountAPIRequest{SubName: "testapi6", Passphrase: "mock-passphrase"})
	require.ErrorIs(t, err, errAPIKeyRequired, "DeleteUTASubAccountAPI must reject an empty API key")
	_, err = e.DeleteUTASubAccountAPI(t.Context(), &DeleteUTASubAccountAPIRequest{APIKey: "670621e3a25958000159c82f", Passphrase: "mock-passphrase"})
	require.ErrorIs(t, err, errInvalidSubAccountName, "DeleteUTASubAccountAPI must reject an empty sub-account name")
	_, err = e.DeleteUTASubAccountAPI(t.Context(), &DeleteUTASubAccountAPIRequest{APIKey: "670621e3a25958000159c82f", SubName: "testapi6"})
	require.ErrorIs(t, err, errInvalidPassPhraseInstance, "DeleteUTASubAccountAPI must reject an empty passphrase")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.DeleteUTASubAccountAPI(t.Context(), &DeleteUTASubAccountAPIRequest{APIKey: "670621e3a25958000159c82f", SubName: "testapi6", Passphrase: "mock-passphrase"})
	require.NoError(t, err, "DeleteUTASubAccountAPI must not error")
	if mockTests {
		assert.Equal(t, &DeleteUTASubAccountAPIResponse{SubName: "testapi6", APIKey: "670621e3a25958000159c82f"}, result, "DeleteUTASubAccountAPI should decode every field")
		return
	}
	assert.Equal(t, "670621e3a25958000159c82f", result.APIKey, "DeleteUTASubAccountAPI should return the deleted API key")
}
