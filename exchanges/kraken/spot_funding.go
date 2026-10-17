package kraken

import (
	"context"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
)

// GetDepositMethods calls Get Deposit Methods, returning the methods an asset can be deposited with. Kraken no longer
// updates this legacy endpoint and recommends Funding (Beta) instead
func (e *Exchange) GetDepositMethods(ctx context.Context, req *DepositMethodsRequest) ([]DepositMethod, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Asset.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	body := make(map[string]any)
	body["asset"] = req.Asset.Upper().String()
	if req.AssetClass != "" {
		body["aclass"] = req.AssetClass
	}
	if req.RebaseMultiplier != "" {
		body["rebase_multiplier"] = req.RebaseMultiplier
	}
	var resp []DepositMethod
	return resp, e.SendAuthenticatedHTTPRequest(ctx, "/0/private/DepositMethods", nil, body, &resp)
}

// GetDepositAddresses calls Get Deposit Addresses, returning an asset's deposit addresses for a method, generating a
// new one when New is set. Kraken no longer updates this legacy endpoint and recommends Funding (Beta) instead
func (e *Exchange) GetDepositAddresses(ctx context.Context, req *DepositAddressesRequest) ([]DepositAddress, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Asset.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.Method == "" {
		return nil, errDepositMethodRequired
	}
	body := make(map[string]any)
	body["asset"] = req.Asset.Upper().String()
	body["method"] = req.Method
	if req.AssetClass != "" {
		body["aclass"] = req.AssetClass
	}
	if req.New {
		body["new"] = true
	}
	if req.Amount > 0 {
		body["amount"] = strconv.FormatFloat(req.Amount, 'f', -1, 64)
	}
	var resp []DepositAddress
	return resp, e.SendAuthenticatedHTTPRequest(ctx, "/0/private/DepositAddresses", nil, body, &resp)
}

// GetRecentDepositsStatus calls Get Status of Recent Deposits, returning deposits newest first. A paginated request
// returns a page of deposits and the cursor of the next. A nil request returns the recent deposits of every asset.
// Kraken no longer updates this legacy endpoint and recommends Funding (Beta) instead
func (e *Exchange) GetRecentDepositsStatus(ctx context.Context, req *RecentTransfersStatusRequest) (*RecentDepositsStatusResponse, error) {
	body, err := recentTransfersBody(req)
	if err != nil {
		return nil, err
	}
	var resp *RecentDepositsStatusResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, "/0/private/DepositStatus", nil, body, &resp)
}

// GetWithdrawalMethods calls Get Withdrawal Methods, returning the methods the account can withdraw with and their
// minimums, fees and limits. A nil request returns every asset's methods. Kraken no longer updates this legacy endpoint
// and recommends Funding (Beta) instead
func (e *Exchange) GetWithdrawalMethods(ctx context.Context, req *WithdrawalMethodsRequest) ([]WithdrawalMethod, error) {
	body := make(map[string]any)
	if req != nil {
		if !req.Asset.IsEmpty() {
			body["asset"] = req.Asset.Upper().String()
		}
		if req.AssetClass != "" {
			body["aclass"] = req.AssetClass
		}
		if req.Network != "" {
			body["network"] = req.Network
		}
		if req.RebaseMultiplier != "" {
			body["rebase_multiplier"] = req.RebaseMultiplier
		}
	}
	var resp []WithdrawalMethod
	return resp, e.SendAuthenticatedHTTPRequest(ctx, "/0/private/WithdrawMethods", nil, body, &resp)
}

// GetWithdrawalAddresses calls Get Withdrawal Addresses, returning the withdrawal addresses set up on the account. A
// nil request returns every address. Kraken no longer updates this legacy endpoint and recommends Funding (Beta)
// instead
func (e *Exchange) GetWithdrawalAddresses(ctx context.Context, req *WithdrawalAddressesRequest) ([]WithdrawalAddress, error) {
	body := make(map[string]any)
	if req != nil {
		if !req.Asset.IsEmpty() {
			body["asset"] = req.Asset.Upper().String()
		}
		if req.AssetClass != "" {
			body["aclass"] = req.AssetClass
		}
		if req.Method != "" {
			body["method"] = req.Method
		}
		if req.Key != "" {
			body["key"] = req.Key
		}
		if req.Verified != nil {
			body["verified"] = *req.Verified
		}
	}
	var resp []WithdrawalAddress
	return resp, e.SendAuthenticatedHTTPRequest(ctx, "/0/private/WithdrawAddresses", nil, body, &resp)
}

// GetWithdrawalInformation calls Get Withdrawal Information, returning the fee and net amount of withdrawing an amount
// to a withdrawal key's address, and the most that can be withdrawn right now. Kraken no longer updates this legacy
// endpoint and recommends Funding (Beta) instead
func (e *Exchange) GetWithdrawalInformation(ctx context.Context, req *WithdrawalInformationRequest) (*WithdrawalInformationResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Asset.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.Key == "" {
		return nil, errWithdrawalKeyRequired
	}
	if req.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	body := make(map[string]any)
	body["asset"] = req.Asset.Upper().String()
	body["key"] = req.Key
	body["amount"] = strconv.FormatFloat(req.Amount, 'f', -1, 64)
	var resp *WithdrawalInformationResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, "/0/private/WithdrawInfo", nil, body, &resp)
}

// WithdrawFunds calls Withdraw Funds, withdrawing to the address set up under a withdrawal key and returning the
// withdrawal's reference ID. Kraken no longer updates this legacy endpoint and recommends Funding (Beta) instead
func (e *Exchange) WithdrawFunds(ctx context.Context, req *WithdrawFundsRequest) (*WithdrawFundsResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Asset.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.Key == "" {
		return nil, errWithdrawalKeyRequired
	}
	if req.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	body := make(map[string]any)
	body["asset"] = req.Asset.Upper().String()
	body["key"] = req.Key
	body["amount"] = strconv.FormatFloat(req.Amount, 'f', -1, 64)
	if req.AssetClass != "" {
		body["aclass"] = req.AssetClass
	}
	if req.Address != "" {
		body["address"] = req.Address
	}
	if req.MaxFee > 0 {
		body["max_fee"] = strconv.FormatFloat(req.MaxFee, 'f', -1, 64)
	}
	if req.RebaseMultiplier != "" {
		body["rebase_multiplier"] = req.RebaseMultiplier
	}
	var resp *WithdrawFundsResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, "/0/private/Withdraw", nil, body, &resp)
}

// GetRecentWithdrawalsStatus calls Get Status of Recent Withdrawals, returning withdrawals newest first. A paginated
// request returns a page of withdrawals and the cursor of the next. A nil request returns the recent withdrawals of
// every asset. Kraken no longer updates this legacy endpoint and recommends Funding (Beta) instead
func (e *Exchange) GetRecentWithdrawalsStatus(ctx context.Context, req *RecentTransfersStatusRequest) (*RecentWithdrawalsStatusResponse, error) {
	body, err := recentTransfersBody(req)
	if err != nil {
		return nil, err
	}
	var resp *RecentWithdrawalsStatusResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, "/0/private/WithdrawStatus", nil, body, &resp)
}

// CancelWithdrawal calls Request Withdrawal Cancellation, cancelling a withdrawal or wallet transfer Kraken has not yet
// processed and reporting whether the cancellation succeeded. Kraken no longer updates this legacy endpoint and
// recommends Funding (Beta) instead
func (e *Exchange) CancelWithdrawal(ctx context.Context, asset currency.Code, referenceID string) (bool, error) {
	if asset.IsEmpty() {
		return false, currency.ErrCurrencyCodeEmpty
	}
	if referenceID == "" {
		return false, errWithdrawalReferenceIDRequired
	}
	body := make(map[string]any)
	body["asset"] = asset.Upper().String()
	body["refid"] = referenceID
	var resp bool
	return resp, e.SendAuthenticatedHTTPRequest(ctx, "/0/private/WithdrawCancel", nil, body, &resp)
}

// WalletTransfer calls Request Wallet Transfer, moving an amount of an asset from the spot wallet to the futures
// wallet and returning the transfer's reference ID. Transfers the other way are requested through the Derivatives
// API's withdrawal to the spot wallet. Kraken no longer updates this legacy endpoint and recommends Funding (Beta)
// instead
func (e *Exchange) WalletTransfer(ctx context.Context, asset currency.Code, amount float64) (*WalletTransferResponse, error) {
	if asset.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	body := make(map[string]any)
	body["asset"] = asset.Upper().String()
	body["from"] = "Spot Wallet"
	body["to"] = "Futures Wallet"
	body["amount"] = strconv.FormatFloat(amount, 'f', -1, 64)
	var resp *WalletTransferResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, "/0/private/WalletTransfer", nil, body, &resp)
}

// recentTransfersBody returns the parameters Get Status of Recent Deposits and Get Status of Recent Withdrawals share
func recentTransfersBody(req *RecentTransfersStatusRequest) (map[string]any, error) {
	body := make(map[string]any)
	if req == nil {
		return body, nil
	}
	if !req.Start.IsZero() && !req.End.IsZero() && req.Start.After(req.End) {
		return nil, common.ErrStartAfterEnd
	}
	if !req.Asset.IsEmpty() {
		body["asset"] = req.Asset.Upper().String()
	}
	if req.AssetClass != "" {
		body["aclass"] = req.AssetClass
	}
	if req.Method != "" {
		body["method"] = req.Method
	}
	if !req.Start.IsZero() {
		body["start"] = strconv.FormatInt(req.Start.Unix(), 10)
	}
	if !req.End.IsZero() {
		body["end"] = strconv.FormatInt(req.End.Unix(), 10)
	}
	// Kraken takes cursor as true for the first page of a paginated response, then as the cursor of each later page
	switch {
	case req.Cursor != "":
		body["cursor"] = req.Cursor
	case req.Paginate:
		body["cursor"] = true
	}
	if req.Limit != 0 {
		body["limit"] = req.Limit
	}
	if req.RebaseMultiplier != "" {
		body["rebase_multiplier"] = req.RebaseMultiplier
	}
	return body, nil
}
