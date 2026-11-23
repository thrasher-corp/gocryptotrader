package okx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

var (
	errIndexComponentNotFound               = errors.New("unable to fetch index components")
	errIndexRequired                        = errors.New("index is required")
	errLimitValueExceedsMaxOf100            = errors.New("limit value exceeds the maximum value 100")
	errInvalidTradeMode                     = errors.New("unacceptable required argument, trade mode")
	errMissingQuantity                      = errors.New("invalid quantity to buy or sell")
	errInstrumentFamilyOrUnderlyingRequired = errors.New("either underlying or instrument family is required")
	errSeriesIDRequired                     = errors.New("series ID is required")
	errTickerAndInstrumentIDExclusive       = errors.New("ticker and instrument ID are mutually exclusive")
	errModuleRequired                       = errors.New("module is required")
	errDateAggregationTypeRequired          = errors.New("date aggregation type is required")
)

const (
	maxPositionTiersFilterEntries        = 5
	maxHistoricalMarketDataFilterEntries = 10
)

// GetInstruments retrieves a list of instruments with open contracts
func (e *Exchange) GetInstruments(ctx context.Context, arg *InstrumentsRequest) ([]InstrumentResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentType == "" {
		return nil, errInvalidInstrumentType
	}
	switch arg.InstrumentType {
	case instTypeOption:
		if arg.InstrumentFamily == "" && arg.Underlying == "" {
			return nil, errInstrumentFamilyOrUnderlyingRequired
		}
	case instTypeEvents:
		if arg.SeriesID == "" {
			return nil, errSeriesIDRequired
		}
	}
	params := url.Values{}
	params.Set("instType", arg.InstrumentType)
	if arg.SeriesID != "" {
		params.Set("seriesId", arg.SeriesID)
	}
	// OKX documents only instFamily, but an underlying such as BTC-USD spans several families (BTC-USD and
	// BTC-USD_UM), so querying by underlying still needs the undocumented uly, which OKX continues to honour
	if arg.Underlying != "" {
		params.Set("uly", arg.Underlying)
	}
	if arg.InstrumentFamily != "" {
		params.Set("instFamily", arg.InstrumentFamily)
	}
	if arg.InstrumentID != "" {
		params.Set("instId", arg.InstrumentID)
	}
	var resp []InstrumentResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getInstrumentsEPL, http.MethodGet, common.EncodeURLValues("public/instruments", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetEventContractSeries retrieves the prediction market series, or one series
func (e *Exchange) GetEventContractSeries(ctx context.Context, arg *EventContractSeriesRequest) ([]EventContractSeriesResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	params := url.Values{}
	if arg.SeriesID != "" {
		params.Set("seriesId", arg.SeriesID)
	}
	var resp []EventContractSeriesResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getEventContractSeriesEPL, http.MethodGet, common.EncodeURLValues("public/event-contract/series", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetEventContractEvents retrieves a prediction market series' events, expired ones included, newest expiry first
func (e *Exchange) GetEventContractEvents(ctx context.Context, arg *EventContractEventsRequest) ([]EventContractEventResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.SeriesID == "" {
		return nil, errSeriesIDRequired
	}
	if !arg.Before.IsZero() && !arg.After.IsZero() && arg.Before.After(arg.After) {
		return nil, common.ErrStartAfterEnd
	}
	params := url.Values{}
	params.Set("seriesId", arg.SeriesID)
	if arg.EventID != "" {
		params.Set("eventId", arg.EventID)
	}
	if arg.State != "" {
		params.Set("state", arg.State)
	}
	setTimeWindowParams(params, arg.Before, arg.After)
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []EventContractEventResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getEventContractEventsEPL, http.MethodGet, common.EncodeURLValues("public/event-contract/events", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetCorporateActions retrieves the corporate action events of the stocks underlying OKX instruments
func (e *Exchange) GetCorporateActions(ctx context.Context, arg *CorporateActionsRequest) (*CorporateActionsResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Ticker != "" && arg.InstrumentID != "" {
		return nil, errTickerAndInstrumentIDExclusive
	}
	params := url.Values{}
	if arg.Ticker != "" {
		params.Set("ticker", arg.Ticker)
	}
	if arg.InstrumentID != "" {
		params.Set("instId", arg.InstrumentID)
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	if arg.After != "" {
		params.Set("after", arg.After)
	}
	if arg.Before != "" {
		params.Set("before", arg.Before)
	}
	var resp *CorporateActionsResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getCorporateActionsEPL, http.MethodGet, common.EncodeURLValues("public/corporate-actions", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetEventContractMarkets retrieves the markets of a prediction market series' events, newest expiry first
func (e *Exchange) GetEventContractMarkets(ctx context.Context, arg *EventContractMarketsRequest) ([]EventContractMarketResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.SeriesID == "" {
		return nil, errSeriesIDRequired
	}
	if !arg.Before.IsZero() && !arg.After.IsZero() && arg.Before.After(arg.After) {
		return nil, common.ErrStartAfterEnd
	}
	params := url.Values{}
	params.Set("seriesId", arg.SeriesID)
	if arg.EventID != "" {
		params.Set("eventId", arg.EventID)
	}
	if arg.InstrumentID != "" {
		params.Set("instId", arg.InstrumentID)
	}
	if arg.State != "" {
		params.Set("state", arg.State)
	}
	setTimeWindowParams(params, arg.Before, arg.After)
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []EventContractMarketResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getEventContractMarketsEPL, http.MethodGet, common.EncodeURLValues("public/event-contract/markets", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetEstimatedDeliveryPrice retrieves a futures, option or event contract's estimated delivery or exercise price, which
// OKX only sets during the 30 minutes before delivery or exercise
func (e *Exchange) GetEstimatedDeliveryPrice(ctx context.Context, instrumentID string) (*EstimatedDeliveryPriceResponse, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	var resp *EstimatedDeliveryPriceResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getEstimatedDeliveryExercisePriceEPL, http.MethodGet, common.EncodeURLValues("public/estimated-price", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetDeliveryHistory retrieves the futures deliveries and option exercises of the last 3 months
func (e *Exchange) GetDeliveryHistory(ctx context.Context, arg *DeliveryHistoryRequest) ([]DeliveryHistoryResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentType == "" {
		return nil, errInvalidInstrumentType
	}
	if arg.InstrumentFamily == "" {
		return nil, errInstrumentFamilyRequired
	}
	if arg.Limit > 100 {
		return nil, errLimitValueExceedsMaxOf100
	}
	if !arg.Before.IsZero() && !arg.After.IsZero() && arg.Before.After(arg.After) {
		return nil, common.ErrStartAfterEnd
	}
	params := url.Values{}
	params.Set("instType", arg.InstrumentType)
	params.Set("instFamily", arg.InstrumentFamily)
	setTimeWindowParams(params, arg.Before, arg.After)
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []DeliveryHistoryResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getDeliveryExerciseHistoryEPL, http.MethodGet, common.EncodeURLValues("public/delivery-exercise-history", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetEstimatedSettlementPrice retrieves a futures contract's estimated settlement price, which OKX only sets during the
// 30 minutes before settlement
func (e *Exchange) GetEstimatedSettlementPrice(ctx context.Context, instrumentID string) (*EstimatedSettlementPriceResponse, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	var resp *EstimatedSettlementPriceResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getEstimatedSettlementPriceEPL, http.MethodGet, common.EncodeURLValues("public/estimated-settlement-info", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetFuturesSettlementHistory retrieves an instrument family's futures settlements of the last 3 months
func (e *Exchange) GetFuturesSettlementHistory(ctx context.Context, arg *FuturesSettlementHistoryRequest) ([]FuturesSettlementHistoryResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentFamily == "" {
		return nil, errInstrumentFamilyRequired
	}
	if !arg.Before.IsZero() && !arg.After.IsZero() && arg.Before.After(arg.After) {
		return nil, common.ErrStartAfterEnd
	}
	params := url.Values{}
	params.Set("instFamily", arg.InstrumentFamily)
	setTimeWindowParams(params, arg.Before, arg.After)
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []FuturesSettlementHistoryResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getFuturesSettlementHistoryEPL, http.MethodGet, common.EncodeURLValues("public/settlement-history", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetFundingRate retrieves the funding rate of a perpetual swap or X-Perps futures contract, or of every one of them
// when instrumentID is ANY
func (e *Exchange) GetFundingRate(ctx context.Context, instrumentID string) ([]FundingRateResponse, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	var resp []FundingRateResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getFundingEPL, http.MethodGet, common.EncodeURLValues("public/funding-rate", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetFundingRateHistory retrieves a perpetual swap's or X-Perps futures contract's funding rates of the last 3 months.
// OKX excludes funding times equal to either bound
func (e *Exchange) GetFundingRateHistory(ctx context.Context, arg *FundingRateHistoryRequest) ([]FundingRateResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	if !arg.Before.IsZero() && !arg.After.IsZero() && arg.Before.After(arg.After) {
		return nil, common.ErrStartAfterEnd
	}
	params := url.Values{}
	params.Set("instId", arg.InstrumentID)
	setTimeWindowParams(params, arg.Before, arg.After)
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []FundingRateResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getFundingRateHistoryEPL, http.MethodGet, common.EncodeURLValues("public/funding-rate-history", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetOpenInterestData retrieves the total open interest of contracts. An option query must name its instrument family;
// an underlying spans several families (BTC-USD spans BTC-USD and BTC-USD_UM), so it cannot substitute for one
func (e *Exchange) GetOpenInterestData(ctx context.Context, arg *OpenInterestRequest) ([]OpenInterestResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentType == "" {
		return nil, errInvalidInstrumentType
	}
	if arg.InstrumentType == instTypeOption && arg.InstrumentFamily == "" {
		return nil, errInstrumentFamilyRequired
	}
	params := url.Values{}
	params.Set("instType", arg.InstrumentType)
	if arg.InstrumentFamily != "" {
		params.Set("instFamily", arg.InstrumentFamily)
	}
	if arg.InstrumentID != "" {
		params.Set("instId", arg.InstrumentID)
	}
	var resp []OpenInterestResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getOpenInterestEPL, http.MethodGet, common.EncodeURLValues("public/open-interest", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetLimitPrice retrieves an instrument's highest buy limit and lowest sell limit
func (e *Exchange) GetLimitPrice(ctx context.Context, instrumentID string) (*LimitPriceResponse, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	var resp *LimitPriceResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getLimitPriceEPL, http.MethodGet, common.EncodeURLValues("public/price-limit", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetOptionMarketData retrieves the market data of an option instrument family's options
func (e *Exchange) GetOptionMarketData(ctx context.Context, arg *OptionMarketDataRequest) ([]OptionMarketDataResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentFamily == "" {
		return nil, errInstrumentFamilyRequired
	}
	params := url.Values{}
	params.Set("instFamily", arg.InstrumentFamily)
	if !arg.ExpiryTime.IsZero() {
		params.Set("expTime", arg.ExpiryTime.UTC().Format("060102"))
	}
	var resp []OptionMarketDataResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getOptionMarketDateEPL, http.MethodGet, common.EncodeURLValues("public/opt-summary", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetDiscountRateAndInterestFreeQuota retrieves the collateral discount rate tiers and interest-free quota of every
// currency, or of one currency
func (e *Exchange) GetDiscountRateAndInterestFreeQuota(ctx context.Context, arg *DiscountRateRequest) ([]DiscountRateResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	params := url.Values{}
	if !arg.Currency.IsEmpty() {
		params.Set("ccy", arg.Currency.String())
	}
	if arg.DiscountLevel != "" {
		params.Set("discountLv", arg.DiscountLevel)
	}
	var resp []DiscountRateResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getDiscountRateAndInterestFreeQuotaEPL, http.MethodGet, common.EncodeURLValues("public/discount-rate-interest-free-quota", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetSystemTime retrieves the API server time
func (e *Exchange) GetSystemTime(ctx context.Context) (time.Time, error) {
	var resp *SystemTimeResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, getSystemTimeEPL, http.MethodGet, "public/time", nil, &resp, request.UnauthenticatedRequest); err != nil {
		return time.Time{}, err
	}
	if resp == nil {
		return time.Time{}, common.ErrNoResponse
	}
	return resp.Timestamp.Time(), nil
}

// GetMarkPrice retrieves the mark prices of an instrument type's instruments
func (e *Exchange) GetMarkPrice(ctx context.Context, arg *MarkPriceRequest) ([]MarkPriceResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentType == "" {
		return nil, errInvalidInstrumentType
	}
	params := url.Values{}
	params.Set("instType", arg.InstrumentType)
	if arg.InstrumentFamily != "" {
		params.Set("instFamily", arg.InstrumentFamily)
	}
	if arg.InstrumentID != "" {
		params.Set("instId", arg.InstrumentID)
	}
	var resp []MarkPriceResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getMarkPriceEPL, http.MethodGet, common.EncodeURLValues("public/mark-price", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetPositionTiers retrieves position tiers; the maximum leverage depends on the borrowings and maintenance margin ratio
func (e *Exchange) GetPositionTiers(ctx context.Context, arg *PositionTiersRequest) ([]PositionTierResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentType == "" {
		return nil, errInvalidInstrumentType
	}
	if arg.TradeMode == "" {
		return nil, errInvalidTradeMode
	}
	switch arg.InstrumentType {
	case instTypeSwap, instTypeFutures, instTypeOption:
		if len(arg.InstrumentFamilies) == 0 {
			return nil, errInstrumentFamilyRequired
		}
	case instTypeMargin:
		if len(arg.InstrumentIDs) == 0 && arg.Currency.IsEmpty() {
			return nil, errEitherInstIDOrCcyIsRequired
		}
	}
	if len(arg.InstrumentFamilies) > maxPositionTiersFilterEntries {
		return nil, fmt.Errorf("%w: %d instrument families exceed the maximum of %d", errExceedLimit, len(arg.InstrumentFamilies), maxPositionTiersFilterEntries)
	}
	if len(arg.InstrumentIDs) > maxPositionTiersFilterEntries {
		return nil, fmt.Errorf("%w: %d instrument IDs exceed the maximum of %d", errExceedLimit, len(arg.InstrumentIDs), maxPositionTiersFilterEntries)
	}
	params := url.Values{}
	params.Set("instType", arg.InstrumentType)
	params.Set("tdMode", arg.TradeMode)
	if len(arg.InstrumentFamilies) > 0 {
		params.Set("instFamily", strings.Join(arg.InstrumentFamilies, ","))
	}
	if len(arg.InstrumentIDs) > 0 {
		params.Set("instId", strings.Join(arg.InstrumentIDs, ","))
	}
	if !arg.Currency.IsEmpty() {
		params.Set("ccy", arg.Currency.String())
	}
	if arg.Tier > 0 {
		params.Set("tier", strconv.FormatUint(arg.Tier, 10))
	}
	var resp []PositionTierResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getPositionTiersEPL, http.MethodGet, common.EncodeURLValues("public/position-tiers", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetInterestRateAndLoanQuota retrieves the borrowing interest rates and loan quotas
func (e *Exchange) GetInterestRateAndLoanQuota(ctx context.Context) (*InterestRateLoanQuotaResponse, error) {
	var resp *InterestRateLoanQuotaResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getInterestRateAndLoanQuotaEPL, http.MethodGet, "public/interest-rate-loan-quota", nil, &resp, request.UnauthenticatedRequest)
}

// GetPublicUnderlyings retrieves the underlyings of an instrument type
func (e *Exchange) GetPublicUnderlyings(ctx context.Context, instrumentType string) ([]string, error) {
	if instrumentType == "" {
		return nil, errInvalidInstrumentType
	}
	params := url.Values{}
	params.Set("instType", instrumentType)
	var resp []string
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getUnderlyingEPL, http.MethodGet, common.EncodeURLValues("public/underlying", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetInsuranceFundInformation retrieves the security fund balance of an instrument family, or of a margin currency
func (e *Exchange) GetInsuranceFundInformation(ctx context.Context, arg *InsuranceFundRequest) (*InsuranceFundResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentType == "" {
		return nil, errInvalidInstrumentType
	}
	switch arg.InstrumentType {
	case instTypeFutures, instTypeSwap, instTypeOption:
		if arg.InstrumentFamily == "" {
			return nil, errInstrumentFamilyRequired
		}
	case instTypeMargin:
		if arg.Currency.IsEmpty() {
			return nil, currency.ErrCurrencyCodeEmpty
		}
	}
	if !arg.Before.IsZero() && !arg.After.IsZero() && arg.Before.After(arg.After) {
		return nil, common.ErrStartAfterEnd
	}
	params := url.Values{}
	params.Set("instType", arg.InstrumentType)
	if arg.Type != "" {
		params.Set("type", arg.Type)
	}
	if arg.InstrumentFamily != "" {
		params.Set("instFamily", arg.InstrumentFamily)
	}
	if !arg.Currency.IsEmpty() {
		params.Set("ccy", arg.Currency.String())
	}
	setTimeWindowParams(params, arg.Before, arg.After)
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp *InsuranceFundResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getInsuranceFundEPL, http.MethodGet, common.EncodeURLValues("public/insurance-fund", params), nil, &resp, request.UnauthenticatedRequest)
}

// CurrencyUnitConvert converts a currency amount to a number of contracts, or a number of contracts to a currency amount
func (e *Exchange) CurrencyUnitConvert(ctx context.Context, arg *UnitConvertRequest) (*UnitConvertResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	if arg.Size <= 0 {
		return nil, errMissingQuantity
	}
	params := url.Values{}
	if arg.ConvertType != "" {
		params.Set("type", arg.ConvertType)
	}
	params.Set("instId", arg.InstrumentID)
	params.Set("sz", strconv.FormatFloat(arg.Size, 'f', -1, 64))
	if arg.Price > 0 {
		params.Set("px", strconv.FormatFloat(arg.Price, 'f', -1, 64))
	}
	if arg.Unit != "" {
		params.Set("unit", arg.Unit)
	}
	if arg.OperationType != "" {
		params.Set("opType", arg.OperationType)
	}
	var resp *UnitConvertResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, unitConvertEPL, http.MethodGet, common.EncodeURLValues("public/convert-contract-coin", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetInstrumentTickBands retrieves the tick size bands of option instrument families or event contract series
func (e *Exchange) GetInstrumentTickBands(ctx context.Context, arg *InstrumentTickBandsRequest) ([]InstrumentTickBandResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentType == "" {
		return nil, errInvalidInstrumentType
	}
	params := url.Values{}
	params.Set("instType", arg.InstrumentType)
	if arg.InstrumentFamily != "" {
		params.Set("instFamily", arg.InstrumentFamily)
	}
	var resp []InstrumentTickBandResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, optionTickBandsEPL, http.MethodGet, common.EncodeURLValues("public/instrument-tick-bands", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetPremiumHistory retrieves a perpetual swap's premium index history of the last 6 months
func (e *Exchange) GetPremiumHistory(ctx context.Context, arg *PremiumHistoryRequest) ([]PremiumHistoryResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	if !arg.Before.IsZero() && !arg.After.IsZero() && arg.Before.After(arg.After) {
		return nil, common.ErrStartAfterEnd
	}
	params := url.Values{}
	params.Set("instId", arg.InstrumentID)
	setTimeWindowParams(params, arg.Before, arg.After)
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []PremiumHistoryResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getPremiumHistoryEPL, http.MethodGet, common.EncodeURLValues("public/premium-history", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetIndexTickers retrieves the index tickers of a quote currency, or of one index
func (e *Exchange) GetIndexTickers(ctx context.Context, arg *IndexTickersRequest) ([]IndexTickerResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentID == "" && arg.QuoteCurrency.IsEmpty() {
		return nil, errEitherInstIDOrCcyIsRequired
	}
	params := url.Values{}
	if !arg.QuoteCurrency.IsEmpty() {
		params.Set("quoteCcy", arg.QuoteCurrency.String())
	}
	if arg.InstrumentID != "" {
		params.Set("instId", arg.InstrumentID)
	}
	var resp []IndexTickerResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getIndexTickersEPL, http.MethodGet, common.EncodeURLValues("market/index-tickers", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetIndexCandlesticks retrieves an index's latest 1,440 candlesticks
func (e *Exchange) GetIndexCandlesticks(ctx context.Context, arg *PriceCandlesticksRequest) ([]PriceCandlestickResponse, error) {
	return e.getPriceCandlesticks(ctx, "market/index-candles", getIndexCandlesticksEPL, arg)
}

// GetIndexCandlesticksHistory retrieves an index's candlesticks from recent years
func (e *Exchange) GetIndexCandlesticksHistory(ctx context.Context, arg *PriceCandlesticksRequest) ([]PriceCandlestickResponse, error) {
	return e.getPriceCandlesticks(ctx, "market/history-index-candles", getIndexCandlesticksHistoryEPL, arg)
}

// GetMarkPriceCandlesticks retrieves an instrument's latest 1,440 mark price candlesticks
func (e *Exchange) GetMarkPriceCandlesticks(ctx context.Context, arg *PriceCandlesticksRequest) ([]PriceCandlestickResponse, error) {
	return e.getPriceCandlesticks(ctx, "market/mark-price-candles", getMarkPriceCandlesticksEPL, arg)
}

// GetMarkPriceCandlesticksHistory retrieves an instrument's mark price candlesticks from recent years
func (e *Exchange) GetMarkPriceCandlesticksHistory(ctx context.Context, arg *PriceCandlesticksRequest) ([]PriceCandlestickResponse, error) {
	return e.getPriceCandlesticks(ctx, "market/history-mark-price-candles", getMarkPriceCandlesticksHistoryEPL, arg)
}

func (e *Exchange) getPriceCandlesticks(ctx context.Context, path string, epl request.EndpointLimit, arg *PriceCandlesticksRequest) ([]PriceCandlestickResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	params, err := candlestickParams(arg.InstrumentID, arg.Bar, arg.After, arg.Before, arg.Limit)
	if err != nil {
		return nil, err
	}
	var resp []PriceCandlestickResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, epl, http.MethodGet, common.EncodeURLValues(path, params), nil, &resp, request.UnauthenticatedRequest)
}

// GetExchangeRate retrieves the two week average USD to CNY exchange rate
func (e *Exchange) GetExchangeRate(ctx context.Context) (*ExchangeRateResponse, error) {
	var resp *ExchangeRateResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getExchangeRateRequestEPL, http.MethodGet, "market/exchange-rate", nil, &resp, request.UnauthenticatedRequest)
}

// GetIndexComponents retrieves an index's components
func (e *Exchange) GetIndexComponents(ctx context.Context, index string) (*IndexComponentsResponse, error) {
	if index == "" {
		return nil, errIndexRequired
	}
	params := url.Values{}
	params.Set("index", index)
	var resp *IndexComponentsResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, getIndexComponentsEPL, http.MethodGet, common.EncodeURLValues("market/index-components", params), nil, &resp, request.UnauthenticatedRequest); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, errIndexComponentNotFound
	}
	return resp, nil
}

// GetEconomicCalendarData retrieves the macro-economic calendar of the last 3 months; older data needs trading fee tier
// VIP1 or above
func (e *Exchange) GetEconomicCalendarData(ctx context.Context, arg *EconomicCalendarRequest) ([]EconomicCalendarEventResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if !arg.Before.IsZero() && !arg.After.IsZero() && arg.Before.After(arg.After) {
		return nil, common.ErrStartAfterEnd
	}
	params := url.Values{}
	if arg.Region != "" {
		params.Set("region", arg.Region)
	}
	if arg.Importance != "" {
		params.Set("importance", arg.Importance)
	}
	setTimeWindowParams(params, arg.Before, arg.After)
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []EconomicCalendarEventResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getEconomicCalendarEPL, http.MethodGet, common.EncodeURLValues("public/economic-calendar", params), nil, &resp, request.AuthenticatedRequest)
}

// GetHistoricalMarketData retrieves the download links of historical market data files
func (e *Exchange) GetHistoricalMarketData(ctx context.Context, arg *HistoricalMarketDataRequest) ([]HistoricalMarketDataResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Module == "" {
		return nil, errModuleRequired
	}
	if arg.InstrumentType == "" {
		return nil, errInvalidInstrumentType
	}
	if arg.InstrumentType == instTypeSpot {
		if len(arg.InstrumentIDs) == 0 {
			return nil, errMissingInstrumentID
		}
	} else if len(arg.InstrumentFamilies) == 0 {
		return nil, errInstrumentFamilyRequired
	}
	if len(arg.InstrumentIDs) > maxHistoricalMarketDataFilterEntries {
		return nil, fmt.Errorf("%w: %d instrument IDs exceed the maximum of %d", errExceedLimit, len(arg.InstrumentIDs), maxHistoricalMarketDataFilterEntries)
	}
	if len(arg.InstrumentFamilies) > maxHistoricalMarketDataFilterEntries {
		return nil, fmt.Errorf("%w: %d instrument families exceed the maximum of %d", errExceedLimit, len(arg.InstrumentFamilies), maxHistoricalMarketDataFilterEntries)
	}
	if arg.DateAggregationType == "" {
		return nil, errDateAggregationTypeRequired
	}
	if arg.Begin.IsZero() {
		return nil, fmt.Errorf("begin %w", common.ErrDateUnset)
	}
	if arg.End.IsZero() {
		return nil, fmt.Errorf("end %w", common.ErrDateUnset)
	}
	if arg.Begin.After(arg.End) {
		return nil, common.ErrStartAfterEnd
	}
	params := url.Values{}
	params.Set("module", arg.Module)
	params.Set("instType", arg.InstrumentType)
	if len(arg.InstrumentIDs) > 0 {
		params.Set("instIdList", strings.Join(arg.InstrumentIDs, ","))
	}
	if len(arg.InstrumentFamilies) > 0 {
		params.Set("instFamilyList", strings.Join(arg.InstrumentFamilies, ","))
	}
	params.Set("dateAggrType", arg.DateAggregationType)
	params.Set("begin", strconv.FormatInt(arg.Begin.UnixMilli(), 10))
	params.Set("end", strconv.FormatInt(arg.End.UnixMilli(), 10))
	var resp []HistoricalMarketDataResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getHistoricalMarketDataEPL, http.MethodGet, common.EncodeURLValues("public/market-data-history", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetMMInstrumentTypes retrieves the market maker program classifications of spot and perpetual swap instruments
func (e *Exchange) GetMMInstrumentTypes(ctx context.Context, arg *MMInstrumentTypesRequest) ([]MMInstrumentTypeResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	params := url.Values{}
	if arg.InstrumentType != "" {
		params.Set("instType", arg.InstrumentType)
	}
	if arg.InstrumentID != "" {
		params.Set("instId", arg.InstrumentID)
	}
	var resp []MMInstrumentTypeResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getMMInstrumentTypesEPL, http.MethodGet, common.EncodeURLValues("public/mm-instrument-types", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetDeltaHedgeCurrencies retrieves the currencies sharing an underlying asset, which can form a delta hedge, such as
// ETH and BETH
func (e *Exchange) GetDeltaHedgeCurrencies(ctx context.Context, arg *DeltaHedgeCurrenciesRequest) ([]DeltaHedgeCurrencyResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	params := url.Values{}
	if !arg.Currency.IsEmpty() {
		params.Set("ccy", arg.Currency.String())
	}
	var resp []DeltaHedgeCurrencyResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getDeltaHedgeCurrenciesEPL, http.MethodGet, common.EncodeURLValues("public/delta-hedge-currencies", params), nil, &resp, request.UnauthenticatedRequest)
}

// setTimeWindowParams sets the before and after millisecond bounds OKX's paginated public endpoints take: before
// returns newer records and after earlier ones
func setTimeWindowParams(params url.Values, before, after time.Time) {
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
}
