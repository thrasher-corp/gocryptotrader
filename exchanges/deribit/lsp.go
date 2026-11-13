package deribit

import (
	"context"

	"github.com/thrasher-corp/gocryptotrader/common"
)

// GetLiquiditySupportProgramParticipantConfig returns the calling subaccount's Liquidity Support Program configuration;
// it needs the participant subaccount's API key, and Deribit returns an error for an account without one
func (e *Exchange) GetLiquiditySupportProgramParticipantConfig(ctx context.Context) (*LiquiditySupportProgramParticipantConfigResponse, error) {
	return e.getLiquiditySupportProgramParticipantConfig(ctx, false)
}

func (e *Exchange) getLiquiditySupportProgramParticipantConfig(ctx context.Context, overWebsocket bool) (*LiquiditySupportProgramParticipantConfigResponse, error) {
	var resp *LiquiditySupportProgramParticipantConfigResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_lsp_participant_config", nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetLiquiditySupportProgramParticipants returns the Liquidity Support Program participants under the main account,
// with each one's configuration when detailed is set; it needs the main account's API key
func (e *Exchange) GetLiquiditySupportProgramParticipants(ctx context.Context, detailed bool) (*LiquiditySupportProgramParticipantsResponse, error) {
	return e.getLiquiditySupportProgramParticipants(ctx, detailed, false)
}

func (e *Exchange) getLiquiditySupportProgramParticipants(ctx context.Context, detailed, overWebsocket bool) (*LiquiditySupportProgramParticipantsResponse, error) {
	var resp *LiquiditySupportProgramParticipantsResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_lsp_participants", &LiquiditySupportProgramParticipantsRequest{Detailed: detailed}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetLiquiditySupportProgramParticipantsUsage returns the assignment usage of each Liquidity Support Program
// participant under the main account, or of the participant userID selects when it is not zero; it needs the main
// account's API key
func (e *Exchange) GetLiquiditySupportProgramParticipantsUsage(ctx context.Context, userID uint64) (*LiquiditySupportProgramParticipantsUsageResponse, error) {
	return e.getLiquiditySupportProgramParticipantsUsage(ctx, userID, false)
}

func (e *Exchange) getLiquiditySupportProgramParticipantsUsage(ctx context.Context, userID uint64, overWebsocket bool) (*LiquiditySupportProgramParticipantsUsageResponse, error) {
	var resp *LiquiditySupportProgramParticipantsUsageResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_lsp_participants_usage", &LiquiditySupportProgramParticipantsUsageRequest{UserID: userID}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetLiquiditySupportProgramUsage returns the calling participant's assignment usage within its current cooldown
// window; it needs the participant subaccount's API key, and Deribit returns an error for an account that does not
// participate
func (e *Exchange) GetLiquiditySupportProgramUsage(ctx context.Context) (*LiquiditySupportProgramUsageResponse, error) {
	return e.getLiquiditySupportProgramUsage(ctx, false)
}

func (e *Exchange) getLiquiditySupportProgramUsage(ctx context.Context, overWebsocket bool) (*LiquiditySupportProgramUsageResponse, error) {
	var resp *LiquiditySupportProgramUsageResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_lsp_usage", nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}
