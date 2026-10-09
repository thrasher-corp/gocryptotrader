package kraken

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// GetFuturesOpenRFQs calls List all open RFQs
func (e *Exchange) GetFuturesOpenRFQs(ctx context.Context) (*FuturesOpenRFQsResponse, error) {
	var resp *FuturesOpenRFQsResponse
	return resp, e.SendFuturesHTTPRequest(ctx, exchange.RestFutures, "/api/v3/rfqs", &resp)
}

// GetFuturesRFQ calls Retrieve a single RFQ (open or recently closed). Kraken holds closed RFQs in memory only, so a
// closed RFQ may be evicted or lost on a restart
func (e *Exchange) GetFuturesRFQ(ctx context.Context, rfqUID string) (*FuturesRFQResponse, error) {
	if rfqUID == "" {
		return nil, errFuturesRFQUIDEmpty
	}
	var resp *FuturesRFQResponse
	return resp, e.SendFuturesHTTPRequest(ctx, exchange.RestFutures, "/api/v3/rfqs/"+url.PathEscape(rfqUID), &resp)
}

// GetFuturesOpenRFQOffers calls List open offers on open RFQs, returning the account's offers on open RFQs
func (e *Exchange) GetFuturesOpenRFQOffers(ctx context.Context) (*FuturesRFQOffersResponse, error) {
	var resp *FuturesRFQOffersResponse
	return resp, e.SendFuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, "/api/v3/rfqs/open-offers", nil, nil, &resp)
}

// GetFuturesClosedRFQOffers calls List offers placed by the account on closed RFQs. Kraken holds closed RFQs in memory
// only, so an offer may be evicted or lost on a restart
func (e *Exchange) GetFuturesClosedRFQOffers(ctx context.Context) (*FuturesRFQOffersResponse, error) {
	var resp *FuturesRFQOffersResponse
	return resp, e.SendFuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, "/api/v3/rfqs/closed-offers", nil, nil, &resp)
}

// PlaceFuturesRFQOffer calls Place new offer on an open RFQ. An offer Kraken refuses returns an APIError holding its
// reason, such as insufficientMargin
func (e *Exchange) PlaceFuturesRFQOffer(ctx context.Context, req *FuturesPlaceRFQOfferRequest) (*FuturesPlaceRFQOfferResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.RFQUID == "" {
		return nil, errFuturesRFQUIDEmpty
	}
	perLeg := len(req.BidSide) != 0 || len(req.AskSide) != 0
	if perLeg && (req.Bid != nil || req.Ask != nil) {
		return nil, errFuturesRFQPricingConflict
	}
	if !perLeg && req.Bid == nil && req.Ask == nil {
		return nil, errFuturesRFQPriceEmpty
	}
	params := url.Values{}
	if req.Bid != nil {
		params.Set("bid", strconv.FormatFloat(*req.Bid, 'f', -1, 64))
	}
	if req.Ask != nil {
		params.Set("ask", strconv.FormatFloat(*req.Ask, 'f', -1, 64))
	}
	if len(req.BidSide) != 0 {
		bidSide, err := e.futuresRFQLegPrices(req.BidSide)
		if err != nil {
			return nil, err
		}
		params.Set("bidSide", bidSide)
	}
	if len(req.AskSide) != 0 {
		askSide, err := e.futuresRFQLegPrices(req.AskSide)
		if err != nil {
			return nil, err
		}
		params.Set("askSide", askSide)
	}
	var resp *FuturesPlaceRFQOfferResponse
	return resp, e.futuresRFQRequest(ctx, http.MethodPost, "/api/v3/rfqs/place-offer/"+url.PathEscape(req.RFQUID), params, &resp)
}

// CancelFuturesRFQOffer calls Cancel open offer on open RFQ. An offer or RFQ Kraken cannot find returns an APIError
// holding the reason, offerNotFound or rfqNotFound
func (e *Exchange) CancelFuturesRFQOffer(ctx context.Context, req *FuturesCancelRFQOfferRequest) error {
	if err := common.NilGuard(req); err != nil {
		return err
	}
	if req.RFQUID == "" {
		return errFuturesRFQUIDEmpty
	}
	params := url.Values{}
	if req.OfferUID != "" {
		params.Set("offerUid", req.OfferUID)
	}
	return e.futuresRFQRequest(ctx, http.MethodDelete, "/api/v3/rfqs/cancel-offer/"+url.PathEscape(req.RFQUID), params, nil)
}

// GetFuturesAccountOpenRFQs calls List open RFQs for account, returning the open RFQs the account created with the best
// offers on them
func (e *Exchange) GetFuturesAccountOpenRFQs(ctx context.Context) (*FuturesAccountOpenRFQsResponse, error) {
	var resp *FuturesAccountOpenRFQsResponse
	return resp, e.SendFuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, "/api/v3/rfqs/open-rfqs", nil, nil, &resp)
}

// CreateFuturesRFQ calls Create a new RFQ, requesting quotes on a package of contract positions. An RFQ Kraken refuses
// returns an APIError holding its reason, such as tooManyOpenRfqs
func (e *Exchange) CreateFuturesRFQ(ctx context.Context, req *FuturesCreateRFQRequest) (*FuturesRFQUIDResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if len(req.Legs) == 0 {
		return nil, errFuturesRFQLegsEmpty
	}
	if len(req.Legs) > 100 {
		return nil, fmt.Errorf("%w: %d legs", errFuturesRFQTooManyLegs, len(req.Legs))
	}
	creation := futuresRFQCreation{Legs: make([]futuresRFQCreationLeg, len(req.Legs))}
	seen := make(map[string]bool, len(req.Legs))
	for i := range req.Legs {
		if req.Legs[i].Contract.IsEmpty() {
			return nil, currency.ErrCurrencyPairEmpty
		}
		if req.Legs[i].Size == 0 {
			return nil, errFuturesRFQLegSizeZero
		}
		symbol, err := e.FormatSymbol(req.Legs[i].Contract, asset.Futures)
		if err != nil {
			return nil, err
		}
		if seen[symbol] {
			return nil, fmt.Errorf("%w: %s", errFuturesRFQDuplicateLeg, symbol)
		}
		seen[symbol] = true
		creation.Legs[i] = futuresRFQCreationLeg{Tradeable: symbol, Size: req.Legs[i].Size}
	}
	if !req.Expiry.IsZero() {
		creation.Expiry = req.Expiry.UTC().Format(time.RFC3339Nano)
	}
	data, err := json.Marshal(creation)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("json", string(data))
	var resp *FuturesRFQUIDResponse
	return resp, e.futuresRFQRequest(ctx, http.MethodPost, "/api/v3/rfqs/open-rfqs", params, &resp)
}

// CancelFuturesRFQ calls Cancel an open RFQ the account created. An RFQ Kraken cannot find returns an APIError holding
// the reason rfqNotFound
func (e *Exchange) CancelFuturesRFQ(ctx context.Context, rfqUID string) (*FuturesRFQUIDResponse, error) {
	if rfqUID == "" {
		return nil, errFuturesRFQUIDEmpty
	}
	var resp *FuturesRFQUIDResponse
	return resp, e.futuresRFQRequest(ctx, http.MethodDelete, "/api/v3/rfqs/open-rfqs/"+url.PathEscape(rfqUID), nil, &resp)
}

// AcceptFuturesRFQOffer calls Accept an offer on an open RFQ the account created, trading the package at an offer's bid
// or ask. An acceptance Kraken refuses returns an APIError holding its reason, such as offerNoLongerValid
func (e *Exchange) AcceptFuturesRFQOffer(ctx context.Context, req *FuturesAcceptRFQOfferRequest) (*FuturesRFQUIDResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.RFQUID == "" {
		return nil, errFuturesRFQUIDEmpty
	}
	if (req.Bid == nil) == (req.Ask == nil) {
		return nil, errFuturesRFQAcceptSide
	}
	params := url.Values{}
	if req.Bid != nil {
		params.Set("bidAccepted", strconv.FormatFloat(*req.Bid, 'f', -1, 64))
	} else {
		params.Set("askAccepted", strconv.FormatFloat(*req.Ask, 'f', -1, 64))
	}
	var resp *FuturesRFQUIDResponse
	return resp, e.futuresRFQRequest(ctx, http.MethodPost, "/api/v3/rfqs/open-rfqs/accept-offer/"+url.PathEscape(req.RFQUID), params, &resp)
}

// futuresRFQRequest sends a signed RFQ request and decodes its response into result, which may be nil. Kraken reports
// a refused request with a success result and a reason, which the shared error handling cannot see, so the reason is
// returned as an APIError
func (e *Exchange) futuresRFQRequest(ctx context.Context, method, path string, params url.Values, result any) error {
	var raw json.RawMessage
	if err := e.SendFuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, method, path, params, nil, &raw); err != nil {
		return err
	}
	var refusal futuresRFQRefusal
	if err := json.Unmarshal(raw, &refusal); err != nil {
		return err
	}
	if refusal.Reason != "" {
		return &APIError{Errors: []string{refusal.Reason}}
	}
	if result == nil {
		return nil
	}
	return json.Unmarshal(raw, result)
}

// futuresRFQLegPrices returns per-leg offer prices as the JSON document Place new offer on an open RFQ takes
func (e *Exchange) futuresRFQLegPrices(legs []FuturesRFQOfferLeg) (string, error) {
	prices := make([]FuturesRFQLegPrice, len(legs))
	for i := range legs {
		if legs[i].Contract.IsEmpty() {
			return "", currency.ErrCurrencyPairEmpty
		}
		symbol, err := e.FormatSymbol(legs[i].Contract, asset.Futures)
		if err != nil {
			return "", err
		}
		prices[i] = FuturesRFQLegPrice{Symbol: symbol, Price: legs[i].Price}
	}
	data, err := json.Marshal(prices)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
