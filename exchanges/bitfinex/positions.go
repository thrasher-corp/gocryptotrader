package bitfinex

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

var (
	errPositionIDRequired         = errors.New("position ID required")
	errInvalidPositionFundingType = errors.New("invalid position funding type, want 0 for daily or 1 for term")
	errInvalidPositionCollateral  = errors.New("invalid collateral, want a positive amount")
)

// GetMarginInfoBase returns the account's margin profit and loss, swaps and balances. Margin info covers margin
// trading only, not derivatives
func (e *Exchange) GetMarginInfoBase(ctx context.Context) (*MarginInfoBaseResponse, error) {
	var resp *MarginInfoBaseResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, marginInfoEPL, "auth/r/info/margin/base", nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetMarginInfoBySymbol returns the account's buying power on a margin trading pair, such as tBTCUSD
func (e *Exchange) GetMarginInfoBySymbol(ctx context.Context, symbol string) (*MarginInfoSymbolResponse, error) {
	if !isTradingSymbol(symbol) {
		return nil, errTradingSymbolRequired
	}
	var resp *MarginInfoSymbolResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, marginInfoEPL, "auth/r/info/margin/"+symbol, nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetMarginInfoAllSymbols returns the account's buying power on every margin trading pair
func (e *Exchange) GetMarginInfoAllSymbols(ctx context.Context) ([]MarginInfoSymbolResponse, error) {
	var resp []MarginInfoSymbolResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, marginInfoEPL, "auth/r/info/margin/sym_all", nil, &resp)
}

// GetActivePositions returns the account's active margin and derivative positions
func (e *Exchange) GetActivePositions(ctx context.Context) ([]Position, error) {
	var resp []Position
	return resp, e.SendAuthenticatedHTTPRequest(ctx, positionsEPL, "auth/r/positions", nil, &resp)
}

// ClaimPosition settles a margin position, or part of it, as an exchange buy or sale with the margin wallet's funds,
// returning the position. The position's realised profit and the margin wallet's funds, net of financing charges,
// must cover the financing the claimed part owes
func (e *Exchange) ClaimPosition(ctx context.Context, req *ClaimPositionRequest) (*Notification[Position], error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.PositionID == 0 {
		return nil, errPositionIDRequired
	}
	body := &claimPositionBody{ID: req.PositionID}
	if req.Amount != 0 {
		body.Amount = strconv.FormatFloat(req.Amount, 'f', -1, 64)
	}
	return sendNotificationRequest[Position](ctx, e, claimPositionEPL, "auth/w/position/claim", body)
}

// IncreasePosition opens or increases a margin position with the margin wallet's funds, the reverse of claiming one,
// returning the position's symbol, amount and base price. It needs a verified or paper trading account, and full
// verification for a long position on a pair quoted in fiat
func (e *Exchange) IncreasePosition(ctx context.Context, req *IncreasePositionRequest) (*Notification[Position], error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !isTradingSymbol(req.Symbol) {
		return nil, errTradingSymbolRequired
	}
	if req.Amount == 0 {
		return nil, errInvalidAmount
	}
	return sendNotificationRequest[Position](ctx, e, increasePositionEPL, "auth/w/position/increase", &increasePositionBody{
		Symbol: req.Symbol,
		Amount: strconv.FormatFloat(req.Amount, 'f', -1, 64),
	})
}

// GetIncreasePositionInfo returns what increasing a position on a trading pair can do: the largest position it can
// reach, the balances it can use and the funding a position of the requested amount needs
func (e *Exchange) GetIncreasePositionInfo(ctx context.Context, req *IncreasePositionInfoRequest) (*IncreasePositionInfoResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !isTradingSymbol(req.Symbol) {
		return nil, errTradingSymbolRequired
	}
	body := &increasePositionInfoBody{Symbol: req.Symbol}
	if req.Amount != 0 {
		body.Amount = strconv.FormatFloat(req.Amount, 'f', -1, 64)
	}
	var resp *IncreasePositionInfoResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, increasePositionInfoEPL, "auth/r/position/increase/info", body, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetPositionsHistory returns the account's past positions, up to 50 a request
func (e *Exchange) GetPositionsHistory(ctx context.Context, req *PositionsHistoryRequest) ([]Position, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	history, err := newHistoryBody(req.Start, req.End, req.Limit, 0)
	if err != nil {
		return nil, err
	}
	var resp []Position
	return resp, e.SendAuthenticatedHTTPRequest(ctx, positionsHistoryEPL, "auth/r/positions/hist", &positionsHistoryBody{historyBody: history, ID: req.PositionID}, &resp)
}

// GetPositionsSnapshot returns the daily snapshots of the account's positions, up to 500 a request
func (e *Exchange) GetPositionsSnapshot(ctx context.Context, req *PositionsSnapshotRequest) ([]Position, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	history, err := newHistoryBody(req.Start, req.End, req.Limit, 0)
	if err != nil {
		return nil, err
	}
	var resp []Position
	return resp, e.SendAuthenticatedHTTPRequest(ctx, positionsSnapshotEPL, "auth/r/positions/snap", &positionsHistoryBody{historyBody: history, ID: req.PositionID}, &resp)
}

// GetPositionsAudit returns the states the account's latest positions, or the positions requested, went through, up
// to 250 a request, each with the reason for the change, such as the trade that made it, in its Meta
func (e *Exchange) GetPositionsAudit(ctx context.Context, req *PositionsAuditRequest) ([]Position, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	history, err := newHistoryBody(req.Start, req.End, req.Limit, 0)
	if err != nil {
		return nil, err
	}
	var resp []Position
	return resp, e.SendAuthenticatedHTTPRequest(ctx, positionsAuditEPL, "auth/r/positions/audit", &positionsAuditBody{historyBody: history, IDs: req.PositionIDs}, &resp)
}

// UpdatePositionFundingType sets whether the position on a trading pair takes daily or term funding
func (e *Exchange) UpdatePositionFundingType(ctx context.Context, req *UpdatePositionFundingTypeRequest) (*Notification[json.RawMessage], error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !isTradingSymbol(req.Symbol) {
		return nil, errTradingSymbolRequired
	}
	if req.FundingType != PositionFundingTypeDaily && req.FundingType != PositionFundingTypeTerm {
		return nil, fmt.Errorf("%w: %d", errInvalidPositionFundingType, req.FundingType)
	}
	return sendNotificationRequest[json.RawMessage](ctx, e, updatePositionFundingTypeEPL, "auth/w/position/update/funding/type", &updatePositionFundingTypeBody{
		Symbol: req.Symbol,
		Type:   req.FundingType,
	})
}

// SetDerivativePositionCollateral sets the collateral of the active position on a derivative pair, such as
// tBTCF0:USTF0. Bitfinex refuses collateral outside the limits GetDerivativePositionCollateralLimits returns, with
// error 10020 "collateral: insufficient", or 110202 "collateral: market risk" for a reduction that leaves the position
// too close to liquidation
func (e *Exchange) SetDerivativePositionCollateral(ctx context.Context, req *DerivativePositionCollateralRequest) (*DerivativePositionCollateralResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !isTradingSymbol(req.Symbol) {
		return nil, errTradingSymbolRequired
	}
	if req.Collateral <= 0 {
		return nil, fmt.Errorf("%w: %v", errInvalidPositionCollateral, req.Collateral)
	}
	body := &derivativePositionCollateralBody{
		Symbol:     req.Symbol,
		Collateral: json.RawMessage(strconv.FormatFloat(req.Collateral, 'f', -1, 64)),
	}
	var resp []DerivativePositionCollateralResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, derivativePositionCollateralEPL, "auth/w/deriv/collateral/set", body, &resp); err != nil {
		return nil, err
	}
	if len(resp) == 0 {
		return nil, common.ErrNoResponse
	}
	return &resp[0], nil
}

// GetDerivativePositionCollateralLimits returns the least and most collateral the active position on a derivative
// pair, such as tBTCF0:USTF0, can be given; Bitfinex errors when the pair has no active position
func (e *Exchange) GetDerivativePositionCollateralLimits(ctx context.Context, symbol string) (*DerivativePositionCollateralLimitsResponse, error) {
	if !isTradingSymbol(symbol) {
		return nil, errTradingSymbolRequired
	}
	var resp *DerivativePositionCollateralLimitsResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, derivativePositionCollateralLimitsEPL, "auth/calc/deriv/collateral/limits", &derivativePositionCollateralLimitsBody{Symbol: symbol}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}
