package htx

import (
	"context"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

// wsHandleUSDTMarginedPrivateMessage decodes authenticated V5 USDT notifications.
func (e *Exchange) wsHandleUSDTMarginedPrivateMessage(ctx context.Context, sub *subscription.Subscription, raw []byte) error {
	if err := common.NilGuard(sub); err != nil {
		return err
	}
	var response any
	switch sub.Channel {
	case subscription.MyOrdersChannel:
		response = new(V5WsOrderUpdate)
	case wsTradeUpdatesChannel:
		response = new(V5WsTradeUpdate)
	case wsExecutionDetailsChannel:
		response = new(V5WsTradeDetailUpdate)
	case wsPositionsChannel:
		response = new(V5WsPositionUpdate)
	case subscription.MyAccountChannel:
		response = new(V5WsAccountUpdate)
	case subscription.MyTradesChannel:
		response = new(V5WsMatchOrderUpdate)
	case wsTriggerOrdersChannel:
		response = new(V5WsAlgoOrderUpdate)
	default:
		return fmt.Errorf("%w: %s", common.ErrNotYetImplemented, sub.Channel)
	}
	if err := json.Unmarshal(raw, response); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, response)
}
