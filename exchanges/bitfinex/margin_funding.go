package bitfinex

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

// minimumFundingOfferPeriod is the shortest funding offer Bitfinex takes, in days
const minimumFundingOfferPeriod = 2

var (
	errInvalidFundingOfferType       = errors.New("invalid funding offer type, want LIMIT, FRRDELTAVAR or FRRDELTAFIX")
	errInvalidFundingOfferRate       = errors.New("invalid funding offer rate")
	errInvalidFundingOfferPeriod     = errors.New("invalid funding offer period, want at least 2 days")
	errFundingOfferIDRequired        = errors.New("funding offer ID required")
	errFundingLoanOrCreditIDRequired = errors.New("funding loan or credit ID required")
	errInvalidKeepFundingType        = errors.New("invalid keep funding type, want credit or loan")
)

// optionalFundingSymbolPath returns an endpoint path with a funding currency symbol, such as fUSD, between prefix and
// suffix, or with none, covering every currency, when symbol is empty
func optionalFundingSymbolPath(prefix, symbol, suffix string) (string, error) {
	if symbol == "" {
		return prefix + suffix, nil
	}
	if !isFundingSymbol(symbol) {
		return "", errFundingSymbolRequired
	}
	return prefix + "/" + symbol + suffix, nil
}

// GetActiveFundingOffers returns the account's active funding offers in a funding currency, such as fUSD, or in every
// currency when symbol is empty
func (e *Exchange) GetActiveFundingOffers(ctx context.Context, symbol string) ([]FundingOffer, error) {
	path, err := optionalFundingSymbolPath("auth/r/funding/offers", symbol, "")
	if err != nil {
		return nil, err
	}
	var resp []FundingOffer
	return resp, e.SendAuthenticatedHTTPRequest(ctx, fundingOffersEPL, path, nil, &resp)
}

// SubmitFundingOffer submits an offer to provide or take funding, returning the offer. A LIMIT offer's rate is fixed.
// FRRDELTAFIX offers at the flash return rate plus the offer's rate, fixed once taken; FRRDELTAVAR does so varying with
// the flash return rate even once taken, and offers at the flash return rate itself with a zero rate
func (e *Exchange) SubmitFundingOffer(ctx context.Context, req *SubmitFundingOfferRequest) (*Notification[FundingOffer], error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	body, err := req.body()
	if err != nil {
		return nil, err
	}
	return sendNotificationRequest[FundingOffer](ctx, e, submitFundingOfferEPL, "auth/w/funding/offer/submit", body)
}

// body validates the offer and returns the body submitting it, over REST or the websocket
func (r *SubmitFundingOfferRequest) body() (*submitFundingOfferBody, error) {
	switch r.Type {
	case FundingOfferTypeLimit, FundingOfferTypeFlashReturnRateDeltaFixed:
	case FundingOfferTypeFlashReturnRateDeltaVariable:
		if r.Rate < 0 {
			return nil, fmt.Errorf("%w: %s takes a rate offset of zero or more, not %v", errInvalidFundingOfferRate, r.Type, r.Rate)
		}
	default:
		return nil, fmt.Errorf("%w: %q", errInvalidFundingOfferType, r.Type)
	}
	if !isFundingSymbol(r.Symbol) {
		return nil, errFundingSymbolRequired
	}
	if r.Amount == 0 {
		return nil, errInvalidAmount
	}
	if r.Period < minimumFundingOfferPeriod {
		return nil, fmt.Errorf("%w: %d", errInvalidFundingOfferPeriod, r.Period)
	}
	return &submitFundingOfferBody{
		Type:   r.Type,
		Symbol: r.Symbol,
		Amount: strconv.FormatFloat(r.Amount, 'f', -1, 64),
		Rate:   strconv.FormatFloat(r.Rate, 'f', -1, 64),
		Period: r.Period,
		Flags:  r.Flags,
	}, nil
}

// CancelFundingOffer cancels a funding offer, as GetActiveFundingOffers returns it, returning the offer
func (e *Exchange) CancelFundingOffer(ctx context.Context, offerID uint64) (*Notification[FundingOffer], error) {
	if offerID == 0 {
		return nil, errFundingOfferIDRequired
	}
	return sendNotificationRequest[FundingOffer](ctx, e, cancelFundingOfferEPL, "auth/w/funding/offer/cancel", &fundingIDBody{ID: offerID})
}

// CancelAllFundingOffers cancels the account's funding offers in a currency, such as USD, or in every currency when c
// is empty. The cancellations are confirmed later; the notification's text gives how many were submitted
func (e *Exchange) CancelAllFundingOffers(ctx context.Context, c currency.Code) (*Notification[json.RawMessage], error) {
	return sendNotificationRequest[json.RawMessage](ctx, e, cancelAllFundingOffersEPL, "auth/w/funding/offer/cancel/all", &cancelAllFundingOffersBody{
		Currency: c.Upper().String(),
	})
}

// FundingClose returns funding taken, whether used in positions or not. It takes the ID of the funding's credit or
// loan, as GetFundingCredits or GetFundingLoans returns it, not that of the offer that took the funding
func (e *Exchange) FundingClose(ctx context.Context, id uint64) (*Notification[json.RawMessage], error) {
	if id == 0 {
		return nil, errFundingLoanOrCreditIDRequired
	}
	return sendNotificationRequest[json.RawMessage](ctx, e, fundingCloseEPL, "auth/w/funding/close", &fundingIDBody{ID: id})
}

// FundingAutoRenew turns auto-renew of a currency's funding on or off, returning the settings it applies
func (e *Exchange) FundingAutoRenew(ctx context.Context, req *FundingAutoRenewRequest) (*Notification[FundingAutoRenewSettings], error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	body := &fundingAutoRenewBody{Currency: req.Currency.Upper().String(), Period: req.Period}
	if req.Activate {
		body.Status = 1
	}
	if req.Amount != 0 {
		body.Amount = strconv.FormatFloat(req.Amount, 'f', -1, 64)
	}
	if req.Rate != 0 {
		body.Rate = strconv.FormatFloat(req.Rate, 'f', -1, 64)
	}
	return sendNotificationRequest[FundingAutoRenewSettings](ctx, e, fundingAutoRenewEPL, "auth/w/funding/auto", body)
}

// KeepFunding sets whether to keep funding the account has taken: its credits, which positions use, or its loans,
// which they do not. It toggles keeping each credit or loan in IDs, and sets keeping each in Changes
func (e *Exchange) KeepFunding(ctx context.Context, req *KeepFundingRequest) (*Notification[json.RawMessage], error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Type != KeepFundingTypeCredit && req.Type != KeepFundingTypeLoan {
		return nil, fmt.Errorf("%w: %q", errInvalidKeepFundingType, req.Type)
	}
	body := &keepFundingBody{Type: req.Type, IDs: req.IDs}
	if len(req.Changes) > 0 {
		body.Changes = make(map[uint64]uint64, len(req.Changes))
		for id, keep := range req.Changes {
			var status uint64
			if keep {
				status = 1
			}
			body.Changes[id] = status
		}
	}
	return sendNotificationRequest[json.RawMessage](ctx, e, keepFundingEPL, "auth/w/funding/keep", body)
}

// GetFundingOffersHistory returns the account's inactive funding offers, executed or cancelled, in a funding currency,
// or in every currency when the request's symbol is empty
func (e *Exchange) GetFundingOffersHistory(ctx context.Context, req *FundingOffersHistoryRequest) ([]FundingOffer, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	path, err := optionalFundingSymbolPath("auth/r/funding/offers", req.Symbol, "/hist")
	if err != nil {
		return nil, err
	}
	body, err := newHistoryBody(req.Start, req.End, req.Limit, 0)
	if err != nil {
		return nil, err
	}
	var resp []FundingOffer
	return resp, e.SendAuthenticatedHTTPRequest(ctx, fundingOffersHistoryEPL, path, &body, &resp)
}

// GetFundingLoans returns the account's active funding loans, funding not used in positions, in a funding currency,
// such as fUSD, or in every currency when symbol is empty
func (e *Exchange) GetFundingLoans(ctx context.Context, symbol string) ([]FundingCredit, error) {
	path, err := optionalFundingSymbolPath("auth/r/funding/loans", symbol, "")
	if err != nil {
		return nil, err
	}
	var resp []FundingCredit
	return resp, e.SendAuthenticatedHTTPRequest(ctx, fundingLoansEPL, path, nil, &resp)
}

// GetFundingLoansHistory returns the account's closed funding loans of the last 3 days in a funding currency, or in
// every currency when the request's symbol is empty
func (e *Exchange) GetFundingLoansHistory(ctx context.Context, req *FundingLoansHistoryRequest) ([]FundingCredit, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	path, err := optionalFundingSymbolPath("auth/r/funding/loans", req.Symbol, "/hist")
	if err != nil {
		return nil, err
	}
	body, err := newHistoryBody(req.Start, req.End, req.Limit, 0)
	if err != nil {
		return nil, err
	}
	var resp []FundingCredit
	return resp, e.SendAuthenticatedHTTPRequest(ctx, fundingLoansHistoryEPL, path, &body, &resp)
}

// GetFundingCredits returns the account's active funding credits, funding used in positions, in a funding currency,
// such as fUSD, or in every currency when symbol is empty
func (e *Exchange) GetFundingCredits(ctx context.Context, symbol string) ([]FundingCredit, error) {
	path, err := optionalFundingSymbolPath("auth/r/funding/credits", symbol, "")
	if err != nil {
		return nil, err
	}
	var resp []FundingCredit
	return resp, e.SendAuthenticatedHTTPRequest(ctx, fundingCreditsEPL, path, nil, &resp)
}

// GetFundingCreditsHistory returns the account's closed funding credits of the last 3 days in a funding currency, or
// in every currency when the request's symbol is empty
func (e *Exchange) GetFundingCreditsHistory(ctx context.Context, req *FundingCreditsHistoryRequest) ([]FundingCredit, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	path, err := optionalFundingSymbolPath("auth/r/funding/credits", req.Symbol, "/hist")
	if err != nil {
		return nil, err
	}
	body, err := newHistoryBody(req.Start, req.End, req.Limit, 0)
	if err != nil {
		return nil, err
	}
	var resp []FundingCredit
	return resp, e.SendAuthenticatedHTTPRequest(ctx, fundingCreditsHistoryEPL, path, &body, &resp)
}

// GetAccountFundingTrades returns the trades the account's funding offers made in a funding currency, or in every
// currency when the request's symbol is empty
func (e *Exchange) GetAccountFundingTrades(ctx context.Context, req *AccountFundingTradesRequest) ([]AccountFundingTrade, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	path, err := optionalFundingSymbolPath("auth/r/funding/trades", req.Symbol, "/hist")
	if err != nil {
		return nil, err
	}
	body, err := newHistoryBody(req.Start, req.End, req.Limit, 0)
	if err != nil {
		return nil, err
	}
	var resp []AccountFundingTrade
	return resp, e.SendAuthenticatedHTTPRequest(ctx, fundingTradesEPL, path, &body, &resp)
}

// GetFundingInfo returns the weighted average rate and duration of the funding the account takes and provides in a
// funding currency, such as fUSD
func (e *Exchange) GetFundingInfo(ctx context.Context, symbol string) (*FundingInfoResponse, error) {
	if !isFundingSymbol(symbol) {
		return nil, errFundingSymbolRequired
	}
	var resp *FundingInfoResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, fundingInfoEPL, "auth/r/info/funding/"+symbol, nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}
