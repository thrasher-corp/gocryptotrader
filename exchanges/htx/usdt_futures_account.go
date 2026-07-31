package htx

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// GetV5AccountBalance gets the USDT-margined unified-margin account balance.
func (e *Exchange) GetV5AccountBalance(ctx context.Context) (*V5AccountBalanceResponse, error) {
	var resp *V5AccountBalanceResponse
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/v5/account/balance", nil, nil, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetV5AssetMode gets the account asset mode.
func (e *Exchange) GetV5AssetMode(ctx context.Context) (*V5AssetModeResponse, error) {
	var resp *V5AssetModeResponse
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/v5/account/asset_mode", nil, nil, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// SetV5AssetMode sets the account asset mode.
func (e *Exchange) SetV5AssetMode(ctx context.Context, assetMode uint64) (*V5AssetModeResponse, error) {
	req := &V5SetAssetModeRequest{AssetMode: assetMode}
	var resp *V5AssetModeResponse
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, "/v5/account/asset_mode", nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetV5AccountBills gets account transaction records.
func (e *Exchange) GetV5AccountBills(ctx context.Context, req *V5AccountBillsRequest) (*V5AccountBillsResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	params := url.Values{}
	if req.ContractCode != "" {
		params.Set("contract_code", req.ContractCode)
	}
	if req.MarginMode != "" {
		params.Set("margin_mode", req.MarginMode)
	}
	if !req.StartTime.IsZero() {
		params.Set("start_time", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if req.Type != "" {
		params.Set("type", req.Type)
	}
	if !req.EndTime.IsZero() {
		params.Set("end_time", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if req.From != 0 {
		params.Set("from", strconv.FormatUint(req.From, 10))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Direction != "" {
		params.Set("direct", req.Direction)
	}
	var resp *V5AccountBillsResponse
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/v5/account/bills", params, nil, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetV5FeeDeductionCurrency gets the configured fee-deduction currency.
func (e *Exchange) GetV5FeeDeductionCurrency(ctx context.Context) (*V5FeeDeductionCurrencyResponse, error) {
	var resp *V5FeeDeductionCurrencyResponse
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/v5/account/fee_deduction_currency", nil, nil, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// SetV5FeeDeductionCurrency configures the fee-deduction currency.
func (e *Exchange) SetV5FeeDeductionCurrency(ctx context.Context, req *V5SetFeeDeductionCurrencyRequest) (*V5FeeDeductionCurrencyResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	var resp *V5FeeDeductionCurrencyResponse
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, "/v5/account/fee_deduction_currency", nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// V5UniversalTransfer transfers funds between supported HTX account types.
func (e *Exchange) V5UniversalTransfer(ctx context.Context, req *V5UniversalTransferRequest) (*V5UniversalTransferResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	var resp *V5UniversalTransferResponse
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, "/v5/account/universal_transfer", nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetV5UniversalTransferRecords gets universal-transfer records.
func (e *Exchange) GetV5UniversalTransferRecords(ctx context.Context, req *V5UniversalTransferRecordsRequest) (*V5UniversalTransferRecordsResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	params := url.Values{}
	if req.TransferID != 0 {
		params.Set("transfer_id", strconv.FormatUint(req.TransferID, 10))
	}
	if req.Currency != "" {
		params.Set("currency", req.Currency)
	}
	if req.Status != "" {
		params.Set("status", req.Status)
	}
	if !req.StartTime.IsZero() {
		params.Set("start_time", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("end_time", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if req.From != 0 {
		params.Set("from", strconv.FormatUint(req.From, 10))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Direction != "" {
		params.Set("direct", req.Direction)
	}
	var resp *V5UniversalTransferRecordsResponse
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/v5/account/universal_transfer_records", params, nil, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}
