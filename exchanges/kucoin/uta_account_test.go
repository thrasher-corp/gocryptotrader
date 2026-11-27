package kucoin

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetUTAAccountOverview(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUTAAccountOverview(t.Context())
	require.NoError(t, err, "GetUTAAccountOverview must not error")
	if mockTests {
		exp := &UTAAccountOverviewResponse{
			AccountType:       "UNIFIED",
			RiskRatio:         0.0085517383,
			Equity:            49.935894267,
			Liability:         1.25,
			AvailableMargin:   46.0677901003,
			AdjustedEquity:    49.935894267,
			InitialMargin:     3.8681041666,
			MaintenanceMargin: 0.371338,
		}
		assert.Equal(t, exp, result, "GetUTAAccountOverview should decode every field")
		return
	}
	assert.NotEmpty(t, result.AccountType, "GetUTAAccountOverview should return an account type")
}

func TestGetUTAAccountCurrencyAssets(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUTAAccountCurrencyAssets(t.Context())
	require.NoError(t, err, "GetUTAAccountCurrencyAssets must not error")
	if mockTests {
		exp := &UTAAccountCurrencyAssetsResponse{
			AccountType: "UNIFIED",
			Timestamp:   types.Time(time.UnixMilli(1776089058432)),
			Accounts: []UTAAccountAssets{
				{
					Currencies: []UTACurrencyAsset{
						{
							Currency:         currency.USDT,
							Equity:           3054.8568208133,
							Hold:             2,
							Balance:          3052.9434208133,
							Available:        3052.8568208133,
							CollateralStatus: "1",
						},
						{
							Currency:         currency.ETH,
							Equity:           -0.02,
							Balance:          -0.02,
							Available:        -0.02,
							Liability:        0.02,
							PotentialBorrow:  0.005,
							CollateralStatus: "3",
						},
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetUTAAccountCurrencyAssets should decode every field")
		return
	}
	assert.NotEmpty(t, result.AccountType, "GetUTAAccountCurrencyAssets should return an account type")
}

func TestGetUTAClassicAccountCurrencyAssets(t *testing.T) {
	t.Parallel()
	_, err := e.GetUTAClassicAccountCurrencyAssets(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUTAClassicAccountCurrencyAssets must reject a nil request")
	_, err = e.GetUTAClassicAccountCurrencyAssets(t.Context(), &UTAClassicAccountCurrencyAssetsRequest{})
	require.ErrorIs(t, err, errAccountTypeMissing, "GetUTAClassicAccountCurrencyAssets must reject an empty account type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		req *UTAClassicAccountCurrencyAssetsRequest
		exp *UTAClassicAccountCurrencyAssetsResponse
	}{
		{
			req: &UTAClassicAccountCurrencyAssetsRequest{AccountType: "FUTURES"},
			exp: &UTAClassicAccountCurrencyAssetsResponse{
				AccountType: "FUTURES",
				Timestamp:   types.Time(time.UnixMilli(1785234449765)),
				Accounts: []UTAClassicAccountAssets{
					{
						RiskRatio: 0.0123,
						Currencies: []UTAClassicCurrencyAsset{
							{
								Currency:                 currency.USDT,
								Available:                947.4,
								Balance:                  1000,
								Equity:                   1012.3,
								CrossOrderMargin:         10.5,
								CrossPositionMargin:      20.25,
								CrossUnrealisedPNL:       8.3,
								IsolatedFundingFeeMargin: 0.15,
								IsolatedOrderMargin:      5.1,
								IsolatedPositionMargin:   8.5,
								IsolatedUnrealisedPNL:    4,
								TotalCrossMargin:         1008.3,
							},
						},
					},
				},
			},
		},
		{
			req: &UTAClassicAccountCurrencyAssetsRequest{AccountType: "ISOLATED", AccountSubtype: "BTC-USDT", Currency: currency.BTC},
			exp: &UTAClassicAccountCurrencyAssetsResponse{
				AccountType: "ISOLATED",
				Timestamp:   types.Time(time.UnixMilli(1785234449765)),
				Accounts: []UTAClassicAccountAssets{
					{
						AccountSubtype: "BTC-USDT",
						RiskRatio:      0.1234,
						Currencies: []UTAClassicCurrencyAsset{
							{
								Currency:  currency.BTC,
								Locked:    0.001,
								Available: 0.009,
								Balance:   0.01,
								Equity:    0.01,
								Liability: 0.002,
							},
						},
					},
				},
			},
		},
	} {
		t.Run(tc.req.AccountType, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetUTAClassicAccountCurrencyAssets(t.Context(), tc.req)
			require.NoError(t, err, "GetUTAClassicAccountCurrencyAssets must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetUTAClassicAccountCurrencyAssets should decode every field")
				return
			}
			assert.Equal(t, tc.req.AccountType, result.AccountType, "GetUTAClassicAccountCurrencyAssets should return the requested account type")
		})
	}
}

func TestUTAClassicAccountAssetsDecoding(t *testing.T) {
	t.Parallel()
	for _, key := range []string{"accountSubtype", "accountSubType"} {
		var got UTAClassicAccountAssets
		require.NoErrorf(t, json.Unmarshal([]byte(`{"`+key+`":"AAVE-USDT","riskRatio":"0.5"}`), &got), "Unmarshal must not error for %s", key)
		assert.Equalf(t, UTAClassicAccountAssets{AccountSubtype: "AAVE-USDT", RiskRatio: 0.5}, got, "Unmarshal should decode %s into the account subtype", key)
	}
}

func TestGetUTAAPIKeyInfo(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUTAAPIKeyInfo(t.Context())
	require.NoError(t, err, "GetUTAAPIKeyInfo must not error")
	if mockTests {
		exp := &UTAAPIKeyInfoResponse{
			UID:           100000002,
			ParentUID:     100000001,
			Region:        "ID",
			KYCStatus:     1,
			SubName:       "testapi2",
			Remark:        "docs only",
			APIKey:        "670621e3a25958000159c82f",
			APIVersion:    3,
			Permission:    "General,Futures,Unified,Spot,Earn,InnerTransfer,Margin",
			IPWhitelist:   "192.0.2.1,192.0.2.2",
			IsMaster:      true,
			CreatedAt:     types.Time(time.UnixMilli(1770014893000)),
			ExpiredAt:     types.Time(time.UnixMilli(1801550893000)),
			ThirdPartyApp: "TradingView",
			SiteType:      "global",
		}
		assert.Equal(t, exp, result, "GetUTAAPIKeyInfo should decode every field")
		return
	}
	assert.NotEmpty(t, result.APIKey, "GetUTAAPIKeyInfo should return the API key")
}

func TestGetUTAAccountMode(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUTAAccountMode(t.Context())
	require.NoError(t, err, "GetUTAAccountMode must not error")
	if mockTests {
		exp := &UTAAccountModeResponse{
			SelfAccountMode:   "UNIFIED",
			UnifiedSubAccount: []uint64{100000003},
			ClassicSubAccount: []uint64{100000004, 100000005},
		}
		assert.Equal(t, exp, result, "GetUTAAccountMode should decode every field")
		return
	}
	assert.NotEmpty(t, result.SelfAccountMode, "GetUTAAccountMode should return the account mode")
}

func TestSetUTAAccountMode(t *testing.T) {
	t.Parallel()
	err := e.SetUTAAccountMode(t.Context(), "")
	require.ErrorIs(t, err, errAccountTypeMissing, "SetUTAAccountMode must reject an empty account type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	err = e.SetUTAAccountMode(t.Context(), "UNIFIED")
	assert.NoError(t, err, "SetUTAAccountMode should not error")
}

func TestGetUTALeverage(t *testing.T) {
	t.Parallel()
	_, err := e.GetUTALeverage(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUTALeverage must reject a nil request")
	_, err = e.GetUTALeverage(t.Context(), &UTALeverageRequest{})
	require.ErrorIs(t, err, errTradeTypeMissing, "GetUTALeverage must reject an empty trade type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		req *UTALeverageRequest
		exp []UTALeverageResponse
	}{
		{
			req: &UTALeverageRequest{Currency: currency.BTC, TradeType: "MARGIN", MarginMode: "CROSS"},
			exp: []UTALeverageResponse{{Currency: currency.BTC, Leverage: 5, MarginMode: "CROSS"}},
		},
		{
			req: &UTALeverageRequest{Symbol: "XBTUSDTM", TradeType: "FUTURES"},
			exp: []UTALeverageResponse{{Leverage: 80, MarginMode: "ISOLATE", Symbol: "XBTUSDTM"}},
		},
	} {
		t.Run(tc.req.TradeType, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetUTALeverage(t.Context(), tc.req)
			require.NoError(t, err, "GetUTALeverage must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetUTALeverage should decode every field")
				return
			}
			assert.NotEmpty(t, result, "GetUTALeverage should return leverage")
		})
	}
}

func TestModifyUTAFuturesLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.ModifyUTAFuturesLeverage(t.Context(), "", 80)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "ModifyUTAFuturesLeverage must reject an empty symbol")
	_, err = e.ModifyUTAFuturesLeverage(t.Context(), "XBTUSDTM", 0)
	require.ErrorIs(t, err, errInvalidLeverage, "ModifyUTAFuturesLeverage must reject a zero leverage")

	sent := sentRequest(t, func(ex *Exchange) error {
		_, err := ex.ModifyUTAFuturesLeverage(t.Context(), "XBTUSDTM", 80)
		return err
	})
	assert.JSONEq(t, `{"symbol":"XBTUSDTM","leverage":"80"}`, sent.body, "ModifyUTAFuturesLeverage should send the leverage as the string KuCoin documents")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.ModifyUTAFuturesLeverage(t.Context(), "XBTUSDTM", 80)
	require.NoError(t, err, "ModifyUTAFuturesLeverage must not error")
	if mockTests {
		exp := &ModifyUTAFuturesLeverageResponse{
			Code:     "200000",
			Leverage: 80,
		}
		assert.Equal(t, exp, result, "ModifyUTAFuturesLeverage should decode every field")
		return
	}
	assert.Positive(t, result.Leverage.Float64(), "ModifyUTAFuturesLeverage should return the leverage")
}

func TestModifyUTACrossMarginLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.ModifyUTACrossMarginLeverage(t.Context(), currency.BTC, 0)
	require.ErrorIs(t, err, errInvalidLeverage, "ModifyUTACrossMarginLeverage must reject a zero leverage")

	sent := sentRequest(t, func(ex *Exchange) error {
		_, err := ex.ModifyUTACrossMarginLeverage(t.Context(), currency.BTC, 5)
		return err
	})
	assert.JSONEq(t, `{"currency":"BTC","leverage":"5"}`, sent.body, "ModifyUTACrossMarginLeverage should send the leverage as the string KuCoin documents")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name     string
		currency currency.Code
		leverage float64
		exp      *ModifyUTACrossMarginLeverageResponse
	}{
		{name: "currency", currency: currency.BTC, leverage: 5, exp: &ModifyUTACrossMarginLeverageResponse{Currency: currency.BTC, Leverage: 5}},
		{name: "account", leverage: 3, exp: &ModifyUTACrossMarginLeverageResponse{Leverage: 3}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.ModifyUTACrossMarginLeverage(t.Context(), tc.currency, tc.leverage)
			require.NoError(t, err, "ModifyUTACrossMarginLeverage must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "ModifyUTACrossMarginLeverage should decode every field")
				return
			}
			assert.Positive(t, result.Leverage.Float64(), "ModifyUTACrossMarginLeverage should return the leverage")
		})
	}
}

func TestGetUTABorrowingRatesAndLimits(t *testing.T) {
	t.Parallel()
	_, err := e.GetUTABorrowingRatesAndLimits(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetUTABorrowingRatesAndLimits must reject an empty currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUTABorrowingRatesAndLimits(t.Context(), currency.USDT)
	require.NoError(t, err, "GetUTABorrowingRatesAndLimits must not error")
	if mockTests {
		exp := &UTABorrowingRatesAndLimitsResponse{
			CurrentRateHourly:       0.00000361,
			CurrentRateDaily:        0.00008658,
			BorrowLimitTotal:        4200000,
			BorrowLimitTotalHold:    1500,
			BorrowLimitHold:         500,
			InterestFreeBorrowLimit: 2000,
		}
		assert.Equal(t, exp, result, "GetUTABorrowingRatesAndLimits should decode every field")
		return
	}
	assert.Positive(t, result.BorrowLimitTotal.Float64(), "GetUTABorrowingRatesAndLimits should return a borrowing limit")
}

func TestGetUTAAccountLedger(t *testing.T) {
	t.Parallel()
	_, err := e.GetUTAAccountLedger(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUTAAccountLedger must reject a nil request")
	_, err = e.GetUTAAccountLedger(t.Context(), &UTAAccountLedgerRequest{})
	require.ErrorIs(t, err, errAccountTypeMissing, "GetUTAAccountLedger must reject an empty account type")
	_, err = e.GetUTAAccountLedger(t.Context(), &UTAAccountLedgerRequest{AccountType: "FUNDING", StartAt: time.UnixMilli(1785150000000), EndAt: time.UnixMilli(1785148000000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUTAAccountLedger must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	funding := &UTAAccountLedgerResponse{
		LastID: "1001",
		Items: []UTALedgerEntry{
			{
				AccountType:  "FUNDING",
				ID:           "6a67377a7fa12a0007ad4916",
				Currency:     currency.USDT,
				Direction:    "OUT",
				BusinessType: "Transfer",
				Amount:       1,
				Balance:      488941.00000108,
				Fee:          0.01,
				Tax:          0.001,
				Remark:       "Funding to Spot",
				Timestamp:    types.Time(time.UnixMilli(1785149306523)),
			},
		},
	}
	for _, tc := range []struct {
		name string
		req  *UTAAccountLedgerRequest
		exp  *UTAAccountLedgerResponse
	}{
		{name: "required", req: &UTAAccountLedgerRequest{AccountType: "FUNDING"}, exp: funding},
		{
			name: "filtered",
			req: &UTAAccountLedgerRequest{
				AccountType:  "FUNDING",
				BusinessType: "TRANSFER",
				Currency:     currency.USDT,
				Direction:    "OUT",
				StartAt:      time.UnixMilli(1785148000000),
				EndAt:        time.UnixMilli(1785150000000),
				PageSize:     50,
				LastID:       "1000",
			},
			exp: funding,
		},
		{
			name: "spot entry with a numeric ID",
			req:  &UTAAccountLedgerRequest{AccountType: "SPOT"},
			exp: &UTAAccountLedgerResponse{
				Items: []UTALedgerEntry{
					{
						AccountType:  "SPOT",
						ID:           "2721830813696",
						Currency:     currency.USDT,
						Direction:    "IN",
						BusinessType: "TRANSFER",
						Amount:       1,
						Balance:      501,
						Remark:       "{}",
						Timestamp:    types.Time(time.UnixMilli(1785149306635)),
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetUTAAccountLedger(t.Context(), tc.req)
			require.NoError(t, err, "GetUTAAccountLedger must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetUTAAccountLedger should decode every field")
				return
			}
			assert.NotNil(t, result, "GetUTAAccountLedger should return a result")
		})
	}
}

func TestUTAIDUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for in, exp := range map[string]UTAID{
		`"1928560862264"`:            "1928560862264",
		`"6a67377a7fa12a0007ad4916"`: "6a67377a7fa12a0007ad4916",
		`22075878662488064`:          "22075878662488064",
		`null`:                       "",
	} {
		id := UTAID("previous")
		require.NoErrorf(t, json.Unmarshal([]byte(in), &id), "UnmarshalJSON must not error for %s", in)
		assert.Equalf(t, exp, id, "UnmarshalJSON should keep the text of %s", in)
	}
	var id UTAID
	for _, in := range []string{`true`, `-5`, `1.5`, `{"id":"1"}`} {
		assert.ErrorIsf(t, json.Unmarshal([]byte(in), &id), common.ErrMalformedData, "UnmarshalJSON should reject %s", in)
	}
	assert.Error(t, id.UnmarshalJSON([]byte(`"1928560862264`)), "UnmarshalJSON should reject an unterminated string")
}

func TestGetUTAInterestHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetUTAInterestHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUTAInterestHistory must reject a nil request")
	_, err = e.GetUTAInterestHistory(t.Context(), &UTAInterestHistoryRequest{})
	require.ErrorIs(t, err, errAccountTypeMissing, "GetUTAInterestHistory must reject an empty account type")
	_, err = e.GetUTAInterestHistory(t.Context(), &UTAInterestHistoryRequest{AccountType: "UNIFIED", StartTime: time.UnixMilli(1764900000000), EndTime: time.UnixMilli(1764800000000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUTAInterestHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	exp := &UTAInterestHistoryResponse{
		Items: []UTAInterestRecord{
			{
				Liability:             0.00665034,
				Interest:              0.000000007344013,
				HourlyInterestRate:    0.00000445,
				Currency:              currency.USDC,
				Timestamp:             types.Time(time.UnixMilli(1764853260122)),
				InterestFreeLiability: 0.005,
			},
		},
		LastID: 2002,
	}
	for name, req := range map[string]*UTAInterestHistoryRequest{
		"required": {AccountType: "UNIFIED"},
		"filtered": {
			AccountType: "UNIFIED",
			Currency:    currency.USDC,
			StartTime:   time.UnixMilli(1764800000000),
			EndTime:     time.UnixMilli(1764900000000),
			Page:        1,
			Size:        50,
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetUTAInterestHistory(t.Context(), req)
			require.NoError(t, err, "GetUTAInterestHistory must not error")
			if mockTests {
				assert.Equal(t, exp, result, "GetUTAInterestHistory should decode every field")
				return
			}
			assert.NotNil(t, result, "GetUTAInterestHistory should return a result")
		})
	}
}

func TestGetUTAAPIRateLimit(t *testing.T) {
	t.Parallel()
	_, err := e.GetUTAAPIRateLimit(t.Context())
	require.ErrorIs(t, err, errUIDRequired, "GetUTAAPIRateLimit must reject no UIDs")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		key, err := e.GetUTAAPIKeyInfo(t.Context())
		require.NoError(t, err, "GetUTAAPIKeyInfo must not error")
		require.NotNil(t, key, "GetUTAAPIKeyInfo must return the key's details")
		uid := strconv.FormatUint(key.UID, 10)
		result, err := e.GetUTAAPIRateLimit(t.Context(), uid)
		require.NoError(t, err, "GetUTAAPIRateLimit must not error")
		require.NotNil(t, result, "GetUTAAPIRateLimit must return a result")
		for i := range result.List {
			assert.Equalf(t, uid, result.List[i].UID, "GetUTAAPIRateLimit should return rate limit %d for the UID asked for", i)
		}
		return
	}
	result, err := e.GetUTAAPIRateLimit(t.Context(), "100000002", "100000003")
	require.NoError(t, err, "GetUTAAPIRateLimit must not error")
	exp := &UTAAPIRateLimitResponse{
		List: []UTAAPIRateLimit{
			{UID: "100000002", Rate: 50},
			{UID: "100000003", Rate: 10},
		},
	}
	assert.Equal(t, exp, result, "GetUTAAPIRateLimit should decode every field")
}

func TestGetUTAAllAPIRateLimit(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUTAAllAPIRateLimit(t.Context())
	require.NoError(t, err, "GetUTAAllAPIRateLimit must not error")
	if mockTests {
		exp := &UTAAPIRateLimitResponse{
			List: []UTAAPIRateLimit{
				{UID: "100000001", Rate: 150},
				{UID: "100000002", Rate: 50},
				{UID: "100000003", Rate: 10},
			},
		}
		assert.Equal(t, exp, result, "GetUTAAllAPIRateLimit should decode every field")
		return
	}
	assert.NotEmpty(t, result.List, "GetUTAAllAPIRateLimit should return the master account's rate limit")
}

func TestGetUTAAPIRateLimitCap(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUTAAPIRateLimitCap(t.Context())
	require.NoError(t, err, "GetUTAAPIRateLimitCap must not error")
	if mockTests {
		exp := &UTAAPIRateLimitCapResponse{
			VIPLevel:       2,
			MainRateLimit:  450,
			SubRateLimit:   50,
			AllocatedQuota: 50,
			RemainingQuota: 400,
			DefaultQuota:   10,
		}
		assert.Equal(t, exp, result, "GetUTAAPIRateLimitCap should decode every field")
		return
	}
	assert.Positive(t, result.MainRateLimit, "GetUTAAPIRateLimitCap should return the main rate limit")
}

func TestGetUTAFeeRate(t *testing.T) {
	t.Parallel()
	_, err := e.GetUTAFeeRate(t.Context(), "", []string{"BTC-USDT"})
	require.ErrorIs(t, err, errTradeTypeMissing, "GetUTAFeeRate must reject an empty trade type")
	_, err = e.GetUTAFeeRate(t.Context(), "SPOT", nil)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetUTAFeeRate must reject no symbols")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUTAFeeRate(t.Context(), "SPOT", []string{"BTC-USDT", "ETH-USDT"})
	require.NoError(t, err, "GetUTAFeeRate must not error")
	if mockTests {
		exp := &UTAFeeRateResponse{
			TradeType: "SPOT",
			List: []UTASymbolFeeRate{
				{Symbol: "BTC-USDT", TakerFeeRate: 0.001, MakerFeeRate: 0.0008},
				{Symbol: "ETH-USDT", TakerFeeRate: 0.001, MakerFeeRate: 0.0008},
			},
		}
		assert.Equal(t, exp, result, "GetUTAFeeRate should decode every field")
		return
	}
	assert.Len(t, result.List, 2, "GetUTAFeeRate should return each symbol's fee rates")
}

func TestSetUTAKCSFeeDeduction(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SetUTAKCSFeeDeduction(t.Context(), true)
	require.NoError(t, err, "SetUTAKCSFeeDeduction must not error")
	if mockTests {
		assert.Equal(t, &SetUTAKCSFeeDeductionResponse{Enabled: true}, result, "SetUTAKCSFeeDeduction should decode every field")
		return
	}
	assert.True(t, result.Enabled, "SetUTAKCSFeeDeduction should enable KCS fee deduction")
}

func TestGetUTAThirdPartyCustodyAccountCurrencyLimits(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	exp := []UTAThirdPartyCustodyCurrencyLimitResponse{
		{
			Custodian:    "BITGO_SG",
			Currency:     currency.BTC,
			CustodyQuota: 2.92936257,
		},
	}
	for _, tc := range []struct {
		name      string
		custodian string
		currency  currency.Code
	}{
		{name: "all"},
		{name: "filtered", custodian: "BITGO_SG", currency: currency.BTC},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetUTAThirdPartyCustodyAccountCurrencyLimits(t.Context(), tc.custodian, tc.currency)
			require.NoError(t, err, "GetUTAThirdPartyCustodyAccountCurrencyLimits must not error")
			if mockTests {
				assert.Equal(t, exp, result, "GetUTAThirdPartyCustodyAccountCurrencyLimits should decode every field")
				return
			}
			assert.NotNil(t, result, "GetUTAThirdPartyCustodyAccountCurrencyLimits should return a result")
		})
	}
}
