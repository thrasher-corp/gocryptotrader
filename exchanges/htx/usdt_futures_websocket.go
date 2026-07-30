package htx

import (
	"context"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

// wsHandleUSDTMarginedPrivateMessage decodes isolated and cross-margin USDT notifications.
func (e *Exchange) wsHandleUSDTMarginedPrivateMessage(ctx context.Context, sub *subscription.Subscription, raw []byte) error {
	if err := common.NilGuard(sub); err != nil {
		return err
	}
	var response any
	switch sub.Channel {
	case subscription.MyOrdersChannel, wsCrossOrdersChannel:
		response = new(SwapWsSubOrderData)
	case subscription.MyTradesChannel, wsCrossTradesChannel:
		response = new(SwapWsSubMatchOrderData)
	case subscription.MyAccountChannel, wsCrossAccountsChannel:
		response = new(SwapWsSubEquityData)
	case wsPositionsChannel, wsCrossPositionsChannel:
		response = new(SwapWsSubPositionUpdates)
	case wsTriggerOrdersChannel, wsCrossTriggersChannel:
		response = new(SwapWsSubTriggerOrderUpdates)
	default:
		return fmt.Errorf("%w: %s", common.ErrNotYetImplemented, sub.Channel)
	}
	if err := json.Unmarshal(raw, response); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, response)
}
