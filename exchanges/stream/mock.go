package stream

import (
	"context"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// MockWebsocketConnection is a mock websocket connection
type MockWebsocketConnection struct {
	WebsocketConnection
}

// SendMessageReturnResponse returns a mock response from context
func (m *MockWebsocketConnection) SendMessageReturnResponse(ctx context.Context, epl request.EndpointLimit, signature, payload any) ([]byte, error) {
	resps, _ := m.SendMessageReturnResponses(ctx, epl, signature, payload, 1)
	return resps[0], nil
}

// SendMessageReturnResponses returns a mock response from context
func (m *MockWebsocketConnection) SendMessageReturnResponses(ctx context.Context, epl request.EndpointLimit, signature, payload any, expected int) ([][]byte, error) {
	return m.SendMessageReturnResponsesWithInspector(ctx, epl, signature, payload, expected, nil)
}

// SendMessageReturnResponsesWithInspector returns a mock response from context
func (*MockWebsocketConnection) SendMessageReturnResponsesWithInspector(ctx context.Context, _ request.EndpointLimit, _, _ any, _ int, _ Inspector) ([][]byte, error) {
	return request.GetMockResponse(ctx), nil
}

// newMockConnection returns a new mock websocket connection, used so that the websocket does not need to be connected
func newMockWebsocketConnection() Connection {
	return &MockWebsocketConnection{}
}
