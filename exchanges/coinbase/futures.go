package coinbase

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"

	"github.com/thrasher-corp/gocryptotrader/common"
)

var (
	errFuturesSweepAmount           = errors.New("futures sweep amount must not be negative")
	errFuturesSweepNotScheduled     = errors.New("futures sweep not scheduled")
	errFuturesSweepNotCancelled     = errors.New("futures sweep not cancelled")
	errInvalidMarginProfileType     = errors.New("invalid margin profile type, want MARGIN_PROFILE_TYPE_RETAIL_REGULAR or MARGIN_PROFILE_TYPE_RETAIL_INTRADAY_MARGIN_1")
	errInvalidIntradayMarginSetting = errors.New("invalid intraday margin setting, want INTRADAY_MARGIN_SETTING_STANDARD or INTRADAY_MARGIN_SETTING_INTRADAY")
)

// CancelPendingFuturesSweep cancels the pending sweep of funds from the US derivatives futures account to the USD
// spot wallet; a sweep that has started processing can no longer be cancelled
func (e *Exchange) CancelPendingFuturesSweep(ctx context.Context) error {
	var resp *FuturesSweepResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, advancedTradePrivateEPL, http.MethodDelete, advancedTradePath+"cfm/sweeps", nil, nil, &resp); err != nil {
		return err
	}
	if resp == nil {
		return common.ErrNoResponse
	}
	if !resp.Success {
		return errFuturesSweepNotCancelled
	}
	return nil
}

// GetCurrentMarginWindow returns the US derivatives margin window in force for a margin profile type, one of the
// MarginProfileType values or empty, and whether Coinbase has turned intraday margin off
func (e *Exchange) GetCurrentMarginWindow(ctx context.Context, marginProfileType string) (*CurrentMarginWindowResponse, error) {
	params := url.Values{}
	switch marginProfileType {
	case "":
	case MarginProfileTypeRetailRegular, MarginProfileTypeRetailIntradayMargin1:
		params.Set("margin_profile_type", marginProfileType)
	default:
		return nil, fmt.Errorf("%w: %q", errInvalidMarginProfileType, marginProfileType)
	}
	var resp *CurrentMarginWindowResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, advancedTradePrivateEPL, http.MethodGet, advancedTradePath+"cfm/intraday/current_margin_window", params, nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetFuturesBalanceSummary returns the US derivatives account's balances and margin, with its margin health in the
// intraday and overnight windows once it is opted in to intraday margin
func (e *Exchange) GetFuturesBalanceSummary(ctx context.Context) (*FuturesBalanceSummary, error) {
	var resp *FuturesBalanceSummaryResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, advancedTradePrivateEPL, http.MethodGet, advancedTradePath+"cfm/balance_summary", nil, nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil || resp.BalanceSummary == nil {
		return nil, common.ErrNoResponse
	}
	return resp.BalanceSummary, nil
}

// GetFuturesPosition returns the account's position in a US derivatives product, such as BIT-30OCT26-CDE
func (e *Exchange) GetFuturesPosition(ctx context.Context, productID string) (*FuturesPosition, error) {
	if productID == "" {
		return nil, errProductIDRequired
	}
	var resp *FuturesPositionResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, advancedTradePrivateEPL, http.MethodGet, advancedTradePath+"cfm/positions/"+productID, nil, nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil || resp.Position == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Position, nil
}

// GetIntradayMarginSetting returns whether the account takes more leverage in the intraday window, one of the
// IntradayMarginSetting values
func (e *Exchange) GetIntradayMarginSetting(ctx context.Context) (string, error) {
	var resp *IntradayMarginSettingResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, advancedTradePrivateEPL, http.MethodGet, advancedTradePath+"cfm/intraday/margin_setting", nil, nil, &resp); err != nil {
		return "", err
	}
	if resp == nil {
		return "", common.ErrNoResponse
	}
	return resp.Setting, nil
}

// ListFuturesPositions returns the account's US derivatives positions, which are only held in its default portfolio
func (e *Exchange) ListFuturesPositions(ctx context.Context) ([]FuturesPosition, error) {
	var resp *FuturesPositionsResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, advancedTradePrivateEPL, http.MethodGet, advancedTradePath+"cfm/positions", nil, nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Positions, nil
}

// ListFuturesSweeps returns the pending and processing sweeps of funds from the US derivatives futures account to the
// USD spot wallet
func (e *Exchange) ListFuturesSweeps(ctx context.Context) ([]FuturesSweep, error) {
	var resp *FuturesSweepsResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, advancedTradePrivateEPL, http.MethodGet, advancedTradePath+"cfm/sweeps", nil, nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Sweeps, nil
}

// ScheduleFuturesSweep schedules a sweep of USD that no futures position needs as margin from the futures account at
// Coinbase Financial Markets to the USD spot wallet at Coinbase Inc., where it can trade spot or be withdrawn; cash
// only moves the other way automatically. A zero amount sweeps every available excess fund, and Coinbase holds one
// pending sweep at a time
func (e *Exchange) ScheduleFuturesSweep(ctx context.Context, usdAmount float64) error {
	// An amount neither positive nor zero, NaN among them, would otherwise sweep every excess fund
	if usdAmount < 0 || math.IsNaN(usdAmount) || math.IsInf(usdAmount, 0) {
		return fmt.Errorf("%w: %v", errFuturesSweepAmount, usdAmount)
	}
	body := &ScheduleFuturesSweepWireRequest{}
	if usdAmount > 0 {
		body.USDAmount = formatFloat(usdAmount)
	}
	var resp *FuturesSweepResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, advancedTradePrivateEPL, http.MethodPost, advancedTradePath+"cfm/sweeps/schedule", nil, body, &resp); err != nil {
		return err
	}
	if resp == nil {
		return common.ErrNoResponse
	}
	if !resp.Success {
		return errFuturesSweepNotScheduled
	}
	return nil
}

// SetIntradayMarginSetting opts the account in to more leverage in the intraday window, 8am to 4pm ET on weekdays
// that are not market holidays, with IntradayMarginSettingIntraday, or out of it with IntradayMarginSettingStandard
func (e *Exchange) SetIntradayMarginSetting(ctx context.Context, setting string) error {
	if setting != IntradayMarginSettingStandard && setting != IntradayMarginSettingIntraday {
		return fmt.Errorf("%w: %q", errInvalidIntradayMarginSetting, setting)
	}
	return e.SendAuthenticatedHTTPRequest(ctx, advancedTradePrivateEPL, http.MethodPost, advancedTradePath+"cfm/intraday/margin_setting", nil, &IntradayMarginSettingWireRequest{Setting: setting}, nil)
}
