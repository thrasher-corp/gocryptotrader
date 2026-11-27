package kucoin

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Fixed query windows keep the mock fixtures' query strings stable
var (
	accountTestStartAt = time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	accountTestEndAt   = time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
)

func TestGetAccountSummaryInformation(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAccountSummaryInformation(t.Context())
	require.NoError(t, err, "GetAccountSummaryInformation must not error")
	if mockTests {
		exp := &AccountSummaryInformationResponse{
			Level:                 2,
			SubQuantity:           3,
			SpotSubQuantity:       3,
			MarginSubQuantity:     2,
			FuturesSubQuantity:    2,
			OptionSubQuantity:     1,
			MaxSubQuantity:        7,
			MaxDefaultSubQuantity: 5,
			MaxSpotSubQuantity:    2,
			MaxMarginSubQuantity:  1,
			MaxFuturesSubQuantity: 1,
			MaxOptionSubQuantity:  1,
		}
		assert.Equal(t, exp, result, "GetAccountSummaryInformation should decode every field")
		return
	}
	require.NotNil(t, result, "GetAccountSummaryInformation must return a result")
	assert.NotZero(t, result.MaxSubQuantity, "GetAccountSummaryInformation should return the sub-account limit")
}

func TestGetAPIKeyInfo(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAPIKeyInfo(t.Context())
	require.NoError(t, err, "GetAPIKeyInfo must not error")
	if mockTests {
		// A sub-account key: IsMaster is only true for master keys, which carry no ParentUID or SubName
		exp := &APIKeyInfoResponse{
			UID:                   100000002,
			ParentUID:             100000001,
			Region:                "PW",
			KYCStatus:             1,
			SubName:               "testapi6",
			Remark:                "sdk_test",
			APIKey:                "68d4a2645f0c2d0001a3841d",
			APIVersion:            3,
			Permission:            "General,Futures,Unified,Spot,Earn,InnerTransfer,Margin",
			IPWhitelist:           "192.0.2.1,198.51.100.2",
			CreatedAt:             types.Time(time.UnixMilli(1758765668000)),
			ExpiredAt:             types.Time(time.UnixMilli(1774317668000)),
			ThirdPartyApplication: "TradingView",
			SiteType:              "global",
		}
		assert.Equal(t, exp, result, "GetAPIKeyInfo should decode every field")
		return
	}
	require.NotNil(t, result, "GetAPIKeyInfo must return a result")
	assert.NotEmpty(t, result.APIKey, "GetAPIKeyInfo should return the API key")
}

func TestGetSpotAccountType(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSpotAccountType(t.Context())
	require.NoError(t, err, "GetSpotAccountType must not error")
	if mockTests {
		assert.True(t, result, "GetSpotAccountType should decode a high-frequency user")
	}
}

func TestGetAllAccounts(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name        string
		ccy         currency.Code
		accountType string
		exp         []AccountInfoResponse
	}{
		{
			name: "all",
			exp: []AccountInfoResponse{
				{ID: "548674591753", Currency: currency.USDT, AccountType: "trade", Balance: 26.66759503, Available: 20.66759503, Holds: 6},
				{ID: "63355cd156298d0001b66e61", Currency: currency.BTC, AccountType: "main", Balance: 0.01, Available: 0.01},
			},
		},
		{
			name:        "main USDT",
			ccy:         currency.USDT,
			accountType: "main",
			exp: []AccountInfoResponse{
				{ID: "63355cd156298d0001b66e61", Currency: currency.USDT, AccountType: "main", Balance: 0.01, Available: 0.005, Holds: 0.005},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetAllAccounts(t.Context(), tc.ccy, tc.accountType)
			require.NoError(t, err, "GetAllAccounts must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetAllAccounts should decode every field")
				return
			}
			assert.NotNil(t, result, "GetAllAccounts should return the accounts")
		})
	}
}

func TestGetAccountDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetAccountDetail(t.Context(), "")
	require.ErrorIs(t, err, errAccountIDMissing, "GetAccountDetail must reject an empty account ID")

	accountID := "62fcd1969474ea0001fd20e4"
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		accounts, err := e.GetAllAccounts(t.Context(), currency.EMPTYCODE, "")
		require.NoError(t, err, "GetAllAccounts must not error")
		if len(accounts) == 0 {
			t.Skip("the account has no accounts to look up")
		}
		accountID = accounts[0].ID
	}
	result, err := e.GetAccountDetail(t.Context(), accountID)
	require.NoError(t, err, "GetAccountDetail must not error")
	if mockTests {
		exp := &AccountDetailResponse{Currency: currency.USDT, Balance: 26.66759503, Available: 20.66759503, Holds: 6}
		assert.Equal(t, exp, result, "GetAccountDetail should decode every field")
		return
	}
	require.NotNil(t, result, "GetAccountDetail must return a result")
	assert.False(t, result.Currency.IsEmpty(), "GetAccountDetail should return the account currency")
}

func TestGetCrossMarginAccountsDetail(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossMarginAccountsDetail(t.Context(), currency.KCS, "MARGIN_V2")
	require.NoError(t, err, "GetCrossMarginAccountsDetail must not error")
	if mockTests {
		exp := &CrossMarginAccountsDetailResponse{
			TotalAssetOfQuoteCurrency:     4.0864837,
			TotalLiabilityOfQuoteCurrency: 0.2,
			DebtRatio:                     0.0489,
			Status:                        "EFFECTIVE",
			Accounts: []MarginAssetDetail{
				{
					Currency:           currency.USDT,
					Total:              38.68855864,
					Available:          20.01916691,
					Hold:               18.66939173,
					Liability:          1.6,
					MaxBorrowSize:      163,
					BorrowEnabled:      true,
					TransferInEnabled:  true,
					LiabilityPrincipal: 1.59,
					LiabilityInterest:  0.01,
				},
			},
		}
		assert.Equal(t, exp, result, "GetCrossMarginAccountsDetail should decode every field")
		return
	}
	require.NotNil(t, result, "GetCrossMarginAccountsDetail must return a result")
	assert.NotEmpty(t, result.Status, "GetCrossMarginAccountsDetail should return the account status")
}

func TestGetIsolatedMarginAccountDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetIsolatedMarginAccountDetail(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetIsolatedMarginAccountDetail must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *IsolatedMarginAccountDetailRequest
		exp  *IsolatedMarginAccountDetailResponse
	}{
		{
			name: "BTC-USDT in BTC",
			req:  &IsolatedMarginAccountDetailRequest{Symbol: "BTC-USDT", QuoteCurrency: currency.BTC},
			exp: &IsolatedMarginAccountDetailResponse{
				TotalAssetOfQuoteCurrency:     0.00006,
				TotalLiabilityOfQuoteCurrency: 0.000000008,
				Timestamp:                     types.Time(time.UnixMilli(1747303659773)),
				Assets: []IsolatedMarginAssetDetail{
					{
						Symbol:    "BTC-USDT",
						Status:    "EFFECTIVE",
						DebtRatio: 0.00013,
						BaseAsset: MarginAssetDetail{
							Currency:           currency.BTC,
							Total:              0.00001,
							Available:          0.000008,
							Hold:               0.000002,
							Liability:          0.000000004,
							MaxBorrowSize:      0.0001,
							BorrowEnabled:      true,
							TransferInEnabled:  true,
							LiabilityPrincipal: 0.000000003,
							LiabilityInterest:  0.000000001,
						},
						QuoteAsset: MarginAssetDetail{
							Currency:           currency.USDT,
							Total:              4.97047372,
							Available:          4.87047372,
							Hold:               0.1,
							Liability:          0.00038891,
							MaxBorrowSize:      44,
							BorrowEnabled:      true,
							TransferInEnabled:  true,
							LiabilityPrincipal: 0.00038888,
							LiabilityInterest:  0.00000003,
						},
					},
				},
			},
		},
		{
			name: "every symbol",
			req:  &IsolatedMarginAccountDetailRequest{QueryType: "ISOLATED"},
			exp: &IsolatedMarginAccountDetailResponse{
				TotalAssetOfQuoteCurrency:     10.5,
				TotalLiabilityOfQuoteCurrency: 0.5,
				Timestamp:                     types.Time(time.UnixMilli(1747303660000)),
				Assets: []IsolatedMarginAssetDetail{
					{
						Symbol:    "ETH-USDT",
						Status:    "BORROW",
						DebtRatio: 0.0476,
						BaseAsset: MarginAssetDetail{
							Currency:          currency.ETH,
							Total:             0.001,
							Available:         0.001,
							MaxBorrowSize:     0.01,
							BorrowEnabled:     true,
							TransferInEnabled: true,
						},
						QuoteAsset: MarginAssetDetail{
							Currency:           currency.USDT,
							Total:              7,
							Available:          7,
							Liability:          0.5,
							MaxBorrowSize:      20,
							LiabilityPrincipal: 0.49,
							LiabilityInterest:  0.01,
						},
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetIsolatedMarginAccountDetail(t.Context(), tc.req)
			require.NoError(t, err, "GetIsolatedMarginAccountDetail must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetIsolatedMarginAccountDetail should decode every field")
				return
			}
			require.NotNil(t, result, "GetIsolatedMarginAccountDetail must return a result")
			assert.NotZero(t, result.Timestamp, "GetIsolatedMarginAccountDetail should return a timestamp")
		})
	}
}

func TestGetFuturesAccountOverview(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		ccy currency.Code
		exp *FuturesAccountOverviewResponse
	}{
		{
			ccy: currency.XBT,
			exp: &FuturesAccountOverviewResponse{
				AccountEquity:     0.0025,
				UnrealisedPNL:     0.0001,
				MarginBalance:     0.0024,
				PositionMargin:    0.0004,
				OrderMargin:       0.0002,
				FrozenFunds:       0.0001,
				AvailableBalance:  0.0018,
				Currency:          currency.XBT,
				RiskRatio:         0.05,
				MaxWithdrawAmount: 0.0017,
				AvailableMargin:   0.0017,
			},
		},
		{
			ccy: currency.USDT,
			exp: &FuturesAccountOverviewResponse{
				AccountEquity:     198.733127406,
				UnrealisedPNL:     1.5,
				MarginBalance:     197.233127406,
				PositionMargin:    20.1,
				OrderMargin:       5.2,
				FrozenFunds:       0.5,
				AvailableBalance:  172.933127406,
				Currency:          currency.USDT,
				RiskRatio:         0.0123,
				MaxWithdrawAmount: 171.4,
				AvailableMargin:   171.4,
			},
		},
	} {
		t.Run(tc.ccy.String(), func(t *testing.T) {
			t.Parallel()
			result, err := e.GetFuturesAccountOverview(t.Context(), tc.ccy)
			require.NoError(t, err, "GetFuturesAccountOverview must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetFuturesAccountOverview should decode every field")
				return
			}
			require.NotNil(t, result, "GetFuturesAccountOverview must return a result")
			assert.Truef(t, result.Currency.Equal(tc.ccy), "GetFuturesAccountOverview should return the %s account", tc.ccy)
		})
	}
}

func TestGetAccountLedgers(t *testing.T) {
	t.Parallel()
	_, err := e.GetAccountLedgers(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAccountLedgers must reject a nil request")
	_, err = e.GetAccountLedgers(t.Context(), &AccountLedgersRequest{StartAt: accountTestEndAt, EndAt: accountTestStartAt})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetAccountLedgers must reject a start time after the end time")
	_, err = e.GetAccountLedgers(t.Context(), &AccountLedgersRequest{Currencies: []currency.Code{currency.BTC, currency.EMPTYCODE}})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetAccountLedgers must reject an empty currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *AccountLedgersRequest
		exp  *AccountLedgersResponse
	}{
		{
			name: "all",
			req:  &AccountLedgersRequest{},
			exp: &AccountLedgersResponse{
				CurrentPage: 1,
				PageSize:    50,
				TotalNumber: 2,
				TotalPage:   1,
				Items: []AccountLedger{
					{
						ID:           "265329987780896",
						Currency:     currency.USDT,
						Amount:       0.01,
						Fee:          0.0001,
						Balance:      0.5,
						AccountType:  "TRADE",
						BusinessType: "SUB_TRANSFER",
						Direction:    "out",
						CreatedAt:    types.Time(time.UnixMilli(1728658481484)),
					},
					{
						ID:           "265329987780897",
						Currency:     currency.BTC,
						Amount:       0.0001,
						Fee:          0.0000001,
						AccountType:  "TRADE",
						BusinessType: "TRADE_EXCHANGE",
						Direction:    "in",
						CreatedAt:    types.Time(time.UnixMilli(1728658481490)),
						Context:      `{"symbol":"BTC-USDT","orderId":"6705f6350dc7210007d6a36d","tradeId":"10046097631627265"}`,
					},
				},
			},
		},
		{
			name: "filtered",
			req: &AccountLedgersRequest{
				Currencies:   []currency.Code{currency.BTC, currency.USDT},
				Direction:    "in",
				BusinessType: "TRADE_EXCHANGE",
				StartAt:      accountTestStartAt,
				EndAt:        accountTestEndAt,
				CurrentPage:  1,
				PageSize:     10,
			},
			exp: &AccountLedgersResponse{
				CurrentPage: 1,
				PageSize:    10,
				TotalNumber: 1,
				TotalPage:   1,
				Items: []AccountLedger{
					{
						ID:           "265329987780898",
						Currency:     currency.BTC,
						Amount:       0.0002,
						Fee:          0.0000002,
						Balance:      0.0003,
						AccountType:  "TRADE",
						BusinessType: "TRADE_EXCHANGE",
						Direction:    "in",
						CreatedAt:    types.Time(time.UnixMilli(1790820000000)),
						Context:      `{"symbol":"BTC-USDT","orderId":"6705f6350dc7210007d6a36e","tradeId":"10046097631627266"}`,
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetAccountLedgers(t.Context(), tc.req)
			require.NoError(t, err, "GetAccountLedgers must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetAccountLedgers should decode every field")
				return
			}
			require.NotNil(t, result, "GetAccountLedgers must return a result")
			assert.NotZero(t, result.PageSize, "GetAccountLedgers should return the page size")
		})
	}
}

func TestGetAccountLedgersHFTrade(t *testing.T) {
	t.Parallel()
	_, err := e.GetAccountLedgersHFTrade(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAccountLedgersHFTrade must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAccountLedgersHFTrade(t.Context(), &HFAccountLedgersRequest{
		Currencies:   []currency.Code{currency.BTC},
		Direction:    "in",
		BusinessType: "TRADE_EXCHANGE",
		LastID:       254062248624418,
		Limit:        10,
		StartAt:      accountTestStartAt,
		EndAt:        accountTestEndAt,
	})
	require.NoError(t, err, "GetAccountLedgersHFTrade must not error")
	if mockTests {
		exp := []HFAccountLedgerResponse{
			{
				ID:           "254062248624417",
				Currency:     currency.BTC,
				Amount:       0.00002,
				Fee:          0.00000002,
				Tax:          0.000000001,
				Balance:      0.0001,
				AccountType:  "TRADE_HF",
				BusinessType: "TRADE_EXCHANGE",
				Direction:    "in",
				CreatedAt:    types.Time(time.UnixMilli(1790820000000)),
				Context:      `{"symbol":"BTC-USDT","orderId":"6705f6350dc7210007d6a36d","tradeId":"10046097631627265"}`,
			},
		}
		assert.Equal(t, exp, result, "GetAccountLedgersHFTrade should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAccountLedgersHFTrade should return the ledger")
}

func TestGetAccountLedgerHFMargin(t *testing.T) {
	t.Parallel()
	_, err := e.GetAccountLedgerHFMargin(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAccountLedgerHFMargin must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAccountLedgerHFMargin(t.Context(), &HFAccountLedgersRequest{Currencies: []currency.Code{currency.BTC}})
	require.NoError(t, err, "GetAccountLedgerHFMargin must not error")
	if mockTests {
		exp := []HFMarginAccountLedgerResponse{
			{
				ID:           1949641706720,
				Currency:     currency.BTC,
				Amount:       0.01,
				Fee:          0.00001,
				Balance:      0.01,
				AccountType:  "MARGIN_V2",
				BusinessType: "TRANSFER",
				Direction:    "in",
				CreatedAt:    types.Time(time.UnixMilli(1728664091208)),
				Context:      "{}",
				Tax:          0.000001,
			},
		}
		assert.Equal(t, exp, result, "GetAccountLedgerHFMargin should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAccountLedgerHFMargin should return the ledger")
}

func TestHFAccountLedgersParams(t *testing.T) {
	t.Parallel()
	_, err := hfAccountLedgersParams(nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "hfAccountLedgersParams must reject a nil request")
	_, err = hfAccountLedgersParams(&HFAccountLedgersRequest{StartAt: accountTestEndAt, EndAt: accountTestStartAt})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "hfAccountLedgersParams must reject a start time after the end time")
	_, err = hfAccountLedgersParams(&HFAccountLedgersRequest{Currencies: []currency.Code{currency.EMPTYCODE}})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "hfAccountLedgersParams must reject an empty currency")

	params, err := hfAccountLedgersParams(&HFAccountLedgersRequest{})
	require.NoError(t, err, "hfAccountLedgersParams must not error")
	assert.Empty(t, params, "hfAccountLedgersParams should leave unset filters out")

	params, err = hfAccountLedgersParams(&HFAccountLedgersRequest{
		Currencies:   []currency.Code{currency.BTC, currency.NewCode("usdt")},
		Direction:    "out",
		BusinessType: "TRANSFER",
		LastID:       42,
		Limit:        200,
		StartAt:      accountTestStartAt,
		EndAt:        accountTestEndAt,
	})
	require.NoError(t, err, "hfAccountLedgersParams must not error")
	exp := url.Values{"bizType": {"TRANSFER"}, "currency": {"BTC,USDT"}, "direction": {"out"}, "endAt": {"1790856000000"}, "lastId": {"42"}, "limit": {"200"}, "startAt": {"1790812800000"}}
	assert.Equal(t, exp, params, "hfAccountLedgersParams should set every filter")
}

func TestJoinLedgerCurrencies(t *testing.T) {
	t.Parallel()
	_, err := joinLedgerCurrencies([]currency.Code{currency.BTC, currency.EMPTYCODE})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "joinLedgerCurrencies must reject an empty currency")

	joined, err := joinLedgerCurrencies([]currency.Code{currency.NewCode("btc"), currency.USDT})
	require.NoError(t, err, "joinLedgerCurrencies must not error")
	assert.Equal(t, "BTC,USDT", joined, "joinLedgerCurrencies should join upper-cased currencies with commas")
}

func TestGetFuturesAccountLedgers(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesAccountLedgers(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFuturesAccountLedgers must reject a nil request")
	_, err = e.GetFuturesAccountLedgers(t.Context(), &FuturesAccountLedgersRequest{StartAt: accountTestEndAt, EndAt: accountTestStartAt})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFuturesAccountLedgers must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *FuturesAccountLedgersRequest
		exp  *FuturesAccountLedgersResponse
	}{
		{
			name: "forward",
			req:  &FuturesAccountLedgersRequest{Currency: currency.USDT, Forward: new(true), MaxCount: 100},
			exp: &FuturesAccountLedgersResponse{
				DataList: []FuturesAccountLedger{
					{
						Time:          types.Time(time.UnixMilli(1728666000000)),
						Type:          "RealisedPNL",
						Amount:        -0.00630042,
						AccountEquity: 14.02294896,
						Status:        "Pending",
						Remark:        "XBTUSDTM",
						Offset:        51360794,
						Currency:      currency.USDT,
					},
					{
						Time:          types.Time(time.UnixMilli(1728665747000)),
						Type:          "TransferIn",
						Amount:        0.01,
						Fee:           0.001,
						AccountEquity: 14.02924938,
						Status:        "Completed",
						Remark:        "Transferred from High-Frequency Trading Account",
						Offset:        51360793,
						Currency:      currency.USDT,
					},
				},
				HasMore: true,
			},
		},
		{
			name: "backward window",
			req: &FuturesAccountLedgersRequest{
				Currency: currency.XBT,
				Type:     "RealizedPNL",
				Offset:   51360793,
				Forward:  new(false),
				MaxCount: 10,
				StartAt:  accountTestStartAt,
				EndAt:    accountTestEndAt,
			},
			exp: &FuturesAccountLedgersResponse{
				DataList: []FuturesAccountLedger{
					{
						Time:          types.Time(time.UnixMilli(1790841600000)),
						Type:          "RealisedPNL",
						Amount:        0.00000315,
						Fee:           0.0000001,
						AccountEquity: 0.0025,
						Status:        "Completed",
						Remark:        "XBTUSDM",
						Offset:        51360792,
						Currency:      currency.XBT,
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetFuturesAccountLedgers(t.Context(), tc.req)
			require.NoError(t, err, "GetFuturesAccountLedgers must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetFuturesAccountLedgers should decode every field")
				return
			}
			assert.NotNil(t, result, "GetFuturesAccountLedgers should return the ledger")
		})
	}
}
