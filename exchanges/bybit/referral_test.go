package bybit

import (
	"fmt"
	"net/http"
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

func TestGetReferralCommissionRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetReferralCommissionRecords(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetReferralCommissionRecords must reject a nil request")
	start, end := time.Date(2025, 3, 1, 10, 20, 30, 0, time.UTC), time.Date(2025, 3, 8, 11, 22, 33, 0, time.UTC)
	_, err = e.GetReferralCommissionRecords(t.Context(), &ReferralCommissionRecordsRequest{StartTime: end, EndTime: start})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetReferralCommissionRecords must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *ReferralCommissionRecordsRequest
		exp  *ReferralCommissionRecordsResponse
	}{
		{
			// A time in another zone is sent as its UTC date time
			name: "invitees in a window",
			req:  &ReferralCommissionRecordsRequest{UserIDList: []uint64{201, 202}, StartTime: start.In(time.FixedZone("AEDT", 11*60*60)), EndTime: end, Page: 3, PageSize: 200, ProductLine: 1},
			exp: &ReferralCommissionRecordsResponse{
				Records: []ReferralCommissionRecord{
					{
						UserID:                 "2***1",
						PaymentTime:            types.Time(time.Unix(1740916800, 0)),
						Rate:                   0.2,
						BusinessUnit:           1,
						CommissionOriginCoin:   currency.BTC,
						CommissionOriginAmount: 1.25,
						CommissionUSDTAmount:   250,
						CommissionStatus:       2,
					},
					{
						UserID:                 "2***2",
						PaymentTime:            types.Time(time.Unix(1741003200, 0)),
						Rate:                   0.3,
						BusinessUnit:           1,
						CommissionOriginCoin:   currency.USDT,
						CommissionOriginAmount: 12.5,
						CommissionUSDTAmount:   12.5,
						CommissionStatus:       1,
						IsBoostM2:              true,
					},
				},
				Total: 29,
			},
		},
		{
			// The gateway may quote int64 values, as ProtoJSON does
			name: "every invitee",
			req:  &ReferralCommissionRecordsRequest{},
			exp: &ReferralCommissionRecordsResponse{
				Records: []ReferralCommissionRecord{
					{
						UserID:                 "1***6",
						PaymentTime:            types.Time(time.Unix(1740916800, 0)),
						Rate:                   0.2,
						BusinessUnit:           2,
						CommissionOriginCoin:   currency.USDT,
						CommissionOriginAmount: 3.5,
						CommissionUSDTAmount:   3.5,
						CommissionStatus:       2,
					},
				},
				Total: 1,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetReferralCommissionRecords(t.Context(), tc.req)
			require.NoError(t, err, "GetReferralCommissionRecords must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetReferralCommissionRecords should decode every field")
				return
			}
			assert.NotNil(t, result, "GetReferralCommissionRecords should return a page")
		})
	}
}

func TestGetReferralCampaignRewards(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetReferralCampaignRewards(t.Context())
	require.NoError(t, err, "GetReferralCampaignRewards must not error")
	if mockTests {
		exp := &ReferralCampaignRewardsResponse{
			RebateRule: ReferralRebateRule{
				DepositAmount: 100,
				TradeAmount:   500,
				RebateRatioRules: []ReferralRebateRatioRule{
					{RebateRatio: 20, ReferralCount: 1},
					{RebateRatio: 30, ReferralCount: 10, HasTradeCondition: true, TradeAmountCondition: 100000},
				},
				RebateDurationDays:  30,
				TradFiDepositAmount: 200,
				TradFiTradeAmount:   1000,
			},
			Tasks: []ReferralCampaign{
				{
					SchemaVersion: 2,
					Source:        "base",
					CampaignID:    "sample",
					Tasks: []ReferralCampaignTask{
						{
							ID:              7,
							AwardType:       100,
							RuleTitle:       []string{"Deposit"},
							RuleDescription: []string{"Deposit $100"},
							RuleParam:       []string{"100"},
							TaskAwards:      []ReferralCampaignAward{{ID: 92, Name: "EARN 10", Type: "Packet"}},
							M1Awards:        []ReferralCampaignAward{},
						},
					},
				},
				{
					SchemaVersion: 2,
					Source:        "fiat",
					CampaignID:    "fiat-first-purchase",
					Tasks: []ReferralCampaignTask{
						{
							ID:              12,
							AwardType:       1,
							RuleTitle:       []string{"First fiat purchase", "Spot trade"},
							RuleDescription: []string{"Buy at least $50 of crypto with fiat", "Trade at least $500 in spot"},
							RuleParam:       []string{"50", "500"},
							TaskAwards: []ReferralCampaignAward{
								{
									ID:          405,
									Name:        "10 USDT Trial Fund",
									Type:        "Trial Fund",
									ProductLine: "Derivatives",
									StartTime:   ReferralAwardTime(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)),
									EndTime:     ReferralAwardTime(time.Date(2026, 10, 31, 23, 59, 59, 0, time.UTC)),
								},
							},
							M1Awards: []ReferralCampaignAward{
								{
									ID:          406,
									Name:        "5 USDT Coupon",
									Type:        "Coupon",
									ProductLine: "Spot",
									StartTime:   ReferralAwardTime(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)),
								},
							},
						},
						{
							ID:         13,
							AwardType:  95,
							TaskAwards: []ReferralCampaignAward{},
							M1Awards:   []ReferralCampaignAward{},
						},
					},
				},
				{
					SchemaVersion: 2,
					Source:        "earn",
					CampaignID:    "earn-boost",
					Tasks:         []ReferralCampaignTask{},
				},
			},
		}
		assert.Equal(t, exp, result, "GetReferralCampaignRewards should decode every field")
		return
	}
	assert.NotNil(t, result, "GetReferralCampaignRewards should return the rules")
}

func TestGetReferralUserIdentityProfile(t *testing.T) {
	t.Parallel()
	// Bybit does not check an affiliate's M1 inviter status, so its profile has no commission figures; the reply quotes
	// its int64 values, as ProtoJSON does
	affiliate := newTransportTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost+" /v5/referral/user/identity-profile", r.Method+" "+r.URL.Path, "GetReferralUserIdentityProfile should post to its endpoint")
		_, err := fmt.Fprint(w, `{"retCode":0,"retMsg":"OK","result":{"isAffiliate":true,"isM1":false,"inviterAffiliateId":"0","inviterAffiliateUserId":"0","inviterUserId":"1"},"retExtInfo":{},"time":1672211918471}`)
		assert.NoError(t, err, "writing the reply should not error")
	})
	result, err := affiliate.GetReferralUserIdentityProfile(t.Context())
	require.NoError(t, err, "GetReferralUserIdentityProfile must not error for an affiliate")
	assert.Equal(t, &ReferralUserIdentityProfileResponse{IsAffiliate: true, InviterUserID: 1}, result, "GetReferralUserIdentityProfile should decode an affiliate's profile")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err = e.GetReferralUserIdentityProfile(t.Context())
	require.NoError(t, err, "GetReferralUserIdentityProfile must not error")
	if mockTests {
		exp := &ReferralUserIdentityProfileResponse{
			IsM1:                   true,
			InviterAffiliateID:     10001,
			InviterAffiliateUserID: 1,
			InviterUserID:          1,
			CommissionInfo: &ReferralCommissionInfo{
				TotalCommission:                         152.35,
				YesterdayCommission:                     3.2,
				TotalFriends:                            12,
				QualifiedFriends:                        5,
				WalletBalance:                           48.5,
				QualifiedFriendsQuarterTotalTradeVolume: 250000,
				CurrentQuarterActiveFriendsCount:        3,
				HasOldReferral:                          1,
				DegradeType:                             1,
				CommissionRate:                          0.3,
				UpgradeUnifiedAccount:                   1,
				WithdrawalSwitch:                        1,
				LastQuarterQualifiedFriendsCount:        4,
			},
		}
		assert.Equal(t, exp, result, "GetReferralUserIdentityProfile should decode every field")
		return
	}
	assert.NotNil(t, result, "GetReferralUserIdentityProfile should return the profile")
}

func TestGetReferralUserInvitations(t *testing.T) {
	t.Parallel()
	_, err := e.GetReferralUserInvitations(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetReferralUserInvitations must reject a nil request")
	start, end := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC)
	_, err = e.GetReferralUserInvitations(t.Context(), &ReferralUserInvitationsRequest{StartTime: end, EndTime: start})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetReferralUserInvitations must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *ReferralUserInvitationsRequest
		exp  *ReferralUserInvitationsResponse
	}{
		{
			name: "qualified invitee in a window",
			req:  &ReferralUserInvitationsRequest{UserIDList: []uint64{100000001}, StartTime: start, EndTime: end, Page: 1, PageSize: 10, FriendStatus: 4},
			exp: &ReferralUserInvitationsResponse{
				Records: []ReferralUserInvitation{
					{
						UserIDMask:        "1***1",
						SignTime:          types.Time(time.UnixMilli(1767312000000)),
						UserStatus:        4,
						TotalCommission:   12.34567891,
						UserID:            "100000001",
						TaskProgress:      "3/5",
						NotShowCommission: true,
						TotalBaseRewards:  5,
					},
				},
				Total: 1,
			},
		},
		{
			// The gateway may quote int64 values, as ProtoJSON does
			name: "second page",
			req:  &ReferralUserInvitationsRequest{Page: 2},
			exp: &ReferralUserInvitationsResponse{
				Records: []ReferralUserInvitation{
					{
						UserIDMask:   "1***9",
						SignTime:     types.Time(time.Unix(1767398400, 0)),
						UserStatus:   1,
						UserID:       "100000009",
						TaskProgress: "0/5",
					},
				},
				Total: 11,
			},
		},
		{
			name: "every invitee",
			req:  &ReferralUserInvitationsRequest{},
			exp:  &ReferralUserInvitationsResponse{Records: []ReferralUserInvitation{}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetReferralUserInvitations(t.Context(), tc.req)
			require.NoError(t, err, "GetReferralUserInvitations must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetReferralUserInvitations should decode every field")
				return
			}
			assert.NotNil(t, result, "GetReferralUserInvitations should return a page")
		})
	}
}

func TestReferralRecordsBody(t *testing.T) {
	t.Parallel()
	start, end := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC)
	_, err := referralRecordsBody(nil, end, start, 0, 0)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "referralRecordsBody must reject a start time after the end time")

	for _, tc := range []struct {
		name           string
		userIDList     []uint64
		start, end     time.Time
		page, pageSize uint64
		exp            map[string]any
	}{
		{name: "no filters", exp: map[string]any{}},
		{
			// Bybit takes UTC date times to the second, so other zones' times and a fraction of a second are converted
			name:       "every filter",
			userIDList: []uint64{201, 202},
			start:      start.In(time.FixedZone("AEDT", 11*60*60)),
			end:        end.Add(999 * time.Millisecond).In(time.FixedZone("EST", -5*60*60)),
			page:       3,
			pageSize:   200,
			exp: map[string]any{
				"userIdList": []map[string]any{{"userId": uint64(201)}, {"userId": uint64(202)}},
				"startTime":  "2026-01-01 00:00:00",
				"endTime":    "2026-01-31 23:59:59",
				"page":       uint64(3),
				"pageSize":   uint64(200),
			},
		},
		{name: "start time alone", start: start, exp: map[string]any{"startTime": "2026-01-01 00:00:00"}},
		{name: "end time alone", end: end, exp: map[string]any{"endTime": "2026-01-31 23:59:59"}},
		{name: "equal times", start: start, end: start, exp: map[string]any{"startTime": "2026-01-01 00:00:00", "endTime": "2026-01-01 00:00:00"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			body, err := referralRecordsBody(tc.userIDList, tc.start, tc.end, tc.page, tc.pageSize)
			require.NoError(t, err, "referralRecordsBody must not error")
			assert.Equal(t, tc.exp, body, "referralRecordsBody should send the filters that are set")
		})
	}
}

func TestReferralCampaignUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var c ReferralCampaign
	require.NoError(t, c.UnmarshalJSON([]byte(`"{\"schemaVersion\":2,\"source\":\"base\",\"campaignId\":\"sample\",\"tasks\":[{\"id\":7,\"awardType\":100,\"ruleTitle\":null,\"ruleDescription\":null,\"ruleParam\":null,\"taskAwards\":[],\"m1Awards\":[]}]}"`)), "UnmarshalJSON must not error")
	exp := ReferralCampaign{
		SchemaVersion: 2,
		Source:        "base",
		CampaignID:    "sample",
		Tasks:         []ReferralCampaignTask{{ID: 7, AwardType: 100, TaskAwards: []ReferralCampaignAward{}, M1Awards: []ReferralCampaignAward{}}},
	}
	assert.Equal(t, exp, c, "UnmarshalJSON should decode the campaign its string encodes")
	require.NoError(t, c.UnmarshalJSON([]byte(`null`)), "UnmarshalJSON must accept null")
	assert.Equal(t, exp, c, "UnmarshalJSON should leave the campaign unchanged for null")

	for _, tc := range []struct{ name, data string }{
		{name: "object", data: `{"schemaVersion":2}`},
		{name: "number", data: `2`},
		{name: "empty string", data: `""`},
		{name: "string that is not JSON", data: `"schemaVersion"`},
		{name: "array", data: `"[]"`},
		{name: "invalid field", data: `"{\"schemaVersion\":\"two\"}"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var c ReferralCampaign
			assert.ErrorIs(t, c.UnmarshalJSON([]byte(tc.data)), errMalformedReferralCampaign, "UnmarshalJSON should reject a malformed campaign")
		})
	}
}

func TestReferralAwardTimeUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, input string
		exp         time.Time
	}{
		{name: "UTC", input: `"2026-10-01T00:00:00Z"`, exp: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)},
		{name: "offset", input: `"2026-10-01T08:00:00+08:00"`, exp: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)},
		{name: "empty", input: `""`},
		{name: "null", input: `null`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// A time already held must be replaced, including by the zero time
			result := ReferralAwardTime(time.Unix(1, 0))
			require.NoError(t, result.UnmarshalJSON([]byte(tc.input)), "UnmarshalJSON must not error")
			assert.Truef(t, result.Time().Equal(tc.exp), "UnmarshalJSON should decode %s as %s, not %s", tc.input, tc.exp, result.Time())
		})
	}
	var result ReferralAwardTime
	assert.ErrorIs(t, result.UnmarshalJSON([]byte(`"2026-10-01 00:00:00"`)), types.ErrInvalidTimestampFormat, "UnmarshalJSON should reject a time that is not in RFC 3339 format")
	assert.ErrorIs(t, result.UnmarshalJSON([]byte(`1790000000000`)), types.ErrInvalidTimestampFormat, "UnmarshalJSON should reject a Unix timestamp")
}

func TestReferralAwardTimeMarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		input time.Time
		exp   string
	}{
		{input: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), exp: `"2026-10-01T00:00:00Z"`},
		{exp: `"0001-01-01T00:00:00Z"`},
	} {
		data, err := json.Marshal(ReferralAwardTime(tc.input))
		require.NoError(t, err, "Marshal must not error")
		assert.Equal(t, tc.exp, string(data), "Marshal should encode the time in RFC 3339 format")
	}
	_, err := json.Marshal(ReferralAwardTime(time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)))
	assert.Error(t, err, "Marshal should reject a year RFC 3339 cannot hold")
}

func TestReferralAwardTimeTime(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, start, ReferralAwardTime(start).Time(), "Time should return the reward's time")
	assert.True(t, ReferralAwardTime{}.Time().IsZero(), "Time should return the zero time for a reward without one")
}
