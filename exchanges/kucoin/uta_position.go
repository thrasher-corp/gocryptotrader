package kucoin

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// errMarginChangeTypeRequired is returned when Modify Position Margin is not told whether to add or remove margin
var errMarginChangeTypeRequired = errors.New("margin change type is required")

// GetUTAPositionList returns the unified trading account's open futures positions, ordered by KuCoin's internal
// symbol ID (Get Position List)
func (e *Exchange) GetUTAPositionList(ctx context.Context, r *UTAPositionListRequest) ([]UTAPositionResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	params := url.Values{}
	if r.Symbol != "" {
		params.Set("symbol", r.Symbol)
	}
	if r.PageNumber != 0 {
		params.Set("pageNumber", strconv.FormatUint(r.PageNumber, 10))
	}
	if r.PageSize != 0 {
		params.Set("pageSize", strconv.FormatUint(r.PageSize, 10))
	}
	var resp []UTAPositionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getUTAPositionListEPL, http.MethodGet, common.EncodeURLValues("/ua/v2/unified/position/open-list", params), nil, &resp)
}

// GetUTAPositionsHistory returns a page of the unified trading account's closed futures positions (Get Positions
// History)
func (e *Exchange) GetUTAPositionsHistory(ctx context.Context, r *UTAPositionsHistoryRequest) (*UTAPositionsHistoryResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	params := url.Values{}
	if r.Symbol != "" {
		params.Set("symbol", r.Symbol)
	}
	if err := setTimeRange(params, "startAt", "endAt", r.StartAt, r.EndAt); err != nil {
		return nil, err
	}
	if r.LastID != 0 {
		params.Set("lastId", strconv.FormatUint(r.LastID, 10))
	}
	if r.PageSize != 0 {
		params.Set("pageSize", strconv.FormatUint(r.PageSize, 10))
	}
	var resp *UTAPositionsHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getUTAPositionsHistoryEPL, http.MethodGet, common.EncodeURLValues("/ua/v2/position/history", params), nil, &resp)
}

// GetUTAPrivateFundingFeeHistory returns a page of the funding fees the unified trading account's futures positions
// settled (Get Private Funding Fee History)
func (e *Exchange) GetUTAPrivateFundingFeeHistory(ctx context.Context, r *UTAPrivateFundingFeeHistoryRequest) (*UTAPrivateFundingFeeHistoryResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	params := url.Values{}
	if r.Symbol != "" {
		params.Set("symbol", r.Symbol)
	}
	if err := setTimeRange(params, "startAt", "endAt", r.StartAt, r.EndAt); err != nil {
		return nil, err
	}
	if r.LastID != 0 {
		params.Set("lastId", strconv.FormatUint(r.LastID, 10))
	}
	if r.PageSize != 0 {
		params.Set("pageSize", strconv.FormatUint(r.PageSize, 10))
	}
	var resp *UTAPrivateFundingFeeHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getUTAPrivateFundingFeeHistoryEPL, http.MethodGet, common.EncodeURLValues("/ua/v2/position/funding-history", params), nil, &resp)
}

// ModifyUTAMarginMode sets the margin mode, CROSS or ISOLATED, of unified trading account futures symbols. Each symbol
// of the result carries KuCoin's verdict on it, since one failed change does not fail the request (Modify Margin Mode)
func (e *Exchange) ModifyUTAMarginMode(ctx context.Context, symbols []string, marginMode string) (*ModifyUTAMarginModeResponse, error) {
	if len(symbols) == 0 || slices.Contains(symbols, "") {
		return nil, currency.ErrSymbolStringEmpty
	}
	if marginMode == "" {
		return nil, margin.ErrInvalidMarginType
	}
	body := make(map[string]any)
	// The docs' only example separates the symbols with a comma and a space
	body["symbol"] = strings.Join(symbols, ", ")
	body["marginMode"] = marginMode
	var resp *ModifyUTAMarginModeResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, modifyUTAMarginModeEPL, http.MethodPost, "/ua/v2/unified/position/margin-mode", body, &resp)
}

// ModifyUTAPositionMargin adds margin to or removes margin from a unified trading account isolated futures position
// (Modify Position Margin)
func (e *Exchange) ModifyUTAPositionMargin(ctx context.Context, r *ModifyUTAPositionMarginRequest) (*ModifyUTAPositionMarginResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	switch {
	case r.Type == "":
		return nil, errMarginChangeTypeRequired
	case r.Amount <= 0:
		return nil, order.ErrAmountIsInvalid
	case r.Symbol == "":
		return nil, currency.ErrSymbolStringEmpty
	case r.TradeType == "":
		return nil, errTradeTypeMissing
	}
	body := make(map[string]any)
	body["type"] = r.Type
	body["amount"] = r.Amount
	body["symbol"] = r.Symbol
	body["tradeType"] = r.TradeType
	var resp *ModifyUTAPositionMarginResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, modifyUTAPositionMarginEPL, http.MethodPost, "/ua/v2/unified/position/modify-margin", body, &resp)
}

// GetUTAMarginMode returns the margin mode of a unified trading account futures symbol, or of the account's symbols
// when symbol is empty (Get Margin Mode)
func (e *Exchange) GetUTAMarginMode(ctx context.Context, symbol string) (*UTAMarginModeResponse, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp *UTAMarginModeResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getUTAMarginModeEPL, http.MethodGet, common.EncodeURLValues("/ua/v2/unified/position/margin-mode", params), nil, &resp)
}
