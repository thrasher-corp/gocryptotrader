package coinbase

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/thrasher-corp/gocryptotrader/common"
)

var (
	errConvertAccountRequired = errors.New("convert account required")
	errConvertTradeIDRequired = errors.New("convert trade ID required")
)

// CommitConvertTrade commits a convert quote, converting its amount; the trade ID is the quote's, and the accounts
// those it was quoted for
func (e *Exchange) CommitConvertTrade(ctx context.Context, req *ConvertTradeRequest) (*ConvertTrade, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.TradeID == "" {
		return nil, errConvertTradeIDRequired
	}
	if err := checkConvertAccounts(req.FromAccount, req.ToAccount); err != nil {
		return nil, err
	}
	var resp *ConvertTradeResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, advancedTradePrivateEPL, http.MethodPost, advancedTradePath+"convert/trade/"+req.TradeID, nil, &ConvertAccountsWireRequest{FromAccount: req.FromAccount, ToAccount: req.ToAccount}, &resp); err != nil {
		return nil, err
	}
	if resp == nil || resp.Trade == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Trade, nil
}

// CreateConvertQuote quotes converting an amount between two currencies, USDC and USD, PYUSD and USD, EURC and EUR, or
// PYUSD and USDC; CommitConvertTrade converts it by the quote's trade ID
func (e *Exchange) CreateConvertQuote(ctx context.Context, req *CreateConvertQuoteRequest) (*ConvertTrade, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if err := checkConvertAccounts(req.FromAccount, req.ToAccount); err != nil {
		return nil, err
	}
	if req.Amount <= 0 {
		return nil, fmt.Errorf("%w: %v", errInvalidAmount, req.Amount)
	}
	body := &CreateConvertQuoteWireRequest{FromAccount: req.FromAccount, ToAccount: req.ToAccount, Amount: formatFloat(req.Amount)}
	if req.UserIncentiveID != "" || req.CodeValue != "" {
		body.TradeIncentiveMetadata = &ConvertTradeIncentiveMetadata{UserIncentiveID: req.UserIncentiveID, CodeValue: req.CodeValue}
	}
	var resp *ConvertTradeResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, advancedTradePrivateEPL, http.MethodPost, advancedTradePath+"convert/quote", nil, body, &resp); err != nil {
		return nil, err
	}
	if resp == nil || resp.Trade == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Trade, nil
}

// GetConvertTrade returns a convert quote or trade by its trade ID and the accounts it converts between
func (e *Exchange) GetConvertTrade(ctx context.Context, req *ConvertTradeRequest) (*ConvertTrade, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.TradeID == "" {
		return nil, errConvertTradeIDRequired
	}
	if err := checkConvertAccounts(req.FromAccount, req.ToAccount); err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("from_account", req.FromAccount)
	params.Set("to_account", req.ToAccount)
	var resp *ConvertTradeResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, advancedTradePrivateEPL, http.MethodGet, advancedTradePath+"convert/trade/"+req.TradeID, params, nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil || resp.Trade == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Trade, nil
}

// checkConvertAccounts returns an error when a convert request leaves the account it converts from or to unset
func checkConvertAccounts(fromAccount, toAccount string) error {
	if fromAccount == "" {
		return fmt.Errorf("%w: from account", errConvertAccountRequired)
	}
	if toAccount == "" {
		return fmt.Errorf("%w: to account", errConvertAccountRequired)
	}
	return nil
}
