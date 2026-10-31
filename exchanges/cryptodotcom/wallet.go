package cryptodotcom

import (
	"context"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

// maxWalletHistoryPageSize is the most deposits or withdrawals a history page holds
const maxWalletHistoryPageSize = 200

// CreateWithdrawal withdraws a currency to an address whitelisted for the API key, which needs the withdrawal
// permission
func (e *Exchange) CreateWithdrawal(ctx context.Context, req *WithdrawalRequest) (*WithdrawalResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.Amount <= 0 {
		return nil, errInvalidAmount
	}
	if req.Address == "" {
		return nil, errAddressRequired
	}
	params := *req
	params.Currency = req.Currency.Upper()
	var resp *WithdrawalResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/create-withdrawal", &params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetCurrencyNetworks returns the networks each currency can be deposited and withdrawn on, with their withdrawal fees
// and minimums
func (e *Exchange) GetCurrencyNetworks(ctx context.Context) (*CurrencyNetworksResponse, error) {
	var resp *CurrencyNetworksResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/get-currency-networks", nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetDepositAddresses returns a currency's deposit addresses, one for each network; the API key needs the withdrawal
// permission
func (e *Exchange) GetDepositAddresses(ctx context.Context, ccy currency.Code) ([]DepositAddressResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp *depositAddressesResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/get-deposit-address", &depositAddressParams{Currency: ccy.Upper()}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp.DepositAddresses, nil
}

// GetDepositHistory returns a page of the account's crypto deposits
func (e *Exchange) GetDepositHistory(ctx context.Context, req *WalletHistoryRequest) ([]DepositResponse, error) {
	params, err := newWalletHistoryParams(req)
	if err != nil {
		return nil, err
	}
	var resp *depositHistoryResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/get-deposit-history", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Deposits, nil
}

// GetWithdrawalHistory returns a page of the account's crypto withdrawals
func (e *Exchange) GetWithdrawalHistory(ctx context.Context, req *WalletHistoryRequest) ([]WithdrawalResponse, error) {
	params, err := newWalletHistoryParams(req)
	if err != nil {
		return nil, err
	}
	var resp *withdrawalHistoryResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/get-withdrawal-history", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Withdrawals, nil
}

// newWalletHistoryParams validates a deposit or withdrawal history request and returns its params
func newWalletHistoryParams(req *WalletHistoryRequest) (*walletHistoryParams, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	if req.PageSize > maxWalletHistoryPageSize {
		return nil, fmt.Errorf("%w: page size %d above %d", errInvalidCount, req.PageSize, maxWalletHistoryPageSize)
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	startTime, err := unixMilliParam(req.StartTime)
	if err != nil {
		return nil, err
	}
	endTime, err := unixMilliParam(req.EndTime)
	if err != nil {
		return nil, err
	}
	return &walletHistoryParams{
		Currency:  req.Currency.Upper().String(),
		StartTime: startTime,
		EndTime:   endTime,
		PageSize:  req.PageSize,
		Page:      req.Page,
		Status:    req.Status,
	}, nil
}
