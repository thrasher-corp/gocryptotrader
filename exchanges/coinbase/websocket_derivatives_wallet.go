package coinbase

import "context"

// SubmitTransferBetweenSubaccounts transfers funds between subaccounts as
// GlobalDerivativesAPI.SubmitTransferBetweenSubaccounts does, over the websocket
func (w WebsocketAPI) SubmitTransferBetweenSubaccounts(ctx context.Context, req *DerivativesSubmitTransferBetweenSubaccountsRequest) (*DerivativesTransfer, error) {
	return w.e.GlobalDerivatives().submitTransferBetweenSubaccounts(ctx, true, req)
}
