package binance

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

func TestGetSpotUsersCustomisedID(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotUsersCustomisedID(t.Context(), "")
	require.ErrorIs(t, err, errAPIAgentCodeRequired, "GetSpotUsersCustomisedID must reject an empty API agent code")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSpotUsersCustomisedID(t.Context(), "1234ABCD")
	require.NoError(t, err, "GetSpotUsersCustomisedID must not error")
	if mockTests {
		assert.Equal(t, &SpotReferralCustomerIDResponse{CustomerID: "abc123"}, result, "GetSpotUsersCustomisedID should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSpotUsersCustomisedID should return a result")
}

func TestCustomiseSpotOwnClientID(t *testing.T) {
	t.Parallel()
	_, err := e.CustomiseSpotOwnClientID(t.Context(), "", "ABCDEFG")
	require.ErrorIs(t, err, errCustomerIDRequired, "CustomiseSpotOwnClientID must reject an empty customer ID")
	_, err = e.CustomiseSpotOwnClientID(t.Context(), "the-unique-id", "")
	require.ErrorIs(t, err, errAPIAgentCodeRequired, "CustomiseSpotOwnClientID must reject an empty API agent code")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CustomiseSpotOwnClientID(t.Context(), "the-unique-id", "ABCDEFG")
	require.NoError(t, err, "CustomiseSpotOwnClientID must not error")
	if mockTests {
		assert.Equal(t, &SpotReferralCustomerIDResponse{CustomerID: "the-unique-id"}, result, "CustomiseSpotOwnClientID should decode every field")
		return
	}
	assert.Equal(t, "the-unique-id", result.CustomerID, "CustomiseSpotOwnClientID should set the customer ID")
}

func TestGetSpotClientEmailCustomisedID(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotClientEmailCustomisedID(t.Context(), "abc123", "test@example.com")
	require.ErrorIs(t, err, errCustomerIDAndEmailExclusive, "GetSpotClientEmailCustomisedID must reject both filters")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSpotClientEmailCustomisedID(t.Context(), "", "")
	require.NoError(t, err, "GetSpotClientEmailCustomisedID must not error")
	if mockTests {
		exp := []ReferralCustomerEmail{{CustomerID: "abc123", Email: "test@example.com"}}
		assert.Equal(t, exp, result, "GetSpotClientEmailCustomisedID should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSpotClientEmailCustomisedID should return a result")
}

func TestCustomiseSpotPartnerClientID(t *testing.T) {
	t.Parallel()
	_, err := e.CustomiseSpotPartnerClientID(t.Context(), "", "test@example.com")
	require.ErrorIs(t, err, errCustomerIDRequired, "CustomiseSpotPartnerClientID must reject an empty customer ID")
	_, err = e.CustomiseSpotPartnerClientID(t.Context(), "1233", "")
	require.ErrorIs(t, err, errValidEmailRequired, "CustomiseSpotPartnerClientID must reject an empty email")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CustomiseSpotPartnerClientID(t.Context(), "1233", "test@example.com")
	require.NoError(t, err, "CustomiseSpotPartnerClientID must not error")
	if mockTests {
		assert.Equal(t, &ReferralCustomerEmailResponse{CustomerID: "1233", Email: "test@example.com"}, result, "CustomiseSpotPartnerClientID should decode every field")
		return
	}
	assert.Equal(t, "1233", result.CustomerID, "CustomiseSpotPartnerClientID should set the customer ID")
}

func TestGetSpotInfoAboutIfUserIsNew(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotInfoAboutIfUserIsNew(t.Context(), "")
	require.ErrorIs(t, err, errAPIAgentCodeRequired, "GetSpotInfoAboutIfUserIsNew must reject an empty API agent code")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSpotInfoAboutIfUserIsNew(t.Context(), "1234")
	require.NoError(t, err, "GetSpotInfoAboutIfUserIsNew must not error")
	if mockTests {
		exp := &SpotReferralNewUserResponse{APIAgentCode: "1234", RebateWorking: true, IfNewUser: true, ReferrerID: 39472261}
		assert.Equal(t, exp, result, "GetSpotInfoAboutIfUserIsNew should decode every field")
		return
	}
	assert.Equal(t, "1234", result.APIAgentCode, "GetSpotInfoAboutIfUserIsNew should return the API agent code")
}

func TestGetSpotOwnRebateRecentRecords(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetSpotOwnRebateRecentRecords(t.Context(), endTime, startTime, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSpotOwnRebateRecentRecords must reject a reversed window")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSpotOwnRebateRecentRecords(t.Context(), startTime, endTime, 10)
	require.NoError(t, err, "GetSpotOwnRebateRecentRecords must not error")
	if mockTests {
		exp := []ReferralRebateRecord{{Income: 0.02063898, Asset: currency.BTC, Symbol: "ETHBTC", Time: types.Time(time.UnixMilli(1744150000000))}}
		assert.Equal(t, exp, result, "GetSpotOwnRebateRecentRecords should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSpotOwnRebateRecentRecords should return a result")
}

func TestGetSpotOthersRebateRecentRecord(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotOthersRebateRecentRecord(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSpotOthersRebateRecentRecord must reject a nil request")
	startTime, endTime := getTime()
	arg := &PartnerRebateRecordRequest{CustomerID: "123123", EndTime: endTime, Limit: 10}
	_, err = e.GetSpotOthersRebateRecentRecord(t.Context(), arg)
	require.ErrorIs(t, err, common.ErrDateUnset, "GetSpotOthersRebateRecentRecord must reject a window with one bound")
	arg.StartTime, arg.EndTime = endTime, startTime
	_, err = e.GetSpotOthersRebateRecentRecord(t.Context(), arg)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSpotOthersRebateRecentRecord must reject a reversed window")
	arg.StartTime, arg.EndTime, arg.Limit = startTime, endTime, 0
	_, err = e.GetSpotOthersRebateRecentRecord(t.Context(), arg)
	require.ErrorIs(t, err, errLimitNumberRequired, "GetSpotOthersRebateRecentRecord must reject an empty limit")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	arg.Limit = 10
	result, err := e.GetSpotOthersRebateRecentRecord(t.Context(), arg)
	require.NoError(t, err, "GetSpotOthersRebateRecentRecord must not error")
	if mockTests {
		exp := []PartnerRebateRecord{
			{
				CustomerID:      "123123",
				Email:           "om***@***.com",
				Income:          0.00240635,
				Asset:           currency.USDC,
				Symbol:          "BNBUSDC",
				Time:            types.Time(time.UnixMilli(1744150000000)),
				OrderID:         54321678,
				TradeID:         12349876,
				DistributeTime:  types.Time(time.UnixMilli(1744154478000)),
				CommissionAsset: currency.BNB,
				Commission:      0.00000284,
				ConvertPrice:    847.3095241,
			},
		}
		assert.Equal(t, exp, result, "GetSpotOthersRebateRecentRecord should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSpotOthersRebateRecentRecord should return a result")
}

func TestGetFuturesUsersCustomisedID(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesUsersCustomisedID(t.Context(), "")
	require.ErrorIs(t, err, errInvalidBrokerID, "GetFuturesUsersCustomisedID must reject an empty broker ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesUsersCustomisedID(t.Context(), "1234ABCD")
	require.NoError(t, err, "GetFuturesUsersCustomisedID must not error")
	if mockTests {
		assert.Equal(t, &FuturesReferralCustomerIDResponse{BrokerID: "1234ABCD", CustomerID: "abc123"}, result, "GetFuturesUsersCustomisedID should decode every field")
		return
	}
	assert.Equal(t, "1234ABCD", result.BrokerID, "GetFuturesUsersCustomisedID should return the broker ID")
}

func TestCustomiseFuturesOwnClientID(t *testing.T) {
	t.Parallel()
	_, err := e.CustomiseFuturesOwnClientID(t.Context(), "", "ABCDEFG")
	require.ErrorIs(t, err, errCustomerIDRequired, "CustomiseFuturesOwnClientID must reject an empty customer ID")
	_, err = e.CustomiseFuturesOwnClientID(t.Context(), "the-unique-id", "")
	require.ErrorIs(t, err, errInvalidBrokerID, "CustomiseFuturesOwnClientID must reject an empty broker ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CustomiseFuturesOwnClientID(t.Context(), "the-unique-id", "ABCDEFG")
	require.NoError(t, err, "CustomiseFuturesOwnClientID must not error")
	if mockTests {
		assert.Equal(t, &FuturesReferralCustomerIDResponse{BrokerID: "ABCDEFG", CustomerID: "the-unique-id"}, result, "CustomiseFuturesOwnClientID should decode every field")
		return
	}
	assert.Equal(t, "the-unique-id", result.CustomerID, "CustomiseFuturesOwnClientID should set the customer ID")
}

func TestGetFuturesClientEmailCustomisedID(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesClientEmailCustomisedID(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFuturesClientEmailCustomisedID must reject a nil request")
	_, err = e.GetFuturesClientEmailCustomisedID(t.Context(), &ReferralCustomerEmailRequest{CustomerID: "abc123", Email: "test@example.com"})
	require.ErrorIs(t, err, errCustomerIDAndEmailExclusive, "GetFuturesClientEmailCustomisedID must reject both filters")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesClientEmailCustomisedID(t.Context(), &ReferralCustomerEmailRequest{})
	require.NoError(t, err, "GetFuturesClientEmailCustomisedID must not error")
	if mockTests {
		exp := []ReferralCustomerEmail{{CustomerID: "abc123", Email: "test@example.com"}}
		assert.Equal(t, exp, result, "GetFuturesClientEmailCustomisedID should decode every field")
		return
	}
	assert.NotNil(t, result, "GetFuturesClientEmailCustomisedID should return a result")
}

func TestCustomiseFuturesPartnerClientID(t *testing.T) {
	t.Parallel()
	_, err := e.CustomiseFuturesPartnerClientID(t.Context(), "", "test@example.com")
	require.ErrorIs(t, err, errCustomerIDRequired, "CustomiseFuturesPartnerClientID must reject an empty customer ID")
	_, err = e.CustomiseFuturesPartnerClientID(t.Context(), "1233", "")
	require.ErrorIs(t, err, errValidEmailRequired, "CustomiseFuturesPartnerClientID must reject an empty email")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CustomiseFuturesPartnerClientID(t.Context(), "1233", "test@example.com")
	require.NoError(t, err, "CustomiseFuturesPartnerClientID must not error")
	if mockTests {
		assert.Equal(t, &ReferralCustomerEmailResponse{CustomerID: "1233", Email: "test@example.com"}, result, "CustomiseFuturesPartnerClientID should decode every field")
		return
	}
	assert.Equal(t, "1233", result.CustomerID, "CustomiseFuturesPartnerClientID should set the customer ID")
}

func TestGetFuturesRebateDataOverview(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesRebateDataOverview(t.Context(), true)
	require.NoError(t, err, "GetFuturesRebateDataOverview must not error")
	if mockTests {
		exp := &ReferralRebateOverviewResponse{
			BrokerID:                  "ABCD1234",
			NewTraderRebateCommission: 0.3,
			OldTraderRebateCommission: 0.2,
			TotalTradeUser:            13,
			Unit:                      currency.BTC,
			TotalTradeVolume:          405.54379,
			TotalRebateVolume:         0.018338,
			Time:                      types.Time(time.UnixMilli(1744156800000)),
		}
		assert.Equal(t, exp, result, "GetFuturesRebateDataOverview should decode every field")
		return
	}
	assert.NotNil(t, result, "GetFuturesRebateDataOverview should return a result")
}

func TestGetRebateVolume(t *testing.T) {
	t.Parallel()
	_, err := e.GetRebateVolume(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetRebateVolume must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetRebateVolume(t.Context(), &ReferralVolumeRequest{StartTime: endTime, EndTime: startTime, Limit: 100})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetRebateVolume must reject a reversed window")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetRebateVolume(t.Context(), &ReferralVolumeRequest{StartTime: startTime, EndTime: endTime, Limit: 100})
	require.NoError(t, err, "GetRebateVolume must not error")
	if mockTests {
		exp := []ReferralRebateVolume{{Unit: currency.USDT, RebateVolume: 0.000232, Time: types.Time(time.UnixMilli(1744156800000))}}
		assert.Equal(t, exp, result, "GetRebateVolume should decode every field")
		return
	}
	assert.NotNil(t, result, "GetRebateVolume should return a result")
}

func TestGetTraderDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetTraderDetail(t.Context(), "sde001", nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetTraderDetail must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetTraderDetail(t.Context(), "sde001", &ReferralVolumeRequest{CoinMargined: true, StartTime: endTime, EndTime: startTime, Limit: 10})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetTraderDetail must reject a reversed window")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetTraderDetail(t.Context(), "sde001", &ReferralVolumeRequest{CoinMargined: true, StartTime: startTime, EndTime: endTime, Limit: 10})
	require.NoError(t, err, "GetTraderDetail must not error")
	if mockTests {
		exp := []ReferralTraderDetail{
			{CustomerID: "sde001", Unit: currency.BTC, TradeVolume: 9.57083, RebateVolume: 0.000354, Time: types.Time(time.UnixMilli(1744156800000))},
		}
		assert.Equal(t, exp, result, "GetTraderDetail should decode every field")
		return
	}
	assert.NotNil(t, result, "GetTraderDetail should return a result")
}

func TestGetFuturesReferredTradersNumber(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesReferredTradersNumber(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFuturesReferredTradersNumber must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetFuturesReferredTradersNumber(t.Context(), &ReferralVolumeRequest{StartTime: endTime, EndTime: startTime, Limit: 10})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFuturesReferredTradersNumber must reject a reversed window")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesReferredTradersNumber(t.Context(), &ReferralVolumeRequest{CoinMargined: true, StartTime: startTime, EndTime: endTime, Limit: 10})
	require.NoError(t, err, "GetFuturesReferredTradersNumber must not error")
	if mockTests {
		exp := []ReferralTraderNumber{{NewTrader: 1, OldTrader: 2, Time: types.Time(time.UnixMilli(1744156800000))}}
		assert.Equal(t, exp, result, "GetFuturesReferredTradersNumber should decode every field")
		return
	}
	assert.NotNil(t, result, "GetFuturesReferredTradersNumber should return a result")
}

func TestGetUserTradeVolume(t *testing.T) {
	t.Parallel()
	_, err := e.GetUserTradeVolume(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUserTradeVolume must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetUserTradeVolume(t.Context(), &ReferralVolumeRequest{StartTime: endTime, EndTime: startTime, Limit: 10})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUserTradeVolume must reject a reversed window")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUserTradeVolume(t.Context(), &ReferralVolumeRequest{CoinMargined: true, StartTime: startTime, EndTime: endTime, Limit: 100})
	require.NoError(t, err, "GetUserTradeVolume must not error")
	if mockTests {
		exp := []ReferralTradeVolume{{Unit: currency.BTC, TradeVolume: 53.60756, Time: types.Time(time.UnixMilli(1744156800000))}}
		assert.Equal(t, exp, result, "GetUserTradeVolume should decode every field")
		return
	}
	assert.NotNil(t, result, "GetUserTradeVolume should return a result")
}

func TestGetFuturesClientIfNewUser(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesClientIfNewUser(t.Context(), "", false)
	require.ErrorIs(t, err, errInvalidBrokerID, "GetFuturesClientIfNewUser must reject an empty broker ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesClientIfNewUser(t.Context(), "1234", false)
	require.NoError(t, err, "GetFuturesClientIfNewUser must not error")
	if mockTests {
		assert.Equal(t, &FuturesReferralNewUserResponse{BrokerID: "1234", RebateWorking: true, IfNewUser: true}, result, "GetFuturesClientIfNewUser should decode every field")
		return
	}
	assert.Equal(t, "1234", result.BrokerID, "GetFuturesClientIfNewUser should return the broker ID")
}

func TestGetPAPIClientIfNewUser(t *testing.T) {
	t.Parallel()
	_, err := e.GetPAPIClientIfNewUser(t.Context(), "", false)
	require.ErrorIs(t, err, errInvalidBrokerID, "GetPAPIClientIfNewUser must reject an empty broker ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, coinMargined := range []bool{false, true} {
		result, err := e.GetPAPIClientIfNewUser(t.Context(), "123123", coinMargined)
		require.NoErrorf(t, err, "GetPAPIClientIfNewUser must not error for coinMargined %t", coinMargined)
		if mockTests {
			exp := &PAPIClientNewUserStatusResponse{
				BrokerID:      "123123",
				RebateWorking: true,
				IfNewUser:     true,
			}
			assert.Equal(t, exp, result, "GetPAPIClientIfNewUser should decode every field")
			continue
		}
		assert.NotEmptyf(t, result.BrokerID, "GetPAPIClientIfNewUser should return the broker ID for coinMargined %t", coinMargined)
	}
}

func TestCustomisePAPIOwnClientID(t *testing.T) {
	t.Parallel()
	_, err := e.CustomisePAPIOwnClientID(t.Context(), "", "987654321")
	require.ErrorIs(t, err, errCustomerIDRequired, "CustomisePAPIOwnClientID must reject an empty customer ID")
	_, err = e.CustomisePAPIOwnClientID(t.Context(), "12345678", "")
	require.ErrorIs(t, err, errInvalidBrokerID, "CustomisePAPIOwnClientID must reject an empty broker ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CustomisePAPIOwnClientID(t.Context(), "12345678", "987654321")
	require.NoError(t, err, "CustomisePAPIOwnClientID must not error")
	if mockTests {
		exp := &PAPICustomisedIDResponse{
			BrokerID:   "987654321",
			CustomerID: "12345678",
		}
		assert.Equal(t, exp, result, "CustomisePAPIOwnClientID should decode every field")
		return
	}
	assert.Equal(t, "12345678", result.CustomerID, "CustomisePAPIOwnClientID should return the customer ID")
}

func TestGetPAPIUsersCustomisedID(t *testing.T) {
	t.Parallel()
	_, err := e.GetPAPIUsersCustomisedID(t.Context(), "")
	require.ErrorIs(t, err, errInvalidBrokerID, "GetPAPIUsersCustomisedID must reject an empty broker ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPAPIUsersCustomisedID(t.Context(), "12345678")
	require.NoError(t, err, "GetPAPIUsersCustomisedID must not error")
	if mockTests {
		exp := &PAPICustomisedIDResponse{
			BrokerID:   "12345678",
			CustomerID: "abc123",
		}
		assert.Equal(t, exp, result, "GetPAPIUsersCustomisedID should decode every field")
		return
	}
	assert.NotEmpty(t, result.BrokerID, "GetPAPIUsersCustomisedID should return the broker ID")
}
