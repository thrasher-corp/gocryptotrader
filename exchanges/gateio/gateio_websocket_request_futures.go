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
	errInvalidSide                = errors.New("invalid side")
	errStatusNotSet               = errors.New("status not set")
)

// AuthenticateFutures sends an authentication message to the websocket connection
func (g *Gateio) authenticateFutures(ctx context.Context, conn stream.Connection) error {
	_, err := g.websocketLogin(ctx, conn, "futures.login")
	return err
}

// WebsocketOrderPlaceFutures places an order via the websocket connection. You can
// send multiple orders in a single request. NOTE: When sending multiple orders
// the response will be an array of responses and a succeeded bool will be
// returned in the response.
func (g *Gateio) WebsocketOrderPlaceFutures(ctx context.Context, batch []OrderCreateParams) ([]WebsocketFuturesOrderResponse, error) {
	if len(batch) == 0 {
		return nil, errBatchSliceEmpty
	}

	var a asset.Item
	for i := range batch {
		if batch[i].Contract.IsEmpty() {
			return nil, currency.ErrCurrencyPairEmpty
		}

		if batch[i].Price == "" && batch[i].TimeInForce != "ioc" {
			return nil, fmt.Errorf("%w: cannot be zero when time in force is not IOC", errInvalidPrice)
		}

		if batch[i].Size == 0 && batch[i].AutoSize == "" {
			return nil, fmt.Errorf("%w: size cannot be zero", errInvalidAmount)
		}

		if batch[i].AutoSize != "" {
			if batch[i].AutoSize != "close_long" && batch[i].AutoSize != "close_short" {
				return nil, fmt.Errorf("%w: %s", errInvalidAutoSize, batch[i].AutoSize)
			}
			if batch[i].Size != 0 {
				return nil, fmt.Errorf("%w: size needs to be zero when auto size is set", errInvalidAmount)
			}
		}

		switch {
		case batch[i].Contract.Quote.Equal(currency.USDT):
			if a != asset.Empty && a != asset.USDTMarginedFutures {
				return nil, fmt.Errorf("%w: either btc or usdt margined can only be batched as they are using different connections", errSettlementCurrencyConflict)
			}
			a = asset.USDTMarginedFutures
		case batch[i].Contract.Quote.Equal(currency.USD):
			if a != asset.Empty && a != asset.CoinMarginedFutures {
				return nil, fmt.Errorf("%w: either btc or usdt margined can only be batched as they are using different connections", errSettlementCurrencyConflict)
			}
			a = asset.CoinMarginedFutures
		}
	}

	if len(batch) == 1 {
		var singleResponse WebsocketFuturesOrderResponse
		err := g.SendWebsocketRequest(ctx, "futures.order_place", a, batch[0], &singleResponse, 2)
		return []WebsocketFuturesOrderResponse{singleResponse}, err
	}

	var resp []WebsocketFuturesOrderResponse
	return resp, g.SendWebsocketRequest(ctx, "futures.order_batch_place", a, batch, &resp, 2)
}

// WebsocketOrderCancelFutures cancels an order via the websocket connection.
// Contract is used for routing the request internally to the correct connection.
func (g *Gateio) WebsocketOrderCancelFutures(ctx context.Context, orderID string, contract currency.Pair) (*WebsocketFuturesOrderResponse, error) {
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
	return &resp, g.SendWebsocketRequest(ctx, "futures.order_cancel", a, params, &resp, 1)
}

// WebsocketOrderCancelAllOpenFuturesOrdersMatched cancels multiple orders via
// the websocket.
func (g *Gateio) WebsocketOrderCancelAllOpenFuturesOrdersMatched(ctx context.Context, contract currency.Pair, side string) ([]WebsocketFuturesOrderResponse, error) {
	if contract.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}

	if side != "" && side != "ask" && side != "bid" {
		return nil, fmt.Errorf("%w: %s", errInvalidSide, side)
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
	return resp, g.SendWebsocketRequest(ctx, "futures.order_cancel_cp", a, params, &resp, 2)
}

// WebsocketOrderAmendFutures amends an order via the websocket connection
func (g *Gateio) WebsocketOrderAmendFutures(ctx context.Context, amend *WebsocketFuturesAmendOrder) (*WebsocketFuturesOrderResponse, error) {
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
	return &resp, g.SendWebsocketRequest(ctx, "futures.order_amend", a, amend, &resp, 1)
}

// WebsocketOrderListFutures fetches a list of orders via the websocket connection
func (g *Gateio) WebsocketOrderListFutures(ctx context.Context, list *WebsocketFutureOrdersList) ([]WebsocketFuturesOrderResponse, error) {
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
	return resp, g.SendWebsocketRequest(ctx, "futures.order_list", a, list, &resp, 1)
}

// WebsocketGetOrderStatusFutures gets the status of an order via the websocket
// connection.
func (g *Gateio) WebsocketGetOrderStatusFutures(ctx context.Context, contract currency.Pair, orderID string) (*WebsocketFuturesOrderResponse, error) {
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
	return &resp, g.SendWebsocketRequest(ctx, "futures.order_status", a, params, &resp, 1)
}
