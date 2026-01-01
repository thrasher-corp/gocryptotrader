package mexc

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// GetBrokerUniversalTransferHistory retrieves universal transfer history for broker users
func (e *Exchange) GetBrokerUniversalTransferHistory(ctx context.Context, fromAccountType, toAccountType asset.Item, fromAccount, toAccount string, startTime, endTime time.Time, page, limit int64) ([]BrokerAssetTransfer, error) {
	if !fromAccountType.IsValid() {
		return nil, fmt.Errorf("%w: FronAccountType is required", errAddressRequired)
	}
	if !toAccountType.IsValid() {
		return nil, fmt.Errorf("%w: ToAccountType is required", errAddressRequired)
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("fromAccountType", fromAccountType.String())
	params.Set("toAccountType", toAccountType.String())
	if fromAccount != "" {
		params.Set("fromAccount", fromAccount)
	}
	if toAccount != "" {
		params.Set("toAccount", toAccount)
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []BrokerAssetTransfer
	return resp, e.SendHTTPRequest(ctx, exchange.RestFutures, request.Auth, http.MethodGet, "broker/sub-account/universalTransfer", params, &resp, true)
}

// CreateBrokerSubAccount holds a broker sub-account detail
func (e *Exchange) CreateBrokerSubAccount(ctx context.Context) (*BrokerSubAccounts, error) {
	var resp *BrokerSubAccounts
	return resp, e.SendHTTPRequest(ctx, exchange.RestFutures, request.Auth, http.MethodPost, "broker/sub-account/virtualSubAccount", nil, &resp, true)
}

// GetBrokerAccountSubAccountList represents a list of broker sub-accounts and their details of the broker account
func (e *Exchange) GetBrokerAccountSubAccountList(ctx context.Context, subAccount string, page, limit int64) (*BrokerSubAccounts, error) {
	params := url.Values{}
	if subAccount != "" {
		params.Set("subAccount", subAccount)
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *BrokerSubAccounts
	return resp, e.SendHTTPRequest(ctx, exchange.RestFutures, request.Auth, http.MethodGet, "broker/sub-account/list", params, &resp, true)
}

// GetSubAccountStatus retrieves broker sub-account status information
func (e *Exchange) GetSubAccountStatus(ctx context.Context, subAccount string) (*BrokerSubAccountStatus, error) {
	if subAccount == "" {
		return nil, errInvalidSubAccountName
	}
	params := url.Values{}
	params.Set("subAccount", subAccount)
	var resp *BrokerSubAccountStatus
	return resp, e.SendHTTPRequest(ctx, exchange.RestFutures, request.Auth, http.MethodGet, "broker/sub-account/status", params, &resp, true)
}

// CreateBrokerSubAccountAPIKey creates a new sub-account api-key for the broker account
func (e *Exchange) CreateBrokerSubAccountAPIKey(ctx context.Context, arg *BrokerSubAccountAPIKeyParams) (*BrokerSubAccountAPIKey, error) {
	if arg.SubAccount == "" {
		return nil, errInvalidSubAccountName
	}
	if len(arg.Permissions) == 0 {
		return nil, errUnsupportedPermissionValue
	}
	if arg.Note == "" {
		return nil, errInvalidSubAccountNote
	}
	var resp *BrokerSubAccountAPIKey
	return resp, e.SendHTTPRequest(ctx, exchange.RestFutures, request.Auth, http.MethodPost, "broker/sub-account/apiKey", nil, arg, &resp, true)
}

// GetBrokerSubAccountAPIKey holds a subaccount API Key information
func (e *Exchange) GetBrokerSubAccountAPIKey(ctx context.Context, subAccount string) (*BrokerSubAccountAPIKeys, error) {
	if subAccount == "" {
		return nil, errInvalidSubAccountName
	}
	params := url.Values{}
	params.Set("subAccount", subAccount)
	var resp *BrokerSubAccountAPIKeys
	return resp, e.SendHTTPRequest(ctx, exchange.RestFutures, request.Auth, http.MethodGet, "broker/sub-account/apiKey", params, nil, &resp, true)
}

// DeleteBrokerAPIKeySubAccount deletes broker's sub-account API key
func (e *Exchange) DeleteBrokerAPIKeySubAccount(ctx context.Context, arg *BrokerSubAccountAPIKeyDeletionParams) (any, error) {
	if arg.SubAccount == "" {
		return nil, errInvalidSubAccountName
	}
	if arg.APIKey == "" {
		return nil, errAPIKeyMissing
	}
	var resp struct {
		SubAccount string `json:"subAccount"`
	}
	return resp.SubAccount, e.SendHTTPRequest(ctx, exchange.RestFutures, request.Auth, http.MethodDelete, "broker/sub-account/apiKey", nil, &arg, &resp, true)
}

// GenerateBrokerSubAccountDepositAddress creates a new deposit address for a broker sub-account
func (e *Exchange) GenerateBrokerSubAccountDepositAddress(ctx context.Context, arg *BrokerSubAccountDepositAddressCreationParams) (*BrokerSubAccountDepositAddress, error) {
	if arg.Coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Network == "" {
		return nil, errNetworkNameRequired
	}
	var resp *BrokerSubAccountDepositAddress
	return resp, e.SendHTTPRequest(ctx, exchange.RestFutures, request.Auth, http.MethodPost, "broker/capital/deposit/subAddress", nil, &arg, &resp, true)
}

// GetBrokerSubAccountDepositAddress retrieves a broker sub-account deposit address
func (e *Exchange) GetBrokerSubAccountDepositAddress(ctx context.Context, coin currency.Code) ([]BrokerSubAccountDepositAddress, error) {
	if coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("coin", coin.String())
	var resp []BrokerSubAccountDepositAddress
	return resp, e.SendHTTPRequest(ctx, exchange.RestFutures, request.Auth, http.MethodGet, "broker/capital/deposit/subAddress", params, nil, &resp, true)
}

// GetSubAccountDepositHistory retrieves a broker sub-account deposit history
func (e *Exchange) GetSubAccountDepositHistory(ctx context.Context, coin currency.Code, depositStatus string, startTime, endTime time.Time, limit, page int64) ([]BrokerSubAccountDepositDetail, error) {
	return e.getSubAccountDepositList(ctx, coin, depositStatus, "broker/capital/deposit/subHisrec", startTime, endTime, limit, page)
}

// GetAllRecentSubAccountDepositHistory retrieves a recent (3-days) broker sub-account deposit history
func (e *Exchange) GetAllRecentSubAccountDepositHistory(ctx context.Context, coin currency.Code, depositStatus string, startTime, endTime time.Time, limit, page int64) ([]BrokerSubAccountDepositDetail, error) {
	return e.getSubAccountDepositList(ctx, coin, depositStatus, "broker/capital/deposit/subHisrec/getall", startTime, endTime, limit, page)
}

func (e *Exchange) getSubAccountDepositList(ctx context.Context, coin currency.Code, depositStatus, path string, startTime, endTime time.Time, limit, page int64) ([]BrokerSubAccountDepositDetail, error) {
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if !coin.IsEmpty() {
		params.Set("coin", coin.String())
	}
	if depositStatus != "" {
		params.Set("status", depositStatus)
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	var resp []BrokerSubAccountDepositDetail
	return resp, e.SendHTTPRequest(ctx, exchange.RestFutures, request.Auth, http.MethodGet, path, params, nil, &resp, true)
}
