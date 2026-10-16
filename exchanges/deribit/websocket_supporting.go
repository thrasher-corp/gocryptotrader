package deribit

import (
	"context"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
)

// GetTime returns Deribit's current time as Exchange.GetTime does, over the websocket
func (w WebsocketAPI) GetTime(ctx context.Context) (time.Time, error) {
	return w.e.getTime(ctx, true)
}

// Hello introduces the client software to Deribit, which collects its name and version for statistics, and returns the
// API version; Deribit only accepts it over the websocket
func (w WebsocketAPI) Hello(ctx context.Context, arg *HelloRequest) (*HelloResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.ClientName == "" {
		return nil, errClientNameRequired
	}
	if arg.ClientVersion == "" {
		return nil, errClientVersionRequired
	}
	var resp *HelloResponse
	if err := w.e.SendWebsocketRequest(ctx, nonMatchingEPL, "public/hello", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetStatus returns the platform's lock status as Exchange.GetStatus does, over the websocket
func (w WebsocketAPI) GetStatus(ctx context.Context) (*StatusResponse, error) {
	return w.e.getStatus(ctx, true)
}

// TestConnection checks the API is reachable as Exchange.TestConnection does, over the websocket; Deribit's heartbeat
// test requests are answered with it
func (w WebsocketAPI) TestConnection(ctx context.Context, expectedResult string) (*TestConnectionResponse, error) {
	return w.e.testConnection(ctx, expectedResult, true)
}
