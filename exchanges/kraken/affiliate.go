package kraken

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
)

// GetAffiliateDailyActivity calls Get Daily Activity, returning a page of referees' activity on a UTC day, or on each
// day of a date range for up to ten referees, never summed across days. A day request lists the referees who traded or
// enrolled that day; AllUsers adds every referee still enrolled, and stubs for those who opted out of sharing
func (e *Exchange) GetAffiliateDailyActivity(ctx context.Context, req *AffiliateDailyActivityRequest) (*AffiliateDailyActivityResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Limit > 200 {
		return nil, fmt.Errorf("%w: %d exceeds 200", errInvalidAffiliateLimit, req.Limit)
	}
	params := url.Values{}
	history := len(req.IIBANs) != 0 || !req.StartDate.IsZero() || !req.EndDate.IsZero()
	switch {
	case !req.ActivityDate.IsZero() && history:
		return nil, errAffiliateVariantConflict
	case !req.ActivityDate.IsZero():
		params.Set("activity_date", req.ActivityDate.UTC().Format(time.DateOnly))
		if req.AllUsers {
			params.Set("all_users", "true")
		}
	case history:
		if req.AllUsers {
			return nil, fmt.Errorf("%w: all users is only for day requests", errAffiliateVariantConflict)
		}
		if len(req.IIBANs) == 0 {
			return nil, errAffiliateIIBANEmpty
		}
		if len(req.IIBANs) > 10 {
			return nil, fmt.Errorf("%w: %d exceeds 10", errAffiliateTooManyIIBANs, len(req.IIBANs))
		}
		for i := range req.IIBANs {
			if err := validateAffiliateIIBAN(req.IIBANs[i]); err != nil {
				return nil, err
			}
			if slices.Contains(req.IIBANs[:i], req.IIBANs[i]) {
				return nil, fmt.Errorf("%w: %s", errAffiliateDuplicateIIBAN, req.IIBANs[i])
			}
		}
		if req.StartDate.IsZero() || req.EndDate.IsZero() {
			return nil, fmt.Errorf("%w: a history request needs start and end dates", common.ErrDateUnset)
		}
		if req.StartDate.After(req.EndDate) {
			return nil, common.ErrStartAfterEnd
		}
		params.Set("iiban", strings.Join(req.IIBANs, ","))
		params.Set("start_date", req.StartDate.UTC().Format(time.DateOnly))
		params.Set("end_date", req.EndDate.UTC().Format(time.DateOnly))
	default:
		return nil, fmt.Errorf("%w: an activity date, or IIBANs with start and end dates, is required", common.ErrDateUnset)
	}
	setAffiliatePage(params, req.Cursor, req.Limit)
	var resp *AffiliateDailyActivityResponse
	return resp, e.SendHeaderAuthenticatedHTTPRequest(ctx, http.MethodGet, "/affiliate/v1/daily-activity", params, nil, &resp)
}

// GetAffiliateCPAProgress calls Get CPA Progress, returning a referee's progress towards each CPA bounty, which stays
// listed once qualified or paid. iiban is the referee's full Kraken account identifier, as GetAffiliateDailyActivity
// returns it. Kraken replies not found for an unknown referee, another affiliate's referee or a referee who opted out
func (e *Exchange) GetAffiliateCPAProgress(ctx context.Context, iiban string) (*AffiliateCPAProgressResponse, error) {
	if err := validateAffiliateIIBAN(iiban); err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("iiban", iiban)
	var resp *AffiliateCPAProgressResponse
	return resp, e.SendHeaderAuthenticatedHTTPRequest(ctx, http.MethodGet, "/affiliate/v1/cpa-progress", params, nil, &resp)
}

// GetAffiliatePayoutHistory calls Get Payout History, returning a page of payouts, newest first, with totals covering
// the whole history. A nil request returns the newest 50 payouts
func (e *Exchange) GetAffiliatePayoutHistory(ctx context.Context, req *AffiliatePayoutHistoryRequest) (*AffiliatePayoutHistoryResponse, error) {
	params := url.Values{}
	if req != nil {
		if req.Limit > 200 {
			return nil, fmt.Errorf("%w: %d exceeds 200", errInvalidAffiliateLimit, req.Limit)
		}
		setAffiliatePage(params, req.Cursor, req.Limit)
	}
	var resp *AffiliatePayoutHistoryResponse
	return resp, e.SendHeaderAuthenticatedHTTPRequest(ctx, http.MethodGet, "/affiliate/v1/payout-history", params, nil, &resp)
}

// validateAffiliateIIBAN rejects an empty IIBAN, or one shorter than the 14 characters Kraken documents as the minimum
func validateAffiliateIIBAN(iiban string) error {
	if iiban == "" {
		return errAffiliateIIBANEmpty
	}
	if len(iiban) < 14 {
		return fmt.Errorf("%w: %q is shorter than 14 characters", errInvalidAffiliateIIBAN, iiban)
	}
	return nil
}

// setAffiliatePage adds the pagination parameters of an Affiliate REST request
func setAffiliatePage(params url.Values, cursor string, limit uint64) {
	if cursor != "" {
		params.Set("cursor", cursor)
	}
	if limit != 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
}
