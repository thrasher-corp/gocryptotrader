package bybit

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// Broker request validation errors
var (
	errBrokerAccountIDRequired     = errors.New("broker account ID required")
	errBrokerVoucherIDRequired     = errors.New("broker voucher ID required")
	errBrokerSpecCodeRequired      = errors.New("broker spec code required")
	errBrokerVoucherAmountRequired = errors.New("broker voucher amount required")
	errBrokerIDRequired            = errors.New("broker ID required")
	errBrokerRateLimitsRequired    = errors.New("broker rate limits required")
	errBrokerUIDsRequired          = errors.New("broker rate limit UIDs required")
	errBrokerBusinessTypeRequired  = errors.New("broker rate limit business type required")
	errBrokerRateRequired          = errors.New("broker rate limit rate required")
)

// errBrokerRateLimitNotSet is returned for each rate limit Set Rate Limit reports it did not set
var errBrokerRateLimitNotSet = errors.New("broker rate limit not set")

// GetBrokerWhitelistIP returns the API broker's dedicated hostnames and the IP addresses whitelisted for each; it must
// be called from a whitelisted IP address (Get Broker Whitelist IP)
func (e *Exchange) GetBrokerWhitelistIP(ctx context.Context) ([]BrokerWhitelistIP, error) {
	var resp []BrokerWhitelistIP
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/broker/whitelist/ip", nil, nil, &resp, defaultEPL)
}

// GetBrokerIPChangeLog returns changes to the API broker's whitelisted IP addresses; it must be called from a
// whitelisted IP address (Get Broker IP Change Log)
func (e *Exchange) GetBrokerIPChangeLog(ctx context.Context, arg *GetBrokerIPChangeLogRequest) (*BrokerIPChangeLogResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if !arg.StartTime.IsZero() && !arg.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(arg.StartTime, arg.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if !arg.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(arg.StartTime.UnixMilli(), 10))
	}
	if !arg.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(arg.EndTime.UnixMilli(), 10))
	}
	if arg.Limit != 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	if arg.Cursor != "" {
		params.Set("cursor", arg.Cursor)
	}
	var resp *BrokerIPChangeLogResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/broker/ip/changelog", params, nil, &resp, defaultEPL)
}

// GetExchangeBrokerEarning returns the exchange broker master account's rebates, from up to a month ago until the day
// before (Get Earning)
func (e *Exchange) GetExchangeBrokerEarning(ctx context.Context, arg *GetExchangeBrokerEarningRequest) (*ExchangeBrokerEarningResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	switch {
	case arg.Begin.IsZero() && !arg.End.IsZero():
		return nil, fmt.Errorf("begin %w", common.ErrDateUnset)
	case !arg.Begin.IsZero() && arg.End.IsZero():
		return nil, fmt.Errorf("end %w", common.ErrDateUnset)
	case arg.Begin.After(arg.End):
		return nil, common.ErrStartAfterEnd
	}
	params := url.Values{}
	if arg.BusinessType != "" {
		params.Set("bizType", arg.BusinessType)
	}
	if !arg.Begin.IsZero() {
		params.Set("begin", arg.Begin.UTC().Format("20060102"))
		params.Set("end", arg.End.UTC().Format("20060102"))
	}
	if arg.UID != "" {
		params.Set("uid", arg.UID)
	}
	if arg.Limit != 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	if arg.Cursor != "" {
		params.Set("cursor", arg.Cursor)
	}
	var resp *ExchangeBrokerEarningResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/broker/earnings-info", params, nil, &resp, getExchangeBrokerEarningEPL)
}

// GetExchangeBrokerAccountInfo returns the exchange broker master account's subaccount quota and rebate rates (Get
// Account Info)
func (e *Exchange) GetExchangeBrokerAccountInfo(ctx context.Context) (*ExchangeBrokerAccountInfoResponse, error) {
	var resp *ExchangeBrokerAccountInfoResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/broker/account-info", nil, nil, &resp, getExchangeBrokerAccountInfoEPL)
}

// GetBrokerSubAccountDepositRecords returns deposits to the exchange broker's subaccounts, queried with the master
// account's API key (Get Sub Account Deposit Records)
func (e *Exchange) GetBrokerSubAccountDepositRecords(ctx context.Context, arg *GetBrokerSubAccountDepositRecordsRequest) (*BrokerSubAccountDepositRecordsResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if !arg.StartTime.IsZero() && !arg.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(arg.StartTime, arg.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if arg.ID != "" {
		params.Set("id", arg.ID)
	}
	if arg.TransactionID != "" {
		params.Set("txID", arg.TransactionID)
	}
	if arg.SubMemberID != "" {
		params.Set("subMemberId", arg.SubMemberID)
	}
	if !arg.Coin.IsEmpty() {
		params.Set("coin", arg.Coin.Upper().String())
	}
	if !arg.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(arg.StartTime.UnixMilli(), 10))
	}
	if !arg.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(arg.EndTime.UnixMilli(), 10))
	}
	if arg.Limit != 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	if arg.Cursor != "" {
		params.Set("cursor", arg.Cursor)
	}
	var resp *BrokerSubAccountDepositRecordsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/broker/asset/query-sub-member-deposit-record", params, nil, &resp, getBrokerSubAccountDepositRecordsEPL)
}

// SetBrokerRateLimit sets the API rate limits of the exchange broker's accounts (Set Rate Limit). Bybit reports each
// limit's outcome separately, so the response is returned with an error naming every limit it did not set
func (e *Exchange) SetBrokerRateLimit(ctx context.Context, limits []BrokerRateLimitRequest) (*SetBrokerRateLimitResponse, error) {
	if len(limits) == 0 {
		return nil, errBrokerRateLimitsRequired
	}
	list := make([]map[string]any, len(limits))
	for i := range limits {
		if len(limits[i].UIDs) == 0 {
			return nil, fmt.Errorf("rate limit %d: %w", i, errBrokerUIDsRequired)
		}
		if limits[i].BusinessType == "" {
			return nil, fmt.Errorf("rate limit %d: %w", i, errBrokerBusinessTypeRequired)
		}
		if limits[i].Rate == 0 {
			return nil, fmt.Errorf("rate limit %d: %w", i, errBrokerRateRequired)
		}
		item := make(map[string]any)
		item["uids"] = strings.Join(limits[i].UIDs, ",")
		item["bizType"] = limits[i].BusinessType
		item["rate"] = limits[i].Rate
		list[i] = item
	}
	body := make(map[string]any)
	body["list"] = list
	var resp *SetBrokerRateLimitResponse
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/broker/apilimit/set", nil, body, &resp, setBrokerRateLimitEPL); err != nil {
		return nil, err
	}
	var errs error
	for i := range resp.Result {
		if r := &resp.Result[i]; !r.Success {
			errs = common.AppendError(errs, fmt.Errorf("%w: uids %s %s: %s", errBrokerRateLimitNotSet, r.UIDs, r.BusinessType, r.Message))
		}
	}
	return resp, errs
}

// GetBrokerRateLimitCap returns the exchange broker's rate limit caps and usage per business type, for its master
// account only (Get Rate Limit Cap)
func (e *Exchange) GetBrokerRateLimitCap(ctx context.Context) (*BrokerRateLimitCapResponse, error) {
	var resp *BrokerRateLimitCapResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/broker/apilimit/query-cap", nil, nil, &resp, getBrokerRateLimitCapEPL)
}

// GetBrokerAllRateLimits returns the rate limits configured for the exchange broker's accounts (Get All Rate Limits)
func (e *Exchange) GetBrokerAllRateLimits(ctx context.Context, arg *GetBrokerAllRateLimitsRequest) (*BrokerAllRateLimitsResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params := url.Values{}
	if arg.Limit != 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	if arg.Cursor != "" {
		params.Set("cursor", arg.Cursor)
	}
	if len(arg.UIDs) != 0 {
		params.Set("uids", strings.Join(arg.UIDs, ","))
	}
	var resp *BrokerAllRateLimitsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/broker/apilimit/query-all", params, nil, &resp, getBrokerAllRateLimitsEPL)
}

// GetBrokerVoucherSpec returns a voucher's specification (Get Voucher Spec)
func (e *Exchange) GetBrokerVoucherSpec(ctx context.Context, voucherID string) (*BrokerVoucherSpecResponse, error) {
	if voucherID == "" {
		return nil, errBrokerVoucherIDRequired
	}
	body := make(map[string]any)
	body["id"] = voucherID
	var resp *BrokerVoucherSpecResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/broker/award/info", nil, body, &resp, defaultEPL)
}

// IssueBrokerVoucher issues a voucher to a user (Issue Voucher)
func (e *Exchange) IssueBrokerVoucher(ctx context.Context, arg *IssueBrokerVoucherRequest) error {
	if err := common.NilGuard(arg); err != nil {
		return err
	}
	if arg.AccountID == "" {
		return errBrokerAccountIDRequired
	}
	if arg.AwardID == "" {
		return errBrokerVoucherIDRequired
	}
	if arg.SpecCode == "" {
		return errBrokerSpecCodeRequired
	}
	if arg.Amount <= 0 {
		return errBrokerVoucherAmountRequired
	}
	if arg.BrokerID == "" {
		return errBrokerIDRequired
	}
	body := make(map[string]any)
	body["accountId"] = arg.AccountID
	body["awardId"] = arg.AwardID
	body["specCode"] = arg.SpecCode
	body["amount"] = strconv.FormatFloat(arg.Amount, 'f', -1, 64)
	body["brokerId"] = arg.BrokerID
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/broker/award/distribute-award", nil, body, nil, defaultEPL)
}

// GetBrokerIssuedVoucher returns a voucher issued to a user (Get Issued Voucher)
func (e *Exchange) GetBrokerIssuedVoucher(ctx context.Context, arg *GetBrokerIssuedVoucherRequest) (*BrokerIssuedVoucherResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.AccountID == "" {
		return nil, errBrokerAccountIDRequired
	}
	if arg.AwardID == "" {
		return nil, errBrokerVoucherIDRequired
	}
	if arg.SpecCode == "" {
		return nil, errBrokerSpecCodeRequired
	}
	body := make(map[string]any)
	body["accountId"] = arg.AccountID
	body["awardId"] = arg.AwardID
	body["specCode"] = arg.SpecCode
	if arg.WithUsedAmount {
		body["withUsedAmount"] = true
	}
	var resp *BrokerIssuedVoucherResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/broker/award/distribution-record", nil, body, &resp, defaultEPL)
}
