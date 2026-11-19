package binanceus

import "context"

// TestConnectivity tests connectivity to the WebSocket API as Exchange.TestConnectivity does, over the WebSocket API
func (w WebsocketAPI) TestConnectivity(ctx context.Context) error {
	return w.e.testConnectivity(ctx, true)
}

// GetExchangeServerTime returns the server time as Exchange.GetExchangeServerTime does, over the WebSocket API
func (w WebsocketAPI) GetExchangeServerTime(ctx context.Context) (*ServerTimeResponse, error) {
	return w.e.getExchangeServerTime(ctx, true)
}

// GetExchangeInfo returns the exchange trading rules and symbol information as Exchange.GetExchangeInfo does, over the
// WebSocket API
func (w WebsocketAPI) GetExchangeInfo(ctx context.Context, req *ExchangeInfoRequest) (*ExchangeInfoResponse, error) {
	return w.e.getExchangeInfo(ctx, req, true)
}
