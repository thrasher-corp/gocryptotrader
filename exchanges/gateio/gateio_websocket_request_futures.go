package gateio

import (
	"context"
	"errors"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var errStatusNotSet = errors.New("status not set")

// authenticateFutures sends an authentication message to the websocket connection
func (e *Exchange) authenticateFutures(ctx context.Context, conn websocket.Connection) error {
	return e.websocketLogin(ctx, conn, "futures.login")
}

// WebsocketFuturesSubmitOrder submits an order via the websocket connection
func (e *Exchange) WebsocketFuturesSubmitOrder(ctx context.Context, a asset.Item, o *ContractOrderCreateParams) (*WebsocketFuturesOrderResponse, error) {
	resps, err := e.WebsocketFuturesSubmitOrders(ctx, a, o)
	if err != nil {
		return nil, err
	}
	if len(resps) != 1 {
		return nil, common.ErrInvalidResponse
	}
	return resps[0], err
}

// WebsocketFuturesSubmitOrders submits orders via the websocket connection. All orders must be for the same asset.
func (e *Exchange) WebsocketFuturesSubmitOrders(ctx context.Context, a asset.Item, orders ...*ContractOrderCreateParams) ([]*WebsocketFuturesOrderResponse, error) {
	if len(orders) == 0 {
		return nil, errOrdersEmpty
	}

	for _, o := range orders {
		if err := o.validate(false); err != nil {
			return nil, err
		}
		if _, err := getSettlementCurrency(o.Contract, a); err != nil {
			return nil, err
		}
	}

	if len(orders) == 1 {
		var singleResponse *WebsocketFuturesOrderResponse
		err := e.SendWebsocketRequest(ctx, perpetualSubmitOrderEPL, "futures.order_place", a, orders[0], &singleResponse, 2)
		return []*WebsocketFuturesOrderResponse{singleResponse}, err
	}

	var resp []*WebsocketFuturesOrderResponse
	return resp, e.SendWebsocketRequest(ctx, perpetualSubmitBatchOrdersEPL, "futures.order_batch_place", a, orders, &resp, 2)
}

// WebsocketFuturesCancelOrder cancels an order via the websocket connection.
func (e *Exchange) WebsocketFuturesCancelOrder(ctx context.Context, orderID string, contract currency.Pair, a asset.Item) (*WebsocketFuturesOrderResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}

	if err := validateFuturesPairAsset(contract, a); err != nil {
		return nil, err
	}

	params := &struct {
		OrderID string `json:"order_id"`
	}{OrderID: orderID}

	var resp WebsocketFuturesOrderResponse
	return &resp, e.SendWebsocketRequest(ctx, perpetualCancelOrderEPL, "futures.order_cancel", a, params, &resp, 1)
}

// WebsocketFuturesCancelAllOpenFuturesOrders cancels multiple orders via the websocket.
func (e *Exchange) WebsocketFuturesCancelAllOpenFuturesOrders(ctx context.Context, contract currency.Pair, a asset.Item, side string) ([]WebsocketFuturesOrderResponse, error) {
	if err := validateFuturesPairAsset(contract, a); err != nil {
		return nil, err
	}

	if side != "" && side != order.Ask.Lower() && side != order.Bid.Lower() {
		return nil, fmt.Errorf("%w: %s", order.ErrSideIsInvalid, side)
	}

	params := &struct {
		Contract currency.Pair `json:"contract"`
		Side     string        `json:"side,omitempty"`
	}{Contract: contract, Side: side}

	var resp []WebsocketFuturesOrderResponse
	return resp, e.SendWebsocketRequest(ctx, perpetualCancelOpenOrdersEPL, "futures.order_cancel_cp", a, params, &resp, 2)
}

// WebsocketFuturesAmendOrder amends an order via the websocket connection
func (e *Exchange) WebsocketFuturesAmendOrder(ctx context.Context, amend *WebsocketFuturesAmendOrder) (*WebsocketFuturesOrderResponse, error) {
	if amend == nil {
		return nil, fmt.Errorf("%w: %T", common.ErrNilPointer, amend)
	}

	if amend.OrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}

	if err := validateFuturesPairAsset(amend.Contract, amend.Asset); err != nil {
		return nil, err
	}

	if amend.Size == 0 && amend.Price == "" {
		return nil, fmt.Errorf("%w: size or price must be set", errInvalidAmount)
	}

	var resp WebsocketFuturesOrderResponse
	return &resp, e.SendWebsocketRequest(ctx, perpetualAmendOrderEPL, "futures.order_amend", amend.Asset, amend, &resp, 1)
}

// WebsocketFuturesOrderList fetches a list of orders via the websocket connection
func (e *Exchange) WebsocketFuturesOrderList(ctx context.Context, list *WebsocketFutureOrdersList) ([]WebsocketFuturesOrderResponse, error) {
	if list == nil {
		return nil, fmt.Errorf("%w: %T", common.ErrNilPointer, list)
	}

	if err := validateFuturesPairAsset(list.Contract, list.Asset); err != nil {
		return nil, err
	}

	if list.Status == "" {
		return nil, errStatusNotSet
	}

	var resp []WebsocketFuturesOrderResponse
	return resp, e.SendWebsocketRequest(ctx, perpetualGetOrdersEPL, "futures.order_list", list.Asset, list, &resp, 1)
}

// WebsocketFuturesGetOrderStatus gets the status of an order via the websocket connection.
func (e *Exchange) WebsocketFuturesGetOrderStatus(ctx context.Context, contract currency.Pair, a asset.Item, orderID string) (*WebsocketFuturesOrderResponse, error) {
	if err := validateFuturesPairAsset(contract, a); err != nil {
		return nil, err
	}

	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}

	params := &struct {
		OrderID string `json:"order_id"`
	}{OrderID: orderID}

	var resp WebsocketFuturesOrderResponse
	return &resp, e.SendWebsocketRequest(ctx, perpetualFetchOrderEPL, "futures.order_status", a, params, &resp, 1)
}

// validateFuturesPairAsset enforces that a futures pair's quote currency matches the given asset
func validateFuturesPairAsset(pair currency.Pair, a asset.Item) error {
	if pair.IsEmpty() {
		return currency.ErrCurrencyPairEmpty
	}
	_, err := getSettlementCurrency(pair, a)
	return err
}
