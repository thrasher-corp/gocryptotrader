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

var errTransferTypeRequired = errors.New("transfer type is required")

// GetCreditLineAccountInformation returns the credit line account's terms, loans and balances
func (e *Exchange) GetCreditLineAccountInformation(ctx context.Context) (*CreditLineAccountInformationResponse, error) {
	var resp *CreditLineAccountInformationResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v2/cl/account", nil, creditLineAccountRate, &resp)
}

// GetCreditLineAlertHistory returns the credit line's margin call and liquidation alerts. A nil request returns the
// latest alerts of either type
func (e *Exchange) GetCreditLineAlertHistory(ctx context.Context, req *CreditLineAlertHistoryRequest) ([]CreditLineAlertRecordResponse, error) {
	params := url.Values{}
	if req != nil {
		if err := setTimeRangeParams(params, req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
		if req.Limit > 0 {
			params.Set("limit", strconv.FormatUint(req.Limit, 10))
		}
		if req.AlertType != "" {
			params.Set("alertType", string(req.AlertType))
		}
	}
	var resp []CreditLineAlertRecordResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v2/cl/alertHistory", params, defaultRate, &resp)
}

// GetCreditLineTransferHistory returns the transfers into and out of the credit line account. A nil request returns the
// latest transfers of every asset
func (e *Exchange) GetCreditLineTransferHistory(ctx context.Context, req *CreditLineTransferHistoryRequest) ([]CreditLineTransferRecordResponse, error) {
	params := url.Values{}
	if req != nil {
		if err := setTimeRangeParams(params, req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
		if req.Limit > 0 {
			params.Set("limit", strconv.FormatUint(req.Limit, 10))
		}
		if req.TransferType != "" {
			params.Set("transferType", string(req.TransferType))
		}
		if !req.Asset.IsEmpty() {
			params.Set("asset", req.Asset.Upper().String())
		}
	}
	var resp []CreditLineTransferRecordResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/cl/transferHistory", params, defaultRate, &resp)
}

// ExecuteCreditLineTransfer transfers an asset into or out of the credit line account
func (e *Exchange) ExecuteCreditLineTransfer(ctx context.Context, req *CreditLineTransferRequest) (*CreditLineTransferResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.TransferType == "" {
		return nil, errTransferTypeRequired
	}
	if req.TransferAssetType.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.Quantity <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	params := url.Values{}
	params.Set("transferType", string(req.TransferType))
	params.Set("transferAssetType", req.TransferAssetType.Upper().String())
	params.Set("quantity", formatFloat(req.Quantity))
	var resp *CreditLineTransferResponse
	return resp, e.SendAuthHTTPRequest(request.WithRetryNotAllowed(ctx), exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/cl/transfer", params, defaultRate, &resp)
}
