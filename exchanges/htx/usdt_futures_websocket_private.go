package htx

import (
	"context"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// wsHandleUSDTMarginedPrivateMessage decodes authenticated V5 USDT notifications.
func (e *Exchange) wsHandleUSDTMarginedPrivateMessage(ctx context.Context, sub *subscription.Subscription, raw []byte) error {
	if err := common.NilGuard(sub); err != nil {
		return err
	}
	switch sub.Channel {
	case subscription.MyOrdersChannel:
		response := new(V5WebsocketOrderUpdate)
		if err := json.Unmarshal(raw, response); err != nil {
			return err
		}
		return e.sendV5WSOrderUpdate(ctx, sub, response.ContractCode, &response.Order)
	case wsTradeUpdatesChannel:
		response := new(V5WebsocketTradeUpdate)
		if err := json.Unmarshal(raw, response); err != nil {
			return err
		}
		return e.sendV5WSExecutionUpdates(ctx, sub, response.ContractCode, response.Executions)
	case wsExecutionDetailsChannel:
		response := new(V5WebsocketTradeDetailUpdate)
		if err := json.Unmarshal(raw, response); err != nil {
			return err
		}
		return e.sendV5WSExecutionUpdates(ctx, sub, response.ContractCode, response.Executions)
	case wsPositionsChannel:
		response := new(V5WebsocketPositionUpdate)
		if err := json.Unmarshal(raw, response); err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, response)
	case subscription.MyAccountChannel:
		response := new(V5WebsocketAccountUpdate)
		if err := json.Unmarshal(raw, response); err != nil {
			return err
		}
		changes := make([]accounts.Change, 0, len(response.Balance.CurrencyBalances))
		for i := range response.Balance.CurrencyBalances {
			free := v5FreeMargin(&response.Balance.CurrencyBalances[i])
			changes = append(changes, accounts.Change{
				AssetType: sub.Asset,
				Balance: accounts.Balance{
					Currency:  response.Balance.CurrencyBalances[i].Currency,
					Total:     (response.Balance.CurrencyBalances[i].Equity.Float64() + response.Balance.CurrencyBalances[i].IsolatedEquity.Float64()),
					Free:      free,
					Hold:      (response.Balance.CurrencyBalances[i].Equity.Float64() + response.Balance.CurrencyBalances[i].IsolatedEquity.Float64()) - free,
					UpdatedAt: response.Timestamp.Time(),
				},
			})
		}
		if err := e.storeWebsocketBalances(ctx, sub.Asset, changes); err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, changes)
	case subscription.MyTradesChannel:
		response := new(V5WebsocketMatchOrderUpdate)
		if err := json.Unmarshal(raw, response); err != nil {
			return err
		}
		for i := range response.Orders {
			match := &response.Orders[i]
			contractCode := match.ContractCode
			if contractCode == "" {
				contractCode = response.ContractCode
			}
			match.V5WebsocketOrder.TradeVolume = match.TotalTradeVolume
			match.V5WebsocketOrder.TradeAveragePrice = match.TradePrice
			match.V5WebsocketOrder.UpdatedTime = match.MatchTime
			if err := e.sendV5WSOrderUpdate(ctx, sub, contractCode, &match.V5WebsocketOrder); err != nil {
				return err
			}
		}
		return nil
	case wsTriggerOrdersChannel:
		response := new(V5WebsocketAlgoOrderUpdate)
		if err := json.Unmarshal(raw, response); err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, response)
	default:
		return fmt.Errorf("%w: %s", common.ErrNotYetImplemented, sub.Channel)
	}
}

// sendV5WSOrderUpdate converts each V5 order-notification variant into the canonical order type.
func (e *Exchange) sendV5WSOrderUpdate(ctx context.Context, sub *subscription.Subscription, contractCode string, data *V5WebsocketOrder) error {
	detail, err := e.formatV5OrderDetail(&V5Order{
		ContractCode:      contractCode,
		Side:              data.Side,
		PositionSide:      data.PositionSide,
		Type:              data.Type,
		OrderID:           data.OrderID,
		ClientOrderID:     data.ClientOrderID,
		MarginMode:        data.MarginMode,
		Price:             data.Price,
		Volume:            data.Volume,
		LeverageRate:      types.Number(data.LeverageRate),
		State:             data.State,
		ReduceOnly:        data.ReduceOnly,
		TimeInForce:       data.TimeInForce,
		TradeAveragePrice: data.TradeAveragePrice,
		TradeVolume:       data.TradeVolume,
		TradeTurnover:     data.TradeTurnover,
		FeeCurrency:       data.FeeCurrency,
		Fee:               data.Fee,
		CreatedTime:       data.CreatedTime,
		UpdatedTime:       data.UpdatedTime,
	}, sub.Asset)
	if err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, &detail)
}

// WebsocketPlaceV5Order places an order through the dedicated V5 trade connection.
func (e *Exchange) WebsocketPlaceV5Order(ctx context.Context, req *V5OrderRequest) (*V5WebsocketOrderResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	var resp *V5WebsocketOrderResponse
	err := e.sendV5TradeRequest(ctx, "place_order", req, &resp)
	return resp, err
}

// WebsocketPlaceV5BatchOrders places multiple orders through the dedicated V5 trade connection.
func (e *Exchange) WebsocketPlaceV5BatchOrders(ctx context.Context, req []*V5OrderRequest) (*V5WebsocketBatchOrderResponse, error) {
	if len(req) == 0 {
		return nil, common.ErrEmptyParams
	}
	var resp *V5WebsocketBatchOrderResponse
	err := e.sendV5TradeRequest(ctx, "place_batch_orders", req, &resp)
	return resp, err
}

// WebsocketCancelV5Order cancels an order through the dedicated V5 trade connection.
func (e *Exchange) WebsocketCancelV5Order(ctx context.Context, req *V5CancelOrderRequest) (*V5WebsocketOrderResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	var resp *V5WebsocketOrderResponse
	err := e.sendV5TradeRequest(ctx, "cancel_order", req, &resp)
	return resp, err
}

// WebsocketCancelV5BatchOrders cancels multiple orders through the dedicated V5 trade connection.
func (e *Exchange) WebsocketCancelV5BatchOrders(ctx context.Context, req *V5CancelBatchOrdersRequest) (*V5WebsocketBatchOrderResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	var resp *V5WebsocketBatchOrderResponse
	err := e.sendV5TradeRequest(ctx, "cancel_batch_orders", req, &resp)
	return resp, err
}

// WebsocketCancelAllV5Orders cancels all matching orders through the dedicated V5 trade connection.
func (e *Exchange) WebsocketCancelAllV5Orders(ctx context.Context, req *V5CancelAllOrdersRequest) (*V5WebsocketBatchOrderResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	var resp *V5WebsocketBatchOrderResponse
	err := e.sendV5TradeRequest(ctx, "cancel_all_orders", req, &resp)
	return resp, err
}

// sendV5TradeRequest centralises response correlation and exchange error
// handling for authenticated V5 websocket operations.
func (e *Exchange) sendV5TradeRequest(ctx context.Context, operation string, data, response any) error {
	if !e.Websocket.CanUseAuthenticatedEndpoints() {
		return request.ErrAuthRequestFailed
	}
	conn, err := e.Websocket.GetConnection(exchange.WebsocketTrade)
	if err != nil {
		return err
	}
	cid := e.MessageID()
	raw, err := conn.SendMessageReturnResponse(ctx, request.Unset, cid, &V5WebsocketTradeRequest{Operation: operation, CID: cid, Parameters: data})
	if err != nil {
		return err
	}
	if err := getErrResp(raw); err != nil {
		return err
	}
	return json.Unmarshal(raw, response)
}

// sendV5WSExecutionUpdates publishes executions without inventing order state or remaining volume.
func (e *Exchange) sendV5WSExecutionUpdates(ctx context.Context, sub *subscription.Subscription, contractCode string, data []V5WebsocketExecution) error {
	fills := make([]fill.Data, 0, len(data))
	for i := range data {
		execution := &data[i]
		symbol := execution.ContractCode
		if symbol == "" {
			symbol = contractCode
		}
		pair, err := currency.NewPairFromString(symbol)
		if err != nil {
			return err
		}
		side := execution.Side
		if side == "" {
			side = execution.Direction
		}
		orderSide, err := order.StringToOrderSide(side)
		if err != nil {
			return err
		}
		timestamp := execution.UpdatedTime.Time()
		if timestamp.IsZero() {
			timestamp = execution.CreatedTime.Time()
		}
		fills = append(fills, fill.Data{
			ID:            execution.ID,
			Timestamp:     timestamp,
			Exchange:      e.Name,
			AssetType:     sub.Asset,
			CurrencyPair:  pair,
			Side:          orderSide,
			OrderID:       execution.OrderID.String(),
			ClientOrderID: execution.ClientOrderID,
			TradeID:       execution.TradeID.String(),
			Price:         execution.TradePrice.Float64(),
			Amount:        execution.TradeVolume.Float64(),
		})
	}
	if len(fills) == 0 {
		return nil
	}
	return e.Websocket.DataHandler.Send(ctx, fills)
}
