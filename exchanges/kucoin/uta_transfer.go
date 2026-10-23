package kucoin

import (
	"context"
	"errors"
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

// errAccountTagRequired is returned when a flex transfer names no source or destination account tag
var errAccountTagRequired = errors.New("account tag is required")

// GetUTATransferQuotas returns how much of a currency can be transferred out of an account type, which can be less
// than its balance (Get Transfer Quotas)
func (e *Exchange) GetUTATransferQuotas(ctx context.Context, req *UTATransferQuotasRequest) (*UTATransferQuotasResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.AccountType == "" {
		return nil, errAccountTypeMissing
	}
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("accountType", req.AccountType)
	params.Set("currency", req.Currency.Upper().String())
	if req.Symbol != "" {
		params.Set("symbol", req.Symbol)
	}
	var resp *UTATransferQuotasResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getUTATransferQuotasEPL, http.MethodGet, common.EncodeURLValues("/ua/v2/account/transfer-quota", params), nil, &resp)
}

// UTAFlexTransfer transfers assets between account types, or between the master account and its sub-accounts (Flex
// Transfer)
func (e *Exchange) UTAFlexTransfer(ctx context.Context, req *UTAFlexTransferRequest) (*UTAFlexTransferResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	switch {
	case req.ClientOrderID == "":
		return nil, order.ErrClientOrderIDMustBeSet
	case req.TransferType == "":
		return nil, errTransferTypeMissing
	case req.Currency.IsEmpty():
		return nil, currency.ErrCurrencyCodeEmpty
	case req.Amount <= 0:
		return nil, limits.ErrAmountBelowMin
	case req.FromAccountType == "":
		return nil, fmt.Errorf("%w: from", errAccountTypeMissing)
	case req.FromAccountTag == "":
		return nil, fmt.Errorf("%w: from", errAccountTagRequired)
	case req.ToAccountType == "":
		return nil, fmt.Errorf("%w: to", errAccountTypeMissing)
	case req.ToAccountTag == "":
		return nil, fmt.Errorf("%w: to", errAccountTagRequired)
	}
	body := make(map[string]any)
	body["clientOid"] = req.ClientOrderID
	body["transferType"] = req.TransferType
	body["currency"] = req.Currency.Upper().String()
	body["amount"] = strconv.FormatFloat(req.Amount, 'f', -1, 64)
	body["fromAccountType"] = req.FromAccountType
	body["fromAccountTag"] = req.FromAccountTag
	body["toAccountType"] = req.ToAccountType
	body["toAccountTag"] = req.ToAccountTag
	if req.FromUID != "" {
		body["fromUid"] = req.FromUID
	}
	if req.ToUID != "" {
		body["toUid"] = req.ToUID
	}
	var resp *UTAFlexTransferResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, utaFlexTransferEPL, http.MethodPost, "/ua/v2/account/transfer", body, &resp)
}
