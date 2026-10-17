package kraken

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

func TestGetAffiliateDailyActivity(t *testing.T) {
	t.Parallel()
	day := time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC)
	start := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	iibans := []string{"AA45N84GQK2VUN7A", "CC34N84GQK2VUN7C"}
	_, err := e.GetAffiliateDailyActivity(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAffiliateDailyActivity must reject a nil request")
	for _, tc := range []struct {
		name string
		req  *AffiliateDailyActivityRequest
		err  error
	}{
		{"a request for neither a day nor a history", &AffiliateDailyActivityRequest{}, common.ErrDateUnset},
		{"a day request with IIBANs", &AffiliateDailyActivityRequest{ActivityDate: day, IIBANs: iibans}, errAffiliateVariantConflict},
		{"a history request for all users", &AffiliateDailyActivityRequest{IIBANs: iibans, StartDate: start, EndDate: day, AllUsers: true}, errAffiliateVariantConflict},
		{"a history request without IIBANs", &AffiliateDailyActivityRequest{StartDate: start, EndDate: day}, errAffiliateIIBANEmpty},
		{"a history request for more than ten IIBANs", &AffiliateDailyActivityRequest{IIBANs: make([]string, 11), StartDate: start, EndDate: day}, errAffiliateTooManyIIBANs},
		{"an empty IIBAN", &AffiliateDailyActivityRequest{IIBANs: []string{"AA45N84GQK2VUN7A", ""}, StartDate: start, EndDate: day}, errAffiliateIIBANEmpty},
		{"a short IIBAN", &AffiliateDailyActivityRequest{IIBANs: []string{"AA45N84GQK"}, StartDate: start, EndDate: day}, errInvalidAffiliateIIBAN},
		{"a duplicate IIBAN", &AffiliateDailyActivityRequest{IIBANs: []string{"AA45N84GQK2VUN7A", "AA45N84GQK2VUN7A"}, StartDate: start, EndDate: day}, errAffiliateDuplicateIIBAN},
		{"a history request without an end date", &AffiliateDailyActivityRequest{IIBANs: iibans, StartDate: start}, common.ErrDateUnset},
		{"a history request starting after its end", &AffiliateDailyActivityRequest{IIBANs: iibans, StartDate: day, EndDate: start}, common.ErrStartAfterEnd},
		{"a limit above 200", &AffiliateDailyActivityRequest{ActivityDate: day, Limit: 201}, errInvalidAffiliateLimit},
	} {
		_, err = e.GetAffiliateDailyActivity(t.Context(), tc.req)
		require.ErrorIsf(t, err, tc.err, "GetAffiliateDailyActivity must reject %s", tc.name)
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		// Kraken only serves the last 90 UTC days, so live requests ask for yesterday
		result, err := e.GetAffiliateDailyActivity(t.Context(), &AffiliateDailyActivityRequest{ActivityDate: time.Now().AddDate(0, 0, -1)})
		require.NoError(t, err, "GetAffiliateDailyActivity must not error")
		assert.False(t, result.Currency.IsEmpty(), "GetAffiliateDailyActivity should return the reporting currency")
		return
	}
	spotTrading := AffiliateProductActivity{
		Volume:     42110.20,
		Fees:       38.20,
		Commission: 11.46,
		EventCount: 12,
		GeoBlocked: AffiliateGeoBlockedActivity{Volume: 1250.50, Fees: 1.25, EventCount: 2},
		Maker:      &AffiliateLiquidityActivity{Volume: 18400, Fees: 9.20, Commission: 2.76, EventCount: 5},
		Taker:      &AffiliateLiquidityActivity{Volume: 23710.20, Fees: 29, Commission: 8.70, EventCount: 7},
	}
	marginRollover := AffiliateProductActivity{Fees: 2.40, Commission: 0.72, EventCount: 3}
	quietReferee := AffiliateRefereeActivity{
		IIBAN:       "BB12N84GQK2VUN7B",
		MaskedIIBAN: "UN7B",
		Plans: []AffiliatePlan{
			{ReferralCode: "EXAMPLE1", ReferralLevel: 1, EnrolledAt: time.Date(2026, 9, 15, 9, 0, 0, 0, time.UTC), Status: "active", Earning: true},
		},
		RefereeReference: "example_quiet_user",
	}
	optedOutReferee := AffiliateRefereeActivity{
		Plans:            []AffiliatePlan{{ReferralCode: "EXAMPLE2"}},
		RefereeReference: "example_opted_out_user",
		OptedOut:         true,
		EnrolledAt:       time.Date(2026, 8, 20, 9, 0, 0, 0, time.UTC),
	}
	for _, tc := range []struct {
		name string
		req  *AffiliateDailyActivityRequest
		exp  *AffiliateDailyActivityResponse
	}{
		{
			name: "day roster",
			req:  &AffiliateDailyActivityRequest{ActivityDate: day, AllUsers: true, Limit: 50},
			exp: &AffiliateDailyActivityResponse{
				ActivityDate: AffiliateDate(day),
				Currency:     currency.USD,
				Revision:     time.Date(2026, 9, 17, 0, 14, 2, 0, time.UTC),
				GeneratedAt:  time.Date(2026, 9, 17, 12, 1, 4, 0, time.UTC),
				ActiveUsers:  1,
				Totals:       map[string]AffiliateProductActivity{"spot_trading": spotTrading, "margin_rollover": marginRollover},
				NextCursor:   "eyJyZWYiOiJleGFtcGxlX3F1aWV0X3VzZXIifQ",
				Referees: []AffiliateRefereeActivity{
					{
						IIBAN:       "AA45N84GQK2VUN7A",
						MaskedIIBAN: "UN7A",
						Plans: []AffiliatePlan{
							{
								ReferralCode:  "EXAMPLE1",
								Campaign:      "tg-vip",
								ReferralLevel: 1,
								EnrolledAt:    time.Date(2026, 3, 14, 9, 0, 0, 0, time.UTC),
								Status:        "active",
								Earning:       true,
								ExpiresAt:     time.Date(2027, 3, 14, 9, 21, 7, 0, time.UTC),
								Products:      map[string]AffiliateProductActivity{"spot_trading": spotTrading, "margin_rollover": marginRollover},
							},
						},
						RefereeReference: "jsd6ggr6qjrzmemn",
					},
					quietReferee,
					optedOutReferee,
				},
				Estimated: true,
				OptedOut: AffiliateOptedOutActivity{Summary: &AffiliateOptedOutSummary{
					ActiveParticipants: 2,
					Products: map[string]AffiliateProductActivity{
						"futures_order_fill": {
							Volume:     98000,
							Fees:       49,
							Commission: 14.70,
							EventCount: 9,
							Maker:      &AffiliateLiquidityActivity{Volume: 40000, Fees: 8, Commission: 2.40, EventCount: 4},
							Taker:      &AffiliateLiquidityActivity{Volume: 58000, Fees: 41, Commission: 12.30, EventCount: 5},
						},
					},
				}},
				Limit: 50,
			},
		},
		{
			name: "day roster next page",
			req:  &AffiliateDailyActivityRequest{ActivityDate: day, AllUsers: true, Cursor: "eyJyZWYiOiJleGFtcGxlX3F1aWV0X3VzZXIifQ", Limit: 50},
			exp: &AffiliateDailyActivityResponse{
				ActivityDate: AffiliateDate(day),
				Currency:     currency.USD,
				GeneratedAt:  time.Date(2026, 9, 17, 12, 1, 5, 0, time.UTC),
				Referees: []AffiliateRefereeActivity{
					{
						IIBAN:       "DD56N84GQK2VUN7D",
						MaskedIIBAN: "UN7D",
						Plans: []AffiliatePlan{
							{ReferralCode: "EXAMPLE3", ReferralLevel: 1, EnrolledAt: time.Date(2026, 1, 5, 14, 0, 0, 0, time.UTC), Status: "completed", Earning: true},
						},
						RefereeReference: "example_roster_user",
					},
				},
				Limit: 50,
			},
		},
		{
			name: "day with suppressed opted-out activity",
			req:  &AffiliateDailyActivityRequest{ActivityDate: time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)},
			exp: &AffiliateDailyActivityResponse{
				ActivityDate: AffiliateDate(time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)),
				Currency:     currency.USD,
				GeneratedAt:  time.Date(2026, 9, 15, 12, 1, 4, 0, time.UTC),
				Referees:     []AffiliateRefereeActivity{quietReferee},
				OptedOut:     AffiliateOptedOutActivity{Suppressed: &AffiliateSuppressedActivity{}},
				Limit:        50,
			},
		},
		{
			name: "history",
			req:  &AffiliateDailyActivityRequest{IIBANs: iibans, StartDate: start, EndDate: day, Limit: 10},
			exp: &AffiliateDailyActivityResponse{
				Currency:    currency.USD,
				GeneratedAt: time.Date(2026, 9, 17, 12, 1, 4, 0, time.UTC),
				Referees: []AffiliateRefereeActivity{
					{
						ActivityDate: AffiliateDate(day),
						IIBAN:        "AA45N84GQK2VUN7A",
						MaskedIIBAN:  "UN7A",
						Plans: []AffiliatePlan{
							{
								ReferralCode:  "EXAMPLE1",
								ReferralLevel: 1,
								EnrolledAt:    time.Date(2026, 3, 14, 9, 0, 0, 0, time.UTC),
								Status:        "active",
								Earning:       true,
								Products:      map[string]AffiliateProductActivity{"futures_order_fill": {Volume: 15000, Fees: 7.50, Commission: 2.25, EventCount: 4}},
							},
						},
						RefereeReference: "jsd6ggr6qjrzmemn",
						Estimated:        true,
					},
					optedOutReferee,
				},
				Limit: 10,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetAffiliateDailyActivity(t.Context(), tc.req)
			require.NoError(t, err, "GetAffiliateDailyActivity must not error")
			assert.Equal(t, tc.exp, result, "GetAffiliateDailyActivity should decode every field")
		})
	}
}

func TestGetAffiliateCPAProgress(t *testing.T) {
	t.Parallel()
	_, err := e.GetAffiliateCPAProgress(t.Context(), "")
	require.ErrorIs(t, err, errAffiliateIIBANEmpty, "GetAffiliateCPAProgress must reject an empty IIBAN")
	_, err = e.GetAffiliateCPAProgress(t.Context(), "AA45N84GQK")
	require.ErrorIs(t, err, errInvalidAffiliateIIBAN, "GetAffiliateCPAProgress must reject an IIBAN shorter than 14 characters")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAffiliateCPAProgress(t.Context(), "AA45N84GQK2VUN7A")
	require.NoError(t, err, "GetAffiliateCPAProgress must not error")
	if !mockTests {
		assert.NotNil(t, result, "GetAffiliateCPAProgress should return the referee's progress")
		return
	}
	exp := &AffiliateCPAProgressResponse{
		Progress: AffiliateCPAProgress{
			Bounties: []AffiliateCPABounty{
				{
					BountyPosition:       1,
					Description:          "Complete the first CPA requirements",
					RewardAmount:         50,
					RewardAsset:          currency.USDC,
					Status:               "in_progress",
					ProgressAvailability: "available",
					Conditions: []AffiliateCPACondition{
						{
							Label:       "Margin trading volume",
							Kind:        "amount",
							State:       "incomplete",
							Measurement: &AffiliateCPAMeasurement{Current: 20, Target: 100, Unit: currency.USD, Percentage: 20},
						},
						{
							Label:       "Trades placed",
							Kind:        "count",
							State:       "incomplete",
							Measurement: &AffiliateCPAMeasurement{Current: 3, Target: 5, Percentage: 60},
						},
						{Label: "Account verification", Kind: "binary", State: "complete"},
					},
				},
				{
					BountyPosition:       2,
					Description:          "Complete the second CPA requirements",
					RewardAmount:         80,
					RewardAsset:          currency.USDT,
					Status:               "qualified_payment_pending",
					QualifiedAt:          time.Date(2026, 9, 30, 14, 22, 9, 0, time.UTC),
					ProgressAvailability: "unavailable",
					Conditions:           []AffiliateCPACondition{{Label: "Futures trading volume", Kind: "amount", State: "complete"}},
				},
			},
		},
	}
	assert.Equal(t, exp, result, "GetAffiliateCPAProgress should decode every field")
}

func TestGetAffiliatePayoutHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetAffiliatePayoutHistory(t.Context(), &AffiliatePayoutHistoryRequest{Limit: 201})
	require.ErrorIs(t, err, errInvalidAffiliateLimit, "GetAffiliatePayoutHistory must reject a limit above 200")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	summary := AffiliatePayoutSummary{
		Paid:    312,
		Pending: 130.50,
		Prop:    AffiliatePropPayoutSummary{Paid: 80, Pending: 30.50, OrderCount: 3},
		CPA:     AffiliateCPAPayoutSummary{Paid: 220, Pending: 100, BountyCount: 5},
	}
	newestSummary := summary
	newestSummary.Paid = 324
	for _, tc := range []struct {
		name string
		req  *AffiliatePayoutHistoryRequest
		exp  *AffiliatePayoutHistoryResponse
	}{
		{
			name: "nil request",
			exp: &AffiliatePayoutHistoryResponse{
				Payouts: []AffiliatePayout{
					{PayoutID: "RPAY-EXAMPLE-REVSHARE", Amount: 12, Asset: currency.USDC, Status: "paid", CreatedAt: time.Date(2026, 10, 5, 9, 10, 0, 0, time.UTC), Source: "revshare"},
				},
				NextCursor: "RPAY-EXAMPLE-REVSHARE",
				Summary:    newestSummary,
				Limit:      50,
			},
		},
		{
			name: "next page",
			req:  &AffiliatePayoutHistoryRequest{Cursor: "RPAY-EXAMPLE-REVSHARE", Limit: 3},
			exp: &AffiliatePayoutHistoryResponse{
				Payouts: []AffiliatePayout{
					{PayoutID: "RPAY-EXAMPLE-CPA", Amount: 50, Asset: currency.USDC, Status: "paid", CreatedAt: time.Date(2026, 9, 30, 15, 4, 12, 0, time.UTC), Source: "cpa"},
					{
						PayoutID:       "RPAY-EXAMPLE-PROP",
						Amount:         30.50,
						Asset:          currency.USDT,
						Status:         "pending",
						CreatedAt:      time.Date(2026, 9, 29, 9, 10, 0, 0, time.UTC),
						Source:         "prop",
						PropPurchaseID: "BRP-EXAMPLE-ORDER",
					},
					{
						PayoutID:  "RPAY-EXAMPLE-FAB",
						Amount:    12.75,
						Asset:     currency.USDC,
						Status:    "on_hold",
						CreatedAt: time.Date(2026, 9, 28, 9, 10, 0, 0, time.UTC),
						Source:    "futures_accelerator_bonus",
					},
				},
				NextCursor: "RPAY-EXAMPLE-FAB",
				Summary:    summary,
				Limit:      3,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetAffiliatePayoutHistory(t.Context(), tc.req)
			require.NoError(t, err, "GetAffiliatePayoutHistory must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetAffiliatePayoutHistory should decode every field")
				return
			}
			assert.NotZero(t, result.Limit, "GetAffiliatePayoutHistory should return the page size")
		})
	}
}

func TestAffiliateDateUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var date AffiliateDate
	require.NoError(t, date.UnmarshalJSON([]byte(`"2026-09-16"`)), "UnmarshalJSON must not error for a date")
	assert.Equal(t, time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC), date.Time(), "UnmarshalJSON should decode the day in UTC")
	require.NoError(t, date.UnmarshalJSON([]byte(`""`)), "UnmarshalJSON must not error for an empty date")
	assert.True(t, date.Time().IsZero(), "UnmarshalJSON should leave an empty date zero")
	var parseErr *time.ParseError
	assert.ErrorAs(t, date.UnmarshalJSON([]byte(`"2026-09-16T00:00:00Z"`)), &parseErr, "UnmarshalJSON should reject a timestamp")
	assert.Error(t, date.UnmarshalJSON([]byte(`20260916`)), "UnmarshalJSON should reject a number")
}

func TestValidateAffiliateIIBAN(t *testing.T) {
	t.Parallel()
	assert.ErrorIs(t, validateAffiliateIIBAN(""), errAffiliateIIBANEmpty, "validateAffiliateIIBAN should reject an empty IIBAN")
	assert.ErrorIs(t, validateAffiliateIIBAN("AA45N84GQK2VU"), errInvalidAffiliateIIBAN, "validateAffiliateIIBAN should reject an IIBAN shorter than 14 characters")
	assert.NoError(t, validateAffiliateIIBAN("AA45N84GQK2VUN"), "validateAffiliateIIBAN should accept an IIBAN of 14 characters")
	assert.NoError(t, validateAffiliateIIBAN("AA45N84GQK2VUN7A"), "validateAffiliateIIBAN should accept a whole IIBAN")
}

func TestSetAffiliatePage(t *testing.T) {
	t.Parallel()
	params := url.Values{}
	setAffiliatePage(params, "", 0)
	assert.Empty(t, params, "setAffiliatePage should add nothing without a cursor or limit")
	setAffiliatePage(params, "eyJwYWdlIjoyfQ", 50)
	assert.Equal(t, url.Values{"cursor": {"eyJwYWdlIjoyfQ"}, "limit": {"50"}}, params, "setAffiliatePage should add the cursor and limit")
}
