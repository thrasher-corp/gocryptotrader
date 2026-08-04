package htx

import (
	"context"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

// wsHandleCoinMarginedPrivateMessage decodes authenticated coin-margined notifications.
func (e *Exchange) wsHandleCoinMarginedPrivateMessage(ctx context.Context, sub *subscription.Subscription, raw []byte) error {
	if err := common.NilGuard(sub); err != nil {
		return err
	}
	var response any
	switch sub.Channel {
	case subscription.MyOrdersChannel:
		response = new(SwapWsSubOrderData)
	case subscription.MyTradesChannel:
		response = new(SwapWsSubMatchOrderData)
	case subscription.MyAccountChannel:
		response = new(SwapWsSubEquityData)
	case wsPositionsChannel:
		response = new(SwapWsSubPositionUpdates)
	case wsTriggerOrdersChannel:
		response = new(SwapWsSubTriggerOrderUpdates)
	default:
		return fmt.Errorf("%w: %s", common.ErrNotYetImplemented, sub.Channel)
	}
	if err := json.Unmarshal(raw, response); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, response)
}
