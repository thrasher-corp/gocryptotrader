package gateio

import (
	"context"
	"errors"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
)

var (
	errInvalidAutoSize            = errors.New("invalid auto size")
	errSettlementCurrencyConflict = errors.New("settlement currency conflict")
	errStatusNotSet               = errors.New("status not set")
)

// authenticateFutures sends an authentication message to the websocket connection
func (g *Gateio) authenticateFutures(ctx context.Context, conn stream.Connection) error {
	return g.websocketLogin(ctx, conn, "futures.login")
}

// WebsocketFuturesSubmitOrder submits an order via the websocket connection
func (g *Gateio) WebsocketFuturesSubmitOrder(ctx context.Context, order *ContractOrderCreateParams) ([]WebsocketFuturesOrderResponse, error) {
	return g.WebsocketFuturesSubmitOrders(ctx, order)
}

// WebsocketFuturesSubmitOrders places an order via the websocket connection. You can
// send multiple orders in a single request. NOTE: When sending multiple orders
// the response will be an array of responses and a succeeded bool will be
// returned in the response.
func (g *Gateio) WebsocketFuturesSubmitOrders(ctx context.Context, orders ...*ContractOrderCreateParams) ([]WebsocketFuturesOrderResponse, error) {
	if len(orders) == 0 {
		return nil, errOrdersEmpty
	}

	var a asset.Item
	for i := range orders {
		if orders[i].Contract.IsEmpty() {
			return nil, currency.ErrCurrencyPairEmpty
		}

		if orders[i].Price == "" && orders[i].TimeInForce != "ioc" {
			return nil, fmt.Errorf("%w: cannot be zero when time in force is not IOC", errInvalidPrice)
		}

		if orders[i].Size == 0 && orders[i].AutoSize == "" {
			return nil, fmt.Errorf("%w: size cannot be zero", errInvalidAmount)
		}

		if orders[i].AutoSize != "" {
			if orders[i].AutoSize != "close_long" && orders[i].AutoSize != "close_short" {
				return nil, fmt.Errorf("%w: %s", errInvalidAutoSize, orders[i].AutoSize)
			}
			if orders[i].Size != 0 {
				return nil, fmt.Errorf("%w: size needs to be zero when auto size is set", errInvalidAmount)
			}
		}

		switch {
		case orders[i].Contract.Quote.Equal(currency.USDT):
			if a != asset.Empty && a != asset.USDTMarginedFutures {
				return nil, fmt.Errorf("%w: either btc or usdt margined can only be batched as they are using different connections", errSettlementCurrencyConflict)
			}
			a = asset.USDTMarginedFutures
		case orders[i].Contract.Quote.Equal(currency.USD):
			if a != asset.Empty && a != asset.CoinMarginedFutures {
				return nil, fmt.Errorf("%w: either btc or usdt margined can only be batched as they are using different connections", errSettlementCurrencyConflict)
			}
			a = asset.CoinMarginedFutures
		}
	}

	if len(orders) == 1 {
		var singleResponse WebsocketFuturesOrderResponse
		err := g.SendWebsocketRequest(ctx, perpetualSubmitOrderEPL, "futures.order_place", a, orders[0], &singleResponse, 2)
		return []WebsocketFuturesOrderResponse{singleResponse}, err
	}

	var resp []WebsocketFuturesOrderResponse
	return resp, g.SendWebsocketRequest(ctx, perpetualSubmitBatchOrdersEPL, "futures.order_batch_place", a, orders, &resp, 2)
}

// WebsocketFuturesCancelOrder cancels an order via the websocket connection.
func (g *Gateio) WebsocketFuturesCancelOrder(ctx context.Context, orderID string, contract currency.Pair) (*WebsocketFuturesOrderResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}

	if contract.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}

	a := asset.USDTMarginedFutures
	if contract.Quote.Equal(currency.USD) {
		a = asset.CoinMarginedFutures
	}

	params := &struct {
		OrderID string `json:"order_id"`
	}{OrderID: orderID}

	var resp WebsocketFuturesOrderResponse
	return &resp, g.SendWebsocketRequest(ctx, perpetualCancelOrderEPL, "futures.order_cancel", a, params, &resp, 1)
}

// WebsocketFuturesCancelAllOpenFuturesOrders cancels multiple orders via the websocket.
func (g *Gateio) WebsocketFuturesCancelAllOpenFuturesOrders(ctx context.Context, contract currency.Pair, side string) ([]WebsocketFuturesOrderResponse, error) {
	if contract.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}

	if side != "" && side != "ask" && side != "bid" {
		return nil, fmt.Errorf("%w: %s", order.ErrSideIsInvalid, side)
	}

	params := struct {
		Contract currency.Pair `json:"contract"`
		Side     string        `json:"side,omitempty"`
	}{Contract: contract, Side: side}

	a := asset.USDTMarginedFutures
	if contract.Quote.Equal(currency.USD) {
		a = asset.CoinMarginedFutures
	}

	var resp []WebsocketFuturesOrderResponse
	return resp, g.SendWebsocketRequest(ctx, perpetualCancelOpenOrdersEPL, "futures.order_cancel_cp", a, params, &resp, 2)
}

// WebsocketFuturesAmendOrder amends an order via the websocket connection
func (g *Gateio) WebsocketFuturesAmendOrder(ctx context.Context, amend *WebsocketFuturesAmendOrder) (*WebsocketFuturesOrderResponse, error) {
	if amend == nil {
		return nil, fmt.Errorf("%w: %T", common.ErrNilPointer, amend)
	}

	if amend.OrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}

	if amend.Contract.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}

	if amend.Size == 0 && amend.Price == "" {
		return nil, fmt.Errorf("%w: size or price must be set", errInvalidAmount)
	}

	a := asset.USDTMarginedFutures
	if amend.Contract.Quote.Equal(currency.USD) {
		a = asset.CoinMarginedFutures
	}

	var resp WebsocketFuturesOrderResponse
	return &resp, g.SendWebsocketRequest(ctx, perpetualAmendOrderEPL, "futures.order_amend", a, amend, &resp, 1)
}

// WebsocketFuturesOrderList fetches a list of orders via the websocket connection
func (g *Gateio) WebsocketFuturesOrderList(ctx context.Context, list *WebsocketFutureOrdersList) ([]WebsocketFuturesOrderResponse, error) {
	if list == nil {
		return nil, fmt.Errorf("%w: %T", common.ErrNilPointer, list)
	}

	if list.Contract.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}

	if list.Status == "" {
		return nil, errStatusNotSet
	}

	a := asset.USDTMarginedFutures
	if list.Contract.Quote.Equal(currency.USD) {
		a = asset.CoinMarginedFutures
	}

	var resp []WebsocketFuturesOrderResponse
	return resp, g.SendWebsocketRequest(ctx, perpetualGetOrdersEPL, "futures.order_list", a, list, &resp, 1)
}

// WebsocketFuturesGetOrderStatus gets the status of an order via the websocket connection.
func (g *Gateio) WebsocketFuturesGetOrderStatus(ctx context.Context, contract currency.Pair, orderID string) (*WebsocketFuturesOrderResponse, error) {
	if contract.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}

	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}

	params := &struct {
		OrderID string `json:"order_id"`
	}{OrderID: orderID}

	a := asset.USDTMarginedFutures
	if contract.Quote.Equal(currency.USD) {
		a = asset.CoinMarginedFutures
	}

	var resp WebsocketFuturesOrderResponse
	return &resp, g.SendWebsocketRequest(ctx, perpetualFetchOrderEPL, "futures.order_status", a, params, &resp, 1)
}
