package cryptodotcom

import (
	"context"
	"errors"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

var (
	errPageSizeRequired         = errors.New("page size required")
	errBankAccountIDRequired    = errors.New("bank account ID required")
	errPlaidAccountIDRequired   = errors.New("account ID from Plaid required")
	errPlaidPublicTokenRequired = errors.New("public token from Plaid required")
)

// GetFiatAccess returns whether each currency can be deposited and withdrawn on each payment network, which brokers
// check on behalf of their users
func (e *Exchange) GetFiatAccess(ctx context.Context) ([]FiatAccessResponse, error) {
	var resp *fiatAccessesResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/fiat/fiat-access", nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Accesses, nil
}

// GetFiatDepositInfo returns the bank accounts to deposit fiat to, one for each payment network
func (e *Exchange) GetFiatDepositInfo(ctx context.Context, req *FiatPaymentNetworksRequest) ([]FiatDepositInfoResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	params := &fiatPaymentNetworksParams{PaymentNetworks: joinPaymentNetworks(req.PaymentNetworks)}
	var resp *fiatDepositInfoListResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/fiat/fiat-deposit-info", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp.DepositInfo, nil
}

// GetFiatDepositHistory returns a page of the account's fiat deposits
func (e *Exchange) GetFiatDepositHistory(ctx context.Context, req *FiatHistoryRequest) (*FiatTransactionHistoryResponse, error) {
	params, err := newFiatHistoryParams(req)
	if err != nil {
		return nil, err
	}
	var resp *FiatTransactionHistoryResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/fiat/fiat-deposit-history", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetFiatWithdrawHistory returns a page of the account's fiat withdrawals
func (e *Exchange) GetFiatWithdrawHistory(ctx context.Context, req *FiatHistoryRequest) (*FiatTransactionHistoryResponse, error) {
	params, err := newFiatHistoryParams(req)
	if err != nil {
		return nil, err
	}
	var resp *FiatTransactionHistoryResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/fiat/fiat-withdraw-history", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// CreateFiatWithdrawal withdraws fiat to a linked bank account; the API key needs the withdrawal permission
func (e *Exchange) CreateFiatWithdrawal(ctx context.Context, req *FiatWithdrawalRequest) (*FiatWithdrawalResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	if req.BankAccountID == "" {
		return nil, errBankAccountIDRequired
	}
	if req.Amount <= 0 {
		return nil, errInvalidAmount
	}
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.PaymentNetwork == "" {
		return nil, errPaymentNetworkRequired
	}
	params := *req
	params.Currency = req.Currency.Upper()
	var resp *FiatWithdrawalResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/fiat/fiat-create-withdraw", &params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetFiatTransactionQuota returns how much can be withdrawn on a payment network each day and month, and how much of
// that is used
func (e *Exchange) GetFiatTransactionQuota(ctx context.Context, paymentNetwork string) (*FiatTransactionQuotaResponse, error) {
	if paymentNetwork == "" {
		return nil, errPaymentNetworkRequired
	}
	var resp *fiatTransactionQuotaDetailsResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/fiat/fiat-transaction-quota", &fiatPaymentNetworkParams{PaymentNetwork: paymentNetwork}, &resp); err != nil {
		return nil, err
	}
	if resp == nil || resp.Details == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Details, nil
}

// GetFiatTransactionLimit returns the limits of deposits and payments on a payment network
func (e *Exchange) GetFiatTransactionLimit(ctx context.Context, paymentNetwork string) (*FiatTransactionLimitResponse, error) {
	if paymentNetwork == "" {
		return nil, errPaymentNetworkRequired
	}
	var resp *FiatTransactionLimitResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/fiat/fiat-transaction-limit", &fiatPaymentNetworkParams{PaymentNetwork: paymentNetwork}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// CreateFiatBankAccount links a bank account for fiat withdrawals through Plaid, and returns the account's bank
// accounts, the new one among them
func (e *Exchange) CreateFiatBankAccount(ctx context.Context, req *FiatBankAccountRequest) ([]FiatBankAccountResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	if req.PlaidAccountID == "" {
		return nil, errPlaidAccountIDRequired
	}
	if req.PlaidPublicToken == "" {
		return nil, errPlaidPublicTokenRequired
	}
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := *req
	params.Currency = req.Currency.Upper()
	var resp *fiatBankAccountsResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/fiat/fiat-create-bank-account", &params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp.BankAccounts, nil
}

// GetFiatBankAccounts returns the bank accounts linked for fiat withdrawals
func (e *Exchange) GetFiatBankAccounts(ctx context.Context, req *FiatPaymentNetworksRequest) ([]FiatBankAccountResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	params := &fiatPaymentNetworksParams{PaymentNetworks: joinPaymentNetworks(req.PaymentNetworks)}
	var resp *fiatBankAccountsResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/fiat/fiat-get-bank-accounts", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp.BankAccounts, nil
}

// newFiatHistoryParams validates a fiat deposit or withdrawal history request and returns its params
func newFiatHistoryParams(req *FiatHistoryRequest) (*fiatHistoryParams, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	if req.PageSize == 0 {
		return nil, errPageSizeRequired
	}
	// Crypto.com needs a start time with an end time, which the check reports as unset
	if !req.EndTime.IsZero() {
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
	return &fiatHistoryParams{
		Page:            req.Page,
		PageSize:        req.PageSize,
		StartTime:       startTime,
		EndTime:         endTime,
		PaymentNetworks: joinPaymentNetworks(req.PaymentNetworks),
	}, nil
}

// joinPaymentNetworks returns payment networks as the comma-separated list Crypto.com filters by, separated as the
// docs' example separates them
func joinPaymentNetworks(paymentNetworks []string) string {
	return strings.Join(paymentNetworks, ", ")
}
