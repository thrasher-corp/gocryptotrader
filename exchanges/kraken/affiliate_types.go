package kraken

import (
	"errors"
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

var (
	errAffiliateIIBANEmpty      = errors.New("affiliate IIBAN is empty")
	errInvalidAffiliateIIBAN    = errors.New("invalid affiliate IIBAN")
	errAffiliateTooManyIIBANs   = errors.New("too many affiliate IIBANs")
	errAffiliateDuplicateIIBAN  = errors.New("duplicate affiliate IIBAN")
	errAffiliateVariantConflict = errors.New("affiliate day and history requests cannot be combined")
	errInvalidAffiliateLimit    = errors.New("invalid affiliate page limit")
)

// AffiliateDate is a UTC calendar day, which the Affiliate REST API sends as YYYY-MM-DD
type AffiliateDate time.Time

// UnmarshalJSON decodes a YYYY-MM-DD date
func (d *AffiliateDate) UnmarshalJSON(data []byte) error {
	var date string
	if err := json.Unmarshal(data, &date); err != nil {
		return err
	}
	if date == "" {
		*d = AffiliateDate{}
		return nil
	}
	t, err := time.Parse(time.DateOnly, date)
	if err != nil {
		return fmt.Errorf("error parsing affiliate date: %w", err)
	}
	*d = AffiliateDate(t)
	return nil
}

// Time returns the start of the day in UTC
func (d AffiliateDate) Time() time.Time {
	return time.Time(d)
}

// AffiliateDailyActivityRequest holds the parameters of Get Daily Activity. A request is either for a day, setting
// ActivityDate, or for referees' history, setting IIBANs, StartDate and EndDate. Dates must be today or within the
// previous 89 days, and their UTC calendar day is sent
type AffiliateDailyActivityRequest struct {
	ActivityDate time.Time
	// AllUsers adds every referee enrolled by the day and still enrolled to a day's items, whether or not they traded
	AllUsers bool
	// IIBANs lists up to ten full Kraken account identifiers, as returned in a referee's IIBAN
	IIBANs    []string
	StartDate time.Time
	EndDate   time.Time
	// Cursor requests the page after the one that returned it; the request's other parameters must not change
	Cursor string
	// Limit is the page size, up to 200, and 50 when it is 0
	Limit uint64
}

// AffiliateDailyActivityResponse holds a page of referees' daily activity. Day-wide fields are only returned on a day
// request's first page, and amounts are in Currency
type AffiliateDailyActivityResponse struct {
	// ActivityDate is the day of a day request's items
	ActivityDate AffiliateDate `json:"activity_date"`
	// Currency is the currency amounts are reported in, which is not the payout asset
	Currency currency.Code `json:"currency"`
	// Revision is when the day's figures were last written; a later revision means they changed. Enrolments and opt-outs
	// leave it unchanged
	Revision    time.Time `json:"revision"`
	GeneratedAt time.Time `json:"generated_at"`
	// ActiveUsers counts the referees sharing trading activity that day, leaving out quiet and opted-out referees
	ActiveUsers uint64 `json:"active_users"`
	// Totals holds the day's activity by product, including that of referees no longer enrolled, who are left out of
	// Referees, but not that of referees who opted out
	Totals map[string]AffiliateProductActivity `json:"totals"`
	// NextCursor requests the next page, and is empty on the last
	NextCursor string                     `json:"next_cursor"`
	Referees   []AffiliateRefereeActivity `json:"items"`
	// Estimated is whether the day's amounts are still estimates
	Estimated bool                      `json:"estimated"`
	OptedOut  AffiliateOptedOutActivity `json:"opted_out"`
	Limit     uint64                    `json:"limit"`
}

// AffiliateRefereeActivity is a referee's activity by plan and product, or the stub of a referee who opted out of
// sharing it, which holds only the referee reference, enrolment time and plans' referral codes
type AffiliateRefereeActivity struct {
	// ActivityDate is the day of a history request's entry, and is zero on an opted-out stub
	ActivityDate AffiliateDate `json:"activity_date"`
	// IIBAN is the referee's full Kraken account identifier, as history requests and GetAffiliateCPAProgress take it
	IIBAN string `json:"iiban"`
	// MaskedIIBAN is the IIBAN's last four characters, which are not unique
	MaskedIIBAN string          `json:"masked_iiban"`
	Plans       []AffiliatePlan `json:"plans"`
	// RefereeReference identifies the referee across days, but not across affiliates, and is no Kraken account ID
	RefereeReference string `json:"referee_reference"`
	// OptedOut marks the stub of a referee whose activity is withheld, which is not zero activity
	OptedOut bool `json:"opted_out"`
	// EnrolledAt is an opted-out stub's enrolment time, truncated to the hour
	EnrolledAt time.Time `json:"enrolled_at"`
	// Estimated is whether a history entry's amounts are still estimates
	Estimated bool `json:"estimated"`
}

// AffiliatePlan is an affiliate plan a referee was referred through
type AffiliatePlan struct {
	ReferralCode string `json:"referral_code"`
	// Campaign is the campaign tag of the referral code the referee signed up with
	Campaign      string `json:"campaign"`
	ReferralLevel uint64 `json:"referral_level"`
	// EnrolledAt is truncated to the hour
	EnrolledAt time.Time `json:"enrolled_at"`
	// Status is active or completed while activity earns commission, though other values may be returned
	Status    string    `json:"status"`
	Earning   bool      `json:"earning"`
	ExpiresAt time.Time `json:"expires_at"`
	// Products holds the referee's activity by product; a product left out had no reported activity
	Products map[string]AffiliateProductActivity `json:"products"`
}

// AffiliateProductActivity is the activity on a product, such as spot_trading, that can earn commission. Activity from
// regions commission cannot be paid for is only counted in GeoBlocked
type AffiliateProductActivity struct {
	// Volume is zero for products charging fees alone, such as margin_rollover
	Volume     types.Number                `json:"volume"`
	Fees       types.Number                `json:"fees"`
	Commission types.Number                `json:"commission"`
	EventCount uint64                      `json:"event_count"`
	GeoBlocked AffiliateGeoBlockedActivity `json:"geo_blocked"`
	// Maker and Taker are only set on execution products, and only for fills classified as making or taking liquidity;
	// nil is unclassified rather than zero activity
	Maker *AffiliateLiquidityActivity `json:"maker"`
	Taker *AffiliateLiquidityActivity `json:"taker"`
}

// AffiliateGeoBlockedActivity is activity from regions commission cannot be paid for
type AffiliateGeoBlockedActivity struct {
	Volume     types.Number `json:"volume"`
	Fees       types.Number `json:"fees"`
	EventCount uint64       `json:"event_count"`
}

// AffiliateLiquidityActivity is the activity making or taking liquidity on a product
type AffiliateLiquidityActivity struct {
	Volume     types.Number `json:"volume"`
	Fees       types.Number `json:"fees"`
	Commission types.Number `json:"commission"`
	EventCount uint64       `json:"event_count"`
}

// AffiliateOptedOutActivity is the activity of referees who opted out of sharing it. Suppressed is set while no more
// than one of them has reportable activity, and Summary combines their activity once two or more have; both are nil
// when no referee opted out
type AffiliateOptedOutActivity struct {
	Suppressed *AffiliateSuppressedActivity `json:"suppressed"`
	Summary    *AffiliateOptedOutSummary    `json:"summary"`
}

// AffiliateSuppressedActivity marks opted-out activity that is withheld; Kraken documents no fields for it
type AffiliateSuppressedActivity struct{}

// AffiliateOptedOutSummary combines the activity of referees who opted out of sharing it, which counts towards the day's
// payable activity alongside the totals
type AffiliateOptedOutSummary struct {
	ActiveParticipants uint64                              `json:"active_participants"`
	Products           map[string]AffiliateProductActivity `json:"products"`
}

// AffiliateCPAProgressResponse holds a referee's progress towards each CPA bounty
type AffiliateCPAProgressResponse struct {
	Progress AffiliateCPAProgress `json:"progress"`
}

// AffiliateCPAProgress holds a referral's CPA bounties. Bounties for regions the referee is not in are left out
type AffiliateCPAProgress struct {
	Bounties []AffiliateCPABounty `json:"bounties"`
}

// AffiliateCPABounty is a referee's progress towards a CPA bounty
type AffiliateCPABounty struct {
	// BountyPosition is the bounty's position in the affiliate plan, from 0, which changes when the plan does
	BountyPosition uint64 `json:"bounty_position"`
	Description    string `json:"description"`
	// RewardAmount is an estimate until a payment is recorded, from the plan's amount, the conversion rate and the
	// affiliate's share, and the recorded payment's amount after
	RewardAmount types.Number  `json:"reward_amount"`
	RewardAsset  currency.Code `json:"reward_asset"`
	// Status is in_progress, qualified_payment_pending, paid, expired, no_longer_eligible or not_tracked, though values
	// may be added
	Status      string    `json:"status"`
	QualifiedAt time.Time `json:"qualified_at"`
	// ProgressAvailability is available or unavailable, measurements being left out while unavailable
	ProgressAvailability string                  `json:"progress_availability"`
	Conditions           []AffiliateCPACondition `json:"conditions"`
}

// AffiliateCPACondition is a requirement of a CPA bounty
type AffiliateCPACondition struct {
	Label string `json:"label"`
	// Kind is amount, count, verification or binary, though values may be added
	Kind string `json:"kind"`
	// State is incomplete or complete
	State string `json:"state"`
	// Measurement is only set for measurable requirements while progress is available
	Measurement *AffiliateCPAMeasurement `json:"measurement"`
}

// AffiliateCPAMeasurement is the progress towards a measurable CPA requirement
type AffiliateCPAMeasurement struct {
	// Current is capped between zero and Target
	Current types.Number `json:"current"`
	Target  types.Number `json:"target"`
	// Unit is the asset of a monetary or volume requirement, and is empty for counts
	Unit currency.Code `json:"unit"`
	// Percentage runs from 0 to 100, and may stay at 100 while a step that cannot be measured, such as a holding period,
	// is incomplete
	Percentage types.Number `json:"percentage"`
}

// AffiliatePayoutHistoryRequest holds the parameters of Get Payout History
type AffiliatePayoutHistoryRequest struct {
	// Cursor requests the page after the one that returned it, which Kraken asks to keep the same Limit
	Cursor string
	// Limit is the page size, up to 200, and 50 when it is 0
	Limit uint64
}

// AffiliatePayoutHistoryResponse holds a page of payouts, newest first, and totals covering the whole history.
// Retainer, sub-affiliate carve-out and remediation payouts are left out
type AffiliatePayoutHistoryResponse struct {
	Payouts []AffiliatePayout `json:"items"`
	// NextCursor requests the next page, and is empty on the last
	NextCursor string                 `json:"next_cursor"`
	Summary    AffiliatePayoutSummary `json:"summary"`
	Limit      uint64                 `json:"limit"`
}

// AffiliatePayout is an affiliate payout
type AffiliatePayout struct {
	PayoutID string        `json:"payout_id"`
	Amount   types.Number  `json:"amount"`
	Asset    currency.Code `json:"asset"`
	// Status is pending, paid, action_needed, on_hold, refunded or cancelled, though values may be added
	Status string `json:"status"`
	// CreatedAt is when the payout record was created
	CreatedAt time.Time `json:"created_at"`
	// Source is revshare, prop, cpa or futures_accelerator_bonus, though values may be added
	Source string `json:"source"`
	// PropPurchaseID is only set for Prop commissions
	PropPurchaseID string `json:"prop_purchase_id"`
}

// AffiliatePayoutSummary holds the totals of the whole payout history. Payouts with a status other than paid or
// pending are not counted. The combined totals add payout assets expected to be pegged to USD
type AffiliatePayoutSummary struct {
	Paid    types.Number               `json:"paid"`
	Pending types.Number               `json:"pending"`
	Prop    AffiliatePropPayoutSummary `json:"prop"`
	CPA     AffiliateCPAPayoutSummary  `json:"cpa"`
}

// AffiliatePropPayoutSummary holds the Prop commission totals of the whole payout history
type AffiliatePropPayoutSummary struct {
	Paid    types.Number `json:"paid"`
	Pending types.Number `json:"pending"`
	// OrderCount counts the Prop orders paid or pending
	OrderCount uint64 `json:"order_count"`
}

// AffiliateCPAPayoutSummary holds the CPA bounty totals of the whole payout history
type AffiliateCPAPayoutSummary struct {
	Paid    types.Number `json:"paid"`
	Pending types.Number `json:"pending"`
	// BountyCount counts the CPA bounties paid or pending
	BountyCount uint64 `json:"bounty_count"`
}
