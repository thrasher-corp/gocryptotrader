package okx

import (
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetSubAccountList(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAccountList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSubAccountList must reject a nil request")
	_, err = e.GetSubAccountList(t.Context(), &SubAccountListRequest{StartTime: time.UnixMilli(1659334800000), EndTime: time.UnixMilli(1659334700000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSubAccountList must reject a start after the end")

	t.Run("request", func(t *testing.T) {
		t.Parallel()
		enabled, frozen := true, false
		for _, tc := range []struct {
			name  string
			req   *SubAccountListRequest
			query string
		}{
			{name: "no filters", req: &SubAccountListRequest{}},
			{name: "frozen sub-accounts", req: &SubAccountListRequest{Enabled: &frozen}, query: "enable=false"},
			{
				name: "every filter",
				req: &SubAccountListRequest{
					Enabled: &enabled, SubAccountName: "D456DDDL", StartTime: time.UnixMilli(1659334700000),
					EndTime: time.UnixMilli(1659334800000), Limit: 5,
				},
				query: "enable=true&subAcct=D456DDDL&after=1659334800000&before=1659334700000&limit=5",
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newFundingRequestExchange(t, http.MethodGet, "/users/subaccount/list", tc.query, "", []any{})
				_, err := ex.GetSubAccountList(t.Context(), tc.req)
				require.NoError(t, err, "GetSubAccountList must not error")
			})
		}
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	enabled := true
	result, err := e.GetSubAccountList(contextGenerate(), &SubAccountListRequest{
		Enabled:        &enabled,
		SubAccountName: "D456DDDL",
		StartTime:      time.UnixMilli(1659334700000),
		EndTime:        time.UnixMilli(1659334800000),
		Limit:          1,
	})
	require.NoError(t, err, "GetSubAccountList must not error")
	if mockTests {
		exp := []SubAccountResponse{{
			Type:                 "1",
			Enabled:              true,
			SubAccountName:       "D456DDDL",
			UID:                  "3400***********7413",
			Label:                "D456DDDLx",
			MobileNumber:         "139****8888",
			GoogleAuthenticator:  true,
			FrozenFunctions:      []string{"convert", "transfer"},
			CanTransferOut:       true,
			CreationTime:         types.Time(time.UnixMilli(1659334756000)),
			SubAccountLevel:      "1",
			FirstLevelSubAccount: "D456DDDL",
			DirectMarketAccess:   true,
		}}
		assert.Equal(t, exp, result, "GetSubAccountList should decode every field")
	} else {
		assert.NotNil(t, result, "GetSubAccountList should return sub-accounts")
	}
}

func TestCreateSubAccount(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *CreateSubAccountRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing sub-account name", req: &CreateSubAccountRequest{Type: "1"}, err: errSubAccountNameRequired},
		{name: "missing type", req: &CreateSubAccountRequest{SubAccountName: "subAccount002"}, err: errSubAccountTypeRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.CreateSubAccount(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "CreateSubAccount should reject the request")
		})
	}

	t.Run("request", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			req  *CreateSubAccountRequest
			body string
		}{
			{name: "required parameters only", req: &CreateSubAccountRequest{SubAccountName: "subAccount002", Type: "1"}, body: `{"subAcct":"subAccount002","type":"1"}`},
			{
				name: "every parameter",
				req:  &CreateSubAccountRequest{SubAccountName: "subAccount002", Type: "5", Label: "123456", Password: "Placeholder-1"},
				body: `{"subAcct":"subAccount002","type":"5","label":"123456","pwd":"Placeholder-1"}`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newFundingRequestExchange(t, http.MethodPost, "/users/subaccount/create-subaccount", "", tc.body, []map[string]string{{"subAcct": "subAccount002"}})
				_, err := ex.CreateSubAccount(t.Context(), tc.req)
				require.NoError(t, err, "CreateSubAccount must not error")
			})
		}
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CreateSubAccount(contextGenerate(), &CreateSubAccountRequest{SubAccountName: "subAccount002", Type: "1", Label: "123456", Password: "Placeholder-1"})
	require.NoError(t, err, "CreateSubAccount must not error")
	if mockTests {
		exp := &CreateSubAccountResponse{
			SubAccountName: "subAccount002",
			Label:          "123456",
			UID:            "698827017768230914",
			CreationTime:   types.Time(time.UnixMilli(1744875304520)),
		}
		assert.Equal(t, exp, result, "CreateSubAccount should decode every field")
	} else {
		assert.NotEmpty(t, result.UID, "CreateSubAccount should return the sub-account UID")
	}
}

func TestCreateSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *CreateSubAccountAPIKeyRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing sub-account name", req: &CreateSubAccountAPIKeyRequest{Label: "v5", Passphrase: "Placeholder-1"}, err: errSubAccountNameRequired},
		{name: "missing label", req: &CreateSubAccountAPIKeyRequest{SubAccountName: "test-1", Passphrase: "Placeholder-1"}, err: errAPIKeyLabelRequired},
		{name: "missing passphrase", req: &CreateSubAccountAPIKeyRequest{SubAccountName: "test-1", Label: "v5"}, err: errAPIKeyPassphraseRequired},
		{
			name: "more than 20 IP addresses",
			req:  &CreateSubAccountAPIKeyRequest{SubAccountName: "test-1", Label: "v5", Passphrase: "Placeholder-1", IPAddresses: slices.Repeat([]string{"1.1.1.1"}, 21)},
			err:  errExceedLimit,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.CreateSubAccountAPIKey(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "CreateSubAccountAPIKey should reject the request")
		})
	}

	t.Run("request", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			req  *CreateSubAccountAPIKeyRequest
			body string
		}{
			{
				name: "required parameters only",
				req:  &CreateSubAccountAPIKeyRequest{SubAccountName: "test-1", Label: "v5", Passphrase: "Placeholder-1"},
				body: `{"subAcct":"test-1","label":"v5","passphrase":"Placeholder-1"}`,
			},
			{
				name: "every parameter",
				req: &CreateSubAccountAPIKeyRequest{
					SubAccountName: "test-1", Label: "v5", Passphrase: "Placeholder-1",
					Permissions: []string{"read_only", "trade"}, IPAddresses: slices.Repeat([]string{"1.1.1.1"}, 20),
				},
				body: `{"subAcct":"test-1","label":"v5","passphrase":"Placeholder-1","perm":"read_only,trade","ip":"1.1.1.1` + strings.Repeat(",1.1.1.1", 19) + `"}`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newFundingRequestExchange(t, http.MethodPost, "/users/subaccount/apikey", "", tc.body, []map[string]string{{"subAcct": "test-1"}})
				_, err := ex.CreateSubAccountAPIKey(t.Context(), tc.req)
				require.NoError(t, err, "CreateSubAccountAPIKey must not error")
			})
		}
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CreateSubAccountAPIKey(contextGenerate(), &CreateSubAccountAPIKeyRequest{
		SubAccountName: "test-1",
		Label:          "v5",
		Passphrase:     "Placeholder-1",
		Permissions:    []string{"read_only", "trade"},
		IPAddresses:    []string{"1.1.1.1", "2.2.2.2"},
	})
	require.NoError(t, err, "CreateSubAccountAPIKey must not error")
	if mockTests {
		exp := &CreateSubAccountAPIKeyResponse{
			SubAccountName: "test-1",
			Label:          "v5",
			APIKey:         "******",
			SecretKey:      "******",
			Passphrase:     "******",
			Permissions:    "read_only,trade",
			IPAddresses:    "1.1.1.1,2.2.2.2",
			CreationTime:   types.Time(time.UnixMilli(1597026383085)),
		}
		assert.Equal(t, exp, result, "CreateSubAccountAPIKey should decode every field")
	} else {
		assert.NotEmpty(t, result.APIKey, "CreateSubAccountAPIKey should return the API key")
	}
}

func TestCreateSubAccountAPIKeyRequestMarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *CreateSubAccountAPIKeyRequest
		body string
	}{
		{name: "no lists", req: &CreateSubAccountAPIKeyRequest{SubAccountName: "test-1", Label: "v5", Passphrase: "Placeholder-1"}, body: `{"subAcct":"test-1","label":"v5","passphrase":"Placeholder-1"}`},
		{
			name: "one entry each",
			req:  &CreateSubAccountAPIKeyRequest{SubAccountName: "test-1", Label: "v5", Passphrase: "Placeholder-1", Permissions: []string{"trade"}, IPAddresses: []string{"1.1.1.1"}},
			body: `{"subAcct":"test-1","label":"v5","passphrase":"Placeholder-1","perm":"trade","ip":"1.1.1.1"}`,
		},
		{
			name: "several entries",
			req:  &CreateSubAccountAPIKeyRequest{SubAccountName: "test-1", Label: "v5", Passphrase: "Placeholder-1", Permissions: []string{"read_only", "trade"}, IPAddresses: []string{"1.1.1.1", "2.2.2.0/24"}},
			body: `{"subAcct":"test-1","label":"v5","passphrase":"Placeholder-1","perm":"read_only,trade","ip":"1.1.1.1,2.2.2.0/24"}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			body, err := json.Marshal(tc.req)
			require.NoError(t, err, "MarshalJSON must not error")
			assert.JSONEq(t, tc.body, string(body), "MarshalJSON should send comma-separated lists")
		})
	}
}

func TestGetSubAccountAPIKeys(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAccountAPIKeys(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSubAccountAPIKeys must reject a nil request")
	_, err = e.GetSubAccountAPIKeys(t.Context(), &SubAccountAPIKeysRequest{APIKey: "******"})
	require.ErrorIs(t, err, errSubAccountNameRequired, "GetSubAccountAPIKeys must reject a missing sub-account name")

	t.Run("request", func(t *testing.T) {
		t.Parallel()
		ex := newFundingRequestExchange(t, http.MethodGet, "/users/subaccount/apikey", "subAcct=test-1&apiKey=sub-account-api-key", "", []any{})
		_, err := ex.GetSubAccountAPIKeys(t.Context(), &SubAccountAPIKeysRequest{SubAccountName: "test-1", APIKey: "sub-account-api-key"})
		require.NoError(t, err, "GetSubAccountAPIKeys must not error")
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSubAccountAPIKeys(contextGenerate(), &SubAccountAPIKeysRequest{SubAccountName: "test-1"})
	require.NoError(t, err, "GetSubAccountAPIKeys must not error")
	if mockTests {
		exp := []SubAccountAPIKeyResponse{
			{Label: "v5", APIKey: "******", Permissions: "read_only,trade", IPAddresses: "1.1.1.1,2.2.2.2", CreationTime: types.Time(time.UnixMilli(1597026383085))},
			{Label: "v5.1", APIKey: "******", Permissions: "read_only", IPAddresses: "1.1.1.1,2.2.2.2", CreationTime: types.Time(time.UnixMilli(1597026383085))},
		}
		assert.Equal(t, exp, result, "GetSubAccountAPIKeys should decode every field")
	} else {
		assert.NotNil(t, result, "GetSubAccountAPIKeys should return API keys")
	}
}

func TestResetSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *ResetSubAccountAPIKeyRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing sub-account name", req: &ResetSubAccountAPIKeyRequest{APIKey: apiCredentials.Key, Permissions: []string{"trade"}}, err: errSubAccountNameRequired},
		{name: "missing API key", req: &ResetSubAccountAPIKeyRequest{SubAccountName: "sam", Permissions: []string{"trade"}}, err: errInvalidAPIKey},
		{
			name: "IP addresses while unlinking every address",
			req:  &ResetSubAccountAPIKeyRequest{SubAccountName: "sam", APIKey: "sub-account-api-key", IPAddresses: []string{"1.2.3.4"}, UnlinkIPAddresses: true},
			err:  errIPAddressesWithUnlink,
		},
		{
			name: "more than 20 IP addresses",
			req:  &ResetSubAccountAPIKeyRequest{SubAccountName: "sam", APIKey: "sub-account-api-key", IPAddresses: slices.Repeat([]string{"1.2.3.4"}, 21)},
			err:  errExceedLimit,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.ResetSubAccountAPIKey(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "ResetSubAccountAPIKey should reject the request")
		})
	}

	t.Run("request", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			req  *ResetSubAccountAPIKeyRequest
			body string
		}{
			{
				name: "required parameters only",
				req:  &ResetSubAccountAPIKeyRequest{SubAccountName: "sam", APIKey: "sub-account-api-key"},
				body: `{"subAcct":"sam","apiKey":"sub-account-api-key"}`,
			},
			{
				name: "every parameter",
				req: &ResetSubAccountAPIKeyRequest{
					SubAccountName: "sam", APIKey: "sub-account-api-key", Label: "v5",
					Permissions: []string{"read_only", "trade"}, IPAddresses: []string{"1.1.1.1", "2.2.2.2"},
				},
				body: `{"subAcct":"sam","apiKey":"sub-account-api-key","label":"v5","perm":"read_only,trade","ip":"1.1.1.1,2.2.2.2"}`,
			},
			{
				name: "unlink every IP address",
				req:  &ResetSubAccountAPIKeyRequest{SubAccountName: "sam", APIKey: "sub-account-api-key", UnlinkIPAddresses: true},
				body: `{"subAcct":"sam","apiKey":"sub-account-api-key","ip":""}`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newFundingRequestExchange(t, http.MethodPost, "/users/subaccount/modify-apikey", "", tc.body, []map[string]string{{"subAcct": "sam"}})
				_, err := ex.ResetSubAccountAPIKey(t.Context(), tc.req)
				require.NoError(t, err, "ResetSubAccountAPIKey must not error")
			})
		}
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.ResetSubAccountAPIKey(contextGenerate(), &ResetSubAccountAPIKeyRequest{
		SubAccountName: "yongxu",
		APIKey:         "******",
		Label:          "v5",
		Permissions:    []string{"read_only", "trade"},
		IPAddresses:    []string{"1.1.1.1"},
	})
	require.NoError(t, err, "ResetSubAccountAPIKey must not error")
	if mockTests {
		exp := &ResetSubAccountAPIKeyResponse{
			SubAccountName: "yongxu",
			APIKey:         "******",
			Label:          "v5",
			Permissions:    "read_only,trade",
			IPAddresses:    "1.1.1.1",
			CreationTime:   types.Time(time.UnixMilli(1597026383085)),
		}
		assert.Equal(t, exp, result, "ResetSubAccountAPIKey should decode every field")
	} else {
		assert.NotEmpty(t, result.APIKey, "ResetSubAccountAPIKey should return the API key")
	}
}

func TestResetSubAccountAPIKeyRequestMarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *ResetSubAccountAPIKeyRequest
		body string
	}{
		{name: "keep the IP addresses", req: &ResetSubAccountAPIKeyRequest{SubAccountName: "sam", APIKey: "key"}, body: `{"subAcct":"sam","apiKey":"key"}`},
		{name: "unlink every IP address", req: &ResetSubAccountAPIKeyRequest{SubAccountName: "sam", APIKey: "key", UnlinkIPAddresses: true}, body: `{"subAcct":"sam","apiKey":"key","ip":""}`},
		{name: "one IP address", req: &ResetSubAccountAPIKeyRequest{SubAccountName: "sam", APIKey: "key", IPAddresses: []string{"1.1.1.1"}}, body: `{"subAcct":"sam","apiKey":"key","ip":"1.1.1.1"}`},
		{
			name: "several permissions and IP addresses",
			req:  &ResetSubAccountAPIKeyRequest{SubAccountName: "sam", APIKey: "key", Label: "v5", Permissions: []string{"read_only", "trade"}, IPAddresses: []string{"1.1.1.1", "2.2.2.2"}},
			body: `{"subAcct":"sam","apiKey":"key","label":"v5","perm":"read_only,trade","ip":"1.1.1.1,2.2.2.2"}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			body, err := json.Marshal(tc.req)
			require.NoError(t, err, "MarshalJSON must not error")
			assert.JSONEq(t, tc.body, string(body), "MarshalJSON should send comma-separated lists")
		})
	}
}

func TestDeleteSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *DeleteSubAccountAPIKeyRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing sub-account name", req: &DeleteSubAccountAPIKeyRequest{APIKey: "******"}, err: errSubAccountNameRequired},
		{name: "missing API key", req: &DeleteSubAccountAPIKeyRequest{SubAccountName: "test00001"}, err: errInvalidAPIKey},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.DeleteSubAccountAPIKey(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "DeleteSubAccountAPIKey should reject the request")
		})
	}

	t.Run("request", func(t *testing.T) {
		t.Parallel()
		ex := newFundingRequestExchange(t, http.MethodPost, "/users/subaccount/delete-apikey", "", `{"subAcct":"test00001","apiKey":"sub-account-api-key"}`, []map[string]string{{"subAcct": "test00001"}})
		_, err := ex.DeleteSubAccountAPIKey(t.Context(), &DeleteSubAccountAPIKeyRequest{SubAccountName: "test00001", APIKey: "sub-account-api-key"})
		require.NoError(t, err, "DeleteSubAccountAPIKey must not error")
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.DeleteSubAccountAPIKey(contextGenerate(), &DeleteSubAccountAPIKeyRequest{SubAccountName: "test00001", APIKey: "******"})
	require.NoError(t, err, "DeleteSubAccountAPIKey must not error")
	if mockTests {
		assert.Equal(t, &SubAccountNameResponse{SubAccountName: "test00001"}, result, "DeleteSubAccountAPIKey should decode every field")
	} else {
		assert.NotEmpty(t, result.SubAccountName, "DeleteSubAccountAPIKey should return the sub-account name")
	}
}

func TestGetSubAccountTradingBalance(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAccountTradingBalance(t.Context(), "")
	require.ErrorIs(t, err, errSubAccountNameRequired, "GetSubAccountTradingBalance must reject a missing sub-account name")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSubAccountTradingBalance(contextGenerate(), "test1")
	require.NoError(t, err, "GetSubAccountTradingBalance must not error")
	if mockTests {
		exp := []SubAccountTradingBalanceResponse{{
			UpdateTime:                   types.Time(time.UnixMilli(1739332269934)),
			TotalEquity:                  63101.46752000000001,
			IsolatedEquity:               3150,
			AdjustedEquity:               101.46752000000001,
			AvailableEquity:              624719833286,
			OrderFrozen:                  3150,
			InitialMarginRequirement:     1500,
			MaintenanceMarginRequirement: 300,
			BorrowFrozen:                 120,
			MarginRatio:                  12.5,
			NotionalUSD:                  63000,
			NotionalUSDForBorrow:         1000,
			NotionalUSDForSwap:           4000,
			NotionalUSDForFutures:        2000,
			NotionalUSDForOption:         3000,
			UnrealisedPNL:                252,
			Delta:                        1500,
			DeltaLeverage:                0.5,
			DeltaNeutralStatus:           "1",
			Details: []SubAccountBalanceDetail{
				{
					Currency:              currency.USDT,
					Equity:                101.5,
					CashBalance:           101.5,
					UpdateTime:            types.Time(time.UnixMilli(1663854334734)),
					AvailableEquity:       101.5,
					DiscountEquity:        101.46752000000001,
					AvailableBalance:      101.5,
					MaximumLoan:           1015.0000000000001,
					EquityUSD:             101.46752000000001,
					ForcedRepaymentType:   "0",
					CollateralRestriction: "0",
					AutoLendStatus:        "off",
				},
				{
					Currency:                       currency.BTC,
					Equity:                         1.05,
					CashBalance:                    1,
					UpdateTime:                     types.Time(time.UnixMilli(1663854334735)),
					IsolatedEquity:                 0.05,
					AvailableEquity:                0.95,
					DiscountEquity:                 61000,
					FixedBalance:                   0.01,
					AvailableBalance:               0.9,
					FrozenBalance:                  0.1,
					OrderFrozen:                    0.05,
					Liabilities:                    0.02,
					UnrealisedPNL:                  0.004,
					UnrealisedPNLLiabilities:       0.005,
					CrossLiabilities:               0.01,
					RewardBalance:                  0.001,
					IsolatedLiabilities:            0.002,
					MarginRatio:                    12.5,
					InitialMarginRequirement:       1500,
					MaintenanceMarginRequirement:   300,
					Interest:                       0.0001,
					TWAP:                           1,
					ForcedRepaymentType:            "1",
					MaximumLoan:                    5,
					EquityUSD:                      63000,
					BorrowFrozen:                   120,
					NotionalLeverage:               1.5,
					StrategyEquity:                 0.15,
					IsolatedUnrealisedPNL:          0.003,
					SpotInUseAmount:                0.3,
					ClientSpotInUseAmount:          0.2,
					MaximumSpotInUse:               0.5,
					SpotIsolatedBalance:            0.4,
					SmartSyncEquity:                0.1,
					SpotCopyTradingEquity:          0.2,
					SpotBalance:                    1,
					OpenAveragePrice:               59000,
					AccumulatedAveragePrice:        60000,
					SpotUnrealisedPNL:              1000,
					SpotUnrealisedPNLRatio:         0.0169,
					TotalPNL:                       1200,
					TotalPNLRatio:                  0.02,
					CollateralRestriction:          "1",
					CollateralBorrowAutoConversion: 2,
					CollateralRestrict:             true,
					CollateralEnabled:              true,
					AutoLendStatus:                 "active",
					AutoLendMatchedAmount:          0.05,
				},
			},
		}}
		assert.Equal(t, exp, result, "GetSubAccountTradingBalance should decode every field")
	} else {
		assert.NotNil(t, result, "GetSubAccountTradingBalance should return the balance")
	}
}

func TestGetSubAccountFundingBalance(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *SubAccountFundingBalanceRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing sub-account name", req: &SubAccountFundingBalanceRequest{Currencies: currency.Currencies{currency.BTC}}, err: errSubAccountNameRequired},
		{name: "empty currency", req: &SubAccountFundingBalanceRequest{SubAccountName: "test1", Currencies: currency.Currencies{currency.EMPTYCODE}}, err: currency.ErrCurrencyCodeEmpty},
		{name: "more than 20 currencies", req: &SubAccountFundingBalanceRequest{SubAccountName: "test1", Currencies: slices.Repeat(currency.Currencies{currency.BTC}, 21)}, err: errExceedLimit},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetSubAccountFundingBalance(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "GetSubAccountFundingBalance should reject the request")
		})
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSubAccountFundingBalance(contextGenerate(), &SubAccountFundingBalanceRequest{SubAccountName: "test1", Currencies: currency.Currencies{currency.BTC, currency.ETH}})
	require.NoError(t, err, "GetSubAccountFundingBalance must not error")
	if mockTests {
		exp := []FundingBalanceResponse{
			{Currency: currency.BTC, Balance: 0.6, FrozenBalance: 0.1, AvailableBalance: 0.5},
			{Currency: currency.ETH, Balance: 37.11827078, AvailableBalance: 37.11827078},
		}
		assert.Equal(t, exp, result, "GetSubAccountFundingBalance should decode every field")
	} else {
		assert.NotNil(t, result, "GetSubAccountFundingBalance should return balances")
	}
}

func TestGetSubAccountMaximumWithdrawal(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *SubAccountMaximumWithdrawalRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing sub-account name", req: &SubAccountMaximumWithdrawalRequest{Currencies: currency.Currencies{currency.BTC}}, err: errSubAccountNameRequired},
		{name: "empty currency", req: &SubAccountMaximumWithdrawalRequest{SubAccountName: "test1", Currencies: currency.Currencies{currency.EMPTYCODE}}, err: currency.ErrCurrencyCodeEmpty},
		{name: "more than 20 currencies", req: &SubAccountMaximumWithdrawalRequest{SubAccountName: "test1", Currencies: slices.Repeat(currency.Currencies{currency.BTC}, 21)}, err: errExceedLimit},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetSubAccountMaximumWithdrawal(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "GetSubAccountMaximumWithdrawal should reject the request")
		})
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSubAccountMaximumWithdrawal(contextGenerate(), &SubAccountMaximumWithdrawalRequest{
		SubAccountName: "test1",
		Currencies:     currency.Currencies{currency.BTC, currency.ETH, currency.USDT},
	})
	require.NoError(t, err, "GetSubAccountMaximumWithdrawal must not error")
	if mockTests {
		exp := []SubAccountMaximumWithdrawalResponse{
			{Currency: currency.BTC, MaximumWithdrawal: 3, MaximumWithdrawalIncludingBorrowed: 3.5, SpotOffsetMaximumWithdrawal: 3, SpotOffsetMaximumWithdrawalIncludingBorrowed: 3.5},
			{Currency: currency.ETH, MaximumWithdrawal: 15, SpotOffsetMaximumWithdrawal: 15},
			{Currency: currency.USDT, MaximumWithdrawal: 10600, SpotOffsetMaximumWithdrawal: 10600},
		}
		assert.Equal(t, exp, result, "GetSubAccountMaximumWithdrawal should decode every field")
	} else {
		assert.NotNil(t, result, "GetSubAccountMaximumWithdrawal should return withdrawal limits")
	}
}

func TestGetSubAccountTransferHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAccountTransferHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSubAccountTransferHistory must reject a nil request")
	_, err = e.GetSubAccountTransferHistory(t.Context(), &SubAccountTransferHistoryRequest{StartTime: time.UnixMilli(1712561000000), EndTime: time.UnixMilli(1712560000000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSubAccountTransferHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSubAccountTransferHistory(contextGenerate(), &SubAccountTransferHistoryRequest{
		Currency:       currency.USDT,
		TransferType:   "0",
		SubAccountName: "hahatest1",
		StartTime:      time.UnixMilli(1712560000000),
		EndTime:        time.UnixMilli(1712561000000),
		Limit:          1,
	})
	require.NoError(t, err, "GetSubAccountTransferHistory must not error")
	if mockTests {
		exp := []SubAccountTransferResponse{{
			BillID:         "89887685",
			Currency:       currency.USDT,
			Amount:         1.1,
			BillType:       "0",
			SubAccountName: "hahatest1",
			Timestamp:      types.Time(time.UnixMilli(1712560959000)),
		}}
		assert.Equal(t, exp, result, "GetSubAccountTransferHistory should decode every field")
	} else {
		assert.NotNil(t, result, "GetSubAccountTransferHistory should return transfers")
	}
}

func TestGetManagedSubAccountTransferHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetManagedSubAccountTransferHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetManagedSubAccountTransferHistory must reject a nil request")
	_, err = e.GetManagedSubAccountTransferHistory(t.Context(), &ManagedSubAccountTransferHistoryRequest{StartTime: time.UnixMilli(1597026400000), EndTime: time.UnixMilli(1597026300000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetManagedSubAccountTransferHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetManagedSubAccountTransferHistory(contextGenerate(), &ManagedSubAccountTransferHistoryRequest{
		Currency:       currency.BTC,
		TransferType:   "1",
		SubAccountName: "test-1",
		SubAccountUID:  "446556018520336384",
		StartTime:      time.UnixMilli(1597026300000),
		EndTime:        time.UnixMilli(1597026400000),
		Limit:          1,
	})
	require.NoError(t, err, "GetManagedSubAccountTransferHistory must not error")
	if mockTests {
		exp := []ManagedSubAccountTransferResponse{{
			BillID:         "12344",
			Currency:       currency.BTC,
			Amount:         2,
			BillType:       "1",
			SubAccountName: "test-1",
			SubAccountUID:  "446556018520336384",
			Timestamp:      types.Time(time.UnixMilli(1597026383085)),
		}}
		assert.Equal(t, exp, result, "GetManagedSubAccountTransferHistory should decode every field")
	} else {
		assert.NotNil(t, result, "GetManagedSubAccountTransferHistory should return transfers")
	}
}

func TestTransferBetweenSubAccounts(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *TransferBetweenSubAccountsRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "empty request", req: &TransferBetweenSubAccountsRequest{}, err: currency.ErrCurrencyCodeEmpty},
		{name: "missing currency", req: &TransferBetweenSubAccountsRequest{LoanTransfer: true}, err: currency.ErrCurrencyCodeEmpty},
		{name: "missing amount", req: &TransferBetweenSubAccountsRequest{Currency: currency.BTC, LoanTransfer: true}, err: limits.ErrAmountBelowMin},
		{name: "missing remitting account", req: &TransferBetweenSubAccountsRequest{Currency: currency.BTC, Amount: 1234}, err: errRemittingAccountTypeRequired},
		{name: "missing beneficiary account", req: &TransferBetweenSubAccountsRequest{Currency: currency.BTC, Amount: 1234, RemittingAccountType: "6"}, err: errBeneficiaryAccountTypeRequired},
		{name: "missing from sub-account", req: &TransferBetweenSubAccountsRequest{Currency: currency.BTC, Amount: 1234, RemittingAccountType: "6", BeneficiaryAccountType: "6"}, err: errFromSubAccountNameRequired},
		{
			name: "missing to sub-account",
			req:  &TransferBetweenSubAccountsRequest{Currency: currency.BTC, Amount: 1234, RemittingAccountType: "6", BeneficiaryAccountType: "6", FromSubAccount: "sami"},
			err:  errToSubAccountNameRequired,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.TransferBetweenSubAccounts(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "TransferBetweenSubAccounts should reject the request")
		})
	}

	t.Run("request", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			req  *TransferBetweenSubAccountsRequest
			body string
		}{
			{
				name: "required parameters only",
				req:  &TransferBetweenSubAccountsRequest{Currency: currency.USDT, Amount: 1.5, RemittingAccountType: "6", BeneficiaryAccountType: "18", FromSubAccount: "test-1", ToSubAccount: "test-2"},
				body: `{"ccy":"USDT","amt":"1.5","from":"6","to":"18","fromSubAccount":"test-1","toSubAccount":"test-2"}`,
			},
			{
				name: "every parameter",
				req: &TransferBetweenSubAccountsRequest{
					Currency: currency.USDT, Amount: 1.5, RemittingAccountType: "6", BeneficiaryAccountType: "18",
					FromSubAccount: "test-1", ToSubAccount: "test-2", LoanTransfer: true, OmitPositionRisk: true,
				},
				body: `{"ccy":"USDT","amt":"1.5","from":"6","to":"18","fromSubAccount":"test-1","toSubAccount":"test-2","loanTrans":true,"omitPosRisk":"true"}`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newFundingRequestExchange(t, http.MethodPost, "/asset/subaccount/transfer", "", tc.body, []map[string]string{{"transId": "12345"}})
				_, err := ex.TransferBetweenSubAccounts(t.Context(), tc.req)
				require.NoError(t, err, "TransferBetweenSubAccounts must not error")
			})
		}
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.TransferBetweenSubAccounts(contextGenerate(), &TransferBetweenSubAccountsRequest{
		Currency:               currency.USDT,
		Amount:                 1.5,
		RemittingAccountType:   "6",
		BeneficiaryAccountType: "18",
		FromSubAccount:         "test-1",
		ToSubAccount:           "test-2",
		LoanTransfer:           true,
		OmitPositionRisk:       true,
	})
	require.NoError(t, err, "TransferBetweenSubAccounts must not error")
	if mockTests {
		assert.Equal(t, &TransferIDResponse{TransferID: "12345"}, result, "TransferBetweenSubAccounts should decode every field")
	} else {
		assert.NotEmpty(t, result.TransferID, "TransferBetweenSubAccounts should return a transfer ID")
	}
}

func TestSetPermissionOfTransferOut(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *PermissionOfTransferOutRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "empty request", req: &PermissionOfTransferOutRequest{}, err: errSubAccountNameRequired},
		{name: "no sub-account names", req: &PermissionOfTransferOutRequest{CanTransferOut: true}, err: errSubAccountNameRequired},
		{name: "empty sub-account name", req: &PermissionOfTransferOutRequest{SubAccountNames: []string{"Test001", ""}}, err: errSubAccountNameRequired},
		{name: "more than 20 sub-accounts", req: &PermissionOfTransferOutRequest{SubAccountNames: slices.Repeat([]string{"Test001"}, 21)}, err: errExceedLimit},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.SetPermissionOfTransferOut(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "SetPermissionOfTransferOut should reject the request")
		})
	}

	t.Run("request", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			req  *PermissionOfTransferOutRequest
			body string
		}{
			{name: "deny one sub-account", req: &PermissionOfTransferOutRequest{SubAccountNames: []string{"Test001"}}, body: `{"subAcct":"Test001","canTransOut":false}`},
			{
				name: "allow 20 sub-accounts",
				req:  &PermissionOfTransferOutRequest{SubAccountNames: slices.Repeat([]string{"Test001"}, 20), CanTransferOut: true},
				body: `{"subAcct":"Test001` + strings.Repeat(",Test001", 19) + `","canTransOut":true}`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newFundingRequestExchange(t, http.MethodPost, "/users/subaccount/set-transfer-out", "", tc.body, []map[string]any{{"subAcct": "Test001", "canTransOut": tc.req.CanTransferOut}})
				_, err := ex.SetPermissionOfTransferOut(t.Context(), tc.req)
				require.NoError(t, err, "SetPermissionOfTransferOut must not error")
			})
		}
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SetPermissionOfTransferOut(contextGenerate(), &PermissionOfTransferOutRequest{SubAccountNames: []string{"Test001", "Test002"}, CanTransferOut: true})
	require.NoError(t, err, "SetPermissionOfTransferOut must not error")
	if mockTests {
		exp := []PermissionOfTransferOutResponse{
			{SubAccountName: "Test001", CanTransferOut: true},
			{SubAccountName: "Test002", CanTransferOut: true},
		}
		assert.Equal(t, exp, result, "SetPermissionOfTransferOut should decode every field")
	} else {
		assert.NotEmpty(t, result, "SetPermissionOfTransferOut should return the permissions")
	}
}

func TestPermissionOfTransferOutRequestMarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *PermissionOfTransferOutRequest
		body string
	}{
		{name: "one sub-account", req: &PermissionOfTransferOutRequest{SubAccountNames: []string{"Test001"}, CanTransferOut: true}, body: `{"subAcct":"Test001","canTransOut":true}`},
		{name: "several sub-accounts", req: &PermissionOfTransferOutRequest{SubAccountNames: []string{"Test001", "Test002"}}, body: `{"subAcct":"Test001,Test002","canTransOut":false}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			body, err := json.Marshal(tc.req)
			require.NoError(t, err, "MarshalJSON must not error")
			assert.JSONEq(t, tc.body, string(body), "MarshalJSON should send the sub-account names comma-separated")
		})
	}
}

func TestGetCustodyTradingSubAccountList(t *testing.T) {
	t.Parallel()
	_, err := e.GetCustodyTradingSubAccountList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetCustodyTradingSubAccountList must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCustodyTradingSubAccountList(contextGenerate(), &CustodyTradingSubAccountListRequest{SubAccountName: "test-1"})
	require.NoError(t, err, "GetCustodyTradingSubAccountList must not error")
	if mockTests {
		assert.Equal(t, []SubAccountNameResponse{{SubAccountName: "test-1"}}, result, "GetCustodyTradingSubAccountList should decode every field")
	} else {
		assert.NotNil(t, result, "GetCustodyTradingSubAccountList should return sub-accounts")
	}
}
