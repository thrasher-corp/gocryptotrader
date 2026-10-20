package deribit

import (
	"context"
	"errors"
	"fmt"
	"unicode/utf8"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

// Block RFQ roles, which select the Block RFQs GetBlockRFQs returns
const (
	BlockRFQRoleAny   = "any"
	BlockRFQRoleMaker = "maker"
	BlockRFQRoleTaker = "taker"
)

// Block RFQ states, which select the Block RFQs GetBlockRFQs returns
const (
	BlockRFQStateOpen      = "open"
	BlockRFQStateFilled    = "filled"
	BlockRFQStateTraded    = "traded"
	BlockRFQStateCancelled = "cancelled"
	BlockRFQStateExpired   = "expired"
	BlockRFQStateClosed    = "closed"
)

// Times in force of a Block RFQ acceptance
const (
	BlockRFQTimeInForceFillOrKill       = "fill_or_kill"
	BlockRFQTimeInForceGoodTilCancelled = "good_til_cancelled"
)

// Execution instructions of a Block RFQ quote
const (
	BlockRFQExecutionAllOrNone = "all_or_none"
	BlockRFQExecutionAnyPartOf = "any_part_of"
)

// maximumBlockRFQLabelLength is the longest label, in characters, of a Block RFQ or a Block RFQ quote
const maximumBlockRFQLabelLength = 64

// blockRFQQuoteEPL limits quote additions, edits and cancellations, which Deribit's rate limits article counts as
// matching engine requests, although its API reference does not tag them so
const blockRFQQuoteEPL = matchingEPL

var (
	errBlockRFQAllocationTargetRequired    = errors.New("block RFQ trade allocation needs a user ID or client info")
	errBlockRFQIDRequired                  = errors.New("block RFQ ID is required")
	errBlockRFQLabelTooLong                = errors.New("block RFQ label must be at most 64 characters")
	errBlockRFQLegsRequired                = errors.New("at least one block RFQ leg is required")
	errBlockRFQQuoteIdentifierRequired     = errors.New("block RFQ quote ID, or block RFQ ID and label, is required")
	errInvalidBlockRFQExecutionInstruction = errors.New("invalid block RFQ execution instruction")
	errInvalidBlockRFQLegRatio             = errors.New("block RFQ leg ratio must be greater than zero")
	errInvalidBlockRFQQuoteCancellation    = errors.New("invalid block RFQ quote cancellation result")
	errInvalidBlockRFQRole                 = errors.New("invalid block RFQ role")
	errInvalidBlockRFQState                = errors.New("invalid block RFQ state")
	errInvalidBlockRFQTimeInForce          = errors.New("invalid block RFQ time in force")
)

// AcceptBlockRFQ accepts a Block RFQ's quotes as its taker with one crossing price, filling at once or leaving a trade
// trigger active until cancelled; for 5 seconds after creating a Block RFQ, its taker can neither see quotes nor trade
func (e *Exchange) AcceptBlockRFQ(ctx context.Context, arg *AcceptBlockRFQRequest) (*AcceptBlockRFQResponse, error) {
	return e.acceptBlockRFQ(ctx, arg, false)
}

func (e *Exchange) acceptBlockRFQ(ctx context.Context, arg *AcceptBlockRFQRequest, overWebsocket bool) (*AcceptBlockRFQResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.BlockRFQID == 0 {
		return nil, errBlockRFQIDRequired
	}
	if err := validateBlockRFQLegs(arg.Legs); err != nil {
		return nil, err
	}
	if arg.Amount <= 0 {
		return nil, fmt.Errorf("%w: %v", errInvalidAmount, arg.Amount)
	}
	if arg.Direction != DirectionBuy && arg.Direction != DirectionSell {
		return nil, fmt.Errorf("%w: %q", errInvalidDirection, arg.Direction)
	}
	if arg.TimeInForce != BlockRFQTimeInForceFillOrKill && arg.TimeInForce != BlockRFQTimeInForceGoodTilCancelled {
		return nil, fmt.Errorf("%w: %q", errInvalidBlockRFQTimeInForce, arg.TimeInForce)
	}
	if err := validateBlockRFQHedge(arg.Hedge); err != nil {
		return nil, err
	}
	var resp *AcceptBlockRFQResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/accept_block_rfq", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// AddBlockRFQQuote quotes a Block RFQ as a maker, pricing each leg; private/get_leg_prices derives leg prices from a
// structure's price
func (e *Exchange) AddBlockRFQQuote(ctx context.Context, arg *AddBlockRFQQuoteRequest) (*BlockRFQQuote, error) {
	return e.addBlockRFQQuote(ctx, arg, false)
}

func (e *Exchange) addBlockRFQQuote(ctx context.Context, arg *AddBlockRFQQuoteRequest, overWebsocket bool) (*BlockRFQQuote, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.BlockRFQID == 0 {
		return nil, errBlockRFQIDRequired
	}
	if arg.Amount <= 0 {
		return nil, fmt.Errorf("%w: %v", errInvalidAmount, arg.Amount)
	}
	if arg.Direction != DirectionBuy && arg.Direction != DirectionSell {
		return nil, fmt.Errorf("%w: %q", errInvalidDirection, arg.Direction)
	}
	if err := validateBlockRFQLegs(arg.Legs); err != nil {
		return nil, err
	}
	if err := validateBlockRFQHedge(arg.Hedge); err != nil {
		return nil, err
	}
	if err := validateBlockRFQLabel(arg.Label); err != nil {
		return nil, err
	}
	switch arg.ExecutionInstruction {
	case "", BlockRFQExecutionAllOrNone, BlockRFQExecutionAnyPartOf:
	default:
		return nil, fmt.Errorf("%w: %q", errInvalidBlockRFQExecutionInstruction, arg.ExecutionInstruction)
	}
	params := &AddBlockRFQQuoteParams{AddBlockRFQQuoteRequest: arg}
	if !arg.ExpiresAt.IsZero() {
		params.ExpiresAt = arg.ExpiresAt.UnixMilli()
	}
	var resp *BlockRFQQuote
	if err := e.sendRequest(ctx, overWebsocket, blockRFQQuoteEPL, "private/add_block_rfq_quote", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// CancelAllBlockRFQQuotes cancels the account's quotes on every Block RFQ, or on one, whatever their labels, and
// returns how many were cancelled, or with Detailed set, Deribit's reports of the cancellations
func (e *Exchange) CancelAllBlockRFQQuotes(ctx context.Context, arg *CancelAllBlockRFQQuotesRequest) (*CancelAllBlockRFQQuotesResponse, error) {
	return e.cancelAllBlockRFQQuotes(ctx, arg, false)
}

func (e *Exchange) cancelAllBlockRFQQuotes(ctx context.Context, arg *CancelAllBlockRFQQuotesRequest, overWebsocket bool) (*CancelAllBlockRFQQuotesResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	var resp *CancelAllBlockRFQQuotesResponse
	if err := e.sendRequest(ctx, overWebsocket, blockRFQQuoteEPL, "private/cancel_all_block_rfq_quotes", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// CancelBlockRFQ cancels a Block RFQ as its taker and returns it
func (e *Exchange) CancelBlockRFQ(ctx context.Context, blockRFQID uint64) (*BlockRFQ, error) {
	return e.cancelBlockRFQ(ctx, blockRFQID, "private/cancel_block_rfq", false)
}

// cancelBlockRFQ cancels a Block RFQ, or its trade trigger, with the given method and returns the Block RFQ
func (e *Exchange) cancelBlockRFQ(ctx context.Context, blockRFQID uint64, method string, overWebsocket bool) (*BlockRFQ, error) {
	if blockRFQID == 0 {
		return nil, errBlockRFQIDRequired
	}
	var resp *BlockRFQ
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, method, &BlockRFQIDRequest{BlockRFQID: blockRFQID}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// CancelBlockRFQQuote cancels one of the account's Block RFQ quotes, identified by its ID, or by its Block RFQ and
// label, and returns it
func (e *Exchange) CancelBlockRFQQuote(ctx context.Context, arg *CancelBlockRFQQuoteRequest) (*BlockRFQQuote, error) {
	return e.cancelBlockRFQQuote(ctx, arg, false)
}

func (e *Exchange) cancelBlockRFQQuote(ctx context.Context, arg *CancelBlockRFQQuoteRequest, overWebsocket bool) (*BlockRFQQuote, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if err := validateBlockRFQQuoteIdentifier(arg.BlockRFQQuoteID, arg.BlockRFQID, arg.Label); err != nil {
		return nil, err
	}
	var resp *BlockRFQQuote
	if err := e.sendRequest(ctx, overWebsocket, blockRFQQuoteEPL, "private/cancel_block_rfq_quote", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// CancelBlockRFQTrigger cancels the active trade trigger of a Block RFQ as its taker and returns the Block RFQ
func (e *Exchange) CancelBlockRFQTrigger(ctx context.Context, blockRFQID uint64) (*BlockRFQ, error) {
	return e.cancelBlockRFQ(ctx, blockRFQID, "private/cancel_block_rfq_trigger", false)
}

// CreateBlockRFQ creates a Block RFQ as its taker, asking the targeted makers, or every maker, to quote it
func (e *Exchange) CreateBlockRFQ(ctx context.Context, arg *CreateBlockRFQRequest) (*BlockRFQ, error) {
	return e.createBlockRFQ(ctx, arg, false)
}

func (e *Exchange) createBlockRFQ(ctx context.Context, arg *CreateBlockRFQRequest, overWebsocket bool) (*BlockRFQ, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if err := validateCreateBlockRFQLegs(arg.Legs); err != nil {
		return nil, err
	}
	if err := validateBlockRFQTradeAllocations(arg.TradeAllocations); err != nil {
		return nil, err
	}
	if err := validateBlockRFQHedge(arg.Hedge); err != nil {
		return nil, err
	}
	if err := validateBlockRFQLabel(arg.Label); err != nil {
		return nil, err
	}
	var resp *BlockRFQ
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/create_block_rfq", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// EditBlockRFQQuote replaces the amount, legs and prices of one of the account's Block RFQ quotes, identified by its
// ID, or by its Block RFQ and label, and returns it
func (e *Exchange) EditBlockRFQQuote(ctx context.Context, arg *EditBlockRFQQuoteRequest) (*BlockRFQQuote, error) {
	return e.editBlockRFQQuote(ctx, arg, false)
}

func (e *Exchange) editBlockRFQQuote(ctx context.Context, arg *EditBlockRFQQuoteRequest, overWebsocket bool) (*BlockRFQQuote, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if err := validateBlockRFQQuoteIdentifier(arg.BlockRFQQuoteID, arg.BlockRFQID, arg.Label); err != nil {
		return nil, err
	}
	if arg.Amount <= 0 {
		return nil, fmt.Errorf("%w: %v", errInvalidAmount, arg.Amount)
	}
	if err := validateBlockRFQLegs(arg.Legs); err != nil {
		return nil, err
	}
	if err := validateBlockRFQHedge(arg.Hedge); err != nil {
		return nil, err
	}
	var resp *BlockRFQQuote
	if err := e.sendRequest(ctx, overWebsocket, blockRFQQuoteEPL, "private/edit_block_rfq_quote", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetBlockRFQMakers returns the aliases of the makers a Block RFQ can target
func (e *Exchange) GetBlockRFQMakers(ctx context.Context) ([]string, error) {
	return e.getBlockRFQMakers(ctx, false)
}

func (e *Exchange) getBlockRFQMakers(ctx context.Context, overWebsocket bool) ([]string, error) {
	var resp []string
	return resp, e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_block_rfq_makers", nil, &resp)
}

// GetBlockRFQQuotes returns the account's open Block RFQ quotes, on every Block RFQ or as filtered
func (e *Exchange) GetBlockRFQQuotes(ctx context.Context, arg *GetBlockRFQQuotesRequest) ([]BlockRFQQuote, error) {
	return e.getBlockRFQQuotes(ctx, arg, false)
}

func (e *Exchange) getBlockRFQQuotes(ctx context.Context, arg *GetBlockRFQQuotesRequest, overWebsocket bool) ([]BlockRFQQuote, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	var resp []BlockRFQQuote
	return resp, e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_block_rfq_quotes", arg, &resp)
}

// GetBlockRFQUserInfo returns the Block RFQ aliases and taker ratings of the account group and its users
func (e *Exchange) GetBlockRFQUserInfo(ctx context.Context) (*BlockRFQUserInfoResponse, error) {
	return e.getBlockRFQUserInfo(ctx, false)
}

func (e *Exchange) getBlockRFQUserInfo(ctx context.Context, overWebsocket bool) (*BlockRFQUserInfoResponse, error) {
	var resp *BlockRFQUserInfoResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_block_rfq_user_info", nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetBlockRFQs returns the Block RFQs the account created or was asked to quote, newest first; a taker also sees the
// targeted makers and the label, and a maker sees its alias on the trades it took part in
func (e *Exchange) GetBlockRFQs(ctx context.Context, arg *GetBlockRFQsRequest) (*BlockRFQsResponse, error) {
	return e.getBlockRFQs(ctx, arg, false)
}

func (e *Exchange) getBlockRFQs(ctx context.Context, arg *GetBlockRFQsRequest, overWebsocket bool) (*BlockRFQsResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	switch arg.State {
	case "", BlockRFQStateOpen, BlockRFQStateFilled, BlockRFQStateTraded, BlockRFQStateCancelled, BlockRFQStateExpired, BlockRFQStateClosed:
	default:
		return nil, fmt.Errorf("%w: %q", errInvalidBlockRFQState, arg.State)
	}
	switch arg.Role {
	case "", BlockRFQRoleAny, BlockRFQRoleMaker, BlockRFQRoleTaker:
	default:
		return nil, fmt.Errorf("%w: %q", errInvalidBlockRFQRole, arg.Role)
	}
	var resp *BlockRFQsResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_block_rfqs", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetBlockRFQTrades returns recent public Block RFQ trades, newest first
func (e *Exchange) GetBlockRFQTrades(ctx context.Context, arg *GetBlockRFQTradesRequest) (*BlockRFQTradesResponse, error) {
	return e.getBlockRFQTrades(ctx, arg, false)
}

func (e *Exchange) getBlockRFQTrades(ctx context.Context, arg *GetBlockRFQTradesRequest, overWebsocket bool) (*BlockRFQTradesResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := *arg
	params.Currency = marketDataCurrency(arg.Currency)
	var resp *BlockRFQTradesResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "public/get_block_rfq_trades", &params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// validateBlockRFQLegs checks a quote or an acceptance has legs, each with an instrument, a direction and a ratio
func validateBlockRFQLegs(legs []BlockRFQLeg) error {
	if len(legs) == 0 {
		return errBlockRFQLegsRequired
	}
	for i := range legs {
		if legs[i].InstrumentName == "" {
			return fmt.Errorf("%w: leg %d", errInstrumentNameRequired, i)
		}
		if legs[i].Direction != DirectionBuy && legs[i].Direction != DirectionSell {
			return fmt.Errorf("%w: leg %d: %q", errInvalidDirection, i, legs[i].Direction)
		}
		if legs[i].Ratio == 0 {
			return fmt.Errorf("%w: leg %d", errInvalidBlockRFQLegRatio, i)
		}
	}
	return nil
}

// validateCreateBlockRFQLegs checks a new Block RFQ has legs, each with an instrument, a direction and an amount
func validateCreateBlockRFQLegs(legs []CreateBlockRFQLeg) error {
	if len(legs) == 0 {
		return errBlockRFQLegsRequired
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

// validateBlockRFQHedge checks a hedge leg, which may be left out, has an instrument, a direction and an amount
func validateBlockRFQHedge(hedge *BlockRFQHedge) error {
	if hedge == nil {
		return nil
	}
	if hedge.InstrumentName == "" {
		return fmt.Errorf("%w: hedge", errInstrumentNameRequired)
	}
	if hedge.Direction != DirectionBuy && hedge.Direction != DirectionSell {
		return fmt.Errorf("%w: hedge: %q", errInvalidDirection, hedge.Direction)
	}
	if hedge.Amount <= 0 {
		return fmt.Errorf("%w: hedge: %v", errInvalidAmount, hedge.Amount)
	}
	return nil
}

// validateBlockRFQTradeAllocations checks each trade allocation names a user or a broker client and has an amount
func validateBlockRFQTradeAllocations(allocations []BlockRFQTradeAllocation) error {
	for i := range allocations {
		if allocations[i].UserID == 0 && allocations[i].ClientInfo == nil {
			return fmt.Errorf("%w: allocation %d", errBlockRFQAllocationTargetRequired, i)
		}
		if allocations[i].Amount <= 0 {
			return fmt.Errorf("%w: allocation %d: %v", errInvalidAmount, i, allocations[i].Amount)
		}
	}
	return nil
}

// validateBlockRFQLabel checks a Block RFQ's or a quote's label is at most 64 characters
func validateBlockRFQLabel(label string) error {
	if utf8.RuneCountInString(label) > maximumBlockRFQLabelLength {
		return fmt.Errorf("%w: %q", errBlockRFQLabelTooLong, label)
	}
	return nil
}

// validateBlockRFQQuoteIdentifier checks a Block RFQ quote is identified by its ID, or by its Block RFQ and label
func validateBlockRFQQuoteIdentifier(blockRFQQuoteID, blockRFQID uint64, label string) error {
	if blockRFQQuoteID == 0 && (blockRFQID == 0 || label == "") {
		return errBlockRFQQuoteIdentifierRequired
	}
	return nil
}
