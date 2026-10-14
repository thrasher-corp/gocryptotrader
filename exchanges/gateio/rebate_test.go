package gateio

import (
	"net/url"
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

// rebateTestPair is the currency pair the recorded rebate fixtures report
var rebateTestPair = currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter)

func TestRebateTransactionHistoryParams(t *testing.T) {
	t.Parallel()
	from := time.Unix(1710000000, 0)
	to := from.Add(time.Hour)

	params, err := rebateTransactionHistoryParams(nil)
	require.NoError(t, err, "rebateTransactionHistoryParams must not error for a nil request")
	assert.Empty(t, params, "rebateTransactionHistoryParams should not set parameters for a nil request")

	_, err = rebateTransactionHistoryParams(&RebateTransactionHistoryRequest{From: to, To: from})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "rebateTransactionHistoryParams must reject a start after the end")

	params, err = rebateTransactionHistoryParams(&RebateTransactionHistoryRequest{CurrencyPair: rebateTestPair, UserID: 10000001, From: from, To: to, Limit: 10, Offset: 5})
	require.NoError(t, err, "rebateTransactionHistoryParams must not error")
	exp := url.Values{
		"currency_pair": {"BTC_USDT"},
		"user_id":       {"10000001"},
		"from":          {strconv.FormatInt(from.Unix(), 10)},
		"to":            {strconv.FormatInt(to.Unix(), 10)},
		"limit":         {"10"},
		"offset":        {"5"},
	}
	assert.Equal(t, exp, params, "rebateTransactionHistoryParams should set every parameter")
}

func TestRebateCommissionHistoryParams(t *testing.T) {
	t.Parallel()
	from := time.Unix(1710000000, 0)
	to := from.Add(time.Hour)

	params, err := rebateCommissionHistoryParams(nil)
	require.NoError(t, err, "rebateCommissionHistoryParams must not error for a nil request")
	assert.Empty(t, params, "rebateCommissionHistoryParams should not set parameters for a nil request")

	_, err = rebateCommissionHistoryParams(&RebateCommissionHistoryRequest{From: to, To: from})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "rebateCommissionHistoryParams must reject a start after the end")

	params, err = rebateCommissionHistoryParams(&RebateCommissionHistoryRequest{Currency: currency.USDT, CommissionType: 2, UserID: 10000001, From: from, To: to, Limit: 10, Offset: 5})
	require.NoError(t, err, "rebateCommissionHistoryParams must not error")
	exp := url.Values{
		"currency":        {"USDT"},
		"commission_type": {"2"},
		"user_id":         {"10000001"},
		"from":            {strconv.FormatInt(from.Unix(), 10)},
		"to":              {strconv.FormatInt(to.Unix(), 10)},
		"limit":           {"10"},
		"offset":          {"5"},
	}
	assert.Equal(t, exp, params, "rebateCommissionHistoryParams should set every parameter")
}

func TestRebateBrokerHistoryParams(t *testing.T) {
	t.Parallel()
	from := time.Unix(1710000000, 0)
	to := from.Add(time.Hour)

	params, err := rebateBrokerHistoryParams(nil)
	require.NoError(t, err, "rebateBrokerHistoryParams must not error for a nil request")
	assert.Empty(t, params, "rebateBrokerHistoryParams should not set parameters for a nil request")

	_, err = rebateBrokerHistoryParams(&RebateBrokerHistoryRequest{From: to, To: from})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "rebateBrokerHistoryParams must reject a start after the end")

	params, err = rebateBrokerHistoryParams(&RebateBrokerHistoryRequest{UserID: 10000001, From: from, To: to, Limit: 10, Offset: 5})
	require.NoError(t, err, "rebateBrokerHistoryParams must not error")
	exp := url.Values{
		"user_id": {"10000001"},
		"from":    {strconv.FormatInt(from.Unix(), 10)},
		"to":      {strconv.FormatInt(to.Unix(), 10)},
		"limit":   {"10"},
		"offset":  {"5"},
	}
	assert.Equal(t, exp, params, "rebateBrokerHistoryParams should set every parameter")
}

func TestGetAgencyTransactionHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetAgencyTransactionHistory(t.Context(), &RebateTransactionHistoryRequest{From: endTime, To: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetAgencyTransactionHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAgencyTransactionHistory(t.Context(), &RebateTransactionHistoryRequest{CurrencyPair: rebateTestPair, From: startTime, To: endTime, Limit: 10})
	require.NoError(t, err, "GetAgencyTransactionHistory must not error")
	if mockTests {
		exp := &AgencyTransactionHistoryResponse{
			CurrencyPair: rebateTestPair,
			Total:        1,
			List: []*RebateTransaction{{
				TransactionTime: types.Time(time.Unix(1715864142, 0)),
				UserID:          10000001,
				GroupName:       "group_a",
				Fee:             0.65,
				FeeAsset:        currency.USDT,
				CurrencyPair:    rebateTestPair,
				Amount:          1500.5,
				AmountAsset:     currency.USDT,
				Source:          "SPOT",
			}},
		}
		assert.Equal(t, exp, result, "GetAgencyTransactionHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAgencyTransactionHistory should return a history")
}

func TestGetAgencyCommissionHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetAgencyCommissionHistory(t.Context(), &RebateCommissionHistoryRequest{From: endTime, To: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetAgencyCommissionHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAgencyCommissionHistory(t.Context(), &RebateCommissionHistoryRequest{Currency: currency.USDT, CommissionType: 1, From: startTime, To: endTime, Limit: 10})
	require.NoError(t, err, "GetAgencyCommissionHistory must not error")
	if mockTests {
		exp := &AgencyCommissionHistoryResponse{
			CurrencyPair: rebateTestPair,
			Total:        1,
			List: []*RebateCommission{{
				CommissionTime:   types.Time(time.Unix(1715864142, 0)),
				UserID:           10000001,
				GroupName:        "group_a",
				CommissionAmount: 12.5,
				CommissionAsset:  currency.USDT,
				Source:           "SPOT",
			}},
		}
		assert.Equal(t, exp, result, "GetAgencyCommissionHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAgencyCommissionHistory should return a history")
}

func TestGetPartnerTransactionHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetPartnerTransactionHistory(t.Context(), &RebateTransactionHistoryRequest{From: endTime, To: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetPartnerTransactionHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPartnerTransactionHistory(t.Context(), &RebateTransactionHistoryRequest{From: startTime, To: endTime, Limit: 10})
	require.NoError(t, err, "GetPartnerTransactionHistory must not error")
	if mockTests {
		exp := &PartnerTransactionHistoryResponse{
			Total: 1,
			List: []*RebateTransaction{{
				TransactionTime: types.Time(time.Unix(1715864142, 0)),
				UserID:          10000001,
				GroupName:       "group_a",
				Fee:             0.65,
				FeeAsset:        currency.USDT,
				CurrencyPair:    rebateTestPair,
				Amount:          1500.5,
				AmountAsset:     currency.USDT,
				Source:          "SPOT",
			}},
		}
		assert.Equal(t, exp, result, "GetPartnerTransactionHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPartnerTransactionHistory should return a history")
}

func TestGetPartnerCommissionHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetPartnerCommissionHistory(t.Context(), &RebateCommissionHistoryRequest{From: endTime, To: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetPartnerCommissionHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPartnerCommissionHistory(t.Context(), &RebateCommissionHistoryRequest{Currency: currency.USDT, UserID: 10100011213, From: startTime, To: endTime, Limit: 10})
	require.NoError(t, err, "GetPartnerCommissionHistory must not error")
	if mockTests {
		exp := &PartnerCommissionHistoryResponse{
			Total: 1,
			List: []*RebateCommission{{
				CommissionTime:   types.Time(time.Unix(1715864142, 0)),
				UserID:           10000001,
				GroupName:        "group_a",
				CommissionAmount: 12.5,
				CommissionAsset:  currency.USDT,
				Source:           "SPOT",
			}},
		}
		assert.Equal(t, exp, result, "GetPartnerCommissionHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPartnerCommissionHistory should return a history")
}

func TestGetPartnerSubordinateList(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPartnerSubordinateList(t.Context(), &PartnerSubordinateListRequest{UserID: 12312312, Limit: 10, Offset: 100})
	require.NoError(t, err, "GetPartnerSubordinateList must not error")
	if mockTests {
		exp := &PartnerSubordinateListResponse{
			Total: 1,
			List:  []*PartnerSubordinate{{UserID: 10000001, UserJoinTime: types.Time(time.Unix(1715864142, 0)), Type: 3}},
		}
		assert.Equal(t, exp, result, "GetPartnerSubordinateList should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPartnerSubordinateList should return a list")
}

func TestGetBrokerCommissionHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetBrokerCommissionHistory(t.Context(), &RebateBrokerHistoryRequest{From: endTime, To: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetBrokerCommissionHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetBrokerCommissionHistory(t.Context(), &RebateBrokerHistoryRequest{UserID: 12312312, From: startTime, To: endTime, Limit: 1, Offset: 100})
	require.NoError(t, err, "GetBrokerCommissionHistory must not error")
	if mockTests {
		exp := &BrokerCommissionHistoryResponse{
			Total: 1,
			List: []*BrokerCommission{{
				CommissionTime: types.Time(time.Unix(1715864142, 0)),
				UserID:         10000001,
				GroupName:      "group_a",
				Amount:         1500.5,
				Fee:            0.65,
				FeeAsset:       currency.USDT,
				RebateFee:      0.32,
				Source:         "Spot",
				CurrencyPair:   rebateTestPair,
				SubBrokerInfo: &BrokerSubBrokerInfo{
					UserID:                 20000002,
					OriginalCommissionRate: 0.0008,
					RelativeCommissionRate: 0.5,
					CommissionRate:         0.0004,
				},
			}},
		}
		assert.Equal(t, exp, result, "GetBrokerCommissionHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetBrokerCommissionHistory should return a history")
}

func TestGetBrokerTransactionHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetBrokerTransactionHistory(t.Context(), &RebateBrokerHistoryRequest{From: endTime, To: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetBrokerTransactionHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetBrokerTransactionHistory(t.Context(), &RebateBrokerHistoryRequest{UserID: 12312312, From: startTime, To: endTime, Limit: 1, Offset: 100})
	require.NoError(t, err, "GetBrokerTransactionHistory must not error")
	if mockTests {
		exp := &BrokerTransactionHistoryResponse{
			Total: 1,
			List: []*BrokerTransaction{{
				TransactionTime: types.Time(time.Unix(1715864142, 0)),
				UserID:          10000001,
				GroupName:       "group_a",
				Fee:             0.65,
				CurrencyPair:    rebateTestPair,
				Amount:          1500.5,
				FeeAsset:        currency.USDT,
				Source:          "Spot",
				SubBrokerInfo: &BrokerSubBrokerInfo{
					UserID:                 20000002,
					OriginalCommissionRate: 0.0008,
					RelativeCommissionRate: 0.5,
					CommissionRate:         0.0004,
				},
			}},
		}
		assert.Equal(t, exp, result, "GetBrokerTransactionHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetBrokerTransactionHistory should return a history")
}

func TestGetUserRebateInformation(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	inviterUserID, err := e.GetUserRebateInformation(t.Context())
	require.NoError(t, err, "GetUserRebateInformation must not error")
	if mockTests {
		assert.Equal(t, uint64(10000001), inviterUserID, "GetUserRebateInformation should decode the inviter's user ID")
	}
}

func TestGetUserSubordinateRelationship(t *testing.T) {
	t.Parallel()
	_, err := e.GetUserSubordinateRelationship(t.Context(), nil)
	require.ErrorIs(t, err, errUserIDRequired, "GetUserSubordinateRelationship must reject an empty user list")

	_, err = e.GetUserSubordinateRelationship(t.Context(), []uint64{12342, 0})
	require.ErrorIs(t, err, errUserIDRequired, "GetUserSubordinateRelationship must reject an empty user ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUserSubordinateRelationship(t.Context(), []uint64{12342, 21312312312})
	require.NoError(t, err, "GetUserSubordinateRelationship must not error")
	if mockTests {
		exp := &UserSubordinateRelationResponse{List: []*UserSubordinateRelation{{UserID: 12342, Affiliation: "partner", Type: 5, InviterUserID: 10000001}}}
		assert.Equal(t, exp, result, "GetUserSubordinateRelationship should decode every field")
		return
	}
	assert.NotNil(t, result, "GetUserSubordinateRelationship should return relationships")
}

func TestGetRecentPartnerApplicationRecords(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetRecentPartnerApplicationRecords(t.Context())
	require.NoError(t, err, "GetRecentPartnerApplicationRecords must not error")
	if mockTests {
		exp := &PartnerApplicationRecord{
			ID:                 64779,
			UserID:             19669134,
			Language:           "en",
			CountryID:          37,
			FirstName:          "Trader",
			LastName:           "One",
			Email:              "trader@example.com",
			JoinUserID:         19669134,
			IdentityComment:    "Content creator",
			PromotionChannels:  "19|15|1",
			ContactDetails:     `{"4":"992937821","2":"13169503913"}`,
			QuestionLanguage:   "en",
			CreateTime:         "2026-03-16 02:22:28",
			UpdateTime:         "2026-03-16 02:22:29",
			ApplyType:          1,
			EditCounts:         1,
			ProofImages:        "rebate/gateio/referral/apply_partner/69b76933b6583",
			ProofURL:           "https://example.com/proof",
			ChannelType:        3,
			Region:             "au",
			Phone:              "13169503913",
			Telegram:           "@trader",
			OtherContact:       PartnerApplicationContact{Type: 4, Value: "992937821"},
			ProofImagesURLList: []string{"https://example.com/proof.png"},
			ProofVideosURLList: []string{},
			ApplyMessage:       "Thanks for applying",
			JumpURL:            "/referral/affiliate/program-application",
		}
		assert.Equal(t, exp, result, "GetRecentPartnerApplicationRecords should decode every field")
		return
	}
	assert.NotNil(t, result, "GetRecentPartnerApplicationRecords should return a record")
}

func TestCheckPartnerApplicationEligibility(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.CheckPartnerApplicationEligibility(t.Context())
	require.NoError(t, err, "CheckPartnerApplicationEligibility must not error")
	if mockTests {
		exp := &PartnerEligibilityResponse{BlockReasons: []string{"Sub-accounts cannot apply"}, BlockReasonCodes: []string{"sub_account"}}
		assert.Equal(t, exp, result, "CheckPartnerApplicationEligibility should decode every field")
		return
	}
	assert.NotNil(t, result, "CheckPartnerApplicationEligibility should return eligibility")
}

func TestGetAggregatedPartnerAgentStatistics(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetAggregatedPartnerAgentStatistics(t.Context(), endTime, startTime, 1)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetAggregatedPartnerAgentStatistics must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	// The recorded request carries the range in UTC+8, as Gate requires
	result, err := e.GetAggregatedPartnerAgentStatistics(t.Context(), startTime, endTime, 1)
	require.NoError(t, err, "GetAggregatedPartnerAgentStatistics must not error")
	if mockTests {
		exp := &RebateAgentStatisticsResponse{
			RebateAmount:     456.123,
			TradeVolume:      2345678.9,
			NetFee:           234.56,
			CustomerCount:    75,
			TimeRangeDesc:    "2025-04-08 ~ 2025-04-09",
			BusinessType:     1,
			BusinessTypeDesc: "Spot",
		}
		assert.Equal(t, exp, result, "GetAggregatedPartnerAgentStatistics should decode every field")

		// Gate reports failures in the response code of its envelope
		_, err = e.GetAggregatedPartnerAgentStatistics(t.Context(), startTime, endTime, 2)
		assert.ErrorContains(t, err, "Current account is not a partner agent", "GetAggregatedPartnerAgentStatistics should surface an envelope error")
		return
	}
	assert.NotNil(t, result, "GetAggregatedPartnerAgentStatistics should return statistics")
}
