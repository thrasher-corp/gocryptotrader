package htx

import (
	"context"
	"net/url"
	"strconv"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// GetV5OpenInterest gets the current USDT-margined contract open interest.
func (e *Exchange) GetV5OpenInterest(ctx context.Context, code currency.Pair) (*V5OpenInterestResponse, error) {
	codeValue, err := e.FormatSymbol(code, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	var resp *V5OpenInterestResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/v5/market/open_interest", params), &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetV5AssetsDeductionCurrencies gets currencies supported for fee deduction.
func (e *Exchange) GetV5AssetsDeductionCurrencies(ctx context.Context) (*V5AssetsDeductionCurrenciesResponse, error) {
	var resp *V5AssetsDeductionCurrenciesResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, "/v5/market/assets_deduction_currency", &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetV5MultiAssetsMarginCurrencies gets currencies supported by multi-assets margin.
func (e *Exchange) GetV5MultiAssetsMarginCurrencies(ctx context.Context) (*V5MultiAssetsMarginCurrenciesResponse, error) {
	var resp *V5MultiAssetsMarginCurrenciesResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, "/v5/market/multi_assets_margin", &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetV5EliteAccountRatio gets the elite-account long/short ratio.
func (e *Exchange) GetV5EliteAccountRatio(ctx context.Context, code currency.Pair, period string) (*V5EliteRatioResponse, error) {
	codeValue, err := e.FormatSymbol(code, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	params.Set("period", period)
	var resp *V5EliteRatioResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/v5/market/elite_account_ratio", params), &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetV5ElitePositionRatio gets the elite-position long/short ratio.
func (e *Exchange) GetV5ElitePositionRatio(ctx context.Context, code currency.Pair, period string) (*V5EliteRatioResponse, error) {
	codeValue, err := e.FormatSymbol(code, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	params.Set("period", period)
	var resp *V5EliteRatioResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/v5/market/elite_position_ratio", params), &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetV5EstimatedSettlementPrice gets estimated settlement prices.
func (e *Exchange) GetV5EstimatedSettlementPrice(ctx context.Context, code currency.Pair) (*V5EstimatedSettlementPriceResponse, error) {
	params := url.Values{}
	if !code.IsEmpty() {
		codeValue, err := e.FormatSymbol(code, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("contract_code", codeValue)
	}
	var resp *V5EstimatedSettlementPriceResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/v5/market/estimated_settlement_price", params), &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetV5FundingRates gets current funding rates for up to 10 contracts.
func (e *Exchange) GetV5FundingRates(ctx context.Context, req *V5FundingRatesRequest) (*V5FundingRatesResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	if len(req.ContractCodes) == 0 || len(req.ContractCodes) > 10 {
		return nil, errContractCodeLimitExceeded
	}
	codes := make([]string, len(req.ContractCodes))
	for i := range req.ContractCodes {
		code, err := e.FormatSymbol(req.ContractCodes[i], asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		codes[i] = code
	}
	params := url.Values{}
	params.Set("contract_code", strings.Join(codes, ","))
	var resp *V5FundingRatesResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/v5/market/funding_rate", params), &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetV5FundingRateHistory gets historical funding rates for one contract.
func (e *Exchange) GetV5FundingRateHistory(ctx context.Context, req *V5FundingRateHistoryRequest) (*V5FundingRateHistoryResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	params := url.Values{}
	params.Set("contract_code", req.ContractCode)
	if !req.StartTime.IsZero() {
		params.Set("start_time", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("end_time", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if req.From != "" {
		params.Set("from", req.From)
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Direction != "" {
		params.Set("direct", req.Direction)
	}
	var resp *V5FundingRateHistoryResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/v5/market/funding_rate_history", params), &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetV5PriceLimits gets the current price limits for one or all contracts.
func (e *Exchange) GetV5PriceLimits(ctx context.Context, code currency.Pair) (*V5PriceLimitsResponse, error) {
	params := url.Values{}
	if !code.IsEmpty() {
		codeValue, err := e.FormatSymbol(code, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("contract_code", codeValue)
	}
	var resp *V5PriceLimitsResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/v5/market/price_limit", params), &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetV5LiquidationOrders gets public liquidation orders.
func (e *Exchange) GetV5LiquidationOrders(ctx context.Context, req *V5LiquidationOrdersRequest) (*V5LiquidationOrdersResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	params := url.Values{}
	if req.ContractCode != "" {
		params.Set("contract_code", req.ContractCode)
	}
	if req.Pair != "" {
		params.Set("pair", req.Pair)
	}
	if !req.StartTime.IsZero() {
		params.Set("start_time", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("end_time", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if req.Direction != "" {
		params.Set("direct", req.Direction)
	}
	if req.From != "" {
		params.Set("from", req.From)
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp *V5LiquidationOrdersResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/v5/market/liquidation_orders", params), &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetV5MarketRiskLimit gets contract risk limits.
func (e *Exchange) GetV5MarketRiskLimit(ctx context.Context, code currency.Pair, marginMode, tier string) (*V5RiskLimitResponse, error) {
	codeValue, err := e.FormatSymbol(code, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	if marginMode != "" {
		params.Set("margin_mode", marginMode)
	}
	if tier != "" {
		params.Set("tier", tier)
	}
	var resp *V5RiskLimitResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/v5/market/risk/limit", params), &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetV5SettlementHistory gets public settlement history.
func (e *Exchange) GetV5SettlementHistory(ctx context.Context, req *V5SettlementHistoryRequest) (*V5SettlementHistoryResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	params := url.Values{}
	params.Set("contract_code", req.ContractCode)
	if !req.StartTime.IsZero() {
		params.Set("start_time", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("end_time", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if req.Direction != "" {
		params.Set("direct", req.Direction)
	}
	if req.From != "" {
		params.Set("from", req.From)
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp *V5SettlementHistoryResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/v5/market/settlement_history", params), &resp); err != nil {
		return nil, err
	}
	return resp, nil
}
