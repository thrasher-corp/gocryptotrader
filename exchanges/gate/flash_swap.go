package gate

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// GetSupportedFlashSwapCurrencyPairs retrieves the flash swap currency pairs, where BTC_USDT means selling BTC for USDT
// A pair being swappable in one direction does not mean the reverse swap is supported
func (e *Exchange) GetSupportedFlashSwapCurrencyPairs(ctx context.Context, ccy currency.Code, limit, page uint64) ([]*FlashSwapCurrencyPairResponse, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	var pairs []*FlashSwapCurrencyPairResponse
	return pairs, e.SendHTTPRequest(ctx, exchange.RestSpot, publicFlashSwapEPL, common.EncodeURLValues("flash_swap/currency_pairs", params), &pairs)
}

// CreateFlashSwapOrder creates a flash swap order from a preview returned by InitiateFlashSwapOrderReview
func (e *Exchange) CreateFlashSwapOrder(ctx context.Context, arg *FlashSwapOrderRequest) (*FlashSwapOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.PreviewID == "" {
		return nil, errMissingPreviewID
	}
	if arg.BuyCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w, buy currency can not empty", currency.ErrCurrencyCodeEmpty)
	}
	if arg.SellCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w, sell currency can not empty", currency.ErrCurrencyCodeEmpty)
	}
	if arg.SellAmount <= 0 {
		return nil, fmt.Errorf("%w, sell_amount can not be less than or equal to 0", order.ErrAmountIsInvalid)
	}
	if arg.BuyAmount <= 0 {
		return nil, fmt.Errorf("%w, buy_amount amount can not be less than or equal to 0", order.ErrAmountIsInvalid)
	}
	var response *FlashSwapOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, flashSwapOrderEPL, http.MethodPost, "flash_swap/orders", nil, arg, &response)
}

// GetAllFlashSwapOrders retrieves flash swap orders, optionally filtered by status (1 success, 2 failure) and currencies
// reverse is always sent, so false lists the oldest orders first rather than applying Gate's newest first default
func (e *Exchange) GetAllFlashSwapOrders(ctx context.Context, status uint64, sellCurrency, buyCurrency currency.Code, reverse bool, limit, page uint64) ([]*FlashSwapOrderResponse, error) {
	params := url.Values{}
	if status == 1 || status == 2 {
		params.Set("status", strconv.FormatUint(status, 10))
	}
	if !sellCurrency.IsEmpty() {
		params.Set("sell_currency", sellCurrency.String())
	}
	if !buyCurrency.IsEmpty() {
		params.Set("buy_currency", buyCurrency.String())
	}
	params.Set("reverse", strconv.FormatBool(reverse))
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var response []*FlashSwapOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, flashGetOrdersEPL, http.MethodGet, "flash_swap/orders", params, nil, &response)
}

// GetSingleFlashSwapOrder retrieves a single flash swap order
func (e *Exchange) GetSingleFlashSwapOrder(ctx context.Context, orderID string) (*FlashSwapOrderResponse, error) {
	if orderID == "" {
		return nil, fmt.Errorf("%w, flash order order_id must not be empty", order.ErrOrderIDNotSet)
	}
	var response *FlashSwapOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, flashGetOrderEPL, http.MethodGet, "flash_swap/orders/"+orderID, nil, nil, &response)
}

// InitiateFlashSwapOrderReview previews a flash swap, quoting the amount received for SellAmount or the amount needed for
// BuyAmount; the returned preview ID is required to create the order
func (e *Exchange) InitiateFlashSwapOrderReview(ctx context.Context, arg *FlashSwapPreviewRequest) (*InitFlashSwapOrderPreviewResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.SellCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w, sell currency can not empty", currency.ErrCurrencyCodeEmpty)
	}
	if arg.BuyCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w, buy currency can not empty", currency.ErrCurrencyCodeEmpty)
	}
	if arg.SellAmount <= 0 && arg.BuyAmount <= 0 {
		return nil, fmt.Errorf("%w: either sell_amount or buy_amount is required", order.ErrAmountIsInvalid)
	}
	var response *InitFlashSwapOrderPreviewResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, flashOrderReviewEPL, http.MethodPost, "flash_swap/orders/preview", nil, arg, &response)
}
