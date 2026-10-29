package bybit

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

var (
	errLimitRequired        = errors.New("limit is required")
	errIntervalTimeRequired = errors.New("interval time is required")
	errPeriodRequired       = errors.New("period is required")
	errIndexNameRequired    = errors.New("index name is required")
	errProductTypeRequired  = errors.New("product type is required")

	errBaseCoinOrSymbolRequired = errors.New("base coin or symbol is required for option")
	errInvalidRPIOrderbookLevel = errors.New("RPI orderbook level must hold a price, a size and an RPI size")
	errOrderbookNotFound        = errors.New("orderbook not found")
)

// GetBybitServerTime returns Bybit's server time (Get Bybit Server Time)
func (e *Exchange) GetBybitServerTime(ctx context.Context) (*ServerTimeResponse, error) {
	var resp *ServerTimeResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, "/v5/market/time", defaultEPL, &resp)
}

// GetKlines returns a symbol's candles, newest first (Get Kline)
func (e *Exchange) GetKlines(ctx context.Context, req *KlineRequest) (*KlineResponse, error) {
	params, err := klineParams(req)
	if err != nil {
		return nil, err
	}
	var resp *KlineResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/market/kline", params), defaultEPL, &resp)
}

// GetMarkPriceKline returns a symbol's mark price candles, newest first (Get Mark Price Kline)
func (e *Exchange) GetMarkPriceKline(ctx context.Context, req *KlineRequest) (*PriceKlineResponse, error) {
	params, err := klineParams(req)
	if err != nil {
		return nil, err
	}
	var resp *PriceKlineResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/market/mark-price-kline", params), defaultEPL, &resp)
}

// GetIndexPriceKline returns a symbol's index price candles, newest first (Get Index Price Kline)
func (e *Exchange) GetIndexPriceKline(ctx context.Context, req *KlineRequest) (*PriceKlineResponse, error) {
	params, err := klineParams(req)
	if err != nil {
		return nil, err
	}
	var resp *PriceKlineResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/market/index-price-kline", params), defaultEPL, &resp)
}

// GetPremiumIndexPriceKline returns a perpetual contract's premium index price candles, newest first (Get Premium
// Index Price Kline)
func (e *Exchange) GetPremiumIndexPriceKline(ctx context.Context, req *KlineRequest) (*PriceKlineResponse, error) {
	params, err := klineParams(req)
	if err != nil {
		return nil, err
	}
	var resp *PriceKlineResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/market/premium-index-price-kline", params), defaultEPL, &resp)
}

// klineParams builds the query the four kline endpoints share
func klineParams(req *KlineRequest) (url.Values, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol == "" {
		return nil, errSymbolMissing
	}
	interval, err := intervalToString(req.Interval)
	if err != nil {
		return nil, err
	}
	if !req.Start.IsZero() && !req.End.IsZero() {
		if err := checkTimeWindow(req.Start, req.End); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if req.Category != "" {
		params.Set("category", req.Category)
	}
	params.Set("symbol", req.Symbol)
	params.Set("interval", interval)
	if !req.Start.IsZero() {
		params.Set("start", strconv.FormatInt(req.Start.UnixMilli(), 10))
	}
	if !req.End.IsZero() {
		params.Set("end", strconv.FormatInt(req.End.UnixMilli(), 10))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	return params, nil
}

// GetInstrumentInfo returns a page of instrument specifications (Get Instruments Info)
func (e *Exchange) GetInstrumentInfo(ctx context.Context, req *InstrumentInfoRequest) (*InstrumentsInfoResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Category == "" {
		return nil, errCategoryNotSet
	}
	params := url.Values{}
	params.Set("category", req.Category)
	if req.Symbol != "" {
		params.Set("symbol", req.Symbol)
	}
	if req.SymbolType != "" {
		params.Set("symbolType", req.SymbolType)
	}
	if req.Status != "" {
		params.Set("status", req.Status)
	}
	if req.BaseCoin != "" {
		params.Set("baseCoin", req.BaseCoin)
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	var resp *InstrumentsInfoResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/market/instruments-info", params), defaultEPL, &resp)
}

// GetInstrumentSummary returns a page of the names, underlying assets and trading increments of linear perpetual
// contracts or options (Get Instrument Summary)
func (e *Exchange) GetInstrumentSummary(ctx context.Context, req *InstrumentSummaryRequest) (*InstrumentSummaryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Category == "" {
		return nil, errCategoryNotSet
	}
	// Bybit answers any other category, and these in another case, with 10001
	if req.Category != cLinear && req.Category != cOption {
		return nil, fmt.Errorf("%w: %q", errUnsupportedCategory, req.Category)
	}
	params := url.Values{}
	params.Set("category", req.Category)
	if req.Symbol != "" {
		params.Set("symbol", req.Symbol)
	}
	if req.SymbolType != "" {
		params.Set("symbolType", req.SymbolType)
	}
	if req.MarketRegion != "" {
		params.Set("marketRegion", req.MarketRegion)
	}
	if !req.BaseCoin.IsEmpty() {
		params.Set("baseCoin", req.BaseCoin.Upper().String())
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	var resp *InstrumentSummaryResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/market/instrument-summary", params), defaultEPL, &resp)
}

// GetOrderBook returns a symbol's orderbook, without retail price improvement orders (Get Orderbook)
func (e *Exchange) GetOrderBook(ctx context.Context, category, symbol string, limit uint64) (*OrderbookResponse, error) {
	if category == "" {
		return nil, errCategoryNotSet
	}
	if symbol == "" {
		return nil, errSymbolMissing
	}
	params := url.Values{}
	params.Set("category", category)
	params.Set("symbol", symbol)
	if limit != 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var raw json.RawMessage
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/market/orderbook", params), defaultEPL, &raw); err != nil {
		return nil, err
	}
	// Bybit answers an option symbol it does not list with success and an empty array in place of the snapshot
	if string(bytes.TrimSpace(raw)) == "[]" {
		return nil, fmt.Errorf("%w: %s %s", errOrderbookNotFound, category, symbol)
	}
	var resp *OrderbookResponse
	return resp, json.Unmarshal(raw, &resp)
}

// GetRPIOrderBook returns a symbol's orderbook with retail price improvement (RPI) order sizes shown separately (Get
// RPI Orderbook)
func (e *Exchange) GetRPIOrderBook(ctx context.Context, category, symbol string, limit uint64) (*RPIOrderbookResponse, error) {
	if symbol == "" {
		return nil, errSymbolMissing
	}
	if limit == 0 {
		return nil, errLimitRequired
	}
	params := url.Values{}
	if category != "" {
		params.Set("category", category)
	}
	params.Set("symbol", symbol)
	params.Set("limit", strconv.FormatUint(limit, 10))
	var resp *RPIOrderbookResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/market/rpi_orderbook", params), defaultEPL, &resp)
}

// GetFullDepthOrderBook returns up to 10,000 levels a side of a symbol's orderbook, without retail price improvement
// orders, to initialise a book kept current by the websocket full orderbook stream (Get Full Depth Orderbook)
func (e *Exchange) GetFullDepthOrderBook(ctx context.Context, category, symbol string) (*OrderbookResponse, error) {
	if category == "" {
		return nil, errCategoryNotSet
	}
	if symbol == "" {
		return nil, errSymbolMissing
	}
	params := url.Values{}
	params.Set("category", category)
	params.Set("symbol", symbol)
	var resp *OrderbookResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/market/full_orderbook", params), defaultEPL, &resp)
}

// GetTickers returns the latest prices and 24 hour statistics of a category's symbols (Get Tickers)
func (e *Exchange) GetTickers(ctx context.Context, req *TickersRequest) (*TickersResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Category == "" {
		return nil, errCategoryNotSet
	}
	// The docs mark neither as required, but Bybit answers an option request without both with 10001
	if req.Category == cOption && req.Symbol == "" && req.BaseCoin.IsEmpty() {
		return nil, errBaseCoinOrSymbolRequired
	}
	params := url.Values{}
	params.Set("category", req.Category)
	if req.Symbol != "" {
		params.Set("symbol", req.Symbol)
	}
	if !req.BaseCoin.IsEmpty() {
		params.Set("baseCoin", req.BaseCoin.Upper().String())
	}
	if !req.ExpiryDate.IsZero() {
		params.Set("expDate", expiryDate(req.ExpiryDate))
	}
	var resp *TickersResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/market/tickers", params), defaultEPL, &resp)
}

// GetFundingRateHistory returns a perpetual contract's settled funding rates, newest first (Get Funding Rate History)
func (e *Exchange) GetFundingRateHistory(ctx context.Context, req *FundingRateHistoryRequest) (*FundingRateHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Category == "" {
		return nil, errCategoryNotSet
	}
	if req.Symbol == "" {
		return nil, errSymbolMissing
	}
	// Bybit rejects a start time without an end time but takes an end time alone
	if !req.StartTime.IsZero() {
		if err := checkTimeWindow(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("category", req.Category)
	params.Set("symbol", req.Symbol)
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp *FundingRateHistoryResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/market/funding/history", params), defaultEPL, &resp)
}

// GetPublicTradingHistory returns recent public trades (Get Recent Public Trades)
func (e *Exchange) GetPublicTradingHistory(ctx context.Context, req *PublicTradingHistoryRequest) (*PublicTradingHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Category == "" {
		return nil, errCategoryNotSet
	}
	if req.Symbol == "" && (req.Category == cSpot || req.Category == cLinear || req.Category == cInverse) {
		return nil, errSymbolMissing
	}
	params := url.Values{}
	params.Set("category", req.Category)
	if req.Symbol != "" {
		params.Set("symbol", req.Symbol)
	}
	if !req.BaseCoin.IsEmpty() {
		params.Set("baseCoin", req.BaseCoin.Upper().String())
	}
	if req.OptionType != "" {
		params.Set("optionType", req.OptionType)
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp *PublicTradingHistoryResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/market/recent-trade", params), defaultEPL, &resp)
}

// GetOpenInterestData returns a contract's open interest, newest first (Get Open Interest)
func (e *Exchange) GetOpenInterestData(ctx context.Context, req *OpenInterestRequest) (*OpenInterestResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Category == "" {
		return nil, errCategoryNotSet
	}
	if req.Symbol == "" {
		return nil, errSymbolMissing
	}
	if req.IntervalTime == "" {
		return nil, errIntervalTimeRequired
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := checkTimeWindow(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("category", req.Category)
	params.Set("symbol", req.Symbol)
	params.Set("intervalTime", req.IntervalTime)
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	var resp *OpenInterestResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/market/open-interest", params), defaultEPL, &resp)
}

// GetHistoricalVolatility returns an option base coin's hourly historical volatility (Get Historical Volatility). A
// time window covers at most 30 days of the last 2 years
func (e *Exchange) GetHistoricalVolatility(ctx context.Context, req *HistoricalVolatilityRequest) ([]HistoricalVolatilityResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Category == "" {
		return nil, errCategoryNotSet
	}
	// Bybit takes both bounds of the window or neither
	if !req.StartTime.IsZero() || !req.EndTime.IsZero() {
		if err := checkTimeWindow(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("category", req.Category)
	if !req.BaseCoin.IsEmpty() {
		params.Set("baseCoin", req.BaseCoin.Upper().String())
	}
	if !req.QuoteCoin.IsEmpty() {
		params.Set("quoteCoin", req.QuoteCoin.Upper().String())
	}
	if req.Period != 0 {
		params.Set("period", strconv.FormatUint(req.Period, 10))
	}
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	var resp []HistoricalVolatilityResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/market/historical-volatility", params), defaultEPL, &resp)
}

// GetInsurance returns insurance pool balances, of every coin when coin is empty (Get Insurance Pool)
func (e *Exchange) GetInsurance(ctx context.Context, coin currency.Code) (*InsuranceResponse, error) {
	params := url.Values{}
	if !coin.IsEmpty() {
		params.Set("coin", coin.Upper().String())
	}
	var resp *InsuranceResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/market/insurance", params), defaultEPL, &resp)
}

// GetRiskLimit returns a page of risk limit tiers (Get Risk Limit)
func (e *Exchange) GetRiskLimit(ctx context.Context, category, symbol, cursor string) (*RiskLimitTiersResponse, error) {
	if category == "" {
		return nil, errCategoryNotSet
	}
	params := url.Values{}
	params.Set("category", category)
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if cursor != "" {
		params.Set("cursor", cursor)
	}
	var resp *RiskLimitTiersResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/market/risk-limit", params), defaultEPL, &resp)
}

// GetDeliveryPrice returns a page of delivery prices of dated contracts and options (Get Delivery Price). Without a
// symbol, options return only the symbols delivering at the time
func (e *Exchange) GetDeliveryPrice(ctx context.Context, req *DeliveryPriceRequest) (*DeliveryPriceResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Category == "" {
		return nil, errCategoryNotSet
	}
	params := url.Values{}
	params.Set("category", req.Category)
	if req.Symbol != "" {
		params.Set("symbol", req.Symbol)
	}
	if !req.BaseCoin.IsEmpty() {
		params.Set("baseCoin", req.BaseCoin.Upper().String())
	}
	if !req.SettleCoin.IsEmpty() {
		params.Set("settleCoin", req.SettleCoin.Upper().String())
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	var resp *DeliveryPriceResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/market/delivery-price", params), defaultEPL, &resp)
}

// GetNewDeliveryPrice returns an option base coin's 50 most recent delivery prices, newest first (Get New Delivery
// Price). settleCoin defaults to USDT
func (e *Exchange) GetNewDeliveryPrice(ctx context.Context, category string, baseCoin, settleCoin currency.Code) (*NewDeliveryPriceResponse, error) {
	if category == "" {
		return nil, errCategoryNotSet
	}
	if baseCoin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("category", category)
	params.Set("baseCoin", baseCoin.Upper().String())
	if !settleCoin.IsEmpty() {
		params.Set("settleCoin", settleCoin.Upper().String())
	}
	var resp *NewDeliveryPriceResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/market/new-delivery-price", params), defaultEPL, &resp)
}

// GetLongShortRatio returns a contract's long short account ratio, newest first (Get Long Short Ratio)
func (e *Exchange) GetLongShortRatio(ctx context.Context, req *LongShortRatioRequest) (*LongShortRatioResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Category == "" {
		return nil, errCategoryNotSet
	}
	if req.Symbol == "" {
		return nil, errSymbolMissing
	}
	if req.Period == "" {
		return nil, errPeriodRequired
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := checkTimeWindow(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("category", req.Category)
	params.Set("symbol", req.Symbol)
	params.Set("period", req.Period)
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	var resp *LongShortRatioResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/market/account-ratio", params), defaultEPL, &resp)
}

// GetIndexPriceComponents returns the exchange prices that make up an index price, such as BTCUSDT's (Get Index
// Price Components)
func (e *Exchange) GetIndexPriceComponents(ctx context.Context, indexName string) (*IndexPriceComponentsResponse, error) {
	if indexName == "" {
		return nil, errIndexNameRequired
	}
	params := url.Values{}
	params.Set("indexName", indexName)
	var resp *IndexPriceComponentsResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/market/index-price-components", params), defaultEPL, &resp)
}

// GetOrderPriceLimit returns the highest bid and lowest ask price a symbol accepts; category defaults to linear (Get
// Order Price Limit)
func (e *Exchange) GetOrderPriceLimit(ctx context.Context, category, symbol string) (*OrderPriceLimitResponse, error) {
	if symbol == "" {
		return nil, errSymbolMissing
	}
	params := url.Values{}
	if category != "" {
		params.Set("category", category)
	}
	params.Set("symbol", symbol)
	var resp *OrderPriceLimitResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/market/price-limit", params), defaultEPL, &resp)
}

// GetADLAlert returns auto-deleveraging (ADL) alerts and insurance pool data, of every contract when symbol is empty
// (Get ADL Alert)
func (e *Exchange) GetADLAlert(ctx context.Context, symbol string) (*ADLAlertResponse, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp *ADLAlertResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/market/adlAlert", params), defaultEPL, &resp)
}

// GetFeeGroupStructure returns the contract fee groups and their fee rates for Pro and market maker clients, of
// every group when groupID is zero (Get Fee Group Structure)
func (e *Exchange) GetFeeGroupStructure(ctx context.Context, productType string, groupID uint64) (*FeeGroupStructureResponse, error) {
	if productType == "" {
		return nil, errProductTypeRequired
	}
	params := url.Values{}
	params.Set("productType", productType)
	if groupID != 0 {
		params.Set("groupId", strconv.FormatUint(groupID, 10))
	}
	var resp *FeeGroupStructureResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/market/fee-group-info", params), defaultEPL, &resp)
}

// GetOptionBaseCoins returns the option base coins of the given underlying types (0 crypto, 1 commodity, 2 stock, 3
// forex, 4 oil), or of every type when none are given (Get Option Base Coins)
func (e *Exchange) GetOptionBaseCoins(ctx context.Context, underlyingTypes []uint64) (*OptionBaseCoinsResponse, error) {
	params := url.Values{}
	if len(underlyingTypes) != 0 {
		values := make([]string, len(underlyingTypes))
		for i := range underlyingTypes {
			values[i] = strconv.FormatUint(underlyingTypes[i], 10)
		}
		params.Set("underlyingType", strings.Join(values, ","))
	}
	var resp *OptionBaseCoinsResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/market/option-base-coins", params), defaultEPL, &resp)
}

// GetCorporateActions returns a page of the forward splits, reverse splits and dividends of TradFi options'
// underlying assets, newest effective time first; Bybit keeps them for 14 days (Query Corporate Action)
func (e *Exchange) GetCorporateActions(ctx context.Context, req *CorporateActionsRequest) (*CorporateActionsResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.Category != "" {
		params.Set("category", req.Category)
	}
	if !req.BaseCoin.IsEmpty() {
		params.Set("baseCoin", req.BaseCoin.Upper().String())
	}
	if req.ActionType != "" {
		params.Set("actionType", req.ActionType)
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	var resp *CorporateActionsResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/market/corporate-action", params), defaultEPL, &resp)
}

func intervalToString(interval kline.Interval) (string, error) {
	inter, okay := intervalMap[interval]
	if okay {
		return inter, nil
	}
	return "", kline.ErrUnsupportedInterval
}

// checkTimeWindow checks the bounds of a time window. Bybit serves a window whose start equals its end with the record
// at that time, so equal bounds are let through unless they are in the future
func checkTimeWindow(start, end time.Time) error {
	err := common.StartEndTimeCheck(start, end)
	if !errors.Is(err, common.ErrStartEqualsEnd) {
		return err
	}
	if start.After(time.Now()) {
		return common.ErrStartAfterTimeNow
	}
	return nil
}

// expiryDate formats an expiry date the way Bybit's symbols write it, such as 6NOV26 or 25DEC26: Bybit matches it
// exactly, so a zero padded day finds nothing. It uses the UTC date, on which options expire
func expiryDate(t time.Time) string {
	return strings.ToUpper(t.UTC().Format(longDatedFormat))
}
