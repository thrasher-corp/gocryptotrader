package binanceus

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

var (
	errEmailRequired          = errors.New("email required")
	errSenderEmailRequired    = errors.New("sender email required")
	errRecipientEmailRequired = errors.New("recipient email required")
)

// GetSubAccountInformation returns the master account's sub-accounts. A nil request returns the first page of every
// sub-account
func (e *Exchange) GetSubAccountInformation(ctx context.Context, req *SubAccountInformationRequest) (*SubAccountInformationResponse, error) {
	params := url.Values{}
	if req != nil {
		if req.Email != "" {
			params.Set("email", req.Email)
		}
		if req.Status != "" {
			params.Set("status", req.Status)
		}
		if req.Page > 0 {
			params.Set("page", strconv.FormatUint(req.Page, 10))
		}
		if req.Limit > 0 {
			params.Set("limit", strconv.FormatUint(req.Limit, 10))
		}
	}
	var resp *SubAccountInformationResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v3/sub-account/list", params, defaultRate, &resp)
}

// GetSubAccountTransferHistory returns the transfers between the master account and its sub-accounts. A nil request
// returns the first page of every transfer
func (e *Exchange) GetSubAccountTransferHistory(ctx context.Context, req *SubAccountTransferHistoryRequest) (*SubAccountTransferHistoryResponse, error) {
	params := url.Values{}
	if req != nil {
		if req.Email != "" {
			params.Set("email", req.Email)
		}
		if err := setTimeRangeParams(params, req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
		if req.Page > 0 {
			params.Set("page", strconv.FormatUint(req.Page, 10))
		}
		if req.Limit > 0 {
			params.Set("limit", strconv.FormatUint(req.Limit, 10))
		}
	}
	var resp *SubAccountTransferHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v3/sub-account/transfer/history", params, defaultRate, &resp)
}

// ExecuteSubAccountTransfer transfers an asset between the master account and a sub-account
func (e *Exchange) ExecuteSubAccountTransfer(ctx context.Context, req *SubAccountTransferRequest) (*SubAccountTransferResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.FromEmail == "" {
		return nil, errSenderEmailRequired
	}
	if req.ToEmail == "" {
		return nil, errRecipientEmailRequired
	}
	if req.Asset.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.Amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	params := url.Values{}
	params.Set("fromEmail", req.FromEmail)
	params.Set("toEmail", req.ToEmail)
	params.Set("asset", req.Asset.Upper().String())
	params.Set("amount", formatFloat(req.Amount))
	var resp *SubAccountTransferResponse
	return resp, e.SendAuthHTTPRequest(request.WithRetryNotAllowed(ctx), exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v3/sub-account/transfer", params, defaultRate, &resp)
}

// GetSubAccountAssets returns a sub-account's balances
func (e *Exchange) GetSubAccountAssets(ctx context.Context, email string) (*SubAccountAssetsResponse, error) {
	if email == "" {
		return nil, errEmailRequired
	}
	params := url.Values{}
	params.Set("email", email)
	var resp *SubAccountAssetsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v3/sub-account/assets", params, defaultRate, &resp)
}

// GetMasterAccountTotalUSDValue returns the master account's total asset value and each sub-account's. A nil request
// returns the first page of every sub-account
func (e *Exchange) GetMasterAccountTotalUSDValue(ctx context.Context, req *MasterAccountTotalUSDValueRequest) (*MasterAccountTotalUSDValueResponse, error) {
	params := url.Values{}
	if req != nil {
		if req.Email != "" {
			params.Set("email", req.Email)
		}
		if req.Page > 0 {
			params.Set("page", strconv.FormatUint(req.Page, 10))
		}
		if req.Size > 0 {
			params.Set("size", strconv.FormatUint(req.Size, 10))
		}
	}
	var resp *MasterAccountTotalUSDValueResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/sub-account/spotSummary", params, defaultRate, &resp)
}

// GetSubAccountStatusList returns the status of the sub-accounts with an email
func (e *Exchange) GetSubAccountStatusList(ctx context.Context, email string) ([]SubAccountStatusResponse, error) {
	if email == "" {
		return nil, errEmailRequired
	}
	params := url.Values{}
	params.Set("email", email)
	var resp []SubAccountStatusResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/sub-account/status", params, defaultRate, &resp)
}
