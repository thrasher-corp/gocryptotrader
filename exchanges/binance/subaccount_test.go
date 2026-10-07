package binance

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const testSubAccountAPIKey = "mock-sub-account-api-key"

func TestCreateVirtualSubAccount(t *testing.T) {
	t.Parallel()
	_, err := e.CreateVirtualSubAccount(t.Context(), "")
	require.ErrorIs(t, err, errSubAccountStringRequired, "CreateVirtualSubAccount must reject an empty sub-account string")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CreateVirtualSubAccount(t.Context(), "something-string")
	require.NoError(t, err, "CreateVirtualSubAccount must not error")
	if mockTests {
		exp := &CreateVirtualSubAccountResponse{
			Email: "something-string_virtual@example.com",
		}
		assert.Equal(t, exp, result, "CreateVirtualSubAccount should decode every field")
		return
	}
	assert.NotEmpty(t, result.Email, "CreateVirtualSubAccount should return an email")
}

func TestEnableFuturesSubAccount(t *testing.T) {
	t.Parallel()
	_, err := e.EnableFuturesSubAccount(t.Context(), "address")
	require.ErrorIs(t, err, errValidEmailRequired, "EnableFuturesSubAccount must reject an invalid email")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.EnableFuturesSubAccount(t.Context(), "address@example.com")
	require.NoError(t, err, "EnableFuturesSubAccount must not error")
	if mockTests {
		exp := &EnableFuturesSubAccountResponse{
			Email:            "address@example.com",
			IsFuturesEnabled: true,
		}
		assert.Equal(t, exp, result, "EnableFuturesSubAccount should decode every field")
		return
	}
	assert.True(t, result.IsFuturesEnabled, "EnableFuturesSubAccount should enable futures")
}

func TestEnableOptionsForSubAccount(t *testing.T) {
	t.Parallel()
	_, err := e.EnableOptionsForSubAccount(t.Context(), "")
	require.ErrorIs(t, err, errValidEmailRequired, "EnableOptionsForSubAccount must reject an empty email")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.EnableOptionsForSubAccount(t.Context(), "address@example.com")
	require.NoError(t, err, "EnableOptionsForSubAccount must not error")
	if mockTests {
		exp := &EnableOptionsForSubAccountResponse{
			Email:             "address@example.com",
			IsEOptionsEnabled: true,
		}
		assert.Equal(t, exp, result, "EnableOptionsForSubAccount should decode every field")
		return
	}
	assert.True(t, result.IsEOptionsEnabled, "EnableOptionsForSubAccount should enable options")
}

func TestGetV1FuturesPositionRiskSubAccount(t *testing.T) {
	t.Parallel()
	_, err := e.GetV1FuturesPositionRiskSubAccount(t.Context(), "address")
	require.ErrorIs(t, err, errValidEmailRequired, "GetV1FuturesPositionRiskSubAccount must reject an invalid email")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetV1FuturesPositionRiskSubAccount(t.Context(), "address@example.com")
	require.NoError(t, err, "GetV1FuturesPositionRiskSubAccount must not error")
	if mockTests {
		exp := []SubAccountFuturesPositionRisk{
			{
				EntryPrice:       9975.12,
				Leverage:         50,
				MaxNotional:      1000000,
				LiquidationPrice: 7963.54,
				MarkPrice:        9973.50770517,
				PositionAmount:   0.01,
				Symbol:           "BTCUSDT",
				UnrealisedProfit: -0.01612295,
			},
		}
		assert.Equal(t, exp, result, "GetV1FuturesPositionRiskSubAccount should decode every field")
		return
	}
	assert.NotNil(t, result, "GetV1FuturesPositionRiskSubAccount should return positions")
}

func TestGetV2FuturesPositionRiskSubAccount(t *testing.T) {
	t.Parallel()
	_, err := e.GetV2FuturesPositionRiskSubAccount(t.Context(), "address", 1)
	require.ErrorIs(t, err, errValidEmailRequired, "GetV2FuturesPositionRiskSubAccount must reject an invalid email")
	_, err = e.GetV2FuturesPositionRiskSubAccount(t.Context(), "address@example.com", 0)
	require.ErrorIs(t, err, errInvalidFuturesType, "GetV2FuturesPositionRiskSubAccount must reject an unset futures type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name        string
		futuresType uint64
		exp         *SubAccountFuturesPositionRiskV2Response
	}{
		{
			name: "USDT-margined", futuresType: 1,
			exp: &SubAccountFuturesPositionRiskV2Response{
				FuturePositionRiskVos: []SubAccountFuturesPositionRisk{
					{
						EntryPrice:       9975.12,
						Leverage:         50,
						MaxNotional:      1000000,
						LiquidationPrice: 7963.54,
						MarkPrice:        9973.50770517,
						PositionAmount:   0.01,
						Symbol:           "BTCUSDT",
						UnrealisedProfit: -0.01612295,
					},
				},
			},
		},
		{
			name: "coin-margined", futuresType: 2,
			exp: &SubAccountFuturesPositionRiskV2Response{
				DeliveryPositionRiskVos: []SubAccountDeliveryPositionRisk{
					{
						EntryPrice:       9975.12,
						MarkPrice:        9973.50770517,
						Leverage:         20,
						Isolated:         true,
						IsolatedWallet:   0.05,
						IsolatedMargin:   0.04999801,
						IsAutoAddMargin:  true,
						PositionSide:     "BOTH",
						PositionAmount:   1.23,
						Symbol:           "BTCUSD_201225",
						UnrealisedProfit: -0.00000199,
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetV2FuturesPositionRiskSubAccount(t.Context(), "address@example.com", tc.futuresType)
			require.NoError(t, err, "GetV2FuturesPositionRiskSubAccount must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetV2FuturesPositionRiskSubAccount should decode every field")
				return
			}
			assert.NotNil(t, result, "GetV2FuturesPositionRiskSubAccount should return positions")
		})
	}
}

func TestGetSubAccountStatusOnMarginFutures(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSubAccountStatusOnMarginFutures(t.Context(), "myemail@example.com")
	require.NoError(t, err, "GetSubAccountStatusOnMarginFutures must not error")
	if mockTests {
		exp := []SubAccountStatus{
			{
				Email:            "myemail@example.com",
				IsSubUserEnabled: true,
				IsUserActive:     true,
				InsertTime:       types.Time(time.UnixMilli(1570791523523)),
				IsMarginEnabled:  true,
				IsFutureEnabled:  true,
				Mobile:           1570791523523,
			},
		}
		assert.Equal(t, exp, result, "GetSubAccountStatusOnMarginFutures should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSubAccountStatusOnMarginFutures should return statuses")
}

func TestGetSubAccountList(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAccountList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSubAccountList must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSubAccountList(t.Context(), &SubAccountListRequest{Email: "testsub@example.com", IsFreeze: new(true), Page: 1, Limit: 10})
	require.NoError(t, err, "GetSubAccountList must not error")
	if mockTests {
		exp := &SubAccountListResponse{
			SubAccounts: []SubAccount{
				{
					SubUserID:                   123456,
					Email:                       "testsub@example.com",
					Remark:                      "remark",
					IsFreeze:                    true,
					CreateTime:                  types.Time(time.Unix(1544433328, 0)),
					IsManagedSubAccount:         true,
					IsAssetManagementSubAccount: true,
				},
			},
			Success: true,
		}
		assert.Equal(t, exp, result, "GetSubAccountList should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSubAccountList should return sub-accounts")
}

func TestGetSubAccountTransactionStatistics(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSubAccountTransactionStatistics(t.Context(), "address@example.com")
	require.NoError(t, err, "GetSubAccountTransactionStatistics must not error")
	if mockTests {
		exp := &SubAccountTransactionStatisticsResponse{
			Recent30DayBTCTotal:         0.0035,
			Recent30DayBTCFuturesTotal:  0.0012,
			Recent30DayBTCMarginTotal:   0.0003,
			Recent30DayBUSDTotal:        230.5,
			Recent30DayBUSDFuturesTotal: 78.25,
			Recent30DayBUSDMarginTotal:  19.5,
			TradeInfoVos: []SubAccountTradeInfo{
				{
					UserID:      1000138138384,
					BTC:         0.0001,
					BTCFutures:  0.00004,
					BTCMargin:   0.00001,
					BUSD:        6.5,
					BUSDFutures: 2.6,
					BUSDMargin:  0.65,
					Date:        types.Time(time.Unix(1744070400, 0)),
				},
			},
		}
		assert.Equal(t, exp, result, "GetSubAccountTransactionStatistics should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSubAccountTransactionStatistics should return statistics")
}

func TestAddIPRestrictionForSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	_, err := e.AddIPRestrictionForSubAccountAPIKey(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "AddIPRestrictionForSubAccountAPIKey must reject a nil request")
	_, err = e.AddIPRestrictionForSubAccountAPIKey(t.Context(), &SubAccountAPIIPRestrictionRequest{Email: "addressthrasher", SubAccountAPIKey: testSubAccountAPIKey})
	require.ErrorIs(t, err, errValidEmailRequired, "AddIPRestrictionForSubAccountAPIKey must reject an invalid email")
	_, err = e.AddIPRestrictionForSubAccountAPIKey(t.Context(), &SubAccountAPIIPRestrictionRequest{Email: "address@example.com"})
	require.ErrorIs(t, err, errEmptySubAccountAPIKey, "AddIPRestrictionForSubAccountAPIKey must reject an empty API key")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.AddIPRestrictionForSubAccountAPIKey(t.Context(), &SubAccountAPIIPRestrictionRequest{
		Email:                "address@example.com",
		SubAccountAPIKey:     testSubAccountAPIKey,
		RestrictToTrustedIPs: true,
		IPAddresses:          []string{"192.0.2.1", "192.0.2.2"},
	})
	require.NoError(t, err, "AddIPRestrictionForSubAccountAPIKey must not error")
	if mockTests {
		exp := &AddSubAccountAPIIPRestrictionResponse{
			Status: "2",
			IPList: []string{
				"192.0.2.1",
				"192.0.2.2",
			},
			UpdateTime: types.Time(time.Unix(1744110000, 0)),
			APIKey:     "mock-sub-account-api-key",
		}
		assert.Equal(t, exp, result, "AddIPRestrictionForSubAccountAPIKey should decode every field")
		return
	}
	assert.NotEmpty(t, result.IPList, "AddIPRestrictionForSubAccountAPIKey should return the IP list")
}

func TestDeleteIPListForSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	_, err := e.DeleteIPListForSubAccountAPIKey(t.Context(), "emailaddress", testSubAccountAPIKey, []string{"192.0.2.1"})
	require.ErrorIs(t, err, errValidEmailRequired, "DeleteIPListForSubAccountAPIKey must reject an invalid email")
	_, err = e.DeleteIPListForSubAccountAPIKey(t.Context(), "emailaddress@example.com", "", []string{"192.0.2.1"})
	require.ErrorIs(t, err, errEmptySubAccountAPIKey, "DeleteIPListForSubAccountAPIKey must reject an empty API key")
	_, err = e.DeleteIPListForSubAccountAPIKey(t.Context(), "emailaddress@example.com", testSubAccountAPIKey, nil)
	require.ErrorIs(t, err, errSubAccountIPAddressRequired, "DeleteIPListForSubAccountAPIKey must reject an empty IP list")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.DeleteIPListForSubAccountAPIKey(t.Context(), "emailaddress@example.com", testSubAccountAPIKey, []string{"192.0.2.1"})
	require.NoError(t, err, "DeleteIPListForSubAccountAPIKey must not error")
	if mockTests {
		exp := &SubAccountAPIIPRestrictionResponse{
			IPRestrict: true,
			IPList: []string{
				"192.0.2.2",
			},
			UpdateTime: types.Time(time.Unix(1744110000, 0)),
			APIKey:     "mock-sub-account-api-key",
		}
		assert.Equal(t, exp, result, "DeleteIPListForSubAccountAPIKey should decode every field")
		return
	}
	assert.NotNil(t, result, "DeleteIPListForSubAccountAPIKey should return the IP restriction")
}

func TestGetIPRestrictionForSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	_, err := e.GetIPRestrictionForSubAccountAPIKey(t.Context(), "emailaddress", testSubAccountAPIKey)
	require.ErrorIs(t, err, errValidEmailRequired, "GetIPRestrictionForSubAccountAPIKey must reject an invalid email")
	_, err = e.GetIPRestrictionForSubAccountAPIKey(t.Context(), "emailaddress@example.com", "")
	require.ErrorIs(t, err, errEmptySubAccountAPIKey, "GetIPRestrictionForSubAccountAPIKey must reject an empty API key")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetIPRestrictionForSubAccountAPIKey(t.Context(), "emailaddress@example.com", testSubAccountAPIKey)
	require.NoError(t, err, "GetIPRestrictionForSubAccountAPIKey must not error")
	if mockTests {
		exp := &SubAccountAPIIPRestrictionResponse{
			IPRestrict: true,
			IPList: []string{
				"192.0.2.1",
				"192.0.2.2",
			},
			UpdateTime: types.Time(time.Unix(1744110000, 0)),
			APIKey:     "mock-sub-account-api-key",
		}
		assert.Equal(t, exp, result, "GetIPRestrictionForSubAccountAPIKey should decode every field")
		return
	}
	assert.NotNil(t, result, "GetIPRestrictionForSubAccountAPIKey should return the IP restriction")
}

func TestFuturesTransferSubAccount(t *testing.T) {
	t.Parallel()
	testSubAccountWalletTransfer(t, e.FuturesTransferSubAccount, &SubAccountTransferResponse{
		TransactionID: "2966662589",
	})
}

func testSubAccountWalletTransfer(t *testing.T, transfer func(context.Context, *SubAccountWalletTransferRequest) (*SubAccountTransferResponse, error), exp *SubAccountTransferResponse) {
	t.Helper()
	_, err := transfer(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "transfer must reject a nil request")
	_, err = transfer(t.Context(), &SubAccountWalletTransferRequest{Email: "someone.com", Asset: currency.BTC, Amount: 1.1, Type: 1})
	require.ErrorIs(t, err, errValidEmailRequired, "transfer must reject an invalid email")
	_, err = transfer(t.Context(), &SubAccountWalletTransferRequest{Email: "someone@example.com", Amount: 1.1, Type: 1})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "transfer must reject an empty asset")
	_, err = transfer(t.Context(), &SubAccountWalletTransferRequest{Email: "someone@example.com", Asset: currency.BTC, Type: 1})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "transfer must reject a zero amount")
	_, err = transfer(t.Context(), &SubAccountWalletTransferRequest{Email: "someone@example.com", Asset: currency.BTC, Amount: 1.1})
	require.ErrorIs(t, err, errTransferTypeRequired, "transfer must reject an unset type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := transfer(t.Context(), &SubAccountWalletTransferRequest{Email: "someone@example.com", Asset: currency.BTC, Amount: 1.1, Type: 1})
	require.NoError(t, err, "transfer must not error")
	if mockTests {
		assert.Equal(t, exp, result, "transfer should decode every field")
		return
	}
	assert.NotEmpty(t, result.TransactionID, "transfer should return a transaction ID")
}

func TestGetDetailSubAccountFuturesAccount(t *testing.T) {
	t.Parallel()
	_, err := e.GetDetailSubAccountFuturesAccount(t.Context(), "address")
	require.ErrorIs(t, err, errValidEmailRequired, "GetDetailSubAccountFuturesAccount must reject an invalid email")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetDetailSubAccountFuturesAccount(t.Context(), "address@example.com")
	require.NoError(t, err, "GetDetailSubAccountFuturesAccount must not error")
	if mockTests {
		exp := &SubAccountFuturesAccountResponse{
			Email: "address@example.com",
			Asset: currency.USDT,
			Assets: []SubAccountFuturesAsset{
				{
					Asset:                  currency.USDT,
					InitialMargin:          12.5,
					MaintenanceMargin:      0.5,
					MarginBalance:          100.88308,
					MaxWithdrawAmount:      88.38308,
					OpenOrderInitialMargin: 2.5,
					PositionInitialMargin:  10,
					UnrealisedProfit:       0.88308,
					WalletBalance:          100,
				},
			},
			CanDeposit:                  true,
			CanTrade:                    true,
			CanWithdraw:                 true,
			FeeTier:                     2,
			MaxWithdrawAmount:           88.38308,
			TotalInitialMargin:          12.5,
			TotalMaintenanceMargin:      0.5,
			TotalMarginBalance:          100.88308,
			TotalOpenOrderInitialMargin: 2.5,
			TotalPositionInitialMargin:  10,
			TotalUnrealisedProfit:       0.88308,
			TotalWalletBalance:          100,
			UpdateTime:                  types.Time(time.Unix(1744110000, 0)),
		}
		assert.Equal(t, exp, result, "GetDetailSubAccountFuturesAccount should decode every field")
		return
	}
	assert.NotNil(t, result, "GetDetailSubAccountFuturesAccount should return the futures account")
}

func TestGetDetailOnSubAccountsFuturesAccountV2(t *testing.T) {
	t.Parallel()
	_, err := e.GetDetailOnSubAccountsFuturesAccountV2(t.Context(), "thrasher", 1)
	require.ErrorIs(t, err, errValidEmailRequired, "GetDetailOnSubAccountsFuturesAccountV2 must reject an invalid email")
	_, err = e.GetDetailOnSubAccountsFuturesAccountV2(t.Context(), "address@example.com", 0)
	require.ErrorIs(t, err, errInvalidFuturesType, "GetDetailOnSubAccountsFuturesAccountV2 must reject an unset futures type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name        string
		futuresType uint64
		exp         *SubAccountFuturesAccountV2Response
	}{
		{
			name: "USDT-margined", futuresType: 1,
			exp: &SubAccountFuturesAccountV2Response{
				FutureAccountResponse: &SubAccountFutureAccount{
					Email: "address@example.com",
					Assets: []SubAccountFuturesAsset{
						{
							Asset:                  currency.USDT,
							InitialMargin:          12.5,
							MaintenanceMargin:      0.5,
							MarginBalance:          100.88308,
							MaxWithdrawAmount:      88.38308,
							OpenOrderInitialMargin: 2.5,
							PositionInitialMargin:  10,
							UnrealisedProfit:       0.88308,
							WalletBalance:          100,
						},
					},
					CanDeposit:                  true,
					CanTrade:                    true,
					CanWithdraw:                 true,
					FeeTier:                     2,
					MaxWithdrawAmount:           88.38308,
					TotalInitialMargin:          12.5,
					TotalMaintenanceMargin:      0.5,
					TotalMarginBalance:          100.88308,
					TotalOpenOrderInitialMargin: 2.5,
					TotalPositionInitialMargin:  10,
					TotalUnrealisedProfit:       0.88308,
					TotalWalletBalance:          100,
					UpdateTime:                  types.Time(time.Unix(1744110000, 0)),
				},
			},
		},
		{
			name: "coin-margined", futuresType: 2,
			exp: &SubAccountFuturesAccountV2Response{
				DeliveryAccountResponse: &SubAccountDeliveryAccount{
					Email: "address@example.com",
					Assets: []SubAccountFuturesAsset{
						{
							Asset:                  currency.BTC,
							InitialMargin:          0.0125,
							MaintenanceMargin:      0.0005,
							MarginBalance:          0.10088308,
							MaxWithdrawAmount:      0.08838308,
							OpenOrderInitialMargin: 0.0025,
							PositionInitialMargin:  0.01,
							UnrealisedProfit:       0.00088308,
							WalletBalance:          0.1,
						},
					},
					CanDeposit:  true,
					CanTrade:    true,
					CanWithdraw: true,
					FeeTier:     2,
					UpdateTime:  types.Time(time.Unix(1744110000, 0)),
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetDetailOnSubAccountsFuturesAccountV2(t.Context(), "address@example.com", tc.futuresType)
			require.NoError(t, err, "GetDetailOnSubAccountsFuturesAccountV2 must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetDetailOnSubAccountsFuturesAccountV2 should decode every field")
				return
			}
			assert.NotNil(t, result, "GetDetailOnSubAccountsFuturesAccountV2 should return the futures account")
		})
	}
}

func TestGetDetailOnSubAccountMarginAccount(t *testing.T) {
	t.Parallel()
	_, err := e.GetDetailOnSubAccountMarginAccount(t.Context(), "com")
	require.ErrorIs(t, err, errValidEmailRequired, "GetDetailOnSubAccountMarginAccount must reject an invalid email")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetDetailOnSubAccountMarginAccount(t.Context(), "test@example.com")
	require.NoError(t, err, "GetDetailOnSubAccountMarginAccount must not error")
	if mockTests {
		exp := &SubAccountMarginAccountDetailResponse{
			Email:               "test@example.com",
			MarginLevel:         11.64405625,
			TotalAssetOfBTC:     6.82728457,
			TotalLiabilityOfBTC: 0.50633215,
			TotalNetAssetOfBTC:  6.32095242,
			MarginTradeCoefficientVo: SubAccountMarginTradeCoefficient{
				ForceLiquidationBar: 1.1,
				MarginCallBar:       1.5,
				NormalBar:           2,
			},
			MarginUserAssetVoList: []MarginAssetBalance{
				{
					Asset:    currency.BTC,
					Borrowed: 0.5,
					Free:     6.8,
					Interest: 0.00633215,
					Locked:   0.02728457,
					NetAsset: 6.32095242,
				},
			},
		}
		assert.Equal(t, exp, result, "GetDetailOnSubAccountMarginAccount should decode every field")
		return
	}
	assert.NotNil(t, result, "GetDetailOnSubAccountMarginAccount should return the margin account")
}

func TestGetSubAccountDepositAddress(t *testing.T) {
	t.Parallel()
	testSubAccountDepositAddress(t, e.GetSubAccountDepositAddress, []subAccountDepositAddressCase{
		{
			req: &SubAccountDepositAddressRequest{Email: "the_address@example.com", Coin: currency.BTC, Network: "LIGHTNING", Amount: 0.1},
			exp: &SubAccountDepositAddressResponse{
				Address: "lnbc100m1pn2s3kspp5qqqsyqcyq5rqwzqfqqqsyqcyq5rqwzqfqqqsyqcyq5rqwzqfqypq",
				Coin:    currency.BTC,
			},
		},
		{
			req: &SubAccountDepositAddressRequest{Email: "the_address@example.com", Coin: currency.XRP, Network: "XRP"},
			exp: &SubAccountDepositAddressResponse{
				Address: "rEb8TK3gBgk5auZkwc6sHnwrGVJH8DuaLh",
				Coin:    currency.XRP,
				Tag:     "101764890",
				URL:     "https://xrpscan.com/account/rEb8TK3gBgk5auZkwc6sHnwrGVJH8DuaLh",
			},
		},
	})
}

// subAccountDepositAddressCase holds a sub-account deposit address request and the response its fixture records
type subAccountDepositAddressCase struct {
	req *SubAccountDepositAddressRequest
	exp *SubAccountDepositAddressResponse
}

func testSubAccountDepositAddress(t *testing.T, depositAddress func(context.Context, *SubAccountDepositAddressRequest) (*SubAccountDepositAddressResponse, error), cases []subAccountDepositAddressCase) {
	t.Helper()
	_, err := depositAddress(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "deposit address must reject a nil request")
	_, err = depositAddress(t.Context(), &SubAccountDepositAddressRequest{Coin: currency.BTC})
	require.ErrorIs(t, err, errValidEmailRequired, "deposit address must reject an empty email")
	_, err = depositAddress(t.Context(), &SubAccountDepositAddressRequest{Email: "the_address@example.com"})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "deposit address must reject an empty coin")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range cases {
		t.Run(tc.req.Coin.String(), func(t *testing.T) {
			t.Parallel()
			result, err := depositAddress(t.Context(), tc.req)
			require.NoError(t, err, "deposit address must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "deposit address should decode every field")
				return
			}
			assert.NotEmpty(t, result.Address, "deposit address should return an address")
		})
	}
}

func TestGetSubAccountDepositHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAccountDepositHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSubAccountDepositHistory must reject a nil request")
	_, err = e.GetSubAccountDepositHistory(t.Context(), &SubAccountDepositHistoryRequest{Email: "someoneio", Coin: currency.BTC})
	require.ErrorIs(t, err, errValidEmailRequired, "GetSubAccountDepositHistory must reject an invalid email")
	startTime, endTime := getTime()
	_, err = e.GetSubAccountDepositHistory(t.Context(), &SubAccountDepositHistoryRequest{Email: "someone@example.com", StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSubAccountDepositHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *SubAccountDepositHistoryRequest
		exp  []SubAccountDepositRecord
	}{
		{
			name: "status",
			req:  &SubAccountDepositHistoryRequest{Email: "someone@example.com", Coin: currency.BTC, Status: new(uint64(1)), StartTime: startTime, EndTime: endTime, Limit: 100, Offset: 1},
			exp: []SubAccountDepositRecord{
				{
					ID:           "769800519366885380",
					Amount:       0.001,
					Coin:         currency.BTC,
					Network:      "BTC",
					Status:       1,
					Address:      "1HPn8Rx2y6nNSfagQBKy27GB99Vbzg89wv",
					TxID:         "Off-chain transfer 5b8e1d3f7a9c2e4b",
					InsertTime:   types.Time(time.Unix(1744110000, 0)),
					TransferType: 1,
					ConfirmTimes: "1/1",
					WalletType:   1,
				},
			},
		},
		{
			name: "source",
			req:  &SubAccountDepositHistoryRequest{Email: "someone@example.com", IncludeSource: true, Coin: currency.XRP, TxID: "98A3EA560C6B3336D348B6C83F0F95ECE4F1F5919E94BD006E5BF3BF264FACFC"},
			exp: []SubAccountDepositRecord{
				{
					ID:            "769800519366885381",
					Amount:        25,
					Coin:          currency.XRP,
					Network:       "XRP",
					Status:        1,
					Address:       "rEb8TK3gBgk5auZkwc6sHnwrGVJH8DuaLh",
					AddressTag:    "101764890",
					TxID:          "98A3EA560C6B3336D348B6C83F0F95ECE4F1F5919E94BD006E5BF3BF264FACFC",
					InsertTime:    types.Time(time.Unix(1744120000, 0)),
					ConfirmTimes:  "1/1",
					UnlockConfirm: 2,
					SourceAddress: "rPEPPER7kfTD9w2To4CQk6UCfuHM9c6GDY",
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetSubAccountDepositHistory(t.Context(), tc.req)
			require.NoError(t, err, "GetSubAccountDepositHistory must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetSubAccountDepositHistory should decode every field")
				return
			}
			assert.NotNil(t, result, "GetSubAccountDepositHistory should return deposits")
		})
	}
}

func TestGetSummaryOfSubAccountFuturesAccount(t *testing.T) {
	t.Parallel()
	_, err := e.GetSummaryOfSubAccountFuturesAccount(t.Context(), 0, 10)
	require.ErrorIs(t, err, errPageNumberRequired, "GetSummaryOfSubAccountFuturesAccount must reject an unset page")
	_, err = e.GetSummaryOfSubAccountFuturesAccount(t.Context(), 1, 0)
	require.ErrorIs(t, err, errLimitNumberRequired, "GetSummaryOfSubAccountFuturesAccount must reject an unset limit")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSummaryOfSubAccountFuturesAccount(t.Context(), 1, 10)
	require.NoError(t, err, "GetSummaryOfSubAccountFuturesAccount must not error")
	if mockTests {
		exp := &SubAccountFuturesAccountSummaryResponse{
			TotalInitialMargin:          9.831374,
			TotalMaintenanceMargin:      0.415687,
			TotalMarginBalance:          23.03235621,
			TotalOpenOrderInitialMargin: 9,
			TotalPositionInitialMargin:  0.831374,
			TotalUnrealisedProfit:       0.0321971,
			TotalWalletBalance:          22.15879444,
			Asset:                       currency.USD,
			SubAccountList: []SubAccountFuturesSummary{
				{
					Email:                       "sub123@example.com",
					TotalInitialMargin:          9,
					TotalMaintenanceMargin:      0.415687,
					TotalMarginBalance:          22.12659734,
					TotalOpenOrderInitialMargin: 9,
					TotalPositionInitialMargin:  0.831374,
					TotalUnrealisedProfit:       0.0321971,
					TotalWalletBalance:          22.09440024,
					Asset:                       currency.USD,
				},
			},
		}
		assert.Equal(t, exp, result, "GetSummaryOfSubAccountFuturesAccount should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSummaryOfSubAccountFuturesAccount should return a summary")
}

func TestGetSummaryOfSubAccountsFuturesAccountV2(t *testing.T) {
	t.Parallel()
	_, err := e.GetSummaryOfSubAccountsFuturesAccountV2(t.Context(), 0, 0, 10)
	require.ErrorIs(t, err, errInvalidFuturesType, "GetSummaryOfSubAccountsFuturesAccountV2 must reject an unset futures type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name        string
		futuresType uint64
		exp         *SubAccountFuturesAccountSummaryV2Response
	}{
		{
			name: "USDT-margined", futuresType: 1,
			exp: &SubAccountFuturesAccountSummaryV2Response{
				FutureAccountSummaryResponse: &SubAccountFuturesAccountSummaryResponse{
					TotalInitialMargin:          9.831374,
					TotalMaintenanceMargin:      0.415687,
					TotalMarginBalance:          23.03235621,
					TotalOpenOrderInitialMargin: 9,
					TotalPositionInitialMargin:  0.831374,
					TotalUnrealisedProfit:       0.0321971,
					TotalWalletBalance:          22.15879444,
					Asset:                       currency.USD,
					SubAccountList: []SubAccountFuturesSummary{
						{
							Email:                       "sub123@example.com",
							TotalInitialMargin:          9,
							TotalMaintenanceMargin:      0.415687,
							TotalMarginBalance:          22.12659734,
							TotalOpenOrderInitialMargin: 9,
							TotalPositionInitialMargin:  0.831374,
							TotalUnrealisedProfit:       0.0321971,
							TotalWalletBalance:          22.09440024,
							Asset:                       currency.USD,
						},
					},
				},
			},
		},
		{
			name: "coin-margined", futuresType: 2,
			exp: &SubAccountFuturesAccountSummaryV2Response{
				DeliveryAccountSummaryResponse: &SubAccountDeliveryAccountSummary{
					TotalMarginBalanceOfBTC:    25.03221121,
					TotalUnrealisedProfitOfBTC: 0.1223341,
					TotalWalletBalanceOfBTC:    22.15879444,
					Asset:                      currency.BTC,
					SubAccountList: []SubAccountDeliverySummary{
						{
							Email:                 "sub123@example.com",
							TotalMarginBalance:    22.12659734,
							TotalUnrealisedProfit: 0.1223341,
							TotalWalletBalance:    22.00426324,
							Asset:                 currency.BTC,
						},
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetSummaryOfSubAccountsFuturesAccountV2(t.Context(), tc.futuresType, 1, 10)
			require.NoError(t, err, "GetSummaryOfSubAccountsFuturesAccountV2 must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetSummaryOfSubAccountsFuturesAccountV2 should decode every field")
				return
			}
			assert.NotNil(t, result, "GetSummaryOfSubAccountsFuturesAccountV2 should return a summary")
		})
	}
}

func TestGetSummaryOfSubAccountMarginAccount(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSummaryOfSubAccountMarginAccount(t.Context())
	require.NoError(t, err, "GetSummaryOfSubAccountMarginAccount must not error")
	if mockTests {
		exp := &SubAccountMarginAccountSummaryResponse{
			TotalAssetOfBTC:     4.33333333,
			TotalLiabilityOfBTC: 2.11111112,
			TotalNetAssetOfBTC:  2.22222221,
			SubAccountList: []SubAccountMarginSummary{
				{
					Email:               "sub123@example.com",
					TotalAssetOfBTC:     2.11111111,
					TotalLiabilityOfBTC: 1.11111111,
					TotalNetAssetOfBTC:  1,
				},
			},
		}
		assert.Equal(t, exp, result, "GetSummaryOfSubAccountMarginAccount should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSummaryOfSubAccountMarginAccount should return a summary")
}

func TestMarginTransferForSubAccount(t *testing.T) {
	t.Parallel()
	testSubAccountWalletTransfer(t, e.MarginTransferForSubAccount, &SubAccountTransferResponse{
		TransactionID: "2966662590",
	})
}

func TestGetSubAccountAssetsV3(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAccountAssetsV3(t.Context(), "")
	require.ErrorIs(t, err, errValidEmailRequired, "GetSubAccountAssetsV3 must reject an empty email")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSubAccountAssetsV3(t.Context(), "someone@example.com")
	require.NoError(t, err, "GetSubAccountAssetsV3 must not error")
	if mockTests {
		exp := &SubAccountAssetsResponse{
			Balances: []SubAccountBalance{
				{
					Freeze:      1,
					Withdrawing: 2,
					Asset:       currency.ADA,
					Free:        11467.6399,
					Locked:      3,
				},
			},
		}
		assert.Equal(t, exp, result, "GetSubAccountAssetsV3 should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSubAccountAssetsV3 should return assets")
}

func TestGetSubAccountAssets(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAccountAssets(t.Context(), "email_address")
	require.ErrorIs(t, err, errValidEmailRequired, "GetSubAccountAssets must reject an invalid email")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSubAccountAssets(t.Context(), "address@example.com")
	require.NoError(t, err, "GetSubAccountAssets must not error")
	if mockTests {
		exp := &SubAccountAssetsResponse{
			Balances: []SubAccountBalance{
				{
					Freeze:      1,
					Withdrawing: 2,
					Asset:       currency.ADA,
					Free:        10000,
					Locked:      3,
				},
			},
		}
		assert.Equal(t, exp, result, "GetSubAccountAssets should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSubAccountAssets should return assets")
}

func TestGetSubAccountFuturesAssetTransferHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAccountFuturesAssetTransferHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSubAccountFuturesAssetTransferHistory must reject a nil request")
	_, err = e.GetSubAccountFuturesAssetTransferHistory(t.Context(), &SubAccountFuturesAssetTransferHistoryRequest{FuturesType: 2})
	require.ErrorIs(t, err, errValidEmailRequired, "GetSubAccountFuturesAssetTransferHistory must reject an empty email")
	_, err = e.GetSubAccountFuturesAssetTransferHistory(t.Context(), &SubAccountFuturesAssetTransferHistoryRequest{Email: "subaccount@example.com"})
	require.ErrorIs(t, err, errInvalidFuturesType, "GetSubAccountFuturesAssetTransferHistory must reject an unset futures type")
	startTime, endTime := getTime()
	_, err = e.GetSubAccountFuturesAssetTransferHistory(t.Context(), &SubAccountFuturesAssetTransferHistoryRequest{Email: "subaccount@example.com", FuturesType: 1, StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSubAccountFuturesAssetTransferHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSubAccountFuturesAssetTransferHistory(t.Context(), &SubAccountFuturesAssetTransferHistoryRequest{
		Email:       "subaccount@example.com",
		FuturesType: 2,
		StartTime:   startTime,
		EndTime:     endTime,
		Page:        1,
		Limit:       50,
	})
	require.NoError(t, err, "GetSubAccountFuturesAssetTransferHistory must not error")
	if mockTests {
		exp := &SubAccountFuturesAssetTransferHistoryResponse{
			Success:     true,
			FuturesType: 2,
			Transfers: []SubAccountFuturesAssetTransfer{
				{
					From:          "aaa@example.com",
					To:            "subaccount@example.com",
					Asset:         currency.BTC,
					Quantity:      1,
					TransactionID: 11897001102,
					Time:          types.Time(time.Unix(1744110000, 0)),
				},
			},
		}
		assert.Equal(t, exp, result, "GetSubAccountFuturesAssetTransferHistory should decode every field")
		return
	}
	assert.True(t, result.Success, "GetSubAccountFuturesAssetTransferHistory should succeed")
}

func TestSubAccountFuturesAssetTransfer(t *testing.T) {
	t.Parallel()
	_, err := e.SubAccountFuturesAssetTransfer(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SubAccountFuturesAssetTransfer must reject a nil request")
	req := &SubAccountFuturesAssetTransferRequest{FromEmail: "from_someone", ToEmail: "to_someone@example.com", FuturesType: 1, Asset: currency.USDT, Amount: 0.1}
	_, err = e.SubAccountFuturesAssetTransfer(t.Context(), req)
	require.ErrorIs(t, err, errValidEmailRequired, "SubAccountFuturesAssetTransfer must reject an invalid from email")
	req.FromEmail, req.ToEmail = "from_someone@example.com", "to_someone"
	_, err = e.SubAccountFuturesAssetTransfer(t.Context(), req)
	require.ErrorIs(t, err, errValidEmailRequired, "SubAccountFuturesAssetTransfer must reject an invalid to email")
	req.ToEmail, req.FuturesType = "to_someone@example.com", 0
	_, err = e.SubAccountFuturesAssetTransfer(t.Context(), req)
	require.ErrorIs(t, err, errInvalidFuturesType, "SubAccountFuturesAssetTransfer must reject an unset futures type")
	req.FuturesType, req.Asset = 1, currency.EMPTYCODE
	_, err = e.SubAccountFuturesAssetTransfer(t.Context(), req)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "SubAccountFuturesAssetTransfer must reject an empty asset")
	req.Asset, req.Amount = currency.USDT, 0
	_, err = e.SubAccountFuturesAssetTransfer(t.Context(), req)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "SubAccountFuturesAssetTransfer must reject a zero amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	req.Amount = 0.1
	result, err := e.SubAccountFuturesAssetTransfer(t.Context(), req)
	require.NoError(t, err, "SubAccountFuturesAssetTransfer must not error")
	if mockTests {
		exp := &SubAccountFuturesAssetTransferResponse{
			Success:       true,
			TransactionID: "2934662589",
		}
		assert.Equal(t, exp, result, "SubAccountFuturesAssetTransfer should decode every field")
		return
	}
	assert.True(t, result.Success, "SubAccountFuturesAssetTransfer should succeed")
}

func TestGetSubAccountSpotAssetsSummary(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSubAccountSpotAssetsSummary(t.Context(), "the_address@example.com", 1, 10)
	require.NoError(t, err, "GetSubAccountSpotAssetsSummary must not error")
	if mockTests {
		exp := &SubAccountSpotAssetsSummaryResponse{
			TotalCount:              1,
			MasterAccountTotalAsset: 0.23231201,
			SpotSubUserAssetBTCVoList: []SubAccountSpotAssetBTC{
				{
					Email:      "the_address@example.com",
					TotalAsset: 9999,
				},
			},
		}
		assert.Equal(t, exp, result, "GetSubAccountSpotAssetsSummary should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSubAccountSpotAssetsSummary should return a summary")
}

func TestGetSubAccountSpotAssetTransferHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAccountSpotAssetTransferHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSubAccountSpotAssetTransferHistory must reject a nil request")
	_, err = e.GetSubAccountSpotAssetTransferHistory(t.Context(), &SubAccountSpotAssetTransferHistoryRequest{FromEmail: "aaa@example.com", ToEmail: "bbb@example.com"})
	require.ErrorIs(t, err, errFromAndToEmailBothSet, "GetSubAccountSpotAssetTransferHistory must reject both emails")
	startTime, endTime := getTime()
	_, err = e.GetSubAccountSpotAssetTransferHistory(t.Context(), &SubAccountSpotAssetTransferHistoryRequest{StartTime: endTime, EndTime: startTime, Limit: 10})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSubAccountSpotAssetTransferHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *SubAccountSpotAssetTransferHistoryRequest
		exp  []SubAccountSpotAssetTransfer
	}{
		{
			name: "from", req: &SubAccountSpotAssetTransferHistoryRequest{FromEmail: "aaa@example.com", StartTime: startTime, EndTime: endTime, Page: 1, Limit: 10},
			exp: []SubAccountSpotAssetTransfer{
				{
					From:          "aaa@example.com",
					To:            "bbb@example.com",
					Asset:         currency.BTC,
					Quantity:      10,
					Status:        "SUCCESS",
					TransactionID: 6489943656,
					Time:          types.Time(time.Unix(1744110000, 0)),
				},
			},
		},
		{
			name: "to", req: &SubAccountSpotAssetTransferHistoryRequest{ToEmail: "bbb@example.com", Limit: 10},
			exp: []SubAccountSpotAssetTransfer{
				{
					From:          "ccc@example.com",
					To:            "bbb@example.com",
					Asset:         currency.ETH,
					Quantity:      2,
					Status:        "PROCESS",
					TransactionID: 6489943657,
					Time:          types.Time(time.Unix(1744120000, 0)),
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetSubAccountSpotAssetTransferHistory(t.Context(), tc.req)
			require.NoError(t, err, "GetSubAccountSpotAssetTransferHistory must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetSubAccountSpotAssetTransferHistory should decode every field")
				return
			}
			assert.NotNil(t, result, "GetSubAccountSpotAssetTransferHistory should return transfers")
		})
	}
}

func TestGetUniversalTransferHistoryForMasterAccount(t *testing.T) {
	t.Parallel()
	_, err := e.GetUniversalTransferHistoryForMasterAccount(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUniversalTransferHistoryForMasterAccount must reject a nil request")
	_, err = e.GetUniversalTransferHistoryForMasterAccount(t.Context(), &SubAccountUniversalTransferHistoryRequest{FromEmail: "abctest@example.com", ToEmail: "deftest@example.com"})
	require.ErrorIs(t, err, errFromAndToEmailBothSet, "GetUniversalTransferHistoryForMasterAccount must reject both emails")
	startTime, endTime := getTime()
	_, err = e.GetUniversalTransferHistoryForMasterAccount(t.Context(), &SubAccountUniversalTransferHistoryRequest{StartTime: endTime, EndTime: startTime, Limit: 10})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUniversalTransferHistoryForMasterAccount must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *SubAccountUniversalTransferHistoryRequest
		exp  *SubAccountUniversalTransferHistoryResponse
	}{
		{
			name: "from", req: &SubAccountUniversalTransferHistoryRequest{FromEmail: "abctest@example.com", ClientTransactionID: "test", StartTime: startTime, EndTime: endTime, Page: 1, Limit: 10},
			exp: &SubAccountUniversalTransferHistoryResponse{
				Result: []SubAccountUniversalTransfer{
					{
						TransactionID:       92275823339,
						FromEmail:           "abctest@example.com",
						ToEmail:             "deftest@example.com",
						Asset:               currency.BNB,
						Amount:              0.01,
						CreateTimeStamp:     types.Time(time.Unix(1744110000, 0)),
						FromAccountType:     "USDT_FUTURE",
						ToAccountType:       "SPOT",
						Status:              "SUCCESS",
						ClientTransactionID: "test",
					},
				},
				TotalCount: 1,
			},
		},
		{
			name: "to", req: &SubAccountUniversalTransferHistoryRequest{ToEmail: "deftest@example.com", Limit: 10},
			exp: &SubAccountUniversalTransferHistoryResponse{
				Result: []SubAccountUniversalTransfer{
					{
						TransactionID:       92275823340,
						FromEmail:           "ghitest@example.com",
						ToEmail:             "deftest@example.com",
						Asset:               currency.USDT,
						Amount:              25,
						CreateTimeStamp:     types.Time(time.Unix(1744120000, 0)),
						FromAccountType:     "SPOT",
						ToAccountType:       "COIN_FUTURE",
						Status:              "SUCCESS",
						ClientTransactionID: "test2",
					},
				},
				TotalCount: 1,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetUniversalTransferHistoryForMasterAccount(t.Context(), tc.req)
			require.NoError(t, err, "GetUniversalTransferHistoryForMasterAccount must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetUniversalTransferHistoryForMasterAccount should decode every field")
				return
			}
			assert.NotNil(t, result, "GetUniversalTransferHistoryForMasterAccount should return transfers")
		})
	}
}

func TestUniversalTransferForMasterAccount(t *testing.T) {
	t.Parallel()
	_, err := e.UniversalTransferForMasterAccount(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "UniversalTransferForMasterAccount must reject a nil request")
	req := &SubAccountUniversalTransferRequest{ClientTransactionID: "transaction-id"}
	_, err = e.UniversalTransferForMasterAccount(t.Context(), req)
	require.ErrorIs(t, err, errInvalidAccountType, "UniversalTransferForMasterAccount must reject an empty from account type")
	req.FromAccountType = "ISOLATED_MARGIN"
	_, err = e.UniversalTransferForMasterAccount(t.Context(), req)
	require.ErrorIs(t, err, errInvalidAccountType, "UniversalTransferForMasterAccount must reject an empty to account type")
	req.ToAccountType = "SPOT"
	_, err = e.UniversalTransferForMasterAccount(t.Context(), req)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "UniversalTransferForMasterAccount must reject an empty asset")
	req.Asset = currency.BTC
	_, err = e.UniversalTransferForMasterAccount(t.Context(), req)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "UniversalTransferForMasterAccount must reject a zero amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name string
		req  *SubAccountUniversalTransferRequest
		exp  *SubAccountUniversalTransferResponse
	}{
		{
			name: "between sub-accounts",
			req: &SubAccountUniversalTransferRequest{
				FromEmail:           "source@example.com",
				ToEmail:             "destination@example.com",
				FromAccountType:     "SPOT",
				ToAccountType:       "SPOT",
				ClientTransactionID: "transaction-id",
				Asset:               currency.BTC,
				Amount:              0.0003,
			},
			exp: &SubAccountUniversalTransferResponse{
				TransactionID:       11945860693,
				ClientTransactionID: "transaction-id",
			},
		},
		{
			name: "isolated margin to master",
			req: &SubAccountUniversalTransferRequest{
				FromEmail:       "source@example.com",
				FromAccountType: "ISOLATED_MARGIN",
				ToAccountType:   "SPOT",
				Symbol:          currency.NewBTCUSDT(),
				Asset:           currency.BTC,
				Amount:          0.0003,
			},
			exp: &SubAccountUniversalTransferResponse{
				TransactionID: 11945860694,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.UniversalTransferForMasterAccount(t.Context(), tc.req)
			require.NoError(t, err, "UniversalTransferForMasterAccount must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "UniversalTransferForMasterAccount should decode every field")
				return
			}
			assert.NotZero(t, result.TransactionID, "UniversalTransferForMasterAccount should return a transaction ID")
		})
	}
}

func TestSubAccountTransferHistoryForSubAccount(t *testing.T) {
	t.Parallel()
	_, err := e.SubAccountTransferHistoryForSubAccount(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SubAccountTransferHistoryForSubAccount must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.SubAccountTransferHistoryForSubAccount(t.Context(), &SubAccountTransferHistoryRequest{Asset: currency.LTC, Type: 2, StartTime: endTime, EndTime: startTime, ReturnFailHistory: true})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "SubAccountTransferHistoryForSubAccount must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.SubAccountTransferHistoryForSubAccount(t.Context(), &SubAccountTransferHistoryRequest{
		Asset:             currency.LTC,
		Type:              2,
		StartTime:         startTime,
		EndTime:           endTime,
		Limit:             10,
		ReturnFailHistory: true,
	})
	require.NoError(t, err, "SubAccountTransferHistoryForSubAccount must not error")
	if mockTests {
		exp := []SubAccountTransferHistoryRecord{
			{
				CounterParty:    "master",
				Email:           "master@example.com",
				Type:            2,
				Asset:           currency.LTC,
				Quantity:        1,
				FromAccountType: "SPOT",
				ToAccountType:   "SPOT",
				Status:          "SUCCESS",
				TransactionID:   11798835829,
				Time:            types.Time(time.Unix(1744110000, 0)),
			},
			{
				CounterParty:    "subAccount",
				Email:           "sub@example.com",
				Type:            2,
				Asset:           currency.LTC,
				Quantity:        0.5,
				FromAccountType: "SPOT",
				ToAccountType:   "USDT_FUTURE",
				Status:          "FAILURE",
				TransactionID:   11798835830,
				Time:            types.Time(time.Unix(1744120000, 0)),
			},
		}
		assert.Equal(t, exp, result, "SubAccountTransferHistoryForSubAccount should decode every field")
		return
	}
	assert.NotNil(t, result, "SubAccountTransferHistoryForSubAccount should return transfers")
}

func TestFromSubAccountTransferToMaster(t *testing.T) {
	t.Parallel()
	_, err := e.FromSubAccountTransferToMaster(t.Context(), currency.EMPTYCODE, 0.1)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "FromSubAccountTransferToMaster must reject an empty asset")
	_, err = e.FromSubAccountTransferToMaster(t.Context(), currency.LTC, 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "FromSubAccountTransferToMaster must reject a zero amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.FromSubAccountTransferToMaster(t.Context(), currency.LTC, 0.1)
	require.NoError(t, err, "FromSubAccountTransferToMaster must not error")
	if mockTests {
		exp := &SubAccountTransferResponse{
			TransactionID: "2966662591",
		}
		assert.Equal(t, exp, result, "FromSubAccountTransferToMaster should decode every field")
		return
	}
	assert.NotEmpty(t, result.TransactionID, "FromSubAccountTransferToMaster should return a transaction ID")
}

func TestTransferToSubAccountOfSameMaster(t *testing.T) {
	t.Parallel()
	_, err := e.TransferToSubAccountOfSameMaster(t.Context(), "thrasher", currency.ETH, 10)
	require.ErrorIs(t, err, errValidEmailRequired, "TransferToSubAccountOfSameMaster must reject an invalid email")
	_, err = e.TransferToSubAccountOfSameMaster(t.Context(), "toEmail@example.com", currency.EMPTYCODE, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "TransferToSubAccountOfSameMaster must reject an empty asset")
	_, err = e.TransferToSubAccountOfSameMaster(t.Context(), "toEmail@example.com", currency.ETH, 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "TransferToSubAccountOfSameMaster must reject a zero amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.TransferToSubAccountOfSameMaster(t.Context(), "toEmail@example.com", currency.ETH, 10)
	require.NoError(t, err, "TransferToSubAccountOfSameMaster must not error")
	if mockTests {
		exp := &SubAccountTransferResponse{
			TransactionID: "2966662592",
		}
		assert.Equal(t, exp, result, "TransferToSubAccountOfSameMaster should decode every field")
		return
	}
	assert.NotEmpty(t, result.TransactionID, "TransferToSubAccountOfSameMaster should return a transaction ID")
}

func TestDepositAssetsIntoTheManagedSubAccount(t *testing.T) {
	t.Parallel()
	_, err := e.DepositAssetsIntoTheManagedSubAccount(t.Context(), "toemail", currency.BTC, 0.0001)
	require.ErrorIs(t, err, errValidEmailRequired, "DepositAssetsIntoTheManagedSubAccount must reject an invalid email")
	_, err = e.DepositAssetsIntoTheManagedSubAccount(t.Context(), "toemail@example.com", currency.EMPTYCODE, 0.0001)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "DepositAssetsIntoTheManagedSubAccount must reject an empty asset")
	_, err = e.DepositAssetsIntoTheManagedSubAccount(t.Context(), "toemail@example.com", currency.BTC, 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "DepositAssetsIntoTheManagedSubAccount must reject a zero amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.DepositAssetsIntoTheManagedSubAccount(t.Context(), "toemail@example.com", currency.BTC, 0.0001)
	require.NoError(t, err, "DepositAssetsIntoTheManagedSubAccount must not error")
	if mockTests {
		exp := &ManagedSubAccountTransferResponse{
			TransactionID: 66157362489,
		}
		assert.Equal(t, exp, result, "DepositAssetsIntoTheManagedSubAccount should decode every field")
		return
	}
	assert.NotZero(t, result.TransactionID, "DepositAssetsIntoTheManagedSubAccount should return a transaction ID")
}

func TestGetManagedSubAccountDepositAddress(t *testing.T) {
	t.Parallel()
	testSubAccountDepositAddress(t, e.GetManagedSubAccountDepositAddress, []subAccountDepositAddressCase{
		{
			req: &SubAccountDepositAddressRequest{Email: "destination@example.com", Coin: currency.ETH},
			exp: &SubAccountDepositAddressResponse{
				Address: "0x206c22d833bb0bb2102da6b7c7d4c3eb14bcf73d",
				Coin:    currency.ETH,
				URL:     "https://etherscan.io/address/0x206c22d833bb0bb2102da6b7c7d4c3eb14bcf73d",
			},
		},
		{
			req: &SubAccountDepositAddressRequest{Email: "destination@example.com", Coin: currency.XRP, Network: "XRP"},
			exp: &SubAccountDepositAddressResponse{
				Address: "rEb8TK3gBgk5auZkwc6sHnwrGVJH8DuaLh",
				Coin:    currency.XRP,
				Tag:     "101764891",
				URL:     "https://xrpscan.com/account/rEb8TK3gBgk5auZkwc6sHnwrGVJH8DuaLh",
			},
		},
	})
}

func TestGetManagedSubAccountAssetsDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetManagedSubAccountAssetsDetails(t.Context(), "emailaddress")
	require.ErrorIs(t, err, errValidEmailRequired, "GetManagedSubAccountAssetsDetails must reject an invalid email")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetManagedSubAccountAssetsDetails(t.Context(), "emailaddress@example.com")
	require.NoError(t, err, "GetManagedSubAccountAssetsDetails must not error")
	if mockTests {
		exp := []ManagedSubAccountAsset{
			{
				Coin:             currency.NewCode("INJ"),
				Name:             "Injective Protocol",
				TotalBalance:     12.5,
				AvailableBalance: 10,
				InOrder:          2.5,
				BTCValue:         0.0021,
			},
		}
		assert.Equal(t, exp, result, "GetManagedSubAccountAssetsDetails should decode every field")
		return
	}
	assert.NotNil(t, result, "GetManagedSubAccountAssetsDetails should return assets")
}

func TestGetManagedSubAccountFuturesAssetDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetManagedSubAccountFuturesAssetDetails(t.Context(), "address", "")
	require.ErrorIs(t, err, errValidEmailRequired, "GetManagedSubAccountFuturesAssetDetails must reject an invalid email")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name        string
		accountType string
		exp         *ManagedSubAccountFuturesAssetDetailsResponse
	}{
		{
			name: "default",
			exp: &ManagedSubAccountFuturesAssetDetailsResponse{
				Code:    "200",
				Message: "OK",
				SnapshotVos: []ManagedSubAccountFuturesSnapshot{
					{
						Type:       "FUTURES",
						UpdateTime: types.Time(time.Unix(1744110000, 0)),
						Data: ManagedSubAccountFuturesSnapshotData{
							Assets: []ManagedSubAccountFuturesAsset{
								{
									Asset:         currency.USDT,
									MarginBalance: 100.5,
									WalletBalance: 120,
								},
							},
							Position: []ManagedSubAccountFuturesPosition{
								{
									Symbol:         "BTCUSDT",
									EntryPrice:     17000,
									MarkPrice:      17100.5,
									PositionAmount: 0.0001,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "coin-margined", accountType: "COIN_FUTURE",
			exp: &ManagedSubAccountFuturesAssetDetailsResponse{
				Code:    "200",
				Message: "OK",
				SnapshotVos: []ManagedSubAccountFuturesSnapshot{
					{
						Type:       "FUTURES",
						UpdateTime: types.Time(time.Unix(1744110000, 0)),
						Data: ManagedSubAccountFuturesSnapshotData{
							Assets: []ManagedSubAccountFuturesAsset{
								{
									Asset:         currency.BTC,
									MarginBalance: 0.5,
									WalletBalance: 0.49,
								},
							},
							Position: []ManagedSubAccountFuturesPosition{
								{
									Symbol:         "BTCUSD_PERP",
									EntryPrice:     84000,
									MarkPrice:      84500.1,
									PositionAmount: 10,
								},
							},
						},
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetManagedSubAccountFuturesAssetDetails(t.Context(), "address@example.com", tc.accountType)
			require.NoError(t, err, "GetManagedSubAccountFuturesAssetDetails must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetManagedSubAccountFuturesAssetDetails should decode every field")
				return
			}
			assert.NotNil(t, result, "GetManagedSubAccountFuturesAssetDetails should return futures assets")
		})
	}
}

func TestGetManagedSubAccountList(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetManagedSubAccountList(t.Context(), "address@example.com", 1, 10)
	require.NoError(t, err, "GetManagedSubAccountList must not error")
	if mockTests {
		exp := &ManagedSubAccountListResponse{
			Total: 1,
			ManagerSubUserInfoVoList: []ManagedSubAccountInfo{
				{
					RootUserID:               1000138475670,
					ManagersubUserID:         1000137842513,
					BindParentUserID:         1000138475669,
					Email:                    "address@example.com",
					InsertTimeStamp:          types.Time(time.Unix(1678435149, 0)),
					BindParentEmail:          "trader@example.com",
					IsSubUserEnabled:         true,
					IsUserActive:             true,
					IsMarginEnabled:          true,
					IsFutureEnabled:          true,
					IsSignedLVTRiskAgreement: true,
				},
			},
		}
		assert.Equal(t, exp, result, "GetManagedSubAccountList should decode every field")
		return
	}
	assert.NotNil(t, result, "GetManagedSubAccountList should return managed sub-accounts")
}

func TestGetManagedSubAccountMarginAssetDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetManagedSubAccountMarginAssetDetails(t.Context(), "address", "")
	require.ErrorIs(t, err, errValidEmailRequired, "GetManagedSubAccountMarginAssetDetails must reject an invalid email")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetManagedSubAccountMarginAssetDetails(t.Context(), "address@example.com", "MARGIN")
	require.NoError(t, err, "GetManagedSubAccountMarginAssetDetails must not error")
	if mockTests {
		exp := &ManagedSubAccountMarginAssetDetailsResponse{
			MarginLevel:         999,
			TotalAssetOfBTC:     0.0015,
			TotalLiabilityOfBTC: 0.0000015,
			TotalNetAssetOfBTC:  0.0014985,
			UserAssets: []MarginAssetBalance{
				{
					Asset:    currency.NewCode("MATIC"),
					Borrowed: 1,
					Free:     10,
					Interest: 0.001,
					Locked:   2,
					NetAsset: 10.999,
				},
			},
		}
		assert.Equal(t, exp, result, "GetManagedSubAccountMarginAssetDetails should decode every field")
		return
	}
	assert.NotNil(t, result, "GetManagedSubAccountMarginAssetDetails should return the margin account")
}

func TestGetManagedSubAccountSnapshot(t *testing.T) {
	t.Parallel()
	_, err := e.GetManagedSubAccountSnapshot(t.Context(), "address", &AccountSnapshotRequest{Type: "SPOT"})
	require.ErrorIs(t, err, errValidEmailRequired, "GetManagedSubAccountSnapshot must reject an invalid email")
	_, err = e.GetManagedSubAccountSnapshot(t.Context(), "address@example.com", nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetManagedSubAccountSnapshot must reject a nil request")
	_, err = e.GetManagedSubAccountSnapshot(t.Context(), "address@example.com", &AccountSnapshotRequest{})
	require.ErrorIs(t, err, errSnapshotTypeRequired, "GetManagedSubAccountSnapshot must reject an empty type")
	startTime, endTime := getTime()
	_, err = e.GetManagedSubAccountSnapshot(t.Context(), "address@example.com", &AccountSnapshotRequest{Type: "SPOT", StartTime: endTime, EndTime: startTime, Limit: 10})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetManagedSubAccountSnapshot must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, snapshotType := range []string{"SPOT", "MARGIN", "FUTURES"} {
		t.Run(snapshotType, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetManagedSubAccountSnapshot(t.Context(), "address@example.com", &AccountSnapshotRequest{Type: snapshotType, StartTime: startTime, EndTime: endTime, Limit: 10})
			require.NoError(t, err, "GetManagedSubAccountSnapshot must not error")
			if mockTests {
				assert.Equal(t, expAccountSnapshots[snapshotType], result, "GetManagedSubAccountSnapshot should decode every field")
				return
			}
			assert.NotNil(t, result, "GetManagedSubAccountSnapshot should return snapshots")
		})
	}
}

func TestGetManagedSubAccountTransferLogForInvestorMasterAccount(t *testing.T) {
	t.Parallel()
	testManagedSubAccountTransferLogForEmail(t, e.GetManagedSubAccountTransferLogForInvestorMasterAccount, &ManagedSubAccountTransferLogRequest{Page: 1, Limit: 100, Transfers: "TO", TransferFunctionAccountType: "SPOT"}, &ManagedSubAccountTransferLogResponse{
		ManagerSubTransferHistoryVos: []ManagedSubAccountTransfer{
			{
				FromEmail:       "manager@example.com",
				FromAccountType: "SPOT",
				ToEmail:         "address@example.com",
				ToAccountType:   "SPOT",
				Asset:           currency.BNB,
				Amount:          0.01,
				ScheduledData:   types.Time(time.Unix(1744156800, 0)),
				CreateTime:      types.Time(time.Unix(1744110000, 0)),
				Status:          "SUCCESS",
				TransactionID:   91077779,
			},
		},
		Count: 1,
	})
}

func TestGetManagedSubAccountTransferLogForTradingTeam(t *testing.T) {
	t.Parallel()
	testManagedSubAccountTransferLogForEmail(t, e.GetManagedSubAccountTransferLogForTradingTeam, &ManagedSubAccountTransferLogRequest{Page: 1, Limit: 10, Transfers: "FROM", TransferFunctionAccountType: "ISOLATED_MARGIN"}, &ManagedSubAccountTransferLogResponse{
		ManagerSubTransferHistoryVos: []ManagedSubAccountTransfer{
			{
				FromEmail:       "address@example.com",
				FromAccountType: "ISOLATED_MARGIN",
				ToEmail:         "manager@example.com",
				ToAccountType:   "ISOLATED_MARGIN",
				Asset:           currency.BNB,
				Amount:          0.01,
				ScheduledData:   types.Time(time.Unix(1744156800, 0)),
				CreateTime:      types.Time(time.Unix(1744110000, 0)),
				Status:          "SUCCESS",
				TransactionID:   91077779,
			},
		},
		Count: 1,
	})
}

func testManagedSubAccountTransferLogForEmail(t *testing.T, transferLog func(context.Context, string, *ManagedSubAccountTransferLogRequest) (*ManagedSubAccountTransferLogResponse, error), req *ManagedSubAccountTransferLogRequest, exp *ManagedSubAccountTransferLogResponse) {
	t.Helper()
	_, err := transferLog(t.Context(), "address.com", req)
	require.ErrorIs(t, err, errValidEmailRequired, "transfer log must reject an invalid email")
	testManagedSubAccountTransferLog(t, func(ctx context.Context, req *ManagedSubAccountTransferLogRequest) (*ManagedSubAccountTransferLogResponse, error) {
		return transferLog(ctx, "address@example.com", req)
	}, req, exp)
}

func TestGetManagedSubAccountTransferLog(t *testing.T) {
	t.Parallel()
	testManagedSubAccountTransferLog(t, e.GetManagedSubAccountTransferLog, &ManagedSubAccountTransferLogRequest{Page: 1, Limit: 10, TransferFunctionAccountType: "MARGIN"}, &ManagedSubAccountTransferLogResponse{
		ManagerSubTransferHistoryVos: []ManagedSubAccountTransfer{
			{
				FromEmail:       "address@example.com",
				FromAccountType: "MARGIN",
				ToEmail:         "manager@example.com",
				ToAccountType:   "MARGIN",
				Asset:           currency.BNB,
				Amount:          0.01,
				ScheduledData:   types.Time(time.Unix(1744156800, 0)),
				CreateTime:      types.Time(time.Unix(1744110000, 0)),
				Status:          "SUCCESS",
				TransactionID:   91077779,
			},
		},
		Count: 1,
	})
}

func testManagedSubAccountTransferLog(t *testing.T, transferLog func(context.Context, *ManagedSubAccountTransferLogRequest) (*ManagedSubAccountTransferLogResponse, error), req *ManagedSubAccountTransferLogRequest, exp *ManagedSubAccountTransferLogResponse) {
	t.Helper()
	_, err := transferLog(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "transfer log must reject a nil request")
	_, err = transferLog(t.Context(), &ManagedSubAccountTransferLogRequest{Page: 1, Limit: 10})
	require.ErrorIs(t, err, common.ErrDateUnset, "transfer log must reject an unset time window")
	startTime, endTime := getTime()
	_, err = transferLog(t.Context(), &ManagedSubAccountTransferLogRequest{StartTime: endTime, EndTime: startTime, Page: 1, Limit: 10})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "transfer log must reject a start after the end")
	_, err = transferLog(t.Context(), &ManagedSubAccountTransferLogRequest{StartTime: startTime, EndTime: endTime, Limit: 10})
	require.ErrorIs(t, err, errPageNumberRequired, "transfer log must reject an unset page")
	_, err = transferLog(t.Context(), &ManagedSubAccountTransferLogRequest{StartTime: startTime, EndTime: endTime, Page: 1})
	require.ErrorIs(t, err, errLimitNumberRequired, "transfer log must reject an unset limit")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	req.StartTime, req.EndTime = startTime, endTime
	result, err := transferLog(t.Context(), req)
	require.NoError(t, err, "transfer log must not error")
	if mockTests {
		assert.Equal(t, exp, result, "transfer log should decode every field")
		return
	}
	assert.NotNil(t, result, "transfer log should return transfers")
}

func TestWithdrawAssetsFromManagedSubAccount(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawAssetsFromManagedSubAccount(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WithdrawAssetsFromManagedSubAccount must reject a nil request")
	transferDate := time.UnixMilli(1744243200000)
	_, err = e.WithdrawAssetsFromManagedSubAccount(t.Context(), &ManagedSubAccountWithdrawRequest{FromEmail: "source", Asset: currency.BTC, Amount: 0.0000001, TransferDate: transferDate})
	require.ErrorIs(t, err, errValidEmailRequired, "WithdrawAssetsFromManagedSubAccount must reject an invalid email")
	_, err = e.WithdrawAssetsFromManagedSubAccount(t.Context(), &ManagedSubAccountWithdrawRequest{FromEmail: "source@example.com", Amount: 0.0000001, TransferDate: transferDate})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "WithdrawAssetsFromManagedSubAccount must reject an empty asset")
	_, err = e.WithdrawAssetsFromManagedSubAccount(t.Context(), &ManagedSubAccountWithdrawRequest{FromEmail: "source@example.com", Asset: currency.BTC, TransferDate: transferDate})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "WithdrawAssetsFromManagedSubAccount must reject a zero amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.WithdrawAssetsFromManagedSubAccount(t.Context(), &ManagedSubAccountWithdrawRequest{FromEmail: "source@example.com", Asset: currency.BTC, Amount: 0.0000001, TransferDate: transferDate})
	require.NoError(t, err, "WithdrawAssetsFromManagedSubAccount must not error")
	if mockTests {
		exp := &ManagedSubAccountTransferResponse{
			TransactionID: 66157362490,
		}
		assert.Equal(t, exp, result, "WithdrawAssetsFromManagedSubAccount should decode every field")
		return
	}
	assert.NotZero(t, result.TransactionID, "WithdrawAssetsFromManagedSubAccount should return a transaction ID")
}
