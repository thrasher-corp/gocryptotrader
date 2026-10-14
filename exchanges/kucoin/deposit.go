package kucoin

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// errChainRequired is returned when a deposit address request has no chain
var errChainRequired = errors.New("chain is required")

// AddDepositAddressV3 creates a deposit address for a currency on a chain (Add Deposit Address (V3))
func (e *Exchange) AddDepositAddressV3(ctx context.Context, arg *AddDepositAddressV3Request) (*AddDepositAddressV3Response, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Chain == "" {
		return nil, errChainRequired
	}
	body := make(map[string]any)
	body["currency"] = arg.Currency.Upper().String()
	body["chain"] = arg.Chain
	if arg.To != "" {
		body["to"] = arg.To
	}
	if arg.Amount != 0 {
		body["amount"] = strconv.FormatFloat(arg.Amount, 'f', -1, 64)
	}
	var resp *AddDepositAddressV3Response
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, addDepositAddressV3EPL, http.MethodPost, "/v3/deposit-address/create", body, &resp)
}

// GetDepositAddressV3 returns every deposit address of a currency, optionally on one chain; an empty list means no
// address has been added yet (Get Deposit Address (V3))
func (e *Exchange) GetDepositAddressV3(ctx context.Context, ccy currency.Code, chain string) ([]DepositAddress, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.Upper().String())
	if chain != "" {
		params.Set("chain", chain)
	}
	var resp []DepositAddress
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getDepositAddressV3EPL, http.MethodGet, common.EncodeURLValues("/v3/deposit-addresses", params), nil, &resp)
}

// GetDepositHistory returns a page of a currency's deposits, latest first (Get Deposit History)
func (e *Exchange) GetDepositHistory(ctx context.Context, arg *DepositHistoryRequest) (*DepositHistoryResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", arg.Currency.Upper().String())
	if arg.Status != "" {
		params.Set("status", arg.Status)
	}
	if err := setTimeRange(params, "startAt", "endAt", arg.StartAt, arg.EndAt); err != nil {
		return nil, err
	}
	if arg.CurrentPage != 0 {
		params.Set("currentPage", strconv.FormatUint(arg.CurrentPage, 10))
	}
	if arg.PageSize != 0 {
		params.Set("pageSize", strconv.FormatUint(arg.PageSize, 10))
	}
	var resp *DepositHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getDepositHistoryEPL, http.MethodGet, common.EncodeURLValues("/v1/deposits", params), nil, &resp)
}
