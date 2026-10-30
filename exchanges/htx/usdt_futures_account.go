package htx

import (
	"context"
	"math"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// GetV5AccountBalance gets the USDT-margined unified-margin account balance.
func (e *Exchange) GetV5AccountBalance(ctx context.Context) (*V5AccountBalanceResponse, error) {
	var resp *V5AccountBalanceResponse
	err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/v5/account/balance", nil, nil, &resp)
	return resp, err
}

// GetV5AssetMode gets the account asset mode.
func (e *Exchange) GetV5AssetMode(ctx context.Context) (*V5AssetModeResponse, error) {
	var resp *V5AssetModeResponse
	err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/v5/account/asset_mode", nil, nil, &resp)
	return resp, err
}

// SetV5AssetMode sets the account asset mode.
func (e *Exchange) SetV5AssetMode(ctx context.Context, assetMode uint64) (*V5SetAssetModeResponse, error) {
	req := &V5SetAssetModeRequest{AssetMode: assetMode}
	var resp *V5SetAssetModeResponse
	err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, "/v5/account/asset_mode", nil, req, &resp)
	return resp, err
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
		params.Set(orderPriceTypeLimit, strconv.FormatUint(req.Limit, 10))
	}
	if req.Direction != "" {
		params.Set("direct", req.Direction)
	}
	var resp *V5AccountBillsResponse
	err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/v5/account/bills", params, nil, &resp)
	return resp, err
}

// GetV5FeeDeductionCurrency gets the configured fee-deduction currency.
func (e *Exchange) GetV5FeeDeductionCurrency(ctx context.Context) (*V5FeeDeductionCurrencyResponse, error) {
	var resp *V5FeeDeductionCurrencyResponse
	err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/v5/account/fee_deduction_currency", nil, nil, &resp)
	return resp, err
}

// SetV5FeeDeductionCurrency configures the fee-deduction currency.
func (e *Exchange) SetV5FeeDeductionCurrency(ctx context.Context, req *V5SetFeeDeductionCurrencyRequest) (*V5FeeDeductionCurrencyResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	var resp *V5FeeDeductionCurrencyResponse
	err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, "/v5/account/fee_deduction_currency", nil, req, &resp)
	return resp, err
}

// V5UniversalTransfer transfers funds between supported HTX account types.
func (e *Exchange) V5UniversalTransfer(ctx context.Context, req *V5UniversalTransferRequest) (*V5UniversalTransferResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	if req.Amount > math.MaxInt64 {
		return nil, errInvalidTransferAmount
	}

	var resp *V5UniversalTransferResponse
	err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/account/universal_transfer", nil, req, &resp)
	return resp, err
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
	if !req.Currency.IsEmpty() {
		params.Set("currency", req.Currency.String())
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
		params.Set(orderPriceTypeLimit, strconv.FormatUint(req.Limit, 10))
	}
	if req.Direction != "" {
		params.Set("direct", req.Direction)
	}
	var resp *V5UniversalTransferRecordsResponse
	err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/account/universal_transfer_records", params, nil, &resp)
	return resp, err
}
