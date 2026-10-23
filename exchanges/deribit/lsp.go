package deribit

import (
	"context"

	"github.com/thrasher-corp/gocryptotrader/common"
)

// GetLSPParticipantConfig returns the calling subaccount's Liquidity Support Program configuration; it needs the
// participant subaccount's API key, and Deribit returns an error for an account without one
func (e *Exchange) GetLSPParticipantConfig(ctx context.Context) (*LSPParticipantConfigResponse, error) {
	return e.getLSPParticipantConfig(ctx, false)
}

func (e *Exchange) getLSPParticipantConfig(ctx context.Context, overWebsocket bool) (*LSPParticipantConfigResponse, error) {
	var resp *LSPParticipantConfigResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_lsp_participant_config", nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetLSPParticipants returns the Liquidity Support Program participants under the main account, with each one's
// configuration when detailed is set; it needs the main account's API key
func (e *Exchange) GetLSPParticipants(ctx context.Context, detailed bool) (*LSPParticipantsResponse, error) {
	return e.getLSPParticipants(ctx, detailed, false)
}

func (e *Exchange) getLSPParticipants(ctx context.Context, detailed, overWebsocket bool) (*LSPParticipantsResponse, error) {
	var resp *LSPParticipantsResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_lsp_participants", &LSPParticipantsRequest{Detailed: detailed}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetLSPParticipantsUsage returns the assignment usage of each Liquidity Support Program participant under the main
// account, or of the participant userID selects when it is not zero; it needs the main account's API key
func (e *Exchange) GetLSPParticipantsUsage(ctx context.Context, userID uint64) (*LSPParticipantsUsageResponse, error) {
	return e.getLSPParticipantsUsage(ctx, userID, false)
}

func (e *Exchange) getLSPParticipantsUsage(ctx context.Context, userID uint64, overWebsocket bool) (*LSPParticipantsUsageResponse, error) {
	var resp *LSPParticipantsUsageResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_lsp_participants_usage", &LSPParticipantsUsageRequest{UserID: userID}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetLSPUsage returns the calling participant's assignment usage within its current cooldown window; it needs the
// participant subaccount's API key, and Deribit returns an error for an account that does not participate
func (e *Exchange) GetLSPUsage(ctx context.Context) (*LSPUsageResponse, error) {
	return e.getLSPUsage(ctx, false)
}

func (e *Exchange) getLSPUsage(ctx context.Context, overWebsocket bool) (*LSPUsageResponse, error) {
	var resp *LSPUsageResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_lsp_usage", nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}
