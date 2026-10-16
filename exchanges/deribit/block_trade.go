package deribit

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
)

// Block trade roles; a block trade's two parties take opposite roles
const (
	BlockTradeRoleMaker = "maker"
	BlockTradeRoleTaker = "taker"
)

var (
	errBlockTradeIDRequired          = errors.New("block trade ID is required")
	errBlockTradeLegsRequired        = errors.New("at least one block trade leg is required")
	errBlockTradeNonceRequired       = errors.New("block trade nonce is required")
	errBlockTradeStartIDRequired     = errors.New("start ID is required with an end ID")
	errCounterpartySignatureRequired = errors.New("counterparty signature is required")
	errInvalidBlockTradeRole         = errors.New("invalid block trade role")
)

// ApproveBlockTrade approves a pending block trade a broker or registered partner executed for the account; it must be
// sent with an API key without the block_trade_approval feature, which the broker's key has
func (e *Exchange) ApproveBlockTrade(ctx context.Context, arg *BlockTradeApprovalRequest) error {
	return e.decidePendingBlockTrade(ctx, arg, "private/approve_block_trade", false)
}

// decidePendingBlockTrade approves or rejects a pending block trade with the given method
func (e *Exchange) decidePendingBlockTrade(ctx context.Context, arg *BlockTradeApprovalRequest, method string, overWebsocket bool) error {
	if arg == nil {
		return common.ErrNilPointer
	}
	if err := validateBlockTradeKey(arg.Timestamp, arg.Nonce, arg.Role); err != nil {
		return err
	}
	params := &blockTradeApprovalParams{BlockTradeApprovalRequest: arg, Timestamp: arg.Timestamp.UnixMilli()}
	return e.sendOKRequest(ctx, overWebsocket, nonMatchingEPL, method, params)
}

// ExecuteBlockTrade executes a block trade with the signature the counterparty's VerifyBlockTrade returned, taking the
// opposite role; Deribit executes a block trade only once, whichever party executes it
func (e *Exchange) ExecuteBlockTrade(ctx context.Context, arg *ExecuteBlockTradeRequest) (*BlockTrade, error) {
	return e.executeBlockTrade(ctx, arg, false)
}

func (e *Exchange) executeBlockTrade(ctx context.Context, arg *ExecuteBlockTradeRequest, overWebsocket bool) (*BlockTrade, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if err := validateBlockTradeKey(arg.Timestamp, arg.Nonce, arg.Role); err != nil {
		return nil, err
	}
	if err := validateBlockTradeLegs(arg.Trades); err != nil {
		return nil, err
	}
	if arg.CounterpartySignature == "" {
		return nil, errCounterpartySignatureRequired
	}
	params := &executeBlockTradeParams{ExecuteBlockTradeRequest: arg, Timestamp: arg.Timestamp.UnixMilli()}
	var resp *BlockTrade
	if err := e.sendRequest(ctx, overWebsocket, matchingEPL, "private/execute_block_trade", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetBlockTrade returns one of the account's block trades and its trades
func (e *Exchange) GetBlockTrade(ctx context.Context, id string) (*BlockTrade, error) {
	return e.getBlockTrade(ctx, id, false)
}

func (e *Exchange) getBlockTrade(ctx context.Context, id string, overWebsocket bool) (*BlockTrade, error) {
	if id == "" {
		return nil, errBlockTradeIDRequired
	}
	var resp *BlockTrade
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_block_trade", &blockTradeIDParams{ID: id}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetBlockTradeRequests returns the block trade requests awaiting the account's approval, and those declined or let
// expire; an empty brokerCode returns every request, and only broker clients can filter by their broker's code
func (e *Exchange) GetBlockTradeRequests(ctx context.Context, brokerCode string) ([]PendingBlockTrade, error) {
	return e.getBlockTradeRequests(ctx, brokerCode, false)
}

func (e *Exchange) getBlockTradeRequests(ctx context.Context, brokerCode string, overWebsocket bool) ([]PendingBlockTrade, error) {
	var resp []PendingBlockTrade
	return resp, e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_block_trade_requests", &brokerCodeParams{BrokerCode: brokerCode}, &resp)
}

// GetBlockTrades returns the account's block trades, newest first, in every currency unless one is selected
func (e *Exchange) GetBlockTrades(ctx context.Context, arg *GetBlockTradesRequest) ([]BlockTrade, error) {
	return e.getBlockTrades(ctx, arg, false)
}

func (e *Exchange) getBlockTrades(ctx context.Context, arg *GetBlockTradesRequest, overWebsocket bool) ([]BlockTrade, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.EndID != "" && arg.StartID == "" {
		return nil, errBlockTradeStartIDRequired
	}
	var resp []BlockTrade
	return resp, e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_block_trades", arg, &resp)
}

// GetBrokerClients returns the clients registered under the broker account, with the client and link IDs that identify
// a client's users in broker calls; it needs a broker account
func (e *Exchange) GetBrokerClients(ctx context.Context, arg *GetBrokerClientsRequest) ([]BrokerClient, error) {
	return e.getBrokerClients(ctx, arg, false)
}

func (e *Exchange) getBrokerClients(ctx context.Context, arg *GetBrokerClientsRequest, overWebsocket bool) ([]BrokerClient, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	var resp []BrokerClient
	return resp, e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_broker_clients", arg, &resp)
}

// GetBrokerTradeRequests returns the broker's block trade requests awaiting its clients' approval, and those declined
// or let expire; it needs a broker account
func (e *Exchange) GetBrokerTradeRequests(ctx context.Context) ([]PendingBrokerTrade, error) {
	return e.getBrokerTradeRequests(ctx, false)
}

func (e *Exchange) getBrokerTradeRequests(ctx context.Context, overWebsocket bool) ([]PendingBrokerTrade, error) {
	var resp []PendingBrokerTrade
	return resp, e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_broker_trade_requests", nil, &resp)
}

// GetBrokerTrades returns the broker's block trades, newest first, in every currency unless one is selected; it needs a
// broker account
func (e *Exchange) GetBrokerTrades(ctx context.Context, arg *GetBrokerTradesRequest) (*BrokerTradesResponse, error) {
	return e.getBrokerTrades(ctx, arg, false)
}

func (e *Exchange) getBrokerTrades(ctx context.Context, arg *GetBrokerTradesRequest, overWebsocket bool) (*BrokerTradesResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.EndID != "" && arg.StartID == "" {
		return nil, errBlockTradeStartIDRequired
	}
	var resp *BrokerTradesResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_broker_trades", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// InvalidateBlockTradeSignature invalidates a signature VerifyBlockTrade returned, cancelling the block trade unless
// the counterparty has already executed it
func (e *Exchange) InvalidateBlockTradeSignature(ctx context.Context, signature string) error {
	return e.invalidateBlockTradeSignature(ctx, signature, false)
}

func (e *Exchange) invalidateBlockTradeSignature(ctx context.Context, signature string, overWebsocket bool) error {
	if signature == "" {
		return errSignatureRequired
	}
	return e.sendOKRequest(ctx, overWebsocket, nonMatchingEPL, "private/invalidate_block_trade_signature", &blockTradeSignatureParams{Signature: signature})
}

// RejectBlockTrade rejects a pending block trade a broker or registered partner executed for the account, which cancels
// it; it must be sent with an API key without the block_trade_approval feature, which the broker's key has
func (e *Exchange) RejectBlockTrade(ctx context.Context, arg *BlockTradeApprovalRequest) error {
	return e.decidePendingBlockTrade(ctx, arg, "private/reject_block_trade", false)
}

// SimulateBlockTrade reports whether the account could execute a block trade, without executing it
func (e *Exchange) SimulateBlockTrade(ctx context.Context, arg *SimulateBlockTradeRequest) (bool, error) {
	return e.simulateBlockTrade(ctx, arg, false)
}

func (e *Exchange) simulateBlockTrade(ctx context.Context, arg *SimulateBlockTradeRequest, overWebsocket bool) (bool, error) {
	if arg == nil {
		return false, common.ErrNilPointer
	}
	if arg.Role != "" {
		if err := validateBlockTradeRole(arg.Role); err != nil {
			return false, err
		}
	}
	if err := validateBlockTradeLegs(arg.Trades); err != nil {
		return false, err
	}
	var resp *bool
	if err := e.sendRequest(ctx, overWebsocket, matchingEPL, "private/simulate_block_trade", arg, &resp); err != nil {
		return false, err
	}
	if resp == nil {
		return false, common.ErrNoResponse
	}
	return *resp, nil
}

// VerifyBlockTrade verifies a block trade and returns the signature the counterparty executes it with, which is the
// first step of a block trade; the signature is valid for 5 minutes around the block trade's timestamp
func (e *Exchange) VerifyBlockTrade(ctx context.Context, arg *VerifyBlockTradeRequest) (*VerifyBlockTradeResponse, error) {
	return e.verifyBlockTrade(ctx, arg, false)
}

func (e *Exchange) verifyBlockTrade(ctx context.Context, arg *VerifyBlockTradeRequest, overWebsocket bool) (*VerifyBlockTradeResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if err := validateBlockTradeKey(arg.Timestamp, arg.Nonce, arg.Role); err != nil {
		return nil, err
	}
	if err := validateBlockTradeLegs(arg.Trades); err != nil {
		return nil, err
	}
	params := &verifyBlockTradeParams{VerifyBlockTradeRequest: arg, Timestamp: arg.Timestamp.UnixMilli()}
	var resp *VerifyBlockTradeResponse
	if err := e.sendRequest(ctx, overWebsocket, matchingEPL, "private/verify_block_trade", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// validateBlockTradeKey checks a block trade's timestamp and nonce, which its parties share, are set and the caller's
// role is maker or taker
func validateBlockTradeKey(timestamp time.Time, nonce, role string) error {
	if timestamp.IsZero() {
		return errTimestampRequired
	}
	if nonce == "" {
		return errBlockTradeNonceRequired
	}
	return validateBlockTradeRole(role)
}

// validateBlockTradeRole checks a block trade role is maker or taker
func validateBlockTradeRole(role string) error {
	switch role {
	case BlockTradeRoleMaker, BlockTradeRoleTaker:
		return nil
	default:
		return fmt.Errorf("%w: %q", errInvalidBlockTradeRole, role)
	}
}

// validateBlockTradeLegs checks a block trade has legs, each with an instrument, a direction and an amount
func validateBlockTradeLegs(legs []BlockTradeLeg) error {
	if len(legs) == 0 {
		return errBlockTradeLegsRequired
	}
	for i := range legs {
		if legs[i].InstrumentName == "" {
			return fmt.Errorf("%w: leg %d", errInstrumentNameRequired, i)
		}
		if legs[i].Direction != DirectionBuy && legs[i].Direction != DirectionSell {
			return fmt.Errorf("%w: leg %d: %q", errInvalidDirection, i, legs[i].Direction)
		}
		if legs[i].Amount <= 0 {
			return fmt.Errorf("%w: leg %d: %v", errInvalidAmount, i, legs[i].Amount)
		}
	}
	return nil
}
