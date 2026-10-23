package coinbase

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

var (
	errAppSendRecipientRequired       = errors.New("send recipient required")
	errAppTransactionIDRequired       = errors.New("transaction ID required")
	errAppInvalidTravelRuleWalletType = errors.New("invalid travel rule beneficiary wallet type")
	errAppInvalidTravelRuleIsSelf     = errors.New("invalid travel rule is_self value")
)

// SendMoney sends funds from an account to a blockchain address or an email address, with no fee for a send that
// stays off the blockchain. Sends are asynchronous, and Coinbase may delay or cancel one, so poll GetTransaction until
// the send's status is completed
func (a AppAPI) SendMoney(ctx context.Context, req *AppSendMoneyRequest) (*AppTransactionResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.AccountID == "" {
		return nil, errAccountIDRequired
	}
	if req.To == "" {
		return nil, errAppSendRecipientRequired
	}
	if req.Amount <= 0 {
		return nil, fmt.Errorf("%w: %v", errInvalidAmount, req.Amount)
	}
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if err := req.TravelRuleData.validate(); err != nil {
		return nil, err
	}
	body := &AppSendMoneyWireRequest{
		Type:              AppTransactionTypeSend,
		To:                req.To,
		Amount:            formatFloat(req.Amount),
		Currency:          req.Currency.Upper().String(),
		Description:       req.Description,
		SkipNotifications: req.SkipNotifications,
		IdempotencyKey:    req.IdempotencyKey,
		DestinationTag:    req.DestinationTag,
		Network:           req.Network,
		TravelRuleData:    req.TravelRuleData,
	}
	resp, err := sendAppRequest[*AppTransactionResponse](ctx, a, http.MethodPost, "accounts/"+req.AccountID+"/transactions", nil, body)
	if err != nil {
		return nil, err
	}
	if resp.Data == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Data, nil
}

// ListTransactions returns a page of an account's transactions
func (a AppAPI) ListTransactions(ctx context.Context, req *AppListTransactionsRequest) (*AppListResponse[AppTransactionResponse], error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.AccountID == "" {
		return nil, errAccountIDRequired
	}
	params, err := req.Page.params()
	if err != nil {
		return nil, err
	}
	resp, err := sendAppRequest[[]AppTransactionResponse](ctx, a, http.MethodGet, "accounts/"+req.AccountID+"/transactions", params, nil)
	if err != nil {
		return nil, err
	}
	return listResponse(resp), nil
}

// GetTransaction returns one of an account's transactions
func (a AppAPI) GetTransaction(ctx context.Context, req *AppGetTransactionRequest) (*AppTransactionResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.AccountID == "" {
		return nil, errAccountIDRequired
	}
	if req.TransactionID == "" {
		return nil, errAppTransactionIDRequired
	}
	resp, err := sendAppRequest[*AppTransactionResponse](ctx, a, http.MethodGet, "accounts/"+req.AccountID+"/transactions/"+req.TransactionID, nil, nil)
	if err != nil {
		return nil, err
	}
	if resp.Data == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Data, nil
}

// validate checks the travel rule fields that take a closed set of values; no data is valid, as most regions require
// none
func (d *AppTravelRuleData) validate() error {
	if d == nil {
		return nil
	}
	switch d.BeneficiaryWalletType {
	case "", AppTravelRuleWalletTypeSelfHosted, AppTravelRuleWalletTypeExchange:
	default:
		return fmt.Errorf("%w: %q", errAppInvalidTravelRuleWalletType, d.BeneficiaryWalletType)
	}
	switch d.IsSelf {
	case "", AppTravelRuleIsSelfFalse, AppTravelRuleIsSelfTrue:
	default:
		return fmt.Errorf("%w: %q", errAppInvalidTravelRuleIsSelf, d.IsSelf)
	}
	return nil
}
