package deribit

import (
	"context"
)

// ApproveBlockTrade approves a pending block trade as Exchange.ApproveBlockTrade does, over the websocket
func (w WebsocketAPI) ApproveBlockTrade(ctx context.Context, arg *BlockTradeApprovalRequest) error {
	return w.e.decidePendingBlockTrade(ctx, arg, "private/approve_block_trade", true)
}

// ExecuteBlockTrade executes a block trade as Exchange.ExecuteBlockTrade does, over the websocket
func (w WebsocketAPI) ExecuteBlockTrade(ctx context.Context, arg *ExecuteBlockTradeRequest) (*BlockTrade, error) {
	return w.e.executeBlockTrade(ctx, arg, true)
}

// GetBlockTrade returns one of the account's block trades as Exchange.GetBlockTrade does, over the websocket
func (w WebsocketAPI) GetBlockTrade(ctx context.Context, id string) (*BlockTrade, error) {
	return w.e.getBlockTrade(ctx, id, true)
}

// GetBlockTradeRequests returns the account's block trade requests as Exchange.GetBlockTradeRequests does, over the
// websocket
func (w WebsocketAPI) GetBlockTradeRequests(ctx context.Context, brokerCode string) ([]PendingBlockTrade, error) {
	return w.e.getBlockTradeRequests(ctx, brokerCode, true)
}

// GetBlockTrades returns the account's block trades as Exchange.GetBlockTrades does, over the websocket
func (w WebsocketAPI) GetBlockTrades(ctx context.Context, arg *GetBlockTradesRequest) ([]BlockTrade, error) {
	return w.e.getBlockTrades(ctx, arg, true)
}

// GetBrokerClients returns the broker's clients as Exchange.GetBrokerClients does, over the websocket
func (w WebsocketAPI) GetBrokerClients(ctx context.Context, arg *GetBrokerClientsRequest) ([]BrokerClient, error) {
	return w.e.getBrokerClients(ctx, arg, true)
}

// GetBrokerTradeRequests returns the broker's block trade requests as Exchange.GetBrokerTradeRequests does, over the
// websocket
func (w WebsocketAPI) GetBrokerTradeRequests(ctx context.Context) ([]PendingBrokerTrade, error) {
	return w.e.getBrokerTradeRequests(ctx, true)
}

// GetBrokerTrades returns the broker's block trades as Exchange.GetBrokerTrades does, over the websocket
func (w WebsocketAPI) GetBrokerTrades(ctx context.Context, arg *GetBrokerTradesRequest) (*BrokerTradesResponse, error) {
	return w.e.getBrokerTrades(ctx, arg, true)
}

// InvalidateBlockTradeSignature invalidates a block trade signature as Exchange.InvalidateBlockTradeSignature does,
// over the websocket
func (w WebsocketAPI) InvalidateBlockTradeSignature(ctx context.Context, signature string) error {
	return w.e.invalidateBlockTradeSignature(ctx, signature, true)
}

// RejectBlockTrade rejects a pending block trade as Exchange.RejectBlockTrade does, over the websocket
func (w WebsocketAPI) RejectBlockTrade(ctx context.Context, arg *BlockTradeApprovalRequest) error {
	return w.e.decidePendingBlockTrade(ctx, arg, "private/reject_block_trade", true)
}

// SimulateBlockTrade reports whether the account could execute a block trade as Exchange.SimulateBlockTrade does, over
// the websocket
func (w WebsocketAPI) SimulateBlockTrade(ctx context.Context, arg *SimulateBlockTradeRequest) (bool, error) {
	return w.e.simulateBlockTrade(ctx, arg, true)
}

// VerifyBlockTrade verifies and signs a block trade as Exchange.VerifyBlockTrade does, over the websocket
func (w WebsocketAPI) VerifyBlockTrade(ctx context.Context, arg *VerifyBlockTradeRequest) (*VerifyBlockTradeResponse, error) {
	return w.e.verifyBlockTrade(ctx, arg, true)
}
