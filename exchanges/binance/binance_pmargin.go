package binance

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// Definitions and Terminology
// Portfolio Margin is an advanced trading mode offered by Binance, designed for experienced traders who seek
// increased leverage and flexibility across various trading products. It incorporates a unique approach to margin
// calculations and risk management to offer a more comprehensive assessment of the trader's overall exposure.

// - Terminology
// Margin refers to Cross Margin
// UM refers to USD-M Futures
// CM refers to Coin-M Futures

// NewUMOrder send in a new USDT margined order/orders.
func (b *Binance) NewUMOrder(ctx context.Context, arg *UMOrderParam) (*UMOrder, error) {
	return b.newUMCMOrder(ctx, arg, "/papi/v1/um/order")
}

// NewCMOrder send in a new Coin margined order/orders.
func (b *Binance) NewCMOrder(ctx context.Context, arg *UMOrderParam) (*UMOrder, error) {
	return b.newUMCMOrder(ctx, arg, "/papi/v1/cm/order")
}

func (b *Binance) newUMCMOrder(ctx context.Context, arg *UMOrderParam, path string) (*UMOrder, error) {
	if arg == nil || (*arg) == (UMOrderParam{}) {
		return nil, common.ErrNilPointer
	}
	if arg.Symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.OrderType == "" {
		return nil, order.ErrTypeIsInvalid
	}
	arg.OrderType = strings.ToUpper(arg.OrderType)
	if arg.OrderType == "limit" {
		if arg.TimeInForce == "" {
			return nil, errTimestampInfoRequired
		}
		if arg.Quantity <= 0 {
			return nil, order.ErrAmountBelowMin
		}
		if arg.Price <= 0 {
			return nil, order.ErrPriceBelowMin
		}
	} else if arg.OrderType == "MARKET" {
		if arg.Quantity <= 0 {
			return nil, order.ErrAmountBelowMin
		}
	} else {
		return nil, order.ErrUnsupportedOrderType
	}
	params := url.Values{}
	params.Set("symbol", arg.Symbol)
	params.Set("side", arg.Side)
	params.Set("type", arg.OrderType)
	if arg.PositionSide != "" {
		params.Set("positionSide", arg.PositionSide)
	}
	if arg.TimeInForce != "" {
		params.Set("timeInForce", arg.TimeInForce)
	}
	if arg.Quantity > 0 {
		params.Set("quantity", strconv.FormatFloat(arg.Quantity, 'f', -1, 64))
	}
	if arg.ReduceOnly {
		params.Set("reduceOnly", "true")
	}
	if arg.Price > 0 {
		params.Set("price", strconv.FormatFloat(arg.Price, 'f', -1, 64))
	}
	if arg.NewClientOrderID != "" {
		params.Set("newClientOrderID", arg.NewClientOrderID)
	}
	if arg.NewOrderRespType != "" {
		params.Set("newOrderRespType", arg.NewOrderRespType)
	}
	if arg.SelfTradePreventionMode != "" {
		params.Set("selfTradePreventionMode", arg.SelfTradePreventionMode)
	}
	if arg.GoodTillDate > 0 {
		params.Set("goodTillDate", strconv.FormatInt(arg.GoodTillDate, 10))
	}
	var resp *UMOrder
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodPost, path, params, spotDefaultRate, &resp)
}

// NewMarginOrder places a new cross margin order
func (b *Binance) NewMarginOrder(ctx context.Context, arg *MarginOrderParam) (*MarginOrderResp, error) {
	if arg == nil || *arg == (MarginOrderParam{}) {
		return nil, common.ErrNilPointer
	}
	if arg.Symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}

	if arg.OrderType == "" {
		return nil, order.ErrTypeIsInvalid
	}
	params := url.Values{}
	params.Set("symbol", arg.Symbol)
	params.Set("side", strings.ToUpper(arg.Side))
	params.Set("type", arg.OrderType)
	if arg.Amount > 0 {
		params.Set("quantity", strconv.FormatFloat(arg.Amount, 'f', -1, 64))
	}
	if arg.QuoteOrderQty > 0 {
		params.Set("quoteOrderQty", strconv.FormatFloat(arg.QuoteOrderQty, 'f', -1, 64))
	}
	if arg.Price > 0 {
		params.Set("price", strconv.FormatFloat(arg.Price, 'f', -1, 64))
	}
	if arg.StopPrice > 0 {
		params.Set("stopPrice", strconv.FormatFloat(arg.StopPrice, 'f', -1, 64))
	}
	if arg.NewClientOrderID != "" {
		params.Set("newClientOrderId", arg.NewClientOrderID)
	}
	if arg.NewOrderRespType != "" {
		params.Set("newOrderRespType", arg.NewOrderRespType)
	}
	var resp *MarginOrderResp
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodPost, "/papi/v1/margin/order", params, spotDefaultRate, &resp)
}

// MarginAccountBorrow apply for margin loan
func (b *Binance) MarginAccountBorrow(ctx context.Context, ccy currency.Code, amount float64) (string, error) {
	return b.marginAccountBorrowRepay(ctx, ccy, amount, "/papi/v1/marginLoan")
}

// MarginAccountRepay repay for margin loan
func (b *Binance) MarginAccountRepay(ctx context.Context, ccy currency.Code, amount float64) (string, error) {
	return b.marginAccountBorrowRepay(ctx, ccy, amount, "/papi/v1/repayLoan")
}

func (b *Binance) marginAccountBorrowRepay(ctx context.Context, ccy currency.Code, amount float64, path string) (string, error) {
	if ccy.IsEmpty() {
		return "", currency.ErrCurrencyCodeEmpty
	}
	if amount <= 0 {
		return "", order.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("asset", ccy.String())
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	resp := &struct {
		TransactionID string `json:"tranId"`
	}{}
	return resp.TransactionID, b.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodPost, path, params, spotDefaultRate, &resp)
}

// MarginAccountNewOCO sends a new OCO order for a margin account.
func (b *Binance) MarginAccountNewOCO(ctx context.Context, arg *OCOOrderParam) (*OCOOrderResponse, error) {
	if arg == nil || *arg == (OCOOrderParam{}) {
		return nil, common.ErrNilPointer
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.Amount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	if arg.Price <= 0 {
		return nil, order.ErrPriceBelowMin
	}
	if arg.StopPrice <= 0 {
		return nil, fmt.Errorf("%w, stopPrice is required", order.ErrPriceBelowMin)
	}
	params := url.Values{}
	params.Set("symbol", arg.Symbol.String())
	params.Set("listClientOrderId", arg.ListClientOrderID)
	params.Set("side", arg.Side)
	params.Set("quantity", strconv.FormatFloat(arg.Amount, 'f', -1, 64))
	if arg.LimitClientOrderID != "" {
		params.Set("limitClientOrderId", arg.LimitClientOrderID)
	}
	if arg.LimitIcebergQuantity > 0 {
		params.Set("limitIcebergQty", strconv.FormatFloat(arg.LimitIcebergQuantity, 'f', -1, 64))
	}
	if arg.StopClientOrderID != "" {
		params.Set("stopClientOrderId", arg.StopClientOrderID)
	}
	if arg.StopLimitPrice > 0 {
		params.Set("stopLimitPrice", strconv.FormatFloat(arg.StopLimitPrice, 'f', -1, 64))
	}
	if arg.StopIcebergQuantity > 0 {
		params.Set("stopIcebergQty", strconv.FormatFloat(arg.StopIcebergQuantity, 'f', -1, 64))
	}
	if arg.StopLimitTimeInForce != "" {
		params.Set("stopLimitTimeInForce", arg.StopLimitTimeInForce)
	}
	if arg.NewOrderRespType != "" {
		params.Set("newOrderRespType", arg.NewOrderRespType)
	}
	if arg.SideEffectType != "" {
		params.Set("sideEffectType", arg.SideEffectType)
	}
	if arg.SelfTradePreventionMode != "" {
		params.Set("selfTradePreventionMode", arg.SelfTradePreventionMode)
	}
	var resp *OCOOrderResponse
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodPost, "/papi/v1/margin/order/oco", params, spotDefaultRate, &resp)
}

// NewUMConditionalOrder places a new conditional USDT margined order
func (b *Binance) NewUMConditionalOrder(ctx context.Context, arg *UMConditionalOrderParam) (*UMConditionalOrder, error) {
	if arg == nil || *arg == (UMConditionalOrderParam{}) {
		return nil, common.ErrNilPointer
	}
	if arg.Symbol == "" {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	params := url.Values{}
	params.Set("symbol", arg.Symbol)
	params.Set("side", arg.Side)
	params.Set("strategyType", arg.StrategyType)
	if arg.PositionSide != "" {
		params.Set("positionSide", arg.PositionSide)
	}
	if arg.TimeInForce != "" {
		params.Set("timeInForce", arg.TimeInForce)
	}
	if arg.Quantity > 0 {
		params.Set("quantity", strconv.FormatFloat(arg.Quantity, 'f', -1, 64))
	}
	if arg.ReduceOnly {
		params.Set("reduceOnly", "true")
	}
	if arg.Price > 0 {
		params.Set("price", strconv.FormatFloat(arg.Price, 'f', -1, 64))
	}
	if arg.WorkingType != "" {
		params.Set("workingType", arg.WorkingType)
	}
	if arg.PriceProtect {
		params.Set("priceProtect", "true")
	}
	if arg.NewClientStrategyID != "" {
		params.Set("newClientStrategyId", arg.NewClientStrategyID)
	}
	if arg.StopPrice > 0 {
		params.Set("stopPrice", strconv.FormatFloat(arg.StopPrice, 'f', -1, 64))
	}
	if arg.ActivationPrice > 0 {
		params.Set("activationPrice", strconv.FormatFloat(arg.ActivationPrice, 'f', -1, 64))
	}
	if arg.CallbackRate > 0 {
		params.Set("callbackRate", strconv.FormatFloat(arg.CallbackRate, 'f', -1, 64))
	}
	if arg.SelfTradePreventionMode != "" {
		params.Set("selfTradePreventionMode", arg.SelfTradePreventionMode)
	}
	if arg.GoodTillDate > 0 {
		params.Set("goodTillDate", strconv.FormatInt(arg.GoodTillDate, 10))
	}
	var resp *UMConditionalOrder
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodPost, "/papi/v1/um/conditional/order", params, spotDefaultRate, &resp)
}
