package htx

import (
	"context"
	"net/http"
	"net/url"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// GetV5Leverage gets available leverage levels.
func (e *Exchange) GetV5Leverage(ctx context.Context, code currency.Pair, marginMode, positionSide string) (*V5LeverageResponse, error) {
	params := url.Values{}
	if !code.IsEmpty() {
		codeValue, err := e.FormatSymbol(code, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("contract_code", codeValue)
	}
	if marginMode != "" {
		params.Set("margin_mode", marginMode)
	}
	if positionSide != "" {
		params.Set("position_side", positionSide)
	}
	var resp *V5LeverageResponse
	err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/v5/position/lever", params, nil, &resp)
	return resp, err
}

// SetV5Leverage sets the leverage for a position.
func (e *Exchange) SetV5Leverage(ctx context.Context, req *V5SetLeverageRequest) (*V5SetLeverageResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	var resp *V5SetLeverageResponse
	err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, "/v5/position/lever", nil, req, &resp)
	return resp, err
}

// AdjustV5PositionMargin adds or removes isolated-position margin.
func (e *Exchange) AdjustV5PositionMargin(ctx context.Context, req *V5AdjustPositionMarginRequest) (*V5Response, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	var resp *V5Response
	err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, "/v5/position/margin", nil, req, &resp)
	return resp, err
}

// GetV5PositionMode gets the account position mode.
func (e *Exchange) GetV5PositionMode(ctx context.Context) (*V5PositionModeResponse, error) {
	var resp *V5PositionModeResponse
	err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/v5/position/mode", nil, nil, &resp)
	return resp, err
}

// SetV5PositionMode sets the account position mode.
func (e *Exchange) SetV5PositionMode(ctx context.Context, positionMode string) (*V5PositionModeResponse, error) {
	if !e.positionModeInUse.CompareAndSwap(false, true) {
		return nil, errPositionModeInUse
	}
	defer e.positionModeInUse.Store(false)
	req := &V5SetPositionModeRequest{PositionMode: positionMode}
	var resp *V5PositionModeResponse
	err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, "/v5/position/mode", nil, req, &resp)
	return resp, err
}

// GetV5PositionRiskLimit gets current position risk limits.
func (e *Exchange) GetV5PositionRiskLimit(ctx context.Context, code currency.Pair, marginMode, positionSide string) (*V5RiskLimitResponse, error) {
	params := url.Values{}
	if !code.IsEmpty() {
		codeValue, err := e.FormatSymbol(code, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("contract_code", codeValue)
	}
	if marginMode != "" {
		params.Set("margin_mode", marginMode)
	}
	if positionSide != "" {
		params.Set("position_side", positionSide)
	}
	var resp *V5RiskLimitResponse
	err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/v5/position/risk/limit", params, nil, &resp)
	return resp, err
}

// GetV5PositionRiskLimitTiers gets all risk-limit tiers for a contract.
func (e *Exchange) GetV5PositionRiskLimitTiers(ctx context.Context, code currency.Pair, marginMode string) (*V5RiskLimitResponse, error) {
	codeValue, err := e.FormatSymbol(code, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	if marginMode != "" {
		params.Set("margin_mode", marginMode)
	}
	var resp *V5RiskLimitResponse
	err = e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/v5/position/risk/limit_tier", params, nil, &resp)
	return resp, err
}

// SwitchLinearSwapLeverage changes leverage through the current V5 endpoint.
func (e *Exchange) SwitchLinearSwapLeverage(ctx context.Context, code currency.Pair, leverage uint64, crossMargin bool, side order.Side) error {
	codeValue, err := e.FormatSymbol(code, asset.USDTMarginedFutures)
	if err != nil {
		return err
	}
	marginMode := marginModeIsolated
	if crossMargin {
		marginMode = marginModeCross
	}
	var positionSide string
	switch {
	case side == order.UnknownSide:
	case side.IsLong():
		positionSide = "long"
	case side.IsShort():
		positionSide = "short"
	default:
		return order.ErrSideIsInvalid
	}
	positionMode, err := e.getV5PositionModeName(ctx)
	if err != nil {
		return err
	}
	if positionMode == "single_side" {
		positionSide = "both"
	} else if positionSide == "" && !crossMargin {
		return order.ErrSideIsInvalid
	}
	_, err = e.SetV5Leverage(ctx, &V5SetLeverageRequest{
		ContractCode: codeValue,
		MarginMode:   marginMode,
		PositionSide: positionSide,
		LeverageRate: types.Number(leverage),
	})
	return err
}
