package cryptodotcom

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// otcHistoryMaximumLimit is the largest page an OTC quote or quote request query returns
const otcHistoryMaximumLimit = 200

var (
	errOTCClientQuoteRequestIDRequired = errors.New("client quote request ID required")
	errOTCClientDealIDRequired         = errors.New("client deal ID required")
	errOTCQuoteIDRequired              = errors.New("quote ID required")
	errOTCQuoteRequestIDRequired       = errors.New("quote request ID required")
	errOTCLegsRequired                 = errors.New("OTC legs required")
	errOTCQuantityOrNotionalRequired   = errors.New("an OTC leg requires a quantity or a notional, not both")
	errInvalidOTCSettlementArrangement = errors.New("invalid OTC settlement arrangement")
	errInvalidOTCQuoteRequestDuration  = errors.New("invalid OTC quote request duration")
	errInvalidOTCQuoteTimeToLive       = errors.New("invalid OTC quote time to live")
	errInvalidOTCSettlementDate        = errors.New("invalid OTC settlement date")
)

// otcQuoteRequestDurations are the durations a quote request can stay active for, and otcQuoteTimesToLive the least
// times a quote can stay valid for
var (
	otcQuoteRequestDurations = []time.Duration{5 * time.Second, 10 * time.Second, 30 * time.Second, time.Minute, 5 * time.Minute, 10 * time.Minute}
	otcQuoteTimesToLive      = []time.Duration{4 * time.Second, 5 * time.Second}
)

// RequestOTCQuote requests quotes on one or more legs, which liquidity providers send on the user.otc_qr.quotes channel
// while the quote request is active, and returns the quote request's IDs
func (e *Exchange) RequestOTCQuote(ctx context.Context, req *OTCQuoteRequest) ([]OTCQuoteRequestResponse, error) {
	params, err := newOTCQuoteRequestParams(req)
	if err != nil {
		return nil, err
	}
	var resp *dataResponse[OTCQuoteRequestResponse]
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/otc/request-quote", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Data, nil
}

// GetOTCInstruments returns the instruments quotes can be requested on
func (e *Exchange) GetOTCInstruments(ctx context.Context) ([]OTCInstrumentResponse, error) {
	var resp *otcInstrumentListResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/otc/get-otc-instruments", nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp.InstrumentList, nil
}

// GetOTCOpenQuoteRequests returns the quote requests still new or active
func (e *Exchange) GetOTCOpenQuoteRequests(ctx context.Context, req *OTCQuoteRequestHistoryRequest) ([]OTCQuoteRequestResponse, error) {
	params, err := newOTCQuoteRequestHistoryParams(req)
	if err != nil {
		return nil, err
	}
	var resp *otcQuoteRequestListResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/otc/get-open-quote-requests", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp.QuoteRequestList, nil
}

// GetOTCOpenQuotes returns the quotes still active
func (e *Exchange) GetOTCOpenQuotes(ctx context.Context, req *OTCQuoteHistoryRequest) ([]OTCQuoteResponse, error) {
	params, err := newOTCQuoteHistoryParams(req)
	if err != nil {
		return nil, err
	}
	var resp *otcQuoteListResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/otc/get-open-quotes", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp.QuoteList, nil
}

// GetOTCQuoteHistory returns the quotes received
func (e *Exchange) GetOTCQuoteHistory(ctx context.Context, req *OTCQuoteHistoryRequest) ([]OTCQuoteResponse, error) {
	params, err := newOTCQuoteHistoryParams(req)
	if err != nil {
		return nil, err
	}
	var resp *otcQuoteListResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/otc/get-quote-history", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp.QuoteList, nil
}

// GetOTCQuoteRequestHistory returns the quote requests completed or rejected
func (e *Exchange) GetOTCQuoteRequestHistory(ctx context.Context, req *OTCQuoteRequestHistoryRequest) ([]OTCQuoteRequestResponse, error) {
	params, err := newOTCQuoteRequestHistoryParams(req)
	if err != nil {
		return nil, err
	}
	var resp *otcQuoteRequestListResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/otc/get-quote-request-history", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp.QuoteRequestList, nil
}

// RequestOTCDeal requests a deal on a quote, whose progress the user.otc.deals channel reports
func (e *Exchange) RequestOTCDeal(ctx context.Context, req *OTCDealRequest) (*OTCDealResponse, error) {
	params, err := newOTCDealParams(req)
	if err != nil {
		return nil, err
	}
	var resp *OTCDealResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/otc/request-deal", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetOTCOpenDeals returns the deals not yet settled or rejected
func (e *Exchange) GetOTCOpenDeals(ctx context.Context, req *OTCDealHistoryRequest) ([]OTCDealResponse, error) {
	params, err := newOTCDealHistoryParams(req)
	if err != nil {
		return nil, err
	}
	var resp *otcDealListResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/otc/get-open-deals", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp.DealList, nil
}

// GetOTCDealHistory returns the deals settled or rejected
func (e *Exchange) GetOTCDealHistory(ctx context.Context, req *OTCDealHistoryRequest) ([]OTCDealResponse, error) {
	params, err := newOTCDealHistoryParams(req)
	if err != nil {
		return nil, err
	}
	var resp *otcDealListResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/otc/get-deal-history", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp.DealList, nil
}

// GetOTCSettleLaterLimit returns the account's limit on deals settled later, and how much of it is used
func (e *Exchange) GetOTCSettleLaterLimit(ctx context.Context) (*OTCSettleLaterLimitResponse, error) {
	var resp *OTCSettleLaterLimitResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/otc/get-settle-later-limit", nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetOTCUnsettledAmounts returns the amounts pending settlement by settlement date and currency, ordered by date and
// then by currency
func (e *Exchange) GetOTCUnsettledAmounts(ctx context.Context) ([]OTCUnsettledAmountResponse, error) {
	var resp *otcUnsettledAmountsResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/otc/get-unsettled-amounts", nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp.amounts()
}

// amounts returns the unsettled amounts, one for each settlement date and currency, ordered by date and then by
// currency, as the map they arrive in has no order
func (r *otcUnsettledAmountsResponse) amounts() ([]OTCUnsettledAmountResponse, error) {
	amounts := make([]OTCUnsettledAmountResponse, 0, len(r.UnsettledCurrencies))
	for date, byCurrency := range r.UnsettledCurrencies {
		var settlementDate types.Time
		if err := settlementDate.UnmarshalJSON([]byte(date)); err != nil {
			return nil, fmt.Errorf("%w %q: %w", errInvalidOTCSettlementDate, date, err)
		}
		for code, amount := range byCurrency {
			amounts = append(amounts, OTCUnsettledAmountResponse{SettlementDate: settlementDate.Time(), Currency: code, Amount: amount})
		}
	}
	slices.SortFunc(amounts, func(a, b OTCUnsettledAmountResponse) int {
		if c := a.SettlementDate.Compare(b.SettlementDate); c != 0 {
			return c
		}
		return strings.Compare(a.Currency.String(), b.Currency.String())
	})
	return amounts, nil
}

// newOTCQuoteRequestParams checks a quote request and returns its params, which the REST and websocket forms send
func newOTCQuoteRequestParams(req *OTCQuoteRequest) (*otcQuoteRequestParams, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: %T", common.ErrNilPointer, req)
	}
	if req.ClientQuoteRequestID == "" {
		return nil, errOTCClientQuoteRequestIDRequired
	}
	if len(req.Legs) == 0 {
		return nil, errOTCLegsRequired
	}
	// Copied without the reported type, so legs taken from a reply can be sent back
	legs := make([]OTCQuoteRequestLeg, len(req.Legs))
	for i := range req.Legs {
		leg := &req.Legs[i]
		if err := validateOTCLeg(leg.InstrumentName, leg.Side, leg.Quantity, leg.Notional); err != nil {
			return nil, fmt.Errorf("leg %d: %w", i, err)
		}
		legs[i] = OTCQuoteRequestLeg{InstrumentName: leg.InstrumentName, Side: leg.Side, Quantity: leg.Quantity, Notional: leg.Notional}
	}
	switch req.SettlementArrangement {
	case "", OTCSettlementArrangementImmediate, OTCSettlementArrangementT1:
	default:
		return nil, fmt.Errorf("%w: %q", errInvalidOTCSettlementArrangement, req.SettlementArrangement)
	}
	duration, err := otcDurationParam(req.Duration, otcQuoteRequestDurations, errInvalidOTCQuoteRequestDuration)
	if err != nil {
		return nil, err
	}
	timeToLive, err := otcDurationParam(req.QuoteTimeToLive, otcQuoteTimesToLive, errInvalidOTCQuoteTimeToLive)
	if err != nil {
		return nil, err
	}
	return &otcQuoteRequestParams{
		ClientQuoteRequestID:  req.ClientQuoteRequestID,
		FirmQuote:             req.FirmQuote,
		SettlementArrangement: req.SettlementArrangement,
		Duration:              duration,
		QuoteTimeToLive:       timeToLive,
		Legs:                  legs,
	}, nil
}

// newOTCDealParams checks a deal request and returns its params, which the REST and websocket forms send
func newOTCDealParams(req *OTCDealRequest) (*otcDealParams, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: %T", common.ErrNilPointer, req)
	}
	if req.ClientDealID == "" {
		return nil, errOTCClientDealIDRequired
	}
	if req.QuoteID == "" {
		return nil, errOTCQuoteIDRequired
	}
	if req.QuoteRequestID == "" {
		return nil, errOTCQuoteRequestIDRequired
	}
	if len(req.Legs) == 0 {
		return nil, errOTCLegsRequired
	}
	for i := range req.Legs {
		leg := &req.Legs[i]
		if err := validateOTCLeg(leg.InstrumentName, leg.Side, leg.Quantity, leg.Notional); err != nil {
			return nil, fmt.Errorf("leg %d: %w", i, err)
		}
		if leg.Price <= 0 {
			return nil, fmt.Errorf("leg %d: %w", i, errInvalidPrice)
		}
	}
	return &otcDealParams{
		DealType:       OTCDealTypeQuoteRequest,
		ClientDealID:   req.ClientDealID,
		QuoteID:        req.QuoteID,
		QuoteRequestID: req.QuoteRequestID,
		Legs:           req.Legs,
	}, nil
}

// validateOTCLeg checks a quote request or deal leg's instrument and side, and that it trades a positive quantity or a
// positive notional but not both
func validateOTCLeg(instrumentName, side string, quantity, notional types.Number) error {
	if instrumentName == "" {
		return errInstrumentNameRequired
	}
	if side != SideBuy && side != SideSell {
		return fmt.Errorf("%w: %q", errInvalidSide, side)
	}
	if quantity < 0 {
		return errInvalidQuantity
	}
	if notional < 0 {
		return errInvalidAmount
	}
	if (quantity > 0) == (notional > 0) {
		return errOTCQuantityOrNotionalRequired
	}
	return nil
}

// otcDurationParam returns a duration as the milliseconds a request param takes, or zero, left unsent, for none; a
// duration not among those allowed is refused with the invalid error
func otcDurationParam(d time.Duration, allowed []time.Duration, invalid error) (uint64, error) {
	if d == 0 {
		return 0, nil
	}
	ms := d.Milliseconds()
	if ms <= 0 || !slices.Contains(allowed, d) {
		return 0, fmt.Errorf("%w: %s", invalid, d)
	}
	return uint64(ms), nil
}

// newOTCQuoteRequestHistoryParams checks a quote request query and returns its params
func newOTCQuoteRequestHistoryParams(req *OTCQuoteRequestHistoryRequest) (*otcQuoteRequestHistoryParams, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: %T", common.ErrNilPointer, req)
	}
	window, err := newOTCWindow(req.StartTime, req.EndTime, req.Limit)
	if err != nil {
		return nil, err
	}
	return &otcQuoteRequestHistoryParams{
		ClientQuoteRequestID: req.ClientQuoteRequestID,
		QuoteRequestID:       req.QuoteRequestID,
		StartTime:            window.start,
		EndTime:              window.end,
		Limit:                req.Limit,
	}, nil
}

// newOTCQuoteHistoryParams checks a quote query and returns its params
func newOTCQuoteHistoryParams(req *OTCQuoteHistoryRequest) (*otcQuoteHistoryParams, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: %T", common.ErrNilPointer, req)
	}
	window, err := newOTCWindow(req.StartTime, req.EndTime, req.Limit)
	if err != nil {
		return nil, err
	}
	return &otcQuoteHistoryParams{
		QuoteID:                  req.QuoteID,
		LiquidityProviderQuoteID: req.LiquidityProviderQuoteID,
		StartTime:                window.start,
		EndTime:                  window.end,
		Limit:                    req.Limit,
	}, nil
}

// newOTCDealHistoryParams checks a deal query and returns its params, which always name the only deal type
func newOTCDealHistoryParams(req *OTCDealHistoryRequest) (*otcDealHistoryParams, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: %T", common.ErrNilPointer, req)
	}
	window, err := newOTCWindow(req.StartTime, req.EndTime, 0)
	if err != nil {
		return nil, err
	}
	return &otcDealHistoryParams{DealType: OTCDealTypeQuoteRequest, DealID: req.DealID, StartTime: window.start, EndTime: window.end}, nil
}

// newOTCWindow checks an OTC query's time window and page size, and returns its window
func newOTCWindow(startTime, endTime time.Time, limit uint64) (otcWindow, error) {
	if err := validateOptionalTimeWindow(startTime, endTime); err != nil {
		return otcWindow{}, err
	}
	if limit > otcHistoryMaximumLimit {
		return otcWindow{}, fmt.Errorf("%w: limit %d is above %d", errInvalidCount, limit, otcHistoryMaximumLimit)
	}
	start, err := unixNanoParam(startTime)
	if err != nil {
		return otcWindow{}, err
	}
	end, err := unixNanoParam(endTime)
	if err != nil {
		return otcWindow{}, err
	}
	return otcWindow{start: start, end: end}, nil
}
