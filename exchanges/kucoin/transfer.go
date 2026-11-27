package kucoin

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// GetTransferQuotas returns the transferable balance of an account; SPOT Margin LF accounts are no longer supported
// (Get Transfer Quotas)
func (e *Exchange) GetTransferQuotas(ctx context.Context, arg *TransferQuotasRequest) (*TransferQuotasResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.AccountType == "" {
		return nil, errAccountTypeMissing
	}
	params := url.Values{}
	params.Set("currency", arg.Currency.Upper().String())
	params.Set("type", arg.AccountType)
	if arg.Tag != "" {
		params.Set("tag", arg.Tag)
	}
	var resp *TransferQuotasResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getTransferQuotasEPL, http.MethodGet, common.EncodeURLValues("/v1/accounts/transferable", params), nil, &resp)
}

// FlexTransfer transfers funds between the account types of an account, between the master account and a
// sub-account, or between sub-accounts; it needs a master account API key with the Universal Transfer permission
// (Flex Transfer)
func (e *Exchange) FlexTransfer(ctx context.Context, arg *FlexTransferRequest) (*FlexTransferResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.ClientOrderID == "" {
		return nil, order.ErrClientOrderIDMustBeSet
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if arg.FromAccountType == "" {
		return nil, fmt.Errorf("%w: from account type", errAccountTypeMissing)
	}
	if arg.Type == "" {
		return nil, errTransferTypeMissing
	}
	if arg.ToAccountType == "" {
		return nil, fmt.Errorf("%w: to account type", errAccountTypeMissing)
	}
	body := make(map[string]any)
	body["clientOid"] = arg.ClientOrderID
	body["currency"] = arg.Currency.Upper().String()
	body["amount"] = strconv.FormatFloat(arg.Amount, 'f', -1, 64)
	body["fromAccountType"] = arg.FromAccountType
	body["type"] = arg.Type
	body["toAccountType"] = arg.ToAccountType
	if arg.FromUserID != "" {
		body["fromUserId"] = arg.FromUserID
	}
	if arg.FromAccountTag != "" {
		body["fromAccountTag"] = arg.FromAccountTag
	}
	if arg.ToUserID != "" {
		body["toUserId"] = arg.ToUserID
	}
	if arg.ToAccountTag != "" {
		body["toAccountTag"] = arg.ToAccountTag
	}
	var resp *FlexTransferResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, flexTransferEPL, http.MethodPost, "/v3/accounts/universal-transfer", body, &resp)
}

// Trade type values for Spot, Isolated Margin, and Cross Margin accounts
const (
	SpotTradeType           = "TRADE"
	IsolatedMarginTradeType = "MARGIN_ISOLATED_TRADE"
	CrossMarginTradeType    = "MARGIN_TRADE"
)
