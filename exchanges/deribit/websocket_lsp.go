package deribit

import (
	"context"
)

// GetLSPParticipantConfig returns the participant configuration as Exchange.GetLSPParticipantConfig does, over the
// websocket
func (w WebsocketAPI) GetLSPParticipantConfig(ctx context.Context) (*LSPParticipantConfigResponse, error) {
	return w.e.getLSPParticipantConfig(ctx, true)
}

// GetLSPParticipants returns the participants as Exchange.GetLSPParticipants does, over the websocket
func (w WebsocketAPI) GetLSPParticipants(ctx context.Context, detailed bool) (*LSPParticipantsResponse, error) {
	return w.e.getLSPParticipants(ctx, detailed, true)
}

// GetLSPParticipantsUsage returns the participants' usage as Exchange.GetLSPParticipantsUsage does, over the websocket
func (w WebsocketAPI) GetLSPParticipantsUsage(ctx context.Context, userID uint64) (*LSPParticipantsUsageResponse, error) {
	return w.e.getLSPParticipantsUsage(ctx, userID, true)
}

// GetLSPUsage returns the participant's usage as Exchange.GetLSPUsage does, over the websocket
func (w WebsocketAPI) GetLSPUsage(ctx context.Context) (*LSPUsageResponse, error) {
	return w.e.getLSPUsage(ctx, true)
}
