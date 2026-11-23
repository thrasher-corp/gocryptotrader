package okx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

var (
	errInvalidCounterParties          = errors.New("missing counter parties")
	errMissingRequestForQuoteID       = errors.New("missing RFQ ID")
	errMissingQuoteID                 = errors.New("missing quote ID")
	errMissingLegs                    = errors.New("missing legs")
	errTooManyLegs                    = errors.New("too many legs")
	errMissingLegSize                 = errors.New("missing leg size")
	errMissingLegPrice                = errors.New("missing leg price")
	errLegLimitPriceMismatch          = errors.New("a limit price must be set on every leg or on none")
	errMissingAllocationAccount       = errors.New("missing allocation account")
	errTooManyCancelIDs               = errors.New("too many IDs to cancel")
	errMissingMakerInstrumentSettings = errors.New("missing maker instrument settings")
	errInvalidMMPTimeInterval         = errors.New("invalid MMP time interval")
	errInvalidQuoteExpiry             = errors.New("invalid quote expiry")
)

const (
	maxRequestForQuoteLegs            = 15
	maxRequestForQuoteCancelIDs       = 100
	maxRequestForQuoteMMPTimeInterval = 600000 // milliseconds
	minQuoteExpirySeconds             = 10
	maxQuoteExpirySeconds             = 120
)

// GetCounterparties retrieves the counterparties the user is permitted to trade block trades with
func (e *Exchange) GetCounterparties(ctx context.Context) ([]CounterpartiesResponse, error) {
	var resp []CounterpartiesResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getCounterpartiesEPL, http.MethodGet, "rfq/counterparties", nil, &resp, request.AuthenticatedRequest)
}

// CreateRequestForQuote creates an RFQ and sends it to the counterparties
func (e *Exchange) CreateRequestForQuote(ctx context.Context, arg *CreateRequestForQuoteRequest) (*RequestForQuoteResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if err := arg.validate(); err != nil {
		return nil, err
	}
	var resp *RequestForQuoteResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, createRequestForQuoteEPL, http.MethodPost, "rfq/create-rfq", arg, &resp, request.AuthenticatedRequest)
}

func (arg *CreateRequestForQuoteRequest) validate() error {
	if len(arg.Counterparties) == 0 {
		return errInvalidCounterParties
	}
	switch {
	case len(arg.Legs) == 0:
		return errMissingLegs
	case len(arg.Legs) > maxRequestForQuoteLegs:
		return fmt.Errorf("%w: %d legs exceed the maximum of %d", errTooManyLegs, len(arg.Legs), maxRequestForQuoteLegs)
	}
	var limitPrices int
	for i := range arg.Legs {
		leg := &arg.Legs[i]
		if leg.InstrumentID == "" {
			return fmt.Errorf("%w: leg %d", errMissingInstrumentID, i)
		}
		if leg.Size <= 0 {
			return fmt.Errorf("%w: leg %d", errMissingLegSize, i)
		}
		if err := normaliseLegSide(&leg.Side); err != nil {
			return fmt.Errorf("%w: leg %d", err, i)
		}
		if leg.LimitPrice != 0 {
			limitPrices++
		}
	}
	if limitPrices != 0 && limitPrices != len(arg.Legs) {
		return fmt.Errorf("%w: %d of %d legs have one", errLegLimitPriceMismatch, limitPrices, len(arg.Legs))
	}
	for i := range arg.AccountAllocations {
		allocation := &arg.AccountAllocations[i]
		if allocation.Account == "" {
			return fmt.Errorf("%w: allocation %d", errMissingAllocationAccount, i)
		}
		if len(allocation.Legs) == 0 {
			return fmt.Errorf("%w: allocation to account %q", errMissingLegs, allocation.Account)
		}
		for j := range allocation.Legs {
			if allocation.Legs[j].InstrumentID == "" {
				return fmt.Errorf("%w: leg %d of the allocation to account %q", errMissingInstrumentID, j, allocation.Account)
			}
			if allocation.Legs[j].Size <= 0 {
				return fmt.Errorf("%w: leg %d of the allocation to account %q", errMissingLegSize, j, allocation.Account)
			}
		}
	}
	return nil
}

// normaliseLegSide lower cases a leg's side and checks it is buy or sell
func normaliseLegSide(side *string) error {
	*side = strings.ToLower(*side)
	if *side != order.Buy.Lower() && *side != order.Sell.Lower() {
		return fmt.Errorf("%w: %q", order.ErrSideIsInvalid, *side)
	}
	return nil
}

// CancelRequestForQuote cancels an active RFQ the user created
func (e *Exchange) CancelRequestForQuote(ctx context.Context, arg *CancelRequestForQuoteRequest) (*CancelRequestForQuoteResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.RequestForQuoteID == "" && arg.ClientRequestForQuoteID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp *CancelRequestForQuoteResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, cancelRequestForQuoteEPL, http.MethodPost, "rfq/cancel-rfq", arg, &resp, request.AuthenticatedRequest)
}

// CancelMultipleRequestForQuotes cancels up to 100 active RFQs in a single batch
func (e *Exchange) CancelMultipleRequestForQuotes(ctx context.Context, arg *CancelMultipleRequestForQuotesRequest) ([]CancelRequestForQuoteResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	switch n := len(arg.RequestForQuoteIDs) + len(arg.ClientRequestForQuoteIDs); {
	case n == 0:
		return nil, order.ErrOrderIDNotSet
	case n > maxRequestForQuoteCancelIDs:
		return nil, fmt.Errorf("%w: %d RFQ IDs exceed the maximum of %d", errTooManyCancelIDs, n, maxRequestForQuoteCancelIDs)
	}
	var resp []CancelRequestForQuoteResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, cancelMultipleRequestForQuotesEPL, http.MethodPost, "rfq/cancel-batch-rfqs", arg, &resp, request.AuthenticatedRequest)
}

// CancelAllRequestForQuotes cancels every active RFQ the user created
func (e *Exchange) CancelAllRequestForQuotes(ctx context.Context) (*BlockTradingTimestampResponse, error) {
	var resp *BlockTradingTimestampResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, cancelAllRequestForQuotesEPL, http.MethodPost, "rfq/cancel-all-rfqs", nil, &resp, request.AuthenticatedRequest)
}

// ExecuteQuote executes a quote; only the creator of the RFQ can execute its quotes
func (e *Exchange) ExecuteQuote(ctx context.Context, arg *ExecuteQuoteRequest) (*ExecuteQuoteResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.RequestForQuoteID == "" {
		return nil, errMissingRequestForQuoteID
	}
	if arg.QuoteID == "" {
		return nil, errMissingQuoteID
	}
	for i := range arg.Legs {
		if arg.Legs[i].InstrumentID == "" {
			return nil, fmt.Errorf("%w: leg %d", errMissingInstrumentID, i)
		}
		if arg.Legs[i].Size <= 0 {
			return nil, fmt.Errorf("%w: leg %d", errMissingLegSize, i)
		}
	}
	var resp *ExecuteQuoteResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, executeQuoteEPL, http.MethodPost, "rfq/execute-quote", arg, &resp, request.AuthenticatedRequest)
}

// GetQuoteProducts retrieves the products a maker quotes and receives RFQs for, with their price and size limits
func (e *Exchange) GetQuoteProducts(ctx context.Context) ([]QuoteProductResponse, error) {
	var resp []QuoteProductResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getQuoteProductsEPL, http.MethodGet, "rfq/maker-instrument-settings", nil, &resp, request.AuthenticatedRequest)
}

// SetQuoteProducts sets the products a maker quotes and receives RFQs for, with their price and size limits
func (e *Exchange) SetQuoteProducts(ctx context.Context, args []SetQuoteProductRequest) (*SetQuoteProductsResponse, error) {
	if len(args) == 0 {
		return nil, common.ErrEmptyParams
	}
	for x := range args {
		args[x].InstrumentType = strings.ToUpper(args[x].InstrumentType)
		if !slices.Contains([]string{instTypeSwap, instTypeSpot, instTypeFutures, instTypeOption}, args[x].InstrumentType) {
			return nil, fmt.Errorf("%w received %v", errInvalidInstrumentType, args[x].InstrumentType)
		}
		if len(args[x].Data) == 0 {
			return nil, errMissingMakerInstrumentSettings
		}
		for y := range args[x].Data {
			if args[x].InstrumentType == instTypeSpot {
				if args[x].Data[y].InstrumentID == "" {
					return nil, fmt.Errorf("%w, for instrument type %s", errMissingInstrumentID, args[x].InstrumentType)
				}
			} else if args[x].Data[y].InstrumentFamily == "" {
				return nil, fmt.Errorf("%w, for instrument type %s", errInstrumentFamilyRequired, args[x].InstrumentType)
			}
		}
	}
	var resp *SetQuoteProductsResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, setQuoteProductsEPL, http.MethodPost, "rfq/maker-instrument-settings", args, &resp, request.AuthenticatedRequest)
}

// ResetRequestForQuoteMMPStatus resets the block trading maker MMP status to inactive
func (e *Exchange) ResetRequestForQuoteMMPStatus(ctx context.Context) (*BlockTradingTimestampResponse, error) {
	var resp *BlockTradingTimestampResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, resetRequestForQuoteMMPEPL, http.MethodPost, "rfq/mmp-reset", nil, &resp, request.AuthenticatedRequest)
}

// SetRequestForQuoteMMP sets the block trading maker MMP configuration
func (e *Exchange) SetRequestForQuoteMMP(ctx context.Context, arg *SetRequestForQuoteMMPRequest) (*SetRequestForQuoteMMPResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.TimeInterval > maxRequestForQuoteMMPTimeInterval {
		return nil, fmt.Errorf("%w: %d milliseconds exceeds the maximum of %d", errInvalidMMPTimeInterval, arg.TimeInterval, maxRequestForQuoteMMPTimeInterval)
	}
	var resp *SetRequestForQuoteMMPResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, setRequestForQuoteMMPEPL, http.MethodPost, "rfq/mmp-config", arg, &resp, request.AuthenticatedRequest)
}

// GetRequestForQuoteMMPConfig retrieves the block trading maker MMP configuration and state
func (e *Exchange) GetRequestForQuoteMMPConfig(ctx context.Context) ([]RequestForQuoteMMPConfigResponse, error) {
	var resp []RequestForQuoteMMPConfigResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getRequestForQuoteMMPConfigEPL, http.MethodGet, "rfq/mmp-config", nil, &resp, request.AuthenticatedRequest)
}

// CreateQuote quotes an RFQ the user is a counterparty to. A quote covers the whole RFQ, every leg and its full size,
// and replaces any active quote the user has for the RFQ
func (e *Exchange) CreateQuote(ctx context.Context, arg *CreateQuoteRequest) (*QuoteResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.RequestForQuoteID == "" {
		return nil, errMissingRequestForQuoteID
	}
	arg.QuoteSide = strings.ToLower(arg.QuoteSide)
	if arg.QuoteSide != order.Buy.Lower() && arg.QuoteSide != order.Sell.Lower() {
		return nil, fmt.Errorf("%w: quote side %q", order.ErrSideIsInvalid, arg.QuoteSide)
	}
	if arg.ExpiresIn != 0 && (arg.ExpiresIn < minQuoteExpirySeconds || arg.ExpiresIn > maxQuoteExpirySeconds) {
		return nil, fmt.Errorf("%w: %d seconds is outside %d to %d", errInvalidQuoteExpiry, arg.ExpiresIn, minQuoteExpirySeconds, maxQuoteExpirySeconds)
	}
	if len(arg.Legs) == 0 {
		return nil, errMissingLegs
	}
	for i := range arg.Legs {
		leg := &arg.Legs[i]
		switch {
		case leg.InstrumentID == "":
			return nil, fmt.Errorf("%w: leg %d", errMissingInstrumentID, i)
		case leg.Size <= 0:
			return nil, fmt.Errorf("%w: leg %d", errMissingLegSize, i)
		case leg.Price <= 0:
			return nil, fmt.Errorf("%w: leg %d", errMissingLegPrice, i)
		}
		if err := normaliseLegSide(&leg.Side); err != nil {
			return nil, fmt.Errorf("%w: leg %d", err, i)
		}
	}
	var resp *QuoteResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, createQuoteEPL, http.MethodPost, "rfq/create-quote", arg, &resp, request.AuthenticatedRequest)
}

// CancelQuote cancels an active quote the user created in response to an RFQ
func (e *Exchange) CancelQuote(ctx context.Context, arg *CancelQuoteRequest) (*CancelQuoteResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.QuoteID == "" && arg.ClientQuoteID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp *CancelQuoteResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, cancelQuoteEPL, http.MethodPost, "rfq/cancel-quote", arg, &resp, request.AuthenticatedRequest)
}

// CancelMultipleQuotes cancels up to 100 active quotes in a single batch
func (e *Exchange) CancelMultipleQuotes(ctx context.Context, arg *CancelMultipleQuotesRequest) ([]CancelQuoteResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	switch n := len(arg.QuoteIDs) + len(arg.ClientQuoteIDs); {
	case n == 0:
		return nil, order.ErrOrderIDNotSet
	case n > maxRequestForQuoteCancelIDs:
		return nil, fmt.Errorf("%w: %d quote IDs exceed the maximum of %d", errTooManyCancelIDs, n, maxRequestForQuoteCancelIDs)
	}
	var resp []CancelQuoteResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, cancelMultipleQuotesEPL, http.MethodPost, "rfq/cancel-batch-quotes", arg, &resp, request.AuthenticatedRequest)
}

// CancelAllQuotes cancels every active quote the user created
func (e *Exchange) CancelAllQuotes(ctx context.Context) (*BlockTradingTimestampResponse, error) {
	var resp *BlockTradingTimestampResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, cancelAllQuotesEPL, http.MethodPost, "rfq/cancel-all-quotes", nil, &resp, request.AuthenticatedRequest)
}

// CancelAllQuotesAfterCountdown cancels every quote once the timeout elapses without another call; the timeout is
// whole seconds from 10 to 120, or 0 to disable the countdown
func (e *Exchange) CancelAllQuotesAfterCountdown(ctx context.Context, timeout time.Duration) (*CancelAllQuotesAfterCountdownResponse, error) {
	if timeout != 0 && (timeout < 10*time.Second || timeout > 120*time.Second) {
		return nil, fmt.Errorf("%w: %s must be 0 or from 10s to 120s", errCountdownTimeoutRequired, timeout)
	}
	arg := &CancelAllQuotesAfterCountdownRequest{Timeout: uint64(timeout / time.Second)} //nolint:gosec // timeout is checked above to be 0 or from 10s to 120s
	var resp *CancelAllQuotesAfterCountdownResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, cancelAllQuotesAfterCountdownEPL, http.MethodPost, "rfq/cancel-all-after", arg, &resp, request.AuthenticatedRequest)
}

// GetRequestForQuotes retrieves the RFQs the user is a counterparty to, as their creator or a receiver
func (e *Exchange) GetRequestForQuotes(ctx context.Context, arg *RequestForQuotesRequest) ([]RequestForQuoteResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	params := url.Values{}
	if arg.RequestForQuoteID != "" {
		params.Set("rfqId", arg.RequestForQuoteID)
	}
	if arg.ClientRequestForQuoteID != "" {
		params.Set("clRfqId", arg.ClientRequestForQuoteID)
	}
	if arg.State != "" {
		params.Set("state", strings.ToLower(arg.State))
	}
	if arg.BeginID != "" {
		params.Set("beginId", arg.BeginID)
	}
	if arg.EndID != "" {
		params.Set("endId", arg.EndID)
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []RequestForQuoteResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getRequestForQuotesEPL, http.MethodGet, common.EncodeURLValues("rfq/rfqs", params), nil, &resp, request.AuthenticatedRequest)
}

// GetQuotes retrieves the quotes the user is a counterparty to, as their creator or a receiver
func (e *Exchange) GetQuotes(ctx context.Context, arg *QuotesRequest) ([]QuoteResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	params := url.Values{}
	if arg.RequestForQuoteID != "" {
		params.Set("rfqId", arg.RequestForQuoteID)
	}
	if arg.ClientRequestForQuoteID != "" {
		params.Set("clRfqId", arg.ClientRequestForQuoteID)
	}
	if arg.QuoteID != "" {
		params.Set("quoteId", arg.QuoteID)
	}
	if arg.ClientQuoteID != "" {
		params.Set("clQuoteId", arg.ClientQuoteID)
	}
	if arg.State != "" {
		params.Set("state", strings.ToLower(arg.State))
	}
	if arg.BeginID != "" {
		params.Set("beginId", arg.BeginID)
	}
	if arg.EndID != "" {
		params.Set("endId", arg.EndID)
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []QuoteResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getQuotesEPL, http.MethodGet, common.EncodeURLValues("rfq/quotes", params), nil, &resp, request.AuthenticatedRequest)
}

// GetRequestForQuoteTrades retrieves the executed block trades the user is a counterparty to, as their creator or a
// receiver
func (e *Exchange) GetRequestForQuoteTrades(ctx context.Context, arg *RequestForQuoteTradesRequest) ([]RequestForQuoteTradeResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if !arg.BeginTime.IsZero() && !arg.EndTime.IsZero() && arg.BeginTime.After(arg.EndTime) {
		return nil, common.ErrStartAfterEnd
	}
	params := url.Values{}
	if arg.RequestForQuoteID != "" {
		params.Set("rfqId", arg.RequestForQuoteID)
	}
	if arg.ClientRequestForQuoteID != "" {
		params.Set("clRfqId", arg.ClientRequestForQuoteID)
	}
	if arg.QuoteID != "" {
		params.Set("quoteId", arg.QuoteID)
	}
	if arg.ClientQuoteID != "" {
		params.Set("clQuoteId", arg.ClientQuoteID)
	}
	if arg.BlockTradeID != "" {
		params.Set("blockTdId", arg.BlockTradeID)
	}
	if arg.BeginID != "" {
		params.Set("beginId", arg.BeginID)
	}
	if arg.EndID != "" {
		params.Set("endId", arg.EndID)
	}
	if !arg.BeginTime.IsZero() {
		params.Set("beginTs", strconv.FormatInt(arg.BeginTime.UnixMilli(), 10))
	}
	if !arg.EndTime.IsZero() {
		params.Set("endTs", strconv.FormatInt(arg.EndTime.UnixMilli(), 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	if arg.IsSuccessful != nil {
		params.Set("isSuccessful", strconv.FormatBool(*arg.IsSuccessful))
	}
	var resp []RequestForQuoteTradeResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getTradesEPL, http.MethodGet, common.EncodeURLValues("rfq/trades", params), nil, &resp, request.AuthenticatedRequest)
}

// GetBlockTickers retrieves the block trading volume of an instrument type's instruments over the last 24 hours
func (e *Exchange) GetBlockTickers(ctx context.Context, arg *GetBlockTickersRequest) ([]BlockTickerResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentType == "" {
		return nil, fmt.Errorf("%w, empty instrument type", errInvalidInstrumentType)
	}
	params := url.Values{}
	params.Set("instType", strings.ToUpper(arg.InstrumentType))
	if arg.InstrumentFamily != "" {
		params.Set("instFamily", arg.InstrumentFamily)
	}
	var resp []BlockTickerResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getBlockTickersEPL, http.MethodGet, common.EncodeURLValues("market/block-tickers", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetBlockTicker retrieves an instrument's block trading volume over the last 24 hours
func (e *Exchange) GetBlockTicker(ctx context.Context, instrumentID string) (*BlockTickerResponse, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	var resp *BlockTickerResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getBlockTickerEPL, http.MethodGet, common.EncodeURLValues("market/block-ticker", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetPublicRequestForQuoteTrades retrieves the executed multi-leg block trades, published 15 minutes after their
// execution
func (e *Exchange) GetPublicRequestForQuoteTrades(ctx context.Context, arg *GetPublicRequestForQuoteTradesRequest) ([]PublicTradesResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	params := url.Values{}
	if arg.BeginID != "" {
		params.Set("beginId", arg.BeginID)
	}
	if arg.EndID != "" {
		params.Set("endId", arg.EndID)
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []PublicTradesResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getPublicTradesEPL, http.MethodGet, common.EncodeURLValues("rfq/public-trades", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetPublicBlockTrades retrieves an instrument's executed block trades leg by leg, newest first, published 15 minutes
// after their execution
func (e *Exchange) GetPublicBlockTrades(ctx context.Context, instrumentID string) ([]PublicBlockTradeResponse, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	var resp []PublicBlockTradeResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getBlockTradesEPL, http.MethodGet, common.EncodeURLValues("public/block-trades", params), nil, &resp, request.UnauthenticatedRequest)
}
