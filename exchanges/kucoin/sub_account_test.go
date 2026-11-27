package kucoin

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestCreateSubUser(t *testing.T) {
	t.Parallel()
	_, err := e.CreateSubUser(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateSubUser must reject a nil request")
	_, err = e.CreateSubUser(t.Context(), &CreateSubUserRequest{Password: "Subaccount1", Access: "Spot"})
	require.ErrorIs(t, err, errInvalidSubAccountName, "CreateSubUser must reject an empty sub-account name")
	_, err = e.CreateSubUser(t.Context(), &CreateSubUserRequest{SubName: "Subaccount-2", Access: "Spot"})
	require.ErrorIs(t, err, errSubAccountPasswordRequired, "CreateSubUser must reject an empty password")
	_, err = e.CreateSubUser(t.Context(), &CreateSubUserRequest{SubName: "Subaccount-2", Password: "Subaccount1"})
	require.ErrorIs(t, err, errSubAccountAccessRequired, "CreateSubUser must reject empty access permissions")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CreateSubUser(t.Context(), &CreateSubUserRequest{
		SubName:  "Subaccount-2",
		Password: "Subaccount1",
		Remarks:  "TheRemark",
		Access:   "Spot",
		Mode:     "CLASSIC",
	})
	require.NoError(t, err, "CreateSubUser must not error")
	if mockTests {
		exp := &CreateSubUserResponse{UID: 100000003, SubName: "Subaccount-2", Remarks: "TheRemark", Access: "Spot"}
		assert.Equal(t, exp, result, "CreateSubUser should decode every field")
		return
	}
	require.NotNil(t, result, "CreateSubUser must return a result")
	assert.NotZero(t, result.UID, "CreateSubUser should return the sub-account UID")
}

func TestAddSubAccountMarginPermission(t *testing.T) {
	t.Parallel()
	err := e.AddSubAccountMarginPermission(t.Context(), 0)
	require.ErrorIs(t, err, errSubAccountUIDRequired, "AddSubAccountMarginPermission must reject an empty UID")

	if !mockTests {
		t.Skip("the sub-account UID is an example, and a test must not change a real sub-account's permissions")
	}
	err = e.AddSubAccountMarginPermission(t.Context(), 100000003)
	assert.NoError(t, err, "AddSubAccountMarginPermission should not error")
}

func TestAddSubAccountFuturesPermission(t *testing.T) {
	t.Parallel()
	err := e.AddSubAccountFuturesPermission(t.Context(), 0)
	require.ErrorIs(t, err, errSubAccountUIDRequired, "AddSubAccountFuturesPermission must reject an empty UID")

	if !mockTests {
		t.Skip("the sub-account UID is an example, and a test must not change a real sub-account's permissions")
	}
	err = e.AddSubAccountFuturesPermission(t.Context(), 100000003)
	assert.NoError(t, err, "AddSubAccountFuturesPermission should not error")
}

func TestGetAllSubAccountsInfoV2(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name                  string
		currentPage, pageSize uint64
		exp                   *AllSubAccountsInfoV2Response
	}{
		{
			name:     "first page",
			pageSize: 30,
			exp: &AllSubAccountsInfoV2Response{
				CurrentPage: 1,
				PageSize:    30,
				TotalNumber: 1,
				TotalPage:   1,
				Items: []SubAccountSummary{
					{
						UserID:           "63743f07e0c5230001761d08",
						UID:              100000004,
						SubName:          "testapi6",
						Status:           2,
						Type:             1,
						Access:           "All",
						CreatedAt:        types.Time(time.UnixMilli(1668562696000)),
						Remarks:          "remarks",
						TradeTypes:       []string{"Spot", "Futures", "Margin"},
						OpenedTradeTypes: []string{"Spot"},
						HostedStatus:     "HOSTED",
					},
				},
			},
		},
		{
			name:        "page one",
			currentPage: 1,
			pageSize:    100,
			exp: &AllSubAccountsInfoV2Response{
				CurrentPage: 1,
				PageSize:    100,
				TotalNumber: 1,
				TotalPage:   1,
				Items: []SubAccountSummary{
					{
						UserID:           "670538a31037eb000115b076",
						UID:              100000005,
						SubName:          "Name1234567",
						Status:           3,
						Access:           "Spot",
						CreatedAt:        types.Time(time.UnixMilli(1728395427000)),
						Remarks:          "TheRemark",
						TradeTypes:       []string{"Spot"},
						OpenedTradeTypes: []string{"Spot"},
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetAllSubAccountsInfoV2(t.Context(), tc.currentPage, tc.pageSize)
			require.NoError(t, err, "GetAllSubAccountsInfoV2 must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetAllSubAccountsInfoV2 should decode every field")
				return
			}
			require.NotNil(t, result, "GetAllSubAccountsInfoV2 must return a result")
			assert.Equal(t, tc.pageSize, result.PageSize, "GetAllSubAccountsInfoV2 should return the requested page size")
		})
	}
}

// liveSubAccount returns the first of the account's sub-accounts, so live tests look up one the account owns, and
// skips the test when there is none
func liveSubAccount(t *testing.T) SubAccountSummary {
	t.Helper()
	page, err := e.GetAllSubAccountsInfoV2(t.Context(), 1, 1)
	require.NoError(t, err, "GetAllSubAccountsInfoV2 must not error")
	require.NotNil(t, page, "GetAllSubAccountsInfoV2 must return a page")
	if len(page.Items) == 0 {
		t.Skip("the account has no sub-accounts to look up")
	}
	return page.Items[0]
}

func TestGetSubAccountDetailBalance(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAccountDetailBalance(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSubAccountDetailBalance must reject a nil request")
	_, err = e.GetSubAccountDetailBalance(t.Context(), &SubAccountDetailBalanceRequest{})
	require.ErrorIs(t, err, errSubUserIDRequired, "GetSubAccountDetailBalance must reject an empty sub-user ID")

	subUserID := "5caefba7d9575a0688f83c45"
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		subUserID = liveSubAccount(t).UserID
	}
	marginAccounts := []SubAccountCurrencyBalance{
		{
			Currency:          currency.BTC,
			Balance:           0.001,
			Available:         0.0008,
			Holds:             0.0002,
			BaseCurrency:      currency.BTC,
			BaseCurrencyPrice: 1,
			BaseAmount:        0.001,
			Tag:               "DEFAULT",
		},
	}
	for _, tc := range []struct {
		name string
		req  *SubAccountDetailBalanceRequest
		exp  *SubAccountDetailBalanceResponse
	}{
		{
			name: "all",
			req:  &SubAccountDetailBalanceRequest{SubUserID: subUserID},
			exp: &SubAccountDetailBalanceResponse{
				SubUserID: "5caefba7d9575a0688f83c45",
				SubName:   "testapi6",
				MainAccounts: []SubAccountCurrencyBalance{
					{
						Currency:          currency.USDT,
						Balance:           0.01,
						Available:         0.005,
						Holds:             0.005,
						BaseCurrency:      currency.BTC,
						BaseCurrencyPrice: 62384.3,
						BaseAmount:        0.00000016,
						Tag:               "DEFAULT",
					},
				},
				TradeAccounts: []SubAccountCurrencyBalance{
					{
						Currency:          currency.USDT,
						Balance:           0.02,
						Available:         0.01,
						Holds:             0.01,
						BaseCurrency:      currency.BTC,
						BaseCurrencyPrice: 62384.3,
						BaseAmount:        0.00000032,
						Tag:               "DEFAULT",
					},
				},
				MarginAccounts:  marginAccounts,
				TradeHFAccounts: []string{"USDT"},
			},
		},
		{
			name: "filtered",
			req: &SubAccountDetailBalanceRequest{
				SubUserID:         subUserID,
				IncludeBaseAmount: true,
				BaseCurrency:      currency.BTC,
				BaseAmount:        0.0001,
			},
			exp: &SubAccountDetailBalanceResponse{
				SubUserID:       "5caefba7d9575a0688f83c45",
				SubName:         "testapi6",
				MainAccounts:    []SubAccountCurrencyBalance{},
				TradeAccounts:   []SubAccountCurrencyBalance{},
				MarginAccounts:  marginAccounts,
				TradeHFAccounts: []string{},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetSubAccountDetailBalance(t.Context(), tc.req)
			require.NoError(t, err, "GetSubAccountDetailBalance must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetSubAccountDetailBalance should decode every field")
				return
			}
			require.NotNil(t, result, "GetSubAccountDetailBalance must return a result")
			assert.Equal(t, tc.req.SubUserID, result.SubUserID, "GetSubAccountDetailBalance should return the requested sub-account")
		})
	}
}

func TestGetAllSubAccountsBalanceV2(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name                  string
		currentPage, pageSize uint64
		exp                   *AllSubAccountsBalanceV2Response
	}{
		{
			name: "default page",
			exp: &AllSubAccountsBalanceV2Response{
				CurrentPage: 1,
				PageSize:    10,
				TotalNumber: 2,
				TotalPage:   1,
				Items: []SubAccountBalanceResponse{
					{
						SubUserID: "63743f07e0c5230001761d08",
						SubName:   "testapi6",
						MainAccounts: []SubAccountCurrencyBalance{
							{
								Currency:          currency.USDT,
								Balance:           0.01,
								Available:         0.005,
								Holds:             0.005,
								BaseCurrency:      currency.BTC,
								BaseCurrencyPrice: 62514.5,
								BaseAmount:        0.00000015,
								Tag:               "DEFAULT",
							},
						},
						TradeAccounts: []SubAccountCurrencyBalance{
							{
								Currency:          currency.USDT,
								Balance:           0.01,
								Available:         0.008,
								Holds:             0.002,
								BaseCurrency:      currency.BTC,
								BaseCurrencyPrice: 62514.5,
								BaseAmount:        0.00000015,
								Tag:               "DEFAULT",
							},
						},
						MarginAccounts: []SubAccountCurrencyBalance{
							{
								Currency:          currency.USDT,
								Balance:           0.01,
								Available:         0.007,
								Holds:             0.003,
								BaseCurrency:      currency.BTC,
								BaseCurrencyPrice: 62514.5,
								BaseAmount:        0.00000015,
								Tag:               "DEFAULT",
							},
						},
						TradeHFAccounts: []string{"USDT"},
					},
					{
						SubUserID:       "670538a31037eb000115b076",
						SubName:         "Name1234567",
						MainAccounts:    []SubAccountCurrencyBalance{},
						TradeAccounts:   []SubAccountCurrencyBalance{},
						MarginAccounts:  []SubAccountCurrencyBalance{},
						TradeHFAccounts: []string{},
					},
				},
			},
		},
		{
			name:        "page two",
			currentPage: 2,
			pageSize:    10,
			exp: &AllSubAccountsBalanceV2Response{
				CurrentPage: 2,
				PageSize:    10,
				TotalNumber: 11,
				TotalPage:   2,
				Items: []SubAccountBalanceResponse{
					{
						SubUserID:       "66b0c0905fc1480001c14c36",
						SubName:         "LTkucoin1491",
						MainAccounts:    []SubAccountCurrencyBalance{},
						TradeAccounts:   []SubAccountCurrencyBalance{},
						MarginAccounts:  []SubAccountCurrencyBalance{},
						TradeHFAccounts: []string{},
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetAllSubAccountsBalanceV2(t.Context(), tc.currentPage, tc.pageSize)
			require.NoError(t, err, "GetAllSubAccountsBalanceV2 must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetAllSubAccountsBalanceV2 should decode every field")
				return
			}
			require.NotNil(t, result, "GetAllSubAccountsBalanceV2 must return a result")
			assert.NotZero(t, result.PageSize, "GetAllSubAccountsBalanceV2 should return the page size")
		})
	}
}

func TestGetAllFuturesSubAccountBalances(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAllFuturesSubAccountBalances(t.Context(), currency.USDT)
	require.NoError(t, err, "GetAllFuturesSubAccountBalances must not error")
	if mockTests {
		exp := &AllFuturesSubAccountBalancesResponse{
			Summary: FuturesSubAccountBalanceSummary{
				AccountEquityTotal:    103.899081508,
				UnrealisedPNLTotal:    38.81075,
				MarginBalanceTotal:    65.336985668,
				PositionMarginTotal:   68.9588320683,
				OrderMarginTotal:      1.7,
				FrozenFundsTotal:      0.3,
				AvailableBalanceTotal: 67.2492494397,
				Currency:              currency.USDT,
			},
			Accounts: []FuturesSubAccountBalance{
				{
					AccountName:      "testapi6",
					AccountEquity:    27.30545128,
					UnrealisedPNL:    22.549,
					MarginBalance:    4.75645128,
					PositionMargin:   24.1223749975,
					OrderMargin:      0.5,
					FrozenFunds:      0.1,
					AvailableBalance: 25.7320762825,
					Currency:         currency.USDT,
				},
				{
					AccountName:      "main",
					AccountEquity:    76.593630228,
					UnrealisedPNL:    16.26175,
					MarginBalance:    60.580534388,
					PositionMargin:   44.8364570708,
					OrderMargin:      1.2,
					FrozenFunds:      0.2,
					AvailableBalance: 41.5171731572,
					Currency:         currency.USDT,
				},
			},
		}
		assert.Equal(t, exp, result, "GetAllFuturesSubAccountBalances should decode every field")
		return
	}
	require.NotNil(t, result, "GetAllFuturesSubAccountBalances must return a result")
	assert.True(t, result.Summary.Currency.Equal(currency.USDT), "GetAllFuturesSubAccountBalances should return the USDT totals")
}

func TestGetSubAccountAPIList(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAccountAPIList(t.Context(), "", "")
	require.ErrorIs(t, err, errInvalidSubAccountName, "GetSubAccountAPIList must reject an empty sub-account name")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		subName := liveSubAccount(t).SubName
		keys, err := e.GetSubAccountAPIList(t.Context(), subName, "")
		require.NoError(t, err, "GetSubAccountAPIList must not error")
		if len(keys) == 0 {
			t.Skip("the sub-account has no API keys to look up")
		}
		one, err := e.GetSubAccountAPIList(t.Context(), subName, keys[0].APIKey)
		require.NoError(t, err, "GetSubAccountAPIList must not error for one key")
		require.Len(t, one, 1, "GetSubAccountAPIList must return the key asked for")
		assert.Equal(t, keys[0].APIKey, one[0].APIKey, "GetSubAccountAPIList should return the key asked for")
		return
	}
	// Every key of a sub-account has IsMaster false
	tradingKey := SubAccountAPIKeyResponse{
		SubName:     "Sam",
		Remark:      "trading",
		APIKey:      "673eab2a955ebf000195d7e5",
		APIVersion:  3,
		Permission:  "General,Spot,Futures",
		IPWhitelist: "192.0.2.1,198.51.100.2",
		CreatedAt:   types.Time(time.UnixMilli(1732160398000)),
		UID:         100000006,
	}
	for _, tc := range []struct {
		name   string
		apiKey string
		exp    []SubAccountAPIKeyResponse
	}{
		{
			name: "every key",
			exp: []SubAccountAPIKeyResponse{
				{
					SubName:     "Sam",
					Remark:      "sdk_test_integration",
					APIKey:      "673eab2a955ebf000195d7e4",
					APIVersion:  3,
					Permission:  "General",
					IPWhitelist: "192.0.2.1",
					CreatedAt:   types.Time(time.UnixMilli(1732160298000)),
					UID:         100000006,
				},
				tradingKey,
			},
		},
		{name: "one key", apiKey: "673eab2a955ebf000195d7e5", exp: []SubAccountAPIKeyResponse{tradingKey}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetSubAccountAPIList(t.Context(), "Sam", tc.apiKey)
			require.NoError(t, err, "GetSubAccountAPIList must not error")
			assert.Equal(t, tc.exp, result, "GetSubAccountAPIList should decode every field")
		})
	}
}

func TestCreateSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	_, err := e.CreateSubAccountAPIKey(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateSubAccountAPIKey must reject a nil request")
	_, err = e.CreateSubAccountAPIKey(t.Context(), &CreateSubAccountAPIKeyRequest{Passphrase: "mysecretPassphrase123", Remark: "the-remark"})
	require.ErrorIs(t, err, errInvalidSubAccountName, "CreateSubAccountAPIKey must reject an empty sub-account name")
	_, err = e.CreateSubAccountAPIKey(t.Context(), &CreateSubAccountAPIKeyRequest{SubName: "gocryptoTrader1", Remark: "the-remark"})
	require.ErrorIs(t, err, errInvalidPassPhraseInstance, "CreateSubAccountAPIKey must reject an empty passphrase")
	_, err = e.CreateSubAccountAPIKey(t.Context(), &CreateSubAccountAPIKeyRequest{SubName: "gocryptoTrader1", Passphrase: "mysecretPassphrase123"})
	require.ErrorIs(t, err, errRemarkIsRequired, "CreateSubAccountAPIKey must reject an empty remark")

	if !mockTests {
		t.Skip("the sub-account name is an example, and a test must not leave an API key on a real sub-account")
	}
	result, err := e.CreateSubAccountAPIKey(t.Context(), &CreateSubAccountAPIKeyRequest{
		SubName:     "gocryptoTrader1",
		Passphrase:  "mysecretPassphrase123",
		Remark:      "the-remark",
		Permission:  "General,Spot",
		IPWhitelist: "192.0.2.1,198.51.100.2",
		Expire:      "30",
	})
	require.NoError(t, err, "CreateSubAccountAPIKey must not error")
	exp := &CreateSubAccountAPIKeyResponse{
		SubName:     "gocryptoTrader1",
		Remark:      "the-remark",
		APIKey:      "69d6606fd5d5ab0001a44645",
		APISecret:   "1b4ae89c-0000-4000-8000-00000023dbc0",
		APIVersion:  3,
		Passphrase:  "mysecretPassphrase123",
		Permission:  "General,Spot",
		IPWhitelist: "192.0.2.1,198.51.100.2",
		CreatedAt:   types.Time(time.UnixMilli(1775657071000)),
	}
	assert.Equal(t, exp, result, "CreateSubAccountAPIKey should decode every field")
}

func TestModifySubAccountAPIKey(t *testing.T) {
	t.Parallel()
	_, err := e.ModifySubAccountAPIKey(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ModifySubAccountAPIKey must reject a nil request")
	_, err = e.ModifySubAccountAPIKey(t.Context(), &ModifySubAccountAPIKeyRequest{APIKey: "65e7f22077172b0001f9ee41", Passphrase: "mysecretPassphrase123"})
	require.ErrorIs(t, err, errInvalidSubAccountName, "ModifySubAccountAPIKey must reject an empty sub-account name")
	_, err = e.ModifySubAccountAPIKey(t.Context(), &ModifySubAccountAPIKeyRequest{SubName: "gocryptoTrader1", Passphrase: "mysecretPassphrase123"})
	require.ErrorIs(t, err, errAPIKeyRequired, "ModifySubAccountAPIKey must reject an empty API key")
	_, err = e.ModifySubAccountAPIKey(t.Context(), &ModifySubAccountAPIKeyRequest{SubName: "gocryptoTrader1", APIKey: "65e7f22077172b0001f9ee41"})
	require.ErrorIs(t, err, errInvalidPassPhraseInstance, "ModifySubAccountAPIKey must reject an empty passphrase")

	if !mockTests {
		t.Skip("the sub-account name and API key are examples, and changing a real API key is not safe in a test")
	}
	result, err := e.ModifySubAccountAPIKey(t.Context(), &ModifySubAccountAPIKeyRequest{
		SubName:     "gocryptoTrader1",
		APIKey:      "65e7f22077172b0001f9ee41",
		Passphrase:  "mysecretPassphrase123",
		Permission:  "General,Spot",
		IPWhitelist: "192.0.2.1",
		Expire:      "90",
	})
	require.NoError(t, err, "ModifySubAccountAPIKey must not error")
	exp := &ModifySubAccountAPIKeyResponse{
		SubName:     "gocryptoTrader1",
		APIKey:      "65e7f22077172b0001f9ee41",
		Permission:  "General,Spot",
		IPWhitelist: "192.0.2.1",
	}
	assert.Equal(t, exp, result, "ModifySubAccountAPIKey should decode every field")
}

func TestDeleteSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	_, err := e.DeleteSubAccountAPIKey(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "DeleteSubAccountAPIKey must reject a nil request")
	_, err = e.DeleteSubAccountAPIKey(t.Context(), &DeleteSubAccountAPIKeyRequest{APIKey: "65e7f22077172b0001f9ee41", Passphrase: "the-passphrase"})
	require.ErrorIs(t, err, errInvalidSubAccountName, "DeleteSubAccountAPIKey must reject an empty sub-account name")
	_, err = e.DeleteSubAccountAPIKey(t.Context(), &DeleteSubAccountAPIKeyRequest{SubName: "gocryptoTrader1", Passphrase: "the-passphrase"})
	require.ErrorIs(t, err, errAPIKeyRequired, "DeleteSubAccountAPIKey must reject an empty API key")
	_, err = e.DeleteSubAccountAPIKey(t.Context(), &DeleteSubAccountAPIKeyRequest{SubName: "gocryptoTrader1", APIKey: "65e7f22077172b0001f9ee41"})
	require.ErrorIs(t, err, errInvalidPassPhraseInstance, "DeleteSubAccountAPIKey must reject an empty passphrase")

	if !mockTests {
		t.Skip("the sub-account name and API key are examples, and deleting a real API key is not safe in a test")
	}
	result, err := e.DeleteSubAccountAPIKey(t.Context(), &DeleteSubAccountAPIKeyRequest{
		SubName:    "gocryptoTrader1",
		APIKey:     "65e7f22077172b0001f9ee41",
		Passphrase: "the-passphrase",
	})
	require.NoError(t, err, "DeleteSubAccountAPIKey must not error")
	exp := &DeleteSubAccountAPIKeyResponse{SubName: "gocryptoTrader1", APIKey: "65e7f22077172b0001f9ee41"}
	assert.Equal(t, exp, result, "DeleteSubAccountAPIKey should decode every field")
}
