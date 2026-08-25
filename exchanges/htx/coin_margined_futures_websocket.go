package htx

import (
	"context"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

// wsHandleCoinMarginedPrivateMessage decodes authenticated coin-margined notifications.
func (e *Exchange) wsHandleCoinMarginedPrivateMessage(ctx context.Context, sub *subscription.Subscription, raw []byte) error {
	if err := common.NilGuard(sub); err != nil {
		return err
	}
	switch sub.Channel {
	case subscription.MyOrdersChannel:
		response := new(SwapWsSubOrderData)
		if err := json.Unmarshal(raw, response); err != nil {
			return err
		}
		detail, err := e.formatLegacyFuturesWSOrder(&legacyFuturesWSOrder{
			asset:          sub.Asset,
			contractCode:   response.ContractCode,
			direction:      response.Direction,
			orderPriceType: response.OrderPriceType,
			status:         response.Status,
			orderID:        response.OrderID,
			orderIDString:  response.OrderIDString,
			clientOrderID:  response.ClientOrderID,
			volume:         response.Volume,
			price:          response.Price,
			tradeVolume:    response.TradeVolume,
			tradeTurnover:  response.TradeTurnover,
			fee:            response.Fee,
			feeAsset:       response.FeeAsset,
			leverage:       response.LeverageRate,
			createdAt:      response.CreatedAt,
			cancelledAt:    response.CanceledAt,
			reduceOnly:     response.Offset == orderOffsetClose,
		})
		if err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, &detail)
	case subscription.MyTradesChannel:
		response := new(SwapWsSubMatchOrderData)
		if err := json.Unmarshal(raw, response); err != nil {
			return err
		}
		detail, err := e.formatLegacyFuturesWSOrder(&legacyFuturesWSOrder{
			asset:          sub.Asset,
			contractCode:   response.ContractCode,
			direction:      response.Direction,
			orderPriceType: response.OrderPriceType,
			orderType:      response.OrderType,
			status:         response.Status,
			orderID:        response.OrderID,
			orderIDString:  response.OrderIDString,
			clientOrderID:  response.ClientOrderID,
			volume:         response.Volume,
			price:          response.Price,
			tradeVolume:    response.TradeVolume,
			leverage:       response.LeverageRate,
			createdAt:      response.CreatedAt,
			reduceOnly:     response.Offset == orderOffsetClose,
		})
		if err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, &detail)
	case subscription.MyAccountChannel:
		response := new(SwapWsSubEquityData)
		if err := json.Unmarshal(raw, response); err != nil {
			return err
		}
		changes := make([]accounts.Change, 0, len(response.Data))
		for i := range response.Data {
			code := response.Data[i].MarginAsset
			if code == "" {
				code = response.Data[i].Symbol
			}
			changes = append(changes, accounts.Change{
				AssetType: sub.Asset,
				Balance: accounts.Balance{
					Currency:  currency.NewCode(code),
					Total:     response.Data[i].MarginBalance,
					Hold:      response.Data[i].MarginFrozen,
					Free:      response.Data[i].MarginAvailable,
					UpdatedAt: response.Timestamp.Time(),
				},
			})
		}
		return e.Websocket.DataHandler.Send(ctx, changes)
	case wsPositionsChannel:
		response := new(SwapWsSubPositionUpdates)
		if err := json.Unmarshal(raw, response); err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, response)
	case wsTriggerOrdersChannel:
		response := new(SwapWsSubTriggerOrderUpdates)
		if err := json.Unmarshal(raw, response); err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, response)
	default:
		return fmt.Errorf("%w: %s", common.ErrNotYetImplemented, sub.Channel)
	}
}
