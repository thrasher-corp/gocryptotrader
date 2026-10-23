package coinbase

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

var (
	errPortfolioNameRequired = errors.New("portfolio name required")
	errInvalidPortfolioType  = errors.New("invalid portfolio type, want DEFAULT, CONSUMER or INTX")
)

// CreatePortfolio creates a portfolio, a separate trading environment; an account holds at most 100 portfolios
func (e *Exchange) CreatePortfolio(ctx context.Context, name string) (*Portfolio, error) {
	if name == "" {
		return nil, errPortfolioNameRequired
	}
	var resp *PortfolioResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, advancedTradePrivateEPL, http.MethodPost, advancedTradePath+"portfolios", nil, &PortfolioNameWireRequest{Name: name}, &resp); err != nil {
		return nil, err
	}
	if resp == nil || resp.Portfolio == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Portfolio, nil
}

// DeletePortfolio deletes an empty portfolio, which stays viewable, read only, on coinbase.com; portfolios can only
// be deleted through the API
func (e *Exchange) DeletePortfolio(ctx context.Context, portfolioUUID string) error {
	if portfolioUUID == "" {
		return errPortfolioIDRequired
	}
	return e.SendAuthenticatedHTTPRequest(ctx, advancedTradePrivateEPL, http.MethodDelete, advancedTradePath+"portfolios/"+portfolioUUID, nil, nil, nil)
}

// EditPortfolio renames a portfolio
func (e *Exchange) EditPortfolio(ctx context.Context, req *EditPortfolioRequest) (*Portfolio, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.PortfolioUUID == "" {
		return nil, errPortfolioIDRequired
	}
	if req.Name == "" {
		return nil, errPortfolioNameRequired
	}
	var resp *PortfolioResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, advancedTradePrivateEPL, http.MethodPut, advancedTradePath+"portfolios/"+req.PortfolioUUID, nil, &PortfolioNameWireRequest{Name: req.Name}, &resp); err != nil {
		return nil, err
	}
	if resp == nil || resp.Portfolio == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Portfolio, nil
}

// GetPortfolioBreakdown returns a portfolio's balances and its spot, perpetual, futures, prediction market and equity
// positions
func (e *Exchange) GetPortfolioBreakdown(ctx context.Context, req *PortfolioBreakdownRequest) (*PortfolioBreakdown, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.PortfolioUUID == "" {
		return nil, errPortfolioIDRequired
	}
	params := url.Values{}
	if !req.Currency.IsEmpty() {
		params.Set("currency", req.Currency.Upper().String())
	}
	var resp *PortfolioBreakdownResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, advancedTradePrivateEPL, http.MethodGet, advancedTradePath+"portfolios/"+req.PortfolioUUID, params, nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil || resp.Breakdown == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Breakdown, nil
}

// ListPortfolios returns the account's portfolios of a type, one of the PortfolioType values, or all of them when
// portfolioType is empty
func (e *Exchange) ListPortfolios(ctx context.Context, portfolioType string) ([]Portfolio, error) {
	params := url.Values{}
	switch portfolioType {
	case "":
	case PortfolioTypeDefault, PortfolioTypeConsumer, PortfolioTypeINTX:
		params.Set("portfolio_type", portfolioType)
	default:
		return nil, fmt.Errorf("%w: %q", errInvalidPortfolioType, portfolioType)
	}
	var resp *PortfoliosResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, advancedTradePrivateEPL, http.MethodGet, advancedTradePath+"portfolios", params, nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Portfolios, nil
}

// MovePortfolioFunds moves an amount of a currency from one portfolio to another, at once and free. It funds a Global
// Derivatives account: collateral is moved into the derivatives portfolio, which ListPortfolios lists, before orders
// are placed on the gateway
func (e *Exchange) MovePortfolioFunds(ctx context.Context, req *MovePortfolioFundsRequest) (*MovePortfolioFundsResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.SourcePortfolioUUID == "" {
		return nil, fmt.Errorf("%w: source portfolio", errPortfolioIDRequired)
	}
	if req.TargetPortfolioUUID == "" {
		return nil, fmt.Errorf("%w: target portfolio", errPortfolioIDRequired)
	}
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.Amount <= 0 {
		return nil, fmt.Errorf("%w: %v", errInvalidAmount, req.Amount)
	}
	body := &MovePortfolioFundsWireRequest{
		Funds:               PortfolioFunds{Value: formatFloat(req.Amount), Currency: req.Currency.Upper().String()},
		SourcePortfolioUUID: req.SourcePortfolioUUID,
		TargetPortfolioUUID: req.TargetPortfolioUUID,
	}
	var resp *MovePortfolioFundsResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, advancedTradePrivateEPL, http.MethodPost, advancedTradePath+"portfolios/move_funds", nil, body, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}
