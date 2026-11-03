package deribit

import (
	"context"
)

// GetLiquiditySupportProgramParticipantConfig returns the participant configuration as
// Exchange.GetLiquiditySupportProgramParticipantConfig does, over the websocket
func (w WebsocketAPI) GetLiquiditySupportProgramParticipantConfig(ctx context.Context) (*LiquiditySupportProgramParticipantConfigResponse, error) {
	return w.e.getLiquiditySupportProgramParticipantConfig(ctx, true)
}

// GetLiquiditySupportProgramParticipants returns the participants as Exchange.GetLiquiditySupportProgramParticipants
// does, over the websocket
func (w WebsocketAPI) GetLiquiditySupportProgramParticipants(ctx context.Context, detailed bool) (*LiquiditySupportProgramParticipantsResponse, error) {
	return w.e.getLiquiditySupportProgramParticipants(ctx, detailed, true)
}

// GetLiquiditySupportProgramParticipantsUsage returns the participants' usage as
// Exchange.GetLiquiditySupportProgramParticipantsUsage does, over the websocket
func (w WebsocketAPI) GetLiquiditySupportProgramParticipantsUsage(ctx context.Context, userID uint64) (*LiquiditySupportProgramParticipantsUsageResponse, error) {
	return w.e.getLiquiditySupportProgramParticipantsUsage(ctx, userID, true)
}

// GetLiquiditySupportProgramUsage returns the participant's usage as Exchange.GetLiquiditySupportProgramUsage does,
// over the websocket
func (w WebsocketAPI) GetLiquiditySupportProgramUsage(ctx context.Context) (*LiquiditySupportProgramUsageResponse, error) {
	return w.e.getLiquiditySupportProgramUsage(ctx, true)
}
