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

func TestGetAccountTradingAPIStatus(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAccountTradingAPIStatus(t.Context())
	require.NoError(t, err, "GetAccountTradingAPIStatus must not error")
	if mockTests {
		exp := &AccountAPITradingStatusResponse{
			Data: AccountAPITradingStatus{
				IsLocked:           true,
				PlannedRecoverTime: types.Time(time.Unix(1744194600, 0)),
				TriggerCondition: TradingAPITriggerCondition{
					GCR:  150,
					IFER: 150,
					UFR:  300,
				},
				UpdateTime: types.Time(time.UnixMilli(1744190254944)),
			},
		}
		assert.Equal(t, exp, result, "GetAccountTradingAPIStatus should decode every field")
		return
	}
	assert.NotZero(t, result.Data.UpdateTime, "GetAccountTradingAPIStatus should return an update time")
}

func TestGetUserAccountInfo(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUserAccountInfo(t.Context())
	require.NoError(t, err, "GetUserAccountInfo must not error")
	if mockTests {
		exp := &AccountInfoResponse{
			VIPLevel:                       2,
			IsMarginEnabled:                true,
			IsFutureEnabled:                true,
			IsOptionsEnabled:               true,
			IsPortfolioMarginRetailEnabled: true,
		}
		assert.Equal(t, exp, result, "GetUserAccountInfo should decode every field")
		return
	}
	assert.NotNil(t, result, "GetUserAccountInfo should return account info")
}

func TestGetAccountStatus(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAccountStatus(t.Context())
	require.NoError(t, err, "GetAccountStatus must not error")
	if mockTests {
		exp := &AccountStatusResponse{
			Data: "Normal",
		}
		assert.Equal(t, exp, result, "GetAccountStatus should decode every field")
		return
	}
	assert.NotEmpty(t, result.Data, "GetAccountStatus should return a status")
}

// expAccountSnapshots holds the snapshots the daily and managed sub-account snapshot fixtures record, by type
var expAccountSnapshots = map[string]*AccountSnapshotResponse{
	"SPOT": {
		Code:    200,
		Message: "OK",
		SnapshotVos: []AccountSnapshot{
			{
				Data: AccountSnapshotData{
					Balances: []AccountSnapshotBalance{
						{
							Asset:  currency.BTC,
							Free:   0.09905021,
							Locked: 0.001,
						},
						{
							Asset:  currency.USDT,
							Free:   1250.5,
							Locked: 10,
						},
					},
					TotalAssetOfBTC: 0.119427,
				},
				Type:       "spot",
				UpdateTime: types.Time(time.Unix(1744156799, 0)),
			},
		},
	},
	"MARGIN": {
		Code:    200,
		Message: "OK",
		SnapshotVos: []AccountSnapshot{
			{
				Data: AccountSnapshotData{
					TotalAssetOfBTC:     0.0027475,
					MarginLevel:         2748.02909813,
					TotalLiabilityOfBTC: 0.000001,
					TotalNetAssetOfBTC:  0.0027465,
					UserAssets: []MarginAssetBalance{
						{
							Asset:    currency.XRP,
							Borrowed: 0.5,
							Free:     1,
							Interest: 0.0001,
							Locked:   0.2,
							NetAsset: 0.6999,
						},
					},
				},
				Type:       "margin",
				UpdateTime: types.Time(time.Unix(1744156799, 0)),
			},
		},
	},
	"FUTURES": {
		Code:    200,
		Message: "OK",
		SnapshotVos: []AccountSnapshot{
			{
				Data: AccountSnapshotData{
					Assets: []AccountSnapshotFuturesAsset{
						{
							Asset:         currency.USDT,
							MarginBalance: 118.99782335,
							WalletBalance: 120.23811389,
						},
					},
					Position: []AccountSnapshotPosition{
						{
							EntryPrice:       7130.41,
							MarkPrice:        7257.66239673,
							PositionAmount:   0.01,
							Symbol:           "BTCUSDT",
							UnrealisedProfit: 1.27252396,
						},
					},
				},
				Type:       "futures",
				UpdateTime: types.Time(time.Unix(1744156799, 0)),
			},
		},
	},
}

func TestGetDailyAccountSnapshot(t *testing.T) {
	t.Parallel()
	_, err := e.GetDailyAccountSnapshot(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetDailyAccountSnapshot must reject a nil request")
	_, err = e.GetDailyAccountSnapshot(t.Context(), &AccountSnapshotRequest{})
	require.ErrorIs(t, err, errSnapshotTypeRequired, "GetDailyAccountSnapshot must reject an empty type")
	startTime, endTime := getTime()
	_, err = e.GetDailyAccountSnapshot(t.Context(), &AccountSnapshotRequest{Type: "SPOT", StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetDailyAccountSnapshot must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, snapshotType := range []string{"SPOT", "MARGIN", "FUTURES"} {
		t.Run(snapshotType, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetDailyAccountSnapshot(t.Context(), &AccountSnapshotRequest{Type: snapshotType, StartTime: startTime, EndTime: endTime, Limit: 7})
			require.NoError(t, err, "GetDailyAccountSnapshot must not error")
			if mockTests {
				assert.Equal(t, expAccountSnapshots[snapshotType], result, "GetDailyAccountSnapshot should decode every field")
				return
			}
			assert.NotEmpty(t, result.SnapshotVos, "GetDailyAccountSnapshot should return snapshots")
		})
	}
}

func TestDisableFastWithdrawalSwitch(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	assert.NoError(t, e.DisableFastWithdrawalSwitch(t.Context()), "DisableFastWithdrawalSwitch should not error")
}

func TestEnableFastWithdrawalSwitch(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	assert.NoError(t, e.EnableFastWithdrawalSwitch(t.Context()), "EnableFastWithdrawalSwitch should not error")
}

func TestGetAPIKeyPermission(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAPIKeyPermission(t.Context())
	require.NoError(t, err, "GetAPIKeyPermission must not error")
	if mockTests {
		exp := &APIKeyPermissionResponse{
			IPRestrict:                   true,
			CreateTime:                   types.Time(time.Unix(1623840271, 0)),
			EnableReading:                true,
			EnableWithdrawals:            true,
			EnableInternalTransfer:       true,
			EnableMargin:                 true,
			EnableFutures:                true,
			PermitsUniversalTransfer:     true,
			EnableVanillaOptions:         true,
			EnableFixAPITrade:            true,
			EnableFixReadOnly:            true,
			EnableSpotAndMarginTrading:   true,
			EnablePortfolioMarginTrading: true,
		}
		assert.Equal(t, exp, result, "GetAPIKeyPermission should decode every field")
		return
	}
	assert.NotZero(t, result.CreateTime, "GetAPIKeyPermission should return a create time")
}

func TestGetAssetDetail(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		ccy  currency.Code
		exp  map[currency.Code]AssetDetail
	}{
		{
			name: "all",
			exp: map[currency.Code]AssetDetail{
				currency.NewCode("CTR"): {
					MinimumWithdrawAmount: 70,
					WithdrawFee:           35,
					WithdrawStatus:        true,
					DepositTip:            "Delisted, Deposit Suspended",
				},
				currency.NewCode("SKY"): {
					MinimumWithdrawAmount: 0.02,
					DepositStatus:         true,
					WithdrawFee:           0.01,
					WithdrawStatus:        true,
				},
			},
		},
		{
			name: "BTC", ccy: currency.BTC,
			exp: map[currency.Code]AssetDetail{
				currency.BTC: {
					MinimumWithdrawAmount: 0.0001,
					DepositStatus:         true,
					WithdrawFee:           0.0000025,
					WithdrawStatus:        true,
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetAssetDetail(t.Context(), tc.ccy)
			require.NoError(t, err, "GetAssetDetail must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetAssetDetail should decode every field")
				return
			}
			assert.NotEmpty(t, result, "GetAssetDetail should return asset details")
		})
	}
}

func TestGetAssetDividendRecord(t *testing.T) {
	t.Parallel()
	_, err := e.GetAssetDividendRecord(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAssetDividendRecord must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetAssetDividendRecord(t.Context(), &AssetDividendRecordRequest{Asset: currency.BTC, StartTime: endTime, EndTime: startTime, Limit: 1})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetAssetDividendRecord must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAssetDividendRecord(t.Context(), &AssetDividendRecordRequest{Asset: currency.BTC, StartTime: startTime, EndTime: endTime, Limit: 100})
	require.NoError(t, err, "GetAssetDividendRecord must not error")
	if mockTests {
		exp := &AssetDividendRecordResponse{
			Rows: []AssetDividend{
				{
					ID:            1637366104,
					Amount:        0.0001,
					Asset:         currency.BTC,
					DividendTime:  types.Time(time.Unix(1744110000, 0)),
					EnglishInfo:   "BTC distribution",
					TransactionID: 2968885920,
					Direction:     1,
				},
			},
			Total: 1,
		}
		assert.Equal(t, exp, result, "GetAssetDividendRecord should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAssetDividendRecord should return records")
}

func TestGetDustLog(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetDustLog(t.Context(), "MARGIN", endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetDustLog must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetDustLog(t.Context(), "MARGIN", startTime, endTime)
	require.NoError(t, err, "GetDustLog must not error")
	if mockTests {
		exp := &DustLogResponse{
			Total: 1,
			UserAssetDribblets: []DustLogDribblet{
				{
					OperateTime:              types.Time(time.Unix(1744110000, 0)),
					TotalTransferedAmount:    0.00132256,
					TotalServiceChargeAmount: 0.00002699,
					TransactionID:            45178372831,
					UserAssetDribbletDetails: []DustLogDribbletDetail{
						{
							TransactionID:       4359321,
							ServiceChargeAmount: 0.00002699,
							Amount:              0.0009,
							OperateTime:         types.Time(time.Unix(1744110000, 0)),
							TransferedAmount:    0.00132256,
							FromAsset:           currency.USDT,
							TargetAsset:         currency.BNB,
						},
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetDustLog should decode every field")
		return
	}
	assert.NotNil(t, result, "GetDustLog should return a dust log")
}

func TestDustTransfer(t *testing.T) {
	t.Parallel()
	_, err := e.DustTransfer(t.Context(), nil, "SPOT")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "DustTransfer must reject no assets")
	_, err = e.DustTransfer(t.Context(), []currency.Code{currency.BTC, currency.EMPTYCODE}, "SPOT")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "DustTransfer must reject an empty asset")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.DustTransfer(t.Context(), []currency.Code{currency.BTC, currency.USDT}, "SPOT")
	require.NoError(t, err, "DustTransfer must not error")
	if mockTests {
		exp := &DustTransferResponse{
			TotalServiceCharge: 0.00002,
			TotalTransfered:    0.00098,
			TransferResult: []DustTransferResult{
				{
					Amount:              0.000001,
					FromAsset:           currency.BTC,
					OperateTime:         types.Time(time.Unix(1744110000, 0)),
					ServiceChargeAmount: 0.00001,
					TransactionID:       2970932918,
					TransferedAmount:    0.00049,
				},
				{
					Amount:              0.3,
					FromAsset:           currency.USDT,
					OperateTime:         types.Time(time.Unix(1744110000, 0)),
					ServiceChargeAmount: 0.00001,
					TransactionID:       2970932919,
					TransferedAmount:    0.00049,
				},
			},
		}
		assert.Equal(t, exp, result, "DustTransfer should decode every field")
		return
	}
	assert.NotEmpty(t, result.TransferResult, "DustTransfer should return transfer results")
}

func TestGetFundingAssets(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFundingAssets(t.Context(), currency.BTC, true)
	require.NoError(t, err, "GetFundingAssets must not error")
	if mockTests {
		exp := []FundingAsset{
			{
				Asset:        currency.BTC,
				Free:         0.5,
				Locked:       0.1,
				Freeze:       0.01,
				Withdrawing:  0.02,
				BTCValuation: 0.63,
			},
		}
		assert.Equal(t, exp, result, "GetFundingAssets should decode every field")
		return
	}
	assert.NotNil(t, result, "GetFundingAssets should return funding assets")
}

func TestGetAssetsThatCanBeConvertedIntoBNB(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAssetsThatCanBeConvertedIntoBNB(t.Context(), "SPOT")
	require.NoError(t, err, "GetAssetsThatCanBeConvertedIntoBNB must not error")
	if mockTests {
		exp := &AssetsConvertibleToBNBResponse{
			Details: []AssetConvertibleToBNB{
				{
					Asset:            currency.ADA,
					AssetFullName:    "ADA",
					AmountFree:       6.21,
					ToBTC:            0.00016848,
					ToBNB:            0.01777302,
					ToBNBOffExchange: 0.01741756,
					Exchange:         0.00035546,
				},
			},
			TotalTransferBTC:   0.00016848,
			TotalTransferBNB:   0.01777302,
			DribbletPercentage: 0.02,
		}
		assert.Equal(t, exp, result, "GetAssetsThatCanBeConvertedIntoBNB should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAssetsThatCanBeConvertedIntoBNB should return convertible assets")
}

func TestGetCloudMiningPaymentAndRefundHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetCloudMiningPaymentAndRefundHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetCloudMiningPaymentAndRefundHistory must reject a nil request")
	_, err = e.GetCloudMiningPaymentAndRefundHistory(t.Context(), &CloudMiningHistoryRequest{})
	require.ErrorIs(t, err, common.ErrDateUnset, "GetCloudMiningPaymentAndRefundHistory must reject an unset time window")
	startTime, endTime := getTime()
	_, err = e.GetCloudMiningPaymentAndRefundHistory(t.Context(), &CloudMiningHistoryRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetCloudMiningPaymentAndRefundHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCloudMiningPaymentAndRefundHistory(t.Context(), &CloudMiningHistoryRequest{
		TransactionID:       1232313,
		ClientTransactionID: "1234",
		Asset:               currency.BTC,
		StartTime:           startTime,
		EndTime:             endTime,
	})
	require.NoError(t, err, "GetCloudMiningPaymentAndRefundHistory must not error")
	if mockTests {
		exp := &CloudMiningHistoryResponse{
			Total: 1,
			Rows: []CloudMiningTransfer{
				{
					CreateTime:    types.Time(time.Unix(1744110000, 0)),
					TransactionID: 1232313,
					Type:          248,
					Asset:         currency.BTC,
					Amount:        0.0025,
					Status:        "S",
				},
			},
		}
		assert.Equal(t, exp, result, "GetCloudMiningPaymentAndRefundHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetCloudMiningPaymentAndRefundHistory should return a history")
}

func TestGetUserDelegationHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetUserDelegationHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUserDelegationHistory must reject a nil request")
	_, err = e.GetUserDelegationHistory(t.Context(), &UserDelegationHistoryRequest{Type: "DELEGATE"})
	require.ErrorIs(t, err, errValidEmailRequired, "GetUserDelegationHistory must reject an empty email")
	_, err = e.GetUserDelegationHistory(t.Context(), &UserDelegationHistoryRequest{Email: "someone@example.com"})
	require.ErrorIs(t, err, common.ErrDateUnset, "GetUserDelegationHistory must reject an unset time window")
	startTime, endTime := getTime()
	_, err = e.GetUserDelegationHistory(t.Context(), &UserDelegationHistoryRequest{Email: "someone@example.com", StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUserDelegationHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUserDelegationHistory(t.Context(), &UserDelegationHistoryRequest{
		Email:     "someone@example.com",
		StartTime: startTime,
		EndTime:   endTime,
		Type:      "DELEGATE",
		Asset:     currency.BTC,
	})
	require.NoError(t, err, "GetUserDelegationHistory must not error")
	if mockTests {
		exp := &UserDelegationHistoryResponse{
			Total: 1,
			Rows: []UserDelegation{
				{
					ClientTransactionID: "293915932290879488",
					TransferType:        "Delegate",
					Asset:               currency.BTC,
					Amount:              0.5,
					Time:                types.Time(time.Unix(1744110000, 0)),
				},
			},
		}
		assert.Equal(t, exp, result, "GetUserDelegationHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetUserDelegationHistory should return a history")
}

func TestGetUserUniversalTransferHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetUserUniversalTransferHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUserUniversalTransferHistory must reject a nil request")
	_, err = e.GetUserUniversalTransferHistory(t.Context(), &UserUniversalTransferHistoryRequest{})
	require.ErrorIs(t, err, errTransferTypeRequired, "GetUserUniversalTransferHistory must reject an empty type")
	startTime, endTime := getTime()
	_, err = e.GetUserUniversalTransferHistory(t.Context(), &UserUniversalTransferHistoryRequest{Type: "UMFUTURE_MARGIN", StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUserUniversalTransferHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUserUniversalTransferHistory(t.Context(), &UserUniversalTransferHistoryRequest{
		Type:       "ISOLATEDMARGIN_ISOLATEDMARGIN",
		StartTime:  startTime,
		EndTime:    endTime,
		Current:    1,
		Size:       100,
		FromSymbol: currency.NewBTCUSDT(),
		ToSymbol:   currency.NewPair(currency.ETH, currency.USDT),
	})
	require.NoError(t, err, "GetUserUniversalTransferHistory must not error")
	if mockTests {
		exp := &UserUniversalTransferHistoryResponse{
			Total: 1,
			Rows: []UserUniversalTransferRecord{
				{
					Asset:         currency.USDT,
					Amount:        25,
					Type:          "ISOLATEDMARGIN_ISOLATEDMARGIN",
					Status:        "CONFIRMED",
					TransactionID: 11415955596,
					Timestamp:     types.Time(time.Unix(1744110000, 0)),
				},
			},
		}
		assert.Equal(t, exp, result, "GetUserUniversalTransferHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetUserUniversalTransferHistory should return a history")
}

func TestUserUniversalTransfer(t *testing.T) {
	t.Parallel()
	_, err := e.UserUniversalTransfer(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "UserUniversalTransfer must reject a nil request")
	_, err = e.UserUniversalTransfer(t.Context(), &UserUniversalTransferRequest{Asset: currency.BTC, Amount: 123.234})
	require.ErrorIs(t, err, errTransferTypeRequired, "UserUniversalTransfer must reject an empty type")
	_, err = e.UserUniversalTransfer(t.Context(), &UserUniversalTransferRequest{Type: "MAIN_UMFUTURE", Amount: 123.234})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "UserUniversalTransfer must reject an empty asset")
	_, err = e.UserUniversalTransfer(t.Context(), &UserUniversalTransferRequest{Type: "MAIN_UMFUTURE", Asset: currency.BTC})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "UserUniversalTransfer must reject a zero amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.UserUniversalTransfer(t.Context(), &UserUniversalTransferRequest{Type: "MAIN_UMFUTURE", Asset: currency.BTC, Amount: 123.234})
	require.NoError(t, err, "UserUniversalTransfer must not error")
	if mockTests {
		exp := &UserUniversalTransferResponse{
			TransactionID: 13526853623,
		}
		assert.Equal(t, exp, result, "UserUniversalTransfer should decode every field")
		return
	}
	assert.NotZero(t, result.TransactionID, "UserUniversalTransfer should return a transaction ID")
}

func TestGetUserWalletBalance(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUserWalletBalance(t.Context(), currency.ETH, true)
	require.NoError(t, err, "GetUserWalletBalance must not error")
	if mockTests {
		exp := []UserWalletBalance{
			{
				Activate:   true,
				Balance:    0.0021,
				WalletName: "Spot",
				AssetBalances: []WalletAssetBalance{
					{
						Asset:        currency.USDT,
						AssetName:    "TetherUS",
						Free:         6.238383,
						Locked:       1,
						Freeze:       0.5,
						Withdrawing:  0.25,
						BTCValuation: 0.00009,
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetUserWalletBalance should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetUserWalletBalance should return wallet balances")
}

func TestToggleBNBBurn(t *testing.T) {
	t.Parallel()
	_, err := e.ToggleBNBBurn(t.Context(), nil, nil)
	require.ErrorIs(t, err, errBNBBurnSettingRequired, "ToggleBNBBurn must require a setting")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name                         string
		spotBNBBurn, interestBNBBurn *bool
		exp                          *BNBBurnStatusResponse
	}{
		{
			name: "spot", spotBNBBurn: new(true),
			exp: &BNBBurnStatusResponse{
				SpotBNBBurn: true,
			},
		},
		{
			name: "interest", interestBNBBurn: new(true),
			exp: &BNBBurnStatusResponse{
				InterestBNBBurn: true,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.ToggleBNBBurn(t.Context(), tc.spotBNBBurn, tc.interestBNBBurn)
			require.NoError(t, err, "ToggleBNBBurn must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "ToggleBNBBurn should decode every field")
				return
			}
			if tc.spotBNBBurn != nil {
				assert.Equal(t, *tc.spotBNBBurn, result.SpotBNBBurn, "ToggleBNBBurn should return the spot BNB burn set")
			}
		})
	}
}

func TestGetTradeFees(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	for _, tc := range []struct {
		name   string
		symbol currency.Pair
		exp    []TradeFee
	}{
		{
			name: "all",
			exp: []TradeFee{
				{
					Symbol:          "ADABNB",
					MakerCommission: 0.001,
					TakerCommission: 0.001,
				},
				{
					Symbol:          "BTCUSDT",
					MakerCommission: 0.00075,
					TakerCommission: 0.00075,
				},
			},
		},
		{
			name: "symbol", symbol: spotTradablePair,
			exp: []TradeFee{
				{
					Symbol:          "BTCUSDT",
					MakerCommission: 0.00075,
					TakerCommission: 0.00075,
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetTradeFees(t.Context(), tc.symbol)
			require.NoError(t, err, "GetTradeFees must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetTradeFees should decode every field")
				return
			}
			assert.NotEmpty(t, result, "GetTradeFees should return trade fees")
		})
	}
}

func TestGetUserAssets(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUserAssets(t.Context(), currency.BTC, true)
	require.NoError(t, err, "GetUserAssets must not error")
	if mockTests {
		exp := []UserAsset{
			{
				Asset:        currency.BTC,
				Free:         1,
				Locked:       0.1,
				Freeze:       0.01,
				Withdrawing:  0.02,
				IPOable:      0.03,
				BTCValuation: 1.16,
			},
		}
		assert.Equal(t, exp, result, "GetUserAssets should decode every field")
		return
	}
	assert.NotNil(t, result, "GetUserAssets should return user assets")
}

func TestGetAllCoinsInfo(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAllCoinsInfo(t.Context())
	require.NoError(t, err, "GetAllCoinsInfo must not error")
	if mockTests {
		exp := []CoinInfo{
			{
				Coin:              currency.BTC,
				DepositAllEnable:  true,
				WithdrawAllEnable: true,
				Name:              "Bitcoin",
				Free:              0.08074558,
				Locked:            0.001,
				Freeze:            0.0002,
				Withdrawing:       0.0003,
				IPOing:            0.0004,
				IPOable:           0.0005,
				Storage:           0.0006,
				Trading:           true,
				NetworkList: []CoinNetwork{
					{
						Network:                 "BTC",
						Coin:                    currency.BTC,
						WithdrawIntegerMultiple: 0.00000001,
						IsDefault:               true,
						DepositEnable:           true,
						WithdrawEnable:          true,
						Name:                    "Bitcoin",
						AddressRegex:            "^[13][a-km-zA-HJ-NP-Z1-9]{25,34}$|^(bc1)[0-9A-Za-z]{39,59}$",
						WithdrawFee:             0.00002,
						WithdrawMinimum:         0.0001,
						WithdrawMaximum:         9999999999,
						WithdrawInternalMinimum: 0.00000001,
						DepositDust:             0.000006,
						MinimumConfirm:          1,
						UnlockConfirm:           2,
						EstimatedArrivalTime:    60,
						Denomination:            1,
					},
					{
						Network:                 "BSC",
						Coin:                    currency.BTC,
						WithdrawIntegerMultiple: 0.00000001,
						DepositEnable:           true,
						WithdrawEnable:          true,
						SpecialTips:             "Please confirm the BTCB token contract address before depositing",
						Name:                    "BNB Smart Chain (BEP20)",
						AddressRegex:            "^(0x)[0-9A-Fa-f]{40}$",
						WithdrawFee:             0.0000035,
						WithdrawMinimum:         0.000007,
						WithdrawMaximum:         9999999999,
						WithdrawInternalMinimum: 0.00000001,
						DepositDust:             0.0000001,
						MinimumConfirm:          15,
						EstimatedArrivalTime:    5,
						ContractAddressURL:      "https://bscscan.com/token/",
						ContractAddress:         "0x7130d2a12b9bcbfae4f2634d864a1ee1ce3ead9c",
						Denomination:            1,
					},
				},
			},
			{
				Coin:        currency.XRP,
				Name:        "Ripple",
				Free:        125.5,
				Locked:      1,
				Freeze:      2,
				Withdrawing: 3,
				IPOing:      4,
				IPOable:     5,
				Storage:     6,
				Trading:     true,
				NetworkList: []CoinNetwork{
					{
						Network:                 "XRP",
						Coin:                    currency.XRP,
						WithdrawIntegerMultiple: 0.000001,
						IsDefault:               true,
						DepositDescription:      "Wallet Maintenance, Deposit Suspended",
						WithdrawDescription:     "Wallet Maintenance, Withdrawal Suspended",
						SpecialTips:             "Both a MEMO and an Address are required to successfully deposit your XRP to Binance.",
						SpecialWithdrawTips:     "Please confirm the MEMO the receiving platform requires",
						Name:                    "Ripple",
						ResetAddressStatus:      true,
						AddressRegex:            "^r[1-9A-HJ-NP-Za-km-z]{25,34}$",
						MemoRegex:               "^((?!0)[0-9]{1,10})$",
						WithdrawFee:             0.2,
						WithdrawMinimum:         2,
						WithdrawMaximum:         9999999,
						WithdrawInternalMinimum: 0.000001,
						DepositDust:             0.0001,
						MinimumConfirm:          1,
						UnlockConfirm:           1,
						SameAddress:             true,
						WithdrawTag:             true,
						EstimatedArrivalTime:    1,
						Busy:                    true,
						Denomination:            1,
					},
				},
			},
			{
				Coin:              currency.EUR,
				DepositAllEnable:  true,
				WithdrawAllEnable: true,
				Name:              "Euro",
				Free:              10,
				IsLegalMoney:      true,
				Trading:           true,
				NetworkList:       []CoinNetwork{},
			},
		}
		assert.Equal(t, exp, result, "GetAllCoinsInfo should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetAllCoinsInfo should return coins")
}

func TestGetDepositAddressForCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.GetDepositAddressForCurrency(t.Context(), currency.EMPTYCODE, "", 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetDepositAddressForCurrency must reject an empty coin")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name    string
		coin    currency.Code
		network string
		amount  float64
		exp     *DepositAddressResponse
	}{
		{
			name: "default network", coin: currency.BTC,
			exp: &DepositAddressResponse{
				Address: "1HPn8Rx2y6nNSfagQBKy27GB99Vbzg89wv",
				Coin:    currency.BTC,
				URL:     "https://btc.com/1HPn8Rx2y6nNSfagQBKy27GB99Vbzg89wv",
			},
		},
		{
			name: "network with tag", coin: currency.XRP, network: "XRP",
			exp: &DepositAddressResponse{
				Address:   "rEb8TK3gBgk5auZkwc6sHnwrGVJH8DuaLh",
				Coin:      currency.XRP,
				IsDefault: 1,
				Tag:       "101764890",
				URL:       "https://xrpscan.com/account/rEb8TK3gBgk5auZkwc6sHnwrGVJH8DuaLh",
			},
		},
		{
			name: "lightning", coin: currency.BTC, network: "LIGHTNING", amount: 0.001,
			exp: &DepositAddressResponse{
				Address: "lnbc1m1pn2s3kspp5qqqsyqcyq5rqwzqfqqqsyqcyq5rqwzqfqqqsyqcyq5rqwzqfqypq",
				Coin:    currency.BTC,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetDepositAddressForCurrency(t.Context(), tc.coin, tc.network, tc.amount)
			require.NoError(t, err, "GetDepositAddressForCurrency must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetDepositAddressForCurrency should decode every field")
				return
			}
			assert.NotEmpty(t, result.Address, "GetDepositAddressForCurrency should return an address")
		})
	}
}

func TestDepositHistory(t *testing.T) {
	t.Parallel()
	_, err := e.DepositHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "DepositHistory must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.DepositHistory(t.Context(), &DepositHistoryRequest{Coin: currency.ETH, StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "DepositHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *DepositHistoryRequest
		exp  []DepositRecord
	}{
		{
			name: "source",
			req:  &DepositHistoryRequest{IncludeSource: true, Coin: currency.XRP, Status: new(uint64(1)), StartTime: startTime, EndTime: endTime, Limit: 1000},
			exp: []DepositRecord{
				{
					ID:            "769800519366885376",
					Amount:        25,
					Coin:          currency.XRP,
					Network:       "XRP",
					Status:        1,
					Address:       "rEb8TK3gBgk5auZkwc6sHnwrGVJH8DuaLh",
					AddressTag:    "101764890",
					TxID:          "98A3EA560C6B3336D348B6C83F0F95ECE4F1F5919E94BD006E5BF3BF264FACFC",
					InsertTime:    types.Time(time.Unix(1744110000, 0)),
					CompleteTime:  types.Time(time.Unix(1744110060, 0)),
					ConfirmTimes:  "1/1",
					UnlockConfirm: 2,
					SourceAddress: "rPEPPER7kfTD9w2To4CQk6UCfuHM9c6GDY",
				},
				{
					ID:               "769800519366885377",
					Amount:           10,
					Coin:             currency.XRP,
					Network:          "XRP",
					Status:           1,
					Address:          "rEb8TK3gBgk5auZkwc6sHnwrGVJH8DuaLh",
					AddressTag:       "101764890",
					TxID:             "Off-chain transfer 9d3c1b7a5e2f4d6c",
					InsertTime:       types.Time(time.Unix(1744120000, 0)),
					CompleteTime:     types.Time(time.Unix(1744120000, 0)),
					TransferType:     1,
					ConfirmTimes:     "1/1",
					WalletType:       1,
					TravelRuleStatus: 1,
				},
			},
		},
		{
			name: "transaction",
			req:  &DepositHistoryRequest{Offset: 1, TxID: "D5D5D1DE4C5D2B9C4DCC8B0E6F2D9E1A6B0F7C3E2A1D4B5C6E7F8091A2B3C4D5"},
			exp: []DepositRecord{
				{
					ID:            "769800519366885378",
					Amount:        0.5,
					Coin:          currency.BNB,
					Network:       "BSC",
					Status:        6,
					Address:       "0xd316e95fd9e8e237cb11f8200babbc5d8d177ba4",
					TxID:          "D5D5D1DE4C5D2B9C4DCC8B0E6F2D9E1A6B0F7C3E2A1D4B5C6E7F8091A2B3C4D5",
					InsertTime:    types.Time(time.Unix(1744130000, 0)),
					CompleteTime:  types.Time(time.Unix(1744130120, 0)),
					ConfirmTimes:  "15/15",
					UnlockConfirm: 15,
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.DepositHistory(t.Context(), tc.req)
			require.NoError(t, err, "DepositHistory must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "DepositHistory should decode every field")
				return
			}
			assert.NotNil(t, result, "DepositHistory should return deposits")
		})
	}
}

func TestGetDepositAddressListWithNetwork(t *testing.T) {
	t.Parallel()
	_, err := e.GetDepositAddressListWithNetwork(t.Context(), currency.EMPTYCODE, "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetDepositAddressListWithNetwork must reject an empty coin")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name    string
		coin    currency.Code
		network string
		exp     []DepositAddressWithNetwork
	}{
		{
			name: "default network", coin: currency.BTC,
			exp: []DepositAddressWithNetwork{
				{
					Coin:      currency.BTC,
					Address:   "1HPn8Rx2y6nNSfagQBKy27GB99Vbzg89wv",
					IsDefault: 1,
				},
				{
					Coin:    currency.BTC,
					Address: "bc1qxy2kgdygjrsqtzq2n0yrf2493p83kkfjhx0wlh",
				},
			},
		},
		{
			name: "network with tag", coin: currency.XRP, network: "XRP",
			exp: []DepositAddressWithNetwork{
				{
					Coin:      currency.XRP,
					Address:   "rEb8TK3gBgk5auZkwc6sHnwrGVJH8DuaLh",
					Tag:       "101764890",
					IsDefault: 1,
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetDepositAddressListWithNetwork(t.Context(), tc.coin, tc.network)
			require.NoError(t, err, "GetDepositAddressListWithNetwork must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetDepositAddressListWithNetwork should decode every field")
				return
			}
			assert.NotEmpty(t, result, "GetDepositAddressListWithNetwork should return addresses")
		})
	}
}

func TestGetWithdrawAddressList(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetWithdrawAddressList(t.Context())
	require.NoError(t, err, "GetWithdrawAddressList must not error")
	if mockTests {
		exp := []WithdrawAddress{
			{
				Address:     "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa",
				Coin:        currency.BTC,
				Name:        "Satoshi",
				Network:     "BTC",
				Origin:      "Hardware wallet",
				OriginType:  "others",
				WhiteStatus: true,
			},
			{
				Address:    "rEb8TK3gBgk5auZkwc6sHnwrGVJH8DuaLh",
				AddressTag: "101764890",
				Coin:       currency.XRP,
				Name:       "Exchange",
				Network:    "XRP",
				OriginType: "exchange",
			},
		}
		assert.Equal(t, exp, result, "GetWithdrawAddressList should decode every field")
		return
	}
	assert.NotNil(t, result, "GetWithdrawAddressList should return addresses")
}

func TestOneClickArrivalDepositApply(t *testing.T) {
	t.Parallel()
	_, err := e.OneClickArrivalDepositApply(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "OneClickArrivalDepositApply must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.OneClickArrivalDepositApply(t.Context(), &DepositCreditApplyRequest{
		DepositID:    769800519366885376,
		TxID:         "0xb5ef8c13b968a406cc62a93a8bd80f9e9a906ef1b3fcf20a2e48573c17659268",
		SubAccountID: "1",
		SubUserID:    2,
	})
	require.NoError(t, err, "OneClickArrivalDepositApply must not error")
	if mockTests {
		exp := &DepositCreditApplyResponse{
			Code:    "000000",
			Message: "success",
			Data:    true,
			Success: true,
		}
		assert.Equal(t, exp, result, "OneClickArrivalDepositApply should decode every field")
		return
	}
	assert.True(t, result.Success, "OneClickArrivalDepositApply should succeed")
}

func TestWithdrawCrypto(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawCrypto(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WithdrawCrypto must reject a nil request")
	_, err = e.WithdrawCrypto(t.Context(), &WithdrawRequest{WithdrawOrderID: "123435", Address: "address-here", Amount: 100})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "WithdrawCrypto must reject an empty coin")
	_, err = e.WithdrawCrypto(t.Context(), &WithdrawRequest{Coin: currency.USDT, Amount: 100})
	require.ErrorIs(t, err, errAddressRequired, "WithdrawCrypto must reject an empty address")
	_, err = e.WithdrawCrypto(t.Context(), &WithdrawRequest{Coin: currency.USDT, WithdrawOrderID: "123435", Address: "address-here", AddressTag: "123213"})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "WithdrawCrypto must reject a zero amount")

	if mockTests {
		// A rejected withdrawal must return the API error rather than dereference the missing response
		result, err := e.WithdrawCrypto(t.Context(), &WithdrawRequest{Coin: currency.USDT, Address: "0x94df8b352de7f46f64b01d3666bf6e936e44ce60", Amount: 1000000})
		require.ErrorIs(t, err, errAPIResponse, "WithdrawCrypto must return the API error of a rejected withdrawal")
		assert.Nil(t, result, "WithdrawCrypto should not return a response for a rejected withdrawal")
	} else {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.WithdrawCrypto(t.Context(), &WithdrawRequest{
		Coin:               currency.USDT,
		WithdrawOrderID:    "123435",
		Network:            "ETH",
		Address:            "0x94df8b352de7f46f64b01d3666bf6e936e44ce60",
		Amount:             100,
		TransactionFeeFlag: true,
		Name:               "Test Address",
		WalletType:         new(uint64(1)),
	})
	require.NoError(t, err, "WithdrawCrypto must not error")
	if mockTests {
		exp := &WithdrawResponse{
			ID: "7213fea8e94b4a5593d507237e5a555b",
		}
		assert.Equal(t, exp, result, "WithdrawCrypto should decode every field")
		return
	}
	assert.NotEmpty(t, result.ID, "WithdrawCrypto should return a withdrawal ID")
}

func TestWithdrawHistory(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WithdrawHistory must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.WithdrawHistory(t.Context(), &WithdrawHistoryRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "WithdrawHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *WithdrawHistoryRequest
		exp  []WithdrawalRecord
	}{
		{
			name: "IDs",
			req: &WithdrawHistoryRequest{
				Coin:      currency.USDT,
				Status:    new(uint64(6)),
				Offset:    1,
				Limit:     1000,
				IDList:    []string{"b6ae22b3aa844210a7041aee7589627c", "c8f1a3b5d7e94b2c8a6f0e1d2c3b4a59"},
				StartTime: startTime,
				EndTime:   endTime,
			},
			exp: []WithdrawalRecord{
				{
					ID:              "b6ae22b3aa844210a7041aee7589627c",
					Amount:          8.91,
					TransactionFee:  0.004,
					Coin:            currency.USDT,
					Status:          6,
					Address:         "0x94df8b352de7f46f64b01d3666bf6e936e44ce60",
					TxID:            "0xb5ef8c13b968a406cc62a93a8bd80f9e9a906ef1b3fcf20a2e48573c17659268",
					ApplyTime:       types.DateTime(time.Date(2025, 4, 8, 11, 12, 2, 0, time.UTC)),
					Network:         "ETH",
					WithdrawOrderID: "WITHDRAWtest123",
					ConfirmNumber:   12,
					CompleteTime:    types.DateTime(time.Date(2025, 4, 8, 11, 20, 41, 0, time.UTC)),
				},
				{
					ID:              "c8f1a3b5d7e94b2c8a6f0e1d2c3b4a59",
					Amount:          5,
					Coin:            currency.USDT,
					Status:          6,
					Address:         "0x5a52e96bacdabb82fd05763e25335261b270efcb",
					TxID:            "Off-chain transfer 2e4d6a8f0c1b3d5f",
					ApplyTime:       types.DateTime(time.Date(2025, 4, 8, 15, 1, 12, 0, time.UTC)),
					Network:         "ETH",
					TransferType:    1,
					WithdrawOrderID: "WITHDRAWtest124",
					Info:            "Internal transfer",
					WalletType:      1,
					TxKey:           "9f86d081884c7d659a2feaa0c55ad015",
					CompleteTime:    types.DateTime(time.Date(2025, 4, 8, 15, 1, 13, 0, time.UTC)),
				},
			},
		},
		{
			name: "withdraw order ID",
			req:  &WithdrawHistoryRequest{WithdrawOrderID: "WITHDRAWtest123"},
			exp: []WithdrawalRecord{
				{
					ID:              "b6ae22b3aa844210a7041aee7589627c",
					Amount:          8.91,
					TransactionFee:  0.004,
					Coin:            currency.USDT,
					Status:          6,
					Address:         "0x94df8b352de7f46f64b01d3666bf6e936e44ce60",
					TxID:            "0xb5ef8c13b968a406cc62a93a8bd80f9e9a906ef1b3fcf20a2e48573c17659268",
					ApplyTime:       types.DateTime(time.Date(2025, 4, 8, 11, 12, 2, 0, time.UTC)),
					Network:         "ETH",
					WithdrawOrderID: "WITHDRAWtest123",
					ConfirmNumber:   12,
					CompleteTime:    types.DateTime(time.Date(2025, 4, 8, 11, 20, 41, 0, time.UTC)),
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.WithdrawHistory(t.Context(), tc.req)
			require.NoError(t, err, "WithdrawHistory must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "WithdrawHistory should decode every field")
				return
			}
			assert.NotNil(t, result, "WithdrawHistory should return withdrawals")
		})
	}
}

func TestGetSymbolsDelistScheduleForSpot(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSymbolsDelistScheduleForSpot(t.Context())
	require.NoError(t, err, "GetSymbolsDelistScheduleForSpot must not error")
	if mockTests {
		exp := []DelistSchedule{
			{
				DelistTime: types.Time(time.Unix(1744156800, 0)),
				Symbols: []string{
					"ADAUSDT",
					"ADABNB",
				},
			},
		}
		assert.Equal(t, exp, result, "GetSymbolsDelistScheduleForSpot should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSymbolsDelistScheduleForSpot should return a schedule")
}

func TestGetSystemStatus(t *testing.T) {
	t.Parallel()
	result, err := e.GetSystemStatus(t.Context())
	require.NoError(t, err, "GetSystemStatus must not error")
	if mockTests {
		exp := &SystemStatusResponse{
			Status:  1,
			Message: "system_maintenance",
		}
		assert.Equal(t, exp, result, "GetSystemStatus should decode every field")
		return
	}
	assert.NotEmpty(t, result.Message, "GetSystemStatus should return a status message")
}

func TestGetLocalEntitiesDepositHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetLocalEntitiesDepositHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLocalEntitiesDepositHistory must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetLocalEntitiesDepositHistory(t.Context(), &LocalEntityDepositHistoryRequest{Network: "BNB", Coin: currency.USDT, StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetLocalEntitiesDepositHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *LocalEntityDepositHistoryRequest
		exp  []LocalEntityDepositRecord
	}{
		{
			name: "records",
			req: &LocalEntityDepositHistoryRequest{
				TravelRuleRecordIDs: []uint64{123451123, 123451124},
				TxIDs:               []string{"98A3EA560C6B3336D348B6C83F0F95ECE4F1F5919E94BD006E5BF3BF264FACFC", "Off-chain transfer 4f6e2a1c9b8d7e3f"},
				TransactionIDs:      []uint64{17644346245865, 17644346245866},
				StartTime:           startTime,
				EndTime:             endTime,
				Offset:              1,
				Limit:               10,
			},
			exp: []LocalEntityDepositRecord{
				{
					TravelRuleRecordID:   123451123,
					TransactionID:        17644346245865,
					Amount:               0.001,
					Coin:                 currency.BNB,
					Network:              "BNB",
					TravelRuleStatus:     1,
					TravelRuleStatusV2:   "PENDING",
					Address:              "bnb136ns6lfw4zs5hg4n85vdthaad7hq5m4gtkgf23",
					AddressTag:           "101764890",
					TxID:                 "98A3EA560C6B3336D348B6C83F0F95ECE4F1F5919E94BD006E5BF3BF264FACFC",
					InsertTime:           types.Time(time.Unix(1744110000, 0)),
					ConfirmTimes:         "1/1",
					RequireQuestionnaire: true,
				},
				{
					TravelRuleRecordID: 123451124,
					TransactionID:      17644346245866,
					Amount:             12.5,
					Coin:               currency.USDT,
					Network:            "BSC",
					DepositStatus:      1,
					TravelRuleStatusV2: "PASSED",
					Address:            "0xd316e95fd9e8e237cb11f8200babbc5d8d177ba4",
					TxID:               "Off-chain transfer 4f6e2a1c9b8d7e3f",
					InsertTime:         types.Time(time.Unix(1744120000, 0)),
					CompleteTime:       types.Time(time.Unix(1744120060, 0)),
					TransferType:       1,
					ConfirmTimes:       "15/15",
					Questionnaire:      "{\"depositOriginator\":2,\"receiveFrom\":1,\"declaration\":true}",
				},
			},
		},
		{
			name: "pending",
			req: &LocalEntityDepositHistoryRequest{
				Network:              "BNB",
				Coin:                 currency.USDT,
				TravelRuleStatus:     new(uint64(1)),
				PendingQuestionnaire: true,
				StartTime:            startTime,
				EndTime:              endTime,
				Limit:                10,
			},
			exp: []LocalEntityDepositRecord{
				{
					TravelRuleRecordID:   123451125,
					TransactionID:        17644346245867,
					Amount:               30,
					Coin:                 currency.USDT,
					Network:              "BNB",
					TravelRuleStatus:     1,
					TravelRuleStatusV2:   "PENDING",
					Address:              "bnb136ns6lfw4zs5hg4n85vdthaad7hq5m4gtkgf23",
					AddressTag:           "101764891",
					TxID:                 "A1B2C3D4E5F60718293A4B5C6D7E8F90A1B2C3D4E5F60718293A4B5C6D7E8F90",
					InsertTime:           types.Time(time.Unix(1744130000, 0)),
					ConfirmTimes:         "1/1",
					RequireQuestionnaire: true,
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetLocalEntitiesDepositHistory(t.Context(), tc.req)
			require.NoError(t, err, "GetLocalEntitiesDepositHistory must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetLocalEntitiesDepositHistory should decode every field")
				return
			}
			assert.NotNil(t, result, "GetLocalEntitiesDepositHistory should return deposits")
		})
	}
}

func TestSubmitDepositQuestionnaire(t *testing.T) {
	t.Parallel()
	questionnaire := map[string]any{
		"isAddressOwner": 2,
		"sendTo":         1,
		"vaspCountry":    "cn",
		"vaspRegion":     "notNortheasternProvinces",
		"txnPurpose":     "3",
	}
	_, err := e.SubmitDepositQuestionnaire(t.Context(), 0, questionnaire)
	require.ErrorIs(t, err, errTransactionIDRequired, "SubmitDepositQuestionnaire must reject an unset transaction ID")
	_, err = e.SubmitDepositQuestionnaire(t.Context(), 765127651, nil)
	require.ErrorIs(t, err, errQuestionnaireRequired, "SubmitDepositQuestionnaire must reject an empty questionnaire")
	_, err = e.SubmitDepositQuestionnaire(t.Context(), 765127651, map[string]any{"sendTo": make(chan int)})
	require.ErrorContains(t, err, "error encoding deposit questionnaire", "SubmitDepositQuestionnaire must report a questionnaire it cannot encode")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SubmitDepositQuestionnaire(t.Context(), 765127651, questionnaire)
	require.NoError(t, err, "SubmitDepositQuestionnaire must not error")
	if mockTests {
		exp := &DepositQuestionnaireResponse{
			TravelRuleRecordID: 765127651,
			Accepted:           true,
			Info:               "Deposit questionnaire accepted.",
		}
		assert.Equal(t, exp, result, "SubmitDepositQuestionnaire should decode every field")
		return
	}
	assert.True(t, result.Accepted, "SubmitDepositQuestionnaire should be accepted")
}

func TestGetOnboardedVASPList(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetOnboardedVASPList(t.Context())
	require.NoError(t, err, "GetOnboardedVASPList must not error")
	if mockTests {
		exp := []VASP{
			{
				VASPName:   "Binance",
				VASPCode:   "BINANCE",
				Identifier: "6f1c2d6e-58a6-4f2b-9d1e-2c3b4a5d6e7f",
			},
		}
		assert.Equal(t, exp, result, "GetOnboardedVASPList should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetOnboardedVASPList should return VASPs")
}

func TestWithdrawalHistoryV1(t *testing.T) {
	t.Parallel()
	testLocalEntityWithdrawHistory(t, e.WithdrawalHistoryV1)
}

func TestWithdrawalHistoryV2(t *testing.T) {
	t.Parallel()
	testLocalEntityWithdrawHistory(t, e.WithdrawalHistoryV2)
}

func testLocalEntityWithdrawHistory(t *testing.T, withdrawalHistory func(context.Context, *LocalEntityWithdrawHistoryRequest) ([]LocalEntityWithdrawalRecord, error)) {
	t.Helper()
	_, err := withdrawalHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "withdrawal history must reject a nil request")
	startTime, endTime := getTime()
	_, err = withdrawalHistory(t.Context(), &LocalEntityWithdrawHistoryRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "withdrawal history must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *LocalEntityWithdrawHistoryRequest
		exp  []LocalEntityWithdrawalRecord
	}{
		{
			name: "records",
			req: &LocalEntityWithdrawHistoryRequest{
				TravelRuleRecordIDs: []uint64{1234, 1235},
				TxIDs:               []string{"0xb5ef8c13b968a406cc62a93a8bd80f9e9a906ef1b3fcf20a2e48573c17659268", "Off-chain transfer 7c2e9a4b1d3f5e6a"},
				Offset:              1,
				Limit:               100,
				StartTime:           startTime,
				EndTime:             endTime,
			},
			exp: []LocalEntityWithdrawalRecord{
				{
					ID:                 "b6ae22b3aa844210a7041aee7589627c",
					TravelRuleRecordID: 1234,
					Amount:             8.91,
					TransactionFee:     0.004,
					Coin:               currency.USDT,
					WithdrawalStatus:   6,
					Address:            "0x94df8b352de7f46f64b01d3666bf6e936e44ce60",
					TxID:               "0xb5ef8c13b968a406cc62a93a8bd80f9e9a906ef1b3fcf20a2e48573c17659268",
					ApplyTime:          types.DateTime(time.Date(2025, 4, 8, 11, 12, 2, 0, time.UTC)),
					Network:            "ETH",
					WithdrawOrderID:    "WITHDRAWtest123",
					ConfirmNumber:      12,
					Questionnaire:      "{\"isAddressOwner\":1,\"sendTo\":1}",
					CompleteTime:       types.DateTime(time.Date(2025, 4, 8, 11, 20, 41, 0, time.UTC)),
				},
				{
					ID:                 "c8f1a3b5d7e94b2c8a6f0e1d2c3b4a59",
					TravelRuleRecordID: 1235,
					Amount:             5,
					Coin:               currency.USDT,
					WithdrawalStatus:   4,
					TravelRuleStatus:   1,
					Address:            "0x5a52e96bacdabb82fd05763e25335261b270efcb",
					TxID:               "Off-chain transfer 7c2e9a4b1d3f5e6a",
					ApplyTime:          types.DateTime(time.Date(2025, 4, 8, 15, 1, 12, 0, time.UTC)),
					Network:            "ETH",
					TransferType:       1,
					WithdrawOrderID:    "WITHDRAWtest124",
					Info:               "Pending travel rule verification",
					WalletType:         1,
					TxKey:              "9f86d081884c7d659a2feaa0c55ad015",
					Questionnaire:      "{\"isAddressOwner\":2,\"sendTo\":2}",
				},
			},
		},
		{
			name: "status",
			req: &LocalEntityWithdrawHistoryRequest{
				WithdrawOrderID:  "WITHDRAWtest123",
				Network:          "ETH",
				Coin:             currency.USDT,
				TravelRuleStatus: new(uint64(0)),
				StartTime:        startTime,
				EndTime:          endTime,
			},
			exp: []LocalEntityWithdrawalRecord{
				{
					ID:                 "b6ae22b3aa844210a7041aee7589627c",
					TravelRuleRecordID: 1234,
					Amount:             8.91,
					TransactionFee:     0.004,
					Coin:               currency.USDT,
					WithdrawalStatus:   6,
					Address:            "0x94df8b352de7f46f64b01d3666bf6e936e44ce60",
					TxID:               "0xb5ef8c13b968a406cc62a93a8bd80f9e9a906ef1b3fcf20a2e48573c17659268",
					ApplyTime:          types.DateTime(time.Date(2025, 4, 8, 11, 12, 2, 0, time.UTC)),
					Network:            "ETH",
					WithdrawOrderID:    "WITHDRAWtest123",
					ConfirmNumber:      12,
					Questionnaire:      "{\"isAddressOwner\":1,\"sendTo\":1}",
					CompleteTime:       types.DateTime(time.Date(2025, 4, 8, 11, 20, 41, 0, time.UTC)),
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := withdrawalHistory(t.Context(), tc.req)
			require.NoError(t, err, "withdrawal history must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "withdrawal history should decode every field")
				return
			}
			assert.NotNil(t, result, "withdrawal history should return withdrawals")
		})
	}
}

func TestGetFutureTickLevelOrderbookHistoricalDataDownloadLink(t *testing.T) {
	t.Parallel()
	_, err := e.GetFutureTickLevelOrderbookHistoricalDataDownloadLink(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFutureTickLevelOrderbookHistoricalDataDownloadLink must reject a nil request")
	_, err = e.GetFutureTickLevelOrderbookHistoricalDataDownloadLink(t.Context(), &HistoricalDataDownloadLinkRequest{DataType: "T_DEPTH"})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetFutureTickLevelOrderbookHistoricalDataDownloadLink must reject an empty symbol")
	_, err = e.GetFutureTickLevelOrderbookHistoricalDataDownloadLink(t.Context(), &HistoricalDataDownloadLinkRequest{Symbol: currency.NewBTCUSDT()})
	require.ErrorIs(t, err, errHistoricalDataTypeRequired, "GetFutureTickLevelOrderbookHistoricalDataDownloadLink must reject an empty data type")
	startTime, endTime := getTime()
	_, err = e.GetFutureTickLevelOrderbookHistoricalDataDownloadLink(t.Context(), &HistoricalDataDownloadLinkRequest{Symbol: currency.NewBTCUSDT(), DataType: "T_DEPTH", StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFutureTickLevelOrderbookHistoricalDataDownloadLink must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetFutureTickLevelOrderbookHistoricalDataDownloadLink(t.Context(), &HistoricalDataDownloadLinkRequest{
		Symbol:    usdtmTradablePair,
		DataType:  "T_DEPTH",
		StartTime: startTime,
		EndTime:   endTime,
	})
	require.NoError(t, err, "GetFutureTickLevelOrderbookHistoricalDataDownloadLink must not error")
	if mockTests {
		exp := &HistoricalDataDownloadLinkResponse{
			Data: []HistoricalDataDownloadLink{
				{
					Day: "2025-04-08",
					URL: "https://bin-prod-user-rebate-bucket.s3.ap-northeast-1.amazonaws.com/data-download/BTCUSDT_T_DEPTH_2025-04-08.tar.gz",
				},
			},
		}
		assert.Equal(t, exp, result, "GetFutureTickLevelOrderbookHistoricalDataDownloadLink should decode every field")
		return
	}
	assert.NotNil(t, result, "GetFutureTickLevelOrderbookHistoricalDataDownloadLink should return links")
}

func TestJoinUint64s(t *testing.T) {
	t.Parallel()
	assert.Empty(t, joinUint64s(nil), "joinUint64s should return an empty list for no IDs")
	assert.Equal(t, "1,23,18446744073709551615", joinUint64s([]uint64{1, 23, 18446744073709551615}), "joinUint64s should join IDs with commas")
}
