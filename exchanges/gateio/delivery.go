package gateio

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// GetAllDeliveryContracts retrieves all futures contracts
func (e *Exchange) GetAllDeliveryContracts(ctx context.Context, settle currency.Code) ([]*DeliveryContractResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	var contracts []*DeliveryContractResponse
	return contracts, e.SendHTTPRequest(ctx, exchange.RestSpot, publicDeliveryContractsEPL, "delivery/"+settle.Item.Lower+"/contracts", &contracts)
}

// GetDeliveryContract retrieves a single delivery contract instance
func (e *Exchange) GetDeliveryContract(ctx context.Context, settle currency.Code, contract currency.Pair) (*DeliveryContractResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	var deliveryContract *DeliveryContractResponse
	return deliveryContract, e.SendHTTPRequest(ctx, exchange.RestSpot, publicDeliveryContractsEPL, "delivery/"+settle.Item.Lower+"/contracts/"+contract.String(), &deliveryContract)
}

// GetDeliveryOrderbook delivery orderbook
func (e *Exchange) GetDeliveryOrderbook(ctx context.Context, settle currency.Code, interval string, contract currency.Pair, limit uint64, withOrderbookID bool) (*OrderbookResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	params := url.Values{}
	params.Set("contract", contract.String())
	if interval != "" {
		params.Set("interval", interval)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if withOrderbookID {
		params.Set("with_id", strconv.FormatBool(withOrderbookID))
	}
	var orderbook *OrderbookResponse
	return orderbook, e.SendHTTPRequest(ctx, exchange.RestSpot, publicOrderbookDeliveryEPL, common.EncodeURLValues("delivery/"+settle.Item.Lower+"/order_book", params), &orderbook)
}

// GetDeliveryTradingHistory retrieves futures trading history
func (e *Exchange) GetDeliveryTradingHistory(ctx context.Context, settle currency.Code, lastID string, contract currency.Pair, limit uint64, from, to time.Time) ([]*TradingHistoryItemResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	params := url.Values{}
	params.Set("contract", contract.String())
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if lastID != "" {
		params.Set("last_id", lastID)
	}
	var histories []*TradingHistoryItemResponse
	return histories, e.SendHTTPRequest(ctx, exchange.RestSpot, publicTradingHistoryDeliveryEPL, common.EncodeURLValues("delivery/"+settle.Item.Lower+"/trades", params), &histories)
}

// GetDeliveryFuturesCandlesticks retrieves specified contract candlesticks
// A limit returns the most recent candles, so Gate rejects one sent alongside a from or to time
func (e *Exchange) GetDeliveryFuturesCandlesticks(ctx context.Context, settle currency.Code, contract currency.Pair, from, to time.Time, limit uint64, interval kline.Interval) ([]*FuturesCandlestickResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	if limit > 0 && (!from.IsZero() || !to.IsZero()) {
		return nil, errLimitWithTimeRange
	}
	params := url.Values{}
	if int64(interval) != 0 {
		intervalString, err := getIntervalString(interval)
		if err != nil {
			return nil, err
		}
		params.Set("interval", intervalString)
	}
	params.Set("contract", contract.Upper().String())
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var candlesticks []*FuturesCandlestickResponse
	return candlesticks, e.SendHTTPRequest(ctx, exchange.RestSpot, publicCandleSticksDeliveryEPL, common.EncodeURLValues("delivery/"+settle.Item.Lower+"/candlesticks", params), &candlesticks)
}

// GetDeliveryFutureTickers retrieves futures ticker information for a specific settle and contract info.
func (e *Exchange) GetDeliveryFutureTickers(ctx context.Context, settle currency.Code, contract currency.Pair) ([]*FuturesTickerResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	var tickers []*FuturesTickerResponse
	return tickers, e.SendHTTPRequest(ctx, exchange.RestSpot, publicTickersDeliveryEPL, common.EncodeURLValues("delivery/"+settle.Item.Lower+"/tickers", params), &tickers)
}

// GetDeliveryInsuranceBalanceHistory retrieves delivery futures insurance balance history
func (e *Exchange) GetDeliveryInsuranceBalanceHistory(ctx context.Context, settle currency.Code, limit uint64) ([]*InsuranceBalanceResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var balances []*InsuranceBalanceResponse
	return balances, e.SendHTTPRequest(ctx, exchange.RestSpot, publicInsuranceDeliveryEPL, common.EncodeURLValues("delivery/"+settle.Item.Lower+"/insurance", params), &balances)
}

// GetDeliveryFuturesAccounts retrieves futures account
func (e *Exchange) GetDeliveryFuturesAccounts(ctx context.Context, settle currency.Code) (*FuturesAccountResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	var response *FuturesAccountResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryAccountEPL, http.MethodGet, "delivery/"+settle.Item.Lower+"/accounts", nil, nil, &response)
}

// GetDeliveryAccountBooks retrieves account books
func (e *Exchange) GetDeliveryAccountBooks(ctx context.Context, settle currency.Code, limit uint64, from, to time.Time, changingType string) ([]*AccountBookItemResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	if changingType != "" {
		params.Set("type", changingType)
	}
	var response []*AccountBookItemResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryAccountBooksEPL, http.MethodGet, "delivery/"+settle.Item.Lower+"/account_book", params, nil, &response)
}

// GetAllDeliveryPositionsOfUser retrieves all positions of user
func (e *Exchange) GetAllDeliveryPositionsOfUser(ctx context.Context, settle currency.Code) ([]*PositionResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	var response []*PositionResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryPositionsEPL, http.MethodGet, "delivery/"+settle.Item.Lower+"/positions", nil, nil, &response)
}

// GetSingleDeliveryPosition get single position
func (e *Exchange) GetSingleDeliveryPosition(ctx context.Context, settle currency.Code, contract currency.Pair) (*PositionResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	var response *PositionResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryPositionsEPL, http.MethodGet, "delivery/"+settle.Item.Lower+"/positions/"+contract.String(), nil, nil, &response)
}

// UpdateDeliveryPositionMargin updates position margin, adding margin for a positive change and removing it for a negative one
func (e *Exchange) UpdateDeliveryPositionMargin(ctx context.Context, settle currency.Code, change float64, contract currency.Pair) (*PositionResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	if change == 0 {
		return nil, errInvalidMarginChange
	}
	params := url.Values{}
	params.Set("change", strconv.FormatFloat(change, 'f', -1, 64))
	var response *PositionResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryUpdateMarginEPL, http.MethodPost, "delivery/"+settle.Item.Lower+"/positions/"+contract.String()+"/margin", params, nil, &response)
}

// UpdateDeliveryPositionLeverage updates position leverage
func (e *Exchange) UpdateDeliveryPositionLeverage(ctx context.Context, settle currency.Code, contract currency.Pair, leverage float64) (*PositionResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	if leverage < 0 {
		return nil, order.ErrSubmitLeverageNotSupported
	}
	params := url.Values{}
	params.Set("leverage", strconv.FormatFloat(leverage, 'f', -1, 64))
	var response *PositionResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx,
		exchange.RestSpot, deliveryUpdateLeverageEPL, http.MethodPost, "delivery/"+settle.Item.Lower+"/positions/"+contract.String()+"/leverage", params, nil, &response)
}

// PlaceDeliveryOrder create a futures order
// Zero-filled order cannot be retrieved 10 minutes after order cancellation
func (e *Exchange) PlaceDeliveryOrder(ctx context.Context, arg *DeliveryOrderCreateRequest) (*FuturesOrderResponse, error) {
	if err := arg.validate(true); err != nil {
		return nil, err
	}
	var response *FuturesOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliverySubmitOrderEPL, http.MethodPost, "delivery/"+arg.Settle.Item.Lower+"/orders", nil, arg, &response)
}

// GetDeliveryOrders list futures orders
// Zero-filled order cannot be retrieved 10 minutes after order cancellation
func (e *Exchange) GetDeliveryOrders(ctx context.Context, contract currency.Pair, status string, settle currency.Code, lastID string, limit, offset uint64, countTotal bool) ([]*FuturesOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if status != statusOpen && status != statusFinished {
		return nil, fmt.Errorf("%w, only 'open' and 'finished' status are supported", errInvalidOrderStatus)
	}
	params := url.Values{}
	params.Set("status", status)
	if !contract.IsEmpty() {
		params.Set("contract", contract.String())
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if lastID != "" {
		params.Set("last_id", lastID)
	}
	if countTotal && status != statusOpen {
		params.Set("count_total", "1")
	}
	var response []*FuturesOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryGetOrdersEPL, http.MethodGet, "delivery/"+settle.Item.Lower+"/orders", params, nil, &response)
}

// CancelMultipleDeliveryOrders cancel all open orders matched
// Zero-filled order cannot be retrieved 10 minutes after order cancellation
func (e *Exchange) CancelMultipleDeliveryOrders(ctx context.Context, contract currency.Pair, side string, settle currency.Code) ([]*FuturesOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsInvalid() {
		return nil, fmt.Errorf("%w, currency pair for contract must not be empty", currency.ErrCurrencyPairEmpty)
	}
	// Dropping an unrecognised side would widen the cancellation to both sides of the book
	if side != "" && side != order.Ask.Lower() && side != order.Bid.Lower() {
		return nil, fmt.Errorf("%w: %q; only 'ask' and 'bid' are supported", order.ErrSideIsInvalid, side)
	}
	params := url.Values{}
	if side != "" {
		params.Set("side", side)
	}
	params.Set("contract", contract.String())
	var response []*FuturesOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryCancelOrdersEPL, http.MethodDelete, "delivery/"+settle.Item.Lower+"/orders", params, nil, &response)
}

// GetSingleDeliveryOrder Get a single order
// Zero-filled order cannot be retrieved 10 minutes after order cancellation
func (e *Exchange) GetSingleDeliveryOrder(ctx context.Context, settle currency.Code, orderID string) (*FuturesOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if orderID == "" {
		return nil, fmt.Errorf("%w, 'order_id' cannot be empty", order.ErrOrderIDNotSet)
	}
	var response *FuturesOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryGetOrderEPL, http.MethodGet, "delivery/"+settle.Item.Lower+"/orders/"+orderID, nil, nil, &response)
}

// CancelSingleDeliveryOrder cancel a single order
func (e *Exchange) CancelSingleDeliveryOrder(ctx context.Context, settle currency.Code, orderID string) (*FuturesOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if orderID == "" {
		return nil, fmt.Errorf("%w, 'order_id' cannot be empty", order.ErrOrderIDNotSet)
	}
	var response *FuturesOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryCancelOrderEPL, http.MethodDelete, "delivery/"+settle.Item.Lower+"/orders/"+orderID, nil, nil, &response)
}

// GetMyDeliveryTradingHistory retrieves authenticated account delivery futures trading history
func (e *Exchange) GetMyDeliveryTradingHistory(ctx context.Context, settle currency.Code, orderID string, contract currency.Pair, limit, offset, countTotal uint64, lastID string) ([]*TradingHistoryItemResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	if orderID != "" {
		params.Set("order", orderID)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if lastID != "" {
		params.Set("last_id", lastID)
	}
	if countTotal == 1 {
		params.Set("count_total", strconv.FormatUint(countTotal, 10))
	}
	var response []*TradingHistoryItemResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryTradingHistoryEPL, http.MethodGet, "delivery/"+settle.Item.Lower+"/my_trades", params, nil, &response)
}

// GetDeliveryPositionCloseHistory retrieves position history
func (e *Exchange) GetDeliveryPositionCloseHistory(ctx context.Context, settle currency.Code, contract currency.Pair, limit uint64) ([]*PositionCloseHistoryResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var response []*PositionCloseHistoryResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryCloseHistoryEPL, http.MethodGet, "delivery/"+settle.Item.Lower+"/position_close", params, nil, &response)
}

// GetDeliveryLiquidationHistory lists liquidation history
func (e *Exchange) GetDeliveryLiquidationHistory(ctx context.Context, settle currency.Code, contract currency.Pair, limit uint64, at time.Time) ([]*LiquidationHistoryItemResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if !at.IsZero() {
		params.Set("at", strconv.FormatInt(at.Unix(), 10))
	}
	var response []*LiquidationHistoryItemResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryLiquidationHistoryEPL, http.MethodGet, "delivery/"+settle.Item.Lower+"/liquidates", params, nil, &response)
}

// GetDeliverySettlementHistory retrieves settlement history
func (e *Exchange) GetDeliverySettlementHistory(ctx context.Context, settle currency.Code, contract currency.Pair, limit uint64, at time.Time) ([]*SettlementHistoryItemResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if !at.IsZero() {
		params.Set("at", strconv.FormatInt(at.Unix(), 10))
	}
	var response []*SettlementHistoryItemResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliverySettlementHistoryEPL, http.MethodGet, "delivery/"+settle.Item.Lower+"/settlements", params, nil, &response)
}

// CreatePriceTriggeredDeliveryOrder creates a price-triggered order
// An initial size of zero closes the whole position and an initial price of zero places a market order
func (e *Exchange) CreatePriceTriggeredDeliveryOrder(ctx context.Context, settle currency.Code, arg *FuturesPriceTriggeredOrderRequest) (*TriggerOrderIDResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if err := arg.validate(); err != nil {
		return nil, err
	}
	var response *TriggerOrderIDResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliverySubmitTriggerOrderEPL, http.MethodPost, "delivery/"+settle.Item.Lower+"/price_orders", nil, arg, &response)
}

// GetDeliveryAllAutoOrder retrieves all auto orders
func (e *Exchange) GetDeliveryAllAutoOrder(ctx context.Context, status string, settle currency.Code, contract currency.Pair, limit, offset uint64) ([]*PriceTriggeredOrderResponse, error) {
	if status != statusOpen && status != statusFinished {
		return nil, fmt.Errorf("%w status %s", errInvalidOrderStatus, status)
	}
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	params.Set("status", status)
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	var response []*PriceTriggeredOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryAutoOrdersEPL, http.MethodGet, "delivery/"+settle.Item.Lower+"/price_orders", params, nil, &response)
}

// CancelAllDeliveryPriceTriggeredOrder cancels all delivery price triggered orders
func (e *Exchange) CancelAllDeliveryPriceTriggeredOrder(ctx context.Context, settle currency.Code, contract currency.Pair) ([]*PriceTriggeredOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	params := url.Values{}
	params.Set("contract", contract.String())
	var response []*PriceTriggeredOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryCancelTriggerOrdersEPL, http.MethodDelete, "delivery/"+settle.Item.Lower+"/price_orders", params, nil, &response)
}

// GetSingleDeliveryPriceTriggeredOrder retrieves a single price triggered order
func (e *Exchange) GetSingleDeliveryPriceTriggeredOrder(ctx context.Context, settle currency.Code, orderID string) (*PriceTriggeredOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var response *PriceTriggeredOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryGetTriggerOrderEPL, http.MethodGet, "delivery/"+settle.Item.Lower+"/price_orders/"+orderID, nil, nil, &response)
}

// CancelDeliveryPriceTriggeredOrder cancel a price-triggered order
func (e *Exchange) CancelDeliveryPriceTriggeredOrder(ctx context.Context, settle currency.Code, orderID string) (*PriceTriggeredOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var response *PriceTriggeredOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryCancelTriggerOrderEPL, http.MethodDelete, "delivery/"+settle.Item.Lower+"/price_orders/"+orderID, nil, nil, &response)
}

// validate validates the DeliveryOrderCreateRequest
func (c *DeliveryOrderCreateRequest) validate(isRest bool) error {
	if err := common.NilGuard(c); err != nil {
		return err
	}
	return validateOrderCreateParams(c.Contract, c.Size, c.Price, c.ClosePosition, c.AutoSize, c.TimeInForce, c.Text, c.Settle, isRest)
}
