package okx

import (
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

var (
	affiliatePeriodBegin = time.UnixMilli(1775001600000)
	affiliatePeriodEnd   = time.UnixMilli(1777593599999)
)

// newUndecodableAffiliateListExchange returns an exchange whose affiliate lists reply with an object in place of the list
func newUndecodableAffiliateListExchange(t *testing.T) *Exchange {
	t.Helper()
	return newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte(`{"code":"0","msg":"","totalPage":"1","data":{"uid":"835449167911924693"}}`))
		assert.NoError(t, err, "writing the reply should not error")
	}))
}

func TestGetAffiliatePerformanceSummary(t *testing.T) {
	t.Parallel()
	_, err := e.GetAffiliatePerformanceSummary(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetAffiliatePerformanceSummary(t.Context(), &AffiliatePerformanceSummaryRequest{PeriodType: affiliatePeriodTypeCustom})
	require.ErrorIs(t, err, common.ErrDateUnset)
	_, err = e.GetAffiliatePerformanceSummary(t.Context(), &AffiliatePerformanceSummaryRequest{PeriodType: affiliatePeriodTypeCustom, Begin: affiliatePeriodEnd, End: affiliatePeriodBegin})
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAffiliatePerformanceSummary(contextGenerate(), &AffiliatePerformanceSummaryRequest{PeriodType: affiliatePeriodTypeCustom, Begin: affiliatePeriodBegin, End: affiliatePeriodEnd})
	require.NoError(t, err, "GetAffiliatePerformanceSummary must not error")
	if mockTests {
		exp := &AffiliatePerformanceSummaryResponse{
			UpdateTime:    types.Time(time.UnixMilli(1777541513000)),
			InviteeCount:  types.Number(102),
			DepositAmount: types.Number(1756.287846940199989393),
			Details: []AffiliateCommissionCategory{
				{CommissionCategory: "SPOT", FirstTraderCount: types.Number(17), TraderCount: types.Number(17), Volume: types.Number(21548.6417826825604), Commission: types.Number(3.322319946747010328)},
				{CommissionCategory: "DERIVATIVE", FirstTraderCount: types.Number(9), TraderCount: types.Number(9), Volume: types.Number(94531.94802), Commission: types.Number(3.25142568535855)},
				{CommissionCategory: "BSC"},
			},
		}
		assert.Equal(t, exp, result, "GetAffiliatePerformanceSummary should decode every field")
	} else {
		assert.NotNil(t, result, "GetAffiliatePerformanceSummary should return the summary")
	}
}

func TestGetInviteeDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetInviteeDetail(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetInviteeDetail(t.Context(), &InviteeDetailRequest{})
	require.ErrorIs(t, err, errUserIDRequired)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetInviteeDetail(contextGenerate(), &InviteeDetailRequest{UID: "1234", PeriodType: "last_7d"})
	require.NoError(t, err, "GetInviteeDetail must not error")
	if mockTests {
		exp := &InviteeDetailResponse{
			InviteeLevel:      types.Number(2),
			JoinTime:          types.Time(time.UnixMilli(1712546713000)),
			InviteeRebateRate: types.Number(0.39),
			TotalCommission:   types.Number(0.05),
			FirstTradeTime:    types.Time(time.UnixMilli(1712633113000)),
			Level:             "Lv1",
			DepositAmount:     types.Number(100),
			WithdrawalAmount:  types.Number(50),
			MonthVolume:       types.Number(300),
			TotalVolume:       types.Number(5000),
			PeriodVolume:      types.Number(1234.56),
			AccumulatedFee:    types.Number(0.12),
			KYCTime:           types.Time(time.UnixMilli(1712505600000)),
			Region:            "Vietnam",
			AffiliateCode:     "HIIIIII",
		}
		assert.Equal(t, exp, result, "GetInviteeDetail should decode every field")
	} else {
		assert.NotNil(t, result, "GetInviteeDetail should return the invitee")
	}
}

func TestGetInviteeList(t *testing.T) {
	t.Parallel()
	_, err := e.GetInviteeList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	tooManyUIDs := make([]string, maxInviteeListUIDs+1)
	for i := range tooManyUIDs {
		tooManyUIDs[i] = strconv.Itoa(i)
	}
	_, err = e.GetInviteeList(t.Context(), &InviteeListRequest{UIDs: tooManyUIDs})
	require.ErrorIs(t, err, errExceedLimit)
	_, err = e.GetInviteeList(t.Context(), &InviteeListRequest{JoinTimeBegin: time.UnixMilli(1777400000000)})
	require.ErrorIs(t, err, common.ErrDateUnset)
	_, err = e.GetInviteeList(t.Context(), &InviteeListRequest{JoinTimeEnd: time.UnixMilli(1777500000000)})
	require.ErrorIs(t, err, common.ErrDateUnset)
	_, err = e.GetInviteeList(t.Context(), &InviteeListRequest{JoinTimeBegin: time.UnixMilli(1777500000000), JoinTimeEnd: time.UnixMilli(1777400000000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd)
	_, err = e.GetInviteeList(t.Context(), &InviteeListRequest{PeriodType: affiliatePeriodTypeCustom, Begin: affiliatePeriodBegin})
	require.ErrorIs(t, err, common.ErrDateUnset)

	t.Run("data that is not a list", func(t *testing.T) {
		t.Parallel()
		_, err := newUndecodableAffiliateListExchange(t).GetInviteeList(t.Context(), &InviteeListRequest{})
		assert.ErrorContains(t, err, "error decoding invitee list", "GetInviteeList should report a list it cannot decode")
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetInviteeList(contextGenerate(), &InviteeListRequest{
		Page:               1,
		Limit:              1,
		PeriodType:         affiliatePeriodTypeCustom,
		Begin:              affiliatePeriodBegin,
		End:                affiliatePeriodEnd,
		Keyword:            "X2UWA2T89",
		CommissionCategory: "SPOT",
		OrderBy:            "vol",
		OrderDirection:     "desc",
		KYCStatus:          "verified",
		SubAffiliateUID:    "668418489887292061",
		UIDs:               []string{"835449167911924693", "835449167911924700"},
		JoinTimeBegin:      time.UnixMilli(1777400000000),
		JoinTimeEnd:        time.UnixMilli(1777500000000),
	})
	require.NoError(t, err, "GetInviteeList must not error")
	if mockTests {
		exp := &InviteeListResponse{
			TotalPage: types.Number(5),
			Invitees: []InviteeResponse{{
				UID:             "835449167911924693",
				Country:         "CN",
				JoinTime:        types.Time(time.UnixMilli(1777448564000)),
				FirstTradeTime:  types.Time(time.UnixMilli(1777448600000)),
				ChannelName:     "X2UWA2T89",
				RebateRate:      types.Number(0.16),
				FeeTierRank:     types.Number(1),
				KYCStatus:       "verified",
				KYCTime:         types.Time(time.UnixMilli(1777448563000)),
				DepositAmount:   types.Number(10.5),
				TotalVolume:     types.Number(2500),
				TotalFee:        types.Number(2.5),
				TotalCommission: types.Number(0.4),
				IsCompliant:     true,
			}},
		}
		assert.Equal(t, exp, result, "GetInviteeList should decode every field")
	} else {
		assert.NotNil(t, result, "GetInviteeList should return a page")
	}
}

func TestGetAffiliateLinkList(t *testing.T) {
	t.Parallel()
	_, err := e.GetAffiliateLinkList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	t.Run("data that is not a list", func(t *testing.T) {
		t.Parallel()
		_, err := newUndecodableAffiliateListExchange(t).GetAffiliateLinkList(t.Context(), &AffiliateLinkListRequest{})
		assert.ErrorContains(t, err, "error decoding affiliate link list", "GetAffiliateLinkList should report a list it cannot decode")
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAffiliateLinkList(contextGenerate(), &AffiliateLinkListRequest{Page: 1, Limit: 1, LinkType: "co_inviter", LinkStatus: "normal"})
	require.NoError(t, err, "GetAffiliateLinkList must not error")
	if mockTests {
		exp := &AffiliateLinkListResponse{
			TotalPage: types.Number(1),
			Links: []AffiliateLinkResponse{{
				ChannelID:               "78295211",
				ChannelName:             "78295211",
				JoinLink:                "https://okx.com/join/78295211",
				LinkType:                "co_inviter",
				InviterCommissionRate:   types.Number(0.29),
				CoInviterCommissionRate: types.Number(0.14),
				InviteeDiscountRate:     types.Number(0.01),
				InviteeCount:            types.Number(1),
				TraderCount:             types.Number(1),
				TotalCommission:         types.Number(2.656307),
				Commission24Hour:        types.Number(0.5),
				CreationTime:            types.Time(time.UnixMilli(1773739123000)),
				IsDefault:               true,
				LinkStatus:              "normal",
			}},
		}
		assert.Equal(t, exp, result, "GetAffiliateLinkList should decode every field")
	} else {
		assert.NotNil(t, result, "GetAffiliateLinkList should return a page")
	}
}

func TestGetCoInviterLinkList(t *testing.T) {
	t.Parallel()
	_, err := e.GetCoInviterLinkList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	t.Run("data that is not a list", func(t *testing.T) {
		t.Parallel()
		_, err := newUndecodableAffiliateListExchange(t).GetCoInviterLinkList(t.Context(), &CoInviterLinkListRequest{})
		assert.ErrorContains(t, err, "error decoding co-inviter link list", "GetCoInviterLinkList should report a list it cannot decode")
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCoInviterLinkList(contextGenerate(), &CoInviterLinkListRequest{Page: 1, Limit: 1, LinkStatus: "normal"})
	require.NoError(t, err, "GetCoInviterLinkList must not error")
	if mockTests {
		exp := &CoInviterLinkListResponse{
			TotalPage: types.Number(1),
			Links: []CoInviterLinkResponse{{
				ChannelID:               "31075853",
				ChannelName:             "MXCUS",
				JoinLink:                "https://okx.com/join/MXCUS",
				InviterCommissionRate:   types.Number(0.01),
				CoInviterCommissionRate: types.Number(0.14),
				InviteeDiscountRate:     types.Number(0.16),
				PartnerUserName:         "12****23",
				CoInviterUserName:       "***",
				IsCompliant:             true,
				Note:                    "partner channel",
				IsDefault:               true,
				TotalCommission:         types.Number(0.032111),
				Commission24Hour:        types.Number(0.01),
				InviteeCount:            types.Number(1),
				TraderCount:             types.Number(1),
				ClickCount:              types.Number(1),
				TotalFee:                types.Number(0.2),
				CreationTime:            types.Time(time.UnixMilli(1773739123000)),
				ChannelAssessmentStatus: "valid",
				InviterChannelStatus:    "valid",
				CoInviterChannelStatus:  "valid",
				LinkStatus:              "normal",
			}},
		}
		assert.Equal(t, exp, result, "GetCoInviterLinkList should decode every field")
	} else {
		assert.NotNil(t, result, "GetCoInviterLinkList should return a page")
	}
}

func TestGetSubAffiliateList(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAffiliateList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	t.Run("data that is not a list", func(t *testing.T) {
		t.Parallel()
		_, err := newUndecodableAffiliateListExchange(t).GetSubAffiliateList(t.Context(), &SubAffiliateListRequest{})
		assert.ErrorContains(t, err, "error decoding sub-affiliate list", "GetSubAffiliateList should report a list it cannot decode")
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSubAffiliateList(contextGenerate(), &SubAffiliateListRequest{Page: 1, Limit: 1, Keyword: "668418489887292061", CommissionCategory: "SPOT", OrderBy: "vol", OrderDirection: "desc"})
	require.NoError(t, err, "GetSubAffiliateList must not error")
	if mockTests {
		exp := &SubAffiliateListResponse{
			TotalPage: types.Number(1),
			SubAffiliates: []SubAffiliateResponse{{
				SubAffiliateUID:   "668418489887292061",
				Country:           "CN",
				JoinTime:          types.Time(time.UnixMilli(1773739123000)),
				SubAffiliateLevel: types.Number(2),
				CommissionRate:    types.Number(0.3),
				IsCompliant:       true,
				InviteeCount:      types.Number(8),
				TraderCount:       types.Number(3),
				DepositAmount:     types.Number(37.281133),
				TotalVolume:       types.Number(3618.56143),
				TotalFee:          types.Number(1.628353),
				TotalCommission:   types.Number(0.289847),
			}},
		}
		assert.Equal(t, exp, result, "GetSubAffiliateList should decode every field")
	} else {
		assert.NotNil(t, result, "GetSubAffiliateList should return a page")
	}
}

func TestGetUserAffiliateRebateInformation(t *testing.T) {
	t.Parallel()
	_, err := e.GetUserAffiliateRebateInformation(t.Context(), "")
	require.ErrorIs(t, err, errInvalidAPIKey)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUserAffiliateRebateInformation(contextGenerate(), "1234")
	require.NoError(t, err, "GetUserAffiliateRebateInformation must not error")
	if mockTests {
		assert.Equal(t, &AffiliateRebateInformationResponse{Result: true, Type: "0"}, result, "GetUserAffiliateRebateInformation should decode every field")
	} else {
		assert.NotNil(t, result, "GetUserAffiliateRebateInformation should return the rebate information")
	}
}

func TestSetAffiliatePeriod(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		periodType string
		begin, end time.Time
		exp        url.Values
		err        error
	}{
		{name: "no period", exp: url.Values{}},
		{name: "named period", periodType: "last_7d", exp: url.Values{"periodType": {"last_7d"}}},
		{
			name:       "custom period",
			periodType: affiliatePeriodTypeCustom,
			begin:      affiliatePeriodBegin,
			end:        affiliatePeriodEnd,
			exp:        url.Values{"periodType": {"custom"}, "begin": {"1775001600000"}, "end": {"1777593599999"}},
		},
		{name: "custom period without begin", periodType: affiliatePeriodTypeCustom, end: affiliatePeriodEnd, err: common.ErrDateUnset},
		{name: "custom period without end", periodType: affiliatePeriodTypeCustom, begin: affiliatePeriodBegin, err: common.ErrDateUnset},
		{name: "begin after end", periodType: affiliatePeriodTypeCustom, begin: affiliatePeriodEnd, end: affiliatePeriodBegin, err: common.ErrStartAfterEnd},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			params := url.Values{}
			err := setAffiliatePeriod(params, tc.periodType, tc.begin, tc.end)
			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err, "setAffiliatePeriod should return the expected error")
				return
			}
			require.NoError(t, err, "setAffiliatePeriod must not error")
			assert.Equal(t, tc.exp, params, "setAffiliatePeriod should set the documented parameters")
		})
	}
}
