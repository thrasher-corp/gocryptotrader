package bingx

import (
	"io"
	"net/http"
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

func TestGetAgentInvitedUsers(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *AgentInvitedUsersRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing page index", arg: &AgentInvitedUsersRequest{PageSize: 2}, err: errPageIndexRequired},
		{name: "missing page size", arg: &AgentInvitedUsersRequest{PageIndex: 1}, err: errPageSizeRequired},
		{name: "reversed time range", arg: &AgentInvitedUsersRequest{StartTime: time.UnixMilli(2000), EndTime: time.UnixMilli(1000), PageIndex: 1, PageSize: 2}, err: common.ErrStartAfterEnd},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetAgentInvitedUsers(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "GetAgentInvitedUsers should reject the request")
		})
	}
	t.Run("optional parameters sent", func(t *testing.T) {
		t.Parallel()
		ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method, "method should be GET")
			verifyQuerySignature(t, r.URL.RawQuery)
			q := r.URL.Query()
			q.Del("timestamp")
			q.Del("signature")
			exp := url.Values{
				"startTime": {"1000"}, "endTime": {"2000"}, "lastUid": {"24828902"}, "pageIndex": {"3"}, "pageSize": {"200"},
				"recvWindow": {"5000"},
			}
			assert.Equal(t, exp, q, "every parameter should be sent")
			_, _ = io.WriteString(w, `{"code":0,"data":{"list":[],"total":0}}`)
		})
		result, err := ex.GetAgentInvitedUsers(t.Context(), &AgentInvitedUsersRequest{
			StartTime: time.UnixMilli(1000),
			EndTime:   time.UnixMilli(2000),
			LastUID:   24828902,
			PageIndex: 3,
			PageSize:  200,
		})
		require.NoError(t, err, "GetAgentInvitedUsers must not error")
		assert.Equal(t, &AgentInvitedUsersResponse{List: []AgentInvitedUser{}}, result, "GetAgentInvitedUsers should decode an empty page")
	})
	t.Run("valid request", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		result, err := e.GetAgentInvitedUsers(t.Context(), &AgentInvitedUsersRequest{PageIndex: 1, PageSize: 2})
		require.NoError(t, err, "GetAgentInvitedUsers must not error")
		require.NotNil(t, result, "GetAgentInvitedUsers must return a result")
		if !mockTests {
			return
		}
		exp := &AgentInvitedUsersResponse{
			List: []AgentInvitedUser{{
				UID:              24828902,
				OwnInviteCode:    "LYA1453",
				SuperiorsUID:     2293934,
				InvitationCode:   "LYA2023",
				RegisterDateTime: types.Time(time.UnixMilli(1688992720000)),
				CommissionRatio:  3,
			}},
			Total:           1,
			CurrentAgentUID: 1115195195666423800,
		}
		assert.Equal(t, exp, result, "GetAgentInvitedUsers should decode the documented example")
	})
}

func TestGetAgentDailyCommissionDetails(t *testing.T) {
	t.Parallel()
	start, end := time.UnixMilli(1688992720000), time.UnixMilli(1689426920000)
	for _, tc := range []struct {
		name string
		arg  *AgentDailyCommissionDetailsRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing start time", arg: &AgentDailyCommissionDetailsRequest{EndTime: end, PageIndex: 1, PageSize: 2}, err: common.ErrDateUnset},
		{name: "missing end time", arg: &AgentDailyCommissionDetailsRequest{StartTime: start, PageIndex: 1, PageSize: 2}, err: common.ErrDateUnset},
		{name: "reversed time range", arg: &AgentDailyCommissionDetailsRequest{StartTime: end, EndTime: start, PageIndex: 1, PageSize: 2}, err: common.ErrStartAfterEnd},
		{name: "missing page index", arg: &AgentDailyCommissionDetailsRequest{StartTime: start, EndTime: end, PageSize: 2}, err: errPageIndexRequired},
		{name: "missing page size", arg: &AgentDailyCommissionDetailsRequest{StartTime: start, EndTime: end, PageIndex: 1}, err: errPageSizeRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetAgentDailyCommissionDetails(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "GetAgentDailyCommissionDetails should reject the request")
		})
	}
	t.Run("valid request", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		result, err := e.GetAgentDailyCommissionDetails(t.Context(), &AgentDailyCommissionDetailsRequest{
			UID:            24391361,
			InvitationCode: "LYA2025",
			BusinessType:   AgentBusinessTypeTraditionalFinance,
			DistanceType:   AgentInvitationTypeDirect,
			StartTime:      start,
			EndTime:        end,
			PageIndex:      1,
			PageSize:       2,
		})
		require.NoError(t, err, "GetAgentDailyCommissionDetails must not error")
		require.NotNil(t, result, "GetAgentDailyCommissionDetails must return a result")
		if !mockTests {
			return
		}
		exp := &AgentDailyCommissionDetailsResponse{
			List: []AgentDailyCommission{{
				UID:                  24391361,
				CommissionTime:       types.Time(time.UnixMilli(1689350400000)),
				TradingVolume:        6660.05,
				Commission:           2.37932209,
				ExpectedTradingFees:  10,
				OffsetTradingFees:    2,
				CollectedTradingFees: 8,
				RebateRatio:          0.2,
				TradingRebate:        1.6,
			}},
			Total:           43,
			CurrentAgentUID: 1115195195666423800,
		}
		assert.Equal(t, exp, result, "GetAgentDailyCommissionDetails should decode the documented example")
	})
}

func TestGetAgentUserInformation(t *testing.T) {
	t.Parallel()
	t.Run("missing UID", func(t *testing.T) {
		t.Parallel()
		_, err := e.GetAgentUserInformation(t.Context(), 0)
		assert.ErrorIs(t, err, errUIDRequired, "GetAgentUserInformation should reject a missing UID")
	})
	t.Run("valid request", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		result, err := e.GetAgentUserInformation(t.Context(), 1645382)
		require.NoError(t, err, "GetAgentUserInformation must not error")
		require.NotNil(t, result, "GetAgentUserInformation must return a result")
		if !mockTests {
			return
		}
		exp := &AgentUserInformationResponse{
			UID:              2489544,
			ExistInviter:     true,
			InviteResult:     true,
			RegisterDateTime: types.Time(time.UnixMilli(1656208955000)),
			SuperiorsUID:     2293934,
			Deposit:          true,
			Trade:            true,
			UserLevel:        2,
			CommissionRatio:  20,
		}
		assert.Equal(t, exp, result, "GetAgentUserInformation should decode the documented example")
	})
}

func TestGetAgentInvitedUserDepositDetails(t *testing.T) {
	t.Parallel()
	start, end := time.UnixMilli(1699183026000), time.UnixMilli(1699269426000)
	valid := AgentInvitedUserDepositDetailsRequest{
		UID:          1645382,
		BusinessType: AgentDepositBusinessTypeCryptoDeposit,
		StartTime:    start,
		EndTime:      end,
		PageIndex:    1,
		PageSize:     10,
	}
	for _, tc := range []struct {
		name   string
		modify func(*AgentInvitedUserDepositDetailsRequest)
		err    error
	}{
		{name: "missing UID", modify: func(r *AgentInvitedUserDepositDetailsRequest) { r.UID = 0 }, err: errUIDRequired},
		{name: "missing business type", modify: func(r *AgentInvitedUserDepositDetailsRequest) { r.BusinessType = 0 }, err: errAgentDepositBusinessTypeRequired},
		{name: "missing start time", modify: func(r *AgentInvitedUserDepositDetailsRequest) { r.StartTime = time.Time{} }, err: common.ErrDateUnset},
		{name: "missing end time", modify: func(r *AgentInvitedUserDepositDetailsRequest) { r.EndTime = time.Time{} }, err: common.ErrDateUnset},
		{name: "reversed time range", modify: func(r *AgentInvitedUserDepositDetailsRequest) { r.StartTime, r.EndTime = end, start }, err: common.ErrStartAfterEnd},
		{name: "missing page index", modify: func(r *AgentInvitedUserDepositDetailsRequest) { r.PageIndex = 0 }, err: errPageIndexRequired},
		{name: "missing page size", modify: func(r *AgentInvitedUserDepositDetailsRequest) { r.PageSize = 0 }, err: errPageSizeRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			arg := valid
			tc.modify(&arg)
			_, err := e.GetAgentInvitedUserDepositDetails(t.Context(), &arg)
			assert.ErrorIs(t, err, tc.err, "GetAgentInvitedUserDepositDetails should reject the request")
		})
	}
	t.Run("nil request", func(t *testing.T) {
		t.Parallel()
		_, err := e.GetAgentInvitedUserDepositDetails(t.Context(), nil)
		assert.ErrorIs(t, err, common.ErrNilPointer, "GetAgentInvitedUserDepositDetails should reject a nil request")
	})
	t.Run("valid request", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		arg := valid
		result, err := e.GetAgentInvitedUserDepositDetails(t.Context(), &arg)
		require.NoError(t, err, "GetAgentInvitedUserDepositDetails must not error")
		require.NotNil(t, result, "GetAgentInvitedUserDepositDetails must return a result")
		if !mockTests {
			return
		}
		exp := &AgentInvitedUserDepositDetailsResponse{
			List: []AgentInvitedUserDeposit{{
				UID:                  2489544,
				InviteResult:         true,
				DirectInvitation:     true,
				BusinessType:         AgentDepositBusinessTypeCryptoDeposit,
				BusinessTime:         types.Time(time.UnixMilli(1673674073000)),
				AssetType:            30,
				AssetTypeName:        "Deposit",
				CurrencyName:         currency.USDT,
				CurrencyAmountVolume: 1000,
			}},
			Total: 1,
		}
		assert.Equal(t, exp, result, "GetAgentInvitedUserDepositDetails should decode the documented example")
	})
}

func TestGetAgentAPITransactionCommission(t *testing.T) {
	t.Parallel()
	start, end := time.UnixMilli(1699183026000), time.UnixMilli(1699269426000)
	valid := AgentAPITransactionCommissionRequest{
		UID:                    1645382,
		CommissionBusinessType: AgentCommissionBusinessTypePerpetualFutures,
		StartTime:              start,
		EndTime:                end,
		PageIndex:              1,
		PageSize:               10,
	}
	for _, tc := range []struct {
		name   string
		modify func(*AgentAPITransactionCommissionRequest)
		err    error
	}{
		{name: "missing commission business type", modify: func(r *AgentAPITransactionCommissionRequest) { r.CommissionBusinessType = 0 }, err: errAgentCommissionBusinessTypeRequired},
		{name: "missing start time", modify: func(r *AgentAPITransactionCommissionRequest) { r.StartTime = time.Time{} }, err: common.ErrDateUnset},
		{name: "missing end time", modify: func(r *AgentAPITransactionCommissionRequest) { r.EndTime = time.Time{} }, err: common.ErrDateUnset},
		{name: "reversed time range", modify: func(r *AgentAPITransactionCommissionRequest) { r.StartTime, r.EndTime = end, start }, err: common.ErrStartAfterEnd},
		{name: "missing page index", modify: func(r *AgentAPITransactionCommissionRequest) { r.PageIndex = 0 }, err: errPageIndexRequired},
		{name: "missing page size", modify: func(r *AgentAPITransactionCommissionRequest) { r.PageSize = 0 }, err: errPageSizeRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			arg := valid
			tc.modify(&arg)
			_, err := e.GetAgentAPITransactionCommission(t.Context(), &arg)
			assert.ErrorIs(t, err, tc.err, "GetAgentAPITransactionCommission should reject the request")
		})
	}
	t.Run("nil request", func(t *testing.T) {
		t.Parallel()
		_, err := e.GetAgentAPITransactionCommission(t.Context(), nil)
		assert.ErrorIs(t, err, common.ErrNilPointer, "GetAgentAPITransactionCommission should reject a nil request")
	})
	t.Run("valid request", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		arg := valid
		result, err := e.GetAgentAPITransactionCommission(t.Context(), &arg)
		require.NoError(t, err, "GetAgentAPITransactionCommission must not error")
		require.NotNil(t, result, "GetAgentAPITransactionCommission must return a result")
		if !mockTests {
			return
		}
		exp := &AgentAPITransactionCommissionResponse{
			List: []AgentAPITransactionCommission{{
				UID:                    25053735,
				CommissionTime:         types.Time(time.UnixMilli(1700759104737)),
				TradeVolume:            10.234,
				CommissionVolume:       0.02663866,
				CommissionBusinessType: AgentCommissionBusinessTypePerpetualFutures,
			}},
			Total: 3,
		}
		assert.Equal(t, exp, result, "GetAgentAPITransactionCommission should decode the documented example")
	})
}

func TestGetAgentPartnerInformation(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *AgentPartnerInformationRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "reversed time range", arg: &AgentPartnerInformationRequest{StartTime: time.UnixMilli(2000), EndTime: time.UnixMilli(1000)}, err: common.ErrStartAfterEnd},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetAgentPartnerInformation(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "GetAgentPartnerInformation should reject the request")
		})
	}
	t.Run("valid request", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		result, err := e.GetAgentPartnerInformation(t.Context(), &AgentPartnerInformationRequest{
			UID:       25414560,
			StartTime: time.UnixMilli(1699183026000),
			EndTime:   time.UnixMilli(1699269426000),
			PageIndex: 1,
			PageSize:  100,
		})
		require.NoError(t, err, "GetAgentPartnerInformation must not error")
		require.NotNil(t, result, "GetAgentPartnerInformation must return a result")
		if !mockTests {
			return
		}
		exp := &AgentPartnerInformationResponse{
			List: []AgentPartner{{
				UID:                 25414560,
				Email:               "jo***@gmail.com",
				Phone:               "*******7294",
				ReferralType:        AgentInvitationTypeDirect,
				Remarks:             "123",
				ReferrerUID:         24186664,
				Language:            "en",
				BranchDeposits:      24534.446799999998,
				BranchTrading:       4,
				BranchTradingVolume: 98434310.99579316,
				CommissionRatio:     0.04,
			}},
			Total: 1,
		}
		assert.Equal(t, exp, result, "GetAgentPartnerInformation should decode the documented example")
	})
}

func TestGetAgentReferralCodeCommission(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *AgentReferralCodeCommissionRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "reversed time range", arg: &AgentReferralCodeCommissionRequest{StartTime: time.UnixMilli(2000), EndTime: time.UnixMilli(1000)}, err: common.ErrStartAfterEnd},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetAgentReferralCodeCommission(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "GetAgentReferralCodeCommission should reject the request")
		})
	}
	t.Run("valid request", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		result, err := e.GetAgentReferralCodeCommission(t.Context(), &AgentReferralCodeCommissionRequest{
			ReferralCode:     "LYA2025",
			DirectInvitation: true,
			StartTime:        time.UnixMilli(1699183026000),
			EndTime:          time.UnixMilli(1699269426000),
			PageIndex:        1,
			PageSize:         100,
		})
		require.NoError(t, err, "GetAgentReferralCodeCommission must not error")
		require.NotNil(t, result, "GetAgentReferralCodeCommission must return a result")
		if !mockTests {
			return
		}
		exp := &AgentReferralCodeCommissionResponse{
			List: []AgentReferralCodeCommission{{
				ReferralCode:       "LYA2025",
				CommissionRatio:    50,
				FriendEarn:         45,
				SwapFeeDiscount:    30,
				ReferralNumber:     8,
				Deposited:          3,
				Traded:             6,
				TradingVolume:      24534.446799999998,
				Fee:                120.492,
				OffsetTradingFees:  120.492,
				PayableTradingFees: 120.492,
				Commission:         10.04,
			}},
		}
		assert.Equal(t, exp, result, "GetAgentReferralCodeCommission should decode the documented example")
	})
}

func TestGetAgentSuperiorVerification(t *testing.T) {
	t.Parallel()
	t.Run("missing UID", func(t *testing.T) {
		t.Parallel()
		_, err := e.GetAgentSuperiorVerification(t.Context(), 0)
		assert.ErrorIs(t, err, errUIDRequired, "GetAgentSuperiorVerification should reject a missing UID")
	})
	t.Run("valid request", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		result, err := e.GetAgentSuperiorVerification(t.Context(), 1645382)
		require.NoError(t, err, "GetAgentSuperiorVerification must not error")
		require.NotNil(t, result, "GetAgentSuperiorVerification must return a result")
		if !mockTests {
			return
		}
		assert.Equal(t, &AgentSuperiorVerificationResponse{InviteResult: true}, result, "GetAgentSuperiorVerification should decode the documented example")
	})
}

func TestGetAgentChannelCommissionDetails(t *testing.T) {
	t.Parallel()
	start, end := time.UnixMilli(1791820800000), time.UnixMilli(1792425600000)
	for _, tc := range []struct {
		name string
		arg  *AgentChannelCommissionDetailsRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing start time", arg: &AgentChannelCommissionDetailsRequest{EndTime: end, PageIndex: 1, PageSize: 20}, err: common.ErrDateUnset},
		{name: "missing end time", arg: &AgentChannelCommissionDetailsRequest{StartTime: start, PageIndex: 1, PageSize: 20}, err: common.ErrDateUnset},
		{name: "reversed time range", arg: &AgentChannelCommissionDetailsRequest{StartTime: end, EndTime: start, PageIndex: 1, PageSize: 20}, err: common.ErrStartAfterEnd},
		{name: "missing page index", arg: &AgentChannelCommissionDetailsRequest{StartTime: start, EndTime: end, PageSize: 20}, err: errPageIndexRequired},
		{name: "missing page size", arg: &AgentChannelCommissionDetailsRequest{StartTime: start, EndTime: end, PageIndex: 1}, err: errPageSizeRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetAgentChannelCommissionDetails(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "GetAgentChannelCommissionDetails should reject the request")
		})
	}
	t.Run("valid request", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		result, err := e.GetAgentChannelCommissionDetails(t.Context(), &AgentChannelCommissionDetailsRequest{
			UID:          24391361,
			BusinessType: AgentBusinessTypePerpetualFutures,
			AssetType:    AgentAssetTypeCrypto,
			StartTime:    start,
			EndTime:      end,
			PageIndex:    1,
			PageSize:     20,
		})
		require.NoError(t, err, "GetAgentChannelCommissionDetails must not error")
		require.NotNil(t, result, "GetAgentChannelCommissionDetails must return a result")
		if !mockTests {
			return
		}
		exp := &AgentChannelCommissionDetailsResponse{
			List: []AgentChannelCommission{{
				UID:                   24391361,
				TradingVolume:         10000,
				ExpectedTradingFees:   10,
				OffsetTradingFees:     2,
				CollectedTradingFees:  8,
				Commission:            1.6,
				PartnerCommissionRate: 0.2,
			}},
			Total:           1,
			CurrentAgentUID: 12345678,
		}
		assert.Equal(t, exp, result, "GetAgentChannelCommissionDetails should decode the documented example")
	})
}

func TestAgentPageParams(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                string
		pageIndex, pageSize uint64
		err                 error
		exp                 url.Values
	}{
		{name: "missing page index", pageSize: 10, err: errPageIndexRequired},
		{name: "missing page size", pageIndex: 1, err: errPageSizeRequired},
		{name: "both set", pageIndex: 2, pageSize: 10, exp: url.Values{"pageIndex": {"2"}, "pageSize": {"10"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			params := url.Values{}
			err := agentPageParams(params, tc.pageIndex, tc.pageSize)
			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err, "agentPageParams should return the expected error")
				assert.Empty(t, params, "agentPageParams should set nothing when it errors")
				return
			}
			require.NoError(t, err, "agentPageParams must not error")
			assert.Equal(t, tc.exp, params, "agentPageParams should set both page parameters")
		})
	}
}

func TestAgentRequiredTimeRange(t *testing.T) {
	t.Parallel()
	start, end := time.UnixMilli(1000), time.UnixMilli(2000)
	for _, tc := range []struct {
		name       string
		start, end time.Time
		err        error
		exp        url.Values
	}{
		{name: "missing start time", end: end, err: common.ErrDateUnset},
		{name: "missing end time", start: start, err: common.ErrDateUnset},
		{name: "reversed", start: end, end: start, err: common.ErrStartAfterEnd},
		{name: "both set", start: start, end: end, exp: url.Values{"startTime": {"1000"}, "endTime": {"2000"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			params := url.Values{}
			err := agentRequiredTimeRange(params, tc.start, tc.end)
			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err, "agentRequiredTimeRange should return the expected error")
				assert.Empty(t, params, "agentRequiredTimeRange should set nothing when it errors")
				return
			}
			require.NoError(t, err, "agentRequiredTimeRange must not error")
			assert.Equal(t, tc.exp, params, "agentRequiredTimeRange should set both bounds in milliseconds")
		})
	}
}
