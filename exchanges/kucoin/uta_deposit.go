package kucoin

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// GetUTADepositAddress returns a currency's deposit addresses. With a chain, that chain's address is returned, and
// created when it does not exist yet; without one, the list is empty when the currency has no address (Get Deposit
// Address)
func (e *Exchange) GetUTADepositAddress(ctx context.Context, ccy currency.Code, chain string) ([]UTADepositAddress, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	if chain != "" {
		params.Set("chain", chain)
	}
	params.Set("currency", ccy.Upper().String())
	var resp []UTADepositAddress
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getUTADepositAddressEPL, http.MethodGet, common.EncodeURLValues("/ua/v2/asset/deposit/address", params), nil, &resp)
}

// GetUTADepositHistory returns a page of deposit records with their confirmation progress (Get Deposit History)
func (e *Exchange) GetUTADepositHistory(ctx context.Context, req *UTAAssetHistoryRequest) (*UTADepositHistoryResponse, error) {
	params, err := req.values()
	if err != nil {
		return nil, err
	}
	var resp *UTADepositHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getUTADepositHistoryEPL, http.MethodGet, common.EncodeURLValues("/ua/v2/asset/deposit/history", params), nil, &resp)
}

// values returns the query parameters of a deposit or withdrawal history request
func (r *UTAAssetHistoryRequest) values() (url.Values, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	params := url.Values{}
	if !r.Currency.IsEmpty() {
		params.Set("currency", r.Currency.Upper().String())
	}
	if r.ID != "" {
		params.Set("id", r.ID)
	}
	if r.Status != "" {
		params.Set("status", r.Status)
	}
	if err := setTimeRange(params, "startAt", "endAt", r.StartAt, r.EndAt); err != nil {
		return nil, err
	}
	if r.CurrentPage != 0 {
		params.Set("currentPage", strconv.FormatUint(r.CurrentPage, 10))
	}
	if r.PageSize != 0 {
		params.Set("pageSize", strconv.FormatUint(r.PageSize, 10))
	}
	return params, nil
}
