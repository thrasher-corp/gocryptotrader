package okx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

var (
	errInvalidOKUSDClientOrderID = errors.New("invalid OKUSD client order ID")
	errInvalidOKUSDRedeemType    = errors.New("invalid OKUSD redeem type")
)

var okusdClientOrderIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,32}$`)

// GetOKUSDLimits returns the master account's remaining daily OKUSD subscription and redemption quotas
func (e *Exchange) GetOKUSDLimits(ctx context.Context) (*OKUSDLimitsResponse, error) {
	var resp *OKUSDLimitsResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, getOKUSDLimitsEPL, http.MethodGet, "finance/okusd/limits", nil, &resp, request.AuthenticatedRequest); err != nil {
		return nil, err
	}
	return resp, nil
}

// SubscribeOKUSD subscribes USDT from the funding account for OKUSD at 1:1; resubmitting a client order ID returns the
// original order without subscribing again
func (e *Exchange) SubscribeOKUSD(ctx context.Context, arg *OKUSDSubscribeRequest) (*OKUSDSubscribeResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if err := validateOKUSDTransaction(arg.Amount, arg.ClientOrderID); err != nil {
		return nil, err
	}
	var resp *OKUSDSubscribeResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, subscribeOKUSDEPL, http.MethodPost, "finance/okusd/subscribe", arg, &resp, request.AuthenticatedRequest); err != nil {
		return nil, err
	}
	return resp, nil
}

// RedeemOKUSD redeems OKUSD from the funding account for USDT by fast or standard redemption; resubmitting a client
// order ID returns the original order without redeeming again
func (e *Exchange) RedeemOKUSD(ctx context.Context, arg *OKUSDRedeemRequest) (*OKUSDRedeemResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if err := validateOKUSDTransaction(arg.Amount, arg.ClientOrderID); err != nil {
		return nil, err
	}
	switch arg.RedeemType {
	case OKUSDRedeemTypeFast, OKUSDRedeemTypeStandard:
	case "":
		return nil, fmt.Errorf("%w: redeemType", errMissingRequiredParameter)
	default:
		return nil, fmt.Errorf("%w %q: OKX takes %q for fast or %q for standard redemption", errInvalidOKUSDRedeemType, arg.RedeemType, OKUSDRedeemTypeFast, OKUSDRedeemTypeStandard)
	}
	var resp *OKUSDRedeemResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, redeemOKUSDEPL, http.MethodPost, "finance/okusd/redeem", arg, &resp, request.AuthenticatedRequest); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetOKUSDAccount returns the master account's OKUSD balance and accrued yield
func (e *Exchange) GetOKUSDAccount(ctx context.Context) (*OKUSDAccountResponse, error) {
	var resp *OKUSDAccountResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, getOKUSDAccountEPL, http.MethodGet, "finance/okusd/account", nil, &resp, request.AuthenticatedRequest); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetOKUSDSubscriptionHistory returns the account's OKUSD subscriptions, newest first
func (e *Exchange) GetOKUSDSubscriptionHistory(ctx context.Context, arg *OKUSDHistoryRequest) ([]OKUSDSubscriptionHistoryResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	params, err := okusdHistoryValues(arg.Limit, arg.Begin, arg.End)
	if err != nil {
		return nil, err
	}
	var resp []OKUSDSubscriptionHistoryResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, getOKUSDSubscriptionHistoryEPL, http.MethodGet, common.EncodeURLValues("finance/okusd/subscribe/history", params), nil, &resp, request.AuthenticatedRequest); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetOKUSDRedemptionHistory returns the account's OKUSD redemptions, newest first; OKX returns standard redemptions
// when no type is set
func (e *Exchange) GetOKUSDRedemptionHistory(ctx context.Context, arg *OKUSDRedemptionHistoryRequest) ([]OKUSDRedemptionHistoryResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	params, err := okusdHistoryValues(arg.Limit, arg.Begin, arg.End)
	if err != nil {
		return nil, err
	}
	if arg.Type != "" {
		params.Set("type", arg.Type)
	}
	var resp []OKUSDRedemptionHistoryResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, getOKUSDRedemptionHistoryEPL, http.MethodGet, common.EncodeURLValues("finance/okusd/redeem/history", params), nil, &resp, request.AuthenticatedRequest); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetOKUSDRewardsHistory returns the account's daily OKUSD yield distributions, newest first
func (e *Exchange) GetOKUSDRewardsHistory(ctx context.Context, arg *OKUSDHistoryRequest) ([]OKUSDRewardsHistoryResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	params, err := okusdHistoryValues(arg.Limit, arg.Begin, arg.End)
	if err != nil {
		return nil, err
	}
	var resp []OKUSDRewardsHistoryResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, getOKUSDRewardsHistoryEPL, http.MethodGet, common.EncodeURLValues("finance/okusd/rewards/history", params), nil, &resp, request.AuthenticatedRequest); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetOKUSDAPRHistory returns OKUSD's APR snapshots, newest first
func (e *Exchange) GetOKUSDAPRHistory(ctx context.Context, arg *OKUSDHistoryRequest) ([]OKUSDAPRHistoryResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	params, err := okusdHistoryValues(arg.Limit, arg.Begin, arg.End)
	if err != nil {
		return nil, err
	}
	var resp []OKUSDAPRHistoryResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, getOKUSDAPRHistoryEPL, http.MethodGet, common.EncodeURLValues("finance/okusd/rate/history", params), nil, &resp, request.AuthenticatedRequest); err != nil {
		return nil, err
	}
	return resp, nil
}

// validateOKUSDTransaction checks a subscription's or redemption's amount and client order ID against the formats OKX
// documents for them
func validateOKUSDTransaction(amount float64, clientOrderID string) error {
	if amount < 1 {
		return fmt.Errorf("%w: amount %v is below OKX's minimum of 1", limits.ErrAmountBelowMin, amount)
	}
	// The request sends the amount in its shortest decimal form
	if _, decimals, _ := strings.Cut(strconv.FormatFloat(amount, 'f', -1, 64), "."); len(decimals) > 8 {
		return fmt.Errorf("%w: amount %v has more than the 8 decimal places OKX takes", limits.ErrAmountExceedsStep, amount)
	}
	if clientOrderID == "" {
		return order.ErrClientOrderIDMustBeSet
	}
	if !okusdClientOrderIDPattern.MatchString(clientOrderID) {
		return fmt.Errorf("%w %q: OKX takes up to 32 letters, digits, hyphens and underscores", errInvalidOKUSDClientOrderID, clientOrderID)
	}
	return nil
}

// okusdHistoryValues validates an OKUSD history time window and encodes it with the page size as query parameters
func okusdHistoryValues(limit uint64, begin, end time.Time) (url.Values, error) {
	if !begin.IsZero() && !end.IsZero() && begin.After(end) {
		return nil, fmt.Errorf("%w: begin %s is after end %s", common.ErrStartAfterEnd, begin, end)
	}
	params := url.Values{}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if !begin.IsZero() {
		params.Set("begin", strconv.FormatInt(begin.UnixMilli(), 10))
	}
	if !end.IsZero() {
		params.Set("end", strconv.FormatInt(end.UnixMilli(), 10))
	}
	return params, nil
}
