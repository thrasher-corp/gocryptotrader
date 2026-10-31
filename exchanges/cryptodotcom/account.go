package cryptodotcom

import (
	"context"
	"errors"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

const (
	// maxDailyBalanceHistoryLimit and maxHourlyBalanceHistoryLimit are the most points of each timeframe a balance
	// history request returns
	maxDailyBalanceHistoryLimit  = 30
	maxHourlyBalanceHistoryLimit = 120
)

var (
	errFromAccountRequired            = errors.New("account to transfer from required")
	errToAccountRequired              = errors.New("account to transfer to required")
	errInvalidBalanceHistoryTimeframe = errors.New("invalid balance history timeframe")
)

// GetUserBalance returns the account's balance: its margin totals, its balance of each collateral and the balances of
// its isolated positions
func (e *Exchange) GetUserBalance(ctx context.Context) ([]UserBalanceResponse, error) {
	var resp *dataResponse[UserBalanceResponse]
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/user-balance", nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Data, nil
}

// GetUserBalanceHistory returns the account's total cash balance over time, which may for a while differ from what
// Crypto.com's website shows
func (e *Exchange) GetUserBalanceHistory(ctx context.Context, req *UserBalanceHistoryRequest) (*UserBalanceHistoryResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	maxLimit := uint64(maxDailyBalanceHistoryLimit)
	switch req.Timeframe {
	case "", BalanceHistoryTimeframeDaily:
	case BalanceHistoryTimeframeHourly:
		maxLimit = maxHourlyBalanceHistoryLimit
	default:
		return nil, fmt.Errorf("%w: %q", errInvalidBalanceHistoryTimeframe, req.Timeframe)
	}
	if req.Limit > maxLimit {
		return nil, fmt.Errorf("%w: %d above %d", errInvalidCount, req.Limit, maxLimit)
	}
	endTime, err := unixNanoParam(req.EndTime)
	if err != nil {
		return nil, err
	}
	params := &userBalanceHistoryParams{Timeframe: req.Timeframe, EndTime: endTime, Limit: req.Limit}
	var resp *UserBalanceHistoryResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/user-balance-history", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetAccounts returns the API key's account, its master account and a page of the master account's sub-accounts
func (e *Exchange) GetAccounts(ctx context.Context, req *AccountsRequest) (*AccountsResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	var resp *AccountsResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/get-accounts", req, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// CreateSubAccountTransfer moves funds between the master account and a sub-account, or between two sub-accounts
func (e *Exchange) CreateSubAccountTransfer(ctx context.Context, req *SubAccountTransferRequest) error {
	if req == nil {
		return common.ErrNilPointer
	}
	if req.FromAccountUUID == "" {
		return errFromAccountRequired
	}
	if req.ToAccountUUID == "" {
		return errToAccountRequired
	}
	if req.Currency.IsEmpty() {
		return currency.ErrCurrencyCodeEmpty
	}
	if req.Amount <= 0 {
		return errInvalidAmount
	}
	params := *req
	params.Currency = req.Currency.Upper()
	return e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/create-subaccount-transfer", &params, nil)
}

// GetSubAccountBalances returns the balance of each of the master account's sub-accounts
func (e *Exchange) GetSubAccountBalances(ctx context.Context) ([]UserBalanceResponse, error) {
	var resp *dataResponse[UserBalanceResponse]
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/get-subaccount-balances", nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Data, nil
}

// GetPositions returns the account's open derivatives positions; the user.positions channel also reports their mark
// and liquidation prices and margins
func (e *Exchange) GetPositions(ctx context.Context, req *PositionsRequest) ([]PositionResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	var resp *dataResponse[PositionResponse]
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/get-positions", req, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Data, nil
}

// GetFeeCreditBalances returns a page of the account's fee credits, by expiry date in ascending order
func (e *Exchange) GetFeeCreditBalances(ctx context.Context, req *FeeCreditBalancesRequest) ([]FeeCreditResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	var resp *feeCreditBalancesResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/get-fee-credit-balances", req, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp.FeeCredits, nil
}
