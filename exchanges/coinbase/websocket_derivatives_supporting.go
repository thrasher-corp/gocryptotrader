package coinbase

import (
	"context"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
)

// GetTime returns the gateway's current time as GlobalDerivativesAPI.GetTime does, over the websocket
func (w WebsocketAPI) GetTime(ctx context.Context) (time.Time, error) {
	return w.e.GlobalDerivatives().getTime(ctx, true)
}

// Hello introduces the client software to the gateway, which collects its name and version for statistics, and returns
// the API version; the gateway only answers it over the websocket
func (w WebsocketAPI) Hello(ctx context.Context, req *DerivativesHelloRequest) (*DerivativesHelloResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.ClientName == "" {
		return nil, errDerivativesClientNameRequired
	}
	if req.ClientVersion == "" {
		return nil, errDerivativesClientVersionRequired
	}
	var resp *DerivativesHelloResponse
	params := &DerivativesHelloWireRequest{ClientName: req.ClientName, ClientVersion: req.ClientVersion}
	if err := w.SendRequest(ctx, derivativesNonMatchingEPL, "public/hello", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetStatus returns the platform's lock status as GlobalDerivativesAPI.GetStatus does, over the websocket
func (w WebsocketAPI) GetStatus(ctx context.Context) (*DerivativesStatusResponse, error) {
	return w.e.GlobalDerivatives().getStatus(ctx, true)
}

// TestConnection checks the gateway is reachable as GlobalDerivativesAPI.TestConnection does, over the websocket
func (w WebsocketAPI) TestConnection(ctx context.Context, expectedResult string) (*DerivativesTestConnectionResponse, error) {
	return w.e.GlobalDerivatives().testConnection(ctx, true, expectedResult)
}
