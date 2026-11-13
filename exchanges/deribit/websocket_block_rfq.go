package deribit

import (
	"context"
)

// AcceptBlockRFQ accepts a Block RFQ's quotes as Exchange.AcceptBlockRFQ does, over the websocket
func (w WebsocketAPI) AcceptBlockRFQ(ctx context.Context, arg *AcceptBlockRFQRequest) (*AcceptBlockRFQResponse, error) {
	return w.e.acceptBlockRFQ(ctx, arg, true)
}

// AddBlockRFQQuote quotes a Block RFQ as Exchange.AddBlockRFQQuote does, over the websocket
func (w WebsocketAPI) AddBlockRFQQuote(ctx context.Context, arg *AddBlockRFQQuoteRequest) (*BlockRFQQuoteResponse, error) {
	return w.e.addBlockRFQQuote(ctx, arg, true)
}

// CancelAllBlockRFQQuotes cancels the account's Block RFQ quotes as Exchange.CancelAllBlockRFQQuotes does, over the
// websocket
func (w WebsocketAPI) CancelAllBlockRFQQuotes(ctx context.Context, arg *CancelAllBlockRFQQuotesRequest) (*CancelAllBlockRFQQuotesResponse, error) {
	return w.e.cancelAllBlockRFQQuotes(ctx, arg, true)
}

// CancelBlockRFQ cancels a Block RFQ as Exchange.CancelBlockRFQ does, over the websocket
func (w WebsocketAPI) CancelBlockRFQ(ctx context.Context, blockRFQID uint64) (*BlockRFQResponse, error) {
	return w.e.cancelBlockRFQ(ctx, blockRFQID, "private/cancel_block_rfq", true)
}

// CancelBlockRFQQuote cancels a Block RFQ quote as Exchange.CancelBlockRFQQuote does, over the websocket
func (w WebsocketAPI) CancelBlockRFQQuote(ctx context.Context, arg *CancelBlockRFQQuoteRequest) (*BlockRFQQuoteResponse, error) {
	return w.e.cancelBlockRFQQuote(ctx, arg, true)
}

// CancelBlockRFQTrigger cancels a Block RFQ's trade trigger as Exchange.CancelBlockRFQTrigger does, over the websocket
func (w WebsocketAPI) CancelBlockRFQTrigger(ctx context.Context, blockRFQID uint64) (*BlockRFQResponse, error) {
	return w.e.cancelBlockRFQ(ctx, blockRFQID, "private/cancel_block_rfq_trigger", true)
}

// CreateBlockRFQ creates a Block RFQ as Exchange.CreateBlockRFQ does, over the websocket
func (w WebsocketAPI) CreateBlockRFQ(ctx context.Context, arg *CreateBlockRFQRequest) (*BlockRFQResponse, error) {
	return w.e.createBlockRFQ(ctx, arg, true)
}

// EditBlockRFQQuote edits a Block RFQ quote as Exchange.EditBlockRFQQuote does, over the websocket
func (w WebsocketAPI) EditBlockRFQQuote(ctx context.Context, arg *EditBlockRFQQuoteRequest) (*BlockRFQQuoteResponse, error) {
	return w.e.editBlockRFQQuote(ctx, arg, true)
}

// GetBlockRFQMakers returns the Block RFQ makers' aliases as Exchange.GetBlockRFQMakers does, over the websocket
func (w WebsocketAPI) GetBlockRFQMakers(ctx context.Context) ([]string, error) {
	return w.e.getBlockRFQMakers(ctx, true)
}

// GetBlockRFQQuotes returns the account's open Block RFQ quotes as Exchange.GetBlockRFQQuotes does, over the websocket
func (w WebsocketAPI) GetBlockRFQQuotes(ctx context.Context, arg *GetBlockRFQQuotesRequest) ([]BlockRFQQuoteResponse, error) {
	return w.e.getBlockRFQQuotes(ctx, arg, true)
}

// GetBlockRFQUserInfo returns the account's Block RFQ identities as Exchange.GetBlockRFQUserInfo does, over the
// websocket
func (w WebsocketAPI) GetBlockRFQUserInfo(ctx context.Context) (*BlockRFQUserInfoResponse, error) {
	return w.e.getBlockRFQUserInfo(ctx, true)
}

// GetBlockRFQs returns the account's Block RFQs as Exchange.GetBlockRFQs does, over the websocket
func (w WebsocketAPI) GetBlockRFQs(ctx context.Context, arg *GetBlockRFQsRequest) (*BlockRFQsResponse, error) {
	return w.e.getBlockRFQs(ctx, arg, true)
}

// GetBlockRFQTrades returns recent Block RFQ trades as Exchange.GetBlockRFQTrades does, over the websocket
func (w WebsocketAPI) GetBlockRFQTrades(ctx context.Context, arg *GetBlockRFQTradesRequest) (*BlockRFQTradesResponse, error) {
	return w.e.getBlockRFQTrades(ctx, arg, true)
}
