package binance

import (
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

const testBrokerSubAccountAPIKey = "vmPUZE6mv9SD5VNHk4HlWFsOr6aKE2zvsw0MuIgwCIPy6utIco14y7Ju91duEh8A"

func TestChangeSubAccountAPIPermission(t *testing.T) {
	t.Parallel()
	_, err := e.ChangeSubAccountAPIPermission(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ChangeSubAccountAPIPermission must reject a nil request")
	_, err = e.ChangeSubAccountAPIPermission(t.Context(), &BrokerSubAccountAPIPermissionRequest{SubAccountAPIKey: testBrokerSubAccountAPIKey})
	require.ErrorIs(t, err, errSubAccountIDMissing, "ChangeSubAccountAPIPermission must reject an empty sub-account ID")
	_, err = e.ChangeSubAccountAPIPermission(t.Context(), &BrokerSubAccountAPIPermissionRequest{SubAccountID: "1"})
	require.ErrorIs(t, err, errEmptySubAccountAPIKey, "ChangeSubAccountAPIPermission must reject an empty API key")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateAPICredentials)
	}
	for _, tc := range []struct {
		name     string
		canTrade bool
	}{
		{name: "revoke spot trading"},
		{name: "grant every permission", canTrade: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.ChangeSubAccountAPIPermission(t.Context(), &BrokerSubAccountAPIPermissionRequest{
				SubAccountID:     "1",
				SubAccountAPIKey: testBrokerSubAccountAPIKey,
				CanTrade:         tc.canTrade,
				MarginTrade:      true,
				FuturesTrade:     true,
			})
			require.NoError(t, err, "ChangeSubAccountAPIPermission must not error")
			if mockTests {
				exp := &BrokerSubAccountAPIPermissionResponse{SubAccountID: "1", APIKey: testBrokerSubAccountAPIKey, CanTrade: tc.canTrade, MarginTrade: true, FuturesTrade: true}
				assert.Equal(t, exp, result, "ChangeSubAccountAPIPermission should decode every field")
				return
			}
			assert.Equal(t, tc.canTrade, result.CanTrade, "ChangeSubAccountAPIPermission should set the spot trading permission")
		})
	}
}

func TestGetBrokerSubAccounts(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetBrokerSubAccounts(t.Context(), "1", 1, 10)
	require.NoError(t, err, "GetBrokerSubAccounts must not error")
	if mockTests {
		exp := []BrokerSubAccount{
			{
				SubAccountID:          "1",
				Email:                 "test_brokersubuser@example.com",
				Tag:                   "bob123",
				MakerCommission:       0.001,
				TakerCommission:       0.001,
				MarginMakerCommission: -1,
				MarginTakerCommission: -1,
				CreateTime:            types.Time(time.Unix(1544433328, 0)),
			},
			{
				SubAccountID:          "2",
				Email:                 "test_brokersubuser2@example.com",
				Tag:                   "bob124",
				MakerCommission:       0.0012,
				TakerCommission:       0.0015,
				MarginMakerCommission: 0.0012,
				MarginTakerCommission: 0.0015,
				CreateTime:            types.Time(time.Unix(1544433329, 0)),
			},
		}
		assert.Equal(t, exp, result, "GetBrokerSubAccounts should decode every field")
		return
	}
	assert.NotNil(t, result, "GetBrokerSubAccounts should return a result")
}

func TestCreateBrokerSubAccount(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateAPICredentials)
	}
	result, err := e.CreateBrokerSubAccount(t.Context(), "Thrasher")
	require.NoError(t, err, "CreateBrokerSubAccount must not error")
	if mockTests {
		exp := &CreateBrokerSubAccountResponse{SubAccountID: "1", Email: "test_brokersubuser@example.com", Tag: "Thrasher"}
		assert.Equal(t, exp, result, "CreateBrokerSubAccount should decode every field")
		return
	}
	assert.NotEmpty(t, result.SubAccountID, "CreateBrokerSubAccount should return a sub-account ID")
}

func TestCreateAPIKeyForSubAccount(t *testing.T) {
	t.Parallel()
	_, err := e.CreateAPIKeyForSubAccount(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateAPIKeyForSubAccount must reject a nil request")
	_, err = e.CreateAPIKeyForSubAccount(t.Context(), &BrokerSubAccountAPIKeyRequest{CanTrade: true})
	require.ErrorIs(t, err, errSubAccountIDMissing, "CreateAPIKeyForSubAccount must reject an empty sub-account ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateAPICredentials)
	}
	for _, tc := range []struct {
		name string
		arg  *BrokerSubAccountAPIKeyRequest
		exp  *BrokerSubAccountAPIKeyResponse
	}{
		{
			name: "HMAC",
			arg:  &BrokerSubAccountAPIKeyRequest{SubAccountID: "1", CanTrade: true, MarginTrade: true, FuturesTrade: true},
			exp: &BrokerSubAccountAPIKeyResponse{
				SubAccountID: "1",
				APIKey:       testBrokerSubAccountAPIKey,
				SecretKey:    "NhqPtmdSJYdKjVHjA7PZj4Mge3R5YNiP1e3UZjInClVN65XAbvqqM6A7H5fATj0j",
				CanTrade:     true,
				MarginTrade:  true,
				FuturesTrade: true,
			},
		},
		{
			name: "Ed25519",
			arg:  &BrokerSubAccountAPIKeyRequest{SubAccountID: "1", PublicKey: "MCowBQYDK2VwAyEAtestpublickeyplaceholder0000000000000000000000"},
			exp:  &BrokerSubAccountAPIKeyResponse{SubAccountID: "1", APIKey: "k5V49ldtn4tszj6W3hystegdfvmGbqDzjmkCtpTvC0G74WhK7yd4rfCTo4lShf"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.CreateAPIKeyForSubAccount(t.Context(), tc.arg)
			require.NoError(t, err, "CreateAPIKeyForSubAccount must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "CreateAPIKeyForSubAccount should decode every field")
				return
			}
			assert.NotEmpty(t, result.APIKey, "CreateAPIKeyForSubAccount should return an API key")
		})
	}
}

func TestDeleteSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	err := e.DeleteSubAccountAPIKey(t.Context(), "", testBrokerSubAccountAPIKey)
	require.ErrorIs(t, err, errSubAccountIDMissing, "DeleteSubAccountAPIKey must reject an empty sub-account ID")
	err = e.DeleteSubAccountAPIKey(t.Context(), "1", "")
	require.ErrorIs(t, err, errEmptySubAccountAPIKey, "DeleteSubAccountAPIKey must reject an empty API key")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateAPICredentials)
	}
	err = e.DeleteSubAccountAPIKey(t.Context(), "1", testBrokerSubAccountAPIKey)
	assert.NoError(t, err, "DeleteSubAccountAPIKey should not error")
}

func TestDeleteIPRestrictionForSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	_, err := e.DeleteIPRestrictionForSubAccountAPIKey(t.Context(), "", testBrokerSubAccountAPIKey, "192.0.2.1")
	require.ErrorIs(t, err, errSubAccountIDMissing, "DeleteIPRestrictionForSubAccountAPIKey must reject an empty sub-account ID")
	_, err = e.DeleteIPRestrictionForSubAccountAPIKey(t.Context(), "1", "", "192.0.2.1")
	require.ErrorIs(t, err, errEmptySubAccountAPIKey, "DeleteIPRestrictionForSubAccountAPIKey must reject an empty API key")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateAPICredentials)
	}
	result, err := e.DeleteIPRestrictionForSubAccountAPIKey(t.Context(), "1", testBrokerSubAccountAPIKey, "192.0.2.1")
	require.NoError(t, err, "DeleteIPRestrictionForSubAccountAPIKey must not error")
	if mockTests {
		exp := &BrokerSubAccountIPRestrictionResponse{
			SubAccountID: "1",
			APIKey:       testBrokerSubAccountAPIKey,
			IPList:       []string{"192.0.2.2", "192.0.2.3"},
			UpdateTime:   types.Time(time.UnixMilli(1744150000000)),
		}
		assert.Equal(t, exp, result, "DeleteIPRestrictionForSubAccountAPIKey should decode every field")
		return
	}
	assert.NotContains(t, result.IPList, "192.0.2.1", "DeleteIPRestrictionForSubAccountAPIKey should remove the IP")
}

func TestEnableFuturesForSubAccount(t *testing.T) {
	t.Parallel()
	_, err := e.EnableFuturesForSubAccount(t.Context(), "")
	require.ErrorIs(t, err, errSubAccountIDMissing, "EnableFuturesForSubAccount must reject an empty sub-account ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.EnableFuturesForSubAccount(t.Context(), "1")
	require.NoError(t, err, "EnableFuturesForSubAccount must not error")
	if mockTests {
		exp := &BrokerEnableFuturesResponse{SubAccountID: "1", EnableFutures: true, UpdateTime: types.Time(time.UnixMilli(1570801523523))}
		assert.Equal(t, exp, result, "EnableFuturesForSubAccount should decode every field")
		return
	}
	assert.True(t, result.EnableFutures, "EnableFuturesForSubAccount should enable futures")
}

func TestEnableOrDisableBNBBurnForSubAccountMarginInterest(t *testing.T) {
	t.Parallel()
	_, err := e.EnableOrDisableBNBBurnForSubAccountMarginInterest(t.Context(), "", true)
	require.ErrorIs(t, err, errSubAccountIDMissing, "EnableOrDisableBNBBurnForSubAccountMarginInterest must reject an empty sub-account ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.EnableOrDisableBNBBurnForSubAccountMarginInterest(t.Context(), "1", true)
	require.NoError(t, err, "EnableOrDisableBNBBurnForSubAccountMarginInterest must not error")
	if mockTests {
		exp := &BrokerBNBBurnMarginInterestResponse{SubAccountID: 1, InterestBNBBurn: true}
		assert.Equal(t, exp, result, "EnableOrDisableBNBBurnForSubAccountMarginInterest should decode every field")
		return
	}
	assert.True(t, result.InterestBNBBurn, "EnableOrDisableBNBBurnForSubAccountMarginInterest should enable BNB burn")
}

func TestEnableOrDisableBNBBurnForSubAccountSpotAndMargin(t *testing.T) {
	t.Parallel()
	_, err := e.EnableOrDisableBNBBurnForSubAccountSpotAndMargin(t.Context(), "", true)
	require.ErrorIs(t, err, errSubAccountIDMissing, "EnableOrDisableBNBBurnForSubAccountSpotAndMargin must reject an empty sub-account ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.EnableOrDisableBNBBurnForSubAccountSpotAndMargin(t.Context(), "1", true)
	require.NoError(t, err, "EnableOrDisableBNBBurnForSubAccountSpotAndMargin must not error")
	if mockTests {
		exp := &BrokerBNBBurnSpotResponse{SubAccountID: 1, SpotBNBBurn: true}
		assert.Equal(t, exp, result, "EnableOrDisableBNBBurnForSubAccountSpotAndMargin should decode every field")
		return
	}
	assert.True(t, result.SpotBNBBurn, "EnableOrDisableBNBBurnForSubAccountSpotAndMargin should enable BNB burn")
}

func TestEnableUniversalTransferPermissionForSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	_, err := e.EnableUniversalTransferPermissionForSubAccountAPIKey(t.Context(), "", testBrokerSubAccountAPIKey, true)
	require.ErrorIs(t, err, errSubAccountIDMissing, "EnableUniversalTransferPermissionForSubAccountAPIKey must reject an empty sub-account ID")
	_, err = e.EnableUniversalTransferPermissionForSubAccountAPIKey(t.Context(), "1", "", true)
	require.ErrorIs(t, err, errEmptySubAccountAPIKey, "EnableUniversalTransferPermissionForSubAccountAPIKey must reject an empty API key")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateAPICredentials)
	}
	result, err := e.EnableUniversalTransferPermissionForSubAccountAPIKey(t.Context(), "1", testBrokerSubAccountAPIKey, true)
	require.NoError(t, err, "EnableUniversalTransferPermissionForSubAccountAPIKey must not error")
	if mockTests {
		exp := &BrokerUniversalTransferPermissionResponse{SubAccountID: "1", APIKey: testBrokerSubAccountAPIKey, CanUniversalTransfer: true}
		assert.Equal(t, exp, result, "EnableUniversalTransferPermissionForSubAccountAPIKey should decode every field")
		return
	}
	assert.True(t, result.CanUniversalTransfer, "EnableUniversalTransferPermissionForSubAccountAPIKey should grant the permission")
}

func TestGetBNBBurnStatusForSubAccount(t *testing.T) {
	t.Parallel()
	_, err := e.GetBNBBurnStatusForSubAccount(t.Context(), "")
	require.ErrorIs(t, err, errSubAccountIDMissing, "GetBNBBurnStatusForSubAccount must reject an empty sub-account ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetBNBBurnStatusForSubAccount(t.Context(), "1")
	require.NoError(t, err, "GetBNBBurnStatusForSubAccount must not error")
	if mockTests {
		exp := &BrokerBNBBurnStatusResponse{SubAccountID: 1, SpotBNBBurn: true, InterestBNBBurn: true}
		assert.Equal(t, exp, result, "GetBNBBurnStatusForSubAccount should decode every field")
		return
	}
	assert.NotNil(t, result, "GetBNBBurnStatusForSubAccount should return a result")
}

func TestLinkAccountInformation(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.LinkAccountInformation(t.Context())
	require.NoError(t, err, "LinkAccountInformation must not error")
	if mockTests {
		exp := &LinkAccountInformationResponse{
			MaxMakerCommission:    0.002,
			MinMakerCommission:    0.001,
			MaxTakerCommission:    0.002,
			MinTakerCommission:    0.001,
			SubAccountQuantity:    400,
			MaxSubAccountQuantity: 1000,
		}
		assert.Equal(t, exp, result, "LinkAccountInformation should decode every field")
		return
	}
	assert.Positive(t, result.MaxSubAccountQuantity, "LinkAccountInformation should return the sub-account allowance")
}

func TestUpdateIPRestrictionForSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateIPRestrictionForSubAccountAPIKey(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "UpdateIPRestrictionForSubAccountAPIKey must reject a nil request")
	arg := &BrokerIPRestrictionUpdateRequest{SubAccountAPIKey: testBrokerSubAccountAPIKey, Status: 2, IPAddress: "192.0.2.1"}
	_, err = e.UpdateIPRestrictionForSubAccountAPIKey(t.Context(), arg)
	require.ErrorIs(t, err, errSubAccountIDMissing, "UpdateIPRestrictionForSubAccountAPIKey must reject an empty sub-account ID")
	arg.SubAccountID, arg.SubAccountAPIKey = "1", ""
	_, err = e.UpdateIPRestrictionForSubAccountAPIKey(t.Context(), arg)
	require.ErrorIs(t, err, errEmptySubAccountAPIKey, "UpdateIPRestrictionForSubAccountAPIKey must reject an empty API key")
	arg.SubAccountAPIKey, arg.Status = testBrokerSubAccountAPIKey, 0
	_, err = e.UpdateIPRestrictionForSubAccountAPIKey(t.Context(), arg)
	require.ErrorIs(t, err, errSubAccountStatusMissing, "UpdateIPRestrictionForSubAccountAPIKey must reject an empty status")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateAPICredentials)
	}
	arg.Status = 2
	result, err := e.UpdateIPRestrictionForSubAccountAPIKey(t.Context(), arg)
	require.NoError(t, err, "UpdateIPRestrictionForSubAccountAPIKey must not error")
	if mockTests {
		exp := &BrokerIPRestrictionUpdateResponse{
			Status:     "2",
			IPList:     []string{"192.0.2.1"},
			UpdateTime: types.Time(time.UnixMilli(1636371437000)),
			APIKey:     testBrokerSubAccountAPIKey,
		}
		assert.Equal(t, exp, result, "UpdateIPRestrictionForSubAccountAPIKey should decode every field")
		return
	}
	assert.Contains(t, result.IPList, "192.0.2.1", "UpdateIPRestrictionForSubAccountAPIKey should add the IP")
}

func TestGetSubAccountDepositHistoryWithBroker(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAccountDepositHistoryWithBroker(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSubAccountDepositHistoryWithBroker must reject a nil request")
	startTime, endTime := getTime()
	arg := &BrokerSubAccountDepositHistoryRequest{Coin: currency.BTC, StartTime: endTime, EndTime: startTime, Limit: 10}
	_, err = e.GetSubAccountDepositHistoryWithBroker(t.Context(), arg)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSubAccountDepositHistoryWithBroker must reject a reversed window")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	arg.StartTime, arg.EndTime = startTime, endTime
	result, err := e.GetSubAccountDepositHistoryWithBroker(t.Context(), arg)
	require.NoError(t, err, "GetSubAccountDepositHistoryWithBroker must not error")
	if mockTests {
		exp := []BrokerSubAccountDeposit{
			{
				DepositID:     1234567890123,
				SubAccountID:  "1",
				Address:       "bc1qexampleaddress0000000000000000000000",
				Amount:        0.5,
				Coin:          currency.BTC,
				InsertTime:    types.Time(time.Unix(1744150000, 0)),
				Network:       "BTC",
				Status:        1,
				TransactionID: "5759dfe9983a4c7619bce9bc736bb6c26f804091753bf66fa91e7cd5cfeebafd",
				SourceAddress: "bc1qsourceaddress000000000000000000000000",
				ConfirmTimes:  "2/2",
			},
			{
				DepositID:        1234567890124,
				SubAccountID:     "2",
				Address:          "bc1qexampleaddress0000000000000000000001",
				AddressTag:       "memo",
				Amount:           0.1,
				Coin:             currency.BTC,
				InsertTime:       types.Time(time.Unix(1744160000, 0)),
				TransferType:     1,
				Network:          "BTC",
				Status:           6,
				TransactionID:    "Off-chain transfer 1234567890",
				SourceAddress:    "bc1qsourceaddress000000000000000000000001",
				ConfirmTimes:     "1/2",
				SelfReturnStatus: 1,
				TravelRuleStatus: 1,
			},
		}
		assert.Equal(t, exp, result, "GetSubAccountDepositHistoryWithBroker should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSubAccountDepositHistoryWithBroker should return a result")
}

func TestGetSubAccountFuturesAssetInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAccountFuturesAssetInfo(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSubAccountFuturesAssetInfo must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSubAccountFuturesAssetInfo(t.Context(), &BrokerFuturesAssetInfoRequest{SubAccountID: "1", CoinMargined: true, Page: 1, Size: 10})
	require.NoError(t, err, "GetSubAccountFuturesAssetInfo must not error")
	if mockTests {
		exp := &BrokerFuturesAssetInfoResponse{
			Data: []BrokerFuturesAsset{
				{
					SubAccountID:                "1",
					TotalInitialMargin:          0.001,
					TotalMaintenanceMargin:      0.0005,
					TotalWalletBalance:          0.01,
					TotalUnrealizedProfit:       -0.0001,
					TotalMarginBalance:          0.0099,
					TotalPositionInitialMargin:  0.0008,
					TotalOpenOrderInitialMargin: 0.0002,
					FuturesEnable:               true,
					Asset:                       currency.BTC,
					TotalWalletBalanceOfUSDT:    781,
					TotalUnrealizedProfitOfUSDT: -7.81,
					TotalMarginBalanceOfUSDT:    773.19,
				},
			},
			Timestamp: types.Time(time.Unix(1744150000, 0)),
		}
		assert.Equal(t, exp, result, "GetSubAccountFuturesAssetInfo should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSubAccountFuturesAssetInfo should return a result")
}

func TestGetSubAccountMarginAssetInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAccountMarginAssetInfo(t.Context(), "", 1, 0)
	require.ErrorIs(t, err, errSubAccountIDOrSizeRequired, "GetSubAccountMarginAssetInfo must reject a missing sub-account ID and size")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSubAccountMarginAssetInfo(t.Context(), "", 1, 20)
	require.NoError(t, err, "GetSubAccountMarginAssetInfo must not error")
	if mockTests {
		exp := &BrokerMarginAssetInfoResponse{
			Data: []BrokerMarginAsset{
				{
					MarginEnable:        true,
					SubAccountID:        "367537027503425913",
					TotalAssetOfBTC:     0.0355852,
					TotalLiabilityOfBTC: 0.0158521,
					TotalNetAssetOfBTC:  0.0197331,
					MarginLevel:         2.244812,
				},
			},
			Timestamp: types.Time(time.Unix(1583127900, 0)),
		}
		assert.Equal(t, exp, result, "GetSubAccountMarginAssetInfo should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSubAccountMarginAssetInfo should return a result")
}

func TestGetSubAccountSpotAssetInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAccountSpotAssetInfo(t.Context(), "", 1, 0)
	require.ErrorIs(t, err, errSubAccountIDOrSizeRequired, "GetSubAccountSpotAssetInfo must reject a missing sub-account ID and size")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSubAccountSpotAssetInfo(t.Context(), "1", 1, 10)
	require.NoError(t, err, "GetSubAccountSpotAssetInfo must not error")
	if mockTests {
		exp := &BrokerSpotAssetInfoResponse{
			Data: []BrokerSpotAsset{
				{
					SubAccountID:      "1",
					TotalBalanceOfBTC: 0.035585215436,
				},
			},
			Timestamp: types.Time(time.Unix(1583432900, 0)),
		}
		assert.Equal(t, exp, result, "GetSubAccountSpotAssetInfo should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSubAccountSpotAssetInfo should return a result")
}

func TestGetFuturesBrokerSubAccountTransferHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesBrokerSubAccountTransferHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFuturesBrokerSubAccountTransferHistory must reject a nil request")
	startTime, endTime := getTime()
	arg := &BrokerFuturesTransferHistoryRequest{StartTime: endTime, EndTime: startTime, Limit: 100}
	_, err = e.GetFuturesBrokerSubAccountTransferHistory(t.Context(), arg)
	require.ErrorIs(t, err, errSubAccountIDMissing, "GetFuturesBrokerSubAccountTransferHistory must reject an empty sub-account ID")
	arg.SubAccountID = "1"
	_, err = e.GetFuturesBrokerSubAccountTransferHistory(t.Context(), arg)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFuturesBrokerSubAccountTransferHistory must reject a reversed window")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	arg.StartTime, arg.EndTime = startTime, endTime
	result, err := e.GetFuturesBrokerSubAccountTransferHistory(t.Context(), arg)
	require.NoError(t, err, "GetFuturesBrokerSubAccountTransferHistory must not error")
	if mockTests {
		exp := &BrokerFuturesTransferHistoryResponse{
			Success:     true,
			FuturesType: 1,
			Transfers: []BrokerFuturesTransfer{
				{
					From:             "1",
					To:               "2",
					Asset:            currency.USDT,
					Quantity:         10,
					TransferID:       "12137888538",
					ClientTransferID: "a123",
					Time:             types.Time(time.Unix(1744150000, 0)),
					FromID:           "1",
					ToID:             "2",
				},
			},
		}
		assert.Equal(t, exp, result, "GetFuturesBrokerSubAccountTransferHistory should decode every field")
		return
	}
	assert.True(t, result.Success, "GetFuturesBrokerSubAccountTransferHistory should succeed")
}

func TestSubAccountTransferWithFuturesBroker(t *testing.T) {
	t.Parallel()
	_, err := e.SubAccountTransferWithFuturesBroker(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SubAccountTransferWithFuturesBroker must reject a nil request")
	arg := &BrokerFuturesTransferRequest{ToID: "1", Amount: 1, ClientTransferID: "a123"}
	_, err = e.SubAccountTransferWithFuturesBroker(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "SubAccountTransferWithFuturesBroker must reject an empty asset")
	arg.Asset, arg.Amount = currency.BTC, 0
	_, err = e.SubAccountTransferWithFuturesBroker(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "SubAccountTransferWithFuturesBroker must reject an empty amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	arg.Amount = 1
	result, err := e.SubAccountTransferWithFuturesBroker(t.Context(), arg)
	require.NoError(t, err, "SubAccountTransferWithFuturesBroker must not error")
	if mockTests {
		exp := &BrokerFuturesTransferResponse{Success: true, TransactionID: "2966662589", ClientTransferID: "a123"}
		assert.Equal(t, exp, result, "SubAccountTransferWithFuturesBroker should decode every field")
		return
	}
	assert.True(t, result.Success, "SubAccountTransferWithFuturesBroker should succeed")
}

func TestGetSpotBrokerSubAccountTransferHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotBrokerSubAccountTransferHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSpotBrokerSubAccountTransferHistory must reject a nil request")
	startTime, endTime := getTime()
	arg := &BrokerSpotTransferHistoryRequest{ShowAllStatus: true, StartTime: endTime, EndTime: startTime, Page: 1, Limit: 100}
	_, err = e.GetSpotBrokerSubAccountTransferHistory(t.Context(), arg)
	require.ErrorIs(t, err, errFromOrToIDRequired, "GetSpotBrokerSubAccountTransferHistory must reject missing account IDs")
	arg.FromID = "1"
	_, err = e.GetSpotBrokerSubAccountTransferHistory(t.Context(), arg)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSpotBrokerSubAccountTransferHistory must reject a reversed window")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	arg.StartTime, arg.EndTime = startTime, endTime
	result, err := e.GetSpotBrokerSubAccountTransferHistory(t.Context(), arg)
	require.NoError(t, err, "GetSpotBrokerSubAccountTransferHistory must not error")
	if mockTests {
		exp := []BrokerSpotTransfer{
			{
				FromID:           "1",
				ToID:             "2",
				Asset:            currency.BTC,
				Quantity:         1,
				Time:             types.Time(time.Unix(1744150000, 0)),
				TransactionID:    "2966662589",
				ClientTransferID: "abc",
				Status:           "SUCCESS",
			},
		}
		assert.Equal(t, exp, result, "GetSpotBrokerSubAccountTransferHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSpotBrokerSubAccountTransferHistory should return a result")
}

func TestSubAccountTransferWithSpotBroker(t *testing.T) {
	t.Parallel()
	_, err := e.SubAccountTransferWithSpotBroker(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SubAccountTransferWithSpotBroker must reject a nil request")
	arg := &BrokerSpotTransferRequest{ToID: "1", Amount: 13, ClientTransferID: "abc"}
	_, err = e.SubAccountTransferWithSpotBroker(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "SubAccountTransferWithSpotBroker must reject an empty asset")
	arg.Asset, arg.Amount = currency.BTC, 0
	_, err = e.SubAccountTransferWithSpotBroker(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "SubAccountTransferWithSpotBroker must reject an empty amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	arg.Amount = 13
	result, err := e.SubAccountTransferWithSpotBroker(t.Context(), arg)
	require.NoError(t, err, "SubAccountTransferWithSpotBroker must not error")
	if mockTests {
		exp := &BrokerSpotTransferResponse{TransactionID: "2966662590", ClientTransferID: "abc"}
		assert.Equal(t, exp, result, "SubAccountTransferWithSpotBroker should decode every field")
		return
	}
	assert.NotEmpty(t, result.TransactionID, "SubAccountTransferWithSpotBroker should return a transaction ID")
}

func TestGetUniversalTransferHistoryThroughBroker(t *testing.T) {
	t.Parallel()
	_, err := e.GetUniversalTransferHistoryThroughBroker(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUniversalTransferHistoryThroughBroker must reject a nil request")
	startTime, endTime := getTime()
	arg := &BrokerUniversalTransferHistoryRequest{StartTime: endTime, EndTime: startTime, Page: 1, Limit: 100}
	_, err = e.GetUniversalTransferHistoryThroughBroker(t.Context(), arg)
	require.ErrorIs(t, err, errFromOrToIDRequired, "GetUniversalTransferHistoryThroughBroker must reject missing account IDs")
	arg.FromID = "1"
	_, err = e.GetUniversalTransferHistoryThroughBroker(t.Context(), arg)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUniversalTransferHistoryThroughBroker must reject a reversed window")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	arg.StartTime, arg.EndTime = startTime, endTime
	result, err := e.GetUniversalTransferHistoryThroughBroker(t.Context(), arg)
	require.NoError(t, err, "GetUniversalTransferHistoryThroughBroker must not error")
	if mockTests {
		exp := []BrokerUniversalTransfer{
			{
				FromID:           "1",
				ToID:             "444016824578949121",
				Asset:            currency.BTC,
				Quantity:         0.1,
				Time:             types.Time(time.Unix(1744150000, 0)),
				Status:           "SUCCESS",
				TransactionID:    "12831078279",
				ClientTransferID: "abc",
				FromAccountType:  "SPOT",
				ToAccountType:    "USDT_FUTURE",
			},
		}
		assert.Equal(t, exp, result, "GetUniversalTransferHistoryThroughBroker should decode every field")
		return
	}
	assert.NotNil(t, result, "GetUniversalTransferHistoryThroughBroker should return a result")
}

func TestUniversalTransferWithBroker(t *testing.T) {
	t.Parallel()
	_, err := e.UniversalTransferWithBroker(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "UniversalTransferWithBroker must reject a nil request")
	arg := &BrokerUniversalTransferRequest{ToAccountType: "USDT_FUTURE", ToID: "1", ClientTransferID: "abc", Asset: currency.BTC, Amount: 1}
	_, err = e.UniversalTransferWithBroker(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidAccountType, "UniversalTransferWithBroker must reject an empty from account type")
	arg.FromAccountType, arg.ToAccountType = "SPOT", ""
	_, err = e.UniversalTransferWithBroker(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidAccountType, "UniversalTransferWithBroker must reject an empty to account type")
	arg.ToAccountType, arg.Asset = "USDT_FUTURE", currency.EMPTYCODE
	_, err = e.UniversalTransferWithBroker(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "UniversalTransferWithBroker must reject an empty asset")
	arg.Asset, arg.Amount = currency.BTC, 0
	_, err = e.UniversalTransferWithBroker(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "UniversalTransferWithBroker must reject an empty amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	arg.Amount = 1
	result, err := e.UniversalTransferWithBroker(t.Context(), arg)
	require.NoError(t, err, "UniversalTransferWithBroker must not error")
	if mockTests {
		exp := &BrokerUniversalTransferResponse{TransactionID: 12831061179, ClientTransferID: "abc"}
		assert.Equal(t, exp, result, "UniversalTransferWithBroker should decode every field")
		return
	}
	assert.NotZero(t, result.TransactionID, "UniversalTransferWithBroker should return a transaction ID")
}

func TestGetSubAccountCoinMarginedFuturesCommissionAdjustment(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAccountCoinMarginedFuturesCommissionAdjustment(t.Context(), "", "BTCUSD")
	require.ErrorIs(t, err, errSubAccountIDMissing, "GetSubAccountCoinMarginedFuturesCommissionAdjustment must reject an empty sub-account ID")
	_, err = e.GetSubAccountCoinMarginedFuturesCommissionAdjustment(t.Context(), "1", "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetSubAccountCoinMarginedFuturesCommissionAdjustment must reject an empty pair")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSubAccountCoinMarginedFuturesCommissionAdjustment(t.Context(), "1", "BTCUSD")
	require.NoError(t, err, "GetSubAccountCoinMarginedFuturesCommissionAdjustment must not error")
	if mockTests {
		exp := []BrokerCoinFuturesCommission{{SubAccountID: 1, Pair: "BTCUSD", MakerCommission: 450, TakerCommission: 550}}
		assert.Equal(t, exp, result, "GetSubAccountCoinMarginedFuturesCommissionAdjustment should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSubAccountCoinMarginedFuturesCommissionAdjustment should return a result")
}

func TestChangeSubAccountCoinMarginedFuturesCommissionAdjustment(t *testing.T) {
	t.Parallel()
	_, err := e.ChangeSubAccountCoinMarginedFuturesCommissionAdjustment(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ChangeSubAccountCoinMarginedFuturesCommissionAdjustment must reject a nil request")
	arg := &BrokerCoinFuturesCommissionAdjustmentRequest{Pair: "BTCUSD", MakerAdjustment: 50, TakerAdjustment: 150}
	_, err = e.ChangeSubAccountCoinMarginedFuturesCommissionAdjustment(t.Context(), arg)
	require.ErrorIs(t, err, errSubAccountIDMissing, "ChangeSubAccountCoinMarginedFuturesCommissionAdjustment must reject an empty sub-account ID")
	arg.SubAccountID, arg.Pair = "1", ""
	_, err = e.ChangeSubAccountCoinMarginedFuturesCommissionAdjustment(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "ChangeSubAccountCoinMarginedFuturesCommissionAdjustment must reject an empty pair")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	arg.Pair = "BTCUSD"
	result, err := e.ChangeSubAccountCoinMarginedFuturesCommissionAdjustment(t.Context(), arg)
	require.NoError(t, err, "ChangeSubAccountCoinMarginedFuturesCommissionAdjustment must not error")
	if mockTests {
		exp := &BrokerCoinFuturesCommissionAdjustmentResponse{SubAccountID: 1, Pair: "BTCUSD", MakerAdjustment: 50, TakerAdjustment: 150, MakerCommission: 250, TakerCommission: 650}
		assert.Equal(t, exp, result, "ChangeSubAccountCoinMarginedFuturesCommissionAdjustment should decode every field")
		return
	}
	assert.Equal(t, uint64(50), result.MakerAdjustment, "ChangeSubAccountCoinMarginedFuturesCommissionAdjustment should set the maker adjustment")
}

func TestChangeSubAccountCommission(t *testing.T) {
	t.Parallel()
	_, err := e.ChangeSubAccountCommission(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ChangeSubAccountCommission must reject a nil request")
	arg := &BrokerCommissionRequest{MakerCommission: 0.001, TakerCommission: 0.002, MarginMakerCommission: 0.0015, MarginTakerCommission: 0.0018}
	_, err = e.ChangeSubAccountCommission(t.Context(), arg)
	require.ErrorIs(t, err, errSubAccountIDMissing, "ChangeSubAccountCommission must reject an empty sub-account ID")
	arg.SubAccountID, arg.MakerCommission = "1", 0
	_, err = e.ChangeSubAccountCommission(t.Context(), arg)
	require.ErrorIs(t, err, errCommissionValueRequired, "ChangeSubAccountCommission must reject an empty maker commission")
	arg.MakerCommission, arg.TakerCommission = 0.001, 0
	_, err = e.ChangeSubAccountCommission(t.Context(), arg)
	require.ErrorIs(t, err, errCommissionValueRequired, "ChangeSubAccountCommission must reject an empty taker commission")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	arg.TakerCommission = 0.002
	result, err := e.ChangeSubAccountCommission(t.Context(), arg)
	require.NoError(t, err, "ChangeSubAccountCommission must not error")
	if mockTests {
		exp := &BrokerCommissionResponse{SubAccountID: "1", MakerCommission: 0.001, TakerCommission: 0.002, MarginMakerCommission: 0.0015, MarginTakerCommission: 0.0018}
		assert.Equal(t, exp, result, "ChangeSubAccountCommission should decode every field")
		return
	}
	assert.Equal(t, 0.001, result.MakerCommission, "ChangeSubAccountCommission should set the maker commission")
}

func TestGetSubAccountUSDTMarginedFuturesCommissionAdjustment(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAccountUSDTMarginedFuturesCommissionAdjustment(t.Context(), "", currency.NewBTCUSDT())
	require.ErrorIs(t, err, errSubAccountIDMissing, "GetSubAccountUSDTMarginedFuturesCommissionAdjustment must reject an empty sub-account ID")
	_, err = e.GetSubAccountUSDTMarginedFuturesCommissionAdjustment(t.Context(), "1", currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetSubAccountUSDTMarginedFuturesCommissionAdjustment must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSubAccountUSDTMarginedFuturesCommissionAdjustment(t.Context(), "1", currency.NewBTCUSDT())
	require.NoError(t, err, "GetSubAccountUSDTMarginedFuturesCommissionAdjustment must not error")
	if mockTests {
		exp := []BrokerUSDTFuturesCommission{{SubAccountID: 1, Symbol: "BTCUSDT", MakerCommission: 450, TakerCommission: 550}}
		assert.Equal(t, exp, result, "GetSubAccountUSDTMarginedFuturesCommissionAdjustment should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSubAccountUSDTMarginedFuturesCommissionAdjustment should return a result")
}

func TestChangeSubAccountUSDTMarginedFuturesCommissionAdjustment(t *testing.T) {
	t.Parallel()
	_, err := e.ChangeSubAccountUSDTMarginedFuturesCommissionAdjustment(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ChangeSubAccountUSDTMarginedFuturesCommissionAdjustment must reject a nil request")
	arg := &BrokerUSDTFuturesCommissionAdjustmentRequest{Symbol: currency.NewBTCUSDT(), MakerAdjustment: 1, TakerAdjustment: 10}
	_, err = e.ChangeSubAccountUSDTMarginedFuturesCommissionAdjustment(t.Context(), arg)
	require.ErrorIs(t, err, errSubAccountIDMissing, "ChangeSubAccountUSDTMarginedFuturesCommissionAdjustment must reject an empty sub-account ID")
	arg.SubAccountID, arg.Symbol = "1", currency.EMPTYPAIR
	_, err = e.ChangeSubAccountUSDTMarginedFuturesCommissionAdjustment(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "ChangeSubAccountUSDTMarginedFuturesCommissionAdjustment must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	arg.Symbol = currency.NewBTCUSDT()
	result, err := e.ChangeSubAccountUSDTMarginedFuturesCommissionAdjustment(t.Context(), arg)
	require.NoError(t, err, "ChangeSubAccountUSDTMarginedFuturesCommissionAdjustment must not error")
	if mockTests {
		exp := &BrokerUSDTFuturesCommissionAdjustmentResponse{SubAccountID: 1, Symbol: "BTCUSDT", MakerAdjustment: 1, TakerAdjustment: 10, MakerCommission: 201, TakerCommission: 510}
		assert.Equal(t, exp, result, "ChangeSubAccountUSDTMarginedFuturesCommissionAdjustment should decode every field")
		return
	}
	assert.Equal(t, uint64(1), result.MakerAdjustment, "ChangeSubAccountUSDTMarginedFuturesCommissionAdjustment should set the maker adjustment")
}

func TestGetSpotBrokerCommissionRebateRecentRecord(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotBrokerCommissionRebateRecentRecord(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSpotBrokerCommissionRebateRecentRecord must reject a nil request")
	startTime, endTime := getTime()
	arg := &BrokerSpotCommissionRebateRequest{SubAccountID: "1", StartTime: endTime, EndTime: startTime, Page: 1, Size: 100}
	_, err = e.GetSpotBrokerCommissionRebateRecentRecord(t.Context(), arg)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSpotBrokerCommissionRebateRecentRecord must reject a reversed window")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	arg.StartTime, arg.EndTime = startTime, endTime
	result, err := e.GetSpotBrokerCommissionRebateRecentRecord(t.Context(), arg)
	require.NoError(t, err, "GetSpotBrokerCommissionRebateRecentRecord must not error")
	if mockTests {
		exp := []BrokerCommissionRebate{
			{
				SubAccountID: "1",
				Income:       0.02063898,
				Asset:        currency.BTC,
				Symbol:       "ETHBTC",
				TradeID:      123456,
				Time:         types.Time(time.Unix(1744150000, 0)),
				Status:       1,
			},
		}
		assert.Equal(t, exp, result, "GetSpotBrokerCommissionRebateRecentRecord should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSpotBrokerCommissionRebateRecentRecord should return a result")
}

func TestGetFuturesBrokerCommissionRebateRecentRecord(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesBrokerCommissionRebateRecentRecord(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFuturesBrokerCommissionRebateRecentRecord must reject a nil request")
	startTime, endTime := getTime()
	arg := &BrokerFuturesCommissionRebateRequest{EndTime: endTime, Size: 10}
	_, err = e.GetFuturesBrokerCommissionRebateRecentRecord(t.Context(), arg)
	require.ErrorIs(t, err, common.ErrDateUnset, "GetFuturesBrokerCommissionRebateRecentRecord must reject a missing start time")
	arg.StartTime, arg.EndTime = endTime, startTime
	_, err = e.GetFuturesBrokerCommissionRebateRecentRecord(t.Context(), arg)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFuturesBrokerCommissionRebateRecentRecord must reject a reversed window")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	arg.StartTime, arg.EndTime = startTime, endTime
	result, err := e.GetFuturesBrokerCommissionRebateRecentRecord(t.Context(), arg)
	require.NoError(t, err, "GetFuturesBrokerCommissionRebateRecentRecord must not error")
	if mockTests {
		exp := []BrokerCommissionRebate{
			{
				SubAccountID: "1",
				Income:       0.02063898,
				Asset:        currency.USDT,
				Symbol:       "ETHUSDT",
				TradeID:      123456,
				Time:         types.Time(time.Unix(1744150000, 0)),
				Status:       1,
			},
		}
		assert.Equal(t, exp, result, "GetFuturesBrokerCommissionRebateRecentRecord should decode every field")
		return
	}
	assert.NotNil(t, result, "GetFuturesBrokerCommissionRebateRecentRecord should return a result")
}
