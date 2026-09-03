package htx

import (
	"context"
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
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
		response := new(V5WsOrderUpdate)
		if err := json.Unmarshal(raw, response); err != nil {
			return err
		}
		return e.sendV5WSOrderUpdate(ctx, sub, response.ContractCode, &response.Data)
	case wsTradeUpdatesChannel:
		response := new(V5WsTradeUpdate)
		if err := json.Unmarshal(raw, response); err != nil {
			return err
		}
		return e.sendV5WSOrderUpdate(ctx, sub, response.ContractCode, &response.Data)
	case wsExecutionDetailsChannel:
		response := new(V5WsTradeDetailUpdate)
		if err := json.Unmarshal(raw, response); err != nil {
			return err
		}
		return e.sendV5WSOrderUpdate(ctx, sub, response.ContractCode, &response.Data)
	case wsPositionsChannel:
		response := new(V5WsPositionUpdate)
		if err := json.Unmarshal(raw, response); err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, response)
	case subscription.MyAccountChannel:
		response := new(V5WsAccountUpdate)
		if err := json.Unmarshal(raw, response); err != nil {
			return err
		}
		changes := make([]accounts.Change, 0, len(response.Data.Details))
		for i := range response.Data.Details {
			free := response.Data.Details[i].Available.Float64() + response.Data.Details[i].IsolatedAvailable.Float64()
			changes = append(changes, accounts.Change{
				AssetType: sub.Asset,
				Balance: accounts.Balance{
					Currency:  currency.NewCode(response.Data.Details[i].Currency),
					Total:     response.Data.Details[i].Equity.Float64(),
					Free:      free,
					Hold:      response.Data.Details[i].Equity.Float64() - free,
					UpdatedAt: response.Timestamp.Time(),
				},
			})
		}
		return e.Websocket.DataHandler.Send(ctx, changes)
	case subscription.MyTradesChannel:
		response := new(V5WsMatchOrderUpdate)
		if err := json.Unmarshal(raw, response); err != nil {
			return err
		}
		return e.sendV5WSOrderUpdate(ctx, sub, response.ContractCode, &response.Data)
	case wsTriggerOrdersChannel:
		response := new(V5WsAlgoOrderUpdate)
		if err := json.Unmarshal(raw, response); err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, response)
	default:
		return fmt.Errorf("%w: %s", common.ErrNotYetImplemented, sub.Channel)
	}
}

// sendV5WSOrderUpdate converts each V5 order-notification variant into the canonical order type.
func (e *Exchange) sendV5WSOrderUpdate(ctx context.Context, sub *subscription.Subscription, contractCode string, data *V5WsOrderData) error {
	detail, err := e.formatV5OrderDetail(&V5OrderData{
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
		State:             V5OrderState(data.State),
		ReduceOnly:        V5Boolean(data.ReduceOnly),
		TimeInForce:       data.TimeInForce,
		TradeAveragePrice: data.TradeAveragePrice,
		TradeVolume:       data.TradeVolume,
		TradeTurnover:     data.TradeTurnover,
		FeeCurrency:       data.FeeCurrency,
		Fee:               data.Fee,
		CreatedTime:       types.Time(time.UnixMilli(data.CreatedTime.Int64())),
		UpdatedTime:       types.Time(time.UnixMilli(data.UpdatedTime.Int64())),
	}, sub.Asset)
	if err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, &detail)
}

// WSPlaceV5Order places an order through the dedicated V5 trade connection.
func (e *Exchange) WSPlaceV5Order(ctx context.Context, req *V5OrderRequest) (*V5WsOrderResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	var resp *V5WsOrderResponse
	if err := e.sendV5TradeRequest(ctx, "place_order", req, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// WSPlaceV5BatchOrders places multiple orders through the dedicated V5 trade connection.
func (e *Exchange) WSPlaceV5BatchOrders(ctx context.Context, req []*V5OrderRequest) (*V5WsBatchOrderResponse, error) {
	if len(req) == 0 {
		return nil, common.ErrEmptyParams
	}
	var resp *V5WsBatchOrderResponse
	if err := e.sendV5TradeRequest(ctx, "place_batch_orders", req, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// WSCancelV5Order cancels an order through the dedicated V5 trade connection.
func (e *Exchange) WSCancelV5Order(ctx context.Context, req *V5CancelOrderRequest) (*V5WsOrderResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	var resp *V5WsOrderResponse
	if err := e.sendV5TradeRequest(ctx, "cancel_order", req, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// WSCancelV5BatchOrders cancels multiple orders through the dedicated V5 trade connection.
func (e *Exchange) WSCancelV5BatchOrders(ctx context.Context, req *V5CancelBatchOrdersRequest) (*V5WsBatchOrderResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	var resp *V5WsBatchOrderResponse
	if err := e.sendV5TradeRequest(ctx, "cancel_batch_orders", req, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// WSCancelAllV5Orders cancels all matching orders through the dedicated V5 trade connection.
func (e *Exchange) WSCancelAllV5Orders(ctx context.Context, req *V5CancelAllOrdersRequest) (*V5WsBatchOrderResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	var resp *V5WsBatchOrderResponse
	if err := e.sendV5TradeRequest(ctx, "cancel_all_orders", req, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// sendV5TradeRequest centralises response correlation and exchange error
// handling for authenticated V5 websocket operations.
func (e *Exchange) sendV5TradeRequest(ctx context.Context, operation string, data, response any) error {
	conn, err := e.Websocket.GetConnection(exchange.WebsocketTrade)
	if err != nil {
		return err
	}
	cid := e.MessageID()
	raw, err := conn.SendMessageReturnResponse(ctx, request.Unset, cid, &V5WsTradeRequest{Operation: operation, CID: cid, Data: data})
	if err != nil {
		return err
	}
	if err := getErrResp(raw); err != nil {
		return err
	}
	return json.Unmarshal(raw, response)
}
